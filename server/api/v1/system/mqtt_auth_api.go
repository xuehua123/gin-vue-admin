package system

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"

	"github.com/flipped-aurora/gin-vue-admin/server/global"
	"github.com/flipped-aurora/gin-vue-admin/server/model/common"
	"github.com/flipped-aurora/gin-vue-admin/server/model/system/request"
	"github.com/flipped-aurora/gin-vue-admin/server/utils"
	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	"github.com/redis/go-redis/v9"
	"go.uber.org/zap"
)

type MqttAuthApi struct{}

// Authenticate MQTT客户端认证
// Webhook for EMQX: /mqtt/auth
func (a *MqttAuthApi) Authenticate(c *gin.Context) {
	var req request.MqttAuthRequest

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusOK, gin.H{"result": "deny"})
		return
	}

	// 根据标准MQTT认证实践，从Password字段获取JWT Token。
	// Flutter客户端将JWT作为MQTT密码发送，EMQX将其传递到password字段。
	tokenString := req.Password
	if tokenString == "" {
		global.GVA_LOG.Warn("MQTT认证失败: 请求中的password字段为空，该字段应用来传递JWT", zap.String("clientID", req.ClientID), zap.String("username", req.Username))
		c.JSON(http.StatusOK, gin.H{"result": "deny"})
		return
	}

	// 验证JWT Token
	jwtUtil := utils.NewJWT()
	claims, err := jwtUtil.ParseMQTTToken(tokenString)
	if err != nil {
		global.GVA_LOG.Warn("MQTT认证失败: Token解析或验证失败", zap.Error(err), zap.String("clientID", req.ClientID))
		c.JSON(http.StatusOK, gin.H{"result": "deny"})
		return
	}

	// 验证ClientID是否与Token中的信息匹配
	if claims.ClientID != req.ClientID {
		global.GVA_LOG.Warn("MQTT认证失败: ClientID不匹配",
			zap.String("tokenClientID", claims.ClientID), zap.String("reqClientID", req.ClientID),
			zap.String("tokenUsername", claims.Username),
			zap.String("reqUsername", req.Username))
		c.JSON(http.StatusOK, gin.H{"result": "deny"})
		return
	}

	// 方案核心：认证成功后，为该客户端"盖上"一个包含角色信息的、有时效的信任凭证
	redisKey := common.MqttAuthSuccessKeyPrefix + req.ClientID
	authCache := request.MqttAuthCache{
		Role:     claims.Role,
		Username: claims.Username,
	}
	payload, err := json.Marshal(authCache)
	if err != nil {
		// JSON序列化失败是一个严重问题，但为了服务可用性，我们仍然允许认证，并记录严重错误
		global.GVA_LOG.Error("MQTT认证信任凭证序列化失败(服务降级)", zap.Error(err), zap.String("clientID", req.ClientID))
	} else {
		if err := global.GVA_REDIS.Set(c, redisKey, payload, common.MqttAuthSuccessKeyTTL).Err(); err != nil {
			// Redis写入失败是一个非关键错误，我们记录日志但仍然允许认证
			global.GVA_LOG.Error("MQTT认证信任凭证写入Redis失败(服务降级)", zap.Error(err), zap.String("clientID", req.ClientID))
		}
	}

	// 注意：在MQTT场景中，客户端通常使用clientID作为MQTT username，
	// 而JWT中的username是真实用户名，两者不需要匹配。
	// 因此移除了严格的username匹配检查，以符合标准MQTT认证实践。

	// TODO: 后续可在此处更新客户端的连接状态，例如记录last_ping

	global.GVA_LOG.Info("MQTT客户端认证成功，信任凭证已签发",
		zap.String("clientID", req.ClientID),
		zap.String("username", claims.Username),
	)

	c.JSON(http.StatusOK, gin.H{
		"result":       "allow",
		"is_superuser": false,
	})
}

