# API和WebSocket接口文档

## 项目概述

- **项目名称**: Gin-Vue-Admin NFC中继系统
- **技术栈**: Go + Gin + Vue.js + WebSocket
- **接口协议**: RESTful API + WebSocket

## API接口列表

### NFC中继管理

| 方法 | 路径 | 描述 | 处理器 |
|------|------|------|--------|
| DELETE | `/admin/nfc-relay/v1/audit-logs-db/cleanup` | 删除过期审计日志 | DatabaseAuditLogsApi.DeleteOldAuditLogs |
| GET | `/admin/nfc-relay/v1/alerts` | 获取告警信息 | DashboardEnhancedApi.GetAlerts |
| GET | `/admin/nfc-relay/v1/audit-logs` | 获取审计日志 | AuditLogsApi.GetAuditLogs |
| GET | `/admin/nfc-relay/v1/audit-logs-db` | 获取审计日志列表 | DatabaseAuditLogsApi.GetAuditLogList |
| GET | `/admin/nfc-relay/v1/audit-logs-db/stats` | 获取审计日志统计 | DatabaseAuditLogsApi.GetAuditLogStats |
| GET | `/admin/nfc-relay/v1/clients` | 获取客户端列表 | ClientsApi.GetClients |
| GET | `/admin/nfc-relay/v1/clients/:clientID/details` | 获取客户端详情 | ClientsApi.GetClientDetails |
| GET | `/admin/nfc-relay/v1/comparison` | 获取对比数据 | DashboardEnhancedApi.GetComparisonData |
| GET | `/admin/nfc-relay/v1/config` | 获取系统配置 | ConfigApi.GetConfig |
| GET | `/admin/nfc-relay/v1/dashboard-stats-enhanced` | 获取增强版仪表盘数据 | DashboardEnhancedApi.GetDashboardStatsEnhanced |
| GET | `/admin/nfc-relay/v1/geographic-distribution` | 获取地理分布 | DashboardEnhancedApi.GetGeographicDistribution |
| GET | `/admin/nfc-relay/v1/performance-metrics` | 获取性能指标 | DashboardEnhancedApi.GetPerformanceMetrics |
| GET | `/admin/nfc-relay/v1/security/client-ban-status/:clientID` | 检查客户端封禁状态 | SecurityManagementApi.IsClientBanned |
| GET | `/admin/nfc-relay/v1/security/client-bans` | 获取客户端封禁列表 | SecurityManagementApi.GetClientBanList |
| GET | `/admin/nfc-relay/v1/security/summary` | 获取安全摘要 | SecurityManagementApi.GetSecuritySummary |
| GET | `/admin/nfc-relay/v1/security/user-security` | 获取用户安全档案列表 | SecurityManagementApi.GetUserSecurityProfileList |
| GET | `/admin/nfc-relay/v1/security/user-security/:userID` | 获取用户安全档案 | SecurityManagementApi.GetUserSecurityProfile |
| GET | `/admin/nfc-relay/v1/sessions` | 获取会话列表 | SessionsApi.GetSessions |
| GET | `/admin/nfc-relay/v1/sessions/:sessionID/details` | 获取会话详情 | SessionsApi.GetSessionDetails |
| POST | `/admin/nfc-relay/v1/alerts/:alert_id/acknowledge` | 确认告警 | DashboardEnhancedApi.AcknowledgeAlert |
| POST | `/admin/nfc-relay/v1/audit-logs-db` | 创建审计日志 | DatabaseAuditLogsApi.CreateAuditLog |
| POST | `/admin/nfc-relay/v1/audit-logs-db/batch` | 批量创建审计日志 | DatabaseAuditLogsApi.BatchCreateAuditLogs |
| POST | `/admin/nfc-relay/v1/clients/:clientID/disconnect` | 强制断开客户端 | ClientsApi.DisconnectClient |
| POST | `/admin/nfc-relay/v1/export` | 导出数据 | DashboardEnhancedApi.ExportDashboardData |
| POST | `/admin/nfc-relay/v1/security/ban-client` | 封禁客户端 | SecurityManagementApi.BanClient |
| POST | `/admin/nfc-relay/v1/security/cleanup` | 清理过期数据 | SecurityManagementApi.CleanupExpiredData |
| POST | `/admin/nfc-relay/v1/security/lock-user` | 锁定用户账户 | SecurityManagementApi.LockUserAccount |
| POST | `/admin/nfc-relay/v1/security/unban-client` | 解封客户端 | SecurityManagementApi.UnbanClient |
| POST | `/admin/nfc-relay/v1/security/unlock-user` | 解锁用户账户 | SecurityManagementApi.UnlockUserAccount |
| POST | `/admin/nfc-relay/v1/sessions/:sessionID/terminate` | 强制终止会话 | SessionsApi.TerminateSession |
| PUT | `/admin/nfc-relay/v1/security/user-security` | 更新用户安全档案 | SecurityManagementApi.UpdateUserSecurityProfile |

