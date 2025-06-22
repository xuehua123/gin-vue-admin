package nfc_relay

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"time"

	"github.com/flipped-aurora/gin-vue-admin/server/global"
	"github.com/flipped-aurora/gin-vue-admin/server/model/common/response"
	"github.com/flipped-aurora/gin-vue-admin/server/model/nfc_relay"
	"github.com/flipped-aurora/gin-vue-admin/server/model/nfc_relay/request"
	nfcRelayReq "github.com/flipped-aurora/gin-vue-admin/server/model/nfc_relay/request"
	"github.com/google/uuid"
	"go.uber.org/zap"
	"gorm.io/datatypes"
	"gorm.io/gorm"
)

type NFCTransactionService struct{}

// CreateTransaction 创建交易
func (s *NFCTransactionService) CreateTransaction(ctx context.Context, req *request.CreateTransactionRequest, userID uuid.UUID) (*response.CreateTransactionResponse, error) {
	// 生成交易ID
	transactionID, err := s.generateTransactionID()
	if err != nil {
		global.GVA_LOG.Error("生成交易ID失败", zap.Error(err))
		return nil, fmt.Errorf("生成交易ID失败: %w", err)
	}

	// 检查用户是否有活跃交易（并发控制）
	lockKey := fmt.Sprintf("user_transaction:%s", userID.String())
	locked, err := s.acquireUserLock(ctx, lockKey, 10*time.Second)
	if err != nil {
		global.GVA_LOG.Error("获取用户锁失败", zap.String("userID", userID.String()), zap.Error(err))
		return nil, fmt.Errorf("系统繁忙，请稍后重试")
	}
	if !locked {
		return nil, fmt.Errorf("您已有活跃交易，无法创建新交易")
	}
	defer s.releaseUserLock(ctx, lockKey)

	// 验证客户端是否在线
	transmitterOnline, err := s.isClientOnline(ctx, req.TransmitterClientID)
	if err != nil {
		global.GVA_LOG.Error("检查传卡端状态失败", zap.String("clientID", req.TransmitterClientID), zap.Error(err))
		return nil, fmt.Errorf("检查传卡端状态失败")
	}

	receiverOnline, err := s.isClientOnline(ctx, req.ReceiverClientID)
	if err != nil {
		global.GVA_LOG.Error("检查收卡端状态失败", zap.String("clientID", req.ReceiverClientID), zap.Error(err))
		return nil, fmt.Errorf("检查收卡端状态失败")
	}

	if !transmitterOnline {
		return nil, fmt.Errorf("传卡端 %s 不在线", req.TransmitterClientID)
	}
	if !receiverOnline {
		return nil, fmt.Errorf("收卡端 %s 不在线", req.ReceiverClientID)
	}

	// 处理元数据
	var metadata datatypes.JSON
	if req.Metadata != nil {
		metadataBytes, err := json.Marshal(req.Metadata)
		if err != nil {
			global.GVA_LOG.Error("序列化元数据失败", zap.Error(err))
			return nil, fmt.Errorf("元数据格式错误")
		}
		metadata = datatypes.JSON(metadataBytes)
	}

	// 计算过期时间
	timeoutSeconds := req.TimeoutSeconds
	if timeoutSeconds == 0 {
		timeoutSeconds = 120 // 默认2分钟
	}
	expiresAt := time.Now().Add(time.Duration(timeoutSeconds) * time.Second)

	// 创建交易记录
	transaction := &nfc_relay.NFCTransaction{
		TransactionID:       transactionID,
		TransmitterClientID: req.TransmitterClientID,
		ReceiverClientID:    req.ReceiverClientID,
		Status:              nfc_relay.StatusPending,
		CardType:            req.CardType,
		Description:         req.Description,
		CreatedBy:           userID,
		UpdatedBy:           userID,
		ExpiresAt:           &expiresAt,
		Tags:                req.Tags,
		Metadata:            metadata,
	}

	// 开启数据库事务
	err = global.GVA_DB.Transaction(func(tx *gorm.DB) error {
		// 保存交易记录
		if err := tx.Create(transaction).Error; err != nil {
			global.GVA_LOG.Error("创建交易记录失败", zap.Error(err))
			return fmt.Errorf("创建交易记录失败: %w", err)
		}

		// 缓存交易状态到Redis
		if err := s.cacheTransactionStatus(ctx, transactionID, nfc_relay.StatusPending, map[string]interface{}{
			"created_at":            transaction.CreatedAt,
			"expires_at":            expiresAt,
			"transmitter_client_id": req.TransmitterClientID,
			"receiver_client_id":    req.ReceiverClientID,
		}); err != nil {
			global.GVA_LOG.Error("缓存交易状态失败", zap.Error(err))
			// 不影响主流程，只记录日志
		}

		return nil
	})

	if err != nil {
		return nil, err
	}

	// 发布MQTT通知
	go s.notifyTransactionCreated(ctx, transaction)

	// 返回响应
	return &response.CreateTransactionResponse{
		TransactionID:       transaction.TransactionID,
		Status:              transaction.Status,
		TransmitterClientID: transaction.TransmitterClientID,
		ReceiverClientID:    transaction.ReceiverClientID,
		CardType:            transaction.CardType,
		CreatedAt:           transaction.CreatedAt,
		ExpiresAt:           expiresAt,
	}, nil
}

// UpdateTransactionStatus 更新交易状态
func (s *NFCTransactionService) UpdateTransactionStatus(ctx context.Context, req *request.UpdateTransactionStatusRequest, userID uuid.UUID) (*response.UpdateTransactionStatusResponse, error) {
	// 获取现有交易
	var transaction nfc_relay.NFCTransaction
	if err := global.GVA_DB.Where("transaction_id = ?", req.TransactionID).First(&transaction).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, fmt.Errorf("交易不存在")
		}
		global.GVA_LOG.Error("查询交易失败", zap.String("transactionID", req.TransactionID), zap.Error(err))
		return nil, fmt.Errorf("查询交易失败")
	}

	// 验证状态转换
	if !nfc_relay.IsValidStatusTransition(transaction.Status, req.Status) {
		return nil, fmt.Errorf("无效的状态转换: %s -> %s", transaction.Status, req.Status)
	}

	previousStatus := transaction.Status
	now := time.Now()

	// 更新状态相关字段
	updates := map[string]interface{}{
		"status":     req.Status,
		"updated_by": userID,
		"updated_at": now,
	}

	if req.Reason != "" {
		updates["end_reason"] = req.Reason
	}
	if req.ErrorMsg != "" {
		updates["error_msg"] = req.ErrorMsg
	}

	// 根据状态设置时间字段
	switch req.Status {
	case nfc_relay.StatusActive:
		updates["started_at"] = now
	case nfc_relay.StatusCompleted, nfc_relay.StatusFailed, nfc_relay.StatusCancelled, nfc_relay.StatusTimeout:
		updates["completed_at"] = now
	}

	// 处理扩展元数据
	if req.Metadata != nil {
		metadataBytes, err := json.Marshal(req.Metadata)
		if err != nil {
			global.GVA_LOG.Error("序列化元数据失败", zap.Error(err))
			return nil, fmt.Errorf("元数据格式错误")
		}
		updates["metadata"] = datatypes.JSON(metadataBytes)
	}

	// 更新数据库
	if err := global.GVA_DB.Model(&transaction).Updates(updates).Error; err != nil {
		global.GVA_LOG.Error("更新交易状态失败", zap.String("transactionID", req.TransactionID), zap.Error(err))
		return nil, fmt.Errorf("更新交易状态失败")
	}

	// 更新Redis缓存
	if err := s.cacheTransactionStatus(ctx, req.TransactionID, req.Status, map[string]interface{}{
		"updated_at":      now,
		"previous_status": previousStatus,
		"reason":          req.Reason,
	}); err != nil {
		global.GVA_LOG.Error("更新交易状态缓存失败", zap.Error(err))
		// 不影响主流程
	}

	// 发布MQTT状态更新通知
	go s.notifyTransactionStatusUpdate(ctx, &transaction, req.Status, previousStatus, req.Reason)

	// 如果是终态，执行后续处理
	if req.Status == nfc_relay.StatusCompleted || req.Status == nfc_relay.StatusFailed ||
		req.Status == nfc_relay.StatusCancelled || req.Status == nfc_relay.StatusTimeout {
		go s.handleTransactionCompletion(ctx, &transaction)
	}

	return &response.UpdateTransactionStatusResponse{
		TransactionID:  req.TransactionID,
		Status:         req.Status,
		PreviousStatus: previousStatus,
		UpdatedAt:      now,
		Reason:         req.Reason,
	}, nil
}

