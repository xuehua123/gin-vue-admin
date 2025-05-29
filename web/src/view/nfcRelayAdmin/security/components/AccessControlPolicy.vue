<template>
  <div class="access-control-policy">
    <el-row :gutter="20">
      <!-- IP访问控制 -->
      <el-col :span="12">
        <el-card>
          <template #header>
            <div class="card-header">
              <span>IP访问控制</span>
              <el-switch 
                v-model="ipControlEnabled" 
                @change="saveIpControlPolicy"
                inline-prompt
                active-text="启用"
                inactive-text="禁用"
              />
            </div>
          </template>
          
          <el-form 
            :model="ipControlPolicy" 
            label-width="120px"
            :disabled="!ipControlEnabled"
          >
            <el-form-item label="白名单模式">
              <el-switch 
                v-model="ipControlPolicy.whitelistMode"
                @change="saveIpControlPolicy"
                inline-prompt
                active-text="启用"
                inactive-text="禁用"
              />
              <div class="help-text">启用后仅允许白名单内的IP访问</div>
            </el-form-item>
            
            <el-form-item label="白名单IP">
              <el-tag
                v-for="ip in ipControlPolicy.whitelist"
                :key="ip"
                closable
                @close="removeWhitelistIp(ip)"
                class="mr-2 mb-2"
              >
                {{ ip }}
              </el-tag>
              <el-input
                v-if="showAddIpInput"
                ref="addIpInputRef"
                v-model="newIpAddress"
                size="small"
                style="width: 120px"
                @keyup.enter="addWhitelistIp"
                @blur="addWhitelistIp"
              />
              <el-button 
                v-else 
                size="small" 
                @click="showAddIpInput = true"
              >
                + 添加IP
              </el-button>
            </el-form-item>
            
            <el-form-item label="黑名单IP">
              <el-tag
                v-for="ip in ipControlPolicy.blacklist"
                :key="ip"
                type="danger"
                closable
                @close="removeBlacklistIp(ip)"
                class="mr-2 mb-2"
              >
                {{ ip }}
              </el-tag>
              <el-input
                v-if="showAddBlackIpInput"
                ref="addBlackIpInputRef"
                v-model="newBlackIpAddress"
                size="small"
                style="width: 120px"
                @keyup.enter="addBlacklistIp"
                @blur="addBlacklistIp"
              />
              <el-button 
                v-else 
                size="small" 
                type="danger"
                @click="showAddBlackIpInput = true"
              >
                + 添加IP
              </el-button>
            </el-form-item>
            
            <el-form-item>
              <el-button 
                type="primary" 
                @click="saveIpControlPolicy"
                :loading="saving"
              >
                保存IP控制策略
              </el-button>
            </el-form-item>
          </el-form>
        </el-card>
      </el-col>

      <!-- 角色权限控制 -->
      <el-col :span="12">
        <el-card>
          <template #header>
            <div class="card-header">
              <span>角色权限控制</span>
              <el-switch 
                v-model="roleControlEnabled" 
                @change="saveRoleControlPolicy"
                inline-prompt
                active-text="启用"
                inactive-text="禁用"
              />
            </div>
          </template>
          
          <el-form 
            :model="roleControlPolicy" 
            label-width="120px"
            :disabled="!roleControlEnabled"
          >
            <el-form-item label="默认角色">
              <el-select 
                v-model="roleControlPolicy.defaultRole" 
                style="width: 100%"
                @change="saveRoleControlPolicy"
              >
                <el-option label="普通用户" value="user" />
                <el-option label="高级用户" value="advanced_user" />
                <el-option label="管理员" value="admin" />
                <el-option label="超级管理员" value="super_admin" />
              </el-select>
              <div class="help-text">新用户的默认角色</div>
            </el-form-item>
            
            <el-form-item label="自动升级">
              <el-checkbox 
                v-model="roleControlPolicy.autoUpgrade"
                @change="saveRoleControlPolicy"
              >
                根据用户活跃度自动升级角色
              </el-checkbox>
            </el-form-item>
            
            <el-form-item label="角色权限">
              <el-table 
                :data="rolePermissions" 
                size="small"
                style="width: 100%"
              >
                <el-table-column prop="role" label="角色" width="120" />
                <el-table-column prop="permissions" label="权限">
                  <template #default="{ row }">
                    <el-tag 
                      v-for="perm in row.permissions"
                      :key="perm"
                      size="small"
                      class="mr-1"
                    >
                      {{ getPermissionText(perm) }}
                    </el-tag>
                  </template>
                </el-table-column>
              </el-table>
            </el-form-item>
            
            <el-form-item>
              <el-button 
                type="primary" 
                @click="saveRoleControlPolicy"
                :loading="saving"
              >
                保存角色策略
              </el-button>
            </el-form-item>
          </el-form>
        </el-card>
      </el-col>
    </el-row>

    <!-- 时间访问控制 -->
    <el-row class="mt-4">
      <el-col :span="24">
        <el-card>
          <template #header>
            <div class="card-header">
              <span>时间访问控制</span>
              <el-switch 
                v-model="timeControlEnabled" 
                @change="saveTimeControlPolicy"
                inline-prompt
                active-text="启用"
                inactive-text="禁用"
              />
            </div>
          </template>
          
          <el-form 
            :model="timeControlPolicy" 
            label-width="120px"
            :disabled="!timeControlEnabled"
          >
            <el-row :gutter="20">
              <el-col :span="12">
                <el-form-item label="工作日限制">
                  <el-checkbox-group 
                    v-model="timeControlPolicy.allowedWeekdays"
                    @change="saveTimeControlPolicy"
                  >
                    <el-checkbox value="1">周一</el-checkbox>
                    <el-checkbox value="2">周二</el-checkbox>
                    <el-checkbox value="3">周三</el-checkbox>
                    <el-checkbox value="4">周四</el-checkbox>
                    <el-checkbox value="5">周五</el-checkbox>
                    <el-checkbox value="6">周六</el-checkbox>
                    <el-checkbox value="0">周日</el-checkbox>
                  </el-checkbox-group>
                </el-form-item>
              </el-col>
              
              <el-col :span="12">
                <el-form-item label="时间范围">
                  <el-time-picker
                    v-model="timeControlPolicy.startTime"
                    placeholder="开始时间"
                    format="HH:mm"
                    value-format="HH:mm"
                    @change="saveTimeControlPolicy"
                  />
                  <span class="mx-2">至</span>
                  <el-time-picker
                    v-model="timeControlPolicy.endTime"
                    placeholder="结束时间"
                    format="HH:mm"
                    value-format="HH:mm"
                    @change="saveTimeControlPolicy"
                  />
                </el-form-item>
              </el-col>
            </el-row>
            
            <el-form-item>
              <el-button 
                type="primary" 
                @click="saveTimeControlPolicy"
                :loading="saving"
              >
                保存时间控制策略
              </el-button>
            </el-form-item>
          </el-form>
        </el-card>
      </el-col>
    </el-row>
  </div>
