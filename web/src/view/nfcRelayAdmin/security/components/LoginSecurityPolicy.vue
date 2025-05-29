<template>
  <div class="login-security-policy">
    <el-row :gutter="20">
      <!-- 登录策略配置 -->
      <el-col :span="12">
        <el-card>
          <template #header>
            <div class="card-header">
              <span>登录策略配置</span>
              <el-switch 
                v-model="loginPolicyEnabled" 
                @change="saveLoginPolicy"
                inline-prompt
                active-text="启用"
                inactive-text="禁用"
              />
            </div>
          </template>
          
          <el-form 
            :model="loginPolicy" 
            label-width="140px"
            :disabled="!loginPolicyEnabled"
          >
            <el-form-item label="最大登录尝试次数">
              <el-input-number 
                v-model="loginPolicy.maxAttempts" 
                :min="1" 
                :max="20"
                controls-position="right"
                @change="saveLoginPolicy"
              />
              <div class="help-text">达到次数后将锁定账户</div>
            </el-form-item>
            
            <el-form-item label="锁定时间">
              <el-input-number 
                v-model="loginPolicy.lockoutDuration" 
                :min="5" 
                :max="1440"
                controls-position="right"
                @change="saveLoginPolicy"
              />
              <span class="unit">分钟</span>
              <div class="help-text">账户锁定的持续时间</div>
            </el-form-item>
            
            <el-form-item label="密码复杂度要求">
              <el-checkbox-group 
                v-model="loginPolicy.passwordRequirements"
                @change="saveLoginPolicy"
              >
                <el-checkbox value="minLength">最少8位</el-checkbox>
                <el-checkbox value="uppercase">包含大写字母</el-checkbox>
                <el-checkbox value="lowercase">包含小写字母</el-checkbox>
                <el-checkbox value="numbers">包含数字</el-checkbox>
                <el-checkbox value="symbols">包含特殊字符</el-checkbox>
              </el-checkbox-group>
            </el-form-item>
            
            <el-form-item label="密码有效期">
              <el-input-number 
                v-model="loginPolicy.passwordExpireDays" 
                :min="30" 
                :max="365"
                controls-position="right"
                @change="saveLoginPolicy"
              />
              <span class="unit">天</span>
              <div class="help-text">0表示永不过期</div>
            </el-form-item>
            
            <el-form-item>
              <el-button 
                type="primary" 
                @click="saveLoginPolicy"
                :loading="saving"
              >
                保存登录策略
              </el-button>
            </el-form-item>
          </el-form>
        </el-card>
      </el-col>

      <!-- 双因子认证 -->
      <el-col :span="12">
        <el-card>
          <template #header>
            <div class="card-header">
              <span>双因子认证</span>
              <el-switch 
                v-model="twoFactorEnabled" 
                @change="saveTwoFactorPolicy"
                inline-prompt
                active-text="启用"
                inactive-text="禁用"
              />
            </div>
          </template>
          
          <el-form 
            :model="twoFactorPolicy" 
            label-width="140px"
            :disabled="!twoFactorEnabled"
          >
            <el-form-item label="强制启用角色">
              <el-checkbox-group 
                v-model="twoFactorPolicy.requiredRoles"
                @change="saveTwoFactorPolicy"
              >
                <el-checkbox value="admin">管理员</el-checkbox>
                <el-checkbox value="super_admin">超级管理员</el-checkbox>
                <el-checkbox value="security_admin">安全管理员</el-checkbox>
              </el-checkbox-group>
            </el-form-item>
            
            <el-form-item label="认证方式">
              <el-checkbox-group 
                v-model="twoFactorPolicy.methods"
                @change="saveTwoFactorPolicy"
              >
                <el-checkbox value="sms">短信验证</el-checkbox>
                <el-checkbox value="email">邮箱验证</el-checkbox>
                <el-checkbox value="totp">TOTP认证器</el-checkbox>
                <el-checkbox value="backup_codes">备用代码</el-checkbox>
              </el-checkbox-group>
            </el-form-item>
            
            <el-form-item label="验证码有效期">
              <el-input-number 
                v-model="twoFactorPolicy.codeExpireMinutes" 
                :min="1" 
                :max="30"
                controls-position="right"
                @change="saveTwoFactorPolicy"
              />
              <span class="unit">分钟</span>
            </el-form-item>
            
            <el-form-item label="记住设备">
              <el-input-number 
                v-model="twoFactorPolicy.rememberDeviceDays" 
                :min="0" 
                :max="90"
                controls-position="right"
                @change="saveTwoFactorPolicy"
              />
              <span class="unit">天</span>
              <div class="help-text">在受信设备上的免验证期限</div>
            </el-form-item>
            
            <el-form-item>
              <el-button 
                type="primary" 
                @click="saveTwoFactorPolicy"
                :loading="saving"
              >
                保存2FA策略
              </el-button>
            </el-form-item>
          </el-form>
        </el-card>
      </el-col>
    </el-row>

    <!-- 登录监控策略 -->
    <el-row class="mt-4">
      <el-col :span="24">
        <el-card>
          <template #header>
            <div class="card-header">
              <span>登录监控策略</span>
              <el-switch 
                v-model="monitoringEnabled" 
                @change="saveMonitoringPolicy"
                inline-prompt
                active-text="启用"
                inactive-text="禁用"
              />
            </div>
          </template>
          
          <el-form 
            :model="monitoringPolicy" 
            label-width="140px"
            :disabled="!monitoringEnabled"
          >
            <el-row :gutter="20">
              <el-col :span="8">
                <el-form-item label="异常登录检测">
                  <el-checkbox 
                    v-model="monitoringPolicy.detectUnusualLogin"
                    @change="saveMonitoringPolicy"
                  >
                    检测异常登录位置
                  </el-checkbox>
                  
                  <el-checkbox 
                    v-model="monitoringPolicy.detectUnusualTime"
                    @change="saveMonitoringPolicy"
                  >
                    检测异常登录时间
                  </el-checkbox>
                  
                  <el-checkbox 
                    v-model="monitoringPolicy.detectMultipleDevices"
                    @change="saveMonitoringPolicy"
                  >
                    检测多设备同时登录
                  </el-checkbox>
                </el-form-item>
              </el-col>
              
              <el-col :span="8">
                <el-form-item label="IP地址监控">
                  <el-checkbox 
                    v-model="monitoringPolicy.trackIpChanges"
                    @change="saveMonitoringPolicy"
                  >
                    跟踪IP地址变化
                  </el-checkbox>
                  
                  <el-checkbox 
                    v-model="monitoringPolicy.blockVpnProxy"
                    @change="saveMonitoringPolicy"
                  >
                    阻止VPN/代理登录
                  </el-checkbox>
                  
                  <el-checkbox 
                    v-model="monitoringPolicy.geoBlocking"
                    @change="saveMonitoringPolicy"
                  >
                    地理位置限制
                  </el-checkbox>
                </el-form-item>
              </el-col>
              
              <el-col :span="8">
                <el-form-item label="响应动作">
                  <el-checkbox-group 
                    v-model="monitoringPolicy.responseActions"
                    @change="saveMonitoringPolicy"
                  >
                    <el-checkbox value="log">记录日志</el-checkbox>
                    <el-checkbox value="alert">发送警报</el-checkbox>
                    <el-checkbox value="challenge">要求额外验证</el-checkbox>
                    <el-checkbox value="block">阻止登录</el-checkbox>
                  </el-checkbox-group>
                </el-form-item>
              </el-col>
            </el-row>
            
            <el-form-item>
              <el-button 
                type="primary" 
                @click="saveMonitoringPolicy"
                :loading="saving"
              >
                保存监控策略
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
  name: 'LoginSecurityPolicy',
  setup() {
    const saving = ref(false)
    
    // 登录策略
    const loginPolicyEnabled = ref(true)
    const loginPolicy = reactive({
      maxAttempts: 5,
      lockoutDuration: 30,
      passwordRequirements: ['minLength', 'uppercase', 'lowercase', 'numbers'],
      passwordExpireDays: 90
    })
    
    // 双因子认证策略
    const twoFactorEnabled = ref(true)
    const twoFactorPolicy = reactive({
      requiredRoles: ['admin', 'super_admin'],
      methods: ['email', 'totp'],
      codeExpireMinutes: 5,
      rememberDeviceDays: 30
    })
    
    // 监控策略
    const monitoringEnabled = ref(true)
    const monitoringPolicy = reactive({
      detectUnusualLogin: true,
      detectUnusualTime: true,
      detectMultipleDevices: false,
      trackIpChanges: true,
      blockVpnProxy: false,
      geoBlocking: false,
      responseActions: ['log', 'alert']
    })
    
    // 保存登录策略
    const saveLoginPolicy = async () => {
      saving.value = true
      try {
        // 这里应该调用API保存登录策略
        await new Promise(resolve => setTimeout(resolve, 500))
        ElMessage.success('登录策略保存成功')
      } catch (error) {
        console.error('保存登录策略失败:', error)
        ElMessage.error('保存登录策略失败')
      } finally {
        saving.value = false
      }
    }
    
    // 保存双因子认证策略
    const saveTwoFactorPolicy = async () => {
      saving.value = true
      try {
        // 这里应该调用API保存双因子认证策略
        await new Promise(resolve => setTimeout(resolve, 500))
        ElMessage.success('双因子认证策略保存成功')
      } catch (error) {
        console.error('保存双因子认证策略失败:', error)
        ElMessage.error('保存双因子认证策略失败')
      } finally {
        saving.value = false
      }
    }
    
    // 保存监控策略
    const saveMonitoringPolicy = async () => {
      saving.value = true
      try {
        // 这里应该调用API保存监控策略
        await new Promise(resolve => setTimeout(resolve, 500))
        ElMessage.success('登录监控策略保存成功')
      } catch (error) {
        console.error('保存监控策略失败:', error)
        ElMessage.error('保存监控策略失败')
      } finally {
        saving.value = false
      }
    }
    
    return {
      saving,
      loginPolicyEnabled,
      loginPolicy,
      twoFactorEnabled,
      twoFactorPolicy,
      monitoringEnabled,
      monitoringPolicy,
      saveLoginPolicy,
      saveTwoFactorPolicy,
      saveMonitoringPolicy
    }
  }
}
</script>

<style lang="scss" scoped>
.login-security-policy {
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
}
</style> 