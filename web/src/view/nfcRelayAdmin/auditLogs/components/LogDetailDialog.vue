<!--
  审计日志详情对话框
  展示审计日志的详细信息、相关上下文和操作历史
-->
<template>
  <el-dialog
    v-model="visible"
    title="审计日志详情"
    width="900px"
    :close-on-click-modal="false"
    append-to-body
    draggable
    @close="handleClose"
  >
    <div v-if="logData" class="log-detail">
      <!-- 基本信息 -->
      <div class="detail-section">
        <div class="section-header">
          <el-icon class="header-icon" color="#409EFF">
            <Document />
          </el-icon>
          <h3 class="section-title">日志信息</h3>
          <el-tag 
            :type="getEventTypeInfo(logData.event_type).type"
            size="default"
          >
            <el-icon class="tag-icon">
              <component :is="getEventTypeInfo(logData.event_type).icon" />
            </el-icon>
            {{ getEventTypeInfo(logData.event_type).text }}
          </el-tag>
        </div>
        
        <div class="info-grid">
          <div class="info-item">
            <span class="info-label">记录时间:</span>
            <span class="info-value">{{ formatDateTime(logData.timestamp) }}</span>
          </div>
          <div class="info-item">
            <span class="info-label">事件类型:</span>
            <span class="info-value">{{ logData.event_type }}</span>
          </div>
          <div class="info-item">
            <span class="info-label">源IP地址:</span>
            <span class="info-value">{{ logData.source_ip || '-' }}</span>
          </div>
          <div class="info-item">
            <span class="info-label">严重级别:</span>
            <el-tag 
              :type="getSeverityType(logData.severity)"
              size="small"
            >
              {{ getSeverityText(logData.severity) }}
            </el-tag>
          </div>
          <div class="info-item">
            <span class="info-label">日志ID:</span>
            <span class="info-value">{{ logData.id || '-' }}</span>
          </div>
        </div>
      </div>

      <!-- 参与者信息 -->
      <div v-if="logData.client_id_initiator || logData.client_id_responder" class="detail-section">
        <div class="section-header">
          <el-icon class="header-icon" color="#67C23A">
            <User />
          </el-icon>
          <h3 class="section-title">参与者信息</h3>
        </div>
        
        <div class="participants-section">
          <div v-if="logData.client_id_initiator" class="participant-card">
            <div class="participant-header">
              <el-tag type="primary">发起方 (Initiator)</el-tag>
            </div>
            <div class="participant-info">
              <div class="info-row">
                <span class="label">客户端ID:</span>
                <el-button 
                  type="primary" 
                  link 
                  size="small"
                  @click="viewClient(logData.client_id_initiator)"
                >
                  {{ logData.client_id_initiator }}
                </el-button>
              </div>
              <div class="info-row">
                <span class="label">用户ID:</span>
                <span class="value">{{ logData.user_id || '-' }}</span>
              </div>
            </div>
          </div>

          <div v-if="logData.client_id_initiator && logData.client_id_responder" class="exchange-indicator">
            <el-icon><Right /></el-icon>
          </div>

          <div v-if="logData.client_id_responder" class="participant-card">
            <div class="participant-header">
              <el-tag type="success">响应方 (Responder)</el-tag>
            </div>
            <div class="participant-info">
              <div class="info-row">
                <span class="label">客户端ID:</span>
                <el-button 
                  type="success" 
                  link 
                  size="small"
                  @click="viewClient(logData.client_id_responder)"
                >
                  {{ logData.client_id_responder }}
                </el-button>
              </div>
            </div>
          </div>
        </div>
      </div>

      <!-- 关联对象 -->
      <div v-if="logData.session_id" class="detail-section">
        <div class="section-header">
          <el-icon class="header-icon" color="#E6A23C">
            <Connection />
          </el-icon>
          <h3 class="section-title">关联对象</h3>
        </div>
        
        <div class="related-objects-grid">
          <div class="related-item">
            <div class="item-label">关联会话:</div>
            <el-button 
              type="warning" 
              link 
              @click="viewSession(logData.session_id)"
            >
              <el-icon><Connection /></el-icon>
              {{ logData.session_id }}
            </el-button>
          </div>
        </div>
      </div>

      <!-- 事件详情 -->
      <div v-if="logData.details && Object.keys(logData.details).length > 0" class="detail-section">
        <div class="section-header">
          <el-icon class="header-icon" color="#F56C6C">
            <InfoFilled />
          </el-icon>
          <h3 class="section-title">事件详情</h3>
        </div>
        
        <div class="details-container">
          <div class="details-grid">
            <div 
              v-for="(value, key) in logData.details" 
              :key="key"
              class="detail-item"
            >
              <div class="detail-label">{{ getDetailLabel(key) }}:</div>
              <div class="detail-value">
                <template v-if="key === 'error_message' || key === 'error'">
                  <el-text type="danger">{{ value }}</el-text>
                </template>
                <template v-else-if="key === 'apdu_length' || key === 'data_length'">
                  <el-tag size="small" type="info">{{ value }} 字节</el-tag>
                </template>
                <template v-else-if="key === 'session_duration' || key === 'duration'">
                  <el-tag size="small" type="success">{{ value }}</el-tag>
                </template>
                <template v-else-if="key === 'response_code'">
                  <el-tag 
                    size="small" 
                    :type="value === '9000' ? 'success' : 'warning'"
                  >
                    {{ value }}
                  </el-tag>
                </template>
                <template v-else-if="typeof value === 'object'">
                  <div class="json-preview">
                    <pre>{{ JSON.stringify(value, null, 2) }}</pre>
                  </div>
                </template>
                <template v-else>
                  <span>{{ value }}</span>
                </template>
              </div>
            </div>
          </div>
        </div>
      </div>

      <!-- 技术信息 -->
      <div class="detail-section">
        <div class="section-header">
          <el-icon class="header-icon" color="#909399">
            <Setting />
          </el-icon>
          <h3 class="section-title">技术信息</h3>
          <el-button 
            type="primary" 
            link 
            size="small"
            @click="copyToClipboard"
          >
            <el-icon><CopyDocument /></el-icon>
            复制JSON
          </el-button>
        </div>
        
        <div class="tech-info">
          <el-collapse>
            <el-collapse-item title="原始JSON数据" name="raw-json">
              <div class="json-display">
                <pre>{{ JSON.stringify(logData, null, 2) }}</pre>
              </div>
            </el-collapse-item>
            
            <el-collapse-item title="字段说明" name="field-desc">
              <div class="field-descriptions">
                <div class="desc-item">
                  <strong>timestamp:</strong> 事件发生的精确时间戳
                </div>
                <div class="desc-item">
                  <strong>event_type:</strong> 事件的分类类型，用于快速识别
                </div>
                <div class="desc-item">
                  <strong>client_id_*:</strong> 参与事件的客户端标识符
                </div>
                <div class="desc-item">
                  <strong>session_id:</strong> 关联的NFC中继会话标识符
                </div>
                <div class="desc-item">
                  <strong>details:</strong> 事件的具体细节和上下文信息
                </div>
              </div>
            </el-collapse-item>
          </el-collapse>
        </div>
      </div>

      <!-- 相关日志 -->
      <div class="detail-section">
        <div class="section-header">
          <el-icon class="header-icon" color="#606266">
            <List />
          </el-icon>
          <h3 class="section-title">相关日志</h3>
          <el-button 
            type="primary" 
            link 
            size="small"
            @click="viewRelatedLogs"
          >
            查看全部
          </el-button>
        </div>
        
        <div class="related-logs">
          <div v-if="relatedLogs.length > 0" class="logs-list">
            <div 
              v-for="log in relatedLogs.slice(0, 5)"
              :key="log.timestamp"
              class="log-item"
            >
              <div class="log-time">{{ formatDateTime(log.timestamp, 'MM-DD HH:mm:ss') }}</div>
              <div class="log-content">
                <el-tag 
                  :type="getEventTypeInfo(log.event_type).type"
                  size="small"
                >
                  {{ getEventTypeInfo(log.event_type).text }}
                </el-tag>
                <span class="log-summary">{{ log.summary || '相关事件' }}</span>
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
          type="primary" 
          @click="handleRefresh"
        >
          刷新
        </el-button>
      </div>
    </template>
  </el-dialog>
