/**
 * NFC中继管理 API接口
 * 包含仪表盘、连接管理、会话管理、审计日志等功能
 * 
 * @description 查询参数设计规范：
 * 
 * 1. 通用查询参数：
 *    - page: 页码 (默认1)
 *    - pageSize: 每页大小 (默认20)
 *    - sortBy: 排序字段
 *    - sortOrder: 排序方向 ('asc' | 'desc')
 *    - startTime: 开始时间 (ISO8601格式)
 *    - endTime: 结束时间 (ISO8601格式)
 *    - timeRange: 预设时间范围 ('1h' | '1d' | '7d' | '30d' | 'custom')
 *    - keyword: 关键词搜索
 *    - status: 状态过滤 (数组)
 * 
 * 2. 连接管理查询参数：
 *    - clientIds: 客户端ID列表
 *    - userIds: 用户ID列表
 *    - roles: 角色过滤 ('provider' | 'receiver' | 'none')
 *    - ipRanges: IP地址范围
 *    - deviceTypes: 设备类型
 *    - connectionStatus: 连接状态 ('online' | 'offline' | 'idle')
 *    - geolocation: 地理位置 {country, region, city}
 *    - tags: 标签过滤
 *    - lastActivityBefore: 最后活动时间之前
 *    - lastActivityAfter: 最后活动时间之后
 *    - onlineDuration: 在线时长范围 {min, max}
 *    - sessionCount: 会话数量范围 {min, max}
 *    - dataTransferred: 数据传输量范围 {min, max}
 * 
 * 3. 会话管理查询参数：
 *    - sessionIds: 会话ID列表
 *    - participantIds: 参与者ID列表
 *    - sessionTypes: 会话类型
 *    - sessionStates: 会话状态 ('active' | 'paused' | 'completed' | 'failed')
 *    - durationRange: 持续时间范围 {min, max}
 *    - apduCountRange: APDU数量范围 {min, max}
 *    - errorThreshold: 错误阈值
 *    - performanceMetrics: 性能指标 {minThroughput, maxLatency}
 *    - hasRecording: 是否有录制
 *    - recordingSize: 录制大小范围 {min, max}
 *    - participants: 参与者数量范围 {min, max}
 * 
 * 4. 审计日志查询参数：
 *    - eventTypes: 事件类型列表
 *    - severityLevels: 严重级别 ('debug' | 'info' | 'warn' | 'error' | 'critical')
 *    - sourceIps: 源IP地址列表
 *    - userAgents: 用户代理列表
 *    - sessionCorrelationIds: 会话关联ID列表
 *    - resourceTypes: 资源类型
 *    - actionTypes: 操作类型 ('create' | 'read' | 'update' | 'delete' | 'execute')
 *    - resultCodes: 结果代码列表
 *    - hasAttachments: 是否有附件
 *    - contextFilters: 上下文过滤器 (JSON对象)
 *    - messagePattern: 消息模式匹配 (正则表达式)
 *    - correlationId: 关联ID
 *    - requestId: 请求ID
 */

import service from '@/utils/request'
import { API_CONFIG } from '@/view/nfcRelayAdmin/constants.js'

const API_PREFIX = API_CONFIG.BASE_URL

// ===========================================
// 仪表盘增强版API
// ===========================================

/**
 * 获取增强版仪表盘统计数据
 * @param {Object} params 查询参数（可选时间范围等）
 * @returns {Promise} API响应
 */
export const getDashboardStatsEnhanced = (params) => {
  return service({
    url: `${API_PREFIX}/dashboard-stats-enhanced`,
    method: 'get',
    params
  })
}

/**
 * 获取性能指标数据
 * @param {Object} params 查询参数
 * @returns {Promise} API响应
 */
export const getPerformanceMetrics = (params) => {
  return service({
    url: `${API_PREFIX}/performance-metrics`,
    method: 'get',
    params
  })
}

/**
 * 获取地理分布数据
 * @returns {Promise} API响应
 */
