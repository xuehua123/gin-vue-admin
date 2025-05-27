package session

import (
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	// "github.com/google/uuid" // Not directly used in session.go, but often in tests
)

// MockClientInfoProvider is a mock implementation of ClientInfoProvider for testing.
type MockClientInfoProvider struct {
	id              string
	userID          string
	sessionID       string
	currentRole     string
	sendShouldFail  bool
	sendCalledCount int
	lastSentMessage []byte
	mu              sync.RWMutex
}

// NewMockClientInfoProvider creates a new mock client info provider.
func NewMockClientInfoProvider(id, userID, sessionID, role string) *MockClientInfoProvider {
	return &MockClientInfoProvider{
		id:          id,
		userID:      userID,
		sessionID:   sessionID,
		currentRole: role,
		mu:          sync.RWMutex{},
	}
}

// Send simulates sending a message.
func (m *MockClientInfoProvider) Send(message []byte) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.sendCalledCount++
	m.lastSentMessage = message
	if m.sendShouldFail {
		return fmt.Errorf("mock send error")
	}
	return nil
}

// GetID returns the client ID.
func (m *MockClientInfoProvider) GetID() string {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.id
}

// GetUserID returns the user ID.
func (m *MockClientInfoProvider) GetUserID() string {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.userID
}

// GetSessionID returns the session ID.
func (m *MockClientInfoProvider) GetSessionID() string {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.sessionID
}

// GetRole implements the ClientInfoProvider interface.
func (m *MockClientInfoProvider) GetRole() string {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.currentRole
}

// SetSessionID sets the session ID (for testing purposes).
func (m *MockClientInfoProvider) SetSessionID(sid string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.sessionID = sid
}

// SetCurrentRole sets the current role (for testing purposes).
func (m *MockClientInfoProvider) SetCurrentRole(role string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.currentRole = role
}

// TestNewSession (Function Point 9.4)
func TestNewSession(t *testing.T) {
	t.Parallel()
	sessionID := "test-session-123"
	s := NewSession(sessionID)

	assert.NotNil(t, s, "NewSession should return a non-nil Session object")
	assert.Equal(t, sessionID, s.SessionID, "SessionID should be set correctly")
	assert.Equal(t, StatusWaitingForPairing, s.Status, "Initial status should be StatusWaitingForPairing")
	assert.NotNil(t, s.LastActivityTime, "LastActivityTime should be initialized")
	assert.WithinDuration(t, time.Now(), s.LastActivityTime, 2*time.Second, "LastActivityTime should be close to time.Now()")
	assert.Nil(t, s.CardEndClient, "CardEndClient should be nil initially")
	assert.Nil(t, s.POSEndClient, "POSEndClient should be nil initially")
	assert.NotNil(t, s.mu, "Mutex should be initialized")
}

// TestSession_UpdateActivityTime (Function Point 9.5)
func TestSession_UpdateActivityTime(t *testing.T) {
	t.Parallel()
	s := NewSession("test-session")
	initialTime := s.LastActivityTime

	time.Sleep(50 * time.Millisecond) // Ensure time progresses enough to see a change
	s.UpdateActivityTime()
	updatedTime := s.LastActivityTime

	assert.True(t, updatedTime.After(initialTime), "LastActivityTime should be updated to a later time")
	assert.WithinDuration(t, time.Now(), updatedTime, 50*time.Millisecond, "Updated LastActivityTime should be close to time.Now()")
}

