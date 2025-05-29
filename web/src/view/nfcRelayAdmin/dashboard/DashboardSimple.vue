<!--
  NFC中继管理 - 概览仪表盘
  简洁高效的实时监控界面
-->
<template>
  <div class="nfc-dashboard">
    <!-- 页面头部 -->
    <div class="dashboard-header">
      <div class="header-left">
        <h1 class="page-title">概览仪表盘</h1>
        <div class="update-info">
          <el-icon class="update-icon" :class="{ rotating: loading }">
            <Refresh />
          </el-icon>
          <span class="update-text">
            {{ loading ? '更新中...' : `最后更新: ${formatters.lastUpdateText}` }}
          </span>
        </div>
      </div>
      
      <div class="header-right">
        <el-button 
          :icon="Refresh" 
          @click="refreshData" 
          :loading="loading"
          type="primary"
        >
          刷新数据
        </el-button>
      </div>
    </div>

    <!-- 连接状态警告 -->
    <el-alert
      v-if="!isOnline"
      title="系统离线"
      description="NFC中继服务当前处于离线状态，请检查服务器连接"
      type="error"
      :closable="false"
      show-icon
      class="status-alert"
    />

    <!-- 核心指标卡片 -->
    <div class="metrics-section">
      <div class="metrics-grid">
        <stat-card
          v-for="card in statCards"
          :key="card.title"
          :title="card.title"
          :value="card.value"
          :subtitle="card.subtitle"
          :icon="card.icon"
          :icon-color="card.iconColor"
          :trend="card.trend"
          class="metric-card"
          @click="handleCardClick(card)"
        />
      </div>
    </div>

    <!-- 图表区域 -->
    <div class="charts-section">
      <el-row :gutter="16">
        <!-- 连接数趋势 -->
        <el-col :span="12">
          <el-card shadow="hover" class="chart-card">
            <template #header>
              <div class="chart-header">
                <el-icon class="header-icon" color="#67C23A">
                  <TrendCharts />
                </el-icon>
                <span class="chart-title">连接数趋势</span>
                <el-tag size="small" type="success">实时</el-tag>
              </div>
            </template>
            
            <trend-chart
              :data="trendData.connection"
              height="280px"
              color="#67C23A"
              :smooth="true"
            />
            
            <div class="chart-footer">
              <div class="stat-item">
                <span class="stat-label">当前连接:</span>
                <span class="stat-value">{{ stats.activeConnections }}</span>
              </div>
              <div class="stat-item">
                <span class="stat-label">Provider:</span>
                <span class="stat-value">{{ details.providerCount }}</span>
              </div>
              <div class="stat-item">
                <span class="stat-label">Receiver:</span>
                <span class="stat-value">{{ details.receiverCount }}</span>
              </div>
            </div>
          </el-card>
        </el-col>

        <!-- 会话数趋势 -->
        <el-col :span="12">
          <el-card shadow="hover" class="chart-card">
            <template #header>
              <div class="chart-header">
                <el-icon class="header-icon" color="#E6A23C">
                  <TrendCharts />
                </el-icon>
                <span class="chart-title">会话数趋势</span>
                <el-tag size="small" type="warning">实时</el-tag>
              </div>
            </template>
            
            <trend-chart
              :data="trendData.session"
              height="280px"
              color="#E6A23C"
              :smooth="true"
            />
            
            <div class="chart-footer">
              <div class="stat-item">
                <span class="stat-label">当前会话:</span>
                <span class="stat-value">{{ stats.activeSessions }}</span>
              </div>
              <div class="stat-item">
                <span class="stat-label">已配对:</span>
                <span class="stat-value">{{ details.pairedSessions }}</span>
              </div>
              <div class="stat-item">
                <span class="stat-label">等待中:</span>
                <span class="stat-value">{{ details.waitingSessions }}</span>
              </div>
            </div>
          </el-card>
        </el-col>
      </el-row>
    </div>

    <!-- 详细信息区域 -->
    <div class="details-section">
      <el-row :gutter="16">
        <!-- 系统性能 -->
        <el-col :span="12">
          <el-card shadow="hover" class="detail-card">
            <template #header>
              <div class="card-header">
                <el-icon class="header-icon" color="#409EFF">
                  <Monitor />
                </el-icon>
                <span class="card-title">系统性能</span>
                <el-tag 
                  :type="performanceLevel.type" 
                  size="small"
                >
                  {{ performanceLevel.text }}
                </el-tag>
              </div>
            </template>
            
            <div class="performance-metrics">
              <!-- 响应时间 -->
              <div class="metric-row">
                <div class="metric-info">
                  <span class="metric-label">平均响应时间</span>
                  <span class="metric-value">{{ formatters.responseTimeText }}</span>
                </div>
                <el-progress 
                  :percentage="Math.min(stats.avgResponseTime / 200 * 100, 100)"
                  :color="getResponseTimeColor(stats.avgResponseTime)"
                  :stroke-width="8"
                  class="metric-progress"
                />
              </div>
              
              <!-- 系统负载 -->
              <div class="metric-row">
                <div class="metric-info">
                  <span class="metric-label">系统负载</span>
                  <span class="metric-value">{{ formatters.systemLoadText }}</span>
                </div>
                <el-progress 
                  :percentage="stats.systemLoad"
                  :color="getLoadColor(stats.systemLoad)"
                  :stroke-width="8"
                  class="metric-progress"
                />
              </div>
              
              <!-- 内存使用 -->
              <div class="metric-row">
                <div class="metric-info">
                  <span class="metric-label">内存使用</span>
                  <span class="metric-value">{{ formatters.memoryUsageText }}</span>
                </div>
                <el-progress 
                  :percentage="stats.memoryUsage"
                  :color="getLoadColor(stats.memoryUsage)"
                  :stroke-width="8"
                  class="metric-progress"
                />
              </div>
            </div>
            
            <div class="health-score">
              <div class="score-label">系统健康分数</div>
              <div class="score-value" :style="{ color: performanceLevel.color }">
                {{ systemHealthScore }}
              </div>
            </div>
          </el-card>
        </el-col>

        <!-- 实时事件 -->
        <el-col :span="12">
          <el-card shadow="hover" class="detail-card">
            <template #header>
              <div class="card-header">
                <el-icon class="header-icon" color="#F56C6C">
                  <Document />
                </el-icon>
                <span class="card-title">实时事件</span>
                <el-button 
                  type="primary" 
                  link 
                  size="small"
                  @click="$router.push('/nfc-relay-admin/audit-logs')"
                >
                  查看全部
                </el-button>
              </div>
            </template>
            
            <div class="events-container">
              <div 
                v-if="recentEvents.length > 0" 
                class="events-list"
              >
                <div 
                  v-for="event in recentEvents.slice(0, 8)" 
                  :key="event.id"
                  class="event-item"
                  :class="`event-${event.type}`"
                >
                  <div class="event-icon">
                    <el-icon>
                      <component :is="getEventIcon(event.type)" />
                    </el-icon>
                  </div>
                  <div class="event-content">
                    <div class="event-title">{{ event.title }}</div>
                    <div class="event-time">{{ formatRelativeTime(event.time) }}</div>
                  </div>
                  <div class="event-status" :class="event.type">
                    <el-tag 
                      :type="getEventTagType(event.type)"
                      size="small"
                    >
                      {{ getEventTypeText(event.type) }}
                    </el-tag>
                  </div>
                </div>
              </div>
              
              <el-empty 
                v-else 
                description="暂无实时事件" 
                :image-size="100"
              />
            </div>
          </el-card>
        </el-col>
      </el-row>
    </div>
  </div>
