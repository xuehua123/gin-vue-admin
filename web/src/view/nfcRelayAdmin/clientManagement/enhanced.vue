<!--
  NFC中继管理 - 增强版连接管理
  集成连接详情、历史记录、地理分布、批量操作等功能
-->
<template>
  <div class="enhanced-client-management">
    <!-- 页面头部 -->
    <div class="page-header">
      <div class="header-left">
        <h1 class="page-title">
          <el-icon><Connection /></el-icon>
          连接管理
        </h1>
        <div class="statistics">
          <el-statistic 
            title="总连接数" 
            :value="statistics.total"
            class="stat-item"
          />
          <el-statistic 
            title="在线连接" 
            :value="statistics.online"
            class="stat-item stat-online"
          />
          <el-statistic 
            title="离线连接" 
            :value="statistics.offline"
            class="stat-item stat-offline"
          />
        </div>
      </div>
      
      <div class="header-right">
        <el-button 
          type="primary" 
          @click="showBulkOperations = true"
          :disabled="selectedClients.length === 0"
        >
          批量操作 ({{ selectedClients.length }})
        </el-button>
        <el-button @click="refreshData" :loading="loading">
          <el-icon><Refresh /></el-icon>
          刷新
        </el-button>
        <el-button @click="showSettings = true">
          <el-icon><Setting /></el-icon>
          设置
        </el-button>
      </div>
    </div>

    <!-- 筛选器和搜索 -->
    <div class="filters-section">
      <el-card>
        <el-form :model="filters" :inline="true" class="filter-form">
          <el-form-item label="搜索">
            <el-input
              v-model="filters.search"
              placeholder="搜索客户端ID、设备名称或IP地址"
              style="width: 250px"
              clearable
              @keyup.enter="applyFilters"
            >
              <template #prefix>
                <el-icon><Search /></el-icon>
              </template>
            </el-input>
          </el-form-item>
          
          <el-form-item label="状态">
            <el-select v-model="filters.status" clearable style="width: 120px">
              <el-option label="全部" value="" />
              <el-option label="在线" value="online" />
              <el-option label="离线" value="offline" />
              <el-option label="重连中" value="reconnecting" />
            </el-select>
          </el-form-item>
          
          <el-form-item label="设备类型">
            <el-select v-model="filters.deviceType" clearable style="width: 150px">
              <el-option label="全部" value="" />
              <el-option label="Android" value="android" />
              <el-option label="iOS" value="ios" />
              <el-option label="Provider" value="provider" />
              <el-option label="其他" value="other" />
            </el-select>
          </el-form-item>
          
          <el-form-item label="连接时间">
            <el-date-picker
              v-model="filters.connectionTime"
              type="datetimerange"
              range-separator="至"
              start-placeholder="开始时间"
              end-placeholder="结束时间"
              style="width: 300px"
            />
          </el-form-item>
          
          <el-form-item>
            <el-button type="primary" @click="applyFilters">
              <el-icon><Search /></el-icon>
              查询
            </el-button>
            <el-button @click="resetFilters">
              <el-icon><RefreshLeft /></el-icon>
              重置
            </el-button>
          </el-form-item>
        </el-form>
      </el-card>
    </div>

    <!-- 主内容区域 -->
    <div class="content-section">
      <el-row :gutter="16">
        <!-- 左侧：客户端列表 -->
        <el-col :span="16">
          <el-card class="clients-card">
            <template #header>
              <div class="card-header">
                <span class="card-title">客户端列表</span>
                <div class="card-actions">
                  <el-button-group size="small">
                    <el-button 
                      :type="viewMode === 'table' ? 'primary' : 'default'"
                      @click="viewMode = 'table'"
                    >
                      <el-icon><List /></el-icon>
                      表格
                    </el-button>
                    <el-button 
                      :type="viewMode === 'card' ? 'primary' : 'default'"
                      @click="viewMode = 'card'"
                    >
                      <el-icon><Grid /></el-icon>
                      卡片
                    </el-button>
                  </el-button-group>
                </div>
              </div>
            </template>
            
            <!-- 表格视图 -->
            <div v-if="viewMode === 'table'" class="table-view">
              <el-table
                ref="clientTable"
                v-loading="loading"
                :data="paginatedClients"
                @selection-change="onSelectionChange"
                @row-click="onRowClick"
                class="clients-table"
              >
                <!-- 选择列 -->
                <el-table-column type="selection" width="50" />
                
                <!-- 状态指示器 -->
                <el-table-column prop="status" label="状态" width="80">
                  <template #default="{ row }">
                    <el-tooltip :content="getStatusText(row.status)">
                      <div class="status-indicator">
                        <el-icon 
                          :class="getStatusClass(row.status)"
                          :size="12"
                        >
                          <component :is="getStatusIcon(row.status)" />
                        </el-icon>
                      </div>
                    </el-tooltip>
                  </template>
                </el-table-column>
                
                <!-- 客户端信息 -->
                <el-table-column prop="clientId" label="客户端ID" min-width="120">
                  <template #default="{ row }">
                    <div class="client-info">
                      <div class="client-id">{{ row.clientId }}</div>
                      <div class="client-type">{{ row.deviceType }}</div>
                    </div>
                  </template>
                </el-table-column>
                
                <!-- 设备信息 -->
                <el-table-column prop="deviceInfo" label="设备信息" min-width="180">
                  <template #default="{ row }">
                    <div class="device-info">
                      <div class="device-name">{{ row.deviceName || '未知设备' }}</div>
                      <div class="device-version">{{ row.appVersion }}</div>
                    </div>
                  </template>
                </el-table-column>
                
                <!-- 网络信息 -->
                <el-table-column prop="networkInfo" label="网络信息" min-width="150">
                  <template #default="{ row }">
                    <div class="network-info">
                      <div class="ip-address">{{ row.ipAddress }}</div>
                      <div class="location">{{ row.location }}</div>
                    </div>
                  </template>
                </el-table-column>
                
                <!-- 连接时间 -->
                <el-table-column prop="connectedAt" label="连接时间" width="160">
                  <template #default="{ row }">
                    <div class="time-info">
                      <div>{{ formatTime(row.connectedAt) }}</div>
                      <div class="duration">{{ getConnectionDuration(row.connectedAt) }}</div>
                    </div>
                  </template>
                </el-table-column>
                
                <!-- 性能指标 -->
                <el-table-column prop="performance" label="性能" width="120">
                  <template #default="{ row }">
                    <div class="performance-info">
                      <el-tag 
                        :type="getPerformanceType(row.avgResponseTime)"
                        size="small"
                      >
                        {{ row.avgResponseTime }}ms
                      </el-tag>
                      <div class="throughput">{{ row.throughput }}/s</div>
                    </div>
                  </template>
                </el-table-column>
                
                <!-- 操作列 -->
                <el-table-column label="操作" width="120" fixed="right">
                  <template #default="{ row }">
                    <el-dropdown @command="(cmd) => handleAction(cmd, row)">
                      <el-button link type="primary" size="small">
                        更多
                        <el-icon><ArrowDown /></el-icon>
                      </el-button>
                      <template #dropdown>
                        <el-dropdown-menu>
                          <el-dropdown-item command="details">
                            <el-icon><InfoFilled /></el-icon>
                            详情
                          </el-dropdown-item>
                          <el-dropdown-item command="history">
                            <el-icon><Clock /></el-icon>
                            历史记录
                          </el-dropdown-item>
                          <el-dropdown-item 
                            v-if="row.status === 'online'"
                            command="disconnect"
                            divided
                          >
                            <el-icon><Close /></el-icon>
                            断开连接
                          </el-dropdown-item>
                          <el-dropdown-item command="blacklist">
                            <el-icon><Lock /></el-icon>
                            加入黑名单
                          </el-dropdown-item>
                        </el-dropdown-menu>
                      </template>
                    </el-dropdown>
                  </template>
                </el-table-column>
              </el-table>
            </div>
            
            <!-- 卡片视图 -->
            <div v-else class="card-view">
              <el-row :gutter="16">
                <el-col 
                  v-for="client in paginatedClients" 
                  :key="client.clientId"
                  :span="8"
                  class="client-card-col"
                >
                  <ClientCard 
                    :client="client"
                    :selected="selectedClients.includes(client.clientId)"
                    @click="onClientCardClick(client)"
                    @action="handleAction"
                  />
                </el-col>
              </el-row>
            </div>
            
            <!-- 分页 -->
            <div class="pagination-wrapper">
              <el-pagination
                v-model:current-page="pagination.current"
                v-model:page-size="pagination.size"
                :total="filteredClients.length"
                :page-sizes="[10, 20, 50, 100]"
                layout="total, sizes, prev, pager, next, jumper"
                @size-change="onPageSizeChange"
                @current-change="onPageChange"
              />
            </div>
          </el-card>
        </el-col>
        
        <!-- 右侧：统计和地理分布 -->
        <el-col :span="8">
          <!-- 实时统计 -->
          <el-card class="stats-card">
            <template #header>
              <span class="card-title">实时统计</span>
            </template>
            <div class="stats-content">
              <RealtimeStats :data="realtimeStats" />
            </div>
          </el-card>
          
          <!-- 地理分布 -->
          <el-card class="geographic-card">
            <template #header>
              <span class="card-title">地理分布</span>
            </template>
            <div class="geographic-content">
              <ConnectionMap :connections="clients" />
            </div>
          </el-card>
          
          <!-- 设备类型分布 -->
          <el-card class="device-type-card">
            <template #header>
              <span class="card-title">设备类型分布</span>
            </template>
            <div class="device-type-content">
              <DeviceTypeChart :data="deviceTypeDistribution" />
            </div>
          </el-card>
        </el-col>
      </el-row>
    </div>

    <!-- 客户端详情对话框 -->
    <ClientDetailsDialog
      v-model="showClientDetails"
      :client-id="selectedClientId"
      @refresh="refreshData"
    />

    <!-- 连接历史对话框 -->
    <ConnectionHistoryDialog
      v-model="showConnectionHistory"
      :client-id="selectedClientId"
    />

    <!-- 批量操作对话框 -->
    <BulkOperationsDialog
      v-model="showBulkOperations"
      :selected-clients="selectedClients"
      @complete="onBulkOperationComplete"
    />

    <!-- 设置对话框 -->
    <ClientManagementSettings
      v-model="showSettings"
      :settings="settings"
      @save="saveSettings"
    />
  </div>
