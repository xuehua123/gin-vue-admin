package nfc_relay_admin

import (
	"fmt"
	"time"

	"github.com/flipped-aurora/gin-vue-admin/server/global"
	"github.com/flipped-aurora/gin-vue-admin/server/model/nfc_relay_admin"
	"github.com/flipped-aurora/gin-vue-admin/server/model/nfc_relay_admin/request"
	"github.com/flipped-aurora/gin-vue-admin/server/model/nfc_relay_admin/response"
	"gorm.io/gorm"
)

type SecurityService struct{}

// BanClient 封禁客户端
func (s *SecurityService) BanClient(req *request.ClientBanRequest, adminID uint) error {
	// 检查是否已存在活跃的封禁记录
	var existingBan nfc_relay_admin.NfcClientBanRecord
	err := global.GVA_DB.Where("client_id = ? AND is_active = ?", req.ClientID, true).First(&existingBan).Error
	if err == nil {
		return fmt.Errorf("客户端 %s 已被封禁", req.ClientID)
	} else if err != gorm.ErrRecordNotFound {
		return err
	}

	// 计算解封时间
	var expiresAt *time.Time
	if req.BanType == "temporary" && req.Duration > 0 {
		expireTime := time.Now().Add(time.Duration(req.Duration) * time.Minute)
		expiresAt = &expireTime
	}

	banRecord := &nfc_relay_admin.NfcClientBanRecord{
		ClientID:   req.ClientID,
		UserID:     req.UserID,
		BanReason:  req.BanReason,
		BanType:    req.BanType,
		BannedBy:   adminID,
		BannedAt:   time.Now(),
		ExpiresAt:  expiresAt,
		IsActive:   true,
		Violations: 1,
		Severity:   req.Severity,
		Notes:      req.Notes,
	}

	return global.GVA_DB.Create(banRecord).Error
}

// UnbanClient 解封客户端
func (s *SecurityService) UnbanClient(req *request.ClientUnbanRequest, adminID uint) error {
	var banRecord nfc_relay_admin.NfcClientBanRecord
	err := global.GVA_DB.Where("client_id = ? AND is_active = ?", req.ClientID, true).First(&banRecord).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return fmt.Errorf("客户端 %s 未被封禁", req.ClientID)
		}
		return err
	}

	// 更新封禁记录
	now := time.Now()
	banRecord.IsActive = false
	banRecord.UnbannedBy = &adminID
	banRecord.UnbannedAt = &now

	return global.GVA_DB.Save(&banRecord).Error
}

// GetClientBanList 获取客户端封禁列表
func (s *SecurityService) GetClientBanList(req *request.ClientBanListRequest) (*response.PaginatedClientBanResponse, error) {
	var banRecords []nfc_relay_admin.NfcClientBanRecord
	var total int64

	db := global.GVA_DB.Model(&nfc_relay_admin.NfcClientBanRecord{})

	// 构建查询条件
	if req.ClientID != "" {
		db = db.Where("client_id = ?", req.ClientID)
	}
	if req.UserID != "" {
		db = db.Where("user_id = ?", req.UserID)
	}
	if req.BanType != "" {
		db = db.Where("ban_type = ?", req.BanType)
	}
	if req.IsActive != nil {
		db = db.Where("is_active = ?", *req.IsActive)
	}
	if req.Severity != "" {
		db = db.Where("severity = ?", req.Severity)
	}

	// 时间范围查询
	if req.StartTime != "" {
		if startTime, err := time.Parse(time.RFC3339, req.StartTime); err == nil {
			db = db.Where("banned_at >= ?", startTime)
		}
	}
	if req.EndTime != "" {
		if endTime, err := time.Parse(time.RFC3339, req.EndTime); err == nil {
			db = db.Where("banned_at <= ?", endTime)
		}
	}

	// 统计总数
	if err := db.Count(&total).Error; err != nil {
		return nil, err
	}

	// 分页查询
	offset := (req.Page - 1) * req.PageSize
	if err := db.Order("banned_at DESC").Offset(offset).Limit(req.PageSize).Find(&banRecords).Error; err != nil {
		return nil, err
	}

	// 转换为响应格式
	var entries []response.ClientBanRecordEntry
	for _, record := range banRecords {
		entries = append(entries, response.ClientBanRecordEntry{
			ID:         record.ID,
			ClientID:   record.ClientID,
			UserID:     record.UserID,
			BanReason:  record.BanReason,
			BanType:    record.BanType,
			BannedBy:   record.BannedBy,
			BannedAt:   record.BannedAt,
			ExpiresAt:  record.ExpiresAt,
			UnbannedBy: record.UnbannedBy,
			UnbannedAt: record.UnbannedAt,
			IsActive:   record.IsActive,
			SourceIP:   record.SourceIP,
			Violations: record.Violations,
			Severity:   record.Severity,
			Notes:      record.Notes,
			CreatedAt:  record.CreatedAt,
			UpdatedAt:  record.UpdatedAt,
		})
	}

	return &response.PaginatedClientBanResponse{
		List:     entries,
		Total:    total,
		Page:     req.Page,
		PageSize: req.PageSize,
	}, nil
}

