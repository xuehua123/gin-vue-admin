package nfc_relay_admin

import (
	v1 "github.com/flipped-aurora/gin-vue-admin/server/api/v1"
	"github.com/flipped-aurora/gin-vue-admin/server/middleware"
	"github.com/gin-gonic/gin"
)

type AuditLogsRouter struct{}

// InitAuditLogsRouter 初始化审计日志路由
func (a *AuditLogsRouter) InitAuditLogsRouter(Router *gin.RouterGroup) {
	auditLogsRouter := Router.Group("audit-logs").Use(middleware.OperationRecord())
	auditLogsRouterWithoutRecord := Router.Group("audit-logs")
	var auditLogsApi = v1.ApiGroupApp.NfcRelayAdminApiGroup.DatabaseAuditLogsApi
	{
		auditLogsRouter.POST("", auditLogsApi.CreateAuditLog)              // 创建审计日志
		auditLogsRouter.POST("batch", auditLogsApi.BatchCreateAuditLogs)   // 批量创建审计日志
		auditLogsRouter.DELETE("cleanup", auditLogsApi.DeleteOldAuditLogs) // 删除过期审计日志
	}
	{
		auditLogsRouterWithoutRecord.GET("", auditLogsApi.GetAuditLogList)       // 获取审计日志列表
		auditLogsRouterWithoutRecord.GET("stats", auditLogsApi.GetAuditLogStats) // 获取审计日志统计
	}
}
