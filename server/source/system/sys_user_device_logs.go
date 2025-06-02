package system

import (
	"github.com/flipped-aurora/gin-vue-admin/server/global"
	"github.com/flipped-aurora/gin-vue-admin/server/model/system"
	"github.com/pkg/errors"
	"gorm.io/gorm"
)

var SysUserDeviceLogs = new(sysUserDeviceLogs)

type sysUserDeviceLogs struct{}

// TableName 设置表名
func (s *sysUserDeviceLogs) TableName() string {
	return "sys_user_device_logs"
}

// Initialize 初始化数据
func (s *sysUserDeviceLogs) Initialize() error {
	// 自动迁移表结构
	return global.GVA_DB.AutoMigrate(&system.SysUserDeviceLog{})
}

// CheckDataExist 检查数据是否存在
func (s *sysUserDeviceLogs) CheckDataExist() bool {
	if errors.Is(global.GVA_DB.Where("1 = 1").First(&system.SysUserDeviceLog{}).Error, gorm.ErrRecordNotFound) {
		return false
	}
	return true
}
