package router

import (
	"github.com/flipped-aurora/gin-vue-admin/server/nfc_relay/api"
	"github.com/flipped-aurora/gin-vue-admin/server/nfc_relay/handler"
	"github.com/gin-gonic/gin"
)

// RouterGroup 是 nfc_relay 模块的路由组结构体。
// 目前为空，可以根据需要添加字段，例如API服务的实例。
type RouterGroup struct {
}

// InitNFCRelayRouter 初始化 NFC 中继相关的 WebSocket 路由 (通常为公开访问或有特定认证)
func InitNFCRelayRouter(Router *gin.RouterGroup) {
	// 为 NFC 中继功能创建一个新的路由组，例如 "/nfc"
	// 最终的 WebSocket 端点将是 /nfc/relay
	nfcRouter := Router.Group("nfc") // WebSocket 路由通常不直接在 /api 前缀下
	{
		// 定义 WebSocket 连接端点
		// GET 请求到 /nfc/relay 将会尝试升级到 WebSocket 连接
		nfcRouter.GET("relay", handler.WSConnectionHandler)
	}

	// 如果有其他与 NFC 中继相关的 HTTP API (例如获取会话列表、状态等)，也可以在这里定义
	// 例如:
	// nfcRouter.GET("sessions", handler.GetSessionsHandler) // 需要创建 GetSessionsHandler
}

// NFCRelayAdminApiRouter 结构体 (如果需要更复杂的API分组，可以实例化)
// var nfcRelayAdminApi = api.AdminDashboardApi{} // 移动到下面的方法作用域内，避免包级别变量冲突

// InitNFCRelayAdminApiRouter 初始化 NFC Relay 管理后台的 API 路由
// 这些路由应该在 PrivateGroup 下，以利用JWT和Casbin中间件
func InitNFCRelayAdminApiRouter(Router *gin.RouterGroup) {
	// 创建 /admin/nfc-relay/v1 路由组
	// Router 参数预期是 PrivateGroup，已经应用了JWTAuth和CasbinHandler
	adminApiRouter := Router.Group("admin/nfc-relay/v1")
	// 如果需要更细致的NFC中继管理员角色授权中间件，可以在此Group上添加
	// adminApiRouter.Use(middleware.YourNFCRelayAdminRoleMiddleware()) // 示例

	var adminDashboardApi = api.AdminDashboardApi{} // 实例化API处理器
	{
		adminApiRouter.GET("dashboard-stats", adminDashboardApi.GetDashboardStats)
		// 后续其他NFC Relay管理API也在此处注册
	}
}
