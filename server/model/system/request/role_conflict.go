package request

// CheckRoleConflictRequest 角色冲突检测请求结构体
type CheckRoleConflictRequest struct {
	Role       string                 `json:"role" binding:"required"`
	ClientID   string                 `json:"client_id" binding:"required"`
	DeviceInfo map[string]interface{} `json:"device_info"`
}

// AssignRoleRequest 分配角色请求结构体（用于生成Token）
type AssignRoleRequest struct {
	Role              string                 `json:"role" binding:"required"`
	ForceKickExisting bool                   `json:"force_kick_existing"`
	DeviceInfo        map[string]interface{} `json:"device_info"`
}

// ConflictCheckResult 角色冲突检测响应结构体
type ConflictCheckResult struct {
	HasConflict    bool                `json:"has_conflict"`
	ConflictDevice *ConflictDeviceInfo `json:"conflict_device,omitempty"`
	CanForceKick   bool                `json:"can_force_kick"`
}

// ConflictDeviceInfo 冲突设备信息结构体
type ConflictDeviceInfo struct {
	ClientID     string `json:"client_id"`
	DeviceModel  string `json:"device_model"`
	ConnectedAt  string `json:"connected_at"`
	LastActivity string `json:"last_activity"`
}

// MQTTTokenResponse MQTT Token响应结构体
type MQTTTokenResponse struct {
	ClientID string `json:"client_id"`
	Token    string `json:"token"`
	Role     string `json:"role"`
}
