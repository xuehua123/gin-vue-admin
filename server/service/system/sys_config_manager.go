package system

import (
	"fmt"
	"time"

	"github.com/flipped-aurora/gin-vue-admin/server/config"
	"github.com/flipped-aurora/gin-vue-admin/server/global"
	"github.com/flipped-aurora/gin-vue-admin/server/model/common/request"
	"github.com/flipped-aurora/gin-vue-admin/server/model/system"
	systemReq "github.com/flipped-aurora/gin-vue-admin/server/model/system/request"
	"go.uber.org/zap"
)

// ConfigManagerService 配置管理服务
type ConfigManagerService struct {
	configManager *config.ConfigManager
	auditLogger   *zap.Logger
}

// NewConfigManagerService 创建配置管理服务
func NewConfigManagerService() *ConfigManagerService {
	// 初始化审计日志器 - 安全检查
	var auditLogger *zap.Logger
	if global.GVA_LOG != nil {
		auditLogger = global.GVA_LOG.Named("config_audit")
	} else {
		// 如果全局日志未初始化，使用默认日志器
		auditLogger, _ = zap.NewDevelopment()
		auditLogger = auditLogger.Named("config_audit")
	}

	// 创建配置管理器
	configManager := config.NewConfigManager(auditLogger, "./config_backups")
	if configManager == nil {
		if global.GVA_LOG != nil {
			global.GVA_LOG.Error("创建配置管理器失败")
		}
		return nil
	}

	service := &ConfigManagerService{
		configManager: configManager,
		auditLogger:   auditLogger,
	}

	// 注册配置变更处理器
	configManager.RegisterChangeHandler(service.handleConfigChange)

	return service
}

// InitializeConfigManager 初始化配置管理器
func (configManagerService *ConfigManagerService) InitializeConfigManager() error {
	if configManagerService.configManager == nil {
		return fmt.Errorf("配置管理器未初始化")
	}

	// 添加主配置文件到监视列表
	mainConfigPath := global.GVA_VP.ConfigFileUsed()
	if mainConfigPath != "" {
		if err := configManagerService.configManager.AddConfigPath(mainConfigPath); err != nil {
			configManagerService.auditLogger.Error("添加主配置文件监视失败", zap.String("path", mainConfigPath), zap.Error(err))
		}
	}

	// 添加合规规则文件到监视列表
	complianceRulesPath := "./config/compliance_rules.json"
	if err := configManagerService.configManager.AddConfigPath(complianceRulesPath); err != nil {
		configManagerService.auditLogger.Warn("添加合规规则文件监视失败", zap.String("path", complianceRulesPath), zap.Error(err))
		// 创建默认合规规则文件
		if err := configManagerService.createDefaultComplianceRules(complianceRulesPath); err != nil {
			configManagerService.auditLogger.Error("创建默认合规规则文件失败", zap.Error(err))
		}
	}

	// 启动配置管理器
	if err := configManagerService.configManager.Start(); err != nil {
		return fmt.Errorf("启动配置管理器失败: %w", err)
	}

	configManagerService.auditLogger.Info("配置管理器初始化完成")
	return nil
}

// StopConfigManager 停止配置管理器
func (configManagerService *ConfigManagerService) StopConfigManager() error {
	if configManagerService.configManager == nil {
		return nil
	}

	return configManagerService.configManager.Stop()
}

// ReloadMainConfig 重载主配置
func (configManagerService *ConfigManagerService) ReloadMainConfig() error {
	if configManagerService.configManager == nil {
		return fmt.Errorf("配置管理器未初始化")
	}

	configPath := global.GVA_VP.ConfigFileUsed()
	if configPath == "" {
		return fmt.Errorf("无法获取配置文件路径")
	}

	return configManagerService.configManager.ReloadConfig(configPath, global.GVA_VP)
}

// GetComplianceRules 获取所有合规规则
func (configManagerService *ConfigManagerService) GetComplianceRules() (*config.ComplianceRules, error) {
	if configManagerService.configManager == nil {
		return nil, fmt.Errorf("配置管理器未初始化")
	}

	return configManagerService.configManager.GetComplianceRules(), nil
}

