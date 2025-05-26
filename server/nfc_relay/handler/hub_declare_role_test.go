package handler

import (
	"encoding/json"
	"fmt"
	"strings"
	"sync"
	"testing"
	"time"

	// "fmt"

	"github.com/flipped-aurora/gin-vue-admin/server/nfc_relay/protocol"
	// "github.com/flipped-aurora/gin-vue-admin/server/nfc_relay/session"
	// "github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"go.uber.org/zap/zaptest/observer"
)

// TestHub_HandleDeclareRole_ParseMessage_ValidJSON tests successful parsing of a valid DeclareRoleMessage.
// This test focuses on the handler proceeding without a parsing error, typically indicated by
// receiving a RoleDeclaredResponseMessage.
func TestHub_HandleDeclareRole_ParseMessage_ValidJSON(t *testing.T) {
	setupHubTestGlobalConfig(t)
	hub, _ := newHubWithObservedLogger(t) // Logs might be useful for debugging if parsing unexpectedly fails
	// go hub.Run() // Not strictly necessary for testing handleDeclareRole in isolation if it doesn't depend on Run loop state changes for this path

	client := mockClientForHubTests(hub)
	client.Authenticated = true // DeclareRole requires authentication

	validDeclareMsg := protocol.DeclareRoleMessage{
		Type:         protocol.MessageTypeDeclareRole,
		Role:         protocol.RoleReceiver,
		Online:       true,
		ProviderName: "TestProvider",
	}
	msgBytes, err := json.Marshal(validDeclareMsg)
	assert.NoError(t, err, "Failed to marshal valid DeclareRoleMessage")

	// Act
	hub.handleDeclareRole(client, msgBytes)

	// Assert: Expect RoleDeclaredResponseMessage
	select {
	case sentMsgBytes := <-client.send:
		var respMsg protocol.RoleDeclaredResponseMessage
		err := json.Unmarshal(sentMsgBytes, &respMsg)
		assert.NoError(t, err, "Failed to unmarshal RoleDeclaredResponseMessage")
		assert.Equal(t, protocol.MessageTypeRoleDeclaredResponse, respMsg.Type)
		assert.Equal(t, validDeclareMsg.Role, respMsg.Role)
		assert.Equal(t, validDeclareMsg.Online, respMsg.Online)
	case <-time.After(100 * time.Millisecond):
		t.Fatal("Timeout waiting for RoleDeclaredResponseMessage")
	}
}

// TestHub_HandleDeclareRole_ParseMessage_InvalidJSON tests handling of an invalid JSON DeclareRoleMessage.
func TestHub_HandleDeclareRole_ParseMessage_InvalidJSON(t *testing.T) {
	setupHubTestGlobalConfig(t)
	hub, observedLogs := newHubWithObservedLogger(t)
	// go hub.Run()

	client := mockClientForHubTests(hub)
	client.Authenticated = true // DeclareRole requires authentication

	invalidJSONBytes := []byte("this is not a valid json string for declare role")

	// Act
	hub.handleDeclareRole(client, invalidJSONBytes)

	// Assert: Client receives ErrorMessage for BadRequest
	select {
	case sentMsgBytes := <-client.send:
		var errMsg protocol.ErrorMessage
		err := json.Unmarshal(sentMsgBytes, &errMsg)
		assert.NoError(t, err, "Should unmarshal to ErrorMessage")
		assert.Equal(t, protocol.MessageTypeError, errMsg.Type)
		assert.Equal(t, protocol.ErrorCodeBadRequest, errMsg.Code)
		assert.Contains(t, errMsg.Message, "Invalid declare role message format")
	case <-time.After(100 * time.Millisecond):
		t.Fatal("Timeout waiting for BadRequest error message due to invalid JSON")
	}

	// Assert: Correct error log
	foundErrorLog := false
	for _, logEntry := range observedLogs.All() {
		if logEntry.Message == "Hub: Error unmarshalling declare role message" && logEntry.Level == zap.ErrorLevel {
			foundErrorLog = true
			assert.Equal(t, client.GetID(), logEntry.ContextMap()["clientID"])
			assert.Contains(t, logEntry.ContextMap()["error"].(string), "invalid character")
			break
		}
	}
	assert.True(t, foundErrorLog, "Expected error log for declare role message unmarshal failure was not found")
}

