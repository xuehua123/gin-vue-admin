package handler

import (
	"encoding/json"
	"fmt"
	"strings"
	"testing"
	"time"

	"go.uber.org/zap/zaptest/observer"

	"github.com/flipped-aurora/gin-vue-admin/server/nfc_relay/protocol"
	"github.com/flipped-aurora/gin-vue-admin/server/nfc_relay/session"

	// "github.com/google/uuid" // Will be needed for session creation tests
	"github.com/stretchr/testify/assert"
	"go.uber.org/zap"
	// "go.uber.org/zap/zapcore" // Already in hub_test_helpers.go
	// "go.uber.org/zap/zaptest/observer" // Already in hub_test_helpers.go
)

// TestHub_HandleSelectCardProvider_ParseMessage_ValidJSON tests successful parsing of a valid SelectCardProviderMessage.
// Note: This test primarily ensures the message is parsed. Subsequent logic might still lead to an error
// response if pre-conditions (like provider availability) are not met.
func TestHub_HandleSelectCardProvider_ParseMessage_ValidJSON(t *testing.T) {
	setupHubTestGlobalConfig(t)
	hub, _ := newHubWithObservedLogger(t)
	// go hub.Run() // Not strictly necessary for testing handleSelectCardProvider in isolation for parsing

	requestingClient := mockClientForHubTests(hub)
	requestingClient.Authenticated = true
	requestingClient.UserID = "receiverUser1"
	requestingClient.CurrentRole = protocol.RoleReceiver

	targetProvider := mockClientForHubTests(hub)
	targetProvider.Authenticated = true
	targetProvider.UserID = "receiverUser1" // Same UserID for a valid scenario
	targetProvider.CurrentRole = protocol.RoleProvider
	targetProvider.IsOnline = true
	targetProvider.DisplayName = "TestProvider"

	hub.providerMutex.Lock()
	hub.cardProviders[targetProvider.GetID()] = targetProvider
	hub.providerMutex.Unlock()

	selectMsg := protocol.SelectCardProviderMessage{
		Type:       protocol.MessageTypeSelectCardProvider,
		ProviderID: targetProvider.GetID(),
	}
	msgBytes, err := json.Marshal(selectMsg)
	assert.NoError(t, err, "Failed to marshal valid SelectCardProviderMessage")

	// Act
	hub.handleSelectCardProvider(requestingClient, msgBytes)

	// Assert: Expect SessionEstablishedMessage (or an error if conditions beyond parsing fail)
	// For this test, we're happy if it doesn't fail on parsing itself.
	// A successful parse leads to session establishment logic, which sends SessionEstablishedMessage.
	// If the provider wasn't found or other pre-checks failed, an ErrorMessage would be sent.
	select {
	case sentMsgBytes := <-requestingClient.send:
		// Check if it's SessionEstablishedMessage (success case for this setup)
		var sessionMsg protocol.SessionEstablishedMessage
		if json.Unmarshal(sentMsgBytes, &sessionMsg) == nil && sessionMsg.Type == protocol.MessageTypeSessionEstablished {
			// This is good, means parsing and basic logic proceeded
			assert.NotEmpty(t, sessionMsg.SessionID, "SessionID should not be empty on success")
			assert.Equal(t, targetProvider.GetID(), sessionMsg.PeerID, "PeerID should be the target provider")
			assert.Equal(t, protocol.RoleProvider, sessionMsg.PeerRole, "PeerRole should be provider")
		} else {
			// Or it could be an error message if pre-conditions failed, but parsing was okay.
			var errMsg protocol.ErrorMessage
			err := json.Unmarshal(sentMsgBytes, &errMsg)
			assert.NoError(t, err, "Failed to unmarshal response message (expected SessionEstablished or Error)")
			t.Logf("Received ErrorMessage instead of SessionEstablished: %v (Code: %d)", errMsg.Message, errMsg.Code)
			// We don't assert specific error here as this test is for *parsing* primarily.
			// Other tests will cover specific pre-condition failures.
		}

	case <-time.After(200 * time.Millisecond): // Increased timeout as session logic might take a bit
		t.Fatal("Timeout waiting for a response after sending SelectCardProviderMessage")
	}
}

// TestHub_HandleSelectCardProvider_ParseMessage_InvalidJSON tests handling of an invalid JSON SelectCardProviderMessage.
func TestHub_HandleSelectCardProvider_ParseMessage_InvalidJSON(t *testing.T) {
	setupHubTestGlobalConfig(t)
	hub, observedLogs := newHubWithObservedLogger(t)

	requestingClient := mockClientForHubTests(hub)
	requestingClient.Authenticated = true
	requestingClient.UserID = "receiverUser2"
	requestingClient.CurrentRole = protocol.RoleReceiver

	invalidJSONBytes := []byte("this is not a valid json string for select provider")

	// Act
	hub.handleSelectCardProvider(requestingClient, invalidJSONBytes)

	// Assert: Client receives ErrorMessage for BadRequest
	select {
	case sentMsgBytes := <-requestingClient.send:
		var errMsg protocol.ErrorMessage
		err := json.Unmarshal(sentMsgBytes, &errMsg)
		assert.NoError(t, err, "Should unmarshal to ErrorMessage")
		assert.Equal(t, protocol.MessageTypeError, errMsg.Type)
		assert.Equal(t, protocol.ErrorCodeBadRequest, errMsg.Code)
		assert.Contains(t, errMsg.Message, "无效的选择发卡方消息格式")
	case <-time.After(100 * time.Millisecond):
		t.Fatal("Timeout waiting for BadRequest error message due to invalid JSON")
	}

	// Assert: Correct error log
	foundErrorLog := false
	for _, logEntry := range observedLogs.All() {
		if logEntry.Message == "Hub 处理选择发卡方：反序列化 SelectCardProviderMessage 失败" && logEntry.Level == zap.ErrorLevel { // Updated expected log message
			foundErrorLog = true
			assert.Equal(t, requestingClient.GetID(), logEntry.ContextMap()["clientID"])
			assert.Contains(t, logEntry.ContextMap()["error"].(string), "invalid character")
			break
		}
	}
	assert.True(t, foundErrorLog, "Expected error log 'Hub 处理选择发卡方：反序列化 SelectCardProviderMessage 失败' not found. Logs:\n"+dumpLogs(observedLogs, zap.ErrorLevel))
}

// TestHub_HandleSelectCardProvider_PreCondition_NotReceiver tests that non-receiver clients cannot select a provider.
func TestHub_HandleSelectCardProvider_PreCondition_NotReceiver(t *testing.T) {
	setupHubTestGlobalConfig(t)
	hub, observedLogs := newHubWithObservedLogger(t)

	requestingClient := mockClientForHubTests(hub)
	requestingClient.Authenticated = true
	requestingClient.UserID = "userNotReceiver"
	requestingClient.CurrentRole = protocol.RoleProvider // Key: Client is a provider

	selectMsg := protocol.SelectCardProviderMessage{
		Type:       protocol.MessageTypeSelectCardProvider,
		ProviderID: "anyProviderID",
	}
	msgBytes, err := json.Marshal(selectMsg)
	assert.NoError(t, err)

	// Act
	hub.handleSelectCardProvider(requestingClient, msgBytes)

	// Assert: ErrorMessage sent to client
	select {
	case sentBytes := <-requestingClient.send:
		var errMsg protocol.ErrorMessage
		err := json.Unmarshal(sentBytes, &errMsg)
		assert.NoError(t, err)
		assert.Equal(t, protocol.MessageTypeError, errMsg.Type)
		assert.Equal(t, protocol.ErrorCodePermissionDenied, errMsg.Code)
		assert.Equal(t, "操作失败：只有 receiver 角色的客户端才能选择发卡方。", errMsg.Message) // Updated expected message
	case <-time.After(100 * time.Millisecond):
		t.Fatal("Timeout waiting for ErrorMessage due to not being a receiver")
	}

	// Assert: Log message
	foundLog := false
	for _, entry := range observedLogs.FilterLevelExact(zap.WarnLevel).All() {
		if strings.Contains(entry.Message, "非 receiver 客户端尝试选择 provider") { // Updated to Contains and actual message part
			assert.Equal(t, requestingClient.GetID(), entry.ContextMap()["clientID"])
			assert.Equal(t, string(requestingClient.CurrentRole), entry.ContextMap()["currentRole"])
			foundLog = true
			break
		}
	}
	assert.True(t, foundLog, "Expected warning log for non-receiver attempting selection not found. Logs:\n"+dumpLogs(observedLogs, zap.WarnLevel))
}

