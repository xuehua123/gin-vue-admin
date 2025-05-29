package security

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"time"

	"github.com/flipped-aurora/gin-vue-admin/server/global"
	"go.uber.org/zap"
)

// APDUEncryption APDU加密管理器
type APDUEncryption struct {
	keyManager *KeyExchangeManager
}

// EncryptedAPDU 加密的APDU结构
type EncryptedAPDU struct {
	SessionID     string `json:"sessionId"`
	EncryptedData string `json:"encryptedData"` // Base64编码的加密数据
	Nonce         string `json:"nonce"`         // Base64编码的随机数
	AuthTag       string `json:"authTag"`       // Base64编码的认证标签
	Timestamp     int64  `json:"timestamp"`     // 时间戳
	Version       string `json:"version"`       // 加密版本
}

// PlainAPDU 明文APDU结构
type PlainAPDU struct {
	CommandAPDU  []byte            `json:"commandApdu,omitempty"`  // 命令APDU
	ResponseAPDU []byte            `json:"responseApdu,omitempty"` // 响应APDU
	Metadata     map[string]string `json:"metadata,omitempty"`     // 元数据
	Timestamp    int64             `json:"timestamp"`              // 时间戳
}

// NewAPDUEncryption 创建APDU加密管理器
func NewAPDUEncryption(keyManager *KeyExchangeManager) *APDUEncryption {
	return &APDUEncryption{
		keyManager: keyManager,
	}
}

// EncryptCommandAPDU 加密命令APDU (收卡端 -> 传卡端)
func (ae *APDUEncryption) EncryptCommandAPDU(sessionID string, commandAPDU []byte, metadata map[string]string) (*EncryptedAPDU, error) {
	// 获取会话密钥
	sessionKeys, err := ae.keyManager.GetSessionKeys(sessionID)
	if err != nil {
		return nil, fmt.Errorf("获取会话密钥失败: %w", err)
	}

	// 构造明文APDU结构
	plainAPDU := &PlainAPDU{
		CommandAPDU: commandAPDU,
		Metadata:    metadata,
		Timestamp:   time.Now().Unix(),
	}

	// 加密APDU
	encryptedData, err := ae.encryptAPDUData(sessionKeys.EncryptionKey, plainAPDU)
	if err != nil {
		return nil, fmt.Errorf("加密APDU失败: %w", err)
	}

	global.GVA_LOG.Debug("命令APDU已加密",
		zap.String("sessionId", sessionID),
		zap.Int("originalSize", len(commandAPDU)),
		zap.Int("encryptedSize", len(encryptedData.EncryptedData)),
	)

	return encryptedData, nil
}

// DecryptCommandAPDU 解密命令APDU (传卡端接收)
func (ae *APDUEncryption) DecryptCommandAPDU(sessionID string, encryptedAPDU *EncryptedAPDU) (*PlainAPDU, error) {
	// 获取会话密钥
	sessionKeys, err := ae.keyManager.GetSessionKeys(sessionID)
	if err != nil {
		return nil, fmt.Errorf("获取会话密钥失败: %w", err)
	}

	// 解密APDU
	plainAPDU, err := ae.decryptAPDUData(sessionKeys.EncryptionKey, encryptedAPDU)
	if err != nil {
		return nil, fmt.Errorf("解密APDU失败: %w", err)
	}

	// 验证时间戳 (防重放)
	if err := ae.validateTimestamp(plainAPDU.Timestamp); err != nil {
		return nil, fmt.Errorf("时间戳验证失败: %w", err)
	}

	global.GVA_LOG.Debug("命令APDU已解密",
		zap.String("sessionId", sessionID),
		zap.Int("apduSize", len(plainAPDU.CommandAPDU)),
	)

	return plainAPDU, nil
}

// EncryptResponseAPDU 加密响应APDU (传卡端 -> 收卡端)
func (ae *APDUEncryption) EncryptResponseAPDU(sessionID string, responseAPDU []byte, metadata map[string]string) (*EncryptedAPDU, error) {
	// 获取会话密钥
	sessionKeys, err := ae.keyManager.GetSessionKeys(sessionID)
	if err != nil {
		return nil, fmt.Errorf("获取会话密钥失败: %w", err)
	}

	// 构造明文APDU结构
	plainAPDU := &PlainAPDU{
		ResponseAPDU: responseAPDU,
		Metadata:     metadata,
		Timestamp:    time.Now().Unix(),
	}

	// 加密APDU
	encryptedData, err := ae.encryptAPDUData(sessionKeys.EncryptionKey, plainAPDU)
	if err != nil {
		return nil, fmt.Errorf("加密响应APDU失败: %w", err)
	}

	global.GVA_LOG.Debug("响应APDU已加密",
		zap.String("sessionId", sessionID),
		zap.Int("originalSize", len(responseAPDU)),
		zap.Int("encryptedSize", len(encryptedData.EncryptedData)),
	)

	return encryptedData, nil
}

