package nfc_relay_admin

import (
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/flipped-aurora/gin-vue-admin/server/global"
	"github.com/flipped-aurora/gin-vue-admin/server/model/common/response"
	"github.com/flipped-aurora/gin-vue-admin/server/nfc_relay/security"
	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

type ComplianceRulesApi struct{}

// ComplianceRuleResponse 合规规则响应
type ComplianceRuleResponse struct {
	ID          string    `json:"id"`
	Name        string    `json:"name"`
	Description string    `json:"description"`
	Pattern     string    `json:"pattern"`
	RiskLevel   string    `json:"riskLevel"`
	Action      string    `json:"action"`
	Enabled     bool      `json:"enabled"`
	CreatedAt   time.Time `json:"createdAt"`
	UpdatedAt   time.Time `json:"updatedAt"`
	CreatedBy   string    `json:"createdBy"`
	UpdatedBy   string    `json:"updatedBy"`
}

// ComplianceRuleRequest 合规规则请求
type ComplianceRuleRequest struct {
	Name        string `json:"name" binding:"required"`
	Description string `json:"description"`
	Pattern     string `json:"pattern"`
	RiskLevel   string `json:"riskLevel" binding:"required,oneof=LOW MEDIUM HIGH CRITICAL"`
	Action      string `json:"action" binding:"required,oneof=BLOCK WARN LOG"`
	Enabled     *bool  `json:"enabled"`
}

// ComplianceRulesListResponse 合规规则列表响应
type ComplianceRulesListResponse struct {
	Rules []ComplianceRuleResponse `json:"rules"`
	Total int64                    `json:"total"`
	Page  int                      `json:"page"`
	Size  int                      `json:"size"`
}

// RuleFileInfo 规则文件信息
type RuleFileInfo struct {
	Filename    string    `json:"filename"`
	Size        int64     `json:"size"`
	ModTime     time.Time `json:"modTime"`
	RuleCount   int       `json:"ruleCount"`
	IsActive    bool      `json:"isActive"`
	Version     string    `json:"version"`
	Description string    `json:"description"`
}

// RuleTestRequest 规则测试请求
type RuleTestRequest struct {
	RuleID      string                 `json:"ruleId" binding:"required"`
	TestData    security.APDUDataClass `json:"testData" binding:"required"`
	TestContext map[string]interface{} `json:"testContext"`
}

// RuleTestResponse 规则测试响应
type RuleTestResponse struct {
	RuleID   string                     `json:"ruleId"`
	Matched  bool                       `json:"matched"`
	Result   *security.ComplianceResult `json:"result"`
	TestTime int64                      `json:"testTime"` // 微秒
	Details  string                     `json:"details"`
}

var complianceEngine *security.ComplianceAuditEngine

func init() {
	complianceEngine = security.NewComplianceAuditEngine()
}

// GetComplianceRules 获取所有合规规则
// @Summary 获取所有合规规则
// @Description 获取系统中的所有合规规则
// @Tags NFC中继管理
// @Accept json
// @Produce json
// @Param page query int false "页码"
// @Param size query int false "每页大小"
// @Param enabled query bool false "是否启用"
// @Param riskLevel query string false "风险级别"
// @Success 200 {object} response.Response{data=ComplianceRulesListResponse}
// @Router /api/admin/nfc-relay/v1/compliance/rules [get]
func (c *ComplianceRulesApi) GetComplianceRules(ctx *gin.Context) {
	page, _ := strconv.Atoi(ctx.DefaultQuery("page", "1"))
	size, _ := strconv.Atoi(ctx.DefaultQuery("size", "20"))
	enabledStr := ctx.Query("enabled")
	riskLevel := ctx.Query("riskLevel")

	operatorUserID := ctx.GetString("userID")

	global.GVA_LOG.Info("获取合规规则列表",
		zap.String("operatorUserID", operatorUserID),
		zap.Int("page", page),
		zap.Int("size", size),
		zap.String("enabled", enabledStr),
		zap.String("riskLevel", riskLevel),
	)

	// 模拟获取规则数据（实际应该从数据库或配置文件获取）
	rules := getComplianceRulesFromEngine()

	// 过滤
	filteredRules := filterRules(rules, enabledStr, riskLevel)

	// 分页
	total := int64(len(filteredRules))
	start := (page - 1) * size
	end := start + size
	if end > len(filteredRules) {
		end = len(filteredRules)
	}
	if start > len(filteredRules) {
		start = len(filteredRules)
	}

	pagedRules := filteredRules[start:end]

	resp := ComplianceRulesListResponse{
		Rules: pagedRules,
		Total: total,
		Page:  page,
		Size:  size,
	}

	// 记录操作
	global.LogAuditEvent("compliance_rules_query", map[string]interface{}{
		"operator_user_id": operatorUserID,
		"filter_enabled":   enabledStr,
		"filter_risk":      riskLevel,
		"result_count":     len(pagedRules),
	})

	response.OkWithDetailed(resp, "获取合规规则成功", ctx)
}

// GetComplianceRule 获取单个合规规则
// @Summary 获取单个合规规则
// @Description 根据ID获取特定的合规规则
// @Tags NFC中继管理
// @Accept json
// @Produce json
// @Param rule_id path string true "规则ID"
// @Success 200 {object} response.Response{data=ComplianceRuleResponse}
// @Router /api/admin/nfc-relay/v1/compliance/rules/{rule_id} [get]
func (c *ComplianceRulesApi) GetComplianceRule(ctx *gin.Context) {
	ruleID := ctx.Param("rule_id")
	if ruleID == "" {
		response.FailWithMessage("规则ID不能为空", ctx)
		return
	}

	operatorUserID := ctx.GetString("userID")

	// 查找规则
	rules := getComplianceRulesFromEngine()
	var foundRule *ComplianceRuleResponse
	for _, rule := range rules {
		if rule.ID == ruleID {
			foundRule = &rule
			break
		}
	}

	if foundRule == nil {
		response.FailWithMessage("规则不存在", ctx)
		return
	}

	// 记录操作
	global.LogAuditEvent("compliance_rule_query", map[string]interface{}{
		"operator_user_id": operatorUserID,
		"rule_id":          ruleID,
		"rule_name":        foundRule.Name,
	})

	response.OkWithDetailed(*foundRule, "获取合规规则成功", ctx)
}

// CreateComplianceRule 创建合规规则
// @Summary 创建合规规则
// @Description 创建新的合规规则
// @Tags NFC中继管理
// @Accept json
// @Produce json
// @Param data body ComplianceRuleRequest true "合规规则信息"
// @Success 200 {object} response.Response
// @Router /api/admin/nfc-relay/v1/compliance/rules [post]
func (c *ComplianceRulesApi) CreateComplianceRule(ctx *gin.Context) {
	var req ComplianceRuleRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		response.FailWithMessage("请求参数格式错误: "+err.Error(), ctx)
		return
	}

	operatorUserID := ctx.GetString("userID")
	if operatorUserID == "" {
		operatorUserID = "unknown"
	}

	// 验证规则
	if err := validateComplianceRule(&req); err != nil {
		response.FailWithMessage("规则验证失败: "+err.Error(), ctx)
		return
	}

	// 生成规则ID
	ruleID := generateRuleID(req.Name)

	// 创建规则（实际应该保存到数据库或配置文件）
	newRule := ComplianceRuleResponse{
		ID:          ruleID,
		Name:        req.Name,
		Description: req.Description,
		Pattern:     req.Pattern,
		RiskLevel:   req.RiskLevel,
		Action:      req.Action,
		Enabled:     true,
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
		CreatedBy:   operatorUserID,
		UpdatedBy:   operatorUserID,
	}

	if req.Enabled != nil {
		newRule.Enabled = *req.Enabled
	}

	// 记录操作
	global.LogAuditEvent("compliance_rule_created", map[string]interface{}{
		"operator_user_id": operatorUserID,
		"rule_id":          ruleID,
		"rule_name":        req.Name,
		"risk_level":       req.RiskLevel,
		"action":           req.Action,
		"enabled":          newRule.Enabled,
	})

	global.GVA_LOG.Info("创建合规规则成功",
		zap.String("ruleID", ruleID),
		zap.String("ruleName", req.Name),
		zap.String("operatorUserID", operatorUserID),
	)

	response.OkWithDetailed(newRule, "创建合规规则成功", ctx)
}

