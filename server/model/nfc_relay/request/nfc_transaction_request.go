package request

import (
	"github.com/flipped-aurora/gin-vue-admin/server/model/common/request"
)

// CreateTransactionRequest 创建交易请求
type CreateTransactionRequest struct {
	TransmitterClientID string                 `json:"transmitter_client_id" binding:"required" validate:"required,min=1,max=128" example:"admin-transmitter-001"` // 传卡端客户端ID
	ReceiverClientID    string                 `json:"receiver_client_id" binding:"required" validate:"required,min=1,max=128" example:"user-receiver-001"`        // 收卡端客户端ID
	CardType            string                 `json:"card_type" validate:"max=50" example:"mifare_classic"`                                                       // 卡片类型
	Description         string                 `json:"description" validate:"max=500" example:"业务卡片中继交易"`                                                          // 交易描述
	TimeoutSeconds      int                    `json:"timeout_seconds" validate:"min=30,max=3600" example:"120"`                                                   // 超时时间(秒) 30秒-1小时
	Tags                string                 `json:"tags" validate:"max=500" example:"test,demo"`                                                                // 标签
	Metadata            map[string]interface{} `json:"metadata" swaggertype:"object,string"`                                                                       // 扩展元数据
}

// UpdateTransactionStatusRequest 更新交易状态请求
type UpdateTransactionStatusRequest struct {
	TransactionID string                 `json:"transaction_id" binding:"required" validate:"required" example:"txn_1234567890"`                                                           // 交易ID
	Status        string                 `json:"status" binding:"required" validate:"required,oneof=pending active processing completed failed cancelled timeout paused" example:"active"` // 新状态
	Reason        string                 `json:"reason" validate:"max=100" example:"用户操作"`                                                                                                 // 状态变更原因
	ErrorMsg      string                 `json:"error_msg" validate:"max=500" example:""`                                                                                                  // 错误信息
	Metadata      map[string]interface{} `json:"metadata" swaggertype:"object,string"`                                                                                                     // 扩展元数据
}

// GetTransactionRequest 获取交易详情请求
type GetTransactionRequest struct {
	TransactionID string `json:"transaction_id" uri:"transaction_id" binding:"required" validate:"required" example:"txn_1234567890"` // 交易ID
	IncludeAPDU   bool   `json:"include_apdu" form:"include_apdu" example:"true"`                                                     // 是否包含APDU消息
}

// GetTransactionListRequest 获取交易列表请求
type GetTransactionListRequest struct {
	request.PageInfo

	// 过滤条件
	TransmitterClientID string `json:"transmitter_client_id" form:"transmitter_client_id" example:"admin-transmitter-001"` // 传卡端客户端ID
	ReceiverClientID    string `json:"receiver_client_id" form:"receiver_client_id" example:"user-receiver-001"`           // 收卡端客户端ID
	Status              string `json:"status" form:"status" example:"completed"`                                           // 交易状态
	CardType            string `json:"card_type" form:"card_type" example:"mifare_classic"`                                // 卡片类型
	CreatedBy           string `json:"created_by" form:"created_by" example:""`                                            // 创建用户ID

	// 时间范围
	StartTime string `json:"start_time" form:"start_time" example:"2024-01-01 00:00:00"` // 开始时间
	EndTime   string `json:"end_time" form:"end_time" example:"2024-01-02 00:00:00"`     // 结束时间

	// 搜索关键词
	Keyword string `json:"keyword" form:"keyword" example:"支付"` // 关键词搜索(描述、标签)

	// 排序
	OrderBy string `json:"order_by" form:"order_by" example:"created_at"` // 排序字段
	Order   string `json:"order" form:"order" example:"desc"`             // 排序方向(asc/desc)
}

// SendAPDURequest 发送APDU消息请求
type SendAPDURequest struct {
	TransactionID  string                 `json:"transaction_id" binding:"required" validate:"required" example:"txn_1234567890"`                          // 交易ID
	Direction      string                 `json:"direction" binding:"required" validate:"required,oneof=to_receiver to_transmitter" example:"to_receiver"` // 消息方向
	APDUHex        string                 `json:"apdu_hex" binding:"required" validate:"required,hexadecimal" example:"00A4040008A000000003000000"`        // APDU十六进制数据
	SequenceNumber int                    `json:"sequence_number" binding:"required" validate:"required,min=1" example:"1"`                                // 序列号
	Priority       string                 `json:"priority" validate:"oneof=high normal low" example:"normal"`                                              // 优先级
	MessageType    string                 `json:"message_type" validate:"max=20" example:"select_application"`                                             // 消息类型
	Metadata       map[string]interface{} `json:"metadata" swaggertype:"object,string"`                                                                    // 扩展元数据
}