// TestSession_IsInactive (Function Point 9.6)
func TestSession_IsInactive(t *testing.T) {
	t.Parallel()
	timeoutDuration := 100 * time.Millisecond

	t.Run("PairedAndNotTimedOut", func(t *testing.T) {
		t.Parallel()
		s := NewSession("paired-active")
		s.Status = StatusPaired
		s.LastActivityTime = time.Now()
		assert.False(t, s.IsInactive(timeoutDuration), "Session paired and not timed out should not be inactive")
	})

	t.Run("PairedAndTimedOut", func(t *testing.T) {
		t.Parallel()
		s := NewSession("paired-inactive")
		s.Status = StatusPaired
		s.LastActivityTime = time.Now().Add(-(timeoutDuration + 50*time.Millisecond)) // Make it timed out
		assert.True(t, s.IsInactive(timeoutDuration), "Session paired and timed out should be inactive")
	})

	t.Run("WaitingForPairingNotTimedOut", func(t *testing.T) {
		t.Parallel()
		s := NewSession("waiting-active")
		s.Status = StatusWaitingForPairing
		s.LastActivityTime = time.Now()
		assert.False(t, s.IsInactive(timeoutDuration), "Session waiting for pairing (regardless of time) should not be considered inactive by this check")
	})

	t.Run("WaitingForPairingButOld", func(t *testing.T) {
		t.Parallel()
		s := NewSession("waiting-old")
		s.Status = StatusWaitingForPairing
		s.LastActivityTime = time.Now().Add(-(timeoutDuration + 50*time.Millisecond))
		assert.False(t, s.IsInactive(timeoutDuration), "Session waiting for pairing (even if old) should not be considered inactive by this check")
	})

	t.Run("Terminated", func(t *testing.T) {
		t.Parallel()
		s := NewSession("terminated")
		s.Status = StatusTerminated
		s.LastActivityTime = time.Now().Add(-(timeoutDuration + 50*time.Millisecond)) // Older than timeout
		assert.False(t, s.IsInactive(timeoutDuration), "Terminated session should not be considered inactive by this check")
	})
}

// TestSession_Terminate (Function Point 9.7)
func TestSession_Terminate(t *testing.T) {
	t.Parallel()
	s := NewSession("test-session-terminate")
	s.Status = StatusPaired // Start from a non-terminated state

	s.Terminate()
	assert.Equal(t, StatusTerminated, s.Status, "Status should be StatusTerminated after calling Terminate()")

	// Calling terminate again should not change the status
	s.Terminate()
	assert.Equal(t, StatusTerminated, s.Status, "Status should remain StatusTerminated after calling Terminate() again")
}

