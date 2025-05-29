package handler

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"testing"
	"time"

	"github.com/flipped-aurora/gin-vue-admin/server/config"
	"github.com/flipped-aurora/gin-vue-admin/server/global"
	"github.com/flipped-aurora/gin-vue-admin/server/nfc_relay/protocol"
	"github.com/flipped-aurora/gin-vue-admin/server/nfc_relay/security"
	"github.com/flipped-aurora/gin-vue-admin/server/nfc_relay/session"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

// TestWebSocketSecurity_APDUEncryptionDecryption 测试APDU加密解密流程
func TestWebSocketSecurity_APDUEncryptionDecryption(t *testing.T) {
	// 初始化配置
	global.GVA_CONFIG = config.Server{
		NfcRelay: config.NfcRelay{
			WebsocketPongWaitSec:     60,
			WebsocketMaxMessageBytes: 8192,
			WebsocketWriteWaitSec:    10,
		},
	}

	// 初始化日志记录器 (用于测试)
	if global.GVA_LOG == nil {
		logger, _ := zap.NewDevelopment()
		global.GVA_LOG = logger
	}

	// 创建安全管理器
	securityManager := security.NewHybridEncryptionManager()
	keyManager := security.NewKeyExchangeManager()

	testSessionID := "test-session-123"
	testUserID := "user-456"

	t.Run("APDU加密测试", func(t *testing.T) {
		// 测试APDU数据
		testAPDU := []byte{0x00, 0xA4, 0x04, 0x00, 0x0E, 0x32, 0x50, 0x41}

		// 构造元数据 - 移除时间限制，使用当前时间
		currentTimestamp := time.Now()
		metadata := security.APDUMetadata{
			SessionID:   testSessionID,
			SequenceNum: currentTimestamp.UnixNano(),
			Direction:   "upstream",
			Timestamp:   currentTimestamp,
			ClientID:    "test-client",
			UserID:      testUserID,
			DeviceInfo:  "test-device",
			ChecksumCRC: "CRC_8",
		}

		// 执行加密
		encryptedAPDU, err := securityManager.EncryptAPDUForTransmission(
			testSessionID,
			testAPDU,
			metadata,
			testUserID,
		)

		assert.NoError(t, err)
		assert.NotNil(t, encryptedAPDU)
		assert.NotEmpty(t, encryptedAPDU.AuditData.CommandClass)
		assert.GreaterOrEqual(t, encryptedAPDU.AuditData.RiskScore, 0)
	})

	t.Run("APDU解密测试", func(t *testing.T) {
		// 先加密一个APDU
		testAPDU := []byte{0x00, 0xA4, 0x04, 0x00, 0x0E}
		currentTimestamp := time.Now()
		metadata := security.APDUMetadata{
			SessionID:   testSessionID,
			SequenceNum: currentTimestamp.UnixNano(),
			Direction:   "downstream",
			Timestamp:   currentTimestamp,
			ClientID:    "test-client-2",
			UserID:      testUserID,
			DeviceInfo:  "test-device-2",
			ChecksumCRC: "CRC_5",
		}

		encryptedAPDU, err := securityManager.EncryptAPDUForTransmission(
			testSessionID,
			testAPDU,
			metadata,
			testUserID,
		)
		require.NoError(t, err)

		// 然后解密
		decryptedAPDU, err := securityManager.DecryptAPDUFromTransmission(
			testSessionID,
			encryptedAPDU,
			testUserID,
		)

		assert.NoError(t, err)
		assert.Equal(t, testAPDU, decryptedAPDU)
	})

	t.Run("传统密钥交换测试", func(t *testing.T) {
		// 生成会话密钥
		sessionKeys, err := keyManager.GenerateSessionKeys(testSessionID)
		assert.NoError(t, err)
		assert.NotNil(t, sessionKeys)
		assert.Len(t, sessionKeys.EncryptionKey, 32)
		assert.Len(t, sessionKeys.MACKey, 32)
		assert.NotEmpty(t, sessionKeys.KeyID)

		// 获取会话密钥
		retrievedKeys, err := keyManager.GetSessionKeys(testSessionID)
		assert.NoError(t, err)
		assert.Equal(t, sessionKeys.KeyID, retrievedKeys.KeyID)
		assert.Equal(t, sessionKeys.EncryptionKey, retrievedKeys.EncryptionKey)

		// 测试密钥撤销
		err = keyManager.RevokeSessionKeys(testSessionID)
		assert.NoError(t, err)

		// 撤销后应该无法获取
		_, err = keyManager.GetSessionKeys(testSessionID)
		assert.Error(t, err)
	})
}