export const getGeographicDistribution = () => {
  return service({
    url: `${API_PREFIX}/geographic-distribution`,
    method: 'get'
  })
}

/**
 * 获取警报列表
 * @param {Object} params 查询参数
 * @returns {Promise} API响应
 */
export const getAlerts = (params) => {
  return service({
    url: `${API_PREFIX}/alerts`,
    method: 'get',
    params
  })
}

/**
 * 确认警报
 * @param {string} alertId 警报ID
 * @returns {Promise} API响应
 */
export const acknowledgeAlert = (alertId) => {
  return service({
    url: `${API_PREFIX}/alerts/${alertId}/acknowledge`,
    method: 'post'
  })
}

/**
 * 导出仪表盘数据
 * @param {Object} exportRequest 导出请求参数
 * @returns {Promise} API响应
 */
export const exportDashboardData = (exportRequest) => {
  return service({
    url: `${API_PREFIX}/export`,
    method: 'post',
    data: exportRequest
  })
}

/**
 * 获取对比分析数据
 * @param {Object} params 查询参数
 * @returns {Promise} API响应
 */
export const getComparisonData = (params) => {
  return service({
    url: `${API_PREFIX}/comparison`,
    method: 'get',
    params
  })
}

// ===========================================
// 连接管理API
// ===========================================

/**
 * 获取客户端连接列表
 * @param {Object} params 查询参数
 * @returns {Promise} API响应
 */
export const getClientsList = (params) => {
  return service({
    url: `${API_PREFIX}/clients`,
    method: 'get',
    params
  })
}

/**
 * 获取客户端详情
 * @param {string} clientId 客户端ID
 * @returns {Promise} API响应
 */
export const getClientDetails = (clientId) => {
  return service({
    url: `${API_PREFIX}/clients/${clientId}/details`,
    method: 'get'
  })
}

/**
 * 强制断开客户端连接
 * @param {string} clientId 客户端ID
 * @returns {Promise} API响应
 */
export const disconnectClient = (clientId) => {
  return service({
    url: `${API_PREFIX}/clients/${clientId}/disconnect`,
    method: 'post'
  })
}

/**
 * 批量断开客户端连接
 * @param {Array} clientIds 客户端ID列表
 * @returns {Promise} API响应
 */
export const batchDisconnectClients = (clientIds) => {
  return service({
    url: `${API_PREFIX}/clients/batch-disconnect`,
    method: 'post',
    data: { client_ids: clientIds }
  })
}

/**
 * 获取客户端连接历史
 * @param {string} clientId 客户端ID
 * @param {Object} params 查询参数
 * @returns {Promise} API响应
 */
export const getClientHistory = (clientId, params) => {
  return service({
    url: `${API_PREFIX}/clients/${clientId}/history`,
    method: 'get',
    params
  })
}

/**
 * 设置客户端黑白名单
 * @param {Object} data 黑白名单数据
 * @returns {Promise} API响应
 */
export const setClientAccess = (data) => {
  return service({
    url: `${API_PREFIX}/clients/access`,
    method: 'post',
    data
  })
}

// ===========================================
// 会话管理API
// ===========================================

/**
 * 获取会话列表
 * @param {Object} params 查询参数
 * @returns {Promise} API响应
 */
export const getSessionsList = (params) => {
  return service({
    url: `${API_PREFIX}/sessions`,
    method: 'get',
    params
  })
}

/**
 * 获取会话详情
 * @param {string} sessionId 会话ID
 * @returns {Promise} API响应
 */
export const getSessionDetails = (sessionId) => {
  return service({
    url: `${API_PREFIX}/sessions/${sessionId}/details`,
    method: 'get'
  })
}

/**
 * 强制终止会话
 * @param {string} sessionId 会话ID
 * @returns {Promise} API响应
 */
export const terminateSession = (sessionId) => {
  return service({
    url: `${API_PREFIX}/sessions/${sessionId}/terminate`,
    method: 'post'
  })
}

/**
 * 批量终止会话
 * @param {Array} sessionIds 会话ID列表
 * @returns {Promise} API响应
 */
