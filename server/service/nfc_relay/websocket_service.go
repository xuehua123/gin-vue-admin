package nfc_relay

import (
	"encoding/json"
	"fmt"
	"net/http"
	"sync"
	"time"

	"github.com/flipped-aurora/gin-vue-admin/server/global"
	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
	"go.uber.org/zap"
)

// WebSocketMessage WebSocket消息结构
type WebSocketMessage struct {
	Type      string      `json:"type"`      // 消息类型: transaction_status, apdu_message, client_online, etc.
	Data      interface{} `json:"data"`      // 消息数据
	Timestamp time.Time   `json:"timestamp"` // 时间戳
}

// WebSocketClient WebSocket客户端连接
type WebSocketClient struct {
	conn     *websocket.Conn
	userID   string
	clientID string
	send     chan []byte
	hub      *WebSocketHub
}

// WebSocketHub WebSocket连接管理中心
type WebSocketHub struct {
	clients     map[*WebSocketClient]bool
	broadcast   chan []byte
	register    chan *WebSocketClient
	unregister  chan *WebSocketClient
	userClients map[string][]*WebSocketClient // 按用户ID组织客户端
	mutex       sync.RWMutex
}

var (
	wsHub      *WebSocketHub
	wsUpgrader = websocket.Upgrader{
		ReadBufferSize:  1024,
		WriteBufferSize: 1024,
		CheckOrigin: func(r *http.Request) bool {
			// 在生产环境中应该检查Origin
			return true
		},
	}
)

// GetWebSocketHub 获取WebSocket Hub单例
func GetWebSocketHub() *WebSocketHub {
	if wsHub == nil {
		wsHub = &WebSocketHub{
			clients:     make(map[*WebSocketClient]bool),
			broadcast:   make(chan []byte),
			register:    make(chan *WebSocketClient),
			unregister:  make(chan *WebSocketClient),
			userClients: make(map[string][]*WebSocketClient),
		}
		go wsHub.run()
	}
	return wsHub
}

// run WebSocket Hub运行循环
func (h *WebSocketHub) run() {
	for {
		select {
		case client := <-h.register:
			h.mutex.Lock()
			h.clients[client] = true
			if h.userClients[client.userID] == nil {
				h.userClients[client.userID] = make([]*WebSocketClient, 0)
			}
			h.userClients[client.userID] = append(h.userClients[client.userID], client)
			h.mutex.Unlock()

			global.GVA_LOG.Info("WebSocket client connected",
				zap.String("userID", client.userID),
				zap.String("clientID", client.clientID))

		case client := <-h.unregister:
			h.mutex.Lock()
			if _, ok := h.clients[client]; ok {
				delete(h.clients, client)
				close(client.send)

				// 从用户客户端列表中移除
				if userClients, exists := h.userClients[client.userID]; exists {
					for i, c := range userClients {
						if c == client {
							h.userClients[client.userID] = append(userClients[:i], userClients[i+1:]...)
							break
						}
					}
					if len(h.userClients[client.userID]) == 0 {
						delete(h.userClients, client.userID)
					}
				}
			}
			h.mutex.Unlock()

			global.GVA_LOG.Info("WebSocket client disconnected",
				zap.String("userID", client.userID),
				zap.String("clientID", client.clientID))

		case message := <-h.broadcast:
			h.mutex.RLock()
			for client := range h.clients {
				select {
				case client.send <- message:
				default:
					close(client.send)
					delete(h.clients, client)
				}
			}
			h.mutex.RUnlock()
		}
	}
}

// BroadcastToUser 向特定用户发送消息
func (h *WebSocketHub) BroadcastToUser(userID string, message WebSocketMessage) error {
	h.mutex.RLock()
	defer h.mutex.RUnlock()

	clients, exists := h.userClients[userID]
	if !exists || len(clients) == 0 {
		return fmt.Errorf("user %s has no active connections", userID)
	}

	message.Timestamp = time.Now()
	data, err := json.Marshal(message)
	if err != nil {
		return err
	}

	for _, client := range clients {
		select {
		case client.send <- data:
		default:
			global.GVA_LOG.Warn("Failed to send message to client",
				zap.String("userID", userID),
				zap.String("clientID", client.clientID))
		}
	}

	return nil
}

// BroadcastAll 广播消息给所有连接
func (h *WebSocketHub) BroadcastAll(message WebSocketMessage) error {
	message.Timestamp = time.Now()
	data, err := json.Marshal(message)
	if err != nil {
		return err
	}

	h.broadcast <- data
	return nil
}

