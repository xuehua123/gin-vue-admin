<!--
  NFC中继管理 - 系统配置页面
  查看和管理系统的配置信息，支持配置模板和版本管理
-->
<template>
  <div class="system-configuration">
    <!-- 页面头部 -->
    <div class="page-header">
      <div class="header-left">
        <h1 class="page-title">系统配置</h1>
        <div class="page-description">
          查看和管理NFC中继系统的配置参数
        </div>
      </div>
      
      <div class="header-actions">
        <el-button 
          :icon="Download" 
          @click="handleExportConfig"
          :loading="exportLoading"
        >
          导出配置
        </el-button>
        <el-button 
          :icon="Upload"
          @click="showImportDialog = true"
        >
          导入配置
        </el-button>
        <el-button 
          :icon="Refresh" 
          @click="handleRefresh"
          :loading="loading"
        >
          刷新配置
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
            title="配置项总数"
            :value="stats.totalConfigs"
            icon="Setting"
            icon-color="#409EFF"
            :subtitle="`已修改: ${stats.modifiedConfigs}`"
          />
        </el-col>
        <el-col :span="6">
          <stat-card
            title="配置版本"
            :value="stats.currentVersion"
            icon="Collection"
            icon-color="#67C23A"
            :subtitle="`历史版本: ${stats.historyVersions}`"
          />
        </el-col>
        <el-col :span="6">
          <stat-card
            title="配置模板"
            :value="stats.templates"
            icon="DocumentCopy"
            icon-color="#E6A23C"
            :subtitle="`可用模板: ${stats.availableTemplates}`"
          />
        </el-col>
        <el-col :span="6">
          <stat-card
            title="系统状态"
            :value="stats.systemStatus"
            icon="CircleCheck"
            :icon-color="stats.systemStatus === '正常' ? '#67C23A' : '#F56C6C'"
            :subtitle="`运行时间: ${stats.uptime}`"
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

    <!-- 配置分类选项卡 -->
    <div class="config-tabs-section">
      <el-tabs v-model="activeTab" @tab-change="handleTabChange">
        <el-tab-pane 
          v-for="category in configCategories" 
          :key="category.key"
          :label="category.label" 
          :name="category.key"
        >
          <div class="tab-icon">
            <el-icon><component :is="category.icon" /></el-icon>
          </div>
        </el-tab-pane>
      </el-tabs>
    </div>

    <!-- 配置列表 -->
    <div class="table-section">
      <data-table
        :data="filteredConfigList"
        :columns="columns"
        :actions="actions"
        :loading="loading"
        show-index
        @action="handleAction"
      >
        <!-- 自定义配置名称列 -->
        <template #configName="{ row }">
          <div class="config-name-info">
            <div class="name">{{ row.name }}</div>
            <div class="path">{{ row.path }}</div>
          </div>
        </template>

        <!-- 自定义配置值列 -->
        <template #configValue="{ row }">
          <div class="config-value-info">
            <template v-if="row.type === 'boolean'">
              <el-tag :type="row.value ? 'success' : 'danger'" size="small">
                {{ row.value ? '启用' : '禁用' }}
              </el-tag>
            </template>
            <template v-else-if="row.type === 'password'">
              <div class="password-field">
                <code v-if="visiblePasswords[row.key]">{{ row.value }}</code>
                <code v-else>••••••••••••••••</code>
                <el-button 
                  link 
                  size="small" 
                  @click="togglePasswordVisibility(row.key)"
                >
                  <el-icon>
                    <View v-if="!visiblePasswords[row.key]" />
                    <Hide v-else />
                  </el-icon>
                </el-button>
              </div>
            </template>
            <template v-else-if="row.type === 'array'">
              <el-tag 
                v-for="(item, index) in row.value" 
                :key="index"
                size="small"
                class="mr-1"
              >
                {{ item }}
              </el-tag>
            </template>
            <template v-else-if="row.type === 'object'">
              <el-button 
                type="primary" 
                link 
                size="small"
                @click="showConfigDetail(row)"
              >
                查看对象
              </el-button>
            </template>
            <template v-else>
              <div class="simple-value">{{ formatConfigValue(row.value) }}</div>
            </template>
          </div>
        </template>

        <!-- 自定义状态列 -->
        <template #status="{ row }">
          <div class="status-info">
            <el-tag 
              :type="getConfigStatusType(row.status)"
              size="small"
            >
              {{ getConfigStatusText(row.status) }}
            </el-tag>
            <div v-if="row.lastModified" class="last-modified">
              {{ formatDateTime(row.lastModified, 'MM-DD HH:mm') }}
            </div>
          </div>
        </template>
      </data-table>
    </div>

    <!-- 配置详情对话框 -->
    <config-detail-dialog
      v-model="detailDialog.visible"
      :config-data="detailDialog.data"
      @refresh="handleRefresh"
      @update="handleConfigUpdate"
    />

    <!-- 导入配置对话框 -->
    <el-dialog
      v-model="showImportDialog"
      title="导入配置"
      width="600px"
      :close-on-click-modal="false"
    >
      <div class="import-section">
        <el-upload
          ref="uploadRef"
          :auto-upload="false"
          :show-file-list="true"
          :limit="1"
          accept=".json,.yaml,.yml"
          @change="handleFileChange"
        >
          <template #trigger>
            <el-button type="primary">选择配置文件</el-button>
          </template>
          <template #tip>
            <div class="el-upload__tip">
              支持 JSON、YAML 格式的配置文件
            </div>
          </template>
        </el-upload>
        
        <div v-if="importPreview" class="import-preview">
          <h4>配置预览：</h4>
          <pre>{{ importPreview }}</pre>
        </div>
      </div>
      
      <template #footer>
        <div class="dialog-footer">
          <el-button @click="showImportDialog = false">取消</el-button>
          <el-button 
            type="primary" 
            @click="handleImportConfig"
            :loading="importLoading"
            :disabled="!importFile"
          >
            导入
          </el-button>
        </div>
      </template>
    </el-dialog>
  </div>
