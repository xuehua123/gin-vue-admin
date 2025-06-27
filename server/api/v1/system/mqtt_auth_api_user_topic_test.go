package system

import (
	"testing"

	"github.com/flipped-aurora/gin-vue-admin/server/model/system/request"
	"github.com/stretchr/testify/assert"
)

// TestCheckUserTopicPermissions 测试用户级主题权限检查 (企业级Bug修复验证)
func TestCheckUserTopicPermissions(t *testing.T) {
	api := &MqttAuthApi{}

	testCases := []struct {
		name          string
		clientID      string
		topic         string
		action        string
		expectedAllow bool
		description   string
	}{
		// 核心修复场景: admin用户的配对通知权限
		{
			name:          "admin用户transmitter角色订阅用户级通知",
			clientID:      "admin-transmitter-005",
			topic:         "nfc_relay/user/admin/notifications",
			action:        "subscribe",
			expectedAllow: true,
			description:   "核心Bug修复: admin用户的transmitter客户端必须能订阅用户级通知",
		},
		{
			name:          "admin用户receiver角色订阅用户级通知",
			clientID:      "admin-receiver-003",
			topic:         "nfc_relay/user/admin/notifications",
			action:        "subscribe",
			expectedAllow: true,
			description:   "核心Bug修复: admin用户的receiver客户端必须能订阅用户级通知",
		},

		// 安全验证: 用户名隔离
		{
			name:          "user123不能订阅admin的通知",
			clientID:      "user123-transmitter-001",
			topic:         "nfc_relay/user/admin/notifications",
			action:        "subscribe",
			expectedAllow: false,
			description:   "安全验证: 不同用户之间的通知必须隔离",
		},
		{
			name:          "admin不能订阅其他用户的通知",
			clientID:      "admin-transmitter-005",
			topic:         "nfc_relay/user/user123/notifications",
			action:        "subscribe",
			expectedAllow: false,
			description:   "安全验证: 即使是admin用户也不能跨用户订阅",
		},

		// 权限边界测试
		{
			name:          "不允许发布到用户级通知主题",
			clientID:      "admin-transmitter-005",
			topic:         "nfc_relay/user/admin/notifications",
			action:        "publish",
			expectedAllow: false,
			description:   "安全边界: 客户端不能发布到通知主题，只能订阅",
		},

		// 主题格式验证
		{
			name:          "错误的主题格式",
			clientID:      "admin-transmitter-005",
			topic:         "nfc_relay/user/admin",
			action:        "subscribe",
			expectedAllow: false,
			description:   "格式验证: 主题必须以/notifications结尾",
		},
		{
			name:          "错误的主题格式(多级)",
			clientID:      "admin-transmitter-005",
			topic:         "nfc_relay/user/admin/notifications/extra",
			action:        "subscribe",
			expectedAllow: false,
			description:   "格式验证: 主题不能有额外的层级",
		},

		// ClientID格式验证
		{
			name:          "无效的ClientID格式",
			clientID:      "invalid-format",
			topic:         "nfc_relay/user/admin/notifications",
			action:        "subscribe",
			expectedAllow: false,
			description:   "格式验证: 无效的ClientID格式无法提取用户名",
		},
		{
			name:          "空ClientID",
			clientID:      "",
			topic:         "nfc_relay/user/admin/notifications",
			action:        "subscribe",
			expectedAllow: false,
			description:   "边界验证: 空ClientID应该被拒绝",
		},

		// 非用户级主题测试
		{
			name:          "非用户级主题应该被拒绝",
			clientID:      "admin-transmitter-005",
			topic:         "nfc_relay/other/topic",
			action:        "subscribe",
			expectedAllow: false,
			description:   "边界验证: 非用户级主题应该返回false",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := api.checkUserTopicPermissions(tc.clientID, tc.topic, tc.action)
			assert.Equal(t, tc.expectedAllow, result,
				"测试失败: %s\n描述: %s\n输入: clientID=%s, topic=%s, action=%s",
				tc.name, tc.description, tc.clientID, tc.topic, tc.action)
		})
	}
}

