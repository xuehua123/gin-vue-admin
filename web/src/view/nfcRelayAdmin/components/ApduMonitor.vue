<!--
  APDU命令监控组件
  实时监控和分析APDU命令流
-->
<template>
  <div class="apdu-monitor">
    <el-card shadow="never">
      <template #header>
        <div class="flex justify-between items-center">
          <div class="flex items-center">
            <el-icon class="mr-2" color="#409eff">
              <Monitor />
            </el-icon>
            <span class="text-lg font-semibold">APDU命令监控</span>
            <el-tag 
              :type="isMonitoring ? 'success' : 'info'" 
              size="small" 
              class="ml-2"
            >
              {{ isMonitoring ? '监控中' : '已停止' }}
            </el-tag>
          </div>
          <div class="flex items-center space-x-2">
            <el-button 
              :type="isMonitoring ? 'danger' : 'success'" 
              size="small"
              @click="toggleMonitoring"
            >
              <el-icon>
                <VideoPlay v-if="!isMonitoring" />
                <VideoPause v-else />
              </el-icon>
              {{ isMonitoring ? '停止监控' : '开始监控' }}
            </el-button>
            <el-button size="small" @click="clearLogs">
              <el-icon><Delete /></el-icon>
              清空日志
            </el-button>
            <el-button size="small" @click="exportLogs">
              <el-icon><Download /></el-icon>
              导出
            </el-button>
          </div>
        </div>
      </template>

      <!-- 统计面板 -->
      <div class="stats-panel mb-4">
        <el-row :gutter="16">
          <el-col :span="6">
            <div class="stat-card">
              <div class="stat-value">{{ totalCommands }}</div>
              <div class="stat-label">总命令数</div>
            </div>
          </el-col>
          <el-col :span="6">
            <div class="stat-card">
              <div class="stat-value">{{ successRate }}%</div>
              <div class="stat-label">成功率</div>
            </div>
          </el-col>
          <el-col :span="6">
            <div class="stat-card">
              <div class="stat-value">{{ avgResponseTime }}ms</div>
              <div class="stat-label">平均响应时间</div>
            </div>
          </el-col>
          <el-col :span="6">
            <div class="stat-card">
              <div class="stat-value">{{ activeStreams }}</div>
              <div class="stat-label">活跃流</div>
            </div>
          </el-col>
        </el-row>
      </div>

      <!-- 过滤器和控制 -->
      <div class="filter-panel mb-4">
        <el-row :gutter="16">
          <el-col :span="6">
            <el-select 
              v-model="filters.sessionId" 
              placeholder="选择会话" 
              clearable
              filterable
            >
              <el-option 
                v-for="session in activeSessions" 
                :key="session.id" 
                :label="session.name" 
                :value="session.id" 
              />
            </el-select>
          </el-col>
          <el-col :span="6">
            <el-select 
              v-model="filters.commandType" 
              placeholder="命令类型" 
              clearable
              multiple
            >
              <el-option label="SELECT" value="SELECT" />
              <el-option label="READ BINARY" value="READ_BINARY" />
              <el-option label="UPDATE BINARY" value="UPDATE_BINARY" />
              <el-option label="GET RESPONSE" value="GET_RESPONSE" />
              <el-option label="VERIFY PIN" value="VERIFY_PIN" />
              <el-option label="CHANGE PIN" value="CHANGE_PIN" />
              <el-option label="GENERATE AC" value="GENERATE_AC" />
              <el-option label="自定义" value="CUSTOM" />
            </el-select>
          </el-col>
          <el-col :span="6">
            <el-select 
              v-model="filters.direction" 
              placeholder="数据流向" 
              clearable
            >
              <el-option label="上行 (Receiver → Provider)" value="upstream" />
              <el-option label="下行 (Provider → Receiver)" value="downstream" />
            </el-select>
          </el-col>
          <el-col :span="6">
            <el-select 
              v-model="filters.status" 
              placeholder="状态" 
              clearable
              multiple
            >
              <el-option label="成功" value="success" />
              <el-option label="错误" value="error" />
              <el-option label="超时" value="timeout" />
              <el-option label="处理中" value="processing" />
            </el-select>
          </el-col>
        </el-row>
      </div>

      <!-- APDU流展示 -->
      <div class="apdu-stream">
        <!-- 实时流模式 -->
        <div v-if="viewMode === 'stream'" class="stream-view">
          <div class="stream-header mb-2">
            <div class="flex justify-between items-center">
              <span class="text-sm font-medium">实时APDU流</span>
              <div class="flex items-center space-x-2">
                <el-switch 
                  v-model="autoScroll" 
                  active-text="自动滚动" 
                  size="small"
                />
                <el-button size="small" @click="pauseStream">
                  <el-icon><VideoPause /></el-icon>
                  {{ streamPaused ? '恢复' : '暂停' }}
                </el-button>
              </div>
            </div>
          </div>
          
          <div ref="streamContainer" class="stream-container" @scroll="handleScroll">
            <div 
              v-for="(command, index) in filteredCommands" 
              :key="command.id"
              class="stream-item"
              :class="getStreamItemClass(command)"
            >
              <div class="stream-meta">
                <span class="timestamp">{{ formatTimestamp(command.timestamp) }}</span>
                <span class="session-id">{{ command.sessionId }}</span>
                <el-tag 
                  :type="getDirectionTagType(command.direction)" 
                  size="small"
                  class="direction-tag"
                >
                  {{ command.direction === 'upstream' ? '↑' : '↓' }}
                </el-tag>
              </div>
              
              <div class="stream-content">
                <div class="apdu-header">
                  <span class="command-type">{{ command.type }}</span>
                  <span class="status" :class="getStatusClass(command.status)">
                    {{ getStatusText(command.status) }}
                  </span>
                  <span class="response-time">{{ command.responseTime }}ms</span>
                </div>
                
                <div class="apdu-data">
                  <div class="hex-display">
                    <div class="hex-line">
                      <span class="hex-label">CMD:</span>
                      <span class="hex-content">{{ formatHex(command.command) }}</span>
                    </div>
                    <div v-if="command.response" class="hex-line">
                      <span class="hex-label">RSP:</span>
                      <span class="hex-content">{{ formatHex(command.response) }}</span>
                    </div>
                  </div>
                  
                  <div v-if="command.parsed" class="parsed-display">
                    <el-collapse accordion>
                      <el-collapse-item title="解析结果" name="parsed">
                        <pre>{{ JSON.stringify(command.parsed, null, 2) }}</pre>
                      </el-collapse-item>
                    </el-collapse>
                  </div>
                </div>
              </div>
            </div>
          </div>
        </div>

        <!-- 表格模式 -->
        <div v-else class="table-view">
          <el-table 
            :data="filteredCommands" 
            style="width: 100%" 
            height="600"
            :default-sort="{ prop: 'timestamp', order: 'descending' }"
            @row-click="handleRowClick"
          >
            <el-table-column prop="timestamp" label="时间" width="180" sortable>
              <template #default="{ row }">
                {{ formatTimestamp(row.timestamp) }}
              </template>
            </el-table-column>
            
            <el-table-column prop="sessionId" label="会话ID" width="200" show-overflow-tooltip />
            
            <el-table-column prop="direction" label="方向" width="100" align="center">
              <template #default="{ row }">
                <el-tag :type="getDirectionTagType(row.direction)" size="small">
                  {{ row.direction === 'upstream' ? '上行' : '下行' }}
                </el-tag>
              </template>
            </el-table-column>
            
            <el-table-column prop="type" label="命令类型" width="150" />
            
            <el-table-column prop="command" label="命令数据" min-width="200" show-overflow-tooltip>
              <template #default="{ row }">
                <code class="hex-display">{{ formatHex(row.command) }}</code>
              </template>
            </el-table-column>
            
            <el-table-column prop="status" label="状态" width="100" align="center">
              <template #default="{ row }">
                <el-tag :type="getStatusTagType(row.status)" size="small">
                  {{ getStatusText(row.status) }}
                </el-tag>
              </template>
            </el-table-column>
            
            <el-table-column prop="responseTime" label="响应时间" width="120" align="right">
              <template #default="{ row }">
                {{ row.responseTime }}ms
              </template>
            </el-table-column>
            
            <el-table-column label="操作" width="120" fixed="right" align="center">
              <template #default="{ row }">
                <el-button link type="primary" size="small" @click="showCommandDetails(row)">
                  详情
                </el-button>
              </template>
            </el-table-column>
          </el-table>
        </div>
      </div>

      <!-- 视图切换 -->
      <div class="view-controls mt-4">
        <el-radio-group v-model="viewMode" size="small">
          <el-radio-button value="stream">实时流</el-radio-button>
          <el-radio-button value="table">表格</el-radio-button>
        </el-radio-group>
      </div>
    </el-card>

    <!-- 命令详情弹窗 -->
    <command-detail-modal 
      v-model:visible="detailModalVisible" 
      :command="selectedCommand" 
    />
  </div>
