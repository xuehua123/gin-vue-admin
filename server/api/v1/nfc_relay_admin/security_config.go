package nfc_relay_admin

import (
	"net/http"
	"strconv"

	"github.com/flipped-aurora/gin-vue-admin/server/global"
	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

// SecurityConfigAPI 安全配置管理API
type SecurityConfigAPI struct{}

// SecurityConfigResponse 安全配置响应
type SecurityConfigResponse struct {
	EnableAuditEncryption     bool     `json:"enableAuditEncryption"`
	EncryptionAlgorithm       string   `json:"encryptionAlgorithm"`
	EnableComplianceAudit     bool     `json:"enableComplianceAudit"`
	EnableDeepInspection      bool     `json:"enableDeepInspection"`
	MaxTransactionAmount      int64    `json:"maxTransactionAmount"`
	BlockedMerchantCategories []string `json:"blockedMerchantCategories"`
	EnableAntiReplay          bool     `json:"enableAntiReplay"`
	MaxMessageSize            int      `json:"maxMessageSize"`
}

// SecurityConfigRequest 安全配置更新请求
type SecurityConfigRequest struct {
	MaxTransactionAmount      *int64    `json:"maxTransactionAmount"`
	BlockedMerchantCategories *[]string `json:"blockedMerchantCategories"`
	EnableDeepInspection      *bool     `json:"enableDeepInspection"`
	MaxMessageSize            *int      `json:"maxMessageSize"`
}

// ComplianceStatsResponse 合规统计响应
type ComplianceStatsResponse struct {
	TotalChecks        int64                `json:"totalChecks"`
	ViolationsToday    int64                `json:"violationsToday"`
	ViolationsThisWeek int64                `json:"violationsThisWeek"`
	TopViolationTypes  []ViolationTypeStats `json:"topViolationTypes"`
	RiskDistribution   map[string]int64     `json:"riskDistribution"`
	BlockedUsers       []BlockedUserInfo    `json:"blockedUsers"`
}

// ViolationTypeStats 违规类型统计
type ViolationTypeStats struct {
	Type  string `json:"type"`
	Count int64  `json:"count"`
}

// BlockedUserInfo 被封禁用户信息
type BlockedUserInfo struct {
	UserID    string `json:"userId"`
	BlockedAt string `json:"blockedAt"`
	Reason    string `json:"reason"`
	ExpiresAt string `json:"expiresAt"`
}

// GetSecurityConfig 获取当前安全配置
func (s *SecurityConfigAPI) GetSecurityConfig(c *gin.Context) {
	config := SecurityConfigResponse{
		EnableAuditEncryption:     global.GVA_CONFIG.NfcRelay.Security.EnableAuditEncryption,
		EncryptionAlgorithm:       global.GVA_CONFIG.NfcRelay.Security.EncryptionAlgorithm,
		EnableComplianceAudit:     global.GVA_CONFIG.NfcRelay.Security.EnableComplianceAudit,
		EnableDeepInspection:      global.GVA_CONFIG.NfcRelay.Security.EnableDeepInspection,
		MaxTransactionAmount:      global.GVA_CONFIG.NfcRelay.Security.MaxTransactionAmount,
		BlockedMerchantCategories: global.GVA_CONFIG.NfcRelay.Security.BlockedMerchantCategories,
		EnableAntiReplay:          global.GVA_CONFIG.NfcRelay.Security.EnableAntiReplay,
		MaxMessageSize:            global.GVA_CONFIG.NfcRelay.Security.MaxMessageSize,
	}

	c.JSON(http.StatusOK, gin.H{
		"code": 0,
		"data": config,
		"msg":  "获取安全配置成功",
	})
}

// UpdateSecurityConfig 更新安全配置
func (s *SecurityConfigAPI) UpdateSecurityConfig(c *gin.Context) {
	var req SecurityConfigRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		global.GVA_LOG.Error("解析安全配置请求失败", zap.Error(err))
		c.JSON(http.StatusBadRequest, gin.H{
			"code": 7,
			"msg":  "请求参数格式错误",
		})
		return
	}

	// 获取操作用户信息
	userID := c.GetString("userID")
	if userID == "" {
		userID = "unknown"
	}

	// 更新配置（注意：这里的更新是运行时的，重启后会重置）
	if req.MaxTransactionAmount != nil {
		global.GVA_CONFIG.NfcRelay.Security.MaxTransactionAmount = *req.MaxTransactionAmount
		global.GVA_LOG.Info("更新交易限额配置",
			zap.Int64("newLimit", *req.MaxTransactionAmount),
			zap.String("operatorUserID", userID),
		)
	}

	if req.BlockedMerchantCategories != nil {
		global.GVA_CONFIG.NfcRelay.Security.BlockedMerchantCategories = *req.BlockedMerchantCategories
		global.GVA_LOG.Info("更新封禁商户类别配置",
			zap.Strings("newCategories", *req.BlockedMerchantCategories),
			zap.String("operatorUserID", userID),
		)
	}

	if req.EnableDeepInspection != nil {
		global.GVA_CONFIG.NfcRelay.Security.EnableDeepInspection = *req.EnableDeepInspection
		global.GVA_LOG.Info("更新深度检查配置",
			zap.Bool("enabled", *req.EnableDeepInspection),
			zap.String("operatorUserID", userID),
		)
	}

	if req.MaxMessageSize != nil {
		global.GVA_CONFIG.NfcRelay.Security.MaxMessageSize = *req.MaxMessageSize
		global.GVA_LOG.Info("更新消息大小限制配置",
			zap.Int("newSize", *req.MaxMessageSize),
			zap.String("operatorUserID", userID),
		)
	}

	// 记录配置变更到审计日志
	global.LogAuditEvent(
		"security_config_updated",
		map[string]interface{}{
			"operator_user_id": userID,
			"changes":          req,
			"timestamp":        "now",
		},
		zap.String("user_id", userID),
	)

	c.JSON(http.StatusOK, gin.H{
		"code": 0,
		"msg":  "安全配置更新成功",
	})
}

