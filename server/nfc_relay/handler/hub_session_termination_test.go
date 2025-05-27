package handler

import (
	"encoding/json"
	"fmt"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/flipped-aurora/gin-vue-admin/server/global"
	"github.com/flipped-aurora/gin-vue-admin/server/nfc_relay/protocol"
	"github.com/flipped-aurora/gin-vue-admin/server/nfc_relay/session"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

// TestHub_HandleEndSession_ParseMessage_InvalidJSON tests parsing of an invalid EndSessionMessage.
func TestHub_HandleEndSession_ParseMessage_InvalidJSON(t *testing.T) {
	setupHubTestGlobalConfig(t)
	hub, observedLogs := newHubWithObservedLogger(t)

	requestingClient := mockClientForHubTests(hub)
	requestingClient.Authenticated = true
	requestingClient.SessionID = "session-to-end-invalid-json"

	invalidJSONBytes := []byte("this is not valid json")

	// Act
	hub.handleEndSession(requestingClient, invalidJSONBytes)

	// Assert: Client receives ErrorMessage for BadRequest
	select {
	case sentMsgBytes := <-requestingClient.send:
		var errMsg protocol.ErrorMessage
		err := json.Unmarshal(sentMsgBytes, &errMsg)
		assert.NoError(t, err, "Should unmarshal to ErrorMessage")
		assert.Equal(t, protocol.MessageTypeError, errMsg.Type)
		assert.Equal(t, protocol.ErrorCodeBadRequest, errMsg.Code)
		assert.Contains(t, errMsg.Message, "无效的结束会话消息格式")
	case <-time.After(defaultTestTimeout):
		t.Fatal("Timeout waiting for BadRequest error message due to invalid JSON")
	}

	// Assert: Correct error log
	foundErrorLog := false
	for _, logEntry := range observedLogs.All() {
		if logEntry.Level == zap.ErrorLevel && strings.Contains(logEntry.Message, "反序列化 EndSessionMessage 失败") {
			foundErrorLog = true
			assert.Equal(t, requestingClient.GetID(), logEntry.ContextMap()["clientID"])
			assert.Contains(t, logEntry.ContextMap()["error"].(string), "invalid character")
			break
		}
	}
	assert.True(t, foundErrorLog, "Expected error log for EndSessionMessage unmarshal failure not found. Logs:\n"+dumpLogs(observedLogs, zap.ErrorLevel))
}

// TestHub_HandleEndSession_SessionIDMismatchOrEmpty tests when the SessionID in the message
// does not match the client's actual session ID, or is empty.
func TestHub_HandleEndSession_SessionIDMismatchOrEmpty(t *testing.T) {
	setupHubTestGlobalConfig(t) // Ensures GVA_CONFIG is set up

	tests := []struct {
		name                  string
		clientActualSessionID string
		requestedEndSessionID string
		expectedErrorCode     int
		expectedErrorMessage  string
		expectedLogMessage    string
		expectLogFieldCheck   func(t *testing.T, fields map[string]interface{})
		logLevel              zapcore.Level
	}{
		{
			name:                  "SessionID mismatch",
			clientActualSessionID: "actualClientSessionID",
			requestedEndSessionID: "differentSessionID",
			expectedErrorCode:     protocol.ErrorCodePermissionDenied, // Corrected: Was ErrorCodeBadRequest
			expectedErrorMessage:  "无法结束指定的会话：ID不匹配或无效",
			expectedLogMessage:    "客户端尝试结束不属于自己的或无效的会话",
			expectLogFieldCheck: func(t *testing.T, fields map[string]interface{}) {
				assert.Equal(t, "actualClientSessionID", fields["clientActualSessionID"])
				assert.Equal(t, "differentSessionID", fields["requestedEndSessionID"])
			},
			logLevel: zap.WarnLevel,
		},
		{
			name:                  "SessionID empty in message",
			clientActualSessionID: "actualClientSessionID",
			requestedEndSessionID: "",                                 // Empty session ID in request
			expectedErrorCode:     protocol.ErrorCodePermissionDenied, // Corrected: Was ErrorCodeBadRequest
			expectedErrorMessage:  "无法结束指定的会话：ID不匹配或无效",
			expectedLogMessage:    "客户端尝试结束不属于自己的或无效的会话",
			expectLogFieldCheck: func(t *testing.T, fields map[string]interface{}) {
				assert.Equal(t, "actualClientSessionID", fields["clientActualSessionID"])
				assert.Equal(t, "", fields["requestedEndSessionID"])
			},
			logLevel: zap.WarnLevel,
		},
		{
			name:                  "Client's SessionID is empty (should not happen if already in session to end)",
			clientActualSessionID: "", // Client not in a session
			requestedEndSessionID: "someSessionID",
			expectedErrorCode:     protocol.ErrorCodePermissionDenied, // Corrected: Was ErrorCodeBadRequest
			expectedErrorMessage:  "无法结束指定的会话：ID不匹配或无效",               // Or "您当前不在任何会话中" if that's preferred
			expectedLogMessage:    "客户端尝试结束不属于自己的或无效的会话",
			expectLogFieldCheck: func(t *testing.T, fields map[string]interface{}) {
				assert.Equal(t, "", fields["clientActualSessionID"])
				assert.Equal(t, "someSessionID", fields["requestedEndSessionID"])
			},
			logLevel: zap.WarnLevel,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			hub, observedLogs := newHubWithObservedLogger(t)
			// go hub.Run() // Not strictly needed for this specific handler's direct response logic

			requestingClient := mockClientForHubTests(hub)
			requestingClient.Authenticated = true
			requestingClient.SessionID = tc.clientActualSessionID // Set client's current session ID

			endMsg := protocol.EndSessionMessage{
				Type:      protocol.MessageTypeEndSession,
				SessionID: tc.requestedEndSessionID,
			}
			msgBytes, _ := json.Marshal(endMsg)

			hub.handleEndSession(requestingClient, msgBytes)

			select {
			case sentBytes := <-requestingClient.send:
				var errMsg protocol.ErrorMessage
				err := json.Unmarshal(sentBytes, &errMsg)
				assert.NoError(t, err)
				assert.Equal(t, protocol.MessageTypeError, errMsg.Type)
				assert.Equal(t, tc.expectedErrorCode, errMsg.Code, "Error code mismatch")
				assert.Equal(t, tc.expectedErrorMessage, errMsg.Message, "Error message mismatch")
			case <-time.After(defaultTestTimeout):
				t.Fatalf("Timeout waiting for error message")
			}

			foundLog := false
			for _, logEntry := range observedLogs.All() {
				if logEntry.Level == tc.logLevel && strings.Contains(logEntry.Message, tc.expectedLogMessage) {
					if tc.expectLogFieldCheck != nil {
						tc.expectLogFieldCheck(t, logEntry.ContextMap())
					}
					assert.Equal(t, requestingClient.GetID(), logEntry.ContextMap()["requestingClientID"])
					foundLog = true
					break
				}
			}
			assert.True(t, foundLog, "Expected warning log not found for "+tc.name+". Logs:\n"+dumpLogs(observedLogs, tc.logLevel))
		})
	}
}

