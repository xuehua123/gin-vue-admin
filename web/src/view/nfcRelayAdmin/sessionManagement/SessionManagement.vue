<!--
  NFC中继管理 - 会话管理页面
  展示当前所有活动的NFC中继会话，并提供管理操作
-->
<template>
  <div class="session-management">
    <!-- 页面头部 -->
    <div class="page-header">
      <div class="header-left">
        <h1 class="page-title">会话管理</h1>
        <div class="page-description">
          管理当前所有活动的NFC中继会话，监控APDU交换状态
        </div>
      </div>
      
      <div class="header-actions">
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
            title="总会话数"
            :value="stats.total"
            icon="Connection"
            icon-color="#409EFF"
            :subtitle="`活跃: ${stats.active}`"
          />
        </el-col>
        <el-col :span="6">
          <stat-card
            title="已配对"
            :value="stats.paired"
            icon="Link"
            icon-color="#67C23A"
            :subtitle="`${((stats.paired / stats.total) * 100 || 0).toFixed(1)}%`"
          />
        </el-col>
        <el-col :span="6">
          <stat-card
            title="等待配对"
            :value="stats.waiting"
            icon="Clock"
            icon-color="#E6A23C"
            :subtitle="`${((stats.waiting / stats.total) * 100 || 0).toFixed(1)}%`"
          />
        </el-col>
        <el-col :span="6">
          <stat-card
            title="APDU总量"
            :value="stats.totalApdu"
            icon="DocumentCopy"
            icon-color="#F56C6C"
            :subtitle="`今日: ${stats.todayApdu}`"
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
    <div v-if="selectedSessions.length > 0" class="bulk-actions">
      <div class="selection-info">
        已选择 {{ selectedSessions.length }} 个会话
      </div>
      <div class="bulk-buttons">
        <el-button 
          type="danger" 
          :icon="Delete"
          @click="showBatchTerminateConfirm"
        >
          批量终止
        </el-button>
        <el-button @click="clearSelection">
          取消选择
        </el-button>
      </div>
    </div>

    <!-- 会话列表 -->
    <div class="table-section">
      <data-table
        :data="sessionList"
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
        <!-- 自定义会话ID列 -->
        <template #sessionId="{ row }">
          <el-button 
            type="primary" 
            link 
            size="small"
            @click="showSessionDetail(row)"
          >
            {{ formatSessionId(row.session_id) }}
          </el-button>
        </template>

        <!-- 自定义参与者列 -->
        <template #participants="{ row }">
          <div class="participants-info">
            <div class="participant">
              <el-tag type="primary" size="small">传卡</el-tag>
              <span class="participant-name">{{ row.provider_display_name || row.provider_client_id }}</span>
            </div>
            <el-icon class="exchange-icon"><Right /></el-icon>
            <div class="participant">
              <el-tag type="success" size="small">收卡</el-tag>
              <span class="participant-name">{{ row.receiver_display_name || row.receiver_client_id }}</span>
            </div>
          </div>
        </template>

        <!-- 自定义性能指标列 -->
        <template #performance="{ row }">
          <div class="performance-metrics">
            <div class="metric">
              <span class="metric-label">APDU:</span>
              <span class="metric-value">{{ row.apdu_count || 0 }}</span>
            </div>
            <div class="metric">
              <span class="metric-label">延迟:</span>
              <span class="metric-value">{{ formatLatency(row.avg_latency) }}</span>
            </div>
          </div>
        </template>
      </data-table>
    </div>

    <!-- 确认对话框 -->
    <confirm-dialog
      v-model="confirmDialogVisible"
      :title="confirmDialogTitle"
      :message="confirmDialogMessage"
      :description="confirmDialogDescription"
      :type="confirmDialogType"
      :loading="confirmDialogLoading"
      :require-input="confirmDialogRequireInput"
      :input-validation="confirmDialogInputValidation"
      @confirm="handleConfirm"
      @cancel="handleCancel"
    />

    <!-- 会话详情对话框 -->
    <session-detail-dialog
      v-model="detailDialog.visible"
      :session-data="detailDialog.data"
      @refresh="handleRefresh"
    />
  </div>
