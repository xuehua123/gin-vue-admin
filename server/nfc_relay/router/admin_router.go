package router

import (
	"github.com/flipped-aurora/gin-vue-admin/server/nfc_relay/api" // 导入API包
	"github.com/gin-gonic/gin"
)

// NFCRelayAdminRouter 结构体，用于NFC Relay管理后台的路由
type NFCRelayAdminRouter struct{}

// InitNFCRelayAdminRouter 初始化NFC Relay管理后台路由
// 参数 Router *gin.RouterGroup: 这个参数应该是外部传入的，带有认证和授权中间件的路由组
func (r *NFCRelayAdminRouter) InitNFCRelayAdminRouter(Router *gin.RouterGroup) {
	adminRouter := Router.Group("nfc-relay/v1") // 在传入的Group下再创建一个子Group
	// 如果需要，可以在这里为 adminRouter 添加更细致的NFC中继管理员角色授权中间件
	// adminRouter.Use(middleware.CasbinHandler()) // 示例：如果需要基于角色的访问控制

	var adminDashboardApi = api.AdminDashboardApi{}
	{
		adminRouter.GET("dashboard-stats", adminDashboardApi.GetDashboardStats)
		// 后续其他NFC Relay管理API也在此处注册
	}
}
