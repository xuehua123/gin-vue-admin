<template>
  <div class="client-ban-management">
    <!-- 操作工具栏 -->
    <div class="toolbar">
      <el-row justify="space-between">
        <el-col :span="16">
          <el-space>
            <el-button type="primary" @click="showAddBanDialog">
              <el-icon><Plus /></el-icon>
              新增封禁
            </el-button>
            <el-button 
              type="warning" 
              :disabled="selectedBans.length === 0"
              @click="batchUnban"
            >
              <el-icon><Unlock /></el-icon>
              批量解封
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
            placeholder="搜索客户端ID或IP地址"
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

    <!-- 封禁列表 -->
    <el-card class="ban-list-card">
      <template #header>
        <span>客户端封禁列表</span>
      </template>
      
      <el-table
        ref="banTableRef"
        v-loading="loading"
        :data="banList"
        @selection-change="handleSelectionChange"
        row-key="id"
        height="400"
      >
        <el-table-column type="selection" width="55" />
        <el-table-column prop="clientId" label="客户端ID" width="200" />
        <el-table-column prop="ipAddress" label="IP地址" width="150" />
        <el-table-column prop="reason" label="封禁原因" show-overflow-tooltip />
        <el-table-column prop="bannedAt" label="封禁时间" width="180">
          <template #default="{ row }">
            {{ formatDateTime(row.bannedAt) }}
          </template>
        </el-table-column>
        <el-table-column prop="expiresAt" label="到期时间" width="180">
          <template #default="{ row }">
            <span v-if="row.expiresAt">{{ formatDateTime(row.expiresAt) }}</span>
            <el-tag v-else type="danger">永久</el-tag>
          </template>
        </el-table-column>
        <el-table-column prop="status" label="状态" width="100">
          <template #default="{ row }">
            <el-tag :type="row.status === 'active' ? 'danger' : 'success'">
              {{ row.status === 'active' ? '生效中' : '已解封' }}
            </el-tag>
          </template>
        </el-table-column>
        <el-table-column label="操作" width="180" fixed="right">
          <template #default="{ row }">
            <el-space>
              <el-button 
                v-if="row.status === 'active'"
                type="warning" 
                size="small"
                @click="unbanClient(row)"
              >
                解封
              </el-button>
              <el-button 
                type="primary" 
                size="small"
                @click="viewBanDetails(row)"
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

    <!-- 新增封禁对话框 -->
    <el-dialog
      v-model="addBanDialogVisible"
      title="新增客户端封禁"
      width="600px"
      :close-on-click-modal="false"
    >
      <el-form
        ref="addBanFormRef"
        :model="addBanForm"
        :rules="addBanRules"
        label-width="100px"
      >
        <el-form-item label="客户端ID" prop="clientId">
          <el-input 
            v-model="addBanForm.clientId" 
            placeholder="请输入客户端ID"
            clearable
          />
        </el-form-item>
        <el-form-item label="IP地址" prop="ipAddress">
          <el-input 
            v-model="addBanForm.ipAddress" 
            placeholder="请输入IP地址(可选)"
            clearable
          />
        </el-form-item>
        <el-form-item label="封禁原因" prop="reason">
          <el-input
            v-model="addBanForm.reason"
            type="textarea"
            :rows="3"
            placeholder="请输入封禁原因"
          />
        </el-form-item>
        <el-form-item label="封禁类型" prop="banType">
          <el-radio-group v-model="addBanForm.banType">
            <el-radio value="temporary">临时封禁</el-radio>
            <el-radio value="permanent">永久封禁</el-radio>
          </el-radio-group>
        </el-form-item>
        <el-form-item 
          v-if="addBanForm.banType === 'temporary'"
          label="到期时间" 
          prop="expiresAt"
        >
          <el-date-picker
            v-model="addBanForm.expiresAt"
            type="datetime"
            placeholder="选择到期时间"
            format="YYYY-MM-DD HH:mm:ss"
            value-format="YYYY-MM-DD HH:mm:ss"
          />
        </el-form-item>
      </el-form>
      
      <template #footer>
        <span class="dialog-footer">
          <el-button @click="addBanDialogVisible = false">取消</el-button>
          <el-button type="primary" @click="submitAddBan" :loading="submitting">
            确定
          </el-button>
        </span>
      </template>
    </el-dialog>

    <!-- 封禁详情对话框 -->
    <el-dialog
      v-model="banDetailsDialogVisible"
      title="封禁详情"
      width="600px"
    >
      <el-descriptions v-if="selectedBanDetails" :column="2" border>
        <el-descriptions-item label="客户端ID">
          {{ selectedBanDetails.clientId }}
        </el-descriptions-item>
        <el-descriptions-item label="IP地址">
          {{ selectedBanDetails.ipAddress || '未指定' }}
        </el-descriptions-item>
        <el-descriptions-item label="封禁原因" :span="2">
          {{ selectedBanDetails.reason }}
        </el-descriptions-item>
        <el-descriptions-item label="封禁时间">
          {{ formatDateTime(selectedBanDetails.bannedAt) }}
        </el-descriptions-item>
        <el-descriptions-item label="到期时间">
          <span v-if="selectedBanDetails.expiresAt">
            {{ formatDateTime(selectedBanDetails.expiresAt) }}
          </span>
          <el-tag v-else type="danger">永久</el-tag>
        </el-descriptions-item>
        <el-descriptions-item label="封禁状态">
          <el-tag :type="selectedBanDetails.status === 'active' ? 'danger' : 'success'">
            {{ selectedBanDetails.status === 'active' ? '生效中' : '已解封' }}
          </el-tag>
        </el-descriptions-item>
        <el-descriptions-item label="操作人员">
          {{ selectedBanDetails.operatorName || '系统' }}
        </el-descriptions-item>
      </el-descriptions>
    </el-dialog>
  </div>
