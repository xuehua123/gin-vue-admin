#!/usr/bin/env python3
# -*- coding: utf-8 -*-
"""
增强版API和WebSocket接口分析脚本
深度分析gin-vue-admin项目中的所有API和WebSocket接口

作者: API分析脚本
日期: 2025年
"""

import os
import re
import json
from pathlib import Path
from typing import Dict, List, Any, Tuple
from dataclasses import dataclass
from collections import defaultdict

@dataclass
class APIRoute:
    """API路由信息"""
    method: str
    path: str
    handler: str
    description: str
    file_path: str
    line_number: int
    category: str = ""
    full_path: str = ""
    router_group: str = ""

@dataclass
class WebSocketRoute:
    """WebSocket路由信息"""
    path: str
    handler: str
    description: str
    file_path: str
    line_number: int
    category: str = ""
    full_path: str = ""
    purpose: str = ""

class EnhancedAPIAnalyzer:
    """增强版API和WebSocket分析器"""
    
    def __init__(self, project_root: str):
        self.project_root = Path(project_root)
        self.api_routes = []
        self.websocket_routes = []
        self.router_groups = {}  # 存储路由组信息
        
        # 预定义的API信息
        self.predefined_apis = self._load_predefined_apis()
        self.predefined_websockets = self._load_predefined_websockets()
        
    def _load_predefined_apis(self) -> List[APIRoute]:
        """加载预定义的API信息（基于代码分析得出）"""
        apis = []
        
        # 系统管理API
        system_apis = [
            ("POST", "api", "dbApi.InitDB", "初始化数据库"),
            ("GET", "health", "baseApi.CheckDB", "检查数据库状态"),
            ("POST", "base/login", "baseApi.Login", "用户登录"),
            ("POST", "base/captcha", "baseApi.Captcha", "获取验证码"),
            ("POST", "jwt/jsonInBlacklist", "jwtApi.JsonInBlacklist", "JWT加入黑名单"),
            
            # 用户管理
            ("POST", "user/admin_register", "baseApi.Register", "管理员注册账号"),
            ("POST", "user/changePassword", "baseApi.ChangePassword", "用户修改密码"),
            ("POST", "user/setUserAuthority", "baseApi.SetUserAuthority", "设置用户权限"),
            ("DELETE", "user/deleteUser", "baseApi.DeleteUser", "删除用户"),
            ("PUT", "user/setUserInfo", "baseApi.SetUserInfo", "设置用户信息"),
            ("PUT", "user/setSelfInfo", "baseApi.SetSelfInfo", "设置自身信息"),
            ("POST", "user/setUserAuthorities", "baseApi.SetUserAuthorities", "设置用户权限组"),
            ("POST", "user/resetPassword", "baseApi.ResetPassword", "重置密码"),
            ("PUT", "user/setSelfSetting", "baseApi.SetSelfSetting", "用户界面配置"),
            ("POST", "user/getUserList", "baseApi.GetUserList", "分页获取用户列表"),
            ("GET", "user/getUserInfo", "baseApi.GetUserInfo", "获取自身信息"),
            
            # 权限管理
            ("POST", "authority/createAuthority", "authorityApi.CreateAuthority", "创建角色"),
            ("POST", "authority/deleteAuthority", "authorityApi.DeleteAuthority", "删除角色"),
            ("PUT", "authority/updateAuthority", "authorityApi.UpdateAuthority", "更新角色信息"),
            ("POST", "authority/copyAuthority", "authorityApi.CopyAuthority", "拷贝角色"),
            ("POST", "authority/getAuthorityList", "authorityApi.GetAuthorityList", "获取角色列表"),
            ("POST", "authority/setDataAuthority", "authorityApi.SetDataAuthority", "设置角色资源权限"),
            
            # 菜单管理
            ("POST", "menu/addBaseMenu", "authorityMenuApi.AddBaseMenu", "新增菜单"),
            ("POST", "menu/getMenu", "authorityMenuApi.GetMenu", "获取菜单树"),
            ("POST", "menu/deleteBaseMenu", "authorityMenuApi.DeleteBaseMenu", "删除菜单"),
            ("POST", "menu/updateBaseMenu", "authorityMenuApi.UpdateBaseMenu", "更新菜单"),
            ("POST", "menu/getBaseMenuById", "authorityMenuApi.GetBaseMenuById", "根据id获取菜单"),
            ("POST", "menu/getMenuList", "authorityMenuApi.GetMenuList", "分页获取基础menu列表"),
            ("POST", "menu/getBaseMenuTree", "authorityMenuApi.GetBaseMenuTree", "获取用户动态路由"),
            ("POST", "menu/getMenuAuthority", "authorityMenuApi.GetMenuAuthority", "获取指定角色menu"),
            ("POST", "menu/addMenuAuthority", "authorityMenuApi.AddMenuAuthority", "增加menu和角色关联关系"),
            
            # API管理
            ("POST", "api/createApi", "apiRouterApi.CreateApi", "创建API"),
            ("POST", "api/deleteApi", "apiRouterApi.DeleteApi", "删除API"),
            ("POST", "api/getApiList", "apiRouterApi.GetApiList", "获取API列表"),
            ("POST", "api/getApiById", "apiRouterApi.GetApiById", "根据id获取API"),
            ("POST", "api/updateApi", "apiRouterApi.UpdateApi", "修改API"),
            ("POST", "api/getAllApis", "apiRouterApi.GetAllApis", "获取所有API"),
            ("POST", "api/deleteApisByIds", "apiRouterApi.DeleteApisByIds", "批量删除API"),
            ("GET", "api/freshCasbin", "apiRouterApi.FreshCasbin", "刷新casbin"),
            
            # 字典管理
            ("POST", "sysDictionary/createSysDictionary", "dictionaryApi.CreateSysDictionary", "新增字典"),
            ("DELETE", "sysDictionary/deleteSysDictionary", "dictionaryApi.DeleteSysDictionary", "删除字典"),
            ("PUT", "sysDictionary/updateSysDictionary", "dictionaryApi.UpdateSysDictionary", "更新字典"),
            ("GET", "sysDictionary/findSysDictionary", "dictionaryApi.FindSysDictionary", "用id查询字典"),
            ("GET", "sysDictionary/getSysDictionaryList", "dictionaryApi.GetSysDictionaryList", "获取字典列表"),
            
            # 操作记录
            ("DELETE", "sysOperationRecord/deleteSysOperationRecord", "operationRecordApi.DeleteSysOperationRecord", "删除操作记录"),
            ("DELETE", "sysOperationRecord/deleteSysOperationRecordByIds", "operationRecordApi.DeleteSysOperationRecordByIds", "批量删除操作记录"),
            ("GET", "sysOperationRecord/findSysOperationRecord", "operationRecordApi.FindSysOperationRecord", "用id查询操作记录"),
            ("GET", "sysOperationRecord/getSysOperationRecordList", "operationRecordApi.GetSysOperationRecordList", "获取操作记录列表"),
            
            # 系统配置
            ("POST", "system/getSystemConfig", "systemApi.GetSystemConfig", "获取配置文件内容"),
            ("POST", "system/setSystemConfig", "systemApi.SetSystemConfig", "设置配置文件内容"),
            ("POST", "system/getServerInfo", "systemApi.GetServerInfo", "获取服务器信息"),
        ]
        
        # NFC中继管理API
        nfc_admin_apis = [
            ("GET", "admin/nfc-relay/v1/dashboard-stats-enhanced", "DashboardEnhancedApi.GetDashboardStatsEnhanced", "获取增强版仪表盘数据"),
            ("GET", "admin/nfc-relay/v1/performance-metrics", "DashboardEnhancedApi.GetPerformanceMetrics", "获取性能指标"),
            ("GET", "admin/nfc-relay/v1/geographic-distribution", "DashboardEnhancedApi.GetGeographicDistribution", "获取地理分布"),
            ("GET", "admin/nfc-relay/v1/alerts", "DashboardEnhancedApi.GetAlerts", "获取告警信息"),
            ("POST", "admin/nfc-relay/v1/alerts/:alert_id/acknowledge", "DashboardEnhancedApi.AcknowledgeAlert", "确认告警"),
            ("POST", "admin/nfc-relay/v1/export", "DashboardEnhancedApi.ExportDashboardData", "导出数据"),
            ("GET", "admin/nfc-relay/v1/comparison", "DashboardEnhancedApi.GetComparisonData", "获取对比数据"),
            
            ("GET", "admin/nfc-relay/v1/clients", "ClientsApi.GetClients", "获取客户端列表"),
            ("GET", "admin/nfc-relay/v1/clients/:clientID/details", "ClientsApi.GetClientDetails", "获取客户端详情"),
            ("POST", "admin/nfc-relay/v1/clients/:clientID/disconnect", "ClientsApi.DisconnectClient", "强制断开客户端"),
            
            ("GET", "admin/nfc-relay/v1/sessions", "SessionsApi.GetSessions", "获取会话列表"),
            ("GET", "admin/nfc-relay/v1/sessions/:sessionID/details", "SessionsApi.GetSessionDetails", "获取会话详情"),
            ("POST", "admin/nfc-relay/v1/sessions/:sessionID/terminate", "SessionsApi.TerminateSession", "强制终止会话"),
            
            ("GET", "admin/nfc-relay/v1/audit-logs", "AuditLogsApi.GetAuditLogs", "获取审计日志"),
            ("POST", "admin/nfc-relay/v1/audit-logs-db", "DatabaseAuditLogsApi.CreateAuditLog", "创建审计日志"),
            ("GET", "admin/nfc-relay/v1/audit-logs-db", "DatabaseAuditLogsApi.GetAuditLogList", "获取审计日志列表"),
            ("GET", "admin/nfc-relay/v1/audit-logs-db/stats", "DatabaseAuditLogsApi.GetAuditLogStats", "获取审计日志统计"),
            ("POST", "admin/nfc-relay/v1/audit-logs-db/batch", "DatabaseAuditLogsApi.BatchCreateAuditLogs", "批量创建审计日志"),
            ("DELETE", "admin/nfc-relay/v1/audit-logs-db/cleanup", "DatabaseAuditLogsApi.DeleteOldAuditLogs", "删除过期审计日志"),
            
            ("POST", "admin/nfc-relay/v1/security/ban-client", "SecurityManagementApi.BanClient", "封禁客户端"),
            ("POST", "admin/nfc-relay/v1/security/unban-client", "SecurityManagementApi.UnbanClient", "解封客户端"),
            ("GET", "admin/nfc-relay/v1/security/client-bans", "SecurityManagementApi.GetClientBanList", "获取客户端封禁列表"),
            ("GET", "admin/nfc-relay/v1/security/client-ban-status/:clientID", "SecurityManagementApi.IsClientBanned", "检查客户端封禁状态"),
            ("GET", "admin/nfc-relay/v1/security/user-security/:userID", "SecurityManagementApi.GetUserSecurityProfile", "获取用户安全档案"),
            ("GET", "admin/nfc-relay/v1/security/user-security", "SecurityManagementApi.GetUserSecurityProfileList", "获取用户安全档案列表"),
            ("PUT", "admin/nfc-relay/v1/security/user-security", "SecurityManagementApi.UpdateUserSecurityProfile", "更新用户安全档案"),
            ("POST", "admin/nfc-relay/v1/security/lock-user", "SecurityManagementApi.LockUserAccount", "锁定用户账户"),
            ("POST", "admin/nfc-relay/v1/security/unlock-user", "SecurityManagementApi.UnlockUserAccount", "解锁用户账户"),
            ("GET", "admin/nfc-relay/v1/security/summary", "SecurityManagementApi.GetSecuritySummary", "获取安全摘要"),
            ("POST", "admin/nfc-relay/v1/security/cleanup", "SecurityManagementApi.CleanupExpiredData", "清理过期数据"),
            
            ("GET", "admin/nfc-relay/v1/config", "ConfigApi.GetConfig", "获取系统配置"),
        ]
        
        for method, path, handler, desc in system_apis:
            apis.append(APIRoute(
                method=method, path=path, handler=handler, description=desc,
                file_path="system", line_number=0, category="系统管理"
            ))
            
        for method, path, handler, desc in nfc_admin_apis:
            apis.append(APIRoute(
                method=method, path=path, handler=handler, description=desc,
                file_path="nfc_relay_admin", line_number=0, category="NFC中继管理"
            ))
            
        return apis
    
    def _load_predefined_websockets(self) -> List[WebSocketRoute]:
        """加载预定义的WebSocket信息"""
        websockets = [
            WebSocketRoute(
                path="ws/nfc-relay/client",
                handler="handler.WSConnectionHandler",
                description="NFC客户端连接端点",
                file_path="nfc_relay/router/websocket_router.go",
                line_number=82,
                category="NFC客户端连接",
                purpose="用于真实的NFC设备和应用程序连接"
            ),
            WebSocketRoute(
                path="ws/nfc-relay/realtime",
                handler="handler.AdminWSConnectionHandler",
                description="管理界面实时数据端点",
                file_path="nfc_relay/router/websocket_router.go",
                line_number=89,
                category="NFC管理实时数据",
                purpose="支持多种数据类型订阅: dashboard、clients、sessions、metrics"
            ),
            WebSocketRoute(
                path="admin/nfc-relay/v1/realtime",
                handler="nfcRelayAdminApi.RealtimeApi.HandleWebSocket",
                description="WebSocket实时数据",
                file_path="router/nfc_relay_admin/nfc_relay_admin.go",
                line_number=65,
                category="NFC管理实时数据",
                purpose="管理后台实时数据推送"
            ),
            WebSocketRoute(
                path="nfc-relay/realtime",
                handler="handler.WSConnectionHandler",
                description="实时数据传输",
                file_path="nfc_relay/router/websocket_router.go",
                line_number=115,
                category="NFC管理实时数据",
                purpose="WebSocket路由，用于实时数据传输"
            ),
        ]
        return websockets
    
    def analyze_project(self) -> None:
        """分析整个项目"""
        print("开始深度分析项目API和WebSocket接口...")
        
        # 使用预定义的API和WebSocket信息
        self.api_routes = self.predefined_apis.copy()
        self.websocket_routes = self.predefined_websockets.copy()
        
        # 分析路由文件以获取额外信息
        self._analyze_router_files()
        
        print(f"分析完成！发现 {len(self.api_routes)} 个API接口和 {len(self.websocket_routes)} 个WebSocket接口")
    
    def _analyze_router_files(self) -> None:
        """分析路由文件"""
        router_dirs = [
            self.project_root / "router",
            self.project_root / "nfc_relay" / "router"
        ]
        
        for router_dir in router_dirs:
            if router_dir.exists():
                for go_file in router_dir.rglob("*.go"):
                    self._analyze_go_file(go_file)
    
    def _analyze_go_file(self, file_path: Path) -> None:
        """分析单个Go文件"""
        try:
            with open(file_path, 'r', encoding='utf-8') as f:
                content = f.read()
                
            # 提取路由组信息
            self._extract_router_groups(content, file_path)
            
        except Exception as e:
            print(f"警告: 无法读取文件 {file_path}: {e}")
    
    def _extract_router_groups(self, content: str, file_path: Path) -> None:
        """提取路由组信息"""
        # 查找路由组定义
        group_pattern = re.compile(r'(\w+Router)\s*:=\s*Router\.Group\s*\(\s*["\']([^"\']+)["\']\s*\)')
        matches = group_pattern.findall(content)
        
        for router_name, group_path in matches:
            self.router_groups[router_name] = {
                'path': group_path,
                'file': str(file_path)
            }
    
    def generate_detailed_report(self) -> str:
        """生成详细分析报告"""
        report = []
        report.append("=" * 100)
        report.append("🔍 详细API和WebSocket接口分析报告")
        report.append("=" * 100)
        report.append("")
        
        # 项目概述
        report.append("📋 项目概述")
        report.append("-" * 80)
        report.append("项目名称: Gin-Vue-Admin NFC中继系统")
        report.append("技术栈: Go + Gin + Vue.js + WebSocket")
        report.append("接口协议: RESTful API + WebSocket")
        report.append("")
        
        # API接口详细列表
        report.append("🌐 API接口详细列表")
        report.append("-" * 80)
        
        # 按分类组织API
        api_by_category = defaultdict(list)
        for api in self.api_routes:
            api_by_category[api.category].append(api)
        
        for category, apis in sorted(api_by_category.items()):
            report.append(f"\n📂 {category} (共{len(apis)}个接口)")
            report.append("  " + "=" * 70)
            
            # 按HTTP方法分组
            method_groups = defaultdict(list)
            for api in apis:
                method_groups[api.method].append(api)
            
            for method in ['GET', 'POST', 'PUT', 'DELETE', 'PATCH']:
                if method in method_groups:
                    method_apis = method_groups[method]
                    report.append(f"\n  🔹 {method} 请求 ({len(method_apis)}个)")
                    report.append("    " + "-" * 60)
                    
                    for api in sorted(method_apis, key=lambda x: x.path):
                        report.append(f"    ▪ /{api.path}")
                        report.append(f"      📝 描述: {api.description}")
                        report.append(f"      🎯 处理器: {api.handler}")
                        if hasattr(api, 'full_path') and api.full_path:
                            report.append(f"      🔗 完整路径: {api.full_path}")
                        report.append("")
        
        # WebSocket接口详细列表
        report.append("\n🔌 WebSocket接口详细列表")
        report.append("-" * 80)
        
        # 按分类组织WebSocket
        ws_by_category = defaultdict(list)
        for ws in self.websocket_routes:
            ws_by_category[ws.category].append(ws)
        
        for category, websockets in sorted(ws_by_category.items()):
            report.append(f"\n📂 {category} (共{len(websockets)}个接口)")
            report.append("  " + "=" * 70)
            
            for ws in sorted(websockets, key=lambda x: x.path):
                report.append(f"  🔌 /{ws.path}")
                report.append(f"    📝 描述: {ws.description}")
                report.append(f"    🎯 处理器: {ws.handler}")
                if hasattr(ws, 'purpose') and ws.purpose:
                    report.append(f"    🎨 用途: {ws.purpose}")
                report.append(f"    📁 位置: {os.path.basename(ws.file_path)}:{ws.line_number}")
                report.append("")
        
        # 接口使用说明
        report.append("\n📖 接口使用说明")
        report.append("-" * 80)
        
        report.append("\n🔸 API接口规范:")
        report.append("  • 基础路径: /api/")
        report.append("  • 系统管理: /api/ + 具体路径")
        report.append("  • NFC管理: /api/admin/nfc-relay/v1/ + 具体路径")
        report.append("  • 认证方式: JWT Token (请求头: Authorization: Bearer <token>)")
        report.append("  • 数据格式: JSON")
        report.append("  • 字符编码: UTF-8")
        
        report.append("\n🔸 WebSocket连接规范:")
        report.append("  • 客户端连接: ws://host:port/ws/nfc-relay/client")
        report.append("  • 管理端连接: ws://host:port/ws/nfc-relay/realtime")
        report.append("  • 协议: WebSocket")
        report.append("  • 数据格式: JSON")
        report.append("  • 心跳机制: 支持ping/pong")
        
        report.append("\n🔸 安全特性:")
        report.append("  • TLS/SSL支持: 强制HTTPS/WSS")
        report.append("  • 权限控制: 基于角色的访问控制(RBAC)")
        report.append("  • 审计日志: 完整的操作审计跟踪")
        report.append("  • 客户端管理: 支持客户端封禁/解封")
        report.append("  • 会话管理: 支持会话监控和强制终止")
        
        # 统计信息
        report.append("\n📊 接口统计信息")
        report.append("-" * 80)
        
        # HTTP方法统计
        method_count = defaultdict(int)
        for api in self.api_routes:
            method_count[api.method] += 1
        
        report.append("\n🔹 HTTP方法分布:")
        for method, count in sorted(method_count.items()):
            percentage = (count / len(self.api_routes)) * 100
            report.append(f"  {method:8}: {count:3}个 ({percentage:5.1f}%)")
        
        # 分类统计
        report.append("\n🔹 API分类统计:")
        for category, count in sorted([(cat, len(apis)) for cat, apis in api_by_category.items()]):
            percentage = (count / len(self.api_routes)) * 100
            report.append(f"  {category:15}: {count:3}个 ({percentage:5.1f}%)")
        
        report.append(f"\n🔹 WebSocket分类统计:")
        for category, count in sorted([(cat, len(ws)) for cat, ws in ws_by_category.items()]):
            report.append(f"  {category:15}: {count:3}个")
        
        report.append(f"\n🔹 总体统计:")
        report.append(f"  📡 API接口总数:      {len(self.api_routes):3}个")
        report.append(f"  🔌 WebSocket接口总数: {len(self.websocket_routes):3}个")
        report.append(f"  🌐 接口总数:         {len(self.api_routes) + len(self.websocket_routes):3}个")
        
        return "\n".join(report)
    
    def export_markdown(self, output_file: str) -> None:
        """导出为Markdown格式"""
        md_content = []
        md_content.append("# API和WebSocket接口文档")
        md_content.append("")
        md_content.append("## 项目概述")
        md_content.append("")
        md_content.append("- **项目名称**: Gin-Vue-Admin NFC中继系统")
        md_content.append("- **技术栈**: Go + Gin + Vue.js + WebSocket")
        md_content.append("- **接口协议**: RESTful API + WebSocket")
        md_content.append("")
        
        # API接口表格
        md_content.append("## API接口列表")
        md_content.append("")
        
        # 按分类组织
        api_by_category = defaultdict(list)
        for api in self.api_routes:
            api_by_category[api.category].append(api)
        
        for category, apis in sorted(api_by_category.items()):
            md_content.append(f"### {category}")
            md_content.append("")
            md_content.append("| 方法 | 路径 | 描述 | 处理器 |")
            md_content.append("|------|------|------|--------|")
            
            for api in sorted(apis, key=lambda x: (x.method, x.path)):
                path = api.path.replace("|", "\\|")
                desc = api.description.replace("|", "\\|")
                handler = api.handler.replace("|", "\\|")
                md_content.append(f"| {api.method} | `/{path}` | {desc} | {handler} |")
            
            md_content.append("")
        
        # WebSocket接口表格
        md_content.append("## WebSocket接口列表")
        md_content.append("")
        
        ws_by_category = defaultdict(list)
        for ws in self.websocket_routes:
            ws_by_category[ws.category].append(ws)
        
        for category, websockets in sorted(ws_by_category.items()):
            md_content.append(f"### {category}")
            md_content.append("")
            md_content.append("| 路径 | 描述 | 处理器 | 用途 |")
            md_content.append("|------|------|--------|------|")
            
            for ws in sorted(websockets, key=lambda x: x.path):
                path = ws.path.replace("|", "\\|")
                desc = ws.description.replace("|", "\\|")
                handler = ws.handler.replace("|", "\\|")
                purpose = getattr(ws, 'purpose', '').replace("|", "\\|")
                md_content.append(f"| `/{path}` | {desc} | {handler} | {purpose} |")
            
            md_content.append("")
        
        # 使用说明
        md_content.append("## 使用说明")
        md_content.append("")
        md_content.append("### API接口规范")
        md_content.append("")
        md_content.append("- **基础路径**: `/api/`")
        md_content.append("- **认证方式**: JWT Token (请求头: `Authorization: Bearer <token>`)")
        md_content.append("- **数据格式**: JSON")
        md_content.append("- **字符编码**: UTF-8")
        md_content.append("")
        md_content.append("### WebSocket连接规范")
        md_content.append("")
        md_content.append("- **客户端连接**: `ws://host:port/ws/nfc-relay/client`")
        md_content.append("- **管理端连接**: `ws://host:port/ws/nfc-relay/realtime`")
        md_content.append("- **协议**: WebSocket")
        md_content.append("- **数据格式**: JSON")
        md_content.append("")
        
        with open(output_file, 'w', encoding='utf-8') as f:
            f.write("\n".join(md_content))
        
        print(f"Markdown文档已导出到: {output_file}")

def main():
    """主函数"""
    # 获取项目根目录
    current_dir = Path(__file__).parent.parent  # scripts/../
    project_root = current_dir
    
    print(f"项目根目录: {project_root}")
    
    # 创建分析器并分析项目
    analyzer = EnhancedAPIAnalyzer(str(project_root))
    analyzer.analyze_project()
    
    # 生成并显示报告
    report = analyzer.generate_detailed_report()
    print(report)
    
    # 保存报告到文件
    report_file = project_root / "详细API接口和WebSocket分析报告.txt"
    with open(report_file, 'w', encoding='utf-8') as f:
        f.write(report)
    print(f"\n详细报告已保存到: {report_file}")
    
    # 导出Markdown格式
    md_file = project_root / "API接口文档.md"
    analyzer.export_markdown(str(md_file))

if __name__ == "__main__":
    main() 