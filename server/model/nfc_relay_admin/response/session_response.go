package response

// SessionInfo 会话基础信息
type SessionInfo struct {
	SessionID           string `json:"session_id"`                 // 会话ID
	ProviderClientID    string `json:"provider_client_id"`         // Provider客户端ID
	ProviderUserID      string `json:"provider_user_id"`           // Provider用户ID
	ProviderDisplayName string `json:"provider_display_name"`      // Provider显示名称
	ReceiverClientID    string `json:"receiver_client_id"`         // Receiver客户端ID
	ReceiverUserID      string `json:"receiver_user_id"`           // Receiver用户ID
	ReceiverDisplayName string `json:"receiver_display_name"`      // Receiver显示名称
	Status              string `json:"status"`                     // 会话状态
	CreatedAt           string `json:"created_at"`                 // 创建时间
	LastActivityAt      string `json:"last_activity_at,omitempty"` // 最后活动时间
}

// PaginatedSessionListResponse 分页会话列表响应
type PaginatedSessionListResponse struct {
	List     []SessionInfo `json:"list"`     // 会话列表
	Total    int           `json:"total"`    // 总记录数
	Page     int           `json:"page"`     // 当前页码
	PageSize int           `json:"pageSize"` // 每页数量
}

// ParticipantInfo 会话参与者信息
type ParticipantInfo struct {
	ClientID    string `json:"client_id"`    // 客户端ID
	UserID      string `json:"user_id"`      // 用户ID
	DisplayName string `json:"display_name"` // 显示名称
	IPAddress   string `json:"ip_address"`   // IP地址
}

// ApduExchangeCount APDU交换统计
type ApduExchangeCount struct {
	Upstream   int64 `json:"upstream"`   // 上行消息数（Receiver到Provider）
	Downstream int64 `json:"downstream"` // 下行消息数（Provider到Receiver）
}

// SessionEvent 会话事件
type SessionEvent struct {
	Timestamp string `json:"timestamp"`           // 事件时间
	Event     string `json:"event"`               // 事件类型
	ClientID  string `json:"client_id,omitempty"` // 相关客户端ID
	Details   string `json:"details,omitempty"`   // 事件详情
}

// SessionDetailsResponse 会话详细信息响应
type SessionDetailsResponse struct {
	SessionID            string            `json:"session_id"`                   // 会话ID
	Status               string            `json:"status"`                       // 会话状态
	CreatedAt            string            `json:"created_at"`                   // 创建时间
	LastActivityAt       string            `json:"last_activity_at,omitempty"`   // 最后活动时间
	TerminatedAt         string            `json:"terminated_at,omitempty"`      // 终止时间
	TerminationReason    string            `json:"termination_reason,omitempty"` // 终止原因
	ProviderInfo         ParticipantInfo   `json:"provider_info"`                // Provider信息
	ReceiverInfo         ParticipantInfo   `json:"receiver_info"`                // Receiver信息
	ApduExchangeCount    ApduExchangeCount `json:"apdu_exchange_count"`          // APDU交换统计
	SessionEventsHistory []SessionEvent    `json:"session_events_history"`       // 会话事件历史
}
