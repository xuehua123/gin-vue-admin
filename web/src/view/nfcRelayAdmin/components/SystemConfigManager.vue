<!--
  系统配置管理组件
  支持配置模板、版本管理、热更新等功能
-->
<template>
  <div class="system-config-manager">
    <el-card shadow="never">
      <template #header>
        <div class="flex justify-between items-center">
          <div class="flex items-center">
            <el-icon class="mr-2" color="#409eff">
              <Setting />
            </el-icon>
            <span class="text-lg font-semibold">系统配置管理</span>
            <el-tag v-if="currentVersion" size="small" class="ml-2" type="success">
              v{{ currentVersion.version }}
            </el-tag>
          </div>
          <div class="flex items-center space-x-2">
            <el-button size="small" @click="reloadConfig">
              <el-icon><Refresh /></el-icon>
              重新加载
            </el-button>
            <el-button size="small" @click="exportConfig">
              <el-icon><Download /></el-icon>
              导出配置
            </el-button>
            <el-button size="small" type="primary" @click="showCreateTemplate = true">
              <el-icon><Plus /></el-icon>
              新建模板
            </el-button>
          </div>
        </div>
      </template>

      <!-- 配置概览面板 -->
      <div class="config-overview mb-4">
        <el-row :gutter="16">
          <el-col :span="6">
            <div class="overview-card">
              <div class="overview-icon">
                <el-icon size="24" color="#409eff">
                  <Files />
                </el-icon>
              </div>
              <div class="overview-content">
                <div class="overview-value">{{ configSections.length }}</div>
                <div class="overview-label">配置分组</div>
              </div>
            </div>
          </el-col>
          <el-col :span="6">
            <div class="overview-card">
              <div class="overview-icon">
                <el-icon size="24" color="#67c23a">
                  <Collection />
                </el-icon>
              </div>
              <div class="overview-content">
                <div class="overview-value">{{ configTemplates.length }}</div>
                <div class="overview-label">配置模板</div>
              </div>
            </div>
          </el-col>
          <el-col :span="6">
            <div class="overview-card">
              <div class="overview-icon">
                <el-icon size="24" color="#e6a23c">
                  <Clock />
                </el-icon>
              </div>
              <div class="overview-content">
                <div class="overview-value">{{ configVersions.length }}</div>
                <div class="overview-label">历史版本</div>
              </div>
            </div>
          </el-col>
          <el-col :span="6">
            <div class="overview-card">
              <div class="overview-icon">
                <el-icon size="24" color="#f56c6c">
                  <Warning />
                </el-icon>
              </div>
              <div class="overview-content">
                <div class="overview-value">{{ pendingChanges }}</div>
                <div class="overview-label">待应用更改</div>
              </div>
            </div>
          </el-col>
        </el-row>
      </div>

      <!-- 主体内容区域 -->
      <el-tabs v-model="activeTab" class="config-tabs">
        <!-- 当前配置 -->
        <el-tab-pane label="当前配置" name="current">
          <div class="current-config">
            <div class="config-toolbar mb-4">
              <div class="flex justify-between items-center">
                <div class="flex items-center space-x-2">
                  <el-input 
                    v-model="searchKeyword" 
                    placeholder="搜索配置项..." 
                    clearable
                    style="width: 300px"
                  >
                    <template #prefix>
                      <el-icon><Search /></el-icon>
                    </template>
                  </el-input>
                  <el-select v-model="selectedSection" placeholder="选择分组" clearable>
                    <el-option 
                      v-for="section in configSections" 
                      :key="section.key" 
                      :label="section.name" 
                      :value="section.key" 
                    />
                  </el-select>
                </div>
                <div class="flex items-center space-x-2">
                  <el-switch v-model="editMode" active-text="编辑模式" />
                  <el-button 
                    v-if="editMode && hasChanges" 
                    type="primary" 
                    size="small"
                    @click="saveChanges"
                  >
                    保存更改
                  </el-button>
                  <el-button 
                    v-if="editMode && hasChanges" 
                    size="small"
                    @click="discardChanges"
                  >
                    放弃更改
                  </el-button>
                </div>
              </div>
            </div>

            <div class="config-sections">
              <el-collapse v-model="activeConfigSections">
                <el-collapse-item 
                  v-for="section in filteredConfigSections" 
                  :key="section.key"
                  :title="section.name"
                  :name="section.key"
                >
                  <template #title>
                    <div class="flex items-center">
                      <el-icon class="mr-2" :color="section.color">
                        <component :is="section.icon" />
                      </el-icon>
                      <span>{{ section.name }}</span>
                      <el-tag size="small" class="ml-2">{{ section.items.length }} 项</el-tag>
                    </div>
                  </template>
                  
                  <div class="config-items">
                    <div 
                      v-for="item in section.items" 
                      :key="item.key"
                      class="config-item"
                    >
                      <div class="config-item-header">
                        <div class="config-item-info">
                          <span class="config-item-key">{{ item.key }}</span>
                          <span v-if="item.description" class="config-item-desc">
                            {{ item.description }}
                          </span>
                        </div>
                        <div class="config-item-actions">
                          <el-tag 
                            v-if="item.changed" 
                            size="small" 
                            type="warning"
                          >
                            已修改
                          </el-tag>
                          <el-button 
                            link 
                            size="small" 
                            @click="showConfigHistory(item)"
                          >
                            历史
                          </el-button>
                        </div>
                      </div>
                      
                      <div class="config-item-value">
                        <!-- 字符串类型 -->
                        <el-input 
                          v-if="item.type === 'string'"
                          v-model="item.value"
                          :disabled="!editMode"
                          @change="markAsChanged(item)"
                        />
                        
                        <!-- 数字类型 -->
                        <el-input-number 
                          v-else-if="item.type === 'number'"
                          v-model="item.value"
                          :disabled="!editMode"
                          :min="item.min"
                          :max="item.max"
                          @change="markAsChanged(item)"
                        />
                        
                        <!-- 布尔类型 -->
                        <el-switch 
                          v-else-if="item.type === 'boolean'"
                          v-model="item.value"
                          :disabled="!editMode"
                          @change="markAsChanged(item)"
                        />
                        
                        <!-- 选择类型 -->
                        <el-select 
                          v-else-if="item.type === 'select'"
                          v-model="item.value"
                          :disabled="!editMode"
                          @change="markAsChanged(item)"
                        >
                          <el-option 
                            v-for="option in item.options" 
                            :key="option.value" 
                            :label="option.label" 
                            :value="option.value" 
                          />
                        </el-select>
                        
                        <!-- JSON类型 -->
                        <div v-else-if="item.type === 'json'" class="json-editor">
                          <el-input 
                            v-model="item.value"
                            type="textarea"
                            :rows="4"
                            :disabled="!editMode"
                            @change="markAsChanged(item)"
                          />
                          <div class="json-actions mt-2">
                            <el-button size="small" @click="formatJSON(item)">
                              格式化
                            </el-button>
                            <el-button size="small" @click="validateJSON(item)">
                              验证
                            </el-button>
                          </div>
                        </div>
                        
                        <!-- 默认文本显示 -->
                        <span v-else class="config-value-text">{{ item.value }}</span>
                      </div>
                    </div>
                  </div>
                </el-collapse-item>
              </el-collapse>
            </div>
          </div>
        </el-tab-pane>

        <!-- 配置模板 -->
        <el-tab-pane label="配置模板" name="templates">
          <div class="config-templates">
            <div class="templates-grid">
              <div 
                v-for="template in configTemplates" 
                :key="template.id"
                class="template-card"
                @click="selectTemplate(template)"
              >
                <div class="template-header">
                  <div class="template-info">
                    <h4>{{ template.name }}</h4>
                    <p>{{ template.description }}</p>
                  </div>
                  <div class="template-actions">
                    <el-dropdown @command="handleTemplateAction">
                      <el-button link>
                        <el-icon><MoreFilled /></el-icon>
                      </el-button>
                      <template #dropdown>
                        <el-dropdown-menu>
                          <el-dropdown-item :command="{action: 'apply', template}">
                            应用模板
                          </el-dropdown-item>
                          <el-dropdown-item :command="{action: 'edit', template}">
                            编辑模板
                          </el-dropdown-item>
                          <el-dropdown-item :command="{action: 'duplicate', template}">
                            复制模板
                          </el-dropdown-item>
                          <el-dropdown-item :command="{action: 'delete', template}" divided>
                            删除模板
                          </el-dropdown-item>
                        </el-dropdown-menu>
                      </template>
                    </el-dropdown>
                  </div>
                </div>
                
                <div class="template-content">
                  <div class="template-meta">
                    <span class="template-version">v{{ template.version }}</span>
                    <span class="template-date">{{ formatDate(template.createdAt) }}</span>
                  </div>
                  <div class="template-stats">
                    <span>{{ template.configCount }} 项配置</span>
                    <span>{{ template.usageCount }} 次使用</span>
                  </div>
                </div>
              </div>

              <!-- 新建模板卡片 -->
              <div class="template-card template-card--create" @click="showCreateTemplate = true">
                <div class="create-content">
                  <el-icon size="32" color="#c0c4cc">
                    <Plus />
                  </el-icon>
                  <span>新建配置模板</span>
                </div>
              </div>
            </div>
          </div>
        </el-tab-pane>

        <!-- 版本历史 -->
        <el-tab-pane label="版本历史" name="versions">
          <div class="config-versions">
            <div class="version-timeline">
              <div 
                v-for="(version, index) in configVersions" 
                :key="version.id"
                class="version-item"
                :class="{ 'version-current': version.isCurrent }"
              >
                <div class="version-marker">
                  <div class="version-dot"></div>
                  <div v-if="index < configVersions.length - 1" class="version-line"></div>
                </div>
                
                <div class="version-content">
                  <div class="version-header">
                    <div class="version-info">
                      <h4>v{{ version.version }}</h4>
                      <span class="version-author">{{ version.author }}</span>
                      <span class="version-date">{{ formatDate(version.createdAt) }}</span>
                    </div>
                    <div class="version-actions">
                      <el-tag v-if="version.isCurrent" size="small" type="success">
                        当前版本
                      </el-tag>
                      <el-button 
                        v-else 
                        size="small" 
                        @click="rollbackToVersion(version)"
                      >
                        回滚到此版本
                      </el-button>
                      <el-button size="small" @click="viewVersionDiff(version)">
                        查看差异
                      </el-button>
                    </div>
                  </div>
                  
                  <div class="version-description">
                    {{ version.description || '无描述' }}
                  </div>
                  
                  <div class="version-changes">
                    <el-tag 
                      v-for="change in version.changes" 
                      :key="change.key"
                      size="small"
                      :type="getChangeType(change.type)"
                      class="mr-1 mb-1"
                    >
                      {{ change.key }}: {{ change.type }}
                    </el-tag>
                  </div>
                </div>
              </div>
            </div>
          </div>
        </el-tab-pane>

        <!-- 热更新 -->
        <el-tab-pane label="热更新" name="hotupdate">
          <div class="hot-update">
            <el-alert
              title="热更新功能"
              type="warning"
              description="热更新允许在不重启系统的情况下应用配置更改。请谨慎使用，某些配置可能需要重启才能生效。"
              show-icon
              class="mb-4"
            />

            <div class="update-status mb-4">
              <el-card>
                <div class="status-content">
                  <div class="status-info">
                    <h4>系统状态</h4>
                    <div class="status-items">
                      <div class="status-item">
                        <span class="status-label">运行时间:</span>
                        <span class="status-value">{{ systemUptime }}</span>
                      </div>
                      <div class="status-item">
                        <span class="status-label">配置版本:</span>
                        <span class="status-value">v{{ currentVersion?.version }}</span>
                      </div>
                      <div class="status-item">
                        <span class="status-label">最后更新:</span>
                        <span class="status-value">{{ formatDate(lastUpdateTime) }}</span>
                      </div>
                    </div>
                  </div>
                  <div class="status-indicator">
                    <div :class="systemStatusClass">
                      <el-icon size="32">
                        <CircleCheck v-if="systemStatus === 'healthy'" />
                        <Warning v-else-if="systemStatus === 'warning'" />
                        <CircleClose v-else />
                      </el-icon>
                    </div>
                    <span>{{ systemStatusText }}</span>
                  </div>
                </div>
              </el-card>
            </div>

            <div class="update-actions">
              <el-row :gutter="16">
                <el-col :span="8">
                  <el-card>
                    <h4>应用待更改配置</h4>
                    <p>将当前未保存的配置更改应用到系统</p>
                    <el-button 
                      type="primary" 
                      :disabled="!hasChanges"
                      @click="applyHotUpdate"
                    >
                      应用更改 ({{ pendingChanges }} 项)
                    </el-button>
                  </el-card>
                </el-col>
                
                <el-col :span="8">
                  <el-card>
                    <h4>重新加载配置</h4>
                    <p>从配置文件重新加载所有配置</p>
                    <el-button @click="reloadConfig">
                      重新加载
                    </el-button>
                  </el-card>
                </el-col>
                
                <el-col :span="8">
                  <el-card>
                    <h4>系统重启</h4>
                    <p>重启整个NFC中继服务</p>
                    <el-button type="danger" @click="confirmSystemRestart">
                      重启系统
                    </el-button>
                  </el-card>
                </el-col>
              </el-row>
            </div>
          </div>
        </el-tab-pane>
      </el-tabs>
    </el-card>

    <!-- 创建模板弹窗 -->
    <template-create-modal 
      v-model:visible="showCreateTemplate" 
      @created="loadTemplates" 
    />

    <!-- 配置历史弹窗 -->
    <config-history-modal 
      v-model:visible="showConfigHistory" 
      :config-item="selectedConfigItem" 
    />

    <!-- 版本差异弹窗 -->
    <version-diff-modal 
      v-model:visible="showVersionDiff" 
      :version="selectedVersion" 
    />
  </div>
