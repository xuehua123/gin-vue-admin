# 构建阶段
FROM node:18-alpine AS builder

# 设置工作目录
WORKDIR /app

# 设置Node.js环境变量优化构建性能
ENV NODE_OPTIONS="--max-old-space-size=2048"
ENV NODE_ENV=production

# 复制 package 文件
COPY web/package*.json ./

# 安装所有依赖（包括开发依赖，构建需要）
RUN npm install && npm cache clean --force

# 复制源代码和环境配置
COPY web/ .

# 构建应用（设置超时和优化参数）
RUN npm run build --timeout=600000

# 生产阶段
FROM nginx:alpine

# 维护者信息
LABEL MAINTAINER="gin-vue-admin"

# 安装时区数据
RUN apk add --no-cache tzdata

# 设置时区
ENV TZ=Asia/Shanghai
RUN ln -sf /usr/share/zoneinfo/$TZ /etc/localtime && echo $TZ > /etc/timezone

# 复制构建好的文件到 nginx 静态文件目录
COPY --from=builder /app/dist /usr/share/nginx/html

# 暴露端口
EXPOSE 80

# 启动 nginx
CMD ["nginx", "-g", "daemon off;"]
