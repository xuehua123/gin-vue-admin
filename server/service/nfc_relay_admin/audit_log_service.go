package nfc_relay_admin

import (
	"encoding/json"
	"time"

	"github.com/flipped-aurora/gin-vue-admin/server/global"
	"github.com/flipped-aurora/gin-vue-admin/server/model/nfc_relay_admin"
	"github.com/flipped-aurora/gin-vue-admin/server/model/nfc_relay_admin/request"
	"github.com/flipped-aurora/gin-vue-admin/server/model/nfc_relay_admin/response"
)

type AuditLogService struct{}

// CreateAuditLog 创建审计日志
func (s *AuditLogService) CreateAuditLog(req *request.CreateAuditLogRequest) error {
	// 将详情转换为JSON字符串
	detailsJSON := ""
	if req.Details != nil {
		if detailsBytes, err := json.Marshal(req.Details); err == nil {
			detailsJSON = string(detailsBytes)
		}
	}

	auditLog := &nfc_relay_admin.NfcAuditLog{
		EventType:         req.EventType,
		SessionID:         req.SessionID,
		ClientIDInitiator: req.ClientIDInitiator,
		ClientIDResponder: req.ClientIDResponder,
		UserID:            req.UserID,
		SourceIP:          req.SourceIP,
		UserAgent:         req.UserAgent,
		Details:           detailsJSON,
		Result:            req.Result,
		ErrorMessage:      req.ErrorMessage,
		Duration:          req.Duration,
		Resource:          req.Resource,
		Action:            req.Action,
		Level:             req.Level,
		Category:          req.Category,
		ServerID:          req.ServerID,
		RequestID:         req.RequestID,
		EventTime:         time.Now(),
	}

	return global.GVA_DB.Create(auditLog).Error
}

// GetAuditLogList 获取审计日志列表
func (s *AuditLogService) GetAuditLogList(req *request.AuditLogListRequest) (*response.PaginatedAuditLogResponse, error) {
	var auditLogs []nfc_relay_admin.NfcAuditLog
	var total int64

	db := global.GVA_DB.Model(&nfc_relay_admin.NfcAuditLog{})

	// 构建查询条件
	if req.EventType != "" {
		db = db.Where("event_type = ?", req.EventType)
	}
	if req.UserID != "" {
		db = db.Where("user_id = ?", req.UserID)
	}
	if req.SessionID != "" {
		db = db.Where("session_id = ?", req.SessionID)
	}
	if req.ClientID != "" {
		db = db.Where("client_id_initiator = ? OR client_id_responder = ?", req.ClientID, req.ClientID)
	}
	if req.Level != "" {
		db = db.Where("level = ?", req.Level)
	}
	if req.Category != "" {
		db = db.Where("category = ?", req.Category)
	}
	if req.Result != "" {
		db = db.Where("result = ?", req.Result)
	}
	if req.SourceIP != "" {
		db = db.Where("source_ip = ?", req.SourceIP)
	}

	// 时间范围查询
	startTime, err := req.GetStartTimeAsTime()
	if err == nil {
		db = db.Where("event_time >= ?", startTime)
	}

	endTime, err := req.GetEndTimeAsTime()
	if err == nil {
		db = db.Where("event_time <= ?", endTime)
	}

	// 关键词搜索
	if req.Keyword != "" {
		keyword := "%" + req.Keyword + "%"
		db = db.Where("details LIKE ? OR error_message LIKE ? OR resource LIKE ? OR action LIKE ?",
			keyword, keyword, keyword, keyword)
	}

	// 统计总数
	if err := db.Count(&total).Error; err != nil {
		return nil, err
	}

	// 分页查询
	offset := (req.Page - 1) * req.PageSize
	if err := db.Order("event_time DESC").Offset(offset).Limit(req.PageSize).Find(&auditLogs).Error; err != nil {
		return nil, err
	}

	// 转换为响应格式
	var entries []response.AuditLogEntry
	for _, log := range auditLogs {
		entries = append(entries, response.AuditLogEntry{
			ID:                log.ID,
			EventType:         log.EventType,
			SessionID:         log.SessionID,
			ClientIDInitiator: log.ClientIDInitiator,
			ClientIDResponder: log.ClientIDResponder,
			UserID:            log.UserID,
			SourceIP:          log.SourceIP,
			UserAgent:         log.UserAgent,
			Details:           log.Details,
			Result:            log.Result,
			ErrorMessage:      log.ErrorMessage,
			Duration:          log.Duration,
			Resource:          log.Resource,
			Action:            log.Action,
			Level:             log.Level,
			Category:          log.Category,
			ServerID:          log.ServerID,
			RequestID:         log.RequestID,
			EventTime:         log.EventTime,
			CreatedAt:         log.CreatedAt,
		})
	}

	return &response.PaginatedAuditLogResponse{
		List:     entries,
		Total:    total,
		Page:     req.Page,
		PageSize: req.PageSize,
	}, nil
}

