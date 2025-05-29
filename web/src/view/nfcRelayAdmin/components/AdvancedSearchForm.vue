<!--
  高级搜索表单组件
  支持复杂的查询条件和过滤器
-->
<template>
  <el-card shadow="never" class="advanced-search-form">
    <template #header>
      <div class="flex justify-between items-center">
        <div class="flex items-center">
          <el-icon class="mr-2">
            <Search />
          </el-icon>
          <span class="font-semibold">高级搜索</span>
        </div>
        <div class="flex items-center space-x-2">
          <el-button 
            size="small" 
            @click="toggleExpanded" 
            :icon="isExpanded ? ArrowUp : ArrowDown"
          >
            {{ isExpanded ? '收起' : '展开' }}
          </el-button>
          <el-button size="small" @click="resetFilters">
            <el-icon><Refresh /></el-icon>
            重置
          </el-button>
          <el-button size="small" @click="saveSearchTemplate">
            <el-icon><Star /></el-icon>
            保存模板
          </el-button>
        </div>
      </div>
    </template>

    <div class="search-content">
      <!-- 基础搜索区域 -->
      <div class="basic-search mb-4">
        <el-row :gutter="16">
          <el-col :span="8">
            <el-input
              v-model="searchParams.keyword"
              placeholder="关键词搜索..."
              clearable
              @keyup.enter="handleSearch"
            >
              <template #prefix>
                <el-icon><Search /></el-icon>
              </template>
            </el-input>
          </el-col>
          <el-col :span="6">
            <el-select
              v-model="searchParams.timeRange"
              placeholder="时间范围"
              clearable
              @change="handleTimeRangeChange"
            >
              <el-option label="最近1小时" value="1h" />
              <el-option label="最近1天" value="1d" />
              <el-option label="最近7天" value="7d" />
              <el-option label="最近30天" value="30d" />
              <el-option label="自定义" value="custom" />
            </el-select>
          </el-col>
          <el-col :span="6">
            <el-select
              v-model="searchParams.sortBy"
              placeholder="排序字段"
              clearable
            >
              <el-option 
                v-for="field in sortFields" 
                :key="field.value" 
                :label="field.label" 
                :value="field.value" 
              />
            </el-select>
          </el-col>
          <el-col :span="4">
            <el-select
              v-model="searchParams.sortOrder"
              placeholder="排序方向"
            >
              <el-option label="升序" value="asc" />
              <el-option label="降序" value="desc" />
            </el-select>
          </el-col>
        </el-row>
      </div>

      <!-- 自定义时间范围 -->
      <div v-if="searchParams.timeRange === 'custom'" class="custom-time-range mb-4">
        <el-row :gutter="16">
          <el-col :span="12">
            <el-date-picker
              v-model="customTimeRange"
              type="datetimerange"
              range-separator="至"
              start-placeholder="开始时间"
              end-placeholder="结束时间"
              format="YYYY-MM-DD HH:mm:ss"
              value-format="YYYY-MM-DDTHH:mm:ss.SSSZ"
              @change="handleCustomTimeRangeChange"
            />
          </el-col>
        </el-row>
      </div>

      <!-- 展开的高级搜索区域 -->
      <div v-show="isExpanded" class="advanced-filters">
        <!-- 连接管理专用过滤器 -->
        <div v-if="searchType === 'clients'" class="filter-section">
          <h4 class="section-title">连接过滤器</h4>
          <el-row :gutter="16" class="mb-3">
            <el-col :span="6">
              <el-select
                v-model="searchParams.roles"
                multiple
                placeholder="角色"
                clearable
              >
                <el-option label="Provider" value="provider" />
                <el-option label="Receiver" value="receiver" />
                <el-option label="None" value="none" />
              </el-select>
            </el-col>
            <el-col :span="6">
              <el-select
                v-model="searchParams.connectionStatus"
                multiple
                placeholder="连接状态"
                clearable
              >
                <el-option label="在线" value="online" />
                <el-option label="离线" value="offline" />
                <el-option label="空闲" value="idle" />
              </el-select>
            </el-col>
            <el-col :span="6">
              <el-select
                v-model="searchParams.deviceTypes"
                multiple
                placeholder="设备类型"
                clearable
              >
                <el-option label="移动设备" value="mobile" />
                <el-option label="POS终端" value="pos" />
                <el-option label="读卡器" value="reader" />
                <el-option label="其他" value="other" />
              </el-select>
            </el-col>
            <el-col :span="6">
              <el-input
                v-model="searchParams.ipRanges"
                placeholder="IP地址范围"
                clearable
              />
            </el-col>
          </el-row>
          
          <el-row :gutter="16" class="mb-3">
            <el-col :span="8">
              <div class="range-input">
                <span class="range-label">在线时长(分钟):</span>
                <el-input-number
                  v-model="searchParams.onlineDurationMin"
                  placeholder="最小值"
                  :min="0"
                  size="small"
                />
                <span class="mx-2">-</span>
                <el-input-number
                  v-model="searchParams.onlineDurationMax"
                  placeholder="最大值"
                  :min="0"
                  size="small"
                />
              </div>
            </el-col>
            <el-col :span="8">
              <div class="range-input">
                <span class="range-label">会话数量:</span>
                <el-input-number
                  v-model="searchParams.sessionCountMin"
                  placeholder="最小值"
                  :min="0"
                  size="small"
                />
                <span class="mx-2">-</span>
                <el-input-number
                  v-model="searchParams.sessionCountMax"
                  placeholder="最大值"
                  :min="0"
                  size="small"
                />
              </div>
            </el-col>
            <el-col :span="8">
              <div class="range-input">
                <span class="range-label">数据传输(MB):</span>
                <el-input-number
                  v-model="searchParams.dataTransferredMin"
                  placeholder="最小值"
                  :min="0"
                  size="small"
                />
                <span class="mx-2">-</span>
                <el-input-number
                  v-model="searchParams.dataTransferredMax"
                  placeholder="最大值"
                  :min="0"
                  size="small"
                />
              </div>
            </el-col>
          </el-row>

          <!-- 地理位置过滤 -->
          <el-row :gutter="16" class="mb-3">
            <el-col :span="6">
              <el-select
                v-model="searchParams.geoCountry"
                placeholder="国家"
                clearable
                filterable
              >
                <el-option 
                  v-for="country in countries" 
                  :key="country.code" 
                  :label="country.name" 
                  :value="country.code" 
                />
              </el-select>
            </el-col>
            <el-col :span="6">
              <el-select
                v-model="searchParams.geoRegion"
                placeholder="地区"
                clearable
                filterable
              >
                <el-option 
                  v-for="region in regions" 
                  :key="region.code" 
                  :label="region.name" 
                  :value="region.code" 
                />
              </el-select>
            </el-col>
            <el-col :span="6">
              <el-select
                v-model="searchParams.geoCity"
                placeholder="城市"
                clearable
                filterable
              >
                <el-option 
                  v-for="city in cities" 
                  :key="city.code" 
                  :label="city.name" 
                  :value="city.code" 
                />
              </el-select>
            </el-col>
            <el-col :span="6">
              <el-input
                v-model="searchParams.tags"
                placeholder="标签 (逗号分隔)"
                clearable
              />
            </el-col>
          </el-row>
        </div>

        <!-- 会话管理专用过滤器 -->
        <div v-if="searchType === 'sessions'" class="filter-section">
          <h4 class="section-title">会话过滤器</h4>
          <el-row :gutter="16" class="mb-3">
            <el-col :span="6">
              <el-select
                v-model="searchParams.sessionStates"
                multiple
                placeholder="会话状态"
                clearable
              >
                <el-option label="活跃" value="active" />
                <el-option label="暂停" value="paused" />
                <el-option label="已完成" value="completed" />
                <el-option label="失败" value="failed" />
              </el-select>
            </el-col>
            <el-col :span="6">
              <el-select
                v-model="searchParams.sessionTypes"
                multiple
                placeholder="会话类型"
                clearable
              >
                <el-option label="标准中继" value="standard" />
                <el-option label="加密中继" value="encrypted" />
                <el-option label="批量操作" value="batch" />
              </el-select>
            </el-col>
            <el-col :span="6">
              <el-checkbox v-model="searchParams.hasRecording">
                包含录制
              </el-checkbox>
            </el-col>
            <el-col :span="6">
              <el-input-number
                v-model="searchParams.errorThreshold"
                placeholder="错误阈值(%)"
                :min="0"
                :max="100"
                size="small"
              />
            </el-col>
          </el-row>
          
          <el-row :gutter="16" class="mb-3">
            <el-col :span="8">
              <div class="range-input">
                <span class="range-label">持续时间(分钟):</span>
                <el-input-number
                  v-model="searchParams.durationMin"
                  placeholder="最小值"
                  :min="0"
                  size="small"
                />
                <span class="mx-2">-</span>
                <el-input-number
                  v-model="searchParams.durationMax"
                  placeholder="最大值"
                  :min="0"
                  size="small"
                />
              </div>
            </el-col>
            <el-col :span="8">
              <div class="range-input">
                <span class="range-label">APDU数量:</span>
                <el-input-number
                  v-model="searchParams.apduCountMin"
                  placeholder="最小值"
                  :min="0"
                  size="small"
                />
                <span class="mx-2">-</span>
                <el-input-number
                  v-model="searchParams.apduCountMax"
                  placeholder="最大值"
                  :min="0"
                  size="small"
                />
              </div>
            </el-col>
            <el-col :span="8">
              <div class="range-input">
                <span class="range-label">参与者数量:</span>
                <el-input-number
                  v-model="searchParams.participantsMin"
                  placeholder="最小值"
                  :min="2"
                  :max="10"
                  size="small"
                />
                <span class="mx-2">-</span>
                <el-input-number
                  v-model="searchParams.participantsMax"
                  placeholder="最大值"
                  :min="2"
                  :max="10"
                  size="small"
                />
              </div>
            </el-col>
          </el-row>

          <!-- 性能指标过滤 -->
          <el-row :gutter="16" class="mb-3">
            <el-col :span="8">
              <div class="range-input">
                <span class="range-label">最小吞吐量(KB/s):</span>
                <el-input-number
                  v-model="searchParams.minThroughput"
                  placeholder="最小吞吐量"
                  :min="0"
                  size="small"
                />
              </div>
            </el-col>
            <el-col :span="8">
              <div class="range-input">
                <span class="range-label">最大延迟(ms):</span>
                <el-input-number
                  v-model="searchParams.maxLatency"
                  placeholder="最大延迟"
                  :min="0"
                  size="small"
                />
              </div>
            </el-col>
            <el-col :span="8">
              <div class="range-input">
                <span class="range-label">录制大小(MB):</span>
                <el-input-number
                  v-model="searchParams.recordingSizeMin"
                  placeholder="最小值"
                  :min="0"
                  size="small"
                />
                <span class="mx-2">-</span>
                <el-input-number
                  v-model="searchParams.recordingSizeMax"
                  placeholder="最大值"
                  :min="0"
                  size="small"
                />
              </div>
            </el-col>
          </el-row>
        </div>

        <!-- 审计日志专用过滤器 -->
        <div v-if="searchType === 'auditLogs'" class="filter-section">
          <h4 class="section-title">日志过滤器</h4>
          <el-row :gutter="16" class="mb-3">
            <el-col :span="6">
              <el-select
                v-model="searchParams.eventTypes"
                multiple
                placeholder="事件类型"
                clearable
              >
                <el-option label="连接事件" value="connection" />
                <el-option label="会话事件" value="session" />
                <el-option label="APDU事件" value="apdu" />
                <el-option label="认证事件" value="auth" />
                <el-option label="系统事件" value="system" />
                <el-option label="错误事件" value="error" />
              </el-select>
            </el-col>
            <el-col :span="6">
              <el-select
                v-model="searchParams.severityLevels"
                multiple
                placeholder="严重级别"
                clearable
              >
                <el-option label="调试" value="debug" />
                <el-option label="信息" value="info" />
                <el-option label="警告" value="warn" />
                <el-option label="错误" value="error" />
                <el-option label="严重" value="critical" />
              </el-select>
            </el-col>
            <el-col :span="6">
              <el-select
                v-model="searchParams.actionTypes"
                multiple
                placeholder="操作类型"
                clearable
              >
                <el-option label="创建" value="create" />
                <el-option label="读取" value="read" />
                <el-option label="更新" value="update" />
                <el-option label="删除" value="delete" />
                <el-option label="执行" value="execute" />
              </el-select>
            </el-col>
            <el-col :span="6">
              <el-input
                v-model="searchParams.sourceIps"
                placeholder="源IP地址"
                clearable
              />
            </el-col>
          </el-row>
          
          <el-row :gutter="16" class="mb-3">
            <el-col :span="6">
              <el-input
                v-model="searchParams.messagePattern"
                placeholder="消息模式 (正则表达式)"
                clearable
              />
            </el-col>
            <el-col :span="6">
              <el-input
                v-model="searchParams.correlationId"
                placeholder="关联ID"
                clearable
              />
            </el-col>
            <el-col :span="6">
              <el-input
                v-model="searchParams.requestId"
                placeholder="请求ID"
                clearable
              />
            </el-col>
            <el-col :span="6">
              <el-checkbox v-model="searchParams.hasAttachments">
                包含附件
              </el-checkbox>
            </el-col>
          </el-row>

          <!-- 结果代码过滤 -->
          <el-row :gutter="16" class="mb-3">
            <el-col :span="12">
              <el-select
                v-model="searchParams.resultCodes"
                multiple
                placeholder="结果代码"
                clearable
                filterable
                allow-create
              >
                <el-option label="200 - 成功" :value="200" />
                <el-option label="400 - 客户端错误" :value="400" />
                <el-option label="401 - 未授权" :value="401" />
                <el-option label="403 - 禁止访问" :value="403" />
                <el-option label="404 - 未找到" :value="404" />
                <el-option label="500 - 服务器错误" :value="500" />
              </el-select>
            </el-col>
            <el-col :span="12">
              <el-input
                v-model="searchParams.userAgents"
                placeholder="用户代理"
                clearable
              />
            </el-col>
          </el-row>
        </div>

        <!-- 搜索模板选择器 -->
        <div class="template-section mb-4">
          <h4 class="section-title">搜索模板</h4>
          <el-row :gutter="16">
            <el-col :span="8">
              <el-select
                v-model="selectedTemplate"
                placeholder="选择搜索模板"
                clearable
                @change="loadSearchTemplate"
              >
                <el-option 
                  v-for="template in searchTemplates" 
                  :key="template.id" 
                  :label="template.name" 
                  :value="template.id" 
                />
              </el-select>
            </el-col>
            <el-col :span="16">
              <el-button-group>
                <el-button size="small" @click="saveCurrentAsTemplate">
                  <el-icon><Plus /></el-icon>
                  保存当前
                </el-button>
                <el-button 
                  size="small" 
                  @click="editTemplate" 
                  :disabled="!selectedTemplate"
                >
                  <el-icon><Edit /></el-icon>
                  编辑
                </el-button>
                <el-button 
                  size="small" 
                  type="danger" 
                  @click="deleteTemplate" 
                  :disabled="!selectedTemplate"
                >
                  <el-icon><Delete /></el-icon>
                  删除
                </el-button>
              </el-button-group>
            </el-col>
          </el-row>
        </div>

        <!-- 操作按钮 -->
        <div class="action-buttons">
          <el-button type="primary" @click="handleSearch" :loading="loading">
            <el-icon><Search /></el-icon>
            搜索
          </el-button>
          <el-button @click="resetFilters">
            <el-icon><Refresh /></el-icon>
            重置
          </el-button>
          <el-button @click="exportSearchResults">
            <el-icon><Download /></el-icon>
            导出结果
          </el-button>
        </div>
      </div>
    </div>
  </el-card>
