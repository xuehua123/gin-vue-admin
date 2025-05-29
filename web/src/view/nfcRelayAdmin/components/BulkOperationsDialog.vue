<template>
  <el-dialog
    v-model="dialogVisible"
    title="批量操作"
    width="600px"
    :close-on-click-modal="false"
    destroy-on-close
  >
    <div class="bulk-operations">
      <!-- 选中的客户端列表 -->
      <div class="selected-clients">
        <h4>已选择的客户端 ({{ selectedClients.length }})</h4>
        <div class="client-list">
          <el-tag
            v-for="clientId in selectedClients"
            :key="clientId"
            class="client-tag"
            size="small"
          >
            {{ clientId }}
          </el-tag>
        </div>
      </div>

      <!-- 操作选择 -->
      <div class="operation-selection">
        <h4>选择操作</h4>
        <el-radio-group v-model="selectedOperation" size="large">
          <el-radio value="disconnect" :disabled="loading">
            <el-icon><Link /></el-icon>
            断开连接
          </el-radio>
          <el-radio value="ban" :disabled="loading">
            <el-icon><Lock /></el-icon>
            封禁客户端
          </el-radio>
          <el-radio value="unban" :disabled="loading">
            <el-icon><Unlock /></el-icon>
            解除封禁
          </el-radio>
          <el-radio value="setAccess" :disabled="loading">
            <el-icon><User /></el-icon>
            设置访问权限
          </el-radio>
        </el-radio-group>
      </div>

      <!-- 操作参数 -->
      <div v-if="selectedOperation" class="operation-params">
        <h4>操作参数</h4>
        
        <!-- 断开连接参数 -->
        <div v-if="selectedOperation === 'disconnect'" class="param-section">
          <el-checkbox v-model="operationParams.forceDisconnect">
            强制断开连接
          </el-checkbox>
          <div class="help-text">勾选后将立即断开连接，不等待客户端响应</div>
        </div>

        <!-- 封禁参数 -->
        <div v-if="selectedOperation === 'ban'" class="param-section">
          <el-form :model="operationParams" label-width="100px">
            <el-form-item label="封禁时长">
              <el-select v-model="operationParams.banDuration" style="width: 200px">
                <el-option label="1小时" :value="3600" />
                <el-option label="6小时" :value="21600" />
                <el-option label="24小时" :value="86400" />
                <el-option label="7天" :value="604800" />
                <el-option label="30天" :value="2592000" />
                <el-option label="永久" :value="-1" />
              </el-select>
            </el-form-item>
            <el-form-item label="封禁原因">
              <el-input
                v-model="operationParams.banReason"
                type="textarea"
                :rows="3"
                placeholder="请输入封禁原因..."
                maxlength="200"
                show-word-limit
              />
            </el-form-item>
          </el-form>
        </div>

        <!-- 访问权限参数 -->
        <div v-if="selectedOperation === 'setAccess'" class="param-section">
          <el-form :model="operationParams" label-width="100px">
            <el-form-item label="访问级别">
              <el-select v-model="operationParams.accessLevel" style="width: 200px">
                <el-option label="只读" value="readonly" />
                <el-option label="标准" value="standard" />
                <el-option label="高级" value="advanced" />
                <el-option label="管理员" value="admin" />
              </el-select>
            </el-form-item>
            <el-form-item label="有效期">
              <el-date-picker
                v-model="operationParams.accessExpiry"
                type="datetime"
                placeholder="选择有效期"
                style="width: 200px"
              />
            </el-form-item>
          </el-form>
        </div>
      </div>

      <!-- 操作确认 -->
      <div v-if="selectedOperation" class="operation-confirm">
        <el-alert
          :type="getOperationAlertType(selectedOperation)"
          :title="getOperationWarningText(selectedOperation)"
          show-icon
          :closable="false"
        />
      </div>
    </div>

    <template #footer>
      <div class="dialog-footer">
        <el-button @click="closeDialog" :disabled="loading">
          取消
        </el-button>
        <el-button 
          type="primary" 
          @click="executeOperation"
          :loading="loading"
          :disabled="!selectedOperation"
        >
          执行操作
        </el-button>
      </div>
    </template>
  </el-dialog>
</template>

<script>
import { ref, reactive, computed } from 'vue'
import { ElMessage, ElMessageBox } from 'element-plus'
import { Link, Lock, Unlock, User } from '@element-plus/icons-vue'

