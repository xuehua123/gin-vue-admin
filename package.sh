#!/bin/bash

# 在命令失败时立即退出
set -e

# 定义打包文件名
FILENAME="gva-source-$(date +'%Y%m%d-%H%M%S').tar.gz"

echo "📦 开始打包项目源码..."
echo "========================================"
echo "将排除以下文件/目录 (由 .dockerignore 和脚本定义):"
echo " - 版本控制 (.git)"
echo " - IDE 配置 (.idea, .vscode)"
echo " - 依赖目录 (node_modules)"
echo " - 构建产物 (dist, a.out, *.exe)"
echo " - 日志和上传文件"
echo " - 本地部署脚本和压缩包"
echo "========================================"

# 使用 tar 命令进行打包和压缩
# --exclude-vcs: 排除版本控制系统文件，如 .git
# -czvf: c(创建归档), z(使用gzip压缩), v(显示过程), f(指定文件名)
tar \
  --exclude-vcs \
  --exclude-from='.dockerignore' \
  --exclude='*.tar.gz' \
  --exclude='package.sh' \
  --exclude='package.bat' \
  -czvf "$FILENAME" .

echo ""
echo "✅ 打包成功！"
echo "📦 文件名: $FILENAME"
echo "   文件大小: $(du -sh $FILENAME | awk '{print $1}')"
echo ""
echo "🚀 您现在可以将此文件上传到服务器并在1panel中使用了。" 