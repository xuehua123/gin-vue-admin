<!--
  NFC中继概览仪表盘 - 实时监控大屏
  支持实时数据更新、全屏监控、数据变化动画
-->
<template>
  <fullscreen-layout
    title="NFC中继监控大屏"
    subtitle="实时数据监控与系统状态展示"
    :connection-status="connectionStatus"
    :last-update-time="lastUpdateTime"
    :refresh-rate="30"
    @refresh="handleRefresh"
    @export="handleExport"
    @screenshot="handleScreenshot"
    @fullscreen-change="handleFullscreenChange"
    @theme-change="handleThemeChange"
  >
    <div class="dashboard-content" :class="{ 'content-fullscreen': isFullscreen }">
      <!-- 连接状态警告 -->
      <el-alert
        v-if="!isOnline"
        title="实时连接已断开"
        description="正在尝试重新连接到服务器，当前显示的可能不是最新数据"
        type="warning"
        :closable="false"
        show-icon
        class="connection-alert"
      />
      
      <!-- 核心指标卡片区域 -->
      <div class="metrics-grid">
        <realtime-stat-card
          title="运行状态"
          :value="statusText"
          :icon="statusIcon"
          :icon-color="statusColor"
          :connection-status="connectionStatus"
          :last-update-time="lastUpdateTime"
          :change-info="getChangeAnimation('dashboard', 'hub_status')"
          clickable
          @click="routeHelper.toConfiguration()"
        />
        
        <realtime-stat-card
          title="活动连接"
          :value="dashboardData.active_connections"
          subtitle="当前在线客户端"
          :icon="ConnectionIcon"
          icon-color="#67c23a"
          :connection-status="connectionStatus"
          :last-update-time="lastUpdateTime"
          :change-info="getChangeAnimation('dashboard', 'active_connections')"
          :extra-info="[
            { label: 'Provider', value: clientsData.provider_count || 0, color: '#67c23a' },
            { label: 'Receiver', value: clientsData.receiver_count || 0, color: '#e6a23c' }
          ]"
          clickable
          @click="routeHelper.toClientManagement()"
        />
        
        <realtime-stat-card
          title="活动会话"
          :value="dashboardData.active_sessions"
          subtitle="正在进行的NFC中继"
          :icon="ChatDotRoundIcon"
          icon-color="#e6a23c"
          :connection-status="connectionStatus"
          :last-update-time="lastUpdateTime"
          :change-info="getChangeAnimation('dashboard', 'active_sessions')"
          :extra-info="[
            { label: '已配对', value: sessionsData.paired_count || 0, color: '#67c23a' },
            { label: '等待中', value: sessionsData.waiting_count || 0, color: '#909399' }
          ]"
          clickable
          @click="routeHelper.toSessionManagement()"
        />
        
        <realtime-stat-card
          title="APDU中继"
          :value="dashboardData.apdu_relayed_last_minute"
          subtitle="最近1分钟转发数量"
          :icon="DataLineIcon"
          icon-color="#409eff"
          :connection-status="connectionStatus"
          :last-update-time="lastUpdateTime"
          :change-info="getChangeAnimation('dashboard', 'apdu_relayed_last_minute')"
          :extra-info="[
            { label: '错误数/小时', value: dashboardData.apdu_errors_last_hour || 0, color: dashboardData.apdu_errors_last_hour > 0 ? '#f56c6c' : '#67c23a' }
          ]"
        />
      </div>

      <!-- 趋势图表区域 -->
      <div class="charts-section">
        <div class="chart-grid">
          <!-- 连接数趋势 -->
          <el-card shadow="hover" class="trend-card">
            <template #header>
              <div class="card-header">
                <div class="header-left">
                  <el-icon class="chart-icon" color="#67c23a">
                    <TrendCharts />
                  </el-icon>
                  <span class="chart-title">连接数趋势</span>
                  <el-tag size="small" class="realtime-tag">实时</el-tag>
                </div>
                <div class="header-right">
                  <el-tooltip content="当前连接数" placement="top">
                    <span class="current-value">{{ dashboardData.active_connections }}</span>
                  </el-tooltip>
                </div>
              </div>
            </template>
            <trend-chart
              :data="dashboardData.connection_trend || []"
              height="200px"
              color="#67c23a"
              :smooth="true"
            />
          </el-card>

          <!-- 会话数趋势 -->
          <el-card shadow="hover" class="trend-card">
            <template #header>
              <div class="card-header">
                <div class="header-left">
                  <el-icon class="chart-icon" color="#e6a23c">
                    <TrendCharts />
                  </el-icon>
                  <span class="chart-title">会话数趋势</span>
                  <el-tag size="small" class="realtime-tag">实时</el-tag>
                </div>
                <div class="header-right">
                  <el-tooltip content="当前会话数" placement="top">
                    <span class="current-value">{{ dashboardData.active_sessions }}</span>
                  </el-tooltip>
                </div>
              </div>
            </template>
            <trend-chart
              :data="dashboardData.session_trend || []"
              height="200px"
              color="#e6a23c"
              :smooth="true"
            />
          </el-card>
        </div>
      </div>

      <!-- 详细信息和系统状态 -->
      <div class="details-section">
        <div class="details-grid">
          <!-- 系统性能监控 -->
          <el-card shadow="hover" class="performance-card">
            <template #header>
              <div class="card-header">
                <el-icon class="section-icon" color="#409eff">
                  <Monitor />
                </el-icon>
                <span class="section-title">系统性能</span>
              </div>
            </template>
            
            <div class="performance-metrics">
              <div class="metric-item">
                <div class="metric-label">平均响应时间</div>
                <div class="metric-value">
                  <span class="value-number">{{ dashboardData.avg_response_time || 45 }}</span>
                  <span class="value-unit">ms</span>
                </div>
                <div class="metric-bar">
                  <el-progress 
                    :percentage="Math.min((dashboardData.avg_response_time || 45) / 200 * 100, 100)" 
                    :color="responseTimeColor"
                    :stroke-width="6"
                    :show-text="false"
                  />
                </div>
              </div>
              
              <div class="metric-item">
                <div class="metric-label">系统负载</div>
                <div class="metric-value">
                  <span class="value-number">{{ dashboardData.system_load || 35 }}</span>
                  <span class="value-unit">%</span>
                </div>
                <div class="metric-bar">
                  <el-progress 
                    :percentage="dashboardData.system_load || 35" 
                    :color="loadColors"
                    :stroke-width="6"
                    :show-text="false"
                  />
                </div>
              </div>
              
              <div class="metric-item">
                <div class="metric-label">内存使用</div>
                <div class="metric-value">
                  <span class="value-number">{{ dashboardData.memory_usage || 68 }}</span>
                  <span class="value-unit">%</span>
                </div>
                <div class="metric-bar">
                  <el-progress 
                    :percentage="dashboardData.memory_usage || 68" 
                    :color="loadColors"
                    :stroke-width="6"
                    :show-text="false"
                  />
                </div>
              </div>
            </div>
          </el-card>

          <!-- 实时事件日志 -->
          <el-card shadow="hover" class="events-card">
            <template #header>
              <div class="card-header">
                <el-icon class="section-icon" color="#f56c6c">
                  <Document />
                </el-icon>
                <span class="section-title">实时事件</span>
                <el-button 
                  link 
                  type="primary" 
                  size="small"
                  @click="routeHelper.toAuditLogs()"
                >
                  查看全部
                </el-button>
              </div>
            </template>
            
            <div class="events-list" v-if="recentEvents.length > 0">
              <div 
                class="event-item" 
                v-for="event in recentEvents" 
                :key="event.id"
                :class="`event-${event.type}`"
              >
                <div class="event-icon">
                  <el-icon>
                    <component :is="getEventIcon(event.type)" />
                  </el-icon>
                </div>
                <div class="event-content">
                  <div class="event-title">{{ event.title }}</div>
                  <div class="event-time">{{ formatTime(event.time) }}</div>
                </div>
                <div class="event-status" :class="event.type">
                  {{ getEventStatusText(event.type) }}
                </div>
              </div>
            </div>
            
            <el-empty 
              v-else 
              description="暂无实时事件" 
              :image-size="80"
            />
          </el-card>
        </div>
      </div>
    </div>
  </fullscreen-layout>
