package config

// MQTT MQTT连接配置
type MQTT struct {
	Host         string `mapstructure:"host" json:"host" yaml:"host"`                            // MQTT Broker地址
	Port         int    `mapstructure:"port" json:"port" yaml:"port"`                            // MQTT Broker端口
	Username     string `mapstructure:"username" json:"username" yaml:"username"`                // MQTT用户名
	Password     string `mapstructure:"password" json:"password" yaml:"password"`                // MQTT密码
	ClientID     string `mapstructure:"client-id" json:"client-id" yaml:"client-id"`             // 服务器端MQTT客户端ID
	QoS          byte   `mapstructure:"qos" json:"qos" yaml:"qos"`                               // 默认QoS级别
	KeepAlive    int    `mapstructure:"keep-alive" json:"keep-alive" yaml:"keep-alive"`          // 心跳间隔(秒)
	CleanSession bool   `mapstructure:"clean-session" json:"clean-session" yaml:"clean-session"` // 清除会话
	UseTLS       bool   `mapstructure:"use-tls" json:"use-tls" yaml:"use-tls"`                   // 是否使用TLS

	// NFC中继相关配置
	NFCRelay NFCRelayMQTT `mapstructure:"nfc-relay" json:"nfc-relay" yaml:"nfc-relay"`
}

// NFCRelayMQTT NFC中继MQTT配置
type NFCRelayMQTT struct {
	// 主题前缀
	TopicPrefix string `mapstructure:"topic-prefix" json:"topic-prefix" yaml:"topic-prefix"`

	// 心跳配置
	HeartbeatInterval int `mapstructure:"heartbeat-interval" json:"heartbeat-interval" yaml:"heartbeat-interval"` // 心跳间隔(秒)
	HeartbeatTimeout  int `mapstructure:"heartbeat-timeout" json:"heartbeat-timeout" yaml:"heartbeat-timeout"`    // 心跳超时(秒)

	// 消息配置
	MessageTimeout int `mapstructure:"message-timeout" json:"message-timeout" yaml:"message-timeout"`    // 消息超时(秒)
	MaxMessageSize int `mapstructure:"max-message-size" json:"max-message-size" yaml:"max-message-size"` // 最大消息大小(字节)
	RetryAttempts  int `mapstructure:"retry-attempts" json:"retry-attempts" yaml:"retry-attempts"`       // 重试次数
	RetryInterval  int `mapstructure:"retry-interval" json:"retry-interval" yaml:"retry-interval"`       // 重试间隔(秒)

	// 权限控制
	EnablePermissionControl bool `mapstructure:"enable-permission-control" json:"enable-permission-control" yaml:"enable-permission-control"` // 启用权限控制

	// 监控配置
	EnableMetrics   bool `mapstructure:"enable-metrics" json:"enable-metrics" yaml:"enable-metrics"`       // 启用指标收集
	MetricsInterval int  `mapstructure:"metrics-interval" json:"metrics-interval" yaml:"metrics-interval"` // 指标收集间隔(秒)
}