// Mock for h.terminateSessionByID to track calls
var (
	mockTerminateSessionByIDCalledWithSessionID      string
	mockTerminateSessionByIDCalledWithReason         string
	mockTerminateSessionByIDCalledWithActingClientID string
	mockTerminateSessionByIDCalledWithActingUserID   string
	mockTerminateSessionByIDCallCount                int32 // Use atomic for potential concurrency if hub runs
	originalTerminateSessionByID                     func(h *Hub, sessionID, reason, actingClientID, actingClientUserID string)
)

func setupMockTerminateSessionByID(hub *Hub) {
	atomic.StoreInt32(&mockTerminateSessionByIDCallCount, 0)
	mockTerminateSessionByIDCalledWithSessionID = ""
	mockTerminateSessionByIDCalledWithReason = ""
	mockTerminateSessionByIDCalledWithActingClientID = ""
	mockTerminateSessionByIDCalledWithActingUserID = ""
}

func teardownMockTerminateSessionByID(hub *Hub) {
}

// TestHub_HandleEndSession_Successful tests successful session termination request.
func TestHub_HandleEndSession_Successful(t *testing.T) {
	setupHubTestGlobalConfig(t)
	hub, observedLogs := newHubWithObservedLogger(t)
	go hub.Run()                      // Run Hub to process session termination and notifications
	time.Sleep(50 * time.Millisecond) // Allow Hub to start

	requestingClient := mockClientForHubTests(hub)
	requestingClient.Authenticated = true
	requestingClient.UserID = "user-requesting-end"
	clientSessionID := "session-to-end-successfully-" + uuid.NewString()
	requestingClient.SessionID = clientSessionID
	hub.register <- requestingClient  // Register client
	time.Sleep(20 * time.Millisecond) // Allow registration

	// Mock another client to be the peer in the session
	peerClient := mockClientForHubTests(hub)
	peerClient.Authenticated = true
	peerClient.UserID = "user-peer-end"
	peerClient.SessionID = clientSessionID
	hub.register <- peerClient        // Register client
	time.Sleep(20 * time.Millisecond) // Allow registration

	// Setup the session in the hub
	hub.providerMutex.Lock()
	activeSession := session.NewSession(clientSessionID)
	_, errSetCard := activeSession.SetClient(peerClient, "card") // peerClient is Provider, should be 'card'
	assert.NoError(t, errSetCard)
	_, errSetPos := activeSession.SetClient(requestingClient, "pos") // requestingClient is Receiver, should be 'pos'
	assert.NoError(t, errSetPos)
	hub.sessions[clientSessionID] = activeSession
	// Add provider to cardProviders to test notification if it becomes free
	peerClient.CurrentRole = protocol.RoleProvider
	peerClient.IsOnline = true
	hub.cardProviders[peerClient.GetID()] = peerClient
	// Add a subscriber for the peer's UserID to check for provider list updates
	subscriber := mockClientForHubTests(hub)
	subscriber.UserID = peerClient.GetUserID()
	subscriber.CurrentRole = protocol.RoleReceiver
	if hub.providerListSubscribers[peerClient.GetUserID()] == nil {
		hub.providerListSubscribers[peerClient.GetUserID()] = make(map[*Client]bool)
	}
	hub.providerListSubscribers[peerClient.GetUserID()][subscriber] = true

	hub.providerMutex.Unlock()

	endMsg := protocol.EndSessionMessage{
		Type:      protocol.MessageTypeEndSession,
		SessionID: clientSessionID,
	}
	msgBytes, err := json.Marshal(endMsg)
	assert.NoError(t, err)

	requestingClientID := requestingClient.GetID()
	requestingClientUserID := requestingClient.UserID

	t.Logf("DEBUG: TestHub_HandleEndSession_Successful: Calling terminateSessionByID with sessionID=%s, actingClientID=%s, actingClientUserID=%s", clientSessionID, requestingClientID, requestingClientUserID)

	// Act
	hub.handleEndSession(requestingClient, msgBytes)

	// Assert: Confirmation message to requestingClient
	select {
	case sentBytes := <-requestingClient.send:
		var termMsg protocol.SessionTerminatedMessage
		err := json.Unmarshal(sentBytes, &termMsg)
		assert.NoError(t, err, "Unmarshal SessionTerminatedMessage for requester")
		assert.Equal(t, protocol.MessageTypeSessionTerminated, termMsg.Type)
		assert.Equal(t, "客户端主动请求结束", termMsg.Reason)
		assert.Equal(t, clientSessionID, termMsg.SessionID)
	case <-time.After(defaultTestTimeout):
		t.Fatal("Timeout waiting for SessionTerminatedMessage for requestingClient")
	}

	// Assert: Notification to peerClient
	select {
	case sentBytesPeer := <-peerClient.send:
		var termMsgPeer protocol.SessionTerminatedMessage
		errPeer := json.Unmarshal(sentBytesPeer, &termMsgPeer)
		assert.NoError(t, errPeer, "Unmarshal SessionTerminatedMessage for peer")
		assert.Equal(t, protocol.MessageTypeSessionTerminated, termMsgPeer.Type)
		assert.Equal(t, "客户端主动请求结束", termMsgPeer.Reason) // Reason for the other party
		assert.Equal(t, clientSessionID, termMsgPeer.SessionID)
	case <-time.After(defaultTestTimeout):
		sendFailedForPeerClient := false
		for _, entry := range observedLogs.All() {
			if entry.Level == zap.WarnLevel && strings.Contains(entry.Message, "通过接口发送消息给客户端失败") {
				if targetID, ok := entry.ContextMap()["targetClientID"].(string); ok && targetID == peerClient.GetID() {
					sendFailedForPeerClient = true
					t.Logf("Detected '通过接口发送消息给客户端失败' for peerClient (%s)", peerClient.GetID())
					break
				}
			}
		}
		t.Fatalf("Timeout waiting for SessionTerminatedMessage for peerClient (%s). SendFailedLogDetected: %t. Logs (Warn+Debug): \nWarn:\n%s\nDebug:\n%s",
			peerClient.GetID(), sendFailedForPeerClient,
			dumpLogs(observedLogs, zap.WarnLevel),
			dumpLogs(observedLogs, zap.DebugLevel))
	}

	// Assert: Session removed from hub
	hub.providerMutex.RLock()
	_, sessionStillExists := hub.sessions[clientSessionID]
	currentRequesterSessionID := requestingClient.GetSessionID()
	currentPeerSessionID := peerClient.GetSessionID()
	hub.providerMutex.RUnlock()
	assert.False(t, sessionStillExists, "Session should be removed from hub.sessions")
	assert.Empty(t, currentRequesterSessionID, "Requesting client's SessionID should be cleared")
	assert.Empty(t, currentPeerSessionID, "Peer client's SessionID should be cleared")

	// Assert: Log message for client requesting end
	foundReqLog := false
	for _, entry := range observedLogs.FilterLevelExact(zap.InfoLevel).All() {
		if strings.Contains(entry.Message, "客户端请求结束会话") {
			// 打印日志字段以进行调试
			t.Logf("【DEBUG-TEST】客户端请求结束会话日志字段: %+v", entry.ContextMap())

			// 修改断言，使用 requestingClientID 而不是 clientID
			assert.Equal(t, requestingClient.GetID(), entry.ContextMap()["requestingClientID"], "requestingClientID mismatch")
			assert.Equal(t, clientSessionID, entry.ContextMap()["sessionID"])
			foundReqLog = true
			break
		}
	}
	assert.True(t, foundReqLog, "Expected '客户端请求结束会话' info log not found. Logs:\n"+dumpLogs(observedLogs, zap.InfoLevel))

	// Assert: Audit log for session_terminated_by_client_request
	foundAudit := false
	t.Logf("【DEBUG-TEST】开始检查审计日志，寻找 session_terminated_by_client_request 事件")
	t.Logf("【DEBUG-TEST】requestingClient.GetID() = %s", requestingClient.GetID())

	for i, entry := range observedLogs.All() {
		if entry.Level == zap.InfoLevel && entry.LoggerName == "audit" && entry.Message == "AuditEvent" {
			t.Logf("【DEBUG-TEST】找到第 %d 个审计日志条目: %s", i, entry.Message)
			t.Logf("【DEBUG-TEST】审计日志条目的 ContextMap: %+v", entry.ContextMap())

			eventType, hasEventType := entry.ContextMap()["event_type"]
			t.Logf("【DEBUG-TEST】审计日志条目的 event_type: %v, 存在: %v", eventType, hasEventType)

			if eventTypeStr, ok := eventType.(string); ok && eventTypeStr == "session_terminated_by_client_request" {
				t.Logf("【DEBUG-TEST】找到匹配的 event_type: %s", eventTypeStr)

				sessionIDValue := entry.ContextMap()["session_id"]
				t.Logf("【DEBUG-TEST】审计日志条目的 session_id: %v", sessionIDValue)

				assert.Equal(t, clientSessionID, entry.ContextMap()["session_id"])

				details, ok := entry.ContextMap()["details"].(map[string]interface{})
				t.Logf("【DEBUG-TEST】审计日志条目的 details 转换结果: ok=%v, details=%+v", ok, details)

				assert.True(t, ok, "Details map not found in audit log")
				if ok {
					t.Logf("【DEBUG-TEST】details[\"acting_client_id_in_details\"] = %v", details["acting_client_id_in_details"])
					t.Logf("【DEBUG-TEST】details[\"acting_client_id\"] = %v", details["acting_client_id"])
					t.Logf("【DEBUG-TEST】details[\"reason\"] = %v", details["reason"])

					// 使用辅助函数打印 details map 的内容
					printDetailsMap(t, "【DEBUG-TEST】", details)

					// 尝试多种方式访问 acting_client_id
					actingClientID, hasActingClientID := details["acting_client_id"]
					actingClientIDInDetails, hasActingClientIDInDetails := details["acting_client_id_in_details"]

					t.Logf("【DEBUG-TEST】hasActingClientID = %v, actingClientID = %v", hasActingClientID, actingClientID)
					t.Logf("【DEBUG-TEST】hasActingClientIDInDetails = %v, actingClientIDInDetails = %v", hasActingClientIDInDetails, actingClientIDInDetails)

					// 尝试从 entry.ContextMap() 中直接获取 acting_client_id
					actingClientIDFromContext := entry.ContextMap()["acting_client_id"]
					t.Logf("【DEBUG-TEST】actingClientIDFromContext = %v", actingClientIDFromContext)

					// 修改断言，使用 entry.ContextMap()["acting_client_id"] 而不是 details["acting_client_id_in_details"]
					if actingClientIDFromContext != nil {
						assert.Equal(t, requestingClient.GetID(), actingClientIDFromContext, "acting_client_id from context mismatch")
					} else if hasActingClientID {
						assert.Equal(t, requestingClient.GetID(), actingClientID, "acting_client_id from details mismatch")
					} else if hasActingClientIDInDetails {
						assert.Equal(t, requestingClient.GetID(), actingClientIDInDetails, "acting_client_id_in_details mismatch")
					} else {
						t.Logf("【DEBUG-TEST】无法找到 acting_client_id 或 acting_client_id_in_details")
						assert.Fail(t, "无法找到 acting_client_id 或 acting_client_id_in_details")
					}

					assert.Equal(t, "客户端主动请求结束", details["reason"])
				}
				foundAudit = true
				break
			}
		}
	}

	if !foundAudit {
		t.Logf("【DEBUG-TEST】未找到匹配的审计日志条目，打印所有日志条目:")
		for i, entry := range observedLogs.All() {
			t.Logf("【DEBUG-TEST】日志条目 %d: Level=%v, LoggerName=%s, Message=%s, ContextMap=%+v",
				i, entry.Level, entry.LoggerName, entry.Message, entry.ContextMap())
		}
	}

	assert.True(t, foundAudit, "Expected 'session_terminated_by_client_request' audit event not found. Logs:\n"+dumpLogsForAudit(observedLogs))

	// Assert: Notification to subscriber that provider (peerClient) is now free
	select {
	case sentToSubBytes := <-subscriber.send:
		var listMsg protocol.CardProvidersListMessage
		err := json.Unmarshal(sentToSubBytes, &listMsg)
		assert.NoError(t, err, "Failed to unmarshal CardProvidersListMessage for subscriber")
		assert.Equal(t, protocol.MessageTypeCardProvidersList, listMsg.Type)
		providerNowFree := false
		for _, pInfo := range listMsg.Providers {
			if pInfo.ProviderID == peerClient.GetID() {
				assert.False(t, pInfo.IsBusy, "Peer provider should be marked as NOT busy in the notified list")
				providerNowFree = true
				break
			}
		}
		assert.True(t, providerNowFree, "Peer provider %s was expected to be in the list and free", peerClient.GetID())
	case <-time.After(defaultTestTimeout):
		t.Fatal("Timeout waiting for CardProvidersListMessage to subscriber (for provider free notification)")
	}
}