export const batchTerminateSessions = (sessionIds) => {
  return service({
    url: `${API_PREFIX}/sessions/batch-terminate`,
    method: 'post',
    data: { session_ids: sessionIds }
  })
}

/**
 * 获取会话APDU日志
 * @param {string} sessionId 会话ID
 * @param {Object} params 查询参数
 * @returns {Promise} API响应
 */
export const getSessionApduLogs = (sessionId, params) => {
  return service({
    url: `${API_PREFIX}/sessions/${sessionId}/apdu-logs`,
    method: 'get',
    params
  })
}

/**
 * 获取会话性能统计
 * @param {string} sessionId 会话ID
 * @returns {Promise} API响应
 */
export const getSessionPerformance = (sessionId) => {
  return service({
    url: `${API_PREFIX}/sessions/${sessionId}/performance`,
    method: 'get'
  })
}

/**
 * 开始会话录制
 * @param {string} sessionId 会话ID
 * @returns {Promise} API响应
 */
export const startSessionRecording = (sessionId) => {
  return service({
    url: `${API_PREFIX}/sessions/${sessionId}/recording/start`,
    method: 'post'
  })
}

/**
 * 停止会话录制
 * @param {string} sessionId 会话ID
 * @returns {Promise} API响应
 */
export const stopSessionRecording = (sessionId) => {
  return service({
    url: `${API_PREFIX}/sessions/${sessionId}/recording/stop`,
    method: 'post'
  })
}

/**
 * 获取会话录制列表
 * @param {Object} params 查询参数
 * @returns {Promise} API响应
 */
export const getSessionRecordings = (params) => {
  return service({
    url: `${API_PREFIX}/sessions/recordings`,
    method: 'get',
    params
  })
}

/**
 * 回放会话录制
 * @param {string} recordingId 录制ID
 * @returns {Promise} API响应
 */
export const playbackRecording = (recordingId) => {
  return service({
    url: `${API_PREFIX}/sessions/recordings/${recordingId}/playback`,
    method: 'post'
  })
}

// ===========================================
// 审计日志API (文件系统)
// ===========================================

/**
 * 获取审计日志列表
 * @param {Object} params 查询参数
 * @returns {Promise} API响应
 */
export const getAuditLogs = (params) => {
  return service({
    url: `${API_PREFIX}/audit-logs`,
    method: 'get',
    params
  })
}

/**
 * 导出审计日志
 * @param {Object} params 导出参数
 * @returns {Promise} API响应
 */
export const exportAuditLogs = (params) => {
  return service({
    url: `${API_PREFIX}/audit-logs/export`,
    method: 'post',
    data: params,
    responseType: 'blob'
  })
}

/**
 * 批量导出审计日志
 * @param {Object} params 批量导出参数
 * @returns {Promise} API响应
 */
export const batchExportAuditLogs = (params) => {
  return service({
    url: `${API_PREFIX}/audit-logs/batch-export`,
    method: 'post',
    data: params,
    responseType: 'blob'
  })
}

// ===========================================
// 数据库审计日志API (新增)
// ===========================================

/**
 * 创建审计日志
 * @param {Object} logData 日志数据
 * @returns {Promise} API响应
 */
export const createAuditLogDb = (logData) => {
  return service({
    url: `${API_PREFIX}/audit-logs-db`,
    method: 'post',
    data: logData
  })
}

/**
 * 获取数据库审计日志列表
 * @param {Object} params 查询参数
 * @returns {Promise} API响应
 */
export const getAuditLogsDb = (params) => {
  return service({
    url: `${API_PREFIX}/audit-logs-db`,
    method: 'get',
    params
  })
}

/**
 * 获取数据库审计日志统计
 * @param {Object} params 查询参数
 * @returns {Promise} API响应
 */
export const getAuditLogsDbStats = (params) => {
  return service({
    url: `${API_PREFIX}/audit-logs-db/stats`,
    method: 'get',
    params
  })
}