export default {
  name: 'BulkOperationsDialog',
  components: {
    Link,
    Lock,
    Unlock,
    User
  },
  props: {
    modelValue: {
      type: Boolean,
      default: false
    },
    selectedClients: {
      type: Array,
      default: () => []
    }
  },
  emits: ['update:modelValue', 'complete'],
  setup(props, { emit }) {
    const loading = ref(false)
    const selectedOperation = ref('')
    
    const operationParams = reactive({
      forceDisconnect: false,
      banDuration: 86400,
      banReason: '',
      accessLevel: 'standard',
      accessExpiry: null
    })

    const dialogVisible = computed({
      get: () => props.modelValue,
      set: (value) => emit('update:modelValue', value)
    })

    // 获取操作警告类型
    const getOperationAlertType = (operation) => {
      const typeMap = {
        disconnect: 'warning',
        ban: 'error',
        unban: 'success',
        setAccess: 'info'
      }
      return typeMap[operation] || 'info'
    }

    // 获取操作警告文本
    const getOperationWarningText = (operation) => {
      const textMap = {
        disconnect: `即将断开 ${props.selectedClients.length} 个客户端的连接`,
        ban: `即将封禁 ${props.selectedClients.length} 个客户端，请谨慎操作`,
        unban: `即将解除 ${props.selectedClients.length} 个客户端的封禁`,
        setAccess: `即将修改 ${props.selectedClients.length} 个客户端的访问权限`
      }
      return textMap[operation] || ''
    }

    // 执行操作
    const executeOperation = async () => {
      if (!selectedOperation.value) {
        ElMessage.warning('请选择要执行的操作')
        return
      }

      try {
        await ElMessageBox.confirm(
          `确定要对 ${props.selectedClients.length} 个客户端执行 "${getOperationText(selectedOperation.value)}" 操作吗？`,
          '确认操作',
          {
            confirmButtonText: '确定',
            cancelButtonText: '取消',
            type: 'warning'
          }
        )

        loading.value = true

        // 模拟API调用
        await new Promise(resolve => setTimeout(resolve, 2000))

        ElMessage.success(`批量操作执行成功，影响 ${props.selectedClients.length} 个客户端`)
        
        emit('complete', {
          operation: selectedOperation.value,
          clients: props.selectedClients,
          params: { ...operationParams }
        })

        closeDialog()
      } catch (error) {
        if (error !== 'cancel') {
          console.error('批量操作失败:', error)
          ElMessage.error('批量操作执行失败')
        }
      } finally {
        loading.value = false
      }
    }

    // 获取操作文本
    const getOperationText = (operation) => {
      const textMap = {
        disconnect: '断开连接',
        ban: '封禁客户端',
        unban: '解除封禁',
        setAccess: '设置访问权限'
      }
      return textMap[operation] || operation
    }

    // 关闭对话框
    const closeDialog = () => {
      selectedOperation.value = ''
      Object.assign(operationParams, {
        forceDisconnect: false,
        banDuration: 86400,
        banReason: '',
        accessLevel: 'standard',
        accessExpiry: null
      })
      dialogVisible.value = false
    }

    return {
      loading,
      selectedOperation,
      operationParams,
      dialogVisible,
      getOperationAlertType,
      getOperationWarningText,
      executeOperation,
      closeDialog
    }
  }
}
</script>

<style lang="scss" scoped>
.bulk-operations {
  .selected-clients {
    margin-bottom: 24px;

    h4 {
      margin: 0 0 12px 0;
      color: #303133;
      font-size: 16px;
    }

    .client-list {
      max-height: 120px;
      overflow-y: auto;
      padding: 8px;
      background: #f5f7fa;
      border-radius: 4px;

      .client-tag {
        margin: 2px 4px 2px 0;
      }
    }
  }

  .operation-selection {
    margin-bottom: 24px;

    h4 {
      margin: 0 0 12px 0;
      color: #303133;
      font-size: 16px;
    }

    .el-radio {
      display: block;
      margin-bottom: 12px;

      .el-icon {
        margin-right: 4px;
      }
    }
  }

  .operation-params {
    margin-bottom: 24px;

    h4 {
      margin: 0 0 12px 0;
      color: #303133;
      font-size: 16px;
    }

    .param-section {
      padding: 16px;
      background: #fafbfc;
      border-radius: 4px;

      .help-text {
        font-size: 12px;
        color: #909399;
        margin-top: 8px;
      }
    }
  }

  .operation-confirm {
    margin-bottom: 16px;
  }
}

.dialog-footer {
  display: flex;
  justify-content: flex-end;
  gap: 12px;
}
</style> 