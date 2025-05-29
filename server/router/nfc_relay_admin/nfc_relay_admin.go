package nfc_relay_admin

import (
	v1 "github.com/flipped-aurora/gin-vue-admin/server/api/v1"
	"github.com/flipped-aurora/gin-vue-admin/server/middleware"
	"github.com/gin-gonic/gin"
)

type NfcRelayAdminRouter struct{}

// InitNfcRelayAdminRouter 初始化NFC中继管理路由
// 路径统一为 /admin/nfc-relay/v1/ 以匹配前端API调用
func (r *NfcRelayAdminRouter) InitNfcRelayAdminRouter(Router *gin.RouterGroup) {
	// 创建 NFC 中继管理 API 路由组，路径为 admin/nfc-relay/v1
	// 这将匹配前端的 API_PREFIX = '/api/admin/nfc-relay/v1'
	nfcRelayAdminRouter := Router.Group("admin/nfc-relay/v1").Use(middleware.OperationRecord())
	nfcRelayAdminApi := v1.ApiGroupApp.NfcRelayAdminApiGroup

	{
		// 仪表盘路由 - 增强版 (基于已实现的API)
		nfcRelayAdminRouter.GET("dashboard-stats-enhanced", nfcRelayAdminApi.DashboardEnhancedApi.GetDashboardStatsEnhanced) // 获取增强版仪表盘数据
		nfcRelayAdminRouter.GET("performance-metrics", nfcRelayAdminApi.DashboardEnhancedApi.GetPerformanceMetrics)          // 获取性能指标
		nfcRelayAdminRouter.GET("geographic-distribution", nfcRelayAdminApi.DashboardEnhancedApi.GetGeographicDistribution)  // 获取地理分布
		nfcRelayAdminRouter.GET("alerts", nfcRelayAdminApi.DashboardEnhancedApi.GetAlerts)                                   // 获取告警信息
		nfcRelayAdminRouter.POST("alerts/:alert_id/acknowledge", nfcRelayAdminApi.DashboardEnhancedApi.AcknowledgeAlert)     // 确认告警
		nfcRelayAdminRouter.POST("export", nfcRelayAdminApi.DashboardEnhancedApi.ExportDashboardData)                        // 导出数据
		nfcRelayAdminRouter.GET("comparison", nfcRelayAdminApi.DashboardEnhancedApi.GetComparisonData)                       // 获取对比数据

		// 客户端管理路由 (基于已实现的API)
		nfcRelayAdminRouter.GET("clients", nfcRelayAdminApi.ClientsApi.GetClients)                             // 获取客户端列表
		nfcRelayAdminRouter.GET("clients/:clientID/details", nfcRelayAdminApi.ClientsApi.GetClientDetails)     // 获取客户端详情
		nfcRelayAdminRouter.POST("clients/:clientID/disconnect", nfcRelayAdminApi.ClientsApi.DisconnectClient) // 强制断开客户端

		// 会话管理路由 (基于已实现的API)
		nfcRelayAdminRouter.GET("sessions", nfcRelayAdminApi.SessionsApi.GetSessions)                            // 获取会话列表
		nfcRelayAdminRouter.GET("sessions/:sessionID/details", nfcRelayAdminApi.SessionsApi.GetSessionDetails)   // 获取会话详情
		nfcRelayAdminRouter.POST("sessions/:sessionID/terminate", nfcRelayAdminApi.SessionsApi.TerminateSession) // 强制终止会话

		// 审计日志路由 (基于文件系统的已实现API)
		nfcRelayAdminRouter.GET("audit-logs", nfcRelayAdminApi.AuditLogsApi.GetAuditLogs) // 获取审计日志

		// 数据库审计日志路由 (新增)
		nfcRelayAdminRouter.POST("audit-logs-db", nfcRelayAdminApi.DatabaseAuditLogsApi.CreateAuditLog)               // 创建审计日志
		nfcRelayAdminRouter.GET("audit-logs-db", nfcRelayAdminApi.DatabaseAuditLogsApi.GetAuditLogList)               // 获取审计日志列表
		nfcRelayAdminRouter.GET("audit-logs-db/stats", nfcRelayAdminApi.DatabaseAuditLogsApi.GetAuditLogStats)        // 获取审计日志统计
		nfcRelayAdminRouter.POST("audit-logs-db/batch", nfcRelayAdminApi.DatabaseAuditLogsApi.BatchCreateAuditLogs)   // 批量创建审计日志
		nfcRelayAdminRouter.DELETE("audit-logs-db/cleanup", nfcRelayAdminApi.DatabaseAuditLogsApi.DeleteOldAuditLogs) // 删除过期审计日志

		// 安全管理路由 (新增)
		nfcRelayAdminRouter.POST("security/ban-client", nfcRelayAdminApi.SecurityManagementApi.BanClient)                        // 封禁客户端
		nfcRelayAdminRouter.POST("security/unban-client", nfcRelayAdminApi.SecurityManagementApi.UnbanClient)                    // 解封客户端
		nfcRelayAdminRouter.GET("security/client-bans", nfcRelayAdminApi.SecurityManagementApi.GetClientBanList)                 // 获取客户端封禁列表
		nfcRelayAdminRouter.GET("security/client-ban-status/:clientID", nfcRelayAdminApi.SecurityManagementApi.IsClientBanned)   // 检查客户端封禁状态
		nfcRelayAdminRouter.GET("security/user-security/:userID", nfcRelayAdminApi.SecurityManagementApi.GetUserSecurityProfile) // 获取用户安全档案
		nfcRelayAdminRouter.GET("security/user-security", nfcRelayAdminApi.SecurityManagementApi.GetUserSecurityProfileList)     // 获取用户安全档案列表
		nfcRelayAdminRouter.PUT("security/user-security", nfcRelayAdminApi.SecurityManagementApi.UpdateUserSecurityProfile)      // 更新用户安全档案
		nfcRelayAdminRouter.POST("security/lock-user", nfcRelayAdminApi.SecurityManagementApi.LockUserAccount)                   // 锁定用户账户
		nfcRelayAdminRouter.POST("security/unlock-user", nfcRelayAdminApi.SecurityManagementApi.UnlockUserAccount)               // 解锁用户账户
		nfcRelayAdminRouter.GET("security/summary", nfcRelayAdminApi.SecurityManagementApi.GetSecuritySummary)                   // 获取安全摘要
		nfcRelayAdminRouter.POST("security/cleanup", nfcRelayAdminApi.SecurityManagementApi.CleanupExpiredData)                  // 清理过期数据

		// 安全配置路由 (新增SecurityConfigAPI)
		nfcRelayAdminRouter.GET("security/config", nfcRelayAdminApi.SecurityConfigAPI.GetSecurityConfig)            // 获取安全配置
		nfcRelayAdminRouter.PUT("security/config", nfcRelayAdminApi.SecurityConfigAPI.UpdateSecurityConfig)         // 更新安全配置
		nfcRelayAdminRouter.GET("security/compliance-stats", nfcRelayAdminApi.SecurityConfigAPI.GetComplianceStats) // 获取合规统计
		nfcRelayAdminRouter.POST("security/test-features", nfcRelayAdminApi.SecurityConfigAPI.TestSecurityFeatures) // 测试安全功能
		nfcRelayAdminRouter.POST("security/unblock-user/:userId", nfcRelayAdminApi.SecurityConfigAPI.UnblockUser)   // 解除用户封禁
		nfcRelayAdminRouter.GET("security/status", nfcRelayAdminApi.SecurityConfigAPI.GetSecurityStatus)            // 获取安全状态

		// 系统配置路由 (基于已实现的API)
		nfcRelayAdminRouter.GET("config", nfcRelayAdminApi.ConfigApi.GetConfig) // 获取系统配置

		// 实时数据路由 (基于已实现的API)
		nfcRelayAdminRouter.GET("realtime", nfcRelayAdminApi.RealtimeApi.HandleWebSocket) // WebSocket实时数据

		// === 新增接口路由 ===

		// 加密验证API路由 (3个接口)
		nfcRelayAdminRouter.POST("encryption/decrypt-verify", nfcRelayAdminApi.EncryptionVerificationApi.DecryptAndVerify)            // 解密和验证APDU数据
		nfcRelayAdminRouter.POST("encryption/batch-decrypt-verify", nfcRelayAdminApi.EncryptionVerificationApi.BatchDecryptAndVerify) // 批量解密和验证
		nfcRelayAdminRouter.GET("encryption/status", nfcRelayAdminApi.EncryptionVerificationApi.GetEncryptionStatus)                  // 获取加密状态

		// 配置热重载API路由 (6个接口)
		nfcRelayAdminRouter.POST("config/reload", nfcRelayAdminApi.ConfigReloadApi.ReloadConfig)                  // 重载配置
		nfcRelayAdminRouter.GET("config/status", nfcRelayAdminApi.ConfigReloadApi.GetConfigStatus)                // 获取配置状态
		nfcRelayAdminRouter.GET("config/hot-reload-status", nfcRelayAdminApi.ConfigReloadApi.GetHotReloadStatus)  // 获取热重载状态
		nfcRelayAdminRouter.POST("config/hot-reload/toggle", nfcRelayAdminApi.ConfigReloadApi.ToggleHotReload)    // 切换热重载功能
		nfcRelayAdminRouter.POST("config/revert/:config_type", nfcRelayAdminApi.ConfigReloadApi.RevertConfig)     // 回滚配置
		nfcRelayAdminRouter.GET("config/history/:config_type", nfcRelayAdminApi.ConfigReloadApi.GetConfigHistory) // 获取配置变更历史

		// 合规规则管理API路由 (8个接口)
		nfcRelayAdminRouter.GET("compliance/rules", nfcRelayAdminApi.ComplianceRulesApi.GetComplianceRules)               // 获取所有合规规则
		nfcRelayAdminRouter.GET("compliance/rules/:rule_id", nfcRelayAdminApi.ComplianceRulesApi.GetComplianceRule)       // 获取单个合规规则
		nfcRelayAdminRouter.POST("compliance/rules", nfcRelayAdminApi.ComplianceRulesApi.CreateComplianceRule)            // 创建合规规则
		nfcRelayAdminRouter.PUT("compliance/rules/:rule_id", nfcRelayAdminApi.ComplianceRulesApi.UpdateComplianceRule)    // 更新合规规则
		nfcRelayAdminRouter.DELETE("compliance/rules/:rule_id", nfcRelayAdminApi.ComplianceRulesApi.DeleteComplianceRule) // 删除合规规则
		nfcRelayAdminRouter.POST("compliance/rules/test", nfcRelayAdminApi.ComplianceRulesApi.TestComplianceRule)         // 测试合规规则
		nfcRelayAdminRouter.GET("compliance/rule-files", nfcRelayAdminApi.ComplianceRulesApi.GetRuleFiles)                // 获取规则文件列表
		nfcRelayAdminRouter.POST("compliance/rule-files/import", nfcRelayAdminApi.ComplianceRulesApi.ImportRuleFile)      // 导入规则文件
		nfcRelayAdminRouter.GET("compliance/rule-files/export", nfcRelayAdminApi.ComplianceRulesApi.ExportRuleFile)       // 导出规则文件

		// 配置变更审计API路由 (5个接口)
		nfcRelayAdminRouter.GET("config-audit/logs", nfcRelayAdminApi.ConfigAuditApi.GetConfigAuditLogs)                  // 获取配置审计日志
		nfcRelayAdminRouter.GET("config-audit/stats", nfcRelayAdminApi.ConfigAuditApi.GetConfigAuditStats)                // 获取配置审计统计
		nfcRelayAdminRouter.GET("config-audit/changes/:change_id", nfcRelayAdminApi.ConfigAuditApi.GetConfigChangeDetail) // 获取配置变更详情
		nfcRelayAdminRouter.POST("config-audit/records", nfcRelayAdminApi.ConfigAuditApi.CreateConfigAuditRecord)         // 创建配置审计记录
		nfcRelayAdminRouter.GET("config-audit/export", nfcRelayAdminApi.ConfigAuditApi.ExportConfigAuditLogs)             // 导出配置审计日志
	}
}
