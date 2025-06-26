package nfc_relay

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/gin-gonic/gin"

	mqtt "github.com/eclipse/paho.mqtt.golang"
	"github.com/flipped-aurora/gin-vue-admin/server/global"
	"github.com/flipped-aurora/gin-vue-admin/server/model/common"
	"github.com/flipped-aurora/gin-vue-admin/server/model/nfc_relay"
	systemReq "github.com/flipped-aurora/gin-vue-admin/server/model/system/request"
	"github.com/redis/go-redis/v9"
	"go.uber.org/zap"
	"gorm.io/datatypes"
	"gorm.io/gorm"
)

const (
	MQTTEventClientConnected    = "client.connected"
	MQTTEventClientDisconnected = "client.disconnected"

	RoleReceiver    = "receiver"
	RoleTransmitter = "transmitter"

	// KickoutTopicFormat defines the MQTT topic format for kicking out a client.
	// %s will be replaced with the clientID.
	KickoutTopicFormat = "kickout/%s"
)

type MqttService struct {
	client      mqtt.Client
	mu          sync.RWMutex
	isConnected bool
}

// NFCMessage NFC消息结构
type NFCMessage struct {
	MessageID      string                 `json:"message_id"`
	TransactionID  string                 `json:"transaction_id"`
	ClientID       string                 `json:"client_id"`
	MessageType    string                 `json:"message_type"`
	Direction      string                 `json:"direction"`
	Timestamp      time.Time              `json:"timestamp"`
	Payload        map[string]interface{} `json:"payload"`
	SequenceNumber int                    `json:"sequence_number,omitempty"`
}

// APDUMessage APDU消息结构
type APDUMessage struct {
	TransactionID  string `json:"transaction_id"`
	SequenceNumber int    `json:"sequence_number"`
	Direction      string `json:"direction"`
	APDUHex        string `json:"apdu_hex"`
	Priority       string `json:"priority"`
	MessageType    string `json:"message_type"`
	Timeout        int    `json:"timeout"`
}

// HeartbeatMessage 心跳消息结构
type HeartbeatMessage struct {
	ClientID     string                 `json:"client_id"`
	Role         string                 `json:"role"`
	Status       string                 `json:"status"`
	Timestamp    time.Time              `json:"timestamp"`
	Version      string                 `json:"version"`
	DeviceInfo   map[string]interface{} `json:"device_info"`
	Capabilities []string               `json:"capabilities"`
}

// StatusMessage 状态消息结构
type StatusMessage struct {
	TransactionID string                 `json:"transaction_id"`
	ClientID      string                 `json:"client_id"`
	Status        string                 `json:"status"`
	Timestamp     time.Time              `json:"timestamp"`
	Metadata      map[string]interface{} `json:"metadata,omitempty"`
}

// PairingNotificationMessage 配对通知消息结构
type PairingNotificationMessage struct {
	MessageID        string                 `json:"message_id"`
	NotificationType string                 `json:"notification_type"` // "success", "cancelled", "timeout", "error"
	ClientID         string                 `json:"client_id"`
	MessageType      string                 `json:"message_type"` // "pairing_notification"
	Direction        string                 `json:"direction"`    // "server_to_client"
	Timestamp        time.Time              `json:"timestamp"`
	Payload          map[string]interface{} `json:"payload"`
}

var mqttService *MqttService
var mqttOnce sync.Once

// GetMQTTService 获取MQTT服务实例（单例模式）
func GetMQTTService() *MqttService {
	mqttOnce.Do(func() {
		mqttService = &MqttService{}
	})
	return mqttService
}

// Initialize 初始化MQTT连接
func (s *MqttService) Initialize() error {
	if global.GVA_CONFIG.MQTT.Host == "" {
		return fmt.Errorf("MQTT配置不完整")
	}

	// 创建MQTT客户端选项
	opts := mqtt.NewClientOptions()

	// 构建连接URL
	var protocol string
	if global.GVA_CONFIG.MQTT.UseTLS {
		protocol = "ssl"
	} else {
		protocol = "tcp"
	}
	brokerURL := fmt.Sprintf("%s://%s:%d", protocol, global.GVA_CONFIG.MQTT.Host, global.GVA_CONFIG.MQTT.Port)
	opts.AddBroker(brokerURL)

	opts.SetClientID(global.GVA_CONFIG.MQTT.ClientID)
	opts.SetUsername(global.GVA_CONFIG.MQTT.Username)
	opts.SetPassword(global.GVA_CONFIG.MQTT.Password)
	opts.SetKeepAlive(time.Duration(global.GVA_CONFIG.MQTT.KeepAlive) * time.Second)
	opts.SetCleanSession(global.GVA_CONFIG.MQTT.CleanSession)

	// 设置连接回调
	opts.SetOnConnectHandler(s.onConnect)
	opts.SetConnectionLostHandler(s.onConnectionLost)
	opts.SetDefaultPublishHandler(s.defaultMessageHandler)

	// 创建客户端
	s.client = mqtt.NewClient(opts)

	// 连接到MQTT Broker
	if token := s.client.Connect(); token.Wait() && token.Error() != nil {
		return fmt.Errorf("连接MQTT失败: %w", token.Error())
	}

	global.GVA_LOG.Info("MQTT服务初始化成功", zap.String("broker", brokerURL))
	return nil
}

// onConnect 连接成功回调
func (s *MqttService) onConnect(client mqtt.Client) {
	s.mu.Lock()
	s.isConnected = true
	s.mu.Unlock()

	global.GVA_LOG.Info("MQTT连接成功")

	// 订阅系统主题
	s.subscribeSystemTopics()
}

// onConnectionLost 连接丢失回调
func (s *MqttService) onConnectionLost(client mqtt.Client, err error) {
	s.mu.Lock()
	s.isConnected = false
	s.mu.Unlock()

	global.GVA_LOG.Error("MQTT连接丢失", zap.Error(err))
}

// defaultMessageHandler 默认消息处理器
func (s *MqttService) defaultMessageHandler(client mqtt.Client, msg mqtt.Message) {
	global.GVA_LOG.Warn("收到未处理的MQTT消息",
		zap.String("topic", msg.Topic()),
		zap.String("payload", string(msg.Payload())),
	)
}

// subscribeSystemTopics 订阅系统主题
func (s *MqttService) subscribeSystemTopics() {
	topicPrefix := global.GVA_CONFIG.MQTT.NFCRelay.TopicPrefix
	qos := global.GVA_CONFIG.MQTT.QoS

	// 订阅客户端心跳主题
	heartbeatTopic := fmt.Sprintf("%s/+/heartbeat", topicPrefix)
	if token := s.client.Subscribe(heartbeatTopic, qos, s.handleHeartbeat); token.Wait() && token.Error() != nil {
		global.GVA_LOG.Error("订阅心跳主题失败", zap.Error(token.Error()))
	}

	// 订阅客户端状态主题
	statusTopic := fmt.Sprintf("%s/+/status", topicPrefix)
	if token := s.client.Subscribe(statusTopic, qos, s.handleClientStatus); token.Wait() && token.Error() != nil {
		global.GVA_LOG.Error("订阅状态主题失败", zap.Error(token.Error()))
	}

	// 订阅APDU响应主题
	apduResponseTopic := fmt.Sprintf("%s/+/apdu/response", topicPrefix)
	if token := s.client.Subscribe(apduResponseTopic, qos, s.handleAPDUResponse); token.Wait() && token.Error() != nil {
		global.GVA_LOG.Error("订阅APDU响应主题失败", zap.Error(token.Error()))
	}

	global.GVA_LOG.Info("MQTT系统主题订阅完成")
}

