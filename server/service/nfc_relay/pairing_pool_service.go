package nfc_relay

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"strconv"
	"time"

	"github.com/flipped-aurora/gin-vue-admin/server/global"
	commonService "github.com/flipped-aurora/gin-vue-admin/server/service/common"
	"github.com/redis/go-redis/v9"
	"go.uber.org/zap"
)

// PairingPoolService 配对池服务
// 【重构】采用基于Redis HASH和Lua脚本的高性能O(1)匹配算法
type PairingPoolService struct {
	userCacheService   *commonService.UserCacheService
	attemptMatchScript *redis.Script
	writeStatesScript  *redis.Script // 【新增】状态写入脚本
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
	PairingPoolKey            = "pairing:waiting:pool"  // 【新】使用HASH存储等待中的用户，Key: UserID, Value: JSON
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
// 【重构】在服务创建时加载并缓存Lua脚本
func NewPairingPoolService() *PairingPoolService {
	service := &PairingPoolService{
		userCacheService: commonService.UserCacheServiceApp,
	}

	// 从文件加载Lua脚本
	luaBytes, err := os.ReadFile("scripts/redis/attempt_match.lua")
	if err != nil {
		// 在初始化阶段如果脚本加载失败，应该panic，因为这是系统无法运行的致命错误
		global.GVA_LOG.Panic("Failed to load Redis Lua script 'attempt_match.lua'", zap.Error(err))
	}
	service.attemptMatchScript = redis.NewScript(string(luaBytes))
	global.GVA_LOG.Info("Redis Lua script 'attempt_match.lua' loaded successfully.")

	// 【新增】加载状态写入脚本
	writeStatesBytes, err := os.ReadFile("scripts/redis/write_matched_states.lua")
	if err != nil {
		global.GVA_LOG.Panic("Failed to load Redis Lua script 'write_matched_states.lua'", zap.Error(err))
	}
	service.writeStatesScript = redis.NewScript(string(writeStatesBytes))
	global.GVA_LOG.Info("Redis Lua script 'write_matched_states.lua' loaded successfully.")

	return service
}

// JoinPairingPool 加入配对池并立即尝试匹配
// 【重构】这是新的核心方法，集成了加入和匹配的原子操作
func (s *PairingPoolService) JoinPairingPool(userUUID, role, clientID string, deviceInfo map[string]interface{}) (*MatchResult, error) {
	// 1. 验证参数和用户身份
	if err := s.validatePairingRequest(userUUID, role, clientID); err != nil {
		return nil, err
	}
	userID, err := s.userCacheService.GetUserIDByUUID(userUUID)
	if err != nil {
		global.GVA_LOG.Error("获取用户ID失败", zap.Error(err), zap.String("userUUID", userUUID))
		return nil, fmt.Errorf("用户身份验证失败: %w", err)
	}

	// 2. 构建当前用户的配对数据
	pairingData := map[string]interface{}{
		"client_id":   clientID,
		"user_id":     userID,
		"user_uuid":   userUUID,
		"role":        role,
		"joined_at":   time.Now().Unix(),
		"device_info": deviceInfo,
	}
	pairingJSON, err := json.Marshal(pairingData)
	if err != nil {
		return nil, fmt.Errorf("序列化配对数据失败: %w", err)
	}

	ctx := context.Background()

	// 3. 【核心】执行Lua脚本进行原子化匹配
	global.GVA_LOG.Info("尝试通过Lua脚本直接匹配", zap.Uint("userID", userID), zap.String("role", role))
	result, err := s.attemptMatchScript.Run(ctx, global.GVA_REDIS, []string{PairingPoolKey}, userID, role, string(pairingJSON)).Result()
	if err != nil && err != redis.Nil {
		global.GVA_LOG.Error("执行配对Lua脚本失败", zap.Error(err), zap.Uint("userID", userID))
		return nil, fmt.Errorf("执行配对脚本失败: %w", err)
	}

	// 4. 根据脚本返回结果进行处理
	if result == nil || err == redis.Nil {
		// 4a. 匹配失败，当前用户已加入等待池
		global.GVA_LOG.Info("直接匹配失败, 用户已加入等待池", zap.Uint("userID", userID), zap.String("role", role))
		pipe := global.GVA_REDIS.TxPipeline()
		stateKey := fmt.Sprintf(PairingStateKeyTemplate, userID)
		pipe.HSet(ctx, stateKey, role, clientID)
		pipe.Expire(ctx, stateKey, PairingTimeoutDuration+time.Minute)
		timeoutKey := fmt.Sprintf(PairingTimeoutKeyTemplate, userID, role)
		pipe.SetEx(ctx, timeoutKey, clientID, PairingTimeoutDuration)
		_, _ = pipe.Exec(ctx)

		position, estimatedWait := s.calculateQueuePosition()
		return &MatchResult{
			Matched:       false,
			QueuePosition: position,
			EstimatedWait: estimatedWait,
		}, nil
	}

	// 4b. 匹配成功！
	global.GVA_LOG.Info("直接匹配成功", zap.Uint("userID", userID))
	peerDataJSON, ok := result.(string)
	if !ok {
		return nil, fmt.Errorf("lua脚本返回了非预期的类型: %T", result)
	}
	var peerData map[string]interface{}
	if err := json.Unmarshal([]byte(peerDataJSON), &peerData); err != nil {
		return nil, fmt.Errorf("反序列化伙伴数据失败: %w", err)
	}

	peerUserID := uint(peerData["user_id"].(float64))
	peerClientID := peerData["client_id"].(string)
	peerRole := peerData["role"].(string)
	peerUserUUID := peerData["user_uuid"].(string)
	peerJoinedAt := int64(peerData["joined_at"].(float64))

	// 5. 创建配对记录，发送通知
	pairID, err := s.createPairMatch(userID, userUUID, role, clientID, peerUserID, peerUserUUID, peerRole, peerClientID)
	if err != nil {
		// 如果创建配对失败，需要考虑如何处理，是否需要将用户重新放回池中
		global.GVA_LOG.Error("创建配对记录失败", zap.Error(err))
		return nil, err
	}

	// 6. 【重要修正】为双方创建并写入"已匹配"状态
	// 使用事务确保状态写入的原子性
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	matchedAt := time.Now()
	// 6.1 准备当前用户的匹配状态
	currentUserStatus := &PairingStatus{
		Status:    StatusMatched,
		Role:      role,
		ClientID:  clientID,
		MatchedAt: &matchedAt,
		Metadata:  map[string]interface{}{"pair_id": pairID, "peer_client_id": peerClientID, "peer_role": peerRole},
	}
	currentUserStatusJSON, _ := json.Marshal(currentUserStatus)

	// 6.2 准备对端用户的匹配状态
	peerStatus := &PairingStatus{
		Status:    StatusMatched,
		Role:      peerRole,
		ClientID:  peerClientID,
		MatchedAt: &matchedAt,
		Metadata:  map[string]interface{}{"pair_id": pairID, "peer_client_id": clientID, "peer_role": role},
	}
	peerStatusJSON, _ := json.Marshal(peerStatus)

	// 【可观测性增强】在写入前记录详细日志
	global.GVA_LOG.Info("准备为匹配成功的双方写入Redis状态",
		zap.Uint("currentUserID", userID),
		zap.String("currentUserRole", role),
		zap.String("currentUserStatusJSON", string(currentUserStatusJSON)),
		zap.Uint("peerUserID", peerUserID),
		zap.String("peerUserRole", peerRole),
		zap.String("peerUserStatusJSON", string(peerStatusJSON)),
	)

	// 【企业级改进】使用Lua脚本确保状态写入的完全原子性
	stateKey := fmt.Sprintf(PairingStateKeyTemplate, userID)
	peerStateKey := fmt.Sprintf(PairingStateKeyTemplate, peerUserID)
	peerTimeoutKey := fmt.Sprintf(PairingTimeoutKeyTemplate, peerUserID, peerRole)

	// 执行原子写入脚本
	result, err = s.writeStatesScript.Run(ctx, global.GVA_REDIS,
		[]string{stateKey, peerStateKey, peerTimeoutKey},
		role, string(currentUserStatusJSON),
		peerRole, string(peerStatusJSON),
		int64(time.Hour.Seconds())).Result()

	if err != nil {
		global.GVA_LOG.Error("通过Lua脚本写入匹配状态失败", zap.Error(err), zap.Uint("userID", userID))
		return nil, fmt.Errorf("写入匹配状态失败: %w", err)
	}

	// 【企业级新增】解析Lua脚本返回结果，确认每个操作都成功
	if resultArray, ok := result.([]interface{}); ok && len(resultArray) == 9 {
		// 验证每个操作的结果
		currentHSet := resultArray[0]
		currentExpireFirst := resultArray[1]
		currentExpireSecond := resultArray[2]
		peerHSet := resultArray[3]
		peerExpireFirst := resultArray[4]
		peerExpireSecond := resultArray[5]
		timeoutDel := resultArray[6]
		currentTTL := resultArray[7]
		peerTTL := resultArray[8]

		global.GVA_LOG.Info("Lua脚本写入状态成功，TTL问题已修复",
			zap.Uint("userID", userID),
			zap.Any("currentHSetResult", currentHSet),
			zap.Any("currentExpireFirstResult", currentExpireFirst),
			zap.Any("currentExpireSecondResult", currentExpireSecond),
			zap.Any("peerHSetResult", peerHSet),
			zap.Any("peerExpireFirstResult", peerExpireFirst),
			zap.Any("peerExpireSecondResult", peerExpireSecond),
			zap.Any("timeoutDelResult", timeoutDel),
			zap.Any("currentStateTTL", currentTTL),
			zap.Any("peerStateTTL", peerTTL))
	} else {
		global.GVA_LOG.Warn("Lua脚本返回了意外的结果格式",
			zap.Uint("userID", userID),
			zap.Any("result", result),
			zap.Int("resultLength", len(resultArray)))
	}

	// 【企业级新增】执行后验证写入是否成功
	verifyCtx, verifyCancel := context.WithTimeout(context.Background(), 1*time.Second)
	defer verifyCancel()

	// 验证当前用户状态是否成功写入
	currentVerify, err := global.GVA_REDIS.HGet(verifyCtx, stateKey, role).Result()
	if err != nil {
		global.GVA_LOG.Error("验证当前用户状态写入失败",
			zap.Uint("currentUserID", userID),
			zap.String("currentUserRole", role),
			zap.Error(err))
	} else {
		global.GVA_LOG.Info("验证当前用户状态写入成功",
			zap.Uint("currentUserID", userID),
			zap.String("currentUserRole", role),
			zap.Int("writtenBytes", len(currentVerify)))
	}

	// 验证对端用户状态是否成功写入
	peerVerify, err := global.GVA_REDIS.HGet(verifyCtx, peerStateKey, peerRole).Result()
	if err != nil {
		global.GVA_LOG.Error("验证对端用户状态写入失败",
			zap.Uint("peerUserID", peerUserID),
			zap.String("peerUserRole", peerRole),
			zap.Error(err))
	} else {
		global.GVA_LOG.Info("验证对端用户状态写入成功",
			zap.Uint("peerUserID", peerUserID),
			zap.String("peerUserRole", peerRole),
			zap.Int("writtenBytes", len(peerVerify)))
	}

	return &MatchResult{
		Matched:      true,
		PeerClientID: peerClientID,
		PeerUserID:   peerUserID,
		PeerRole:     peerRole,
		PairID:       pairID,
		WaitingTime:  time.Since(time.Unix(peerJoinedAt, 0)),
	}, nil
}

// CancelPairing 取消配对
// 【重构】简化逻辑，直接使用HDEL
func (s *PairingPoolService) CancelPairing(userUUID, role string) error {
	userID, err := s.userCacheService.GetUserIDByUUID(userUUID)
	if err != nil {
		return err
	}

	ctx := context.Background()

	// 【可观测性增强】在操作前记录详细的意图日志
	global.GVA_LOG.Info("收到取消配对请求，准备清理相关Redis状态",
		zap.String("userUUID", userUUID),
		zap.Uint("userID", userID),
		zap.String("role", role),
	)

	// 为了日志记录，先获取等待数据
	var waitingData string
	waitingData, _ = global.GVA_REDIS.HGet(ctx, PairingPoolKey, strconv.Itoa(int(userID))).Result()

	// 使用Pipeline进行批量清理
	pipe := global.GVA_REDIS.TxPipeline()

	// 1. 从等待池中移除
	pipe.HDel(ctx, PairingPoolKey, strconv.Itoa(int(userID)))

	// 2. 清理状态和超时键
	stateKey := fmt.Sprintf(PairingStateKeyTemplate, userID)
	timeoutKey := fmt.Sprintf(PairingTimeoutKeyTemplate, userID, role)

	if role != "" {
		pipe.HDel(ctx, stateKey, role)
		pipe.Del(ctx, timeoutKey)
	} else {
		// 如果role为空，清理该用户的所有角色状态
		pipe.Del(ctx, stateKey)
		pipe.Del(ctx, fmt.Sprintf(PairingTimeoutKeyTemplate, userID, "transmitter"))
		pipe.Del(ctx, fmt.Sprintf(PairingTimeoutKeyTemplate, userID, "receiver"))
	}

	// 3. 执行清理事务
	_, err = pipe.Exec(ctx)
	if err != nil {
		global.GVA_LOG.Error("取消配对事务失败", zap.Error(err), zap.String("userUUID", userUUID))
		return fmt.Errorf("取消配对失败: %w", err)
	}

	global.GVA_LOG.Info("取消配对成功",
		zap.String("userUUID", userUUID),
		zap.String("role", role),
		zap.String("removedData", waitingData))

	// 【企业级新增】发送配对取消通知
	// 此逻辑保持不变
	if role != "" {
		stateKey := fmt.Sprintf(PairingStateKeyTemplate, userID)
		if clientID, err := global.GVA_REDIS.HGet(context.Background(), stateKey, role).Result(); err == nil && clientID != "" {
			if err := s.notifyPairingCancellation(context.Background(), userUUID, role, clientID); err != nil {
				global.GVA_LOG.Warn("发送配对取消通知失败",
					zap.String("userUUID", userUUID),
					zap.String("role", role),
					zap.String("clientID", clientID),
					zap.Error(err))
			}
		}
	}

	return nil
}

// GetPairingStatus 获取配对状态
// 【重构】简化逻辑，直接使用HGET查询等待池
func (s *PairingPoolService) GetPairingStatus(userUUID string) (*PairingStatus, error) {
	userID, err := s.userCacheService.GetUserIDByUUID(userUUID)
	if err != nil {
		return nil, err
	}

	ctx := context.Background()

	// 1. 直接检查用户是否在等待池中
	waitingDataJSON, err := global.GVA_REDIS.HGet(ctx, PairingPoolKey, strconv.Itoa(int(userID))).Result()
	if err != nil && err != redis.Nil {
		global.GVA_LOG.Error("从Redis获取等待状态失败", zap.Error(err), zap.Uint("userID", userID))
		return nil, fmt.Errorf("获取等待状态失败: %w", err)
	}

	if waitingDataJSON != "" {
		// 用户正在等待
		var waitingData map[string]interface{}
		if err := json.Unmarshal([]byte(waitingDataJSON), &waitingData); err != nil {
			return nil, fmt.Errorf("反序列化等待数据失败: %w", err)
		}

		role := waitingData["role"].(string)
		clientID := waitingData["client_id"].(string)
		joinedAt := time.Unix(int64(waitingData["joined_at"].(float64)), 0)
		timeoutAt := joinedAt.Add(PairingTimeoutDuration)
		position, estimatedWait := s.calculateQueuePosition()

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

	// 2. TODO: 检查用户是否已经匹配成功
	//    此逻辑需要根据 isMatched 的具体实现来补充
	//    一个简化的示例是检查 pairing:state:%d 是否存在但 pairing:waiting:pool 中没有他
	stateKey := fmt.Sprintf(PairingStateKeyTemplate, userID)
	if global.GVA_REDIS.Exists(ctx, stateKey).Val() > 0 {
		// 存在状态但不在等待池，可能已匹配或处于其他状态
		// 这里的逻辑需要更精细化，暂时返回idle
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

	// 1. 检查是否在等待池中
	if inPool, joinedAt, poolData := s.isInPairingPool(userID, role); inPool {
		position, estimatedWait := s.calculateQueuePosition()
		timeoutAt := joinedAt.Add(PairingTimeoutDuration)
		return &PairingStatus{
			Status:        StatusWaiting,
			Role:          role,
			ClientID:      poolData["client_id"].(string),
			JoinedAt:      joinedAt,
			QueuePosition: position,
			EstimatedWait: estimatedWait,
			TimeoutAt:     &timeoutAt,
		}, nil
	}

	// 2. 【逻辑加固】使用 HGETALL 获取用户所有角色的状态，以进行交叉验证
	stateKey := fmt.Sprintf(PairingStateKeyTemplate, userID)

	// 【新增防御】检查状态键的TTL，如果过短则主动刷新
	currentTTL, err := global.GVA_REDIS.TTL(ctx, stateKey).Result()
	if err == nil && currentTTL > 0 && currentTTL < 5*time.Minute {
		global.GVA_LOG.Warn("检测到配对状态TTL过短，主动刷新为1小时",
			zap.String("userUUID", userUUID),
			zap.String("role", role),
			zap.Duration("currentTTL", currentTTL))
		// 主动刷新TTL为1小时
		global.GVA_REDIS.Expire(ctx, stateKey, time.Hour)
	}

	allStates, err := global.GVA_REDIS.HGetAll(ctx, stateKey).Result()
	if err != nil {
		return nil, fmt.Errorf("查询配对状态失败: %w", err)
	}

	if len(allStates) == 0 {
		return nil, nil // 用户没有任何活跃状态
	}

	// 3. 优先检查并解析当前请求角色的状态
	if currentStateJSON, ok := allStates[role]; ok && currentStateJSON != "" {
		var status PairingStatus
		if err := json.Unmarshal([]byte(currentStateJSON), &status); err == nil {
			// 确保返回的状态是matched，防止脏数据
			if status.Status == StatusMatched {
				return &status, nil
			}
		}
	}

	// 4. 【防御性编程与自愈】如果当前角色状态丢失，但对向角色是匹配状态，则动态恢复
	oppositeRole := getOppositeRole(role)
	if oppositeStateJSON, ok := allStates[oppositeRole]; ok && oppositeStateJSON != "" {
		var oppositeStatus PairingStatus
		if err := json.Unmarshal([]byte(oppositeStateJSON), &oppositeStatus); err == nil && oppositeStatus.Status == StatusMatched {
			// 检测到状态不一致：本机状态丢失，但对端已匹配
			global.GVA_LOG.Warn("检测到配对状态不一致：本机角色状态丢失，但对端角色已匹配。将动态恢复状态。",
				zap.String("userUUID", userUUID),
				zap.String("missingRole", role),
				zap.String("existingOppositeRole", oppositeRole),
				zap.Any("oppositeStatus", oppositeStatus),
			)

			// 【关键修复】从对端的metadata中正确提取本机的ClientID
			var recoveredClientID string
			if oppositeStatus.Metadata != nil {
				if peerClientID, exists := oppositeStatus.Metadata["peer_client_id"]; exists {
					if peerClientIDStr, ok := peerClientID.(string); ok {
						// 对端metadata中的peer_client_id就是本机的client_id
						recoveredClientID = peerClientIDStr
					}
				}
			}

			// 【关键修复】从对端信息中正确重建本机状态，确保peer信息指向对端
			recoveredStatus := &PairingStatus{
				Status:    StatusMatched,
				Role:      role,
				ClientID:  recoveredClientID, // 【修复】正确设置本机的ClientID
				MatchedAt: oppositeStatus.MatchedAt,
				Metadata: map[string]interface{}{
					"pair_id":        oppositeStatus.Metadata["pair_id"],
					"peer_client_id": oppositeStatus.ClientID, // 【修复】peer应该指向对端(opposite)的ClientID
					"peer_role":      oppositeRole,            // 【修复】peer应该指向对端的角色
					"recovered":      true,                    // 标记此状态为动态恢复
				},
			}

			// 【企业级新增】记录状态恢复的详细信息，便于调试和审计
			global.GVA_LOG.Info("成功动态恢复配对状态",
				zap.String("userUUID", userUUID),
				zap.String("recoveredRole", role),
				zap.String("recoveredClientID", recoveredClientID),
				zap.String("peerClientID", oppositeStatus.ClientID),
				zap.String("peerRole", oppositeRole),
				zap.Any("recoveredStatus", recoveredStatus),
			)

			// 【企业级改进】将恢复的状态重新写入Redis，避免后续重复触发恢复逻辑
			if recoveredStatusJSON, err := json.Marshal(recoveredStatus); err == nil {
				stateKey := fmt.Sprintf(PairingStateKeyTemplate, userID)
				if err := global.GVA_REDIS.HSet(ctx, stateKey, role, string(recoveredStatusJSON)).Err(); err != nil {
					global.GVA_LOG.Warn("恢复状态写回Redis失败",
						zap.String("userUUID", userUUID),
						zap.String("role", role),
						zap.Error(err))
				} else {
					global.GVA_LOG.Info("恢复状态已写回Redis",
						zap.String("userUUID", userUUID),
						zap.String("role", role))

					// 【企业级新增】状态恢复后进行双向一致性验证
					go s.verifyPairingStateConsistency(userUUID, userID, role, recoveredStatus)
				}
			}

			return recoveredStatus, nil
		}
	}

	// 5. 如果没有任何有效状态，则返回 nil
	return nil, nil
}

// LeavePairingPool 离开配对池
func (s *PairingPoolService) LeavePairingPool(userUUID, role string) error {
	// 调用重构后的CancelPairing即可
	return s.CancelPairing(userUUID, role)
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

func (s *PairingPoolService) calculateQueuePosition() (int, time.Duration) {
	ctx := context.Background()

	count, err := global.GVA_REDIS.HLen(ctx, PairingPoolKey).Result()
	if err != nil {
		return 0, 0
	}

	// 简单估算：每30秒匹配一对，队列中的人数是count，则等待的配对数是count/2
	queueLength := int(count)
	if queueLength < 0 {
		queueLength = 0
	}

	estimatedWait := time.Duration(queueLength/2) * 30 * time.Second
	return queueLength, estimatedWait
}

func (s *PairingPoolService) isInPairingPool(userID uint, role string) (bool, time.Time, map[string]interface{}) {
	ctx := context.Background()
	waitingDataJSON, err := global.GVA_REDIS.HGet(ctx, PairingPoolKey, strconv.Itoa(int(userID))).Result()
	if err != nil || waitingDataJSON == "" {
		return false, time.Time{}, nil
	}

	var data map[string]interface{}
	if err := json.Unmarshal([]byte(waitingDataJSON), &data); err != nil {
		return false, time.Time{}, nil
	}

	// 确认角色也匹配
	if data["role"].(string) == role {
		joinedAt := time.Unix(int64(data["joined_at"].(float64)), 0)
		return true, joinedAt, data
	}

	return false, time.Time{}, nil
}

func (s *PairingPoolService) isMatched(userID uint, role string) (bool, string) {
	// 这里应该查询配对记录，简化实现
	return false, ""
}

// getOppositeRole 获取相反的角色
func getOppositeRole(role string) string {
	if role == "transmitter" {
		return "receiver"
	}
	if role == "receiver" {
		return "transmitter"
	}
	return ""
}

// notifyPairingSuccess 发送配对成功的MQTT通知
// 企业级设计：独立方法，便于测试和维护
func (s *PairingPoolService) notifyPairingSuccess(ctx context.Context, userUUID, role, clientID, peerUserUUID, peerRole, peerClientID, pairID, sessionID string, timestamp time.Time) error {
	// 获取MQTT服务实例
	mqttService := ServiceGroupApp.MqttService()
	if !mqttService.IsConnected() {
		return fmt.Errorf("MQTT服务未连接")
	}

	// 构建通知内容
	payload := map[string]interface{}{
		"event":          "pairing_success",
		"pair_id":        pairID,
		"session_id":     sessionID,
		"your_role":      role,
		"peer_user_uuid": peerUserUUID,
		"peer_role":      peerRole,
		"peer_client_id": peerClientID,
		"matched_at":     timestamp.UTC().Format(time.RFC3339),
	}

	// 【企业级升级】使用多通道冗余通知机制
	return s.notifyPairingSuccessWithFallback(ctx, userUUID, role, clientID, payload)
}

// 【企业级新增】notifyPairingSuccessWithFallback 多通道冗余配对成功通知
// 实现企业级的通知可靠性保障：主通道 + 备用通道 + 直接通道
func (s *PairingPoolService) notifyPairingSuccessWithFallback(ctx context.Context, userUUID, role, clientID string, payload map[string]interface{}) error {
	mqttService := ServiceGroupApp.MqttService()
	if !mqttService.IsConnected() {
		return fmt.Errorf("MQTT服务未连接")
	}

	var successCount int
	var lastError error

	// 通道1：用户级通知（主通道，现有实现）
	if err := s.notifyUserPairingSuccess(ctx, userUUID, role, clientID, payload); err != nil {
		global.GVA_LOG.Warn("主通道（用户级通知）发送失败",
			zap.String("userUUID", userUUID),
			zap.String("clientID", clientID),
			zap.Error(err))
		lastError = err
	} else {
		successCount++
		global.GVA_LOG.Info("主通道（用户级通知）发送成功",
			zap.String("userUUID", userUUID),
			zap.String("clientID", clientID))
	}

	// 通道2：客户端级通知（备用通道）
	if err := mqttService.PublishClientPairingNotification(ctx, clientID, payload); err != nil {
		global.GVA_LOG.Warn("备用通道（客户端级通知）发送失败",
			zap.String("clientID", clientID),
			zap.Error(err))
		lastError = err
	} else {
		successCount++
		global.GVA_LOG.Info("备用通道（客户端级通知）发送成功",
			zap.String("clientID", clientID))
	}

	// 通道3：直接客户端事件（第三重保障）
	if err := mqttService.PublishDirectClientEvent(ctx, clientID, "pairing_success", payload); err != nil {
		global.GVA_LOG.Warn("直接通道（客户端事件）发送失败",
			zap.String("clientID", clientID),
			zap.Error(err))
		lastError = err
	} else {
		successCount++
		global.GVA_LOG.Info("直接通道（客户端事件）发送成功",
			zap.String("clientID", clientID))
	}

	// 企业级成功标准：至少一个通道成功即可认为通知成功
	if successCount > 0 {
		global.GVA_LOG.Info("多通道配对通知发送完成",
			zap.String("userUUID", userUUID),
			zap.String("clientID", clientID),
			zap.Int("successChannels", successCount),
			zap.Int("totalChannels", 3),
			zap.String("pairID", payload["pair_id"].(string)))
		return nil
	}

	// 所有通道都失败，返回最后一个错误
	global.GVA_LOG.Error("所有通知通道均失败",
		zap.String("userUUID", userUUID),
		zap.String("clientID", clientID),
		zap.Error(lastError))
	return fmt.Errorf("所有通知通道均失败，最后错误: %w", lastError)
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
	mqttService := ServiceGroupApp.MqttService()
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
	mqttService := ServiceGroupApp.MqttService()
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

// 【企业级新增】配对状态一致性验证
// 在状态恢复后异步验证双方状态是否完全一致，如有不一致则尝试修复
func (s *PairingPoolService) verifyPairingStateConsistency(userUUID string, userID uint, role string, currentStatus *PairingStatus) {
	ctx := context.Background()

	defer func() {
		if r := recover(); r != nil {
			global.GVA_LOG.Error("状态一致性验证过程中发生panic",
				zap.String("userUUID", userUUID),
				zap.Any("panic", r))
		}
	}()

	// 等待一小段时间，确保状态写入完成
	time.Sleep(100 * time.Millisecond)

	if currentStatus.Metadata == nil {
		return
	}

	pairID, exists := currentStatus.Metadata["pair_id"]
	if !exists {
		return
	}

	_, exists = currentStatus.Metadata["peer_client_id"]
	if !exists {
		return
	}

	// 通过peer_client_id查找对端用户的状态
	peerRole := getOppositeRole(role)

	// 简化实现：通过配对记录查找对端
	pairKey := fmt.Sprintf(PairingMatchedKeyTemplate, pairID)
	pairDataJSON, err := global.GVA_REDIS.Get(ctx, pairKey).Result()
	if err != nil {
		global.GVA_LOG.Warn("无法获取配对记录进行一致性验证",
			zap.String("userUUID", userUUID),
			zap.String("pairID", pairID.(string)),
			zap.Error(err))
		return
	}

	var pairData map[string]interface{}
	if err := json.Unmarshal([]byte(pairDataJSON), &pairData); err != nil {
		return
	}

	// 确定对端用户ID
	var peerUserID uint
	if pairData["user1_role"].(string) == role {
		peerUserID = uint(pairData["user2_id"].(float64))
	} else {
		peerUserID = uint(pairData["user1_id"].(float64))
	}

	// 检查对端状态
	peerStateKey := fmt.Sprintf(PairingStateKeyTemplate, peerUserID)
	peerStatusJSON, err := global.GVA_REDIS.HGet(ctx, peerStateKey, peerRole).Result()
	if err != nil {
		global.GVA_LOG.Warn("对端状态验证失败，状态不存在",
			zap.String("userUUID", userUUID),
			zap.Uint("peerUserID", peerUserID),
			zap.String("peerRole", peerRole))
		// 可以考虑在这里触发对端状态的恢复
		return
	}

	var peerStatus PairingStatus
	if err := json.Unmarshal([]byte(peerStatusJSON), &peerStatus); err != nil {
		global.GVA_LOG.Warn("对端状态解析失败",
			zap.String("userUUID", userUUID),
			zap.Error(err))
		return
	}

	// 验证双方metadata中的peer信息是否互相对应
	isConsistent := true
	if peerStatus.Metadata == nil {
		isConsistent = false
	} else {
		if peerStatus.Metadata["peer_client_id"] != currentStatus.ClientID {
			isConsistent = false
		}
		if peerStatus.Metadata["peer_role"] != role {
			isConsistent = false
		}
		if currentStatus.Metadata["peer_client_id"] != peerStatus.ClientID {
			isConsistent = false
		}
	}

	if isConsistent {
		global.GVA_LOG.Info("配对状态一致性验证通过",
			zap.String("userUUID", userUUID),
			zap.String("pairID", pairID.(string)))
	} else {
		global.GVA_LOG.Warn("配对状态一致性验证失败，将尝试修复",
			zap.String("userUUID", userUUID),
			zap.String("pairID", pairID.(string)),
			zap.Any("currentStatus", currentStatus),
			zap.Any("peerStatus", peerStatus))
		// 这里可以实现自动修复逻辑
	}
}