</template>

<script setup>
import { ref, reactive, computed, onMounted } from 'vue'
import { ElMessage, ElMessageBox } from 'element-plus'
import {
  Setting,
  Refresh,
  Download,
  Plus,
  Files,
  Collection,
  Clock,
  Warning,
  Search,
  MoreFilled,
  CircleCheck,
  CircleClose
} from '@element-plus/icons-vue'
import TemplateCreateModal from './TemplateCreateModal.vue'
import ConfigHistoryModal from './ConfigHistoryModal.vue'
import VersionDiffModal from './VersionDiffModal.vue'

// 响应式数据
const activeTab = ref('current')
const searchKeyword = ref('')
const selectedSection = ref('')
const editMode = ref(false)
const hasChanges = ref(false)
const activeConfigSections = ref(['server', 'security'])

const showCreateTemplate = ref(false)
const showConfigHistory = ref(false)
const showVersionDiff = ref(false)
const selectedConfigItem = ref(null)
const selectedVersion = ref(null)

// 配置数据
const configSections = ref([
  {
    key: 'server',
    name: '服务器配置',
    icon: 'Monitor',
    color: '#409eff',
    items: [
      {
        key: 'server.host',
        description: '服务器监听地址',
        type: 'string',
        value: '0.0.0.0',
        changed: false
      },
      {
        key: 'server.port',
        description: '服务器监听端口',
        type: 'number',
        value: 8888,
        min: 1024,
        max: 65535,
        changed: false
      },
      {
        key: 'server.enable_ssl',
        description: '启用SSL加密',
        type: 'boolean',
        value: true,
        changed: false
      }
    ]
  },
  {
    key: 'security',
    name: '安全配置',
    icon: 'Lock',
    color: '#67c23a',
    items: [
      {
        key: 'security.auth_mode',
        description: '认证模式',
        type: 'select',
        value: 'token',
        options: [
          { label: 'Token认证', value: 'token' },
          { label: '证书认证', value: 'cert' },
          { label: '无认证', value: 'none' }
        ],
        changed: false
      },
      {
        key: 'security.session_timeout',
        description: '会话超时时间(秒)',
        type: 'number',
        value: 3600,
        min: 60,
        max: 86400,
        changed: false
      }
    ]
  },
  {
    key: 'logging',
    name: '日志配置',
    icon: 'Document',
    color: '#e6a23c',
    items: [
      {
        key: 'logging.level',
        description: '日志级别',
        type: 'select',
        value: 'INFO',
        options: [
          { label: 'DEBUG', value: 'DEBUG' },
          { label: 'INFO', value: 'INFO' },
          { label: 'WARN', value: 'WARN' },
          { label: 'ERROR', value: 'ERROR' }
        ],
        changed: false
      },
      {
        key: 'logging.config',
        description: '日志配置JSON',
        type: 'json',
        value: '{"appenders": {"console": {"type": "console"}}}',
        changed: false
      }
    ]
  }
])

