<template>
  <div class="transaction-monitor">
    <!-- 状态指示器 -->
    <div class="status-bar">
      <el-card class="status-card">
        <div class="status-item">
          <el-icon class="status-icon" :class="{ 'online': isWSConnected }">
            <Connection />
          </el-icon>
          <span>WebSocket: {{ wsStatus }}</span>
        </div>
        <div class="status-item">
          <el-icon class="status-icon mqtt-icon" :class="{ 'online': isMQTTConnected }">
            <MessageBox />
          </el-icon>
          <span>MQTT: {{ mqttStatus }}</span>
        </div>
        <div class="status-item">
          <span>实时交易: {{ realTimeStats.activeTransactions }}</span>
        </div>
        <div class="status-item">
          <span>在线客户端: {{ realTimeStats.onlineClients }}</span>
        </div>
      </el-card>
    </div>

    <!-- 统计图表区域 -->
    <el-row :gutter="20" class="charts-row">
      <!-- 交易状态分布 -->
      <el-col :span="12">
        <el-card class="chart-card">
          <template #header>
            <div class="card-header">
              <span>交易状态分布</span>
              <el-button size="small" @click="refreshCharts">刷新</el-button>
            </div>
          </template>
          <div class="chart-container">
            <v-chart 
              ref="statusChart"
              :option="statusChartOption" 
              style="height: 300px;"
              autoresize
            />
          </div>
        </el-card>
      </el-col>

      <!-- 实时交易趋势 -->
      <el-col :span="12">
        <el-card class="chart-card">
          <template #header>
            <div class="card-header">
              <span>实时交易趋势</span>
              <el-radio-group v-model="trendPeriod" size="small" @change="refreshTrendData">
                <el-radio-button label="5m">5分钟</el-radio-button>
                <el-radio-button label="1h">1小时</el-radio-button>
                <el-radio-button label="24h">24小时</el-radio-button>
              </el-radio-group>
            </div>
          </template>
          <div class="chart-container">
            <v-chart 
              ref="trendChart"
              :option="trendChartOption" 
              style="height: 300px;"
              autoresize
            />
          </div>
        </el-card>
      </el-col>
    </el-row>

    <!-- 实时交易列表 -->
    <el-card class="transaction-list-card">
      <template #header>
        <div class="card-header">
          <span>实时交易列表</span>
          <div class="header-controls">
            <el-switch
              v-model="autoRefresh"
              active-text="自动刷新"
              @change="toggleAutoRefresh"
            />
            <el-button 
              size="small" 
              type="primary" 
              @click="loadTransactions"
              :loading="loading"
            >
              手动刷新
            </el-button>
          </div>
        </div>
      </template>

      <!-- 筛选器 -->
      <div class="filters">
        <el-form :model="filters" inline>
          <el-form-item label="状态">
            <el-select v-model="filters.status" placeholder="选择状态" clearable @change="loadTransactions">
              <el-option label="全部" value="" />
              <el-option label="待处理" value="pending" />
              <el-option label="进行中" value="active" />
              <el-option label="已完成" value="completed" />
              <el-option label="失败" value="failed" />
              <el-option label="已取消" value="cancelled" />
              <el-option label="超时" value="timeout" />
            </el-select>
          </el-form-item>
          <el-form-item label="交易ID">
            <el-input 
              v-model="filters.transactionId" 
              placeholder="输入交易ID"
              clearable
              @keyup.enter="loadTransactions"
            />
          </el-form-item>
          <el-form-item label="客户端ID">
            <el-input 
              v-model="filters.clientId" 
              placeholder="输入客户端ID"
              clearable
              @keyup.enter="loadTransactions"
            />
          </el-form-item>
        </el-form>
      </div>

      <!-- 交易表格 -->
      <el-table 
        :data="transactions" 
        v-loading="loading"
        stripe
        border
        style="width: 100%"
        :row-class-name="getRowClassName"
      >
        <el-table-column prop="transaction_id" label="交易ID" width="200" fixed="left">
          <template #default="{ row }">
            <el-link type="primary" @click="viewTransaction(row)">
              {{ row.transaction_id }}
            </el-link>
          </template>
        </el-table-column>
        
        <el-table-column prop="status" label="状态" width="120">
          <template #default="{ row }">
            <el-tag :type="getStatusType(row.status)" size="small">
              {{ getStatusText(row.status) }}
            </el-tag>
          </template>
        </el-table-column>

        <el-table-column prop="transmitter_client_id" label="传卡端" width="180">
          <template #default="{ row }">
            <span class="client-id">{{ row.transmitter_client_id }}</span>
          </template>
        </el-table-column>

        <el-table-column prop="receiver_client_id" label="收卡端" width="180">
          <template #default="{ row }">
            <span class="client-id">{{ row.receiver_client_id || '未分配' }}</span>
          </template>
        </el-table-column>

        <el-table-column prop="amount" label="金额" width="100">
          <template #default="{ row }">
            <span v-if="row.amount">¥{{ (row.amount / 100).toFixed(2) }}</span>
            <span v-else>--</span>
          </template>
        </el-table-column>

        <el-table-column prop="created_at" label="创建时间" width="160">
          <template #default="{ row }">
            {{ formatTime(row.created_at) }}
          </template>
        </el-table-column>

        <el-table-column prop="updated_at" label="更新时间" width="160">
          <template #default="{ row }">
            {{ formatTime(row.updated_at) }}
          </template>
        </el-table-column>

        <el-table-column prop="duration" label="耗时" width="100">
          <template #default="{ row }">
            {{ getDuration(row) }}
          </template>
        </el-table-column>

        <el-table-column label="操作" width="150" fixed="right">
          <template #default="{ row }">
            <el-button size="small" @click="viewTransaction(row)">详情</el-button>
            <el-button 
              v-if="canCancel(row)" 
              size="small" 
              type="warning"
              @click="cancelTransaction(row)"
            >
              取消
            </el-button>
          </template>
        </el-table-column>
      </el-table>

      <!-- 分页 -->
      <div class="pagination">
        <el-pagination
          v-model:current-page="pagination.page"
          v-model:page-size="pagination.pageSize"
          :page-sizes="[10, 20, 50, 100]"
          :total="pagination.total"
          layout="total, sizes, prev, pager, next, jumper"
          @size-change="loadTransactions"
          @current-change="loadTransactions"
        />
      </div>
    </el-card>

    <!-- 交易详情对话框 -->
    <el-dialog 
      v-model="detailDialog.visible" 
      title="交易详情" 
      width="80%"
      :before-close="closeDetailDialog"
    >
      <transaction-detail 
        v-if="detailDialog.transaction"
        :transaction="detailDialog.transaction"
        @refresh="loadTransactions"
      />
    </el-dialog>
  </div>