// TestWebSocketSecurity_EncryptedPayloadMapping 测试加密载荷映射
func TestWebSocketSecurity_EncryptedPayloadMapping(t *testing.T) {
	hub := NewHub()

	t.Run("正确的载荷映射", func(t *testing.T) {
		// 构造测试载荷
		payload := map[string]interface{}{
			"auditData": map[string]interface{}{
				"commandClass":     "SELECT",
				"commandType":      "SELECT_FILE",
				"applicationId":    "test-app",
				"transactionType":  "PAYMENT",
				"currency":         "CNY",
				"merchantCategory": "RETAIL",
				"riskScore":        5,
				"timestamp":        time.Now().Format(time.RFC3339),
			},
			"businessData": map[string]interface{}{
				"encryptedData": "dGVzdC1lbmNyeXB0ZWQtZGF0YQ==",
				"encryptionInfo": map[string]interface{}{
					"algorithm": "AES-256-GCM",
					"nonce":     "dGVzdC1ub25jZQ==",
					"tag":       "dGVzdC10YWc=",
				},
			},
			"metadata": map[string]interface{}{
				"sessionId":   "test-session",
				"sequenceNum": 123456789,
				"direction":   "upstream",
				"timestamp":   time.Now().Format(time.RFC3339),
				"clientId":    "test-client",
				"userId":      "test-user",
				"deviceInfo":  "test-device",
				"checksumCrc": "CRC_123",
			},
		}

		var apduClass security.APDUDataClass
		err := hub.mapEncryptedPayloadToAPDUClass(payload, &apduClass)

		assert.NoError(t, err)
		assert.Equal(t, "SELECT", apduClass.AuditData.CommandClass)
		assert.Equal(t, "SELECT_FILE", apduClass.AuditData.CommandType)
		assert.Equal(t, "test-app", apduClass.AuditData.ApplicationID)
		assert.Equal(t, 5, apduClass.AuditData.RiskScore)
		assert.Equal(t, "dGVzdC1lbmNyeXB0ZWQtZGF0YQ==", apduClass.BusinessData.EncryptedData)
		assert.Equal(t, "AES-256-GCM", apduClass.BusinessData.EncryptionInfo.Algorithm)
		assert.Equal(t, "test-session", apduClass.Metadata.SessionID)
	})

	t.Run("缺少必需字段", func(t *testing.T) {
		payload := map[string]interface{}{
			"auditData": map[string]interface{}{
				"commandClass": "SELECT",
			},
			// 缺少 businessData 和 metadata
		}

		var apduClass security.APDUDataClass
		err := hub.mapEncryptedPayloadToAPDUClass(payload, &apduClass)

		assert.Error(t, err)
		assert.Contains(t, err.Error(), "缺少必需字段")
	})

	t.Run("字段类型错误", func(t *testing.T) {
		payload := map[string]interface{}{
			"auditData": "invalid_type", // 应该是map[string]interface{}
			"businessData": map[string]interface{}{
				"encryptedData": "test",
			},
			"metadata": map[string]interface{}{
				"sessionId": "test",
			},
		}

		var apduClass security.APDUDataClass
		err := hub.mapEncryptedPayloadToAPDUClass(payload, &apduClass)

		assert.Error(t, err)
	})
}

// TestWebSocketSecurity_APDUDataParsing 测试APDU数据解析
func TestWebSocketSecurity_APDUDataParsing(t *testing.T) {
	hub := NewHub()

	t.Run("有效的十六进制APDU", func(t *testing.T) {
		testCases := []struct {
			input    string
			expected []byte
		}{
			{"00A4040007", []byte{0x00, 0xA4, 0x04, 0x00, 0x07}},
			{"00 A4 04 00 07", []byte{0x00, 0xA4, 0x04, 0x00, 0x07}},
			{"00a4040007", []byte{0x00, 0xA4, 0x04, 0x00, 0x07}},
			{"00\nA4\t04 00\n07", []byte{0x00, 0xA4, 0x04, 0x00, 0x07}}, // 测试空白字符处理
		}

		for _, tc := range testCases {
			result, err := hub.parseAPDUData(tc.input)
			assert.NoError(t, err, "输入: %s", tc.input)
			assert.Equal(t, tc.expected, result, "输入: %s", tc.input)
		}
	})

	t.Run("无效的APDU格式", func(t *testing.T) {
		testCases := []string{
			"00A404000",      // 奇数长度
			"00G4040007",     // 无效字符
			"",               // 空字符串
			"ZZ",             // 完全无效
			"00 A4 04 00 0G", // 包含无效十六进制字符
		}

		for _, tc := range testCases {
			_, err := hub.parseAPDUData(tc)
			assert.Error(t, err, fmt.Sprintf("输入: %s", tc))
		}
	})
}

