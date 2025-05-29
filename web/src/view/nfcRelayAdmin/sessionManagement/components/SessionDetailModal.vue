<!--
  会话详情弹窗组件
  显示会话的详细信息和APDU交换统计
-->
<template>
  <el-dialog
    v-model="visible"
    title="会话详细信息"
    width="900px"
    :before-close="handleClose"
    destroy-on-close
  >
    <div v-if="loading" v-loading="loading" style="height: 400px;"></div>
    
    <div v-else-if="sessionData" class="session-detail-content">
      <!-- 基本信息 -->
      <el-card shadow="never" class="mb-4">
        <template #header>
          <div class="flex items-center">
            <el-icon class="mr-2" :color="getStatusColor(sessionData.status)">
              <component :is="getStatusIcon(sessionData.status)" />
            </el-icon>
            <span class="font-semibold">会话基本信息</span>
          </div>
        </template>
        
        <el-descriptions :column="2" border>
          <el-descriptions-item label="会话ID">
            <code class="font-mono text-sm">{{ sessionData.session_id }}</code>
          </el-descriptions-item>
          
          <el-descriptions-item label="状态">
            <el-tag :type="getStatusTagType(sessionData.status)" size="small">
              {{ getStatusText(sessionData.status) }}
            </el-tag>
          </el-descriptions-item>
          
          <el-descriptions-item label="创建时间">
            {{ formatTime(sessionData.created_at) }}
          </el-descriptions-item>
          
          <el-descriptions-item label="最后活动">
            {{ formatTime(sessionData.last_activity_at) }}
          </el-descriptions-item>
          
          <el-descriptions-item label="终止时间" v-if="sessionData.terminated_at">
            {{ formatTime(sessionData.terminated_at) }}
          </el-descriptions-item>
          
          <el-descriptions-item label="终止原因" v-if="sessionData.termination_reason">
            {{ sessionData.termination_reason }}
          </el-descriptions-item>
        </el-descriptions>
      </el-card>

      <!-- 参与方信息 -->
      <div class="grid grid-cols-2 gap-4 mb-4">
        <el-card shadow="never">
          <template #header>
            <div class="flex items-center">
              <el-icon class="mr-2" color="#67c23a">
                <Phone />
              </el-icon>
              <span class="font-semibold">传卡端 (Provider)</span>
            </div>
          </template>
          
          <el-descriptions :column="1" border>
            <el-descriptions-item label="客户端ID">
              <el-link 
                type="success" 
                @click="$router.push(`/nfc-relay-admin/clients?clientID=${sessionData.provider_info.client_id}`)"
                class="font-mono text-sm"
              >
                {{ sessionData.provider_info.client_id }}
              </el-link>
            </el-descriptions-item>
            <el-descriptions-item label="用户ID">
              {{ sessionData.provider_info.user_id }}
            </el-descriptions-item>
            <el-descriptions-item label="设备名称">
              {{ sessionData.provider_info.display_name }}
            </el-descriptions-item>
            <el-descriptions-item label="IP地址">
              {{ sessionData.provider_info.ip_address }}
            </el-descriptions-item>
          </el-descriptions>
        </el-card>
        
        <el-card shadow="never">
          <template #header>
            <div class="flex items-center">
              <el-icon class="mr-2" color="#e6a23c">
                <Monitor />
              </el-icon>
              <span class="font-semibold">收卡端 (Receiver)</span>
            </div>
          </template>
          
          <el-descriptions :column="1" border>
            <el-descriptions-item label="客户端ID">
              <el-link 
                type="warning" 
                @click="$router.push(`/nfc-relay-admin/clients?clientID=${sessionData.receiver_info.client_id}`)"
                class="font-mono text-sm"
              >
                {{ sessionData.receiver_info.client_id }}
              </el-link>
            </el-descriptions-item>
            <el-descriptions-item label="用户ID">
              {{ sessionData.receiver_info.user_id }}
            </el-descriptions-item>
            <el-descriptions-item label="设备名称">
              {{ sessionData.receiver_info.display_name }}
            </el-descriptions-item>
            <el-descriptions-item label="IP地址">
              {{ sessionData.receiver_info.ip_address }}
            </el-descriptions-item>
          </el-descriptions>
        </el-card>
      </div>

      <!-- APDU交换统计 -->
      <el-card shadow="never" class="mb-4">
        <template #header>
          <span class="font-semibold">APDU交换统计</span>
        </template>
        
        <div class="grid grid-cols-2 gap-4">
          <div class="stat-item">
            <div class="stat-label">上行 (Receiver → Provider)</div>
            <div class="stat-value">{{ sessionData.apdu_exchange_count?.upstream || 0 }}</div>
            <div class="stat-desc">通常是指令</div>
          </div>
          <div class="stat-item">
            <div class="stat-label">下行 (Provider → Receiver)</div>
            <div class="stat-value">{{ sessionData.apdu_exchange_count?.downstream || 0 }}</div>
            <div class="stat-desc">通常是响应</div>
          </div>
        </div>
      </el-card>

      <!-- 会话事件历史 -->
      <el-card shadow="never" class="mb-4" v-if="sessionData.session_events_history && sessionData.session_events_history.length">
        <template #header>
          <span class="font-semibold">会话事件历史</span>
        </template>
        
        <el-timeline>
          <el-timeline-item
            v-for="(event, index) in sessionData.session_events_history"
            :key="index"
            :timestamp="formatTime(event.timestamp)"
            placement="top"
          >
            <el-card shadow="never" class="timeline-card">
              <div class="flex items-center justify-between">
                <div class="flex items-center">
                  <el-icon class="mr-2" :color="getEventIconColor(event.event)">
                    <component :is="getEventIcon(event.event)" />
                  </el-icon>
                  <span>{{ getEventText(event.event) }}</span>
                </div>
                <span v-if="event.client_id" class="text-xs text-gray-400 font-mono">
                  {{ event.client_id }}
                </span>
              </div>
            </el-card>
          </el-timeline-item>
        </el-timeline>
      </el-card>

      <!-- 相关审计日志摘要 -->
      <el-card shadow="never" v-if="sessionData.related_audit_logs_summary && sessionData.related_audit_logs_summary.length">
        <template #header>
          <div class="flex justify-between items-center">
            <span class="font-semibold">最近审计日志</span>
            <el-button 
              link 
              type="primary" 
              @click="$router.push(`/nfc-relay-admin/audit-logs?sessionID=${sessionData.session_id}`)"
            >
              查看全部
            </el-button>
          </div>
        </template>
        
        <el-table :data="sessionData.related_audit_logs_summary" size="small">
          <el-table-column prop="timestamp" label="时间" width="180">
            <template #default="{ row }">
              {{ formatTime(row.timestamp) }}
            </template>
          </el-table-column>
          <el-table-column prop="event_type" label="事件类型" width="180" />
          <el-table-column prop="details_summary" label="详情" />
        </el-table>
      </el-card>
    </div>

    <div v-else class="text-center py-8 text-gray-500">
      获取会话详情失败
    </div>

    <template #footer>
      <div class="flex justify-between">
        <div>
          <el-button 
            v-if="sessionData && sessionData.status === 'paired'" 
            type="danger" 
            @click="handleTerminate"
            :loading="terminating"
          >
            <el-icon><Close /></el-icon>
            强制终止会话
          </el-button>
        </div>
        <div>
          <el-button @click="handleClose">关闭</el-button>
          <el-button type="primary" @click="refreshData" :loading="loading">
            <el-icon><Refresh /></el-icon>
            刷新
          </el-button>
        </div>
      </div>
    </template>
  </el-dialog>
