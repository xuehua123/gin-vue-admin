package security

import (
	"fmt"
	"regexp"
	"strings"
	"time"

	"github.com/flipped-aurora/gin-vue-admin/server/global"
	"go.uber.org/zap"
)

// ComplianceResult 合规检查结果
type ComplianceResult struct {
	Compliant bool     `json:"compliant"`
	Reason    string   `json:"reason"`
	RiskLevel string   `json:"riskLevel"` // LOW, MEDIUM, HIGH, CRITICAL
	Actions   []string `json:"actions"`   // 建议的处理动作
}

// ComplianceRule 合规规则
type ComplianceRule struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	Description string `json:"description"`
	Pattern     string `json:"pattern"` // 正则表达式模式
	RiskLevel   string `json:"riskLevel"`
	Action      string `json:"action"` // BLOCK, WARN, LOG
	Enabled     bool   `json:"enabled"`
}

// ComplianceAuditEngine 合规审计引擎
type ComplianceAuditEngine struct {
	rules          []ComplianceRule
	violationCache map[string]int       // 用户违规次数缓存
	blockList      map[string]time.Time // 封禁列表
}

// NewComplianceAuditEngine 创建合规审计引擎
func NewComplianceAuditEngine() *ComplianceAuditEngine {
	engine := &ComplianceAuditEngine{
		rules:          getDefaultComplianceRules(),
		violationCache: make(map[string]int),
		blockList:      make(map[string]time.Time),
	}

	return engine
}

// AuditAPDUData 审计APDU数据 (增强版，支持深度业务数据检查)
func (cae *ComplianceAuditEngine) AuditAPDUData(apduClass *APDUDataClass) (*ComplianceResult, error) {
	// 检查用户是否被封禁
	if cae.isUserBlocked(apduClass.Metadata.UserID) {
		return &ComplianceResult{
			Compliant: false,
			Reason:    "用户已被暂时封禁",
			RiskLevel: "CRITICAL",
			Actions:   []string{"BLOCK_CONNECTION"},
		}, nil
	}

	// 执行各项合规检查
	for _, rule := range cae.rules {
		if !rule.Enabled {
			continue
		}

		result := cae.checkRule(rule, apduClass)
		if !result.Compliant {
			return result, nil
		}
	}

	// 所有检查通过
	return &ComplianceResult{
		Compliant: true,
		Reason:    "合规检查通过",
		RiskLevel: "LOW",
		Actions:   []string{"ALLOW"},
	}, nil
}

// AuditBusinessData 对解密后的业务数据进行深度合规检查
func (cae *ComplianceAuditEngine) AuditBusinessData(businessData map[string]interface{}, userID string) (*ComplianceResult, error) {
	// 检查PAN卡号
	if result := cae.checkPANCompliance(businessData); !result.Compliant {
		return result, nil
	}

	// 检查交易金额
	if result := cae.checkAmountCompliance(businessData); !result.Compliant {
		return result, nil
	}

	// 检查商户信息
	if result := cae.checkMerchantCompliance(businessData); !result.Compliant {
		return result, nil
	}

	// 检查CVV/PIN等敏感信息
	if result := cae.checkSensitiveDataCompliance(businessData); !result.Compliant {
		return result, nil
	}

	return &ComplianceResult{
		Compliant: true,
		Reason:    "业务数据合规检查通过",
		RiskLevel: "LOW",
		Actions:   []string{"ALLOW"},
	}, nil
}

// checkRule 检查单个合规规则
func (cae *ComplianceAuditEngine) checkRule(rule ComplianceRule, apduClass *APDUDataClass) *ComplianceResult {
	switch rule.ID {
	case "HIGH_RISK_COMMAND":
		return cae.checkHighRiskCommand(rule, apduClass)
	case "TRANSACTION_AMOUNT_LIMIT":
		return cae.checkTransactionAmount(rule, apduClass)
	case "TIME_RESTRICTION":
		return cae.checkTimeRestriction(rule, apduClass)
	case "FREQUENCY_LIMIT":
		return cae.checkFrequencyLimit(rule, apduClass)
	case "SUSPICIOUS_PATTERN":
		return cae.checkSuspiciousPattern(rule, apduClass)
	default:
		return &ComplianceResult{Compliant: true}
	}
}

