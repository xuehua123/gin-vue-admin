package response

import (
	"time"
)

// AuditLogEntry 审计日志条目
type AuditLogEntry struct {
	ID                uint      `json:"id"`                            // 日志ID
	EventType         string    `json:"event_type"`                    // 事件类型
	SessionID         string    `json:"session_id,omitempty"`          // 会话ID
	ClientIDInitiator string    `json:"client_id_initiator,omitempty"` // 发起方客户端ID
	ClientIDResponder string    `json:"client_id_responder,omitempty"` // 响应方客户端ID
	UserID            string    `json:"user_id,omitempty"`             // 用户ID
	SourceIP          string    `json:"source_ip,omitempty"`           // 源IP地址
	UserAgent         string    `json:"user_agent,omitempty"`          // 用户代理
	Details           string    `json:"details,omitempty"`             // 事件详情JSON
	Result            string    `json:"result"`                        // 操作结果
	ErrorMessage      string    `json:"error_message,omitempty"`       // 错误信息
	Duration          int64     `json:"duration"`                      // 操作耗时(毫秒)
	Resource          string    `json:"resource,omitempty"`            // 操作资源
	Action            string    `json:"action,omitempty"`              // 操作动作
	Level             string    `json:"level"`                         // 日志级别
	Category          string    `json:"category,omitempty"`            // 事件分类
	ServerID          string    `json:"server_id,omitempty"`           // 服务器ID
	RequestID         string    `json:"request_id,omitempty"`          // 请求ID
	EventTime         time.Time `json:"event_time"`                    // 事件发生时间
	CreatedAt         time.Time `json:"created_at"`                    // 创建时间
}

// PaginatedAuditLogResponse 分页审计日志响应
type PaginatedAuditLogResponse struct {
	List     []AuditLogEntry `json:"list"`     // 审计日志列表
	Total    int64           `json:"total"`    // 总数
	Page     int             `json:"page"`     // 当前页码
	PageSize int             `json:"pageSize"` // 每页条数
}

// ClientBanRecordEntry 客户端封禁记录条目
type ClientBanRecordEntry struct {
	ID         uint       `json:"id"`                  // 记录ID
	ClientID   string     `json:"client_id"`           // 客户端ID
	UserID     string     `json:"user_id,omitempty"`   // 用户ID
	BanReason  string     `json:"ban_reason"`          // 封禁原因
	BanType    string     `json:"ban_type"`            // 封禁类型
	BannedBy   uint       `json:"banned_by"`           // 执行封禁的管理员ID
	BannedAt   time.Time  `json:"banned_at"`           // 封禁时间
	ExpiresAt  *time.Time `json:"expires_at"`          // 解封时间
	UnbannedBy *uint      `json:"unbanned_by"`         // 执行解封的管理员ID
	UnbannedAt *time.Time `json:"unbanned_at"`         // 实际解封时间
	IsActive   bool       `json:"is_active"`           // 是否激活
	SourceIP   string     `json:"source_ip,omitempty"` // 客户端IP
	Violations int        `json:"violations"`          // 违规次数
	Severity   string     `json:"severity"`            // 严重程度
	Notes      string     `json:"notes,omitempty"`     // 备注信息
	CreatedAt  time.Time  `json:"created_at"`          // 创建时间
	UpdatedAt  time.Time  `json:"updated_at"`          // 更新时间
}

// PaginatedClientBanResponse 分页客户端封禁记录响应
type PaginatedClientBanResponse struct {
	List     []ClientBanRecordEntry `json:"list"`     // 封禁记录列表
	Total    int64                  `json:"total"`    // 总数
	Page     int                    `json:"page"`     // 当前页码
	PageSize int                    `json:"pageSize"` // 每页条数
}

// UserSecurityProfileEntry 用户安全档案条目
type UserSecurityProfileEntry struct {
	ID               uint       `json:"id"`                      // 档案ID
	UserID           string     `json:"user_id"`                 // 用户ID
	Status           string     `json:"status"`                  // 账户状态
	SecurityLevel    string     `json:"security_level"`          // 安全级别
	FailedLoginCount int        `json:"failed_login_count"`      // 连续失败登录次数
	LastLoginAt      time.Time  `json:"last_login_at"`           // 最后登录时间
	LastLoginIP      string     `json:"last_login_ip,omitempty"` // 最后登录IP
	LoginAttempts    int        `json:"login_attempts"`          // 今日登录尝试次数
	LastAttemptAt    time.Time  `json:"last_attempt_at"`         // 最后尝试时间
	AccountLockedAt  *time.Time `json:"account_locked_at"`       // 账户锁定时间
	LockExpiresAt    *time.Time `json:"lock_expires_at"`         // 锁定过期时间
	TwoFactorEnabled bool       `json:"two_factor_enabled"`      // 是否启用双因子认证
	ViolationCount   int        `json:"violation_count"`         // 违规次数
	LastViolationAt  *time.Time `json:"last_violation_at"`       // 最后违规时间
	RiskScore        float64    `json:"risk_score"`              // 风险评分
	Notes            string     `json:"notes,omitempty"`         // 安全备注
	CreatedAt        time.Time  `json:"created_at"`              // 创建时间
	UpdatedAt        time.Time  `json:"updated_at"`              // 更新时间
}

// PaginatedUserSecurityProfileResponse 分页用户安全档案响应
type PaginatedUserSecurityProfileResponse struct {
	List     []UserSecurityProfileEntry `json:"list"`     // 用户安全档案列表
	Total    int64                      `json:"total"`    // 总数
	Page     int                        `json:"page"`     // 当前页码
	PageSize int                        `json:"pageSize"` // 每页条数
}

// AuditLogStatsResponse 审计日志统计响应
type AuditLogStatsResponse struct {
	TotalLogs      int64            `json:"total_logs"`       // 总日志数
	ErrorCount     int64            `json:"error_count"`      // 错误数量
	WarningCount   int64            `json:"warning_count"`    // 警告数量
	InfoCount      int64            `json:"info_count"`       // 信息数量
	EventTypeStats map[string]int64 `json:"event_type_stats"` // 事件类型统计
	CategoryStats  map[string]int64 `json:"category_stats"`   // 事件分类统计
	HourlyStats    []HourlyLogStats `json:"hourly_stats"`     // 按小时统计
}

// HourlyLogStats 按小时日志统计
type HourlyLogStats struct {
	Hour  int   `json:"hour"`  // 小时
	Count int64 `json:"count"` // 数量
}

// SecuritySummaryResponse 安全摘要响应
type SecuritySummaryResponse struct {
	ActiveBanCount     int64 `json:"active_ban_count"`     // 活跃封禁数量
	LockedAccountCount int64 `json:"locked_account_count"` // 锁定账户数量
	HighRiskUserCount  int64 `json:"high_risk_user_count"` // 高风险用户数量
	TotalViolations    int64 `json:"total_violations"`     // 总违规次数
}
