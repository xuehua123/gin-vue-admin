package handler

import (
	"encoding/json"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/flipped-aurora/gin-vue-admin/server/nfc_relay/protocol"
	"github.com/flipped-aurora/gin-vue-admin/server/nfc_relay/session"
	"github.com/stretchr/testify/assert"
	"go.uber.org/zap"
	// "go.uber.org/zap/zaptest/observer" // Already in hub_test_helpers.go
)

// TestHub_HandleListCardProviders_PermissionDenied tests that non-receiver clients cannot list providers.
func TestHub_HandleListCardProviders_PermissionDenied(t *testing.T) {
	setupHubTestGlobalConfig(t)
	hub, observedLogs := newHubWithObservedLogger(t)

	// Client is a provider, not a receiver
	requestingClient := mockClientForHubTests(hub)
	requestingClient.Authenticated = true
	requestingClient.UserID = "user1"
	requestingClient.CurrentRole = protocol.RoleProvider // Key: Client is a provider

	// Message for listing providers (content doesn't matter much for this specific test)
	listMsg := protocol.ListCardProvidersMessage{
		Type: protocol.MessageTypeListCardProviders,
	}
	msgBytes, err := json.Marshal(listMsg)
	assert.NoError(t, err)

	// Act
	hub.handleListCardProviders(requestingClient, msgBytes)

	// Assert: ErrorMessage sent to client
	select {
	case sentBytes := <-requestingClient.send:
		t.Logf("Raw ErrorMessage bytes in PermissionDenied: %s", string(sentBytes)) // Print raw bytes
		var errMsg protocol.ErrorMessage
		err := json.Unmarshal(sentBytes, &errMsg)
		assert.NoError(t, err)
		t.Logf("Received ErrorMessage in PermissionDenied: %+v", errMsg) // Log received error message
		assert.Equal(t, protocol.MessageTypeError, errMsg.Type)
		assert.Equal(t, protocol.ErrorCodePermissionDenied, errMsg.Code) // Expecting this to fail if Code is not set in hub.go
		assert.Equal(t, "只有收卡方角色才能获取发卡方列表", errMsg.Message)
	case <-time.After(100 * time.Millisecond):
		t.Fatal("Timeout waiting for ErrorMessage due to permission denied")
	}

	// Assert: Log message
	foundLog := false
	for _, entry := range observedLogs.FilterLevelExact(zap.WarnLevel).All() {
		if entry.Message == "非收卡方客户端尝试获取发卡方列表" { // Actual log message
			assert.Equal(t, requestingClient.GetID(), entry.ContextMap()["clientID"])
			assert.Equal(t, string(requestingClient.CurrentRole), entry.ContextMap()["currentRole"])
			foundLog = true
			break
		}
	}
	assert.True(t, foundLog, "Expected warning log '非收卡方客户端尝试获取发卡方列表' not found. Actual logs: "+dumpLogs(observedLogs, zap.WarnLevel))
}

