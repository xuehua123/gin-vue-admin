package system

import (
	"fmt"
	"net/http"
	"strings"

	"github.com/flipped-aurora/gin-vue-admin/server/global"
	"github.com/flipped-aurora/gin-vue-admin/server/utils"
	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

type MqttAuthApi struct{}

// Authenticate MQTT客户端认证
// Webhook for EMQX: /mqtt/auth
func (a *MqttAuthApi) Authenticate(c *gin.Context) {
	var req struct {
		ClientID string `json:"clientid"`
		Username string `json:"username"`
		Password string `json:"password"`
	}

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
	var req struct {
		ClientID string `json:"clientid"`
		Username string `json:"username"`
		Topic    string `json:"topic"`
		Action   string `json:"action"` // "publish" or "subscribe"
	}

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

	// 规则2：客户端只能订阅和发布到属于自己的私有主题
	// client/{clientID}/...
	clientPrefix := fmt.Sprintf("client/%s/", clientID)
	if strings.HasPrefix(topic, clientPrefix) {
		return true
	}

	// 规则3: 允许客户端发布到对端同步主题（需要从角色服务获取对端信息，此处简化）
	// user/{userID}/sync/peer_state_update
	// 这是一个复杂规则，暂时允许所有包含sync的发布操作
	if action == "publish" && strings.Contains(topic, "/sync/") {
		return true
	}
	
	// 规则4: 允许订阅用户级别的主题
	// user/{userID}/...
	// 需要从 clientID 解析出 userID，并与topic中的userID进行匹配
	// 暂时简化，如果topic以"user/"开头且是订阅操作，则允许
	if action == "subscribe" && strings.HasPrefix(topic, "user/") {
		// 在真实场景下，需要解析userID并进行严格匹配
		return true
	}


	// 默认拒绝
	return false
} 