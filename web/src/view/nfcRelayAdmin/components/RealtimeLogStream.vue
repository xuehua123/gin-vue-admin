<!--
  实时日志流组件
  实时显示审计日志和系统事件
-->
<template>
  <div class="realtime-log-stream">
    <el-card shadow="never">
      <template #header>
        <div class="flex justify-between items-center">
          <div class="flex items-center">
            <el-icon class="mr-2" color="#67c23a">
              <Document />
            </el-icon>
            <span class="text-lg font-semibold">实时日志流</span>
            <el-tag 
              :type="isStreaming ? 'success' : 'info'" 
              size="small" 
              class="ml-2"
              effect="light"
            >
              {{ isStreaming ? '流式传输中' : '已停止' }}
            </el-tag>
          </div>
          <div class="flex items-center space-x-2">
            <el-button 
              :type="isStreaming ? 'danger' : 'success'" 
              size="small"
              @click="toggleStreaming"
            >
              <el-icon>
                <VideoPlay v-if="!isStreaming" />
                <VideoPause v-else />
              </el-icon>
              {{ isStreaming ? '停止流式传输' : '开始流式传输' }}
            </el-button>
            <el-button size="small" @click="clearLogs">
              <el-icon><Delete /></el-icon>
              清空
            </el-button>
            <el-button size="small" @click="exportLogs">
              <el-icon><Download /></el-icon>
              导出
            </el-button>
            <el-button size="small" @click="showFilters = !showFilters">
              <el-icon><Filter /></el-icon>
              过滤器
            </el-button>
          </div>
        </div>
      </template>

      <!-- 过滤器面板 -->
      <div v-show="showFilters" class="filter-panel mb-4">
        <el-row :gutter="16">
          <el-col :span="6">
            <el-select 
              v-model="filters.level" 
              placeholder="日志级别" 
              clearable
              multiple
            >
              <el-option label="DEBUG" value="debug" />
              <el-option label="INFO" value="info" />
              <el-option label="WARN" value="warn" />
              <el-option label="ERROR" value="error" />
              <el-option label="CRITICAL" value="critical" />
            </el-select>
          </el-col>
          <el-col :span="6">
            <el-select 
              v-model="filters.eventType" 
              placeholder="事件类型" 
              clearable
              multiple
            >
              <el-option label="连接事件" value="connection" />
              <el-option label="会话事件" value="session" />
              <el-option label="APDU事件" value="apdu" />
              <el-option label="认证事件" value="auth" />
              <el-option label="系统事件" value="system" />
              <el-option label="错误事件" value="error" />
            </el-select>
          </el-col>
          <el-col :span="6">
            <el-input 
              v-model="filters.keyword" 
              placeholder="关键词搜索..." 
              clearable
            >
              <template #prefix>
                <el-icon><Search /></el-icon>
              </template>
            </el-input>
          </el-col>
          <el-col :span="6">
            <el-input 
              v-model="filters.source" 
              placeholder="来源过滤..." 
              clearable
            />
          </el-col>
        </el-row>
        
        <el-row :gutter="16" class="mt-3">
          <el-col :span="8">
            <el-checkbox v-model="filters.autoScroll">
              自动滚动到底部
            </el-checkbox>
          </el-col>
          <el-col :span="8">
            <el-checkbox v-model="filters.highlightErrors">
              高亮错误日志
            </el-checkbox>
          </el-col>
          <el-col :span="8">
            <el-checkbox v-model="filters.showTimestamp">
              显示完整时间戳
            </el-checkbox>
          </el-col>
        </el-row>
      </div>

      <!-- 日志统计面板 -->
      <div class="stats-panel mb-4">
        <el-row :gutter="16">
          <el-col :span="4">
            <div class="stat-item stat-item--total">
              <div class="stat-value">{{ totalLogs }}</div>
              <div class="stat-label">总日志数</div>
            </div>
          </el-col>
          <el-col :span="4">
            <div class="stat-item stat-item--error">
              <div class="stat-value">{{ errorCount }}</div>
              <div class="stat-label">错误数</div>
            </div>
          </el-col>
          <el-col :span="4">
            <div class="stat-item stat-item--warn">
              <div class="stat-value">{{ warnCount }}</div>
              <div class="stat-label">警告数</div>
            </div>
          </el-col>
          <el-col :span="4">
            <div class="stat-item stat-item--info">
              <div class="stat-value">{{ infoCount }}</div>
              <div class="stat-label">信息数</div>
            </div>
          </el-col>
          <el-col :span="4">
            <div class="stat-item stat-item--rate">
              <div class="stat-value">{{ logRate }}/s</div>
              <div class="stat-label">日志速率</div>
            </div>
          </el-col>
          <el-col :span="4">
            <div class="stat-item stat-item--sources">
              <div class="stat-value">{{ uniqueSources }}</div>
              <div class="stat-label">日志源</div>
            </div>
          </el-col>
        </el-row>
      </div>

      <!-- 日志流展示 -->
      <div class="log-stream">
        <div ref="logContainer" class="log-container" @scroll="handleScroll">
          <div 
            v-for="(log, index) in filteredLogs" 
            :key="log.id"
            class="log-entry"
            :class="getLogEntryClass(log)"
          >
            <div class="log-line">
              <div class="log-meta">
                <span class="log-timestamp">
                  {{ formatTimestamp(log.timestamp) }}
                </span>
                <el-tag 
                  :type="getLevelTagType(log.level)" 
                  size="small"
                  class="log-level"
                >
                  {{ log.level.toUpperCase() }}
                </el-tag>
                <span class="log-source">{{ log.source }}</span>
                <span v-if="log.eventType" class="log-event-type">
                  [{{ log.eventType }}]
                </span>
              </div>
              
              <div class="log-content">
                <div class="log-message" v-html="formatLogMessage(log.message)"></div>
                
                <div v-if="log.context && Object.keys(log.context).length > 0" class="log-context">
                  <el-collapse accordion>
                    <el-collapse-item title="上下文信息" name="context">
                      <pre class="context-content">{{ JSON.stringify(log.context, null, 2) }}</pre>
                    </el-collapse-item>
                  </el-collapse>
                </div>
                
                <div v-if="log.stackTrace" class="log-stack">
                  <el-collapse accordion>
                    <el-collapse-item title="堆栈信息" name="stack">
                      <pre class="stack-content">{{ log.stackTrace }}</pre>
                    </el-collapse-item>
                  </el-collapse>
                </div>
              </div>
            </div>
          </div>
          
          <!-- 空状态 -->
          <div v-if="filteredLogs.length === 0" class="empty-state">
            <el-icon size="48" color="#c0c4cc">
              <Document />
            </el-icon>
            <div class="empty-text">暂无日志数据</div>
            <div class="empty-subtext">
              {{ isStreaming ? '等待日志流入...' : '请启动日志流或调整过滤条件' }}
            </div>
          </div>
        </div>
      </div>

      <!-- 底部控制栏 -->
      <div class="controls-bar mt-4">
        <div class="flex justify-between items-center">
          <div class="flex items-center space-x-4">
            <span class="text-sm text-gray-500">
              显示: {{ filteredLogs.length }} / {{ totalLogs }} 条日志
            </span>
            <el-button 
              v-if="!filters.autoScroll" 
              size="small" 
              @click="scrollToBottom"
            >
              <el-icon><ArrowDown /></el-icon>
              滚动到底部
            </el-button>
          </div>
          
          <div class="flex items-center space-x-2">
            <span class="text-sm">缓冲区大小:</span>
            <el-input-number 
              v-model="maxLogCount" 
              size="small"
              :min="100"
              :max="10000"
              :step="100"
              style="width: 120px"
              @change="handleBufferSizeChange"
            />
            <span class="text-sm">条</span>
          </div>
        </div>
      </div>
    </el-card>

    <!-- 日志详情弹窗 -->
    <log-detail-modal 
      v-model:visible="detailModalVisible" 
      :log="selectedLog" 
    />
  </div>