</template>

<script setup>
import { ref, reactive, computed, onMounted, onUnmounted, nextTick } from 'vue'
import { ElMessage } from 'element-plus'
import {
  Monitor,
  VideoPlay,
  VideoPause,
  Delete,
  Download
} from '@element-plus/icons-vue'
import CommandDetailModal from './CommandDetailModal.vue'
import { API_CONFIG, WEBSOCKET_CONFIG, MESSAGE_TYPES } from '../constants.js'

const props = defineProps({
  sessionId: {
    type: String,
    default: ''
  }
})

// 响应式数据
const isMonitoring = ref(false)
const streamPaused = ref(false)
const autoScroll = ref(true)
const viewMode = ref('stream')
const detailModalVisible = ref(false)
const selectedCommand = ref(null)
const streamContainer = ref(null)

// 统计数据
const totalCommands = ref(0)
const successRate = ref(0)
const avgResponseTime = ref(0)
const activeStreams = ref(0)

// 过滤器
const filters = reactive({
  sessionId: '',
  commandType: [],
  direction: '',
  status: []
})

// 模拟数据
const apduCommands = ref([])
const activeSessions = ref([
  { id: 'session-1', name: 'Session 1 (Provider: iPhone, Receiver: POS)' },
  { id: 'session-2', name: 'Session 2 (Provider: Android, Receiver: ATM)' }
])

