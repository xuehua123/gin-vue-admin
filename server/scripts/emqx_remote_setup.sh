#!/bin/bash

##--------------------------------------------------------------------
## EMQX远程配置脚本
## 用途：安全卡片中继系统远程MQTT Broker配置
## 作者：NFC Relay System Team
## 版本：1.0.0
## 远程EMQX地址：http://49.235.40.39/
##--------------------------------------------------------------------

set -e

# 脚本配置
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(dirname "$SCRIPT_DIR")"

# 远程EMQX配置
EMQX_HOST="49.235.40.39"
EMQX_HTTP_PORT="18083"
EMQX_MQTT_PORT="1883"
EMQX_MQTTS_PORT="8883"
EMQX_WS_PORT="8083"
EMQX_WSS_PORT="8084"
EMQX_DASHBOARD_URL="http://${EMQX_HOST}:${EMQX_HTTP_PORT}"
EMQX_USERNAME="admin"
EMQX_PASSWORD="xuehua123"

# 颜色输出
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# 日志函数
log_info() {
    echo -e "${GREEN}[INFO]${NC} $1"
}

log_warn() {
    echo -e "${YELLOW}[WARN]${NC} $1"
}

log_error() {
    echo -e "${RED}[ERROR]${NC} $1"
}

log_debug() {
    echo -e "${BLUE}[DEBUG]${NC} $1"
}

# 检查依赖
check_dependencies() {
    log_info "检查系统依赖..."
    
    # 检查curl
    if ! command -v curl &> /dev/null; then
        log_error "curl未安装，请先安装curl"
        exit 1
    fi
    
    # 检查jq (用于处理JSON响应)
    if ! command -v jq &> /dev/null; then
        log_warn "jq未安装，建议安装jq以便更好地处理API响应"
    fi
    
    log_info "依赖检查通过"
}

# 测试EMQX连接
test_emqx_connection() {
    log_info "测试EMQX连接..."
    
    # 测试Dashboard连接
    local response_code=$(curl -s -o /dev/null -w "%{http_code}" "$EMQX_DASHBOARD_URL" --connect-timeout 10)
    
    if [[ "$response_code" == "200" ]]; then
        log_info "EMQX Dashboard连接成功 (${EMQX_DASHBOARD_URL})"
    else
        log_error "EMQX Dashboard连接失败，响应码: $response_code"
        log_error "请检查EMQX服务是否正常运行"
        exit 1
    fi
    
    # 测试MQTT端口
    if timeout 5 bash -c "</dev/tcp/$EMQX_HOST/$EMQX_MQTT_PORT"; then
        log_info "MQTT端口 $EMQX_MQTT_PORT 连接成功"
    else
        log_warn "MQTT端口 $EMQX_MQTT_PORT 连接失败，请检查防火墙设置"
    fi
}

# 获取API Token
get_api_token() {
    log_info "获取EMQX API Token..."
    
    local response=$(curl -s -X POST "$EMQX_DASHBOARD_URL/api/v5/login" \
        -H "Content-Type: application/json" \
        -d "{\"username\":\"$EMQX_USERNAME\",\"password\":\"$EMQX_PASSWORD\"}")
    
    if command -v jq &> /dev/null; then
        local token=$(echo "$response" | jq -r '.token // empty')
        if [[ -n "$token" && "$token" != "null" ]]; then
            echo "$token"
            log_info "API Token获取成功"
        else
            log_error "API Token获取失败: $(echo "$response" | jq -r '.message // "未知错误"')"
            exit 1
        fi
    else
        # 如果没有jq，尝试简单的文本处理
        local token=$(echo "$response" | grep -o '"token":"[^"]*"' | cut -d'"' -f4)
        if [[ -n "$token" ]]; then
            echo "$token"
            log_info "API Token获取成功"
        else
            log_error "API Token获取失败，请安装jq以获得更好的错误信息"
            log_error "响应: $response"
            exit 1
        fi
    fi
}

# 上传ACL配置
upload_acl_config() {
    local token="$1"
    log_info "上传ACL配置..."
    
    if [[ ! -f "$PROJECT_ROOT/config/emqx_acl.conf" ]]; then
        log_error "未找到ACL配置文件: $PROJECT_ROOT/config/emqx_acl.conf"
        exit 1
    fi
    
    # 检查是否已存在基于文件的授权器
    local auth_sources=$(curl -s -H "Authorization: Bearer $token" \
        "$EMQX_DASHBOARD_URL/api/v5/authorization/sources")
    
    # 上传ACL文件
    local response=$(curl -s -X POST "$EMQX_DASHBOARD_URL/api/v5/authorization/sources/file/acl" \
        -H "Authorization: Bearer $token" \
        -H "Content-Type: text/plain" \
        --data-binary "@$PROJECT_ROOT/config/emqx_acl.conf")
    
    if command -v jq &> /dev/null; then
        local success=$(echo "$response" | jq -r '.message // empty')
        if [[ "$success" == "ok" ]] || echo "$response" | jq -e '.data' > /dev/null 2>&1; then
            log_info "ACL配置上传成功"
        else
            log_warn "ACL配置上传可能失败: $(echo "$response" | jq -r '.message // "未知错误"')"
            log_info "响应: $response"
        fi
    else
        log_info "ACL配置上传完成，响应: $response"
    fi
}

