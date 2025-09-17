#!/bin/bash

# Kafka 连接测试脚本

set -e

# 默认配置
KAFKA_BROKER=${KAFKA_BROKER:-"localhost:9092,localhost:119092,localhost:29092"}

# 颜色输出
GREEN='\033[0;32m'
RED='\033[0;31m'
YELLOW='\033[1;33m'
NC='\033[0m'

log_info() {
    echo -e "${GREEN}[INFO]${NC} $1"
}

log_error() {
    echo -e "${RED}[ERROR]${NC} $1"
}

log_warn() {
    echo -e "${YELLOW}[WARN]${NC} $1"
}

log_info "测试 Kafka 连接: $KAFKA_BROKER"

# 测试基本连接
if kafka-topics.sh --bootstrap-server "$KAFKA_BROKER" --list >/dev/null 2>&1; then
    log_info "✅ Kafka 连接成功"
else
    log_error "❌ Kafka 连接失败"
    log_error "请确保 Kafka 服务在以下端口运行: 9092, 119092, 29092"
    exit 1
fi

# 列出现有 topics
log_info "📋 现有 Topics:"
kafka-topics.sh --bootstrap-server "$KAFKA_BROKER" --list 2>/dev/null | grep -E "(example|gochat)" || echo "  无相关 topics"

# 测试创建 example topic
TOPIC_NAME="example.test-connection"
log_info "🔧 创建测试 Topic: $TOPIC_NAME"

if kafka-topics.sh --bootstrap-server "$KAFKA_BROKER" \
    --create \
    --topic "$TOPIC_NAME" \
    --partitions 1 \
    --replication-factor 1 \
    --config retention.ms=86400000 >/dev/null 2>&1; then
    log_info "✅ Topic 创建成功"
else
    log_error "❌ Topic 创建失败"
    exit 1
fi

# 验证 topic 是否创建成功
if kafka-topics.sh --bootstrap-server "$KAFKA_BROKER" \
    --topic "$TOPIC_NAME" \
    --describe >/dev/null 2>&1; then
    log_info "✅ Topic 验证成功"
else
    log_error "❌ Topic 验证失败"
    exit 1
fi

# 清理测试 topic
log_info "🧹 清理测试 Topic"
kafka-topics.sh --bootstrap-server "$KAFKA_BROKER" \
    --delete \
    --topic "$TOPIC_NAME" >/dev/null 2>&1 || true

log_info "🎉 Kafka 连接测试完成！"
log_info "现在可以运行 example: cd /Users/harrick/CodeField/gochat/im-infra/kafka/examples && go run main.go"