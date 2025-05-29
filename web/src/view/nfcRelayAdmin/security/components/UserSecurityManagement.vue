<template>
  <div class="user-security-management">
    <!-- 操作工具栏 -->
    <div class="toolbar">
      <el-row justify="space-between">
        <el-col :span="16">
          <el-space>
            <el-button 
              type="warning" 
              :disabled="selectedUsers.length === 0"
              @click="batchLockUsers"
            >
              <el-icon><Lock /></el-icon>
              批量锁定
            </el-button>
            <el-button 
              type="success" 
              :disabled="selectedUsers.length === 0"
              @click="batchUnlockUsers"
            >
              <el-icon><Unlock /></el-icon>
              批量解锁
            </el-button>
            <el-button @click="refreshList">
              <el-icon><Refresh /></el-icon>
              刷新
            </el-button>
          </el-space>
        </el-col>
        <el-col :span="8">
          <el-input
            v-model="searchKeyword"
            placeholder="搜索用户ID或用户名"
            clearable
            @clear="handleSearch"
            @keyup.enter="handleSearch"
          >
            <template #append>
              <el-button @click="handleSearch">
                <el-icon><Search /></el-icon>
              </el-button>
            </template>
          </el-input>
        </el-col>
      </el-row>
    </div>

    <!-- 用户安全档案列表 -->
    <el-card class="user-security-card">
      <template #header>
        <div class="card-header">
          <span>用户安全档案</span>
          <el-space>
            <el-select 
              v-model="statusFilter" 
              placeholder="状态过滤" 
              clearable
              @change="handleSearch"
            >
              <el-option label="正常" value="normal" />
              <el-option label="锁定" value="locked" />
              <el-option label="风险" value="risk" />
            </el-select>
            <el-select 
              v-model="riskLevelFilter" 
              placeholder="风险等级" 
              clearable
              @change="handleSearch"
            >
              <el-option label="低风险" value="low" />
              <el-option label="中风险" value="medium" />
              <el-option label="高风险" value="high" />
            </el-select>
          </el-space>
        </div>
      </template>
      
      <el-table
        ref="userTableRef"
        v-loading="loading"
        :data="userSecurityList"
        @selection-change="handleSelectionChange"
        row-key="userId"
        height="500"
      >
        <el-table-column type="selection" width="55" />
        <el-table-column prop="userId" label="用户ID" width="120" />
        <el-table-column prop="username" label="用户名" width="150" />
        <el-table-column prop="status" label="状态" width="100">
          <template #default="{ row }">
            <el-tag :type="getStatusTagType(row.status)">
              {{ getStatusText(row.status) }}
            </el-tag>
          </template>
        </el-table-column>
        <el-table-column prop="riskLevel" label="风险等级" width="120">
          <template #default="{ row }">
            <el-tag :type="getRiskLevelTagType(row.riskLevel)">
              {{ getRiskLevelText(row.riskLevel) }}
            </el-tag>
          </template>
        </el-table-column>
        <el-table-column prop="loginAttempts" label="登录尝试" width="100" />
        <el-table-column prop="failedAttempts" label="失败次数" width="100" />
        <el-table-column prop="lastLoginTime" label="最后登录" width="180">
          <template #default="{ row }">
            {{ formatDateTime(row.lastLoginTime) }}
          </template>
        </el-table-column>
        <el-table-column prop="lastLoginIp" label="最后登录IP" width="130" />
        <el-table-column prop="securityScore" label="安全评分" width="100">
          <template #default="{ row }">
            <el-progress 
              :percentage="row.securityScore" 
              :color="getScoreColor(row.securityScore)"
              :format="() => row.securityScore"
            />
          </template>
        </el-table-column>
        <el-table-column label="操作" width="200" fixed="right">
          <template #default="{ row }">
            <el-space>
              <el-button 
                v-if="row.status !== 'locked'"
                type="warning" 
                size="small"
                @click="lockUser(row)"
              >
                锁定
              </el-button>
              <el-button 
                v-if="row.status === 'locked'"
                type="success" 
                size="small"
                @click="unlockUser(row)"
              >
                解锁
              </el-button>
              <el-button 
                type="primary" 
                size="small"
                @click="editSecurityProfile(row)"
              >
                编辑
              </el-button>
              <el-button 
                type="info" 
                size="small"
                @click="viewSecurityDetails(row)"
              >
                详情
              </el-button>
            </el-space>
          </template>
        </el-table-column>
      </el-table>

      <!-- 分页 -->
      <div class="pagination-wrapper">
        <el-pagination
          v-model:current-page="pageInfo.page"
          v-model:page-size="pageInfo.pageSize"
          :total="pageInfo.total"
          :page-sizes="[10, 20, 50, 100]"
          layout="total, sizes, prev, pager, next, jumper"
          @size-change="handleSizeChange"
          @current-change="handleCurrentChange"
        />
      </div>
    </el-card>

    <!-- 编辑安全档案对话框 -->
    <el-dialog
      v-model="editDialogVisible"
      title="编辑用户安全档案"
      width="700px"
      :close-on-click-modal="false"
    >
      <el-form
        ref="editFormRef"
        :model="editForm"
        :rules="editRules"
        label-width="120px"
      >
        <el-row :gutter="20">
          <el-col :span="12">
            <el-form-item label="用户ID" prop="userId">
              <el-input v-model="editForm.userId" disabled />
            </el-form-item>
          </el-col>
          <el-col :span="12">
            <el-form-item label="用户名" prop="username">
              <el-input v-model="editForm.username" disabled />
            </el-form-item>
          </el-col>
        </el-row>
        
        <el-row :gutter="20">
          <el-col :span="12">
            <el-form-item label="风险等级" prop="riskLevel">
              <el-select v-model="editForm.riskLevel" style="width: 100%">
                <el-option label="低风险" value="low" />
                <el-option label="中风险" value="medium" />
                <el-option label="高风险" value="high" />
              </el-select>
            </el-form-item>
          </el-col>
          <el-col :span="12">
            <el-form-item label="安全评分" prop="securityScore">
              <el-input-number 
                v-model="editForm.securityScore" 
                :min="0" 
                :max="100" 
                style="width: 100%"
              />
            </el-form-item>
          </el-col>
        </el-row>

        <el-form-item label="安全标签" prop="securityTags">
          <el-select 
            v-model="editForm.securityTags" 
            multiple 
            filterable 
            allow-create
            placeholder="选择或输入安全标签"
            style="width: 100%"
          >
            <el-option label="可信用户" value="trusted" />
            <el-option label="高频用户" value="frequent" />
            <el-option label="异常行为" value="anomaly" />
            <el-option label="IP变化频繁" value="ip_change" />
            <el-option label="设备变化频繁" value="device_change" />
          </el-select>
        </el-form-item>

        <el-form-item label="备注信息" prop="notes">
          <el-input
            v-model="editForm.notes"
            type="textarea"
            :rows="3"
            placeholder="请输入备注信息"
          />
        </el-form-item>
      </el-form>
      
      <template #footer>
        <span class="dialog-footer">
          <el-button @click="editDialogVisible = false">取消</el-button>
          <el-button type="primary" @click="submitEdit" :loading="submitting">
            保存
          </el-button>
        </span>
      </template>
    </el-dialog>

    <!-- 安全详情对话框 -->
    <el-dialog
      v-model="detailsDialogVisible"
      title="用户安全详情"
      width="800px"
    >
      <div v-if="selectedUserDetails" class="security-details">
        <!-- 基本信息 -->
        <el-descriptions title="基本信息" :column="2" border>
          <el-descriptions-item label="用户ID">
            {{ selectedUserDetails.userId }}
          </el-descriptions-item>
          <el-descriptions-item label="用户名">
            {{ selectedUserDetails.username }}
          </el-descriptions-item>
          <el-descriptions-item label="状态">
            <el-tag :type="getStatusTagType(selectedUserDetails.status)">
              {{ getStatusText(selectedUserDetails.status) }}
            </el-tag>
          </el-descriptions-item>
          <el-descriptions-item label="风险等级">
            <el-tag :type="getRiskLevelTagType(selectedUserDetails.riskLevel)">
              {{ getRiskLevelText(selectedUserDetails.riskLevel) }}
            </el-tag>
          </el-descriptions-item>
        </el-descriptions>

        <!-- 安全统计 -->
        <el-descriptions title="安全统计" :column="3" border class="mt-4">
          <el-descriptions-item label="登录次数">
            {{ selectedUserDetails.loginAttempts || 0 }}
          </el-descriptions-item>
          <el-descriptions-item label="失败次数">
            {{ selectedUserDetails.failedAttempts || 0 }}
          </el-descriptions-item>
          <el-descriptions-item label="安全评分">
            <el-progress 
              :percentage="selectedUserDetails.securityScore" 
              :color="getScoreColor(selectedUserDetails.securityScore)"
            />
          </el-descriptions-item>
          <el-descriptions-item label="最后登录时间" :span="2">
            {{ formatDateTime(selectedUserDetails.lastLoginTime) }}
          </el-descriptions-item>
          <el-descriptions-item label="最后登录IP">
            {{ selectedUserDetails.lastLoginIp }}
          </el-descriptions-item>
        </el-descriptions>

        <!-- 安全标签 -->
        <div class="security-tags mt-4">
          <h4>安全标签</h4>
          <el-tag 
            v-for="tag in selectedUserDetails.securityTags" 
            :key="tag"
            class="mr-2"
          >
            {{ tag }}
          </el-tag>
        </div>

        <!-- 备注信息 -->
        <div v-if="selectedUserDetails.notes" class="notes mt-4">
          <h4>备注信息</h4>
          <p>{{ selectedUserDetails.notes }}</p>
        </div>
      </div>
    </el-dialog>
  </div>
