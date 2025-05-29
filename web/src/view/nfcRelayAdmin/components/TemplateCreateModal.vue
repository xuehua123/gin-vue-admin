<!-- 
  配置模板创建对话框
  用于创建新的配置模板
-->
<template>
  <el-dialog
    :model-value="modelValue"
    @update:model-value="emit('update:modelValue', $event)"
    title="创建配置模板"
    width="500px"
    :close-on-click-modal="false"
    @closed="handleClosed"
  >
    <el-form ref="formRef" :model="form" :rules="rules" label-width="100px">
      <el-form-item label="模板名称" prop="name">
        <el-input v-model="form.name" placeholder="请输入模板名称" />
      </el-form-item>
      <el-form-item label="描述" prop="description">
        <el-input 
          v-model="form.description" 
          type="textarea" 
          :rows="3" 
          placeholder="请输入模板描述" 
        />
      </el-form-item>
      <el-form-item label="来源" prop="source">
        <el-radio-group v-model="form.source">
          <el-radio label="current">当前配置</el-radio>
          <el-radio label="blank">空白模板</el-radio>
          <el-radio label="existing">已有模板</el-radio>
        </el-radio-group>
      </el-form-item>
      <el-form-item v-if="form.source === 'existing'" label="选择模板" prop="templateId">
        <el-select v-model="form.templateId" placeholder="请选择模板">
          <el-option
            v-for="template in templates"
            :key="template.id"
            :label="template.name"
            :value="template.id"
          />
        </el-select>
      </el-form-item>
      <el-form-item label="版本号" prop="version">
        <el-input v-model="form.version" placeholder="请输入版本号，如 1.0.0" />
      </el-form-item>
      <el-form-item v-if="form.source === 'current'" label="包含选项">
        <el-checkbox v-model="form.includeAll">包含所有配置项</el-checkbox>
        <div v-if="!form.includeAll" class="mt-2">
          <el-checkbox-group v-model="form.includedSections">
            <el-checkbox 
              v-for="section in configSections"
              :key="section.key"
              :label="section.key"
            >
              {{ section.name }}
            </el-checkbox>
          </el-checkbox-group>
        </div>
      </el-form-item>
    </el-form>
    <template #footer>
      <span class="dialog-footer">
        <el-button @click="handleCancel">取消</el-button>
        <el-button type="primary" @click="handleSubmit" :loading="loading">
          创建
        </el-button>
      </span>
    </template>
  </el-dialog>
</template>

<script setup>
import { ref, computed, watch } from 'vue'
import { ElMessage } from 'element-plus'

const props = defineProps({
  modelValue: {
    type: Boolean,
    default: false
  },
  configSections: {
    type: Array,
    default: () => []
  },
  templates: {
    type: Array,
    default: () => []
  }
})

const emit = defineEmits(['update:modelValue', 'created'])

const form = ref({
  name: '',
  description: '',
  version: '1.0.0',
  source: 'current',
  templateId: '',
  includeAll: true,
  includedSections: []
})

const rules = {
  name: [
    { required: true, message: '请输入模板名称', trigger: 'blur' },
    { min: 2, max: 50, message: '长度在 2 到 50 个字符', trigger: 'blur' }
  ],
  description: [
    { max: 200, message: '描述不能超过200个字符', trigger: 'blur' }
  ],
  version: [
    { required: true, message: '请输入版本号', trigger: 'blur' },
    { 
      pattern: /^\d+\.\d+\.\d+$/, 
      message: '版本号格式应为 X.Y.Z', 
      trigger: 'blur' 
    }
  ],
  templateId: [
    { 
      required: true, 
      message: '请选择模板', 
      trigger: 'change',
      validator: (rule, value, callback) => {
        if (form.value.source === 'existing' && !value) {
          callback(new Error('请选择模板'))
        } else {
          callback()
        }
      }
    }
  ]
}

const formRef = ref(null)
const loading = ref(false)

// 重置表单
const resetForm = () => {
  form.value = {
    name: '',
    description: '',
    version: '1.0.0',
    source: 'current',
    templateId: '',
    includeAll: true,
    includedSections: []
  }
  if (formRef.value) {
    formRef.value.resetFields()
  }
}

// 处理提交
const handleSubmit = async () => {
  if (!formRef.value) return
  
  try {
    await formRef.value.validate()
    
    loading.value = true
    
    // 这里添加创建模板的实际逻辑，通常是API调用
    // const response = await createConfigTemplate(form.value)
    
    // 模拟API调用延迟
    await new Promise(resolve => setTimeout(resolve, 1000))
    
    // 成功后发出事件
    emit('created', {
      ...form.value,
      id: Date.now(), // 模拟新ID
      createdAt: new Date(),
      configCount: 15,
      usageCount: 0
    })
    
    ElMessage.success('配置模板创建成功')
    emit('update:modelValue', false)
  } catch (error) {
    console.error('表单验证失败或API错误:', error)
  } finally {
    loading.value = false
  }
}

// 取消操作
const handleCancel = () => {
  emit('update:modelValue', false)
}

// 对话框关闭时重置表单
const handleClosed = () => {
  resetForm()
}

// 监听对话框显示状态变化
watch(() => props.modelValue, (val) => {
  if (val) {
    // 对话框打开时，可以在这里加载数据
    form.value.includedSections = props.configSections.map(section => section.key)
  }
})
</script>

<style scoped>
.dialog-footer {
  display: flex;
  justify-content: flex-end;
}
</style> 