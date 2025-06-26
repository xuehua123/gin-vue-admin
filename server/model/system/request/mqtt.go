package request

import (
	"encoding/json"
)

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
	Access   int    `json:"access"`
	Username string `json:"username"`
	ClientID string `json:"clientid"`
	IPAddr   string `json:"ipaddr"`
	Topic    string `json:"topic"`
	Action   string `json:"action"`
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

// EmqxWebhookRequest EMQX标准WebHook请求结构
// 支持客户端连接/断开、会话创建/终止等事件
type EmqxWebhookRequest struct {
	// 基础事件信息
	Event     string      `json:"event"`               // 事件类型：client.connected, client.disconnected, session.created, session.terminated
	EventType string      `json:"event_type"`          // 事件分类，用于一些版本的EMQX
	Timestamp json.Number `json:"timestamp,omitempty"` // 事件时间戳（毫秒）
	Node      string      `json:"node"`                // EMQX节点名称

	// 客户端信息
	ClientID string `json:"clientid"` // 客户端ID
	Username string `json:"username"` // 用户名
	PeerHost string `json:"peerhost"` // 客户端IP地址
	PeerPort *int   `json:"peerport"` // 客户端端口
	SockPort *int   `json:"sockport"` // 服务器端口

	// 连接协议信息
	Protocol        string `json:"protocol"`        // 协议类型：mqtt, mqtt-sn, coap
	ProtocolVersion *int   `json:"proto_ver"`       // 协议版本：3, 4, 5
	Keepalive       *int   `json:"keepalive"`       // 心跳间隔
	CleanStart      *bool  `json:"clean_start"`     // 是否清理会话
	ExpiryInterval  *int64 `json:"expiry_interval"` // 会话过期间隔

	// 断开连接信息
	Reason         string      `json:"reason"`          // 断开原因
	DisconnectedAt json.Number `json:"disconnected_at"` // 断开时间戳
	ConnectedAt    json.Number `json:"connected_at"`    // 连接时间戳

	// 会话信息
	SessionID       string `json:"session_id"` // 会话ID
	CreateSessionAt *int64 `json:"created_at"` // 会话创建时间

	// ACL检查特定信息
	Action string `json:"action"` // "publish" or "subscribe"
	Topic  string `json:"topic"`  // 主题
}

// MqttAuthCache 是存储在Redis中的MQTT认证成功后的缓存信息
type MqttAuthCache struct {
	Role     string `json:"role"`
	Username string `json:"username"`
}

// MqttEmqxHookRequest MQTT EMQX WebHook请求结构体
type MqttEmqxHookRequest struct {
	// ... existing code ...
}
