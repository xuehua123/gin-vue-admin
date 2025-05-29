/**
 * 数据格式化工具函数
 * 用于统一处理各种数据的显示格式
 */

import dayjs from 'dayjs'
import 'dayjs/locale/zh-cn'
import relativeTime from 'dayjs/plugin/relativeTime'
import duration from 'dayjs/plugin/duration'

dayjs.extend(relativeTime)
dayjs.extend(duration)
dayjs.locale('zh-cn')

/**
 * 格式化日期时间
 * @param {string|Date|number} date - 日期
 * @param {string} format - 格式化模板
 * @returns {string} 格式化后的日期字符串
 */
export const formatDateTime = (date, format = 'YYYY-MM-DD HH:mm:ss') => {
  if (!date) return '-'
  return dayjs(date).format(format)
}

/**
 * 格式化相对时间
 * @param {string|Date|number} date - 日期
 * @returns {string} 相对时间字符串
 */
export const formatRelativeTime = (date) => {
  if (!date) return '-'
  return dayjs(date).fromNow()
}

/**
 * 格式化时长
 * @param {number} seconds - 秒数
 * @returns {string} 格式化后的时长
 */
export const formatDuration = (seconds) => {
  if (!seconds || seconds < 0) return '-'
  
  const duration = dayjs.duration(seconds, 'seconds')
  const days = duration.days()
  const hours = duration.hours()
  const minutes = duration.minutes()
  const secs = duration.seconds()
  
  if (days > 0) {
    return `${days}天 ${hours}小时`
  } else if (hours > 0) {
    return `${hours}小时 ${minutes}分钟`
  } else if (minutes > 0) {
    return `${minutes}分钟 ${secs}秒`
  } else {
    return `${secs}秒`
  }
}

/**
 * 格式化数字
 * @param {number} value - 数值
 * @param {object} options - 格式化选项
 * @returns {string} 格式化后的数字
 */
export const formatNumber = (value, options = {}) => {
  if (value === null || value === undefined || isNaN(value)) return '-'
  
  const {
    decimals = 0,
    thousandsSeparator = ',',
    decimalSeparator = '.',
    prefix = '',
    suffix = ''
  } = options
  
  const num = Number(value).toFixed(decimals)
  const parts = num.split('.')
  parts[0] = parts[0].replace(/\B(?=(\d{3})+(?!\d))/g, thousandsSeparator)
  
  return prefix + parts.join(decimalSeparator) + suffix
}

/**
 * 格式化文件大小
 * @param {number} bytes - 字节数
 * @param {number} decimals - 小数位数
 * @returns {string} 格式化后的文件大小
 */
export const formatFileSize = (bytes, decimals = 2) => {
  if (!bytes || bytes === 0) return '0 B'
  
  const k = 1024
  const dm = decimals < 0 ? 0 : decimals
  const sizes = ['B', 'KB', 'MB', 'GB', 'TB', 'PB']
  
  const i = Math.floor(Math.log(bytes) / Math.log(k))
  
  return parseFloat((bytes / Math.pow(k, i)).toFixed(dm)) + ' ' + sizes[i]
}

/**
 * 格式化百分比
 * @param {number} value - 数值 (0-1 或 0-100)
 * @param {boolean} isDecimal - 是否为小数形式
 * @param {number} decimals - 小数位数
 * @returns {string} 格式化后的百分比
 */
export const formatPercentage = (value, isDecimal = true, decimals = 1) => {
  if (value === null || value === undefined || isNaN(value)) return '-'
  
  const percentage = isDecimal ? value * 100 : value
  return percentage.toFixed(decimals) + '%'
}

/**
 * 格式化IP地址
 * @param {string} ip - IP地址
 * @returns {string} 格式化后的IP地址
 */
export const formatIPAddress = (ip) => {
  if (!ip) return '-'
  
  // 提取IPv4地址（去除端口号）
  const ipv4Match = ip.match(/(\d+\.\d+\.\d+\.\d+)/)
  if (ipv4Match) {
    return ipv4Match[1]
  }
  
  return ip
}

/**
 * 格式化客户端ID（缩短显示）
 * @param {string} clientId - 客户端ID
 * @param {number} length - 显示长度
 * @returns {string} 格式化后的客户端ID
 */
export const formatClientId = (clientId, length = 8) => {
  if (!clientId) return '-'
  
  if (clientId.length <= length) return clientId
  
  return clientId.substring(0, length) + '...'
}

