package service

import (
	"fmt"
	"net/http"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
	"go.uber.org/zap"

	"github.com/flipped-aurora/gin-vue-admin/server/global"
	"github.com/flipped-aurora/gin-vue-admin/server/nfc_relay/handler"
)

// WebSocketMessage WebSocket消息结构
type WebSocketMessage struct {
	Type    string      `json:"type"`
	Payload interface{} `json:"payload"`
}

// RealtimeStats 实时统计数据
type RealtimeStats struct {
	HubStatus             string      `json:"hub_status"`
	ActiveConnections     int         `json:"active_connections"`
	ActiveSessions        int         `json:"active_sessions"`
	ApduRelayedLastMinute int         `json:"apdu_relayed_last_minute"`
	ApduErrorsLastHour    int         `json:"apdu_errors_last_hour"`
	ConnectionTrend       []TrendData `json:"connection_trend"`
	SessionTrend          []TrendData `json:"session_trend"`
	SystemLoad            float64     `json:"system_load"`
	MemoryUsage           float64     `json:"memory_usage"`
	AvgResponseTime       int64       `json:"avg_response_time"`
}

// TrendData 趋势数据点
type TrendData struct {
	Time  string `json:"time"`
	Count int    `json:"count"`
}

// ClientInfo 客户端信息
type ClientInfo struct {
	ClientID     string                 `json:"client_id"`
	UserID       string                 `json:"user_id"`
	DisplayName  string                 `json:"display_name"`
	Role         string                 `json:"role"`
	IPAddress    string                 `json:"ip_address"`
	ConnectedAt  time.Time              `json:"connected_at"`
	IsOnline     bool                   `json:"is_online"`
	SessionID    string                 `json:"session_id,omitempty"`
	MessageStats map[string]interface{} `json:"message_stats,omitempty"`
}

// SessionInfo 会话信息
type SessionInfo struct {
	SessionID         string         `json:"session_id"`
	Status            string         `json:"status"`
	CreatedAt         time.Time      `json:"created_at"`
	LastActivity      time.Time      `json:"last_activity_at"`
	ProviderInfo      ClientInfo     `json:"provider_info"`
	ReceiverInfo      ClientInfo     `json:"receiver_info"`
	ApduExchangeCount map[string]int `json:"apdu_exchange_count"`
}

// ClientConnection 带写入锁的客户端连接
type ClientConnection struct {
	conn   *websocket.Conn
	mutex  sync.Mutex
	closed bool
}

// Write 安全地写入消息到WebSocket连接
func (cc *ClientConnection) Write(message WebSocketMessage) error {
	cc.mutex.Lock()
	defer cc.mutex.Unlock()

	if cc.closed {
		return fmt.Errorf("connection is closed")
	}

	return cc.conn.WriteJSON(message)
}

// Close 安全地关闭连接
func (cc *ClientConnection) Close() error {
	cc.mutex.Lock()
	defer cc.mutex.Unlock()

	if !cc.closed {
		cc.closed = true
		return cc.conn.Close()
	}
	return nil
}

// RealtimeDataService 实时数据服务
type RealtimeDataService struct {
	clients        map[*ClientConnection]bool
	broadcast      chan WebSocketMessage
	register       chan *ClientConnection
	unregister     chan *ClientConnection
	mutex          sync.RWMutex
	relayHub       *handler.Hub
	statsCollector *StatsCollector
	logger         *zap.Logger
}

// StatsCollector 统计数据收集器
type StatsCollector struct {
	connectionHistory []TrendData
	sessionHistory    []TrendData
	apduCounts        map[string]int
	errorCounts       map[string]int
	mutex             sync.RWMutex
	startTime         time.Time
}

var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool {
		// 在生产环境中应该检查Origin
		return true
	},
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
}

// NewRealtimeDataService 创建实时数据服务
func NewRealtimeDataService(relayHub *handler.Hub) *RealtimeDataService {
	service := &RealtimeDataService{
		clients:    make(map[*ClientConnection]bool),
		broadcast:  make(chan WebSocketMessage, 256),
		register:   make(chan *ClientConnection),
		unregister: make(chan *ClientConnection),
		relayHub:   relayHub,
		logger:     global.GVA_LOG,
		statsCollector: &StatsCollector{
			connectionHistory: make([]TrendData, 0),
			sessionHistory:    make([]TrendData, 0),
			apduCounts:        make(map[string]int),
			errorCounts:       make(map[string]int),
			startTime:         time.Now(),
		},
	}

	go service.run()
	go service.collectStats()

	// 启动模拟数据生成器（用于演示）
	//mockGenerator := NewMockDataGenerator(service)
	//mockGenerator.StartMockData()
	//service.logger.Info("模拟数据生成器已启动")

	return service
}

