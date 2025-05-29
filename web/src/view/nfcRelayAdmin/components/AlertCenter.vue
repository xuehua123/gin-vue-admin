<!--
  告警通知中心组件
  支持告警列表、实时通知、告警确认和筛选
-->
<template>
  <div class="alert-center">
    <!-- 告警中心标题栏 -->
    <div class="alert-header">
      <div class="header-left">
        <el-icon class="alert-icon" :class="alertIconClass">
          <component :is="getAlertIcon()" />
        </el-icon>
        <span class="alert-title">告警中心</span>
        <el-badge 
          v-if="unacknowledgedCount > 0" 
          :value="unacknowledgedCount" 
          class="alert-badge"
        />
      </div>
      
      <div class="header-right">
        <!-- 告警级别筛选 -->
        <el-select 
          v-model="selectedLevel" 
          placeholder="筛选级别"
          size="small"
          style="width: 120px"
          @change="onLevelChange"
        >
          <el-option label="全部" value="" />
          <el-option label="严重" value="critical">
            <el-icon color="#f56c6c"><Warning /></el-icon>
            严重
          </el-option>
          <el-option label="错误" value="error">
            <el-icon color="#e6a23c"><CircleClose /></el-icon>
            错误
          </el-option>
          <el-option label="警告" value="warning">
            <el-icon color="#f56c6c"><WarningFilled /></el-icon>
            警告
          </el-option>
          <el-option label="信息" value="info">
            <el-icon color="#409eff"><InfoFilled /></el-icon>
            信息
          </el-option>
        </el-select>
        
        <!-- 状态筛选 -->
        <el-select 
          v-model="selectedStatus" 
          placeholder="筛选状态"
          size="small"
          style="width: 120px"
          @change="onStatusChange"
        >
          <el-option label="全部" value="" />
          <el-option label="未确认" value="unacknowledged" />
          <el-option label="已确认" value="acknowledged" />
        </el-select>
        
        <!-- 批量操作 -->
        <el-dropdown @command="handleBatchCommand">
          <el-button size="small" :disabled="selectedAlerts.length === 0">
            批量操作
            <el-icon><ArrowDown /></el-icon>
          </el-button>
          <template #dropdown>
            <el-dropdown-menu>
              <el-dropdown-item command="acknowledge">
                <el-icon><Check /></el-icon>
                确认选中
              </el-dropdown-item>
              <el-dropdown-item command="delete">
                <el-icon><Delete /></el-icon>
                删除选中
              </el-dropdown-item>
              <el-dropdown-item command="export">
                <el-icon><Download /></el-icon>
                导出选中
              </el-dropdown-item>
            </el-dropdown-menu>
          </template>
        </el-dropdown>
        
        <!-- 刷新按钮 -->
        <el-button 
          size="small" 
          @click="refreshAlerts"
          :loading="loading"
          circle
        >
          <el-icon><Refresh /></el-icon>
        </el-button>
        
        <!-- 设置按钮 -->
        <el-button 
          size="small" 
          @click="showSettings = true"
          circle
        >
          <el-icon><Setting /></el-icon>
        </el-button>
      </div>
    </div>

    <!-- 告警列表 -->
    <div class="alert-content">
      <el-table
        ref="alertTable"
        v-loading="loading"
        :data="filteredAlerts"
        @selection-change="onSelectionChange"
        height="400"
        class="alert-table"
      >
        <!-- 选择列 -->
        <el-table-column
          type="selection"
          width="50"
          :selectable="row => !row.acknowledged"
        />
        
        <!-- 告警级别 -->
        <el-table-column prop="level" label="级别" width="80">
          <template #default="{ row }">
            <el-tag 
              :type="getLevelTagType(row.level)" 
              size="small"
              :icon="getLevelIcon(row.level)"
            >
              {{ getLevelText(row.level) }}
            </el-tag>
          </template>
        </el-table-column>
        
        <!-- 告警类型 -->
        <el-table-column prop="type" label="类型" width="100">
          <template #default="{ row }">
            <el-tag size="small" effect="plain">
              {{ getTypeText(row.type) }}
            </el-tag>
          </template>
        </el-table-column>
        
        <!-- 告警消息 -->
        <el-table-column prop="message" label="消息" min-width="200">
          <template #default="{ row }">
            <div class="alert-message">
              <span class="message-text">{{ row.message }}</span>
              <el-button 
                v-if="row.details" 
                link 
                size="small"
                @click="showAlertDetails(row)"
              >
                详情
              </el-button>
            </div>
          </template>
        </el-table-column>
        
        <!-- 来源 -->
        <el-table-column prop="source" label="来源" width="120">
          <template #default="{ row }">
            <el-text size="small" type="info">{{ row.source }}</el-text>
          </template>
        </el-table-column>
        
        <!-- 时间 -->
        <el-table-column prop="timestamp" label="时间" width="160">
          <template #default="{ row }">
            <el-tooltip :content="formatTime(row.timestamp, 'YYYY-MM-DD HH:mm:ss')">
              <span class="timestamp">{{ getRelativeTime(row.timestamp) }}</span>
            </el-tooltip>
          </template>
        </el-table-column>
        
        <!-- 状态 -->
        <el-table-column prop="acknowledged" label="状态" width="100">
          <template #default="{ row }">
            <el-tag 
              :type="row.acknowledged ? 'success' : 'warning'" 
              size="small"
            >
              {{ row.acknowledged ? '已确认' : '未确认' }}
            </el-tag>
          </template>
        </el-table-column>
        
        <!-- 操作 -->
        <el-table-column label="操作" width="120" fixed="right">
          <template #default="{ row }">
            <el-button
              v-if="!row.acknowledged"
              link
              type="primary"
              size="small"
              @click="acknowledgeAlert(row)"
            >
              确认
            </el-button>
            <el-button
              link
              type="danger"
              size="small"
              @click="deleteAlert(row)"
            >
              删除
            </el-button>
          </template>
        </el-table-column>
      </el-table>
      
      <!-- 分页 -->
      <div class="pagination-wrapper">
        <el-pagination
          v-model:current-page="pagination.current"
          v-model:page-size="pagination.size"
          :total="pagination.total"
          :page-sizes="[10, 20, 50, 100]"
          layout="total, sizes, prev, pager, next, jumper"
          @size-change="onPageSizeChange"
          @current-change="onPageChange"
        />
      </div>
    </div>

    <!-- 告警详情对话框 -->
    <el-dialog
      v-model="detailsVisible"
      title="告警详情"
      width="600px"
      @close="selectedAlert = null"
    >
      <div v-if="selectedAlert" class="alert-details">
        <el-descriptions :column="2" border>
          <el-descriptions-item label="告警ID">
            {{ selectedAlert.id }}
          </el-descriptions-item>
          <el-descriptions-item label="级别">
            <el-tag :type="getLevelTagType(selectedAlert.level)">
              {{ getLevelText(selectedAlert.level) }}
            </el-tag>
          </el-descriptions-item>
          <el-descriptions-item label="类型">
            {{ getTypeText(selectedAlert.type) }}
          </el-descriptions-item>
          <el-descriptions-item label="来源">
            {{ selectedAlert.source }}
          </el-descriptions-item>
          <el-descriptions-item label="时间" :span="2">
            {{ formatTime(selectedAlert.timestamp, 'YYYY-MM-DD HH:mm:ss') }}
          </el-descriptions-item>
          <el-descriptions-item label="消息" :span="2">
            {{ selectedAlert.message }}
          </el-descriptions-item>
        </el-descriptions>
        
        <div v-if="selectedAlert.details" class="alert-extra-details">
          <h4>详细信息</h4>
          <pre>{{ JSON.stringify(selectedAlert.details, null, 2) }}</pre>
        </div>
      </div>
      
      <template #footer>
        <el-button @click="detailsVisible = false">关闭</el-button>
        <el-button 
          v-if="selectedAlert && !selectedAlert.acknowledged"
          type="primary"
          @click="acknowledgeAlert(selectedAlert); detailsVisible = false"
        >
          确认告警
        </el-button>
      </template>
    </el-dialog>

    <!-- 告警设置对话框 -->
    <el-dialog
      v-model="showSettings"
      title="告警设置"
      width="500px"
    >
      <el-form :model="alertSettings" label-width="120px">
        <el-form-item label="声音提醒">
          <el-switch v-model="alertSettings.soundEnabled" />
        </el-form-item>
        <el-form-item label="桌面通知">
          <el-switch v-model="alertSettings.desktopNotification" />
        </el-form-item>
        <el-form-item label="自动刷新">
          <el-switch v-model="alertSettings.autoRefresh" />
        </el-form-item>
        <el-form-item label="刷新间隔">
          <el-input-number
            v-model="alertSettings.refreshInterval"
            :min="5"
            :max="300"
            :step="5"
            :disabled="!alertSettings.autoRefresh"
          />
          <span style="margin-left: 8px;">秒</span>
        </el-form-item>
        <el-form-item label="最大显示条数">
          <el-input-number
            v-model="alertSettings.maxDisplayCount"
            :min="10"
            :max="500"
            :step="10"
          />
        </el-form-item>
      </el-form>
      
      <template #footer>
        <el-button @click="showSettings = false">取消</el-button>
        <el-button type="primary" @click="saveSettings">保存</el-button>
      </template>
    </el-dialog>
  </div>
