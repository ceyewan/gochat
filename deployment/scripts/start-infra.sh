#!/bin/bash

# GoChat 基础设施启动脚本
# 用于启动所有基础设施组件：etcd、kafka、mysql、redis 等

set -e  # 遇到错误立即退出

# 脚本配置
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
DEPLOYMENT_DIR="$(dirname "$SCRIPT_DIR")"
INFRA_DIR="$DEPLOYMENT_DIR/infrastructure"

# 颜色输出
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# 日志函数
log_info() {
    echo -e "${BLUE}[INFO]${NC} $1"
}

log_success() {
    echo -e "${GREEN}[SUCCESS]${NC} $1"
}

log_warning() {
    echo -e "${YELLOW}[WARNING]${NC} $1"
}

log_error() {
    echo -e "${RED}[ERROR]${NC} $1"
}

# 检查 Docker 和 Docker Compose
check_prerequisites() {
    log_info "检查系统依赖..."
    
    if ! command -v docker &> /dev/null; then
        log_error "Docker 未安装，请先安装 Docker"
        exit 1
    fi
    
    if ! command -v docker-compose &> /dev/null; then
        log_error "Docker Compose 未安装，请先安装 Docker Compose"
        exit 1
    fi
    
    # 检查 Docker 是否运行
    if ! docker info &> /dev/null; then
        log_error "Docker 服务未运行，请启动 Docker 服务"
        exit 1
    fi
    
    log_success "系统依赖检查通过"
}

# 创建必要的目录
create_directories() {
    log_info "创建必要的目录..."
    
    # 创建数据目录
    mkdir -p "$INFRA_DIR/data/etcd"{1,2,3}
    mkdir -p "$INFRA_DIR/data/kafka"{1,2,3}
    mkdir -p "$INFRA_DIR/data/mysql"
    mkdir -p "$INFRA_DIR/data/redis"
    
    # 创建日志目录
    mkdir -p "$INFRA_DIR/logs/mysql"
    mkdir -p "$INFRA_DIR/logs/redis"
    
    # 设置权限
    chmod -R 755 "$INFRA_DIR/data"
    chmod -R 755 "$INFRA_DIR/logs"
    
    log_success "目录创建完成"
}

# 生成环境变量文件
generate_env_file() {
    log_info "生成环境变量文件..."
    
    cat > "$INFRA_DIR/.env" << EOF
# GoChat 基础设施环境变量
COMPOSE_PROJECT_NAME=gochat-infra
COMPOSE_FILE=docker-compose.yml

# Kafka 集群 ID
KAFKA_CLUSTER_ID=gochat-kafka-cluster-2024

# 时区设置
TZ=Asia/Shanghai

# 数据目录
DATA_DIR=./data
LOGS_DIR=./logs
CONFIG_DIR=./config
EOF
    
    log_success "环境变量文件生成完成"
}

# 启动基础设施服务
start_infrastructure() {
    log_info "启动基础设施服务..."
    
    cd "$INFRA_DIR"
    
    # 拉取最新镜像
    log_info "拉取 Docker 镜像..."
    docker-compose pull
    
    # 启动服务
    log_info "启动基础设施容器..."
    docker-compose up -d
    
    log_success "基础设施服务启动完成"
}

# 等待服务就绪
wait_for_services() {
    log_info "等待服务就绪..."
    
    # 等待 etcd 集群就绪
    log_info "等待 etcd 集群就绪..."
    for i in {1..30}; do
        if docker exec gochat-etcd1 etcdctl endpoint health --endpoints=http://localhost:2379 &> /dev/null; then
            log_success "etcd 集群就绪"
            break
        fi
        if [ $i -eq 30 ]; then
            log_error "etcd 集群启动超时"
            exit 1
        fi
        sleep 5
    done
    
    # 等待 MySQL 就绪
    log_info "等待 MySQL 就绪..."
    for i in {1..30}; do
        if docker exec gochat-mysql mysqladmin ping -h localhost -u root -pgochat_root_2024 &> /dev/null; then
            log_success "MySQL 就绪"
            break
        fi
        if [ $i -eq 30 ]; then
            log_error "MySQL 启动超时"
            exit 1
        fi
        sleep 5
    done
    
    # 等待 Redis 就绪
    log_info "等待 Redis 就绪..."
    for i in {1..30}; do
        if docker exec gochat-redis redis-cli ping | grep -q PONG; then
            log_success "Redis 就绪"
            break
        fi
        if [ $i -eq 30 ]; then
            log_error "Redis 启动超时"
            exit 1
        fi
        sleep 5
    done
    
    # 等待 Kafka 集群就绪
    log_info "等待 Kafka 集群就绪..."
    for i in {1..60}; do
        if docker exec gochat-kafka1 kafka-broker-api-versions.sh --bootstrap-server localhost:9092 &> /dev/null; then
            log_success "Kafka 集群就绪"
            break
        fi
        if [ $i -eq 60 ]; then
            log_error "Kafka 集群启动超时"
            exit 1
        fi
        sleep 5
    done
}

# 显示服务状态
show_status() {
    log_info "基础设施服务状态："
    
    cd "$INFRA_DIR"
    docker-compose ps
    
    echo ""
    log_info "服务访问地址："
    echo "  etcd 管理界面:      http://localhost:8081"
    echo "  Kafka UI:          http://localhost:8080"
    echo "  RedisInsight:      http://localhost:8001"
    echo "  phpMyAdmin:        http://localhost:8083"
    echo ""
    echo "  监控和日志界面："
    echo "  Grafana:           http://localhost:3000 (admin/gochat_grafana_2024)"
    echo "  Prometheus:        http://localhost:9090"
    echo "  Loki API:          http://localhost:3100"
    echo "  Vector API:        http://localhost:8686"
    echo "  Jaeger:            http://localhost:16686"
    echo ""
    echo "  etcd 端点:         localhost:2379,localhost:2389,localhost:2399"
    echo "  Kafka 端点:        localhost:19092,localhost:29092,localhost:39092"
    echo "  MySQL 端点:        localhost:3306"
    echo "  Redis 端点:        localhost:6379"
}

# 主函数
main() {
    log_info "开始启动 GoChat 基础设施..."
    
    check_prerequisites
    create_directories
    generate_env_file
    start_infrastructure
    wait_for_services
    show_status
    
    log_success "GoChat 基础设施启动完成！"
}

# 脚本入口
if [[ "${BASH_SOURCE[0]}" == "${0}" ]]; then
    main "$@"
fi