/**
 * 批量创建审计日志
 * @param {Array} logDataList 日志数据列表
 * @returns {Promise} API响应
 */
export const batchCreateAuditLogsDb = (logDataList) => {
  return service({
    url: `${API_PREFIX}/audit-logs-db/batch`,
    method: 'post',
    data: logDataList
  })
}

/**
 * 清理过期审计日志
 * @param {Object} params 清理参数
 * @returns {Promise} API响应
 */
export const cleanupAuditLogsDb = (params) => {
  return service({
    url: `${API_PREFIX}/audit-logs-db/cleanup`,
    method: 'delete',
    params
  })
}

// ===========================================
// 安全管理API (新增)
// ===========================================

/**
 * 封禁客户端
 * @param {Object} banData 封禁数据
 * @returns {Promise} API响应
 */
export const banClient = (banData) => {
  return service({
    url: `${API_PREFIX}/security/ban-client`,
    method: 'post',
    data: banData
  })
}

/**
 * 解封客户端
 * @param {Object} unbanData 解封数据
 * @returns {Promise} API响应
 */
export const unbanClient = (unbanData) => {
  return service({
    url: `${API_PREFIX}/security/unban-client`,
    method: 'post',
    data: unbanData
  })
}

/**
 * 获取客户端封禁列表
 * @param {Object} params 查询参数
 * @returns {Promise} API响应
 */
export const getClientBans = (params) => {
  return service({
    url: `${API_PREFIX}/security/client-bans`,
    method: 'get',
    params
  })
}

/**
 * 获取客户端封禁状态
 * @param {string} clientId 客户端ID
 * @returns {Promise} API响应
 */
export const getClientBanStatus = (clientId) => {
  return service({
    url: `${API_PREFIX}/security/client-ban-status/${clientId}`,
    method: 'get'
  })
}

/**
 * 获取用户安全档案
 * @param {string} userId 用户ID
 * @returns {Promise} API响应
 */
export const getUserSecurityProfile = (userId) => {
  return service({
    url: `${API_PREFIX}/security/user-security/${userId}`,
    method: 'get'
  })
}

/**
 * 获取用户安全档案列表
 * @param {Object} params 查询参数
 * @returns {Promise} API响应
 */
export const getUserSecurityProfiles = (params) => {
  return service({
    url: `${API_PREFIX}/security/user-security`,
    method: 'get',
    params
  })
}

/**
 * 更新用户安全档案
 * @param {Object} profileData 档案数据
 * @returns {Promise} API响应
 */
export const updateUserSecurityProfile = (profileData) => {
  return service({
    url: `${API_PREFIX}/security/user-security`,
    method: 'put',
    data: profileData
  })
}

/**
 * 锁定用户账户
 * @param {Object} lockData 锁定数据
 * @returns {Promise} API响应
 */
export const lockUser = (lockData) => {
  return service({
    url: `${API_PREFIX}/security/lock-user`,
    method: 'post',
    data: lockData
  })
}

/**
 * 解锁用户账户
 * @param {Object} unlockData 解锁数据
 * @returns {Promise} API响应
 */
export const unlockUser = (unlockData) => {
  return service({
    url: `${API_PREFIX}/security/unlock-user`,
    method: 'post',
    data: unlockData
  })
}

/**
 * 获取安全摘要
 * @param {Object} params 查询参数
 * @returns {Promise} API响应
 */
export const getSecuritySummary = (params) => {
  return service({
    url: `${API_PREFIX}/security/summary`,
    method: 'get',
    params
  })
}

/**
 * 清理过期安全数据
 * @param {Object} cleanupData 清理参数
 * @returns {Promise} API响应
 */
export const securityCleanup = (cleanupData) => {
  return service({
    url: `${API_PREFIX}/security/cleanup`,
    method: 'post',
    data: cleanupData
  })
}

// ===========================================
// 系统配置API
// ===========================================

/**
 * 获取系统配置
 * @returns {Promise} API响应
 */
