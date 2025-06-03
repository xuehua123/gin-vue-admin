<template>
  <div>
    <warning-bar title="设备日志管理 - 查看用户设备登录记录，管理在线设备状态" />
    
    <!-- 统计卡片 -->
    <div class="stats-cards" v-if="statsData">
      <el-row :gutter="20" style="margin-bottom: 20px;">
        <el-col :span="6">
          <el-card class="stats-card">
            <div class="stats-content">
              <div class="stats-icon primary">
                <el-icon><Monitor /></el-icon>
              </div>
              <div class="stats-info">
                <div class="stats-value">{{ statsData.totalLogins }}</div>
                <div class="stats-label">总登录次数</div>
              </div>
            </div>
          </el-card>
        </el-col>
        <el-col :span="6">
          <el-card class="stats-card">
            <div class="stats-content">
              <div class="stats-icon success">
                <el-icon><Connection /></el-icon>
              </div>
              <div class="stats-info">
                <div class="stats-value">{{ statsData.currentOnlineCount }}</div>
                <div class="stats-label">当前在线设备</div>
              </div>
            </div>
          </el-card>
        </el-col>
        <el-col :span="6">
          <el-card class="stats-card">
            <div class="stats-content">
              <div class="stats-icon warning">
                <el-icon><Cellphone /></el-icon>
              </div>
              <div class="stats-info">
                <div class="stats-value">{{ statsData.uniqueDevicesCount }}</div>
                <div class="stats-label">唯一设备数</div>
              </div>
            </div>
          </el-card>
        </el-col>
        <el-col :span="6">
          <el-card class="stats-card">
            <div class="stats-content">
              <div class="stats-icon info">
                <el-icon><Timer /></el-icon>
              </div>
              <div class="stats-info">
                <div class="stats-value">{{ formatDuration(statsData.averageSessionDuration) }}</div>
                <div class="stats-label">平均会话时长</div>
              </div>
            </div>
          </el-card>
        </el-col>
      </el-row>
    </div>

    <!-- 搜索条件 -->
    <div class="gva-search-box">
      <el-form ref="searchForm" :inline="true" :model="searchInfo">
        <el-form-item label="用户ID">
          <el-input v-model="searchInfo.userId" placeholder="用户ID" clearable />
        </el-form-item>
        <el-form-item label="客户端ID">
          <el-input v-model="searchInfo.clientId" placeholder="客户端ID" clearable />
        </el-form-item>
        <el-form-item label="设备型号">
          <el-input v-model="searchInfo.deviceModel" placeholder="设备型号" clearable />
        </el-form-item>
        <el-form-item label="IP地址">
          <el-input v-model="searchInfo.ipAddress" placeholder="IP地址" clearable />
        </el-form-item>
        <el-form-item label="登录时间">
          <el-date-picker
            v-model="loginTimeRange"
            type="datetimerange"
            range-separator="至"
            start-placeholder="开始时间"
            end-placeholder="结束时间"
            @change="handleTimeRangeChange"
            format="YYYY-MM-DD HH:mm:ss"
            value-format="YYYY-MM-DD HH:mm:ss"
          />
        </el-form-item>
        <el-form-item label="状态">
          <el-select v-model="searchInfo.onlineOnly" placeholder="选择状态" clearable>
            <el-option label="全部" :value="false" />
            <el-option label="仅在线" :value="true" />
          </el-select>
        </el-form-item>
        <el-form-item>
          <el-button type="primary" icon="search" @click="onSubmit">
            查询
          </el-button>
          <el-button icon="refresh" @click="onReset">
            重置
          </el-button>
          <el-button type="success" icon="refresh" @click="refreshData">
            刷新
          </el-button>
        </el-form-item>
      </el-form>
    </div>

    <!-- 操作按钮 -->
    <div class="gva-table-box">
      <div class="gva-btn-list">
        <el-button type="warning" icon="warning" @click="batchForceLogout" :disabled="!selectedRows.length">
          批量强制下线 ({{ selectedRows.length }})
        </el-button>
        <el-button type="info" icon="download" @click="exportLogs">
          导出日志
        </el-button>
        <el-button type="primary" icon="refresh" @click="getStatsData">
          刷新统计
        </el-button>
      </div>

      <!-- 设备日志表格 -->
      <el-table 
        :data="tableData" 
        row-key="ID" 
        v-loading="loading"
        @selection-change="handleSelectionChange"
        @sort-change="handleSortChange"
      >
        <el-table-column type="selection" width="55" />
        
        <el-table-column align="left" label="ID" min-width="80" prop="ID" sortable="custom" />
        
        <el-table-column align="left" label="用户ID" min-width="120" prop="userId">
          <template #default="scope">
            <el-link type="primary" @click="viewUserDetails(scope.row.userId)">
              {{ scope.row.userId }}
            </el-link>
          </template>
        </el-table-column>
        
        <el-table-column align="left" label="客户端ID" min-width="150" prop="clientId" show-overflow-tooltip />
        
        <el-table-column align="left" label="设备信息" min-width="200">
          <template #default="scope">
            <div class="device-info">
              <div class="device-model">
                <el-tag type="primary" size="small">{{ scope.row.deviceModel }}</el-tag>
              </div>
              <div class="device-details">
                <span class="device-os">{{ scope.row.deviceOs }}</span>
                <span class="app-version">v{{ scope.row.appVersion }}</span>
              </div>
            </div>
          </template>
        </el-table-column>
        
        <el-table-column align="left" label="网络信息" min-width="150">
          <template #default="scope">
            <div class="network-info">
              <div class="ip-address">
                <el-text size="small">IP: {{ scope.row.ipAddress }}</el-text>
              </div>
              <div class="location" v-if="scope.row.location">
                <el-text size="small" type="info">{{ scope.row.location }}</el-text>
              </div>
            </div>
          </template>
        </el-table-column>
        
        <el-table-column align="center" label="状态" min-width="100">
          <template #default="scope">
            <el-tag 
              :type="scope.row.logoutAt ? 'info' : 'success'"
              effect="dark"
              size="small"
            >
              <el-icon>
                <component :is="scope.row.logoutAt ? 'CircleCloseFilled' : 'CircleCheckFilled'" />
              </el-icon>
              {{ scope.row.logoutAt ? '已下线' : '在线' }}
            </el-tag>
          </template>
        </el-table-column>
        
        <el-table-column align="center" label="登录时间" min-width="180" prop="loginAt" sortable="custom">
          <template #default="scope">
            <div class="time-info">
              <div>{{ formatTime(scope.row.loginAt) }}</div>
              <el-text size="small" type="info">{{ formatRelativeTime(scope.row.loginAt) }}</el-text>
            </div>
          </template>
        </el-table-column>
        
        <el-table-column align="center" label="下线时间" min-width="180" prop="logoutAt" sortable="custom">
          <template #default="scope">
            <div v-if="scope.row.logoutAt" class="time-info">
              <div>{{ formatTime(scope.row.logoutAt) }}</div>
              <el-text size="small" type="info">{{ formatRelativeTime(scope.row.logoutAt) }}</el-text>
            </div>
            <el-text v-else type="success" size="small">在线中</el-text>
          </template>
        </el-table-column>
        
        <el-table-column align="left" label="会话时长" min-width="120">
          <template #default="scope">
            <div class="session-duration">
              <div v-if="scope.row.logoutAt">
                {{ calculateSessionDuration(scope.row.loginAt, scope.row.logoutAt) }}
              </div>
              <div v-else class="online-duration">
                {{ calculateOnlineDuration(scope.row.loginAt) }}
                <el-text size="small" type="success">(进行中)</el-text>
              </div>
            </div>
          </template>
        </el-table-column>
        
        <el-table-column align="left" label="下线原因" min-width="150" prop="logoutReason">
          <template #default="scope">
            <el-tag 
              v-if="scope.row.logoutReason" 
              :type="getLogoutReasonType(scope.row.logoutReason)"
              size="small"
            >
              {{ getLogoutReasonText(scope.row.logoutReason) }}
            </el-tag>
            <el-text v-else type="info" size="small">-</el-text>
          </template>
        </el-table-column>
        
        <el-table-column label="操作" min-width="200" fixed="right">
          <template #default="scope">
            <el-button 
              v-if="!scope.row.logoutAt" 
              type="warning" 
              link 
              icon="switch-button" 
              @click="forceLogoutSingle(scope.row)"
            >
              强制下线
            </el-button>
            <el-button type="primary" link icon="view" @click="viewDeviceDetails(scope.row)">
              详情
            </el-button>
            <el-button type="info" link icon="location" @click="viewIPLocation(scope.row)">
              IP归属
            </el-button>
            <el-button type="success" link icon="document" @click="viewUserAgent(scope.row)">
              User Agent
            </el-button>
          </template>
        </el-table-column>
      </el-table>

      <!-- 分页 -->
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
      width="600px"
      :close-on-click-modal="false"
    >
      <div class="force-logout-content">
        <el-alert 
          title="警告" 
          type="warning" 
          description="此操作将强制指定设备下线，请谨慎操作！" 
          :closable="false"
          style="margin-bottom: 20px;"
        />
        
        <div v-if="selectedRows.length > 1" class="batch-logout-info">
          <el-text size="large" type="warning">批量下线操作</el-text>
          <p>将要下线 <strong>{{ selectedRows.length }}</strong> 个设备</p>
          <el-table :data="selectedRows" max-height="200" size="small">
            <el-table-column prop="userId" label="用户ID" width="120" />
            <el-table-column prop="deviceModel" label="设备型号" width="150" />
            <el-table-column prop="ipAddress" label="IP地址" width="120" />
          </el-table>
        </div>
        
        <div v-else-if="forceLogoutInfo.deviceModel" class="single-logout-info">
          <el-descriptions :column="2" border>
            <el-descriptions-item label="用户ID">{{ forceLogoutInfo.userId }}</el-descriptions-item>
            <el-descriptions-item label="客户端ID">{{ forceLogoutInfo.clientId }}</el-descriptions-item>
            <el-descriptions-item label="设备型号">{{ forceLogoutInfo.deviceModel }}</el-descriptions-item>
            <el-descriptions-item label="IP地址">{{ forceLogoutInfo.ipAddress }}</el-descriptions-item>
          </el-descriptions>
        </div>
        
        <el-form :model="forceLogoutInfo" label-width="100px" style="margin-top: 20px;">
          <el-form-item label="下线原因" required>
            <el-select v-model="forceLogoutInfo.reason" placeholder="选择下线原因" style="width: 100%;">
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
          <el-button type="danger" @click="confirmForceLogout">
            确认下线{{ selectedRows.length > 1 ? ` (${selectedRows.length}个设备)` : '' }}
          </el-button>
        </div>
      </template>
    </el-dialog>

    <!-- 设备详情对话框 -->
    <el-dialog
      v-model="deviceDetailsDialog"
      title="设备详情"
      width="700px"
    >
      <div v-if="deviceDetails">
        <el-descriptions :column="2" border>
          <el-descriptions-item label="设备ID">{{ deviceDetails.ID }}</el-descriptions-item>
          <el-descriptions-item label="用户ID">{{ deviceDetails.userId }}</el-descriptions-item>
          <el-descriptions-item label="客户端ID">{{ deviceDetails.clientId }}</el-descriptions-item>
          <el-descriptions-item label="设备指纹">{{ deviceDetails.deviceFingerprint || '-' }}</el-descriptions-item>
          <el-descriptions-item label="设备型号">{{ deviceDetails.deviceModel }}</el-descriptions-item>
          <el-descriptions-item label="操作系统">{{ deviceDetails.deviceOs }}</el-descriptions-item>
          <el-descriptions-item label="应用版本">{{ deviceDetails.appVersion }}</el-descriptions-item>
          <el-descriptions-item label="IP地址">{{ deviceDetails.ipAddress }}</el-descriptions-item>
          <el-descriptions-item label="登录时间">{{ formatTime(deviceDetails.loginAt) }}</el-descriptions-item>
          <el-descriptions-item label="下线时间">{{ deviceDetails.logoutAt ? formatTime(deviceDetails.logoutAt) : '在线中' }}</el-descriptions-item>
          <el-descriptions-item label="下线原因">{{ deviceDetails.logoutReason || '-' }}</el-descriptions-item>
          <el-descriptions-item label="会话时长">
            {{ deviceDetails.logoutAt 
              ? calculateSessionDuration(deviceDetails.loginAt, deviceDetails.logoutAt)
              : calculateOnlineDuration(deviceDetails.loginAt) + ' (进行中)'
            }}
          </el-descriptions-item>
        </el-descriptions>
        
        <div style="margin-top: 20px;" v-if="deviceDetails.userAgent">
          <el-text size="large">User Agent</el-text>
          <el-input 
            v-model="deviceDetails.userAgent" 
            type="textarea" 
            :rows="3" 
            readonly 
            style="margin-top: 10px;"
          />
        </div>
      </div>
    </el-dialog>

    <!-- IP归属地对话框 -->
    <el-dialog
      v-model="ipLocationDialog"
      title="IP归属地信息"
      width="500px"
    >
      <div v-if="ipLocationInfo">
        <el-descriptions :column="1" border>
          <el-descriptions-item label="IP地址">{{ ipLocationInfo.ip }}</el-descriptions-item>
          <el-descriptions-item label="国家">{{ ipLocationInfo.country || '-' }}</el-descriptions-item>
          <el-descriptions-item label="省份">{{ ipLocationInfo.province || '-' }}</el-descriptions-item>
          <el-descriptions-item label="城市">{{ ipLocationInfo.city || '-' }}</el-descriptions-item>
          <el-descriptions-item label="ISP">{{ ipLocationInfo.isp || '-' }}</el-descriptions-item>
        </el-descriptions>
      </div>
      <div v-else v-loading="ipLocationLoading">
        <el-empty description="正在查询IP归属地信息..." />
      </div>
    </el-dialog>
  </div>
