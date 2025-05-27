package admin_response

// DashboardStatsData 定义了仪表盘统计数据的结构
type DashboardStatsData struct {
	HubStatus              string                        `json:"hub_status"`                // Hub 运行状态 ("online")
	ActiveConnections      int                           `json:"active_connections"`        // 当前活动 WebSocket 连接总数
	ActiveSessions         int                           `json:"active_sessions"`           // 当前 NFC 中继活动会话总数
	TotalApduRelayed       float64                       `json:"total_apdu_relayed"`        // APDU 消息中继总数 (所有方向)
	ApduRelayedByDirection map[string]float64            `json:"apdu_relayed_by_direction"` // APDU 消息按方向统计 (upstream, downstream)
	TotalApduErrors        float64                       `json:"total_apdu_errors"`         // APDU 转发失败总数 (来自 ApduRelayErrors metric)
	TotalHubErrors         float64                       `json:"total_hub_errors"`          // Hub 内部错误总数 (来自 HubErrors metric)
	SessionTerminations    map[string]float64            `json:"session_terminations"`      // 会话终止原因统计
	AuthEvents             map[string]map[string]float64 `json:"auth_events"`               // 认证事件统计 (map[type]map[reason]count)
}