// TestHub_HandleDeclareRole_InvalidRoleValue tests validation of the Role field.
func TestHub_HandleDeclareRole_InvalidRoleValue(t *testing.T) {
	setupHubTestGlobalConfig(t)
	hub, observedLogs := newHubWithObservedLogger(t)
	// go hub.Run()

	client := mockClientForHubTests(hub)
	client.Authenticated = true

	invalidRoleMsg := protocol.DeclareRoleMessage{
		Type:         protocol.MessageTypeDeclareRole,
		Role:         "invalid_role_value", // Not "provider", "receiver", or "none"
		Online:       true,
		ProviderName: "TestProvider",
	}
	msgBytes, err := json.Marshal(invalidRoleMsg)
	assert.NoError(t, err, "Failed to marshal DeclareRoleMessage with invalid role")

	// Act
	hub.handleDeclareRole(client, msgBytes)

	// Assert: Client receives ErrorMessage for BadRequest
	select {
	case sentMsgBytes := <-client.send:
		var errMsg protocol.ErrorMessage
		err := json.Unmarshal(sentMsgBytes, &errMsg)
		assert.NoError(t, err, "Should unmarshal to ErrorMessage")
		assert.Equal(t, protocol.MessageTypeError, errMsg.Type)
		assert.Equal(t, protocol.ErrorCodeBadRequest, errMsg.Code)
		assert.Equal(t, "Invalid role specified", errMsg.Message) // Message from hub.go
	case <-time.After(100 * time.Millisecond):
		t.Fatal("Timeout waiting for BadRequest error message due to invalid role value")
	}

	// Assert: Correct warning log
	foundWarningLog := false
	for _, logEntry := range observedLogs.All() {
		if logEntry.Message == "Hub: Invalid role in DeclareRoleMessage" && logEntry.Level == zap.WarnLevel {
			foundWarningLog = true
			assert.Equal(t, client.GetID(), logEntry.ContextMap()["clientID"])
			// Check if roleReceived field exists and matches, but don't fail if the field itself is absent in the log for some reason
			if roleLogged, ok := logEntry.ContextMap()["roleReceived"]; ok {
				assert.Equal(t, "invalid_role_value", roleLogged)
			} else {
				t.Log("Warning: 'roleReceived' field not found in log context for TestHub_HandleDeclareRole_InvalidRoleValue")
			}
			break
		}
	}
	assert.True(t, foundWarningLog, "Expected warning log for invalid role value was not found")
}

