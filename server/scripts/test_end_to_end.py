#!/usr/bin/env python3
# -*- coding: utf-8 -*-
"""
端到端角色冲突测试脚本

模拟两个设备(A和B)之间的完整交互，测试角色冲突和挤下线通知的完整流程。
"""

import requests
import json
import time
import paho.mqtt.client as mqtt
from datetime import datetime
import threading

# --- 导入配置 ---
try:
    from config import SERVER_BASE_URL, EMQX_HOST, EMQX_MQTT_PORT, USER1_CREDENTIALS
except ImportError:
    print("❌ 无法导入配置文件 `scripts/config.py`. 请确保该文件存在且路径正确。")
    exit(1)

# --- 辅助类 ---

class TestResult:
    """用于记录测试结果"""
    def __init__(self, name):
        self.name = name
        self.success = True
        self.message = "通过"

    def set_failed(self, reason):
        self.success = False
        self.message = reason
        print(f"❌ 测试失败: {self.name} - {reason}")

class MqttClientHandler:
    """封装MQTT客户端逻辑"""
    def __init__(self, client_id):
        self.client_id = client_id
        self.is_connected = False
        self.received_message = None
        self.message_event = threading.Event()
        
        self.client = mqtt.Client(client_id=client_id, protocol=mqtt.MQTTv5)
        self.client.on_connect = self._on_connect
        self.client.on_disconnect = self._on_disconnect
        self.client.on_message = self._on_message

    def connect(self, username, password, topic_to_subscribe):
        self.client.username_pw_set(username, password)
        try:
            print(f"🔌 [{self.client_id}] 尝试连接到 {EMQX_HOST}:{EMQX_MQTT_PORT}...")
            self.client.connect(EMQX_HOST, EMQX_MQTT_PORT, 60)
            self.client.loop_start()
            
            # 等待连接成功
            time.sleep(2)
            
            if self.is_connected:
                print(f"✅ [{self.client_id}] 连接成功，订阅主题: {topic_to_subscribe}")
                self.client.subscribe(topic_to_subscribe)
                return True
            else:
                print(f"❌ [{self.client_id}] 连接超时")
                return False

        except Exception as e:
            print(f"❌ [{self.client_id}] 连接异常: {e}")
            return False
            
    def disconnect(self):
        print(f"🔌 [{self.client_id}] 断开连接...")
        self.client.loop_stop()
        self.client.disconnect()
        
    def wait_for_message(self, timeout=10):
        print(f"⏳ [{self.client_id}] 等待消息...")
        if self.message_event.wait(timeout):
            return self.received_message
        return None

    def _on_connect(self, client, userdata, flags, rc, properties=None):
        if rc == 0:
            self.is_connected = True
        else:
            print(f"❌ [{self.client_id}] 连接失败，返回码: {rc}")

    def _on_disconnect(self, client, userdata, rc, properties=None):
        self.is_connected = False
        print(f"🔌 [{self.client_id}] 已断开连接，返回码: {rc}")
        # 如果是因为被服务器踢下线（rc=141），这是一个预期的行为
        if rc == 141: # Disconnect with reason code
            self.message_event.set() # 触发事件，表示收到了预期断线


    def _on_message(self, client, userdata, msg):
        self.received_message = msg.payload.decode('utf-8')
        print(f"📨 [{self.client_id}] 收到消息 - 主题: {msg.topic}, 内容: {self.received_message}")
        self.message_event.set()

# --- 测试函数 ---

def login_and_get_auth_token():
    """登录并获取API认证Token"""
    session = requests.Session()

    # 1. 获取验证码ID
    print("🖼️  获取登录验证码...")
    captcha_id = None
    try:
        captcha_response = session.post(f"{SERVER_BASE_URL}/base/captcha", timeout=10)
        captcha_response.raise_for_status()
        captcha_data = captcha_response.json()
        if captcha_data.get("code") == 0:
            captcha_id = captcha_data["data"]["captchaId"]
            print(f"✅ 获取验证码ID成功: {captcha_id}")
        else:
            print(f"❌ 获取验证码失败: {captcha_data.get('msg')}")
            return None
    except Exception as e:
        print(f"❌ 获取验证码异常: {e}")
        return None

    # 2. 使用验证码ID进行登录
    try:
        payload = {
            "username": USER1_CREDENTIALS["username"],
            "password": USER1_CREDENTIALS["password"],
            "captcha": "",
            "captchaId": captcha_id
        }
        response = session.post(f"{SERVER_BASE_URL}/base/login", json=payload, timeout=10)
        response.raise_for_status()
        data = response.json()
        if data.get("code") == 0:
            print(f"✅ 登录成功, 用户: {USER1_CREDENTIALS['username']}")
            return data["data"]["token"]
    except requests.exceptions.RequestException as e:
        print(f"❌ 登录请求失败: {e}")
    except json.JSONDecodeError:
        print("❌ 登录失败: 无法解析服务器响应")
    return None

