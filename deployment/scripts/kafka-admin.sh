#!/bin/bash

# GoChat Kafka 管理脚本
# 用于创建和管理 Kafka Topics 和 Consumer Groups

set -e

# 默认配置
KAFKA_BROKER=${KAFKA_BROKER:-"localhost:9092,localhost:119092,localhost:29092"}
REPLICATION_FACTOR=${REPLICATION_FACTOR:-1}
PARTITIONS=${PARTITIONS:-3}

# 颜色输出
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# 日志函数
log_info() {
    echo -e "${GREEN}[INFO]${NC} $1"
}

log_warn() {
    echo -e "${YELLOW}[WARN]${NC} $1"
}

log_error() {
    echo -e "${RED}[ERROR]${NC} $1"
}

log_debug() {
    echo -e "${BLUE}[DEBUG]${NC} $1"
}

# 检查 Kafka 连接
check_kafka_connection() {
    log_info "检查 Kafka 连接: $KAFKA_BROKER"
    
    if kafka-topics.sh --bootstrap-server "$KAFKA_BROKER" --list >/dev/null 2>&1; then
        log_info "Kafka 连接成功"
        return 0
    else
        log_error "无法连接到 Kafka broker: $KAFKA_BROKER"
        log_error "请确保 Kafka 服务正在运行，或设置正确的 KAFKA_BROKER 环境变量"
        return 1
    fi
}

# 创建单个 Topic
create_topic() {
    local topic_name=$1
    local partitions=${2:-$PARTITIONS}
    local replication_factor=${3:-$REPLICATION_FACTOR}
    
    log_info "创建 Topic: $topic_name (分区: $partitions, 副本: $replication_factor)"
    
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
        --config retention.ms=604800000 \
        --config segment.ms=86400000
    
    if [ $? -eq 0 ]; then
        log_info "Topic '$topic_name' 创建成功"
    else
        log_error "Topic '$topic_name' 创建失败"
        return 1
    fi
}

# 创建所有 GoChat Topics
create_all_topics() {
    log_info "开始创建 GoChat 所有 Topics..."
    
    # 核心消息流 Topics
    create_topic "gochat.messages.upstream" 6 $REPLICATION_FACTOR
    create_topic "gochat.messages.persist" 3 $REPLICATION_FACTOR
    create_topic "gochat.tasks.fanout" 3 $REPLICATION_FACTOR
    
    # 领域事件 Topics
    create_topic "gochat.user-events" 3 $REPLICATION_FACTOR
    create_topic "gochat.message-events" 6 $REPLICATION_FACTOR
    create_topic "gochat.notifications" 3 $REPLICATION_FACTOR
    
    # 下行消息 Topics (为 3 个 gateway 实例创建)
    for i in {1..3}; do
        create_topic "gochat.messages.downstream.gateway-$i" 3 $REPLICATION_FACTOR
    done
    
    log_info "所有 Topics 创建完成"
}

# 列出所有 Topics
list_topics() {
    log_info "当前 Kafka Topics:"
    kafka-topics.sh --bootstrap-server "$KAFKA_BROKER" --list | grep "gochat\." | sort
}

# 查看 Topic 详情
describe_topic() {
    local topic_name=$1
    
    if [ -z "$topic_name" ]; then
        log_error "请提供 Topic 名称"
        return 1
    fi
    
    log_info "Topic '$topic_name' 详情:"
    kafka-topics.sh --bootstrap-server "$KAFKA_BROKER" --topic "$topic_name" --describe
}

# 删除 Topic
delete_topic() {
    local topic_name=$1
    
    if [ -z "$topic_name" ]; then
        log_error "请提供 Topic 名称"
        return 1
    fi
    
    log_warn "确认删除 Topic '$topic_name'? (y/N)"
    read -r confirmation
    
    if [[ $confirmation =~ ^[Yy]$ ]]; then
        kafka-topics.sh --bootstrap-server "$KAFKA_BROKER" --topic "$topic_name" --delete
        log_info "Topic '$topic_name' 已删除"
    else
        log_info "取消删除操作"
    fi
}

# 列出所有 Consumer Groups
list_consumer_groups() {
    log_info "当前 Consumer Groups:"
    kafka-consumer-groups.sh --bootstrap-server "$KAFKA_BROKER" --list | grep "group" | sort
}

# 查看 Consumer Group 详情
describe_consumer_group() {
    local group_name=$1
    
    if [ -z "$group_name" ]; then
        log_error "请提供 Consumer Group 名称"
        return 1
    fi
    
    log_info "Consumer Group '$group_name' 详情:"
    kafka-consumer-groups.sh --bootstrap-server "$KAFKA_BROKER" --group "$group_name" --describe
}