const configTemplates = ref([
  {
    id: 1,
    name: '开发环境',
    description: '适用于开发和测试的配置模板',
    version: '1.0.0',
    createdAt: new Date('2024-01-15'),
    configCount: 15,
    usageCount: 23
  },
  {
    id: 2,
    name: '生产环境',
    description: '适用于生产环境的安全配置模板',
    version: '2.1.0',
    createdAt: new Date('2024-02-20'),
    configCount: 18,
    usageCount: 8
  },
  {
    id: 3,
    name: '高性能',
    description: '针对高并发场景优化的配置模板',
    version: '1.5.0',
    createdAt: new Date('2024-03-10'),
    configCount: 12,
    usageCount: 5
  }
])

const configVersions = ref([
  {
    id: 1,
    version: '2.1.0',
    author: 'admin',
    createdAt: new Date(),
    description: '优化性能配置，增加安全参数',
    isCurrent: true,
    changes: [
      { key: 'server.port', type: '修改' },
      { key: 'security.auth_mode', type: '修改' },
      { key: 'logging.level', type: '新增' }
    ]
  },
  {
    id: 2,
    version: '2.0.0',
    author: 'admin',
    createdAt: new Date(Date.now() - 24 * 60 * 60 * 1000),
    description: '重大版本更新，重构配置结构',
    isCurrent: false,
    changes: [
      { key: 'server.enable_ssl', type: '新增' },
      { key: 'security.session_timeout', type: '修改' }
    ]
  }
])