### 系统管理

| 方法 | 路径 | 描述 | 处理器 |
|------|------|------|--------|
| DELETE | `/sysDictionary/deleteSysDictionary` | 删除字典 | dictionaryApi.DeleteSysDictionary |
| DELETE | `/sysOperationRecord/deleteSysOperationRecord` | 删除操作记录 | operationRecordApi.DeleteSysOperationRecord |
| DELETE | `/sysOperationRecord/deleteSysOperationRecordByIds` | 批量删除操作记录 | operationRecordApi.DeleteSysOperationRecordByIds |
| DELETE | `/user/deleteUser` | 删除用户 | baseApi.DeleteUser |
| GET | `/api/freshCasbin` | 刷新casbin | apiRouterApi.FreshCasbin |
| GET | `/health` | 检查数据库状态 | baseApi.CheckDB |
| GET | `/sysDictionary/findSysDictionary` | 用id查询字典 | dictionaryApi.FindSysDictionary |
| GET | `/sysDictionary/getSysDictionaryList` | 获取字典列表 | dictionaryApi.GetSysDictionaryList |
| GET | `/sysOperationRecord/findSysOperationRecord` | 用id查询操作记录 | operationRecordApi.FindSysOperationRecord |
| GET | `/sysOperationRecord/getSysOperationRecordList` | 获取操作记录列表 | operationRecordApi.GetSysOperationRecordList |
| GET | `/user/getUserInfo` | 获取自身信息 | baseApi.GetUserInfo |
| POST | `/api` | 初始化数据库 | dbApi.InitDB |
| POST | `/api/createApi` | 创建API | apiRouterApi.CreateApi |
| POST | `/api/deleteApi` | 删除API | apiRouterApi.DeleteApi |
| POST | `/api/deleteApisByIds` | 批量删除API | apiRouterApi.DeleteApisByIds |
| POST | `/api/getAllApis` | 获取所有API | apiRouterApi.GetAllApis |
| POST | `/api/getApiById` | 根据id获取API | apiRouterApi.GetApiById |
| POST | `/api/getApiList` | 获取API列表 | apiRouterApi.GetApiList |
| POST | `/api/updateApi` | 修改API | apiRouterApi.UpdateApi |
| POST | `/authority/copyAuthority` | 拷贝角色 | authorityApi.CopyAuthority |
| POST | `/authority/createAuthority` | 创建角色 | authorityApi.CreateAuthority |
| POST | `/authority/deleteAuthority` | 删除角色 | authorityApi.DeleteAuthority |
| POST | `/authority/getAuthorityList` | 获取角色列表 | authorityApi.GetAuthorityList |
| POST | `/authority/setDataAuthority` | 设置角色资源权限 | authorityApi.SetDataAuthority |
| POST | `/base/captcha` | 获取验证码 | baseApi.Captcha |
| POST | `/base/login` | 用户登录 | baseApi.Login |
| POST | `/jwt/jsonInBlacklist` | JWT加入黑名单 | jwtApi.JsonInBlacklist |
| POST | `/menu/addBaseMenu` | 新增菜单 | authorityMenuApi.AddBaseMenu |
| POST | `/menu/addMenuAuthority` | 增加menu和角色关联关系 | authorityMenuApi.AddMenuAuthority |
| POST | `/menu/deleteBaseMenu` | 删除菜单 | authorityMenuApi.DeleteBaseMenu |
| POST | `/menu/getBaseMenuById` | 根据id获取菜单 | authorityMenuApi.GetBaseMenuById |
| POST | `/menu/getBaseMenuTree` | 获取用户动态路由 | authorityMenuApi.GetBaseMenuTree |
| POST | `/menu/getMenu` | 获取菜单树 | authorityMenuApi.GetMenu |
| POST | `/menu/getMenuAuthority` | 获取指定角色menu | authorityMenuApi.GetMenuAuthority |
| POST | `/menu/getMenuList` | 分页获取基础menu列表 | authorityMenuApi.GetMenuList |
| POST | `/menu/updateBaseMenu` | 更新菜单 | authorityMenuApi.UpdateBaseMenu |
| POST | `/sysDictionary/createSysDictionary` | 新增字典 | dictionaryApi.CreateSysDictionary |
| POST | `/system/getServerInfo` | 获取服务器信息 | systemApi.GetServerInfo |
| POST | `/system/getSystemConfig` | 获取配置文件内容 | systemApi.GetSystemConfig |
| POST | `/system/setSystemConfig` | 设置配置文件内容 | systemApi.SetSystemConfig |
| POST | `/user/admin_register` | 管理员注册账号 | baseApi.Register |
| POST | `/user/changePassword` | 用户修改密码 | baseApi.ChangePassword |
| POST | `/user/getUserList` | 分页获取用户列表 | baseApi.GetUserList |
| POST | `/user/resetPassword` | 重置密码 | baseApi.ResetPassword |
| POST | `/user/setUserAuthorities` | 设置用户权限组 | baseApi.SetUserAuthorities |
| POST | `/user/setUserAuthority` | 设置用户权限 | baseApi.SetUserAuthority |
| PUT | `/authority/updateAuthority` | 更新角色信息 | authorityApi.UpdateAuthority |
| PUT | `/sysDictionary/updateSysDictionary` | 更新字典 | dictionaryApi.UpdateSysDictionary |
| PUT | `/user/setSelfInfo` | 设置自身信息 | baseApi.SetSelfInfo |
| PUT | `/user/setSelfSetting` | 用户界面配置 | baseApi.SetSelfSetting |
| PUT | `/user/setUserInfo` | 设置用户信息 | baseApi.SetUserInfo |

