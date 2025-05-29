<!--
  详情弹窗组件
  支持多种数据类型的详细信息展示
-->
<template>
  <el-dialog
    :model-value="visible"
    @update:model-value="(val) => emit('update:visible', val)"
    :title="modalTitle"
    :width="dialogWidth"
    :destroy-on-close="true"
    :close-on-click-modal="false"
    class="detail-modal"
  >
    <div class="detail-content" v-loading="loading">
      <!-- 客户端详情 -->
      <template v-if="type === 'client'">
        <div class="detail-header">
          <div class="header-left">
            <el-icon class="detail-icon" :color="getStatusColor(data.is_online)">
              <component :is="data.is_online ? CircleCheck : CircleClose" />
            </el-icon>
            <div class="header-info">
              <h3 class="detail-title">{{ data.display_name || data.client_id }}</h3>
              <p class="detail-subtitle">{{ data.user_id }}</p>
            </div>
          </div>
          <div class="header-right">
            <el-tag :type="getStatusTagType(data.is_online)">
              {{ data.is_online ? '在线' : '离线' }}
            </el-tag>
          </div>
        </div>
        
        <el-divider />
        
        <div class="detail-sections">
          <!-- 基本信息 -->
          <div class="detail-section">
            <h4 class="section-title">基本信息</h4>
            <el-descriptions :column="2" border>
              <el-descriptions-item label="客户端ID">
                <el-text type="primary" class="mono-text">{{ data.client_id }}</el-text>
              </el-descriptions-item>
              <el-descriptions-item label="用户ID">
                <el-text>{{ data.user_id }}</el-text>
              </el-descriptions-item>
              <el-descriptions-item label="显示名称">
                <el-text>{{ data.display_name || '未设置' }}</el-text>
              </el-descriptions-item>
              <el-descriptions-item label="角色">
                <el-tag :type="getRoleTagType(data.role)">
                  {{ getRoleText(data.role) }}
                </el-tag>
              </el-descriptions-item>
              <el-descriptions-item label="IP地址">
                <el-text class="mono-text">{{ data.ip_address }}</el-text>
              </el-descriptions-item>
              <el-descriptions-item label="用户代理">
                <el-text class="truncate-text">{{ data.user_agent || '未知' }}</el-text>
              </el-descriptions-item>
            </el-descriptions>
          </div>
          
          <!-- 连接信息 -->
          <div class="detail-section">
            <h4 class="section-title">连接信息</h4>
            <el-descriptions :column="2" border>
              <el-descriptions-item label="连接时间">
                <el-text>{{ formatTime(data.connected_at) }}</el-text>
              </el-descriptions-item>
              <el-descriptions-item label="最后消息">
                <el-text>{{ formatTime(data.last_message_at) || '无' }}</el-text>
              </el-descriptions-item>
              <el-descriptions-item label="发送消息数">
                <el-text type="success">{{ data.sent_message_count || 0 }}</el-text>
              </el-descriptions-item>
              <el-descriptions-item label="接收消息数">
                <el-text type="warning">{{ data.received_message_count || 0 }}</el-text>
              </el-descriptions-item>
              <el-descriptions-item label="当前会话">
                <el-text v-if="data.session_id" type="primary" class="mono-text">
                  {{ data.session_id }}
                </el-text>
                <el-text v-else type="info">无</el-text>
              </el-descriptions-item>
            </el-descriptions>
          </div>
          
          <!-- 连接事件历史 -->
          <div class="detail-section" v-if="data.connection_events">
            <h4 class="section-title">连接事件</h4>
            <el-timeline>
              <el-timeline-item
                v-for="event in data.connection_events"
                :key="event.timestamp"
                :timestamp="formatTime(event.timestamp)"
                placement="top"
              >
                <el-card>
                  <p>{{ event.event }}</p>
                </el-card>
              </el-timeline-item>
            </el-timeline>
          </div>
        </div>
      </template>
      
      <!-- 会话详情 -->
      <template v-else-if="type === 'session'">
        <div class="detail-header">
          <div class="header-left">
            <el-icon class="detail-icon" color="#409eff">
              <ChatDotRound />
            </el-icon>
            <div class="header-info">
              <h3 class="detail-title">会话详情</h3>
              <p class="detail-subtitle">{{ data.session_id }}</p>
            </div>
          </div>
          <div class="header-right">
            <el-tag :type="getSessionStatusTagType(data.status)">
              {{ getSessionStatusText(data.status) }}
            </el-tag>
          </div>
        </div>
        
        <el-divider />
        
        <div class="detail-sections">
          <!-- 会话信息 -->
          <div class="detail-section">
            <h4 class="section-title">会话信息</h4>
            <el-descriptions :column="2" border>
              <el-descriptions-item label="会话ID">
                <el-text type="primary" class="mono-text">{{ data.session_id }}</el-text>
              </el-descriptions-item>
              <el-descriptions-item label="状态">
                <el-tag :type="getSessionStatusTagType(data.status)">
                  {{ getSessionStatusText(data.status) }}
                </el-tag>
              </el-descriptions-item>
              <el-descriptions-item label="创建时间">
                <el-text>{{ formatTime(data.created_at) }}</el-text>
              </el-descriptions-item>
              <el-descriptions-item label="最后活动">
                <el-text>{{ formatTime(data.last_activity_at) }}</el-text>
              </el-descriptions-item>
              <el-descriptions-item label="终止时间" v-if="data.terminated_at">
                <el-text type="warning">{{ formatTime(data.terminated_at) }}</el-text>
              </el-descriptions-item>
              <el-descriptions-item label="终止原因" v-if="data.termination_reason">
                <el-text type="warning">{{ data.termination_reason }}</el-text>
              </el-descriptions-item>
            </el-descriptions>
          </div>
          
          <!-- 参与方信息 -->
          <div class="detail-section">
            <h4 class="section-title">参与方信息</h4>
            <div class="participants-grid">
              <!-- Provider信息 -->
              <el-card class="participant-card">
                <template #header>
                  <span class="participant-title">Provider (传卡方)</span>
                </template>
                <el-descriptions :column="1" size="small">
                  <el-descriptions-item label="客户端ID">
                    <el-text type="primary" class="mono-text">{{ data.provider_info?.client_id }}</el-text>
                  </el-descriptions-item>
                  <el-descriptions-item label="用户ID">
                    <el-text>{{ data.provider_info?.user_id }}</el-text>
                  </el-descriptions-item>
                  <el-descriptions-item label="显示名称">
                    <el-text>{{ data.provider_info?.display_name }}</el-text>
                  </el-descriptions-item>
                  <el-descriptions-item label="IP地址">
                    <el-text class="mono-text">{{ data.provider_info?.ip_address }}</el-text>
                  </el-descriptions-item>
                </el-descriptions>
              </el-card>
              
              <!-- Receiver信息 -->
              <el-card class="participant-card">
                <template #header>
                  <span class="participant-title">Receiver (收卡方)</span>
                </template>
                <el-descriptions :column="1" size="small">
                  <el-descriptions-item label="客户端ID">
                    <el-text type="primary" class="mono-text">{{ data.receiver_info?.client_id }}</el-text>
                  </el-descriptions-item>
                  <el-descriptions-item label="用户ID">
                    <el-text>{{ data.receiver_info?.user_id }}</el-text>
                  </el-descriptions-item>
                  <el-descriptions-item label="显示名称">
                    <el-text>{{ data.receiver_info?.display_name }}</el-text>
                  </el-descriptions-item>
                  <el-descriptions-item label="IP地址">
                    <el-text class="mono-text">{{ data.receiver_info?.ip_address }}</el-text>
                  </el-descriptions-item>
                </el-descriptions>
              </el-card>
            </div>
          </div>
          
          <!-- APDU交换统计 -->
          <div class="detail-section" v-if="data.apdu_exchange_count">
            <h4 class="section-title">APDU交换统计</h4>
            <div class="apdu-stats">
              <el-statistic title="上行消息" :value="data.apdu_exchange_count.upstream || 0" />
              <el-statistic title="下行消息" :value="data.apdu_exchange_count.downstream || 0" />
              <el-statistic 
                title="成功率" 
                :value="getSuccessRate(data.apdu_exchange_count)" 
                suffix="%" 
              />
            </div>
          </div>
          
          <!-- 会话事件历史 -->
          <div class="detail-section" v-if="data.session_events_history">
            <h4 class="section-title">会话事件</h4>
            <el-timeline>
              <el-timeline-item
                v-for="event in data.session_events_history"
                :key="event.timestamp"
                :timestamp="formatTime(event.timestamp)"
                placement="top"
              >
                <el-card>
                  <p>{{ event.event }}</p>
                  <p v-if="event.client_id" class="event-detail">
                    客户端: <el-text type="primary" class="mono-text">{{ event.client_id }}</el-text>
                  </p>
                </el-card>
              </el-timeline-item>
            </el-timeline>
          </div>
        </div>
      </template>
      
      <!-- 系统配置详情 -->
      <template v-else-if="type === 'config'">
        <div class="detail-header">
          <div class="header-left">
            <el-icon class="detail-icon" color="#909399">
              <Setting />
            </el-icon>
            <div class="header-info">
              <h3 class="detail-title">系统配置</h3>
              <p class="detail-subtitle">NFC中继服务配置</p>
            </div>
          </div>
        </div>
        
        <el-divider />
        
        <div class="detail-sections">
          <div class="detail-section">
            <h4 class="section-title">服务配置</h4>
            <el-descriptions :column="2" border>
              <el-descriptions-item 
                v-for="(value, key) in data" 
                :key="key"
                :label="formatConfigKey(key)"
              >
                <el-text class="mono-text">{{ value }}</el-text>
              </el-descriptions-item>
            </el-descriptions>
          </div>
        </div>
      </template>
    </div>
    
    <!-- 操作按钮 -->
    <template #footer>
      <div class="dialog-footer">
        <el-button @click="handleClose">关闭</el-button>
        <el-button 
          v-if="type === 'client' && data.is_online" 
          type="danger" 
          @click="handleDisconnectClient"
        >
          断开连接
        </el-button>
        <el-button 
          v-if="type === 'session' && data.status === 'paired'" 
          type="warning" 
          @click="handleTerminateSession"
        >
          终止会话
        </el-button>
        <el-button type="primary" @click="handleRefresh">刷新</el-button>
      </div>
    </template>
  </el-dialog>
