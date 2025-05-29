package global

import (
	"encoding/json"
	"fmt"
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

	// 【DEBUG】记录传入的参数
	GVA_LOG.Debug("【DEBUG-AUDIT】LogAuditEvent 被调用",
		zap.String("eventType", eventType),
		zap.Any("details", details),
		zap.Int("fields_count", len(fields)),
	)

	// 【DEBUG】检查 details 参数的类型
	switch d := details.(type) {
	case map[string]interface{}:
		GVA_LOG.Debug("【DEBUG-AUDIT】details 参数类型为 map[string]interface{}",
			zap.Int("map_size", len(d)),
		)
		// 检查关键字段
		for key, value := range d {
			GVA_LOG.Debug("【DEBUG-AUDIT】details map 中的键值对",
				zap.String("key", key),
				zap.Any("value", value),
				zap.String("value_type", fmt.Sprintf("%T", value)),
			)
		}
	case string:
		GVA_LOG.Debug("【DEBUG-AUDIT】details 参数类型为 string", zap.String("value", d))
	case nil:
		GVA_LOG.Debug("【DEBUG-AUDIT】details 参数为 nil")
	default:
		GVA_LOG.Debug("【DEBUG-AUDIT】details 参数类型为其他类型",
			zap.String("type", fmt.Sprintf("%T", details)),
		)
	}

	event := AuditEvent{
		Timestamp: time.Now().Format(time.RFC3339Nano),
		EventType: eventType,
		Details:   details,
	}

	// 【DEBUG】记录创建的 AuditEvent 结构体
	detailsBytes, _ := json.Marshal(details)
	GVA_LOG.Debug("【DEBUG-AUDIT】创建的 AuditEvent 结构体",
		zap.String("event.EventType", event.EventType),
		zap.Any("event.Details", event.Details),
		zap.String("details_json", string(detailsBytes)),
	)

	// 将 AuditEvent 转换为 zap.Field 切片
	// 这允许对事件对象本身进行结构化日志记录。
	var eventFields []zap.Field

	eventMap := make(map[string]interface{})
	eventBytes, err := json.Marshal(event)
	if err == nil {
		// 【DEBUG】记录序列化后的 JSON 字符串
		GVA_LOG.Debug("【DEBUG-AUDIT】序列化后的 AuditEvent JSON 字符串", zap.String("json", string(eventBytes)))

		// 【DEBUG】检查序列化后的 JSON 是否包含 details 字段
		var rawMap map[string]json.RawMessage
		if err := json.Unmarshal(eventBytes, &rawMap); err == nil {
			if detailsRaw, ok := rawMap["details"]; ok {
				GVA_LOG.Debug("【DEBUG-AUDIT】序列化后的 JSON 包含 details 字段",
					zap.String("details_raw", string(detailsRaw)),
				)

				// 尝试解析 details 字段
				var detailsMap map[string]interface{}
				if err := json.Unmarshal(detailsRaw, &detailsMap); err == nil {
					GVA_LOG.Debug("【DEBUG-AUDIT】details 字段成功解析为 map",
						zap.Any("detailsMap", detailsMap),
					)

					// 检查关键字段
					if actingClientID, ok := detailsMap["acting_client_id_in_details"]; ok {
						GVA_LOG.Debug("【DEBUG-AUDIT】details 字段中包含 acting_client_id_in_details",
							zap.Any("value", actingClientID),
						)
					} else {
						GVA_LOG.Debug("【DEBUG-AUDIT】details 字段中不包含 acting_client_id_in_details")
					}
				} else {
					GVA_LOG.Debug("【DEBUG-AUDIT】details 字段无法解析为 map",
						zap.Error(err),
					)
				}
			} else {
				GVA_LOG.Debug("【DEBUG-AUDIT】序列化后的 JSON 不包含 details 字段")
			}
		}

		_ = json.Unmarshal(eventBytes, &eventMap) //暂时忽略反序列化错误

		// 【DEBUG】记录 eventMap 的内容
		GVA_LOG.Debug("【DEBUG-AUDIT】eventMap 的内容", zap.Any("eventMap", eventMap))

		// 【DEBUG】检查 eventMap 中的 details 字段
		if detailsFromMap, ok := eventMap["details"]; ok {
			GVA_LOG.Debug("【DEBUG-AUDIT】eventMap 中包含 details 字段",
				zap.Any("detailsFromMap", detailsFromMap),
				zap.String("detailsFromMap_type", fmt.Sprintf("%T", detailsFromMap)),
			)

			// 如果 details 是 map[string]interface{} 类型，检查其中的字段
			if detailsMap, ok := detailsFromMap.(map[string]interface{}); ok {
				GVA_LOG.Debug("【DEBUG-AUDIT】eventMap 中的 details 字段是 map[string]interface{} 类型")

				// 检查关键字段
				if actingClientID, ok := detailsMap["acting_client_id_in_details"]; ok {
					GVA_LOG.Debug("【DEBUG-AUDIT】eventMap 中的 details 字段包含 acting_client_id_in_details",
						zap.Any("value", actingClientID),
					)
				} else {
					GVA_LOG.Debug("【DEBUG-AUDIT】eventMap 中的 details 字段不包含 acting_client_id_in_details")
				}
			} else {
				GVA_LOG.Debug("【DEBUG-AUDIT】eventMap 中的 details 字段不是 map[string]interface{} 类型")
			}
		} else {
			GVA_LOG.Debug("【DEBUG-AUDIT】eventMap 中不包含 details 字段")
		}

		for k, v := range eventMap {
			eventFields = append(eventFields, zap.Any(k, v))

			// 【DEBUG】记录每个添加的字段
			GVA_LOG.Debug("【DEBUG-AUDIT】添加字段到 eventFields", zap.String("key", k), zap.Any("value", v))
		}
	} else {
		GVA_LOG.Error("序列化 AuditEvent 以进行结构化日志记录失败", zap.Error(err))
		// 如果序列化失败，则回退到仅记录事件类型和详情
		eventFields = append(eventFields, zap.String("eventType", eventType), zap.Any("details", details))
	}

	// 添加任何额外传入的字段
	for _, field := range fields {
		// 【DEBUG】记录每个额外的字段
		GVA_LOG.Debug("【DEBUG-AUDIT】添加额外字段到 eventFields", zap.String("field_key", field.Key), zap.Any("field_type", field.Type))
	}
	eventFields = append(eventFields, fields...)

	// 【DEBUG】记录最终的 eventFields
	GVA_LOG.Debug("【DEBUG-AUDIT】最终的 eventFields",
		zap.Int("eventFields_count", len(eventFields)),
	)

	AuditLogger.Info("AuditEvent", eventFields...)

	// 【DEBUG】记录日志记录完成
	GVA_LOG.Debug("【DEBUG-AUDIT】AuditEvent 日志记录完成", zap.String("eventType", eventType))
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

// ComplianceViolation 包含合规违规的详细信息
type ComplianceViolation struct {
	UserID       string    `json:"user_id"`
	SessionID    string    `json:"session_id"`
	CommandClass string    `json:"command_class"`
	Reason       string    `json:"reason"`
	RiskLevel    string    `json:"risk_level"`
	Actions      string    `json:"actions"`
	Timestamp    time.Time `json:"timestamp"`
}
