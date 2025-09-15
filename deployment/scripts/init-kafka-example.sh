#!/bin/bash

# Kafka Example 测试环境初始化脚本
# 为 GoChat Kafka 组件的 example 创建必要的 topics

set -e

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
KAFKA_ADMIN_SCRIPT="$SCRIPT_DIR/kafka-admin.sh"

# 颜色输出
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m'

log_info() {
    echo -e "${GREEN}[EXAMPLE]${NC} $1"
}

log_warn() {
    echo -e "${YELLOW}[WARN]${NC} $1"
}

log_debug() {
    echo -e "${BLUE}[DEBUG]${NC} $1"
}

# 检查脚本是否存在
if [ ! -f "$KAFKA_ADMIN_SCRIPT" ]; then
    echo "错误: 找不到 kafka-admin.sh 脚本: $KAFKA_ADMIN_SCRIPT"
    exit 1
fi

# 使脚本可执行
chmod +x "$KAFKA_ADMIN_SCRIPT"

log_info "开始初始化 Kafka Example 测试环境..."

# 等待 Kafka 服务启动
log_info "等待 Kafka 服务启动..."
for i in {1..15}; do
    if "$KAFKA_ADMIN_SCRIPT" check >/dev/null 2>&1; then
        log_info "Kafka 服务已就绪"
        break
    fi

    if [ $i -eq 15 ]; then
        echo "错误: Kafka 服务启动超时"
        exit 1
    fi

    echo "等待 Kafka 启动... ($i/15)"
    sleep 2
done

# 创建 Example 测试 Topics
log_info "创建 Example 测试 Topics..."

# 基础测试 Topic
create_example_topic() {
    local topic_name=$1
    local partitions=${2:-1}
    local replication_factor=${3:-1}

    log_debug "创建测试 Topic: $topic_name (分区: $partitions, 副本: $replication_factor)"

    if kafka-topics.sh --bootstrap-server "$KAFKA_BROKER" \
        --topic "$topic_name" \
        --describe >/dev/null 2>&1; then
        log_warn "Topic '$topic_name' 已存在，跳过创建"
        return 0
    fi

    kafka-topics.sh --bootstrap-server "$KAFKA_BROKER" \
        --create \
        --topic "$topic_name" \
        --partitions "$partitions" \
        --replication-factor "$replication_factor" \
        --config retention.ms=86400000 \
        --config segment.ms=3600000

    if [ $? -eq 0 ]; then
        log_info "测试 Topic '$topic_name' 创建成功"
    else
        log_error "测试 Topic '$topic_name' 创建失败"
        return 1
    fi
}

# 创建 Example 相关的 Topics
create_example_topic "example.user.events" 3 1
create_example_topic "example.test-topic" 1 1
create_example_topic "example.performance" 6 1
create_example_topic "example.dead-letter" 1 1

# 显示创建结果
log_info "验证 Example Topics 创建结果:"
kafka-topics.sh --bootstrap-server "$KAFKA_BROKER" --list | grep "example\." | sort

log_info "Kafka Example 测试环境初始化完成!"
log_warn "提示: 现在可以运行 example 程序了:"
log_warn "  cd /Users/harrick/CodeField/gochat/im-infra/kafka/examples && go run main.go"
log_warn "提示: 使用 '$KAFKA_ADMIN_SCRIPT monitor' 可以监控 Topics 状态"

# 显示一些有用的命令
echo ""
log_info "有用的命令:"
echo "  查看所有 topics:      $KAFKA_ADMIN_SCRIPT list"
echo "  查看特定 topic:       $KAFKA_ADMIN_SCRIPT describe example.user.events"
echo "  监控 topics:         $KAFKA_ADMIN_SCRIPT monitor"
echo "  删除测试 topics:      $KAFKA_ADMIN_SCRIPT delete example.user.events"