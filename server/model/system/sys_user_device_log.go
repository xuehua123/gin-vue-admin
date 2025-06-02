package system

import (
	"time"

	"github.com/flipped-aurora/gin-vue-admin/server/global"
	"github.com/google/uuid"
)

// UserDeviceLog 用户设备登录日志表
type UserDeviceLog struct {
	global.GVA_MODEL             // LogEntryID (作为主键), CreatedAt, UpdatedAt, DeletedAt
	UserID            uuid.UUID  `json:"userId" gorm:"type:char(36);comment:用户ID"`                      // 用户ID (外键关联 users.uuid)
	ClientID          string     `json:"clientId" gorm:"type:varchar(255);index;comment:客户端ID"`         // 本次登录会话的客户端ID
	DeviceFingerprint string     `json:"deviceFingerprint" gorm:"type:varchar(255);index;comment:设备指纹"` // 更持久的设备唯一标识
	DeviceModel       string     `json:"deviceModel" gorm:"type:varchar(255);comment:设备型号"`             // 例如："iPhone 15 Pro", "Pixel 8"
	DeviceOs          string     `json:"deviceOs" gorm:"type:varchar(255);comment:设备操作系统"`              // 例如："iOS 17.1", "Android 14"
	AppVersion        string     `json:"appVersion" gorm:"type:varchar(255);comment:应用版本"`              // 例如："1.0.1"
	IpAddress         string     `json:"ipAddress" gorm:"type:varchar(255);comment:登录IP"`               // 登录时IP (INET 类型在GORM中通常映射为 string)
	UserAgent         string     `json:"userAgent" gorm:"type:text;comment:用户代理"`                       // 客户端HTTP请求的User-Agent
	LoginAt           time.Time  `json:"loginAt" gorm:"comment:登录时间"`                                   // 本次登录时间
	LogoutAt          *time.Time `json:"logoutAt" gorm:"comment:登出时间"`                                  // 本次登出或被踢下线时间 (使用指针以允许 NULL)
	LogoutReason      string     `json:"logoutReason" gorm:"type:varchar(255);comment:登出原因"`            // 例如："user_logout", "kicked_by_new_login"
}

func (UserDeviceLog) TableName() string {
	return "sys_user_device_logs"
}
