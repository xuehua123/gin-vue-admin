package response

import (
	"time"

	"github.com/flipped-aurora/gin-vue-admin/server/model/nfc_relay"
)

// CreateTransactionResponse 创建交易响应
type CreateTransactionResponse struct {
	TransactionID       string    `json:"transaction_id"`        // 交易ID
	Status              string    `json:"status"`                // 交易状态
	TransmitterClientID string    `json:"transmitter_client_id"` // 传卡端客户端ID
	ReceiverClientID    string    `json:"receiver_client_id"`    // 收卡端客户端ID
	CardType            string    `json:"card_type"`             // 卡片类型
	CreatedAt           time.Time `json:"created_at"`            // 创建时间
	ExpiresAt           time.Time `json:"expires_at"`            // 过期时间
}

// TransactionDetailResponse 交易详情响应
type TransactionDetailResponse struct {
	nfc_relay.NFCTransaction

	// 扩展信息
	Statistics TransactionStatistics `json:"statistics"` // 统计信息
	Timeline   []TransactionEvent    `json:"timeline"`   // 时间线
}

// TransactionStatistics 交易统计信息
type TransactionStatistics struct {
	APDUMessageCount      int        `json:"apdu_message_count"`       // APDU消息数量
	TotalProcessingTimeMs int        `json:"total_processing_time_ms"` // 总处理时间(毫秒)
	AverageResponseTimeMs float64    `json:"average_response_time_ms"` // 平均响应时间(毫秒)
	SuccessRate           float64    `json:"success_rate"`             // 成功率
	ErrorCount            int        `json:"error_count"`              // 错误数量
	LastActivityAt        *time.Time `json:"last_activity_at"`         // 最后活动时间
}

// TransactionEvent 交易事件
type TransactionEvent struct {
	Timestamp   time.Time              `json:"timestamp"`   // 时间戳
	EventType   string                 `json:"event_type"`  // 事件类型
	Description string                 `json:"description"` // 描述
	Actor       string                 `json:"actor"`       // 操作者
	Metadata    map[string]interface{} `json:"metadata"`    // 扩展数据
}

// TransactionListResponse 交易列表响应
type TransactionListResponse struct {
	List     []TransactionListItem `json:"list"`      // 交易列表
	Total    int64                 `json:"total"`     // 总数
	Page     int                   `json:"page"`      // 当前页
	PageSize int                   `json:"page_size"` // 页大小
	Summary  TransactionSummary    `json:"summary"`   // 汇总信息
}

// TransactionListItem 交易列表项
type TransactionListItem struct {
	ID                    uint       `json:"id"`
	TransactionID         string     `json:"transaction_id"`
	TransmitterClientID   string     `json:"transmitter_client_id"`
	ReceiverClientID      string     `json:"receiver_client_id"`
	Status                string     `json:"status"`
	CardType              string     `json:"card_type"`
	Description           string     `json:"description"`
	APDUCount             int        `json:"apdu_count"`
	TotalProcessingTimeMs int        `json:"total_processing_time_ms"`
	CreatedAt             time.Time  `json:"created_at"`
	StartedAt             *time.Time `json:"started_at"`
	CompletedAt           *time.Time `json:"completed_at"`
	ExpiresAt             *time.Time `json:"expires_at"`
	EndReason             string     `json:"end_reason"`
	Tags                  string     `json:"tags"`
}

// TransactionSummary 交易汇总信息
type TransactionSummary struct {
	TotalCount          int     `json:"total_count"`           // 总数量
	CompletedCount      int     `json:"completed_count"`       // 已完成数量
	FailedCount         int     `json:"failed_count"`          // 失败数量
	PendingCount        int     `json:"pending_count"`         // 等待数量
	ActiveCount         int     `json:"active_count"`          // 活跃数量
	SuccessRate         float64 `json:"success_rate"`          // 成功率
	AverageProcessingMs float64 `json:"average_processing_ms"` // 平均处理时间
}