// IsClientBanned 检查客户端是否被封禁
func (s *SecurityService) IsClientBanned(clientID string) (bool, *nfc_relay_admin.NfcClientBanRecord, error) {
	var banRecord nfc_relay_admin.NfcClientBanRecord
	err := global.GVA_DB.Where("client_id = ? AND is_active = ?", clientID, true).First(&banRecord).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return false, nil, nil
		}
		return false, nil, err
	}

	// 检查是否已过期
	if banRecord.IsExpired() {
		// 自动解封过期的封禁
		banRecord.IsActive = false
		global.GVA_DB.Save(&banRecord)
		return false, nil, nil
	}

	return true, &banRecord, nil
}

// GetUserSecurityProfile 获取用户安全档案
func (s *SecurityService) GetUserSecurityProfile(userID string) (*response.UserSecurityProfileEntry, error) {
	var profile nfc_relay_admin.NfcUserSecurityProfile
	err := global.GVA_DB.Where("user_id = ?", userID).First(&profile).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			// 如果档案不存在，创建一个默认档案
			profile = nfc_relay_admin.NfcUserSecurityProfile{
				UserID:        userID,
				Status:        "active",
				SecurityLevel: "normal",
				RiskScore:     0.0,
			}
			if createErr := global.GVA_DB.Create(&profile).Error; createErr != nil {
				return nil, createErr
			}
		} else {
			return nil, err
		}
	}

	return &response.UserSecurityProfileEntry{
		ID:               profile.ID,
		UserID:           profile.UserID,
		Status:           profile.Status,
		SecurityLevel:    profile.SecurityLevel,
		FailedLoginCount: profile.FailedLoginCount,
		LastLoginAt:      profile.LastLoginAt,
		LastLoginIP:      profile.LastLoginIP,
		LoginAttempts:    profile.LoginAttempts,
		LastAttemptAt:    profile.LastAttemptAt,
		AccountLockedAt:  profile.AccountLockedAt,
		LockExpiresAt:    profile.LockExpiresAt,
		TwoFactorEnabled: profile.TwoFactorEnabled,
		ViolationCount:   profile.ViolationCount,
		LastViolationAt:  profile.LastViolationAt,
		RiskScore:        profile.RiskScore,
		Notes:            profile.Notes,
		CreatedAt:        profile.CreatedAt,
		UpdatedAt:        profile.UpdatedAt,
	}, nil
}