def get_mqtt_token(auth_token, role, force_kick=False, device_model="TestDevice"):
    """从服务器获取MQTT连接Token"""
    headers = {"x-token": auth_token, "Content-Type": "application/json"}
    payload = {
        "role": role,
        "force_kick_existing": force_kick,
        "device_info": {"device_model": device_model, "app_version": "test-1.0"}
    }
    try:
        response = requests.post(f"{SERVER_BASE_URL}/role/generateMQTTToken", headers=headers, json=payload, timeout=10)
        response.raise_for_status()
        result = response.json()
        if result.get("code") == 0:
            return result["data"]
    except requests.exceptions.RequestException as e:
        print(f"❌ 获取MQTT Token失败: {e}")
    except json.JSONDecodeError:
        print("❌ 获取MQTT Token失败: 无法解析服务器响应")
    return None

def run_end_to_end_test():
    """运行完整的端到端测试"""
    print("="*60)
    print("🚀 开始端到端角色冲突测试")
    print("="*60)
    
    auth_token = login_and_get_auth_token()
    if not auth_token:
        print("❌ 获取API认证Token失败，测试中止")
        return
        
    test_result = TestResult("端到端挤下线流程")
    client_a = None
    client_b = None
    
    try:
        # 1. 设备A获取 'transmitter' 角色并连接
        print("\n--- 步骤1: 设备A获取角色并连接 ---")
        token_a_info = get_mqtt_token(auth_token, "transmitter", device_model="Device-A")
        if not token_a_info:
            test_result.set_failed("设备A未能获取MQTT Token")
            return
            
        client_id_a = token_a_info['client_id']
        token_a = token_a_info['token']
        
        client_a = MqttClientHandler(client_id_a)
        # 订阅自己的控制主题，用于接收挤下线通知
        # 根据`notification_service.go`中的实现，挤下线通知会发布到这个主题
        revoked_topic = f"client/{client_id_a}/control/role_revoked_notification"
        
        if not client_a.connect(client_id_a, token_a, revoked_topic):
            test_result.set_failed("设备A连接MQTT失败")
            return
        
        print(f"✅ 设备A ({client_id_a}) 成功连接\n")
        time.sleep(2)
        
        # 2. 设备B强制获取 'transmitter' 角色
        print("--- 步骤2: 设备B强制获取相同角色 ---")
        token_b_info = get_mqtt_token(auth_token, "transmitter", force_kick=True, device_model="Device-B")
        if not token_b_info:
            test_result.set_failed("设备B未能强制获取MQTT Token")
            return
        
        client_id_b = token_b_info['client_id']
        if client_id_a == client_id_b:
            test_result.set_failed("设备B获取了与设备A相同的ClientID，挤下线逻辑可能未触发")
            return
        print(f"✅ 设备B ({client_id_b}) 成功获取新的MQTT Token，挤下线任务已触发\n")
        
        # 3. 验证设备A是否收到通知并被断开连接
        print("--- 步骤3: 验证设备A的状态 ---")
        
        # 等待消息或断线
        received_payload = client_a.wait_for_message(timeout=15)
        
        if received_payload:
            print(f"✅ 设备A收到通知: {received_payload}")
            # 验证通知内容
            try:
                notification = json.loads(received_payload)
                if notification.get("reason") == "role_revoked_by_peer":
                    print("✅ 通知内容正确")
                else:
                    test_result.set_failed(f"设备A收到的通知内容不正确: {notification}")
            except json.JSONDecodeError:
                test_result.set_failed(f"设备A收到的通知不是有效的JSON: {received_payload}")
        else:
            print("⚠️ 设备A未在超时时间内收到消息，检查是否被直接断开")
            
        # 验证设备A是否已被断开
        time.sleep(2) # 等待断开状态更新
        if client_a.is_connected:
            test_result.set_failed("设备A在挤下线后仍处于连接状态")
        else:
            print("✅ 设备A已按预期断开连接\n")
            
        # 4. 设备B使用新Token连接
        print("--- 步骤4: 设备B使用新Token连接 ---")
        token_b = token_b_info['token']
        client_b = MqttClientHandler(client_id_b)
        if not client_b.connect(client_id_b, token_b, f"client/{client_id_b}/#"):
            test_result.set_failed("设备B使用新Token连接MQTT失败")
            return
        
        print(f"✅ 设备B ({client_id_b}) 成功连接，测试通过！")
        
    except Exception as e:
        import traceback
        traceback.print_exc()
        test_result.set_failed(f"测试过程中出现异常: {e}")
    finally:
        if client_a:
            client_a.disconnect()
        if client_b:
            client_b.disconnect()
        
    print("\n" + "="*60)
    print("📊 端到端测试结果")
    print("="*60)
    status = "✅ 通过" if test_result.success else "❌ 失败"
    print(f"{status} - {test_result.name}: {test_result.message}")
    
    if test_result.success:
        print("\n🎉 端到端测试成功，核心功能工作正常！")
    else:
        print("\n⚠️  端到端测试失败，请检查服务器日志和EMQX配置")

if __name__ == "__main__":
    run_end_to_end_test() 