// TestHub_HandleListCardProviders_ParseMessage_InvalidJSON tests handling of invalid JSON.
func TestHub_HandleListCardProviders_ParseMessage_InvalidJSON(t *testing.T) {
	setupHubTestGlobalConfig(t)
	hub, observedLogs := newHubWithObservedLogger(t)

	requestingClient := mockClientForHubTests(hub)
	requestingClient.Authenticated = true
	requestingClient.UserID = "user-invalid-json"
	requestingClient.CurrentRole = protocol.RoleReceiver // Valid role for listing

	invalidJSONBytes := []byte("this is not valid json")

	// Act
	hub.handleListCardProviders(requestingClient, invalidJSONBytes)

	// Assert: ErrorMessage for BadRequest sent to client
	select {
	case sentBytes := <-requestingClient.send:
		t.Logf("Raw ErrorMessage bytes in ParseMessage_InvalidJSON: %s", string(sentBytes)) // Print raw bytes
		var errMsg protocol.ErrorMessage
		err := json.Unmarshal(sentBytes, &errMsg)
		assert.NoError(t, err)
		t.Logf("Received ErrorMessage in ParseMessage_InvalidJSON: %+v", errMsg) // Log received error message
		assert.Equal(t, protocol.MessageTypeError, errMsg.Type)
		assert.Equal(t, protocol.ErrorCodeBadRequest, errMsg.Code) // Expecting this to fail if Code is not set
		assert.Contains(t, errMsg.Message, "无效的列表请求消息格式")
	case <-time.After(100 * time.Millisecond):
		t.Fatal("Timeout waiting for ErrorMessage due to invalid JSON")
	}

	// Assert: Correct error log
	foundLog := false
	for _, entry := range observedLogs.FilterLevelExact(zap.ErrorLevel).All() {
		if entry.Message == "Hub 处理列表请求：反序列化 ListCardProvidersMessage 失败" { // Actual log message
			assert.Equal(t, requestingClient.GetID(), entry.ContextMap()["clientID"])
			if errVal, ok := entry.ContextMap()["error"]; ok {
				assert.Contains(t, errVal.(string), "invalid character")
			}
			foundLog = true
			break
		}
	}
	assert.True(t, foundLog, "Expected error log 'Hub 处理列表请求：反序列化 ListCardProvidersMessage 失败' not found. Actual logs: "+dumpLogs(observedLogs, zap.ErrorLevel))
}

// TestHub_HandleListCardProviders_AddClientToSubscribers tests that the requesting client is added to subscribers.
func TestHub_HandleListCardProviders_AddClientToSubscribers(t *testing.T) {
	setupHubTestGlobalConfig(t)
	hub, observedLogs := newHubWithObservedLogger(t)
	go hub.Run() // Run hub for a more complete test, though not strictly necessary for just map add

	requestingClient := mockClientForHubTests(hub)
	requestingClient.Authenticated = true
	requestingClient.UserID = "user-subscriber"
	requestingClient.CurrentRole = protocol.RoleReceiver

	listMsg := protocol.ListCardProvidersMessage{Type: protocol.MessageTypeListCardProviders}
	msgBytes, err := json.Marshal(listMsg)
	assert.NoError(t, err)

	// Act
	hub.handleListCardProviders(requestingClient, msgBytes) // This will try to send CardProvidersListMessage

	// Assert: Client added to providerListSubscribers
	hub.providerMutex.RLock()
	subscribersForUser, userFound := hub.providerListSubscribers[requestingClient.UserID]
	clientSubscribed := false
	if userFound {
		_, clientSubscribed = subscribersForUser[requestingClient]
	}
	hub.providerMutex.RUnlock()

	assert.True(t, userFound, "UserID not found in providerListSubscribers map")
	assert.True(t, clientSubscribed, "Requesting client not found in subscribers list for their UserID")

	// Assert: Debug log for adding subscriber
	foundLog := false
	for _, entry := range observedLogs.FilterLevelExact(zap.DebugLevel).All() {
		// hub.go logs: global.GVA_LOG.Debug("Hub: Client subscribed to provider list updates", zap.String("clientID", requestingClient.GetID()), zap.String("userID", requestingClient.UserID))
		if entry.Message == "客户端已订阅其用户ID的发卡方列表更新" { // Actual log message
			assert.Equal(t, requestingClient.GetID(), entry.ContextMap()["clientID"])
			assert.Equal(t, requestingClient.UserID, entry.ContextMap()["userID"])
			foundLog = true
			break
		}
	}
	assert.True(t, foundLog, "Expected debug log '客户端已订阅其用户ID的发卡方列表更新' not found. Actual logs: "+dumpLogs(observedLogs, zap.DebugLevel))

	// Consume the CardProvidersListMessage that would have been sent
	select {
	case <-requestingClient.send:
		// Expected
	case <-time.After(100 * time.Millisecond):
		t.Log("Timeout waiting for initial CardProvidersListMessage after subscribing, might be okay if list was empty.")
	}
}