</template>

<script setup>
import { ref, reactive, computed, onMounted } from 'vue'
import { ElMessage, ElMessageBox } from 'element-plus'
import { useRouter, useRoute } from 'vue-router'
import { getDeviceLogsList, forceLogoutDevice, getDeviceLogStats } from '@/api/deviceLog'
import { Monitor, Connection, Cellphone, Timer, CircleCheckFilled, CircleCloseFilled } from '@element-plus/icons-vue'
import WarningBar from '@/components/warningBar/warningBar.vue'

defineOptions({
  name: 'DeviceLogManagement'
})

const router = useRouter()
const route = useRoute()

// 状态管理
const loading = ref(false)
const statsData = ref(null)
const selectedRows = ref([])

// 搜索条件
const searchInfo = reactive({
  userId: '',
  clientId: '',
  deviceModel: '',
  ipAddress: '',
  loginTimeStart: null,
  loginTimeEnd: null,
  onlineOnly: false
})

const loginTimeRange = ref([])

// 分页
const page = ref(1)
const total = ref(0)
const pageSize = ref(10)
const tableData = ref([])

// 排序
const sortInfo = reactive({
  prop: '',
  order: ''
})

// 对话框状态
const forceLogoutDialog = ref(false)
const deviceDetailsDialog = ref(false)
const ipLocationDialog = ref(false)

