<!--
  配置详情对话框
  展示配置项的详细信息、历史记录和编辑功能
-->
<template>
  <el-dialog
    v-model="visible"
    title="配置详情"
    width="900px"
    :close-on-click-modal="false"
    append-to-body
    draggable
    @close="handleClose"
  >
    <div v-if="configData" class="config-detail">
      <!-- 基本信息 -->
      <div class="detail-section">
        <div class="section-header">
          <el-icon class="header-icon" color="#409EFF">
            <Setting />
          </el-icon>
          <h3 class="section-title">配置信息</h3>
          <el-tag 
            :type="getConfigStatusType(configData.status)"
            size="default"
          >
            {{ getConfigStatusText(configData.status) }}
          </el-tag>
        </div>
        
        <div class="info-grid">
          <div class="info-item">
            <span class="info-label">配置名称:</span>
            <span class="info-value">{{ configData.name }}</span>
          </div>
          <div class="info-item">
            <span class="info-label">配置路径:</span>
            <code class="info-value config-path">{{ configData.path }}</code>
          </div>
          <div class="info-item">
            <span class="info-label">数据类型:</span>
            <el-tag 
              :type="getConfigTypeColor(configData.type)" 
              size="small"
            >
              {{ getConfigTypeText(configData.type) }}
            </el-tag>
          </div>
          <div class="info-item">
            <span class="info-label">配置分类:</span>
            <span class="info-value">{{ getCategoryLabel(configData.category) }}</span>
          </div>
          <div class="info-item">
            <span class="info-label">最后修改:</span>
            <span class="info-value">
              {{ configData.lastModified ? formatDateTime(configData.lastModified) : '从未修改' }}
            </span>
          </div>
          <div class="info-item">
            <span class="info-label">修改人:</span>
            <span class="info-value">{{ configData.modifiedBy || '-' }}</span>
          </div>
        </div>
      </div>

      <!-- 当前值 -->
      <div class="detail-section">
        <div class="section-header">
          <el-icon class="header-icon" color="#67C23A">
            <Document />
          </el-icon>
          <h3 class="section-title">当前值</h3>
          <el-button 
            type="primary" 
            link 
            size="small"
            @click="copyValue"
          >
            <el-icon><CopyDocument /></el-icon>
            复制值
          </el-button>
        </div>
        
        <div class="value-display">
          <template v-if="configData.type === 'boolean'">
            <div class="boolean-value">
              <el-switch 
                :model-value="configData.value"
                :disabled="!canEdit"
                @change="handleValueChange"
              />
              <span class="boolean-text">
                {{ configData.value ? '启用' : '禁用' }}
              </span>
            </div>
          </template>
          
          <template v-else-if="configData.type === 'password'">
            <div class="password-value">
              <el-input
                :model-value="showPassword ? configData.value : '••••••••••••••••'"
                readonly
                :type="showPassword ? 'text' : 'password'"
                show-password
                @click="togglePasswordVisibility"
              />
            </div>
          </template>
          
          <template v-else-if="configData.type === 'array'">
            <div class="array-value">
              <el-tag 
                v-for="(item, index) in configData.value" 
                :key="index"
                size="default"
                class="mr-2 mb-2"
                closable
                @close="removeArrayItem(index)"
              >
                {{ item }}
              </el-tag>
              <el-button 
                v-if="canEdit"
                size="small" 
                type="primary" 
                link
                @click="addArrayItem"
              >
                <el-icon><Plus /></el-icon>
                添加项
              </el-button>
            </div>
          </template>
          
          <template v-else-if="configData.type === 'object'">
            <div class="object-value">
              <el-collapse>
                <el-collapse-item title="查看对象内容" name="object-content">
                  <pre class="json-display">{{ JSON.stringify(configData.value, null, 2) }}</pre>
                </el-collapse-item>
              </el-collapse>
            </div>
          </template>
          
          <template v-else>
            <div class="simple-value">
              <el-input
                v-if="canEdit && isEditing"
                v-model="editValue"
                :type="configData.type === 'number' ? 'number' : 'text'"
                @blur="saveValue"
                @keyup.enter="saveValue"
                ref="editInput"
              />
              <div v-else class="display-value" @dblclick="startEdit">
                {{ formatDisplayValue(configData.value) }}
                <el-icon v-if="canEdit" class="edit-hint">
                  <Edit />
                </el-icon>
              </div>
            </div>
          </template>
        </div>
      </div>

      <!-- 配置说明 -->
      <div v-if="configData.description" class="detail-section">
        <div class="section-header">
          <el-icon class="header-icon" color="#E6A23C">
            <InfoFilled />
          </el-icon>
          <h3 class="section-title">配置说明</h3>
        </div>
        
        <div class="description-content">
          <p>{{ configData.description }}</p>
          
          <div v-if="configData.constraints" class="constraints">
            <h4>配置约束：</h4>
            <ul>
              <li v-if="configData.constraints.required">
                <strong>必填项</strong>
              </li>
              <li v-if="configData.constraints.minValue !== undefined">
                <strong>最小值:</strong> {{ configData.constraints.minValue }}
              </li>
              <li v-if="configData.constraints.maxValue !== undefined">
                <strong>最大值:</strong> {{ configData.constraints.maxValue }}
              </li>
              <li v-if="configData.constraints.pattern">
                <strong>格式要求:</strong> <code>{{ configData.constraints.pattern }}</code>
              </li>
              <li v-if="configData.constraints.options">
                <strong>可选值:</strong> 
                <el-tag 
                  v-for="option in configData.constraints.options" 
                  :key="option"
                  size="small"
                  class="mr-1"
                >
                  {{ option }}
                </el-tag>
              </li>
            </ul>
          </div>
          
          <div v-if="configData.examples" class="examples">
            <h4>配置示例：</h4>
            <div class="example-list">
              <div 
                v-for="(example, index) in configData.examples" 
                :key="index"
                class="example-item"
              >
                <code>{{ example }}</code>
              </div>
            </div>
          </div>
        </div>
      </div>

      <!-- 历史记录 -->
      <div class="detail-section">
        <div class="section-header">
          <el-icon class="header-icon" color="#606266">
            <Clock />
          </el-icon>
          <h3 class="section-title">修改历史</h3>
          <el-button 
            type="primary" 
            link 
            size="small"
            @click="loadHistory"
          >
            刷新历史
          </el-button>
        </div>
        
        <div class="history-section">
          <div v-if="configHistory.length > 0" class="history-list">
            <div 
              v-for="record in configHistory.slice(0, 5)"
              :key="record.timestamp"
              class="history-item"
            >
              <div class="history-header">
                <span class="history-time">{{ formatDateTime(record.timestamp) }}</span>
                <span class="history-user">{{ record.modifiedBy }}</span>
                <el-tag 
                  :type="record.action === 'create' ? 'success' : 'warning'"
                  size="small"
                >
                  {{ getActionText(record.action) }}
                </el-tag>
              </div>
              <div class="history-content">
                <div class="value-change">
                  <span class="old-value">{{ record.oldValue }}</span>
                  <el-icon><Right /></el-icon>
                  <span class="new-value">{{ record.newValue }}</span>
                </div>
                <div v-if="record.comment" class="change-comment">
                  {{ record.comment }}
                </div>
              </div>
            </div>
          </div>
          
          <el-empty 
            v-else 
            description="暂无修改历史" 
            :image-size="80"
          />
        </div>
      </div>

      <!-- 相关配置 -->
      <div class="detail-section">
        <div class="section-header">
          <el-icon class="header-icon" color="#909399">
            <Connection />
          </el-icon>
          <h3 class="section-title">相关配置</h3>
        </div>
        
        <div class="related-configs">
          <div v-if="relatedConfigs.length > 0" class="config-list">
            <div 
              v-for="config in relatedConfigs"
              :key="config.key"
              class="config-item"
              @click="viewRelatedConfig(config)"
            >
              <div class="config-name">{{ config.name }}</div>
              <div class="config-value">{{ formatDisplayValue(config.value) }}</div>
            </div>
          </div>
          
          <div v-else class="no-related">
            暂无相关配置
          </div>
        </div>
      </div>
    </div>

    <template #footer>
      <div class="dialog-footer">
        <el-button @click="handleClose">关闭</el-button>
        <el-button 
          v-if="canEdit && hasChanges"
          type="primary" 
          @click="handleSave"
          :loading="saving"
        >
          保存更改
        </el-button>
        <el-button 
          v-if="canEdit"
          type="warning"
          @click="resetToDefault"
        >
          恢复默认
        </el-button>
      </div>
    </template>
  </el-dialog>
