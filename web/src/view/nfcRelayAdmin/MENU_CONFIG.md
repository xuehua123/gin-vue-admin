# NFC中继管理模块菜单配置指南

## 背景说明

在gin-vue-admin系统中添加NFC中继管理模块菜单时，由于该模块包含多个子菜单，需要按照框架要求创建二级路由页面。

## 重要提示：路由名称与跳转

gin-vue-admin框架使用**路由名称**而不是路径进行页面跳转。添加菜单后，系统会自动生成对应的路由名称。

### 路由名称对照表

| 菜单名称 | 组件路径 | 生成的路由名称 | 跳转代码 |
|---------|---------|-------------|---------|
| NFC中继管理 | view/nfcRelayAdmin/index.vue | nfcRelayAdminLayout | - |
| 概览仪表盘 | view/nfcRelayAdmin/dashboard/index.vue | nfcRelayDashboard | `$router.push({ name: 'nfcRelayDashboard' })` |
| 连接管理 | view/nfcRelayAdmin/clientManagement/index.vue | nfcRelayClientManagement | `$router.push({ name: 'nfcRelayClientManagement' })` |
| 会话管理 | view/nfcRelayAdmin/sessionManagement/index.vue | nfcRelaySessionManagement | `$router.push({ name: 'nfcRelaySessionManagement' })` |
| 审计日志 | view/nfcRelayAdmin/auditLogs/index.vue | nfcRelayAuditLogs | `$router.push({ name: 'nfcRelayAuditLogs' })` |
| 系统配置 | view/nfcRelayAdmin/configuration/index.vue | nfcRelayConfiguration | `$router.push({ name: 'nfcRelayConfiguration' })` |

### 带查询参数的跳转示例

```javascript
// 跳转到连接管理页面并传递客户端ID
$router.push({ 
  name: 'nfcRelayClientManagement', 
  query: { clientID: 'client-123' } 
})

// 跳转到会话管理页面并传递会话ID
$router.push({ 
  name: 'nfcRelaySessionManagement', 
  query: { sessionID: 'session-456' } 
})
```

## 已完成的配置

### 1. 父级路由页面
已创建 `frontend/src/view/nfcRelayAdmin/index.vue` 作为父级路由容器页面，包含 `<router-view>` 来承载子路由。

### 2. 路由配置文件
- `frontend/src/view/nfcRelayAdmin/router/index.js` - 完整的路由配置
- `frontend/src/view/nfcRelayAdmin/routes.js` - 简化的路由配置，用于菜单注册

## 菜单配置步骤

### 方式一：通过管理后台界面添加

1. **登录gin-vue-admin管理后台**
2. **进入菜单管理页面**（通常在系统管理 > 菜单管理）
3. **添加父级菜单**：
   ```
   菜单名称：NFC中继管理
   路径：/nfc-relay-admin
   组件路径：view/nfcRelayAdmin/index.vue
   图标：Connection
   排序：根据需要设置
   ```

4. **添加子菜单**：
   
   **4.1 概览仪表盘**
   ```
   父级菜单：NFC中继管理
   菜单名称：概览仪表盘
   路径：dashboard
   组件路径：view/nfcRelayAdmin/dashboard/index.vue
   图标：Odometer
   ```
   
   **4.2 连接管理**
   ```
   父级菜单：NFC中继管理
   菜单名称：连接管理
   路径：clients
   组件路径：view/nfcRelayAdmin/clientManagement/index.vue
   图标：User
   ```
   
   **4.3 会话管理**
   ```
   父级菜单：NFC中继管理
   菜单名称：会话管理
   路径：sessions
   组件路径：view/nfcRelayAdmin/sessionManagement/index.vue
   图标：ChatDotRound
   ```
   
   **4.4 审计日志**
   ```
   父级菜单：NFC中继管理
   菜单名称：审计日志
   路径：audit-logs
   组件路径：view/nfcRelayAdmin/auditLogs/index.vue
   图标：Document
   ```
   
   **4.5 系统配置**
   ```
   父级菜单：NFC中继管理
   菜单名称：系统配置
   路径：configuration
   组件路径：view/nfcRelayAdmin/configuration/index.vue
   图标：Setting
   ```