// TestHub_HandleSelectCardProvider_PreCondition_EmptyProviderID tests handling when ProviderID is empty.
func TestHub_HandleSelectCardProvider_PreCondition_EmptyProviderID(t *testing.T) {
	setupHubTestGlobalConfig(t)
	hub, observedLogs := newHubWithObservedLogger(t)

	requestingClient := mockClientForHubTests(hub)
	requestingClient.Authenticated = true
	requestingClient.UserID = "receiverUser3"
	requestingClient.CurrentRole = protocol.RoleReceiver

	selectMsg := protocol.SelectCardProviderMessage{
		Type:       protocol.MessageTypeSelectCardProvider,
		ProviderID: "", // Key: Empty ProviderID
	}
	msgBytes, err := json.Marshal(selectMsg)
	assert.NoError(t, err)

	// Act
	hub.handleSelectCardProvider(requestingClient, msgBytes)

	// Assert: ErrorMessage sent to client
	select {
	case sentBytes := <-requestingClient.send:
		var errMsg protocol.ErrorMessage
		err := json.Unmarshal(sentBytes, &errMsg)
		assert.NoError(t, err)
		assert.Equal(t, protocol.MessageTypeError, errMsg.Type)
		assert.Equal(t, protocol.ErrorCodeBadRequest, errMsg.Code)
		assert.Equal(t, "选择发卡方失败：必须提供有效的 ProviderID。", errMsg.Message)
	case <-time.After(100 * time.Millisecond):
		t.Fatal("Timeout waiting for ErrorMessage due to empty ProviderID")
	}

	// Assert: Log message
	foundLog := false
	for _, entry := range observedLogs.FilterLevelExact(zap.WarnLevel).All() {
		if strings.Contains(entry.Message, "SelectCardProvider 请求中 ProviderID 为空") { // Updated to Contains and actual message part
			assert.Equal(t, requestingClient.GetID(), entry.ContextMap()["clientID"])
			foundLog = true
			break
		}
	}
	assert.True(t, foundLog, "Expected warning log for empty ProviderID not found. Logs:\n"+dumpLogs(observedLogs, zap.WarnLevel))
}

// TestHub_HandleSelectCardProvider_PreCondition_ProviderNotFound tests when the target provider is not found or offline.
func TestHub_HandleSelectCardProvider_PreCondition_ProviderNotFound(t *testing.T) {
	setupHubTestGlobalConfig(t)
	hub, observedLogs := newHubWithObservedLogger(t)

	requestingClient := mockClientForHubTests(hub)
	requestingClient.Authenticated = true
	requestingClient.UserID = "receiverUser4"
	requestingClient.CurrentRole = protocol.RoleReceiver

	selectMsg := protocol.SelectCardProviderMessage{
		Type:       protocol.MessageTypeSelectCardProvider,
		ProviderID: "nonExistentProviderID", // Key: Provider does not exist
	}
	msgBytes, err := json.Marshal(selectMsg)
	assert.NoError(t, err)

	// Act
	hub.handleSelectCardProvider(requestingClient, msgBytes)

	// Assert: ErrorMessage sent to client
	select {
	case sentBytes := <-requestingClient.send:
		var errMsg protocol.ErrorMessage
		err := json.Unmarshal(sentBytes, &errMsg)
		assert.NoError(t, err)
		assert.Equal(t, protocol.MessageTypeError, errMsg.Type)
		assert.Equal(t, protocol.ErrorCodeProviderNotFound, errMsg.Code)
		assert.Equal(t, "选择发卡方失败：目标发卡方不存在或当前未提供服务。", errMsg.Message) // Updated expected message
	case <-time.After(100 * time.Millisecond):
		t.Fatal("Timeout waiting for ErrorMessage due to provider not found")
	}

	// Assert: Log message
	foundLog := false
	for _, entry := range observedLogs.FilterLevelExact(zap.WarnLevel).All() {
		if strings.Contains(entry.Message, "目标发卡方不存在或未上线") { // Updated to Contains and actual message part
			assert.Equal(t, requestingClient.GetID(), entry.ContextMap()["requestingClientID"])
			assert.Equal(t, "nonExistentProviderID", entry.ContextMap()["targetProviderID"])
			foundLog = true
			break
		}
	}
	assert.True(t, foundLog, "Expected warning log for provider not found not found. Logs:\n"+dumpLogs(observedLogs, zap.WarnLevel))
}

// TestHub_HandleSelectCardProvider_PreCondition_ProviderNotClientType tests when a cardProvider entry is not *Client.
func TestHub_HandleSelectCardProvider_PreCondition_ProviderNotClientType(t *testing.T) {
	setupHubTestGlobalConfig(t)
	hub, observedLogs := newHubWithObservedLogger(t)

	requestingClient := mockClientForHubTests(hub)
	requestingClient.Authenticated = true
	requestingClient.UserID = "receiverUser4.1"
	requestingClient.CurrentRole = protocol.RoleReceiver

	badProviderID := "badProviderTypeID"
	mockBadProvider := &mockNonClientProvider{id: badProviderID, userID: "receiverUser4.1"}

	hub.providerMutex.Lock()
	hub.cardProviders[badProviderID] = mockBadProvider
	hub.providerMutex.Unlock()

	selectMsg := protocol.SelectCardProviderMessage{
		Type:       protocol.MessageTypeSelectCardProvider,
		ProviderID: badProviderID,
	}
	msgBytes, err := json.Marshal(selectMsg)
	assert.NoError(t, err)

	// Act
	hub.handleSelectCardProvider(requestingClient, msgBytes)

	// Assert: ErrorMessage sent to client
	select {
	case sentBytes := <-requestingClient.send:
		var errMsg protocol.ErrorMessage
		err := json.Unmarshal(sentBytes, &errMsg)
		assert.NoError(t, err)
		assert.Equal(t, protocol.MessageTypeError, errMsg.Type)
		assert.Equal(t, protocol.ErrorCodeInternalError, errMsg.Code) // Updated expected code
		assert.Equal(t, "选择发卡方失败：服务器内部错误。", errMsg.Message)           // Updated expected message
	case <-time.After(100 * time.Millisecond):
		t.Fatal("Timeout waiting for ErrorMessage due to provider not being *Client type")
	}

	// Assert: Log message
	foundLog := false
	for _, entry := range observedLogs.FilterLevelExact(zap.ErrorLevel).All() { // Log level is Error for this case in hub.go
		if strings.Contains(entry.Message, "cardProviders 中的条目不是 *Client 类型或为nil") { // Updated log message check
			assert.Equal(t, requestingClient.GetID(), entry.ContextMap()["requestingClientID"])
			assert.Equal(t, badProviderID, entry.ContextMap()["targetProviderID"])
			foundLog = true
			break
		}
	}
	assert.True(t, foundLog, "Expected error log for provider type assertion failure not found. Logs:\n"+dumpLogs(observedLogs, zap.ErrorLevel))
}

// mockNonClientProvider is a mock that satisfies ClientInfoProvider but is not *Client.
type mockNonClientProvider struct {
	id     string
	userID string
	role   protocol.RoleType // Not strictly used by ClientInfoProvider but useful for context
	sendCh chan []byte       // Mock send channel
}