</template>

<script setup>
import { ref, watch, computed } from 'vue'
import { ElMessage, ElMessageBox } from 'element-plus'
import { 
  Phone, 
  Monitor,
  Link,
  Clock,
  CircleCheck,
  Close,
  Refresh,
  Connection,
  User,
  Warning
} from '@element-plus/icons-vue'
import { getSessionDetails, terminateSession } from '@/api/nfcRelayAdmin'
import { formatTime } from '@/utils/format'

const props = defineProps({
  visible: {
    type: Boolean,
    default: false
  },
  sessionId: {
    type: String,
    default: ''
  }
})

const emit = defineEmits(['update:visible'])

// 响应式数据
const loading = ref(false)
const terminating = ref(false)
const sessionData = ref(null)

// 计算属性
const visible = computed({
  get: () => props.visible,
  set: (val) => emit('update:visible', val)
})

// 工具函数
const getStatusTagType = (status) => {
  switch (status) {
    case 'paired': return 'success'
    case 'waiting_for_pairing': return 'warning'
    default: return 'info'
  }
}

const getStatusIcon = (status) => {
  switch (status) {
    case 'paired': return Link
    case 'waiting_for_pairing': return Clock
    default: return CircleCheck
  }
}

const getStatusText = (status) => {
  switch (status) {
    case 'paired': return '已配对'
    case 'waiting_for_pairing': return '等待配对'
    default: return status
  }
}

const getStatusColor = (status) => {
  switch (status) {
    case 'paired': return '#67c23a'
    case 'waiting_for_pairing': return '#e6a23c'
    default: return '#909399'
  }
}

const getEventIcon = (event) => {
  switch (event) {
    case 'SessionCreated': return Connection
    case 'ProviderJoined': return Phone
    case 'ReceiverJoined': return Monitor
    case 'SessionPaired': return Link
    case 'SessionTerminatedByClientRequest': return Close
    default: return CircleCheck
  }
}

const getEventIconColor = (event) => {
  switch (event) {
    case 'SessionCreated': return '#409eff'
    case 'ProviderJoined': return '#67c23a'
    case 'ReceiverJoined': return '#e6a23c'
    case 'SessionPaired': return '#67c23a'
    case 'SessionTerminatedByClientRequest': return '#f56c6c'
    default: return '#909399'
  }
}

const getEventText = (event) => {
  switch (event) {
    case 'SessionCreated': return '会话创建'
    case 'ProviderJoined': return 'Provider加入'
    case 'ReceiverJoined': return 'Receiver加入'
    case 'SessionPaired': return '会话配对成功'
    case 'SessionTerminatedByClientRequest': return '客户端请求终止'
    default: return event
  }
}