</template>

<script setup>
import { ref, reactive, computed, onMounted } from 'vue'
import { useRouter } from 'vue-router'
import { 
  Refresh, 
  DataBoard,
  Delete,
  Right,
  View,
  CircleClose,
  Connection,
  Link,
  Clock,
  DocumentCopy
} from '@element-plus/icons-vue'
import { ElMessage } from 'element-plus'

// 组件导入
import { StatCard, SearchForm, DataTable, ConfirmDialog } from '../components'
import SessionDetailDialog from './components/SessionDetailDialog.vue'
import { useRealTimeData } from '../hooks/useRealTime'
import { useMockSessionData } from '../hooks/useMockData'
import { 
  formatDateTime, 
  formatDuration,
  formatSessionId,
  formatLatency,
  formatFileSize
} from '../utils/formatters'

// API导入
import { 
  getSessionsList, 
  getSessionDetails,
  terminateSession,
  batchTerminateSessions
} from '@/api/nfcRelayAdmin'

defineOptions({
  name: 'SessionManagement'
})

const router = useRouter()

// 使用模拟数据hook
const { generateMockSessions } = useMockSessionData()

// 状态管理
const loading = ref(false)
const sessionList = ref([])
const selectedSessions = ref([])

// 分页
const pagination = reactive({
  page: 1,
  pageSize: 20,
  total: 0
})

// 搜索参数
const searchParams = reactive({
  sessionID: '',
  participantClientID: '',
  participantUserID: '',
  status: '',
  sessionType: '',
  durationRange: ''
})

// 确认对话框
const confirmDialogVisible = ref(false)
const confirmDialogTitle = ref('')
const confirmDialogMessage = ref('')
const confirmDialogDescription = ref('')
const confirmDialogType = ref('warning')
const confirmDialogLoading = ref(false)
const confirmDialogRequireInput = ref(false)
const confirmDialogInputValidation = ref(null)
const confirmDialogAction = ref(null)
const confirmDialogData = ref(null)

// 详情对话框
const detailDialog = reactive({
  visible: false,
  data: null
})

// 搜索字段配置
const searchFields = [
  {
    key: 'sessionID',
    label: '会话ID',
    type: 'input',
    placeholder: '请输入会话ID',
    clearable: true
  },
  {
    key: 'participantClientID',
    label: '参与者客户端ID',
    type: 'input',
    placeholder: '请输入客户端ID',
    clearable: true
  },
  {
    key: 'participantUserID',
    label: '参与者用户ID',
    type: 'input',
    placeholder: '请输入用户ID',
    clearable: true
  },
  {
    key: 'status',
    label: '会话状态',
    type: 'select',
    placeholder: '请选择状态',
    options: [
      { label: '已配对', value: 'paired' },
      { label: '等待配对', value: 'waiting' },
      { label: '已终止', value: 'terminated' }
    ]
  },
  {
    key: 'sessionType',
    label: '会话类型',
    type: 'select',
    placeholder: '请选择类型',
    options: [
      { label: '卡到POS', value: 'card_to_pos' },
      { label: 'POS到卡', value: 'pos_to_card' },
      { label: '点对点', value: 'peer_to_peer' }
    ]
  },
  {
    key: 'durationRange',
    label: '持续时间',
    type: 'select',
    placeholder: '请选择时间范围',
    options: [
      { label: '小于1分钟', value: 'lt_1m' },
      { label: '1-5分钟', value: '1m_5m' },
      { label: '5-15分钟', value: '5m_15m' },
      { label: '大于15分钟', value: 'gt_15m' }
    ]
  }
]

