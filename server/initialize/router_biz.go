package initialize

import (
	"github.com/flipped-aurora/gin-vue-admin/server/router"
	"github.com/gin-gonic/gin"
)

// 占位方法，保证文件可以正确加载，避免go空变量检测报错，请勿删除。
func holder(routers ...*gin.RouterGroup) {
	_ = routers
	_ = router.RouterGroupApp
}

func initBizRouter(routers ...*gin.RouterGroup) {
	privateGroup := routers[0]
	publicGroup := routers[1]

	// 注册NFC中继管理路由
	nfcRelayAdminRouter := router.RouterGroupApp.NfcRelayAdmin
	nfcRelayAdminRouter.InitNfcRelayAdminRouter(privateGroup) // 需要认证的管理功能

	holder(publicGroup, privateGroup)
}
