package system

import (
	"net/http"
	"time"

	"github.com/flipped-aurora/gin-vue-admin/server/global"
	"github.com/flipped-aurora/gin-vue-admin/server/model/common/response"
	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

type MqttWebhookApi struct{}

// 角色请求Hook请求结构体
type RoleRequestHookRequest struct {
	ClientID  string             `json:"clientid"`
	Username  string             `json:"username"`
	Topic     string             `json:"topic"`
	Payload   RoleRequestPayload `json:"payload"`
	Timestamp int64              `json:"timestamp"`
}

type RoleRequestPayload struct {
	Role       string                 `json:"role"`
	ForceKick  bool                   `json:"force_kick"`
	DeviceInfo map[string]interface{} `json:"device_info"`
}

// 角色请求Hook响应结构体
type RoleRequestHookResponse struct {
	Code int                 `json:"code"`
	Msg  string              `json:"msg"`
	Data RoleRequestHookData `json:"data"`
}

type RoleRequestHookData struct {
	Action           string                `json:"action"` // role_conflict_detected|role_assigned|role_denied
	ConflictDevice   *ConflictDeviceInfo   `json:"conflict_device,omitempty"`
	KickNotification *KickNotificationInfo `json:"kick_notification,omitempty"`
}

type ConflictDeviceInfo struct {
	ClientID    string `json:"client_id"`
	DeviceModel string `json:"device_model"`
	ConnectedAt string `json:"connected_at"`
}

type KickNotificationInfo struct {
	TargetClientID string `json:"target_client_id"`
	Message        string `json:"message"`
}

// 连接状态Hook请求结构体
type ConnectionStatusHookRequest struct {
	EventType      string `json:"event_type"` // client_connected|client_disconnected
	ClientID       string `json:"clientid"`
	Username       string `json:"username"`
	ConnectedAt    string `json:"connected_at,omitempty"`
	DisconnectedAt string `json:"disconnected_at,omitempty"`
	Reason         string `json:"reason,omitempty"`
}

// RoleRequestHook 角色请求处理Hook
// @Tags      MqttWebhook
// @Summary   处理MQTT客户端角色请求
// @Accept    application/json
// @Produce   application/json
// @Param     data  body      RoleRequestHookRequest true "角色请求Hook数据"
// @Success   200   {object}  RoleRequestHookResponse  "处理成功"
// @Router    /mqtt/hooks/role_request [post]
func (a *MqttWebhookApi) RoleRequestHook(c *gin.Context) {
	var req RoleRequestHookRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		global.GVA_LOG.Error("角色请求Hook参数解析失败", zap.Error(err))
		c.JSON(http.StatusOK, RoleRequestHookResponse{
			Code: 1,
			Msg:  "参数解析失败",
			Data: RoleRequestHookData{Action: "role_denied"},
		})
		return
	}

	global.GVA_LOG.Info("收到角色请求Hook",
		zap.String("clientid", req.ClientID),
		zap.String("role", req.Payload.Role),
		zap.Bool("force_kick", req.Payload.ForceKick))

	// 1. 从clientID中提取用户ID (格式: username-role-timestamp)
	userID := extractUserIDFromClientID(req.ClientID)
	if userID == "" {
		global.GVA_LOG.Error("无法从ClientID提取用户ID", zap.String("clientid", req.ClientID))
		c.JSON(http.StatusOK, RoleRequestHookResponse{
			Code: 1,
			Msg:  "ClientID格式无效",
			Data: RoleRequestHookData{Action: "role_denied"},
		})
		return
	}

	// 2. 检查角色冲突
	conflictResult, err := roleConflictService.CheckRoleConflict(userID, req.Payload.Role, req.ClientID)
	if err != nil {
		global.GVA_LOG.Error("检查角色冲突失败", zap.Error(err))
		c.JSON(http.StatusOK, RoleRequestHookResponse{
			Code: 1,
			Msg:  "角色冲突检查失败",
			Data: RoleRequestHookData{Action: "role_denied"},
		})
		return
	}

	// 3. 如果有冲突且不强制踢下线，返回冲突信息
	if conflictResult.HasConflict && !req.Payload.ForceKick {
		c.JSON(http.StatusOK, RoleRequestHookResponse{
			Code: 0,
			Msg:  "检测到角色冲突",
			Data: RoleRequestHookData{
				Action: "role_conflict_detected",
				ConflictDevice: &ConflictDeviceInfo{
					ClientID:    conflictResult.ConflictDevice.ClientID,
					DeviceModel: conflictResult.ConflictDevice.DeviceModel,
					ConnectedAt: conflictResult.ConflictDevice.ConnectedAt,
				},
			},
		})
		return
	}

	// 4. 分配角色 (包含强制踢下线逻辑)
	// 生成一个临时JTI，实际的JWT Token生成应该在客户端通过正常API获取
	tempJTI := generateTempJTI(req.ClientID)
	err = roleConflictService.AssignRole(userID, req.Payload.Role, req.ClientID, tempJTI, req.Payload.DeviceInfo, req.Payload.ForceKick)
	if err != nil {
		global.GVA_LOG.Error("角色分配失败", zap.Error(err))
		c.JSON(http.StatusOK, RoleRequestHookResponse{
			Code: 1,
			Msg:  "角色分配失败",
			Data: RoleRequestHookData{Action: "role_denied"},
		})
		return
	}

	// 5. 如果是强制踢下线，返回踢下线通知信息
	responseData := RoleRequestHookData{Action: "role_assigned"}
	if conflictResult.HasConflict && req.Payload.ForceKick {
		responseData.KickNotification = &KickNotificationInfo{
			TargetClientID: conflictResult.ConflictDevice.ClientID,
			Message:        "您的角色已被其他设备获取",
		}
	}

	c.JSON(http.StatusOK, RoleRequestHookResponse{
		Code: 0,
		Msg:  "角色分配成功",
		Data: responseData,
	})
}

