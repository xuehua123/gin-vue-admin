package system

import (
	"time"

	"github.com/flipped-aurora/gin-vue-admin/server/global"
)

// ConfigChangeHistory 配置变更历史
type ConfigChangeHistory struct {
	global.GVA_MODEL
	ConfigType       string    `json:"config_type" gorm:"type:varchar(50);not null;comment:配置类型" example:"compliance"`
	ConfigPath       string    `json:"config_path" gorm:"type:varchar(500);not null;comment:配置路径" example:"rules.rule_001"`
	ChangeType       string    `json:"change_type" gorm:"type:varchar(20);not null;comment:变更类型" example:"create"`
	OperatorID       string    `json:"operator_id" gorm:"type:varchar(100);comment:操作者ID" example:"admin"`
	OperatorName     string    `json:"operator_name" gorm:"type:varchar(100);comment:操作者名称" example:"管理员"`
	ChangeReason     string    `json:"change_reason" gorm:"type:text;comment:变更原因" example:"新增交易限制规则"`
	OldValue         string    `json:"old_value" gorm:"type:longtext;comment:变更前值"`
	NewValue         string    `json:"new_value" gorm:"type:longtext;comment:变更后值"`
	ValidationPassed bool      `json:"validation_passed" gorm:"type:boolean;default:true;comment:验证是否通过" example:"true"`
	ValidationErrors string    `json:"validation_errors" gorm:"type:text;comment:验证错误信息"`
	BackupPath       string    `json:"backup_path" gorm:"type:varchar(500);comment:备份文件路径"`
	ChangeTime       time.Time `json:"change_time" gorm:"type:datetime;not null;comment:变更时间"`
	SourceIP         string    `json:"source_ip" gorm:"type:varchar(45);comment:来源IP" example:"192.168.1.100"`
	UserAgent        string    `json:"user_agent" gorm:"type:text;comment:用户代理"`
	SessionID        string    `json:"session_id" gorm:"type:varchar(100);comment:会话ID"`
	Metadata         string    `json:"metadata" gorm:"type:json;comment:元数据"`
}

// TableName 表名
func (ConfigChangeHistory) TableName() string {
	return "sys_config_change_history"
}

// ConfigBackup 配置备份记录
type ConfigBackup struct {
	global.GVA_MODEL
	ConfigType   string    `json:"config_type" gorm:"type:varchar(50);not null;comment:配置类型" example:"compliance"`
	BackupName   string    `json:"backup_name" gorm:"type:varchar(200);not null;comment:备份名称" example:"compliance_backup_20231201"`
	BackupPath   string    `json:"backup_path" gorm:"type:varchar(500);not null;comment:备份文件路径"`
	BackupSize   int64     `json:"backup_size" gorm:"type:bigint;comment:备份文件大小" example:"1024"`
	BackupReason string    `json:"backup_reason" gorm:"type:text;comment:备份原因" example:"定期备份"`
	BackupTime   time.Time `json:"backup_time" gorm:"type:datetime;not null;comment:备份时间"`
	OperatorID   string    `json:"operator_id" gorm:"type:varchar(100);comment:操作者ID" example:"admin"`
	OperatorName string    `json:"operator_name" gorm:"type:varchar(100);comment:操作者名称" example:"管理员"`
	IsAutoBackup bool      `json:"is_auto_backup" gorm:"type:boolean;default:false;comment:是否自动备份" example:"false"`
	Status       string    `json:"status" gorm:"type:varchar(20);default:'success';comment:备份状态" example:"success"`
	ErrorMessage string    `json:"error_message" gorm:"type:text;comment:错误信息"`
}

// TableName 表名
func (ConfigBackup) TableName() string {
	return "sys_config_backup"
}

// ConfigValidationResult 配置验证结果
type ConfigValidationResult struct {
	global.GVA_MODEL
	ConfigType         string    `json:"config_type" gorm:"type:varchar(50);not null;comment:配置类型" example:"compliance"`
	ValidationTime     time.Time `json:"validation_time" gorm:"type:datetime;not null;comment:验证时间"`
	IsValid            bool      `json:"is_valid" gorm:"type:boolean;not null;comment:是否有效" example:"true"`
	ErrorCount         int       `json:"error_count" gorm:"type:int;default:0;comment:错误数量" example:"0"`
	WarningCount       int       `json:"warning_count" gorm:"type:int;default:0;comment:警告数量" example:"2"`
	ValidationErrors   string    `json:"validation_errors" gorm:"type:text;comment:验证错误"`
	ValidationWarnings string    `json:"validation_warnings" gorm:"type:text;comment:验证警告"`
	ValidatedBy        string    `json:"validated_by" gorm:"type:varchar(100);comment:验证者" example:"admin"`
	ValidationTrigger  string    `json:"validation_trigger" gorm:"type:varchar(50);comment:验证触发方式" example:"manual"`
	ValidationDetails  string    `json:"validation_details" gorm:"type:json;comment:验证详细信息"`
}

// TableName 表名
func (ConfigValidationResult) TableName() string {
	return "sys_config_validation_result"
}
