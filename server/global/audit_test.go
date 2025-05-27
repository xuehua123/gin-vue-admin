package global

import (
	"encoding/json"
	"fmt"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"go.uber.org/zap/zaptest/observer"
)

// Test helper to safely get GVA_LOG or a Nop logger if GVA_LOG is nil
func getSafeGVALog() *zap.Logger {
	if GVA_LOG == nil {
		return zap.NewNop()
	}
	return GVA_LOG
}

// TestInitializeAuditLogger_WithGVALOG tests InitializeAuditLogger when GVA_LOG is set.
func TestInitializeAuditLogger_WithGVALOG(t *testing.T) {
	originalGVALOG := GVA_LOG
	originalAuditLogger := AuditLogger
	defer func() {
		GVA_LOG = originalGVALOG
		AuditLogger = originalAuditLogger
	}()

	obsCore, observedLogs := observer.New(zapcore.InfoLevel) // Capture the ObservedLogs object
	GVA_LOG = zap.New(obsCore)                               // Set a valid GVA_LOG
	AuditLogger = nil                                        // Ensure it's reset before test

	InitializeAuditLogger()
	assert.NotNil(t, AuditLogger, "AuditLogger should be initialized")

	// Check if AuditLogger is a child of GVA_LOG by trying to log and observe
	AuditLogger.Info("test_audit_log_init_child_check")
	logs := observedLogs.TakeAll() // Use observedLogs.TakeAll()
	found := false
	for _, log := range logs {
		if strings.Contains(log.LoggerName, "audit") && log.Message == "test_audit_log_init_child_check" {
			found = true
			break
		}
	}
	assert.True(t, found, "AuditLogger should be a named logger from GVA_LOG and log correctly via GVA_LOG's core")
}

// TestInitializeAuditLogger_GVALOGIsNil tests InitializeAuditLogger when GVA_LOG is nil.
func TestInitializeAuditLogger_GVALOGIsNil(t *testing.T) {
	originalGVALOG := GVA_LOG
	originalAuditLogger := AuditLogger
	// This defer must be first to ensure it runs even if panics occur elsewhere in setup
	defer func() {
		GVA_LOG = originalGVALOG
		AuditLogger = originalAuditLogger
	}()

	GVA_LOG = nil     // Explicitly set GVA_LOG to nil
	AuditLogger = nil // Ensure AuditLogger is reset

	// We expect a panic because the original code will call GVA_LOG.Warn() on a nil GVA_LOG
	// after a fallback logger is created for AuditLogger.
	assert.Panics(t, func() {
		InitializeAuditLogger()
	}, "Expected a panic when GVA_LOG is nil and InitializeAuditLogger tries to use it for GVA_LOG.Warn")

	// After the panic from GVA_LOG.Warn, AuditLogger itself should have been initialized
	// by the fallback zap.NewProduction().Named("audit") just before the panic line.
	assert.NotNil(t, AuditLogger, "AuditLogger should still be initialized with a fallback logger before the GVA_LOG.Warn panic")
}

