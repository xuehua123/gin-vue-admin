package response

import (
	"time"

	"github.com/flipped-aurora/gin-vue-admin/server/model/system"
)

type SysUserResponse struct {
	User system.SysUser `json:"user"`
}

type LoginResponse struct {
	User      system.SysUser `json:"user"`
	Token     string         `json:"token"`
	ExpiresAt int64          `json:"expiresAt"`
}

// UserWithOnlineStatus 带在线状态的用户信息
type UserWithOnlineStatus struct {
	system.SysUser
	OnlineStatus *OnlineStatusInfo `json:"onlineStatus"`
	DeviceInfo   []DeviceInfo      `json:"deviceInfo"`
	RoleInfo     *RoleInfo         `json:"roleInfo"`
}

// OnlineStatusInfo 在线状态信息
type OnlineStatusInfo struct {
	IsOnline       bool      `json:"isOnline"`
	OnlineCount    int       `json:"onlineCount"`
	LastActiveTime time.Time `json:"lastActiveTime"`
}

// DeviceInfo 设备信息
type DeviceInfo struct {
	ClientID       string    `json:"clientId"`
	DeviceModel    string    `json:"deviceModel"`
	DeviceOS       string    `json:"deviceOs"`
	AppVersion     string    `json:"appVersion"`
	IPAddress      string    `json:"ipAddress"`
	LoginTime      time.Time `json:"loginTime"`
	LastActiveTime time.Time `json:"lastActiveTime"`
	CurrentScreen  string    `json:"currentScreen"`
	NetworkInfo    string    `json:"networkInfo"`
	IsOnline       bool      `json:"isOnline"`
}

// RoleInfo 角色信息
type RoleInfo struct {
	CurrentRole       string    `json:"currentRole"` // "transmitter", "receiver", "none"
	RoleSetTime       time.Time `json:"roleSetTime"`
	NFCStatus         string    `json:"nfcStatus"`
	HCEStatus         string    `json:"hceStatus"`
	PeerClientID      string    `json:"peerClientId"`
	TransactionStatus string    `json:"transactionStatus"`
}

// DeviceLogResponse 设备日志响应
type DeviceLogResponse struct {
	ID                uint       `json:"id"`
	UserID            string     `json:"userId"`
	Username          string     `json:"username"`
	NickName          string     `json:"nickName"`
	ClientID          string     `json:"clientId"`
	DeviceFingerprint string     `json:"deviceFingerprint"`
	DeviceModel       string     `json:"deviceModel"`
	DeviceOS          string     `json:"deviceOs"`
	AppVersion        string     `json:"appVersion"`
	IPAddress         string     `json:"ipAddress"`
	LoginAt           time.Time  `json:"loginAt"`
	LogoutAt          *time.Time `json:"logoutAt"`
	LogoutReason      string     `json:"logoutReason"`
	SessionDuration   int64      `json:"sessionDuration"` // 会话时长（秒）
	IsOnline          bool       `json:"isOnline"`        // 当前是否在线
	Location          string     `json:"location"`        // IP归属地
}

// DeviceLogStats 设备日志统计
type DeviceLogStats struct {
	TotalLogins    int64     `json:"totalLogins"`
	UniqueDevices  int64     `json:"uniqueDevices"`
	CurrentOnline  int64     `json:"currentOnline"`
	AvgSessionTime float64   `json:"avgSessionTime"`
	LastLoginTime  time.Time `json:"lastLoginTime"`
	MostUsedDevice string    `json:"mostUsedDevice"`
}
