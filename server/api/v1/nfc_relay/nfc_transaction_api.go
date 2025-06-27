package nfc_relay

import (
	"context"
	"fmt"
	"net/http"
	"strings"

	"github.com/flipped-aurora/gin-vue-admin/server/global"
	"github.com/flipped-aurora/gin-vue-admin/server/model/common/response"
	"github.com/flipped-aurora/gin-vue-admin/server/model/nfc_relay/request"
	nfcreq "github.com/flipped-aurora/gin-vue-admin/server/model/nfc_relay/request"
	nfcres "github.com/flipped-aurora/gin-vue-admin/server/model/nfc_relay/response"
	"github.com/flipped-aurora/gin-vue-admin/server/service"
	"github.com/flipped-aurora/gin-vue-admin/server/utils"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"go.uber.org/zap"
)

type NFCTransactionApi struct{}

// RegisterForPairing handles client pairing requests.
// @Tags NFCPairing
// @Summary 请求自动配对
// @Description 客户端（传卡端或收卡端）请求进行自动配对。
// @Security ApiKeyAuth
// @accept application/json
// @Produce application/json
// @Param data body nfcreq.RegisterForPairingRequest true "配对请求"
// @Param force query bool false "是否强制接管现有会话" default(false)
// @Success 200 {object} response.Response{data=nfcres.PairingResponse} "成功"
// @Success 202 {object} response.Response{data=nfcres.PairingResponse} "匹配成功"
// @Failure 409 {object} response.Response "冲突：角色已被占用，可尝试强制接管"
// @Failure 500 {object} response.Response "失败：服务器内部错误"
// @Router /nfc-relay/pairing/register [post]
func (a *NFCTransactionApi) RegisterForPairing(c *gin.Context) {
	var req nfcreq.RegisterForPairingRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.FailWithMessage(err.Error(), c)
		return
	}

	claims := utils.GetUserInfo(c)
	if claims == nil {
		response.FailWithMessage("获取用户信息失败，请重新登录", c)
		return
	}

	force := c.Query("force") == "true"

	global.GVA_LOG.Info("收到配对请求",
		zap.String("userID", claims.GetUserID()),
		zap.String("role", req.Role),
		zap.Bool("force", force),
	)

	// 1. 生成权威的MQTT凭证
	jwt := utils.NewJWT()
	mqttClaims, err := jwt.CreateMQTTClaims(claims.GetUserID(), claims.Username, req.Role)
	if err != nil {
		global.GVA_LOG.Error("创建MQTT Claims失败", zap.Error(err), zap.String("userID", claims.GetUserID()))
		response.FailWithMessage("创建MQTT凭证失败", c)
		return
	}

	mqttToken, err := jwt.CreateMQTTToken(mqttClaims)
	if err != nil {
		global.GVA_LOG.Error("生成MQTT Token失败", zap.Error(err), zap.String("userID", claims.GetUserID()))
		response.FailWithMessage("生成MQTT凭证失败", c)
		return
	}

	// 2. 使用生成的权威ClientID进行配对注册
	ctx := context.Background()
	nfcTransactionService := service.ServiceGroupApp.NFCRelayServiceGroup.TransactionService()
	pairingResult, err := nfcTransactionService.RegisterForPairing(ctx, &req, claims.UUID, force, mqttClaims.ClientID)
	if err != nil {
		// 根据企业级API契约，将特定的业务错误映射到正确的HTTP状态码
		if strings.Contains(err.Error(), "客户端离线") {
			global.GVA_LOG.Warn("配对失败，客户端离线", zap.Error(err), zap.String("userID", claims.GetUserID()))
			c.JSON(http.StatusConflict, response.Response{
				Code: response.ERROR, // 业务错误码 (通常为 7)
				Data: nil,
				Msg:  err.Error(),
			})
			return
		}

		// 对于其他未知错误，返回 HTTP 500 Internal Server Error
		global.GVA_LOG.Error("配对服务发生未知内部错误", zap.Error(err), zap.String("userID", claims.GetUserID()))
		c.JSON(http.StatusInternalServerError, response.Response{
			Code: response.ERROR,
			Data: nil,
			Msg:  "服务器内部错误，请稍后重试",
		})
		return
	}

	// 3. 构造统一响应
	resp := nfcres.PairingResponse{
		Status:        pairingResult.Status,
		QueuePosition: pairingResult.QueuePosition,
		EstimatedWait: pairingResult.EstimatedWait,
		TransactionID: pairingResult.TransactionID,
		ClientID:      mqttClaims.ClientID,
		MqttToken:     mqttToken,
		ExpiresAt:     mqttClaims.ExpiresAt.Unix(),
		Role:          mqttClaims.Role,
	}

	switch pairingResult.Status {
	case "waiting":
		global.GVA_LOG.Info("加入等待队列", zap.String("userID", claims.GetUserID()), zap.String("role", req.Role))
		response.OkWithDetailed(resp, "已将您加入等待队列，请等待匹配...", c)
	case "matched":
		global.GVA_LOG.Info("匹配成功", zap.String("userID", claims.GetUserID()), zap.Any("data", resp))
		c.JSON(http.StatusAccepted, response.Response{
			Code: 0,
			Data: resp,
			Msg:  "匹配成功！请注意查收系统通知获取交易ID。",
		})
	case "session_conflict":
		global.GVA_LOG.Info("响应会话冲突", zap.String("userID", claims.GetUserID()), zap.Any("data", pairingResult.ConflictDetails))
		c.JSON(http.StatusConflict, response.Response{
			Code: 40901,
			Data: pairingResult.ConflictDetails,
			Msg:  "客户端与服务器会话不一致，请同步。",
		})
	case "conflict":
		global.GVA_LOG.Warn("角色冲突", zap.String("userID", claims.GetUserID()), zap.String("role", req.Role))
		conflictMessage := "角色已被占用，可尝试强制接管。"
		if pairingResult.ConflictDetails != nil {
			conflictMessage = fmt.Sprintf("角色已被其他设备占用，您可以选择强制接管或等待重试。")
		}
		c.JSON(http.StatusConflict, response.Response{
			Code: 409,
			Data: pairingResult.ConflictDetails,
			Msg:  conflictMessage,
		})
	default: // "error" or other cases
		global.GVA_LOG.Error("配对失败，未知状态", zap.String("status", pairingResult.Status), zap.String("userID", claims.GetUserID()))
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
// @Param data body nfcreq.CreateTransactionRequest true "创建交易请求"
// @Success 200 {object} response.Response{data=response.CreateTransactionResponse} "创建成功"
// @Router /nfc-relay/transactions [post]
func (a *NFCTransactionApi) CreateTransaction(c *gin.Context) {
	var req nfcreq.CreateTransactionRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.FailWithMessage(err.Error(), c)
		return
	}
	userUUID := utils.GetUserUuid(c)
	nfcTransactionService := service.ServiceGroupApp.NFCRelayServiceGroup.TransactionService()
	result, err := nfcTransactionService.CreateTransaction(context.Background(), &req, userUUID)
	if err != nil {
		global.GVA_LOG.Error("创建交易失败", zap.Error(err))
		response.FailWithMessage(err.Error(), c)
		return
	}
	response.OkWithDetailed(result, "创建交易成功", c)
}

