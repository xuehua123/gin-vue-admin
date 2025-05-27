package service

import (
	"bufio"
	"encoding/json"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/flipped-aurora/gin-vue-admin/server/global"
	"github.com/flipped-aurora/gin-vue-admin/server/nfc_relay/model/admin_request"
	"github.com/flipped-aurora/gin-vue-admin/server/nfc_relay/model/admin_response"
	"go.uber.org/zap"
)

// AdminAuditLogService 审计日志服务
type AdminAuditLogService struct{}

// GetAuditLogs 获取审计日志列表
func (s *AdminAuditLogService) GetAuditLogs(params admin_request.AuditLogListParams) (admin_response.PaginatedAuditLogResponse, error) {
	// 获取日志目录
	logDir := filepath.Join(global.GVA_CONFIG.Zap.Director)
	global.GVA_LOG.Info("准备从日志目录读取审计日志", zap.String("logDir", logDir))

	// 列出日期目录，并按降序排序（最新的日期在前）
	dateDirs, err := listDateDirs(logDir)
	if err != nil {
		global.GVA_LOG.Error("列出日志日期目录失败", zap.Error(err))
		return admin_response.PaginatedAuditLogResponse{}, err
	}

	startTime, err := params.GetStartTimeAsTime()
	if err != nil {
		global.GVA_LOG.Error("解析开始时间失败", zap.Error(err), zap.String("startTime", params.StartTime))
		return admin_response.PaginatedAuditLogResponse{}, err
	}

	endTime, err := params.GetEndTimeAsTime()
	if err != nil {
		global.GVA_LOG.Error("解析结束时间失败", zap.Error(err), zap.String("endTime", params.EndTime))
		return admin_response.PaginatedAuditLogResponse{}, err
	}

	// 收集所有审计日志
	var allLogs []admin_response.AuditLogItem
	for _, dateDir := range dateDirs {
		// 解析目录名称中的日期
		dirDate, err := time.Parse("2006-01-02", filepath.Base(dateDir))
		if err != nil {
			global.GVA_LOG.Warn("跳过无效日期格式的目录", zap.String("dir", dateDir), zap.Error(err))
			continue
		}

		// 检查日期是否在查询范围内
		if dirDate.Before(startTime.Truncate(24*time.Hour)) || dirDate.After(endTime.Truncate(24*time.Hour).Add(24*time.Hour)) {
			continue
		}

		// 处理当天的日志文件
		infoLogPath := filepath.Join(dateDir, "info.log")
		logs, err := readAuditLogsFromFile(infoLogPath, startTime, endTime, params)
		if err != nil {
			global.GVA_LOG.Warn("从日志文件读取审计日志失败", zap.String("file", infoLogPath), zap.Error(err))
			continue
		}

		allLogs = append(allLogs, logs...)
	}

	// 按时间戳降序排序（最新的在前）
	sort.Slice(allLogs, func(i, j int) bool {
		return allLogs[i].Timestamp > allLogs[j].Timestamp
	})

	// 计算总数
	total := len(allLogs)

	// 计算分页
	start := (params.Page - 1) * params.PageSize
	end := start + params.PageSize
	if start >= total {
		// 超出范围，返回空列表
		return admin_response.PaginatedAuditLogResponse{
			List:     []admin_response.AuditLogItem{},
			Total:    total,
			Page:     params.Page,
			PageSize: params.PageSize,
		}, nil
	}

	if end > total {
		end = total
	}

	// 返回分页后的结果
	return admin_response.PaginatedAuditLogResponse{
		List:     allLogs[start:end],
		Total:    total,
		Page:     params.Page,
		PageSize: params.PageSize,
	}, nil
}

// listDateDirs 列出日志目录下的日期子目录
func listDateDirs(logDir string) ([]string, error) {
	entries, err := os.ReadDir(logDir)
	if err != nil {
		return nil, err
	}

	var dateDirs []string
	for _, entry := range entries {
		if entry.IsDir() {
			// 检查目录名是否符合日期格式 YYYY-MM-DD
			if matched, _ := filepath.Match("????-??-??", entry.Name()); matched {
				dateDirs = append(dateDirs, filepath.Join(logDir, entry.Name()))
			}
		}
	}

	// 按降序排序（最新的日期在前）
	sort.Slice(dateDirs, func(i, j int) bool {
		return filepath.Base(dateDirs[i]) > filepath.Base(dateDirs[j])
	})

	return dateDirs, nil
}

// readAuditLogsFromFile 从日志文件中读取审计日志
func readAuditLogsFromFile(filePath string, startTime, endTime time.Time, params admin_request.AuditLogListParams) ([]admin_response.AuditLogItem, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	var logs []admin_response.AuditLogItem
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := scanner.Text()

		// 检查是否是审计日志（来自 AuditLogger.Named("audit")）
		if !strings.Contains(line, "[audit]") {
			continue
		}

		// 尝试解析出JSON部分
		jsonStartIndex := strings.Index(line, "{")
		if jsonStartIndex == -1 {
			continue
		}

		jsonStr := line[jsonStartIndex:]
		var logEvent global.AuditEvent
		if err := json.Unmarshal([]byte(jsonStr), &logEvent); err != nil {
			global.GVA_LOG.Debug("解析日志JSON失败", zap.String("json", jsonStr), zap.Error(err))
			continue
		}

		// 检查事件类型过滤
		if params.EventType != "" && logEvent.EventType != params.EventType {
			continue
		}

		// 检查会话ID过滤
		if params.SessionID != "" && !strings.Contains(logEvent.SessionID, params.SessionID) {
			continue
		}

		// 检查用户ID过滤
		if params.UserID != "" && !strings.Contains(logEvent.UserID, params.UserID) {
			continue
		}

		// 检查客户端ID过滤
		if params.ClientID != "" {
			matchesInitiator := strings.Contains(logEvent.ClientIDInitiator, params.ClientID)
			matchesResponder := strings.Contains(logEvent.ClientIDResponder, params.ClientID)
			if !matchesInitiator && !matchesResponder {
				continue
			}
		}

		// 检查时间范围
		logTime, err := time.Parse(time.RFC3339Nano, logEvent.Timestamp)
		if err != nil {
			global.GVA_LOG.Debug("解析日志时间戳失败", zap.String("timestamp", logEvent.Timestamp), zap.Error(err))
			continue
		}
		if logTime.Before(startTime) || logTime.After(endTime) {
			continue
		}

		// 将日志事件转换为响应项
		logItem := admin_response.AuditLogItem{
			Timestamp:         logEvent.Timestamp,
			EventType:         logEvent.EventType,
			SessionID:         logEvent.SessionID,
			ClientIDInitiator: logEvent.ClientIDInitiator,
			ClientIDResponder: logEvent.ClientIDResponder,
			UserID:            logEvent.UserID,
			SourceIP:          logEvent.SourceIP,
			Details:           logEvent.Details,
		}
		logs = append(logs, logItem)
	}

	if err := scanner.Err(); err != nil {
		return nil, err
	}

	return logs, nil
}