// handleHeartbeat 处理客户端心跳
func (s *MqttService) handleHeartbeat(client mqtt.Client, msg mqtt.Message) {
	var heartbeat HeartbeatMessage
	if err := json.Unmarshal(msg.Payload(), &heartbeat); err != nil {
		global.GVA_LOG.Error("解析心跳消息失败", zap.Error(err))
		return
	}

	// 更新客户端状态到Redis
	s.updateClientHeartbeat(heartbeat)

	global.GVA_LOG.Debug("收到客户端心跳",
		zap.String("clientID", heartbeat.ClientID),
		zap.String("role", heartbeat.Role),
		zap.String("status", heartbeat.Status),
	)
}

// handleClientStatus 处理客户端状态
func (s *MqttService) handleClientStatus(client mqtt.Client, msg mqtt.Message) {
	var status StatusMessage
	if err := json.Unmarshal(msg.Payload(), &status); err != nil {
		global.GVA_LOG.Error("解析状态消息失败", zap.Error(err))
		return
	}

	global.GVA_LOG.Info("收到客户端状态更新",
		zap.String("clientID", status.ClientID),
		zap.String("status", status.Status),
		zap.String("transactionID", status.TransactionID),
	)

	// 更新交易状态
	if status.TransactionID != "" {
		s.updateTransactionStatusFromClient(status)
	}
}

// handleAPDUResponse 处理APDU响应（完善版本）
func (s *MqttService) handleAPDUResponse(client mqtt.Client, msg mqtt.Message) {
	var apduMsg APDUMessage
	if err := json.Unmarshal(msg.Payload(), &apduMsg); err != nil {
		global.GVA_LOG.Error("解析APDU响应消息失败",
			zap.String("topic", msg.Topic()),
			zap.Error(err))
		return
	}

	// 验证APDU消息
	if apduMsg.TransactionID == "" || apduMsg.APDUHex == "" {
		global.GVA_LOG.Error("APDU消息格式无效",
			zap.String("transactionID", apduMsg.TransactionID),
			zap.String("apduHex", apduMsg.APDUHex))
		return
	}

	// 记录APDU消息到数据库
	go s.saveAPDUMessage(apduMsg, msg.Topic())

	// 转发APDU消息到目标客户端
	go s.forwardAPDUMessage(apduMsg)

	global.GVA_LOG.Info("收到APDU响应消息",
		zap.String("transactionID", apduMsg.TransactionID),
		zap.String("direction", apduMsg.Direction),
		zap.Int("sequenceNumber", apduMsg.SequenceNumber),
		zap.String("messageType", apduMsg.MessageType))
}

// forwardAPDUMessage 转发APDU消息到目标客户端
func (s *MqttService) forwardAPDUMessage(apduMsg APDUMessage) error {
	// 根据方向确定目标客户端
	var targetClientID string

	// 获取交易信息
	var transaction nfc_relay.NFCTransaction
	if err := global.GVA_DB.Where("transaction_id = ?", apduMsg.TransactionID).First(&transaction).Error; err != nil {
		global.GVA_LOG.Error("查询交易信息失败",
			zap.String("transactionID", apduMsg.TransactionID),
			zap.Error(err))
		return fmt.Errorf("查询交易信息失败: %w", err)
	}

	// 确定目标客户端
	if apduMsg.Direction == nfc_relay.DirectionToReceiver {
		targetClientID = transaction.ReceiverClientID
	} else if apduMsg.Direction == nfc_relay.DirectionToTransmitter {
		targetClientID = transaction.TransmitterClientID
	} else {
		return fmt.Errorf("无效的APDU消息方向: %s", apduMsg.Direction)
	}

	// 构建转发消息
	forwardMsg := map[string]interface{}{
		"transaction_id":  apduMsg.TransactionID,
		"sequence_number": apduMsg.SequenceNumber,
		"apdu_hex":        apduMsg.APDUHex,
		"message_type":    apduMsg.MessageType,
		"priority":        apduMsg.Priority,
		"timestamp":       time.Now().Format(time.RFC3339),
		"direction":       apduMsg.Direction,
		"timeout":         apduMsg.Timeout,
	}

	// 发布到目标客户端
	if err := s.publishToClient(targetClientID, "apdu", forwardMsg); err != nil {
		global.GVA_LOG.Error("转发APDU消息失败",
			zap.String("transactionID", apduMsg.TransactionID),
			zap.String("targetClientID", targetClientID),
			zap.String("direction", apduMsg.Direction),
			zap.Error(err))
		return err
	}

	global.GVA_LOG.Info("APDU消息转发成功",
		zap.String("transactionID", apduMsg.TransactionID),
		zap.String("targetClientID", targetClientID),
		zap.String("direction", apduMsg.Direction),
		zap.Int("sequenceNumber", apduMsg.SequenceNumber))

	return nil
}

// PublishTransactionCreated 发布交易创建通知
func (s *MqttService) PublishTransactionCreated(ctx context.Context, transaction *nfc_relay.NFCTransaction) error {
	if !s.IsConnected() {
		return fmt.Errorf("MQTT未连接")
	}

	message := NFCMessage{
		MessageID:     fmt.Sprintf("txn_created_%d", time.Now().UnixNano()),
		TransactionID: transaction.TransactionID,
		MessageType:   "transaction_created",
		Direction:     "server_to_clients",
		Timestamp:     time.Now(),
		Payload: map[string]interface{}{
			"transaction_id":        transaction.TransactionID,
			"transmitter_client_id": transaction.TransmitterClientID,
			"receiver_client_id":    transaction.ReceiverClientID,
			"card_type":             transaction.CardType,
			"description":           transaction.Description,
			"expires_at":            transaction.ExpiresAt,
		},
	}

	// 发送给传卡端
	if err := s.publishToClient(transaction.TransmitterClientID, "transaction/created", message); err != nil {
		global.GVA_LOG.Error("发送交易创建通知到传卡端失败", zap.Error(err))
	}

	// 发送给收卡端
	if err := s.publishToClient(transaction.ReceiverClientID, "transaction/created", message); err != nil {
		global.GVA_LOG.Error("发送交易创建通知到收卡端失败", zap.Error(err))
	}

	return nil
}

// PublishTransactionStatusUpdate 发布交易状态更新
func (s *MqttService) PublishTransactionStatusUpdate(ctx context.Context, transactionID, clientID, newStatus, oldStatus, reason string) error {
	if !s.IsConnected() {
		return fmt.Errorf("MQTT未连接")
	}

	message := NFCMessage{
		MessageID:     fmt.Sprintf("status_update_%d", time.Now().UnixNano()),
		TransactionID: transactionID,
		ClientID:      clientID,
		MessageType:   "status_update",
		Direction:     "server_to_client",
		Timestamp:     time.Now(),
		Payload: map[string]interface{}{
			"transaction_id":  transactionID,
			"new_status":      newStatus,
			"previous_status": oldStatus,
			"reason":          reason,
			"updated_at":      time.Now().Format(time.RFC3339),
		},
	}

	return s.publishToClient(clientID, "transaction/status", message)
}