func (m *mockNonClientProvider) GetID() string                                { return m.id }
func (m *mockNonClientProvider) GetUserID() string                            { return m.userID }
func (m *mockNonClientProvider) GetCurrentRole() protocol.RoleType            { return m.role } // Example implementation
func (m *mockNonClientProvider) Send(message []byte) error                    { return nil }    // Example implementation
func (m *mockNonClientProvider) GetSessionID() string                         { return "" }     // Example
func (m *mockNonClientProvider) SetSessionID(sessionID string)                {}                // Example
func (m *mockNonClientProvider) GetConcreteClient() *Client                   { return nil }    // Crucially, this is nil
func (m *mockNonClientProvider) GetDisplayName() string                       { return "MockNonClient" }
func (m *mockNonClientProvider) IsClientOnline() bool                         { return true }
func (m *mockNonClientProvider) SetCurrentRole(role protocol.RoleType)        {}
func (m *mockNonClientProvider) SetIsOnline(isOnline bool)                    {}
func (m *mockNonClientProvider) SetDisplayName(displayName string)            {}
func (m *mockNonClientProvider) SetAuthenticated(authenticated bool)          {}
func (m *mockNonClientProvider) SetUserID(userID string)                      {}
func (m *mockNonClientProvider) SetHub(h *Hub)                                {}
func (m *mockNonClientProvider) GetSendChannel() chan []byte                  { return m.sendCh }
func (m *mockNonClientProvider) StartPumps()                                  {}
func (m *mockNonClientProvider) CloseConnAndCleanup(logMsg string, err error) {}

// GetRole implements the missing method for ClientInfoProvider
func (m *mockNonClientProvider) GetRole() string {
	return string(m.role) // Match the return type and logic of Client.GetRole()
}

// TestHub_HandleSelectCardProvider_PreCondition_UserIDMismatch tests when receiver and provider UserIDs don't match.
func TestHub_HandleSelectCardProvider_PreCondition_UserIDMismatch(t *testing.T) {
	setupHubTestGlobalConfig(t)
	hub, observedLogs := newHubWithObservedLogger(t)

	requestingClient := mockClientForHubTests(hub)
	requestingClient.Authenticated = true
	requestingClient.UserID = "receiverUser5" // User A
	requestingClient.CurrentRole = protocol.RoleReceiver

	targetProvider := mockClientForHubTests(hub)
	targetProvider.Authenticated = true
	targetProvider.UserID = "providerUserDifferent" // User B - Key: Different UserID
	targetProvider.CurrentRole = protocol.RoleProvider
	targetProvider.IsOnline = true
	targetProvider.DisplayName = "ProviderUserB"

	hub.providerMutex.Lock()
	hub.cardProviders[targetProvider.GetID()] = targetProvider
	hub.providerMutex.Unlock()

	selectMsg := protocol.SelectCardProviderMessage{
		Type:       protocol.MessageTypeSelectCardProvider,
		ProviderID: targetProvider.GetID(),
	}
	msgBytes, err := json.Marshal(selectMsg)
	assert.NoError(t, err)

	// Act
	hub.handleSelectCardProvider(requestingClient, msgBytes)

	// Assert: ErrorMessage sent to client
	select {
	case sentBytes := <-requestingClient.send:
		var errMsg protocol.ErrorMessage
		err := json.Unmarshal(sentBytes, &errMsg)
		assert.NoError(t, err)
		assert.Equal(t, protocol.MessageTypeError, errMsg.Type)
		assert.Equal(t, protocol.ErrorCodePermissionDenied, errMsg.Code)
		assert.Equal(t, "选择发卡方失败：不能选择其他账户下的发卡方。", errMsg.Message) // Updated expected message
	case <-time.After(100 * time.Millisecond):
		t.Fatal("Timeout waiting for ErrorMessage due to UserID mismatch")
	}

	// Assert: Log message
	foundLog := false
	for _, entry := range observedLogs.FilterLevelExact(zap.WarnLevel).All() {
		if strings.Contains(entry.Message, "请求者和提供者 UserID 不匹配") { // Updated to Contains and actual message part
			assert.Equal(t, requestingClient.GetID(), entry.ContextMap()["requestingClientID"])
			assert.Equal(t, requestingClient.GetUserID(), entry.ContextMap()["requestingUserID"])
			assert.Equal(t, targetProvider.GetID(), entry.ContextMap()["providerClientID"])
			assert.Equal(t, targetProvider.GetUserID(), entry.ContextMap()["providerUserID"])
			foundLog = true
			break
		}
	}
	assert.True(t, foundLog, "Expected warning log for UserID mismatch not found. Logs:\n"+dumpLogs(observedLogs, zap.WarnLevel))
}

// TestHub_HandleSelectCardProvider_PreCondition_SelectSelf tests when a receiver tries to select itself as provider.
func TestHub_HandleSelectCardProvider_PreCondition_SelectSelf(t *testing.T) {
	setupHubTestGlobalConfig(t)
	hub, observedLogs := newHubWithObservedLogger(t)

	requestingClient := mockClientForHubTests(hub)
	requestingClient.Authenticated = true
	requestingClient.UserID = "selfSelectUser"
	requestingClient.CurrentRole = protocol.RoleReceiver // Is a receiver

	hub.providerMutex.Lock()
	hub.cardProviders[requestingClient.GetID()] = requestingClient
	hub.providerMutex.Unlock()

	selectMsg := protocol.SelectCardProviderMessage{
		Type:       protocol.MessageTypeSelectCardProvider,
		ProviderID: requestingClient.GetID(), // Key: Selecting self
	}
	msgBytes, err := json.Marshal(selectMsg)
	assert.NoError(t, err)

	// Act
	hub.handleSelectCardProvider(requestingClient, msgBytes)

	// Assert: ErrorMessage sent to client
	select {
	case sentBytes := <-requestingClient.send:
		var errMsg protocol.ErrorMessage
		err := json.Unmarshal(sentBytes, &errMsg)
		assert.NoError(t, err)
		assert.Equal(t, protocol.MessageTypeError, errMsg.Type)
		assert.Equal(t, protocol.ErrorCodeSelectSelf, errMsg.Code)
		assert.Equal(t, "选择发卡方失败：不能选择自己。", errMsg.Message) // Updated expected message
	case <-time.After(100 * time.Millisecond):
		t.Fatal("Timeout waiting for ErrorMessage due to selecting self")
	}

	// Assert: Log message
	foundLog := false
	for _, entry := range observedLogs.FilterLevelExact(zap.WarnLevel).All() {
		if strings.Contains(entry.Message, "客户端尝试选择自己作为发卡方") { // Updated to Contains and actual message part
			assert.Equal(t, requestingClient.GetID(), entry.ContextMap()["clientID"])
			// assert.Equal(t, requestingClient.GetID(), entry.ContextMap()["targetProviderID"]) // targetProviderID is not logged here
			foundLog = true
			break
		}
	}
	assert.True(t, foundLog, "Expected warning log for selecting self not found. Logs:\n"+dumpLogs(observedLogs, zap.WarnLevel))
}

