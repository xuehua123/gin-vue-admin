package handler

import (
	"encoding/json"
	"errors"
	"fmt"
	"strconv"
	"sync"
	"time"

	"github.com/flipped-aurora/gin-vue-admin/server/global"
	//"github.com/flipped-aurora/gin-vue-admin/server/model/system/request" // JWT claims 结构定义，必须保留
	"github.com/flipped-aurora/gin-vue-admin/server/nfc_relay/protocol"
	"github.com/flipped-aurora/gin-vue-admin/server/nfc_relay/session"
	"github.com/flipped-aurora/gin-vue-admin/server/utils" // 导入JWT工具包
	"github.com/google/uuid"

	// "github.com/prometheus/client_golang/prometheus" // Removed: imported and not used
	"go.uber.org/zap"
)

// Hub 维护活动客户端的集合并将消息广播给客户端。
// TODO: 这个 Hub 是一个简化版本。后续会针对会话管理、特定的客户端到客户端消息路由，
// 以及与 nfc_relay/session 和 nfc_relay/service 的集成进行增强。
// 目前，它主要管理客户端的注册/注销，并可以广播消息。
type Hub struct {
	// 已注册的客户端。
	// 使用 map，键是 *Client 结构体指针，值是布尔型 (true 表示客户端存在)。
	// 这样可以高效地添加、删除和检查存在性。
	clients map[*Client]bool

	// 来自客户端的需要处理或广播的入站消息。
	// 为简单起见，使用通用的 []byte。在更复杂的系统中，这可能是一个结构化的消息类型。
	// broadcast chan []byte // 旧的广播通道，暂时保留或移除，根据后续消息处理逻辑决定
	// 替换为更结构化的消息处理通道
	processMessage chan ProcessableMessage // 用于处理来自客户端的特定消息

	// 来自客户端的注册请求。
	register chan *Client

	// 来自客户端的注销请求。
	unregister chan *Client

	// sessions 用于存储活动会话，键是 SessionID
	sessions map[string]*session.Session

	// cardProviders 存储当前声明为 'provider' (发卡方) 并在线的客户端。
	// 键是客户端的唯一ID (client.ID)，值是客户端的接口表示。
	cardProviders map[string]session.ClientInfoProvider

	// providerListSubscribers 跟踪哪些客户端订阅了特定用户ID的发卡方列表更新
	// Key: UserID, Value: map[*Client]bool (订阅了该UserID列表的客户端集合)
	providerListSubscribers map[string]map[*Client]bool

	// providerMutex 保护对 cardProviders, sessions, providerListSubscribers 的并发访问
	providerMutex sync.RWMutex

	// metricsMutex 保护对 Prometheus 指标的并发访问
	metricsMutex sync.Mutex
}

// NewHub 创建一个新的 Hub 实例。
func NewHub() *Hub {
	return &Hub{
		// broadcast:      make(chan []byte),
		processMessage:          make(chan ProcessableMessage),
		register:                make(chan *Client),
		unregister:              make(chan *Client),
		clients:                 make(map[*Client]bool),
		sessions:                make(map[string]*session.Session),
		cardProviders:           make(map[string]session.ClientInfoProvider),
		providerListSubscribers: make(map[string]map[*Client]bool),
		// pendingConnections: make(map[string]*Client), // 已移除
	}
}

// GetActiveConnectionsCount 返回当前活动的 WebSocket 连接总数。
// 此方法是并发安全的。
func (h *Hub) GetActiveConnectionsCount() int {
	h.providerMutex.RLock() // 虽然 clients map 本身的并发由 Hub 的主循环保证，但为了与其他 getter 一致性，这里也加锁
	defer h.providerMutex.RUnlock()
	return len(h.clients)
}

// GetActiveSessionsCount 返回当前活动的 NFC 中继会话总数。
// 此方法是并发安全的。
func (h *Hub) GetActiveSessionsCount() int {
	h.providerMutex.RLock()
	defer h.providerMutex.RUnlock()
	return len(h.sessions)
}

// Run 启动 Hub 的事件处理循环。
// 它应该以 goroutine 方式运行。
func (h *Hub) Run() {
	// 从配置加载 Hub 检查时间间隔
	hubCheckInterval := time.Duration(global.GVA_CONFIG.NfcRelay.HubCheckIntervalSec) * time.Second
	if hubCheckInterval <= 0 {
		hubCheckInterval = 60 * time.Second // Default to 60 seconds if config is invalid
		global.GVA_LOG.Warn("HubCheckIntervalSec config is invalid, using default 60s")
	}
	checkTicker := time.NewTicker(hubCheckInterval)
	defer checkTicker.Stop()

	for {
		select {
		case client := <-h.register:
			h.clients[client] = true
			h.metricsMutex.Lock()
			ActiveConnections.Inc() // Increment active connections metric
			h.metricsMutex.Unlock()
			global.GVA_LOG.Info("客户端已注册到 Hub", zap.String("clientID", client.ID), zap.String("clientAddr", client.conn.RemoteAddr().String()))

		case client := <-h.unregister:
			if _, ok := h.clients[client]; ok {
				delete(h.clients, client)
				close(client.send) // 重要：关闭客户端的 send 通道
				h.metricsMutex.Lock()
				ActiveConnections.Dec() // Decrement active connections metric
				h.metricsMutex.Unlock()
				global.GVA_LOG.Info("客户端已从 Hub 注销", zap.String("clientID", client.ID), zap.String("clientAddr", client.conn.RemoteAddr().String()))

				// 处理客户端断开连接与会话的关联
				h.handleClientDisconnect(client)
			}

		case procMsg := <-h.processMessage:
			h.handleIncomingMessage(procMsg)

		case <-checkTicker.C:
			h.checkInactiveSessions()

			// case messageBytes := <-h.broadcast: // 旧的广播逻辑，暂时注释
			// 	global.GVA_LOG.Info("Hub 正在广播消息", zap.ByteString("message", messageBytes))
			// 	for client := range h.clients {
			// 		select {
			// 		case client.send <- messageBytes:
			// 		default:
			// 			close(client.send)
			// 			delete(h.clients, client)
			// 		}
			// 	}
		}
	}
}

// handleIncomingMessage 处理来自客户端的已包装消息
func (h *Hub) handleIncomingMessage(procMsg ProcessableMessage) {
	client := procMsg.Client
	messageBytes := procMsg.RawMessage

	var genericMsg protocol.GenericMessage
	if err := json.Unmarshal(messageBytes, &genericMsg); err != nil {
		global.GVA_LOG.Error("Hub 处理消息：反序列化通用消息失败", zap.Error(err), zap.String("clientID", client.GetID()))
		// Audit Log for bad request format
		global.LogAuditEvent(
			"error_occurred",
			global.ErrorDetails{
				ErrorCode:    strconv.Itoa(protocol.ErrorCodeBadRequest),
				ErrorMessage: "无效的消息格式: " + err.Error(),
				Component:    "nfc_relay_hub.handleIncomingMessage",
				AffectedData: string(messageBytes),
			},
			zap.String("clientID", client.GetID()),
			zap.String("user_id", client.UserID),
			zap.String("source_ip", client.conn.RemoteAddr().String()),
		)
		sendErrorMessage(client, protocol.ErrorCodeBadRequest, "无效的消息格式")
		return
	}

	// 消息类型检查：除了认证消息，其他都需要认证
	if genericMsg.Type != protocol.MessageTypeClientAuth && !client.Authenticated {
		global.GVA_LOG.Warn("未认证的客户端尝试发送非认证消息",
			zap.String("clientID", client.GetID()),
			zap.String("messageType", string(genericMsg.Type)),
		)
		// Audit Log for auth required
		global.LogAuditEvent(
			"error_occurred",
			global.ErrorDetails{
				ErrorCode:    strconv.Itoa(protocol.ErrorCodeAuthRequired),
				ErrorMessage: "请先进行认证",
				Component:    "nfc_relay_hub.handleIncomingMessage",
			},
			zap.String("clientID", client.GetID()),
			zap.String("user_id", client.UserID),
			zap.String("source_ip", client.conn.RemoteAddr().String()),
			zap.String("message_type", string(genericMsg.Type)),
		)
		sendErrorMessage(client, protocol.ErrorCodeAuthRequired, "请先进行认证")
		return
	}

	global.GVA_LOG.Info("Hub 正在处理消息",
		zap.String("clientID", client.GetID()),
		zap.String("userID", client.UserID),
		zap.Bool("authenticated", client.Authenticated),
		zap.String("currentRole", string(client.CurrentRole)),
		zap.ByteString("message", messageBytes),
	)

	switch genericMsg.Type {
	case protocol.MessageTypeClientAuth:
		h.handleClientAuth(client, messageBytes)
	// case protocol.MessageTypeClientRegister: // 已废弃
	// 	global.GVA_LOG.Info("收到 ClientRegister 消息 (已废弃)", zap.String("clientID", client.GetID()))
	case protocol.MessageTypeDeclareRole:
		h.handleDeclareRole(client, messageBytes)
	case protocol.MessageTypeListCardProviders:
		if client.CurrentRole != protocol.RoleReceiver {
			global.GVA_LOG.Warn("Hub: 非 receiver 客户端尝试获取发卡方列表",
				zap.String("clientID", client.GetID()),
				zap.String("currentRole", string(client.CurrentRole)),
			)
			sendErrorMessage(client, protocol.ErrorCodePermissionDenied, "只有收卡方角色才能获取发卡方列表")
			return
		}
		h.handleListCardProviders(client, messageBytes)
	case protocol.MessageTypeSelectCardProvider:
		h.handleSelectCardProvider(client, messageBytes)
	case protocol.MessageTypeAPDUUpstream:
		h.handleAPDUExchange(client, messageBytes, "upstream")
	case protocol.MessageTypeAPDUDownstream:
		h.handleAPDUExchange(client, messageBytes, "downstream")
	case protocol.MessageTypeEndSession:
		h.handleEndSession(client, messageBytes)
	default:
		global.GVA_LOG.Warn("Hub 收到未处理的消息类型",
			zap.String("type", string(genericMsg.Type)),
			zap.String("clientID", client.ID),
		)
		// Audit Log for unsupported message type
		global.LogAuditEvent(
			"error_occurred",
			global.ErrorDetails{
				ErrorCode:    strconv.Itoa(protocol.ErrorCodeUnsupportedType),
				ErrorMessage: "不支持的消息类型: " + string(genericMsg.Type),
				Component:    "nfc_relay_hub.handleIncomingMessage",
			},
			zap.String("clientID", client.GetID()),
			zap.String("user_id", client.UserID),
			zap.String("source_ip", client.conn.RemoteAddr().String()),
			zap.String("message_type", string(genericMsg.Type)),
		)
		sendErrorMessage(client, protocol.ErrorCodeUnsupportedType, "不支持的消息类型: "+string(genericMsg.Type))
	}
}

