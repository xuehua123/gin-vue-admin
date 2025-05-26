package handler

import (
	"net/http"

	"github.com/flipped-aurora/gin-vue-admin/server/global"
	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
	"go.uber.org/zap"
)

var upgrader = websocket.Upgrader{
	// ReadBufferSize 和 WriteBufferSize 指定I/O缓冲区的大小。
	// 默认值通常足够，但可以根据需要调整。
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	// CheckOrigin 函数用于验证请求的来源，防止跨站 WebSocket 劫持。
	// 在生产环境中，应确保此函数正确验证来源。
	// 出于开发目的，暂时允许所有来源。
	CheckOrigin: func(r *http.Request) bool {
		// TODO: 在生产环境中实施严格的来源检查
		// 例如:
		// origin := r.Header.Get("Origin")
		// return origin == "https://yourfrontend.com"
		return true
	},
}

// WSConnectionHandler 处理 WebSocket 连接请求
func WSConnectionHandler(c *gin.Context) {
	// 尝试将 HTTP 连接升级到 WebSocket 连接
	conn, err := upgrader.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		global.GVA_LOG.Error("Failed to upgrade WebSocket connection", zap.Error(err))
		// Gin 通常会自动处理响应，但如果升级失败，客户端不会收到标准的 WebSocket 错误。
		// 根据 upgrader 的文档，如果 Upgrade 方法返回错误，它已经向客户端发送了HTTP错误响应。
		return
	}
	// defer conn.Close() // Client的readPump和writePump的defer中会负责关闭

	global.GVA_LOG.Info("WebSocket connection successfully upgraded", zap.String("remoteAddr", conn.RemoteAddr().String()))

	client := NewClient(GlobalRelayHub, conn) // 使用全局的 Hub 实例
	client.hub.register <- client

	// Allow collection of memory referenced by the caller by doing all work in
	// new goroutines.
	go client.writePump()
	go client.readPump() // readPump 会阻塞，直到连接关闭

	// 当 readPump 退出时 (连接关闭或错误)，此处的 WSConnectionHandler 协程也会结束。
	// client 的 unregister 和 conn.Close() 由 readPump 的 defer 语句处理。
}