</template>

<script setup>
import { ref, reactive, computed, watch, nextTick } from 'vue'
import { 
  Setting,
  Document,
  InfoFilled,
  Clock,
  Connection,
  CopyDocument,
  Edit,
  Plus,
  Right
} from '@element-plus/icons-vue'
import { ElMessage } from 'element-plus'

import { formatDateTime } from '../../utils/formatters'

const props = defineProps({
  modelValue: {
    type: Boolean,
    default: false
  },
  configData: {
    type: Object,
    default: null
  }
})

const emit = defineEmits(['update:modelValue', 'refresh', 'update'])

// 状态管理
const visible = ref(false)
const isEditing = ref(false)
const editValue = ref('')
const showPassword = ref(false)
const saving = ref(false)
const configHistory = ref([])
const relatedConfigs = ref([])
const editInput = ref(null)

// 权限控制
const canEdit = computed(() => {
  // 根据配置类型和权限决定是否可编辑
  return props.configData && !props.configData.readonly
})

const hasChanges = computed(() => {
  // 检查是否有未保存的更改
  return editValue.value !== props.configData?.value
})

// 监听modelValue变化
watch(() => props.modelValue, (val) => {
  visible.value = val
  if (val && props.configData) {
    loadConfigDetails()
  }
})

watch(visible, (val) => {
  emit('update:modelValue', val)
})

