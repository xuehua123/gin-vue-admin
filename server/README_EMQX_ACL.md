# EMQX ACL规则配置指南

## 概述

本项目为安全卡片中继系统提供了完整的EMQX MQTT Broker ACL(访问控制列表)规则配置。配置遵循最小权限原则，确保客户端权限最小化，只能访问与其`clientID`相关的主题。

## 项目结构

```
.
├── config/
│   ├── emqx.conf                    # EMQX主配置文件
│   └── emqx_acl.conf               # ACL权限规则文件
├── deploy/
│   └── emqx/
│       ├── docker-compose.yml      # Docker部署配置
│       ├── config/                 # 配置文件目录
│       └── certs/                  # SSL证书目录
├── scripts/
│   └── emqx_setup.sh              # 自动化部署脚本
├── docs/
│   └── EMQX_ACL_Design.md         # 详细设计文档
└── README_EMQX_ACL.md             # 本文件
```

## 核心特性

### 🔒 安全权限控制
- **最小权限原则**: 客户端只能访问`client/{clientID}/...`主题
- **JWT认证**: 使用项目统一的JWT密钥进行身份验证
- **角色分离**: 服务器组件和客户端权限完全分离
- **服务器中继**: 设备间通信通过服务器中继，防止直接跨客户端通信
- **默认拒绝**: 未明确允许的操作均被拒绝

### 🏗️ 主题结构设计
```
client/{clientID}/
├── status                      # 在线状态
├── heartbeat                   # 心跳消息
├── control/                    # 控制指令
├── event/                      # 事件上报  
├── sync/                       # 状态同步
└── transaction/{transactionID}/ # 交易会话
```

### 🚀 自动化部署
- Docker Compose一键部署
- 自动SSL证书生成
- 配置文件自动复制
- 健康检查和监控

## 快速开始

### 部署方式选择

#### 方式一：使用远程EMQX实例 (推荐)

如果您已有部署好的EMQX实例，可直接配置使用：

```bash
# 进入项目目录
cd backend

# 给脚本执行权限
chmod +x scripts/emqx_remote_setup.sh

# 配置远程EMQX实例
./scripts/emqx_remote_setup.sh setup

# 测试连接和配置
./scripts/emqx_remote_setup.sh test

# 查看连接信息
./scripts/emqx_remote_setup.sh info
```

**远程EMQX实例信息：**
- 地址：49.235.40.39
- Dashboard：http://49.235.40.39:18083  
- 用户名：admin
- 密码：xuehua123

#### 方式二：本地部署 (不推荐)

如果确实需要本地部署，请参考EMQX官方文档进行手动安装配置。

## 使用指南

### 客户端连接示例

使用项目的JWT Token连接EMQX:

```javascript
// JavaScript MQTT.js示例
const mqtt = require('mqtt');

// 从后端API获取JWT
const jwtToken = "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9...";
const clientId = "your-client-id-from-jwt";

// 连接远程EMQX实例
const client = mqtt.connect('mqtt://49.235.40.39:1883', {
  clientId: clientId,
  username: clientId,
  password: jwtToken,
  protocol: 'mqtt',
  protocolVersion: 5
});

// 连接成功后发布状态
client.on('connect', () => {
  console.log('Connected to EMQX');
  
  // 发布在线状态
  client.publish(`client/${clientId}/status`, JSON.stringify({
    online: true,
    timestamp_utc: new Date().toISOString()
  }));
  
  // 订阅控制消息
  client.subscribe(`client/${clientId}/control/#`);
});
```

### 设备间通信机制

**重要说明**: 客户端无法直接通信，所有设备间通信都通过服务器中继：

#### 通信流程：
1. **传卡端状态更新**:
   ```bash
   # 传卡端(client123)发布状态
   mosquitto_pub -h 49.235.40.39 -p 1883 -u client123 -P jwt_token \
     -t "client/client123/event/state_update" \
     -m '{"event_type":"nfc_transmitter_status_change","status_details":{"nfc_status":"card_detected"}}'
   ```

2. **服务器中继处理**:
   - 服务器监听 `client/+/event/state_update`
   - 查找配对的收卡端clientID
   - 转发处理后的消息

3. **收卡端接收同步**:
   ```bash
   # 收卡端(client456)订阅同步消息
   mosquitto_sub -h 49.235.40.39 -p 1883 -u client456 -P jwt_token \
     -t "client/client456/sync/#"
   ```

#### 角色配对机制：
- 用户登录后选择角色(传卡端/收卡端)
- 服务器在Redis中维护用户角色映射
- 同一用户只能有一个激活的传卡端和一个收卡端
- 新设备选择角色会"挤下线"旧设备

### 权限测试

#### 允许的操作：
```bash
# 客户端可以发布到自己的主题
mosquitto_pub -h 49.235.40.39 -p 1883 -u client123 -P jwt_token \
  -t "client/client123/status" -m '{"online":true}'

