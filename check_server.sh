#!/bin/bash

echo "🔍 检查服务器状态脚本"
echo "===================="
echo "服务器IP: 43.165.186.134"
echo "检查时间: $(date)"
echo ""

echo "📋 1. 检查系统信息"
echo "系统版本: $(cat /etc/os-release | grep PRETTY_NAME | cut -d'=' -f2 | tr -d '\"')"
echo "内核版本: $(uname -r)"
echo ""

echo "🐳 2. 检查Docker状态"
if command -v docker &> /dev/null; then
    echo "✅ Docker已安装: $(docker --version)"
    systemctl is-active docker && echo "✅ Docker服务运行中" || echo "❌ Docker服务未运行"
    echo ""
    
    echo "📦 3. 检查容器状态"
    docker ps -a --format "table {{.Names}}\t{{.Status}}\t{{.Ports}}"
    echo ""
    
    echo "🔗 4. 检查Docker Compose"
    if [ -f "/opt/gin-vue-admin/deploy/docker-compose.yml" ]; then
        echo "✅ docker-compose.yml 存在"
        cd /opt/gin-vue-admin
        docker-compose -f deploy/docker-compose.yml ps
    else
        echo "❌ docker-compose.yml 不存在"
    fi
else
    echo "❌ Docker未安装"
fi
echo ""

echo "🌐 5. 检查端口占用"
echo "检查80端口: $(ss -tlnp | grep :80 || echo '未占用')"
echo "检查8888端口: $(ss -tlnp | grep :8888 || echo '未占用')"
echo ""

echo "📁 6. 检查项目目录"
if [ -d "/opt/gin-vue-admin" ]; then
    echo "✅ 项目目录存在: /opt/gin-vue-admin"
    echo "目录大小: $(du -sh /opt/gin-vue-admin)"
    echo "文件列表:"
    ls -la /opt/gin-vue-admin/
    echo ""
    
    if [ -d "/opt/gin-vue-admin/deploy" ]; then
        echo "📂 deploy目录内容:"
        ls -la /opt/gin-vue-admin/deploy/
    else
        echo "❌ deploy目录不存在"
    fi
else
    echo "❌ 项目目录不存在: /opt/gin-vue-admin"
fi
echo ""

echo "🔑 7. 检查SSH配置"
if [ -f "~/.ssh/id_rsa" ]; then
    echo "✅ SSH密钥存在"
else
    echo "❌ SSH密钥不存在"
fi
echo ""

echo "📊 8. 系统资源使用情况"
echo "内存使用: $(free -h | grep Mem | awk '{print $3"/"$2}')"
echo "磁盘使用: $(df -h / | tail -1 | awk '{print $3"/"$2" ("$5")"}')"
echo "CPU负载: $(uptime | awk -F'load average:' '{print $2}')"
echo ""

echo "🌐 9. 网络连接测试"
echo "测试GitHub连接:"
timeout 5 curl -s https://github.com > /dev/null && echo "✅ GitHub连接正常" || echo "❌ GitHub连接失败"

echo "测试DNS解析:"
nslookup github.com > /dev/null 2>&1 && echo "✅ DNS解析正常" || echo "❌ DNS解析失败"
echo ""

echo "🔍 检查完成！" 