// handleClientAuth 处理客户端认证 (Phase A)
func (h *Hub) handleClientAuth(client *Client, rawMsg json.RawMessage) {
	var authMsg protocol.ClientAuthMessage
	if err := json.Unmarshal(rawMsg, &authMsg); err != nil {
		global.GVA_LOG.Error("Hub: Error unmarshalling auth message", zap.Error(err), zap.String("clientID", client.GetID()))
		sendErrorMessage(client, protocol.ErrorCodeBadRequest, "Invalid auth message format")
		return
	}

	if authMsg.Token == "" {
		global.GVA_LOG.Warn("Hub: Token is missing in auth message", zap.String("clientID", client.GetID()))
		sendErrorMessage(client, protocol.ErrorCodeAuthFailed, "Token is missing")
		return
	}

	jwtService := utils.NewJWT()                        // utils.NewJWT() 返回 *JWT
	claims, err := jwtService.ParseToken(authMsg.Token) // ParseToken 需要 string, 返回 *request.CustomClaims, error

	if err != nil {
		global.GVA_LOG.Warn("Hub: Token validation failed", zap.String("clientID", client.GetID()), zap.Error(err))
		errMsg := "Authentication failed"
		if errors.Is(err, utils.TokenExpired) {
			errMsg = "Token has expired"
		} else if errors.Is(err, utils.TokenNotValidYet) {
			errMsg = "Token not valid yet"
		} else if errors.Is(err, utils.TokenMalformed) {
			errMsg = "Token is malformed"
		} else if errors.Is(err, utils.TokenSignatureInvalid) {
			errMsg = "Token signature is invalid"
		} else if errors.Is(err, utils.TokenInvalid) {
			errMsg = "Token is invalid"
		}
		sendErrorMessage(client, protocol.ErrorCodeAuthFailed, errMsg)

		// Audit Log for authentication failure (already exists)
		AuthEvents.WithLabelValues("failure", errMsg).Inc()

		// 考虑是否需要关闭连接 client.conn.Close()
		return
	}

	// 检查 claims 和 BaseClaims 是否为 nil，以及 ID 是否有效
	if claims == nil || claims.BaseClaims.ID == 0 {
		global.GVA_LOG.Error("Hub: UserID is missing in token claims", zap.String("clientID", client.GetID()))
		sendErrorMessage(client, protocol.ErrorCodeAuthFailed, "UserID missing or invalid in token")
		// Audit Log for authentication failure (missing UserID) (already exists)
		AuthEvents.WithLabelValues("failure", "UserID missing or invalid in token").Inc()
		return
	}

	h.providerMutex.Lock() // 加锁保护 client 字段的修改
	client.Authenticated = true
	client.UserID = strconv.FormatUint(uint64(claims.BaseClaims.ID), 10)
	// 可以考虑在这里也存储 claims.BaseClaims.Username 或 claims.BaseClaims.NickName 到 client 结构体的新字段中，如果需要
	// client.Username = claims.BaseClaims.Username
	global.GVA_LOG.Info("Hub: Client authenticated successfully", zap.String("clientID", client.GetID()), zap.String("userID", client.UserID))

	// Audit Log for authentication success (already exists)
	AuthEvents.WithLabelValues("success", "").Inc() // Reason is empty for success

	h.providerMutex.Unlock()

	response := protocol.ServerAuthResponseMessage{
		Type:    protocol.MessageTypeServerAuthResponse,
		Success: true,
		UserID:  strconv.FormatUint(uint64(claims.BaseClaims.ID), 10),
	}
	if err := sendProtoMessage(client, response); err != nil {
		global.GVA_LOG.Error("Hub: Failed to send auth response to client", zap.Error(err), zap.String("clientID", client.GetID()))
	}
}

// handleDeclareRole 处理客户端声明角色和在线状态的请求
func (h *Hub) handleDeclareRole(client *Client, messageBytes []byte) {
	var declareMsg protocol.DeclareRoleMessage
	if err := json.Unmarshal(messageBytes, &declareMsg); err != nil {
		global.GVA_LOG.Error("Hub: Error unmarshalling declare role message", zap.Error(err), zap.String("clientID", client.GetID()))
		sendErrorMessage(client, protocol.ErrorCodeBadRequest, "Invalid declare role message format")
		return
	}

	global.GVA_LOG.Info("Hub: Received DeclareRoleMessage",
		zap.String("clientID", client.GetID()),
		zap.String("role", string(declareMsg.Role)),
		zap.Bool("online", declareMsg.Online),
		zap.String("providerName", declareMsg.ProviderName),
	)

	if declareMsg.Role != protocol.RoleProvider && declareMsg.Role != protocol.RoleReceiver && declareMsg.Role != protocol.RoleNone {
		global.GVA_LOG.Warn("Hub: Invalid role in DeclareRoleMessage",
			zap.String("clientID", client.GetID()),
			zap.String("roleReceived", string(declareMsg.Role)),
		)
		sendErrorMessage(client, protocol.ErrorCodeBadRequest, "Invalid role specified")
		return
	}

	var (
		shouldNotifyProviderList bool
		userIDToNotify           string
		finalDisplayName         string
		logMessage               string
		logFields                []zap.Field
	)

	// --- Critical section for updating shared Hub state and client state ---
	h.providerMutex.Lock()

	oldRole := client.CurrentRole
	oldIsOnline := client.IsOnline
	// oldDisplayName := client.DisplayName // Keep a copy if needed for complex logic not present now

	client.CurrentRole = declareMsg.Role
	client.IsOnline = declareMsg.Online
	finalDisplayName = client.DisplayName // Default to current if not changed by provider logic

	providerStatusChanged := false
	roleChangedFromProvider := false
	roleChangedToProviderOffline := false

	if client.CurrentRole == protocol.RoleProvider {
		if declareMsg.ProviderName != "" {
			client.DisplayName = declareMsg.ProviderName
			finalDisplayName = declareMsg.ProviderName
		} else if client.DisplayName == "" { // Only set default if current is empty and no new name provided
			id := client.GetID()
			idSuffix := id
			if len(id) > 8 {
				idSuffix = id[:8]
			}
			defaultName := "Provider " + idSuffix
			client.DisplayName = defaultName
			finalDisplayName = defaultName
			global.GVA_LOG.Info("Hub: Provider declared without name, using default",
				zap.String("clientID", client.GetID()),
				zap.String("defaultName", client.DisplayName),
			)
		} // If ProviderName in msg is empty but client.DisplayName already exists, keep existing client.DisplayName

		if client.IsOnline {
			if _, exists := h.cardProviders[client.GetID()]; !exists || oldRole != protocol.RoleProvider || !oldIsOnline {
				h.cardProviders[client.GetID()] = client
				providerStatusChanged = true
				logMessage = "Hub: Provider declared and online, added to available list."
				logFields = []zap.Field{zap.String("providerID", client.GetID()), zap.String("displayName", client.DisplayName)}
			} else {
				// Provider was already online and in list, possibly just a DisplayName update or re-affirmation
				logMessage = "Hub: Online provider re-declared role/status (or updated display name)."
				logFields = []zap.Field{zap.String("providerID", client.GetID()), zap.String("newDisplayName", client.DisplayName)}
				if oldIsOnline && oldRole == protocol.RoleProvider && client.DisplayName != finalDisplayName { // Check if only display name changed for an existing online provider
					// Notification might not be strictly needed for only display name change,
					// but current logic below triggers if providerStatusChanged is true.
					// For now, we assume any change to an online provider or bringing one online needs notification.
				}
			}
		} else { // Provider is declaring as offline
			if _, exists := h.cardProviders[client.GetID()]; exists {
				delete(h.cardProviders, client.GetID())
				providerStatusChanged = true        // Status changed from online (in map) to offline
				roleChangedToProviderOffline = true // Specifically for notification logic
				logMessage = "Hub: Provider declared offline, removed from available list."
				logFields = []zap.Field{zap.String("providerID", client.GetID())}
			} else {
				logMessage = "Hub: Provider declared offline (was not in available list)."
				logFields = []zap.Field{zap.String("providerID", client.GetID())}
			}
		}
	} else { // Role is Receiver or None
		finalDisplayName = ""         // Receivers/None don't have display names in this context
		if client.DisplayName != "" { // Clear display name if it was set (e.g. from previous provider role)
			client.DisplayName = ""
		}
		if oldRole == protocol.RoleProvider { // Was a provider before
			if _, exists := h.cardProviders[client.GetID()]; exists {
				delete(h.cardProviders, client.GetID())
				providerStatusChanged = true   // Status changed as provider removed
				roleChangedFromProvider = true // Specifically for notification logic
				logMessage = "Hub: Client changed role from provider, removed from available list."
				logFields = []zap.Field{zap.String("clientID", client.GetID()), zap.String("newRole", string(client.CurrentRole))}
			} else {
				logMessage = "Hub: Client changed role from provider (was not in available list)."
				logFields = []zap.Field{zap.String("clientID", client.GetID()), zap.String("newRole", string(client.CurrentRole))}
			}
		} else {
			logMessage = "Hub: Client declared role."
			logFields = []zap.Field{zap.String("clientID", client.GetID()), zap.String("role", string(client.CurrentRole))}
		}
	}

	if logMessage != "" {
		global.GVA_LOG.Info(logMessage, append(logFields, zap.String("userID", client.GetUserID()))...)
	}

	// Determine if notification is needed based on status changes relevant to provider list
	if providerStatusChanged || roleChangedFromProvider || roleChangedToProviderOffline {
		shouldNotifyProviderList = true
		userIDToNotify = client.GetUserID() // UserID of the client whose provider status might have changed
	}

	// Capture necessary state for response before unlocking
	responseRole := string(client.CurrentRole)
	responseIsOnline := client.IsOnline

	h.providerMutex.Unlock()
	// --- End of critical section ---

	// Send response message outside of the lock
	responseMessage := protocol.RoleDeclaredResponseMessage{
		Type:    protocol.MessageTypeRoleDeclaredResponse,
		Success: true,
		Role:    protocol.RoleType(responseRole),
		Online:  responseIsOnline,
		// SessionID: responseClientID, // Removed: Field is commented out in struct definition
	}
	if err := sendProtoMessage(client, responseMessage); err != nil {
		global.GVA_LOG.Error("Hub: Failed to send role declared response",
			zap.String("clientID", client.GetID()),
			zap.Error(err),
		)
		// Do not return, still attempt to notify if needed as state change might have occurred
	}

	// Spawn notification goroutine outside of the lock
	if shouldNotifyProviderList && userIDToNotify != "" {
		global.GVA_LOG.Debug("Hub (handleDeclareRole): Notifying provider list subscribers due to role/status change.", zap.String("userID", userIDToNotify), zap.String("triggeringClientID", client.GetID()))
		go h.notifyProviderListSubscribers(userIDToNotify)
	}
}