// TestAuthorizeMqttActionWithUserTopics 测试完整的权限检查流程包含用户级主题
func TestAuthorizeMqttActionWithUserTopics(t *testing.T) {
	api := &MqttAuthApi{}

	testCases := []struct {
		name          string
		claims        *request.MQTTClaims
		topic         string
		action        string
		expectedAllow bool
		description   string
	}{
		// 核心修复验证: 用户级通知主题权限
		{
			name: "admin-transmitter订阅用户级通知(主要Bug修复)",
			claims: &request.MQTTClaims{
				Role:     "transmitter",
				Username: "admin",
				ClientID: "admin-transmitter-005",
			},
			topic:         "nfc_relay/user/admin/notifications",
			action:        "subscribe",
			expectedAllow: true,
			description:   "核心Bug修复: admin用户的transmitter客户端必须能够订阅用户级通知",
		},
		{
			name: "admin-receiver订阅用户级通知",
			claims: &request.MQTTClaims{
				Role:     "receiver",
				Username: "admin",
				ClientID: "admin-receiver-003",
			},
			topic:         "nfc_relay/user/admin/notifications",
			action:        "subscribe",
			expectedAllow: true,
			description:   "核心Bug修复: admin用户的receiver客户端必须能够订阅用户级通知",
		},

		// admin角色特殊权限仍然有效
		{
			name: "admin角色特殊权限订阅任意用户级通知",
			claims: &request.MQTTClaims{
				Role:     "admin",
				Username: "admin",
				ClientID: "admin-system-001",
			},
			topic:         "nfc_relay/user/anyuser/notifications",
			action:        "subscribe",
			expectedAllow: true,
			description:   "确保admin角色的特殊权限仍然有效",
		},

		// 回归测试: 确保现有权限规则未被破坏
		{
			name: "回归测试: 客户端专属命名空间仍然有效",
			claims: &request.MQTTClaims{
				Role:     "transmitter",
				Username: "admin",
				ClientID: "admin-transmitter-005",
			},
			topic:         "nfc_relay/clients/admin-transmitter-005/state",
			action:        "subscribe",
			expectedAllow: true,
			description:   "回归测试: 确保客户端专属命名空间权限未被破坏",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := api.authorizeMqttAction(tc.claims, tc.topic, tc.action)
			assert.Equal(t, tc.expectedAllow, result,
				"测试失败: %s\n描述: %s\n输入: role=%s, clientID=%s, topic=%s, action=%s",
				tc.name, tc.description, tc.claims.Role, tc.claims.ClientID, tc.topic, tc.action)
		})
	}
}

// TestExtractUsernameFromClientID 测试用户名提取逻辑
func TestExtractUsernameFromClientID(t *testing.T) {
	api := &MqttAuthApi{}

	testCases := []struct {
		clientID         string
		expectedUsername string
		description      string
	}{
		{
			clientID:         "admin-transmitter-005",
			expectedUsername: "admin",
			description:      "标准格式: admin用户transmitter角色",
		},
		{
			clientID:         "admin-receiver-003",
			expectedUsername: "admin",
			description:      "标准格式: admin用户receiver角色",
		},
		{
			clientID:         "user123-transmitter-001",
			expectedUsername: "user123",
			description:      "标准格式: user123用户",
		},
		{
			clientID:         "test-user-receiver-999",
			expectedUsername: "test",
			description:      "用户名包含破折号的情况(当前实现限制)",
		},
		{
			clientID:         "admin-transmitter",
			expectedUsername: "",
			description:      "格式不正确: 缺少序列号",
		},
		{
			clientID:         "admin",
			expectedUsername: "",
			description:      "格式不正确: 只有用户名",
		},
		{
			clientID:         "",
			expectedUsername: "",
			description:      "边界情况: 空ClientID",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.description, func(t *testing.T) {
			result := api.extractUsernameFromClientID(tc.clientID)
			assert.Equal(t, tc.expectedUsername, result,
				"用户名提取失败: %s\n输入ClientID: %s",
				tc.description, tc.clientID)
		})
	}
}
