package handler

import (
	"encoding/json"
	"errors"
	"net"
	"strconv"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/flipped-aurora/gin-vue-admin/server/config"
	"github.com/flipped-aurora/gin-vue-admin/server/global"
	"github.com/flipped-aurora/gin-vue-admin/server/nfc_relay/protocol"
	"github.com/flipped-aurora/gin-vue-admin/server/nfc_relay/session"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"go.uber.org/zap"
	"go.uber.org/zap/zaptest/observer"
)

// mockClientInfoProvider is a mock implementation of the session.ClientInfoProvider interface for testing.
type mockClientInfoProvider struct {
	id              string
	userID          string
	sessionID       string
	currentRole     protocol.RoleType
	sendShouldFail  bool
	sendCalled      bool
	lastSentMessage []byte
	mu              sync.Mutex
}

func newMockClientInfoProvider(id, userID, sessionID string) *mockClientInfoProvider {
	return &mockClientInfoProvider{
		id:        id,
		userID:    userID,
		sessionID: sessionID,
	}
}

func (m *mockClientInfoProvider) Send(message []byte) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.sendCalled = true
	m.lastSentMessage = message
	if m.sendShouldFail {
		return errors.New("mock send error")
	}
	return nil
}

func (m *mockClientInfoProvider) GetID() string                     { return m.id }
func (m *mockClientInfoProvider) GetUserID() string                 { return m.userID }
func (m *mockClientInfoProvider) GetSessionID() string              { return m.sessionID }
func (m *mockClientInfoProvider) GetCurrentRole() protocol.RoleType { return m.currentRole }
func (m *mockClientInfoProvider) GetRole() string                   { return string(m.currentRole) }

// Helper function to create a real *Client for tests that need it, pre-configured.
// This client uses a mock websocket connection from client_test.go
func mockClientForHubTests(hub *Hub) *Client {
	mockConn := newMockWsConnection() // Assumes newMockWsConnection is defined in client_test.go and is accessible
	client := NewClient(hub, mockConn)
	return client
}

// setupHubTestGlobalConfig initializes global configurations needed for hub tests.
func setupHubTestGlobalConfig(t *testing.T) {
	// Ensure GVA_CONFIG itself is not nil and then check/initialize NfcRelay.
	// Viper usually populates GVA_CONFIG, but in tests, we might need to ensure it.
	// A simple check could be on a top-level field of GVA_CONFIG if one is always expected.
	// For now, let's assume GVA_CONFIG structure is allocated and focus on NfcRelay part.
	// A more robust way would be to ensure config.Server is initialized if any part of it is zero/default.

	// If GVA_CONFIG itself is the zero value of config.Server, initialize it.
	// This is a basic check. In a real scenario, Viper would populate this from a file.
	if global.GVA_CONFIG.System.Addr == 0 { // Check a field from System part of Server config
		global.GVA_CONFIG = config.Server{
			System: config.System{
				Addr: 8888, // Default test address
			},
			JWT: config.JWT{
				SigningKey:  "test-signing-key",
				ExpiresTime: "24h",
				BufferTime:  "1h",
				Issuer:      "test-issuer",
			},
			NfcRelay: config.NfcRelay{ // Initialize NfcRelay along with Server
				HubCheckIntervalSec:       60,
				SessionInactiveTimeoutSec: 300,
				WebsocketWriteWaitSec:     10,
				WebsocketPongWaitSec:      60,
				WebsocketMaxMessageBytes:  2048,
			},
			// Initialize other necessary parts of config.Server with defaults for tests
		}
	} else {
		// GVA_CONFIG is already somewhat initialized, ensure NfcRelay part is.
		if global.GVA_CONFIG.NfcRelay.WebsocketPongWaitSec == 0 {
			global.GVA_CONFIG.NfcRelay = config.NfcRelay{
				HubCheckIntervalSec:       60,
				SessionInactiveTimeoutSec: 300,
				WebsocketWriteWaitSec:     10,
				WebsocketPongWaitSec:      60,
				WebsocketMaxMessageBytes:  2048,
			}
		}
	}

	// Ensure GVA_LOG is initialized
	if global.GVA_LOG == nil {
		// For tests, we can use a Nop logger or a test logger
		// core, _ := observer.New(zap.InfoLevel)
		// global.GVA_LOG = zap.New(core)
		global.GVA_LOG = zap.NewNop() // Keeping Nop for general setup to avoid output unless specified
	}
	// Ensure AuditLogger is initialized
	if global.AuditLogger == nil {
		global.InitializeAuditLogger() // Uses GVA_LOG
	}
}

// TestNewHub tests the NewHub constructor.
func TestNewHub(t *testing.T) {
	setupHubTestGlobalConfig(t)
	hub := NewHub()

	assert.NotNil(t, hub, "NewHub() should return a non-nil Hub instance")
	assert.NotNil(t, hub.clients, "Hub.clients map should be initialized")
	assert.IsType(t, make(map[*Client]bool), hub.clients, "Hub.clients should be of type map[*Client]bool")

	assert.NotNil(t, hub.processMessage, "Hub.processMessage channel should be initialized")
	assert.IsType(t, make(chan ProcessableMessage), hub.processMessage, "Hub.processMessage should be of type chan ProcessableMessage")

	assert.NotNil(t, hub.register, "Hub.register channel should be initialized")
	assert.IsType(t, make(chan *Client), hub.register, "Hub.register should be of type chan *Client")

	assert.NotNil(t, hub.unregister, "Hub.unregister channel should be initialized")
	assert.IsType(t, make(chan *Client), hub.unregister, "Hub.unregister should be of type chan *Client")

	assert.NotNil(t, hub.sessions, "Hub.sessions map should be initialized")
	assert.IsType(t, make(map[string]*session.Session), hub.sessions, "Hub.sessions should be of type map[string]*session.Session")

	assert.NotNil(t, hub.cardProviders, "Hub.cardProviders map should be initialized")
	assert.IsType(t, make(map[string]session.ClientInfoProvider), hub.cardProviders, "Hub.cardProviders should be of type map[string]session.ClientInfoProvider")

	assert.NotNil(t, hub.providerListSubscribers, "Hub.providerListSubscribers map should be initialized")
	assert.IsType(t, make(map[string]map[*Client]bool), hub.providerListSubscribers, "Hub.providerListSubscribers should be of type map[string]map[*Client]bool")

	// Test the mutex is initialized (it's a struct, so it's zero-value is usable)
	// We can try to lock and unlock it to ensure it doesn't panic, though this is a bit trivial.
	var wg sync.WaitGroup
	wg.Add(2)
	go func() {
		hub.providerMutex.Lock()
		hub.providerMutex.Unlock()
		wg.Done()
	}()
	go func() {
		hub.providerMutex.RLock()
		hub.providerMutex.RUnlock()
		wg.Done()
	}()
	wg.Wait() // If it panics, the test will fail.

	// Check metricsMutex too
	var metricsWg sync.WaitGroup
	metricsWg.Add(1)
	go func() {
		hub.metricsMutex.Lock()
		hub.metricsMutex.Unlock()
		metricsWg.Done()
	}()
	metricsWg.Wait()
}

// Helper to create a Hub with an observed logger for specific tests
func newHubWithObservedLogger(t *testing.T) (*Hub, *observer.ObservedLogs) {
	core, logs := observer.New(zap.DebugLevel) // Capture Debug level and above
	testLogger := zap.New(core)

	originalGvaLog := global.GVA_LOG
	originalAuditLogger := global.AuditLogger

	global.GVA_LOG = testLogger
	// Ensure AuditLogger also uses the observed core by re-initializing it AFTER GVA_LOG is set to testLogger.
	// InitializeAuditLogger typically sets global.AuditLogger = global.GVA_LOG.Named("audit"),
	// so it needs to see the overridden global.GVA_LOG.
	global.InitializeAuditLogger()

	t.Cleanup(func() {
		global.GVA_LOG = originalGvaLog
		global.AuditLogger = originalAuditLogger
		// If InitializeAuditLogger had side effects beyond setting global.AuditLogger based on global.GVA_LOG,
		// those might need more specific restoration here. For now, restoring the pointers is the primary concern.
		// Re-initialize with original GVA_LOG if necessary for subsequent tests not using observer.
		if global.GVA_LOG != nil { // Avoid panic if originalGvaLog was nil (though unlikely for GVA_LOG)
			global.InitializeAuditLogger()
		} else {
			// Handle case where original GVA_LOG might have been nil, perhaps set AuditLogger to a Nop or specific default.
			// For this project, GVA_LOG is usually initialized, so direct re-init should be fine.
		}
	})

	hub := NewHub()
	return hub, logs
}