</template>

<script setup>
import { ref, computed, watch } from 'vue'
import { useRouter } from 'vue-router'
import { 
  Document,
  User,
  Connection,
  InfoFilled,
  Setting,
  List,
  Right,
  CopyDocument,
  SuccessFilled,
  CircleClose,
  Warning,
  Switch,
  UserFilled
} from '@element-plus/icons-vue'
import { ElMessage } from 'element-plus'

import { 
  formatDateTime,
  formatEventType
} from '../../utils/formatters'

const props = defineProps({
  modelValue: {
    type: Boolean,
    default: false
  },
  logData: {
    type: Object,
    default: null
  }
})

const emit = defineEmits(['update:modelValue', 'refresh'])

const router = useRouter()

// 状态管理
const visible = ref(false)
const relatedLogs = ref([])

// 监听modelValue变化
watch(() => props.modelValue, (val) => {
  visible.value = val
  if (val && props.logData) {
    generateRelatedLogs()
  }
})

watch(visible, (val) => {
  emit('update:modelValue', val)
})

// 生成相关日志
const generateRelatedLogs = () => {
  if (!props.logData) return
  
  // 模拟相关日志
  const mockRelatedLogs = []
  const baseTime = new Date(props.logData.timestamp).getTime()
  
  for (let i = 1; i <= 3; i++) {
    mockRelatedLogs.push({
      timestamp: new Date(baseTime - i * 60000).toISOString(),
      event_type: i === 1 ? 'session_established' : 'apdu_relayed_success',
      summary: i === 1 ? '相关会话建立' : `APDU交换 #${i}`
    })
  }
  
  relatedLogs.value = mockRelatedLogs
}

