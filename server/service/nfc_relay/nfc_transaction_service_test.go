package nfc_relay

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/flipped-aurora/gin-vue-admin/server/model/nfc_relay"
	nfc_relay_request "github.com/flipped-aurora/gin-vue-admin/server/model/nfc_relay/request"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"gorm.io/datatypes"
	"gorm.io/gorm"
)

// MockDB 模拟数据库连接
type MockDB struct {
	mock.Mock
}

func (m *MockDB) Create(value interface{}) *gorm.DB {
	args := m.Called(value)
	return args.Get(0).(*gorm.DB)
}

func (m *MockDB) First(dest interface{}, conds ...interface{}) *gorm.DB {
	args := m.Called(dest, conds)
	return args.Get(0).(*gorm.DB)
}

func (m *MockDB) Where(query interface{}, args ...interface{}) *gorm.DB {
	mockArgs := m.Called(query, args)
	return mockArgs.Get(0).(*gorm.DB)
}

func (m *MockDB) Updates(values interface{}) *gorm.DB {
	args := m.Called(values)
	return args.Get(0).(*gorm.DB)
}

// TestNFCTransactionService_CreateTransaction 测试创建交易
func TestNFCTransactionService_CreateTransaction(t *testing.T) {
	// 准备测试数据
	userID := uuid.New()
	transmitterClientID := "transmitter_123"
	receiverClientID := "receiver_456"

	// 测试用例
	tests := []struct {
		name        string
		request     *nfc_relay_request.CreateTransactionRequest
		expectedErr bool
	}{
		{
			name: "成功创建交易",
			request: &nfc_relay_request.CreateTransactionRequest{
				TransmitterClientID: transmitterClientID,
				ReceiverClientID:    &receiverClientID,
				Amount:              10000, // 100.00元
				Currency:            "CNY",
				Description:         "测试交易",
				Metadata: map[string]interface{}{
					"test": "data",
				},
			},
			expectedErr: false,
		},
		{
			name: "无效的传卡端ID",
			request: &nfc_relay_request.CreateTransactionRequest{
				TransmitterClientID: "",
				ReceiverClientID:    &receiverClientID,
				Amount:              10000,
				Currency:            "CNY",
			},
			expectedErr: true,
		},
	}

	// 创建服务实例
	service := &NFCTransactionService{}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()

			response, err := service.CreateTransaction(ctx, tt.request, userID)

			if tt.expectedErr {
				assert.Error(t, err)
				assert.Nil(t, response)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, response)
				assert.NotEmpty(t, response.TransactionID)
				assert.Equal(t, nfc_relay.StatusPending, response.Status)
			}
		})
	}
}

// TestNFCTransactionService_UpdateTransactionStatus 测试更新交易状态
func TestNFCTransactionService_UpdateTransactionStatus(t *testing.T) {
	userID := uuid.New()
	transactionID := "txn_20240101_test123"

	tests := []struct {
		name           string
		request        *nfc_relay_request.UpdateTransactionStatusRequest
		currentStatus  string
		expectedErr    bool
		expectedStatus string
	}{
		{
			name: "从pending转换到active",
			request: &nfc_relay_request.UpdateTransactionStatusRequest{
				TransactionID: transactionID,
				Status:        nfc_relay.StatusActive,
				Reason:        "开始处理",
			},
			currentStatus:  nfc_relay.StatusPending,
			expectedErr:    false,
			expectedStatus: nfc_relay.StatusActive,
		},
		{
			name: "无效的状态转换",
			request: &nfc_relay_request.UpdateTransactionStatusRequest{
				TransactionID: transactionID,
				Status:        nfc_relay.StatusCompleted,
				Reason:        "直接完成",
			},
			currentStatus:  nfc_relay.StatusPending,
			expectedErr:    true,
			expectedStatus: nfc_relay.StatusPending,
		},
	}

	service := &NFCTransactionService{}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()

			response, err := service.UpdateTransactionStatus(ctx, tt.request, userID)

			if tt.expectedErr {
				assert.Error(t, err)
				assert.Nil(t, response)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, response)
				assert.Equal(t, tt.expectedStatus, response.Status)
				assert.Equal(t, tt.currentStatus, response.PreviousStatus)
			}
		})
	}
}

