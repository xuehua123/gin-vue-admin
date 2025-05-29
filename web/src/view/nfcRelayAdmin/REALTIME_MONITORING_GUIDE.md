# NFC中继监控大屏 - 使用指南

## 🎯 概述

本指南介绍了如何使用新开发的NFC中继实时监控大屏功能，包括实时数据更新、全屏监控、详情查看等核心功能。

## 🚀 核心功能

### 1. 实时数据更新

#### WebSocket连接
系统使用WebSocket与后端建立长连接，实现数据的实时推送：

```javascript
// 连接示例
const { connect, isOnline, dashboardData } = useRealtimeData()

// 启动连接
await connect()

// 监听连接状态
watch(isOnline, (connected) => {
  console.log('连接状态:', connected ? '已连接' : '已断开')
})
```

#### 支持的实时事件
- `client_connected` - 客户端连接
- `client_disconnected` - 客户端断开
- `session_created` - 会话创建
- `session_terminated` - 会话终止
- `apdu_relayed` - APDU中继
- `dashboard_update` - 仪表盘数据更新

### 2. 监控大屏布局

#### 全屏切换
```vue
<template>
  <fullscreen-layout
    title="NFC中继监控大屏"
    subtitle="实时数据监控与系统状态展示"
    :connection-status="connectionStatus"
    :last-update-time="lastUpdateTime"
    @fullscreen-change="handleFullscreenChange"
    @theme-change="handleThemeChange"
  >
    <!-- 内容区域 -->
  </fullscreen-layout>
</template>
```

#### 主要功能
- **全屏模式**: 进入/退出全屏显示
- **深色主题**: 适合监控环境的深色主题切换
- **自动隐藏**: 全屏模式下自动隐藏控制栏
- **状态指示**: 实时连接状态和更新时间显示

### 3. 实时数据卡片

#### 基本用法
```vue
<template>
  <realtime-stat-card
    title="活动连接"
    :value="dashboardData.active_connections"
    subtitle="当前在线客户端"
    :icon="ConnectionIcon"
    icon-color="#67c23a"
    :connection-status="connectionStatus"
    :last-update-time="lastUpdateTime"
    :change-info="getChangeAnimation('dashboard', 'active_connections')"
    clickable
    @click="navigateToDetails"
  />
</template>
```

#### 数据变化动画
卡片支持自动检测数据变化并显示动画效果：
- 数值增加时显示绿色提示
- 数值减少时显示红色提示
- 状态变化时触发脉冲动画

### 4. 详情弹窗

#### 客户端详情
```vue
<template>
  <detail-modal
    v-model:visible="showClientDetail"
    type="client"
    :data="selectedClient"
    :loading="detailLoading"
    @refresh="refreshClientDetail"
    @disconnect-client="handleDisconnectClient"
  />
</template>
```

#### 会话详情
```vue
<template>
  <detail-modal
    v-model:visible="showSessionDetail"
    type="session"
    :data="selectedSession"
    :loading="detailLoading"
    @refresh="refreshSessionDetail"
    @terminate-session="handleTerminateSession"
  />
</template>
```

## 📊 页面功能详解

### 概览仪表盘 (Dashboard)

#### 核心指标卡片
- **运行状态**: Hub在线/离线状态
- **活动连接**: 当前连接数及Provider/Receiver分布
- **活动会话**: 当前会话数及配对状态
- **APDU中继**: 实时APDU转发统计

#### 趋势图表
- **连接数趋势**: 实时更新的连接数变化曲线
- **会话数趋势**: 实时更新的会话数变化曲线

#### 系统监控
- **性能指标**: 响应时间、系统负载、内存使用
- **实时事件**: 最近的系统事件实时流

### 连接管理 (Client Management)

#### 实时列表
- 客户端连接状态实时更新
- 在线/离线状态变化动画
- 角色变更提示

#### 详情查看
- 点击客户端查看详细信息
- 连接历史和事件时间线
- 消息统计和状态监控

#### 管理操作
- 强制断开客户端连接
- 查看客户端关联的会话
- 实时状态监控

### 会话管理 (Session Management)

#### 实时列表
- 会话创建/终止实时通知
- 配对状态变化提示
- APDU交换统计

#### 详情查看
- 双方客户端详细信息
- APDU交换统计和成功率
- 会话事件时间线

#### 管理操作
- 强制终止会话
- 查看参与方详情
- 交换历史分析

## 🎛️ 操作指南

### 启动监控大屏

1. **进入仪表盘页面**
   ```
   访问: /nfc-relay-admin/dashboard
   ```

2. **检查连接状态**
   - 顶部显示连接状态指示器
   - 绿色圆点表示连接正常
   - 橙色/红色表示连接异常