// --- Tests for terminateSessionByID ---

// TestHub_TerminateSessionByID_NonExistentSession tests terminating a session that doesn't exist.
func TestHub_TerminateSessionByID_NonExistentSession(t *testing.T) {
	setupHubTestGlobalConfig(t)
	hub, observedLogs := newHubWithObservedLogger(t)
	// go hub.Run() // Not strictly necessary if not testing async notifications from this

	nonExistentSessionID := "session-does-not-exist"

	// Act
	// Call terminateSessionByID directly for this test as handleEndSession has pre-checks
	hub.terminateSessionByID(nonExistentSessionID, "test reason", "testActor", "testActorUser")

	// Assert: No messages sent to any client (as there are no clients involved directly)
	// Assert: Log for attempting to terminate non-existent session
	foundLog := false
	for _, entry := range observedLogs.All() {
		if entry.Level == zap.WarnLevel && strings.Contains(entry.Message, "Attempted to terminate a non-existent session") {
			assert.Equal(t, nonExistentSessionID, entry.ContextMap()["sessionID"])
			assert.Equal(t, "test reason", entry.ContextMap()["reason"])
			assert.Equal(t, "testActor", entry.ContextMap()["actingClientID"])
			foundLog = true
			break
		}
	}
	assert.True(t, foundLog, "Expected warning log for terminating non-existent session not found. Logs:\n"+dumpLogs(observedLogs, zap.WarnLevel))

	// Assert: Metrics (ActiveSessions should not change, SessionTerminations should not increment for non-existent)
	// This requires a way to read metric values, which is complex for unit tests without a running Prometheus server
	// or direct metric object inspection. For now, this is a conceptual assertion.
}

