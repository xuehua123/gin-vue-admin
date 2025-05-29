<!--
  日志详情对话框
  用于显示日志的详细信息
-->
<template>
  <el-dialog
    :model-value="modelValue"
    @update:model-value="emit('update:modelValue', $event)"
    title="日志详情"
    width="800px"
    :close-on-click-modal="false"
  >
    <div v-if="logEntry" class="log-detail">
      <!-- 日志基本信息 -->
      <div class="log-header mb-4">
        <el-descriptions :column="2" border size="small">
          <el-descriptions-item label="日志ID">
            {{ logEntry.id }}
          </el-descriptions-item>
          <el-descriptions-item label="时间戳">
            {{ formatDate(logEntry.timestamp) }}
          </el-descriptions-item>
          <el-descriptions-item label="日志级别">
            <el-tag :type="getLogLevelType(logEntry.level)">
              {{ logEntry.level }}
            </el-tag>
          </el-descriptions-item>
          <el-descriptions-item label="来源">
            {{ logEntry.source || '系统' }}
          </el-descriptions-item>
        </el-descriptions>
      </div>
      
      <!-- 日志内容 -->
      <div class="log-content mb-4">
        <div class="text-sm font-medium mb-2">日志内容</div>
        <el-card shadow="never" class="content-card">
          <pre class="log-message">{{ logEntry.message }}</pre>
        </el-card>
      </div>
      
      <!-- 上下文信息 -->
      <div v-if="logEntry.context" class="log-context mb-4">
        <div class="text-sm font-medium mb-2">上下文信息</div>
        <el-card shadow="never" class="context-card">
          <pre class="log-context-data">{{ formatContext(logEntry.context) }}</pre>
        </el-card>
      </div>
      
      <!-- 堆栈信息 -->
      <div v-if="logEntry.stack" class="log-stack">
        <div class="text-sm font-medium mb-2">堆栈跟踪</div>
        <el-collapse>
          <el-collapse-item title="查看堆栈信息" name="1">
            <pre class="log-stack-trace">{{ logEntry.stack }}</pre>
          </el-collapse-item>
        </el-collapse>
      </div>
      
      <!-- 相关日志 -->
      <div v-if="relatedLogs && relatedLogs.length > 0" class="related-logs mt-4">
        <div class="text-sm font-medium mb-2">相关日志</div>
        <el-table :data="relatedLogs" border style="width: 100%" size="small">
          <el-table-column prop="timestamp" label="时间" width="180">
            <template #default="scope">
              {{ formatDate(scope.row.timestamp) }}
            </template>
          </el-table-column>
          <el-table-column prop="level" label="级别" width="100">
            <template #default="scope">
              <el-tag :type="getLogLevelType(scope.row.level)" size="small">
                {{ scope.row.level }}
              </el-tag>
            </template>
          </el-table-column>
          <el-table-column prop="message" label="内容" show-overflow-tooltip />
          <el-table-column label="操作" width="80">
            <template #default="scope">
              <el-button link type="primary" size="small" @click="viewRelatedLog(scope.row)">
                查看
              </el-button>
            </template>
          </el-table-column>
        </el-table>
      </div>
    </div>
    <div v-else class="text-center py-8 text-gray-500">
      没有选择日志或日志不存在
    </div>
    
    <template #footer>
      <span class="dialog-footer">
        <el-button @click="handleClose">关闭</el-button>
        <el-button v-if="logEntry && logEntry.level === 'ERROR'" type="primary" @click="handleAnalyze">
          分析错误
        </el-button>
      </span>
    </template>
  </el-dialog>
</template>

<script setup>
import { ref, computed } from 'vue'
import { ElMessage } from 'element-plus'

const props = defineProps({
  modelValue: {
    type: Boolean,
    default: false
  },
  logEntry: {
    type: Object,
    default: null
  }
})

const emit = defineEmits(['update:modelValue', 'analyze'])

// 模拟相关日志数据
const relatedLogs = ref([
  {
    id: 101,
    timestamp: new Date(Date.now() - 5000),
    level: 'INFO',
    message: '系统初始化完成',
    source: '系统'
  },
  {
    id: 102,
    timestamp: new Date(Date.now() - 3000),
    level: 'WARN',
    message: '连接超时，准备重试',
    source: '网络'
  }
])

// 格式化日期
const formatDate = (timestamp) => {
  if (!timestamp) return ''
  return new Date(timestamp).toLocaleString('zh-CN', {
    year: 'numeric',
    month: '2-digit',
    day: '2-digit',
    hour: '2-digit',
    minute: '2-digit',
    second: '2-digit',
    fractionalSecondDigits: 3
  })
}

// 获取日志级别对应的类型
const getLogLevelType = (level) => {
  const typeMap = {
    'DEBUG': 'info',
    'INFO': 'success',
    'WARN': 'warning',
    'ERROR': 'danger'
  }
  return typeMap[level] || 'info'
}

// 格式化上下文信息
const formatContext = (context) => {
  if (!context) return ''
  
  if (typeof context === 'string') {
    return context
  }
  
  try {
    return JSON.stringify(context, null, 2)
  } catch (e) {
    return String(context)
  }
}

// 查看相关日志
const viewRelatedLog = (log) => {
  // 这里可以实现切换到相关日志的逻辑
  ElMessage.info(`查看日志ID: ${log.id}`)
}

// 关闭对话框
const handleClose = () => {
  emit('update:modelValue', false)
}

// 分析错误
const handleAnalyze = () => {
  if (!props.logEntry) return
  
  emit('analyze', props.logEntry)
  ElMessage.success('开始分析错误...')
}
</script>

<style scoped>
.log-detail {
  max-height: 600px;
  overflow-y: auto;
}

.content-card,
.context-card {
  background-color: #f8f9fa;
}

.log-message,
.log-context-data,
.log-stack-trace {
  font-family: 'Courier New', monospace;
  white-space: pre-wrap;
  word-break: break-all;
  margin: 0;
  line-height: 1.5;
}

.log-stack-trace {
  color: #e74c3c;
}

.dialog-footer {
  display: flex;
  justify-content: flex-end;
}
</style> 