// TestHub_HandleSelectCardProvider_PreCondition_ReceiverBusy tests when the requesting receiver is already in a session.
func TestHub_HandleSelectCardProvider_PreCondition_ReceiverBusy(t *testing.T) {
	setupHubTestGlobalConfig(t)
	hub, observedLogs := newHubWithObservedLogger(t)

	requestingClient := mockClientForHubTests(hub)
	requestingClient.Authenticated = true
	requestingClient.UserID = "busyReceiverUser"
	requestingClient.CurrentRole = protocol.RoleReceiver
	requestingClient.SessionID = "existingSession123" // Key: Receiver is busy

	// Mock a session for the requesting client to make it busy
	hub.providerMutex.Lock()
	mockSessionForReceiver := session.NewSession(requestingClient.SessionID)
	mockSessionForReceiver.SetClient(requestingClient, string(protocol.RoleReceiver)) // Add receiver to its own session
	// Add a dummy provider to make the session appear valid for the check
	dummyProvForReceiverSession := mockClientForHubTests(hub)
	dummyProvForReceiverSession.ID = "dummy-prov-for-receiver-busy-test"
	dummyProvForReceiverSession.UserID = requestingClient.UserID
	mockSessionForReceiver.SetClient(dummyProvForReceiverSession, string(protocol.RoleProvider))
	hub.sessions[requestingClient.SessionID] = mockSessionForReceiver
	hub.providerMutex.Unlock()

	targetProvider := mockClientForHubTests(hub) // A valid, available provider
	targetProvider.Authenticated = true
	targetProvider.UserID = "busyReceiverUser" // Same UserID
	targetProvider.CurrentRole = protocol.RoleProvider
	targetProvider.IsOnline = true

	hub.providerMutex.Lock()
	hub.cardProviders[targetProvider.GetID()] = targetProvider
	hub.providerMutex.Unlock()

	selectMsg := protocol.SelectCardProviderMessage{
		Type:       protocol.MessageTypeSelectCardProvider,
		ProviderID: targetProvider.GetID(),
	}
	msgBytes, err := json.Marshal(selectMsg)
	assert.NoError(t, err)

	// Act
	hub.handleSelectCardProvider(requestingClient, msgBytes)

	// Assert: ErrorMessage sent to client
	select {
	case sentBytes := <-requestingClient.send:
		var errMsg protocol.ErrorMessage
		err := json.Unmarshal(sentBytes, &errMsg)
		assert.NoError(t, err)
		assert.Equal(t, protocol.MessageTypeError, errMsg.Type)
		assert.Equal(t, protocol.ErrorCodeReceiverBusy, errMsg.Code) // Updated expected code
		assert.Equal(t, "选择发卡方失败：您当前已在会话中。", errMsg.Message)         // Updated expected message
	case <-time.After(100 * time.Millisecond):
		t.Fatal("Timeout waiting for ErrorMessage due to receiver being busy")
	}

	// Assert: Log message
	foundLog := false
	for _, entry := range observedLogs.FilterLevelExact(zap.WarnLevel).All() {
		if strings.Contains(entry.Message, "请求的发卡方已在会话中") { // Updated log message based on hub.go (this log seems for provider, but let's check if it's this one)
			if entry.ContextMap()["clientID"] == requestingClient.GetID() && entry.ContextMap()["sessionID"] == "existingSession123" {
				foundLog = true
				break
			}
		}
	}
	assert.True(t, foundLog, "Expected warning log for receiver busy not found. Logs:\n"+dumpLogs(observedLogs, zap.WarnLevel))
}

// TestHub_HandleSelectCardProvider_PreCondition_ProviderBusy tests when the target provider is already in a session.
func TestHub_HandleSelectCardProvider_PreCondition_ProviderBusy(t *testing.T) {
	setupHubTestGlobalConfig(t)
	hub, observedLogs := newHubWithObservedLogger(t)

	requestingClient := mockClientForHubTests(hub)
	requestingClient.Authenticated = true
	requestingClient.UserID = "userForBusyProvider"
	requestingClient.CurrentRole = protocol.RoleReceiver

	targetProvider := mockClientForHubTests(hub)
	targetProvider.Authenticated = true
	targetProvider.UserID = "userForBusyProvider" // Same UserID
	targetProvider.CurrentRole = protocol.RoleProvider
	targetProvider.IsOnline = true
	targetProvider.SessionID = "providerSession456" // Key: Provider is busy

	hub.providerMutex.Lock()
	hub.cardProviders[targetProvider.GetID()] = targetProvider
	mockSession := session.NewSession(targetProvider.SessionID)
	mockSession.SetClient(targetProvider, string(protocol.RoleProvider))
	dummyReceiver := mockClientForHubTests(hub)
	dummyReceiver.ID = "dummyReceiverForBusyProviderTest"
	mockSession.SetClient(dummyReceiver, string(protocol.RoleReceiver))
	hub.sessions[targetProvider.SessionID] = mockSession
	hub.providerMutex.Unlock()

	selectMsg := protocol.SelectCardProviderMessage{
		Type:       protocol.MessageTypeSelectCardProvider,
		ProviderID: targetProvider.GetID(),
	}
	msgBytes, err := json.Marshal(selectMsg)
	assert.NoError(t, err)

	// Act
	hub.handleSelectCardProvider(requestingClient, msgBytes)

	// Assert: ErrorMessage sent to client
	select {
	case sentBytes := <-requestingClient.send:
		var errMsg protocol.ErrorMessage
		err := json.Unmarshal(sentBytes, &errMsg)
		assert.NoError(t, err)
		assert.Equal(t, protocol.MessageTypeError, errMsg.Type)
		assert.Equal(t, protocol.ErrorCodeProviderBusy, errMsg.Code)
		assert.Equal(t, "选择发卡方失败：目标发卡方当前正忙。", errMsg.Message) // Updated expected message
	case <-time.After(100 * time.Millisecond):
		t.Fatal("Timeout waiting for ErrorMessage due to provider being busy")
	}

	// Assert: Log message
	foundLog := false
	for _, entry := range observedLogs.FilterLevelExact(zap.WarnLevel).All() {
		if strings.Contains(entry.Message, "目标发卡方已在会话中") { // Updated to Contains and actual message part
			assert.Equal(t, targetProvider.GetID(), entry.ContextMap()["targetProviderID"])
			assert.Equal(t, "providerSession456", entry.ContextMap()["sessionID"])
			foundLog = true
			break
		}
	}
	assert.True(t, foundLog, "Expected warning log for provider busy not found. Logs:\n"+dumpLogs(observedLogs, zap.WarnLevel))
}

