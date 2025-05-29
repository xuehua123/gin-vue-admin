# NFC中继系统审计级安全架构部署指南

## 🎯 部署目标

本指南将指导您部署具备**完全审计能力**的NFC中继系统，该系统可以：
- 解密所有敏感数据进行合规检查
- 实时拦截高风险交易
- 提供完整的审计日志
- 满足金融监管要求

## 📋 系统要求

### 硬件要求
- **CPU**: 4核心或以上
- **内存**: 8GB或以上
- **存储**: 100GB SSD（用于日志存储）
- **网络**: 稳定的网络连接，支持WSS

### 软件要求
- **操作系统**: Linux/Windows Server
- **Go**: 1.19或以上版本
- **数据库**: MySQL 8.0或以上
- **SSL证书**: 用于WSS连接

## 🔧 配置步骤

### 1. 更新配置文件

编辑 `config.yaml` 文件，添加审计级安全配置：

```yaml
nfc-relay:
  websocket-pong-wait-sec: 60
  websocket-max-message-bytes: 8192
  websocket-write-wait-sec: 10
  hub-check-interval-sec: 60
  session-inactive-timeout-sec: 300
  
  security:
    # TLS配置
    enable-tls: true
    force-tls: true
    cert-file: "./certs/server.crt"
    key-file: "./certs/server.key"
    
    # 审计级加密配置
    enable-audit-encryption: true
    encryption-algorithm: "AES-256-GCM"
    audit-key-rotation-hours: 24
    
    # 合规审计配置
    enable-compliance-audit: true
    enable-deep-inspection: true
    max-transaction-amount: 1000000  # 10000元 (分为单位)
    blocked-merchant-categories: 
      - "GAMBLING"
      - "ADULT"
      - "TOBACCO"
      - "WEAPONS"
    
    # 防重放攻击配置
    enable-anti-replay: true
    replay-window-ms: 30000
    max-nonce-cache: 10000
    
    # 输入验证配置
    max-message-size: 65536
    enable-input-sanitize: true
    
    # 客户端认证配置
    require-client-cert: false
    client-ca-file: ""
```

### 2. 生成SSL证书

如果还没有SSL证书，可以使用以下脚本生成自签名证书：

```bash
#!/bin/bash
# generate_certs.sh

# 创建证书目录
mkdir -p certs

# 生成私钥
openssl genrsa -out certs/server.key 2048

# 生成证书签名请求
openssl req -new -key certs/server.key -out certs/server.csr -subj "/C=CN/ST=Beijing/L=Beijing/O=YourCompany/CN=localhost"

# 生成自签名证书
openssl x509 -req -days 365 -in certs/server.csr -signkey certs/server.key -out certs/server.crt

# 设置权限
chmod 600 certs/server.key
chmod 644 certs/server.crt

echo "SSL证书生成完成！"
echo "证书文件: certs/server.crt"
echo "私钥文件: certs/server.key"
```

运行脚本：
```bash
chmod +x generate_certs.sh
./generate_certs.sh
```

### 3. 数据库初始化

确保数据库中存在必要的审计日志表。如果没有，运行以下SQL：

```sql
-- 创建审计日志表
CREATE TABLE IF NOT EXISTS audit_logs (
    id BIGINT AUTO_INCREMENT PRIMARY KEY,
    timestamp DATETIME NOT NULL,
    event_type VARCHAR(100) NOT NULL,
    session_id VARCHAR(100),
    client_id_initiator VARCHAR(100),
    client_id_responder VARCHAR(100),
    source_ip VARCHAR(45),
    user_id VARCHAR(100),
    details JSON,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    INDEX idx_timestamp (timestamp),
    INDEX idx_event_type (event_type),
    INDEX idx_session_id (session_id),
    INDEX idx_user_id (user_id)
);

-- 创建合规违规记录表
CREATE TABLE IF NOT EXISTS compliance_violations (
    id BIGINT AUTO_INCREMENT PRIMARY KEY,
    user_id VARCHAR(100) NOT NULL,
    session_id VARCHAR(100),
    command_class VARCHAR(50),
    reason TEXT,
    risk_level VARCHAR(20),
    actions TEXT,
    timestamp DATETIME NOT NULL,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    INDEX idx_user_id (user_id),
    INDEX idx_timestamp (timestamp),
    INDEX idx_risk_level (risk_level)
);
```

## 🚀 启动服务

### 1. 编译项目

```bash
go build -o nfc_server main.go
```

### 2. 启动服务器

```bash
# 开发环境
./nfc_server

# 生产环境（后台运行）
nohup ./nfc_server > logs/server.log 2>&1 &
```

### 3. 验证启动

检查服务启动日志：
```bash
tail -f logs/server.log
```

应该看到类似以下的日志：
```
INFO    NFC中继系统启动
INFO    审计级安全管理器初始化完成
INFO    TLS配置已启用: cert=./certs/server.crt, key=./certs/server.key
INFO    合规审计引擎启动，启用规则: 5个
INFO    WebSocket服务器启动在端口: 8080 (WSS)
```

## 📊 监控和验证

