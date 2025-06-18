# -*- coding: utf-8 -*-
import requests
import json
import time
import datetime
import os
import sys

# ==============================================================================
# 配置
# ==============================================================================
# 从命令行参数获取BASE_URL，如果没有则使用默认值
BASE_URL = sys.argv[1] if len(sys.argv) > 1 else "http://43.165.186.134:8888"
LOG_FILE = f"mqtt_webhook_test_{datetime.datetime.now().strftime('%Y%m%d_%H%M%S')}.log"

# 强制禁用代理
PROXIES = {
    "http": None,
    "https": None,
}

# ==============================================================================
# 颜色和日志记录
# ==============================================================================
class Color:
    HEADER = '\033[95m'
    OKBLUE = '\033[94m'
    OKCYAN = '\033[96m'
    OKGREEN = '\033[92m'
    WARNING = '\033[93m'
    FAIL = '\033[91m'
    ENDC = '\033[0m'
    BOLD = '\033[1m'
    UNDERLINE = '\033[4m'

def log_message(message, color=None):
    """打印彩色消息到控制台并写入日志文件"""
    timestamp = datetime.datetime.now().strftime('%Y-%m-%d %H:%M:%S')
    
    # 移除颜色代码以便写入纯文本日志
    plain_message = message
    if isinstance(message, str):
        for c in vars(Color).values():
            if isinstance(c, str):
                plain_message = plain_message.replace(c, '')

    with open(LOG_FILE, 'a', encoding='utf-8') as f:
        f.write(f"[{timestamp}] {plain_message}\n")

    # 在CI/CD或非TTY环境禁用颜色
    if sys.stdout.isatty():
        if color:
            print(f"{color}{message}{Color.ENDC}")
        else:
            print(message)
    else:
        print(plain_message)


def print_success(message):
    log_message(f"✅ {message}", Color.OKGREEN)

def print_error(message):
    log_message(f"❌ {message}", Color.FAIL)

def print_warning(message):
    log_message(f"⚠️  {message}", Color.WARNING)

def print_info(message):
    log_message(f"ℹ️  {message}", Color.OKCYAN)
    
def print_section(title):
    log_message("\n" + "=" * 80, Color.HEADER)
    log_message(f"  {title}", Color.HEADER)
    log_message("=" * 80, Color.HEADER)

# ==============================================================================
# 测试核心函数
# ==============================================================================
test_results = {
    "total": 0,
    "success": 0,
    "failed": 0,
    "details": []
}

def invoke_test_request(url, body, test_name):
    """执行单个HTTP POST测试请求"""
    global test_results
    test_results["total"] += 1
    
    print_info(f"正在测试: {test_name}")
    print_info(f"请求URL: {url}")
    print_info(f"请求数据: {json.dumps(body, indent=2, ensure_ascii=False)}")
    
    try:
        headers = {
            'Content-Type': 'application/json',
            'Accept': 'application/json'
        }
        response = requests.post(url, json=body, headers=headers, timeout=30, proxies=PROXIES)
        
        # 对于Webhook，EMQX通常只关心200 OK，响应体是给客户端的
        if response.status_code == 200:
            print_success(f"[{test_name}] 测试通过 (HTTP 200 OK)")
            try:
                response_json = response.json()
                print_info("响应数据:")
                log_message(json.dumps(response_json, indent=2, ensure_ascii=False))
                test_results["success"] += 1
                test_results["details"].append({"name": test_name, "status": "SUCCESS", "details": "HTTP 200"})
            except json.JSONDecodeError:
                # 即使不是JSON，对于某些Webhook响应也可以是正常的
                print_warning(f"响应内容不是有效的JSON格式，但HTTP状态码为200")
                log_message(f"响应原文: {response.text}")
                test_results["success"] += 1 # 依然算成功
                test_results["details"].append({"name": test_name, "status": "SUCCESS", "details": "HTTP 200, non-JSON response"})

        else:
            print_error(f"[{test_name}] 测试失败: HTTP状态码 {response.status_code}")
            log_message(f"响应原文: {response.text}")
            test_results["failed"] += 1
            test_results["details"].append({"name": test_name, "status": "FAILED", "details": f"HTTP {response.status_code}"})

    except requests.exceptions.RequestException as e:
        error_message = f"请求异常: {e}"
        print_error(f"[{test_name}] 测试失败: {error_message}")
        test_results["failed"] += 1
        test_results["details"].append({"name": test_name, "status": "FAILED", "details": str(e)})