// TestLogAuditEvent_Success tests successful logging of an audit event.
func TestLogAuditEvent_Success(t *testing.T) {
	originalGVALOG := GVA_LOG
	originalAuditLogger := AuditLogger
	defer func() {
		GVA_LOG = originalGVALOG
		AuditLogger = originalAuditLogger
	}()

	// Setup observed logger for GVA_LOG (to catch debug/info/warn from audit.go itself)
	gvaCore, gvaLogs := observer.New(zapcore.DebugLevel)
	GVA_LOG = zap.New(gvaCore)

	// Setup observed logger for AuditLogger
	auditCore, auditLogs := observer.New(zapcore.InfoLevel)
	AuditLogger = zap.New(auditCore).Named("audit") // Initialize AuditLogger directly for this test

	eventType := "test_event_success"
	details := map[string]interface{}{"key": "value", "number": 123}
	extraField := zap.String("extraKey", "extraValue")

	LogAuditEvent(eventType, details, extraField)

	// Check GVA_LOG for debug messages (optional, but good to ensure they ran)
	assert.GreaterOrEqual(t, len(gvaLogs.All()), 1, "Expected some debug logs in GVA_LOG")

	// Check AuditLogger output
	allAuditLogs := auditLogs.TakeAll()
	if !assert.Len(t, allAuditLogs, 1, "Expected one log entry in AuditLogger") {
		t.FailNow()
	}
	logEntry := allAuditLogs[0]

	assert.Equal(t, "AuditEvent", logEntry.Message, "Log message should be AuditEvent")
	assert.Equal(t, zapcore.InfoLevel, logEntry.Level, "Log level should be Info")
	assert.True(t, strings.HasSuffix(logEntry.LoggerName, "audit"), "Logger name should end with audit")

	contextMap := logEntry.ContextMap()
	assert.Equal(t, eventType, contextMap["event_type"], "Event type mismatch")

	timestamp, ok := contextMap["timestamp"].(string)
	assert.True(t, ok, "Timestamp should be a string")
	_, err := time.Parse(time.RFC3339Nano, timestamp)
	assert.NoError(t, err, "Timestamp should be in RFC3339Nano format")

	loggedDetails, ok := contextMap["details"].(map[string]interface{})
	assert.True(t, ok, "Details should be a map[string]interface{}")
	assert.Equal(t, details["key"], loggedDetails["key"])
	assert.Equal(t, float64(details["number"].(int)), loggedDetails["number"])

	assert.Equal(t, extraField.String, contextMap[extraField.Key], "Extra field mismatch")
}

// TestLogAuditEvent_AuditLoggerNil_Initializes tests that AuditLogger is initialized if nil.
func TestLogAuditEvent_AuditLoggerNil_Initializes(t *testing.T) {
	originalGVALOG := GVA_LOG
	originalAuditLogger := AuditLogger
	defer func() {
		GVA_LOG = originalGVALOG
		AuditLogger = originalAuditLogger
	}()

	gvaCore, gvaLogs := observer.New(zapcore.InfoLevel)
	GVA_LOG = zap.New(gvaCore)
	AuditLogger = nil

	LogAuditEvent("test_init_event", nil)

	assert.NotNil(t, AuditLogger, "AuditLogger should have been initialized")
	initLogFound := false
	for _, log := range gvaLogs.All() {
		if log.Message == "AuditLogger 为 nil，并在首次使用时已初始化。" {
			initLogFound = true
			break
		}
	}
	assert.True(t, initLogFound, "Expected GVA_LOG message about AuditLogger initialization")
}

// TestLogAuditEvent_DetailsSerializationFailure tests fallback when event.Details fails to marshal.
func TestLogAuditEvent_DetailsSerializationFailure(t *testing.T) {
	originalGVALOG := GVA_LOG
	originalAuditLogger := AuditLogger
	defer func() {
		GVA_LOG = originalGVALOG
		AuditLogger = originalAuditLogger
	}()

	gvaCore, gvaLogs := observer.New(zapcore.DebugLevel)
	GVA_LOG = zap.New(gvaCore)

	auditCore, auditLogs := observer.New(zapcore.InfoLevel)
	AuditLogger = zap.New(auditCore).Named("audit")

	eventType := "test_marshal_fail_event"
	unmarshalableDetails := map[string]interface{}{
		"channel": make(chan int),
	}

	LogAuditEvent(eventType, unmarshalableDetails)

	marshalErrorLogged := false
	for _, log := range gvaLogs.All() {
		if log.Level == zapcore.ErrorLevel && strings.Contains(log.Message, "序列化 AuditEvent 以进行结构化日志记录失败") {
			marshalErrorLogged = true
			assert.Equal(t, "json: unsupported type: chan int", log.ContextMap()["error"].(string))
			break
		}
	}
	assert.True(t, marshalErrorLogged, "Expected GVA_LOG to record the marshal error")

	allAuditLogs := auditLogs.TakeAll()
	if !assert.Len(t, allAuditLogs, 1, "Expected one log entry in AuditLogger (fallback)") {
		t.FailNow()
	}
	logEntry := allAuditLogs[0]
	assert.Equal(t, "AuditEvent", logEntry.Message)

	contextMap := logEntry.ContextMap()
	assert.Equal(t, eventType, contextMap["eventType"], "Event type should be present in fallback")

	loggedDetails, ok := contextMap["details"].(map[string]interface{})
	assert.True(t, ok, "Details should be present as fallback")
	_, chanOk := loggedDetails["channel"]
	assert.True(t, chanOk, "Unmarshable part of details should be in fallback log")
}

