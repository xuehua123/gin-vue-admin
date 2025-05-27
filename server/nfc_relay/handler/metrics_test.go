package handler

import (
	"testing"

	"github.com/flipped-aurora/gin-vue-admin/server/global"
	"github.com/prometheus/client_golang/prometheus"
	dto "github.com/prometheus/client_model/go"

	// "github.com/prometheus/client_golang/prometheus/promauto" // Not needed for test code directly accessing vars
	"github.com/prometheus/client_golang/prometheus/testutil"
	"github.com/stretchr/testify/assert"
	"go.uber.org/zap"
)

// setupMetricsTest initializes GVA_LOG for tests.
func setupMetricsTest(t *testing.T) {
	// Prevent panic if any code path (even indirectly) tries to log.
	// Metrics themselves usually don't log, but good practice from other tests.
	if global.GVA_LOG == nil {
		global.GVA_LOG = zap.NewNop()
	}
}

// findMetricFamily is a helper to find a metric family by name from the default gatherer.
func findMetricFamily(t *testing.T, name string) *dto.MetricFamily {
	families, err := prometheus.DefaultGatherer.Gather()
	assert.NoError(t, err, "Error gathering metrics from DefaultGatherer")
	for _, family := range families {
		if family.GetName() == name {
			return family
		}
	}
	return nil
}

func TestMetricsRegistrationAndProperties(t *testing.T) {
	setupMetricsTest(t)

	testCases := []struct {
		name       string
		metricName string
		helpText   string
		metricType dto.MetricType
		collector  prometheus.Collector
	}{
		{
			name:       "ActiveConnections",
			metricName: "nfc_relay_active_connections_total",
			helpText:   "NFC 中继 Hub 的活动 WebSocket 连接总数。",
			metricType: dto.MetricType_GAUGE,
			collector:  ActiveConnections,
		},
		{
			name:       "ActiveSessions",
			metricName: "nfc_relay_active_sessions_total",
			helpText:   "活动 NFC 中继会话总数。",
			metricType: dto.MetricType_GAUGE,
			collector:  ActiveSessions,
		},
		{
			name:       "ApduMessagesRelayed",
			metricName: "nfc_relay_apdu_messages_relayed_total",
			helpText:   "按方向划分的中继 APDU 消息总数。",
			metricType: dto.MetricType_COUNTER,
			collector:  ApduMessagesRelayed,
		},
		{
			name:       "ApduRelayErrors",
			metricName: "nfc_relay_apdu_relay_errors_total",
			helpText:   "按方向划分的 APDU 中继错误总数。",
			metricType: dto.MetricType_COUNTER,
			collector:  ApduRelayErrors,
		},
		{
			name:       "SessionTerminations",
			metricName: "nfc_relay_session_terminations_total",
			helpText:   "按原因划分的 NFC 中继会话终止总数。",
			metricType: dto.MetricType_COUNTER,
			collector:  SessionTerminations,
		},
		{
			name:       "HubErrors",
			metricName: "nfc_relay_hub_errors_total",
			helpText:   "按类型或代码划分的 Hub 常规错误总数。",
			metricType: dto.MetricType_COUNTER,
			collector:  HubErrors,
		},
		{
			name:       "AuthEvents",
			metricName: "nfc_relay_auth_events_total",
			helpText:   "按类型（成功/失败）划分的认证事件总数。",
			metricType: dto.MetricType_COUNTER,
			collector:  AuthEvents,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			assert.NotNil(t, tc.collector, "%s collector should not be nil", tc.name)

			// For Vec types, "touch" the metric with dummy labels to ensure its family is gatherable.
			// This is because a MetricFamily for a Vec might not be exposed by Gather()
			// until at least one child (specific label combination) has been instantiated.
			if cv, ok := tc.collector.(*prometheus.CounterVec); ok {
				var dummyLabels prometheus.Labels
				// Determine required labels based on metric name
				switch tc.metricName {
				case "nfc_relay_apdu_messages_relayed_total", "nfc_relay_apdu_relay_errors_total":
					dummyLabels = prometheus.Labels{"direction": "test_dir", "session_id": "test_sid"}
				case "nfc_relay_session_terminations_total":
					dummyLabels = prometheus.Labels{"reason": "test_reason"}
				case "nfc_relay_hub_errors_total":
					dummyLabels = prometheus.Labels{"error_code": "test_code", "component": "test_comp"}
				case "nfc_relay_auth_events_total":
					dummyLabels = prometheus.Labels{"type": "test_type", "reason": "test_reason"}
				}
				if dummyLabels != nil {
					cv.With(dummyLabels) // This call instantiates a child metric.
				}
			}

			family := findMetricFamily(t, tc.metricName)
			if !assert.NotNil(t, family, "Metric family %s not found in default registry", tc.metricName) {
				return // Skip further checks if family is not found
			}

			assert.Equal(t, tc.metricName, family.GetName(), "Metric name mismatch for %s", tc.name)
			assert.Equal(t, tc.helpText, family.GetHelp(), "Help text mismatch for %s", tc.name)
			assert.Equal(t, tc.metricType, family.GetType(), "Metric type mismatch for %s", tc.name)
		})
	}
}

