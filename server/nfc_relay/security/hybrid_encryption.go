package security

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"strings"
	"time"

	"github.com/flipped-aurora/gin-vue-admin/server/global"
	"go.uber.org/zap"
)

// DataSensitivityLevel 数据敏感级别
type DataSensitivityLevel int

const (
	SensitivityPublic DataSensitivityLevel = iota // 公开数据，服务器可读
	SensitivityAudit                              // 审计数据，服务器可解密用于合规检查
)

// APDUDataClass APDU数据分类
type APDUDataClass struct {
	// 审计数据 - 服务器可读，用于合规检查
	AuditData AuditableData `json:"auditData"`

	// 业务数据 - 服务器可解密，用于合规检查
	BusinessData EncryptedBusinessData `json:"businessData"`

	// 元数据 - 明文，用于路由和监控
	Metadata APDUMetadata `json:"metadata"`
}

// AuditableData 可审计的数据结构
type AuditableData struct {
	CommandClass     string    `json:"commandClass"`     // 命令类别 (SELECT, READ, WRITE等)
	CommandType      string    `json:"commandType"`      // 具体命令类型
	ApplicationID    string    `json:"applicationId"`    // 应用ID (脱敏后)
	TransactionType  string    `json:"transactionType"`  // 交易类型
	Amount           *int64    `json:"amount,omitempty"` // 交易金额 (分为单位)
	Currency         string    `json:"currency"`         // 货币代码
	MerchantCategory string    `json:"merchantCategory"` // 商户类别
	RiskScore        int       `json:"riskScore"`        // 风险评分
	Timestamp        time.Time `json:"timestamp"`        // 时间戳
}

// EncryptedBusinessData 加密的业务数据
type EncryptedBusinessData struct {
	// 所有业务数据都使用审计密钥加密，服务器可解密进行合规检查
	EncryptedData string `json:"encryptedData"` // 审计密钥加密的所有业务数据

	// 加密元信息
	EncryptionInfo EncryptionInfo `json:"encryptionInfo"` // 加密信息
}

// EncryptionInfo 加密信息
type EncryptionInfo struct {
	Algorithm string `json:"algorithm"` // 加密算法
	Nonce     string `json:"nonce"`     // 随机数
	Tag       string `json:"tag"`       // 认证标签
}

// APDUMetadata APDU元数据
type APDUMetadata struct {
	SessionID   string    `json:"sessionId"`
	SequenceNum int64     `json:"sequenceNum"` // 序列号，防重放
	Direction   string    `json:"direction"`   // "command" 或 "response"
	Timestamp   time.Time `json:"timestamp"`
	ClientID    string    `json:"clientId"`
	UserID      string    `json:"userId"`
	DeviceInfo  string    `json:"deviceInfo"`  // 设备指纹
	ChecksumCRC string    `json:"checksumCrc"` // 数据完整性校验
}

// HybridEncryptionManager 混合加密管理器
type HybridEncryptionManager struct {
	auditKeyManager *AuditKeyManager       // 审计密钥管理器
	auditEngine     *ComplianceAuditEngine // 合规审计引擎
}

// AuditKeyManager 审计密钥管理器
type AuditKeyManager struct {
	masterKey    []byte // 服务器主密钥
	rotationTime time.Duration
	currentKeyID string
}

// NewHybridEncryptionManager 创建混合加密管理器
func NewHybridEncryptionManager() *HybridEncryptionManager {
	return &HybridEncryptionManager{
		auditKeyManager: NewAuditKeyManager(),
		auditEngine:     NewComplianceAuditEngine(),
	}
}

// NewAuditKeyManager 创建审计密钥管理器
func NewAuditKeyManager() *AuditKeyManager {
	// 从配置或HSM获取主密钥
	masterKey := make([]byte, 32) // AES-256
	if _, err := rand.Read(masterKey); err != nil {
		panic("生成审计主密钥失败: " + err.Error())
	}

	return &AuditKeyManager{
		masterKey:    masterKey,
		rotationTime: 24 * time.Hour, // 24小时轮换
		currentKeyID: generateKeyID(),
	}
}

