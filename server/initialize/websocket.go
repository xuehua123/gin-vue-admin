package initialize

import (
	"github.com/flipped-aurora/gin-vue-admin/server/global"
	"github.com/flipped-aurora/gin-vue-admin/server/nfc_relay/handler"
	"github.com/flipped-aurora/gin-vue-admin/server/nfc_relay/service"
	"go.uber.org/zap"
)

var (
	// RealtimeService 全局实时数据服务实例
	RealtimeService *service.RealtimeDataService
)

// InitWebSocketService 初始化WebSocket实时数据服务
func InitWebSocketService() {
	RealtimeService = service.NewRealtimeDataService(handler.GlobalRelayHub)

	// 设置handler包中的全局变量，避免循环依赖
	handler.SetRealtimeDataService(RealtimeService)

	global.GVA_LOG.Info("WebSocket实时数据服务已初始化", zap.String("service", "RealtimeDataService"))
}

// GetRealtimeService 获取WebSocket服务实例
func GetRealtimeService() *service.RealtimeDataService {
	return RealtimeService
}
