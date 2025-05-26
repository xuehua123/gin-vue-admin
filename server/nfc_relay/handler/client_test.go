package handler

import (
	"errors"
	"io"
	"net"
	"sync"
	"testing"
	"time"

	"github.com/flipped-aurora/gin-vue-admin/server/config"
	"github.com/flipped-aurora/gin-vue-admin/server/global"
	"github.com/flipped-aurora/gin-vue-admin/server/nfc_relay/protocol"
	"github.com/gorilla/websocket"
	"github.com/stretchr/testify/assert"
	"go.uber.org/zap"
)

// WebsocketConnection interface (ensure this or similaire is used in client.go)
// Duplicating here for clarity in thought process, but should be defined once, accessible to client.go
// type WebsocketConnection interface {
// 	Close() error
// 	SetReadLimit(limit int64)
// 	SetReadDeadline(t time.Time) error
// 	SetWriteDeadline(t time.Time) error
// 	SetPongHandler(h func(string) error)
// 	ReadMessage() (messageType int, p []byte, err error)
// 	NextWriter(messageType int) (io.WriteCloser, error)
// 	WriteMessage(messageType int, data []byte) error
// 	RemoteAddr() net.Addr
// }

// mockWsConnection 模拟 websocket.Conn (implements WebsocketConnection)
type mockWsConnection struct {
	mu                   sync.Mutex
	readMessageFunc      func() (int, []byte, error)
	writeMessageFunc     func(messageType int, data []byte) error
	nextWriterFunc       func(messageType int) (io.WriteCloser, error)
	closeFunc            func() error
	setReadDeadlineFunc  func(t time.Time) error
	setWriteDeadlineFunc func(t time.Time) error
	setPongHandlerFunc   func(h func(string) error)
	setReadLimitFunc     func(limit int64)
	remoteAddrFunc       func() net.Addr

	writeMessageCalledWith [][]byte
	_closeCalledLock       sync.Mutex    // To protect manual setting if still used alongside channel
	_closeCalled           bool          // Private flag if we still want to check it directly sometimes
	closeSignal            chan struct{} // For synchronizing test assertion
	nextWriterCalled       bool
}

func newMockWsConnection() *mockWsConnection { // Helper to create mock with initialized channel
	return &mockWsConnection{
		closeSignal: make(chan struct{}, 1),
	}
}

func (m *mockWsConnection) ReadMessage() (messageType int, p []byte, err error) {
	if m.readMessageFunc != nil {
		return m.readMessageFunc()
	}
	return 0, nil, io.EOF
}

func (m *mockWsConnection) WriteMessage(messageType int, data []byte) error {
	m.mu.Lock()
	m.writeMessageCalledWith = append(m.writeMessageCalledWith, data)
	m.mu.Unlock()
	if m.writeMessageFunc != nil {
		return m.writeMessageFunc(messageType, data)
	}
	return nil
}

func (m *mockWsConnection) NextWriter(messageType int) (io.WriteCloser, error) {
	m.mu.Lock()
	m.nextWriterCalled = true
	m.mu.Unlock()
	if m.nextWriterFunc != nil {
		return m.nextWriterFunc(messageType)
	}
	return &mockWriteCloser{}, nil
}

func (m *mockWsConnection) Close() error {
	m._closeCalledLock.Lock()
	m._closeCalled = true
	m._closeCalledLock.Unlock()

	var err error
	if m.closeFunc != nil {
		err = m.closeFunc()
	}
	// Signal that Close was called, non-blocking send due to buffered channel
	select {
	case m.closeSignal <- struct{}{}:
	default: // Avoid blocking if channel is full or not listened to (should not happen with buffer 1)
	}
	return err
}

func (m *mockWsConnection) isCloseCalled() bool { // Getter if direct check is still desired
	m._closeCalledLock.Lock()
	defer m._closeCalledLock.Unlock()
	return m._closeCalled
}

func (m *mockWsConnection) SetReadDeadline(t time.Time) error {
	if m.setReadDeadlineFunc != nil {
		return m.setReadDeadlineFunc(t)
	}
	return nil
}

