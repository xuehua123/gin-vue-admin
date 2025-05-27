package admin_request

// ClientListParams 定义了获取客户端列表的请求参数
type ClientListParams struct {
	Page      int    `json:"page" form:"page"`           // 页码
	PageSize  int    `json:"pageSize" form:"pageSize"`   // 每页条数
	ClientID  string `json:"clientID" form:"clientID"`   // 按客户端ID筛选
	UserID    string `json:"userID" form:"userID"`       // 按用户ID筛选
	Role      string `json:"role" form:"role"`           // 按角色筛选 (provider/receiver/none)
	IPAddress string `json:"ipAddress" form:"ipAddress"` // 按IP地址筛选
	SessionID string `json:"sessionID" form:"sessionID"` // 按会话ID筛选
	IsOnline  *bool  `json:"isOnline" form:"isOnline"`   // 按在线状态筛选
}

// ClientDetailParams 定义了获取客户端详细信息的请求参数
type ClientDetailParams struct {
	ClientID string `json:"clientID" form:"clientID" uri:"clientID"` // 客户端ID
}

// ClientDisconnectParams 定义了断开客户端连接的请求参数
type ClientDisconnectParams struct {
	ClientID string `json:"clientID" form:"clientID" uri:"clientID"` // 客户端ID
}