// EncryptAPDUForTransmission 加密APDU用于传输
func (hem *HybridEncryptionManager) EncryptAPDUForTransmission(
	sessionID string,
	rawAPDU []byte,
	metadata APDUMetadata,
	userID string,
) (*APDUDataClass, error) {

	// 1. 解析和分类APDU数据
	auditData, businessData, err := hem.classifyAPDUData(rawAPDU, metadata)
	if err != nil {
		return nil, fmt.Errorf("APDU数据分类失败: %w", err)
	}

	// 2. 准备加密的业务数据
	encryptedBusinessData, err := hem.encryptBusinessData(sessionID, businessData, userID)
	if err != nil {
		return nil, fmt.Errorf("业务数据加密失败: %w", err)
	}

	// 3. 构造完整的数据结构
	apduClass := &APDUDataClass{
		AuditData:    *auditData,
		BusinessData: *encryptedBusinessData,
		Metadata:     metadata,
	}

	// 4. 实施合规审计
	auditResult, err := hem.auditEngine.AuditAPDUData(apduClass)
	if err != nil {
		return nil, fmt.Errorf("合规审计失败: %w", err)
	}

	if !auditResult.Compliant {
		return nil, fmt.Errorf("APDU数据不符合合规要求: %s", auditResult.Reason)
	}

	global.GVA_LOG.Info("APDU混合加密完成",
		zap.String("sessionId", sessionID),
		zap.String("commandClass", auditData.CommandClass),
		zap.Int("riskScore", auditData.RiskScore),
		zap.Bool("compliant", auditResult.Compliant),
	)

	return apduClass, nil
}

// DecryptAPDUFromTransmission 解密传输的APDU
func (hem *HybridEncryptionManager) DecryptAPDUFromTransmission(
	sessionID string,
	apduClass *APDUDataClass,
	userID string,
) ([]byte, error) {

	// 1. 先进行合规审计
	auditResult, err := hem.auditEngine.AuditAPDUData(apduClass)
	if err != nil {
		return nil, fmt.Errorf("接收数据合规审计失败: %w", err)
	}

	if !auditResult.Compliant {
		// 记录违规尝试
		hem.logViolationAttempt(apduClass, auditResult, userID)
		return nil, fmt.Errorf("接收到不合规APDU数据: %s", auditResult.Reason)
	}

	// 2. 解密业务数据
	businessData, err := hem.decryptBusinessData(sessionID, &apduClass.BusinessData, userID)
	if err != nil {
		return nil, fmt.Errorf("业务数据解密失败: %w", err)
	}

	// 3. 对解密后的业务数据进行深度合规检查
	businessAuditResult, err := hem.auditEngine.AuditBusinessData(businessData, userID)
	if err != nil {
		return nil, fmt.Errorf("业务数据合规审计失败: %w", err)
	}

	if !businessAuditResult.Compliant {
		// 记录业务数据违规
		hem.logBusinessDataViolation(businessData, businessAuditResult, userID, sessionID)
		return nil, fmt.Errorf("业务数据不符合合规要求: %s", businessAuditResult.Reason)
	}

	// 4. 重组原始APDU
	rawAPDU, err := hem.reconstructAPDU(&apduClass.AuditData, businessData)
	if err != nil {
		return nil, fmt.Errorf("APDU重组失败: %w", err)
	}

	global.GVA_LOG.Info("APDU混合解密完成，通过所有合规检查",
		zap.String("sessionId", sessionID),
		zap.String("commandClass", apduClass.AuditData.CommandClass),
		zap.Int("apduSize", len(rawAPDU)),
		zap.String("businessAuditResult", businessAuditResult.Reason),
	)

	return rawAPDU, nil
}

