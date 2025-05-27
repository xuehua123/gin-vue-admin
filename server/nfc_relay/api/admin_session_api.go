package api

import (
	"strconv"

	"github.com/flipped-aurora/gin-vue-admin/server/global"
	"github.com/flipped-aurora/gin-vue-admin/server/model/common/response"
	"github.com/flipped-aurora/gin-vue-admin/server/nfc_relay/model/admin_request"
	"github.com/flipped-aurora/gin-vue-admin/server/nfc_relay/service"
	"github.com/flipped-aurora/gin-vue-admin/server/utils"
	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

// AdminSessionApi 结构体，用于会话管理相关的API处理
type AdminSessionApi struct{}

var adminSessionService = service.AdminSessionService{}

// GetSessionList
// @Tags NFCRelayAdmin
// @Summary 获取NFC Relay会话列表
// @Security ApiKeyAuth
// @Produce application/json
// @Param page query int false "页码"
// @Param pageSize query int false "每页条数"
// @Param sessionId query string false "会话ID（模糊匹配）"
// @Param participantClientId query string false "参与方客户端ID（模糊匹配）"
// @Param participantUserId query string false "参与方用户ID（模糊匹配）"
// @Param status query string false "会话状态（paired/waiting_for_pairing/terminated）"
// @Success 200 {object} response.Response{data=admin_response.PaginatedSessionListResponse,msg=string} "获取成功"
// @Router /admin/nfc-relay/v1/sessions [get]
func (a *AdminSessionApi) GetSessionList(c *gin.Context) {
	var params admin_request.SessionListParams
	// 绑定查询参数
	if err := c.ShouldBindQuery(&params); err != nil {
		global.GVA_LOG.Error("获取会话列表请求参数错误!", zap.Error(err))
		response.FailWithMessage("请求参数错误: "+err.Error(), c)
		return
	}

	// 调用服务获取会话列表
	sessionList, err := adminSessionService.GetSessionList(params)
	if err != nil {
		global.GVA_LOG.Error("获取会话列表失败!", zap.Error(err))
		response.FailWithMessage("获取会话列表失败: "+err.Error(), c)
		return
	}

	response.OkWithDetailed(sessionList, "获取会话列表成功", c)
}

// GetSessionDetail
// @Tags NFCRelayAdmin
// @Summary 获取NFC Relay单个会话详情
// @Security ApiKeyAuth
// @Produce application/json
// @Param sessionID path string true "会话ID"
// @Success 200 {object} response.Response{data=admin_response.SessionDetailsResponse,msg=string} "获取成功"
// @Router /admin/nfc-relay/v1/sessions/{sessionID}/details [get]
func (a *AdminSessionApi) GetSessionDetail(c *gin.Context) {
	sessionID := c.Param("sessionID")
	if sessionID == "" {
		response.FailWithMessage("会话ID不能为空", c)
		return
	}

	// 调用服务获取会话详情
	sessionDetail, err := adminSessionService.GetSessionDetail(sessionID)
	if err != nil {
		global.GVA_LOG.Error("获取会话详情失败!", zap.Error(err))
		response.FailWithMessage("获取会话详情失败: "+err.Error(), c)
		return
	}

	response.OkWithDetailed(sessionDetail, "获取会话详情成功", c)
}

// TerminateSession
// @Tags NFCRelayAdmin
// @Summary 终止NFC Relay会话
// @Security ApiKeyAuth
// @Produce application/json
// @Param sessionID path string true "会话ID"
// @Param request body admin_request.TerminateSessionRequest false "终止原因"
// @Success 200 {object} response.Response{msg=string} "操作成功"
// @Router /admin/nfc-relay/v1/sessions/{sessionID}/terminate [post]
func (a *AdminSessionApi) TerminateSession(c *gin.Context) {
	sessionID := c.Param("sessionID")
	if sessionID == "" {
		response.FailWithMessage("会话ID不能为空", c)
		return
	}

	var req admin_request.TerminateSessionRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		// 这里我们可以忽略绑定错误，因为reason字段是可选的
		global.GVA_LOG.Warn("解析终止会话请求体出错，但继续处理", zap.Error(err))
	}

	// 记录操作的用户信息
	userID := utils.GetUserID(c) // 从 utils 包中获取当前登录用户ID
	global.GVA_LOG.Info("管理员操作：终止会话",
		zap.String("adminUserID", strconv.FormatUint(uint64(userID), 10)),
		zap.String("targetSessionID", sessionID),
		zap.String("reason", req.Reason),
	)

	// 调用服务终止会话
	err := adminSessionService.TerminateSession(sessionID, req.Reason, userID)
	if err != nil {
		global.GVA_LOG.Error("终止会话失败!", zap.Error(err))
		response.FailWithMessage("终止会话失败: "+err.Error(), c)
		return
	}

	// 记录审计日志（已在Service层实现）

	response.OkWithMessage("终止会话成功", c)
}
