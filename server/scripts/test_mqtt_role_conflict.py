import requests
import paho.mqtt.client as mqtt
import time
import json
import threading
import sys

# --- 配置 ---
BASE_URL = "http://localhost:8888"
MQTT_HOST = "localhost"
MQTT_PORT = 1883
# 请确保该测试用户在您的数据库中存在
TEST_USERNAME = "admin"
TEST_PASSWORD = "123456"

# 全局变量用于存储获取到的Token
AUTH_TOKEN = None

class TestResult:
    def __init__(self, name):
        self.name = name
        self.success = True
        self.message = "OK"

    def set_failed(self, message):
        self.success = False
        self.message = message

    def __str__(self):
        status = "✅ SUCCESS" if self.success else "❌ FAILED"
        return f"[{status}] Test: {self.name}: {self.message}"

class MqttClientHandler:
    """一个简单的MQTT客户端处理器，用于连接和监听消息"""
    def __init__(self, client_id):
        self.client_id = client_id
        self.client = mqtt.Client(client_id=client_id)
        self.client.on_connect = self.on_connect
        self.client.on_message = self.on_message
        self.received_message = None
        self.is_connected = False
        self.lock = threading.Lock()
        self.message_event = threading.Event()

    def on_connect(self, client, userdata, flags, rc):
        if rc == 0:
            self.is_connected = True
            print(f"MQTT Client {self.client_id} connected successfully.")
        else:
            print(f"MQTT Client {self.client_id} failed to connect, return code {rc}")

    def on_message(self, client, userdata, msg):
        with self.lock:
            self.received_message = msg.payload.decode()
            print(f"\nClient {self.client_id} received message on topic '{msg.topic}': {self.received_message}")
            self.message_event.set() # 通知已收到消息

    def connect_and_subscribe(self, username, token, topic):
        self.client.username_pw_set(username, token)
        try:
            self.client.connect(MQTT_HOST, MQTT_PORT, 60)
            self.client.subscribe(topic)
            threading.Thread(target=self.client.loop_forever, daemon=True).start()
            time.sleep(1) # 等待连接建立
        except Exception as e:
            self.is_connected = False
            print(f"Error connecting {self.client_id}: {e}")

    def wait_for_message(self, timeout=15):
        """等待消息，直到超时"""
        return self.message_event.wait(timeout)

    def disconnect(self):
        if self.is_connected:
            self.client.disconnect()
            self.is_connected = False
            print(f"MQTT Client {self.client_id} disconnected.")

def login_and_get_token():
    """登录并获取x-token"""
    global AUTH_TOKEN
    if AUTH_TOKEN:
        return AUTH_TOKEN

    url = f"{BASE_URL}/base/login"
    payload = {
        "username": TEST_USERNAME,
        "password": TEST_PASSWORD,
        "captcha": "", # 通常本地测试不需要验证码
        "captchaId": ""
    }
    headers = {"Content-Type": "application/json"}
    try:
        print(f"Attempting to log in as user '{TEST_USERNAME}'...")
        response = requests.post(url, headers=headers, json=payload)
        response.raise_for_status()
        data = response.json()
        if data.get('code') == 0 and 'token' in data['data']:
            AUTH_TOKEN = data['data']['token']
            print(f"Login successful. Token acquired.")
            return AUTH_TOKEN
        else:
            print(f"Login failed: {data.get('msg')}")
            return None
    except requests.exceptions.RequestException as e:
        print(f"Error during login: {e}")
        return None

def get_mqtt_token(role, force_kick=False, device_model="TestDevice"):
    """向后端请求MQTT Token"""
    token = login_and_get_token()
    if not token:
        raise Exception("Could not get auth token. Halting test.")

    url = f"{BASE_URL}/jwt/generateMQTTToken"
    payload = {
        "role": role,
        "force_kick_existing": force_kick,
        "device_info": { "model": device_model }
    }
    headers = {
        "Content-Type": "application/json",
        "x-token": token
    }
    try:
        response = requests.post(url, headers=headers, json=payload)
        response.raise_for_status()
        return response.json()
    except requests.exceptions.RequestException as e:
        print(f"Error getting MQTT token: {e}")
        return None

def run_test(test_func):
    """测试运行器"""
    result = TestResult(test_func.__name__)
    try:
        test_func(result)
    except Exception as e:
        result.set_failed(f"An unexpected error occurred: {e}")
    print(str(result))
    return result.success

