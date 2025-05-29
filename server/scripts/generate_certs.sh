#!/bin/bash

# SSL证书生成脚本
# 用于为NFC中继系统生成自签名SSL证书

CERT_DIR="./certs"
DOMAIN="localhost"
COUNTRY="CN"
STATE="Beijing"
CITY="Beijing"
ORG="NFC Relay System"
ORG_UNIT="Security Department"

# 创建证书目录
mkdir -p "$CERT_DIR"

echo "🔐 正在生成SSL证书..."

# 生成私钥
openssl genrsa -out "$CERT_DIR/server.key" 4096

# 生成证书签名请求
openssl req -new -key "$CERT_DIR/server.key" -out "$CERT_DIR/server.csr" -subj "/C=$COUNTRY/ST=$STATE/L=$CITY/O=$ORG/OU=$ORG_UNIT/CN=$DOMAIN"

# 生成自签名证书
openssl x509 -req -days 365 -in "$CERT_DIR/server.csr" -signkey "$CERT_DIR/server.key" -out "$CERT_DIR/server.crt" \
  -extensions v3_req -extfile <(cat <<EOF
[v3_req]
keyUsage = keyEncipherment, dataEncipherment
extendedKeyUsage = serverAuth
subjectAltName = @alt_names

[alt_names]
DNS.1 = localhost
DNS.2 = *.localhost
IP.1 = 127.0.0.1
IP.2 = ::1
EOF
)

# 生成CA证书 (用于客户端认证，如果需要)
openssl genrsa -out "$CERT_DIR/ca.key" 4096
openssl req -new -x509 -days 365 -key "$CERT_DIR/ca.key" -out "$CERT_DIR/ca.crt" -subj "/C=$COUNTRY/ST=$STATE/L=$CITY/O=$ORG CA/OU=$ORG_UNIT/CN=NFC Relay CA"

# 设置文件权限
chmod 600 "$CERT_DIR"/*.key
chmod 644 "$CERT_DIR"/*.crt

# 清理临时文件
rm "$CERT_DIR/server.csr"

echo "✅ SSL证书生成完成："
echo "   服务器证书: $CERT_DIR/server.crt"
echo "   服务器私钥: $CERT_DIR/server.key" 
echo "   CA证书: $CERT_DIR/ca.crt"
echo ""
echo "🔧 配置说明："
echo "   1. 将config.yaml中的enable-tls设置为true"
echo "   2. 确保cert-file和key-file路径正确"
echo "   3. 客户端连接使用 wss:// 协议"
echo ""
echo "⚠️  注意：这是自签名证书，生产环境请使用CA签发的证书" 