</template>

<script setup>
import { ref, reactive, computed, onMounted, onUnmounted, nextTick, watch } from 'vue'
import { ElMessage } from 'element-plus'
import {
  Document,
  VideoPlay,
  VideoPause,
  Delete,
  Download,
  Filter,
  Search,
  ArrowDown
} from '@element-plus/icons-vue'
import LogDetailModal from './LogDetailModal.vue'
import { API_CONFIG, WEBSOCKET_CONFIG, MESSAGE_TYPES, LOG_LEVELS } from '../constants.js'

const props = defineProps({
  bufferSize: {
    type: Number,
    default: 1000
  }
})

// 响应式数据
const isStreaming = ref(false)
const showFilters = ref(false)
const detailModalVisible = ref(false)
const selectedLog = ref(null)
const logContainer = ref(null)
const maxLogCount = ref(props.bufferSize)

// 统计数据
const totalLogs = ref(0)
const errorCount = ref(0)
const warnCount = ref(0)
const infoCount = ref(0)
const logRate = ref(0)
const uniqueSources = ref(0)

// 过滤器
const filters = reactive({
  level: [],
  eventType: [],
  keyword: '',
  source: '',
  autoScroll: true,
  highlightErrors: true,
  showTimestamp: true
})

// 日志数据
const logs = ref([])

