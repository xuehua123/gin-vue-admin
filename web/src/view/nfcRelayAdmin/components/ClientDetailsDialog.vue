<template>
  <el-dialog
    v-model="dialogVisible"
    :title="dialogTitle"
    width="1000px"
    :close-on-click-modal="false"
    destroy-on-close
  >
    <div v-if="clientData" class="client-details">
      <!-- 基本信息 -->
      <el-descriptions 
        title="基本信息" 
        :column="2" 
        border
        class="mb-4"
      >
        <el-descriptions-item label="客户端ID">
          {{ clientData.clientId }}
        </el-descriptions-item>
        <el-descriptions-item label="连接状态">
          <el-tag :type="getStatusTagType(clientData.status)">
            {{ getStatusText(clientData.status) }}
          </el-tag>
        </el-descriptions-item>
        <el-descriptions-item label="设备名称">
          {{ clientData.deviceName || '未知设备' }}
        </el-descriptions-item>
        <el-descriptions-item label="设备类型">
          {{ clientData.deviceType }}
        </el-descriptions-item>
        <el-descriptions-item label="应用版本">
          {{ clientData.appVersion }}
        </el-descriptions-item>
        <el-descriptions-item label="系统版本">
          {{ clientData.systemVersion }}
        </el-descriptions-item>
        <el-descriptions-item label="IP地址">
          {{ clientData.ipAddress }}
        </el-descriptions-item>
        <el-descriptions-item label="地理位置">
          {{ clientData.location }}
        </el-descriptions-item>
      </el-descriptions>

      <!-- 连接信息 -->
      <el-descriptions 
        title="连接信息" 
        :column="2" 
        border
        class="mb-4"
      >
        <el-descriptions-item label="首次连接时间">
          {{ formatDateTime(clientData.firstConnectedAt) }}
        </el-descriptions-item>
        <el-descriptions-item label="最后活动时间">
          {{ formatDateTime(clientData.lastActiveAt) }}
        </el-descriptions-item>
        <el-descriptions-item label="在线时长">
          {{ formatDuration(clientData.onlineDuration) }}
        </el-descriptions-item>
        <el-descriptions-item label="重连次数">
          {{ clientData.reconnectCount || 0 }}
        </el-descriptions-item>
        <el-descriptions-item label="数据传输量">
          {{ formatBytes(clientData.dataTransferred) }}
        </el-descriptions-item>
        <el-descriptions-item label="会话数量">
          {{ clientData.sessionCount || 0 }}
        </el-descriptions-item>
      </el-descriptions>

      <!-- 标签页 -->
      <el-tabs v-model="activeTab" type="card">
        <!-- 会话历史 -->
        <el-tab-pane label="会话历史" name="sessions">
          <el-table 
            :data="sessionHistory" 
            v-loading="loadingSessions"
            max-height="300"
          >
            <el-table-column prop="sessionId" label="会话ID" width="200" />
            <el-table-column prop="startTime" label="开始时间" width="180">
              <template #default="{ row }">
                {{ formatDateTime(row.startTime) }}
              </template>
            </el-table-column>
            <el-table-column prop="endTime" label="结束时间" width="180">
              <template #default="{ row }">
                {{ formatDateTime(row.endTime) }}
              </template>
            </el-table-column>
            <el-table-column prop="duration" label="持续时间" width="120">
              <template #default="{ row }">
                {{ formatDuration(row.duration) }}
              </template>
            </el-table-column>
            <el-table-column prop="apduCount" label="APDU数量" width="100" />
            <el-table-column prop="status" label="状态" width="100">
              <template #default="{ row }">
                <el-tag :type="getSessionStatusType(row.status)" size="small">
                  {{ getSessionStatusText(row.status) }}
                </el-tag>
              </template>
            </el-table-column>
          </el-table>
        </el-tab-pane>

        <!-- 连接日志 -->
        <el-tab-pane label="连接日志" name="logs">
          <el-table 
            :data="connectionLogs" 
            v-loading="loadingLogs"
            max-height="300"
          >
            <el-table-column prop="timestamp" label="时间" width="180">
              <template #default="{ row }">
                {{ formatDateTime(row.timestamp) }}
              </template>
            </el-table-column>
            <el-table-column prop="event" label="事件" width="120" />
            <el-table-column prop="level" label="级别" width="80">
              <template #default="{ row }">
                <el-tag :type="getLogLevelType(row.level)" size="small">
                  {{ row.level }}
                </el-tag>
              </template>
            </el-table-column>
            <el-table-column prop="message" label="消息" show-overflow-tooltip />
          </el-table>
        </el-tab-pane>

        <!-- 性能指标 -->
        <el-tab-pane label="性能指标" name="metrics">
          <el-row :gutter="16">
            <el-col :span="12">
              <el-card>
                <template #header>
                  <span>连接质量</span>
                </template>
                <div class="metric-item">
                  <span class="metric-label">平均延迟:</span>
                  <span class="metric-value">{{ clientData.averageLatency || 0 }}ms</span>
                </div>
                <div class="metric-item">
                  <span class="metric-label">丢包率:</span>
                  <span class="metric-value">{{ (clientData.packetLossRate || 0).toFixed(2) }}%</span>
                </div>
                <div class="metric-item">
                  <span class="metric-label">吞吐量:</span>
                  <span class="metric-value">{{ formatBytes(clientData.throughput) }}/s</span>
                </div>
              </el-card>
            </el-col>
            <el-col :span="12">
              <el-card>
                <template #header>
                  <span>APDU统计</span>
                </template>
                <div class="metric-item">
                  <span class="metric-label">总APDU数:</span>
                  <span class="metric-value">{{ clientData.totalApduCount || 0 }}</span>
                </div>
                <div class="metric-item">
                  <span class="metric-label">成功率:</span>
                  <span class="metric-value">{{ (clientData.apduSuccessRate || 0).toFixed(2) }}%</span>
                </div>
                <div class="metric-item">
                  <span class="metric-label">平均响应时间:</span>
                  <span class="metric-value">{{ clientData.averageResponseTime || 0 }}ms</span>
                </div>
              </el-card>
            </el-col>
          </el-row>
        </el-tab-pane>

        <!-- 设备信息 -->
        <el-tab-pane label="设备信息" name="device">
          <el-descriptions :column="2" border>
            <el-descriptions-item label="制造商">
              {{ clientData.manufacturer || '未知' }}
            </el-descriptions-item>
            <el-descriptions-item label="型号">
              {{ clientData.deviceModel || '未知' }}
            </el-descriptions-item>
            <el-descriptions-item label="屏幕分辨率">
              {{ clientData.screenResolution || '未知' }}
            </el-descriptions-item>
            <el-descriptions-item label="内存">
              {{ clientData.memory || '未知' }}
            </el-descriptions-item>
            <el-descriptions-item label="存储空间">
              {{ clientData.storage || '未知' }}
            </el-descriptions-item>
            <el-descriptions-item label="电池状态">
              {{ clientData.batteryLevel || '未知' }}
            </el-descriptions-item>
            <el-descriptions-item label="网络类型">
              {{ clientData.networkType || '未知' }}
            </el-descriptions-item>
            <el-descriptions-item label="运营商">
              {{ clientData.carrier || '未知' }}
            </el-descriptions-item>
          </el-descriptions>
        </el-tab-pane>
      </el-tabs>
    </div>

    <template #footer>
      <div class="dialog-footer">
        <el-button @click="closeDialog">关闭</el-button>
        <el-button type="primary" @click="refreshData">刷新数据</el-button>
        <el-button 
          v-if="clientData && clientData.status === 'online'"
          type="warning" 
          @click="disconnectClient"
        >
          断开连接
        </el-button>
      </div>
    </template>
  </el-dialog>
