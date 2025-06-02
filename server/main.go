package main

import (
	"github.com/flipped-aurora/gin-vue-admin/server/core"
	"github.com/flipped-aurora/gin-vue-admin/server/global"
	"github.com/flipped-aurora/gin-vue-admin/server/initialize"
	"github.com/flipped-aurora/gin-vue-admin/server/nfc_relay/service"
	_ "go.uber.org/automaxprocs"
	"go.uber.org/zap"
)

//go:generate go env -w GO111MODULE=on
//go:generate go env -w GOPROXY=https://goproxy.cn,direct
//go:generate go mod tidy
//go:generate go mod download

// 这部分 @Tag 设置用于排序, 需要排序的接口请按照下面的格式添加
// swag init 对 @Tag 只会从入口文件解析, 默认 main.go
// 也可通过 --generalInfo flag 指定其他文件
// @Tag.Name        Base
// @Tag.Name        SysUser
// @Tag.Description 用户

// @title                       Gin-Vue-Admin Swagger API接口文档
// @version                     v2.8.2
// @description                 使用gin+vue进行极速开发的全栈开发基础平台
// @securityDefinitions.apikey  ApiKeyAuth
// @in                          header
// @name                        x-token
// @BasePath                    /

// 全局WebSocket服务实例
var realtimeService *service.RealtimeDataService

func main() {
	// 初始化系统
	initializeSystem()
	// 运行服务器
	core.RunServer()
}

// initializeSystem 初始化系统所有组件
// 提取为单独函数以便于系统重载时调用
func initializeSystem() {
	global.GVA_VP = core.Viper()   // 初始化Viper
	global.GVA_LOG = core.Zap()    // 初始化zap日志库 (必须在其他使用logger的初始化之前)
	global.InitializeAuditLogger() // 初始化审计日志记录器
	zap.ReplaceGlobals(global.GVA_LOG)

	initialize.OtherInit()            // 移到logger初始化之后
	global.GVA_DB = initialize.Gorm() // gorm连接数据库
	initialize.Timer()
	initialize.DBList()
	initialize.SetupHandlers() // 注册全局函数
	if global.GVA_DB != nil {
		initialize.RegisterTables() // 初始化表
	}

	//// 设置全局NFC中继Hub变量
	//global.GVA_NFC_RELAY_HUB = handler.GlobalRelayHub
	//
	//// 启动 NFC Relay Hub
	//go handler.GlobalRelayHub.Run()
	//global.GVA_LOG.Info("NFC中继服务已启动")
	//
	//// 🔥 关键修复：在路由初始化之前初始化WebSocket服务
	//initialize.InitWebSocketService()
	//global.GVA_LOG.Info("WebSocket实时数据服务初始化完成，准备注册路由")
}

// GetRealtimeService 获取全局WebSocket服务实例
