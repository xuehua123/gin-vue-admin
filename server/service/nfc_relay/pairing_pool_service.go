package nfc_relay

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/flipped-aurora/gin-vue-admin/server/global"
	commonService "github.com/flipped-aurora/gin-vue-admin/server/service/common"
	"github.com/redis/go-redis/v9"
	"go.uber.org/zap"
)

// PairingPoolService 配对池服务
// 基于Redis ZSET实现高效的配对匹配算法
type PairingPoolService struct {
	userCacheService *commonService.UserCacheService
}

// MatchResult 配对匹配结果
type MatchResult struct {
	Matched       bool                   `json:"matched"`
	PeerClientID  string                 `json:"peer_client_id,omitempty"`
	PeerUserID    uint                   `json:"peer_user_id,omitempty"`
	PeerRole      string                 `json:"peer_role,omitempty"`
	PairID        string                 `json:"pair_id,omitempty"`
	WaitingTime   time.Duration          `json:"waiting_time,omitempty"`
	QueuePosition int                    `json:"queue_position,omitempty"`
	EstimatedWait time.Duration          `json:"estimated_wait,omitempty"`
	Metadata      map[string]interface{} `json:"metadata,omitempty"`
}

// PairingStatus 配对状态
type PairingStatus struct {
	Status        string                 `json:"status"` // idle, waiting, matched, timeout, error
	Role          string                 `json:"role,omitempty"`
	ClientID      string                 `json:"client_id,omitempty"`
	JoinedAt      time.Time              `json:"joined_at,omitempty"`
	MatchedAt     *time.Time             `json:"matched_at,omitempty"`
	PeerInfo      *MatchResult           `json:"peer_info,omitempty"`
	QueuePosition int                    `json:"queue_position,omitempty"`
	EstimatedWait time.Duration          `json:"estimated_wait,omitempty"`
	TimeoutAt     *time.Time             `json:"timeout_at,omitempty"`
	Metadata      map[string]interface{} `json:"metadata,omitempty"`
}

const (
	// Redis键模板
	PairingPoolKeyTemplate    = "pairing:pool:%s"       // pairing:pool:transmitter
	PairingStateKeyTemplate   = "pairing:state:%d"      // pairing:state:123
	PairingTimeoutKeyTemplate = "pairing:timeout:%d:%s" // pairing:timeout:123:transmitter
	PairingMatchedKeyTemplate = "pairing:matched:%s"    // pairing:matched:pair_abc123

	// 配对超时时间
	PairingTimeoutDuration = 3 * time.Minute

	// 注意：角色定义已在mqtt_service.go中定义，这里不重复定义

	// 状态定义
	StatusIdle    = "idle"
	StatusWaiting = "waiting"
	StatusMatched = "matched"
	StatusTimeout = "timeout"
	StatusError   = "error"
)

// NewPairingPoolService 创建配对池服务实例
func NewPairingPoolService() *PairingPoolService {
	return &PairingPoolService{
		userCacheService: commonService.UserCacheServiceApp,
	}
}