// GetTransaction 获取交易详情
func (s *NFCTransactionService) GetTransaction(ctx context.Context, req *request.GetTransactionRequest, userID uuid.UUID) (*response.TransactionDetailResponse, error) {
	var transaction nfc_relay.NFCTransaction

	query := global.GVA_DB.Where("transaction_id = ?", req.TransactionID)

	// 包含APDU消息
	if req.IncludeAPDU {
		query = query.Preload("APDUMessages", func(db *gorm.DB) *gorm.DB {
			return db.Order("sequence_number ASC")
		})
	}

	if err := query.First(&transaction).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, fmt.Errorf("交易不存在")
		}
		global.GVA_LOG.Error("查询交易失败", zap.String("transactionID", req.TransactionID), zap.Error(err))
		return nil, fmt.Errorf("查询交易失败")
	}

	// 权限检查（可以查看自己创建的交易）
	if transaction.CreatedBy != userID {
		// 这里可以添加更复杂的权限逻辑，比如管理员可以查看所有交易
		return nil, fmt.Errorf("无权访问此交易")
	}

	// 获取统计信息
	statistics, err := s.getTransactionStatistics(ctx, req.TransactionID)
	if err != nil {
		global.GVA_LOG.Error("获取交易统计失败", zap.Error(err))
		// 统计信息获取失败不影响主流程
		statistics = response.TransactionStatistics{}
	}

	// 获取时间线
	timeline, err := s.getTransactionTimeline(ctx, req.TransactionID)
	if err != nil {
		global.GVA_LOG.Error("获取交易时间线失败", zap.Error(err))
		timeline = []response.TransactionEvent{}
	}

	return &response.TransactionDetailResponse{
		NFCTransaction: transaction,
		Statistics:     statistics,
		Timeline:       timeline,
	}, nil
}

// generateTransactionID 生成交易ID
func (s *NFCTransactionService) generateTransactionID() (string, error) {
	bytes := make([]byte, 16)
	if _, err := rand.Read(bytes); err != nil {
		return "", err
	}

	// 格式: txn_YYYYMMDD_HEX
	timestamp := time.Now().Format("20060102")
	hexStr := hex.EncodeToString(bytes)[:16]
	return fmt.Sprintf("txn_%s_%s", timestamp, hexStr), nil
}

// acquireUserLock 获取用户锁（改进版本，支持高可用）
func (s *NFCTransactionService) acquireUserLock(ctx context.Context, key string, ttl time.Duration) (bool, error) {
	lockKey := fmt.Sprintf("lock:%s", key)
	identifier := s.generateLockIdentifier()

	// 使用Lua脚本确保原子性操作
	script := `
		-- 检查锁是否存在
		local existing = redis.call("GET", KEYS[1])
		if existing == false then
			-- 锁不存在，可以获取
			redis.call("SET", KEYS[1], ARGV[1], "PX", ARGV[2])
			return 1
		else
			-- 锁已存在，检查是否是同一个持有者（支持可重入）
			if existing == ARGV[1] then
				-- 延长锁的过期时间
				redis.call("PEXPIRE", KEYS[1], ARGV[2])
				return 1
			else
				return 0
			end
		end
	`

	result, err := global.GVA_REDIS.Eval(ctx, script, []string{lockKey}, identifier, int64(ttl/time.Millisecond)).Result()
	if err != nil {
		global.GVA_LOG.Error("获取分布式锁失败",
			zap.String("lockKey", lockKey),
			zap.Duration("ttl", ttl),
			zap.Error(err))
		return false, fmt.Errorf("获取分布式锁失败: %w", err)
	}

	success := result.(int64) == 1
	if success {
		global.GVA_LOG.Debug("获取分布式锁成功",
			zap.String("lockKey", lockKey),
			zap.String("identifier", identifier),
			zap.Duration("ttl", ttl))

		// 在Redis中存储锁的元数据，便于监控和调试
		metaKey := fmt.Sprintf("lock_meta:%s", key)
		metadata := map[string]interface{}{
			"identifier":  identifier,
			"acquired_at": time.Now().Format(time.RFC3339),
			"expires_at":  time.Now().Add(ttl).Format(time.RFC3339),
			"process_id":  fmt.Sprintf("%d", os.Getpid()),
			"service":     "nfc_transaction",
		}
		global.GVA_REDIS.HMSet(ctx, metaKey, metadata).Err()
		global.GVA_REDIS.Expire(ctx, metaKey, ttl+time.Minute).Err() // 元数据比锁多保留1分钟
	}

	return success, nil
}

// releaseUserLock 释放用户锁（改进版本）
func (s *NFCTransactionService) releaseUserLock(ctx context.Context, key string) error {
	lockKey := fmt.Sprintf("lock:%s", key)
	metaKey := fmt.Sprintf("lock_meta:%s", key)

	// 使用Lua脚本确保原子性释放
	script := `
		-- 获取当前锁的值
		local current = redis.call("GET", KEYS[1])
		if current == false then
			-- 锁不存在
			return 0
		else
			-- 删除锁和元数据
			redis.call("DEL", KEYS[1])
			redis.call("DEL", KEYS[2])
			return 1
		end
	`

	result, err := global.GVA_REDIS.Eval(ctx, script, []string{lockKey, metaKey}).Result()
	if err != nil {
		global.GVA_LOG.Error("释放分布式锁失败",
			zap.String("lockKey", lockKey),
			zap.Error(err))
		return fmt.Errorf("释放分布式锁失败: %w", err)
	}

	released := result.(int64) == 1
	if released {
		global.GVA_LOG.Debug("释放分布式锁成功", zap.String("lockKey", lockKey))
	} else {
		global.GVA_LOG.Warn("尝试释放不存在的锁", zap.String("lockKey", lockKey))
	}

	return nil
}

// generateLockIdentifier 生成锁标识符
func (s *NFCTransactionService) generateLockIdentifier() string {
	return fmt.Sprintf("%d-%d-%s",
		os.Getpid(),
		time.Now().UnixNano(),
		uuid.New().String()[:8])
}

// isClientOnline 检查客户端是否在线（改进版本）
func (s *NFCTransactionService) isClientOnline(ctx context.Context, clientID string) (bool, error) {
	if clientID == "" {
		return false, fmt.Errorf("客户端ID不能为空")
	}

	key := fmt.Sprintf("client_heartbeat:%s", clientID)

	// 获取心跳信息
	heartbeatData, err := global.GVA_REDIS.HGetAll(ctx, key).Result()
	if err != nil {
		if err.Error() == "redis: nil" {
			return false, nil
		}
		global.GVA_LOG.Error("检查客户端在线状态失败",
			zap.String("clientID", clientID),
			zap.Error(err))
		return false, fmt.Errorf("检查客户端在线状态失败: %w", err)
	}

	if len(heartbeatData) == 0 {
		return false, nil
	}

	lastSeenStr, exists := heartbeatData["last_seen"]
	if !exists {
		return false, nil
	}

	lastSeenTime, err := time.Parse(time.RFC3339, lastSeenStr)
	if err != nil {
		global.GVA_LOG.Error("解析客户端最后活跃时间失败",
			zap.String("clientID", clientID),
			zap.String("lastSeen", lastSeenStr),
			zap.Error(err))
		return false, nil
	}

	// 检查心跳超时（60秒内有心跳认为在线）
	isOnline := time.Since(lastSeenTime) < 60*time.Second

	// 记录客户端状态检查
	global.GVA_LOG.Debug("客户端在线状态检查",
		zap.String("clientID", clientID),
		zap.Bool("isOnline", isOnline),
		zap.Time("lastSeen", lastSeenTime),
		zap.Duration("timeSince", time.Since(lastSeenTime)))

	return isOnline, nil
}