// GetComplianceRule 获取单个合规规则
func (configManagerService *ConfigManagerService) GetComplianceRule(ruleID string) (*config.ComplianceRule, error) {
	if configManagerService.configManager == nil {
		return nil, fmt.Errorf("配置管理器未初始化")
	}

	rule, exists := configManagerService.configManager.GetComplianceRule(ruleID)
	if !exists {
		return nil, fmt.Errorf("合规规则不存在: %s", ruleID)
	}

	return &rule, nil
}

// CreateComplianceRule 创建合规规则
func (configManagerService *ConfigManagerService) CreateComplianceRule(req systemReq.CreateComplianceRuleRequest) error {
	if configManagerService.configManager == nil {
		return fmt.Errorf("配置管理器未初始化")
	}

	rule := config.ComplianceRule{
		ID:           req.ID,
		Name:         req.Name,
		Description:  req.Description,
		Category:     req.Category,
		Severity:     req.Severity,
		Enabled:      req.Enabled,
		Conditions:   convertToRuleConditions(req.Conditions),
		Actions:      convertToRuleActions(req.Actions),
		CreatedTime:  time.Now(),
		LastModified: time.Now(),
		Metadata:     req.Metadata,
	}

	// 设置有效期
	if req.ValidFrom != nil {
		rule.ValidFrom = req.ValidFrom
	}
	if req.ValidUntil != nil {
		rule.ValidUntil = req.ValidUntil
	}

	return configManagerService.configManager.UpdateComplianceRule(req.ID, rule)
}

// UpdateComplianceRule 更新合规规则
func (configManagerService *ConfigManagerService) UpdateComplianceRule(ruleID string, req systemReq.UpdateComplianceRuleRequest) error {
	if configManagerService.configManager == nil {
		return fmt.Errorf("配置管理器未初始化")
	}

	// 获取现有规则
	existingRule, exists := configManagerService.configManager.GetComplianceRule(ruleID)
	if !exists {
		return fmt.Errorf("合规规则不存在: %s", ruleID)
	}

	// 更新规则字段
	if req.Name != nil {
		existingRule.Name = *req.Name
	}
	if req.Description != nil {
		existingRule.Description = *req.Description
	}
	if req.Category != nil {
		existingRule.Category = *req.Category
	}
	if req.Severity != nil {
		existingRule.Severity = *req.Severity
	}
	if req.Enabled != nil {
		existingRule.Enabled = *req.Enabled
	}
	if req.Conditions != nil {
		existingRule.Conditions = convertToRuleConditions(*req.Conditions)
	}
	if req.Actions != nil {
		existingRule.Actions = convertToRuleActions(*req.Actions)
	}
	if req.ValidFrom != nil {
		existingRule.ValidFrom = req.ValidFrom
	}
	if req.ValidUntil != nil {
		existingRule.ValidUntil = req.ValidUntil
	}
	if req.Metadata != nil {
		existingRule.Metadata = *req.Metadata
	}

	existingRule.LastModified = time.Now()

	return configManagerService.configManager.UpdateComplianceRule(ruleID, existingRule)
}

// DeleteComplianceRule 删除合规规则
func (configManagerService *ConfigManagerService) DeleteComplianceRule(ruleID string) error {
	if configManagerService.configManager == nil {
		return fmt.Errorf("配置管理器未初始化")
	}

	return configManagerService.configManager.DeleteComplianceRule(ruleID)
}

// SaveComplianceRules 保存合规规则到文件
func (configManagerService *ConfigManagerService) SaveComplianceRules() error {
	if configManagerService.configManager == nil {
		return fmt.Errorf("配置管理器未初始化")
	}

	complianceRulesPath := "./config/compliance_rules.json"
	return configManagerService.configManager.SaveComplianceRules(complianceRulesPath)
}

