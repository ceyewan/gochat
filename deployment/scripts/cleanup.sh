#!/bin/bash
#
# 停止并清理环境
#
# 用法:
#   ./cleanup.sh [component]
#
# 参数:
#   all         (默认) 停止所有服务: apps, core, monitoring, admin
#   core        只停止核心服务
#   monitoring  只停止监控服务
#   admin       只停止管理工具
#   apps        只停止应用服务
#
set -e

# 获取脚本所在的目录
SCRIPT_DIR=$(cd "$(dirname "${BASH_SOURCE[0]}")" &> /dev/null && pwd)
INFRA_DIR="$SCRIPT_DIR/../infrastructure"
APPS_DIR="$SCRIPT_DIR/../applications"

# 定义 compose 文件
CORE_COMPOSE="-f $INFRA_DIR/docker-compose.yml"
MONITORING_COMPOSE="-f $INFRA_DIR/docker-compose.monitoring.yml"
ADMIN_COMPOSE="-f $INFRA_DIR/docker-compose.admin.yml"
APPS_COMPOSE="-f $APPS_DIR/docker-compose.yml"

# 根据参数组合命令
COMPONENT=${1:-all}

case "$COMPONENT" in
  core)
    echo "==> 停止核心基础设施..."
    docker compose $CORE_COMPOSE down
    ;;
  monitoring)
    echo "==> 停止监控服务..."
    docker compose $MONITORING_COMPOSE down
    ;;
  admin)
    echo "==> 停止管理工具..."
    docker compose $ADMIN_COMPOSE down
    ;;
  apps)
    echo "==> 停止应用服务..."
    docker compose $APPS_COMPOSE down
    ;;
  all|*)
    echo "==> 停止所有应用和基础设施服务..."
    echo "  -> 停止应用..."
    docker compose $APPS_COMPOSE down
    echo "  -> 停止基础设施..."
    docker compose -f "$INFRA_DIR/docker-compose.yml" -f "$INFRA_DIR/docker-compose.monitoring.yml" -f "$INFRA_DIR/docker-compose.admin.yml" down
    ;;
esac

echo ""
echo "✅ 清理命令已执行。"