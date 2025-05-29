<!--
  NFC中继管理 - 增强版仪表盘
  集成高级筛选、告警中心、数据对比、导出等功能
-->
<template>
  <div class="enhanced-dashboard">
    <!-- 页面头部 -->
    <div class="dashboard-header">
      <div class="header-left">
        <h1 class="page-title">
          <el-icon class="title-icon"><Monitor /></el-icon>
          NFC中继监控大屏
        </h1>
        <div class="status-indicators">
          <el-tag 
            :type="systemStatus.type" 
            size="large"
            class="system-status"
          >
            {{ systemStatus.text }}
          </el-tag>
          <span class="last-update">
            最后更新: {{ formatTime(lastUpdateTime) }}
          </span>
        </div>
      </div>
      
      <div class="header-right">
        <!-- 实时/历史模式切换 -->
        <el-segmented 
          v-model="viewMode" 
          :options="viewModeOptions"
          @change="onViewModeChange"
        />
        
        <!-- 数据对比按钮 -->
        <el-button 
          type="primary" 
          @click="showComparison = true"
          :icon="TrendCharts"
        >
          数据对比
        </el-button>
        
        <!-- 导出按钮 -->
        <el-dropdown @command="handleExport">
          <el-button type="success" :icon="Download">
            导出数据
            <el-icon><ArrowDown /></el-icon>
          </el-button>
          <template #dropdown>
            <el-dropdown-menu>
              <el-dropdown-item command="excel">导出Excel</el-dropdown-item>
              <el-dropdown-item command="csv">导出CSV</el-dropdown-item>
              <el-dropdown-item command="pdf">导出PDF报告</el-dropdown-item>
              <el-dropdown-item command="image">导出图片</el-dropdown-item>
            </el-dropdown-menu>
          </template>
        </el-dropdown>
        
        <!-- 全屏按钮 -->
        <el-button 
          @click="toggleFullscreen"
          :icon="isFullscreen ? OfficeBuilding : FullScreen"
          circle
        />
        
        <!-- 设置按钮 -->
        <el-button 
          @click="showSettings = true"
          :icon="Setting"
          circle
        />
      </div>
    </div>

    <!-- 高级筛选器 -->
    <AdvancedFilters 
      v-model="filterParams"
      @filter-change="onFilterChange"
      class="dashboard-filters"
    />

    <!-- 主要内容区域 -->
    <div class="dashboard-content">
      <!-- 左侧面板 -->
      <div class="left-panel">
        <!-- 核心指标卡片 -->
        <div class="metrics-grid">
          <MetricCard
            v-for="metric in coreMetrics"
            :key="metric.key"
            :title="metric.title"
            :value="metric.value"
            :trend="metric.trend"
            :icon="metric.icon"
            :color="metric.color"
            :unit="metric.unit"
            :change="getChangeAnimation('dashboard', metric.key)"
            @click="showMetricDetails(metric)"
          />
        </div>
        
        <!-- 性能指标图表 -->
        <div class="performance-charts">
          <el-card class="chart-card">
            <template #header>
              <div class="chart-header">
                <span class="chart-title">性能趋势</span>
                <el-select 
                  v-model="selectedPerformanceMetric" 
                  size="small"
                  style="width: 120px"
                  @change="onPerformanceMetricChange"
                >
                  <el-option label="响应时间" value="response_time" />
                  <el-option label="吞吐量" value="throughput" />
                  <el-option label="错误率" value="error_rate" />
                  <el-option label="内存使用" value="memory_usage" />
                </el-select>
              </div>
            </template>
            <div class="chart-container">
              <LineChart 
                :data="performanceChartData"
                :options="performanceChartOptions"
                height="300px"
              />
            </div>
          </el-card>
        </div>
        
        <!-- 连接分布图 -->
        <div class="connection-distribution">
          <el-card class="chart-card">
            <template #header>
              <span class="chart-title">连接分布</span>
            </template>
            <el-row :gutter="16">
              <el-col :span="12">
                <PieChart 
                  :data="deviceTypeDistribution"
                  title="设备类型"
                  height="200px"
                />
              </el-col>
              <el-col :span="12">
                <PieChart 
                  :data="connectionStatusDistribution"
                  title="连接状态"
                  height="200px"
                />
              </el-col>
            </el-row>
          </el-card>
        </div>
      </div>

      <!-- 右侧面板 -->
      <div class="right-panel">
        <!-- 告警中心 -->
        <div class="alert-section">
          <AlertCenter 
            :alerts="dashboardData.alerts"
            @alert-acknowledged="onAlertAcknowledged"
            @alert-deleted="onAlertDeleted"
            @refresh-alerts="refreshAlerts"
          />
        </div>
        
        <!-- 地理分布热力图 -->
        <div class="geographic-section">
          <el-card class="chart-card">
            <template #header>
              <div class="chart-header">
                <span class="chart-title">地理分布</span>
                <el-button-group size="small">
                  <el-button 
                    :type="geoViewType === 'heatmap' ? 'primary' : 'default'"
                    @click="geoViewType = 'heatmap'"
                  >
                    热力图
                  </el-button>
                  <el-button 
                    :type="geoViewType === 'scatter' ? 'primary' : 'default'"
                    @click="geoViewType = 'scatter'"
                  >
                    散点图
                  </el-button>
                </el-button-group>
              </div>
            </template>
            <div class="chart-container">
              <WorldMap 
                :data="dashboardData.geographic"
                :view-type="geoViewType"
                height="300px"
              />
            </div>
          </el-card>
        </div>
        
        <!-- 实时活动流 -->
        <div class="activity-stream">
          <el-card class="chart-card">
            <template #header>
              <span class="chart-title">实时活动</span>
            </template>
            <ActivityStream 
              :activities="realtimeActivities"
              height="250px"
            />
          </el-card>
        </div>
      </div>
    </div>

    <!-- 数据对比对话框 -->
    <el-dialog
      v-model="showComparison"
      title="数据对比分析"
      width="80%"
      top="5vh"
    >
      <DataComparison 
        v-if="showComparison"
        :current-period="filterParams"
        @close="showComparison = false"
      />
    </el-dialog>

    <!-- 指标详情对话框 -->
    <el-dialog
      v-model="showMetricDetail"
      :title="selectedMetricDetail?.title + ' 详情'"
      width="70%"
    >
      <MetricDetail 
        v-if="selectedMetricDetail"
        :metric="selectedMetricDetail"
        :time-range="filterParams"
      />
    </el-dialog>

    <!-- 设置对话框 -->
    <el-dialog
      v-model="showSettings"
      title="仪表盘设置"
      width="500px"
    >
      <DashboardSettings 
        v-model="dashboardSettings"
        @save="saveDashboardSettings"
      />
    </el-dialog>
  </div>
