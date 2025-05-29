<template>
  <div class="realtime-stats">
    <div class="stat-item">
      <div class="stat-value">{{ data.totalConnections || 0 }}</div>
      <div class="stat-label">总连接数</div>
    </div>
    <div class="stat-item">
      <div class="stat-value">{{ data.activeConnections || 0 }}</div>
      <div class="stat-label">活跃连接</div>
    </div>
    <div class="stat-item">
      <div class="stat-value">{{ data.avgLatency || 0 }}ms</div>
      <div class="stat-label">平均延迟</div>
    </div>
    <div class="stat-item">
      <div class="stat-value">{{ data.errorRate || 0 }}%</div>
      <div class="stat-label">错误率</div>
    </div>
  </div>
</template>

<script>
export default {
  name: 'RealtimeStats',
  props: {
    data: {
      type: Object,
      default: () => ({})
    }
  }
}
</script>

<style lang="scss" scoped>
.realtime-stats {
  display: flex;
  flex-direction: column;
  gap: 16px;

  .stat-item {
    text-align: center;
    padding: 12px;
    background: #f5f7fa;
    border-radius: 4px;

    .stat-value {
      font-size: 24px;
      font-weight: 600;
      color: #303133;
      margin-bottom: 4px;
    }

    .stat-label {
      font-size: 12px;
      color: #909399;
    }
  }
}
</style> 