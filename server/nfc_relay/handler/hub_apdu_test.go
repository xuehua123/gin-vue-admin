package handler

import (
	"encoding/json"
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/gorilla/websocket"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"go.uber.org/zap/zaptest/observer"

	"github.com/flipped-aurora/gin-vue-admin/server/global"
	"github.com/flipped-aurora/gin-vue-admin/server/nfc_relay/protocol"
	"github.com/flipped-aurora/gin-vue-admin/server/nfc_relay/session"
)

// MockPeerClient is a mock type for the session.ClientInfoProvider interface, specifically for the peer.
type MockPeerClient struct {
	mock.Mock
	// Add fields that are part of ClientInfoProvider if needed for GetID, GetRole etc.
	// For APDU exchange, primarily Send is important.
	IDValue        string
	RoleValue      string
	SessionIDValue string
	UserIDValue    string
}

func (m *MockPeerClient) GetID() string {
	args := m.Called()
	if args.Get(0) != nil {
		return args.String(0)
	}
	return m.IDValue
}

func (m *MockPeerClient) GetRole() string { // Corrected signature
	args := m.Called()
	if args.Get(0) != nil {
		return args.String(0)
	}
	return m.RoleValue
}

func (m *MockPeerClient) GetSessionID() string {
	args := m.Called()
	if args.Get(0) != nil {
		return args.String(0)
	}
	return m.SessionIDValue
}

func (m *MockPeerClient) GetUserID() string {
	args := m.Called()
	if args.Get(0) != nil {
		return args.String(0)
	}
	return m.UserIDValue
}

func (m *MockPeerClient) Send(message []byte) error {
	args := m.Called(message)
	return args.Error(0)
}

// --- Test Setup Helper ---
type apduTestMocks struct {
	hub            *Hub
	sourceClient   *Client
	mockPeerClient *MockPeerClient
	activeSession  *session.Session
	observedLogs   *observer.ObservedLogs
}

func setupAPDUTest(t *testing.T) apduTestMocks {
	core, observed := observer.New(zapcore.InfoLevel)
	global.GVA_LOG = zap.New(core)
	global.InitializeAuditLogger()

	h := NewHub()
	// For most tests, a non-nil conn is needed, but for RemoteAddr() to work without panic,
	// it needs to be a real or properly mocked connection.
	// Using &websocket.Conn{} can lead to panic if RemoteAddr() is called on its zero-value internal net.Conn.
	// We will handle this per test case if needed, or by improving sendErrorMessage in hub.go.
	mockWsConn := &websocket.Conn{}
	sourceClient := NewClient(h, mockWsConn)
	sourceClient.ID = "sourceClient123"
	sourceClient.UserID = "user1"
	sourceClient.SessionID = "session123"
	sourceClient.Authenticated = true
	// sourceClient.CurrentRole is implicitly protocol.RoleNone initially, or set by DeclareRole.
	// For the purpose of setting up a session, its intended role in the session matters.

	mockPeer := &MockPeerClient{
		IDValue:        "peerClient456",
		UserIDValue:    "user1",
		SessionIDValue: "session123",
		RoleValue:      "pos",
		// RoleValue for MockPeerClient GetRole(), not directly for session.SetClient role arg
	}

	// Add default .Maybe() expectations for all getter methods
	mockPeer.On("GetID").Maybe().Return(mockPeer.IDValue)
	mockPeer.On("GetRole").Maybe().Return(mockPeer.RoleValue)
	mockPeer.On("GetSessionID").Maybe().Return(mockPeer.SessionIDValue)
	mockPeer.On("GetUserID").Maybe().Return(mockPeer.UserIDValue)

	activeSess := session.NewSession("session123")
	// sourceClient (acting as Provider in NFC context) is the "card" end.
	// mockPeer (acting as Receiver in NFC context) is the "pos" end.
	paired1, errSess1 := activeSess.SetClient(sourceClient, "card")
	assert.NoError(t, errSess1, "SetClient for sourceClient (card) should not error")
	assert.False(t, paired1, "Session should not be paired after first client")

	paired2, errSess2 := activeSess.SetClient(mockPeer, "pos")
	assert.NoError(t, errSess2, "SetClient for mockPeer (pos) should not error")
	assert.True(t, paired2, "Session should be paired after second client")

	h.sessions[activeSess.SessionID] = activeSess

	return apduTestMocks{
		hub:            h,
		sourceClient:   sourceClient,
		mockPeerClient: mockPeer,
		activeSession:  activeSess,
		observedLogs:   observed,
	}
}

// --- Tests for handleAPDUExchange ---