</template>

<script setup>
import { ref, computed, onMounted, onUnmounted, watch, nextTick } from 'vue'
import { ElMessage, ElLoading } from 'element-plus'
import { 
  Monitor,
  TrendCharts,
  Download,
  ArrowDown,
  FullScreen,
  OfficeBuilding,
  Setting
} from '@element-plus/icons-vue'

// 组件导入
import AdvancedFilters from '../components/AdvancedFilters.vue'
import AlertCenter from '../components/AlertCenter.vue'
import MetricCard from '../components/MetricCard.vue'
import LineChart from '../components/charts/LineChart.vue'
import PieChart from '../components/charts/PieChart.vue'
import WorldMap from '../components/charts/WorldMap.vue'
import ActivityStream from '../components/ActivityStream.vue'
import DataComparison from '../components/DataComparison.vue'
import MetricDetail from '../components/MetricDetail.vue'
import DashboardSettings from '../components/DashboardSettings.vue'

// API导入
import { getDashboardStatsEnhanced, exportDashboardData } from '@/api/nfcRelayAdmin'
import { useRealtimeData } from '../utils/realtimeDataManager'
import { formatTime } from '@/utils/format'

// 实时数据管理器
const { 
  dashboardData, 
  isConnected, 
  connect, 
  getChangeAnimation 
} = useRealtimeData()

