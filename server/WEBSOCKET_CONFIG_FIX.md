# WebSocket连接配置修复指南

## 🔍 问题根源
经过系统分析，WebSocket连接失败的根本原因如下：

### 1. **端口不匹配**
- ❌ 前端连接: `ws://localhost:8082/api/nfc-relay/realtime`
- ✅ 后端运行: `localhost:8888` (配置文件 `config.yaml` 中 `system.addr: 8888`)

### 2. **路由前缀**
- 配置文件中 `system.router-prefix: ""` (空字符串)
- 实际WebSocket路径: `/nfc-relay/realtime`

## 🚀 解决方案

### 修复前端配置
请修改前端 `realtimeDataManager.js` 文件中的WebSocket连接地址：

```javascript
// 修改前
const WS_URL = 'ws://localhost:8082/api/nfc-relay/realtime'

// 修改后
const WS_URL = 'ws://localhost:8888/nfc-relay/realtime'
```

### 完整的前端修复代码

```javascript
// realtimeDataManager.js
class RealtimeDataManager {
  constructor() {
    // 🔥 关键修复：使用正确的端口和路径
    this.wsUrl = 'ws://localhost:8888/nfc-relay/realtime'
    this.ws = null
    this.reconnectAttempts = 0
    this.maxReconnectAttempts = 5
    this.reconnectInterval = 3000
    
    // ... 其余代码保持不变
  }
  
  // ... 其余代码保持不变
}
```

## 🔧 后端修复状态

✅ **已完成的后端修复:**
1. 修正了初始化顺序，确保WebSocket服务在路由注册前初始化
2. 增强了路由注册的日志和错误处理
3. 添加了模拟数据生成器用于测试
4. 服务现在正确运行在8888端口

## 📡 正确的WebSocket端点信息

- **协议**: WebSocket (ws://)
- **主机**: localhost
- **端口**: 8888 
- **路径**: /nfc-relay/realtime
- **完整地址**: `ws://localhost:8888/nfc-relay/realtime`

## 🧪 验证方法

### 1. 后端验证
```bash
# 检查8888端口是否监听
netstat -an | findstr :8888

# 应该看到类似输出：
# TCP    0.0.0.0:8888           0.0.0.0:0              LISTENING
```

### 2. 前端验证
1. 修改 `realtimeDataManager.js` 中的WebSocket URL
2. 刷新页面
3. 打开浏览器开发者工具，查看控制台
4. 应该看到WebSocket连接成功的日志

### 3. 功能验证
- ✅ WebSocket连接建立成功
- ✅ 接收初始仪表盘数据
- ✅ 每10秒接收模拟事件数据
- ✅ 实时数据卡片显示更新

## 🎯 测试WebSocket连接

你可以使用浏览器开发者工具或Postman等工具测试WebSocket连接：

```javascript
// 浏览器控制台测试
const ws = new WebSocket('ws://localhost:8888/nfc-relay/realtime')
ws.onopen = () => console.log('✅ WebSocket连接成功')
ws.onmessage = (e) => console.log('📨 收到消息:', JSON.parse(e.data))
ws.onerror = (e) => console.log('❌ WebSocket错误:', e)
```

## 📝 预期的WebSocket消息类型

连接成功后，你将收到以下类型的消息：

1. **初始数据推送**:
   - `dashboard_update`: 仪表盘统计
   - `clients_update`: 客户端列表
   - `sessions_update`: 会话列表

2. **实时事件推送** (每10秒):
   - `client_connected`: 客户端连接事件
   - `session_created`: 会话创建事件
   - `apdu_relayed`: APDU中继事件
   - `client_disconnected`: 客户端断开事件

## 🎉 修复完成

完成上述前端URL修改后，你的NFC中继监控大屏应该能够正常显示实时数据了！ 