<template>
  <div class="world-map">
    <div class="map-placeholder">
      <el-icon class="map-icon"><Location /></el-icon>
      <div class="map-text">{{ title || '全球分布' }}</div>
      <div class="region-stats">
        <div 
          v-for="region in processedData"
          :key="region.name"
          class="region-item"
        >
          <span class="region-name">{{ region.name }}</span>
          <el-progress 
            :percentage="region.percentage" 
            :show-text="false"
            :stroke-width="8"
          />
          <span class="region-count">{{ region.count }}</span>
        </div>
      </div>
    </div>
  </div>
</template>

<script>
import { computed } from 'vue'
import { Location } from '@element-plus/icons-vue'

export default {
  name: 'WorldMap',
  components: {
    Location
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
      default: '400px'
    }
  },
  setup(props) {
    const processedData = computed(() => {
      const total = props.data.reduce((sum, item) => sum + item.count, 0)
      
      return props.data
        .map(item => ({
          ...item,
          percentage: total > 0 ? Math.round((item.count / total) * 100) : 0
        }))
        .sort((a, b) => b.count - a.count)
        .slice(0, 8) // 显示前8个地区
    })

    return {
      processedData
    }
  }
}
</script>

<style lang="scss" scoped>
.world-map {
  height: v-bind(height);
  
  .map-placeholder {
    height: 100%;
    display: flex;
    flex-direction: column;
    align-items: center;
    justify-content: center;
    background: #f5f7fa;
    border-radius: 4px;
    border: 1px dashed #dcdfe6;
    padding: 20px;

    .map-icon {
      font-size: 48px;
      color: #909399;
      margin-bottom: 12px;
    }

    .map-text {
      color: #606266;
      font-size: 16px;
      margin-bottom: 24px;
      font-weight: 600;
    }

    .region-stats {
      width: 100%;
      max-width: 300px;

      .region-item {
        display: flex;
        align-items: center;
        gap: 12px;
        margin-bottom: 12px;
        font-size: 13px;

        .region-name {
          color: #606266;
          min-width: 80px;
          text-align: left;
        }

        .el-progress {
          flex: 1;
        }

        .region-count {
          color: #409EFF;
          font-weight: 600;
          min-width: 30px;
          text-align: right;
        }
      }
    }
  }
}
</style> 