# 重置 Consumer Group 偏移量
reset_consumer_group_offset() {
    local group_name=$1
    local topic_name=$2
    local reset_policy=${3:-"latest"}  # earliest, latest, or specific offset
    
    if [ -z "$group_name" ] || [ -z "$topic_name" ]; then
        log_error "请提供 Consumer Group 名称和 Topic 名称"
        return 1
    fi
    
    log_warn "确认重置 Consumer Group '$group_name' 在 Topic '$topic_name' 的偏移量到 '$reset_policy'? (y/N)"
    read -r confirmation
    
    if [[ $confirmation =~ ^[Yy]$ ]]; then
        kafka-consumer-groups.sh --bootstrap-server "$KAFKA_BROKER" \
            --group "$group_name" \
            --topic "$topic_name" \
            --reset-offsets \
            --to-$reset_policy \
            --execute
        log_info "Consumer Group '$group_name' 偏移量已重置"
    else
        log_info "取消重置操作"
    fi
}

# 监控消息生产和消费情况
monitor_topics() {
    log_info "监控 GoChat Topics 消息情况 (按 Ctrl+C 退出):"
    
    local topics=(
        "gochat.messages.upstream"
        "gochat.messages.persist"
        "gochat.user-events"
        "gochat.message-events"
        "gochat.notifications"
    )
    
    while true; do
        clear
        echo "================ GoChat Kafka 监控 ================"
        echo "时间: $(date)"
        echo "Broker: $KAFKA_BROKER"
        echo "=================================================="
        
        for topic in "${topics[@]}"; do
            echo -n "[$topic] "
            # 获取 topic 的分区数和消息数（简化显示）
            kafka-log-dirs.sh --bootstrap-server "$KAFKA_BROKER" --topic-list "$topic" 2>/dev/null | \
                grep -o '"size":[0-9]*' | cut -d':' -f2 | \
                awk '{sum+=$1} END {printf "大小: %d bytes\n", sum}' 2>/dev/null || echo "N/A"
        done
        
        echo "=================================================="
        echo "Consumer Groups:"
        kafka-consumer-groups.sh --bootstrap-server "$KAFKA_BROKER" --list 2>/dev/null | \
            grep -E "(logic|gateway|task|analytics|notification)" | \
            head -5
        
        sleep 5
    done
}

# 显示帮助信息
show_help() {
    cat << EOF
GoChat Kafka 管理脚本

用法: $0 [命令] [参数]

环境变量:
  KAFKA_BROKER        Kafka broker 地址 (默认: localhost:9092)
  REPLICATION_FACTOR  副本因子 (默认: 1)
  PARTITIONS          默认分区数 (默认: 3)

命令:
  create-all          创建所有 GoChat Topics
  create <topic>      创建指定 Topic
  list                列出所有 GoChat Topics
  describe <topic>    查看 Topic 详情
  delete <topic>      删除指定 Topic
  
  list-groups         列出所有 Consumer Groups
  describe-group <group>  查看 Consumer Group 详情
  reset-offset <group> <topic> [policy]  重置 Consumer Group 偏移量
  
  monitor             监控 Topics 状态
  check               检查 Kafka 连接
  help                显示此帮助信息

示例:
  $0 create-all
  $0 create gochat.test-topic
  $0 describe gochat.messages.upstream
  $0 reset-offset logic.upstream.group gochat.messages.upstream latest
  $0 monitor

EOF
}

# 主函数
main() {
    local command=${1:-help}
    
    case $command in
        "create-all")
            check_kafka_connection && create_all_topics
            ;;
        "create")
            check_kafka_connection && create_topic "$2" "$3" "$4"
            ;;
        "list")
            check_kafka_connection && list_topics
            ;;
        "describe")
            check_kafka_connection && describe_topic "$2"
            ;;
        "delete")
            check_kafka_connection && delete_topic "$2"
            ;;
        "list-groups")
            check_kafka_connection && list_consumer_groups
            ;;
        "describe-group")
            check_kafka_connection && describe_consumer_group "$2"
            ;;
        "reset-offset")
            check_kafka_connection && reset_consumer_group_offset "$2" "$3" "$4"
            ;;
        "monitor")
            check_kafka_connection && monitor_topics
            ;;
        "check")
            check_kafka_connection
            ;;
        "help"|"-h"|"--help")
            show_help
            ;;
        *)
            log_error "未知命令: $command"
            show_help
            exit 1
            ;;
    esac
}

# 执行主函数
main "$@"