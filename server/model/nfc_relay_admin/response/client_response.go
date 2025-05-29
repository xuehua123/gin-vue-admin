package response

// ClientInfo 客户端基础信息
type ClientInfo struct {
	ClientID    string `json:"client_id"`            // 客户端ID
	UserID      string `json:"user_id"`              // 用户ID
	DisplayName string `json:"display_name"`         // 显示名称
	Role        string `json:"role"`                 // 角色：provider/receiver/none
	IPAddress   string `json:"ip_address"`           // IP地址
	ConnectedAt string `json:"connected_at"`         // 连接时间
	IsOnline    bool   `json:"is_online"`            // 在线状态
	SessionID   string `json:"session_id,omitempty"` // 当前会话ID
}

// PaginatedClientListResponse 分页客户端列表响应
type PaginatedClientListResponse struct {
	List     []ClientInfo `json:"list"`     // 客户端列表
	Total    int          `json:"total"`    // 总记录数
	Page     int          `json:"page"`     // 当前页码
	PageSize int          `json:"pageSize"` // 每页数量
}

// ConnectionEvent 连接事件
type ConnectionEvent struct {
	Timestamp string `json:"timestamp"`           // 事件时间
	Event     string `json:"event"`               // 事件类型
	ClientID  string `json:"client_id,omitempty"` // 相关客户端ID
	Details   string `json:"details,omitempty"`   // 事件详情
}

// ClientDetailsResponse 客户端详细信息响应
type ClientDetailsResponse struct {
	ClientID             string            `json:"client_id"`                 // 客户端ID
	UserID               string            `json:"user_id"`                   // 用户ID
	DisplayName          string            `json:"display_name"`              // 显示名称
	Role                 string            `json:"role"`                      // 角色：provider/receiver/none
	IPAddress            string            `json:"ip_address"`                // IP地址
	UserAgent            string            `json:"user_agent,omitempty"`      // 用户代理
	ConnectedAt          string            `json:"connected_at"`              // 连接时间
	LastMessageAt        string            `json:"last_message_at,omitempty"` // 最后消息时间
	IsOnline             bool              `json:"is_online"`                 // 在线状态
	SessionID            string            `json:"session_id,omitempty"`      // 当前会话ID
	SentMessageCount     int64             `json:"sent_message_count"`        // 已发送消息数
	ReceivedMessageCount int64             `json:"received_message_count"`    // 已接收消息数
	ConnectionEvents     []ConnectionEvent `json:"connection_events"`         // 连接事件历史
}
