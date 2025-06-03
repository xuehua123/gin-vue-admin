<template>
  <div>
    <warning-bar title="注：右上角头像下拉可切换角色" />
    <div class="gva-search-box">
      <el-form ref="searchForm" :inline="true" :model="searchInfo">
        <el-form-item label="用户名">
          <el-input v-model="searchInfo.username" placeholder="用户名" />
        </el-form-item>
        <el-form-item label="昵称">
          <el-input v-model="searchInfo.nickname" placeholder="昵称" />
        </el-form-item>
        <el-form-item label="手机号">
          <el-input v-model="searchInfo.phone" placeholder="手机号" />
        </el-form-item>
        <el-form-item label="邮箱">
          <el-input v-model="searchInfo.email" placeholder="邮箱" />
        </el-form-item>
        <el-form-item label="在线状态">
          <el-select v-model="searchInfo.onlineStatus" placeholder="选择状态" clearable>
            <el-option label="全部" value="" />
            <el-option label="在线" value="online" />
            <el-option label="离线" value="offline" />
          </el-select>
        </el-form-item>
        <el-form-item>
          <el-button type="primary" icon="search" @click="onSubmit">
            查询
          </el-button>
          <el-button icon="refresh" @click="onReset"> 重置 </el-button>
          <el-button type="success" icon="refresh" @click="refreshStatus">
            刷新状态
          </el-button>
        </el-form-item>
      </el-form>
    </div>
    <div class="gva-table-box">
      <div class="gva-btn-list">
        <el-button type="primary" icon="plus" @click="addUser">
          新增用户
        </el-button>
        <el-button type="warning" icon="view" @click="viewDeviceLogs">
          设备日志管理
        </el-button>
      </div>
      <el-table :data="tableData" row-key="ID" v-loading="loading">
        <el-table-column align="left" label="头像" min-width="75">
          <template #default="scope">
            <CustomPic style="margin-top: 8px" :pic-src="scope.row.headerImg" />
          </template>
        </el-table-column>
        <el-table-column align="left" label="ID" min-width="50" prop="ID" />
        <el-table-column align="left" label="用户名" min-width="150" prop="userName" />
        <el-table-column align="left" label="昵称" min-width="150" prop="nickName" />
        
        <!-- 在线状态列 -->
        <el-table-column align="center" label="在线状态" min-width="120">
          <template #default="scope">
            <div class="online-status-container">
              <el-tag 
                :type="scope.row.onlineStatus?.isOnline ? 'success' : 'info'"
                effect="dark"
                size="small"
              >
                <el-icon class="status-icon">
                  <component :is="scope.row.onlineStatus?.isOnline ? 'CircleCheckFilled' : 'CircleCloseFilled'" />
                </el-icon>
                {{ scope.row.onlineStatus?.isOnline ? '在线' : '离线' }}
              </el-tag>
              <div v-if="scope.row.onlineStatus?.isOnline" class="online-count">
                {{ scope.row.onlineStatus.onlineCount }}个设备
              </div>
            </div>
          </template>
        </el-table-column>

        <!-- 设备信息列 -->
        <el-table-column align="left" label="设备信息" min-width="200">
          <template #default="scope">
            <div v-if="scope.row.deviceInfo && scope.row.deviceInfo.length > 0">
              <el-popover 
                placement="right" 
                :width="400" 
                trigger="hover"
                :content="`共${scope.row.deviceInfo.length}台设备`"
              >
                <template #reference>
                  <el-tag 
                    v-for="device in scope.row.deviceInfo.slice(0, 2)" 
                    :key="device.clientId"
                    :type="device.isOnline ? 'success' : 'warning'"
                    size="small"
                    style="margin: 2px;"
                  >
                    {{ device.deviceModel }}
                  </el-tag>
                  <el-tag v-if="scope.row.deviceInfo.length > 2" type="info" size="small">
                    +{{ scope.row.deviceInfo.length - 2 }}
                  </el-tag>
                </template>
                <template #default>
                  <div class="device-info-popup">
                    <div v-for="device in scope.row.deviceInfo" :key="device.clientId" class="device-item">
                      <div class="device-header">
                        <el-tag :type="device.isOnline ? 'success' : 'warning'" size="small">
                          {{ device.isOnline ? '在线' : '离线' }}
                        </el-tag>
                        <span class="device-model">{{ device.deviceModel }}</span>
                      </div>
                      <div class="device-details">
                        <p><strong>系统:</strong> {{ device.deviceOs }}</p>
                        <p><strong>版本:</strong> {{ device.appVersion }}</p>
                        <p><strong>IP:</strong> {{ device.ipAddress }}</p>
                        <p><strong>登录时间:</strong> {{ formatTime(device.loginTime) }}</p>
                        <p v-if="device.isOnline"><strong>最后活跃:</strong> {{ formatTime(device.lastActiveTime) }}</p>
                      </div>
                      <div class="device-actions">
                        <el-button 
                          v-if="device.isOnline" 
                          type="danger" 
                          size="small" 
                                                     @click="forceLogoutDeviceAction(scope.row, device)"
                        >
                          强制下线
                        </el-button>
                      </div>
                    </div>
                  </div>
                </template>
              </el-popover>
            </div>
            <el-text v-else type="info" size="small">暂无设备</el-text>
          </template>
        </el-table-column>

        <!-- 角色状态列 -->
        <el-table-column align="center" label="角色状态" min-width="150">
          <template #default="scope">
            <div v-if="scope.row.roleInfo">
              <el-tag 
                :type="getRoleTagType(scope.row.roleInfo.currentRole)" 
                size="small"
                v-if="scope.row.roleInfo.currentRole !== 'none'"
              >
                {{ getRoleText(scope.row.roleInfo.currentRole) }}
              </el-tag>
              <div v-if="scope.row.roleInfo.currentRole === 'transmitter'" class="role-status">
                <el-text size="small" type="primary">NFC: {{ scope.row.roleInfo.nfcStatus || '未知' }}</el-text>
              </div>
              <div v-if="scope.row.roleInfo.currentRole === 'receiver'" class="role-status">
                <el-text size="small" type="success">HCE: {{ scope.row.roleInfo.hceStatus || '未知' }}</el-text>
              </div>
              <el-text v-if="scope.row.roleInfo.currentRole === 'none'" type="info" size="small">
                未选择角色
              </el-text>
            </div>
            <el-text v-else type="info" size="small">无角色信息</el-text>
          </template>
        </el-table-column>

        <el-table-column align="left" label="手机号" min-width="180" prop="phone" />
        <el-table-column align="left" label="邮箱" min-width="180" prop="email" />
        
        <el-table-column align="left" label="用户角色" min-width="200">
          <template #default="scope">
            <el-cascader
              v-model="scope.row.authorityIds"
              :options="authOptions"
              :show-all-levels="false"
              collapse-tags
              :props="{
                multiple: true,
                checkStrictly: true,
                label: 'authorityName',
                value: 'authorityId',
                disabled: 'disabled',
                emitPath: false
              }"
              :clearable="false"
              @visible-change="
                (flag) => {
                  changeAuthority(scope.row, flag, 0)
                }
              "
              @remove-tag="
                (removeAuth) => {
                  changeAuthority(scope.row, false, removeAuth)
                }
              "
            />
          </template>
        </el-table-column>
        
        <el-table-column align="left" label="启用" min-width="80">
          <template #default="scope">
            <el-switch
              v-model="scope.row.enable"
              inline-prompt
              :active-value="1"
              :inactive-value="2"
              @change="() => { switchEnable(scope.row) }"
            />
          </template>
        </el-table-column>

        <el-table-column label="操作" :min-width="280" fixed="right">
          <template #default="scope">
            <el-button type="primary" link icon="delete" @click="deleteUserFunc(scope.row)">
              删除
            </el-button>
            <el-button type="primary" link icon="edit" @click="openEdit(scope.row)">
              编辑
            </el-button>
            <el-button type="primary" link icon="magic-stick" @click="resetPasswordFunc(scope.row)">
              重置密码
            </el-button>
            <el-button 
              v-if="scope.row.onlineStatus?.isOnline" 
              type="warning" 
              link 
              icon="switch-button" 
              @click="forceLogoutUser(scope.row)"
            >
              强制下线
            </el-button>
            <el-button type="info" link icon="view" @click="viewUserDeviceLogs(scope.row)">
              设备记录
            </el-button>
          </template>
        </el-table-column>
      </el-table>
      <div class="gva-pagination">
        <el-pagination
          :current-page="page"
          :page-size="pageSize"
          :page-sizes="[10, 30, 50, 100]"
          :total="total"
          layout="total, sizes, prev, pager, next, jumper"
          @current-change="handleCurrentChange"
          @size-change="handleSizeChange"
        />
      </div>
    </div>

    <!-- 强制下线对话框 -->
    <el-dialog
      v-model="forceLogoutDialog"
      title="强制下线确认"
      width="500px"
      :close-on-click-modal="false"
    >
      <div class="force-logout-content">
        <el-alert 
          title="警告" 
          type="warning" 
          description="此操作将强制指定用户/设备下线，请谨慎操作！" 
          :closable="false"
          style="margin-bottom: 20px;"
        />
        <el-form :model="forceLogoutInfo" label-width="100px">
          <el-form-item label="用户名">
            <el-input v-model="forceLogoutInfo.userName" disabled />
          </el-form-item>
          <el-form-item label="设备型号" v-if="forceLogoutInfo.deviceModel">
            <el-input v-model="forceLogoutInfo.deviceModel" disabled />
          </el-form-item>
          <el-form-item label="下线原因">
            <el-select v-model="forceLogoutInfo.reason" placeholder="选择下线原因">
              <el-option label="管理员强制下线" value="admin_forced_logout" />
              <el-option label="安全原因下线" value="security_logout" />
              <el-option label="违规操作下线" value="violation_logout" />
              <el-option label="系统维护下线" value="maintenance_logout" />
              <el-option label="其他原因" value="other_reason" />
            </el-select>
          </el-form-item>
          <el-form-item label="备注说明">
            <el-input 
              v-model="forceLogoutInfo.remark" 
              type="textarea" 
              placeholder="请输入详细说明（可选）"
              :rows="3"
            />
          </el-form-item>
        </el-form>
      </div>
      <template #footer>
        <div class="dialog-footer">
          <el-button @click="closeForceLogoutDialog">取 消</el-button>
          <el-button type="danger" @click="confirmForceLogout">确认下线</el-button>
        </div>
      </template>
    </el-dialog>

    <!-- 重置密码对话框 -->
    <el-dialog
      v-model="resetPwdDialog"
      title="重置密码"
      width="500px"
      :close-on-click-modal="false"
      :close-on-press-escape="false"
    >
      <el-form :model="resetPwdInfo" ref="resetPwdForm" label-width="100px">
        <el-form-item label="用户账号">
          <el-input v-model="resetPwdInfo.userName" disabled />
        </el-form-item>
        <el-form-item label="用户昵称">
          <el-input v-model="resetPwdInfo.nickName" disabled />
        </el-form-item>
        <el-form-item label="新密码">
          <div class="flex w-full">
            <el-input class="flex-1" v-model="resetPwdInfo.password" placeholder="请输入新密码" show-password />
            <el-button type="primary" @click="generateRandomPassword" style="margin-left: 10px">
              生成随机密码
            </el-button>
          </div>
        </el-form-item>
      </el-form>
      <template #footer>
        <div class="dialog-footer">
          <el-button @click="closeResetPwdDialog">取 消</el-button>
          <el-button type="primary" @click="confirmResetPassword">确 定</el-button>
        </div>
      </template>
    </el-dialog>
    
    <!-- 用户编辑/添加抽屉 -->
    <el-drawer
      v-model="addUserDialog"
      :size="appStore.drawerSize"
      :show-close="false"
      :close-on-press-escape="false"
      :close-on-click-modal="false"
    >
      <template #header>
        <div class="flex justify-between items-center">
          <span class="text-lg">{{ dialogFlag === 'add' ? '新增' : '编辑' }}用户</span>
          <div>
            <el-button @click="closeAddUserDialog">取 消</el-button>
            <el-button type="primary" @click="enterAddUserDialog">确 定</el-button>
          </div>
        </div>
      </template>

      <el-form
        ref="userForm"
        :rules="rules"
        :model="userInfo"
        label-width="100px"
      >
        <el-form-item label="用户名" prop="userName">
          <el-input v-model="userInfo.userName" />
        </el-form-item>
        <el-form-item v-if="dialogFlag === 'add'" label="密码" prop="password">
          <el-input v-model="userInfo.password" show-password />
        </el-form-item>
        <el-form-item label="昵称" prop="nickName">
          <el-input v-model="userInfo.nickName" />
        </el-form-item>
        <el-form-item label="手机号" prop="phone">
          <el-input v-model="userInfo.phone" />
        </el-form-item>
        <el-form-item label="邮箱" prop="email">
          <el-input v-model="userInfo.email" />
        </el-form-item>
        <el-form-item label="用户角色" prop="authorityId">
          <el-cascader
            v-model="userInfo.authorityIds"
            :options="authOptions"
            :show-all-levels="false"
            collapse-tags
            :props="{
              multiple: true,
              checkStrictly: true,
              label: 'authorityName',
              value: 'authorityId',
              disabled: 'disabled',
              emitPath: false
            }"
            :clearable="false"
          />
        </el-form-item>
        <el-form-item label="启用" prop="disabled">
          <el-switch
            v-model="userInfo.enable"
            inline-prompt
            :active-value="1"
            :inactive-value="2"
          />
        </el-form-item>
        <el-form-item label="头像" label-width="80px">
          <SelectImage v-model="userInfo.headerImg" />
        </el-form-item>
      </el-form>
    </el-drawer>
  </div>