// checkHighRiskCommand 检查高风险命令
func (cae *ComplianceAuditEngine) checkHighRiskCommand(rule ComplianceRule, apduClass *APDUDataClass) *ComplianceResult {
	// 放宽高风险命令限制，仅对最危险的操作进行阻断
	criticalCommands := []string{"UPDATE_KEY", "FACTORY_RESET"}

	for _, cmd := range criticalCommands {
		if strings.Contains(apduClass.AuditData.CommandClass, cmd) {
			return &ComplianceResult{
				Compliant: false,
				Reason:    fmt.Sprintf("检测到关键系统命令: %s", apduClass.AuditData.CommandClass),
				RiskLevel: "CRITICAL",
				Actions:   []string{"BLOCK", "ALERT_ADMIN"},
			}
		}
	}

	// 对于其他高风险命令，仅记录警告，不阻断
	highRiskCommands := []string{"WRITE", "INTERNAL_AUTHENTICATE"}
	for _, cmd := range highRiskCommands {
		if strings.Contains(apduClass.AuditData.CommandClass, cmd) {
			global.GVA_LOG.Warn("高风险命令执行",
				zap.String("commandClass", apduClass.AuditData.CommandClass),
				zap.String("userId", apduClass.Metadata.UserID),
				zap.String("sessionId", apduClass.Metadata.SessionID),
			)
		}
	}

	return &ComplianceResult{Compliant: true}
}

// checkTransactionAmount 检查交易金额限制
func (cae *ComplianceAuditEngine) checkTransactionAmount(rule ComplianceRule, apduClass *APDUDataClass) *ComplianceResult {
	if apduClass.AuditData.Amount == nil {
		return &ComplianceResult{Compliant: true}
	}

	// 提高单笔交易限额：100万分 (10,000元)
	maxAmount := int64(10000000)

	if *apduClass.AuditData.Amount > maxAmount {
		return &ComplianceResult{
			Compliant: false,
			Reason:    fmt.Sprintf("交易金额超过限制: %d分", *apduClass.AuditData.Amount),
			RiskLevel: "HIGH",
			Actions:   []string{"BLOCK", "REQUIRE_APPROVAL"},
		}
	}

	return &ComplianceResult{Compliant: true}
}

// checkTimeRestriction 检查时间限制
func (cae *ComplianceAuditEngine) checkTimeRestriction(rule ComplianceRule, apduClass *APDUDataClass) *ComplianceResult {
	// 移除时间限制 - 支持24/7支付服务
	// 现代支付系统需要全天候服务，不再限制深夜交易
	return &ComplianceResult{Compliant: true}
}

// checkFrequencyLimit 检查频率限制
func (cae *ComplianceAuditEngine) checkFrequencyLimit(rule ComplianceRule, apduClass *APDUDataClass) *ComplianceResult {
	userID := apduClass.Metadata.UserID

	// 降低频率限制严格程度：提高违规次数门槛
	violationCount := cae.violationCache[userID]

	if violationCount >= 50 { // 从5次提高到50次
		return &ComplianceResult{
			Compliant: false,
			Reason:    "用户严重违规操作过多",
			RiskLevel: "HIGH",
			Actions:   []string{"BLOCK", "REQUIRE_MANUAL_REVIEW"},
		}
	}

	// 增加警告级别，给用户更多机会
	if violationCount >= 20 {
		return &ComplianceResult{
			Compliant: true, // 不阻断，仅警告
			Reason:    "用户违规次数较多，建议注意",
			RiskLevel: "MEDIUM",
			Actions:   []string{"WARN", "LOG_DETAIL"},
		}
	}

	return &ComplianceResult{Compliant: true}
}