</template>

<script setup>
import { ref, computed, onMounted, watch } from 'vue'
import { ElMessage, ElMessageBox } from 'element-plus'
import {
  Connection,
  Refresh,
  Setting,
  Search,
  RefreshLeft,
  List,
  Grid,
  ArrowDown,
  InfoFilled,
  Clock,
  Close,
  Lock,
  CircleFilled,
  WarningFilled,
  Clock as ClockIcon
} from '@element-plus/icons-vue'

// 组件导入
import ClientCard from '../components/ClientCard.vue'
import ClientDetailsDialog from '../components/ClientDetailsDialog.vue'
import ConnectionHistoryDialog from '../components/ConnectionHistoryDialog.vue'
import BulkOperationsDialog from '../components/BulkOperationsDialog.vue'
import ClientManagementSettings from '../components/ClientManagementSettings.vue'
import RealtimeStats from '../components/RealtimeStats.vue'
import ConnectionMap from '../components/ConnectionMap.vue'
import DeviceTypeChart from '../components/DeviceTypeChart.vue'

// API导入
import { 
  getClientsList, 
  disconnectClient, 
  batchDisconnectClients,
  setClientAccess 
} from '@/api/nfcRelayAdmin'
import { formatTime } from '@/utils/format'

// 响应式数据
const loading = ref(false)
const clients = ref([])
const selectedClients = ref([])
const viewMode = ref('table')
const showClientDetails = ref(false)
const showConnectionHistory = ref(false)
const showBulkOperations = ref(false)
const showSettings = ref(false)
const selectedClientId = ref('')

