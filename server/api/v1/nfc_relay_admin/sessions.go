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

type SessionsApi struct{}

// GetSessions 获取活动会话列表
// @Summary 获取活动会话列表
// @Description 支持分页和筛选的会话列表查询
// @Tags NFC中继管理
// @Accept json
// @Produce json
// @Param page query int false "页码"
// @Param pageSize query int false "每页数量"
// @Param sessionID query string false "会话ID筛选"
// @Param participantClientID query string false "参与方客户端ID筛选"
// @Param participantUserID query string false "参与方用户ID筛选"
// @Success 200 {object} response.Response{data=nfcResponse.PaginatedSessionListResponse}
// @Router /admin/nfc-relay/v1/sessions [get]
func (s *SessionsApi) GetSessions(ctx *gin.Context) {
	// 获取查询参数
	pageStr := ctx.DefaultQuery("page", "1")
	pageSizeStr := ctx.DefaultQuery("pageSize", "10")
	sessionIDFilter := ctx.Query("sessionID")
	participantClientIDFilter := ctx.Query("participantClientID")
	participantUserIDFilter := ctx.Query("participantUserID")

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

	// 使用Hub的公共方法获取所有会话
	allSessions := hub.GetAllSessions()
	var sessionList []nfcResponse.SessionInfo

	for sessionID, session := range allSessions {
		// 基础信息提取
		sessionInfo := nfcResponse.SessionInfo{
			SessionID: sessionID,
			Status:    "active", // 默认活动状态
			CreatedAt: session.CreatedAt.Format(time.RFC3339),
		}

		// 提取Provider信息 - 需要通过Hub查找对应的Client来获取DisplayName
		if session.CardEndClient != nil {
			sessionInfo.ProviderClientID = session.CardEndClient.GetID()
			sessionInfo.ProviderUserID = session.CardEndClient.GetUserID()
			// 从Hub中查找对应的Client来获取DisplayName
			if client := hub.FindClientByID(session.CardEndClient.GetID()); client != nil {
				sessionInfo.ProviderDisplayName = client.DisplayName
			}
		}

		// 提取Receiver信息 - 需要通过Hub查找对应的Client来获取DisplayName
		if session.POSEndClient != nil {
			sessionInfo.ReceiverClientID = session.POSEndClient.GetID()
			sessionInfo.ReceiverUserID = session.POSEndClient.GetUserID()
			// 从Hub中查找对应的Client来获取DisplayName
			if client := hub.FindClientByID(session.POSEndClient.GetID()); client != nil {
				sessionInfo.ReceiverDisplayName = client.DisplayName
			}
		}

		// 更新最后活动时间 - 使用正确的字段名LastActivityTime
		if !session.LastActivityTime.IsZero() {
			sessionInfo.LastActivityAt = session.LastActivityTime.Format(time.RFC3339)
		}

		// 应用筛选条件
		if sessionIDFilter != "" && !strings.Contains(sessionID, sessionIDFilter) {
			continue
		}
		if participantClientIDFilter != "" {
			if !strings.Contains(sessionInfo.ProviderClientID, participantClientIDFilter) &&
				!strings.Contains(sessionInfo.ReceiverClientID, participantClientIDFilter) {
				continue
			}
		}
		if participantUserIDFilter != "" {
			if !strings.Contains(sessionInfo.ProviderUserID, participantUserIDFilter) &&
				!strings.Contains(sessionInfo.ReceiverUserID, participantUserIDFilter) {
				continue
			}
		}

		sessionList = append(sessionList, sessionInfo)
	}

	// 计算分页
	total := len(sessionList)
	start := (page - 1) * pageSize
	end := start + pageSize

	if start >= total {
		sessionList = []nfcResponse.SessionInfo{}
	} else {
		if end > total {
			end = total
		}
		sessionList = sessionList[start:end]
	}

	resp := nfcResponse.PaginatedSessionListResponse{
		List:     sessionList,
		Total:    total,
		Page:     page,
		PageSize: pageSize,
	}

	response.OkWithData(resp, ctx)
}

