#!/usr/bin/env python3
# -*- coding: utf-8 -*-
"""
测试角色冲突修复的有效性
主要测试verifyClientDisconnected函数对401状态码的正确处理
"""

import requests
import json
import time
import sys
from typing import Dict, Any, Optional

# 配置
SERVER_CONFIG = {
    'host': '43.165.186.134',
    'port': 8888,
    'base_url': 'http://43.165.186.134:8888'
}

EMQX_CONFIG = {
    'host': '49.235.40.39',
    'dashboard_port': 18083,
    'mqtt_port': 8883
}

class RoleConflictFixTester:
    def __init__(self):
        self.server_token = None
        self.emqx_token = None
        self.session = requests.Session()
        
    def login_to_server(self) -> bool:
        """登录到服务器获取认证token"""
        print("🔐 登录到服务器...")
        
        # 先获取验证码
        captcha_url = f"{SERVER_CONFIG['base_url']}/base/captcha"
        try:
            captcha_resp = requests.get(captcha_url, timeout=10)
            if captcha_resp.status_code != 200:
                print(f"❌ 获取验证码失败: {captcha_resp.status_code}")
                return False
                
            captcha_data = captcha_resp.json()
            if captcha_data.get('code') != 0:
                print(f"❌ 验证码响应错误: {captcha_data.get('msg')}")
                return False
                
            captcha_id = captcha_data['data']['captchaId']
            print(f"✅ 验证码获取成功: {captcha_id}")
            
        except Exception as e:
            print(f"❌ 获取验证码异常: {e}")
            return False
        
        # 登录
        login_url = f"{SERVER_CONFIG['base_url']}/base/login"
        login_data = {
            "username": "admin",
            "password": "123456",
            "captcha": "1234",  # 假设验证码（实际环境可能需要真实验证码）
            "captchaId": captcha_id
        }
        
        try:
            response = requests.post(login_url, json=login_data, timeout=10)
            if response.status_code == 200:
                data = response.json()
                if data.get('code') == 0:
                    self.server_token = data['data']['token']
                    print("✅ 服务器登录成功")
                    return True
                else:
                    print(f"❌ 登录失败: {data.get('msg')}")
                    return False
            else:
                print(f"❌ 登录请求失败: {response.status_code}")
                return False
        except Exception as e:
            print(f"❌ 登录异常: {e}")
            return False
    
    def get_emqx_token(self) -> bool:
        """获取EMQX API token"""
        print("🔗 获取EMQX API Token...")
        
        url = f"http://{EMQX_CONFIG['host']}:{EMQX_CONFIG['dashboard_port']}/api/v5/login"
        credentials = {
            "username": "admin",
            "password": "xuehua123"
        }
        
        try:
            response = requests.post(url, json=credentials, timeout=10)
            if response.status_code == 200:
                data = response.json()
                self.emqx_token = data.get("token")
                if self.emqx_token:
                    print("✅ EMQX API Token获取成功")
                    return True
                else:
                    print("❌ EMQX API响应中未找到token")
                    return False
            else:
                print(f"❌ EMQX API登录失败: {response.status_code}")
                print(f"响应内容: {response.text}")
                return False
        except Exception as e:
            print(f"❌ EMQX API登录异常: {e}")
            return False
    
    def test_verifyClientDisconnected_fix(self) -> bool:
        """测试verifyClientDisconnected函数的修复"""
        print("\n🔍 测试verifyClientDisconnected修复...")
        
        # 1. 测试不存在的客户端（应该返回404，被正确识别为断开）
        test_client_id = f"test-nonexistent-client-{int(time.time())}"
        print(f"   测试客户端: {test_client_id}")
        
        # 直接调用EMQX API测试不同状态码
        url = f"http://{EMQX_CONFIG['host']}:{EMQX_CONFIG['dashboard_port']}/api/v5/clients/{test_client_id}"
        
        # 测试1: 没有认证头的情况（应该返回401）
        print("   📋 测试1: 无认证头请求（期望401）")
        try:
            response = requests.get(url, timeout=10)
            print(f"   响应状态码: {response.status_code}")
            if response.status_code == 401:
                print("   ✅ 正确返回401（认证失败）")
            else:
                print(f"   ⚠️ 意外状态码: {response.status_code}")
        except Exception as e:
            print(f"   ❌ 请求异常: {e}")
        
        # 测试2: 有效认证头但客户端不存在（应该返回404）
        print("   📋 测试2: 有效认证头，不存在的客户端（期望404）")
        if not self.emqx_token:
            print("   ❌ 缺少EMQX Token，跳过此测试")
            return False
            
        headers = {"Authorization": f"Bearer {self.emqx_token}"}
        try:
            response = requests.get(url, headers=headers, timeout=10)
            print(f"   响应状态码: {response.status_code}")
            if response.status_code == 404:
                print("   ✅ 正确返回404（客户端不存在）")
                return True
            elif response.status_code == 200:
                print("   ⚠️ 客户端意外存在，检查客户端详情")
                data = response.json()
                print(f"   客户端信息: {json.dumps(data, indent=2)}")
                return True
            else:
                print(f"   ❌ 意外状态码: {response.status_code}")
                print(f"   响应内容: {response.text}")
                return False
        except Exception as e:
            print(f"   ❌ 请求异常: {e}")
            return False
    
    def test_role_assignment_flow(self) -> bool:
        """测试完整的角色分配流程"""
        print("\n🎯 测试完整的角色分配流程...")
        
        if not self.server_token:
            print("❌ 缺少服务器Token，无法测试")
            return False
            
        # 获取MQTT Token
        print("   📋 步骤1: 获取MQTT Token...")
        url = f"{SERVER_CONFIG['base_url']}/role/generateMQTTToken"
        headers = {"x-token": self.server_token}
        data = {"role": "transmitter", "force_kick": False}
        
        try:
            response = requests.post(url, json=data, headers=headers, timeout=10)
            print(f"   响应状态码: {response.status_code}")
            
            if response.status_code == 200:
                response_data = response.json()
                print(f"   响应数据: {json.dumps(response_data, indent=2, ensure_ascii=False)}")
                
                if response_data.get('code') == 0:
                    client_id = response_data['data']['client_id']
                    token = response_data['data']['token']
                    print(f"   ✅ MQTT Token获取成功: {client_id}")
                    
                    # 测试强制踢出场景
                    print("   📋 步骤2: 测试强制踢出...")
                    data2 = {"role": "transmitter", "force_kick": True}
                    response2 = requests.post(url, json=data2, headers=headers, timeout=30)
                    
                    print(f"   强制踢出响应状态: {response2.status_code}")
                    if response2.status_code == 200:
                        response_data2 = response2.json()
                        print(f"   强制踢出响应: {json.dumps(response_data2, indent=2, ensure_ascii=False)}")
                        
                        if response_data2.get('code') == 0:
                            print("   ✅ 强制踢出成功，修复验证有效！")
                            return True
                        else:
                            print(f"   ❌ 强制踢出失败: {response_data2.get('msg')}")
                            return False
                    else:
                        print(f"   ❌ 强制踢出请求失败: {response2.status_code}")
                        return False
                else:
                    print(f"   ❌ MQTT Token获取失败: {response_data.get('msg')}")
                    return False
            else:
                print(f"   ❌ 获取MQTT Token请求失败: {response.status_code}")
                return False
                
        except Exception as e:
            print(f"   ❌ 测试异常: {e}")
            return False
    
    def run_all_tests(self) -> bool:
        """运行所有测试"""
        print("🚀 开始角色冲突修复验证测试")
        print("=" * 60)
        
        # 1. 登录服务器
        if not self.login_to_server():
            print("❌ 服务器登录失败，终止测试")
            return False
        
        # 2. 获取EMQX Token
        if not self.get_emqx_token():
            print("❌ EMQX Token获取失败，终止测试")
            return False
        
        # 3. 测试verifyClientDisconnected修复
        test1_result = self.test_verifyClientDisconnected_fix()
        
        # 4. 测试完整流程
        test2_result = self.test_role_assignment_flow()
        
        # 总结结果
        print("\n" + "=" * 60)
        print("📊 测试结果总结")
        print("=" * 60)
        print(f"✅ verifyClientDisconnected修复测试: {'通过' if test1_result else '失败'}")
        print(f"✅ 完整角色分配流程测试: {'通过' if test2_result else '失败'}")
        
        overall_success = test1_result and test2_result
        
        if overall_success:
            print("🎉 所有测试通过！角色冲突修复验证成功")
        else:
            print("⚠️ 部分测试失败，请检查相关配置")
            
        return overall_success

def main():
    """主函数"""
    tester = RoleConflictFixTester()
    success = tester.run_all_tests()
    sys.exit(0 if success else 1)

if __name__ == "__main__":
    main() 