package system

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
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
	global.GVA_LOG.Info("[AssignRole] 开始处理角色分配",
		zap.String("userID", userID),
		zap.String("role", role),
		zap.String("newClientID", clientID),
		zap.Bool("forceKick", forceKick))

	// 1. 如果需要，首先处理强制挤下线（在事务之外独立执行）
	if forceKick {
		global.GVA_LOG.Info("[AssignRole] 检测到forceKick=true，开始处理强制下线")
		if err := s.handleForceKick(ctx, userID, role, clientID); err != nil {
			global.GVA_LOG.Error("[AssignRole] 处理强制下线失败", zap.Error(err))
			return fmt.Errorf("处理强制挤下线失败: %w", err)
		}
		global.GVA_LOG.Info("[AssignRole] 处理强制下线成功")
	}

	// 2. 启动一个新的Redis事务来处理新角色的分配
	pipe := global.GVA_REDIS.TxPipeline()

	// 3. 创建并分配新角色信息
	assignmentInfo := s.createAssignmentInfo(clientID, deviceInfo)
	userRoleKey := fmt.Sprintf("user:%s:roles", userID)
	pipe.HSet(ctx, userRoleKey, role, assignmentInfo)

	// 4. 更新角色占用状态
	roleAssignmentKey := fmt.Sprintf("role_assignments:%s", role)
	roleAssignmentValue := map[string]interface{}{
		"current_user": userID,
		"client_id":    clientID,
		"assigned_at":  time.Now().Unix(),
	}
	pipe.HSet(ctx, roleAssignmentKey, roleAssignmentValue)

	// 5. 记录客户端连接状态 (增加jti用于后续的JWT吊销)
	now := time.Now().Unix()
	connectionInfo := fmt.Sprintf("user:%s|role:%s|connected_at:%d|last_ping:%d|jti:%s",
		userID, role, now, now, jti)
	pipe.HSet(ctx, "client_connections", clientID, connectionInfo)

	// 6. 为关键key设置过期时间，防止数据无限增长
	pipe.Expire(ctx, userRoleKey, 24*time.Hour)
	pipe.Expire(ctx, roleAssignmentKey, 24*time.Hour)

	// 7. 执行事务
	if _, err := pipe.Exec(ctx); err != nil {
		global.GVA_LOG.Error("[AssignRole] 分配角色事务执行失败", zap.Error(err))
		return fmt.Errorf("分配角色事务执行失败: %w", err)
	}

	global.GVA_LOG.Info("[AssignRole] 角色分配成功完成", zap.String("newClientID", clientID))
	return nil
}

// handleForceKick 处理强制挤下线（独立操作）
func (s *RoleConflictService) handleForceKick(ctx context.Context, userID, role, newClientID string) error {
	global.GVA_LOG.Info("[handleForceKick] 开始执行",
		zap.String("userID", userID),
		zap.String("role", role),
		zap.String("newClientID", newClientID))

	// 1. 获取被挤设备的信息（直接查询，不在事务中）
	userRoleKey := fmt.Sprintf("user:%s:roles", userID)
	currentAssignment, err := global.GVA_REDIS.HGet(ctx, userRoleKey, role).Result()
	if err != nil {
		if err == redis.Nil {
			global.GVA_LOG.Info("[handleForceKick] 角色未被分配，无需处理", zap.String("role", role))
			return nil // 角色未分配，无需处理
		}
		global.GVA_LOG.Error("[handleForceKick] 获取当前角色分配信息失败", zap.Error(err), zap.String("userRoleKey", userRoleKey))
		return fmt.Errorf("获取当前角色分配信息失败: %w", err)
	}
	global.GVA_LOG.Info("[handleForceKick] 获取到当前角色分配信息", zap.String("currentAssignment", currentAssignment))

	targetClientID := s.extractFromAssignment(currentAssignment, "client_id")
	if targetClientID == "" || targetClientID == newClientID {
		global.GVA_LOG.Info("[handleForceKick] 无需处理的客户端",
			zap.String("targetClientID", targetClientID),
			zap.String("newClientID", newClientID))
		return nil // 没有有效的旧客户端或与新客户端相同，无需处理
	}
	global.GVA_LOG.Info("[handleForceKick] 找到需要被挤下线的客户端", zap.String("targetClientID", targetClientID))

	// 2. 立即撤销旧JWT并断开连接
	global.GVA_LOG.Info("[handleForceKick] 准备调用revokeOldClientJWT", zap.String("targetClientID", targetClientID))
	if err := s.revokeOldClientJWT(ctx, targetClientID); err != nil {
		global.GVA_LOG.Error("[handleForceKick] revokeOldClientJWT失败", zap.Error(err), zap.String("targetClientID", targetClientID))
		return fmt.Errorf("强制挤下线失败: %w", err)
	}
	global.GVA_LOG.Info("[handleForceKick] revokeOldClientJWT成功", zap.String("targetClientID", targetClientID))

	// 3. 开启一个独立的事务来清理被挤下线客户端的数据
	global.GVA_LOG.Info("[handleForceKick] 准备开启事务清理旧客户端数据", zap.String("targetClientID", targetClientID))
	pipe := global.GVA_REDIS.TxPipeline()

	// 4. 创建挤下线通知
	kickNotification := map[string]interface{}{
		"target_client_id": targetClientID,
		"kicker_client_id": newClientID,
		"role":             role,
		"reason":           "role_taken_by_another_device",
		"timestamp":        time.Now().Unix(),
	}
	notificationJSON, _ := json.Marshal(kickNotification)
	pipe.LPush(ctx, "kick_notifications", string(notificationJSON))

	// 5. 清除被挤设备的状态
	pipe.HDel(ctx, "client_connections", targetClientID)
	// 同时从user:roles中也删除，确保状态一致
	pipe.HDel(ctx, userRoleKey, role)

	// 6. 执行清理事务
	if _, err := pipe.Exec(ctx); err != nil {
		global.GVA_LOG.Error("[handleForceKick] 清理被挤下线客户端状态事务失败", zap.Error(err))
		// 即使清理失败，挤下线操作本身已成功，可以选择只记录日志而不返回错误
	}

	global.GVA_LOG.Info("[handleForceKick] 成功完成",
		zap.String("target_client_id", targetClientID),
		zap.String("kicker_client_id", newClientID))

	return nil
}