// classifyAPDUData 分类APDU数据
func (hem *HybridEncryptionManager) classifyAPDUData(rawAPDU []byte, metadata APDUMetadata) (*AuditableData, map[string]interface{}, error) {
	// 解析APDU命令
	if len(rawAPDU) < 4 {
		return nil, nil, errors.New("APDU长度不足")
	}

	cla := rawAPDU[0] // 类字节
	ins := rawAPDU[1] // 指令字节
	p1 := rawAPDU[2]  // 参数1
	p2 := rawAPDU[3]  // 参数2

	// 分析命令类型
	commandClass := analyzeCommandClass(ins)
	commandType := analyzeCommandType(cla, ins, p1, p2)

	// 提取审计相关信息
	auditData := &AuditableData{
		CommandClass:     commandClass,
		CommandType:      commandType,
		ApplicationID:    extractApplicationID(rawAPDU), // 脱敏处理
		TransactionType:  extractTransactionType(rawAPDU),
		Amount:           extractAmount(rawAPDU),
		Currency:         extractCurrency(rawAPDU),
		MerchantCategory: extractMerchantCategory(rawAPDU),
		RiskScore:        calculateRiskScore(rawAPDU, metadata),
		Timestamp:        time.Now(),
	}

	// 提取业务数据 (可能包含敏感信息)
	businessData := map[string]interface{}{
		"rawAPDU":   hex.EncodeToString(rawAPDU),
		"dataField": extractDataField(rawAPDU),
		"response":  extractResponseData(rawAPDU),
	}

	return auditData, businessData, nil
}

// encryptBusinessData 加密业务数据 (使用审计密钥，服务器可解密)
func (hem *HybridEncryptionManager) encryptBusinessData(sessionID string, businessData map[string]interface{}, userID string) (*EncryptedBusinessData, error) {
	// 所有业务数据都使用审计密钥加密，服务器可以解密进行合规检查
	encryptedData, encInfo, err := hem.encryptWithAuditKey(businessData)
	if err != nil {
		return nil, fmt.Errorf("审计加密失败: %w", err)
	}

	return &EncryptedBusinessData{
		EncryptedData:  encryptedData,
		EncryptionInfo: *encInfo,
	}, nil
}

// separateSensitiveData 现在不再分离数据，所有数据都作为可审计数据处理
func (hem *HybridEncryptionManager) separateSensitiveData(businessData map[string]interface{}) (map[string]interface{}, map[string]interface{}) {
	// 所有数据都归类为可审计数据，服务器需要解密进行合规检查
	auditable := make(map[string]interface{})
	for key, value := range businessData {
		auditable[key] = value
	}

	// 返回空的敏感数据和完整的可审计数据
	return make(map[string]interface{}), auditable
}

// encryptWithSessionKey 使用会话密钥加密 (端到端)
func (hem *HybridEncryptionManager) encryptWithSessionKey(data map[string]interface{}, key []byte) (string, *EncryptionInfo, error) {
	// 序列化数据
	jsonData, err := json.Marshal(data)
	if err != nil {
		return "", nil, err
	}

	// AES-GCM加密
	block, err := aes.NewCipher(key)
	if err != nil {
		return "", nil, err
	}

	aesGCM, err := cipher.NewGCM(block)
	if err != nil {
		return "", nil, err
	}

	nonce := make([]byte, aesGCM.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return "", nil, err
	}

	ciphertext := aesGCM.Seal(nil, nonce, jsonData, nil)

	// 分离密文和标签
	tagSize := aesGCM.Overhead()
	encryptedData := ciphertext[:len(ciphertext)-tagSize]
	authTag := ciphertext[len(ciphertext)-tagSize:]

	encInfo := &EncryptionInfo{
		Algorithm: "AES-256-GCM",
		Nonce:     hex.EncodeToString(nonce),
		Tag:       hex.EncodeToString(authTag),
	}

	return hex.EncodeToString(encryptedData), encInfo, nil
}

// encryptWithAuditKey 使用审计密钥加密
func (hem *HybridEncryptionManager) encryptWithAuditKey(data map[string]interface{}) (string, *EncryptionInfo, error) {
	return hem.encryptWithSessionKey(data, hem.auditKeyManager.masterKey)
}

// 辅助函数实现
func analyzeCommandClass(ins byte) string {
	switch ins {
	case 0xA4:
		return "SELECT"
	case 0xB0, 0xB1:
		return "READ"
	case 0xD0, 0xD1, 0xD6:
		return "WRITE"
	case 0x84:
		return "GET_CHALLENGE"
	case 0x88:
		return "INTERNAL_AUTHENTICATE"
	default:
		return "UNKNOWN"
	}
}