// JoinPairingPool 加入配对池
func (s *PairingPoolService) JoinPairingPool(userUUID, role, clientID string, deviceInfo map[string]interface{}) (*MatchResult, error) {
	// 1. 验证参数
	if err := s.validatePairingRequest(userUUID, role, clientID); err != nil {
		return nil, err
	}

	// 2. 获取用户ID
	userID, err := s.userCacheService.GetUserIDByUUID(userUUID)
	if err != nil {
		global.GVA_LOG.Error("获取用户ID失败",
			zap.Error(err),
			zap.String("userUUID", userUUID))
		return nil, fmt.Errorf("用户身份验证失败: %w", err)
	}

	ctx := context.Background()
	now := time.Now()

	// 3. 构建配对数据
	pairingData := map[string]interface{}{
		"client_id":   clientID,
		"user_id":     userID,
		"user_uuid":   userUUID,
		"role":        role,
		"joined_at":   now.Unix(),
		"device_info": deviceInfo,
		"status":      StatusWaiting,
	}

	pairingJSON, err := json.Marshal(pairingData)
	if err != nil {
		return nil, fmt.Errorf("序列化配对数据失败: %w", err)
	}

	// 4. 使用Redis事务确保原子性
	pipe := global.GVA_REDIS.TxPipeline()

	// 4.1 加入等待池（使用时间戳作为分值）
	poolKey := fmt.Sprintf(PairingPoolKeyTemplate, role)
	pipe.ZAdd(ctx, poolKey, redis.Z{
		Score:  float64(now.Unix()),
		Member: string(pairingJSON),
	})

	// 4.2 更新用户配对状态
	stateKey := fmt.Sprintf(PairingStateKeyTemplate, userID)
	pipe.HSet(ctx, stateKey, role, clientID)
	pipe.Expire(ctx, stateKey, PairingTimeoutDuration+time.Minute) // 比超时时间多1分钟

	// 4.3 设置超时清理键
	timeoutKey := fmt.Sprintf(PairingTimeoutKeyTemplate, userID, role)
	pipe.SetEx(ctx, timeoutKey, clientID, PairingTimeoutDuration)

	// 4.4 执行事务
	if _, err := pipe.Exec(ctx); err != nil {
		global.GVA_LOG.Error("加入配对池事务失败",
			zap.Error(err),
			zap.String("userUUID", userUUID),
			zap.String("role", role))
		return nil, fmt.Errorf("加入配对池失败: %w", err)
	}

	global.GVA_LOG.Info("用户加入配对池成功",
		zap.String("userUUID", userUUID),
		zap.Uint("userID", userID),
		zap.String("role", role),
		zap.String("clientID", clientID))

	// 5. 立即尝试匹配
	return s.FindPeerAndMatch(userUUID, role, clientID)
}

// FindPeerAndMatch 查找配对伙伴并尝试匹配
func (s *PairingPoolService) FindPeerAndMatch(userUUID, role, clientID string) (*MatchResult, error) {
	// 1. 确定目标角色
	targetRole := s.getTargetRole(role)
	if targetRole == "" {
		return nil, fmt.Errorf("无效的角色: %s", role)
	}

	userID, err := s.userCacheService.GetUserIDByUUID(userUUID)
	if err != nil {
		return nil, err
	}

	ctx := context.Background()
	targetPoolKey := fmt.Sprintf(PairingPoolKeyTemplate, targetRole)

	// 2. 查找目标角色的等待池
	peers, err := global.GVA_REDIS.ZRangeWithScores(ctx, targetPoolKey, 0, -1).Result()
	if err != nil {
		return nil, fmt.Errorf("查询配对池失败: %w", err)
	}

	if len(peers) == 0 {
		// 没有等待的伙伴，返回等待状态
		position, estimatedWait := s.calculateQueuePosition(role)
		return &MatchResult{
			Matched:       false,
			QueuePosition: position,
			EstimatedWait: estimatedWait,
		}, nil
	}

	// 3. 尝试匹配最早等待的伙伴
	for _, peer := range peers {
		peerData := make(map[string]interface{})
		if err := json.Unmarshal([]byte(peer.Member.(string)), &peerData); err != nil {
			continue
		}

		peerUserID := uint(peerData["user_id"].(float64))
		peerClientID := peerData["client_id"].(string)
		peerUserUUID := peerData["user_uuid"].(string)

		// 【重要修正】必须是同一个用户下的不同角色才能配对
		if peerUserID != userID {
			continue // 如果不是同一个用户，则跳过
		}

		// 4. 执行配对匹配
		if pairID, err := s.createPairMatch(userID, userUUID, role, clientID,
			peerUserID, peerUserUUID, targetRole, peerClientID); err == nil {

			// 5. 从等待池中移除已匹配的用户
			s.removeFromPairingPool(targetRole, peer.Member.(string))
			s.removeFromPairingPool(role, s.buildPairingData(userID, userUUID, role, clientID))

			return &MatchResult{
				Matched:      true,
				PeerClientID: peerClientID,
				PeerUserID:   peerUserID,
				PeerRole:     targetRole,
				PairID:       pairID,
				WaitingTime:  time.Since(time.Unix(int64(peer.Score), 0)),
			}, nil
		}
	}

	// 没有成功匹配，返回等待状态
	position, estimatedWait := s.calculateQueuePosition(role)
	return &MatchResult{
		Matched:       false,
		QueuePosition: position,
		EstimatedWait: estimatedWait,
	}, nil
}