// 计算属性
const filteredCommands = computed(() => {
  let commands = apduCommands.value
  
  if (filters.sessionId) {
    commands = commands.filter(cmd => cmd.sessionId === filters.sessionId)
  }
  
  if (filters.commandType.length > 0) {
    commands = commands.filter(cmd => filters.commandType.includes(cmd.type))
  }
  
  if (filters.direction) {
    commands = commands.filter(cmd => cmd.direction === filters.direction)
  }
  
  if (filters.status.length > 0) {
    commands = commands.filter(cmd => filters.status.includes(cmd.status))
  }
  
  return commands
})

// WebSocket连接
let ws = null

// 方法
const toggleMonitoring = () => {
  if (isMonitoring.value) {
    stopMonitoring()
  } else {
    startMonitoring()
  }
}

const startMonitoring = () => {
  isMonitoring.value = true
  streamPaused.value = false
  
  // 建立WebSocket连接
  connectWebSocket()
  
  // 模拟数据生成
  startMockDataGeneration()
  
  ElMessage.success('APDU监控已启动')
}

const stopMonitoring = () => {
  isMonitoring.value = false
  
  // 关闭WebSocket连接
  if (ws) {
    ws.close()
    ws = null
  }
  
  ElMessage.info('APDU监控已停止')
}

