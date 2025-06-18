#!/usr/bin/env python3
# -*- coding: utf-8 -*-
"""
服务器端API完整功能测试脚本
测试JWT认证、角色冲突检测、MQTT Token生成等核心功能
"""

import requests
import json
import time
from datetime import datetime
import base64

# 配置信息
SERVER_BASE = "http://43.165.186.134:8888"
TEST_USER = {
    "username": "admin",
    "password": "123456"
}

class ServerAPITester:
    def __init__(self):
        self.session = requests.Session()
        self.auth_token = None
        self.user_info = None
        
    def login(self):
        """用户登录获取JWT Token"""
        print("🔐 测试用户登录...")
        
        try:
            response = self.session.post(
                f"{SERVER_BASE}/base/login",
                json={
                    "username": TEST_USER["username"],
                    "password": TEST_USER["password"],
                    "captcha": "",
                    "captchaId": ""
                },
                timeout=10
            )
            
            if response.status_code == 200:
                data = response.json()
                if data.get("code") == 0:
                    self.auth_token = data["data"]["token"]
                    self.user_info = data["data"]["user"]
                    print(f"✅ 登录成功，用户: {self.user_info['userName']}")
                    
                    # 设置后续请求的认证头
                    self.session.headers.update({
                        "x-token": self.auth_token,
                        "Authorization": f"Bearer {self.auth_token}"
                    })
                    return True
                else:
                    print(f"❌ 登录失败: {data.get('msg', '未知错误')}")
                    return False
            else:
                print(f"❌ 登录请求失败: HTTP {response.status_code}")
                return False
                
        except Exception as e:
            print(f"❌ 登录异常: {e}")
            return False
    
    def test_jwt_generate_mqtt_token(self, role="transmitter"):
        """测试生成MQTT JWT Token"""
        print(f"🎫 测试生成MQTT Token (角色: {role})...")
        
        try:
            response = self.session.post(
                f"{SERVER_BASE}/jwt/generateMQTTToken",
                json={"role": role},
                timeout=10
            )
            
            if response.status_code == 200:
                data = response.json()
                if data.get("code") == 0:
                    token_info = data["data"]
                    print(f"✅ MQTT Token生成成功")
                    print(f"   ClientID: {token_info['client_id']}")
                    print(f"   角色: {token_info['role']}")
                    print(f"   序号: {token_info['sequence']}")
                    
                    # 解析JWT Token内容
                    self.decode_jwt_token(token_info['token'])
                    return token_info
                else:
                    print(f"❌ 生成失败: {data.get('msg', '未知错误')}")
                    return None
            else:
                print(f"❌ 请求失败: HTTP {response.status_code}")
                return None
                
        except Exception as e:
            print(f"❌ 生成异常: {e}")
            return None
    
    def test_role_conflict_detection(self, role="transmitter"):
        """测试角色冲突检测"""
        print(f"⚔️ 测试角色冲突检测 (角色: {role})...")
        
        # 先生成一个Client ID用于测试
        test_client_id = f"test-{role}-conflict-001"
        
        try:
            response = self.session.post(
                f"{SERVER_BASE}/role/checkConflict",
                json={
                    "role": role,
                    "client_id": test_client_id,
                    "device_info": {
                        "device_model": "TestDevice",
                        "app_version": "1.0.0"
                    }
                },
                timeout=10
            )
            
            if response.status_code == 200:
                data = response.json()
                if data.get("code") == 0:
                    conflict_info = data["data"]
                    if conflict_info["has_conflict"]:
                        print("⚠️ 检测到角色冲突")
                        print(f"   冲突设备: {conflict_info['conflict_device']['client_id']}")
                        print(f"   可强制挤下线: {conflict_info['can_force_kick']}")
                    else:
                        print("✅ 无角色冲突，可以正常分配")
                    return conflict_info
                else:
                    print(f"❌ 检测失败: {data.get('msg', '未知错误')}")
                    return None
            else:
                print(f"❌ 请求失败: HTTP {response.status_code}")
                return None
                
        except Exception as e:
            print(f"❌ 检测异常: {e}")
            return None
    
    def test_role_generation_with_force_kick(self, role="transmitter"):
        """测试带强制挤下线的角色分配"""
        print(f"💥 测试强制挤下线角色分配 (角色: {role})...")
        
        try:
            response = self.session.post(
                f"{SERVER_BASE}/role/generateMQTTToken",
                json={
                    "role": role,
                    "force_kick_existing": True,
                    "device_info": {
                        "device_model": "TestDevice-ForceKick",
                        "app_version": "1.0.0"
                    }
                },
                timeout=10
            )
            
            if response.status_code == 200:
                data = response.json()
                if data.get("code") == 0:
                    token_info = data["data"]
                    print(f"✅ 强制挤下线成功，获得新Token")
                    print(f"   新ClientID: {token_info['client_id']}")
                    return token_info
                else:
                    print(f"❌ 强制分配失败: {data.get('msg', '未知错误')}")
                    return None
            else:
                print(f"❌ 请求失败: HTTP {response.status_code}")
                return None
                
        except Exception as e:
            print(f"❌ 强制分配异常: {e}")
            return None
    
    def test_get_user_mqtt_tokens(self):
        """测试获取用户所有MQTT Tokens"""
        print("📋 测试获取用户所有MQTT Tokens...")
        
        try:
            response = self.session.get(
                f"{SERVER_BASE}/jwt/getUserMQTTTokens",
                timeout=10
            )
            
            if response.status_code == 200:
                data = response.json()
                if data.get("code") == 0:
                    tokens = data["data"]
                    print(f"✅ 获取成功，共有 {len(tokens)} 个活跃Token")
                    for i, token in enumerate(tokens, 1):
                        print(f"   {i}. ClientID: {token['client_id']}")
                        print(f"      角色: {token['role']}")
                        print(f"      用户名: {token['username']}")
                    return tokens
                else:
                    print(f"❌ 获取失败: {data.get('msg', '未知错误')}")
                    return None
            else:
                print(f"❌ 请求失败: HTTP {response.status_code}")
                return None
                
        except Exception as e:
            print(f"❌ 获取异常: {e}")
            return None
    
    def test_revoke_mqtt_token(self, client_id):
        """测试撤销MQTT Token"""
        print(f"🗑️ 测试撤销MQTT Token: {client_id}...")
        
        try:
            response = self.session.post(
                f"{SERVER_BASE}/jwt/revokeMQTTToken",
                json={"client_id": client_id},
                timeout=10
            )
            
            if response.status_code == 200:
                data = response.json()
                if data.get("code") == 0:
                    print("✅ Token撤销成功")
                    return True
                else:
                    print(f"❌ 撤销失败: {data.get('msg', '未知错误')}")
                    return False
            else:
                print(f"❌ 请求失败: HTTP {response.status_code}")
                return False
                
        except Exception as e:
            print(f"❌ 撤销异常: {e}")
            return False
    
    def decode_jwt_token(self, token):
        """解析JWT Token内容"""
        try:
            # JWT Token格式: header.payload.signature
            parts = token.split('.')
            if len(parts) != 3:
                print("❌ JWT Token格式错误")
                return None
            
            # 解析payload (需要补齐padding)
            payload = parts[1]
            # 补齐base64 padding
            missing_padding = len(payload) % 4
            if missing_padding:
                payload += '=' * (4 - missing_padding)
            
            decoded_payload = base64.urlsafe_b64decode(payload)
            payload_json = json.loads(decoded_payload)
            
            print("🔍 JWT Token内容:")
            print(f"   用户ID: {payload_json.get('user_id', 'N/A')}")
            print(f"   用户名: {payload_json.get('username', 'N/A')}")
            print(f"   角色: {payload_json.get('role', 'N/A')}")
            print(f"   ClientID: {payload_json.get('client_id', 'N/A')}")
            print(f"   序号: {payload_json.get('sequence', 'N/A')}")
            print(f"   过期时间: {datetime.fromtimestamp(payload_json.get('exp', 0))}")
            
            return payload_json
            
        except Exception as e:
            print(f"❌ 解析JWT Token失败: {e}")
            return None