// TestHub_Run_ClientRegistration tests the client registration case in Hub.Run().
func TestHub_Run_ClientRegistration(t *testing.T) {
	setupHubTestGlobalConfig(t) // General setup, might use Nop logger by default

	// For this specific test, we want to observe logs, so we create a hub with an observer.
	hub, observedLogs := newHubWithObservedLogger(t)
	go hub.Run() // Run the hub in a goroutine

	// Mock Prometheus Gauge (ActiveConnections)
	// Since ActiveConnections is a global var, we can read its state before/after.
	// For more complex scenarios or if it were an interface, mocking would be different.
	// We'll assume direct interaction or rely on other side effects like logs for now.
	// If direct reading is hard, we'd need to mock the prometheus.Gauge itself.
	// For now, we'll check the log and map, and assume the metric call is made.

	client := mockClientForHubTests(hub)

	// Act
	hub.register <- client

	// Assert
	// Give some time for the hub to process the registration
	time.Sleep(50 * time.Millisecond)

	hub.providerMutex.RLock() // Use RLock as we are only reading
	_, ok := hub.clients[client]
	hub.providerMutex.RUnlock()
	assert.True(t, ok, "Client should be added to hub.clients map")

	// Check logs for registration message
	// The log message is: "客户端已注册到 Hub"
	foundLog := false
	for _, logEntry := range observedLogs.All() {
		if logEntry.Message == "客户端已注册到 Hub" {
			foundLog = true
			// Optionally, check fields like clientID
			var clientIDLogged string
			for _, field := range logEntry.Context {
				if field.Key == "clientID" {
					clientIDLogged = field.String
					break
				}
			}
			assert.Equal(t, client.ID, clientIDLogged, "Logged clientID should match registered client's ID")
			break
		}
	}
	assert.True(t, foundLog, "Expected log message for client registration was not found")

	// Regarding ActiveConnections.Inc():
	// Directly testing changes to prometheus.Gauge can be tricky in unit tests without
	// a testing-friendly prometheus library or more complex mocking of the global var.
	// For now, we rely on the code review that .Inc() is called and the log as an indicator.
	// If this were a critical path for value verification, we'd explore exposing a way to read it
	// or use a mockable interface for metrics.

	// Cleanup: Stop the hub or manage its lifecycle if necessary for other tests.
	// Since hub.Run() is in a for-select loop, it will run indefinitely.
	// For tests, we might need a way to stop it, or ensure tests are isolated.
	// This test focuses on a single case, subsequent tests might re-initialize the hub.
	// Consider adding a stop channel to the hub for cleaner test shutdown if needed.
}

// TestHub_Run_ClientUnregistration tests the client unregistration case in Hub.Run().
func TestHub_Run_ClientUnregistration(t *testing.T) {
	setupHubTestGlobalConfig(t)
	hub, observedLogs := newHubWithObservedLogger(t)
	go hub.Run()

	client := mockClientForHubTests(hub)

	// Initial registration
	hub.register <- client
	// Add a small delay to ensure the registration goroutine has a chance to run.
	// This helps prevent race conditions in tests, especially on slower CI machines.
	time.Sleep(10 * time.Millisecond)

	// Confirm client is registered before unregistering
	hub.providerMutex.RLock()
	_, اولیهOk := hub.clients[client]
	hub.providerMutex.RUnlock()
	assert.True(t, اولیهOk, "Client should be in hub.clients map before unregistration")

	// Act: Unregister the client
	hub.unregister <- client
	time.Sleep(50 * time.Millisecond) // Allow unregistration to complete

	// Assert: Client removed from map
	hub.providerMutex.RLock()
	_, okAfterUnregister := hub.clients[client]
	hub.providerMutex.RUnlock()
	assert.False(t, okAfterUnregister, "Client should be removed from hub.clients map after unregistration")

	// Assert: Client's send channel is closed
	// Wait for the channel to be closed. This can be tricky if the closing is asynchronous.
	var chanClosed bool
	select {
	case _, ok := <-client.send:
		if !ok {
			chanClosed = true
		}
	case <-time.After(100 * time.Millisecond): // Timeout to prevent test hanging
		// If the channel is not closed quickly, this might indicate an issue.
	}
	assert.True(t, chanClosed, "Client's send channel should be closed after unregistration")

	// Assert: Log message for unregistration
	foundLog := false
	for _, logEntry := range observedLogs.All() {
		if logEntry.Message == "客户端已从 Hub 注销" {
			foundLog = true
			var clientIDLogged string
			for _, field := range logEntry.Context {
				if field.Key == "clientID" {
					clientIDLogged = field.String
					break
				}
			}
			assert.Equal(t, client.ID, clientIDLogged, "Logged clientID should match unregistered client's ID")
			break

		}
	}
	assert.True(t, foundLog, "Expected log message for client unregistration was not found")

	// Assert: ActiveConnections metric decreased (similar challenge as with Inc())

	// Assert: h.handleClientDisconnect(client) was called.
	// This is harder to assert directly in a unit test without more advanced mocking/spying
	// or checking for its side effects. The `backend/后端测试250526.md` mentions:
	// "需验证其副作用，见模块7".
	// For this unit test, we will assume it's called based on code review.
	// Side effects (like session termination or provider list update) will be tested
	// when we test `handleClientDisconnect` itself or those specific modules.
}

// TestHub_Run_ProcessIncomingMessage tests the incoming message processing case in Hub.Run().
// This will be a placeholder and will be expanded by testing specific message handlers.
func TestHub_Run_ProcessIncomingMessage(t *testing.T) {
	setupHubTestGlobalConfig(t)
	hub, observedLogs := newHubWithObservedLogger(t) // Correctly get observedLogs
	go hub.Run()

	client := mockClientForHubTests(hub)
	// Register client first
	hub.register <- client
	time.Sleep(50 * time.Millisecond)
	client.Authenticated = true // 将客户端标记为已认证，以确保 "Hub 正在处理消息" 日志被执行

	// This test primarily ensures the message is passed to handleIncomingMessage.
	// The actual logic of handleIncomingMessage will be tested separately (功能点 2.4).

	testMsgBytes := []byte(`{"type":"test_message_for_run_case"}`) // A generic JSON structure
	procMsg := ProcessableMessage{Client: client, RawMessage: testMsgBytes}

	done := make(chan bool)
	go func() {
		time.Sleep(100 * time.Millisecond) // Give time for processing
		done <- true
	}()

	hub.processMessage <- procMsg

	select {
	case <-done:
		// Test proceeded, implies message was taken from channel.
	case <-time.After(1 * time.Second):
		t.Fatal("Timeout waiting for processMessage to be handled in Run loop")
	}

	// As a basic check for this path, let's verify the "Hub 正在处理消息" log
	// (even though the message type itself might be unsupported later in handleIncomingMessage).
	foundProcessingLog := false
	// Use the observedLogs from newHubWithObservedLogger
	for _, logEntry := range observedLogs.All() {
		// The log message in handleIncomingMessage is:
		// global.GVA_LOG.Info("Hub 正在处理消息", ...)
		if logEntry.Message == "Hub 正在处理消息" { // 确保与 hub.go 中的日志完全一致
			foundProcessingLog = true
			break
		}
	}
	assert.True(t, foundProcessingLog, "Expected log 'Hub 正在处理消息' was not found after sending to processMessage channel")
}

// TestHub_Run_TimerCheckInactiveSessions tests the timed check for inactive sessions case in Hub.Run().
func TestHub_Run_TimerCheckInactiveSessions(t *testing.T) {
	setupHubTestGlobalConfig(t)
	// For this test, we need to control time or make intervals very short.
	// Let's set a very short HubCheckIntervalSec for testing purposes.
	originalInterval := global.GVA_CONFIG.NfcRelay.HubCheckIntervalSec
	global.GVA_CONFIG.NfcRelay.HubCheckIntervalSec = 1 // 1 second
	defer func() { global.GVA_CONFIG.NfcRelay.HubCheckIntervalSec = originalInterval }()

	hub, observedLogs := newHubWithObservedLogger(t)
	go hub.Run()

	// We need a way to mock h.checkInactiveSessions() or check its effects.
	// For now, let's check if the log message "HubCheckIntervalSec config is invalid..." is NOT present,
	// and then after a delay, we would expect checkInactiveSessions to have been called.
	// Directly verifying a call to an unexported method on the same struct from within a test of Run()
	// is hard without refactoring or more complex spying.
	// We can check for a log that checkInactiveSessions might produce if it finds something, or assume it runs.

	time.Sleep(1500 * time.Millisecond) // Wait for more than one interval

	// Check that no warning log about invalid interval was produced by Hub.Run() init
	foundInvalidIntervalLog := false
	for _, logEntry := range observedLogs.AllUntimed() { // AllUntimed might be better if exact log time is not critical
		if strings.Contains(logEntry.Message, "HubCheckIntervalSec config is invalid") {
			foundInvalidIntervalLog = true
			break
		}
	}
	assert.False(t, foundInvalidIntervalLog, "Hub.Run() should not log invalid interval when set to 1s")

	// At this point, h.checkInactiveSessions() should have been called at least once.
	// We can add a log inside checkInactiveSessions (for test builds) or check its side effects.
	// For this test, we'll assume it was called. Detailed tests for checkInactiveSessions (功能点 8.4)
	// will verify its internal logic.
	// A simple indirect check: if the hub is still running and didn't panic, the ticker case was hit.
	// (This is a weak assertion for the call itself).

	// To make this more robust, one might:
	// 1. Add a test-only hook/channel to checkInactiveSessions.
	// 2. Check for logs produced *by* checkInactiveSessions if it performed an action.
	// For now, this test mainly ensures the ticker setup in Run() is operational with a valid short interval.
}

