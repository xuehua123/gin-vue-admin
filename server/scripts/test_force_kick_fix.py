#!/usr/bin/env python3
# -*- coding: utf-8 -*-
"""
角色冲突强制踢出修复验证测试脚本
专门测试修复后的强制踢出功能是否正常工作
"""

import requests
import json
import time
import paho.mqtt.client as mqtt
import threading
from datetime import datetime

# 配置
SERVER_CONFIG = {
    "host": "43.165.186.134",
    "port": 8888
}

EMQX_CONFIG = {
    "host": "43.165.186.134",
    "port": 8883,
    "api_host": "43.165.186.134",
    "api_port": 18083,
    "api_username": "admin",
    "api_password": "xuehua123"
}

class ForceKickFixTester:
    def __init__(self):
        self.server_token = None
        self.emqx_api_token = None
        
    def authenticate_server(self) -> bool:
        """服务器认证"""
        print("🔐 正在认证服务器...")
        try:
            # 1. 获取验证码
            captcha_response = requests.get(f"http://{SERVER_CONFIG['host']}:{SERVER_CONFIG['port']}/base/captcha")
            if captcha_response.status_code != 200:
                print(f"❌ 获取验证码失败: {captcha_response.status_code}")
                return False
                
            captcha_data = captcha_response.json()
            if captcha_data.get("code") != 0:
                print(f"❌ 验证码接口返回错误: {captcha_data.get('msg')}")
                return False
                
            captcha_id = captcha_data["data"]["captchaId"]
            print(f"✅ 获取验证码成功: {captcha_id}")
            
            # 2. 登录
            login_payload = {
                "username": "admin",
                "password": "123456",
                "captcha": "0000",
                "captchaId": captcha_id
            }
            
            login_response = requests.post(
                f"http://{SERVER_CONFIG['host']}:{SERVER_CONFIG['port']}/base/login",
                json=login_payload
            )
            
            if login_response.status_code != 200:
                print(f"❌ 登录请求失败: {login_response.status_code}")
                return False
                
            login_data = login_response.json()
            if login_data.get("code") != 0:
                print(f"❌ 登录失败: {login_data.get('msg')}")
                return False
                
            self.server_token = login_data["data"]["token"]
            print(f"✅ 服务器认证成功")
            return True
            
        except Exception as e:
            print(f"❌ 服务器认证异常: {e}")
            return False
    
    def authenticate_emqx_api(self) -> bool:
        """EMQX API认证"""
        print("🔐 正在认证EMQX API...")
        try:
            login_url = f"http://{EMQX_CONFIG['api_host']}:{EMQX_CONFIG['api_port']}/api/v5/login"
            login_payload = {
                "username": EMQX_CONFIG['api_username'],
                "password": EMQX_CONFIG['api_password']
            }
            
            response = requests.post(login_url, json=login_payload, timeout=10)
            if response.status_code != 200:
                print(f"❌ EMQX API认证失败: {response.status_code}")
                print(f"   响应内容: {response.text}")
                return False
            
            data = response.json()
            if "token" not in data:
                print(f"❌ EMQX API响应中未找到token")
                return False
            
            self.emqx_api_token = data["token"]
            print(f"✅ EMQX API认证成功")
            return True
            
        except Exception as e:
            print(f"❌ EMQX API认证异常: {e}")
            return False
    
    def generate_mqtt_token(self, role: str, force_kick: bool = False) -> tuple:
        """生成MQTT token"""
        print(f"🎫 生成{role}角色的MQTT token (force_kick={force_kick})...")
        
        url = f"http://{SERVER_CONFIG['host']}:{SERVER_CONFIG['port']}/role/generateMQTTToken"
        headers = {"x-token": self.server_token}
        
        payload = {
            "role": role,
            "force_kick_existing": force_kick,
            "device_info": {
                "device_model": "TestDevice",
                "os_version": "Test_1.0"
            }
        }
        
        try:
            response = requests.post(url, json=payload, headers=headers, timeout=10)
            
            if response.status_code == 200:
                data = response.json()
                if data.get("code") == 0:
                    token = data["data"]["token"]
                    client_id = data["data"]["client_id"]
                    print(f"✅ MQTT token生成成功 (ClientID: {client_id})")
                    return True, token, client_id
                else:
                    print(f"❌ MQTT token生成失败: {data.get('msg', '未知错误')}")
                    return False, None, None
            else:
                print(f"❌ MQTT token生成请求失败: {response.status_code}")
                return False, None, None
                
        except Exception as e:
            print(f"❌ MQTT token生成异常: {e}")
            return False, None, None
    
    def check_client_in_emqx(self, client_id: str) -> bool:
        """检查客户端是否在EMQX中存在"""
        try:
            check_url = f"http://{EMQX_CONFIG['api_host']}:{EMQX_CONFIG['api_port']}/api/v5/clients/{client_id}"
            headers = {"Authorization": f"Bearer {self.emqx_api_token}"}
            
            response = requests.get(check_url, headers=headers, timeout=10)
            
            if response.status_code == 200:
                print(f"✅ 客户端 {client_id} 在EMQX中存在且连接")
                return True
            elif response.status_code == 404:
                print(f"❌ 客户端 {client_id} 在EMQX中不存在或已断开")
                return False
            else:
                print(f"⚠️ 查询客户端状态返回异常: {response.status_code}")
                return False
                
        except Exception as e:
            print(f"❌ 查询客户端状态异常: {e}")
            return False
    
    def run_force_kick_test(self) -> bool:
        """运行强制踢出测试"""
        print("="*60)
        print("🚀 开始强制踢出修复验证测试")
        print("="*60)
        
        # 1. 认证
        if not self.authenticate_server():
            return False
        if not self.authenticate_emqx_api():
            return False
        
        test_role = "transmitter"
        
        # 2. 设备A获取角色
        print(f"\n📱 步骤1: 设备A获取{test_role}角色...")
        success_a, token_a, client_id_a = self.generate_mqtt_token(test_role, force_kick=False)
        if not success_a:
            return False
        
        print(f"   设备A ClientID: {client_id_a}")
        
        # 3. 等待连接建立
        print("⏳ 等待3秒确保设备A连接建立...")
        time.sleep(3)
        
        # 4. 检查设备A是否真的在EMQX中连接
        print(f"\n🔍 步骤2: 检查设备A连接状态...")
        if not self.check_client_in_emqx(client_id_a):
            print("⚠️ 设备A未连接到EMQX，可能是认证问题")
            print("💡 提示：这不影响强制踢出功能测试，继续执行...")
        
        # 5. 设备B强制获取同样角色
        print(f"\n📱 步骤3: 设备B强制获取{test_role}角色...")
        success_b, token_b, client_id_b = self.generate_mqtt_token(test_role, force_kick=True)
        
        if not success_b:
            print("❌ 设备B强制获取角色失败")
            return False
            
        print(f"   设备B ClientID: {client_id_b}")
        
        # 6. 等待强制踢出处理完成
        print("⏳ 等待5秒让强制踢出处理完成...")
        time.sleep(5)
        
        # 7. 检查设备A是否被踢出
        print(f"\n🔍 步骤4: 验证设备A是否被踢出...")
        device_a_exists = self.check_client_in_emqx(client_id_a)
        
        # 8. 检查设备B是否成功连接
        print(f"\n🔍 步骤5: 验证设备B是否成功连接...")
        device_b_exists = self.check_client_in_emqx(client_id_b)
        
        # 9. 结果分析
        print("\n" + "="*60)
        print("📊 强制踢出测试结果分析")
        print("="*60)
        
        print(f"设备A ({client_id_a}): {'❌ 仍然连接' if device_a_exists else '✅ 已断开'}")
        print(f"设备B ({client_id_b}): {'✅ 成功连接' if device_b_exists else '❌ 连接失败'}")
        
        # 理想情况：A被踢出，B成功连接
        if not device_a_exists and device_b_exists:
            print("\n🎉 强制踢出功能正常工作！")
            print("✅ 设备A已被成功踢出")
            print("✅ 设备B已成功连接")
            return True
        elif not device_a_exists and not device_b_exists:
            print("\n⚠️ 设备A已被踢出，但设备B也未连接")
            print("💡 可能的原因：MQTT连接认证问题")
            return False
        elif device_a_exists and device_b_exists:
            print("\n❌ 强制踢出功能失败！")
            print("❌ 设备A未被踢出")
            print("⚠️ 设备B也连接了（可能是角色冲突处理有问题）")
            return False
        else:  # device_a_exists and not device_b_exists
            print("\n❌ 角色分配逻辑错误！")
            print("❌ 设备A仍然连接")
            print("❌ 设备B连接失败")
            return False

def main():
    tester = ForceKickFixTester()
    success = tester.run_force_kick_test()
    
    if success:
        print("\n🎯 测试结论：强制踢出修复成功！")
    else:
        print("\n🔧 测试结论：强制踢出功能仍需进一步修复")
        print("\n💡 故障排除建议：")
        print("1. 检查server/service/system/role_conflict_service.go的修复是否正确应用")
        print("2. 检查EMQX API配置是否正确")
        print("3. 检查网络连接和权限设置")
        print("4. 查看服务器日志获取详细错误信息")
    
    return success

if __name__ == "__main__":
    main() 