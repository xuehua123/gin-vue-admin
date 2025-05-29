package security

import (
	"crypto/rand"
	"crypto/sha256"
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/flipped-aurora/gin-vue-admin/server/global"
	"go.uber.org/zap"
)

// SessionKeys 会话密钥结构
type SessionKeys struct {
	EncryptionKey []byte    // AES加密密钥 (32字节 for AES-256)
	MACKey        []byte    // HMAC密钥 (32字节)
	KeyID         string    // 密钥ID
	CreatedAt     time.Time // 密钥创建时间
	ExpiresAt     time.Time // 密钥过期时间
}

// KeyExchangeManager 密钥交换管理器
type KeyExchangeManager struct {
	// 会话密钥存储
	sessionKeys map[string]*SessionKeys
	mutex       sync.RWMutex

	// 主密钥 (从配置或HSM获取)
	masterKey []byte

	// 密钥生命周期配置
	keyRotationInterval time.Duration
	keyExpiryTime       time.Duration
}

// NewKeyExchangeManager 创建新的密钥交换管理器
func NewKeyExchangeManager() *KeyExchangeManager {
	masterKey := make([]byte, 32) // AES-256主密钥
	if _, err := rand.Read(masterKey); err != nil {
		panic("生成主密钥失败: " + err.Error())
	}

	return &KeyExchangeManager{
		sessionKeys:         make(map[string]*SessionKeys),
		masterKey:           masterKey,
		keyRotationInterval: 24 * time.Hour, // 24小时轮换
		keyExpiryTime:       48 * time.Hour, // 48小时过期
	}
}

// GenerateSessionKeys 为会话生成新的密钥对
func (kem *KeyExchangeManager) GenerateSessionKeys(sessionID string) (*SessionKeys, error) {
	kem.mutex.Lock()
	defer kem.mutex.Unlock()

	// 检查是否已存在有效密钥
	if existingKeys, exists := kem.sessionKeys[sessionID]; exists {
		if time.Now().Before(existingKeys.ExpiresAt) {
			global.GVA_LOG.Debug("使用现有会话密钥",
				zap.String("sessionId", sessionID),
				zap.String("keyId", existingKeys.KeyID),
			)
			return existingKeys, nil
		}
		// 密钥已过期，删除旧密钥
		delete(kem.sessionKeys, sessionID)
	}

	// 生成新的会话密钥
	encryptionKey := make([]byte, 32) // AES-256
	macKey := make([]byte, 32)        // HMAC-SHA256

	if _, err := rand.Read(encryptionKey); err != nil {
		return nil, fmt.Errorf("生成加密密钥失败: %w", err)
	}

	if _, err := rand.Read(macKey); err != nil {
		return nil, fmt.Errorf("生成MAC密钥失败: %w", err)
	}

	// 使用会话ID和时间戳生成密钥ID
	keyID := kem.generateKeyID(sessionID)

	sessionKeys := &SessionKeys{
		EncryptionKey: encryptionKey,
		MACKey:        macKey,
		KeyID:         keyID,
		CreatedAt:     time.Now(),
		ExpiresAt:     time.Now().Add(kem.keyExpiryTime),
	}

	// 存储会话密钥
	kem.sessionKeys[sessionID] = sessionKeys

	global.GVA_LOG.Info("生成新的会话密钥",
		zap.String("sessionId", sessionID),
		zap.String("keyId", keyID),
		zap.Time("expiresAt", sessionKeys.ExpiresAt),
	)

	return sessionKeys, nil
}

// GetSessionKeys 获取会话密钥
func (kem *KeyExchangeManager) GetSessionKeys(sessionID string) (*SessionKeys, error) {
	kem.mutex.RLock()
	defer kem.mutex.RUnlock()

	sessionKeys, exists := kem.sessionKeys[sessionID]
	if !exists {
		return nil, errors.New("会话密钥不存在")
	}

	// 检查密钥是否过期
	if time.Now().After(sessionKeys.ExpiresAt) {
		return nil, errors.New("会话密钥已过期")
	}

	return sessionKeys, nil
}

// RevokeSessionKeys 撤销会话密钥
func (kem *KeyExchangeManager) RevokeSessionKeys(sessionID string) error {
	kem.mutex.Lock()
	defer kem.mutex.Unlock()

	if _, exists := kem.sessionKeys[sessionID]; !exists {
		return errors.New("会话密钥不存在")
	}

	delete(kem.sessionKeys, sessionID)

	global.GVA_LOG.Info("撤销会话密钥",
		zap.String("sessionId", sessionID),
	)

	return nil
}

// CleanupExpiredKeys 清理过期的密钥
func (kem *KeyExchangeManager) CleanupExpiredKeys() {
	kem.mutex.Lock()
	defer kem.mutex.Unlock()

	now := time.Now()
	expiredSessions := make([]string, 0)

	for sessionID, keys := range kem.sessionKeys {
		if now.After(keys.ExpiresAt) {
			expiredSessions = append(expiredSessions, sessionID)
		}
	}

	for _, sessionID := range expiredSessions {
		delete(kem.sessionKeys, sessionID)
	}

	if len(expiredSessions) > 0 {
		global.GVA_LOG.Info("清理过期会话密钥",
			zap.Int("expiredCount", len(expiredSessions)),
			zap.Strings("expiredSessions", expiredSessions),
		)
	}
}

// StartKeyRotation 启动密钥轮换定时器
func (kem *KeyExchangeManager) StartKeyRotation() {
	go func() {
		ticker := time.NewTicker(kem.keyRotationInterval)
		defer ticker.Stop()

		for range ticker.C {
			kem.CleanupExpiredKeys()
		}
	}()

	global.GVA_LOG.Info("密钥轮换定时器已启动",
		zap.Duration("interval", kem.keyRotationInterval),
	)
}

// GetActiveSessionCount 获取活跃会话密钥数量
func (kem *KeyExchangeManager) GetActiveSessionCount() int {
	kem.mutex.RLock()
	defer kem.mutex.RUnlock()

	now := time.Now()
	activeCount := 0

	for _, keys := range kem.sessionKeys {
		if now.Before(keys.ExpiresAt) {
			activeCount++
		}
	}

	return activeCount
}

// generateKeyID 生成密钥ID
func (kem *KeyExchangeManager) generateKeyID(sessionID string) string {
	data := fmt.Sprintf("%s:%d", sessionID, time.Now().UnixNano())
	hash := sha256.Sum256([]byte(data))
	return fmt.Sprintf("key_%x", hash[:8]) // 使用前8字节作为ID
}

// 全局密钥交换管理器实例
var GlobalKeyExchangeManager = NewKeyExchangeManager()