</template>

<script setup>
import { ref, computed, watch } from 'vue'
import { 
  CircleCheck, 
  CircleClose, 
  ChatDotRound, 
  Setting 
} from '@element-plus/icons-vue'
import { ElMessage, ElMessageBox } from 'element-plus'
import { formatTime } from '@/utils/format'

const props = defineProps({
  visible: {
    type: Boolean,
    default: false
  },
  type: {
    type: String,
    required: true
    // 'client', 'session', 'config'
  },
  data: {
    type: Object,
    default: () => ({})
  },
  loading: {
    type: Boolean,
    default: false
  }
})

const emit = defineEmits(['update:visible', 'refresh', 'disconnect-client', 'terminate-session'])

// 计算属性
const modalTitle = computed(() => {
  switch (props.type) {
    case 'client': return '客户端详情'
    case 'session': return '会话详情'
    case 'config': return '系统配置'
    default: return '详情'
  }
})

const dialogWidth = computed(() => {
  switch (props.type) {
    case 'session': return '800px'
    case 'config': return '600px'
    default: return '700px'
  }
})

// 状态相关方法
const getStatusColor = (isOnline) => {
  return isOnline ? '#67c23a' : '#c0c4cc'
}

const getStatusTagType = (isOnline) => {
  return isOnline ? 'success' : 'info'
}

