package config

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/fsnotify/fsnotify"
	"github.com/spf13/viper"
	"go.uber.org/zap"
)

// ConfigManager 配置管理器
type ConfigManager struct {
	watcher          *fsnotify.Watcher
	configPaths      []string
	complianceRules  *ComplianceRules
	changeHandlers   []ConfigChangeHandler
	auditLogger      *zap.Logger
	mu               sync.RWMutex
	enabled          bool
	lastChangeTime   time.Time
	debounceInterval time.Duration
	backupDir        string
}

// ConfigChangeHandler 配置变更处理器类型
type ConfigChangeHandler func(configType ConfigType, oldValue, newValue interface{}) error

// ConfigType 配置类型枚举
type ConfigType string

const (
	ConfigTypeMain       ConfigType = "main"
	ConfigTypeCompliance ConfigType = "compliance"
	ConfigTypeNfcRelay   ConfigType = "nfc_relay"
	ConfigTypeSecurity   ConfigType = "security"
)

// ConfigChangeEvent 配置变更事件
type ConfigChangeEvent struct {
	Type             ConfigType       `json:"type"`
	Path             string           `json:"path"`
	OldValue         interface{}      `json:"old_value,omitempty"`
	NewValue         interface{}      `json:"new_value"`
	ChangeTime       time.Time        `json:"change_time"`
	OperatorID       string           `json:"operator_id,omitempty"`
	ChangeReason     string           `json:"change_reason,omitempty"`
	ValidationResult ValidationResult `json:"validation_result"`
}

// ValidationResult 验证结果
type ValidationResult struct {
	IsValid    bool     `json:"is_valid"`
	Errors     []string `json:"errors,omitempty"`
	Warnings   []string `json:"warnings,omitempty"`
	BackupPath string   `json:"backup_path,omitempty"`
}

// ComplianceRules 合规规则配置
type ComplianceRules struct {
	Version             string                    `json:"version" yaml:"version"`
	Rules               map[string]ComplianceRule `json:"rules" yaml:"rules"`
	GlobalSettings      GlobalComplianceSettings  `json:"global_settings" yaml:"global_settings"`
	LastModified        time.Time                 `json:"last_modified" yaml:"last_modified"`
	RuleValidationRules map[string]interface{}    `json:"rule_validation" yaml:"rule_validation"`
}

// ComplianceRule 单个合规规则
type ComplianceRule struct {
	ID           string                 `json:"id" yaml:"id"`
	Name         string                 `json:"name" yaml:"name"`
	Description  string                 `json:"description" yaml:"description"`
	Category     string                 `json:"category" yaml:"category"`
	Severity     string                 `json:"severity" yaml:"severity"`
	Enabled      bool                   `json:"enabled" yaml:"enabled"`
	Conditions   []RuleCondition        `json:"conditions" yaml:"conditions"`
	Actions      []RuleAction           `json:"actions" yaml:"actions"`
	CreatedTime  time.Time              `json:"created_time" yaml:"created_time"`
	LastModified time.Time              `json:"last_modified" yaml:"last_modified"`
	ValidFrom    *time.Time             `json:"valid_from,omitempty" yaml:"valid_from,omitempty"`
	ValidUntil   *time.Time             `json:"valid_until,omitempty" yaml:"valid_until,omitempty"`
	Metadata     map[string]interface{} `json:"metadata,omitempty" yaml:"metadata,omitempty"`
}

// RuleCondition 规则条件
type RuleCondition struct {
	Field    string      `json:"field" yaml:"field"`
	Operator string      `json:"operator" yaml:"operator"`
	Value    interface{} `json:"value" yaml:"value"`
	LogicOp  string      `json:"logic_op,omitempty" yaml:"logic_op,omitempty"` // AND, OR
}

// RuleAction 规则动作
type RuleAction struct {
	Type       string                 `json:"type" yaml:"type"`
	Parameters map[string]interface{} `json:"parameters,omitempty" yaml:"parameters,omitempty"`
}

// GlobalComplianceSettings 全局合规设置
type GlobalComplianceSettings struct {
	EnableAuditLogging      bool  `json:"enable_audit_logging" yaml:"enable_audit_logging"`
	MaxTransactionAmount    int64 `json:"max_transaction_amount" yaml:"max_transaction_amount"`
	RequireDeepInspection   bool  `json:"require_deep_inspection" yaml:"require_deep_inspection"`
	BlockSuspiciousActivity bool  `json:"block_suspicious_activity" yaml:"block_suspicious_activity"`
	RetentionDays           int   `json:"retention_days" yaml:"retention_days"`
}

