<template>
  <div class="line-chart">
    <div class="chart-placeholder">
      <el-icon class="chart-icon"><TrendCharts /></el-icon>
      <div class="chart-text">{{ title || '趋势图表' }}</div>
      <div class="chart-data">
        <div 
          v-for="(point, index) in limitedData"
          :key="index"
          class="data-point"
        >
          <span class="point-label">{{ point.label }}</span>
          <span class="point-value">{{ point.value }}</span>
        </div>
      </div>
    </div>
  </div>
</template>

<script>
import { computed } from 'vue'
import { TrendCharts } from '@element-plus/icons-vue'

export default {
  name: 'LineChart',
  components: {
    TrendCharts
  },
  props: {
    title: {
      type: String,
      default: ''
    },
    data: {
      type: Array,
      default: () => []
    },
    height: {
      type: String,
      default: '300px'
    }
  },
  setup(props) {
    // 限制显示的数据点数量
    const limitedData = computed(() => {
      return props.data.slice(-10) // 只显示最后10个数据点
    })

    return {
      limitedData
    }
  }
}
</script>

<style lang="scss" scoped>
.line-chart {
  height: v-bind(height);
  
  .chart-placeholder {
    height: 100%;
    display: flex;
    flex-direction: column;
    align-items: center;
    justify-content: center;
    background: #f5f7fa;
    border-radius: 4px;
    border: 1px dashed #dcdfe6;

    .chart-icon {
      font-size: 32px;
      color: #909399;
      margin-bottom: 8px;
    }

    .chart-text {
      color: #606266;
      font-size: 14px;
      margin-bottom: 16px;
    }

    .chart-data {
      display: flex;
      flex-wrap: wrap;
      gap: 12px;
      max-width: 80%;
      justify-content: center;

      .data-point {
        display: flex;
        flex-direction: column;
        align-items: center;
        font-size: 12px;

        .point-label {
          color: #909399;
          margin-bottom: 2px;
        }

        .point-value {
          color: #409EFF;
          font-weight: 600;
        }
      }
    }
  }
}
</style> 