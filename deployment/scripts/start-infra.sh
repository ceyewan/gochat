#!/bin/bash
#
# 启动基础设施服务
#
# 用法:
#   ./start-infra.sh [component]
#
# 参数:
#   all         (默认) 启动所有服务: core, monitoring, admin
#   core        只启动核心服务
#   monitoring  启动核心和监控服务
#   admin       启动核心和管理工具
#
set -e

# 获取脚本所在的目录
SCRIPT_DIR=$(cd "$(dirname "${BASH_SOURCE[0]}")" &> /dev/null && pwd)
INFRA_DIR="$SCRIPT_DIR/../infrastructure"

echo "==> 切换到基础设施目录: $INFRA_DIR"
cd "$INFRA_DIR"

# 检查并生成 Kafka Cluster ID
if [ ! -f .env ] || ! grep -q 'KAFKA_CLUSTER_ID' .env; then
  echo "==> 生成新的 Kafka Cluster ID..."
  KAFKA_CLUSTER_ID=$(docker run --rm bitnami/kafka:3.5 kafka-storage.sh random-uuid)
  echo "KAFKA_CLUSTER_ID=$KAFKA_CLUSTER_ID" > .env
  echo "    -> 新的 ID: $KAFKA_CLUSTER_ID (已保存到 .env 文件)"
else
  echo "==> 使用已存在的 Kafka Cluster ID."
fi

# 定义 compose 文件
CORE_COMPOSE="-f docker-compose.yml"
MONITORING_COMPOSE="-f docker-compose.monitoring.yml"
ADMIN_COMPOSE="-f docker-compose.admin.yml"

# 根据参数组合命令
COMPONENT=${1:-all}
COMMAND="docker compose"

case "$COMPONENT" in
  core)
    echo "==> 启动核心基础设施..."
    COMMAND="$COMMAND $CORE_COMPOSE"
    ;;
  monitoring)
    echo "==> 启动核心及监控服务..."
    COMMAND="$COMMAND $CORE_COMPOSE $MONITORING_COMPOSE"
    ;;
  admin)
    echo "==> 启动核心及管理工具..."
    COMMAND="$COMMAND $CORE_COMPOSE $ADMIN_COMPOSE"
    ;;
  all|*)
    echo "==> 启动所有基础设施服务..."
    COMMAND="$COMMAND $CORE_COMPOSE $MONITORING_COMPOSE $ADMIN_COMPOSE"
    ;;
esac

# 执行启动命令
$COMMAND up -d

echo ""
echo "✅ 基础设施启动命令已执行。请使用 'docker compose ps' 查看状态。"
