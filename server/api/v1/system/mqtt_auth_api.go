package system

import (
	"context"
	"fmt"
	"net/http"
	"strings"

	"github.com/flipped-aurora/gin-vue-admin/server/global"
	"github.com/flipped-aurora/gin-vue-admin/server/model/system/request"
	"github.com/flipped-aurora/gin-vue-admin/server/utils"
	"github.com/gin-gonic/gin"
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

	// 使用Password字段传递JWT Token
	tokenString := req.Password
	if tokenString == "" {
		global.GVA_LOG.Warn("MQTT认证失败: Token为空", zap.String("clientID", req.ClientID), zap.String("username", req.Username))
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

	// 验证ClientID, Username是否与Token中的信息匹配
	if claims.ClientID != req.ClientID || claims.Username != req.Username {
		global.GVA_LOG.Warn("MQTT认证失败: ClientID或Username不匹配",
			zap.String("tokenClientID", claims.ClientID), zap.String("reqClientID", req.ClientID),
			zap.String("tokenUsername", claims.Username), zap.String("reqUsername", req.Username))
		c.JSON(http.StatusOK, gin.H{"result": "deny"})
		return
	}

	// 验证JWT是否依然有效（防止重放攻击和已吊销的token）
	isActive, err := jwtUtil.IsMQTTJWTActive(claims)
	if err != nil || !isActive {
		global.GVA_LOG.Warn("MQTT认证失败: Token无效或已被吊销", zap.Error(err), zap.String("clientID", req.ClientID))
		c.JSON(http.StatusOK, gin.H{"result": "deny"})
		return
	}

	// TODO: 后续可在此处更新客户端的连接状态，例如记录last_ping

	global.GVA_LOG.Info("MQTT客户端认证成功", zap.String("clientID", req.ClientID), zap.String("username", req.Username))
	c.JSON(http.StatusOK, gin.H{
		"result":       "allow",
		"is_superuser": false,
	})
}

// CheckACL MQTT权限控制
// Webhook for EMQX: /mqtt/acl
func (a *MqttAuthApi) CheckACL(c *gin.Context) {
	var req request.MqttAuthRequest

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusOK, gin.H{"result": "deny"})
		return
	}

	// 检查主题权限
	if checkTopicPermission(req.ClientID, req.Topic, req.Action) {
		c.JSON(http.StatusOK, gin.H{"result": "allow"})
	} else {
		global.GVA_LOG.Warn("MQTT ACL拒绝",
			zap.String("clientID", req.ClientID),
			zap.String("action", req.Action),
			zap.String("topic", req.Topic))
		c.JSON(http.StatusOK, gin.H{"result": "deny"})
	}
}

// checkTopicPermission 根据主题设计规范检查权限
// 此处为简化版实现，可根据实际需求扩展
func checkTopicPermission(clientID, topic, action string) bool {
	// 规则1：任何客户端都可以发布到系统主题，用于上报信息
	if action == "publish" {
		if strings.HasPrefix(topic, "system/") {
			return true
		}
	}

	// 规则2：新的基于交易ID的动态主题权限检查
	if strings.Contains(topic, "/transactions/") {
		return checkTransactionTopicPermission(clientID, topic, action)
	}

	// 规则3：兼容旧的客户端私有主题（逐步废弃）
	// client/{clientID}/... 或 clients/{clientID}/...
	clientPrefix := fmt.Sprintf("client/%s/", clientID)
	clientsPrefix := fmt.Sprintf("clients/%s/", clientID)
	if strings.HasPrefix(topic, clientPrefix) || strings.HasPrefix(topic, clientsPrefix) {
		global.GVA_LOG.Warn("使用了废弃的客户端主题格式",
			zap.String("clientID", clientID),
			zap.String("topic", topic),
			zap.String("action", action))
		return true
	}

	// 规则4: 允许客户端发布到对端同步主题（兼容旧格式，逐步废弃）
	if action == "publish" && strings.Contains(topic, "/sync/") {
		global.GVA_LOG.Warn("使用了废弃的同步主题格式",
			zap.String("clientID", clientID),
			zap.String("topic", topic))
		return true
	}

	// 规则5: 允许订阅用户级别的主题（兼容旧格式）
	if action == "subscribe" && strings.HasPrefix(topic, "user/") {
		return true
	}

	// 规则6: 允许订阅 nfc_relay 全局主题（兼容旧格式，逐步废弃）
	if action == "subscribe" {
		if strings.HasPrefix(topic, "nfc_relay/transactions/") {
			if strings.Contains(topic, "/status") ||
				strings.Contains(topic, "/apdu/") ||
				strings.Contains(topic, "/heartbeat") {
				global.GVA_LOG.Warn("使用了废弃的全局主题格式",
					zap.String("clientID", clientID),
					zap.String("topic", topic))
				return true
			}
		}
	}

	// 规则7: 允许发布到 nfc_relay 系统主题（兼容旧格式）
	if action == "publish" && strings.HasPrefix(topic, "nfc_relay/") {
		return true
	}

	// 默认拒绝
	global.GVA_LOG.Warn("MQTT主题权限被拒绝",
		zap.String("clientID", clientID),
		zap.String("topic", topic),
		zap.String("action", action))
	return false
}