// revokeOldClientJWT 立即撤销旧客户端的JWT并断开EMQX连接
func (s *RoleConflictService) revokeOldClientJWT(ctx context.Context, clientID string) error {
	global.GVA_LOG.Info("[revokeOldClientJWT] 开始执行", zap.String("clientID", clientID))

	// 从 client_connections 中获取JTI和UserID
	connectionInfoKey := "client_connections"
	connectionInfo, err := global.GVA_REDIS.HGet(ctx, connectionInfoKey, clientID).Result()
	if err != nil {
		global.GVA_LOG.Error("[revokeOldClientJWT] 从Redis获取连接信息失败",
			zap.String("key", connectionInfoKey),
			zap.String("clientID", clientID),
			zap.Error(err))
		return fmt.Errorf("获取连接信息失败: %w", err)
	}
	global.GVA_LOG.Info("[revokeOldClientJWT] 从Redis获取连接信息成功",
		zap.String("clientID", clientID),
		zap.String("connectionInfo", connectionInfo))

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

	global.GVA_LOG.Info("[revokeOldClientJWT] 解析出用户信息",
		zap.String("clientID", clientID),
		zap.String("parsedUserID", userID),
		zap.String("parsedJTI", jti))

	if jti == "" || userID == "" {
		err := fmt.Errorf("无法从'%s'解析出JTI或UserID", connectionInfo)
		global.GVA_LOG.Error("[revokeOldClientJWT] 解析JTI或UserID失败", zap.Error(err))
		return err
	}

	// 立即撤销MQTT JWT
	global.GVA_LOG.Info("[revokeOldClientJWT] 准备撤销MQTT JWT", zap.String("jti", jti), zap.String("userID", userID))
	jwtUtil := utils.NewJWT()
	err = jwtUtil.RevokeMQTTJWTByID(userID, jti)
	if err != nil {
		global.GVA_LOG.Error("[revokeOldClientJWT] 撤销MQTT JWT失败", zap.Error(err))
		return fmt.Errorf("撤销MQTT JWT失败: %w", err)
	}

	global.GVA_LOG.Info("[revokeOldClientJWT] 撤销MQTT JWT成功", zap.String("jti", jti))

	// 通过EMQX API强制断开客户端连接 - 修复：增加重试机制
	global.GVA_LOG.Info("[revokeOldClientJWT] 准备通过EMQX API强制断开客户端连接", zap.String("clientID", clientID))
	err = s.forceDisconnectClientWithRetry(clientID, 3)
	if err != nil {
		global.GVA_LOG.Error("[revokeOldClientJWT] 强制断开EMQX客户端失败", zap.Error(err), zap.String("clientID", clientID))
		return fmt.Errorf("强制断开EMQX客户端失败: %w", err)
	}

	global.GVA_LOG.Info("[revokeOldClientJWT] 强制断开EMQX客户端成功", zap.String("clientID", clientID))
	return nil
}