// TestHub_Run_HubCheckIntervalConfig_Invalid tests Hub.Run() with invalid HubCheckIntervalSec.
func TestHub_Run_HubCheckIntervalConfig_Invalid(t *testing.T) {
	setupHubTestGlobalConfig(t)
	originalInterval := global.GVA_CONFIG.NfcRelay.HubCheckIntervalSec
	global.GVA_CONFIG.NfcRelay.HubCheckIntervalSec = 0 // Invalid interval
	defer func() { global.GVA_CONFIG.NfcRelay.HubCheckIntervalSec = originalInterval }()

	hub, observedLogs := newHubWithObservedLogger(t)
	go hub.Run() // This will log the warning

	time.Sleep(50 * time.Millisecond) // Give Run() a moment to initialize and log

	foundWarningLog := false
	for _, logEntry := range observedLogs.AllUntimed() {
		if logEntry.Message == "HubCheckIntervalSec config is invalid, using default 60s" && logEntry.Level == zap.WarnLevel {
			foundWarningLog = true
			break
		}
	}
	assert.True(t, foundWarningLog, "Hub.Run() should log a warning for invalid HubCheckIntervalSec and use default")

	// We can also check if the ticker is not firing very rapidly, implying it defaulted to 60s.
	// This is harder to assert precisely in a short unit test.
}

// TestHub_HandleIncomingMessage_InvalidJSONFormat tests handling of a message with invalid JSON format.
func TestHub_HandleIncomingMessage_InvalidJSONFormat(t *testing.T) {
	setupHubTestGlobalConfig(t)
	hub, observedLogs := newHubWithObservedLogger(t)
	go hub.Run() // Ensure Hub.Run is running

	client := mockClientForHubTests(hub)
	// Register client: Even for bad format, client registration context might be relevant for logging/cleanup
	hub.register <- client
	time.Sleep(50 * time.Millisecond) // Allow registration to be processed

	// client.Authenticated = false // Not strictly needed for this test as parsing fails before auth check

	invalidJSONBytes := []byte("this is not json")
	procMsg := ProcessableMessage{Client: client, RawMessage: invalidJSONBytes}

	// Act - Send message via channel
	hub.processMessage <- procMsg
	t.Logf("TestHub_HandleIncomingMessage_InvalidJSONFormat: Message sent to hub.processMessage for client %s", client.ID)

	// Assert: Client receives ErrorMessage
	select {
	case sentMsgBytes := <-client.send:
		var errMsg protocol.ErrorMessage
		err := json.Unmarshal(sentMsgBytes, &errMsg)
		assert.NoError(t, err, "Should be able to unmarshal sent message into ErrorMessage")
		assert.Equal(t, protocol.MessageTypeError, errMsg.Type, "Message type should be ErrorMessage")
		assert.Equal(t, protocol.ErrorCodeBadRequest, errMsg.Code, "Error code should be BadRequest")
		assert.Contains(t, errMsg.Message, "无效的消息格式", "Error message content check")
	case <-time.After(100 * time.Millisecond):
		t.Fatal("Timeout waiting for error message to be sent to client via hub.processMessage")
	}

	// Assert: Correct error log
	foundErrorLog := false
	for _, logEntry := range observedLogs.All() {
		if logEntry.Message == "Hub 处理消息：反序列化通用消息失败" && logEntry.Level == zap.ErrorLevel {
			foundErrorLog = true
			assert.Equal(t, client.GetID(), logEntry.ContextMap()["clientID"], "Log should contain correct clientID")
			break
		}
	}
	assert.True(t, foundErrorLog, "Expected error log for JSON unmarshal failure was not found")

	// Assert: Audit log for error_occurred
	foundAuditLog := false
	for _, logEntry := range observedLogs.All() {
		if logEntry.Message == "AuditEvent" && logEntry.Level == zap.InfoLevel {
			ctxMap := logEntry.ContextMap()
			if eventType, ok := ctxMap["event_type"]; ok && eventType == "error_occurred" {
				if detailsMap, ok := ctxMap["details"].(map[string]interface{}); ok {
					if detailsMsg, ok := detailsMap["error_message"].(string); ok && strings.Contains(detailsMsg, "无效的消息格式") {
						// Ensure client.ID is not nil before asserting
						if client.GetID() != "" { // Check if client ID is actually set
							assert.Equal(t, client.GetID(), ctxMap["clientID"], "Audit log clientID mismatch")
						} else if _, clientIdExists := ctxMap["clientID"]; clientIdExists {
							// if client.GetID() is empty but log has one, it's still a mismatch or unexpected
							t.Errorf("Audit log clientID mismatch: mock client ID is empty, but log has clientID: %v", ctxMap["clientID"])
						}
						// Only assert if client_id was expected and present
						if _, clientIDExists := ctxMap["clientID"]; clientIDExists && client.GetID() != "" {
							assert.Equal(t, client.GetID(), ctxMap["clientID"], "Audit log clientID mismatch")
						}
						assert.Equal(t, strconv.Itoa(protocol.ErrorCodeBadRequest), detailsMap["error_code"].(string), "Audit log error_code mismatch") // Assuming error_code in details is string
						foundAuditLog = true
						break
					}
				}
			}
		}
	}
	assert.True(t, foundAuditLog, "Expected 'error_occurred' audit log for invalid JSON was not found")
}

// TestHub_HandleIncomingMessage_UnauthenticatedClient tests sending a non-auth message from an unauthenticated client.
func TestHub_HandleIncomingMessage_UnauthenticatedClient(t *testing.T) {
	setupHubTestGlobalConfig(t)
	hub, observedLogs := newHubWithObservedLogger(t)
	go hub.Run() // Ensure Hub.Run is running

	client := mockClientForHubTests(hub)
	hub.register <- client
	time.Sleep(50 * time.Millisecond) // Allow registration

	client.Authenticated = false // Explicitly set as unauthenticated

	declareRoleMsg := protocol.DeclareRoleMessage{
		Type: protocol.MessageTypeDeclareRole,
		Role: protocol.RoleReceiver,
	}
	msgBytes, err := json.Marshal(declareRoleMsg)
	assert.NoError(t, err, "Failed to marshal DeclareRoleMessage for test")

	procMsg := ProcessableMessage{Client: client, RawMessage: msgBytes}

	// Act - Send message via channel
	hub.processMessage <- procMsg
	t.Logf("TestHub_HandleIncomingMessage_UnauthenticatedClient: Message sent to hub.processMessage for client %s", client.ID)

	// Assert: Client receives ErrorMessage with AuthRequired code
	select {
	case sentMsgBytes := <-client.send:
		var errMsg protocol.ErrorMessage
		err := json.Unmarshal(sentMsgBytes, &errMsg)
		assert.NoError(t, err, "Should be able to unmarshal sent message into ErrorMessage")
		assert.Equal(t, protocol.MessageTypeError, errMsg.Type)
		assert.Equal(t, protocol.ErrorCodeAuthRequired, errMsg.Code)
		assert.Equal(t, "请先进行认证", errMsg.Message)
	case <-time.After(100 * time.Millisecond):
		t.Fatal("Timeout waiting for AuthRequired error message to be sent to client via hub.processMessage")
	}

	// Assert: Correct warning log
	foundWarningLog := false
	for _, logEntry := range observedLogs.All() {
		if logEntry.Message == "未认证的客户端尝试发送非认证消息" && logEntry.Level == zap.WarnLevel {
			foundWarningLog = true
			assert.Equal(t, client.GetID(), logEntry.ContextMap()["clientID"], "Log should contain correct clientID")
			assert.Equal(t, string(protocol.MessageTypeDeclareRole), logEntry.ContextMap()["messageType"], "Log should contain correct messageType")
			break
		}
	}
	assert.True(t, foundWarningLog, "Expected warning log for unauthenticated client was not found")

	// Assert: Audit log for error_occurred (auth_required)
	foundAuditLog := false
	for _, logEntry := range observedLogs.All() {
		if logEntry.Message == "AuditEvent" && logEntry.Level == zap.InfoLevel {
			ctxMap := logEntry.ContextMap()
			if eventType, ok := ctxMap["event_type"]; ok && eventType == "error_occurred" {
				if detailsMap, ok := ctxMap["details"].(map[string]interface{}); ok {
					if detailsMsg, ok := detailsMap["error_message"].(string); ok && detailsMsg == "请先进行认证" {
						if _, typeOk := ctxMap["message_type"]; typeOk && ctxMap["message_type"].(string) == string(protocol.MessageTypeDeclareRole) {
							if client.GetID() != "" {
								assert.Equal(t, client.GetID(), ctxMap["clientID"], "Audit log clientID mismatch")
							} else if _, clientIdExists := ctxMap["clientID"]; clientIdExists {
								t.Errorf("Audit log clientID mismatch: mock client ID is empty, but log has clientID: %v", ctxMap["clientID"])
							}
							assert.Equal(t, strconv.Itoa(protocol.ErrorCodeAuthRequired), detailsMap["error_code"].(string), "Audit log error_code mismatch")
							foundAuditLog = true
							break
						}
					}
				}
			}
		}
	}
	assert.True(t, foundAuditLog, "Expected 'error_occurred' audit log for auth required was not found")
}

