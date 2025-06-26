# EMQX ACL规则设计文档

**项目**: 安全卡片中继系统  
**版本**: 1.0.0  
**日期**: 2025-01-20  
**作者**: NFC Relay System Team  

## 1. 概述

本文档详细描述了安全卡片中继系统中EMQX MQTT Broker的访问控制列表(ACL)规则设计。设计遵循最小权限原则，确保客户端只能访问与其`clientID`相关的主题，防止未经授权的跨客户端通信。

### 部署架构

系统支持两种EMQX部署方式：

1. **远程EMQX实例** (当前推荐): 
   - 地址: 192.168.50.194
   - Dashboard: http://192.168.50.194:18083
   - 使用 `emqx_remote_setup.sh` 脚本进行配置

2. **本地Docker部署**: 
   - 本地容器化部署
   - 使用 `emqx_setup.sh` 脚本进行部署

## 2. 系统架构

### 2.1 角色定义

系统中存在以下角色：
- **传卡端客户端**: 通过NFC读取真实银行卡的移动端设备
- **收卡端客户端**: 通过HCE模拟银行卡的移动端设备  
- **服务器端组件**: 负责状态同步、角色管理、APDU中继的后端服务
- **管理员**: 具有完全访问权限的系统管理员

### 2.2 主题结构设计

基于开发手册中的规范，系统采用以下主题结构：

```
client/{clientID}/
├── status                          # 客户端在线状态
├── heartbeat                       # 心跳消息
├── control/                        # 控制指令
│   ├── set_role_request            # 角色设置请求
│   ├── set_role_response           # 角色设置响应
│   ├── role_revoked_notification   # 角色撤销通知
│   ├── peer_status_notification    # 对端状态通知
│   ├── transaction_started         # 交易开始通知
│   └── transaction_ended           # 交易结束通知
├── event/                          # 事件上报
│   └── state_update                # 状态更新事件
├── sync/                           # 状态同步
│   └── peer_state_update           # 对端状态同步
└── transaction/{transactionID}/     # 交易会话
    ├── apdu/
    │   ├── up                      # 上行APDU
    │   └── down                    # 下行APDU
    └── control/
        ├── abort                   # 中止交易
        └── complete                # 完成交易
```

## 3. ACL规则详细设计

### 3.1 服务器端权限

服务器端组件根据功能分为不同的用户角色，每个角色具有特定的权限：

#### 3.1.1 角色管理服务 (`server_role_manager`)

```erlang
% 订阅所有客户端的角色设置请求
{allow, {user, "server_role_manager"}, subscribe, ["client/+/control/set_role_request"]}.

% 发布角色设置响应
{allow, {user, "server_role_manager"}, publish, ["client/+/control/set_role_response"]}.

% 发布角色撤销通知
{allow, {user, "server_role_manager"}, publish, ["client/+/control/role_revoked_notification"]}.

% 发布对端状态通知
{allow, {user, "server_role_manager"}, publish, ["client/+/control/peer_status_notification"]}.
```

#### 3.1.2 状态同步服务 (`server_state_sync`)

```erlang
% 订阅所有客户端的状态更新事件
{allow, {user, "server_state_sync"}, subscribe, ["client/+/event/state_update"]}.

% 发布对端状态同步消息
{allow, {user, "server_state_sync"}, publish, ["client/+/sync/peer_state_update"]}.
```

#### 3.1.3 APDU中继服务 (`server_apdu_relay`)

```erlang
% 订阅所有交易的上行APDU
{allow, {user, "server_apdu_relay"}, subscribe, ["client/+/transaction/+/apdu/up"]}.

% 发布下行APDU
{allow, {user, "server_apdu_relay"}, publish, ["client/+/transaction/+/apdu/down"]}.

% 发布交易控制消息
{allow, {user, "server_apdu_relay"}, publish, ["client/+/control/transaction_started"]}.
{allow, {user, "server_apdu_relay"}, publish, ["client/+/control/transaction_ended"]}.
```

#### 3.1.4 监控服务 (`server_monitor`)

```erlang
% 监控所有客户端状态
{allow, {user, "server_monitor"}, subscribe, ["client/+/status"]}.
{allow, {user, "server_monitor"}, subscribe, ["client/+/heartbeat"]}.
```

### 3.2 客户端权限

客户端权限基于JWT中的`clientID`字段进行控制，确保客户端只能访问自己的命名空间：

#### 3.2.1 状态发布权限

```erlang
% 发布自己的在线状态
{allow, {clientid, "%c"}, publish, ["client/%c/status"]}.

% 发布心跳消息
{allow, {clientid, "%c"}, publish, ["client/%c/heartbeat"]}.

% 发布状态更新事件
{allow, {clientid, "%c"}, publish, ["client/%c/event/state_update"]}.

% 发布角色设置请求
{allow, {clientid, "%c"}, publish, ["client/%c/control/set_role_request"]}.
```

#### 3.2.2 控制消息订阅权限

```erlang
% 订阅控制指令
{allow, {clientid, "%c"}, subscribe, ["client/%c/control/#"]}.

% 订阅状态同步消息
{allow, {clientid, "%c"}, subscribe, ["client/%c/sync/#"]}.
```

#### 3.2.3 交易权限

