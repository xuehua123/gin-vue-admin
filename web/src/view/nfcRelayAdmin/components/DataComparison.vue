<template>
  <el-card class="data-comparison">
    <template #header>
      <div class="comparison-header">
        <span class="comparison-title">{{ title }}</span>
        <el-select v-model="selectedPeriod" size="small" style="width: 120px">
          <el-option label="今日 vs 昨日" value="day" />
          <el-option label="本周 vs 上周" value="week" />
          <el-option label="本月 vs 上月" value="month" />
        </el-select>
      </div>
    </template>
    
    <div class="comparison-content">
      <div 
        v-for="metric in processedData"
        :key="metric.name"
        class="comparison-item"
      >
        <div class="metric-header">
          <span class="metric-name">{{ metric.name }}</span>
          <div class="metric-trend" :class="metric.trend">
            <el-icon>
              <component :is="metric.trend === 'up' ? ArrowUp : metric.trend === 'down' ? ArrowDown : Minus" />
            </el-icon>
            <span>{{ metric.changePercent }}%</span>
          </div>
        </div>
        
        <div class="metric-values">
          <div class="current-value">
            <span class="label">当前:</span>
            <span class="value">{{ metric.current }}</span>
          </div>
          <div class="previous-value">
            <span class="label">对比:</span>
            <span class="value">{{ metric.previous }}</span>
          </div>
        </div>
        
        <div class="progress-bar">
          <el-progress 
            :percentage="metric.progressPercent" 
            :show-text="false"
            :stroke-width="6"
            :color="getProgressColor(metric.trend)"
          />
        </div>
      </div>
    </div>
  </el-card>
</template>

<script>
import { ref, computed } from 'vue'
import { ArrowUp, ArrowDown, Minus } from '@element-plus/icons-vue'

export default {
  name: 'DataComparison',
  components: {
    ArrowUp,
    ArrowDown,
    Minus
  },
  props: {
    title: {
      type: String,
      default: '数据对比'
    },
    data: {
      type: Array,
      default: () => []
    }
  },
  setup(props) {
    const selectedPeriod = ref('day')

    const processedData = computed(() => {
      return props.data.map(item => {
        const current = Number(item.current) || 0
        const previous = Number(item.previous) || 0
        const change = current - previous
        const changePercent = previous !== 0 ? Math.abs((change / previous) * 100).toFixed(1) : 0
        
        let trend = 'same'
        if (change > 0) trend = 'up'
        else if (change < 0) trend = 'down'
        
        const maxValue = Math.max(current, previous)
        const progressPercent = maxValue > 0 ? Math.round((current / maxValue) * 100) : 0

        return {
          ...item,
          changePercent,
          trend,
          progressPercent
        }
      })
    })

    const getProgressColor = (trend) => {
      const colorMap = {
        up: '#67C23A',
        down: '#F56C6C',
        same: '#909399'
      }
      return colorMap[trend] || '#909399'
    }

    return {
      selectedPeriod,
      processedData,
      getProgressColor
    }
  }
}
</script>

<style lang="scss" scoped>
.data-comparison {
  .comparison-header {
    display: flex;
    justify-content: space-between;
    align-items: center;

    .comparison-title {
      font-size: 16px;
      font-weight: 600;
      color: #303133;
    }
  }

  .comparison-content {
    .comparison-item {
      padding: 16px 0;
      border-bottom: 1px solid #f0f0f0;

      &:last-child {
        border-bottom: none;
      }

      .metric-header {
        display: flex;
        justify-content: space-between;
        align-items: center;
        margin-bottom: 12px;

        .metric-name {
          font-size: 14px;
          color: #606266;
          font-weight: 500;
        }

        .metric-trend {
          display: flex;
          align-items: center;
          gap: 4px;
          font-size: 12px;
          font-weight: 600;

          &.up {
            color: #67C23A;
          }

          &.down {
            color: #F56C6C;
          }

          &.same {
            color: #909399;
          }
        }
      }

      .metric-values {
        display: flex;
        justify-content: space-between;
        margin-bottom: 8px;

        .current-value,
        .previous-value {
          display: flex;
          align-items: center;
          gap: 8px;
          font-size: 13px;

          .label {
            color: #909399;
          }

          .value {
            color: #303133;
            font-weight: 600;
          }
        }

        .current-value .value {
          color: #409EFF;
        }
      }

      .progress-bar {
        margin-top: 8px;
      }
    }
  }
}
</style> 