</template>

<script setup>
import { getUserList, setUserAuthorities, register, deleteUser, setUserInfo, resetPassword } from '@/api/user'
import { forceLogoutDevice } from '@/api/deviceLog'
import { getAuthorityList } from '@/api/authority'
import CustomPic from '@/components/customPic/index.vue'
import WarningBar from '@/components/warningBar/warningBar.vue'
import SelectImage from '@/components/selectImage/selectImage.vue'
import { nextTick, ref, watch } from 'vue'
import { ElMessage, ElMessageBox } from 'element-plus'
import { useAppStore } from "@/pinia"
import { useRouter } from 'vue-router'
import { CircleCheckFilled, CircleCloseFilled } from '@element-plus/icons-vue'

defineOptions({
  name: 'UserEnhanced'
})

const appStore = useAppStore()
const router = useRouter()

const loading = ref(false)
const searchInfo = ref({
  username: '',
  nickname: '',
  phone: '',
  email: '',
  onlineStatus: ''
})

// 搜索和重置
const onSubmit = () => {
  page.value = 1
  getTableData()
}

const onReset = () => {
  searchInfo.value = {
    username: '',
    nickname: '',
    phone: '',
    email: '',
    onlineStatus: ''
  }
  getTableData()
}

// 刷新状态
const refreshStatus = () => {
  ElMessage.success('正在刷新用户状态...')
  getTableData()
}

