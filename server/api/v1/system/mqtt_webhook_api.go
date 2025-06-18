package system

import (
	"github.com/flipped-aurora/gin-vue-admin/server/service"
	"github.com/gin-gonic/gin"
)

// MqttWebhookApi Mqtt Webhook API
type MqttWebhookApi struct{}

var mqttRelayService = service.ServiceGroupApp.NFCRelayServiceGroup.MQTTService

// HandleRoleRequest 处理角色请求的Webhook
func (api *MqttWebhookApi) HandleRoleRequest(c *gin.Context) {
	mqttRelayService.HandleRoleRequestWebhook(c)
}

// HandleConnectionStatus 处理连接状态的Webhook
func (api *MqttWebhookApi) HandleConnectionStatus(c *gin.Context) {
	mqttRelayService.HandleConnectionStatusWebhook(c)
}
