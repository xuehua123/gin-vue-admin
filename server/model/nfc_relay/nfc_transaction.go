package nfc_relay

import (
	"time"

	"github.com/flipped-aurora/gin-vue-admin/server/global"
	"github.com/google/uuid"
	"gorm.io/datatypes"
)

// NFCTransaction NFC交易表
type NFCTransaction struct {
	global.GVA_MODEL

	// 基础信息
	TransactionID string `json:"transaction_id" gorm:"type:varchar(64);uniqueIndex;not null;comment:交易唯一标识"`

	// 客户端信息
	TransmitterClientID string `json:"transmitter_client_id" gorm:"type:varchar(128);not null;index;comment:传卡端客户端ID"`
	ReceiverClientID    string `json:"receiver_client_id" gorm:"type:varchar(128);not null;index;comment:收卡端客户端ID"`

	// 交易状态
	Status string `json:"status" gorm:"type:varchar(20);not null;default:'creating';index;comment:交易状态"`

	// 业务信息
	CardType    string `json:"card_type" gorm:"type:varchar(50);comment:卡片类型"`
	Description string `json:"description" gorm:"type:text;comment:交易描述"`

	// 用户信息
	CreatedBy uuid.UUID `json:"created_by" gorm:"type:char(36);index;comment:创建用户ID"`
	UpdatedBy uuid.UUID `json:"updated_by" gorm:"type:char(36);comment:更新用户ID"`

	// 时间信息
	StartedAt   *time.Time `json:"started_at" gorm:"comment:开始时间"`
	CompletedAt *time.Time `json:"completed_at" gorm:"comment:完成时间"`
	ExpiresAt   *time.Time `json:"expires_at" gorm:"index;comment:过期时间"`

	// 结果信息
	EndReason string `json:"end_reason" gorm:"type:varchar(100);comment:结束原因"`
	ErrorMsg  string `json:"error_msg" gorm:"type:text;comment:错误信息"`

	// 统计信息
	APDUCount             int `json:"apdu_count" gorm:"default:0;comment:APDU消息数量"`
	TotalProcessingTimeMs int `json:"total_processing_time_ms" gorm:"comment:总处理时间(毫秒)"`
	AverageResponseTimeMs int `json:"average_response_time_ms" gorm:"comment:平均响应时间(毫秒)"`

	// 扩展信息
	Metadata datatypes.JSON `json:"metadata" gorm:"type:json;comment:扩展元数据" swaggertype:"object"`
	Tags     string         `json:"tags" gorm:"type:varchar(500);comment:标签(逗号分隔)"`

	// 关联关系
	APDUMessages []NFCAPDUMessage `json:"apdu_messages" gorm:"foreignKey:TransactionID;references:TransactionID"`
}

// TableName 指定表名
func (NFCTransaction) TableName() string {
	return "nfc_transactions"
}

// NFCAPDUMessage APDU消息表
type NFCAPDUMessage struct {
	global.GVA_MODEL

	// 关联信息
	TransactionID string `json:"transaction_id" gorm:"type:varchar(64);not null;index;comment:交易ID"`

	// 消息信息
	Direction      string `json:"direction" gorm:"type:varchar(20);not null;comment:消息方向(to_receiver/to_transmitter)"`
	APDUHex        string `json:"apdu_hex" gorm:"type:text;not null;comment:APDU十六进制数据"`
	SequenceNumber int    `json:"sequence_number" gorm:"not null;comment:序列号"`

	// 优先级和类型
	Priority    string `json:"priority" gorm:"type:varchar(10);default:'normal';comment:消息优先级"`
	MessageType string `json:"message_type" gorm:"type:varchar(20);comment:消息类型"`

	// 时间信息
	SentAt       *time.Time `json:"sent_at" gorm:"comment:发送时间"`
	ReceivedAt   *time.Time `json:"received_at" gorm:"comment:接收时间"`
	ResponseTime int        `json:"response_time_ms" gorm:"comment:响应时间(毫秒)"`

	// 状态信息
	Status   string `json:"status" gorm:"type:varchar(20);default:'pending';comment:消息状态"`
	ErrorMsg string `json:"error_msg" gorm:"type:text;comment:错误信息"`

	// 扩展信息
	Metadata datatypes.JSON `json:"metadata" gorm:"type:json;comment:扩展元数据" swaggertype:"object"`
}

