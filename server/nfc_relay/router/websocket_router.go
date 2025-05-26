package router

import (
	"github.com/flipped-aurora/gin-vue-admin/server/nfc_relay/handler"
	"github.com/gin-gonic/gin"
)

// RouterGroup 是 nfc_relay 模块的路由组结构体。
// 目前为空，可以根据需要添加字段，例如API服务的实例。
type RouterGroup struct {
}

// InitNFCRelayRouter 初始化 NFC 中继相关的 WebSocket 路由
func InitNFCRelayRouter(Router *gin.RouterGroup) {
	// 为 NFC 中继功能创建一个新的路由组，例如 "/nfc"
	// 最终的 WebSocket 端点将是 /nfc/relay
	nfcRouter := Router.Group("nfc")
	{
		// 定义 WebSocket 连接端点
		// GET 请求到 /nfc/relay 将会尝试升级到 WebSocket 连接
		nfcRouter.GET("relay", handler.WSConnectionHandler)
	}

	// 如果有其他与 NFC 中继相关的 HTTP API (例如获取会话列表、状态等)，也可以在这里定义
	// 例如:
	// nfcRouter.GET("sessions", handler.GetSessionsHandler) // 需要创建 GetSessionsHandler
}
