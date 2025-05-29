package system

import (
	"errors"
	"strconv"

	"github.com/flipped-aurora/gin-vue-admin/server/global"
	"github.com/flipped-aurora/gin-vue-admin/server/model/common/response"
	systemReq "github.com/flipped-aurora/gin-vue-admin/server/model/system/request"
	"github.com/flipped-aurora/gin-vue-admin/server/service"
	"github.com/flipped-aurora/gin-vue-admin/server/utils"
	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

type ConfigManagerApi struct{}

// @Tags ConfigManager
// @Summary 重载主配置
// @Security ApiKeyAuth
// @accept application/json
// @Produce application/json
// @Param data body systemReq.ReloadConfigRequest true "重载配置请求"
// @Success 200 {object} response.Response{msg=string} "重载成功"
// @Router /configManager/reloadMainConfig [post]
func (configManagerApi *ConfigManagerApi) ReloadMainConfig(c *gin.Context) {
	var req systemReq.ReloadConfigRequest
	err := c.ShouldBindJSON(&req)
	if err != nil {
		response.FailWithMessage(err.Error(), c)
		return
	}

	err = utils.Verify(req, utils.ReloadConfigVerify)
	if err != nil {
		response.FailWithMessage(err.Error(), c)
		return
	}

	configManagerService := service.ServiceGroupApp.SystemServiceGroup.GetConfigManagerService()
	if configManagerService == nil {
		response.FailWithMessage("配置管理服务未初始化", c)
		return
	}

	if err := configManagerService.ReloadMainConfig(); err != nil {
		global.GVA_LOG.Error("重载主配置失败!", zap.Error(err))
		response.FailWithMessage("重载主配置失败: "+err.Error(), c)
		return
	}

	response.OkWithMessage("主配置重载成功", c)
}

// @Tags ConfigManager
// @Summary 获取所有合规规则
// @Security ApiKeyAuth
// @accept application/json
// @Produce application/json
// @Success 200 {object} response.Response{data=config.ComplianceRules,msg=string} "获取成功"
// @Router /configManager/complianceRules [get]
func (configManagerApi *ConfigManagerApi) GetComplianceRules(c *gin.Context) {
	configManagerService := service.ServiceGroupApp.SystemServiceGroup.GetConfigManagerService()
	if configManagerService == nil {
		response.FailWithMessage("配置管理服务未初始化", c)
		return
	}

	rules, err := configManagerService.GetComplianceRules()
	if err != nil {
		global.GVA_LOG.Error("获取合规规则失败!", zap.Error(err))
		response.FailWithMessage("获取合规规则失败: "+err.Error(), c)
		return
	}

	response.OkWithDetailed(rules, "获取合规规则成功", c)
}

// @Tags ConfigManager
// @Summary 获取合规规则列表（分页）
// @Security ApiKeyAuth
// @accept application/json
// @Produce application/json
// @Param data query systemReq.GetComplianceRulesRequest true "分页获取合规规则"
// @Success 200 {object} response.Response{data=response.PageResult,msg=string} "获取成功"
// @Router /configManager/complianceRulesList [get]
func (configManagerApi *ConfigManagerApi) GetComplianceRulesList(c *gin.Context) {
	var pageInfo systemReq.GetComplianceRulesRequest
	err := c.ShouldBindQuery(&pageInfo)
	if err != nil {
		response.FailWithMessage(err.Error(), c)
		return
	}

	configManagerService := service.ServiceGroupApp.SystemServiceGroup.GetConfigManagerService()
	if configManagerService == nil {
		response.FailWithMessage("配置管理服务未初始化", c)
		return
	}

	list, total, err := configManagerService.GetComplianceRulesList(pageInfo.PageInfo)
	if err != nil {
		global.GVA_LOG.Error("获取合规规则列表失败!", zap.Error(err))
		response.FailWithMessage("获取合规规则列表失败: "+err.Error(), c)
		return
	}

	response.OkWithDetailed(response.PageResult{
		List:     list,
		Total:    total,
		Page:     pageInfo.Page,
		PageSize: pageInfo.PageSize,
	}, "获取合规规则列表成功", c)
}

// @Tags ConfigManager
// @Summary 获取单个合规规则
// @Security ApiKeyAuth
// @accept application/json
// @Produce application/json
// @Param rule_id path string true "规则ID"
// @Success 200 {object} response.Response{data=config.ComplianceRule,msg=string} "获取成功"
// @Router /configManager/complianceRule/{rule_id} [get]
func (configManagerApi *ConfigManagerApi) GetComplianceRule(c *gin.Context) {
	ruleID := c.Param("rule_id")
	if ruleID == "" {
		response.FailWithMessage("规则ID不能为空", c)
		return
	}

	configManagerService := service.ServiceGroupApp.SystemServiceGroup.GetConfigManagerService()
	if configManagerService == nil {
		response.FailWithMessage("配置管理服务未初始化", c)
		return
	}

	rule, err := configManagerService.GetComplianceRule(ruleID)
	if err != nil {
		global.GVA_LOG.Error("获取合规规则失败!", zap.Error(err))
		response.FailWithMessage("获取合规规则失败: "+err.Error(), c)
		return
	}

	response.OkWithDetailed(rule, "获取合规规则成功", c)
}

// @Tags ConfigManager
// @Summary 创建合规规则
// @Security ApiKeyAuth
// @accept application/json
// @Produce application/json
// @Param data body systemReq.CreateComplianceRuleRequest true "创建合规规则请求"
// @Success 200 {object} response.Response{msg=string} "创建成功"
// @Router /configManager/complianceRule [post]
func (configManagerApi *ConfigManagerApi) CreateComplianceRule(c *gin.Context) {
	var req systemReq.CreateComplianceRuleRequest
	err := c.ShouldBindJSON(&req)
	if err != nil {
		response.FailWithMessage(err.Error(), c)
		return
	}

	err = utils.Verify(req, utils.CreateComplianceRuleVerify)
	if err != nil {
		response.FailWithMessage(err.Error(), c)
		return
	}

	configManagerService := service.ServiceGroupApp.SystemServiceGroup.GetConfigManagerService()
	if configManagerService == nil {
		response.FailWithMessage("配置管理服务未初始化", c)
		return
	}

	if err := configManagerService.CreateComplianceRule(req); err != nil {
		global.GVA_LOG.Error("创建合规规则失败!", zap.Error(err))
		response.FailWithMessage("创建合规规则失败: "+err.Error(), c)
		return
	}

	response.OkWithMessage("创建合规规则成功", c)
}

// @Tags ConfigManager
// @Summary 更新合规规则
// @Security ApiKeyAuth
// @accept application/json
// @Produce application/json
// @Param rule_id path string true "规则ID"
// @Param data body systemReq.UpdateComplianceRuleRequest true "更新合规规则请求"
// @Success 200 {object} response.Response{msg=string} "更新成功"
// @Router /configManager/complianceRule/{rule_id} [put]
func (configManagerApi *ConfigManagerApi) UpdateComplianceRule(c *gin.Context) {
	ruleID := c.Param("rule_id")
	if ruleID == "" {
		response.FailWithMessage("规则ID不能为空", c)
		return
	}

	var req systemReq.UpdateComplianceRuleRequest
	err := c.ShouldBindJSON(&req)
	if err != nil {
		response.FailWithMessage(err.Error(), c)
		return
	}

	configManagerService := service.ServiceGroupApp.SystemServiceGroup.GetConfigManagerService()
	if configManagerService == nil {
		response.FailWithMessage("配置管理服务未初始化", c)
		return
	}

	if err := configManagerService.UpdateComplianceRule(ruleID, req); err != nil {
		global.GVA_LOG.Error("更新合规规则失败!", zap.Error(err))
		response.FailWithMessage("更新合规规则失败: "+err.Error(), c)
		return
	}

	response.OkWithMessage("更新合规规则成功", c)
}

// @Tags ConfigManager
// @Summary 删除合规规则
// @Security ApiKeyAuth
// @accept application/json
// @Produce application/json
// @Param rule_id path string true "规则ID"
// @Success 200 {object} response.Response{msg=string} "删除成功"
// @Router /configManager/complianceRule/{rule_id} [delete]
func (configManagerApi *ConfigManagerApi) DeleteComplianceRule(c *gin.Context) {
	ruleID := c.Param("rule_id")
	if ruleID == "" {
		response.FailWithMessage("规则ID不能为空", c)
		return
	}

	configManagerService := service.ServiceGroupApp.SystemServiceGroup.GetConfigManagerService()
	if configManagerService == nil {
		response.FailWithMessage("配置管理服务未初始化", c)
		return
	}

	if err := configManagerService.DeleteComplianceRule(ruleID); err != nil {
		global.GVA_LOG.Error("删除合规规则失败!", zap.Error(err))
		response.FailWithMessage("删除合规规则失败: "+err.Error(), c)
		return
	}

	response.OkWithMessage("删除合规规则成功", c)
}

// @Tags ConfigManager
// @Summary 保存合规规则到文件
// @Security ApiKeyAuth
// @accept application/json
// @Produce application/json
// @Success 200 {object} response.Response{msg=string} "保存成功"
// @Router /configManager/saveComplianceRules [post]
func (configManagerApi *ConfigManagerApi) SaveComplianceRules(c *gin.Context) {
	configManagerService := service.ServiceGroupApp.SystemServiceGroup.GetConfigManagerService()
	if configManagerService == nil {
		response.FailWithMessage("配置管理服务未初始化", c)
		return
	}

	if err := configManagerService.SaveComplianceRules(); err != nil {
		global.GVA_LOG.Error("保存合规规则失败!", zap.Error(err))
		response.FailWithMessage("保存合规规则失败: "+err.Error(), c)
		return
	}

	response.OkWithMessage("保存合规规则成功", c)
}

// @Tags ConfigManager
// @Summary 从文件加载合规规则
// @Security ApiKeyAuth
// @accept application/json
// @Produce application/json
// @Success 200 {object} response.Response{msg=string} "加载成功"
// @Router /configManager/loadComplianceRules [post]
func (configManagerApi *ConfigManagerApi) LoadComplianceRules(c *gin.Context) {
	configManagerService := service.ServiceGroupApp.SystemServiceGroup.GetConfigManagerService()
	if configManagerService == nil {
		response.FailWithMessage("配置管理服务未初始化", c)
		return
	}

	if err := configManagerService.LoadComplianceRules(); err != nil {
		global.GVA_LOG.Error("加载合规规则失败!", zap.Error(err))
		response.FailWithMessage("加载合规规则失败: "+err.Error(), c)
		return
	}

	response.OkWithMessage("加载合规规则成功", c)
}

// @Tags ConfigManager
// @Summary 验证配置
// @Security ApiKeyAuth
// @accept application/json
// @Produce application/json
// @Param data body systemReq.ValidateConfigRequest true "验证配置请求"
// @Success 200 {object} response.Response{data=config.ValidationResult,msg=string} "验证完成"
// @Router /configManager/validateConfig [post]
func (configManagerApi *ConfigManagerApi) ValidateConfiguration(c *gin.Context) {
	var req systemReq.ValidateConfigRequest
	err := c.ShouldBindJSON(&req)
	if err != nil {
		response.FailWithMessage(err.Error(), c)
		return
	}

	configManagerService := service.ServiceGroupApp.SystemServiceGroup.GetConfigManagerService()
	if configManagerService == nil {
		response.FailWithMessage("配置管理服务未初始化", c)
		return
	}

	result, err := configManagerService.ValidateConfiguration()
	if err != nil {
		global.GVA_LOG.Error("验证配置失败!", zap.Error(err))
		response.FailWithMessage("验证配置失败: "+err.Error(), c)
		return
	}

	response.OkWithDetailed(result, "配置验证完成", c)
}

// @Tags ConfigManager
// @Summary 获取配置变更历史
// @Security ApiKeyAuth
// @accept application/json
// @Produce application/json
// @Param data query systemReq.ConfigChangeHistoryRequest true "获取配置变更历史"
// @Success 200 {object} response.Response{data=response.PageResult,msg=string} "获取成功"
// @Router /configManager/changeHistory [get]
func (configManagerApi *ConfigManagerApi) GetConfigChangeHistory(c *gin.Context) {
	var pageInfo systemReq.ConfigChangeHistoryRequest
	err := c.ShouldBindQuery(&pageInfo)
	if err != nil {
		response.FailWithMessage(err.Error(), c)
		return
	}

	configManagerService := service.ServiceGroupApp.SystemServiceGroup.GetConfigManagerService()
	if configManagerService == nil {
		response.FailWithMessage("配置管理服务未初始化", c)
		return
	}

	list, total, err := configManagerService.GetConfigChangeHistory(pageInfo)
	if err != nil {
		global.GVA_LOG.Error("获取配置变更历史失败!", zap.Error(err))
		response.FailWithMessage("获取配置变更历史失败: "+err.Error(), c)
		return
	}

	response.OkWithDetailed(response.PageResult{
		List:     list,
		Total:    total,
		Page:     pageInfo.Page,
		PageSize: pageInfo.PageSize,
	}, "获取配置变更历史成功", c)
}

// @Tags ConfigManager
// @Summary 批量更新合规规则
// @Security ApiKeyAuth
// @accept application/json
// @Produce application/json
// @Param data body systemReq.BatchUpdateComplianceRulesRequest true "批量更新合规规则请求"
// @Success 200 {object} response.Response{msg=string} "批量更新成功"
// @Router /configManager/batchUpdateComplianceRules [post]
func (configManagerApi *ConfigManagerApi) BatchUpdateComplianceRules(c *gin.Context) {
	var req systemReq.BatchUpdateComplianceRulesRequest
	err := c.ShouldBindJSON(&req)
	if err != nil {
		response.FailWithMessage(err.Error(), c)
		return
	}

	if len(req.Operations) == 0 {
		response.FailWithMessage("操作列表不能为空", c)
		return
	}

	configManagerService := service.ServiceGroupApp.SystemServiceGroup.GetConfigManagerService()
	if configManagerService == nil {
		response.FailWithMessage("配置管理服务未初始化", c)
		return
	}

	successCount := 0
	failedOperations := make([]string, 0)

	// 执行批量操作
	for i, operation := range req.Operations {
		var operationErr error

		switch operation.Type {
		case "create":
			if operation.Rule != nil {
				operationErr = configManagerService.CreateComplianceRule(*operation.Rule)
			} else {
				operationErr = errors.New("创建操作缺少规则数据")
			}
		case "update":
			if operation.Rule != nil {
				updateReq := systemReq.UpdateComplianceRuleRequest{
					Name:        &operation.Rule.Name,
					Description: &operation.Rule.Description,
					Category:    &operation.Rule.Category,
					Severity:    &operation.Rule.Severity,
					Enabled:     &operation.Rule.Enabled,
					Conditions:  &operation.Rule.Conditions,
					Actions:     &operation.Rule.Actions,
					ValidFrom:   operation.Rule.ValidFrom,
					ValidUntil:  operation.Rule.ValidUntil,
					Metadata:    &operation.Rule.Metadata,
				}
				operationErr = configManagerService.UpdateComplianceRule(operation.RuleID, updateReq)
			} else {
				operationErr = errors.New("更新操作缺少规则数据")
			}
		case "delete":
			operationErr = configManagerService.DeleteComplianceRule(operation.RuleID)
		default:
			operationErr = errors.New("未知的操作类型: " + operation.Type)
		}

		if operationErr != nil {
			failedOperations = append(failedOperations, "操作 "+strconv.Itoa(i+1)+" ("+operation.Type+" "+operation.RuleID+"): "+operationErr.Error())
			global.GVA_LOG.Error("批量操作失败",
				zap.Int("operation_index", i+1),
				zap.String("operation_type", operation.Type),
				zap.String("rule_id", operation.RuleID),
				zap.Error(operationErr),
			)
		} else {
			successCount++
		}
	}

	// 保存更改到文件
	if successCount > 0 {
		if err := configManagerService.SaveComplianceRules(); err != nil {
			global.GVA_LOG.Error("保存合规规则失败!", zap.Error(err))
			response.FailWithMessage("批量操作部分成功，但保存失败: "+err.Error(), c)
			return
		}
	}

	message := "批量操作完成：成功 " + strconv.Itoa(successCount) + " 个，失败 " + strconv.Itoa(len(failedOperations)) + " 个"
	if len(failedOperations) > 0 {
		response.OkWithDetailed(map[string]interface{}{
			"success_count":     successCount,
			"failed_count":      len(failedOperations),
			"failed_operations": failedOperations,
		}, message, c)
	} else {
		response.OkWithMessage(message, c)
	}
}