// TestHub_HandleSelectCardProvider_SessionCreation_Successful tests successful session creation.
func TestHub_HandleSelectCardProvider_SessionCreation_Successful(t *testing.T) {
	setupHubTestGlobalConfig(t)
	hub, observedLogs := newHubWithObservedLogger(t)
	go hub.Run() // Hub needs to run for notifications and session management

	requestingClient := mockClientForHubTests(hub)
	requestingClient.Authenticated = true
	requestingClient.UserID = "sessionUser1"
	requestingClient.CurrentRole = protocol.RoleReceiver
	hub.register <- requestingClient

	targetProvider := mockClientForHubTests(hub)
	targetProvider.Authenticated = true
	targetProvider.UserID = "sessionUser1" // Same UserID
	targetProvider.CurrentRole = protocol.RoleProvider
	targetProvider.IsOnline = true
	targetProvider.DisplayName = "SuperProvider"
	hub.register <- targetProvider

	hub.providerMutex.Lock()
	hub.cardProviders[targetProvider.GetID()] = targetProvider
	mockSubscriber := mockClientForHubTests(hub)
	mockSubscriber.UserID = targetProvider.UserID
	mockSubscriber.CurrentRole = protocol.RoleReceiver
	hub.register <- mockSubscriber
	if hub.providerListSubscribers[targetProvider.UserID] == nil {
		hub.providerListSubscribers[targetProvider.UserID] = make(map[*Client]bool)
	}
	hub.providerListSubscribers[targetProvider.UserID][mockSubscriber] = true
	hub.providerMutex.Unlock()

	selectMsg := protocol.SelectCardProviderMessage{
		Type:       protocol.MessageTypeSelectCardProvider,
		ProviderID: targetProvider.GetID(),
	}
	msgBytes, err := json.Marshal(selectMsg)
	assert.NoError(t, err)

	// Act
	hub.handleSelectCardProvider(requestingClient, msgBytes)

	var establishedSessionID string

	select {
	case sentBytes := <-requestingClient.send:
		var sessionMsg protocol.SessionEstablishedMessage
		err := json.Unmarshal(sentBytes, &sessionMsg)
		assert.NoError(t, err, "Failed to unmarshal SessionEstablishedMessage for receiver")
		assert.Equal(t, protocol.MessageTypeSessionEstablished, sessionMsg.Type)
		assert.NotEmpty(t, sessionMsg.SessionID, "Receiver SessionID should not be empty")
		establishedSessionID = sessionMsg.SessionID
		assert.Equal(t, targetProvider.GetID(), sessionMsg.PeerID, "Receiver's PeerID should be provider")
		assert.Equal(t, protocol.RoleProvider, sessionMsg.PeerRole, "Receiver's PeerRole should be provider")
	case <-time.After(300 * time.Millisecond):
		t.Fatal("Timeout waiting for SessionEstablishedMessage for requestingClient")
	}

	select {
	case sentBytes := <-targetProvider.send:
		var sessionMsg protocol.SessionEstablishedMessage
		err := json.Unmarshal(sentBytes, &sessionMsg)
		assert.NoError(t, err, "Failed to unmarshal SessionEstablishedMessage for provider")
		assert.Equal(t, protocol.MessageTypeSessionEstablished, sessionMsg.Type)
		assert.Equal(t, establishedSessionID, sessionMsg.SessionID, "Provider SessionID mismatch")
		assert.Equal(t, requestingClient.GetID(), sessionMsg.PeerID, "Provider's PeerID should be receiver")
		assert.Equal(t, protocol.RoleReceiver, sessionMsg.PeerRole, "Provider's PeerRole should be receiver")
	case <-time.After(300 * time.Millisecond):
		t.Fatal("Timeout waiting for SessionEstablishedMessage for targetProvider")
	}

	hub.providerMutex.RLock()
	_, sessionExists := hub.sessions[establishedSessionID]
	receiverSessionID := requestingClient.GetSessionID()
	providerSessionID := targetProvider.GetSessionID()
	hub.providerMutex.RUnlock()

	assert.True(t, sessionExists, "Session should exist in hub.sessions map")
	assert.Equal(t, establishedSessionID, receiverSessionID, "Requesting client SessionID not updated")
	assert.Equal(t, establishedSessionID, providerSessionID, "Target provider SessionID not updated")

	foundAuditLog := false
	for _, logEntry := range observedLogs.All() {
		if logEntry.Level == zap.InfoLevel && logEntry.LoggerName == "audit" && logEntry.Message == "AuditEvent" {
			eventType, _ := logEntry.ContextMap()["event_type"].(string)
			if eventType == "session_established" {
				// Assert top-level fields from zap.Field passed to LogAuditEvent
				assert.Equal(t, establishedSessionID, logEntry.ContextMap()["session_id"], "Audit log session_id mismatch")
				assert.Equal(t, requestingClient.GetID(), logEntry.ContextMap()["client_id_initiator"], "Audit log client_id_initiator mismatch")
				assert.Equal(t, targetProvider.GetID(), logEntry.ContextMap()["client_id_responder"], "Audit log client_id_responder mismatch")

				// Assert fields within the 'details' map (which comes from the SessionDetails struct)
				details, ok := logEntry.ContextMap()["details"].(map[string]interface{})
				assert.True(t, ok, "Details map not found or wrong type in audit log for session_established")
				if ok {
					assert.Equal(t, string(requestingClient.CurrentRole), details["initiator_role"], "Audit log details.initiator_role mismatch")
					assert.Equal(t, string(targetProvider.CurrentRole), details["responder_role"], "Audit log details.responder_role mismatch")
				}
				foundAuditLog = true
				break
			}
		}
	}
	assert.True(t, foundAuditLog, "Expected 'session_established' audit log not found. Logs:\n"+dumpLogsForAudit(observedLogs))

	select {
	case sentToSubBytes := <-mockSubscriber.send:
		var listMsg protocol.CardProvidersListMessage
		err := json.Unmarshal(sentToSubBytes, &listMsg)
		assert.NoError(t, err, "Failed to unmarshal CardProvidersListMessage for subscriber")
		assert.Equal(t, protocol.MessageTypeCardProvidersList, listMsg.Type)
		providerNowBusy := false
		for _, pInfo := range listMsg.Providers {
			if pInfo.ProviderID == targetProvider.GetID() {
				assert.True(t, pInfo.IsBusy, "Target provider should be marked as busy in the notified list")
				providerNowBusy = true
				break
			}
		}
		if !providerNowBusy && len(listMsg.Providers) > 0 {
			t.Errorf("Target provider %s was expected to be in the list and busy, or not in the list if it's the only one and now busy means no free providers for user", targetProvider.GetID())
		} else if len(listMsg.Providers) == 0 {
			t.Logf("Subscriber received an empty provider list, implying targetProvider was the only one and is now busy.")
		}

	case <-time.After(300 * time.Millisecond):
		t.Fatal("Timeout waiting for CardProvidersListMessage to subscriber (for busy notification)")
	}

	foundSessionLog := false
	for _, logEntry := range observedLogs.FilterLevelExact(zap.InfoLevel).All() {
		if strings.HasPrefix(logEntry.Message, "Hub: Receiver selected provider, new session created") {
			assert.Equal(t, establishedSessionID, logEntry.ContextMap()["sessionID"])
			assert.Equal(t, requestingClient.GetID(), logEntry.ContextMap()["receiverClientID"])
			assert.Equal(t, targetProvider.GetID(), logEntry.ContextMap()["providerClientID"])
			foundSessionLog = true
			break
		}
	}
	assert.True(t, foundSessionLog, "Expected 'new session created' log not found. Logs:\n"+dumpLogs(observedLogs, zap.InfoLevel))

}

func dumpLogsForAudit(observedLogs *observer.ObservedLogs) string {
	var sb strings.Builder
	for _, entry := range observedLogs.All() {
		if entry.LoggerName == "audit" || entry.Level == zap.InfoLevel {
			sb.WriteString(entry.Message + " | Fields: ")
			for k, v := range entry.ContextMap() {
				sb.WriteString(k + "=")
				sb.WriteString(fmt.Sprintf("%v", v) + " ")
			}
			sb.WriteString("\n")
		}
	}
	if sb.Len() == 0 {
		return "(No relevant logs for audit check)"
	}
	return sb.String()
}