// 筛选和分页
const filters = ref({
  search: '',
  status: '',
  deviceType: '',
  connectionTime: null
})

const pagination = ref({
  current: 1,
  size: 20
})

// 设置
const settings = ref({
  autoRefresh: true,
  refreshInterval: 30,
  showPerformanceMetrics: true,
  enableGeolocation: true
})

// 计算属性
const statistics = computed(() => {
  const total = clients.value.length
  const online = clients.value.filter(c => c.status === 'online').length
  const offline = total - online
  
  return { total, online, offline }
})

const filteredClients = computed(() => {
  let filtered = [...clients.value]
  
  // 搜索筛选
  if (filters.value.search) {
    const search = filters.value.search.toLowerCase()
    filtered = filtered.filter(client => 
      client.clientId.toLowerCase().includes(search) ||
      client.deviceName?.toLowerCase().includes(search) ||
      client.ipAddress.includes(search)
    )
  }
  
  // 状态筛选
  if (filters.value.status) {
    filtered = filtered.filter(client => client.status === filters.value.status)
  }
  
  // 设备类型筛选
  if (filters.value.deviceType) {
    filtered = filtered.filter(client => client.deviceType === filters.value.deviceType)
  }
  
  // 时间范围筛选
  if (filters.value.connectionTime) {
    const [start, end] = filters.value.connectionTime
    filtered = filtered.filter(client => {
      const connectedAt = new Date(client.connectedAt)
      return connectedAt >= start && connectedAt <= end
    })
  }
  
  return filtered
})

