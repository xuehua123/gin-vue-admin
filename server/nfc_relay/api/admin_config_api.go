package api

import (
	"github.com/flipped-aurora/gin-vue-admin/server/global"
	"github.com/flipped-aurora/gin-vue-admin/server/model/common/response"
	"github.com/flipped-aurora/gin-vue-admin/server/nfc_relay/service"
	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

// AdminConfigApi 配置API处理器
type AdminConfigApi struct{}

var adminConfigService = service.AdminConfigService{}

// GetNfcRelayConfig
// @Tags NFCRelayAdmin
// @Summary 获取NFC Relay系统配置
// @Security ApiKeyAuth
// @Produce application/json
// @Success 200 {object} response.Response{data=admin_response.NfcRelayConfigResponse,msg=string} "获取成功"
// @Router /admin/nfc-relay/v1/config [get]
func (a *AdminConfigApi) GetNfcRelayConfig(c *gin.Context) {
	// 调用服务获取配置
	global.GVA_LOG.Info("开始获取NFC Relay配置")

	config := adminConfigService.GetNfcRelayConfig()

	global.GVA_LOG.Info("获取NFC Relay配置成功",
		zap.Int("hubCheckIntervalSec", config.HubCheckIntervalSec),
		zap.Int("sessionInactiveTimeoutSec", config.SessionInactiveTimeoutSec),
		zap.Int("websocketWriteWaitSec", config.WebsocketWriteWaitSec),
		zap.Int("websocketPongWaitSec", config.WebsocketPongWaitSec),
		zap.Int("websocketMaxMessageBytes", config.WebsocketMaxMessageBytes),
	)

	response.OkWithDetailed(config, "获取NFC Relay配置成功", c)
}
