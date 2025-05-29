<!-- 
  配置历史记录对话框
  用于显示单个配置项的历史版本记录
-->
<template>
  <el-dialog
    :model-value="modelValue"
    @update:model-value="emit('update:modelValue', $event)"
    :title="`配置历史 - ${configItem?.key || ''}`"
    width="700px"
    :close-on-click-modal="false"
  >
    <div v-if="configItem" class="config-history">
      <div class="mb-4 config-item-info">
        <div class="text-gray-500 mb-1">
          <span class="font-medium">配置项:</span> {{ configItem.key }}
        </div>
        <div v-if="configItem.description" class="text-gray-500 mb-1">
          <span class="font-medium">描述:</span> {{ configItem.description }}
        </div>
        <div class="text-gray-500">
          <span class="font-medium">当前值:</span> 
          <span>{{ formatValue(configItem.value) }}</span>
        </div>
      </div>
      
      <el-table
        :data="historyRecords"
        style="width: 100%"
        :default-sort="{ prop: 'timestamp', order: 'descending' }"
        border
      >
        <el-table-column prop="version" label="版本" width="80" sortable />
        <el-table-column prop="timestamp" label="修改时间" width="160" sortable>
          <template #default="scope">
            {{ formatDate(scope.row.timestamp) }}
          </template>
        </el-table-column>
        <el-table-column prop="author" label="操作人" width="100" />
        <el-table-column prop="value" label="配置值">
          <template #default="scope">
            <div class="value-display">
              {{ formatValue(scope.row.value) }}
            </div>
          </template>
        </el-table-column>
        <el-table-column label="操作" width="120">
          <template #default="scope">
            <el-button 
              link 
              type="primary" 
              size="small"
              @click="handleRestore(scope.row)"
            >
              恢复此值
            </el-button>
          </template>
        </el-table-column>
      </el-table>
      
      <div class="mt-4 flex justify-between items-center">
        <div class="text-gray-500 text-sm">
          共 {{ historyRecords.length }} 条历史记录
        </div>
        <el-pagination
          v-if="historyRecords.length > 10"
          :total="historyRecords.length"
          :page-size="10"
          layout="prev, pager, next"
        />
      </div>
    </div>
    <div v-else class="text-center py-8 text-gray-500">
      没有选择配置项或配置项不存在
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

const props = defineProps({
  modelValue: {
    type: Boolean,
    default: false
  },
  configItem: {
    type: Object,
    default: null
  }
})

const emit = defineEmits(['update:modelValue', 'restore'])

// 模拟历史记录数据
const historyRecords = ref([
  {
    id: 1,
    version: '2.1.0',
    timestamp: new Date(Date.now() - 1000 * 60 * 60 * 24 * 2),
    author: 'admin',
    value: 'true',
    description: '启用安全配置'
  },
  {
    id: 2,
    version: '2.0.0',
    timestamp: new Date(Date.now() - 1000 * 60 * 60 * 24 * 10),
    author: 'system',
    value: 'false',
    description: '初始配置'
  }
])

// 格式化日期显示
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

// 根据类型格式化值显示
const formatValue = (value) => {
  if (value === undefined || value === null) return '空'
  
  if (typeof value === 'object') {
    try {
      return JSON.stringify(value, null, 2)
    } catch (e) {
      return String(value)
    }
  }
  
  return String(value)
}

// 处理恢复历史版本
const handleRestore = (record) => {
  ElMessageBox.confirm(
    `确定要将配置项 "${props.configItem?.key}" 恢复到版本 ${record.version} 的值吗？`,
    '恢复确认',
    {
      confirmButtonText: '确定',
      cancelButtonText: '取消',
      type: 'warning'
    }
  ).then(() => {
    // 触发恢复事件
    emit('restore', {
      configKey: props.configItem?.key,
      value: record.value,
      version: record.version
    })
    
    ElMessage.success('配置值已恢复')
    emit('update:modelValue', false)
  }).catch(() => {
    // 用户取消操作
  })
}

// 关闭对话框
const handleClose = () => {
  emit('update:modelValue', false)
}

// 在实际应用中，可以在对话框打开时加载历史数据
watch(() => props.modelValue, (val) => {
  if (val && props.configItem) {
    // 这里可以根据configItem.key加载实际的历史记录
    // 目前使用模拟数据
  }
})
</script>

<style scoped>
.config-history {
  max-height: 500px;
  overflow-y: auto;
}

.config-item-info {
  background-color: #f8f9fa;
  padding: 12px;
  border-radius: 4px;
}

.value-display {
  max-height: 80px;
  overflow-y: auto;
  white-space: pre-wrap;
  word-break: break-all;
}

.dialog-footer {
  display: flex;
  justify-content: flex-end;
}
</style> 