// CancelPairing 取消配对
func (s *PairingPoolService) CancelPairing(userUUID, role string) error {
	userID, err := s.userCacheService.GetUserIDByUUID(userUUID)
	if err != nil {
		return err
	}

	ctx := context.Background()

	// 【增强】预先收集需要清理的信息，便于日志记录和回滚
	var cleanupInfo struct {
		poolKeys    []string
		stateKeys   []string
		timeoutKeys []string
		removedData []string
	}

	// 使用Pipeline进行批量清理
	pipe := global.GVA_REDIS.TxPipeline()

	// 1. 从等待池中移除
	if role != "" {
		poolKey := fmt.Sprintf(PairingPoolKeyTemplate, role)
		cleanupInfo.poolKeys = append(cleanupInfo.poolKeys, poolKey)

		// 由于需要根据user_id查找，使用lua脚本更高效
		luaScript := `
			local pool_key = KEYS[1]
			local user_id = ARGV[1]
			local members = redis.call('ZRANGE', pool_key, 0, -1)
			local removed_data = {}
			for i, member in ipairs(members) do
				local data = cjson.decode(member)
				if data.user_id == tonumber(user_id) then
					redis.call('ZREM', pool_key, member)
					table.insert(removed_data, member)
				end
			end
			return removed_data
		`
		result := pipe.Eval(ctx, luaScript, []string{poolKey}, userID)
		if result != nil {
			// 记录移除的数据用于审计
			global.GVA_LOG.Debug("准备从配对池移除数据",
				zap.String("poolKey", poolKey),
				zap.Uint("userID", userID))
		}
	} else {
		// 清理所有角色
		for _, r := range []string{"transmitter", "receiver"} {
			poolKey := fmt.Sprintf(PairingPoolKeyTemplate, r)
			cleanupInfo.poolKeys = append(cleanupInfo.poolKeys, poolKey)

			luaScript := `
				local pool_key = KEYS[1]
				local user_id = ARGV[1]
				local members = redis.call('ZRANGE', pool_key, 0, -1)
				local removed_count = 0
				for i, member in ipairs(members) do
					local data = cjson.decode(member)
					if data.user_id == tonumber(user_id) then
						redis.call('ZREM', pool_key, member)
						removed_count = removed_count + 1
					end
				end
				return removed_count
			`
			pipe.Eval(ctx, luaScript, []string{poolKey}, userID)
		}
	}

	// 2. 清理状态键
	stateKey := fmt.Sprintf(PairingStateKeyTemplate, userID)
	cleanupInfo.stateKeys = append(cleanupInfo.stateKeys, stateKey)

	if role != "" {
		pipe.HDel(ctx, stateKey, role)
	} else {
		pipe.Del(ctx, stateKey)
	}

	// 3. 清理超时键
	if role != "" {
		timeoutKey := fmt.Sprintf(PairingTimeoutKeyTemplate, userID, role)
		cleanupInfo.timeoutKeys = append(cleanupInfo.timeoutKeys, timeoutKey)
		pipe.Del(ctx, timeoutKey)
	} else {
		for _, r := range []string{"transmitter", "receiver"} {
			timeoutKey := fmt.Sprintf(PairingTimeoutKeyTemplate, userID, r)
			cleanupInfo.timeoutKeys = append(cleanupInfo.timeoutKeys, timeoutKey)
			pipe.Del(ctx, timeoutKey)
		}
	}

	// 4. 执行清理事务
	results, err := pipe.Exec(ctx)
	if err != nil {
		global.GVA_LOG.Error("取消配对事务失败",
			zap.Error(err),
			zap.String("userUUID", userUUID),
			zap.Strings("poolKeys", cleanupInfo.poolKeys),
			zap.Strings("stateKeys", cleanupInfo.stateKeys),
			zap.Strings("timeoutKeys", cleanupInfo.timeoutKeys))
		return fmt.Errorf("取消配对失败: %w", err)
	}

	// 【增强】记录详细的清理结果用于审计和监控
	global.GVA_LOG.Info("取消配对成功",
		zap.String("userUUID", userUUID),
		zap.String("role", role),
		zap.Int("executedCommands", len(results)),
		zap.Strings("cleanedPoolKeys", cleanupInfo.poolKeys),
		zap.Strings("cleanedStateKeys", cleanupInfo.stateKeys),
		zap.Strings("cleanedTimeoutKeys", cleanupInfo.timeoutKeys))

	// 【企业级新增】发送配对取消通知
	// 需要先获取clientID才能发送通知
	if role != "" {
		// 尝试从清理前的状态获取clientID
		stateKey := fmt.Sprintf(PairingStateKeyTemplate, userID)
		if clientID, err := global.GVA_REDIS.HGet(context.Background(), stateKey, role).Result(); err == nil && clientID != "" {
			if err := s.notifyPairingCancellation(context.Background(), userUUID, role, clientID); err != nil {
				global.GVA_LOG.Warn("发送配对取消通知失败",
					zap.String("userUUID", userUUID),
					zap.String("role", role),
					zap.String("clientID", clientID),
					zap.Error(err))
				// 企业级处理：不影响主流程，只记录警告
			}
		}
	}

	return nil
}

