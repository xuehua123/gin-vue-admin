package system

import (
	"time"

	"github.com/flipped-aurora/gin-vue-admin/server/global"
)

// SysUserDeviceLog 用户设备登录日志模型
type SysUserDeviceLog struct {
	global.GVA_MODEL
	UserID            string     `json:"userId" gorm:"column:user_id;type:char(36);index;comment:用户ID"`
	ClientID          string     `json:"clientId" gorm:"column:client_id;type:varchar(255);index;comment:客户端ID"`
	DeviceFingerprint string     `json:"deviceFingerprint" gorm:"column:device_fingerprint;type:varchar(255);index;comment:设备指纹"`
	DeviceModel       string     `json:"deviceModel" gorm:"column:device_model;type:varchar(255);comment:设备型号"`
	DeviceOS          string     `json:"deviceOs" gorm:"column:device_os;type:varchar(255);comment:设备操作系统"`
	AppVersion        string     `json:"appVersion" gorm:"column:app_version;type:varchar(255);comment:应用版本"`
	IPAddress         string     `json:"ipAddress" gorm:"column:ip_address;type:varchar(255);comment:登录IP"`
	UserAgent         string     `json:"userAgent" gorm:"column:user_agent;type:text;comment:用户代理"`
	LoginAt           *time.Time `json:"loginAt" gorm:"column:login_at;comment:登录时间"`
	LogoutAt          *time.Time `json:"logoutAt" gorm:"column:logout_at;comment:登出时间"`
	LogoutReason      string     `json:"logoutReason" gorm:"column:logout_reason;type:varchar(255);comment:登出原因"`
}

// TableName 设置表名
func (SysUserDeviceLog) TableName() string {
	return "sys_user_device_logs"
}
