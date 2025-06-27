package system

import (
	"github.com/flipped-aurora/gin-vue-admin/server/service/nfc_relay"
	"github.com/gin-gonic/gin"
)

// MqttWebhookApi Mqtt Webhook API
type MqttWebhookApi struct{}

// HandleRoleRequest handles the webhook for role requests.
func (h *MqttWebhookApi) HandleRoleRequest(c *gin.Context) {
	nfc_relay.ServiceGroupApp.MqttService().HandleRoleRequestWebhook(c)
}

// HandleConnectionStatus handles the webhook for connection status updates.
func (h *MqttWebhookApi) HandleConnectionStatus(c *gin.Context) {
	nfc_relay.ServiceGroupApp.MqttService().HandleConnectionStatusWebhook(c)
}
