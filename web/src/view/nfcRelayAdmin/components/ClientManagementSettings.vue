<template>
  <el-dialog
    v-model="dialogVisible"
    title="客户端管理设置"
    width="500px"
    :close-on-click-modal="false"
  >
    <el-form :model="localSettings" label-width="120px">
      <el-form-item label="自动刷新">
        <el-switch v-model="localSettings.autoRefresh" />
      </el-form-item>
      <el-form-item label="刷新间隔">
        <el-input-number 
          v-model="localSettings.refreshInterval" 
          :min="5" 
          :max="300"
          :disabled="!localSettings.autoRefresh"
        />
        <span class="unit">秒</span>
      </el-form-item>
      <el-form-item label="显示性能指标">
        <el-switch v-model="localSettings.showPerformanceMetrics" />
      </el-form-item>
      <el-form-item label="启用地理定位">
        <el-switch v-model="localSettings.enableGeolocation" />
      </el-form-item>
    </el-form>

    <template #footer>
      <div class="dialog-footer">
        <el-button @click="closeDialog">取消</el-button>
        <el-button type="primary" @click="saveSettings">保存</el-button>
      </div>
    </template>
  </el-dialog>
</template>

<script>
import { ref, computed, watch } from 'vue'

export default {
  name: 'ClientManagementSettings',
  props: {
    modelValue: {
      type: Boolean,
      default: false
    },
    settings: {
      type: Object,
      default: () => ({})
    }
  },
  emits: ['update:modelValue', 'save'],
  setup(props, { emit }) {
    const localSettings = ref({})

    const dialogVisible = computed({
      get: () => props.modelValue,
      set: (value) => emit('update:modelValue', value)
    })

    watch(() => props.settings, (newSettings) => {
      localSettings.value = { ...newSettings }
    }, { immediate: true })

    const saveSettings = () => {
      emit('save', { ...localSettings.value })
      closeDialog()
    }

    const closeDialog = () => {
      dialogVisible.value = false
    }

    return {
      localSettings,
      dialogVisible,
      saveSettings,
      closeDialog
    }
  }
}
</script>

<style lang="scss" scoped>
.unit {
  margin-left: 8px;
  color: #909399;
}

.dialog-footer {
  display: flex;
  justify-content: flex-end;
  gap: 12px;
}
</style> 