// TestValidStatusTransition 测试状态转换验证
func TestValidStatusTransition(t *testing.T) {
	tests := []struct {
		name       string
		fromStatus string
		toStatus   string
		expected   bool
	}{
		// 有效转换
		{"pending到active", nfc_relay.StatusPending, nfc_relay.StatusActive, true},
		{"active到completed", nfc_relay.StatusActive, nfc_relay.StatusCompleted, true},
		{"active到failed", nfc_relay.StatusActive, nfc_relay.StatusFailed, true},
		{"pending到cancelled", nfc_relay.StatusPending, nfc_relay.StatusCancelled, true},

		// 无效转换
		{"pending直接到completed", nfc_relay.StatusPending, nfc_relay.StatusCompleted, false},
		{"completed到active", nfc_relay.StatusCompleted, nfc_relay.StatusActive, false},
		{"failed到active", nfc_relay.StatusFailed, nfc_relay.StatusActive, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := nfc_relay.IsValidStatusTransition(tt.fromStatus, tt.toStatus)
			assert.Equal(t, tt.expected, result)
		})
	}
}

// TestTransactionIDGeneration 测试交易ID生成
func TestTransactionIDGeneration(t *testing.T) {
	service := &NFCTransactionService{}

	// 生成多个ID
	ids := make([]string, 100)
	for i := 0; i < 100; i++ {
		id, err := service.generateTransactionID()
		assert.NoError(t, err)
		assert.NotEmpty(t, id)
		assert.Contains(t, id, "txn_")
		assert.Len(t, id, 29) // txn_YYYYMMDD_16位hex = 29字符
		ids[i] = id
	}

	// 验证唯一性
	uniqueIDs := make(map[string]bool)
	for _, id := range ids {
		assert.False(t, uniqueIDs[id], "交易ID应该是唯一的: %s", id)
		uniqueIDs[id] = true
	}
}

// TestMetadataMarshaling 测试元数据序列化
func TestMetadataMarshaling(t *testing.T) {
	testData := map[string]interface{}{
		"cardType":   "visa",
		"amount":     10000,
		"merchantId": "merchant_123",
		"terminal": map[string]interface{}{
			"id":       "term_456",
			"location": "Beijing",
		},
	}

	// 序列化
	jsonData, err := json.Marshal(testData)
	assert.NoError(t, err)

	// 创建datatypes.JSON
	metadata := datatypes.JSON(jsonData)

	// 反序列化验证
	var result map[string]interface{}
	err = json.Unmarshal(metadata, &result)
	assert.NoError(t, err)
	assert.Equal(t, testData["cardType"], result["cardType"])
	assert.Equal(t, float64(10000), result["amount"]) // JSON数字类型转换
}

// TestTransactionValidation 测试交易验证
func TestTransactionValidation(t *testing.T) {
	tests := []struct {
		name    string
		request *nfc_relay_request.CreateTransactionRequest
		valid   bool
	}{
		{
			name: "有效交易",
			request: &nfc_relay_request.CreateTransactionRequest{
				TransmitterClientID: "transmitter_123",
				Amount:              10000,
				Currency:            "CNY",
			},
			valid: true,
		},
		{
			name: "缺少传卡端ID",
			request: &nfc_relay_request.CreateTransactionRequest{
				TransmitterClientID: "",
				Amount:              10000,
				Currency:            "CNY",
			},
			valid: false,
		},
		{
			name: "无效金额",
			request: &nfc_relay_request.CreateTransactionRequest{
				TransmitterClientID: "transmitter_123",
				Amount:              0,
				Currency:            "CNY",
			},
			valid: false,
		},
		{
			name: "无效货币",
			request: &nfc_relay_request.CreateTransactionRequest{
				TransmitterClientID: "transmitter_123",
				Amount:              10000,
				Currency:            "",
			},
			valid: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			isValid := validateCreateTransactionRequest(tt.request)
			assert.Equal(t, tt.valid, isValid)
		})
	}
}

