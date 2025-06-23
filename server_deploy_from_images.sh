 #!/bin/bash

# 服务器端使用预构建镜像部署脚本
# 使用方法: ./server_deploy_from_images.sh [dockerhub-username]

set -e

# 颜色定义
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# 配置
DOCKER_USERNAME="${1:-your-username}"
PROJECT_NAME="gin-vue-admin"

# 镜像名称
BACKEND_IMAGE="${DOCKER_USERNAME}/${PROJECT_NAME}-backend:latest"
FRONTEND_IMAGE="${DOCKER_USERNAME}/${PROJECT_NAME}-frontend:latest"

echo -e "${BLUE}🚀 开始部署 ${PROJECT_NAME} (使用预构建镜像)${NC}"
echo -e "${BLUE}================================================${NC}"
echo -e "后端镜像: ${BACKEND_IMAGE}"
echo -e "前端镜像: ${FRONTEND_IMAGE}"
echo -e "${BLUE}================================================${NC}"

# 1. 检查Docker是否安装
if ! command -v docker &> /dev/null; then
    echo -e "${YELLOW}📦 正在安装Docker...${NC}"
    curl -fsSL https://get.docker.com | bash -s docker --mirror Aliyun
    sudo systemctl start docker
    sudo systemctl enable docker
    echo -e "${GREEN}✅ Docker 安装完成${NC}"
else
    echo -e "${GREEN}✅ Docker 已安装${NC}"
fi

# 2. 检查Docker Compose是否安装
if ! command -v docker-compose &> /dev/null; then
    echo -e "${YELLOW}📦 正在安装Docker Compose...${NC}"
    sudo curl -L "https://get.daocloud.io/docker/compose/releases/download/v2.24.1/docker-compose-$(uname -s)-$(uname -m)" -o /usr/local/bin/docker-compose
    sudo chmod +x /usr/local/bin/docker-compose
    sudo ln -sf /usr/local/bin/docker-compose /usr/bin/docker-compose
    echo -e "${GREEN}✅ Docker Compose 安装完成${NC}"
else
    echo -e "${GREEN}✅ Docker Compose 已安装${NC}"
fi

# 3. 配置Docker镜像加速
echo -e "${YELLOW}⚙️  配置Docker镜像加速...${NC}"
sudo mkdir -p /etc/docker
sudo tee /etc/docker/daemon.json > /dev/null <<EOF
{
  "registry-mirrors": [
    "https://dockerproxy.com",
    "https://hub-mirror.c.163.com",
    "https://registry.docker-cn.com"
  ],
  "log-driver": "json-file",
  "log-opts": {
    "max-size": "100m",
    "max-file": "3"
  }
}
EOF
sudo systemctl restart docker

# 4. 创建项目目录结构
echo -e "${YELLOW}📁 创建项目目录...${NC}"
mkdir -p gin-vue-admin/deploy/mysql/init
mkdir -p gin-vue-admin/server/log
mkdir -p gin-vue-admin/server/uploads
cd gin-vue-admin

# 5. 创建配置文件（如果不存在）
if [ ! -f "server/config.docker.yaml" ]; then
    echo -e "${YELLOW}📝 创建默认配置文件...${NC}"
    mkdir -p server
    cat > server/config.docker.yaml << 'EOL'
# gin-vue-admin Docker 配置文件

jwt:
  signing-key: GvaJwtStrong2024!@#
  expires-time: 7d
  buffer-time: 1d
  issuer: gin-vue-admin

zap:
  level: info
  format: console
  prefix: "[gin-vue-admin]"
  director: log
  show-line: true
  encode-level: LowercaseColorLevelEncoder
  stacktrace-key: stacktrace
  log-in-console: true

redis:
  db: 0
  addr: redis:6379
  password: "Gva123456!"

system:
  env: public
  addr: 8888
  db-type: mysql
  oss-type: local
  use-redis: true
  use-mongo: false
  use-multipoint: false

captcha:
  key-long: 6
  img-width: 240
  img-height: 80
  open-captcha: 3
  open-captcha-timeout: 3600

mysql:
  path: "mysql"
  port: "3306"
  config: "charset=utf8mb4&parseTime=True&loc=Local"
  db-name: "gin_vue_admin"
  username: gva_user
  password: "Gva123456!"
  max-idle-conns: 10
  max-open-conns: 100
  log-mode: error
  log-zap: false

local:
  path: uploads/file
  store-path: uploads/file

autocode:
  transfer-restart: true
  root: ""
  server: /server
  server-api: /api/v1/%s
  server-initialize: /initialize
  server-model: /model/%s
  server-request: /model/%s/request/
  server-router: /router/%s
  server-service: /service/%s
  web: /web/src
  web-api: /api
  web-form: /view
  web-table: /view
EOL
fi

# 6. 创建docker-compose配置
echo -e "${YELLOW}📝 创建 docker-compose 配置...${NC}"
mkdir -p deploy
cat > deploy/docker-compose-production.yml << EOF
version: '3.8'

networks:
  gva-network:
    driver: bridge

volumes:
  mysql-data:
    driver: local
  redis-data:
    driver: local

