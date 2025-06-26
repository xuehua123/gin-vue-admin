# scripts/config.py - 测试脚本配置文件

# --- 后端服务器配置 ---
# 您本地开发环境的服务器地址
# SERVER_HOST = "127.0.0.1" 
# 您的公网服务器IP地址
SERVER_HOST = "43.165.186.134"
SERVER_PORT = 8888
SERVER_BASE_URL = f"http://{SERVER_HOST}:{SERVER_PORT}"

# --- EMQX 配置 ---
# 您本地开发环境的EMQX地址
# EMQX_HOST = "127.0.0.1"
# 您的公网EMQX IP地址
EMQX_HOST = "192.168.50.194"
EMQX_API_PORT = 18083
EMQX_MQTT_PORT = 1883
EMQX_DASHBOARD_URL = f"http://{EMQX_HOST}:{EMQX_API_PORT}"

# EMQX API 密钥配置 (从上次获取的配置中获得)
EMQX_API_KEY = "99bb839bf1dbdc90"
EMQX_SECRET_KEY = "YwYdYJtH4i9CL0wYnnV7xjZWAPKnMwTaUX9AG9CxTU7usF"

# --- 测试用户配置 ---
# 用于测试的用户凭据
USER1_CREDENTIALS = {
    "username": "admin",
    "password": "xuehua123"
}

USER2_CREDENTIALS = {
    "username": "admin",
    "password": "xuehua123"
}

# --- 角色配置 ---
# 用于测试的角色名称
ROLE_CARD_ISSUER = "card_issuer"
ROLE_CARD_RECEIVER = "card_receiver" 