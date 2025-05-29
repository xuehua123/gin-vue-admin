<!--
  NFC中继管理 - 审计日志页面
  查看和分析系统的所有审计日志，提供强大的搜索和筛选功能
-->
<template>
  <div class="audit-logs">
    <!-- 页面头部 -->
    <div class="page-header">
      <div class="header-left">
        <h1 class="page-title">审计日志</h1>
        <div class="page-description">
          查看系统的审计日志，追踪所有关键操作和事件
        </div>
      </div>
      
      <div class="header-actions">
        <el-button 
          :icon="Download" 
          @click="handleExport"
          :loading="exportLoading"
        >
          导出日志
        </el-button>
        <el-button 
          :icon="Refresh" 
          @click="handleRefresh"
          :loading="loading"
        >
          刷新列表
        </el-button>
        <el-button 
          :icon="DataBoard"
          @click="$router.push('/nfc-relay-admin/dashboard')"
        >
          返回仪表盘
        </el-button>
      </div>
    </div>

    <!-- 统计信息 -->
    <div class="stats-section">
      <el-row :gutter="16">
        <el-col :span="6">
          <stat-card
            title="总日志数"
            :value="stats.total"
            icon="Document"
            icon-color="#409EFF"
            :subtitle="`今日: ${stats.today}`"
          />
        </el-col>
        <el-col :span="6">
          <stat-card
            title="错误事件"
            :value="stats.errors"
            icon="Warning"
            icon-color="#F56C6C"
            :subtitle="`错误率: ${stats.errorRate}`"
          />
        </el-col>
        <el-col :span="6">
          <stat-card
            title="会话事件"
            :value="stats.sessionEvents"
            icon="Connection"
            icon-color="#67C23A"
            :subtitle="`占比: ${stats.sessionPercentage}`"
          />
        </el-col>
        <el-col :span="6">
          <stat-card
            title="APDU事件"
            :value="stats.apduEvents"
            icon="DocumentCopy"
            icon-color="#E6A23C"
            :subtitle="`成功率: ${stats.apduSuccessRate}`"
          />
        </el-col>
      </el-row>
    </div>

    <!-- 搜索和筛选 -->
    <div class="search-section">
      <search-form
        v-model="searchParams"
        :fields="searchFields"
        :loading="loading"
        @search="handleSearch"
        @reset="handleReset"
      />
    </div>

    <!-- 批量操作栏 -->
    <div v-if="selectedLogs.length > 0" class="bulk-actions">
      <div class="selection-info">
        已选择 {{ selectedLogs.length }} 条日志记录
      </div>
      <div class="bulk-buttons">
        <el-button 
          type="primary" 
          :icon="Download"
          @click="handleBatchExport"
        >
          导出选中
        </el-button>
        <el-button @click="clearSelection">
          取消选择
        </el-button>
      </div>
    </div>

    <!-- 日志列表 -->
    <div class="table-section">
      <data-table
        :data="logList"
        :columns="columns"
        :actions="actions"
        :loading="loading"
        :total="pagination.total"
        :current-page="pagination.page"
        :page-size="pagination.pageSize"
        selectable
        show-index
        @selection-change="handleSelectionChange"
        @action="handleAction"
        @page-change="handlePageChange"
        @size-change="handleSizeChange"
      >
        <!-- 自定义时间列 -->
        <template #timestamp="{ row }">
          <div class="timestamp-info">
            <div class="date">{{ formatDateTime(row.timestamp, 'MM-DD') }}</div>
            <div class="time">{{ formatDateTime(row.timestamp, 'HH:mm:ss') }}</div>
          </div>
        </template>

        <!-- 自定义事件类型列 -->
        <template #eventType="{ row }">
          <div class="event-type-info">
            <el-tag 
              :type="getEventTypeInfo(row.event_type).type"
              size="small"
            >
              <el-icon class="event-icon">
                <component :is="getEventTypeInfo(row.event_type).icon" />
              </el-icon>
              {{ getEventTypeInfo(row.event_type).text }}
            </el-tag>
          </div>
        </template>

        <!-- 自定义参与者列 -->
        <template #participants="{ row }">
          <div class="participants-info">
            <div v-if="row.client_id_initiator" class="participant">
              <el-tag type="primary" size="small">发起</el-tag>
              <el-button 
                type="primary" 
                link 
                size="small"
                @click="viewClient(row.client_id_initiator)"
              >
                {{ formatClientId(row.client_id_initiator) }}
              </el-button>
            </div>
            <div v-if="row.client_id_responder" class="participant">
              <el-tag type="success" size="small">响应</el-tag>
              <el-button 
                type="success" 
                link 
                size="small"
                @click="viewClient(row.client_id_responder)"
              >
                {{ formatClientId(row.client_id_responder) }}
              </el-button>
            </div>
            <div v-if="!row.client_id_initiator && !row.client_id_responder" class="no-participants">
              <span class="text-muted">-</span>
            </div>
          </div>
        </template>

        <!-- 自定义关联对象列 -->
        <template #relatedObjects="{ row }">
          <div class="related-objects">
            <div v-if="row.session_id" class="related-item">
              <el-button 
                type="warning" 
                link 
                size="small"
                @click="viewSession(row.session_id)"
              >
                <el-icon><Connection /></el-icon>
                {{ formatSessionId(row.session_id) }}
              </el-button>
            </div>
            <div v-if="row.user_id" class="related-item">
              <el-tag size="small" type="info">{{ row.user_id }}</el-tag>
            </div>
          </div>
        </template>

        <!-- 自定义详情预览列 -->
        <template #detailsPreview="{ row }">
          <div class="details-preview">
            <div class="preview-text">{{ getDetailsPreview(row.details) }}</div>
            <el-button 
              v-if="hasDetails(row.details)"
              type="primary" 
              link 
              size="small"
              @click="showLogDetail(row)"
            >
              查看详情
            </el-button>
          </div>
        </template>
      </data-table>
    </div>

    <!-- 日志详情对话框 -->
    <log-detail-dialog
      v-model="detailDialog.visible"
      :log-data="detailDialog.data"
      @refresh="handleRefresh"
    />
  </div>
