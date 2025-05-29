<template>
  <div class="pie-chart">
    <div class="chart-placeholder">
      <el-icon class="chart-icon"><PieChart /></el-icon>
      <div class="chart-text">{{ title || '饼图' }}</div>
      <div class="chart-legend">
        <div 
          v-for="(item, index) in processedData"
          :key="index"
          class="legend-item"
        >
          <div 
            class="legend-color"
            :style="{ backgroundColor: getColor(index) }"
          ></div>
          <span class="legend-label">{{ item.name }}: {{ item.value }}</span>
        </div>
      </div>
    </div>
  </div>
</template>

<script>
import { computed } from 'vue'
import { PieChart } from '@element-plus/icons-vue'

export default {
  name: 'PieChart',
  components: {
    PieChart
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
    const colors = ['#409EFF', '#67C23A', '#E6A23C', '#F56C6C', '#909399', '#c71585', '#00bfff', '#32cd32']

    const processedData = computed(() => {
      return props.data.slice(0, 8) // 最多显示8个项目
    })

    const getColor = (index) => {
      return colors[index % colors.length]
    }

    return {
      processedData,
      getColor
    }
  }
}
</script>

<style lang="scss" scoped>
.pie-chart {
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
    padding: 16px;

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

    .chart-legend {
      display: flex;
      flex-direction: column;
      gap: 8px;
      max-width: 100%;

      .legend-item {
        display: flex;
        align-items: center;
        gap: 8px;
        font-size: 12px;

        .legend-color {
          width: 12px;
          height: 12px;
          border-radius: 2px;
          flex-shrink: 0;
        }

        .legend-label {
          color: #606266;
        }
      }
    }
  }
}
</style> 