// 加载配置详情
const loadConfigDetails = () => {
  editValue.value = props.configData.value
  loadHistory()
  loadRelatedConfigs()
}

// 工具函数
const getConfigStatusType = (status) => {
  const typeMap = {
    default: 'info',
    modified: 'warning',
    restart_required: 'danger',
    error: 'danger'
  }
  return typeMap[status] || 'info'
}

const getConfigStatusText = (status) => {
  const textMap = {
    default: '默认值',
    modified: '已修改',
    restart_required: '需要重启',
    error: '配置错误'
  }
  return textMap[status] || status
}

const getConfigTypeColor = (type) => {
  const colorMap = {
    string: 'info',
    number: 'warning',
    boolean: 'success',
    object: 'primary',
    array: 'danger'
  }
  return colorMap[type] || 'info'
}

const getConfigTypeText = (type) => {
  const textMap = {
    string: '字符串',
    number: '数字',
    boolean: '布尔值',
    object: '对象',
    array: '数组'
  }
  return textMap[type] || type
}

const getCategoryLabel = (category) => {
  const labelMap = {
    server: '服务器配置',
    session: '会话配置',
    security: '安全配置',
    logging: '日志配置',
    monitoring: '监控配置',
    network: '网络配置'
  }
  return labelMap[category] || category
}

const getActionText = (action) => {
  const textMap = {
    create: '创建',
    update: '修改',
    delete: '删除',
    reset: '重置'
  }
  return textMap[action] || action
}

const formatDisplayValue = (value) => {
  if (value === null || value === undefined) return '-'
  if (typeof value === 'object') return '[对象]'
  return String(value)
}

// 事件处理
const handleClose = () => {
  visible.value = false
  isEditing.value = false
}

const togglePasswordVisibility = () => {
  showPassword.value = !showPassword.value
}

const startEdit = () => {
  if (!canEdit.value) return
  isEditing.value = true
  nextTick(() => {
    editInput.value?.focus()
  })
}

const saveValue = () => {
  isEditing.value = false
  // 这里可以添加值验证逻辑
}

