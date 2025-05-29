package nfc_relay_admin

import (
	"bufio"
	"encoding/json"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/flipped-aurora/gin-vue-admin/server/global"
	"github.com/flipped-aurora/gin-vue-admin/server/model/common/response"
	nfcResponse "github.com/flipped-aurora/gin-vue-admin/server/model/nfc_relay_admin/response"
	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

type AuditLogsApi struct{}

// GetAuditLogs 获取审计日志
// @Summary 获取审计日志列表
// @Description 支持分页和筛选的审计日志查询
// @Tags NFC中继管理
// @Accept json
// @Produce json
// @Param page query int false "页码"
// @Param pageSize query int false "每页数量"
// @Param eventType query string false "事件类型筛选"
// @Param userID query string false "用户ID筛选"
// @Param sessionID query string false "会话ID筛选"
// @Param clientID query string false "客户端ID筛选"
// @Param startTime query string false "开始时间(ISO8601格式)"
// @Param endTime query string false "结束时间(ISO8601格式)"
// @Success 200 {object} response.Response{data=nfcResponse.PaginatedAuditLogResponse}
// @Router /api/admin/nfc-relay/v1/audit-logs [get]
func (a *AuditLogsApi) GetAuditLogs(ctx *gin.Context) {
	// 获取查询参数
	pageStr := ctx.DefaultQuery("page", "1")
	pageSizeStr := ctx.DefaultQuery("pageSize", "10")
	eventTypeFilter := ctx.Query("eventType")
	userIDFilter := ctx.Query("userID")
	sessionIDFilter := ctx.Query("sessionID")
	clientIDFilter := ctx.Query("clientID")
	startTimeStr := ctx.Query("startTime")
	endTimeStr := ctx.Query("endTime")

	page, err := strconv.Atoi(pageStr)
	if err != nil || page < 1 {
		page = 1
	}

	pageSize, err := strconv.Atoi(pageSizeStr)
	if err != nil || pageSize < 1 || pageSize > 100 {
		pageSize = 10
	}

	// 解析时间范围
	var startTime, endTime time.Time
	if startTimeStr != "" {
		if t, err := time.Parse(time.RFC3339, startTimeStr); err == nil {
			startTime = t
		}
	}
	if endTimeStr != "" {
		if t, err := time.Parse(time.RFC3339, endTimeStr); err == nil {
			endTime = t
		}
	}

	// 读取审计日志文件
	auditLogs, err := a.readAuditLogsFromFile()
	if err != nil {
		global.GVA_LOG.Error("Failed to read audit logs", zap.Error(err))
		response.FailWithMessage("读取审计日志失败", ctx)
		return
	}

	// 应用筛选条件
	var filteredLogs []nfcResponse.AuditLogEntry
	for _, log := range auditLogs {
		// 时间范围筛选
		if !startTime.IsZero() && log.EventTime.Before(startTime) {
			continue
		}
		if !endTime.IsZero() && log.EventTime.After(endTime) {
			continue
		}

		// 事件类型筛选
		if eventTypeFilter != "" && !strings.Contains(log.EventType, eventTypeFilter) {
			continue
		}

		// 用户ID筛选
		if userIDFilter != "" && !strings.Contains(log.UserID, userIDFilter) {
			continue
		}

		// 会话ID筛选
		if sessionIDFilter != "" && !strings.Contains(log.SessionID, sessionIDFilter) {
			continue
		}

		// 客户端ID筛选
		if clientIDFilter != "" && !strings.Contains(log.ClientIDInitiator, clientIDFilter) &&
			!strings.Contains(log.ClientIDResponder, clientIDFilter) {
			continue
		}

		filteredLogs = append(filteredLogs, log)
	}

	// 计算分页
	total := len(filteredLogs)
	start := (page - 1) * pageSize
	end := start + pageSize

	if start >= total {
		filteredLogs = []nfcResponse.AuditLogEntry{}
	} else {
		if end > total {
			end = total
		}
		filteredLogs = filteredLogs[start:end]
	}

	resp := nfcResponse.PaginatedAuditLogResponse{
		List:     filteredLogs,
		Total:    int64(total),
		Page:     page,
		PageSize: pageSize,
	}

	response.OkWithData(resp, ctx)
}

// readAuditLogsFromFile 从日志文件读取审计日志
func (a *AuditLogsApi) readAuditLogsFromFile() ([]nfcResponse.AuditLogEntry, error) {
	var logs []nfcResponse.AuditLogEntry

	// 获取日志文件路径
	logDir := global.GVA_CONFIG.Zap.Director
	if logDir == "" {
		logDir = "log"
	}

	// 读取最近几天的日志文件（限制读取范围，避免性能问题）
	for i := 0; i < 7; i++ { // 读取最近7天的日志
		date := time.Now().AddDate(0, 0, -i)
		dateStr := date.Format("2006-01-02")
		logFile := filepath.Join(logDir, dateStr, "nfc-relay-audit.log")

		if _, err := os.Stat(logFile); os.IsNotExist(err) {
			continue // 文件不存在，跳过
		}

		fileLogs, err := a.readLogFile(logFile)
		if err != nil {
			global.GVA_LOG.Warn("Failed to read log file", zap.String("file", logFile), zap.Error(err))
			continue
		}

		logs = append(logs, fileLogs...)
	}

	// 按时间倒序排序（最新的在前面）
	for i := 0; i < len(logs)/2; i++ {
		logs[i], logs[len(logs)-1-i] = logs[len(logs)-1-i], logs[i]
	}

	return logs, nil
}

// readLogFile 读取单个日志文件
func (a *AuditLogsApi) readLogFile(filename string) ([]nfcResponse.AuditLogEntry, error) {
	var logs []nfcResponse.AuditLogEntry

	file, err := os.Open(filename)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := scanner.Text()
		if line == "" {
			continue
		}

		// 尝试解析JSON格式的日志行
		var rawLog map[string]interface{}
		if err := json.Unmarshal([]byte(line), &rawLog); err != nil {
			continue // 跳过无法解析的行
		}

		// 只处理审计日志相关的条目
		if level, ok := rawLog["level"].(string); !ok || level != "info" {
			continue
		}

		msg, ok := rawLog["msg"].(string)
		if !ok || !strings.Contains(msg, "audit") {
			continue
		}

		// 解析时间戳
		var timestamp time.Time
		if ts, ok := rawLog["ts"].(string); ok {
			if t, err := time.Parse(time.RFC3339, ts); err == nil {
				timestamp = t
			}
		}

		// 构建审计日志条目
		detailsJSON, _ := json.Marshal(rawLog)
		entry := nfcResponse.AuditLogEntry{
			EventTime:         timestamp,
			EventType:         a.extractFieldString(rawLog, "event_type"),
			SessionID:         a.extractFieldString(rawLog, "session_id"),
			ClientIDInitiator: a.extractFieldString(rawLog, "client_id_initiator"),
			ClientIDResponder: a.extractFieldString(rawLog, "client_id_responder"),
			UserID:            a.extractFieldString(rawLog, "user_id"),
			SourceIP:          a.extractFieldString(rawLog, "source_ip"),
			Details:           string(detailsJSON), // 转换为JSON字符串
		}

		logs = append(logs, entry)
	}

	if err := scanner.Err(); err != nil {
		return nil, err
	}

	return logs, nil
}

// extractFieldString 从map中提取字符串字段
func (a *AuditLogsApi) extractFieldString(data map[string]interface{}, field string) string {
	if value, ok := data[field].(string); ok {
		return value
	}
	return ""
}