// GetAuditLogStats 获取审计日志统计信息
func (s *AuditLogService) GetAuditLogStats() (*response.AuditLogStatsResponse, error) {
	var stats response.AuditLogStatsResponse

	// 获取总日志数
	if err := global.GVA_DB.Model(&nfc_relay_admin.NfcAuditLog{}).Count(&stats.TotalLogs).Error; err != nil {
		return nil, err
	}

	// 按级别统计
	if err := global.GVA_DB.Model(&nfc_relay_admin.NfcAuditLog{}).
		Where("level = ?", "error").Count(&stats.ErrorCount).Error; err != nil {
		return nil, err
	}

	if err := global.GVA_DB.Model(&nfc_relay_admin.NfcAuditLog{}).
		Where("level = ?", "warn").Count(&stats.WarningCount).Error; err != nil {
		return nil, err
	}

	if err := global.GVA_DB.Model(&nfc_relay_admin.NfcAuditLog{}).
		Where("level = ?", "info").Count(&stats.InfoCount).Error; err != nil {
		return nil, err
	}

	// 事件类型统计
	var eventTypeResults []struct {
		EventType string `gorm:"column:event_type"`
		Count     int64  `gorm:"column:count"`
	}
	if err := global.GVA_DB.Model(&nfc_relay_admin.NfcAuditLog{}).
		Select("event_type, COUNT(*) as count").
		Group("event_type").
		Scan(&eventTypeResults).Error; err != nil {
		return nil, err
	}

	stats.EventTypeStats = make(map[string]int64)
	for _, result := range eventTypeResults {
		stats.EventTypeStats[result.EventType] = result.Count
	}

	// 分类统计
	var categoryResults []struct {
		Category string `gorm:"column:category"`
		Count    int64  `gorm:"column:count"`
	}
	if err := global.GVA_DB.Model(&nfc_relay_admin.NfcAuditLog{}).
		Where("category != ''").
		Select("category, COUNT(*) as count").
		Group("category").
		Scan(&categoryResults).Error; err != nil {
		return nil, err
	}

	stats.CategoryStats = make(map[string]int64)
	for _, result := range categoryResults {
		stats.CategoryStats[result.Category] = result.Count
	}

	// 按小时统计（最近24小时）
	now := time.Now()
	stats.HourlyStats = make([]response.HourlyLogStats, 0, 24)

	for i := 23; i >= 0; i-- {
		hour := now.Add(-time.Duration(i) * time.Hour)
		hourStart := time.Date(hour.Year(), hour.Month(), hour.Day(), hour.Hour(), 0, 0, 0, hour.Location())
		hourEnd := hourStart.Add(time.Hour)

		var count int64
		if err := global.GVA_DB.Model(&nfc_relay_admin.NfcAuditLog{}).
			Where("event_time >= ? AND event_time < ?", hourStart, hourEnd).
			Count(&count).Error; err != nil {
			return nil, err
		}

		stats.HourlyStats = append(stats.HourlyStats, response.HourlyLogStats{
			Hour:  hour.Hour(),
			Count: count,
		})
	}

	return &stats, nil
}

// DeleteOldAuditLogs 删除过期的审计日志
func (s *AuditLogService) DeleteOldAuditLogs(retentionDays int) error {
	cutoffTime := time.Now().AddDate(0, 0, -retentionDays)

	return global.GVA_DB.Where("created_at < ?", cutoffTime).
		Delete(&nfc_relay_admin.NfcAuditLog{}).Error
}