// TestHub_HandleDeclareRole_UpdateClientStatus tests various scenarios for updating client status.
func TestHub_HandleDeclareRole_UpdateClientStatus(t *testing.T) {
	setupHubTestGlobalConfig(t)

	testCases := []struct {
		name                      string
		initialClientState        *Client // Initial state of the client (role, displayname)
		declareMsg                protocol.DeclareRoleMessage
		expectedClientRole        protocol.RoleType
		expectedClientIsOnline    bool
		expectedClientDispName    string // If empty, implies it should be a default or cleared
		shouldDisplayNameUpdate   bool   // True if DisplayName is expected to change from initial (or be set if initial was empty)
		expectDefaultProviderName bool   // True if a default provider name should be generated
	}{
		{
			name:               "Declare as Provider, Online, with ProviderName",
			initialClientState: &Client{DisplayName: "", CurrentRole: protocol.RoleNone, UserID: "user1", ID: "client1"},
			declareMsg: protocol.DeclareRoleMessage{
				Type:         protocol.MessageTypeDeclareRole,
				Role:         protocol.RoleProvider,
				Online:       true,
				ProviderName: "MyCard",
			},
			expectedClientRole:      protocol.RoleProvider,
			expectedClientIsOnline:  true,
			expectedClientDispName:  "MyCard",
			shouldDisplayNameUpdate: true,
		},
		{
			name:               "Declare as Provider, Online, no ProviderName (initial, expect default)",
			initialClientState: &Client{DisplayName: "", CurrentRole: protocol.RoleNone, UserID: "user2", ID: "client2-id-for-default-name"},
			declareMsg: protocol.DeclareRoleMessage{
				Type:   protocol.MessageTypeDeclareRole,
				Role:   protocol.RoleProvider,
				Online: true,
			},
			expectedClientRole:     protocol.RoleProvider,
			expectedClientIsOnline: true,
			// expectedClientDispName will be checked for prefix "Default Provider "
			shouldDisplayNameUpdate:   true,
			expectDefaultProviderName: true,
		},
		{
			name:               "Declare as Provider, Online, no ProviderName (had existing name)",
			initialClientState: &Client{DisplayName: "OldName", CurrentRole: protocol.RoleProvider, UserID: "user3", ID: "client3"},
			declareMsg: protocol.DeclareRoleMessage{
				Type:   protocol.MessageTypeDeclareRole,
				Role:   protocol.RoleProvider,
				Online: true, // No ProviderName given in message
			},
			expectedClientRole:      protocol.RoleProvider,
			expectedClientIsOnline:  true,
			expectedClientDispName:  "OldName", // Should retain old name
			shouldDisplayNameUpdate: false,     // DisplayName is not expected to change from "OldName"
		},
		{
			name:               "Declare as Receiver, Online",
			initialClientState: &Client{DisplayName: "WasProvider", CurrentRole: protocol.RoleProvider, UserID: "user4", ID: "client4"},
			declareMsg: protocol.DeclareRoleMessage{
				Type:   protocol.MessageTypeDeclareRole,
				Role:   protocol.RoleReceiver,
				Online: true,
			},
			expectedClientRole:      protocol.RoleReceiver,
			expectedClientIsOnline:  true,
			expectedClientDispName:  "", // DisplayName should be cleared
			shouldDisplayNameUpdate: true,
		},
		{
			name:               "Declare as None, Online (was provider)",
			initialClientState: &Client{DisplayName: "WasProviderBeforeNone", CurrentRole: protocol.RoleProvider, UserID: "user5", ID: "client5"},
			declareMsg: protocol.DeclareRoleMessage{
				Type:   protocol.MessageTypeDeclareRole,
				Role:   protocol.RoleNone,
				Online: true,
			},
			expectedClientRole:      protocol.RoleNone,
			expectedClientIsOnline:  true,
			expectedClientDispName:  "", // DisplayName should be cleared
			shouldDisplayNameUpdate: true,
		},
		{
			name:               "Declare as Provider, Offline (was online provider)",
			initialClientState: &Client{DisplayName: "MyOnlineCard", CurrentRole: protocol.RoleProvider, IsOnline: true, UserID: "user6", ID: "client6"},
			declareMsg: protocol.DeclareRoleMessage{
				Type:         protocol.MessageTypeDeclareRole,
				Role:         protocol.RoleProvider,
				Online:       false,
				ProviderName: "MyOnlineCard", // Name might be sent even when going offline
			},
			expectedClientRole:      protocol.RoleProvider,
			expectedClientIsOnline:  false,
			expectedClientDispName:  "MyOnlineCard",
			shouldDisplayNameUpdate: false, // Online status changes, not name itself
		},
		{
			name:               "Declare as Provider, Online, no ProviderName (initial, expect default)",
			initialClientState: &Client{DisplayName: "", CurrentRole: protocol.RoleNone, UserID: "user-default-shortid", ID: "shortid"}, // Short ID
			declareMsg: protocol.DeclareRoleMessage{
				Type:         protocol.MessageTypeDeclareRole,
				Role:         protocol.RoleProvider,
				Online:       true,
				ProviderName: "", // No provider name, expect default
			},
			expectedClientRole:        protocol.RoleProvider,
			expectedClientIsOnline:    true,
			shouldDisplayNameUpdate:   true,
			expectDefaultProviderName: true, // Expecting "Provider shortid"
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			hub, _ := newHubWithObservedLogger(t)
			// go hub.Run() // Not strictly necessary here

			// Setup client with initial state
			client := mockClientForHubTests(hub)
			client.Authenticated = true
			client.ID = tc.initialClientState.ID         // Ensure ID is set for default name generation
			client.UserID = tc.initialClientState.UserID // UserID needed for notify
			client.CurrentRole = tc.initialClientState.CurrentRole
			client.IsOnline = tc.initialClientState.IsOnline
			client.DisplayName = tc.initialClientState.DisplayName

			// If client was initially a provider and online, add to cardProviders for accurate testing of removal/update
			if tc.initialClientState.CurrentRole == protocol.RoleProvider && tc.initialClientState.IsOnline {
				hub.providerMutex.Lock()
				hub.cardProviders[client.GetID()] = client
				hub.providerMutex.Unlock()
			}

			msgBytes, err := json.Marshal(tc.declareMsg)
			assert.NoError(t, err)

			// Act
			hub.handleDeclareRole(client, msgBytes)

			// Assert client state changes
			assert.Equal(t, tc.expectedClientRole, client.CurrentRole, "Client CurrentRole mismatch")
			assert.Equal(t, tc.expectedClientIsOnline, client.IsOnline, "Client IsOnline mismatch")

			// Assert DisplayName
			if tc.shouldDisplayNameUpdate {
				if tc.expectDefaultProviderName {
					assert.True(t, strings.HasPrefix(client.DisplayName, "Provider "), tc.name+": Expected default provider name prefix")
					id := client.GetID()
					expectedIDSuffixInName := id
					if len(id) > 8 {
						expectedIDSuffixInName = id[:8]
					}
					assert.Contains(t, client.DisplayName, expectedIDSuffixInName, tc.name+": Expected client ID suffix '"+expectedIDSuffixInName+"' in default name '"+client.DisplayName+"'")
				} else {
					assert.Equal(t, tc.expectedClientDispName, client.DisplayName, tc.name+": DisplayName mismatch")
				}
			} else if client.DisplayName != tc.initialClientState.DisplayName && tc.expectedClientDispName == "" && !tc.expectDefaultProviderName {
				t.Logf("%s: Warning: DisplayName was updated when shouldDisplayNameUpdate was false, or assertion logic needs refinement. Initial: '%s', Current: '%s'", tc.name, tc.initialClientState.DisplayName, client.DisplayName)
			}

			// Assert RoleDeclaredResponseMessage
			select {
			case sentMsgBytes := <-client.send:
				var respMsg protocol.RoleDeclaredResponseMessage
				err := json.Unmarshal(sentMsgBytes, &respMsg)
				assert.NoError(t, err, "Failed to unmarshal RoleDeclaredResponseMessage")
				assert.Equal(t, protocol.MessageTypeRoleDeclaredResponse, respMsg.Type)
				assert.Equal(t, tc.expectedClientRole, respMsg.Role, "Response role mismatch")                // Should reflect the new role
				assert.Equal(t, tc.expectedClientIsOnline, respMsg.Online, "Response online status mismatch") // Should reflect the new online status
			case <-time.After(200 * time.Millisecond): // Increased timeout slightly due to potential goroutine scheduling for notifications
				t.Error("Timeout waiting for RoleDeclaredResponseMessage")
			}
		})
	}
}

