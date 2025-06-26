package common

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"
	"time"

	"github.com/flipped-aurora/gin-vue-admin/server/global"
	"github.com/google/uuid"
	"go.uber.org/zap"
)

// UserCacheService 用户缓存服务
// 用于解决UUID和数字ID之间的转换问题，提高性能
type UserCacheService struct{}

// UserCacheInfo 用户缓存信息
type UserCacheInfo struct {
	ID       uint   `json:"id"`
	UUID     string `json:"uuid"`
	Username string `json:"username"`
	NickName string `json:"nick_name"`
}

// GetUserIDByUUID 通过UUID获取用户的数字ID
// 优先从Redis缓存查找，未命中时查询数据库并缓存结果
func (s *UserCacheService) GetUserIDByUUID(userUUID string) (uint, error) {
	if userUUID == "" {
		return 0, fmt.Errorf("用户UUID不能为空")
	}

	ctx := context.Background()
	cacheKey := fmt.Sprintf("auth:user_cache:%s", userUUID)

	// 1. 先从Redis缓存查找
	cachedID, err := global.GVA_REDIS.HGet(ctx, cacheKey, "id").Result()
	if err == nil {
		if id, parseErr := strconv.ParseUint(cachedID, 10, 32); parseErr == nil {
			global.GVA_LOG.Debug("从缓存获取用户ID成功",
				zap.String("uuid", userUUID),
				zap.Uint64("id", id))
			return uint(id), nil
		}
	}

	// 2. 缓存未命中，查询数据库
	var user struct {
		ID       uint   `gorm:"column:id"`
		Username string `gorm:"column:username"`
		NickName string `gorm:"column:nick_name"`
	}

	if err := global.GVA_DB.Table("sys_users").
		Select("id, username, nick_name").
		Where("uuid = ?", userUUID).
		First(&user).Error; err != nil {
		global.GVA_LOG.Error("查询用户信息失败",
			zap.Error(err),
			zap.String("uuid", userUUID))
		return 0, fmt.Errorf("用户不存在或查询失败: %w", err)
	}

	// 3. 将结果写入缓存
	if err := s.CacheUserInfo(userUUID, user.ID, user.Username, user.NickName); err != nil {
		global.GVA_LOG.Warn("缓存用户信息失败",
			zap.Error(err),
			zap.String("uuid", userUUID))
		// 不影响主流程，继续返回结果
	}

	global.GVA_LOG.Debug("从数据库获取用户ID成功",
		zap.String("uuid", userUUID),
		zap.Uint("id", user.ID))

	return user.ID, nil
}

// GetUserInfoByUUID 获取完整的用户缓存信息
func (s *UserCacheService) GetUserInfoByUUID(userUUID string) (*UserCacheInfo, error) {
	if userUUID == "" {
		return nil, fmt.Errorf("用户UUID不能为空")
	}

	ctx := context.Background()
	cacheKey := fmt.Sprintf("auth:user_cache:%s", userUUID)

	// 从Redis获取完整用户信息
	userInfo, err := global.GVA_REDIS.HGetAll(ctx, cacheKey).Result()
	if err == nil && len(userInfo) > 0 {
		// 缓存命中，解析数据
		if idStr, exists := userInfo["id"]; exists {
			if id, parseErr := strconv.ParseUint(idStr, 10, 32); parseErr == nil {
				return &UserCacheInfo{
					ID:       uint(id),
					UUID:     userInfo["uuid"],
					Username: userInfo["username"],
					NickName: userInfo["nick_name"],
				}, nil
			}
		}
	}

	// 缓存未命中或数据不完整，查询数据库
	userID, err := s.GetUserIDByUUID(userUUID)
	if err != nil {
		return nil, err
	}

	// 重新从缓存获取（此时应该已被GetUserIDByUUID方法缓存）
	userInfo, err = global.GVA_REDIS.HGetAll(ctx, cacheKey).Result()
	if err != nil {
		global.GVA_LOG.Error("获取缓存用户信息失败",
			zap.Error(err),
			zap.String("uuid", userUUID))
		// 返回基础信息
		return &UserCacheInfo{
			ID:   userID,
			UUID: userUUID,
		}, nil
	}

	return &UserCacheInfo{
		ID:       userID,
		UUID:     userInfo["uuid"],
		Username: userInfo["username"],
		NickName: userInfo["nick_name"],
	}, nil
}