</template>

<script setup>
import { ref, reactive, onMounted, onUnmounted, computed, nextTick } from 'vue'
import { ElMessage, ElMessageBox } from 'element-plus'
import { Connection, MessageBox } from '@element-plus/icons-vue'
import { use } from 'echarts/core'
import { CanvasRenderer } from 'echarts/renderers'
import { PieChart, LineChart } from 'echarts/charts'
import { TitleComponent, TooltipComponent, LegendComponent, GridComponent } from 'echarts/components'
import VChart from 'vue-echarts'
import { wsManager } from '@/utils/websocket'
import { 
  getTransactionList, 
  getStatistics, 
  updateTransactionStatus,
  getMQTTStatus
} from '@/api/nfcRelay'
import TransactionDetail from './TransactionDetail.vue'

// 注册ECharts组件
use([
  CanvasRenderer,
  PieChart,
  LineChart,
  TitleComponent,
  TooltipComponent,
  LegendComponent,
  GridComponent
])

// 响应式数据
const loading = ref(false)
const autoRefresh = ref(true)
const trendPeriod = ref('1h')
const isWSConnected = ref(false)
const isMQTTConnected = ref(false)

// 状态计算
const wsStatus = computed(() => isWSConnected.value ? '已连接' : '未连接')
const mqttStatus = computed(() => isMQTTConnected.value ? '已连接' : '未连接')

// 实时统计数据
const realTimeStats = reactive({
  activeTransactions: 0,
  onlineClients: 0,
  totalToday: 0,
  successRate: 0
})