export const getSystemConfig = () => {
  return service({
    url: `${API_PREFIX}/config`,
    method: 'get'
  })
}

/**
 * 更新系统配置
 * @param {Object} config 配置数据
 * @returns {Promise} API响应
 */
export const updateSystemConfig = (config) => {
  return service({
    url: `${API_PREFIX}/config`,
    method: 'put',
    data: config
  })
}

/**
 * 验证配置
 * @param {Object} config 配置数据
 * @returns {Promise} API响应
 */
export const validateSystemConfig = (config) => {
  return service({
    url: `${API_PREFIX}/config/validate`,
    method: 'post',
    data: config
  })
}

/**
 * 获取配置模板
 * @returns {Promise} API响应
 */
export const getConfigTemplates = () => {
  return service({
    url: `${API_PREFIX}/config/templates`,
    method: 'get'
  })
}

/**
 * 创建配置模板
 * @param {Object} template 模板数据
 * @returns {Promise} API响应
 */
export const createConfigTemplate = (template) => {
  return service({
    url: `${API_PREFIX}/config/templates`,
    method: 'post',
    data: template
  })
}

/**
 * 应用配置模板
 * @param {string} templateId 模板ID
 * @returns {Promise} API响应
 */
export const applyConfigTemplate = (templateId) => {
  return service({
    url: `${API_PREFIX}/config/templates/${templateId}/apply`,
    method: 'post'
  })
}

/**
 * 获取配置版本历史
 * @param {Object} params 查询参数
 * @returns {Promise} API响应
 */
export const getConfigVersions = (params) => {
  return service({
    url: `${API_PREFIX}/config/versions`,
    method: 'get',
    params
  })
}

/**
 * 回滚配置版本
 * @param {string} versionId 版本ID
 * @returns {Promise} API响应
 */
export const rollbackConfig = (versionId) => {
  return service({
    url: `${API_PREFIX}/config/versions/${versionId}/rollback`,
    method: 'post'
  })
}

/**
 * 热重载配置
 * @returns {Promise} API响应
 */
export const hotReloadConfig = () => {
  return service({
    url: `${API_PREFIX}/config/hot-reload`,
    method: 'post'
  })
}

// ===========================================
// 系统管理API
// ===========================================

/**
 * 获取系统状态
 * @returns {Promise} API响应
 */
export const getSystemStatus = () => {
  return service({
    url: `${API_PREFIX}/status`,
    method: 'get'
  })
}

/**
 * 重启服务
 * @returns {Promise} API响应
 */
export const restartService = () => {
  return service({
    url: `${API_PREFIX}/system/restart`,
    method: 'post'
  })
}

/**
 * 清理缓存
 * @returns {Promise} API响应
 */
export const clearCache = () => {
  return service({
    url: `${API_PREFIX}/system/cache/clear`,
    method: 'post'
  })
}

/**
 * 获取系统日志
 * @param {Object} params 查询参数
 * @returns {Promise} API响应
 */
export const getSystemLogs = (params) => {
  return service({
    url: `${API_PREFIX}/system/logs`,
    method: 'get',
    params
  })
}

/**
 * 下载系统日志
 * @param {Object} params 查询参数
 * @returns {Promise} API响应
 */
export const downloadSystemLogs = (params) => {
  return service({
    url: `${API_PREFIX}/system/logs/download`,
    method: 'get',
    params,
    responseType: 'blob'
  })
}

/**
 * 健康检查
 * @returns {Promise} API响应
 */
export const getHealthCheck = () => {
  return service({
    url: `${API_PREFIX}/system/health`,
    method: 'get'
  })
}

/**
 * 运行系统诊断
 * @returns {Promise} API响应
 */
export const runSystemDiagnostics = () => {
  return service({
    url: `${API_PREFIX}/system/diagnostics`,
    method: 'post'
  })
}

// ===========================================
// 监控与警报API
// ===========================================

/**
 * 获取监控指标
 * @param {Object} params 查询参数
 * @returns {Promise} API响应
 */