## WebSocket接口列表

### NFC客户端连接

| 路径 | 描述 | 处理器 | 用途 |
|------|------|--------|------|
| `/ws/nfc-relay/client` | NFC客户端连接端点 | handler.WSConnectionHandler | 用于真实的NFC设备和应用程序连接 |

### NFC管理实时数据

| 路径 | 描述 | 处理器 | 用途 |
|------|------|--------|------|
| `/admin/nfc-relay/v1/realtime` | WebSocket实时数据 | nfcRelayAdminApi.RealtimeApi.HandleWebSocket | 管理后台实时数据推送 |
| `/nfc-relay/realtime` | 实时数据传输 | handler.WSConnectionHandler | WebSocket路由，用于实时数据传输 |
| `/ws/nfc-relay/realtime` | 管理界面实时数据端点 | handler.AdminWSConnectionHandler | 支持多种数据类型订阅: dashboard、clients、sessions、metrics |

## 使用说明

### API接口规范

- **基础路径**: `/api/`
- **认证方式**: JWT Token (请求头: `Authorization: Bearer <token>`)
- **数据格式**: JSON
- **字符编码**: UTF-8

### WebSocket连接规范

- **客户端连接**: `ws://host:port/ws/nfc-relay/client`
- **管理端连接**: `ws://host:port/ws/nfc-relay/realtime`
- **协议**: WebSocket
- **数据格式**: JSON