func (m *mockWsConnection) SetWriteDeadline(t time.Time) error {
	if m.setWriteDeadlineFunc != nil {
		return m.setWriteDeadlineFunc(t)
	}
	return nil
}

func (m *mockWsConnection) SetPongHandler(h func(string) error) {
	if m.setPongHandlerFunc != nil {
		m.setPongHandlerFunc(h)
	}
}

func (m *mockWsConnection) SetReadLimit(limit int64) {
	if m.setReadLimitFunc != nil {
		m.setReadLimitFunc(limit)
	}
}

func (m *mockWsConnection) RemoteAddr() net.Addr {
	if m.remoteAddrFunc != nil {
		return m.remoteAddrFunc()
	}
	return &mockAddr{}
}

// mockWriteCloser 模拟 io.WriteCloser
type mockWriteCloser struct {
	writeFunc func(p []byte) (n int, err error)
	closeFunc func() error
}

func (mwc *mockWriteCloser) Write(p []byte) (n int, err error) {
	if mwc.writeFunc != nil {
		return mwc.writeFunc(p)
	}
	return len(p), nil
}

func (mwc *mockWriteCloser) Close() error {
	if mwc.closeFunc != nil {
		return mwc.closeFunc()
	}
	return nil
}

// mockAddr 模拟 net.Addr
type mockAddr struct{}

func (m *mockAddr) Network() string { return "testNetwork" }
func (m *mockAddr) String() string  { return "testAddr" }

// mockHub 模拟 Hub
type mockHub struct {
	unregister     chan *Client
	processMessage chan ProcessableMessage
	// 可以添加其他需要的方法或字段
}

func newMockHub() *mockHub {
	return &mockHub{
		unregister:     make(chan *Client, 1),            // Buffered to avoid blocking in simple tests
		processMessage: make(chan ProcessableMessage, 1), // Buffered
	}
}

// setupGlobalConfigForTests 初始化测试所需的全局配置
// 注意：这会修改全局变量，测试完成后可能需要恢复或确保测试隔离
func setupGlobalConfigForTests() {
	// global.GVA_CONFIG is config.Server (struct), not *config.Server (pointer)
	// Initialize its fields directly.
	global.GVA_CONFIG.NfcRelay = config.NfcRelay{
		WebsocketPongWaitSec:     60,
		WebsocketWriteWaitSec:    10,
		WebsocketMaxMessageBytes: 2048,
		// HubCheckIntervalSec: 60, // Set other necessary fields if client.go depends on them
		// SessionInactiveTimeoutSec: 300,
	}
	// Initialize other parts of GVA_CONFIG if necessary for the tests, e.g.:
	// global.GVA_CONFIG.System.UseMultipoint = false

	if global.GVA_LOG == nil {
		logger := zap.NewNop()
		global.GVA_LOG = logger
	}
}

// membutuhkanInisialisasiNfcRelay 是一个辅助函数，用于判断是否需要初始化NfcRelay配置
// 这只是一个示例，具体逻辑可能需要根据实际情况调整
func membutuhkanInisialisasiNfcRelay(nfcConfig config.NfcRelay) bool {
	// 如果所有关键NFC中继配置都是其类型的零值，则可能需要初始化
	return nfcConfig.WebsocketPongWaitSec == 0 &&
		nfcConfig.WebsocketWriteWaitSec == 0 &&
		nfcConfig.WebsocketMaxMessageBytes == 0
}

// TestNewClient 测试 NewClient 构造函数
func TestNewClient(t *testing.T) {
	setupGlobalConfigForTests()
	mockConn := &mockWsConnection{}
	realHub := NewHub()

	client := NewClient(realHub, mockConn)

	assert.NotNil(t, client, "Client should not be nil")
	assert.Equal(t, realHub, client.hub, "Client hub should be the real hub")
	assert.Equal(t, mockConn, client.conn, "Client conn should be the mock connection")
	assert.NotNil(t, client.send, "Client send channel should be initialized")
	assert.Equal(t, 256, cap(client.send), "Client send channel capacity should be 256")
	assert.NotEmpty(t, client.ID, "Client ID should not be empty")
	assert.False(t, client.Authenticated, "Client Authenticated should be false initially")
	assert.Equal(t, protocol.RoleType(""), client.CurrentRole, "Client CurrentRole should be empty initially")
}

