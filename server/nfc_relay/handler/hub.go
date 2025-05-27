package handler

import (
	"encoding/json"
	"errors"
	"fmt"
	"strconv"
	"strings"
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
		global.GVA_LOG.Debug("Hub: Notifying provider list subscribers due to role/status change", zap.String("userID", userIDToNotify), zap.String("triggeringClientID", client.GetID()))
		go h.notifyProviderListSubscribers(userIDToNotify)
	}
}

// handleClientDisconnect 处理客户端断开连接时与会话相关的逻辑
func (h *Hub) handleClientDisconnect(client *Client) {
	// 从发卡方列表中移除 (如果存在)
	if client.CurrentRole == "provider" {
		h.providerMutex.Lock()
		delete(h.cardProviders, client.GetID())
		// 需要通知相关 UserID 的订阅者，因为一个 provider 下线了
		userIDOfDisconnectedProvider := client.UserID
		h.providerMutex.Unlock() // 先解锁再通知，避免死锁
		global.GVA_LOG.Info("发卡方客户端断开连接，已从可用列表移除", zap.String("clientID", client.GetID()), zap.String("userID", client.UserID))
		if userIDOfDisconnectedProvider != "" {
			go h.notifyProviderListSubscribers(userIDOfDisconnectedProvider)
		}
	}

	var sessionToEndNotifyUserID string
	originalSessionID := client.SessionID // Store original session ID before it's potentially cleared

	if client.SessionID != "" {
		// We need to lock earlier to safely read client.SessionID and then h.sessions
		h.providerMutex.Lock()
		s, exists := h.sessions[client.SessionID]
		if exists {
			global.GVA_LOG.Info("客户端断开，开始处理其活动会话的终止",
				zap.String("clientID", client.ID),
				zap.String("sessionID", client.SessionID),
			)
			// Store peer info before session is deleted by terminateSessionByID
			peer := s.GetPeer(client)
			var peerUserIDIfProvider string
			if peerConcrete, ok := peer.(*Client); ok {
				if peerConcrete.CurrentRole == "provider" {
					peerUserIDIfProvider = peerConcrete.UserID
				}
			}
			h.providerMutex.Unlock() // Unlock before calling terminateSessionByID which locks internally

			h.terminateSessionByID(originalSessionID, "客户端断开连接", client.GetID(), client.GetUserID())

			// After termination, if the peer was a provider and became free, we might need to notify its subscribers
			// This logic is now partly handled within terminateSessionByID if a provider becomes free.
			// However, notifyProviderListSubscribers relies on UserID. Let's check if the peer (now free) needs its list updated.
			if peerUserIDIfProvider != "" {
				sessionToEndNotifyUserID = peerUserIDIfProvider
			}

		} else {
			// Session doesn't exist in h.sessions, might have been terminated by other means.
			// Clear client's session ID just in case.
			client.SessionID = ""
			h.providerMutex.Unlock()
		}
	} else {
		// Client was not in a session or SessionID was already cleared.
	}

	// If a provider (the peer of the disconnected client) became free due to session termination
	if sessionToEndNotifyUserID != "" {
		go h.notifyProviderListSubscribers(sessionToEndNotifyUserID)
	}

	// 从所有 providerListSubscribers 中移除断开的客户端
	h.providerMutex.Lock()
	for userID, subscribers := range h.providerListSubscribers {
		if _, ok := subscribers[client]; ok {
			delete(h.providerListSubscribers[userID], client)
			if len(h.providerListSubscribers[userID]) == 0 {
				delete(h.providerListSubscribers, userID)
			}
			global.GVA_LOG.Debug("已从发卡方列表订阅者中移除断开的客户端", zap.String("clientID", client.GetID()), zap.String("subscribedToUserID", userID))
		}
	}
	h.providerMutex.Unlock()
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
	targetProviderInterface, providerExists := h.cardProviders[providerIDToSelect]
	if !providerExists {
		h.providerMutex.RUnlock()
		global.GVA_LOG.Warn("Hub: 目标发卡方不存在或未上线", zap.String("targetProviderID", providerIDToSelect), zap.String("requestingClientID", requestingClient.GetID()))
		sendErrorMessage(requestingClient, protocol.ErrorCodeProviderNotFound, "选择发卡方失败：目标发卡方不存在或当前未提供服务。")
		// 尝试通知发卡方列表的订阅者，如果此 provider 真的消失了 (可能因为之前的逻辑没及时通知)
		// go h.notifyProviderListSubscribers(requestingClient.GetUserID()) // 谨慎使用，避免不必要的通知风暴
		return
	}

	var targetProviderConcrete *Client
	var ok bool
	if targetProviderInterface != nil {
		targetProviderConcrete, ok = targetProviderInterface.(*Client)
	} else {
		ok = false
	}

	if !ok {
		h.providerMutex.RUnlock()
		global.GVA_LOG.Error("Hub: cardProviders map 中的条目不是 *Client 类型或接口为nil", zap.String("providerID", providerIDToSelect))
		sendErrorMessage(requestingClient, protocol.ErrorCodeInternalError, "选择发卡方失败：服务器内部错误。")
		return
	}

	// 将 *Client 重新转换为 ClientInfoProvider 接口类型进行后续操作
	var providerAsInterface session.ClientInfoProvider = targetProviderConcrete

	// 检查1: 请求者和提供者是否属于同一个 UserID
	if requestingClient.GetUserID() != providerAsInterface.GetUserID() {
		h.providerMutex.RUnlock()
		global.GVA_LOG.Warn("Hub: 请求者和提供者 UserID 不匹配",
			zap.String("requestingUserID", requestingClient.GetUserID()),
			zap.String("providerUserID", providerAsInterface.GetUserID()),
			zap.String("requestingClientID", requestingClient.GetID()),
			zap.String("providerClientID", providerAsInterface.GetID()),
		)
		sendErrorMessage(requestingClient, protocol.ErrorCodePermissionDenied, "选择发卡方失败：不能选择其他账户下的发卡方。")
		return
	}

	// 检查2: 不能选择自己作为发卡方
	if requestingClient.GetID() == providerAsInterface.GetID() {
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
	if providerAsInterface.GetSessionID() != "" { // 使用 providerAsInterface
		if _, sessionExists := h.sessions[providerAsInterface.GetSessionID()]; sessionExists { // 使用 providerAsInterface
			h.providerMutex.RUnlock()
			global.GVA_LOG.Warn("Hub: 目标发卡方已在会话中",
				zap.String("targetProviderID", providerAsInterface.GetID()), // 使用 providerAsInterface
				zap.String("sessionID", providerAsInterface.GetSessionID()), // 使用 providerAsInterface
			)
			sendErrorMessage(requestingClient, protocol.ErrorCodeProviderBusy, "选择发卡方失败：目标发卡方当前正忙。")
			return
		}
	}
	h.providerMutex.RUnlock() // 读取完成，释放读锁

	// --- 到此所有只读检查完成，如果需要修改共享数据，则需要重新获取写锁 ---
	h.providerMutex.Lock() // 获取写锁以创建会话和更新客户端状态
	defer h.providerMutex.Unlock()

	// 再次检查条件（因为在释放读锁和获取写锁之间状态可能改变）
	// 检查 Provider 是否仍然存在且在线
	// currentTargetProvider 仍然是接口类型 session.ClientInfoProvider
	currentTargetProviderInterface, stillExists := h.cardProviders[providerIDToSelect]
	if !stillExists {
		sendErrorMessage(requestingClient, protocol.ErrorCodeProviderUnavailable, "选择发卡方失败：目标发卡方状态已改变(不存在)，请重试。")
		return
	}
	// 需要再次断言，或者确保 currentTargetProviderInterface 就是我们想要的 *Client
	currentTargetProviderConcrete, castOk := currentTargetProviderInterface.(*Client)
	if !castOk {
		sendErrorMessage(requestingClient, protocol.ErrorCodeInternalError, "选择发卡方失败：目标发卡方类型错误。")
		return
	}

	if currentTargetProviderConcrete.GetUserID() != requestingClient.GetUserID() || currentTargetProviderConcrete.GetID() == requestingClient.GetID() {
		sendErrorMessage(requestingClient, protocol.ErrorCodeProviderUnavailable, "选择发卡方失败：目标发卡方状态已改变(UID或ID不匹配)，请重试。")
		return
	}

	// 检查双方是否仍然空闲
	if requestingClient.GetSessionID() != "" || currentTargetProviderConcrete.GetSessionID() != "" { // 使用 concrete 类型进行检查
		sendErrorMessage(requestingClient, protocol.ErrorCodeSessionConflict, "选择发卡方失败：一方或双方状态已改变（不再空闲），请重试。")
		return
	}

	// 创建新会话
	formalSessionID := uuid.NewString()
	newSession := session.NewSession(formalSessionID)
	newSession.SetClient(requestingClient, string(protocol.RoleReceiver))
	newSession.SetClient(currentTargetProviderConcrete, string(protocol.RoleProvider)) // 传递 concrete type
	h.sessions[formalSessionID] = newSession

	// 更新客户端的 SessionID
	requestingClient.SessionID = formalSessionID
	currentTargetProviderConcrete.SessionID = formalSessionID

	global.GVA_LOG.Info("Hub: Receiver selected provider, new session created and clients paired.",
		zap.String("sessionID", formalSessionID),
		zap.String("receiverClientID", requestingClient.GetID()),
		zap.String("providerClientID", currentTargetProviderConcrete.GetID()),
	)

	// Audit Log for session establishment
	global.LogAuditEvent(
		"session_established",
		global.SessionDetails{
			InitiatorRole: string(requestingClient.CurrentRole),
			ResponderRole: string(currentTargetProviderConcrete.CurrentRole),
		},
		zap.String("session_id", formalSessionID),
		zap.String("client_id_initiator", requestingClient.GetID()),
		zap.String("user_id_initiator", requestingClient.UserID),
		zap.String("client_id_responder", currentTargetProviderConcrete.GetID()),
		zap.String("user_id_responder", currentTargetProviderConcrete.GetUserID()),
		zap.String("source_ip_initiator", requestingClient.conn.RemoteAddr().String()),
		zap.String("source_ip_responder", currentTargetProviderConcrete.conn.RemoteAddr().String()),
	)

	ActiveSessions.Inc() // Increment active sessions metric

	// 通知双方会话已建立
	// 向 receiver 发送
	errReceiver := sendProtoMessage(requestingClient, protocol.SessionEstablishedMessage{
		Type:      protocol.MessageTypeSessionEstablished,
		SessionID: formalSessionID,
		PeerID:    currentTargetProviderConcrete.GetID(),
		PeerRole:  protocol.RoleProvider,
	})
	if errReceiver != nil {
		global.GVA_LOG.Error("发送会话建立成功的消息给收卡方失败", zap.Error(errReceiver), zap.String("receiverClientID", requestingClient.GetID()))
		// TODO: 考虑回滚会话创建或通知另一方
	}

	// 向 provider 发送
	errProvider := sendProtoMessage(currentTargetProviderConcrete, protocol.SessionEstablishedMessage{ // 使用 concrete type
		Type:      protocol.MessageTypeSessionEstablished,
		SessionID: formalSessionID,
		PeerID:    requestingClient.GetID(),
		PeerRole:  protocol.RoleReceiver,
	})
	if errProvider != nil {
		global.GVA_LOG.Error("发送会话建立成功的消息给发卡方失败", zap.Error(errProvider), zap.String("providerClientID", currentTargetProviderConcrete.GetID()))
		// TODO: 考虑回滚会话创建或通知另一方
	}

	// 提供者进入会话后状态变为繁忙，通知其 UserID 下的订阅者列表更新
	go h.notifyProviderListSubscribers(currentTargetProviderConcrete.GetUserID())

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
		return // No one is subscribed to this user's provider list
	}

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
		if subClient.GetUserID() == targetUserID { // Ensure only relevant subscribers get this specific list
			if err := sendProtoMessage(subClient, response); err != nil {
				global.GVA_LOG.Error("notifyProviderListSubscribers: Failed to send provider list to subscriber",
					zap.Error(err),
					zap.String("subscriberID", subClient.GetID()),
					zap.String("targetUserID", targetUserID),
				)
				// Optionally, handle unresponsive subscriber (e.g., remove from subscribers)
			}
		}
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
	h.providerMutex.Lock()
	defer h.providerMutex.Unlock()

	now := time.Now()
	for sessionID, s := range h.sessions {
		// 检查会话是否超时
		inactivityTimeout := time.Duration(global.GVA_CONFIG.NfcRelay.HubCheckIntervalSec) * time.Second // 使用与 Hub 检查相同的间隔，或定义单独的会话超时配置
		if s.LastActivityTime.IsZero() || now.Sub(s.LastActivityTime) > inactivityTimeout*2 {            // 乘以2作为宽限期
			global.GVA_LOG.Info("检测到不活动会话，正在终止",
				zap.String("sessionID", sessionID),
				zap.Time("lastActivity", s.LastActivityTime),
				zap.Duration("inactivityDuration", now.Sub(s.LastActivityTime)),
			)
			// 从 sessions map 中删除
			delete(h.sessions, sessionID)
			// 其他清理逻辑，例如通知相关客户端（如果需要）
			// ...
			ActiveSessions.Dec()
			SessionTerminations.WithLabelValues("timeout").Inc()
		}
	}
}