const handleValueChange = (newValue) => {
  editValue.value = newValue
}

const removeArrayItem = (index) => {
  if (!canEdit.value) return
  const newArray = [...props.configData.value]
  newArray.splice(index, 1)
  editValue.value = newArray
}

const addArrayItem = () => {
  ElMessage.info('添加数组项功能开发中...')
}

const copyValue = async () => {
  try {
    const value = typeof props.configData.value === 'object' 
      ? JSON.stringify(props.configData.value, null, 2)
      : String(props.configData.value)
    await navigator.clipboard.writeText(value)
    ElMessage.success('已复制到剪贴板')
  } catch (error) {
    ElMessage.error('复制失败')
  }
}

const handleSave = async () => {
  try {
    saving.value = true
    
    const updateData = {
      ...props.configData,
      value: editValue.value,
      lastModified: new Date().toISOString(),
      modifiedBy: 'admin', // 应该从用户上下文获取
      status: 'modified'
    }
    
    emit('update', updateData)
    ElMessage.success('配置保存成功')
  } catch (error) {
    ElMessage.error('保存失败: ' + error.message)
  } finally {
    saving.value = false
  }
}

const resetToDefault = () => {
  editValue.value = props.configData.defaultValue || ''
  ElMessage.info('已恢复为默认值')
}

const loadHistory = () => {
  // 模拟历史记录
  const mockHistory = []
  const now = Date.now()
  
  for (let i = 0; i < 3; i++) {
    mockHistory.push({
      timestamp: new Date(now - i * 24 * 60 * 60 * 1000).toISOString(),
      action: i === 0 ? 'update' : 'create',
      oldValue: `旧值 ${i + 1}`,
      newValue: `新值 ${i + 1}`,
      modifiedBy: 'admin',
      comment: i === 0 ? '优化性能配置' : ''
    })
  }
  
  configHistory.value = mockHistory
}

const loadRelatedConfigs = () => {
  // 模拟相关配置
  relatedConfigs.value = [
    { key: 'related1', name: '相关配置1', value: '值1' },
    { key: 'related2', name: '相关配置2', value: '值2' }
  ]
}

const viewRelatedConfig = (config) => {
  ElMessage.info(`查看相关配置: ${config.name}`)
}
</script>