</template>

<script setup>
import { ref, reactive, computed, onMounted } from 'vue'
import { useRouter, useRoute } from 'vue-router'
import { 
  Refresh, 
  DataBoard,
  Download,
  Warning,
  Document,
  DocumentCopy,
  Connection,
  SuccessFilled,
  CircleClose,
  Switch,
  UserFilled,
  Setting
} from '@element-plus/icons-vue'
import { ElMessage } from 'element-plus'

// 组件导入
import { StatCard, SearchForm, DataTable } from '../components'
import LogDetailDialog from './components/LogDetailDialog.vue'
import { useRealTimeData } from '../hooks/useRealTime'
import { useMockAuditLogData } from '../hooks/useMockData'
import { 
  formatDateTime, 
  formatClientId,
  formatSessionId,
  formatEventType
} from '../utils/formatters'

// API导入
import { 
  getAuditLogs,
  exportAuditLogs,
  batchExportAuditLogs
} from '@/api/nfcRelayAdmin'

defineOptions({
  name: 'AuditLogs'
})

const router = useRouter()
const route = useRoute()

// 使用模拟数据hook
const { generateMockAuditLogs } = useMockAuditLogData()

// 状态管理
const loading = ref(false)
const exportLoading = ref(false)
const logList = ref([])
const selectedLogs = ref([])

// 分页
const pagination = reactive({
  page: 1,
  pageSize: 20,
  total: 0
})

// 搜索参数
const searchParams = reactive({
  eventType: '',
  sessionID: '',
  clientID: '',
  userID: '',
  sourceIP: '',
  startTime: '',
  endTime: '',
  severity: ''
})

// 详情对话框
const detailDialog = reactive({
  visible: false,
  data: null
})