</template>

<script setup>
import { ref, computed, onMounted, onUnmounted, watch } from 'vue'
import { ElMessage, ElMessageBox, ElNotification } from 'element-plus'
import { 
  Warning, 
  CircleClose, 
  WarningFilled, 
  InfoFilled,
  Check,
  Delete,
  Download,
  ArrowDown,
  Refresh,
  Setting,
  Bell,
  BellFilled
} from '@element-plus/icons-vue'
import { formatTime } from '@/utils/format'

const props = defineProps({
  alerts: {
    type: Array,
    default: () => []
  },
  autoRefresh: {
    type: Boolean,
    default: true
  }
})

const emit = defineEmits(['alert-acknowledged', 'alert-deleted', 'refresh-alerts'])

// 响应式数据
const loading = ref(false)
const selectedLevel = ref('')
const selectedStatus = ref('')
const selectedAlerts = ref([])
const detailsVisible = ref(false)
const selectedAlert = ref(null)
const showSettings = ref(false)
const refreshTimer = ref(null)

// 分页
const pagination = ref({
  current: 1,
  size: 20,
  total: 0
})

// 告警设置
const alertSettings = ref({
  soundEnabled: true,
  desktopNotification: true,
  autoRefresh: true,
  refreshInterval: 30,
  maxDisplayCount: 100
})

