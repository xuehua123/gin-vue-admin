<!--
  NFC中继管理 - 连接管理页面
  展示当前所有连接的客户端，并提供管理操作
-->
<template>
  <div class="client-management">
    <!-- 页面头部 -->
    <div class="page-header">
      <div class="header-left">
        <h1 class="page-title">连接管理</h1>
        <div class="page-description">
          管理当前通过WebSocket连接到NFC中继Hub的所有客户端
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
          :icon="Setting"
          @click="$router.push('/nfc-relay-admin/configuration')"
        >
          系统配置
        </el-button>
      </div>
    </div>

    <!-- 统计信息 -->
    <div class="stats-section">
      <el-row :gutter="16">
        <el-col :span="6">
          <stat-card
            title="总连接数"
            :value="stats.total"
            icon="Connection"
            icon-color="#409EFF"
            :subtitle="`在线: ${stats.online}`"
          />
        </el-col>
        <el-col :span="6">
          <stat-card
            title="传卡端"
            :value="stats.providers"
            icon="CreditCard"
            icon-color="#67C23A"
            :subtitle="`${((stats.providers / stats.total) * 100 || 0).toFixed(1)}%`"
          />
        </el-col>
        <el-col :span="6">
          <stat-card
            title="收卡端"
            :value="stats.receivers"
            icon="Monitor"
            icon-color="#E6A23C"
            :subtitle="`${((stats.receivers / stats.total) * 100 || 0).toFixed(1)}%`"
          />
        </el-col>
        <el-col :span="6">
          <stat-card
            title="未分配"
            :value="stats.unassigned"
            icon="User"
            icon-color="#909399"
            :subtitle="`${((stats.unassigned / stats.total) * 100 || 0).toFixed(1)}%`"
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

    <!-- 客户端列表 -->
    <div class="table-section">
      <data-table
        :data="clientList"
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
      />
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

    <!-- 客户端详情对话框 -->
    <client-detail-dialog
      v-model="detailDialog.visible"
      :client-data="detailDialog.data"
      @refresh="handleRefresh"
    />
  </div>
</template>

<script setup>
import { ref, reactive, computed, onMounted } from 'vue'
import { useRouter } from 'vue-router'
import { 
  Refresh, 
  Setting, 
  View, 
  Delete,
  Connection,
  CreditCard,
  Monitor,
  User,
  InfoFilled
} from '@element-plus/icons-vue'
import { ElMessage, ElMessageBox } from 'element-plus'

// 组件导入
import { StatCard, SearchForm, DataTable, ConfirmDialog } from '../components'
import ClientDetailDialog from './components/ClientDetailDialog.vue'
import { useRealTimeData } from '../hooks/useRealTime'
import { useMockClientData } from '../hooks/useMockData'
import { 
  formatDateTime, 
  formatIPAddress, 
  formatClientId, 
  formatOnlineStatus,
  formatRole 
} from '../utils/formatters'

// API导入 (需要根据实际API文件位置调整)
import { 
  getClientsList as getClientList, 
  getClientDetails,
  disconnectClient 
} from '@/api/nfcRelayAdmin'

defineOptions({
  name: 'ClientManagement'
})

const router = useRouter()

// 使用模拟数据hook
const { generateMockClients } = useMockClientData()

// 状态管理
const loading = ref(false)
const clientList = ref([])
const selectedClients = ref([])

// 分页
const pagination = reactive({
  page: 1,
  pageSize: 20,
  total: 0
})

// 搜索参数
const searchParams = reactive({
  clientID: '',
  userID: '',
  role: '',
  ipAddress: '',
  isOnline: ''
})

// 确认对话框 - 修改为独立的 ref 以避免响应式警告
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
    prop: 'clientID',
    label: '客户端ID',
    type: 'input',
    placeholder: '请输入客户端ID',
    clearable: true
  },
  {
    prop: 'userID',
    label: '用户ID',
    type: 'input',
    placeholder: '请输入用户ID',
    clearable: true
  },
  {
    prop: 'role',
    label: '角色',
    type: 'select',
    placeholder: '请选择角色',
    options: [
      { label: '传卡端', value: 'provider' },
      { label: '收卡端', value: 'receiver' },
      { label: '未分配', value: 'none' }
    ]
  },
  {
    prop: 'ipAddress',
    label: 'IP地址',
    type: 'input',
    placeholder: '请输入IP地址',
    clearable: true
  },
  {
    prop: 'isOnline',
    label: '在线状态',
    type: 'select',
    placeholder: '请选择状态',
    options: [
      { label: '在线', value: '1' },
      { label: '离线', value: '0' }
    ]
  }
]

