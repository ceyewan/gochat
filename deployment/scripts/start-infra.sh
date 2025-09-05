#!/bin/bash
set -e

# 脚本移动到基础设施目录
cd "$(dirname "${BASH_SOURCE[0]}")/../infrastructure"

# 创建并设置 Kafka 数据目录权限
echo "创建并设置 Kafka 数据目录权限..."
mkdir -p ./data/kafka1 ./data/kafka2 ./data/kafka3
chmod -R 777 ./data/kafka1 ./data/kafka2 ./data/kafka3
echo "目录权限设置完成"

# 生成 Kafka 集群 ID
KAFKA_CLUSTER_ID=$(docker run --rm apache/kafka:latest /bin/sh -c '/opt/kafka/bin/kafka-storage.sh random-uuid')
echo "KAFKA_CLUSTER_ID=$KAFKA_CLUSTER_ID" > .env
echo "生成的 Kafka 集群 ID: $KAFKA_CLUSTER_ID"
echo "已将集群 ID 保存到 .env 文件"

# 导出集群 ID 环境变量并启动 Docker Compose
export KAFKA_CLUSTER_ID
docker-compose up -d

echo "Kafka KRaft 集群启动中..."
sleep 5
echo "可以使用以下命令查看容器状态: docker-compose ps"
