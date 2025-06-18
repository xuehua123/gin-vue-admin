package request

// MqttAuthRequest MQTT认证/ACL请求结构
type MqttAuthRequest struct {
	ClientID string `json:"clientid"`
	Username string `json:"username"`
	Password string `json:"password,omitempty"` // 认证时需要
	Topic    string `json:"topic,omitempty"`    // ACL检查时需要
	Action   string `json:"action,omitempty"`   // ACL检查时需要
}