// TestLogAuditEvent_NilDetails tests logging with nil details.
func TestLogAuditEvent_NilDetails(t *testing.T) {
	originalGVALOG := GVA_LOG
	originalAuditLogger := AuditLogger
	defer func() {
		GVA_LOG = originalGVALOG
		AuditLogger = originalAuditLogger
	}()

	GVA_LOG = zap.NewNop()
	auditCore, auditLogs := observer.New(zapcore.InfoLevel)
	AuditLogger = zap.New(auditCore).Named("audit")

	LogAuditEvent("test_nil_details_event", nil)

	allAuditLogs := auditLogs.TakeAll()
	if !assert.Len(t, allAuditLogs, 1, "Expected one log entry") {
		t.FailNow()
	}
	logEntry := allAuditLogs[0]
	contextMap := logEntry.ContextMap()

	_, detailsExist := contextMap["details"]
	assert.False(t, detailsExist, "Details field should be omitted in log context if event.Details was nil and marshaled with omitempty")
}

// TestLogAuditEvent_NoExtraFields tests logging without any extra zap.Fields.
func TestLogAuditEvent_NoExtraFields(t *testing.T) {
	originalGVALOG := GVA_LOG
	originalAuditLogger := AuditLogger
	defer func() {
		GVA_LOG = originalGVALOG
		AuditLogger = originalAuditLogger
	}()

	GVA_LOG = zap.NewNop()
	auditCore, auditLogs := observer.New(zapcore.InfoLevel)
	AuditLogger = zap.New(auditCore).Named("audit")

	eventType := "test_no_extra_fields_event"
	details := map[string]string{"data": "content"}
	LogAuditEvent(eventType, details)

	allAuditLogs := auditLogs.TakeAll()
	if !assert.Len(t, allAuditLogs, 1, "Expected one log entry") {
		t.FailNow()
	}
	logEntry := allAuditLogs[0]
	contextMap := logEntry.ContextMap()

	assert.Equal(t, eventType, contextMap["event_type"])
	loggedDetails, _ := contextMap["details"].(map[string]interface{})
	assert.Equal(t, details["data"], loggedDetails["data"].(string)) // Cast to string for direct comparison

	standardFieldCount := 0
	if _, ok := contextMap["timestamp"]; ok {
		standardFieldCount++
	}
	if _, ok := contextMap["event_type"]; ok {
		standardFieldCount++
	}
	if _, ok := contextMap["details"]; ok {
		standardFieldCount++
	}
	// Potentially other standard zap fields like "logger" or "caller" if not disabled
	// For this test, focus on absence of non-standard fields.
	assert.LessOrEqual(t, len(contextMap), standardFieldCount+2, "Should not have many extra fields beyond standard ones")

	_, extraKeyFound := contextMap["extraKey"] // A key we used in other tests
	assert.False(t, extraKeyFound, "No non-standard extra fields should be present")
}