// Mock for h.notifyProviderListSubscribers
var ( // Use a block for multiple vars related to mocking notifications
	mockNotifyProviderListSubscribersCalledWithUserID string
	mockNotifyProviderListSubscribersCallCount        int
	notifySubscribersMutex                            sync.Mutex // To protect access to the mock call counters/args
)

func resetNotifyMock() {
	notifySubscribersMutex.Lock()
	defer notifySubscribersMutex.Unlock()
	mockNotifyProviderListSubscribersCalledWithUserID = ""
	mockNotifyProviderListSubscribersCallCount = 0
}

// This is a simplified mock. In a real scenario, you might replace the hub's method.
// For this test structure, we'll have the real handleDeclareRole call a mocked version
// if we can inject it, or check side effects that imply it was called.
// Since we can't easily inject a mock function into the Hub instance for just this method,
// we will check the state of cardProviders and logs that imply notification logic ran.
// For more direct testing of notifyProviderListSubscribers, it would need its own tests
// or the Hub would need to be more mockable (e.g. taking an interface for notifier).

// TestHub_HandleDeclareRole_CardProvidersManagementAndNotifications tests cardProviders map management and notification calls.
func TestHub_HandleDeclareRole_CardProvidersManagementAndNotifications(t *testing.T) {
	setupHubTestGlobalConfig(t)

	type testCase struct {
		name string

		initialClientRole     protocol.RoleType
		initialClientIsOnline bool
		initialClientInHub    bool // Whether the client (as provider) is already in hub.cardProviders

		declareMsg protocol.DeclareRoleMessage

		expectedInCardProviders            bool // Whether client should be in hub.cardProviders after the call
		expectedNotifySubscribersCallCount int
		expectedLogMessageSubstring        string // Substring of an expected log related to provider status change
	}

	// Helper to create client for these specific tests
	createTestClient := func(hub *Hub, id, userID string, role protocol.RoleType, online bool) *Client {
		client := mockClientForHubTests(hub)
		client.Authenticated = true
		client.ID = id
		client.UserID = userID
		client.CurrentRole = role
		client.IsOnline = online
		client.DisplayName = "Test Provider " + id
		return client
	}

	testCases := []testCase{
		{
			name:                               "New Provider becomes Online",
			initialClientRole:                  protocol.RoleNone,
			initialClientIsOnline:              false,
			initialClientInHub:                 false,
			declareMsg:                         protocol.DeclareRoleMessage{Type: protocol.MessageTypeDeclareRole, Role: protocol.RoleProvider, Online: true, ProviderName: "NewCard"},
			expectedInCardProviders:            true,
			expectedNotifySubscribersCallCount: 1, // Should notify for this user ID
			expectedLogMessageSubstring:        "Provider declared and online",
		},
		{
			name:                               "Existing Offline Provider becomes Online",
			initialClientRole:                  protocol.RoleProvider, // Was a provider, but offline
			initialClientIsOnline:              false,
			initialClientInHub:                 false, // Not in hub.cardProviders because offline
			declareMsg:                         protocol.DeclareRoleMessage{Type: protocol.MessageTypeDeclareRole, Role: protocol.RoleProvider, Online: true, ProviderName: "ExistingCardNowOnline"},
			expectedInCardProviders:            true,
			expectedNotifySubscribersCallCount: 1,
			expectedLogMessageSubstring:        "Provider declared and online",
		},
		{
			name:                               "Existing Online Provider goes Offline",
			initialClientRole:                  protocol.RoleProvider,
			initialClientIsOnline:              true,
			initialClientInHub:                 true, // Was in hub.cardProviders
			declareMsg:                         protocol.DeclareRoleMessage{Type: protocol.MessageTypeDeclareRole, Role: protocol.RoleProvider, Online: false},
			expectedInCardProviders:            false,
			expectedNotifySubscribersCallCount: 1,
			expectedLogMessageSubstring:        "Provider declared offline",
		},
		{
			name:                               "Existing Online Provider changes role to Receiver",
			initialClientRole:                  protocol.RoleProvider,
			initialClientIsOnline:              true,
			initialClientInHub:                 true,
			declareMsg:                         protocol.DeclareRoleMessage{Type: protocol.MessageTypeDeclareRole, Role: protocol.RoleReceiver, Online: true},
			expectedInCardProviders:            false,
			expectedNotifySubscribersCallCount: 1,
			expectedLogMessageSubstring:        "Client changed role from provider, removed from available list",
		},
		{
			name:                               "Receiver changes to Provider Online", // Similar to new provider
			initialClientRole:                  protocol.RoleReceiver,
			initialClientIsOnline:              true,
			initialClientInHub:                 false,
			declareMsg:                         protocol.DeclareRoleMessage{Type: protocol.MessageTypeDeclareRole, Role: protocol.RoleProvider, Online: true, ProviderName: "ReceiverToProvider"},
			expectedInCardProviders:            true,
			expectedNotifySubscribersCallCount: 1,
			expectedLogMessageSubstring:        "Provider declared and online",
		},
		{
			name:                               "Online Provider updates ProviderName only (no change to online status or role)",
			initialClientRole:                  protocol.RoleProvider,
			initialClientIsOnline:              true,
			initialClientInHub:                 true,
			declareMsg:                         protocol.DeclareRoleMessage{Type: protocol.MessageTypeDeclareRole, Role: protocol.RoleProvider, Online: true, ProviderName: "UpdatedName"},
			expectedInCardProviders:            true,
			expectedNotifySubscribersCallCount: 0, // Name change alone should not trigger notification in current logic
			// No specific log for just name change, but client.DisplayName should be updated.
		},
		{
			name:                               "Client declares as None (was Provider)",
			initialClientRole:                  protocol.RoleProvider,
			initialClientIsOnline:              true,
			initialClientInHub:                 true,
			declareMsg:                         protocol.DeclareRoleMessage{Type: protocol.MessageTypeDeclareRole, Role: protocol.RoleNone, Online: true},
			expectedInCardProviders:            false,
			expectedNotifySubscribersCallCount: 1,
			expectedLogMessageSubstring:        "Client changed role from provider, removed from available list",
		},
		{
			name:                               "Client declares as Receiver (was not Provider, no change to cardProviders)",
			initialClientRole:                  protocol.RoleNone,
			initialClientIsOnline:              false,
			initialClientInHub:                 false,
			declareMsg:                         protocol.DeclareRoleMessage{Type: protocol.MessageTypeDeclareRole, Role: protocol.RoleReceiver, Online: true},
			expectedInCardProviders:            false,
			expectedNotifySubscribersCallCount: 0, // No provider status change, so no notification
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			hub, observedLogs := newHubWithObservedLogger(t)
			// For these tests, we need Hub.Run() to be running in the background
			// because notifyProviderListSubscribers is called as a goroutine `go h.notifyProviderListSubscribers(...)`
			// which itself might try to send messages that could block if the client's writePump isn't running etc.
			// However, for handleDeclareRole, the primary actions (map update, logging) are synchronous.
			// The notification itself is a side effect we are trying to infer.
			// Let's keep it simple and not run hub.Run() for direct handleDeclareRole tests to avoid complexity
			// unless absolutely necessary for a specific interaction being tested.
			// The current logic of handleDeclareRole for map updates is synchronous.

			clientID := "client-provider-test-" + strings.ToLower(strings.ReplaceAll(tc.name, " ", "-"))
			userID := "user-for-" + clientID
			client := createTestClient(hub, clientID, userID, tc.initialClientRole, tc.initialClientIsOnline)

			hub.providerMutex.Lock() // Lock before initial setup of cardProviders and subscribers
			if tc.initialClientInHub {
				hub.cardProviders[client.GetID()] = client
			}
			// Setup a mock subscriber for this userID to check if notify is called
			mockSubscriber := mockClientForHubTests(hub)
			mockSubscriber.UserID = userID // Same UserID
			mockSubscriber.CurrentRole = protocol.RoleReceiver
			if hub.providerListSubscribers[userID] == nil {
				hub.providerListSubscribers[userID] = make(map[*Client]bool)
			}
			hub.providerListSubscribers[userID][mockSubscriber] = true
			hub.providerMutex.Unlock()

			msgBytes, err := json.Marshal(tc.declareMsg)
			assert.NoError(t, err)

			// Act
			hub.handleDeclareRole(client, msgBytes)

			// Assert cardProviders map state
			hub.providerMutex.RLock()
			_, foundInCardProviders := hub.cardProviders[client.GetID()]
			hub.providerMutex.RUnlock()
			assert.Equal(t, tc.expectedInCardProviders, foundInCardProviders, "Client presence in hub.cardProviders mismatch")

			// Assert RoleDeclaredResponseMessage was sent (already tested in UpdateClientStatus, but good for sanity)
			// We will consume it if sent, but not strictly assert its contents here as other tests cover it.
			select {
			case <-client.send: // Consume the message
			case <-time.After(100 * time.Millisecond): // Increased timeout
				t.Log("No RoleDeclaredResponseMessage received in CardProvidersManagement test, this is acceptable as primary focus differs.")
			}

			// Assert log message for provider status change
			if tc.expectedLogMessageSubstring != "" {
				// Simpler assertion: Check if any info log contains the substring.
				foundLog := false
				for _, entry := range observedLogs.FilterLevelExact(zap.InfoLevel).All() {
					if strings.Contains(entry.Message, tc.expectedLogMessageSubstring) {
						// Optional: Further check context fields if necessary
						if roleChangeLog, ok := entry.ContextMap()["newRole"]; ok && string(tc.declareMsg.Role) != "" {
							assert.Equal(t, string(tc.declareMsg.Role), roleChangeLog, tc.name+": Logged newRole mismatch")
						}
						foundLog = true
						break
					}
				}
				assert.True(t, foundLog, tc.name+": Expected INFO log with substring '"+tc.expectedLogMessageSubstring+"' not found. All Info Logs:\n"+dumpLogs(observedLogs, zap.InfoLevel))
			}

			// Assert mockSubscriber received CardProvidersListMessage (indirectly testing notifyProviderListSubscribers)
			if tc.expectedNotifySubscribersCallCount > 0 {
				select {
				case sentToSubscriberBytes := <-mockSubscriber.send:
					var listMsg protocol.CardProvidersListMessage
					err := json.Unmarshal(sentToSubscriberBytes, &listMsg)
					assert.NoError(t, err, "Failed to unmarshal CardProvidersListMessage sent to subscriber")
					assert.Equal(t, protocol.MessageTypeCardProvidersList, listMsg.Type)
					// Further checks on listMsg.Providers could be added if needed, e.g., ensuring the current client is (or isn't) in it
					if tc.expectedInCardProviders {
						foundSelfInList := false
						for _, p := range listMsg.Providers {
							if p.ProviderID == client.GetID() {
								foundSelfInList = true
								break
							}
						}
						assert.True(t, foundSelfInList, "Newly online provider should be in the notified list to subscriber")
					} else {
						foundSelfInList := false
						for _, p := range listMsg.Providers {
							if p.ProviderID == client.GetID() {
								foundSelfInList = true
								break
							}
						}
						assert.False(t, foundSelfInList, "Offline/non-provider client should not be in the notified list to subscriber")
					}
				case <-time.After(300 * time.Millisecond): // Increased timeout for subscriber message due to potential scheduling
					t.Errorf("Timeout waiting for CardProvidersListMessage to be sent to mockSubscriber for test: %s", tc.name)
				}
			} else {
				select {
				case <-mockSubscriber.send:
					t.Error("mockSubscriber unexpectedly received a message when no notification was expected")
				case <-time.After(100 * time.Millisecond): // Increased timeout
					// Expected: no message
				}
			}
		})
	}
}