// 格式化时间
const formatTime = (time) => {
  if (!time) return '-'
  return new Date(time).toLocaleString('zh-CN')
}

// 获取角色标签类型
const getRoleTagType = (role) => {
  switch (role) {
    case 'transmitter': return 'primary'
    case 'receiver': return 'success'
    default: return 'info'
  }
}

// 获取角色文本
const getRoleText = (role) => {
  switch (role) {
    case 'transmitter': return '传卡端'
    case 'receiver': return '收卡端'
    default: return '无角色'
  }
}

// 强制下线相关
const forceLogoutDialog = ref(false)
const forceLogoutInfo = ref({
  userId: '',
  clientId: '',
  userName: '',
  deviceModel: '',
  reason: '',
  remark: ''
})

// 强制下线用户（所有设备）
const forceLogoutUser = (row) => {
  forceLogoutInfo.value = {
    userId: row.uuid,
    clientId: '',
    userName: row.userName,
    deviceModel: '所有设备',
    reason: 'admin_forced_logout',
    remark: ''
  }
  forceLogoutDialog.value = true
}

// 强制下线特定设备
const forceLogoutDeviceAction = (user, device) => {
  forceLogoutInfo.value = {
    userId: user.uuid,
    clientId: device.clientId,
    userName: user.userName,
    deviceModel: device.deviceModel,
    reason: 'admin_forced_logout',
    remark: ''
  }
  forceLogoutDialog.value = true
}

