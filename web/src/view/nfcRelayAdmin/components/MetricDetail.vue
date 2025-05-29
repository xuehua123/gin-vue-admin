<template>
  <el-card class="metric-detail">
    <template #header>
      <div class="detail-header">
        <span class="detail-title">{{ title }}</span>
        <el-tag v-if="status" :type="getStatusType(status)" size="small">
          {{ status }}
        </el-tag>
      </div>
    </template>
    
    <div class="detail-content">
      <div class="detail-value">
        <span class="value">{{ value }}</span>
        <span v-if="unit" class="unit">{{ unit }}</span>
      </div>
      
      <div v-if="description" class="detail-description">
        {{ description }}
      </div>
      
      <div v-if="metrics && metrics.length" class="detail-metrics">
        <div 
          v-for="metric in metrics"
          :key="metric.name"
          class="metric-item"
        >
          <span class="metric-name">{{ metric.name }}:</span>
          <span class="metric-value">{{ metric.value }}</span>
        </div>
      </div>
      
      <div v-if="trend" class="detail-trend">
        <span class="trend-label">变化趋势:</span>
        <span class="trend-value" :class="trend.direction">
          <el-icon>
            <component :is="trend.direction === 'up' ? ArrowUp : ArrowDown" />
          </el-icon>
          {{ trend.value }}
        </span>
      </div>
    </div>
  </el-card>
</template>

<script>
import { ArrowUp, ArrowDown } from '@element-plus/icons-vue'

export default {
  name: 'MetricDetail',
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
    unit: {
      type: String,
      default: ''
    },
    description: {
      type: String,
      default: ''
    },
    status: {
      type: String,
      default: ''
    },
    metrics: {
      type: Array,
      default: () => []
    },
    trend: {
      type: Object,
      default: null // { direction: 'up|down', value: '12%' }
    }
  },
  setup() {
    const getStatusType = (status) => {
      const statusMap = {
        normal: 'success',
        warning: 'warning',
        error: 'danger',
        info: 'info'
      }
      return statusMap[status] || 'info'
    }

    return {
      getStatusType
    }
  }
}
</script>

<style lang="scss" scoped>
.metric-detail {
  .detail-header {
    display: flex;
    justify-content: space-between;
    align-items: center;

    .detail-title {
      font-size: 16px;
      font-weight: 600;
      color: #303133;
    }
  }

  .detail-content {
    .detail-value {
      display: flex;
      align-items: baseline;
      margin-bottom: 12px;

      .value {
        font-size: 28px;
        font-weight: 700;
        color: #409EFF;
      }

      .unit {
        margin-left: 8px;
        font-size: 14px;
        color: #909399;
      }
    }

    .detail-description {
      font-size: 14px;
      color: #606266;
      margin-bottom: 16px;
      line-height: 1.5;
    }

    .detail-metrics {
      margin-bottom: 16px;

      .metric-item {
        display: flex;
        justify-content: space-between;
        padding: 6px 0;
        font-size: 13px;
        border-bottom: 1px solid #f0f0f0;

        &:last-child {
          border-bottom: none;
        }

        .metric-name {
          color: #909399;
        }

        .metric-value {
          color: #303133;
          font-weight: 500;
        }
      }
    }

    .detail-trend {
      display: flex;
      align-items: center;
      gap: 8px;
      font-size: 13px;

      .trend-label {
        color: #909399;
      }

      .trend-value {
        display: flex;
        align-items: center;
        gap: 4px;
        font-weight: 600;

        &.up {
          color: #67C23A;
        }

        &.down {
          color: #F56C6C;
        }
      }
    }
  }
}
</style> 