func TestActiveConnectionsMetric(t *testing.T) {
	setupMetricsTest(t)
	initialValue := testutil.ToFloat64(ActiveConnections)

	ActiveConnections.Inc()
	assert.Equal(t, initialValue+1, testutil.ToFloat64(ActiveConnections), "ActiveConnections should increment by 1")

	ActiveConnections.Dec()
	assert.Equal(t, initialValue, testutil.ToFloat64(ActiveConnections), "ActiveConnections should decrement back to initial value")

	ActiveConnections.Set(5)
	assert.Equal(t, float64(5), testutil.ToFloat64(ActiveConnections), "ActiveConnections should be set to 5")

	// Attempt to reset for subsequent tests if any dependency exists, though tests should ideally be independent.
	ActiveConnections.Set(initialValue)
}

func TestActiveSessionsMetric(t *testing.T) {
	setupMetricsTest(t)
	initialValue := testutil.ToFloat64(ActiveSessions)

	ActiveSessions.Inc()
	assert.Equal(t, initialValue+1, testutil.ToFloat64(ActiveSessions), "ActiveSessions should increment by 1")

	ActiveSessions.Dec()
	assert.Equal(t, initialValue, testutil.ToFloat64(ActiveSessions), "ActiveSessions should decrement back to initial value")

	ActiveSessions.Set(3)
	assert.Equal(t, float64(3), testutil.ToFloat64(ActiveSessions), "ActiveSessions should be set to 3")

	ActiveSessions.Set(initialValue)
}

func TestApduMessagesRelayedMetric(t *testing.T) {
	setupMetricsTest(t)

	labels1 := prometheus.Labels{"direction": "upstream", "session_id": "s1_apdu_msg"}
	counter1 := ApduMessagesRelayed.With(labels1)
	initialValue1 := testutil.ToFloat64(counter1)

	counter1.Inc()
	assert.Equal(t, initialValue1+1, testutil.ToFloat64(counter1), "ApduMessagesRelayed for upstream/s1_apdu_msg should increment by 1")

	labels2 := prometheus.Labels{"direction": "downstream", "session_id": "s2_apdu_msg"}
	counter2 := ApduMessagesRelayed.With(labels2)
	initialValue2 := testutil.ToFloat64(counter2)

	counter2.Inc()
	counter2.Inc()
	assert.Equal(t, initialValue2+2, testutil.ToFloat64(counter2), "ApduMessagesRelayed for downstream/s2_apdu_msg should increment by 2")

	// Ensure one label set doesn't affect the other if they were pre-existing with different values
	assert.Equal(t, initialValue1+1, testutil.ToFloat64(counter1), "ApduMessagesRelayed for upstream/s1_apdu_msg should remain unchanged by operations on other labels")

	// It's good practice to clean up specific label instances if the test logic allows,
	// but for promauto, this might not always be straightforward or necessary if relying on initial value checks.
	// ApduMessagesRelayed.Delete(labels1)
	// ApduMessagesRelayed.Delete(labels2)
}

func TestApduRelayErrorsMetric(t *testing.T) {
	setupMetricsTest(t)
	labels := prometheus.Labels{"direction": "downstream", "session_id": "s_err1_apdu"}
	counter := ApduRelayErrors.With(labels)
	initialValue := testutil.ToFloat64(counter)

	counter.Inc()
	assert.Equal(t, initialValue+1, testutil.ToFloat64(counter), "ApduRelayErrors for downstream/s_err1_apdu should increment")
}

func TestSessionTerminationsMetric(t *testing.T) {
	setupMetricsTest(t)
	labels := prometheus.Labels{"reason": "client_disconnect"}
	counter := SessionTerminations.With(labels)
	initialValue := testutil.ToFloat64(counter)

	counter.Inc()
	assert.Equal(t, initialValue+1, testutil.ToFloat64(counter), "SessionTerminations for reason 'client_disconnect' should increment")
}

func TestHubErrorsMetric(t *testing.T) {
	setupMetricsTest(t)
	labels := prometheus.Labels{"error_code": "400", "component": "auth_handler"}
	counter := HubErrors.With(labels)
	initialValue := testutil.ToFloat64(counter)

	counter.Inc()
	assert.Equal(t, initialValue+1, testutil.ToFloat64(counter), "HubErrors for 400/auth_handler should increment")
}

func TestAuthEventsMetric(t *testing.T) {
	setupMetricsTest(t)

	labelsSuccess := prometheus.Labels{"type": "success", "reason": ""} // reason is empty for success
	counterSuccess := AuthEvents.With(labelsSuccess)
	initialValueSuccess := testutil.ToFloat64(counterSuccess)

	counterSuccess.Inc()
	assert.Equal(t, initialValueSuccess+1, testutil.ToFloat64(counterSuccess), "AuthEvents for type 'success' should increment")

	labelsFailure := prometheus.Labels{"type": "failure", "reason": "invalid_credentials"}
	counterFailure := AuthEvents.With(labelsFailure)
	initialValueFailure := testutil.ToFloat64(counterFailure)

	counterFailure.Inc()
	assert.Equal(t, initialValueFailure+1, testutil.ToFloat64(counterFailure), "AuthEvents for type 'failure' reason 'invalid_credentials' should increment")
}
