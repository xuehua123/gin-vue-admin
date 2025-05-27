package admin_request

import "time"

// AuditLogListParams 审计日志查询参数
type AuditLogListParams struct {
	Page      int    `form:"page,default=1" json:"page"`          // 页码
	PageSize  int    `form:"pageSize,default=10" json:"pageSize"` // 每页条数
	EventType string `form:"eventType" json:"eventType"`          // 事件类型
	UserID    string `form:"userID" json:"userID"`                // 用户ID
	SessionID string `form:"sessionID" json:"sessionID"`          // 会话ID
	ClientID  string `form:"clientID" json:"clientID"`            // 客户端ID
	StartTime string `form:"startTime" json:"startTime"`          // 开始时间（ISO8601格式）
	EndTime   string `form:"endTime" json:"endTime"`              // 结束时间（ISO8601格式）
}

// GetStartTimeAsTime 将字符串开始时间转换为time.Time
func (p *AuditLogListParams) GetStartTimeAsTime() (time.Time, error) {
	if p.StartTime == "" {
		// 如果未提供，返回一个很早的时间
		return time.Date(1970, 1, 1, 0, 0, 0, 0, time.UTC), nil
	}
	return time.Parse(time.RFC3339, p.StartTime)
}

// GetEndTimeAsTime 将字符串结束时间转换为time.Time
func (p *AuditLogListParams) GetEndTimeAsTime() (time.Time, error) {
	if p.EndTime == "" {
		// 如果未提供，返回当前时间
		return time.Now(), nil
	}
	return time.Parse(time.RFC3339, p.EndTime)
}
