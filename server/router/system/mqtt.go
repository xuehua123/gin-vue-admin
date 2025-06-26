package system

import (
	"github.com/gin-gonic/gin"
)

type MqttRouter struct{}

func (s *MqttRouter) InitMqttRouter(Router *gin.RouterGroup) {
	mqttRouter := Router.Group("mqtt")
	{
		// EMQX认证接口 - 使用现有的MqttAuthApi
		mqttRouter.POST("auth", mqttAuthApi.Authenticate) // MQTT客户端认证
		mqttRouter.POST("acl", mqttAuthApi.CheckACL)      // ACL权限检查

		// 新增：EMQX WebHook事件处理接口
		mqttRouter.POST("webhook", mqttAuthApi.WebHookEvent) // 处理客户端连接/断开事件
	}

	// MQTT Webhook路由 (使用现有的MqttWebhookApi)
	mqttHooksRouter := Router.Group("mqtt/hooks")
	{
		mqttHooksRouter.POST("role_request", mqttWebhookApi.HandleRoleRequest)           // 角色请求Hook
		mqttHooksRouter.POST("connection_status", mqttWebhookApi.HandleConnectionStatus) // 连接状态Hook
	}
}