</template>

<script setup>
import { ref, reactive, computed, watch, onMounted } from 'vue'
import { ElMessage, ElMessageBox } from 'element-plus'
import {
  Search,
  ArrowUp,
  ArrowDown,
  Refresh,
  Star,
  Plus,
  Edit,
  Delete,
  Download
} from '@element-plus/icons-vue'

const props = defineProps({
  searchType: {
    type: String,
    required: true,
    validator: value => ['clients', 'sessions', 'auditLogs'].includes(value)
  },
  loading: {
    type: Boolean,
    default: false
  }
})

const emit = defineEmits(['search', 'reset', 'export'])

// 响应式数据
const isExpanded = ref(false)
const selectedTemplate = ref('')
const customTimeRange = ref([])

// 搜索参数
const searchParams = reactive({
  // 通用参数
  keyword: '',
  timeRange: '',
  sortBy: '',
  sortOrder: 'desc',
  startTime: '',
  endTime: '',
  
  // 连接管理参数
  roles: [],
  connectionStatus: [],
  deviceTypes: [],
  ipRanges: '',
  onlineDurationMin: null,
  onlineDurationMax: null,
  sessionCountMin: null,
  sessionCountMax: null,
  dataTransferredMin: null,
  dataTransferredMax: null,
  geoCountry: '',
  geoRegion: '',
  geoCity: '',
  tags: '',
  
  // 会话管理参数
  sessionStates: [],
  sessionTypes: [],
  hasRecording: false,
  errorThreshold: null,
  durationMin: null,
  durationMax: null,
  apduCountMin: null,
  apduCountMax: null,
  participantsMin: null,
  participantsMax: null,
  minThroughput: null,
  maxLatency: null,
  recordingSizeMin: null,
  recordingSizeMax: null,
  
  // 审计日志参数
  eventTypes: [],
  severityLevels: [],
  actionTypes: [],
  sourceIps: '',
  messagePattern: '',
  correlationId: '',
  requestId: '',
  hasAttachments: false,
  resultCodes: [],
  userAgents: ''
})