const currentVersion = ref(configVersions.value.find(v => v.isCurrent))
const systemUptime = ref('2天 14小时 23分钟')
const systemStatus = ref('healthy')
const lastUpdateTime = ref(new Date())

// 计算属性
const filteredConfigSections = computed(() => {
  return configSections.value.filter(section => {
    if (selectedSection.value && section.key !== selectedSection.value) {
      return false
    }
    
    if (searchKeyword.value) {
      const keyword = searchKeyword.value.toLowerCase()
      return section.items.some(item => 
        item.key.toLowerCase().includes(keyword) ||
        (item.description && item.description.toLowerCase().includes(keyword))
      )
    }
    
    return true
  })
})

const pendingChanges = computed(() => {
  return configSections.value.reduce((count, section) => {
    return count + section.items.filter(item => item.changed).length
  }, 0)
})

const systemStatusClass = computed(() => ({
  'status-healthy': systemStatus.value === 'healthy',
  'status-warning': systemStatus.value === 'warning',
  'status-error': systemStatus.value === 'error'
}))

const systemStatusText = computed(() => {
  switch (systemStatus.value) {
    case 'healthy': return '系统正常'
    case 'warning': return '系统警告'
    case 'error': return '系统错误'
    default: return '未知状态'
  }
})

// 方法
const markAsChanged = (item) => {
  item.changed = true
  hasChanges.value = true
}