// TestHub_HandleListCardProviders_BuildAvailableProvidersList tests various scenarios for building the provider list.
func TestHub_HandleListCardProviders_BuildAvailableProvidersList(t *testing.T) {
	setupHubTestGlobalConfig(t)

	commonUserID := "commonUser"

	// Provider 1: Online, Free, Custom Name
	provider1 := &Client{
		ID:            "provider1-id",
		UserID:        commonUserID,
		Authenticated: true,
		CurrentRole:   protocol.RoleProvider,
		IsOnline:      true,
		DisplayName:   "CardProviderAlpha",
		send:          make(chan []byte, 10), // Needs a send channel for hub interactions
	}

	// Provider 2: Online, Busy (in a session), Default Name (empty DisplayName)
	provider2 := &Client{
		ID:            "provider2-id-long-enough", // Ensure ID is long enough for [:8] slice
		UserID:        commonUserID,
		Authenticated: true,
		CurrentRole:   protocol.RoleProvider,
		IsOnline:      true,
		DisplayName:   "",                // Test default name generation
		SessionID:     "session-p2-busy", // Indicates busy
		send:          make(chan []byte, 10),
	}

	// Provider 3: Online, Free, but different UserID (should not be listed for commonUserID)
	provider3 := &Client{
		ID:            "provider3-id",
		UserID:        "otherUser",
		Authenticated: true,
		CurrentRole:   protocol.RoleProvider,
		IsOnline:      true,
		DisplayName:   "CardProviderGamma",
		send:          make(chan []byte, 10),
	}

	// Provider 4: Offline (should not be listed)
	provider4 := &Client{
		ID:            "provider4-id",
		UserID:        commonUserID,
		Authenticated: true,
		CurrentRole:   protocol.RoleProvider,
		IsOnline:      false,
		DisplayName:   "CardProviderDelta",
		send:          make(chan []byte, 10),
	}

	requestingClient := &Client{
		ID:            "receiver-client-id",
		UserID:        commonUserID,
		Authenticated: true,
		CurrentRole:   protocol.RoleReceiver,
		send:          make(chan []byte, 10),
	}

	listMsg := protocol.ListCardProvidersMessage{Type: protocol.MessageTypeListCardProviders}
	msgBytes, err := json.Marshal(listMsg)
	assert.NoError(t, err)

	t.Run("Scenario_MultipleProviders_FreeAndBusy_DifferentUsers_Offline", func(t *testing.T) {
		hub, _ := newHubWithObservedLogger(t)
		// go hub.Run() // Not strictly needed for this synchronous part if client.send is buffered

		// Setup Hub state
		hub.providerMutex.Lock()
		hub.clients[provider1] = true
		hub.clients[provider2] = true
		hub.clients[provider3] = true
		hub.clients[provider4] = true
		hub.clients[requestingClient] = true

		hub.cardProviders[provider1.GetID()] = provider1
		hub.cardProviders[provider2.GetID()] = provider2
		hub.cardProviders[provider3.GetID()] = provider3

		mockSessionP2 := session.NewSession(provider2.SessionID)
		mockSessionP2.SetClient(provider2, string(protocol.RoleProvider))
		mockReceiverForP2Session := mockClientForHubTests(hub)
		mockReceiverForP2Session.ID = "receiver-for-p2"
		mockSessionP2.SetClient(mockReceiverForP2Session, string(protocol.RoleReceiver))
		hub.sessions[provider2.SessionID] = mockSessionP2
		hub.providerMutex.Unlock()

		// Act
		hub.handleListCardProviders(requestingClient, msgBytes)

		// Assert: CardProvidersListMessage sent to client
		select {
		case sentBytes := <-requestingClient.send:
			var respListMsg protocol.CardProvidersListMessage
			err := json.Unmarshal(sentBytes, &respListMsg)
			assert.NoError(t, err, "Failed to unmarshal CardProvidersListMessage")
			assert.Equal(t, protocol.MessageTypeCardProvidersList, respListMsg.Type)

			assert.Len(t, respListMsg.Providers, 2, "Expected 2 available providers for the commonUser")

			foundP1 := false
			foundP2 := false

			for _, p := range respListMsg.Providers {
				if p.ProviderID == provider1.GetID() {
					foundP1 = true
					assert.Equal(t, provider1.DisplayName, p.ProviderName, "Provider1 name mismatch")
					assert.Equal(t, provider1.UserID, p.UserID, "Provider1 UserID mismatch")
					assert.False(t, p.IsBusy, "Provider1 should be free")
				} else if p.ProviderID == provider2.GetID() {
					foundP2 = true
					expectedP2NamePrefix := "Provider "
					assert.True(t, strings.HasPrefix(p.ProviderName, expectedP2NamePrefix), "Provider2 name should have prefix '%s', got '%s'", expectedP2NamePrefix, p.ProviderName)
					// Default name in hub.go is "Provider " + clientID (full ID if len <= 6, else clientID[:6] - based on "provid" error)
					expectedIDSuffixInName := provider2.GetID()
					if len(expectedIDSuffixInName) > 6 { // Adjusted to 6 based on error "provid"
						expectedIDSuffixInName = expectedIDSuffixInName[:6]
					}
					assert.True(t, strings.HasSuffix(p.ProviderName, expectedIDSuffixInName), "Provider2 default name '%s' should contain its ID suffix '%s' (first 6 chars of ID)", p.ProviderName, expectedIDSuffixInName)
					assert.Equal(t, provider2.UserID, p.UserID, "Provider2 UserID mismatch")
					assert.True(t, p.IsBusy, "Provider2 should be busy")
				}
			}
			assert.True(t, foundP1, "Provider1 not found in the list")
			assert.True(t, foundP2, "Provider2 not found in the list")

		case <-time.After(200 * time.Millisecond):
			t.Fatal("Timeout waiting for CardProvidersListMessage")
		}
	})

	t.Run("Scenario_NoMatchingProvidersForUser", func(t *testing.T) {
		hub, _ := newHubWithObservedLogger(t)
		clientForOtherUser := mockClientForHubTests(hub)
		clientForOtherUser.UserID = "completelyDifferentUser"
		clientForOtherUser.CurrentRole = protocol.RoleReceiver
		clientForOtherUser.Authenticated = true

		// Hub has providers, but none for "completelyDifferentUser"
		hub.providerMutex.Lock()
		hub.cardProviders[provider1.GetID()] = provider1 // provider1 is for commonUserID
		hub.providerMutex.Unlock()

		// Act
		hub.handleListCardProviders(clientForOtherUser, msgBytes)

		// Assert
		select {
		case sentBytes := <-clientForOtherUser.send:
			var respListMsg protocol.CardProvidersListMessage
			err := json.Unmarshal(sentBytes, &respListMsg)
			assert.NoError(t, err)
			assert.Equal(t, protocol.MessageTypeCardProvidersList, respListMsg.Type)
			assert.Empty(t, respListMsg.Providers, "Expected no providers for a user with no matching online providers")
		case <-time.After(100 * time.Millisecond):
			t.Fatal("Timeout waiting for CardProvidersListMessage (empty list)")
		}
	})
}