// TestSession_SetClient (Function Point 9.8)
func TestSession_SetClient(t *testing.T) {
	t.Parallel()
	mockClient1 := NewMockClientInfoProvider("client1", "user1", "", RoleCardEnd)
	mockClient2 := NewMockClientInfoProvider("client2", "user1", "", RolePOSEnd)
	mockClient3 := NewMockClientInfoProvider("client3", "user1", "", RoleCardEnd)

	t.Run("SetClientOnTerminatedSession", func(t *testing.T) {
		t.Parallel()
		s := NewSession("terminated-session")
		s.Terminate()
		paired, err := s.SetClient(mockClient1, RoleCardEnd)
		assert.False(t, paired, "Should not be paired")
		assert.Error(t, err, "Should return an error when setting client on a terminated session")
		assert.IsType(t, &SessionError{}, err, "Error should be of type SessionError")
		assert.Equal(t, "会话已终止，无法加入", err.Error())
	})

	t.Run("SetFirstClient_CardEnd", func(t *testing.T) {
		t.Parallel()
		s := NewSession("set-client-card")
		paired, err := s.SetClient(mockClient1, RoleCardEnd)
		assert.NoError(t, err, "Setting first client (CardEnd) should not return an error")
		assert.False(t, paired, "Session should not be paired after setting only one client")
		assert.Equal(t, StatusWaitingForPairing, s.Status, "Status should remain WaitingForPairing")
		assert.Equal(t, mockClient1, s.CardEndClient, "CardEndClient should be set")
		assert.Nil(t, s.POSEndClient, "POSEndClient should remain nil")
	})

	t.Run("SetFirstClient_POSEnd", func(t *testing.T) {
		t.Parallel()
		s := NewSession("set-client-pos")
		paired, err := s.SetClient(mockClient2, RolePOSEnd)
		assert.NoError(t, err, "Setting first client (POSEnd) should not return an error")
		assert.False(t, paired, "Session should not be paired after setting only one client")
		assert.Equal(t, StatusWaitingForPairing, s.Status, "Status should remain WaitingForPairing")
		assert.Equal(t, mockClient2, s.POSEndClient, "POSEndClient should be set")
		assert.Nil(t, s.CardEndClient, "CardEndClient should remain nil")
	})

	t.Run("SetSecondClient_PairSuccess_CardThenPOS", func(t *testing.T) {
		t.Parallel()
		s := NewSession("pair-card-pos")
		s.SetClient(mockClient1, RoleCardEnd) // First client
		initialActivityTime := s.LastActivityTime
		time.Sleep(10 * time.Millisecond) // ensure activity time will be different

		paired, err := s.SetClient(mockClient2, RolePOSEnd) // Second client
		assert.NoError(t, err, "Setting second client (POSEnd) should not return an error")
		assert.True(t, paired, "Session should be paired after setting both clients")
		assert.Equal(t, StatusPaired, s.Status, "Status should change to Paired")
		assert.Equal(t, mockClient1, s.CardEndClient, "CardEndClient should remain set")
		assert.Equal(t, mockClient2, s.POSEndClient, "POSEndClient should be set")
		assert.True(t, s.LastActivityTime.After(initialActivityTime), "LastActivityTime should be updated on pairing")
	})

	t.Run("SetSecondClient_PairSuccess_POSThenCard", func(t *testing.T) {
		t.Parallel()
		s := NewSession("pair-pos-card")
		s.SetClient(mockClient2, RolePOSEnd) // First client
		initialActivityTime := s.LastActivityTime
		time.Sleep(10 * time.Millisecond)

		paired, err := s.SetClient(mockClient1, RoleCardEnd) // Second client
		assert.NoError(t, err, "Setting second client (CardEnd) should not return an error")
		assert.True(t, paired, "Session should be paired after setting both clients")
		assert.Equal(t, StatusPaired, s.Status, "Status should change to Paired")
		assert.Equal(t, mockClient1, s.CardEndClient, "CardEndClient should be set")
		assert.Equal(t, mockClient2, s.POSEndClient, "POSEndClient should remain set")
		assert.True(t, s.LastActivityTime.After(initialActivityTime), "LastActivityTime should be updated on pairing")
	})

	t.Run("SetClient_RoleAlreadyOccupied_CardEnd", func(t *testing.T) {
		t.Parallel()
		s := NewSession("role-occupied-card")
		s.SetClient(mockClient1, RoleCardEnd)
		paired, err := s.SetClient(mockClient3, RoleCardEnd) // Attempt to set another CardEnd
		assert.Error(t, err, "Should return an error if CardEnd role is already occupied")
		assert.IsType(t, &SessionError{}, err)
		assert.Equal(t, fmt.Sprintf("角色 '%s' 已被占用", RoleCardEnd), err.Error())
		assert.False(t, paired, "Session should not be paired")
		assert.Equal(t, mockClient1, s.CardEndClient, "Original CardEndClient should remain")
	})

	t.Run("SetClient_RoleAlreadyOccupied_POSEnd", func(t *testing.T) {
		t.Parallel()
		s := NewSession("role-occupied-pos")
		s.SetClient(mockClient2, RolePOSEnd)
		paired, err := s.SetClient(mockClient3, RolePOSEnd) // Attempt to set another POSEnd (using client3 as a distinct mock)
		assert.Error(t, err, "Should return an error if POSEnd role is already occupied")
		assert.IsType(t, &SessionError{}, err)
		assert.Equal(t, fmt.Sprintf("角色 '%s' 已被占用", RolePOSEnd), err.Error())
		assert.False(t, paired, "Session should not be paired")
		assert.Equal(t, mockClient2, s.POSEndClient, "Original POSEndClient should remain")
	})

	t.Run("SetClient_InvalidRole", func(t *testing.T) {
		t.Parallel()
		s := NewSession("invalid-role")
		invalidRole := "invalid_role_string"
		paired, err := s.SetClient(mockClient1, invalidRole)
		assert.Error(t, err, "Should return an error for an invalid role string")
		assert.IsType(t, &SessionError{}, err)
		assert.Equal(t, fmt.Sprintf("无效的客户端角色: %s", invalidRole), err.Error())
		assert.False(t, paired, "Session should not be paired")
		assert.Nil(t, s.CardEndClient, "CardEndClient should not be set with invalid role")
		assert.Nil(t, s.POSEndClient, "POSEndClient should not be set with invalid role")
	})

	t.Run("SetClient_NilClient", func(t *testing.T) {
		t.Parallel()
		s := NewSession("nil-client")
		paired, err := s.SetClient(nil, RoleCardEnd)
		assert.Error(t, err, "Should return an error if client is nil")
		assert.IsType(t, &SessionError{}, err)
		assert.Equal(t, "客户端实例不能为nil", err.Error())
		assert.False(t, paired)
	})
}