// CacheUserInfo 缓存用户信息到Redis
func (s *UserCacheService) CacheUserInfo(userUUID string, id uint, username, nickName string) error {
	if userUUID == "" {
		return fmt.Errorf("用户UUID不能为空")
	}

	ctx := context.Background()
	cacheKey := fmt.Sprintf("auth:user_cache:%s", userUUID)

	// 构建缓存数据
	userInfo := map[string]interface{}{
		"id":        id,
		"uuid":      userUUID,
		"username":  username,
		"nick_name": nickName,
		"cached_at": time.Now().Unix(),
	}

	// 使用Pipeline提高性能
	pipe := global.GVA_REDIS.Pipeline()
	pipe.HMSet(ctx, cacheKey, userInfo)
	pipe.Expire(ctx, cacheKey, time.Hour) // 1小时过期

	if _, err := pipe.Exec(ctx); err != nil {
		global.GVA_LOG.Error("缓存用户信息失败",
			zap.Error(err),
			zap.String("uuid", userUUID),
			zap.Uint("id", id))
		return fmt.Errorf("缓存用户信息失败: %w", err)
	}

	global.GVA_LOG.Debug("缓存用户信息成功",
		zap.String("uuid", userUUID),
		zap.Uint("id", id),
		zap.String("username", username))

	return nil
}

// InvalidateUserCache 使指定用户的缓存失效
func (s *UserCacheService) InvalidateUserCache(userUUID string) error {
	if userUUID == "" {
		return fmt.Errorf("用户UUID不能为空")
	}

	ctx := context.Background()
	cacheKey := fmt.Sprintf("auth:user_cache:%s", userUUID)

	// 【增强】先检查键是否存在，用于统计
	exists, err := global.GVA_REDIS.Exists(ctx, cacheKey).Result()
	if err != nil {
		global.GVA_LOG.Warn("检查用户缓存键存在性失败",
			zap.Error(err),
			zap.String("uuid", userUUID),
			zap.String("cacheKey", cacheKey))
	}

	if err := global.GVA_REDIS.Del(ctx, cacheKey).Err(); err != nil {
		global.GVA_LOG.Error("清除用户缓存失败",
			zap.Error(err),
			zap.String("uuid", userUUID),
			zap.String("cacheKey", cacheKey))
		return fmt.Errorf("清除用户缓存失败: %w", err)
	}

	// 【增强】记录更详细的清理信息
	global.GVA_LOG.Info("清除用户缓存成功",
		zap.String("uuid", userUUID),
		zap.String("cacheKey", cacheKey),
		zap.Int64("existsBefore", exists))

	return nil
}

// ValidateUUID 验证UUID格式是否正确
func (s *UserCacheService) ValidateUUID(userUUID string) error {
	if userUUID == "" {
		return fmt.Errorf("UUID不能为空")
	}

	if _, err := uuid.Parse(userUUID); err != nil {
		return fmt.Errorf("无效的UUID格式: %w", err)
	}

	return nil
}

// BatchCacheUserInfo 批量缓存用户信息
func (s *UserCacheService) BatchCacheUserInfo(users []UserCacheInfo) error {
	if len(users) == 0 {
		return nil
	}

	ctx := context.Background()
	pipe := global.GVA_REDIS.Pipeline()

	for _, user := range users {
		if err := s.ValidateUUID(user.UUID); err != nil {
			global.GVA_LOG.Warn("跳过无效UUID的用户",
				zap.String("uuid", user.UUID),
				zap.Error(err))
			continue
		}

		cacheKey := fmt.Sprintf("auth:user_cache:%s", user.UUID)
		userInfo := map[string]interface{}{
			"id":        user.ID,
			"uuid":      user.UUID,
			"username":  user.Username,
			"nick_name": user.NickName,
			"cached_at": time.Now().Unix(),
		}

		pipe.HMSet(ctx, cacheKey, userInfo)
		pipe.Expire(ctx, cacheKey, time.Hour)
	}

	if _, err := pipe.Exec(ctx); err != nil {
		global.GVA_LOG.Error("批量缓存用户信息失败",
			zap.Error(err),
			zap.Int("count", len(users)))
		return fmt.Errorf("批量缓存用户信息失败: %w", err)
	}

	global.GVA_LOG.Info("批量缓存用户信息成功", zap.Int("count", len(users)))
	return nil
}

// GetCacheStats 获取缓存统计信息
func (s *UserCacheService) GetCacheStats() (map[string]interface{}, error) {
	ctx := context.Background()

	// 统计缓存键数量
	cacheKeyPattern := "auth:user_cache:*"
	keys, err := global.GVA_REDIS.Keys(ctx, cacheKeyPattern).Result()
	if err != nil {
		return nil, fmt.Errorf("获取缓存统计失败: %w", err)
	}

	stats := map[string]interface{}{
		"total_cached_users": len(keys),
		"cache_key_pattern":  cacheKeyPattern,
		"ttl_hours":          1,
		"last_check":         time.Now().Format(time.RFC3339),
	}

	return stats, nil
}

// 全局用户缓存服务实例
var UserCacheServiceApp = new(UserCacheService)