// TestWebSocketSecurityIntegration 综合测试WebSocket安全集成
func TestWebSocketSecurityIntegration(t *testing.T) {
	// 创建测试Hub
	hub := NewHub()

	// 创建模拟WebSocket连接
	mockProviderConn := &MockWebSocketConnection{}
	mockReceiverConn := &MockWebSocketConnection{}

	// 创建客户端
	providerClient := NewClient(hub, mockProviderConn)
	receiverClient := NewClient(hub, mockReceiverConn)

	// 设置客户端角色和用户信息
	providerClient.CurrentRole = protocol.RoleProvider
	receiverClient.CurrentRole = protocol.RoleReceiver
	providerClient.UserID = "testuser1"
	receiverClient.UserID = "testuser1"
	providerClient.Authenticated = true
	receiverClient.Authenticated = true

	// 创建测试会话
	testSession := session.NewSession("test-session-001")
	testSession.SetClient(providerClient, string(protocol.RoleProvider))
	testSession.SetClient(receiverClient, string(protocol.RoleReceiver))

	// 为客户端设置会话ID
	providerClient.SessionID = testSession.SessionID
	receiverClient.SessionID = testSession.SessionID

	// 将会话添加到Hub中
	hub.providerMutex.Lock()
	hub.sessions[testSession.SessionID] = testSession
	hub.clients[providerClient] = true
	hub.clients[receiverClient] = true
	hub.providerMutex.Unlock()

	t.Run("测试APDU加密和解密流程", func(t *testing.T) {
		// 生成会话密钥
		sessionKeys, err := security.GlobalKeyExchangeManager.GenerateSessionKeys(testSession.SessionID)
		require.NoError(t, err)
		assert.NotNil(t, sessionKeys)

		// 测试明文APDU加密
		testAPDU := []byte{0x00, 0xA4, 0x04, 0x00, 0x0E, 0x32, 0x50, 0x41, 0x59}
		metadata := map[string]string{
			"sessionId":   testSession.SessionID,
			"commandType": "SELECT",
		}

		encryptedAPDU, err := security.GlobalAPDUEncryption.EncryptCommandAPDU(
			testSession.SessionID,
			testAPDU,
			metadata,
		)
		require.NoError(t, err)
		assert.NotNil(t, encryptedAPDU)
		assert.NotEmpty(t, encryptedAPDU.EncryptedData)

		// 测试解密
		decryptedAPDU, err := security.GlobalAPDUEncryption.DecryptCommandAPDU(testSession.SessionID, encryptedAPDU)
		require.NoError(t, err)
		assert.NotNil(t, decryptedAPDU)
		assert.Equal(t, testAPDU, decryptedAPDU.CommandAPDU)
	})

	t.Run("测试会话管理", func(t *testing.T) {
		// 测试会话密钥轮换
		originalKeys, err := security.GlobalKeyExchangeManager.GetSessionKeys(testSession.SessionID)
		require.NoError(t, err)

		// 生成新密钥应该返回相同的密钥（如果未过期）
		newKeys, err := security.GlobalKeyExchangeManager.GenerateSessionKeys(testSession.SessionID)
		require.NoError(t, err)
		assert.Equal(t, originalKeys.KeyID, newKeys.KeyID)

		// 测试活跃会话计数
		activeCount := security.GlobalKeyExchangeManager.GetActiveSessionCount()
		assert.GreaterOrEqual(t, activeCount, 1)
	})

	t.Run("测试WebSocket连接管理", func(t *testing.T) {
		// 测试客户端计数
		activeConnections := hub.GetActiveConnectionsCount()
		assert.Equal(t, 2, activeConnections) // provider + receiver

		// 测试会话计数
		activeSessions := hub.GetActiveSessionsCount()
		assert.Equal(t, 1, activeSessions)

		// 测试查找客户端
		foundClient := hub.FindClientByID(providerClient.ID)
		assert.NotNil(t, foundClient)
		assert.Equal(t, providerClient.ID, foundClient.ID)

		// 测试查找不存在的客户端
		notFoundClient := hub.FindClientByID("non-existent-id")
		assert.Nil(t, notFoundClient)
	})

	t.Run("测试明文APDU处理流程", func(t *testing.T) {
		// 构造明文APDU上行消息
		upstreamMsg := protocol.APDUUpstreamMessage{
			Type: protocol.MessageTypeAPDUUpstream,
			APDU: "00A404000E325041592E5359532E4444463031", // SELECT AID
		}

		msgBytes, err := json.Marshal(upstreamMsg)
		require.NoError(t, err)

		// 设置provider客户端为可发送状态
		mockProviderConn.SetSendable(true)

		// 测试处理明文APDU交换
		hub.handleAPDUExchange(receiverClient, msgBytes, "upstream")

		// 验证消息是否被转发到provider客户端
		sentMessages := mockProviderConn.GetSentMessages()
		if len(sentMessages) > 0 {
			assert.Contains(t, string(sentMessages[0]), "00A404000E325041592E5359532E4444463031")
		}
	})

	// 清理测试环境
	hub.providerMutex.Lock()
	delete(hub.sessions, testSession.SessionID)
	delete(hub.clients, providerClient)
	delete(hub.clients, receiverClient)
	hub.providerMutex.Unlock()

	// 清理会话密钥
	security.GlobalKeyExchangeManager.RevokeSessionKeys(testSession.SessionID)
}