// 计算属性
const sortFields = computed(() => {
  const commonFields = [
    { label: '创建时间', value: 'created_at' },
    { label: '更新时间', value: 'updated_at' }
  ]
  
  switch (props.searchType) {
    case 'clients':
      return [
        ...commonFields,
        { label: '连接时间', value: 'connected_at' },
        { label: '最后活动', value: 'last_activity_at' },
        { label: '客户端ID', value: 'client_id' },
        { label: '用户ID', value: 'user_id' }
      ]
    case 'sessions':
      return [
        ...commonFields,
        { label: '持续时间', value: 'duration' },
        { label: 'APDU数量', value: 'apdu_count' },
        { label: '会话ID', value: 'session_id' }
      ]
    case 'auditLogs':
      return [
        { label: '时间戳', value: 'timestamp' },
        { label: '事件类型', value: 'event_type' },
        { label: '严重级别', value: 'severity' }
      ]
    default:
      return commonFields
  }
})

// 模拟数据
const countries = ref([
  { code: 'CN', name: '中国' },
  { code: 'US', name: '美国' },
  { code: 'JP', name: '日本' },
  { code: 'KR', name: '韩国' }
])

const regions = ref([
  { code: 'BJ', name: '北京' },
  { code: 'SH', name: '上海' },
  { code: 'GD', name: '广东' },
  { code: 'JS', name: '江苏' }
])

