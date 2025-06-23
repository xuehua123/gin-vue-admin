# 🚀 1panel 部署指南 (修正版 - 推荐)

有了 `1panel` 和您项目现有的 `docker-compose.yml`，部署 `gin-vue-admin` 变得前所未有的简单。

## 准备工作

1.  **上传代码**: 将您的整个项目（包含刚刚优化好的 `deploy/docker-compose.yml`）上传到服务器。
    -   推荐路径：`/opt/apps/gin-vue-admin`

2.  **创建目录 (如果需要)**: 确保日志和上传目录存在。
    ```bash
    mkdir -p /opt/apps/gin-vue-admin/server/log
    mkdir -p /opt/apps/gin-vue-admin/server/uploads
    ```

3.  **检查配置文件**: 确保 `/server/config.docker.yaml` 文件存在。其中数据库 `host` 应该指向 `mysql`，Redis `host` 指向 `redis`，这在默认配置中已经OK了。

---

## 部署步骤

### 步骤 1: 配置 Docker 加速器

1.  登录 `1panel`，进入 **主机** -> **设置**。
2.  在 **Docker** 配置项中，填入您的镜像加速地址：
    ```
    https://mirror.ccs.tencentyun.com
    ```
3.  点击 **保存**。

### 步骤 2: 使用 Compose 文件创建应用 (核心步骤)

1.  进入 **应用** -> **应用商店**。
2.  找到 **Docker Compose**（如果未安装，请先安装）。
3.  点击 **创建应用** -> **Compose 创建**。
4.  填写信息：
    -   **名称**: `gin-vue-admin` (或其他您喜欢的名称)
    -   **工作目录**: 选择您上传代码的**根目录**，例如 `/opt/apps/gin-vue-admin`。
    -   **Compose 模板来源**: 选择 **文件路径**。
    -   **Compose 文件路径**: 填写您项目中的 `docker-compose.yml` 的**绝对路径**，例如：
        ```
        /opt/apps/gin-vue-admin/deploy/docker-compose.yml
        ```
5.  点击 **确认**。

`1panel` 现在会读取您的 `deploy/docker-compose.yml` 文件，并开始工作：拉取基础镜像、构建前端和后端... 这可能需要几分钟，您可以在日志中看到进度。

### 步骤 3: 配置网站反向代理

应用启动后，它会在 `8080` 端口上运行。为了能通过域名或IP直接访问，我们需要设置一个反向代理。

1.  进入 **网站** -> **创建网站**。
2.  选择 **反向代理** 模式。
3.  填写信息：
    -   **主域名**: 填入您的服务器IP地址或域名。
    -   **代理地址**: `http://127.0.0.1:8080`
4.  点击 **确认**。

---

## 🎉 部署完成！

现在，直接在浏览器中输入您的域名或IP地址，就可以访问您的 `gin-vue-admin` 了！

### ✨ 这个方案的优势

-   **原生部署**: 完全利用您项目自带的 `docker-compose.yml`，最正宗的部署方式。
-   **一键部署**: 所有操作都在 `1panel` 面板中完成，无需敲命令。
-   **自动构建**: `1panel` 会在服务器上完成所有编译和构建工作。
-   **轻松管理**: 启动、停止、重启、查看日志、更新，都可以在 `1panel` 中轻松搞定。
-   **国内优化**: `docker-compose.yml` 中已经为您配置好了国内的 `npm` 和 `go` 代理。

### 🔄 如何更新？

1.  在服务器上，进入项目目录，拉取最新代码：
    ```bash
    cd /opt/apps/gin-vue-admin
    git pull
    ```
2.  在 `1panel` 中，进入 **应用**，找到 `gin-vue-admin`。
3.  点击 **重建** 按钮。`1panel` 就会用最新的代码重新构建和部署您的应用。

这个方案完全贴合您项目的现有结构，是目前最简单、最现代化的部署方式，希望您喜欢！ 