// TestAuditEventStruct_JSONTags tests the JSON tags of AuditEvent.
func TestAuditEventStruct_JSONTags(t *testing.T) {
	event := AuditEvent{
		Timestamp:         "ts",
		EventType:         "et",
		SessionID:         "sid",
		ClientIDInitiator: "cid_init",
		ClientIDResponder: "cid_resp",
		SourceIP:          "sip",
		UserID:            "uid",
		Details:           map[string]string{"key": "val"},
	}
	bytes, err := json.Marshal(event)
	assert.NoError(t, err)
	jsonString := string(bytes)

	assert.Contains(t, jsonString, `"timestamp":"ts"`)
	assert.Contains(t, jsonString, `"event_type":"et"`)
	assert.Contains(t, jsonString, `"session_id":"sid"`)
	assert.Contains(t, jsonString, `"client_id_initiator":"cid_init"`)
	assert.Contains(t, jsonString, `"client_id_responder":"cid_resp"`)
	assert.Contains(t, jsonString, `"source_ip":"sip"`)
	assert.Contains(t, jsonString, `"user_id":"uid"`)
	assert.Contains(t, jsonString, `"details":{"key":"val"}`)

	eventOnlyRequired := AuditEvent{
		Timestamp: "ts_req",
		EventType: "et_req",
	}
	bytesReq, errReq := json.Marshal(eventOnlyRequired)
	assert.NoError(t, errReq)
	jsonStringReq := string(bytesReq)

	assert.Contains(t, jsonStringReq, `"timestamp":"ts_req"`)
	assert.Contains(t, jsonStringReq, `"event_type":"et_req"`)
	assert.NotContains(t, jsonStringReq, "session_id")
	assert.NotContains(t, jsonStringReq, "client_id_initiator")
	assert.NotContains(t, jsonStringReq, "client_id_responder")
	assert.NotContains(t, jsonStringReq, "source_ip")
	assert.NotContains(t, jsonStringReq, "user_id")
	assert.NotContains(t, jsonStringReq, "details")
}

// TestDetailStructs_JSONSerialization tests JSON serialization and deserialization of detail structs.
func TestDetailStructs_JSONSerialization(t *testing.T) {
	tests := []struct {
		name        string
		input       interface{}
		assertEmpty bool
		checkFields func(t *testing.T, original, unmarshaled interface{})
	}{
		{
			name:  "AuthDetails_Full",
			input: &AuthDetails{Username: "user1", Reason: "login_failed"},
			checkFields: func(t *testing.T, original, unmarshaled interface{}) {
				orig := original.(*AuthDetails)
				um := unmarshaled.(*AuthDetails)
				assert.Equal(t, orig.Username, um.Username)
				assert.Equal(t, orig.Reason, um.Reason)
			},
		},
		{
			name:        "AuthDetails_Empty",
			input:       &AuthDetails{},
			assertEmpty: true,
		},
		{
			name:  "SessionDetails_Full",
			input: &SessionDetails{InitiatorRole: "card", ResponderRole: "pos"},
			checkFields: func(t *testing.T, original, unmarshaled interface{}) {
				orig := original.(*SessionDetails)
				um := unmarshaled.(*SessionDetails)
				assert.Equal(t, orig.InitiatorRole, um.InitiatorRole)
				assert.Equal(t, orig.ResponderRole, um.ResponderRole)
			},
		},
		{
			name:        "SessionDetails_Empty",
			input:       &SessionDetails{},
			assertEmpty: true,
		},
		{
			name:  "APDUDetails_Full",
			input: &APDUDetails{Direction: "c->s", Length: 20},
			checkFields: func(t *testing.T, original, unmarshaled interface{}) {
				orig := original.(*APDUDetails)
				um := unmarshaled.(*APDUDetails)
				assert.Equal(t, orig.Direction, um.Direction)
				assert.Equal(t, orig.Length, um.Length)
			},
		},
		{
			name:        "APDUDetails_Empty",
			input:       &APDUDetails{},
			assertEmpty: true,
		},
		{
			name:  "ErrorDetails_Full",
			input: &ErrorDetails{ErrorCode: "E1001", ErrorMessage: "network error", Component: "nfc_hub", AffectedData: "0xDEADBEEF"},
			checkFields: func(t *testing.T, original, unmarshaled interface{}) {
				orig := original.(*ErrorDetails)
				um := unmarshaled.(*ErrorDetails)
				assert.Equal(t, orig.ErrorCode, um.ErrorCode)
				assert.Equal(t, orig.ErrorMessage, um.ErrorMessage)
				assert.Equal(t, orig.Component, um.Component)
				assert.Equal(t, orig.AffectedData, um.AffectedData)
			},
		},
		{
			name:  "ErrorDetails_OnlyRequiredMessage",
			input: &ErrorDetails{ErrorMessage: "A message"},
			checkFields: func(t *testing.T, original, unmarshaled interface{}) {
				orig := original.(*ErrorDetails)
				um := unmarshaled.(*ErrorDetails)
				assert.Equal(t, orig.ErrorMessage, um.ErrorMessage)
				assert.Empty(t, um.ErrorCode)
			},
		},
		{
			name:        "ErrorDetails_Empty",
			input:       &ErrorDetails{},
			assertEmpty: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			data, err := json.Marshal(tt.input)
			assert.NoError(t, err, "json.Marshal should not fail for %s", tt.name)

			if tt.assertEmpty {
				assert.Equal(t, "{}", string(data), "Expected empty JSON object for %s with zero values", tt.name)
			} else if tt.name == "ErrorDetails_Empty" {
				assert.Equal(t, `{"error_message":""}`, string(data))
			}

			var unmarshaledInstance interface{}
			switch tt.input.(type) {
			case *AuthDetails:
				unmarshaledInstance = new(AuthDetails)
			case *SessionDetails:
				unmarshaledInstance = new(SessionDetails)
			case *APDUDetails:
				unmarshaledInstance = new(APDUDetails)
			case *ErrorDetails:
				unmarshaledInstance = new(ErrorDetails)
			default:
				t.Fatalf("Unsupported type in test: %T", tt.input)
			}

			err = json.Unmarshal(data, unmarshaledInstance)
			assert.NoError(t, err, "json.Unmarshal should not fail for %s", tt.name)

			if tt.checkFields != nil {
				tt.checkFields(t, tt.input, unmarshaledInstance)
			}
		})
	}
}