// handleClientDisconnect 处理客户端断开连接时与会话相关的逻辑
func (h *Hub) handleClientDisconnect(client *Client) {
	clientID := client.GetID() // Cache client ID for logging after client might be nilled or its state changed
	clientUserID := client.GetUserID()

	global.GVA_LOG.Info("开始处理客户端断开连接", zap.String("clientID", clientID), zap.String("userID", clientUserID))

	// 检查客户端是否在会话中，如果是，则终止会话
	// 注意：terminateSessionByID 内部有自己的锁，并且会修改 client.SessionID
	// 为避免死锁或竞争，在调用它之前，不要持有 h.providerMutex
	currentSessionID := client.GetSessionID() // Get SessionID before it's cleared by terminateSessionByID
	if currentSessionID != "" {
		global.GVA_LOG.Info("客户端断开连接时仍在会话中，准备终止会话",
			zap.String("clientID", clientID),
			zap.String("sessionID", currentSessionID),
		)
		// 异步终止会话，以避免阻塞此处理流程，并确保锁的正确管理
		// terminateSessionByID 将处理通知和清理
		go h.terminateSessionByID(currentSessionID, "客户端断开连接", clientID, clientUserID)
		// 等待一小段时间让 terminateSessionByID 中的清理逻辑（如SessionID清空）有机会执行
		// 这是一个临时的辅助手段，理想情况下应有更健壮的同步机制。
		// 但考虑到 terminateSessionByID 是异步的，直接检查 client.SessionID 可能不是最新的。
		// 主要目的是让后续的 provider 状态检查能基于会话已尝试终止的假设。
		// time.Sleep(50 * time.Millisecond) // 暂时移除，看是否是导致测试不稳定的原因
	}

	wasProvider := false
	userIDOfDisconnectedProvider := client.GetUserID() // Ensure we have this before the client's state might change further

	h.providerMutex.Lock()
	// 从 cardProviders 中移除 (如果存在)
	if _, ok := h.cardProviders[clientID]; ok {
		delete(h.cardProviders, clientID)
		wasProvider = true
		global.GVA_LOG.Debug("Provider explicitly deleted from cardProviders map in handleClientDisconnect", zap.String("clientID", clientID))
		global.GVA_LOG.Info("发卡方客户端断开连接，从可用列表中移除 (in lock)", zap.String("clientID", clientID), zap.String("userID", userIDOfDisconnectedProvider))
	}

	// 从所有 providerListSubscribers 订阅列表中移除此断开连接的客户端
	// 这确保如果这个客户端本身是一个订阅者，它会被清理
	for forUserID, subscribersMap := range h.providerListSubscribers {
		if _, subscribed := subscribersMap[client]; subscribed {
			delete(subscribersMap, client)
			global.GVA_LOG.Debug("已断开的客户端从其订阅的提供者列表(for UserID)中移除",
				zap.String("disconnectedClientID", clientID),
				zap.String("subscribedToProviderListForUserID", forUserID),
			)
			if len(subscribersMap) == 0 {
				delete(h.providerListSubscribers, forUserID)
				global.GVA_LOG.Debug("特定UserID的提供者列表订阅者集合已空，移除该UserID的订阅者映射", zap.String("userID", forUserID))
			}
		}
	}
	h.providerMutex.Unlock()

	// 如果断开的是一个 Provider，需要通知其 UserID 的订阅者列表已更改
	if wasProvider {
		global.GVA_LOG.Info("发卡方客户端已断开，准备通知其 UserID 的订阅者有关列表状态变更",
			zap.String("disconnectedProviderClientID", clientID),
			zap.String("userIDOfDisconnectedProvider", userIDOfDisconnectedProvider),
		)
		go h.notifyProviderListSubscribers(userIDOfDisconnectedProvider) // Corrected method name and called in a goroutine
	}

	global.GVA_LOG.Info("客户端断开连接处理完毕", zap.String("clientID", clientID), zap.String("userID", clientUserID))
}

// sendProtoMessage 是一个辅助函数，用于将 protocol 包中定义的结构体序列化为 JSON 并发送给客户端
// 它现在接收 ClientInfoProvider 接口，而不是具体的 *Client 类型。
// 返回一个 error 以指示发送是否成功。
func sendProtoMessage(client session.ClientInfoProvider, message interface{}) error {
	if client == nil {
		return errors.New("client is nil")
	}

	// Log the message content before serialization, especially if it's an ErrorMessage
	if errMsg, ok := message.(protocol.ErrorMessage); ok {
		global.GVA_LOG.Debug("sendProtoMessage: Preparing to serialize ErrorMessage",
			zap.String("targetClientID", client.GetID()),
			zap.Int("messageCode", errMsg.Code),
			zap.String("messageType", string(errMsg.Type)),
			zap.String("messageContent", errMsg.Message),
		)
	} else if genericMsg, ok := message.(protocol.GenericMessage); ok { // Log other common types if needed
		global.GVA_LOG.Debug("sendProtoMessage: Preparing to serialize GenericMessage",
			zap.String("targetClientID", client.GetID()),
			zap.String("messageType", string(genericMsg.Type)),
		)
	} else {
		// For other types, just log a generic message or the type
		global.GVA_LOG.Debug("sendProtoMessage: Preparing to serialize message",
			zap.String("targetClientID", client.GetID()),
			zap.String("messageGoType", fmt.Sprintf("%T", message)),
		)
	}

	bytes, err := json.Marshal(message)
	if err != nil {
		global.GVA_LOG.Error("序列化消息失败", zap.Error(err), zap.String("targetClientID", client.GetID()))
		return err
	}

	// Log the serialized bytes
	global.GVA_LOG.Debug("sendProtoMessage: Serialized message bytes",
		zap.String("targetClientID", client.GetID()),
		zap.String("jsonBytes", string(bytes)),
	)

	// 使用接口提供的 Send 方法
	if err := client.Send(bytes); err != nil {
		global.GVA_LOG.Warn("通过接口发送消息给客户端失败", zap.Error(err), zap.String("targetClientID", client.GetID()))
		return err
	}
	return nil
}

// sendErrorMessage 是一个辅助函数，用于向客户端发送标准错误消息
// 它调用 sendProtoMessage，如果发送错误消息本身失败，则记录日志。
func sendErrorMessage(client session.ClientInfoProvider, code int, message string) {
	errMsg := protocol.ErrorMessage{
		Type:    protocol.MessageTypeError,
		Code:    code,
		Message: message,
		// SessionID can be added if available/needed from client.GetSessionID()
	}

	// 新增日志：在调用 sendProtoMessage 之前记录 errMsg 结构体的内容
	clientID := "unknown"
	if c, ok := client.(interface{ GetID() string }); ok {
		clientID = c.GetID()
	}
	global.GVA_LOG.Info("sendErrorMessage: Constructed ErrorMessage before calling sendProtoMessage",
		zap.String("targetClientID", clientID),
		zap.Int("structCodeField", errMsg.Code), // 明确记录结构体中的 Code 字段
		zap.String("structMessageField", errMsg.Message),
		zap.String("structTypeField", string(errMsg.Type)),
		zap.Any("fullErrMsgStruct", errMsg), // 记录完整的结构体以便检查
	)

	// Audit Log for sending an error message to the client
	details := global.ErrorDetails{
		ErrorCode:    strconv.Itoa(code),
		ErrorMessage: message,
		Component:    "nfc_relay_hub.sendErrorMessage",
	}

	HubErrors.WithLabelValues(strconv.Itoa(code), "nfc_relay_hub.sendErrorMessage").Inc()

	logFields := []zap.Field{}
	if client != nil { // Add client related fields only if client is not nil
		logFields = append(logFields, zap.String("client_id", client.GetID()))
		if client.GetUserID() != "" {
			logFields = append(logFields, zap.String("user_id", client.GetUserID()))
		}
		if client.GetSessionID() != "" {
			logFields = append(logFields, zap.String("session_id", client.GetSessionID()))
		}
		if c, ok := client.(*Client); ok { // Attempt to cast to *Client to get more specific details
			if c.conn != nil {
				logFields = append(logFields, zap.String("client_ip", c.conn.RemoteAddr().String()))
			}
			// If c.ID is different from client.GetID() or provides more specificty, ensure "client_id" is prioritized.
			// Current GetID() for *Client returns c.ID, so it's covered.
		}
	}

	global.LogAuditEvent("client_error_notification_sent", details, logFields...)

	if err := sendProtoMessage(client, errMsg); err != nil {
		// Log the error if sending the error message itself fails.
		global.GVA_LOG.Warn("发送标准错误消息本身失败",
			zap.Error(err),
			zap.String("targetClientID", client.GetID()), // Use interface GetID here
			zap.String("originalErrorMessage", message),
		)
	}
}

