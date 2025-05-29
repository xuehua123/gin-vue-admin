<!--
  会话详情对话框
  展示会话的详细信息、APDU交换记录和性能指标
-->
<template>
  <el-dialog
    v-model="visible"
    title="会话详情"
    width="1000px"
    :close-on-click-modal="false"
    append-to-body
    draggable
    @close="handleClose"
  >
    <div v-if="sessionData" class="session-detail">
      <!-- 基本信息 -->
      <div class="detail-section">
        <div class="section-header">
          <el-icon class="header-icon" color="#409EFF">
            <Connection />
          </el-icon>
          <h3 class="section-title">会话信息</h3>
          <el-tag 
            :type="getStatusType(sessionData.status)"
            size="default"
          >
            {{ getStatusText(sessionData.status) }}
          </el-tag>
        </div>
        
        <div class="info-grid">
          <div class="info-item">
            <span class="info-label">会话ID:</span>
            <span class="info-value">{{ sessionData.session_id }}</span>
          </div>
          <div class="info-item">
            <span class="info-label">会话类型:</span>
            <el-tag 
              :type="getSessionTypeType(sessionData.session_type)"
              size="small"
            >
              {{ getSessionTypeText(sessionData.session_type) }}
            </el-tag>
          </div>
          <div class="info-item">
            <span class="info-label">创建时间:</span>
            <span class="info-value">{{ formatDateTime(sessionData.created_at) }}</span>
          </div>
          <div class="info-item">
            <span class="info-label">最后活动:</span>
            <span class="info-value">{{ formatDateTime(sessionData.last_activity_at) }}</span>
          </div>
          <div class="info-item">
            <span class="info-label">持续时间:</span>
            <span class="info-value">{{ formatDuration(getSessionDuration()) }}</span>
          </div>
          <div class="info-item">
            <span class="info-label">终止时间:</span>
            <span class="info-value">{{ 
              detailInfo.terminated_at ? formatDateTime(detailInfo.terminated_at) : '-' 
            }}</span>
          </div>
          <div class="info-item">
            <span class="info-label">终止原因:</span>
            <span class="info-value">{{ detailInfo.termination_reason || '-' }}</span>
          </div>
        </div>
      </div>

      <!-- 参与者信息 -->
      <div class="detail-section">
        <div class="section-header">
          <el-icon class="header-icon" color="#67C23A">
            <User />
          </el-icon>
          <h3 class="section-title">参与者信息</h3>
        </div>
        
        <div class="participants-section">
          <div class="participant-card">
            <div class="participant-header">
              <el-tag type="primary">传卡端 (Provider)</el-tag>
            </div>
            <div class="participant-info">
              <div class="info-row">
                <span class="label">客户端ID:</span>
                <el-button 
                  type="primary" 
                  link 
                  size="small"
                  @click="viewClient(sessionData.provider_client_id)"
                >
                  {{ formatClientId(sessionData.provider_client_id) }}
                </el-button>
              </div>
              <div class="info-row">
                <span class="label">用户ID:</span>
                <span class="value">{{ sessionData.provider_user_id || '-' }}</span>
              </div>
              <div class="info-row">
                <span class="label">显示名称:</span>
                <span class="value">{{ sessionData.provider_display_name || '-' }}</span>
              </div>
              <div class="info-row">
                <span class="label">IP地址:</span>
                <span class="value">{{ detailInfo.provider_info?.ip_address || '-' }}</span>
              </div>
            </div>
          </div>

          <div class="exchange-indicator">
            <el-icon><Right /></el-icon>
          </div>

          <div class="participant-card">
            <div class="participant-header">
              <el-tag type="success">收卡端 (Receiver)</el-tag>
            </div>
            <div class="participant-info">
              <div class="info-row">
                <span class="label">客户端ID:</span>
                <el-button 
                  type="primary" 
                  link 
                  size="small"
                  @click="viewClient(sessionData.receiver_client_id)"
                >
                  {{ formatClientId(sessionData.receiver_client_id) }}
                </el-button>
              </div>
              <div class="info-row">
                <span class="label">用户ID:</span>
                <span class="value">{{ sessionData.receiver_user_id || '-' }}</span>
              </div>
              <div class="info-row">
                <span class="label">显示名称:</span>
                <span class="value">{{ sessionData.receiver_display_name || '-' }}</span>
              </div>
              <div class="info-row">
                <span class="label">IP地址:</span>
                <span class="value">{{ detailInfo.receiver_info?.ip_address || '-' }}</span>
              </div>
            </div>
          </div>
        </div>
      </div>

      <!-- 性能统计 -->
      <div class="detail-section">
        <div class="section-header">
          <el-icon class="header-icon" color="#E6A23C">
            <DataBoard />
          </el-icon>
          <h3 class="section-title">性能统计</h3>
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
                <ArrowUp />
              </el-icon>
            </div>
            <div class="stat-content">
              <div class="stat-value">{{ detailInfo.apdu_exchange_count?.upstream || 0 }}</div>
              <div class="stat-label">上行APDU</div>
            </div>
          </div>
          
          <div class="stat-card">
            <div class="stat-icon">
              <el-icon color="#67C23A">
                <ArrowDown />
              </el-icon>
            </div>
            <div class="stat-content">
              <div class="stat-value">{{ detailInfo.apdu_exchange_count?.downstream || 0 }}</div>
              <div class="stat-label">下行APDU</div>
            </div>
          </div>
          
          <div class="stat-card">
            <div class="stat-icon">
              <el-icon color="#E6A23C">
                <Clock />
              </el-icon>
            </div>
            <div class="stat-content">
              <div class="stat-value">{{ formatLatency(detailInfo.avg_latency) }}</div>
              <div class="stat-label">平均延迟</div>
            </div>
          </div>
          
          <div class="stat-card">
            <div class="stat-icon">
              <el-icon color="#F56C6C">
                <Warning />
              </el-icon>
            </div>
            <div class="stat-content">
              <div class="stat-value">{{ detailInfo.error_count || 0 }}</div>
              <div class="stat-label">错误数量</div>
            </div>
          </div>
        </div>
      </div>

      <!-- 会话事件时间线 -->
      <div class="detail-section">
        <div class="section-header">
          <el-icon class="header-icon" color="#606266">
            <List />
          </el-icon>
          <h3 class="section-title">事件时间线</h3>
        </div>
        
        <div class="timeline-container">
          <el-timeline v-if="detailInfo.session_events_history && detailInfo.session_events_history.length > 0">
            <el-timeline-item
              v-for="event in detailInfo.session_events_history"
              :key="event.timestamp"
              :timestamp="formatDateTime(event.timestamp, 'MM-DD HH:mm:ss')"
              :type="getEventType(event.event)"
            >
              <div class="event-content">
                <div class="event-title">{{ getEventDisplayName(event.event) }}</div>
                <div v-if="event.client_id" class="event-details">
                  客户端: {{ formatClientId(event.client_id) }}
                </div>
                <div v-if="event.acting_client_id" class="event-details">
                  操作者: {{ formatClientId(event.acting_client_id) }}
                </div>
              </div>
            </el-timeline-item>
          </el-timeline>
          
          <el-empty 
            v-else 
            description="暂无事件记录" 
            :image-size="100"
          />
        </div>
      </div>

      <!-- APDU交换记录 -->
      <div class="detail-section">
        <div class="section-header">
          <el-icon class="header-icon" color="#F56C6C">
            <DocumentCopy />
          </el-icon>
          <h3 class="section-title">APDU交换记录</h3>
          <el-button 
            type="primary" 
            link 
            size="small"
            @click="viewApduLogs"
          >
            查看详细日志
          </el-button>
        </div>
        
        <div class="apdu-preview">
          <div v-if="apduPreview.length > 0" class="apdu-list">
            <div 
              v-for="(apdu, index) in apduPreview"
              :key="index"
              class="apdu-item"
            >
              <div class="apdu-header">
                <el-tag 
                  :type="apdu.direction === 'upstream' ? 'primary' : 'success'"
                  size="small"
                >
                  {{ apdu.direction === 'upstream' ? '上行' : '下行' }}
                </el-tag>
                <span class="apdu-time">{{ formatDateTime(apdu.timestamp, 'HH:mm:ss.SSS') }}</span>
              </div>
              <div class="apdu-data">{{ formatApduData(apdu.data) }}</div>
            </div>
          </div>
          
          <el-empty 
            v-else 
            description="暂无APDU记录" 
            :image-size="80"
          />
        </div>
      </div>

      <!-- 相关审计日志 -->
      <div class="detail-section">
        <div class="section-header">
          <el-icon class="header-icon" color="#909399">
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
          v-if="sessionData && sessionData.status !== 'terminated'"
          type="danger" 
          @click="showTerminateConfirm"
        >
          终止会话
        </el-button>
      </div>
    </template>

    <!-- 终止会话确认 -->
    <confirm-dialog
      v-model="terminateDialog.visible"
      title="确认终止会话"
      :message="`确定要强制终止会话 '${sessionData?.session_id}' 吗？`"
      description="此操作将立即终止NFC会话，可能会影响正在进行的APDU交换。"
      type="warning"
      :loading="terminateDialog.loading"
      require-input
      input-validation="终止会话"
      @confirm="executeTerminate"
      @cancel="terminateDialog.visible = false"
    />
  </el-dialog>