// 对话框数据
const forceLogoutInfo = reactive({
  userId: '',
  clientId: '',
  deviceModel: '',
  ipAddress: '',
  reason: '',
  remark: ''
})

const deviceDetails = ref(null)
const ipLocationInfo = ref(null)
const ipLocationLoading = ref(false)

// 初始化：如果有路由参数，设置搜索条件
onMounted(() => {
  if (route.query.userId) {
    searchInfo.userId = route.query.userId
  }
  if (route.query.userName) {
    // 显示用户名信息
    ElMessage.info(`正在查看用户 ${route.query.userName} 的设备日志`)
  }
  initPage()
})

// 格式化时间
const formatTime = (time) => {
  if (!time) return '-'
  return new Date(time).toLocaleString('zh-CN')
}

// 格式化相对时间
const formatRelativeTime = (time) => {
  if (!time) return '-'
  const now = new Date()
  const target = new Date(time)
  const diff = now - target
  
  const seconds = Math.floor(diff / 1000)
  const minutes = Math.floor(seconds / 60)
  const hours = Math.floor(minutes / 60)
  const days = Math.floor(hours / 24)
  
  if (days > 0) return `${days}天前`
  if (hours > 0) return `${hours}小时前`
  if (minutes > 0) return `${minutes}分钟前`
  return `${seconds}秒前`
}