// HandleWebSocket 处理WebSocket连接
func (s *RealtimeDataService) HandleWebSocket(c *gin.Context) {
	conn, err := upgrader.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		s.logger.Error("WebSocket upgrade failed", zap.Error(err))
		return
	}

	s.logger.Info("New WebSocket connection established",
		zap.String("remote_addr", conn.RemoteAddr().String()))

	// 创建单一的ClientConnection实例
	clientConn := &ClientConnection{conn: conn}

	// 注册连接
	s.register <- clientConn

	// 处理连接 - 不再单独创建新的goroutine
	s.handleConnection(clientConn)
}

// run 运行主循环
func (s *RealtimeDataService) run() {
	for {
		select {
		case conn := <-s.register:
			s.mutex.Lock()
			s.clients[conn] = true
			s.mutex.Unlock()
			s.logger.Info("WebSocket client registered",
				zap.Int("total_clients", len(s.clients)))

			// 立即发送初始数据给新连接的客户端
			go s.sendInitialData(conn)

		case conn := <-s.unregister:
			s.mutex.Lock()
			if _, ok := s.clients[conn]; ok {
				delete(s.clients, conn)
				conn.Close()
			}
			s.mutex.Unlock()
			s.logger.Info("WebSocket client unregistered",
				zap.Int("total_clients", len(s.clients)))

		case message := <-s.broadcast:
			s.mutex.RLock()
			for conn := range s.clients {
				select {
				case <-time.After(10 * time.Second):
					// 写入超时，移除连接
					s.logger.Warn("WebSocket write timeout, removing client")
					delete(s.clients, conn)
					conn.Close()
				default:
					if err := conn.Write(message); err != nil {
						s.logger.Error("WebSocket write error", zap.Error(err))
						delete(s.clients, conn)
						conn.Close()
					}
				}
			}
			s.mutex.RUnlock()
		}
	}
}

// handleConnection 处理单个WebSocket连接
func (s *RealtimeDataService) handleConnection(clientConn *ClientConnection) {
	defer func() {
		s.unregister <- clientConn
	}()

	// 设置读取超时
	clientConn.conn.SetReadDeadline(time.Now().Add(60 * time.Second))
	clientConn.conn.SetPongHandler(func(string) error {
		clientConn.conn.SetReadDeadline(time.Now().Add(60 * time.Second))
		return nil
	})

	// 发送初始数据（在注册完成后发送）
	go s.sendInitialData(clientConn)

	for {
		var msg struct {
			Type string `json:"type"`
		}

		if err := clientConn.conn.ReadJSON(&msg); err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				s.logger.Error("WebSocket read error", zap.Error(err))
			}
			break
		}

		// 处理状态更新请求
		if msg.Type == "request_status_update" {
			s.logger.Info("Received status update request from client")
			go s.sendInitialData(clientConn)
		}
	}
}

// sendInitialData 发送初始数据
func (s *RealtimeDataService) sendInitialData(clientConn *ClientConnection) {
	// 短暂延迟确保连接完全建立
	time.Sleep(100 * time.Millisecond)

	s.logger.Info("Sending initial data to WebSocket client")

	// 发送仪表盘数据
	dashboardStats := s.getDashboardStats()
	s.logger.Info("Sending dashboard data",
		zap.String("hub_status", dashboardStats.HubStatus),
		zap.Int("active_connections", dashboardStats.ActiveConnections),
		zap.Int("active_sessions", dashboardStats.ActiveSessions))

	s.sendToClient(clientConn, WebSocketMessage{
		Type:    "dashboard_update",
		Payload: dashboardStats,
	})

	// 发送客户端数据
	s.sendToClient(clientConn, WebSocketMessage{
		Type:    "clients_update",
		Payload: s.getClientsData(),
	})

	// 发送会话数据
	s.sendToClient(clientConn, WebSocketMessage{
		Type:    "sessions_update",
		Payload: s.getSessionsData(),
	})

	s.logger.Info("Initial data sent to WebSocket client successfully")
}

// sendToClient 发送消息给特定客户端
func (s *RealtimeDataService) sendToClient(clientConn *ClientConnection, message WebSocketMessage) {
	if err := clientConn.Write(message); err != nil {
		s.logger.Error("Failed to send message to client", zap.Error(err))
	}
}