// GetTransaction 获取交易详情
// @Tags NFCTransaction
// @Summary 获取交易详情
// @Description 根据交易ID获取详细信息
// @Security ApiKeyAuth
// @accept application/json
// @Produce application/json
// @Param transaction_id path string true "交易ID"
// @Success 200 {object} response.Response{data=response.TransactionDetailResponse} "获取成功"
// @Router /nfc-relay/transactions/{transaction_id} [get]
func (a *NFCTransactionApi) GetTransaction(c *gin.Context) {
	transactionID := c.Param("transaction_id")
	userUUID := utils.GetUserUuid(c)
	req := &nfcreq.GetTransactionRequest{
		TransactionID: transactionID,
	}
	nfcTransactionService := service.ServiceGroupApp.NFCRelayServiceGroup.TransactionService()
	result, err := nfcTransactionService.GetTransaction(context.Background(), req, userUUID)
	if err != nil {
		global.GVA_LOG.Error("获取交易详情失败", zap.Error(err))
		response.FailWithMessage(err.Error(), c)
		return
	}
	response.OkWithDetailed(result, "获取交易详情成功", c)
}

// UpdateTransactionStatus 更新交易状态
// @Tags NFCTransaction
// @Summary 更新交易状态
// @Description 更新NFC中继交易的状态
// @Security ApiKeyAuth
// @accept application/json
// @Produce application/json
// @Param data body nfcreq.UpdateTransactionStatusRequest true "更新状态请求"
// @Success 200 {object} response.Response{data=response.UpdateTransactionStatusResponse} "更新成功"
// @Router /nfc-relay/transactions/status [put]
func (a *NFCTransactionApi) UpdateTransactionStatus(c *gin.Context) {
	var req nfcreq.UpdateTransactionStatusRequest
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
	nfcTransactionService := service.ServiceGroupApp.NFCRelayServiceGroup.TransactionService()
	result, err := nfcTransactionService.UpdateTransactionStatus(ctx, &req, userUUID)
	if err != nil {
		global.GVA_LOG.Error("更新交易状态失败", zap.Error(err))
		response.FailWithMessage(err.Error(), c)
		return
	}

	response.OkWithDetailed(result, "更新交易状态成功", c)
}