// LoadComplianceRules 从文件加载合规规则
func (configManagerService *ConfigManagerService) LoadComplianceRules() error {
	if configManagerService.configManager == nil {
		return fmt.Errorf("配置管理器未初始化")
	}

	complianceRulesPath := "./config/compliance_rules.json"
	return configManagerService.configManager.LoadComplianceRules(complianceRulesPath)
}

// GetComplianceRulesList 获取合规规则列表（分页）
func (configManagerService *ConfigManagerService) GetComplianceRulesList(info request.PageInfo) (list []config.ComplianceRule, total int64, err error) {
	if configManagerService.configManager == nil {
		return nil, 0, fmt.Errorf("配置管理器未初始化")
	}

	rules := configManagerService.configManager.GetComplianceRules()
	if rules == nil {
		return make([]config.ComplianceRule, 0), 0, nil
	}

	// 转换为切片并排序
	ruleList := make([]config.ComplianceRule, 0, len(rules.Rules))
	for _, rule := range rules.Rules {
		ruleList = append(ruleList, rule)
	}

	total = int64(len(ruleList))

	// 简单分页
	startIndex := (info.Page - 1) * info.PageSize
	endIndex := startIndex + info.PageSize

	if startIndex >= len(ruleList) {
		return make([]config.ComplianceRule, 0), total, nil
	}

	if endIndex > len(ruleList) {
		endIndex = len(ruleList)
	}

	return ruleList[startIndex:endIndex], total, nil
}

// GetConfigChangeHistory 获取配置变更历史
func (configManagerService *ConfigManagerService) GetConfigChangeHistory(info systemReq.ConfigChangeHistoryRequest) (list []system.ConfigChangeHistory, total int64, err error) {
	// 这里可以实现从数据库或日志文件中获取配置变更历史
	// 暂时返回空列表
	return make([]system.ConfigChangeHistory, 0), 0, nil
}

// ValidateConfiguration 验证配置
func (configManagerService *ConfigManagerService) ValidateConfiguration() (*config.ValidationResult, error) {
	if configManagerService.configManager == nil {
		return nil, fmt.Errorf("配置管理器未初始化")
	}

	result := &config.ValidationResult{
		IsValid:  true,
		Errors:   make([]string, 0),
		Warnings: make([]string, 0),
	}

	// 验证主配置
	if err := configManagerService.validateMainConfig(); err != nil {
		result.IsValid = false
		result.Errors = append(result.Errors, fmt.Sprintf("主配置验证失败: %v", err))
	}

	// 验证合规规则
	rules := configManagerService.configManager.GetComplianceRules()
	if rules != nil {
		for ruleID, rule := range rules.Rules {
			if !rule.Enabled {
				result.Warnings = append(result.Warnings, fmt.Sprintf("合规规则 %s 已禁用", ruleID))
			}

			// 检查规则有效期
			now := time.Now()
			if rule.ValidUntil != nil && rule.ValidUntil.Before(now) {
				result.Warnings = append(result.Warnings, fmt.Sprintf("合规规则 %s 已过期", ruleID))
			}
		}
	}

	return result, nil
}

// handleConfigChange 处理配置变更
func (configManagerService *ConfigManagerService) handleConfigChange(configType config.ConfigType, oldValue, newValue interface{}) error {
	configManagerService.auditLogger.Info("配置变更处理",
		zap.String("config_type", string(configType)),
		zap.Any("old_value", oldValue),
		zap.Any("new_value", newValue),
	)

	switch configType {
	case config.ConfigTypeMain:
		// 处理主配置变更
		return configManagerService.handleMainConfigChange(oldValue, newValue)
	case config.ConfigTypeCompliance:
		// 处理合规规则变更
		return configManagerService.handleComplianceRulesChange(oldValue, newValue)
	case config.ConfigTypeNfcRelay:
		// 处理NFC中继配置变更
		return configManagerService.handleNfcRelayConfigChange(oldValue, newValue)
	case config.ConfigTypeSecurity:
		// 处理安全配置变更
		return configManagerService.handleSecurityConfigChange(oldValue, newValue)
	default:
		configManagerService.auditLogger.Warn("未知的配置类型", zap.String("config_type", string(configType)))
	}

	return nil
}