// handleListCardProviders 处理收卡方请求可用发卡方列表的请求
func (h *Hub) handleListCardProviders(requestingClient *Client, messageBytes []byte) {
	// 确保请求者是收卡方 (或者根据业务逻辑允许其他角色查看)
	if requestingClient.CurrentRole != "receiver" && requestingClient.CurrentRole != "" { // 允许未指定角色的客户端查看
		// 如果严格要求必须是 receiver 才能查看，可以取消注释下面的代码
		global.GVA_LOG.Warn("非收卡方客户端尝试获取发卡方列表",
			zap.String("clientID", requestingClient.GetID()),
			zap.String("currentRole", string(requestingClient.CurrentRole)),
		)
		sendErrorMessage(requestingClient, protocol.ErrorCodePermissionDenied, "只有收卡方角色才能获取发卡方列表")
		return
	}

	var listMsg protocol.ListCardProvidersMessage
	if err := json.Unmarshal(messageBytes, &listMsg); err != nil {
		global.GVA_LOG.Error("Hub 处理列表请求：反序列化 ListCardProvidersMessage 失败", zap.Error(err), zap.String("clientID", requestingClient.GetID()))
		sendErrorMessage(requestingClient, protocol.ErrorCodeBadRequest, "无效的列表请求消息格式")
		return
	}

	global.GVA_LOG.Info("处理获取可用发卡方列表请求",
		zap.String("requestingClientID", requestingClient.GetID()),
		zap.String("requestingUserID", requestingClient.UserID),
	)

	h.providerMutex.Lock() // Lock for subscribers and RLock for cardProviders/sessions in the same scope
	if requestingClient.Authenticated && requestingClient.UserID != "" {
		if _, ok := h.providerListSubscribers[requestingClient.UserID]; !ok {
			h.providerListSubscribers[requestingClient.UserID] = make(map[*Client]bool)
		}
		h.providerListSubscribers[requestingClient.UserID][requestingClient] = true
		global.GVA_LOG.Debug("客户端已订阅其用户ID的发卡方列表更新", zap.String("clientID", requestingClient.GetID()), zap.String("userID", requestingClient.UserID))
	}

	var availableProviders []protocol.CardProviderInfo
	for _, providerClient := range h.cardProviders {
		if providerClient.GetUserID() == requestingClient.UserID {
			isBusy := false
			if providerClient.GetSessionID() != "" {
				_, sessionExists := h.sessions[providerClient.GetSessionID()]
				if sessionExists {
					isBusy = true
				}
			}
			// 检查 providerClient 是否实现了 ClientInfoProvider 接口的所有方法，特别是 DisplayName
			// 我们的 Client 结构体现在有了 DisplayName 字段
			providerDisplayName := providerClient.(*Client).DisplayName
			if providerDisplayName == "" { // 后备名称
				providerDisplayName = "Provider " + providerClient.GetID()[:6]
			}

			info := protocol.CardProviderInfo{
				ProviderID:   providerClient.GetID(),
				ProviderName: providerDisplayName,
				UserID:       providerClient.GetUserID(),
				IsBusy:       isBusy,
			}
			availableProviders = append(availableProviders, info)
		}
	}
	h.providerMutex.Unlock()

	response := protocol.CardProvidersListMessage{
		Type:      protocol.MessageTypeCardProvidersList,
		Providers: availableProviders,
	}
	if err := sendProtoMessage(requestingClient, response); err != nil {
		global.GVA_LOG.Error("向请求列表的客户端发送列表失败", zap.Error(err), zap.String("clientID", requestingClient.GetID()))
	}

	global.GVA_LOG.Info("已向收卡方发送可用发卡方列表",
		zap.String("requestingClientID", requestingClient.GetID()),
		zap.Int("count", len(availableProviders)),
	)
}

// handleSelectCardProvider 处理收卡方选择发卡方并请求连接的逻辑
func (h *Hub) handleSelectCardProvider(requestingClient *Client, messageBytes []byte) {
	// 验证请求者是否为 Receiver (这个检查也可以放在 handleIncomingMessage 的 switch case 中，但在这里再次确认无害)
	if requestingClient.CurrentRole != protocol.RoleReceiver {
		global.GVA_LOG.Warn("Hub: (handleSelectCardProvider) 非 receiver 客户端尝试选择 provider",
			zap.String("clientID", requestingClient.GetID()),
			zap.String("currentRole", string(requestingClient.CurrentRole)),
		)
		sendErrorMessage(requestingClient, protocol.ErrorCodePermissionDenied, "操作失败：只有 receiver 角色的客户端才能选择发卡方。")
		return
	}

	var selectMsg protocol.SelectCardProviderMessage
	if err := json.Unmarshal(messageBytes, &selectMsg); err != nil {
		global.GVA_LOG.Error("Hub 处理选择发卡方：反序列化 SelectCardProviderMessage 失败", zap.Error(err), zap.String("clientID", requestingClient.GetID()))
		sendErrorMessage(requestingClient, protocol.ErrorCodeBadRequest, "无效的选择发卡方消息格式")
		return
	}

	providerIDToSelect := selectMsg.ProviderID
	if providerIDToSelect == "" {
		global.GVA_LOG.Warn("Hub: SelectCardProvider 请求中 ProviderID 为空", zap.String("clientID", requestingClient.GetID()))
		sendErrorMessage(requestingClient, protocol.ErrorCodeBadRequest, "选择发卡方失败：必须提供有效的 ProviderID。")
		return
	}

	h.providerMutex.RLock() // RLock for reading cardProviders and sessions
	targetProviderEntry, providerExists := h.cardProviders[providerIDToSelect]
	if !providerExists {
		h.providerMutex.RUnlock()
		global.GVA_LOG.Warn("Hub: 目标发卡方不存在或未上线", zap.String("targetProviderID", providerIDToSelect), zap.String("requestingClientID", requestingClient.GetID()))
		sendErrorMessage(requestingClient, protocol.ErrorCodeProviderNotFound, "选择发卡方失败：目标发卡方不存在或当前未提供服务。")
		// 尝试通知发卡方列表的订阅者，如果此 provider 真的消失了 (可能因为之前的逻辑没及时通知)
		// go h.notifyProviderListSubscribers(requestingClient.GetUserID()) // 谨慎使用，避免不必要的通知风暴
		return
	}

	// 尝试将会话的客户端转换为具体的 *Client 类型，以便访问更多信息
	targetProviderConcrete, ok := targetProviderEntry.(*Client)
	if !ok || targetProviderConcrete == nil { // 添加 targetProviderConcrete == nil 检查
		global.GVA_LOG.Error("Hub: (handleSelectCardProvider) cardProviders 中的条目不是 *Client 类型或为nil",
			zap.String("requestingClientID", requestingClient.GetID()),
			zap.String("targetProviderID", selectMsg.ProviderID), // Use selectMsg.ProviderID as targetProviderConcrete might be nil
		)
		sendErrorMessage(requestingClient, protocol.ErrorCodeInternalError, "选择发卡方失败：服务器内部错误。")
		return
	}

	// 检查 UserID 是否匹配
	if requestingClient.GetUserID() != targetProviderConcrete.GetUserID() {
		h.providerMutex.RUnlock()
		global.GVA_LOG.Warn("Hub: 请求者和提供者 UserID 不匹配",
			zap.String("requestingUserID", requestingClient.GetUserID()),
			zap.String("providerUserID", targetProviderConcrete.GetUserID()),
			zap.String("requestingClientID", requestingClient.GetID()),
			zap.String("providerClientID", targetProviderConcrete.GetID()),
		)
		sendErrorMessage(requestingClient, protocol.ErrorCodePermissionDenied, "选择发卡方失败：不能选择其他账户下的发卡方。")
		return
	}

	// 检查2: 不能选择自己作为发卡方
	if requestingClient.GetID() == targetProviderConcrete.GetID() {
		h.providerMutex.RUnlock()
		global.GVA_LOG.Warn("Hub: 客户端尝试选择自己作为发卡方", zap.String("clientID", requestingClient.GetID()))
		sendErrorMessage(requestingClient, protocol.ErrorCodeSelectSelf, "选择发卡方失败：不能选择自己。")
		return
	}

	// 检查3: 请求者 (Receiver) 当前是否已在会话中
	if requestingClient.GetSessionID() != "" {
		if _, sessionExists := h.sessions[requestingClient.GetSessionID()]; sessionExists {
			h.providerMutex.RUnlock()
			global.GVA_LOG.Warn("Hub: 请求的发卡方已在会话中", zap.String("clientID", requestingClient.GetID()), zap.String("sessionID", requestingClient.GetSessionID()))
			sendErrorMessage(requestingClient, protocol.ErrorCodeReceiverBusy, "选择发卡方失败：您当前已在会话中。")
			return
		}
	}

	// 检查4: 目标提供者 (Provider) 当前是否已在会话中 (IsBusy)
	if targetProviderConcrete.GetSessionID() != "" { // 使用 targetProviderConcrete
		h.providerMutex.RUnlock()
		global.GVA_LOG.Warn("Hub: 目标发卡方已在会话中",
			zap.String("targetProviderID", targetProviderConcrete.GetID()), // 使用 targetProviderConcrete
			zap.String("sessionID", targetProviderConcrete.GetSessionID()), // 使用 targetProviderConcrete
		)
		sendErrorMessage(requestingClient, protocol.ErrorCodeProviderBusy, "选择发卡方失败：目标发卡方当前正忙。")
		return
	}
	h.providerMutex.RUnlock() // 读取完成，释放读锁

	// --- 到此所有只读检查完成，如果需要修改共享数据，则需要重新获取写锁 ---
	h.providerMutex.Lock() // 获取写锁以创建会话和更新客户端状态
	defer h.providerMutex.Unlock()

	// 双重检查：再次确认 provider 仍然是同一个并且在线，并且双方都空闲
	// （在释放读锁和获取写锁之间，状态可能已改变）
	doubleCheckProviderEntry, stillExists := h.cardProviders[targetProviderConcrete.GetID()]
	if !stillExists || doubleCheckProviderEntry != targetProviderEntry { // 确保还是原来的那个 entry
		global.GVA_LOG.Warn("Hub: (handleSelectCardProvider) 双重检查失败 - Provider 实例已改变或不再存在",
			zap.String("requestingClientID", requestingClient.GetID()),
			zap.String("targetProviderID", targetProviderConcrete.GetID()),
		)
		sendErrorMessage(requestingClient, protocol.ErrorCodeSessionConflict, "选择发卡方失败：发卡方状态已改变，请重试。") // 更具体的错误消息
		return
	}

	doubleCheckProviderConcrete, ok := doubleCheckProviderEntry.(*Client)
	if !ok || doubleCheckProviderConcrete == nil { // 再次检查类型和nil
		global.GVA_LOG.Error("Hub: (handleSelectCardProvider) 双重检查失败 - Provider 类型断言失败或为nil",
			zap.String("requestingClientID", requestingClient.GetID()),
			zap.String("targetProviderID", targetProviderConcrete.GetID()), // If nil, this was caught by the previous check on targetProviderConcrete
		)
		sendErrorMessage(requestingClient, protocol.ErrorCodeInternalError, "选择发卡方失败：服务器内部错误。")
		return
	}

	if !doubleCheckProviderConcrete.IsOnline || // 确保仍然在线
		doubleCheckProviderConcrete.GetSessionID() != "" || // 确保 Provider 仍然空闲
		requestingClient.GetSessionID() != "" { // 确保 Receiver 仍然空闲
		global.GVA_LOG.Warn("Hub: (handleSelectCardProvider) 双重检查失败 - 一方或双方状态已改变（不再空闲/在线）",
			zap.String("requestingClientID", requestingClient.GetID()),
			zap.String("targetProviderID", doubleCheckProviderConcrete.GetID()),
			zap.Bool("providerOnline", doubleCheckProviderConcrete.IsOnline),
			zap.String("providerSessionID", doubleCheckProviderConcrete.GetSessionID()),
			zap.String("receiverSessionID", requestingClient.GetSessionID()),
		)
		sendErrorMessage(requestingClient, protocol.ErrorCodeSessionConflict, "选择发卡方失败：一方或双方状态已改变（不再空闲），请重试。")
		return
	}

	// 创建新会话
	formalSessionID := uuid.NewString()
	newSession := session.NewSession(formalSessionID)
	newSession.SetClient(requestingClient, string(protocol.RoleReceiver))
	newSession.SetClient(doubleCheckProviderConcrete, string(protocol.RoleProvider)) // 传递 concrete type
	h.sessions[formalSessionID] = newSession

	// 更新客户端的 SessionID
	requestingClient.SessionID = formalSessionID
	doubleCheckProviderConcrete.SessionID = formalSessionID

	global.GVA_LOG.Info("Hub: Receiver selected provider, new session created and clients paired.",
		zap.String("sessionID", formalSessionID),
		zap.String("receiverClientID", requestingClient.GetID()),
		zap.String("providerClientID", doubleCheckProviderConcrete.GetID()),
	)

	// Audit Log for session establishment
	global.LogAuditEvent(
		"session_established",
		global.SessionDetails{
			InitiatorRole: string(requestingClient.CurrentRole),
			ResponderRole: string(doubleCheckProviderConcrete.CurrentRole),
		},
		zap.String("session_id", formalSessionID),
		zap.String("client_id_initiator", requestingClient.GetID()),
		zap.String("user_id_initiator", requestingClient.UserID),
		zap.String("client_id_responder", doubleCheckProviderConcrete.GetID()),
		zap.String("user_id_responder", doubleCheckProviderConcrete.GetUserID()),
		zap.String("source_ip_initiator", requestingClient.conn.RemoteAddr().String()),
		zap.String("source_ip_responder", doubleCheckProviderConcrete.conn.RemoteAddr().String()),
	)

	ActiveSessions.Inc() // Increment active sessions metric

	// 通知双方会话已建立
	// 向 receiver 发送
	errReceiver := sendProtoMessage(requestingClient, protocol.SessionEstablishedMessage{
		Type:      protocol.MessageTypeSessionEstablished,
		SessionID: formalSessionID,
		PeerID:    doubleCheckProviderConcrete.GetID(),
		PeerRole:  protocol.RoleProvider,
	})
	if errReceiver != nil {
		global.GVA_LOG.Error("发送会话建立成功的消息给收卡方失败", zap.Error(errReceiver), zap.String("receiverClientID", requestingClient.GetID()))
		// TODO: 考虑回滚会话创建或通知另一方
	}

	// 向 provider 发送
	errProvider := sendProtoMessage(doubleCheckProviderConcrete, protocol.SessionEstablishedMessage{ // 使用 concrete type
		Type:      protocol.MessageTypeSessionEstablished,
		SessionID: formalSessionID,
		PeerID:    requestingClient.GetID(),
		PeerRole:  protocol.RoleReceiver,
	})
	if errProvider != nil {
		global.GVA_LOG.Error("发送会话建立成功的消息给发卡方失败", zap.Error(errProvider), zap.String("providerClientID", doubleCheckProviderConcrete.GetID()))
		// TODO: 考虑回滚会话创建或通知另一方
	}

	// 提供者进入会话后状态变为繁忙，通知其 UserID 下的订阅者列表更新
	go h.notifyProviderListSubscribers(doubleCheckProviderConcrete.GetUserID())

}

