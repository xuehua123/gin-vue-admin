/**
 * 实时数据更新组合式函数
 * 用于处理需要定时刷新的数据
 */

import { ref, onMounted, onUnmounted } from 'vue'

/**
 * 实时数据更新hook
 * @param {Function} fetchFunction - 数据获取函数
 * @param {number} interval - 刷新间隔（毫秒）
 * @param {boolean} immediate - 是否立即执行
 * @returns {object} 状态和方法
 */
export function useRealTimeData(fetchFunction, interval = 30000, immediate = true) {
  const data = ref(null)
  const loading = ref(false)
  const error = ref(null)
  const isPolling = ref(false)
  
  let timer = null
  
  /**
   * 获取数据
   */
  const fetchData = async () => {
    try {
      loading.value = true
      error.value = null
      
      const result = await fetchFunction()
      data.value = result
      
      return result
    } catch (err) {
      error.value = err
      console.error('获取实时数据失败:', err)
      throw err
    } finally {
      loading.value = false
    }
  }
  
  /**
   * 开始轮询
   */
  const startPolling = () => {
    if (isPolling.value) return
    
    isPolling.value = true
    
    if (immediate) {
      fetchData()
    }
    
    timer = setInterval(fetchData, interval)
  }
  
  /**
   * 停止轮询
   */
  const stopPolling = () => {
    if (timer) {
      clearInterval(timer)
      timer = null
    }
    isPolling.value = false
  }
  
  /**
   * 手动刷新
   */
  const refresh = () => {
    return fetchData()
  }
  
  /**
   * 重置状态
   */
  const reset = () => {
    data.value = null
    error.value = null
    loading.value = false
  }
  
  // 生命周期管理
  onMounted(() => {
    startPolling()
  })
  
  onUnmounted(() => {
    stopPolling()
  })
  
  return {
    data,
    loading,
    error,
    isPolling,
    startPolling,
    stopPolling,
    refresh,
    reset
  }
}

/**
 * 实时状态监控hook
 * @param {Function} fetchFunction - 数据获取函数
 * @param {object} options - 配置选项
 * @returns {object} 状态和方法
 */
export function useRealTimeStatus(fetchFunction, options = {}) {
  const {
    interval = 10000,
    errorRetryDelay = 5000,
    maxRetries = 3,
    onError = null,
    onSuccess = null
  } = options
  
  const status = ref('idle') // idle, loading, success, error
  const data = ref(null)
  const error = ref(null)
  const retryCount = ref(0)
  const lastUpdateTime = ref(null)
  
  let timer = null
  let retryTimer = null
  
  /**
   * 获取状态数据
   */
  const fetchStatus = async () => {
    try {
      status.value = 'loading'
      
      const result = await fetchFunction()
      
      data.value = result
      error.value = null
      status.value = 'success'
      retryCount.value = 0
      lastUpdateTime.value = new Date()
      
      if (onSuccess) {
        onSuccess(result)
      }
      
      // 设置下次定时器
      scheduleNext()
      
    } catch (err) {
      error.value = err
      status.value = 'error'
      retryCount.value++
      
      console.error('获取状态数据失败:', err)
      
      if (onError) {
        onError(err)
      }
      
      // 重试逻辑
      if (retryCount.value <= maxRetries) {
        retryTimer = setTimeout(fetchStatus, errorRetryDelay)
      } else {
        console.error('达到最大重试次数，停止重试')
      }
    }
  }
  
  /**
   * 安排下次执行
   */
  const scheduleNext = () => {
    timer = setTimeout(fetchStatus, interval)
  }
  
  /**
   * 开始监控
   */
  const start = () => {
    stop() // 清理之前的定时器
    fetchStatus()
  }
  
  /**
   * 停止监控
   */
  const stop = () => {
    if (timer) {
      clearTimeout(timer)
      timer = null
    }
    if (retryTimer) {
      clearTimeout(retryTimer)
      retryTimer = null
    }
  }
  
  /**
   * 重置重试计数
   */
  const resetRetry = () => {
    retryCount.value = 0
  }
  
  onMounted(start)
  onUnmounted(stop)
  
  return {
    status,
    data,
    error,
    retryCount,
    lastUpdateTime,
    start,
    stop,
    resetRetry,
    refresh: fetchStatus
  }
}

/**
 * WebSocket实时连接hook
 * @param {string} url - WebSocket URL
 * @param {object} options - 配置选项
 * @returns {object} 连接状态和方法
 */
export function useWebSocketConnection(url, options = {}) {
  const {
    reconnectDelay = 3000,
    maxReconnects = 5,
    heartbeatInterval = 30000,
    onMessage = null,
    onError = null,
    onOpen = null,
    onClose = null
  } = options
  
  const ws = ref(null)
  const status = ref('disconnected') // disconnected, connecting, connected, error
  const lastMessage = ref(null)
  const reconnectCount = ref(0)
  
  let heartbeatTimer = null
  let reconnectTimer = null
  
  /**
   * 连接WebSocket
   */
  const connect = () => {
    if (ws.value && ws.value.readyState === WebSocket.OPEN) {
      return
    }
    
    status.value = 'connecting'
    
    try {
      ws.value = new WebSocket(url)
      
      ws.value.onopen = (event) => {
        status.value = 'connected'
        reconnectCount.value = 0
        
        // 启动心跳
        startHeartbeat()
        
        if (onOpen) {
          onOpen(event)
        }
      }
      
      ws.value.onmessage = (event) => {
        lastMessage.value = event.data
        
        if (onMessage) {
          onMessage(event.data)
        }
      }
      
      ws.value.onerror = (event) => {
        status.value = 'error'
        
        if (onError) {
          onError(event)
        }
      }
      
      ws.value.onclose = (event) => {
        status.value = 'disconnected'
        stopHeartbeat()
        
        if (onClose) {
          onClose(event)
        }
        
        // 自动重连
        if (reconnectCount.value < maxReconnects) {
          reconnectCount.value++
          reconnectTimer = setTimeout(connect, reconnectDelay)
        }
      }
      
    } catch (err) {
      status.value = 'error'
      console.error('WebSocket连接失败:', err)
    }
  }
  
  /**
   * 断开连接
   */
  const disconnect = () => {
    stopHeartbeat()
    
    if (reconnectTimer) {
      clearTimeout(reconnectTimer)
      reconnectTimer = null
    }
    
    if (ws.value) {
      ws.value.close()
      ws.value = null
    }
    
    status.value = 'disconnected'
  }
  
  /**
   * 发送消息
   */
  const send = (message) => {
    if (ws.value && ws.value.readyState === WebSocket.OPEN) {
      ws.value.send(message)
      return true
    }
    return false
  }
  
  /**
   * 启动心跳
   */
  const startHeartbeat = () => {
    stopHeartbeat()
    heartbeatTimer = setInterval(() => {
      if (ws.value && ws.value.readyState === WebSocket.OPEN) {
        send('ping')
      }
    }, heartbeatInterval)
  }
  
  /**
   * 停止心跳
   */
  const stopHeartbeat = () => {
    if (heartbeatTimer) {
      clearInterval(heartbeatTimer)
      heartbeatTimer = null
    }
  }
  
  onUnmounted(disconnect)
  
  return {
    status,
    lastMessage,
    reconnectCount,
    connect,
    disconnect,
    send
  }
} 