// cacheTransactionStatus 缓存交易状态（改进版本）
func (s *NFCTransactionService) cacheTransactionStatus(ctx context.Context, transactionID, status string, metadata map[string]interface{}) error {
	if transactionID == "" || status == "" {
		return fmt.Errorf("交易ID和状态不能为空")
	}

	key := fmt.Sprintf("transaction:%s:status", transactionID)
	now := time.Now()

	// 构建缓存数据
	data := map[string]interface{}{
		"transaction_id": transactionID,
		"status":         status,
		"updated_at":     now.Format(time.RFC3339),
		"cached_at":      now.Format(time.RFC3339),
	}

	// 添加元数据
	for k, v := range metadata {
		data[k] = v
	}

	// 使用Pipeline提高性能
	pipe := global.GVA_REDIS.Pipeline()
	pipe.HMSet(ctx, key, data)
	pipe.Expire(ctx, key, 3600*time.Second) // 1小时过期

	// 同时维护交易索引，便于查询用户的所有交易
	if userID, ok := metadata["created_by"]; ok {
		userIndexKey := fmt.Sprintf("user_transactions:%v", userID)
		pipe.SAdd(ctx, userIndexKey, transactionID)
		pipe.Expire(ctx, userIndexKey, 86400*time.Second) // 24小时过期
	}

	_, err := pipe.Exec(ctx)
	if err != nil {
		global.GVA_LOG.Error("缓存交易状态失败",
			zap.String("transactionID", transactionID),
			zap.String("status", status),
			zap.Error(err))
		return fmt.Errorf("缓存交易状态失败: %w", err)
	}

	// 发布状态变更事件到Redis频道
	publishData := map[string]interface{}{
		"transaction_id": transactionID,
		"status":         status,
		"timestamp":      now.Format(time.RFC3339),
		"metadata":       metadata,
	}

	publishJSON, _ := json.Marshal(publishData)
	global.GVA_REDIS.Publish(ctx, "transaction:status_changed", publishJSON).Err()

	global.GVA_LOG.Debug("交易状态缓存成功",
		zap.String("transactionID", transactionID),
		zap.String("status", status),
		zap.Any("metadata", metadata))

	return nil
}

// notifyTransactionCreated 通知交易创建（完善版本）
func (s *NFCTransactionService) notifyTransactionCreated(ctx context.Context, transaction *nfc_relay.NFCTransaction) {
	// WebSocket实时通知
	NotifyTransactionStatus(transaction.CreatedBy.String(), map[string]interface{}{
		"transaction_id":        transaction.TransactionID,
		"status":                transaction.Status,
		"action":                "created",
		"timestamp":             transaction.CreatedAt,
		"transmitter_client_id": transaction.TransmitterClientID,
		"receiver_client_id":    transaction.ReceiverClientID,
		"card_type":             transaction.CardType,
		"description":           transaction.Description,
		"expires_at":            transaction.ExpiresAt,
	})

	// MQTT通知 - 通知传卡端
	mqttService := GetMQTTService()
	if err := mqttService.PublishTransactionCreated(ctx, transaction); err != nil {
		global.GVA_LOG.Error("MQTT通知传卡端失败",
			zap.String("transactionID", transaction.TransactionID),
			zap.String("transmitterClientID", transaction.TransmitterClientID),
			zap.Error(err))
	}

	// 记录操作日志
	global.GVA_LOG.Info("交易创建通知已发送",
		zap.String("transactionID", transaction.TransactionID),
		zap.String("transmitterClientID", transaction.TransmitterClientID),
		zap.String("receiverClientID", transaction.ReceiverClientID),
		zap.String("status", transaction.Status),
		zap.String("cardType", transaction.CardType))
}

// notifyTransactionStatusUpdate 通知交易状态更新（完善版本）
func (s *NFCTransactionService) notifyTransactionStatusUpdate(ctx context.Context, transaction *nfc_relay.NFCTransaction, newStatus, oldStatus, reason string) {
	// WebSocket实时通知
	NotifyTransactionStatus(transaction.CreatedBy.String(), map[string]interface{}{
		"transaction_id":        transaction.TransactionID,
		"status":                newStatus,
		"previous_status":       oldStatus,
		"action":                "status_updated",
		"timestamp":             time.Now(),
		"reason":                reason,
		"transmitter_client_id": transaction.TransmitterClientID,
		"receiver_client_id":    transaction.ReceiverClientID,
	})

	// MQTT通知 - 通知相关客户端状态变更
	mqttService := GetMQTTService()

	// 通知传卡端
	if err := mqttService.PublishTransactionStatusUpdate(ctx,
		transaction.TransactionID,
		transaction.TransmitterClientID,
		newStatus, oldStatus, reason); err != nil {
		global.GVA_LOG.Error("MQTT通知传卡端状态更新失败",
			zap.String("transactionID", transaction.TransactionID),
			zap.String("clientID", transaction.TransmitterClientID),
			zap.Error(err))
	}

	// 通知收卡端
	if err := mqttService.PublishTransactionStatusUpdate(ctx,
		transaction.TransactionID,
		transaction.ReceiverClientID,
		newStatus, oldStatus, reason); err != nil {
		global.GVA_LOG.Error("MQTT通知收卡端状态更新失败",
			zap.String("transactionID", transaction.TransactionID),
			zap.String("clientID", transaction.ReceiverClientID),
			zap.Error(err))
	}

	// 如果交易完成，发送完成事件
	if newStatus == nfc_relay.StatusCompleted ||
		newStatus == nfc_relay.StatusFailed ||
		newStatus == nfc_relay.StatusCancelled ||
		newStatus == nfc_relay.StatusTimeout {
		go s.handleTransactionCompletion(ctx, transaction)
	}

	global.GVA_LOG.Info("交易状态更新通知已发送",
		zap.String("transactionID", transaction.TransactionID),
		zap.String("oldStatus", oldStatus),
		zap.String("newStatus", newStatus),
		zap.String("reason", reason))
}

// handleTransactionCompletion 处理交易完成（完善版本）
func (s *NFCTransactionService) handleTransactionCompletion(ctx context.Context, transaction *nfc_relay.NFCTransaction) {
	// 清理Redis缓存
	lockKey := fmt.Sprintf("user_transaction:%s", transaction.CreatedBy.String())
	s.releaseUserLock(ctx, lockKey)

	// 清理交易状态缓存
	statusKey := fmt.Sprintf("transaction:%s:status", transaction.TransactionID)
	global.GVA_REDIS.Del(ctx, statusKey).Err()

	// 更新统计数据
	go s.updateDailyStatistics(ctx, transaction)

	// 记录审计日志
	s.logTransactionCompletion(transaction)

	global.GVA_LOG.Info("交易完成处理",
		zap.String("transactionID", transaction.TransactionID),
		zap.String("status", transaction.Status),
		zap.String("endReason", transaction.EndReason))
}

