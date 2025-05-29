package request

import "time"

// AuditLogListRequest 审计日志查询请求
type AuditLogListRequest struct {
	Page      int    `form:"page,default=1" json:"page" binding:"min=1"`                  // 页码
	PageSize  int    `form:"pageSize,default=10" json:"pageSize" binding:"min=1,max=100"` // 每页条数
	EventType string `form:"eventType" json:"eventType"`                                  // 事件类型
	UserID    string `form:"userID" json:"userID"`                                        // 用户ID
	SessionID string `form:"sessionID" json:"sessionID"`                                  // 会话ID
	ClientID  string `form:"clientID" json:"clientID"`                                    // 客户端ID
	Level     string `form:"level" json:"level"`                                          // 日志级别
	Category  string `form:"category" json:"category"`                                    // 事件分类
	Result    string `form:"result" json:"result"`                                        // 操作结果
	SourceIP  string `form:"sourceIP" json:"sourceIP"`                                    // 源IP
	StartTime string `form:"startTime" json:"startTime"`                                  // 开始时间（ISO8601格式）
	EndTime   string `form:"endTime" json:"endTime"`                                      // 结束时间（ISO8601格式）
	Keyword   string `form:"keyword" json:"keyword"`                                      // 关键词搜索
}

// GetStartTimeAsTime 将字符串开始时间转换为time.Time
func (r *AuditLogListRequest) GetStartTimeAsTime() (time.Time, error) {
	if r.StartTime == "" {
		// 如果未提供，返回24小时前
		return time.Now().AddDate(0, 0, -1), nil
	}
	return time.Parse(time.RFC3339, r.StartTime)
}

// GetEndTimeAsTime 将字符串结束时间转换为time.Time
func (r *AuditLogListRequest) GetEndTimeAsTime() (time.Time, error) {
	if r.EndTime == "" {
		// 如果未提供，返回当前时间
		return time.Now(), nil
	}
	return time.Parse(time.RFC3339, r.EndTime)
}

// CreateAuditLogRequest 创建审计日志请求
type CreateAuditLogRequest struct {
	EventType         string      `json:"event_type" binding:"required"` // 事件类型
	SessionID         string      `json:"session_id"`                    // 会话ID
	ClientIDInitiator string      `json:"client_id_initiator"`           // 发起方客户端ID
	ClientIDResponder string      `json:"client_id_responder"`           // 响应方客户端ID
	UserID            string      `json:"user_id"`                       // 用户ID
	SourceIP          string      `json:"source_ip"`                     // 源IP地址
	UserAgent         string      `json:"user_agent"`                    // 用户代理
	Details           interface{} `json:"details"`                       // 事件详情
	Result            string      `json:"result" binding:"required"`     // 操作结果
	ErrorMessage      string      `json:"error_message"`                 // 错误信息
	Duration          int64       `json:"duration"`                      // 操作耗时(毫秒)
	Resource          string      `json:"resource"`                      // 操作资源
	Action            string      `json:"action"`                        // 操作动作
	Level             string      `json:"level" binding:"required"`      // 日志级别
	Category          string      `json:"category"`                      // 事件分类
	ServerID          string      `json:"server_id"`                     // 服务器ID
	RequestID         string      `json:"request_id"`                    // 请求ID
}

// ClientBanRequest 客户端封禁请求
type ClientBanRequest struct {
	ClientID  string `json:"client_id" binding:"required"`                               // 客户端ID
	UserID    string `json:"user_id"`                                                    // 用户ID
	BanReason string `json:"ban_reason" binding:"required"`                              // 封禁原因
	BanType   string `json:"ban_type" binding:"required,oneof=temporary permanent"`      // 封禁类型
	Duration  int    `json:"duration"`                                                   // 封禁时长(分钟，仅临时封禁需要)
	Severity  string `json:"severity" binding:"required,oneof=low medium high critical"` // 严重程度
	Notes     string `json:"notes"`                                                      // 备注信息
}

// ClientUnbanRequest 客户端解封请求
type ClientUnbanRequest struct {
	ClientID string `json:"client_id" binding:"required"` // 客户端ID
	Reason   string `json:"reason"`                       // 解封原因
}

// ClientBanListRequest 客户端封禁列表查询请求
type ClientBanListRequest struct {
	Page      int    `form:"page,default=1" json:"page" binding:"min=1"`                  // 页码
	PageSize  int    `form:"pageSize,default=10" json:"pageSize" binding:"min=1,max=100"` // 每页条数
	ClientID  string `form:"clientID" json:"clientID"`                                    // 客户端ID
	UserID    string `form:"userID" json:"userID"`                                        // 用户ID
	BanType   string `form:"banType" json:"banType"`                                      // 封禁类型
	IsActive  *bool  `form:"isActive" json:"isActive"`                                    // 是否激活
	Severity  string `form:"severity" json:"severity"`                                    // 严重程度
	StartTime string `form:"startTime" json:"startTime"`                                  // 开始时间
	EndTime   string `form:"endTime" json:"endTime"`                                      // 结束时间
}

// UserSecurityProfileRequest 用户安全档案查询请求
type UserSecurityProfileRequest struct {
	UserID string `form:"userID" json:"userID" uri:"userID" binding:"required"` // 用户ID
}

// UserSecurityProfileListRequest 用户安全档案列表查询请求
type UserSecurityProfileListRequest struct {
	Page          int     `form:"page,default=1" json:"page" binding:"min=1"`                  // 页码
	PageSize      int     `form:"pageSize,default=10" json:"pageSize" binding:"min=1,max=100"` // 每页条数
	Status        string  `form:"status" json:"status"`                                        // 账户状态
	SecurityLevel string  `form:"securityLevel" json:"securityLevel"`                          // 安全级别
	MinRiskScore  float64 `form:"minRiskScore" json:"minRiskScore"`                            // 最小风险评分
	MaxRiskScore  float64 `form:"maxRiskScore" json:"maxRiskScore"`                            // 最大风险评分
	IsLocked      *bool   `form:"isLocked" json:"isLocked"`                                    // 是否锁定
	UserIDLike    string  `form:"userIDLike" json:"userIDLike"`                                // 用户ID模糊查询
}

// UpdateUserSecurityRequest 更新用户安全档案请求
type UpdateUserSecurityRequest struct {
	UserID           string  `json:"user_id" binding:"required"` // 用户ID
	Status           string  `json:"status"`                     // 账户状态
	SecurityLevel    string  `json:"security_level"`             // 安全级别
	TwoFactorEnabled bool    `json:"two_factor_enabled"`         // 是否启用双因子认证
	RiskScore        float64 `json:"risk_score"`                 // 风险评分
	Notes            string  `json:"notes"`                      // 安全备注
}

// LockUserAccountRequest 锁定用户账户请求
type LockUserAccountRequest struct {
	UserID   string `json:"user_id" binding:"required"` // 用户ID
	Duration int    `json:"duration"`                   // 锁定时长(分钟，0表示永久锁定)
	Reason   string `json:"reason" binding:"required"`  // 锁定原因
}

// UnlockUserAccountRequest 解锁用户账户请求
type UnlockUserAccountRequest struct {
	UserID string `json:"user_id" binding:"required"` // 用户ID
	Reason string `json:"reason"`                     // 解锁原因
}