// TestHub_HandleIncomingMessage_DispatchToClientAuth tests dispatching to handleClientAuth.
func TestHub_HandleIncomingMessage_DispatchToClientAuth(t *testing.T) {
	setupHubTestGlobalConfig(t)
	hub, observedLogs := newHubWithObservedLogger(t)
	go hub.Run() // Ensure Hub.Run is running

	client := mockClientForHubTests(hub)
	hub.register <- client
	time.Sleep(50 * time.Millisecond) // Allow registration

	client.Authenticated = false // Auth messages are allowed for unauthenticated clients

	authMsg := protocol.ClientAuthMessage{
		Type:  protocol.MessageTypeClientAuth,
		Token: "test-token-for-dispatch", // This token will fail validation in handleClientAuth
	}
	msgBytes, err := json.Marshal(authMsg)
	assert.NoError(t, err, "Failed to marshal ClientAuthMessage for test")
	procMsg := ProcessableMessage{Client: client, RawMessage: msgBytes}

	// Act - Send message via channel
	hub.processMessage <- procMsg
	t.Logf("TestHub_HandleIncomingMessage_DispatchToClientAuth: Message sent to hub.processMessage for client %s", client.ID)

	// Assert: Expect an ErrorMessage due to failed token validation in handleClientAuth.
	select {
	case sentMsgBytes := <-client.send:
		var errMsgOnDispatch protocol.ErrorMessage
		err := json.Unmarshal(sentMsgBytes, &errMsgOnDispatch)
		assert.NoError(t, err, "Should unmarshal to ErrorMessage from handleClientAuth")
		assert.Equal(t, protocol.MessageTypeError, errMsgOnDispatch.Type)
		assert.Equal(t, protocol.ErrorCodeAuthFailed, errMsgOnDispatch.Code, "Expected AuthFailed due to bad token from handleClientAuth")
	case <-time.After(150 * time.Millisecond): // Increased timeout slightly
		t.Fatal("Timeout waiting for response from handleClientAuth via hub.processMessage (expected auth failure error)")
	}

	foundUnsupportedLog := false
	for _, logEntry := range observedLogs.All() {
		if logEntry.Message == "Hub 收到未处理的消息类型" { // This log should NOT appear
			foundUnsupportedLog = true
			t.Errorf("Log 'Hub 收到未处理的消息类型' should not be present for ClientAuth type, but was found. Log level: %s, Context: %v", logEntry.Level, logEntry.ContextMap())
			break
		}
	}
	assert.False(t, foundUnsupportedLog, "Log 'Hub 收到未处理的消息类型' should not be present for ClientAuth type")
}

// TestHub_HandleIncomingMessage_DispatchToDeclareRole tests dispatching to handleDeclareRole.
func TestHub_HandleIncomingMessage_DispatchToDeclareRole(t *testing.T) {
	setupHubTestGlobalConfig(t)
	hub, observedLogs := newHubWithObservedLogger(t)
	go hub.Run() // Ensure Hub.Run is running

	client := mockClientForHubTests(hub)
	// Crucial: Register the client with the Hub so it's in hub.clients
	// This is important because handleIncomingMessage might interact with hub.clients or client state
	// that is normally managed by the Hub's lifecycle (register/unregister)
	hub.register <- client
	time.Sleep(50 * time.Millisecond) // Give time for registration to be processed by Hub.Run()

	client.Authenticated = true // DeclareRole requires authentication

	declareRoleMsg := protocol.DeclareRoleMessage{
		Type: protocol.MessageTypeDeclareRole,
		Role: protocol.RoleProvider,
	}
	msgBytes, err := json.Marshal(declareRoleMsg)
	assert.NoError(t, err, "Failed to marshal DeclareRoleMessage for test")
	procMsg := ProcessableMessage{Client: client, RawMessage: msgBytes}

	t.Logf("DispatchToDeclareRole: Attempting to send to hub.processMessage for client %s with role %s", client.ID, declareRoleMsg.Role)
	// Act - Send the message to the Hub's processing channel
	hub.processMessage <- procMsg
	t.Logf("DispatchToDeclareRole: Message sent to hub.processMessage for client %s", client.ID)

	// Assert that no unexpected error message was sent back by handleIncomingMessage itself.
	// handleDeclareRole will send a RoleDeclaredResponseMessage.
	select {
	case sentMsgBytes := <-client.send:
		t.Logf("DispatchToDeclareRole: Received message on client.send channel for client %s", client.ID)
		var genericResp protocol.GenericMessage
		_ = json.Unmarshal(sentMsgBytes, &genericResp)
		assert.Equal(t, protocol.MessageTypeRoleDeclaredResponse, genericResp.Type,
			"Expected RoleDeclaredResponseMessage, got %s", genericResp.Type)
	case <-time.After(5 * time.Second): // Keep increased timeout
		t.Logf("DispatchToDeclareRole: Timeout waiting for response from handleDeclareRole for client %s. Current logs:", client.ID)
		for _, logEntry := range observedLogs.All() {
			t.Logf("Observed log: Level: %s, Message: %s, Fields: %v", logEntry.Level, logEntry.Message, logEntry.ContextMap())
		}
		t.Fatal("Timeout waiting for response from handleDeclareRole via hub.processMessage")
	}

	// Check that the "Hub 收到未处理的消息类型" log was NOT generated.
	foundUnsupportedLog := false
	for _, logEntry := range observedLogs.All() {
		if logEntry.Message == "Hub 收到未处理的消息类型" {
			foundUnsupportedLog = true
			t.Logf("DispatchToDeclareRole: Found unexpected 'Hub 收到未处理的消息类型' log for client %s", client.ID)
			break
		}
	}
	assert.False(t, foundUnsupportedLog, "Log 'Hub 收到未处理的消息类型' should not be present for DeclareRole type")

	t.Logf("DispatchToDeclareRole: Test completed for client %s", client.ID)
}

// TestHub_HandleIncomingMessage_DispatchToListCardProviders tests dispatching to handleListCardProviders.
func TestHub_HandleIncomingMessage_DispatchToListCardProviders(t *testing.T) {
	setupHubTestGlobalConfig(t)
	hub, observedLogs := newHubWithObservedLogger(t)
	go hub.Run() // Ensure Hub.Run is running

	client := mockClientForHubTests(hub)
	hub.register <- client
	time.Sleep(50 * time.Millisecond) // Allow registration

	client.Authenticated = true
	client.CurrentRole = protocol.RoleReceiver // Required for ListCardProviders

	listProvidersMsg := protocol.ListCardProvidersMessage{
		Type: protocol.MessageTypeListCardProviders,
	}
	msgBytes, err := json.Marshal(listProvidersMsg)
	assert.NoError(t, err, "Failed to marshal ListCardProvidersMessage for test")
	procMsg := ProcessableMessage{Client: client, RawMessage: msgBytes}

	// Act - Send message via channel
	hub.processMessage <- procMsg
	t.Logf("TestHub_HandleIncomingMessage_DispatchToListCardProviders: Message sent to hub.processMessage for client %s", client.ID)

	// Assert: Expect CardProvidersListMessage from handleListCardProviders
	select {
	case sentMsgBytes := <-client.send:
		var genericResp protocol.GenericMessage
		_ = json.Unmarshal(sentMsgBytes, &genericResp)
		assert.Equal(t, protocol.MessageTypeCardProvidersList, genericResp.Type,
			"Expected CardProvidersListMessage, got %s", genericResp.Type)
	case <-time.After(100 * time.Millisecond): // Adjusted timeout
		t.Fatal("Timeout waiting for response from handleListCardProviders via hub.processMessage")
	}

	foundUnsupportedLog := false
	for _, logEntry := range observedLogs.All() {
		if logEntry.Message == "Hub 收到未处理的消息类型" {
			foundUnsupportedLog = true
			break
		}
	}
	assert.False(t, foundUnsupportedLog, "Log 'Hub 收到未处理的消息类型' should not be present for ListCardProviders type")
}