const connectWebSocket = () => {
  try {
    // 使用统一的WebSocket配置
    const wsUrl = API_CONFIG.WEBSOCKET.getUrl(API_CONFIG.WEBSOCKET.ENDPOINTS.REALTIME)
    console.log('连接APDU监控WebSocket:', wsUrl)
    
    ws = new WebSocket(wsUrl)
    
    ws.onopen = () => {
      console.log('APDU监控WebSocket连接已建立')
      // 订阅APDU数据
      ws.send(JSON.stringify({
        type: MESSAGE_TYPES.SUBSCRIBE,
        topic: 'apdu'
      }))
    }
    
    ws.onmessage = (event) => {
      try {
        const message = JSON.parse(event.data)
        
        // 处理不同类型的消息
        if (message.type === MESSAGE_TYPES.APDU_DATA) {
          addApduEntry(message.data)
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
      console.log('APDU监控WebSocket连接已关闭')
    }
    
    ws.onerror = (error) => {
      console.error('APDU监控WebSocket错误:', error)
    }
  } catch (error) {
    console.error('无法建立APDU监控WebSocket连接:', error)
  }
}

const addCommand = (command) => {
  if (!streamPaused.value) {
    apduCommands.value.unshift(command)
    
    // 限制最大命令数量
    if (apduCommands.value.length > 1000) {
      apduCommands.value = apduCommands.value.slice(0, 1000)
    }
    
    updateStats()
    
    if (autoScroll.value && viewMode.value === 'stream') {
      nextTick(() => {
        scrollToTop()
      })
    }
  }
}

const updateStats = () => {
  totalCommands.value = apduCommands.value.length
  
  if (totalCommands.value > 0) {
    const successCount = apduCommands.value.filter(cmd => cmd.status === 'success').length
    successRate.value = Math.round((successCount / totalCommands.value) * 100)
    
    const totalResponseTime = apduCommands.value.reduce((sum, cmd) => sum + cmd.responseTime, 0)
    avgResponseTime.value = Math.round(totalResponseTime / totalCommands.value)
    
    const uniqueSessions = new Set(apduCommands.value.map(cmd => cmd.sessionId))
    activeStreams.value = uniqueSessions.size
  }
}

const clearLogs = () => {
  apduCommands.value = []
  totalCommands.value = 0
  successRate.value = 0
  avgResponseTime.value = 0
  activeStreams.value = 0
  ElMessage.success('日志已清空')
}

const exportLogs = () => {
  const data = JSON.stringify(filteredCommands.value, null, 2)
  const blob = new Blob([data], { type: 'application/json' })
  const url = URL.createObjectURL(blob)
  const a = document.createElement('a')
  a.href = url
  a.download = `apdu-logs-${new Date().toISOString().slice(0, 19)}.json`
  a.click()
  URL.revokeObjectURL(url)
  ElMessage.success('日志已导出')
}

const pauseStream = () => {
  streamPaused.value = !streamPaused.value
  ElMessage.info(streamPaused.value ? '流已暂停' : '流已恢复')
}

const scrollToTop = () => {
  if (streamContainer.value) {
    streamContainer.value.scrollTop = 0
  }
}

const handleScroll = () => {
  // 如果用户手动滚动，暂时禁用自动滚动
  if (autoScroll.value && streamContainer.value) {
    if (streamContainer.value.scrollTop > 50) {
      autoScroll.value = false
    }
  }
}

const handleRowClick = (row) => {
  showCommandDetails(row)
}

const showCommandDetails = (command) => {
  selectedCommand.value = command
  detailModalVisible.value = true
}

// 格式化和样式方法
const formatTimestamp = (timestamp) => {
  return new Date(timestamp).toLocaleString()
}

const formatHex = (hexString) => {
  if (!hexString) return ''
  return hexString.replace(/(.{2})/g, '$1 ').trim().toUpperCase()
}

const getStreamItemClass = (command) => {
  return {
    'stream-item--success': command.status === 'success',
    'stream-item--error': command.status === 'error',
    'stream-item--timeout': command.status === 'timeout',
    'stream-item--processing': command.status === 'processing'
  }
}

const getDirectionTagType = (direction) => {
  return direction === 'upstream' ? 'warning' : 'success'
}

const getStatusClass = (status) => {
  return {
    'status--success': status === 'success',
    'status--error': status === 'error',
    'status--timeout': status === 'timeout',
    'status--processing': status === 'processing'
  }
}

const getStatusTagType = (status) => {
  switch (status) {
    case 'success': return 'success'
    case 'error': return 'danger'
    case 'timeout': return 'warning'
    case 'processing': return 'info'
    default: return 'info'
  }
}

const getStatusText = (status) => {
  switch (status) {
    case 'success': return '成功'
    case 'error': return '错误'
    case 'timeout': return '超时'
    case 'processing': return '处理中'
    default: return status
  }
}

// 模拟数据生成
const startMockDataGeneration = () => {
  const interval = setInterval(() => {
    if (!isMonitoring.value) {
      clearInterval(interval)
      return
    }
    
    const mockCommand = generateMockCommand()
    addCommand(mockCommand)
  }, Math.random() * 2000 + 500) // 500ms - 2.5s随机间隔
}

const generateMockCommand = () => {
  const commandTypes = ['SELECT', 'READ_BINARY', 'UPDATE_BINARY', 'GET_RESPONSE', 'VERIFY_PIN']
  const directions = ['upstream', 'downstream']
  const statuses = ['success', 'error', 'timeout', 'processing']
  const sessionIds = ['session-1', 'session-2']
  
  const status = statuses[Math.floor(Math.random() * statuses.length)]
  const responseTime = Math.floor(Math.random() * 1000) + 10
  
  return {
    id: `cmd-${Date.now()}-${Math.random().toString(36).substr(2, 9)}`,
    timestamp: new Date().toISOString(),
    sessionId: sessionIds[Math.floor(Math.random() * sessionIds.length)],
    direction: directions[Math.floor(Math.random() * directions.length)],
    type: commandTypes[Math.floor(Math.random() * commandTypes.length)],
    command: generateRandomHex(8),
    response: status === 'success' ? generateRandomHex(4) + '9000' : (status === 'error' ? generateRandomHex(4) + '6F00' : null),
    status: status,
    responseTime: responseTime,
    parsed: status === 'success' ? {
      cla: '00',
      ins: '84',
      p1: '00',
      p2: '00',
      le: '08',
      sw1: '90',
      sw2: '00'
    } : null
  }
}

const generateRandomHex = (length) => {
  const hex = '0123456789ABCDEF'
  let result = ''
  for (let i = 0; i < length; i++) {
    result += hex.charAt(Math.floor(Math.random() * hex.length))
  }
  return result
}

// 生命周期
onMounted(() => {
  if (props.sessionId) {
    filters.sessionId = props.sessionId
  }
})

onUnmounted(() => {
  if (ws) {
    ws.close()
  }
})
</script>

<style scoped lang="scss">
.apdu-monitor {
  .stats-panel {
    .stat-card {
      text-align: center;
      padding: 16px;
      background: linear-gradient(135deg, #667eea 0%, #764ba2 100%);
      border-radius: 8px;
      color: white;
      
      .stat-value {
        font-size: 24px;
        font-weight: bold;
        margin-bottom: 4px;
      }
      
      .stat-label {
        font-size: 12px;
        opacity: 0.8;
      }
    }
  }
  
  .filter-panel {
    padding: 16px;
    background-color: #f5f7fa;
    border-radius: 8px;
    border: 1px solid #e4e7ed;
  }
  
  .apdu-stream {
    .stream-view {
      .stream-container {
        max-height: 600px;
        overflow-y: auto;
        border: 1px solid #e4e7ed;
        border-radius: 4px;
        background-color: #fafafa;
        
        .stream-item {
          padding: 12px;
          border-bottom: 1px solid #ebeef5;
          transition: all 0.3s ease;
          
          &:hover {
            background-color: #f0f9ff;
          }
          
          &.stream-item--success {
            border-left: 3px solid #67c23a;
          }
          
          &.stream-item--error {
            border-left: 3px solid #f56c6c;
          }
          
          &.stream-item--timeout {
            border-left: 3px solid #e6a23c;
          }
          
          &.stream-item--processing {
            border-left: 3px solid #409eff;
          }
          
          .stream-meta {
            display: flex;
            align-items: center;
            gap: 12px;
            margin-bottom: 8px;
            font-size: 12px;
            color: #606266;
            
            .timestamp {
              font-family: monospace;
            }
            
            .session-id {
              font-family: monospace;
              background-color: #e1f3ff;
              padding: 2px 6px;
              border-radius: 3px;
            }
            
            .direction-tag {
              font-weight: bold;
            }
          }
          
          .stream-content {
            .apdu-header {
              display: flex;
              align-items: center;
              gap: 12px;
              margin-bottom: 8px;
              
              .command-type {
                font-weight: 600;
                color: #303133;
              }
              
              .status {
                font-size: 12px;
                padding: 2px 6px;
                border-radius: 3px;
                
                &.status--success {
                  background-color: #f0f9ff;
                  color: #67c23a;
                }
                
                &.status--error {
                  background-color: #fef0f0;
                  color: #f56c6c;
                }
                
                &.status--timeout {
                  background-color: #fdf6ec;
                  color: #e6a23c;
                }
                
                &.status--processing {
                  background-color: #e1f3ff;
                  color: #409eff;
                }
              }
              
              .response-time {
                font-size: 12px;
                color: #909399;
                margin-left: auto;
              }
            }
            
            .apdu-data {
              .hex-display {
                background-color: #2d3748;
                color: #e2e8f0;
                padding: 8px 12px;
                border-radius: 4px;
                font-family: 'Courier New', monospace;
                font-size: 13px;
                line-height: 1.5;
                
                .hex-line {
                  margin-bottom: 4px;
                  
                  .hex-label {
                    color: #90cdf4;
                    margin-right: 8px;
                    font-weight: bold;
                  }
                  
                  .hex-content {
                    letter-spacing: 1px;
                  }
                }
              }
              
              .parsed-display {
                margin-top: 8px;
                
                pre {
                  background-color: #f8f9fa;
                  padding: 8px;
                  border-radius: 4px;
                  font-size: 12px;
                  margin: 0;
                }
              }
            }
          }
        }
      }
    }
    
    .table-view {
      .hex-display {
        font-family: 'Courier New', monospace;
        font-size: 12px;
        background-color: #f5f7fa;
        padding: 2px 4px;
        border-radius: 3px;
      }
    }
  }
  
  .view-controls {
    text-align: center;
  }
}

.text-lg {
  font-size: 1.125rem;
}

.font-semibold {
  font-weight: 600;
}

.font-medium {
  font-weight: 500;
}

.text-sm {
  font-size: 0.875rem;
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

.mr-2 {
  margin-right: 0.5rem;
}

.ml-2 {
  margin-left: 0.5rem;
}

.mb-2 {
  margin-bottom: 0.5rem;
}

.mb-4 {
  margin-bottom: 1rem;
}

.mt-4 {
  margin-top: 1rem;
}
</style> 