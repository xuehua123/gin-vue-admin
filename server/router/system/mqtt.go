package system

import (
	"github.com/gin-gonic/gin"
)

type MqttRouter struct{}

func (s *MqttRouter) InitMqttRouter(Router *gin.RouterGroup) {
	mqttRouter := Router.Group("mqtt")
	{
		mqttRouter.POST("auth", mqttAuthApi.Authenticate)
		mqttRouter.POST("acl", mqttAuthApi.CheckACL)
	}

	// MQTT Webhook路由
	mqttHooksRouter := Router.Group("mqtt/hooks")
	{
		mqttHooksRouter.POST("role_request", mqttWebhookApi.RoleRequestHook)
		mqttHooksRouter.POST("connection_status", mqttWebhookApi.ConnectionStatusHook)
	}
}
