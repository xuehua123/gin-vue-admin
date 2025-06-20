#!/usr/bin/env python3
# -*- coding: utf-8 -*-
"""
EMQX API修复验证脚本
测试修复后的配置是否能正确调用EMQX API
"""

import requests
import json
import time

# 配置信息 - 使用修复后的配置
EMQX_CONFIG = {
    "host": "49.235.40.39",
    "dashboard_port": 18083,
    "username": "admin",
    "password": "xuehua123"  # 修复后的密码
}

def test_emqx_api_login():
    """测试EMQX API登录"""
    print("🔍 测试EMQX API登录...")
    url = f"http://{EMQX_CONFIG['host']}:{EMQX_CONFIG['dashboard_port']}/api/v5/login"
    
    payload = {
        "username": EMQX_CONFIG["username"],
        "password": EMQX_CONFIG["password"]
    }
    
    try:
        response = requests.post(url, json=payload, timeout=10)
        
        print(f"请求URL: {url}")
        print(f"请求载荷: {json.dumps(payload, indent=2)}")
        print(f"响应状态: {response.status_code}")
        
        if response.status_code == 200:
            data = response.json()
            print(f"响应数据: {json.dumps(data, indent=2)}")
            
            token = data.get("token")
            if token:
                print("✅ EMQX API登录成功")
                return token
            else:
                print("❌ API响应中未找到token")
                return None
        else:
            print(f"❌ API登录失败，状态码: {response.status_code}")
            print(f"响应内容: {response.text}")
            return None
            
    except requests.exceptions.RequestException as e:
        print(f"❌ API请求失败: {e}")
        return None

def test_client_disconnect(token, client_id="test-client-123"):
    """测试客户端断开连接API"""
    print(f"\n🔌 测试客户端断开连接API (ClientID: {client_id})...")
    
    url = f"http://{EMQX_CONFIG['host']}:{EMQX_CONFIG['dashboard_port']}/api/v5/clients/{client_id}"
    headers = {
        "Authorization": f"Bearer {token}",
        "Content-Type": "application/json"
    }
    
    try:
        response = requests.delete(url, headers=headers, timeout=10)
        
        print(f"请求URL: {url}")
        print(f"请求头: {headers}")
        print(f"响应状态: {response.status_code}")
        
        if response.status_code == 200 or response.status_code == 204:
            print("✅ 客户端断开连接API调用成功")
            return True
        elif response.status_code == 404:
            print("⚠️ 客户端不存在（这是正常的，因为我们使用的是测试ClientID）")
            return True
        else:
            print(f"❌ 客户端断开连接API调用失败")
            print(f"响应内容: {response.text}")
            return False
            
    except requests.exceptions.RequestException as e:
        print(f"❌ API请求失败: {e}")
        return False

def test_list_clients(token):
    """测试获取客户端列表API"""
    print(f"\n📋 测试获取客户端列表API...")
    
    url = f"http://{EMQX_CONFIG['host']}:{EMQX_CONFIG['dashboard_port']}/api/v5/clients"
    headers = {
        "Authorization": f"Bearer {token}",
        "Content-Type": "application/json"
    }
    
    try:
        response = requests.get(url, headers=headers, timeout=10)
        
        print(f"请求URL: {url}")
        print(f"响应状态: {response.status_code}")
        
        if response.status_code == 200:
            data = response.json()
            client_count = len(data.get("data", []))
            print(f"✅ 获取客户端列表成功，当前连接客户端数量: {client_count}")
            
            # 打印前3个客户端信息
            if client_count > 0:
                print("前3个客户端信息:")
                for i, client in enumerate(data["data"][:3]):
                    print(f"  {i+1}. ClientID: {client.get('clientid', 'N/A')}")
                    print(f"     连接状态: {client.get('connected', 'N/A')}")
                    print(f"     连接时间: {client.get('connected_at', 'N/A')}")
            
            return True
        else:
            print(f"❌ 获取客户端列表失败")
            print(f"响应内容: {response.text}")
            return False
            
    except requests.exceptions.RequestException as e:
        print(f"❌ API请求失败: {e}")
        return False

def main():
    print("="*60)
    print("🚀 EMQX API修复验证测试")
    print("="*60)
    
    # 1. 测试API登录
    token = test_emqx_api_login()
    if not token:
        print("\n❌ API登录失败，无法继续测试")
        return
    
    # 2. 测试客户端断开连接API
    disconnect_success = test_client_disconnect(token)
    
    # 3. 测试获取客户端列表API
    list_success = test_list_clients(token)
    
    # 结果汇总
    print("\n" + "="*60)
    print("📊 测试结果汇总")
    print("="*60)
    
    tests = [
        ("EMQX API登录", token is not None),
        ("客户端断开连接API", disconnect_success),
        ("获取客户端列表API", list_success)
    ]
    
    all_passed = True
    for name, success in tests:
        status = "✅ 通过" if success else "❌ 失败"
        print(f"{status} - {name}")
        if not success:
            all_passed = False
    
    if all_passed:
        print("\n🎉 所有EMQX API测试通过！配置修复成功！")
    else:
        print("\n⚠️ 部分EMQX API测试失败，请检查配置")

if __name__ == "__main__":
    main() 