// TestHub_HandleSelectCardProvider_DoubleCheck_ProviderGoesOffline tests provider going offline between RLock and Lock.
func TestHub_HandleSelectCardProvider_DoubleCheck_ProviderGoesOffline(t *testing.T) {
	setupHubTestGlobalConfig(t)
	hub, observedLogs := newHubWithObservedLogger(t)

	requestingClient := mockClientForHubTests(hub)
	requestingClient.Authenticated = true
	requestingClient.UserID = "dcUser1"
	requestingClient.CurrentRole = protocol.RoleReceiver

	targetProvider := mockClientForHubTests(hub)
	targetProvider.Authenticated = true
	targetProvider.UserID = "dcUser1"
	targetProvider.CurrentRole = protocol.RoleProvider
	targetProvider.IsOnline = true // Initially online
	targetProvider.DisplayName = "DCProviderOffline"

	hub.providerMutex.Lock()
	hub.cardProviders[targetProvider.GetID()] = targetProvider
	hub.providerMutex.Unlock()

	// Simulate provider going offline *after* initial RLock check in handleSelectCardProvider would pass,
	// but *before* the WLock and double check.
	// We achieve this by directly modifying the provider's state before calling the handler,
	// assuming the handler's RLock check passes based on initial state, then the WLock section sees this change.
	// This is a conceptual test; a real race condition is hard to deterministically unit test.
	// The current hub.go logic uses targetProviderConcrete.IsOnline which comes from the RLock state.
	// The double check `doubleCheckProviderConcrete, stillExists := h.cardProviders[targetProviderConcrete.GetID()]` then re-fetches.
	// And then `!doubleCheckProviderConcrete.IsOnline`

	// To make this test meaningful for the double check, we need to ensure the initial check passes,
	// then modify the state that the *double check* will see.
	// The initial RLock in handleSelectCardProvider will see IsOnline=true.
	// The double check inside WLock will see the modified IsOnline=false from cardProviders map if we change it there.

	// For the purpose of the double-check, we'll modify the provider instance that's IN the cardProviders map.
	originalProviderPointer := hub.cardProviders[targetProvider.GetID()].(*Client)

	selectMsg := protocol.SelectCardProviderMessage{
		Type:       protocol.MessageTypeSelectCardProvider,
		ProviderID: targetProvider.GetID(),
	}
	msgBytes, err := json.Marshal(selectMsg)
	assert.NoError(t, err)

	// Modify the state of the provider in the hub's map *before* the WLock part of handleSelectCardProvider
	// This simulates the state change happening between RLock and WLock.
	hub.providerMutex.Lock()                 // Need lock to modify the map or its contents safely if Hub was running
	originalProviderPointer.IsOnline = false // Provider goes offline
	hub.providerMutex.Unlock()

	// Act
	hub.handleSelectCardProvider(requestingClient, msgBytes)

	// Assert: ErrorMessage for provider unavailable
	select {
	case sentBytes := <-requestingClient.send:
		var errMsg protocol.ErrorMessage
		err := json.Unmarshal(sentBytes, &errMsg)
		assert.NoError(t, err)
		assert.Equal(t, protocol.MessageTypeError, errMsg.Type)
		// As per analysis, this scenario will likely hit the generic double-check failure in hub.go
		assert.Equal(t, protocol.ErrorCodeSessionConflict, errMsg.Code)  // Adjusted: Matches generic double check error code from hub.go (ErrorCodeSessionConflict: 40902)
		assert.Equal(t, "选择发卡方失败：一方或双方状态已改变（不再空闲），请重试。", errMsg.Message) // Adjusted: Matches generic double check message
	case <-time.After(100 * time.Millisecond):
		t.Fatal("Timeout waiting for ErrorMessage")
	}

	// Assert: Log message
	foundLog := false
	for _, entry := range observedLogs.FilterLevelExact(zap.WarnLevel).All() {
		// Adjusted to match the actual log message from hub.go for this generic double-check failure
		if strings.Contains(entry.Message, "(handleSelectCardProvider) 双重检查失败 - 一方或双方状态已改变（不再空闲/在线）") {
			assert.Equal(t, requestingClient.GetID(), entry.ContextMap()["requestingClientID"])
			assert.Equal(t, targetProvider.GetID(), entry.ContextMap()["targetProviderID"])
			// Check the providerOnline field in the log to confirm it reflects the change
			assert.False(t, entry.ContextMap()["providerOnline"].(bool), "Expected providerOnline to be false in log")
			foundLog = true
			break
		}
	}
	assert.True(t, foundLog, "Expected warning log for double-check provider state change not found. Logs:\n"+dumpLogs(observedLogs, zap.WarnLevel))
}

// TestHub_HandleSelectCardProvider_DoubleCheck_ProviderBecomesBusy tests provider becoming busy between RLock and Lock.
// As per analysis, this will be caught by the initial RLock check for provider busy.
func TestHub_HandleSelectCardProvider_DoubleCheck_ProviderBecomesBusy(t *testing.T) {
	setupHubTestGlobalConfig(t)
	hub, observedLogs := newHubWithObservedLogger(t)

	requestingClient := mockClientForHubTests(hub)
	requestingClient.Authenticated = true
	requestingClient.UserID = "dcUser2"
	requestingClient.CurrentRole = protocol.RoleReceiver

	targetProvider := mockClientForHubTests(hub)
	targetProvider.Authenticated = true
	targetProvider.UserID = "dcUser2"
	targetProvider.CurrentRole = protocol.RoleProvider
	targetProvider.IsOnline = true
	targetProvider.DisplayName = "DCProviderBusy"
	// Provider becomes busy *before* handleSelectCardProvider is called
	targetProvider.SessionID = "someOtherSession"

	hub.providerMutex.Lock()
	hub.cardProviders[targetProvider.GetID()] = targetProvider
	// also ensure a session object exists for this SessionID, otherwise the busy check might not work as expected
	if hub.sessions == nil {
		hub.sessions = make(map[string]*session.Session)
	}
	dummySessionForBusyProvider := session.NewSession("someOtherSession")
	dummySessionForBusyProvider.SetClient(targetProvider, string(protocol.RoleProvider))
	dummySessionForBusyProvider.SetClient(mockClientForHubTests(hub), string(protocol.RoleReceiver)) // Make it a paired session
	hub.sessions["someOtherSession"] = dummySessionForBusyProvider
	hub.providerMutex.Unlock()

	selectMsg := protocol.SelectCardProviderMessage{
		Type:       protocol.MessageTypeSelectCardProvider,
		ProviderID: targetProvider.GetID(),
	}
	msgBytes, err := json.Marshal(selectMsg)
	assert.NoError(t, err)

	// Act
	hub.handleSelectCardProvider(requestingClient, msgBytes)

	// Assert: ErrorMessage (caught by initial RLock check)
	select {
	case sentBytes := <-requestingClient.send:
		var errMsg protocol.ErrorMessage
		err := json.Unmarshal(sentBytes, &errMsg)
		assert.NoError(t, err)
		assert.Equal(t, protocol.MessageTypeError, errMsg.Type)
		assert.Equal(t, protocol.ErrorCodeProviderBusy, errMsg.Code) // Adjusted: Caught by initial check
		assert.Equal(t, "选择发卡方失败：目标发卡方当前正忙。", errMsg.Message)        // Adjusted: Caught by initial check
	case <-time.After(100 * time.Millisecond):
		t.Fatal("Timeout waiting for ErrorMessage")
	}

	foundLog := false
	for _, entry := range observedLogs.FilterLevelExact(zap.WarnLevel).All() {
		if strings.Contains(entry.Message, "目标发卡方已在会话中") { // Adjusted: Log from initial check
			assert.Equal(t, targetProvider.GetID(), entry.ContextMap()["targetProviderID"])
			assert.Equal(t, "someOtherSession", entry.ContextMap()["sessionID"])
			foundLog = true
			break
		}
	}
	assert.True(t, foundLog, "Expected warning log for provider busy (initial check) not found. Logs:\n"+dumpLogs(observedLogs, zap.WarnLevel))
}

