<!--
  命令详情对话框
  用于显示APDU命令的详细信息
-->
<template>
  <el-dialog
    :model-value="modelValue"
    @update:model-value="emit('update:modelValue', $event)"
    title="命令详情"
    width="700px"
    :close-on-click-modal="false"
  >
    <div v-if="command" class="command-detail">
      <!-- 命令基本信息 -->
      <div class="command-header mb-4">
        <el-descriptions :column="2" border>
          <el-descriptions-item label="命令类型">
            {{ getCommandTypeName(command.type) }}
          </el-descriptions-item>
          <el-descriptions-item label="时间戳">
            {{ formatDate(command.timestamp) }}
          </el-descriptions-item>
          <el-descriptions-item label="方向">
            <el-tag :type="command.direction === 'outgoing' ? 'success' : 'info'">
              {{ command.direction === 'outgoing' ? '发送' : '接收' }}
            </el-tag>
          </el-descriptions-item>
          <el-descriptions-item label="状态">
            <el-tag :type="getStatusType(command.status)">
              {{ getStatusName(command.status) }}
            </el-tag>
          </el-descriptions-item>
        </el-descriptions>
      </div>
      
      <!-- 命令数据 -->
      <div class="command-data mb-4">
        <div class="text-sm font-medium mb-2">命令数据</div>
        <el-card shadow="never" class="data-card">
          <pre class="hex-data">{{ formatHexData(command.data) }}</pre>
        </el-card>
      </div>
      
      <!-- 命令解析 -->
      <div class="command-parsing mb-4">
        <div class="text-sm font-medium mb-2">命令解析</div>
        <el-table :data="parseCommand(command)" border style="width: 100%">
          <el-table-column prop="field" label="字段" width="150" />
          <el-table-column prop="value" label="值" min-width="180" />
          <el-table-column prop="description" label="描述" min-width="200" />
        </el-table>
      </div>
      
      <!-- 响应数据 -->
      <div v-if="command.response" class="command-response">
        <div class="text-sm font-medium mb-2">响应数据</div>
        <el-card shadow="never" class="data-card">
          <pre class="hex-data">{{ formatHexData(command.response.data) }}</pre>
        </el-card>
        
        <div class="mt-2 response-status">
          <span class="text-sm mr-2">状态码:</span>
          <el-tag :type="getResponseStatusType(command.response.status)">
            {{ command.response.status }} 
            <span v-if="getResponseStatusDesc(command.response.status)" class="ml-1">
              ({{ getResponseStatusDesc(command.response.status) }})
            </span>
          </el-tag>
        </div>
      </div>
    </div>
    <div v-else class="text-center py-8 text-gray-500">
      没有选择命令或命令不存在
    </div>
    
    <template #footer>
      <span class="dialog-footer">
        <el-button @click="handleClose">关闭</el-button>
        <el-button 
          v-if="command && command.type === 'apdu'" 
          type="primary"
          @click="handleReplay"
        >
          重放命令
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
  command: {
    type: Object,
    default: null
  }
})

const emit = defineEmits(['update:modelValue', 'replay'])

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

// 获取命令类型名称
const getCommandTypeName = (type) => {
  const typeMap = {
    'apdu': 'APDU 命令',
    'control': '控制命令',
    'status': '状态更新',
    'raw': '原始数据'
  }
  return typeMap[type] || type
}

// 获取状态类型样式
const getStatusType = (status) => {
  switch (status) {
    case 'success': return 'success'
    case 'pending': return 'warning'
    case 'error': return 'danger'
    default: return 'info'
  }
}

// 获取状态名称
const getStatusName = (status) => {
  const statusMap = {
    'success': '成功',
    'pending': '处理中',
    'error': '错误',
    'timeout': '超时'
  }
  return statusMap[status] || status
}

// 格式化十六进制数据
const formatHexData = (data) => {
  if (!data) return ''
  
  // 模拟数据，实际应用中根据数据格式处理
  if (typeof data === 'string') {
    // 按照每8个字节分组显示
    return data.match(/.{1,16}/g).join('\n')
  }
  
  return JSON.stringify(data, null, 2)
}

// 解析命令数据
const parseCommand = (command) => {
  if (!command || !command.data) return []
  
  // 这里只是模拟解析逻辑，实际应用中需要根据命令类型实现相应的解析
  if (command.type === 'apdu') {
    return [
      { field: 'CLA', value: '00', description: '指令类' },
      { field: 'INS', value: 'A4', description: '指令码 (SELECT)' },
      { field: 'P1', value: '04', description: '参数1' },
      { field: 'P2', value: '00', description: '参数2' },
      { field: 'Lc', value: '0A', description: '数据长度' },
      { field: 'DATA', value: 'A0000000031010', description: '应用标识符 (AID)' }
    ]
  }
  
  return [{ field: '数据', value: command.data, description: '原始数据' }]
}

// 获取响应状态类型
const getResponseStatusType = (status) => {
  if (!status) return 'info'
  
  const code = status.substring(0, 2)
  switch (code) {
    case '90': return 'success'
    case '61': return 'success'
    case '62': 
    case '63': return 'warning'
    default: return 'danger'
  }
}

// 获取响应状态描述
const getResponseStatusDesc = (status) => {
  if (!status) return ''
  
  const statusMap = {
    '9000': '处理成功',
    '6A82': '文件或应用未找到',
    '6A86': '参数错误',
    '6700': '数据长度错误',
    '6E00': '不支持的指令'
  }
  
  return statusMap[status] || ''
}

// 关闭对话框
const handleClose = () => {
  emit('update:modelValue', false)
}

// 重放命令
const handleReplay = () => {
  if (!props.command) return
  
  emit('replay', props.command)
  ElMessage.success('命令已发送，等待响应...')
  emit('update:modelValue', false)
}
</script>

<style scoped>
.command-detail {
  max-height: 600px;
  overflow-y: auto;
}

.data-card {
  background-color: #f8f9fa;
}

.hex-data {
  font-family: 'Courier New', monospace;
  white-space: pre-wrap;
  word-break: break-all;
  margin: 0;
  line-height: 1.5;
}

.dialog-footer {
  display: flex;
  justify-content: flex-end;
}
</style> 