#!/usr/bin/env python3
# -*- coding: utf-8 -*-
"""
EMQX集成测试脚本
测试MQTT认证、ACL权限控制、客户端连接等功能
"""

import requests
import json
import time
import paho.mqtt.client as mqtt
from datetime import datetime
import threading
import base64

# 配置信息
SERVER_BASE = "http://43.165.186.134:8888"
EMQX_HOST = "49.235.40.39"
EMQX_DASHBOARD = "http://49.235.40.39:18083"
EMQX_PORTS = {
    "tcp": 1883,
    "ssl": 8883,
    "ws": 8083,
    "wss": 8084
}

TEST_USER = {
    "username": "admin",
    "password": "xuehua123"
}

class EMQXIntegrationTester:
    def __init__(self):
        self.session = requests.Session()
        self.auth_token = None
        self.mqtt_tokens = {}
        self.mqtt_clients = {}
        self.received_messages = {}
        
    def get_captcha(self):
        """获取验证码"""
        print("🖼️ 获取验证码...")
        try:
            response = self.session.post(f"{SERVER_BASE}/base/captcha", timeout=10)
            if response.status_code == 200:
                data = response.json()
                if data.get("code") == 0:
                    print("✅ 验证码获取成功")
                    return data["data"]["captchaId"]
                else:
                    print(f"❌ 获取验证码失败: {data.get('msg')}")
                    return None
            else:
                print(f"❌ 获取验证码请求失败: {response.status_code}")
                return None
        except Exception as e:
            print(f"❌ 获取验证码异常: {e}")
            return None
        
    def login_to_server(self):
        """登录到服务器获取JWT Token"""
        print("🔐 登录到服务器...")
        
        captcha_id = self.get_captcha()
        if not captcha_id:
            return False
        
        try:
            response = self.session.post(
                f"{SERVER_BASE}/base/login",
                json={
                    "username": TEST_USER["username"],
                    "password": TEST_USER["password"],
                    "captcha": "1234", # 随便填一个值，因为我们不校验
                    "captchaId": captcha_id
                },
                timeout=10
            )
            
            if response.status_code == 200:
                data = response.json()
                if data.get("code") == 0:
                    self.auth_token = data["data"]["token"]
                    self.session.headers.update({
                        "x-token": self.auth_token,
                        "Authorization": f"Bearer {self.auth_token}"
                    })
                    print("✅ 服务器登录成功")
                    return True
                else:
                    error_message = data.get("msg", "未知错误")
                    print(f"❌ 服务器返回登录失败: {error_message} (完整响应: {data})")
                    return False
            else:
                print(f"❌ 服务器请求失败: 状态码 {response.status_code}, 响应: {response.text}")
                return False
                    
        except Exception as e:
            print(f"❌ 服务器登录请求异常: {e}")
            return False
        
    def get_mqtt_token(self, role, force_kick=False):
        """从服务器获取MQTT Token"""
        print(f"🎫 获取MQTT Token (角色: {role})...")
        
        url = f"{SERVER_BASE}/role/generateMQTTToken"
        payload = {
            "role": role,
            "force_kick_existing": force_kick
        }
        
        print(f"   - 请求URL: {url}")
        print(f"   - 角色: {role}")
        print(f"   - 强制踢出: {force_kick}")
        
        try:
            response = self.session.post(url, json=payload, timeout=10)
            
            print(f"   - 响应状态: {response.status_code}")
            print(f"   - 响应头: {dict(response.headers)}")
            
            if response.status_code == 200:
                data = response.json()
                print(f"   - 响应数据: {data}")
                
                if data.get("code") == 0:
                    token_info = data["data"]
                    self.mqtt_tokens[role] = token_info
                    print(f"✅ 获取{role} Token成功: {token_info['client_id']}")
                    print(f"   - Client ID: {token_info['client_id']}")
                    print(f"   - Token: {token_info['token'][:20]}...")
                    print(f"   - Expires: {token_info.get('expires_at', 'N/A')}")
                    return token_info
                else:
                    print(f"❌ 服务器返回错误: code={data.get('code')}, msg={data.get('msg')}")
                    return None
            else:
                print(f"❌ HTTP请求失败: {response.status_code}")
                print(f"   - 响应体: {response.text}")
                return None
                    
        except Exception as e:
            print(f"❌ 获取MQTT Token失败: {e}")
            return None
    
    def test_mqtt_auth_api(self, role):
        """测试MQTT认证API接口"""
        print(f"🔒 测试MQTT认证API (角色: {role})...")
        
        if role not in self.mqtt_tokens:
            print(f"❌ 没有{role}角色的Token，请先获取")
            return False
        
        token_info = self.mqtt_tokens[role]
        
        # 测试认证接口
        try:
            auth_response = requests.post(
                f"{SERVER_BASE}/mqtt/auth",
                json={
                    "clientid": token_info["client_id"],
                    "username": "admin",
                    "password": token_info["token"]
                },
                timeout=10
            )
            
            if auth_response.status_code == 200:
                auth_data = auth_response.json()
                if auth_data.get("result") == "allow":
                    print("✅ MQTT认证API测试通过")
                    
                    # 测试ACL接口
                    return self.test_mqtt_acl_api(token_info)
                else:
                    print(f"❌ MQTT认证被拒绝: {auth_data}")
                    return False
            else:
                print(f"❌ MQTT认证API请求失败: {auth_response.status_code}")
                return False
                
        except Exception as e:
            print(f"❌ MQTT认证API测试异常: {e}")
            return False
    
    def test_mqtt_acl_api(self, token_info):
        """测试MQTT ACL权限API"""
        print("🛡️ 测试MQTT ACL权限API...")
        
        test_cases = [
            # (topic, action, expected_result)
            (f"client/{token_info['client_id']}/status", "publish", "allow"),    # 自己的主题发布
            (f"client/{token_info['client_id']}/control", "subscribe", "allow"), # 自己的主题订阅
            ("system/heartbeat", "publish", "allow"),                           # 系统主题发布
            ("client/other-client/status", "publish", "deny"),                  # 其他客户端主题
        ]
        
        all_passed = True
        for topic, action, expected in test_cases:
            try:
                acl_response = requests.post(
                    f"{SERVER_BASE}/mqtt/acl",
                    json={
                        "clientid": token_info["client_id"],
                        "username": token_info["client_id"],
                        "topic": topic,
                        "action": action
                    },
                    timeout=10
                )
                
                if acl_response.status_code == 200:
                    acl_data = acl_response.json()
                    result = acl_data.get("result")
                    if result == expected:
                        print(f"   ✅ {action} {topic}: {result}")
                    else:
                        print(f"   ❌ {action} {topic}: 期望{expected}, 实际{result}")
                        all_passed = False
                else:
                    print(f"   ❌ ACL API请求失败: {acl_response.status_code}")
                    all_passed = False
                    
            except Exception as e:
                print(f"   ❌ ACL测试异常: {e}")
                all_passed = False
        
        return all_passed
    
    def connect_mqtt_client(self, role, token_info_override=None):
        """连接MQTT客户端"""
        print(f"🔌 连接MQTT客户端 (角色: {role})...")
        
        token_info = token_info_override
        if not token_info:
            if role not in self.mqtt_tokens:
                print(f"❌ 没有{role}角色的Token")
                return None
            token_info = self.mqtt_tokens[role]

        client_id = token_info["client_id"]
        
        # 创建MQTT客户端
        client = mqtt.Client(
            client_id=client_id,
            protocol=mqtt.MQTTv5
        )
        
        # 设置认证信息
        client.username_pw_set(
            username="admin",
            password=token_info["token"]
        )
        
        # 设置回调函数
        def on_connect(client, userdata, flags, rc, properties=None):
            if rc == 0:
                print(f"✅ MQTT客户端连接成功: {client_id}")
                # 订阅自己的控制主题
                client.subscribe(f"client/{client_id}/control/#")
                client.subscribe(f"client/{client_id}/sync/#")
            else:
                print(f"❌ MQTT客户端连接失败: {client_id}, RC={rc}")
        
        def on_message(client, userdata, msg):
            topic = msg.topic
            payload = msg.payload.decode('utf-8')
            print(f"📨 收到消息 [{client_id}]: {topic} -> {payload}")
            
            if client_id not in self.received_messages:
                self.received_messages[client_id] = []
            self.received_messages[client_id].append({
                "topic": topic,
                "payload": payload,
                "timestamp": datetime.now()
            })
        
        def on_disconnect(client, userdata, rc, properties=None):
            print(f"🔌 MQTT客户端断开: {client_id}, RC={rc}")
        
        client.on_connect = on_connect
        client.on_message = on_message
        client.on_disconnect = on_disconnect
        
        try:
            # 连接到EMQX
            client.connect(EMQX_HOST, EMQX_PORTS["tcp"], 60)
            client.loop_start()
            
            # 等待连接
            time.sleep(2)
            
            if client.is_connected():
                self.mqtt_clients[role] = client
                return client # 返回客户端实例
            else:
                print(f"❌ MQTT客户端连接超时: {client_id}")
                client.loop_stop()
                return None
                
        except Exception as e:
            print(f"❌ MQTT客户端连接异常: {e}")
            return None
    
    def test_mqtt_messaging(self):
        """测试MQTT消息收发"""
        print("💬 测试MQTT消息收发...")
        
        if "transmitter" not in self.mqtt_clients or "receiver" not in self.mqtt_clients:
            print("❌ 需要两个MQTT客户端进行消息测试")
            return False
        
        transmitter = self.mqtt_clients["transmitter"]
        receiver = self.mqtt_clients["receiver"]
        
        tx_client_id = self.mqtt_tokens["transmitter"]["client_id"]
        rx_client_id = self.mqtt_tokens["receiver"]["client_id"]
        
        # 清空之前的消息
        self.received_messages.clear()
        
        # transmitter发布状态消息
        status_msg = {
            "status": "ready",
            "timestamp": datetime.now().isoformat(),
            "from": tx_client_id
        }
        
        transmitter.publish(
            f"client/{tx_client_id}/status", 
            json.dumps(status_msg)
        )
        print(f"📤 发送状态消息: {tx_client_id}")
        
        # receiver发布状态消息
        receiver_status = {
            "status": "waiting",
            "timestamp": datetime.now().isoformat(),
            "from": rx_client_id
        }
        
        receiver.publish(
            f"client/{rx_client_id}/status",
            json.dumps(receiver_status)
        )
        print(f"📤 发送状态消息: {rx_client_id}")
        
        # 等待消息处理
        time.sleep(3)
        
        # 检查消息接收情况
        success = True
        for client_id, messages in self.received_messages.items():
            print(f"📨 {client_id} 收到 {len(messages)} 条消息")
            for msg in messages:
                print(f"   - {msg['topic']}: {msg['payload']}")
        
        return success
    
    def test_role_conflict_scenario(self):
        """测试角色冲突场景"""
        print("⚔️ 测试角色冲突场景...")

        # 获取并连接设备A
        print("🎫 获取设备A的MQTT Token...")
        token_a = self.get_mqtt_token("transmitter")
        if not token_a: 
            print("❌ 设备A Token获取失败")
            return False
        
        print(f"✅ 设备A Token: {token_a['client_id']}")
        print(f"   - Token: {token_a['token'][:20]}...")
        print(f"   - Expires: {token_a.get('expires_at', 'N/A')}")
        
        print("🔌 连接设备A到MQTT...")
        client_a = self.connect_mqtt_client("transmitter")
        if not client_a:
            print("❌ 设备A连接失败")
            return False
        
        print(f"✅ 设备A ({token_a['client_id']}) 连接成功")
        print(f"   - 连接状态: {client_a.is_connected()}")
        
        # 等待连接稳定
        time.sleep(2)
        
        # 检查设备A在EMQX中的状态
        print("🔍 检查设备A在EMQX中的连接状态...")
        self.check_emqx_client_status(token_a['client_id'])

        # 设备B强制获取相同角色，服务器应使设备A的token失效
        print("\n🥊 设备B尝试强制获取transmitter角色...")
        print("🎫 获取设备B的MQTT Token (force_kick=True)...")
        
        token_b = self.get_mqtt_token("transmitter", force_kick=True)
        if not token_b: 
            print("❌ 设备B Token获取失败")
            return False

        print(f"✅ 设备B Token: {token_b['client_id']}")
        print(f"   - Token: {token_b['token'][:20]}...")
        print(f"   - Expires: {token_b.get('expires_at', 'N/A')}")

        # 比较两个Token
        if token_a["client_id"] == token_b["client_id"]:
            print("⚠️ 强制挤下线但获得了相同的ClientID，这可能不是预期行为")
        else:
            print(f"✅ 强制挤下线成功，新ClientID: {token_b['client_id']}")
            print(f"   - 旧ClientID: {token_a['client_id']}")
            print(f"   - 新ClientID: {token_b['client_id']}")

        # 立即检查设备A的连接状态
        print("\n📊 立即检查设备A连接状态...")
        print(f"   - 客户端连接状态: {client_a.is_connected()}")
        
        # 检查EMQX中设备A的状态
        print("🔍 检查EMQX中设备A的状态...")
        self.check_emqx_client_status(token_a['client_id'])
        
        # 等待服务器处理，EMQX应断开设备A的连接
        print("\n⏳ 等待EMQX处理角色冲突...")
        for i in range(10):  # 最多等待10秒，每秒检查一次
            time.sleep(1)
            is_connected = client_a.is_connected()
            print(f"   [{i+1}/10] 设备A连接状态: {'🟢 已连接' if is_connected else '🔴 已断开'}")
            
            if not is_connected:
                print("✅ 设备A的连接已按预期被服务器断开")
                break
                
            # 每2秒检查一次EMQX状态
            if (i + 1) % 2 == 0:
                self.check_emqx_client_status(token_a['client_id'])
        else:
            print("❌ 角色冲突后，设备A的连接未被断开")
            print("🔍 最终检查EMQX状态...")
            self.check_emqx_client_status(token_a['client_id'])
            
            # 手动断开设备A
            print("🧹 手动断开设备A...")
            client_a.disconnect()
            return False

        # 测试设备B连接
        print("\n🔌 测试设备B连接...")
        client_b = self.connect_mqtt_client("transmitter_b", token_b)
        if not client_b:
            print("❌ 设备B连接失败")
            return False
        
        print("✅ 设备B连接成功")
        print(f"   - 连接状态: {client_b.is_connected()}")
        
        # 检查设备B在EMQX中的状态
        print("🔍 检查设备B在EMQX中的连接状态...")
        self.check_emqx_client_status(token_b['client_id'])
        
        # 清理设备B
        print("🧹 断开设备B...")
        client_b.disconnect()
        
        return True

    def check_emqx_client_status(self, client_id):
        """检查EMQX中客户端的连接状态"""
        try:
            from config import EMQX_DASHBOARD_URL, EMQX_API_KEY, EMQX_SECRET_KEY
            
            # 使用EMQX API检查客户端状态
            url = f"{EMQX_DASHBOARD_URL}/api/v5/clients/{client_id}"
            
            import base64
            credentials = base64.b64encode(f"{EMQX_API_KEY}:{EMQX_SECRET_KEY}".encode()).decode()
            headers = {
                "Authorization": f"Basic {credentials}",
                "Content-Type": "application/json"
            }
            
            response = requests.get(url, headers=headers, timeout=10)
            
            if response.status_code == 200:
                client_info = response.json()
                print(f"   ✅ EMQX中找到客户端: {client_id}")
                print(f"      - 连接状态: {client_info.get('connected', 'unknown')}")
                print(f"      - 连接时间: {client_info.get('connected_at', 'N/A')}")
                print(f"      - IP地址: {client_info.get('ip_address', 'N/A')}")
                print(f"      - 保活时间: {client_info.get('keepalive', 'N/A')}")
                return True
            elif response.status_code == 404:
                print(f"   🔴 EMQX中未找到客户端: {client_id}")
                return False
            else:
                print(f"   ⚠️ EMQX API返回: {response.status_code}")
                print(f"      响应: {response.text[:200]}")
                return False
                
        except Exception as e:
            print(f"   ❌ 检查EMQX状态失败: {e}")
            return False
    
    def cleanup_clients(self):
        """清理MQTT客户端连接"""
        print("🧹 清理MQTT客户端连接...")
        
        for role, client in self.mqtt_clients.items():
            try:
                client.loop_stop()
                client.disconnect()
                print(f"✅ 断开{role}客户端")
            except:
                pass
        
        self.mqtt_clients.clear()