</template>

<script setup>
import { ref, computed, onMounted, onUnmounted } from 'vue'
import { useRouter } from 'vue-router'
import { 
  Connection as ConnectionIcon,
  ChatDotRound as ChatDotRoundIcon,
  DataLine as DataLineIcon,
  CircleCheck,
  CircleClose,
  TrendCharts,
  Monitor,
  Document,
  User,
  ChatDotRound,
  Setting,
  Link,
  Close
} from '@element-plus/icons-vue'
import { ElMessage } from 'element-plus'
import { useRealtimeData } from '../utils/realtimeDataManager'
import { FullscreenLayout, RealtimeStatCard, TrendChart } from '../components'
import { createRouteHelper } from '../utils/routeHelper'
import { formatTime } from '@/utils/format'

defineOptions({
  name: 'NfcRelayDashboard'
})

const router = useRouter()
const routeHelper = createRouteHelper(router)

// 实时数据管理
const {
  isConnected,
  connectionStatus,
  lastUpdateTime,
  dashboardData,
  clientsData,
  sessionsData,
  connect,
  disconnect,
  on,
  off,
  getChangeAnimation,
  isOnline
} = useRealtimeData()

// 组件状态
const isFullscreen = ref(false)
const recentEvents = ref([])

// 计算属性
const statusText = computed(() => {
  return dashboardData.hub_status === 'online' ? '在线' : '离线'
})

