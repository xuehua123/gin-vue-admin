package system

import (
	"testing"

	"github.com/flipped-aurora/gin-vue-admin/server/model/system/request"
	"github.com/stretchr/testify/assert"
)

// TestAuthorizeMqttAction 包含了对MQTT权限检查逻辑的全面单元测试
func TestAuthorizeMqttAction(t *testing.T) {
	// 创建一个MqttAuthApi实例以调用其方法
	authApi := MqttAuthApi{}

	// 定义测试用例结构体
	type testCase struct {
		name          string              // 测试用例的名称
		claims        *request.MQTTClaims // 模拟的JWT Claims
		topic         string              // 请求的MQTT主题
		action        string              // "subscribe" 或 "publish"
		expectedAllow bool                // 期望的结果 (true for allow, false for deny)
		description   string              // 测试用例的描述
	}

	// 定义测试用例
	testCases := []testCase{
		// --- 核心场景: 验证新增的配对权限规则 ---
		{
			name:          "客户端订阅自己的配对通知",
			claims:        &request.MQTTClaims{Role: "transmitter", ClientID: "client-tx-001"},
			topic:         "nfc_relay/pairing/notifications/client-tx-001",
			action:        "subscribe",
			expectedAllow: true,
			description:   "核心功能：客户端必须能够订阅自己的配对成功/失败通知。",
		},
		{
			name:          "客户端订阅自己的状态更新",
			claims:        &request.MQTTClaims{Role: "receiver", ClientID: "client-rx-002"},
			topic:         "nfc_relay/pairing/status_updates/client-rx-002",
			action:        "subscribe",
			expectedAllow: true,
			description:   "核心功能：客户端必须能够订阅自己的配对状态更新（如排队中、超时等）。",
		},
		{
			name:          "客户端订阅全局配对事件",
			claims:        &request.MQTTClaims{Role: "transmitter", ClientID: "client-tx-001"},
			topic:         "nfc_relay/pairing/events",
			action:        "subscribe",
			expectedAllow: true,
			description:   "核心功能：客户端需要订阅全局事件以了解系统范围内的配对活动。",
		},

		// --- 安全场景: 验证权限边界 ---
		{
			name:          "客户端试图订阅他人的配对通知",
			claims:        &request.MQTTClaims{Role: "transmitter", ClientID: "client-tx-001"},
			topic:         "nfc_relay/pairing/notifications/another-client-007",
			action:        "subscribe",
			expectedAllow: false,
			description:   "安全边界：客户端绝对不能窥探其他客户端的私人配对通知。",
		},
		{
			name:          "客户端试图发布到自己的配对通知主题",
			claims:        &request.MQTTClaims{Role: "transmitter", ClientID: "client-tx-001"},
			topic:         "nfc_relay/pairing/notifications/client-tx-001",
			action:        "publish",
			expectedAllow: false,
			description:   "安全边界：通知主题是服务器下发的，客户端不能伪造通知。",
		},
		{
			name:          "客户端试图发布到全局配对事件主题",
			claims:        &request.MQTTClaims{Role: "receiver", ClientID: "client-rx-002"},
			topic:         "nfc_relay/pairing/events",
			action:        "publish",
			expectedAllow: false,
			description:   "安全边界：全局事件只能由服务器发布，防止客户端广播虚假事件。",
		},

		// --- 回归场景: 验证现有规则未被破坏 ---
		{
			name:          "回归测试：客户端订阅自己的状态",
			claims:        &request.MQTTClaims{Role: "transmitter", ClientID: "client-tx-001"},
			topic:         "nfc_relay/clients/client-tx-001/state",
			action:        "subscribe",
			expectedAllow: true,
			description:   "回归测试：确保对 'clients' 主题的现有规则仍然有效。",
		},
		{
			name:          "回归测试：客户端发布自己的APDU",
			claims:        &request.MQTTClaims{Role: "receiver", ClientID: "client-rx-002"},
			topic:         "nfc_relay/sessions/client-rx-002/receiver/apdu",
			action:        "publish",
			expectedAllow: true,
			description:   "回归测试：确保对 'sessions' 主题的现有规则仍然有效。",
		},
		{
			name:          "回归测试：admin订阅任意配对主题",
			claims:        &request.MQTTClaims{Role: "admin", ClientID: "admin-client"},
			topic:         "nfc_relay/pairing/notifications/any-client-id",
			action:        "subscribe",
			expectedAllow: true,
			description:   "回归测试：确保 'admin' 角色的超级订阅权限未受影响。",
		},
		{
			name:          "回归测试：admin订阅任意客户端状态",
			claims:        &request.MQTTClaims{Role: "admin", ClientID: "admin-client"},
			topic:         "nfc_relay/clients/any-client-id/state",
			action:        "subscribe",
			expectedAllow: true,
			description:   "回归测试：确保 'admin' 对 'clients' 主题的超级订阅权限仍然有效。",
		},
	}

	// 循环执行所有测试用例
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// 调用被测试的函数
			actualResult := authApi.authorizeMqttAction(tc.claims, tc.topic, tc.action)

			// 使用 testify/assert 进行断言，提供更清晰的测试失败信息
			assert.Equal(t, tc.expectedAllow, actualResult, "描述: %s", tc.description)
		})
	}
}