// DecryptResponseAPDU 解密响应APDU (收卡端接收)
func (ae *APDUEncryption) DecryptResponseAPDU(sessionID string, encryptedAPDU *EncryptedAPDU) (*PlainAPDU, error) {
	// 获取会话密钥
	sessionKeys, err := ae.keyManager.GetSessionKeys(sessionID)
	if err != nil {
		return nil, fmt.Errorf("获取会话密钥失败: %w", err)
	}

	// 解密APDU
	plainAPDU, err := ae.decryptAPDUData(sessionKeys.EncryptionKey, encryptedAPDU)
	if err != nil {
		return nil, fmt.Errorf("解密响应APDU失败: %w", err)
	}

	// 验证时间戳
	if err := ae.validateTimestamp(plainAPDU.Timestamp); err != nil {
		return nil, fmt.Errorf("时间戳验证失败: %w", err)
	}

	global.GVA_LOG.Debug("响应APDU已解密",
		zap.String("sessionId", sessionID),
		zap.Int("apduSize", len(plainAPDU.ResponseAPDU)),
	)

	return plainAPDU, nil
}

// encryptAPDUData 使用AES-GCM加密APDU数据
func (ae *APDUEncryption) encryptAPDUData(key []byte, plainAPDU *PlainAPDU) (*EncryptedAPDU, error) {
	// 将APDU结构序列化为JSON
	jsonData, err := json.Marshal(plainAPDU)
	if err != nil {
		return nil, fmt.Errorf("序列化APDU失败: %w", err)
	}

	// 创建AES加密器
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, fmt.Errorf("创建AES加密器失败: %w", err)
	}

	// 创建GCM模式
	aesGCM, err := cipher.NewGCM(block)
	if err != nil {
		return nil, fmt.Errorf("创建GCM模式失败: %w", err)
	}

	// 生成随机nonce
	nonce := make([]byte, aesGCM.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return nil, fmt.Errorf("生成nonce失败: %w", err)
	}

	// 加密数据
	ciphertext := aesGCM.Seal(nil, nonce, jsonData, nil)

	// 分离密文和认证标签
	tagSize := aesGCM.Overhead()

	encryptedData := ciphertext[:len(ciphertext)-tagSize]
	authTag := ciphertext[len(ciphertext)-tagSize:]

	return &EncryptedAPDU{
		SessionID:     plainAPDU.Metadata["sessionId"],
		EncryptedData: base64.StdEncoding.EncodeToString(encryptedData),
		Nonce:         base64.StdEncoding.EncodeToString(nonce),
		AuthTag:       base64.StdEncoding.EncodeToString(authTag),
		Timestamp:     time.Now().Unix(),
		Version:       "AES-256-GCM-v1",
	}, nil
}

// decryptAPDUData 使用AES-GCM解密APDU数据
func (ae *APDUEncryption) decryptAPDUData(key []byte, encryptedAPDU *EncryptedAPDU) (*PlainAPDU, error) {
	// 验证加密版本
	if encryptedAPDU.Version != "AES-256-GCM-v1" {
		return nil, errors.New("不支持的加密版本")
	}

	// 解码Base64数据
	encryptedData, err := base64.StdEncoding.DecodeString(encryptedAPDU.EncryptedData)
	if err != nil {
		return nil, fmt.Errorf("解码加密数据失败: %w", err)
	}

	nonce, err := base64.StdEncoding.DecodeString(encryptedAPDU.Nonce)
	if err != nil {
		return nil, fmt.Errorf("解码nonce失败: %w", err)
	}

	authTag, err := base64.StdEncoding.DecodeString(encryptedAPDU.AuthTag)
	if err != nil {
		return nil, fmt.Errorf("解码认证标签失败: %w", err)
	}

	// 重组密文
	ciphertext := append(encryptedData, authTag...)

	// 创建AES解密器
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, fmt.Errorf("创建AES解密器失败: %w", err)
	}

	// 创建GCM模式
	aesGCM, err := cipher.NewGCM(block)
	if err != nil {
		return nil, fmt.Errorf("创建GCM模式失败: %w", err)
	}

	// 解密数据
	plaintext, err := aesGCM.Open(nil, nonce, ciphertext, nil)
	if err != nil {
		return nil, fmt.Errorf("解密失败: %w", err)
	}

	// 反序列化JSON
	var plainAPDU PlainAPDU
	if err := json.Unmarshal(plaintext, &plainAPDU); err != nil {
		return nil, fmt.Errorf("反序列化APDU失败: %w", err)
	}

	return &plainAPDU, nil
}

// validateTimestamp 验证时间戳 (防重放)
func (ae *APDUEncryption) validateTimestamp(timestamp int64) error {
	now := time.Now().Unix()
	maxAge := int64(30) // 30秒窗口

	if timestamp > now+maxAge {
		return errors.New("时间戳过于超前")
	}

	if timestamp < now-maxAge {
		return errors.New("时间戳过于陈旧")
	}

	return nil
}

// 全局APDU加密管理器
var GlobalAPDUEncryption = NewAPDUEncryption(GlobalKeyExchangeManager)