// 响应式数据
const loading = ref(false)
const viewMode = ref('realtime')
const lastUpdateTime = ref(new Date())
const showComparison = ref(false)
const showMetricDetail = ref(false)
const showSettings = ref(false)
const selectedMetricDetail = ref(null)
const selectedPerformanceMetric = ref('response_time')
const geoViewType = ref('heatmap')
const isFullscreen = ref(false)

// 筛选参数
const filterParams = ref({
  timeRange: '24h',
  granularity: 'hour',
  metrics: ['connections', 'sessions'],
  includeAlerts: true,
  realtime: true
})

// 仪表盘设置
const dashboardSettings = ref({
  autoRefresh: true,
  refreshInterval: 30,
  enableAnimations: true,
  showGrid: true,
  compactMode: false
})

// 实时活动
const realtimeActivities = ref([])

// 计算属性
const viewModeOptions = [
  { label: '实时监控', value: 'realtime' },
  { label: '历史分析', value: 'historical' }
]

const systemStatus = computed(() => {
  if (!isConnected.value) {
    return { type: 'danger', text: '连接断开' }
  }
  if (dashboardData.hub_status === 'online') {
    return { type: 'success', text: '系统正常' }
  }
  return { type: 'warning', text: '系统异常' }
})

const coreMetrics = computed(() => [
  {
    key: 'active_connections',
    title: '活跃连接',
    value: dashboardData.active_connections,
    unit: '个',
    icon: 'Connection',
    color: '#409eff',
    trend: getTrend('connections')
  },
  {
    key: 'active_sessions',
    title: '活跃会话',
    value: dashboardData.active_sessions,
    unit: '个',
    icon: 'Link',
    color: '#67c23a',
    trend: getTrend('sessions')
  },
  {
    key: 'apdu_relayed_last_minute',
    title: 'APDU转发/分钟',
    value: dashboardData.apdu_relayed_last_minute,
    unit: '次',
    icon: 'Transfer',
    color: '#e6a23c',
    trend: getTrend('apdu')
  },
  {
    key: 'avg_response_time',
    title: '平均响应时间',
    value: dashboardData.avg_response_time,
    unit: 'ms',
    icon: 'Timer',
    color: '#f56c6c',
    trend: getTrend('response_time')
  }
])

const performanceChartData = computed(() => {
  // 根据选择的性能指标返回对应的图表数据
  const data = dashboardData.connection_trend || []
  return {
    labels: data.map(item => formatTime(item.timestamp, 'HH:mm')),
    datasets: [{
      label: getPerformanceMetricLabel(selectedPerformanceMetric.value),
      data: data.map(item => item.value),
      borderColor: '#409eff',
      backgroundColor: 'rgba(64, 158, 255, 0.1)',
      tension: 0.4
    }]
  }
})

const performanceChartOptions = {
  responsive: true,
  maintainAspectRatio: false,
  plugins: {
    legend: {
      display: false
    }
  },
  scales: {
    y: {
      beginAtZero: true
    }
  }
}

const deviceTypeDistribution = computed(() => {
  const deviceTypes = dashboardData.statistics?.device_types || []
  return deviceTypes.map(item => ({
    name: item.device_type,
    value: item.count
  }))
})

const connectionStatusDistribution = computed(() => [
  { name: '在线', value: dashboardData.active_connections },
  { name: '离线', value: Math.max(0, dashboardData.total_connections - dashboardData.active_connections) }
])

// 方法
const getTrend = (type) => {
  // 计算趋势（上升/下降/平稳）
  const change = getChangeAnimation('dashboard', type)
  if (!change) return 'stable'
  return change.type === 'increase' ? 'up' : 'down'
}

const getPerformanceMetricLabel = (metric) => {
  const labels = {
    response_time: '响应时间',
    throughput: '吞吐量',
    error_rate: '错误率',
    memory_usage: '内存使用率'
  }
  return labels[metric] || metric
}

