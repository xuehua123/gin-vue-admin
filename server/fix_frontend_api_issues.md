# 前端API问题修复指南

## 🔍 问题分析

根据错误日志分析，存在以下两个主要问题：

### 1. API路径重复问题
**错误**: `http://localhost:8082/api/api/admin/nfc-relay/v1/clients`
**原因**: 
- 前端 `baseURL` 设置为 `/api`
- 后端API路径为 `/api/admin/nfc-relay/v1/...`
- 导致重复的 `/api/api/` 路径

### 2. WebSocket连接失败
**错误**: `ws://localhost:8888/ws/nfc-relay/realtime` 连接失败
**根本原因**:
1. **CORS跨域限制**: 后端没有启用CORS中间件
2. **同源策略**: 前端(8082)直接连接后端(8888)被浏览器阻止
3. **缺少WebSocket代理**: Vite只配置了HTTP代理，没有WebSocket代理

### 3. 环境配置缺失
**核心问题**: 缺少 `.env.development` 文件

## 🛠️ 完整解决方案

### ✅ 步骤1: 创建环境配置文件

在 `frontend/` 目录下创建 `.env.development` 文件：

```bash
# 前端开发环境配置
VITE_CLI_PORT = 8082
VITE_SERVER_PORT = 8888
VITE_BASE_PATH = http://localhost
VITE_BASE_API = /api
```

### ✅ 步骤2: 修改前端API配置

修改 `frontend/src/view/nfcRelayAdmin/constants.js`：

```javascript
// 修改API配置
export const API_CONFIG = {
  // HTTP API基础路径 - 去掉重复的 /api
  BASE_URL: '/admin/nfc-relay/v1',  // 从 '/api/admin/nfc-relay/v1' 改为 '/admin/nfc-relay/v1'
  
  // WebSocket配置
  WEBSOCKET: {
    getBaseUrl: () => {
      // 开发环境使用后端端口
      if (process.env.NODE_ENV === 'development') {
        return 'ws://localhost:8888'
      }
      // 生产环境使用动态检测
      const protocol = window.location.protocol === 'https:' ? 'wss:' : 'ws:'
      const host = window.location.host
      return `${protocol}//${host}`
    }
    // ... 其他配置保持不变
  }
}
```

### 🔧 步骤3: 修复WebSocket连接

#### 3.1 启用后端CORS支持
```go
// initialize/router.go
Router.Use(middleware.Cors()) // 启用跨域支持
```

#### 3.2 添加Vite WebSocket代理
```javascript
// frontend/vite.config.js
server: {
  open: true,
  port: process.env.VITE_CLI_PORT,
  proxy: {
    [process.env.VITE_BASE_API]: {
      target: `${process.env.VITE_BASE_PATH}:${process.env.VITE_SERVER_PORT}/`,
      changeOrigin: true,
      rewrite: (path) => path.replace(new RegExp('^' + process.env.VITE_BASE_API), '')
    },
    '/ws': {
      target: `ws://localhost:${process.env.VITE_SERVER_PORT}`,
      ws: true, // 启用WebSocket代理
      changeOrigin: true
    }
  }
}
```

#### 3.3 修改前端WebSocket连接方式
```javascript
// frontend/src/view/nfcRelayAdmin/utils/realtimeDataManager.js
if (process.env.NODE_ENV === 'development') {
  // 使用Vite WebSocket代理
  const host = window.location.host
  wsUrl = `${protocol}//${host}/ws/nfc-relay/realtime`
}
```

## 🚀 执行修复步骤

### 1. 重启后端服务
```bash
cd server/
# 停止当前服务 (Ctrl+C)
go run main.go
```

### 2. 重启前端服务
```bash
cd web/
# 停止当前服务 (Ctrl+C)  
npm run dev
```

## 📋 验证修复效果

修复后应该看到：

1. **API调用正确** ✅:
   - `http://localhost:8082/api/admin/nfc-relay/v1/clients`
   - 返回200状态码

2. **WebSocket连接成功** 🎯:
   - 开发环境: `ws://localhost:8082/ws/nfc-relay/realtime` (通过Vite代理)
   - 后端日志显示: "✅ WebSocket connected"
   - 前端显示: "✅ WebSocket connected"

3. **实时功能正常** 🎯:
   - 仪表盘实时数据更新
   - 客户端状态实时同步
   - 不再显示连接失败错误

## 🎯 根本原因总结

| 问题 | 根本原因 | 解决方案 |
|------|----------|----------|
| API路径重复 | 前端baseURL配置错误 | 修改constants.js中BASE_URL |
| WebSocket连接失败 | 1. 后端CORS未启用<br>2. 前端缺少WebSocket代理<br>3. 跨域策略限制 | 1. 启用后端CORS<br>2. 配置Vite WebSocket代理<br>3. 修改前端连接方式 |
| 环境变量缺失 | 缺少.env.development文件 | 创建环境配置文件 |

## 🚨 注意事项

1. **必须重启服务**: CORS和Vite代理配置需要重启服务才能生效
2. **浏览器缓存**: 可能需要清除浏览器缓存和WebSocket连接
3. **防火墙设置**: 确保8888端口未被防火墙阻止
4. **生产环境配置**: 生产环境可能需要不同的CORS和代理设置

修复这些配置问题后，前端应该能够正常与后端API和WebSocket服务通信。关键是解决跨域问题和WebSocket代理配置。

---

## 🔧 附加修复：WebSocket初始状态问题

### 问题描述
虽然WebSocket连接成功，但前端仍显示"离线"状态，需要等待30秒才能看到"在线"状态。

### 根本原因
后端WebSocket服务在客户端连接时没有立即发送初始状态数据，只有定期广播（每30秒）才会发送状态更新。

### 解决方案
修改 `nfc_relay/service/websocket_manager.go` 中的 `run()` 方法，在客户端注册时立即发送初始数据：

```go
case conn := <-s.register:
    s.mutex.Lock()
    s.clients[conn] = true
    s.mutex.Unlock()
    s.logger.Info("WebSocket client registered",
        zap.Int("total_clients", len(s.clients)))
    
    // 立即发送初始数据给新连接的客户端
    go s.sendInitialData(conn)
```

### 修复效果
- ✅ WebSocket连接后立即显示正确的"在线"状态
- ✅ 不需要等待30秒的定期广播
- ✅ 前端实时数据功能完全正常

### 重启验证
```bash
# 重启后端服务
cd server/
go run main.go

# 重启前端服务  
cd web/
npm run dev
```

修复后，前端页面应该立即显示"在线"状态和正确的实时数据。 