// UpdateComplianceRule 更新合规规则
// @Summary 更新合规规则
// @Description 更新现有的合规规则
// @Tags NFC中继管理
// @Accept json
// @Produce json
// @Param rule_id path string true "规则ID"
// @Param data body ComplianceRuleRequest true "合规规则信息"
// @Success 200 {object} response.Response
// @Router /api/admin/nfc-relay/v1/compliance/rules/{rule_id} [put]
func (c *ComplianceRulesApi) UpdateComplianceRule(ctx *gin.Context) {
	ruleID := ctx.Param("rule_id")
	if ruleID == "" {
		response.FailWithMessage("规则ID不能为空", ctx)
		return
	}

	var req ComplianceRuleRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		response.FailWithMessage("请求参数格式错误: "+err.Error(), ctx)
		return
	}

	operatorUserID := ctx.GetString("userID")
	if operatorUserID == "" {
		operatorUserID = "unknown"
	}

	// 验证规则存在
	rules := getComplianceRulesFromEngine()
	var foundRule *ComplianceRuleResponse
	for i, rule := range rules {
		if rule.ID == ruleID {
			foundRule = &rules[i]
			break
		}
	}

	if foundRule == nil {
		response.FailWithMessage("规则不存在", ctx)
		return
	}

	// 验证更新内容
	if err := validateComplianceRule(&req); err != nil {
		response.FailWithMessage("规则验证失败: "+err.Error(), ctx)
		return
	}

	// 更新规则
	oldData := map[string]interface{}{
		"name":       foundRule.Name,
		"risk_level": foundRule.RiskLevel,
		"action":     foundRule.Action,
		"enabled":    foundRule.Enabled,
	}

	foundRule.Name = req.Name
	foundRule.Description = req.Description
	foundRule.Pattern = req.Pattern
	foundRule.RiskLevel = req.RiskLevel
	foundRule.Action = req.Action
	foundRule.UpdatedAt = time.Now()
	foundRule.UpdatedBy = operatorUserID

	if req.Enabled != nil {
		foundRule.Enabled = *req.Enabled
	}

	// 记录操作
	global.LogAuditEvent("compliance_rule_updated", map[string]interface{}{
		"operator_user_id": operatorUserID,
		"rule_id":          ruleID,
		"old_data":         oldData,
		"new_data": map[string]interface{}{
			"name":       req.Name,
			"risk_level": req.RiskLevel,
			"action":     req.Action,
			"enabled":    foundRule.Enabled,
		},
	})

	response.OkWithDetailed(*foundRule, "更新合规规则成功", ctx)
}