const statusIcon = computed(() => {
  return dashboardData.hub_status === 'online' ? CircleCheck : CircleClose
})

const statusColor = computed(() => {
  return dashboardData.hub_status === 'online' ? '#67c23a' : '#f56c6c'
})

const responseTimeColor = computed(() => {
  const time = dashboardData.avg_response_time || 45
  if (time < 50) return '#67c23a'
  if (time < 100) return '#e6a23c'
  return '#f56c6c'
})

const loadColors = [
  { color: '#67c23a', percentage: 30 },
  { color: '#e6a23c', percentage: 70 },
  { color: '#f56c6c', percentage: 100 }
]

// 事件处理
const handleRefresh = async () => {
  try {
    await connect()
    ElMessage.success('数据已刷新')
  } catch (error) {
    ElMessage.error('刷新失败: ' + error.message)
  }
}

const handleExport = () => {
  ElMessage.info('导出功能开发中...')
}

const handleScreenshot = () => {
  ElMessage.info('截图功能开发中...')
}

const handleFullscreenChange = (fullscreen) => {
  isFullscreen.value = fullscreen
}

const handleThemeChange = (isDark) => {
  console.log('主题切换:', isDark ? '深色' : '浅色')
}

// 实时事件相关
const getEventIcon = (type) => {
  switch (type) {
    case 'connect': return User
    case 'disconnect': return Close
    case 'session': return ChatDotRound
    case 'error': return CircleClose
    default: return CircleCheck
  }
}

const getEventStatusText = (type) => {
  switch (type) {
    case 'connect': return '连接'
    case 'disconnect': return '断开'
    case 'session': return '会话'
    case 'error': return '错误'
    default: return '信息'
  }
}

const addRecentEvent = (title, type = 'info') => {
  const event = {
    id: Date.now(),
    title,
    type,
    time: new Date()
  }
  
  recentEvents.value.unshift(event)
  
  // 只保留最近10条事件
  if (recentEvents.value.length > 10) {
    recentEvents.value = recentEvents.value.slice(0, 10)
  }
}

// 实时事件监听
const setupEventListeners = () => {
  // 监听客户端连接事件
  on('client_connected', (data) => {
    addRecentEvent(`客户端 ${data.display_name || data.client_id} 已连接`, 'connect')
  })
  
  // 监听客户端断开事件
  on('client_disconnected', (data) => {
    addRecentEvent(`客户端 ${data.display_name || data.client_id} 已断开`, 'disconnect')
  })
  
  // 监听会话创建事件
  on('session_created', (data) => {
    addRecentEvent(`新会话 ${data.session_id} 已建立`, 'session')
  })
  
  // 监听会话终止事件
  on('session_terminated', (data) => {
    addRecentEvent(`会话 ${data.session_id} 已终止`, 'disconnect')
  })
}

const removeEventListeners = () => {
  off('client_connected')
  off('client_disconnected')
  off('session_created')
  off('session_terminated')
}

// 生命周期
onMounted(async () => {
  try {
    // 连接实时数据
    await connect()
    
    // 设置事件监听
    setupEventListeners()
    
    ElMessage.success('实时监控已启动')
  } catch (error) {
    console.error('Failed to connect to realtime service:', error)
    ElMessage.warning('实时连接失败，将显示模拟数据')
    
    // 生成模拟事件
    setTimeout(() => {
      addRecentEvent('客户端 iPhone-123 已连接', 'connect')
    }, 1000)
    
    setTimeout(() => {
      addRecentEvent('会话 session-abc 已建立', 'session')
    }, 3000)
  }
})

onUnmounted(() => {
  removeEventListeners()
  disconnect()
})
</script>

