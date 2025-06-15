<template>
  <div class="flex justify-end">
    <el-radio-group v-model="trendPeriod" size="small" @change="refreshTrendData">
      <el-radio-button value="5m">5分钟</el-radio-button>
      <el-radio-button value="1h">1小时</el-radio-button>
      <el-radio-button value="24h">24小时</el-radio-button>
    </el-radio-group>
  </div>
</template>

<script>
export default {
  // ... existing code ...
};
</script>

<style>
  // ... existing code ...
</style> 