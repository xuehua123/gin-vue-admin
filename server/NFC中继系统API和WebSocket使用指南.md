# NFC中继系统 API和WebSocket 使用指南

## 📋 概述

本文档说明NFC中继系统的API接口和WebSocket连接的正确使用方式，以及修复前端调用问题的方案。

## 🌐 WebSocket 接口

### 🔗 连接信息

| 属性 | 值 |
|------|---|
| **端点URL** | `/ws/nfc-relay/realtime` |
| **完整路径示例** | `ws://localhost:8888/ws/nfc-relay/realtime` |
| **协议** | WebSocket (ws:// 或 wss://) |
| **用途** | 实时数据传输、日志流、APDU监控、系统指标 |

### 📨 消息格式

WebSocket使用JSON格式的结构化消息：

```json
{
  "type": "subscribe|unsubscribe|ping|pong|realtime_data|log_entry|apdu_data|metrics_data|error",
  "topic": "logs|apdu|metrics|realtime",
  "data": { /* 具体数据内容 */ },
  "timestamp": "2025-01-29T10:30:00Z",
  "client_id": "optional-client-id"
}
```

### 🎯 支持的订阅主题

| 主题 | 描述 | 数据类型 |
|------|------|----------|
| `logs` | 系统日志流 | 日志条目 |
| `apdu` | APDU命令监控 | APDU命令和响应 |
| `metrics` | 系统性能指标 | CPU、内存、网络等指标 |
| `realtime` | 实时状态数据 | 连接数、会话数等 |

### 🔄 前端连接示例

#### 使用统一配置连接

```javascript
import { API_CONFIG, MESSAGE_TYPES } from '@/view/nfcRelayAdmin/constants.js'

// 获取WebSocket URL
const wsUrl = API_CONFIG.WEBSOCKET.getUrl(API_CONFIG.WEBSOCKET.ENDPOINTS.REALTIME)
// 结果: ws://localhost:8888/ws/nfc-relay/realtime

const ws = new WebSocket(wsUrl)

// 连接成功后订阅数据
ws.onopen = () => {
  console.log('WebSocket连接成功')
  
  // 订阅日志流
  ws.send(JSON.stringify({
    type: MESSAGE_TYPES.SUBSCRIBE,
    topic: 'logs'
  }))
  
  // 订阅实时数据
  ws.send(JSON.stringify({
    type: MESSAGE_TYPES.SUBSCRIBE,
    topic: 'realtime'
  }))
}

// 处理接收到的消息
ws.onmessage = (event) => {
  try {
    const message = JSON.parse(event.data)
    
    switch (message.type) {
      case MESSAGE_TYPES.LOG_ENTRY:
        console.log('收到日志:', message.data)
        break
      case MESSAGE_TYPES.REALTIME_DATA:
        console.log('收到实时数据:', message.data)
        break
      case MESSAGE_TYPES.PONG:
        console.log('心跳响应:', message.data)
        break
    }
  } catch (error) {
    console.error('解析WebSocket消息失败:', error)
  }
}
```

#### 使用前端Hook

```javascript
import { useWebSocketConnection } from '@/view/nfcRelayAdmin/hooks/useRealTime.js'
import { API_CONFIG, MESSAGE_TYPES } from '@/view/nfcRelayAdmin/constants.js'

const { status, connect, disconnect, send } = useWebSocketConnection(
  API_CONFIG.WEBSOCKET.getUrl(API_CONFIG.WEBSOCKET.ENDPOINTS.REALTIME),
  {
    onMessage: (data) => {
      const message = JSON.parse(data)
      // 处理消息
    },
    onOpen: () => {
      // 订阅需要的主题
      send(JSON.stringify({
        type: MESSAGE_TYPES.SUBSCRIBE,
        topic: 'logs'
      }))
    }
  }
)

// 连接
connect()
```

## 🔌 HTTP API 接口

### 🌍 基础配置

| 属性 | 值 |
|------|---|
| **基础路径** | `/admin/nfc-relay/v1` |
| **认证方式** | JWT Token (在请求头中) |
| **数据格式** | JSON |

### 📑 API 分类

#### 1. 仪表盘 API

| 方法 | 路径 | 描述 |
|------|------|------|
| GET | `/dashboard-stats-enhanced` | 获取增强版仪表盘数据 |
| GET | `/performance-metrics` | 获取性能指标 |
| GET | `/geographic-distribution` | 获取地理分布 |
| GET | `/alerts` | 获取告警信息 |
| POST | `/alerts/:alert_id/acknowledge` | 确认告警 |
| POST | `/export` | 导出数据 |
| GET | `/comparison` | 获取对比数据 |

#### 2. 客户端管理 API

| 方法 | 路径 | 描述 |
|------|------|------|
| GET | `/clients` | 获取客户端列表 |
| GET | `/clients/:clientID/details` | 获取客户端详情 |
| POST | `/clients/:clientID/disconnect` | 强制断开客户端 |

#### 3. 会话管理 API

| 方法 | 路径 | 描述 |
|------|------|------|
| GET | `/sessions` | 获取会话列表 |
| GET | `/sessions/:sessionID/details` | 获取会话详情 |
| POST | `/sessions/:sessionID/terminate` | 强制终止会话 |

#### 4. 审计日志 API

| 方法 | 路径 | 描述 |
|------|------|------|
| GET | `/audit-logs` | 获取审计日志 |

#### 5. 系统配置 API

| 方法 | 路径 | 描述 |
|------|------|------|
| GET | `/config` | 获取系统配置 |

### 📞 前端API调用示例

```javascript
import { getDashboardStatsEnhanced, getClientsList } from '@/api/nfcRelayAdmin.js'

// 获取仪表盘数据
try {
  const response = await getDashboardStatsEnhanced({
    timeRange: '1h',
    includeDetails: true
  })
  console.log('仪表盘数据:', response.data)
} catch (error) {
  console.error('获取仪表盘数据失败:', error)
}

// 获取客户端列表
try {
  const response = await getClientsList({
    page: 1,
    pageSize: 20,
    status: ['online', 'offline']
  })
  console.log('客户端列表:', response.data)
} catch (error) {
  console.error('获取客户端列表失败:', error)
}
```

## 🔧 修复的问题

### 1. ❌ 原问题：WebSocket URL 硬编码

**修复前**:
```javascript
// 错误的硬编码URL
ws = new WebSocket('ws://localhost:8888/nfc-relay/log-stream')
ws = new WebSocket('ws://localhost:8888/nfc-relay/apdu-monitor')
```

**修复后**:
```javascript
// 使用统一配置
const wsUrl = API_CONFIG.WEBSOCKET.getUrl(API_CONFIG.WEBSOCKET.ENDPOINTS.REALTIME)
ws = new WebSocket(wsUrl)
```

### 2. ❌ 原问题：消息格式不统一

**修复前**:
```javascript
// 直接解析为对象
const logEntry = JSON.parse(event.data)
```

**修复后**:
```javascript
// 使用结构化消息格式
const message = JSON.parse(event.data)
if (message.type === MESSAGE_TYPES.LOG_ENTRY) {
  const logEntry = message.data
}
```

### 3. ✅ API路径已正确

前端API配置已经正确使用 `/admin/nfc-relay/v1` 路径，与后端路由匹配。

## 🏗 架构设计

### WebSocket 消息流

```
前端组件 → WebSocket连接 → 订阅管理器 → 数据生产者
    ↑                                           ↓
 消息处理 ← JSON消息格式 ← 主题广播 ← 业务数据
```

### API 请求流

```
前端组件 → API调用 → HTTP路由 → 业务逻辑 → 数据库
    ↑                                      ↓
JSON响应 ← 中间件处理 ← 控制器 ← 服务层 ← 数据访问
```

## 🔑 配置要点

### 1. 环境变量

确保配置文件中有正确的端口和路径设置：

```yaml
# config.yaml
server:
  addr: 8888  # 服务器端口

nfcRelay:
  websocketPongWaitSec: 60
  websocketMaxMessageBytes: 2048
```

### 2. 前端环境配置

```javascript
// .env.development
VUE_APP_BASE_API = http://localhost:8888

// .env.production  
VUE_APP_BASE_API = https://your-domain.com
```

### 3. WebSocket 安全配置

生产环境中需要配置CORS和Origin验证：

```go
// websocket_handler.go
CheckOrigin: func(r *http.Request) bool {
    origin := r.Header.Get("Origin")
    return origin == "https://your-frontend-domain.com"
}
```

## 🚀 使用建议

### 1. 错误处理

```javascript
// WebSocket连接失败处理
ws.onerror = (error) => {
  console.error('WebSocket错误:', error)
  // 实现重连逻辑
}

ws.onclose = (event) => {
  console.log('WebSocket连接关闭:', event.code, event.reason)
  // 根据关闭代码决定是否重连
}
```

### 2. 性能优化

- 只订阅需要的主题
- 适当设置心跳间隔
- 限制消息缓冲区大小
- 实现消息去重

### 3. 调试技巧

```javascript
// 启用WebSocket调试
const ws = new WebSocket(wsUrl)
ws.addEventListener('message', (event) => {
  console.log('🔥 WebSocket收到消息:', event.data)
})

// API调试
import { service } from '@/utils/request'
service.interceptors.response.use(response => {
  console.log('📡 API响应:', response.config.url, response.data)
  return response
})
```

## 📚 参考资料

- [WebSocket API 文档](https://developer.mozilla.org/en-US/docs/Web/API/WebSocket)
- [Gin WebSocket 示例](https://github.com/gin-gonic/examples/tree/master/websocket)
- [Vue.js WebSocket 最佳实践](https://vuejs.org/guide/extras/reactivity-in-depth.html)

---

本指南涵盖了NFC中继系统的完整API和WebSocket使用方式。如有问题，请参考具体的错误日志进行调试。 