// 确认强制下线
const confirmForceLogout = async () => {
  if (!forceLogoutInfo.value.reason) {
    ElMessage.warning('请选择下线原因')
    return
  }

  try {
    const reasonText = forceLogoutInfo.value.remark 
      ? `${forceLogoutInfo.value.reason} - ${forceLogoutInfo.value.remark}`
      : forceLogoutInfo.value.reason

    const res = await forceLogoutDevice({
      userId: forceLogoutInfo.value.userId,
      clientId: forceLogoutInfo.value.clientId,
      reason: reasonText
    })

    if (res.code === 0) {
      ElMessage.success('强制下线成功')
      closeForceLogoutDialog()
      await getTableData() // 刷新数据
    }
  } catch (error) {
    ElMessage.error('强制下线失败：' + error.message)
  }
}

// 关闭强制下线对话框
const closeForceLogoutDialog = () => {
  forceLogoutInfo.value = {
    userId: '',
    clientId: '',
    userName: '',
    deviceModel: '',
    reason: '',
    remark: ''
  }
  forceLogoutDialog.value = false
}

// 查看用户设备日志
const viewUserDeviceLogs = (row) => {
  router.push({
    name: 'DeviceLogManagement',
    query: { userId: row.uuid, userName: row.userName }
  })
}

