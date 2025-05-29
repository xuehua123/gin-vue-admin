// 实时数据管理器
// 负责WebSocket连接、数据更新、状态管理

import { ref, reactive, computed } from 'vue'
import { ElMessage, ElNotification } from 'element-plus'

class RealtimeDataManager {
  constructor() {
    this.ws = null
    this.reconnectTimer = null
    this.heartbeatTimer = null
    this.reconnectAttempts = 0
    this.maxReconnectAttempts = 5
    this.reconnectInterval = 3000
    this.heartbeatInterval = 30000
    
    // 响应式数据
    this.isConnected = ref(false)
    this.connectionStatus = ref('disconnected') // disconnected, connecting, connected, error
    this.lastUpdateTime = ref(null)
    
    // 仪表盘数据
    this.dashboardData = reactive({
      hub_status: 'offline',
      active_connections: 0,
      active_sessions: 0,
      apdu_relayed_last_minute: 0,
      apdu_errors_last_hour: 0,
      connection_trend: [],
      session_trend: [],
      system_load: 0,
      memory_usage: 0,
      avg_response_time: 0
    })
    
    // 连接管理数据
    this.clientsData = reactive({
      list: [],
      total: 0,
      online_count: 0,
      provider_count: 0,
      receiver_count: 0
    })
    
    // 会话管理数据
    this.sessionsData = reactive({
      list: [],
      total: 0,
      paired_count: 0,
      waiting_count: 0
    })
    
    // 事件监听器
    this.eventListeners = new Map()
    
    // 数据变化历史（用于动画效果）
    this.changeHistory = reactive({
      dashboard: {},
      clients: {},
      sessions: {}
    })
  }
  
  /**
   * 连接WebSocket
   */
  connect() {
    if (this.ws && this.ws.readyState === WebSocket.OPEN) {
      return Promise.resolve()
    }
    
    return new Promise((resolve, reject) => {
      try {
        this.connectionStatus.value = 'connecting'
        
        // 构建WebSocket URL - 智能检测环境
        let wsUrl
        
        if (process.env.NODE_ENV === 'development') {
          // 开发环境：使用Vite WebSocket代理
          const protocol = window.location.protocol === 'https:' ? 'wss:' : 'ws:'
          const host = window.location.host // 使用前端地址和端口
          wsUrl = `${protocol}//${host}/ws/nfc-relay/realtime`
        } else {
          // 生产环境：使用动态检测
          const protocol = window.location.protocol === 'https:' ? 'wss:' : 'ws:'
          const host = window.location.host
          wsUrl = `${protocol}//${host}/ws/nfc-relay/realtime`
        }
        
        console.log('🔄 Connecting to WebSocket:', wsUrl)
        this.ws = new WebSocket(wsUrl)
        
        this.ws.onopen = () => {
          console.log('✅ WebSocket connected')
          this.isConnected.value = true
          this.connectionStatus.value = 'connected'
          this.reconnectAttempts = 0
          
          // 立即请求初始数据
          setTimeout(() => {
            this.requestStatusUpdate()
          }, 200)
          
          this.startHeartbeat()
          resolve()
        }
        
        this.ws.onmessage = (event) => {
          this.handleMessage(event)
        }
        
        this.ws.onclose = () => {
          console.log('❌ WebSocket disconnected')
          this.isConnected.value = false
          this.connectionStatus.value = 'disconnected'
          this.stopHeartbeat()
          this.scheduleReconnect()
        }
        
        this.ws.onerror = (error) => {
          console.error('WebSocket error:', error)
          this.connectionStatus.value = 'error'
          reject(error)
        }
        
      } catch (error) {
        this.connectionStatus.value = 'error'
        reject(error)
      }
    })
  }
  
  /**
   * 断开连接
   */
  disconnect() {
    if (this.ws) {
      this.ws.close()
      this.ws = null
    }
    
    this.stopHeartbeat()
    if (this.reconnectTimer) {
      clearTimeout(this.reconnectTimer)
      this.reconnectTimer = null
    }
    
    this.isConnected.value = false
    this.connectionStatus.value = 'disconnected'
  }
  