// 格式化持续时间
const formatDuration = (seconds) => {
  if (!seconds) return '-'
  
  const hours = Math.floor(seconds / 3600)
  const minutes = Math.floor((seconds % 3600) / 60)
  const secs = seconds % 60
  
  if (hours > 0) {
    return `${hours}时${minutes}分${secs}秒`
  } else if (minutes > 0) {
    return `${minutes}分${secs}秒`
  } else {
    return `${secs}秒`
  }
}

// 计算会话时长
const calculateSessionDuration = (loginAt, logoutAt) => {
  if (!loginAt || !logoutAt) return '-'
  const login = new Date(loginAt)
  const logout = new Date(logoutAt)
  const diff = logout - login
  return formatDuration(Math.floor(diff / 1000))
}

// 计算在线时长
const calculateOnlineDuration = (loginAt) => {
  if (!loginAt) return '-'
  const login = new Date(loginAt)
  const now = new Date()
  const diff = now - login
  return formatDuration(Math.floor(diff / 1000))
}

// 获取下线原因类型
const getLogoutReasonType = (reason) => {
  const typeMap = {
    'user_logout': 'success',
    'kicked_by_new_login': 'warning',
    'jwt_expired': 'info',
    'session_revoked': 'danger',
    'admin_forced_logout': 'danger',
    'security_logout': 'danger',
    'violation_logout': 'danger',
    'maintenance_logout': 'warning'
  }
  return typeMap[reason] || 'info'
}