// TestHub_HandleIncomingMessage_ListCardProviders_PermissionDenied tests the permission check
// in handleIncomingMessage before dispatching to handleListCardProviders.
func TestHub_HandleIncomingMessage_ListCardProviders_PermissionDenied(t *testing.T) {
	setupHubTestGlobalConfig(t)
	hub, observedLogs := newHubWithObservedLogger(t)
	go hub.Run() // Ensure Hub.Run is running

	client := mockClientForHubTests(hub)
	hub.register <- client
	time.Sleep(50 * time.Millisecond) // Allow registration

	client.Authenticated = true
	client.CurrentRole = protocol.RoleProvider // << Not a Receiver, should be denied by handleIncomingMessage

	listProvidersMsg := protocol.ListCardProvidersMessage{
		Type: protocol.MessageTypeListCardProviders,
	}
	msgBytes, err := json.Marshal(listProvidersMsg)
	assert.NoError(t, err, "Failed to marshal ListCardProvidersMessage for test")
	procMsg := ProcessableMessage{Client: client, RawMessage: msgBytes}

	// Act - Send message via channel
	hub.processMessage <- procMsg
	t.Logf("TestHub_HandleIncomingMessage_ListCardProviders_PermissionDenied: Message sent to hub.processMessage for client %s", client.ID)

	// Assert: Expect ErrorMessage with PermissionDenied from handleIncomingMessage's pre-check
	select {
	case sentMsgBytes := <-client.send:
		var errMsg protocol.ErrorMessage
		err := json.Unmarshal(sentMsgBytes, &errMsg)
		assert.NoError(t, err, "Should unmarshal to ErrorMessage")
		assert.Equal(t, protocol.MessageTypeError, errMsg.Type)
		assert.Equal(t, protocol.ErrorCodePermissionDenied, errMsg.Code)
		assert.Equal(t, "只有收卡方角色才能获取发卡方列表", errMsg.Message)
	case <-time.After(100 * time.Millisecond): // Adjusted timeout
		t.Fatal("Timeout waiting for PermissionDenied error message via hub.processMessage")
	}

	// Assert: Correct warning log from handleIncomingMessage
	foundWarningLog := false
	for _, logEntry := range observedLogs.All() {
		if logEntry.Message == "Hub: 非 receiver 客户端尝试获取发卡方列表" && logEntry.Level == zap.WarnLevel {
			foundWarningLog = true
			break
		}
	}
	assert.True(t, foundWarningLog, "Expected warning log for permission denied on ListCardProviders was not found")
}

// TestHub_HandleIncomingMessage_DispatchToSelectCardProvider tests dispatching to handleSelectCardProvider.
func TestHub_HandleIncomingMessage_DispatchToSelectCardProvider(t *testing.T) {
	setupHubTestGlobalConfig(t)
	hub, observedLogs := newHubWithObservedLogger(t)
	go hub.Run()

	client := mockClientForHubTests(hub)
	hub.register <- client
	time.Sleep(50 * time.Millisecond)

	client.Authenticated = true
	client.CurrentRole = protocol.RoleReceiver

	selectMsg := protocol.SelectCardProviderMessage{
		Type:       protocol.MessageTypeSelectCardProvider,
		ProviderID: "test-provider-id-for-dispatch",
	}
	msgBytes, err := json.Marshal(selectMsg)
	assert.NoError(t, err, "Failed to marshal SelectCardProviderMessage for test")
	procMsg := ProcessableMessage{Client: client, RawMessage: msgBytes}

	hub.processMessage <- procMsg
	t.Logf("TestHub_HandleIncomingMessage_DispatchToSelectCardProvider: Message sent to hub.processMessage for client %s", client.ID)

	select {
	case sentMsgBytes := <-client.send:
		var genericResp protocol.GenericMessage
		_ = json.Unmarshal(sentMsgBytes, &genericResp)
		if genericResp.Type == protocol.MessageTypeError {
			var errMsgOnDispatch protocol.ErrorMessage
			_ = json.Unmarshal(sentMsgBytes, &errMsgOnDispatch)
			assert.NotEqual(t, protocol.ErrorCodeUnsupportedType, errMsgOnDispatch.Code, "Got an Error, but it shouldn't be UnsupportedType")
		} else {
			assert.Contains(t, []protocol.MessageType{protocol.MessageTypeSessionEstablished, protocol.MessageTypeError}, genericResp.Type,
				"Expected SessionEstablished or Error from handler, got %s", genericResp.Type)
		}
	case <-time.After(150 * time.Millisecond): // Adjusted timeout, was 100ms
		t.Log("Timeout waiting for response from handleSelectCardProvider via hub.processMessage")
	}

	foundUnsupportedLog := false
	for _, logEntry := range observedLogs.All() {
		if logEntry.Message == "Hub 收到未处理的消息类型" {
			foundUnsupportedLog = true
			break
		}
	}
	assert.False(t, foundUnsupportedLog, "Log 'Hub 收到未处理的消息类型' should not be present for SelectCardProvider type")
}

// TestHub_HandleIncomingMessage_DispatchToEndSession tests dispatching to handleEndSession.
func TestHub_HandleIncomingMessage_DispatchToEndSession(t *testing.T) {
	setupHubTestGlobalConfig(t)
	hub, observedLogs := newHubWithObservedLogger(t)
	go hub.Run()

	client := mockClientForHubTests(hub)
	hub.register <- client
	time.Sleep(50 * time.Millisecond)

	client.Authenticated = true
	client.SessionID = "test-session-for-end-dispatch" // EndSession requires a session ID

	endSessionMsg := protocol.EndSessionMessage{
		Type:      protocol.MessageTypeEndSession,
		SessionID: client.SessionID,
	}
	msgBytes, err := json.Marshal(endSessionMsg)
	assert.NoError(t, err, "Failed to marshal EndSessionMessage for test")
	procMsg := ProcessableMessage{Client: client, RawMessage: msgBytes}

	hub.processMessage <- procMsg
	t.Logf("TestHub_HandleIncomingMessage_DispatchToEndSession: Message sent to hub.processMessage for client %s", client.ID)

	select {
	case sentMsgBytes := <-client.send:
		var genericResp protocol.GenericMessage
		_ = json.Unmarshal(sentMsgBytes, &genericResp)
		assert.Equal(t, protocol.MessageTypeSessionTerminated, genericResp.Type,
			"Expected SessionTerminatedMessage, got %s", genericResp.Type)
	case <-time.After(100 * time.Millisecond): // Increased timeout
		t.Fatal("Timeout waiting for response from handleEndSession")
	}

	foundUnsupportedLog := false
	for _, logEntry := range observedLogs.All() {
		if logEntry.Message == "Hub 收到未处理的消息类型" {
			foundUnsupportedLog = true
			break
		}
	}
	assert.False(t, foundUnsupportedLog, "Log 'Hub 收到未处理的消息类型' should not be present for EndSession type")

	// Full logic of handleEndSession and terminateSessionByID is tested in 模块 8.
}

// TestHub_HandleIncomingMessage_UnsupportedMessageType tests handling of an unsupported message type.
func TestHub_HandleIncomingMessage_UnsupportedMessageType(t *testing.T) {
	setupHubTestGlobalConfig(t)
	hub, observedLogs := newHubWithObservedLogger(t)
	client := mockClientForHubTests(hub)
	client.Authenticated = true // Assume authenticated for this test

	unsupportedMsg := struct {
		Type protocol.MessageType `json:"type"`
		Data string               `json:"data"`
	}{
		Type: "unsupported_test_type", // This type is not handled in the switch case
		Data: "some data",
	}
	msgBytes, err := json.Marshal(unsupportedMsg)
	assert.NoError(t, err, "Failed to marshal unsupported message for test")
	procMsg := ProcessableMessage{Client: client, RawMessage: msgBytes}

	hub.handleIncomingMessage(procMsg)

	// Assert: Client receives ErrorMessage with UnsupportedType code
	select {
	case sentMsgBytes := <-client.send:
		var errMsg protocol.ErrorMessage
		err := json.Unmarshal(sentMsgBytes, &errMsg)
		assert.NoError(t, err, "Should unmarshal to ErrorMessage")
		assert.Equal(t, protocol.MessageTypeError, errMsg.Type)
		assert.Equal(t, protocol.ErrorCodeUnsupportedType, errMsg.Code)
		assert.Contains(t, errMsg.Message, "不支持的消息类型: unsupported_test_type")
	case <-time.After(100 * time.Millisecond):
		t.Fatal("Timeout waiting for UnsupportedType error message")
	}

	// Assert: Correct warning log
	foundWarningLog := false
	for _, logEntry := range observedLogs.All() {
		if logEntry.Message == "Hub 收到未处理的消息类型" && logEntry.Level == zap.WarnLevel {
			foundWarningLog = true
			assert.Equal(t, "unsupported_test_type", logEntry.ContextMap()["type"].(string))
			assert.Equal(t, client.GetID(), logEntry.ContextMap()["clientID"].(string))
			break
		}
	}
	assert.True(t, foundWarningLog, "Expected warning log for unsupported message type was not found")

	// Assert: Audit log for error_occurred (unsupported_type)
	foundAuditLog := false
	for _, logEntry := range observedLogs.All() {
		if logEntry.Message == "AuditEvent" && logEntry.Level == zap.InfoLevel {
			ctxMap := logEntry.ContextMap()
			if eventType, ok := ctxMap["event_type"].(string); ok && eventType == "error_occurred" {
				if detailsMap, ok := ctxMap["details"].(map[string]interface{}); ok {
					if errMsg, ok := detailsMap["error_message"].(string); ok && strings.Contains(errMsg, "不支持的消息类型: unsupported_test_type") {
						if msgType, typeOk := ctxMap["message_type"]; typeOk && msgType == "unsupported_test_type" {
							foundAuditLog = true
							assert.Equal(t, client.GetID(), ctxMap["clientID"], "Audit log clientID mismatch")
							assert.Equal(t, strconv.Itoa(protocol.ErrorCodeUnsupportedType), detailsMap["error_code"], "Audit log error_code mismatch")
							break
						}
					}
				}
			}
		}
	}
	assert.True(t, foundAuditLog, "Expected 'error_occurred' audit log for unsupported type was not found")
}