// BroadcastMessage 广播消息
func (s *RealtimeDataService) BroadcastMessage(msgType string, payload interface{}) {
	select {
	case s.broadcast <- WebSocketMessage{Type: msgType, Payload: payload}:
	default:
		s.logger.Warn("Broadcast channel full, dropping message", zap.String("type", msgType))
	}
}

// collectStats 收集统计数据
func (s *RealtimeDataService) collectStats() {
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			s.updateTrendData()
			s.BroadcastMessage("dashboard_update", s.getDashboardStats())
			s.BroadcastMessage("clients_update", s.getClientsData())
			s.BroadcastMessage("sessions_update", s.getSessionsData())
		}
	}
}

// updateTrendData 更新趋势数据
func (s *RealtimeDataService) updateTrendData() {
	s.statsCollector.mutex.Lock()
	defer s.statsCollector.mutex.Unlock()

	now := time.Now()
	timeStr := now.Format("15:04")

	// 获取当前连接数和会话数
	connectionCount := s.getCurrentConnectionCount()
	sessionCount := s.getCurrentSessionCount()

	// 添加趋势数据点
	s.statsCollector.connectionHistory = append(s.statsCollector.connectionHistory, TrendData{
		Time:  timeStr,
		Count: connectionCount,
	})
	s.statsCollector.sessionHistory = append(s.statsCollector.sessionHistory, TrendData{
		Time:  timeStr,
		Count: sessionCount,
	})

	// 保持最近100个数据点
	if len(s.statsCollector.connectionHistory) > 100 {
		s.statsCollector.connectionHistory = s.statsCollector.connectionHistory[1:]
	}
	if len(s.statsCollector.sessionHistory) > 100 {
		s.statsCollector.sessionHistory = s.statsCollector.sessionHistory[1:]
	}
}

// getDashboardStats 获取仪表盘统计数据
func (s *RealtimeDataService) getDashboardStats() RealtimeStats {
	s.statsCollector.mutex.RLock()
	defer s.statsCollector.mutex.RUnlock()

	return RealtimeStats{
		HubStatus:             s.getHubStatus(),
		ActiveConnections:     s.getCurrentConnectionCount(),
		ActiveSessions:        s.getCurrentSessionCount(),
		ApduRelayedLastMinute: s.getApduCountLastMinute(),
		ApduErrorsLastHour:    s.getErrorCountLastHour(),
		ConnectionTrend:       s.statsCollector.connectionHistory,
		SessionTrend:          s.statsCollector.sessionHistory,
		SystemLoad:            s.getSystemLoad(),
		MemoryUsage:           s.getMemoryUsage(),
		AvgResponseTime:       s.getAvgResponseTime(),
	}
}

// getClientsData 获取客户端数据
func (s *RealtimeDataService) getClientsData() map[string]interface{} {
	clients := s.getAllClients()

	onlineCount := 0
	providerCount := 0
	receiverCount := 0

	for _, client := range clients {
		if client.IsOnline {
			onlineCount++
			switch client.Role {
			case "provider":
				providerCount++
			case "receiver":
				receiverCount++
			}
		}
	}

	return map[string]interface{}{
		"list":           clients,
		"total":          len(clients),
		"online_count":   onlineCount,
		"provider_count": providerCount,
		"receiver_count": receiverCount,
	}
}

// getSessionsData 获取会话数据
func (s *RealtimeDataService) getSessionsData() map[string]interface{} {
	sessions := s.getAllSessions()

	pairedCount := 0
	waitingCount := 0

	for _, session := range sessions {
		switch session.Status {
		case "paired":
			pairedCount++
		case "waiting_for_pairing":
			waitingCount++
		}
	}

	return map[string]interface{}{
		"list":          sessions,
		"total":         len(sessions),
		"paired_count":  pairedCount,
		"waiting_count": waitingCount,
	}
}

// Event handlers for NFC Relay events
func (s *RealtimeDataService) OnClientConnected(clientInfo ClientInfo) {
	s.BroadcastMessage("client_connected", clientInfo)
	s.logger.Info("Client connected event broadcasted", zap.String("client_id", clientInfo.ClientID))
}

func (s *RealtimeDataService) OnClientDisconnected(clientInfo ClientInfo) {
	s.BroadcastMessage("client_disconnected", clientInfo)
	s.logger.Info("Client disconnected event broadcasted", zap.String("client_id", clientInfo.ClientID))
}

