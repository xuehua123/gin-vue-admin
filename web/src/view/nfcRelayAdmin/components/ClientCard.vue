<template>
  <el-card 
    :class="['client-card', { 
      'selected': selected, 
      'online': client.status === 'online',
      'offline': client.status === 'offline',
      'reconnecting': client.status === 'reconnecting',
      'error': client.status === 'error'
    }]"
    shadow="hover"
    @click="handleCardClick"
  >
    <!-- 卡片头部 -->
    <template #header>
      <div class="card-header">
        <div class="client-info">
          <div class="status-indicator">
            <el-icon 
              :class="getStatusClass(client.status)"
              :size="12"
            >
              <component :is="getStatusIcon(client.status)" />
            </el-icon>
            <span class="status-text">{{ getStatusText(client.status) }}</span>
          </div>
          <div class="client-id">{{ client.clientId }}</div>
        </div>
        
        <div class="card-actions">
          <el-dropdown @command="handleAction" trigger="click">
            <el-button 
              type="text" 
              size="small"
              @click.stop
            >
              <el-icon><MoreFilled /></el-icon>
            </el-button>
            <template #dropdown>
              <el-dropdown-menu>
                <el-dropdown-item command="details">
                  <el-icon><View /></el-icon>
                  查看详情
                </el-dropdown-item>
                <el-dropdown-item command="history">
                  <el-icon><Clock /></el-icon>
                  连接历史
                </el-dropdown-item>
                <el-dropdown-item 
                  v-if="client.status === 'online'"
                  command="disconnect"
                  divided
                >
                  <el-icon><Link /></el-icon>
                  断开连接
                </el-dropdown-item>
                <el-dropdown-item 
                  command="ban"
                  type="danger"
                >
                  <el-icon><Lock /></el-icon>
                  封禁客户端
                </el-dropdown-item>
              </el-dropdown-menu>
            </template>
          </el-dropdown>
        </div>
      </div>
    </template>

    <!-- 卡片内容 -->
    <div class="card-content">
      <!-- 设备信息 -->
      <div class="device-info">
        <div class="device-name">
          <el-icon><Monitor /></el-icon>
          <span>{{ client.deviceName || '未知设备' }}</span>
        </div>
        <div class="device-type">
          <el-tag :type="getDeviceTypeTagType(client.deviceType)" size="small">
            {{ getDeviceTypeText(client.deviceType) }}
          </el-tag>
        </div>
      </div>

      <!-- 网络信息 -->
      <div class="network-info">
        <div class="ip-address">
          <el-icon><Connection /></el-icon>
          <span>{{ client.ipAddress }}</span>
        </div>
        <div class="location" v-if="client.location">
          <el-icon><Location /></el-icon>
          <span>{{ client.location }}</span>
        </div>
      </div>

      <!-- 连接信息 -->
      <div class="connection-info">
        <div class="connect-time">
          <span class="label">连接时间:</span>
          <span class="value">{{ formatDateTime(client.connectedAt) }}</span>
        </div>
        <div class="online-duration" v-if="client.status === 'online'">
          <span class="label">在线时长:</span>
          <span class="value">{{ formatDuration(client.onlineDuration) }}</span>
        </div>
        <div class="last-activity" v-else>
          <span class="label">最后活动:</span>
          <span class="value">{{ formatDateTime(client.lastActiveAt) }}</span>
        </div>
      </div>

      <!-- 性能指标 -->
      <div class="performance-metrics" v-if="client.status === 'online'">
        <div class="metric-item">
          <span class="metric-label">延迟:</span>
          <span class="metric-value" :class="getLatencyClass(client.latency)">
            {{ client.latency || 0 }}ms
          </span>
        </div>
        <div class="metric-item">
          <span class="metric-label">会话:</span>
          <span class="metric-value">{{ client.sessionCount || 0 }}</span>
        </div>
        <div class="metric-item">
          <span class="metric-label">APDU:</span>
          <span class="metric-value">{{ client.apduCount || 0 }}</span>
        </div>
      </div>

      <!-- 版本信息 -->
      <div class="version-info">
        <div class="app-version">
          <span class="label">应用版本:</span>
          <span class="value">{{ client.appVersion || 'N/A' }}</span>
        </div>
        <div class="system-version">
          <span class="label">系统版本:</span>
          <span class="value">{{ client.systemVersion || 'N/A' }}</span>
        </div>
      </div>
    </div>

    <!-- 选中状态指示器 -->
    <div v-if="selected" class="selection-indicator">
      <el-icon><Check /></el-icon>
    </div>
  </el-card>
</template>

<script>
import { 
  MoreFilled, 
  View, 
  Clock, 
  Link, 
  Lock, 
  Monitor, 
  Connection, 
  Location, 
  Check,
  CircleFilled,
  WarningFilled,
  InfoFilled,
  Close
} from '@element-plus/icons-vue'
import { formatDateTime } from '@/utils/index'

