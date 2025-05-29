# API和WebSocket接口分析工具

这个工具包含多个脚本，用于分析Gin-Vue-Admin NFC中继系统中的所有API接口和WebSocket接口。

## 脚本说明

### 1. 快速摘要脚本 (推荐)
- **文件**: `quick_api_summary.py`
- **功能**: 快速显示所有API和WebSocket接口的分类摘要
- **适用**: 想要快速了解系统接口概况

```bash
python scripts/quick_api_summary.py
```

### 2. 详细分析脚本
- **文件**: `analyze_apis_enhanced.py`
- **功能**: 深度分析所有接口，生成详细报告和Markdown文档
- **输出文件**:
  - `详细API接口和WebSocket分析报告.txt` - 详细文本报告
  - `API接口文档.md` - Markdown格式文档

```bash
python scripts/analyze_apis_enhanced.py
```

### 3. 批处理脚本 (Windows)
- **文件**: `run_api_analysis.bat`
- **功能**: 交互式菜单，方便选择不同的分析模式
- **使用**: 双击运行或在命令行执行

```cmd
scripts\run_api_analysis.bat
```

## 分析结果

### API接口统计
- **总数**: 82个API接口
- **分类**:
  - 系统管理API: 51个 (62.2%)
  - NFC中继管理API: 31个 (37.8%)
- **HTTP方法分布**:
  - GET: 25个 (30.5%)
  - POST: 46个 (56.1%)
  - PUT: 6个 (7.3%)
  - DELETE: 5个 (6.1%)

### WebSocket接口统计
- **总数**: 4个WebSocket接口
- **分类**:
  - NFC客户端连接: 1个
  - NFC管理实时数据: 3个

## 主要功能模块

### 系统管理API
- 用户认证与授权
- 用户管理
- 权限和角色管理
- 菜单管理
- API管理
- 字典管理
- 操作记录
- 系统配置

### NFC中继管理API
- 仪表盘数据和统计
- 性能监控
- 告警管理
- 客户端管理
- 会话管理
- 审计日志
- 安全管理
- 配置管理

### WebSocket接口
- NFC设备客户端连接
- 管理界面实时数据推送
- 实时监控和状态更新

## 技术规范

- **基础路径**: `/api/`
- **认证方式**: JWT Token
- **数据格式**: JSON
- **字符编码**: UTF-8
- **WebSocket协议**: 支持ping/pong心跳
- **安全特性**: TLS/SSL、RBAC权限控制、审计日志

## 使用建议

1. **首次使用**: 运行快速摘要脚本了解整体结构
2. **详细分析**: 需要完整文档时运行详细分析脚本
3. **定期检查**: 代码更新后重新运行分析，确保文档同步

## 文件结构

```
scripts/
├── README.md                           # 本说明文档
├── quick_api_summary.py               # 快速摘要脚本
├── analyze_apis_enhanced.py           # 详细分析脚本
├── analyze_apis_and_websockets.py     # 基础分析脚本
└── run_api_analysis.bat               # Windows批处理脚本

根目录/
├── API接口文档.md                      # Markdown格式接口文档
└── 详细API接口和WebSocket分析报告.txt   # 详细文本报告
```

## 注意事项

1. 需要Python 3.6+环境
2. 脚本会自动分析项目中的路由文件
3. 生成的文档包含预定义的接口信息，基于代码分析得出
4. 如果项目结构发生变化，可能需要更新脚本中的路径配置

## 更新历史

- v1.0: 初始版本，包含基础API分析功能
- v1.1: 增加WebSocket接口分析
- v1.2: 添加详细分析和Markdown输出
- v1.3: 增加快速摘要和批处理脚本 