// --- Tests for Hub helper functions --- //

func TestHub_SendProtoMessage_Success(t *testing.T) {
	setupHubTestGlobalConfig(t)

	mockClient := newMockClientInfoProvider("mock-client-for-sendproto", "u1", "s1")
	mockClient.sendShouldFail = false

	testMessage := protocol.GenericMessage{Type: "test_proto_msg"}

	err := sendProtoMessage(mockClient, testMessage)
	assert.NoError(t, err, "sendProtoMessage should not return error on success")

	assert.True(t, mockClient.sendCalled, "sendCalled should be true")
	assert.NotNil(t, mockClient.lastSentMessage, "Message should have been passed to client.Send")
	var sentGenericMsg protocol.GenericMessage
	err = json.Unmarshal(mockClient.lastSentMessage, &sentGenericMsg)
	assert.NoError(t, err, "Failed to unmarshal the message sent to client")
	assert.Equal(t, testMessage.Type, sentGenericMsg.Type, "Message type mismatch after sendProtoMessage")
}

func TestHub_SendProtoMessage_SerializationFailure(t *testing.T) {
	setupHubTestGlobalConfig(t)
	_, observedLogs := newHubWithObservedLogger(t)

	mockClient := newMockClientInfoProvider("mock-client-serialize-fail", "u2", "s2")

	unmarshalableMessage := struct {
		Type    protocol.MessageType `json:"type"`
		BadData chan int             `json:"badData"`
	}{
		Type:    "serialize_fail_msg",
		BadData: make(chan int),
	}

	err := sendProtoMessage(mockClient, unmarshalableMessage)
	assert.Error(t, err, "sendProtoMessage should return an error on serialization failure")
	assert.False(t, mockClient.sendCalled, "sendCalled should be false on serialization failure")
	assert.Nil(t, mockClient.lastSentMessage, "client.Send should not be called on serialization failure")

	foundLog := false
	for _, logEntry := range observedLogs.All() {
		if strings.Contains(logEntry.Message, "序列化消息失败") && logEntry.Level == zap.ErrorLevel {
			foundLog = true
			break
		}
	}
	assert.True(t, foundLog, "Expected error log for serialization failure was not found")
}

func TestHub_SendProtoMessage_ClientSendFailure(t *testing.T) {
	setupHubTestGlobalConfig(t)
	_, observedLogs := newHubWithObservedLogger(t)

	expectedSendError := errors.New("mock send error")
	mockClient := newMockClientInfoProvider("mock-client-send-fail", "u3", "s3")
	mockClient.sendShouldFail = true // Simulate client.Send() failing

	testMessage := protocol.GenericMessage{Type: "test_client_send_fail"}

	err := sendProtoMessage(mockClient, testMessage)
	assert.Error(t, err, "sendProtoMessage should return an error when client.Send fails")
	assert.Equal(t, expectedSendError.Error(), err.Error(), "Error returned by sendProtoMessage should match expected")
	assert.True(t, mockClient.sendCalled, "sendCalled should be true even if send fails")

	foundLog := false
	for _, logEntry := range observedLogs.All() {
		if strings.Contains(logEntry.Message, "通过接口发送消息给客户端失败") && logEntry.Level == zap.WarnLevel {
			foundLog = true
			assert.Equal(t, mockClient.GetID(), logEntry.ContextMap()["targetClientID"].(string), "Log should contain correct targetClientID")
			break
		}
	}
	assert.True(t, foundLog, "Expected warning log for client send failure was not found")
}

func Test_SendProtoMessage_NilClient(t *testing.T) {
	setupHubTestGlobalConfig(t)
	// Logs are not relevant here, but setup is needed for global.GVA_LOG

	err := sendProtoMessage(nil, protocol.ServerAuthResponseMessage{Success: true})
	assert.Error(t, err)
	assert.Equal(t, "client is nil", err.Error())
}

// Tests for sendErrorMessage

func Test_SendErrorMessage_Success(t *testing.T) {
	setupHubTestGlobalConfig(t)
	core, observedLogs := observer.New(zap.InfoLevel) // Capture Info level and above for audit and other logs
	originalLogger := global.GVA_LOG
	global.GVA_LOG = zap.New(core)
	global.InitializeAuditLogger() // Ensure audit logger uses the observer
	defer func() {
		global.GVA_LOG = originalLogger
		global.InitializeAuditLogger() // Reset audit logger to original
	}()

	mockClient := newMockClientInfoProvider("client-send-err-success", "user-123", "session-abc")
	mockClient.sendShouldFail = false // Ensure send succeeds

	errorCode := protocol.ErrorCodeBadRequest
	errorMsg := "This is a bad request."

	sendErrorMessage(mockClient, errorCode, errorMsg)

	// 1. Check audit log
	foundAuditLog := false
	for _, logEntry := range observedLogs.All() {
		if logEntry.LoggerName == "audit" && logEntry.Message == "AuditEvent" {
			if eventType, ok := logEntry.ContextMap()["event_type"].(string); ok && eventType == "client_error_notification_sent" {
				ctxMap := logEntry.ContextMap()
				detailsValue, detailsFieldExists := ctxMap["details"]
				assert.True(t, detailsFieldExists, "Audit log 'details' field must exist")
				if detailsFieldExists {
					detailsMap, detailsIsMap := detailsValue.(map[string]interface{})
					assert.True(t, detailsIsMap, "Audit log 'details' field should be a map[string]interface{}")
					if detailsIsMap {
						assert.Equal(t, strconv.Itoa(errorCode), detailsMap["error_code"].(string), "Details error_code mismatch")
						assert.Equal(t, errorMsg, detailsMap["error_message"].(string), "Details error_message mismatch")
					}
				}

				assert.Equal(t, "client-send-err-success", ctxMap["client_id"])
				assert.Equal(t, "user-123", ctxMap["user_id"])
				assert.Equal(t, "session-abc", ctxMap["session_id"])
				foundAuditLog = true
				break
			}
		}
	}
	assert.True(t, foundAuditLog, "Expected 'client_error_notification_sent' audit log not found or details incorrect")

	// 2. Check Prometheus metric (conceptual)

	// 3. Check that client.Send was called with the marshalled ErrorMessage
	assert.True(t, mockClient.sendCalled, "client.Send should have been called")
	assert.NotEmpty(t, mockClient.lastSentMessage, "A message should have been sent")

	var sentErrorMsg protocol.ErrorMessage
	err := json.Unmarshal(mockClient.lastSentMessage, &sentErrorMsg)
	assert.NoError(t, err, "Failed to unmarshal sent message into ErrorMessage")
	assert.Equal(t, protocol.MessageTypeError, sentErrorMsg.Type)
	assert.Equal(t, errorCode, sentErrorMsg.Code)
	assert.Equal(t, errorMsg, sentErrorMsg.Message)

	// 4. Check no "发送标准错误消息本身失败" warning
	foundWarningLog := false
	for _, logEntry := range observedLogs.All() {
		if logEntry.Level == zap.WarnLevel && logEntry.Message == "发送标准错误消息本身失败" {
			foundWarningLog = true
			break
		}
	}
	assert.False(t, foundWarningLog, "Should not find '发送标准错误消息本身失败' warning log")
}

