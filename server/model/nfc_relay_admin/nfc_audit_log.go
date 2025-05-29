package nfc_relay_admin

import (
	"time"

	"github.com/flipped-aurora/gin-vue-admin/server/global"
)

// NfcAuditLog NFC中继系统审计日志表
type NfcAuditLog struct {
	global.GVA_MODEL
	EventType         string    `json:"event_type" gorm:"index;size:50;not null;comment:事件类型"`       // 事件类型
	SessionID         string    `json:"session_id" gorm:"index;size:64;comment:会话ID"`                // 会话ID
	ClientIDInitiator string    `json:"client_id_initiator" gorm:"index;size:64;comment:发起方客户端ID"`   // 发起方客户端ID
	ClientIDResponder string    `json:"client_id_responder" gorm:"index;size:64;comment:响应方客户端ID"`   // 响应方客户端ID
	UserID            string    `json:"user_id" gorm:"index;size:64;comment:用户ID"`                   // 用户ID
	SourceIP          string    `json:"source_ip" gorm:"size:45;comment:源IP地址"`                      // 源IP地址
	UserAgent         string    `json:"user_agent" gorm:"type:text;comment:用户代理"`                    // 用户代理
	Details           string    `json:"details" gorm:"type:text;comment:事件详情JSON"`                   // 事件详情JSON
	Result            string    `json:"result" gorm:"size:20;comment:操作结果(success/failure/timeout)"` // 操作结果
	ErrorMessage      string    `json:"error_message" gorm:"type:text;comment:错误信息"`                 // 错误信息
	Duration          int64     `json:"duration" gorm:"comment:操作耗时(毫秒)"`                            // 操作耗时(毫秒)
	Resource          string    `json:"resource" gorm:"size:100;comment:操作资源"`                       // 操作资源
	Action            string    `json:"action" gorm:"size:50;comment:操作动作"`                          // 操作动作
	Level             string    `json:"level" gorm:"index;size:20;default:info;comment:日志级别"`        // 日志级别(info/warn/error/critical)
	Category          string    `json:"category" gorm:"index;size:50;comment:事件分类"`                  // 事件分类
	ServerID          string    `json:"server_id" gorm:"size:64;comment:服务器ID"`                      // 服务器ID
	RequestID         string    `json:"request_id" gorm:"index;size:64;comment:请求ID"`                // 请求ID
	EventTime         time.Time `json:"event_time" gorm:"index;not null;comment:事件发生时间"`             // 事件发生时间
}

// TableName 指定表名
func (NfcAuditLog) TableName() string {
	return "nfc_audit_logs"
}

// NfcClientBanRecord NFC客户端封禁记录表
type NfcClientBanRecord struct {
	global.GVA_MODEL
	ClientID   string     `json:"client_id" gorm:"uniqueIndex;size:64;not null;comment:客户端ID"`        // 客户端ID
	UserID     string     `json:"user_id" gorm:"index;size:64;comment:用户ID"`                          // 用户ID
	BanReason  string     `json:"ban_reason" gorm:"type:text;not null;comment:封禁原因"`                  // 封禁原因
	BanType    string     `json:"ban_type" gorm:"size:20;not null;comment:封禁类型(temporary/permanent)"` // 封禁类型
	BannedBy   uint       `json:"banned_by" gorm:"not null;comment:执行封禁的管理员ID"`                       // 执行封禁的管理员ID
	BannedAt   time.Time  `json:"banned_at" gorm:"not null;comment:封禁时间"`                             // 封禁时间
	ExpiresAt  *time.Time `json:"expires_at" gorm:"comment:解封时间(永久封禁为null)"`                          // 解封时间
	UnbannedBy *uint      `json:"unbanned_by" gorm:"comment:执行解封的管理员ID"`                              // 执行解封的管理员ID
	UnbannedAt *time.Time `json:"unbanned_at" gorm:"comment:实际解封时间"`                                  // 实际解封时间
	IsActive   bool       `json:"is_active" gorm:"index;default:true;comment:是否激活(true=封禁中)"`         // 是否激活
	SourceIP   string     `json:"source_ip" gorm:"size:45;comment:客户端IP"`                             // 客户端IP
	Violations int        `json:"violations" gorm:"default:1;comment:违规次数"`                           // 违规次数
	Severity   string     `json:"severity" gorm:"size:20;comment:严重程度(low/medium/high/critical)"`     // 严重程度
	Notes      string     `json:"notes" gorm:"type:text;comment:备注信息"`                                // 备注信息
}

