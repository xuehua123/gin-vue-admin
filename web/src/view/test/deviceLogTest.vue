<template>
  <div class="test-page">
    <h1>设备日志管理测试页面</h1>
    
    <el-card style="margin-bottom: 20px;">
      <template #header>
        <span>API测试</span>
      </template>
      
      <el-row :gutter="20">
        <el-col :span="8">
          <el-button type="primary" @click="testGetDeviceLogs" :loading="loading1">
            测试获取设备日志
          </el-button>
        </el-col>
        <el-col :span="8">
          <el-button type="warning" @click="testGetStats" :loading="loading2">
            测试获取统计信息
          </el-button>
        </el-col>
        <el-col :span="8">
          <el-button type="danger" @click="testForceLogout" :loading="loading3">
            测试强制下线
          </el-button>
        </el-col>
      </el-row>
    </el-card>

    <el-card v-if="testResults.length > 0">
      <template #header>
        <span>测试结果</span>
      </template>
      
      <div v-for="(result, index) in testResults" :key="index" style="margin-bottom: 10px;">
        <el-tag :type="result.success ? 'success' : 'danger'">
          {{ result.name }}
        </el-tag>
        <span style="margin-left: 10px;">{{ result.message }}</span>
        <pre v-if="result.data" style="margin-top: 10px; background: #f5f5f5; padding: 10px;">{{ JSON.stringify(result.data, null, 2) }}</pre>
      </div>
    </el-card>
  </div>
</template>

<script setup>
import { ref } from 'vue'
import { ElMessage } from 'element-plus'
import { getDeviceLogsList, getDeviceLogStats, forceLogoutDevice } from '@/api/deviceLog'

defineOptions({
  name: 'DeviceLogTest'
})

const loading1 = ref(false)
const loading2 = ref(false)
const loading3 = ref(false)
const testResults = ref([])

const addResult = (name, success, message, data = null) => {
  testResults.value.push({
    name,
    success,
    message,
    data,
    time: new Date().toLocaleString()
  })
}

const testGetDeviceLogs = async () => {
  loading1.value = true
  try {
    const params = {
      page: 1,
      pageSize: 10,
      userId: '',
      clientId: '',
      deviceModel: '',
      ipAddress: '',
      onlineOnly: false
    }
    
    const res = await getDeviceLogsList(params)
    
    if (res.code === 0) {
      addResult('获取设备日志', true, '成功获取设备日志列表', res.data)
      ElMessage.success('获取设备日志成功')
    } else {
      addResult('获取设备日志', false, res.msg || '获取失败', res)
      ElMessage.error(res.msg || '获取设备日志失败')
    }
  } catch (error) {
    addResult('获取设备日志', false, error.message, error)
    ElMessage.error('请求失败：' + error.message)
  } finally {
    loading1.value = false
  }
}

const testGetStats = async () => {
  loading2.value = true
  try {
    const res = await getDeviceLogStats('')
    
    if (res.code === 0) {
      addResult('获取统计信息', true, '成功获取统计信息', res.data)
      ElMessage.success('获取统计信息成功')
    } else {
      addResult('获取统计信息', false, res.msg || '获取失败', res)
      ElMessage.error(res.msg || '获取统计信息失败')
    }
  } catch (error) {
    addResult('获取统计信息', false, error.message, error)
    ElMessage.error('请求失败：' + error.message)
  } finally {
    loading2.value = false
  }
}

const testForceLogout = async () => {
  loading3.value = true
  try {
    const params = {
      userId: 'test-user-id',
      clientId: 'test-client-id',
      reason: 'admin_forced_logout - 测试强制下线'
    }
    
    const res = await forceLogoutDevice(params)
    
    if (res.code === 0) {
      addResult('强制下线', true, '强制下线成功', res.data)
      ElMessage.success('强制下线成功')
    } else {
      addResult('强制下线', false, res.msg || '操作失败', res)
      ElMessage.error(res.msg || '强制下线失败')
    }
  } catch (error) {
    addResult('强制下线', false, error.message, error)
    ElMessage.error('请求失败：' + error.message)
  } finally {
    loading3.value = false
  }
}
</script>

<style scoped>
.test-page {
  padding: 20px;
}

h1 {
  color: #303133;
  margin-bottom: 20px;
}

pre {
  font-size: 12px;
  overflow-x: auto;
  white-space: pre-wrap;
  word-wrap: break-word;
}
</style> 