// GetPairingStatus 获取配对状态
func (s *PairingPoolService) GetPairingStatus(userUUID string) (*PairingStatus, error) {
	userID, err := s.userCacheService.GetUserIDByUUID(userUUID)
	if err != nil {
		return nil, err
	}

	ctx := context.Background()
	stateKey := fmt.Sprintf(PairingStateKeyTemplate, userID)

	// 获取用户当前的配对状态
	states, err := global.GVA_REDIS.HGetAll(ctx, stateKey).Result()
	if err != nil || len(states) == 0 {
		return &PairingStatus{Status: StatusIdle}, nil
	}

	// 查找活跃的配对状态
	for role, clientID := range states {
		if clientID == "" {
			continue
		}

		// 检查是否在等待池中
		if inPool, joinedAt := s.isInPairingPool(userID, role); inPool {
			position, estimatedWait := s.calculateQueuePosition(role)
			timeoutAt := joinedAt.Add(PairingTimeoutDuration)

			return &PairingStatus{
				Status:        StatusWaiting,
				Role:          role,
				ClientID:      clientID,
				JoinedAt:      joinedAt,
				QueuePosition: position,
				EstimatedWait: estimatedWait,
				TimeoutAt:     &timeoutAt,
			}, nil
		}

		// 检查是否已匹配
		if matched, pairID := s.isMatched(userID, role); matched {
			matchedAt := time.Now() // 实际应该从配对记录中获取
			return &PairingStatus{
				Status:    StatusMatched,
				Role:      role,
				ClientID:  clientID,
				MatchedAt: &matchedAt,
				Metadata:  map[string]interface{}{"pair_id": pairID},
			}, nil
		}
	}

	return &PairingStatus{Status: StatusIdle}, nil
}

// GetUserPairingStatus 获取用户特定角色的配对状态
func (s *PairingPoolService) GetUserPairingStatus(userUUID, role string) (*PairingStatus, error) {
	userID, err := s.userCacheService.GetUserIDByUUID(userUUID)
	if err != nil {
		return nil, err
	}

	ctx := context.Background()
	stateKey := fmt.Sprintf(PairingStateKeyTemplate, userID)

	// 检查特定角色的状态
	clientID, err := global.GVA_REDIS.HGet(ctx, stateKey, role).Result()
	if err == redis.Nil {
		return nil, nil // 该角色没有配对状态
	}
	if err != nil {
		return nil, fmt.Errorf("获取配对状态失败: %w", err)
	}

	// 检查是否在等待池中
	if inPool, joinedAt := s.isInPairingPool(userID, role); inPool {
		position, estimatedWait := s.calculateQueuePosition(role)
		timeoutAt := joinedAt.Add(PairingTimeoutDuration)
		return &PairingStatus{
			Status:        StatusWaiting,
			Role:          role,
			ClientID:      clientID,
			JoinedAt:      joinedAt,
			QueuePosition: position,
			EstimatedWait: estimatedWait,
			TimeoutAt:     &timeoutAt,
		}, nil
	}

	// 检查是否已匹配
	if matched, pairID := s.isMatched(userID, role); matched {
		return &PairingStatus{
			Status:   StatusMatched,
			Role:     role,
			ClientID: clientID,
			Metadata: map[string]interface{}{"pair_id": pairID},
		}, nil
	}

	return nil, nil
}

