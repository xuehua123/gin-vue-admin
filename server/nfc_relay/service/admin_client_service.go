package service

import (
	"errors"
	"strings"
	"time"

	"github.com/flipped-aurora/gin-vue-admin/server/global"
	"github.com/flipped-aurora/gin-vue-admin/server/nfc_relay/handler"
	"github.com/flipped-aurora/gin-vue-admin/server/nfc_relay/model/admin_request"
	"github.com/flipped-aurora/gin-vue-admin/server/nfc_relay/model/admin_response"
	"go.uber.org/zap"
)

// AdminClientService 结构体，用于客户端管理相关的服务
type AdminClientService struct{}

// GetClientList 获取客户端列表，支持分页和筛选
func (s *AdminClientService) GetClientList(params admin_request.ClientListParams) (response admin_response.PaginatedClientListResponse, err error) {
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

	// 获取所有客户端
	clients := hub.GetAllClients()

	// 创建客户端信息列表
	var clientInfoList []admin_response.ClientInfo
	var filteredClientInfoList []admin_response.ClientInfo

	// 转换客户端信息
	for _, client := range clients {
		// 将客户端信息转换为 DTO
		clientInfo := admin_response.ClientInfo{
			ClientID:    client.ID,
			UserID:      client.UserID,
			DisplayName: client.DisplayName,
			Role:        string(client.CurrentRole),
			IPAddress:   client.GetRemoteAddr(),
			// ConnectedAt 和 LastMessageAt 字段需要在 Client 结构体中添加，这里假设已添加
			// 这部分需要修改 Client 结构体，暂时使用当前时间代替
			ConnectedAt:   time.Now(), // 实际实现需修改
			IsOnline:      client.IsOnline,
			SessionID:     client.SessionID,
			LastMessageAt: time.Now(), // 实际实现需修改
		}

		clientInfoList = append(clientInfoList, clientInfo)
	}

	// 应用筛选条件
	for _, clientInfo := range clientInfoList {
		// 检查是否满足所有筛选条件
		if params.ClientID != "" && !strings.Contains(clientInfo.ClientID, params.ClientID) {
			continue
		}
		if params.UserID != "" && !strings.Contains(clientInfo.UserID, params.UserID) {
			continue
		}
		if params.Role != "" && clientInfo.Role != params.Role {
			continue
		}
		if params.IPAddress != "" && !strings.Contains(clientInfo.IPAddress, params.IPAddress) {
			continue
		}
		if params.SessionID != "" && clientInfo.SessionID != params.SessionID {
			continue
		}
		if params.IsOnline != nil && clientInfo.IsOnline != *params.IsOnline {
			continue
		}

		// 添加到过滤后的列表
		filteredClientInfoList = append(filteredClientInfoList, clientInfo)
	}

	// 计算总记录数
	total := int64(len(filteredClientInfoList))

	// 计算分页
	start := (params.Page - 1) * params.PageSize
	end := params.Page * params.PageSize
	if start >= int(total) {
		// 页码超出范围，返回空列表
		response.List = []admin_response.ClientInfo{}
	} else {
		if end > int(total) {
			end = int(total)
		}
		response.List = filteredClientInfoList[start:end]
	}

	response.Total = total
	response.Page = params.Page
	response.PageSize = params.PageSize

	return response, nil
}

// GetClientDetail 获取单个客户端的详细信息
func (s *AdminClientService) GetClientDetail(clientID string) (response admin_response.ClientDetailResponse, err error) {
	// 获取 Hub 实例
	hub := handler.GlobalRelayHub
	if hub == nil {
		return response, errors.New("NFC Relay Hub 未初始化")
	}

	// 查找指定的客户端
	targetClient := hub.FindClientByID(clientID)

	// 如果未找到客户端
	if targetClient == nil {
		return response, errors.New("未找到指定的客户端")
	}

	// 填充基本信息
	response.ClientInfo = admin_response.ClientInfo{
		ClientID:    targetClient.ID,
		UserID:      targetClient.UserID,
		DisplayName: targetClient.DisplayName,
		Role:        string(targetClient.CurrentRole),
		IPAddress:   targetClient.GetRemoteAddr(),
		// ConnectedAt 和 LastMessageAt 字段需要在 Client 结构体中添加，这里假设已添加
		ConnectedAt:   time.Now(), // 实际实现需修改
		IsOnline:      targetClient.IsOnline,
		SessionID:     targetClient.SessionID,
		LastMessageAt: time.Now(), // 实际实现需修改
	}

	// 这里可以添加更多详细信息，如消息计数、连接事件等
	// 但这些信息需要在 Client 结构体中添加相应字段进行跟踪
	// 暂时返回一些模拟数据
	response.SentMessageCount = 0
	response.ReceivedMessageCount = 0
	response.ConnectionEvents = []admin_response.ConnectionEvent{
		{
			Timestamp: time.Now(),
			Event:     "Connected",
			Details:   "客户端已连接",
		},
	}
	response.RelatedAuditLogs = []admin_response.RelatedAuditLogItem{
		{
			Timestamp:      time.Now(),
			EventType:      "connection",
			DetailsSummary: "客户端连接事件",
		},
	}

	return response, nil
}

// DisconnectClient 强制断开客户端的连接
func (s *AdminClientService) DisconnectClient(clientID string) error {
	// 获取 Hub 实例
	hub := handler.GlobalRelayHub
	if hub == nil {
		return errors.New("NFC Relay Hub 未初始化")
	}

	// 记录操作日志
	global.GVA_LOG.Info("管理员操作：准备强制断开客户端连接", zap.String("clientID", clientID))

	// 调用 Hub 的断开连接方法
	return hub.DisconnectClientByID(clientID, "管理员强制断开连接")
}