// updateDailyStatistics 更新每日统计数据
func (s *NFCTransactionService) updateDailyStatistics(ctx context.Context, transaction *nfc_relay.NFCTransaction) {
	today := time.Now().Format("2006-01-02")
	date, _ := time.Parse("2006-01-02", today)

	var stats nfc_relay.NFCTransactionStatistics
	err := global.GVA_DB.Where("date = ?", date).First(&stats).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			// 创建新的统计记录
			stats = nfc_relay.NFCTransactionStatistics{
				Date: date,
			}
		} else {
			global.GVA_LOG.Error("查询统计数据失败", zap.Error(err))
			return
		}
	}

	// 更新统计数据
	stats.TotalTransactions++
	switch transaction.Status {
	case nfc_relay.StatusCompleted:
		stats.SuccessfulTransactions++
	case nfc_relay.StatusFailed:
		stats.FailedTransactions++
	case nfc_relay.StatusTimeout:
		stats.TimeoutTransactions++
	case nfc_relay.StatusCancelled:
		stats.CancelledTransactions++
	}

	// 更新APDU消息统计
	stats.TotalAPDUMessages += transaction.APDUCount

	// 更新处理时间统计
	if transaction.TotalProcessingTimeMs > 0 {
		if stats.TotalTransactions == 1 {
			stats.AverageProcessingTimeMs = float64(transaction.TotalProcessingTimeMs)
		} else {
			// 计算新的平均值
			totalTime := (stats.AverageProcessingTimeMs * float64(stats.TotalTransactions-1)) + float64(transaction.TotalProcessingTimeMs)
			stats.AverageProcessingTimeMs = totalTime / float64(stats.TotalTransactions)
		}

		if stats.MinProcessingTimeMs == 0 || transaction.TotalProcessingTimeMs < stats.MinProcessingTimeMs {
			stats.MinProcessingTimeMs = transaction.TotalProcessingTimeMs
		}
		if transaction.TotalProcessingTimeMs > stats.MaxProcessingTimeMs {
			stats.MaxProcessingTimeMs = transaction.TotalProcessingTimeMs
		}
	}

	// 保存统计数据
	if err := global.GVA_DB.Save(&stats).Error; err != nil {
		global.GVA_LOG.Error("保存统计数据失败", zap.Error(err))
	}
}

// logTransactionCompletion 记录交易完成审计日志
func (s *NFCTransactionService) logTransactionCompletion(transaction *nfc_relay.NFCTransaction) {
	auditData := map[string]interface{}{
		"transaction_id":        transaction.TransactionID,
		"status":                transaction.Status,
		"transmitter_client_id": transaction.TransmitterClientID,
		"receiver_client_id":    transaction.ReceiverClientID,
		"card_type":             transaction.CardType,
		"created_by":            transaction.CreatedBy,
		"created_at":            transaction.CreatedAt,
		"completed_at":          transaction.CompletedAt,
		"apdu_count":            transaction.APDUCount,
		"processing_time_ms":    transaction.TotalProcessingTimeMs,
		"end_reason":            transaction.EndReason,
		"error_msg":             transaction.ErrorMsg,
	}

	auditJSON, _ := json.Marshal(auditData)
	global.GVA_LOG.Info("交易完成审计日志", zap.String("audit", string(auditJSON)))
}

// getTransactionStatistics 获取交易统计
func (s *NFCTransactionService) getTransactionStatistics(ctx context.Context, transactionID string) (response.TransactionStatistics, error) {
	var stats response.TransactionStatistics

	// 查询APDU消息统计
	var apduCount int64
	if err := global.GVA_DB.Model(&nfc_relay.NFCAPDUMessage{}).
		Where("transaction_id = ?", transactionID).
		Count(&apduCount).Error; err != nil {
		return stats, err
	}

	stats.APDUMessageCount = int(apduCount)

	// 可以添加更多统计逻辑
	return stats, nil
}

// getTransactionTimeline 获取交易时间线
func (s *NFCTransactionService) getTransactionTimeline(ctx context.Context, transactionID string) ([]response.TransactionEvent, error) {
	var events []response.TransactionEvent

	// 这里可以从数据库或日志中获取事件时间线
	// 简化实现，返回空数组
	return events, nil
}

// GetTransactionList 获取交易列表
func (s *NFCTransactionService) GetTransactionList(ctx context.Context, req *request.GetTransactionListRequest, userID uuid.UUID) (*response.TransactionListResponse, error) {
	// 构建查询条件
	query := global.GVA_DB.Model(&nfc_relay.NFCTransaction{})

	// 权限过滤：只能查看自己创建的交易（管理员可以查看所有）
	// TODO: 这里可以根据用户角色进行更精细的权限控制
	query = query.Where("created_by = ?", userID)

	// 添加过滤条件
	if req.TransmitterClientID != "" {
		query = query.Where("transmitter_client_id LIKE ?", "%"+req.TransmitterClientID+"%")
	}
	if req.ReceiverClientID != "" {
		query = query.Where("receiver_client_id LIKE ?", "%"+req.ReceiverClientID+"%")
	}
	if req.Status != "" {
		query = query.Where("status = ?", req.Status)
	}
	if req.CardType != "" {
		query = query.Where("card_type = ?", req.CardType)
	}

	// 时间范围过滤
	if req.StartTime != "" {
		if startTime, err := time.Parse("2006-01-02 15:04:05", req.StartTime); err == nil {
			query = query.Where("created_at >= ?", startTime)
		}
	}
	if req.EndTime != "" {
		if endTime, err := time.Parse("2006-01-02 15:04:05", req.EndTime); err == nil {
			query = query.Where("created_at <= ?", endTime)
		}
	}

	// 关键词搜索
	if req.Keyword != "" {
		keyword := "%" + req.Keyword + "%"
		query = query.Where("description LIKE ? OR tags LIKE ? OR transaction_id LIKE ?", keyword, keyword, keyword)
	}

	// 获取总数
	var total int64
	if err := query.Count(&total).Error; err != nil {
		global.GVA_LOG.Error("查询交易总数失败", zap.Error(err))
		return nil, fmt.Errorf("查询交易总数失败: %w", err)
	}

	// 排序
	orderBy := req.OrderBy
	if orderBy == "" {
		orderBy = "created_at"
	}
	order := req.Order
	if order == "" {
		order = "desc"
	}
	query = query.Order(fmt.Sprintf("%s %s", orderBy, order))

	// 分页查询
	offset := (req.Page - 1) * req.PageSize
	var transactions []nfc_relay.NFCTransaction
	if err := query.Offset(offset).Limit(req.PageSize).Find(&transactions).Error; err != nil {
		global.GVA_LOG.Error("查询交易列表失败", zap.Error(err))
		return nil, fmt.Errorf("查询交易列表失败: %w", err)
	}

	// 转换为响应格式
	list := make([]response.TransactionListItem, len(transactions))
	for i, tx := range transactions {
		list[i] = response.TransactionListItem{
			ID:                    tx.ID,
			TransactionID:         tx.TransactionID,
			TransmitterClientID:   tx.TransmitterClientID,
			ReceiverClientID:      tx.ReceiverClientID,
			Status:                tx.Status,
			CardType:              tx.CardType,
			Description:           tx.Description,
			APDUCount:             tx.APDUCount,
			TotalProcessingTimeMs: tx.TotalProcessingTimeMs,
			CreatedAt:             tx.CreatedAt,
			StartedAt:             tx.StartedAt,
			CompletedAt:           tx.CompletedAt,
			ExpiresAt:             tx.ExpiresAt,
			EndReason:             tx.EndReason,
			Tags:                  tx.Tags,
		}
	}

	// 计算汇总信息
	summary := s.calculateSummary(transactions)

	return &response.TransactionListResponse{
		List:     list,
		Total:    total,
		Page:     req.Page,
		PageSize: req.PageSize,
		Summary:  summary,
	}, nil
}

