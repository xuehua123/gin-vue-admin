<!-- 
  版本差异对话框
  用于比较两个配置版本之间的差异
-->
<template>
  <el-dialog
    :model-value="modelValue"
    @update:model-value="emit('update:modelValue', $event)"
    title="版本差异比较"
    width="800px"
    :close-on-click-modal="false"
  >
    <div v-if="version && compareVersion" class="version-diff">
      <!-- 版本选择 -->
      <div class="version-selection mb-4">
        <div class="flex items-center justify-between">
          <div class="version-select-container">
            <div class="text-sm mb-1">当前版本</div>
            <el-select 
              v-model="selectedVersionId" 
              size="default"
              placeholder="选择版本"
              style="width: 320px"
            >
              <el-option
                v-for="v in versionOptions"
                :key="v.id"
                :label="`v${v.version} (${formatDate(v.createdAt)})`"
                :value="v.id"
              />
            </el-select>
          </div>
          
          <div class="comparison-arrow flex items-center justify-center mx-4">
            <el-icon class="text-gray-400" :size="24"><ArrowRight /></el-icon>
          </div>
          
          <div class="version-select-container">
            <div class="text-sm mb-1">比较版本</div>
            <el-select 
              v-model="selectedCompareVersionId" 
              size="default"
              placeholder="选择比较版本"
              style="width: 320px"
            >
              <el-option
                v-for="v in versionOptions"
                :key="v.id"
                :label="`v${v.version} (${formatDate(v.createdAt)})`"
                :value="v.id"
                :disabled="v.id === selectedVersionId"
              />
            </el-select>
          </div>
        </div>
      </div>
      
      <!-- 版本信息 -->
      <div class="version-info-container mb-4">
        <div class="grid grid-cols-2 gap-4">
          <!-- 第一个版本信息 -->
          <div class="version-info bg-gray-50 p-3 rounded">
            <div class="text-base font-medium mb-1">
              v{{ version.version }}
              <el-tag v-if="version.isCurrent" size="small" type="success" class="ml-1">当前</el-tag>
            </div>
            <div class="text-sm text-gray-500 mb-1">
              <span class="font-medium">创建时间:</span> {{ formatDate(version.createdAt) }}
            </div>
            <div class="text-sm text-gray-500 mb-1">
              <span class="font-medium">创建人:</span> {{ version.author }}
            </div>
            <div class="text-sm text-gray-500">
              <span class="font-medium">描述:</span> {{ version.description || '无' }}
            </div>
          </div>
          
          <!-- 第二个版本信息 -->
          <div class="version-info bg-gray-50 p-3 rounded">
            <div class="text-base font-medium mb-1">
              v{{ compareVersion.version }}
              <el-tag v-if="compareVersion.isCurrent" size="small" type="success" class="ml-1">当前</el-tag>
            </div>
            <div class="text-sm text-gray-500 mb-1">
              <span class="font-medium">创建时间:</span> {{ formatDate(compareVersion.createdAt) }}
            </div>
            <div class="text-sm text-gray-500 mb-1">
              <span class="font-medium">创建人:</span> {{ compareVersion.author }}
            </div>
            <div class="text-sm text-gray-500">
              <span class="font-medium">描述:</span> {{ compareVersion.description || '无' }}
            </div>
          </div>
        </div>
      </div>
      
      <!-- 差异表格 -->
      <div class="diff-table-container">
        <div class="text-sm font-medium mb-2">配置差异 ({{ diffItems.length }} 项)</div>
        <el-table
          :data="diffItems"
          style="width: 100%"
          border
          :header-cell-style="{ background: '#f5f7fa' }"
        >
          <el-table-column prop="key" label="配置项" min-width="150" />
          <el-table-column label="v1 值" min-width="200">
            <template #default="scope">
              <div class="diff-value" :class="{ 'diff-removed': scope.row.changeType === 'removed' }">
                {{ formatDiffValue(scope.row.oldValue) }}
              </div>
            </template>
          </el-table-column>
          <el-table-column label="v2 值" min-width="200">
            <template #default="scope">
              <div class="diff-value" :class="{ 'diff-added': scope.row.changeType === 'added' }">
                {{ formatDiffValue(scope.row.newValue) }}
              </div>
            </template>
          </el-table-column>
          <el-table-column label="变更类型" width="100">
            <template #default="scope">
              <el-tag :type="getDiffTypeTag(scope.row.changeType)" size="small">
                {{ getDiffTypeLabel(scope.row.changeType) }}
              </el-tag>
            </template>
          </el-table-column>
        </el-table>
      </div>
      
      <!-- 操作按钮 -->
      <div class="mt-4 flex justify-end">
        <el-button 
          type="primary" 
          size="default"
          @click="handleApplyVersion"
        >
          应用选中版本
        </el-button>
      </div>
    </div>
    <div v-else class="text-center py-8 text-gray-500">
      请选择要比较的版本
    </div>
    
    <template #footer>
      <span class="dialog-footer">
        <el-button @click="handleClose">关闭</el-button>
      </span>
    </template>
  </el-dialog>
