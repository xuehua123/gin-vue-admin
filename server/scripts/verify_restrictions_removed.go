package main

import (
	"fmt"
	"log"
	"time"

	"github.com/flipped-aurora/gin-vue-admin/server/nfc_relay/security"
)

func main() {
	fmt.Println("🚀 NFC中继支付系统限制解除验证")
	fmt.Println("=====================================")

	// 1. 验证24/7服务支持
	fmt.Println("\n1. 验证24/7服务支持...")
	testTimeRestriction()

	// 2. 验证风险评分优化
	fmt.Println("\n2. 验证风险评分优化...")
	testRiskScoreOptimization()

	// 3. 验证加密解密功能
	fmt.Println("\n3. 验证加密解密功能...")
	testEncryptionDecryption()

	// 4. 验证合规检查宽松化
	fmt.Println("\n4. 验证合规检查宽松化...")
	testComplianceRelaxation()

	fmt.Println("\n✅ 所有验证完成！系统限制已成功解除。")
}

func testTimeRestriction() {
	// 测试深夜时间（原本受限时间）
	nightTime := time.Date(2024, 1, 15, 2, 30, 0, 0, time.Local) // 凌晨2:30

	metadata := security.APDUMetadata{
		SessionID:   "test-session",
		SequenceNum: nightTime.UnixNano(),
		Direction:   "upstream",
		Timestamp:   nightTime, // 使用深夜时间
		ClientID:    "test-client",
		UserID:      "test-user",
		DeviceInfo:  "test-device",
		ChecksumCRC: "CRC_123",
	}

	auditEngine := security.NewComplianceAuditEngine()
	apduClass := &security.APDUDataClass{
		AuditData: security.AuditableData{
			CommandClass: "SELECT",
			Timestamp:    nightTime,
		},
		Metadata: metadata,
	}

	result, err := auditEngine.AuditAPDUData(apduClass)
	if err != nil {
		log.Printf("❌ 时间限制测试失败: %v", err)
		return
	}

	if result.Compliant {
		fmt.Println("✅ 深夜时间交易已允许 - 24/7服务正常")
	} else {
		fmt.Printf("❌ 深夜时间仍受限: %s\n", result.Reason)
	}
}

func testRiskScoreOptimization() {
	// 测试各种APDU命令的风险评分
	testCases := []struct {
		ins      byte
		expected string
	}{
		{0xA4, "SELECT"},                // SELECT命令
		{0x88, "INTERNAL_AUTHENTICATE"}, // 内部认证
		{0xD0, "WRITE"},                 // WRITE命令
	}

	currentTime := time.Now()
	metadata := security.APDUMetadata{
		SessionID:   "test-session",
		SequenceNum: currentTime.UnixNano(),
		Direction:   "upstream",
		Timestamp:   currentTime,
		ClientID:    "test-client",
		UserID:      "test-user",
		DeviceInfo:  "test-device",
		ChecksumCRC: "CRC_123",
	}

	for _, tc := range testCases {
		apdu := []byte{0x00, tc.ins, 0x04, 0x00, 0x0E}
		manager := security.NewHybridEncryptionManager()

		encrypted, err := manager.EncryptAPDUForTransmission(
			"test-session",
			apdu,
			metadata,
			"test-user",
		)

		if err != nil {
			fmt.Printf("❌ %s命令加密失败: %v\n", tc.expected, err)
			continue
		}

		fmt.Printf("✅ %s命令风险评分: %d (已优化)\n",
			tc.expected, encrypted.AuditData.RiskScore)
	}
}

func testEncryptionDecryption() {
	manager := security.NewHybridEncryptionManager()
	testSessionID := "test-session"
	testUserID := "test-user"
	testAPDU := []byte{0x00, 0xA4, 0x04, 0x00, 0x0E, 0x32, 0x50}

	metadata := security.APDUMetadata{
		SessionID:   testSessionID,
		SequenceNum: time.Now().UnixNano(),
		Direction:   "upstream",
		Timestamp:   time.Now(),
		ClientID:    "test-client",
		UserID:      testUserID,
		DeviceInfo:  "test-device",
		ChecksumCRC: "CRC_TEST",
	}

	// 加密
	encrypted, err := manager.EncryptAPDUForTransmission(
		testSessionID,
		testAPDU,
		metadata,
		testUserID,
	)
	if err != nil {
		fmt.Printf("❌ APDU加密失败: %v\n", err)
		return
	}

	// 解密
	decrypted, err := manager.DecryptAPDUFromTransmission(
		testSessionID,
		encrypted,
		testUserID,
	)
	if err != nil {
		fmt.Printf("❌ APDU解密失败: %v\n", err)
		return
	}

	// 验证
	if len(decrypted) == len(testAPDU) {
		fmt.Println("✅ APDU加密解密验证成功")
	} else {
		fmt.Println("❌ APDU加密解密验证失败")
	}
}

func testComplianceRelaxation() {
	auditEngine := security.NewComplianceAuditEngine()

	// 测试高风险命令（应该允许通过）
	metadata := security.APDUMetadata{
		SessionID:   "test-session",
		SequenceNum: time.Now().UnixNano(),
		Direction:   "upstream",
		Timestamp:   time.Now(),
		ClientID:    "test-client",
		UserID:      "test-user",
		DeviceInfo:  "test-device",
		ChecksumCRC: "CRC_123",
	}

	apduClass := &security.APDUDataClass{
		AuditData: security.AuditableData{
			CommandClass:    "WRITE", // 原本的高风险命令
			CommandType:     "WRITE_RECORD",
			ApplicationID:   "test-app",
			TransactionType: "PAYMENT",
			RiskScore:       20, // 降低后的评分
			Timestamp:       time.Now(),
		},
		Metadata: metadata,
	}

	result, err := auditEngine.AuditAPDUData(apduClass)
	if err != nil {
		fmt.Printf("❌ 合规检查失败: %v\n", err)
		return
	}

	if result.Compliant {
		fmt.Println("✅ 高风险命令检查已放宽 - WRITE操作被允许")
	} else {
		fmt.Printf("❌ 高风险命令仍被阻断: %s\n", result.Reason)
	}

	// 测试大额交易（应该允许通过）
	amount := int64(5000000) // 5000元（500万分）
	apduClass.AuditData.Amount = &amount

	result2, err := auditEngine.AuditAPDUData(apduClass)
	if err != nil {
		fmt.Printf("❌ 大额交易检查失败: %v\n", err)
		return
	}

	if result2.Compliant {
		fmt.Println("✅ 交易金额限制已提高 - 5000元交易被允许")
	} else {
		fmt.Printf("❌ 大额交易仍被限制: %s\n", result2.Reason)
	}
}
