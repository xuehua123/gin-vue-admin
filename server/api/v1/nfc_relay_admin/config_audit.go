package nfc_relay_admin

import (
	"strconv"
	"time"

	"github.com/flipped-aurora/gin-vue-admin/server/global"
	"github.com/flipped-aurora/gin-vue-admin/server/model/common/response"
	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

type ConfigAuditApi struct{}

// ConfigChangeRecord 配置变更记录
type ConfigChangeRecord struct {
	ID            int64                  `json:"id"`
	ConfigType    string                 `json:"configType"`
	ChangeType    string                 `json:"changeType"` // CREATE, UPDATE, DELETE, RELOAD
	OperatorID    string                 `json:"operatorId"`
	OperatorName  string                 `json:"operatorName"`
	Timestamp     time.Time              `json:"timestamp"`
	OldValue      map[string]interface{} `json:"oldValue,omitempty"`
	NewValue      map[string]interface{} `json:"newValue,omitempty"`
	ChangedFields []string               `json:"changedFields"`
	Reason        string                 `json:"reason"`
	IPAddress     string                 `json:"ipAddress"`
	UserAgent     string                 `json:"userAgent"`
	SessionID     string                 `json:"sessionId"`
	Status        string                 `json:"status"` // SUCCESS, FAILED, PENDING
	ErrorMessage  string                 `json:"errorMessage,omitempty"`
	RollbackID    *int64                 `json:"rollbackId,omitempty"`
}

// ConfigAuditQuery 配置审计查询
type ConfigAuditQuery struct {
	ConfigType string    `json:"configType,omitempty"`
	ChangeType string    `json:"changeType,omitempty"`
	OperatorID string    `json:"operatorId,omitempty"`
	StartTime  time.Time `json:"startTime,omitempty"`
	EndTime    time.Time `json:"endTime,omitempty"`
	Status     string    `json:"status,omitempty"`
	Page       int       `json:"page"`
	PageSize   int       `json:"pageSize"`
}

// ConfigAuditResponse 配置审计响应
type ConfigAuditResponse struct {
	Records []ConfigChangeRecord `json:"records"`
	Total   int64                `json:"total"`
	Page    int                  `json:"page"`
	Size    int                  `json:"size"`
}

// ConfigAuditStats 配置审计统计
type ConfigAuditStats struct {
	TotalChanges       int64                    `json:"totalChanges"`
	ChangesToday       int64                    `json:"changesToday"`
	ChangesThisWeek    int64                    `json:"changesThisWeek"`
	ChangesThisMonth   int64                    `json:"changesThisMonth"`
	ChangesByType      map[string]int64         `json:"changesByType"`
	ChangesByOperator  map[string]int64         `json:"changesByOperator"`
	ChangesByConfig    map[string]int64         `json:"changesByConfig"`
	SuccessRate        float64                  `json:"successRate"`
	RecentOperators    []OperatorActivity       `json:"recentOperators"`
	ConfigChangesTrend []ConfigChangeTrendPoint `json:"configChangesTrend"`
}

// OperatorActivity 操作员活动
type OperatorActivity struct {
	OperatorID   string    `json:"operatorId"`
	OperatorName string    `json:"operatorName"`
	ChangeCount  int64     `json:"changeCount"`
	LastActivity time.Time `json:"lastActivity"`
	SuccessRate  float64   `json:"successRate"`
}

// ConfigChangeTrendPoint 配置变更趋势点
type ConfigChangeTrendPoint struct {
	Date         time.Time `json:"date"`
	ChangeCount  int64     `json:"changeCount"`
	SuccessCount int64     `json:"successCount"`
	FailedCount  int64     `json:"failedCount"`
}

// GetConfigAuditLogs 获取配置审计日志
// @Summary 获取配置审计日志
// @Description 查询配置变更的审计日志记录
// @Tags NFC中继管理
// @Accept json
// @Produce json
// @Param configType query string false "配置类型"
// @Param changeType query string false "变更类型"
// @Param operatorId query string false "操作员ID"
// @Param startTime query string false "开始时间"
// @Param endTime query string false "结束时间"
// @Param status query string false "状态"
// @Param page query int false "页码"
// @Param pageSize query int false "每页大小"
// @Success 200 {object} response.Response{data=ConfigAuditResponse}
// @Router /api/admin/nfc-relay/v1/config-audit/logs [get]
func (c *ConfigAuditApi) GetConfigAuditLogs(ctx *gin.Context) {
	var query ConfigAuditQuery

	// 解析查询参数
	query.ConfigType = ctx.Query("configType")
	query.ChangeType = ctx.Query("changeType")
	query.OperatorID = ctx.Query("operatorId")
	query.Status = ctx.Query("status")

	// 解析时间参数
	if startTimeStr := ctx.Query("startTime"); startTimeStr != "" {
		if startTime, err := time.Parse("2006-01-02T15:04:05Z", startTimeStr); err == nil {
			query.StartTime = startTime
		}
	}
	if endTimeStr := ctx.Query("endTime"); endTimeStr != "" {
		if endTime, err := time.Parse("2006-01-02T15:04:05Z", endTimeStr); err == nil {
			query.EndTime = endTime
		}
	}

	// 解析分页参数
	page, _ := strconv.Atoi(ctx.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(ctx.DefaultQuery("pageSize", "20"))
	query.Page = page
	query.PageSize = pageSize

	operatorUserID := ctx.GetString("userID")

	global.GVA_LOG.Info("查询配置审计日志",
		zap.String("operatorUserID", operatorUserID),
		zap.String("configType", query.ConfigType),
		zap.String("changeType", query.ChangeType),
		zap.Int("page", page),
		zap.Int("pageSize", pageSize),
	)

	// 模拟查询配置审计记录
	records := getMockConfigAuditRecords(query)

	// 模拟分页
	total := int64(len(records))
	start := (query.Page - 1) * query.PageSize
	end := start + query.PageSize
	if end > len(records) {
		end = len(records)
	}
	if start > len(records) {
		start = len(records)
	}

	pagedRecords := records[start:end]

	resp := ConfigAuditResponse{
		Records: pagedRecords,
		Total:   total,
		Page:    query.Page,
		Size:    query.PageSize,
	}

	// 记录查询操作
	global.LogAuditEvent("config_audit_logs_query", map[string]interface{}{
		"operator_user_id": operatorUserID,
		"query_params":     query,
		"result_count":     len(pagedRecords),
	})

	response.OkWithDetailed(resp, "获取配置审计日志成功", ctx)
}

// GetConfigAuditStats 获取配置审计统计
// @Summary 获取配置审计统计
// @Description 获取配置变更的统计信息和趋势
// @Tags NFC中继管理
// @Accept json
// @Produce json
// @Param days query int false "统计天数"
// @Success 200 {object} response.Response{data=ConfigAuditStats}
// @Router /api/admin/nfc-relay/v1/config-audit/stats [get]
func (c *ConfigAuditApi) GetConfigAuditStats(ctx *gin.Context) {
	daysStr := ctx.DefaultQuery("days", "30")
	days, _ := strconv.Atoi(daysStr)
	if days <= 0 || days > 365 {
		days = 30
	}

	operatorUserID := ctx.GetString("userID")

	// 模拟统计数据
	stats := ConfigAuditStats{
		TotalChanges:     1245,
		ChangesToday:     23,
		ChangesThisWeek:  156,
		ChangesThisMonth: 623,
		ChangesByType: map[string]int64{
			"UPDATE": 756,
			"RELOAD": 324,
			"CREATE": 123,
			"DELETE": 42,
		},
		ChangesByOperator: map[string]int64{
			"admin_001": 456,
			"admin_002": 234,
			"system":    555,
		},
		ChangesByConfig: map[string]int64{
			"main":       234,
			"security":   345,
			"nfc_relay":  456,
			"compliance": 210,
		},
		SuccessRate: 96.8,
		RecentOperators: []OperatorActivity{
			{
				OperatorID:   "admin_001",
				OperatorName: "系统管理员",
				ChangeCount:  23,
				LastActivity: time.Now().Add(-2 * time.Hour),
				SuccessRate:  98.5,
			},
			{
				OperatorID:   "admin_002",
				OperatorName: "安全管理员",
				ChangeCount:  15,
				LastActivity: time.Now().Add(-5 * time.Hour),
				SuccessRate:  95.2,
			},
		},
		ConfigChangesTrend: generateTrendData(days),
	}

	// 记录统计查询
	global.LogAuditEvent("config_audit_stats_query", map[string]interface{}{
		"operator_user_id": operatorUserID,
		"stats_days":       days,
		"total_changes":    stats.TotalChanges,
	})

	response.OkWithDetailed(stats, "获取配置审计统计成功", ctx)
}

// GetConfigChangeDetail 获取配置变更详情
// @Summary 获取配置变更详情
// @Description 获取指定配置变更记录的详细信息
// @Tags NFC中继管理
// @Accept json
// @Produce json
// @Param change_id path int true "变更记录ID"
// @Success 200 {object} response.Response{data=ConfigChangeRecord}
// @Router /api/admin/nfc-relay/v1/config-audit/changes/{change_id} [get]
func (c *ConfigAuditApi) GetConfigChangeDetail(ctx *gin.Context) {
	changeIDStr := ctx.Param("change_id")
	changeID, err := strconv.ParseInt(changeIDStr, 10, 64)
	if err != nil {
		response.FailWithMessage("无效的变更记录ID", ctx)
		return
	}

	operatorUserID := ctx.GetString("userID")

	// 模拟获取变更详情
	record := ConfigChangeRecord{
		ID:           changeID,
		ConfigType:   "security",
		ChangeType:   "UPDATE",
		OperatorID:   "admin_001",
		OperatorName: "系统管理员",
		Timestamp:    time.Now().Add(-2 * time.Hour),
		OldValue: map[string]interface{}{
			"maxTransactionAmount": 50000,
			"enableDeepInspection": false,
		},
		NewValue: map[string]interface{}{
			"maxTransactionAmount": 100000,
			"enableDeepInspection": true,
		},
		ChangedFields: []string{"maxTransactionAmount", "enableDeepInspection"},
		Reason:        "提高交易限额，启用深度检查",
		IPAddress:     "192.168.1.100",
		UserAgent:     "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36",
		SessionID:     "sess_" + changeIDStr,
		Status:        "SUCCESS",
	}

	// 记录详情查询
	global.LogAuditEvent("config_change_detail_query", map[string]interface{}{
		"operator_user_id": operatorUserID,
		"change_id":        changeID,
		"config_type":      record.ConfigType,
	})

	response.OkWithDetailed(record, "获取配置变更详情成功", ctx)
}

// CreateConfigAuditRecord 创建配置审计记录
// @Summary 创建配置审计记录
// @Description 手动创建配置变更审计记录
// @Tags NFC中继管理
// @Accept json
// @Produce json
// @Param data body ConfigChangeRecord true "配置变更记录"
// @Success 200 {object} response.Response
// @Router /api/admin/nfc-relay/v1/config-audit/records [post]
func (c *ConfigAuditApi) CreateConfigAuditRecord(ctx *gin.Context) {
	var record ConfigChangeRecord
	if err := ctx.ShouldBindJSON(&record); err != nil {
		response.FailWithMessage("请求参数格式错误: "+err.Error(), ctx)
		return
	}

	operatorUserID := ctx.GetString("userID")

	// 设置记录信息
	record.ID = time.Now().Unix()
	record.Timestamp = time.Now()
	record.OperatorID = operatorUserID
	record.IPAddress = ctx.ClientIP()
	record.UserAgent = ctx.GetHeader("User-Agent")
	record.SessionID = ctx.GetString("sessionID")
	record.Status = "SUCCESS"

	// 这里应该实际保存到数据库
	global.GVA_LOG.Info("创建配置审计记录",
		zap.Int64("recordID", record.ID),
		zap.String("configType", record.ConfigType),
		zap.String("changeType", record.ChangeType),
		zap.String("operatorUserID", operatorUserID),
	)

	// 记录操作
	global.LogAuditEvent("config_audit_record_created", map[string]interface{}{
		"operator_user_id": operatorUserID,
		"record_id":        record.ID,
		"config_type":      record.ConfigType,
		"change_type":      record.ChangeType,
	})

	response.OkWithDetailed(record, "创建配置审计记录成功", ctx)
}

// ExportConfigAuditLogs 导出配置审计日志
// @Summary 导出配置审计日志
// @Description 导出配置审计日志到文件
// @Tags NFC中继管理
// @Accept json
// @Produce application/octet-stream
// @Param format query string false "导出格式" Enums(csv,xlsx,json)
// @Param configType query string false "配置类型"
// @Param startTime query string false "开始时间"
// @Param endTime query string false "结束时间"
// @Success 200 {file} file
// @Router /api/admin/nfc-relay/v1/config-audit/export [get]
func (c *ConfigAuditApi) ExportConfigAuditLogs(ctx *gin.Context) {
	format := ctx.DefaultQuery("format", "csv")
	configType := ctx.Query("configType")
	startTime := ctx.Query("startTime")
	endTime := ctx.Query("endTime")

	operatorUserID := ctx.GetString("userID")

	if format != "csv" && format != "xlsx" && format != "json" {
		response.FailWithMessage("不支持的导出格式", ctx)
		return
	}

	// 构建查询条件
	query := ConfigAuditQuery{
		ConfigType: configType,
		Page:       1,
		PageSize:   10000, // 导出时使用大页面
	}

	if startTime != "" {
		if st, err := time.Parse("2006-01-02T15:04:05Z", startTime); err == nil {
			query.StartTime = st
		}
	}
	if endTime != "" {
		if et, err := time.Parse("2006-01-02T15:04:05Z", endTime); err == nil {
			query.EndTime = et
		}
	}

	// 获取数据
	records := getMockConfigAuditRecords(query)

	// 生成文件名
	filename := "config_audit_logs_" + time.Now().Format("20060102_150405") + "." + format

	// 设置响应头
	ctx.Header("Content-Disposition", "attachment; filename="+filename)
	ctx.Header("Content-Type", "application/octet-stream")

	// 模拟文件内容
	content := "Configuration Audit Export\n" +
		"Export Time: " + time.Now().Format("2006-01-02 15:04:05") + "\n" +
		"Record Count: " + strconv.Itoa(len(records)) + "\n" +
		"Operator: " + operatorUserID + "\n"

	// 记录导出操作
	global.LogAuditEvent("config_audit_exported", map[string]interface{}{
		"operator_user_id": operatorUserID,
		"format":           format,
		"config_type":      configType,
		"record_count":     len(records),
		"filename":         filename,
	})

	ctx.String(200, content)
}

// 辅助函数
func getMockConfigAuditRecords(query ConfigAuditQuery) []ConfigChangeRecord {
	// 模拟配置审计记录
	now := time.Now()
	return []ConfigChangeRecord{
		{
			ID:           1001,
			ConfigType:   "security",
			ChangeType:   "UPDATE",
			OperatorID:   "admin_001",
			OperatorName: "系统管理员",
			Timestamp:    now.Add(-2 * time.Hour),
			OldValue: map[string]interface{}{
				"maxTransactionAmount": 50000,
			},
			NewValue: map[string]interface{}{
				"maxTransactionAmount": 100000,
			},
			ChangedFields: []string{"maxTransactionAmount"},
			Reason:        "提高交易限额",
			IPAddress:     "192.168.1.100",
			Status:        "SUCCESS",
		},
		{
			ID:           1002,
			ConfigType:   "compliance",
			ChangeType:   "RELOAD",
			OperatorID:   "admin_002",
			OperatorName: "安全管理员",
			Timestamp:    now.Add(-4 * time.Hour),
			Reason:       "重载合规规则",
			IPAddress:    "192.168.1.101",
			Status:       "SUCCESS",
		},
		{
			ID:           1003,
			ConfigType:   "nfc_relay",
			ChangeType:   "UPDATE",
			OperatorID:   "system",
			OperatorName: "系统",
			Timestamp:    now.Add(-6 * time.Hour),
			OldValue: map[string]interface{}{
				"timeout": 30,
			},
			NewValue: map[string]interface{}{
				"timeout": 60,
			},
			ChangedFields: []string{"timeout"},
			Reason:        "自动配置调整",
			IPAddress:     "127.0.0.1",
			Status:        "SUCCESS",
		},
	}
}

func generateTrendData(days int) []ConfigChangeTrendPoint {
	var trend []ConfigChangeTrendPoint
	for i := days; i >= 0; i-- {
		date := time.Now().AddDate(0, 0, -i)
		trend = append(trend, ConfigChangeTrendPoint{
			Date:         date,
			ChangeCount:  int64(10 + i%5),
			SuccessCount: int64(9 + i%4),
			FailedCount:  int64(1 + i%2),
		})
	}
	return trend
}