// UpdateTransactionStatusResponse 更新交易状态响应
type UpdateTransactionStatusResponse struct {
	TransactionID  string    `json:"transaction_id"`  // 交易ID
	Status         string    `json:"status"`          // 新状态
	PreviousStatus string    `json:"previous_status"` // 原状态
	UpdatedAt      time.Time `json:"updated_at"`      // 更新时间
	Reason         string    `json:"reason"`          // 变更原因
}

// SendAPDUResponse 发送APDU消息响应
type SendAPDUResponse struct {
	MessageID      uint      `json:"message_id"`      // 消息ID
	TransactionID  string    `json:"transaction_id"`  // 交易ID
	Direction      string    `json:"direction"`       // 消息方向
	SequenceNumber int       `json:"sequence_number"` // 序列号
	Status         string    `json:"status"`          // 消息状态
	SentAt         time.Time `json:"sent_at"`         // 发送时间
}

// APDUMessageListResponse APDU消息列表响应
type APDUMessageListResponse struct {
	List     []APDUMessageItem `json:"list"`      // 消息列表
	Total    int64             `json:"total"`     // 总数
	Page     int               `json:"page"`      // 当前页
	PageSize int               `json:"page_size"` // 页大小
}

// APDUMessageItem APDU消息项
type APDUMessageItem struct {
	ID             uint       `json:"id"`
	TransactionID  string     `json:"transaction_id"`
	Direction      string     `json:"direction"`
	APDUHex        string     `json:"apdu_hex"`
	SequenceNumber int        `json:"sequence_number"`
	Priority       string     `json:"priority"`
	MessageType    string     `json:"message_type"`
	Status         string     `json:"status"`
	SentAt         *time.Time `json:"sent_at"`
	ReceivedAt     *time.Time `json:"received_at"`
	ResponseTime   int        `json:"response_time_ms"`
	ErrorMsg       string     `json:"error_msg"`
	CreatedAt      time.Time  `json:"created_at"`
}

// TransactionStatisticsResponse 交易统计响应
type TransactionStatisticsResponse struct {
	DateRange     DateRange           `json:"date_range"`     // 日期范围
	Summary       StatisticsSummary   `json:"summary"`        // 汇总数据
	DailyStats    []DailyStatistics   `json:"daily_stats"`    // 每日统计
	ChartData     StatisticsChartData `json:"chart_data"`     // 图表数据
	TopClients    []ClientStatistics  `json:"top_clients"`    // 客户端统计
	ErrorAnalysis ErrorAnalysis       `json:"error_analysis"` // 错误分析
}

// DateRange 日期范围
type DateRange struct {
	StartDate string `json:"start_date"` // 开始日期
	EndDate   string `json:"end_date"`   // 结束日期
	Days      int    `json:"days"`       // 天数
}

// StatisticsSummary 统计汇总
type StatisticsSummary struct {
	TotalTransactions      int     `json:"total_transactions"`       // 总交易数
	SuccessfulTransactions int     `json:"successful_transactions"`  // 成功交易数
	FailedTransactions     int     `json:"failed_transactions"`      // 失败交易数
	SuccessRate            float64 `json:"success_rate"`             // 成功率
	TotalAPDUMessages      int     `json:"total_apdu_messages"`      // 总APDU消息数
	AverageProcessingMs    float64 `json:"average_processing_ms"`    // 平均处理时间
	TotalProcessingTimeMs  int64   `json:"total_processing_time_ms"` // 总处理时间
}

// DailyStatistics 每日统计
type DailyStatistics struct {
	Date                   string  `json:"date"`                    // 日期
	TotalTransactions      int     `json:"total_transactions"`      // 总交易数
	SuccessfulTransactions int     `json:"successful_transactions"` // 成功交易数
	FailedTransactions     int     `json:"failed_transactions"`     // 失败交易数
	SuccessRate            float64 `json:"success_rate"`            // 成功率
	AverageProcessingMs    float64 `json:"average_processing_ms"`   // 平均处理时间
	TotalAPDUMessages      int     `json:"total_apdu_messages"`     // APDU消息数
}