// TableName 指定表名
func (NFCAPDUMessage) TableName() string {
	return "nfc_apdu_messages"
}

// NFCTransactionStatistics 交易统计表
type NFCTransactionStatistics struct {
	ID   uint      `json:"id" gorm:"primarykey"`
	Date time.Time `json:"date" gorm:"type:date;uniqueIndex;not null;comment:统计日期"`

	// 交易数量统计
	TotalTransactions      int `json:"total_transactions" gorm:"default:0;comment:总交易数"`
	SuccessfulTransactions int `json:"successful_transactions" gorm:"default:0;comment:成功交易数"`
	FailedTransactions     int `json:"failed_transactions" gorm:"default:0;comment:失败交易数"`
	TimeoutTransactions    int `json:"timeout_transactions" gorm:"default:0;comment:超时交易数"`
	CancelledTransactions  int `json:"cancelled_transactions" gorm:"default:0;comment:取消交易数"`

	// APDU消息统计
	TotalAPDUMessages int `json:"total_apdu_messages" gorm:"default:0;comment:总APDU消息数"`

	// 性能统计
	AverageProcessingTimeMs float64 `json:"average_processing_time_ms" gorm:"comment:平均处理时间(毫秒)"`
	MinProcessingTimeMs     int     `json:"min_processing_time_ms" gorm:"comment:最小处理时间(毫秒)"`
	MaxProcessingTimeMs     int     `json:"max_processing_time_ms" gorm:"comment:最大处理时间(毫秒)"`

	// 错误统计
	ErrorCount   int    `json:"error_count" gorm:"default:0;comment:错误数量"`
	CommonErrors string `json:"common_errors" gorm:"type:text;comment:常见错误(JSON)"`

	// 扩展信息
	Metadata datatypes.JSON `json:"metadata" gorm:"type:json;comment:扩展统计数据" swaggertype:"object"`

	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// TableName 指定表名
func (NFCTransactionStatistics) TableName() string {
	return "nfc_transaction_statistics"
}

// 常量定义
const (
	// 交易状态
	StatusCreating   = "creating"   // 创建中
	StatusPending    = "pending"    // 等待中
	StatusActive     = "active"     // 活跃中
	StatusProcessing = "processing" // 处理中
	StatusCompleted  = "completed"  // 已完成
	StatusFailed     = "failed"     // 失败
	StatusCancelled  = "cancelled"  // 已取消
	StatusTimeout    = "timeout"    // 超时

	// 消息方向
	DirectionToReceiver    = "to_receiver"    // 发送到收卡端
	DirectionToTransmitter = "to_transmitter" // 发送到传卡端

	// 消息优先级
	PriorityHigh   = "high"
	PriorityNormal = "normal"
	PriorityLow    = "low"

	// 消息状态
	MessageStatusPending   = "pending"   // 等待发送
	MessageStatusSent      = "sent"      // 已发送
	MessageStatusReceived  = "received"  // 已接收
	MessageStatusProcessed = "processed" // 已处理
	MessageStatusFailed    = "failed"    // 失败
)

// 状态转换验证
var ValidStatusTransitions = map[string][]string{
	StatusCreating:   {StatusPending, StatusFailed, StatusCancelled},
	StatusPending:    {StatusActive, StatusFailed, StatusCancelled, StatusTimeout},
	StatusActive:     {StatusProcessing, StatusCompleted, StatusFailed, StatusCancelled, StatusTimeout},
	StatusProcessing: {StatusCompleted, StatusFailed, StatusCancelled, StatusTimeout},
	StatusCompleted:  {}, // 终态
	StatusFailed:     {}, // 终态
	StatusCancelled:  {}, // 终态
	StatusTimeout:    {}, // 终态
}

// IsValidStatusTransition 验证状态转换是否有效
func IsValidStatusTransition(currentStatus, newStatus string) bool {
	validTransitions, exists := ValidStatusTransitions[currentStatus]
	if !exists {
		return false
	}

	for _, validStatus := range validTransitions {
		if validStatus == newStatus {
			return true
		}
	}
	return false
}
