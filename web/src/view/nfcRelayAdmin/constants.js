/**
 * NFC中继管理常量配置
 * 统一管理API和WebSocket地址，避免硬编码
 */

// API配置
export const API_CONFIG = {
  // HTTP API基础路径 - 去掉重复的 /api，因为request.js中已经配置了baseURL
  BASE_URL: '/admin/nfc-relay/v1',
  
  // WebSocket配置
  WEBSOCKET: {
    // 获取WebSocket基础URL
    getBaseUrl: () => {
      // 开发环境直接使用后端端口
      if (process.env.NODE_ENV === 'development') {
        return 'ws://localhost:8888'
      }
      // 生产环境使用动态检测
      const protocol = window.location.protocol === 'https:' ? 'wss:' : 'ws:'
      const host = window.location.host
      return `${protocol}//${host}`
    },
    
    // WebSocket端点路径
    ENDPOINTS: {
      // 主要的实时数据WebSocket
      REALTIME: '/ws/nfc-relay/realtime',
      
      // 以下为扩展端点，需要后端支持
      LOG_STREAM: '/ws/nfc-relay/logs',
      APDU_MONITOR: '/ws/nfc-relay/apdu',
      SYSTEM_METRICS: '/ws/nfc-relay/metrics'
    },
    
    // 获取完整的WebSocket URL
    getUrl: (endpoint) => {
      const baseUrl = API_CONFIG.WEBSOCKET.getBaseUrl()
      return `${baseUrl}${endpoint}`
    }
  }
}

// WebSocket连接配置
export const WEBSOCKET_CONFIG = {
  // 重连配置
  RECONNECT: {
    MAX_ATTEMPTS: 5,
    DELAY: 3000,
    BACKOFF_MULTIPLIER: 1.5
  },
  
  // 心跳配置
  HEARTBEAT: {
    INTERVAL: 30000,
    TIMEOUT: 5000,
    MESSAGE: 'ping'
  },
  
  // 连接超时
  CONNECT_TIMEOUT: 10000
}

// 状态映射
export const CONNECTION_STATUS = {
  CONNECTING: 'connecting',
  CONNECTED: 'connected',
  DISCONNECTED: 'disconnected',
  ERROR: 'error',
  RECONNECTING: 'reconnecting'
}

// 消息类型
export const MESSAGE_TYPES = {
  // 心跳消息
  PING: 'ping',
  PONG: 'pong',
  
  // 数据消息
  REAL_TIME_DATA: 'realtime_data',
  LOG_ENTRY: 'log_entry',
  APDU_DATA: 'apdu_data',
  METRICS_DATA: 'metrics_data',
  
  // 控制消息
  SUBSCRIBE: 'subscribe',
  UNSUBSCRIBE: 'unsubscribe',
  ERROR: 'error'
}

// 日志级别
export const LOG_LEVELS = {
  DEBUG: 'debug',
  INFO: 'info',
  WARN: 'warn',
  ERROR: 'error',
  CRITICAL: 'critical'
}

// 日志级别颜色映射
export const LOG_LEVEL_COLORS = {
  [LOG_LEVELS.DEBUG]: '#909399',
  [LOG_LEVELS.INFO]: '#67c23a',
  [LOG_LEVELS.WARN]: '#e6a23c',
  [LOG_LEVELS.ERROR]: '#f56c6c',
  [LOG_LEVELS.CRITICAL]: '#f56c6c'
}

// 系统状态
export const SYSTEM_STATUS = {
  HEALTHY: 'healthy',
  WARNING: 'warning',
  ERROR: 'error',
  UNKNOWN: 'unknown'
}

// 客户端状态
export const CLIENT_STATUS = {
  ONLINE: 'online',
  OFFLINE: 'offline',
  IDLE: 'idle',
  BUSY: 'busy'
}

// 会话状态
export const SESSION_STATUS = {
  ACTIVE: 'active',
  PAUSED: 'paused',
  COMPLETED: 'completed',
  FAILED: 'failed',
  TERMINATED: 'terminated'
}

// 页面刷新间隔（毫秒）
export const REFRESH_INTERVALS = {
  REAL_TIME: 1000,      // 实时数据
  DASHBOARD: 30000,     // 仪表盘
  LIST: 60000,          // 列表页面
  DETAILS: 10000        // 详情页面
}

// 分页配置
export const PAGINATION = {
  DEFAULT_PAGE_SIZE: 20,
  PAGE_SIZE_OPTIONS: [10, 20, 50, 100],
  MAX_PAGE_SIZE: 100
}

// 导出格式
export const EXPORT_FORMATS = {
  JSON: 'json',
  CSV: 'csv',
  EXCEL: 'xlsx',
  PDF: 'pdf'
}

// 时间范围预设
export const TIME_RANGES = {
  LAST_HOUR: '1h',
  LAST_DAY: '1d',
  LAST_WEEK: '7d',
  LAST_MONTH: '30d',
  CUSTOM: 'custom'
}

// 时间范围标签
export const TIME_RANGE_LABELS = {
  [TIME_RANGES.LAST_HOUR]: '最近1小时',
  [TIME_RANGES.LAST_DAY]: '最近1天',
  [TIME_RANGES.LAST_WEEK]: '最近1周',
  [TIME_RANGES.LAST_MONTH]: '最近1月',
  [TIME_RANGES.CUSTOM]: '自定义'
}

// NFC中继管理模块常量配置

// 客户端角色类型
export const CLIENT_ROLES = {
  PROVIDER: 'provider',
  RECEIVER: 'receiver',
  NONE: 'none'
}

// 客户端角色显示文本
export const CLIENT_ROLE_LABELS = {
  [CLIENT_ROLES.PROVIDER]: 'Provider',
  [CLIENT_ROLES.RECEIVER]: 'Receiver', 
  [CLIENT_ROLES.NONE]: 'None'
}

// 审计日志事件类型
export const AUDIT_EVENT_TYPES = {
  SESSION_ESTABLISHED: 'session_established',
  APDU_RELAYED_SUCCESS: 'apdu_relayed_success',
  APDU_RELAYED_FAILURE: 'apdu_relayed_failure',
  AUTH_FAILURE: 'auth_failure',
  CLIENT_DISCONNECTED: 'client_disconnected',
  SESSION_TERMINATED: 'session_terminated'
}

// 审计日志事件类型显示文本
export const AUDIT_EVENT_TYPE_LABELS = {
  [AUDIT_EVENT_TYPES.SESSION_ESTABLISHED]: '会话建立',
  [AUDIT_EVENT_TYPES.APDU_RELAYED_SUCCESS]: 'APDU转发成功',
  [AUDIT_EVENT_TYPES.APDU_RELAYED_FAILURE]: 'APDU转发失败',
  [AUDIT_EVENT_TYPES.AUTH_FAILURE]: '认证失败',
  [AUDIT_EVENT_TYPES.CLIENT_DISCONNECTED]: '客户端断开',
  [AUDIT_EVENT_TYPES.SESSION_TERMINATED]: '会话终止'
}

// 数据导出配置
export const EXPORT_CONFIG = {
  MAX_EXPORT_RECORDS: 10000,
  DEFAULT_FILENAME_PREFIX: 'nfc_relay_export'
}

// 错误信息
export const ERROR_MESSAGES = {
  NETWORK_ERROR: '网络连接失败，请检查网络设置',
  PERMISSION_DENIED: '权限不足，无法执行此操作',
  DATA_LOAD_FAILED: '数据加载失败，请重试',
  OPERATION_FAILED: '操作失败，请重试',
  INVALID_PARAMS: '参数格式不正确'
} 