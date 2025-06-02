package system

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/flipped-aurora/gin-vue-admin/server/global"
	"github.com/flipped-aurora/gin-vue-admin/server/model/system"
	systemRes "github.com/flipped-aurora/gin-vue-admin/server/model/system/response"
	"go.uber.org/zap"
)

type UserOnlineService struct{}

// GetUserOnlineStatus 获取用户在线状态
func (s *UserOnlineService) GetUserOnlineStatus(userID string) (*systemRes.OnlineStatusInfo, error) {
	ctx := context.Background()

	// 获取用户所有活跃的JWT
	pattern := fmt.Sprintf("jwt_active:%s:*", userID)
	keys, err := global.GVA_REDIS.Keys(ctx, pattern).Result()
	if err != nil {
		global.GVA_LOG.Error("获取用户JWT键值失败", zap.Error(err))
		return &systemRes.OnlineStatusInfo{IsOnline: false, OnlineCount: 0}, nil
	}

	onlineCount := len(keys)
	isOnline := onlineCount > 0

	// 获取最后活跃时间
	var lastActiveTime time.Time
	if isOnline {
		// 从client_state中获取最后活跃时间
		for _, key := range keys {
			parts := strings.Split(key, ":")
			if len(parts) >= 3 {
				clientID, err := global.GVA_REDIS.Get(ctx, key).Result()
				if err != nil {
					continue
				}

				stateKey := fmt.Sprintf("client_state:%s", clientID)
				lastEventStr, err := global.GVA_REDIS.HGet(ctx, stateKey, "last_event_timestamp_utc").Result()
				if err != nil {
					continue
				}

				if lastEventTime, err := time.Parse(time.RFC3339, lastEventStr); err == nil {
					if lastEventTime.After(lastActiveTime) {
						lastActiveTime = lastEventTime
					}
				}
			}
		}
	}

	return &systemRes.OnlineStatusInfo{
		IsOnline:       isOnline,
		OnlineCount:    onlineCount,
		LastActiveTime: lastActiveTime,
	}, nil
}

// GetUserDevices 获取用户设备信息
func (s *UserOnlineService) GetUserDevices(userID string) ([]systemRes.DeviceInfo, error) {
	ctx := context.Background()
	var devices []systemRes.DeviceInfo

	// 获取用户所有活跃的JWT
	pattern := fmt.Sprintf("jwt_active:%s:*", userID)
	keys, err := global.GVA_REDIS.Keys(ctx, pattern).Result()
	if err != nil {
		global.GVA_LOG.Error("获取用户JWT键值失败", zap.Error(err))
		return devices, err
	}

	for _, key := range keys {
		clientID, err := global.GVA_REDIS.Get(ctx, key).Result()
		if err != nil {
			continue
		}

		// 获取客户端状态
		stateKey := fmt.Sprintf("client_state:%s", clientID)
		stateData, err := global.GVA_REDIS.HGetAll(ctx, stateKey).Result()
		if err != nil {
			continue
		}

		device := systemRes.DeviceInfo{
			ClientID:      clientID,
			DeviceModel:   stateData["device_model"],
			DeviceOS:      stateData["device_os"],
			AppVersion:    stateData["app_version"],
			IPAddress:     stateData["ip_address"],
			CurrentScreen: stateData["current_screen"],
			NetworkInfo:   stateData["network_info"],
			IsOnline:      stateData["is_online"] == "true",
		}

		// 解析时间
		if loginTime, err := time.Parse(time.RFC3339, stateData["mqtt_connected_at_utc"]); err == nil {
			device.LoginTime = loginTime
		}
		if lastActiveTime, err := time.Parse(time.RFC3339, stateData["last_event_timestamp_utc"]); err == nil {
			device.LastActiveTime = lastActiveTime
		}

		devices = append(devices, device)
	}

	return devices, nil
}