// handleAPDUExchange 处理来自任一端的APDU消息并转发
func (h *Hub) handleAPDUExchange(sourceClient *Client, messageBytes []byte, direction string) {
	if sourceClient.SessionID == "" {
		global.GVA_LOG.Warn("收到APDU消息，但客户端不在任何会话中",
			zap.String("clientID", sourceClient.GetID()),
			zap.String("direction", direction),
		)
		sendErrorMessage(sourceClient, protocol.ErrorCodeBadRequest, "您当前不在任何APDU中继会话中，无法发送APDU")
		return
	}

	h.providerMutex.RLock() // RLock 因为我们主要读取 sessions map
	activeSession, sessionExists := h.sessions[sourceClient.SessionID]
	h.providerMutex.RUnlock()

	if !sessionExists {
		global.GVA_LOG.Error("收到APDU消息，但客户端关联的会话ID无效或已终止",
			zap.String("clientID", sourceClient.GetID()),
			zap.String("sessionID", sourceClient.SessionID),
		)
		sourceClient.SessionID = "" // 清理无效的 SessionID
		sendErrorMessage(sourceClient, protocol.ErrorCodeSessionConflict, "您当前的APDU会话已失效，请重新建立连接")
		return
	}

	// 更新会话活动时间
	activeSession.UpdateActivityTime()

	peerClient := activeSession.GetPeer(sourceClient)
	if peerClient == nil {
		global.GVA_LOG.Warn("APDU交换：未找到会话中的对端客户端，可能已掉线",
			zap.String("clientID", sourceClient.GetID()),
			zap.String("sessionID", sourceClient.SessionID),
		)
		sendErrorMessage(sourceClient, protocol.ErrorCodeProviderNotFound, "APDU发送失败：未能找到您的通信对端")
		// 可以在这里考虑将会话标记为终止或清理
		return
	}

	var apduData string
	// var targetMessageType protocol.MessageType // Removed: declared and not used

	if direction == "upstream" { // 来自收卡端 (POS/Receiver)，发往传卡端 (Card/Provider)
		var msg protocol.APDUUpstreamMessage
		if err := json.Unmarshal(messageBytes, &msg); err != nil {
			global.GVA_LOG.Error("APDU交换：反序列化APDUUpstreamMessage失败", zap.Error(err), zap.String("clientID", sourceClient.GetID()))
			sendErrorMessage(sourceClient, protocol.ErrorCodeBadRequest, "无效的APDU消息格式 (upstream)")
			return
		}
		apduData = msg.APDU
		// targetMessageType = protocol.MessageTypeAPDUToCard //发给卡端的消息类型 (Handled by direct send now)
		global.GVA_LOG.Info("APDU upstream", zap.String("from", sourceClient.GetID()), zap.String("to", peerClient.GetID()), zap.String("sessionID", activeSession.SessionID), zap.Int("apdu_len", len(apduData)))

	} else if direction == "downstream" { // 来自传卡端 (Card/Provider)，发往收卡端 (POS/Receiver)
		var msg protocol.APDUDownstreamMessage
		if err := json.Unmarshal(messageBytes, &msg); err != nil {
			global.GVA_LOG.Error("APDU交换：反序列化APDUDownstreamMessage失败", zap.Error(err), zap.String("clientID", sourceClient.GetID()))
			sendErrorMessage(sourceClient, protocol.ErrorCodeBadRequest, "无效的APDU消息格式 (downstream)")
			return
		}
		apduData = msg.APDU
		// targetMessageType = protocol.MessageTypeAPDUFromCard //发给POS的消息类型 (来自卡端的响应) (Handled by direct send now)
		global.GVA_LOG.Info("APDU downstream", zap.String("from", sourceClient.GetID()), zap.String("to", peerClient.GetID()), zap.String("sessionID", activeSession.SessionID), zap.Int("apdu_len", len(apduData)))
	} else {
		global.GVA_LOG.Error("APDU交换：未知的APDU方向", zap.String("direction", direction), zap.String("clientID", sourceClient.GetID()))
		return // 不应该发生
	}

	// Audit Log for APDU relay attempt
	global.LogAuditEvent(
		"apdu_relayed_attempt", // Or "apdu_exchange_started" if we want start/end events
		global.APDUDetails{
			Direction: direction,
			Length:    len(apduData), // Or len(messageBytes) if it's just the APDU
		},
		zap.String("session_id", sourceClient.SessionID),
		zap.String("source_client_id", sourceClient.GetID()),
		zap.String("target_client_id", peerClient.GetID()),
		zap.String("user_id_source", sourceClient.UserID),
	)

	// 将 APDU 消息转发给对端客户端
	// The original messageBytes (which is the full protocol message) should be forwarded,
	// not just apduData. The peer expects a full protocol message.
	if err := peerClient.Send(messageBytes); err != nil {
		global.GVA_LOG.Error("Hub: 转发 APDU 消息失败", zap.Error(err), zap.String("targetClientID", peerClient.GetID()))
		sendErrorMessage(sourceClient, protocol.ErrorCodeInternalError, "Failed to forward APDU to peer: "+err.Error()) // Corrected ErrorCodeMessageSendFailed
		ApduRelayErrors.WithLabelValues(direction, sourceClient.SessionID).Inc()
		// Audit log for APDU relay failure is already here

		// Terminate the session as APDU exchange is critical
		// We need RUnlock before calling terminateSessionByID which will Lock
		// However, we are already outside the RLock for session reading in the current structure.
		// Let's verify the lock scope. The RLock was for `h.sessions[sourceClient.SessionID]`
		// which is done before this point.
		// So, we can call terminateSessionByID directly.
		global.LogAuditEvent(
			"apdu_relayed_failure",
			global.ErrorDetails{
				ErrorCode:    strconv.Itoa(protocol.ErrorCodeInternalError), // Corrected ErrorCodeMessageSendFailed
				ErrorMessage: "Failed to forward APDU to peer, terminating session: " + err.Error(),
				Component:    "nfc_relay_hub.handleAPDUExchange",
			},
			zap.String("session_id", sourceClient.SessionID),
			zap.String("source_client_id", sourceClient.GetID()),
			zap.String("target_client_id", peerClient.GetID()),
			zap.String("user_id_source", sourceClient.UserID),
			zap.String("direction", direction),
		)

		// Terminate the session. The acting client is the sourceClient, as it initiated the APDU that failed to be relayed.
		h.terminateSessionByID(sourceClient.SessionID, "APDU转发失败导致会话终止", sourceClient.GetID(), sourceClient.GetUserID())
		return
	}

	ApduMessagesRelayed.WithLabelValues(direction, sourceClient.SessionID).Inc()
	global.GVA_LOG.Info("Hub: APDU 消息已成功转发",
		zap.String("sessionID", sourceClient.SessionID),
		zap.String("fromClientID", sourceClient.GetID()),
		zap.String("toClientID", peerClient.GetID()),
		zap.String("direction", direction),
		zap.Int("apduLength", len(apduData)),
	)

	// Audit Log for APDU relay success
	// This could be redundant if apdu_relayed_attempt is sufficient
	// Or change apdu_relayed_attempt to just "apdu_relayed" and log it here after successful send.
	global.LogAuditEvent(
		"apdu_relayed_success",
		global.APDUDetails{
			Direction: direction,
			Length:    len(apduData),
		},
		zap.String("session_id", sourceClient.SessionID),
		zap.String("source_client_id", sourceClient.GetID()),
		zap.String("target_client_id", peerClient.GetID()),
		zap.String("user_id_source", sourceClient.UserID),
	)
}

