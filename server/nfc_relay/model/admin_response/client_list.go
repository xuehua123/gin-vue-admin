package admin_response

import "time"

// ClientInfo 定义了客户端信息的响应数据结构
type ClientInfo struct {
	ClientID      string    `json:"client_id"`       // 客户端ID
	UserID        string    `json:"user_id"`         // 用户ID
	DisplayName   string    `json:"display_name"`    // 客户端显示名称
	Role          string    `json:"role"`            // 当前角色 (Provider/Receiver/None)
	IPAddress     string    `json:"ip_address"`      // IP地址
	ConnectedAt   time.Time `json:"connected_at"`    // 连接建立时间戳
	IsOnline      bool      `json:"is_online"`       // 在线状态
	SessionID     string    `json:"session_id"`      // 当前参与的会话ID (如果有)
	UserAgent     string    `json:"user_agent"`      // 用户代理 (可选，考虑扩展)
	LastMessageAt time.Time `json:"last_message_at"` // 最后接收消息时间 (可选，考虑扩展)
}

// PaginatedClientListResponse 定义了分页的客户端列表响应
type PaginatedClientListResponse struct {
	List     []ClientInfo `json:"list"`      // 客户端列表
	Total    int64        `json:"total"`     // 总记录数
	Page     int          `json:"page"`      // 当前页码
	PageSize int          `json:"page_size"` // 每页条数
}

// ClientDetailResponse 定义了单个客户端详细信息的响应
type ClientDetailResponse struct {
	ClientInfo
	// 以下为扩展字段，可以根据需要添加
	SentMessageCount     int64                 `json:"sent_message_count"`     // 已发送消息计数
	ReceivedMessageCount int64                 `json:"received_message_count"` // 已接收消息计数
	ConnectionEvents     []ConnectionEvent     `json:"connection_events"`      // 连接相关的关键事件历史
	RelatedAuditLogs     []RelatedAuditLogItem `json:"related_audit_logs"`     // 相关审计日志
}

// ConnectionEvent 定义了连接事件
type ConnectionEvent struct {
	Timestamp time.Time `json:"timestamp"` // 时间戳
	Event     string    `json:"event"`     // 事件类型
	Details   string    `json:"details"`   // 事件详情
}

// RelatedAuditLogItem 定义了相关审计日志项
type RelatedAuditLogItem struct {
	Timestamp      time.Time `json:"timestamp"`       // 时间戳
	EventType      string    `json:"event_type"`      // 事件类型
	DetailsSummary string    `json:"details_summary"` // 详情摘要
}
