package handler

import (
	"bytes"
	"errors"
	"io"
	"net"
	"time"

	"github.com/flipped-aurora/gin-vue-admin/server/global"
	"github.com/flipped-aurora/gin-vue-admin/server/nfc_relay/protocol"
	"github.com/google/uuid"
	"github.com/gorilla/websocket"
	"go.uber.org/zap"
)

const (
// writeWait 是允许向对端写入消息的时间。
// writeWait = 10 * time.Second // Replaced by config

// pongWait 是允许从对端读取下一个 pong 消息的时间。
// pongWait = 60 * time.Second // Replaced by config

// pingPeriod 是向对端发送 ping 的周期。必须小于 pongWait。
// pingPeriod = (pongWait * 9) / 10 // Will be derived from config

// maxMessageSize 是允许从对端接收的最大消息大小。
// maxMessageSize = 2048 // Replaced by config
)

var (
	newline = []byte{'\n'}
	space   = []byte{' '}
)

// ProcessableMessage 包装了从客户端收到的消息及其来源客户端，
// 以便 Hub 可以识别消息发送者并进行相应处理。
type ProcessableMessage struct {
	Client     *Client // 发送消息的客户端
	RawMessage []byte  // 原始消息字节
}

// WebsocketConnection 定义了 Client 所需的 WebSocket 连接方法。
// 这允许在测试中使用模拟连接。
type WebsocketConnection interface {
	Close() error
	SetReadLimit(limit int64)
	SetReadDeadline(t time.Time) error
	SetWriteDeadline(t time.Time) error
	SetPongHandler(h func(string) error)
	ReadMessage() (messageType int, p []byte, err error)
	NextWriter(messageType int) (io.WriteCloser, error)
	WriteMessage(messageType int, data []byte) error
	RemoteAddr() net.Addr
}

// Client 是 WebSocket 连接和 Hub 之间的中间人。
type Client struct {
	hub *Hub

	// conn 是 WebSocket 连接。
	conn WebsocketConnection

	// send 是一个缓冲的出站消息通道。
	send chan []byte

	// Client特定数据
	ID            string            // 唯一的客户端ID
	Role          string            // 角色 ("card" 或 "pos"), 在特定流程后由 Hub 设置 (此字段似乎用途不多了，主要看CurrentRole)
	SessionID     string            // 关联的会话ID, 在特定流程后由 Hub 设置
	UserID        string            // 关联的用户ID
	Authenticated bool              // 客户端是否已通过 WebSocket 连接认证
	CurrentRole   protocol.RoleType // 由 DeclareRole 设置的当前角色 ("provider", "receiver", "none")
	IsOnline      bool              // 由 DeclareRole 设置的在线状态 (主要用于 provider)
	DisplayName   string            // 由 provider 客户端在 DeclareRole 时提供的显示名称
}

// GetID 返回客户端的唯一ID。
func (c *Client) GetID() string {
	return c.ID
}

// GetRole 返回客户端的当前声明角色。
// 这是 ClientInfoProvider 接口的一部分。
func (c *Client) GetRole() string {
	return string(c.CurrentRole) // 返回 CurrentRole 而不是旧的 Role 字段
}

// GetSessionID 返回客户端关联的会话ID。
func (c *Client) GetSessionID() string {
	return c.SessionID
}

// GetUserID 返回客户端关联的用户ID。
func (c *Client) GetUserID() string {
	return c.UserID
}

// GetCurrentRole 返回客户端当前的声明角色
func (c *Client) GetCurrentRole() protocol.RoleType {
	return c.CurrentRole
}

// Send 将消息发送到客户端的出站通道。
// 这是 ClientInfoProvider 接口的一部分。
func (c *Client) Send(message []byte) (err error) {
	defer func() {
		if r := recover(); r != nil {
			// 检查 panic 是否因为 "send on closed channel"
			// 注意：依赖具体的 panic 信息字符串可能不够健壮，但对于这个特定场景是常见的做法
			if e, ok := r.(error); ok && e.Error() == "send on closed channel" {
				global.GVA_LOG.Warn("客户端 Send 方法：尝试向已关闭的 send 通道发送消息", zap.String("clientID", c.ID))
				err = errors.New("failed to send message to client: channel is closed")
			} else {
				// 如果是其他 panic，重新抛出
				panic(r)
			}
		}
	}()

	select {
	case c.send <- message:
		return nil
	default:
		// 如果 default 被执行，说明通道已满（因为如果通道关闭，上面的 case 会 panic 被 recover 捕获）
		// 或者 c.send 为 nil (虽然本代码中不太可能，因为 make 初始化了)
		global.GVA_LOG.Warn("客户端 Send 方法：发送消息到 c.send 通道失败（已满）", zap.String("clientID", c.ID))
		return errors.New("failed to send message to client: channel full") // 调整错误信息
	}
}

// NewClient 创建一个新的 Client 实例。
// hub 被传递进来，以便客户端可以注册自己并发送消息。
func NewClient(hub *Hub, conn WebsocketConnection) *Client {
	return &Client{
		hub:  hub,
		conn: conn,
		send: make(chan []byte, 256), // 出站消息的缓冲通道
		ID:   uuid.NewString(),       // 为客户端生成一个唯一的ID
	}
}