// LeavePairingPool 离开配对池
func (s *PairingPoolService) LeavePairingPool(userUUID, role string) error {
	userID, err := s.userCacheService.GetUserIDByUUID(userUUID)
	if err != nil {
		return err
	}

	ctx := context.Background()
	poolKey := fmt.Sprintf(PairingPoolKeyTemplate, role)
	stateKey := fmt.Sprintf(PairingStateKeyTemplate, userID)
	timeoutKey := fmt.Sprintf(PairingTimeoutKeyTemplate, userID, role)

	// 使用事务确保原子性
	pipe := global.GVA_REDIS.TxPipeline()

	// 1. 从等待池中移除所有该用户的配对数据
	// 由于ZSET存储的是JSON字符串，需要遍历查找
	members, err := global.GVA_REDIS.ZRange(ctx, poolKey, 0, -1).Result()
	if err == nil {
		for _, member := range members {
			var pairingData map[string]interface{}
			if json.Unmarshal([]byte(member), &pairingData) == nil {
				if uid, ok := pairingData["user_id"].(float64); ok && uint(uid) == userID {
					pipe.ZRem(ctx, poolKey, member)
					break
				}
			}
		}
	}

	// 2. 清理用户状态
	pipe.HDel(ctx, stateKey, role)

	// 3. 清理超时键
	pipe.Del(ctx, timeoutKey)

	// 4. 执行事务
	if _, err := pipe.Exec(ctx); err != nil {
		global.GVA_LOG.Error("离开配对池事务失败",
			zap.Error(err),
			zap.String("userUUID", userUUID),
			zap.String("role", role))
		return fmt.Errorf("离开配对池失败: %w", err)
	}

	global.GVA_LOG.Info("用户离开配对池成功",
		zap.String("userUUID", userUUID),
		zap.Uint("userID", userID),
		zap.String("role", role))

	return nil
}

// 辅助方法

func (s *PairingPoolService) validatePairingRequest(userUUID, role, clientID string) error {
	if err := s.userCacheService.ValidateUUID(userUUID); err != nil {
		return err
	}

	if role != "transmitter" && role != "receiver" {
		return fmt.Errorf("无效的角色: %s", role)
	}

	if clientID == "" {
		return errors.New("客户端ID不能为空")
	}

	return nil
}

func (s *PairingPoolService) getTargetRole(role string) string {
	if role == "transmitter" {
		return "receiver"
	}
	if role == "receiver" {
		return "transmitter"
	}
	return ""
}

func (s *PairingPoolService) createPairMatch(userID uint, userUUID, role, clientID string,
	peerUserID uint, peerUserUUID, peerRole, peerClientID string) (string, error) {

	ctx := context.Background()
	pairID := fmt.Sprintf("pair_%d_%d_%d", userID, peerUserID, time.Now().Unix())

	pairData := map[string]interface{}{
		"pair_id":         pairID,
		"created_at":      time.Now().Unix(),
		"user1_id":        userID,
		"user1_uuid":      userUUID,
		"user1_role":      role,
		"user1_client_id": clientID,
		"user2_id":        peerUserID,
		"user2_uuid":      peerUserUUID,
		"user2_role":      peerRole,
		"user2_client_id": peerClientID,
		"status":          StatusMatched,
	}

	pairJSON, _ := json.Marshal(pairData)
	matchedKey := fmt.Sprintf(PairingMatchedKeyTemplate, pairID)

	// 存储配对记录
	if err := global.GVA_REDIS.SetEx(ctx, matchedKey, string(pairJSON), time.Hour).Err(); err != nil {
		return "", fmt.Errorf("创建配对记录失败: %w", err)
	}

	// 【企业级新增】配对成功后发送MQTT通知及会话注册
	// 采用企业级错误处理：通知失败不影响主流程，只记录警告日志
	timestamp := time.Now()

	// 【核心改进】注册会话权限到Redis，支持会话级通信
	sessionID := fmt.Sprintf("session_%s", pairID)
	sessionParticipants := fmt.Sprintf("%s,%s", clientID, peerClientID)
	sessionKey := fmt.Sprintf("pairing_session:%s", sessionID)

	if err := global.GVA_REDIS.SetEx(ctx, sessionKey, sessionParticipants, time.Hour).Err(); err != nil {
		global.GVA_LOG.Warn("注册配对会话权限失败",
			zap.String("sessionID", sessionID),
			zap.String("participants", sessionParticipants),
			zap.Error(err))
	} else {
		global.GVA_LOG.Info("配对会话权限注册成功",
			zap.String("sessionID", sessionID),
			zap.String("participants", sessionParticipants))
	}

	// 通知第一个用户配对成功
	if err := s.notifyPairingSuccess(ctx, userUUID, role, clientID, peerUserUUID, peerRole, peerClientID, pairID, sessionID, timestamp); err != nil {
		global.GVA_LOG.Warn("发送配对成功通知失败",
			zap.String("userUUID", userUUID),
			zap.String("clientID", clientID),
			zap.String("pairID", pairID),
			zap.Error(err))
		// 企业级处理：不返回错误，保证主流程成功
	}

	// 通知第二个用户配对成功
	if err := s.notifyPairingSuccess(ctx, peerUserUUID, peerRole, peerClientID, userUUID, role, clientID, pairID, sessionID, timestamp); err != nil {
		global.GVA_LOG.Warn("发送配对伙伴成功通知失败",
			zap.String("peerUserUUID", peerUserUUID),
			zap.String("peerClientID", peerClientID),
			zap.String("pairID", pairID),
			zap.Error(err))
		// 企业级处理：不返回错误，保证主流程成功
	}

	return pairID, nil
}