const getRoleTagType = (role) => {
  switch (role) {
    case 'provider': return 'success'
    case 'receiver': return 'warning'
    default: return 'info'
  }
}

const getRoleText = (role) => {
  switch (role) {
    case 'provider': return 'Provider (传卡方)'
    case 'receiver': return 'Receiver (收卡方)'
    default: return '未知'
  }
}

const getSessionStatusTagType = (status) => {
  switch (status) {
    case 'paired': return 'success'
    case 'waiting_for_pairing': return 'warning'
    default: return 'info'
  }
}

const getSessionStatusText = (status) => {
  switch (status) {
    case 'paired': return '已配对'
    case 'waiting_for_pairing': return '等待配对'
    default: return '未知'
  }
}

const getSuccessRate = (exchangeCount) => {
  const total = (exchangeCount.upstream || 0) + (exchangeCount.downstream || 0)
  if (total === 0) return 0
  return Math.round((exchangeCount.downstream || 0) / (exchangeCount.upstream || 1) * 100)
}

const formatConfigKey = (key) => {
  return key.replace(/_/g, ' ').replace(/\b\w/g, l => l.toUpperCase())
}

// 事件处理
const handleClose = () => {
  emit('update:visible', false)
}

const handleRefresh = () => {
  emit('refresh')
}