const cities = ref([
  { code: 'BJ', name: '北京' },
  { code: 'SH', name: '上海' },
  { code: 'GZ', name: '广州' },
  { code: 'SZ', name: '深圳' }
])

const searchTemplates = ref([
  { id: 1, name: '今日活动连接', type: 'clients' },
  { id: 2, name: '长时间会话', type: 'sessions' },
  { id: 3, name: '错误日志', type: 'auditLogs' }
])

// 方法
const toggleExpanded = () => {
  isExpanded.value = !isExpanded.value
}

const handleTimeRangeChange = (value) => {
  if (value !== 'custom') {
    customTimeRange.value = []
    // 根据预设范围设置时间
    const now = new Date()
    let startTime = new Date()
    
    switch (value) {
      case '1h':
        startTime.setHours(now.getHours() - 1)
        break
      case '1d':
        startTime.setDate(now.getDate() - 1)
        break
      case '7d':
        startTime.setDate(now.getDate() - 7)
        break
      case '30d':
        startTime.setDate(now.getDate() - 30)
        break
      default:
        return
    }
    
    searchParams.startTime = startTime.toISOString()
    searchParams.endTime = now.toISOString()
  }
}

const handleCustomTimeRangeChange = (value) => {
  if (value && value.length === 2) {
    searchParams.startTime = value[0]
    searchParams.endTime = value[1]
  } else {
    searchParams.startTime = ''
    searchParams.endTime = ''
  }
}