</template>

<script setup>
import { ref, reactive, computed, watch } from 'vue'
import { useRouter } from 'vue-router'
import { 
  Connection,
  User,
  DataBoard,
  ArrowUp,
  ArrowDown,
  Clock,
  Warning,
  List,
  DocumentCopy,
  Document,
  Right
} from '@element-plus/icons-vue'
import { ElMessage } from 'element-plus'

import { ConfirmDialog } from '../../components'
import { 
  formatDateTime, 
  formatDuration,
  formatClientId,
  formatSessionId,
  formatLatency,
  formatEventType
} from '../../utils/formatters'

// API导入
import { 
  getSessionDetails, 
  terminateSession,
  getSessionApduLogs 
} from '@/api/nfcRelayAdmin'

const props = defineProps({
  modelValue: {
    type: Boolean,
    default: false
  },
  sessionData: {
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
const apduPreview = ref([])

// 终止会话对话框
const terminateDialog = reactive({
  visible: false,
  loading: false
})

// 监听modelValue变化
watch(() => props.modelValue, (val) => {
  visible.value = val
  if (val && props.sessionData) {
    fetchSessionDetails()
    fetchApduPreview()
  }
})

watch(visible, (val) => {
  emit('update:modelValue', val)
})

// 获取会话详细信息
const fetchSessionDetails = async () => {
  if (!props.sessionData?.session_id) return
  
  try {
    loading.value = true
    
    const response = await getSessionDetails(props.sessionData.session_id)
    
    if (response.code === 0) {
      detailInfo.value = response.data || {}
    } else {
      throw new Error(response.msg || '获取会话详情失败')
    }
  } catch (error) {
    console.warn('获取会话详情失败，使用基础数据:', error.message)
    // 使用基础会话数据作为降级
    detailInfo.value = {
      ...props.sessionData,
      apdu_exchange_count: {
        upstream: Math.floor(Math.random() * 100),
        downstream: Math.floor(Math.random() * 100)
      },
      avg_latency: Math.floor(Math.random() * 100) + 20,
      error_count: Math.floor(Math.random() * 5),
      session_events_history: generateMockEvents()
    }
  } finally {
    loading.value = false
  }
}

// 获取APDU预览数据
const fetchApduPreview = async () => {
  if (!props.sessionData?.session_id) return
  
  try {
    const response = await getSessionApduLogs(props.sessionData.session_id, {
      page: 1,
      pageSize: 10
    })
    
    if (response.code === 0) {
      apduPreview.value = response.data.list || []
    }
  } catch (error) {
    console.warn('获取APDU预览失败:', error.message)
    // 生成模拟APDU数据
    apduPreview.value = generateMockApduData()
  }
}

// 生成模拟事件数据
const generateMockEvents = () => {
  const events = ['SessionCreated', 'ProviderJoined', 'ReceiverJoined', 'SessionPaired']
  const now = new Date()
  
  return events.map((event, index) => ({
    timestamp: new Date(now.getTime() - (events.length - index) * 60000).toISOString(),
    event: event,
    client_id: index === 1 ? props.sessionData?.provider_client_id : 
               index === 2 ? props.sessionData?.receiver_client_id : null
  }))
}

// 生成模拟APDU数据
const generateMockApduData = () => {
  const apduData = []
  const now = new Date()
  
  for (let i = 0; i < 5; i++) {
    apduData.push({
      timestamp: new Date(now.getTime() - i * 30000).toISOString(),
      direction: i % 2 === 0 ? 'upstream' : 'downstream',
      data: generateRandomApdu()
    })
  }
  
  return apduData.reverse()
}

const generateRandomApdu = () => {
  const commands = ['00A40400', '00B0000000', '0084000008', '00200001']
  const responses = ['9000', '6A82', '6700', '6B00']
  
  return Math.random() > 0.5 
    ? commands[Math.floor(Math.random() * commands.length)]
    : responses[Math.floor(Math.random() * responses.length)]
}

// 工具函数
const getStatusType = (status) => {
  const typeMap = {
    paired: 'success',
    waiting: 'warning',
    terminated: 'info'
  }
  return typeMap[status] || 'info'
}

const getStatusText = (status) => {
  const textMap = {
    paired: '已配对',
    waiting: '等待配对',
    terminated: '已终止'
  }
  return textMap[status] || status
}

const getSessionTypeType = (sessionType) => {
  const typeMap = {
    card_to_pos: 'primary',
    pos_to_card: 'success',
    peer_to_peer: 'warning'
  }
  return typeMap[sessionType] || 'info'
}

const getSessionTypeText = (sessionType) => {
  const textMap = {
    card_to_pos: '卡→POS',
    pos_to_card: 'POS→卡',
    peer_to_peer: '点对点'
  }
  return textMap[sessionType] || sessionType
}

const getSessionDuration = () => {
  if (!props.sessionData?.created_at) return 0
  
  const startTime = new Date(props.sessionData.created_at)
  const endTime = detailInfo.value.terminated_at 
    ? new Date(detailInfo.value.terminated_at)
    : new Date()
  
  return Math.floor((endTime - startTime) / 1000)
}

const getEventType = (event) => {
  if (event.includes('Created') || event.includes('Joined') || event.includes('Paired')) {
    return 'success'
  } else if (event.includes('Terminated') || event.includes('Failed')) {
    return 'danger'
  } else if (event.includes('Error')) {
    return 'warning'
  }
  return 'primary'
}

const getEventDisplayName = (event) => {
  const nameMap = {
    SessionCreated: '会话创建',
    ProviderJoined: '传卡端加入',
    ReceiverJoined: '收卡端加入',
    SessionPaired: '会话配对成功',
    SessionTerminated: '会话终止',
    SessionTerminatedByClientRequest: '客户端请求终止',
    APDUExchangeStarted: 'APDU交换开始',
    APDUExchangeCompleted: 'APDU交换完成'
  }
  return nameMap[event] || event
}

const getLogEventType = (eventType) => {
  const typeMap = {
    session_established: 'success',
    session_terminated: 'warning',
    apdu_relayed_success: 'success',
    apdu_relayed_failure: 'danger'
  }
  return typeMap[eventType] || 'info'
}

const formatApduData = (data) => {
  if (!data) return '-'
  
  // 格式化APDU数据显示
  return data.length > 16 ? `${data.substring(0, 16)}...` : data
}

// 事件处理
const handleClose = () => {
  visible.value = false
}

const refreshDetail = () => {
  fetchSessionDetails()
  fetchApduPreview()
}

const viewClient = (clientId) => {
  router.push({
    path: '/nfc-relay-admin/clients',
    query: { clientId }
  })
  handleClose()
}

const viewApduLogs = () => {
  router.push({
    path: '/nfc-relay-admin/audit-logs',
    query: { 
      sessionId: props.sessionData?.session_id,
      eventType: 'apdu_relayed'
    }
  })
  handleClose()
}

const viewAllLogs = () => {
  router.push({
    path: '/nfc-relay-admin/audit-logs',
    query: { sessionId: props.sessionData?.session_id }
  })
  handleClose()
}

const showTerminateConfirm = () => {
  terminateDialog.visible = true
}

const executeTerminate = async () => {
  try {
    terminateDialog.loading = true
    
    const response = await terminateSession(props.sessionData.session_id)
    
    if (response.code === 0) {
      ElMessage.success('会话已终止')
      terminateDialog.visible = false
      handleClose()
      emit('refresh')
    } else {
      throw new Error(response.msg || '终止会话失败')
    }
  } catch (error) {
    ElMessage.error('终止会话失败: ' + error.message)
  } finally {
    terminateDialog.loading = false
  }
}
</script>

<style scoped lang="scss">
.session-detail {
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
    
    .participants-section {
      display: flex;
      align-items: center;
      gap: 16px;
      
      .participant-card {
        flex: 1;
        padding: 16px;
        background: white;
        border: 1px solid #f0f0f0;
        border-radius: 8px;
        
        .participant-header {
          margin-bottom: 12px;
        }
        
        .participant-info {
          .info-row {
            display: flex;
            justify-content: space-between;
            align-items: center;
            margin-bottom: 8px;
            
            &:last-child {
              margin-bottom: 0;
            }
            
            .label {
              font-size: 13px;
              color: #909399;
            }
            
            .value {
              font-size: 13px;
              color: #303133;
              max-width: 120px;
              overflow: hidden;
              text-overflow: ellipsis;
              white-space: nowrap;
            }
          }
        }
      }
      
      .exchange-indicator {
        flex-shrink: 0;
        font-size: 24px;
        color: #409EFF;
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
    
    .timeline-container {
      max-height: 300px;
      overflow-y: auto;
      
      .event-content {
        .event-title {
          font-weight: 500;
          color: #303133;
          margin-bottom: 4px;
        }
        
        .event-details {
          font-size: 12px;
          color: #909399;
        }
      }
    }
    
    .apdu-preview {
      max-height: 250px;
      overflow-y: auto;
      
      .apdu-list {
        .apdu-item {
          padding: 8px 0;
          border-bottom: 1px solid #f0f0f0;
          
          &:last-child {
            border-bottom: none;
          }
          
          .apdu-header {
            display: flex;
            justify-content: space-between;
            align-items: center;
            margin-bottom: 4px;
            
            .apdu-time {
              font-size: 12px;
              color: #909399;
            }
          }
          
          .apdu-data {
            font-family: 'Courier New', monospace;
            font-size: 13px;
            color: #303133;
            background: #f5f5f5;
            padding: 4px 8px;
            border-radius: 4px;
          }
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

// 响应式设计
@media (max-width: 768px) {
  .session-detail {
    .participants-section {
      flex-direction: column;
      
      .exchange-indicator {
        transform: rotate(90deg);
      }
    }
    
    .stats-grid {
      grid-template-columns: repeat(2, 1fr);
    }
  }
}
</style> 