package request

import (
	"time"

	"github.com/flipped-aurora/gin-vue-admin/server/model/common/request"
)

// GetDeviceLogsRequest 获取设备日志列表请求
type GetDeviceLogsRequest struct {
	request.PageInfo
	UserID         string     `json:"userId" form:"userId"`                 // 用户ID筛选
	ClientID       string     `json:"clientId" form:"clientId"`             // 客户端ID筛选
	DeviceModel    string     `json:"deviceModel" form:"deviceModel"`       // 设备型号筛选
	IPAddress      string     `json:"ipAddress" form:"ipAddress"`           // IP地址筛选
	LoginTimeStart *time.Time `json:"loginTimeStart" form:"loginTimeStart"` // 登录时间开始
	LoginTimeEnd   *time.Time `json:"loginTimeEnd" form:"loginTimeEnd"`     // 登录时间结束
	OnlineOnly     bool       `json:"onlineOnly" form:"onlineOnly"`         // 只显示在线用户
}

// ForceLogoutRequest 强制下线请求
type ForceLogoutRequest struct {
	UserID   string `json:"userId" binding:"required"`   // 用户ID
	ClientID string `json:"clientId" binding:"required"` // 客户端ID
	Reason   string `json:"reason"`                      // 下线原因
}