// 搜索字段配置
const searchFields = [
  {
    key: 'eventType',
    label: '事件类型',
    type: 'select',
    placeholder: '请选择事件类型',
    clearable: true,
    options: [
      { label: '会话建立', value: 'session_established' },
      { label: '会话终止', value: 'session_terminated' },
      { label: 'APDU转发成功', value: 'apdu_relayed_success' },
      { label: 'APDU转发失败', value: 'apdu_relayed_failure' },
      { label: '客户端连接', value: 'client_connected' },
      { label: '客户端断开', value: 'client_disconnected' },
      { label: '认证失败', value: 'auth_failure' },
      { label: '权限拒绝', value: 'permission_denied' }
    ]
  },
  {
    key: 'sessionID',
    label: '会话ID',
    type: 'input',
    placeholder: '请输入会话ID',
    clearable: true
  },
  {
    key: 'clientID',
    label: '客户端ID',
    type: 'input',
    placeholder: '请输入客户端ID',
    clearable: true
  },
  {
    key: 'userID',
    label: '用户ID',
    type: 'input',
    placeholder: '请输入用户ID',
    clearable: true
  },
  {
    key: 'sourceIP',
    label: '源IP地址',
    type: 'input',
    placeholder: '请输入IP地址',
    clearable: true
  },
  {
    key: 'timeRange',
    label: '时间范围',
    type: 'datetimerange',
    placeholder: '请选择时间范围',
    clearable: true
  },
  {
    key: 'severity',
    label: '严重级别',
    type: 'select',
    placeholder: '请选择级别',
    clearable: true,
    options: [
      { label: '低', value: 'low' },
      { label: '中', value: 'medium' },
      { label: '高', value: 'high' }
    ]
  }
]

// 表格列配置
const columns = [
  {
    prop: 'timestamp',
    label: '时间',
    width: 120,
    slot: 'timestamp'
  },
  {
    prop: 'event_type',
    label: '事件类型',
    width: 140,
    slot: 'eventType'
  },
  {
    prop: 'participants',
    label: '参与者',
    minWidth: 200,
    slot: 'participants'
  },
  {
    prop: 'related_objects',
    label: '关联对象',
    width: 180,
    slot: 'relatedObjects'
  },
  {
    prop: 'source_ip',
    label: '源IP',
    width: 120,
    showOverflowTooltip: true
  },
  {
    prop: 'severity',
    label: '级别',
    width: 80,
    type: 'tag',
    tagMap: {
      low: { text: '低', type: 'info' },
      medium: { text: '中', type: 'warning' },
      high: { text: '高', type: 'danger' }
    }
  },
  {
    prop: 'details',
    label: '详情',
    minWidth: 200,
    slot: 'detailsPreview'
  }
]

// 操作按钮配置
const actions = [
  {
    key: 'view',
    label: '详情',
    type: 'primary',
    icon: Document
  }
]

// 统计信息
const stats = computed(() => {
  const total = logList.value.length
  const errors = logList.value.filter(log => 
    log.event_type.includes('failure') || log.event_type.includes('error')
  ).length
  const sessionEvents = logList.value.filter(log => 
    log.event_type.includes('session')
  ).length
  const apduEvents = logList.value.filter(log => 
    log.event_type.includes('apdu')
  ).length
  const apduSuccess = logList.value.filter(log => 
    log.event_type === 'apdu_relayed_success'
  ).length
  
  return {
    total,
    today: Math.floor(total * 0.15), // 模拟今日数据
    errors,
    errorRate: total > 0 ? `${((errors / total) * 100).toFixed(1)}%` : '0%',
    sessionEvents,
    sessionPercentage: total > 0 ? `${((sessionEvents / total) * 100).toFixed(1)}%` : '0%',
    apduEvents,
    apduSuccessRate: apduEvents > 0 ? `${((apduSuccess / apduEvents) * 100).toFixed(1)}%` : '0%'
  }
})