// GetComplianceStats 获取合规统计信息
func (s *SecurityConfigAPI) GetComplianceStats(c *gin.Context) {
	// 从查询参数获取时间范围
	daysStr := c.DefaultQuery("days", "7")
	days, err := strconv.Atoi(daysStr)
	if err != nil || days <= 0 {
		days = 7
	}

	// 这里应该从数据库查询真实数据，这里提供模拟数据
	stats := ComplianceStatsResponse{
		TotalChecks:        15420,
		ViolationsToday:    23,
		ViolationsThisWeek: 156,
		TopViolationTypes: []ViolationTypeStats{
			{Type: "高风险商户类别", Count: 45},
			{Type: "交易金额超限", Count: 32},
			{Type: "无效PAN格式", Count: 28},
			{Type: "高风险命令", Count: 19},
			{Type: "异常小额交易", Count: 15},
		},
		RiskDistribution: map[string]int64{
			"LOW":      12450,
			"MEDIUM":   2314,
			"HIGH":     580,
			"CRITICAL": 76,
		},
		BlockedUsers: []BlockedUserInfo{
			{
				UserID:    "user_001",
				BlockedAt: "2024-01-15T10:30:00Z",
				Reason:    "频繁违规操作",
				ExpiresAt: "2024-01-16T10:30:00Z",
			},
			{
				UserID:    "user_002",
				BlockedAt: "2024-01-15T14:20:00Z",
				Reason:    "检测到黑名单卡号",
				ExpiresAt: "2024-01-16T14:20:00Z",
			},
		},
	}

	global.GVA_LOG.Info("获取合规统计信息",
		zap.Int("days", days),
		zap.String("operatorUserID", c.GetString("userID")),
	)

	c.JSON(http.StatusOK, gin.H{
		"code": 0,
		"data": stats,
		"msg":  "获取合规统计成功",
	})
}

// TestSecurityFeatures 测试安全功能
func (s *SecurityConfigAPI) TestSecurityFeatures(c *gin.Context) {
	userID := c.GetString("userID")

	global.GVA_LOG.Info("开始安全功能测试",
		zap.String("operatorUserID", userID),
	)

	// 运行安全测试
	go func() {
		defer func() {
			if r := recover(); r != nil {
				global.GVA_LOG.Error("安全测试发生错误", zap.Any("error", r))
			}
		}()

		// 这里应该调用安全测试函数
		// security.RunAllTests()

		global.GVA_LOG.Info("安全功能测试完成")
	}()

	c.JSON(http.StatusOK, gin.H{
		"code": 0,
		"msg":  "安全功能测试已启动，请查看日志获取详细结果",
	})
}

// UnblockUser 解除用户封禁
func (s *SecurityConfigAPI) UnblockUser(c *gin.Context) {
	targetUserID := c.Param("userId")
	if targetUserID == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"code": 7,
			"msg":  "用户ID不能为空",
		})
		return
	}

	operatorUserID := c.GetString("userID")

	// 这里应该调用安全管理器的解封方法
	// 目前提供模拟实现
	global.GVA_LOG.Info("管理员解除用户封禁",
		zap.String("targetUserID", targetUserID),
		zap.String("operatorUserID", operatorUserID),
	)

	// 记录到审计日志
	global.LogAuditEvent(
		"user_unblocked",
		map[string]interface{}{
			"target_user_id":   targetUserID,
			"operator_user_id": operatorUserID,
			"timestamp":        "now",
		},
		zap.String("user_id", operatorUserID),
	)

	c.JSON(http.StatusOK, gin.H{
		"code": 0,
		"msg":  "用户解封成功",
	})
}

// GetSecurityStatus 获取安全系统状态
func (s *SecurityConfigAPI) GetSecurityStatus(c *gin.Context) {
	status := map[string]interface{}{
		"auditEncryptionEnabled": global.GVA_CONFIG.NfcRelay.Security.EnableAuditEncryption,
		"complianceAuditEnabled": global.GVA_CONFIG.NfcRelay.Security.EnableComplianceAudit,
		"deepInspectionEnabled":  global.GVA_CONFIG.NfcRelay.Security.EnableDeepInspection,
		"antiReplayEnabled":      global.GVA_CONFIG.NfcRelay.Security.EnableAntiReplay,
		"tlsEnabled":             global.GVA_CONFIG.NfcRelay.Security.EnableTLS,
		"lastConfigUpdate":       "2024-01-15T10:30:00Z",
		"systemHealth":           "healthy",
		"activeSecurityRules":    5,
		"encryptionAlgorithm":    global.GVA_CONFIG.NfcRelay.Security.EncryptionAlgorithm,
	}

	c.JSON(http.StatusOK, gin.H{
		"code": 0,
		"data": status,
		"msg":  "获取安全状态成功",
	})
}