// 计算属性
const unacknowledgedCount = computed(() => {
  return props.alerts.filter(alert => !alert.acknowledged).length
})

const filteredAlerts = computed(() => {
  let filtered = [...props.alerts]
  
  // 按级别筛选
  if (selectedLevel.value) {
    filtered = filtered.filter(alert => alert.level === selectedLevel.value)
  }
  
  // 按状态筛选
  if (selectedStatus.value) {
    const isAcknowledged = selectedStatus.value === 'acknowledged'
    filtered = filtered.filter(alert => alert.acknowledged === isAcknowledged)
  }
  
  // 更新总数
  pagination.value.total = filtered.length
  
  // 分页
  const start = (pagination.value.current - 1) * pagination.value.size
  const end = start + pagination.value.size
  return filtered.slice(start, end)
})

const alertIconClass = computed(() => {
  if (unacknowledgedCount.value === 0) return 'icon-normal'
  
  const criticalCount = props.alerts.filter(a => !a.acknowledged && a.level === 'critical').length
  const errorCount = props.alerts.filter(a => !a.acknowledged && a.level === 'error').length
  
  if (criticalCount > 0) return 'icon-critical'
  if (errorCount > 0) return 'icon-error'
  return 'icon-warning'
})

// 方法
const getAlertIcon = () => {
  return unacknowledgedCount.value > 0 ? BellFilled : Bell
}

const getLevelTagType = (level) => {
  const types = {
    critical: 'danger',
    error: 'danger',
    warning: 'warning',
    info: 'info'
  }
  return types[level] || 'info'
}

const getLevelIcon = (level) => {
  const icons = {
    critical: Warning,
    error: CircleClose,
    warning: WarningFilled,
    info: InfoFilled
  }
  return icons[level] || InfoFilled
}