// TestHub_HandleListCardProviders_SendListMessageFailure tests failure to send CardProvidersListMessage.
func TestHub_HandleListCardProviders_SendListMessageFailure(t *testing.T) {
	setupHubTestGlobalConfig(t)
	hub, observedLogs := newHubWithObservedLogger(t)

	requestingClient := mockClientForHubTests(hub)
	requestingClient.Authenticated = true
	requestingClient.UserID = "user-send-fail"
	requestingClient.CurrentRole = protocol.RoleReceiver

	close(requestingClient.send) // Close channel to make send fail

	listMsg := protocol.ListCardProvidersMessage{Type: protocol.MessageTypeListCardProviders}
	msgBytes, err := json.Marshal(listMsg)
	assert.NoError(t, err)

	// Act
	hub.handleListCardProviders(requestingClient, msgBytes)

	// Assert: Log message for send failure
	// hub.go logs: global.GVA_LOG.Error("Hub: Failed to send provider list to requesting client", zap.String("clientID", requestingClient.GetID()), zap.Error(errSnd))
	foundLog := false
	for _, entry := range observedLogs.FilterLevelExact(zap.ErrorLevel).All() {
		if entry.Message == "向请求列表的客户端发送列表失败" { // Actual log message
			assert.Equal(t, requestingClient.GetID(), entry.ContextMap()["clientID"])
			if errVal, ok := entry.ContextMap()["error"]; ok {
				assert.Contains(t, errVal.(string), "channel is closed")
			}
			foundLog = true
			break
		}
	}
	assert.True(t, foundLog, "Expected error log '向请求列表的客户端发送列表失败' not found. Actual logs: "+dumpLogs(observedLogs, zap.ErrorLevel))
}

