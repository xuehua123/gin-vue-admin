<!--
  客户端详情弹窗组件
  显示客户端的详细信息和连接历史
-->
<template>
  <el-dialog
    v-model="visible"
    title="客户端详细信息"
    width="800px"
    :before-close="handleClose"
    destroy-on-close
  >
    <div v-if="loading" v-loading="loading" style="height: 400px;"></div>
    
    <div v-else-if="clientData" class="client-detail-content">
      <!-- 基本信息 -->
      <el-card shadow="never" class="mb-4">
        <template #header>
          <div class="flex items-center">
            <el-icon class="mr-2" :color="getStatusColor(clientData.is_online)">
              <CircleCheck v-if="clientData.is_online" />
              <CircleClose v-else />
            </el-icon>
            <span class="font-semibold">基本信息</span>
          </div>
        </template>
        
        <el-descriptions :column="2" border>
          <el-descriptions-item label="客户端ID">
            <code class="font-mono text-sm">{{ clientData.client_id }}</code>
          </el-descriptions-item>
          
          <el-descriptions-item label="用户ID">
            {{ clientData.user_id }}
          </el-descriptions-item>
          
          <el-descriptions-item label="显示名称">
            <div class="flex items-center">
              <el-icon class="mr-1" :color="getDeviceIconColor(clientData.role)">
                <component :is="getDeviceIcon(clientData.role)" />
              </el-icon>
              {{ clientData.display_name || '-' }}
            </div>
          </el-descriptions-item>
          
          <el-descriptions-item label="角色">
            <el-tag :type="getRoleTagType(clientData.role)" size="small">
              {{ getRoleText(clientData.role) }}
            </el-tag>
          </el-descriptions-item>
          
          <el-descriptions-item label="IP地址">
            {{ clientData.ip_address }}
          </el-descriptions-item>
          
          <el-descriptions-item label="User Agent">
            <el-tooltip :content="clientData.user_agent || '未知'" placement="top">
              <div class="text-truncate max-w-200px">
                {{ clientData.user_agent || '未知' }}
              </div>
            </el-tooltip>
          </el-descriptions-item>
          
          <el-descriptions-item label="连接时间">
            {{ formatTime(clientData.connected_at) }}
          </el-descriptions-item>
          
          <el-descriptions-item label="最后消息时间">
            {{ formatTime(clientData.last_message_at) }}
          </el-descriptions-item>
          
          <el-descriptions-item label="在线状态">
            <el-tag :type="clientData.is_online ? 'success' : 'info'" size="small">
              {{ clientData.is_online ? '在线' : '离线' }}
            </el-tag>
          </el-descriptions-item>
          
          <el-descriptions-item label="当前会话">
            <el-link 
              v-if="clientData.session_id" 
              type="warning"
              @click="$router.push(`/nfc-relay-admin/sessions?sessionID=${clientData.session_id}`)"
            >
              {{ clientData.session_id }}
            </el-link>
            <span v-else class="text-gray-400">无</span>
          </el-descriptions-item>
        </el-descriptions>
      </el-card>

      <!-- 统计信息 -->
      <el-card shadow="never" class="mb-4">
        <template #header>
          <span class="font-semibold">统计信息</span>
        </template>
        
        <div class="grid grid-cols-2 gap-4">
          <div class="stat-item">
            <div class="stat-label">发送消息数</div>
            <div class="stat-value">{{ clientData.sent_message_count || 0 }}</div>
          </div>
          <div class="stat-item">
            <div class="stat-label">接收消息数</div>
            <div class="stat-value">{{ clientData.received_message_count || 0 }}</div>
          </div>
        </div>
      </el-card>

      <!-- 连接事件历史 -->
      <el-card shadow="never" class="mb-4" v-if="clientData.connection_events && clientData.connection_events.length">
        <template #header>
          <span class="font-semibold">连接事件</span>
        </template>
        
        <el-timeline>
          <el-timeline-item
            v-for="(event, index) in clientData.connection_events"
            :key="index"
            :timestamp="formatTime(event.timestamp)"
            placement="top"
          >
            <el-card shadow="never" class="timeline-card">
              <div class="flex items-center">
                <el-icon class="mr-2" :color="getEventIconColor(event.event)">
                  <component :is="getEventIcon(event.event)" />
                </el-icon>
                <span>{{ getEventText(event.event) }}</span>
              </div>
            </el-card>
          </el-timeline-item>
        </el-timeline>
      </el-card>

      <!-- 相关审计日志摘要 -->
      <el-card shadow="never" v-if="clientData.related_audit_logs_summary && clientData.related_audit_logs_summary.length">
        <template #header>
          <div class="flex justify-between items-center">
            <span class="font-semibold">最近审计日志</span>
            <el-button 
              link 
              type="primary" 
              @click="$router.push(`/nfc-relay-admin/audit-logs?clientID=${clientData.client_id}`)"
            >
              查看全部
            </el-button>
          </div>
        </template>
        
        <el-table :data="clientData.related_audit_logs_summary" size="small">
          <el-table-column prop="timestamp" label="时间" width="180">
            <template #default="{ row }">
              {{ formatTime(row.timestamp) }}
            </template>
          </el-table-column>
          <el-table-column prop="event_type" label="事件类型" width="150" />
          <el-table-column prop="details_summary" label="详情" />
        </el-table>
      </el-card>
    </div>

    <div v-else class="text-center py-8 text-gray-500">
      获取客户端详情失败
    </div>

    <template #footer>
      <div class="flex justify-between">
        <div>
          <el-button 
            v-if="clientData && clientData.is_online" 
            type="danger" 
            @click="handleDisconnect"
            :loading="disconnecting"
          >
            <el-icon><Close /></el-icon>
            强制断开连接
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
  CircleCheck, 
  CircleClose, 
  Phone, 
  Monitor, 
  QuestionFilled,
  Connection,
  User,
  Link,
  Close,
  Refresh,
  Warning
} from '@element-plus/icons-vue'
import { getClientDetails, disconnectClient } from '@/api/nfcRelayAdmin'
import { formatTime } from '@/utils/format'