const getLevelText = (level) => {
  const texts = {
    critical: '严重',
    error: '错误',
    warning: '警告',
    info: '信息'
  }
  return texts[level] || level
}

const getTypeText = (type) => {
  const texts = {
    performance: '性能',
    connection: '连接',
    apdu: 'APDU',
    system: '系统',
    security: '安全'
  }
  return texts[type] || type
}

const getRelativeTime = (timestamp) => {
  const now = new Date()
  const time = new Date(timestamp)
  const diff = now - time
  
  if (diff < 60000) return '刚刚'
  if (diff < 3600000) return `${Math.floor(diff / 60000)}分钟前`
  if (diff < 86400000) return `${Math.floor(diff / 3600000)}小时前`
  return `${Math.floor(diff / 86400000)}天前`
}

const onLevelChange = () => {
  pagination.value.current = 1
}

const onStatusChange = () => {
  pagination.value.current = 1
}

const onPageSizeChange = () => {
  pagination.value.current = 1
}

const onPageChange = () => {
  // 页面变化时的处理
}

const onSelectionChange = (selection) => {
  selectedAlerts.value = selection
}

const showAlertDetails = (alert) => {
  selectedAlert.value = alert
  detailsVisible.value = true
}

const acknowledgeAlert = async (alert) => {
  try {
    loading.value = true
    // TODO: 调用API确认告警
    alert.acknowledged = true
    alert.acknowledgedAt = new Date()
    
    emit('alert-acknowledged', alert)
    ElMessage.success('告警已确认')
  } catch (error) {
    ElMessage.error('确认告警失败: ' + error.message)
  } finally {
    loading.value = false
  }
}

const deleteAlert = async (alert) => {
  try {
    await ElMessageBox.confirm('确定要删除这条告警吗？', '确认删除', {
      type: 'warning'
    })
    
    emit('alert-deleted', alert)
    ElMessage.success('告警已删除')
  } catch {
    // 用户取消删除
  }
}

const handleBatchCommand = async (command) => {
  if (selectedAlerts.value.length === 0) return
  
  try {
    switch (command) {
      case 'acknowledge':
        await ElMessageBox.confirm(`确定要确认选中的 ${selectedAlerts.value.length} 条告警吗？`, '批量确认')
        selectedAlerts.value.forEach(alert => {
          alert.acknowledged = true
          alert.acknowledgedAt = new Date()
          emit('alert-acknowledged', alert)
        })
        ElMessage.success('批量确认成功')
        break
        
      case 'delete':
        await ElMessageBox.confirm(`确定要删除选中的 ${selectedAlerts.value.length} 条告警吗？`, '批量删除', {
          type: 'warning'
        })
        selectedAlerts.value.forEach(alert => {
          emit('alert-deleted', alert)
        })
        ElMessage.success('批量删除成功')
        break
        
      case 'export':
        // TODO: 实现导出功能
        ElMessage.info('导出功能开发中...')
        break
    }
  } catch {
    // 用户取消操作
  }
}

const refreshAlerts = () => {
  emit('refresh-alerts')
}

const saveSettings = () => {
  // 保存设置到localStorage
  localStorage.setItem('alertSettings', JSON.stringify(alertSettings.value))
  
  // 重新设置自动刷新
  setupAutoRefresh()
  
  showSettings.value = false
  ElMessage.success('设置已保存')
}

const setupAutoRefresh = () => {
  if (refreshTimer.value) {
    clearInterval(refreshTimer.value)
    refreshTimer.value = null
  }
  
  if (alertSettings.value.autoRefresh) {
    refreshTimer.value = setInterval(() => {
      refreshAlerts()
    }, alertSettings.value.refreshInterval * 1000)
  }
}

const loadSettings = () => {
  const saved = localStorage.getItem('alertSettings')
  if (saved) {
    try {
      Object.assign(alertSettings.value, JSON.parse(saved))
    } catch (error) {
      console.error('加载告警设置失败:', error)
    }
  }
}

