# SSH密钥部署设置指南

## 🔑 第1步：生成SSH密钥对

在你的本地电脑上运行：

```bash
# 生成专用于部署的SSH密钥对
ssh-keygen -t rsa -b 4096 -C "gin-vue-admin-deploy" -f gin-vue-admin-key

# 输入时直接回车，不设置密码（用于自动化部署）
```

这会生成两个文件：
- `gin-vue-admin-key` - 私钥（保密）
- `gin-vue-admin-key.pub` - 公钥（上传到服务器）

## 🖥️ 第2步：配置服务器

### 2.1 上传公钥到服务器

```bash
# 方法1：使用ssh-copy-id（推荐）
ssh-copy-id -i gin-vue-admin-key.pub root@your-server-ip

# 方法2：手动复制
# 先复制公钥内容
cat gin-vue-admin-key.pub

# 然后登录服务器，添加到authorized_keys
ssh root@your-server-ip
mkdir -p ~/.ssh
chmod 700 ~/.ssh
echo "公钥内容" >> ~/.ssh/authorized_keys
chmod 600 ~/.ssh/authorized_keys
```

### 2.2 测试SSH连接

```bash
# 测试密钥是否工作
ssh -i gin-vue-admin-key root@your-server-ip

# 如果能免密登录，说明设置成功
```

## 🔐 第3步：配置GitHub Secrets

在GitHub项目中设置以下Secrets：

### 必需的Secrets：

1. **SSH_PRIVATE_KEY**
   ```bash
   # 复制私钥内容
   cat gin-vue-admin-key
   ```
   复制完整内容（包括 `-----BEGIN OPENSSH PRIVATE KEY-----` 和 `-----END OPENSSH PRIVATE KEY-----`）

2. **SERVER_HOST**
   ```
   your-server-ip
   ```

3. **SERVER_USER**
   ```
   root
   ```

4. **SERVER_PORT**（可选，默认22）
   ```
   22
   ```

### GitHub Secrets设置步骤：

1. 进入GitHub项目页面
2. 点击 `Settings` → `Secrets and variables` → `Actions`
3. 点击 `New repository secret`
4. 逐个添加上述secrets

## 🚀 第4步：部署流程

设置完成后，推送代码即可自动部署：

```bash
git add .
git commit -m "启用SSH密钥部署"
git push origin dev-mqtt
```

## 🔧 故障排除

### 常见问题：

1. **权限错误**
   ```bash
   # 检查服务器权限
   chmod 700 ~/.ssh
   chmod 600 ~/.ssh/authorized_keys
   ```

2. **密钥格式错误**
   - 确保复制完整的私钥内容
   - 包括开始和结束的标记行

3. **连接测试失败**
   ```bash
   # 调试连接
   ssh -v -i gin-vue-admin-key root@your-server-ip
   ```

### 安全最佳实践：

1. ✅ **专用密钥** - 只用于部署，不用于其他用途
2. ✅ **定期轮换** - 定期更新密钥
3. ✅ **最小权限** - 只给部署必需的权限
4. ✅ **删除旧密钥** - 生成新密钥后删除本地文件

## 📋 验证清单

- [ ] SSH密钥对已生成
- [ ] 公钥已上传到服务器
- [ ] 私钥已添加到GitHub Secrets
- [ ] 其他必要的Secrets已设置
- [ ] 本地可以免密SSH登录服务器
- [ ] GitHub Actions可以访问所有Secrets

## 🎉 优势

相比密码认证，SSH密钥具有：

- ✅ **更安全** - 密钥比密码更难破解
- ✅ **更稳定** - 不会因密码策略变化而失效
- ✅ **更方便** - 无需在多个地方同步密码
- ✅ **可审计** - 更容易追踪访问记录 