</template>

<script setup>
import { useRouter } from 'vue-router'
import { 
  Refresh, 
  TrendCharts, 
  Monitor, 
  Document,
  Connection,
  ChatDotRound,
  DataLine,
  CircleCheck,
  CircleClose,
  User,
  WarningFilled,
  InfoFilled
} from '@element-plus/icons-vue'
import { ElMessage } from 'element-plus'

// 组件导入
import { StatCard, TrendChart } from '../components'
import { useDashboard } from '../hooks/useDashboard'
import { formatRelativeTime } from '../utils/formatters'

defineOptions({
  name: 'NFCRelayDashboard'
})

const router = useRouter()

// 使用仪表盘hook
const {
  loading,
  error,
  stats,
  details,
  recentEvents,
  isOnline,
  systemHealthScore,
  performanceLevel,
  statCards,
  trendData,
  formatters,
  refreshData
} = useDashboard()

// 事件处理
const handleCardClick = (card) => {
  switch (card.title) {
    case '运行状态':
      router.push('/nfc-relay-admin/configuration')
      break
    case '活动连接':
      router.push('/nfc-relay-admin/clients')
      break
    case '活动会话':
      router.push('/nfc-relay-admin/sessions')
      break
    case 'APDU转发':
      router.push('/nfc-relay-admin/audit-logs')
      break
  }
}

// 工具函数
const getResponseTimeColor = (time) => {
  if (time < 50) return '#67C23A'
  if (time < 100) return '#E6A23C'
  return '#F56C6C'
}

const getLoadColor = (percentage) => {
  if (percentage < 30) return '#67C23A'
  if (percentage < 70) return '#E6A23C'
  return '#F56C6C'
}

const getEventIcon = (type) => {
  const iconMap = {
    connect: User,
    disconnect: CircleClose,
    session: ChatDotRound,
    error: WarningFilled,
    info: InfoFilled
  }
  return iconMap[type] || InfoFilled
}

const getEventTagType = (type) => {
  const typeMap = {
    connect: 'success',
    disconnect: 'warning',
    session: 'primary',
    error: 'danger',
    info: 'info'
  }
  return typeMap[type] || 'info'
}