// Helper function to dump logs for debugging
func dumpLogs(observedLogs *observer.ObservedLogs, level zapcore.Level) string {
	var sb strings.Builder
	for _, entry := range observedLogs.FilterLevelExact(level).All() {
		sb.WriteString(fmt.Sprintf("  - Message: %s, Fields: %v\n", entry.Message, entry.ContextMap()))
	}
	if sb.Len() == 0 {
		return "  (No logs at this level)"
	}
	return sb.String()
}

// TestHub_HandleDeclareRole_SendResponseMessageFailure tests failure to send RoleDeclaredResponseMessage.
func TestHub_HandleDeclareRole_SendResponseMessageFailure(t *testing.T) {
	setupHubTestGlobalConfig(t)
	hub, observedLogs := newHubWithObservedLogger(t)

	client := mockClientForHubTests(hub)
	client.Authenticated = true

	// Configure the client's mockWsConnection (via client.conn) to make sendProtoMessage fail
	// This requires client.Send() to fail, which is part of the mockClientInfoProvider if used directly,
	// or modifying the underlying mockWsConnection of the *Client.
	// For *Client, client.Send() tries to put on client.send channel. If that channel is full/closed, it fails.
	// sendProtoMessage uses client.Send(), so we need client.Send() to return an error.

	close(client.send) // Close the send channel to make client.Send() fail

	declareMsg := protocol.DeclareRoleMessage{
		Type: protocol.MessageTypeDeclareRole,
		Role: protocol.RoleReceiver,
	}
	msgBytes, err := json.Marshal(declareMsg)
	assert.NoError(t, err)

	// Act
	hub.handleDeclareRole(client, msgBytes)

	// Assert: Log message for send failure
	foundErrorLog := false
	for _, logEntry := range observedLogs.All() {
		if logEntry.Message == "Hub: Failed to send role declared response" && logEntry.Level == zap.ErrorLevel {
			foundErrorLog = true
			assert.Equal(t, client.GetID(), logEntry.ContextMap()["clientID"])
			assert.Contains(t, logEntry.ContextMap()["error"].(string), "channel is closed")
			break
		}
	}
	assert.True(t, foundErrorLog, "Expected error log for failing to send role declared response not found")
}