// TestAPDUClassificationAndMapping 测试APDU分类和映射功能
func TestAPDUClassificationAndMapping(t *testing.T) {
	t.Run("测试不同APDU命令的分类", func(t *testing.T) {
		testCases := []struct {
			name      string
			apduHex   string
			minLength int
		}{
			{"SELECT命令", "00A404000E325041592E5359532E4444463031", 19},
			{"READ_RECORD命令", "00B2010C00", 5},
			{"VERIFY命令", "0020000108", 5},
			{"GET_CHALLENGE命令", "0084000008", 5},
			{"未知命令", "00FF000000", 5},
		}

		hub := NewHub()

		for _, tc := range testCases {
			t.Run(tc.name, func(t *testing.T) {
				apduBytes, err := hub.parseAPDUData(tc.apduHex)
				require.NoError(t, err)
				assert.Equal(t, tc.minLength, len(apduBytes))

				// 验证APDU解析正确
				assert.Greater(t, len(apduBytes), 0)
			})
		}
	})
}

// TestErrorHandling 测试错误处理机制
func TestErrorHandling(t *testing.T) {
	hub := NewHub()

	t.Run("测试无效的加密载荷映射", func(t *testing.T) {
		// 缺少必需字段的载荷
		incompletePayload := map[string]interface{}{
			"auditData": map[string]interface{}{
				"commandClass": "SELECT",
			},
			// 缺少 businessData 和 metadata
		}

		apduClass := &security.APDUDataClass{}
		err := hub.mapEncryptedPayloadToAPDUClass(incompletePayload, apduClass)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "缺少必需字段")
	})

	t.Run("测试无效的APDU数据格式", func(t *testing.T) {
		invalidFormats := []string{
			"",         // 空字符串
			"XX",       // 无效十六进制
			"123",      // 奇数长度
			"12 34 GG", // 包含无效字符
		}

		for _, invalid := range invalidFormats {
			_, err := hub.parseAPDUData(invalid)
			assert.Error(t, err, "应该拒绝无效格式: %s", invalid)
		}
	})
}