// 表格列配置
const columns = [
  {
    prop: 'client_id',
    label: '客户端ID',
    width: 120,
    showOverflowTooltip: true,
    slot: 'clientId'
  },
  {
    prop: 'user_id',
    label: '用户ID',
    width: 120,
    showOverflowTooltip: true
  },
  {
    prop: 'display_name',
    label: '显示名称',
    minWidth: 150,
    showOverflowTooltip: true
  },
  {
    prop: 'role',
    label: '角色',
    width: 100,
    type: 'tag',
    tagMap: {
      provider: { text: '传卡端', type: 'primary' },
      receiver: { text: '收卡端', type: 'success' },
      none: { text: '未分配', type: 'info' }
    }
  },
  {
    prop: 'ip_address',
    label: 'IP地址',
    width: 130
  },
  {
    prop: 'connected_at',
    label: '连接时间',
    width: 160,
    type: 'datetime'
  },
  {
    prop: 'is_online',
    label: '在线状态',
    width: 100,
    type: 'tag',
    tagMap: {
      true: { text: '在线', type: 'success' },
      false: { text: '离线', type: 'danger' }
    }
  },
  {
    prop: 'session_id',
    label: '当前会话',
    width: 120,
    showOverflowTooltip: true,
    slot: 'sessionId'
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
    key: 'disconnect',
    label: '断开连接',
    type: 'danger',
    icon: Delete,
    disabled: (row) => !row.is_online
  }
]

// 统计信息
const stats = computed(() => {
  const total = clientList.value.length
  const online = clientList.value.filter(c => c.is_online).length
  const providers = clientList.value.filter(c => c.role === 'provider').length
  const receivers = clientList.value.filter(c => c.role === 'receiver').length
  const unassigned = clientList.value.filter(c => c.role === 'none').length
  
  return {
    total,
    online,
    providers,
    receivers,
    unassigned
  }
})

// 数据获取
const fetchClientList = async () => {
  try {
    loading.value = true
    
    const params = {
      page: pagination.page,
      pageSize: pagination.pageSize,
      ...searchParams
    }
    
    const response = await getClientList(params)
    
    if (response.code === 0) {
      clientList.value = response.data.list || []
      pagination.total = response.data.total || 0
    } else {
      throw new Error(response.msg || '获取客户端列表失败')
    }
  } catch (error) {
    console.warn('API调用失败，使用模拟数据:', error.message)
    ElMessage.warning('连接后端失败，正在使用模拟数据进行演示')
    
    // 使用模拟数据
    const mockData = generateMockClients(pagination.pageSize)
    clientList.value = mockData
    pagination.total = 150 // 模拟总数
  } finally {
    loading.value = false
  }
}

// 使用实时数据更新
const { refresh: realTimeRefresh } = useRealTimeData(fetchClientList, 30000)

// 事件处理
const handleRefresh = async () => {
  await fetchClientList()
  ElMessage.success('列表已刷新')
}

const handleSearch = (params) => {
  Object.assign(searchParams, params)
  pagination.page = 1
  fetchClientList()
}

const handleReset = () => {
  Object.keys(searchParams).forEach(key => {
    searchParams[key] = ''
  })
  pagination.page = 1
  fetchClientList()
}

const handleSelectionChange = (selection) => {
  selectedClients.value = selection
}

const handlePageChange = ({ page }) => {
  pagination.page = page
  fetchClientList()
}

const handleSizeChange = ({ pageSize }) => {
  pagination.pageSize = pageSize
  pagination.page = 1
  fetchClientList()
}

const handleAction = ({ action, row }) => {
  switch (action) {
    case 'view':
      showClientDetail(row)
      break
    case 'disconnect':
      showDisconnectConfirm(row)
      break
  }
}

const showClientDetail = (client) => {
  detailDialog.data = client
  detailDialog.visible = true
}

const showDisconnectConfirm = (client) => {
  confirmDialogVisible.value = true
  confirmDialogTitle.value = '确认断开连接'
  confirmDialogMessage.value = `确定要强制断开客户端 "${client.display_name || client.client_id}" 的连接吗？`
  confirmDialogDescription.value = '此操作将立即关闭WebSocket连接，可能会影响正在进行的NFC会话。'
  confirmDialogType.value = 'warning'
  confirmDialogRequireInput.value = true
  confirmDialogInputValidation.value = '断开连接'
  confirmDialogAction.value = 'disconnect'
  confirmDialogData.value = client
}

const handleConfirm = async ({ inputValue }) => {
  if (confirmDialogAction.value === 'disconnect') {
    await executeDisconnect(confirmDialogData.value)
  }
}

const handleCancel = () => {
  confirmDialogVisible.value = false
  confirmDialogAction.value = null
  confirmDialogData.value = null
}

const executeDisconnect = async (client) => {
  try {
    confirmDialogLoading.value = true
    
    const response = await disconnectClient(client.client_id)
    
    if (response.code === 0) {
      ElMessage.success('客户端连接已断开')
      confirmDialogVisible.value = false
      await fetchClientList() // 刷新列表
    } else {
      throw new Error(response.msg || '断开连接失败')
    }
  } catch (error) {
    ElMessage.error('断开连接失败: ' + error.message)
  } finally {
    confirmDialogLoading.value = false
  }
}

// 生命周期
onMounted(() => {
  fetchClientList()
})
</script>

<style scoped lang="scss">
.client-management {
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
  
  .table-section {
    background: white;
    border-radius: 8px;
    overflow: hidden;
    box-shadow: 0 2px 8px rgba(0, 0, 0, 0.1);
  }
}

// 响应式设计
@media (max-width: 768px) {
  .client-management {
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
  }
}
</style> 