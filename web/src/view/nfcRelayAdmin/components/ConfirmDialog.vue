<!--
  确认对话框组件
  用于重要操作的二次确认
-->
<template>
  <el-dialog
    v-model="visible"
    :title="title"
    :width="width"
    :close-on-click-modal="false"
    :close-on-press-escape="false"
    append-to-body
    draggable
  >
    <div class="confirm-content">
      <div class="confirm-icon">
        <el-icon :size="48" :color="iconColor">
          <component :is="iconComponent" />
        </el-icon>
      </div>
      
      <div class="confirm-text">
        <div class="confirm-message">{{ message }}</div>
        <div v-if="description" class="confirm-description">{{ description }}</div>
        
        <!-- 自定义内容插槽 -->
        <div v-if="$slots.default" class="confirm-custom">
          <slot />
        </div>
        
        <!-- 输入确认框 -->
        <div v-if="requireInput" class="confirm-input">
          <el-input
            v-model="inputValue"
            :placeholder="inputPlaceholder"
            clearable
            @keyup.enter="handleConfirm"
          />
          <div v-if="inputValidation && inputError" class="input-error">
            {{ inputError }}
          </div>
        </div>
      </div>
    </div>
    
    <template #footer>
      <div class="confirm-footer">
        <el-button @click="handleCancel" :disabled="loading">
          {{ cancelText }}
        </el-button>
        <el-button 
          :type="confirmType" 
          @click="handleConfirm" 
          :loading="loading"
          :disabled="isConfirmDisabled"
        >
          {{ confirmText }}
        </el-button>
      </div>
    </template>
  </el-dialog>
</template>

<script setup>
import { ref, computed, watch, markRaw } from 'vue'
import { 
  WarningFilled, 
  QuestionFilled, 
  InfoFilled, 
  CircleCheckFilled,
  CircleCloseFilled 
} from '@element-plus/icons-vue'

const props = defineProps({
  // 基础配置
  modelValue: {
    type: Boolean,
    default: false
  },
  
  title: {
    type: String,
    default: '确认操作'
  },
  
  message: {
    type: String,
    required: true
  },
  
  description: {
    type: String,
    default: ''
  },
  
  type: {
    type: String,
    default: 'warning',
    validator: (value) => ['warning', 'info', 'success', 'error', 'question'].includes(value)
  },
  
  width: {
    type: String,
    default: '420px'
  },
  
  // 按钮配置
  confirmText: {
    type: String,
    default: '确定'
  },
  
  cancelText: {
    type: String,
    default: '取消'
  },
  
  confirmType: {
    type: String,
    default: 'primary'
  },
  
  loading: {
    type: Boolean,
    default: false
  },
  
  // 输入确认
  requireInput: {
    type: Boolean,
    default: false
  },
  
  inputPlaceholder: {
    type: String,
    default: '请输入确认内容'
  },
  
  inputValidation: {
    type: [String, Function],
    default: null
    // string: 需要输入的确认文本
    // function: 自定义验证函数 (value) => boolean | string
  },
  
  // 数据
  data: {
    type: Object,
    default: () => ({})
  }
})

const emit = defineEmits(['update:modelValue', 'confirm', 'cancel'])

const visible = ref(false)
const inputValue = ref('')
const inputError = ref('')

// 图标映射
const iconMap = {
  warning: markRaw(WarningFilled),
  info: markRaw(InfoFilled),
  success: markRaw(CircleCheckFilled),
  error: markRaw(CircleCloseFilled),
  question: markRaw(QuestionFilled)
}

// 颜色映射
const colorMap = {
  warning: '#E6A23C',
  info: '#409EFF',
  success: '#67C23A',
  error: '#F56C6C',
  question: '#909399'
}

const iconComponent = computed(() => iconMap[props.type])
const iconColor = computed(() => colorMap[props.type])

// 确认按钮是否禁用
const isConfirmDisabled = computed(() => {
  if (!props.requireInput) return false
  
  if (props.inputValidation) {
    if (typeof props.inputValidation === 'string') {
      return inputValue.value !== props.inputValidation
    } else if (typeof props.inputValidation === 'function') {
      const result = props.inputValidation(inputValue.value)
      return result !== true
    }
  }
  
  return !inputValue.value.trim()
})

// 监听modelValue变化
watch(() => props.modelValue, (val) => {
  visible.value = val
  if (val) {
    // 重置输入状态
    inputValue.value = ''
    inputError.value = ''
  }
})

watch(visible, (val) => {
  emit('update:modelValue', val)
})

// 验证输入
const validateInput = () => {
  if (!props.requireInput) return true
  
  if (props.inputValidation) {
    if (typeof props.inputValidation === 'string') {
      if (inputValue.value !== props.inputValidation) {
        inputError.value = `请输入 "${props.inputValidation}" 以确认操作`
        return false
      }
    } else if (typeof props.inputValidation === 'function') {
      const result = props.inputValidation(inputValue.value)
      if (result !== true) {
        inputError.value = typeof result === 'string' ? result : '输入验证失败'
        return false
      }
    }
  } else if (!inputValue.value.trim()) {
    inputError.value = '请输入确认内容'
    return false
  }
  
  inputError.value = ''
  return true
}

// 事件处理
const handleConfirm = () => {
  if (props.requireInput && !validateInput()) {
    return
  }
  
  emit('confirm', {
    inputValue: inputValue.value,
    data: props.data
  })
}

const handleCancel = () => {
  visible.value = false
  emit('cancel')
}

// 暴露方法
const open = () => {
  visible.value = true
}

const close = () => {
  visible.value = false
}

defineExpose({
  open,
  close
})
</script>

<style scoped lang="scss">
.confirm-content {
  display: flex;
  align-items: flex-start;
  gap: 16px;
  
  .confirm-icon {
    flex-shrink: 0;
    margin-top: 4px;
  }
  
  .confirm-text {
    flex: 1;
    
    .confirm-message {
      font-size: 16px;
      font-weight: 500;
      color: #303133;
      margin-bottom: 8px;
      line-height: 1.5;
    }
    
    .confirm-description {
      font-size: 14px;
      color: #606266;
      line-height: 1.5;
      margin-bottom: 16px;
    }
    
    .confirm-custom {
      margin: 16px 0;
    }
    
    .confirm-input {
      margin-top: 16px;
      
      .input-error {
        color: #f56c6c;
        font-size: 12px;
        margin-top: 8px;
        line-height: 1.4;
      }
    }
  }
}

.confirm-footer {
  display: flex;
  justify-content: flex-end;
  gap: 12px;
}

:deep(.el-dialog__header) {
  padding: 20px 20px 10px;
  
  .el-dialog__title {
    font-size: 18px;
    font-weight: 600;
  }
}

:deep(.el-dialog__body) {
  padding: 10px 20px 20px;
}

:deep(.el-dialog__footer) {
  padding: 10px 20px 20px;
}
</style> 