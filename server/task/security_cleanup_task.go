package task

import (
	"time"

	"github.com/flipped-aurora/gin-vue-admin/server/global"
	"github.com/flipped-aurora/gin-vue-admin/server/service"
	"go.uber.org/zap"
)

// SecurityCleanupTask 安全数据清理任务
func SecurityCleanupTask() {
	global.GVA_LOG.Info("开始执行安全数据清理任务")

	// 清理过期的客户端封禁记录
	err := service.ServiceGroupApp.NfcRelayAdminServiceGroup.SecurityService.CleanupExpiredBans()
	if err != nil {
		global.GVA_LOG.Error("清理过期封禁记录失败", zap.Error(err))
	} else {
		global.GVA_LOG.Info("清理过期封禁记录成功")
	}

	// 清理过期的账户锁定
	err = service.ServiceGroupApp.NfcRelayAdminServiceGroup.SecurityService.CleanupExpiredLocks()
	if err != nil {
		global.GVA_LOG.Error("清理过期账户锁定失败", zap.Error(err))
	} else {
		global.GVA_LOG.Info("清理过期账户锁定成功")
	}

	global.GVA_LOG.Info("安全数据清理任务执行完成")
}

// AuditLogCleanupTask 审计日志清理任务
func AuditLogCleanupTask() {
	global.GVA_LOG.Info("开始执行审计日志清理任务")

	// 默认保留90天的审计日志
	retentionDays := 90

	err := service.ServiceGroupApp.NfcRelayAdminServiceGroup.AuditLogService.DeleteOldAuditLogs(retentionDays)
	if err != nil {
		global.GVA_LOG.Error("清理过期审计日志失败", zap.Error(err))
	} else {
		global.GVA_LOG.Info("清理过期审计日志成功", zap.Int("retention_days", retentionDays))
	}

	global.GVA_LOG.Info("审计日志清理任务执行完成")
}

// StartSecurityTasks 启动安全相关的定时任务
func StartSecurityTasks() {
	// 每小时执行一次安全数据清理
	securityTicker := time.NewTicker(1 * time.Hour)
	go func() {
		for {
			select {
			case <-securityTicker.C:
				SecurityCleanupTask()
			}
		}
	}()

	// 每天凌晨2点执行审计日志清理
	auditLogTicker := time.NewTicker(24 * time.Hour)
	go func() {
		// 等待到下一个凌晨2点
		now := time.Now()
		next := time.Date(now.Year(), now.Month(), now.Day()+1, 2, 0, 0, 0, now.Location())
		if now.Hour() < 2 {
			next = time.Date(now.Year(), now.Month(), now.Day(), 2, 0, 0, 0, now.Location())
		}

		time.Sleep(time.Until(next))
		AuditLogCleanupTask() // 立即执行一次

		for {
			select {
			case <-auditLogTicker.C:
				AuditLogCleanupTask()
			}
		}
	}()

	global.GVA_LOG.Info("安全相关定时任务已启动")
}
