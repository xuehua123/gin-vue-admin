// NFC中继管理模块路由辅助工具

/**
 * NFC中继管理模块的路由名称常量
 */
export const ROUTE_NAMES = {
  // 父级布局
  LAYOUT: 'nfcRelayAdminLayout',
  
  // 子页面
  DASHBOARD: 'nfcRelayDashboard',
  CLIENT_MANAGEMENT: 'nfcRelayClientManagement', 
  SESSION_MANAGEMENT: 'nfcRelaySessionManagement',
  AUDIT_LOGS: 'nfcRelayAuditLogs',
  CONFIGURATION: 'nfcRelayConfiguration'
}

/**
 * 路由跳转辅助函数
 * @param {Object} router Vue Router实例
 */
export const createRouteHelper = (router) => {
  return {
    // 跳转到仪表盘
    toDashboard() {
      router.push({ name: ROUTE_NAMES.DASHBOARD })
    },
    
    // 跳转到连接管理
    toClientManagement(clientID = null) {
      const route = { name: ROUTE_NAMES.CLIENT_MANAGEMENT }
      if (clientID) {
        route.query = { clientID }
      }
      router.push(route)
    },
    
    // 跳转到会话管理
    toSessionManagement(sessionID = null) {
      const route = { name: ROUTE_NAMES.SESSION_MANAGEMENT }
      if (sessionID) {
        route.query = { sessionID }
      }
      router.push(route)
    },
    
    // 跳转到审计日志
    toAuditLogs(filters = {}) {
      const route = { name: ROUTE_NAMES.AUDIT_LOGS }
      if (Object.keys(filters).length > 0) {
        route.query = filters
      }
      router.push(route)
    },
    
    // 跳转到系统配置
    toConfiguration() {
      router.push({ name: ROUTE_NAMES.CONFIGURATION })
    }
  }
}

/**
 * 检查当前路由是否在NFC中继管理模块内
 * @param {Object} route 当前路由对象
 * @returns {boolean}
 */
export const isNfcRelayAdminRoute = (route) => {
  return Object.values(ROUTE_NAMES).includes(route.name)
}

/**
 * 获取页面标题
 * @param {string} routeName 路由名称
 * @returns {string}
 */
export const getPageTitle = (routeName) => {
  const titleMap = {
    [ROUTE_NAMES.DASHBOARD]: '概览仪表盘',
    [ROUTE_NAMES.CLIENT_MANAGEMENT]: '连接管理',
    [ROUTE_NAMES.SESSION_MANAGEMENT]: '会话管理', 
    [ROUTE_NAMES.AUDIT_LOGS]: '审计日志',
    [ROUTE_NAMES.CONFIGURATION]: '系统配置'
  }
  
  return titleMap[routeName] || 'NFC中继管理'
} 