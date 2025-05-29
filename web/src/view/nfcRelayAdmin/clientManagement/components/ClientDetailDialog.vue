<!--
  客户端详情对话框
  展示客户端的详细信息和统计数据
-->
<template>
  <el-dialog
    v-model="visible"
    title="客户端详情"
    width="900px"
    :close-on-click-modal="false"
    append-to-body
    draggable
    @close="handleClose"
  >
    <div v-if="clientData" class="client-detail">
      <!-- 基本信息 -->
      <div class="detail-section">
        <div class="section-header">
          <el-icon class="header-icon" color="#409EFF">
            <User />
          </el-icon>
          <h3 class="section-title">基本信息</h3>
          <el-tag 
            :type="getStatusType(clientData.is_online)"
            size="small"
          >
            {{ clientData.is_online ? '在线' : '离线' }}
          </el-tag>
        </div>
        
        <div class="info-grid">
          <div class="info-item">
            <span class="info-label">客户端ID:</span>
            <span class="info-value">{{ clientData.client_id }}</span>
          </div>
          <div class="info-item">
            <span class="info-label">用户ID:</span>
            <span class="info-value">{{ clientData.user_id || '-' }}</span>
          </div>
          <div class="info-item">
            <span class="info-label">显示名称:</span>
            <span class="info-value">{{ clientData.display_name || '-' }}</span>
          </div>
          <div class="info-item">
            <span class="info-label">角色:</span>
            <el-tag 
              :type="getRoleType(clientData.role)"
              size="small"
            >
              {{ getRoleText(clientData.role) }}
            </el-tag>
          </div>
          <div class="info-item">
            <span class="info-label">IP地址:</span>
            <span class="info-value">{{ formatIPAddress(clientData.ip_address) }}</span>
          </div>
          <div class="info-item">
            <span class="info-label">连接时间:</span>
            <span class="info-value">{{ formatDateTime(clientData.connected_at) }}</span>
          </div>
          <div class="info-item">
            <span class="info-label">当前会话:</span>
            <span class="info-value">
              <el-button 
                v-if="clientData.session_id"
                type="primary"
                link
                size="small"
                @click="viewSession(clientData.session_id)"
              >
                {{ formatSessionId(clientData.session_id) }}
              </el-button>
              <span v-else>-</span>
            </span>
          </div>
          <div class="info-item">
            <span class="info-label">用户代理:</span>
            <span class="info-value" :title="clientData.user_agent">
              {{ clientData.user_agent ? formatUserAgent(clientData.user_agent) : '-' }}
            </span>
          </div>
        </div>
      </div>

      <!-- 连接统计 -->
      <div class="detail-section">
        <div class="section-header">
          <el-icon class="header-icon" color="#67C23A">
            <DataBoard />
          </el-icon>
          <h3 class="section-title">连接统计</h3>
          <el-button 
            type="primary" 
            link 
            size="small"
            @click="refreshDetail"
            :loading="loading"
          >
            刷新
          </el-button>
        </div>
        
        <div class="stats-grid">
          <div class="stat-card">
            <div class="stat-icon">
              <el-icon color="#409EFF">
                <Message />
              </el-icon>
            </div>
            <div class="stat-content">
              <div class="stat-value">{{ detailInfo.sent_message_count || 0 }}</div>
              <div class="stat-label">已发送消息</div>
            </div>
          </div>
          
          <div class="stat-card">
            <div class="stat-icon">
              <el-icon color="#67C23A">
                <MessageBox />
              </el-icon>
            </div>
            <div class="stat-content">
              <div class="stat-value">{{ detailInfo.received_message_count || 0 }}</div>
              <div class="stat-label">已接收消息</div>
            </div>
          </div>
          
          <div class="stat-card">
            <div class="stat-icon">
              <el-icon color="#E6A23C">
                <Clock />
              </el-icon>
            </div>
            <div class="stat-content">
              <div class="stat-value">{{ formatDuration(getOnlineDuration()) }}</div>
              <div class="stat-label">在线时长</div>
            </div>
          </div>
          
          <div class="stat-card">
            <div class="stat-icon">
              <el-icon color="#F56C6C">
                <Calendar />
              </el-icon>
            </div>
            <div class="stat-content">
              <div class="stat-value">{{ formatRelativeTime(detailInfo.last_message_at) }}</div>
              <div class="stat-label">最后活动</div>
            </div>
          </div>
        </div>
      </div>

      <!-- 连接事件 -->
      <div class="detail-section">
        <div class="section-header">
          <el-icon class="header-icon" color="#E6A23C">
            <List />
          </el-icon>
          <h3 class="section-title">连接事件</h3>
        </div>
        
        <div class="events-container">
          <el-timeline v-if="detailInfo.connection_events && detailInfo.connection_events.length > 0">
            <el-timeline-item
              v-for="event in detailInfo.connection_events"
              :key="event.timestamp"
              :timestamp="formatDateTime(event.timestamp)"
              :type="getEventType(event.event)"
            >
              <div class="event-content">
                <span class="event-title">{{ event.event }}</span>
                <span v-if="event.details" class="event-details">{{ event.details }}</span>
              </div>
            </el-timeline-item>
          </el-timeline>
          
          <el-empty 
            v-else 
            description="暂无连接事件记录" 
            :image-size="100"
          />
        </div>
      </div>

      <!-- 相关审计日志 -->
      <div class="detail-section">
        <div class="section-header">
          <el-icon class="header-icon" color="#F56C6C">
            <Document />
          </el-icon>
          <h3 class="section-title">相关审计日志</h3>
          <el-button 
            type="primary" 
            link 
            size="small"
            @click="viewAllLogs"
          >
            查看全部
          </el-button>
        </div>
        
        <div class="logs-container">
          <div 
            v-if="detailInfo.related_audit_logs_summary && detailInfo.related_audit_logs_summary.length > 0"
            class="logs-list"
          >
            <div 
              v-for="log in detailInfo.related_audit_logs_summary.slice(0, 5)"
              :key="log.timestamp"
              class="log-item"
            >
              <div class="log-time">{{ formatDateTime(log.timestamp, 'MM-DD HH:mm:ss') }}</div>
              <div class="log-content">
                <el-tag 
                  :type="getLogEventType(log.event_type)"
                  size="small"
                >
                  {{ formatEventType(log.event_type).text }}
                </el-tag>
                <span class="log-details">{{ log.details_summary }}</span>
              </div>
            </div>
          </div>
          
          <el-empty 
            v-else 
            description="暂无相关日志" 
            :image-size="80"
          />
        </div>
      </div>
    </div>

    <template #footer>
      <div class="dialog-footer">
        <el-button @click="handleClose">关闭</el-button>
        <el-button 
          v-if="clientData && clientData.is_online"
          type="danger" 
          @click="showDisconnectConfirm"
        >
          断开连接
        </el-button>
      </div>
    </template>

    <!-- 断开连接确认 -->
    <confirm-dialog
      v-model="disconnectDialog.visible"
      title="确认断开连接"
      :message="`确定要强制断开客户端 '${clientData?.display_name || clientData?.client_id}' 的连接吗？`"
      description="此操作将立即关闭WebSocket连接，可能会影响正在进行的NFC会话。"
      type="warning"
      :loading="disconnectDialog.loading"
      require-input
      input-validation="断开连接"
      @confirm="executeDisconnect"
      @cancel="disconnectDialog.visible = false"
    />
  </el-dialog>