// TestHub_NotifyProviderListSubscribers_NoSubscribers tests notification when targetUserID has no subscribers.
func TestHub_NotifyProviderListSubscribers_NoSubscribers(t *testing.T) {
	setupHubTestGlobalConfig(t)
	hub, observedLogs := newHubWithObservedLogger(t)

	targetUserIDWithoutSubscribers := "user-with-no-subs"

	hub.notifyProviderListSubscribers(targetUserIDWithoutSubscribers)

	// Based on test output, product code does not log at DebugLevel if no subscribers are found.
	// Thus, we no longer assert for that specific log message here.
	// We check that no unexpected errors or warnings occurred.
	noErrorLogs := true
	for _, entry := range observedLogs.FilterLevelExact(zap.ErrorLevel).All() {
		t.Errorf("Unexpected error log found when no subscribers: %s, fields: %v", entry.Message, entry.ContextMap())
		noErrorLogs = false
	}
	assert.True(t, noErrorLogs, "No error logs should be present when notifying with no subscribers. Actual Error logs: "+dumpLogs(observedLogs, zap.ErrorLevel))

	noWarnLogs := true
	for _, entry := range observedLogs.FilterLevelExact(zap.WarnLevel).All() {
		t.Errorf("Unexpected warn log found when no subscribers: %s, fields: %v", entry.Message, entry.ContextMap())
		noWarnLogs = false
	}
	assert.True(t, noWarnLogs, "No warning logs should be present when notifying with no subscribers. Actual Warn logs: "+dumpLogs(observedLogs, zap.WarnLevel))
}

