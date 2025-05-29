<template>
  <el-card class="metric-card" :class="type">
    <div class="metric-content">
      <div class="metric-icon">
        <el-icon :size="32">
          <component :is="icon" />
        </el-icon>
      </div>
      <div class="metric-info">
        <div class="metric-value">{{ value }}</div>
        <div class="metric-title">{{ title }}</div>
        <div v-if="subtitle" class="metric-subtitle">{{ subtitle }}</div>
      </div>
    </div>
    <div v-if="trend" class="metric-trend" :class="trend.type">
      <el-icon>
        <component :is="trend.type === 'up' ? ArrowUp : ArrowDown" />
      </el-icon>
      <span>{{ trend.value }}</span>
    </div>
  </el-card>
</template>

<script>
import { ArrowUp, ArrowDown } from '@element-plus/icons-vue'

export default {
  name: 'MetricCard',
  components: {
    ArrowUp,
    ArrowDown
  },
  props: {
    title: {
      type: String,
      required: true
    },
    value: {
      type: [String, Number],
      required: true
    },
    subtitle: {
      type: String,
      default: ''
    },
    icon: {
      type: [String, Object],
      default: 'InfoFilled'
    },
    type: {
      type: String,
      default: 'default', // default, primary, success, warning, danger
      validator: (value) => ['default', 'primary', 'success', 'warning', 'danger'].includes(value)
    },
    trend: {
      type: Object,
      default: null // { type: 'up|down', value: '12%' }
    }
  }
}
</script>

<style lang="scss" scoped>
.metric-card {
  position: relative;
  transition: transform 0.2s ease;

  &:hover {
    transform: translateY(-2px);
  }

  &.primary {
    border-left: 4px solid #409EFF;
  }

  &.success {
    border-left: 4px solid #67C23A;
  }

  &.warning {
    border-left: 4px solid #E6A23C;
  }

  &.danger {
    border-left: 4px solid #F56C6C;
  }

  .metric-content {
    display: flex;
    align-items: center;
    gap: 16px;

    .metric-icon {
      color: #909399;
      
      .primary & {
        color: #409EFF;
      }
      
      .success & {
        color: #67C23A;
      }
      
      .warning & {
        color: #E6A23C;
      }
      
      .danger & {
        color: #F56C6C;
      }
    }

    .metric-info {
      flex: 1;

      .metric-value {
        font-size: 24px;
        font-weight: 600;
        color: #303133;
        margin-bottom: 4px;
      }

      .metric-title {
        font-size: 14px;
        color: #606266;
        margin-bottom: 2px;
      }

      .metric-subtitle {
        font-size: 12px;
        color: #909399;
      }
    }
  }

  .metric-trend {
    position: absolute;
    top: 12px;
    right: 12px;
    display: flex;
    align-items: center;
    gap: 2px;
    font-size: 12px;
    font-weight: 600;

    &.up {
      color: #67C23A;
    }

    &.down {
      color: #F56C6C;
    }
  }
}
</style> 