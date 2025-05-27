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

// AdminClientApi 结构体，用于客户端管理相关的API处理
type AdminClientApi struct{}

var adminClientService = service.AdminClientService{}

// GetClientList
// @Tags NFCRelayAdmin
// @Summary 获取NFC Relay客户端列表
// @Security ApiKeyAuth
// @Produce application/json
// @Param page query int false "页码"
// @Param pageSize query int false "每页条数"
// @Param clientID query string false "客户端ID（模糊匹配）"
// @Param userID query string false "用户ID（模糊匹配）"
// @Param role query string false "角色（provider/receiver/none）"
// @Param ipAddress query string false "IP地址（模糊匹配）"
// @Param sessionID query string false "会话ID"
// @Success 200 {object} response.Response{data=admin_response.PaginatedClientListResponse,msg=string} "获取成功"
// @Router /admin/nfc-relay/v1/clients [get]
func (a *AdminClientApi) GetClientList(c *gin.Context) {
	var params admin_request.ClientListParams
	// 绑定查询参数
	if err := c.ShouldBindQuery(&params); err != nil {
		global.GVA_LOG.Error("获取客户端列表请求参数错误!", zap.Error(err))
		response.FailWithMessage("请求参数错误: "+err.Error(), c)
		return
	}

	// 调用服务获取客户端列表
	clientList, err := adminClientService.GetClientList(params)
	if err != nil {
		global.GVA_LOG.Error("获取客户端列表失败!", zap.Error(err))
		response.FailWithMessage("获取客户端列表失败: "+err.Error(), c)
		return
	}

	response.OkWithDetailed(clientList, "获取客户端列表成功", c)
}

// GetClientDetail
// @Tags NFCRelayAdmin
// @Summary 获取NFC Relay单个客户端详情
// @Security ApiKeyAuth
// @Produce application/json
// @Param clientID path string true "客户端ID"
// @Success 200 {object} response.Response{data=admin_response.ClientDetailResponse,msg=string} "获取成功"
// @Router /admin/nfc-relay/v1/clients/{clientID} [get]
func (a *AdminClientApi) GetClientDetail(c *gin.Context) {
	clientID := c.Param("clientID")
	if clientID == "" {
		response.FailWithMessage("客户端ID不能为空", c)
		return
	}

	// 调用服务获取客户端详情
	clientDetail, err := adminClientService.GetClientDetail(clientID)
	if err != nil {
		global.GVA_LOG.Error("获取客户端详情失败!", zap.Error(err))
		response.FailWithMessage("获取客户端详情失败: "+err.Error(), c)
		return
	}

	response.OkWithDetailed(clientDetail, "获取客户端详情成功", c)
}

// DisconnectClient
// @Tags NFCRelayAdmin
// @Summary 强制断开NFC Relay客户端连接
// @Security ApiKeyAuth
// @Produce application/json
// @Param clientID path string true "客户端ID"
// @Success 200 {object} response.Response{msg=string} "操作成功"
// @Router /admin/nfc-relay/v1/clients/{clientID}/disconnect [post]
func (a *AdminClientApi) DisconnectClient(c *gin.Context) {
	clientID := c.Param("clientID")
	if clientID == "" {
		response.FailWithMessage("客户端ID不能为空", c)
		return
	}

	// 记录操作的用户信息（如果需要）
	userID := utils.GetUserID(c) // 从 utils 包中获取当前登录用户ID
	global.GVA_LOG.Info("管理员操作：强制断开客户端连接",
		zap.String("adminUserID", strconv.FormatUint(uint64(userID), 10)), // 转换为 string
		zap.String("targetClientID", clientID),
	)

	// 调用服务断开客户端连接
	err := adminClientService.DisconnectClient(clientID)
	if err != nil {
		global.GVA_LOG.Error("断开客户端连接失败!", zap.Error(err))
		response.FailWithMessage("断开客户端连接失败: "+err.Error(), c)
		return
	}

	// 记录审计日志
	global.LogAuditEvent(
		"admin_action",
		map[string]interface{}{
			"action":        "disconnect_client",
			"target_client": clientID,
			"admin_user_id": userID, // 直接使用 uint 类型
		},
		zap.String("admin_action", "disconnect_client"),
		zap.String("target_client", clientID),
		zap.Uint("admin_user_id", userID), // 使用 zap.Uint
	)

	response.OkWithMessage("断开客户端连接成功", c)
}

// 获取当前登录用户ID
func getUserID(c *gin.Context) uint {
	return utils.GetUserID(c)
}