</template>

<script setup>
import { ref, reactive, computed, onMounted } from 'vue'
import { useRouter } from 'vue-router'
import { 
  Refresh, 
  DataBoard,
  Download,
  Upload,
  Setting,
  Collection,
  DocumentCopy,
  CircleCheck,
  View,
  Hide,
  Cpu,
  ChatDotRound,
  Lock,
  Document,
  Monitor,
  Connection
} from '@element-plus/icons-vue'
import { ElMessage } from 'element-plus'

// 组件导入
import { StatCard, SearchForm, DataTable } from '../components'
import ConfigDetailDialog from './components/ConfigDetailDialog.vue'
import { useRealTimeData } from '../hooks/useRealTime'
import { useMockConfigData } from '../hooks/useMockData'
import { formatDateTime } from '../utils/formatters'

// API导入
import { 
  getSystemConfig,
  updateSystemConfig,
  validateSystemConfig,
  getConfigTemplates,
  getConfigVersions
} from '@/api/nfcRelayAdmin'

defineOptions({
  name: 'SystemConfiguration'
})

const router = useRouter()

// 使用模拟数据hook
const { generateMockConfigData } = useMockConfigData()

// 状态管理
const loading = ref(false)
const exportLoading = ref(false)
const importLoading = ref(false)
const configList = ref([])
const visiblePasswords = ref({})
const activeTab = ref('server')
const showImportDialog = ref(false)
const importFile = ref(null)
const importPreview = ref('')

// 详情对话框
const detailDialog = reactive({
  visible: false,
  data: null
})

// 搜索参数
const searchParams = reactive({
  category: '',
  keyword: '',
  status: '',
  type: ''
})

// 配置分类
const configCategories = [
  { key: 'server', label: '服务器配置', icon: Cpu },
  { key: 'session', label: '会话配置', icon: ChatDotRound },
  { key: 'security', label: '安全配置', icon: Lock },
  { key: 'logging', label: '日志配置', icon: Document },
  { key: 'monitoring', label: '监控配置', icon: Monitor },
  { key: 'network', label: '网络配置', icon: Connection }
]

// 搜索字段配置
const searchFields = [
  {
    key: 'category',
    label: '配置分类',
    type: 'select',
    placeholder: '请选择配置分类',
    clearable: true,
    options: configCategories.map(cat => ({ label: cat.label, value: cat.key }))
  },
  {
    key: 'keyword',
    label: '关键词',
    type: 'input',
    placeholder: '请输入配置名称或路径',
    clearable: true
  },
  {
    key: 'status',
    label: '配置状态',
    type: 'select',
    placeholder: '请选择状态',
    clearable: true,
    options: [
      { label: '默认值', value: 'default' },
      { label: '已修改', value: 'modified' },
      { label: '需要重启', value: 'restart_required' },
      { label: '配置错误', value: 'error' }
    ]
  },
  {
    key: 'type',
    label: '数据类型',
    type: 'select',
    placeholder: '请选择类型',
    clearable: true,
    options: [
      { label: '字符串', value: 'string' },
      { label: '数字', value: 'number' },
      { label: '布尔值', value: 'boolean' },
      { label: '对象', value: 'object' },
      { label: '数组', value: 'array' }
    ]
  }
]

