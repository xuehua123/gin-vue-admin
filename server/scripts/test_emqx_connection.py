#!/usr/bin/env python3
# -*- coding: utf-8 -*-

"""
EMQX连接测试脚本
用于测试远程EMQX实例的连接性和认证
"""

import json
import time
import requests
import sys
from typing import Dict, Any

# 远程EMQX配置
EMQX_CONFIG = {
    "host": "49.235.40.39",
    "dashboard_port": 18083,
    "mqtt_port": 1883,
    "username": "admin",
    "password": "xuehua123"
}

def test_dashboard_connection() -> bool:
    """测试EMQX Dashboard连接"""
    url = f"http://{EMQX_CONFIG['host']}:{EMQX_CONFIG['dashboard_port']}"
    
    try:
        print(f"🔍 测试Dashboard连接: {url}")
        response = requests.get(url, timeout=10)
        
        if response.status_code == 200:
            print("✅ Dashboard连接成功")
            return True
        else:
            print(f"❌ Dashboard连接失败，状态码: {response.status_code}")
            return False
            
    except requests.exceptions.RequestException as e:
        print(f"❌ Dashboard连接失败: {e}")
        return False

def get_api_token() -> str:
    """获取API Token"""
    url = f"http://{EMQX_CONFIG['host']}:{EMQX_CONFIG['dashboard_port']}/api/v5/login"
    
    payload = {
        "username": EMQX_CONFIG["username"],
        "password": EMQX_CONFIG["password"]
    }
    
    try:
        print("🔍 获取API Token...")
        response = requests.post(url, json=payload, timeout=10)
        
        if response.status_code == 200:
            data = response.json()
            token = data.get("token")
            if token:
                print("✅ API Token获取成功")
                return token
            else:
                print("❌ API响应中未找到token")
                return ""
        else:
            print(f"❌ API Token获取失败，状态码: {response.status_code}")
            print(f"响应: {response.text}")
            return ""
            
    except requests.exceptions.RequestException as e:
        print(f"❌ API Token获取失败: {e}")
        return ""

def test_api_endpoints(token: str) -> Dict[str, Any]:
    """测试EMQX API端点"""
    base_url = f"http://{EMQX_CONFIG['host']}:{EMQX_CONFIG['dashboard_port']}"
    headers = {"Authorization": f"Bearer {token}"}
    
    endpoints = {
        "节点状态": "/api/v5/nodes",
        "监听器": "/api/v5/listeners", 
        "认证器": "/api/v5/authentication",
        "授权器": "/api/v5/authorization/sources",
        "客户端列表": "/api/v5/clients"
    }
    
    results = {}
    
    for name, endpoint in endpoints.items():
        try:
            print(f"🔍 测试API端点: {name}")
            url = base_url + endpoint
            response = requests.get(url, headers=headers, timeout=10)
            
            if response.status_code == 200:
                data = response.json()
                results[name] = {
                    "status": "success",
                    "data_count": len(data) if isinstance(data, list) else 1
                }
                print(f"✅ {name} API测试成功")
            else:
                results[name] = {
                    "status": "failed", 
                    "code": response.status_code
                }
                print(f"❌ {name} API测试失败，状态码: {response.status_code}")
                
        except requests.exceptions.RequestException as e:
            results[name] = {
                "status": "error",
                "error": str(e)
            }
            print(f"❌ {name} API测试异常: {e}")
    
    return results

def test_mqtt_port() -> bool:
    """测试MQTT端口连通性"""
    import socket
    
    try:
        print(f"🔍 测试MQTT端口 {EMQX_CONFIG['mqtt_port']} 连通性...")
        sock = socket.socket(socket.AF_INET, socket.SOCK_STREAM)
        sock.settimeout(5)
        
        result = sock.connect_ex((EMQX_CONFIG['host'], EMQX_CONFIG['mqtt_port']))
        sock.close()
        
        if result == 0:
            print("✅ MQTT端口连接成功")
            return True
        else:
            print("❌ MQTT端口连接失败")
            return False
            
    except Exception as e:
        print(f"❌ MQTT端口测试异常: {e}")
        return False

def generate_connection_info():
    """生成连接信息"""
    info = {
        "emqx_info": {
            "host": EMQX_CONFIG['host'],
            "dashboard_url": f"http://{EMQX_CONFIG['host']}:{EMQX_CONFIG['dashboard_port']}",
            "mqtt_tcp": f"mqtt://{EMQX_CONFIG['host']}:{EMQX_CONFIG['mqtt_port']}",
            "mqtt_ssl": f"mqtts://{EMQX_CONFIG['host']}:8883",
            "websocket": f"ws://{EMQX_CONFIG['host']}:8083",
            "websocket_ssl": f"wss://{EMQX_CONFIG['host']}:8084"
        },
        "credentials": {
            "dashboard_username": EMQX_CONFIG['username'],
            "dashboard_password": EMQX_CONFIG['password'],
            "jwt_secret": "78c0f08f-9663-4c9c-a399-cc4ec36b8112",
            "jwt_issuer": "qmPlus",
            "jwt_audience": "GVA"
        },
        "client_connection": {
            "username_field": "clientid",
            "password_field": "jwt_token",
            "authentication_method": "JWT"
        }
    }
    
    print("\n📋 EMQX连接信息:")
    print("=" * 50)
    print(json.dumps(info, indent=2, ensure_ascii=False))
    
    # 保存到文件
    try:
        with open("../config/emqx_connection_info.json", "w", encoding="utf-8") as f:
            json.dump(info, f, indent=2, ensure_ascii=False)
        print(f"\n💾 连接信息已保存到: ../config/emqx_connection_info.json")
    except Exception as e:
        print(f"\n❌ 保存连接信息失败: {e}")

def main():
    """主函数"""
    print("🚀 EMQX连接测试开始")
    print("=" * 50)
    
    # 测试Dashboard连接
    if not test_dashboard_connection():
        print("❌ Dashboard连接失败，退出测试")
        sys.exit(1)
    
    # 测试MQTT端口
    test_mqtt_port()
    
    # 获取API Token
    token = get_api_token()
    if not token:
        print("❌ 无法获取API Token，跳过API端点测试")
    else:
        # 测试API端点
        print("\n🔍 测试API端点...")
        api_results = test_api_endpoints(token)
        
        print("\n📊 API测试结果汇总:")
        for endpoint, result in api_results.items():
            status = result.get("status", "unknown")
            if status == "success":
                print(f"✅ {endpoint}: 成功")
            else:
                print(f"❌ {endpoint}: {status}")
    
    # 生成连接信息
    print("\n" + "=" * 50)
    generate_connection_info()
    
    print("\n🎉 EMQX连接测试完成！")
    print("\n💡 提示:")
    print("1. 可以使用 bash scripts/emqx_remote_setup.sh setup 配置ACL规则")
    print("2. 使用上述连接信息在客户端应用中连接EMQX")
    print("3. JWT认证需要在客户端实现，使用项目的JWT密钥生成Token")

if __name__ == "__main__":
    main() 