// PublishTransactionSessionActive 发布交易会话激活通知
func (s *MqttService) PublishTransactionSessionActive(ctx context.Context, transaction *nfc_relay.NFCTransaction) error {
	if !s.IsConnected() {
		return fmt.Errorf("MQTT未连接")
	}

	// 构建激活通知消息
	message := map[string]interface{}{
		"event_type":            "session_active",
		"transaction_id":        transaction.TransactionID,
		"transmitter_client_id": transaction.TransmitterClientID,
		"receiver_client_id":    transaction.ReceiverClientID,
		"timestamp":             time.Now().Unix(),
		"topic_config": map[string]interface{}{
			"transmitter_state_topic":   transaction.TransmitterStateTopic,
			"receiver_state_topic":      transaction.ReceiverStateTopic,
			"apdu_to_transmitter_topic": transaction.APDUToTransmitterTopic,
			"apdu_to_receiver_topic":    transaction.APDUToReceiverTopic,
			"control_topic":             transaction.ControlTopic,
			"heartbeat_topic":           transaction.HeartbeatTopic,
		},
	}

	// 发送给传卡端
	if err := s.publishToTransactionClient(transaction.TransactionID, "transmitter", "session/active", message); err != nil {
		global.GVA_LOG.Error("发送会话激活通知到传卡端失败", zap.Error(err))
	}

	// 发送给收卡端
	if err := s.publishToTransactionClient(transaction.TransactionID, "receiver", "session/active", message); err != nil {
		global.GVA_LOG.Error("发送会话激活通知到收卡端失败", zap.Error(err))
	}

	global.GVA_LOG.Info("交易会话激活通知发送完成",
		zap.String("transactionID", transaction.TransactionID),
		zap.String("transmitterClientID", transaction.TransmitterClientID),
		zap.String("receiverClientID", transaction.ReceiverClientID))

	return nil
}

// publishToTransactionClient 发布消息到交易中的指定角色客户端
func (s *MqttService) publishToTransactionClient(transactionID, role, subtopic string, payload interface{}) error {
	// 使用新的主题结构
	topicPrefix := global.GVA_CONFIG.MQTT.NFCRelay.TopicPrefix
	topic := fmt.Sprintf("%s/transactions/%s/%s/%s", topicPrefix, transactionID, role, subtopic)

	data, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("序列化消息失败: %w", err)
	}

	qos := global.GVA_CONFIG.MQTT.QoS
	token := s.client.Publish(topic, qos, false, data)

	if token.Wait() && token.Error() != nil {
		return fmt.Errorf("发布消息失败: %w", token.Error())
	}

	global.GVA_LOG.Debug("交易客户端MQTT消息发布成功",
		zap.String("topic", topic),
		zap.String("transactionID", transactionID),
		zap.String("role", role),
		zap.String("subtopic", subtopic),
	)

	return nil
}

// SendAPDUToClient 发送APDU消息到客户端
func (s *MqttService) SendAPDUToClient(ctx context.Context, clientID string, apduMsg APDUMessage) error {
	if !s.IsConnected() {
		return fmt.Errorf("MQTT未连接")
	}

	return s.publishToClient(clientID, "apdu/command", apduMsg)
}

// PublishPairingNotification 发布配对通知
// 遵循现有企业级模式：连接检查 + 消息构建 + 发布 + 错误处理 + 日志记录
func (s *MqttService) PublishPairingNotification(ctx context.Context, clientID, notificationType string, payload map[string]interface{}) error {
	s.mu.RLock()
	defer s.mu.RUnlock()
	if !s.isConnected {
		return errors.New("MQTT客户端未连接")
	}

	// 统一使用gin.H进行JSON序列化
	fullPayload := gin.H{
		"message_id":        fmt.Sprintf("pairing_%s_%d", notificationType, time.Now().UnixNano()),
		"notification_type": notificationType,
		"message_type":      "pairing_notification",
		"direction":         "server_to_client",
		"timestamp":         time.Now().UTC().Format(time.RFC3339),
		"payload":           payload,
	}

	err := s.publishToClient(clientID, "events", fullPayload)
	if err != nil {
		global.GVA_LOG.Error("发布配对通知失败",
			zap.String("clientID", clientID),
			zap.String("type", notificationType),
			zap.Error(err),
		)
	}

	return err
}

// NotifySessionSuperseded informs a client that its session has been terminated
// because a new session was initiated from another location.
func (s *MqttService) NotifySessionSuperseded(clientID, reason string) error {
	s.mu.RLock()
	defer s.mu.RUnlock()
	if !s.isConnected {
		return errors.New("MQTT client not connected")
	}

	payload := gin.H{
		"event":     "session_superseded",
		"details":   reason,
		"timestamp": time.Now().UTC().Format(time.RFC3339),
	}

	// Using the existing helper to publish to a specific client's event topic
	// e.g., nfc_relay/clients/{clientID}/events
	err := s.publishToClient(clientID, "events", payload)
	if err != nil {
		global.GVA_LOG.Error("Failed to publish session superseded notification",
			zap.String("clientID", clientID),
			zap.Error(err),
		)
		return err
	}

	global.GVA_LOG.Info("Successfully sent session superseded notification",
		zap.String("clientID", clientID),
		zap.String("reason", reason),
	)
	return nil
}

// PublishPairingStatusUpdate 发布配对状态更新
// 用于配对过程中的状态变更通知（等待、取消、超时等）
func (s *MqttService) PublishPairingStatusUpdate(ctx context.Context, clientID, status, message string) error {
	if !s.IsConnected() {
		return fmt.Errorf("MQTT未连接")
	}

	// 参数验证
	if clientID == "" {
		return fmt.Errorf("客户端ID不能为空")
	}
	if status == "" {
		return fmt.Errorf("状态不能为空")
	}

	// 构建状态更新消息
	statusUpdate := map[string]interface{}{
		"status":    status,
		"message":   message,
		"timestamp": time.Now().Unix(),
		"client_id": clientID,
	}

	// 发布到前端期望的主题格式: nfc_relay/pairing/status_updates/{clientId}
	topicPrefix := global.GVA_CONFIG.MQTT.NFCRelay.TopicPrefix
	topic := fmt.Sprintf("%s/pairing/status_updates/%s", topicPrefix, clientID)

	data, err := json.Marshal(statusUpdate)
	if err != nil {
		global.GVA_LOG.Error("序列化配对状态更新失败",
			zap.String("clientID", clientID),
			zap.String("status", status),
			zap.Error(err))
		return fmt.Errorf("序列化配对状态更新失败: %w", err)
	}

	qos := global.GVA_CONFIG.MQTT.QoS
	token := s.client.Publish(topic, qos, false, data)

	if token.Wait() && token.Error() != nil {
		global.GVA_LOG.Error("发布配对状态更新失败",
			zap.String("topic", topic),
			zap.String("clientID", clientID),
			zap.String("status", status),
			zap.Error(token.Error()))
		return fmt.Errorf("发布配对状态更新失败: %w", token.Error())
	}

	global.GVA_LOG.Debug("配对状态更新发送成功",
		zap.String("topic", topic),
		zap.String("clientID", clientID),
		zap.String("status", status))

	return nil
}