// notifyProviderListSubscribers 通知特定UserID的订阅者发卡方列表已更新
func (h *Hub) notifyProviderListSubscribers(targetUserID string) {
	h.providerMutex.RLock() // RLock for reading cardProviders and sessions
	defer h.providerMutex.RUnlock()

	subscribersToNotify, ok := h.providerListSubscribers[targetUserID]
	if !ok || len(subscribersToNotify) == 0 {
		global.GVA_LOG.Debug("notifyProviderListSubscribers: No subscribers to notify for UserID or list is empty.", zap.String("targetUserID", targetUserID))
		return // No one is subscribed to this user's provider list
	}

	global.GVA_LOG.Debug("notifyProviderListSubscribers: Found subscribers to notify.",
		zap.String("targetUserID", targetUserID),
		zap.Int("subscriberCount", len(subscribersToNotify)),
	)

	var currentProvidersForUser []protocol.CardProviderInfo
	for pID, providerEntry := range h.cardProviders {
		if providerEntry.GetUserID() == targetUserID {
			// Determine if the provider is busy
			var isBusy bool
			providerSessionID := providerEntry.GetSessionID()
			if providerSessionID != "" {
				if _, sessionExists := h.sessions[providerSessionID]; sessionExists {
					isBusy = true
				}
			}

			providerClient, castOk := providerEntry.(*Client)
			providerName := "Unknown Provider"
			if castOk {
				providerName = providerClient.DisplayName
				if providerName == "" {
					providerName = "Provider " + pID[:6] // Fallback name
				}
			} else {
				global.GVA_LOG.Error("notifyProviderListSubscribers: providerEntry is not of type *Client", zap.String("providerID", pID), zap.String("targetUserID", targetUserID))
			}

			currentProvidersForUser = append(currentProvidersForUser, protocol.CardProviderInfo{
				ProviderID:   pID,
				ProviderName: providerName,
				UserID:       providerEntry.GetUserID(), // Use original providerEntry here
				IsBusy:       isBusy,
			})
		}
	}

	response := protocol.CardProvidersListMessage{
		Type:      protocol.MessageTypeCardProvidersList,
		Providers: currentProvidersForUser,
	}

	for subClient := range subscribersToNotify {
		// Log details for each subscriber before the UserID check
		global.GVA_LOG.Debug("notifyProviderListSubscribers: Processing subscriber in loop",
			zap.String("targetUserIDLoop", targetUserID), // Renamed to avoid conflict with outer scope var if any
			zap.String("subscriberClientIDLoop", subClient.GetID()),
			zap.String("subscriberClientUserIDLoop", subClient.GetUserID()),
		)
		// REMOVED: if subClient.GetUserID() == targetUserID { // Ensure only relevant subscribers get this specific list
		global.GVA_LOG.Debug("notifyProviderListSubscribers: Attempting to send CardProvidersListMessage to subscriber for targetUserID's provider list",
			zap.String("targetUserIDForList", targetUserID),
			zap.String("subscriberClientID", subClient.GetID()),
			zap.String("subscriberActualUserID", subClient.GetUserID()), // Added for clarity
			zap.Int("providerCountInList", len(response.Providers)),
		)
		if err := sendProtoMessage(subClient, response); err != nil {
			global.GVA_LOG.Error("notifyProviderListSubscribers: Failed to send provider list to subscriber",
				zap.Error(err),
				zap.String("subscriberID", subClient.GetID()),
				zap.String("targetUserIDForList", targetUserID),
			)
			// Optionally, handle unresponsive subscriber (e.g., remove from subscribers)
		}
		// REMOVED: }
	}
	global.GVA_LOG.Info("Notified subscribers about provider list update", zap.String("targetUserID", targetUserID), zap.Int("subscriberCount", len(subscribersToNotify)), zap.Int("providerCount", len(currentProvidersForUser)))
}

// handleEndSession 处理客户端主动结束会话的请求
func (h *Hub) handleEndSession(requestingClient *Client, messageBytes []byte) {
	var endMsg protocol.EndSessionMessage
	if err := json.Unmarshal(messageBytes, &endMsg); err != nil {
		global.GVA_LOG.Error("Hub 处理结束会话请求：反序列化 EndSessionMessage 失败", zap.Error(err), zap.String("clientID", requestingClient.GetID()))
		sendErrorMessage(requestingClient, protocol.ErrorCodeBadRequest, "无效的结束会话消息格式")
		return
	}

	global.GVA_LOG.Info("处理客户端结束会话请求",
		zap.String("requestingClientID", requestingClient.GetID()),
		zap.String("targetSessionID", endMsg.SessionID),
	)

	if endMsg.SessionID == "" || requestingClient.SessionID != endMsg.SessionID {
		global.GVA_LOG.Warn("客户端尝试结束不属于自己的或无效的会话",
			zap.String("requestingClientID", requestingClient.GetID()),
			zap.String("clientActualSessionID", requestingClient.SessionID),
			zap.String("requestedEndSessionID", endMsg.SessionID),
		)
		sendErrorMessage(requestingClient, protocol.ErrorCodePermissionDenied, "无法结束指定的会话：ID不匹配或无效")
		return
	}

	// We need to release the lock before calling terminateSessionByID, as it acquires its own lock.
	// However, we need the session information (activeSession) which was fetched under lock.
	// And terminateSessionByID also needs to look up the session.
	// The current terminateSessionByID handles the session deletion and participant notification.

	// We can simplify: get the session ID, unlock, then call terminate.
	targetSessionID := endMsg.SessionID
	requestingClientID := requestingClient.GetID()
	requestingClientUserID := requestingClient.UserID

	// No unlock needed here as terminateSessionByID handles its own locking.
	// h.providerMutex.Unlock() // Unlock before calling terminate // REMOVED: This was incorrect lock management

	global.GVA_LOG.Info("客户端请求结束会话，调用 terminateSessionByID",
		zap.String("sessionID", targetSessionID),
		zap.String("requestingClientID", requestingClientID),
	)

	h.terminateSessionByID(targetSessionID, "客户端主动请求结束", requestingClientID, requestingClientUserID)

	// Confirmation is now sent from within terminateSessionByID (or should be if we want consistent notification)
	// For now, let's keep the confirmation here, but ensure it's clear that termination is handled by the call above.
	// The sendProtoMessage for SessionTerminatedMessage to the *requestingClient* can remain here.
	confirmMsg := protocol.SessionTerminatedMessage{
		Type:      protocol.MessageTypeSessionTerminated,
		SessionID: targetSessionID,
		Reason:    "您已成功结束会话",
	}
	_ = sendProtoMessage(requestingClient, confirmMsg) // Ignore error for confirmation

	// The logic for notifyProviderListSubscribers if a provider becomes free is handled within terminateSessionByID.
}