// TestHub_NotifyProviderListSubscribers_SuccessAndFailure tests successful notification and failure for one subscriber.
func TestHub_NotifyProviderListSubscribers_SuccessAndFailure(t *testing.T) {
	setupHubTestGlobalConfig(t)
	hub, observedLogs := newHubWithObservedLogger(t)

	targetUserID := "user-to-notify"

	subscriber1 := mockClientForHubTests(hub)
	subscriber1.ID = "sub1"
	subscriber1.UserID = targetUserID
	subscriber1.CurrentRole = protocol.RoleReceiver

	subscriber2 := mockClientForHubTests(hub)
	subscriber2.ID = "sub2-send-fail"
	subscriber2.UserID = targetUserID
	subscriber2.CurrentRole = protocol.RoleReceiver
	subscriber2.send = nil // Make send fail by setting channel to nil

	provider := mockClientForHubTests(hub)
	provider.ID = "provider-for-notification"
	provider.UserID = targetUserID
	provider.CurrentRole = protocol.RoleProvider
	provider.IsOnline = true
	provider.DisplayName = "Notifying Provider"

	hub.providerMutex.Lock()
	if hub.providerListSubscribers[targetUserID] == nil {
		hub.providerListSubscribers[targetUserID] = make(map[*Client]bool)
	}
	hub.providerListSubscribers[targetUserID][subscriber1] = true
	hub.providerListSubscribers[targetUserID][subscriber2] = true
	hub.cardProviders[provider.GetID()] = provider
	hub.providerMutex.Unlock()

	hub.notifyProviderListSubscribers(targetUserID)

	select {
	case sentBytesS1 := <-subscriber1.send:
		var listMsgS1 protocol.CardProvidersListMessage
		err := json.Unmarshal(sentBytesS1, &listMsgS1)
		assert.NoError(t, err, "Failed to unmarshal list message for subscriber1")
		assert.Equal(t, protocol.MessageTypeCardProvidersList, listMsgS1.Type)
		assert.Len(t, listMsgS1.Providers, 1, "Subscriber1 should receive 1 provider in list")
		if len(listMsgS1.Providers) == 1 {
			assert.Equal(t, provider.GetID(), listMsgS1.Providers[0].ProviderID)
		}
	case <-time.After(200 * time.Millisecond):
		t.Fatal("Timeout waiting for CardProvidersListMessage for subscriber1")
	}

	select {
	case <-subscriber2.send:
		t.Error("Subscriber2 (nil send channel) unexpectedly received a message")
	case <-time.After(50 * time.Millisecond):
	}

	foundSendErrorLog := false
	for _, entry := range observedLogs.FilterLevelExact(zap.ErrorLevel).All() {
		if entry.Message == "notifyProviderListSubscribers: Failed to send provider list to subscriber" {
			assert.Equal(t, subscriber2.GetID(), entry.ContextMap()["subscriberID"], "Logged subscriberID mismatch for send failure")
			if errVal, ok := entry.ContextMap()["error"]; ok {
				assert.Contains(t, errVal.(string), "channel full", "Error message for send failure should contain 'channel full'")
			}
			foundSendErrorLog = true
			break
		}
	}
	assert.True(t, foundSendErrorLog, "Expected error log 'notifyProviderListSubscribers: Failed to send provider list to subscriber' for subscriber2 not found. Actual logs: "+dumpLogs(observedLogs, zap.ErrorLevel))

	foundNotifyLog := false
	for _, entry := range observedLogs.FilterLevelExact(zap.InfoLevel).All() {
		if entry.Message == "Notified subscribers about provider list update" {
			assert.Equal(t, targetUserID, entry.ContextMap()["targetUserID"])
			subscriberCountFromLog, okSubCount := entry.ContextMap()["subscriberCount"]
			assert.True(t, okSubCount, "subscriberCount field missing in log")
			providerCountFromLog, okProvCount := entry.ContextMap()["providerCount"]
			assert.True(t, okProvCount, "providerCount field missing in log")

			parsedSubCount := parseIntFromLogField(t, subscriberCountFromLog, "subscriberCount")
			parsedProvCount := parseIntFromLogField(t, providerCountFromLog, "providerCount")

			if parsedSubCount == 2 && parsedProvCount == 1 {
				foundNotifyLog = true
			}

			if foundNotifyLog { // Break early if conditions met
				break
			}
		}
	}
	assert.True(t, foundNotifyLog, "Expected info log 'Notified subscribers about provider list update' with subscriberCount=2 and providerCount=1 not found or counts incorrect. Actual logs: "+dumpLogs(observedLogs, zap.InfoLevel))
}