<style scoped lang="scss">
.config-detail {
  .detail-section {
    margin-bottom: 24px;
    
    &:last-child {
      margin-bottom: 0;
    }
    
    .section-header {
      display: flex;
      align-items: center;
      gap: 8px;
      margin-bottom: 16px;
      padding-bottom: 8px;
      border-bottom: 1px solid #f0f0f0;
      
      .header-icon {
        font-size: 18px;
      }
      
      .section-title {
        flex: 1;
        margin: 0;
        font-size: 16px;
        font-weight: 600;
        color: #303133;
      }
    }
    
    .info-grid {
      display: grid;
      grid-template-columns: repeat(auto-fit, minmax(300px, 1fr));
      gap: 16px;
      
      .info-item {
        display: flex;
        justify-content: space-between;
        align-items: center;
        padding: 12px;
        background: #f8f9fa;
        border-radius: 6px;
        
        .info-label {
          font-size: 14px;
          color: #606266;
          font-weight: 500;
        }
        
        .info-value {
          font-size: 14px;
          color: #303133;
          max-width: 200px;
          overflow: hidden;
          text-overflow: ellipsis;
          white-space: nowrap;
          
          &.config-path {
            font-family: 'Courier New', monospace;
            font-size: 12px;
            background: #f5f7fa;
            padding: 2px 6px;
            border-radius: 3px;
          }
        }
      }
    }
    
    .value-display {
      .boolean-value {
        display: flex;
        align-items: center;
        gap: 12px;
        
        .boolean-text {
          font-size: 14px;
          color: #303133;
        }
      }
      
      .password-value {
        max-width: 400px;
      }
      
      .array-value {
        .mr-2 {
          margin-right: 8px;
        }
        .mb-2 {
          margin-bottom: 8px;
        }
      }
      
      .object-value {
        .json-display {
          background: #f5f7fa;
          border: 1px solid #e4e7ed;
          border-radius: 4px;
          padding: 12px;
          font-family: 'Courier New', monospace;
          font-size: 12px;
          line-height: 1.5;
          max-height: 300px;
          overflow-y: auto;
        }
      }
      
      .simple-value {
        .display-value {
          position: relative;
          padding: 8px 12px;
          background: #f8f9fa;
          border-radius: 4px;
          cursor: pointer;
          transition: background-color 0.3s;
          
          &:hover {
            background: #e8f4fd;
            
            .edit-hint {
              opacity: 1;
            }
          }
          
          .edit-hint {
            position: absolute;
            right: 8px;
            top: 50%;
            transform: translateY(-50%);
            opacity: 0;
            transition: opacity 0.3s;
            color: #409EFF;
            font-size: 12px;
          }
        }
      }
    }
    
    .description-content {
      .constraints {
        margin-top: 16px;
        
        h4 {
          margin-bottom: 8px;
          font-size: 14px;
          color: #303133;
        }
        
        ul {
          margin: 0;
          padding-left: 20px;
          
          li {
            margin-bottom: 4px;
            font-size: 13px;
            line-height: 1.5;
            
            code {
              font-family: 'Courier New', monospace;
              background: #f5f7fa;
              padding: 1px 4px;
              border-radius: 2px;
            }
          }
        }
      }
      
      .examples {
        margin-top: 16px;
        
        h4 {
          margin-bottom: 8px;
          font-size: 14px;
          color: #303133;
        }
        
        .example-list {
          .example-item {
            margin-bottom: 4px;
            
            code {
              font-family: 'Courier New', monospace;
              background: #f5f7fa;
              padding: 4px 8px;
              border-radius: 3px;
              font-size: 12px;
            }
          }
        }
      }
    }
    
    .history-section {
      .history-list {
        .history-item {
          padding: 12px;
          border: 1px solid #f0f0f0;
          border-radius: 6px;
          margin-bottom: 8px;
          
          &:last-child {
            margin-bottom: 0;
          }
          
          .history-header {
            display: flex;
            align-items: center;
            gap: 12px;
            margin-bottom: 8px;
            
            .history-time {
              font-size: 12px;
              color: #909399;
            }
            
            .history-user {
              font-size: 12px;
              color: #606266;
            }
          }
          
          .history-content {
            .value-change {
              display: flex;
              align-items: center;
              gap: 8px;
              font-size: 13px;
              
              .old-value {
                color: #F56C6C;
                text-decoration: line-through;
              }
              
              .new-value {
                color: #67C23A;
                font-weight: 500;
              }
            }
            
            .change-comment {
              margin-top: 4px;
              font-size: 12px;
              color: #909399;
              font-style: italic;
            }
          }
        }
      }
    }
    
    .related-configs {
      .config-list {
        .config-item {
          display: flex;
          justify-content: space-between;
          align-items: center;
          padding: 8px 12px;
          border: 1px solid #f0f0f0;
          border-radius: 4px;
          margin-bottom: 4px;
          cursor: pointer;
          transition: background-color 0.3s;
          
          &:hover {
            background: #f8f9fa;
          }
          
          &:last-child {
            margin-bottom: 0;
          }
          
          .config-name {
            font-size: 13px;
            color: #303133;
          }
          
          .config-value {
            font-size: 12px;
            color: #909399;
            font-family: 'Courier New', monospace;
          }
        }
      }
      
      .no-related {
        text-align: center;
        color: #c0c4cc;
        font-size: 13px;
        padding: 20px;
      }
    }
  }
}

.dialog-footer {
  display: flex;
  justify-content: flex-end;
  gap: 12px;
}

// 响应式设计
@media (max-width: 768px) {
  .config-detail {
    .info-grid {
      grid-template-columns: 1fr;
    }
    
    .history-item .history-header {
      flex-direction: column;
      align-items: flex-start;
      gap: 4px;
    }
  }
}
</style> 