// publishToClient 发布消息到指定客户端
func (s *MqttService) publishToClient(clientID, subtopic string, payload interface{}) error {
	topicPrefix := global.GVA_CONFIG.MQTT.NFCRelay.TopicPrefix
	topic := fmt.Sprintf("%s/%s/%s", topicPrefix, clientID, subtopic)

	data, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("序列化消息失败: %w", err)
	}

	qos := global.GVA_CONFIG.MQTT.QoS
	token := s.client.Publish(topic, qos, false, data)

	if token.Wait() && token.Error() != nil {
		return fmt.Errorf("发布消息失败: %w", token.Error())
	}

	global.GVA_LOG.Debug("MQTT消息发布成功",
		zap.String("topic", topic),
		zap.String("clientID", clientID),
		zap.String("subtopic", subtopic),
	)

	return nil
}

// updateClientHeartbeat 更新客户端心跳（完善版本）
func (s *MqttService) updateClientHeartbeat(heartbeat HeartbeatMessage) {
	if heartbeat.ClientID == "" {
		global.GVA_LOG.Error("心跳消息缺少客户端ID")
		return
	}

	ctx := context.Background()
	key := fmt.Sprintf("client_heartbeat:%s", heartbeat.ClientID)

	// 构建心跳数据
	heartbeatData := map[string]interface{}{
		"client_id":  heartbeat.ClientID,
		"role":       heartbeat.Role,
		"status":     heartbeat.Status,
		"last_seen":  heartbeat.Timestamp.Format(time.RFC3339),
		"version":    heartbeat.Version,
		"updated_at": time.Now().Format(time.RFC3339),
	}

	// 序列化设备信息和功能
	if heartbeat.DeviceInfo != nil {
		if deviceInfoJSON, err := json.Marshal(heartbeat.DeviceInfo); err == nil {
			heartbeatData["device_info"] = string(deviceInfoJSON)
		}
	}

	if len(heartbeat.Capabilities) > 0 {
		if capabilitiesJSON, err := json.Marshal(heartbeat.Capabilities); err == nil {
			heartbeatData["capabilities"] = string(capabilitiesJSON)
		}
	}

	// 使用Pipeline更新心跳信息
	pipe := global.GVA_REDIS.Pipeline()
	pipe.HMSet(ctx, key, heartbeatData)
	pipe.Expire(ctx, key, 120*time.Second) // 2分钟过期

	// 更新客户端在线列表
	onlineKey := fmt.Sprintf("clients_online:%s", heartbeat.Role)
	pipe.SAdd(ctx, onlineKey, heartbeat.ClientID)
	pipe.Expire(ctx, onlineKey, 300*time.Second) // 5分钟过期

	// 更新全局在线客户端集合
	pipe.SAdd(ctx, "clients_online_all", heartbeat.ClientID)
	pipe.Expire(ctx, "clients_online_all", 300*time.Second)

	// 执行Pipeline
	if _, err := pipe.Exec(ctx); err != nil {
		global.GVA_LOG.Error("更新客户端心跳失败",
			zap.String("clientID", heartbeat.ClientID),
			zap.Error(err))
		return
	}

	// 发布客户端状态变更事件
	statusEvent := map[string]interface{}{
		"client_id":  heartbeat.ClientID,
		"role":       heartbeat.Role,
		"status":     heartbeat.Status,
		"event_type": "heartbeat",
		"timestamp":  heartbeat.Timestamp.Format(time.RFC3339),
	}

	if eventJSON, err := json.Marshal(statusEvent); err == nil {
		global.GVA_REDIS.Publish(ctx, "client:status_changed", eventJSON).Err()
	}

	global.GVA_LOG.Debug("客户端心跳更新成功",
		zap.String("clientID", heartbeat.ClientID),
		zap.String("role", heartbeat.Role),
		zap.String("status", heartbeat.Status),
		zap.String("version", heartbeat.Version))
}

// updateTransactionStatusFromClient 从客户端更新交易状态（完善版本）
func (s *MqttService) updateTransactionStatusFromClient(status StatusMessage) {
	if status.TransactionID == "" || status.ClientID == "" {
		global.GVA_LOG.Error("状态消息缺少必要字段",
			zap.String("transactionID", status.TransactionID),
			zap.String("clientID", status.ClientID))
		return
	}

	// 查询交易信息验证权限
	var transaction nfc_relay.NFCTransaction
	if err := global.GVA_DB.Where("transaction_id = ?", status.TransactionID).First(&transaction).Error; err != nil {
		global.GVA_LOG.Error("查询交易信息失败",
			zap.String("transactionID", status.TransactionID),
			zap.Error(err))
		return
	}

	// 验证客户端是否有权限更新此交易状态
	if status.ClientID != transaction.TransmitterClientID &&
		status.ClientID != transaction.ReceiverClientID {
		global.GVA_LOG.Error("客户端无权限更新交易状态",
			zap.String("clientID", status.ClientID),
			zap.String("transactionID", status.TransactionID))
		return
	}

	// 验证状态转换是否有效
	if !nfc_relay.IsValidStatusTransition(transaction.Status, status.Status) {
		global.GVA_LOG.Error("无效的状态转换",
			zap.String("transactionID", status.TransactionID),
			zap.String("currentStatus", transaction.Status),
			zap.String("newStatus", status.Status))
		return
	}

	// 直接使用数据库更新交易状态
	updateMap := map[string]interface{}{
		"status":     status.Status,
		"updated_at": time.Now(),
	}

	// 如果有元数据，也一并更新
	if status.Metadata != nil {
		if metadataJSON, err := json.Marshal(status.Metadata); err == nil {
			updateMap["metadata"] = datatypes.JSON(metadataJSON)
		}
	}

	if err := global.GVA_DB.Model(&nfc_relay.NFCTransaction{}).
		Where("transaction_id = ?", status.TransactionID).
		Updates(updateMap).Error; err != nil {
		global.GVA_LOG.Error("更新交易状态失败",
			zap.String("transactionID", status.TransactionID),
			zap.String("clientID", status.ClientID),
			zap.String("status", status.Status),
			zap.Error(err))
		return
	}

	global.GVA_LOG.Info("客户端更新交易状态成功",
		zap.String("transactionID", status.TransactionID),
		zap.String("clientID", status.ClientID),
		zap.String("status", status.Status))
}

// saveAPDUMessage 保存APDU消息到数据库（完善版本）
func (s *MqttService) saveAPDUMessage(apduMsg APDUMessage, topic string) {
	// 构建APDU消息记录
	apduRecord := &nfc_relay.NFCAPDUMessage{
		TransactionID:  apduMsg.TransactionID,
		Direction:      apduMsg.Direction,
		APDUHex:        apduMsg.APDUHex,
		SequenceNumber: apduMsg.SequenceNumber,
		Priority:       apduMsg.Priority,
		MessageType:    apduMsg.MessageType,
		Status:         nfc_relay.MessageStatusReceived,
		SentAt:         &time.Time{}, // 设置为接收时间
	}

	// 设置接收时间
	now := time.Now()
	apduRecord.ReceivedAt = &now

	// 处理元数据
	if len(apduMsg.APDUHex) > 0 {
		metadata := map[string]interface{}{
			"topic":        topic,
			"received_via": "mqtt",
			"timeout":      apduMsg.Timeout,
		}

		if metadataJSON, err := json.Marshal(metadata); err == nil {
			apduRecord.Metadata = datatypes.JSON(metadataJSON)
		}
	}

	// 保存到数据库
	if err := global.GVA_DB.Create(apduRecord).Error; err != nil {
		global.GVA_LOG.Error("保存APDU消息失败",
			zap.String("transactionID", apduMsg.TransactionID),
			zap.String("direction", apduMsg.Direction),
			zap.Int("sequenceNumber", apduMsg.SequenceNumber),
			zap.Error(err))
		return
	}

	// 更新交易的APDU计数
	global.GVA_DB.Model(&nfc_relay.NFCTransaction{}).
		Where("transaction_id = ?", apduMsg.TransactionID).
		UpdateColumn("apdu_count", gorm.Expr("apdu_count + ?", 1))

	// 缓存到Redis用于快速查询
	s.cacheAPDUMessage(apduMsg)

	global.GVA_LOG.Debug("APDU消息保存成功",
		zap.String("transactionID", apduMsg.TransactionID),
		zap.String("direction", apduMsg.Direction),
		zap.Int("sequenceNumber", apduMsg.SequenceNumber),
		zap.Uint("messageID", apduRecord.ID))
}

