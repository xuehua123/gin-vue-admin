package service

import (
	"errors"
	"strconv"
	"strings"
	"time"

	"github.com/flipped-aurora/gin-vue-admin/server/global"
	"github.com/flipped-aurora/gin-vue-admin/server/nfc_relay/handler"
	"github.com/flipped-aurora/gin-vue-admin/server/nfc_relay/model/admin_request"
	"github.com/flipped-aurora/gin-vue-admin/server/nfc_relay/model/admin_response"
	"github.com/flipped-aurora/gin-vue-admin/server/nfc_relay/session"
	"go.uber.org/zap"
)

// AdminSessionService 结构体，用于会话管理相关的服务
type AdminSessionService struct{}

// GetSessionList 获取会话列表，支持分页和筛选
func (s *AdminSessionService) GetSessionList(params admin_request.SessionListParams) (response admin_response.PaginatedSessionListResponse, err error) {
	// 默认值处理
	if params.Page <= 0 {
		params.Page = 1
	}
	if params.PageSize <= 0 {
		params.PageSize = 10
	} else if params.PageSize > 100 {
		params.PageSize = 100 // 限制最大每页条数
	}

	// 获取 Hub 实例
	hub := handler.GlobalRelayHub
	if hub == nil {
		return response, errors.New("NFC Relay Hub 未初始化")
	}

	// 获取所有会话 - 现在使用新方法
	allSessions := hub.GetAllSessions()
	global.GVA_LOG.Info("获取到会话列表", zap.Int("sessionCount", len(allSessions)))

	// 创建会话信息列表
	var sessionInfoList []admin_response.SessionInfo

	// 从会话映射中收集信息
	for sessionID, sessionObj := range allSessions {
		// 从会话中获取客户端信息
		var providerClient, receiverClient session.ClientInfoProvider
		if sessionObj.CardEndClient != nil && sessionObj.CardEndClient.GetRole() == "provider" {
			providerClient = sessionObj.CardEndClient
		}
		if sessionObj.POSEndClient != nil && sessionObj.POSEndClient.GetRole() == "receiver" {
			receiverClient = sessionObj.POSEndClient
		}

		// 如果角色分配不是上述默认情况
		if providerClient == nil && sessionObj.POSEndClient != nil && sessionObj.POSEndClient.GetRole() == "provider" {
			providerClient = sessionObj.POSEndClient
		}
		if receiverClient == nil && sessionObj.CardEndClient != nil && sessionObj.CardEndClient.GetRole() == "receiver" {
			receiverClient = sessionObj.CardEndClient
		}

		// 创建会话信息对象
		sessionInfo := admin_response.SessionInfo{
			SessionID:      sessionID,
			Status:         "paired", // 默认状态，可根据会话对象中的标志更新
			CreatedAt:      sessionObj.CreatedAt,
			LastActivityAt: sessionObj.LastActivityTime,
		}

		// 会话状态判断
		if sessionObj.IsTerminated() {
			sessionInfo.Status = "terminated"
		}

		// 填充提供者信息
		if providerClient != nil {
			sessionInfo.ProviderClientID = providerClient.GetID()
			sessionInfo.ProviderUserID = providerClient.GetUserID()
			// 尝试获取显示名称 - 需要类型断言
			if client, ok := providerClient.(*handler.Client); ok && client != nil {
				sessionInfo.ProviderDisplayName = client.DisplayName
			}
		}

		// 填充接收者信息
		if receiverClient != nil {
			sessionInfo.ReceiverClientID = receiverClient.GetID()
			sessionInfo.ReceiverUserID = receiverClient.GetUserID()
			// 尝试获取显示名称 - 需要类型断言
			if client, ok := receiverClient.(*handler.Client); ok && client != nil {
				sessionInfo.ReceiverDisplayName = client.DisplayName
			}
		}

		sessionInfoList = append(sessionInfoList, sessionInfo)
	}

	// 应用筛选条件
	var filteredSessionInfoList []admin_response.SessionInfo
	for _, sessionInfo := range sessionInfoList {
		// 检查是否满足所有筛选条件
		if params.SessionID != "" && !strings.Contains(sessionInfo.SessionID, params.SessionID) {
			continue
		}
		if params.ParticipantClientID != "" &&
			!strings.Contains(sessionInfo.ProviderClientID, params.ParticipantClientID) &&
			!strings.Contains(sessionInfo.ReceiverClientID, params.ParticipantClientID) {
			continue
		}
		if params.ParticipantUserID != "" &&
			!strings.Contains(sessionInfo.ProviderUserID, params.ParticipantUserID) &&
			!strings.Contains(sessionInfo.ReceiverUserID, params.ParticipantUserID) {
			continue
		}
		if params.Status != "" && sessionInfo.Status != params.Status {
			continue
		}

		// 添加到过滤后的列表
		filteredSessionInfoList = append(filteredSessionInfoList, sessionInfo)
	}

	// 计算总记录数
	total := int64(len(filteredSessionInfoList))

	// 计算分页
	start := (params.Page - 1) * params.PageSize
	end := params.Page * params.PageSize
	if start >= int(total) {
		// 页码超出范围，返回空列表
		response.List = []admin_response.SessionInfo{}
	} else {
		if end > int(total) {
			end = int(total)
		}
		response.List = filteredSessionInfoList[start:end]
	}

	response.Total = total
	response.Page = params.Page
	response.PageSize = params.PageSize

	return response, nil
}