def run_server_api_tests():
    """运行所有服务器端API测试"""
    print("="*60)
    print("🚀 开始服务器端API功能测试")
    print("="*60)
    
    tester = ServerAPITester()
    
    # 测试步骤
    test_results = []
    
    # 1. 用户登录
    if not tester.login():
        print("❌ 登录失败，无法继续后续测试")
        return False
    test_results.append(("用户登录", True))
    
    print("\n" + "-"*40)
    
    # 2. 生成transmitter角色Token
    token1 = tester.test_jwt_generate_mqtt_token("transmitter")
    test_results.append(("生成transmitter Token", token1 is not None))
    
    print("\n" + "-"*40)
    
    # 3. 生成receiver角色Token
    token2 = tester.test_jwt_generate_mqtt_token("receiver")
    test_results.append(("生成receiver Token", token2 is not None))
    
    print("\n" + "-"*40)
    
    # 4. 测试角色冲突检测
    conflict_result = tester.test_role_conflict_detection("transmitter")
    test_results.append(("角色冲突检测", conflict_result is not None))
    
    print("\n" + "-"*40)
    
    # 5. 测试强制挤下线
    force_kick_result = tester.test_role_generation_with_force_kick("transmitter")
    test_results.append(("强制挤下线", force_kick_result is not None))
    
    print("\n" + "-"*40)
    
    # 6. 获取用户所有Token
    user_tokens = tester.test_get_user_mqtt_tokens()
    test_results.append(("获取用户Tokens", user_tokens is not None))
    
    print("\n" + "-"*40)
    
    # 7. 撤销Token (如果有的话)
    if user_tokens and len(user_tokens) > 0:
        revoke_result = tester.test_revoke_mqtt_token(user_tokens[0]['client_id'])
        test_results.append(("撤销Token", revoke_result))
    
    # 输出测试结果
    print("\n" + "="*60)
    print("📊 服务器端API测试结果")
    print("="*60)
    
    all_passed = True
    for name, success in test_results:
        status = "✅ 通过" if success else "❌ 失败"
        print(f"{status} - {name}")
        if not success:
            all_passed = False
    
    if all_passed:
        print("\n🎉 服务器端API测试全部通过！")
        print("💡 接下来可以进行EMQX集成测试")
    else:
        print("\n⚠️  服务器端API测试存在问题，请检查服务器状态")
    
    return all_passed

if __name__ == "__main__":
    run_server_api_tests() 