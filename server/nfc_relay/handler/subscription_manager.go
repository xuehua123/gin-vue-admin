package handler

import (
	"encoding/json"
	"errors"
	"sync"
	"time"

	"github.com/flipped-aurora/gin-vue-admin/server/global"
	"go.uber.org/zap"
)

// 定义错误类型
var (
	ErrClientNotFound = errors.New("client not found")
)

// MessageType 定义WebSocket消息类型
type MessageType string

const (
	// 心跳消息
	MessageTypePing MessageType = "ping"
	MessageTypePong MessageType = "pong"

	// 数据消息
	MessageTypeRealTimeData MessageType = "realtime_data"
	MessageTypeLogEntry     MessageType = "log_entry"
	MessageTypeApduData     MessageType = "apdu_data"
	MessageTypeMetricsData  MessageType = "metrics_data"

	// 控制消息
	MessageTypeSubscribe   MessageType = "subscribe"
	MessageTypeUnsubscribe MessageType = "unsubscribe"
	MessageTypeError       MessageType = "error"
)

// SubscriptionTopic 定义订阅主题
type SubscriptionTopic string

const (
	TopicLogs     SubscriptionTopic = "logs"
	TopicApdu     SubscriptionTopic = "apdu"
	TopicMetrics  SubscriptionTopic = "metrics"
	TopicRealtime SubscriptionTopic = "realtime"
)

// WebSocketMessage 定义WebSocket消息结构
type WebSocketMessage struct {
	Type      MessageType       `json:"type"`
	Topic     SubscriptionTopic `json:"topic,omitempty"`
	Data      interface{}       `json:"data,omitempty"`
	Timestamp time.Time         `json:"timestamp"`
	ClientID  string            `json:"client_id,omitempty"`
}

// SubscriptionManager 管理WebSocket客户端的订阅
type SubscriptionManager struct {
	mu            sync.RWMutex
	subscriptions map[string]map[SubscriptionTopic]bool // clientID -> topic -> subscribed
	clients       map[string]*Client                    // clientID -> client
}

// NewSubscriptionManager 创建新的订阅管理器
func NewSubscriptionManager() *SubscriptionManager {
	return &SubscriptionManager{
		subscriptions: make(map[string]map[SubscriptionTopic]bool),
		clients:       make(map[string]*Client),
	}
}

// AddClient 添加客户端
func (sm *SubscriptionManager) AddClient(client *Client) {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	sm.clients[client.ID] = client
	sm.subscriptions[client.ID] = make(map[SubscriptionTopic]bool)

	global.GVA_LOG.Info("订阅管理器：添加客户端", zap.String("clientID", client.ID))
}

// RemoveClient 移除客户端
func (sm *SubscriptionManager) RemoveClient(clientID string) {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	delete(sm.clients, clientID)
	delete(sm.subscriptions, clientID)

	global.GVA_LOG.Info("订阅管理器：移除客户端", zap.String("clientID", clientID))
}

// Subscribe 订阅主题
func (sm *SubscriptionManager) Subscribe(clientID string, topic SubscriptionTopic) {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	if subscriptions, exists := sm.subscriptions[clientID]; exists {
		subscriptions[topic] = true
		global.GVA_LOG.Info("订阅管理器：客户端订阅主题",
			zap.String("clientID", clientID),
			zap.String("topic", string(topic)))
	}
}

// Unsubscribe 取消订阅主题
func (sm *SubscriptionManager) Unsubscribe(clientID string, topic SubscriptionTopic) {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	if subscriptions, exists := sm.subscriptions[clientID]; exists {
		subscriptions[topic] = false
		global.GVA_LOG.Info("订阅管理器：客户端取消订阅主题",
			zap.String("clientID", clientID),
			zap.String("topic", string(topic)))
	}
}

// IsSubscribed 检查客户端是否订阅了主题
func (sm *SubscriptionManager) IsSubscribed(clientID string, topic SubscriptionTopic) bool {
	sm.mu.RLock()
	defer sm.mu.RUnlock()

	if subscriptions, exists := sm.subscriptions[clientID]; exists {
		return subscriptions[topic]
	}
	return false
}