const handleSearch = () => {
  // 构建搜索参数
  const params = {}
  
  // 复制非空参数
  Object.keys(searchParams).forEach(key => {
    const value = searchParams[key]
    if (value !== '' && value !== null && value !== undefined) {
      if (Array.isArray(value) && value.length > 0) {
        params[key] = value
      } else if (!Array.isArray(value)) {
        params[key] = value
      }
    }
  })
  
  emit('search', params)
}

const resetFilters = () => {
  Object.keys(searchParams).forEach(key => {
    if (Array.isArray(searchParams[key])) {
      searchParams[key] = []
    } else {
      searchParams[key] = ''
    }
  })
  customTimeRange.value = []
  selectedTemplate.value = ''
  emit('reset')
}

const saveSearchTemplate = () => {
  ElMessageBox.prompt('请输入模板名称', '保存搜索模板', {
    confirmButtonText: '保存',
    cancelButtonText: '取消'
  }).then(({ value }) => {
    // 保存模板逻辑
    ElMessage.success('搜索模板已保存')
  })
}

const loadSearchTemplate = (templateId) => {
  // 加载模板逻辑
  ElMessage.info('已加载搜索模板')
}

const saveCurrentAsTemplate = () => {
  saveSearchTemplate()
}

const editTemplate = () => {
  ElMessage.info('编辑模板功能开发中')
}