func TestHandleAPDUExchange_PreCondition_NoSessionID(t *testing.T) {
	mocks := setupAPDUTest(t)
	h := mocks.hub
	sourceClient := mocks.sourceClient
	originalConn := sourceClient.conn // Store original conn
	sourceClient.conn = nil           // Set conn to nil to avoid RemoteAddr panic in sendErrorMessage for this specific test
	sourceClient.SessionID = ""       // Override for this test case

	go func() {
		select {
		case msgBytes := <-sourceClient.send:
			var errMsg protocol.ErrorMessage
			err := json.Unmarshal(msgBytes, &errMsg)
			assert.NoError(t, err)
			assert.Equal(t, protocol.MessageTypeError, errMsg.Type)
			assert.Contains(t, errMsg.Message, "您当前不在任何APDU中继会话中")
			assert.Equal(t, protocol.ErrorCodeBadRequest, errMsg.Code)
		case <-time.After(1 * time.Second):
			t.Error("timed out waiting for error message on sourceClient.send")
		}
	}()

	apduMsg := protocol.APDUUpstreamMessage{Type: protocol.MessageTypeAPDUUpstream, APDU: "00A40400"}
	rawMsg, _ := json.Marshal(apduMsg)

	h.handleAPDUExchange(sourceClient, rawMsg, "upstream")
	sourceClient.conn = originalConn // Restore conn

	assert.GreaterOrEqual(t, mocks.observedLogs.FilterMessage("收到APDU消息，但客户端不在任何会话中").Len(), 1, "Expected warning log for no session ID")
}

func TestHandleAPDUExchange_PreCondition_SessionNotFoundInHub(t *testing.T) {
	mocks := setupAPDUTest(t)
	h := mocks.hub
	sourceClient := mocks.sourceClient
	originalConn := sourceClient.conn
	sourceClient.conn = nil // Set conn to nil for this test too
	delete(h.sessions, sourceClient.SessionID)
	originalSessionID := sourceClient.SessionID

	go func() {
		select {
		case msgBytes := <-sourceClient.send:
			var errMsg protocol.ErrorMessage
			err := json.Unmarshal(msgBytes, &errMsg)
			assert.NoError(t, err)
			assert.Equal(t, protocol.MessageTypeError, errMsg.Type)
			assert.Contains(t, errMsg.Message, "您当前的APDU会话已失效，请重新建立连接")
			assert.Equal(t, protocol.ErrorCodeSessionConflict, errMsg.Code)
		case <-time.After(1 * time.Second):
			t.Error("timed out waiting for error message")
		}
	}()

	apduMsg := protocol.APDUUpstreamMessage{Type: protocol.MessageTypeAPDUUpstream, APDU: "00A40400"}
	rawMsg, _ := json.Marshal(apduMsg)

	h.handleAPDUExchange(sourceClient, rawMsg, "upstream")
	assert.Equal(t, "", sourceClient.SessionID, "Client's SessionID should be cleared by the handler")
	sourceClient.conn = originalConn
	sourceClient.SessionID = originalSessionID

	assert.GreaterOrEqual(t, mocks.observedLogs.FilterMessage("收到APDU消息，但客户端关联的会话ID无效或已终止").Len(), 1, "Expected error log for session not found in Hub")
}

func TestHandleAPDUExchange_PreCondition_PeerNotFoundInSession(t *testing.T) {
	mocks := setupAPDUTest(t)
	h := mocks.hub
	sourceClient := mocks.sourceClient
	activeSession := mocks.activeSession
	activeSession.POSEndClient = nil
	originalConn := sourceClient.conn
	sourceClient.conn = nil

	go func() {
		select {
		case msgBytes := <-sourceClient.send:
			var errMsg protocol.ErrorMessage
			err := json.Unmarshal(msgBytes, &errMsg)
			assert.NoError(t, err)
			assert.Equal(t, protocol.MessageTypeError, errMsg.Type)
			assert.Contains(t, errMsg.Message, "APDU发送失败：未能找到您的通信对端")
			assert.Equal(t, protocol.ErrorCodeProviderNotFound, errMsg.Code)
		case <-time.After(1 * time.Second):
			t.Error("timed out waiting for error message")
		}
	}()

	apduMsg := protocol.APDUUpstreamMessage{Type: protocol.MessageTypeAPDUUpstream, APDU: "00A40400"}
	rawMsg, _ := json.Marshal(apduMsg)

	h.handleAPDUExchange(sourceClient, rawMsg, "upstream")
	sourceClient.conn = originalConn

	assert.GreaterOrEqual(t, mocks.observedLogs.FilterMessage("APDU交换：未找到会话中的对端客户端，可能已掉线").Len(), 1, "Expected warning log for peer not found")
}