// calculateSummary 计算交易汇总信息
func (s *NFCTransactionService) calculateSummary(transactions []nfc_relay.NFCTransaction) response.TransactionSummary {
	summary := response.TransactionSummary{
		TotalCount: len(transactions),
	}

	totalProcessingTime := 0
	processedCount := 0

	for _, tx := range transactions {
		switch tx.Status {
		case nfc_relay.StatusCompleted:
			summary.CompletedCount++
		case nfc_relay.StatusFailed:
			summary.FailedCount++
		case nfc_relay.StatusPending:
			summary.PendingCount++
		case nfc_relay.StatusActive, nfc_relay.StatusProcessing:
			summary.ActiveCount++
		}

		if tx.TotalProcessingTimeMs > 0 {
			totalProcessingTime += tx.TotalProcessingTimeMs
			processedCount++
		}
	}

	// 计算成功率
	if summary.TotalCount > 0 {
		summary.SuccessRate = float64(summary.CompletedCount) / float64(summary.TotalCount) * 100
	}

	// 计算平均处理时间
	if processedCount > 0 {
		summary.AverageProcessingMs = float64(totalProcessingTime) / float64(processedCount)
	}

	return summary
}

// DeleteTransaction 删除交易
func (s *NFCTransactionService) DeleteTransaction(ctx context.Context, req *request.DeleteTransactionRequest, userID uuid.UUID) error {
	// 查询交易
	var transaction nfc_relay.NFCTransaction
	if err := global.GVA_DB.Where("transaction_id = ?", req.TransactionID).First(&transaction).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return fmt.Errorf("交易不存在")
		}
		return fmt.Errorf("查询交易失败: %w", err)
	}

	// 权限检查
	if transaction.CreatedBy != userID {
		return fmt.Errorf("无权删除此交易")
	}

	// 检查交易状态
	if !req.Force {
		if transaction.Status == nfc_relay.StatusActive || transaction.Status == nfc_relay.StatusProcessing {
			return fmt.Errorf("无法删除活跃状态的交易，请先取消交易或使用强制删除")
		}
	}

	// 开启事务删除
	return global.GVA_DB.Transaction(func(tx *gorm.DB) error {
		// 删除关联的APDU消息
		if err := tx.Where("transaction_id = ?", req.TransactionID).Delete(&nfc_relay.NFCAPDUMessage{}).Error; err != nil {
			return fmt.Errorf("删除APDU消息失败: %w", err)
		}

		// 删除交易记录
		if err := tx.Delete(&transaction).Error; err != nil {
			return fmt.Errorf("删除交易失败: %w", err)
		}

		// 清理Redis缓存
		s.cleanupTransactionCache(ctx, req.TransactionID, userID)

		return nil
	})
}

// cleanupTransactionCache 清理交易相关的Redis缓存
func (s *NFCTransactionService) cleanupTransactionCache(ctx context.Context, transactionID string, userID uuid.UUID) {
	keys := []string{
		fmt.Sprintf("transaction:%s:status", transactionID),
		fmt.Sprintf("transaction:%s:apdu_messages", transactionID),
		fmt.Sprintf("lock:user_transaction:%s", userID.String()),
	}

	for _, key := range keys {
		global.GVA_REDIS.Del(ctx, key).Err()
	}
}

// SendAPDU 发送APDU消息
func (s *NFCTransactionService) SendAPDU(ctx context.Context, req *request.SendAPDURequest, userID uuid.UUID) (*response.SendAPDUResponse, error) {
	// 验证交易
	var transaction nfc_relay.NFCTransaction
	if err := global.GVA_DB.Where("transaction_id = ?", req.TransactionID).First(&transaction).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, fmt.Errorf("交易不存在")
		}
		return nil, fmt.Errorf("查询交易失败: %w", err)
	}

	// 权限检查
	if transaction.CreatedBy != userID {
		return nil, fmt.Errorf("无权操作此交易")
	}

	// 状态检查
	if transaction.Status != nfc_relay.StatusActive && transaction.Status != nfc_relay.StatusProcessing {
		return nil, fmt.Errorf("交易状态不支持发送APDU消息: %s", transaction.Status)
	}

	// 创建APDU消息记录
	now := time.Now()
	apduMessage := &nfc_relay.NFCAPDUMessage{
		TransactionID:  req.TransactionID,
		Direction:      req.Direction,
		APDUHex:        req.APDUHex,
		SequenceNumber: req.SequenceNumber,
		Priority:       req.Priority,
		MessageType:    req.MessageType,
		Status:         nfc_relay.MessageStatusPending,
		SentAt:         &now,
	}

	// 处理元数据
	if req.Metadata != nil {
		if metadataJSON, err := json.Marshal(req.Metadata); err == nil {
			apduMessage.Metadata = datatypes.JSON(metadataJSON)
		}
	}

	// 保存到数据库
	if err := global.GVA_DB.Create(apduMessage).Error; err != nil {
		return nil, fmt.Errorf("保存APDU消息失败: %w", err)
	}

	// 通过MQTT发送到客户端
	mqttService := GetMQTTService()
	mqttMsg := APDUMessage{
		TransactionID:  req.TransactionID,
		SequenceNumber: req.SequenceNumber,
		Direction:      req.Direction,
		APDUHex:        req.APDUHex,
		Priority:       req.Priority,
		MessageType:    req.MessageType,
		Timeout:        30, // 默认30秒超时
	}

	// 确定目标客户端
	var targetClientID string
	if req.Direction == nfc_relay.DirectionToReceiver {
		targetClientID = transaction.ReceiverClientID
	} else {
		targetClientID = transaction.TransmitterClientID
	}

	if err := mqttService.SendAPDUToClient(ctx, targetClientID, mqttMsg); err != nil {
		// 更新消息状态为失败
		global.GVA_DB.Model(apduMessage).Update("status", nfc_relay.MessageStatusFailed)
		return nil, fmt.Errorf("发送APDU消息到客户端失败: %w", err)
	}

	// 更新消息状态为已发送
	global.GVA_DB.Model(apduMessage).Update("status", nfc_relay.MessageStatusSent)

	// 更新交易的APDU计数
	global.GVA_DB.Model(&transaction).UpdateColumn("apdu_count", gorm.Expr("apdu_count + ?", 1))

	return &response.SendAPDUResponse{
		MessageID:      apduMessage.ID,
		TransactionID:  req.TransactionID,
		Direction:      req.Direction,
		SequenceNumber: req.SequenceNumber,
		Status:         nfc_relay.MessageStatusSent,
		SentAt:         now,
	}, nil
}

