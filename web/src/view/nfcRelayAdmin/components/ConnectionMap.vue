<template>
  <div class="connection-map">
    <div class="map-placeholder">
      <el-icon class="map-icon"><LocationFilled /></el-icon>
      <div class="map-text">地理分布图</div>
      <div class="location-list">
        <div 
          v-for="location in processedLocations"
          :key="location.name"
          class="location-item"
        >
          <span class="location-name">{{ location.name }}</span>
          <el-tag size="small" type="primary">{{ location.count }}</el-tag>
        </div>
      </div>
    </div>
  </div>
</template>

<script>
import { computed } from 'vue'
import { LocationFilled } from '@element-plus/icons-vue'

export default {
  name: 'ConnectionMap',
  components: {
    LocationFilled
  },
  props: {
    connections: {
      type: Array,
      default: () => []
    }
  },
  setup(props) {
    // 处理位置数据
    const processedLocations = computed(() => {
      const locationMap = {}
      
      props.connections.forEach(conn => {
        if (conn.location) {
          locationMap[conn.location] = (locationMap[conn.location] || 0) + 1
        }
      })
      
      return Object.entries(locationMap)
        .map(([name, count]) => ({ name, count }))
        .sort((a, b) => b.count - a.count)
        .slice(0, 10) // 只显示前10个位置
    })

    return {
      processedLocations
    }
  }
}
</script>

<style lang="scss" scoped>
.connection-map {
  height: 250px;
  
  .map-placeholder {
    height: 100%;
    display: flex;
    flex-direction: column;
    align-items: center;
    justify-content: center;
    background: #f5f7fa;
    border-radius: 4px;
    border: 1px dashed #dcdfe6;

    .map-icon {
      font-size: 32px;
      color: #909399;
      margin-bottom: 8px;
    }

    .map-text {
      color: #606266;
      font-size: 14px;
      margin-bottom: 16px;
    }

    .location-list {
      max-height: 120px;
      overflow-y: auto;
      width: 100%;
      padding: 0 16px;

      .location-item {
        display: flex;
        justify-content: space-between;
        align-items: center;
        padding: 4px 0;
        font-size: 12px;

        .location-name {
          color: #606266;
        }
      }
    }
  }
}
</style> 