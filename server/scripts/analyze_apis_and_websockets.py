#!/usr/bin/env python3
# -*- coding: utf-8 -*-
"""
API和WebSocket接口分析脚本
分析gin-vue-admin项目中的所有API和WebSocket接口

作者: API分析脚本
日期: 2025年
"""

import os
import re
import json
from pathlib import Path
from typing import Dict, List, Any
from dataclasses import dataclass
from collections import defaultdict

@dataclass
class APIEndpoint:
    """API端点信息"""
    method: str
    path: str
    handler: str
    description: str
    file_path: str
    line_number: int
    category: str = ""

@dataclass
class WebSocketEndpoint:
    """WebSocket端点信息"""
    path: str
    handler: str
    description: str
    file_path: str
    line_number: int
    category: str = ""

class APIAnalyzer:
    """API和WebSocket分析器"""
    
    def __init__(self, project_root: str):
        self.project_root = Path(project_root)
        self.api_endpoints = []
        self.websocket_endpoints = []
        
        # HTTP方法模式
        self.http_methods = ['GET', 'POST', 'PUT', 'DELETE', 'PATCH', 'OPTIONS', 'HEAD']
        
        # 路由模式
        self.route_pattern = re.compile(
            r'(\w+Router(?:WithoutRecord)?)\.(GET|POST|PUT|DELETE|PATCH|OPTIONS|HEAD)\s*\(\s*["\']([^"\']+)["\']\s*,\s*(\w+(?:\.\w+)*)',
            re.IGNORECASE
        )
        
        # WebSocket模式
        self.websocket_pattern = re.compile(
            r'(\w+Router)\.(GET)\s*\(\s*["\']([^"\']+)["\']\s*,\s*(\w+(?:\.\w+)*)',
            re.IGNORECASE
        )
        
        # 注释模式（用于获取API说明）
        self.comment_pattern = re.compile(r'//\s*(.+)')
        
    def analyze_go_file(self, file_path: Path) -> None:
        """分析单个Go文件"""
        try:
            with open(file_path, 'r', encoding='utf-8') as f:
                content = f.read()
                lines = content.split('\n')
                
            self._extract_routes_from_content(lines, file_path)
            
        except Exception as e:
            print(f"警告: 无法读取文件 {file_path}: {e}")
    
    def _extract_routes_from_content(self, lines: List[str], file_path: Path) -> None:
        """从文件内容中提取路由信息"""
        for i, line in enumerate(lines):
            line = line.strip()
            
            # 跳过注释行
            if line.startswith('//') or line.startswith('/*'):
                continue
            
            # 匹配路由定义
            route_match = self.route_pattern.search(line)
            if route_match:
                method = route_match.group(2).upper()
                path = route_match.group(3)
                handler = route_match.group(4)
                
                # 获取注释说明
                description = self._get_description_from_context(lines, i)
                
                # 判断是否为WebSocket
                if self._is_websocket_route(path, handler, description):
                    ws_endpoint = WebSocketEndpoint(
                        path=path,
                        handler=handler,
                        description=description,
                        file_path=str(file_path),
                        line_number=i + 1,
                        category=self._categorize_websocket(path, handler)
                    )
                    self.websocket_endpoints.append(ws_endpoint)
                else:
                    api_endpoint = APIEndpoint(
                        method=method,
                        path=path,
                        handler=handler,
                        description=description,
                        file_path=str(file_path),
                        line_number=i + 1,
                        category=self._categorize_api(path, handler)
                    )
                    self.api_endpoints.append(api_endpoint)
    
    def _get_description_from_context(self, lines: List[str], line_index: int) -> str:
        """从代码上下文获取API描述"""
        description = ""
        
        # 查找同行注释
        current_line = lines[line_index]
        comment_match = re.search(r'//\s*(.+)', current_line)
        if comment_match:
            description = comment_match.group(1).strip()
        
        # 如果同行没有注释，查找上一行注释
        if not description and line_index > 0:
            prev_line = lines[line_index - 1].strip()
            comment_match = re.search(r'//\s*(.+)', prev_line)
            if comment_match:
                description = comment_match.group(1).strip()
        
        return description
    
    def _is_websocket_route(self, path: str, handler: str, description: str) -> bool:
        """判断是否为WebSocket路由"""
        websocket_indicators = [
            'websocket', 'ws', 'realtime', 'HandleWebSocket',
            'WSConnectionHandler', 'AdminWSConnectionHandler'
        ]
        
        text_to_check = f"{path} {handler} {description}".lower()
        return any(indicator.lower() in text_to_check for indicator in websocket_indicators)
    
    def _categorize_api(self, path: str, handler: str) -> str:
        """对API进行分类"""
        if 'nfc' in path.lower() or 'nfc' in handler.lower():
            if 'admin' in path.lower():
                return "NFC中继管理"
            else:
                return "NFC中继核心"
        elif 'system' in path.lower() or 'sys' in handler.lower():
            return "系统管理"
        elif 'user' in path.lower():
            return "用户管理"
        elif 'auth' in path.lower():
            return "权限管理"
        elif 'menu' in path.lower():
            return "菜单管理"
        elif 'api' in path.lower():
            return "API管理"
        elif 'config' in path.lower():
            return "配置管理"
        elif 'log' in path.lower() or 'audit' in path.lower():
            return "日志审计"
        elif 'security' in path.lower():
            return "安全管理"
        elif 'example' in path.lower():
            return "示例功能"
        else:
            return "其他"
    
    def _categorize_websocket(self, path: str, handler: str) -> str:
        """对WebSocket进行分类"""
        if 'nfc' in path.lower():
            if 'admin' in path.lower() or 'realtime' in path.lower():
                return "NFC管理实时数据"
            else:
                return "NFC客户端连接"
        else:
            return "实时通信"
    
    def analyze_project(self) -> None:
        """分析整个项目"""
        print("开始分析项目API和WebSocket接口...")
        
        # 分析路由文件
        router_dirs = [
            self.project_root / "router",
            self.project_root / "nfc_relay" / "router"
        ]
        
        for router_dir in router_dirs:
            if router_dir.exists():
                for go_file in router_dir.rglob("*.go"):
                    self.analyze_go_file(go_file)
        
        print(f"分析完成！发现 {len(self.api_endpoints)} 个API接口和 {len(self.websocket_endpoints)} 个WebSocket接口")
    
    def generate_report(self) -> str:
        """生成分析报告"""
        report = []
        report.append("=" * 80)
        report.append("API和WebSocket接口分析报告")
        report.append("=" * 80)
        report.append("")
        
        # API接口报告
        report.append("📡 API接口列表")
        report.append("-" * 50)
        
        # 按分类组织API
        api_by_category = defaultdict(list)
        for api in self.api_endpoints:
            api_by_category[api.category].append(api)
        
        for category, apis in sorted(api_by_category.items()):
            report.append(f"\n🔸 {category} ({len(apis)}个接口)")
            report.append("  " + "-" * 40)
            
            for api in sorted(apis, key=lambda x: x.path):
                report.append(f"  {api.method:8} {api.path}")
                if api.description:
                    report.append(f"           📝 {api.description}")
                report.append(f"           🎯 处理器: {api.handler}")
                report.append(f"           📁 文件: {os.path.basename(api.file_path)}:{api.line_number}")
                report.append("")
        
        # WebSocket接口报告
        report.append("\n🔌 WebSocket接口列表")
        report.append("-" * 50)
        
        # 按分类组织WebSocket
        ws_by_category = defaultdict(list)
        for ws in self.websocket_endpoints:
            ws_by_category[ws.category].append(ws)
        
        for category, websockets in sorted(ws_by_category.items()):
            report.append(f"\n🔸 {category} ({len(websockets)}个接口)")
            report.append("  " + "-" * 40)
            
            for ws in sorted(websockets, key=lambda x: x.path):
                report.append(f"  WS       {ws.path}")
                if ws.description:
                    report.append(f"           📝 {ws.description}")
                report.append(f"           🎯 处理器: {ws.handler}")
                report.append(f"           📁 文件: {os.path.basename(ws.file_path)}:{ws.line_number}")
                report.append("")
        
        # 统计信息
        report.append("\n📊 统计信息")
        report.append("-" * 50)
        
        # HTTP方法统计
        method_count = defaultdict(int)
        for api in self.api_endpoints:
            method_count[api.method] += 1
        
        report.append("HTTP方法分布:")
        for method, count in sorted(method_count.items()):
            report.append(f"  {method:8}: {count:3}个")
        
        # 分类统计
        report.append("\nAPI分类统计:")
        for category, count in sorted([(cat, len(apis)) for cat, apis in api_by_category.items()]):
            report.append(f"  {category:15}: {count:3}个")
        
        report.append(f"\nWebSocket分类统计:")
        for category, count in sorted([(cat, len(ws)) for cat, ws in ws_by_category.items()]):
            report.append(f"  {category:15}: {count:3}个")
        
        report.append(f"\n总计:")
        report.append(f"  API接口总数:      {len(self.api_endpoints)}个")
        report.append(f"  WebSocket接口总数: {len(self.websocket_endpoints)}个")
        report.append(f"  接口总数:         {len(self.api_endpoints) + len(self.websocket_endpoints)}个")
        
        return "\n".join(report)
    
    def export_json(self, output_file: str) -> None:
        """导出为JSON格式"""
        data = {
            "api_endpoints": [
                {
                    "method": api.method,
                    "path": api.path,
                    "handler": api.handler,
                    "description": api.description,
                    "category": api.category,
                    "file_path": api.file_path,
                    "line_number": api.line_number
                }
                for api in self.api_endpoints
            ],
            "websocket_endpoints": [
                {
                    "path": ws.path,
                    "handler": ws.handler,
                    "description": ws.description,
                    "category": ws.category,
                    "file_path": ws.file_path,
                    "line_number": ws.line_number
                }
                for ws in self.websocket_endpoints
            ],
            "summary": {
                "total_apis": len(self.api_endpoints),
                "total_websockets": len(self.websocket_endpoints),
                "total_endpoints": len(self.api_endpoints) + len(self.websocket_endpoints)
            }
        }
        
        with open(output_file, 'w', encoding='utf-8') as f:
            json.dump(data, f, ensure_ascii=False, indent=2)
        
        print(f"JSON报告已导出到: {output_file}")

def main():
    """主函数"""
    # 获取项目根目录
    current_dir = Path(__file__).parent.parent  # scripts/../
    project_root = current_dir
    
    print(f"项目根目录: {project_root}")
    
    # 创建分析器并分析项目
    analyzer = APIAnalyzer(str(project_root))
    analyzer.analyze_project()
    
    # 生成并显示报告
    report = analyzer.generate_report()
    print(report)
    
    # 保存报告到文件
    report_file = project_root / "API接口和WebSocket分析报告.txt"
    with open(report_file, 'w', encoding='utf-8') as f:
        f.write(report)
    print(f"\n报告已保存到: {report_file}")
    
    # 导出JSON格式
    json_file = project_root / "api_websocket_analysis.json"
    analyzer.export_json(str(json_file))

if __name__ == "__main__":
    main() 