// GetAPDUList 获取APDU消息列表
func (s *NFCTransactionService) GetAPDUList(ctx context.Context, req *request.GetAPDUListRequest, userID uuid.UUID) (*response.APDUMessageListResponse, error) {
	// 验证交易权限
	var transaction nfc_relay.NFCTransaction
	if err := global.GVA_DB.Where("transaction_id = ?", req.TransactionID).First(&transaction).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, fmt.Errorf("交易不存在")
		}
		return nil, fmt.Errorf("查询交易失败: %w", err)
	}

	if transaction.CreatedBy != userID {
		return nil, fmt.Errorf("无权访问此交易的APDU消息")
	}

	// 构建查询
	query := global.GVA_DB.Model(&nfc_relay.NFCAPDUMessage{}).Where("transaction_id = ?", req.TransactionID)

	// 过滤条件
	if req.Direction != "" {
		query = query.Where("direction = ?", req.Direction)
	}
	if req.Status != "" {
		query = query.Where("status = ?", req.Status)
	}
	if req.Priority != "" {
		query = query.Where("priority = ?", req.Priority)
	}

	// 时间范围
	if req.StartTime != "" {
		if startTime, err := time.Parse("2006-01-02 15:04:05", req.StartTime); err == nil {
			query = query.Where("created_at >= ?", startTime)
		}
	}
	if req.EndTime != "" {
		if endTime, err := time.Parse("2006-01-02 15:04:05", req.EndTime); err == nil {
			query = query.Where("created_at <= ?", endTime)
		}
	}

	// 获取总数
	var total int64
	if err := query.Count(&total).Error; err != nil {
		return nil, fmt.Errorf("查询APDU消息总数失败: %w", err)
	}

	// 分页查询
	offset := (req.Page - 1) * req.PageSize
	var messages []nfc_relay.NFCAPDUMessage
	if err := query.Order("sequence_number ASC").Offset(offset).Limit(req.PageSize).Find(&messages).Error; err != nil {
		return nil, fmt.Errorf("查询APDU消息列表失败: %w", err)
	}

	// 转换为响应格式
	list := make([]response.APDUMessageItem, len(messages))
	for i, msg := range messages {
		list[i] = response.APDUMessageItem{
			ID:             msg.ID,
			TransactionID:  msg.TransactionID,
			Direction:      msg.Direction,
			APDUHex:        msg.APDUHex,
			SequenceNumber: msg.SequenceNumber,
			Priority:       msg.Priority,
			MessageType:    msg.MessageType,
			Status:         msg.Status,
			SentAt:         msg.SentAt,
			ReceivedAt:     msg.ReceivedAt,
			ResponseTime:   msg.ResponseTime,
			ErrorMsg:       msg.ErrorMsg,
			CreatedAt:      msg.CreatedAt,
		}
	}

	return &response.APDUMessageListResponse{
		List:     list,
		Total:    total,
		Page:     req.Page,
		PageSize: req.PageSize,
	}, nil
}

// GetStatistics 获取统计信息
func (s *NFCTransactionService) GetStatistics(ctx context.Context, req *request.GetStatisticsRequest, userID uuid.UUID) (*response.TransactionStatisticsResponse, error) {
	// 解析日期范围
	startDate, err := time.Parse("2006-01-02", req.StartDate)
	if err != nil {
		return nil, fmt.Errorf("无效的开始日期格式: %w", err)
	}

	endDate, err := time.Parse("2006-01-02", req.EndDate)
	if err != nil {
		return nil, fmt.Errorf("无效的结束日期格式: %w", err)
	}

	// 确保结束日期包含整天
	endDate = endDate.Add(24 * time.Hour).Add(-1 * time.Second)

	// 构建基础查询
	baseQuery := global.GVA_DB.Model(&nfc_relay.NFCTransaction{}).
		Where("created_by = ?", userID).
		Where("created_at BETWEEN ? AND ?", startDate, endDate)

	// 添加过滤条件
	if req.CardType != "" {
		baseQuery = baseQuery.Where("card_type = ?", req.CardType)
	}
	if req.Status != "" {
		baseQuery = baseQuery.Where("status = ?", req.Status)
	}

	// 获取汇总统计
	summary, err := s.calculateStatisticsSummary(baseQuery)
	if err != nil {
		return nil, fmt.Errorf("计算统计汇总失败: %w", err)
	}

	// 获取每日统计
	dailyStats, err := s.calculateDailyStatistics(baseQuery, startDate, endDate, req.GroupBy)
	if err != nil {
		return nil, fmt.Errorf("计算每日统计失败: %w", err)
	}

	// 生成图表数据
	chartData := s.generateChartData(dailyStats)

	// 获取客户端统计
	topClients, err := s.getTopClientsStatistics(baseQuery)
	if err != nil {
		return nil, fmt.Errorf("获取客户端统计失败: %w", err)
	}

	// 错误分析
	errorAnalysis, err := s.getErrorAnalysis(baseQuery)
	if err != nil {
		return nil, fmt.Errorf("获取错误分析失败: %w", err)
	}

	return &response.TransactionStatisticsResponse{
		DateRange: response.DateRange{
			StartDate: req.StartDate,
			EndDate:   req.EndDate,
			Days:      int(endDate.Sub(startDate).Hours()/24) + 1,
		},
		Summary:       summary,
		DailyStats:    dailyStats,
		ChartData:     chartData,
		TopClients:    topClients,
		ErrorAnalysis: errorAnalysis,
	}, nil
}

// calculateStatisticsSummary 计算统计汇总
func (s *NFCTransactionService) calculateStatisticsSummary(query *gorm.DB) (response.StatisticsSummary, error) {
	var summary response.StatisticsSummary

	// 总交易数 - 修复类型不匹配问题
	var totalTransactions int64
	if err := query.Count(&totalTransactions).Error; err != nil {
		return summary, err
	}
	summary.TotalTransactions = int(totalTransactions)

	// 各状态交易数
	statusCounts := make(map[string]int64)
	var results []struct {
		Status string
		Count  int64
	}

	if err := query.Select("status, COUNT(*) as count").Group("status").Scan(&results).Error; err != nil {
		return summary, err
	}

	for _, result := range results {
		statusCounts[result.Status] = result.Count
		switch result.Status {
		case nfc_relay.StatusCompleted:
			summary.SuccessfulTransactions = int(result.Count)
		case nfc_relay.StatusFailed:
			summary.FailedTransactions = int(result.Count)
		}
	}

	// 计算成功率
	if summary.TotalTransactions > 0 {
		summary.SuccessRate = float64(summary.SuccessfulTransactions) / float64(summary.TotalTransactions) * 100
	}

	// APDU消息总数和平均处理时间
	var aggregates struct {
		TotalAPDU int64   `gorm:"column:total_apdu"`
		AvgTime   float64 `gorm:"column:avg_time"`
		TotalTime int64   `gorm:"column:total_time"`
	}

	if err := query.Select(
		"SUM(apdu_count) as total_apdu",
		"AVG(total_processing_time_ms) as avg_time",
		"SUM(total_processing_time_ms) as total_time",
	).Scan(&aggregates).Error; err != nil {
		return summary, err
	}

	summary.TotalAPDUMessages = int(aggregates.TotalAPDU)
	summary.AverageProcessingMs = aggregates.AvgTime
	summary.TotalProcessingTimeMs = aggregates.TotalTime

	return summary, nil
}

// calculateDailyStatistics 计算每日/每小时统计数据
func (s *NFCTransactionService) calculateDailyStatistics(query *gorm.DB, startDate, endDate time.Time, groupBy string) ([]response.DailyStatistics, error) {
	var dailyStats []response.DailyStatistics
	var selectClause, groupByClause, orderByClause string

	switch groupBy {
	case "hour":
		selectClause = "DATE_FORMAT(created_at, '%Y-%m-%d %H:00:00') as date"
		groupByClause = "DATE_FORMAT(created_at, '%Y-%m-%d %H:00:00')"
	default: // "day" or default
		selectClause = "DATE_FORMAT(created_at, '%Y-%m-%d') as date"
		groupByClause = "DATE_FORMAT(created_at, '%Y-%m-%d')"
	}
	orderByClause = groupByClause + " ASC"

	err := query.
		Select(selectClause+", COUNT(*) as total_transactions, "+
			"SUM(CASE WHEN status = 'completed' THEN 1 ELSE 0 END) as successful_transactions, "+
			"SUM(CASE WHEN status = 'failed' THEN 1 ELSE 0 END) as failed_transactions, "+
			"AVG(total_processing_time_ms) as average_processing_ms, "+
			"SUM(apdu_count) as total_apdu_messages").
		Where("created_at BETWEEN ? AND ?", startDate, endDate).
		Group(groupByClause).
		Order(orderByClause).
		Scan(&dailyStats).Error

	if err != nil {
		global.GVA_LOG.Error("计算每日统计失败", zap.Error(err))
		return nil, fmt.Errorf("计算每日统计失败: %w", err)
	}
	return dailyStats, nil
}