// handleMainConfigChange 处理主配置变更
func (configManagerService *ConfigManagerService) handleMainConfigChange(oldValue, newValue interface{}) error {
	// 重新初始化相关组件
	configManagerService.auditLogger.Info("主配置已变更，重新初始化相关组件")

	// 这里可以添加具体的重新初始化逻辑
	// 例如重新初始化数据库连接、Redis连接等

	return nil
}

// handleComplianceRulesChange 处理合规规则变更
func (configManagerService *ConfigManagerService) handleComplianceRulesChange(oldValue, newValue interface{}) error {
	configManagerService.auditLogger.Info("合规规则已变更，更新相关组件")

	// 这里可以添加合规规则变更后的处理逻辑
	// 例如更新合规检查引擎

	return nil
}

// handleNfcRelayConfigChange 处理NFC中继配置变更
func (configManagerService *ConfigManagerService) handleNfcRelayConfigChange(oldValue, newValue interface{}) error {
	configManagerService.auditLogger.Info("NFC中继配置已变更，更新相关组件")

	// 这里可以添加NFC中继配置变更后的处理逻辑

	return nil
}

// handleSecurityConfigChange 处理安全配置变更
func (configManagerService *ConfigManagerService) handleSecurityConfigChange(oldValue, newValue interface{}) error {
	configManagerService.auditLogger.Info("安全配置已变更，更新相关组件")

	// 这里可以添加安全配置变更后的处理逻辑

	return nil
}

// validateMainConfig 验证主配置
func (configManagerService *ConfigManagerService) validateMainConfig() error {
	// 这里可以添加主配置的验证逻辑
	return nil
}

// createDefaultComplianceRules 创建默认合规规则文件
func (configManagerService *ConfigManagerService) createDefaultComplianceRules(path string) error {
	defaultRules := &config.ComplianceRules{
		Version: "1.0.0",
		Rules:   make(map[string]config.ComplianceRule),
		GlobalSettings: config.GlobalComplianceSettings{
			EnableAuditLogging:      true,
			MaxTransactionAmount:    100000, // 10万分（1000元）
			RequireDeepInspection:   true,
			BlockSuspiciousActivity: true,
			RetentionDays:           90,
		},
		LastModified: time.Now(),
	}

	// 添加一些默认规则
	defaultRules.Rules["max_transaction_amount"] = config.ComplianceRule{
		ID:          "max_transaction_amount",
		Name:        "最大交易金额限制",
		Description: "检查交易金额是否超过限制",
		Category:    "transaction",
		Severity:    "high",
		Enabled:     true,
		Conditions: []config.RuleCondition{
			{
				Field:    "amount",
				Operator: "greater_than",
				Value:    100000,
			},
		},
		Actions: []config.RuleAction{
			{
				Type: "block",
				Parameters: map[string]interface{}{
					"reason": "交易金额超过限制",
				},
			},
		},
		CreatedTime:  time.Now(),
		LastModified: time.Now(),
	}

	return configManagerService.configManager.SaveComplianceRules(path)
}

// 辅助函数：转换请求中的条件为配置条件
func convertToRuleConditions(conditions []systemReq.RuleConditionRequest) []config.RuleCondition {
	result := make([]config.RuleCondition, len(conditions))
	for i, condition := range conditions {
		result[i] = config.RuleCondition{
			Field:    condition.Field,
			Operator: condition.Operator,
			Value:    condition.Value,
			LogicOp:  condition.LogicOp,
		}
	}
	return result
}

// 辅助函数：转换请求中的动作为配置动作
func convertToRuleActions(actions []systemReq.RuleActionRequest) []config.RuleAction {
	result := make([]config.RuleAction, len(actions))
	for i, action := range actions {
		result[i] = config.RuleAction{
			Type:       action.Type,
			Parameters: action.Parameters,
		}
	}
	return result
}