// 获取下线原因文本
const getLogoutReasonText = (reason) => {
  const textMap = {
    'user_logout': '用户主动下线',
    'kicked_by_new_login': '被新登录挤下线',
    'jwt_expired': 'Token过期',
    'session_revoked': '会话撤销',
    'admin_forced_logout': '管理员强制下线',
    'security_logout': '安全原因下线',
    'violation_logout': '违规操作下线',
    'maintenance_logout': '系统维护下线'
  }
  return textMap[reason] || reason
}

// 时间范围改变处理
const handleTimeRangeChange = (value) => {
  if (value && value.length === 2) {
    searchInfo.loginTimeStart = value[0]
    searchInfo.loginTimeEnd = value[1]
  } else {
    searchInfo.loginTimeStart = null
    searchInfo.loginTimeEnd = null
  }
}

// 搜索和重置
const onSubmit = () => {
  page.value = 1
  getTableData()
}

const onReset = () => {
  Object.assign(searchInfo, {
    userId: '',
    clientId: '',
    deviceModel: '',
    ipAddress: '',
    loginTimeStart: null,
    loginTimeEnd: null,
    onlineOnly: false
  })
  loginTimeRange.value = []
  getTableData()
}

// 刷新数据
const refreshData = () => {
  ElMessage.success('正在刷新数据...')
  getTableData()
  getStatsData()
}