const paginatedClients = computed(() => {
  const start = (pagination.value.current - 1) * pagination.value.size
  const end = start + pagination.value.size
  return filteredClients.value.slice(start, end)
})

const deviceTypeDistribution = computed(() => {
  const distribution = {}
  clients.value.forEach(client => {
    distribution[client.deviceType] = (distribution[client.deviceType] || 0) + 1
  })
  
  return Object.entries(distribution).map(([type, count]) => ({
    name: type,
    value: count
  }))
})

const realtimeStats = computed(() => ({
  totalConnections: statistics.value.total,
  onlineConnections: statistics.value.online,
  avgResponseTime: calculateAvgResponseTime(),
  totalThroughput: calculateTotalThroughput(),
  errorRate: calculateErrorRate()
}))

// 方法
const getStatusText = (status) => {
  const texts = {
    online: '在线',
    offline: '离线',
    reconnecting: '重连中'
  }
  return texts[status] || status
}

const getStatusClass = (status) => {
  const classes = {
    online: 'status-online',
    offline: 'status-offline',
    reconnecting: 'status-reconnecting'
  }
  return classes[status] || 'status-unknown'
}

const getStatusIcon = (status) => {
  const icons = {
    online: CircleFilled,
    offline: CircleFilled,
    reconnecting: WarningFilled
  }
  return icons[status] || CircleFilled
}

const getConnectionDuration = (connectedAt) => {
  const now = new Date()
  const connected = new Date(connectedAt)
  const duration = now - connected
  
  const hours = Math.floor(duration / (1000 * 60 * 60))
  const minutes = Math.floor((duration % (1000 * 60 * 60)) / (1000 * 60))
  
  if (hours > 0) {
    return `${hours}小时${minutes}分钟`
  }
  return `${minutes}分钟`
}

const getPerformanceType = (responseTime) => {
  if (responseTime < 100) return 'success'
  if (responseTime < 300) return 'warning'
  return 'danger'
}

const calculateAvgResponseTime = () => {
  if (clients.value.length === 0) return 0
  const total = clients.value.reduce((sum, client) => sum + (client.avgResponseTime || 0), 0)
  return Math.round(total / clients.value.length)
}