</template>

<script>
import { ref, computed, watch } from 'vue'
import { ElMessage, ElMessageBox } from 'element-plus'
import { formatDateTime, formatBytes } from '@/utils/index'

export default {
  name: 'ClientDetailsDialog',
  props: {
    modelValue: {
      type: Boolean,
      default: false
    },
    clientData: {
      type: Object,
      default: null
    }
  },
  emits: ['update:modelValue', 'refresh', 'disconnect'],
  setup(props, { emit }) {
    const activeTab = ref('sessions')
    const loadingSessions = ref(false)
    const loadingLogs = ref(false)
    const sessionHistory = ref([])
    const connectionLogs = ref([])

    const dialogVisible = computed({
      get: () => props.modelValue,
      set: (value) => emit('update:modelValue', value)
    })

    const dialogTitle = computed(() => {
      return props.clientData 
        ? `客户端详情 - ${props.clientData.clientId}`
        : '客户端详情'
    })

    // 监听对话框显示状态
    watch(dialogVisible, (visible) => {
      if (visible && props.clientData) {
        loadClientDetails()
      }
    })

    // 获取状态标签类型
    const getStatusTagType = (status) => {
      const statusMap = {
        online: 'success',
        offline: 'info',
        reconnecting: 'warning',
        error: 'danger'
      }
      return statusMap[status] || 'info'
    }

    // 获取状态文本
    const getStatusText = (status) => {
      const statusMap = {
        online: '在线',
        offline: '离线',
        reconnecting: '重连中',
        error: '错误'
      }
      return statusMap[status] || '未知'
    }

    // 获取会话状态类型
    const getSessionStatusType = (status) => {
      const statusMap = {
        completed: 'success',
        active: 'primary',
        failed: 'danger',
        terminated: 'warning'
      }
      return statusMap[status] || 'info'
    }

    // 获取会话状态文本
    const getSessionStatusText = (status) => {
      const statusMap = {
        completed: '已完成',
        active: '进行中',
        failed: '失败',
        terminated: '已终止'
      }
      return statusMap[status] || '未知'
    }

    // 获取日志级别类型
    const getLogLevelType = (level) => {
      const levelMap = {
        ERROR: 'danger',
        WARN: 'warning',
        INFO: 'primary',
        DEBUG: 'info'
      }
      return levelMap[level] || 'info'
    }

    // 格式化持续时间
    const formatDuration = (seconds) => {
      if (!seconds) return '0秒'
      
      const hours = Math.floor(seconds / 3600)
      const minutes = Math.floor((seconds % 3600) / 60)
      const secs = seconds % 60
      
      if (hours > 0) {
        return `${hours}小时${minutes}分钟${secs}秒`
      } else if (minutes > 0) {
        return `${minutes}分钟${secs}秒`
      } else {
        return `${secs}秒`
      }
    }

    // 加载客户端详情
    const loadClientDetails = async () => {
      try {
        // 模拟加载会话历史
        loadingSessions.value = true
        await new Promise(resolve => setTimeout(resolve, 500))
        sessionHistory.value = [
          {
            sessionId: 'sess_001',
            startTime: Date.now() - 3600000,
            endTime: Date.now() - 3000000,
            duration: 600,
            apduCount: 150,
            status: 'completed'
          },
          {
            sessionId: 'sess_002',
            startTime: Date.now() - 1800000,
            endTime: Date.now() - 1200000,
            duration: 600,
            apduCount: 89,
            status: 'completed'
          }
        ]
        loadingSessions.value = false

        // 模拟加载连接日志
        loadingLogs.value = true
        await new Promise(resolve => setTimeout(resolve, 300))
        connectionLogs.value = [
          {
            timestamp: Date.now() - 300000,
            event: 'CONNECT',
            level: 'INFO',
            message: '客户端成功连接到服务器'
          },
          {
            timestamp: Date.now() - 600000,
            event: 'DISCONNECT',
            level: 'WARN',
            message: '客户端断开连接'
          }
        ]
        loadingLogs.value = false
      } catch (error) {
        console.error('加载客户端详情失败:', error)
        ElMessage.error('加载客户端详情失败')
      }
    }

    // 刷新数据
    const refreshData = () => {
      if (props.clientData) {
        loadClientDetails()
        emit('refresh')
      }
    }

    // 断开客户端连接
    const disconnectClient = async () => {
      try {
        await ElMessageBox.confirm(
          `确定要断开客户端 ${props.clientData.clientId} 的连接吗？`,
          '确认断开连接',
          {
            confirmButtonText: '确定',
            cancelButtonText: '取消',
            type: 'warning'
          }
        )
        
        emit('disconnect', props.clientData.clientId)
        ElMessage.success('客户端连接已断开')
        closeDialog()
      } catch (error) {
        if (error !== 'cancel') {
          console.error('断开连接失败:', error)
          ElMessage.error('断开连接失败')
        }
      }
    }

    // 关闭对话框
    const closeDialog = () => {
      dialogVisible.value = false
    }

    return {
      activeTab,
      loadingSessions,
      loadingLogs,
      sessionHistory,
      connectionLogs,
      dialogVisible,
      dialogTitle,
      getStatusTagType,
      getStatusText,
      getSessionStatusType,
      getSessionStatusText,
      getLogLevelType,
      formatDuration,
      refreshData,
      disconnectClient,
      closeDialog,
      formatDateTime,
      formatBytes
    }
  }
}
</script>

<style lang="scss" scoped>
.client-details {
  .mb-4 {
    margin-bottom: 16px;
  }

  .metric-item {
    display: flex;
    justify-content: space-between;
    margin-bottom: 8px;
    
    .metric-label {
      color: #606266;
      font-size: 14px;
    }
    
    .metric-value {
      font-weight: 600;
      color: #303133;
    }
  }
}

.dialog-footer {
  display: flex;
  justify-content: flex-end;
  gap: 12px;
}
</style> 