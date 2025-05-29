// NFC中继管理模块权限配置
export const nfcRelayPermissions = {
  // 模块基础权限
  MODULE_ACCESS: 'nfc_relay:access',
  
  // 仪表盘权限
  DASHBOARD_VIEW: 'nfc_relay:dashboard:view',
  
  // 客户端管理权限
  CLIENT_LIST: 'nfc_relay:client:list',
  CLIENT_VIEW: 'nfc_relay:client:view',
  CLIENT_DISCONNECT: 'nfc_relay:client:disconnect',
  
  // 会话管理权限
  SESSION_LIST: 'nfc_relay:session:list',
  SESSION_VIEW: 'nfc_relay:session:view',
  SESSION_TERMINATE: 'nfc_relay:session:terminate',
  
  // 审计日志权限
  AUDIT_LOG_VIEW: 'nfc_relay:audit:view',
  AUDIT_LOG_EXPORT: 'nfc_relay:audit:export',
  
  // 配置查看权限
  CONFIG_VIEW: 'nfc_relay:config:view'
}

// 权限检查工具函数
export const hasPermission = (permission) => {
  // 这里应该与实际的权限系统集成
  // 目前返回true用于开发测试
  return true
}

// 权限组合检查
export const hasAnyPermission = (permissions) => {
  return permissions.some(permission => hasPermission(permission))
}

export const hasAllPermissions = (permissions) => {
  return permissions.every(permission => hasPermission(permission))
} 