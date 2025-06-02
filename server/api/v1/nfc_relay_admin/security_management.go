package nfc_relay_admin

import (
	"github.com/flipped-aurora/gin-vue-admin/server/global"
	"github.com/flipped-aurora/gin-vue-admin/server/model/common/response"
	"github.com/flipped-aurora/gin-vue-admin/server/model/nfc_relay_admin/request"
	"github.com/flipped-aurora/gin-vue-admin/server/service"
	"github.com/flipped-aurora/gin-vue-admin/server/utils"
	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

type SecurityManagementApi struct{}

// BanClient 封禁客户端
// @Summary 封禁客户端
// @Description 封禁指定的客户端
// @Tags NFC中继安全管理
// @Accept json
// @Produce json
// @Param data body request.ClientBanRequest true "封禁请求"
// @Success 200 {object} response.Response{} "封禁成功"
// @Router /admin/nfc-relay/v1/security/ban-client [post]
func (s *SecurityManagementApi) BanClient(c *gin.Context) {
	var req request.ClientBanRequest
	err := c.ShouldBindJSON(&req)
	if err != nil {
		response.FailWithMessage(err.Error(), c)
		return
	}

	// 获取当前管理员ID
	adminID := utils.GetUserID(c)

	err = service.ServiceGroupApp.NfcRelayAdminServiceGroup.SecurityService.BanClient(&req, adminID)
	if err != nil {
		global.GVA_LOG.Error("封禁客户端失败!", zap.Error(err))
		response.FailWithMessage(err.Error(), c)
		return
	}

	response.OkWithMessage("封禁成功", c)
}

// UnbanClient 解封客户端
// @Summary 解封客户端
// @Description 解封指定的客户端
// @Tags NFC中继安全管理
// @Accept json
// @Produce json
// @Param data body request.ClientUnbanRequest true "解封请求"
// @Success 200 {object} response.Response{} "解封成功"
// @Router /admin/nfc-relay/v1/security/unban-client [post]
func (s *SecurityManagementApi) UnbanClient(c *gin.Context) {
	var req request.ClientUnbanRequest
	err := c.ShouldBindJSON(&req)
	if err != nil {
		response.FailWithMessage(err.Error(), c)
		return
	}

	// 获取当前管理员ID
	adminID := utils.GetUserID(c)

	err = service.ServiceGroupApp.NfcRelayAdminServiceGroup.SecurityService.UnbanClient(&req, adminID)
	if err != nil {
		global.GVA_LOG.Error("解封客户端失败!", zap.Error(err))
		response.FailWithMessage(err.Error(), c)
		return
	}

	response.OkWithMessage("解封成功", c)
}

// GetClientBanList 获取客户端封禁列表
// @Summary 获取客户端封禁列表
// @Description 获取客户端封禁记录列表，支持分页和筛选
// @Tags NFC中继安全管理
// @Accept json
// @Produce json
// @Param data query request.ClientBanListRequest true "查询参数"
// @Success 200 {object} response.Response{data=response.PaginatedClientBanResponse} "获取成功"
// @Router /admin/nfc-relay/v1/security/client-bans [get]
func (s *SecurityManagementApi) GetClientBanList(c *gin.Context) {
	var req request.ClientBanListRequest
	err := c.ShouldBindQuery(&req)
	if err != nil {
		response.FailWithMessage(err.Error(), c)
		return
	}

	list, err := service.ServiceGroupApp.NfcRelayAdminServiceGroup.SecurityService.GetClientBanList(&req)
	if err != nil {
		global.GVA_LOG.Error("获取客户端封禁列表失败!", zap.Error(err))
		response.FailWithMessage("获取失败", c)
		return
	}

	response.OkWithDetailed(list, "获取成功", c)
}

// IsClientBanned 检查客户端是否被封禁
// @Summary 检查客户端是否被封禁
// @Description 检查指定客户端的封禁状态
// @Tags NFC中继安全管理
// @Accept json
// @Produce json
// @Param clientID path string true "客户端ID"
// @Success 200 {object} response.Response{data=map[string]interface{}} "检查成功"
// @Router /admin/nfc-relay/v1/security/client-ban-status/{clientID} [get]
func (s *SecurityManagementApi) IsClientBanned(c *gin.Context) {
	clientID := c.Param("clientID")
	if clientID == "" {
		response.FailWithMessage("客户端ID不能为空", c)
		return
	}

	isBanned, banRecord, err := service.ServiceGroupApp.NfcRelayAdminServiceGroup.SecurityService.IsClientBanned(clientID)
	if err != nil {
		global.GVA_LOG.Error("检查客户端封禁状态失败!", zap.Error(err))
		response.FailWithMessage("检查失败", c)
		return
	}

	result := map[string]interface{}{
		"is_banned": isBanned,
	}

	if isBanned && banRecord != nil {
		result["ban_record"] = map[string]interface{}{
			"ban_reason":     banRecord.BanReason,
			"ban_type":       banRecord.BanType,
			"banned_at":      banRecord.BannedAt,
			"expires_at":     banRecord.ExpiresAt,
			"severity":       banRecord.Severity,
			"remaining_time": banRecord.GetRemainingTime().String(),
		}
	}

	response.OkWithDetailed(result, "检查成功", c)
}

// GetUserSecurityProfile 获取用户安全档案
// @Summary 获取用户安全档案
// @Description 获取指定用户的安全档案信息
// @Tags NFC中继安全管理
// @Accept json
// @Produce json
// @Param userID path string true "用户ID"
// @Success 200 {object} response.Response{data=response.UserSecurityProfileEntry} "获取成功"
// @Router /admin/nfc-relay/v1/security/user-security/{userID} [get]
func (s *SecurityManagementApi) GetUserSecurityProfile(c *gin.Context) {
	userID := c.Param("userID")
	if userID == "" {
		response.FailWithMessage("用户ID不能为空", c)
		return
	}

	profile, err := service.ServiceGroupApp.NfcRelayAdminServiceGroup.SecurityService.GetUserSecurityProfile(userID)
	if err != nil {
		global.GVA_LOG.Error("获取用户安全档案失败!", zap.Error(err))
		response.FailWithMessage("获取失败", c)
		return
	}

	response.OkWithDetailed(profile, "获取成功", c)
}

// GetUserSecurityProfileList 获取用户安全档案列表
// @Summary 获取用户安全档案列表
// @Description 获取用户安全档案列表，支持分页和筛选
// @Tags NFC中继安全管理
// @Accept json
// @Produce json
// @Param data query request.UserSecurityProfileListRequest true "查询参数"
// @Success 200 {object} response.Response{data=response.PaginatedUserSecurityProfileResponse} "获取成功"
// @Router /admin/nfc-relay/v1/security/user-security [get]
func (s *SecurityManagementApi) GetUserSecurityProfileList(c *gin.Context) {
	var req request.UserSecurityProfileListRequest
	err := c.ShouldBindQuery(&req)
	if err != nil {
		response.FailWithMessage(err.Error(), c)
		return
	}

	list, err := service.ServiceGroupApp.NfcRelayAdminServiceGroup.SecurityService.GetUserSecurityProfileList(&req)
	if err != nil {
		global.GVA_LOG.Error("获取用户安全档案列表失败!", zap.Error(err))
		response.FailWithMessage("获取失败", c)
		return
	}

	response.OkWithDetailed(list, "获取成功", c)
}

// UpdateUserSecurityProfile 更新用户安全档案
// @Summary 更新用户安全档案
// @Description 更新指定用户的安全档案信息
// @Tags NFC中继安全管理
// @Accept json
// @Produce json
// @Param data body request.UpdateUserSecurityRequest true "更新请求"
// @Success 200 {object} response.Response{} "更新成功"
// @Router /admin/nfc-relay/v1/security/user-security [put]
func (s *SecurityManagementApi) UpdateUserSecurityProfile(c *gin.Context) {
	var req request.UpdateUserSecurityRequest
	err := c.ShouldBindJSON(&req)
	if err != nil {
		response.FailWithMessage(err.Error(), c)
		return
	}

	err = service.ServiceGroupApp.NfcRelayAdminServiceGroup.SecurityService.UpdateUserSecurityProfile(&req)
	if err != nil {
		global.GVA_LOG.Error("更新用户安全档案失败!", zap.Error(err))
		response.FailWithMessage("更新失败", c)
		return
	}

	response.OkWithMessage("更新成功", c)
}

// LockUserAccount 锁定用户账户
// @Summary 锁定用户账户
// @Description 锁定指定用户的账户
// @Tags NFC中继安全管理
// @Accept json
// @Produce json
// @Param data body request.LockUserAccountRequest true "锁定请求"
// @Success 200 {object} response.Response{} "锁定成功"
// @Router /admin/nfc-relay/v1/security/lock-user [post]
func (s *SecurityManagementApi) LockUserAccount(c *gin.Context) {
	var req request.LockUserAccountRequest
	err := c.ShouldBindJSON(&req)
	if err != nil {
		response.FailWithMessage(err.Error(), c)
		return
	}

	err = service.ServiceGroupApp.NfcRelayAdminServiceGroup.SecurityService.LockUserAccount(&req)
	if err != nil {
		global.GVA_LOG.Error("锁定用户账户失败!", zap.Error(err))
		response.FailWithMessage(err.Error(), c)
		return
	}

	response.OkWithMessage("锁定成功", c)
}

// UnlockUserAccount 解锁用户账户
// @Summary 解锁用户账户
// @Description 解锁指定用户的账户
// @Tags NFC中继安全管理
// @Accept json
// @Produce json
// @Param data body request.UnlockUserAccountRequest true "解锁请求"
// @Success 200 {object} response.Response{} "解锁成功"
// @Router /admin/nfc-relay/v1/security/unlock-user [post]
func (s *SecurityManagementApi) UnlockUserAccount(c *gin.Context) {
	var req request.UnlockUserAccountRequest
	err := c.ShouldBindJSON(&req)
	if err != nil {
		response.FailWithMessage(err.Error(), c)
		return
	}

	err = service.ServiceGroupApp.NfcRelayAdminServiceGroup.SecurityService.UnlockUserAccount(&req)
	if err != nil {
		global.GVA_LOG.Error("解锁用户账户失败!", zap.Error(err))
		response.FailWithMessage(err.Error(), c)
		return
	}

	response.OkWithMessage("解锁成功", c)
}

// GetSecuritySummary 获取安全摘要
// @Summary 获取安全摘要
// @Description 获取系统安全状态摘要信息
// @Tags NFC中继安全管理
// @Accept json
// @Produce json
// @Success 200 {object} response.Response{data=response.SecuritySummaryResponse} "获取成功"
// @Router /admin/nfc-relay/v1/security/summary [get]
func (s *SecurityManagementApi) GetSecuritySummary(c *gin.Context) {
	summary, err := service.ServiceGroupApp.NfcRelayAdminServiceGroup.SecurityService.GetSecuritySummary()
	if err != nil {
		global.GVA_LOG.Error("获取安全摘要失败!", zap.Error(err))
		response.FailWithMessage("获取失败", c)
		return
	}

	response.OkWithDetailed(summary, "获取成功", c)
}

// CleanupExpiredData 清理过期数据
// @Summary 清理过期数据
// @Description 清理过期的封禁记录和账户锁定
// @Tags NFC中继安全管理
// @Accept json
// @Produce json
// @Success 200 {object} response.Response{} "清理成功"
// @Router /admin/nfc-relay/v1/security/cleanup [post]
func (s *SecurityManagementApi) CleanupExpiredData(c *gin.Context) {
	// 清理过期封禁
	err := service.ServiceGroupApp.NfcRelayAdminServiceGroup.SecurityService.CleanupExpiredBans()
	if err != nil {
		global.GVA_LOG.Error("清理过期封禁记录失败!", zap.Error(err))
		response.FailWithMessage("清理失败", c)
		return
	}

	// 清理过期锁定
	err = service.ServiceGroupApp.NfcRelayAdminServiceGroup.SecurityService.CleanupExpiredLocks()
	if err != nil {
		global.GVA_LOG.Error("清理过期锁定记录失败!", zap.Error(err))
		response.FailWithMessage("清理失败", c)
		return
	}

	response.OkWithMessage("清理成功", c)
}
