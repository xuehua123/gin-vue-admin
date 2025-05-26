package config

// NfcRelay 定义了 NFC Relay 服务的相关配置参数
type NfcRelay struct {
	WebsocketPongWaitSec      int `mapstructure:"websocket-pong-wait-sec" json:"websocketPongWaitSec" yaml:"websocket-pong-wait-sec"`
	WebsocketMaxMessageBytes  int `mapstructure:"websocket-max-message-bytes" json:"websocketMaxMessageBytes" yaml:"websocket-max-message-bytes"`
	WebsocketWriteWaitSec     int `mapstructure:"websocket-write-wait-sec" json:"websocketWriteWaitSec" yaml:"websocket-write-wait-sec"`
	HubCheckIntervalSec       int `mapstructure:"hub-check-interval-sec" json:"hubCheckIntervalSec" yaml:"hub-check-interval-sec"`
	SessionInactiveTimeoutSec int `mapstructure:"session-inactive-timeout-sec" json:"sessionInactiveTimeoutSec" yaml:"session-inactive-timeout-sec"`
}
