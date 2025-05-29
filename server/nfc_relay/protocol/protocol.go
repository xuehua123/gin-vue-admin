package protocol

// MessageType 定义了 WebSocket 消息的类型
type MessageType string

const (
	// Client to Server
	MessageTypeClientAuth         MessageType = "client_auth"             // 客户端认证 (携带token)
	MessageTypeDeclareRole        MessageType = "declare_role"            // 客户端声明角色和状态
	MessageTypeListCardProviders  MessageType = "list_card_providers"     // 收卡方请求可用发卡方列表
	MessageTypeSelectCardProvider MessageType = "select_card_provider"    // 收卡方选择一个发卡方请求连接
	MessageTypeAPDUUpstream       MessageType = "apdu_upstream"           // 收卡端发送给服务器的APDU
	MessageTypeAPDUDownstream     MessageType = "apdu_downstream"         // 传卡端发送给服务器的APDU
	MessageTypeStatusUpdate       MessageType = "status_update_to_server" // 客户端状态更新
	MessageTypeEndSession         MessageType = "end_session"             // 客户端主动请求结束当前会话
	MessageTypeHeartbeat          MessageType = "heartbeat"               // 心跳消息 (不需要认证)

	// Server to Client
	MessageTypeServerAuthResponse   MessageType = "server_auth_response"   // 服务器认证响应
	MessageTypeRoleDeclaredResponse MessageType = "role_declared_response" // 服务器对角色声明的响应
	MessageTypeCardProvidersList    MessageType = "card_providers_list"    // 服务器响应发卡方列表
	MessageTypeSessionEstablished   MessageType = "session_established"    // 通知客户端会话已建立
	MessageTypeSessionFailed        MessageType = "session_failed"         // 通知客户端会话建立失败
	MessageTypeSessionTerminated    MessageType = "session_terminated"     // 会话终止
	MessageTypePeerDisconnected     MessageType = "peer_disconnected"      // 配对的另一端断开连接
	MessageTypeAPDUToCard           MessageType = "apdu_to_card"           // 服务器转发给传卡端的APDU
	MessageTypeAPDUFromCard         MessageType = "apdu_from_card"         // 服务器转发给收卡端的APDU (来自传卡端的响应)
	MessageTypeError                MessageType = "error"                  // 通用错误消息
	MessageTypePeerStatusUpdate     MessageType = "peer_status_update"     // 对端状态更新
	MessageTypeHeartbeatResponse    MessageType = "heartbeat_response"     // 心跳响应消息
)

// RoleType 定义了客户端的角色类型
type RoleType string

const (
	RoleProvider RoleType = "provider"
	RoleReceiver RoleType = "receiver"
	RoleNone     RoleType = "none" // 用于清除角色或表示未定义
)

// ErrorCode 定义了标准错误代码
const (
	ErrorCodeBadRequest          = 40001 // 无效的请求格式或参数
	ErrorCodeAuthRequired        = 40101 // 需要认证
	ErrorCodeAuthFailed          = 40102 // 认证失败 (例如 Token 无效、过期、被拒)
	ErrorCodePermissionDenied    = 40301 // 权限不足
	ErrorCodeNotFound            = 40401 // 资源未找到
	ErrorCodeProviderNotFound    = 40402 // 指定的发卡方未找到或不可用
	ErrorCodeMethodNotAllowed    = 40501 // 不支持的操作或方法
	ErrorCodeConflict            = 40901 // 状态冲突 (例如尝试连接已在会话中的客户端)
	ErrorCodeSessionConflict     = 40902 // 会话相关的冲突 (例如一方或双方状态已改变)
	ErrorCodeProviderBusy        = 40903 // 发卡方正忙
	ErrorCodeReceiverBusy        = 40904 // 收卡方正忙
	ErrorCodeSelectSelf          = 40905 // 不能选择自己
	ErrorCodeProviderUnavailable = 40906 // 提供者状态变更，不可用

	ErrorCodeUnsupportedType    = 41501 // 不支持的消息类型
	ErrorCodeInternalError      = 50001 // 服务器内部错误
	ErrorCodeNotImplemented     = 50101 // 功能未实现
	ErrorCodeServiceUnavailable = 50301 // 服务不可用
)

// GenericMessage 是所有 WebSocket 消息的基础结构，用于解析消息类型
type GenericMessage struct {
	Type MessageType `json:"type"`
	Seq  int64       `json:"seq,omitempty"` // 可选的消息序列号，用于追踪
}

// ClientInfo 包含客户端的一些基本信息

