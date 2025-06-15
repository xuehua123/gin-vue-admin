package system

import (
	"github.com/gin-gonic/gin"
)

type JwtRouter struct{}

func (s *JwtRouter) InitJwtRouter(Router *gin.RouterGroup) {
	jwtRouter := Router.Group("jwt")
	{
		jwtRouter.POST("jsonInBlacklist", jwtApi.JsonInBlacklist)     // jwt加入黑名单(登出)
		jwtRouter.POST("generateMQTTToken", jwtApi.GenerateMQTTToken) // 生成MQTT专用JWT
		jwtRouter.POST("revokeMQTTToken", jwtApi.RevokeMQTTToken)     // 撤销MQTT JWT
		jwtRouter.GET("getUserMQTTTokens", jwtApi.GetUserMQTTTokens)  // 获取用户的所有MQTT Token
	}
}