// forceDisconnectClientWithRetry 带重试机制的强制断开客户端
func (s *RoleConflictService) forceDisconnectClientWithRetry(clientID string, maxRetries int) error {
	for attempt := 1; attempt <= maxRetries; attempt++ {
		global.GVA_LOG.Info("[Retry] 尝试强制断开EMQX客户端",
			zap.String("clientID", clientID),
			zap.Int("attempt", attempt),
			zap.Int("maxRetries", maxRetries))

		err := s.forceDisconnectClient(clientID)
		if err == nil {
			// 成功后验证客户端是否真正断开
			if s.verifyClientDisconnected(clientID) {
				global.GVA_LOG.Info("[Retry] 强制断开并验证成功", zap.String("clientID", clientID))
				return nil
			}
			global.GVA_LOG.Warn("[Retry] API调用成功但客户端仍然连接，继续重试",
				zap.String("clientID", clientID),
				zap.Int("attempt", attempt))
		} else {
			global.GVA_LOG.Warn("[Retry] 强制断开客户端失败，准备重试",
				zap.String("clientID", clientID),
				zap.Int("attempt", attempt),
				zap.Error(err))
		}

		// 如果不是最后一次尝试，等待一段时间再重试
		if attempt < maxRetries {
			time.Sleep(time.Duration(attempt) * time.Second)
		}
	}

	return fmt.Errorf("经过 %d 次尝试后，仍无法断开客户端 %s", maxRetries, clientID)
}

