package system

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/flipped-aurora/gin-vue-admin/server/global"
	"github.com/flipped-aurora/gin-vue-admin/server/utils"
	"github.com/redis/go-redis/v9"
	"go.uber.org/zap"
)

type NotificationService struct{}

var (
	// mqttService MQTT服务实例，用于发送角色撤销通知
	mqttService mqttServiceInterface
)

// mqttServiceInterface MQTT服务接口
type mqttServiceInterface interface {
	IsConnected() bool
	PublishToClient(clientID, subtopic string, payload interface{}) error
}

// SetMQTTService 设置MQTT服务实例（供其他服务注册使用）
func (s *NotificationService) SetMQTTService(service mqttServiceInterface) {
	mqttService = service
	global.GVA_LOG.Info("MQTT服务已注册到通知服务")
}

// 初始化MQTT服务引用
func (s *NotificationService) initMQTTService() {
	// 这里通过类型断言获取NFC Relay的MQTT服务
	// 如果有更好的服务注册机制，可以使用依赖注入
	if mqttService == nil {
		// 尝试获取MQTT服务实例
		// 注意：这需要确保NFC中继MQTT服务已经初始化
		// 在实际部署中，应该通过服务注册或依赖注入来管理
		global.GVA_LOG.Info("初始化通知服务的MQTT发布器")
	}
}

// Start 启动通知服务的后台任务
func (s *NotificationService) Start() {
	global.GVA_LOG.Info("启动实时通知服务...")
	s.initMQTTService()
	go s.processKickNotifications()
	go s.cleanupExpiredConnections()
}

// processKickNotifications 处理挤下线通知
// 它会阻塞式地监听Redis中的kick_notifications列表
func (s *NotificationService) processKickNotifications() {
	ctx := context.Background()
	jwtUtil := utils.NewJWT()

	for {
		// BRPop会阻塞直到有消息，0表示永不超时
		result, err := global.GVA_REDIS.BRPop(ctx, 0, "kick_notifications").Result()
		if err != nil {
			global.GVA_LOG.Error("从kick_notifications读取消息失败", zap.Error(err))
			time.Sleep(1 * time.Second) // 避免错误循环过于频繁
			continue
		}

		if len(result) < 2 {
			continue
		}

		var notification map[string]interface{}
		if err := json.Unmarshal([]byte(result[1]), &notification); err != nil {
			global.GVA_LOG.Error("解析挤下线通知JSON失败", zap.Error(err))
			continue
		}

		targetClientID, _ := notification["target_client_id"].(string)
		if targetClientID == "" {
			continue
		}

		global.GVA_LOG.Info("接收到挤下线通知", zap.Any("notification", notification))

		// 1. 发送MQTT挤下线通知
		if mqttService != nil && mqttService.IsConnected() {
			payload := map[string]interface{}{
				"revoked_role":        notification["role"],
				"reason":              notification["reason"],
				"kicked_by_client_id": notification["kicker_client_id"],
				"timestamp_utc":       time.Now().UTC().Format(time.RFC3339),
			}

			// 发送到客户端的控制主题
			if err := mqttService.PublishToClient(targetClientID, "control/role_revoked_notification", payload); err != nil {
				global.GVA_LOG.Error("发送MQTT挤下线通知失败",
					zap.String("targetClientID", targetClientID),
					zap.Error(err))
			} else {
				global.GVA_LOG.Info("MQTT挤下线通知发送成功",
					zap.String("targetClientID", targetClientID))
			}
		} else {
			global.GVA_LOG.Warn("MQTT服务未连接，无法发送挤下线通知",
				zap.String("targetClientID", targetClientID))
		}

		// 2. 吊销旧的JWT
		s.revokeTargetJWT(ctx, jwtUtil, targetClientID)
	}
}