const onViewModeChange = (mode) => {
  filterParams.value.realtime = mode === 'realtime'
  refreshDashboard()
}

const onFilterChange = (filters) => {
  refreshDashboard()
}

const onPerformanceMetricChange = () => {
  // 刷新性能图表数据
  refreshPerformanceChart()
}

const showMetricDetails = (metric) => {
  selectedMetricDetail.value = metric
  showMetricDetail.value = true
}

const handleExport = async (format) => {
  try {
    const loadingInstance = ElLoading.service({
      lock: true,
      text: '正在生成导出文件...'
    })

    const exportRequest = {
      type: format,
      startTime: filterParams.value.startTime,
      endTime: filterParams.value.endTime,
      sections: ['dashboard', 'performance', 'geographic'],
      format: 'detailed'
    }

    const response = await exportDashboardData(exportRequest)
    
    // 下载文件
    const link = document.createElement('a')
    link.href = response.download_url
    link.download = response.file_name
    document.body.appendChild(link)
    link.click()
    document.body.removeChild(link)

    ElMessage.success('导出成功')
    loadingInstance.close()
  } catch (error) {
    ElMessage.error('导出失败: ' + error.message)
  }
}

const toggleFullscreen = () => {
  if (!isFullscreen.value) {
    document.documentElement.requestFullscreen()
    isFullscreen.value = true
  } else {
    document.exitFullscreen()
    isFullscreen.value = false
  }
}

const refreshDashboard = async () => {
  try {
    loading.value = true
    
    if (filterParams.value.realtime) {
      // 实时模式：使用WebSocket数据
      await connect()
    } else {
      // 历史模式：调用API获取数据
      const response = await getDashboardStatsEnhanced(filterParams.value)
      Object.assign(dashboardData, response.data)
    }
    
    lastUpdateTime.value = new Date()
  } catch (error) {
    ElMessage.error('刷新数据失败: ' + error.message)
  } finally {
    loading.value = false
  }
}

const refreshPerformanceChart = async () => {
  // 根据选择的性能指标刷新图表
  // TODO: 实现性能指标数据获取
}

const refreshAlerts = () => {
  // 刷新告警数据
  refreshDashboard()
}

const onAlertAcknowledged = (alert) => {
  // 处理告警确认
  console.log('Alert acknowledged:', alert)
}

const onAlertDeleted = (alert) => {
  // 处理告警删除
  const index = dashboardData.alerts.findIndex(a => a.id === alert.id)
  if (index > -1) {
    dashboardData.alerts.splice(index, 1)
  }
}

const saveDashboardSettings = () => {
  localStorage.setItem('dashboardSettings', JSON.stringify(dashboardSettings.value))
  showSettings.value = false
  ElMessage.success('设置已保存')
}

const loadDashboardSettings = () => {
  const saved = localStorage.getItem('dashboardSettings')
  if (saved) {
    try {
      Object.assign(dashboardSettings.value, JSON.parse(saved))
    } catch (error) {
      console.error('加载仪表盘设置失败:', error)
    }
  }
}

// 模拟实时活动数据
const generateRealtimeActivity = () => {
  const activities = [
    '客户端 Android-Device-123 连接成功',
    'APDU 命令转发: SELECT AID',
    '会话 session-456 建立成功',
    '检测到异常响应时间',
    '新的Provider设备上线'
  ]
  
  setInterval(() => {
    const activity = {
      id: Date.now(),
      message: activities[Math.floor(Math.random() * activities.length)],
      timestamp: new Date(),
      type: Math.random() > 0.8 ? 'warning' : 'info'
    }
    
    realtimeActivities.value.unshift(activity)
    if (realtimeActivities.value.length > 50) {
      realtimeActivities.value.pop()
    }
  }, 5000)
}

