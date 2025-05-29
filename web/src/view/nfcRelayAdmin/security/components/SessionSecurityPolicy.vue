<template>
  <div class="session-security-policy">
    <el-row :gutter="20">
      <!-- 会话超时设置 -->
      <el-col :span="12">
        <el-card>
          <template #header>
            <div class="card-header">
              <span>会话超时设置</span>
              <el-switch 
                v-model="sessionTimeoutEnabled" 
                @change="saveSessionTimeoutPolicy"
                inline-prompt
                active-text="启用"
                inactive-text="禁用"
              />
            </div>
          </template>
          
          <el-form 
            :model="sessionTimeoutPolicy" 
            label-width="140px"
            :disabled="!sessionTimeoutEnabled"
          >
            <el-form-item label="空闲超时时间">
              <el-input-number 
                v-model="sessionTimeoutPolicy.idleTimeout" 
                :min="1" 
                :max="1440"
                controls-position="right"
                @change="saveSessionTimeoutPolicy"
              />
              <span class="unit">分钟</span>
              <div class="help-text">会话空闲多长时间后自动断开</div>
            </el-form-item>
            
            <el-form-item label="最大会话时长">
              <el-input-number 
                v-model="sessionTimeoutPolicy.maxDuration" 
                :min="30" 
                :max="1440"
                controls-position="right"
                @change="saveSessionTimeoutPolicy"
              />
              <span class="unit">分钟</span>
              <div class="help-text">单个会话的最大持续时间</div>
            </el-form-item>
            
            <el-form-item label="超时警告提前">
              <el-input-number 
                v-model="sessionTimeoutPolicy.warningTime" 
                :min="1" 
                :max="10"
                controls-position="right"
                @change="saveSessionTimeoutPolicy"
              />
              <span class="unit">分钟</span>
              <div class="help-text">超时前多长时间发出警告</div>
            </el-form-item>
            
            <el-form-item label="自动续期">
              <el-checkbox 
                v-model="sessionTimeoutPolicy.autoRenewal"
                @change="saveSessionTimeoutPolicy"
              >
                检测到活动时自动续期会话
              </el-checkbox>
            </el-form-item>
            
            <el-form-item>
              <el-button 
                type="primary" 
                @click="saveSessionTimeoutPolicy"
                :loading="saving"
              >
                保存超时策略
              </el-button>
            </el-form-item>
          </el-form>
        </el-card>
      </el-col>

      <!-- 并发会话限制 -->
      <el-col :span="12">
        <el-card>
          <template #header>
            <div class="card-header">
              <span>并发会话限制</span>
              <el-switch 
                v-model="concurrentSessionEnabled" 
                @change="saveConcurrentSessionPolicy"
                inline-prompt
                active-text="启用"
                inactive-text="禁用"
              />
            </div>
          </template>
          
          <el-form 
            :model="concurrentSessionPolicy" 
            label-width="140px"
            :disabled="!concurrentSessionEnabled"
          >
            <el-form-item label="每用户最大会话数">
              <el-input-number 
                v-model="concurrentSessionPolicy.maxSessionsPerUser" 
                :min="1" 
                :max="20"
                controls-position="right"
                @change="saveConcurrentSessionPolicy"
              />
              <div class="help-text">单个用户最多可同时建立的会话数</div>
            </el-form-item>
            
            <el-form-item label="系统最大会话数">
              <el-input-number 
                v-model="concurrentSessionPolicy.maxSystemSessions" 
                :min="10" 
                :max="1000"
                controls-position="right"
                @change="saveConcurrentSessionPolicy"
              />
              <div class="help-text">系统全局最大并发会话数</div>
            </el-form-item>
            
            <el-form-item label="会话排队">
              <el-checkbox 
                v-model="concurrentSessionPolicy.enableQueue"
                @change="saveConcurrentSessionPolicy"
              >
                达到限制时将新会话加入队列
              </el-checkbox>
            </el-form-item>
            
            <el-form-item 
              v-if="concurrentSessionPolicy.enableQueue"
              label="队列最大长度"
            >
              <el-input-number 
                v-model="concurrentSessionPolicy.queueSize" 
                :min="1" 
                :max="100"
                controls-position="right"
                @change="saveConcurrentSessionPolicy"
              />
            </el-form-item>
            
            <el-form-item label="冲突处理策略">
              <el-select 
                v-model="concurrentSessionPolicy.conflictStrategy" 
                style="width: 100%"
                @change="saveConcurrentSessionPolicy"
              >
                <el-option label="拒绝新连接" value="reject_new" />
                <el-option label="踢掉最旧连接" value="kick_oldest" />
                <el-option label="踢掉最空闲连接" value="kick_idle" />
              </el-select>
            </el-form-item>
            
            <el-form-item>
              <el-button 
                type="primary" 
                @click="saveConcurrentSessionPolicy"
                :loading="saving"
              >
                保存并发策略
              </el-button>
            </el-form-item>
          </el-form>
        </el-card>
      </el-col>
    </el-row>

    <!-- 会话安全监控 -->
    <el-row class="mt-4">
      <el-col :span="24">
        <el-card>
          <template #header>
            <div class="card-header">
              <span>会话安全监控</span>
              <el-switch 
                v-model="sessionMonitoringEnabled" 
                @change="saveSessionMonitoringPolicy"
                inline-prompt
                active-text="启用"
                inactive-text="禁用"
              />
            </div>
          </template>
          
          <el-form 
            :model="sessionMonitoringPolicy" 
            label-width="140px"
            :disabled="!sessionMonitoringEnabled"
          >
            <el-row :gutter="20">
              <el-col :span="8">
                <el-form-item label="异常检测">
                  <el-checkbox 
                    v-model="sessionMonitoringPolicy.anomalyDetection"
                    @change="saveSessionMonitoringPolicy"
                  >
                    启用会话异常行为检测
                  </el-checkbox>
                  
                  <el-checkbox 
                    v-model="sessionMonitoringPolicy.frequencyCheck"
                    @change="saveSessionMonitoringPolicy"
                  >
                    检测APDU频率异常
                  </el-checkbox>
                  
                  <el-checkbox 
                    v-model="sessionMonitoringPolicy.patternCheck"
                    @change="saveSessionMonitoringPolicy"
                  >
                    检测异常通信模式
                  </el-checkbox>
                </el-form-item>
              </el-col>
              
              <el-col :span="8">
                <el-form-item label="阈值设置">
                  <div class="threshold-setting">
                    <label>APDU频率阈值:</label>
                    <el-input-number 
                      v-model="sessionMonitoringPolicy.apduFrequencyThreshold" 
                      :min="1" 
                      :max="1000"
                      size="small"
                      @change="saveSessionMonitoringPolicy"
                    />
                    <span class="unit">次/分钟</span>
                  </div>
                  
                  <div class="threshold-setting">
                    <label>数据量阈值:</label>
                    <el-input-number 
                      v-model="sessionMonitoringPolicy.dataVolumeThreshold" 
                      :min="1" 
                      :max="10000"
                      size="small"
                      @change="saveSessionMonitoringPolicy"
                    />
                    <span class="unit">KB/分钟</span>
                  </div>
                  
                  <div class="threshold-setting">
                    <label>错误率阈值:</label>
                    <el-input-number 
                      v-model="sessionMonitoringPolicy.errorRateThreshold" 
                      :min="0" 
                      :max="100"
                      :precision="1"
                      size="small"
                      @change="saveSessionMonitoringPolicy"
                    />
                    <span class="unit">%</span>
                  </div>
                </el-form-item>
              </el-col>
              
              <el-col :span="8">
                <el-form-item label="响应动作">
                  <el-checkbox-group 
                    v-model="sessionMonitoringPolicy.responseActions"
                    @change="saveSessionMonitoringPolicy"
                  >
                    <el-checkbox value="log">记录日志</el-checkbox>
                    <el-checkbox value="alert">发送警报</el-checkbox>
                    <el-checkbox value="throttle">限制速率</el-checkbox>
                    <el-checkbox value="terminate">终止会话</el-checkbox>
                    <el-checkbox value="ban">封禁用户</el-checkbox>
                  </el-checkbox-group>
                </el-form-item>
              </el-col>
            </el-row>
            
            <el-form-item>
              <el-button 
                type="primary" 
                @click="saveSessionMonitoringPolicy"
                :loading="saving"
              >
                保存监控策略
              </el-button>
            </el-form-item>
          </el-form>
        </el-card>
      </el-col>
    </el-row>

    <!-- 会话记录策略 -->
    <el-row class="mt-4">
      <el-col :span="24">
        <el-card>
          <template #header>
            <div class="card-header">
              <span>会话记录策略</span>
              <el-switch 
                v-model="sessionRecordingEnabled" 
                @change="saveSessionRecordingPolicy"
                inline-prompt
                active-text="启用"
                inactive-text="禁用"
              />
            </div>
          </template>
          
          <el-form 
            :model="sessionRecordingPolicy" 
            label-width="140px"
            :disabled="!sessionRecordingEnabled"
          >
            <el-row :gutter="20">
              <el-col :span="12">
                <el-form-item label="记录类型">
                  <el-checkbox-group 
                    v-model="sessionRecordingPolicy.recordTypes"
                    @change="saveSessionRecordingPolicy"
                  >
                    <el-checkbox value="metadata">会话元数据</el-checkbox>
                    <el-checkbox value="apdu">APDU数据</el-checkbox>
                    <el-checkbox value="timing">时序信息</el-checkbox>
                    <el-checkbox value="errors">错误信息</el-checkbox>
                  </el-checkbox-group>
                </el-form-item>
                
                <el-form-item label="记录条件">
                  <el-checkbox-group 
                    v-model="sessionRecordingPolicy.recordConditions"
                    @change="saveSessionRecordingPolicy"
                  >
                    <el-checkbox value="all">全部会话</el-checkbox>
                    <el-checkbox value="long_duration">长时间会话</el-checkbox>
                    <el-checkbox value="high_volume">高频会话</el-checkbox>
                    <el-checkbox value="error_prone">错误频发会话</el-checkbox>
                    <el-checkbox value="suspicious">可疑会话</el-checkbox>
                  </el-checkbox-group>
                </el-form-item>
              </el-col>
              
              <el-col :span="12">
                <el-form-item label="存储设置">
                  <div class="storage-setting">
                    <label>保留期限:</label>
                    <el-input-number 
                      v-model="sessionRecordingPolicy.retentionDays" 
                      :min="1" 
                      :max="365"
                      @change="saveSessionRecordingPolicy"
                    />
                    <span class="unit">天</span>
                  </div>
                  
                  <div class="storage-setting">
                    <label>压缩存储:</label>
                    <el-switch 
                      v-model="sessionRecordingPolicy.compressed"
                      @change="saveSessionRecordingPolicy"
                    />
                  </div>
                  
                  <div class="storage-setting">
                    <label>加密存储:</label>
                    <el-switch 
                      v-model="sessionRecordingPolicy.encrypted"
                      @change="saveSessionRecordingPolicy"
                    />
                  </div>
                </el-form-item>
              </el-col>
            </el-row>
            
            <el-form-item>
              <el-button 
                type="primary" 
                @click="saveSessionRecordingPolicy"
                :loading="saving"
              >
                保存记录策略
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
  name: 'SessionSecurityPolicy',
  setup() {
    const saving = ref(false)
    
    // 会话超时设置
    const sessionTimeoutEnabled = ref(true)
    const sessionTimeoutPolicy = reactive({
      idleTimeout: 30,
      maxDuration: 240,
      warningTime: 5,
      autoRenewal: true
    })
    
    // 并发会话限制
    const concurrentSessionEnabled = ref(true)
    const concurrentSessionPolicy = reactive({
      maxSessionsPerUser: 3,
      maxSystemSessions: 100,
      enableQueue: true,
      queueSize: 20,
      conflictStrategy: 'kick_oldest'
    })
    
    // 会话安全监控
    const sessionMonitoringEnabled = ref(true)
    const sessionMonitoringPolicy = reactive({
      anomalyDetection: true,
      frequencyCheck: true,
      patternCheck: true,
      apduFrequencyThreshold: 100,
      dataVolumeThreshold: 1000,
      errorRateThreshold: 10.0,
      responseActions: ['log', 'alert']
    })
    
    // 会话记录策略
    const sessionRecordingEnabled = ref(true)
    const sessionRecordingPolicy = reactive({
      recordTypes: ['metadata', 'timing', 'errors'],
      recordConditions: ['long_duration', 'error_prone', 'suspicious'],
      retentionDays: 30,
      compressed: true,
      encrypted: true
    })
    
    // 保存会话超时策略
    const saveSessionTimeoutPolicy = async () => {
      saving.value = true
      try {
        // 这里应该调用API保存会话超时策略
        await new Promise(resolve => setTimeout(resolve, 500))
        ElMessage.success('会话超时策略保存成功')
      } catch (error) {
        console.error('保存会话超时策略失败:', error)
        ElMessage.error('保存会话超时策略失败')
      } finally {
        saving.value = false
      }
    }
    
    // 保存并发会话策略
    const saveConcurrentSessionPolicy = async () => {
      saving.value = true
      try {
        // 这里应该调用API保存并发会话策略
        await new Promise(resolve => setTimeout(resolve, 500))
        ElMessage.success('并发会话策略保存成功')
      } catch (error) {
        console.error('保存并发会话策略失败:', error)
        ElMessage.error('保存并发会话策略失败')
      } finally {
        saving.value = false
      }
    }
    
    // 保存会话监控策略
    const saveSessionMonitoringPolicy = async () => {
      saving.value = true
      try {
        // 这里应该调用API保存会话监控策略
        await new Promise(resolve => setTimeout(resolve, 500))
        ElMessage.success('会话监控策略保存成功')
      } catch (error) {
        console.error('保存会话监控策略失败:', error)
        ElMessage.error('保存会话监控策略失败')
      } finally {
        saving.value = false
      }
    }
    
    // 保存会话记录策略
    const saveSessionRecordingPolicy = async () => {
      saving.value = true
      try {
        // 这里应该调用API保存会话记录策略
        await new Promise(resolve => setTimeout(resolve, 500))
        ElMessage.success('会话记录策略保存成功')
      } catch (error) {
        console.error('保存会话记录策略失败:', error)
        ElMessage.error('保存会话记录策略失败')
      } finally {
        saving.value = false
      }
    }
    
    return {
      saving,
      sessionTimeoutEnabled,
      sessionTimeoutPolicy,
      concurrentSessionEnabled,
      concurrentSessionPolicy,
      sessionMonitoringEnabled,
      sessionMonitoringPolicy,
      sessionRecordingEnabled,
      sessionRecordingPolicy,
      saveSessionTimeoutPolicy,
      saveConcurrentSessionPolicy,
      saveSessionMonitoringPolicy,
      saveSessionRecordingPolicy
    }
  }
}
</script>

<style lang="scss" scoped>
.session-security-policy {
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

  .threshold-setting,
  .storage-setting {
    display: flex;
    align-items: center;
    margin-bottom: 12px;

    label {
      font-size: 12px;
      color: #606266;
      margin-right: 8px;
      min-width: 100px;
    }

    .unit {
      margin-left: 4px;
      font-size: 12px;
    }
  }
}
</style> 