// 查看设备日志管理
const viewDeviceLogs = () => {
  router.push({ name: 'DeviceLogManagement' })
}

// 分页相关
const page = ref(1)
const total = ref(0)
const pageSize = ref(10)
const tableData = ref([])

const handleSizeChange = (val) => {
  pageSize.value = val
  getTableData()
}

const handleCurrentChange = (val) => {
  page.value = val
  getTableData()
}

// 获取表格数据
const getTableData = async () => {
  loading.value = true
  try {
    const params = {
      page: page.value,
      pageSize: pageSize.value,
      ...searchInfo.value
    }

    // 如果搜索在线状态，进行过滤
    if (searchInfo.value.onlineStatus) {
      delete params.onlineStatus // 删除这个参数，因为后端可能不支持
    }

    const table = await getUserList(params)
    if (table.code === 0) {
      let list = table.data.list || []
      
      // 前端过滤在线状态
      if (searchInfo.value.onlineStatus) {
        list = list.filter(user => {
          const isOnline = user.onlineStatus?.isOnline
          return searchInfo.value.onlineStatus === 'online' ? isOnline : !isOnline
        })
      }
      
      tableData.value = list
      total.value = table.data.total
      page.value = table.data.page
      pageSize.value = table.data.pageSize
    }
  } catch (error) {
    ElMessage.error('获取用户列表失败：' + error.message)
  } finally {
    loading.value = false
  }
}

// 权限相关
const authOptions = ref([])
const setAuthorityIds = () => {
  tableData.value &&
    tableData.value.forEach((user) => {
      user.authorityIds =
        user.authorities &&
        user.authorities.map((i) => {
          return i.authorityId
        })
    })
}