func analyzeCommandType(cla, ins, p1, p2 byte) string {
	return fmt.Sprintf("CLA:%02X,INS:%02X,P1:%02X,P2:%02X", cla, ins, p1, p2)
}

func extractApplicationID(apdu []byte) string {
	// 简化实现：提取并脱敏AID
	if len(apdu) > 5 {
		endIdx := min(len(apdu), 10)
		hexStr := hex.EncodeToString(apdu[5:endIdx])
		// 确保不会发生切片越界
		if len(hexStr) >= 8 {
			return fmt.Sprintf("AID_%s", hexStr[0:8])
		} else {
			return fmt.Sprintf("AID_%s", hexStr)
		}
	}
	return "UNKNOWN"
}

func extractTransactionType(apdu []byte) string {
	// 根据APDU模式识别交易类型
	return "PAYMENT" // 简化实现
}

func extractAmount(apdu []byte) *int64 {
	// 从APDU中提取金额信息
	// 这里需要根据具体的卡片应用协议实现
	return nil
}

func extractCurrency(apdu []byte) string {
	return "CNY" // 简化实现
}

func extractMerchantCategory(apdu []byte) string {
	return "RETAIL" // 简化实现
}

func calculateRiskScore(apdu []byte, metadata APDUMetadata) int {
	// 实现风险评分算法
	score := 0

	// 基于命令类型的风险
	if len(apdu) > 1 {
		switch apdu[1] {
		case 0x88: // INTERNAL_AUTHENTICATE
			score += 30 // 降低风险评分
		case 0xD0, 0xD1, 0xD6: // WRITE operations
			score += 20 // 降低风险评分
		case 0xA4: // SELECT
			score += 5 // 降低风险评分
		}
	}

	// 移除时间限制 - 系统支持24/7运行
	// 原有的深夜操作限制已被移除，支持全天候支付服务

	return min(score, 100)
}

func extractDataField(apdu []byte) []byte {
	if len(apdu) <= 5 {
		return nil
	}
	return apdu[5:]
}

func extractResponseData(apdu []byte) []byte {
	// 对于响应APDU，返回数据部分
	if len(apdu) >= 2 {
		return apdu[:len(apdu)-2] // 移除SW1SW2
	}
	return apdu
}