// TestClient_Getters 测试 Client 的 Getter 方法
func TestClient_Getters(t *testing.T) {
	setupGlobalConfigForTests()
	client := &Client{
		ID:          "test-id",
		SessionID:   "test-session-id",
		UserID:      "test-user-id",
		CurrentRole: protocol.RoleProvider,
	}

	assert.Equal(t, "test-id", client.GetID())
	assert.Equal(t, "test-session-id", client.GetSessionID())
	assert.Equal(t, "test-user-id", client.GetUserID())
	assert.Equal(t, protocol.RoleProvider, client.GetCurrentRole())
	assert.Equal(t, string(protocol.RoleProvider), client.GetRole()) // GetRole() 现在也应该返回 CurrentRole
}

// TestClient_Send_Success 测试成功发送消息到 send 通道
func TestClient_Send_Success(t *testing.T) {
	setupGlobalConfigForTests()
	client := &Client{
		send: make(chan []byte, 1), // Buffer of 1 for the test
		ID:   "test-client",
	}
	msg := []byte("hello")

	err := client.Send(msg)

	assert.NoError(t, err, "Send should not return an error")
	select {
	case sentMsg := <-client.send:
		assert.Equal(t, msg, sentMsg, "Message sent should be the same as received from channel")
	default:
		t.Fatal("Message was not sent to the channel")
	}
}

// TestClient_Send_ChannelFull 测试当 send 通道满时发送消息
func TestClient_Send_ChannelFull(t *testing.T) {
	setupGlobalConfigForTests()
	client := &Client{
		send: make(chan []byte, 1), // Buffer of 1
		ID:   "test-client-full",
	}
	msg1 := []byte("message1")
	msg2 := []byte("message2")

	// Fill the channel
	client.send <- msg1

	// Try to send another message, expecting an error
	err := client.Send(msg2)

	assert.Error(t, err, "Send should return an error when channel is full")
	assert.Equal(t, "failed to send message to client: channel full", err.Error())

	// Ensure original message is still there and no new message was added
	assert.Len(t, client.send, 1, "Channel should still have one message")
	assert.Equal(t, msg1, <-client.send, "The original message should be in the channel")
}

// TestClient_Send_ChannelClosed 测试当 send 通道关闭时发送消息
func TestClient_Send_ChannelClosed(t *testing.T) {
	setupGlobalConfigForTests()
	client := &Client{
		send: make(chan []byte, 1),
		ID:   "test-client-closed",
	}
	msg := []byte("hello")

	close(client.send)

	err := client.Send(msg)

	assert.Error(t, err, "Send should return an error when channel is closed")
	assert.Equal(t, "failed to send message to client: channel is closed", err.Error())
}

// TestClient_ReadPump_SuccessfulReadTextMessage 测试 readPump 成功读取文本消息
func TestClient_ReadPump_SuccessfulReadTextMessage(t *testing.T) {
	setupGlobalConfigForTests()
	mockConn := newMockWsConnection() // +++ Use constructor
	realHub := NewHub()
	client := NewClient(realHub, mockConn)

	expectedMessage := []byte("test message")
	var readCount int
	mockConn.readMessageFunc = func() (int, []byte, error) {
		if readCount == 0 {
			readCount++
			return websocket.TextMessage, expectedMessage, nil
		}
		return 0, nil, io.EOF
	}

	pongHandlerSet, readDeadlineSet, readLimitSet := false, false, false
	mockConn.setPongHandlerFunc = func(h func(string) error) { pongHandlerSet = true }
	mockConn.setReadDeadlineFunc = func(tm time.Time) error { readDeadlineSet = true; return nil }
	mockConn.setReadLimitFunc = func(limit int64) { readLimitSet = true }

	go client.readPump()

	select {
	case procMsg := <-realHub.processMessage:
		assert.Equal(t, client, procMsg.Client)
		assert.Equal(t, expectedMessage, procMsg.RawMessage)
	case <-time.After(1 * time.Second):
		t.Fatal("Timeout waiting for message to be processed by hub")
	}

	select {
	case unregClient := <-realHub.unregister:
		assert.Equal(t, client, unregClient)
	case <-time.After(1 * time.Second):
		t.Fatal("Timeout waiting for client to unregister")
	}

	// +++ Wait for Close signal
	select {
	case <-mockConn.closeSignal:
		// Successfully closed
	case <-time.After(200 * time.Millisecond): // Increased timeout slightly for close signal
		t.Fatal("Timeout waiting for mockConn.Close to be signaled")
	}

	assert.True(t, pongHandlerSet)
	assert.True(t, readDeadlineSet)
	assert.True(t, readLimitSet)
	assert.True(t, mockConn.isCloseCalled()) // This assertion can now be more reliably true
}