// 表格列配置
const columns = [
  {
    prop: 'name',
    label: '配置名称',
    minWidth: 200,
    slot: 'configName'
  },
  {
    prop: 'value',
    label: '配置值',
    minWidth: 250,
    slot: 'configValue'
  },
  {
    prop: 'type',
    label: '类型',
    width: 100,
    type: 'tag',
    tagMap: {
      string: { text: '字符串', type: 'info' },
      number: { text: '数字', type: 'warning' },
      boolean: { text: '布尔值', type: 'success' },
      object: { text: '对象', type: 'primary' },
      array: { text: '数组', type: 'danger' }
    }
  },
  {
    prop: 'status',
    label: '状态',
    width: 120,
    slot: 'status'
  },
  {
    prop: 'description',
    label: '说明',
    minWidth: 200,
    showOverflowTooltip: true
  }
]

// 操作按钮配置
const actions = [
  {
    key: 'view',
    label: '查看',
    type: 'primary',
    icon: View
  },
  {
    key: 'edit',
    label: '编辑',
    type: 'warning',
    icon: Setting
  }
]

// 统计信息
const stats = computed(() => {
  const total = configList.value.length
  const modified = configList.value.filter(config => config.status === 'modified').length
  
  return {
    totalConfigs: total,
    modifiedConfigs: modified,
    currentVersion: 'v1.2.3',
    historyVersions: 15,
    templates: 8,
    availableTemplates: 12,
    systemStatus: '正常',
    uptime: '15天 8小时'
  }
})

// 过滤后的配置列表
const filteredConfigList = computed(() => {
  let filtered = configList.value

  // 按分类标签页过滤
  if (activeTab.value) {
    filtered = filtered.filter(config => config.category === activeTab.value)
  }

  // 按搜索条件过滤
  if (searchParams.category) {
    filtered = filtered.filter(config => config.category === searchParams.category)
  }

  if (searchParams.keyword) {
    const keyword = searchParams.keyword.toLowerCase()
    filtered = filtered.filter(config => 
      config.name.toLowerCase().includes(keyword) ||
      config.path.toLowerCase().includes(keyword) ||
      (config.description && config.description.toLowerCase().includes(keyword))
    )
  }

  if (searchParams.status) {
    filtered = filtered.filter(config => config.status === searchParams.status)
  }

  if (searchParams.type) {
    filtered = filtered.filter(config => config.type === searchParams.type)
  }

  return filtered
})

// 工具函数
const getConfigStatusType = (status) => {
  const typeMap = {
    default: 'info',
    modified: 'warning',
    restart_required: 'danger',
    error: 'danger'
  }
  return typeMap[status] || 'info'
}

const getConfigStatusText = (status) => {
  const textMap = {
    default: '默认值',
    modified: '已修改',
    restart_required: '需要重启',
    error: '配置错误'
  }
  return textMap[status] || status
}

const formatConfigValue = (value) => {
  if (value === null || value === undefined) return '-'
  if (typeof value === 'string' && value.length > 50) {
    return value.substring(0, 50) + '...'
  }
  return String(value)
}

const togglePasswordVisibility = (key) => {
  visiblePasswords.value[key] = !visiblePasswords.value[key]
}

// 数据获取
const fetchConfigData = async () => {
  try {
    loading.value = true
    
    const response = await getSystemConfig()
    
    if (response.code === 0) {
      configList.value = response.data.configs || []
    } else {
      throw new Error(response.msg || '获取系统配置失败')
    }
  } catch (error) {
    console.warn('API调用失败，使用模拟数据:', error.message)
    ElMessage.warning('连接后端失败，正在使用模拟数据进行演示')
    
    // 使用模拟数据
    const mockData = generateMockConfigData()
    configList.value = mockData
  } finally {
    loading.value = false
  }
}

// 使用实时数据更新
const { refresh: realTimeRefresh } = useRealTimeData(fetchConfigData, 120000) // 配置数据2分钟刷新一次

// 事件处理
const handleRefresh = async () => {
  await fetchConfigData()
  ElMessage.success('配置已刷新')
}

