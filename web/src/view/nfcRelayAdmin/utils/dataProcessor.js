// NFC中继管理模块数据处理工具函数

import { 
  CLIENT_ROLE_LABELS, 
  SESSION_STATUS_LABELS, 
  AUDIT_EVENT_TYPE_LABELS 
} from '../constants'

/**
 * 处理客户端列表数据
 * @param {Array} rawData 原始客户端数据
 * @returns {Array} 处理后的客户端数据
 */
export const processClientData = (rawData) => {
  if (!Array.isArray(rawData)) return []
  
  return rawData.map(client => ({
    ...client,
    roleText: CLIENT_ROLE_LABELS[client.role] || client.role,
    statusText: client.is_online ? '在线' : '离线',
    connectedDuration: calculateDuration(client.connected_at)
  }))
}

/**
 * 处理会话列表数据
 * @param {Array} rawData 原始会话数据
 * @returns {Array} 处理后的会话数据
 */
export const processSessionData = (rawData) => {
  if (!Array.isArray(rawData)) return []
  
  return rawData.map(session => ({
    ...session,
    statusText: SESSION_STATUS_LABELS[session.status] || session.status,
    duration: calculateDuration(session.created_at),
    lastActivityDuration: calculateDuration(session.last_activity_at)
  }))
}

/**
 * 处理审计日志数据
 * @param {Array} rawData 原始日志数据
 * @returns {Array} 处理后的日志数据
 */
export const processAuditLogData = (rawData) => {
  if (!Array.isArray(rawData)) return []
  
  return rawData.map(log => ({
    ...log,
    eventTypeText: AUDIT_EVENT_TYPE_LABELS[log.event_type] || log.event_type,
    detailsPreview: extractLogDetailsPreview(log.details),
    severity: getLogSeverity(log.event_type)
  }))
}

/**
 * 计算时间差
 * @param {string} startTime 开始时间
 * @param {string} endTime 结束时间（可选，默认为当前时间）
 * @returns {string} 格式化的时间差
 */
export const calculateDuration = (startTime, endTime = null) => {
  if (!startTime) return '-'
  
  try {
    const start = new Date(startTime)
    const end = endTime ? new Date(endTime) : new Date()
    const diff = end - start
    
    if (diff < 0) return '-'
    
    const minutes = Math.floor(diff / 60000)
    const hours = Math.floor(minutes / 60)
    const days = Math.floor(hours / 24)
    
    if (days > 0) {
      return `${days}天${hours % 24}小时`
    } else if (hours > 0) {
      return `${hours}小时${minutes % 60}分钟`
    } else {
      return `${minutes}分钟`
    }
  } catch (error) {
    return '-'
  }
}

/**
 * 提取日志详情预览
 * @param {Object} details 日志详情对象
 * @returns {string} 预览文本
 */
export const extractLogDetailsPreview = (details) => {
  if (!details || typeof details !== 'object') return '-'
  
  const preview = []
  if (details.apdu_length) preview.push(`长度:${details.apdu_length}`)
  if (details.error_message) preview.push(`错误:${details.error_message}`)
  if (details.reason) preview.push(`原因:${details.reason}`)
  if (details.session_duration) preview.push(`时长:${details.session_duration}`)
  
  return preview.length > 0 ? preview.join(', ') : '详细信息可用'
}

/**
 * 获取日志严重程度
 * @param {string} eventType 事件类型
 * @returns {string} 严重程度等级
 */
export const getLogSeverity = (eventType) => {
  const errorEvents = ['apdu_relayed_failure', 'auth_failure']
  const warningEvents = ['client_disconnected', 'session_terminated']
  
  if (errorEvents.includes(eventType)) return 'error'
  if (warningEvents.includes(eventType)) return 'warning'
  return 'info'
}

/**
 * 生成统计摘要
 * @param {Array} data 数据数组
 * @param {string} type 数据类型 ('clients' | 'sessions' | 'logs')
 * @returns {Object} 统计摘要
 */
export const generateStatsSummary = (data, type) => {
  if (!Array.isArray(data)) return {}
  
  const summary = {
    total: data.length,
    lastUpdate: new Date().toISOString()
  }
  
  switch (type) {
    case 'clients':
      summary.online = data.filter(item => item.is_online).length
      summary.offline = summary.total - summary.online
      summary.byRole = {
        provider: data.filter(item => item.role === 'provider').length,
        receiver: data.filter(item => item.role === 'receiver').length,
        none: data.filter(item => item.role === 'none').length
      }
      break
      
    case 'sessions':
      summary.paired = data.filter(item => item.status === 'paired').length
      summary.waiting = data.filter(item => item.status === 'waiting_for_pairing').length
      break
      
    case 'logs':
      summary.byType = {}
      data.forEach(log => {
        const type = log.event_type
        summary.byType[type] = (summary.byType[type] || 0) + 1
      })
      summary.errors = data.filter(log => getLogSeverity(log.event_type) === 'error').length
      summary.warnings = data.filter(log => getLogSeverity(log.event_type) === 'warning').length
      break
  }
  
  return summary
}

/**
 * 数据验证函数
 * @param {any} data 待验证数据
 * @param {string} type 数据类型
 * @returns {boolean} 验证结果
 */
export const validateData = (data, type) => {
  if (!data) return false
  
  switch (type) {
    case 'client':
      return data.client_id && data.user_id && data.role
      
    case 'session':
      return data.session_id && data.provider_client_id && data.receiver_client_id
      
    case 'auditLog':
      return data.timestamp && data.event_type
      
    default:
      return true
  }
} 