const calculateTotalThroughput = () => {
  return clients.value.reduce((sum, client) => sum + (client.throughput || 0), 0)
}

const calculateErrorRate = () => {
  if (clients.value.length === 0) return 0
  const totalErrors = clients.value.reduce((sum, client) => sum + (client.errorCount || 0), 0)
  const totalRequests = clients.value.reduce((sum, client) => sum + (client.requestCount || 0), 0)
  return totalRequests > 0 ? (totalErrors / totalRequests * 100).toFixed(2) : 0
}

const refreshData = async () => {
  try {
    loading.value = true
    const response = await getClientsList({
      ...filters.value,
      page: pagination.value.current,
      pageSize: pagination.value.size
    })
    clients.value = response.data.list || []
  } catch (error) {
    ElMessage.error('获取客户端列表失败: ' + error.message)
  } finally {
    loading.value = false
  }
}

const applyFilters = () => {
  pagination.value.current = 1
  refreshData()
}

const resetFilters = () => {
  filters.value = {
    search: '',
    status: '',
    deviceType: '',
    connectionTime: null
  }
  applyFilters()
}

const onSelectionChange = (selection) => {
  selectedClients.value = selection.map(item => item.clientId)
}

const onRowClick = (row) => {
  selectedClientId.value = row.clientId
  showClientDetails.value = true
}

const onClientCardClick = (client) => {
  if (selectedClients.value.includes(client.clientId)) {
    selectedClients.value = selectedClients.value.filter(id => id !== client.clientId)
  } else {
    selectedClients.value.push(client.clientId)
  }
}

const onPageSizeChange = () => {
  pagination.value.current = 1
  refreshData()
}

const onPageChange = () => {
  refreshData()
}

const handleAction = async (action, client) => {
  selectedClientId.value = client.clientId
  
  try {
    switch (action) {
      case 'details':
        showClientDetails.value = true
        break
        
      case 'history':
        showConnectionHistory.value = true
        break
        
      case 'disconnect':
        await ElMessageBox.confirm('确定要断开此连接吗？', '确认断开')
        await disconnectClient(client.clientId)
        ElMessage.success('连接已断开')
        refreshData()
        break
        
      case 'blacklist':
        await ElMessageBox.confirm('确定要将此客户端加入黑名单吗？', '确认操作')
        await setClientAccess({
          clientId: client.clientId,
          action: 'blacklist'
        })
        ElMessage.success('已加入黑名单')
        refreshData()
        break
    }
  } catch (error) {
    if (error !== 'cancel') {
      ElMessage.error('操作失败: ' + error.message)
    }
  }
}

const onBulkOperationComplete = () => {
  selectedClients.value = []
  refreshData()
}

const saveSettings = (newSettings) => {
  Object.assign(settings.value, newSettings)
  localStorage.setItem('clientManagementSettings', JSON.stringify(settings.value))
  showSettings.value = false
  ElMessage.success('设置已保存')
}

const loadSettings = () => {
  const saved = localStorage.getItem('clientManagementSettings')
  if (saved) {
    try {
      Object.assign(settings.value, JSON.parse(saved))
    } catch (error) {
      console.error('加载设置失败:', error)
    }
  }
}

// 自动刷新
let refreshTimer = null
const setupAutoRefresh = () => {
  if (refreshTimer) {
    clearInterval(refreshTimer)
    refreshTimer = null
  }
  
  if (settings.value.autoRefresh) {
    refreshTimer = setInterval(() => {
      refreshData()
    }, settings.value.refreshInterval * 1000)
  }
}

// 生命周期
onMounted(() => {
  loadSettings()
  refreshData()
  setupAutoRefresh()
})

// 监听设置变化
watch(() => settings.value.autoRefresh, setupAutoRefresh)
watch(() => settings.value.refreshInterval, setupAutoRefresh)
</script>