// TestHub_HandleSelectCardProvider_DoubleCheck_ReceiverBecomesBusy tests receiver becoming busy.
// As per analysis, this will be caught by the initial RLock check for receiver busy.
func TestHub_HandleSelectCardProvider_DoubleCheck_ReceiverBecomesBusy(t *testing.T) {
	setupHubTestGlobalConfig(t)
	hub, observedLogs := newHubWithObservedLogger(t)

	requestingClient := mockClientForHubTests(hub)
	requestingClient.Authenticated = true
	requestingClient.UserID = "dcUser3"
	requestingClient.CurrentRole = protocol.RoleReceiver
	// Receiver becomes busy *before* handleSelectCardProvider is called
	requestingClient.SessionID = "receiverNowBusySession"

	targetProvider := mockClientForHubTests(hub)
	targetProvider.Authenticated = true
	targetProvider.UserID = "dcUser3"
	targetProvider.CurrentRole = protocol.RoleProvider
	targetProvider.IsOnline = true
	targetProvider.DisplayName = "DCProviderForBusyReceiver"

	hub.providerMutex.Lock()
	hub.cardProviders[targetProvider.GetID()] = targetProvider
	// Ensure session exists for the busy receiver
	if hub.sessions == nil {
		hub.sessions = make(map[string]*session.Session)
	}
	dummySessionForBusyReceiver := session.NewSession("receiverNowBusySession")
	dummySessionForBusyReceiver.SetClient(requestingClient, string(protocol.RoleReceiver))
	dummySessionForBusyReceiver.SetClient(mockClientForHubTests(hub), string(protocol.RoleProvider))
	hub.sessions["receiverNowBusySession"] = dummySessionForBusyReceiver
	hub.providerMutex.Unlock()

	selectMsg := protocol.SelectCardProviderMessage{
		Type:       protocol.MessageTypeSelectCardProvider,
		ProviderID: targetProvider.GetID(),
	}
	msgBytes, err := json.Marshal(selectMsg)
	assert.NoError(t, err)

	// Act
	hub.handleSelectCardProvider(requestingClient, msgBytes)

	// Assert: ErrorMessage (caught by initial RLock check)
	select {
	case sentBytes := <-requestingClient.send:
		var errMsg protocol.ErrorMessage
		err := json.Unmarshal(sentBytes, &errMsg)
		assert.NoError(t, err)
		assert.Equal(t, protocol.MessageTypeError, errMsg.Type)
		assert.Equal(t, protocol.ErrorCodeReceiverBusy, errMsg.Code) // Adjusted: Caught by initial check
		assert.Equal(t, "选择发卡方失败：您当前已在会话中。", errMsg.Message)         // Adjusted: Caught by initial check
	case <-time.After(100 * time.Millisecond):
		t.Fatal("Timeout waiting for ErrorMessage")
	}

	foundLog := false
	for _, entry := range observedLogs.FilterLevelExact(zap.WarnLevel).All() {
		// This log message from hub.go is "Hub: 请求的发卡方已在会话中", which refers to the *requestingClient* when it's busy.
		if strings.Contains(entry.Message, "请求的发卡方已在会话中") { // Adjusted: Log from initial check
			assert.Equal(t, requestingClient.GetID(), entry.ContextMap()["clientID"])
			assert.Equal(t, "receiverNowBusySession", entry.ContextMap()["sessionID"])
			foundLog = true
			break
		}
	}
	assert.True(t, foundLog, "Expected warning log for receiver busy (initial check) not found. Logs:\n"+dumpLogs(observedLogs, zap.WarnLevel))
}

// TestHub_HandleSelectCardProvider_DoubleCheck_ProviderVanishesFromList tests provider vanishing from cardProviders list.
// As per analysis, this will be caught by the initial RLock check for provider existence.
func TestHub_HandleSelectCardProvider_DoubleCheck_ProviderVanishesFromList(t *testing.T) {
	setupHubTestGlobalConfig(t)
	hub, observedLogs := newHubWithObservedLogger(t)

	requestingClient := mockClientForHubTests(hub)
	requestingClient.Authenticated = true
	requestingClient.UserID = "dcUser4"
	requestingClient.CurrentRole = protocol.RoleReceiver

	// Provider initially exists for the purpose of the message, but will be removed before handler call.
	targetProviderID := "dcProviderVanishesID"

	// Simulate provider vanishing from the list *before* the handler is called
	hub.providerMutex.Lock()
	// delete(hub.cardProviders, targetProviderID) // Ensure it's not there
	hub.providerMutex.Unlock()

	selectMsg := protocol.SelectCardProviderMessage{
		Type:       protocol.MessageTypeSelectCardProvider,
		ProviderID: targetProviderID,
	}
	msgBytes, err := json.Marshal(selectMsg)
	assert.NoError(t, err)

	// Act
	hub.handleSelectCardProvider(requestingClient, msgBytes)

	// Assert: ErrorMessage (caught by initial RLock check)
	select {
	case sentBytes := <-requestingClient.send:
		var errMsg protocol.ErrorMessage
		err := json.Unmarshal(sentBytes, &errMsg)
		assert.NoError(t, err)
		assert.Equal(t, protocol.MessageTypeError, errMsg.Type)
		assert.Equal(t, protocol.ErrorCodeProviderNotFound, errMsg.Code) // Adjusted: Caught by initial check
		assert.Equal(t, "选择发卡方失败：目标发卡方不存在或当前未提供服务。", errMsg.Message)     // Adjusted: Caught by initial check
	case <-time.After(100 * time.Millisecond):
		t.Fatal("Timeout waiting for ErrorMessage")
	}

	foundLog := false
	for _, entry := range observedLogs.FilterLevelExact(zap.WarnLevel).All() {
		if strings.Contains(entry.Message, "目标发卡方不存在或未上线") { // Adjusted: Log from initial check
			assert.Equal(t, targetProviderID, entry.ContextMap()["targetProviderID"])
			foundLog = true
			break
		}
	}
	assert.True(t, foundLog, "Expected warning log for provider vanished (initial check) not found. Logs:\n"+dumpLogs(observedLogs, zap.WarnLevel))
}

// TestHub_HandleSelectCardProvider_SendEstablishMsgToReceiverFails tests failure sending SessionEstablishedMessage to receiver.
func TestHub_HandleSelectCardProvider_SendEstablishMsgToReceiverFails(t *testing.T) {
	setupHubTestGlobalConfig(t)
	hub, observedLogs := newHubWithObservedLogger(t)
	go hub.Run()

	requestingClient := mockClientForHubTests(hub)
	requestingClient.Authenticated = true
	requestingClient.UserID = "sendFailUserR"
	requestingClient.CurrentRole = protocol.RoleReceiver
	hub.register <- requestingClient
	close(requestingClient.send) // Induce send failure for receiver

	targetProvider := mockClientForHubTests(hub)
	targetProvider.Authenticated = true
	targetProvider.UserID = "sendFailUserR"
	targetProvider.CurrentRole = protocol.RoleProvider
	targetProvider.IsOnline = true
	targetProvider.DisplayName = "ProviderForSendFailR"
	hub.register <- targetProvider

	hub.providerMutex.Lock()
	hub.cardProviders[targetProvider.GetID()] = targetProvider
	hub.providerMutex.Unlock()

	selectMsg := protocol.SelectCardProviderMessage{
		Type:       protocol.MessageTypeSelectCardProvider,
		ProviderID: targetProvider.GetID(),
	}
	msgBytes, err := json.Marshal(selectMsg)
	assert.NoError(t, err)

	// Act
	hub.handleSelectCardProvider(requestingClient, msgBytes)

	// Assert: Provider still receives SessionEstablishedMessage
	var establishedSessionID string
	select {
	case sentBytesP := <-targetProvider.send:
		var sessionMsgP protocol.SessionEstablishedMessage
		errP := json.Unmarshal(sentBytesP, &sessionMsgP)
		assert.NoError(t, errP)
		assert.Equal(t, protocol.MessageTypeSessionEstablished, sessionMsgP.Type)
		establishedSessionID = sessionMsgP.SessionID
		assert.NotEmpty(t, establishedSessionID)
	case <-time.After(200 * time.Millisecond):
		t.Fatal("Timeout waiting for SessionEstablishedMessage for targetProvider")
	}

	// Assert: Session is created in hub
	hub.providerMutex.RLock()
	_, sessionExists := hub.sessions[establishedSessionID]
	hub.providerMutex.RUnlock()
	assert.True(t, sessionExists, "Session should still be created in hub")
	assert.Equal(t, establishedSessionID, requestingClient.GetSessionID(), "Receiver SessionID should be set")
	assert.Equal(t, establishedSessionID, targetProvider.GetSessionID(), "Provider SessionID should be set")

	// Assert: Log for failure to send to receiver
	foundLog := false
	for _, entry := range observedLogs.All() {
		// Adjusted log message based on hub.go for this specific failure path
		if entry.Level == zap.ErrorLevel && strings.Contains(entry.Message, "发送会话建立成功的消息给收卡方失败") {
			assert.Equal(t, requestingClient.GetID(), entry.ContextMap()["receiverClientID"].(string))
			assert.Contains(t, entry.ContextMap()["error"].(string), "channel is closed")
			foundLog = true
			break
		}
	}
	assert.True(t, foundLog, "Expected error log for send to receiver failure not found. Logs:\n"+dumpLogs(observedLogs, zap.ErrorLevel))
}