// cacheAPDUMessage 缓存APDU消息到Redis
func (s *MqttService) cacheAPDUMessage(apduMsg APDUMessage) {
	ctx := context.Background()
	key := fmt.Sprintf("transaction:%s:apdu_messages", apduMsg.TransactionID)

	// 构建缓存数据
	cacheData := map[string]interface{}{
		"sequence_number": apduMsg.SequenceNumber,
		"direction":       apduMsg.Direction,
		"apdu_hex":        apduMsg.APDUHex,
		"message_type":    apduMsg.MessageType,
		"priority":        apduMsg.Priority,
		"timestamp":       time.Now().Format(time.RFC3339),
	}

	// 使用List存储APDU消息序列
	if cacheJSON, err := json.Marshal(cacheData); err == nil {
		pipe := global.GVA_REDIS.Pipeline()
		pipe.LPush(ctx, key, string(cacheJSON))
		pipe.LTrim(ctx, key, 0, 999)            // 保留最近1000条消息
		pipe.Expire(ctx, key, 3600*time.Second) // 1小时过期
		pipe.Exec(ctx)
	}
}

// IsConnected 检查MQTT连接状态
func (s *MqttService) IsConnected() bool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.isConnected
}

// Disconnect 断开MQTT连接
func (s *MqttService) Disconnect() {
	if s.client != nil && s.client.IsConnected() {
		s.client.Disconnect(250)
		global.GVA_LOG.Info("MQTT客户端已断开")
	}
}

// HandleRoleRequestWebhook 处理角色请求的Webhook
func (s *MqttService) HandleRoleRequestWebhook(c *gin.Context) {
	var req systemReq.MqttAuthRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		global.GVA_LOG.Error("角色请求Webhook绑定参数失败", zap.Error(err))
		c.JSON(400, gin.H{"result": "deny"})
		return
	}

	global.GVA_LOG.Info("收到角色请求Webhook",
		zap.String("clientid", req.ClientID),
		zap.String("username", req.Username),
		zap.String("topic", req.Topic),
		zap.String("action", req.Action),
	)

	// TODO: 根据实际业务逻辑判断角色权限
	// 示例：默认允许所有角色请求
	c.JSON(200, gin.H{"result": "allow"})
}

