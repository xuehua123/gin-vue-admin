<template>
  <div class="device-type-chart">
    <div class="chart-placeholder">
      <el-icon class="chart-icon"><PieChart /></el-icon>
      <div class="chart-text">设备类型分布</div>
      <div class="device-list">
        <div 
          v-for="item in processedData"
          :key="item.type"
          class="device-item"
        >
          <div class="device-type">{{ item.type }}</div>
          <div class="device-count">{{ item.count }}</div>
        </div>
      </div>
    </div>
  </div>
</template>

<script>
import { computed } from 'vue'
import { PieChart } from '@element-plus/icons-vue'

export default {
  name: 'DeviceTypeChart',
  components: {
    PieChart
  },
  props: {
    data: {
      type: Array,
      default: () => []
    }
  },
  setup(props) {
    const processedData = computed(() => {
      const typeMap = {
        android: 'Android',
        ios: 'iOS', 
        provider: 'Provider',
        other: '其他'
      }
      
      return props.data.map(item => ({
        type: typeMap[item.type] || item.type,
        count: item.count
      }))
    })

    return {
      processedData
    }
  }
}
</script>

<style lang="scss" scoped>
.device-type-chart {
  height: 200px;
  
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
      font-size: 28px;
      color: #909399;
      margin-bottom: 8px;
    }

    .chart-text {
      color: #606266;
      font-size: 14px;
      margin-bottom: 12px;
    }

    .device-list {
      .device-item {
        display: flex;
        justify-content: space-between;
        align-items: center;
        padding: 4px 16px;
        margin-bottom: 4px;
        font-size: 12px;
        width: 150px;

        .device-type {
          color: #606266;
        }

        .device-count {
          color: #409EFF;
          font-weight: 600;
        }
      }
    }
  }
}
</style> 