```erlang
% 订阅下行APDU
{allow, {clientid, "%c"}, subscribe, ["client/%c/transaction/+/apdu/down"]}.

% 发布上行APDU
{allow, {clientid, "%c"}, publish, ["client/%c/transaction/+/apdu/up"]}.

% 发布交易控制消息
{allow, {clientid, "%c"}, publish, ["client/%c/transaction/+/control/abort"]}.
{allow, {clientid, "%c"}, publish, ["client/%c/transaction/+/control/complete"]}.
```

### 3.3 安全限制

#### 3.3.1 系统主题限制

```erlang
% 禁止所有用户访问系统主题
{deny, all, pubsub, ["$SYS/#"]}.
```

#### 3.3.2 跨客户端访问限制

```erlang
% 禁止客户端访问其他客户端的主题
{deny, {clientid, "%c"}, pubsub, ["client/+/#"]}.
```

#### 3.3.3 默认拒绝规则

```erlang
% 除明确允许的主题外，拒绝所有其他访问
{deny, all, pubsub, ["#"]}.
```

## 4. 权限验证流程

### 4.1 客户端连接认证

1. **JWT验证**: 客户端使用JWT作为密码连接EMQX
2. **Claims验证**: 验证JWT中的`exp`、`iss`、`client_id`字段
3. **ClientID绑定**: 将JWT中的`client_id`与MQTT的ClientID关联

### 4.2 权限检查流程

1. **规则匹配**: 按顺序匹配ACL规则
2. **用户角色检查**: 检查连接用户的角色(服务器组件 vs 客户端)
3. **主题权限验证**: 验证用户对特定主题的操作权限
4. **默认处理**: 无匹配规则时默认拒绝访问

### 4.3 设备间通信机制

由于客户端只能访问自己的`clientID`命名空间，设备间通信通过服务器中继实现：

1. **状态发布**: 传卡端发布状态到 `client/ABC/event/state_update`
2. **服务器接收**: 状态同步服务监听 `client/+/event/state_update`
3. **服务器处理**: 查找配对设备，处理状态信息
4. **状态转发**: 服务器向收卡端发布到 `client/XYZ/sync/peer_state_update`
5. **对端接收**: 收卡端订阅 `client/XYZ/sync/#` 接收同步消息

这种设计确保了客户端之间不能直接通信，所有跨设备通信都经过服务器验证和中继。

## 5. 安全考虑

### 5.1 最小权限原则

- 客户端只能访问自己的`clientID`命名空间
- 服务器组件按功能划分，具有精确的权限范围
- 默认拒绝策略确保未明确允许的操作被禁止

### 5.2 身份验证

- 使用JWT进行强身份验证
- JWT包含过期时间和签发者验证
- 客户端ID绑定防止身份冒用

### 5.3 网络隔离

- 客户端间无法直接通信
- 所有跨客户端通信必须通过服务器中继
- 系统主题完全隔离

### 5.4 审计能力

- 所有MQTT操作都会被EMQX记录
- 可通过日志追踪异常访问尝试
- 支持与后端审计系统集成

## 6. 部署配置

### 6.1 EMQX配置

在`emqx.conf`中启用ACL：

```hocon
authorization {
  sources = [
    {
      type = file
      enable = true
      path = "./etc/acl.conf"
    }
  ]
  no_match = deny
  deny_action = ignore
}
```

### 6.2 JWT认证配置

```hocon
authentication = [
  {
    enable = true
    backend = "jwt"
    mechanism = "password_based"
    secret = "78c0f08f-9663-4c9c-a399-cc4ec36b8112"
    verify_claims = [
      {name = "exp", value = "${timestamp}"},
      {name = "iss", value = "qmPlus"},
      {name = "client_id", value = "${clientid}"}
    ]
  }
]
```

## 7. 测试验证

### 7.1 权限测试用例

1. **客户端自身权限测试**
   - 验证客户端可以发布到自己的状态主题
   - 验证客户端可以订阅自己的控制主题

2. **跨客户端访问测试**
   - 验证客户端无法访问其他客户端的主题
   - 验证拒绝访问时的正确行为

3. **服务器权限测试**
   - 验证服务器组件可以访问指定的全局主题
   - 验证不同服务器角色的权限边界

### 7.2 安全测试

1. **未授权访问测试**
   - 测试无效JWT的拒绝
   - 测试过期JWT的处理

2. **权限提升测试**
   - 尝试访问更高权限主题
   - 验证ACL规则的严格执行

## 8. 维护指南

### 8.1 规则更新流程

1. 修改ACL配置文件
2. 使用EMQX API重新加载配置
3. 验证新规则生效
4. 更新相关文档

### 8.2 监控建议

- 监控ACL拒绝事件的频率
- 分析异常访问模式
- 定期审查权限配置

### 8.3 故障排除

1. **连接失败**
   - 检查JWT格式和有效性
   - 验证clientID匹配

2. **发布/订阅失败**
   - 检查主题格式
   - 验证ACL规则匹配

## 9. 版本历史

| 版本 | 日期 | 变更内容 |
|------|------|----------|
| 1.0.0 | 2025-01-20 | 初始版本，完整的ACL规则设计 |

## 10. 相关文档

- [开发手册：安全卡片中继系统](../开发手册：安全卡片中继系统.md)
  > **注意**: 开发手册中存在BifroMQ和EMQX的混合引用，建议统一更新为EMQX
- [EMQX官方文档](https://www.emqx.io/docs/)
- [JWT RFC 7519](https://tools.ietf.org/html/rfc7519) 