<!--
  搜索表单组件
  可复用的搜索和筛选表单
-->
<template>
  <el-card shadow="never" class="search-form-card">
    <el-form 
      :inline="true" 
      :model="searchModel" 
      @submit.prevent="handleSearch"
      label-width="auto"
    >
      <template v-for="field in fields" :key="field.prop">
        <el-form-item :label="field.label">
          <!-- 输入框 -->
          <el-input
            v-if="field.type === 'input'"
            v-model="searchModel[field.prop]"
            :placeholder="field.placeholder"
            clearable
            :style="{ width: field.width || '200px' }"
          />
          
          <!-- 选择器 -->
          <el-select
            v-else-if="field.type === 'select'"
            v-model="searchModel[field.prop]"
            :placeholder="field.placeholder"
            clearable
            :style="{ width: field.width || '200px' }"
          >
            <el-option
              v-for="option in field.options"
              :key="option.value"
              :label="option.label"
              :value="option.value"
            />
          </el-select>
          
          <!-- 日期时间范围选择器 -->
          <el-date-picker
            v-else-if="field.type === 'datetimerange'"
            v-model="searchModel[field.prop]"
            type="datetimerange"
            range-separator="至"
            start-placeholder="开始时间"
            end-placeholder="结束时间"
            clearable
            :style="{ width: field.width || '350px' }"
            format="YYYY-MM-DD HH:mm:ss"
            value-format="YYYY-MM-DD HH:mm:ss"
          />
          
          <!-- 日期范围选择器 -->
          <el-date-picker
            v-else-if="field.type === 'daterange'"
            v-model="searchModel[field.prop]"
            type="daterange"
            range-separator="至"
            start-placeholder="开始日期"
            end-placeholder="结束日期"
            clearable
            :style="{ width: field.width || '300px' }"
            format="YYYY-MM-DD"
            value-format="YYYY-MM-DD"
          />
        </el-form-item>
      </template>
      
      <el-form-item>
        <el-button type="primary" @click="handleSearch" :loading="loading">
          <el-icon><Search /></el-icon>
          查询
        </el-button>
        <el-button @click="handleReset">
          <el-icon><Refresh /></el-icon>
          重置
        </el-button>
        <slot name="actions" />
      </el-form-item>
    </el-form>
  </el-card>
</template>

<script setup>
import { reactive, watch } from 'vue'
import { Search, Refresh } from '@element-plus/icons-vue'

const props = defineProps({
  fields: {
    type: Array,
    required: true
    // fields: [{ prop: string, label: string, type: 'input'|'select'|'datetimerange'|'daterange', placeholder: string, options?: [], width?: string }]
  },
  modelValue: {
    type: Object,
    required: true
  },
  loading: {
    type: Boolean,
    default: false
  }
})

const emit = defineEmits(['update:modelValue', 'search', 'reset'])

// 创建搜索模型的响应式副本
const searchModel = reactive({ ...props.modelValue })

// 监听搜索模型变化，同步到父组件
watch(searchModel, (newVal) => {
  emit('update:modelValue', { ...newVal })
}, { deep: true })

// 监听props.modelValue变化，同步到本地模型
watch(() => props.modelValue, (newVal) => {
  Object.assign(searchModel, newVal)
}, { deep: true })

const handleSearch = () => {
  emit('search', { ...searchModel })
}

const handleReset = () => {
  // 重置所有字段
  props.fields.forEach(field => {
    searchModel[field.prop] = field.type === 'datetimerange' || field.type === 'daterange' ? null : ''
  })
  emit('reset', { ...searchModel })
  emit('search', { ...searchModel })
}
</script>

<style scoped lang="scss">
.search-form-card {
  margin-bottom: 16px;
  
  :deep(.el-card__body) {
    padding: 16px 20px;
  }
  
  .el-form {
    .el-form-item {
      margin-bottom: 0;
      margin-right: 16px;
      
      &:last-child {
        margin-right: 0;
      }
    }
  }
}
</style> 