# 脚本使用说明

## EMQX配置脚本

### 远程EMQX配置 (推荐)

#### PowerShell版本 (Windows推荐)

如果您在Windows上遇到WSL问题，推荐使用PowerShell版本：

```powershell
# 推荐方式 (绕过执行策略限制)
PowerShell -ExecutionPolicy Bypass -File .\scripts\setup_emqx.ps1 setup

# 或直接运行 (如果执行策略允许)
.\scripts\setup_emqx.ps1 setup
```

#### Bash版本 (Linux/macOS)

```bash
# Linux/macOS/WSL
chmod +x scripts/emqx_remote_setup.sh
./scripts/emqx_remote_setup.sh setup
```

### 连接测试

使用 `test_emqx_connection.py` 测试远程EMQX连接。

```bash
python scripts/test_emqx_connection.py
```

## 远程EMQX实例信息

- **地址**: 49.235.40.39
- **Dashboard**: http://49.235.40.39:18083
- **用户名**: admin  
- **密码**: xuehua123
- **MQTT端口**: 1883 (TCP), 8883 (SSL)
- **WebSocket**: 8083 (WS), 8084 (WSS)

## 脚本功能

### setup_emqx.ps1 (PowerShell版本)

- `setup` - 配置远程EMQX实例 (包含验证功能)
- `test` - 测试连接和配置
- `info` - 显示连接信息
- `help` - 显示帮助信息

### emqx_remote_setup.sh (Bash版本)

- `setup` - 配置远程EMQX实例 (包含验证功能)
- `test` - 测试连接和配置
- `info` - 显示连接信息
- `help` - 显示帮助信息

### test_emqx_connection.py

- 测试远程EMQX实例连接性
- 验证API端点可用性
- 生成连接信息JSON文件

## 注意事项

1. **Windows用户**：推荐使用PowerShell版本脚本，避免WSL路径问题
2. **Linux/macOS用户**：使用Bash版本脚本
3. 确保网络可以访问远程EMQX实例
4. 如果使用代理，请配置相应的环境变量

## 故障排除

### WSL路径翻译错误

如果遇到类似以下错误：
```
WSL (11) ERROR: UtilTranslatePathList:2866: Failed to translate...
```

**解决方案**：使用PowerShell版本脚本
```powershell
PowerShell -ExecutionPolicy Bypass -File .\scripts\setup_emqx.ps1 setup
```

### PowerShell执行策略限制

如果遇到执行策略错误，使用以下命令：
```powershell
Set-ExecutionPolicy -ExecutionPolicy RemoteSigned -Scope CurrentUser
# 或者临时绕过
PowerShell -ExecutionPolicy Bypass -File .\scripts\setup_emqx.ps1 setup
``` 