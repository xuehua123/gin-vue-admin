package admin_response

import "time"

// SessionInfo 定义了会话信息的响应数据结构
type SessionInfo struct {
	SessionID           string    `json:"session_id"`                      // 会话ID
	ProviderClientID    string    `json:"provider_client_id,omitempty"`    // 发卡方客户端ID
	ProviderUserID      string    `json:"provider_user_id,omitempty"`      // 发卡方用户ID
	ProviderDisplayName string    `json:"provider_display_name,omitempty"` // 发卡方显示名称
	ReceiverClientID    string    `json:"receiver_client_id,omitempty"`    // 收卡方客户端ID
	ReceiverUserID      string    `json:"receiver_user_id,omitempty"`      // 收卡方用户ID
	ReceiverDisplayName string    `json:"receiver_display_name,omitempty"` // 收卡方显示名称
	Status              string    `json:"status"`                          // 会话状态 (paired, waiting_for_pairing, terminated)
	CreatedAt           time.Time `json:"created_at"`                      // 会话创建时间
	LastActivityAt      time.Time `json:"last_activity_at"`                // 最后活动时间
}

// PaginatedSessionListResponse 定义了分页的会话列表响应
type PaginatedSessionListResponse struct {
	List     []SessionInfo `json:"list"`      // 会话列表
	Total    int64         `json:"total"`     // 总记录数
	Page     int           `json:"page"`      // 当前页码
	PageSize int           `json:"page_size"` // 每页条数
}

// ClientSummaryInfo 定义了客户端的摘要信息
type ClientSummaryInfo struct {
	ClientID    string `json:"client_id"`              // 客户端ID
	UserID      string `json:"user_id"`                // 用户ID
	DisplayName string `json:"display_name,omitempty"` // 显示名称
	IPAddress   string `json:"ip_address,omitempty"`   // IP地址
}

// APDUExchangeCount 定义了APDU交换计数
type APDUExchangeCount struct {
	Upstream   int64 `json:"upstream"`   // 从Receiver到Provider (通常是指令)
	Downstream int64 `json:"downstream"` // 从Provider到Receiver (通常是响应)
}

// SessionEvent 定义了会话生命周期中的事件
type SessionEvent struct {
	Timestamp time.Time `json:"timestamp"`           // 时间戳
	Event     string    `json:"event"`               // 事件类型
	ClientID  string    `json:"client_id,omitempty"` // 相关客户端ID
	UserID    string    `json:"user_id,omitempty"`   // 相关用户ID
	Details   string    `json:"details,omitempty"`   // 事件详情
}

// SessionDetailsResponse 定义了会话详情的响应
type SessionDetailsResponse struct {
	SessionID               string                `json:"session_id"`                           // 会话ID
	Status                  string                `json:"status"`                               // 会话状态
	CreatedAt               time.Time             `json:"created_at"`                           // 会话创建时间
	LastActivityAt          time.Time             `json:"last_activity_at"`                     // 最后活动时间
	TerminatedAt            *time.Time            `json:"terminated_at,omitempty"`              // 终止时间，如果已终止
	TerminationReason       string                `json:"termination_reason,omitempty"`         // 终止原因，如果已终止
	ProviderInfo            *ClientSummaryInfo    `json:"provider_info,omitempty"`              // Provider客户端摘要信息
	ReceiverInfo            *ClientSummaryInfo    `json:"receiver_info,omitempty"`              // Receiver客户端摘要信息
	APDUExchangeCount       APDUExchangeCount     `json:"apdu_exchange_count"`                  // APDU交换计数
	SessionEventsHistory    []SessionEvent        `json:"session_events_history,omitempty"`     // 会话事件历史
	RelatedAuditLogsSummary []RelatedAuditLogItem `json:"related_audit_logs_summary,omitempty"` // 相关审计日志摘要
}