// TestClient_ReadPump_PongMessageReceived 测试 readPump 收到 Pong 消息
func TestClient_ReadPump_PongMessageReceived(t *testing.T) {
	setupGlobalConfigForTests()
	mockConn := newMockWsConnection() // +++ Use constructor
	realHub := NewHub()
	client := NewClient(realHub, mockConn)

	var pongHandler func(string) error
	mockConn.setPongHandlerFunc = func(h func(string) error) { pongHandler = h }

	readDeadlineUpdatedCount := 0
	mockConn.setReadDeadlineFunc = func(tm time.Time) error {
		readDeadlineUpdatedCount++
		return nil
	}
	mockConn.readMessageFunc = func() (int, []byte, error) {
		time.Sleep(50 * time.Millisecond)
		return 0, nil, io.EOF
	}

	go client.readPump()
	time.Sleep(100 * time.Millisecond) // Allow time for pong handler to be set

	assert.NotNil(t, pongHandler, "Pong handler should be set")
	if pongHandler != nil {
		err := pongHandler("test pong data")
		assert.NoError(t, err)
	}

	select {
	case <-realHub.unregister:
	case <-time.After(1 * time.Second):
		t.Fatal("Timeout waiting for client to unregister after pong")
	}

	// +++ Wait for Close signal
	select {
	case <-mockConn.closeSignal:
		// Successfully closed
	case <-time.After(200 * time.Millisecond):
		t.Fatal("Timeout waiting for mockConn.Close to be signaled after pong")
	}

	assert.GreaterOrEqual(t, readDeadlineUpdatedCount, 1, "SetReadDeadline should have been called at least once (initial or after pong)")
	assert.True(t, mockConn.isCloseCalled())
}

// TestClient_ReadPump_ReadError 测试 readPump 读取错误导致注销
func TestClient_ReadPump_ReadError(t *testing.T) {
	setupGlobalConfigForTests()
	mockConn := &mockWsConnection{}
	realHub := NewHub()
	client := NewClient(realHub, mockConn)

	mockConn.readMessageFunc = func() (int, []byte, error) { return 0, nil, errors.New("simulated read error") }

	go client.readPump()

	select {
	case unregClient := <-realHub.unregister:
		assert.Equal(t, client, unregClient)
	case <-time.After(1 * time.Second):
		t.Fatal("Timeout waiting for client to unregister on read error")
	}
	assert.True(t, mockConn.isCloseCalled())
}

// TestClient_ReadPump_ClientClosesConnection 测试客户端主动关闭连接
func TestClient_ReadPump_ClientClosesConnection(t *testing.T) {
	setupGlobalConfigForTests()
	mockConn := &mockWsConnection{}
	realHub := NewHub()
	client := NewClient(realHub, mockConn)
	mockConn.readMessageFunc = func() (int, []byte, error) {
		return 0, nil, &websocket.CloseError{Code: websocket.CloseGoingAway, Text: "Client disconnected"}
	}
	go client.readPump()
	<-realHub.unregister
	assert.True(t, mockConn.isCloseCalled())
}