// NewConfigManager 创建新的配置管理器
func NewConfigManager(auditLogger *zap.Logger, backupDir string) *ConfigManager {
	if backupDir == "" {
		backupDir = "./config_backups"
	}

	// 确保备份目录存在
	if err := os.MkdirAll(backupDir, 0755); err != nil {
		auditLogger.Error("创建配置备份目录失败", zap.String("dir", backupDir), zap.Error(err))
	}

	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		auditLogger.Error("创建文件监视器失败", zap.Error(err))
		return nil
	}

	return &ConfigManager{
		watcher:          watcher,
		configPaths:      make([]string, 0),
		complianceRules:  &ComplianceRules{Rules: make(map[string]ComplianceRule)},
		changeHandlers:   make([]ConfigChangeHandler, 0),
		auditLogger:      auditLogger,
		debounceInterval: 1 * time.Second,
		backupDir:        backupDir,
		enabled:          true,
	}
}

// Start 启动配置管理器
func (cm *ConfigManager) Start() error {
	cm.mu.Lock()
	defer cm.mu.Unlock()

	if !cm.enabled {
		return fmt.Errorf("配置管理器已禁用")
	}

	// 启动文件监视协程
	go cm.watchConfigFiles()

	cm.auditLogger.Info("配置管理器已启动")
	return nil
}

// Stop 停止配置管理器
func (cm *ConfigManager) Stop() error {
	cm.mu.Lock()
	defer cm.mu.Unlock()

	cm.enabled = false
	if cm.watcher != nil {
		err := cm.watcher.Close()
		if err != nil {
			cm.auditLogger.Error("关闭文件监视器失败", zap.Error(err))
			return err
		}
	}

	cm.auditLogger.Info("配置管理器已停止")
	return nil
}

// AddConfigPath 添加配置文件路径到监视列表
func (cm *ConfigManager) AddConfigPath(path string) error {
	cm.mu.Lock()
	defer cm.mu.Unlock()

	absPath, err := filepath.Abs(path)
	if err != nil {
		return fmt.Errorf("获取绝对路径失败: %w", err)
	}

	// 检查文件是否存在
	if _, err := os.Stat(absPath); os.IsNotExist(err) {
		return fmt.Errorf("配置文件不存在: %s", absPath)
	}

	// 添加到监视器
	if err := cm.watcher.Add(absPath); err != nil {
		return fmt.Errorf("添加文件监视失败: %w", err)
	}

	cm.configPaths = append(cm.configPaths, absPath)
	cm.auditLogger.Info("已添加配置文件到监视列表", zap.String("path", absPath))

	return nil
}

// RegisterChangeHandler 注册配置变更处理器
func (cm *ConfigManager) RegisterChangeHandler(handler ConfigChangeHandler) {
	cm.mu.Lock()
	defer cm.mu.Unlock()

	cm.changeHandlers = append(cm.changeHandlers, handler)
	cm.auditLogger.Info("已注册配置变更处理器")
}

// LoadComplianceRules 加载合规规则
func (cm *ConfigManager) LoadComplianceRules(path string) error {
	cm.mu.Lock()
	defer cm.mu.Unlock()

	data, err := os.ReadFile(path)
	if err != nil {
		return fmt.Errorf("读取合规规则文件失败: %w", err)
	}

	var rules ComplianceRules
	if err := json.Unmarshal(data, &rules); err != nil {
		return fmt.Errorf("解析合规规则失败: %w", err)
	}

	// 验证规则
	if err := cm.validateComplianceRules(&rules); err != nil {
		return fmt.Errorf("合规规则验证失败: %w", err)
	}

	oldRules := cm.complianceRules
	cm.complianceRules = &rules

	// 记录变更审计
	cm.logConfigChange(ConfigChangeEvent{
		Type:       ConfigTypeCompliance,
		Path:       path,
		OldValue:   oldRules,
		NewValue:   &rules,
		ChangeTime: time.Now(),
		ValidationResult: ValidationResult{
			IsValid: true,
		},
	})

	cm.auditLogger.Info("合规规则加载成功", zap.String("path", path), zap.Int("rule_count", len(rules.Rules)))
	return nil
}

// SaveComplianceRules 保存合规规则
func (cm *ConfigManager) SaveComplianceRules(path string) error {
	cm.mu.RLock()
	rules := cm.complianceRules
	cm.mu.RUnlock()

	// 创建备份
	if err := cm.createBackup(path); err != nil {
		cm.auditLogger.Warn("创建配置备份失败", zap.String("path", path), zap.Error(err))
	}

	rules.LastModified = time.Now()

	data, err := json.MarshalIndent(rules, "", "  ")
	if err != nil {
		return fmt.Errorf("序列化合规规则失败: %w", err)
	}

	if err := os.WriteFile(path, data, 0644); err != nil {
		return fmt.Errorf("保存合规规则文件失败: %w", err)
	}

	cm.auditLogger.Info("合规规则保存成功", zap.String("path", path))
	return nil
}

