<!--
  日志详情弹窗组件
  显示审计日志的详细信息
-->
<template>
  <el-dialog
    v-model="visible"
    title="审计日志详情"
    width="700px"
    destroy-on-close
  >
    <div v-if="logData" class="log-detail-content">
      <!-- 基本信息 -->
      <el-descriptions :column="2" border class="mb-4">
        <el-descriptions-item label="时间">
          {{ formatTime(logData.timestamp) }}
        </el-descriptions-item>
        
        <el-descriptions-item label="事件类型">
          <el-tag :type="getEventTypeColor(logData.event_type)" size="small">
            {{ getEventTypeText(logData.event_type) }}
          </el-tag>
        </el-descriptions-item>
        
        <el-descriptions-item label="会话ID" v-if="logData.session_id">
          <el-link 
            type="warning" 
            @click="$router.push(`/nfc-relay-admin/sessions?sessionID=${logData.session_id}`)"
            class="font-mono text-sm"
          >
            {{ logData.session_id }}
          </el-link>
        </el-descriptions-item>
        
        <el-descriptions-item label="源IP">
          {{ logData.source_ip }}
        </el-descriptions-item>
        
        <el-descriptions-item label="发起方客户端" v-if="logData.client_id_initiator">
          <el-link 
            type="primary" 
            @click="$router.push(`/nfc-relay-admin/clients?clientID=${logData.client_id_initiator}`)"
            class="font-mono text-sm"
          >
            {{ logData.client_id_initiator }}
          </el-link>
        </el-descriptions-item>
        
        <el-descriptions-item label="响应方客户端" v-if="logData.client_id_responder">
          <el-link 
            type="info" 
            @click="$router.push(`/nfc-relay-admin/clients?clientID=${logData.client_id_responder}`)"
            class="font-mono text-sm"
          >
            {{ logData.client_id_responder }}
          </el-link>
        </el-descriptions-item>
        
        <el-descriptions-item label="用户ID" v-if="logData.user_id">
          {{ logData.user_id }}
        </el-descriptions-item>
      </el-descriptions>

      <!-- 详细信息 -->
      <el-card shadow="never" v-if="logData.details && Object.keys(logData.details).length > 0">
        <template #header>
          <span class="font-semibold">详细信息</span>
        </template>
        
        <el-descriptions :column="1" border>
          <el-descriptions-item 
            v-for="(value, key) in logData.details" 
            :key="key"
            :label="getDetailLabel(key)"
          >
            <div class="detail-value">
              <template v-if="key === 'error_message'">
                <el-text type="danger">{{ value }}</el-text>
              </template>
              <template v-else-if="key === 'apdu_length'">
                <el-tag size="small" type="info">{{ value }} 字节</el-tag>
              </template>
              <template v-else-if="key === 'session_duration'">
                <el-tag size="small" type="success">{{ value }}</el-tag>
              </template>
              <template v-else-if="typeof value === 'object'">
                <pre class="json-display">{{ JSON.stringify(value, null, 2) }}</pre>
              </template>
              <template v-else>
                {{ value }}
              </template>
            </div>
          </el-descriptions-item>
        </el-descriptions>
      </el-card>

      <!-- 原始JSON数据 -->
      <el-card shadow="never" class="mt-4">
        <template #header>
          <div class="flex justify-between items-center">
            <span class="font-semibold">原始数据 (JSON)</span>
            <el-button size="small" @click="copyToClipboard">
              <el-icon><CopyDocument /></el-icon>
              复制
            </el-button>
          </div>
        </template>
        
        <pre class="json-display">{{ JSON.stringify(logData, null, 2) }}</pre>
      </el-card>
    </div>

    <template #footer>
      <el-button @click="handleClose">关闭</el-button>
    </template>
  </el-dialog>
</template>

<script setup>
import { computed } from 'vue'
import { ElMessage } from 'element-plus'
import { CopyDocument } from '@element-plus/icons-vue'
import { formatTime } from '@/utils/format'

const props = defineProps({
  visible: {
    type: Boolean,
    default: false
  },
  logData: {
    type: Object,
    default: null
  }
})

const emit = defineEmits(['update:visible'])

// 计算属性
const visible = computed({
  get: () => props.visible,
  set: (val) => emit('update:visible', val)
})

// 工具函数
const getEventTypeColor = (eventType) => {
  switch (eventType) {
    case 'session_established':
    case 'apdu_relayed_success': return 'success'
    case 'apdu_relayed_failure':
    case 'auth_failure': return 'danger'
    case 'client_disconnected':
    case 'session_terminated': return 'warning'
    default: return 'info'
  }
}

const getEventTypeText = (eventType) => {
  switch (eventType) {
    case 'session_established': return '会话建立'
    case 'apdu_relayed_success': return 'APDU转发成功'
    case 'apdu_relayed_failure': return 'APDU转发失败'
    case 'auth_failure': return '认证失败'
    case 'client_disconnected': return '客户端断开连接'
    case 'session_terminated': return '会话终止'
    default: return eventType
  }
}

const getDetailLabel = (key) => {
  const labelMap = {
    apdu_length: 'APDU长度',
    error_message: '错误信息',
    reason: '原因',
    session_duration: '会话持续时间',
    response_code: '响应码',
    client_version: '客户端版本',
    user_agent: 'User Agent',
    connection_type: '连接类型'
  }
  return labelMap[key] || key
}

// 方法
const handleClose = () => {
  visible.value = false
}

const copyToClipboard = async () => {
  try {
    await navigator.clipboard.writeText(JSON.stringify(props.logData, null, 2))
    ElMessage.success('已复制到剪贴板')
  } catch (error) {
    ElMessage.error('复制失败')
  }
}
</script>

<style scoped lang="scss">
.log-detail-content {
  .mb-4 {
    margin-bottom: 1rem;
  }
  
  .mt-4 {
    margin-top: 1rem;
  }
  
  .font-semibold {
    font-weight: 600;
  }
  
  .font-mono {
    font-family: ui-monospace, SFMono-Regular, monospace;
  }
  
  .text-sm {
    font-size: 0.875rem;
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
  
  .detail-value {
    word-break: break-all;
  }
  
  .json-display {
    background-color: #f5f7fa;
    border: 1px solid #e4e7ed;
    border-radius: 4px;
    padding: 12px;
    font-family: ui-monospace, SFMono-Regular, monospace;
    font-size: 12px;
    line-height: 1.5;
    max-height: 300px;
    overflow-y: auto;
    white-space: pre-wrap;
    word-break: break-all;
  }
}
</style> 