</template>

<script setup>
import { ref, computed, watch } from 'vue'
import { ElMessage, ElMessageBox } from 'element-plus'
import { ArrowRight } from '@element-plus/icons-vue'

const props = defineProps({
  modelValue: {
    type: Boolean,
    default: false
  },
  versions: {
    type: Array,
    default: () => []
  },
  initialVersionId: {
    type: [Number, String],
    default: null
  },
  initialCompareVersionId: {
    type: [Number, String],
    default: null
  }
})

const emit = defineEmits(['update:modelValue', 'apply-version'])

// 选中的版本ID
const selectedVersionId = ref(props.initialVersionId || null)
const selectedCompareVersionId = ref(props.initialCompareVersionId || null)

// 获取版本选项
const versionOptions = computed(() => {
  return props.versions || []
})

// 获取选中的版本对象
const version = computed(() => {
  if (!selectedVersionId.value) return null
  return props.versions.find(v => v.id === selectedVersionId.value) || null
})

// 获取比较的版本对象
const compareVersion = computed(() => {
  if (!selectedCompareVersionId.value) return null
  return props.versions.find(v => v.id === selectedCompareVersionId.value) || null
})

// 模拟差异数据
const diffItems = ref([
  {
    key: 'server.port',
    oldValue: 8080,
    newValue: 8888,
    changeType: 'modified'
  },
  {
    key: 'security.auth_mode',
    oldValue: 'none',
    newValue: 'token',
    changeType: 'modified'
  },
  {
    key: 'logging.level',
    oldValue: null,
    newValue: 'INFO',
    changeType: 'added'
  },
  {
    key: 'security.old_param',
    oldValue: 'old_value',
    newValue: null,
    changeType: 'removed'
  }
])

// 格式化日期
const formatDate = (date) => {
  if (!date) return ''
  return new Date(date).toLocaleString('zh-CN', {
    year: 'numeric',
    month: '2-digit',
    day: '2-digit',
    hour: '2-digit',
    minute: '2-digit'
  })
}

// 格式化差异值显示
const formatDiffValue = (value) => {
  if (value === undefined || value === null) return '-'
  
  if (typeof value === 'object') {
    try {
      return JSON.stringify(value, null, 2)
    } catch (e) {
      return String(value)
    }
  }
  
  return String(value)
}

// 获取差异类型标签
const getDiffTypeTag = (type) => {
  switch (type) {
    case 'added': return 'success'
    case 'removed': return 'danger'
    case 'modified': return 'warning'
    default: return 'info'
  }
}

// 获取差异类型文本
const getDiffTypeLabel = (type) => {
  switch (type) {
    case 'added': return '新增'
    case 'removed': return '删除'
    case 'modified': return '修改'
    default: return '未知'
  }
}

// 应用选中版本
const handleApplyVersion = () => {
  if (!version.value) return
  
  ElMessageBox.confirm(
    `确定要应用版本 v${version.value.version} 的配置吗？这将覆盖当前配置。`,
    '应用确认',
    {
      confirmButtonText: '确定',
      cancelButtonText: '取消',
      type: 'warning'
    }
  ).then(() => {
    // 触发应用版本事件
    emit('apply-version', version.value)
    
    ElMessage.success(`已应用版本 v${version.value.version} 的配置`)
    emit('update:modelValue', false)
  }).catch(() => {
    // 用户取消操作
  })
}

// 关闭对话框
const handleClose = () => {
  emit('update:modelValue', false)
}

// 监听对话框显示状态变化
watch(() => props.modelValue, (val) => {
  if (val) {
    // 初始化选择
    selectedVersionId.value = props.initialVersionId || 
      (props.versions.length > 0 ? props.versions[0].id : null)
      
    // 默认比较第二个版本，如果有的话
    selectedCompareVersionId.value = props.initialCompareVersionId || 
      (props.versions.length > 1 ? props.versions[1].id : null)
      
    // 加载差异数据
    // 实际应用中应该根据两个版本ID加载实际差异
    // loadDiff(selectedVersionId.value, selectedCompareVersionId.value)
  }
})

// 监听版本选择变化
watch([selectedVersionId, selectedCompareVersionId], ([vId, cId]) => {
  if (vId && cId && vId !== cId) {
    // 加载差异数据
    // loadDiff(vId, cId)
  }
})
</script>

<style scoped>
.version-diff {
  max-height: 600px;
  overflow-y: auto;
}

.diff-value {
  max-height: 80px;
  overflow-y: auto;
  white-space: pre-wrap;
  word-break: break-all;
}

.diff-added {
  background-color: rgba(103, 194, 58, 0.1);
}

.diff-removed {
  background-color: rgba(245, 108, 108, 0.1);
  text-decoration: line-through;
}

.dialog-footer {
  display: flex;
  justify-content: flex-end;
}
</style> 