// GetTransactionList 获取交易列表（完善实现）
// @Tags NFCTransaction
// @Summary 获取交易列表
// @Description 分页获取交易列表
// @Security ApiKeyAuth
// @accept application/json
// @Produce application/json
// @Param page query int false "页码"
// @Param pageSize query int false "每页数量"
// @Param orderKey query string false "排序字段"
// @Param orderDesc query bool false "是否降序"
// @Success 200 {object} response.Response{data=response.TransactionListResponse} "获取成功"
// @Router /nfc-relay/transactions [get]
func (a *NFCTransactionApi) GetTransactionList(c *gin.Context) {
	var pageInfo request.GetTransactionListRequest
	if err := c.ShouldBindQuery(&pageInfo); err != nil {
		response.FailWithMessage(err.Error(), c)
		return
	}
	userUUID := utils.GetUserUuid(c)
	ctx := context.WithValue(context.Background(), "userID", userUUID)
	nfcTransactionService := service.ServiceGroupApp.NFCRelayServiceGroup.TransactionService()
	result, err := nfcTransactionService.GetTransactionList(ctx, &pageInfo, userUUID)
	if err != nil {
		global.GVA_LOG.Error("获取交易列表失败", zap.Error(err))
		response.FailWithMessage(err.Error(), c)
		return
	}
	response.OkWithDetailed(result, "获取交易列表成功", c)
}

// DeleteTransaction 删除交易
// @Tags NFCTransaction
// @Summary 删除交易
// @Description 根据ID删除NFC中继交易
// @Security ApiKeyAuth
// @accept application/json
// @Produce application/json
// @Param data body request.DeleteTransactionRequest true "删除交易请求"
// @Success 200 {object} response.Response "删除成功"
// @Router /nfc-relay/transactions [delete]
func (a *NFCTransactionApi) DeleteTransaction(c *gin.Context) {
	var req request.DeleteTransactionRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.FailWithMessage(err.Error(), c)
		return
	}
	userUUID := utils.GetUserUuid(c)
	ctx := context.WithValue(context.Background(), "userID", userUUID)
	nfcTransactionService := service.ServiceGroupApp.NFCRelayServiceGroup.TransactionService()
	err := nfcTransactionService.DeleteTransaction(ctx, &req, userUUID)
	if err != nil {
		global.GVA_LOG.Error("删除交易失败", zap.Error(err))
		response.FailWithMessage(err.Error(), c)
		return
	}
	response.OkWithMessage("删除交易成功", c)
}