// verifyClientDisconnected 验证客户端是否已断开连接
func (s *RoleConflictService) verifyClientDisconnected(clientID string) bool {
	// 1. 从全局配置中获取EMQX API配置
	cfg := global.GVA_CONFIG.MQTT
	apiCfg := cfg.API

	apiHost := apiCfg.Host
	if apiHost == "" {
		apiHost = cfg.Host
	}
	if apiHost == "" {
		global.GVA_LOG.Error("[verifyClientDisconnected] EMQX API主机地址未配置")
		return false
	}

	port := apiCfg.Port
	if port == 0 {
		port = 18083 // 默认EMQX API端口
	}

	apiURL := fmt.Sprintf("http://%s:%d/api/v5/clients/%s", apiHost, port, clientID)
	global.GVA_LOG.Info("[verifyClientDisconnected] 检查客户端连接状态", zap.String("url", apiURL))

	req, err := http.NewRequest("GET", apiURL, nil)
	if err != nil {
		global.GVA_LOG.Error("[verifyClientDisconnected] 创建HTTP请求失败", zap.Error(err))
		return false
	}

	client := &http.Client{Timeout: 5 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		global.GVA_LOG.Error("[verifyClientDisconnected] 查询客户端状态失败", zap.Error(err))
		return false
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusNotFound {
		global.GVA_LOG.Info("[verifyClientDisconnected] 客户端已断开 (API返回404)", zap.String("clientID", clientID))
		return true
	}

	global.GVA_LOG.Warn("[verifyClientDisconnected] 客户端似乎仍然连接",
		zap.String("clientID", clientID),
		zap.Int("statusCode", resp.StatusCode))
	return false
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
	// 1. 从全局配置中获取EMQX API配置
	cfg := global.GVA_CONFIG.MQTT
	apiCfg := cfg.API

	// 使用API专用配置，如果未配置则回退到MQTT配置
	apiHost := apiCfg.Host
	if apiHost == "" {
		apiHost = cfg.Host
	}
	if apiHost == "" {
		global.GVA_LOG.Error("[forceDisconnectClient] EMQX API主机地址未配置")
		return fmt.Errorf("EMQX API主机地址未配置")
	}

	port := apiCfg.Port
	if port == 0 {
		port = 18083 // 默认EMQX管理API端口
	}

	apiUsername := apiCfg.Username
	if apiUsername == "" {
		apiUsername = cfg.Username // 回退到MQTT用户名
	}

	apiPassword := apiCfg.Password
	if apiPassword == "" {
		apiPassword = cfg.Password // 回退到MQTT密码
	}

	// 增加调试日志
	global.GVA_LOG.Info("[forceDisconnectClient] 开始强制断开EMQX客户端",
		zap.String("clientID", clientID),
		zap.String("apiHost", apiHost),
		zap.Int("apiPort", port),
		zap.String("apiUsername", apiUsername),
		zap.Bool("useTLS", apiCfg.UseTLS))

	// 2. 构造登录请求
	protocol := "http"
	if apiCfg.UseTLS {
		protocol = "https"
	}
	loginURL := fmt.Sprintf("%s://%s:%d/api/v5/login", protocol, apiHost, port)

	loginPayload := map[string]string{
		"username": apiUsername,
		"password": apiPassword,
	}
	loginData, err := json.Marshal(loginPayload)
	if err != nil {
		return fmt.Errorf("序列化EMQX登录载荷失败: %w", err)
	}

	global.GVA_LOG.Info("[forceDisconnectClient] 发送EMQX API登录请求", zap.String("url", loginURL), zap.String("username", apiUsername))

	// 3. 发起登录请求获取Token
	client := &http.Client{Timeout: 5 * time.Second}
	loginResp, err := client.Post(loginURL, "application/json", bytes.NewBuffer(loginData))
	if err != nil {
		global.GVA_LOG.Error("[forceDisconnectClient] 请求EMQX API Token失败", zap.String("url", loginURL), zap.Error(err))
		return fmt.Errorf("请求EMQX API Token失败: %w", err)
	}
	defer loginResp.Body.Close()

	bodyBytes, _ := io.ReadAll(loginResp.Body)
	if loginResp.StatusCode != http.StatusOK {
		global.GVA_LOG.Error("[forceDisconnectClient] EMQX API登录认证失败",
			zap.String("url", loginURL),
			zap.Int("status", loginResp.StatusCode),
			zap.String("response", string(bodyBytes)))
		return fmt.Errorf("EMQX API登录认证失败: status=%d", loginResp.StatusCode)
	}

	var loginResult struct {
		Token string `json:"token"`
	}
	if err := json.Unmarshal(bodyBytes, &loginResult); err != nil {
		global.GVA_LOG.Error("[forceDisconnectClient] 解析EMQX API Token响应失败", zap.Error(err), zap.String("response", string(bodyBytes)))
		return fmt.Errorf("解析EMQX API Token响应失败: %w", err)
	}

	if loginResult.Token == "" {
		return fmt.Errorf("从EMQX API响应中未能获取Token")
	}

	global.GVA_LOG.Info("[forceDisconnectClient] EMQX API登录成功，获取到Token")

	// 4. 使用Token强制断开客户端
	disconnectURL := fmt.Sprintf("%s://%s:%d/api/v5/clients/%s", protocol, apiHost, port, clientID)
	req, err := http.NewRequest("DELETE", disconnectURL, nil)
	if err != nil {
		return fmt.Errorf("创建EMQX断开连接请求失败: %w", err)
	}

	req.Header.Set("Authorization", "Bearer "+loginResult.Token)

	global.GVA_LOG.Info("[forceDisconnectClient] 发送EMQX客户端断开请求", zap.String("url", disconnectURL))

	// 5. 发送断开请求
	disconnectResp, err := client.Do(req)
	if err != nil {
		global.GVA_LOG.Error("[forceDisconnectClient] 发送EMQX断开连接请求失败", zap.String("url", disconnectURL), zap.Error(err))
		return fmt.Errorf("发送EMQX断开连接请求失败: %w", err)
	}
	defer disconnectResp.Body.Close()

	// 6. 检查响应
	if disconnectResp.StatusCode == http.StatusOK || disconnectResp.StatusCode == http.StatusNoContent {
		global.GVA_LOG.Info("[forceDisconnectClient] 成功通过EMQX API请求断开客户端连接", zap.String("clientID", clientID))
		return nil
	}

	if disconnectResp.StatusCode == http.StatusNotFound {
		global.GVA_LOG.Warn("[forceDisconnectClient] 尝试断开一个不存在的EMQX客户端（可能已离线）", zap.String("clientID", clientID))
		return nil // 客户端已经离线，当作成功处理
	}

	disconnectBodyBytes, _ := io.ReadAll(disconnectResp.Body)
	global.GVA_LOG.Error("[forceDisconnectClient] EMQX API断开客户端操作失败",
		zap.String("url", disconnectURL),
		zap.Int("status", disconnectResp.StatusCode),
		zap.String("response", string(disconnectBodyBytes)))
	return fmt.Errorf("EMQX API返回错误: status=%d", disconnectResp.StatusCode)
}