// 计算属性
const filteredLogs = computed(() => {
  let filtered = logs.value
  
  if (filters.level.length > 0) {
    filtered = filtered.filter(log => filters.level.includes(log.level))
  }
  
  if (filters.eventType.length > 0) {
    filtered = filtered.filter(log => filters.eventType.includes(log.eventType))
  }
  
  if (filters.keyword) {
    const keyword = filters.keyword.toLowerCase()
    filtered = filtered.filter(log => 
      log.message.toLowerCase().includes(keyword) ||
      log.source.toLowerCase().includes(keyword) ||
      (log.eventType && log.eventType.toLowerCase().includes(keyword))
    )
  }
  
  if (filters.source) {
    filtered = filtered.filter(log => 
      log.source.toLowerCase().includes(filters.source.toLowerCase())
    )
  }
  
  return filtered
})

// WebSocket连接
let ws = null
let logRateInterval = null

// 方法
const toggleStreaming = () => {
  if (isStreaming.value) {
    stopStreaming()
  } else {
    startStreaming()
  }
}

const startStreaming = () => {
  isStreaming.value = true
  connectWebSocket()
  startLogRateCalculation()
  startMockLogGeneration()
  ElMessage.success('日志流已启动')
}

const stopStreaming = () => {
  isStreaming.value = false
  if (ws) {
    ws.close()
    ws = null
  }
  if (logRateInterval) {
    clearInterval(logRateInterval)
    logRateInterval = null
  }
  ElMessage.info('日志流已停止')
}

const connectWebSocket = () => {
  try {
    // 使用统一的WebSocket配置
    const wsUrl = API_CONFIG.WEBSOCKET.getUrl(API_CONFIG.WEBSOCKET.ENDPOINTS.REALTIME)
    console.log('连接WebSocket:', wsUrl)
    
    ws = new WebSocket(wsUrl)
    
    ws.onopen = () => {
      console.log('日志流WebSocket连接已建立')
      // 订阅日志流
      ws.send(JSON.stringify({
        type: MESSAGE_TYPES.SUBSCRIBE,
        topic: 'logs'
      }))
    }
    
    ws.onmessage = (event) => {
      try {
        const message = JSON.parse(event.data)
        
        // 处理不同类型的消息
        if (message.type === MESSAGE_TYPES.LOG_ENTRY) {
          addLogEntry(message.data)
        } else if (message.type === MESSAGE_TYPES.PONG) {
          // 心跳响应，忽略
        } else {
          console.log('收到其他类型消息:', message.type)
        }
      } catch (error) {
        console.error('解析WebSocket消息失败:', error)
      }
    }
    
    ws.onclose = () => {
      console.log('日志流WebSocket连接已关闭')
    }
    
    ws.onerror = (error) => {
      console.error('日志流WebSocket错误:', error)
    }
  } catch (error) {
    console.error('无法建立日志流WebSocket连接:', error)
  }
}