// CheckACL MQTT权限控制
// Webhook for EMQX: /mqtt/acl
func (m *MqttAuthApi) CheckACL(c *gin.Context) {
	var req request.MqttAclRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		global.GVA_LOG.Warn("MQTT ACL检查失败: 无效的请求格式", zap.Error(err))
		c.JSON(http.StatusOK, gin.H{"result": "deny"})
		return
	}

	// 方案核心：通过Redis"验章"，检查客户端是否在近期通过了认证
	redisKey := common.MqttAuthSuccessKeyPrefix + req.ClientID
	payload, err := global.GVA_REDIS.Get(c, redisKey).Result()
	if err != nil {
		if err == redis.Nil {
			global.GVA_LOG.Warn("MQTT ACL检查拒绝: 客户端缺少信任凭证(未认证或凭证过期)", zap.String("clientID", req.ClientID), zap.String("topic", req.Topic))
		} else {
			global.GVA_LOG.Error("MQTT ACL检查失败: 查询Redis信任凭证时出错", zap.Error(err), zap.String("clientID", req.ClientID))
		}
		c.JSON(http.StatusOK, gin.H{"result": "deny"})
		return
	}

	var authCache request.MqttAuthCache
	if err := json.Unmarshal([]byte(payload), &authCache); err != nil {
		global.GVA_LOG.Error("MQTT ACL检查失败: 信任凭证解析失败(JSON无效)", zap.Error(err), zap.String("clientID", req.ClientID))
		c.JSON(http.StatusOK, gin.H{"result": "deny"})
		return
	}

	// "验章"成功，我们现在信任这个客户端，并使用缓存的角色信息进行ACL规则判断
	claims := &request.MQTTClaims{
		RegisteredClaims: jwt.RegisteredClaims{}, // 这里不需要指针
		Role:             authCache.Role,
		Username:         authCache.Username,
		ClientID:         req.ClientID,
	}

	if m.authorizeMqttAction(claims, req.Topic, req.Action) {
		global.GVA_LOG.Debug("MQTT ACL授权通过",
			zap.String("clientID", claims.ClientID),
			zap.String("action", req.Action),
			zap.String("topic", req.Topic))
		c.JSON(http.StatusOK, gin.H{"result": "allow"})
	} else {
		global.GVA_LOG.Warn("MQTT ACL拒绝: 权限不足",
			zap.String("clientID", claims.ClientID),
			zap.String("role", claims.Role),
			zap.String("action", req.Action),
			zap.String("topic", req.Topic))
		c.JSON(http.StatusOK, gin.H{"result": "deny"})
	}
}

// authorizeMqttAction 根据V1.2最终方案，统一检查MQTT操作权限
func (m *MqttAuthApi) authorizeMqttAction(claims *request.MQTTClaims, topic, action string) bool {
	myRole := claims.Role
	myClientID := claims.ClientID

	// 规则1: 全局主题 (服务器广播)
	if topic == "nfc_relay/server/status" {
		return action == "subscribe" // 只允许订阅
	}

	// 规则2: 会话数据主题 (APDU) - 保持兼容
	if strings.HasPrefix(topic, "nfc_relay/sessions/") {
		parts := strings.Split(topic, "/")
		if len(parts) == 5 && parts[1] == "sessions" {
			topicClientID := parts[2]
			topicRole := parts[3]
			subTopic := parts[4]
			// 客户端只能订阅/发布到自己的会话主题
			if topicClientID == myClientID && topicRole == myRole && (subTopic == "apdu" || subTopic == "events") {
				return true
			}
		}
	}

	// 规则3: 客户端专属命名空间 (本次重构的核心)
	// 格式: nfc_relay/clients/{clientID}/(state|events|commands)
	const clientPrefix = "nfc_relay/clients/"
	if strings.HasPrefix(topic, clientPrefix) {
		trimmedTopic := strings.TrimPrefix(topic, clientPrefix)
		parts := strings.Split(trimmedTopic, "/")

		if len(parts) >= 1 { // 至少要有 clientID 部分
			topicClientID := parts[0]

			// 场景 A: 客户端操作自己的资源 (e.g., 发布/订阅自己的状态)
			if topicClientID == myClientID {
				if len(parts) == 2 {
					subTopic := parts[1]
					// 对自己的 state, events, commands, status 主题拥有完全的读写权限
					if subTopic == "state" || subTopic == "events" || subTopic == "commands" || subTopic == "status" {
						return true
					}
				}
			} else {
				// 场景 B: 客户端操作他人的资源 (e.g., 订阅对端的状态)
				// 对于他人的资源，只拥有只读权限
				if action == "subscribe" {
					if len(parts) == 2 {
						subTopic := parts[1]
						// 只允许订阅他人的 state 和 events 主题
						if subTopic == "state" || subTopic == "events" {
							return true
						}
					}
				}
			}
		}
	}

	// 规则4: 配对业务主题 (新增)
	// 格式: nfc_relay/pairing/...
	const pairingPrefix = "nfc_relay/pairing/"
	if strings.HasPrefix(topic, pairingPrefix) {
		return m.checkPairingTopicPermissions(myClientID, topic, action)
	}

	// 规则5: admin角色的特殊权限 (原规则4)
	if myRole == "admin" {
		// Admin可以订阅所有nfc_relay相关的主题
		if action == "subscribe" && (strings.HasPrefix(topic, "nfc_relay/")) {
			return true
		}
		// Admin可以发布到系统管理主题
		if action == "publish" && strings.HasPrefix(topic, "nfc_relay/admin/") {
			return true
		}
	}

	// 默认拒绝所有其他情况
	return false
}