### 方式二：通过数据库直接添加

如果需要通过SQL直接添加菜单，可以参考以下SQL语句结构（需要根据实际的数据库表结构调整）：

```sql
-- 添加父级菜单
INSERT INTO sys_base_menus (created_at, updated_at, menu_level, parent_id, path, name, hidden, component, sort, meta_title, meta_icon, meta_keep_alive) 
VALUES (NOW(), NOW(), 0, 0, '/nfc-relay-admin', 'nfc-relay-admin', 0, 'view/nfcRelayAdmin/index.vue', 10, 'NFC中继管理', 'Connection', 1);

-- 获取父级菜单ID（假设为 parent_menu_id）

-- 添加子菜单
INSERT INTO sys_base_menus (created_at, updated_at, menu_level, parent_id, path, name, hidden, component, sort, meta_title, meta_icon, meta_keep_alive) VALUES 
(NOW(), NOW(), 1, parent_menu_id, 'dashboard', 'nfc-relay-dashboard', 0, 'view/nfcRelayAdmin/dashboard/index.vue', 1, '概览仪表盘', 'Odometer', 1),
(NOW(), NOW(), 1, parent_menu_id, 'clients', 'nfc-relay-clients', 0, 'view/nfcRelayAdmin/clientManagement/index.vue', 2, '连接管理', 'User', 1),
(NOW(), NOW(), 1, parent_menu_id, 'sessions', 'nfc-relay-sessions', 0, 'view/nfcRelayAdmin/sessionManagement/index.vue', 3, '会话管理', 'ChatDotRound', 1),
(NOW(), NOW(), 1, parent_menu_id, 'audit-logs', 'nfc-relay-audit-logs', 0, 'view/nfcRelayAdmin/auditLogs/index.vue', 4, '审计日志', 'Document', 1),
(NOW(), NOW(), 1, parent_menu_id, 'configuration', 'nfc-relay-configuration', 0, 'view/nfcRelayAdmin/configuration/index.vue', 5, '系统配置', 'Setting', 1);
```

## 权限配置

添加菜单后，还需要配置相应的API权限：

1. **进入API管理页面**
2. **添加NFC中继相关的API接口**：
   ```
   路径：/admin/nfc-relay/v1/dashboard-stats
   请求方法：GET
   API分组：NFC中继管理
   API描述：获取仪表盘统计数据
   ```
   
   依此类推，添加所有相关API接口。

3. **进入角色管理页面**
4. **为相应角色分配菜单和API权限**

## 图标说明

使用的图标都是Element Plus图标库中的图标：
- `Connection` - 连接图标，用于主菜单
- `Odometer` - 仪表盘图标
- `User` - 用户图标
- `ChatDotRound` - 聊天图标
- `Document` - 文档图标
- `Setting` - 设置图标

## 注意事项

1. **组件路径**：所有组件路径都相对于 `src/` 目录
2. **路径命名**：遵循kebab-case命名规范
3. **权限控制**：确保在角色管理中正确分配权限
4. **缓存设置**：`keepAlive` 设置为 `true` 可以缓存页面状态
5. **排序**：通过 `sort` 字段控制菜单显示顺序

## 验证配置

配置完成后，可以通过以下方式验证：

1. **重新登录系统**，查看侧边栏是否显示新菜单
2. **点击各个菜单项**，确认页面能够正常加载
3. **检查浏览器控制台**，确认没有路由错误
4. **测试页面功能**，确认API调用正常

## 故障排除

如果遇到问题，可以检查：

1. **组件路径是否正确**
2. **父级路由页面是否包含 `<router-view>`**
3. **路由名称是否冲突**
4. **权限是否正确分配**
5. **浏览器缓存是否需要清理** 