export default {
  name: 'ClientCard',
  components: {
    MoreFilled,
    View,
    Clock,
    Link,
    Lock,
    Monitor,
    Connection,
    Location,
    Check,
    CircleFilled,
    WarningFilled,
    InfoFilled,
    Close
  },
  props: {
    client: {
      type: Object,
      required: true
    },
    selected: {
      type: Boolean,
      default: false
    }
  },
  emits: ['click', 'action'],
  setup(props, { emit }) {
    // 获取状态图标
    const getStatusIcon = (status) => {
      const iconMap = {
        online: CircleFilled,
        offline: InfoFilled,
        reconnecting: WarningFilled,
        error: Close
      }
      return iconMap[status] || InfoFilled
    }

    // 获取状态样式类
    const getStatusClass = (status) => {
      return `status-${status}`
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

    // 获取设备类型标签类型
    const getDeviceTypeTagType = (deviceType) => {
      const typeMap = {
        android: 'success',
        ios: 'primary',
        provider: 'warning',
        other: 'info'
      }
      return typeMap[deviceType] || 'info'
    }

    // 获取设备类型文本
    const getDeviceTypeText = (deviceType) => {
      const typeMap = {
        android: 'Android',
        ios: 'iOS',
        provider: 'Provider',
        other: '其他'
      }
      return typeMap[deviceType] || deviceType
    }

    // 获取延迟样式类
    const getLatencyClass = (latency) => {
      if (latency < 50) return 'latency-good'
      if (latency < 100) return 'latency-normal'
      return 'latency-poor'
    }

    // 格式化持续时间
    const formatDuration = (seconds) => {
      if (!seconds) return '0秒'
      
      const hours = Math.floor(seconds / 3600)
      const minutes = Math.floor((seconds % 3600) / 60)
      const secs = seconds % 60
      
      if (hours > 0) {
        return `${hours}h ${minutes}m`
      } else if (minutes > 0) {
        return `${minutes}m ${secs}s`
      } else {
        return `${secs}s`
      }
    }

    // 处理卡片点击
    const handleCardClick = () => {
      emit('click', props.client)
    }

    // 处理操作
    const handleAction = (command) => {
      emit('action', {
        action: command,
        client: props.client
      })
    }

    return {
      getStatusIcon,
      getStatusClass,
      getStatusText,
      getDeviceTypeTagType,
      getDeviceTypeText,
      getLatencyClass,
      formatDuration,
      formatDateTime,
      handleCardClick,
      handleAction
    }
  }
}
</script>

<style lang="scss" scoped>
.client-card {
  position: relative;
  transition: all 0.3s ease;
  cursor: pointer;
  margin-bottom: 16px;

  &:hover {
    transform: translateY(-2px);
    box-shadow: 0 8px 25px rgba(0, 0, 0, 0.1);
  }

  &.selected {
    border-color: #409EFF;
    box-shadow: 0 0 10px rgba(64, 158, 255, 0.3);
  }

  &.online {
    border-left: 4px solid #67C23A;
  }

  &.offline {
    border-left: 4px solid #909399;
  }

  &.reconnecting {
    border-left: 4px solid #E6A23C;
  }

  &.error {
    border-left: 4px solid #F56C6C;
  }

  .card-header {
    display: flex;
    justify-content: space-between;
    align-items: center;
    padding: 0;

    .client-info {
      flex: 1;

      .status-indicator {
        display: flex;
        align-items: center;
        margin-bottom: 4px;

        .status-text {
          margin-left: 4px;
          font-size: 12px;
          color: #606266;
        }
      }

      .client-id {
        font-weight: 600;
        color: #303133;
        font-size: 14px;
      }
    }

    .card-actions {
      opacity: 0;
      transition: opacity 0.3s ease;
    }
  }

  &:hover .card-actions {
    opacity: 1;
  }

  .card-content {
    .device-info {
      display: flex;
      justify-content: space-between;
      align-items: center;
      margin-bottom: 12px;

      .device-name {
        display: flex;
        align-items: center;
        font-size: 13px;
        color: #606266;

        .el-icon {
          margin-right: 4px;
        }
      }
    }

    .network-info {
      margin-bottom: 12px;

      .ip-address, .location {
        display: flex;
        align-items: center;
        font-size: 12px;
        color: #909399;
        margin-bottom: 4px;

        .el-icon {
          margin-right: 4px;
        }
      }
    }

    .connection-info {
      margin-bottom: 12px;

      .connect-time, .online-duration, .last-activity {
        display: flex;
        justify-content: space-between;
        font-size: 12px;
        margin-bottom: 4px;

        .label {
          color: #909399;
        }

        .value {
          color: #606266;
          font-weight: 500;
        }
      }
    }

    .performance-metrics {
      display: flex;
      justify-content: space-between;
      margin-bottom: 12px;

      .metric-item {
        text-align: center;
        flex: 1;

        .metric-label {
          display: block;
          font-size: 11px;
          color: #909399;
          margin-bottom: 2px;
        }

        .metric-value {
          font-size: 12px;
          font-weight: 600;

          &.latency-good {
            color: #67C23A;
          }

          &.latency-normal {
            color: #E6A23C;
          }

          &.latency-poor {
            color: #F56C6C;
          }
        }
      }
    }

    .version-info {
      .app-version, .system-version {
        display: flex;
        justify-content: space-between;
        font-size: 11px;
        margin-bottom: 2px;

        .label {
          color: #C0C4CC;
        }

        .value {
          color: #909399;
        }
      }
    }
  }

  .selection-indicator {
    position: absolute;
    top: 8px;
    right: 8px;
    width: 20px;
    height: 20px;
    background: #409EFF;
    border-radius: 50%;
    display: flex;
    align-items: center;
    justify-content: center;
    color: white;
    font-size: 12px;
  }

  // 状态图标颜色
  .status-online {
    color: #67C23A;
  }

  .status-offline {
    color: #909399;
  }

  .status-reconnecting {
    color: #E6A23C;
  }

  .status-error {
    color: #F56C6C;
  }
}
</style> 