// GetUserSecurityProfileList 获取用户安全档案列表
func (s *SecurityService) GetUserSecurityProfileList(req *request.UserSecurityProfileListRequest) (*response.PaginatedUserSecurityProfileResponse, error) {
	var profiles []nfc_relay_admin.NfcUserSecurityProfile
	var total int64

	db := global.GVA_DB.Model(&nfc_relay_admin.NfcUserSecurityProfile{})

	// 构建查询条件
	if req.Status != "" {
		db = db.Where("status = ?", req.Status)
	}
	if req.SecurityLevel != "" {
		db = db.Where("security_level = ?", req.SecurityLevel)
	}
	if req.MinRiskScore > 0 {
		db = db.Where("risk_score >= ?", req.MinRiskScore)
	}
	if req.MaxRiskScore > 0 {
		db = db.Where("risk_score <= ?", req.MaxRiskScore)
	}
	if req.IsLocked != nil {
		if *req.IsLocked {
			db = db.Where("account_locked_at IS NOT NULL AND (lock_expires_at IS NULL OR lock_expires_at > ?)", time.Now())
		} else {
			db = db.Where("account_locked_at IS NULL OR (lock_expires_at IS NOT NULL AND lock_expires_at <= ?)", time.Now())
		}
	}
	if req.UserIDLike != "" {
		db = db.Where("user_id LIKE ?", "%"+req.UserIDLike+"%")
	}

	// 统计总数
	if err := db.Count(&total).Error; err != nil {
		return nil, err
	}

	// 分页查询
	offset := (req.Page - 1) * req.PageSize
	if err := db.Order("created_at DESC").Offset(offset).Limit(req.PageSize).Find(&profiles).Error; err != nil {
		return nil, err
	}

	// 转换为响应格式
	var entries []response.UserSecurityProfileEntry
	for _, profile := range profiles {
		entries = append(entries, response.UserSecurityProfileEntry{
			ID:               profile.ID,
			UserID:           profile.UserID,
			Status:           profile.Status,
			SecurityLevel:    profile.SecurityLevel,
			FailedLoginCount: profile.FailedLoginCount,
			LastLoginAt:      profile.LastLoginAt,
			LastLoginIP:      profile.LastLoginIP,
			LoginAttempts:    profile.LoginAttempts,
			LastAttemptAt:    profile.LastAttemptAt,
			AccountLockedAt:  profile.AccountLockedAt,
			LockExpiresAt:    profile.LockExpiresAt,
			TwoFactorEnabled: profile.TwoFactorEnabled,
			ViolationCount:   profile.ViolationCount,
			LastViolationAt:  profile.LastViolationAt,
			RiskScore:        profile.RiskScore,
			Notes:            profile.Notes,
			CreatedAt:        profile.CreatedAt,
			UpdatedAt:        profile.UpdatedAt,
		})
	}

	return &response.PaginatedUserSecurityProfileResponse{
		List:     entries,
		Total:    total,
		Page:     req.Page,
		PageSize: req.PageSize,
	}, nil
}

// UpdateUserSecurityProfile 更新用户安全档案
func (s *SecurityService) UpdateUserSecurityProfile(req *request.UpdateUserSecurityRequest) error {
	var profile nfc_relay_admin.NfcUserSecurityProfile
	err := global.GVA_DB.Where("user_id = ?", req.UserID).First(&profile).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			// 如果档案不存在，创建新档案
			profile = nfc_relay_admin.NfcUserSecurityProfile{
				UserID: req.UserID,
			}
		} else {
			return err
		}
	}

	// 更新字段
	if req.Status != "" {
		profile.Status = req.Status
	}
	if req.SecurityLevel != "" {
		profile.SecurityLevel = req.SecurityLevel
	}
	profile.TwoFactorEnabled = req.TwoFactorEnabled
	if req.RiskScore >= 0 {
		profile.RiskScore = req.RiskScore
	}
	if req.Notes != "" {
		profile.Notes = req.Notes
	}

	return global.GVA_DB.Save(&profile).Error
}

// LockUserAccount 锁定用户账户
func (s *SecurityService) LockUserAccount(req *request.LockUserAccountRequest) error {
	var profile nfc_relay_admin.NfcUserSecurityProfile
	err := global.GVA_DB.Where("user_id = ?", req.UserID).First(&profile).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return fmt.Errorf("用户 %s 的安全档案不存在", req.UserID)
		}
		return err
	}

	now := time.Now()
	profile.AccountLockedAt = &now
	profile.Status = "locked"

	// 计算锁定过期时间
	if req.Duration > 0 {
		lockExpires := now.Add(time.Duration(req.Duration) * time.Minute)
		profile.LockExpiresAt = &lockExpires
	} else {
		profile.LockExpiresAt = nil // 永久锁定
	}

	return global.GVA_DB.Save(&profile).Error
}

