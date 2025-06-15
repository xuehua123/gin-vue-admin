<template>
  <div class="transaction-detail">
    <!-- 基本信息 -->
    <el-card class="detail-card" shadow="hover">
      <template #header>
        <div class="card-header">
          <span class="card-title">交易基本信息</span>
          <el-tag :type="getStatusType(transaction.status)" size="large">
            {{ getStatusText(transaction.status) }}
          </el-tag>
        </div>
      </template>
      
      <el-row :gutter="20">
        <el-col :span="12">
          <div class="info-item">
            <span class="label">交易ID:</span>
            <span class="value">{{ transaction.transaction_id }}</span>
          </div>
          <div class="info-item">
            <span class="label">传卡端:</span>
            <span class="value">{{ transaction.transmitter_client_id }}</span>
          </div>
          <div class="info-item">
            <span class="label">收卡端:</span>
            <span class="value">{{ transaction.receiver_client_id || '未分配' }}</span>
          </div>
          <div class="info-item">
            <span class="label">卡片类型:</span>
            <span class="value">{{ transaction.card_type || '--' }}</span>
          </div>
          <div class="info-item">
            <span class="label">描述:</span>
            <span class="value">{{ transaction.description || '--' }}</span>
          </div>
        </el-col>
        <el-col :span="12">
          <div class="info-item">
            <span class="label">创建时间:</span>
            <span class="value">{{ formatTime(transaction.created_at) }}</span>
          </div>
          <div class="info-item">
            <span class="label">开始时间:</span>
            <span class="value">{{ formatTime(transaction.started_at) }}</span>
          </div>
          <div class="info-item">
            <span class="label">完成时间:</span>
            <span class="value">{{ formatTime(transaction.completed_at) }}</span>
          </div>
          <div class="info-item">
            <span class="label">过期时间:</span>
            <span class="value">{{ formatTime(transaction.expires_at) }}</span>
          </div>
          <div class="info-item">
            <span class="label">耗时:</span>
            <span class="value">{{ getDuration() }}</span>
          </div>
        </el-col>
      </el-row>

      <!-- 错误信息 -->
      <div v-if="transaction.error_msg" class="error-section">
        <el-alert
          title="错误信息"
          type="error"
          :description="transaction.error_msg"
          show-icon
          :closable="false"
        />
      </div>

      <!-- 结束原因 -->
      <div v-if="transaction.end_reason" class="info-item">
        <span class="label">结束原因:</span>
        <span class="value">{{ transaction.end_reason }}</span>
      </div>
    </el-card>

    <!-- 统计信息 -->
    <el-card class="detail-card" shadow="hover" v-if="showStatistics">
      <template #header>
        <span class="card-title">统计信息</span>
      </template>
      
      <el-row :gutter="20">
        <el-col :span="8">
          <div class="stat-item">
            <div class="stat-value">{{ transaction.apdu_count || 0 }}</div>
            <div class="stat-label">APDU消息数</div>
          </div>
        </el-col>
        <el-col :span="8">
          <div class="stat-item">
            <div class="stat-value">{{ transaction.total_processing_time_ms || 0 }}ms</div>
            <div class="stat-label">总处理时间</div>
          </div>
        </el-col>
        <el-col :span="8">
          <div class="stat-item">
            <div class="stat-value">{{ transaction.average_response_time_ms || 0 }}ms</div>
            <div class="stat-label">平均响应时间</div>
          </div>
        </el-col>
      </el-row>
    </el-card>

    <!-- APDU消息列表 -->
    <el-card class="detail-card" shadow="hover">
      <template #header>
        <div class="card-header">
          <span class="card-title">APDU消息</span>
          <div class="header-controls">
            <el-button size="small" @click="loadAPDUMessages" :loading="apduLoading">
              刷新
            </el-button>
            <el-button size="small" type="primary" @click="toggleAutoScroll">
              {{ autoScroll ? '停止滚动' : '自动滚动' }}
            </el-button>
          </div>
        </div>
      </template>

      <div class="apdu-container" ref="apduContainer">
        <div v-if="apduMessages.length === 0" class="empty-state">
          <el-empty description="暂无APDU消息" />
        </div>
        
        <div v-else class="apdu-timeline">
          <div
            v-for="(message, index) in apduMessages"
            :key="message.ID"
            class="apdu-message"
            :class="getMessageClass(message)"
          >
            <div class="message-header">
              <div class="message-info">
                <span class="sequence">#{{ message.sequence_number }}</span>
                <span class="direction">{{ getDirectionText(message.direction) }}</span>
                <span class="time">{{ formatTime(message.created_at) }}</span>
                <el-tag
                  :type="getMessageStatusType(message.status)"
                  size="small"
                >
                  {{ getMessageStatusText(message.status) }}
                </el-tag>
              </div>
              <div class="message-controls">
                <el-button size="small" @click="toggleMessageDetail(index)">
                  {{ expandedMessages.includes(index) ? '收起' : '详情' }}
                </el-button>
              </div>
            </div>
            
            <div class="message-content">
              <div class="apdu-hex">
                <span class="hex-label">APDU:</span>
                <code class="hex-value">{{ formatHex(message.apdu_hex) }}</code>
                <el-button
                  size="small"
                  text
                  @click="copyToClipboard(message.apdu_hex)"
                >
                  复制
                </el-button>
              </div>
              
              <!-- 详细信息（可展开） -->
              <div v-if="expandedMessages.includes(index)" class="message-detail">
                <div class="detail-row">
                  <span class="detail-label">消息类型:</span>
                  <span class="detail-value">{{ message.message_type || '--' }}</span>
                </div>
                <div class="detail-row">
                  <span class="detail-label">优先级:</span>
                  <span class="detail-value">{{ message.priority || 'normal' }}</span>
                </div>
                <div class="detail-row">
                  <span class="detail-label">发送时间:</span>
                  <span class="detail-value">{{ formatTime(message.sent_at) }}</span>
                </div>
                <div class="detail-row">
                  <span class="detail-label">接收时间:</span>
                  <span class="detail-value">{{ formatTime(message.received_at) }}</span>
                </div>
                <div class="detail-row">
                  <span class="detail-label">响应时间:</span>
                  <span class="detail-value">{{ message.response_time_ms || 0 }}ms</span>
                </div>
                <div v-if="message.error_msg" class="detail-row error">
                  <span class="detail-label">错误信息:</span>
                  <span class="detail-value">{{ message.error_msg }}</span>
                </div>
              </div>
            </div>
          </div>
        </div>
      </div>
    </el-card>

    <!-- 操作按钮 -->
    <div class="action-bar">
      <el-button @click="$emit('refresh')">
        刷新列表
      </el-button>
      <el-button
        v-if="canCancel"
        type="warning"
        @click="handleCancel"
        :loading="cancelLoading"
      >
        取消交易
      </el-button>
      <el-button
        v-if="canRetry"
        type="primary"
        @click="handleRetry"
        :loading="retryLoading"
      >
        重试交易
      </el-button>
    </div>
  </div>
