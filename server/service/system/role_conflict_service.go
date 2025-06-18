package system

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/flipped-aurora/gin-vue-admin/server/global"
	"github.com/flipped-aurora/gin-vue-admin/server/model/system/request"
	"github.com/flipped-aurora/gin-vue-admin/server/utils"
	"github.com/redis/go-redis/v9"
	"go.uber.org/zap"
)

type RoleConflictService struct{}

// CheckRoleConflict 检查角色冲突
// 检查指定的userID和role是否已经被另一个clientID占用
func (s *RoleConflictService) CheckRoleConflict(userID, role, newClientID string) (*request.ConflictCheckResult, error) {
	ctx := context.Background()
	userRoleKey := fmt.Sprintf("user:%s:roles", userID)

	// 1. 检查用户当前的角色分配
	currentAssignment, err := global.GVA_REDIS.HGet(ctx, userRoleKey, role).Result()
	if err != nil && err != redis.Nil {
		global.GVA_LOG.Error("检查用户角色失败", zap.Error(err), zap.String("key", userRoleKey), zap.String("role", role))
		return nil, fmt.Errorf("检查用户角色失败: %v", err)
	}

	// 2. 如果角色未被分配，则无冲突
	if err == redis.Nil || currentAssignment == "" {
		return &request.ConflictCheckResult{HasConflict: false}, nil
	}

	// 3. 解析当前分配信息
	currentClientID := s.extractFromAssignment(currentAssignment, "client_id")
	if currentClientID == "" {
		// 数据格式错误，当作无冲突处理，后续AssignRole会覆盖
		global.GVA_LOG.Warn("角色分配信息格式错误，无法解析client_id", zap.String("assignment", currentAssignment))
		return &request.ConflictCheckResult{HasConflict: false}, nil
	}

	// 4. 如果是同一设备重连，无冲突
	if currentClientID == newClientID {
		return &request.ConflictCheckResult{HasConflict: false}, nil
	}

	// 5. 不同设备，存在冲突
	conflictDevice := &request.ConflictDeviceInfo{
		ClientID:     currentClientID,
		DeviceModel:  s.extractFromAssignment(currentAssignment, "device"),
		ConnectedAt:  s.extractFromAssignment(currentAssignment, "connected_at"),
		LastActivity: s.getLastActivity(currentClientID),
	}

	return &request.ConflictCheckResult{
		HasConflict:    true,
		ConflictDevice: conflictDevice,
		CanForceKick:   true, // 默认允许强制挤下线
	}, nil
}

// AssignRole 执行角色分配（包含挤下线）
func (s *RoleConflictService) AssignRole(userID, role, clientID string, jti string, deviceInfo map[string]interface{}, forceKick bool) error {
	ctx := context.Background()
	pipe := global.GVA_REDIS.TxPipeline()

	// 1. 如果需要，处理强制挤下线
	if forceKick {
		if err := s.handleForceKick(ctx, pipe, userID, role, clientID); err != nil {
			return fmt.Errorf("处理强制挤下线失败: %w", err)
		}
	}

	// 2. 创建并分配新角色信息
	assignmentInfo := s.createAssignmentInfo(clientID, deviceInfo)
	userRoleKey := fmt.Sprintf("user:%s:roles", userID)
	pipe.HSet(ctx, userRoleKey, role, assignmentInfo)

	// 3. 更新角色占用状态
	roleAssignmentKey := fmt.Sprintf("role_assignments:%s", role)
	roleAssignmentValue := map[string]interface{}{
		"current_user": userID,
		"client_id":    clientID,
		"assigned_at":  time.Now().Unix(),
	}
	pipe.HSet(ctx, roleAssignmentKey, roleAssignmentValue)

	// 4. 记录客户端连接状态 (增加jti用于后续的JWT吊销)
	now := time.Now().Unix()
	connectionInfo := fmt.Sprintf("user:%s|role:%s|connected_at:%d|last_ping:%d|jti:%s",
		userID, role, now, now, jti)
	pipe.HSet(ctx, "client_connections", clientID, connectionInfo)

	// 5. 为关键key设置过期时间，防止数据无限增长
	pipe.Expire(ctx, userRoleKey, 24*time.Hour)
	pipe.Expire(ctx, roleAssignmentKey, 24*time.Hour)

	// 6. 执行事务
	if _, err := pipe.Exec(ctx); err != nil {
		global.GVA_LOG.Error("分配角色事务执行失败", zap.Error(err))
		return fmt.Errorf("分配角色事务执行失败: %w", err)
	}
	return nil
}

// handleForceKick 处理强制挤下线（在Redis事务中执行）
func (s *RoleConflictService) handleForceKick(ctx context.Context, pipe redis.Pipeliner, userID, role, newClientID string) error {
	// 1. 获取被挤设备的信息
	userRoleKey := fmt.Sprintf("user:%s:roles", userID)
	currentAssignment, err := global.GVA_REDIS.HGet(ctx, userRoleKey, role).Result()
	if err != nil || currentAssignment == "" {
		return nil // 如果查询失败或角色未分配，则无需处理
	}

	targetClientID := s.extractFromAssignment(currentAssignment, "client_id")
	if targetClientID == "" || targetClientID == newClientID {
		return nil // 没有有效的旧客户端或与新客户端相同，无需处理
	}

	// 2. 立即撤销旧JWT（不等待异步处理）
	if err := s.revokeOldClientJWT(ctx, targetClientID); err != nil {
		global.GVA_LOG.Error("立即撤销旧JWT失败", zap.Error(err), zap.String("targetClientID", targetClientID))
		// 不返回错误，继续处理其他步骤
	}

	// 3. 创建挤下线通知（用于额外的清理工作）
	kickNotification := map[string]interface{}{
		"target_client_id": targetClientID,
		"kicker_client_id": newClientID,
		"role":             role,
		"reason":           "role_taken_by_another_device",
		"timestamp":        time.Now().Unix(),
	}
	notificationJSON, _ := json.Marshal(kickNotification)
	pipe.LPush(ctx, "kick_notifications", string(notificationJSON))

	// 4. 清除被挤设备的状态
	pipe.HDel(ctx, "client_connections", targetClientID)

	global.GVA_LOG.Info("已处理强制挤下线",
		zap.String("target_client_id", targetClientID),
		zap.String("kicker_client_id", newClientID))

	return nil
}