const playAlertSound = () => {
  if (alertSettings.value.soundEnabled) {
    // 播放告警声音
    const audio = new Audio('/sounds/alert.mp3')
    audio.play().catch(() => {
      // 忽略音频播放失败
    })
  }
}

const showDesktopNotification = (alert) => {
  if (alertSettings.value.desktopNotification && Notification.permission === 'granted') {
    new Notification(`${getLevelText(alert.level)}告警`, {
      body: alert.message,
      icon: '/favicon.ico'
    })
  }
}

// 监听告警变化
watch(() => props.alerts, (newAlerts, oldAlerts) => {
  if (oldAlerts && newAlerts.length > oldAlerts.length) {
    // 有新告警
    const newAlert = newAlerts[newAlerts.length - 1]
    if (!newAlert.acknowledged) {
      playAlertSound()
      showDesktopNotification(newAlert)
    }
  }
}, { deep: true })

// 生命周期
onMounted(() => {
  loadSettings()
  setupAutoRefresh()
  
  // 请求桌面通知权限
  if ('Notification' in window && Notification.permission === 'default') {
    Notification.requestPermission()
  }
})

onUnmounted(() => {
  if (refreshTimer.value) {
    clearInterval(refreshTimer.value)
  }
})
</script>

<style scoped lang="scss">
.alert-center {
  background: #ffffff;
  border-radius: 8px;
  box-shadow: 0 2px 8px rgba(0, 0, 0, 0.1);
  
  .alert-header {
    display: flex;
    justify-content: space-between;
    align-items: center;
    padding: 12px 16px;
    border-bottom: 1px solid #f0f0f0;
    
    .header-left {
      display: flex;
      align-items: center;
      gap: 8px;
      
      .alert-icon {
        font-size: 18px;
        transition: all 0.3s ease;
        
        &.icon-normal {
          color: #909399;
        }
        
        &.icon-warning {
          color: #e6a23c;
          animation: pulse 2s infinite;
        }
        
        &.icon-error {
          color: #f56c6c;
          animation: shake 1s infinite;
        }
        
        &.icon-critical {
          color: #f56c6c;
          animation: critical-blink 0.5s infinite;
        }
      }
      
      .alert-title {
        font-weight: 600;
        color: #303133;
        font-size: 16px;
      }
      
      .alert-badge {
        margin-left: 4px;
      }
    }
    
    .header-right {
      display: flex;
      align-items: center;
      gap: 8px;
    }
  }
  
  .alert-content {
    .alert-table {
      :deep(.el-table__header) {
        background-color: #fafafa;
      }
      
      .alert-message {
        display: flex;
        justify-content: space-between;
        align-items: center;
        
        .message-text {
          flex: 1;
          margin-right: 8px;
        }
      }
      
      .timestamp {
        font-size: 12px;
        color: #909399;
      }
    }
    
    .pagination-wrapper {
      padding: 16px;
      display: flex;
      justify-content: center;
    }
  }
  
  .alert-details {
    .alert-extra-details {
      margin-top: 16px;
      
      h4 {
        margin-bottom: 8px;
        color: #303133;
      }
      
      pre {
        background: #f5f7fa;
        padding: 12px;
        border-radius: 4px;
        font-size: 12px;
        max-height: 200px;
        overflow-y: auto;
      }
    }
  }
}

// 动画效果
@keyframes pulse {
  0%, 100% { opacity: 1; }
  50% { opacity: 0.5; }
}

@keyframes shake {
  0%, 100% { transform: translateX(0); }
  25% { transform: translateX(-2px); }
  75% { transform: translateX(2px); }
}

@keyframes critical-blink {
  0%, 100% { opacity: 1; }
  50% { opacity: 0.3; }
}

// 暗色主题
.dark .alert-center {
  background: #1f1f1f;
  border-color: #303030;
  
  .alert-header {
    border-color: #303030;
    
    .alert-title {
      color: #e5eaf3;
    }
  }
  
  .alert-table {
    :deep(.el-table__header) {
      background-color: #262626;
    }
  }
  
  .alert-details {
    .alert-extra-details {
      h4 {
        color: #e5eaf3;
      }
      
      pre {
        background: #262626;
        color: #e5eaf3;
      }
    }
  }
}
</style> 