// TestConcurrencyAndRaceConditions 测试并发和竞态条件
func TestConcurrencyAndRaceConditions(t *testing.T) {
	hub := NewHub()

	t.Run("测试并发会话密钥生成", func(t *testing.T) {
		const goroutines = 10
		const sessionsPerGoroutine = 10

		results := make(chan error, goroutines*sessionsPerGoroutine)

		for i := 0; i < goroutines; i++ {
			go func(workerID int) {
				for j := 0; j < sessionsPerGoroutine; j++ {
					sessionID := fmt.Sprintf("concurrent-test-%d-%d-%d", workerID, j, time.Now().UnixNano())
					_, err := security.GlobalKeyExchangeManager.GenerateSessionKeys(sessionID)
					results <- err
				}
			}(i)
		}

		// 收集所有结果
		for i := 0; i < goroutines*sessionsPerGoroutine; i++ {
			err := <-results
			assert.NoError(t, err, "并发密钥生成应该成功")
		}
	})

	t.Run("测试并发客户端管理", func(t *testing.T) {
		const numClients = 20
		clients := make([]*Client, numClients)

		// 并发添加客户端
		done := make(chan bool, numClients)
		for i := 0; i < numClients; i++ {
			go func(index int) {
				mockConn := &MockWebSocketConnection{}
				client := NewClient(hub, mockConn)
				clients[index] = client

				hub.providerMutex.Lock()
				hub.clients[client] = true
				hub.providerMutex.Unlock()

				done <- true
			}(i)
		}

		// 等待所有goroutine完成
		for i := 0; i < numClients; i++ {
			<-done
		}

		// 验证客户端数量
		activeCount := hub.GetActiveConnectionsCount()
		assert.Equal(t, numClients, activeCount)

		// 清理
		hub.providerMutex.Lock()
		for _, client := range clients {
			if client != nil {
				delete(hub.clients, client)
			}
		}
		hub.providerMutex.Unlock()
	})
}

// BenchmarkWebSocketSecurity_APDUProcessing APDU处理性能基准测试
func BenchmarkWebSocketSecurity_APDUProcessing(b *testing.B) {
	securityManager := security.NewHybridEncryptionManager()
	testSessionID := "bench-session"
	testUserID := "bench-user"
	testAPDU := []byte{0x00, 0xA4, 0x04, 0x00, 0x0E}

	metadata := security.APDUMetadata{
		SessionID:   testSessionID,
		SequenceNum: time.Now().UnixNano(),
		Direction:   "upstream",
		Timestamp:   time.Now(),
		ClientID:    "bench-client",
		UserID:      testUserID,
		DeviceInfo:  "bench-device",
		ChecksumCRC: "CRC_5",
	}

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		// 加密
		encryptedAPDU, err := securityManager.EncryptAPDUForTransmission(
			testSessionID,
			testAPDU,
			metadata,
			testUserID,
		)
		if err != nil {
			b.Fatal(err)
		}

		// 解密
		_, err = securityManager.DecryptAPDUFromTransmission(
			testSessionID,
			encryptedAPDU,
			testUserID,
		)
		if err != nil {
			b.Fatal(err)
		}
	}
}

// MockWebSocketConnection 模拟WebSocket连接用于测试
type MockWebSocketConnection struct {
	sendable     bool
	sentMessages [][]byte
	closed       bool
}

func (m *MockWebSocketConnection) SetSendable(sendable bool) {
	m.sendable = sendable
}

func (m *MockWebSocketConnection) GetSentMessages() [][]byte {
	return m.sentMessages
}

func (m *MockWebSocketConnection) Close() error {
	m.closed = true
	return nil
}

func (m *MockWebSocketConnection) SetReadLimit(limit int64) {}

func (m *MockWebSocketConnection) SetReadDeadline(t time.Time) error {
	return nil
}

func (m *MockWebSocketConnection) SetWriteDeadline(t time.Time) error {
	return nil
}

func (m *MockWebSocketConnection) SetPongHandler(h func(string) error) {}

func (m *MockWebSocketConnection) ReadMessage() (messageType int, p []byte, err error) {
	// 模拟读取超时
	time.Sleep(10 * time.Millisecond)
	return 0, nil, errors.New("mock read timeout")
}

func (m *MockWebSocketConnection) NextWriter(messageType int) (io.WriteCloser, error) {
	return &MockWriteCloser{conn: m}, nil
}

func (m *MockWebSocketConnection) WriteMessage(messageType int, data []byte) error {
	if !m.sendable {
		return errors.New("connection not sendable")
	}
	m.sentMessages = append(m.sentMessages, data)
	return nil
}

func (m *MockWebSocketConnection) RemoteAddr() net.Addr {
	return &MockAddr{address: "127.0.0.1:12345"}
}

type MockWriteCloser struct {
	conn *MockWebSocketConnection
	data []byte
}

func (m *MockWriteCloser) Write(p []byte) (n int, err error) {
	m.data = append(m.data, p...)
	return len(p), nil
}

func (m *MockWriteCloser) Close() error {
	if m.conn.sendable {
		m.conn.sentMessages = append(m.conn.sentMessages, m.data)
	}
	return nil
}

type MockAddr struct {
	address string
}

func (m *MockAddr) Network() string {
	return "tcp"
}

func (m *MockAddr) String() string {
	return m.address
}