// checkPairingTopicPermissions 检查配对相关主题的权限 (企业级新增辅助函数)
func (m *MqttAuthApi) checkPairingTopicPermissions(clientID, topic, action string) bool {
	// 【方案一：兼容现有架构】配对通知主题: nfc_relay/pairing/notifications/{clientID}
	const notificationsPrefix = "nfc_relay/pairing/notifications/"
	if strings.HasPrefix(topic, notificationsPrefix) {
		if action == "subscribe" {
			// 客户端只能订阅自己的个人通知主题
			targetClientID := strings.TrimPrefix(topic, notificationsPrefix)
			return targetClientID == clientID
		}
		// 出于安全考虑，客户端不允许直接发布到通知主题
		return false
	}

	// 【方案二：新增用户级通知主题】nfc_relay/user/{username}/notifications
	// 这允许同一用户的不同ClientID都能接收到配对通知
	const userNotificationsPrefix = "nfc_relay/user/"
	if strings.HasPrefix(topic, userNotificationsPrefix) {
		trimmedTopic := strings.TrimPrefix(topic, userNotificationsPrefix)
		parts := strings.Split(trimmedTopic, "/")

		if len(parts) == 2 && parts[1] == "notifications" {
			topicUsername := parts[0]
			// 从ClientID中提取用户名进行比较
			if clientUsername := m.extractUsernameFromClientID(clientID); clientUsername != "" {
				if action == "subscribe" && topicUsername == clientUsername {
					return true
				}
			}
		}
		// 不允许发布到用户通知主题
		return false
	}

	// 配对状态更新主题: nfc_relay/pairing/status_updates/{clientID}
	const statusUpdatesPrefix = "nfc_relay/pairing/status_updates/"
	if strings.HasPrefix(topic, statusUpdatesPrefix) {
		if action == "subscribe" {
			// 客户端只能订阅自己的个人状态更新主题
			targetClientID := strings.TrimPrefix(topic, statusUpdatesPrefix)
			return targetClientID == clientID
		}
		// 客户端不允许直接发布到状态更新主题
		return false
	}

	// 会话数据主题: nfc_relay/session/{sessionID}/data
	// 支持会话级的数据交换
	const sessionPrefix = "nfc_relay/session/"
	if strings.HasPrefix(topic, sessionPrefix) {
		return m.checkSessionTopicPermissions(clientID, topic, action)
	}

	// 全局配对事件主题: nfc_relay/pairing/events
	const eventsTopic = "nfc_relay/pairing/events"
	if topic == eventsTopic {
		// 客户端只能订阅全局事件，不能发布
		return action == "subscribe"
	}

	// 默认情况下，拒绝所有其他未明确定义的配对子主题
	return false
}

// extractUsernameFromClientID 从ClientID中提取用户名
// ClientID格式: {username}-{role}-{sequence}
func (m *MqttAuthApi) extractUsernameFromClientID(clientID string) string {
	if clientID == "" {
		return ""
	}

	// 查找第一个'-'的位置
	firstDash := strings.Index(clientID, "-")
	if firstDash == -1 {
		return ""
	}

	// 查找第二个'-'的位置
	secondDash := strings.Index(clientID[firstDash+1:], "-")
	if secondDash == -1 {
		return ""
	}

	// 提取用户名部分
	username := clientID[:firstDash]
	return username
}

// checkSessionTopicPermissions 检查会话主题权限
// 支持基于会话ID的动态权限控制
func (m *MqttAuthApi) checkSessionTopicPermissions(clientID, topic, action string) bool {
	// 格式: nfc_relay/session/{sessionID}/data
	const sessionPrefix = "nfc_relay/session/"
	trimmedTopic := strings.TrimPrefix(topic, sessionPrefix)
	parts := strings.Split(trimmedTopic, "/")

	if len(parts) != 2 || parts[1] != "data" {
		return false
	}

	sessionID := parts[0]
	if sessionID == "" {
		return false
	}

	// 通过Redis查询会话参与者
	// 格式: session:{sessionID} -> "clientA,clientB"
	ctx := context.Background()
	sessionKey := fmt.Sprintf("pairing_session:%s", sessionID)
	participants, err := global.GVA_REDIS.Get(ctx, sessionKey).Result()
	if err != nil {
		global.GVA_LOG.Debug("会话主题权限检查：会话不存在或已过期",
			zap.String("sessionID", sessionID),
			zap.String("clientID", clientID),
			zap.Error(err))
		return false
	}

	// 检查当前ClientID是否在参与者列表中
	participantList := strings.Split(participants, ",")
	for _, participant := range participantList {
		if strings.TrimSpace(participant) == clientID {
			global.GVA_LOG.Debug("会话主题权限检查通过",
				zap.String("sessionID", sessionID),
				zap.String("clientID", clientID),
				zap.String("action", action))
			return true
		}
	}

	global.GVA_LOG.Debug("会话主题权限检查失败：客户端不在参与者列表中",
		zap.String("sessionID", sessionID),
		zap.String("clientID", clientID),
		zap.Strings("participants", participantList))
	return false
}

