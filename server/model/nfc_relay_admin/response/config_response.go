package response

// NfcRelayConfigResponse NFC中继系统配置响应
type NfcRelayConfigResponse struct {
	HubCheckIntervalSec       int `json:"hub_check_interval_sec"`       // Hub检查间隔（秒）
	SessionInactiveTimeoutSec int `json:"session_inactive_timeout_sec"` // 会话非活动超时（秒）
	WebsocketWriteWaitSec     int `json:"websocket_write_wait_sec"`     // WebSocket写入等待时间（秒）
	WebsocketPongWaitSec      int `json:"websocket_pong_wait_sec"`      // WebSocket Pong等待时间（秒）
	WebsocketMaxMessageBytes  int `json:"websocket_max_message_bytes"`  // WebSocket最大消息字节数
}