// UpdateComplianceRule 更新单个合规规则
func (cm *ConfigManager) UpdateComplianceRule(ruleID string, rule ComplianceRule) error {
	cm.mu.Lock()
	defer cm.mu.Unlock()

	oldRule, exists := cm.complianceRules.Rules[ruleID]
	rule.LastModified = time.Now()
	if !exists {
		rule.CreatedTime = time.Now()
	}

	// 验证规则
	if err := cm.validateSingleRule(&rule); err != nil {
		return fmt.Errorf("规则验证失败: %w", err)
	}

	cm.complianceRules.Rules[ruleID] = rule

	// 记录变更审计
	changeEvent := ConfigChangeEvent{
		Type:       ConfigTypeCompliance,
		Path:       fmt.Sprintf("rules.%s", ruleID),
		NewValue:   rule,
		ChangeTime: time.Now(),
		ValidationResult: ValidationResult{
			IsValid: true,
		},
	}

	if exists {
		changeEvent.OldValue = oldRule
	}

	cm.logConfigChange(changeEvent)

	cm.auditLogger.Info("合规规则更新成功", zap.String("rule_id", ruleID), zap.String("rule_name", rule.Name))
	return nil
}

// DeleteComplianceRule 删除合规规则
func (cm *ConfigManager) DeleteComplianceRule(ruleID string) error {
	cm.mu.Lock()
	defer cm.mu.Unlock()

	oldRule, exists := cm.complianceRules.Rules[ruleID]
	if !exists {
		return fmt.Errorf("规则不存在: %s", ruleID)
	}

	delete(cm.complianceRules.Rules, ruleID)

	// 记录变更审计
	cm.logConfigChange(ConfigChangeEvent{
		Type:       ConfigTypeCompliance,
		Path:       fmt.Sprintf("rules.%s", ruleID),
		OldValue:   oldRule,
		ChangeTime: time.Now(),
		ValidationResult: ValidationResult{
			IsValid: true,
		},
	})

	cm.auditLogger.Info("合规规则删除成功", zap.String("rule_id", ruleID))
	return nil
}

// GetComplianceRules 获取所有合规规则
func (cm *ConfigManager) GetComplianceRules() *ComplianceRules {
	cm.mu.RLock()
	defer cm.mu.RUnlock()

	// 深拷贝避免并发修改
	rulesCopy := &ComplianceRules{
		Version:        cm.complianceRules.Version,
		Rules:          make(map[string]ComplianceRule),
		GlobalSettings: cm.complianceRules.GlobalSettings,
		LastModified:   cm.complianceRules.LastModified,
	}

	for k, v := range cm.complianceRules.Rules {
		rulesCopy.Rules[k] = v
	}

	return rulesCopy
}

// GetComplianceRule 获取单个合规规则
func (cm *ConfigManager) GetComplianceRule(ruleID string) (ComplianceRule, bool) {
	cm.mu.RLock()
	defer cm.mu.RUnlock()

	rule, exists := cm.complianceRules.Rules[ruleID]
	return rule, exists
}

// ReloadConfig 手动重载配置
func (cm *ConfigManager) ReloadConfig(configPath string, v *viper.Viper) error {
	cm.mu.Lock()
	defer cm.mu.Unlock()

	// 创建备份
	if err := cm.createBackup(configPath); err != nil {
		cm.auditLogger.Warn("创建配置备份失败", zap.String("path", configPath), zap.Error(err))
	}

	oldConfig := make(map[string]interface{})
	for _, key := range v.AllKeys() {
		oldConfig[key] = v.Get(key)
	}

	// 重新读取配置
	if err := v.ReadInConfig(); err != nil {
		return fmt.Errorf("重新读取配置文件失败: %w", err)
	}

	newConfig := make(map[string]interface{})
	for _, key := range v.AllKeys() {
		newConfig[key] = v.Get(key)
	}

	// 触发变更处理器
	for _, handler := range cm.changeHandlers {
		if err := handler(ConfigTypeMain, oldConfig, newConfig); err != nil {
			cm.auditLogger.Error("配置变更处理器执行失败", zap.Error(err))
		}
	}

	// 记录变更审计
	cm.logConfigChange(ConfigChangeEvent{
		Type:       ConfigTypeMain,
		Path:       configPath,
		OldValue:   oldConfig,
		NewValue:   newConfig,
		ChangeTime: time.Now(),
		ValidationResult: ValidationResult{
			IsValid: true,
		},
	})

	cm.auditLogger.Info("配置重载成功", zap.String("path", configPath))
	return nil
}

