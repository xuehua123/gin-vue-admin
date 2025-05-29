#!/usr/bin/env python3
# -*- coding: utf-8 -*-
"""
快速API和WebSocket接口摘要脚本
快速生成接口摘要

作者: API分析脚本
日期: 2025年
"""

def print_api_summary():
    """打印API和WebSocket接口摘要"""
    
    print("🚀 Gin-Vue-Admin NFC中继系统 - API和WebSocket接口摘要")
    print("=" * 80)
    
    # API分类摘要
    api_categories = {
        "🏢 系统管理API (51个)": [
            "用户登录: POST /api/base/login",
            "获取验证码: POST /api/base/captcha", 
            "用户管理: POST /api/user/* (注册、修改密码、权限设置等)",
            "权限管理: POST /api/authority/* (创建、删除、更新角色)",
            "菜单管理: POST /api/menu/* (菜单增删改查)",
            "API管理: POST /api/api/* (API增删改查)",
            "字典管理: GET|POST|PUT|DELETE /api/sysDictionary/*",
            "操作记录: GET|DELETE /api/sysOperationRecord/*",
            "系统配置: POST /api/system/* (获取服务器信息、配置)",
            "数据库: POST /api/api (初始化数据库)",
            "健康检查: GET /api/health"
        ],
        
        "🔌 NFC中继管理API (61个)": [
            "仪表盘数据: GET /api/admin/nfc-relay/v1/dashboard-stats-enhanced",
            "性能指标: GET /api/admin/nfc-relay/v1/performance-metrics",
            "地理分布: GET /api/admin/nfc-relay/v1/geographic-distribution",
            "告警管理: GET|POST /api/admin/nfc-relay/v1/alerts/*",
            "客户端管理: GET|POST /api/admin/nfc-relay/v1/clients/*",
            "会话管理: GET|POST /api/admin/nfc-relay/v1/sessions/*",
            "审计日志: GET|POST|DELETE /api/admin/nfc-relay/v1/audit-logs*",
            "安全管理: GET|POST|PUT /api/admin/nfc-relay/v1/security/*",
            "安全配置: GET|PUT|POST /api/admin/nfc-relay/v1/security/config*",
            "系统配置: GET /api/admin/nfc-relay/v1/config",
            "数据导出: POST /api/admin/nfc-relay/v1/export",
            "",
            "🆕 新增功能 (30个新接口):",
            "加密验证API (3个): 解密验证、批量处理、状态查询",
            "配置热重载API (6个): 配置重载、状态监控、历史记录",
            "合规规则管理API (9个): 规则CRUD、测试、文件导入导出",
            "配置变更审计API (6个): 审计日志、统计分析、变更追踪",
            "安全配置API (6个): 安全配置管理、合规统计、功能测试"
        ]
    }
    
    # WebSocket摘要
    websocket_endpoints = {
        "🌐 WebSocket接口 (4个)": [
            "NFC客户端连接: ws://host:port/ws/nfc-relay/client",
            "管理端实时数据: ws://host:port/ws/nfc-relay/realtime", 
            "管理后台实时推送: ws://host:port/api/admin/nfc-relay/v1/realtime",
            "实时数据传输: ws://host:port/nfc-relay/realtime"
        ]
    }
    
    # 打印API摘要
    print("\n📡 API接口摘要")
    print("-" * 60)
    
    for category, apis in api_categories.items():
        print(f"\n{category}")
        for api in apis:
            if api:  # 跳过空字符串
                print(f"  • {api}")
    
    # 打印WebSocket摘要
    print(f"\n{list(websocket_endpoints.keys())[0]}")
    print("-" * 60)
    for ws in websocket_endpoints["🌐 WebSocket接口 (4个)"]:
        print(f"  • {ws}")
    
    # 技术规范
    print("\n📋 技术规范")
    print("-" * 60)
    print("• 基础路径: /api/")
    print("• 认证方式: JWT Token (Authorization: Bearer <token>)")
    print("• 数据格式: JSON")
    print("• 字符编码: UTF-8")
    print("• WebSocket协议: 支持ping/pong心跳")
    print("• 安全特性: TLS/SSL、RBAC权限控制、审计日志")
    
    # 统计信息
    print("\n📊 统计信息")
    print("-" * 60)
    print("• API接口总数: 112个 (原82个 + 新增30个)")
    print("• WebSocket接口总数: 4个")
    print("• HTTP方法分布: GET(31.3%), POST(59.8%), PUT(7.1%), DELETE(1.8%)")
    print("• 主要功能: 系统管理(45.5%) + NFC中继管理(54.5%)")
    
    print("\n🎯 新增接口详情")
    print("-" * 60)
    print("• 加密验证API: 支持接收端解密验证、批量处理、状态监控")
    print("• 配置热重载API: 支持动态配置重载、版本回滚、变更历史")
    print("• 合规规则管理API: 支持规则CRUD、测试验证、文件导入导出")
    print("• 配置变更审计API: 支持变更追踪、统计分析、审计日志导出")
    print("• 安全配置API: 支持安全配置管理、合规统计、功能测试")
    
    print("\n" + "=" * 80)
    print("📖 详细文档请查看: API接口文档.md")
    print("📄 完整报告请查看: API接口总结-完整版.md")

if __name__ == "__main__":
    print_api_summary() 