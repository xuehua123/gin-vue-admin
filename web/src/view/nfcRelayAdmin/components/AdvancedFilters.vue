<!--
  高级筛选组件
  支持时间范围、指标类型、地理位置等多维度筛选
-->
<template>
  <div class="advanced-filters">
    <!-- 筛选标题栏 -->
    <div class="filter-header">
      <div class="header-left">
        <el-icon class="filter-icon">
          <Filter />
        </el-icon>
        <span class="filter-title">高级筛选</span>
        <el-tag 
          v-if="hasActiveFilters" 
          type="primary" 
          size="small"
          class="active-count"
        >
          {{ activeFiltersCount }}个筛选条件
        </el-tag>
      </div>
      
      <div class="header-right">
        <el-button 
          size="small" 
          @click="resetFilters"
          :disabled="!hasActiveFilters"
        >
          重置
        </el-button>
        <el-button 
          size="small" 
          type="primary" 
          @click="applyFilters"
        >
          应用筛选
        </el-button>
        <el-button 
          size="small" 
          text 
          @click="toggleExpanded"
        >
          <el-icon>
            <component :is="expanded ? ArrowUp : ArrowDown" />
          </el-icon>
        </el-button>
      </div>
    </div>

    <!-- 筛选内容区域 -->
    <transition name="el-collapse">
      <div v-show="expanded" class="filter-content">
        <el-row :gutter="16">
          <!-- 时间范围选择 -->
          <el-col :span="8">
            <div class="filter-group">
              <label class="filter-label">时间范围</label>
              <el-select 
                v-model="filters.timeRange" 
                class="filter-select"
                @change="onTimeRangeChange"
              >
                <el-option label="最近1小时" value="1h" />
                <el-option label="最近6小时" value="6h" />
                <el-option label="最近24小时" value="24h" />
                <el-option label="最近7天" value="7d" />
                <el-option label="最近30天" value="30d" />
                <el-option label="自定义" value="custom" />
              </el-select>
              
              <!-- 自定义时间范围 -->
              <div v-if="filters.timeRange === 'custom'" class="custom-time-range">
                <el-date-picker
                  v-model="customTimeRange"
                  type="datetimerange"
                  range-separator="至"
                  start-placeholder="开始时间"
                  end-placeholder="结束时间"
                  format="YYYY-MM-DD HH:mm:ss"
                  value-format="YYYY-MM-DDTHH:mm:ss.SSSZ"
                  class="custom-picker"
                  @change="onCustomTimeChange"
                />
              </div>
            </div>
          </el-col>

          <!-- 数据粒度 -->
          <el-col :span="6">
            <div class="filter-group">
              <label class="filter-label">数据粒度</label>
              <el-radio-group 
                v-model="filters.granularity" 
                size="small"
                class="granularity-group"
              >
                <el-radio-button label="minute">分钟</el-radio-button>
                <el-radio-button label="hour">小时</el-radio-button>
                <el-radio-button label="day">天</el-radio-button>
              </el-radio-group>
            </div>
          </el-col>

          <!-- 指标类型 -->
          <el-col :span="10">
            <div class="filter-group">
              <label class="filter-label">指标类型</label>
              <el-checkbox-group 
                v-model="filters.metrics" 
                class="metrics-group"
              >
                <el-checkbox label="connections">连接数</el-checkbox>
                <el-checkbox label="sessions">会话数</el-checkbox>
                <el-checkbox label="apdu">APDU转发</el-checkbox>
                <el-checkbox label="errors">错误率</el-checkbox>
                <el-checkbox label="performance">性能指标</el-checkbox>
              </el-checkbox-group>
            </div>
          </el-col>
        </el-row>

        <el-row :gutter="16" class="second-row">
          <!-- 设备类型筛选 -->
          <el-col :span="8">
            <div class="filter-group">
              <label class="filter-label">设备类型</label>
              <el-select 
                v-model="filters.deviceTypes" 
                multiple 
                placeholder="选择设备类型"
                class="filter-select"
              >
                <el-option label="Android" value="android" />
                <el-option label="iOS" value="ios" />
                <el-option label="Desktop" value="desktop" />
                <el-option label="Web" value="web" />
              </el-select>
            </div>
          </el-col>

          <!-- 地理位置筛选 -->
          <el-col :span="8">
            <div class="filter-group">
              <label class="filter-label">地理位置</label>
              <el-cascader
                v-model="filters.location"
                :options="locationOptions"
                :props="cascaderProps"
                placeholder="选择国家/地区"
                class="filter-select"
                clearable
              />
            </div>
          </el-col>

          <!-- 状态筛选 -->
          <el-col :span="8">
            <div class="filter-group">
              <label class="filter-label">状态筛选</label>
              <el-select 
                v-model="filters.status" 
                multiple 
                placeholder="选择状态"
                class="filter-select"
              >
                <el-option label="在线" value="online" />
                <el-option label="离线" value="offline" />
                <el-option label="错误" value="error" />
                <el-option label="活跃" value="active" />
              </el-select>
            </div>
          </el-col>
        </el-row>

        <!-- 高级选项 -->
        <div class="advanced-options">
          <el-divider content-position="left">
            <span class="divider-text">高级选项</span>
          </el-divider>
          
          <el-row :gutter="16">
            <el-col :span="8">
              <div class="filter-group">
                <label class="filter-label">聚合方式</label>
                <el-select v-model="filters.aggregation" class="filter-select">
                  <el-option label="平均值" value="avg" />
                  <el-option label="最大值" value="max" />
                  <el-option label="最小值" value="min" />
                  <el-option label="总和" value="sum" />
                  <el-option label="计数" value="count" />
                </el-select>
              </div>
            </el-col>

            <el-col :span="8">
              <div class="filter-group">
                <label class="filter-label">数据限制</label>
                <el-input-number
                  v-model="filters.limit"
                  :min="10"
                  :max="1000"
                  :step="10"
                  placeholder="数据条数"
                  class="filter-input"
                />
              </div>
            </el-col>

            <el-col :span="8">
              <div class="filter-group">
                <el-checkbox v-model="filters.includeAlerts" class="include-alerts">
                  包含告警信息
                </el-checkbox>
                <el-checkbox v-model="filters.realtime" class="realtime-mode">
                  实时模式
                </el-checkbox>
              </div>
            </el-col>
          </el-row>
        </div>

        <!-- 预设筛选方案 -->
        <div class="preset-filters">
          <el-divider content-position="left">
            <span class="divider-text">预设方案</span>
          </el-divider>
          
          <div class="preset-buttons">
            <el-button 
              v-for="preset in presetFilters" 
              :key="preset.name"
              size="small"
              @click="applyPreset(preset)"
              class="preset-btn"
            >
              {{ preset.label }}
            </el-button>
          </div>
        </div>
      </div>
    </transition>
  </div>