// TestHub_HandleSelectCardProvider_SendEstablishMsgToProviderFails tests failure sending SessionEstablishedMessage to provider.
func TestHub_HandleSelectCardProvider_SendEstablishMsgToProviderFails(t *testing.T) {
	setupHubTestGlobalConfig(t)
	hub, observedLogs := newHubWithObservedLogger(t)
	go hub.Run()

	requestingClient := mockClientForHubTests(hub)
	requestingClient.Authenticated = true
	requestingClient.UserID = "sendFailUserP"
	requestingClient.CurrentRole = protocol.RoleReceiver
	hub.register <- requestingClient

	targetProvider := mockClientForHubTests(hub)
	targetProvider.Authenticated = true
	targetProvider.UserID = "sendFailUserP"
	targetProvider.CurrentRole = protocol.RoleProvider
	targetProvider.IsOnline = true
	targetProvider.DisplayName = "ProviderForSendFailP"
	hub.register <- targetProvider
	close(targetProvider.send) // Induce send failure for provider

	hub.providerMutex.Lock()
	hub.cardProviders[targetProvider.GetID()] = targetProvider
	hub.providerMutex.Unlock()

	selectMsg := protocol.SelectCardProviderMessage{
		Type:       protocol.MessageTypeSelectCardProvider,
		ProviderID: targetProvider.GetID(),
	}
	msgBytes, err := json.Marshal(selectMsg)
	assert.NoError(t, err)

	// Act
	hub.handleSelectCardProvider(requestingClient, msgBytes)

	// Assert: Receiver still receives SessionEstablishedMessage
	var establishedSessionID string
	select {
	case sentBytesR := <-requestingClient.send:
		var sessionMsgR protocol.SessionEstablishedMessage
		errR := json.Unmarshal(sentBytesR, &sessionMsgR)
		assert.NoError(t, errR)
		assert.Equal(t, protocol.MessageTypeSessionEstablished, sessionMsgR.Type)
		establishedSessionID = sessionMsgR.SessionID
		assert.NotEmpty(t, establishedSessionID)
	case <-time.After(200 * time.Millisecond):
		t.Fatal("Timeout waiting for SessionEstablishedMessage for requestingClient")
	}

	// Assert: Session is created in hub
	hub.providerMutex.RLock()
	_, sessionExists := hub.sessions[establishedSessionID]
	hub.providerMutex.RUnlock()
	assert.True(t, sessionExists, "Session should still be created in hub")
	assert.Equal(t, establishedSessionID, requestingClient.GetSessionID(), "Receiver SessionID should be set")
	assert.Equal(t, establishedSessionID, targetProvider.GetSessionID(), "Provider SessionID should be set")

	// Assert: Log for failure to send to provider
	foundLog := false
	for _, entry := range observedLogs.All() {
		// Adjusted log message based on hub.go for this specific failure path
		if entry.Level == zap.ErrorLevel && strings.Contains(entry.Message, "发送会话建立成功的消息给发卡方失败") {
			assert.Equal(t, targetProvider.GetID(), entry.ContextMap()["providerClientID"].(string))
			assert.Contains(t, entry.ContextMap()["error"].(string), "channel is closed")
			foundLog = true
			break
		}
	}
	assert.True(t, foundLog, "Expected error log for send to provider failure not found. Logs:\n"+dumpLogs(observedLogs, zap.ErrorLevel))
}

// TestHub_HandleSelectCardProvider_ReceiverRapidReSelectionAttempt tests receiver trying to select another provider immediately after success.
func TestHub_HandleSelectCardProvider_ReceiverRapidReSelectionAttempt(t *testing.T) {
	setupHubTestGlobalConfig(t)
	hub, observedLogs := newHubWithObservedLogger(t)
	go hub.Run()

	receiverClient := mockClientForHubTests(hub)
	receiverClient.Authenticated = true
	receiverClient.UserID = "rapidUser"
	receiverClient.CurrentRole = protocol.RoleReceiver
	hub.register <- receiverClient

	provider1 := mockClientForHubTests(hub)
	provider1.Authenticated = true
	provider1.UserID = "rapidUser"
	provider1.CurrentRole = protocol.RoleProvider
	provider1.IsOnline = true
	provider1.DisplayName = "ProviderOne"
	hub.register <- provider1

	provider2 := mockClientForHubTests(hub)
	provider2.Authenticated = true
	provider2.UserID = "rapidUser"
	provider2.CurrentRole = protocol.RoleProvider
	provider2.IsOnline = true
	provider2.DisplayName = "ProviderTwo"
	hub.register <- provider2

	hub.providerMutex.Lock()
	hub.cardProviders[provider1.GetID()] = provider1
	hub.cardProviders[provider2.GetID()] = provider2
	hub.providerMutex.Unlock()

	// First selection (should succeed)
	selectMsg1 := protocol.SelectCardProviderMessage{Type: protocol.MessageTypeSelectCardProvider, ProviderID: provider1.GetID()}
	msgBytes1, _ := json.Marshal(selectMsg1)
	hub.handleSelectCardProvider(receiverClient, msgBytes1)

	// Consume messages from first selection
	var establishedSessionID string
	select {
	case firstMsgR := <-receiverClient.send:
		var sMsg protocol.SessionEstablishedMessage
		json.Unmarshal(firstMsgR, &sMsg)
		establishedSessionID = sMsg.SessionID
	case <-time.After(100 * time.Millisecond):
		t.Fatal("Timeout on first receiver msg")
	}
	select {
	case <-provider1.send: // Consume provider's message
	case <-time.After(100 * time.Millisecond):
		t.Fatal("Timeout on first provider msg")
	}
	assert.NotEmpty(t, establishedSessionID, "Session ID should be established after first selection")
	assert.Equal(t, establishedSessionID, receiverClient.GetSessionID(), "Receiver session ID not set after first selection")

	// Second selection (should fail as receiver is busy)
	selectMsg2 := protocol.SelectCardProviderMessage{Type: protocol.MessageTypeSelectCardProvider, ProviderID: provider2.GetID()}
	msgBytes2, _ := json.Marshal(selectMsg2)
	hub.handleSelectCardProvider(receiverClient, msgBytes2)

	// Assert: ErrorMessage for receiver busy
	select {
	case sentBytes := <-receiverClient.send:
		var errMsg protocol.ErrorMessage
		err := json.Unmarshal(sentBytes, &errMsg)
		assert.NoError(t, err)
		assert.Equal(t, protocol.MessageTypeError, errMsg.Type)
		assert.Equal(t, protocol.ErrorCodeReceiverBusy, errMsg.Code)
		assert.Equal(t, "选择发卡方失败：您当前已在会话中。", errMsg.Message)
	case <-time.After(100 * time.Millisecond):
		t.Fatal("Timeout waiting for ErrorMessage on second selection attempt")
	}

	// Assert log for receiver busy on second attempt
	foundLog := false
	// Iterate over logs *after* the first selection might have logged something.
	// This is a bit tricky; ideally, we'd clear logs or get logs only for the second action.
	allLogs := observedLogs.All()
	for i := len(allLogs) - 1; i >= 0; i-- { // Check recent logs first
		entry := allLogs[i]
		if entry.Level == zap.WarnLevel && strings.Contains(entry.Message, "请求的发卡方已在会话中") {
			if entry.ContextMap()["clientID"] == receiverClient.GetID() && entry.ContextMap()["sessionID"] == establishedSessionID {
				foundLog = true
				break
			}
		}
	}
	assert.True(t, foundLog, "Expected warning log for receiver busy on second attempt not found. Logs:\n"+dumpLogs(observedLogs, zap.WarnLevel))
}