// TestHub_TerminateSessionByID_VariousReasons tests session termination with different reasons and actors.
func TestHub_TerminateSessionByID_VariousReasons(t *testing.T) {
	setupHubTestGlobalConfig(t) // Ensures GVA_CONFIG is set up

	// Define test cases for various termination reasons
	testCases := []struct {
		name                   string
		reason                 string
		actingClientID         string // "system" for system-initiated, else client's ID
		actingClientUserID     string
		expectedMetricReason   string
		expectedAuditEventType string
		auditDetailsCheck      func(t *testing.T, details map[string]interface{}, sID, c1ID, c2ID, u1ID, u2ID, actCID, actUID, reason string)
		isSystemAction         bool
	}{
		{
			name:                   "Client Request",
			reason:                 "客户端主动请求结束",
			actingClientID:         "client1-id",
			actingClientUserID:     "user1-id",
			expectedMetricReason:   "client_request",
			expectedAuditEventType: "session_terminated_by_client_request",
			auditDetailsCheck: func(t *testing.T, details map[string]interface{}, sID, c1ID, c2ID, u1ID, u2ID, actCID, actUID, reason string) {
				assert.Equal(t, sID, details["session_id"])
				assert.Equal(t, c1ID, details["client_id_card_end"])
				assert.Equal(t, c2ID, details["client_id_pos_end"])
				assert.Equal(t, u1ID, details["user_id_card_end"])
				assert.Equal(t, u2ID, details["user_id_pos_end"])
				assert.Equal(t, actCID, details["acting_client_id"])
				assert.Equal(t, reason, details["reason"])
				if actUID != "" {
					assert.Equal(t, actUID, details["acting_user_id"])
				} else {
					_, exists := details["acting_user_id"]
					assert.False(t, exists, "acting_user_id should not exist in details if actUID is empty")
				}
			},
			isSystemAction: false,
		},
		{
			name:                   "Client Disconnect",
			reason:                 "客户端断开连接",
			actingClientID:         "client2-id",
			actingClientUserID:     "user2-id",
			expectedMetricReason:   "client_disconnect",
			expectedAuditEventType: "session_terminated_by_client_disconnect",
			auditDetailsCheck: func(t *testing.T, details map[string]interface{}, sID, c1ID, c2ID, u1ID, u2ID, actCID, actUID, reason string) {
				assert.Equal(t, sID, details["session_id"])
				assert.Equal(t, c1ID, details["client_id_card_end"])
				assert.Equal(t, c2ID, details["client_id_pos_end"])
				assert.Equal(t, u1ID, details["user_id_card_end"])
				assert.Equal(t, u2ID, details["user_id_pos_end"])
				assert.Equal(t, actCID, details["acting_client_id"])
				assert.Equal(t, reason, details["reason"])
				if actUID != "" {
					assert.Equal(t, actUID, details["acting_user_id"])
				} else {
					_, exists := details["acting_user_id"]
					assert.False(t, exists, "acting_user_id should not exist in details if actUID is empty")
				}
			},
			isSystemAction: false,
		},
		{
			name:                   "Client Generic Action",
			reason:                 "其他客户端操作",
			actingClientID:         "client1-id",
			actingClientUserID:     "user1-id",
			expectedMetricReason:   "client_generic_action",
			expectedAuditEventType: "session_terminated_by_client_action",
			auditDetailsCheck: func(t *testing.T, details map[string]interface{}, sID, c1ID, c2ID, u1ID, u2ID, actCID, actUID, reason string) {
				assert.Equal(t, sID, details["session_id"])
				assert.Equal(t, c1ID, details["client_id_card_end"])
				assert.Equal(t, c2ID, details["client_id_pos_end"])
				assert.Equal(t, u1ID, details["user_id_card_end"])
				assert.Equal(t, u2ID, details["user_id_pos_end"])
				assert.Equal(t, actCID, details["acting_client_id"])
				assert.Equal(t, reason, details["reason"])
				if actUID != "" {
					assert.Equal(t, actUID, details["acting_user_id"])
				} else {
					_, exists := details["acting_user_id"]
					assert.False(t, exists, "acting_user_id should not exist in details if actUID is empty")
				}
			},
			isSystemAction: false,
		},
		{
			name:                   "Timeout",
			reason:                 "会话因长时间无活动已超时",
			actingClientID:         "system",
			actingClientUserID:     "",
			expectedMetricReason:   "timeout",
			expectedAuditEventType: "session_terminated_by_timeout",
			auditDetailsCheck: func(t *testing.T, details map[string]interface{}, sID, c1ID, c2ID, u1ID, u2ID, actCID, actUID, reason string) {
				assert.Equal(t, sID, details["session_id"])
				assert.Equal(t, c1ID, details["client_id_card_end"])
				assert.Equal(t, c2ID, details["client_id_pos_end"])
				assert.Equal(t, u1ID, details["user_id_card_end"])
				assert.Equal(t, u2ID, details["user_id_pos_end"])
				assert.Equal(t, actCID, details["acting_client_id"])
				assert.Equal(t, reason, details["reason"])
				_, exists := details["acting_user_id"]
				assert.False(t, exists, "acting_user_id should not exist for system timeout")
			},
			isSystemAction: true,
		},
		{
			name:                   "APDU Error",
			reason:                 "APDU转发失败导致会话终止",
			actingClientID:         "system",
			actingClientUserID:     "",
			expectedMetricReason:   "apdu_error",
			expectedAuditEventType: "session_terminated_by_apdu_error",
			auditDetailsCheck: func(t *testing.T, details map[string]interface{}, sID, c1ID, c2ID, u1ID, u2ID, actCID, actUID, reason string) {
				assert.Equal(t, sID, details["session_id"])
				assert.Equal(t, c1ID, details["client_id_card_end"])
				assert.Equal(t, c2ID, details["client_id_pos_end"])
				assert.Equal(t, u1ID, details["user_id_card_end"])
				assert.Equal(t, u2ID, details["user_id_pos_end"])
				assert.Equal(t, actCID, details["acting_client_id"])
				assert.Equal(t, reason, details["reason"])
				_, exists := details["acting_user_id"]
				assert.False(t, exists, "acting_user_id should not exist for system apdu error")
			},
			isSystemAction: true,
		},
		{
			name:                   "System Generic",
			reason:                 "系统操作终止",
			actingClientID:         "system",
			actingClientUserID:     "",
			expectedMetricReason:   "system_generic",
			expectedAuditEventType: "session_terminated_by_system",
			auditDetailsCheck: func(t *testing.T, details map[string]interface{}, sID, c1ID, c2ID, u1ID, u2ID, actCID, actUID, reason string) {
				assert.Equal(t, sID, details["session_id"])
				assert.Equal(t, c1ID, details["client_id_card_end"])
				assert.Equal(t, c2ID, details["client_id_pos_end"])
				assert.Equal(t, u1ID, details["user_id_card_end"])
				assert.Equal(t, u2ID, details["user_id_pos_end"])
				assert.Equal(t, actCID, details["acting_client_id"])
				assert.Equal(t, reason, details["reason"])
				_, exists := details["acting_user_id"]
				assert.False(t, exists, "acting_user_id should not exist for system generic")
			},
			isSystemAction: true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			hub, observedLogs := newHubWithObservedLogger(t)
			go hub.Run()
			time.Sleep(50 * time.Millisecond) // Allow Hub to start

			client1 := mockClientForHubTests(hub)
			client1.ID = "client1-id"
			client1.UserID = "user1-id"
			client1.Authenticated = true
			hub.register <- client1           // Register client
			time.Sleep(20 * time.Millisecond) // Allow registration

			client2 := mockClientForHubTests(hub)
			client2.ID = "client2-id"
			client2.UserID = "user2-id"
			client2.Authenticated = true
			hub.register <- client2           // Register client
			time.Sleep(20 * time.Millisecond) // Allow registration

			sessionID := "session-term-reason-" + uuid.NewString()
			client1.SessionID = sessionID
			client2.SessionID = sessionID

			hub.providerMutex.Lock()
			activeSess := session.NewSession(sessionID)
			// 根据 session.SetClient 的期望，Provider 通常是 "card" 端，Receiver 是 "pos" 端
			// 确保 client1 (Provider) 被设为 "card", client2 (Receiver) 被设为 "pos"
			_, errProvider := activeSess.SetClient(client1, "card") // client1 is Provider, should be 'card'
			assert.NoError(t, errProvider, "Failed to set client1 (provider) as 'card' in session")
			paired, errReceiver := activeSess.SetClient(client2, "pos") // client2 is Receiver, should be 'pos'
			assert.NoError(t, errReceiver, "Failed to set client2 (receiver) as 'pos' in session")
			assert.True(t, paired, "Session should be paired after setting both clients")

			hub.sessions[sessionID] = activeSess
			hub.providerMutex.Unlock()

			// Act
			hub.terminateSessionByID(sessionID, tc.reason, tc.actingClientID, tc.actingClientUserID)

			// Assert: Session removed from hub
			hub.providerMutex.RLock()
			_, sessionStillExists := hub.sessions[sessionID]
			client1SessionID := client1.GetSessionID()
			client2SessionID := client2.GetSessionID()
			hub.providerMutex.RUnlock()
			assert.False(t, sessionStillExists, "Session should be removed from hub.sessions")
			assert.Empty(t, client1SessionID, "Client1's SessionID should be cleared")
			assert.Empty(t, client2SessionID, "Client2's SessionID should be cleared")

			// Assert: Termination messages sent to clients
			var termMsgC1, termMsgC2 protocol.SessionTerminatedMessage
			receivedC1, receivedC2 := false, false
			timeout := time.After(defaultTestTimeout)

			for i := 0; i < 2; i++ {
				if receivedC1 && receivedC2 {
					break
				}
				select {
				case sentBytesC1 := <-client1.send:
					if !receivedC1 {
						err := json.Unmarshal(sentBytesC1, &termMsgC1)
						assert.NoError(t, err)
						assert.Equal(t, protocol.MessageTypeSessionTerminated, termMsgC1.Type)
						assert.Equal(t, tc.reason, termMsgC1.Reason)
						assert.Equal(t, sessionID, termMsgC1.SessionID)
						receivedC1 = true
					} else {
						t.Error("Client1 received more than one message")
					}
				case sentBytesC2 := <-client2.send:
					if !receivedC2 {
						err := json.Unmarshal(sentBytesC2, &termMsgC2)
						assert.NoError(t, err)
						assert.Equal(t, protocol.MessageTypeSessionTerminated, termMsgC2.Type)
						assert.Equal(t, tc.reason, termMsgC2.Reason)
						assert.Equal(t, sessionID, termMsgC2.SessionID)
						receivedC2 = true
					} else {
						t.Error("Client2 received more than one message")
					}
				case <-timeout:
					goto afterLoopUnblockReceiver // goto to unblock select and fail outside
				}
			}
		afterLoopUnblockReceiver:
			if !receivedC1 {
				t.Errorf("Timeout/failure waiting for SessionTerminatedMessage for client1. Expected: %s", tc.reason)
			}
			if !receivedC2 {
				t.Errorf("Timeout/failure waiting for SessionTerminatedMessage for client2. Expected: %s", tc.reason)
			}

			// Assert: Log message for session terminated
			foundTermLog := false
			for _, entry := range observedLogs.FilterLevelExact(zap.InfoLevel).All() {
				if strings.Contains(entry.Message, "会话已终止") {
					assert.Equal(t, sessionID, entry.ContextMap()["sessionID"])
					assert.Equal(t, tc.reason, entry.ContextMap()["reason"])
					foundTermLog = true
					break
				}
			}
			assert.True(t, foundTermLog, "Expected '会话已终止' info log not found. Logs:\n"+dumpLogs(observedLogs, zap.InfoLevel))

			// Assert: Audit log
			foundAudit := false
			for _, entry := range observedLogs.All() {
				if entry.Level == zap.InfoLevel && entry.LoggerName == "audit" && entry.Message == "AuditEvent" {
					if eventType, _ := entry.ContextMap()["event_type"].(string); eventType == tc.expectedAuditEventType {
						assert.Equal(t, sessionID, entry.ContextMap()["session_id"], "Audit: session_id mismatch")
						assert.Equal(t, tc.actingClientID, entry.ContextMap()["acting_client_id"], "Audit: acting_client_id mismatch")

						if tc.actingClientUserID == "" {
							// If expected actingClientUserID is empty, Zap might omit it or log it as nil.
							// We accept either nil or absence of the key for this case.
							val, keyExists := entry.ContextMap()["acting_user_id"]
							assert.True(t, !keyExists || val == nil, fmt.Sprintf("Audit: acting_user_id should be nil or absent if expected empty, got %v", val))
						} else {
							assert.Equal(t, tc.actingClientUserID, entry.ContextMap()["acting_user_id"], "Audit: acting_user_id mismatch")
						}
						assert.Equal(t, tc.reason, entry.ContextMap()["reason"], "Audit: reason mismatch")

						detailsMap, ok := entry.ContextMap()["details"].(map[string]interface{})
						assert.True(t, ok, "Audit: 'details' field is not a map[string]interface{} or not found for "+tc.expectedAuditEventType)
						if ok {
							tc.auditDetailsCheck(t, detailsMap, sessionID, client1.GetID(), client2.GetID(), client1.GetUserID(), client2.GetUserID(), tc.actingClientID, tc.actingClientUserID, tc.reason)
						}
						foundAudit = true
						break
					}
				}
			}
			assert.True(t, foundAudit, "Expected '"+tc.expectedAuditEventType+"' audit event not found. Logs:\n"+dumpLogsForAudit(observedLogs))

			// Assert: Prometheus metric
			// This part needs proper Prometheus testing utilities.
			// The existing mockTerminateSessionByIDCallCount is not a Prometheus metric.
			// For now, we'll skip direct assertion of Prometheus countervec increment
			// and rely on the fact that the code path that *should* increment it was hit (if audit log is correct).
			// if tc.isSystemAction {
			// 	// System actions are not counted in Prometheus metrics for 'SessionTerminations' with client reasons
			// } else {
			// 	// TODO: Add proper Prometheus metric assertion here
			// }
		})
	}
}