// BatchUpdateStatus 批量更新交易状态
// @Tags NFCTransaction
// @Summary 批量更新交易状态
// @Description 批量更新NFC中继交易的状态
// @Security ApiKeyAuth
// @accept application/json
// @Produce application/json
// @Param data body request.BatchUpdateTransactionRequest true "批量更新状态请求"
// @Success 200 {object} response.Response{data=response.BatchOperationResponse} "批量更新成功"
// @Router /nfc-relay/transactions/status/batch [put]
func (a *NFCTransactionApi) BatchUpdateStatus(c *gin.Context) {
	var req request.BatchUpdateTransactionRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.FailWithMessage(err.Error(), c)
		return
	}
	userUUID := utils.GetUserUuid(c)
	ctx := context.WithValue(context.Background(), "userID", userUUID)
	nfcTransactionService := service.ServiceGroupApp.NFCRelayServiceGroup.TransactionService()
	result, err := nfcTransactionService.BatchUpdateTransactionStatus(ctx, &req, userUUID)
	if err != nil {
		global.GVA_LOG.Error("批量更新交易状态失败", zap.Error(err))
		response.FailWithMessage(err.Error(), c)
		return
	}
	response.OkWithDetailed(result, "批量更新交易状态成功", c)
}

// SendAPDU 发送APDU指令
// @Tags NFCTransaction
// @Summary 发送APDU指令
// @Description 向指定客户端发送APDU指令
// @Security ApiKeyAuth
// @accept application/json
// @Produce application/json
// @Param data body request.SendAPDURequest true "APDU请求"
// @Success 200 {object} response.Response{data=response.SendAPDUResponse} "发送成功"
// @Router /nfc-relay/apdu/send [post]
func (a *NFCTransactionApi) SendAPDU(c *gin.Context) {
	var req request.SendAPDURequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.FailWithMessage(err.Error(), c)
		return
	}
	userUUID := utils.GetUserUuid(c)
	ctx := context.WithValue(context.Background(), "userID", userUUID)
	nfcTransactionService := service.ServiceGroupApp.NFCRelayServiceGroup.TransactionService()
	result, err := nfcTransactionService.SendAPDU(ctx, &req, userUUID)
	if err != nil {
		global.GVA_LOG.Error("发送APDU失败", zap.Error(err))
		response.FailWithMessage(err.Error(), c)
		return
	}
	response.OkWithDetailed(result, "发送APDU成功", c)
}

// GetAPDUList 获取APDU消息列表
// @Tags NFCTransaction
// @Summary 获取APDU消息列表
// @Description 根据交易ID获取APDU消息列表
// @Security ApiKeyAuth
// @accept application/json
// @Produce application/json
// @Param transaction_id query string true "交易ID"
// @Param page query int false "页码"
// @Param pageSize query int false "每页数量"
// @Success 200 {object} response.Response{data=response.APDUMessageListResponse} "获取成功"
// @Router /nfc-relay/apdu/list [get]
func (a *NFCTransactionApi) GetAPDUList(c *gin.Context) {
	var req request.GetAPDUListRequest
	if err := c.ShouldBindQuery(&req); err != nil {
		response.FailWithMessage(err.Error(), c)
		return
	}
	userUUID := utils.GetUserUuid(c)
	ctx := context.WithValue(context.Background(), "userID", userUUID)
	nfcTransactionService := service.ServiceGroupApp.NFCRelayServiceGroup.TransactionService()
	result, err := nfcTransactionService.GetAPDUList(ctx, &req, userUUID)
	if err != nil {
		global.GVA_LOG.Error("获取APDU列表失败", zap.Error(err))
		response.FailWithMessage(err.Error(), c)
		return
	}
	response.OkWithDetailed(result, "获取APDU列表成功", c)
}

// GetStatistics 获取交易统计信息
// @Tags NFCTransaction
// @Summary 获取交易统计信息
// @Description 获取NFC中继交易的统计信息
// @Security ApiKeyAuth
// @accept application/json
// @Produce application/json
// @Success 200 {object} response.Response{data=response.TransactionStatisticsResponse} "获取成功"
// @Router /nfc-relay/statistics [get]
func (a *NFCTransactionApi) GetStatistics(c *gin.Context) {
	var req request.GetStatisticsRequest
	if err := c.ShouldBindQuery(&req); err != nil {
		response.FailWithMessage(err.Error(), c)
		return
	}
	userUUID := utils.GetUserUuid(c)
	ctx := context.WithValue(context.Background(), "userID", userUUID)
	nfcTransactionService := service.ServiceGroupApp.NFCRelayServiceGroup.TransactionService()
	result, err := nfcTransactionService.GetStatistics(ctx, &req, userUUID)
	if err != nil {
		global.GVA_LOG.Error("获取统计数据失败", zap.Error(err))
		response.FailWithMessage(err.Error(), c)
		return
	}
	response.OkWithDetailed(result, "获取统计数据成功", c)
}