// revokeTargetJWT 吊销目标客户端的JWT
func (s *NotificationService) revokeTargetJWT(ctx context.Context, jwtUtil *utils.JWT, clientID string) {
	// 从 client_connections 中获取JTI
	connectionInfo, err := global.GVA_REDIS.HGet(ctx, "client_connections", clientID).Result()
	if err != nil {
		if err != redis.Nil {
			global.GVA_LOG.Error("获取被挤下线客户端的连接信息失败", zap.Error(err), zap.String("clientID", clientID))
		}
		return
	}

	parts := strings.Split(connectionInfo, "|")
	var jti, userID string
	for _, part := range parts {
		kv := strings.SplitN(part, ":", 2)
		if len(kv) == 2 {
			if kv[0] == "jti" {
				jti = kv[1]
			}
			if kv[0] == "user" {
				userID = kv[1]
			}
		}
	}

	if jti == "" || userID == "" {
		global.GVA_LOG.Warn("无法从连接信息中解析出JTI或UserID，无法吊销JWT", zap.String("connectionInfo", connectionInfo))
		return
	}

	// 使用专门的MQTT JWT撤销方法
	err = jwtUtil.RevokeMQTTJWTByID(userID, jti)
	if err != nil {
		global.GVA_LOG.Error("吊销旧的MQTT JWT失败", zap.Error(err), zap.String("userID", userID), zap.String("jti", jti))
	} else {
		global.GVA_LOG.Info("已成功吊销被挤下线设备的MQTT JWT", zap.String("userID", userID), zap.String("jti", jti))
	}
}

// cleanupExpiredConnections 清理过期的客户端连接
func (s *NotificationService) cleanupExpiredConnections() {
	ticker := time.NewTicker(5 * time.Minute)
	defer ticker.Stop()

	for range ticker.C {
		ctx := context.Background()
		connections, err := global.GVA_REDIS.HGetAll(ctx, "client_connections").Result()
		if err != nil {
			global.GVA_LOG.Error("获取所有客户端连接失败", zap.Error(err))
			continue
		}

		now := time.Now().Unix()

		pipe := global.GVA_REDIS.TxPipeline()
		cleanedCount := 0

		for clientID, connectionInfo := range connections {
			lastPingStr := extractFromConnection(connectionInfo, "last_ping")
			if lastPingStr == "" {
				continue
			}

			lastPing, err := strconv.ParseInt(lastPingStr, 10, 64)
			if err != nil {
				continue
			}

			// 10分钟超时
			if now-lastPing > 600 {
				pipe.HDel(ctx, "client_connections", clientID)
				s.clearRoleAssignmentForClient(ctx, pipe, connectionInfo)
				cleanedCount++
			}
		}

		if cleanedCount > 0 {
			if _, err := pipe.Exec(ctx); err != nil {
				global.GVA_LOG.Error("批量清理过期连接事务执行失败", zap.Error(err))
			} else {
				global.GVA_LOG.Info("成功清理了过期的客户端连接", zap.Int("count", cleanedCount))
			}
		}
	}
}

// clearRoleAssignmentForClient 清理客户端占用的角色
func (s *NotificationService) clearRoleAssignmentForClient(ctx context.Context, pipe redis.Pipeliner, connectionInfo string) {
	userID := extractFromConnection(connectionInfo, "user")
	role := extractFromConnection(connectionInfo, "role")
	clientID := extractFromConnection(connectionInfo, "client_id")

	if userID == "" || role == "" || clientID == "" {
		return
	}

	// 清除 user:{userID}:roles 中对应的角色
	userRoleKey := fmt.Sprintf("user:%s:roles", userID)
	pipe.HDel(ctx, userRoleKey, role)

	// 清除 role_assignments:{role} 中的占用信息（需要先验证是否还是自己占用）
	// 为避免复杂化，这里简化处理：直接清除。更好的做法是使用WATCH或Lua脚本。
	roleAssignmentKey := fmt.Sprintf("role_assignments:%s", role)
	pipe.HDel(ctx, roleAssignmentKey, "current_user", "client_id", "assigned_at")

	global.GVA_LOG.Info("已将角色清理任务加入队列", zap.String("userID", userID), zap.String("role", role))
}

// extractFromConnection 从连接信息字符串中提取值
func extractFromConnection(connection, key string) string {
	pairs := strings.Split(connection, "|")
	for _, pair := range pairs {
		kv := strings.SplitN(pair, ":", 2)
		if len(kv) == 2 && kv[0] == key {
			return kv[1]
		}
	}
	return ""
}