const saveChanges = async () => {
  try {
    // 模拟保存API调用
    await new Promise(resolve => setTimeout(resolve, 1000))
    
    // 重置更改状态
    configSections.value.forEach(section => {
      section.items.forEach(item => {
        item.changed = false
      })
    })
    
    hasChanges.value = false
    lastUpdateTime.value = new Date()
    
    // 创建新版本
    const newVersion = {
      id: configVersions.value.length + 1,
      version: `2.${configVersions.value.length}.0`,
      author: 'admin',
      createdAt: new Date(),
      description: '保存配置更改',
      isCurrent: true,
      changes: []
    }
    
    configVersions.value.forEach(v => v.isCurrent = false)
    configVersions.value.unshift(newVersion)
    currentVersion.value = newVersion
    
    ElMessage.success('配置已保存')
  } catch (error) {
    ElMessage.error('保存配置失败')
  }
}

const discardChanges = () => {
  ElMessageBox.confirm('确定要放弃所有未保存的更改吗？', '确认操作', {
    type: 'warning'
  }).then(() => {
    // 重置所有更改
    configSections.value.forEach(section => {
      section.items.forEach(item => {
        item.changed = false
        // 这里应该恢复原始值
      })
    })
    hasChanges.value = false
    ElMessage.info('已放弃所有更改')
  })
}