<style scoped lang="scss">
.enhanced-client-management {
  padding: 16px;
  background: #f5f7fa;
  min-height: 100vh;
  
  .page-header {
    display: flex;
    justify-content: space-between;
    align-items: center;
    margin-bottom: 16px;
    padding: 20px;
    background: white;
    border-radius: 8px;
    box-shadow: 0 2px 8px rgba(0, 0, 0, 0.1);
    
    .header-left {
      .page-title {
        display: flex;
        align-items: center;
        gap: 8px;
        margin: 0 0 16px 0;
        font-size: 20px;
        font-weight: 600;
        color: #303133;
      }
      
      .statistics {
        display: flex;
        gap: 24px;
        
        .stat-item {
          :deep(.el-statistic__content) {
            font-size: 18px;
            font-weight: 600;
          }
          
          &.stat-online :deep(.el-statistic__content) {
            color: #67c23a;
          }
          
          &.stat-offline :deep(.el-statistic__content) {
            color: #f56c6c;
          }
        }
      }
    }
    
    .header-right {
      display: flex;
      gap: 12px;
    }
  }
  
  .filters-section {
    margin-bottom: 16px;
    
    .filter-form {
      margin: 0;
      
      :deep(.el-form-item) {
        margin-bottom: 0;
      }
    }
  }
  
  .content-section {
    .clients-card {
      min-height: 600px;
      
      .card-header {
        display: flex;
        justify-content: space-between;
        align-items: center;
        
        .card-title {
          font-weight: 600;
          color: #303133;
        }
      }
      
      .table-view {
        .clients-table {
          .status-indicator {
            display: flex;
            justify-content: center;
            
            .status-online {
              color: #67c23a;
            }
            
            .status-offline {
              color: #909399;
            }
            
            .status-reconnecting {
              color: #e6a23c;
            }
          }
          
          .client-info {
            .client-id {
              font-weight: 500;
              color: #303133;
            }
            
            .client-type {
              font-size: 12px;
              color: #909399;
            }
          }
          
          .device-info {
            .device-name {
              color: #303133;
            }
            
            .device-version {
              font-size: 12px;
              color: #909399;
            }
          }
          
          .network-info {
            .ip-address {
              font-family: monospace;
              color: #303133;
            }
            
            .location {
              font-size: 12px;
              color: #909399;
            }
          }
          
          .time-info {
            .duration {
              font-size: 12px;
              color: #909399;
            }
          }
          
          .performance-info {
            .throughput {
              font-size: 12px;
              color: #909399;
              margin-top: 2px;
            }
          }
        }
      }
      
      .card-view {
        .client-card-col {
          margin-bottom: 16px;
        }
      }
      
      .pagination-wrapper {
        margin-top: 16px;
        display: flex;
        justify-content: center;
      }
    }
    
    .stats-card,
    .geographic-card,
    .device-type-card {
      margin-bottom: 16px;
      
      .card-title {
        font-weight: 600;
        color: #303133;
      }
    }
  }
}

// 响应式设计
@media (max-width: 1200px) {
  .enhanced-client-management {
    .content-section {
      :deep(.el-col) {
        &:first-child {
          span: 24;
          margin-bottom: 16px;
        }
        
        &:last-child {
          span: 24;
        }
      }
    }
  }
}

@media (max-width: 768px) {
  .enhanced-client-management {
    padding: 8px;
    
    .page-header {
      flex-direction: column;
      gap: 16px;
      text-align: center;
      
      .statistics {
        flex-direction: column;
        gap: 12px;
      }
    }
    
    .filters-section {
      :deep(.el-form--inline) .el-form-item {
        display: block;
        margin-right: 0;
        margin-bottom: 12px;
      }
    }
  }
}

// 暗色主题
.dark .enhanced-client-management {
  background: #1a1a1a;
  
  .page-header,
  .filters-section .el-card,
  .clients-card,
  .stats-card,
  .geographic-card,
  .device-type-card {
    background: #2d2d2d;
    border-color: #404040;
  }
  
  .page-title,
  .card-title {
    color: #e5eaf3;
  }
}
</style> 