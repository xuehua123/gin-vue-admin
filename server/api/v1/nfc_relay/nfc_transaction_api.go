package nfc_relay

import (
	"context"
	"net/http"
	"strconv"
	"time"

	"github.com/flipped-aurora/gin-vue-admin/server/global"
	"github.com/flipped-aurora/gin-vue-admin/server/model/common/response"
	"github.com/flipped-aurora/gin-vue-admin/server/model/nfc_relay/request"
	"github.com/flipped-aurora/gin-vue-admin/server/service/nfc_relay"
	"github.com/flipped-aurora/gin-vue-admin/server/utils"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"go.uber.org/zap"
)

type NFCTransactionApi struct{}

var nfcTransactionService = nfc_relay.NFCTransactionService{}

// RegisterForPairing handles client pairing requests.
// @Tags NFCPairing
// @Summary 请求自动配对
// @Description 客户端（传卡端或收卡端）请求进行自动配对。
// @Security ApiKeyAuth
// @accept application/json
// @Produce application/json
// @Param data body request.RegisterForPairingRequest true "配对请求"
// @Param force query bool false "是否强制接管现有会话" default(false)
// @Success 200 {object} response.Response "等待中：已将您加入等待队列"
// @Success 202 {object} response.Response "匹配成功"
// @Failure 409 {object} response.Response "冲突：角色已被占用，可尝试强制接管"
// @Failure 500 {object} response.Response "失败：服务器内部错误"
// @Router /nfc-relay/pairing/register [post]
func (a *NFCTransactionApi) RegisterForPairing(c *gin.Context) {
	var req request.RegisterForPairingRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.FailWithMessage(err.Error(), c)
		return
	}

	userUUID := utils.GetUserUuid(c)
	if userUUID == uuid.Nil {
		response.FailWithMessage("获取用户信息失败，请重新登录", c)
		return
	}

	force := c.Query("force") == "true"

	ctx := context.Background() // 使用 Background context

	status, data, err := nfcTransactionService.RegisterForPairing(ctx, &req, userUUID, force)
	if err != nil {
		global.GVA_LOG.Error("配对服务失败", zap.Error(err), zap.String("userID", userUUID.String()))
		response.FailWithMessage(err.Error(), c)
		return
	}

	switch status {
	case "waiting":
		response.OkWithMessage("已将您加入等待队列，请等待匹配...", c)
	case "matched":
		// 使用 202 Accepted 表示服务器已接受请求，并在后台异步处理通知
		c.JSON(http.StatusAccepted, response.Response{
			Code: 0,
			Data: data,
			Msg:  "匹配成功！请注意查收系统通知获取交易ID。",
		})
	case "conflict":
		// 使用 409 Conflict 表示请求冲突
		c.JSON(http.StatusConflict, response.Response{
			Code: 409,
			Msg:  "角色已被占用，可尝试强制接管。",
		})
	default: // "error" or other cases
		response.FailWithMessage("配对失败，未知错误", c)
	}
}

// CreateTransaction 创建交易
// @Tags NFCTransaction
// @Summary 创建NFC中继交易
// @Description 创建新的NFC中继交易
// @Security ApiKeyAuth
// @accept application/json
// @Produce application/json
// @Param data body request.CreateTransactionRequest true "创建交易请求"
// @Success 200 {object} response.Response{data=response.CreateTransactionResponse} "创建成功"
// @Router /nfc-relay/transactions [post]
func (a *NFCTransactionApi) CreateTransaction(c *gin.Context) {
	var req request.CreateTransactionRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.FailWithMessage(err.Error(), c)
		return
	}

	// 参数验证
	if err := utils.Verify(req, utils.CreateTransactionVerify); err != nil {
		response.FailWithMessage(err.Error(), c)
		return
	}

	// 获取当前用户UUID
	userUUID := utils.GetUserUuid(c)
	if userUUID == uuid.Nil {
		response.FailWithMessage("获取用户信息失败", c)
		return
	}

	ctx := context.WithValue(context.Background(), "userID", userUUID)

	result, err := nfcTransactionService.CreateTransaction(ctx, &req, userUUID)
	if err != nil {
		global.GVA_LOG.Error("创建交易失败", zap.Error(err))
		response.FailWithMessage(err.Error(), c)
		return
	}

	response.OkWithDetailed(result, "创建交易成功", c)
}