// checkInactiveSessions 检查并清理不活动的会话
func (h *Hub) checkInactiveSessions() {
	h.providerMutex.RLock() // Use RLock for initial scan

	now := time.Now()
	var sessionsToTerminate []string // Slice to store IDs of sessions to terminate

	// First loop: Identify sessions to terminate
	for sessionID, s := range h.sessions {
		// 从配置加载会话不活动超时时间
		sessionTimeoutDuration := time.Duration(global.GVA_CONFIG.NfcRelay.SessionInactiveTimeoutSec) * time.Second
		if sessionTimeoutDuration <= 0 {
			// 如果配置无效或未设置，则使用一个合理的默认值，例如2分钟
			sessionTimeoutDuration = 2 * time.Minute
			global.GVA_LOG.Warn("SessionInactiveTimeoutSec 配置无效或过小，使用默认值2分钟", zap.Duration("configuredValue", sessionTimeoutDuration))
		}

		if s.LastActivityTime.IsZero() || now.Sub(s.LastActivityTime) > sessionTimeoutDuration {
			global.GVA_LOG.Info("标记不活动会话准备终止",
				zap.String("sessionID", sessionID),
				zap.Time("lastActivity", s.LastActivityTime),
				zap.Duration("inactivityDuration", now.Sub(s.LastActivityTime)),
				zap.Duration("timeoutThreshold", sessionTimeoutDuration),
			)
			sessionsToTerminate = append(sessionsToTerminate, sessionID)
		}
	}
	h.providerMutex.RUnlock() // Release RLock after scanning

	// Second part: Terminate identified sessions
	// terminateSessionByID 会自行处理其内部的锁 (h.providerMutex.Lock())
	// 所以在这里调用它之前，我们不应该持有 h.providerMutex
	if len(sessionsToTerminate) > 0 {
		for _, sessionID := range sessionsToTerminate {
			global.GVA_LOG.Info("执行不活动会话的终止", zap.String("sessionID", sessionID))
			h.terminateSessionByID(sessionID, "会话因长时间无活动已超时", "system", "")
		}
	}
}