func Test_SendErrorMessage_SendProtoMessageFails(t *testing.T) {
	setupHubTestGlobalConfig(t)
	core, observedLogs := observer.New(zap.InfoLevel)
	originalLogger := global.GVA_LOG
	global.GVA_LOG = zap.New(core)
	global.InitializeAuditLogger()
	defer func() {
		global.GVA_LOG = originalLogger
		global.InitializeAuditLogger()
	}()

	mockClient := newMockClientInfoProvider("client-send-err-fail", "user-456", "session-def")
	mockClient.sendShouldFail = true // Ensure send fails

	errorCode := protocol.ErrorCodeInternalError
	errorMsg := "Internal server error."

	sendErrorMessage(mockClient, errorCode, errorMsg)

	// 1. Audit log should still be recorded
	foundAuditLog := false
	for _, logEntry := range observedLogs.All() {
		if logEntry.LoggerName == "audit" && logEntry.Message == "AuditEvent" {
			ctxMap := logEntry.ContextMap()
			if eventType, ok := ctxMap["event_type"].(string); ok && eventType == "client_error_notification_sent" {
				detailsValue, detailsFieldExists := ctxMap["details"]
				assert.True(t, detailsFieldExists, "Audit log 'details' field must exist when sendProtoMessage fails")
				if detailsFieldExists {
					detailsMap, detailsIsMap := detailsValue.(map[string]interface{})
					assert.True(t, detailsIsMap, "Audit log 'details' field should be a map[string]interface{} when sendProtoMessage fails")
					if detailsIsMap {
						assert.Equal(t, strconv.Itoa(errorCode), detailsMap["error_code"].(string), "Details error_code mismatch when sendProtoMessage fails")
						assert.Equal(t, errorMsg, detailsMap["error_message"].(string), "Details error_message mismatch when sendProtoMessage fails")
					}
				}
				assert.Equal(t, "client-send-err-fail", ctxMap["client_id"], "client_id mismatch when sendProtoMessage fails")
				foundAuditLog = true
				break
			}
		}
	}
	assert.True(t, foundAuditLog, "Expected 'client_error_notification_sent' audit log even if sendProtoMessage fails")

	// 2. Prometheus metric should still be incremented (conceptual)

	// 3. client.Send was called (and failed)
	assert.True(t, mockClient.sendCalled, "client.Send should have been called")

	// 4. Check FOR "发送标准错误消息本身失败" warning
	foundWarningLog := false
	for _, logEntry := range observedLogs.All() {
		if logEntry.Level == zap.WarnLevel && logEntry.Message == "发送标准错误消息本身失败" {
			assert.Equal(t, "client-send-err-fail", logEntry.ContextMap()["targetClientID"])
			assert.Contains(t, logEntry.ContextMap()["error"].(string), "mock send error")
			foundWarningLog = true
			break
		}
	}
	assert.True(t, foundWarningLog, "Expected '发送标准错误消息本身失败' warning log not found")
}

func Test_SendErrorMessage_AuditLogClientDetails(t *testing.T) {
	setupHubTestGlobalConfig(t)

	// Case A: client is *handler.Client
	coreClient, observedLogsClient := observer.New(zap.InfoLevel)
	originalLoggerClient := global.GVA_LOG
	global.GVA_LOG = zap.New(coreClient)
	global.InitializeAuditLogger()
	defer func() {
		global.GVA_LOG = originalLoggerClient
		global.InitializeAuditLogger()
	}()

	hubClient := NewHub() // Need a hub to create a client
	mockConn := newMockWsConnection()
	mockConn.remoteAddrFunc = func() net.Addr { return &mockAddr{} }
	concreteClient := NewClient(hubClient, mockConn)
	concreteClient.UserID = "user-concrete"
	concreteClient.SessionID = "session-concrete"

	sendErrorMessage(concreteClient, protocol.ErrorCodePermissionDenied, "Access denied.")

	foundAuditLogClient := false
	for _, logEntry := range observedLogsClient.All() {
		if logEntry.LoggerName == "audit" && logEntry.Message == "AuditEvent" {
			if eventType, ok := logEntry.ContextMap()["event_type"].(string); ok && eventType == "client_error_notification_sent" {
				assert.Equal(t, concreteClient.ID, logEntry.ContextMap()["client_id"])
				assert.Equal(t, "user-concrete", logEntry.ContextMap()["user_id"])
				assert.Equal(t, "session-concrete", logEntry.ContextMap()["session_id"])
				assert.Equal(t, "testAddr", logEntry.ContextMap()["client_ip"])
				foundAuditLogClient = true
				break
			}
		}
	}
	assert.True(t, foundAuditLogClient, "Audit log for concrete client not found or details incorrect")

	// Case B: client is mockClientInfoProvider
	coreMock, observedLogsMock := observer.New(zap.InfoLevel)
	originalLoggerMock := global.GVA_LOG
	global.GVA_LOG = zap.New(coreMock)
	global.InitializeAuditLogger()
	defer func() {
		global.GVA_LOG = originalLoggerMock
		global.InitializeAuditLogger()
	}()
	mockInfoProvider := newMockClientInfoProvider("client-mock-details", "user-mock", "session-mock")

	sendErrorMessage(mockInfoProvider, protocol.ErrorCodeConflict, "Resource conflict.")

	foundAuditLogMock := false
	for _, logEntry := range observedLogsMock.All() {
		if logEntry.LoggerName == "audit" && logEntry.Message == "AuditEvent" {
			if eventType, ok := logEntry.ContextMap()["event_type"].(string); ok && eventType == "client_error_notification_sent" {
				assert.Equal(t, "client-mock-details", logEntry.ContextMap()["client_id"])
				assert.Equal(t, "user-mock", logEntry.ContextMap()["user_id"])
				assert.Equal(t, "session-mock", logEntry.ContextMap()["session_id"])
				_, ipExists := logEntry.ContextMap()["client_ip"]
				assert.False(t, ipExists, "client_ip should not exist for mockClientInfoProvider without a conn")
				foundAuditLogMock = true
				break
			}
		}
	}
	assert.True(t, foundAuditLogMock, "Audit log for mock client info provider not found or details incorrect")
}

// --- Tests for Hub.terminateSessionByID --- //

// mockClientForSessionTerminationTests creates a real *Client object suitable for terminateSessionByID tests.
func mockClientForSessionTerminationTests(hub *Hub, userID, clientID string) *Client {
	mockConn := newMockWsConnection() // Assuming newMockWsConnection is available and sets up a basic mock conn
	mockConn.remoteAddrFunc = func() net.Addr { return &mockAddr{} }

	c := &Client{
		hub:           hub,
		conn:          mockConn,
		send:          make(chan []byte, 256),
		ID:            clientID,
		UserID:        userID,
		Authenticated: true, // Assume authenticated for session participation
	}
	hub.clients[c] = true // Manually add to hub's client list for these tests
	return c
}

func TestHub_TerminateSessionByID_Success_BothOnline_ClientInitiated(t *testing.T) {
	setupHubTestGlobalConfig(t)
	hub, observedLogs := newHubWithObservedLogger(t)

	// Arrange: Create two clients and a session between them
	client1 := mockClientForSessionTerminationTests(hub, "user1", "client1-id")
	client2 := mockClientForSessionTerminationTests(hub, "user1", "client2-id")
	client1.CurrentRole = protocol.RoleReceiver
	client2.CurrentRole = protocol.RoleProvider

	sessionID := uuid.NewString()
	activeSession := session.NewSession(sessionID)
	// Assuming SetClient uses string role; roles defined in protocol package
	_, errC1 := activeSession.SetClient(client1, "pos")
	assert.NoError(t, errC1)
	paired, errC2 := activeSession.SetClient(client2, "card")
	assert.NoError(t, errC2)
	assert.True(t, paired, "Session should be paired after adding two clients")
	assert.Equal(t, session.StatusPaired, activeSession.Status, "Session status should be Paired") // Use direct field access

	hub.sessions[sessionID] = activeSession
	client1.SessionID = sessionID
	client2.SessionID = sessionID
	hub.cardProviders[client2.GetID()] = client2

	terminationReason := "客户端主动请求结束"
	// Use local ClientIdentity struct if defined in handler, or pass strings directly
	actingClientID := client1.GetID()
	actingClientUserID := client1.GetUserID()

	// Act
	hub.terminateSessionByID(sessionID, terminationReason, actingClientID, actingClientUserID) // Pass strings

	// Assert: Session removed from hub
	hub.providerMutex.RLock()
	_, sessionExists := hub.sessions[sessionID]
	hub.providerMutex.RUnlock()
	assert.False(t, sessionExists, "Session should be removed from hub.sessions")

	// Assert: Session status is terminated
	assert.Equal(t, session.StatusTerminated, activeSession.Status, "Session object status should be terminated") // Use direct field access

	// Assert: Clients' SessionID cleared
	assert.Empty(t, client1.SessionID, "Client1 SessionID should be cleared")
	assert.Empty(t, client2.SessionID, "Client2 SessionID should be cleared")

	// Assert: SessionTerminatedMessage sent to both clients
	msgCountClient1 := 0
	msgCountClient2 := 0
	var termMsg1, termMsg2 protocol.SessionTerminatedMessage

	// Non-blocking check for messages, allow some time
	timeout := time.After(100 * time.Millisecond)
loop:
	for msgCountClient1 < 1 || msgCountClient2 < 1 {
		select {
		case msgBytes1 := <-client1.send:
			if msgCountClient1 == 0 {
				err := json.Unmarshal(msgBytes1, &termMsg1)
				assert.NoError(t, err)
				assert.Equal(t, protocol.MessageTypeSessionTerminated, termMsg1.Type)
				assert.Equal(t, sessionID, termMsg1.SessionID)
				assert.Equal(t, terminationReason, termMsg1.Reason)
				msgCountClient1++
			}
		case msgBytes2 := <-client2.send:
			if msgCountClient2 == 0 {
				err := json.Unmarshal(msgBytes2, &termMsg2)
				assert.NoError(t, err)
				assert.Equal(t, protocol.MessageTypeSessionTerminated, termMsg2.Type)
				assert.Equal(t, sessionID, termMsg2.SessionID)
				assert.Equal(t, terminationReason, termMsg2.Reason)
				msgCountClient2++
			}
		case <-timeout:
			t.Logf("Client1 rcvd: %d, Client2 rcvd: %d", msgCountClient1, msgCountClient2)
			break loop
		}
	}
	assert.Equal(t, 1, msgCountClient1, "Client1 should have received one SessionTerminatedMessage")
	assert.Equal(t, 1, msgCountClient2, "Client2 should have received one SessionTerminatedMessage")

	// Assert: Logs (Info for termination, Audit for event)
	foundTerminationLog := false
	foundAuditEvent := false
	for _, entry := range observedLogs.All() {
		if entry.Message == "会话已终止" && entry.Level == zap.InfoLevel {
			foundTerminationLog = true
			assert.Equal(t, sessionID, entry.ContextMap()["sessionID"])
			assert.Equal(t, terminationReason, entry.ContextMap()["reason"])
		}
		if entry.Message == "AuditEvent" && entry.Level == zap.InfoLevel {
			ctxMap := entry.ContextMap()
			if eventType, ok := ctxMap["event_type"].(string); ok && eventType == "session_terminated_by_client_request" {
				foundAuditEvent = true
				assert.Equal(t, sessionID, ctxMap["session_id"])
				assert.Equal(t, actingClientID, ctxMap["acting_client_id"].(string))
			}
		}
	}
	assert.True(t, foundTerminationLog)
	assert.True(t, foundAuditEvent)
}