// GetAPDUListRequest 获取APDU消息列表请求
type GetAPDUListRequest struct {
	request.PageInfo

	TransactionID string `json:"transaction_id" form:"transaction_id" binding:"required" validate:"required" example:"txn_1234567890"` // 交易ID
	Direction     string `json:"direction" form:"direction" example:"to_receiver"`                                                     // 消息方向
	Status        string `json:"status" form:"status" example:"sent"`                                                                  // 消息状态
	Priority      string `json:"priority" form:"priority" example:"normal"`                                                            // 优先级

	// 时间范围
	StartTime string `json:"start_time" form:"start_time" example:"2024-01-01 00:00:00"` // 开始时间
	EndTime   string `json:"end_time" form:"end_time" example:"2024-01-02 00:00:00"`     // 结束时间
}

// DeleteTransactionRequest 删除交易请求
type DeleteTransactionRequest struct {
	TransactionID string `json:"transaction_id" uri:"transaction_id" binding:"required" validate:"required" example:"txn_1234567890"` // 交易ID
	Force         bool   `json:"force" form:"force" example:"false"`                                                                  // 强制删除(包括未完成的交易)
}

// GetStatisticsRequest 获取统计信息请求
type GetStatisticsRequest struct {
	// 时间范围
	StartDate string `json:"start_date" form:"start_date" binding:"required" validate:"required" example:"2024-01-01"` // 开始日期
	EndDate   string `json:"end_date" form:"end_date" binding:"required" validate:"required" example:"2024-01-31"`     // 结束日期

	// 分组方式
	GroupBy string `json:"group_by" form:"group_by" validate:"oneof=day week month" example:"day"` // 分组方式

	// 过滤条件
	CardType string `json:"card_type" form:"card_type" example:"mifare_classic"` // 卡片类型
	Status   string `json:"status" form:"status" example:"completed"`            // 交易状态
}

// BatchUpdateTransactionRequest 批量更新交易请求
type BatchUpdateTransactionRequest struct {
	TransactionIDs []string               `json:"transaction_ids" binding:"required" validate:"required,min=1" example:"[\"txn_1\",\"txn_2\"]"`     // 交易ID列表
	Status         string                 `json:"status" binding:"required" validate:"required,oneof=cancelled timeout paused" example:"cancelled"` // 新状态(仅支持取消、超时和暂停)
	Reason         string                 `json:"reason" validate:"max=100" example:"批量操作"`                                                         // 操作原因
	Metadata       map[string]interface{} `json:"metadata" swaggertype:"object,string"`                                                             // 扩展元数据
}

// ExportTransactionRequest 导出交易请求
type ExportTransactionRequest struct {
	// 导出格式
	Format string `json:"format" form:"format" validate:"oneof=excel csv json" example:"excel"` // 导出格式

	// 过滤条件(同GetTransactionListRequest)
	TransmitterClientID string `json:"transmitter_client_id" form:"transmitter_client_id"`
	ReceiverClientID    string `json:"receiver_client_id" form:"receiver_client_id"`
	Status              string `json:"status" form:"status"`
	CardType            string `json:"card_type" form:"card_type"`
	CreatedBy           string `json:"created_by" form:"created_by"`
	StartTime           string `json:"start_time" form:"start_time"`
	EndTime             string `json:"end_time" form:"end_time"`
	Keyword             string `json:"keyword" form:"keyword"`

	// 导出选项
	IncludeAPDU bool `json:"include_apdu" form:"include_apdu" example:"false"` // 是否包含APDU消息
	MaxRecords  int  `json:"max_records" form:"max_records" example:"10000"`   // 最大导出记录数
}

// ClientHeartbeatRequest 客户端心跳请求
type ClientHeartbeatRequest struct {
	ClientID     string                 `json:"client_id" binding:"required" validate:"required" example:"admin-transmitter-001"`             // 客户端ID
	Role         string                 `json:"role" binding:"required" validate:"required,oneof=transmitter receiver" example:"transmitter"` // 客户端角色
	Status       string                 `json:"status" validate:"oneof=online offline busy" example:"online"`                                 // 客户端状态
	Version      string                 `json:"version" example:"1.0.0"`                                                                      // 客户端版本
	DeviceInfo   map[string]interface{} `json:"device_info" swaggertype:"object,string"`                                                      // 设备信息
	Capabilities []string               `json:"capabilities" example:"[\"mifare_classic\",\"iso14443a\"]"`                                    // 支持的功能
}

// GetOnlineClientsRequest 获取在线客户端请求
type GetOnlineClientsRequest struct {
	request.PageInfo

	Role         string `json:"role" form:"role" validate:"oneof=transmitter receiver" example:"transmitter"` // 客户端角色
	Status       string `json:"status" form:"status" example:"online"`                                        // 客户端状态
	Keyword      string `json:"keyword" form:"keyword" example:"admin"`                                       // 关键词搜索
	LastSeenFrom string `json:"last_seen_from" form:"last_seen_from" example:"2024-01-01 00:00:00"`           // 最后活跃时间范围(开始)
	LastSeenTo   string `json:"last_seen_to" form:"last_seen_to" example:"2024-01-02 00:00:00"`               // 最后活跃时间范围(结束)
}