const deleteTemplate = () => {
  ElMessageBox.confirm('确定要删除此搜索模板吗？', '删除确认', {
    type: 'warning'
  }).then(() => {
    ElMessage.success('模板已删除')
    selectedTemplate.value = ''
  })
}

const exportSearchResults = () => {
  emit('export', searchParams)
}

// 监听器
watch(() => props.searchType, () => {
  resetFilters()
})

// 生命周期
onMounted(() => {
  // 初始化默认排序
  searchParams.sortOrder = 'desc'
})
</script>

<style scoped lang="scss">
.advanced-search-form {
  margin-bottom: 16px;
  
  .search-content {
    .basic-search {
      .el-input,
      .el-select {
        width: 100%;
      }
    }
    
    .custom-time-range {
      .el-date-picker {
        width: 100%;
      }
    }
    
    .advanced-filters {
      .filter-section {
        margin-bottom: 24px;
        padding: 16px;
        background-color: #f9f9f9;
        border-radius: 8px;
        border: 1px solid #e4e7ed;
        
        .section-title {
          margin: 0 0 16px 0;
          font-size: 14px;
          font-weight: 600;
          color: #303133;
          display: flex;
          align-items: center;
          
          &:before {
            content: '';
            width: 3px;
            height: 14px;
            background-color: #409eff;
            margin-right: 8px;
          }
        }
        
        .el-input,
        .el-select,
        .el-input-number {
          width: 100%;
        }
        
        .range-input {
          display: flex;
          align-items: center;
          gap: 8px;
          
          .range-label {
            font-size: 12px;
            color: #606266;
            white-space: nowrap;
            min-width: 80px;
          }
          
          .el-input-number {
            flex: 1;
          }
        }
      }
      
      .template-section {
        margin-bottom: 24px;
        
        .section-title {
          margin: 0 0 16px 0;
          font-size: 14px;
          font-weight: 600;
          color: #303133;
        }
      }
      
      .action-buttons {
        text-align: center;
        padding-top: 16px;
        border-top: 1px solid #e4e7ed;
        
        .el-button {
          margin: 0 8px;
        }
      }
    }
  }
}

.font-semibold {
  font-weight: 600;
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

.mb-3 {
  margin-bottom: 0.75rem;
}

.mb-4 {
  margin-bottom: 1rem;
}

.mx-2 {
  margin-left: 0.5rem;
  margin-right: 0.5rem;
}
</style> 