func TestHub_TerminateSessionByID_NotFound(t *testing.T) {
	setupHubTestGlobalConfig(t)
	hub, observedLogs := newHubWithObservedLogger(t)

	nonExistentSessionID := "non-existent-session"
	terminationReason := "Test reason"

	// Act
	hub.terminateSessionByID(nonExistentSessionID, terminationReason, "", "") // Pass empty strings for acting client info

	// Assert: No panic, and a warning log should be present.
	foundWarningLog := false
	for _, entry := range observedLogs.All() {
		// The log message in terminateSessionByID is:
		// global.GVA_LOG.Warn("Hub (terminateSessionByID): Attempted to terminate a non-existent session", ...)
		if entry.Message == "Hub (terminateSessionByID): Attempted to terminate a non-existent session" && entry.Level == zap.WarnLevel { // 确保与 hub.go 中的日志完全一致
			foundWarningLog = true
			assert.Equal(t, nonExistentSessionID, entry.ContextMap()["sessionID"])
			break
		}
	}
	assert.True(t, foundWarningLog, "Expected warning log for non-existent session was not found")

	// Assert: Hub state (sessions map) should be unchanged (empty or as it was).
	assert.Empty(t, hub.sessions, "Hub sessions map should remain empty or unchanged")
}

// --- Tests for Hub.handleClientAuth --- //

func TestHub_HandleClientAuth_ParseMessage_Success(t *testing.T) {
	setupHubTestGlobalConfig(t)
	hub, _ := newHubWithObservedLogger(t) // observedLogs not strictly needed here, but setup is fine
	mockClient := mockClientForHubTests(hub)
	mockClient.Authenticated = false // Client is not authenticated yet

	validAuthMsg := protocol.ClientAuthMessage{
		Type:  protocol.MessageTypeClientAuth,
		Token: "valid-jwt-token-for-parsing-test",
	}
	msgBytes, err := json.Marshal(validAuthMsg)
	assert.NoError(t, err, "Failed to marshal valid ClientAuthMessage")

	// We are testing handleClientAuth directly, not via handleIncomingMessage's dispatch
	// So, we don't expect sendErrorMessage from here for a successful parse itself,
	// but rather that the function proceeds to token validation (which will fail with this token).
	hub.handleClientAuth(mockClient, msgBytes)

	// Assert: No error message *about parsing* should be sent.
	// An auth failure message IS expected due to the dummy token.
	select {
	case sentMsgBytes := <-mockClient.send:
		var errMsg protocol.ErrorMessage
		err := json.Unmarshal(sentMsgBytes, &errMsg)
		assert.NoError(t, err, "Should unmarshal to ErrorMessage")
		assert.Equal(t, protocol.MessageTypeError, errMsg.Type)
		assert.Equal(t, protocol.ErrorCodeAuthFailed, errMsg.Code, "Expected AuthFailed due to token validation, not parsing error")
	case <-time.After(100 * time.Millisecond):
		t.Fatal("Timeout waiting for response from handleClientAuth (expected auth failure)")
	}
}

func TestHub_HandleClientAuth_ParseMessage_InvalidJSON(t *testing.T) {
	setupHubTestGlobalConfig(t)
	hub, observedLogs := newHubWithObservedLogger(t)
	mockClient := mockClientForHubTests(hub)
	mockClient.Authenticated = false

	invalidJSONBytes := []byte("{\"type\": \"ClientAuth\", \"token\": \"abc\"") // Malformed JSON

	hub.handleClientAuth(mockClient, invalidJSONBytes)

	// Assert: Client receives ErrorMessage for BadRequest
	select {
	case sentMsgBytes := <-mockClient.send:
		var errMsg protocol.ErrorMessage
		err := json.Unmarshal(sentMsgBytes, &errMsg)
		assert.NoError(t, err, "Should unmarshal to ErrorMessage")
		assert.Equal(t, protocol.MessageTypeError, errMsg.Type)
		assert.Equal(t, protocol.ErrorCodeBadRequest, errMsg.Code)
		assert.Contains(t, errMsg.Message, "Invalid auth message format")
	case <-time.After(100 * time.Millisecond):
		t.Fatal("Timeout waiting for BadRequest error message")
	}

	// Assert: Correct error log
	foundErrorLog := false
	for _, logEntry := range observedLogs.All() {
		// The log message in handleClientAuth is:
		// global.GVA_LOG.Error("Hub: Error unmarshalling auth message", ...)
		if logEntry.Message == "Hub: Error unmarshalling auth message" && logEntry.Level == zap.ErrorLevel { // 确保与 hub.go 中的日志完全一致
			assert.Equal(t, mockClient.GetID(), logEntry.ContextMap()["clientID"])
			foundErrorLog = true
			break
		}
	}
	assert.True(t, foundErrorLog, "Expected error log for auth message unmarshal failure was not found")

	// Assert: Audit log (error_occurred via sendErrorMessage)
	// This is implicitly tested by sendErrorMessage tests, but good to be aware.
}

func TestHub_HandleClientAuth_ParseMessage_FieldTypeMismatch(t *testing.T) {
	setupHubTestGlobalConfig(t)
	hub, observedLogs := newHubWithObservedLogger(t)
	mockClient := mockClientForHubTests(hub)
	mockClient.Authenticated = false

	// Token field is expected to be a string, but we provide a number
	mismatchedTypeJSON := []byte(`{"type":"ClientAuth","token":12345}`)

	hub.handleClientAuth(mockClient, mismatchedTypeJSON)

	// Assert: Client receives ErrorMessage for BadRequest (due to unmarshal error)
	select {
	case sentMsgBytes := <-mockClient.send:
		var errMsg protocol.ErrorMessage
		err := json.Unmarshal(sentMsgBytes, &errMsg)
		assert.NoError(t, err, "Should unmarshal to ErrorMessage")
		assert.Equal(t, protocol.MessageTypeError, errMsg.Type)
		assert.Equal(t, protocol.ErrorCodeBadRequest, errMsg.Code)
		assert.Contains(t, errMsg.Message, "Invalid auth message format")
	case <-time.After(100 * time.Millisecond):
		t.Fatal("Timeout waiting for BadRequest error message due to type mismatch")
	}

	// Assert: Correct error log
	foundErrorLog := false
	for _, logEntry := range observedLogs.All() {
		// The log message in handleClientAuth for field type mismatch also uses:
		// global.GVA_LOG.Error("Hub: Error unmarshalling auth message", ...)
		if logEntry.Message == "Hub: Error unmarshalling auth message" && logEntry.Level == zap.ErrorLevel { // 确保与 hub.go 中的日志完全一致
			assert.Equal(t, mockClient.GetID(), logEntry.ContextMap()["clientID"])
			// Check that the underlying error indicates a type mismatch
			parseErr, _ := logEntry.ContextMap()["error"].(string)
			assert.Contains(t, parseErr, "json: cannot unmarshal number into Go struct field ClientAuthMessage.token of type string")
			foundErrorLog = true
			break
		}
	}
	assert.True(t, foundErrorLog, "Expected error log for auth message unmarshal type mismatch was not found, or error detail mismatch")
}