// HandleConnectionStatusWebhook 处理连接状态的Webhook
func (s *MqttService) HandleConnectionStatusWebhook(c *gin.Context) {
	var req systemReq.MqttConnectionStatusRequest

	// 预读请求体，以便在解析失败或事件未知时能记录原始数据
	bodyBytes, err := io.ReadAll(c.Request.Body)
	if err != nil {
		global.GVA_LOG.Error("读取MQTT Webhook请求体失败", zap.Error(err))
		c.JSON(http.StatusBadRequest, gin.H{"error": "Cannot read request body"})
		return
	}
	// 重新填充请求体，以供ShouldBindJSON使用
	c.Request.Body = io.NopCloser(bytes.NewBuffer(bodyBytes))

	if err := c.ShouldBindJSON(&req); err != nil {
		// 如果解析失败，记录原始请求体以供调试
		global.GVA_LOG.Error("解析MQTT Webhook连接状态请求失败",
			zap.Error(err),
			zap.String("rawBody", string(bodyBytes)),
		)
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request body"})
		return
	}

	global.GVA_LOG.Info("收到MQTT Webhook连接状态事件",
		zap.String("event", req.Event),
		zap.String("clientID", req.ClientID),
		zap.String("username", req.Username),
		zap.String("reason", req.Reason),
	)

	switch req.Event {
	case MQTTEventClientConnected:
		s.handleClientConnected(req)
	case MQTTEventClientDisconnected:
		s.handleClientDisconnected(req)
	default:
		// 增强日志: 对于未知或空事件，记录为错误级别并包含原始请求体
		global.GVA_LOG.Error("收到无法处理的Webhook事件类型",
			zap.String("event", req.Event),
			zap.String("clientID", req.ClientID),
			zap.String("rawBody", string(bodyBytes)),
		)
		// 返回一个明确的错误码，而不是200 OK
		c.JSON(http.StatusUnprocessableEntity, gin.H{"status": "error", "message": "unknown event"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"status": "ok"})
}

// handleClientConnected 处理客户端连接事件
func (s *MqttService) handleClientConnected(req systemReq.MqttConnectionStatusRequest) {
	ctx := context.Background()
	global.GVA_LOG.Info("[Webhook] 开始处理客户端连接事件",
		zap.String("clientID", req.ClientID),
		zap.String("username", req.Username))

	// 从Redis获取角色信息
	role, err := s.getClientRoleFromRedis(req.ClientID)
	if err != nil {
		global.GVA_LOG.Error("无法从Redis获取客户端角色", zap.Error(err), zap.String("clientID", req.ClientID))
		role = "" // 继续执行，某些客户端可能没有角色
	}

	// 更新Redis中的客户端在线状态
	s.updateClientOnlineStatus(req.ClientID, role, true)

	// 根据角色执行特定逻辑
	switch role {
	case RoleReceiver:
		s.handleReceiverConnected(ctx, req.ClientID)
	case RoleTransmitter:
		s.handleTransmitterConnected(ctx, req.ClientID)
	default:
		global.GVA_LOG.Info("客户端已连接，但没有特定角色或角色未知",
			zap.String("clientID", req.ClientID),
			zap.String("role", role))
	}
}

// handleReceiverConnected 处理 receiver 客户端连接
func (s *MqttService) handleReceiverConnected(ctx context.Context, clientID string) {
	global.GVA_LOG.Info("Receiver客户端已连接，查找待分配的交易", zap.String("clientID", clientID))

	// 查找状态为 "pending" 且还没有分配 receiver 的交易
	var pendingTransactions []nfc_relay.NFCTransaction
	if err := global.GVA_DB.Where("status = ? AND receiver_client_id = ?", "pending", "").
		Order("created_at ASC").
		Limit(10). // 限制查询数量，避免性能问题
		Find(&pendingTransactions).Error; err != nil {
		global.GVA_LOG.Error("查询待分配交易失败", zap.Error(err), zap.String("clientID", clientID))
		return
	}

	if len(pendingTransactions) == 0 {
		global.GVA_LOG.Info("没有找到待分配的交易", zap.String("clientID", clientID))
		return
	}

	// 选择第一个待分配的交易
	transaction := pendingTransactions[0]

	// 更新交易，分配 receiver
	if err := global.GVA_DB.Model(&transaction).Updates(map[string]interface{}{
		"receiver_client_id": clientID,
		"status":             "active",
		"updated_at":         time.Now(),
	}).Error; err != nil {
		global.GVA_LOG.Error("分配Receiver到交易失败", zap.Error(err),
			zap.String("clientID", clientID),
			zap.String("transactionID", transaction.TransactionID))
		return
	}

	global.GVA_LOG.Info("成功分配Receiver到交易",
		zap.String("clientID", clientID),
		zap.String("transactionID", transaction.TransactionID))

	// 通知 transmitter 客户端 receiver 已就绪
	go s.notifyTransmitterReceiverReady(ctx, transaction.TransmitterClientID, clientID, transaction.TransactionID)

	// 通知 receiver 客户端已被分配到交易
	go s.notifyReceiverAssignedToTransaction(ctx, clientID, transaction.TransactionID)
}

// handleTransmitterConnected 处理 transmitter 客户端连接
func (s *MqttService) handleTransmitterConnected(ctx context.Context, clientID string) {
	global.GVA_LOG.Info("Transmitter客户端已连接", zap.String("clientID", clientID))

	// 检查是否有该 transmitter 的活跃交易
	var activeTransaction nfc_relay.NFCTransaction
	if err := global.GVA_DB.Where("transmitter_client_id = ? AND status IN (?)", clientID, []string{"pending", "active"}).
		First(&activeTransaction).Error; err != nil {
		if err != gorm.ErrRecordNotFound {
			global.GVA_LOG.Error("查询Transmitter活跃交易失败", zap.Error(err), zap.String("clientID", clientID))
		}
		return
	}

	// 如果有活跃交易，通知 transmitter 恢复交易状态
	global.GVA_LOG.Info("发现Transmitter的活跃交易",
		zap.String("clientID", clientID),
		zap.String("transactionID", activeTransaction.TransactionID))

	go s.notifyTransmitterResumeTransaction(ctx, clientID, activeTransaction.TransactionID, activeTransaction.Status)
}

// handleClientDisconnected 处理客户端断开
func (s *MqttService) handleClientDisconnected(req systemReq.MqttConnectionStatusRequest) {
	ctx := context.Background()
	clientID := req.ClientID
	username := req.Username // username is userID
	userID, _ := strconv.ParseUint(username, 10, 64)

	global.GVA_LOG.Info("[Webhook] 开始处理客户端断开事件", zap.String("clientID", clientID), zap.String("username", username), zap.String("reason", req.Reason))

	// 从clientID解析角色
	role, err := getRoleFromClientID(clientID)
	if err != nil {
		global.GVA_LOG.Warn("无法从clientID解析角色，跳过状态清理", zap.String("clientID", clientID), zap.Error(err))
	} else if userID > 0 {
		// 调用我们新的服务来安全地注销设备
		err := s.DeregisterDevice(uint(userID), role, clientID)
		if err != nil {
			global.GVA_LOG.Error("注销设备状态失败", zap.Error(err), zap.String("clientID", clientID))
		} else {
			global.GVA_LOG.Info("成功从配对池中移除下线设备", zap.String("clientID", clientID), zap.String("role", role), zap.Uint64("userID", userID))
		}
	}

	// 清理客户端在Redis中的所有相关数据
	s.cleanupClientRedisData(ctx, clientID)

	// 处理交易相关的清理逻辑
	s.handleTransactionCleanupOnDisconnect(ctx, clientID, req.Reason)
}

// cleanupClientRedisData removes all related data for a client from Redis upon disconnection.
// It uses the reverse reference key to find and delete all associated data atomically.
func (s *MqttService) cleanupClientRedisData(ctx context.Context, clientID string) {
	refKey := "mqtt:client_ref:" + clientID

	// 1. Get the reference JSON string
	refDataJSON, err := global.GVA_REDIS.Get(ctx, refKey).Result()
	if err != nil {
		if errors.Is(err, redis.Nil) {
			// This can happen if the key expired or was already deleted. Not a critical error.
			global.GVA_LOG.Warn("MQTT客户端引用键未找到，可能已被清理",
				zap.String("refKey", refKey),
				zap.String("clientID", clientID))
		} else {
			global.GVA_LOG.Error("获取MQTT客户端引用数据失败",
				zap.Error(err),
				zap.String("refKey", refKey),
				zap.String("clientID", clientID))
		}
		// In either case, we cannot proceed with this key.
		return
	}

	// 2. Unmarshal the reference data
	var refData struct {
		ActiveKey string `json:"activeKey"`
		RoleKey   string `json:"roleKey"`
	}
	if err := json.Unmarshal([]byte(refDataJSON), &refData); err != nil {
		global.GVA_LOG.Error("解析MQTT客户端引用数据失败",
			zap.Error(err),
			zap.String("refKey", refKey),
			zap.String("rawData", refDataJSON))
		// If we can't parse, we should still try to delete the refKey itself.
		_ = global.GVA_REDIS.Del(ctx, refKey).Err()
		return
	}

	// 3. Collect all keys to be deleted
	keysToDelete := []string{refKey}
	if refData.ActiveKey != "" {
		keysToDelete = append(keysToDelete, refData.ActiveKey)
	}
	if refData.RoleKey != "" {
		keysToDelete = append(keysToDelete, refData.RoleKey)
	}

	// 4. Delete all keys in a single atomic operation
	if len(keysToDelete) > 0 {
		deletedCount, err := global.GVA_REDIS.Del(ctx, keysToDelete...).Result()
		if err != nil {
			global.GVA_LOG.Error("清理MQTT客户端Redis数据失败",
				zap.Error(err),
				zap.Strings("keys", keysToDelete))
		} else {
			global.GVA_LOG.Info("成功清理MQTT客户端Redis数据",
				zap.String("clientID", clientID),
				zap.Int64("deletedKeysCount", deletedCount),
				zap.Strings("deletedKeys", keysToDelete))
		}
	}
}

// handleTransactionCleanupOnDisconnect handles the cleanup of transaction states when a client disconnects.
func (s *MqttService) handleTransactionCleanupOnDisconnect(ctx context.Context, clientID string, reason string) {
	// Find active transactions involving this client
	var activeTransactions []nfc_relay.NFCTransaction
	err := global.GVA_DB.Where(
		"(transmitter_client_id = ? OR receiver_client_id = ?) AND status IN (?)",
		clientID, clientID, []string{nfc_relay.StatusPending, nfc_relay.StatusActive, nfc_relay.StatusProcessing},
	).Find(&activeTransactions).Error

	if err != nil {
		global.GVA_LOG.Error("查询客户端相关交易失败",
			zap.String("clientID", clientID),
			zap.Error(err))
		return
	}

	if len(activeTransactions) == 0 {
		global.GVA_LOG.Info("断开连接的客户端与任何活跃交易无关",
			zap.String("clientID", clientID))
		return
	}

	// Process each affected transaction
	for _, transaction := range activeTransactions {
		global.GVA_LOG.Info("处理断开连接对交易的影响",
			zap.String("clientID", clientID),
			zap.String("transactionID", transaction.TransactionID),
			zap.String("currentStatus", transaction.Status),
			zap.String("reason", reason))

		// 根据断开的客户端类型决定处理策略
		if transaction.TransmitterClientID == clientID {
			// Transmitter断开：暂停交易
			s.pauseTransactionOnTransmitterDisconnect(ctx, transaction, reason)
		} else if transaction.ReceiverClientID == clientID {
			// Receiver 断开：尝试重新分配或暂停交易
			s.handleReceiverDisconnect(ctx, transaction, reason)
		}
	}
}

// pauseTransactionOnTransmitterDisconnect pauses the transaction when the transmitter disconnects.
func (s *MqttService) pauseTransactionOnTransmitterDisconnect(ctx context.Context, transaction nfc_relay.NFCTransaction, reason string) {
	// 更新交易状态为暂停
	if err := global.GVA_DB.Model(&transaction).Updates(map[string]interface{}{
		"status":     "paused",
		"updated_at": time.Now(),
	}).Error; err != nil {
		global.GVA_LOG.Error("暂停交易失败", zap.Error(err), zap.String("transactionID", transaction.TransactionID))
		return
	}

	global.GVA_LOG.Info("由于Transmitter断开连接，交易已暂停",
		zap.String("transactionID", transaction.TransactionID),
		zap.String("transmitterClientID", transaction.TransmitterClientID),
		zap.String("reason", reason))

	// 通知 receiver（如果在线）transmitter 已断开
	if transaction.ReceiverClientID != "" {
		go s.notifyReceiverTransmitterDisconnected(ctx, transaction.ReceiverClientID, transaction.TransactionID, reason)
	}
}

// handleReceiverDisconnect handles the logic when a receiver client disconnects.
func (s *MqttService) handleReceiverDisconnect(ctx context.Context, transaction nfc_relay.NFCTransaction, reason string) {
	global.GVA_LOG.Info("接收端已断开，开始处理交易",
		zap.String("transactionID", transaction.TransactionID),
		zap.String("receiverClientID", transaction.ReceiverClientID))

	// 1. 将交易状态更新为 "paused"
	updateMap := map[string]interface{}{
		"status":     nfc_relay.StatusPending, // 使用常量而不是硬编码字符串
		"updated_at": time.Now(),
		"end_reason": "Receiver disconnected: " + reason,
	}

	err := global.GVA_DB.Model(&nfc_relay.NFCTransaction{}).
		Where("transaction_id = ? AND status = ?", transaction.TransactionID, nfc_relay.StatusActive).
		Updates(updateMap).Error
	if err != nil {
		global.GVA_LOG.Error("更新交易状态为 paused 失败",
			zap.Error(err),
			zap.String("transactionID", transaction.TransactionID))
		return
	}
	global.GVA_LOG.Info("交易已暂停", zap.String("transactionID", transaction.TransactionID))

	// 2. 通知发送端，接收端已断开连接
	s.notifyTransmitterReceiverDisconnected(ctx, transaction.TransmitterClientID, transaction.TransactionID, reason)

	// 3. (可选) 可以在这里触发一个逻辑，去寻找一个新的可用的接收端
	//    这部分逻辑可以后续实现
	global.GVA_LOG.Info("已通知发送端，接收端断开", zap.String("transactionID", transaction.TransactionID))
}

// notifyTransmitterReceiverReady notifies the transmitter that a receiver is ready.
func (s *MqttService) notifyTransmitterReceiverReady(ctx context.Context, transmitterClientID, receiverClientID, transactionID string) {
	message := map[string]interface{}{
		"event_type":         "receiver_ready",
		"transaction_id":     transactionID,
		"receiver_client_id": receiverClientID,
		"timestamp":          time.Now().Unix(),
	}

	if err := s.publishToClient(transmitterClientID, "transaction/receiver_ready", message); err != nil {
		global.GVA_LOG.Error("通知Transmitter Receiver就绪失败", zap.Error(err),
			zap.String("transmitterClientID", transmitterClientID))
	}
}

func (s *MqttService) notifyReceiverAssignedToTransaction(ctx context.Context, receiverClientID, transactionID string) {
	message := map[string]interface{}{
		"event_type":     "assigned_to_transaction",
		"transaction_id": transactionID,
		"timestamp":      time.Now().Unix(),
	}

	if err := s.publishToClient(receiverClientID, "transaction/assigned", message); err != nil {
		global.GVA_LOG.Error("通知Receiver交易分配失败", zap.Error(err),
			zap.String("receiverClientID", receiverClientID))
	}
}

func (s *MqttService) notifyTransmitterResumeTransaction(ctx context.Context, transmitterClientID, transactionID, status string) {
	message := map[string]interface{}{
		"event_type":     "resume_transaction",
		"transaction_id": transactionID,
		"status":         status,
		"timestamp":      time.Now().Unix(),
	}

	if err := s.publishToClient(transmitterClientID, "transaction/resume", message); err != nil {
		global.GVA_LOG.Error("通知Transmitter恢复交易失败", zap.Error(err),
			zap.String("transmitterClientID", transmitterClientID))
	}
}

func (s *MqttService) notifyReceiverTransmitterDisconnected(ctx context.Context, receiverClientID, transactionID, reason string) {
	message := map[string]interface{}{
		"event_type":     "transmitter_disconnected",
		"transaction_id": transactionID,
		"reason":         reason,
		"timestamp":      time.Now().Unix(),
	}

	if err := s.publishToClient(receiverClientID, "transaction/transmitter_disconnected", message); err != nil {
		global.GVA_LOG.Error("通知Receiver Transmitter断开失败", zap.Error(err),
			zap.String("receiverClientID", receiverClientID))
	}
}

func (s *MqttService) notifyTransmitterReceiverDisconnected(ctx context.Context, transmitterClientID, transactionID, reason string) {
	message := map[string]interface{}{
		"event_type":     "receiver_disconnected",
		"transaction_id": transactionID,
		"reason":         reason,
		"timestamp":      time.Now().Unix(),
	}

	if err := s.publishToClient(transmitterClientID, "transaction/receiver_disconnected", message); err != nil {
		global.GVA_LOG.Error("通知Transmitter Receiver断开失败", zap.Error(err),
			zap.String("transmitterClientID", transmitterClientID))
	}
}

// getClientRoleFromRedis 从Redis获取客户端的角色
// [企业级修复] 修正了此函数的逻辑，使其与 'StoreMQTTRoleByClaims' 的存储逻辑完全匹配。
func (s *MqttService) getClientRoleFromRedis(clientID string) (string, error) {
	if clientID == "" {
		return "", errors.New("clientID cannot be empty for role lookup")
	}
	ctx := context.Background()
	// 使用 'GET' 命令从正确的键 'mqtt:role:<clientID>' 获取角色。
	redisKey := common.RedisMqttRoleKeyPrefix + clientID
	role, err := global.GVA_REDIS.Get(ctx, redisKey).Result()
	if err != nil {
		if errors.Is(err, redis.Nil) {
			global.GVA_LOG.Warn("Role for clientID not found in Redis",
				zap.String("clientID", clientID),
				zap.String("redisKey", redisKey))
			return "", fmt.Errorf("role for client %s not found", clientID)
		}
		global.GVA_LOG.Error("Failed to get role for clientID from Redis",
			zap.Error(err),
			zap.String("clientID", clientID),
			zap.String("redisKey", redisKey))
		return "", err
	}
	return role, nil
}

// updateClientOnlineStatus 更新客户端在线状态
func (s *MqttService) updateClientOnlineStatus(clientID, role string, isOnline bool) {
	ctx := context.Background()
	statusKey := fmt.Sprintf("client_status:%s", clientID)
	statusData := map[string]interface{}{
		"is_online":   isOnline,
		"role":        role,
		"last_change": time.Now().Unix(),
	}
	if err := global.GVA_REDIS.HSet(ctx, statusKey, statusData).Err(); err != nil {
		global.GVA_LOG.Error("更新客户端在线状态失败", zap.Error(err), zap.String("clientID", clientID))
	}
	// 为状态设置过期时间，防止离线客户端信息永久留存
	global.GVA_REDIS.Expire(ctx, statusKey, 24*time.Hour)
}

// PublishToClient 公开的发布消息到客户端方法（供其他服务调用）
func (s *MqttService) PublishToClient(clientID, subtopic string, payload interface{}) error {
	return s.publishToClient(clientID, subtopic, payload)
}

// checkClientOnlineViaAPI 通过EMQX的管理API实时检查客户端是否在线
// 这是获取客户端在线状态的最可靠方法
func (s *MqttService) checkClientOnlineViaAPI(ctx context.Context, clientID string) (bool, error) {
	cfg := global.GVA_CONFIG.MQTT.API
	if cfg.Host == "" || cfg.Port == 0 || cfg.Username == "" {
		global.GVA_LOG.Warn("EMQX API配置不完整，无法进行实时在线状态检查")
		// 在此返回错误，让调用者决定如何处理（例如，回退到Redis检查）
		return false, errors.New("EMQX API配置不完整")
	}

	// 构建请求URL
	protocol := "http"
	if cfg.UseTLS {
		protocol = "https"
	}
	url := fmt.Sprintf("%s://%s:%d/api/v5/clients/%s", protocol, cfg.Host, cfg.Port, clientID)

	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		global.GVA_LOG.Error("创建EMQX API请求失败", zap.Error(err), zap.String("url", url))
		return false, fmt.Errorf("创建API请求失败: %w", err)
	}

	// 添加Basic Auth认证头
	req.SetBasicAuth(cfg.Username, cfg.Password)

	// 使用可配置超时的HTTP客户端
	httpClient := &http.Client{
		Timeout: 5 * time.Second, // 设置合理的超时时间
	}

	resp, err := httpClient.Do(req)
	if err != nil {
		global.GVA_LOG.Error("调用EMQX API失败", zap.Error(err), zap.String("url", url))
		return false, fmt.Errorf("调用EMQX API失败: %w", err)
	}
	defer resp.Body.Close()

	// 分析响应
	switch resp.StatusCode {
	case http.StatusOK:
		// 200 OK: 客户端在线
		global.GVA_LOG.Debug("EMQX API check: client is online", zap.String("clientID", clientID))
		return true, nil
	case http.StatusNotFound:
		// 404 Not Found: 客户端离线
		global.GVA_LOG.Info("EMQX API check: client is offline", zap.String("clientID", clientID))
		return false, nil
	default:
		// 其他状态码表示出现问题
		bodyBytes, _ := io.ReadAll(resp.Body)
		global.GVA_LOG.Error("EMQX API返回非预期状态码",
			zap.Int("status_code", resp.StatusCode),
			zap.String("clientID", clientID),
			zap.String("response_body", string(bodyBytes)),
		)
		return false, fmt.Errorf("EMQX API返回错误, 状态码: %d", resp.StatusCode)
	}
}