// TestSession_RemoveClient (Function Point 9.9)
func TestSession_RemoveClient(t *testing.T) {
	t.Parallel()
	mockClientCard := NewMockClientInfoProvider("cardClient", "user1", "", RoleCardEnd)
	mockClientPOS := NewMockClientInfoProvider("posClient", "user1", "", RolePOSEnd)
	mockClientOther := NewMockClientInfoProvider("otherClient", "user2", "", "")

	t.Run("RemoveClientOnTerminatedSession", func(t *testing.T) {
		t.Parallel()
		s := NewSession("terminated-remove")
		s.SetClient(mockClientCard, RoleCardEnd)
		s.Terminate()
		s.RemoveClient(mockClientCard) // Should do nothing
		assert.Equal(t, StatusTerminated, s.Status, "Status should remain Terminated")
		// Clients might still be referenced if RemoveClient bails early, which is fine.
	})

	t.Run("RemoveCardEndClientFromPairedSession", func(t *testing.T) {
		t.Parallel()
		s := NewSession("remove-card-from-paired")
		s.SetClient(mockClientCard, RoleCardEnd)
		s.SetClient(mockClientPOS, RolePOSEnd) // Paired

		s.RemoveClient(mockClientCard)
		assert.Nil(t, s.CardEndClient, "CardEndClient should be nil after removal")
		assert.Equal(t, mockClientPOS, s.POSEndClient, "POSEndClient should remain")
		assert.Equal(t, StatusWaitingForPairing, s.Status, "Status should revert to WaitingForPairing")
	})

	t.Run("RemovePOSEndClientFromPairedSession", func(t *testing.T) {
		t.Parallel()
		s := NewSession("remove-pos-from-paired")
		s.SetClient(mockClientCard, RoleCardEnd)
		s.SetClient(mockClientPOS, RolePOSEnd) // Paired

		s.RemoveClient(mockClientPOS)
		assert.Nil(t, s.POSEndClient, "POSEndClient should be nil after removal")
		assert.Equal(t, mockClientCard, s.CardEndClient, "CardEndClient should remain")
		assert.Equal(t, StatusWaitingForPairing, s.Status, "Status should revert to WaitingForPairing")
	})

	t.Run("RemoveClientNotActiveInSession", func(t *testing.T) {
		t.Parallel()
		s := NewSession("remove-other-client")
		s.SetClient(mockClientCard, RoleCardEnd) // Only CardEnd is set
		initialStatus := s.Status

		s.RemoveClient(mockClientOther) // mockClientOther is not in the session
		assert.Equal(t, mockClientCard, s.CardEndClient, "CardEndClient should not be affected")
		assert.Nil(t, s.POSEndClient, "POSEndClient should still be nil")
		assert.Equal(t, initialStatus, s.Status, "Status should not change")
	})

	t.Run("RemoveTheOnlyClientFromWaitingSession_Card", func(t *testing.T) {
		t.Parallel()
		s := NewSession("remove-only-card")
		s.SetClient(mockClientCard, RoleCardEnd)
		s.RemoveClient(mockClientCard)
		assert.Nil(t, s.CardEndClient, "CardEndClient should be nil")
		assert.Equal(t, StatusWaitingForPairing, s.Status, "Status should remain WaitingForPairing")
	})

	t.Run("RemoveTheOnlyClientFromWaitingSession_POS", func(t *testing.T) {
		t.Parallel()
		s := NewSession("remove-only-pos")
		s.SetClient(mockClientPOS, RolePOSEnd)
		s.RemoveClient(mockClientPOS)
		assert.Nil(t, s.POSEndClient, "POSEndClient should be nil")
		assert.Equal(t, StatusWaitingForPairing, s.Status, "Status should remain WaitingForPairing")
	})

	t.Run("RemoveNilClient", func(t *testing.T) {
		t.Parallel()
		s := NewSession("remove-nil")
		s.SetClient(mockClientCard, RoleCardEnd)
		s.RemoveClient(nil) // Should not panic and ideally do nothing
		assert.NotNil(t, s.CardEndClient, "CardEndClient should remain after trying to remove nil")
	})
}