// ConnectionStatusHook 连接状态跟踪Hook
// @Tags      MqttWebhook
// @Summary   处理MQTT客户端连接状态变化
// @Accept    application/json
// @Produce   application/json
// @Param     data  body      ConnectionStatusHookRequest true "连接状态Hook数据"
// @Success   200   {object}  response.Response  "处理成功"
// @Router    /mqtt/hooks/connection_status [post]
func (a *MqttWebhookApi) ConnectionStatusHook(c *gin.Context) {
	var req ConnectionStatusHookRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		global.GVA_LOG.Error("连接状态Hook参数解析失败", zap.Error(err))
		response.FailWithMessage("参数解析失败", c)
		return
	}

	global.GVA_LOG.Info("收到连接状态Hook",
		zap.String("event_type", req.EventType),
		zap.String("clientid", req.ClientID),
		zap.String("username", req.Username))

	// 处理连接状态变化
	switch req.EventType {
	case "client_connected":
		err := a.handleClientConnected(req)
		if err != nil {
			global.GVA_LOG.Error("处理客户端连接失败", zap.Error(err))
			response.FailWithMessage("处理连接状态失败", c)
			return
		}
	case "client_disconnected":
		err := a.handleClientDisconnected(req)
		if err != nil {
			global.GVA_LOG.Error("处理客户端断开失败", zap.Error(err))
			response.FailWithMessage("处理断开状态失败", c)
			return
		}
	default:
		global.GVA_LOG.Warn("未知的连接事件类型", zap.String("event_type", req.EventType))
	}

	response.OkWithMessage("状态更新成功", c)
}

// handleClientConnected 处理客户端连接
func (a *MqttWebhookApi) handleClientConnected(req ConnectionStatusHookRequest) error {
	// 更新客户端连接状态
	// 这里可以更新Redis中的连接信息，记录连接时间等
	global.GVA_LOG.Info("客户端连接",
		zap.String("clientid", req.ClientID),
		zap.String("connected_at", req.ConnectedAt))

	// TODO: 可以在这里触发其他业务逻辑，如通知其他客户端
	return nil
}

// handleClientDisconnected 处理客户端断开
func (a *MqttWebhookApi) handleClientDisconnected(req ConnectionStatusHookRequest) error {
	// 清理客户端状态
	global.GVA_LOG.Info("客户端断开",
		zap.String("clientid", req.ClientID),
		zap.String("disconnected_at", req.DisconnectedAt),
		zap.String("reason", req.Reason))

	// TODO: 可以在这里清理角色分配、通知其他客户端等
	return nil
}

// 辅助函数：从ClientID提取用户ID
func extractUserIDFromClientID(clientID string) string {
	// ClientID格式通常是: username-role-timestamp 或类似格式
	// 这里需要根据实际的ClientID生成规则来解析
	// 暂时简化处理，假设clientID就是username或者包含username
	if len(clientID) > 0 {
		// 简化处理：假设clientID的前缀就是用户名
		// 实际实现时需要根据具体的ClientID生成规则来解析
		return clientID // 或者进行更复杂的解析
	}
	return ""
}

// 辅助函数：生成临时JTI
func generateTempJTI(clientID string) string {
	return "temp_" + clientID + "_" + string(rune(time.Now().Unix()))
}