const getEventTypeText = (type) => {
  const textMap = {
    connect: '连接',
    disconnect: '断开',
    session: '会话',
    error: '错误',
    info: '信息'
  }
  return textMap[type] || '未知'
}
</script>

<style scoped lang="scss">
.nfc-dashboard {
  padding: 20px;
  background: #f5f7fa;
  min-height: 100vh;
  
  .dashboard-header {
    display: flex;
    justify-content: space-between;
    align-items: center;
    margin-bottom: 24px;
    
    .header-left {
      .page-title {
        margin: 0 0 8px 0;
        font-size: 24px;
        font-weight: 600;
        color: #303133;
      }
      
      .update-info {
        display: flex;
        align-items: center;
        gap: 8px;
        font-size: 14px;
        color: #606266;
        
        .update-icon {
          &.rotating {
            animation: rotate 1s linear infinite;
          }
        }
      }
    }
  }
  
  .status-alert {
    margin-bottom: 24px;
  }
  
  .metrics-section {
    margin-bottom: 24px;
    
    .metrics-grid {
      display: grid;
      grid-template-columns: repeat(auto-fit, minmax(280px, 1fr));
      gap: 16px;
    }
  }
  
  .charts-section {
    margin-bottom: 24px;
    
    .chart-card {
      .chart-header {
        display: flex;
        align-items: center;
        gap: 8px;
        
        .header-icon {
          font-size: 18px;
        }
        
        .chart-title {
          flex: 1;
          font-weight: 500;
        }
      }
      
      .chart-footer {
        display: flex;
        justify-content: space-around;
        padding-top: 16px;
        border-top: 1px solid #f0f0f0;
        margin-top: 16px;
        
        .stat-item {
          text-align: center;
          
          .stat-label {
            display: block;
            font-size: 12px;
            color: #909399;
            margin-bottom: 4px;
          }
          
          .stat-value {
            font-size: 18px;
            font-weight: 600;
            color: #303133;
          }
        }
      }
    }
  }
  
  .details-section {
    .detail-card {
      .card-header {
        display: flex;
        align-items: center;
        gap: 8px;
        
        .header-icon {
          font-size: 18px;
        }
        
        .card-title {
          flex: 1;
          font-weight: 500;
        }
      }
      
      .performance-metrics {
        .metric-row {
          margin-bottom: 20px;
          
          &:last-child {
            margin-bottom: 0;
          }
          
          .metric-info {
            display: flex;
            justify-content: space-between;
            align-items: center;
            margin-bottom: 8px;
            
            .metric-label {
              font-size: 14px;
              color: #606266;
            }
            
            .metric-value {
              font-size: 16px;
              font-weight: 600;
              color: #303133;
            }
          }
        }
        
        .health-score {
          text-align: center;
          padding-top: 20px;
          border-top: 1px solid #f0f0f0;
          margin-top: 20px;
          
          .score-label {
            font-size: 14px;
            color: #606266;
            margin-bottom: 8px;
          }
          
          .score-value {
            font-size: 32px;
            font-weight: bold;
          }
        }
      }
      
      .events-container {
        max-height: 400px;
        overflow-y: auto;
        
        .events-list {
          .event-item {
            display: flex;
            align-items: center;
            gap: 12px;
            padding: 12px 0;
            border-bottom: 1px solid #f0f0f0;
            
            &:last-child {
              border-bottom: none;
            }
            
            .event-icon {
              flex-shrink: 0;
              width: 32px;
              height: 32px;
              border-radius: 50%;
              display: flex;
              align-items: center;
              justify-content: center;
              font-size: 16px;
              
              &.event-connect {
                background: #f0f9ff;
                color: #409eff;
              }
              
              &.event-disconnect {
                background: #fef0f0;
                color: #f56c6c;
              }
              
              &.event-session {
                background: #fdf6ec;
                color: #e6a23c;
              }
              
              &.event-error {
                background: #fef0f0;
                color: #f56c6c;
              }
            }
            
            .event-content {
              flex: 1;
              
              .event-title {
                font-size: 14px;
                font-weight: 500;
                color: #303133;
                margin-bottom: 4px;
              }
              
              .event-time {
                font-size: 12px;
                color: #909399;
              }
            }
            
            .event-status {
              flex-shrink: 0;
            }
          }
        }
      }
    }
  }
}

@keyframes rotate {
  from {
    transform: rotate(0deg);
  }
  to {
    transform: rotate(360deg);
  }
}

// 响应式设计
@media (max-width: 768px) {
  .nfc-dashboard {
    padding: 16px;
    
    .dashboard-header {
      flex-direction: column;
      align-items: flex-start;
      gap: 16px;
    }
    
    .metrics-grid {
      grid-template-columns: 1fr;
    }
  }
}
</style> 