</template>

<script setup>
import { ref, computed, watch } from 'vue'
import { 
  Filter, 
  ArrowUp, 
  ArrowDown 
} from '@element-plus/icons-vue'

const props = defineProps({
  modelValue: {
    type: Object,
    default: () => ({})
  }
})

const emit = defineEmits(['update:modelValue', 'filter-change'])

// 响应式数据
const expanded = ref(true)
const customTimeRange = ref([])

// 筛选条件
const filters = ref({
  timeRange: '24h',
  startTime: null,
  endTime: null,
  granularity: 'hour',
  metrics: ['connections', 'sessions'],
  deviceTypes: [],
  location: [],
  status: [],
  aggregation: 'avg',
  limit: 100,
  includeAlerts: true,
  realtime: false
})

// 地理位置选项
const locationOptions = [
  {
    value: 'CN',
    label: '中国',
    children: [
      { value: 'Beijing', label: '北京' },
      { value: 'Shanghai', label: '上海' },
      { value: 'Guangzhou', label: '广州' }
    ]
  },
  {
    value: 'US',
    label: '美国',
    children: [
      { value: 'NewYork', label: '纽约' },
      { value: 'LosAngeles', label: '洛杉矶' },
      { value: 'Chicago', label: '芝加哥' }
    ]
  },
  {
    value: 'JP',
    label: '日本',
    children: [
      { value: 'Tokyo', label: '东京' },
      { value: 'Osaka', label: '大阪' }
    ]
  }
]

const cascaderProps = {
  expandTrigger: 'hover',
  checkStrictly: true
}

// 预设筛选方案
const presetFilters = [
  {
    name: 'performance_monitoring',
    label: '性能监控',
    config: {
      timeRange: '6h',
      granularity: 'minute',
      metrics: ['performance', 'errors'],
      includeAlerts: true
    }
  },
  {
    name: 'connection_analysis',
    label: '连接分析',
    config: {
      timeRange: '24h',
      granularity: 'hour',
      metrics: ['connections', 'sessions'],
      includeAlerts: false
    }
  },
  {
    name: 'error_tracking',
    label: '错误追踪',
    config: {
      timeRange: '7d',
      granularity: 'day',
      metrics: ['errors'],
      includeAlerts: true,
      status: ['error']
    }
  },
  {
    name: 'geographic_overview',
    label: '地理分布',
    config: {
      timeRange: '30d',
      granularity: 'day',
      metrics: ['connections'],
      includeAlerts: false
    }
  }
]

// 计算属性
const hasActiveFilters = computed(() => {
  return filters.value.deviceTypes.length > 0 ||
         filters.value.location.length > 0 ||
         filters.value.status.length > 0 ||
         filters.value.timeRange === 'custom' ||
         filters.value.metrics.length !== 2 // 默认有2个指标
})

const activeFiltersCount = computed(() => {
  let count = 0
  if (filters.value.deviceTypes.length > 0) count++
  if (filters.value.location.length > 0) count++
  if (filters.value.status.length > 0) count++
  if (filters.value.timeRange === 'custom') count++
  if (filters.value.metrics.length > 0) count++
  return count
})

