package initialize

import (
	"bufio"
	"os"
	"strings"

	"github.com/songzhibin97/gkit/cache/local_cache"

	"github.com/flipped-aurora/gin-vue-admin/server/global"
	"github.com/flipped-aurora/gin-vue-admin/server/service"
	"github.com/flipped-aurora/gin-vue-admin/server/utils"
	"go.uber.org/zap"
)

func OtherInit() {
	dr, err := utils.ParseDuration(global.GVA_CONFIG.JWT.ExpiresTime)
	if err != nil {
		panic(err)
	}
	_, err = utils.ParseDuration(global.GVA_CONFIG.JWT.BufferTime)
	if err != nil {
		panic(err)
	}

	global.BlackCache = local_cache.NewCache(
		local_cache.SetDefaultExpire(dr),
	)
	file, err := os.Open("go.mod")
	if err == nil && global.GVA_CONFIG.AutoCode.Module == "" {
		scanner := bufio.NewScanner(file)
		scanner.Scan()
		global.GVA_CONFIG.AutoCode.Module = strings.TrimPrefix(scanner.Text(), "module ")
	}

	// 初始化配置管理服务
	InitConfigManager()
}

// InitConfigManager 初始化配置管理器
func InitConfigManager() {
	configManagerService := service.ServiceGroupApp.SystemServiceGroup.GetConfigManagerService()
	if configManagerService != nil {
		if err := configManagerService.InitializeConfigManager(); err != nil {
			// 检查logger是否已初始化，避免空指针异常
			if global.GVA_LOG != nil {
				global.GVA_LOG.Error("配置管理器初始化失败", zap.Error(err))
			} else {
				// 如果logger未初始化，使用标准输出
				println("配置管理器初始化失败:", err.Error())
			}
		} else {
			// 检查logger是否已初始化
			if global.GVA_LOG != nil {
				global.GVA_LOG.Info("配置管理器初始化成功")
			} else {
				// 如果logger未初始化，使用标准输出
				println("配置管理器初始化成功")
			}
		}
	}
}