// getPairingStateKey generates the Redis key for storing the pairing state HASH of a user.
func (mqttService *MqttService) getPairingStateKey(userID uint) string {
	return fmt.Sprintf("pairing:state:%d", userID)
}

// CheckRoleConflict checks if a role for a given user is already occupied by another client.
// It returns the existing clientID if a conflict is found, otherwise returns an empty string.
func (mqttService *MqttService) CheckRoleConflict(userID uint, role string) (string, error) {
	key := mqttService.getPairingStateKey(userID)
	existingClientID, err := global.GVA_REDIS.HGet(context.Background(), key, role).Result()
	if err == redis.Nil {
		// No conflict found, this is the expected "not found" error.
		return "", nil
	}
	if err != nil {
		// A real error occurred.
		return "", err
	}
	// Conflict found.
	return existingClientID, nil
}

// RegisterDevice registers a new device for a given role, effectively updating the pairing state.
// This will overwrite any existing clientID for the same role.
func (mqttService *MqttService) RegisterDevice(userID uint, role string, clientID string) error {
	key := mqttService.getPairingStateKey(userID)
	return global.GVA_REDIS.HSet(context.Background(), key, role, clientID).Err()
}

// DeregisterDevice removes a device from the pairing state.
// It uses a Redis transaction to atomically check and delete, ensuring that it only removes the device
// if the provided clientID matches the one currently stored. This prevents a stale disconnect message
// from accidentally removing a newly logged-in device.
func (mqttService *MqttService) DeregisterDevice(userID uint, role string, clientID string) error {
	key := mqttService.getPairingStateKey(userID)
	ctx := context.Background()

	// Use WATCH for an optimistic lock to ensure atomicity.
	return global.GVA_REDIS.Watch(ctx, func(tx *redis.Tx) error {
		currentClientID, err := tx.HGet(ctx, key, role).Result()
		if err == redis.Nil {
			// Role is already empty, nothing to do.
			return nil
		}
		if err != nil {
			return err
		}

		// Only proceed if the clientID from the disconnect event matches the one in Redis.
		if currentClientID == clientID {
			// The key hasn't been changed by another process, so we can safely delete it.
			_, err = tx.TxPipelined(ctx, func(pipe redis.Pipeliner) error {
				pipe.HDel(ctx, key, role)
				return nil
			})
			return err
		}

		// If clientIDs do not match, it means a new client has already registered for this role.
		// We do nothing to avoid incorrectly removing the new session.
		return nil
	}, key)
}