const reloadConfig = async () => {
  try {
    ElMessage.info('正在重新加载配置...')
    await new Promise(resolve => setTimeout(resolve, 1500))
    ElMessage.success('配置已重新加载')
  } catch (error) {
    ElMessage.error('重新加载配置失败')
  }
}

const exportConfig = () => {
  const config = {}
  configSections.value.forEach(section => {
    section.items.forEach(item => {
      config[item.key] = item.value
    })
  })
  
  const blob = new Blob([JSON.stringify(config, null, 2)], { type: 'application/json' })
  const url = URL.createObjectURL(blob)
  const a = document.createElement('a')
  a.href = url
  a.download = `nfc-relay-config-${Date.now()}.json`
  a.click()
  URL.revokeObjectURL(url)
  ElMessage.success('配置已导出')
}

const formatJSON = (item) => {
  try {
    const parsed = JSON.parse(item.value)
    item.value = JSON.stringify(parsed, null, 2)
    ElMessage.success('JSON格式化成功')
  } catch (error) {
    ElMessage.error('JSON格式错误')
  }
}

const validateJSON = (item) => {
  try {
    JSON.parse(item.value)
    ElMessage.success('JSON格式正确')
  } catch (error) {
    ElMessage.error('JSON格式错误: ' + error.message)
  }
}

const selectTemplate = (template) => {
  ElMessageBox.confirm(`确定要应用模板 "${template.name}" 吗？这将覆盖当前配置。`, '确认操作', {
    type: 'warning'
  }).then(() => {
    ElMessage.success(`已应用模板: ${template.name}`)
  })
}

const handleTemplateAction = (command) => {
  const { action, template } = command
  
  switch (action) {
    case 'apply':
      selectTemplate(template)
      break
    case 'edit':
      ElMessage.info('编辑模板功能开发中')
      break
    case 'duplicate':
      ElMessage.info('复制模板功能开发中')
      break
    case 'delete':
      ElMessageBox.confirm(`确定要删除模板 "${template.name}" 吗？`, '确认删除', {
        type: 'warning'
      }).then(() => {
        ElMessage.success('模板已删除')
      })
      break
  }
}

const rollbackToVersion = (version) => {
  ElMessageBox.confirm(`确定要回滚到版本 v${version.version} 吗？`, '确认回滚', {
    type: 'warning'
  }).then(() => {
    ElMessage.success(`已回滚到版本 v${version.version}`)
  })
}

