# NFC中继管理后台模块

## 模块概述

NFC中继管理后台是一个基于Vue 3 + Element Plus的管理界面，用于监控和管理NFC中继系统的运行状态、连接客户端和活动会话。

## 功能模块

### 1. 概览仪表盘 (Dashboard)
- **路径**: `/nfc-relay-admin/dashboard`
- **功能**: 展示系统整体运行状态和关键指标
- **特性**:
  - 实时连接数、会话数统计
  - APDU消息转发统计
  - 趋势图表展示
  - 自动刷新（每分钟）
  - 快速导航到其他模块

### 2. 连接管理 (Client Management)
- **路径**: `/nfc-relay-admin/clients`
- **功能**: 管理当前连接的WebSocket客户端
- **特性**:
  - 客户端列表查看
  - 支持按ID、用户、角色、IP筛选
  - 客户端详情查看
  - 强制断开连接功能
  - 连接历史记录

### 3. 会话管理 (Session Management)
- **路径**: `/nfc-relay-admin/sessions`
- **功能**: 管理活动的NFC中继会话
- **特性**:
  - 会话列表查看
  - Provider和Receiver信息展示
  - 会话状态监控
  - 强制终止会话功能
  - APDU交换统计

### 4. 审计日志 (Audit Logs)
- **路径**: `/nfc-relay-admin/audit-logs`
- **功能**: 查看系统审计日志
- **特性**:
  - 多维度日志筛选
  - 事件类型分类
  - 日志详情查看
  - 支持导出功能
  - 错误等级标识

### 5. 系统配置 (Configuration)
- **路径**: `/nfc-relay-admin/configuration`
- **功能**: 查看系统配置信息
- **特性**:
  - 服务器配置展示
  - 会话配置参数
  - 安全配置状态
  - 日志配置信息
  - 系统运行状态

## 技术架构

### 前端技术栈
- **框架**: Vue 3 (Composition API)
- **UI库**: Element Plus
- **构建工具**: Vite
- **状态管理**: Pinia
- **路由**: Vue Router 4
- **图表**: ECharts + vue-echarts
- **样式**: SCSS + Tailwind CSS

### 组件结构
```
nfcRelayAdmin/
├── components/              # 公共组件
│   ├── StatCard.vue        # 统计卡片
│   ├── TrendChart.vue      # 趋势图表
│   ├── SearchForm.vue      # 搜索表单
│   └── index.js            # 组件导出
├── dashboard/              # 仪表盘
├── clientManagement/       # 连接管理
├── sessionManagement/      # 会话管理
├── auditLogs/             # 审计日志
├── configuration/         # 系统配置
├── router/                # 路由配置
├── utils/                 # 工具函数
├── constants.js           # 常量定义
├── permission.js          # 权限配置
└── README.md             # 模块文档
```

### API接口设计
- **基础路径**: `/admin/nfc-relay/v1/`
- **认证**: JWT Token
- **响应格式**: 统一JSON响应结构
- **错误处理**: 标准HTTP状态码 + 业务错误码

## 开发指南

### 环境要求
- Node.js >= 16.0.0
- npm >= 8.0.0 或 yarn >= 1.22.0

### 本地开发
```bash
# 安装依赖
npm install

# 启动开发服务器
npm run dev

# 构建生产版本
npm run build
```

### 代码规范
- 使用ESLint + Prettier进行代码格式化
- 遵循Vue 3 Composition API最佳实践
- 组件命名采用PascalCase
- 文件命名采用kebab-case
- 使用TypeScript类型检查

### 最佳实践

#### 1. 组件开发
- 使用Composition API的setup语法糖
- 合理使用ref和reactive
- 组件间通信使用props/emit或provide/inject
- 复杂状态管理使用Pinia

#### 2. 数据处理
- API调用统一使用async/await
- 错误处理使用try/catch
- 数据验证在前端和后端都要进行
- 使用工具函数处理数据格式化

#### 3. 性能优化
- 合理使用v-memo和computed缓存
- 大列表使用虚拟滚动
- 图片和资源懒加载
- 组件级别的代码分割

#### 4. 用户体验
- 加载状态提示
- 错误信息友好提示
- 操作确认对话框
- 响应式设计适配

## 部署配置

### 生产环境配置
```javascript
// vite.config.js
export default {
  base: '/nfc-admin/',
  build: {
    outDir: 'dist',
    assetsDir: 'assets',
    sourcemap: false,
    minify: 'terser'
  }
}
```

### 环境变量
```bash
# .env.production
VITE_API_BASE_URL=https://api.example.com
VITE_WS_BASE_URL=wss://ws.example.com
VITE_APP_TITLE=NFC中继管理后台
```

## 权限控制

### 权限配置
- 模块级权限：控制整个NFC中继管理模块的访问
- 页面级权限：控制具体功能页面的访问
- 按钮级权限：控制操作按钮的显示和功能

### 权限检查
```javascript
import { hasPermission } from './permission'

// 检查单个权限
if (hasPermission('nfc_relay:client:disconnect')) {
  // 显示断开连接按钮
}

// 检查多个权限
if (hasAnyPermission(['nfc_relay:session:view', 'nfc_relay:session:terminate'])) {
  // 显示会话管理功能
}
```

## 常见问题

### Q: 如何添加新的统计指标？
A: 在仪表盘页面的`fetchData`方法中添加新的数据获取逻辑，在模板中使用`StatCard`组件展示。

### Q: 如何自定义搜索表单字段？
A: 修改对应页面的`searchFields`配置数组，支持input、select、datetimerange等类型。

### Q: 如何处理WebSocket实时数据？
A: 可以在页面的`onMounted`钩子中建立WebSocket连接，在`onUnmounted`中清理连接。

### Q: 如何添加新的审计日志类型？
A: 在`constants.js`中的`AUDIT_EVENT_TYPES`和`AUDIT_EVENT_TYPE_LABELS`中添加新的事件类型。

## 更新日志

### v1.0.0 (2024-01-15)
- 完成基础功能模块开发
- 实现仪表盘、连接管理、会话管理功能
- 添加审计日志查看功能
- 完成系统配置展示功能
- 建立完整的组件库和工具函数

## 贡献指南

1. Fork项目
2. 创建功能分支 (`git checkout -b feature/new-feature`)
3. 提交更改 (`git commit -am 'Add new feature'`)
4. 推送到分支 (`git push origin feature/new-feature`)
5. 创建Pull Request

## 许可证

本项目基于MIT许可证开源。 