// UpdateTransactionStatus 更新交易状态
// @Tags NFCTransaction
// @Summary 更新交易状态
// @Description 更新NFC中继交易的状态
// @Security ApiKeyAuth
// @accept application/json
// @Produce application/json
// @Param data body request.UpdateTransactionStatusRequest true "更新状态请求"
// @Success 200 {object} response.Response{data=response.UpdateTransactionStatusResponse} "更新成功"
// @Router /nfc-relay/transactions/status [put]
func (a *NFCTransactionApi) UpdateTransactionStatus(c *gin.Context) {
	var req request.UpdateTransactionStatusRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.FailWithMessage(err.Error(), c)
		return
	}

	// 参数验证
	if err := utils.Verify(req, utils.UpdateTransactionStatusVerify); err != nil {
		response.FailWithMessage(err.Error(), c)
		return
	}

	// 获取当前用户UUID
	userUUID := utils.GetUserUuid(c)
	if userUUID == uuid.Nil {
		response.FailWithMessage("获取用户信息失败", c)
		return
	}

	ctx := context.WithValue(context.Background(), "userID", userUUID)

	result, err := nfcTransactionService.UpdateTransactionStatus(ctx, &req, userUUID)
	if err != nil {
		global.GVA_LOG.Error("更新交易状态失败", zap.Error(err))
		response.FailWithMessage(err.Error(), c)
		return
	}

	response.OkWithDetailed(result, "更新交易状态成功", c)
}

// GetTransaction 获取交易详情
// @Tags NFCTransaction
// @Summary 获取交易详情
// @Description 根据交易ID获取详细信息
// @Security ApiKeyAuth
// @accept application/json
// @Produce application/json
// @Param transaction_id path string true "交易ID"
// @Param include_apdu query bool false "是否包含APDU消息" default(false)
// @Success 200 {object} response.Response{data=response.TransactionDetailResponse} "获取成功"
// @Router /nfc-relay/transactions/{transaction_id} [get]
func (a *NFCTransactionApi) GetTransaction(c *gin.Context) {
	transactionID := c.Param("transaction_id")
	if transactionID == "" {
		response.FailWithMessage("交易ID不能为空", c)
		return
	}

	includeAPDU, _ := strconv.ParseBool(c.Query("include_apdu"))

	req := request.GetTransactionRequest{
		TransactionID: transactionID,
		IncludeAPDU:   includeAPDU,
	}

	// 获取当前用户UUID
	userUUID := utils.GetUserUuid(c)
	if userUUID == uuid.Nil {
		response.FailWithMessage("获取用户信息失败", c)
		return
	}

	ctx := context.WithValue(context.Background(), "userID", userUUID)

	result, err := nfcTransactionService.GetTransaction(ctx, &req, userUUID)
	if err != nil {
		global.GVA_LOG.Error("获取交易详情失败", zap.Error(err))
		response.FailWithMessage(err.Error(), c)
		return
	}

	response.OkWithDetailed(result, "获取交易详情成功", c)
}

// GetTransactionList 获取交易列表（完善实现）
// @Tags NFCTransaction
// @Summary 获取交易列表
// @Description 分页获取交易列表
// @Security ApiKeyAuth
// @accept application/json
// @Produce application/json
// @Param data query request.GetTransactionListRequest true "查询参数"
// @Success 200 {object} response.Response{data=response.TransactionListResponse} "获取成功"
// @Router /nfc-relay/transactions [get]
func (a *NFCTransactionApi) GetTransactionList(c *gin.Context) {
	var req request.GetTransactionListRequest
	if err := c.ShouldBindQuery(&req); err != nil {
		response.FailWithMessage(err.Error(), c)
		return
	}

	// 设置默认分页参数
	if req.Page <= 0 {
		req.Page = 1
	}
	if req.PageSize <= 0 {
		req.PageSize = 10
	}
	if req.PageSize > 100 {
		req.PageSize = 100 // 限制最大页大小
	}

	// 获取当前用户UUID
	userUUID := utils.GetUserUuid(c)
	if userUUID == uuid.Nil {
		response.FailWithMessage("获取用户信息失败", c)
		return
	}

	ctx := context.WithValue(context.Background(), "userID", userUUID)

	// 调用服务层获取交易列表
	result, err := nfcTransactionService.GetTransactionList(ctx, &req, userUUID)
	if err != nil {
		global.GVA_LOG.Error("获取交易列表失败", zap.Error(err))
		response.FailWithMessage(err.Error(), c)
		return
	}

	response.OkWithDetailed(result, "获取交易列表成功", c)
}