// 筛选器
const filters = reactive({
  status: '',
  transactionId: '',
  clientId: ''
})

// 分页
const pagination = reactive({
  page: 1,
  pageSize: 20,
  total: 0
})

// 交易列表
const transactions = ref([])

// 图表选项
const statusChartOption = ref({})
const trendChartOption = ref({})

// 详情对话框
const detailDialog = reactive({
  visible: false,
  transaction: null
})

// 定时器
let refreshTimer = null
let statsTimer = null

// 组件挂载
onMounted(() => {
  initWebSocket()
  initCharts()
  loadTransactions()
  loadStatistics()
  checkMQTTStatus()
  
  // 开始定时任务
  if (autoRefresh.value) {
    startAutoRefresh()
  }
  startStatsRefresh()
})

// 组件卸载
onUnmounted(() => {
  wsManager.disconnect()
  stopAutoRefresh()
  stopStatsRefresh()
})

// 初始化WebSocket
function initWebSocket() {
  wsManager.connect()
  
  // 监听连接状态
  wsManager.on('open', () => {
    isWSConnected.value = true
    ElMessage.success('WebSocket连接成功')
  })
  
  wsManager.on('close', () => {
    isWSConnected.value = false
  })
  
  wsManager.on('error', () => {
    isWSConnected.value = false
  })
  
  // 监听交易状态更新
  wsManager.on('transaction_status', (data) => {
    handleTransactionUpdate(data)
  })
  
  // 监听APDU消息
  wsManager.on('apdu_message', (data) => {
    handleAPDUMessage(data)
  })
  
  // 监听客户端状态
  wsManager.on('client_status', (data) => {
    handleClientStatus(data)
  })
}

// 处理交易更新
function handleTransactionUpdate(data) {
  const { transaction_id, status, action } = data
  
  // 更新列表中的交易状态
  const index = transactions.value.findIndex(t => t.transaction_id === transaction_id)
  if (index > -1) {
    transactions.value[index] = { ...transactions.value[index], ...data }
  } else if (action === 'created') {
    // 如果是新创建的交易，重新加载列表
    loadTransactions()
  }
  
  // 更新统计
  updateRealTimeStats()
  
  // 显示通知
  ElMessage({
    type: getNotificationType(status),
    message: `交易 ${transaction_id} ${getStatusText(status)}`
  })
}

// 处理APDU消息
function handleAPDUMessage(data) {
  // 这里可以添加APDU消息的实时显示逻辑
  console.log('收到APDU消息:', data)
}

// 处理客户端状态
function handleClientStatus(data) {
  updateRealTimeStats()
}

// 初始化图表
function initCharts() {
  initStatusChart()
  initTrendChart()
}

// 初始化状态分布图
function initStatusChart() {
  statusChartOption.value = {
    tooltip: {
      trigger: 'item',
      formatter: '{a} <br/>{b}: {c} ({d}%)'
    },
    legend: {
      orient: 'vertical',
      left: 'left'
    },
    series: [
      {
        name: '交易状态',
        type: 'pie',
        radius: '50%',
        data: [],
        emphasis: {
          itemStyle: {
            shadowBlur: 10,
            shadowOffsetX: 0,
            shadowColor: 'rgba(0, 0, 0, 0.5)'
          }
        }
      }
    ]
  }
}

// 初始化趋势图
function initTrendChart() {
  trendChartOption.value = {
    tooltip: {
      trigger: 'axis'
    },
    legend: {
      data: ['成功', '失败', '进行中']
    },
    grid: {
      left: '3%',
      right: '4%',
      bottom: '3%',
      containLabel: true
    },
    xAxis: {
      type: 'category',
      data: []
    },
    yAxis: {
      type: 'value'
    },
    series: [
      {
        name: '成功',
        type: 'line',
        data: [],
        itemStyle: { color: '#67C23A' }
      },
      {
        name: '失败',
        type: 'line',
        data: [],
        itemStyle: { color: '#F56C6C' }
      },
      {
        name: '进行中',
        type: 'line',
        data: [],
        itemStyle: { color: '#E6A23C' }
      }
    ]
  }
}