// 方法
const toggleExpanded = () => {
  expanded.value = !expanded.value
}

const onTimeRangeChange = (value) => {
  if (value !== 'custom') {
    const now = new Date()
    let startTime = new Date(now)
    
    switch (value) {
      case '1h':
        startTime.setHours(now.getHours() - 1)
        break
      case '6h':
        startTime.setHours(now.getHours() - 6)
        break
      case '24h':
        startTime.setDate(now.getDate() - 1)
        break
      case '7d':
        startTime.setDate(now.getDate() - 7)
        break
      case '30d':
        startTime.setDate(now.getDate() - 30)
        break
    }
    
    filters.value.startTime = startTime.toISOString()
    filters.value.endTime = now.toISOString()
    customTimeRange.value = []
  }
}

const onCustomTimeChange = (value) => {
  if (value && value.length === 2) {
    filters.value.startTime = value[0]
    filters.value.endTime = value[1]
  }
}

const resetFilters = () => {
  filters.value = {
    timeRange: '24h',
    startTime: null,
    endTime: null,
    granularity: 'hour',
    metrics: ['connections', 'sessions'],
    deviceTypes: [],
    location: [],
    status: [],
    aggregation: 'avg',
    limit: 100,
    includeAlerts: true,
    realtime: false
  }
  customTimeRange.value = []
  onTimeRangeChange('24h')
  applyFilters()
}

const applyFilters = () => {
  emit('update:modelValue', { ...filters.value })
  emit('filter-change', { ...filters.value })
}

const applyPreset = (preset) => {
  Object.assign(filters.value, preset.config)
  if (preset.config.timeRange !== 'custom') {
    onTimeRangeChange(preset.config.timeRange)
  }
  applyFilters()
}

// 监听器
watch(() => props.modelValue, (newValue) => {
  if (newValue) {
    Object.assign(filters.value, newValue)
  }
}, { immediate: true, deep: true })

// 初始化
onTimeRangeChange(filters.value.timeRange)
</script>

<style scoped lang="scss">
.advanced-filters {
  background: #ffffff;
  border-radius: 8px;
  box-shadow: 0 2px 8px rgba(0, 0, 0, 0.1);
  margin-bottom: 16px;
  
  .filter-header {
    display: flex;
    justify-content: space-between;
    align-items: center;
    padding: 12px 16px;
    border-bottom: 1px solid #f0f0f0;
    
    .header-left {
      display: flex;
      align-items: center;
      gap: 8px;
      
      .filter-icon {
        color: #409eff;
        font-size: 16px;
      }
      
      .filter-title {
        font-weight: 600;
        color: #303133;
      }
      
      .active-count {
        font-size: 12px;
      }
    }
    
    .header-right {
      display: flex;
      align-items: center;
      gap: 8px;
    }
  }
  
  .filter-content {
    padding: 16px;
    
    .filter-group {
      margin-bottom: 16px;
      
      .filter-label {
        display: block;
        font-size: 13px;
        font-weight: 500;
        color: #606266;
        margin-bottom: 6px;
      }
      
      .filter-select,
      .filter-input {
        width: 100%;
      }
      
      .granularity-group {
        width: 100%;
        
        :deep(.el-radio-button) {
          flex: 1;
          
          .el-radio-button__inner {
            width: 100%;
          }
        }
      }
      
      .metrics-group {
        display: grid;
        grid-template-columns: repeat(3, 1fr);
        gap: 8px;
        
        :deep(.el-checkbox) {
          margin-right: 0;
        }
      }
      
      .custom-time-range {
        margin-top: 8px;
        
        .custom-picker {
          width: 100%;
        }
      }
      
      .include-alerts,
      .realtime-mode {
        display: block;
        margin-bottom: 8px;
        
        :deep(.el-checkbox__label) {
          font-size: 13px;
        }
      }
    }
    
    .second-row {
      margin-top: 8px;
    }
    
    .advanced-options {
      margin-top: 16px;
      
      .divider-text {
        font-size: 12px;
        color: #909399;
        font-weight: 500;
      }
    }
    
    .preset-filters {
      margin-top: 16px;
      
      .preset-buttons {
        display: flex;
        flex-wrap: wrap;
        gap: 8px;
        
        .preset-btn {
          border-radius: 16px;
          font-size: 12px;
          padding: 6px 12px;
        }
      }
    }
  }
}

// 暗色主题支持
.dark .advanced-filters {
  background: #1f1f1f;
  border-color: #303030;
  
  .filter-header {
    border-color: #303030;
    
    .filter-title {
      color: #e5eaf3;
    }
  }
  
  .filter-label {
    color: #a3a6ad;
  }
  
  .divider-text {
    color: #6c6e72;
  }
}

// 动画
.el-collapse-enter-active,
.el-collapse-leave-active {
  transition: all 0.3s ease;
}

.el-collapse-enter-from,
.el-collapse-leave-to {
  opacity: 0;
  transform: translateY(-10px);
}
</style> 