// StatisticsChartData 图表数据
type StatisticsChartData struct {
	TransactionTrend     []ChartPoint   `json:"transaction_trend"`      // 交易趋势
	SuccessRateTrend     []ChartPoint   `json:"success_rate_trend"`     // 成功率趋势
	ResponseTimeTrend    []ChartPoint   `json:"response_time_trend"`    // 响应时间趋势
	StatusDistribution   []PieChartItem `json:"status_distribution"`    // 状态分布
	CardTypeDistribution []PieChartItem `json:"card_type_distribution"` // 卡片类型分布
}

// ChartPoint 图表数据点
type ChartPoint struct {
	X string  `json:"x"` // X轴值(通常是时间)
	Y float64 `json:"y"` // Y轴值
}

// PieChartItem 饼图项
type PieChartItem struct {
	Name  string  `json:"name"`  // 名称
	Value float64 `json:"value"` // 值
	Count int     `json:"count"` // 数量
}

// ClientStatistics 客户端统计
type ClientStatistics struct {
	ClientID         string     `json:"client_id"`         // 客户端ID
	Role             string     `json:"role"`              // 角色
	TransactionCount int        `json:"transaction_count"` // 交易数量
	SuccessRate      float64    `json:"success_rate"`      // 成功率
	AvgProcessingMs  float64    `json:"avg_processing_ms"` // 平均处理时间
	LastActiveAt     *time.Time `json:"last_active_at"`    // 最后活跃时间
}

// ErrorAnalysis 错误分析
type ErrorAnalysis struct {
	TotalErrors    int                `json:"total_errors"`     // 总错误数
	ErrorRate      float64            `json:"error_rate"`       // 错误率
	CommonErrors   []CommonError      `json:"common_errors"`    // 常见错误
	ErrorTrend     []ChartPoint       `json:"error_trend"`      // 错误趋势
	ErrorsByClient []ClientErrorStats `json:"errors_by_client"` // 按客户端分组的错误
}

// CommonError 常见错误
type CommonError struct {
	ErrorType   string  `json:"error_type"`  // 错误类型
	Count       int     `json:"count"`       // 出现次数
	Percentage  float64 `json:"percentage"`  // 占比
	Description string  `json:"description"` // 描述
}

// ClientErrorStats 客户端错误统计
type ClientErrorStats struct {
	ClientID   string  `json:"client_id"`   // 客户端ID
	Role       string  `json:"role"`        // 角色
	ErrorCount int     `json:"error_count"` // 错误数量
	ErrorRate  float64 `json:"error_rate"`  // 错误率
}

// OnlineClientsResponse 在线客户端响应
type OnlineClientsResponse struct {
	List     []OnlineClientItem   `json:"list"`      // 客户端列表
	Total    int64                `json:"total"`     // 总数
	Page     int                  `json:"page"`      // 当前页
	PageSize int                  `json:"page_size"` // 页大小
	Summary  OnlineClientsSummary `json:"summary"`   // 汇总信息
}

// OnlineClientItem 在线客户端项
type OnlineClientItem struct {
	ClientID     string                 `json:"client_id"`    // 客户端ID
	Role         string                 `json:"role"`         // 角色
	Status       string                 `json:"status"`       // 状态
	Version      string                 `json:"version"`      // 版本
	LastSeenAt   time.Time              `json:"last_seen_at"` // 最后活跃时间
	ConnectedAt  time.Time              `json:"connected_at"` // 连接时间
	Duration     int64                  `json:"duration"`     // 在线时长(秒)
	DeviceInfo   map[string]interface{} `json:"device_info"`  // 设备信息
	Capabilities []string               `json:"capabilities"` // 功能列表

	// 统计信息
	TransactionCount int     `json:"transaction_count"` // 交易数量
	SuccessRate      float64 `json:"success_rate"`      // 成功率
	AvgResponseMs    float64 `json:"avg_response_ms"`   // 平均响应时间
}