// 方法
const fetchData = async () => {
  if (!props.sessionId) return
  
  try {
    loading.value = true
    const res = await getSessionDetails(props.sessionId)
    if (res.code === 0) {
      sessionData.value = res.data
    } else {
      ElMessage.error(res.msg || '获取会话详情失败')
      // 生成模拟数据
      sessionData.value = generateMockSessionDetail()
    }
  } catch (error) {
    console.error('Failed to fetch session details:', error)
    ElMessage.error('获取会话详情失败')
    // 生成模拟数据
    sessionData.value = generateMockSessionDetail()
  } finally {
    loading.value = false
  }
}

const handleTerminate = async () => {
  try {
    await ElMessageBox.confirm(
      '确定要强制终止此会话吗？',
      '确认操作',
      {
        confirmButtonText: '确定',
        cancelButtonText: '取消',
        type: 'warning'
      }
    )
    
    terminating.value = true
    const res = await terminateSession(props.sessionId)
    if (res.code === 0) {
      ElMessage.success('会话已终止')
      handleClose()
    } else {
      ElMessage.error(res.msg || '终止会话失败')
    }
  } catch (error) {
    if (error !== 'cancel') {
      console.error('Failed to terminate session:', error)
      ElMessage.error('终止会话失败')
    }
  } finally {
    terminating.value = false
  }
}

const handleClose = () => {
  visible.value = false
  sessionData.value = null
}

const refreshData = () => {
  fetchData()
}

const generateMockSessionDetail = () => {
  return {
    session_id: props.sessionId,
    status: 'paired',
    created_at: new Date(Date.now() - 7200000).toISOString(),
    last_activity_at: new Date(Date.now() - 300000).toISOString(),
    terminated_at: null,
    termination_reason: null,
    provider_info: {
      client_id: 'provider-client-123',
      user_id: 'provider-user-456',
      display_name: 'iPhone 13 Pro',
      ip_address: '192.168.1.100'
    },
    receiver_info: {
      client_id: 'receiver-client-789',
      user_id: 'receiver-user-012',
      display_name: 'POS Terminal A',
      ip_address: '192.168.1.200'
    },
    apdu_exchange_count: {
      upstream: 25,
      downstream: 23
    },
    session_events_history: [
      {
        timestamp: new Date(Date.now() - 7200000).toISOString(),
        event: 'SessionCreated'
      },
      {
        timestamp: new Date(Date.now() - 7180000).toISOString(),
        event: 'ProviderJoined',
        client_id: 'provider-client-123'
      },
      {
        timestamp: new Date(Date.now() - 7160000).toISOString(),
        event: 'ReceiverJoined',
        client_id: 'receiver-client-789'
      },
      {
        timestamp: new Date(Date.now() - 7140000).toISOString(),
        event: 'SessionPaired'
      }
    ],
    related_audit_logs_summary: [
      {
        timestamp: new Date(Date.now() - 1800000).toISOString(),
        event_type: 'apdu_relayed_success',
        details_summary: 'APDU转发成功，长度24字节，从receiver到provider'
      },
      {
        timestamp: new Date(Date.now() - 900000).toISOString(),
        event_type: 'apdu_relayed_success',
        details_summary: 'APDU转发成功，长度16字节，从provider到receiver'
      }
    ]
  }
}

// 监听器
watch(() => props.visible, (newVal) => {
  if (newVal && props.sessionId) {
    fetchData()
  }
})
</script>

<style scoped lang="scss">
.session-detail-content {
  .mb-4 {
    margin-bottom: 1rem;
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
  
  .text-xs {
    font-size: 0.75rem;
  }
  
  .text-gray-400 {
    color: #c0c4cc;
  }
  
  .text-gray-500 {
    color: #909399;
  }
  
  .flex {
    display: flex;
    
    &.items-center {
      align-items: center;
    }
    
    &.justify-between {
      justify-content: space-between;
    }
  }
  
  .mr-2 {
    margin-right: 0.5rem;
  }
  
  .grid {
    display: grid;
    
    &.grid-cols-2 {
      grid-template-columns: repeat(2, minmax(0, 1fr));
    }
  }
  
  .gap-4 {
    gap: 1rem;
  }
  
  .stat-item {
    text-align: center;
    padding: 16px;
    background-color: #f9f9f9;
    border-radius: 8px;
    
    .stat-label {
      font-size: 14px;
      color: #606266;
      margin-bottom: 8px;
      font-weight: 500;
    }
    
    .stat-value {
      font-size: 24px;
      font-weight: bold;
      color: #303133;
      margin-bottom: 4px;
    }
    
    .stat-desc {
      font-size: 12px;
      color: #909399;
    }
  }
  
  .timeline-card {
    :deep(.el-card__body) {
      padding: 8px 12px;
    }
  }
  
  .text-center {
    text-align: center;
  }
  
  .py-8 {
    padding-top: 2rem;
    padding-bottom: 2rem;
  }
}
</style> 