// DeleteComplianceRule 删除合规规则
// @Summary 删除合规规则
// @Description 删除指定的合规规则
// @Tags NFC中继管理
// @Accept json
// @Produce json
// @Param rule_id path string true "规则ID"
// @Success 200 {object} response.Response
// @Router /api/admin/nfc-relay/v1/compliance/rules/{rule_id} [delete]
func (c *ComplianceRulesApi) DeleteComplianceRule(ctx *gin.Context) {
	ruleID := ctx.Param("rule_id")
	if ruleID == "" {
		response.FailWithMessage("规则ID不能为空", ctx)
		return
	}

	operatorUserID := ctx.GetString("userID")

	// 检查规则是否存在
	rules := getComplianceRulesFromEngine()
	var foundRule *ComplianceRuleResponse
	for _, rule := range rules {
		if rule.ID == ruleID {
			foundRule = &rule
			break
		}
	}

	if foundRule == nil {
		response.FailWithMessage("规则不存在", ctx)
		return
	}

	// 检查是否为系统内置规则
	if isSystemRule(ruleID) {
		response.FailWithMessage("不能删除系统内置规则", ctx)
		return
	}

	// 记录操作
	global.LogAuditEvent("compliance_rule_deleted", map[string]interface{}{
		"operator_user_id": operatorUserID,
		"rule_id":          ruleID,
		"rule_name":        foundRule.Name,
		"risk_level":       foundRule.RiskLevel,
	})

	response.OkWithMessage("删除合规规则成功", ctx)
}

// TestComplianceRule 测试合规规则
// @Summary 测试合规规则
// @Description 使用测试数据验证合规规则的效果
// @Tags NFC中继管理
// @Accept json
// @Produce json
// @Param data body RuleTestRequest true "规则测试请求"
// @Success 200 {object} response.Response{data=RuleTestResponse}
// @Router /api/admin/nfc-relay/v1/compliance/rules/test [post]
func (c *ComplianceRulesApi) TestComplianceRule(ctx *gin.Context) {
	var req RuleTestRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		response.FailWithMessage("请求参数格式错误: "+err.Error(), ctx)
		return
	}

	operatorUserID := ctx.GetString("userID")
	startTime := getCurrentMicroseconds()

	// 使用合规引擎测试规则
	auditResult, err := complianceEngine.AuditAPDUData(&req.TestData)
	if err != nil {
		global.GVA_LOG.Error("测试合规规则失败", zap.Error(err))
		response.FailWithMessage("测试规则失败: "+err.Error(), ctx)
		return
	}

	testTime := getCurrentMicroseconds() - startTime

	resp := RuleTestResponse{
		RuleID:   req.RuleID,
		Matched:  !auditResult.Compliant,
		Result:   auditResult,
		TestTime: testTime,
		Details:  fmt.Sprintf("规则测试完成，合规状态: %s", auditResult.RiskLevel),
	}

	// 记录测试操作
	global.LogAuditEvent("compliance_rule_tested", map[string]interface{}{
		"operator_user_id": operatorUserID,
		"rule_id":          req.RuleID,
		"test_result":      auditResult.Compliant,
		"risk_level":       auditResult.RiskLevel,
		"test_time":        testTime,
	})

	response.OkWithDetailed(resp, "规则测试完成", ctx)
}