// TableName 指定表名
func (NfcClientBanRecord) TableName() string {
	return "nfc_client_ban_records"
}

// IsExpired 检查封禁是否已过期
func (r *NfcClientBanRecord) IsExpired() bool {
	if !r.IsActive {
		return true
	}
	if r.ExpiresAt == nil {
		return false // 永久封禁不会过期
	}
	return time.Now().After(*r.ExpiresAt)
}

// GetRemainingTime 获取剩余封禁时间
func (r *NfcClientBanRecord) GetRemainingTime() time.Duration {
	if !r.IsActive || r.ExpiresAt == nil {
		return 0
	}
	remaining := r.ExpiresAt.Sub(time.Now())
	if remaining < 0 {
		return 0
	}
	return remaining
}

// NfcUserSecurityProfile 用户安全档案表
type NfcUserSecurityProfile struct {
	global.GVA_MODEL
	UserID           string     `json:"user_id" gorm:"uniqueIndex;size:64;not null;comment:用户ID"`      // 用户ID
	Status           string     `json:"status" gorm:"index;size:20;default:active;comment:账户状态"`       // 账户状态(active/banned/suspended/locked)
	SecurityLevel    string     `json:"security_level" gorm:"size:20;default:normal;comment:安全级别"`     // 安全级别(low/normal/high/critical)
	FailedLoginCount int        `json:"failed_login_count" gorm:"default:0;comment:连续失败登录次数"`          // 连续失败登录次数
	LastLoginAt      time.Time  `json:"last_login_at" gorm:"comment:最后登录时间"`                           // 最后登录时间
	LastLoginIP      string     `json:"last_login_ip" gorm:"size:45;comment:最后登录IP"`                   // 最后登录IP
	LoginAttempts    int        `json:"login_attempts" gorm:"default:0;comment:今日登录尝试次数"`              // 今日登录尝试次数
	LastAttemptAt    time.Time  `json:"last_attempt_at" gorm:"comment:最后尝试时间"`                         // 最后尝试时间
	AccountLockedAt  *time.Time `json:"account_locked_at" gorm:"comment:账户锁定时间"`                       // 账户锁定时间
	LockExpiresAt    *time.Time `json:"lock_expires_at" gorm:"comment:锁定过期时间"`                         // 锁定过期时间
	TwoFactorEnabled bool       `json:"two_factor_enabled" gorm:"default:false;comment:是否启用双因子认证"`     // 是否启用双因子认证
	ViolationCount   int        `json:"violation_count" gorm:"default:0;comment:违规次数"`                 // 违规次数
	LastViolationAt  *time.Time `json:"last_violation_at" gorm:"comment:最后违规时间"`                       // 最后违规时间
	RiskScore        float64    `json:"risk_score" gorm:"type:decimal(5,2);default:0.00;comment:风险评分"` // 风险评分
	Notes            string     `json:"notes" gorm:"type:text;comment:安全备注"`                           // 安全备注
}

// TableName 指定表名
func (NfcUserSecurityProfile) TableName() string {
	return "nfc_user_security_profiles"
}

// IsLocked 检查账户是否被锁定
func (p *NfcUserSecurityProfile) IsLocked() bool {
	if p.AccountLockedAt == nil {
		return false
	}
	if p.LockExpiresAt == nil {
		return true // 永久锁定
	}
	return time.Now().Before(*p.LockExpiresAt)
}

// IsBanned 检查账户是否被封禁
func (p *NfcUserSecurityProfile) IsBanned() bool {
	return p.Status == "banned"
}

// IsSuspended 检查账户是否被暂停
func (p *NfcUserSecurityProfile) IsSuspended() bool {
	return p.Status == "suspended"
}

// IsActive 检查账户是否激活
func (p *NfcUserSecurityProfile) IsActive() bool {
	return p.Status == "active" && !p.IsLocked()
}
