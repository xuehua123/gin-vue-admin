package service

import (
	"github.com/flipped-aurora/gin-vue-admin/server/global"
	"github.com/flipped-aurora/gin-vue-admin/server/nfc_relay/handler" // 导入 handler 包以访问 GlobalRelayHub 和指标
	"github.com/flipped-aurora/gin-vue-admin/server/nfc_relay/model/admin_response"
	"github.com/prometheus/client_golang/prometheus"

	// "github.com/prometheus/client_golang/prometheus/testutil" // testutil 不适用于 CounterVec 的复杂提取
	// dto "github.com/prometheus/client_model/go" // 导入 DTO 包 - 已不再需要
	"go.uber.org/zap"
)

// AdminDashboardService 结构体，目前不需要任何内部状态
type AdminDashboardService struct{}

// GetDashboardStats 获取仪表盘统计数据
func (s *AdminDashboardService) GetDashboardStats() (stats admin_response.DashboardStatsData, err error) {
	stats.HubStatus = "online" // Hub 持续运行，除非应用崩溃

	// 从 GlobalRelayHub 获取实时数据
	stats.ActiveConnections = handler.GlobalRelayHub.GetActiveConnectionsCount()
	stats.ActiveSessions = handler.GlobalRelayHub.GetActiveSessionsCount()

	// 初始化 maps
	stats.ApduRelayedByDirection = make(map[string]float64)
	stats.SessionTerminations = make(map[string]float64)
	stats.AuthEvents = make(map[string]map[string]float64)

	// 从 Prometheus 指标获取统计数据
	mfs, err := prometheus.DefaultGatherer.Gather()
	if err != nil {
		global.GVA_LOG.Error("获取Prometheus指标失败", zap.Error(err))
		// 即使获取失败，也返回部分数据，或者可以决定返回错误
		return stats, err // 或者 return stats, nil 如果希望部分成功
	}

	for _, mf := range mfs {
		switch mf.GetName() {
		case "nfc_relay_apdu_messages_relayed_total":
			var totalRelayed float64
			for _, m := range mf.Metric {
				var direction string
				for _, lp := range m.Label {
					if lp.GetName() == "direction" {
						direction = lp.GetValue()
						break
					}
				}
				if m.Counter != nil && m.Counter.Value != nil {
					val := *m.Counter.Value
					totalRelayed += val
					if direction != "" {
						stats.ApduRelayedByDirection[direction] += val
					}
				}
			}
			stats.TotalApduRelayed = totalRelayed

		case "nfc_relay_apdu_relay_errors_total":
			var totalErrors float64
			for _, m := range mf.Metric {
				if m.Counter != nil && m.Counter.Value != nil {
					totalErrors += *m.Counter.Value
				}
			}
			stats.TotalApduErrors = totalErrors

		case "nfc_relay_hub_errors_total":
			var totalHubErrors float64
			for _, m := range mf.Metric {
				if m.Counter != nil && m.Counter.Value != nil {
					totalHubErrors += *m.Counter.Value
				}
			}
			stats.TotalHubErrors = totalHubErrors

		case "nfc_relay_session_terminations_total":
			for _, m := range mf.Metric {
				var reason string
				for _, lp := range m.Label {
					if lp.GetName() == "reason" {
						reason = lp.GetValue()
						break
					}
				}
				if reason != "" && m.Counter != nil && m.Counter.Value != nil {
					stats.SessionTerminations[reason] += *m.Counter.Value
				}
			}

		case "nfc_relay_auth_events_total":
			for _, m := range mf.Metric {
				var authType, authReason string
				for _, lp := range m.Label {
					if lp.GetName() == "type" {
						authType = lp.GetValue()
					} else if lp.GetName() == "reason" {
						authReason = lp.GetValue()
					}
				}
				if authType != "" && m.Counter != nil && m.Counter.Value != nil {
					if stats.AuthEvents[authType] == nil {
						stats.AuthEvents[authType] = make(map[string]float64)
					}
					stats.AuthEvents[authType][authReason] += *m.Counter.Value
				}
			}
		}
	}

	global.GVA_LOG.Debug("NFC Relay Dashboard Stats Prepared", zap.Any("stats", stats))
	return stats, nil
}
