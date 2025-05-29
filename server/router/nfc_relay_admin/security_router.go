package nfc_relay_admin

import (
	v1 "github.com/flipped-aurora/gin-vue-admin/server/api/v1"
	"github.com/flipped-aurora/gin-vue-admin/server/middleware"
	"github.com/gin-gonic/gin"
)

type SecurityRouter struct{}

// InitSecurityRouter 初始化安全管理路由
func (s *SecurityRouter) InitSecurityRouter(Router *gin.RouterGroup) {
	securityRouter := Router.Group("security").Use(middleware.OperationRecord())
	securityRouterWithoutRecord := Router.Group("security")
	var securityApi = v1.ApiGroupApp.NfcRelayAdminApiGroup.SecurityManagementApi
	{
		securityRouter.POST("ban-client", securityApi.BanClient)                   // 封禁客户端
		securityRouter.POST("unban-client", securityApi.UnbanClient)               // 解封客户端
		securityRouter.POST("lock-user", securityApi.LockUserAccount)              // 锁定用户账户
		securityRouter.POST("unlock-user", securityApi.UnlockUserAccount)          // 解锁用户账户
		securityRouter.PUT("user-security", securityApi.UpdateUserSecurityProfile) // 更新用户安全档案
		securityRouter.POST("cleanup", securityApi.CleanupExpiredData)             // 清理过期数据
	}
	{
		securityRouterWithoutRecord.GET("client-bans", securityApi.GetClientBanList)                 // 获取客户端封禁列表
		securityRouterWithoutRecord.GET("client-ban-status/:clientID", securityApi.IsClientBanned)   // 检查客户端封禁状态
		securityRouterWithoutRecord.GET("user-security/:userID", securityApi.GetUserSecurityProfile) // 获取用户安全档案
		securityRouterWithoutRecord.GET("user-security", securityApi.GetUserSecurityProfileList)     // 获取用户安全档案列表
		securityRouterWithoutRecord.GET("summary", securityApi.GetSecuritySummary)                   // 获取安全摘要
	}
}
