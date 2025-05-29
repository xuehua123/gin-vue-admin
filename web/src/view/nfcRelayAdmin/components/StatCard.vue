<!--
  统计卡片组件
  用于显示关键指标和状态信息
-->
<template>
  <div class="stat-card">
    <el-card shadow="hover" :body-style="{ padding: '20px' }">
      <div class="stat-content">
        <div class="stat-icon">
          <el-icon :size="32" :color="iconColor">
            <component :is="icon" />
          </el-icon>
        </div>
        <div class="stat-info">
          <div class="stat-title">{{ title }}</div>
          <div class="stat-value">{{ value }}</div>
          <div v-if="subtitle" class="stat-subtitle">{{ subtitle }}</div>
        </div>
      </div>
      <div v-if="trend" class="stat-trend">
        <div class="trend-content" :class="trend.type">
          <el-icon :size="16">
            <ArrowUp v-if="trend.type === 'up'" />
            <ArrowDown v-if="trend.type === 'down'" />
            <Minus v-if="trend.type === 'neutral'" />
          </el-icon>
          <span>{{ trend.value }}</span>
        </div>
        <div class="trend-label">{{ trend.label }}</div>
      </div>
    </el-card>
  </div>
</template>

<script setup>
import { ArrowUp, ArrowDown, Minus } from '@element-plus/icons-vue'

defineProps({
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
    required: true
  },
  iconColor: {
    type: String,
    default: '#409eff'
  },
  trend: {
    type: Object,
    default: null
    // trend: { type: 'up'|'down'|'neutral', value: string, label: string }
  }
})
</script>

<style scoped lang="scss">
.stat-card {
  .stat-content {
    display: flex;
    align-items: center;
    gap: 16px;
    margin-bottom: 12px;
    
    .stat-icon {
      flex-shrink: 0;
    }
    
    .stat-info {
      flex: 1;
      
      .stat-title {
        font-size: 14px;
        color: #606266;
        margin-bottom: 4px;
        font-weight: 500;
      }
      
      .stat-value {
        font-size: 24px;
        font-weight: bold;
        color: #303133;
        margin-bottom: 2px;
      }
      
      .stat-subtitle {
        font-size: 12px;
        color: #909399;
      }
    }
  }
  
  .stat-trend {
    display: flex;
    align-items: center;
    justify-content: space-between;
    padding-top: 12px;
    border-top: 1px solid #f0f0f0;
    
    .trend-content {
      display: flex;
      align-items: center;
      gap: 4px;
      font-size: 14px;
      font-weight: 500;
      
      &.up {
        color: #67c23a;
      }
      
      &.down {
        color: #f56c6c;
      }
      
      &.neutral {
        color: #909399;
      }
    }
    
    .trend-label {
      font-size: 12px;
      color: #909399;
    }
  }
}
</style> 