// terminateSessionByID 终止指定ID的会话并通知参与者
// reason 用于通知客户端会话终止的原因
// actingClientID 是执行此终止操作的客户端ID，如果是系统行为则为空或特定字符串 (如 "system")
func (h *Hub) terminateSessionByID(sessionID string, reason string, actingClientID string, actingClientUserID string) {
	h.providerMutex.Lock()
	defer h.providerMutex.Unlock()

	activeSession, sessionExists := h.sessions[sessionID]
	if !sessionExists {
		global.GVA_LOG.Warn("Hub (terminateSessionByID): Attempted to terminate a non-existent session", zap.String("sessionID", sessionID))
		return // 会话已不存在
	}

	// 从 Hub 的活动会话列表中移除
	delete(h.sessions, sessionID)
	activeSession.Terminate() // 标记会话对象本身为终止状态
	ActiveSessions.Dec()      // Decrement active sessions metric

	global.GVA_LOG.Info("会话已终止", zap.String("sessionID", sessionID), zap.String("reason", reason), zap.String("actingClientID", actingClientID))

	// Audit Log for session termination
	var eventType string

	// Determine eventType based on reason or actingClientID
	if actingClientID != "" && actingClientID != "system" { // Assumes system passes "system" or empty
		if strings.Contains(reason, "客户端主动请求结束") {
			eventType = "session_terminated_by_client_request"
		} else if strings.Contains(reason, "客户端断开连接") { // Check if reason indicates a disconnect
			eventType = "session_terminated_by_client_disconnect"
		} else {
			// Default if an actingClientID is present but reason doesn't match specific client-initiated reasons
			eventType = "session_terminated_by_client_action"
		}
	} else if strings.Contains(reason, "会话因长时间无活动已超时") {
		eventType = "session_terminated_by_timeout"
	} else if strings.Contains(reason, "APDU转发失败导致会话终止") {
		eventType = "session_terminated_by_apdu_error"
	} else {
		eventType = "session_terminated_by_system" // Default system termination
	}

	// Increment session termination metric
	// Use a simplified reason for the metric label to avoid too many unique label values.
	metricReason := eventType // Default to eventType
	if strings.Contains(reason, "客户端主动请求结束") {
		metricReason = "client_request"
	} else if strings.Contains(reason, "客户端断开连接") {
		metricReason = "client_disconnect"
	} else if strings.Contains(reason, "会话因长时间无活动已超时") {
		metricReason = "timeout"
	} else if strings.Contains(reason, "APDU转发失败导致会话终止") {
		metricReason = "apdu_error"
	} else if actingClientID == "system" {
		metricReason = "system_generic"
	} else if actingClientID != "" {
		metricReason = "client_generic_action"
	}
	SessionTerminations.WithLabelValues(metricReason).Inc()

	// Determine participants from activeSession
	var p1ID, p2ID, p1UserID, p2UserID string
	var p1Role, p2Role string

	if activeSession.CardEndClient != nil {
		p1ID = activeSession.CardEndClient.GetID()
		p1UserID = activeSession.CardEndClient.GetUserID()
		p1Role = activeSession.CardEndClient.GetRole()
	}
	if activeSession.POSEndClient != nil {
		p2ID = activeSession.POSEndClient.GetID()
		p2UserID = activeSession.POSEndClient.GetUserID()
		p2Role = activeSession.POSEndClient.GetRole()
	}

	fields := []zap.Field{
		zap.String("session_id", sessionID),
		zap.String("reason", reason),
		zap.String("participant1_id", p1ID),
		zap.String("participant1_user_id", p1UserID),
		zap.String("participant1_role", p1Role),
		zap.String("participant2_id", p2ID),
		zap.String("participant2_user_id", p2UserID),
		zap.String("participant2_role", p2Role),
	}

	if actingClientID != "" {
		fields = append(fields, zap.String("acting_client_id", actingClientID))
	}
	if actingClientUserID != "" {
		fields = append(fields, zap.String("acting_user_id", actingClientUserID))
	}

	global.LogAuditEvent(eventType, global.SessionDetails{}, fields...)

	var participantsToNotify []session.ClientInfoProvider
	if activeSession.CardEndClient != nil {
		participantsToNotify = append(participantsToNotify, activeSession.CardEndClient)
	}
	if activeSession.POSEndClient != nil && activeSession.POSEndClient != activeSession.CardEndClient {
		participantsToNotify = append(participantsToNotify, activeSession.POSEndClient)
	}

	var providerBecameFreeUserID string

	for _, clientProvider := range participantsToNotify {
		if client, ok := clientProvider.(*Client); ok {
			client.SessionID = "" // 清理客户端的会话ID
			// 如果此客户端是 Provider 并且之前在会话中，现在它变为空闲
			if client.CurrentRole == protocol.RoleProvider {
				providerBecameFreeUserID = client.UserID
			}
		}
		// 发送会话终止通知给客户端
		// 注意: 使用 protocol.MessageTypeSessionTerminated
		// 这里的 SessionTerminatedMessage 定义与 nfc-api.md 一致，包含 sessionId 和 reason
		// 新文档中也推荐了一个类似的 session_terminated 消息。
		termMsg := protocol.SessionTerminatedMessage{
			Type:      protocol.MessageTypeSessionTerminated,
			SessionID: sessionID, // 使用原始的会话ID
			Reason:    reason,
		}
		_ = sendProtoMessage(clientProvider, termMsg) // 尝试通知，忽略错误
	}

	if providerBecameFreeUserID != "" {
		// 异步通知，避免锁竞争和阻塞Hub主循环
		go h.notifyProviderListSubscribers(providerBecameFreeUserID)
	}
}

// GlobalRelayHub 是 Hub 的一个全局实例。
// 这是一个常见的模式，但对于大型应用程序，请考虑使用依赖注入。
var GlobalRelayHub = NewHub()