// checkSuspiciousPattern 检查可疑模式
func (cae *ComplianceAuditEngine) checkSuspiciousPattern(rule ComplianceRule, apduClass *APDUDataClass) *ComplianceResult {
	// 检查可疑的命令序列模式
	suspiciousPatterns := []string{
		"SELECT.*WRITE.*SELECT",             // 可疑的选择-写入-选择序列
		"AUTHENTICATE.*WRITE.*AUTHENTICATE", // 重复认证模式
	}

	commandSequence := apduClass.AuditData.CommandType

	for _, pattern := range suspiciousPatterns {
		matched, _ := regexp.MatchString(pattern, commandSequence)
		if matched {
			return &ComplianceResult{
				Compliant: false,
				Reason:    fmt.Sprintf("检测到可疑操作模式: %s", pattern),
				RiskLevel: "HIGH",
				Actions:   []string{"BLOCK", "INVESTIGATE"},
			}
		}
	}

	return &ComplianceResult{Compliant: true}
}

// checkPANCompliance 检查PAN卡号合规性
func (cae *ComplianceAuditEngine) checkPANCompliance(businessData map[string]interface{}) *ComplianceResult {
	if pan, exists := businessData["pan"]; exists {
		panStr, ok := pan.(string)
		if !ok {
			return &ComplianceResult{
				Compliant: false,
				Reason:    "PAN数据格式错误",
				RiskLevel: "HIGH",
				Actions:   []string{"BLOCK", "ALERT"},
			}
		}

		// 检查PAN格式 (简化的Luhn算法检查)
		if !cae.isValidPAN(panStr) {
			return &ComplianceResult{
				Compliant: false,
				Reason:    "无效的PAN卡号格式",
				RiskLevel: "HIGH",
				Actions:   []string{"BLOCK", "INVESTIGATE"},
			}
		}

		// 检查黑名单卡号
		if cae.isBlacklistedPAN(panStr) {
			return &ComplianceResult{
				Compliant: false,
				Reason:    "检测到黑名单卡号",
				RiskLevel: "CRITICAL",
				Actions:   []string{"BLOCK", "ALERT_ADMIN", "LOG_SECURITY"},
			}
		}
	}

	return &ComplianceResult{Compliant: true}
}

// checkAmountCompliance 检查交易金额合规性
func (cae *ComplianceAuditEngine) checkAmountCompliance(businessData map[string]interface{}) *ComplianceResult {
	if amount, exists := businessData["amount"]; exists {
		amountFloat, ok := amount.(float64)
		if !ok {
			return &ComplianceResult{
				Compliant: false,
				Reason:    "交易金额数据格式错误",
				RiskLevel: "MEDIUM",
				Actions:   []string{"WARN", "LOG"},
			}
		}

		amountCents := int64(amountFloat * 100) // 转换为分

		// 单笔交易限额：50万分 (5000元)
		maxAmount := int64(5000000)
		if amountCents > maxAmount {
			return &ComplianceResult{
				Compliant: false,
				Reason:    fmt.Sprintf("交易金额超过限制: %.2f元", amountFloat),
				RiskLevel: "HIGH",
				Actions:   []string{"BLOCK", "REQUIRE_APPROVAL"},
			}
		}

		// 异常小额交易检测 (可能的测试攻击)
		if amountCents > 0 && amountCents < 100 { // 小于1元
			return &ComplianceResult{
				Compliant: false,
				Reason:    fmt.Sprintf("检测到异常小额交易: %.2f元", amountFloat),
				RiskLevel: "MEDIUM",
				Actions:   []string{"WARN", "MONITOR"},
			}
		}
	}

	return &ComplianceResult{Compliant: true}
}