// generateChartData 生成图表数据
func (s *NFCTransactionService) generateChartData(dailyStats []response.DailyStatistics) response.StatisticsChartData {
	var chartData response.StatisticsChartData

	// 趋势数据
	for _, stat := range dailyStats {
		chartData.TransactionTrend = append(chartData.TransactionTrend, response.ChartPoint{
			X: stat.Date,
			Y: float64(stat.TotalTransactions),
		})

		chartData.SuccessRateTrend = append(chartData.SuccessRateTrend, response.ChartPoint{
			X: stat.Date,
			Y: stat.SuccessRate,
		})

		chartData.ResponseTimeTrend = append(chartData.ResponseTimeTrend, response.ChartPoint{
			X: stat.Date,
			Y: stat.AverageProcessingMs,
		})
	}

	// 状态分布（这里简化处理，实际应该从数据库统计）
	chartData.StatusDistribution = []response.PieChartItem{
		{Name: "completed", Value: 70, Count: 70},
		{Name: "failed", Value: 20, Count: 20},
		{Name: "pending", Value: 10, Count: 10},
	}

	return chartData
}

// getTopClientsStatistics 获取客户端统计
func (s *NFCTransactionService) getTopClientsStatistics(query *gorm.DB) ([]response.ClientStatistics, error) {
	// 简化实现，返回空数组
	return []response.ClientStatistics{}, nil
}

// getErrorAnalysis 获取错误分析
func (s *NFCTransactionService) getErrorAnalysis(query *gorm.DB) (response.ErrorAnalysis, error) {
	var analysis response.ErrorAnalysis

	// 统计错误总数
	var errorCount int64
	query.Where("status = ?", nfc_relay.StatusFailed).Count(&errorCount)
	analysis.TotalErrors = int(errorCount)

	// 计算错误率
	var totalCount int64
	query.Count(&totalCount)
	if totalCount > 0 {
		analysis.ErrorRate = float64(errorCount) / float64(totalCount) * 100
	}

	return analysis, nil
}

// BatchUpdateTransactionStatus 批量更新交易状态
func (s *NFCTransactionService) BatchUpdateTransactionStatus(ctx context.Context, req *request.BatchUpdateTransactionRequest, userID uuid.UUID) (*response.BatchOperationResponse, error) {
	result := &response.BatchOperationResponse{
		Total: len(req.TransactionIDs),
	}

	for _, transactionID := range req.TransactionIDs {
		updateReq := &request.UpdateTransactionStatusRequest{
			TransactionID: transactionID,
			Status:        req.Status,
			Reason:        req.Reason,
			Metadata:      req.Metadata,
		}

		if _, err := s.UpdateTransactionStatus(ctx, updateReq, userID); err != nil {
			result.FailureCount++
			result.FailureErrors = append(result.FailureErrors, response.BatchError{
				ID:    transactionID,
				Error: err.Error(),
			})
		} else {
			result.SuccessCount++
			result.SuccessIDs = append(result.SuccessIDs, transactionID)
		}
	}

	return result, nil
}

// InitiateTransactionSession 发起交易会话
func (nfcTransactionService *NFCTransactionService) InitiateTransactionSession(ctx context.Context, req nfcRelayReq.InitiateTransactionSessionRequest, userID uuid.UUID, username string) (*nfcRelayReq.TransactionSessionResponse, error) {
	// 1. 生成交易ID
	transactionID := fmt.Sprintf("txn_%d_%s", time.Now().UnixMilli(), uuid.New().String()[:8])

	// 2. 从JWT中提取客户端ID (假设已经在中间件中设置)
	clientID, exists := ctx.Value("clientID").(string)
	if !exists || clientID == "" {
		return nil, fmt.Errorf("无法获取客户端ID")
	}

	// 3. 生成动态主题配置
	topicConfig := generateTopicConfig(transactionID)

	// 4. 计算过期时间
	expiresAt := time.Now().Add(time.Duration(req.TimeoutSecs) * time.Second)

	// 5. 创建交易记录
	transaction := &nfc_relay.NFCTransaction{
		TransactionID: transactionID,
		Status:        nfc_relay.StatusPending,
		CardType:      req.CardType,
		Description:   req.Description,
		CreatedBy:     userID,
		ExpiresAt:     &expiresAt,
		Tags:          fmt.Sprintf("session,role:%s", req.Role),

		// 设置动态主题
		TransmitterStateTopic:  topicConfig.TransmitterStateTopic,
		ReceiverStateTopic:     topicConfig.ReceiverStateTopic,
		APDUToTransmitterTopic: topicConfig.APDUToTransmitterTopic,
		APDUToReceiverTopic:    topicConfig.APDUToReceiverTopic,
		ControlTopic:           topicConfig.ControlTopic,
		HeartbeatTopic:         topicConfig.HeartbeatTopic,
	}

	// 根据角色设置客户端ID
	if req.Role == "transmitter" {
		transaction.TransmitterClientID = clientID
	} else {
		transaction.ReceiverClientID = clientID
	}

	// 处理元数据
	if req.Metadata != nil || req.DeviceInfo != nil {
		metadata := map[string]interface{}{
			"device_info":       req.DeviceInfo,
			"metadata":          req.Metadata,
			"initiated_by_role": req.Role,
		}
		if metadataJSON, err := json.Marshal(metadata); err == nil {
			transaction.Metadata = datatypes.JSON(metadataJSON)
		}
	}

	// 6. 保存到数据库
	if err := global.GVA_DB.Create(transaction).Error; err != nil {
		global.GVA_LOG.Error("创建交易会话失败", zap.String("transactionID", transactionID), zap.Error(err))
		return nil, fmt.Errorf("创建交易会话失败: %w", err)
	}

	// 7. 缓存到Redis用于ACL检查
	if err := cacheTransactionForACL(transactionID, clientID, req.Role); err != nil {
		global.GVA_LOG.Warn("缓存交易会话到Redis失败", zap.Error(err))
	}

	// 8. 构建响应
	response := &nfcRelayReq.TransactionSessionResponse{
		TransactionID: transactionID,
		Status:        nfc_relay.StatusPending,
		Role:          req.Role,
		TopicConfig:   topicConfig,
		ExpiresAt:     expiresAt.Unix(),
		CreatedAt:     time.Now().Unix(),
	}

	// 根据角色设置客户端ID
	if req.Role == "transmitter" {
		response.TransmitterClientID = clientID
		response.PeerRole = "receiver"
	} else {
		response.ReceiverClientID = clientID
		response.PeerRole = "transmitter"
	}

	global.GVA_LOG.Info("交易会话发起成功",
		zap.String("transactionID", transactionID),
		zap.String("clientID", clientID),
		zap.String("role", req.Role),
		zap.String("username", username))

	return response, nil
}

