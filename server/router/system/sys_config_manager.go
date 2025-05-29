package system

import (
	api "github.com/flipped-aurora/gin-vue-admin/server/api/v1"
	"github.com/flipped-aurora/gin-vue-admin/server/middleware"
	"github.com/gin-gonic/gin"
)

type ConfigManagerRouter struct{}

// InitConfigManagerRouter 初始化配置管理路由
func (s *ConfigManagerRouter) InitConfigManagerRouter(Router *gin.RouterGroup) (R gin.IRoutes) {
	configManagerRouter := Router.Group("configManager").Use(middleware.OperationRecord())
	configManagerRouterWithoutRecord := Router.Group("configManager")
	configManagerApi := api.ApiGroupApp.SystemApiGroup.ConfigManagerApi
	{
		configManagerRouter.POST("reloadMainConfig", configManagerApi.ReloadMainConfig)                     // 重载主配置
		configManagerRouter.POST("saveComplianceRules", configManagerApi.SaveComplianceRules)               // 保存合规规则
		configManagerRouter.POST("loadComplianceRules", configManagerApi.LoadComplianceRules)               // 加载合规规则
		configManagerRouter.POST("validateConfig", configManagerApi.ValidateConfiguration)                  // 验证配置
		configManagerRouter.POST("complianceRule", configManagerApi.CreateComplianceRule)                   // 创建合规规则
		configManagerRouter.PUT("complianceRule/:rule_id", configManagerApi.UpdateComplianceRule)           // 更新合规规则
		configManagerRouter.DELETE("complianceRule/:rule_id", configManagerApi.DeleteComplianceRule)        // 删除合规规则
		configManagerRouter.POST("batchUpdateComplianceRules", configManagerApi.BatchUpdateComplianceRules) // 批量更新合规规则
	}
	{
		configManagerRouterWithoutRecord.GET("complianceRules", configManagerApi.GetComplianceRules)         // 获取所有合规规则
		configManagerRouterWithoutRecord.GET("complianceRulesList", configManagerApi.GetComplianceRulesList) // 获取合规规则列表（分页）
		configManagerRouterWithoutRecord.GET("complianceRule/:rule_id", configManagerApi.GetComplianceRule)  // 获取单个合规规则
		configManagerRouterWithoutRecord.GET("changeHistory", configManagerApi.GetConfigChangeHistory)       // 获取配置变更历史
	}
	return configManagerRouter
}
