package common

import (
	"time"
)

const (
	// RedisMqttRoleKeyPrefix is the prefix for storing MQTT client roles in Redis.
	// The full key is in the format "mqtt:role:<clientID>"
	RedisMqttRoleKeyPrefix = "mqtt:role:"

	// === EMQX WebHook 相关常量 ===

	// Redis键前缀 - 在线客户端管理
	RedisOnlineClientsSetKey = "nfc:clients:online" // 所有在线客户端集合

	// [V2.1 新增] 使用一个统一的Hash结构来存储所有客户端的心跳时间戳，替代旧的独立键模式
	RedisClientHeartbeatKey = "nfc:clients:heartbeats"
	// [V2.1 废弃] RedisClientHeartbeatPrefix is the prefix for storing client heartbeat info.
	RedisClientHeartbeatPrefix = "nfc:client:hb:" // 客户端心跳信息前缀: nfc:client:hb:<clientID>

	// Redis键前缀 - 配对槽位管理
	RedisPairingSlotPrefix = "nfc:pairing:" // 配对槽位前缀

	// EMQX WebHook事件类型
	EmqxEventClientConnected    = "client.connected"
	EmqxEventClientDisconnected = "client.disconnected"
	EmqxEventSessionCreated     = "session.created"
	EmqxEventSessionTerminated  = "session.terminated"

	// 客户端在线状态TTL
	ClientHeartbeatTTL   = 300 // 5分钟，心跳过期时间
	ClientOnlineCheckTTL = 60  // 1分钟，在线状态检查间隔

	// 业务错误码
	BizErrCodeRoleConflict   = "ROLE_CONFLICT"
	BizErrCodeClientOffline  = "CLIENT_OFFLINE"
	BizErrCodePairingTimeout = "PAIRING_TIMEOUT"

	// 配对冲突操作类型
	ActionForceTakeover = "force_takeover"
	ActionWaitRetry     = "wait_retry"
	ActionCancel        = "cancel"

	// Redis Key for storing detailed client status as a Hash.
	RedisClientStatusHashKey = "mqtt:status:%s"

	// Redis Key for caching transaction status
	RedisTransactionStatusKey = "transaction:%s:status"

	// 交易状态常量
	TransactionStatusPending = "pending"
	TransactionStatusActive  = "active"

	MqttAuthSuccessKeyPrefix = "mqtt:authed:"
	MqttAuthSuccessKeyTTL    = 60 * time.Second // 凭证有效期60秒
)
