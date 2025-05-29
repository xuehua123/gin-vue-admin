package config

// NfcRelay 定义了 NFC Relay 服务的相关配置参数
type NfcRelay struct {
	WebsocketPongWaitSec      int `mapstructure:"websocket-pong-wait-sec" json:"websocketPongWaitSec" yaml:"websocket-pong-wait-sec"`
	WebsocketMaxMessageBytes  int `mapstructure:"websocket-max-message-bytes" json:"websocketMaxMessageBytes" yaml:"websocket-max-message-bytes"`
	WebsocketWriteWaitSec     int `mapstructure:"websocket-write-wait-sec" json:"websocketWriteWaitSec" yaml:"websocket-write-wait-sec"`
	HubCheckIntervalSec       int `mapstructure:"hub-check-interval-sec" json:"hubCheckIntervalSec" yaml:"hub-check-interval-sec"`
	SessionInactiveTimeoutSec int `mapstructure:"session-inactive-timeout-sec" json:"sessionInactiveTimeoutSec" yaml:"session-inactive-timeout-sec"`

	// 安全配置
	Security NfcRelaySecurity `mapstructure:"security" json:"security" yaml:"security"`
}

// NfcRelaySecurity 定义了 NFC Relay 安全相关配置
type NfcRelaySecurity struct {
	// TLS/WSS 配置
	EnableTLS bool   `mapstructure:"enable-tls" json:"enableTLS" yaml:"enable-tls"`
	CertFile  string `mapstructure:"cert-file" json:"certFile" yaml:"cert-file"`
	KeyFile   string `mapstructure:"key-file" json:"keyFile" yaml:"key-file"`
	ForceTLS  bool   `mapstructure:"force-tls" json:"forceTLS" yaml:"force-tls"`

	// 审计级加密配置 (更新配置项)
	EnableAuditEncryption bool   `mapstructure:"enable-audit-encryption" json:"enableAuditEncryption" yaml:"enable-audit-encryption"`
	EncryptionAlgorithm   string `mapstructure:"encryption-algorithm" json:"encryptionAlgorithm" yaml:"encryption-algorithm"`
	AuditKeyRotationHours int    `mapstructure:"audit-key-rotation-hours" json:"auditKeyRotationHours" yaml:"audit-key-rotation-hours"`

	// 合规审计配置
	EnableComplianceAudit     bool     `mapstructure:"enable-compliance-audit" json:"enableComplianceAudit" yaml:"enable-compliance-audit"`
	EnableDeepInspection      bool     `mapstructure:"enable-deep-inspection" json:"enableDeepInspection" yaml:"enable-deep-inspection"`
	MaxTransactionAmount      int64    `mapstructure:"max-transaction-amount" json:"maxTransactionAmount" yaml:"max-transaction-amount"`
	BlockedMerchantCategories []string `mapstructure:"blocked-merchant-categories" json:"blockedMerchantCategories" yaml:"blocked-merchant-categories"`

	// 端到端加密配置
	EnableE2EEncryption bool   `mapstructure:"enable-e2e-encryption" json:"enableE2EEncryption" yaml:"enable-e2e-encryption"`
	KeyExchangeMethod   string `mapstructure:"key-exchange-method" json:"keyExchangeMethod" yaml:"key-exchange-method"`

	// 防重放攻击配置
	EnableAntiReplay bool `mapstructure:"enable-anti-replay" json:"enableAntiReplay" yaml:"enable-anti-replay"`
	ReplayWindowMs   int  `mapstructure:"replay-window-ms" json:"replayWindowMs" yaml:"replay-window-ms"`
	MaxNonceCache    int  `mapstructure:"max-nonce-cache" json:"maxNonceCache" yaml:"max-nonce-cache"`

	// 输入验证配置
	MaxMessageSize      int  `mapstructure:"max-message-size" json:"maxMessageSize" yaml:"max-message-size"`
	EnableInputSanitize bool `mapstructure:"enable-input-sanitize" json:"enableInputSanitize" yaml:"enable-input-sanitize"`

	// 客户端认证配置
	RequireClientCert bool   `mapstructure:"require-client-cert" json:"requireClientCert" yaml:"require-client-cert"`
	ClientCAFile      string `mapstructure:"client-ca-file" json:"clientCAFile" yaml:"client-ca-file"`
}
