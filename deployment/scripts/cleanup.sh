#!/bin/bash
#
# 停止并清理环境
#
set -e

# 获取脚本所在的目录
SCRIPT_DIR=$(cd "$(dirname "${BASH_SOURCE[0]}")" &> /dev/null && pwd)
INFRA_DIR="$SCRIPT_DIR/../infrastructure"
APPS_DIR="$SCRIPT_DIR/../applications"

echo "==> 停止应用服务..."
cd "$APPS_DIR"
docker compose down

echo "==> 停止基础设施服务..."
cd "$INFRA_DIR"
docker compose down

echo ""
echo "✅ 所有服务已停止。"