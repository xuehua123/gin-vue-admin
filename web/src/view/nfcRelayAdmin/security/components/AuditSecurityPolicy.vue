<template>
  <div class="audit-security-policy">
    <el-row :gutter="20">
      <!-- 审计日志级别 -->
      <el-col :span="12">
        <el-card>
          <template #header>
            <div class="card-header">
              <span>审计日志级别</span>
              <el-switch 
                v-model="auditLoggingEnabled" 
                @change="saveAuditLoggingPolicy"
                inline-prompt
                active-text="启用"
                inactive-text="禁用"
              />
            </div>
          </template>
          
          <el-form 
            :model="auditLoggingPolicy" 
            label-width="140px"
            :disabled="!auditLoggingEnabled"
          >
            <el-form-item label="日志级别">
              <el-checkbox-group 
                v-model="auditLoggingPolicy.logLevels"
                @change="saveAuditLoggingPolicy"
              >
                <el-checkbox value="debug">调试 (DEBUG)</el-checkbox>
                <el-checkbox value="info">信息 (INFO)</el-checkbox>
                <el-checkbox value="warn">警告 (WARN)</el-checkbox>
                <el-checkbox value="error">错误 (ERROR)</el-checkbox>
                <el-checkbox value="critical">严重 (CRITICAL)</el-checkbox>
              </el-checkbox-group>
            </el-form-item>
            
            <el-form-item label="事件类型">
              <el-checkbox-group 
                v-model="auditLoggingPolicy.eventTypes"
                @change="saveAuditLoggingPolicy"
              >
                <el-checkbox value="login">登录事件</el-checkbox>
                <el-checkbox value="logout">登出事件</el-checkbox>
                <el-checkbox value="session_start">会话开始</el-checkbox>
                <el-checkbox value="session_end">会话结束</el-checkbox>
                <el-checkbox value="apdu_exchange">APDU交换</el-checkbox>
                <el-checkbox value="config_change">配置变更</el-checkbox>
                <el-checkbox value="security_alert">安全警报</el-checkbox>
                <el-checkbox value="admin_operation">管理操作</el-checkbox>
              </el-checkbox-group>
            </el-form-item>
            
            <el-form-item label="详细级别">
              <el-radio-group 
                v-model="auditLoggingPolicy.detailLevel"
                @change="saveAuditLoggingPolicy"
              >
                <el-radio value="minimal">最小</el-radio>
                <el-radio value="standard">标准</el-radio>
                <el-radio value="detailed">详细</el-radio>
                <el-radio value="full">完整</el-radio>
              </el-radio-group>
              <div class="help-text">控制每个事件记录的信息详细程度</div>
            </el-form-item>
            
            <el-form-item>
              <el-button 
                type="primary" 
                @click="saveAuditLoggingPolicy"
                :loading="saving"
              >
                保存日志策略
              </el-button>
            </el-form-item>
          </el-form>
        </el-card>
      </el-col>

      <!-- 数据保留策略 -->
      <el-col :span="12">
        <el-card>
          <template #header>
            <div class="card-header">
              <span>数据保留策略</span>
              <el-switch 
                v-model="dataRetentionEnabled" 
                @change="saveDataRetentionPolicy"
                inline-prompt
                active-text="启用"
                inactive-text="禁用"
              />
            </div>
          </template>
          
          <el-form 
            :model="dataRetentionPolicy" 
            label-width="140px"
            :disabled="!dataRetentionEnabled"
          >
            <el-form-item label="审计日志保留期">
              <el-input-number 
                v-model="dataRetentionPolicy.auditLogRetentionDays" 
                :min="1" 
                :max="3650"
                controls-position="right"
                @change="saveDataRetentionPolicy"
              />
              <span class="unit">天</span>
              <div class="help-text">审计日志的保留天数</div>
            </el-form-item>
            
            <el-form-item label="会话数据保留期">
              <el-input-number 
                v-model="dataRetentionPolicy.sessionDataRetentionDays" 
                :min="1" 
                :max="365"
                controls-position="right"
                @change="saveDataRetentionPolicy"
              />
              <span class="unit">天</span>
              <div class="help-text">会话相关数据的保留天数</div>
            </el-form-item>
            
            <el-form-item label="安全事件保留期">
              <el-input-number 
                v-model="dataRetentionPolicy.securityEventRetentionDays" 
                :min="30" 
                :max="3650"
                controls-position="right"
                @change="saveDataRetentionPolicy"
              />
              <span class="unit">天</span>
              <div class="help-text">安全相关事件的保留天数</div>
            </el-form-item>
            
            <el-form-item label="自动清理">
              <el-checkbox 
                v-model="dataRetentionPolicy.autoCleanup"
                @change="saveDataRetentionPolicy"
              >
                启用过期数据自动清理
              </el-checkbox>
            </el-form-item>
            
            <el-form-item 
              v-if="dataRetentionPolicy.autoCleanup"
              label="清理时间"
            >
              <el-time-select
                v-model="dataRetentionPolicy.cleanupTime"
                start="00:00"
                step="01:00"
                end="23:00"
                placeholder="选择清理时间"
                @change="saveDataRetentionPolicy"
              />
            </el-form-item>
            
            <el-form-item>
              <el-button 
                type="primary" 
                @click="saveDataRetentionPolicy"
                :loading="saving"
              >
                保存保留策略
              </el-button>
            </el-form-item>
          </el-form>
        </el-card>
      </el-col>
    </el-row>

    <!-- 合规性设置 -->
    <el-row class="mt-4">
      <el-col :span="24">
        <el-card>
          <template #header>
            <div class="card-header">
              <span>合规性设置</span>
              <el-switch 
                v-model="complianceEnabled" 
                @change="saveCompliancePolicy"
                inline-prompt
                active-text="启用"
                inactive-text="禁用"
              />
            </div>
          </template>
          
          <el-form 
            :model="compliancePolicy" 
            label-width="140px"
            :disabled="!complianceEnabled"
          >
            <el-row :gutter="20">
              <el-col :span="8">
                <el-form-item label="合规标准">
                  <el-checkbox-group 
                    v-model="compliancePolicy.standards"
                    @change="saveCompliancePolicy"
                  >
                    <el-checkbox value="iso27001">ISO 27001</el-checkbox>
                    <el-checkbox value="gdpr">GDPR</el-checkbox>
                    <el-checkbox value="pci_dss">PCI DSS</el-checkbox>
                    <el-checkbox value="sox">SOX</el-checkbox>
                    <el-checkbox value="hipaa">HIPAA</el-checkbox>
                  </el-checkbox-group>
                </el-form-item>
              </el-col>
              
              <el-col :span="8">
                <el-form-item label="数据分类">
                  <el-checkbox-group 
                    v-model="compliancePolicy.dataClassifications"
                    @change="saveCompliancePolicy"
                  >
                    <el-checkbox value="public">公开</el-checkbox>
                    <el-checkbox value="internal">内部</el-checkbox>
                    <el-checkbox value="confidential">机密</el-checkbox>
                    <el-checkbox value="restricted">限制</el-checkbox>
                  </el-checkbox-group>
                  <div class="help-text">需要特殊处理的数据分类</div>
                </el-form-item>
              </el-col>
              
              <el-col :span="8">
                <el-form-item label="完整性验证">
                  <el-checkbox 
                    v-model="compliancePolicy.integrityCheck"
                    @change="saveCompliancePolicy"
                  >
                    启用日志完整性验证
                  </el-checkbox>
                  
                  <el-checkbox 
                    v-model="compliancePolicy.digitalSignature"
                    @change="saveCompliancePolicy"
                  >
                    启用数字签名
                  </el-checkbox>
                  
                  <el-checkbox 
                    v-model="compliancePolicy.tamperDetection"
                    @change="saveCompliancePolicy"
                  >
                    启用篡改检测
                  </el-checkbox>
                </el-form-item>
              </el-col>
            </el-row>
            
            <el-form-item>
              <el-button 
                type="primary" 
                @click="saveCompliancePolicy"
                :loading="saving"
              >
                保存合规策略
              </el-button>
            </el-form-item>
          </el-form>
        </el-card>
      </el-col>
    </el-row>

    <!-- 报告和导出 -->
    <el-row class="mt-4">
      <el-col :span="24">
        <el-card>
          <template #header>
            <div class="card-header">
              <span>报告和导出</span>
              <el-switch 
                v-model="reportingEnabled" 
                @change="saveReportingPolicy"
                inline-prompt
                active-text="启用"
                inactive-text="禁用"
              />
            </div>
          </template>
          
          <el-form 
            :model="reportingPolicy" 
            label-width="140px"
            :disabled="!reportingEnabled"
          >
            <el-row :gutter="20">
              <el-col :span="12">
                <el-form-item label="自动报告">
                  <el-checkbox-group 
                    v-model="reportingPolicy.autoReports"
                    @change="saveReportingPolicy"
                  >
                    <el-checkbox value="daily">每日摘要</el-checkbox>
                    <el-checkbox value="weekly">周度报告</el-checkbox>
                    <el-checkbox value="monthly">月度报告</el-checkbox>
                    <el-checkbox value="incident">事件报告</el-checkbox>
                  </el-checkbox-group>
                </el-form-item>
                
                <el-form-item label="报告格式">
                  <el-checkbox-group 
                    v-model="reportingPolicy.exportFormats"
                    @change="saveReportingPolicy"
                  >
                    <el-checkbox value="pdf">PDF</el-checkbox>
                    <el-checkbox value="excel">Excel</el-checkbox>
                    <el-checkbox value="csv">CSV</el-checkbox>
                    <el-checkbox value="json">JSON</el-checkbox>
                  </el-checkbox-group>
                </el-form-item>
              </el-col>
              
              <el-col :span="12">
                <el-form-item label="报告接收人">
                  <el-tag
                    v-for="email in reportingPolicy.recipients"
                    :key="email"
                    closable
                    @close="removeRecipient(email)"
                    class="mr-2 mb-2"
                  >
                    {{ email }}
                  </el-tag>
                  <el-input
                    v-if="showAddEmailInput"
                    ref="addEmailInputRef"
                    v-model="newRecipientEmail"
                    size="small"
                    style="width: 200px"
                    placeholder="输入邮箱地址"
                    @keyup.enter="addRecipient"
                    @blur="addRecipient"
                  />
                  <el-button 
                    v-else 
                    size="small" 
                    @click="showAddEmailInput = true"
                  >
                    + 添加接收人
                  </el-button>
                </el-form-item>
                
                <el-form-item label="加密导出">
                  <el-checkbox 
                    v-model="reportingPolicy.encryptExport"
                    @change="saveReportingPolicy"
                  >
                    导出文件加密保护
                  </el-checkbox>
                </el-form-item>
                
                <el-form-item label="访问控制">
                  <el-checkbox 
                    v-model="reportingPolicy.accessControl"
                    @change="saveReportingPolicy"
                  >
                    基于角色的报告访问控制
                  </el-checkbox>
                </el-form-item>
              </el-col>
            </el-row>
            
            <el-form-item>
              <el-button 
                type="primary" 
                @click="saveReportingPolicy"
                :loading="saving"
              >
                保存报告策略
              </el-button>
              <el-button 
                type="success" 
                @click="generateTestReport"
                :loading="generatingReport"
              >
                生成测试报告
              </el-button>
            </el-form-item>
          </el-form>
        </el-card>
      </el-col>
    </el-row>
  </div>
