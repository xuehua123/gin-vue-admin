package response

import "time"

// DashboardStatsResponse 仪表盘统计数据响应
type DashboardStatsResponse struct {
	HubStatus             string       `json:"hub_status"`               // Hub状态
	ActiveConnections     int          `json:"active_connections"`       // 活动连接数
	ActiveSessions        int          `json:"active_sessions"`          // 活动会话数
	ApduRelayedLastMinute int64        `json:"apdu_relayed_last_minute"` // 最近一分钟APDU中继数
	ApduErrorsLastHour    int64        `json:"apdu_errors_last_hour"`    // 最近一小时APDU错误数
	ConnectionTrend       []TrendPoint `json:"connection_trend"`         // 连接趋势
	SessionTrend          []TrendPoint `json:"session_trend"`            // 会话趋势
}

// TrendPoint 趋势数据点
type TrendPoint struct {
	Time  string `json:"time"`  // 时间
	Count int    `json:"count"` // 数量
}

// PerformanceMetricsResponse 性能指标响应
type PerformanceMetricsResponse struct {
	CPUUsage       float64 `json:"cpu_usage"`       // CPU使用率
	MemoryUsage    float64 `json:"memory_usage"`    // 内存使用率
	NetworkLatency float64 `json:"network_latency"` // 网络延迟（毫秒）
	ThroughputAPDU int64   `json:"throughput_apdu"` // APDU吞吐量
}

// LocationData 位置数据
type LocationData struct {
	Location string `json:"location"` // 位置名称
	Count    int    `json:"count"`    // 数量
}

// GeographicDistributionResponse 地理分布响应
type GeographicDistributionResponse struct {
	Locations []LocationData `json:"locations"` // 位置分布
}

// Alert 告警信息
type Alert struct {
	ID        string    `json:"id"`        // 告警ID
	Level     string    `json:"level"`     // 告警级别
	Type      string    `json:"type"`      // 告警类型
	Message   string    `json:"message"`   // 告警消息
	Timestamp time.Time `json:"timestamp"` // 告警时间
	Resolved  bool      `json:"resolved"`  // 是否已解决
}

// AlertsResponse 告警响应
type AlertsResponse struct {
	Alerts []Alert `json:"alerts"` // 告警列表
}

// ExportResponse 导出响应
type ExportResponse struct {
	DownloadURL string `json:"download_url"` // 下载链接
	ExpiresAt   string `json:"expires_at"`   // 过期时间
}

// PeriodData 时期数据
type PeriodData struct {
	StartTime      string  `json:"start_time"`      // 开始时间
	EndTime        string  `json:"end_time"`        // 结束时间
	Connections    int     `json:"connections"`     // 连接数
	Sessions       int     `json:"sessions"`        // 会话数
	ApduCount      int64   `json:"apdu_count"`      // APDU数量
	SuccessRate    float64 `json:"success_rate"`    // 成功率
	AverageLatency float64 `json:"average_latency"` // 平均延迟
}

// ComparisonDataResponse 对比数据响应
type ComparisonDataResponse struct {
	CurrentPeriod  PeriodData `json:"current_period"`  // 当前时期数据
	PreviousPeriod PeriodData `json:"previous_period"` // 之前时期数据
}
