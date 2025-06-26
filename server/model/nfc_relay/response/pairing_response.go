package response

// PairingResponse 是配对请求的统一响应体
type PairingResponse struct {
	// 通用的等待/匹配状态信息
	QueuePosition int    `json:"queue_position"`
	EstimatedWait int    `json:"estimated_wait"`
	Status        string `json:"status"` // "waiting", "matched", "conflict" 等
	TransactionID string `json:"transaction_id,omitempty"`

	// 统一的、权威的身份凭证
	ClientID  string `json:"client_id"`
	MqttToken string `json:"mqtt_token"`
	ExpiresAt int64  `json:"mqtt_token_expires_at"` // Token 过期时间戳 (Unix秒)
	Role      string `json:"role"`
}
