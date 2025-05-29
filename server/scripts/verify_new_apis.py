#!/usr/bin/env python3
# -*- coding: utf-8 -*-
"""
验证新增API接口脚本
检查24个新增接口是否正确注册

作者: API分析脚本
日期: 2025年
"""

import os
import re

def verify_new_apis():
    """验证新增的24个API接口"""
    
    print("🔍 验证新增API接口注册状态")
    print("=" * 80)
    
    # 定义要验证的新增接口
    new_apis = {
        "加密验证API": [
            'nfcRelayAdminRouter.POST("encryption/decrypt-verify"',
            'nfcRelayAdminRouter.POST("encryption/batch-decrypt-verify"',
            'nfcRelayAdminRouter.GET("encryption/status"'
        ],
        "配置热重载API": [
            'nfcRelayAdminRouter.POST("config/reload"',
            'nfcRelayAdminRouter.GET("config/status"',
            'nfcRelayAdminRouter.GET("config/hot-reload-status"',
            'nfcRelayAdminRouter.POST("config/hot-reload/toggle"',
            'nfcRelayAdminRouter.POST("config/revert/:config_type"',
            'nfcRelayAdminRouter.GET("config/history/:config_type"'
        ],
        "合规规则管理API": [
            'nfcRelayAdminRouter.GET("compliance/rules"',
            'nfcRelayAdminRouter.GET("compliance/rules/:rule_id"',
            'nfcRelayAdminRouter.POST("compliance/rules"',
            'nfcRelayAdminRouter.PUT("compliance/rules/:rule_id"',
            'nfcRelayAdminRouter.DELETE("compliance/rules/:rule_id"',
            'nfcRelayAdminRouter.POST("compliance/rules/test"',
            'nfcRelayAdminRouter.GET("compliance/rule-files"',
            'nfcRelayAdminRouter.POST("compliance/rule-files/import"',
            'nfcRelayAdminRouter.GET("compliance/rule-files/export"'
        ],
        "配置变更审计API": [
            'nfcRelayAdminRouter.GET("config-audit/logs"',
            'nfcRelayAdminRouter.GET("config-audit/stats"',
            'nfcRelayAdminRouter.GET("config-audit/changes/:change_id"',
            'nfcRelayAdminRouter.POST("config-audit/records"',
            'nfcRelayAdminRouter.GET("config-audit/export"'
        ]
    }
    
    # 验证路由文件
    router_file = "../router/nfc_relay_admin/nfc_relay_admin.go"
    if not os.path.exists(router_file):
        print("❌ 路由文件不存在!")
        return False
    
    # 读取路由文件内容
    with open(router_file, 'r', encoding='utf-8') as f:
        router_content = f.read()
    
    print("📁 检查文件: router/nfc_relay_admin/nfc_relay_admin.go")
    print("-" * 60)
    
    total_apis = 0
    verified_apis = 0
    
    for category, apis in new_apis.items():
        print(f"\n🔧 {category} ({len(apis)}个接口):")
        category_verified = 0
        
        for api_route in apis:
            total_apis += 1
            if api_route in router_content:
                print(f"  ✅ {api_route}")
                verified_apis += 1
                category_verified += 1
            else:
                print(f"  ❌ {api_route}")
        
        print(f"  📊 {category}: {category_verified}/{len(apis)} 已注册")
    
    # 验证API组注册
    print("\n🔧 API组注册验证:")
    api_group_file = "../api/v1/nfc_relay_admin/enter.go"
    if os.path.exists(api_group_file):
        with open(api_group_file, 'r', encoding='utf-8') as f:
            api_group_content = f.read()
        
        required_apis = [
            "EncryptionVerificationApi",
            "ConfigReloadApi", 
            "ComplianceRulesApi",
            "ConfigAuditApi"
        ]
        
        for api_type in required_apis:
            if api_type in api_group_content:
                print(f"  ✅ {api_type} 已注册")
            else:
                print(f"  ❌ {api_type} 未注册")
    
    # 验证API文件存在性
    print("\n📂 API文件存在性验证:")
    api_files = [
        "../api/v1/nfc_relay_admin/encryption_verification.go",
        "../api/v1/nfc_relay_admin/config_reload.go",
        "../api/v1/nfc_relay_admin/compliance_rules.go",
        "../api/v1/nfc_relay_admin/config_audit.go"
    ]
    
    for api_file in api_files:
        if os.path.exists(api_file):
            print(f"  ✅ {os.path.basename(api_file)} 存在")
        else:
            print(f"  ❌ {os.path.basename(api_file)} 不存在")
    
    # 总结
    print("\n" + "=" * 80)
    print("📊 验证结果总结:")
    print(f"• 总接口数: {total_apis}")
    print(f"• 已注册接口: {verified_apis}")
    print(f"• 注册成功率: {(verified_apis/total_apis)*100:.1f}%")
    
    if verified_apis == total_apis:
        print("🎉 所有新增API接口已成功注册!")
        return True
    else:
        print(f"⚠️  有 {total_apis - verified_apis} 个接口未正确注册")
        return False

if __name__ == "__main__":
    verify_new_apis() 