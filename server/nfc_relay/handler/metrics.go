package handler

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

var (
	ActiveConnections = promauto.NewGauge(
		prometheus.GaugeOpts{
			Name: "nfc_relay_active_connections_total",
			Help: "NFC 中继 Hub 的活动 WebSocket 连接总数。",
		},
	)

	ActiveSessions = promauto.NewGauge(
		prometheus.GaugeOpts{
			Name: "nfc_relay_active_sessions_total",
			Help: "活动 NFC 中继会话总数。",
		},
	)

	ApduMessagesRelayed = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "nfc_relay_apdu_messages_relayed_total",
			Help: "按方向划分的中继 APDU 消息总数。",
		},
		[]string{"direction", "session_id"}, // direction: upstream (上行), downstream (下行)
	)

	ApduRelayErrors = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "nfc_relay_apdu_relay_errors_total",
			Help: "按方向划分的 APDU 中继错误总数。",
		},
		[]string{"direction", "session_id"},
	)

	SessionTerminations = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "nfc_relay_session_terminations_total",
			Help: "按原因划分的 NFC 中继会话终止总数。",
		},
		[]string{"reason"}, // 例如：client_request (客户端请求), client_disconnect (客户端断开连接), timeout (超时), apdu_error (APDU错误)
	)

	HubErrors = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "nfc_relay_hub_errors_total",
			Help: "按类型或代码划分的 Hub 常规错误总数。",
		},
		[]string{"error_code", "component"},
	)

	AuthEvents = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "nfc_relay_auth_events_total",
			Help: "按类型（成功/失败）划分的认证事件总数。",
		},
		[]string{"type", "reason"}, // type: success (成功), failure (失败). reason: 失败原因
	)
)

// 注意：如果需要更详细的监控，可以稍后添加用于延迟的直方图或用于通道填充率的仪表盘，
// 同时要考虑到潜在的性能开销。