// GetConnectedUsers 获取已连接的用户列表
func (h *WebSocketHub) GetConnectedUsers() []string {
	h.mutex.RLock()
	defer h.mutex.RUnlock()

	users := make([]string, 0, len(h.userClients))
	for userID := range h.userClients {
		users = append(users, userID)
	}
	return users
}

// readPump 读取客户端消息
func (c *WebSocketClient) readPump() {
	defer func() {
		c.hub.unregister <- c
		c.conn.Close()
	}()

	c.conn.SetReadLimit(512)
	c.conn.SetReadDeadline(time.Now().Add(60 * time.Second))
	c.conn.SetPongHandler(func(string) error {
		c.conn.SetReadDeadline(time.Now().Add(60 * time.Second))
		return nil
	})

	for {
		_, message, err := c.conn.ReadMessage()
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				global.GVA_LOG.Error("WebSocket error", zap.Error(err))
			}
			break
		}

		// 处理客户端发送的消息（如心跳等）
		var msg WebSocketMessage
		if err := json.Unmarshal(message, &msg); err == nil {
			switch msg.Type {
			case "ping":
				response := WebSocketMessage{
					Type: "pong",
					Data: map[string]interface{}{
						"timestamp": time.Now(),
					},
				}
				if data, err := json.Marshal(response); err == nil {
					select {
					case c.send <- data:
					default:
						close(c.send)
						return
					}
				}
			}
		}
	}
}

// writePump 向客户端发送消息
func (c *WebSocketClient) writePump() {
	ticker := time.NewTicker(54 * time.Second)
	defer func() {
		ticker.Stop()
		c.conn.Close()
	}()

	for {
		select {
		case message, ok := <-c.send:
			c.conn.SetWriteDeadline(time.Now().Add(10 * time.Second))
			if !ok {
				c.conn.WriteMessage(websocket.CloseMessage, []byte{})
				return
			}

			w, err := c.conn.NextWriter(websocket.TextMessage)
			if err != nil {
				return
			}
			w.Write(message)

			// 添加排队的消息到同一帧
			n := len(c.send)
			for i := 0; i < n; i++ {
				w.Write([]byte{'\n'})
				w.Write(<-c.send)
			}

			if err := w.Close(); err != nil {
				return
			}

		case <-ticker.C:
			c.conn.SetWriteDeadline(time.Now().Add(10 * time.Second))
			if err := c.conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				return
			}
		}
	}
}

// HandleWebSocket WebSocket连接处理器
func HandleWebSocket(c *gin.Context) {
	// 从查询参数或JWT中获取用户信息
	userID := c.Query("user_id")
	clientID := c.Query("client_id")

	if userID == "" || clientID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "user_id and client_id are required"})
		return
	}

	conn, err := wsUpgrader.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		global.GVA_LOG.Error("WebSocket upgrade failed", zap.Error(err))
		return
	}

	client := &WebSocketClient{
		conn:     conn,
		userID:   userID,
		clientID: clientID,
		send:     make(chan []byte, 256),
		hub:      GetWebSocketHub(),
	}

	client.hub.register <- client

	// 启动goroutines
	go client.writePump()
	go client.readPump()
}

// NotifyTransactionStatus 通知交易状态变更
func NotifyTransactionStatus(userID string, transactionData interface{}) {
	hub := GetWebSocketHub()
	message := WebSocketMessage{
		Type: "transaction_status",
		Data: transactionData,
	}

	err := hub.BroadcastToUser(userID, message)
	if err != nil {
		global.GVA_LOG.Warn("Failed to notify transaction status",
			zap.String("userID", userID),
			zap.Error(err))
	}
}

// NotifyAPDUMessage 通知APDU消息
func NotifyAPDUMessage(userID string, apduData interface{}) {
	hub := GetWebSocketHub()
	message := WebSocketMessage{
		Type: "apdu_message",
		Data: apduData,
	}

	err := hub.BroadcastToUser(userID, message)
	if err != nil {
		global.GVA_LOG.Warn("Failed to notify APDU message",
			zap.String("userID", userID),
			zap.Error(err))
	}
}

// NotifyClientStatus 通知客户端状态变更
func NotifyClientStatus(userID string, statusData interface{}) {
	hub := GetWebSocketHub()
	message := WebSocketMessage{
		Type: "client_status",
		Data: statusData,
	}

	err := hub.BroadcastToUser(userID, message)
	if err != nil {
		global.GVA_LOG.Warn("Failed to notify client status",
			zap.String("userID", userID),
			zap.Error(err))
	}
}
