#!/usr/bin/env python3
# -*- coding: utf-8 -*-
"""
基础设施连通性测试脚本
测试服务器和EMQX的基本连接状态
"""

import requests
import json
import time
from datetime import datetime

# 导入配置
try:
    from config import SERVER_BASE_URL, EMQX_DASHBOARD_URL, EMQX_HOST, EMQX_MQTT_PORT
except ImportError:
    print("❌ 无法导入配置文件 `scripts/config.py`. 请确保该文件存在且路径正确。")
    exit(1)

def test_server_health():
    """测试服务器健康状态"""
    print("🔍 测试服务器连通性...")
    try:
        # 假设服务器有一个/health端点，如果没有，请更改为实际可用的端点
        response = requests.get(f"{SERVER_BASE_URL}/health", timeout=10)
        if response.status_code == 200:
            print(f"✅ 服务器连通正常 ({SERVER_BASE_URL})")
            return True
        else:
            print(f"❌ 服务器响应异常: {response.status_code} ({SERVER_BASE_URL})")
            return False
    except requests.exceptions.RequestException as e:
        print(f"❌ 服务器连接失败: {e} ({SERVER_BASE_URL})")
        return False

def test_emqx_dashboard():
    """测试EMQX控制台连通性"""
    print("🔍 测试EMQX控制台连通性...")
    try:
        response = requests.get(f"{EMQX_DASHBOARD_URL}/api/v5/status", timeout=10)
        if response.status_code == 200:
            print(f"✅ EMQX控制台连通正常 ({EMQX_DASHBOARD_URL})")
            return True
        else:
            print(f"❌ EMQX控制台响应异常: {response.status_code} ({EMQX_DASHBOARD_URL})")
            return False
    except requests.exceptions.RequestException as e:
        print(f"❌ EMQX控制台连接失败: {e} ({EMQX_DASHBOARD_URL})")
        return False

def test_emqx_mqtt_port():
    """测试EMQX MQTT端口连通性"""
    print(f"🔍 测试EMQX MQTT端口({EMQX_MQTT_PORT})连通性...")
    import socket
    try:
        sock = socket.socket(socket.AF_INET, socket.SOCK_STREAM)
        sock.settimeout(5)
        result = sock.connect_ex((EMQX_HOST, EMQX_MQTT_PORT))
        sock.close()
        
        if result == 0:
            print(f"✅ EMQX MQTT端口({EMQX_MQTT_PORT})连通正常 ({EMQX_HOST})")
            return True
        else:
            print(f"❌ EMQX MQTT端口({EMQX_MQTT_PORT})连接失败 ({EMQX_HOST})")
            return False
    except Exception as e:
        print(f"❌ EMQX MQTT端口测试异常: {e} ({EMQX_HOST})")
        return False

def run_infrastructure_tests():
    """运行所有基础设施测试"""
    print("="*60)
    print("🚀 开始基础设施连通性测试")
    print("="*60)
    
    tests = [
        ("服务器健康检查", test_server_health),
        ("EMQX控制台连通性", test_emqx_dashboard),
        ("EMQX MQTT端口连通性", test_emqx_mqtt_port),
    ]
    
    results = []
    for name, test_func in tests:
        print(f"\n📋 {name}")
        success = test_func()
        results.append((name, success))
        time.sleep(1)
    
    print("\n" + "="*60)
    print("📊 基础设施测试结果")
    print("="*60)
    
    all_passed = True
    for name, success in results:
        status = "✅ 通过" if success else "❌ 失败"
        print(f"{status} - {name}")
        if not success:
            all_passed = False
    
    if all_passed:
        print("\n🎉 基础设施测试全部通过，可以进行下一阶段测试！")
    else:
        print("\n⚠️  基础设施测试存在问题，请先解决连通性问题")
    
    return all_passed

if __name__ == "__main__":
    run_infrastructure_tests() 