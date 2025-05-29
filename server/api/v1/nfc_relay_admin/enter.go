package nfc_relay_admin

type ApiGroup struct {
	DashboardEnhancedApi      DashboardEnhancedApi      // 增强仪表盘API
	RealtimeApi               RealtimeApi               // 实时数据API
	ClientsApi                ClientsApi                // 客户端管理API
	SessionsApi               SessionsApi               // 会话管理API
	AuditLogsApi              AuditLogsApi              // 审计日志API (文件系统)
	DatabaseAuditLogsApi      DatabaseAuditLogsApi      // 审计日志API (数据库)
	SecurityManagementApi     SecurityManagementApi     // 安全管理API
	SecurityConfigAPI         SecurityConfigAPI         // 安全配置API
	ConfigApi                 ConfigApi                 // 系统配置API
	EncryptionVerificationApi EncryptionVerificationApi // 加密验证API
	ConfigReloadApi           ConfigReloadApi           // 配置热重载API
	ComplianceRulesApi        ComplianceRulesApi        // 合规规则管理API
	ConfigAuditApi            ConfigAuditApi            // 配置变更审计API
}