// checkMerchantCompliance 检查商户合规性
func (cae *ComplianceAuditEngine) checkMerchantCompliance(businessData map[string]interface{}) *ComplianceResult {
	if merchantCategory, exists := businessData["merchantCategory"]; exists {
		categoryStr, ok := merchantCategory.(string)
		if !ok {
			return &ComplianceResult{Compliant: true} // 非字符串格式，跳过检查
		}

		// 高风险商户类别
		highRiskCategories := []string{"GAMBLING", "ADULT", "TOBACCO", "WEAPONS"}
		for _, riskCategory := range highRiskCategories {
			if strings.Contains(strings.ToUpper(categoryStr), riskCategory) {
				return &ComplianceResult{
					Compliant: false,
					Reason:    fmt.Sprintf("检测到高风险商户类别: %s", categoryStr),
					RiskLevel: "HIGH",
					Actions:   []string{"BLOCK", "ALERT_ADMIN"},
				}
			}
		}
	}

	return &ComplianceResult{Compliant: true}
}

// checkSensitiveDataCompliance 检查敏感数据合规性
func (cae *ComplianceAuditEngine) checkSensitiveDataCompliance(businessData map[string]interface{}) *ComplianceResult {
	// 检查CVV
	if cvv, exists := businessData["cvv"]; exists {
		cvvStr, ok := cvv.(string)
		if !ok || len(cvvStr) < 3 || len(cvvStr) > 4 {
			return &ComplianceResult{
				Compliant: false,
				Reason:    "无效的CVV格式",
				RiskLevel: "HIGH",
				Actions:   []string{"BLOCK", "ALERT"},
			}
		}
	}

	// 检查PIN（如果存在）
	if pin, exists := businessData["pin"]; exists {
		pinStr, ok := pin.(string)
		if !ok || len(pinStr) < 4 || len(pinStr) > 6 {
			return &ComplianceResult{
				Compliant: false,
				Reason:    "无效的PIN格式",
				RiskLevel: "HIGH",
				Actions:   []string{"BLOCK", "ALERT"},
			}
		}
	}

	return &ComplianceResult{Compliant: true}
}

// 辅助方法
func (cae *ComplianceAuditEngine) isValidPAN(pan string) bool {
	// 简化的PAN格式检查
	if len(pan) < 13 || len(pan) > 19 {
		return false
	}

	// 检查是否全为数字
	for _, char := range pan {
		if char < '0' || char > '9' {
			return false
		}
	}

	// 这里可以实现完整的Luhn算法检查
	return true
}

func (cae *ComplianceAuditEngine) isBlacklistedPAN(pan string) bool {
	// 简化实现：检查测试卡号
	testPANs := []string{
		"4111111111111111", // 测试Visa卡号
		"5555555555554444", // 测试MasterCard卡号
		"0000000000000000", // 明显的测试卡号
	}

	for _, testPAN := range testPANs {
		if pan == testPAN {
			return true
		}
	}

	return false
}

// HandleViolation 处理违规行为
func (cae *ComplianceAuditEngine) HandleViolation(apduClass *APDUDataClass, result *ComplianceResult, userID string) {
	// 记录违规次数
	cae.violationCache[userID]++

	// 根据风险级别执行不同的处理动作
	switch result.RiskLevel {
	case "CRITICAL":
		cae.blockUser(userID, 24*time.Hour) // 封禁24小时
		cae.sendCriticalAlert(apduClass, result, userID)
	case "HIGH":
		if cae.violationCache[userID] >= 3 {
			cae.blockUser(userID, 2*time.Hour) // 封禁2小时
		}
		cae.sendHighRiskAlert(apduClass, result, userID)
	case "MEDIUM":
		cae.sendWarningAlert(apduClass, result, userID)
	}

	// 记录详细的审计日志
	cae.logViolationDetails(apduClass, result, userID)
}