// readPump 将消息从 WebSocket 连接泵送到 Hub。
//
// 应用程序为每个连接运行一个 readPump goroutine。
// 应用程序通过从此 goroutine 执行所有读取操作来确保连接上最多只有一个读取器。
func (c *Client) readPump() {
	defer func() {
		c.hub.unregister <- c // 从 Hub 注销
		c.conn.Close()
		global.GVA_LOG.Info("客户端 readPump：已注销并关闭连接", zap.String("clientID", c.ID), zap.String("remoteAddr", c.conn.RemoteAddr().String()))
	}()

	pongWait := time.Duration(global.GVA_CONFIG.NfcRelay.WebsocketPongWaitSec) * time.Second
	maxMessageSize := int64(global.GVA_CONFIG.NfcRelay.WebsocketMaxMessageBytes)
	if maxMessageSize <= 0 {
		maxMessageSize = 2048 // Default fallback
	}

	c.conn.SetReadLimit(maxMessageSize)
	_ = c.conn.SetReadDeadline(time.Now().Add(pongWait)) // 初始读取截止时间
	c.conn.SetPongHandler(func(string) error {           // Pong 消息重置读取截止时间
		_ = c.conn.SetReadDeadline(time.Now().Add(pongWait))
		return nil
	})

	for {
		messageType, messageBytes, err := c.conn.ReadMessage()
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure, websocket.CloseNoStatusReceived) {
				global.GVA_LOG.Error("客户端 readPump：意外关闭错误", zap.Error(err), zap.String("clientID", c.ID))
			} else if e, ok := err.(*websocket.CloseError); ok {
				global.GVA_LOG.Info("客户端 readPump：WebSocket 连接由客户端关闭", zap.Uint16("code", uint16(e.Code)), zap.String("text", e.Text), zap.String("clientID", c.ID))
			} else {
				global.GVA_LOG.Error("客户端 readPump：读取错误", zap.Error(err), zap.String("clientID", c.ID))
			}
			break
		}

		if messageType == websocket.TextMessage {
			trimmedMessage := bytes.TrimSpace(bytes.Replace(messageBytes, newline, space, -1))
			global.GVA_LOG.Debug("客户端 readPump：收到文本消息", zap.String("clientID", c.ID), zap.ByteString("message", trimmedMessage))

			// 将消息和客户端信息一起发送到 Hub 进行处理
			c.hub.processMessage <- ProcessableMessage{Client: c, RawMessage: trimmedMessage}

		} else if messageType == websocket.BinaryMessage {
			global.GVA_LOG.Debug("客户端 readPump：收到二进制消息", zap.String("clientID", c.ID), zap.Int("size", len(messageBytes)))
			// 目前不处理二进制消息，可以考虑发送错误或忽略
		}
	}
}

// writePump 将消息从 Hub 泵送到 WebSocket 连接。
//
// 为每个连接启动一个运行 writePump 的 goroutine。
// 应用程序通过从此 goroutine 执行所有写入操作来确保连接上最多只有一个写入器。
func (c *Client) writePump() {
	pongWait := time.Duration(global.GVA_CONFIG.NfcRelay.WebsocketPongWaitSec) * time.Second
	pingPeriod := (pongWait * 9) / 10
	if pingPeriod <= 0 { // Ensure pingPeriod is positive
		pingPeriod = 54 * time.Second // Default based on 60s pongWait
	}
	writeWait := time.Duration(global.GVA_CONFIG.NfcRelay.WebsocketWriteWaitSec) * time.Second

	ticker := time.NewTicker(pingPeriod)
	defer func() {
		ticker.Stop()
		c.conn.Close()
		global.GVA_LOG.Info("客户端 writePump：Ticker 已停止并关闭连接", zap.String("clientID", c.ID), zap.String("remoteAddr", c.conn.RemoteAddr().String()))
	}()
	for {
		select {
		case message, ok := <-c.send:
			if !ok {
				// Hub 关闭了通道。
				global.GVA_LOG.Info("客户端 writePump：Hub 关闭了客户端的 send 通道", zap.String("clientID", c.ID))
				_ = c.conn.WriteMessage(websocket.CloseMessage, []byte{}) // 发送关闭消息
				return
			}

			_ = c.conn.SetWriteDeadline(time.Now().Add(writeWait))
			w, err := c.conn.NextWriter(websocket.TextMessage) // 假设所有消息都是文本 (JSON)
			if err != nil {
				global.GVA_LOG.Error("客户端 writePump：获取 NextWriter 失败", zap.Error(err), zap.String("clientID", c.ID))
				return
			}
			_, err = w.Write(message)
			if err != nil {
				global.GVA_LOG.Error("客户端 writePump：向 Writer 写入消息失败", zap.Error(err), zap.String("clientID", c.ID))
				return
			}

			// 将排队的消息添加到当前的 WebSocket 消息中。
			// 这是一种优化，如果可用，可以在一个 WebSocket 帧中发送多个排队的消息。
			n := len(c.send)
			for i := 0; i < n; i++ {
				_, _ = w.Write(newline) // 可选：如果以这种方式发送多个JSON对象，则用作分隔符，尽管通常每个消息一个JSON
				_, _ = w.Write(<-c.send)
			}

			if err := w.Close(); err != nil {
				global.GVA_LOG.Error("客户端 writePump：关闭 Writer 失败", zap.Error(err), zap.String("clientID", c.ID))
				return
			}
		case <-ticker.C:
			_ = c.conn.SetWriteDeadline(time.Now().Add(writeWait))
			if err := c.conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				global.GVA_LOG.Error("客户端 writePump：发送 ping 失败", zap.Error(err), zap.String("clientID", c.ID))
				return
			}
			global.GVA_LOG.Debug("客户端 writePump：已发送 ping", zap.String("clientID", c.ID))
		}
	}
}