# 客户端可以订阅自己的控制主题
mosquitto_sub -h 49.235.40.39 -p 1883 -u client123 -P jwt_token \
  -t "client/client123/control/#"
```

#### 被拒绝的操作：
```bash
# 尝试访问其他客户端主题(将被拒绝)
mosquitto_pub -h 49.235.40.39 -p 1883 -u client123 -P jwt_token \
  -t "client/other_client/status" -m '{"test":true}'

# 尝试访问系统主题(将被拒绝)  
mosquitto_sub -h 49.235.40.39 -p 1883 -u client123 -P jwt_token \
  -t '$SYS/#'
```

## 运维管理

### 管理命令

```bash
# 配置远程EMQX实例
./scripts/emqx_remote_setup.sh setup

# 测试连接和配置
./scripts/emqx_remote_setup.sh test

# 查看连接信息
./scripts/emqx_remote_setup.sh info

# 测试连接详情
python scripts/test_emqx_connection.py
```

### 配置修改

1. **修改ACL规则**:
   ```bash
   # 编辑ACL配置
   vi config/emqx_acl.conf
   
   # 重启服务使配置生效
   ./scripts/emqx_setup.sh --restart
   ```

2. **修改EMQX配置**:
   ```bash
   # 编辑主配置
   vi config/emqx.conf
   
   # 重启服务
   ./scripts/emqx_setup.sh --restart
   ```

3. **更新JWT密钥**:
   ```bash
   # 更新config/emqx.conf中的JWT secret
   # 同时需要更新后端API的JWT配置
   ```

### 监控和日志

```bash
# 查看EMQX日志
docker logs -f emqx_nfc_relay

# 查看连接统计
curl -u admin:nfc_relay_admin_2024 \
  http://localhost:18083/api/v5/stats

# 查看客户端连接
curl -u admin:nfc_relay_admin_2024 \
  http://localhost:18083/api/v5/clients
```

## 故障排除

### 常见问题

1. **连接被拒绝**
   ```
   原因: JWT验证失败
   解决: 检查JWT格式、过期时间、签名密钥
   ```

2. **发布/订阅失败**
   ```
   原因: ACL权限不足
   解决: 检查主题格式是否匹配client/{clientID}/...
   ```

3. **SSL连接失败**
   ```
   原因: 证书问题
   解决: 重新生成证书或检查证书配置
   ```

### 调试步骤

1. **检查服务状态**:
   ```bash
   docker ps | grep emqx
   ./scripts/emqx_setup.sh --verify
   ```

2. **查看详细日志**:
   ```bash
   # EMQX日志
   ./scripts/emqx_setup.sh --logs
   
   # 认证日志
   docker exec emqx_nfc_relay tail -f /opt/emqx/log/emqx.log | grep auth
   ```

3. **测试连接**:
   ```bash
   # 使用MQTT客户端测试
   mosquitto_pub -h localhost -p 1883 -u test -P test \
     -t "test/topic" -m "test message" -d
   ```

## 安全建议

### 生产环境配置

1. **修改默认密码**:
   ```bash
   # 修改Dashboard密码
   vi deploy/emqx/docker-compose.yml
   # 更新 EMQX_DASHBOARD__DEFAULT_PASSWORD
   ```

2. **启用TLS**:
   ```bash
   # 使用正式SSL证书替换自签名证书
   cp your-cert.pem deploy/emqx/certs/cert.pem
   cp your-key.pem deploy/emqx/certs/key.pem
   ```

3. **网络安全**:
   ```bash
   # 限制Dashboard访问IP
   # 使用防火墙限制MQTT端口访问
   # 配置反向代理
   ```

4. **密钥管理**:
   ```bash
   # 定期轮换JWT密钥
   # 使用环境变量管理敏感配置
   # 启用密钥审计
   ```

## 相关文档

- [EMQX ACL设计文档](docs/EMQX_ACL_Design.md) - 详细的设计规范
- [开发手册：安全卡片中继系统](开发手册：安全卡片中继系统.md) - 系统整体设计
- [EMQX官方文档](https://www.emqx.io/docs/) - EMQX产品文档

## 技术支持

如需技术支持，请提供以下信息：
- 系统环境信息
- 错误日志
- 配置文件内容
- 复现步骤

---

**项目**: 安全卡片中继系统  
**版本**: 1.0.0  
**维护**: NFC Relay System Team 