</template>

<script setup>
import { ref, reactive, computed, watch } from 'vue'
import { useRouter } from 'vue-router'
import { 
  User, 
  DataBoard, 
  Message, 
  MessageBox, 
  Clock, 
  Calendar,
  List,
  Document
} from '@element-plus/icons-vue'
import { ElMessage } from 'element-plus'

import { ConfirmDialog } from '../../components'
import { 
  formatDateTime, 
  formatRelativeTime,
  formatDuration,
  formatIPAddress, 
  formatSessionId,
  formatEventType
} from '../../utils/formatters'

// API导入
import { getClientDetails, disconnectClient } from '@/api/nfcRelayAdmin'

const props = defineProps({
  modelValue: {
    type: Boolean,
    default: false
  },
  clientData: {
    type: Object,
    default: null
  }
})

const emit = defineEmits(['update:modelValue', 'refresh'])

const router = useRouter()

// 状态管理
const visible = ref(false)
const loading = ref(false)
const detailInfo = ref({})

// 断开连接对话框
const disconnectDialog = reactive({
  visible: false,
  loading: false
})

// 监听modelValue变化
watch(() => props.modelValue, (val) => {
  visible.value = val
  if (val && props.clientData) {
    fetchClientDetails()
  }
})

watch(visible, (val) => {
  emit('update:modelValue', val)
})

// 获取客户端详细信息
const fetchClientDetails = async () => {
  if (!props.clientData?.client_id) return
  
  try {
    loading.value = true
    
    const response = await getClientDetails(props.clientData.client_id)
    
    if (response.code === 0) {
      detailInfo.value = response.data || {}
    } else {
      throw new Error(response.msg || '获取客户端详情失败')
    }
  } catch (error) {
    ElMessage.error('获取客户端详情失败: ' + error.message)
  } finally {
    loading.value = false
  }
}

// 工具函数
const getStatusType = (isOnline) => {
  return isOnline ? 'success' : 'danger'
}

const getRoleType = (role) => {
  const typeMap = {
    provider: 'primary',
    receiver: 'success',
    none: 'info'
  }
  return typeMap[role] || 'info'
}

const getRoleText = (role) => {
  const textMap = {
    provider: '传卡端',
    receiver: '收卡端',
    none: '未分配'
  }
  return textMap[role] || role
}

const formatUserAgent = (userAgent) => {
  if (!userAgent) return '-'
  
  // 简化用户代理字符串显示
  if (userAgent.length > 50) {
    return userAgent.substring(0, 50) + '...'
  }
  return userAgent
}

