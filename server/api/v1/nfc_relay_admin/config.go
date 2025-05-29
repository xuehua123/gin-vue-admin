package nfc_relay_admin

import (
	"github.com/flipped-aurora/gin-vue-admin/server/global"
	"github.com/flipped-aurora/gin-vue-admin/server/model/common/response"
	"github.com/flipped-aurora/gin-vue-admin/server/nfc_relay/service"
	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

type ConfigApi struct{}

// 复用旧版服务层
var adminConfigService = service.AdminConfigService{}

// GetConfig 获取系统配置
// @Summary 获取系统配置
// @Description 获取NFC中继系统的配置信息
// @Tags NFC中继管理
// @Accept json
// @Produce json
// @Success 200 {object} response.Response
// @Router /api/admin/nfc-relay/v1/config [get]
func (c *ConfigApi) GetConfig(ctx *gin.Context) {
	// 调用旧版服务获取配置
	global.GVA_LOG.Info("开始获取NFC Relay配置")

	config := adminConfigService.GetNfcRelayConfig()

	global.GVA_LOG.Info("获取NFC Relay配置成功",
		zap.Int("hubCheckIntervalSec", config.HubCheckIntervalSec),
		zap.Int("sessionInactiveTimeoutSec", config.SessionInactiveTimeoutSec),
		zap.Int("websocketWriteWaitSec", config.WebsocketWriteWaitSec),
		zap.Int("websocketPongWaitSec", config.WebsocketPongWaitSec),
		zap.Int("websocketMaxMessageBytes", config.WebsocketMaxMessageBytes),
	)

	response.OkWithDetailed(config, "获取NFC Relay配置成功", ctx)
}