const props = defineProps({
  visible: {
    type: Boolean,
    default: false
  },
  clientId: {
    type: String,
    default: ''
  }
})

const emit = defineEmits(['update:visible'])

// 响应式数据
const loading = ref(false)
const disconnecting = ref(false)
const clientData = ref(null)

// 计算属性
const visible = computed({
  get: () => props.visible,
  set: (val) => emit('update:visible', val)
})

// 工具函数
const getRoleTagType = (role) => {
  switch (role) {
    case 'provider': return 'success'
    case 'receiver': return 'warning'
    default: return 'info'
  }
}

const getRoleText = (role) => {
  switch (role) {
    case 'provider': return 'Provider'
    case 'receiver': return 'Receiver'
    default: return 'None'
  }
}

const getDeviceIcon = (role) => {
  switch (role) {
    case 'provider': return Phone
    case 'receiver': return Monitor
    default: return QuestionFilled
  }
}

const getDeviceIconColor = (role) => {
  switch (role) {
    case 'provider': return '#67c23a'
    case 'receiver': return '#e6a23c'
    default: return '#909399'
  }
}

const getStatusColor = (isOnline) => {
  return isOnline ? '#67c23a' : '#f56c6c'
}

const getEventIcon = (event) => {
  switch (event) {
    case 'Connected': return Connection
    case 'Authenticated': return User
    case 'RoleDeclared': return Link
    default: return QuestionFilled
  }
}

const getEventIconColor = (event) => {
  switch (event) {
    case 'Connected': return '#67c23a'
    case 'Authenticated': return '#409eff'
    case 'RoleDeclared': return '#e6a23c'
    default: return '#909399'
  }
}