const getOnlineDuration = () => {
  if (!props.clientData?.connected_at) return 0
  
  const connectedTime = new Date(props.clientData.connected_at)
  const now = new Date()
  return Math.floor((now - connectedTime) / 1000)
}

const getEventType = (event) => {
  if (event.includes('Connected') || event.includes('Authenticated')) {
    return 'success'
  } else if (event.includes('Error') || event.includes('Failed')) {
    return 'danger'
  } else if (event.includes('Declared') || event.includes('Joined')) {
    return 'primary'
  }
  return 'info'
}

const getLogEventType = (eventType) => {
  const typeMap = {
    session_established: 'success',
    session_terminated: 'warning',
    apdu_relayed_success: 'success',
    apdu_relayed_failure: 'danger',
    client_connected: 'primary',
    client_disconnected: 'warning',
    auth_failure: 'danger'
  }
  return typeMap[eventType] || 'info'
}

// 事件处理
const handleClose = () => {
  visible.value = false
}

const refreshDetail = () => {
  fetchClientDetails()
}

const viewSession = (sessionId) => {
  router.push({
    path: '/nfc-relay-admin/sessions',
    query: { sessionId }
  })
  handleClose()
}

const viewAllLogs = () => {
  router.push({
    path: '/nfc-relay-admin/audit-logs',
    query: { clientId: props.clientData?.client_id }
  })
  handleClose()
}

const showDisconnectConfirm = () => {
  disconnectDialog.visible = true
}

const executeDisconnect = async () => {
  try {
    disconnectDialog.loading = true
    
    const response = await disconnectClient(props.clientData.client_id)
    
    if (response.code === 0) {
      ElMessage.success('客户端连接已断开')
      disconnectDialog.visible = false
      handleClose()
      emit('refresh')
    } else {
      throw new Error(response.msg || '断开连接失败')
    }
  } catch (error) {
    ElMessage.error('断开连接失败: ' + error.message)
  } finally {
    disconnectDialog.loading = false
  }
}
</script>

<style scoped lang="scss">
.client-detail {
  .detail-section {
    margin-bottom: 24px;
    
    &:last-child {
      margin-bottom: 0;
    }
    
    .section-header {
      display: flex;
      align-items: center;
      gap: 8px;
      margin-bottom: 16px;
      padding-bottom: 8px;
      border-bottom: 1px solid #f0f0f0;
      
      .header-icon {
        font-size: 18px;
      }
      
      .section-title {
        flex: 1;
        margin: 0;
        font-size: 16px;
        font-weight: 600;
        color: #303133;
      }
    }
    
    .info-grid {
      display: grid;
      grid-template-columns: repeat(auto-fit, minmax(300px, 1fr));
      gap: 16px;
      
      .info-item {
        display: flex;
        justify-content: space-between;
        align-items: center;
        padding: 12px;
        background: #f8f9fa;
        border-radius: 6px;
        
        .info-label {
          font-size: 14px;
          color: #606266;
          font-weight: 500;
        }
        
        .info-value {
          font-size: 14px;
          color: #303133;
          font-weight: 400;
          max-width: 200px;
          overflow: hidden;
          text-overflow: ellipsis;
          white-space: nowrap;
        }
      }
    }
    
    .stats-grid {
      display: grid;
      grid-template-columns: repeat(auto-fit, minmax(200px, 1fr));
      gap: 16px;
      
      .stat-card {
        display: flex;
        align-items: center;
        gap: 12px;
        padding: 16px;
        background: white;
        border: 1px solid #f0f0f0;
        border-radius: 8px;
        
        .stat-icon {
          flex-shrink: 0;
          width: 40px;
          height: 40px;
          display: flex;
          align-items: center;
          justify-content: center;
          background: #f8f9fa;
          border-radius: 50%;
          font-size: 18px;
        }
        
        .stat-content {
          .stat-value {
            font-size: 20px;
            font-weight: 600;
            color: #303133;
            margin-bottom: 4px;
          }
          
          .stat-label {
            font-size: 12px;
            color: #909399;
          }
        }
      }
    }
    
    .events-container {
      max-height: 300px;
      overflow-y: auto;
      
      .event-content {
        .event-title {
          font-weight: 500;
          color: #303133;
        }
        
        .event-details {
          margin-left: 8px;
          font-size: 12px;
          color: #909399;
        }
      }
    }
    
    .logs-container {
      max-height: 200px;
      overflow-y: auto;
      
      .logs-list {
        .log-item {
          display: flex;
          align-items: center;
          gap: 12px;
          padding: 8px 0;
          border-bottom: 1px solid #f0f0f0;
          
          &:last-child {
            border-bottom: none;
          }
          
          .log-time {
            flex-shrink: 0;
            font-size: 12px;
            color: #909399;
            width: 100px;
          }
          
          .log-content {
            flex: 1;
            display: flex;
            align-items: center;
            gap: 8px;
            
            .log-details {
              font-size: 13px;
              color: #606266;
            }
          }
        }
      }
    }
  }
}

.dialog-footer {
  display: flex;
  justify-content: flex-end;
  gap: 12px;
}
</style> 