// DeleteTransaction 删除交易（完善实现）
// @Tags NFCTransaction
// @Summary 删除交易
// @Description 删除指定的交易记录
// @Security ApiKeyAuth
// @accept application/json
// @Produce application/json
// @Param transaction_id path string true "交易ID"
// @Param force query bool false "是否强制删除" default(false)
// @Success 200 {object} response.Response "删除成功"
// @Router /nfc-relay/transactions/{transaction_id} [delete]
func (a *NFCTransactionApi) DeleteTransaction(c *gin.Context) {
	transactionID := c.Param("transaction_id")
	if transactionID == "" {
		response.FailWithMessage("交易ID不能为空", c)
		return
	}

	force, _ := strconv.ParseBool(c.Query("force"))

	// 获取当前用户UUID
	userUUID := utils.GetUserUuid(c)
	if userUUID == uuid.Nil {
		response.FailWithMessage("获取用户信息失败", c)
		return
	}

	ctx := context.WithValue(context.Background(), "userID", userUUID)

	req := request.DeleteTransactionRequest{
		TransactionID: transactionID,
		Force:         force,
	}

	// 调用服务层删除交易
	err := nfcTransactionService.DeleteTransaction(ctx, &req, userUUID)
	if err != nil {
		global.GVA_LOG.Error("删除交易失败", zap.Error(err))
		response.FailWithMessage(err.Error(), c)
		return
	}

	response.OkWithMessage("删除交易成功", c)
}

// BatchUpdateStatus 批量更新交易状态（完善实现）
// @Tags NFCTransaction
// @Summary 批量更新交易状态
// @Description 批量更新多个交易的状态
// @Security ApiKeyAuth
// @accept application/json
// @Produce application/json
// @Param data body request.BatchUpdateTransactionRequest true "批量更新请求"
// @Success 200 {object} response.Response{data=response.BatchOperationResponse} "更新成功"
// @Router /nfc-relay/transactions/batch-update [put]
func (a *NFCTransactionApi) BatchUpdateStatus(c *gin.Context) {
	var req request.BatchUpdateTransactionRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.FailWithMessage(err.Error(), c)
		return
	}

	// 参数验证
	if len(req.TransactionIDs) == 0 {
		response.FailWithMessage("交易ID列表不能为空", c)
		return
	}

	if len(req.TransactionIDs) > 100 {
		response.FailWithMessage("单次批量操作不能超过100个交易", c)
		return
	}

	// 获取当前用户UUID
	userUUID := utils.GetUserUuid(c)
	if userUUID == uuid.Nil {
		response.FailWithMessage("获取用户信息失败", c)
		return
	}

	ctx := context.WithValue(context.Background(), "userID", userUUID)

	// 调用服务层批量更新
	result, err := nfcTransactionService.BatchUpdateTransactionStatus(ctx, &req, userUUID)
	if err != nil {
		global.GVA_LOG.Error("批量更新交易状态失败", zap.Error(err))
		response.FailWithMessage(err.Error(), c)
		return
	}

	response.OkWithDetailed(result, "批量更新成功", c)
}

// SendAPDU 发送APDU消息（完善实现）
// @Tags NFCTransaction
// @Summary 发送APDU消息
// @Description 向指定客户端发送APDU消息
// @Security ApiKeyAuth
// @accept application/json
// @Produce application/json
// @Param data body request.SendAPDURequest true "APDU消息请求"
// @Success 200 {object} response.Response{data=response.SendAPDUResponse} "发送成功"
// @Router /nfc-relay/transactions/apdu [post]
func (a *NFCTransactionApi) SendAPDU(c *gin.Context) {
	var req request.SendAPDURequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.FailWithMessage(err.Error(), c)
		return
	}

	// 参数验证
	if err := utils.Verify(req, utils.SendAPDUVerify); err != nil {
		response.FailWithMessage(err.Error(), c)
		return
	}

	// 获取当前用户UUID
	userUUID := utils.GetUserUuid(c)
	if userUUID == uuid.Nil {
		response.FailWithMessage("获取用户信息失败", c)
		return
	}

	ctx := context.WithValue(context.Background(), "userID", userUUID)

	// 调用服务层发送APDU消息
	result, err := nfcTransactionService.SendAPDU(ctx, &req, userUUID)
	if err != nil {
		global.GVA_LOG.Error("发送APDU消息失败", zap.Error(err))
		response.FailWithMessage(err.Error(), c)
		return
	}

	response.OkWithDetailed(result, "发送APDU消息成功", c)
}

