#!/bin/bash

# GoChat Kafka 快速初始化脚本
# 在启动应用之前初始化所有必要的 Kafka Topics

set -e

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
KAFKA_ADMIN_SCRIPT="$SCRIPT_DIR/kafka-admin.sh"

# 颜色输出
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m'

log_info() {
    echo -e "${GREEN}[INIT]${NC} $1"
}

log_warn() {
    echo -e "${YELLOW}[WARN]${NC} $1"
}

# 检查脚本是否存在
if [ ! -f "$KAFKA_ADMIN_SCRIPT" ]; then
    echo "错误: 找不到 kafka-admin.sh 脚本: $KAFKA_ADMIN_SCRIPT"
    exit 1
fi

# 使脚本可执行
chmod +x "$KAFKA_ADMIN_SCRIPT"

log_info "开始初始化 GoChat Kafka Topics..."

# 等待 Kafka 服务启动
log_info "等待 Kafka 服务启动..."
for i in {1..30}; do
    if "$KAFKA_ADMIN_SCRIPT" check >/dev/null 2>&1; then
        log_info "Kafka 服务已就绪"
        break
    fi
    
    if [ $i -eq 30 ]; then
        echo "错误: Kafka 服务启动超时"
        exit 1
    fi
    
    echo "等待 Kafka 启动... ($i/30)"
    sleep 2
done

# 创建所有 Topics
log_info "创建 GoChat Topics..."
"$KAFKA_ADMIN_SCRIPT" create-all

# 验证创建结果
log_info "验证 Topics 创建结果:"
"$KAFKA_ADMIN_SCRIPT" list

log_info "GoChat Kafka 初始化完成!"
log_warn "提示: 使用 '$KAFKA_ADMIN_SCRIPT monitor' 可以监控 Topics 状态"