// revokeOldClientJWT 立即撤销旧客户端的JWT并断开EMQX连接
func (s *RoleConflictService) revokeOldClientJWT(ctx context.Context, clientID string) error {
	// 从 client_connections 中获取JTI和UserID
	connectionInfo, err := global.GVA_REDIS.HGet(ctx, "client_connections", clientID).Result()
	if err != nil {
		return fmt.Errorf("获取连接信息失败: %w", err)
	}

	var jti, userID string
	pairs := strings.Split(connectionInfo, "|")
	for _, pair := range pairs {
		kv := strings.SplitN(pair, ":", 2)
		if len(kv) == 2 {
			switch kv[0] {
			case "jti":
				jti = kv[1]
			case "user":
				userID = kv[1]
			}
		}
	}

	if jti == "" || userID == "" {
		return fmt.Errorf("无法解析JTI或UserID")
	}

	// 立即撤销JWT
	jwtUtil := utils.NewJWT()
	err = jwtUtil.RevokeJWTByID(userID, jti)
	if err != nil {
		return fmt.Errorf("撤销JWT失败: %w", err)
	}

	global.GVA_LOG.Info("立即撤销旧JWT成功",
		zap.String("clientID", clientID),
		zap.String("userID", userID),
		zap.String("jti", jti))

	// 通过EMQX API强制断开客户端连接
	err = s.forceDisconnectClient(clientID)
	if err != nil {
		global.GVA_LOG.Error("强制断开EMQX客户端失败", zap.Error(err), zap.String("clientID", clientID))
		// 不返回错误，JWT撤销已成功，客户端在下次操作时会被拒绝
	} else {
		global.GVA_LOG.Info("强制断开EMQX客户端成功", zap.String("clientID", clientID))
	}

	return nil
}

// createAssignmentInfo 创建存储在 user:{userID}:roles 哈希中的值
func (s *RoleConflictService) createAssignmentInfo(clientID string, deviceInfo map[string]interface{}) string {
	deviceModel := "unknown"
	if dm, ok := deviceInfo["device_model"].(string); ok {
		deviceModel = dm
	}
	return fmt.Sprintf("client_id:%s|device:%s|connected_at:%d",
		clientID, deviceModel, time.Now().Unix())
}

// extractFromAssignment 从 'key:value|key2:value2' 格式的字符串中提取值
func (s *RoleConflictService) extractFromAssignment(assignment, key string) string {
	pairs := strings.Split(assignment, "|")
	for _, pair := range pairs {
		kv := strings.SplitN(pair, ":", 2)
		if len(kv) == 2 && kv[0] == key {
			return kv[1]
		}
	}
	return ""
}

// getLastActivity 获取客户端的最后活动时间
func (s *RoleConflictService) getLastActivity(clientID string) string {
	ctx := context.Background()
	connectionInfo, err := global.GVA_REDIS.HGet(ctx, "client_connections", clientID).Result()
	if err != nil {
		return "N/A"
	}
	lastPingStr := s.extractFromAssignment(connectionInfo, "last_ping")
	lastPing, _ := strconv.ParseInt(lastPingStr, 10, 64)
	return time.Unix(lastPing, 0).Format(time.RFC3339)
}

// forceDisconnectClient 通过EMQX API强制断开客户端连接
func (s *RoleConflictService) forceDisconnectClient(clientID string) error {
	// EMQX配置
	emqxHost := "49.235.40.39"
	emqxPort := "18083"
	emqxUser := "admin"
	emqxPass := "xuehua123"

	// 获取EMQX API Token
	loginURL := fmt.Sprintf("http://%s:%s/api/v5/login", emqxHost, emqxPort)
	loginPayload := map[string]string{
		"username": emqxUser,
		"password": emqxPass,
	}

	loginData, _ := json.Marshal(loginPayload)
	loginResp, err := http.Post(loginURL, "application/json", bytes.NewBuffer(loginData))
	if err != nil {
		return fmt.Errorf("EMQX登录请求失败: %w", err)
	}
	defer loginResp.Body.Close()

	if loginResp.StatusCode != 200 {
		return fmt.Errorf("EMQX登录失败: %d", loginResp.StatusCode)
	}

	var loginResult map[string]interface{}
	if err := json.NewDecoder(loginResp.Body).Decode(&loginResult); err != nil {
		return fmt.Errorf("解析EMQX登录响应失败: %w", err)
	}

	token, ok := loginResult["token"].(string)
	if !ok {
		return fmt.Errorf("无法获取EMQX API Token")
	}

	// 强制断开客户端
	disconnectURL := fmt.Sprintf("http://%s:%s/api/v5/clients/%s/kickout", emqxHost, emqxPort, clientID)
	req, err := http.NewRequest("DELETE", disconnectURL, nil)
	if err != nil {
		return fmt.Errorf("创建断开连接请求失败: %w", err)
	}

	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("断开连接请求失败: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 204 && resp.StatusCode != 404 {
		return fmt.Errorf("断开连接失败: %d", resp.StatusCode)
	}

	return nil
}