// terminateSessionByID 终止指定ID的会话并通知参与者
// reason 用于通知客户端会话终止的原因
// actingClientID 是执行此终止操作的客户端ID，如果是系统行为则为空或特定字符串 (如 "system")
func (h *Hub) terminateSessionByID(sessionID string, reason string, actingClientID string, actingClientUserID string) {
	h.providerMutex.Lock() // Acquire lock

	sessionToEnd, ok := h.sessions[sessionID]
	if !ok {
		h.providerMutex.Unlock() // Unlock before return
		global.GVA_LOG.Warn("Hub (terminateSessionByID): Attempted to terminate a non-existent session",
			zap.String("sessionID", sessionID),
			zap.String("reason", reason),
			zap.String("actingClientID", actingClientID),
		)
		return
	}

	// Log initial client details from sessionToEnd
	var logCardClientID, logCardClientUserID, logPosClientID, logPosClientUserID string
	if sessionToEnd.CardEndClient != nil {
		logCardClientID = sessionToEnd.CardEndClient.GetID()
		logCardClientUserID = sessionToEnd.CardEndClient.GetUserID()
	} else {
		logCardClientID = "<nil>"
		logCardClientUserID = "<nil>"
	}
	if sessionToEnd.POSEndClient != nil {
		logPosClientID = sessionToEnd.POSEndClient.GetID()
		logPosClientUserID = sessionToEnd.POSEndClient.GetUserID()
	} else {
		logPosClientID = "<nil>"
		logPosClientUserID = "<nil>"
	}
	global.GVA_LOG.Debug("terminateSessionByID: Initial client details from sessionToEnd",
		zap.String("sessionID", sessionID),
		zap.String("sessionCardClientID", logCardClientID),
		zap.String("sessionCardClientUserID", logCardClientUserID),
		zap.String("sessionPosClientID", logPosClientID),
		zap.String("sessionPosClientUserID", logPosClientUserID),
	)

	// Extract client information for notifications and audit *before* unlocking and modifying client state
	// This also ensures we are working with the state as it was when the session existed.
	var cardClientConcrete, posClientConcrete *Client
	var cardClientID, cardClientUserID, cardClientRole string
	var posClientID, posClientUserID, posClientRole string

	cardClientProvider := sessionToEnd.CardEndClient
	posClientProvider := sessionToEnd.POSEndClient

	if cardClientProvider != nil {
		if cc, okAssert := cardClientProvider.(*Client); okAssert {
			cardClientConcrete = cc // Store concrete client for later use outside lock
			cardClientID = cc.GetID()
			cardClientUserID = cc.GetUserID()
			cardClientRole = string(cc.GetCurrentRole())
		} else {
			cardClientID = cardClientProvider.GetID()
			cardClientUserID = cardClientProvider.GetUserID()
			cardClientRole = string(cardClientProvider.GetRole())
			global.GVA_LOG.Warn("terminateSessionByID: CardEndClient from session was not a concrete *Client instance for audit", zap.String("sessionID", sessionID), zap.String("cardClientID", cardClientID))
		}
	} else {
		cardClientConcrete = nil // Ensure it's nil if provider is nil
	}
	if posClientProvider != nil {
		if pc, okAssert := posClientProvider.(*Client); okAssert {
			posClientConcrete = pc // Store concrete client for later use outside lock
			posClientID = pc.GetID()
			posClientUserID = pc.GetUserID()
			posClientRole = string(pc.GetCurrentRole())
		} else {
			posClientID = posClientProvider.GetID()
			posClientUserID = posClientProvider.GetUserID()
			posClientRole = string(posClientProvider.GetRole())
			global.GVA_LOG.Warn("terminateSessionByID: POSEndClient from session was not a concrete *Client instance for audit", zap.String("sessionID", sessionID), zap.String("posClientID", posClientID))
		}
	} else {
		posClientConcrete = nil // Ensure it's nil if provider is nil
	}

	// Log concrete client details after extraction
	var logConcreteCardClientID, logConcretePosClientID string
	if cardClientConcrete != nil {
		logConcreteCardClientID = cardClientConcrete.GetID()
	} else {
		logConcreteCardClientID = "<nil>"
	}
	if posClientConcrete != nil {
		logConcretePosClientID = posClientConcrete.GetID()
	} else {
		logConcretePosClientID = "<nil>"
	}
	global.GVA_LOG.Debug("terminateSessionByID: Concrete client details after extraction",
		zap.String("sessionID", sessionID),
		zap.String("concreteCardClientID", logConcreteCardClientID),
		zap.String("concretePosClientID", logConcretePosClientID),
	)

	delete(h.sessions, sessionID)
	sessionToEnd.Terminate() // Mark session object itself as terminated

	// Clear SessionID for concrete clients under the hub lock
	if cardClientConcrete != nil {
		cardClientConcrete.SessionID = ""
	}
	if posClientConcrete != nil {
		posClientConcrete.SessionID = ""
	}

	h.providerMutex.Unlock() // IMPORTANT: Unlock before sending messages

	// Metrics update
	h.metricsMutex.Lock()
	ActiveSessions.Dec()
	// Determine metricReason (copied from original logic, ensure it's correct)
	metricReason := "system_generic"
	if actingClientID != "system" {
		switch reason {
		case "客户端主动请求结束":
			metricReason = "client_request"
		case "客户端断开连接":
			metricReason = "client_disconnect"
		default:
			metricReason = "client_generic_action"
		}
	} else {
		switch reason {
		case "会话因长时间无活动已超时":
			metricReason = "timeout"
		case "APDU转发失败导致会话终止":
			metricReason = "apdu_error"
		}
	}
	SessionTerminations.WithLabelValues(metricReason).Inc()
	h.metricsMutex.Unlock()

	global.GVA_LOG.Info("会话已终止 (post-unlock log)", // Differentiate log timing if necessary
		zap.String("sessionID", sessionID),
		zap.String("reason", reason),
		zap.String("actingClientID", actingClientID),
		zap.String("actingClientUserID", actingClientUserID),
	)

	// Determine audit event type (copied from original logic)
	eventType := "session_terminated_by_system"
	if actingClientID != "system" {
		switch reason {
		case "客户端主动请求结束":
			eventType = "session_terminated_by_client_request"
		case "客户端断开连接":
			eventType = "session_terminated_by_client_disconnect"
		default:
			eventType = "session_terminated_by_client_action"
		}
	} else {
		switch reason {
		case "会话因长时间无活动已超时":
			eventType = "session_terminated_by_timeout"
		case "APDU转发失败导致会话终止":
			eventType = "session_terminated_by_apdu_error"
		}
	}

	auditTerminationDetails := map[string]interface{}{
		"session_id":       sessionID,
		"reason":           reason,
		"acting_client_id": actingClientID,
		// "acting_user_id":         actingClientUserID, // See conditional logic below
		"client_id_card_end":          cardClientID,
		"user_id_card_end":            cardClientUserID,
		"role_card_end":               cardClientRole,
		"client_id_pos_end":           posClientID,
		"user_id_pos_end":             posClientUserID,
		"role_pos_end":                posClientRole,
		"acting_client_id_in_details": actingClientID, // 添加这个字段到 details map 中
	}

	// Conditionally add acting_user_id to the map only if it's not empty
	if actingClientUserID != "" {
		auditTerminationDetails["acting_user_id"] = actingClientUserID
	}

	// 检查 auditTerminationDetails 的类型
	checkDetailsMapType := func(details map[string]interface{}, key string) {
		value := details[key]
		if value == nil {
			global.GVA_LOG.Debug("【DEBUG-TYPE】auditTerminationDetails 中的键值为 nil", zap.String("key", key))
			return
		}

		switch v := value.(type) {
		case string:
			global.GVA_LOG.Debug("【DEBUG-TYPE】auditTerminationDetails 中的键值类型为 string",
				zap.String("key", key),
				zap.String("value", v),
				zap.String("type", "string"),
			)
		case int:
			global.GVA_LOG.Debug("【DEBUG-TYPE】auditTerminationDetails 中的键值类型为 int",
				zap.String("key", key),
				zap.Int("value", v),
				zap.String("type", "int"),
			)
		case bool:
			global.GVA_LOG.Debug("【DEBUG-TYPE】auditTerminationDetails 中的键值类型为 bool",
				zap.String("key", key),
				zap.Bool("value", v),
				zap.String("type", "bool"),
			)
		case map[string]interface{}:
			global.GVA_LOG.Debug("【DEBUG-TYPE】auditTerminationDetails 中的键值类型为 map[string]interface{}",
				zap.String("key", key),
				zap.Any("value", v),
				zap.String("type", "map[string]interface{}"),
			)
		default:
			global.GVA_LOG.Debug("【DEBUG-TYPE】auditTerminationDetails 中的键值类型为其他类型",
				zap.String("key", key),
				zap.Any("value", v),
				zap.String("type", fmt.Sprintf("%T", v)),
			)
		}
	}

	// 检查所有关键字段的类型
	checkDetailsMapType(auditTerminationDetails, "acting_client_id")
	checkDetailsMapType(auditTerminationDetails, "acting_client_id_in_details")
	checkDetailsMapType(auditTerminationDetails, "reason")
	checkDetailsMapType(auditTerminationDetails, "session_id")

	// DEBUG: 详细记录 auditTerminationDetails 的内容
	global.GVA_LOG.Debug("【DEBUG】terminateSessionByID: auditTerminationDetails 详细内容",
		zap.String("sessionID", sessionID),
		zap.String("actingClientID", actingClientID),
		zap.Any("acting_client_id_in_details_value", auditTerminationDetails["acting_client_id_in_details"]),
		zap.Any("acting_client_id_value", auditTerminationDetails["acting_client_id"]),
		zap.Any("full_details_map", auditTerminationDetails),
	)

	// DEBUG: Print the acting_client_id from the auditTerminationDetails map before logging
	debugActingClientID, debugOk := auditTerminationDetails["acting_client_id"].(string)
	if !debugOk {
		global.GVA_LOG.Debug("【DEBUG】terminateSessionByID DEBUG: auditTerminationDetails[\"acting_client_id\"] is not a string or not ok", zap.Any("value", auditTerminationDetails["acting_client_id"]))
	} else {
		global.GVA_LOG.Debug("【DEBUG】terminateSessionByID DEBUG: auditTerminationDetails[\"acting_client_id\"]", zap.String("debugActingClientID", debugActingClientID))
	}

	debugActingClientIDInDetails, debugInDetailsOk := auditTerminationDetails["acting_client_id_in_details"].(string)
	if !debugInDetailsOk {
		global.GVA_LOG.Debug("【DEBUG】terminateSessionByID DEBUG: auditTerminationDetails[\"acting_client_id_in_details\"] is not a string or not ok", zap.Any("value", auditTerminationDetails["acting_client_id_in_details"]))
	} else {
		global.GVA_LOG.Debug("【DEBUG】terminateSessionByID DEBUG: auditTerminationDetails[\"acting_client_id_in_details\"]", zap.String("debugActingClientIDInDetails", debugActingClientIDInDetails))
	}

	global.GVA_LOG.Debug("【DEBUG】terminateSessionByID DEBUG: full auditTerminationDetails map", zap.Reflect("auditDetailsMap", auditTerminationDetails))

	// Reconstruct the original zap fields for LogAuditEvent
	logAuditFields := []zap.Field{
		zap.String("session_id", sessionID),
		zap.String("reason", reason),
		zap.String("acting_client_id", actingClientID),
		// zap.String("acting_user_id", actingClientUserID), // May be empty, which is fine for Zap
		zap.String("client_id_card_end", cardClientID),
		zap.String("user_id_card_end", cardClientUserID),
		zap.String("client_id_pos_end", posClientID),
		zap.String("user_id_pos_end", posClientUserID),
	}
	if actingClientUserID != "" {
		logAuditFields = append(logAuditFields, zap.String("acting_user_id", actingClientUserID))
	}

	global.GVA_LOG.Debug("【DEBUG】terminateSessionByID: 准备调用 LogAuditEvent",
		zap.String("eventType", eventType),
		zap.String("sessionID", sessionID),
		zap.String("actingClientID", actingClientID),
		zap.Int("logAuditFields_length", len(logAuditFields)),
	)

	// 调用 LogAuditEvent 前再次检查 auditTerminationDetails
	actingClientIDValue, hasActingClientID := auditTerminationDetails["acting_client_id"]
	actingClientIDInDetailsValue, hasActingClientIDInDetails := auditTerminationDetails["acting_client_id_in_details"]
	global.GVA_LOG.Debug("【DEBUG】调用 LogAuditEvent 前的 auditTerminationDetails 检查",
		zap.Bool("hasActingClientID", hasActingClientID),
		zap.Any("actingClientIDValue", actingClientIDValue),
		zap.Bool("hasActingClientIDInDetails", hasActingClientIDInDetails),
		zap.Any("actingClientIDInDetailsValue", actingClientIDInDetailsValue),
	)

	global.LogAuditEvent(
		eventType,
		auditTerminationDetails,
		logAuditFields..., // 移除单独的 acting_client_id_in_details 字段
	)

	global.GVA_LOG.Debug("【DEBUG】terminateSessionByID: LogAuditEvent 调用完成",
		zap.String("eventType", eventType),
		zap.String("sessionID", sessionID),
		zap.String("actingClientID", actingClientID),
	)

	terminationMsg := protocol.SessionTerminatedMessage{
		Type:      protocol.MessageTypeSessionTerminated,
		SessionID: sessionID, // Use the original sessionID for the message
		Reason:    reason,
	}

	var providerBecameFreeUserID string // For notifying subscribers

	// Send messages to clients (now outside the hub lock)
	if cardClientConcrete != nil {
		global.GVA_LOG.Debug("terminateSessionByID: Attempting to send SessionTerminatedMessage to cardClientConcrete",
			zap.String("sessionID", sessionID),
			zap.String("cardClientID", cardClientConcrete.GetID()),
			zap.String("reasonForMsg", terminationMsg.Reason),
		)
		if err := sendProtoMessage(cardClientConcrete, terminationMsg); err != nil {
			global.GVA_LOG.Warn("Hub (terminateSessionByID): 发送会话终止消息给 CardEndClient 失败",
				zap.Error(err), zap.String("sessionID", sessionID), zap.String("cardClientID", cardClientConcrete.GetID()))
		}
		if cardClientConcrete.CurrentRole == protocol.RoleProvider {
			providerBecameFreeUserID = cardClientConcrete.GetUserID()
		}
	} else if cardClientProvider != nil { // Fallback if not concrete, but existed
		global.GVA_LOG.Debug("terminateSessionByID: Attempting to send SessionTerminatedMessage to cardClientProvider (interface)",
			zap.String("sessionID", sessionID),
			zap.String("cardClientID", cardClientProvider.GetID()),
			zap.String("reasonForMsg", terminationMsg.Reason),
		)
		if err := sendProtoMessage(cardClientProvider, terminationMsg); err != nil { // Assumes ClientInfoProvider has Send
			global.GVA_LOG.Warn("Hub (terminateSessionByID): 发送会话终止消息给 CardEndClient (interface) 失败", zap.Error(err), zap.String("sessionID", sessionID), zap.String("clientID", cardClientProvider.GetID()))
		}
	}

	if posClientConcrete != nil {
		global.GVA_LOG.Debug("terminateSessionByID: Attempting to send SessionTerminatedMessage to posClientConcrete",
			zap.String("sessionID", sessionID),
			zap.String("posClientID", posClientConcrete.GetID()),
			zap.String("reasonForMsg", terminationMsg.Reason),
		)
		if err := sendProtoMessage(posClientConcrete, terminationMsg); err != nil {
			global.GVA_LOG.Warn("Hub (terminateSessionByID): 发送会话终止消息给 POSEndClient 失败",
				zap.Error(err), zap.String("sessionID", sessionID), zap.String("posClientID", posClientConcrete.GetID()))
		}
		if posClientConcrete.CurrentRole == protocol.RoleProvider {
			if providerBecameFreeUserID == "" {
				providerBecameFreeUserID = posClientConcrete.GetUserID()
			} else if providerBecameFreeUserID != posClientConcrete.GetUserID() {
				// This case means both card and pos ends were providers (unlikely for typical NFC relay but possible if roles are flexible)
				// and they belong to different users. Notify subscribers for the second provider user too.
				global.GVA_LOG.Info("Hub (terminateSessionByID): Both session ends were Providers of different users. Notifying for second provider.",
					zap.String("sessionID", sessionID),
					zap.String("secondProviderClientID", posClientConcrete.GetID()),
					zap.String("secondProviderUserID", posClientConcrete.GetUserID()))
				go h.notifyProviderListSubscribers(posClientConcrete.GetUserID())
			}
		}
	} else if posClientProvider != nil { // Fallback
		global.GVA_LOG.Debug("terminateSessionByID: Attempting to send SessionTerminatedMessage to posClientProvider (interface)",
			zap.String("sessionID", sessionID),
			zap.String("posClientID", posClientProvider.GetID()),
			zap.String("reasonForMsg", terminationMsg.Reason),
		)
		if err := sendProtoMessage(posClientProvider, terminationMsg); err != nil {
			global.GVA_LOG.Warn("Hub (terminateSessionByID): 发送会话终止消息给 POSEndClient (interface) 失败", zap.Error(err), zap.String("sessionID", sessionID), zap.String("clientID", posClientProvider.GetID()))
		}
	}

	// Notify subscribers if a provider became free
	if providerBecameFreeUserID != "" {
		global.GVA_LOG.Info("Hub (terminateSessionByID): Session terminated, a provider may have become free. Preparing to notify subscribers.",
			zap.String("sessionID", sessionID),
			zap.String("providerFreedForUserID", providerBecameFreeUserID), // Log the actual UserID
		)
		go h.notifyProviderListSubscribers(providerBecameFreeUserID)
	} else {
		global.GVA_LOG.Debug("Hub (terminateSessionByID): No provider became free or providerBecameFreeUserID is empty, skipping subscriber notification.",
			zap.String("sessionID", sessionID),
			zap.String("evaluatedProviderBecameFreeUserID", providerBecameFreeUserID), // Log the (empty) UserID
		)
	}
}

// GlobalRelayHub 是 Hub 的一个全局实例。
// 这是一个常见的模式，但对于大型应用程序，请考虑使用依赖注入。
var GlobalRelayHub = NewHub()