const addLogEntry = (logEntry) => {
  logs.value.push(logEntry)
  
  // 限制缓冲区大小
  if (logs.value.length > maxLogCount.value) {
    logs.value = logs.value.slice(-maxLogCount.value)
  }
  
  updateStats()
  
  if (filters.autoScroll) {
    nextTick(() => {
      scrollToBottom()
    })
  }
}

const updateStats = () => {
  totalLogs.value = logs.value.length
  errorCount.value = logs.value.filter(log => log.level === 'error' || log.level === 'critical').length
  warnCount.value = logs.value.filter(log => log.level === 'warn').length
  infoCount.value = logs.value.filter(log => log.level === 'info' || log.level === 'debug').length
  
  const sources = new Set(logs.value.map(log => log.source))
  uniqueSources.value = sources.size
}

const startLogRateCalculation = () => {
  let lastLogCount = 0
  logRateInterval = setInterval(() => {
    const currentLogCount = logs.value.length
    logRate.value = Math.max(0, currentLogCount - lastLogCount)
    lastLogCount = currentLogCount
  }, 1000)
}

const clearLogs = () => {
  logs.value = []
  totalLogs.value = 0
  errorCount.value = 0
  warnCount.value = 0
  infoCount.value = 0
  logRate.value = 0
  uniqueSources.value = 0
  ElMessage.success('日志已清空')
}

const exportLogs = () => {
  const data = JSON.stringify(filteredLogs.value, null, 2)
  const blob = new Blob([data], { type: 'application/json' })
  const url = URL.createObjectURL(blob)
  const a = document.createElement('a')
  a.href = url
  a.download = `logs-${new Date().toISOString().slice(0, 19)}.json`
  a.click()
  URL.revokeObjectURL(url)
  ElMessage.success('日志已导出')
}

const scrollToBottom = () => {
  if (logContainer.value) {
    logContainer.value.scrollTop = logContainer.value.scrollHeight
  }
}

const handleScroll = () => {
  if (filters.autoScroll && logContainer.value) {
    const { scrollTop, scrollHeight, clientHeight } = logContainer.value
    if (scrollTop + clientHeight < scrollHeight - 100) {
      filters.autoScroll = false
    }
  }
}

const handleBufferSizeChange = (value) => {
  if (logs.value.length > value) {
    logs.value = logs.value.slice(-value)
    updateStats()
  }
}

// 格式化方法
const formatTimestamp = (timestamp) => {
  const date = new Date(timestamp)
  if (filters.showTimestamp) {
    return date.toLocaleString()
  } else {
    return date.toLocaleTimeString()
  }
}

const formatLogMessage = (message) => {
  if (!filters.highlightErrors) {
    return message
  }
  
  // 高亮关键词
  let formatted = message
  if (filters.keyword) {
    const regex = new RegExp(`(${filters.keyword})`, 'gi')
    formatted = formatted.replace(regex, '<mark>$1</mark>')
  }
  
  // 高亮IP地址
  formatted = formatted.replace(/\b(?:[0-9]{1,3}\.){3}[0-9]{1,3}\b/g, '<span class="highlight-ip">$&</span>')
  
  // 高亮错误代码
  formatted = formatted.replace(/\b[4-5]\d{2}\b/g, '<span class="highlight-error">$&</span>')
  
  return formatted
}

const getLogEntryClass = (log) => {
  return {
    'log-entry--debug': log.level === 'debug',
    'log-entry--info': log.level === 'info',
    'log-entry--warn': log.level === 'warn',
    'log-entry--error': log.level === 'error',
    'log-entry--critical': log.level === 'critical'
  }
}

const getLevelTagType = (level) => {
  switch (level) {
    case 'debug': return 'info'
    case 'info': return 'success'
    case 'warn': return 'warning'
    case 'error': return 'danger'
    case 'critical': return 'danger'
    default: return 'info'
  }
}

// 模拟日志生成
const startMockLogGeneration = () => {
  const interval = setInterval(() => {
    if (!isStreaming.value) {
      clearInterval(interval)
      return
    }
    
    const mockLog = generateMockLog()
    addLogEntry(mockLog)
  }, Math.random() * 3000 + 1000) // 1-4秒随机间隔
}

