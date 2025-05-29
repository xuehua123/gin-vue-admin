#!/usr/bin/env python3
# -*- coding: utf-8 -*-
"""
分析未注册API接口脚本
扫描所有已实现但未在路由中注册的API接口

作者: API分析脚本
日期: 2025年
"""

import os
import re
import glob

def extract_api_functions():
    """提取所有API处理函数"""
    api_functions = {}
    
    # 搜索所有API文件
    api_files = glob.glob("../api/v1/**/*.go", recursive=True)
    
    for file_path in api_files:
        try:
            with open(file_path, 'r', encoding='utf-8') as f:
                content = f.read()
                
            # 提取API结构体名 - 支持Api和API结尾
            struct_match = re.search(r'type\s+(\w+(?:Api|API))\s+struct', content)
            if not struct_match:
                continue
                
            api_struct = struct_match.group(1)
            
            # 提取所有处理函数
            func_pattern = r'func\s+\([^)]+\s+\*' + api_struct + r'\)\s+(\w+)\s*\([^)]*\*gin\.Context\)'
            functions = re.findall(func_pattern, content)
            
            if functions:
                relative_path = file_path.replace("../", "").replace("\\", "/")
                api_functions[api_struct] = {
                    'file': relative_path,
                    'functions': functions
                }
                
        except Exception as e:
            print(f"处理文件 {file_path} 时出错: {e}")
    
    return api_functions

def extract_router_registrations():
    """提取所有路由注册"""
    registered_routes = set()
    
    # 搜索所有路由文件
    router_files = glob.glob("../router/**/*.go", recursive=True)
    
    for file_path in router_files:
        try:
            with open(file_path, 'r', encoding='utf-8') as f:
                content = f.read()
                
            # 提取路由注册模式 - 包括复合API模式
            route_patterns = [
                # 简单模式: api.FunctionName
                r'\.GET\s*\(\s*"[^"]*"\s*,\s*\w+\.(\w+)\s*\)',
                r'\.POST\s*\(\s*"[^"]*"\s*,\s*\w+\.(\w+)\s*\)',
                r'\.PUT\s*\(\s*"[^"]*"\s*,\s*\w+\.(\w+)\s*\)',
                r'\.DELETE\s*\(\s*"[^"]*"\s*,\s*\w+\.(\w+)\s*\)',
                r'\.PATCH\s*\(\s*"[^"]*"\s*,\s*\w+\.(\w+)\s*\)',
                # 复合模式: apiGroup.SubApi.FunctionName
                r'\.GET\s*\(\s*"[^"]*"\s*,\s*\w+\.\w+\.(\w+)\s*\)',
                r'\.POST\s*\(\s*"[^"]*"\s*,\s*\w+\.\w+\.(\w+)\s*\)',
                r'\.PUT\s*\(\s*"[^"]*"\s*,\s*\w+\.\w+\.(\w+)\s*\)',
                r'\.DELETE\s*\(\s*"[^"]*"\s*,\s*\w+\.\w+\.(\w+)\s*\)',
                r'\.PATCH\s*\(\s*"[^"]*"\s*,\s*\w+\.\w+\.(\w+)\s*\)',
                # 更复杂的模式: apiGroup.SubApi.SubSubApi.FunctionName
                r'\.GET\s*\(\s*"[^"]*"\s*,\s*\w+\.\w+\.\w+\.(\w+)\s*\)',
                r'\.POST\s*\(\s*"[^"]*"\s*,\s*\w+\.\w+\.\w+\.(\w+)\s*\)',
                r'\.PUT\s*\(\s*"[^"]*"\s*,\s*\w+\.\w+\.\w+\.(\w+)\s*\)',
                r'\.DELETE\s*\(\s*"[^"]*"\s*,\s*\w+\.\w+\.\w+\.(\w+)\s*\)',
                r'\.PATCH\s*\(\s*"[^"]*"\s*,\s*\w+\.\w+\.\w+\.(\w+)\s*\)',
            ]
            
            for pattern in route_patterns:
                matches = re.findall(pattern, content)
                registered_routes.update(matches)
                
        except Exception as e:
            print(f"处理路由文件 {file_path} 时出错: {e}")
    
    return registered_routes

def analyze_unregistered_apis():
    """分析未注册的API"""
    
    print("🔍 分析未注册的API接口")
    print("=" * 80)
    
    # 获取所有API函数
    api_functions = extract_api_functions()
    print(f"📁 发现 {len(api_functions)} 个API结构体")
    
    # 获取所有已注册的路由
    registered_routes = extract_router_registrations()
    print(f"🔗 发现 {len(registered_routes)} 个已注册的路由")
    
    print("\n📊 详细分析结果:")
    print("-" * 60)
    
    total_functions = 0
    total_unregistered = 0
    unregistered_apis = {}
    
    for api_struct, info in api_functions.items():
        functions = info['functions']
        total_functions += len(functions)
        
        unregistered = []
        for func in functions:
            if func not in registered_routes:
                unregistered.append(func)
                total_unregistered += 1
        
        if unregistered:
            unregistered_apis[api_struct] = {
                'file': info['file'],
                'unregistered': unregistered,
                'total': len(functions)
            }
            
        print(f"\n🔧 {api_struct} ({info['file']}):")
        print(f"   总函数: {len(functions)}, 未注册: {len(unregistered)}")
        
        if unregistered:
            for func in unregistered:
                print(f"   ❌ {func}")
        else:
            print(f"   ✅ 所有函数已注册")
    
    # 生成修复建议
    print("\n" + "=" * 80)
    print("🛠️  修复建议:")
    print("-" * 60)
    
    if total_unregistered == 0:
        print("🎉 所有API接口都已正确注册!")
    else:
        print(f"⚠️  发现 {total_unregistered} 个未注册的API函数")
        print("建议按以下步骤修复:")
        print("1. 检查API组注册 (api/v1/*/enter.go)")
        print("2. 添加路由配置 (router/*/)")
        print("3. 验证编译无误")
        
        # 按类别分组建议
        if unregistered_apis:
            print("\n📋 需要修复的API:")
            for api_struct, info in unregistered_apis.items():
                print(f"\n• {api_struct}:")
                print(f"  文件: {info['file']}")
                print(f"  未注册函数: {', '.join(info['unregistered'])}")
    
    print(f"\n📊 统计总结:")
    print(f"• API结构体总数: {len(api_functions)}")
    print(f"• API函数总数: {total_functions}")
    print(f"• 已注册函数: {total_functions - total_unregistered}")
    print(f"• 未注册函数: {total_unregistered}")
    print(f"• 注册完成率: {((total_functions - total_unregistered) / total_functions * 100):.1f}%")
    
    return unregistered_apis

if __name__ == "__main__":
    analyze_unregistered_apis() 