// ExportTransactions 导出交易记录
// @Tags NFCTransaction
// @Summary 导出交易记录
// @Description 将交易记录导出为Excel文件
// @Security ApiKeyAuth
// @Produce application/octet-stream
// @Success 200 {file} file "Excel文件"
// @Router /nfc-relay/transactions/export [get]
func (a *NFCTransactionApi) ExportTransactions(c *gin.Context) {
	// 实际的导出逻辑会比较复杂，这里仅为示例
	c.Writer.Header().Set("Content-Type", "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet")
	c.Writer.Header().Set("Content-Disposition", "attachment; filename=transactions.xlsx")
	// 此处应调用服务层生成Excel文件并写入c.Writer
	// ...
	c.Status(http.StatusOK)
}

// InitiateTransactionSession 发起交易会话
// @Tags NFCTransactionSession
// @Summary 发起交易会话
// @Description 发起一个新的交易会话，并为接收端生成一个会话ID
// @Security ApiKeyAuth
// @accept application/json
// @Produce application/json
// @Param data body nfcreq.InitiateTransactionSessionRequest true "发起会话请求"
// @Success 200 {object} response.Response{data=nfcreq.TransactionSessionResponse} "发起成功"
// @Router /nfc-relay/session/initiate [post]
func (a *NFCTransactionApi) InitiateTransactionSession(c *gin.Context) {
	var req nfcreq.InitiateTransactionSessionRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.FailWithMessage(err.Error(), c)
		return
	}
	claims := utils.GetUserInfo(c)
	if claims == nil {
		response.FailWithMessage("获取用户信息失败", c)
		return
	}
	userUUID := claims.UUID
	username := claims.Username
	ctx := context.Background()

	nfcTransactionService := service.ServiceGroupApp.NFCRelayServiceGroup.TransactionService()
	result, err := nfcTransactionService.InitiateTransactionSession(ctx, req, userUUID, username)
	if err != nil {
		global.GVA_LOG.Error("发起交易会话失败", zap.Error(err))
		response.FailWithMessage(err.Error(), c)
		return
	}
	response.OkWithDetailed(result, "发起交易会话成功", c)
}

// JoinTransactionSession 加入交易会话
// @Tags NFCTransactionSession
// @Summary 加入交易会话
// @Description 接收端使用会话ID加入一个已发起的交易会话
// @Security ApiKeyAuth
// @accept application/json
// @Produce application/json
// @Param data body nfcreq.JoinTransactionSessionRequest true "加入会话请求"
// @Success 200 {object} response.Response{data=nfcreq.TransactionSessionResponse} "加入成功"
// @Router /nfc-relay/session/join [post]
func (a *NFCTransactionApi) JoinTransactionSession(c *gin.Context) {
	var req nfcreq.JoinTransactionSessionRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.FailWithMessage(err.Error(), c)
		return
	}
	claims := utils.GetUserInfo(c)
	if claims == nil {
		response.FailWithMessage("获取用户信息失败", c)
		return
	}
	userUUID := claims.UUID
	username := claims.Username
	ctx := context.Background()

	nfcTransactionService := service.ServiceGroupApp.NFCRelayServiceGroup.TransactionService()
	result, err := nfcTransactionService.JoinTransactionSession(ctx, req, userUUID, username)
	if err != nil {
		global.GVA_LOG.Error("加入交易会话失败", zap.Error(err))
		response.FailWithMessage(err.Error(), c)
		return
	}
	response.OkWithDetailed(result, "加入交易会话成功", c)
}