// FindPeerForPairing finds an available peer for a given user and their current role.
func (mqttService *MqttService) FindPeerForPairing(userID uint, currentRole string) (string, error) {
	var targetRole string
	if currentRole == "transmitter" {
		targetRole = "receiver"
	} else if currentRole == "receiver" {
		targetRole = "transmitter"
	} else {
		return "", errors.New("invalid role for pairing")
	}

	key := mqttService.getPairingStateKey(userID)
	peerClientID, err := global.GVA_REDIS.HGet(context.Background(), key, targetRole).Result()
	if err == redis.Nil {
		// No peer found for the target role. This is not an error, just no one is available.
		return "", nil
	}
	return peerClientID, err
}

// Publish sends a message to a specific MQTT topic.
// This is a generic publish method that can be used by other services.
func (mqttService *MqttService) Publish(topic string, qos byte, retained bool, payload interface{}) error {
	s := GetMQTTService() // Get the singleton instance
	if !s.IsConnected() {
		return errors.New("mqtt client is not connected")
	}

	token := s.client.Publish(topic, qos, retained, payload)
	// We can wait for a short period to see if the publish was successful.
	// This is optional and depends on whether we need to ensure the message is sent.
	if token.WaitTimeout(2*time.Second) && token.Error() != nil {
		global.GVA_LOG.Error("Failed to publish MQTT message", zap.String("topic", topic), zap.Error(token.Error()))
		return token.Error()
	}
	return nil
}

// getRoleFromClientID extracts the role from a clientID string.
// Example: "admin-transmitter-007" -> "transmitter"
func getRoleFromClientID(clientID string) (string, error) {
	parts := strings.Split(clientID, "-")
	if len(parts) < 2 {
		return "", errors.New("invalid clientID format: not enough parts")
	}
	// The role is expected to be the second-to-last part.
	// This is more robust than assuming a fixed position.
	role := parts[len(parts)-2]
	if role == RoleReceiver || role == RoleTransmitter {
		return role, nil
	}

	return "", fmt.Errorf("unknown role '%s' in clientID", role)
}