// TestClient_ReadPump_UnexpectedCloseError 测试意外关闭错误
func TestClient_ReadPump_UnexpectedCloseError(t *testing.T) {
	setupGlobalConfigForTests()
	mockConn := &mockWsConnection{}
	realHub := NewHub()
	client := NewClient(realHub, mockConn)
	mockConn.readMessageFunc = func() (int, []byte, error) {
		return 0, nil, &websocket.CloseError{Code: websocket.CloseAbnormalClosure, Text: "Something bad"}
	}
	go client.readPump()
	<-realHub.unregister
	assert.True(t, mockConn.isCloseCalled())
}

// TestClient_ReadPump_BinaryMessageReceived 测试收到二进制消息
func TestClient_ReadPump_BinaryMessageReceived(t *testing.T) {
	setupGlobalConfigForTests()
	mockConn := newMockWsConnection() // +++ Use constructor
	realHub := NewHub()
	client := NewClient(realHub, mockConn)

	binaryMessage := []byte{0x01, 0x02, 0x03}
	var readCount int
	mockConn.readMessageFunc = func() (int, []byte, error) {
		if readCount == 0 {
			readCount++
			return websocket.BinaryMessage, binaryMessage, nil
		}
		return 0, nil, io.EOF
	}
	go client.readPump()
	select {
	case <-realHub.processMessage:
		t.Fatal("Binary message should not be sent to hub.processMessage")
	case <-time.After(100 * time.Millisecond):
	}

	select {
	case <-realHub.unregister:
	case <-time.After(1 * time.Second):
		t.Fatal("Timeout waiting for client to unregister after binary message")
	}

	// +++ Wait for Close signal
	select {
	case <-mockConn.closeSignal:
		// Successfully closed
	case <-time.After(200 * time.Millisecond):
		t.Fatal("Timeout waiting for mockConn.Close to be signaled after binary message")
	}

	assert.True(t, mockConn.isCloseCalled())
}

// TestClient_ReadPump_ConfigDefaults_MaxMessageSizeZero 测试 readPump 在 WebsocketMaxMessageBytes 为0时使用默认值
func TestClient_ReadPump_ConfigDefaults_MaxMessageSizeZero(t *testing.T) {
	setupGlobalConfigForTests()
	originalMaxMessage := global.GVA_CONFIG.NfcRelay.WebsocketMaxMessageBytes
	global.GVA_CONFIG.NfcRelay.WebsocketMaxMessageBytes = 0
	defer func() { global.GVA_CONFIG.NfcRelay.WebsocketMaxMessageBytes = originalMaxMessage }()

	mockConn := &mockWsConnection{}
	realHub := NewHub()
	client := NewClient(realHub, mockConn)

	var actualReadLimit int64
	mockConn.setReadLimitFunc = func(limit int64) { actualReadLimit = limit }
	mockConn.readMessageFunc = func() (int, []byte, error) { return 0, nil, io.EOF }
	go client.readPump()
	<-realHub.unregister
	assert.Equal(t, int64(2048), actualReadLimit)
}

// TestClient_WritePump_SuccessfulWrite 测试 writePump 成功写消息
func TestClient_WritePump_SuccessfulWrite(t *testing.T) {
	setupGlobalConfigForTests()
	mockConn := &mockWsConnection{}
	client := &Client{
		conn: mockConn,
		send: make(chan []byte, 1),
		ID:   "write-test-client",
	}

	writerClosed := false
	mc := &mockWriteCloser{
		closeFunc: func() error { writerClosed = true; return nil },
		writeFunc: func(p []byte) (n int, err error) { return len(p), nil },
	}
	mockConn.nextWriterFunc = func(messageType int) (io.WriteCloser, error) {
		assert.Equal(t, websocket.TextMessage, messageType)
		return mc, nil
	}
	writeDeadlineSet := false
	mockConn.setWriteDeadlineFunc = func(t time.Time) error { writeDeadlineSet = true; return nil }

	go client.writePump()
	client.send <- []byte("message to write")
	time.Sleep(100 * time.Millisecond)

	assert.True(t, writeDeadlineSet)
	assert.True(t, mockConn.nextWriterCalled)
	assert.True(t, writerClosed)

	close(client.send)
	// Wait for writePump to finish
	startTime := time.Now()
	for !mockConn.isCloseCalled() && time.Since(startTime) < 1*time.Second {
		time.Sleep(10 * time.Millisecond)
	}
	assert.True(t, mockConn.isCloseCalled(), "Connection Close should be called")
}