  /**
   * 处理WebSocket消息
   */
  handleMessage(event) {
    try {
      const data = JSON.parse(event.data)
      this.lastUpdateTime.value = new Date()
      
      switch (data.type) {
        case 'dashboard_update':
          this.updateDashboardData(data.payload)
          break
        case 'clients_update':
          this.updateClientsData(data.payload)
          break
        case 'sessions_update':
          this.updateSessionsData(data.payload)
          break
        case 'client_connected':
          this.handleClientConnected(data.payload)
          break
        case 'client_disconnected':
          this.handleClientDisconnected(data.payload)
          break
        case 'session_created':
          this.handleSessionCreated(data.payload)
          break
        case 'session_terminated':
          this.handleSessionTerminated(data.payload)
          break
        case 'apdu_relayed':
          this.handleApduRelayed(data.payload)
          break
        case 'heartbeat':
          // 心跳响应，不需要处理
          break
        case 'error':
          this.handleError(data)
          break
        default:
          console.warn('Unknown message type:', data.type)
      }
      
      // 触发事件监听器
      this.emit(data.type, data.payload)
      
    } catch (error) {
      console.error('Failed to parse WebSocket message:', error)
    }
  }
  
  /**
   * 更新仪表盘数据
   */
  updateDashboardData(newData) {
    console.log('📊 Updating dashboard data:', newData)
    
    // 记录变化用于动画效果
    Object.keys(newData).forEach(key => {
      if (this.dashboardData[key] !== newData[key]) {
        console.log(`🔄 Dashboard field changed: ${key} from ${this.dashboardData[key]} to ${newData[key]}`)
        this.changeHistory.dashboard[key] = {
          oldValue: this.dashboardData[key],
          newValue: newData[key],
          timestamp: Date.now()
        }
      }
    })
    
    // 更新数据
    Object.assign(this.dashboardData, newData)
    
    console.log('✅ Dashboard data updated:', {
      hub_status: this.dashboardData.hub_status,
      active_connections: this.dashboardData.active_connections,
      active_sessions: this.dashboardData.active_sessions
    })
  }
  
  /**
   * 更新客户端数据
   */
  updateClientsData(newData) {
    // 计算变化
    const oldTotal = this.clientsData.total
    Object.assign(this.clientsData, newData)
    
    if (oldTotal !== newData.total) {
      this.changeHistory.clients.total = {
        oldValue: oldTotal,
        newValue: newData.total,
        timestamp: Date.now()
      }
    }
  }
  
  /**
   * 更新会话数据
   */
  updateSessionsData(newData) {
    const oldTotal = this.sessionsData.total
    Object.assign(this.sessionsData, newData)
    
    if (oldTotal !== newData.total) {
      this.changeHistory.sessions.total = {
        oldValue: oldTotal,
        newValue: newData.total,
        timestamp: Date.now()
      }
    }
  }
  
  /**
   * 处理客户端连接事件
   */
  handleClientConnected(clientData) {
    ElNotification({
      title: '客户端连接',
      message: `${clientData.display_name || clientData.client_id} 已连接`,
      type: 'success',
      position: 'bottom-right',
      duration: 3000
    })
  }
  
  /**
   * 处理客户端断开事件
   */
  handleClientDisconnected(clientData) {
    ElNotification({
      title: '客户端断开',
      message: `${clientData.display_name || clientData.client_id} 已断开`,
      type: 'warning',
      position: 'bottom-right',
      duration: 3000
    })
  }
  
  /**
   * 处理会话创建事件
   */
  handleSessionCreated(sessionData) {
    ElNotification({
      title: '新会话创建',
      message: `会话 ${sessionData.session_id} 已建立`,
      type: 'info',
      position: 'bottom-right',
      duration: 3000
    })
  }
  
  /**
   * 处理会话终止事件
   */
  handleSessionTerminated(sessionData) {
    ElNotification({
      title: '会话终止',
      message: `会话 ${sessionData.session_id} 已终止`,
      type: 'warning',
      position: 'bottom-right',
      duration: 3000
    })
  }
  
  /**
   * 处理APDU中继事件
   */
  handleApduRelayed(apduData) {
    // 更新APDU计数等实时指标
    this.dashboardData.apdu_relayed_last_minute++
  }
  
  /**
   * 开始心跳
   */
  startHeartbeat() {
    this.heartbeatTimer = setInterval(() => {
      if (this.ws && this.ws.readyState === WebSocket.OPEN) {
        this.ws.send(JSON.stringify({ type: 'heartbeat' }))
      }
    }, this.heartbeatInterval)
  }
  
  /**
   * 停止心跳
   */
  stopHeartbeat() {
    if (this.heartbeatTimer) {
      clearInterval(this.heartbeatTimer)
      this.heartbeatTimer = null
    }
  }
  