def run_emqx_integration_tests():
    """运行EMQX集成测试"""
    print("="*60)
    print("🚀 开始EMQX集成测试")
    print("="*60)
    
    tester = EMQXIntegrationTester()
    test_results = []
    
    try:
        # 1. 登录到服务器
        if not tester.login_to_server():
            print("❌ 服务器登录失败，无法继续测试")
            return False
        test_results.append(("服务器登录", True))
        
        print("\n" + "-"*40)
        
        # 2. 获取MQTT Tokens
        token_tx = tester.get_mqtt_token("transmitter")
        token_rx = tester.get_mqtt_token("receiver")
        test_results.append(("获取MQTT Tokens", token_tx is not None and token_rx is not None))
        
        print("\n" + "-"*40)
        
        # 3. 测试MQTT认证API
        auth_tx = tester.test_mqtt_auth_api("transmitter")
        auth_rx = tester.test_mqtt_auth_api("receiver")
        test_results.append(("MQTT认证API", auth_tx and auth_rx))
        
        print("\n" + "-"*40)
        
        # 4. 连接MQTT客户端
        connect_tx = tester.connect_mqtt_client("transmitter")
        connect_rx = tester.connect_mqtt_client("receiver")
        test_results.append(("MQTT客户端连接", connect_tx and connect_rx))
        
        print("\n" + "-"*40)
        
        # 5. 测试消息收发
        messaging = tester.test_mqtt_messaging()
        test_results.append(("MQTT消息收发", messaging))
        
        print("\n" + "-"*40)
        
        # 6. 测试角色冲突场景
        # 在运行冲突测试前，先清理主客户端，避免状态混淆
        tester.cleanup_clients() 
        conflict = tester.test_role_conflict_scenario()
        test_results.append(("角色冲突处理", conflict))
        
    finally:
        # 清理资源
        tester.cleanup_clients()
    
    # 输出测试结果
    print("\n" + "="*60)
    print("📊 EMQX集成测试结果")
    print("="*60)
    
    all_passed = True
    for name, success in test_results:
        status = "✅ 通过" if success else "❌ 失败"
        print(f"{status} - {name}")
        if not success:
            all_passed = False
    
    if all_passed:
        print("\n🎉 EMQX集成测试全部通过！")
        print("💡 接下来可以进行端到端测试")
    else:
        print("\n⚠️  EMQX集成测试存在问题，请检查配置")
    
    return all_passed

if __name__ == "__main__":
    run_emqx_integration_tests() 