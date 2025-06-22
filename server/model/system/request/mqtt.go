package request

// MqttAuthRequest MQTT认证/ACL请求结构
type MqttAuthRequest struct {
	ClientID string `json:"clientid"`
	Username string `json:"username"`
	Password string `json:"password,omitempty"` // 认证时需要
	Topic    string `json:"topic,omitempty"`    // ACL检查时需要
	Action   string `json:"action,omitempty"`   // ACL检查时需要
}

// MqttAclRequest MQTT ACL检查请求结构
type MqttAclRequest struct {
	Access   string `json:"access"`
	Username string `json:"username"`
	ClientID string `json:"clientid"`
	IPAddr   string `json:"ipaddr"`
	Topic    string `json:"topic"`
	Mount    string `json:"mountpoint"`
}

// MqttConnectionStatusRequest MQTT连接状态变化Webhook请求
type MqttConnectionStatusRequest struct {
	Event          string  `json:"event"`
	ClientID       string  `json:"clientid"`
	Username       string  `json:"username"`
	Reason         string  `json:"reason"`
	ConnectedAt    *string `json:"connected_at,omitempty"`    // 使用字符串指针，避免 "undefined" 解析错误
	DisconnectedAt *string `json:"disconnected_at,omitempty"` // 使用字符串指针，避免 "undefined" 解析错误
}
