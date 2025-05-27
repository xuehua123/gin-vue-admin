package admin_response

// NfcRelayConfigResponse NFC Relay配置响应
type NfcRelayConfigResponse struct {
	WebsocketPongWaitSec      int `json:"websocketPongWaitSec"`      // WebSocket Pong等待时间（秒）
	WebsocketMaxMessageBytes  int `json:"websocketMaxMessageBytes"`  // WebSocket最大消息字节数
	WebsocketWriteWaitSec     int `json:"websocketWriteWaitSec"`     // WebSocket写入等待时间（秒）
	HubCheckIntervalSec       int `json:"hubCheckIntervalSec"`       // Hub检查间隔（秒）
	SessionInactiveTimeoutSec int `json:"sessionInactiveTimeoutSec"` // 会话不活跃超时时间（秒）
}