// TestLogAuditEvent_MultipleCalls_GoroutineSafety uses a WaitGroup for basic goroutine safety check.
func TestLogAuditEvent_MultipleCalls_GoroutineSafety(t *testing.T) {
	originalGVALOG := GVA_LOG
	originalAuditLogger := AuditLogger
	defer func() {
		GVA_LOG = originalGVALOG
		AuditLogger = originalAuditLogger
	}()

	GVA_LOG = zap.NewNop()
	AuditLogger = zap.NewNop().Named("audit")

	var wg sync.WaitGroup
	numGoroutines := 20

	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()
			LogAuditEvent(
				fmt.Sprintf("concurrent_event_%d", idx),
				map[string]interface{}{"goroutine": idx, "timestamp_ms": time.Now().UnixMilli()},
				zap.Int("index", idx),
			)
		}(i)
	}
	wg.Wait()
}

// TestLogAuditEvent_TimestampFormatAndUniqueness is a more focused test on timestamp.
func TestLogAuditEvent_TimestampFormatAndUniqueness(t *testing.T) {
	originalGVALOG := GVA_LOG
	originalAuditLogger := AuditLogger
	defer func() {
		GVA_LOG = originalGVALOG
		AuditLogger = originalAuditLogger
	}()

	GVA_LOG = zap.NewNop()
	auditCore, auditLogs := observer.New(zapcore.InfoLevel)
	AuditLogger = zap.New(auditCore).Named("audit")

	LogAuditEvent("ts_event_1", nil)
	time.Sleep(2 * time.Millisecond) // Increased sleep to ensure time difference
	LogAuditEvent("ts_event_2", nil)

	allLogs := auditLogs.TakeAll()
	if !assert.Len(t, allLogs, 2) {
		t.FailNow()
	}

	ts1Str, ok1 := allLogs[0].ContextMap()["timestamp"].(string)
	assert.True(t, ok1)
	ts2Str, ok2 := allLogs[1].ContextMap()["timestamp"].(string)
	assert.True(t, ok2)

	ts1, err1 := time.Parse(time.RFC3339Nano, ts1Str)
	assert.NoError(t, err1)
	ts2, err2 := time.Parse(time.RFC3339Nano, ts2Str)
	assert.NoError(t, err2)

	assert.True(t, ts2.After(ts1), fmt.Sprintf("Timestamp of second log (%s) should be after the first (%s)", ts2Str, ts1Str))
}
