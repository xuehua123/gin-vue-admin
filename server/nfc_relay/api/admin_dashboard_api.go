package api

import (
	"github.com/flipped-aurora/gin-vue-admin/server/global"
	"github.com/flipped-aurora/gin-vue-admin/server/model/common/response"
	"github.com/flipped-aurora/gin-vue-admin/server/nfc_relay/service"
	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

// AdminDashboardApi 结构体，用于仪表盘相关的API处理
type AdminDashboardApi struct{}

var adminDashboardService = service.AdminDashboardService{}

// GetDashboardStats
// @Tags NFCRelayAdmin
// @Summary 获取NFC Relay仪表盘统计数据
// @Security ApiKeyAuth
// @Produce  application/json
// @Success 200 {object} response.Response{data=admin_response.DashboardStatsData,msg=string}  "获取成功"
// @Router /admin/nfc-relay/v1/dashboard-stats [get]
func (a *AdminDashboardApi) GetDashboardStats(c *gin.Context) {
	stats, err := adminDashboardService.GetDashboardStats()
	if err != nil {
		global.GVA_LOG.Error("获取仪表盘统计数据失败!", zap.Error(err))
		response.FailWithMessage("获取仪表盘统计数据失败: "+err.Error(), c)
		return
	}
	response.OkWithDetailed(stats, "获取仪表盘统计数据成功", c)
}