// 【新增】VerifyCleanupCompleteness 验证配对取消后的缓存清理完整性
// 这个方法用于运维监控和故障排查，检查是否有遗漏的缓存
func (s *UserCacheService) VerifyCleanupCompleteness(userUUID string, userID uint, clientID string, role string) *CacheCleanupReport {
	ctx := context.Background()
	report := &CacheCleanupReport{
		UserUUID:    userUUID,
		UserID:      userID,
		ClientID:    clientID,
		Role:        role,
		CheckedAt:   time.Now(),
		Issues:      make([]string, 0),
		CleanedKeys: make([]string, 0),
	}

	// 检查所有可能残留的缓存键
	keysToCheck := []struct {
		Key         string
		Description string
		Critical    bool
	}{
		{fmt.Sprintf("auth:user_cache:%s", userUUID), "用户认证缓存", true},
		{fmt.Sprintf("client_heartbeat:%s", clientID), "客户端心跳缓存", false},
		{fmt.Sprintf("client_status:%s", clientID), "客户端状态缓存", false},
		{fmt.Sprintf("pairing:state:%d", userID), "配对状态缓存", true},
		{fmt.Sprintf("pairing:timeout:%d:%s", userID, role), "配对超时缓存", false},
		{fmt.Sprintf("mqtt:client_ref:%s", clientID), "MQTT客户端引用", false},
		{fmt.Sprintf("user_roles:%d", userID), "用户角色缓存", false},
	}

	for _, check := range keysToCheck {
		exists, err := global.GVA_REDIS.Exists(ctx, check.Key).Result()
		if err != nil {
			report.Issues = append(report.Issues,
				fmt.Sprintf("检查失败: %s - %v", check.Description, err))
			continue
		}

		if exists > 0 {
			if check.Critical {
				report.Issues = append(report.Issues,
					fmt.Sprintf("关键缓存未清理: %s (%s)", check.Key, check.Description))
			} else {
				report.Issues = append(report.Issues,
					fmt.Sprintf("缓存残留: %s (%s)", check.Key, check.Description))
			}
		} else {
			report.CleanedKeys = append(report.CleanedKeys, check.Key)
		}
	}

	// 检查集合类型的缓存
	sets := []struct {
		Key         string
		Member      string
		Description string
	}{
		{"clients_online_all", clientID, "全局在线客户端集合"},
		{fmt.Sprintf("clients_online:%s", role), clientID, "角色在线客户端集合"},
	}

	for _, set := range sets {
		isMember, err := global.GVA_REDIS.SIsMember(ctx, set.Key, set.Member).Result()
		if err != nil {
			report.Issues = append(report.Issues,
				fmt.Sprintf("检查集合失败: %s - %v", set.Description, err))
			continue
		}

		if isMember {
			report.Issues = append(report.Issues,
				fmt.Sprintf("集合缓存残留: %s 中仍有 %s", set.Key, set.Member))
		}
	}

	// 检查配对池中是否还有该用户的数据
	poolKey := fmt.Sprintf("pairing:pool:%s", role)
	members, err := global.GVA_REDIS.ZRange(ctx, poolKey, 0, -1).Result()
	if err == nil {
		for _, member := range members {
			var data map[string]interface{}
			if json.Unmarshal([]byte(member), &data) == nil {
				if uid, ok := data["user_id"].(float64); ok && uint(uid) == userID {
					report.Issues = append(report.Issues,
						fmt.Sprintf("配对池残留: %s 中仍有用户 %d 的数据", poolKey, userID))
					break
				}
			}
		}
	}

	report.IsComplete = len(report.Issues) == 0
	report.IssueCount = len(report.Issues)

	// 记录验证结果
	if report.IsComplete {
		global.GVA_LOG.Info("缓存清理验证通过",
			zap.String("userUUID", userUUID),
			zap.Uint("userID", userID),
			zap.String("clientID", clientID),
			zap.String("role", role),
			zap.Int("cleanedKeysCount", len(report.CleanedKeys)))
	} else {
		global.GVA_LOG.Warn("缓存清理验证发现问题",
			zap.String("userUUID", userUUID),
			zap.Uint("userID", userID),
			zap.String("clientID", clientID),
			zap.String("role", role),
			zap.Int("issueCount", report.IssueCount),
			zap.Strings("issues", report.Issues))
	}

	return report
}

// CacheCleanupReport 缓存清理报告
type CacheCleanupReport struct {
	UserUUID    string    `json:"user_uuid"`
	UserID      uint      `json:"user_id"`
	ClientID    string    `json:"client_id"`
	Role        string    `json:"role"`
	CheckedAt   time.Time `json:"checked_at"`
	IsComplete  bool      `json:"is_complete"`
	IssueCount  int       `json:"issue_count"`
	Issues      []string  `json:"issues"`
	CleanedKeys []string  `json:"cleaned_keys"`
}
