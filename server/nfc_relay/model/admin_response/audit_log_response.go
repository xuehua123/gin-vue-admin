package admin_response

// AuditLogItem 审计日志项
type AuditLogItem struct {
	Timestamp         string      `json:"timestamp"`                     // 时间戳
	EventType         string      `json:"event_type"`                    // 事件类型
	SessionID         string      `json:"session_id,omitempty"`          // 会话ID
	ClientIDInitiator string      `json:"client_id_initiator,omitempty"` // 发起方客户端ID
	ClientIDResponder string      `json:"client_id_responder,omitempty"` // 响应方客户端ID
	UserID            string      `json:"user_id,omitempty"`             // 用户ID
	SourceIP          string      `json:"source_ip,omitempty"`           // 源IP地址
	Details           interface{} `json:"details,omitempty"`             // 事件详情
}

// PaginatedAuditLogResponse 分页审计日志响应
type PaginatedAuditLogResponse struct {
	List     []AuditLogItem `json:"list"`     // 日志列表
	Total    int            `json:"total"`    // 总数
	Page     int            `json:"page"`     // 当前页码
	PageSize int            `json:"pageSize"` // 每页条数
}