# 配置JWT认证
configure_jwt_auth() {
    local token="$1"
    log_info "配置JWT认证..."
    
    # JWT认证配置 - 根据EMQX官方文档格式
    local jwt_config='{
        "mechanism": "password_based",
        "backend": "jwt",
        "enable": true,
        "use_jwks": false,
        "algorithm": "hmac-based",
        "secret": "78c0f08f-9663-4c9c-a399-cc4ec36b8112",
        "secret_base64_encoded": false,
        "from": "password",
        "verify_claims": {
            "exp": "${timestamp}",
            "iss": "qmPlus",
            "aud": "GVA",
            "client_id": "${clientid}"
        },
        "disconnect_after_expire": true
    }'
    
    # 检查现有认证器
    local auth_list=$(curl -s -H "Authorization: Bearer $token" \
        "$EMQX_DASHBOARD_URL/api/v5/authentication")
    
    # 创建或更新JWT认证器
    local response=$(curl -s -X POST "$EMQX_DASHBOARD_URL/api/v5/authentication" \
        -H "Authorization: Bearer $token" \
        -H "Content-Type: application/json" \
        -d "$jwt_config")
    
    if command -v jq &> /dev/null; then
        local success=$(echo "$response" | jq -r '.message // empty')
        if [[ "$success" == "ok" ]] || echo "$response" | jq -e '.data' > /dev/null 2>&1; then
            log_info "JWT认证配置成功"
        else
            log_warn "JWT认证配置可能失败: $(echo "$response" | jq -r '.message // .code // "未知错误"')"
            log_info "响应: $response"
        fi
    else
        log_info "JWT认证配置完成，响应: $response"
    fi
}

# 配置授权设置
configure_authorization() {
    local token="$1"
    log_info "配置授权设置..."
    
    # 授权配置
    local authz_config='{
        "no_match": "deny",
        "deny_action": "ignore",
        "cache": {
            "enable": true,
            "max_size": 32,
            "ttl": "1m"
        }
    }'
    
    local response=$(curl -s -X PUT "$EMQX_DASHBOARD_URL/api/v5/authorization/settings" \
        -H "Authorization: Bearer $token" \
        -H "Content-Type: application/json" \
        -d "$authz_config")
    
    if command -v jq &> /dev/null; then
        local success=$(echo "$response" | jq -r '.message // empty')
        if [[ "$success" == "ok" ]] || echo "$response" | jq -e '.data' > /dev/null 2>&1; then
            log_info "授权设置配置成功"
        else
            log_warn "授权设置配置可能失败: $(echo "$response" | jq -r '.message // "未知错误"')"
        fi
    else
        log_info "授权设置配置完成"
    fi
}

# 验证配置
verify_configuration() {
    local token="$1"
    log_info "验证EMQX配置..."
    
    # 验证认证器状态
    log_debug "检查认证器状态..."
    local auth_status=$(curl -s -H "Authorization: Bearer $token" \
        "$EMQX_DASHBOARD_URL/api/v5/authentication")
    
    # 验证授权器状态
    log_debug "检查授权器状态..."
    local authz_status=$(curl -s -H "Authorization: Bearer $token" \
        "$EMQX_DASHBOARD_URL/api/v5/authorization/sources")
    
    if command -v jq &> /dev/null; then
        local auth_count=$(echo "$auth_status" | jq length 2>/dev/null || echo "0")
        local authz_count=$(echo "$authz_status" | jq length 2>/dev/null || echo "0")
        
        log_info "配置验证完成:"
        log_info "  - 认证器数量: $auth_count"
        log_info "  - 授权器数量: $authz_count"
    else
        log_info "配置验证完成 (安装jq以获得详细信息)"
    fi
}