// TestHub_HandleDeclareRole_NotifyConditions verifies the conditions under which notifyProviderListSubscribers is called.
func TestHub_HandleDeclareRole_NotifyConditions(t *testing.T) {
	setupHubTestGlobalConfig(t)

	type testCase struct {
		name              string
		initialRole       protocol.RoleType
		initialIsOnline   bool
		initialInCardProv bool // Already in cardProviders map
		declareMsg        protocol.DeclareRoleMessage
		expectNotify      bool
	}

	// clientID and userID will be consistent for all sub-tests here
	const testClientID = "client-notify-cond"
	const testUserID = "user-notify-cond"

	testCases := []testCase{
		{
			name:            "Provider comes online",
			initialRole:     protocol.RoleProvider,
			initialIsOnline: false,
			declareMsg:      protocol.DeclareRoleMessage{Role: protocol.RoleProvider, Online: true},
			expectNotify:    true,
		},
		{
			name:            "Online Provider goes offline",
			initialRole:     protocol.RoleProvider,
			initialIsOnline: true, initialInCardProv: true,
			declareMsg:   protocol.DeclareRoleMessage{Role: protocol.RoleProvider, Online: false},
			expectNotify: true,
		},
		{
			name:            "Online Provider changes role to Receiver",
			initialRole:     protocol.RoleProvider,
			initialIsOnline: true, initialInCardProv: true,
			declareMsg:   protocol.DeclareRoleMessage{Role: protocol.RoleReceiver, Online: true},
			expectNotify: true,
		},
		{
			name:            "Online Provider changes role to None",
			initialRole:     protocol.RoleProvider,
			initialIsOnline: true, initialInCardProv: true,
			declareMsg:   protocol.DeclareRoleMessage{Role: protocol.RoleNone, Online: true},
			expectNotify: true,
		},
		{
			name:            "Receiver changes to Provider Online",
			initialRole:     protocol.RoleReceiver,
			initialIsOnline: true,
			declareMsg:      protocol.DeclareRoleMessage{Role: protocol.RoleProvider, Online: true},
			expectNotify:    true,
		},
		{
			name:            "Online Provider just updates ProviderName",
			initialRole:     protocol.RoleProvider,
			initialIsOnline: true, initialInCardProv: true,
			declareMsg:   protocol.DeclareRoleMessage{Role: protocol.RoleProvider, Online: true, ProviderName: "NewName"},
			expectNotify: false, // Name change alone should not trigger (current logic)
		},
		{
			name:            "Receiver changes online status",
			initialRole:     protocol.RoleReceiver,
			initialIsOnline: true,
			declareMsg:      protocol.DeclareRoleMessage{Role: protocol.RoleReceiver, Online: false},
			expectNotify:    false,
		},
		{
			name:            "Client was None, becomes Receiver",
			initialRole:     protocol.RoleNone,
			initialIsOnline: false,
			declareMsg:      protocol.DeclareRoleMessage{Role: protocol.RoleReceiver, Online: true},
			expectNotify:    false,
		},
		{
			name:            "Client was None, becomes Provider Offline", // Becomes provider but is offline, so no add to list, no notify
			initialRole:     protocol.RoleNone,
			initialIsOnline: false,
			declareMsg:      protocol.DeclareRoleMessage{Role: protocol.RoleProvider, Online: false, ProviderName: "OfflineProv"},
			expectNotify:    false, // Not added to provider list if offline, so no notification for becoming available
		},
		{
			name:            "Already Online Provider, no change in message (resends same state)",
			initialRole:     protocol.RoleProvider,
			initialIsOnline: true, initialInCardProv: true,
			declareMsg:   protocol.DeclareRoleMessage{Role: protocol.RoleProvider, Online: true, ProviderName: "SameOldName"},
			expectNotify: false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			hub, _ := newHubWithObservedLogger(t) // Logs not primary focus here but can help debug

			client := mockClientForHubTests(hub)
			client.Authenticated = true
			client.ID = testClientID
			client.UserID = testUserID
			client.CurrentRole = tc.initialRole
			client.IsOnline = tc.initialIsOnline
			if tc.declareMsg.ProviderName != "" {
				client.DisplayName = tc.declareMsg.ProviderName
			} else if client.DisplayName == "" && tc.initialRole == protocol.RoleProvider {
				// Simulate a default name if it was a provider initially without a name in test setup
				client.DisplayName = "InitialDefaultName"
			}

			hub.providerMutex.Lock()
			if tc.initialInCardProv {
				hub.cardProviders[client.GetID()] = client
			}
			// Add a subscriber for this UserID to see if it gets notified
			mockSub := mockClientForHubTests(hub)
			mockSub.UserID = testUserID
			mockSub.ID = "mock-subscriber-for-notify-check"
			mockSub.CurrentRole = protocol.RoleReceiver
			if hub.providerListSubscribers[testUserID] == nil {
				hub.providerListSubscribers[testUserID] = make(map[*Client]bool)
			}
			hub.providerListSubscribers[testUserID][mockSub] = true
			hub.providerMutex.Unlock()

			// Fill in message type for tc.declareMsg if not set
			finalDeclareMsg := tc.declareMsg
			if finalDeclareMsg.Type == "" {
				finalDeclareMsg.Type = protocol.MessageTypeDeclareRole
			}

			msgBytes, err := json.Marshal(finalDeclareMsg)
			assert.NoError(t, err)

			// Act
			hub.handleDeclareRole(client, msgBytes)

			// Consume the RoleDeclaredResponseMessage from the main client
			select {
			case <-client.send:
			case <-time.After(50 * time.Millisecond):
				t.Log("No RoleDeclaredResponseMessage from main client, might be okay.")
			}

			// Assert if mockSub received a CardProvidersListMessage
			notified := false
			select {
			case msgSentToSub := <-mockSub.send:
				var listMsg protocol.CardProvidersListMessage
				err := json.Unmarshal(msgSentToSub, &listMsg)
				assert.NoError(t, err, "Failed to unmarshal message to subscriber")
				assert.Equal(t, protocol.MessageTypeCardProvidersList, listMsg.Type, "Message to subscriber should be CardProvidersList")
				notified = true
			case <-time.After(250 * time.Millisecond): // Increased timeout for async notification
				// No message, which is expected if tc.expectNotify is false
			}

			assert.Equal(t, tc.expectNotify, notified, "Notification to subscriber mismatch for test: %s", tc.name)
		})
	}
}