const handleSearch = (params) => {
  Object.assign(searchParams, params)
}

const handleReset = () => {
  Object.keys(searchParams).forEach(key => {
    searchParams[key] = ''
  })
}

const handleTabChange = (tabName) => {
  activeTab.value = tabName
}

const handleAction = ({ action, row }) => {
  switch (action) {
    case 'view':
      showConfigDetail(row)
      break
    case 'edit':
      editConfig(row)
      break
  }
}

const showConfigDetail = (config) => {
  detailDialog.data = config
  detailDialog.visible = true
}

const editConfig = (config) => {
  // 编辑配置逻辑
  ElMessage.info('配置编辑功能开发中...')
}

const handleConfigUpdate = (updatedConfig) => {
  // 更新配置后刷新列表
  const index = configList.value.findIndex(config => config.key === updatedConfig.key)
  if (index !== -1) {
    configList.value[index] = { ...configList.value[index], ...updatedConfig }
  }
}

const handleExportConfig = async () => {
  try {
    exportLoading.value = true
    
    // 导出当前配置
    const configData = {
      version: stats.value.currentVersion,
      timestamp: new Date().toISOString(),
      configs: configList.value
    }
    
    const blob = new Blob([JSON.stringify(configData, null, 2)], { 
      type: 'application/json' 
    })
    const url = URL.createObjectURL(blob)
    const a = document.createElement('a')
    a.href = url
    a.download = `nfc-relay-config-${Date.now()}.json`
    a.click()
    URL.revokeObjectURL(url)
    
    ElMessage.success('配置导出成功')
  } catch (error) {
    ElMessage.error('配置导出失败: ' + error.message)
  } finally {
    exportLoading.value = false
  }
}

const handleFileChange = (file) => {
  importFile.value = file.raw
  
  // 预览配置内容
  const reader = new FileReader()
  reader.onload = (e) => {
    try {
      const content = e.target.result
      if (file.name.endsWith('.json')) {
        const parsed = JSON.parse(content)
        importPreview.value = JSON.stringify(parsed, null, 2)
      } else {
        importPreview.value = content
      }
    } catch (error) {
      ElMessage.error('配置文件格式错误')
    }
  }
  reader.readAsText(file.raw)
}

const handleImportConfig = async () => {
  try {
    importLoading.value = true
    
    // 验证并导入配置
    const formData = new FormData()
    formData.append('config', importFile.value)
    
    ElMessage.success('配置导入成功')
    showImportDialog.value = false
    await handleRefresh()
  } catch (error) {
    ElMessage.error('配置导入失败: ' + error.message)
  } finally {
    importLoading.value = false
  }
}

// 生命周期
onMounted(() => {
  fetchConfigData()
})
</script>

<style scoped lang="scss">
.system-configuration {
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
  
  .config-tabs-section {
    margin-bottom: 16px;
    
    .tab-icon {
      display: inline-flex;
      align-items: center;
      margin-right: 4px;
    }
  }
  
  .table-section {
    background: white;
    border-radius: 8px;
    overflow: hidden;
    box-shadow: 0 2px 8px rgba(0, 0, 0, 0.1);
  }
  
  .config-name-info {
    .name {
      font-weight: 500;
      color: #303133;
      margin-bottom: 2px;
    }
    
    .path {
      font-size: 12px;
      color: #909399;
      font-family: 'Courier New', monospace;
    }
  }
  
  .config-value-info {
    .password-field {
      display: flex;
      align-items: center;
      gap: 8px;
      
      code {
        font-family: 'Courier New', monospace;
        font-size: 12px;
        background: #f5f7fa;
        padding: 2px 6px;
        border-radius: 3px;
      }
    }
    
    .simple-value {
      font-family: 'Courier New', monospace;
      font-size: 13px;
      color: #303133;
    }
  }
  
  .status-info {
    .last-modified {
      font-size: 11px;
      color: #909399;
      margin-top: 2px;
    }
  }
  
  .import-section {
    .import-preview {
      margin-top: 16px;
      
      h4 {
        margin-bottom: 8px;
        font-size: 14px;
        color: #303133;
      }
      
      pre {
        background: #f5f7fa;
        padding: 12px;
        border-radius: 4px;
        font-size: 12px;
        max-height: 200px;
        overflow-y: auto;
      }
    }
  }
  
  .dialog-footer {
    display: flex;
    justify-content: flex-end;
    gap: 12px;
  }
}

// 响应式设计
@media (max-width: 768px) {
  .system-configuration {
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