// watchConfigFiles 监视配置文件变更
func (cm *ConfigManager) watchConfigFiles() {
	for {
		select {
		case event, ok := <-cm.watcher.Events:
			if !ok {
				return
			}

			// 防抖处理
			now := time.Now()
			if now.Sub(cm.lastChangeTime) < cm.debounceInterval {
				continue
			}
			cm.lastChangeTime = now

			if event.Op&fsnotify.Write == fsnotify.Write {
				cm.handleConfigFileChange(event.Name)
			}

		case err, ok := <-cm.watcher.Errors:
			if !ok {
				return
			}
			cm.auditLogger.Error("文件监视器错误", zap.Error(err))
		}
	}
}

// handleConfigFileChange 处理配置文件变更
func (cm *ConfigManager) handleConfigFileChange(path string) {
	cm.auditLogger.Info("检测到配置文件变更", zap.String("path", path))

	// 根据文件路径确定配置类型
	configType := cm.determineConfigType(path)

	// 触发热重载
	switch configType {
	case ConfigTypeCompliance:
		if err := cm.LoadComplianceRules(path); err != nil {
			cm.auditLogger.Error("合规规则热重载失败", zap.String("path", path), zap.Error(err))
		}
	default:
		// 对于主配置文件，通知外部处理器
		for _, handler := range cm.changeHandlers {
			if err := handler(configType, nil, nil); err != nil {
				cm.auditLogger.Error("配置变更处理器执行失败", zap.Error(err))
			}
		}
	}
}

// determineConfigType 确定配置文件类型
func (cm *ConfigManager) determineConfigType(path string) ConfigType {
	filename := filepath.Base(path)
	switch {
	case filename == "compliance_rules.json" || filepath.Dir(path) == "compliance":
		return ConfigTypeCompliance
	case filename == "nfc_relay.yaml" || filename == "nfc_relay.yml":
		return ConfigTypeNfcRelay
	case filename == "security.yaml" || filename == "security.yml":
		return ConfigTypeSecurity
	default:
		return ConfigTypeMain
	}
}

// createBackup 创建配置文件备份
func (cm *ConfigManager) createBackup(configPath string) error {
	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		return nil // 文件不存在，无需备份
	}

	data, err := os.ReadFile(configPath)
	if err != nil {
		return err
	}

	timestamp := time.Now().Format("20060102_150405")
	filename := filepath.Base(configPath)
	backupPath := filepath.Join(cm.backupDir, fmt.Sprintf("%s.%s.bak", filename, timestamp))

	return os.WriteFile(backupPath, data, 0644)
}

// validateComplianceRules 验证合规规则
func (cm *ConfigManager) validateComplianceRules(rules *ComplianceRules) error {
	if rules.Version == "" {
		return fmt.Errorf("合规规则版本不能为空")
	}

	for ruleID, rule := range rules.Rules {
		if err := cm.validateSingleRule(&rule); err != nil {
			return fmt.Errorf("规则 %s 验证失败: %w", ruleID, err)
		}
	}

	return nil
}

// validateSingleRule 验证单个规则
func (cm *ConfigManager) validateSingleRule(rule *ComplianceRule) error {
	if rule.ID == "" {
		return fmt.Errorf("规则ID不能为空")
	}
	if rule.Name == "" {
		return fmt.Errorf("规则名称不能为空")
	}
	if len(rule.Conditions) == 0 {
		return fmt.Errorf("规则至少需要一个条件")
	}
	if len(rule.Actions) == 0 {
		return fmt.Errorf("规则至少需要一个动作")
	}

	// 验证条件
	for i, condition := range rule.Conditions {
		if condition.Field == "" {
			return fmt.Errorf("条件 %d 的字段不能为空", i)
		}
		if condition.Operator == "" {
			return fmt.Errorf("条件 %d 的操作符不能为空", i)
		}
	}

	// 验证动作
	for i, action := range rule.Actions {
		if action.Type == "" {
			return fmt.Errorf("动作 %d 的类型不能为空", i)
		}
	}

	return nil
}

// logConfigChange 记录配置变更审计日志
func (cm *ConfigManager) logConfigChange(event ConfigChangeEvent) {
	auditDetails := map[string]interface{}{
		"config_change_event": event,
		"change_type":         string(event.Type),
		"configuration_path":  event.Path,
		"validation_passed":   event.ValidationResult.IsValid,
		"backup_created":      event.ValidationResult.BackupPath != "",
		"operator_id":         event.OperatorID,
		"change_reason":       event.ChangeReason,
	}

	cm.auditLogger.Info("配置变更记录",
		zap.String("event_type", "CONFIG_CHANGE"),
		zap.String("config_type", string(event.Type)),
		zap.String("config_path", event.Path),
		zap.Bool("validation_passed", event.ValidationResult.IsValid),
		zap.Any("details", auditDetails),
	)
}
