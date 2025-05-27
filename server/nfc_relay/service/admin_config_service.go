package service

import (
	"github.com/flipped-aurora/gin-vue-admin/server/global"
	"github.com/flipped-aurora/gin-vue-admin/server/nfc_relay/model/admin_response"
)

// AdminConfigService 配置服务
type AdminConfigService struct{}

// GetNfcRelayConfig 获取NFC Relay配置
func (s *AdminConfigService) GetNfcRelayConfig() admin_response.NfcRelayConfigResponse {
	// 直接返回全局配置中的NfcRelay配置
	config := global.GVA_CONFIG.NfcRelay

	return admin_response.NfcRelayConfigResponse{
		WebsocketPongWaitSec:      config.WebsocketPongWaitSec,
		WebsocketMaxMessageBytes:  config.WebsocketMaxMessageBytes,
		WebsocketWriteWaitSec:     config.WebsocketWriteWaitSec,
		HubCheckIntervalSec:       config.HubCheckIntervalSec,
		SessionInactiveTimeoutSec: config.SessionInactiveTimeoutSec,
	}
}