// GetRuleFiles 获取规则文件列表
// @Summary 获取规则文件列表
// @Description 获取所有合规规则文件的信息
// @Tags NFC中继管理
// @Accept json
// @Produce json
// @Success 200 {object} response.Response{data=[]RuleFileInfo}
// @Router /api/admin/nfc-relay/v1/compliance/rule-files [get]
func (c *ComplianceRulesApi) GetRuleFiles(ctx *gin.Context) {
	operatorUserID := ctx.GetString("userID")

	// 模拟规则文件信息
	ruleFiles := []RuleFileInfo{
		{
			Filename:    "default_rules.json",
			Size:        15420,
			ModTime:     time.Now().Add(-24 * time.Hour),
			RuleCount:   25,
			IsActive:    true,
			Version:     "1.2.0",
			Description: "默认合规规则集",
		},
		{
			Filename:    "custom_rules.json",
			Size:        8750,
			ModTime:     time.Now().Add(-2 * time.Hour),
			RuleCount:   12,
			IsActive:    false,
			Version:     "1.0.1",
			Description: "自定义规则集",
		},
		{
			Filename:    "strict_rules.json",
			Size:        22100,
			ModTime:     time.Now().Add(-48 * time.Hour),
			RuleCount:   35,
			IsActive:    false,
			Version:     "2.0.0",
			Description: "严格模式规则集",
		},
	}

	// 记录操作
	global.LogAuditEvent("rule_files_query", map[string]interface{}{
		"operator_user_id": operatorUserID,
		"file_count":       len(ruleFiles),
	})

	response.OkWithDetailed(ruleFiles, "获取规则文件列表成功", ctx)
}

// ImportRuleFile 导入规则文件
// @Summary 导入规则文件
// @Description 从文件导入合规规则
// @Tags NFC中继管理
// @Accept multipart/form-data
// @Produce json
// @Param file formData file true "规则文件"
// @Param replace formData bool false "是否替换现有规则"
// @Success 200 {object} response.Response
// @Router /api/admin/nfc-relay/v1/compliance/rule-files/import [post]
func (c *ComplianceRulesApi) ImportRuleFile(ctx *gin.Context) {
	operatorUserID := ctx.GetString("userID")

	file, header, err := ctx.Request.FormFile("file")
	if err != nil {
		response.FailWithMessage("获取上传文件失败: "+err.Error(), ctx)
		return
	}
	defer file.Close()

	// 验证文件类型
	if !strings.HasSuffix(header.Filename, ".json") {
		response.FailWithMessage("只支持JSON格式的规则文件", ctx)
		return
	}

	// 验证文件大小
	if header.Size > 10*1024*1024 { // 10MB限制
		response.FailWithMessage("文件大小不能超过10MB", ctx)
		return
	}

	replaceStr := ctx.PostForm("replace")
	replace := replaceStr == "true"

	// 这里应该实际处理文件导入逻辑
	// 简化实现，只记录操作
	global.LogAuditEvent("rule_file_imported", map[string]interface{}{
		"operator_user_id": operatorUserID,
		"filename":         header.Filename,
		"file_size":        header.Size,
		"replace_existing": replace,
	})

	global.GVA_LOG.Info("导入规则文件",
		zap.String("filename", header.Filename),
		zap.Int64("size", header.Size),
		zap.Bool("replace", replace),
		zap.String("operatorUserID", operatorUserID),
	)

	response.OkWithMessage("导入规则文件成功", ctx)
}