func (s *RealtimeDataService) OnSessionCreated(sessionInfo SessionInfo) {
	s.BroadcastMessage("session_created", sessionInfo)
	s.logger.Info("Session created event broadcasted", zap.String("session_id", sessionInfo.SessionID))
}

func (s *RealtimeDataService) OnSessionTerminated(sessionInfo SessionInfo) {
	s.BroadcastMessage("session_terminated", sessionInfo)
	s.logger.Info("Session terminated event broadcasted", zap.String("session_id", sessionInfo.SessionID))
}

func (s *RealtimeDataService) OnApduRelayed(sessionID, direction string, length int) {
	s.statsCollector.mutex.Lock()
	s.statsCollector.apduCounts[fmt.Sprintf("%d", time.Now().Unix()/60)]++
	s.statsCollector.mutex.Unlock()

	payload := map[string]interface{}{
		"session_id": sessionID,
		"direction":  direction,
		"length":     length,
		"timestamp":  time.Now(),
	}
	s.BroadcastMessage("apdu_relayed", payload)
}

// Helper methods - 这些需要根据实际的NFC Relay Hub实现来填充
func (s *RealtimeDataService) getHubStatus() string {
	if s.relayHub != nil {
		return "online"
	}
	return "offline"
}

func (s *RealtimeDataService) getCurrentConnectionCount() int {
	// 从 RelayHub 获取当前连接数
	if s.relayHub != nil {
		// 这里需要根据实际的RelayHub实现来获取
		return len(s.relayHub.GetAllClients())
	}
	return 0
}

func (s *RealtimeDataService) getCurrentSessionCount() int {
	// 从 RelayHub 获取当前会话数
	if s.relayHub != nil {
		return len(s.relayHub.GetAllSessions())
	}
	return 0
}

func (s *RealtimeDataService) getAllClients() []ClientInfo {
	// 从 RelayHub 获取所有客户端信息
	clients := make([]ClientInfo, 0)

	if s.relayHub != nil {
		// 这里需要根据实际的RelayHub实现来转换数据
		for _, client := range s.relayHub.GetAllClients() {
			clients = append(clients, ClientInfo{
				ClientID:    client.ID,
				UserID:      client.UserID,
				DisplayName: client.DisplayName,
				Role:        string(client.CurrentRole),
				IPAddress:   client.GetRemoteAddr(),
				ConnectedAt: time.Now(), // Client结构中没有ConnectedAt字段，使用当前时间
				IsOnline:    client.IsOnline,
				SessionID:   client.SessionID,
			})
		}
	}

	return clients
}

func (s *RealtimeDataService) getAllSessions() []SessionInfo {
	// 从 RelayHub 获取所有会话信息
	sessions := make([]SessionInfo, 0)

	if s.relayHub != nil {
		// 这里需要根据实际的RelayHub实现来转换数据
		for _, sess := range s.relayHub.GetAllSessions() {
			sessions = append(sessions, SessionInfo{
				SessionID:    sess.SessionID,
				Status:       string(sess.Status),
				CreatedAt:    sess.CreatedAt,
				LastActivity: sess.LastActivityTime,
				// ProviderInfo 和 ReceiverInfo 需要从会话中提取
			})
		}
	}

	return sessions
}

func (s *RealtimeDataService) getApduCountLastMinute() int {
	s.statsCollector.mutex.RLock()
	defer s.statsCollector.mutex.RUnlock()

	currentMinute := fmt.Sprintf("%d", time.Now().Unix()/60)
	return s.statsCollector.apduCounts[currentMinute]
}

func (s *RealtimeDataService) getErrorCountLastHour() int {
	s.statsCollector.mutex.RLock()
	defer s.statsCollector.mutex.RUnlock()

	// 统计最近一小时的错误数
	now := time.Now().Unix()
	count := 0
	for timeStr, errorCount := range s.statsCollector.errorCounts {
		if timestamp, err := time.Parse("2006-01-02 15:04", timeStr); err == nil {
			if now-timestamp.Unix() <= 3600 { // 1小时
				count += errorCount
			}
		}
	}
	return count
}

func (s *RealtimeDataService) getSystemLoad() float64 {
	// 实现系统负载获取
	// 可以使用 github.com/shirou/gopsutil 库
	return 35.0 // 模拟数据
}

func (s *RealtimeDataService) getMemoryUsage() float64 {
	// 实现内存使用率获取
	return 68.0 // 模拟数据
}

func (s *RealtimeDataService) getAvgResponseTime() int64 {
	// 实现平均响应时间获取
	return 45 // 模拟数据，单位：毫秒
}