// 表格列配置
const columns = [
  {
    prop: 'session_id',
    label: '会话ID',
    width: 120,
    showOverflowTooltip: true,
    slot: 'sessionId'
  },
  {
    prop: 'participants',
    label: '参与者',
    minWidth: 300,
    slot: 'participants'
  },
  {
    prop: 'status',
    label: '状态',
    width: 100,
    type: 'tag',
    tagMap: {
      paired: { text: '已配对', type: 'success' },
      waiting: { text: '等待配对', type: 'warning' },
      terminated: { text: '已终止', type: 'info' }
    }
  },
  {
    prop: 'session_type',
    label: '类型',
    width: 120,
    type: 'tag',
    tagMap: {
      card_to_pos: { text: '卡→POS', type: 'primary' },
      pos_to_card: { text: 'POS→卡', type: 'success' },
      peer_to_peer: { text: '点对点', type: 'warning' }
    }
  },
  {
    prop: 'created_at',
    label: '创建时间',
    width: 160,
    type: 'datetime'
  },
  {
    prop: 'last_activity_at',
    label: '最后活动',
    width: 160,
    type: 'datetime'
  },
  {
    prop: 'performance',
    label: '性能指标',
    width: 140,
    slot: 'performance'
  }
]

// 操作按钮配置
const actions = [
  {
    key: 'view',
    label: '详情',
    type: 'primary',
    icon: View
  },
  {
    key: 'terminate',
    label: '终止会话',
    type: 'danger',
    icon: CircleClose,
    disabled: (row) => row.status === 'terminated'
  }
]

// 统计信息
const stats = computed(() => {
  const total = sessionList.value.length
  const active = sessionList.value.filter(s => s.status !== 'terminated').length
  const paired = sessionList.value.filter(s => s.status === 'paired').length
  const waiting = sessionList.value.filter(s => s.status === 'waiting').length
  const totalApdu = sessionList.value.reduce((sum, s) => sum + (s.apdu_count || 0), 0)
  const todayApdu = Math.floor(totalApdu * 0.8) // 模拟今日数据
  
  return {
    total,
    active,
    paired,
    waiting,
    totalApdu,
    todayApdu
  }
})

// 数据获取
const fetchSessionList = async () => {
  try {
    loading.value = true
    
    const params = {
      page: pagination.page,
      pageSize: pagination.pageSize,
      ...searchParams
    }
    
    const response = await getSessionsList(params)
    
    if (response.code === 0) {
      sessionList.value = response.data.list || []
      pagination.total = response.data.total || 0
    } else {
      throw new Error(response.msg || '获取会话列表失败')
    }
  } catch (error) {
    console.warn('API调用失败，使用模拟数据:', error.message)
    ElMessage.warning('连接后端失败，正在使用模拟数据进行演示')
    
    // 使用模拟数据
    const mockData = generateMockSessions(pagination.pageSize)
    sessionList.value = mockData
    pagination.total = 200 // 模拟总数
  } finally {
    loading.value = false
  }
}

// 使用实时数据更新
const { refresh: realTimeRefresh } = useRealTimeData(fetchSessionList, 30000)

// 事件处理
const handleRefresh = async () => {
  await fetchSessionList()
  ElMessage.success('列表已刷新')
}

const handleSearch = (params) => {
  Object.assign(searchParams, params)
  pagination.page = 1
  fetchSessionList()
}

const handleReset = () => {
  Object.keys(searchParams).forEach(key => {
    searchParams[key] = ''
  })
  pagination.page = 1
  fetchSessionList()
}

const handleSelectionChange = (selection) => {
  selectedSessions.value = selection
}

const handlePageChange = ({ page }) => {
  pagination.page = page
  fetchSessionList()
}

const handleSizeChange = ({ pageSize }) => {
  pagination.pageSize = pageSize
  pagination.page = 1
  fetchSessionList()
}

const handleAction = ({ action, row }) => {
  switch (action) {
    case 'view':
      showSessionDetail(row)
      break
    case 'terminate':
      showTerminateConfirm(row)
      break
  }
}

const showSessionDetail = (session) => {
  detailDialog.data = session
  detailDialog.visible = true
}