// OnlineClientsSummary 在线客户端汇总
type OnlineClientsSummary struct {
	TotalOnline        int `json:"total_online"`        // 总在线数
	TransmitterCount   int `json:"transmitter_count"`   // 传卡端数量
	ReceiverCount      int `json:"receiver_count"`      // 收卡端数量
	ActiveTransactions int `json:"active_transactions"` // 活跃交易数
	BusyClients        int `json:"busy_clients"`        // 忙碌客户端数
}

// BatchOperationResponse 批量操作响应
type BatchOperationResponse struct {
	Total         int          `json:"total"`          // 总数
	SuccessCount  int          `json:"success_count"`  // 成功数量
	FailureCount  int          `json:"failure_count"`  // 失败数量
	SuccessIDs    []string     `json:"success_ids"`    // 成功的ID列表
	FailureErrors []BatchError `json:"failure_errors"` // 失败错误列表
}

// BatchError 批量操作错误
type BatchError struct {
	ID    string `json:"id"`    // ID
	Error string `json:"error"` // 错误信息
}

// ExportResponse 导出响应
type ExportResponse struct {
	Filename    string    `json:"filename"`     // 文件名
	FileSize    int64     `json:"file_size"`    // 文件大小
	RecordCount int       `json:"record_count"` // 记录数量
	DownloadURL string    `json:"download_url"` // 下载链接
	ExpiresAt   time.Time `json:"expires_at"`   // 过期时间
}

// OccupyingDeviceInfo 占用设备信息
type OccupyingDeviceInfo struct {
	ClientID       string                 `json:"client_id"`       // 客户端ID
	DeviceModel    string                 `json:"device_model"`    // 设备型号
	OSVersion      string                 `json:"os_version"`      // 操作系统版本
	UserAgent      string                 `json:"user_agent"`      // 用户代理
	IPAddress      string                 `json:"ip_address"`      // IP地址
	LastSeen       string                 `json:"last_seen"`       // 最后活跃时间
	OccupiedSince  int64                  `json:"occupied_since"`  // 占用开始时间戳
	OnlineDuration int64                  `json:"online_duration"` // 在线时长(秒)
	DeviceInfo     map[string]interface{} `json:"device_info"`     // 扩展设备信息
}

// PairingConflictResponse 配对冲突响应
type PairingConflictResponse struct {
	ConflictRole        string              `json:"conflict_role"`         // 冲突的角色
	ExistingClientID    string              `json:"existing_client_id"`    // 已存在的客户端ID
	OccupyingDevice     OccupyingDeviceInfo `json:"occupying_device"`      // 占用设备信息
	Message             string              `json:"message"`               // 冲突消息
	AvailableActions    []string            `json:"available_actions"`     // 可用操作
	SuggestedRetryAfter int                 `json:"suggested_retry_after"` // 建议重试时间(秒)
}

// PairingRegisterResponse defines the successful response for a pairing registration.
type PairingRegisterResponse struct {
	ClientID   string `json:"client_id"`
	MqttToken  string `json:"mqtt_token"`
	MqttServer string `json:"mqtt_server"`
}

// PairingConflictConstants 配对冲突常量
const (
	ErrorCodePairingRoleConflict = "PAIRING_ROLE_CONFLICT"
	ErrorCodePairingTimeout      = "PAIRING_TIMEOUT"
	ErrorCodePairingInvalidRole  = "PAIRING_INVALID_ROLE"

	ErrorTypePairingConflict = "pairing_conflict"
	ErrorTypePairingTimeout  = "pairing_timeout"
	ErrorTypeValidation      = "validation_error"

	ActionTypeForceTakeover = "force_takeover"
	ActionTypeWaitRetry     = "wait_retry"
	ActionTypeCancel        = "cancel"
)