// checkTransactionTopicPermission 检查基于交易ID的主题权限
func checkTransactionTopicPermission(clientID, topic, action string) bool {
	// 解析主题格式: {prefix}/transactions/{transactionID}/{role}/{subtopic}
	// 例如: nfc_relay/transactions/txn_123456/transmitter/state

	parts := strings.Split(topic, "/")
	if len(parts) < 5 {
		global.GVA_LOG.Debug("主题格式不符合交易主题规范",
			zap.String("topic", topic),
			zap.Int("parts", len(parts)))
		return false
	}

	// 查找 transactions 位置
	transactionIndex := -1
	for i, part := range parts {
		if part == "transactions" {
			transactionIndex = i
			break
		}
	}

	if transactionIndex == -1 || transactionIndex+3 >= len(parts) {
		global.GVA_LOG.Debug("主题格式中未找到有效的交易段",
			zap.String("topic", topic))
		return false
	}

	transactionID := parts[transactionIndex+1]
	targetRole := parts[transactionIndex+2]
	subtopic := parts[transactionIndex+3]

	// 从Redis检查客户端是否参与此交易
	ctx := context.Background()
	clientCurrentTransactionKey := fmt.Sprintf("client:%s:current_transaction", clientID)
	currentTransactionID, err := global.GVA_REDIS.Get(ctx, clientCurrentTransactionKey).Result()
	if err != nil || currentTransactionID != transactionID {
		global.GVA_LOG.Debug("客户端未参与此交易或交易不匹配",
			zap.String("clientID", clientID),
			zap.String("requestedTransactionID", transactionID),
			zap.String("currentTransactionID", currentTransactionID),
			zap.Error(err))
		return false
	}

	// 获取客户端在交易中的角色
	transactionClientsKey := fmt.Sprintf("transaction:%s:clients", transactionID)
	clientRole := ""

	if transmitterClientID, err := global.GVA_REDIS.HGet(ctx, transactionClientsKey, "transmitter_client_id").Result(); err == nil && transmitterClientID == clientID {
		clientRole = "transmitter"
	} else if receiverClientID, err := global.GVA_REDIS.HGet(ctx, transactionClientsKey, "receiver_client_id").Result(); err == nil && receiverClientID == clientID {
		clientRole = "receiver"
	} else {
		global.GVA_LOG.Debug("无法确定客户端在交易中的角色",
			zap.String("clientID", clientID),
			zap.String("transactionID", transactionID))
		return false
	}

	// 权限检查逻辑
	switch subtopic {
	case "state":
		// 状态主题权限：客户端只能发布到自己的状态主题，只能订阅对端的状态主题
		if action == "publish" {
			return targetRole == clientRole
		} else if action == "subscribe" {
			return targetRole != clientRole // 订阅对端状态
		}

	case "apdu":
		// APDU主题权限检查
		if len(parts) > transactionIndex+4 {
			apduDirection := parts[transactionIndex+4] // to_transmitter 或 to_receiver

			if action == "publish" {
				// 客户端只能发布到对端的APDU主题
				if clientRole == "transmitter" {
					return apduDirection == "to_receiver"
				} else {
					return apduDirection == "to_transmitter"
				}
			} else if action == "subscribe" {
				// 客户端只能订阅发给自己的APDU主题
				if clientRole == "transmitter" {
					return apduDirection == "to_transmitter"
				} else {
					return apduDirection == "to_receiver"
				}
			}
		}

	case "control":
		// 控制主题：双方都可以发布和订阅
		return true

	case "heartbeat":
		// 心跳主题：双方都可以发布和订阅
		return true

	case "session":
		// 会话主题：只能订阅，不能发布（由服务器发布）
		return action == "subscribe"
	}

	global.GVA_LOG.Debug("未匹配到任何交易主题权限规则",
		zap.String("clientID", clientID),
		zap.String("topic", topic),
		zap.String("action", action),
		zap.String("subtopic", subtopic),
		zap.String("clientRole", clientRole),
		zap.String("targetRole", targetRole))

	return false
}