services:
  # MySQL 数据库服务
  mysql:
    image: mysql:8.0
    container_name: gva-mysql
    restart: unless-stopped
    environment:
      MYSQL_ROOT_PASSWORD: Gva123456!
      MYSQL_DATABASE: gin_vue_admin
      MYSQL_USER: gva_user
      MYSQL_PASSWORD: Gva123456!
      TZ: Asia/Shanghai
    ports:
      - "3306:3306"
    volumes:
      - mysql-data:/var/lib/mysql
      - ./mysql/init:/docker-entrypoint-initdb.d
    command: >
      mysqld
      --character-set-server=utf8mb4
      --collation-server=utf8mb4_unicode_ci
      --default-authentication-plugin=mysql_native_password
      --max_connections=1000
      --innodb_buffer_pool_size=512M
    healthcheck:
      test: ["CMD", "mysqladmin", "ping", "-h", "localhost", "-u", "root", "-pGva123456!"]
      interval: 30s
      timeout: 10s
      retries: 5
      start_period: 120s
    networks:
      - gva-network

  # Redis 缓存服务
  redis:
    image: redis:7-alpine
    container_name: gva-redis
    restart: unless-stopped
    ports:
      - "6379:6379"
    volumes:
      - redis-data:/data
    command: >
      redis-server
      --requirepass "Gva123456!"
      --appendonly yes
      --appendfsync everysec
      --maxmemory 256mb
      --maxmemory-policy allkeys-lru
    healthcheck:
      test: ["CMD", "redis-cli", "-a", "Gva123456!", "ping"]
      interval: 30s
      timeout: 10s
      retries: 3
      start_period: 30s
    networks:
      - gva-network

  # 后端服务 - 使用预构建镜像
  backend:
    image: ${BACKEND_IMAGE}
    container_name: gin-vue-admin-backend
    restart: unless-stopped
    volumes:
      - ../server/config.docker.yaml:/go/src/github.com/flipped-aurora/gin-vue-admin/server/config.docker.yaml:ro
      - ../server/log:/go/src/github.com/flipped-aurora/gin-vue-admin/server/log
      - ../server/uploads:/go/src/github.com/flipped-aurora/gin-vue-admin/server/uploads
    environment:
      TZ: Asia/Shanghai
      GIN_MODE: release
    ports:
      - "8888:8888"
    depends_on:
      mysql:
        condition: service_healthy
      redis:
        condition: service_healthy
    healthcheck:
      test: ["CMD-SHELL", "wget -q --spider http://localhost:8888 || exit 1"]
      interval: 30s
      timeout: 10s
      retries: 3
      start_period: 60s
    networks:
      - gva-network

  # 前端服务 - 使用预构建镜像
  frontend:
    image: ${FRONTEND_IMAGE}
    container_name: gin-vue-admin-frontend
    restart: unless-stopped
    environment:
      TZ: Asia/Shanghai
    ports:
      - "80:80"
    depends_on:
      backend:
        condition: service_healthy
    healthcheck:
      test: ["CMD-SHELL", "wget -q --spider http://localhost:80 || exit 1"]
      interval: 30s
      timeout: 10s
      retries: 3
      start_period: 30s
    networks:
      - gva-network
EOF

# 7. 停止旧服务
echo -e "${YELLOW}🛑 停止旧服务...${NC}"
docker-compose -f deploy/docker-compose-production.yml down --remove-orphans 2>/dev/null || true

# 8. 拉取最新镜像
echo -e "${YELLOW}📥 拉取最新镜像...${NC}"
docker pull mysql:8.0
docker pull redis:7-alpine
docker pull ${BACKEND_IMAGE}
docker pull ${FRONTEND_IMAGE}

# 9. 启动服务
echo -e "${YELLOW}🏗️  启动服务...${NC}"
docker-compose -f deploy/docker-compose-production.yml up -d

# 10. 等待服务启动
echo -e "${YELLOW}⏳ 等待服务启动...${NC}"
sleep 60

# 11. 检查服务状态
echo -e "${YELLOW}🔍 检查服务状态...${NC}"
docker-compose -f deploy/docker-compose-production.yml ps

# 12. 显示服务日志
echo -e "${YELLOW}📋 显示最近日志...${NC}"
docker-compose -f deploy/docker-compose-production.yml logs --tail=20

# 13. 获取服务器IP
SERVER_IP=$(curl -s ifconfig.me 2>/dev/null || curl -s icanhazip.com 2>/dev/null || echo "localhost")

echo ""
echo -e "${GREEN}🎉 部署完成！${NC}"
echo -e "${GREEN}================================================${NC}"
echo -e "🌐 前端地址: http://${SERVER_IP}"
echo -e "🔧 后端API: http://${SERVER_IP}:8888"
echo -e "📚 API文档: http://${SERVER_IP}:8888/swagger/index.html"
echo -e "${GREEN}================================================${NC}"
echo ""
echo -e "${YELLOW}📋 管理命令:${NC}"
echo "查看日志: docker-compose -f deploy/docker-compose-production.yml logs -f"
echo "重启服务: docker-compose -f deploy/docker-compose-production.yml restart"
echo "停止服务: docker-compose -f deploy/docker-compose-production.yml down"
echo "更新镜像: docker-compose -f deploy/docker-compose-production.yml pull && docker-compose -f deploy/docker-compose-production.yml up -d"
echo ""
echo -e "${GREEN}✨ 服务已成功部署！${NC}"