</template>

<script setup>
import { ref, computed, onMounted, onUnmounted, nextTick } from 'vue'
import { ElMessage, ElMessageBox } from 'element-plus'
import { getAPDUList, updateTransactionStatus } from '@/api/nfcRelay'

defineOptions({
  name: 'TransactionDetail'
})

const props = defineProps({
  transaction: {
    type: Object,
    required: true
  }
})

const emit = defineEmits(['refresh'])

// 响应式数据
const apduMessages = ref([])
const apduLoading = ref(false)
const cancelLoading = ref(false)
const retryLoading = ref(false)
const expandedMessages = ref([])
const autoScroll = ref(false)
const apduContainer = ref(null)

// 计算属性
const showStatistics = computed(() => {
  return props.transaction.apdu_count > 0 || 
         props.transaction.total_processing_time_ms > 0
})

const canCancel = computed(() => {
  return ['pending', 'active', 'processing'].includes(props.transaction.status)
})

const canRetry = computed(() => {
  return ['failed', 'timeout'].includes(props.transaction.status)
})

// 组件挂载
onMounted(() => {
  loadAPDUMessages()
})

// 组件卸载
onUnmounted(() => {
  // 清理定时器等
})

// 加载APDU消息
async function loadAPDUMessages() {
  apduLoading.value = true
  try {
    const response = await getAPDUList({
      transaction_id: props.transaction.transaction_id
    })
    apduMessages.value = response.data || []
    
    if (autoScroll.value) {
      await nextTick()
      scrollToBottom()
    }
  } catch (error) {
    console.error('加载APDU消息失败:', error)
    ElMessage.error('加载APDU消息失败')
  } finally {
    apduLoading.value = false
  }
}

