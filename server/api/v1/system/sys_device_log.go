package system

import (
	"github.com/flipped-aurora/gin-vue-admin/server/global"
	"github.com/flipped-aurora/gin-vue-admin/server/model/common/response"
	systemReq "github.com/flipped-aurora/gin-vue-admin/server/model/system/request"
	systemRes "github.com/flipped-aurora/gin-vue-admin/server/model/system/response"
	"github.com/flipped-aurora/gin-vue-admin/server/service"
	"github.com/flipped-aurora/gin-vue-admin/server/utils"
	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

var _ = systemRes.DeviceLogStats{}

type DeviceLogApi struct{}

// 使用已注册的服务
var deviceLogServiceApi = service.ServiceGroupApp.SystemServiceGroup.DeviceLogService

// GetDeviceLogsList
// @Tags      DeviceLog
// @Summary   分页获取设备日志列表
// @Security  ApiKeyAuth
// @accept    application/json
// @Produce   application/json
// @Param     data  body      systemReq.GetDeviceLogsRequest                               true  "页码, 每页大小, 筛选条件"
// @Success   200   {object}  response.Response{data=response.PageResult,msg=string}  "分页获取设备日志列表"
// @Router    /deviceLog/getDeviceLogsList [post]
func (d *DeviceLogApi) GetDeviceLogsList(c *gin.Context) {
	var pageInfo systemReq.GetDeviceLogsRequest
	err := c.ShouldBindJSON(&pageInfo)
	if err != nil {
		response.FailWithMessage(err.Error(), c)
		return
	}
	err = utils.Verify(pageInfo, utils.PageInfoVerify)
	if err != nil {
		response.FailWithMessage(err.Error(), c)
		return
	}

	list, total, err := deviceLogServiceApi.GetDeviceLogsList(pageInfo)
	if err != nil {
		global.GVA_LOG.Error("获取设备日志失败!", zap.Error(err))
		response.FailWithMessage("获取设备日志失败", c)
		return
	}

	response.OkWithDetailed(response.PageResult{
		List:     list,
		Total:    total,
		Page:     pageInfo.Page,
		PageSize: pageInfo.PageSize,
	}, "获取成功", c)
}

// GetDeviceLogStats
// @Tags      DeviceLog
// @Summary   获取设备日志统计
// @Security  ApiKeyAuth
// @accept    application/json
// @Produce   application/json
// @Param     userId  query     string                                                   false  "用户ID"
// @Success   200     {object}  response.Response{data=systemRes.DeviceLogStats,msg=string}  "获取设备日志统计"
// @Router    /deviceLog/getDeviceLogStats [get]
func (d *DeviceLogApi) GetDeviceLogStats(c *gin.Context) {
	userId := c.Query("userId")

	stats, err := deviceLogServiceApi.GetDeviceLogStats(userId)
	if err != nil {
		global.GVA_LOG.Error("获取设备日志统计失败!", zap.Error(err))
		response.FailWithMessage("获取设备日志统计失败", c)
		return
	}

	response.OkWithData(stats, c)
}

// ForceLogoutDevice
// @Tags      DeviceLog
// @Summary   强制设备下线
// @Security  ApiKeyAuth
// @accept    application/json
// @Produce   application/json
// @Param     data  body      systemReq.ForceLogoutRequest                        true  "用户ID, 客户端ID, 下线原因"
// @Success   200   {object}  response.Response{msg=string}                      "强制设备下线"
// @Router    /deviceLog/forceLogoutDevice [post]
func (d *DeviceLogApi) ForceLogoutDevice(c *gin.Context) {
	var req systemReq.ForceLogoutRequest
	err := c.ShouldBindJSON(&req)
	if err != nil {
		response.FailWithMessage(err.Error(), c)
		return
	}

	err = deviceLogServiceApi.ForceLogoutDevice(req.UserID, req.ClientID, req.Reason)
	if err != nil {
		global.GVA_LOG.Error("强制下线失败!", zap.Error(err))
		response.FailWithMessage("强制下线失败: "+err.Error(), c)
		return
	}

	response.OkWithMessage("强制下线成功", c)
}
