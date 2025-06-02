package system

import (
	v1 "github.com/flipped-aurora/gin-vue-admin/server/api/v1"
	"github.com/flipped-aurora/gin-vue-admin/server/middleware"
	"github.com/gin-gonic/gin"
)

type DeviceLogRouter struct{}

// InitDeviceLogRouter 初始化设备日志路由
func (s *DeviceLogRouter) InitDeviceLogRouter(Router *gin.RouterGroup, PublicRouter *gin.RouterGroup) {
	deviceLogApi := v1.ApiGroupApp.SystemApiGroup.DeviceLogApi
	deviceLogRouter := Router.Group("deviceLog").Use(middleware.OperationRecord())
	deviceLogRouterWithoutRecord := Router.Group("deviceLog")
	{
		deviceLogRouter.POST("forceLogoutDevice", deviceLogApi.ForceLogoutDevice) // 强制设备下线
	}
	{
		deviceLogRouterWithoutRecord.POST("getDeviceLogsList", deviceLogApi.GetDeviceLogsList) // 分页获取设备日志列表
		deviceLogRouterWithoutRecord.GET("getDeviceLogStats", deviceLogApi.GetDeviceLogStats)  // 获取设备日志统计
	}
}
