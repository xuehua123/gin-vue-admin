<template>
  <div class="security-policy-management">
    <!-- 安全策略概览 -->
    <el-row :gutter="20" class="policy-overview">
      <el-col :span="8">
        <el-card class="overview-card">
          <div class="card-content">
            <div class="icon">
              <el-icon color="#67c23a"><Shield /></el-icon>
            </div>
            <div class="info">
              <div class="value">{{ policyStats.activePolicies || 0 }}</div>
              <div class="label">激活策略</div>
            </div>
          </div>
        </el-card>
      </el-col>
      <el-col :span="8">
        <el-card class="overview-card">
          <div class="card-content">
            <div class="icon">
              <el-icon color="#e6a23c"><Warning /></el-icon>
            </div>
            <div class="info">
              <div class="value">{{ policyStats.violationCount || 0 }}</div>
              <div class="label">策略违规</div>
            </div>
          </div>
        </el-card>
      </el-col>
      <el-col :span="8">
        <el-card class="overview-card">
          <div class="card-content">
            <div class="icon">
              <el-icon color="#409eff"><Monitor /></el-icon>
            </div>
            <div class="info">
              <div class="value">{{ policyStats.monitoringRules || 0 }}</div>
              <div class="label">监控规则</div>
            </div>
          </div>
        </el-card>
      </el-col>
    </el-row>

    <!-- 功能标签页 -->
    <el-tabs v-model="activeTab" type="card" class="policy-tabs">
      <!-- 登录安全策略 -->
      <el-tab-pane label="登录安全" name="login-security">
        <LoginSecurityPolicy />
      </el-tab-pane>

      <!-- 访问控制策略 -->
      <el-tab-pane label="访问控制" name="access-control">
        <AccessControlPolicy />
      </el-tab-pane>

      <!-- 会话安全策略 -->
      <el-tab-pane label="会话安全" name="session-security">
        <SessionSecurityPolicy />
      </el-tab-pane>

      <!-- 审计策略 -->
      <el-tab-pane label="审计策略" name="audit-policy">
        <AuditSecurityPolicy />
      </el-tab-pane>
    </el-tabs>
  </div>
</template>

<script>
import { ref, reactive, onMounted } from 'vue'
import { ElMessage } from 'element-plus'
import { Shield, Warning, Monitor } from '@element-plus/icons-vue'
import LoginSecurityPolicy from './LoginSecurityPolicy.vue'
import AccessControlPolicy from './AccessControlPolicy.vue'
import SessionSecurityPolicy from './SessionSecurityPolicy.vue'
import AuditSecurityPolicy from './AuditSecurityPolicy.vue'

export default {
  name: 'SecurityPolicyManagement',
  components: {
    LoginSecurityPolicy,
    AccessControlPolicy,
    SessionSecurityPolicy,
    AuditSecurityPolicy
  },
  setup() {
    const activeTab = ref('login-security')
    const policyStats = reactive({
      activePolicies: 12,
      violationCount: 3,
      monitoringRules: 8
    })

    // 模拟获取策略统计
    const fetchPolicyStats = async () => {
      // 这里应该调用实际的API获取策略统计数据
      // const response = await getPolicyStats()
      // Object.assign(policyStats, response.data)
    }

    onMounted(() => {
      fetchPolicyStats()
    })

    return {
      activeTab,
      policyStats,
      // Icons
      Shield,
      Warning,
      Monitor
    }
  }
}
</script>

<style lang="scss" scoped>
.security-policy-management {
  .policy-overview {
    margin-bottom: 24px;

    .overview-card {
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

  .policy-tabs {
    :deep(.el-tabs__content) {
      padding: 20px 0;
    }

    :deep(.el-tab-pane) {
      min-height: 400px;
    }
  }
}
</style> 