// TestSession_GetPeer (Function Point 9.10)
func TestSession_GetPeer(t *testing.T) {
	t.Parallel()
	mockClientCard := NewMockClientInfoProvider("cardClient", "user1", "", RoleCardEnd)
	mockClientPOS := NewMockClientInfoProvider("posClient", "user1", "", RolePOSEnd)
	mockClientOther := NewMockClientInfoProvider("otherClient", "user2", "", "") // Not in session

	t.Run("GetPeerWhenNotPaired_Waiting", func(t *testing.T) {
		t.Parallel()
		s := NewSession("getpeer-waiting")
		s.SetClient(mockClientCard, RoleCardEnd)
		assert.Nil(t, s.GetPeer(mockClientCard), "Peer should be nil if session is not fully paired (StatusWaitingForPairing)")
		assert.Nil(t, s.GetPeer(mockClientPOS), "Peer for a non-participant should be nil")
	})

	t.Run("GetPeerWhenNotPaired_Terminated", func(t *testing.T) {
		t.Parallel()
		s := NewSession("getpeer-terminated")
		s.SetClient(mockClientCard, RoleCardEnd)
		s.SetClient(mockClientPOS, RolePOSEnd)
		s.Terminate()
		assert.Nil(t, s.GetPeer(mockClientCard), "Peer should be nil if session is terminated")
	})

	t.Run("GetPeer_CardEndClient_Success", func(t *testing.T) {
		t.Parallel()
		s := NewSession("getpeer-card-success")
		s.SetClient(mockClientCard, RoleCardEnd)
		s.SetClient(mockClientPOS, RolePOSEnd) // Paired
		peer := s.GetPeer(mockClientCard)
		assert.Equal(t, mockClientPOS, peer, "Peer of CardEndClient should be POSEndClient")
	})

	t.Run("GetPeer_POSEndClient_Success", func(t *testing.T) {
		t.Parallel()
		s := NewSession("getpeer-pos-success")
		s.SetClient(mockClientCard, RoleCardEnd)
		s.SetClient(mockClientPOS, RolePOSEnd) // Paired
		peer := s.GetPeer(mockClientPOS)
		assert.Equal(t, mockClientCard, peer, "Peer of POSEndClient should be CardEndClient")
	})

	t.Run("GetPeer_ClientNotInSession", func(t *testing.T) {
		t.Parallel()
		s := NewSession("getpeer-other")
		s.SetClient(mockClientCard, RoleCardEnd)
		s.SetClient(mockClientPOS, RolePOSEnd) // Paired
		assert.Nil(t, s.GetPeer(mockClientOther), "Peer for a client not in the session should be nil")
	})

	t.Run("GetPeer_NilClient", func(t *testing.T) {
		t.Parallel()
		s := NewSession("getpeer-nil")
		s.SetClient(mockClientCard, RoleCardEnd)
		s.SetClient(mockClientPOS, RolePOSEnd)
		assert.Nil(t, s.GetPeer(nil), "GetPeer with nil client should return nil")
	})
}

// TestSessionError (Function Point 9.11)
func TestSessionError(t *testing.T) {
	t.Parallel()
	errorMessage := "this is a test session error"
	err := &SessionError{Message: errorMessage}
	assert.Equal(t, errorMessage, err.Error(), "SessionError's Error() method should return the Message field")
}

// TestSession_Concurrency (Function Point 9.12)
func TestSession_Concurrency(t *testing.T) {
	// This test is not run in t.Parallel() itself, but its sub-goroutines will run concurrently.
	// Run this test with the -race flag to detect race conditions: go test -race ./...
	s := NewSession("concurrent-session")
	mockC1 := NewMockClientInfoProvider("c1", "u1", "", RoleCardEnd)
	mockC2 := NewMockClientInfoProvider("c2", "u1", "", RolePOSEnd)
	mockC3 := NewMockClientInfoProvider("c3", "u2", "", RoleCardEnd)

	var wg sync.WaitGroup
	numGoroutines := 50

	// Mix of SetClient, RemoveClient, GetPeer, UpdateActivityTime, IsInactive, Terminate
	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()
			switch idx % 7 {
			case 0:
				s.SetClient(mockC1, RoleCardEnd)
			case 1:
				s.SetClient(mockC2, RolePOSEnd)
			case 2:
				s.RemoveClient(mockC1)
			case 3:
				s.GetPeer(mockC2)
			case 4:
				s.UpdateActivityTime()
			case 5:
				s.IsInactive(10 * time.Millisecond)
			case 6:
				if idx == numGoroutines-1 { // Only terminate once at the end
					s.Terminate()
				} else {
					s.SetClient(mockC3, RoleCardEnd) // Another client
				}
			}
		}(i)
	}
	wg.Wait()

	// Basic assertions on final state (less important than race detection)
	// The final state is non-deterministic due to concurrency, so we only check for non-nil.
	assert.NotNil(t, s, "Session should still exist")
	// It's hard to assert a specific status due to the race, but Terminate() should eventually set it.
	// If the last operation was Terminate(), it should be Terminated.
	// For a robust check of final state, specific ordered concurrency tests are needed.
	// This test primarily serves to run with -race.
}