// 加载交易列表
async function loadTransactions() {
  loading.value = true
  try {
    const params = {
      page: pagination.page,
      page_size: pagination.pageSize,
      ...filters
    }
    
    const response = await getTransactionList(params)
    transactions.value = response.data.list || []
    pagination.total = response.data.total || 0
    
    updateRealTimeStats()
  } catch (error) {
    console.error('加载交易列表失败:', error)
    ElMessage.error('加载交易列表失败')
  } finally {
    loading.value = false
  }
}

// 加载统计数据
async function loadStatistics() {
  try {
    // 根据周期计算日期范围
    const endDate = new Date()
    const startDate = new Date()
    
    switch (trendPeriod.value) {
      case '1h':
        startDate.setHours(endDate.getHours() - 1)
        break
      case '24h':
        startDate.setDate(endDate.getDate() - 1)
        break
      case '7d':
        startDate.setDate(endDate.getDate() - 7)
        break
      case '30d':
        startDate.setDate(endDate.getDate() - 30)
        break
      default:
        startDate.setDate(endDate.getDate() - 1) // 默认24小时
    }
    
    const response = await getStatistics({
      start_date: startDate.toISOString().split('T')[0], // 格式化为 YYYY-MM-DD
      end_date: endDate.toISOString().split('T')[0],
      group_by: trendPeriod.value === '1h' ? 'hour' : 
               trendPeriod.value === '24h' ? 'hour' :
               trendPeriod.value === '7d' ? 'day' : 'day'
    })
    
    const data = response.data
    updateStatusChart(data.statusDistribution)
    updateTrendChart(data.trendData)
  } catch (error) {
    console.error('加载统计数据失败:', error)
  }
}

// 检查MQTT状态
async function checkMQTTStatus() {
  try {
    const response = await getMQTTStatus()
    isMQTTConnected.value = response.data.mqtt_connected
  } catch (error) {
    isMQTTConnected.value = false
  }
}

// 更新状态图表
function updateStatusChart(data) {
  statusChartOption.value.series[0].data = Object.entries(data || {}).map(([key, value]) => ({
    name: getStatusText(key),
    value: value
  }))
}

// 更新趋势图表
function updateTrendChart(data) {
  if (!data) return
  
  trendChartOption.value.xAxis.data = data.timeLabels || []
  trendChartOption.value.series[0].data = data.successData || []
  trendChartOption.value.series[1].data = data.failureData || []
  trendChartOption.value.series[2].data = data.activeData || []
}

// 更新实时统计
function updateRealTimeStats() {
  const activeCount = transactions.value.filter(t => 
    ['pending', 'active'].includes(t.status)
  ).length
  
  realTimeStats.activeTransactions = activeCount
  realTimeStats.totalToday = transactions.value.length
  
  const completedCount = transactions.value.filter(t => t.status === 'completed').length
  realTimeStats.successRate = transactions.value.length > 0 
    ? Math.round((completedCount / transactions.value.length) * 100)
    : 0
}

// 刷新图表
function refreshCharts() {
  loadStatistics()
}

// 更新趋势图表周期
function refreshTrendData() {
  loadStatistics()
}

// 切换自动刷新
function toggleAutoRefresh() {
  if (autoRefresh.value) {
    startAutoRefresh()
  } else {
    stopAutoRefresh()
  }
}

// 开始自动刷新
function startAutoRefresh() {
  stopAutoRefresh()
  refreshTimer = setInterval(() => {
    loadTransactions()
  }, 30000) // 30秒刷新一次
}

// 停止自动刷新
function stopAutoRefresh() {
  if (refreshTimer) {
    clearInterval(refreshTimer)
    refreshTimer = null
  }
}

// 开始统计刷新
function startStatsRefresh() {
  statsTimer = setInterval(() => {
    checkMQTTStatus()
    loadStatistics()
  }, 60000) // 1分钟刷新一次统计
}

// 停止统计刷新
function stopStatsRefresh() {
  if (statsTimer) {
    clearInterval(statsTimer)
    statsTimer = null
  }
}

// 查看交易详情
function viewTransaction(transaction) {
  detailDialog.transaction = transaction
  detailDialog.visible = true
}

// 关闭详情对话框
function closeDetailDialog() {
  detailDialog.visible = false
  detailDialog.transaction = null
}

