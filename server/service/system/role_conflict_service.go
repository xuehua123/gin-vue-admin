package system

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/flipped-aurora/gin-vue-admin/server/global"
	"github.com/flipped-aurora/gin-vue-admin/server/model/system/request"
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
	// 注意：这里不能调用CheckRoleConflict，因为它会发起新的Redis查询，破坏事务的原子性
	// 我们需要直接从Redis中获取旧的ClientID
	userRoleKey := fmt.Sprintf("user:%s:roles", userID)
	currentAssignment, err := global.GVA_REDIS.HGet(ctx, userRoleKey, role).Result() // 在事务外独立查询
	if err != nil || currentAssignment == "" {
		// 如果查询失败或角色未分配，则无需处理
		return nil
	}

	targetClientID := s.extractFromAssignment(currentAssignment, "client_id")
	if targetClientID == "" || targetClientID == newClientID {
		// 没有有效的旧客户端或与新客户端相同，无需处理
		return nil
	}

	// 2. 创建挤下线通知
	kickNotification := map[string]interface{}{
		"target_client_id": targetClientID,
		"kicker_client_id": newClientID,
		"role":             role,
		"reason":           "role_taken_by_another_device",
		"timestamp":        time.Now().Unix(),
	}
	notificationJSON, _ := json.Marshal(kickNotification)
	pipe.LPush(ctx, "kick_notifications", string(notificationJSON))

	// 3. 清除被挤设备的状态
	pipe.HDel(ctx, "client_connections", targetClientID)

	// **重要**: 我们不在这里清除 user:{userID}:roles, 因为后续的 HSet 会自动覆盖它。
	// 也不在这里吊销JWT，这个任务由NotificationService完成。

	global.GVA_LOG.Info("已将挤下线任务加入队列",
		zap.String("target_client_id", targetClientID),
		zap.String("kicker_client_id", newClientID))

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