// GetTransactionSession 获取交易会话状态
// @Tags NFCTransactionSession
// @Summary 获取交易会话状态
// @Description 根据会话ID查询交易会话的当前状态
// @Security ApiKeyAuth
// @Produce application/json
// @Param session_id path string true "会话ID"
// @Success 200 {object} response.Response{data=response.TransactionDetailResponse} "获取成功"
// @Router /nfc-relay/session/{session_id} [get]
func (a *NFCTransactionApi) GetTransactionSession(c *gin.Context) {
	sessionID := c.Param("session_id")
	if sessionID == "" {
		response.FailWithMessage("会话ID不能为空", c)
		return
	}

	// 企业级实践：会话ID通常映射到交易ID，或在缓存中有其映射关系
	// 假设 session_id 就是 transaction_id
	transactionID := sessionID
	userUUID := utils.GetUserUuid(c)

	ctx := context.Background()
	getReq := nfcreq.GetTransactionRequest{TransactionID: transactionID}

	nfcTransactionService := service.ServiceGroupApp.NFCRelayServiceGroup.TransactionService()
	transactionDetail, err := nfcTransactionService.GetTransaction(ctx, &getReq, userUUID)
	if err != nil {
		global.GVA_LOG.Error("获取交易会话详情失败", zap.Error(err), zap.String("sessionID", sessionID))
		response.FailWithMessage("获取交易会话详情失败", c)
		return
	}

	// 获取MQTT客户端连接状态
	if transactionDetail != nil && transactionDetail.TransmitterClientID != "" {
		mqttService := service.ServiceGroupApp.NFCRelayServiceGroup.MqttService()
		isOnline, _ := mqttService.CheckClientOnlineViaAPI(ctx, transactionDetail.TransmitterClientID)
		transactionDetail.TransmitterClientOnline = isOnline
	}
	if transactionDetail != nil && transactionDetail.ReceiverClientID != "" {
		mqttService := service.ServiceGroupApp.NFCRelayServiceGroup.MqttService()
		isOnline, _ := mqttService.CheckClientOnlineViaAPI(ctx, transactionDetail.ReceiverClientID)
		transactionDetail.ReceiverClientOnline = isOnline
	}

	response.OkWithDetailed(transactionDetail, "获取交易会话详情成功", c)
}

// CancelPairing 取消配对
// @Tags NFCPairing
// @Summary 取消配对请求
// @Description 取消一个正在等待的配对请求
// @Security ApiKeyAuth
// @accept application/json
// @Produce application/json
// @Param data body nfcreq.CancelPairingRequest true "取消配对请求"
// @Success 200 {object} response.Response "取消成功"
// @Router /nfc-relay/pairing/cancel [post]
func (n *NFCTransactionApi) CancelPairing(c *gin.Context) {
	var req nfcreq.CancelPairingRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.FailWithMessage(err.Error(), c)
		return
	}

	claims := utils.GetUserInfo(c)
	if claims == nil {
		response.FailWithMessage("获取用户信息失败", c)
		return
	}

	nfcTransactionService := service.ServiceGroupApp.NFCRelayServiceGroup.TransactionService()
	err := nfcTransactionService.CancelPairing(context.Background(), claims.UUID, req.Role)
	if err != nil {
		global.GVA_LOG.Error("取消配对失败", zap.Error(err), zap.String("userID", claims.GetUserID()))
		response.FailWithMessage("取消配对失败", c)
		return
	}
	response.OkWithMessage("取消配对请求已处理", c)
}

// GetPairingStatus 获取配对状态
// @Tags NFCPairing
// @Summary 获取配对状态
// @Description 客户端轮询此接口以检查其配对状态
// @Security ApiKeyAuth
// @Produce application/json
// @Param role query string true "角色 (transmitter 或 receiver)"
// @Success 200 {object} response.Response{data=nfc_relay.PairingStatus} "获取成功"
// @Router /nfc-relay/pairing/status [get]
func (a *NFCTransactionApi) GetPairingStatus(c *gin.Context) {
	role := c.Query("role")
	if role != "transmitter" && role != "receiver" {
		response.FailWithMessage("角色参数无效", c)
		return
	}
	claims := utils.GetUserInfo(c)
	if claims == nil {
		response.FailWithMessage("获取用户信息失败", c)
		return
	}

	nfcTransactionService := service.ServiceGroupApp.NFCRelayServiceGroup.TransactionService()
	status, err := nfcTransactionService.GetPairingStatus(context.Background(), claims.UUID, role)
	if err != nil {
		global.GVA_LOG.Error("获取配对状态失败", zap.Error(err), zap.String("userID", claims.GetUserID()))
		response.FailWithMessage("获取配对状态失败", c)
		return
	}

	response.OkWithData(status, c)
}