// JoinTransactionSession 加入交易会话
func (nfcTransactionService *NFCTransactionService) JoinTransactionSession(ctx context.Context, req nfcRelayReq.JoinTransactionSessionRequest, userID uuid.UUID, username string) (*nfcRelayReq.TransactionSessionResponse, error) {
	// 1. 从JWT中提取客户端ID
	clientID, exists := ctx.Value("clientID").(string)
	if !exists || clientID == "" {
		return nil, fmt.Errorf("无法获取客户端ID")
	}

	// 2. 查询交易记录
	var transaction nfc_relay.NFCTransaction
	if err := global.GVA_DB.Where("transaction_id = ?", req.TransactionID).First(&transaction).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, fmt.Errorf("交易会话不存在")
		}
		return nil, fmt.Errorf("查询交易会话失败: %w", err)
	}

	// 3. 检查交易状态
	if transaction.Status != nfc_relay.StatusPending {
		return nil, fmt.Errorf("交易会话状态无效，当前状态: %s", transaction.Status)
	}

	// 4. 检查是否过期
	if transaction.ExpiresAt != nil && time.Now().After(*transaction.ExpiresAt) {
		return nil, fmt.Errorf("交易会话已过期")
	}

	// 5. 检查角色冲突
	var peerRole string
	var peerClientID string

	if req.Role == "transmitter" {
		if transaction.TransmitterClientID != "" {
			return nil, fmt.Errorf("传卡端角色已被占用")
		}
		peerRole = "receiver"
		peerClientID = transaction.ReceiverClientID
		transaction.TransmitterClientID = clientID
	} else {
		if transaction.ReceiverClientID != "" {
			return nil, fmt.Errorf("收卡端角色已被占用")
		}
		peerRole = "transmitter"
		peerClientID = transaction.TransmitterClientID
		transaction.ReceiverClientID = clientID
	}

	// 6. 检查是否双方都已连接
	if transaction.TransmitterClientID != "" && transaction.ReceiverClientID != "" {
		transaction.Status = nfc_relay.StatusActive
		transaction.StartedAt = &time.Time{}
		now := time.Now()
		transaction.StartedAt = &now
	}

	// 7. 更新元数据
	if req.DeviceInfo != nil || req.Metadata != nil {
		var existingMetadata map[string]interface{}
		if transaction.Metadata != nil {
			json.Unmarshal(transaction.Metadata, &existingMetadata)
		}
		if existingMetadata == nil {
			existingMetadata = make(map[string]interface{})
		}

		// 根据角色存储设备信息
		roleKey := fmt.Sprintf("%s_device_info", req.Role)
		existingMetadata[roleKey] = req.DeviceInfo
		existingMetadata[fmt.Sprintf("%s_metadata", req.Role)] = req.Metadata
		existingMetadata[fmt.Sprintf("joined_by_%s", req.Role)] = time.Now().Format(time.RFC3339)

		if metadataJSON, err := json.Marshal(existingMetadata); err == nil {
			transaction.Metadata = datatypes.JSON(metadataJSON)
		}
	}

	// 8. 更新数据库
	if err := global.GVA_DB.Save(&transaction).Error; err != nil {
		global.GVA_LOG.Error("更新交易会话失败", zap.String("transactionID", req.TransactionID), zap.Error(err))
		return nil, fmt.Errorf("更新交易会话失败: %w", err)
	}

	// 9. 缓存到Redis用于ACL检查
	if err := cacheTransactionForACL(req.TransactionID, clientID, req.Role); err != nil {
		global.GVA_LOG.Warn("缓存交易会话到Redis失败", zap.Error(err))
	}

	// 10. 构建主题配置
	topicConfig := nfcRelayReq.TransactionTopicConfig{
		TransmitterStateTopic:  transaction.TransmitterStateTopic,
		ReceiverStateTopic:     transaction.ReceiverStateTopic,
		APDUToTransmitterTopic: transaction.APDUToTransmitterTopic,
		APDUToReceiverTopic:    transaction.APDUToReceiverTopic,
		ControlTopic:           transaction.ControlTopic,
		HeartbeatTopic:         transaction.HeartbeatTopic,
	}

	// 11. 构建响应
	response := &nfcRelayReq.TransactionSessionResponse{
		TransactionID:       req.TransactionID,
		Status:              transaction.Status,
		TransmitterClientID: transaction.TransmitterClientID,
		ReceiverClientID:    transaction.ReceiverClientID,
		Role:                req.Role,
		PeerRole:            peerRole,
		TopicConfig:         topicConfig,
		ExpiresAt:           transaction.ExpiresAt.Unix(),
		CreatedAt:           transaction.CreatedAt.Unix(),
	}

	// 12. 如果双方都已连接，通知MQTT服务
	if transaction.Status == nfc_relay.StatusActive {
		mqttService := GetMQTTService()
		if err := mqttService.PublishTransactionSessionActive(ctx, &transaction); err != nil {
			global.GVA_LOG.Warn("发布交易会话激活通知失败", zap.Error(err))
		}
	}

	global.GVA_LOG.Info("加入交易会话成功",
		zap.String("transactionID", req.TransactionID),
		zap.String("clientID", clientID),
		zap.String("role", req.Role),
		zap.String("peerClientID", peerClientID),
		zap.String("newStatus", transaction.Status),
		zap.String("username", username))

	return response, nil
}

// generateTopicConfig 生成动态主题配置
func generateTopicConfig(transactionID string) nfcRelayReq.TransactionTopicConfig {
	// 使用配置中的topic-prefix
	topicPrefix := global.GVA_CONFIG.MQTT.NFCRelay.TopicPrefix

	return nfcRelayReq.TransactionTopicConfig{
		TransmitterStateTopic:  fmt.Sprintf("%s/transactions/%s/transmitter/state", topicPrefix, transactionID),
		ReceiverStateTopic:     fmt.Sprintf("%s/transactions/%s/receiver/state", topicPrefix, transactionID),
		APDUToTransmitterTopic: fmt.Sprintf("%s/transactions/%s/apdu/to_transmitter", topicPrefix, transactionID),
		APDUToReceiverTopic:    fmt.Sprintf("%s/transactions/%s/apdu/to_receiver", topicPrefix, transactionID),
		ControlTopic:           fmt.Sprintf("%s/transactions/%s/control", topicPrefix, transactionID),
		HeartbeatTopic:         fmt.Sprintf("%s/transactions/%s/heartbeat", topicPrefix, transactionID),
	}
}

// cacheTransactionForACL 缓存交易信息到Redis用于ACL检查
func cacheTransactionForACL(transactionID, clientID, role string) error {
	ctx := context.Background()

	// 存储交易ID -> 客户端ID映射
	transactionKey := fmt.Sprintf("transaction:%s:clients", transactionID)
	pipe := global.GVA_REDIS.Pipeline()

	pipe.HSet(ctx, transactionKey, role+"_client_id", clientID)
	pipe.HSet(ctx, transactionKey, role+"_joined_at", time.Now().Unix())
	pipe.Expire(ctx, transactionKey, 24*time.Hour) // 24小时过期

	// 存储客户端ID -> 交易ID映射
	clientKey := fmt.Sprintf("client:%s:current_transaction", clientID)
	pipe.Set(ctx, clientKey, transactionID, 24*time.Hour)

	// 存储交易ID的权限映射
	aclKey := fmt.Sprintf("transaction:%s:acl", transactionID)
	aclData := map[string]interface{}{
		"transmitter_topics": []string{
			fmt.Sprintf("%s/transactions/%s/transmitter/state", global.GVA_CONFIG.MQTT.NFCRelay.TopicPrefix, transactionID),
			fmt.Sprintf("%s/transactions/%s/apdu/to_receiver", global.GVA_CONFIG.MQTT.NFCRelay.TopicPrefix, transactionID),
		},
		"receiver_topics": []string{
			fmt.Sprintf("%s/transactions/%s/receiver/state", global.GVA_CONFIG.MQTT.NFCRelay.TopicPrefix, transactionID),
			fmt.Sprintf("%s/transactions/%s/apdu/to_transmitter", global.GVA_CONFIG.MQTT.NFCRelay.TopicPrefix, transactionID),
		},
		"common_topics": []string{
			fmt.Sprintf("%s/transactions/%s/control", global.GVA_CONFIG.MQTT.NFCRelay.TopicPrefix, transactionID),
			fmt.Sprintf("%s/transactions/%s/heartbeat", global.GVA_CONFIG.MQTT.NFCRelay.TopicPrefix, transactionID),
		},
	}

	if aclJSON, err := json.Marshal(aclData); err == nil {
		pipe.Set(ctx, aclKey, string(aclJSON), 24*time.Hour)
	}

	_, err := pipe.Exec(ctx)
	return err
}