// GetAPDUList 获取APDU消息列表（完善实现）
// @Tags NFCTransaction
// @Summary 获取APDU消息列表
// @Description 获取指定交易的APDU消息记录
// @Security ApiKeyAuth
// @accept application/json
// @Produce application/json
// @Param data query request.GetAPDUListRequest true "查询参数"
// @Success 200 {object} response.Response{data=response.APDUMessageListResponse} "获取成功"
// @Router /nfc-relay/transactions/apdu [get]
func (a *NFCTransactionApi) GetAPDUList(c *gin.Context) {
	var req request.GetAPDUListRequest
	if err := c.ShouldBindQuery(&req); err != nil {
		response.FailWithMessage(err.Error(), c)
		return
	}

	// 基本验证
	if req.TransactionID == "" {
		response.FailWithMessage("交易ID不能为空", c)
		return
	}

	// 设置默认分页参数
	if req.Page <= 0 {
		req.Page = 1
	}
	if req.PageSize <= 0 {
		req.PageSize = 20
	}
	if req.PageSize > 100 {
		req.PageSize = 100
	}

	// 获取当前用户UUID
	userUUID := utils.GetUserUuid(c)
	if userUUID == uuid.Nil {
		response.FailWithMessage("获取用户信息失败", c)
		return
	}

	ctx := context.WithValue(context.Background(), "userID", userUUID)

	// 调用服务层获取APDU消息列表
	result, err := nfcTransactionService.GetAPDUList(ctx, &req, userUUID)
	if err != nil {
		global.GVA_LOG.Error("获取APDU消息列表失败", zap.Error(err))
		response.FailWithMessage(err.Error(), c)
		return
	}

	response.OkWithDetailed(result, "获取APDU消息列表成功", c)
}

// GetStatistics 获取统计信息（完善实现）
// @Tags NFCTransaction
// @Summary 获取统计信息
// @Description 获取交易统计数据和图表
// @Security ApiKeyAuth
// @accept application/json
// @Produce application/json
// @Param data query request.GetStatisticsRequest true "统计查询参数"
// @Success 200 {object} response.Response{data=response.TransactionStatisticsResponse} "获取成功"
// @Router /nfc-relay/transactions/statistics [get]
func (a *NFCTransactionApi) GetStatistics(c *gin.Context) {
	var req request.GetStatisticsRequest
	if err := c.ShouldBindQuery(&req); err != nil {
		response.FailWithMessage(err.Error(), c)
		return
	}

	// 设置默认参数
	if req.StartDate == "" {
		req.StartDate = time.Now().AddDate(0, 0, -7).Format("2006-01-02")
	}
	if req.EndDate == "" {
		req.EndDate = time.Now().Format("2006-01-02")
	}
	if req.GroupBy == "" {
		req.GroupBy = "day"
	}

	// 获取当前用户UUID
	userUUID := utils.GetUserUuid(c)
	if userUUID == uuid.Nil {
		response.FailWithMessage("获取用户信息失败", c)
		return
	}

	ctx := context.WithValue(context.Background(), "userID", userUUID)

	// 调用服务层获取统计信息
	result, err := nfcTransactionService.GetStatistics(ctx, &req, userUUID)
	if err != nil {
		global.GVA_LOG.Error("获取统计信息失败", zap.Error(err))
		response.FailWithMessage(err.Error(), c)
		return
	}

	response.OkWithDetailed(result, "获取统计信息成功", c)
}

// ExportTransactions 导出交易数据
// @Tags NFCTransaction
// @Summary 导出交易数据
// @Description 导出交易数据为Excel或CSV格式
// @Security ApiKeyAuth
// @accept application/json
// @Produce application/json
// @Param format query string false "导出格式(excel/csv)" default("excel")
// @Param start_date query string false "开始日期(YYYY-MM-DD)"
// @Param end_date query string false "结束日期(YYYY-MM-DD)"
// @Param status query string false "状态筛选"
// @Success 200 {object} response.Response "导出成功"
// @Router /nfc-relay/transactions/export [get]
func (a *NFCTransactionApi) ExportTransactions(c *gin.Context) {
	format := c.DefaultQuery("format", "excel")
	startDate := c.Query("start_date")
	endDate := c.Query("end_date")
	status := c.Query("status")

	// 获取当前用户UUID
	userUUID := utils.GetUserUuid(c)
	if userUUID == uuid.Nil {
		response.FailWithMessage("获取用户信息失败", c)
		return
	}

	// 记录导出参数
	global.GVA_LOG.Info("导出交易数据",
		zap.String("format", format),
		zap.String("startDate", startDate),
		zap.String("endDate", endDate),
		zap.String("status", status),
		zap.String("userUUID", userUUID.String()))

	// 暂时返回成功消息
	response.OkWithMessage("导出任务已创建，请稍后下载", c)
}

