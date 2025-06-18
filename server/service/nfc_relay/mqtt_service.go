package nfc_relay

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"
	"time"

	"github.com/gin-gonic/gin"

	mqtt "github.com/eclipse/paho.mqtt.golang"
	"github.com/flipped-aurora/gin-vue-admin/server/global"
	"github.com/flipped-aurora/gin-vue-admin/server/model/nfc_relay"
	"github.com/flipped-aurora/gin-vue-admin/server/model/nfc_relay/request"
	systemReq "github.com/flipped-aurora/gin-vue-admin/server/model/system/request"
	"go.uber.org/zap"
	"gorm.io/datatypes"
	"gorm.io/gorm"
)

type MQTTService struct {
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

var mqttService *MQTTService
var mqttOnce sync.Once

// GetMQTTService 获取MQTT服务实例（单例模式）
func GetMQTTService() *MQTTService {
	mqttOnce.Do(func() {
		mqttService = &MQTTService{}
	})
	return mqttService
}

// Initialize 初始化MQTT连接
func (s *MQTTService) Initialize() error {
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
func (s *MQTTService) onConnect(client mqtt.Client) {
	s.mu.Lock()
	s.isConnected = true
	s.mu.Unlock()

	global.GVA_LOG.Info("MQTT连接成功")

	// 订阅系统主题
	s.subscribeSystemTopics()
}

// onConnectionLost 连接丢失回调
func (s *MQTTService) onConnectionLost(client mqtt.Client, err error) {
	s.mu.Lock()
	s.isConnected = false
	s.mu.Unlock()

	global.GVA_LOG.Error("MQTT连接丢失", zap.Error(err))
}

// defaultMessageHandler 默认消息处理器
func (s *MQTTService) defaultMessageHandler(client mqtt.Client, msg mqtt.Message) {
	global.GVA_LOG.Warn("收到未处理的MQTT消息",
		zap.String("topic", msg.Topic()),
		zap.String("payload", string(msg.Payload())),
	)
}

// subscribeSystemTopics 订阅系统主题
func (s *MQTTService) subscribeSystemTopics() {
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
func (s *MQTTService) handleHeartbeat(client mqtt.Client, msg mqtt.Message) {
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
func (s *MQTTService) handleClientStatus(client mqtt.Client, msg mqtt.Message) {
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
func (s *MQTTService) handleAPDUResponse(client mqtt.Client, msg mqtt.Message) {
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
func (s *MQTTService) forwardAPDUMessage(apduMsg APDUMessage) error {
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
func (s *MQTTService) PublishTransactionCreated(ctx context.Context, transaction *nfc_relay.NFCTransaction) error {
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
func (s *MQTTService) PublishTransactionStatusUpdate(ctx context.Context, transactionID, clientID, newStatus, oldStatus, reason string) error {
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

// SendAPDUToClient 发送APDU消息到客户端
func (s *MQTTService) SendAPDUToClient(ctx context.Context, clientID string, apduMsg APDUMessage) error {
	if !s.IsConnected() {
		return fmt.Errorf("MQTT未连接")
	}

	return s.publishToClient(clientID, "apdu/command", apduMsg)
}

// publishToClient 发布消息到指定客户端
func (s *MQTTService) publishToClient(clientID, subtopic string, payload interface{}) error {
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
func (s *MQTTService) updateClientHeartbeat(heartbeat HeartbeatMessage) {
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
func (s *MQTTService) updateTransactionStatusFromClient(status StatusMessage) {
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

	// 使用交易服务更新状态
	ctx := context.Background()
	transactionService := &NFCTransactionService{}

	updateReq := &request.UpdateTransactionStatusRequest{
		TransactionID: status.TransactionID,
		Status:        status.Status,
		Reason:        fmt.Sprintf("客户端 %s 更新", status.ClientID),
		Metadata:      status.Metadata,
	}

	// 假设使用系统用户ID进行更新
	systemUserID := transaction.CreatedBy // 使用交易创建者ID

	if _, err := transactionService.UpdateTransactionStatus(ctx, updateReq, systemUserID); err != nil {
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
func (s *MQTTService) saveAPDUMessage(apduMsg APDUMessage, topic string) {
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
func (s *MQTTService) cacheAPDUMessage(apduMsg APDUMessage) {
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

// IsConnected 检查MQTT是否连接
func (s *MQTTService) IsConnected() bool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.isConnected && s.client != nil && s.client.IsConnected()
}

// Disconnect 断开MQTT连接
func (s *MQTTService) Disconnect() {
	if s.client != nil && s.client.IsConnected() {
		s.client.Disconnect(250)
		global.GVA_LOG.Info("MQTT客户端已断开")
	}
}

// HandleRoleRequestWebhook 处理角色请求的Webhook
func (s *MQTTService) HandleRoleRequestWebhook(c *gin.Context) {
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
func (s *MQTTService) HandleConnectionStatusWebhook(c *gin.Context) {
	var payload map[string]interface{}
	if err := c.ShouldBindJSON(&payload); err != nil {
		global.GVA_LOG.Error("连接状态Webhook绑定参数失败", zap.Error(err))
		c.Status(400)
		return
	}

	global.GVA_LOG.Info("收到连接状态Webhook", zap.Any("payload", payload))

	// TODO: 可根据连接状态（如断开连接）执行清理任务

	c.Status(200)
}