const showTerminateConfirm = (session) => {
  confirmDialogVisible.value = true
  confirmDialogTitle.value = '确认终止会话'
  confirmDialogMessage.value = `确定要强制终止会话 "${formatSessionId(session.session_id)}" 吗？`
  confirmDialogDescription.value = '此操作将立即终止NFC会话，可能会影响正在进行的APDU交换。'
  confirmDialogType.value = 'warning'
  confirmDialogRequireInput.value = true
  confirmDialogInputValidation.value = '终止会话'
  confirmDialogAction.value = 'terminate'
  confirmDialogData.value = session
}

const showBatchTerminateConfirm = () => {
  confirmDialogVisible.value = true
  confirmDialogTitle.value = '批量终止会话'
  confirmDialogMessage.value = `确定要终止选中的 ${selectedSessions.value.length} 个会话吗？`
  confirmDialogDescription.value = '此操作将立即终止所有选中的会话，可能会影响正在进行的APDU交换。'
  confirmDialogType.value = 'warning'
  confirmDialogRequireInput.value = true
  confirmDialogInputValidation.value = '批量终止'
  confirmDialogAction.value = 'batchTerminate'
  confirmDialogData.value = selectedSessions.value
}

const clearSelection = () => {
  selectedSessions.value = []
}

const handleConfirm = async ({ inputValue }) => {
  if (confirmDialogAction.value === 'terminate') {
    await executeTerminate(confirmDialogData.value)
  } else if (confirmDialogAction.value === 'batchTerminate') {
    await executeBatchTerminate(confirmDialogData.value)
  }
}

const handleCancel = () => {
  confirmDialogVisible.value = false
  confirmDialogAction.value = null
  confirmDialogData.value = null
}

const executeTerminate = async (session) => {
  try {
    confirmDialogLoading.value = true
    
    const response = await terminateSession(session.session_id)
    
    if (response.code === 0) {
      ElMessage.success('会话已终止')
      confirmDialogVisible.value = false
      await fetchSessionList() // 刷新列表
    } else {
      throw new Error(response.msg || '终止会话失败')
    }
  } catch (error) {
    ElMessage.error('终止会话失败: ' + error.message)
  } finally {
    confirmDialogLoading.value = false
  }
}

const executeBatchTerminate = async (sessions) => {
  try {
    confirmDialogLoading.value = true
    
    const sessionIds = sessions.map(s => s.session_id)
    const response = await batchTerminateSessions(sessionIds)
    
    if (response.code === 0) {
      ElMessage.success(`成功终止 ${sessionIds.length} 个会话`)
      confirmDialogVisible.value = false
      clearSelection()
      await fetchSessionList() // 刷新列表
    } else {
      throw new Error(response.msg || '批量终止失败')
    }
  } catch (error) {
    ElMessage.error('批量终止失败: ' + error.message)
  } finally {
    confirmDialogLoading.value = false
  }
}

// 生命周期
onMounted(() => {
  fetchSessionList()
})
</script>

<style scoped lang="scss">
.session-management {
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
  
  .participants-info {
    display: flex;
    align-items: center;
    gap: 8px;
    
    .participant {
      display: flex;
      align-items: center;
      gap: 4px;
      
      .participant-name {
        font-size: 12px;
        color: #606266;
        max-width: 80px;
        overflow: hidden;
        text-overflow: ellipsis;
        white-space: nowrap;
      }
    }
    
    .exchange-icon {
      color: #909399;
      font-size: 12px;
    }
  }
  
  .performance-metrics {
    font-size: 12px;
    
    .metric {
      display: flex;
      justify-content: space-between;
      margin-bottom: 2px;
      
      &:last-child {
        margin-bottom: 0;
      }
      
      .metric-label {
        color: #909399;
      }
      
      .metric-value {
        color: #303133;
        font-weight: 500;
      }
    }
  }
}

// 响应式设计
@media (max-width: 768px) {
  .session-management {
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