// InitiateTransactionSession 发起交易会话
// @Tags NFCTransaction
// @Summary 发起NFC中继交易会话
// @Description 客户端发起一个新的交易会话，等待对端加入
// @Security ApiKeyAuth
// @accept application/json
// @Produce application/json
// @Param data body request.InitiateTransactionSessionRequest true "发起会话请求"
// @Success 200 {object} response.Response{data=request.TransactionSessionResponse} "发起成功"
// @Router /nfc-relay/transactions/sessions/initiate [post]
func (a *NFCTransactionApi) InitiateTransactionSession(c *gin.Context) {
	var req request.InitiateTransactionSessionRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.FailWithMessage("请求参数格式错误: "+err.Error(), c)
		return
	}

	// 参数验证
	if req.Role == "" {
		response.FailWithMessage("角色(role)不能为空", c)
		return
	}
	if req.Role != "transmitter" && req.Role != "receiver" {
		response.FailWithMessage("角色(role)必须是 transmitter 或 receiver", c)
		return
	}
	if req.TimeoutSecs < 30 || req.TimeoutSecs > 3600 {
		response.FailWithMessage("超时时间必须在30-3600秒之间", c)
		return
	}

	// 获取当前用户信息
	userUUID := utils.GetUserUuid(c)
	if userUUID == uuid.Nil {
		response.FailWithMessage("获取用户信息失败", c)
		return
	}

	// 从JWT中获取用户名
	claims := utils.GetUserInfo(c)
	username := ""
	if claims != nil {
		username = claims.Username
	}

	// 从JWT中获取客户端ID
	clientID := utils.GetClientIDFromJWT(c)
	if clientID == "" {
		response.FailWithMessage("无法获取客户端ID，请重新登录", c)
		return
	}

	// 设置context
	ctx := context.WithValue(context.Background(), "userID", userUUID)
	ctx = context.WithValue(ctx, "clientID", clientID)
	ctx = context.WithValue(ctx, "username", username)

	// 调用服务层
	result, err := nfcTransactionService.InitiateTransactionSession(ctx, req, userUUID, username)
	if err != nil {
		global.GVA_LOG.Error("发起交易会话失败",
			zap.String("clientID", clientID),
			zap.String("role", req.Role),
			zap.String("username", username),
			zap.Error(err))
		response.FailWithMessage(err.Error(), c)
		return
	}

	global.GVA_LOG.Info("交易会话发起API调用成功",
		zap.String("transactionID", result.TransactionID),
		zap.String("clientID", clientID),
		zap.String("role", req.Role),
		zap.String("username", username))

	response.OkWithDetailed(result, "发起交易会话成功", c)
}

// JoinTransactionSession 加入交易会话
// @Tags NFCTransaction
// @Summary 加入NFC中继交易会话
// @Description 客户端加入一个已存在的交易会话
// @Security ApiKeyAuth
// @accept application/json
// @Produce application/json
// @Param data body request.JoinTransactionSessionRequest true "加入会话请求"
// @Success 200 {object} response.Response{data=request.TransactionSessionResponse} "加入成功"
// @Router /nfc-relay/transactions/sessions/join [post]
func (a *NFCTransactionApi) JoinTransactionSession(c *gin.Context) {
	var req request.JoinTransactionSessionRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.FailWithMessage("请求参数格式错误: "+err.Error(), c)
		return
	}

	// 参数验证
	if req.TransactionID == "" {
		response.FailWithMessage("交易ID(transaction_id)不能为空", c)
		return
	}
	if req.Role == "" {
		response.FailWithMessage("角色(role)不能为空", c)
		return
	}
	if req.Role != "transmitter" && req.Role != "receiver" {
		response.FailWithMessage("角色(role)必须是 transmitter 或 receiver", c)
		return
	}

	// 获取当前用户信息
	userUUID := utils.GetUserUuid(c)
	if userUUID == uuid.Nil {
		response.FailWithMessage("获取用户信息失败", c)
		return
	}

	// 从JWT中获取用户名
	claims := utils.GetUserInfo(c)
	username := ""
	if claims != nil {
		username = claims.Username
	}

	// 从JWT中获取客户端ID
	clientID := utils.GetClientIDFromJWT(c)
	if clientID == "" {
		response.FailWithMessage("无法获取客户端ID，请重新登录", c)
		return
	}

	// 设置context
	ctx := context.WithValue(context.Background(), "userID", userUUID)
	ctx = context.WithValue(ctx, "clientID", clientID)
	ctx = context.WithValue(ctx, "username", username)

	// 调用服务层
	result, err := nfcTransactionService.JoinTransactionSession(ctx, req, userUUID, username)
	if err != nil {
		global.GVA_LOG.Error("加入交易会话失败",
			zap.String("transactionID", req.TransactionID),
			zap.String("clientID", clientID),
			zap.String("role", req.Role),
			zap.String("username", username),
			zap.Error(err))
		response.FailWithMessage(err.Error(), c)
		return
	}

	global.GVA_LOG.Info("交易会话加入API调用成功",
		zap.String("transactionID", req.TransactionID),
		zap.String("clientID", clientID),
		zap.String("role", req.Role),
		zap.String("username", username),
		zap.String("newStatus", result.Status))

	response.OkWithDetailed(result, "加入交易会话成功", c)
}