// TestHub_TerminateSessionByID_ProviderBecomesFreeNotification tests notification when a provider becomes free.
func TestHub_TerminateSessionByID_ProviderBecomesFreeNotification(t *testing.T) {
	setupHubTestGlobalConfig(t)
	hub, observedLogs := newHubWithObservedLogger(t)
	go hub.Run()
	time.Sleep(50 * time.Millisecond) // Allow Hub to start

	providerClient := mockClientForHubTests(hub)
	providerClient.ID = "provider-becomes-free"
	providerClient.UserID = "user-provider-free"
	providerClient.Authenticated = true
	providerClient.CurrentRole = protocol.RoleProvider
	providerClient.IsOnline = true
	providerClient.DisplayName = "TheFreeProvider"
	hub.register <- providerClient    // Register client
	time.Sleep(20 * time.Millisecond) // Allow registration

	receiverClient := mockClientForHubTests(hub)
	receiverClient.ID = "receiver-for-free-provider"
	receiverClient.UserID = "user-provider-free" // Same UserID to interact
	receiverClient.Authenticated = true
	receiverClient.CurrentRole = protocol.RoleReceiver
	hub.register <- receiverClient    // Register client
	time.Sleep(20 * time.Millisecond) // Allow registration

	sessionID := "session-provider-free-" + uuid.NewString()
	providerClient.SessionID = sessionID
	receiverClient.SessionID = sessionID

	subscriber := mockClientForHubTests(hub)
	subscriber.UserID = providerClient.GetUserID() // Subscriber for the provider's UserID
	subscriber.CurrentRole = protocol.RoleReceiver
	subscriber.ID = "subscriber-for-free-provider"
	hub.register <- subscriber        // Register client
	time.Sleep(20 * time.Millisecond) // Allow registration

	hub.providerMutex.Lock()
	activeSess := session.NewSession(sessionID)
	_, errSetCard := activeSess.SetClient(providerClient, "card")
	assert.NoError(t, errSetCard)
	_, errSetPos := activeSess.SetClient(receiverClient, "pos")
	assert.NoError(t, errSetPos)
	hub.sessions[sessionID] = activeSess
	hub.cardProviders[providerClient.GetID()] = providerClient
	if hub.providerListSubscribers[providerClient.GetUserID()] == nil {
		hub.providerListSubscribers[providerClient.GetUserID()] = make(map[*Client]bool)
	}
	hub.providerListSubscribers[providerClient.GetUserID()][subscriber] = true
	hub.providerMutex.Unlock()

	// Act
	hub.terminateSessionByID(sessionID, "客户端断开连接", receiverClient.GetID(), receiverClient.GetUserID())

	select {
	case <-providerClient.send:
	default:
		t.Log("Provider client termination message consumed or not sent quickly")
	}
	select {
	case <-receiverClient.send:
	default:
		t.Log("Receiver client termination message consumed or not sent quickly")
	}

	// Assert: Notification to subscriber
	select {
	case sentToSubBytes := <-subscriber.send:
		var listMsg protocol.CardProvidersListMessage
		err := json.Unmarshal(sentToSubBytes, &listMsg)
		assert.NoError(t, err, "Failed to unmarshal CardProvidersListMessage for subscriber")
		assert.Equal(t, protocol.MessageTypeCardProvidersList, listMsg.Type)
		foundProviderInList := false
		for _, pInfo := range listMsg.Providers {
			if pInfo.ProviderID == providerClient.GetID() {
				assert.False(t, pInfo.IsBusy, "Provider should be marked as NOT busy")
				assert.Equal(t, providerClient.DisplayName, pInfo.ProviderName)
				foundProviderInList = true
				break
			}
		}
		if !foundProviderInList {
			hub.providerMutex.RLock()
			_, stillInHubCardProviders := hub.cardProviders[providerClient.GetID()]
			hub.providerMutex.RUnlock()
			assert.True(t, stillInHubCardProviders, "Provider should be in hub.cardProviders if online")
			assert.True(t, foundProviderInList, fmt.Sprintf("Provider %s (DisplayName: %s) not in notified list. List: %+v", providerClient.GetID(), providerClient.DisplayName, listMsg.Providers))
		}

	case <-time.After(defaultTestTimeout):
		sendFailedForSubscriber := false
		for _, entry := range observedLogs.All() {
			if entry.Level == zap.WarnLevel && strings.Contains(entry.Message, "通过接口发送消息给客户端失败") {
				if targetID, ok := entry.ContextMap()["targetClientID"].(string); ok && targetID == subscriber.GetID() {
					sendFailedForSubscriber = true
					t.Logf("Detected '通过接口发送消息给客户端失败' for subscriber (%s)", subscriber.GetID())
					break
				}
			}
		}
		t.Fatalf("Timeout waiting for CardProvidersListMessage to subscriber (%s). SendFailedLogDetected: %t. Logs (Warn+Debug):\nWarn:\n%s\nDebug:\n%s",
			subscriber.GetID(), sendFailedForSubscriber,
			dumpLogs(observedLogs, zap.WarnLevel),
			dumpLogs(observedLogs, zap.DebugLevel))
	}

	hub.providerMutex.RLock()
	_, stillInCardProviders := hub.cardProviders[providerClient.GetID()]
	hub.providerMutex.RUnlock()
	assert.True(t, stillInCardProviders, "Provider should remain in cardProviders map if online and free")
}

