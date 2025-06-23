 #!/bin/bash

# 本地构建并推送Docker镜像脚本
# 使用方法: ./build_and_push.sh [your-dockerhub-username]

set -e

# 颜色定义
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# 配置
DOCKER_USERNAME="${1:-your-username}"  # 从参数获取或使用默认值
PROJECT_NAME="gin-vue-admin"
VERSION=$(date +%Y%m%d_%H%M%S)
TAG_LATEST="latest"

# 镜像名称
BACKEND_IMAGE="${DOCKER_USERNAME}/${PROJECT_NAME}-backend"
FRONTEND_IMAGE="${DOCKER_USERNAME}/${PROJECT_NAME}-frontend"

echo -e "${BLUE}🚀 开始构建 ${PROJECT_NAME} Docker 镜像${NC}"
echo -e "${BLUE}================================================${NC}"
echo -e "Docker用户名: ${DOCKER_USERNAME}"
echo -e "项目名称: ${PROJECT_NAME}"
echo -e "版本标签: ${VERSION}"
echo -e "${BLUE}================================================${NC}"

# 检查Docker是否运行
if ! docker info >/dev/null 2>&1; then
    echo -e "${RED}❌ Docker 未运行，请启动 Docker 后重试${NC}"
    exit 1
fi

# 检查是否已登录Docker Hub
echo -e "${YELLOW}🔐 检查 Docker Hub 登录状态...${NC}"
if ! docker info | grep -q "Username"; then
    echo -e "${YELLOW}请登录 Docker Hub:${NC}"
    docker login
fi

# 1. 构建后端镜像
echo -e "${YELLOW}🏗️  构建后端镜像...${NC}"
cd server
docker build \
    --build-arg GOPROXY=https://goproxy.cn,direct \
    --build-arg GO111MODULE=on \
    -t ${BACKEND_IMAGE}:${VERSION} \
    -t ${BACKEND_IMAGE}:${TAG_LATEST} \
    .

if [ $? -eq 0 ]; then
    echo -e "${GREEN}✅ 后端镜像构建成功${NC}"
else
    echo -e "${RED}❌ 后端镜像构建失败${NC}"
    exit 1
fi

cd ..

# 2. 构建前端镜像
echo -e "${YELLOW}🏗️  构建前端镜像...${NC}"
docker build \
    --build-arg NPM_REGISTRY=https://registry.npmmirror.com \
    -f web/Dockerfile \
    -t ${FRONTEND_IMAGE}:${VERSION} \
    -t ${FRONTEND_IMAGE}:${TAG_LATEST} \
    .

if [ $? -eq 0 ]; then
    echo -e "${GREEN}✅ 前端镜像构建成功${NC}"
else
    echo -e "${RED}❌ 前端镜像构建失败${NC}"
    exit 1
fi

# 3. 显示构建的镜像
echo -e "${YELLOW}📋 构建完成的镜像:${NC}"
docker images | grep -E "${DOCKER_USERNAME}/${PROJECT_NAME}"

# 4. 推送镜像到Docker Hub
echo -e "${YELLOW}📤 推送镜像到 Docker Hub...${NC}"

echo -e "${BLUE}推送后端镜像...${NC}"
docker push ${BACKEND_IMAGE}:${VERSION}
docker push ${BACKEND_IMAGE}:${TAG_LATEST}

echo -e "${BLUE}推送前端镜像...${NC}"
docker push ${FRONTEND_IMAGE}:${VERSION}
docker push ${FRONTEND_IMAGE}:${TAG_LATEST}

# 5. 生成服务器部署配置
echo -e "${YELLOW}📝 生成服务器部署配置...${NC}"

# 创建用于服务器部署的docker-compose文件
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
  mysql-config:
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
      - mysql-config:/etc/mysql/conf.d
      - ./mysql/init:/docker-entrypoint-initdb.d
    command: >
      mysqld
      --character-set-server=utf8mb4
      --collation-server=utf8mb4_unicode_ci
      --default-authentication-plugin=mysql_native_password
      --max_connections=1000
      --innodb_buffer_pool_size=512M
    healthcheck:
      test: ["CMD", "mysqladmin", "ping", "-h", "localhost", "-u", "root", "-p\$\$MYSQL_ROOT_PASSWORD"]
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
      test: ["CMD", "redis-cli", "--raw", "incr", "ping"]
      interval: 30s
      timeout: 10s
      retries: 3
      start_period: 30s
    networks:
      - gva-network

  # 后端服务 - 使用预构建镜像
  backend:
    image: ${BACKEND_IMAGE}:${TAG_LATEST}
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
      test: ["CMD", "wget", "--quiet", "--tries=1", "--spider", "http://localhost:8888/health", "||", "exit", "1"]
      interval: 30s
      timeout: 10s
      retries: 3
      start_period: 60s
    networks:
      - gva-network

  # 前端服务 - 使用预构建镜像
  frontend:
    image: ${FRONTEND_IMAGE}:${TAG_LATEST}
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
      test: ["CMD", "wget", "--quiet", "--tries=1", "--spider", "http://localhost:80", "||", "exit", "1"]
      interval: 30s
      timeout: 10s
      retries: 3
      start_period: 30s
    networks:
      - gva-network