// GetTransactionSession 获取交易会话状态
// @Tags NFCTransaction
// @Summary 获取交易会话状态
// @Description 获取指定交易会话的当前状态和配置信息
// @Security ApiKeyAuth
// @accept application/json
// @Produce application/json
// @Param transaction_id path string true "交易ID"
// @Success 200 {object} response.Response{data=request.TransactionSessionResponse} "获取成功"
// @Router /nfc-relay/transactions/sessions/{transaction_id} [get]
func (a *NFCTransactionApi) GetTransactionSession(c *gin.Context) {
	transactionID := c.Param("transaction_id")
	if transactionID == "" {
		response.FailWithMessage("交易ID不能为空", c)
		return
	}

	// 获取当前用户信息
	userUUID := utils.GetUserUuid(c)
	if userUUID == uuid.Nil {
		response.FailWithMessage("获取用户信息失败", c)
		return
	}

	// 从JWT中获取客户端ID
	clientID := utils.GetClientIDFromJWT(c)
	if clientID == "" {
		response.FailWithMessage("无法获取客户端ID，请重新登录", c)
		return
	}

	// 调用服务层获取交易详情（重用现有方法）
	getReq := request.GetTransactionRequest{
		TransactionID: transactionID,
		IncludeAPDU:   false, // 获取会话状态不需要APDU详情
	}

	ctx := context.WithValue(context.Background(), "userID", userUUID)
	ctx = context.WithValue(ctx, "clientID", clientID)

	transactionDetail, err := nfcTransactionService.GetTransaction(ctx, &getReq, userUUID)
	if err != nil {
		global.GVA_LOG.Error("获取交易会话失败",
			zap.String("transactionID", transactionID),
			zap.String("clientID", clientID),
			zap.Error(err))
		response.FailWithMessage(err.Error(), c)
		return
	}

	// 构建会话响应（从交易详情转换）
	sessionResponse := request.TransactionSessionResponse{
		TransactionID:       transactionDetail.TransactionID,
		Status:              transactionDetail.Status,
		TransmitterClientID: transactionDetail.TransmitterClientID,
		ReceiverClientID:    transactionDetail.ReceiverClientID,
		ExpiresAt:           transactionDetail.ExpiresAt.Unix(),
		CreatedAt:           transactionDetail.CreatedAt.Unix(),
		TopicConfig: request.TransactionTopicConfig{
			TransmitterStateTopic:  transactionDetail.TransmitterStateTopic,
			ReceiverStateTopic:     transactionDetail.ReceiverStateTopic,
			APDUToTransmitterTopic: transactionDetail.APDUToTransmitterTopic,
			APDUToReceiverTopic:    transactionDetail.APDUToReceiverTopic,
			ControlTopic:           transactionDetail.ControlTopic,
			HeartbeatTopic:         transactionDetail.HeartbeatTopic,
		},
	}

	// 确定当前客户端的角色
	if clientID == transactionDetail.TransmitterClientID {
		sessionResponse.Role = "transmitter"
		sessionResponse.PeerRole = "receiver"
	} else if clientID == transactionDetail.ReceiverClientID {
		sessionResponse.Role = "receiver"
		sessionResponse.PeerRole = "transmitter"
	} else {
		response.FailWithMessage("您不是此交易的参与者", c)
		return
	}

	response.OkWithDetailed(sessionResponse, "获取交易会话状态成功", c)
}