// 分页处理
const handleSizeChange = (val) => {
  pageSize.value = val
  getTableData()
}

const handleCurrentChange = (val) => {
  page.value = val
  getTableData()
}

// 排序处理
const handleSortChange = ({ prop, order }) => {
  sortInfo.prop = prop
  sortInfo.order = order
  getTableData()
}

// 选择处理
const handleSelectionChange = (selection) => {
  selectedRows.value = selection.filter(row => !row.logoutAt) // 只能选择在线的设备
}

// 获取表格数据
const getTableData = async () => {
  loading.value = true
  try {
    const params = {
      page: page.value,
      pageSize: pageSize.value,
      ...searchInfo
    }
    
    // 添加排序参数
    if (sortInfo.prop) {
      params.sortField = sortInfo.prop
      params.sortOrder = sortInfo.order === 'ascending' ? 'asc' : 'desc'
    }

    const res = await getDeviceLogsList(params)
    if (res.code === 0) {
      tableData.value = res.data.list || []
      total.value = res.data.total
      page.value = res.data.page
      pageSize.value = res.data.pageSize
    }
  } catch (error) {
    ElMessage.error('获取设备日志失败：' + error.message)
  } finally {
    loading.value = false
  }
}

// 获取统计数据
const getStatsData = async () => {
  try {
    const res = await getDeviceLogStats(searchInfo.userId)
    if (res.code === 0) {
      statsData.value = res.data
    }
  } catch (error) {
    ElMessage.error('获取统计数据失败：' + error.message)
  }
}

// 单个强制下线
const forceLogoutSingle = (row) => {
  Object.assign(forceLogoutInfo, {
    userId: row.userId,
    clientId: row.clientId,
    deviceModel: row.deviceModel,
    ipAddress: row.ipAddress,
    reason: '',
    remark: ''
  })
  selectedRows.value = [row]
  forceLogoutDialog.value = true
}

// 批量强制下线
const batchForceLogout = () => {
  if (selectedRows.value.length === 0) {
    ElMessage.warning('请先选择要下线的设备')
    return
  }
  
  Object.assign(forceLogoutInfo, {
    userId: '',
    clientId: '',
    deviceModel: '',
    ipAddress: '',
    reason: '',
    remark: ''
  })
  forceLogoutDialog.value = true
}

// 确认强制下线
const confirmForceLogout = async () => {
  if (!forceLogoutInfo.reason) {
    ElMessage.warning('请选择下线原因')
    return
  }

  try {
    const reasonText = forceLogoutInfo.remark 
      ? `${forceLogoutInfo.reason} - ${forceLogoutInfo.remark}`
      : forceLogoutInfo.reason

    if (selectedRows.value.length === 1) {
      // 单个下线
      const device = selectedRows.value[0]
      const res = await forceLogoutDevice({
        userId: device.userId,
        clientId: device.clientId,
        reason: reasonText
      })
      
      if (res.code === 0) {
        ElMessage.success('强制下线成功')
      }
    } else {
      // 批量下线
      const promises = selectedRows.value.map(device => 
        forceLogoutDevice({
          userId: device.userId,
          clientId: device.clientId,
          reason: reasonText
        })
      )
      
      const results = await Promise.allSettled(promises)
      const successful = results.filter(r => r.status === 'fulfilled' && r.value.code === 0).length
      const failed = results.length - successful
      
      if (failed === 0) {
        ElMessage.success(`批量下线成功，共处理 ${successful} 个设备`)
      } else {
        ElMessage.warning(`批量下线完成，成功 ${successful} 个，失败 ${failed} 个`)
      }
    }
    
    closeForceLogoutDialog()
    await getTableData()
    await getStatsData()
  } catch (error) {
    ElMessage.error('强制下线失败：' + error.message)
  }
}

