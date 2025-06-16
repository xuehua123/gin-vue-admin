package system

import (
	"github.com/gin-gonic/gin"
)

type RoleConflictRouter struct{}

func (s *RoleConflictRouter) InitRoleConflictRouter(Router *gin.RouterGroup) {
	roleRouter := Router.Group("role")
	{
		roleRouter.POST("generateMQTTToken", roleConflictApi.GenerateMQTTToken)
		roleRouter.POST("checkConflict", roleConflictApi.CheckRoleConflict)
	}
}