</template>

<script>
import { ref, reactive, onMounted } from 'vue'
import { ElMessage, ElMessageBox } from 'element-plus'
import { Plus, Unlock, Refresh, Search } from '@element-plus/icons-vue'
import { 
  getClientBans, 
  banClient, 
  unbanClient 
} from '@/api/nfcRelayAdmin'
import { formatDateTime } from '@/utils/index'

export default {
  name: 'ClientBanManagement',
  setup() {
    const loading = ref(false)
    const submitting = ref(false)
    const searchKeyword = ref('')
    const banList = ref([])
    const selectedBans = ref([])
    
    // 分页信息
    const pageInfo = reactive({
      page: 1,
      pageSize: 20,
      total: 0
    })

    // 新增封禁对话框
    const addBanDialogVisible = ref(false)
    const addBanFormRef = ref()
    const addBanForm = reactive({
      clientId: '',
      ipAddress: '',
      reason: '',
      banType: 'temporary',
      expiresAt: ''
    })

    // 表单验证规则
    const addBanRules = reactive({
      clientId: [
        { required: true, message: '请输入客户端ID', trigger: 'blur' }
      ],
      reason: [
        { required: true, message: '请输入封禁原因', trigger: 'blur' }
      ],
      expiresAt: [
        { 
          required: true, 
          message: '请选择到期时间', 
          trigger: 'change',
          validator: (rule, value, callback) => {
            if (addBanForm.banType === 'temporary' && !value) {
              callback(new Error('临时封禁必须设置到期时间'))
            } else {
              callback()
            }
          }
        }
      ]
    })

    // 封禁详情对话框
    const banDetailsDialogVisible = ref(false)
    const selectedBanDetails = ref(null)

    // 获取封禁列表
    const fetchBanList = async () => {
      loading.value = true
      try {
        const params = {
          page: pageInfo.page,
          pageSize: pageInfo.pageSize,
          keyword: searchKeyword.value
        }
        
        const response = await getClientBans(params)
        if (response.success) {
          banList.value = response.data.list || []
          pageInfo.total = response.data.total || 0
        }
      } catch (error) {
        console.error('获取封禁列表失败:', error)
        ElMessage.error('获取封禁列表失败')
      } finally {
        loading.value = false
      }
    }

    // 搜索处理
    const handleSearch = () => {
      pageInfo.page = 1
      fetchBanList()
    }

    // 分页处理
    const handleSizeChange = (size) => {
      pageInfo.pageSize = size
      pageInfo.page = 1
      fetchBanList()
    }

    const handleCurrentChange = (page) => {
      pageInfo.page = page
      fetchBanList()
    }

    // 选择变化处理
    const handleSelectionChange = (selection) => {
      selectedBans.value = selection
    }

    // 显示新增封禁对话框
    const showAddBanDialog = () => {
      // 重置表单
      Object.assign(addBanForm, {
        clientId: '',
        ipAddress: '',
        reason: '',
        banType: 'temporary',
        expiresAt: ''
      })
      addBanDialogVisible.value = true
    }

    // 提交新增封禁
    const submitAddBan = async () => {
      if (!addBanFormRef.value) return
      
      try {
        await addBanFormRef.value.validate()
        submitting.value = true
        
        const banData = {
          clientId: addBanForm.clientId,
          ipAddress: addBanForm.ipAddress,
          reason: addBanForm.reason,
          banType: addBanForm.banType,
          expiresAt: addBanForm.banType === 'permanent' ? null : addBanForm.expiresAt
        }
        
        const response = await banClient(banData)
        if (response.success) {
          ElMessage.success('客户端封禁成功')
          addBanDialogVisible.value = false
          fetchBanList()
        }
      } catch (error) {
        console.error('封禁客户端失败:', error)
        ElMessage.error('封禁客户端失败')
      } finally {
        submitting.value = false
      }
    }

    // 解封客户端
    const unbanClientItem = async (banItem) => {
      try {
        await ElMessageBox.confirm(
          `确定要解封客户端 ${banItem.clientId} 吗？`,
          '确认解封',
          {
            confirmButtonText: '确定',
            cancelButtonText: '取消',
            type: 'warning'
          }
        )
        
        const response = await unbanClient({ 
          clientId: banItem.clientId,
          banId: banItem.id 
        })
        
        if (response.success) {
          ElMessage.success('客户端解封成功')
          fetchBanList()
        }
      } catch (error) {
        if (error !== 'cancel') {
          console.error('解封客户端失败:', error)
          ElMessage.error('解封客户端失败')
        }
      }
    }

    // 批量解封
    const batchUnban = async () => {
      if (selectedBans.value.length === 0) {
        ElMessage.warning('请选择要解封的客户端')
        return
      }
      
      try {
        await ElMessageBox.confirm(
          `确定要解封选中的 ${selectedBans.value.length} 个客户端吗？`,
          '确认批量解封',
          {
            confirmButtonText: '确定',
            cancelButtonText: '取消',
            type: 'warning'
          }
        )
        
        for (const ban of selectedBans.value) {
          await unbanClient({ 
            clientId: ban.clientId,
            banId: ban.id 
          })
        }
        
        ElMessage.success('批量解封成功')
        fetchBanList()
      } catch (error) {
        if (error !== 'cancel') {
          console.error('批量解封失败:', error)
          ElMessage.error('批量解封失败')
        }
      }
    }

    // 查看封禁详情
    const viewBanDetails = (banItem) => {
      selectedBanDetails.value = banItem
      banDetailsDialogVisible.value = true
    }

    // 刷新列表
    const refreshList = () => {
      fetchBanList()
    }

    // 页面初始化
    onMounted(() => {
      fetchBanList()
    })

    return {
      loading,
      submitting,
      searchKeyword,
      banList,
      selectedBans,
      pageInfo,
      addBanDialogVisible,
      addBanFormRef,
      addBanForm,
      addBanRules,
      banDetailsDialogVisible,
      selectedBanDetails,
      fetchBanList,
      handleSearch,
      handleSizeChange,
      handleCurrentChange,
      handleSelectionChange,
      showAddBanDialog,
      submitAddBan,
      unbanClient: unbanClientItem,
      batchUnban,
      viewBanDetails,
      refreshList,
      formatDateTime,
      // Icons
      Plus,
      Unlock,
      Refresh,
      Search
    }
  }
}
</script>

<style lang="scss" scoped>
.client-ban-management {
  .toolbar {
    margin-bottom: 16px;
  }

  .ban-list-card {
    .pagination-wrapper {
      margin-top: 16px;
      text-align: center;
    }
  }

  .dialog-footer {
    display: flex;
    justify-content: flex-end;
    gap: 12px;
  }
}
</style> 