// GetSessionDetails 获取会话详情
// @Summary 获取指定会话的详细信息
// @Description 获取会话的详细信息包括参与者信息和交换统计
// @Tags NFC中继管理
// @Accept json
// @Produce json
// @Param sessionID path string true "会话ID"
// @Success 200 {object} response.Response{data=nfcResponse.SessionDetailsResponse}
// @Router /admin/nfc-relay/v1/sessions/{sessionID}/details [get]
func (s *SessionsApi) GetSessionDetails(ctx *gin.Context) {
	sessionID := ctx.Param("sessionID")
	if sessionID == "" {
		response.FailWithMessage("会话ID不能为空", ctx)
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

	// 使用Hub的公共方法获取会话
	session, exists := hub.GetSessionByID(sessionID)
	if !exists {
		response.FailWithMessage("会话不存在", ctx)
		return
	}

	// 构建详细信息
	details := nfcResponse.SessionDetailsResponse{
		SessionID: sessionID,
		Status:    "active",
		CreatedAt: session.CreatedAt.Format(time.RFC3339),
	}

	// 使用正确的字段名LastActivityTime
	if !session.LastActivityTime.IsZero() {
		details.LastActivityAt = session.LastActivityTime.Format(time.RFC3339)
	}

	// Provider信息
	if session.CardEndClient != nil {
		details.ProviderInfo = nfcResponse.ParticipantInfo{
			ClientID: session.CardEndClient.GetID(),
			UserID:   session.CardEndClient.GetUserID(),
		}
		// 获取DisplayName和IP地址 - 需要从Hub中查找对应的Client
		if client := hub.FindClientByID(session.CardEndClient.GetID()); client != nil {
			details.ProviderInfo.DisplayName = client.DisplayName
			details.ProviderInfo.IPAddress = client.GetRemoteAddr()
		}
	}

	// Receiver信息
	if session.POSEndClient != nil {
		details.ReceiverInfo = nfcResponse.ParticipantInfo{
			ClientID: session.POSEndClient.GetID(),
			UserID:   session.POSEndClient.GetUserID(),
		}
		// 获取DisplayName和IP地址 - 需要从Hub中查找对应的Client
		if client := hub.FindClientByID(session.POSEndClient.GetID()); client != nil {
			details.ReceiverInfo.DisplayName = client.DisplayName
			details.ReceiverInfo.IPAddress = client.GetRemoteAddr()
		}
	}

	// APDU交换统计 - 使用正确的getter方法
	details.ApduExchangeCount = nfcResponse.ApduExchangeCount{
		Upstream:   session.GetUpstreamAPDUCount(),
		Downstream: session.GetDownstreamAPDUCount(),
	}

	// 会话事件历史 - 使用正确的类型
	details.SessionEventsHistory = []nfcResponse.SessionEvent{}

	response.OkWithData(details, ctx)
}

// TerminateSession 强制终止会话
// @Summary 管理员强制终止指定会话
// @Description 强制终止指定的NFC中继会话
// @Tags NFC中继管理
// @Accept json
// @Produce json
// @Param sessionID path string true "会话ID"
// @Success 200 {object} response.Response
// @Router /admin/nfc-relay/v1/sessions/{sessionID}/terminate [post]
func (s *SessionsApi) TerminateSession(ctx *gin.Context) {
	sessionID := ctx.Param("sessionID")
	if sessionID == "" {
		response.FailWithMessage("会话ID不能为空", ctx)
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

	global.GVA_LOG.Info("Admin terminating session",
		zap.String("sessionID", sessionID),
		zap.String("adminUser", adminUser),
	)

	// 使用Hub的公共方法终止会话
	err := hub.TerminateSessionByAdmin(sessionID, "管理员操作终止", adminUser)
	if err != nil {
		global.GVA_LOG.Error("Failed to terminate session",
			zap.String("sessionID", sessionID),
			zap.Error(err),
		)
		response.FailWithMessage(err.Error(), ctx)
		return
	}

	global.GVA_LOG.Info("Session terminated by admin successfully",
		zap.String("sessionID", sessionID),
	)

	response.OkWithMessage("会话已终止", ctx)
}