def run_health_check():
    """服务器连通性测试"""
    global test_results
    test_results["total"] += 1
    test_name = "服务器连通性"
    print_section(test_name)
    try:
        # Webhook API没有专门的健康检查端点, 我们尝试访问一个不存在的路径来测试连通性
        # 预期会收到404, 这证明服务器是活的并且网络是通的
        response = requests.get(f"{BASE_URL}/health_check", timeout=10, proxies=PROXIES)
        if response.status_code == 404:
             print_success("服务器连通性正常 (能收到404响应)")
             test_results["success"] += 1
             test_results["details"].append({"name": test_name, "status": "SUCCESS", "details": "Received 404, server is alive"})
        else:
             print_error(f"服务器连通性测试异常: 期望404, 但收到 {response.status_code}")
             test_results["failed"] += 1
             test_results["details"].append({"name": test_name, "status": "FAILED", "details": f"Expected 404, got {response.status_code}"})
    except requests.exceptions.RequestException as e:
        print_error(f"服务器连通性失败: {e}")
        test_results["failed"] += 1
        test_results["details"].append({"name": test_name, "status": "FAILED", "details": str(e)})

# ==============================================================================
# 测试用例
# ==============================================================================
def run_all_tests():
    """按顺序执行所有测试用例"""
    print_section(f"MQTT Webhook API 自动化测试脚本 (Python)")
    print_info(f"测试目标: {BASE_URL}")
    print_info(f"日志文件: {LOG_FILE}")
    print_info(f"开始时间: {datetime.datetime.now().strftime('%Y-%m-%d %H:%M:%S')}")

    # 服务器连通性
    run_health_check()
    
    # 1. 角色请求Hook - 正常分配角色
    print_section("测试1: 角色请求Hook - 正常分配角色")
    invoke_test_request(
        url=f"{BASE_URL}/mqtt/hooks/role_request",
        body={
            "clientid": f"user123-mobile-{int(time.time())}",
            "username": "user123",
            "topic": "nfc_relay/role_request",
            "payload": {
                "role": "mobile",
                "force_kick": False,
                "device_info": {
                    "device_model": "iPhone 14",
                    "os_version": "iOS 16.0",
                    "app_version": "1.0.0"
                }
            },
            "timestamp": int(time.time())
        },
        test_name="正常角色分配"
    )

    # 2. 角色请求Hook - 角色冲突检测 (需要先模拟一个设备在线)
    print_section("测试2: 角色请求Hook - 角色冲突检测")
    invoke_test_request(
        url=f"{BASE_URL}/mqtt/hooks/role_request",
        body={
            "clientid": f"user123-pc-{int(time.time())}",
            "username": "user123",
            "topic": "nfc_relay/role_request",
            "payload": {
                "role": "pc",
                "force_kick": False,
                "device_info": {
                    "device_model": "Windows PC",
                    "os_version": "Windows 11",
                }
            },
            "timestamp": int(time.time())
        },
        test_name="角色冲突检测 (无真实冲突)"
    )

    # 3. 角色请求Hook - 强制踢下线
    print_section("测试3: 角色请求Hook - 强制踢下线")
    invoke_test_request(
        url=f"{BASE_URL}/mqtt/hooks/role_request",
        body={
            "clientid": f"user123-pc-force-{int(time.time())}",
            "username": "user123",
            "topic": "nfc_relay/role_request",
            "payload": {
                "role": "pc",
                "force_kick": True,
                "device_info": {
                    "device_model": "Windows PC New",
                }
            },
            "timestamp": int(time.time())
        },
        test_name="强制踢下线"
    )

    # 4. 角色请求Hook - 无效ClientID
    print_section("测试4: 角色请求Hook - 无效ClientID格式")
    invoke_test_request(
        url=f"{BASE_URL}/mqtt/hooks/role_request",
        body={
            "clientid": "invalid_format",
            "username": "user123",
            "topic": "nfc_relay/role_request",
            "payload": {"role": "mobile"},
            "timestamp": int(time.time())
        },
        test_name="无效ClientID格式"
    )

    # 5. 连接状态Hook - 客户端连接
    print_section("测试5: 连接状态Hook - 客户端连接")
    invoke_test_request(
        url=f"{BASE_URL}/mqtt/hooks/connection_status",
        body={
            "event_type": "client_connected",
            "clientid": f"user123-mobile-{int(time.time())}",
            "username": "user123",
            "connected_at": datetime.datetime.now().isoformat()
        },
        test_name="客户端连接事件"
    )

    # 6. 连接状态Hook - 客户端断开
    print_section("测试6: 连接状态Hook - 客户端断开")
    invoke_test_request(
        url=f"{BASE_URL}/mqtt/hooks/connection_status",
        body={
            "event_type": "client_disconnected",
            "clientid": f"user123-mobile-{int(time.time())}",
            "username": "user123",
            "disconnected_at": datetime.datetime.now().isoformat(),
            "reason": "normal"
        },
        test_name="客户端断开事件"
    )
    
    # 7. 连接状态Hook - 无效事件类型
    print_section("测试7: 连接状态Hook - 无效事件类型")
    invoke_test_request(
        url=f"{BASE_URL}/mqtt/hooks/connection_status",
        body={
            "event_type": "invalid_event",
            "clientid": "test-client",
            "username": "testuser"
        },
        test_name="无效事件类型"
    )

    # 8. 参数错误测试 - 缺少必要字段
    print_section("测试8: 参数错误测试 - 缺少必要字段")
    invoke_test_request(
        url=f"{BASE_URL}/mqtt/hooks/role_request",
        body={"clientid": "test-client"},
        test_name="缺少必要字段"
    )