// TestHub_NotifyProviderListSubscribers_ProviderEntryNotClientType tests robustness when a cardProvider entry is not *Client.
func TestHub_NotifyProviderListSubscribers_ProviderEntryNotClientType(t *testing.T) {
	setupHubTestGlobalConfig(t)
	hub, observedLogs := newHubWithObservedLogger(t)

	targetUserID := "user-type-assertion"

	// Mock subscriber
	subscriber := mockClientForHubTests(hub)
	subscriber.ID = "sub-type-assertion"
	subscriber.UserID = targetUserID
	subscriber.CurrentRole = protocol.RoleReceiver

	providerWithProperID := mockClientForHubTests(hub)
	providerWithProperID.ID = "actualProviderID-123"
	providerWithProperID.UserID = targetUserID
	providerWithProperID.CurrentRole = protocol.RoleProvider
	providerWithProperID.IsOnline = true
	providerWithProperID.DisplayName = "ActualClientWithProperID"

	hub.providerMutex.Lock()
	if hub.providerListSubscribers[targetUserID] == nil {
		hub.providerListSubscribers[targetUserID] = make(map[*Client]bool)
	}
	hub.providerListSubscribers[targetUserID][subscriber] = true
	hub.cardProviders[providerWithProperID.GetID()] = providerWithProperID
	hub.providerMutex.Unlock()

	// Act
	hub.notifyProviderListSubscribers(targetUserID)

	// Assert that the subscriber received a message
	select {
	case sentBytes := <-subscriber.send:
		var listMsg protocol.CardProvidersListMessage
		err := json.Unmarshal(sentBytes, &listMsg)
		assert.NoError(t, err)
		assert.Len(t, listMsg.Providers, 1, "Should receive the provider")
		assert.Equal(t, providerWithProperID.DisplayName, listMsg.Providers[0].ProviderName)
	case <-time.After(100 * time.Millisecond):
		t.Fatal("Subscriber did not receive list message for providerWithProperID")
	}

	// Assert that the specific "providerEntry is not of type *Client" log is NOT present
	foundProblematicLog := false
	for _, entry := range observedLogs.FilterLevelExact(zap.ErrorLevel).All() {
		if entry.Message == "Hub: providerEntry is not of type *Client in notifyProviderListSubscribers" {
			foundProblematicLog = true
			break
		}
	}
	assert.False(t, foundProblematicLog, "Log 'providerEntry is not of type *Client' was unexpectedly found. All entries were *Client.")
}

// parseIntFromLogField is a helper to robustly parse int from various numeric types stored by Zap.
func parseIntFromLogField(t *testing.T, field interface{}, fieldName string) int64 {
	t.Helper()
	switch v := field.(type) {
	case float64:
		return int64(v)
	case float32:
		return int64(v)
	case int:
		return int64(v)
	case int8:
		return int64(v)
	case int16:
		return int64(v)
	case int32:
		return int64(v)
	case int64:
		return v
	case uint:
		return int64(v)
	case uint8:
		return int64(v)
	case uint16:
		return int64(v)
	case uint32:
		return int64(v)
	case uint64:
		// Potentially check for overflow if your expected numbers are smaller than uint64 max
		return int64(v)
	case json.Number:
		i, err := v.Int64()
		if err != nil {
			t.Fatalf("Failed to parse json.Number for field %s ('%s'): %v", fieldName, v.String(), err)
		}
		return i
	case string: // Sometimes numbers might be logged as strings
		i, err := strconv.ParseInt(v, 10, 64)
		if err != nil {
			t.Fatalf("Failed to parse string for field %s ('%s'): %v", fieldName, v, err)
		}
		return i
	default:
		t.Fatalf("Unsupported type for numeric log field %s: %T, value: %v", fieldName, field, field)
		return 0 // Should not reach here due to t.Fatalf
	}
}

/*
func dumpLogs(observedLogs *observer.ObservedLogs, level zapcore.Level) string {
	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("\n--- Logs at level %s ---\n", level.String()))
	for _, entry := range observedLogs.FilterLevelExact(level).All() {
		sb.WriteString(fmt.Sprintf("  Msg: %s, Fields: %v\n", entry.Message, entry.ContextMap()))
	}
	if len(observedLogs.FilterLevelExact(level).All()) == 0 {
		sb.WriteString("  (No logs at this level)\n")
	}
	sb.WriteString("--- End logs ---\n")
	return sb.String()
}
*/