<style scoped lang="scss">
.dashboard-content {
  display: flex;
  flex-direction: column;
  gap: 20px;
  min-height: 100%;
  
  &.content-fullscreen {
    gap: 15px;
  }
  
  .connection-alert {
    margin-bottom: 16px;
  }
  
  .metrics-grid {
    display: grid;
    grid-template-columns: repeat(auto-fit, minmax(280px, 1fr));
    gap: 16px;
  }
  
  .charts-section {
    .chart-grid {
      display: grid;
      grid-template-columns: repeat(auto-fit, minmax(400px, 1fr));
      gap: 16px;
    }
    
    .trend-card {
      .card-header {
        display: flex;
        justify-content: space-between;
        align-items: center;
        
        .header-left {
          display: flex;
          align-items: center;
          gap: 8px;
          
          .chart-icon {
            font-size: 16px;
          }
          
          .chart-title {
            font-weight: 600;
            font-size: 14px;
          }
          
          .realtime-tag {
            background: linear-gradient(45deg, #67c23a, #85ce61);
            color: white;
            border: none;
            animation: pulse 2s infinite;
          }
        }
        
        .header-right {
          .current-value {
            font-size: 20px;
            font-weight: 700;
            color: #409eff;
            font-family: 'Roboto Mono', monospace;
          }
        }
      }
    }
  }
  
  .details-section {
    .details-grid {
      display: grid;
      grid-template-columns: repeat(auto-fit, minmax(300px, 1fr));
      gap: 16px;
    }
    
    .performance-card {
      .performance-metrics {
        display: flex;
        flex-direction: column;
        gap: 16px;
        
        .metric-item {
          .metric-label {
            font-size: 12px;
            color: #909399;
            margin-bottom: 4px;
          }
          
          .metric-value {
            display: flex;
            align-items: baseline;
            gap: 4px;
            margin-bottom: 8px;
            
            .value-number {
              font-size: 20px;
              font-weight: 600;
              color: #303133;
              font-family: 'Roboto Mono', monospace;
            }
            
            .value-unit {
              font-size: 12px;
              color: #606266;
            }
          }
          
          .metric-bar {
            margin-top: 4px;
          }
        }
      }
    }
    
    .events-card {
      .events-list {
        max-height: 300px;
        overflow-y: auto;
        
        .event-item {
          display: flex;
          align-items: center;
          gap: 12px;
          padding: 8px 0;
          border-bottom: 1px solid #f0f0f0;
          
          &:last-child {
            border-bottom: none;
          }
          
          .event-icon {
            width: 24px;
            height: 24px;
            display: flex;
            align-items: center;
            justify-content: center;
            border-radius: 50%;
            font-size: 12px;
            
            &.event-connect {
              background: rgba(103, 194, 58, 0.1);
              color: #67c23a;
            }
            
            &.event-disconnect {
              background: rgba(245, 108, 108, 0.1);
              color: #f56c6c;
            }
            
            &.event-session {
              background: rgba(64, 158, 255, 0.1);
              color: #409eff;
            }
            
            &.event-error {
              background: rgba(245, 108, 108, 0.1);
              color: #f56c6c;
            }
          }
          
          .event-content {
            flex: 1;
            
            .event-title {
              font-size: 13px;
              color: #303133;
              line-height: 1.2;
            }
            
            .event-time {
              font-size: 11px;
              color: #c0c4cc;
              margin-top: 2px;
            }
          }
          
          .event-status {
            font-size: 11px;
            padding: 2px 6px;
            border-radius: 4px;
            
            &.connect {
              background: rgba(103, 194, 58, 0.1);
              color: #67c23a;
            }
            
            &.disconnect {
              background: rgba(245, 108, 108, 0.1);
              color: #f56c6c;
            }
            
            &.session {
              background: rgba(64, 158, 255, 0.1);
              color: #409eff;
            }
            
            &.error {
              background: rgba(245, 108, 108, 0.1);
              color: #f56c6c;
            }
          }
        }
      }
    }
    
    .card-header {
      display: flex;
      justify-content: space-between;
      align-items: center;
      
      .section-icon {
        margin-right: 8px;
      }
      
      .section-title {
        font-weight: 600;
        font-size: 14px;
      }
    }
  }
}

@keyframes pulse {
  0%, 50% { opacity: 1; }
  51%, 100% { opacity: 0.7; }
}

// 深色主题适配
.dark {
  .dashboard-content {
    .trend-card,
    .performance-card,
    .events-card {
      background: rgba(40, 40, 40, 0.9);
      border-color: #414243;
    }
    
    .events-list .event-item {
      border-bottom-color: #414243;
    }
  }
}
</style> 