// 工具函数
const getEventTypeInfo = (eventType) => {
  const { text, type } = formatEventType(eventType)
  
  const iconMap = {
    session_established: Connection,
    session_terminated: Switch,
    apdu_relayed_success: SuccessFilled,
    apdu_relayed_failure: CircleClose,
    client_connected: UserFilled,
    client_disconnected: UserFilled,
    auth_failure: Warning,
    permission_denied: Setting
  }
  
  return {
    text,
    type,
    icon: iconMap[eventType] || Document
  }
}

const getSeverityType = (severity) => {
  const typeMap = {
    low: 'info',
    medium: 'warning',
    high: 'danger'
  }
  return typeMap[severity] || 'info'
}

const getSeverityText = (severity) => {
  const textMap = {
    low: '低',
    medium: '中',
    high: '高'
  }
  return textMap[severity] || severity || '-'
}

const getDetailLabel = (key) => {
  const labelMap = {
    message: '消息',
    error: '错误信息',
    error_message: '错误信息',
    reason: '原因',
    duration: '持续时间',
    session_duration: '会话持续时间',
    apdu_length: 'APDU长度',
    data_length: '数据长度',
    response_code: '响应码',
    client_version: '客户端版本',
    user_agent: 'User Agent',
    connection_type: '连接类型',
    attempts: '尝试次数',
    protocol: '协议',
    method: '方法'
  }
  return labelMap[key] || key.replace(/_/g, ' ').replace(/\b\w/g, l => l.toUpperCase())
}

// 事件处理
const handleClose = () => {
  visible.value = false
}

const handleRefresh = () => {
  emit('refresh')
}

const viewClient = (clientId) => {
  router.push({
    path: '/nfc-relay-admin/clients',
    query: { clientId }
  })
  handleClose()
}

const viewSession = (sessionId) => {
  router.push({
    path: '/nfc-relay-admin/sessions',
    query: { sessionId }
  })
  handleClose()
}

const viewRelatedLogs = () => {
  const query = {}
  if (props.logData?.session_id) query.sessionId = props.logData.session_id
  if (props.logData?.client_id_initiator) query.clientId = props.logData.client_id_initiator
  
  router.push({
    path: '/nfc-relay-admin/audit-logs',
    query
  })
  handleClose()
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
.log-detail {
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
      
      .tag-icon {
        margin-right: 4px;
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
              max-width: 150px;
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
    
    .related-objects-grid {
      display: grid;
      grid-template-columns: repeat(auto-fit, minmax(250px, 1fr));
      gap: 16px;
      
      .related-item {
        padding: 12px;
        background: #f8f9fa;
        border-radius: 6px;
        
        .item-label {
          font-size: 13px;
          color: #909399;
          margin-bottom: 4px;
        }
      }
    }
    
    .details-container {
      .details-grid {
        display: grid;
        grid-template-columns: repeat(auto-fit, minmax(300px, 1fr));
        gap: 12px;
        
        .detail-item {
          padding: 12px;
          background: white;
          border: 1px solid #f0f0f0;
          border-radius: 6px;
          
          .detail-label {
            font-size: 13px;
            color: #909399;
            margin-bottom: 4px;
            font-weight: 500;
          }
          
          .detail-value {
            font-size: 14px;
            color: #303133;
            
            .json-preview {
              background: #f5f5f5;
              padding: 8px;
              border-radius: 4px;
              margin-top: 4px;
              
              pre {
                margin: 0;
                font-size: 12px;
                max-height: 150px;
                overflow-y: auto;
              }
            }
          }
        }
      }
    }
    
    .tech-info {
      .json-display {
        background: #f5f7fa;
        border: 1px solid #e4e7ed;
        border-radius: 4px;
        padding: 12px;
        
        pre {
          margin: 0;
          font-family: 'Courier New', monospace;
          font-size: 12px;
          line-height: 1.5;
          max-height: 300px;
          overflow-y: auto;
          white-space: pre-wrap;
          word-break: break-all;
        }
      }
      
      .field-descriptions {
        .desc-item {
          margin-bottom: 8px;
          font-size: 13px;
          line-height: 1.5;
          
          &:last-child {
            margin-bottom: 0;
          }
          
          strong {
            color: #409EFF;
            font-family: 'Courier New', monospace;
          }
        }
      }
    }
    
    .related-logs {
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
            
            .log-summary {
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
  .log-detail {
    .participants-section {
      flex-direction: column;
      
      .exchange-indicator {
        transform: rotate(90deg);
      }
    }
    
    .details-grid {
      grid-template-columns: 1fr;
    }
    
    .related-objects-grid {
      grid-template-columns: 1fr;
    }
  }
}
</style> 