</template>

<script>
import { ref, reactive, nextTick } from 'vue'
import { ElMessage } from 'element-plus'

export default {
  name: 'AccessControlPolicy',
  setup() {
    const saving = ref(false)
    
    // IP访问控制
    const ipControlEnabled = ref(true)
    const ipControlPolicy = reactive({
      whitelistMode: false,
      whitelist: ['192.168.1.0/24', '10.0.0.0/8'],
      blacklist: ['192.168.1.100']
    })
    
    const showAddIpInput = ref(false)
    const showAddBlackIpInput = ref(false)
    const newIpAddress = ref('')
    const newBlackIpAddress = ref('')
    const addIpInputRef = ref()
    const addBlackIpInputRef = ref()
    
    // 角色权限控制
    const roleControlEnabled = ref(true)
    const roleControlPolicy = reactive({
      defaultRole: 'user',
      autoUpgrade: false
    })
    
    const rolePermissions = ref([
      {
        role: '普通用户',
        permissions: ['read', 'basic_operation']
      },
      {
        role: '高级用户',
        permissions: ['read', 'write', 'advanced_operation']
      },
      {
        role: '管理员',
        permissions: ['read', 'write', 'delete', 'admin_operation']
      },
      {
        role: '超级管理员',
        permissions: ['all']
      }
    ])
    
    // 时间访问控制
    const timeControlEnabled = ref(false)
    const timeControlPolicy = reactive({
      allowedWeekdays: ['1', '2', '3', '4', '5'],
      startTime: '09:00',
      endTime: '18:00'
    })
    
    // 获取权限文本
    const getPermissionText = (permission) => {
      const permissionMap = {
        'read': '读取',
        'write': '写入',
        'delete': '删除',
        'basic_operation': '基础操作',
        'advanced_operation': '高级操作',
        'admin_operation': '管理操作',
        'all': '全部权限'
      }
      return permissionMap[permission] || permission
    }
    
    // IP地址验证
    const validateIpAddress = (ip) => {
      const ipRegex = /^(\d{1,3}\.){3}\d{1,3}(\/\d{1,2})?$/
      return ipRegex.test(ip)
    }
    
    // 添加白名单IP
    const addWhitelistIp = () => {
      if (newIpAddress.value && validateIpAddress(newIpAddress.value)) {
        if (!ipControlPolicy.whitelist.includes(newIpAddress.value)) {
          ipControlPolicy.whitelist.push(newIpAddress.value)
          saveIpControlPolicy()
        }
        newIpAddress.value = ''
      } else if (newIpAddress.value) {
        ElMessage.error('请输入有效的IP地址或CIDR格式')
      }
      showAddIpInput.value = false
    }
    
    // 移除白名单IP
    const removeWhitelistIp = (ip) => {
      const index = ipControlPolicy.whitelist.indexOf(ip)
      if (index > -1) {
        ipControlPolicy.whitelist.splice(index, 1)
        saveIpControlPolicy()
      }
    }
    
    // 添加黑名单IP
    const addBlacklistIp = () => {
      if (newBlackIpAddress.value && validateIpAddress(newBlackIpAddress.value)) {
        if (!ipControlPolicy.blacklist.includes(newBlackIpAddress.value)) {
          ipControlPolicy.blacklist.push(newBlackIpAddress.value)
          saveIpControlPolicy()
        }
        newBlackIpAddress.value = ''
      } else if (newBlackIpAddress.value) {
        ElMessage.error('请输入有效的IP地址或CIDR格式')
      }
      showAddBlackIpInput.value = false
    }
    
    // 移除黑名单IP
    const removeBlacklistIp = (ip) => {
      const index = ipControlPolicy.blacklist.indexOf(ip)
      if (index > -1) {
        ipControlPolicy.blacklist.splice(index, 1)
        saveIpControlPolicy()
      }
    }
    
    // 保存IP控制策略
    const saveIpControlPolicy = async () => {
      saving.value = true
      try {
        // 这里应该调用API保存IP控制策略
        await new Promise(resolve => setTimeout(resolve, 500))
        ElMessage.success('IP访问控制策略保存成功')
      } catch (error) {
        console.error('保存IP控制策略失败:', error)
        ElMessage.error('保存IP控制策略失败')
      } finally {
        saving.value = false
      }
    }
    
    // 保存角色控制策略
    const saveRoleControlPolicy = async () => {
      saving.value = true
      try {
        // 这里应该调用API保存角色控制策略
        await new Promise(resolve => setTimeout(resolve, 500))
        ElMessage.success('角色权限控制策略保存成功')
      } catch (error) {
        console.error('保存角色控制策略失败:', error)
        ElMessage.error('保存角色控制策略失败')
      } finally {
        saving.value = false
      }
    }
    
    // 保存时间控制策略
    const saveTimeControlPolicy = async () => {
      saving.value = true
      try {
        // 这里应该调用API保存时间控制策略
        await new Promise(resolve => setTimeout(resolve, 500))
        ElMessage.success('时间访问控制策略保存成功')
      } catch (error) {
        console.error('保存时间控制策略失败:', error)
        ElMessage.error('保存时间控制策略失败')
      } finally {
        saving.value = false
      }
    }
    
    return {
      saving,
      ipControlEnabled,
      ipControlPolicy,
      showAddIpInput,
      showAddBlackIpInput,
      newIpAddress,
      newBlackIpAddress,
      addIpInputRef,
      addBlackIpInputRef,
      roleControlEnabled,
      roleControlPolicy,
      rolePermissions,
      timeControlEnabled,
      timeControlPolicy,
      getPermissionText,
      addWhitelistIp,
      removeWhitelistIp,
      addBlacklistIp,
      removeBlacklistIp,
      saveIpControlPolicy,
      saveRoleControlPolicy,
      saveTimeControlPolicy
    }
  }
}
</script>

<style lang="scss" scoped>
.access-control-policy {
  .card-header {
    display: flex;
    justify-content: space-between;
    align-items: center;
  }

  .help-text {
    font-size: 12px;
    color: #909399;
    margin-top: 4px;
  }

  .mr-2 {
    margin-right: 8px;
  }

  .mb-2 {
    margin-bottom: 8px;
  }

  .mr-1 {
    margin-right: 4px;
  }

  .mx-2 {
    margin: 0 8px;
  }

  .mt-4 {
    margin-top: 16px;
  }
}
</style> 