// TestClient_WritePump_SendChannelClosed 测试 writePump 在 send 通道关闭时行为
func TestClient_WritePump_SendChannelClosed(t *testing.T) {
	setupGlobalConfigForTests()
	mockConn := &mockWsConnection{}
	client := &Client{conn: mockConn, send: make(chan []byte, 1), ID: "write-close-test"}

	var closeMessageSent bool
	mockConn.writeMessageFunc = func(messageType int, data []byte) error {
		if messageType == websocket.CloseMessage {
			closeMessageSent = true
		}
		return nil
	}
	go client.writePump()
	close(client.send)
	// Wait
	startTime := time.Now()
	for !mockConn.isCloseCalled() && time.Since(startTime) < 1*time.Second {
		time.Sleep(10 * time.Millisecond)
	}

	assert.True(t, closeMessageSent, "websocket.CloseMessage should have been sent")
	assert.True(t, mockConn.isCloseCalled(), "Connection Close should have been called")
}

// TestClient_WritePump_Ping 测试 writePump 定期发送 Ping
func TestClient_WritePump_Ping(t *testing.T) {
	setupGlobalConfigForTests()
	originalPongWait := global.GVA_CONFIG.NfcRelay.WebsocketPongWaitSec
	global.GVA_CONFIG.NfcRelay.WebsocketPongWaitSec = 1 // For ~0.9s ping period
	defer func() { global.GVA_CONFIG.NfcRelay.WebsocketPongWaitSec = originalPongWait }()

	mockConn := &mockWsConnection{}
	client := &Client{conn: mockConn, send: make(chan []byte), ID: "ping-test-client"}

	pingSentCount := 0
	mockConn.writeMessageFunc = func(messageType int, data []byte) error {
		if messageType == websocket.PingMessage {
			pingSentCount++
		}
		return nil
	}
	go client.writePump()
	time.Sleep(1 * time.Second) // Wait longer than one ping period
	close(client.send)
	// Wait
	startTime := time.Now()
	for !mockConn.isCloseCalled() && time.Since(startTime) < 1*time.Second {
		time.Sleep(10 * time.Millisecond)
	}

	assert.GreaterOrEqual(t, pingSentCount, 1, "At least one PingMessage should have been sent")
	assert.True(t, mockConn.isCloseCalled(), "Connection Close should have been called")
}

// TestClient_WritePump_NextWriterError 测试 writePump 中 NextWriter 返回错误
func TestClient_WritePump_NextWriterError(t *testing.T) {
	setupGlobalConfigForTests()
	mockConn := &mockWsConnection{}
	client := &Client{conn: mockConn, send: make(chan []byte, 1), ID: "nextwriter-err-client"}
	mockConn.nextWriterFunc = func(messageType int) (io.WriteCloser, error) { return nil, errors.New("simulated NextWriter error") }
	go client.writePump()
	client.send <- []byte("some data")
	// Wait
	startTime := time.Now()
	for !mockConn.isCloseCalled() && time.Since(startTime) < 1*time.Second {
		time.Sleep(10 * time.Millisecond)
	}
	assert.True(t, mockConn.isCloseCalled(), "Connection Close should be called on NextWriter error")
}