func generateKeyID() string {
	return fmt.Sprintf("audit_key_%d", time.Now().Unix())
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

// logViolationAttempt 记录违规尝试
func (hem *HybridEncryptionManager) logViolationAttempt(apduClass *APDUDataClass, auditResult *ComplianceResult, userID string) {
	global.GVA_LOG.Warn("🚨 检测到不合规APDU数据",
		zap.String("userId", userID),
		zap.String("sessionId", apduClass.Metadata.SessionID),
		zap.String("commandClass", apduClass.AuditData.CommandClass),
		zap.String("violationReason", auditResult.Reason),
		zap.Int("riskScore", apduClass.AuditData.RiskScore),
		zap.String("remoteAddr", apduClass.Metadata.ClientID),
	)

	// 可以触发告警、封禁等操作
	hem.auditEngine.HandleViolation(apduClass, auditResult, userID)
}

// logBusinessDataViolation 记录业务数据违规
func (hem *HybridEncryptionManager) logBusinessDataViolation(businessData map[string]interface{}, auditResult *ComplianceResult, userID, sessionID string) {
	// 脱敏处理敏感数据用于日志记录
	sanitizedData := hem.sanitizeBusinessDataForLogging(businessData)

	global.GVA_LOG.Warn("🚨 检测到业务数据违规",
		zap.String("userId", userID),
		zap.String("sessionId", sessionID),
		zap.String("violationReason", auditResult.Reason),
		zap.String("riskLevel", auditResult.RiskLevel),
		zap.Any("sanitizedBusinessData", sanitizedData),
	)

	// 记录到审计日志
	global.LogAuditEvent(
		"business_data_violation",
		global.ComplianceViolation{
			UserID:       userID,
			SessionID:    sessionID,
			CommandClass: "BUSINESS_DATA_CHECK",
			Reason:       auditResult.Reason,
			RiskLevel:    auditResult.RiskLevel,
			Actions:      strings.Join(auditResult.Actions, ","),
			Timestamp:    time.Now(),
		},
	)
}

// sanitizeBusinessDataForLogging 脱敏业务数据用于日志记录
func (hem *HybridEncryptionManager) sanitizeBusinessDataForLogging(businessData map[string]interface{}) map[string]interface{} {
	sanitized := make(map[string]interface{})

	for key, value := range businessData {
		switch strings.ToLower(key) {
		case "pan", "cardnumber":
			// PAN脱敏：只显示前6位和后4位
			if panStr, ok := value.(string); ok && len(panStr) >= 10 {
				sanitized[key] = panStr[:6] + "****" + panStr[len(panStr)-4:]
			} else {
				sanitized[key] = "****"
			}
		case "cvv", "cvc":
			sanitized[key] = "***"
		case "pin":
			sanitized[key] = "****"
		case "amount":
			// 金额可以记录，但保留为字符串格式避免精度问题
			sanitized[key] = fmt.Sprintf("%.2f", value)
		default:
			// 其他数据可以记录
			sanitized[key] = value
		}
	}

	return sanitized
}

// 其他辅助方法...
func (hem *HybridEncryptionManager) decryptBusinessData(sessionID string, encBusinessData *EncryptedBusinessData, userID string) (map[string]interface{}, error) {
	// 使用审计密钥解密所有业务数据
	decryptedData, err := hem.decryptWithAuditKey(encBusinessData.EncryptedData, &encBusinessData.EncryptionInfo)
	if err != nil {
		return nil, fmt.Errorf("审计解密失败: %w", err)
	}

	// 将解密后的JSON数据转换为map
	var businessData map[string]interface{}
	if err := json.Unmarshal([]byte(decryptedData), &businessData); err != nil {
		return nil, fmt.Errorf("解析业务数据失败: %w", err)
	}

	return businessData, nil
}

func (hem *HybridEncryptionManager) reconstructAPDU(auditData *AuditableData, businessData map[string]interface{}) ([]byte, error) {
	// 从业务数据中提取原始APDU
	if rawAPDUHex, ok := businessData["rawAPDU"].(string); ok {
		rawAPDU, err := hex.DecodeString(rawAPDUHex)
		if err != nil {
			return nil, fmt.Errorf("解码APDU失败: %w", err)
		}
		return rawAPDU, nil
	}

	return nil, fmt.Errorf("无法从业务数据中提取APDU")
}

// decryptWithAuditKey 使用审计密钥解密数据
func (hem *HybridEncryptionManager) decryptWithAuditKey(encryptedData string, encInfo *EncryptionInfo) (string, error) {
	// 解码加密数据
	ciphertext, err := hex.DecodeString(encryptedData)
	if err != nil {
		return "", fmt.Errorf("解码密文失败: %w", err)
	}

	nonce, err := hex.DecodeString(encInfo.Nonce)
	if err != nil {
		return "", fmt.Errorf("解码nonce失败: %w", err)
	}

	authTag, err := hex.DecodeString(encInfo.Tag)
	if err != nil {
		return "", fmt.Errorf("解码认证标签失败: %w", err)
	}

	// 重新组合密文和标签
	fullCiphertext := append(ciphertext, authTag...)

	// AES-GCM解密
	block, err := aes.NewCipher(hem.auditKeyManager.masterKey)
	if err != nil {
		return "", err
	}

	aesGCM, err := cipher.NewGCM(block)
	if err != nil {
		return "", err
	}

	plaintext, err := aesGCM.Open(nil, nonce, fullCiphertext, nil)
	if err != nil {
		return "", fmt.Errorf("解密失败: %w", err)
	}

	return string(plaintext), nil
}

// 全局混合加密管理器
var GlobalHybridEncryption = NewHybridEncryptionManager()
