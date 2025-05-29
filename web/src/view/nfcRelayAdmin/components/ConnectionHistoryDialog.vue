<template>
  <el-dialog
    v-model="dialogVisible"
    title="连接历史"
    width="800px"
    :close-on-click-modal="false"
  >
    <div class="connection-history">
      <el-table 
        :data="connectionHistory" 
        v-loading="loading"
        max-height="400"
      >
        <el-table-column prop="timestamp" label="时间" width="180">
          <template #default="{ row }">
            {{ formatDateTime(row.timestamp) }}
          </template>
        </el-table-column>
        <el-table-column prop="event" label="事件" width="120" />
        <el-table-column prop="ipAddress" label="IP地址" width="140" />
        <el-table-column prop="duration" label="持续时间" width="120">
          <template #default="{ row }">
            {{ formatDuration(row.duration) }}
          </template>
        </el-table-column>
        <el-table-column prop="reason" label="原因" show-overflow-tooltip />
      </el-table>
    </div>

    <template #footer>
      <div class="dialog-footer">
        <el-button @click="closeDialog">关闭</el-button>
      </div>
    </template>
  </el-dialog>
</template>

<script>
import { ref, computed, watch } from 'vue'
import { formatDateTime } from '@/utils/index'

export default {
  name: 'ConnectionHistoryDialog',
  props: {
    modelValue: {
      type: Boolean,
      default: false
    },
    clientId: {
      type: String,
      default: ''
    }
  },
  emits: ['update:modelValue'],
  setup(props, { emit }) {
    const loading = ref(false)
    const connectionHistory = ref([])

    const dialogVisible = computed({
      get: () => props.modelValue,
      set: (value) => emit('update:modelValue', value)
    })

    // 监听对话框显示状态
    watch(dialogVisible, (visible) => {
      if (visible && props.clientId) {
        loadConnectionHistory()
      }
    })

    // 格式化持续时间
    const formatDuration = (seconds) => {
      if (!seconds) return '0秒'
      const hours = Math.floor(seconds / 3600)
      const minutes = Math.floor((seconds % 3600) / 60)
      const secs = seconds % 60
      
      if (hours > 0) {
        return `${hours}h ${minutes}m ${secs}s`
      } else if (minutes > 0) {
        return `${minutes}m ${secs}s`
      } else {
        return `${secs}s`
      }
    }

    // 加载连接历史
    const loadConnectionHistory = async () => {
      loading.value = true
      try {
        // 模拟API调用
        await new Promise(resolve => setTimeout(resolve, 500))
        connectionHistory.value = [
          {
            timestamp: Date.now() - 3600000,
            event: '连接',
            ipAddress: '192.168.1.100',
            duration: 1800,
            reason: '正常连接'
          },
          {
            timestamp: Date.now() - 7200000,
            event: '断开',
            ipAddress: '192.168.1.100',
            duration: 3600,
            reason: '用户主动断开'
          }
        ]
      } catch (error) {
        console.error('加载连接历史失败:', error)
      } finally {
        loading.value = false
      }
    }

    // 关闭对话框
    const closeDialog = () => {
      dialogVisible.value = false
    }

    return {
      loading,
      connectionHistory,
      dialogVisible,
      formatDateTime,
      formatDuration,
      closeDialog
    }
  }
}
</script>

<style lang="scss" scoped>
.dialog-footer {
  display: flex;
  justify-content: flex-end;
}
</style> 