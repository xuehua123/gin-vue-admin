package request

// MqttAuthRequest MQTT认证/ACL请求结构
type MqttAuthRequest struct {
	ClientID string `json:"clientid"`
	Username string `json:"username"`
	Password string `json:"password,omitempty"` // 认证时需要
	Topic    string `json:"topic,omitempty"`    // ACL检查时需要
	Action   string `json:"action,omitempty"`   // ACL检查时需要
}

// MqttConnectionStatusRequest MQTT连接状态变化请求结构
type MqttConnectionStatusRequest struct {
	Event          string `json:"event"`                     // client_connected 或 client_disconnected
	ClientID       string `json:"clientid"`                  // 客户端ID
	Username       string `json:"username"`                  // 用户名
	ConnectedAt    int64  `json:"connected_at,omitempty"`    // 连接时间戳
	DisconnectedAt int64  `json:"disconnected_at,omitempty"` // 断开时间戳
	Reason         string `json:"reason,omitempty"`          // 断开原因
	Timestamp      int64  `json:"timestamp"`                 // 事件时间戳
}