// UnlockUserAccount 解锁用户账户
func (s *SecurityService) UnlockUserAccount(req *request.UnlockUserAccountRequest) error {
	var profile nfc_relay_admin.NfcUserSecurityProfile
	err := global.GVA_DB.Where("user_id = ?", req.UserID).First(&profile).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return fmt.Errorf("用户 %s 的安全档案不存在", req.UserID)
		}
		return err
	}

	profile.AccountLockedAt = nil
	profile.LockExpiresAt = nil
	profile.Status = "active"
	profile.FailedLoginCount = 0

	return global.GVA_DB.Save(&profile).Error
}

// RecordLoginAttempt 记录登录尝试
func (s *SecurityService) RecordLoginAttempt(userID, sourceIP string, success bool) error {
	var profile nfc_relay_admin.NfcUserSecurityProfile
	err := global.GVA_DB.Where("user_id = ?", userID).First(&profile).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			// 创建新的安全档案
			profile = nfc_relay_admin.NfcUserSecurityProfile{
				UserID:        userID,
				Status:        "active",
				SecurityLevel: "normal",
			}
		} else {
			return err
		}
	}

	now := time.Now()
	profile.LastAttemptAt = now
	profile.LoginAttempts++

	if success {
		profile.LastLoginAt = now
		profile.LastLoginIP = sourceIP
		profile.FailedLoginCount = 0
	} else {
		profile.FailedLoginCount++

		// 检查是否需要自动锁定账户（连续失败5次）
		if profile.FailedLoginCount >= 5 {
			profile.AccountLockedAt = &now
			profile.Status = "locked"
			lockExpires := now.Add(30 * time.Minute) // 锁定30分钟
			profile.LockExpiresAt = &lockExpires
		}
	}

	return global.GVA_DB.Save(&profile).Error
}

// GetSecuritySummary 获取安全摘要
func (s *SecurityService) GetSecuritySummary() (*response.SecuritySummaryResponse, error) {
	var summary response.SecuritySummaryResponse

	// 活跃封禁数量
	if err := global.GVA_DB.Model(&nfc_relay_admin.NfcClientBanRecord{}).
		Where("is_active = ?", true).Count(&summary.ActiveBanCount).Error; err != nil {
		return nil, err
	}

	// 锁定账户数量
	if err := global.GVA_DB.Model(&nfc_relay_admin.NfcUserSecurityProfile{}).
		Where("account_locked_at IS NOT NULL AND (lock_expires_at IS NULL OR lock_expires_at > ?)", time.Now()).
		Count(&summary.LockedAccountCount).Error; err != nil {
		return nil, err
	}

	// 高风险用户数量（风险评分 >= 7.0）
	if err := global.GVA_DB.Model(&nfc_relay_admin.NfcUserSecurityProfile{}).
		Where("risk_score >= ?", 7.0).Count(&summary.HighRiskUserCount).Error; err != nil {
		return nil, err
	}

	// 总违规次数
	if err := global.GVA_DB.Model(&nfc_relay_admin.NfcUserSecurityProfile{}).
		Select("COALESCE(SUM(violation_count), 0)").Scan(&summary.TotalViolations).Error; err != nil {
		return nil, err
	}

	return &summary, nil
}

// CleanupExpiredBans 清理过期的封禁记录
func (s *SecurityService) CleanupExpiredBans() error {
	return global.GVA_DB.Model(&nfc_relay_admin.NfcClientBanRecord{}).
		Where("is_active = ? AND expires_at IS NOT NULL AND expires_at <= ?", true, time.Now()).
		Update("is_active", false).Error
}

// CleanupExpiredLocks 清理过期的账户锁定
func (s *SecurityService) CleanupExpiredLocks() error {
	return global.GVA_DB.Model(&nfc_relay_admin.NfcUserSecurityProfile{}).
		Where("account_locked_at IS NOT NULL AND lock_expires_at IS NOT NULL AND lock_expires_at <= ?", time.Now()).
		Updates(map[string]interface{}{
			"account_locked_at": nil,
			"lock_expires_at":   nil,
			"status":            "active",
		}).Error
}
