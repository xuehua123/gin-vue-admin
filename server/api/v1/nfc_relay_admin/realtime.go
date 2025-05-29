package nfc_relay_admin

import (
	"net/http"

	"github.com/flipped-aurora/gin-vue-admin/server/global"
	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

type RealtimeApi struct{}

// HandleWebSocket 处理WebSocket连接
// @Summary WebSocket实时数据
// @Description 建立WebSocket连接获取实时数据
// @Tags NFC中继管理
// @Accept json
// @Produce json
// @Success 101 {string} string "Switching Protocols"
// @Router /api/admin/nfc-relay/v1/realtime [get]
func (r *RealtimeApi) HandleWebSocket(ctx *gin.Context) {
	// 记录WebSocket连接请求
	global.GVA_LOG.Info("WebSocket connection request",
		zap.String("remote_addr", ctx.ClientIP()),
		zap.String("user_agent", ctx.GetHeader("User-Agent")),
	)

	// 这里暂时返回一个简单的HTTP响应，表示WebSocket功能正在开发中
	// 实际的WebSocket实现需要在路由层面单独处理
	ctx.JSON(http.StatusOK, gin.H{
		"message": "WebSocket实时数据功能正在开发中",
		"status":  "coming_soon",
	})
}
