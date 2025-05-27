package admin_request

// SessionListParams 定义了获取会话列表的请求参数
type SessionListParams struct {
	Page                int    `form:"page,default=1" json:"page,omitempty"`                     // 页码
	PageSize            int    `form:"pageSize,default=10" json:"pageSize,omitempty"`            // 每页数量
	SessionID           string `form:"sessionId" json:"sessionId,omitempty"`                     // 按会话ID筛选
	ParticipantClientID string `form:"participantClientId" json:"participantClientId,omitempty"` // 按参与方任一客户端ID筛选
	ParticipantUserID   string `form:"participantUserId" json:"participantUserId,omitempty"`     // 按参与方任一用户ID筛选
	Status              string `form:"status" json:"status,omitempty"`                           // 按会话状态筛选 (paired, waiting_for_pairing, terminated)
}

// TerminateSessionRequest 定义了终止会话的请求参数
type TerminateSessionRequest struct {
	Reason string `json:"reason,omitempty"` // 终止原因，可选
}