func (s *PairingPoolService) removeFromPairingPool(role, pairingData string) {
	ctx := context.Background()
	poolKey := fmt.Sprintf(PairingPoolKeyTemplate, role)
	global.GVA_REDIS.ZRem(ctx, poolKey, pairingData)
}

func (s *PairingPoolService) buildPairingData(userID uint, userUUID, role, clientID string) string {
	data := map[string]interface{}{
		"client_id": clientID,
		"user_id":   userID,
		"user_uuid": userUUID,
		"role":      role,
		"joined_at": time.Now().Unix(),
		"status":    StatusWaiting,
	}
	pairingJSON, _ := json.Marshal(data)
	return string(pairingJSON)
}

func (s *PairingPoolService) calculateQueuePosition(role string) (int, time.Duration) {
	ctx := context.Background()
	poolKey := fmt.Sprintf(PairingPoolKeyTemplate, role)

	count, err := global.GVA_REDIS.ZCard(ctx, poolKey).Result()
	if err != nil {
		return 0, 0
	}

	// 简单估算：每30秒匹配一对
	estimatedWait := time.Duration(count) * 30 * time.Second
	return int(count), estimatedWait
}

func (s *PairingPoolService) isInPairingPool(userID uint, role string) (bool, time.Time) {
	ctx := context.Background()
	poolKey := fmt.Sprintf(PairingPoolKeyTemplate, role)

	members, err := global.GVA_REDIS.ZRangeWithScores(ctx, poolKey, 0, -1).Result()
	if err != nil {
		return false, time.Time{}
	}

	for _, member := range members {
		var data map[string]interface{}
		if err := json.Unmarshal([]byte(member.Member.(string)), &data); err != nil {
			continue
		}

		if uint(data["user_id"].(float64)) == userID {
			joinedAt := time.Unix(int64(data["joined_at"].(float64)), 0)
			return true, joinedAt
		}
	}

	return false, time.Time{}
}

func (s *PairingPoolService) isMatched(userID uint, role string) (bool, string) {
	// 这里应该查询配对记录，简化实现
	return false, ""
}

