package nfc_relay_admin

import (
	"strconv"

	"github.com/flipped-aurora/gin-vue-admin/server/global"
	"github.com/flipped-aurora/gin-vue-admin/server/model/common/response"
	"github.com/flipped-aurora/gin-vue-admin/server/model/nfc_relay_admin/request"
	"github.com/flipped-aurora/gin-vue-admin/server/service"
	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

type DatabaseAuditLogsApi struct{}

// CreateAuditLog 创建审计日志
// @Summary 创建审计日志
// @Description 创建新的审计日志记录
// @Tags NFC中继审计日志
// @Accept json
// @Produce json
// @Param data body request.CreateAuditLogRequest true "创建审计日志"
// @Success 200 {object} response.Response{} "创建成功"
// @Router /admin/nfc-relay/v1/audit-logs [post]
func (a *DatabaseAuditLogsApi) CreateAuditLog(c *gin.Context) {
	var req request.CreateAuditLogRequest
	err := c.ShouldBindJSON(&req)
	if err != nil {
		response.FailWithMessage(err.Error(), c)
		return
	}

	err = service.ServiceGroupApp.NfcRelayAdminServiceGroup.AuditLogService.CreateAuditLog(&req)
	if err != nil {
		global.GVA_LOG.Error("创建审计日志失败!", zap.Error(err))
		response.FailWithMessage("创建失败", c)
		return
	}

	response.OkWithMessage("创建成功", c)
}

// GetAuditLogList 获取审计日志列表
// @Summary 获取审计日志列表
// @Description 获取审计日志列表，支持分页和筛选
// @Tags NFC中继审计日志
// @Accept json
// @Produce json
// @Param data query request.AuditLogListRequest true "查询参数"
// @Success 200 {object} response.Response{data=response.PaginatedAuditLogResponse} "获取成功"
// @Router /admin/nfc-relay/v1/audit-logs [get]
func (a *DatabaseAuditLogsApi) GetAuditLogList(c *gin.Context) {
	var req request.AuditLogListRequest
	err := c.ShouldBindQuery(&req)
	if err != nil {
		response.FailWithMessage(err.Error(), c)
		return
	}

	list, err := service.ServiceGroupApp.NfcRelayAdminServiceGroup.AuditLogService.GetAuditLogList(&req)
	if err != nil {
		global.GVA_LOG.Error("获取审计日志列表失败!", zap.Error(err))
		response.FailWithMessage("获取失败", c)
		return
	}

	response.OkWithDetailed(list, "获取成功", c)
}

// GetAuditLogStats 获取审计日志统计信息
// @Summary 获取审计日志统计信息
// @Description 获取审计日志的统计数据，包括按级别、类型等维度的统计
// @Tags NFC中继审计日志
// @Accept json
// @Produce json
// @Success 200 {object} response.Response{data=response.AuditLogStatsResponse} "获取成功"
// @Router /admin/nfc-relay/v1/audit-logs/stats [get]
func (a *DatabaseAuditLogsApi) GetAuditLogStats(c *gin.Context) {
	stats, err := service.ServiceGroupApp.NfcRelayAdminServiceGroup.AuditLogService.GetAuditLogStats()
	if err != nil {
		global.GVA_LOG.Error("获取审计日志统计失败!", zap.Error(err))
		response.FailWithMessage("获取失败", c)
		return
	}

	response.OkWithDetailed(stats, "获取成功", c)
}

// DeleteOldAuditLogs 删除过期审计日志
// @Summary 删除过期审计日志
// @Description 删除指定天数之前的审计日志
// @Tags NFC中继审计日志
// @Accept json
// @Produce json
// @Param retentionDays query int true "保留天数"
// @Success 200 {object} response.Response{} "删除成功"
// @Router /admin/nfc-relay/v1/audit-logs/cleanup [delete]
func (a *DatabaseAuditLogsApi) DeleteOldAuditLogs(c *gin.Context) {
	retentionDaysStr := c.Query("retentionDays")
	if retentionDaysStr == "" {
		response.FailWithMessage("保留天数参数不能为空", c)
		return
	}

	retentionDays, err := strconv.Atoi(retentionDaysStr)
	if err != nil {
		response.FailWithMessage("保留天数参数格式错误", c)
		return
	}

	if retentionDays < 1 {
		response.FailWithMessage("保留天数必须大于0", c)
		return
	}

	err = service.ServiceGroupApp.NfcRelayAdminServiceGroup.AuditLogService.DeleteOldAuditLogs(retentionDays)
	if err != nil {
		global.GVA_LOG.Error("删除过期审计日志失败!", zap.Error(err))
		response.FailWithMessage("删除失败", c)
		return
	}

	response.OkWithMessage("删除成功", c)
}

// BatchCreateAuditLogs 批量创建审计日志
// @Summary 批量创建审计日志
// @Description 批量创建多条审计日志记录
// @Tags NFC中继审计日志
// @Accept json
// @Produce json
// @Param data body []request.CreateAuditLogRequest true "批量创建审计日志"
// @Success 200 {object} response.Response{} "创建成功"
// @Router /admin/nfc-relay/v1/audit-logs/batch [post]
func (a *DatabaseAuditLogsApi) BatchCreateAuditLogs(c *gin.Context) {
	var logs []request.CreateAuditLogRequest
	err := c.ShouldBindJSON(&logs)
	if err != nil {
		response.FailWithMessage(err.Error(), c)
		return
	}

	err = service.ServiceGroupApp.NfcRelayAdminServiceGroup.AuditLogService.BatchCreateAuditLogs(logs)
	if err != nil {
		global.GVA_LOG.Error("批量创建审计日志失败!", zap.Error(err))
		response.FailWithMessage("创建失败", c)
		return
	}

	response.OkWithMessage("批量创建成功", c)
}
