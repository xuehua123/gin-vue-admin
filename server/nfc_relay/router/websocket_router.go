package router

import (
	"fmt"
	"math/rand"
	"net/http"
	"time"

	"github.com/flipped-aurora/gin-vue-admin/server/global"
	"github.com/flipped-aurora/gin-vue-admin/server/nfc_relay/handler"
	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
	"go.uber.org/zap"
)

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	CheckOrigin: func(r *http.Request) bool {
		// 在生产环境中，应该检查请求的Origin头部以防止CSRF攻击
		// 这里暂时允许所有Origin，后续可以根据需要修改
		return true
	},
}

// TLSOnlyMiddleware 强制使用TLS连接的中间件
func TLSOnlyMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		// 检查是否启用了强制TLS
		if global.GVA_CONFIG.NfcRelay.Security.ForceTLS {
			// 检查连接是否为TLS
			if c.Request.TLS == nil {
				global.GVA_LOG.Warn("拒绝非TLS WebSocket连接",
					zap.String("remoteAddr", c.ClientIP()),
					zap.String("userAgent", c.GetHeader("User-Agent")),
				)
				c.JSON(http.StatusUpgradeRequired, gin.H{
					"error": "此服务要求使用安全连接(WSS)，请使用 wss:// 协议",
					"code":  "TLS_REQUIRED",
				})
				c.Abort()
				return
			}

			// 验证TLS版本
			if c.Request.TLS.Version < 0x0304 { // TLS 1.3
				global.GVA_LOG.Warn("TLS版本过低",
					zap.String("remoteAddr", c.ClientIP()),
					zap.Uint16("tlsVersion", c.Request.TLS.Version),
				)
				c.JSON(http.StatusUpgradeRequired, gin.H{
					"error": "TLS版本过低，要求TLS 1.3或更高版本",
					"code":  "TLS_VERSION_TOO_LOW",
				})
				c.Abort()
				return
			}

			global.GVA_LOG.Info("WSS连接验证通过",
				zap.String("remoteAddr", c.ClientIP()),
				zap.Uint16("tlsVersion", c.Request.TLS.Version),
			)
		}

		c.Next()
	}
}

// RouterGroup 是 nfc_relay 模块的路由组结构体。
// 目前为空，可以根据需要添加字段，例如API服务的实例。
type RouterGroup struct {
	// 这里可以加入一些针对nfc_relay特有的配置或中间件
}

// InitNFCRelayRouter 初始化 NFC 中继相关的 WebSocket 路由 (通常为公开访问或有特定认证)
func InitNFCRelayRouter(Router *gin.RouterGroup) {
	// 为 NFC 中继功能创建一个新的路由组
	// 最终的 WebSocket 端点将是 /ws/nfc-relay/realtime
	wsRouter := Router.Group("ws/nfc-relay") // WebSocket 路由通常不直接在 /api 前缀下
	{
		// NFC客户端连接端点 - 需要客户端认证
		// GET 请求到 /ws/nfc-relay/client 将会尝试升级到 WebSocket 连接
		// 用于真实的NFC设备和应用程序连接
		wsRouter.GET("client", handler.WSConnectionHandler)

		// 管理界面实时数据端点 - 支持多种数据类型订阅
		// GET 请求到 /ws/nfc-relay/realtime 将会尝试升级到 WebSocket 连接
		// 客户端可以通过发送订阅消息来选择接收的数据类型：
		// - dashboard: 仪表盘数据
		// - clients: 客户端状态
		// - sessions: 会话信息
		// - metrics: 系统指标
		wsRouter.GET("realtime", handler.AdminWSConnectionHandler)

		// 以下端点为扩展预留，可以根据需要实现独立的WebSocket处理器
		// wsRouter.GET("logs", handler.LogStreamHandler)       // 专用日志流
		// wsRouter.GET("apdu", handler.ApduMonitorHandler)     // 专用APDU监控
		// wsRouter.GET("metrics", handler.MetricsHandler)      // 专用系统指标
	}

	// 如果有其他与 NFC 中继相关的 HTTP API (例如获取会话列表、状态等)，也可以在这里定义
	// 例如:
	// nfcRouter.GET("sessions", handler.GetSessionsHandler) // 需要创建 GetSessionsHandler
}

// 注意：旧版的 InitNFCRelayAdminApiRouter 已移除
// 所有NFC管理API现在通过 router/nfc_relay_admin/nfc_relay_admin.go 中的 InitNfcRelayAdminRouter 注册

func InitNFCRelayWebSocketRouter(Router *gin.RouterGroup) {
	nfcRelayRouter := Router.Group("nfc-relay")

	// 应用TLS检查中间件
	nfcRelayRouter.Use(TLSOnlyMiddleware())

	// WebSocket 路由，用于实时数据传输 - 使用现有的处理器
	nfcRelayRouter.GET("/realtime", handler.WSConnectionHandler)
}

// generateClientID 生成唯一的客户端ID
func generateClientID() string {
	// 这里可以使用UUID或其他方法生成唯一ID
	// 简单起见，使用时间戳+随机数
	return fmt.Sprintf("client_%d_%d", time.Now().UnixNano(), rand.Intn(10000))
}
