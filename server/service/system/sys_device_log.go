package system

import (
	"time"

	"github.com/flipped-aurora/gin-vue-admin/server/global"
	"github.com/flipped-aurora/gin-vue-admin/server/model/system"
	systemReq "github.com/flipped-aurora/gin-vue-admin/server/model/system/request"
	systemRes "github.com/flipped-aurora/gin-vue-admin/server/model/system/response"
)

type DeviceLogService struct{}

// GetDeviceLogsList 获取设备日志列表
func (s *DeviceLogService) GetDeviceLogsList(info systemReq.GetDeviceLogsRequest) (list []systemRes.DeviceLogResponse, total int64, err error) {
	limit := info.PageSize
	offset := info.PageSize * (info.Page - 1)

	db := global.GVA_DB.Table("sys_user_device_logs").
		Select("sys_user_device_logs.*, sys_users.username, sys_users.nick_name").
		Joins("LEFT JOIN sys_users ON sys_user_device_logs.user_id = sys_users.uuid")

	// 条件筛选
	if info.UserID != "" {
		db = db.Where("sys_user_device_logs.user_id = ?", info.UserID)
	}
	if info.ClientID != "" {
		db = db.Where("sys_user_device_logs.client_id LIKE ?", "%"+info.ClientID+"%")
	}
	if info.DeviceModel != "" {
		db = db.Where("sys_user_device_logs.device_model LIKE ?", "%"+info.DeviceModel+"%")
	}
	if info.IPAddress != "" {
		db = db.Where("sys_user_device_logs.ip_address = ?", info.IPAddress)
	}
	if info.LoginTimeStart != nil {
		db = db.Where("sys_user_device_logs.login_at >= ?", info.LoginTimeStart)
	}
	if info.LoginTimeEnd != nil {
		db = db.Where("sys_user_device_logs.login_at <= ?", info.LoginTimeEnd)
	}
	if info.OnlineOnly {
		db = db.Where("sys_user_device_logs.logout_at IS NULL")
	}

	// 计算总数
	err = db.Count(&total).Error
	if err != nil {
		return
	}

	// 查询数据
	var logs []struct {
		ID                uint       `json:"id"`
		UserID            string     `json:"user_id"`
		Username          string     `json:"username"`
		NickName          string     `json:"nick_name"`
		ClientID          string     `json:"client_id"`
		DeviceFingerprint string     `json:"device_fingerprint"`
		DeviceModel       string     `json:"device_model"`
		DeviceOS          string     `json:"device_os"`
		AppVersion        string     `json:"app_version"`
		IPAddress         string     `json:"ip_address"`
		LoginAt           *time.Time `json:"login_at"`
		LogoutAt          *time.Time `json:"logout_at"`
		LogoutReason      string     `json:"logout_reason"`
	}

	err = db.Order("sys_user_device_logs.login_at DESC").
		Limit(limit).Offset(offset).
		Find(&logs).Error
	if err != nil {
		return
	}

	// 创建在线状态服务实例
	onlineService := &UserOnlineService{}

	// 转换为响应格式
	for _, log := range logs {
		logResponse := systemRes.DeviceLogResponse{
			ID:                log.ID,
			UserID:            log.UserID,
			Username:          log.Username,
			NickName:          log.NickName,
			ClientID:          log.ClientID,
			DeviceFingerprint: log.DeviceFingerprint,
			DeviceModel:       log.DeviceModel,
			DeviceOS:          log.DeviceOS,
			AppVersion:        log.AppVersion,
			IPAddress:         log.IPAddress,
			LogoutReason:      log.LogoutReason,
			Location:          "", // TODO: 实现IP归属地查询
		}

		if log.LoginAt != nil {
			logResponse.LoginAt = *log.LoginAt
		}
		if log.LogoutAt != nil {
			logResponse.LogoutAt = log.LogoutAt
			// 计算会话时长
			logResponse.SessionDuration = int64(log.LogoutAt.Sub(*log.LoginAt).Seconds())
		}

		// 检查是否在线（无登出时间且Redis中存在状态）
		if log.LogoutAt == nil {
			onlineStatus, err := onlineService.GetUserOnlineStatus(log.UserID)
			if err == nil && onlineStatus != nil {
				// 检查该客户端是否在线
				devices, err := onlineService.GetUserDevices(log.UserID)
				if err == nil {
					for _, device := range devices {
						if device.ClientID == log.ClientID && device.IsOnline {
							logResponse.IsOnline = true
							break
						}
					}
				}
			}
		}

		list = append(list, logResponse)
	}

	return list, total, err
}

// GetDeviceLogStats 获取设备日志统计
func (s *DeviceLogService) GetDeviceLogStats(userID string) (*systemRes.DeviceLogStats, error) {
	var stats systemRes.DeviceLogStats

	db := global.GVA_DB.Model(&system.SysUserDeviceLog{})
	if userID != "" {
		db = db.Where("user_id = ?", userID)
	}

	// 总登录次数
	db.Count(&stats.TotalLogins)

	// 唯一设备数
	var uniqueDevicesCount int64
	global.GVA_DB.Model(&system.SysUserDeviceLog{}).
		Select("DISTINCT device_fingerprint").
		Where("device_fingerprint != '' AND device_fingerprint IS NOT NULL").
		Count(&uniqueDevicesCount)
	stats.UniqueDevices = uniqueDevicesCount

	// 当前在线数
	var currentOnlineCount int64
	global.GVA_DB.Model(&system.SysUserDeviceLog{}).
		Where("logout_at IS NULL").
		Count(&currentOnlineCount)
	stats.CurrentOnline = currentOnlineCount

	// 平均会话时长
	var avgDuration struct {
		AvgSeconds float64 `json:"avg_seconds"`
	}
	global.GVA_DB.Raw(`
		SELECT AVG(TIMESTAMPDIFF(SECOND, login_at, logout_at)) as avg_seconds 
		FROM sys_user_device_logs 
		WHERE logout_at IS NOT NULL AND login_at IS NOT NULL
	`).Scan(&avgDuration)
	stats.AvgSessionTime = avgDuration.AvgSeconds

	// 最后登录时间
	var lastLogin system.SysUserDeviceLog
	err := global.GVA_DB.Order("login_at DESC").First(&lastLogin).Error
	if err == nil && lastLogin.LoginAt != nil {
		stats.LastLoginTime = *lastLogin.LoginAt
	}

	// 最常用设备
	var mostUsedDevice struct {
		DeviceModel string `json:"device_model"`
		Count       int64  `json:"count"`
	}
	global.GVA_DB.Model(&system.SysUserDeviceLog{}).
		Select("device_model, COUNT(*) as count").
		Group("device_model").
		Order("count DESC").
		First(&mostUsedDevice)
	stats.MostUsedDevice = mostUsedDevice.DeviceModel

	return &stats, nil
}

// ForceLogoutDevice 强制设备下线
func (s *DeviceLogService) ForceLogoutDevice(userID, clientID, reason string) error {
	onlineService := &UserOnlineService{}

	// 调用在线服务的强制下线方法
	err := onlineService.ForceLogoutUser(userID, clientID)
	if err != nil {
		return err
	}

	// 更新登出原因
	if reason != "" {
		err = global.GVA_DB.Model(&system.SysUserDeviceLog{}).
			Where("client_id = ? AND logout_at IS NOT NULL", clientID).
			Update("logout_reason", reason).Error
	}

	return err
}
