<template>
  <div class="system-cleanup">
    <!-- 清理概览 -->
    <el-row :gutter="20" class="cleanup-overview">
      <el-col :span="6">
        <el-card class="overview-card">
          <div class="card-content">
            <div class="icon">
              <el-icon color="#e6a23c"><FolderOpened /></el-icon>
            </div>
            <div class="info">
              <div class="value">{{ cleanupStats.totalLogs || 0 }}</div>
              <div class="label">日志条目</div>
            </div>
          </div>
        </el-card>
      </el-col>
      <el-col :span="6">
        <el-card class="overview-card">
          <div class="card-content">
            <div class="icon">
              <el-icon color="#f56c6c"><Delete /></el-icon>
            </div>
            <div class="info">
              <div class="value">{{ cleanupStats.expiredItems || 0 }}</div>
              <div class="label">过期数据</div>
            </div>
          </div>
        </el-card>
      </el-col>
      <el-col :span="6">
        <el-card class="overview-card">
          <div class="card-content">
            <div class="icon">
              <el-icon color="#67c23a"><DocumentCopy /></el-icon>
            </div>
            <div class="info">
              <div class="value">{{ cleanupStats.sessionRecords || 0 }}</div>
              <div class="label">会话记录</div>
            </div>
          </div>
        </el-card>
      </el-col>
      <el-col :span="6">
        <el-card class="overview-card">
          <div class="card-content">
            <div class="icon">
              <el-icon color="#909399"><Histogram /></el-icon>
            </div>
            <div class="info">
              <div class="value">{{ cleanupStats.diskUsage || '0MB' }}</div>
              <div class="label">磁盘占用</div>
            </div>
          </div>
        </el-card>
      </el-col>
    </el-row>

    <!-- 清理操作面板 -->
    <el-row :gutter="20">
      <!-- 自动清理配置 -->
      <el-col :span="12">
        <el-card class="cleanup-config-card">
          <template #header>
            <div class="card-header">
              <span>自动清理配置</span>
              <el-switch 
                v-model="autoCleanupEnabled" 
                @change="toggleAutoCleanup"
                inline-prompt
                active-text="启用"
                inactive-text="禁用"
              />
            </div>
          </template>
          
          <el-form 
            :model="cleanupConfig" 
            label-width="120px"
            :disabled="!autoCleanupEnabled"
          >
            <el-form-item label="清理周期">
              <el-select v-model="cleanupConfig.schedule" style="width: 100%">
                <el-option label="每天" value="daily" />
                <el-option label="每周" value="weekly" />
                <el-option label="每月" value="monthly" />
              </el-select>
            </el-form-item>
            
            <el-form-item label="日志保留期">
              <el-input-number 
                v-model="cleanupConfig.logRetentionDays" 
                :min="1" 
                :max="365"
                controls-position="right"
              />
              <span class="unit">天</span>
            </el-form-item>
            
            <el-form-item label="会话记录保留期">
              <el-input-number 
                v-model="cleanupConfig.sessionRetentionDays" 
                :min="1" 
                :max="180"
                controls-position="right"
              />
              <span class="unit">天</span>
            </el-form-item>
            
            <el-form-item label="临时文件清理">
              <el-checkbox v-model="cleanupConfig.cleanTempFiles">
                清理临时文件和缓存
              </el-checkbox>
            </el-form-item>
            
            <el-form-item>
              <el-button 
                type="primary" 
                @click="saveCleanupConfig"
                :loading="saving"
              >
                保存配置
              </el-button>
            </el-form-item>
          </el-form>
        </el-card>
      </el-col>

      <!-- 手动清理操作 -->
      <el-col :span="12">
        <el-card class="manual-cleanup-card">
          <template #header>
            <span>手动清理操作</span>
          </template>
          
          <div class="cleanup-actions">
            <el-space direction="vertical" size="large" style="width: 100%">
              <!-- 清理审计日志 -->
              <div class="cleanup-item">
                <div class="item-header">
                  <h4>清理审计日志</h4>
                  <p>清理指定时间之前的审计日志记录</p>
                </div>
                <div class="item-content">
                  <el-row :gutter="10">
                    <el-col :span="16">
                      <el-date-picker
                        v-model="cleanupForms.auditLogs.beforeDate"
                        type="date"
                        placeholder="选择清理截止日期"
                        format="YYYY-MM-DD"
                        value-format="YYYY-MM-DD"
                        style="width: 100%"
                      />
                    </el-col>
                    <el-col :span="8">
                      <el-button 
                        type="danger" 
                        @click="cleanupAuditLogs"
                        :loading="cleanupForms.auditLogs.loading"
                        style="width: 100%"
                      >
                        清理日志
                      </el-button>
                    </el-col>
                  </el-row>
                </div>
              </div>

              <!-- 清理会话记录 -->
              <div class="cleanup-item">
                <div class="item-header">
                  <h4>清理会话记录</h4>
                  <p>清理已完成的会话记录和相关文件</p>
                </div>
                <div class="item-content">
                  <el-row :gutter="10">
                    <el-col :span="16">
                      <el-select 
                        v-model="cleanupForms.sessions.status" 
                        placeholder="选择会话状态"
                        style="width: 100%"
                      >
                        <el-option label="已完成" value="completed" />
                        <el-option label="已失败" value="failed" />
                        <el-option label="已终止" value="terminated" />
                        <el-option label="全部" value="all" />
                      </el-select>
                    </el-col>
                    <el-col :span="8">
                      <el-button 
                        type="danger" 
                        @click="cleanupSessions"
                        :loading="cleanupForms.sessions.loading"
                        style="width: 100%"
                      >
                        清理记录
                      </el-button>
                    </el-col>
                  </el-row>
                </div>
              </div>

              <!-- 清理过期封禁 -->
              <div class="cleanup-item">
                <div class="item-header">
                  <h4>清理过期封禁</h4>
                  <p>移除已过期的客户端封禁记录</p>
                </div>
                <div class="item-content">
                  <el-button 
                    type="warning" 
                    @click="cleanupExpiredBans"
                    :loading="cleanupForms.bans.loading"
                    style="width: 100%"
                  >
                    清理过期封禁
                  </el-button>
                </div>
              </div>

              <!-- 系统缓存清理 -->
              <div class="cleanup-item">
                <div class="item-header">
                  <h4>系统缓存清理</h4>
                  <p>清理系统缓存和临时文件</p>
                </div>
                <div class="item-content">
                  <el-button 
                    type="info" 
                    @click="cleanupCache"
                    :loading="cleanupForms.cache.loading"
                    style="width: 100%"
                  >
                    清理缓存
                  </el-button>
                </div>
              </div>
            </el-space>
          </div>
        </el-card>
      </el-col>
    </el-row>

    <!-- 清理历史记录 -->
    <el-card class="cleanup-history-card">
      <template #header>
        <span>清理历史记录</span>
      </template>
      
      <el-table 
        v-loading="historyLoading"
        :data="cleanupHistory" 
        height="300"
      >
        <el-table-column prop="operation" label="清理操作" width="150" />
        <el-table-column prop="type" label="数据类型" width="120" />
        <el-table-column prop="itemsCount" label="清理数量" width="100" />
        <el-table-column prop="dataSize" label="释放空间" width="120" />
        <el-table-column prop="executedAt" label="执行时间" width="180">
          <template #default="{ row }">
            {{ formatDateTime(row.executedAt) }}
          </template>
        </el-table-column>
        <el-table-column prop="status" label="状态" width="100">
          <template #default="{ row }">
            <el-tag :type="row.status === 'success' ? 'success' : 'danger'">
              {{ row.status === 'success' ? '成功' : '失败' }}
            </el-tag>
          </template>
        </el-table-column>
        <el-table-column prop="operator" label="操作人员" />
      </el-table>
    </el-card>
  </div>
