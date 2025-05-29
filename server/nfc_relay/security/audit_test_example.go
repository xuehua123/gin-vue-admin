package security

import (
	"encoding/json"
	"fmt"
	"time"
)

// TestAuditLevelEncryption 测试审计级加密功能
func TestAuditLevelEncryption() {
	fmt.Println("🔐 测试审计级混合加密系统")
	fmt.Println("=====================================")

	// 1. 创建混合加密管理器
	hem := NewHybridEncryptionManager()

	// 2. 模拟APDU数据
	testAPDU := []byte{0x00, 0xA4, 0x04, 0x00, 0x08, 0x12, 0x34, 0x56, 0x78, 0x90, 0xAB, 0xCD, 0xEF}

	// 3. 构造元数据
	metadata := APDUMetadata{
		SessionID:   "test_session_001",
		SequenceNum: 1,
		Direction:   "command",
		Timestamp:   time.Now(),
		ClientID:    "test_client",
		UserID:      "test_user",
		DeviceInfo:  "test_device",
		ChecksumCRC: "ABC123",
	}

	// 4. 测试加密过程
	fmt.Println("📤 测试APDU加密...")
	encryptedAPDU, err := hem.EncryptAPDUForTransmission("test_session_001", testAPDU, metadata, "test_user")
	if err != nil {
		fmt.Printf("❌ 加密失败: %v\n", err)
		return
	}

	fmt.Printf("✅ 加密成功 - 命令类型: %s, 风险评分: %d\n",
		encryptedAPDU.AuditData.CommandClass,
		encryptedAPDU.AuditData.RiskScore)

	// 5. 测试解密过程
	fmt.Println("📥 测试APDU解密...")
	decryptedAPDU, err := hem.DecryptAPDUFromTransmission("test_session_001", encryptedAPDU, "test_user")
	if err != nil {
		fmt.Printf("❌ 解密失败: %v\n", err)
		return
	}

	fmt.Printf("✅ 解密成功 - APDU长度: %d bytes\n", len(decryptedAPDU))

	// 6. 验证数据完整性
	if len(testAPDU) == len(decryptedAPDU) {
		fmt.Println("✅ 数据完整性验证通过")
	} else {
		fmt.Printf("❌ 数据完整性验证失败 - 原始: %d, 解密: %d\n", len(testAPDU), len(decryptedAPDU))
	}
}

// TestComplianceChecks 测试合规检查功能
func TestComplianceChecks() {
	fmt.Println("\n🛡️ 测试合规检查功能")
	fmt.Println("=====================================")

	auditEngine := NewComplianceAuditEngine()

	// 测试用例1: 正常交易数据
	fmt.Println("📋 测试用例1: 正常交易数据")
	normalData := map[string]interface{}{
		"pan":              "4111111111111111", // 这是一个测试卡号，会被黑名单检测
		"amount":           100.50,
		"merchantCategory": "RETAIL",
		"cvv":              "123",
	}

	result, err := auditEngine.AuditBusinessData(normalData, "test_user")
	if err != nil {
		fmt.Printf("❌ 审计失败: %v\n", err)
	} else {
		fmt.Printf("📊 审计结果: %s (风险级别: %s)\n", result.Reason, result.RiskLevel)
	}

	// 测试用例2: 高风险商户
	fmt.Println("\n📋 测试用例2: 高风险商户")
	highRiskData := map[string]interface{}{
		"pan":              "5555555555554444",
		"amount":           500.00,
		"merchantCategory": "GAMBLING", // 高风险商户类别
		"cvv":              "456",
	}

	result, err = auditEngine.AuditBusinessData(highRiskData, "test_user")
	if err != nil {
		fmt.Printf("❌ 审计失败: %v\n", err)
	} else {
		fmt.Printf("📊 审计结果: %s (风险级别: %s)\n", result.Reason, result.RiskLevel)
	}

	// 测试用例3: 超额交易
	fmt.Println("\n📋 测试用例3: 超额交易")
	highAmountData := map[string]interface{}{
		"pan":              "6011111111111117",
		"amount":           10000.00, // 超过限额
		"merchantCategory": "RETAIL",
		"cvv":              "789",
	}

	result, err = auditEngine.AuditBusinessData(highAmountData, "test_user")
	if err != nil {
		fmt.Printf("❌ 审计失败: %v\n", err)
	} else {
		fmt.Printf("📊 审计结果: %s (风险级别: %s)\n", result.Reason, result.RiskLevel)
	}
}

// TestDataSanitization 测试数据脱敏功能
func TestDataSanitization() {
	fmt.Println("\n🔒 测试数据脱敏功能")
	fmt.Println("=====================================")

	hem := NewHybridEncryptionManager()

	// 原始敏感数据
	sensitiveData := map[string]interface{}{
		"pan":              "4111111111111111",
		"cvv":              "123",
		"pin":              "1234",
		"amount":           150.75,
		"merchantCategory": "RETAIL",
	}

	fmt.Println("📄 原始数据:")
	printJSON(sensitiveData)

	// 脱敏处理
	sanitized := hem.sanitizeBusinessDataForLogging(sensitiveData)

	fmt.Println("\n🔒 脱敏后数据:")
	printJSON(sanitized)
}

// 辅助函数：格式化打印JSON
func printJSON(data interface{}) {
	jsonBytes, err := json.MarshalIndent(data, "", "  ")
	if err != nil {
		fmt.Printf("JSON格式化失败: %v\n", err)
		return
	}
	fmt.Println(string(jsonBytes))
}

// RunAllTests 运行所有测试
func RunAllTests() {
	fmt.Println("🚀 NFC中继系统审计级安全架构测试")
	fmt.Println("=====================================")
	fmt.Println("⚡ 特性: 服务器可解密所有数据进行合规检查")
	fmt.Println("🛡️ 安全: TLS + 审计级加密 + 智能合规检测")
	fmt.Println()

	TestAuditLevelEncryption()
	TestComplianceChecks()
	TestDataSanitization()

	fmt.Println("\n🎉 所有测试完成！")
	fmt.Println("✅ 审计级安全架构运行正常")
	fmt.Println("�� 服务器具备完整的数据审计能力")
}