// WebHookEvent 处理EMQX连接状态WebHook事件
// @Tags MQTT
// @Summary EMQX WebHook事件处理器
// @Description 处理客户端连接/断开事件，维护在线状态
// @Security ApiKeyAuth
// @accept application/json
// @Produce application/json
// @Param data body request.EmqxWebhookRequest true "WebHook事件数据"
// @Success 200 {object} response.Response{msg=string} "事件已接收"
// @Failure 400 {object} response.Response{msg=string} "请求参数错误或无效的事件格式"
// @Router /mqtt/webhook [post]
func (a *MqttAuthApi) WebHookEvent(c *gin.Context) {
	// 增加原始body读取，用于错误日志记录和数据清洗
	bodyBytes, err := io.ReadAll(c.Request.Body)
	if err != nil {
		global.GVA_LOG.Error("EMQX WebHook读取请求体失败", zap.Error(err), zap.String("ip", c.ClientIP()))
		c.JSON(http.StatusInternalServerError, gin.H{"success": false, "message": "读取请求体失败"})
		return
	}
	defer c.Request.Body.Close()

	// 数据清洗：将 EMQX 可能产生的非法 "undefined" 替换为合法的 "null"
	// 这是一个防御性措施，增强了系统对上游不规范数据的容忍度
	cleanedBody := bytes.ReplaceAll(bodyBytes, []byte("undefined"), []byte("null"))

	var req request.EmqxWebhookRequest
	if err := json.Unmarshal(cleanedBody, &req); err != nil {
		global.GVA_LOG.Error("EMQX WebHook请求解析失败: 无法解析为JSON",
			zap.Error(err),
			zap.String("ip", c.ClientIP()),
			zap.String("raw_body", string(bodyBytes)), // 记录原始请求体以供调试
		)
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "message": "无效的JSON格式"})
		return
	}

	// 使用后台协程处理，避免阻塞EMQX的Webhook调用
	go func() {
		// 创建一个新的context，以防原始的HTTP请求context被取消
		ctx := context.Background()

		var peerHost string
		if req.PeerHost != "" {
			peerHost = req.PeerHost
		}

		global.GVA_LOG.Info("收到EMQX WebHook事件",
			zap.String("event", req.Event),
			zap.String("clientID", req.ClientID),
			zap.String("username", req.Username),
			zap.String("ip", peerHost),
		)

		switch req.Event {
		case "client.connected":
			a.handleClientConnected(ctx, req)
		case "client.disconnected":
			a.handleClientDisconnected(ctx, req)
		default:
			global.GVA_LOG.Debug("忽略不处理的EMQX WebHook事件", zap.String("event", req.Event), zap.String("clientID", req.ClientID))
		}
	}()

	c.JSON(http.StatusOK, gin.H{"success": true, "message": "事件已接收"})
}

func (a *MqttAuthApi) handleClientConnected(ctx context.Context, req request.EmqxWebhookRequest) {
	if req.ClientID == "" {
		global.GVA_LOG.Warn("EMQX Webhook 'client.connected' 事件缺少 clientID")
		return
	}

	// 安全地将 json.Number 转换为 int64
	connectedAt, err := req.ConnectedAt.Int64()
	if err != nil {
		global.GVA_LOG.Warn("无法将 connected_at 转换为int64",
			zap.Error(err),
			zap.String("clientID", req.ClientID),
			zap.String("originalValue", string(req.ConnectedAt)),
		)
		// 在转换失败时可以根据业务需求设置一个默认值，例如0或当前时间戳
		connectedAt = 0
	}

	global.GVA_LOG.Info("客户端连接事件处理完成",
		zap.String("clientID", req.ClientID),
		zap.String("username", req.Username),
		zap.Int64("connectedAt", connectedAt))
}

func (a *MqttAuthApi) handleClientDisconnected(ctx context.Context, req request.EmqxWebhookRequest) {
	if req.ClientID == "" {
		global.GVA_LOG.Warn("EMQX Webhook 'client.disconnected' 事件缺少 clientID")
		return
	}

	// 安全地将 json.Number 转换为 int64
	disconnectedAt, err := req.DisconnectedAt.Int64()
	if err != nil {
		global.GVA_LOG.Warn("无法将 disconnected_at 转换为int64",
			zap.Error(err),
			zap.String("clientID", req.ClientID),
			zap.String("originalValue", string(req.DisconnectedAt)),
		)
		disconnectedAt = 0
	}

	global.GVA_LOG.Info("客户端断开事件处理完成",
		zap.String("clientID", req.ClientID),
		zap.String("username", req.Username),
		zap.String("reason", req.Reason),
		zap.Int64("disconnectedAt", disconnectedAt))
}