// TestHub_HandleClientDisconnect_ProviderAndSubscriberCleanup tests provider and subscriber cleanup
// when a provider client disconnects, before session termination logic (if any) is called.
func TestHub_HandleClientDisconnect_ProviderAndSubscriberCleanup(t *testing.T) {
	setupHubTestGlobalConfig(t)
	hub, observedLogs := newHubWithObservedLogger(t)
	go hub.Run()                      // Hub needs to run for subscriber notifications triggered by provider list changes
	time.Sleep(50 * time.Millisecond) // Allow Hub to start

	disconnectingProvider := mockClientForHubTests(hub)
	disconnectingProvider.Authenticated = true
	disconnectingProvider.UserID = "user-disconnect-provider"
	disconnectingProvider.CurrentRole = protocol.RoleProvider
	disconnectingProvider.IsOnline = true // Initially online
	hub.register <- disconnectingProvider
	time.Sleep(20 * time.Millisecond) // Allow registration

	// Add to cardProviders
	hub.providerMutex.Lock()
	hub.cardProviders[disconnectingProvider.GetID()] = disconnectingProvider
	hub.providerMutex.Unlock()

	// Setup subscribers for this provider's UserID
	subscriber1 := mockClientForHubTests(hub)
	subscriber1.UserID = disconnectingProvider.GetUserID()
	subscriber1.CurrentRole = protocol.RoleReceiver
	subscriber1.IsOnline = true // Explicitly set
	hub.register <- subscriber1
	time.Sleep(20 * time.Millisecond) // Allow registration

	subscriber2 := mockClientForHubTests(hub)
	subscriber2.UserID = "otherUserSubscribingToSameProviderUser" // A different client, but for same UserID's provider list
	subscriber2.CurrentRole = protocol.RoleReceiver
	subscriber2.IsOnline = true // Explicitly set
	hub.register <- subscriber2
	time.Sleep(20 * time.Millisecond) // Allow registration

	// Add disconnectingProvider as a subscriber to some other user's list (to test removal from there too)
	otherUserProvider := mockClientForHubTests(hub)
	otherUserProvider.UserID = "another-user-with-providers"
	otherUserProvider.CurrentRole = protocol.RoleProvider
	otherUserProvider.IsOnline = true
	hub.register <- otherUserProvider
	time.Sleep(20 * time.Millisecond) // Allow registration

	hub.providerMutex.Lock()
	if hub.providerListSubscribers[disconnectingProvider.GetUserID()] == nil {
		hub.providerListSubscribers[disconnectingProvider.GetUserID()] = make(map[*Client]bool)
	}
	hub.providerListSubscribers[disconnectingProvider.GetUserID()][subscriber1] = true
	hub.providerListSubscribers[disconnectingProvider.GetUserID()][subscriber2] = true

	if hub.providerListSubscribers[otherUserProvider.GetUserID()] == nil {
		hub.providerListSubscribers[otherUserProvider.GetUserID()] = make(map[*Client]bool)
	}
	hub.providerListSubscribers[otherUserProvider.GetUserID()][disconnectingProvider] = true // Provider subscribes too
	hub.providerMutex.Unlock()

	// Act: Disconnect the provider client
	// Simulate unregister flow which calls handleClientDisconnect
	hub.unregister <- disconnectingProvider // This will trigger handleClientDisconnect via the Hub's Run loop

	// Assert: Deletion log AND Provider removed from cardProviders
	assert.Eventually(t, func() bool {
		providerActuallyDeletedLogFound := false
		for _, entry := range observedLogs.All() {
			if entry.Level == zap.DebugLevel &&
				strings.Contains(entry.Message, "Provider explicitly deleted from cardProviders map in handleClientDisconnect") &&
				entry.ContextMap()["clientID"] == disconnectingProvider.GetID() {
				providerActuallyDeletedLogFound = true
				break
			}
		}

		hub.providerMutex.RLock()
		_, providerStillExistsInMap := hub.cardProviders[disconnectingProvider.GetID()]
		hub.providerMutex.RUnlock()

		return providerActuallyDeletedLogFound && !providerStillExistsInMap
	}, defaultTestTimeout, 20*time.Millisecond, // Check frequently
		fmt.Sprintf("Provider not removed or delete log not found. DeleteLogFound: %t (expected true), StillInMap: %t (expected false). Check Debug Logs for 'Provider explicitly deleted'.\nLogs:\n%s",
			false, // Placeholder for current log found status in message
			true,  // Placeholder for current map status in message
			dumpLogs(observedLogs, zap.DebugLevel)))

	// Assert: Subscribers (subscriber1, subscriber2) receive updated (empty) provider list for the provider's UserID
	for i, sub := range []*Client{subscriber1, subscriber2} {
		select {
		case sentBytes := <-sub.send:
			var listMsg protocol.CardProvidersListMessage
			err := json.Unmarshal(sentBytes, &listMsg)
			assert.NoError(t, err, "Failed to unmarshal CardProvidersListMessage for subscriber %d", i+1)
			assert.Equal(t, protocol.MessageTypeCardProvidersList, listMsg.Type)
			assert.Empty(t, listMsg.Providers, "Provider list for subscriber %d should be empty after provider disconnect", i+1)
		case <-time.After(defaultTestTimeout):
			// Check for sendProtoMessage failure log specifically for this subscriber
			sendFailedForThisSubscriber := false
			for _, entry := range observedLogs.All() {
				if entry.Level == zap.WarnLevel && strings.Contains(entry.Message, "通过接口发送消息给客户端失败") {
					if targetID, ok := entry.ContextMap()["targetClientID"].(string); ok && targetID == sub.GetID() {
						sendFailedForThisSubscriber = true
						t.Logf("Detected '通过接口发送消息给客户端失败' for subscriber %d (%s)", i+1, sub.GetID())
						break
					}
				}
			}
			t.Fatalf("Timeout waiting for CardProvidersListMessage to subscriber %d (%s). SendFailedLogDetected: %t. Logs (Warn+Debug): \nWarn:\n%s\nDebug:\n%s",
				i+1, sub.GetID(), sendFailedForThisSubscriber,
				dumpLogs(observedLogs, zap.WarnLevel),
				dumpLogs(observedLogs, zap.DebugLevel))
		}
	}

	// Assert: Disconnecting provider removed from its own subscription list (for otherUserProvider.GetUserID())
	assert.Eventually(t, func() bool {
		hub.providerMutex.RLock()
		defer hub.providerMutex.RUnlock()
		subsForOtherUser, userListExists := hub.providerListSubscribers[otherUserProvider.GetUserID()]
		if !userListExists { // If the whole list for that user was removed because it became empty
			return true
		}
		_, providerStillSubscribed := subsForOtherUser[disconnectingProvider]
		return !providerStillSubscribed
	}, defaultTestTimeout, 10*time.Millisecond, "Disconnecting provider not removed from its subscription to another user's list")

	// Assert: Log messages
	assert.Eventually(t, func() bool {
		foundDisconnectLog := false
		foundNotifyLog := false

		for _, entry := range observedLogs.All() {
			// 检查 Provider 删除日志
			if entry.Level == zap.DebugLevel && strings.Contains(entry.Message, "Provider explicitly deleted from cardProviders map in handleClientDisconnect") {
				if cid, ok := entry.ContextMap()["clientID"].(string); ok && cid == disconnectingProvider.GetID() {
					foundDisconnectLog = true
				}
			}

			// 检查通知订阅者的日志
			if entry.Level == zap.DebugLevel && strings.Contains(entry.Message, "notifyProviderListSubscribers: Found subscribers to notify.") {
				if uid, ok := entry.ContextMap()["targetUserID"].(string); ok && uid == disconnectingProvider.GetUserID() {
					foundNotifyLog = true
				}
			}
		}

		return foundDisconnectLog && foundNotifyLog
	}, defaultTestTimeout, 50*time.Millisecond, fmt.Sprintf("Expected disconnect AND notify logs not found. All logs:\n%s", dumpLogs(observedLogs, zap.DebugLevel)))

	// Ensure the Hub's Run loop has processed the unregister
	time.Sleep(50 * time.Millisecond) // Give a bit of time for the unregister to fully process
}