// 取消交易
async function handleCancel() {
  try {
    await ElMessageBox.confirm(
      `确定要取消交易 ${props.transaction.transaction_id} 吗？`,
      '确认取消',
      {
        confirmButtonText: '确定',
        cancelButtonText: '取消',
        type: 'warning'
      }
    )
    
    cancelLoading.value = true
    await updateTransactionStatus({
      transaction_id: props.transaction.transaction_id,
      status: 'cancelled',
      reason: '用户手动取消'
    })
    
    ElMessage.success('交易已取消')
    emit('refresh')
  } catch (error) {
    if (error !== 'cancel') {
      console.error('取消交易失败:', error)
      ElMessage.error('取消交易失败')
    }
  } finally {
    cancelLoading.value = false
  }
}

// 重试交易
async function handleRetry() {
  try {
    await ElMessageBox.confirm(
      `确定要重试交易 ${props.transaction.transaction_id} 吗？`,
      '确认重试',
      {
        confirmButtonText: '确定',
        cancelButtonText: '取消',
        type: 'info'
      }
    )
    
    retryLoading.value = true
    await updateTransactionStatus({
      transaction_id: props.transaction.transaction_id,
      status: 'pending',
      reason: '用户手动重试'
    })
    
    ElMessage.success('交易已重新启动')
    emit('refresh')
  } catch (error) {
    if (error !== 'cancel') {
      console.error('重试交易失败:', error)
      ElMessage.error('重试交易失败')
    }
  } finally {
    retryLoading.value = false
  }
}

// 切换消息详情展开状态
function toggleMessageDetail(index) {
  const expandedIndex = expandedMessages.value.indexOf(index)
  if (expandedIndex > -1) {
    expandedMessages.value.splice(expandedIndex, 1)
  } else {
    expandedMessages.value.push(index)
  }
}

// 切换自动滚动
function toggleAutoScroll() {
  autoScroll.value = !autoScroll.value
  if (autoScroll.value) {
    scrollToBottom()
  }
}

// 滚动到底部
function scrollToBottom() {
  if (apduContainer.value) {
    apduContainer.value.scrollTop = apduContainer.value.scrollHeight
  }
}

// 复制到剪贴板
async function copyToClipboard(text) {
  try {
    await navigator.clipboard.writeText(text)
    ElMessage.success('已复制到剪贴板')
  } catch (error) {
    ElMessage.error('复制失败')
  }
}

// 工具函数
function getStatusType(status) {
  const typeMap = {
    'creating': 'info',
    'pending': 'warning',
    'active': 'primary',
    'processing': 'primary',
    'completed': 'success',
    'failed': 'danger',
    'cancelled': 'info',
    'timeout': 'danger'
  }
  return typeMap[status] || 'info'
}

function getStatusText(status) {
  const textMap = {
    'creating': '创建中',
    'pending': '等待中',
    'active': '活跃中',
    'processing': '处理中',
    'completed': '已完成',
    'failed': '失败',
    'cancelled': '已取消',
    'timeout': '超时'
  }
  return textMap[status] || status
}

function getDirectionText(direction) {
  const textMap = {
    'to_receiver': '→ 收卡端',
    'to_transmitter': '→ 传卡端'
  }
  return textMap[direction] || direction
}

function getMessageStatusType(status) {
  const typeMap = {
    'pending': 'warning',
    'sent': 'primary',
    'received': 'success',
    'processed': 'success',
    'failed': 'danger'
  }
  return typeMap[status] || 'info'
}

function getMessageStatusText(status) {
  const textMap = {
    'pending': '等待',
    'sent': '已发送',
    'received': '已接收',
    'processed': '已处理',
    'failed': '失败'
  }
  return textMap[status] || status
}