# 显示连接信息
show_connection_info() {
    log_info "EMQX连接信息:"
    echo -e "${BLUE}┌─────────────────────────────────────────────┐${NC}"
    echo -e "${BLUE}│                EMQX连接信息                   │${NC}"
    echo -e "${BLUE}├─────────────────────────────────────────────┤${NC}"
    echo -e "${BLUE}│${NC} MQTT TCP:   mqtt://${EMQX_HOST}:${EMQX_MQTT_PORT}          ${BLUE}│${NC}"
    echo -e "${BLUE}│${NC} MQTT SSL:   mqtts://${EMQX_HOST}:${EMQX_MQTTS_PORT}         ${BLUE}│${NC}"
    echo -e "${BLUE}│${NC} WebSocket:  ws://${EMQX_HOST}:${EMQX_WS_PORT}             ${BLUE}│${NC}"
    echo -e "${BLUE}│${NC} WebSocket SSL: wss://${EMQX_HOST}:${EMQX_WSS_PORT}        ${BLUE}│${NC}"
    echo -e "${BLUE}│${NC} Dashboard:  ${EMQX_DASHBOARD_URL}              ${BLUE}│${NC}"
    echo -e "${BLUE}│${NC} 用户名:     ${EMQX_USERNAME}                           ${BLUE}│${NC}"
    echo -e "${BLUE}│${NC} 密码:       ${EMQX_PASSWORD}                     ${BLUE}│${NC}"
    echo -e "${BLUE}└─────────────────────────────────────────────┘${NC}"
    
    log_info "客户端连接配置:"
    echo -e "${GREEN}  Broker地址: ${EMQX_HOST}${NC}"
    echo -e "${GREEN}  MQTT端口: ${EMQX_MQTT_PORT} (TCP) / ${EMQX_MQTTS_PORT} (SSL)${NC}"
    echo -e "${GREEN}  认证方式: JWT (password字段)${NC}"
    echo -e "${GREEN}  Client ID: 使用登录时获取的clientID${NC}"
    echo -e "${GREEN}  Username: clientID${NC}"
    echo -e "${GREEN}  Password: JWT Token${NC}"
}

# 生成客户端配置文件
generate_client_config() {
    log_info "生成客户端配置文件..."
    
    mkdir -p "$PROJECT_ROOT/config"
    cat > "$PROJECT_ROOT/config/emqx_client_config.json" << EOF
{
  "emqx": {
    "host": "${EMQX_HOST}",
    "ports": {
      "mqtt": ${EMQX_MQTT_PORT},
      "mqtts": ${EMQX_MQTTS_PORT},
      "ws": ${EMQX_WS_PORT},
      "wss": ${EMQX_WSS_PORT}
    },
    "endpoints": {
      "mqtt_tcp": "mqtt://${EMQX_HOST}:${EMQX_MQTT_PORT}",
      "mqtt_ssl": "mqtts://${EMQX_HOST}:${EMQX_MQTTS_PORT}",
      "websocket": "ws://${EMQX_HOST}:${EMQX_WS_PORT}",
      "websocket_ssl": "wss://${EMQX_HOST}:${EMQX_WSS_PORT}"
    },
    "dashboard_url": "${EMQX_DASHBOARD_URL}",
    "authentication": {
      "method": "jwt",
      "jwt_secret": "78c0f08f-9663-4c9c-a399-cc4ec36b8112",
      "issuer": "qmPlus",
      "audience": "GVA"
    },
    "connection": {
      "username_field": "clientid",
      "password_field": "jwt_token",
      "clean_session": true,
      "keep_alive": 60,
      "timeout": 30
    }
  }
}
EOF
    
    log_info "客户端配置文件已生成: $PROJECT_ROOT/config/emqx_client_config.json"
}

# 主函数
main() {
    echo -e "${BLUE}════════════════════════════════════════════════════════${NC}"
    echo -e "${BLUE}           EMQX远程配置脚本 - NFC卡片中继系统              ${NC}"
    echo -e "${BLUE}════════════════════════════════════════════════════════${NC}"
    
    case "${1:-setup}" in
        "setup"|"install")
            check_dependencies
            test_emqx_connection
            
            local token=$(get_api_token)
            
            upload_acl_config "$token"
            configure_jwt_auth "$token"
            configure_authorization "$token"
            verify_configuration "$token"
            generate_client_config
            
            show_connection_info
            log_info "EMQX远程配置完成！"
            ;;
            
        "test"|"verify")
            check_dependencies
            test_emqx_connection
            
            local token=$(get_api_token)
            verify_configuration "$token"
            ;;
            
        "info"|"status")
            show_connection_info
            ;;
            
        "help"|"-h"|"--help")
            echo "使用方法: $0 [命令]"
            echo ""
            echo "命令:"
            echo "  setup, install  配置远程EMQX实例 (默认)"
            echo "  test, verify    测试EMQX连接和配置"
            echo "  info, status    显示连接信息"
            echo "  help           显示此帮助信息"
            echo ""
            echo "远程EMQX信息:"
            echo "  地址: ${EMQX_HOST}"
            echo "  Dashboard: ${EMQX_DASHBOARD_URL}"
            ;;
            
        *)
            log_error "未知命令: $1"
            echo "使用 '$0 help' 查看可用命令"
            exit 1
            ;;
    esac
}

# 执行主函数
main "$@" 