/**
 * 格式化会话ID（缩短显示）
 * @param {string} sessionId - 会话ID
 * @param {number} length - 显示长度
 * @returns {string} 格式化后的会话ID
 */
export const formatSessionId = (sessionId, length = 8) => {
  return formatClientId(sessionId, length)
}

/**
 * 格式化状态文本
 * @param {string} status - 状态值
 * @param {object} statusMap - 状态映射
 * @returns {string} 格式化后的状态文本
 */
export const formatStatus = (status, statusMap = {}) => {
  if (!status) return '-'
  
  return statusMap[status] || status
}

/**
 * 格式化APDU数据长度
 * @param {number} length - 数据长度
 * @returns {string} 格式化后的长度描述
 */
export const formatApduLength = (length) => {
  if (!length || length === 0) return '-'
  
  if (length < 1024) {
    return `${length} bytes`
  } else {
    return `${(length / 1024).toFixed(1)} KB`
  }
}

/**
 * 格式化错误率
 * @param {number} errorCount - 错误数量
 * @param {number} totalCount - 总数量
 * @returns {string} 格式化后的错误率
 */
export const formatErrorRate = (errorCount, totalCount) => {
  if (!totalCount || totalCount === 0) return '0%'
  
  const rate = (errorCount / totalCount) * 100
  return rate.toFixed(2) + '%'
}

/**
 * 格式化速率（每秒）
 * @param {number} count - 数量
 * @param {number} timeSeconds - 时间（秒）
 * @returns {string} 格式化后的速率
 */
export const formatRate = (count, timeSeconds) => {
  if (!timeSeconds || timeSeconds === 0) return '0/s'
  
  const rate = count / timeSeconds
  
  if (rate < 1) {
    return rate.toFixed(2) + '/s'
  } else if (rate < 1000) {
    return Math.round(rate) + '/s'
  } else {
    return (rate / 1000).toFixed(1) + 'K/s'
  }
}

/**
 * 格式化延迟时间
 * @param {number} latency - 延迟时间(毫秒)
 * @returns {string} 格式化后的延迟时间
 */
export const formatLatency = (latency) => {
  if (latency === null || latency === undefined) return '-'
  
  const ms = Number(latency)
  if (isNaN(ms)) return '-'
  
  if (ms < 1) {
    return '<1ms'
  } else if (ms < 1000) {
    return `${ms.toFixed(0)}ms`
  } else {
    return `${(ms / 1000).toFixed(2)}s`
  }
}

/**
 * 格式化在线状态
 * @param {boolean} isOnline - 是否在线
 * @returns {object} 状态信息对象
 */
export const formatOnlineStatus = (isOnline) => {
  return {
    text: isOnline ? '在线' : '离线',
    type: isOnline ? 'success' : 'danger',
    color: isOnline ? '#67C23A' : '#F56C6C'
  }
}

/**
 * 格式化连接角色
 * @param {string} role - 角色
 * @returns {object} 角色信息对象
 */
export const formatRole = (role) => {
  const roleMap = {
    provider: { text: '传卡端', type: 'primary' },
    receiver: { text: '收卡端', type: 'success' },
    none: { text: '未分配', type: 'info' }
  }
  
  return roleMap[role] || { text: role || '-', type: 'info' }
}

/**
 * 格式化会话状态
 * @param {string} status - 会话状态
 * @returns {object} 状态信息对象
 */
export const formatSessionStatus = (status) => {
  const statusMap = {
    paired: { text: '已配对', type: 'success' },
    waiting: { text: '等待配对', type: 'warning' },
    terminated: { text: '已终止', type: 'danger' },
    error: { text: '错误', type: 'danger' }
  }
  
  return statusMap[status] || { text: status || '-', type: 'info' }
}

/**
 * 格式化事件类型
 * @param {string} eventType - 事件类型
 * @returns {object} 事件类型信息对象
 */
export const formatEventType = (eventType) => {
  const typeMap = {
    session_established: { text: '会话建立', type: 'success' },
    session_terminated: { text: '会话终止', type: 'warning' },
    apdu_relayed_success: { text: 'APDU转发成功', type: 'success' },
    apdu_relayed_failure: { text: 'APDU转发失败', type: 'danger' },
    client_connected: { text: '客户端连接', type: 'primary' },
    client_disconnected: { text: '客户端断开', type: 'warning' },
    auth_failure: { text: '认证失败', type: 'danger' },
    permission_denied: { text: '权限拒绝', type: 'danger' }
  }
  
  return typeMap[eventType] || { text: eventType || '-', type: 'info' }
} 