3. **启动全屏模式**
   - 点击右上角全屏按钮
   - 或使用键盘快捷键 F11

### 自定义设置

#### 主题切换
```javascript
// 手动切换主题
const handleThemeChange = (isDark) => {
  if (isDark) {
    document.body.classList.add('dark')
  } else {
    document.body.classList.remove('dark')
  }
}
```

#### 刷新频率配置
```javascript
// 配置WebSocket心跳间隔
const heartbeatInterval = 30000 // 30秒

// 配置重连间隔
const reconnectInterval = 3000 // 3秒
```

### 数据导出功能

#### 截图保存
- 点击设置菜单中的"截图保存"
- 自动生成当前大屏截图
- 支持PNG格式下载

#### 数据导出
- 支持导出Excel格式数据
- 包含当前所有监控指标
- 可选择时间范围和数据类型

## 🔧 开发集成

### 集成实时数据管理器

```javascript
import { useRealtimeData } from '../utils/realtimeDataManager'

export default {
  setup() {
    const {
      isConnected,
      connectionStatus,
      lastUpdateTime,
      dashboardData,
      clientsData,
      sessionsData,
      connect,
      disconnect,
      on,
      off,
      getChangeAnimation
    } = useRealtimeData()
    
    // 连接到服务
    onMounted(async () => {
      await connect()
    })
    
    // 清理连接
    onUnmounted(() => {
      disconnect()
    })
    
    return {
      isConnected,
      dashboardData,
      // ... 其他状态
    }
  }
}
```

### 自定义事件监听

```javascript
// 监听特定事件
const setupEventListeners = () => {
  on('client_connected', (data) => {
    ElNotification({
      title: '客户端连接',
      message: `${data.display_name} 已连接`,
      type: 'success'
    })
  })
  
  on('session_created', (data) => {
    // 处理会话创建事件
    updateSessionList()
  })
}

// 移除监听器
onUnmounted(() => {
  off('client_connected')
  off('session_created')
})
```

### 自定义组件开发

```vue
<template>
  <realtime-stat-card
    :title="customTitle"
    :value="customValue"
    :icon="customIcon"
    :connection-status="connectionStatus"
    :change-info="getChangeAnimation('custom', 'field')"
    @click="handleCustomClick"
  />
</template>

<script setup>
import { RealtimeStatCard } from '../components'

const handleCustomClick = () => {
  // 自定义点击处理
}
</script>
```

## 🚨 故障排除

### 连接问题

#### WebSocket连接失败
```javascript
// 检查连接状态
if (!isConnected.value) {
  // 手动重连
  await connect()
}

// 检查网络连接
if (navigator.onLine) {
  console.log('网络连接正常')
} else {
  console.log('网络连接断开')
}
```

#### 自动重连机制
- 最大重连次数: 5次
- 重连间隔: 3秒
- 连接超时后自动切换到模拟数据模式

### 性能优化

#### 内存管理
```javascript
// 限制事件历史记录数量
const maxEventHistory = 100

// 定期清理旧数据
setInterval(() => {
  if (recentEvents.value.length > maxEventHistory) {
    recentEvents.value = recentEvents.value.slice(0, maxEventHistory)
  }
}, 60000)
```

#### 渲染优化
- 使用虚拟滚动处理大量数据
- 图表数据点限制在100个以内
- 实时更新频率控制在30秒

### 兼容性说明

#### 浏览器支持
- Chrome 70+
- Firefox 65+
- Safari 12+
- Edge 79+

#### 移动端适配
- 响应式布局自动适配
- 触摸操作支持
- 移动端全屏API兼容

## 📈 最佳实践

### 监控大屏部署
1. **硬件要求**
   - 分辨率: 1920x1080 或更高
   - 显卡: 支持硬件加速
   - 内存: 4GB以上

2. **浏览器配置**
   - 启用硬件加速
   - 禁用自动休眠
   - 设置为默认全屏

3. **网络要求**
   - 稳定的WebSocket连接
   - 低延迟网络环境
   - 带宽要求: 1Mbps以上

### 监控策略
1. **实时监控**: 关注连接数和会话数变化
2. **异常告警**: 设置错误率和响应时间阈值
3. **历史分析**: 定期导出数据进行趋势分析
4. **性能调优**: 根据监控数据优化系统配置

## 🔗 相关文档

- [路由配置指南](./ROUTE_GUIDE.md)
- [菜单配置说明](./MENU_CONFIG.md)
- [性能优化文档](./PERFORMANCE_OPTIMIZATION.md)
- [API接口文档](../../../api/nfcRelayAdmin/README.md)

---

如有任何问题或建议，请联系开发团队或查看项目文档。 