// 关闭强制下线对话框
const closeForceLogoutDialog = () => {
  forceLogoutDialog.value = false
  selectedRows.value = []
  Object.assign(forceLogoutInfo, {
    userId: '',
    clientId: '',
    deviceModel: '',
    ipAddress: '',
    reason: '',
    remark: ''
  })
}

// 查看设备详情
const viewDeviceDetails = (row) => {
  deviceDetails.value = row
  deviceDetailsDialog.value = true
}

// 查看用户详情
const viewUserDetails = (userId) => {
  router.push({
    name: 'UserEnhanced',
    query: { userId }
  })
}

// 查看IP归属地
const viewIPLocation = async (row) => {
  ipLocationDialog.value = true
  ipLocationLoading.value = true
  ipLocationInfo.value = null
  
  try {
    // 这里应该调用IP归属地查询API
    // 暂时模拟数据
    setTimeout(() => {
      ipLocationInfo.value = {
        ip: row.ipAddress,
        country: '中国',
        province: '广东省', 
        city: '深圳市',
        isp: '中国电信'
      }
      ipLocationLoading.value = false
    }, 1000)
  } catch (error) {
    ElMessage.error('查询IP归属地失败：' + error.message)
    ipLocationLoading.value = false
  }
}

// 查看User Agent
const viewUserAgent = (row) => {
  ElMessageBox.alert(row.userAgent || '无User Agent信息', 'User Agent', {
    confirmButtonText: '确定',
    type: 'info'
  })
}

// 导出日志
const exportLogs = () => {
  ElMessage.info('导出功能开发中...')
}

// 初始化页面
const initPage = async () => {
  await getTableData()
  await getStatsData()
}

</script>

<style lang="scss" scoped>
.stats-cards {
  margin-bottom: 20px;
  
  .stats-card {
    .stats-content {
      display: flex;
      align-items: center;
      
      .stats-icon {
        width: 60px;
        height: 60px;
        border-radius: 12px;
        display: flex;
        align-items: center;
        justify-content: center;
        margin-right: 16px;
        
        .el-icon {
          font-size: 24px;
          color: white;
        }
        
        &.primary {
          background: linear-gradient(135deg, #667eea 0%, #764ba2 100%);
        }
        
        &.success {
          background: linear-gradient(135deg, #11998e 0%, #38ef7d 100%);
        }
        
        &.warning {
          background: linear-gradient(135deg, #f093fb 0%, #f5576c 100%);
        }
        
        &.info {
          background: linear-gradient(135deg, #4facfe 0%, #00f2fe 100%);
        }
      }
      
      .stats-info {
        .stats-value {
          font-size: 28px;
          font-weight: bold;
          color: #303133;
        }
        
        .stats-label {
          font-size: 14px;
          color: #909399;
          margin-top: 4px;
        }
      }
    }
  }
}

.device-info {
  .device-model {
    margin-bottom: 8px;
  }
  
  .device-details {
    display: flex;
    gap: 8px;
    font-size: 12px;
    color: #909399;
    
    .device-os {
      background: #f0f2f5;
      padding: 2px 6px;
      border-radius: 4px;
    }
    
    .app-version {
      background: #e8f4fd;
      color: #1890ff;
      padding: 2px 6px;
      border-radius: 4px;
    }
  }
}

.network-info {
  .ip-address {
    margin-bottom: 4px;
  }
  
  .location {
    font-size: 12px;
  }
}

.time-info {
  text-align: center;
  
  .el-text {
    display: block;
    margin-top: 4px;
  }
}

.session-duration {
  text-align: center;
  
  .online-duration {
    color: #67c23a;
    font-weight: 500;
  }
}

.force-logout-content {
  .batch-logout-info {
    margin-bottom: 20px;
    
    p {
      margin: 10px 0;
      color: #e6a23c;
    }
  }
  
  .single-logout-info {
    margin-bottom: 20px;
  }
}

.gva-btn-list {
  margin-bottom: 16px;
}
</style> 