  /**
   * 计划重连
   */
  scheduleReconnect() {
    if (this.reconnectAttempts >= this.maxReconnectAttempts) {
      ElMessage.error('WebSocket连接失败，请检查网络连接')
      return
    }
    
    this.reconnectTimer = setTimeout(() => {
      this.reconnectAttempts++
      console.log(`Attempting to reconnect... (${this.reconnectAttempts}/${this.maxReconnectAttempts})`)
      this.connect().catch(() => {
        // 重连失败，会自动触发scheduleReconnect
      })
    }, this.reconnectInterval)
  }
  
  /**
   * 发送消息
   */
  send(data) {
    if (this.ws && this.ws.readyState === WebSocket.OPEN) {
      this.ws.send(JSON.stringify(data))
    } else {
      console.warn('WebSocket is not connected')
    }
  }
  
  /**
   * 添加事件监听器
   */
  on(event, callback) {
    if (!this.eventListeners.has(event)) {
      this.eventListeners.set(event, [])
    }
    this.eventListeners.get(event).push(callback)
  }
  
  /**
   * 移除事件监听器
   */
  off(event, callback) {
    if (this.eventListeners.has(event)) {
      const listeners = this.eventListeners.get(event)
      const index = listeners.indexOf(callback)
      if (index > -1) {
        listeners.splice(index, 1)
      }
    }
  }
  
  /**
   * 触发事件
   */
  emit(event, data) {
    if (this.eventListeners.has(event)) {
      this.eventListeners.get(event).forEach(callback => {
        try {
          callback(data)
        } catch (error) {
          console.error('Error in event listener:', error)
        }
      })
    }
  }
  
  /**
   * 获取数据变化动画信息
   */
  getChangeAnimation(category, key) {
    const change = this.changeHistory[category][key]
    if (!change) return null
    
    // 如果变化时间超过3秒，不显示动画
    if (Date.now() - change.timestamp > 3000) {
      return null
    }
    
    return {
      type: change.newValue > change.oldValue ? 'increase' : 'decrease',
      oldValue: change.oldValue,
      newValue: change.newValue,
      diff: Math.abs(change.newValue - change.oldValue)
    }
  }
  
  /**
   * 清理资源
   */
  destroy() {
    this.disconnect()
    this.eventListeners.clear()
  }
  
  /**
   * 请求状态更新
   */
  requestStatusUpdate() {
    if (this.ws && this.ws.readyState === WebSocket.OPEN) {
      this.send({ type: 'request_status_update' })
      console.log('🔄 Requested status update from server')
    }
  }
  
  /**
   * 处理服务器错误消息
   */
  handleError(errorData) {
    const errorCode = errorData.code || 'UNKNOWN'
    const errorMessage = errorData.message || '未知错误'
    
    console.error(`WebSocket Error [${errorCode}]:`, errorMessage)
    
    // 根据错误类型显示不同的提示
    switch (errorCode) {
      case 40101: // 认证错误
        ElMessage.warning('WebSocket连接需要重新认证，正在尝试重连...')
        // 可以在这里触发重新认证或重连逻辑
        break
      case 40001: // 权限错误
        ElMessage.error('权限不足：' + errorMessage)
        break
      default:
        ElMessage.error(`连接错误：${errorMessage}`)
    }
  }
}

// 创建全局实例
export const realtimeDataManager = new RealtimeDataManager()

// 导出钩子函数
export const useRealtimeData = () => {
  return {
    // 连接状态
    isConnected: realtimeDataManager.isConnected,
    connectionStatus: realtimeDataManager.connectionStatus,
    lastUpdateTime: realtimeDataManager.lastUpdateTime,
    
    // 数据
    dashboardData: realtimeDataManager.dashboardData,
    clientsData: realtimeDataManager.clientsData,
    sessionsData: realtimeDataManager.sessionsData,
    
    // 方法
    connect: () => realtimeDataManager.connect(),
    disconnect: () => realtimeDataManager.disconnect(),
    send: (data) => realtimeDataManager.send(data),
    on: (event, callback) => realtimeDataManager.on(event, callback),
    off: (event, callback) => realtimeDataManager.off(event, callback),
    getChangeAnimation: (category, key) => realtimeDataManager.getChangeAnimation(category, key),
    
    // 计算属性
    isOnline: computed(() => realtimeDataManager.connectionStatus.value === 'connected')
  }
} 