func TestHandleAPDUExchange_UpdateActivityTime(t *testing.T) {
	mocks := setupAPDUTest(t)
	h := mocks.hub
	sourceClient := mocks.sourceClient
	peerClient := mocks.mockPeerClient
	activeSession := mocks.activeSession
	originalActivityTime := activeSession.LastActivityTime

	peerClient.On("Send", mock.AnythingOfType("[]uint8")).Return(nil).Once()

	apduMsg := protocol.APDUUpstreamMessage{Type: protocol.MessageTypeAPDUUpstream, APDU: "00A40400"}
	rawMsg, _ := json.Marshal(apduMsg)
	time.Sleep(5 * time.Millisecond)

	h.handleAPDUExchange(sourceClient, rawMsg, "upstream")

	peerClient.AssertExpectations(t)
	assert.True(t, activeSession.LastActivityTime.After(originalActivityTime), "Session activity time should be updated")
}

func TestHandleAPDUExchange_ParseUpstreamMessage_Success_And_Forward(t *testing.T) {
	mocks := setupAPDUTest(t)
	h := mocks.hub
	sourceClient := mocks.sourceClient
	peerClient := mocks.mockPeerClient

	apduData := "00A4040007A0000000031010"
	apduMsg := protocol.APDUUpstreamMessage{Type: protocol.MessageTypeAPDUUpstream, APDU: apduData, SessionID: sourceClient.SessionID}
	rawMsg, _ := json.Marshal(apduMsg)

	peerClient.On("Send", rawMsg).Return(nil).Once()

	h.handleAPDUExchange(sourceClient, rawMsg, "upstream")

	peerClient.AssertExpectations(t)
	assert.GreaterOrEqual(t, mocks.observedLogs.FilterMessage("Hub: APDU 消息已成功转发").Len(), 1)

	var attemptLogFound, successLogFound bool
	for _, logEntry := range mocks.observedLogs.All() {
		if logEntry.Message == "AuditEvent" {
			ctxMap := logEntry.ContextMap()
			if eventType, ok := ctxMap["event_type"].(string); ok {
				if eventType == "apdu_relayed_attempt" {
					attemptLogFound = true
					details, _ := ctxMap["details"].(map[string]interface{})
					assert.Equal(t, "upstream", details["direction"])
					assert.Equal(t, float64(len(apduData)), details["length"])
				}
				if eventType == "apdu_relayed_success" {
					successLogFound = true
				}
			}
		}
	}
	assert.True(t, attemptLogFound, "Expected apdu_relayed_attempt audit log")
	assert.True(t, successLogFound, "Expected apdu_relayed_success audit log")
}

func TestHandleAPDUExchange_ParseDownstreamMessage_Success_And_Forward(t *testing.T) {
	mocks := setupAPDUTest(t)
	h := mocks.hub
	sourceClient := mocks.sourceClient // This client is acting as the "card" end sending a downstream APDU
	peerClient := mocks.mockPeerClient // This client is the "pos" end receiving the downstream APDU
	// Ensure the roles in the session are set up as CardEndClient = sourceClient, POSEndClient = peerClient
	// which is done by setupAPDUTest if sourceClient is "card" and mockPeer is "pos"
	// activeSession.CardEndClient = sourceClient (already done by setup if mapping is correct)
	// activeSession.POSEndClient = peerClient (already done by setup if mapping is correct)

	apduData := "9000"
	apduMsg := protocol.APDUDownstreamMessage{Type: protocol.MessageTypeAPDUDownstream, APDU: apduData, SessionID: sourceClient.SessionID}
	rawMsg, _ := json.Marshal(apduMsg)

	// The peerClient (POS) should receive this rawMsg
	peerClient.On("Send", rawMsg).Return(nil).Once()

	h.handleAPDUExchange(sourceClient, rawMsg, "downstream")

	peerClient.AssertExpectations(t)
	assert.GreaterOrEqual(t, mocks.observedLogs.FilterMessage("Hub: APDU 消息已成功转发").Len(), 1)
}