</template>

<script>
import { ref, reactive } from 'vue'
import { ElMessage } from 'element-plus'

export default {
  name: 'AuditSecurityPolicy',
  setup() {
    const saving = ref(false)
    const generatingReport = ref(false)
    
    // 审计日志策略
    const auditLoggingEnabled = ref(true)
    const auditLoggingPolicy = reactive({
      logLevels: ['info', 'warn', 'error', 'critical'],
      eventTypes: ['login', 'logout', 'session_start', 'session_end', 'security_alert', 'admin_operation'],
      detailLevel: 'standard'
    })
    
    // 数据保留策略
    const dataRetentionEnabled = ref(true)
    const dataRetentionPolicy = reactive({
      auditLogRetentionDays: 365,
      sessionDataRetentionDays: 90,
      securityEventRetentionDays: 1095,
      autoCleanup: true,
      cleanupTime: '02:00'
    })
    
    // 合规性策略
    const complianceEnabled = ref(true)
    const compliancePolicy = reactive({
      standards: ['iso27001'],
      dataClassifications: ['internal', 'confidential'],
      integrityCheck: true,
      digitalSignature: false,
      tamperDetection: true
    })
    
    // 报告策略
    const reportingEnabled = ref(true)
    const reportingPolicy = reactive({
      autoReports: ['daily', 'weekly'],
      exportFormats: ['pdf', 'excel'],
      recipients: ['admin@company.com'],
      encryptExport: true,
      accessControl: true
    })
    
    const showAddEmailInput = ref(false)
    const newRecipientEmail = ref('')
    const addEmailInputRef = ref()
    
    // 邮箱验证
    const validateEmail = (email) => {
      const emailRegex = /^[^\s@]+@[^\s@]+\.[^\s@]+$/
      return emailRegex.test(email)
    }
    
    // 添加接收人
    const addRecipient = () => {
      if (newRecipientEmail.value && validateEmail(newRecipientEmail.value)) {
        if (!reportingPolicy.recipients.includes(newRecipientEmail.value)) {
          reportingPolicy.recipients.push(newRecipientEmail.value)
          saveReportingPolicy()
        }
        newRecipientEmail.value = ''
      } else if (newRecipientEmail.value) {
        ElMessage.error('请输入有效的邮箱地址')
      }
      showAddEmailInput.value = false
    }
    
    // 移除接收人
    const removeRecipient = (email) => {
      const index = reportingPolicy.recipients.indexOf(email)
      if (index > -1) {
        reportingPolicy.recipients.splice(index, 1)
        saveReportingPolicy()
      }
    }
    
    // 保存审计日志策略
    const saveAuditLoggingPolicy = async () => {
      saving.value = true
      try {
        // 这里应该调用API保存审计日志策略
        await new Promise(resolve => setTimeout(resolve, 500))
        ElMessage.success('审计日志策略保存成功')
      } catch (error) {
        console.error('保存审计日志策略失败:', error)
        ElMessage.error('保存审计日志策略失败')
      } finally {
        saving.value = false
      }
    }
    
    // 保存数据保留策略
    const saveDataRetentionPolicy = async () => {
      saving.value = true
      try {
        // 这里应该调用API保存数据保留策略
        await new Promise(resolve => setTimeout(resolve, 500))
        ElMessage.success('数据保留策略保存成功')
      } catch (error) {
        console.error('保存数据保留策略失败:', error)
        ElMessage.error('保存数据保留策略失败')
      } finally {
        saving.value = false
      }
    }
    
    // 保存合规性策略
    const saveCompliancePolicy = async () => {
      saving.value = true
      try {
        // 这里应该调用API保存合规性策略
        await new Promise(resolve => setTimeout(resolve, 500))
        ElMessage.success('合规性策略保存成功')
      } catch (error) {
        console.error('保存合规性策略失败:', error)
        ElMessage.error('保存合规性策略失败')
      } finally {
        saving.value = false
      }
    }
    
    // 保存报告策略
    const saveReportingPolicy = async () => {
      saving.value = true
      try {
        // 这里应该调用API保存报告策略
        await new Promise(resolve => setTimeout(resolve, 500))
        ElMessage.success('报告策略保存成功')
      } catch (error) {
        console.error('保存报告策略失败:', error)
        ElMessage.error('保存报告策略失败')
      } finally {
        saving.value = false
      }
    }
    
    // 生成测试报告
    const generateTestReport = async () => {
      generatingReport.value = true
      try {
        // 这里应该调用API生成测试报告
        await new Promise(resolve => setTimeout(resolve, 2000))
        ElMessage.success('测试报告生成成功，已发送到指定邮箱')
      } catch (error) {
        console.error('生成测试报告失败:', error)
        ElMessage.error('生成测试报告失败')
      } finally {
        generatingReport.value = false
      }
    }
    
    return {
      saving,
      generatingReport,
      auditLoggingEnabled,
      auditLoggingPolicy,
      dataRetentionEnabled,
      dataRetentionPolicy,
      complianceEnabled,
      compliancePolicy,
      reportingEnabled,
      reportingPolicy,
      showAddEmailInput,
      newRecipientEmail,
      addEmailInputRef,
      addRecipient,
      removeRecipient,
      saveAuditLoggingPolicy,
      saveDataRetentionPolicy,
      saveCompliancePolicy,
      saveReportingPolicy,
      generateTestReport
    }
  }
}
</script>

<style lang="scss" scoped>
.audit-security-policy {
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

  .unit {
    margin-left: 8px;
    color: #909399;
    font-size: 14px;
  }

  .mt-4 {
    margin-top: 16px;
  }

  .mr-2 {
    margin-right: 8px;
  }

  .mb-2 {
    margin-bottom: 8px;
  }
}
</style> 