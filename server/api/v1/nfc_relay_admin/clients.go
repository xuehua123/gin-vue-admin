package nfc_relay_admin

import (
	"strconv"
	"strings"
	"time"

	"github.com/flipped-aurora/gin-vue-admin/server/global"
	"github.com/flipped-aurora/gin-vue-admin/server/model/common/response"
	nfcResponse "github.com/flipped-aurora/gin-vue-admin/server/model/nfc_relay_admin/response"
	"github.com/flipped-aurora/gin-vue-admin/server/nfc_relay/handler"
	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

type ClientsApi struct{}

// GetClients 获取连接客户端列表
// @Summary 获取连接客户端列表
// @Description 支持分页和筛选的客户端列表查询
// @Tags NFC中继管理
// @Accept json
// @Produce json
// @Param page query int false "页码"
// @Param pageSize query int false "每页数量"
// @Param clientID query string false "客户端ID筛选"
// @Param userID query string false "用户ID筛选"
// @Param role query string false "角色筛选"
// @Param ipAddress query string false "IP地址筛选"
// @Success 200 {object} response.Response{data=nfcResponse.PaginatedClientListResponse}
// @Router /api/admin/nfc-relay/v1/clients [get]
func (c *ClientsApi) GetClients(ctx *gin.Context) {
	// 获取查询参数
	pageStr := ctx.DefaultQuery("page", "1")
	pageSizeStr := ctx.DefaultQuery("pageSize", "10")
	clientIDFilter := ctx.Query("clientID")
	userIDFilter := ctx.Query("userID")
	roleFilter := ctx.Query("role")
	ipFilter := ctx.Query("ipAddress")

	page, err := strconv.Atoi(pageStr)
	if err != nil || page < 1 {
		page = 1
	}

	pageSize, err := strconv.Atoi(pageSizeStr)
	if err != nil || pageSize < 1 || pageSize > 100 {
		pageSize = 10
	}

	// 获取全局Hub实例并进行类型断言
	if global.GVA_NFC_RELAY_HUB == nil {
		global.GVA_LOG.Error("NFC Relay Hub not initialized")
		response.FailWithMessage("NFC中继服务未初始化", ctx)
		return
	}

	hub, ok := global.GVA_NFC_RELAY_HUB.(*handler.Hub)
	if !ok {
		global.GVA_LOG.Error("Invalid NFC Relay Hub type")
		response.FailWithMessage("NFC中继服务类型错误", ctx)
		return
	}

	// 使用Hub的公共方法获取所有客户端
	allClients := hub.GetAllClients()
	var clientList []nfcResponse.ClientInfo

	for _, client := range allClients {
		// 基础信息提取
		clientInfo := nfcResponse.ClientInfo{
			ClientID:    client.ID,
			UserID:      client.UserID,
			DisplayName: client.DisplayName,
			IsOnline:    client.IsOnline,
			ConnectedAt: time.Now().Format(time.RFC3339), // 暂时使用当前时间，后续可以添加ConnectedAt字段到Client
		}

		// 获取角色信息 - Role是string类型
		if client.Role != "" {
			clientInfo.Role = client.Role
		} else {
			// 使用CurrentRole作为备选
			clientInfo.Role = string(client.CurrentRole)
		}

		// 获取IP地址 - 使用GetRemoteAddr方法
		clientInfo.IPAddress = client.GetRemoteAddr()

		// 获取会话ID
		clientInfo.SessionID = client.SessionID

		// 应用筛选条件
		if clientIDFilter != "" && !strings.Contains(client.ID, clientIDFilter) {
			continue
		}
		if userIDFilter != "" && !strings.Contains(client.UserID, userIDFilter) {
			continue
		}
		if roleFilter != "" && clientInfo.Role != roleFilter {
			continue
		}
		if ipFilter != "" && !strings.Contains(clientInfo.IPAddress, ipFilter) {
			continue
		}

		clientList = append(clientList, clientInfo)
	}

	// 计算分页
	total := len(clientList)
	start := (page - 1) * pageSize
	end := start + pageSize

	if start >= total {
		clientList = []nfcResponse.ClientInfo{}
	} else {
		if end > total {
			end = total
		}
		clientList = clientList[start:end]
	}

	resp := nfcResponse.PaginatedClientListResponse{
		List:     clientList,
		Total:    total,
		Page:     page,
		PageSize: pageSize,
	}

	response.OkWithData(resp, ctx)
}

// GetClientDetails 获取客户端详情
// @Summary 获取指定客户端的详细信息
// @Description 获取客户端的详细信息包括连接历史和统计
// @Tags NFC中继管理
// @Accept json
// @Produce json
// @Param clientID path string true "客户端ID"
// @Success 200 {object} response.Response{data=nfcResponse.ClientDetailsResponse}
// @Router /api/admin/nfc-relay/v1/clients/{clientID}/details [get]
func (c *ClientsApi) GetClientDetails(ctx *gin.Context) {
	clientID := ctx.Param("clientID")
	if clientID == "" {
		response.FailWithMessage("客户端ID不能为空", ctx)
		return
	}

	if global.GVA_NFC_RELAY_HUB == nil {
		response.FailWithMessage("NFC中继服务未初始化", ctx)
		return
	}

	hub, ok := global.GVA_NFC_RELAY_HUB.(*handler.Hub)
	if !ok {
		response.FailWithMessage("NFC中继服务类型错误", ctx)
		return
	}

	// 使用Hub的公共方法查找客户端
	client := hub.FindClientByID(clientID)
	if client == nil {
		response.FailWithMessage("客户端不存在", ctx)
		return
	}

	// 构建详细信息
	details := nfcResponse.ClientDetailsResponse{
		ClientID:    clientID,
		UserID:      client.UserID,
		DisplayName: client.DisplayName,
		IsOnline:    client.IsOnline,
		ConnectedAt: time.Now().Format(time.RFC3339), // 暂时使用当前时间
	}

	// 获取角色信息
	if client.Role != "" {
		details.Role = client.Role
	} else {
		details.Role = string(client.CurrentRole)
	}

	// 获取IP地址
	details.IPAddress = client.GetRemoteAddr()

	// 会话信息
	details.SessionID = client.SessionID

	// 认证状态
	details.UserAgent = ""     // Client结构体中没有UserAgent字段，暂时留空
	details.LastMessageAt = "" // Client结构体中没有LastMessageAt字段，暂时留空

	// 消息统计 - Client结构体中没有这些字段，暂时设为0
	details.SentMessageCount = 0
	details.ReceivedMessageCount = 0

	// 连接事件历史 - 暂时留空，可以后续添加
	details.ConnectionEvents = []nfcResponse.ConnectionEvent{}

	response.OkWithData(details, ctx)
}

// DisconnectClient 强制断开客户端连接
// @Summary 管理员强制断开指定客户端
// @Description 强制断开指定客户端的WebSocket连接
// @Tags NFC中继管理
// @Accept json
// @Produce json
// @Param clientID path string true "客户端ID"
// @Success 200 {object} response.Response
// @Router /api/admin/nfc-relay/v1/clients/{clientID}/disconnect [post]
func (c *ClientsApi) DisconnectClient(ctx *gin.Context) {
	clientID := ctx.Param("clientID")
	if clientID == "" {
		response.FailWithMessage("客户端ID不能为空", ctx)
		return
	}

	if global.GVA_NFC_RELAY_HUB == nil {
		response.FailWithMessage("NFC中继服务未初始化", ctx)
		return
	}

	hub, ok := global.GVA_NFC_RELAY_HUB.(*handler.Hub)
	if !ok {
		response.FailWithMessage("NFC中继服务类型错误", ctx)
		return
	}

	// 记录管理员操作
	adminUser := ctx.GetString("username")
	if adminUser == "" {
		adminUser = "unknown_admin"
	}

	global.GVA_LOG.Info("Admin disconnecting client",
		zap.String("clientID", clientID),
		zap.String("adminUser", adminUser),
	)

	// 使用Hub的公共方法断开客户端连接
	err := hub.DisconnectClientByID(clientID, "管理员强制断开连接")
	if err != nil {
		global.GVA_LOG.Error("Failed to disconnect client",
			zap.String("clientID", clientID),
			zap.Error(err),
		)
		response.FailWithMessage(err.Error(), ctx)
		return
	}

	global.GVA_LOG.Info("Client disconnected by admin successfully",
		zap.String("clientID", clientID),
	)

	response.OkWithMessage("客户端已断开连接", ctx)
}