// validateCreateTransactionRequest 验证创建交易请求
func validateCreateTransactionRequest(req *nfc_relay_request.CreateTransactionRequest) bool {
	if req.TransmitterClientID == "" {
		return false
	}
	if req.Amount <= 0 {
		return false
	}
	if req.Currency == "" {
		return false
	}
	return true
}

// BenchmarkTransactionIDGeneration 性能测试：交易ID生成
func BenchmarkTransactionIDGeneration(b *testing.B) {
	service := &NFCTransactionService{}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := service.generateTransactionID()
		if err != nil {
			b.Fatal(err)
		}
	}
}

// BenchmarkStatusTransitionValidation 性能测试：状态转换验证
func BenchmarkStatusTransitionValidation(b *testing.B) {
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		nfc_relay.IsValidStatusTransition(nfc_relay.StatusPending, nfc_relay.StatusActive)
	}
}

// TestTransactionTimeout 测试交易超时处理
func TestTransactionTimeout(t *testing.T) {
	transaction := &nfc_relay.NFCTransaction{
		TransactionID: "txn_test_timeout",
		Status:        nfc_relay.StatusActive,
		CreatedAt:     time.Now().Add(-time.Hour),        // 1小时前创建
		UpdatedAt:     time.Now().Add(-30 * time.Minute), // 30分钟前更新
	}

	// 检查是否超时（假设30分钟超时）
	timeout := 30 * time.Minute
	isTimeout := time.Since(transaction.UpdatedAt) > timeout

	assert.True(t, isTimeout, "交易应该被认为超时")
}

// TestConcurrentTransactionCreation 测试并发创建交易
func TestConcurrentTransactionCreation(t *testing.T) {
	service := &NFCTransactionService{}
	userID := uuid.New()

	// 并发创建多个交易
	concurrency := 10
	results := make(chan string, concurrency)
	errors := make(chan error, concurrency)

	for i := 0; i < concurrency; i++ {
		go func(index int) {
			ctx := context.Background()
			req := &nfc_relay_request.CreateTransactionRequest{
				TransmitterClientID: "transmitter_123",
				Amount:              10000,
				Currency:            "CNY",
				Description:         "并发测试交易",
			}

			response, err := service.CreateTransaction(ctx, req, userID)
			if err != nil {
				errors <- err
				return
			}
			results <- response.TransactionID
		}(i)
	}

	// 收集结果
	transactionIDs := make([]string, 0, concurrency)
	for i := 0; i < concurrency; i++ {
		select {
		case id := <-results:
			transactionIDs = append(transactionIDs, id)
		case err := <-errors:
			t.Errorf("并发创建交易失败: %v", err)
		case <-time.After(5 * time.Second):
			t.Fatal("测试超时")
		}
	}

	// 验证所有ID都是唯一的
	uniqueIDs := make(map[string]bool)
	for _, id := range transactionIDs {
		assert.False(t, uniqueIDs[id], "交易ID应该唯一: %s", id)
		uniqueIDs[id] = true
	}

	assert.Equal(t, concurrency, len(transactionIDs), "应该创建指定数量的交易")
}

// TestTransactionMetrics 测试交易指标计算
func TestTransactionMetrics(t *testing.T) {
	transactions := []nfc_relay.NFCTransaction{
		{Status: nfc_relay.StatusCompleted, CreatedAt: time.Now()},
		{Status: nfc_relay.StatusCompleted, CreatedAt: time.Now()},
		{Status: nfc_relay.StatusFailed, CreatedAt: time.Now()},
		{Status: nfc_relay.StatusActive, CreatedAt: time.Now()},
	}

	// 计算成功率
	completed := 0
	total := len(transactions)

	for _, tx := range transactions {
		if tx.Status == nfc_relay.StatusCompleted {
			completed++
		}
	}

	successRate := float64(completed) / float64(total) * 100
	expectedRate := 50.0 // 2/4 = 50%

	assert.Equal(t, expectedRate, successRate, "成功率计算错误")
}
