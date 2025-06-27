package initialize

import (
	"github.com/flipped-aurora/gin-vue-admin/server/global"
	"github.com/flipped-aurora/gin-vue-admin/server/service"
	"github.com/flipped-aurora/gin-vue-admin/server/service/nfc_relay"
	"go.uber.org/zap"
)

func Service() {
	// 启动系统通知服务
	service.ServiceGroupApp.SystemServiceGroup.NotificationService.Start()

	// 【企业级新增】启动MQTT服务
	// 确保MQTT配置存在才初始化
	if global.GVA_CONFIG.MQTT.Host != "" {
		mqttService := nfc_relay.ServiceGroupApp.MqttService()
		if err := mqttService.Initialize(); err != nil {
			global.GVA_LOG.Error("MQTT服务初始化失败", zap.Error(err))
			// 企业级处理：不panic，记录错误继续运行，但功能会受限
			global.GVA_LOG.Warn("MQTT功能将不可用，配对通知等功能将受影响")
		} else {
			global.GVA_LOG.Info("MQTT服务初始化成功，配对通知功能已启用")
		}
	} else {
		global.GVA_LOG.Warn("MQTT配置不存在，配对通知功能将不可用")
	}
}
