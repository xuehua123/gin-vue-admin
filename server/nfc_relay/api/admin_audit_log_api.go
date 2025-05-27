package api

import (
	"github.com/flipped-aurora/gin-vue-admin/server/global"
	"github.com/flipped-aurora/gin-vue-admin/server/model/common/response"
	"github.com/flipped-aurora/gin-vue-admin/server/nfc_relay/model/admin_request"
	"github.com/flipped-aurora/gin-vue-admin/server/nfc_relay/service"
	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

// AdminAuditLogApi 审计日志API处理器
type AdminAuditLogApi struct{}

var adminAuditLogService = service.AdminAuditLogService{}

// GetAuditLogs
// @Tags NFCRelayAdmin
// @Summary 获取NFC Relay审计日志列表
// @Security ApiKeyAuth
// @Produce application/json
// @Param page query int false "页码"
// @Param pageSize query int false "每页条数"
// @Param eventType query string false "事件类型"
// @Param userID query string false "用户ID"
// @Param sessionID query string false "会话ID"
// @Param clientID query string false "客户端ID（匹配发起方或响应方）"
// @Param startTime query string false "开始时间（ISO8601格式，例如：2023-10-27T10:00:00Z）"
// @Param endTime query string false "结束时间（ISO8601格式）"
// @Success 200 {object} response.Response{data=admin_response.PaginatedAuditLogResponse,msg=string} "获取成功"
// @Router /admin/nfc-relay/v1/audit-logs [get]
func (a *AdminAuditLogApi) GetAuditLogs(c *gin.Context) {
	var params admin_request.AuditLogListParams
	// 绑定查询参数
	if err := c.ShouldBindQuery(&params); err != nil {
		global.GVA_LOG.Error("获取审计日志列表请求参数错误!", zap.Error(err))
		response.FailWithMessage("请求参数错误: "+err.Error(), c)
		return
	}

	// 验证分页参数
	if params.Page < 1 {
		params.Page = 1
	}
	if params.PageSize < 1 || params.PageSize > 100 {
		params.PageSize = 10 // 限制每页最大条数
	}

	// 调用服务获取审计日志
	global.GVA_LOG.Info("开始获取审计日志",
		zap.String("eventType", params.EventType),
		zap.String("userID", params.UserID),
		zap.String("sessionID", params.SessionID),
		zap.String("clientID", params.ClientID),
		zap.String("startTime", params.StartTime),
		zap.String("endTime", params.EndTime),
		zap.Int("page", params.Page),
		zap.Int("pageSize", params.PageSize),
	)

	auditLogs, err := adminAuditLogService.GetAuditLogs(params)
	if err != nil {
		global.GVA_LOG.Error("获取审计日志失败!", zap.Error(err))
		response.FailWithMessage("获取审计日志失败: "+err.Error(), c)
		return
	}

	global.GVA_LOG.Info("获取审计日志成功",
		zap.Int("total", auditLogs.Total),
		zap.Int("page", auditLogs.Page),
		zap.Int("pageSize", auditLogs.PageSize),
		zap.Int("listSize", len(auditLogs.List)),
	)

	response.OkWithDetailed(auditLogs, "获取审计日志成功", c)
}
