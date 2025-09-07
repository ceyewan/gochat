#!/bin/bash
#
# 启动所有应用服务
#
set -e

# 获取脚本所在的目录
SCRIPT_DIR=$(cd "$(dirname "${BASH_SOURCE[0]}")" &> /dev/null && pwd)
APPS_DIR="$SCRIPT_DIR/../applications"

echo "==> 切换到应用服务目录: $APPS_DIR"
cd "$APPS_DIR"

echo "==> 使用 'docker compose' 启动所有应用服务..."
docker compose up -d

echo ""
echo "✅ 应用服务启动命令已执行。请使用 'docker compose ps' 查看状态。"