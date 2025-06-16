#!/bin/bash

# 简单的管理脚本
cd /opt/gin-vue-admin

case $1 in
    start)
        echo "🚀 启动服务..."
        sudo docker-compose -f deploy/docker-compose.yml up -d --build
        echo "✅ 服务已启动！"
        ;;
    stop)
        echo "⏹️ 停止服务..."
        sudo docker-compose -f deploy/docker-compose.yml down
        echo "✅ 服务已停止！"
        ;;
    restart)
        echo "🔄 重启服务..."
        sudo docker-compose -f deploy/docker-compose.yml down
        sudo docker-compose -f deploy/docker-compose.yml up -d --build
        echo "✅ 服务已重启！"
        ;;
    status)
        echo "📊 服务状态："
        sudo docker-compose -f deploy/docker-compose.yml ps
        ;;
    logs)
        echo "📋 查看日志："
        sudo docker-compose -f deploy/docker-compose.yml logs -f
        ;;
    *)
        echo "用法: $0 {start|stop|restart|status|logs}"
        echo ""
        echo "  start   - 启动服务"
        echo "  stop    - 停止服务" 
        echo "  restart - 重启服务"
        echo "  status  - 查看状态"
        echo "  logs    - 查看日志"
        ;;
esac 