// 工具函数
const getEventTypeInfo = (eventType) => {
  const { text, type } = formatEventType(eventType)
  
  const iconMap = {
    session_established: Connection,
    session_terminated: Switch,
    apdu_relayed_success: SuccessFilled,
    apdu_relayed_failure: CircleClose,
    client_connected: UserFilled,
    client_disconnected: UserFilled,
    auth_failure: Warning,
    permission_denied: Setting
  }
  
  return {
    text,
    type,
    icon: iconMap[eventType] || Document
  }
}

const getDetailsPreview = (details) => {
  if (!details || typeof details !== 'object') return '-'
  
  const preview = []
  if (details.message) preview.push(details.message)
  if (details.error) preview.push(`错误: ${details.error}`)
  if (details.duration) preview.push(`时长: ${details.duration}`)
  if (details.data_length) preview.push(`长度: ${details.data_length}`)
  
  return preview.length > 0 ? preview.join(', ') : '详细信息可用'
}

const hasDetails = (details) => {
  return details && typeof details === 'object' && Object.keys(details).length > 0
}

// 数据获取
const fetchLogList = async () => {
  try {
    loading.value = true
    
    const params = {
      page: pagination.page,
      pageSize: pagination.pageSize,
      ...searchParams
    }
    
    // 处理时间范围
    if (searchParams.timeRange && searchParams.timeRange.length === 2) {
      params.startTime = searchParams.timeRange[0]
      params.endTime = searchParams.timeRange[1]
    }
    
    const response = await getAuditLogs(params)
    
    if (response.code === 0) {
      logList.value = response.data.list || []
      pagination.total = response.data.total || 0
    } else {
      throw new Error(response.msg || '获取审计日志失败')
    }
  } catch (error) {
    console.warn('API调用失败，使用模拟数据:', error.message)
    ElMessage.warning('连接后端失败，正在使用模拟数据进行演示')
    
    // 使用模拟数据
    const mockData = generateMockAuditLogs(pagination.pageSize)
    logList.value = mockData
    pagination.total = 1000 // 模拟总数
  } finally {
    loading.value = false
  }
}

// 使用实时数据更新
const { refresh: realTimeRefresh } = useRealTimeData(fetchLogList, 60000) // 审计日志1分钟刷新一次

// 事件处理
const handleRefresh = async () => {
  await fetchLogList()
  ElMessage.success('列表已刷新')
}

const handleSearch = (params) => {
  Object.assign(searchParams, params)
  
  // 处理时间范围
  if (params.timeRange && params.timeRange.length === 2) {
    searchParams.startTime = params.timeRange[0]
    searchParams.endTime = params.timeRange[1]
  } else {
    searchParams.startTime = ''
    searchParams.endTime = ''
  }
  
  pagination.page = 1
  fetchLogList()
}

const handleReset = () => {
  Object.keys(searchParams).forEach(key => {
    if (key === 'timeRange') {
      searchParams[key] = []
    } else {
      searchParams[key] = ''
    }
  })
  pagination.page = 1
  fetchLogList()
}

const handleSelectionChange = (selection) => {
  selectedLogs.value = selection
}

const handlePageChange = ({ page }) => {
  pagination.page = page
  fetchLogList()
}

const handleSizeChange = ({ pageSize }) => {
  pagination.pageSize = pageSize
  pagination.page = 1
  fetchLogList()
}

const handleAction = ({ action, row }) => {
  switch (action) {
    case 'view':
      showLogDetail(row)
      break
  }
}

const showLogDetail = (log) => {
  detailDialog.data = log
  detailDialog.visible = true
}

const clearSelection = () => {
  selectedLogs.value = []
}

const viewClient = (clientId) => {
  router.push({
    path: '/nfc-relay-admin/clients',
    query: { clientId }
  })
}

const viewSession = (sessionId) => {
  router.push({
    path: '/nfc-relay-admin/sessions',
    query: { sessionId }
  })
}