const generateMockLog = () => {
  const levels = ['debug', 'info', 'warn', 'error', 'critical']
  const eventTypes = ['connection', 'session', 'apdu', 'auth', 'system', 'error']
  const sources = ['nfc-relay-hub', 'websocket-server', 'session-manager', 'auth-service', 'apdu-processor']
  
  const messages = [
    'Client connected from IP {ip}',
    'Session {sessionId} established successfully',
    'APDU command processed: {command}',
    'Authentication failed for user {userId}',
    'System performance metrics updated',
    'Error processing request: {error}',
    'WebSocket connection closed: {reason}',
    'Session {sessionId} terminated by admin',
    'Database connection pool exhausted',
    'Configuration reloaded successfully'
  ]
  
  const level = levels[Math.floor(Math.random() * levels.length)]
  const eventType = eventTypes[Math.floor(Math.random() * eventTypes.length)]
  const source = sources[Math.floor(Math.random() * sources.length)]
  const message = messages[Math.floor(Math.random() * messages.length)]
  
  // 替换消息中的占位符
  const formattedMessage = message
    .replace('{ip}', `192.168.1.${Math.floor(Math.random() * 254) + 1}`)
    .replace('{sessionId}', `session-${Date.now()}`)
    .replace('{command}', generateRandomHex(8))
    .replace('{userId}', `user-${Math.floor(Math.random() * 1000) + 1}`)
    .replace('{error}', 'Connection timeout')
    .replace('{reason}', 'Normal closure')
  
  const log = {
    id: `log-${Date.now()}-${Math.random().toString(36).substr(2, 9)}`,
    timestamp: new Date().toISOString(),
    level: level,
    source: source,
    eventType: eventType,
    message: formattedMessage,
    context: Math.random() > 0.7 ? {
      requestId: `req-${Date.now()}`,
      correlationId: `corr-${Math.random().toString(36).substr(2, 9)}`,
      duration: Math.floor(Math.random() * 1000) + 'ms'
    } : null,
    stackTrace: level === 'error' && Math.random() > 0.5 ? generateMockStackTrace() : null
  }
  
  return log
}

const generateRandomHex = (length) => {
  const hex = '0123456789ABCDEF'
  let result = ''
  for (let i = 0; i < length; i++) {
    result += hex.charAt(Math.floor(Math.random() * hex.length))
  }
  return result
}

const generateMockStackTrace = () => {
  return `Error: Connection timeout
    at WebSocketManager.connect (websocket.js:45:23)
    at SessionManager.createSession (session.js:78:12)
    at NfcRelayHub.handleRequest (hub.js:156:7)
    at Server.handleMessage (server.js:89:5)`
}

// 监听器
watch(() => filters.autoScroll, (newValue) => {
  if (newValue) {
    nextTick(() => {
      scrollToBottom()
    })
  }
})

// 生命周期
onUnmounted(() => {
  if (ws) {
    ws.close()
  }
  if (logRateInterval) {
    clearInterval(logRateInterval)
  }
})
</script>

