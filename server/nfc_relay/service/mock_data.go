package service

import (
	"fmt"
	"math/rand"
	"time"
)

// MockDataGenerator 模拟数据生成器
type MockDataGenerator struct {
	service *RealtimeDataService
}

// NewMockDataGenerator 创建模拟数据生成器
func NewMockDataGenerator(service *RealtimeDataService) *MockDataGenerator {
	return &MockDataGenerator{
		service: service,
	}
}

// StartMockData 开始生成模拟数据
func (m *MockDataGenerator) StartMockData() {
	go m.generateMockEvents()
}

// generateMockEvents 生成模拟事件
func (m *MockDataGenerator) generateMockEvents() {
	ticker := time.NewTicker(10 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			// 随机生成不同类型的事件
			eventType := rand.Intn(4)

			switch eventType {
			case 0:
				// 模拟客户端连接
				m.simulateClientConnection()
			case 1:
				// 模拟会话创建
				m.simulateSessionCreation()
			case 2:
				// 模拟APDU中继
				m.simulateApduRelay()
			case 3:
				// 模拟客户端断开
				m.simulateClientDisconnection()
			}
		}
	}
}

// simulateClientConnection 模拟客户端连接
func (m *MockDataGenerator) simulateClientConnection() {
	clientInfo := ClientInfo{
		ClientID:    generateRandomID(),
		UserID:      generateRandomUserID(),
		DisplayName: generateRandomDisplayName(),
		Role:        generateRandomRole(),
		IPAddress:   generateRandomIP(),
		ConnectedAt: time.Now(),
		IsOnline:    true,
	}

	m.service.OnClientConnected(clientInfo)
}

// simulateSessionCreation 模拟会话创建
func (m *MockDataGenerator) simulateSessionCreation() {
	sessionInfo := SessionInfo{
		SessionID:    generateRandomID(),
		Status:       "paired",
		CreatedAt:    time.Now(),
		LastActivity: time.Now(),
		ProviderInfo: ClientInfo{
			ClientID:    generateRandomID(),
			DisplayName: "Provider " + generateRandomID()[:8],
			Role:        "provider",
		},
		ReceiverInfo: ClientInfo{
			ClientID:    generateRandomID(),
			DisplayName: "Receiver " + generateRandomID()[:8],
			Role:        "receiver",
		},
		ApduExchangeCount: map[string]int{
			"upstream":   rand.Intn(100),
			"downstream": rand.Intn(100),
		},
	}

	m.service.OnSessionCreated(sessionInfo)
}

// simulateApduRelay 模拟APDU中继
func (m *MockDataGenerator) simulateApduRelay() {
	directions := []string{"upstream", "downstream"}
	direction := directions[rand.Intn(len(directions))]

	m.service.OnApduRelayed(
		generateRandomID(),
		direction,
		rand.Intn(256)+32, // 32-288字节
	)
}

// simulateClientDisconnection 模拟客户端断开
func (m *MockDataGenerator) simulateClientDisconnection() {
	clientInfo := ClientInfo{
		ClientID:    generateRandomID(),
		UserID:      generateRandomUserID(),
		DisplayName: generateRandomDisplayName(),
		Role:        generateRandomRole(),
		IPAddress:   generateRandomIP(),
		ConnectedAt: time.Now().Add(-time.Hour),
		IsOnline:    false,
	}

	m.service.OnClientDisconnected(clientInfo)
}

// 辅助函数
func generateRandomID() string {
	const charset = "abcdefghijklmnopqrstuvwxyz0123456789"
	b := make([]byte, 16)
	for i := range b {
		b[i] = charset[rand.Intn(len(charset))]
	}
	return string(b)
}

func generateRandomUserID() string {
	return "user_" + generateRandomID()[:8]
}

func generateRandomDisplayName() string {
	names := []string{
		"iPhone 15 Pro", "Samsung Galaxy S24", "POS Terminal A",
		"Card Reader B", "NFC Device C", "Mobile Payment D",
		"Smart Card E", "Terminal F", "Device G", "Reader H",
	}
	return names[rand.Intn(len(names))]
}

func generateRandomRole() string {
	roles := []string{"provider", "receiver", "none"}
	return roles[rand.Intn(len(roles))]
}

func generateRandomIP() string {
	return fmt.Sprintf("192.168.1.%d", rand.Intn(254)+1)
}