# ==============================================================================
# 总结报告
# ==============================================================================
def print_summary():
    """打印最终的测试总结报告"""
    print_section("测试结果汇总")
    print_info(f"总测试数: {test_results['total']}")
    print_success(f"成功: {test_results['success']}")
    print_error(f"失败: {test_results['failed']}")
    
    success_rate = (test_results['success'] / test_results['total'] * 100) if test_results['total'] > 0 else 0
    print_info(f"成功率: {success_rate:.2f}%")
    
    print("\n" + "-"*40)
    for detail in test_results["details"]:
        if detail["status"] == "SUCCESS":
            print_success(f"  - {detail['name']:<30} [  OK  ]")
        else:
            print_error(f"  - {detail['name']:<30} [ FAIL ] --> {detail['details']}")
            
    print_section("测试完成")
    print_info(f"完成时间: {datetime.datetime.now().strftime('%Y-%m-%d %H:%M:%S')}")
    print_info(f"详细日志已保存到: {os.path.abspath(LOG_FILE)}")
    
    if test_results['failed'] > 0:
        print_warning(f"\n有 {test_results['failed']} 个测试失败，请检查日志获取详细信息。")
        exit(1)
    else:
        print_success("\n所有测试通过! 🎉")
        exit(0)

# ==============================================================================
# 脚本入口
# ==============================================================================
if __name__ == "__main__":
    try:
        run_all_tests()
    except Exception as e:
        print_error(f"测试脚本发生未处理的异常: {e}")
    finally:
        print_summary()
        # 确保在Windows上颜色能正常重置
        if 'colorama' not in sys.modules:
             print(Color.ENDC) 