EOF

# 6. 生成服务器部署脚本
cat > deploy_production.sh << EOF
#!/bin/bash

# 生产环境部署脚本 - 使用预构建镜像
echo "🚀 开始部署生产环境"

# 1. 安装Docker（如果未安装）
if ! command -v docker &> /dev/null; then
    echo "📦 安装Docker..."
    curl -fsSL https://get.docker.com | bash -s docker --mirror Aliyun
    sudo systemctl start docker
    sudo systemctl enable docker
fi

# 2. 安装Docker Compose（如果未安装）
if ! command -v docker-compose &> /dev/null; then
    echo "📦 安装Docker Compose..."
    sudo curl -L "https://get.daocloud.io/docker/compose/releases/download/v2.24.1/docker-compose-\$(uname -s)-\$(uname -m)" -o /usr/local/bin/docker-compose
    sudo chmod +x /usr/local/bin/docker-compose
fi

# 3. 配置Docker镜像加速
sudo mkdir -p /etc/docker
sudo tee /etc/docker/daemon.json > /dev/null <<EOL
{
  "registry-mirrors": [
    "https://dockerproxy.com",
    "https://hub-mirror.c.163.com"
  ]
}
EOL
sudo systemctl restart docker

# 4. 创建必要目录
mkdir -p deploy/mysql/init
mkdir -p server/log
mkdir -p server/uploads

# 5. 停止旧服务
docker-compose -f deploy/docker-compose-production.yml down 2>/dev/null || true

# 6. 拉取最新镜像
echo "📥 拉取最新镜像..."
docker pull ${BACKEND_IMAGE}:${TAG_LATEST}
docker pull ${FRONTEND_IMAGE}:${TAG_LATEST}

# 7. 启动服务
echo "🏗️  启动服务..."
docker-compose -f deploy/docker-compose-production.yml up -d

# 8. 等待启动
sleep 30

# 9. 检查状态
echo "🔍 检查服务状态..."
docker-compose -f deploy/docker-compose-production.yml ps

echo ""
echo "🎉 部署完成！"
echo "🌐 访问地址: http://\$(curl -s ifconfig.me)"
EOF

chmod +x deploy_production.sh

# 7. 创建镜像信息文件
cat > image_info.txt << EOF
# 镜像信息
构建时间: $(date)
版本标签: ${VERSION}

后端镜像: ${BACKEND_IMAGE}:${TAG_LATEST}
前端镜像: ${FRONTEND_IMAGE}:${TAG_LATEST}

# 服务器部署命令
1. 上传 deploy_production.sh 到服务器
2. 上传 deploy/docker-compose-production.yml 到服务器
3. 上传 server/config.docker.yaml 到服务器
4. 在服务器执行: ./deploy_production.sh
EOF

echo -e "${GREEN}🎉 构建和推送完成！${NC}"
echo -e "${GREEN}================================================${NC}"
echo -e "✅ 后端镜像: ${BACKEND_IMAGE}:${TAG_LATEST}"
echo -e "✅ 前端镜像: ${FRONTEND_IMAGE}:${TAG_LATEST}"
echo -e ""
echo -e "${YELLOW}📋 服务器部署步骤:${NC}"
echo -e "1. 将以下文件上传到服务器:"
echo -e "   - deploy_production.sh"
echo -e "   - deploy/docker-compose-production.yml"
echo -e "   - server/config.docker.yaml"
echo -e ""
echo -e "2. 在服务器执行:"
echo -e "   chmod +x deploy_production.sh"
echo -e "   ./deploy_production.sh"
echo -e ""
echo -e "${GREEN}🚀 镜像已推送到 Docker Hub，服务器可以直接拉取部署！${NC}"