// ExportRuleFile 导出规则文件
// @Summary 导出规则文件
// @Description 导出当前的合规规则到文件
// @Tags NFC中继管理
// @Accept json
// @Produce application/octet-stream
// @Param format query string false "导出格式" Enums(json,yaml)
// @Success 200 {file} file
// @Router /api/admin/nfc-relay/v1/compliance/rule-files/export [get]
func (c *ComplianceRulesApi) ExportRuleFile(ctx *gin.Context) {
	operatorUserID := ctx.GetString("userID")
	format := ctx.DefaultQuery("format", "json")

	if format != "json" && format != "yaml" {
		response.FailWithMessage("不支持的导出格式，仅支持json和yaml", ctx)
		return
	}

	// 获取所有规则
	rules := getComplianceRulesFromEngine()

	// 生成文件名
	filename := fmt.Sprintf("compliance_rules_%s.%s", time.Now().Format("20060102_150405"), format)

	// 设置响应头
	ctx.Header("Content-Disposition", fmt.Sprintf("attachment; filename=%s", filename))
	ctx.Header("Content-Type", "application/octet-stream")

	// 模拟文件内容
	content := fmt.Sprintf("// 合规规则导出文件\n// 导出时间: %s\n// 规则数量: %d\n// 导出操作员: %s\n",
		time.Now().Format("2006-01-02 15:04:05"), len(rules), operatorUserID)

	// 记录操作
	global.LogAuditEvent("rule_file_exported", map[string]interface{}{
		"operator_user_id": operatorUserID,
		"format":           format,
		"rule_count":       len(rules),
		"filename":         filename,
	})

	ctx.String(200, content)
}

// 辅助函数
func getComplianceRulesFromEngine() []ComplianceRuleResponse {
	// 从合规引擎获取规则（模拟实现）
	now := time.Now()
	return []ComplianceRuleResponse{
		{
			ID:          "HIGH_RISK_COMMAND",
			Name:        "高风险命令检测",
			Description: "检测可能的高风险APDU命令",
			Pattern:     "WRITE|INTERNAL_AUTHENTICATE",
			RiskLevel:   "HIGH",
			Action:      "WARN",
			Enabled:     true,
			CreatedAt:   now.Add(-30 * 24 * time.Hour),
			UpdatedAt:   now.Add(-24 * time.Hour),
			CreatedBy:   "system",
			UpdatedBy:   "admin",
		},
		{
			ID:          "TRANSACTION_AMOUNT_LIMIT",
			Name:        "交易金额限制",
			Description: "检查交易金额是否超过限制",
			Pattern:     "amount_check",
			RiskLevel:   "MEDIUM",
			Action:      "BLOCK",
			Enabled:     true,
			CreatedAt:   now.Add(-15 * 24 * time.Hour),
			UpdatedAt:   now.Add(-1 * time.Hour),
			CreatedBy:   "system",
			UpdatedBy:   "admin",
		},
		{
			ID:          "FREQUENCY_LIMIT",
			Name:        "频率限制",
			Description: "检查操作频率是否过高",
			Pattern:     "frequency_check",
			RiskLevel:   "LOW",
			Action:      "LOG",
			Enabled:     true,
			CreatedAt:   now.Add(-7 * 24 * time.Hour),
			UpdatedAt:   now.Add(-12 * time.Hour),
			CreatedBy:   "admin",
			UpdatedBy:   "admin",
		},
	}
}

func filterRules(rules []ComplianceRuleResponse, enabledStr, riskLevel string) []ComplianceRuleResponse {
	var filtered []ComplianceRuleResponse
	for _, rule := range rules {
		// 过滤启用状态
		if enabledStr != "" {
			enabled := enabledStr == "true"
			if rule.Enabled != enabled {
				continue
			}
		}

		// 过滤风险级别
		if riskLevel != "" && rule.RiskLevel != riskLevel {
			continue
		}

		filtered = append(filtered, rule)
	}
	return filtered
}

func validateComplianceRule(req *ComplianceRuleRequest) error {
	if req.Name == "" {
		return fmt.Errorf("规则名称不能为空")
	}

	if len(req.Name) > 100 {
		return fmt.Errorf("规则名称长度不能超过100字符")
	}

	if req.RiskLevel != "LOW" && req.RiskLevel != "MEDIUM" && req.RiskLevel != "HIGH" && req.RiskLevel != "CRITICAL" {
		return fmt.Errorf("无效的风险级别")
	}

	if req.Action != "BLOCK" && req.Action != "WARN" && req.Action != "LOG" {
		return fmt.Errorf("无效的处理动作")
	}

	return nil
}

func generateRuleID(name string) string {
	// 生成规则ID
	cleaned := strings.ReplaceAll(strings.ToUpper(name), " ", "_")
	return fmt.Sprintf("RULE_%s_%d", cleaned, time.Now().Unix())
}

func isSystemRule(ruleID string) bool {
	systemRules := []string{
		"HIGH_RISK_COMMAND",
		"TRANSACTION_AMOUNT_LIMIT",
		"TIME_RESTRICTION",
		"FREQUENCY_LIMIT",
		"SUSPICIOUS_PATTERN",
	}

	for _, sysRule := range systemRules {
		if sysRule == ruleID {
			return true
		}
	}
	return false
}