const handleExport = async () => {
  try {
    exportLoading.value = true
    
    const params = {
      ...searchParams,
      startTime: searchParams.startTime,
      endTime: searchParams.endTime
    }
    
    const response = await exportAuditLogs(params)
    
    if (response.code === 0) {
      // 处理文件下载
      ElMessage.success('导出成功')
    } else {
      throw new Error(response.msg || '导出失败')
    }
  } catch (error) {
    ElMessage.error('导出功能暂未实现: ' + error.message)
  } finally {
    exportLoading.value = false
  }
}

const handleBatchExport = async () => {
  try {
    const logIds = selectedLogs.value.map(log => log.id)
    const response = await batchExportAuditLogs(logIds)
    
    if (response.code === 0) {
      ElMessage.success(`成功导出 ${logIds.length} 条日志`)
      clearSelection()
    } else {
      throw new Error(response.msg || '批量导出失败')
    }
  } catch (error) {
    ElMessage.error('批量导出失败: ' + error.message)
  }
}

// 初始化：处理路由参数
const initializeFromRoute = () => {
  const { sessionId, clientId, eventType } = route.query
  if (sessionId) searchParams.sessionID = sessionId
  if (clientId) searchParams.clientID = clientId
  if (eventType) searchParams.eventType = eventType
}

// 生命周期
onMounted(() => {
  initializeFromRoute()
  fetchLogList()
})
</script>

<style scoped lang="scss">
.audit-logs {
  padding: 20px;
  
  .page-header {
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
      
      .page-description {
        font-size: 14px;
        color: #606266;
        line-height: 1.5;
      }
    }
    
    .header-actions {
      display: flex;
      gap: 12px;
    }
  }
  
  .stats-section {
    margin-bottom: 24px;
  }
  
  .search-section {
    margin-bottom: 16px;
  }
  
  .bulk-actions {
    display: flex;
    justify-content: space-between;
    align-items: center;
    padding: 12px 16px;
    margin-bottom: 16px;
    background: #e8f4fd;
    border: 1px solid #b3d8ff;
    border-radius: 6px;
    
    .selection-info {
      font-size: 14px;
      color: #409EFF;
      font-weight: 500;
    }
    
    .bulk-buttons {
      display: flex;
      gap: 8px;
    }
  }
  
  .table-section {
    background: white;
    border-radius: 8px;
    overflow: hidden;
    box-shadow: 0 2px 8px rgba(0, 0, 0, 0.1);
  }
  
  .timestamp-info {
    font-size: 12px;
    text-align: center;
    
    .date {
      color: #303133;
      font-weight: 500;
    }
    
    .time {
      color: #909399;
      margin-top: 2px;
    }
  }
  
  .event-type-info {
    .event-icon {
      margin-right: 4px;
    }
  }
  
  .participants-info {
    .participant {
      display: flex;
      align-items: center;
      gap: 4px;
      margin-bottom: 4px;
      
      &:last-child {
        margin-bottom: 0;
      }
    }
    
    .no-participants {
      color: #c0c4cc;
      font-style: italic;
    }
  }
  
  .related-objects {
    .related-item {
      margin-bottom: 4px;
      
      &:last-child {
        margin-bottom: 0;
      }
    }
  }
  
  .details-preview {
    .preview-text {
      font-size: 12px;
      color: #606266;
      margin-bottom: 4px;
      line-height: 1.4;
      overflow: hidden;
      text-overflow: ellipsis;
      display: -webkit-box;
      -webkit-line-clamp: 2;
      -webkit-box-orient: vertical;
    }
  }
  
  .text-muted {
    color: #c0c4cc;
  }
}

// 响应式设计
@media (max-width: 768px) {
  .audit-logs {
    padding: 16px;
    
    .page-header {
      flex-direction: column;
      align-items: flex-start;
      gap: 16px;
    }
    
    .stats-section {
      :deep(.el-col) {
        margin-bottom: 16px;
      }
    }
    
    .bulk-actions {
      flex-direction: column;
      align-items: flex-start;
      gap: 12px;
    }
  }
}
</style> 