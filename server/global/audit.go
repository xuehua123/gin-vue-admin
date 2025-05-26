package global

import (
	"encoding/json"
	"time"

	"go.uber.org/zap"
)

// AuditEvent 代表一个结构化的审计日志条目。
type AuditEvent struct {
	Timestamp         string      `json:"timestamp"`
	EventType         string      `json:"event_type"`
	SessionID         string      `json:"session_id,omitempty"`          // 会话ID，可选
	ClientIDInitiator string      `json:"client_id_initiator,omitempty"` // 发起方客户端ID，可选
	ClientIDResponder string      `json:"client_id_responder,omitempty"` // 响应方客户端ID，可选
	SourceIP          string      `json:"source_ip,omitempty"`           // 源IP地址，可选
	UserID            string      `json:"user_id,omitempty"`             // 用户ID，可选
	Details           interface{} `json:"details,omitempty"`             // 事件详情，可选
}

// AuditLogger 是一个专用于审计事件的记录器。
// 我们可以配置此记录器以输出到单独的文件或系统。
var AuditLogger *zap.Logger

// InitializeAuditLogger 初始化审计记录器。
// 目前，它将使用全局记录器的配置，
// 但之后可以自定义以输出到不同的目标。
func InitializeAuditLogger() {
	// 作为起点，我们克隆 GVA_LOG 并可能重新配置它。
	// 若要真正分离，您需要创建一个新的 zap.Config 并构建一个新的记录器。
	if GVA_LOG != nil {
		AuditLogger = GVA_LOG.Named("audit") // 创建一个名为 "audit" 的新记录器
	} else {
		// 如果 GVA_LOG 未初始化时的回退方案（正常操作中不应发生）
		// 此基本设置将写入 stderr。请替换为您的实际回退方案或 panic。
		logger, err := zap.NewProduction()
		if err != nil {
			panic("初始化回退审计记录器失败: " + err.Error())
		}
		AuditLogger = logger.Named("audit")
		GVA_LOG.Warn("由于在 InitializeAuditLogger 调用时 GVA_LOG 为 nil，AuditLogger 已使用回退方案初始化。")
	}
}

// LogAuditEvent 创建并记录一个审计事件。
func LogAuditEvent(eventType string, details interface{}, fields ...zap.Field) {
	if AuditLogger == nil {
		// 如果为 nil，则尝试初始化它
		InitializeAuditLogger()
		if AuditLogger == nil { // 如果尝试后仍然为 nil，则说明有问题
			GVA_LOG.Error("AuditLogger 未初始化，无法记录审计事件", zap.String("eventType", eventType))
			return
		}
		GVA_LOG.Info("AuditLogger 为 nil，并在首次使用时已初始化。")
	}

	event := AuditEvent{
		Timestamp: time.Now().Format(time.RFC3339Nano),
		EventType: eventType,
		Details:   details,
	}

	// 将 AuditEvent 转换为 zap.Field 切片
	// 这允许对事件对象本身进行结构化日志记录。
	var eventFields []zap.Field

	eventMap := make(map[string]interface{})
	eventBytes, err := json.Marshal(event)
	if err == nil {
		_ = json.Unmarshal(eventBytes, &eventMap) //暂时忽略反序列化错误
		for k, v := range eventMap {
			eventFields = append(eventFields, zap.Any(k, v))
		}
	} else {
		GVA_LOG.Error("序列化 AuditEvent 以进行结构化日志记录失败", zap.Error(err))
		// 如果序列化失败，则回退到仅记录事件类型和详情
		eventFields = append(eventFields, zap.String("eventType", eventType), zap.Any("details", details))
	}

	// 添加任何额外传入的字段
	eventFields = append(eventFields, fields...)

	AuditLogger.Info("AuditEvent", eventFields...)
}

// ExampleDetail 结构体用于常见事件（可以扩展）

// AuthDetails 包含认证相关的详细信息
type AuthDetails struct {
	Username string `json:"username,omitempty"` // 用户名，可选
	Reason   string `json:"reason,omitempty"`   // 失败原因，可选
}

// SessionDetails 包含会话相关的详细信息
type SessionDetails struct {
	InitiatorRole string `json:"initiator_role,omitempty"` // 发起方角色，可选
	ResponderRole string `json:"responder_role,omitempty"` // 响应方角色，可选
}

// APDUDetails 包含APDU相关的详细信息
type APDUDetails struct {
	Direction string `json:"direction,omitempty"` // 方向，例如 "initiator_to_responder" 或 "responder_to_initiator" 或 "client_to_server", "server_to_client"
	Length    int    `json:"length,omitempty"`    // 长度，可选
}

// ErrorDetails 包含错误相关的详细信息
type ErrorDetails struct {
	ErrorCode    string `json:"error_code,omitempty"`    // 内部错误码，可选
	ErrorMessage string `json:"error_message"`           // 错误消息
	Component    string `json:"component,omitempty"`     // 组件，例如 "nfc_relay_hub", "auth_service"
	AffectedData string `json:"affected_data,omitempty"` // 受影响的数据，例如 APDU 十六进制, 消息内容，可选
}