const handleDisconnectClient = async () => {
  try {
    await ElMessageBox.confirm(
      '确定要断开此客户端的连接吗？这将立即终止其当前会话。',
      '确认断开',
      {
        confirmButtonText: '确定',
        cancelButtonText: '取消',
        type: 'warning'
      }
    )
    
    emit('disconnect-client', props.data.client_id)
  } catch {
    // 用户取消操作
  }
}

const handleTerminateSession = async () => {
  try {
    await ElMessageBox.confirm(
      '确定要终止此会话吗？这将断开所有参与方的连接。',
      '确认终止',
      {
        confirmButtonText: '确定',
        cancelButtonText: '取消',
        type: 'warning'
      }
    )
    
    emit('terminate-session', props.data.session_id)
  } catch {
    // 用户取消操作
  }
}
</script>

<style scoped lang="scss">
.detail-modal {
  .detail-content {
    max-height: 70vh;
    overflow-y: auto;
    
    .detail-header {
      display: flex;
      justify-content: space-between;
      align-items: center;
      margin-bottom: 16px;
      
      .header-left {
        display: flex;
        align-items: center;
        gap: 12px;
        
        .detail-icon {
          font-size: 32px;
        }
        
        .header-info {
          .detail-title {
            margin: 0;
            font-size: 18px;
            font-weight: 600;
            color: #303133;
          }
          
          .detail-subtitle {
            margin: 4px 0 0;
            font-size: 12px;
            color: #909399;
            font-family: 'Roboto Mono', monospace;
          }
        }
      }
    }
    
    .detail-sections {
      display: flex;
      flex-direction: column;
      gap: 20px;
      
      .detail-section {
        .section-title {
          margin: 0 0 12px 0;
          font-size: 14px;
          font-weight: 600;
          color: #606266;
          border-left: 3px solid #409eff;
          padding-left: 8px;
        }
        
        .participants-grid {
          display: grid;
          grid-template-columns: 1fr 1fr;
          gap: 16px;
          
          .participant-card {
            .participant-title {
              font-weight: 600;
              font-size: 13px;
            }
          }
        }
        
        .apdu-stats {
          display: grid;
          grid-template-columns: repeat(3, 1fr);
          gap: 16px;
          
          :deep(.el-statistic) {
            text-align: center;
            padding: 16px;
            border: 1px solid #e4e7ed;
            border-radius: 8px;
            
            .el-statistic__content {
              font-weight: 600;
            }
          }
        }
      }
    }
    
    .mono-text {
      font-family: 'Roboto Mono', monospace;
    }
    
    .truncate-text {
      max-width: 200px;
      overflow: hidden;
      text-overflow: ellipsis;
      white-space: nowrap;
    }
    
    .event-detail {
      margin-top: 4px;
      font-size: 12px;
      color: #909399;
    }
  }
  
  .dialog-footer {
    display: flex;
    justify-content: flex-end;
    gap: 12px;
  }
}

// 响应式设计
@media (max-width: 768px) {
  .detail-modal {
    .detail-content {
      .detail-sections {
        .detail-section {
          .participants-grid {
            grid-template-columns: 1fr;
          }
          
          .apdu-stats {
            grid-template-columns: 1fr;
          }
        }
      }
    }
  }
}

// 深色主题适配
.dark {
  .detail-modal {
    .detail-content {
      .detail-header {
        .header-info {
          .detail-title {
            color: #ffffff;
          }
        }
      }
      
      .detail-sections {
        .detail-section {
          .section-title {
            color: #cccccc;
          }
        }
      }
    }
  }
}
</style> 