// 取消交易
async function cancelTransaction(transaction) {
  try {
    await ElMessageBox.confirm(
      `确定要取消交易 ${transaction.transaction_id} 吗？`,
      '确认取消',
      {
        confirmButtonText: '确定',
        cancelButtonText: '取消',
        type: 'warning'
      }
    )
    
    await updateTransactionStatus({
      transaction_id: transaction.transaction_id,
      status: 'cancelled',
      reason: '用户取消'
    })
    
    ElMessage.success('交易已取消')
    loadTransactions()
  } catch (error) {
    if (error !== 'cancel') {
      console.error('取消交易失败:', error)
      ElMessage.error('取消交易失败')
    }
  }
}

// 工具函数
function getStatusType(status) {
  const typeMap = {
    'pending': 'info',
    'active': 'warning', 
    'completed': 'success',
    'failed': 'danger',
    'cancelled': 'info',
    'timeout': 'danger'
  }
  return typeMap[status] || 'info'
}

function getStatusText(status) {
  const textMap = {
    'pending': '待处理',
    'active': '进行中',
    'completed': '已完成',
    'failed': '失败',
    'cancelled': '已取消',
    'timeout': '超时'
  }
  return textMap[status] || status
}

function getNotificationType(status) {
  const typeMap = {
    'completed': 'success',
    'failed': 'error',
    'cancelled': 'warning',
    'timeout': 'error'
  }
  return typeMap[status] || 'info'
}

function formatTime(timeStr) {
  if (!timeStr) return '--'
  return new Date(timeStr).toLocaleString('zh-CN')
}

function getDuration(transaction) {
  if (!transaction.created_at) return '--'
  
  const start = new Date(transaction.created_at)
  const end = transaction.completed_at ? new Date(transaction.completed_at) : new Date()
  const duration = Math.floor((end - start) / 1000)
  
  if (duration < 60) return `${duration}s`
  if (duration < 3600) return `${Math.floor(duration / 60)}m ${duration % 60}s`
  return `${Math.floor(duration / 3600)}h ${Math.floor((duration % 3600) / 60)}m`
}

function canCancel(transaction) {
  return ['pending', 'active'].includes(transaction.status)
}

function getRowClassName({ row }) {
  if (row.status === 'failed' || row.status === 'timeout') {
    return 'error-row'
  }
  if (row.status === 'active') {
    return 'active-row'
  }
  return ''
}
</script>

<style scoped>
.transaction-monitor {
  padding: 20px;
}

.status-bar {
  margin-bottom: 20px;
}

.status-card .el-card__body {
  display: flex;
  align-items: center;
  gap: 30px;
  padding: 15px 20px;
}

.status-item {
  display: flex;
  align-items: center;
  gap: 8px;
  font-size: 14px;
}

.status-icon {
  font-size: 16px;
  color: #909399;
  transition: color 0.3s;
}

.status-icon.online {
  color: #67C23A;
}

.mqtt-icon.online {
  color: #409EFF;
}

.charts-row {
  margin-bottom: 20px;
}

.chart-card .card-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
}

.chart-container {
  width: 100%;
  height: 300px;
}

.transaction-list-card .card-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
}

.header-controls {
  display: flex;
  align-items: center;
  gap: 15px;
}

.filters {
  margin-bottom: 20px;
  padding: 15px;
  background-color: #f8f9fa;
  border-radius: 4px;
}

.client-id {
  font-family: 'Monaco', 'Consolas', monospace;
  font-size: 12px;
  color: #606266;
}

.pagination {
  margin-top: 20px;
  display: flex;
  justify-content: center;
}

/* 表格行样式 */
:deep(.error-row) {
  background-color: #fef0f0;
}

:deep(.active-row) {
  background-color: #fdf6ec;
}

/* 响应式 */
@media (max-width: 768px) {
  .charts-row .el-col {
    margin-bottom: 20px;
  }
  
  .status-card .el-card__body {
    flex-direction: column;
    align-items: flex-start;
    gap: 10px;
  }
  
  .filters .el-form {
    display: block;
  }
  
  .filters .el-form-item {
    display: block;
    margin-bottom: 10px;
  }
}
</style> 