export const getMonitoringMetrics = (params) => {
  return service({
    url: `${API_PREFIX}/monitoring/metrics`,
    method: 'get',
    params
  })
}

/**
 * 创建监控规则
 * @param {Object} rule 规则数据
 * @returns {Promise} API响应
 */
export const createMonitoringRule = (rule) => {
  return service({
    url: `${API_PREFIX}/monitoring/rules`,
    method: 'post',
    data: rule
  })
}

/**
 * 获取监控规则列表
 * @returns {Promise} API响应
 */
export const getMonitoringRules = () => {
  return service({
    url: `${API_PREFIX}/monitoring/rules`,
    method: 'get'
  })
}

/**
 * 更新监控规则
 * @param {string} ruleId 规则ID
 * @param {Object} rule 规则数据
 * @returns {Promise} API响应
 */
export const updateMonitoringRule = (ruleId, rule) => {
  return service({
    url: `${API_PREFIX}/monitoring/rules/${ruleId}`,
    method: 'put',
    data: rule
  })
}

/**
 * 删除监控规则
 * @param {string} ruleId 规则ID
 * @returns {Promise} API响应
 */
export const deleteMonitoringRule = (ruleId) => {
  return service({
    url: `${API_PREFIX}/monitoring/rules/${ruleId}`,
    method: 'delete'
  })
}

/**
 * 测试警报规则
 * @param {Object} rule 规则数据
 * @returns {Promise} API响应
 */
export const testAlertRule = (rule) => {
  return service({
    url: `${API_PREFIX}/monitoring/rules/test`,
    method: 'post',
    data: rule
  })
}

// ===========================================
// 备份与恢复API
// ===========================================

/**
 * 创建备份
 * @param {Object} backupRequest 备份请求参数
 * @returns {Promise} API响应
 */
export const createBackup = (backupRequest) => {
  return service({
    url: `${API_PREFIX}/backup/create`,
    method: 'post',
    data: backupRequest
  })
}

/**
 * 获取备份列表
 * @param {Object} params 查询参数
 * @returns {Promise} API响应
 */
export const getBackupList = (params) => {
  return service({
    url: `${API_PREFIX}/backup/list`,
    method: 'get',
    params
  })
}

/**
 * 恢复备份
 * @param {string} backupId 备份ID
 * @returns {Promise} API响应
 */
export const restoreBackup = (backupId) => {
  return service({
    url: `${API_PREFIX}/backup/${backupId}/restore`,
    method: 'post'
  })
}

/**
 * 删除备份
 * @param {string} backupId 备份ID
 * @returns {Promise} API响应
 */
export const deleteBackup = (backupId) => {
  return service({
    url: `${API_PREFIX}/backup/${backupId}`,
    method: 'delete'
  })
}

/**
 * 下载备份
 * @param {string} backupId 备份ID
 * @returns {Promise} API响应
 */
export const downloadBackup = (backupId) => {
  return service({
    url: `${API_PREFIX}/backup/${backupId}/download`,
    method: 'get',
    responseType: 'blob'
  })
}

// ===========================================
// 实时监控API
// ===========================================

/**
 * 获取实时统计数据
 */
export const getRealTimeStats = () => {
  return service({
    url: `${API_PREFIX}/realtime/stats`,
    method: 'get'
  })
}

/**
 * 获取实时事件流
 * @param {Object} params - 查询参数
 * @param {number} params.limit - 限制数量
 * @param {string} params.since - 从什么时间开始
 */
export const getRealTimeEvents = (params) => {
  return service({
    url: `${API_PREFIX}/realtime/events`,
    method: 'get',
    params
  })
}

/**
 * 获取系统指标
 */
export const getSystemMetrics = () => {
  return service({
    url: `${API_PREFIX}/metrics`,
    method: 'get'
  })
}

/**
 * 获取实时数据 (WebSocket)
 * @returns {Promise} API响应
 */
export const getRealTimeData = () => {
  return service({
    url: `${API_PREFIX}/realtime`,
    method: 'get'
  })
} 