def test_normal_role_assignment(result: TestResult):
    """测试场景1: 正常角色分配，无冲突"""
    print("\n--- Running Test: Normal Role Assignment ---")
    
    # 1. 设备A获取 'transmitter' 角色
    resp_a = get_mqtt_token("transmitter", device_model="Device-A")
    if not resp_a or resp_a.get('code') != 0:
        result.set_failed(f"Failed to get token for Device-A. Response: {resp_a}")
        return
    
    client_id_a = resp_a['data']['client_id']
    token_a = resp_a['data']['token']

    client_a = MqttClientHandler(client_id_a)
    client_a.connect_and_subscribe(TEST_USERNAME, token_a, f"client/{client_id_a}/#")

    if not client_a.is_connected:
        result.set_failed("Device-A MQTT client failed to connect.")
        client_a.disconnect()
        return

    # 2. 设备B获取 'receiver' 角色
    resp_b = get_mqtt_token("receiver", device_model="Device-B")
    if not resp_b or resp_b.get('code') != 0:
        result.set_failed(f"Failed to get token for Device-B. Response: {resp_b}")
        client_a.disconnect()
        return

    client_id_b = resp_b['data']['client_id']
    token_b = resp_b['data']['token']
    
    client_b = MqttClientHandler(client_id_b)
    client_b.connect_and_subscribe(TEST_USERNAME, token_b, f"client/{client_id_b}/#")

    if not client_b.is_connected:
        result.set_failed("Device-B MQTT client failed to connect.")
        client_a.disconnect()
        client_b.disconnect()
        return

    result.message = "Device-A and Device-B assigned roles and connected successfully."
    client_a.disconnect()
    client_b.disconnect()

def test_role_conflict_and_force_kick(result: TestResult):
    """测试场景2: 角色冲突和强制挤下线"""
    print("\n--- Running Test: Role Conflict and Force Kick ---")
    
    # 1. 设备A获取 'transmitter' 角色并连接
    resp_a = get_mqtt_token("transmitter", device_model="Device-A")
    if not resp_a or resp_a.get('code') != 0:
        result.set_failed(f"Failed to get token for Device-A. Response: {resp_a}")
        return

    client_id_a = resp_a['data']['client_id']
    token_a = resp_a['data']['token']
    
    client_a = MqttClientHandler(client_id_a)
    revoked_topic = f"client/{client_id_a}/control/role_revoked_notification"
    client_a.connect_and_subscribe(TEST_USERNAME, token_a, revoked_topic)

    if not client_a.is_connected:
        result.set_failed("Device-A MQTT client failed to connect.")
        client_a.disconnect()
        return

    # 2. 设备C强制获取 'transmitter' 角色
    print("Device-C attempting to force-kick Device-A...")
    resp_c = get_mqtt_token("transmitter", force_kick=True, device_model="Device-C")
    if not resp_c or resp_c.get('code') != 0:
        result.set_failed(f"Device-C failed to get token with force_kick. Response: {resp_c}")
        client_a.disconnect()
        return
    
    # 3. 验证设备A收到了下线通知
    print("Waiting for kick notification on Device-A...")
    message_received = client_a.wait_for_message(timeout=10)

    if not message_received:
        result.set_failed("Device-A did not receive kick notification within timeout.")
    else:
        try:
            msg_data = json.loads(client_a.received_message)
            if msg_data.get('revoked_role') == 'transmitter':
                 result.message = "Device-C successfully kicked Device-A and Device-A received notification."
            else:
                result.set_failed(f"Received notification with wrong payload: {msg_data}")
        except json.JSONDecodeError:
            result.set_failed(f"Failed to decode JSON from kick notification: {client_a.received_message}")
            
    client_a.disconnect()


if __name__ == "__main__":
    print("Starting MQTT interaction test suite...")
    print(f"Will use user '{TEST_USERNAME}' for authentication.")
    print("Please ensure the backend server, Redis, and EMQX are running.")
    print("You may need to run 'pip install paho-mqtt requests'")
    
    # 首先尝试登录
    if not login_and_get_token():
        print("\n🔥 Login failed. Cannot proceed with tests. Please check credentials and server status.")
        sys.exit(1)

    tests_passed = True
    tests_passed &= run_test(test_normal_role_assignment)
    time.sleep(2) # 等待连接清理
    tests_passed &= run_test(test_role_conflict_and_force_kick)

    print("\n--- Test Suite Finished ---")
    if tests_passed:
        print("🎉 All tests passed successfully!")
    else:
        print("🔥 Some tests failed. Please review the logs above.") 