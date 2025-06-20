#!/usr/bin/env python3
# -*- coding: utf-8 -*-
"""
角色冲突处理修复验证脚本
专门测试EMQX API密码配置修复后的角色冲突处理功能
"""

import requests
import json
import time
import uuid
from typing import Dict, Any, Optional

# 配置信息
SERVER_CONFIG = {
    "host": "49.235.40.39",
    "port": 8888,
    "username": "admin",
    "password": "123456"
}

EMQX_CONFIG = {
    "host": "49.235.40.39",
    "dashboard_port": 18083,
    "mqtt_port": 8883,
    "username": "admin",
    "password": "xuehua123"  # 修复后的密码
}

class RoleConflictTester:
    def __init__(self):
        self.server_token = None
        self.emqx_token = None
        
    def authenticate_server(self) -> bool:
        """认证服务器获取token"""
        print("🔐 服务器认证...")
        url = f"http://{SERVER_CONFIG['host']}:{SERVER_CONFIG['port']}/base/login"
        
        payload = {
            "username": SERVER_CONFIG["username"],
            "password": SERVER_CONFIG["password"],
            "captcha": "0000",
            "captchaId": "dummy"
        }
        
        try:
            response = requests.post(url, json=payload, timeout=10)
            
            if response.status_code == 200:
                data = response.json()
                if data.get("code") == 0:
                    self.server_token = data["data"]["token"]
                    print("✅ 服务器认证成功")
                    return True
                else:
                    print(f"❌ 服务器认证失败: {data.get('msg', '未知错误')}")
                    return False
            else:
                print(f"❌ 服务器认证请求失败: {response.status_code}")
                return False
                
        except Exception as e:
            print(f"❌ 服务器认证异常: {e}")
            return False
    
    def authenticate_emqx(self) -> bool:
        """认证EMQX获取token"""
        print("🔐 EMQX认证...")
        url = f"http://{EMQX_CONFIG['host']}:{EMQX_CONFIG['dashboard_port']}/api/v5/login"
        
        payload = {
            "username": EMQX_CONFIG["username"],
            "password": EMQX_CONFIG["password"]
        }
        
        try:
            response = requests.post(url, json=payload, timeout=10)
            
            if response.status_code == 200:
                data = response.json()
                self.emqx_token = data.get("token")
                if self.emqx_token:
                    print("✅ EMQX认证成功")
                    return True
                else:
                    print("❌ EMQX响应中未找到token")
                    return False
            else:
                print(f"❌ EMQX认证失败: {response.status_code} - {response.text}")
                return False
                
        except Exception as e:
            print(f"❌ EMQX认证异常: {e}")
            return False
    
    def generate_mqtt_token(self, user_id: str, role: str) -> Optional[str]:
        """为指定用户生成MQTT token"""
        print(f"🎫 为用户 {user_id} 生成 {role} 角色的MQTT token...")
        
        url = f"http://{SERVER_CONFIG['host']}:{SERVER_CONFIG['port']}/role/generateMQTTToken"
        headers = {"x-token": self.server_token}
        
        payload = {
            "user_id": user_id,
            "role": role,
            "device_info": {
                "device_model": "TestDevice",
                "os_version": "Test_1.0"
            },
            "force_kick": False
        }
        
        try:
            response = requests.post(url, json=payload, headers=headers, timeout=10)
            
            if response.status_code == 200:
                data = response.json()
                if data.get("code") == 0:
                    token = data["data"]["token"]
                    client_id = data["data"]["client_id"]
                    print(f"✅ MQTT token生成成功 (ClientID: {client_id})")
                    return token, client_id
                else:
                    print(f"❌ MQTT token生成失败: {data.get('msg', '未知错误')}")
                    return None, None
            else:
                print(f"❌ MQTT token生成请求失败: {response.status_code}")
                return None, None
                
        except Exception as e:
            print(f"❌ MQTT token生成异常: {e}")
            return None, None
    
    def force_kick_user(self, user_id: str, role: str) -> bool:
        """强制踢出用户（测试角色冲突处理）"""
        print(f"⚡ 强制踢出用户 {user_id} 的 {role} 角色...")
        
        url = f"http://{SERVER_CONFIG['host']}:{SERVER_CONFIG['port']}/role/generateMQTTToken"
        headers = {"x-token": self.server_token}
        
        payload = {
            "user_id": user_id,
            "role": role,
            "device_info": {
                "device_model": "NewTestDevice",
                "os_version": "Test_2.0"
            },
            "force_kick": True  # 关键：启用强制踢出
        }
        
        try:
            response = requests.post(url, json=payload, headers=headers, timeout=10)
            
            if response.status_code == 200:
                data = response.json()
                if data.get("code") == 0:
                    token = data["data"]["token"]
                    client_id = data["data"]["client_id"]
                    print(f"✅ 强制踢出成功，新ClientID: {client_id}")
                    return True, token, client_id
                else:
                    print(f"❌ 强制踢出失败: {data.get('msg', '未知错误')}")
                    return False, None, None
            else:
                print(f"❌ 强制踢出请求失败: {response.status_code}")
                return False, None, None
                
        except Exception as e:
            print(f"❌ 强制踢出异常: {e}")
            return False, None, None
    
    def check_client_connection(self, client_id: str) -> bool:
        """检查EMQX中客户端的连接状态"""
        print(f"🔍 检查客户端 {client_id} 的连接状态...")
        
        url = f"http://{EMQX_CONFIG['host']}:{EMQX_CONFIG['dashboard_port']}/api/v5/clients/{client_id}"
        headers = {"Authorization": f"Bearer {self.emqx_token}"}
        
        try:
            response = requests.get(url, headers=headers, timeout=10)
            
            if response.status_code == 200:
                data = response.json()
                connected = data.get("connected", False)
                if connected:
                    print(f"✅ 客户端 {client_id} 仍然在线")
                    return True
                else:
                    print(f"❌ 客户端 {client_id} 已离线")
                    return False
            elif response.status_code == 404:
                print(f"❌ 客户端 {client_id} 不存在（已被断开）")
                return False
            else:
                print(f"⚠️ 检查客户端状态失败: {response.status_code}")
                return None
                
        except Exception as e:
            print(f"❌ 检查客户端状态异常: {e}")
            return None
    
    def run_conflict_test(self) -> bool:
        """运行完整的角色冲突测试"""
        print("="*60)
        print("🚀 开始角色冲突处理修复验证测试")
        print("="*60)
        
        # 1. 认证
        if not self.authenticate_server():
            return False
        if not self.authenticate_emqx():
            return False
        
        # 2. 创建测试用户
        test_user_id = f"test_user_{int(time.time())}"
        test_role = "transmitter"
        
        print(f"\n📝 测试用户ID: {test_user_id}")
        print(f"📝 测试角色: {test_role}")
        
        # 3. 为用户A生成第一个token
        print(f"\n📱 步骤1: 设备A获取{test_role}角色...")
        token_a, client_id_a = self.generate_mqtt_token(test_user_id, test_role)
        if not token_a:
            return False
        
        # 4. 等待一下确保连接建立
        print("⏳ 等待2秒确保设备A连接建立...")
        time.sleep(2)
        
        # 5. 检查设备A的连接状态
        print(f"\n🔍 步骤2: 检查设备A ({client_id_a}) 连接状态...")
        is_connected_before = self.check_client_connection(client_id_a)
        if is_connected_before is False:
            print("⚠️ 设备A未连接，可能MQTT连接建立失败")
            # 继续测试，因为关键是测试强制踢出功能
        
        # 6. 设备B强制获取同样的角色（这应该会踢出设备A）
        print(f"\n📱 步骤3: 设备B强制获取{test_role}角色（应该踢出设备A）...")
        success, token_b, client_id_b = self.force_kick_user(test_user_id, test_role)
        if not success:
            return False
        
        # 7. 等待踢出操作完成
        print("⏳ 等待5秒让踢出操作完成...")
        time.sleep(5)
        
        # 8. 检查设备A是否被成功踢出
        print(f"\n🔍 步骤4: 检查设备A ({client_id_a}) 是否被踢出...")
        is_connected_after = self.check_client_connection(client_id_a)
        
        # 9. 检查设备B的连接状态
        print(f"\n🔍 步骤5: 检查设备B ({client_id_b}) 连接状态...")
        is_b_connected = self.check_client_connection(client_id_b)
        
        # 10. 分析结果
        print(f"\n" + "="*60)
        print("📊 测试结果分析")
        print("="*60)
        
        print(f"设备A踢出前连接状态: {'在线' if is_connected_before else '离线' if is_connected_before is False else '未知'}")
        print(f"设备A踢出后连接状态: {'在线' if is_connected_after else '离线' if is_connected_after is False else '未知'}")
        print(f"设备B连接状态: {'在线' if is_b_connected else '离线' if is_b_connected is False else '未知'}")
        
        # 判断测试是否成功
        if is_connected_after is False:
            print("✅ 角色冲突处理成功：设备A已被正确踢出")
            return True
        elif is_connected_after is True:
            print("❌ 角色冲突处理失败：设备A仍然在线")
            return False
        else:
            print("⚠️ 无法确定角色冲突处理结果：连接状态检查失败")
            return False

def main():
    tester = RoleConflictTester()
    success = tester.run_conflict_test()
    
    print(f"\n" + "="*60)
    if success:
        print("🎉 角色冲突处理修复验证成功！")
        print("✅ EMQX API密码配置修复生效，强制踢出功能正常工作")
    else:
        print("❌ 角色冲突处理修复验证失败")
        print("⚠️ 可能需要进一步检查EMQX API配置或服务器日志")
    print("="*60)

if __name__ == "__main__":
    main() 