func TestHandleAPDUExchange_ParseMessage_Failure(t *testing.T) {
	mocks := setupAPDUTest(t)
	h := mocks.hub
	sourceClient := mocks.sourceClient
	originalConn := sourceClient.conn
	sourceClient.conn = nil

	rawMsg := []byte("this is not valid json")

	go func() {
		select {
		case msgBytes := <-sourceClient.send:
			var errMsg protocol.ErrorMessage
			err := json.Unmarshal(msgBytes, &errMsg)
			assert.NoError(t, err)
			assert.Equal(t, protocol.MessageTypeError, errMsg.Type)
			assert.Contains(t, errMsg.Message, "无效的APDU消息格式")
			assert.Equal(t, protocol.ErrorCodeBadRequest, errMsg.Code)
		case <-time.After(1 * time.Second):
			t.Error("timed out waiting for error message")
		}
	}()

	h.handleAPDUExchange(sourceClient, rawMsg, "upstream")
	sourceClient.conn = originalConn

	assert.GreaterOrEqual(t, mocks.observedLogs.FilterMessage("APDU交换：反序列化APDUUpstreamMessage失败").Len(), 1, "Expected error log for APDU parsing failure")
}

func TestHandleAPDUExchange_ForwardToPeer_Failure_TerminatesSession(t *testing.T) {
	mocks := setupAPDUTest(t)
	h := mocks.hub
	sourceClient := mocks.sourceClient
	peerClient := mocks.mockPeerClient
	originalConn := sourceClient.conn
	sourceClient.conn = nil // Avoid panic in sendErrorMessage for the first error sent to sourceClient

	sendError := errors.New("simulated send error to peer")
	apduMsg := protocol.APDUUpstreamMessage{Type: protocol.MessageTypeAPDUUpstream, APDU: "00C0000000", SessionID: sourceClient.SessionID}
	rawMsg, _ := json.Marshal(apduMsg)

	peerClient.On("Send", rawMsg).Return(sendError).Once()

	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()
		select {
		case msgBytes := <-sourceClient.send:
			var errMsg protocol.ErrorMessage
			err := json.Unmarshal(msgBytes, &errMsg)
			assert.NoError(t, err)
			assert.Equal(t, protocol.MessageTypeError, errMsg.Type)
			assert.Contains(t, errMsg.Message, "Failed to forward APDU to peer")
			assert.Equal(t, protocol.ErrorCodeInternalError, errMsg.Code)
		case <-time.After(2 * time.Second):
			t.Errorf("timed out waiting for error message to sourceClient")
		}
	}()

	wg.Add(1)
	go func() {
		defer wg.Done()
		select {
		case msgBytes := <-sourceClient.send:
			var termMsg protocol.SessionTerminatedMessage
			err := json.Unmarshal(msgBytes, &termMsg)
			assert.NoError(t, err)
			assert.Equal(t, protocol.MessageTypeSessionTerminated, termMsg.Type)
			assert.Equal(t, "APDU转发失败导致会话终止", termMsg.Reason)
		case <-time.After(2 * time.Second):
			t.Errorf("timed out waiting for SessionTerminatedMessage to sourceClient")
		}
	}()

	peerClient.On("Send", mock.MatchedBy(func(data []byte) bool {
		var termMsg protocol.SessionTerminatedMessage
		if json.Unmarshal(data, &termMsg) == nil {
			return termMsg.Type == protocol.MessageTypeSessionTerminated && termMsg.Reason == "APDU转发失败导致会话终止"
		}
		return false
	})).Return(nil).Once()

	h.handleAPDUExchange(sourceClient, rawMsg, "upstream")
	sourceClient.conn = originalConn
	wg.Wait()

	peerClient.AssertExpectations(t)
	assert.GreaterOrEqual(t, mocks.observedLogs.FilterMessage("Hub: 转发 APDU 消息失败").Len(), 1)
	assert.GreaterOrEqual(t, mocks.observedLogs.FilterMessage("会话已终止").Len(), 1)

	var failureAuditLogFound bool
	for _, logEntry := range mocks.observedLogs.All() {
		if logEntry.Message == "AuditEvent" {
			ctxMap := logEntry.ContextMap()
			if eventType, ok := ctxMap["event_type"].(string); ok && eventType == "apdu_relayed_failure" {
				if details, ok := ctxMap["details"].(map[string]interface{}); ok {
					assert.Contains(t, details["error_message"], "simulated send error to peer")
				}
				failureAuditLogFound = true
				assert.Equal(t, "upstream", ctxMap["direction"], "Direction in apdu_relayed_failure audit log mismatch")
			}
		}
	}
	assert.True(t, failureAuditLogFound, "apdu_relayed_failure audit log not found or incorrect")

	h.providerMutex.RLock()
	_, sessionExists := h.sessions[mocks.activeSession.SessionID]
	h.providerMutex.RUnlock()
	assert.False(t, sessionExists, "Session should have been removed from the hub")
}

// Prometheus mock types and ClientInfoProvider implementation check remain the same.
// init() function remains the same.
// getTestGinContext() remains the same or can be removed if not used.

// ... (rest of the mock Prometheus types and var _ declarations, init function) ...