const setAuthorityOptions = (AuthorityData, optionsData) => {
  AuthorityData &&
    AuthorityData.forEach((item) => {
      if (item.children && item.children.length) {
        const option = {
          authorityId: item.authorityId,
          authorityName: item.authorityName,
          children: []
        }
        setAuthorityOptions(item.children, option.children)
        optionsData.push(option)
      } else {
        const option = {
          authorityId: item.authorityId,
          authorityName: item.authorityName
        }
        optionsData.push(option)
      }
    })
}

const setOptions = (authData) => {
  authOptions.value = []
  setAuthorityOptions(authData, authOptions.value)
}

watch(
  () => tableData.value,
  () => {
    setAuthorityIds()
  }
)

// 删除用户
const deleteUserFunc = async (row) => {
  ElMessageBox.confirm('确定要删除吗?', '提示', {
    confirmButtonText: '确定',
    cancelButtonText: '取消',
    type: 'warning'
  }).then(async () => {
    const res = await deleteUser({ id: row.ID })
    if (res.code === 0) {
      ElMessage.success('删除成功')
      await getTableData()
    }
  })
}

// 重置密码相关
const resetPwdDialog = ref(false)
const resetPwdForm = ref(null)
const resetPwdInfo = ref({
  ID: '',
  userName: '',
  nickName: '',
  password: ''
})

const generateRandomPassword = () => {
  const chars = 'ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz0123456789!@#$%^&*'
  let password = ''
  for (let i = 0; i < 12; i++) {
    password += chars.charAt(Math.floor(Math.random() * chars.length))
  }
  resetPwdInfo.value.password = password
  navigator.clipboard.writeText(password).then(() => {
    ElMessage({
      type: 'success',
      message: '密码已复制到剪贴板'
    })
  }).catch(() => {
    ElMessage({
      type: 'error',
      message: '复制失败，请手动复制'
    })
  })
}

const resetPasswordFunc = (row) => {
  resetPwdInfo.value.ID = row.ID
  resetPwdInfo.value.userName = row.userName
  resetPwdInfo.value.nickName = row.nickName
  resetPwdInfo.value.password = ''
  resetPwdDialog.value = true
}

const confirmResetPassword = async () => {
  if (!resetPwdInfo.value.password) {
    ElMessage({
      type: 'warning',
      message: '请输入或生成密码'
    })
    return
  }
  
  const res = await resetPassword({
    ID: resetPwdInfo.value.ID,
    password: resetPwdInfo.value.password
  })
  
  if (res.code === 0) {
    ElMessage({
      type: 'success',
      message: res.msg || '密码重置成功'
    })
    resetPwdDialog.value = false
  } else {
    ElMessage({
      type: 'error',
      message: res.msg || '密码重置失败'
    })
  }
}

const closeResetPwdDialog = () => {
  resetPwdInfo.value.password = ''
  resetPwdDialog.value = false
}

// 用户编辑相关
const userInfo = ref({
  userName: '',
  password: '',
  nickName: '',
  headerImg: '',
  authorityId: '',
  authorityIds: [],
  enable: 1
})

const rules = ref({
  userName: [
    { required: true, message: '请输入用户名', trigger: 'blur' },
    { min: 5, message: '最低5位字符', trigger: 'blur' }
  ],
  password: [
    { required: true, message: '请输入用户密码', trigger: 'blur' },
    { min: 6, message: '最低6位字符', trigger: 'blur' }
  ],
  nickName: [{ required: true, message: '请输入用户昵称', trigger: 'blur' }],
  phone: [
    {
      pattern: /^1([38][0-9]|4[014-9]|[59][0-35-9]|6[2567]|7[0-8])\d{8}$/,
      message: '请输入合法手机号',
      trigger: 'blur'
    }
  ],
  email: [
    {
      pattern: /^([0-9A-Za-z\-_.]+)@([0-9a-z]+\.[a-z]{2,3}(\.[a-z]{2})?)$/g,
      message: '请输入正确的邮箱',
      trigger: 'blur'
    }
  ],
  authorityId: [
    { required: true, message: '请选择用户角色', trigger: 'blur' }
  ]
})