const viewVersionDiff = (version) => {
  selectedVersion.value = version
  showVersionDiff.value = true
}

const applyHotUpdate = async () => {
  try {
    ElMessage.info('正在应用热更新...')
    await new Promise(resolve => setTimeout(resolve, 2000))
    await saveChanges()
    ElMessage.success('热更新已应用')
  } catch (error) {
    ElMessage.error('热更新应用失败')
  }
}

const confirmSystemRestart = () => {
  ElMessageBox.confirm('确定要重启系统吗？这将中断所有当前连接。', '确认重启', {
    type: 'warning',
    confirmButtonText: '确定重启',
    cancelButtonText: '取消'
  }).then(() => {
    ElMessage.success('系统重启指令已发送')
  })
}

const loadTemplates = () => {
  // 重新加载模板列表
  ElMessage.success('模板列表已更新')
}

const showConfigHistoryModal = (item) => {
  selectedConfigItem.value = item
  showConfigHistory.value = true
}

const getChangeType = (type) => {
  switch (type) {
    case '新增': return 'success'
    case '修改': return 'warning'
    case '删除': return 'danger'
    default: return 'info'
  }
}

const formatDate = (date) => {
  return new Date(date).toLocaleDateString()
}

// 生命周期
onMounted(() => {
  // 初始化数据
})
</script>