// 生命周期
onMounted(async () => {
  loadDashboardSettings()
  await refreshDashboard()
  generateRealtimeActivity()
  
  // 监听全屏状态变化
  document.addEventListener('fullscreenchange', () => {
    isFullscreen.value = !!document.fullscreenElement
  })
})

onUnmounted(() => {
  document.removeEventListener('fullscreenchange', () => {})
})

// 监听筛选参数变化
watch(filterParams, () => {
  if (!filterParams.value.realtime) {
    refreshDashboard()
  }
}, { deep: true })
</script>

<style scoped lang="scss">
.enhanced-dashboard {
  min-height: 100vh;
  background: linear-gradient(135deg, #667eea 0%, #764ba2 100%);
  padding: 16px;
  
  .dashboard-header {
    display: flex;
    justify-content: space-between;
    align-items: center;
    margin-bottom: 16px;
    padding: 16px 20px;
    background: rgba(255, 255, 255, 0.95);
    backdrop-filter: blur(10px);
    border-radius: 12px;
    box-shadow: 0 4px 20px rgba(0, 0, 0, 0.1);
    
    .header-left {
      .page-title {
        display: flex;
        align-items: center;
        gap: 12px;
        margin: 0 0 8px 0;
        font-size: 24px;
        font-weight: 700;
        color: #2c3e50;
        
        .title-icon {
          font-size: 28px;
          color: #409eff;
        }
      }
      
      .status-indicators {
        display: flex;
        align-items: center;
        gap: 16px;
        
        .system-status {
          font-weight: 600;
        }
        
        .last-update {
          font-size: 13px;
          color: #666;
        }
      }
    }
    
    .header-right {
      display: flex;
      align-items: center;
      gap: 12px;
    }
  }
  
  .dashboard-filters {
    margin-bottom: 16px;
  }
  
  .dashboard-content {
    display: grid;
    grid-template-columns: 2fr 1fr;
    gap: 16px;
    
    .left-panel {
      display: flex;
      flex-direction: column;
      gap: 16px;
      
      .metrics-grid {
        display: grid;
        grid-template-columns: repeat(2, 1fr);
        gap: 16px;
      }
      
      .performance-charts,
      .connection-distribution {
        .chart-card {
          height: 100%;
          
          .chart-header {
            display: flex;
            justify-content: space-between;
            align-items: center;
            
            .chart-title {
              font-weight: 600;
              color: #303133;
            }
          }
          
          .chart-container {
            padding: 16px 0;
          }
        }
      }
    }
    
    .right-panel {
      display: flex;
      flex-direction: column;
      gap: 16px;
      
      .alert-section {
        flex: 0 0 auto;
      }
      
      .geographic-section,
      .activity-stream {
        .chart-card {
          height: 100%;
        }
      }
    }
  }
}

// 响应式设计
@media (max-width: 1200px) {
  .enhanced-dashboard {
    .dashboard-content {
      grid-template-columns: 1fr;
      
      .left-panel {
        .metrics-grid {
          grid-template-columns: repeat(2, 1fr);
        }
      }
    }
  }
}

@media (max-width: 768px) {
  .enhanced-dashboard {
    padding: 8px;
    
    .dashboard-header {
      flex-direction: column;
      gap: 16px;
      text-align: center;
    }
    
    .dashboard-content {
      .left-panel {
        .metrics-grid {
          grid-template-columns: 1fr;
        }
      }
    }
  }
}

// 暗色主题
.dark .enhanced-dashboard {
  background: linear-gradient(135deg, #2c3e50 0%, #34495e 100%);
  
  .dashboard-header {
    background: rgba(31, 31, 31, 0.95);
    
    .page-title {
      color: #e5eaf3;
    }
    
    .last-update {
      color: #a3a6ad;
    }
  }
}

// 全屏模式样式
.enhanced-dashboard:-webkit-full-screen {
  .dashboard-header {
    position: fixed;
    top: 0;
    left: 0;
    right: 0;
    z-index: 1000;
    margin-bottom: 0;
  }
  
  .dashboard-content {
    padding-top: 100px;
  }
}
</style> 