// APDUUpstreamMessage 收卡端 (POS) 发送给服务器的 APDU
// 方向: Client (POS) -> Server
type APDUUpstreamMessage struct {
	Type      MessageType `json:"type"` // 应为 "apdu_upstream"
	SessionID string      `json:"sessionId"`
	APDU      string      `json:"apdu"` // APDU 数据，通常为 Base64 编码的字符串
}

// APDUToCardMessage 服务器转发给传卡端 (Card) 的 APDU
// 方向: Server -> Client (Card)
type APDUToCardMessage struct {
	Type      MessageType `json:"type"` // 应为 "apdu_to_card"
	SessionID string      `json:"sessionId"`
	APDU      string      `json:"apdu"` // APDU 数据
}

// APDUFromCardMessage 服务器转发给收卡端 (POS) 的 APDU (来自传卡端的响应)
// 方向: Client (Card) -> Server (然后 Server -> Client (POS))
// 注意：这个消息类型用于从传卡端到服务器，服务器再包装成 apdu_from_card 发给POS。
// 或者传卡端可以直接发送一个包含APDU响应的通用消息，由服务器判断后转发。
// 这里简化处理，假设服务器收到这个后，会构造一个新的消息发给POS。
// 另一种设计是客户端统一发送 "apdu_response", 服务器根据角色转发。
// 为清晰，定义一个 "apdu_downstream" 从传卡端到服务器。
type APDUDownstreamMessage struct {
	Type      MessageType `json:"type"` // 例如 "apdu_downstream" (传卡端 -> 服务器)
	SessionID string      `json:"sessionId"`
	APDU      string      `json:"apdu"` // APDU 响应数据
}

// ServerAPDUFromCardMessage 服务器转发给收卡端 (POS) 的来自传卡端的 APDU 响应
// 方向: Server -> Client (POS)
type ServerAPDUFromCardMessage struct {
	Type      MessageType `json:"type"` // 应为 "apdu_from_card"
	SessionID string      `json:"sessionId"`
	APDU      string      `json:"apdu"` // APDU 响应数据
}

// ErrorMessage 通用错误消息
// 方向: Server -> Client
type ErrorMessage struct {
	Type      MessageType `json:"type"` // 应为 "error"
	SessionID string      `json:"sessionId,omitempty"`
	Code      int         `json:"code"`
	Message   string      `json:"message"`
}

// PeerDisconnectedMessage 对端断开连接
// 方向: Server -> Client
type PeerDisconnectedMessage struct {
	Type      MessageType `json:"type"` // 应为 "peer_disconnected"
	SessionID string      `json:"sessionId"`
	Reason    string      `json:"reason,omitempty"`
}

// SessionTerminatedMessage 会话终止
// 方向: Server -> Client (both)
type SessionTerminatedMessage struct {
	Type      MessageType `json:"type"` // 应为 "session_terminated"
	SessionID string      `json:"sessionId"`
	Reason    string      `json:"reason,omitempty"`
}

// StatusUpdateToServerMessage 客户端状态更新消息
// 方向: Client -> Server
type StatusUpdateToServerMessage struct {
	Type      MessageType `json:"type"` // 应为 "status_update_to_server"
	SessionID string      `json:"sessionId"`
	Status    string      `json:"status"`            // 例如 "CARD_CONNECTED", "CARD_REMOVED", "NFC_ERROR"
	Details   string      `json:"details,omitempty"` // 可选的详细信息
}

// PeerStatusUpdateMessage 对端状态更新消息
// 方向: Server -> Client
type PeerStatusUpdateMessage struct {
	Type      MessageType `json:"type"` // 应为 "peer_status_update"
	SessionID string      `json:"sessionId"`
	Status    string      `json:"status"`
	Details   string      `json:"details,omitempty"`
}

// PongMessage (用于心跳检测，如果需要自定义)
// type PongMessage struct {
// 	Type MessageType `json:"type"` // "pong"
// 	Timestamp int64 `json:"timestamp"`
// }

// PingMessage (用于心跳检测，如果需要自定义)
// type PingMessage struct {
//  Type MessageType `json:"type"` // "ping"
// 	Timestamp int64 `json:"timestamp"`
// }

// ClientAuthMessage 客户端认证消息
// 方向: Client -> Server
type ClientAuthMessage struct {
	Type  MessageType `json:"type"` // 应为 "client_auth"
	Token string      `json:"token"`
}

// ServerAuthResponseMessage 服务器认证响应消息
// 方向: Server -> Client
type ServerAuthResponseMessage struct {
	Type    MessageType `json:"type"` // 应为 "server_auth_response"
	Success bool        `json:"success"`
	UserID  string      `json:"userId,omitempty"`  // 认证成功时返回用户ID
	Message string      `json:"message,omitempty"` // 认证失败时的原因
}