const userForm = ref(null)
const addUserDialog = ref(false)
const dialogFlag = ref('add')

const addUser = () => {
  dialogFlag.value = 'add'
  addUserDialog.value = true
}

const openEdit = (row) => {
  dialogFlag.value = 'edit'
  userInfo.value = JSON.parse(JSON.stringify(row))
  addUserDialog.value = true
}

const enterAddUserDialog = async () => {
  userInfo.value.authorityId = userInfo.value.authorityIds[0]
  userForm.value.validate(async (valid) => {
    if (valid) {
      const req = { ...userInfo.value }
      if (dialogFlag.value === 'add') {
        const res = await register(req)
        if (res.code === 0) {
          ElMessage({ type: 'success', message: '创建成功' })
          await getTableData()
          closeAddUserDialog()
        }
      }
      if (dialogFlag.value === 'edit') {
        const res = await setUserInfo(req)
        if (res.code === 0) {
          ElMessage({ type: 'success', message: '编辑成功' })
          await getTableData()
          closeAddUserDialog()
        }
      }
    }
  })
}

const closeAddUserDialog = () => {
  userForm.value.resetFields()
  userInfo.value.headerImg = ''
  userInfo.value.authorityIds = []
  addUserDialog.value = false
}

const tempAuth = {}
const changeAuthority = async (row, flag, removeAuth) => {
  if (flag) {
    if (!removeAuth) {
      tempAuth[row.ID] = [...row.authorityIds]
    }
    return
  }
  await nextTick()
  const res = await setUserAuthorities({
    ID: row.ID,
    authorityIds: row.authorityIds
  })
  if (res.code === 0) {
    ElMessage({ type: 'success', message: '角色设置成功' })
  } else {
    if (!removeAuth) {
      row.authorityIds = [...tempAuth[row.ID]]
      delete tempAuth[row.ID]
    } else {
      row.authorityIds = [removeAuth, ...row.authorityIds]
    }
  }
}

const switchEnable = async (row) => {
  userInfo.value = JSON.parse(JSON.stringify(row))
  await nextTick()
  const req = { ...userInfo.value }
  const res = await setUserInfo(req)
  if (res.code === 0) {
    ElMessage({
      type: 'success',
      message: `${req.enable === 2 ? '禁用' : '启用'}成功`
    })
    await getTableData()
    userInfo.value.headerImg = ''
    userInfo.value.authorityIds = []
  }
}

// 初始化
const initPage = async () => {
  getTableData()
  const res = await getAuthorityList()
  setOptions(res.data)
}

initPage()
</script>

<style lang="scss" scoped>
.online-status-container {
  display: flex;
  flex-direction: column;
  align-items: center;
  gap: 4px;
  
  .status-icon {
    margin-right: 4px;
  }
  
  .online-count {
    font-size: 12px;
    color: #909399;
  }
}

.device-info-popup {
  max-height: 300px;
  overflow-y: auto;
  
  .device-item {
    border-bottom: 1px solid #ebeef5;
    padding: 10px 0;
    
    &:last-child {
      border-bottom: none;
    }
    
    .device-header {
      display: flex;
      justify-content: space-between;
      align-items: center;
      margin-bottom: 8px;
      
      .device-model {
        font-weight: bold;
        font-size: 14px;
      }
    }
    
    .device-details {
      font-size: 12px;
      color: #606266;
      margin-bottom: 8px;
      
      p {
        margin: 2px 0;
      }
    }
    
    .device-actions {
      text-align: right;
    }
  }
}

.role-status {
  margin-top: 4px;
  font-size: 12px;
}

.force-logout-content {
  .el-alert {
    margin-bottom: 20px;
  }
}

.header-img-box {
  @apply w-52 h-52 border border-solid border-gray-300 rounded-xl flex justify-center items-center cursor-pointer;
}
</style> 