// ExportAuditLogs 导出审计日志
func (s *AuditLogService) ExportAuditLogs(req *request.AuditLogListRequest) ([]nfc_relay_admin.NfcAuditLog, error) {
	var auditLogs []nfc_relay_admin.NfcAuditLog

	db := global.GVA_DB.Model(&nfc_relay_admin.NfcAuditLog{})

	// 应用相同的过滤条件（复用查询逻辑）
	if req.EventType != "" {
		db = db.Where("event_type = ?", req.EventType)
	}
	if req.UserID != "" {
		db = db.Where("user_id = ?", req.UserID)
	}
	if req.SessionID != "" {
		db = db.Where("session_id = ?", req.SessionID)
	}
	if req.ClientID != "" {
		db = db.Where("client_id_initiator = ? OR client_id_responder = ?", req.ClientID, req.ClientID)
	}

	// 时间范围查询
	startTime, err := req.GetStartTimeAsTime()
	if err == nil {
		db = db.Where("event_time >= ?", startTime)
	}

	endTime, err := req.GetEndTimeAsTime()
	if err == nil {
		db = db.Where("event_time <= ?", endTime)
	}

	if err := db.Order("event_time DESC").Find(&auditLogs).Error; err != nil {
		return nil, err
	}

	return auditLogs, nil
}

// BatchCreateAuditLogs 批量创建审计日志
func (s *AuditLogService) BatchCreateAuditLogs(logs []request.CreateAuditLogRequest) error {
	if len(logs) == 0 {
		return nil
	}

	var auditLogs []nfc_relay_admin.NfcAuditLog

	for _, req := range logs {
		detailsJSON := ""
		if req.Details != nil {
			if detailsBytes, err := json.Marshal(req.Details); err == nil {
				detailsJSON = string(detailsBytes)
			}
		}

		auditLogs = append(auditLogs, nfc_relay_admin.NfcAuditLog{
			EventType:         req.EventType,
			SessionID:         req.SessionID,
			ClientIDInitiator: req.ClientIDInitiator,
			ClientIDResponder: req.ClientIDResponder,
			UserID:            req.UserID,
			SourceIP:          req.SourceIP,
			UserAgent:         req.UserAgent,
			Details:           detailsJSON,
			Result:            req.Result,
			ErrorMessage:      req.ErrorMessage,
			Duration:          req.Duration,
			Resource:          req.Resource,
			Action:            req.Action,
			Level:             req.Level,
			Category:          req.Category,
			ServerID:          req.ServerID,
			RequestID:         req.RequestID,
			EventTime:         time.Now(),
		})
	}

	// 使用批量插入提高性能
	return global.GVA_DB.CreateInBatches(auditLogs, 100).Error
}

// SearchAuditLogs 高级搜索审计日志
func (s *AuditLogService) SearchAuditLogs(eventTypes []string, userIDs []string, ipAddresses []string,
	startTime, endTime time.Time, page, pageSize int) (*response.PaginatedAuditLogResponse, error) {

	var auditLogs []nfc_relay_admin.NfcAuditLog
	var total int64

	db := global.GVA_DB.Model(&nfc_relay_admin.NfcAuditLog{})

	// 事件类型过滤
	if len(eventTypes) > 0 {
		db = db.Where("event_type IN ?", eventTypes)
	}

	// 用户ID过滤
	if len(userIDs) > 0 {
		db = db.Where("user_id IN ?", userIDs)
	}

	// IP地址过滤
	if len(ipAddresses) > 0 {
		db = db.Where("source_ip IN ?", ipAddresses)
	}

	// 时间范围
	if !startTime.IsZero() {
		db = db.Where("event_time >= ?", startTime)
	}
	if !endTime.IsZero() {
		db = db.Where("event_time <= ?", endTime)
	}

	// 统计总数
	if err := db.Count(&total).Error; err != nil {
		return nil, err
	}

	// 分页查询
	offset := (page - 1) * pageSize
	if err := db.Order("event_time DESC").Offset(offset).Limit(pageSize).Find(&auditLogs).Error; err != nil {
		return nil, err
	}

	// 转换为响应格式
	var entries []response.AuditLogEntry
	for _, log := range auditLogs {
		entries = append(entries, response.AuditLogEntry{
			ID:                log.ID,
			EventType:         log.EventType,
			SessionID:         log.SessionID,
			ClientIDInitiator: log.ClientIDInitiator,
			ClientIDResponder: log.ClientIDResponder,
			UserID:            log.UserID,
			SourceIP:          log.SourceIP,
			UserAgent:         log.UserAgent,
			Details:           log.Details,
			Result:            log.Result,
			ErrorMessage:      log.ErrorMessage,
			Duration:          log.Duration,
			Resource:          log.Resource,
			Action:            log.Action,
			Level:             log.Level,
			Category:          log.Category,
			ServerID:          log.ServerID,
			RequestID:         log.RequestID,
			EventTime:         log.EventTime,
			CreatedAt:         log.CreatedAt,
		})
	}

	return &response.PaginatedAuditLogResponse{
		List:     entries,
		Total:    total,
		Page:     page,
		PageSize: pageSize,
	}, nil
}