<style scoped lang="scss">
.system-config-manager {
  .config-overview {
    .overview-card {
      display: flex;
      align-items: center;
      padding: 20px;
      background: linear-gradient(135deg, #667eea 0%, #764ba2 100%);
      border-radius: 12px;
      color: white;
      
      .overview-icon {
        margin-right: 16px;
      }
      
      .overview-content {
        .overview-value {
          font-size: 24px;
          font-weight: bold;
          margin-bottom: 4px;
        }
        
        .overview-label {
          font-size: 14px;
          opacity: 0.8;
        }
      }
    }
  }
  
  .config-tabs {
    .current-config {
      .config-toolbar {
        padding: 16px;
        background-color: #f5f7fa;
        border-radius: 8px;
        border: 1px solid #e4e7ed;
      }
      
      .config-sections {
        .config-item {
          padding: 16px;
          border: 1px solid #e4e7ed;
          border-radius: 8px;
          margin-bottom: 12px;
          
          .config-item-header {
            display: flex;
            justify-content: space-between;
            align-items: flex-start;
            margin-bottom: 12px;
            
            .config-item-info {
              .config-item-key {
                font-family: monospace;
                font-weight: 600;
                color: #303133;
                margin-bottom: 4px;
                display: block;
              }
              
              .config-item-desc {
                font-size: 14px;
                color: #606266;
              }
            }
            
            .config-item-actions {
              display: flex;
              align-items: center;
              gap: 8px;
            }
          }
          
          .config-item-value {
            .json-editor {
              .json-actions {
                display: flex;
                gap: 8px;
              }
            }
            
            .config-value-text {
              font-family: monospace;
              color: #606266;
            }
          }
        }
      }
    }
    
    .config-templates {
      .templates-grid {
        display: grid;
        grid-template-columns: repeat(auto-fill, minmax(300px, 1fr));
        gap: 16px;
        
        .template-card {
          border: 1px solid #e4e7ed;
          border-radius: 12px;
          padding: 20px;
          cursor: pointer;
          transition: all 0.3s ease;
          
          &:hover {
            border-color: #409eff;
            box-shadow: 0 4px 12px rgba(64, 158, 255, 0.1);
          }
          
          &.template-card--create {
            border-style: dashed;
            display: flex;
            align-items: center;
            justify-content: center;
            min-height: 160px;
            
            .create-content {
              display: flex;
              flex-direction: column;
              align-items: center;
              gap: 12px;
              color: #909399;
              
              span {
                font-size: 16px;
              }
            }
          }
          
          .template-header {
            display: flex;
            justify-content: space-between;
            align-items: flex-start;
            margin-bottom: 16px;
            
            .template-info {
              h4 {
                margin: 0 0 8px 0;
                color: #303133;
              }
              
              p {
                margin: 0;
                color: #606266;
                font-size: 14px;
              }
            }
          }
          
          .template-content {
            .template-meta {
              display: flex;
              justify-content: space-between;
              margin-bottom: 8px;
              font-size: 12px;
              color: #909399;
            }
            
            .template-stats {
              display: flex;
              gap: 16px;
              font-size: 12px;
              color: #606266;
            }
          }
        }
      }
    }
    
    .config-versions {
      .version-timeline {
        .version-item {
          display: flex;
          margin-bottom: 24px;
          
          &.version-current {
            .version-content {
              border-color: #67c23a;
              background-color: #f0f9ff;
            }
          }
          
          .version-marker {
            width: 40px;
            display: flex;
            flex-direction: column;
            align-items: center;
            margin-right: 16px;
            
            .version-dot {
              width: 12px;
              height: 12px;
              border-radius: 50%;
              background-color: #409eff;
              border: 3px solid #e1f3ff;
            }
            
            .version-line {
              flex: 1;
              width: 2px;
              background-color: #e4e7ed;
              margin-top: 8px;
            }
          }
          
          .version-content {
            flex: 1;
            border: 1px solid #e4e7ed;
            border-radius: 8px;
            padding: 16px;
            
            .version-header {
              display: flex;
              justify-content: space-between;
              align-items: flex-start;
              margin-bottom: 12px;
              
              .version-info {
                h4 {
                  margin: 0 0 8px 0;
                  color: #303133;
                }
                
                .version-author,
                .version-date {
                  font-size: 12px;
                  color: #909399;
                  margin-right: 16px;
                }
              }
              
              .version-actions {
                display: flex;
                gap: 8px;
              }
            }
            
            .version-description {
              color: #606266;
              margin-bottom: 12px;
            }
            
            .version-changes {
              .mr-1 {
                margin-right: 4px;
              }
              
              .mb-1 {
                margin-bottom: 4px;
              }
            }
          }
        }
      }
    }
    
    .hot-update {
      .update-status {
        .status-content {
          display: flex;
          justify-content: space-between;
          align-items: center;
          
          .status-info {
            h4 {
              margin: 0 0 16px 0;
              color: #303133;
            }
            
            .status-items {
              .status-item {
                display: flex;
                margin-bottom: 8px;
                
                .status-label {
                  width: 100px;
                  color: #606266;
                }
                
                .status-value {
                  color: #303133;
                  font-weight: 500;
                }
              }
            }
          }
          
          .status-indicator {
            text-align: center;
            
            .status-healthy {
              color: #67c23a;
            }
            
            .status-warning {
              color: #e6a23c;
            }
            
            .status-error {
              color: #f56c6c;
            }
            
            span {
              display: block;
              margin-top: 8px;
              font-weight: 500;
            }
          }
        }
      }
      
      .update-actions {
        .el-card {
          text-align: center;
          
          h4 {
            margin: 0 0 12px 0;
            color: #303133;
          }
          
          p {
            margin: 0 0 16px 0;
            color: #606266;
            font-size: 14px;
          }
        }
      }
    }
  }
}

.text-lg {
  font-size: 1.125rem;
}

.font-semibold {
  font-weight: 600;
}

.flex {
  display: flex;
  
  &.justify-between {
    justify-content: space-between;
  }
  
  &.items-center {
    align-items: center;
  }
}

.space-x-2 > * + * {
  margin-left: 0.5rem;
}

.mr-2 {
  margin-right: 0.5rem;
}

.ml-2 {
  margin-left: 0.5rem;
}

.mb-4 {
  margin-bottom: 1rem;
}

.mt-2 {
  margin-top: 0.5rem;
}
</style> 