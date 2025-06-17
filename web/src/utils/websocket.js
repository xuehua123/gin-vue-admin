// 移除直接导入
// import { useUserStore } from '@/pinia/modules/user'

class WebSocketManager {
  constructor() {
    this.ws = null
    this.reconnectTimer = null
    this.heartbeatTimer = null
    this.messageHandlers = new Map()
    this.reconnectDelay = 1000
    this.maxReconnectDelay = 30000
    this.heartbeatInterval = 30000
    this.isConnecting = false
  }

  /**
   * 连接WebSocket
   * @param {Object} options 连接选项
   * @param {string} options.url WebSocket服务器地址
   * @param {string} options.userId 用户ID
   * @param {string} options.clientId 客户端ID
   */
  async connect(options = {}) {
    if (this.isConnecting || (this.ws && this.ws.readyState === WebSocket.CONNECTING)) {
      return
    }

    this.isConnecting = true
    // 动态导入useUserStore，避免循环依赖
    const { useUserStore } = await import('@/pinia/modules/user')
    const userStore = useUserStore()
    
    const wsUrl = options.url || this.getWebSocketURL()
    const userId = options.userId || userStore.userInfo.ID
    const clientId = options.clientId || this.generateClientId()

    const url = `${wsUrl}?user_id=${userId}&client_id=${clientId}`

    try {
      this.ws = new WebSocket(url)
      this.setupEventHandlers()
    } catch (error) {
      console.error('WebSocket连接失败:', error)
      this.isConnecting = false
      this.scheduleReconnect()
    }
  }

  /**
   * 获取WebSocket服务器地址
   */
  getWebSocketURL() {
    const protocol = window.location.protocol === 'https:' ? 'wss:' : 'ws:'
    
    // 开发环境：通过Vite代理连接
    if (import.meta.env.DEV) {
      const host = window.location.host // 这是Vite开发服务器的地址和端口
      const apiPrefix = import.meta.env.VITE_BASE_API || '' // 从环境变量获取API前缀
      return `${protocol}//${host}${apiPrefix}/nfc-relay/ws`
    }
    
    // 生产环境：使用当前域名
    const host = window.location.host
    return `${protocol}//${host}/nfc-relay/ws`
  }

  /**
   * 生成客户端ID
   */
  generateClientId() {
    return `web_${Date.now()}_${Math.random().toString(36).substr(2, 9)}`
  }

  /**
   * 设置事件处理器
   */
  setupEventHandlers() {
    this.ws.onopen = (event) => {
      console.log('WebSocket连接已建立')
      this.isConnecting = false
      this.reconnectDelay = 1000
      this.startHeartbeat()
      this.emit('open', event)
    }

    this.ws.onmessage = (event) => {
      try {
        const message = JSON.parse(event.data)
        this.handleMessage(message)
      } catch (error) {
        console.error('解析WebSocket消息失败:', error, event.data)
      }
    }

    this.ws.onclose = (event) => {
      console.log('WebSocket连接已断开:', event.code, event.reason)
      this.isConnecting = false
      this.stopHeartbeat()
      this.emit('close', event)
      
      if (!event.wasClean) {
        this.scheduleReconnect()
      }
    }

    this.ws.onerror = (error) => {
      console.error('WebSocket错误:', error)
      this.isConnecting = false
      this.emit('error', error)
    }
  }

  /**
   * 处理收到的消息
   */
  handleMessage(message) {
    const { type, data, timestamp } = message

    // 处理心跳响应
    if (type === 'pong') {
      return
    }

    // 触发对应类型的处理器
    if (this.messageHandlers.has(type)) {
      const handlers = this.messageHandlers.get(type)
      handlers.forEach(handler => {
        try {
          handler(data, message)
        } catch (error) {
          console.error(`处理${type}类型消息失败:`, error)
        }
      })
    }

    // 触发通用消息处理器
    this.emit('message', message)
  }

  /**
   * 添加消息处理器
   * @param {string} type 消息类型
   * @param {Function} handler 处理函数
   */
  on(type, handler) {
    if (!this.messageHandlers.has(type)) {
      this.messageHandlers.set(type, [])
    }
    this.messageHandlers.get(type).push(handler)
  }

  /**
   * 移除消息处理器
   * @param {string} type 消息类型
   * @param {Function} handler 处理函数
   */
  off(type, handler) {
    if (this.messageHandlers.has(type)) {
      const handlers = this.messageHandlers.get(type)
      const index = handlers.indexOf(handler)
      if (index > -1) {
        handlers.splice(index, 1)
      }
    }
  }

  /**
   * 触发事件
   */
  emit(type, data) {
    if (this.messageHandlers.has(type)) {
      const handlers = this.messageHandlers.get(type)
      handlers.forEach(handler => handler(data))
    }
  }

  /**
   * 发送消息
   * @param {Object} message 消息对象
   */
  send(message) {
    if (this.ws && this.ws.readyState === WebSocket.OPEN) {
      this.ws.send(JSON.stringify(message))
      return true
    } else {
      console.warn('WebSocket未连接，无法发送消息:', message)
      return false
    }
  }

  /**
   * 启动心跳
   */
  startHeartbeat() {
    this.stopHeartbeat()
    this.heartbeatTimer = setInterval(() => {
      this.send({
        type: 'ping',
        timestamp: new Date().toISOString()
      })
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
    if (this.reconnectTimer) {
      return
    }

    console.log(`${this.reconnectDelay}ms后尝试重连...`)
    this.reconnectTimer = setTimeout(() => {
      this.reconnectTimer = null
      this.connect()
      this.reconnectDelay = Math.min(this.reconnectDelay * 2, this.maxReconnectDelay)
    }, this.reconnectDelay)
  }

  /**
   * 断开连接
   */
  disconnect() {
    if (this.reconnectTimer) {
      clearTimeout(this.reconnectTimer)
      this.reconnectTimer = null
    }
    
    this.stopHeartbeat()
    
    if (this.ws) {
      this.ws.close(1000, 'Client disconnect')
      this.ws = null
    }
  }

  /**
   * 获取连接状态
   */
  getState() {
    if (!this.ws) return 'CLOSED'
    
    switch (this.ws.readyState) {
      case WebSocket.CONNECTING:
        return 'CONNECTING'
      case WebSocket.OPEN:
        return 'OPEN'
      case WebSocket.CLOSING:
        return 'CLOSING'
      case WebSocket.CLOSED:
        return 'CLOSED'
      default:
        return 'UNKNOWN'
    }
  }

  /**
   * 是否已连接
   */
  isConnected() {
    return this.ws && this.ws.readyState === WebSocket.OPEN
  }
}

// 全局WebSocket管理器实例
export const wsManager = new WebSocketManager()

// 导出类以便创建多个实例
export default WebSocketManager 