### 1. 健康检查

访问以下端点检查服务状态：
- `https://your-domain:8080/health` - 服务健康状态
- `https://your-domain:8080/metrics` - Prometheus指标

### 2. 测试安全功能

运行安全测试：
```go
package main

import (
    "github.com/flipped-aurora/gin-vue-admin/server/nfc_relay/security"
)

func main() {
    // 运行所有安全测试
    security.RunAllTests()
}
```

### 3. 监控关键指标

设置监控以下关键指标：

```yaml
# prometheus.yml
global:
  scrape_interval: 15s

scrape_configs:
  - job_name: 'nfc-relay'
    static_configs:
      - targets: ['localhost:8080']
    metrics_path: '/metrics'
    scheme: 'https'
    tls_config:
      insecure_skip_verify: true
```

关键指标：
- `active_connections` - 活跃连接数
- `active_sessions` - 活跃会话数
- `apdu_messages_relayed` - APDU消息转发数
- `compliance_violations` - 合规违规次数
- `security_blocks` - 安全拦截次数

## 🛡️ 安全最佳实践

### 1. 日志管理

设置日志轮换：
```bash
# /etc/logrotate.d/nfc-relay
/path/to/nfc-relay/logs/*.log {
    daily
    rotate 30
    compress
    delaycompress
    missingok
    notifempty
    create 644 nfc-user nfc-group
}
```

### 2. 防火墙配置

```bash
# 只开放必要端口
ufw allow 8080/tcp  # NFC中继WebSocket端口
ufw allow 22/tcp    # SSH
ufw deny incoming
ufw allow outgoing
ufw enable
```

### 3. 系统监控

设置系统级监控：
```bash
# CPU和内存监控
top -p $(pgrep nfc_server)

# 网络连接监控
netstat -tulpn | grep :8080

# 磁盘空间监控（日志目录）
df -h /path/to/logs
```

## 🔍 故障排除

### 常见问题

**1. SSL证书错误**
```
ERROR: TLS握手失败
```
解决方案：
- 检查证书文件路径
- 验证证书有效性：`openssl x509 -in certs/server.crt -text -noout`
- 确保私钥匹配：`openssl rsa -in certs/server.key -check`

**2. 合规检查失败**
```
ERROR: APDU数据不符合合规要求
```
解决方案：
- 检查配置中的限额设置
- 查看审计日志确定具体违规原因
- 调整合规规则配置

**3. 性能问题**
```
WARNING: APDU处理延迟过高
```
解决方案：
- 增加服务器资源
- 优化数据库索引
- 检查网络延迟

### 日志分析

查看关键日志：
```bash
# 安全相关日志
grep "🚨\|🛡️\|SECURITY" logs/server.log

# 合规违规日志
grep "compliance_violation" logs/server.log

# 性能监控日志
grep "延迟\|latency" logs/server.log
```

## 📈 性能优化

### 1. 数据库优化

```sql
-- 添加适当索引
ALTER TABLE audit_logs ADD INDEX idx_composite (event_type, timestamp, user_id);

-- 配置数据库连接池
SET GLOBAL max_connections = 200;
SET GLOBAL innodb_buffer_pool_size = 2G;
```

### 2. 应用优化

```go
// 在main.go中配置
func main() {
    // 设置Go runtime参数
    runtime.GOMAXPROCS(runtime.NumCPU())
    
    // 配置垃圾回收
    debug.SetGCPercent(100)
}
```

### 3. 系统优化

```bash
# 增加文件描述符限制
echo "* soft nofile 65536" >> /etc/security/limits.conf
echo "* hard nofile 65536" >> /etc/security/limits.conf

# 优化TCP参数
echo 'net.core.somaxconn = 8192' >> /etc/sysctl.conf
echo 'net.ipv4.tcp_max_syn_backlog = 8192' >> /etc/sysctl.conf
sysctl -p
```

## 📝 维护指南

### 定期维护任务

**每日任务**：
- 检查服务状态
- 查看错误日志
- 监控资源使用

**每周任务**：
- 轮换日志文件
- 备份配置文件
- 更新安全规则

**每月任务**：
- 更新SSL证书（如需要）
- 数据库性能分析
- 安全漏洞扫描

### 备份策略

```bash
#!/bin/bash
# backup.sh

DATE=$(date +%Y%m%d_%H%M%S)

# 备份配置文件
tar -czf backup/config_$DATE.tar.gz config/

# 备份SSL证书
tar -czf backup/certs_$DATE.tar.gz certs/

# 备份数据库（包含审计日志）
mysqldump -u user -p database_name > backup/db_$DATE.sql

echo "备份完成: $DATE"
```

## 🎉 总结

现在您的NFC中继系统已经具备了：

✅ **完全审计能力** - 服务器可解密所有数据  
✅ **实时合规检查** - 自动拦截违规交易  
✅ **安全传输保护** - WSS + 审计级加密  
✅ **详细监控日志** - 全面的审计跟踪  
✅ **高性能架构** - 优化的处理流程  

系统现在已经准备好处理生产环境的NFC中继交易，并满足严格的金融监管要求！ 