<style scoped lang="scss">
.realtime-log-stream {
  .filter-panel {
    padding: 16px;
    background-color: #f5f7fa;
    border-radius: 8px;
    border: 1px solid #e4e7ed;
    
    .mt-3 {
      margin-top: 0.75rem;
    }
  }
  
  .stats-panel {
    .stat-item {
      text-align: center;
      padding: 12px;
      border-radius: 8px;
      color: white;
      background: linear-gradient(135deg, #667eea 0%, #764ba2 100%);
      
      &.stat-item--error {
        background: linear-gradient(135deg, #f093fb 0%, #f5576c 100%);
      }
      
      &.stat-item--warn {
        background: linear-gradient(135deg, #ffd89b 0%, #19547b 100%);
      }
      
      &.stat-item--info {
        background: linear-gradient(135deg, #a8edea 0%, #fed6e3 100%);
        color: #333;
      }
      
      &.stat-item--rate {
        background: linear-gradient(135deg, #d299c2 0%, #fef9d7 100%);
        color: #333;
      }
      
      &.stat-item--sources {
        background: linear-gradient(135deg, #89f7fe 0%, #66a6ff 100%);
      }
      
      .stat-value {
        font-size: 18px;
        font-weight: bold;
        margin-bottom: 4px;
      }
      
      .stat-label {
        font-size: 11px;
        opacity: 0.9;
      }
    }
  }
  
  .log-stream {
    .log-container {
      height: 500px;
      overflow-y: auto;
      border: 1px solid #e4e7ed;
      border-radius: 4px;
      background-color: #1a1a1a;
      color: #e2e8f0;
      font-family: 'Courier New', monospace;
      font-size: 13px;
      
      .log-entry {
        border-bottom: 1px solid rgba(255, 255, 255, 0.1);
        
        &:hover {
          background-color: rgba(255, 255, 255, 0.05);
        }
        
        &.log-entry--debug {
          border-left: 3px solid #718096;
        }
        
        &.log-entry--info {
          border-left: 3px solid #48bb78;
        }
        
        &.log-entry--warn {
          border-left: 3px solid #ed8936;
        }
        
        &.log-entry--error,
        &.log-entry--critical {
          border-left: 3px solid #f56565;
          background-color: rgba(245, 101, 101, 0.1);
        }
        
        .log-line {
          padding: 8px 12px;
          
          .log-meta {
            display: flex;
            align-items: center;
            gap: 8px;
            margin-bottom: 4px;
            font-size: 11px;
            
            .log-timestamp {
              color: #a0aec0;
              min-width: 150px;
            }
            
            .log-level {
              min-width: 60px;
            }
            
            .log-source {
              color: #63b3ed;
              background-color: rgba(99, 179, 237, 0.2);
              padding: 2px 6px;
              border-radius: 3px;
              font-size: 10px;
            }
            
            .log-event-type {
              color: #f6ad55;
              font-weight: 500;
            }
          }
          
          .log-content {
            .log-message {
              line-height: 1.4;
              word-break: break-word;
              
              :deep(mark) {
                background-color: #ffd700;
                color: #000;
                padding: 1px 2px;
                border-radius: 2px;
              }
              
              :deep(.highlight-ip) {
                color: #63b3ed;
                font-weight: 500;
              }
              
              :deep(.highlight-error) {
                color: #f56565;
                font-weight: 500;
              }
            }
            
            .log-context,
            .log-stack {
              margin-top: 8px;
              
              .context-content,
              .stack-content {
                background-color: #2d3748;
                color: #e2e8f0;
                padding: 8px;
                border-radius: 4px;
                font-size: 11px;
                margin: 0;
                line-height: 1.3;
              }
            }
          }
        }
      }
      
      .empty-state {
        display: flex;
        flex-direction: column;
        align-items: center;
        justify-content: center;
        height: 100%;
        color: #a0aec0;
        
        .empty-text {
          margin-top: 16px;
          font-size: 16px;
          font-weight: 500;
        }
        
        .empty-subtext {
          margin-top: 8px;
          font-size: 14px;
          color: #718096;
        }
      }
    }
  }
  
  .controls-bar {
    padding: 12px 0;
    border-top: 1px solid #e4e7ed;
    
    .text-sm {
      font-size: 0.875rem;
    }
    
    .text-gray-500 {
      color: #6b7280;
    }
  }
}

.text-lg {
  font-size: 1.125rem;
}

.font-semibold {
  font-weight: 600;
}

.flex {
  display: flex;
  
  &.justify-between {
    justify-content: space-between;
  }
  
  &.items-center {
    align-items: center;
  }
}

.space-x-2 > * + * {
  margin-left: 0.5rem;
}

.space-x-4 > * + * {
  margin-left: 1rem;
}

.mr-2 {
  margin-right: 0.5rem;
}

.ml-2 {
  margin-left: 0.5rem;
}

.mb-4 {
  margin-bottom: 1rem;
}

.mt-4 {
  margin-top: 1rem;
}
</style> 