</template>

<script>
import { ref, reactive, onMounted } from 'vue'
import { ElMessage, ElMessageBox } from 'element-plus'
import { 
  FolderOpened, 
  Delete, 
  DocumentCopy, 
  Histogram 
} from '@element-plus/icons-vue'
import { 
  securityCleanup,
  cleanupAuditLogsDb
} from '@/api/nfcRelayAdmin'
import { formatDateTime } from '@/utils/index'

export default {
  name: 'SystemCleanup',
  setup() {
    const saving = ref(false)
    const historyLoading = ref(false)
    const autoCleanupEnabled = ref(true)
    
    // 清理统计数据
    const cleanupStats = reactive({
      totalLogs: 15420,
      expiredItems: 238,
      sessionRecords: 1205,
      diskUsage: '2.1GB'
    })

    // 自动清理配置
    const cleanupConfig = reactive({
      schedule: 'daily',
      logRetentionDays: 30,
      sessionRetentionDays: 7,
      cleanTempFiles: true
    })

    // 手动清理表单
    const cleanupForms = reactive({
      auditLogs: {
        beforeDate: '',
        loading: false
      },
      sessions: {
        status: 'completed',
        loading: false
      },
      bans: {
        loading: false
      },
      cache: {
        loading: false
      }
    })

    // 清理历史记录
    const cleanupHistory = ref([
      {
        operation: '自动清理',
        type: '审计日志',
        itemsCount: 1580,
        dataSize: '125MB',
        executedAt: '2024-01-15 02:00:00',
        status: 'success',
        operator: '系统'
      },
      {
        operation: '手动清理',
        type: '会话记录',
        itemsCount: 320,
        dataSize: '85MB',
        executedAt: '2024-01-14 14:30:00',
        status: 'success',
        operator: '管理员'
      }
    ])

    // 切换自动清理
    const toggleAutoCleanup = async (enabled) => {
      try {
        // 这里应该调用API保存自动清理开关状态
        ElMessage.success(`自动清理已${enabled ? '启用' : '禁用'}`)
      } catch (error) {
        console.error('切换自动清理失败:', error)
        ElMessage.error('操作失败')
        autoCleanupEnabled.value = !enabled
      }
    }

    // 保存清理配置
    const saveCleanupConfig = async () => {
      saving.value = true
      try {
        // 这里应该调用API保存清理配置
        await new Promise(resolve => setTimeout(resolve, 1000))
        ElMessage.success('配置保存成功')
      } catch (error) {
        console.error('保存配置失败:', error)
        ElMessage.error('保存配置失败')
      } finally {
        saving.value = false
      }
    }

    // 清理审计日志
    const cleanupAuditLogs = async () => {
      if (!cleanupForms.auditLogs.beforeDate) {
        ElMessage.warning('请选择清理截止日期')
        return
      }

      try {
        await ElMessageBox.confirm(
          `确定要清理 ${cleanupForms.auditLogs.beforeDate} 之前的所有审计日志吗？`,
          '确认清理',
          {
            confirmButtonText: '确定清理',
            cancelButtonText: '取消',
            type: 'warning'
          }
        )

        cleanupForms.auditLogs.loading = true
        
        const response = await cleanupAuditLogsDb({
          beforeDate: cleanupForms.auditLogs.beforeDate
        })
        
        if (response.success) {
          ElMessage.success('审计日志清理成功')
          // 刷新统计数据和历史记录
          fetchCleanupStats()
          fetchCleanupHistory()
        }
      } catch (error) {
        if (error !== 'cancel') {
          console.error('清理审计日志失败:', error)
          ElMessage.error('清理审计日志失败')
        }
      } finally {
        cleanupForms.auditLogs.loading = false
      }
    }

    // 清理会话记录
    const cleanupSessions = async () => {
      try {
        await ElMessageBox.confirm(
          `确定要清理状态为"${cleanupForms.sessions.status}"的会话记录吗？`,
          '确认清理',
          {
            confirmButtonText: '确定清理',
            cancelButtonText: '取消',
            type: 'warning'
          }
        )

        cleanupForms.sessions.loading = true
        
        const response = await securityCleanup({
          type: 'sessions',
          status: cleanupForms.sessions.status
        })
        
        if (response.success) {
          ElMessage.success('会话记录清理成功')
          fetchCleanupStats()
          fetchCleanupHistory()
        }
      } catch (error) {
        if (error !== 'cancel') {
          console.error('清理会话记录失败:', error)
          ElMessage.error('清理会话记录失败')
        }
      } finally {
        cleanupForms.sessions.loading = false
      }
    }

    // 清理过期封禁
    const cleanupExpiredBans = async () => {
      try {
        await ElMessageBox.confirm(
          '确定要清理所有过期的客户端封禁记录吗？',
          '确认清理',
          {
            confirmButtonText: '确定清理',
            cancelButtonText: '取消',
            type: 'warning'
          }
        )

        cleanupForms.bans.loading = true
        
        const response = await securityCleanup({
          type: 'expired_bans'
        })
        
        if (response.success) {
          ElMessage.success('过期封禁记录清理成功')
          fetchCleanupStats()
          fetchCleanupHistory()
        }
      } catch (error) {
        if (error !== 'cancel') {
          console.error('清理过期封禁失败:', error)
          ElMessage.error('清理过期封禁失败')
        }
      } finally {
        cleanupForms.bans.loading = false
      }
    }

    // 清理系统缓存
    const cleanupCache = async () => {
      try {
        await ElMessageBox.confirm(
          '确定要清理系统缓存和临时文件吗？',
          '确认清理',
          {
            confirmButtonText: '确定清理',
            cancelButtonText: '取消',
            type: 'warning'
          }
        )

        cleanupForms.cache.loading = true
        
        const response = await securityCleanup({
          type: 'cache'
        })
        
        if (response.success) {
          ElMessage.success('系统缓存清理成功')
          fetchCleanupStats()
          fetchCleanupHistory()
        }
      } catch (error) {
        if (error !== 'cancel') {
          console.error('清理系统缓存失败:', error)
          ElMessage.error('清理系统缓存失败')
        }
      } finally {
        cleanupForms.cache.loading = false
      }
    }

    // 获取清理统计
    const fetchCleanupStats = async () => {
      // 这里应该调用API获取实际的清理统计数据
    }

    // 获取清理历史
    const fetchCleanupHistory = async () => {
      historyLoading.value = true
      try {
        // 这里应该调用API获取清理历史记录
        await new Promise(resolve => setTimeout(resolve, 500))
      } catch (error) {
        console.error('获取清理历史失败:', error)
      } finally {
        historyLoading.value = false
      }
    }

    onMounted(() => {
      fetchCleanupStats()
      fetchCleanupHistory()
    })

    return {
      saving,
      historyLoading,
      autoCleanupEnabled,
      cleanupStats,
      cleanupConfig,
      cleanupForms,
      cleanupHistory,
      toggleAutoCleanup,
      saveCleanupConfig,
      cleanupAuditLogs,
      cleanupSessions,
      cleanupExpiredBans,
      cleanupCache,
      formatDateTime,
      // Icons
      FolderOpened,
      Delete,
      DocumentCopy,
      Histogram
    }
  }
}
</script>

<style lang="scss" scoped>
.system-cleanup {
  .cleanup-overview {
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

  .cleanup-config-card,
  .manual-cleanup-card {
    margin-bottom: 24px;

    .card-header {
      display: flex;
      justify-content: space-between;
      align-items: center;
    }

    .unit {
      margin-left: 8px;
      color: #909399;
      font-size: 14px;
    }
  }

  .cleanup-actions {
    .cleanup-item {
      padding: 16px;
      border: 1px solid #e4e7ed;
      border-radius: 4px;

      .item-header {
        margin-bottom: 12px;

        h4 {
          margin: 0 0 4px 0;
          font-size: 16px;
          color: #303133;
        }

        p {
          margin: 0;
          font-size: 12px;
          color: #909399;
        }
      }

      .item-content {
        // 样式已在内联中定义
      }
    }
  }

  .cleanup-history-card {
    // 基础卡片样式
  }
}
</style> 