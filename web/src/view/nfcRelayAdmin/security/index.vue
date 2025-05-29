<template>
  <div class="nfc-security-management">
    <!-- 页面标题 -->
    <div class="page-header">
      <h1>安全管理</h1>
      <p>管理客户端封禁、用户安全档案和系统安全策略</p>
    </div>

    <!-- 安全摘要卡片 -->
    <div class="security-summary">
      <el-row :gutter="20">
        <el-col :span="6">
          <el-card class="summary-card">
            <div class="card-content">
              <div class="icon">
                <el-icon color="#f56c6c"><Warning /></el-icon>
              </div>
              <div class="info">
                <div class="value">{{ securitySummary.bannedClients || 0 }}</div>
                <div class="label">封禁客户端</div>
              </div>
            </div>
          </el-card>
        </el-col>
        <el-col :span="6">
          <el-card class="summary-card">
            <div class="card-content">
              <div class="icon">
                <el-icon color="#e6a23c"><Lock /></el-icon>
              </div>
              <div class="info">
                <div class="value">{{ securitySummary.lockedUsers || 0 }}</div>
                <div class="label">锁定用户</div>
              </div>
            </div>
          </el-card>
        </el-col>
        <el-col :span="6">
          <el-card class="summary-card">
            <div class="card-content">
              <div class="icon">
                <el-icon color="#67c23a"><Shield /></el-icon>
              </div>
              <div class="info">
                <div class="value">{{ securitySummary.totalSecurityEvents || 0 }}</div>
                <div class="label">安全事件</div>
              </div>
            </div>
          </el-card>
        </el-col>
        <el-col :span="6">
          <el-card class="summary-card">
            <div class="card-content">
              <div class="icon">
                <el-icon color="#409eff"><Monitor /></el-icon>
              </div>
              <div class="info">
                <div class="value">{{ securitySummary.activeMonitoringRules || 0 }}</div>
                <div class="label">监控规则</div>
              </div>
            </div>
          </el-card>
        </el-col>
      </el-row>
    </div>

    <!-- 功能标签页 -->
    <el-tabs v-model="activeTab" type="card" class="security-tabs">
      <!-- 客户端封禁管理 -->
      <el-tab-pane label="客户端封禁" name="client-bans">
        <ClientBanManagement ref="clientBanRef" />
      </el-tab-pane>

      <!-- 用户安全档案 -->
      <el-tab-pane label="用户安全档案" name="user-security">
        <UserSecurityManagement ref="userSecurityRef" />
      </el-tab-pane>

      <!-- 安全策略 -->
      <el-tab-pane label="安全策略" name="security-policies">
        <SecurityPolicyManagement ref="securityPolicyRef" />
      </el-tab-pane>

      <!-- 系统清理 -->
      <el-tab-pane label="系统清理" name="system-cleanup">
        <SystemCleanup ref="systemCleanupRef" />
      </el-tab-pane>
    </el-tabs>
  </div>
</template>

<script>
import { ref, reactive, onMounted } from 'vue'
import { ElMessage } from 'element-plus'
import { Warning, Lock, Shield, Monitor } from '@element-plus/icons-vue'
import { getSecuritySummary } from '@/api/nfcRelayAdmin'
import ClientBanManagement from './components/ClientBanManagement.vue'
import UserSecurityManagement from './components/UserSecurityManagement.vue'
import SecurityPolicyManagement from './components/SecurityPolicyManagement.vue'
import SystemCleanup from './components/SystemCleanup.vue'

export default {
  name: 'NfcSecurityManagement',
  components: {
    ClientBanManagement,
    UserSecurityManagement,
    SecurityPolicyManagement,
    SystemCleanup
  },
  setup() {
    const activeTab = ref('client-bans')
    const securitySummary = reactive({
      bannedClients: 0,
      lockedUsers: 0,
      totalSecurityEvents: 0,
      activeMonitoringRules: 0
    })

    // 引用子组件
    const clientBanRef = ref()
    const userSecurityRef = ref()
    const securityPolicyRef = ref()
    const systemCleanupRef = ref()

    // 获取安全摘要
    const fetchSecuritySummary = async () => {
      try {
        const response = await getSecuritySummary()
        if (response.success) {
          Object.assign(securitySummary, response.data)
        }
      } catch (error) {
        console.error('获取安全摘要失败:', error)
        ElMessage.error('获取安全摘要失败')
      }
    }

    // 页面初始化
    onMounted(() => {
      fetchSecuritySummary()
    })

    return {
      activeTab,
      securitySummary,
      clientBanRef,
      userSecurityRef,
      securityPolicyRef,
      systemCleanupRef,
      // Icons
      Warning,
      Lock,
      Shield,
      Monitor
    }
  }
}
</script>

<style lang="scss" scoped>
.nfc-security-management {
  padding: 20px;

  .page-header {
    margin-bottom: 20px;

    h1 {
      margin: 0 0 8px 0;
      font-size: 24px;
      font-weight: 600;
      color: #303133;
    }

    p {
      margin: 0;
      color: #606266;
      font-size: 14px;
    }
  }

  .security-summary {
    margin-bottom: 24px;

    .summary-card {
      .card-content {
        display: flex;
        align-items: center;

        .icon {
          margin-right: 16px;
          font-size: 24px;
        }

        .info {
          flex: 1;

          .value {
            font-size: 24px;
            font-weight: 600;
            color: #303133;
            margin-bottom: 4px;
          }

          .label {
            font-size: 12px;
            color: #909399;
          }
        }
      }
    }
  }

  .security-tabs {
    :deep(.el-tabs__content) {
      padding: 20px 0;
    }

    :deep(.el-tab-pane) {
      min-height: 500px;
    }
  }
}
</style> 