// DeclareRoleMessage 客户端声明角色和在线状态
// 方向: Client -> Server
type DeclareRoleMessage struct {
	Type         MessageType `json:"type"`                   // 应为 "declare_role"
	Role         RoleType    `json:"role"`                   // "provider" (发卡方), "receiver" (收卡方), 或 "none" (清除角色/下线)
	Online       bool        `json:"online,omitempty"`       // 对于 provider，true 表示上线服务，false 表示下线；对于 receiver 可忽略或用于表示主动寻找状态
	ProviderName string      `json:"providerName,omitempty"` // 可选的，当 Role 为 "provider" 时，客户端可以提供一个显示名称
}

// RoleDeclaredResponseMessage 服务器对角色声明的响应
// 方向: Server -> Client
type RoleDeclaredResponseMessage struct {
	Type    MessageType `json:"type"` // 应为 "role_declared_response"
	Success bool        `json:"success"`
	Role    RoleType    `json:"role,omitempty"`    // 确认的角色
	Online  bool        `json:"online,omitempty"`  // 确认的在线状态
	Message string      `json:"message,omitempty"` // 失败时的原因
}

// ListCardProvidersMessage 收卡方请求可用发卡方列表
// 方向: Client -> Server
type ListCardProvidersMessage struct {
	Type MessageType `json:"type"` // 应为 "list_card_providers"
	// 可以添加过滤条件，例如按用户ID过滤（如果允许查看其他用户的发卡方）
	// string UserIDFilter `json:"userIdFilter,omitempty"`
}

// CardProviderInfo 单个发卡方的信息，用于列表
type CardProviderInfo struct {
	ProviderID   string `json:"providerId"`   // 唯一标识一个发卡方实例 (例如 client.ID)
	ProviderName string `json:"providerName"` // 发卡方名称 (例如用户昵称或自定义名称)
	UserID       string `json:"userId"`       // 该发卡方所属的用户ID
	IsBusy       bool   `json:"isBusy"`       // 该发卡方当前是否正忙于一个会话
	// 可以添加其他元数据，如在线时长，状态等
}

// CardProvidersListMessage 服务器响应的发卡方列表
// 方向: Server -> Client
type CardProvidersListMessage struct {
	Type      MessageType        `json:"type"` // 应为 "card_providers_list"
	Providers []CardProviderInfo `json:"providers"`
}

// SelectCardProviderMessage 收卡方选择一个发卡方请求连接
// 方向: Client (Receiver) -> Server
type SelectCardProviderMessage struct {
	Type       MessageType `json:"type"`       // 应为 "select_card_provider"
	ProviderID string      `json:"providerId"` // 被选择的发卡方实例ID (即发卡方 Client.ID)
}

// SessionEstablishedMessage 通知客户端会话已建立
// 方向: Server -> Client (both parties)
type SessionEstablishedMessage struct {
	Type      MessageType `json:"type"`      // 应为 "session_established"
	SessionID string      `json:"sessionId"` // 正式的会话ID (由服务器生成和管理)
	PeerID    string      `json:"peerId"`    // 对端客户端的ID
	PeerRole  RoleType    `json:"peerRole"`  // 对端客户端的角色 ("provider" 或 "receiver")
}

// SessionFailedMessage 通知客户端会话建立失败
// 方向: Server -> Client (Receiver, or Provider if a late failure occurs)
type SessionFailedMessage struct {
	Type             MessageType `json:"type"`                       // 应为 "session_failed"
	TargetProviderID string      `json:"targetProviderId,omitempty"` // 尝试连接的发卡方ID
	Reason           string      `json:"reason"`
}

// EndSessionMessage 客户端主动请求结束当前会话
// 方向: Client -> Server
type EndSessionMessage struct {
	Type      MessageType `json:"type"`      // 应为 "end_session"
	SessionID string      `json:"sessionId"` // 要结束的会话ID
}

// HeartbeatMessage 心跳消息 (不需要认证)
// 方向: Client -> Server
type HeartbeatMessage struct {
	Type      MessageType `json:"type"`      // 应为 "heartbeat"
	Timestamp int64       `json:"timestamp"` // 可选的时间戳
}

// HeartbeatResponseMessage 心跳响应消息
// 方向: Server -> Client
type HeartbeatResponseMessage struct {
	Type      MessageType `json:"type"`      // 应为 "heartbeat_response"
	Timestamp int64       `json:"timestamp"` // 服务器当前时间戳
}

// TODO: 根据《NFC中继支付系统技术开发手册》3.5节，补充和完善所有消息类型和字段。
// TODO: 考虑端到端加密时，APDU 字段可能需要传输加密后的数据，并可能附带其他如MAC或Nonce的字段。
