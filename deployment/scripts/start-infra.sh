#!/bin/bash
#
# 启动所有基础设施服务
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

echo "==> 使用 'docker compose' 启动所有基础设施服务..."
docker compose up -d

echo ""
echo "✅ 基础设施启动命令已执行。请使用 'docker compose ps' 查看状态。"