// GetSessionDetail 获取单个会话的详细信息
func (s *AdminSessionService) GetSessionDetail(sessionID string) (response admin_response.SessionDetailsResponse, err error) {
	// 获取 Hub 实例
	hub := handler.GlobalRelayHub
	if hub == nil {
		return response, errors.New("NFC Relay Hub 未初始化")
	}

	// 使用新方法获取会话
	sessionObj, exists := hub.GetSessionByID(sessionID)
	if !exists || sessionObj == nil {
		return response, errors.New("未找到指定的会话")
	}

	// 从会话中获取客户端信息
	var providerClient, receiverClient session.ClientInfoProvider
	if sessionObj.CardEndClient != nil && sessionObj.CardEndClient.GetRole() == "provider" {
		providerClient = sessionObj.CardEndClient
	}
	if sessionObj.POSEndClient != nil && sessionObj.POSEndClient.GetRole() == "receiver" {
		receiverClient = sessionObj.POSEndClient
	}

	// 如果角色分配不是上述默认情况
	if providerClient == nil && sessionObj.POSEndClient != nil && sessionObj.POSEndClient.GetRole() == "provider" {
		providerClient = sessionObj.POSEndClient
	}
	if receiverClient == nil && sessionObj.CardEndClient != nil && sessionObj.CardEndClient.GetRole() == "receiver" {
		receiverClient = sessionObj.CardEndClient
	}

	// 填充基本信息
	response.SessionID = sessionID
	response.Status = "paired" // 默认状态
	response.CreatedAt = sessionObj.CreatedAt
	response.LastActivityAt = sessionObj.LastActivityTime

	// 会话状态判断
	if sessionObj.IsTerminated() {
		response.Status = "terminated"
		terminatedAt := sessionObj.TerminatedAt
		response.TerminatedAt = &terminatedAt
		response.TerminationReason = sessionObj.TerminationReason
	}

	// 填充Provider信息
	if providerClient != nil {
		providerInfo := &admin_response.ClientSummaryInfo{
			ClientID: providerClient.GetID(),
			UserID:   providerClient.GetUserID(),
		}

		// 尝试获取更多信息
		if client, ok := providerClient.(*handler.Client); ok && client != nil {
			providerInfo.DisplayName = client.DisplayName
			providerInfo.IPAddress = client.GetRemoteAddr()
		}

		response.ProviderInfo = providerInfo
	}

	// 填充Receiver信息
	if receiverClient != nil {
		receiverInfo := &admin_response.ClientSummaryInfo{
			ClientID: receiverClient.GetID(),
			UserID:   receiverClient.GetUserID(),
		}

		// 尝试获取更多信息
		if client, ok := receiverClient.(*handler.Client); ok && client != nil {
			receiverInfo.DisplayName = client.DisplayName
			receiverInfo.IPAddress = client.GetRemoteAddr()
		}

		response.ReceiverInfo = receiverInfo
	}

	// APDU交换计数
	response.APDUExchangeCount = admin_response.APDUExchangeCount{
		Upstream:   sessionObj.GetUpstreamAPDUCount(),
		Downstream: sessionObj.GetDownstreamAPDUCount(),
	}

	// 会话事件历史
	response.SessionEventsHistory = []admin_response.SessionEvent{
		{
			Timestamp: sessionObj.CreatedAt,
			Event:     "SessionCreated",
		},
	}

	// 添加更多会话事件
	if sessionObj.IsTerminated() {
		response.SessionEventsHistory = append(response.SessionEventsHistory, admin_response.SessionEvent{
			Timestamp: sessionObj.TerminatedAt,
			Event:     "SessionTerminated",
			Details:   sessionObj.TerminationReason,
		})
	}

	// 相关审计日志摘要（这部分可以从其他服务获取，此处为示例）
	response.RelatedAuditLogsSummary = []admin_response.RelatedAuditLogItem{
		{
			Timestamp:      time.Now().Add(-5 * time.Minute),
			EventType:      "apdu_relayed",
			DetailsSummary: "APDU成功中继",
		},
	}

	return response, nil
}

// TerminateSession 终止指定的会话
func (s *AdminSessionService) TerminateSession(sessionID, reason string, actingUserID uint) error {
	// 获取 Hub 实例
	hub := handler.GlobalRelayHub
	if hub == nil {
		return errors.New("NFC Relay Hub 未初始化")
	}

	// 记录操作日志
	global.GVA_LOG.Info("管理员操作：准备终止会话",
		zap.String("sessionID", sessionID),
		zap.String("reason", reason),
		zap.Uint("actingUserID", actingUserID),
	)

	// 使用新方法终止会话
	adminUserIDStr := strconv.FormatUint(uint64(actingUserID), 10)
	if reason == "" {
		reason = "管理员操作终止会话"
	}

	err := hub.TerminateSessionByAdmin(sessionID, reason, adminUserIDStr)
	if err != nil {
		global.GVA_LOG.Error("终止会话失败", zap.Error(err), zap.String("sessionID", sessionID))
		return err
	}

	global.GVA_LOG.Info("会话终止成功", zap.String("sessionID", sessionID), zap.String("reason", reason))
	return nil
}