// TestClient_WritePump_WriteError 测试 writePump 中 Writer.Write 返回错误
func TestClient_WritePump_WriteError(t *testing.T) {
	setupGlobalConfigForTests()
	mockConn := &mockWsConnection{}
	client := &Client{conn: mockConn, send: make(chan []byte, 1), ID: "writer-write-err-client"}
	mc := &mockWriteCloser{writeFunc: func(p []byte) (n int, err error) { return 0, errors.New("simulated Write error") }}
	mockConn.nextWriterFunc = func(messageType int) (io.WriteCloser, error) { return mc, nil }
	go client.writePump()
	client.send <- []byte("some data")
	// Wait
	startTime := time.Now()
	for !mockConn.isCloseCalled() && time.Since(startTime) < 1*time.Second {
		time.Sleep(10 * time.Millisecond)
	}
	assert.True(t, mockConn.isCloseCalled(), "Connection Close should be called on Writer.Write error")
}

// TestClient_WritePump_WriterCloseError 测试 writePump 中 Writer.Close 返回错误
func TestClient_WritePump_WriterCloseError(t *testing.T) {
	setupGlobalConfigForTests()
	mockConn := &mockWsConnection{}
	client := &Client{conn: mockConn, send: make(chan []byte, 1), ID: "writer-close-err-client"}
	mc := &mockWriteCloser{closeFunc: func() error { return errors.New("simulated Close error") }}
	mockConn.nextWriterFunc = func(messageType int) (io.WriteCloser, error) { return mc, nil }
	go client.writePump()
	client.send <- []byte("some data")
	// Wait
	startTime := time.Now()
	for !mockConn.isCloseCalled() && time.Since(startTime) < 1*time.Second {
		time.Sleep(10 * time.Millisecond)
	}
	assert.True(t, mockConn.isCloseCalled(), "Connection Close should be called on Writer.Close error")
}

// TestClient_WritePump_PingError 测试 writePump 中发送 Ping 失败
func TestClient_WritePump_PingError(t *testing.T) {
	setupGlobalConfigForTests()
	originalPongWait := global.GVA_CONFIG.NfcRelay.WebsocketPongWaitSec
	global.GVA_CONFIG.NfcRelay.WebsocketPongWaitSec = 1
	defer func() { global.GVA_CONFIG.NfcRelay.WebsocketPongWaitSec = originalPongWait }()

	mockConn := &mockWsConnection{}
	client := &Client{conn: mockConn, send: make(chan []byte), ID: "ping-err-client"}
	mockConn.writeMessageFunc = func(messageType int, data []byte) error {
		if messageType == websocket.PingMessage {
			return errors.New("simulated ping error")
		}
		return nil
	}
	go client.writePump()
	time.Sleep(1 * time.Second) // Wait for ping attempt
	// Wait
	startTime := time.Now()
	for !mockConn.isCloseCalled() && time.Since(startTime) < 1*time.Second {
		time.Sleep(10 * time.Millisecond)
	}
	assert.True(t, mockConn.isCloseCalled(), "Connection Close should be called on ping error")
}

// TestClient_WritePump_ConfigDefaults_PingPeriodZero 测试 writePump 在 pingPeriod <=0 时使用默认值
func TestClient_WritePump_ConfigDefaults_PingPeriodZero(t *testing.T) {
	setupGlobalConfigForTests()
	originalPongWait := global.GVA_CONFIG.NfcRelay.WebsocketPongWaitSec
	global.GVA_CONFIG.NfcRelay.WebsocketPongWaitSec = 0 // pingPeriod will be 0, then default to 54s
	defer func() { global.GVA_CONFIG.NfcRelay.WebsocketPongWaitSec = originalPongWait }()

	mockConn := &mockWsConnection{}
	client := &Client{conn: mockConn, send: make(chan []byte), ID: "ping-default-client"}

	pingSent := false
	mockConn.writeMessageFunc = func(messageType int, data []byte) error {
		if messageType == websocket.PingMessage {
			pingSent = true
		}
		return nil
	}
	go client.writePump()
	time.Sleep(1 * time.Second) // Should NOT send a ping if default 54s is used
	assert.False(t, pingSent, "Ping should not be sent quickly if 54s default ping period is used")
	close(client.send)
	// Wait
	startTime := time.Now()
	for !mockConn.isCloseCalled() && time.Since(startTime) < 1*time.Second {
		time.Sleep(10 * time.Millisecond)
	}
	assert.True(t, mockConn.isCloseCalled(), "Connection Close should have been called")
}