</template>

<script>
import { ref, reactive, onMounted } from 'vue'
import { ElMessage, ElMessageBox } from 'element-plus'
import { Lock, Unlock, Refresh, Search } from '@element-plus/icons-vue'
import { 
  getUserSecurityProfiles, 
  updateUserSecurityProfile,
  lockUser as lockUserApi,
  unlockUser as unlockUserApi
} from '@/api/nfcRelayAdmin'
import { formatDateTime } from '@/utils/index'

export default {
  name: 'UserSecurityManagement',
  setup() {
    const loading = ref(false)
    const submitting = ref(false)
    const searchKeyword = ref('')
    const statusFilter = ref('')
    const riskLevelFilter = ref('')
    const userSecurityList = ref([])
    const selectedUsers = ref([])
    
    // 分页信息
    const pageInfo = reactive({
      page: 1,
      pageSize: 20,
      total: 0
    })

    // 编辑对话框
    const editDialogVisible = ref(false)
    const editFormRef = ref()
    const editForm = reactive({
      userId: '',
      username: '',
      riskLevel: 'low',
      securityScore: 80,
      securityTags: [],
      notes: ''
    })

    // 表单验证规则
    const editRules = reactive({
      riskLevel: [
        { required: true, message: '请选择风险等级', trigger: 'change' }
      ],
      securityScore: [
        { required: true, message: '请输入安全评分', trigger: 'blur' },
        { type: 'number', min: 0, max: 100, message: '安全评分必须在0-100之间', trigger: 'blur' }
      ]
    })

    // 详情对话框
    const detailsDialogVisible = ref(false)
    const selectedUserDetails = ref(null)

    // 获取状态标签类型
    const getStatusTagType = (status) => {
      const statusMap = {
        normal: 'success',
        locked: 'danger',
        risk: 'warning'
      }
      return statusMap[status] || 'info'
    }

    // 获取状态文本
    const getStatusText = (status) => {
      const statusMap = {
        normal: '正常',
        locked: '锁定',
        risk: '风险'
      }
      return statusMap[status] || '未知'
    }

    // 获取风险等级标签类型
    const getRiskLevelTagType = (riskLevel) => {
      const riskMap = {
        low: 'success',
        medium: 'warning',
        high: 'danger'
      }
      return riskMap[riskLevel] || 'info'
    }

    // 获取风险等级文本
    const getRiskLevelText = (riskLevel) => {
      const riskMap = {
        low: '低风险',
        medium: '中风险',
        high: '高风险'
      }
      return riskMap[riskLevel] || '未知'
    }

    // 获取评分颜色
    const getScoreColor = (score) => {
      if (score >= 80) return '#67c23a'
      if (score >= 60) return '#e6a23c'
      return '#f56c6c'
    }

    // 获取用户安全档案列表
    const fetchUserSecurityList = async () => {
      loading.value = true
      try {
        const params = {
          page: pageInfo.page,
          pageSize: pageInfo.pageSize,
          keyword: searchKeyword.value,
          status: statusFilter.value,
          riskLevel: riskLevelFilter.value
        }
        
        const response = await getUserSecurityProfiles(params)
        if (response.success) {
          userSecurityList.value = response.data.list || []
          pageInfo.total = response.data.total || 0
        }
      } catch (error) {
        console.error('获取用户安全档案失败:', error)
        ElMessage.error('获取用户安全档案失败')
      } finally {
        loading.value = false
      }
    }

    // 搜索处理
    const handleSearch = () => {
      pageInfo.page = 1
      fetchUserSecurityList()
    }

    // 分页处理
    const handleSizeChange = (size) => {
      pageInfo.pageSize = size
      pageInfo.page = 1
      fetchUserSecurityList()
    }

    const handleCurrentChange = (page) => {
      pageInfo.page = page
      fetchUserSecurityList()
    }

    // 选择变化处理
    const handleSelectionChange = (selection) => {
      selectedUsers.value = selection
    }

    // 锁定用户
    const lockUser = async (user) => {
      try {
        await ElMessageBox.confirm(
          `确定要锁定用户 ${user.username} 吗？`,
          '确认锁定',
          {
            confirmButtonText: '确定',
            cancelButtonText: '取消',
            type: 'warning'
          }
        )
        
        const response = await lockUserApi({ 
          userId: user.userId,
          reason: '管理员手动锁定'
        })
        
        if (response.success) {
          ElMessage.success('用户锁定成功')
          fetchUserSecurityList()
        }
      } catch (error) {
        if (error !== 'cancel') {
          console.error('锁定用户失败:', error)
          ElMessage.error('锁定用户失败')
        }
      }
    }

    // 解锁用户
    const unlockUser = async (user) => {
      try {
        await ElMessageBox.confirm(
          `确定要解锁用户 ${user.username} 吗？`,
          '确认解锁',
          {
            confirmButtonText: '确定',
            cancelButtonText: '取消',
            type: 'warning'
          }
        )
        
        const response = await unlockUserApi({ 
          userId: user.userId 
        })
        
        if (response.success) {
          ElMessage.success('用户解锁成功')
          fetchUserSecurityList()
        }
      } catch (error) {
        if (error !== 'cancel') {
          console.error('解锁用户失败:', error)
          ElMessage.error('解锁用户失败')
        }
      }
    }

    // 批量锁定用户
    const batchLockUsers = async () => {
      if (selectedUsers.value.length === 0) {
        ElMessage.warning('请选择要锁定的用户')
        return
      }
      
      try {
        await ElMessageBox.confirm(
          `确定要锁定选中的 ${selectedUsers.value.length} 个用户吗？`,
          '确认批量锁定',
          {
            confirmButtonText: '确定',
            cancelButtonText: '取消',
            type: 'warning'
          }
        )
        
        for (const user of selectedUsers.value) {
          await lockUserApi({ 
            userId: user.userId,
            reason: '管理员批量锁定'
          })
        }
        
        ElMessage.success('批量锁定成功')
        fetchUserSecurityList()
      } catch (error) {
        if (error !== 'cancel') {
          console.error('批量锁定失败:', error)
          ElMessage.error('批量锁定失败')
        }
      }
    }

    // 批量解锁用户
    const batchUnlockUsers = async () => {
      if (selectedUsers.value.length === 0) {
        ElMessage.warning('请选择要解锁的用户')
        return
      }
      
      try {
        await ElMessageBox.confirm(
          `确定要解锁选中的 ${selectedUsers.value.length} 个用户吗？`,
          '确认批量解锁',
          {
            confirmButtonText: '确定',
            cancelButtonText: '取消',
            type: 'warning'
          }
        )
        
        for (const user of selectedUsers.value) {
          await unlockUserApi({ userId: user.userId })
        }
        
        ElMessage.success('批量解锁成功')
        fetchUserSecurityList()
      } catch (error) {
        if (error !== 'cancel') {
          console.error('批量解锁失败:', error)
          ElMessage.error('批量解锁失败')
        }
      }
    }

    // 编辑安全档案
    const editSecurityProfile = (user) => {
      Object.assign(editForm, {
        userId: user.userId,
        username: user.username,
        riskLevel: user.riskLevel,
        securityScore: user.securityScore,
        securityTags: user.securityTags || [],
        notes: user.notes || ''
      })
      editDialogVisible.value = true
    }

    // 提交编辑
    const submitEdit = async () => {
      if (!editFormRef.value) return
      
      try {
        await editFormRef.value.validate()
        submitting.value = true
        
        const response = await updateUserSecurityProfile(editForm)
        if (response.success) {
          ElMessage.success('安全档案更新成功')
          editDialogVisible.value = false
          fetchUserSecurityList()
        }
      } catch (error) {
        console.error('更新安全档案失败:', error)
        ElMessage.error('更新安全档案失败')
      } finally {
        submitting.value = false
      }
    }

    // 查看安全详情
    const viewSecurityDetails = (user) => {
      selectedUserDetails.value = user
      detailsDialogVisible.value = true
    }

    // 刷新列表
    const refreshList = () => {
      fetchUserSecurityList()
    }

    // 页面初始化
    onMounted(() => {
      fetchUserSecurityList()
    })

    return {
      loading,
      submitting,
      searchKeyword,
      statusFilter,
      riskLevelFilter,
      userSecurityList,
      selectedUsers,
      pageInfo,
      editDialogVisible,
      editFormRef,
      editForm,
      editRules,
      detailsDialogVisible,
      selectedUserDetails,
      fetchUserSecurityList,
      handleSearch,
      handleSizeChange,
      handleCurrentChange,
      handleSelectionChange,
      lockUser,
      unlockUser,
      batchLockUsers,
      batchUnlockUsers,
      editSecurityProfile,
      submitEdit,
      viewSecurityDetails,
      refreshList,
      getStatusTagType,
      getStatusText,
      getRiskLevelTagType,
      getRiskLevelText,
      getScoreColor,
      formatDateTime,
      // Icons
      Lock,
      Unlock,
      Refresh,
      Search
    }
  }
}
</script>

<style lang="scss" scoped>
.user-security-management {
  .toolbar {
    margin-bottom: 16px;
  }

  .user-security-card {
    .card-header {
      display: flex;
      justify-content: space-between;
      align-items: center;
    }

    .pagination-wrapper {
      margin-top: 16px;
      text-align: center;
    }
  }

  .security-details {
    .mt-4 {
      margin-top: 16px;
    }

    .mr-2 {
      margin-right: 8px;
    }

    h4 {
      margin-bottom: 8px;
      color: #303133;
    }

    p {
      margin: 0;
      color: #606266;
      line-height: 1.5;
    }
  }

  .dialog-footer {
    display: flex;
    justify-content: flex-end;
    gap: 12px;
  }
}
</style> 