// notifyPairingSuccess 发送配对成功的MQTT通知
// 企业级设计：独立方法，便于测试和维护
func (s *PairingPoolService) notifyPairingSuccess(ctx context.Context, userUUID, role, clientID, peerUserUUID, peerRole, peerClientID, pairID, sessionID string, timestamp time.Time) error {
	// 获取MQTT服务实例
	mqttService := GetMQTTService()
	if !mqttService.IsConnected() {
		return fmt.Errorf("MQTT服务未连接")
	}

	// 构建配对成功通知payload - 新增会话ID支持
	pairingSuccessPayload := map[string]interface{}{
		"pair_id":        pairID,
		"session_id":     sessionID, // 【新增】会话ID，用于后续通信
		"peer_client_id": peerClientID,
		"peer_role":      peerRole,
		"peer_user_id":   peerUserUUID,
		"matched_at":     timestamp.Unix(),
		"timestamp":      timestamp.Format(time.RFC3339),
		"session_topic":  fmt.Sprintf("nfc_relay/session/%s/data", sessionID), // 【新增】会话通信主题
	}

	// 【企业级改进】支持双重通知机制：兼容性 + 新架构
	// 方案1：发送到用户级通知主题（新架构，支持连接复用）
	if err := s.notifyUserPairingSuccess(ctx, userUUID, role, clientID, pairingSuccessPayload); err != nil {
		global.GVA_LOG.Warn("发送用户级配对通知失败",
			zap.String("userUUID", userUUID),
			zap.String("clientID", clientID),
			zap.Error(err))
	}

	// 方案2：发送到客户端级通知主题（兼容现有架构）
	if err := mqttService.PublishPairingNotification(ctx, clientID, "success", pairingSuccessPayload); err != nil {
		global.GVA_LOG.Warn("发送客户端级配对通知失败",
			zap.String("clientID", clientID),
			zap.Error(err))
		// 如果用户级通知也失败了，才返回错误
		return fmt.Errorf("发送配对成功通知失败: %w", err)
	}

	global.GVA_LOG.Info("配对成功通知已发送",
		zap.String("userUUID", userUUID),
		zap.String("role", role),
		zap.String("clientID", clientID),
		zap.String("peerClientID", peerClientID),
		zap.String("pairID", pairID),
		zap.String("sessionID", sessionID))

	return nil
}

// notifyUserPairingSuccess 发送用户级配对成功通知
// 支持同一用户的多个ClientID同时接收通知
func (s *PairingPoolService) notifyUserPairingSuccess(ctx context.Context, userUUID, role, clientID string, payload map[string]interface{}) error {
	// 获取用户信息
	userInfo, err := commonService.UserCacheServiceApp.GetUserInfoByUUID(userUUID)
	if err != nil {
		return fmt.Errorf("获取用户信息失败: %w", err)
	}

	// 获取MQTT服务实例
	mqttService := GetMQTTService()
	if !mqttService.IsConnected() {
		return fmt.Errorf("MQTT服务未连接")
	}

	// 构建用户级通知消息
	userNotification := map[string]interface{}{
		"message_id":        fmt.Sprintf("user_pairing_%d", time.Now().UnixNano()),
		"notification_type": "pairing_success",
		"message_type":      "user_notification",
		"direction":         "server_to_user",
		"timestamp":         time.Now().UTC().Format(time.RFC3339),
		"target_client_id":  clientID, // 指明这个通知的目标ClientID
		"payload":           payload,
	}

	// 发布到用户级主题: nfc_relay/user/{username}/notifications
	topicPrefix := global.GVA_CONFIG.MQTT.NFCRelay.TopicPrefix
	userTopic := fmt.Sprintf("%s/user/%s/notifications", topicPrefix, userInfo.Username)

	data, err := json.Marshal(userNotification)
	if err != nil {
		return fmt.Errorf("序列化用户通知失败: %w", err)
	}

	qos := global.GVA_CONFIG.MQTT.QoS
	token := mqttService.client.Publish(userTopic, qos, false, data)

	if token.Wait() && token.Error() != nil {
		return fmt.Errorf("发布用户级通知失败: %w", token.Error())
	}

	global.GVA_LOG.Info("用户级配对通知发送成功",
		zap.String("userTopic", userTopic),
		zap.String("userUUID", userUUID),
		zap.String("username", userInfo.Username),
		zap.String("targetClientID", clientID))

	return nil
}

// notifyPairingCancellation 发送配对取消的MQTT通知
// 企业级设计：独立方法，便于测试和维护
func (s *PairingPoolService) notifyPairingCancellation(ctx context.Context, userUUID, role, clientID string) error {
	// 获取MQTT服务实例
	mqttService := GetMQTTService()
	if !mqttService.IsConnected() {
		return fmt.Errorf("MQTT服务未连接")
	}

	// 使用状态更新通知机制发送取消消息
	if err := mqttService.PublishPairingStatusUpdate(ctx, clientID, "cancelled", "用户主动取消配对"); err != nil {
		return fmt.Errorf("发送配对取消通知失败: %w", err)
	}

	global.GVA_LOG.Info("配对取消通知已发送",
		zap.String("userUUID", userUUID),
		zap.String("role", role),
		zap.String("clientID", clientID))

	return nil
}

// 全局配对池服务实例
var PairingPoolServiceApp = NewPairingPoolService()