function getMessageClass(message) {
  return {
    'to-receiver': message.direction === 'to_receiver',
    'to-transmitter': message.direction === 'to_transmitter',
    'failed': message.status === 'failed'
  }
}

function formatTime(timeStr) {
  if (!timeStr) return '--'
  return new Date(timeStr).toLocaleString('zh-CN')
}

function formatHex(hex) {
  if (!hex) return ''
  // 每两个字符添加空格
  return hex.replace(/(.{2})/g, '$1 ').trim().toUpperCase()
}

function getDuration() {
  const { created_at, completed_at } = props.transaction
  if (!created_at) return '--'
  
  const start = new Date(created_at)
  const end = completed_at ? new Date(completed_at) : new Date()
  const duration = Math.floor((end - start) / 1000)
  
  if (duration < 60) return `${duration}s`
  if (duration < 3600) return `${Math.floor(duration / 60)}m ${duration % 60}s`
  return `${Math.floor(duration / 3600)}h ${Math.floor((duration % 3600) / 60)}m`
}
</script>

<style scoped>
.transaction-detail {
  max-height: 80vh;
  overflow-y: auto;
}

.detail-card {
  margin-bottom: 20px;
}

.card-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
}

.card-title {
  font-size: 16px;
  font-weight: bold;
  color: #303133;
}

.header-controls {
  display: flex;
  gap: 10px;
}

.info-item {
  display: flex;
  margin-bottom: 12px;
  align-items: flex-start;
}

.info-item .label {
  width: 100px;
  color: #909399;
  font-size: 14px;
  flex-shrink: 0;
}

.info-item .value {
  color: #303133;
  font-size: 14px;
  word-break: break-all;
}

.error-section {
  margin-top: 20px;
}

.stat-item {
  text-align: center;
  padding: 20px;
  background: #f8f9fa;
  border-radius: 8px;
}

.stat-value {
  font-size: 24px;
  font-weight: bold;
  color: #409EFF;
  margin-bottom: 5px;
}

.stat-label {
  font-size: 14px;
  color: #909399;
}

.apdu-container {
  max-height: 400px;
  overflow-y: auto;
  border: 1px solid #EBEEF5;
  border-radius: 4px;
  padding: 10px;
}

.empty-state {
  text-align: center;
  padding: 40px;
}

.apdu-timeline {
  position: relative;
}

.apdu-message {
  margin-bottom: 16px;
  padding: 12px;
  border: 1px solid #EBEEF5;
  border-radius: 6px;
  background: #fff;
}

.apdu-message.to-receiver {
  border-left: 4px solid #409EFF;
}

.apdu-message.to-transmitter {
  border-left: 4px solid #67C23A;
}

.apdu-message.failed {
  border-left: 4px solid #F56C6C;
  background: #FEF0F0;
}

.message-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-bottom: 8px;
}

.message-info {
  display: flex;
  align-items: center;
  gap: 12px;
}

.sequence {
  font-weight: bold;
  color: #303133;
}

.direction {
  font-size: 12px;
  color: #909399;
}

.time {
  font-size: 12px;
  color: #C0C4CC;
}

.message-content {
  font-size: 14px;
}

.apdu-hex {
  display: flex;
  align-items: center;
  gap: 8px;
  margin-bottom: 8px;
}

.hex-label {
  color: #909399;
  font-size: 12px;
}

.hex-value {
  flex: 1;
  padding: 4px 8px;
  background: #F5F7FA;
  border-radius: 4px;
  font-family: 'Courier New', monospace;
  font-size: 12px;
  word-break: break-all;
}

.message-detail {
  margin-top: 8px;
  padding: 8px;
  background: #F8F9FA;
  border-radius: 4px;
}

.detail-row {
  display: flex;
  margin-bottom: 4px;
}

.detail-row.error {
  color: #F56C6C;
}

.detail-label {
  width: 80px;
  color: #909399;
  font-size: 12px;
}

.detail-value {
  color: #303133;
  font-size: 12px;
}

.action-bar {
  display: flex;
  justify-content: center;
  gap: 12px;
  padding: 20px;
  border-top: 1px solid #EBEEF5;
  background: #FAFAFA;
}
</style> 