// GetUserRoleInfo 获取用户角色信息
func (s *UserOnlineService) GetUserRoleInfo(userID string) (*systemRes.RoleInfo, error) {
	ctx := context.Background()

	// 获取用户角色信息
	roleKey := fmt.Sprintf("user_roles:%s", userID)
	roleData, err := global.GVA_REDIS.HGetAll(ctx, roleKey).Result()
	if err != nil {
		global.GVA_LOG.Error("获取用户角色信息失败", zap.Error(err))
		return nil, err
	}

	if len(roleData) == 0 {
		return &systemRes.RoleInfo{CurrentRole: "none"}, nil
	}

	roleInfo := &systemRes.RoleInfo{CurrentRole: "none"}

	// 确定当前角色
	if transmitterClientID := roleData["transmitter_client_id"]; transmitterClientID != "" {
		roleInfo.CurrentRole = "transmitter"
		if setTime, err := time.Parse(time.RFC3339, roleData["transmitter_set_at_utc"]); err == nil {
			roleInfo.RoleSetTime = setTime
		}

		// 获取传卡端状态
		stateKey := fmt.Sprintf("client_state:%s", transmitterClientID)
		stateData, err := global.GVA_REDIS.HGetAll(ctx, stateKey).Result()
		if err == nil {
			roleInfo.NFCStatus = stateData["nfc_status_transmitter"]
		}
	} else if receiverClientID := roleData["receiver_client_id"]; receiverClientID != "" {
		roleInfo.CurrentRole = "receiver"
		if setTime, err := time.Parse(time.RFC3339, roleData["receiver_set_at_utc"]); err == nil {
			roleInfo.RoleSetTime = setTime
		}

		// 获取收卡端状态
		stateKey := fmt.Sprintf("client_state:%s", receiverClientID)
		stateData, err := global.GVA_REDIS.HGetAll(ctx, stateKey).Result()
		if err == nil {
			roleInfo.HCEStatus = stateData["hce_status_receiver"]
		}
	}

	return roleInfo, nil
}

// ForceLogoutUser 强制用户下线
func (s *UserOnlineService) ForceLogoutUser(userID, clientID string) error {
	ctx := context.Background()

	// 1. 从Redis中移除JWT活跃状态
	pattern := fmt.Sprintf("jwt_active:%s:*", userID)
	keys, err := global.GVA_REDIS.Keys(ctx, pattern).Result()
	if err != nil {
		return err
	}

	for _, key := range keys {
		storedClientID, err := global.GVA_REDIS.Get(ctx, key).Result()
		if err != nil {
			continue
		}
		if storedClientID == clientID {
			global.GVA_REDIS.Del(ctx, key)
			break
		}
	}

	// 2. 清理客户端状态
	stateKey := fmt.Sprintf("client_state:%s", clientID)
	global.GVA_REDIS.Del(ctx, stateKey)

	// 3. 清理用户角色（如果是该客户端设置的）
	roleKey := fmt.Sprintf("user_roles:%s", userID)
	roleData, err := global.GVA_REDIS.HGetAll(ctx, roleKey).Result()
	if err == nil {
		if roleData["transmitter_client_id"] == clientID {
			global.GVA_REDIS.HDel(ctx, roleKey, "transmitter_client_id", "transmitter_set_at_utc")
		}
		if roleData["receiver_client_id"] == clientID {
			global.GVA_REDIS.HDel(ctx, roleKey, "receiver_client_id", "receiver_set_at_utc")
		}
	}

	// 4. 更新数据库日志
	now := time.Now()
	err = global.GVA_DB.Model(&system.SysUserDeviceLog{}).
		Where("client_id = ? AND logout_at IS NULL", clientID).
		Updates(map[string]interface{}{
			"logout_at":     &now,
			"logout_reason": "forced_logout_by_admin",
			"updated_at":    now,
		}).Error

	return err
}

// GetAllOnlineUsers 获取所有在线用户统计
func (s *UserOnlineService) GetAllOnlineUsers() (map[string]systemRes.OnlineStatusInfo, error) {
	ctx := context.Background()
	result := make(map[string]systemRes.OnlineStatusInfo)

	// 获取所有活跃JWT
	pattern := "jwt_active:*"
	keys, err := global.GVA_REDIS.Keys(ctx, pattern).Result()
	if err != nil {
		return result, err
	}

	userStats := make(map[string]int)
	userLastActive := make(map[string]time.Time)

	for _, key := range keys {
		parts := strings.Split(key, ":")
		if len(parts) >= 3 {
			userID := parts[1]
			userStats[userID]++

			// 获取最后活跃时间
			clientID, err := global.GVA_REDIS.Get(ctx, key).Result()
			if err != nil {
				continue
			}

			stateKey := fmt.Sprintf("client_state:%s", clientID)
			lastEventStr, err := global.GVA_REDIS.HGet(ctx, stateKey, "last_event_timestamp_utc").Result()
			if err != nil {
				continue
			}

			if lastEventTime, err := time.Parse(time.RFC3339, lastEventStr); err == nil {
				if lastEventTime.After(userLastActive[userID]) {
					userLastActive[userID] = lastEventTime
				}
			}
		}
	}

	for userID, count := range userStats {
		result[userID] = systemRes.OnlineStatusInfo{
			IsOnline:       true,
			OnlineCount:    count,
			LastActiveTime: userLastActive[userID],
		}
	}

	return result, nil
}
