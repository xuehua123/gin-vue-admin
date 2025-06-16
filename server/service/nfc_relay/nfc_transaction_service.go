package nfc_relay

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"time"

	"github.com/flipped-aurora/gin-vue-admin/server/global"
	"github.com/flipped-aurora/gin-vue-admin/server/model/common/response"
	"github.com/flipped-aurora/gin-vue-admin/server/model/nfc_relay"
	"github.com/flipped-aurora/gin-vue-admin/server/model/nfc_relay/request"
	"github.com/google/uuid"
	"go.uber.org/zap"
	"gorm.io/datatypes"
	"gorm.io/gorm"
)

type NFCTransactionService struct{}

// CreateTransaction 鍒涘缓浜ゆ槗
func (s *NFCTransactionService) CreateTransaction(ctx context.Context, req *request.CreateTransactionRequest, userID uuid.UUID) (*response.CreateTransactionResponse, error) {
	// 鐢熸垚浜ゆ槗ID
	transactionID, err := s.generateTransactionID()
	if err != nil {
		global.GVA_LOG.Error("鐢熸垚浜ゆ槗ID澶辫触", zap.Error(err))
		return nil, fmt.Errorf("鐢熸垚浜ゆ槗ID澶辫触: %w", err)
	}

	// 妫€鏌ョ敤鎴锋槸鍚︽湁娲昏穬浜ゆ槗锛堝苟鍙戞帶鍒讹級
	lockKey := fmt.Sprintf("user_transaction:%s", userID.String())
	locked, err := s.acquireUserLock(ctx, lockKey, 10*time.Second)
	if err != nil {
		global.GVA_LOG.Error("鑾峰彇鐢ㄦ埛閿佸け璐?, zap.String("userID", userID.String()), zap.Error(err))
		return nil, fmt.Errorf("绯荤粺绻佸繖锛岃绋嶅悗閲嶈瘯")
	}
	if !locked {
		return nil, fmt.Errorf("鎮ㄥ凡鏈夋椿璺冧氦鏄擄紝鏃犳硶鍒涘缓鏂颁氦鏄?)
	}
	defer s.releaseUserLock(ctx, lockKey)

	// 楠岃瘉瀹㈡埛绔槸鍚﹀湪绾?	transmitterOnline, err := s.isClientOnline(ctx, req.TransmitterClientID)
	if err != nil {
		global.GVA_LOG.Error("妫€鏌ヤ紶鍗＄鐘舵€佸け璐?, zap.String("clientID", req.TransmitterClientID), zap.Error(err))
		return nil, fmt.Errorf("妫€鏌ヤ紶鍗＄鐘舵€佸け璐?)
	}

	receiverOnline, err := s.isClientOnline(ctx, req.ReceiverClientID)
	if err != nil {
		global.GVA_LOG.Error("妫€鏌ユ敹鍗＄鐘舵€佸け璐?, zap.String("clientID", req.ReceiverClientID), zap.Error(err))
		return nil, fmt.Errorf("妫€鏌ユ敹鍗＄鐘舵€佸け璐?)
	}

	if !transmitterOnline {
		return nil, fmt.Errorf("浼犲崱绔?%s 涓嶅湪绾?, req.TransmitterClientID)
	}
	if !receiverOnline {
		return nil, fmt.Errorf("鏀跺崱绔?%s 涓嶅湪绾?, req.ReceiverClientID)
	}

	// 澶勭悊鍏冩暟鎹?	var metadata datatypes.JSON
	if req.Metadata != nil {
		metadataBytes, err := json.Marshal(req.Metadata)
		if err != nil {
			global.GVA_LOG.Error("搴忓垪鍖栧厓鏁版嵁澶辫触", zap.Error(err))
			return nil, fmt.Errorf("鍏冩暟鎹牸寮忛敊璇?)
		}
		metadata = datatypes.JSON(metadataBytes)
	}

	// 璁＄畻杩囨湡鏃堕棿
	timeoutSeconds := req.TimeoutSeconds
	if timeoutSeconds == 0 {
		timeoutSeconds = 120 // 榛樿2鍒嗛挓
	}
	expiresAt := time.Now().Add(time.Duration(timeoutSeconds) * time.Second)

	// 鍒涘缓浜ゆ槗璁板綍
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

	// 寮€鍚暟鎹簱浜嬪姟
	err = global.GVA_DB.Transaction(func(tx *gorm.DB) error {
		// 淇濆瓨浜ゆ槗璁板綍
		if err := tx.Create(transaction).Error; err != nil {
			global.GVA_LOG.Error("鍒涘缓浜ゆ槗璁板綍澶辫触", zap.Error(err))
			return fmt.Errorf("鍒涘缓浜ゆ槗璁板綍澶辫触: %w", err)
		}

		// 缂撳瓨浜ゆ槗鐘舵€佸埌Redis
		if err := s.cacheTransactionStatus(ctx, transactionID, nfc_relay.StatusPending, map[string]interface{}{
			"created_at":            transaction.CreatedAt,
			"expires_at":            expiresAt,
			"transmitter_client_id": req.TransmitterClientID,
			"receiver_client_id":    req.ReceiverClientID,
		}); err != nil {
			global.GVA_LOG.Error("缂撳瓨浜ゆ槗鐘舵€佸け璐?, zap.Error(err))
			// 涓嶅奖鍝嶄富娴佺▼锛屽彧璁板綍鏃ュ織
		}

		return nil
	})

	if err != nil {
		return nil, err
	}

	// 鍙戝竷MQTT閫氱煡
	go s.notifyTransactionCreated(ctx, transaction)

	// 杩斿洖鍝嶅簲
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

// UpdateTransactionStatus 鏇存柊浜ゆ槗鐘舵€?func (s *NFCTransactionService) UpdateTransactionStatus(ctx context.Context, req *request.UpdateTransactionStatusRequest, userID uuid.UUID) (*response.UpdateTransactionStatusResponse, error) {
	// 鑾峰彇鐜版湁浜ゆ槗
	var transaction nfc_relay.NFCTransaction
	if err := global.GVA_DB.Where("transaction_id = ?", req.TransactionID).First(&transaction).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, fmt.Errorf("浜ゆ槗涓嶅瓨鍦?)
		}
		global.GVA_LOG.Error("鏌ヨ浜ゆ槗澶辫触", zap.String("transactionID", req.TransactionID), zap.Error(err))
		return nil, fmt.Errorf("鏌ヨ浜ゆ槗澶辫触")
	}

	// 楠岃瘉鐘舵€佽浆鎹?	if !nfc_relay.IsValidStatusTransition(transaction.Status, req.Status) {
		return nil, fmt.Errorf("鏃犳晥鐨勭姸鎬佽浆鎹? %s -> %s", transaction.Status, req.Status)
	}

	previousStatus := transaction.Status
	now := time.Now()

	// 鏇存柊鐘舵€佺浉鍏冲瓧娈?	updates := map[string]interface{}{
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

	// 鏍规嵁鐘舵€佽缃椂闂村瓧娈?	switch req.Status {
	case nfc_relay.StatusActive:
		updates["started_at"] = now
	case nfc_relay.StatusCompleted, nfc_relay.StatusFailed, nfc_relay.StatusCancelled, nfc_relay.StatusTimeout:
		updates["completed_at"] = now
	}

	// 澶勭悊鎵╁睍鍏冩暟鎹?	if req.Metadata != nil {
		metadataBytes, err := json.Marshal(req.Metadata)
		if err != nil {
			global.GVA_LOG.Error("搴忓垪鍖栧厓鏁版嵁澶辫触", zap.Error(err))
			return nil, fmt.Errorf("鍏冩暟鎹牸寮忛敊璇?)
		}
		updates["metadata"] = datatypes.JSON(metadataBytes)
	}

	// 鏇存柊鏁版嵁搴?	if err := global.GVA_DB.Model(&transaction).Updates(updates).Error; err != nil {
		global.GVA_LOG.Error("鏇存柊浜ゆ槗鐘舵€佸け璐?, zap.String("transactionID", req.TransactionID), zap.Error(err))
		return nil, fmt.Errorf("鏇存柊浜ゆ槗鐘舵€佸け璐?)
	}

	// 鏇存柊Redis缂撳瓨
	if err := s.cacheTransactionStatus(ctx, req.TransactionID, req.Status, map[string]interface{}{
		"updated_at":      now,
		"previous_status": previousStatus,
		"reason":          req.Reason,
	}); err != nil {
		global.GVA_LOG.Error("鏇存柊浜ゆ槗鐘舵€佺紦瀛樺け璐?, zap.Error(err))
		// 涓嶅奖鍝嶄富娴佺▼
	}

	// 鍙戝竷MQTT鐘舵€佹洿鏂伴€氱煡
	go s.notifyTransactionStatusUpdate(ctx, &transaction, req.Status, previousStatus, req.Reason)

	// 濡傛灉鏄粓鎬侊紝鎵ц鍚庣画澶勭悊
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

// GetTransaction 鑾峰彇浜ゆ槗璇︽儏
func (s *NFCTransactionService) GetTransaction(ctx context.Context, req *request.GetTransactionRequest, userID uuid.UUID) (*response.TransactionDetailResponse, error) {
	var transaction nfc_relay.NFCTransaction

	query := global.GVA_DB.Where("transaction_id = ?", req.TransactionID)

	// 鍖呭惈APDU娑堟伅
	if req.IncludeAPDU {
		query = query.Preload("APDUMessages", func(db *gorm.DB) *gorm.DB {
			return db.Order("sequence_number ASC")
		})
	}

	if err := query.First(&transaction).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, fmt.Errorf("浜ゆ槗涓嶅瓨鍦?)
		}
		global.GVA_LOG.Error("鏌ヨ浜ゆ槗澶辫触", zap.String("transactionID", req.TransactionID), zap.Error(err))
		return nil, fmt.Errorf("鏌ヨ浜ゆ槗澶辫触")
	}

	// 鏉冮檺妫€鏌ワ紙鍙互鏌ョ湅鑷繁鍒涘缓鐨勪氦鏄擄級
	if transaction.CreatedBy != userID {
		// 杩欓噷鍙互娣诲姞鏇村鏉傜殑鏉冮檺閫昏緫锛屾瘮濡傜鐞嗗憳鍙互鏌ョ湅鎵€鏈変氦鏄?		return nil, fmt.Errorf("鏃犳潈璁块棶姝や氦鏄?)
	}

	// 鑾峰彇缁熻淇℃伅
	statistics, err := s.getTransactionStatistics(ctx, req.TransactionID)
	if err != nil {
		global.GVA_LOG.Error("鑾峰彇浜ゆ槗缁熻澶辫触", zap.Error(err))
		// 缁熻淇℃伅鑾峰彇澶辫触涓嶅奖鍝嶄富娴佺▼
		statistics = response.TransactionStatistics{}
	}

	// 鑾峰彇鏃堕棿绾?	timeline, err := s.getTransactionTimeline(ctx, req.TransactionID)
	if err != nil {
		global.GVA_LOG.Error("鑾峰彇浜ゆ槗鏃堕棿绾垮け璐?, zap.Error(err))
		timeline = []response.TransactionEvent{}
	}

	return &response.TransactionDetailResponse{
		NFCTransaction: transaction,
		Statistics:     statistics,
		Timeline:       timeline,
	}, nil
}

// generateTransactionID 鐢熸垚浜ゆ槗ID
func (s *NFCTransactionService) generateTransactionID() (string, error) {
	bytes := make([]byte, 16)
	if _, err := rand.Read(bytes); err != nil {
		return "", err
	}

	// 鏍煎紡: txn_YYYYMMDD_HEX
	timestamp := time.Now().Format("20060102")
	hexStr := hex.EncodeToString(bytes)[:16]
	return fmt.Sprintf("txn_%s_%s", timestamp, hexStr), nil
}

// acquireUserLock 鑾峰彇鐢ㄦ埛閿侊紙鏀硅繘鐗堟湰锛屾敮鎸侀珮鍙敤锛?func (s *NFCTransactionService) acquireUserLock(ctx context.Context, key string, ttl time.Duration) (bool, error) {
	lockKey := fmt.Sprintf("lock:%s", key)
	identifier := s.generateLockIdentifier()

	// 浣跨敤Lua鑴氭湰纭繚鍘熷瓙鎬ф搷浣?	script := `
		-- 妫€鏌ラ攣鏄惁瀛樺湪
		local existing = redis.call("GET", KEYS[1])
		if existing == false then
			-- 閿佷笉瀛樺湪锛屽彲浠ヨ幏鍙?			redis.call("SET", KEYS[1], ARGV[1], "PX", ARGV[2])
			return 1
		else
			-- 閿佸凡瀛樺湪锛屾鏌ユ槸鍚︽槸鍚屼竴涓寔鏈夎€咃紙鏀寔鍙噸鍏ワ級
			if existing == ARGV[1] then
				-- 寤堕暱閿佺殑杩囨湡鏃堕棿
				redis.call("PEXPIRE", KEYS[1], ARGV[2])
				return 1
			else
				return 0
			end
		end
	`

	result, err := global.GVA_REDIS.Eval(ctx, script, []string{lockKey}, identifier, int64(ttl/time.Millisecond)).Result()
	if err != nil {
		global.GVA_LOG.Error("鑾峰彇鍒嗗竷寮忛攣澶辫触",
			zap.String("lockKey", lockKey),
			zap.Duration("ttl", ttl),
			zap.Error(err))
		return false, fmt.Errorf("鑾峰彇鍒嗗竷寮忛攣澶辫触: %w", err)
	}

	success := result.(int64) == 1
	if success {
		global.GVA_LOG.Debug("鑾峰彇鍒嗗竷寮忛攣鎴愬姛",
			zap.String("lockKey", lockKey),
			zap.String("identifier", identifier),
			zap.Duration("ttl", ttl))

		// 鍦≧edis涓瓨鍌ㄩ攣鐨勫厓鏁版嵁锛屼究浜庣洃鎺у拰璋冭瘯
		metaKey := fmt.Sprintf("lock_meta:%s", key)
		metadata := map[string]interface{}{
			"identifier":  identifier,
			"acquired_at": time.Now().Format(time.RFC3339),
			"expires_at":  time.Now().Add(ttl).Format(time.RFC3339),
			"process_id":  fmt.Sprintf("%d", os.Getpid()),
			"service":     "nfc_transaction",
		}
		global.GVA_REDIS.HMSet(ctx, metaKey, metadata).Err()
		global.GVA_REDIS.Expire(ctx, metaKey, ttl+time.Minute).Err() // 鍏冩暟鎹瘮閿佸淇濈暀1鍒嗛挓
	}

	return success, nil
}

// releaseUserLock 閲婃斁鐢ㄦ埛閿侊紙鏀硅繘鐗堟湰锛?func (s *NFCTransactionService) releaseUserLock(ctx context.Context, key string) error {
	lockKey := fmt.Sprintf("lock:%s", key)
	metaKey := fmt.Sprintf("lock_meta:%s", key)

	// 浣跨敤Lua鑴氭湰纭繚鍘熷瓙鎬ч噴鏀?	script := `
		-- 鑾峰彇褰撳墠閿佺殑鍊?		local current = redis.call("GET", KEYS[1])
		if current == false then
			-- 閿佷笉瀛樺湪
			return 0
		else
			-- 鍒犻櫎閿佸拰鍏冩暟鎹?			redis.call("DEL", KEYS[1])
			redis.call("DEL", KEYS[2])
			return 1
		end
	`

	result, err := global.GVA_REDIS.Eval(ctx, script, []string{lockKey, metaKey}).Result()
	if err != nil {
		global.GVA_LOG.Error("閲婃斁鍒嗗竷寮忛攣澶辫触",
			zap.String("lockKey", lockKey),
			zap.Error(err))
		return fmt.Errorf("閲婃斁鍒嗗竷寮忛攣澶辫触: %w", err)
	}

	released := result.(int64) == 1
	if released {
		global.GVA_LOG.Debug("閲婃斁鍒嗗竷寮忛攣鎴愬姛", zap.String("lockKey", lockKey))
	} else {
		global.GVA_LOG.Warn("灏濊瘯閲婃斁涓嶅瓨鍦ㄧ殑閿?, zap.String("lockKey", lockKey))
	}

	return nil
}

// generateLockIdentifier 鐢熸垚閿佹爣璇嗙
func (s *NFCTransactionService) generateLockIdentifier() string {
	return fmt.Sprintf("%d-%d-%s",
		os.Getpid(),
		time.Now().UnixNano(),
		uuid.New().String()[:8])
}

// isClientOnline 妫€鏌ュ鎴风鏄惁鍦ㄧ嚎锛堟敼杩涚増鏈級
func (s *NFCTransactionService) isClientOnline(ctx context.Context, clientID string) (bool, error) {
	if clientID == "" {
		return false, fmt.Errorf("瀹㈡埛绔疘D涓嶈兘涓虹┖")
	}

	key := fmt.Sprintf("client_heartbeat:%s", clientID)

	// 鑾峰彇蹇冭烦淇℃伅
	heartbeatData, err := global.GVA_REDIS.HGetAll(ctx, key).Result()
	if err != nil {
		if err.Error() == "redis: nil" {
			return false, nil
		}
		global.GVA_LOG.Error("妫€鏌ュ鎴风鍦ㄧ嚎鐘舵€佸け璐?,
			zap.String("clientID", clientID),
			zap.Error(err))
		return false, fmt.Errorf("妫€鏌ュ鎴风鍦ㄧ嚎鐘舵€佸け璐? %w", err)
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
		global.GVA_LOG.Error("瑙ｆ瀽瀹㈡埛绔渶鍚庢椿璺冩椂闂村け璐?,
			zap.String("clientID", clientID),
			zap.String("lastSeen", lastSeenStr),
			zap.Error(err))
		return false, nil
	}

	// 妫€鏌ュ績璺宠秴鏃讹紙60绉掑唴鏈夊績璺宠涓哄湪绾匡級
	isOnline := time.Since(lastSeenTime) < 60*time.Second

	// 璁板綍瀹㈡埛绔姸鎬佹鏌?	global.GVA_LOG.Debug("瀹㈡埛绔湪绾跨姸鎬佹鏌?,
		zap.String("clientID", clientID),
		zap.Bool("isOnline", isOnline),
		zap.Time("lastSeen", lastSeenTime),
		zap.Duration("timeSince", time.Since(lastSeenTime)))

	return isOnline, nil
}

// cacheTransactionStatus 缂撳瓨浜ゆ槗鐘舵€侊紙鏀硅繘鐗堟湰锛?func (s *NFCTransactionService) cacheTransactionStatus(ctx context.Context, transactionID, status string, metadata map[string]interface{}) error {
	if transactionID == "" || status == "" {
		return fmt.Errorf("浜ゆ槗ID鍜岀姸鎬佷笉鑳戒负绌?)
	}

	key := fmt.Sprintf("transaction:%s:status", transactionID)
	now := time.Now()

	// 鏋勫缓缂撳瓨鏁版嵁
	data := map[string]interface{}{
		"transaction_id": transactionID,
		"status":         status,
		"updated_at":     now.Format(time.RFC3339),
		"cached_at":      now.Format(time.RFC3339),
	}

	// 娣诲姞鍏冩暟鎹?	for k, v := range metadata {
		data[k] = v
	}

	// 浣跨敤Pipeline鎻愰珮鎬ц兘
	pipe := global.GVA_REDIS.Pipeline()
	pipe.HMSet(ctx, key, data)
	pipe.Expire(ctx, key, 3600*time.Second) // 1灏忔椂杩囨湡

	// 鍚屾椂缁存姢浜ゆ槗绱㈠紩锛屼究浜庢煡璇㈢敤鎴风殑鎵€鏈変氦鏄?	if userID, ok := metadata["created_by"]; ok {
		userIndexKey := fmt.Sprintf("user_transactions:%v", userID)
		pipe.SAdd(ctx, userIndexKey, transactionID)
		pipe.Expire(ctx, userIndexKey, 86400*time.Second) // 24灏忔椂杩囨湡
	}

	_, err := pipe.Exec(ctx)
	if err != nil {
		global.GVA_LOG.Error("缂撳瓨浜ゆ槗鐘舵€佸け璐?,
			zap.String("transactionID", transactionID),
			zap.String("status", status),
			zap.Error(err))
		return fmt.Errorf("缂撳瓨浜ゆ槗鐘舵€佸け璐? %w", err)
	}

	// 鍙戝竷鐘舵€佸彉鏇翠簨浠跺埌Redis棰戦亾
	publishData := map[string]interface{}{
		"transaction_id": transactionID,
		"status":         status,
		"timestamp":      now.Format(time.RFC3339),
		"metadata":       metadata,
	}

	publishJSON, _ := json.Marshal(publishData)
	global.GVA_REDIS.Publish(ctx, "transaction:status_changed", publishJSON).Err()

	global.GVA_LOG.Debug("浜ゆ槗鐘舵€佺紦瀛樻垚鍔?,
		zap.String("transactionID", transactionID),
		zap.String("status", status),
		zap.Any("metadata", metadata))

	return nil
}

// notifyTransactionCreated 閫氱煡浜ゆ槗鍒涘缓锛堝畬鍠勭増鏈級
func (s *NFCTransactionService) notifyTransactionCreated(ctx context.Context, transaction *nfc_relay.NFCTransaction) {
	// WebSocket瀹炴椂閫氱煡
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

	// MQTT閫氱煡 - 閫氱煡浼犲崱绔?	mqttService := GetMQTTService()
	if err := mqttService.PublishTransactionCreated(ctx, transaction); err != nil {
		global.GVA_LOG.Error("MQTT閫氱煡浼犲崱绔け璐?,
			zap.String("transactionID", transaction.TransactionID),
			zap.String("transmitterClientID", transaction.TransmitterClientID),
			zap.Error(err))
	}

	// 璁板綍鎿嶄綔鏃ュ織
	global.GVA_LOG.Info("浜ゆ槗鍒涘缓閫氱煡宸插彂閫?,
		zap.String("transactionID", transaction.TransactionID),
		zap.String("transmitterClientID", transaction.TransmitterClientID),
		zap.String("receiverClientID", transaction.ReceiverClientID),
		zap.String("status", transaction.Status),
		zap.String("cardType", transaction.CardType))
}

// notifyTransactionStatusUpdate 閫氱煡浜ゆ槗鐘舵€佹洿鏂帮紙瀹屽杽鐗堟湰锛?func (s *NFCTransactionService) notifyTransactionStatusUpdate(ctx context.Context, transaction *nfc_relay.NFCTransaction, newStatus, oldStatus, reason string) {
	// WebSocket瀹炴椂閫氱煡
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

	// MQTT閫氱煡 - 閫氱煡鐩稿叧瀹㈡埛绔姸鎬佸彉鏇?	mqttService := GetMQTTService()

	// 閫氱煡浼犲崱绔?	if err := mqttService.PublishTransactionStatusUpdate(ctx,
		transaction.TransactionID,
		transaction.TransmitterClientID,
		newStatus, oldStatus, reason); err != nil {
		global.GVA_LOG.Error("MQTT閫氱煡浼犲崱绔姸鎬佹洿鏂板け璐?,
			zap.String("transactionID", transaction.TransactionID),
			zap.String("clientID", transaction.TransmitterClientID),
			zap.Error(err))
	}

	// 閫氱煡鏀跺崱绔?	if err := mqttService.PublishTransactionStatusUpdate(ctx,
		transaction.TransactionID,
		transaction.ReceiverClientID,
		newStatus, oldStatus, reason); err != nil {
		global.GVA_LOG.Error("MQTT閫氱煡鏀跺崱绔姸鎬佹洿鏂板け璐?,
			zap.String("transactionID", transaction.TransactionID),
			zap.String("clientID", transaction.ReceiverClientID),
			zap.Error(err))
	}

	// 濡傛灉浜ゆ槗瀹屾垚锛屽彂閫佸畬鎴愪簨浠?	if newStatus == nfc_relay.StatusCompleted ||
		newStatus == nfc_relay.StatusFailed ||
		newStatus == nfc_relay.StatusCancelled ||
		newStatus == nfc_relay.StatusTimeout {
		go s.handleTransactionCompletion(ctx, transaction)
	}

	global.GVA_LOG.Info("浜ゆ槗鐘舵€佹洿鏂伴€氱煡宸插彂閫?,
		zap.String("transactionID", transaction.TransactionID),
		zap.String("oldStatus", oldStatus),
		zap.String("newStatus", newStatus),
		zap.String("reason", reason))
}

// handleTransactionCompletion 澶勭悊浜ゆ槗瀹屾垚锛堝畬鍠勭増鏈級
func (s *NFCTransactionService) handleTransactionCompletion(ctx context.Context, transaction *nfc_relay.NFCTransaction) {
	// 娓呯悊Redis缂撳瓨
	lockKey := fmt.Sprintf("user_transaction:%s", transaction.CreatedBy.String())
	s.releaseUserLock(ctx, lockKey)

	// 娓呯悊浜ゆ槗鐘舵€佺紦瀛?	statusKey := fmt.Sprintf("transaction:%s:status", transaction.TransactionID)
	global.GVA_REDIS.Del(ctx, statusKey).Err()

	// 鏇存柊缁熻鏁版嵁
	go s.updateDailyStatistics(ctx, transaction)

	// 璁板綍瀹¤鏃ュ織
	s.logTransactionCompletion(transaction)

	global.GVA_LOG.Info("浜ゆ槗瀹屾垚澶勭悊",
		zap.String("transactionID", transaction.TransactionID),
		zap.String("status", transaction.Status),
		zap.String("endReason", transaction.EndReason))
}

// updateDailyStatistics 鏇存柊姣忔棩缁熻鏁版嵁
func (s *NFCTransactionService) updateDailyStatistics(ctx context.Context, transaction *nfc_relay.NFCTransaction) {
	today := time.Now().Format("2006-01-02")
	date, _ := time.Parse("2006-01-02", today)

	var stats nfc_relay.NFCTransactionStatistics
	err := global.GVA_DB.Where("date = ?", date).First(&stats).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			// 鍒涘缓鏂扮殑缁熻璁板綍
			stats = nfc_relay.NFCTransactionStatistics{
				Date: date,
			}
		} else {
			global.GVA_LOG.Error("鏌ヨ缁熻鏁版嵁澶辫触", zap.Error(err))
			return
		}
	}

	// 鏇存柊缁熻鏁版嵁
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

	// 鏇存柊APDU娑堟伅缁熻
	stats.TotalAPDUMessages += transaction.APDUCount

	// 鏇存柊澶勭悊鏃堕棿缁熻
	if transaction.TotalProcessingTimeMs > 0 {
		if stats.TotalTransactions == 1 {
			stats.AverageProcessingTimeMs = float64(transaction.TotalProcessingTimeMs)
		} else {
			// 璁＄畻鏂扮殑骞冲潎鍊?			totalTime := (stats.AverageProcessingTimeMs * float64(stats.TotalTransactions-1)) + float64(transaction.TotalProcessingTimeMs)
			stats.AverageProcessingTimeMs = totalTime / float64(stats.TotalTransactions)
		}

		if stats.MinProcessingTimeMs == 0 || transaction.TotalProcessingTimeMs < stats.MinProcessingTimeMs {
			stats.MinProcessingTimeMs = transaction.TotalProcessingTimeMs
		}
		if transaction.TotalProcessingTimeMs > stats.MaxProcessingTimeMs {
			stats.MaxProcessingTimeMs = transaction.TotalProcessingTimeMs
		}
	}

	// 淇濆瓨缁熻鏁版嵁
	if err := global.GVA_DB.Save(&stats).Error; err != nil {
		global.GVA_LOG.Error("淇濆瓨缁熻鏁版嵁澶辫触", zap.Error(err))
	}
}

// logTransactionCompletion 璁板綍浜ゆ槗瀹屾垚瀹¤鏃ュ織
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
	global.GVA_LOG.Info("浜ゆ槗瀹屾垚瀹¤鏃ュ織", zap.String("audit", string(auditJSON)))
}

// getTransactionStatistics 鑾峰彇浜ゆ槗缁熻
func (s *NFCTransactionService) getTransactionStatistics(ctx context.Context, transactionID string) (response.TransactionStatistics, error) {
	var stats response.TransactionStatistics

	// 鏌ヨAPDU娑堟伅缁熻
	var apduCount int64
	if err := global.GVA_DB.Model(&nfc_relay.NFCAPDUMessage{}).
		Where("transaction_id = ?", transactionID).
		Count(&apduCount).Error; err != nil {
		return stats, err
	}

	stats.APDUMessageCount = int(apduCount)

	// 鍙互娣诲姞鏇村缁熻閫昏緫
	return stats, nil
}

// getTransactionTimeline 鑾峰彇浜ゆ槗鏃堕棿绾?func (s *NFCTransactionService) getTransactionTimeline(ctx context.Context, transactionID string) ([]response.TransactionEvent, error) {
	var events []response.TransactionEvent

	// 杩欓噷鍙互浠庢暟鎹簱鎴栨棩蹇椾腑鑾峰彇浜嬩欢鏃堕棿绾?	// 绠€鍖栧疄鐜帮紝杩斿洖绌烘暟缁?	return events, nil
}

// GetTransactionList 鑾峰彇浜ゆ槗鍒楄〃
func (s *NFCTransactionService) GetTransactionList(ctx context.Context, req *request.GetTransactionListRequest, userID uuid.UUID) (*response.TransactionListResponse, error) {
	// 鏋勫缓鏌ヨ鏉′欢
	query := global.GVA_DB.Model(&nfc_relay.NFCTransaction{})

	// 鏉冮檺杩囨护锛氬彧鑳芥煡鐪嬭嚜宸卞垱寤虹殑浜ゆ槗锛堢鐞嗗憳鍙互鏌ョ湅鎵€鏈夛級
	// TODO: 杩欓噷鍙互鏍规嵁鐢ㄦ埛瑙掕壊杩涜鏇寸簿缁嗙殑鏉冮檺鎺у埗
	query = query.Where("created_by = ?", userID)

	// 娣诲姞杩囨护鏉′欢
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

	// 鏃堕棿鑼冨洿杩囨护
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

	// 鍏抽敭璇嶆悳绱?	if req.Keyword != "" {
		keyword := "%" + req.Keyword + "%"
		query = query.Where("description LIKE ? OR tags LIKE ? OR transaction_id LIKE ?", keyword, keyword, keyword)
	}

	// 鑾峰彇鎬绘暟
	var total int64
	if err := query.Count(&total).Error; err != nil {
		global.GVA_LOG.Error("鏌ヨ浜ゆ槗鎬绘暟澶辫触", zap.Error(err))
		return nil, fmt.Errorf("鏌ヨ浜ゆ槗鎬绘暟澶辫触: %w", err)
	}

	// 鎺掑簭
	orderBy := req.OrderBy
	if orderBy == "" {
		orderBy = "created_at"
	}
	order := req.Order
	if order == "" {
		order = "desc"
	}
	query = query.Order(fmt.Sprintf("%s %s", orderBy, order))

	// 鍒嗛〉鏌ヨ
	offset := (req.Page - 1) * req.PageSize
	var transactions []nfc_relay.NFCTransaction
	if err := query.Offset(offset).Limit(req.PageSize).Find(&transactions).Error; err != nil {
		global.GVA_LOG.Error("鏌ヨ浜ゆ槗鍒楄〃澶辫触", zap.Error(err))
		return nil, fmt.Errorf("鏌ヨ浜ゆ槗鍒楄〃澶辫触: %w", err)
	}

	// 杞崲涓哄搷搴旀牸寮?	list := make([]response.TransactionListItem, len(transactions))
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

	// 璁＄畻姹囨€讳俊鎭?	summary := s.calculateSummary(transactions)

	return &response.TransactionListResponse{
		List:     list,
		Total:    total,
		Page:     req.Page,
		PageSize: req.PageSize,
		Summary:  summary,
	}, nil
}

// calculateSummary 璁＄畻浜ゆ槗姹囨€讳俊鎭?func (s *NFCTransactionService) calculateSummary(transactions []nfc_relay.NFCTransaction) response.TransactionSummary {
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

	// 璁＄畻鎴愬姛鐜?	if summary.TotalCount > 0 {
		summary.SuccessRate = float64(summary.CompletedCount) / float64(summary.TotalCount) * 100
	}

	// 璁＄畻骞冲潎澶勭悊鏃堕棿
	if processedCount > 0 {
		summary.AverageProcessingMs = float64(totalProcessingTime) / float64(processedCount)
	}

	return summary
}

// DeleteTransaction 鍒犻櫎浜ゆ槗
func (s *NFCTransactionService) DeleteTransaction(ctx context.Context, req *request.DeleteTransactionRequest, userID uuid.UUID) error {
	// 鏌ヨ浜ゆ槗
	var transaction nfc_relay.NFCTransaction
	if err := global.GVA_DB.Where("transaction_id = ?", req.TransactionID).First(&transaction).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return fmt.Errorf("浜ゆ槗涓嶅瓨鍦?)
		}
		return fmt.Errorf("鏌ヨ浜ゆ槗澶辫触: %w", err)
	}

	// 鏉冮檺妫€鏌?	if transaction.CreatedBy != userID {
		return fmt.Errorf("鏃犳潈鍒犻櫎姝や氦鏄?)
	}

	// 妫€鏌ヤ氦鏄撶姸鎬?	if !req.Force {
		if transaction.Status == nfc_relay.StatusActive || transaction.Status == nfc_relay.StatusProcessing {
			return fmt.Errorf("鏃犳硶鍒犻櫎娲昏穬鐘舵€佺殑浜ゆ槗锛岃鍏堝彇娑堜氦鏄撴垨浣跨敤寮哄埗鍒犻櫎")
		}
	}

	// 寮€鍚簨鍔″垹闄?	return global.GVA_DB.Transaction(func(tx *gorm.DB) error {
		// 鍒犻櫎鍏宠仈鐨凙PDU娑堟伅
		if err := tx.Where("transaction_id = ?", req.TransactionID).Delete(&nfc_relay.NFCAPDUMessage{}).Error; err != nil {
			return fmt.Errorf("鍒犻櫎APDU娑堟伅澶辫触: %w", err)
		}

		// 鍒犻櫎浜ゆ槗璁板綍
		if err := tx.Delete(&transaction).Error; err != nil {
			return fmt.Errorf("鍒犻櫎浜ゆ槗澶辫触: %w", err)
		}

		// 娓呯悊Redis缂撳瓨
		s.cleanupTransactionCache(ctx, req.TransactionID, userID)

		return nil
	})
}

// cleanupTransactionCache 娓呯悊浜ゆ槗鐩稿叧鐨凴edis缂撳瓨
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

// SendAPDU 鍙戦€丄PDU娑堟伅
func (s *NFCTransactionService) SendAPDU(ctx context.Context, req *request.SendAPDURequest, userID uuid.UUID) (*response.SendAPDUResponse, error) {
	// 楠岃瘉浜ゆ槗
	var transaction nfc_relay.NFCTransaction
	if err := global.GVA_DB.Where("transaction_id = ?", req.TransactionID).First(&transaction).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, fmt.Errorf("浜ゆ槗涓嶅瓨鍦?)
		}
		return nil, fmt.Errorf("鏌ヨ浜ゆ槗澶辫触: %w", err)
	}

	// 鏉冮檺妫€鏌?	if transaction.CreatedBy != userID {
		return nil, fmt.Errorf("鏃犳潈鎿嶄綔姝や氦鏄?)
	}

	// 鐘舵€佹鏌?	if transaction.Status != nfc_relay.StatusActive && transaction.Status != nfc_relay.StatusProcessing {
		return nil, fmt.Errorf("浜ゆ槗鐘舵€佷笉鏀寔鍙戦€丄PDU娑堟伅: %s", transaction.Status)
	}

	// 鍒涘缓APDU娑堟伅璁板綍
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

	// 澶勭悊鍏冩暟鎹?	if req.Metadata != nil {
		if metadataJSON, err := json.Marshal(req.Metadata); err == nil {
			apduMessage.Metadata = datatypes.JSON(metadataJSON)
		}
	}

	// 淇濆瓨鍒版暟鎹簱
	if err := global.GVA_DB.Create(apduMessage).Error; err != nil {
		return nil, fmt.Errorf("淇濆瓨APDU娑堟伅澶辫触: %w", err)
	}

	// 閫氳繃MQTT鍙戦€佸埌瀹㈡埛绔?	mqttService := GetMQTTService()
	mqttMsg := APDUMessage{
		TransactionID:  req.TransactionID,
		SequenceNumber: req.SequenceNumber,
		Direction:      req.Direction,
		APDUHex:        req.APDUHex,
		Priority:       req.Priority,
		MessageType:    req.MessageType,
		Timeout:        30, // 榛樿30绉掕秴鏃?	}

	// 纭畾鐩爣瀹㈡埛绔?	var targetClientID string
	if req.Direction == nfc_relay.DirectionToReceiver {
		targetClientID = transaction.ReceiverClientID
	} else {
		targetClientID = transaction.TransmitterClientID
	}

	if err := mqttService.SendAPDUToClient(ctx, targetClientID, mqttMsg); err != nil {
		// 鏇存柊娑堟伅鐘舵€佷负澶辫触
		global.GVA_DB.Model(apduMessage).Update("status", nfc_relay.MessageStatusFailed)
		return nil, fmt.Errorf("鍙戦€丄PDU娑堟伅鍒板鎴风澶辫触: %w", err)
	}

	// 鏇存柊娑堟伅鐘舵€佷负宸插彂閫?	global.GVA_DB.Model(apduMessage).Update("status", nfc_relay.MessageStatusSent)

	// 鏇存柊浜ゆ槗鐨凙PDU璁℃暟
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

// GetAPDUList 鑾峰彇APDU娑堟伅鍒楄〃
func (s *NFCTransactionService) GetAPDUList(ctx context.Context, req *request.GetAPDUListRequest, userID uuid.UUID) (*response.APDUMessageListResponse, error) {
	// 楠岃瘉浜ゆ槗鏉冮檺
	var transaction nfc_relay.NFCTransaction
	if err := global.GVA_DB.Where("transaction_id = ?", req.TransactionID).First(&transaction).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, fmt.Errorf("浜ゆ槗涓嶅瓨鍦?)
		}
		return nil, fmt.Errorf("鏌ヨ浜ゆ槗澶辫触: %w", err)
	}

	if transaction.CreatedBy != userID {
		return nil, fmt.Errorf("鏃犳潈璁块棶姝や氦鏄撶殑APDU娑堟伅")
	}

	// 鏋勫缓鏌ヨ
	query := global.GVA_DB.Model(&nfc_relay.NFCAPDUMessage{}).Where("transaction_id = ?", req.TransactionID)

	// 杩囨护鏉′欢
	if req.Direction != "" {
		query = query.Where("direction = ?", req.Direction)
	}
	if req.Status != "" {
		query = query.Where("status = ?", req.Status)
	}
	if req.Priority != "" {
		query = query.Where("priority = ?", req.Priority)
	}

	// 鏃堕棿鑼冨洿
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

	// 鑾峰彇鎬绘暟
	var total int64
	if err := query.Count(&total).Error; err != nil {
		return nil, fmt.Errorf("鏌ヨAPDU娑堟伅鎬绘暟澶辫触: %w", err)
	}

	// 鍒嗛〉鏌ヨ
	offset := (req.Page - 1) * req.PageSize
	var messages []nfc_relay.NFCAPDUMessage
	if err := query.Order("sequence_number ASC").Offset(offset).Limit(req.PageSize).Find(&messages).Error; err != nil {
		return nil, fmt.Errorf("鏌ヨAPDU娑堟伅鍒楄〃澶辫触: %w", err)
	}

	// 杞崲涓哄搷搴旀牸寮?	list := make([]response.APDUMessageItem, len(messages))
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

// GetStatistics 鑾峰彇缁熻淇℃伅
func (s *NFCTransactionService) GetStatistics(ctx context.Context, req *request.GetStatisticsRequest, userID uuid.UUID) (*response.TransactionStatisticsResponse, error) {
	// 瑙ｆ瀽鏃ユ湡鑼冨洿
	startDate, err := time.Parse("2006-01-02", req.StartDate)
	if err != nil {
		return nil, fmt.Errorf("鏃犳晥鐨勫紑濮嬫棩鏈熸牸寮? %w", err)
	}

	endDate, err := time.Parse("2006-01-02", req.EndDate)
	if err != nil {
		return nil, fmt.Errorf("鏃犳晥鐨勭粨鏉熸棩鏈熸牸寮? %w", err)
	}

	// 纭繚缁撴潫鏃ユ湡鍖呭惈鏁村ぉ
	endDate = endDate.Add(24 * time.Hour).Add(-1 * time.Second)

	// 鏋勫缓鍩虹鏌ヨ
	baseQuery := global.GVA_DB.Model(&nfc_relay.NFCTransaction{}).
		Where("created_by = ?", userID).
		Where("created_at BETWEEN ? AND ?", startDate, endDate)

	// 娣诲姞杩囨护鏉′欢
	if req.CardType != "" {
		baseQuery = baseQuery.Where("card_type = ?", req.CardType)
	}
	if req.Status != "" {
		baseQuery = baseQuery.Where("status = ?", req.Status)
	}

	// 鑾峰彇姹囨€荤粺璁?	summary, err := s.calculateStatisticsSummary(baseQuery)
	if err != nil {
		return nil, fmt.Errorf("璁＄畻缁熻姹囨€诲け璐? %w", err)
	}

	// 鑾峰彇姣忔棩缁熻
	dailyStats, err := s.calculateDailyStatistics(baseQuery, startDate, endDate, req.GroupBy)
	if err != nil {
		return nil, fmt.Errorf("璁＄畻姣忔棩缁熻澶辫触: %w", err)
	}

	// 鐢熸垚鍥捐〃鏁版嵁
	chartData := s.generateChartData(dailyStats)

	// 鑾峰彇瀹㈡埛绔粺璁?	topClients, err := s.getTopClientsStatistics(baseQuery)
	if err != nil {
		return nil, fmt.Errorf("鑾峰彇瀹㈡埛绔粺璁″け璐? %w", err)
	}

	// 閿欒鍒嗘瀽
	errorAnalysis, err := s.getErrorAnalysis(baseQuery)
	if err != nil {
		return nil, fmt.Errorf("鑾峰彇閿欒鍒嗘瀽澶辫触: %w", err)
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

// calculateStatisticsSummary 璁＄畻缁熻姹囨€?func (s *NFCTransactionService) calculateStatisticsSummary(query *gorm.DB) (response.StatisticsSummary, error) {
	var summary response.StatisticsSummary

	// 鎬讳氦鏄撴暟 - 淇绫诲瀷涓嶅尮閰嶉棶棰?	var totalTransactions int64
	if err := query.Count(&totalTransactions).Error; err != nil {
		return summary, err
	}
	summary.TotalTransactions = int(totalTransactions)

	// 鍚勭姸鎬佷氦鏄撴暟
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

	// 璁＄畻鎴愬姛鐜?	if summary.TotalTransactions > 0 {
		summary.SuccessRate = float64(summary.SuccessfulTransactions) / float64(summary.TotalTransactions) * 100
	}

	// APDU娑堟伅鎬绘暟鍜屽钩鍧囧鐞嗘椂闂?	var aggregates struct {
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

// calculateDailyStatistics 璁＄畻姣忔棩/姣忓皬鏃剁粺璁℃暟鎹?func (s *NFCTransactionService) calculateDailyStatistics(query *gorm.DB, startDate, endDate time.Time, groupBy string) ([]response.DailyStatistics, error) {
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
		global.GVA_LOG.Error("璁＄畻姣忔棩缁熻澶辫触", zap.Error(err))
		return nil, fmt.Errorf("璁＄畻姣忔棩缁熻澶辫触: %w", err)
	}
	return dailyStats, nil
}

// generateChartData 鐢熸垚鍥捐〃鏁版嵁
func (s *NFCTransactionService) generateChartData(dailyStats []response.DailyStatistics) response.StatisticsChartData {
	var chartData response.StatisticsChartData

	// 瓒嬪娍鏁版嵁
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

	// 鐘舵€佸垎甯冿紙杩欓噷绠€鍖栧鐞嗭紝瀹為檯搴旇浠庢暟鎹簱缁熻锛?	chartData.StatusDistribution = []response.PieChartItem{
		{Name: "completed", Value: 70, Count: 70},
		{Name: "failed", Value: 20, Count: 20},
		{Name: "pending", Value: 10, Count: 10},
	}

	return chartData
}

// getTopClientsStatistics 鑾峰彇瀹㈡埛绔粺璁?func (s *NFCTransactionService) getTopClientsStatistics(query *gorm.DB) ([]response.ClientStatistics, error) {
	// 绠€鍖栧疄鐜帮紝杩斿洖绌烘暟缁?	return []response.ClientStatistics{}, nil
}

// getErrorAnalysis 鑾峰彇閿欒鍒嗘瀽
func (s *NFCTransactionService) getErrorAnalysis(query *gorm.DB) (response.ErrorAnalysis, error) {
	var analysis response.ErrorAnalysis

	// 缁熻閿欒鎬绘暟
	var errorCount int64
	query.Where("status = ?", nfc_relay.StatusFailed).Count(&errorCount)
	analysis.TotalErrors = int(errorCount)

	// 璁＄畻閿欒鐜?	var totalCount int64
	query.Count(&totalCount)
	if totalCount > 0 {
		analysis.ErrorRate = float64(errorCount) / float64(totalCount) * 100
	}

	return analysis, nil
}

// BatchUpdateTransactionStatus 鎵归噺鏇存柊浜ゆ槗鐘舵€?func (s *NFCTransactionService) BatchUpdateTransactionStatus(ctx context.Context, req *request.BatchUpdateTransactionRequest, userID uuid.UUID) (*response.BatchOperationResponse, error) {
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