const getEventText = (event) => {
  switch (event) {
    case 'Connected': return '已连接'
    case 'Authenticated': return '已认证'
    case 'RoleDeclared': return '角色声明'
    default: return event
  }
}

// 方法
const fetchData = async () => {
  if (!props.clientId) return
  
  try {
    loading.value = true
    const res = await getClientDetails(props.clientId)
    if (res.code === 0) {
      clientData.value = res.data
    } else {
      ElMessage.error(res.msg || '获取客户端详情失败')
      // 生成模拟数据
      clientData.value = generateMockClientDetail()
    }
  } catch (error) {
    console.error('Failed to fetch client details:', error)
    ElMessage.error('获取客户端详情失败')
    // 生成模拟数据
    clientData.value = generateMockClientDetail()
  } finally {
    loading.value = false
  }
}

const handleDisconnect = async () => {
  try {
    await ElMessageBox.confirm(
      '确定要强制断开此客户端的连接吗？',
      '确认操作',
      {
        confirmButtonText: '确定',
        cancelButtonText: '取消',
        type: 'warning'
      }
    )
    
    disconnecting.value = true
    const res = await disconnectClient(props.clientId)
    if (res.code === 0) {
      ElMessage.success('客户端已断开连接')
      handleClose()
    } else {
      ElMessage.error(res.msg || '断开客户端失败')
    }
  } catch (error) {
    if (error !== 'cancel') {
      console.error('Failed to disconnect client:', error)
      ElMessage.error('断开客户端失败')
    }
  } finally {
    disconnecting.value = false
  }
}

const handleClose = () => {
  visible.value = false
  clientData.value = null
}

const refreshData = () => {
  fetchData()
}

const generateMockClientDetail = () => {
  return {
    client_id: props.clientId,
    user_id: 'user-' + Math.floor(Math.random() * 1000),
    display_name: 'iPhone 13 Pro',
    role: 'provider',
    ip_address: '192.168.1.100',
    user_agent: 'Mozilla/5.0 (iPhone; CPU iPhone OS 15_0 like Mac OS X) AppleWebKit/605.1.15',
    connected_at: new Date(Date.now() - 3600000).toISOString(),
    last_message_at: new Date(Date.now() - 300000).toISOString(),
    is_online: true,
    session_id: 'session-' + Date.now(),
    sent_message_count: 45,
    received_message_count: 38,
    connection_events: [
      {
        timestamp: new Date(Date.now() - 3600000).toISOString(),
        event: 'Connected'
      },
      {
        timestamp: new Date(Date.now() - 3590000).toISOString(),
        event: 'Authenticated'
      },
      {
        timestamp: new Date(Date.now() - 3580000).toISOString(),
        event: 'RoleDeclared'
      }
    ],
    related_audit_logs_summary: [
      {
        timestamp: new Date(Date.now() - 600000).toISOString(),
        event_type: 'apdu_relayed_success',
        details_summary: 'APDU转发成功，长度32字节'
      },
      {
        timestamp: new Date(Date.now() - 300000).toISOString(),
        event_type: 'session_joined',
        details_summary: '加入NFC中继会话'
      }
    ]
  }
}

// 监听器
watch(() => props.visible, (newVal) => {
  if (newVal && props.clientId) {
    fetchData()
  }
})
</script>

<style scoped lang="scss">
.client-detail-content {
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
  
  .text-gray-400 {
    color: #c0c4cc;
  }
  
  .text-gray-500 {
    color: #909399;
  }
  
  .text-truncate {
    overflow: hidden;
    text-overflow: ellipsis;
    white-space: nowrap;
  }
  
  .max-w-200px {
    max-width: 200px;
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
  
  .mr-1 {
    margin-right: 0.25rem;
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
    }
    
    .stat-value {
      font-size: 24px;
      font-weight: bold;
      color: #303133;
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