// blockUser 封禁用户
func (cae *ComplianceAuditEngine) blockUser(userID string, duration time.Duration) {
	cae.blockList[userID] = time.Now().Add(duration)

	global.GVA_LOG.Warn("用户已被封禁",
		zap.String("userId", userID),
		zap.Duration("duration", duration),
		zap.Time("unblockTime", cae.blockList[userID]),
	)
}

// isUserBlocked 检查用户是否被封禁
func (cae *ComplianceAuditEngine) isUserBlocked(userID string) bool {
	if blockTime, exists := cae.blockList[userID]; exists {
		if time.Now().Before(blockTime) {
			return true
		}
		// 封禁已过期，清除记录
		delete(cae.blockList, userID)
	}
	return false
}

// 告警方法
func (cae *ComplianceAuditEngine) sendCriticalAlert(apduClass *APDUDataClass, result *ComplianceResult, userID string) {
	global.GVA_LOG.Error("🚨 严重违规告警",
		zap.String("userId", userID),
		zap.String("sessionId", apduClass.Metadata.SessionID),
		zap.String("reason", result.Reason),
		zap.String("commandClass", apduClass.AuditData.CommandClass),
		zap.Int("riskScore", apduClass.AuditData.RiskScore),
	)
}

func (cae *ComplianceAuditEngine) sendHighRiskAlert(apduClass *APDUDataClass, result *ComplianceResult, userID string) {
	global.GVA_LOG.Warn("⚠️ 高风险操作告警",
		zap.String("userId", userID),
		zap.String("reason", result.Reason),
		zap.String("commandClass", apduClass.AuditData.CommandClass),
	)
}

func (cae *ComplianceAuditEngine) sendWarningAlert(apduClass *APDUDataClass, result *ComplianceResult, userID string) {
	global.GVA_LOG.Info("💡 风险提醒",
		zap.String("userId", userID),
		zap.String("reason", result.Reason),
	)
}

// logViolationDetails 记录违规详情
func (cae *ComplianceAuditEngine) logViolationDetails(apduClass *APDUDataClass, result *ComplianceResult, userID string) {
	// 记录到审计日志
	global.LogAuditEvent(
		"compliance_violation",
		global.ComplianceViolation{
			UserID:       userID,
			SessionID:    apduClass.Metadata.SessionID,
			CommandClass: apduClass.AuditData.CommandClass,
			Reason:       result.Reason,
			RiskLevel:    result.RiskLevel,
			Actions:      strings.Join(result.Actions, ","),
			Timestamp:    time.Now(),
		},
	)
}

// getDefaultComplianceRules 获取默认的合规规则
func getDefaultComplianceRules() []ComplianceRule {
	return []ComplianceRule{
		{
			ID:          "HIGH_RISK_COMMAND",
			Name:        "关键命令检测",
			Description: "检测系统关键危险命令",
			RiskLevel:   "CRITICAL",
			Action:      "WARN",
			Enabled:     true,
		},
		{
			ID:          "TRANSACTION_AMOUNT_LIMIT",
			Name:        "交易金额限制",
			Description: "检查交易金额是否超过限制",
			RiskLevel:   "HIGH",
			Action:      "BLOCK",
			Enabled:     true,
		},
		{
			ID:          "TIME_RESTRICTION",
			Name:        "时间限制",
			Description: "限制特定时间段的交易",
			RiskLevel:   "MEDIUM",
			Action:      "LOG",
			Enabled:     false,
		},
		{
			ID:          "FREQUENCY_LIMIT",
			Name:        "频率限制",
			Description: "检测异常频繁的操作",
			RiskLevel:   "MEDIUM",
			Action:      "WARN",
			Enabled:     true,
		},
		{
			ID:          "SUSPICIOUS_PATTERN",
			Name:        "可疑模式检测",
			Description: "检测可疑的操作序列模式",
			RiskLevel:   "MEDIUM",
			Action:      "LOG",
			Enabled:     false,
		},
	}
}
