package request

import (
	"time"

	"github.com/flipped-aurora/gin-vue-admin/server/model/common/request"
)

// CreateComplianceRuleRequest 创建合规规则请求
type CreateComplianceRuleRequest struct {
	ID          string                 `json:"id" binding:"required" example:"rule_001"`
	Name        string                 `json:"name" binding:"required" example:"交易金额限制"`
	Description string                 `json:"description" example:"检查单笔交易金额是否超过限制"`
	Category    string                 `json:"category" binding:"required" example:"transaction"`
	Severity    string                 `json:"severity" binding:"required" example:"high"`
	Enabled     bool                   `json:"enabled" example:"true"`
	Conditions  []RuleConditionRequest `json:"conditions" binding:"required"`
	Actions     []RuleActionRequest    `json:"actions" binding:"required"`
	ValidFrom   *time.Time             `json:"valid_from,omitempty"`
	ValidUntil  *time.Time             `json:"valid_until,omitempty"`
	Metadata    map[string]interface{} `json:"metadata,omitempty"`
}

// UpdateComplianceRuleRequest 更新合规规则请求
type UpdateComplianceRuleRequest struct {
	Name        *string                 `json:"name,omitempty" example:"交易金额限制"`
	Description *string                 `json:"description,omitempty" example:"检查单笔交易金额是否超过限制"`
	Category    *string                 `json:"category,omitempty" example:"transaction"`
	Severity    *string                 `json:"severity,omitempty" example:"high"`
	Enabled     *bool                   `json:"enabled,omitempty" example:"true"`
	Conditions  *[]RuleConditionRequest `json:"conditions,omitempty"`
	Actions     *[]RuleActionRequest    `json:"actions,omitempty"`
	ValidFrom   *time.Time              `json:"valid_from,omitempty"`
	ValidUntil  *time.Time              `json:"valid_until,omitempty"`
	Metadata    *map[string]interface{} `json:"metadata,omitempty"`
}

// RuleConditionRequest 规则条件请求
type RuleConditionRequest struct {
	Field    string      `json:"field" binding:"required" example:"amount"`
	Operator string      `json:"operator" binding:"required" example:"greater_than"`
	Value    interface{} `json:"value" binding:"required" example:"100000"`
	LogicOp  string      `json:"logic_op,omitempty" example:"AND"`
}

// RuleActionRequest 规则动作请求
type RuleActionRequest struct {
	Type       string                 `json:"type" binding:"required" example:"block"`
	Parameters map[string]interface{} `json:"parameters,omitempty"`
}

// ConfigChangeHistoryRequest 配置变更历史查询请求
type ConfigChangeHistoryRequest struct {
	request.PageInfo
	ConfigType string     `json:"config_type,omitempty" form:"config_type" example:"compliance"`
	OperatorID string     `json:"operator_id,omitempty" form:"operator_id" example:"admin"`
	StartTime  *time.Time `json:"start_time,omitempty" form:"start_time"`
	EndTime    *time.Time `json:"end_time,omitempty" form:"end_time"`
	ChangeType string     `json:"change_type,omitempty" form:"change_type" example:"create"`
}

// ReloadConfigRequest 重载配置请求
type ReloadConfigRequest struct {
	ConfigType string `json:"config_type" binding:"required" example:"main"`
	Force      bool   `json:"force" example:"false"`
	Reason     string `json:"reason,omitempty" example:"配置更新"`
}

// ValidateConfigRequest 验证配置请求
type ValidateConfigRequest struct {
	ConfigType string `json:"config_type,omitempty" example:"compliance"`
	Deep       bool   `json:"deep" example:"true"`
}

// BatchUpdateComplianceRulesRequest 批量更新合规规则请求
type BatchUpdateComplianceRulesRequest struct {
	Operations []ComplianceRuleOperation `json:"operations" binding:"required"`
	Reason     string                    `json:"reason,omitempty" example:"批量更新规则"`
}

// ComplianceRuleOperation 合规规则操作
type ComplianceRuleOperation struct {
	Type   string                       `json:"type" binding:"required" example:"update"` // create, update, delete
	RuleID string                       `json:"rule_id" binding:"required" example:"rule_001"`
	Rule   *CreateComplianceRuleRequest `json:"rule,omitempty"`
}

// GetComplianceRuleRequest 获取合规规则请求
type GetComplianceRuleRequest struct {
	RuleID string `json:"rule_id" uri:"rule_id" binding:"required" example:"rule_001"`
}

// DeleteComplianceRuleRequest 删除合规规则请求
type DeleteComplianceRuleRequest struct {
	RuleID string `json:"rule_id" uri:"rule_id" binding:"required" example:"rule_001"`
	Reason string `json:"reason,omitempty" example:"规则已过期"`
}

// GetComplianceRulesRequest 获取合规规则列表请求
type GetComplianceRulesRequest struct {
	request.PageInfo
	Category string `json:"category,omitempty" form:"category" example:"transaction"`
	Severity string `json:"severity,omitempty" form:"severity" example:"high"`
	Enabled  *bool  `json:"enabled,omitempty" form:"enabled" example:"true"`
	Search   string `json:"search,omitempty" form:"search" example:"交易"`
}

// ConfigBackupRequest 配置备份请求
type ConfigBackupRequest struct {
	ConfigType string `json:"config_type" binding:"required" example:"compliance"`
	BackupName string `json:"backup_name,omitempty" example:"backup_20231201"`
	Reason     string `json:"reason,omitempty" example:"定期备份"`
}

// ConfigRestoreRequest 配置恢复请求
type ConfigRestoreRequest struct {
	ConfigType string `json:"config_type" binding:"required" example:"compliance"`
	BackupPath string `json:"backup_path" binding:"required" example:"./config_backups/compliance_rules.json.20231201_120000.bak"`
	Reason     string `json:"reason,omitempty" example:"恢复到之前版本"`
}
