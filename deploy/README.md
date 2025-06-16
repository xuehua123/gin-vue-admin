# 🚀 超简单部署指南

**只需要：服务器 + GitHub，不需要其他任何服务！**

## 📋 只需要在GitHub设置3个密码

1. 打开您的GitHub项目
2. 点击 **Settings** → **Secrets and variables** → **Actions**  
3. 点击 **New repository secret**，添加以下3个：

```
名称：SERVER_HOST
值：您服务器的IP地址（比如：123.456.789.123）

名称：SERVER_USER  
值：root（或者您的服务器用户名）

名称：SERVER_PASSWORD
值：您服务器的登录密码
```

就这3个，完了！

## 🎯 一键部署

设置好后，每次您推送代码：

```bash
git add .
git commit -m "更新代码"
git push
```

GitHub就会自动帮您部署到服务器！

## 🌐 访问您的网站

部署完成后，在浏览器输入：`http://您的服务器IP`

## 🛠️ 手动部署（如果不想用自动部署）

在您的服务器上运行：

```bash
# 1. 安装Docker（只需要运行一次）
curl -fsSL https://get.docker.com | bash

# 2. 克隆项目
git clone https://github.com/您的用户名/gin-vue-admin.git
cd gin-vue-admin

# 3. 启动
sudo docker-compose -f deploy/docker-compose.yml up -d --build
```

## 🔧 管理命令

```bash
# 复制管理脚本到系统路径（只需要运行一次）
sudo cp deploy/manage.sh /usr/local/bin/gva
sudo chmod +x /usr/local/bin/gva

# 然后就可以用简单命令管理了：
gva start     # 启动服务
gva stop      # 停止服务
gva restart   # 重启服务
gva status    # 查看状态
gva logs      # 查看日志
```

## 🎉 就这么简单！

- **前端**：`http://您的服务器IP`
- **后端API**：`http://您的服务器IP:8888`
- **数据库**：MySQL（端口3306）
- **缓存**：Redis（端口6379）

有问题就问我！😊 