// BroadcastToTopic 向订阅了特定主题的所有客户端广播消息
func (sm *SubscriptionManager) BroadcastToTopic(topic SubscriptionTopic, msgType MessageType, data interface{}) {
	sm.mu.RLock()
	defer sm.mu.RUnlock()

	message := WebSocketMessage{
		Type:      msgType,
		Topic:     topic,
		Data:      data,
		Timestamp: time.Now(),
	}

	messageBytes, err := json.Marshal(message)
	if err != nil {
		global.GVA_LOG.Error("订阅管理器：序列化消息失败",
			zap.Error(err),
			zap.String("topic", string(topic)))
		return
	}

	sentCount := 0
	for clientID, subscriptions := range sm.subscriptions {
		if subscriptions[topic] {
			if client, exists := sm.clients[clientID]; exists {
				err := client.Send(messageBytes)
				if err != nil {
					global.GVA_LOG.Warn("订阅管理器：发送消息到客户端失败",
						zap.Error(err),
						zap.String("clientID", clientID),
						zap.String("topic", string(topic)))
				} else {
					sentCount++
				}
			}
		}
	}

	global.GVA_LOG.Debug("订阅管理器：广播消息完成",
		zap.String("topic", string(topic)),
		zap.Int("sentCount", sentCount))
}

// SendToClient 向特定客户端发送消息
func (sm *SubscriptionManager) SendToClient(clientID string, msgType MessageType, data interface{}) error {
	sm.mu.RLock()
	defer sm.mu.RUnlock()

	client, exists := sm.clients[clientID]
	if !exists {
		return ErrClientNotFound
	}

	message := WebSocketMessage{
		Type:      msgType,
		Data:      data,
		Timestamp: time.Now(),
		ClientID:  clientID,
	}

	messageBytes, err := json.Marshal(message)
	if err != nil {
		return err
	}

	return client.Send(messageBytes)
}

// GetSubscribedClients 获取订阅了特定主题的客户端列表
func (sm *SubscriptionManager) GetSubscribedClients(topic SubscriptionTopic) []string {
	sm.mu.RLock()
	defer sm.mu.RUnlock()

	var clients []string
	for clientID, subscriptions := range sm.subscriptions {
		if subscriptions[topic] {
			clients = append(clients, clientID)
		}
	}

	return clients
}

// GetClientSubscriptions 获取客户端的所有订阅
func (sm *SubscriptionManager) GetClientSubscriptions(clientID string) []SubscriptionTopic {
	sm.mu.RLock()
	defer sm.mu.RUnlock()

	var topics []SubscriptionTopic
	if subscriptions, exists := sm.subscriptions[clientID]; exists {
		for topic, subscribed := range subscriptions {
			if subscribed {
				topics = append(topics, topic)
			}
		}
	}

	return topics
}

// ProcessSubscriptionMessage 处理订阅相关消息
func (sm *SubscriptionManager) ProcessSubscriptionMessage(client *Client, message WebSocketMessage) {
	switch message.Type {
	case MessageTypeSubscribe:
		if message.Topic != "" {
			sm.Subscribe(client.ID, message.Topic)
			// 发送确认消息
			sm.SendToClient(client.ID, MessageTypePong, map[string]interface{}{
				"action": "subscribed",
				"topic":  message.Topic,
			})
		}

	case MessageTypeUnsubscribe:
		if message.Topic != "" {
			sm.Unsubscribe(client.ID, message.Topic)
			// 发送确认消息
			sm.SendToClient(client.ID, MessageTypePong, map[string]interface{}{
				"action": "unsubscribed",
				"topic":  message.Topic,
			})
		}

	case MessageTypePing:
		// 响应心跳
		sm.SendToClient(client.ID, MessageTypePong, map[string]interface{}{
			"action": "heartbeat",
		})
	}
}

// GetStats 获取订阅统计信息
func (sm *SubscriptionManager) GetStats() map[string]interface{} {
	sm.mu.RLock()
	defer sm.mu.RUnlock()

	stats := make(map[string]interface{})
	stats["total_clients"] = len(sm.clients)

	topicStats := make(map[string]int)
	for _, subscriptions := range sm.subscriptions {
		for topic, subscribed := range subscriptions {
			if subscribed {
				topicStats[string(topic)]++
			}
		}
	}
	stats["topic_subscriptions"] = topicStats

	return stats
}
