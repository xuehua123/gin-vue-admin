package nfc_relay

import (
	v1 "github.com/flipped-aurora/gin-vue-admin/server/api/v1"
	"github.com/flipped-aurora/gin-vue-admin/server/middleware"
	nfcRelayService "github.com/flipped-aurora/gin-vue-admin/server/service/nfc_relay"
	"github.com/gin-gonic/gin"
)

type NFCTransactionRouter struct{}

// InitNFCTransactionRouter 初始化NFC交易路由
func (r *NFCTransactionRouter) InitNFCTransactionRouter(Router *gin.RouterGroup, PublicRouter *gin.RouterGroup) {
	nfcTransactionApi := v1.ApiGroupApp.NFCRelayApiGroup.NFCTransactionApi

	// 私有路由组 - 需要JWT认证
	nfcTransactionRouter := Router.Group("nfc-relay").Use(middleware.OperationRecord())
	nfcTransactionRouterWithoutRecord := Router.Group("nfc-relay")

	// 公开路由组 - 无需认证（用于MQTT客户端等）
	nfcTransactionPublicRouter := PublicRouter.Group("nfc-relay")

	{
		// === 交易管理路由 ===

		// 创建交易
		nfcTransactionRouter.POST("transactions", nfcTransactionApi.CreateTransaction)

		// 获取交易列表
		nfcTransactionRouterWithoutRecord.GET("transactions", nfcTransactionApi.GetTransactionList)

		// 获取交易详情
		nfcTransactionRouterWithoutRecord.GET("transactions/:transaction_id", nfcTransactionApi.GetTransaction)

		// 更新交易状态
		nfcTransactionRouter.PUT("transactions/status", nfcTransactionApi.UpdateTransactionStatus)

		// 删除交易
		nfcTransactionRouter.DELETE("transactions/:transaction_id", nfcTransactionApi.DeleteTransaction)

		// 批量更新交易状态
		nfcTransactionRouter.PUT("transactions/batch-update", nfcTransactionApi.BatchUpdateStatus)

		// === APDU消息路由 ===

		// 发送APDU消息
		nfcTransactionRouter.POST("transactions/apdu", nfcTransactionApi.SendAPDU)

		// 获取APDU消息列表
		nfcTransactionRouterWithoutRecord.GET("transactions/apdu", nfcTransactionApi.GetAPDUList)

		// === 统计和导出路由 ===

		// 获取统计信息
		nfcTransactionRouterWithoutRecord.GET("transactions/statistics", nfcTransactionApi.GetStatistics)

		// 导出交易数据
		nfcTransactionRouterWithoutRecord.GET("transactions/export", nfcTransactionApi.ExportTransactions)

		// === WebSocket路由 ===

		// WebSocket连接（需要认证）
		nfcTransactionRouterWithoutRecord.GET("ws", nfcRelayService.HandleWebSocket)

		// 交易会话管理路由 - 新增
		nfcTransactionRouter.POST("/transactions/sessions/initiate", nfcTransactionApi.InitiateTransactionSession)  // 发起交易会话
		nfcTransactionRouter.POST("/transactions/sessions/join", nfcTransactionApi.JoinTransactionSession)          // 加入交易会话
		nfcTransactionRouter.GET("/transactions/sessions/:transaction_id", nfcTransactionApi.GetTransactionSession) // 获取会话状态
	}

	// 公开路由（用于系统集成）
	{
		// 健康检查
		nfcTransactionPublicRouter.GET("health", func(c *gin.Context) {
			c.JSON(200, gin.H{
				"status":  "ok",
				"service": "nfc-relay",
				"version": "1.0.0",
			})
		})

		// MQTT服务状态
		nfcTransactionPublicRouter.GET("mqtt/status", func(c *gin.Context) {
			// 这里可以添加MQTT服务状态检查
			c.JSON(200, gin.H{
				"mqtt_connected": true, // 实际应该检查MQTT连接状态
				"timestamp":      "2024-01-01T00:00:00Z",
			})
		})
	}
}