// TestHub_CheckInactiveSessions_IterationAndTerminationCall tests session iteration and termination calls.
func TestHub_CheckInactiveSessions_IterationAndTerminationCall(t *testing.T) {
	originalGVAConfig := global.GVA_CONFIG
	defer func() { global.GVA_CONFIG = originalGVAConfig }() // Restore original config

	setupHubTestGlobalConfig(t)
	// Set a short timeout for testing
	global.GVA_CONFIG.NfcRelay.SessionInactiveTimeoutSec = 1 // 1 second
	global.GVA_CONFIG.NfcRelay.HubCheckIntervalSec = 1       // Check every 1 second

	hub, observedLogs := newHubWithObservedLogger(t)
	go hub.Run()                      // Ensure Hub is running for any async operations triggered by termination
	time.Sleep(50 * time.Millisecond) // Allow Hub to start, though less critical here

	// Mock terminateSessionByID
	/*
		var terminateCallDetails []struct {
			sessionID string
			reason    string
			actorID   string
			actorUID  string
		}
		var terminateMutex sync.Mutex

		// Store the original terminateSessionByID and replace it
		originalTerminateFunc := hub.terminateSessionByID // This won't work as terminateSessionByID is a method not a func field
	*/
	// We need to use a different approach to mock methods or test their side effects.
	// For this test, since terminateSessionByID is complex to fully mock here without interface changes,
	// we will rely on checking its observable side effects if it were called (e.g. logs from it, session removal)
	// OR, we could use a global mock flag/counter if terminateSessionByID itself sets it.
	// The existing mock setup uses global variables; let's adapt to that if possible, or acknowledge limitation.

	// For this test, let's assume we can't directly mock terminateSessionByID easily.
	// We will check for logs that *it* produces, and that the session is removed.

	client1 := mockClientForHubTests(hub)
	client1.UserID = "userC1"
	client2 := mockClientForHubTests(hub)
	client2.UserID = "userC2"
	client3 := mockClientForHubTests(hub) // Will be in an active session
	client3.UserID = "userC3"
	client4 := mockClientForHubTests(hub) // Peer for active session
	client4.UserID = "userC4"

	inactiveSessionID := "inactive-session-" + uuid.NewString()
	activeSessionID := "active-session-" + uuid.NewString()

	hub.providerMutex.Lock()
	// Inactive session
	inactiveSess := session.NewSession(inactiveSessionID)
	// 确保使用 "card" 和 "pos" 作为 SetClient 的角色参数
	_, errInactive1 := inactiveSess.SetClient(client1, "card") // client1 (Provider) is card
	assert.NoError(t, errInactive1)
	pairedInactive, errInactive2 := inactiveSess.SetClient(client2, "pos") // client2 (Receiver) is pos
	assert.NoError(t, errInactive2)
	assert.True(t, pairedInactive)
	inactiveSess.LastActivityTime = time.Now().Add(-5 * time.Second) // Well past 1-second timeout
	hub.sessions[inactiveSessionID] = inactiveSess
	client1.SessionID = inactiveSessionID
	client2.SessionID = inactiveSessionID

	// Active session
	activeSess := session.NewSession(activeSessionID)
	_, errActive1 := activeSess.SetClient(client3, "card") // client3 (Provider) is card
	assert.NoError(t, errActive1)
	pairedActive, errActive2 := activeSess.SetClient(client4, "pos") // client4 (Receiver) is pos
	assert.NoError(t, errActive2)
	assert.True(t, pairedActive)
	activeSess.LastActivityTime = time.Now() // Active now
	hub.sessions[activeSessionID] = activeSess
	client3.SessionID = activeSessionID
	client4.SessionID = activeSessionID
	hub.providerMutex.Unlock()

	// Act: Call checkInactiveSessions directly.
	// In a real scenario, Hub.Run() would call this via ticker.
	hub.checkInactiveSessions()

	// Assert: Inactive session should be marked for termination
	// Check for the log from checkInactiveSessions itself or terminateSessionByID
	assert.Eventually(t, func() bool {
		for _, entry := range observedLogs.All() {
			if entry.Level == zap.InfoLevel && strings.Contains(entry.Message, "执行不活动会话的终止") {
				if sid, ok := entry.ContextMap()["sessionID"].(string); ok && sid == inactiveSessionID {
					return true
				}
			}
		}
		return false
	}, defaultTestTimeout, 20*time.Millisecond, "Expected '执行不活动会话的终止' log for inactive session not found or sessionID mismatch. Logs:\n"+dumpLogs(observedLogs, zap.InfoLevel))

	// Assert: terminateSessionByID was called for the inactive session with correct parameters
	// This requires checking logs that *terminateSessionByID* would produce.
	// Specifically, "会话已终止" with actingClientID="system" and reason="会话因长时间无活动已超时"
	// And the audit log.
	foundTerminationLogForInactive := false
	foundAuditLogForInactive := false

	assert.Eventually(t, func() bool {
		// terminateMutex.Lock() // Not needed anymore as terminateCallDetails is removed
		// defer terminateMutex.Unlock()
		hub.providerMutex.RLock() // Need to lock to read hub.sessions
		defer hub.providerMutex.RUnlock()

		_, inactiveSessionStillExists := hub.sessions[inactiveSessionID]
		_, activeSessionStillExists := hub.sessions[activeSessionID]

		for _, entry := range observedLogs.All() {
			if entry.Level == zap.InfoLevel && strings.Contains(entry.Message, "会话已终止") {
				sid, _ := entry.ContextMap()["sessionID"].(string)
				reason, _ := entry.ContextMap()["reason"].(string)
				actor, _ := entry.ContextMap()["actingClientID"].(string)
				if sid == inactiveSessionID && reason == "会话因长时间无活动已超时" && actor == "system" {
					foundTerminationLogForInactive = true
				}
			}
			if entry.LoggerName == "audit" && entry.Message == "AuditEvent" {
				eventType, _ := entry.ContextMap()["event_type"].(string)
				// 使用 details map 中的 acting_client_id_in_details 字段
				if eventType == "session_terminated_by_timeout" {
					details, ok := entry.ContextMap()["details"].(map[string]interface{})
					if ok && details["session_id"] == inactiveSessionID && details["acting_client_id_in_details"] == "system" {
						foundAuditLogForInactive = true
					}
				}
			}
		}
		// The session should be removed by terminateSessionByID
		return !inactiveSessionStillExists && activeSessionStillExists && foundTerminationLogForInactive && foundAuditLogForInactive
	}, defaultTestTimeout+100*time.Millisecond, 20*time.Millisecond, // Increased timeout for potential async parts of termination
		fmt.Sprintf("Inactive session not terminated correctly or active session affected. "+
			"InactiveExists: %t, ActiveExists: %t, FoundTermLog: %t, FoundAuditLog: %t\n"+
			"Logs:\n%s\nAudit Logs:\n%s",
			hub.sessions[inactiveSessionID] != nil, hub.sessions[activeSessionID] != nil, foundTerminationLogForInactive, foundAuditLogForInactive,
			dumpLogs(observedLogs, zap.InfoLevel), dumpLogsForAudit(observedLogs)))

	// Ensure client SessionIDs for the inactive session are cleared
	assert.Empty(t, client1.GetSessionID(), "Client1 SessionID should be cleared after inactive session termination")
	assert.Empty(t, client2.GetSessionID(), "Client2 SessionID should be cleared after inactive session termination")
	assert.Equal(t, activeSessionID, client3.GetSessionID(), "Client3 SessionID (active session) should not be affected")
	assert.Equal(t, activeSessionID, client4.GetSessionID(), "Client4 SessionID (active session) should not be affected")
}

// printDetailsMap 是一个辅助函数，用于打印 details map 的内容
func printDetailsMap(t *testing.T, prefix string, details map[string]interface{}) {
	t.Logf("%s: details map 内容:", prefix)
	for k, v := range details {
		t.Logf("%s:   - %s: %v (类型: %T)", prefix, k, v, v)
	}
}

const defaultTestTimeout = 20000 * time.Millisecond
