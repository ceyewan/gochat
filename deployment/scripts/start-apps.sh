#!/bin/bash

# GoChat 应用服务启动脚本
# 用于启动所有应用服务：im-repo、im-logic、im-gateway、im-task

set -e  # 遇到错误立即退出

# 脚本配置
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
DEPLOYMENT_DIR="$(dirname "$SCRIPT_DIR")"
APPS_DIR="$DEPLOYMENT_DIR/applications"

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

# 检查基础设施是否就绪
check_infrastructure() {
    log_info "检查基础设施状态..."
    
    # 检查 etcd
    if ! docker exec gochat-etcd1 etcdctl endpoint health --endpoints=http://localhost:2379 &> /dev/null; then
        log_error "etcd 集群未就绪，请先启动基础设施"
        exit 1
    fi
    
    # 检查 MySQL
    if ! docker exec gochat-mysql mysqladmin ping -h localhost -u root -pgochat_root_2024 &> /dev/null; then
        log_error "MySQL 未就绪，请先启动基础设施"
        exit 1
    fi
    
    # 检查 Redis
    if ! docker exec gochat-redis redis-cli ping | grep -q PONG; then
        log_error "Redis 未就绪，请先启动基础设施"
        exit 1
    fi
    
    # 检查 Kafka
    if ! docker exec gochat-kafka1 kafka-broker-api-versions.sh --bootstrap-server localhost:9092 &> /dev/null; then
        log_error "Kafka 集群未就绪，请先启动基础设施"
        exit 1
    fi
    
    log_success "基础设施状态检查通过"
}

# 创建必要的目录
create_directories() {
    log_info "创建应用服务目录..."
    
    # 创建配置目录
    mkdir -p "$APPS_DIR/config/im-repo"
    mkdir -p "$APPS_DIR/config/im-logic"
    mkdir -p "$APPS_DIR/config/im-gateway"
    mkdir -p "$APPS_DIR/config/im-task"
    
    # 创建日志目录
    mkdir -p "$APPS_DIR/logs/im-repo"
    mkdir -p "$APPS_DIR/logs/im-logic"
    mkdir -p "$APPS_DIR/logs/im-gateway"
    mkdir -p "$APPS_DIR/logs/im-task"
    
    # 创建临时目录
    mkdir -p "$APPS_DIR/tmp/im-repo"
    mkdir -p "$APPS_DIR/tmp/im-logic"
    mkdir -p "$APPS_DIR/tmp/im-gateway"
    mkdir -p "$APPS_DIR/tmp/im-task"
    
    # 设置权限
    chmod -R 755 "$APPS_DIR/config"
    chmod -R 755 "$APPS_DIR/logs"
    chmod -R 755 "$APPS_DIR/tmp"
    
    log_success "应用服务目录创建完成"
}

# 生成环境变量文件
generate_env_file() {
    log_info "生成应用服务环境变量文件..."
    
    cat > "$APPS_DIR/.env" << EOF
# GoChat 应用服务环境变量
COMPOSE_PROJECT_NAME=gochat-apps
COMPOSE_FILE=docker-compose.yml

# 应用版本
APP_VERSION=latest

# 环境配置
APP_ENV=dev
TZ=Asia/Shanghai

# 网络配置
INFRA_NETWORK=infrastructure_infra-net
MONITORING_NETWORK=infrastructure_monitoring-net

# 目录配置
CONFIG_DIR=./config
LOGS_DIR=./logs
TMP_DIR=./tmp
EOF
    
    log_success "应用服务环境变量文件生成完成"
}

# 构建应用镜像（如果需要）
build_images() {
    log_info "检查应用镜像..."
    
    # 检查镜像是否存在，如果不存在则提示用户构建
    local images=("gochat/im-repo:latest" "gochat/im-logic:latest" "gochat/im-gateway:latest" "gochat/im-task:latest")
    local missing_images=()
    
    for image in "${images[@]}"; do
        if ! docker image inspect "$image" &> /dev/null; then
            missing_images+=("$image")
        fi
    done
    
    if [ ${#missing_images[@]} -gt 0 ]; then
        log_warning "以下镜像不存在："
        for image in "${missing_images[@]}"; do
            echo "  - $image"
        done
        log_warning "请先构建应用镜像或使用现有镜像"
        log_info "提示：可以使用 'docker build' 命令构建镜像"
        return 1
    fi
    
    log_success "应用镜像检查通过"
}

# 启动应用服务
start_applications() {
    log_info "启动应用服务..."
    
    cd "$APPS_DIR"
    
    # 拉取镜像（如果是远程镜像）
    log_info "拉取应用镜像..."
    docker-compose pull --ignore-pull-failures
    
    # 启动服务
    log_info "启动应用容器..."
    docker-compose up -d
    
    log_success "应用服务启动完成"
}

# 等待应用服务就绪
wait_for_applications() {
    log_info "等待应用服务就绪..."
    
    # 等待 im-repo 就绪
    log_info "等待 im-repo 服务就绪..."
    for i in {1..30}; do
        if curl -f http://localhost:8090/health &> /dev/null; then
            log_success "im-repo 服务就绪"
            break
        fi
        if [ $i -eq 30 ]; then
            log_warning "im-repo 服务健康检查超时，请检查服务状态"
        fi
        sleep 5
    done
    
    # 等待 im-logic 就绪
    log_info "等待 im-logic 服务就绪..."
    for i in {1..30}; do
        if docker exec gochat-im-logic grpc_health_probe -addr=localhost:9000 &> /dev/null; then
            log_success "im-logic 服务就绪"
            break
        fi
        if [ $i -eq 30 ]; then
            log_warning "im-logic 服务健康检查超时，请检查服务状态"
        fi
        sleep 5
    done
    
    # 等待 im-gateway 就绪
    log_info "等待 im-gateway 服务就绪..."
    for i in {1..30}; do
        if curl -f http://localhost:8080/health &> /dev/null; then
            log_success "im-gateway 服务就绪"
            break
        fi
        if [ $i -eq 30 ]; then
            log_warning "im-gateway 服务健康检查超时，请检查服务状态"
        fi
        sleep 5
    done
    
    # 等待 im-task 就绪
    log_info "等待 im-task 服务就绪..."
    for i in {1..30}; do
        if curl -f http://localhost:9094/health &> /dev/null; then
            log_success "im-task 服务就绪"
            break
        fi
        if [ $i -eq 30 ]; then
            log_warning "im-task 服务健康检查超时，请检查服务状态"
        fi
        sleep 5
    done
}

# 显示服务状态
show_status() {
    log_info "应用服务状态："
    
    cd "$APPS_DIR"
    docker-compose ps
    
    echo ""
    log_info "服务访问地址："
    echo "  im-repo API:       http://localhost:8090"
    echo "  im-gateway API:    http://localhost:8080"
    echo "  im-gateway WS:     ws://localhost:8081"
    echo ""
    echo "  im-repo Metrics:   http://localhost:9091/metrics"
    echo "  im-logic Metrics:  http://localhost:9092/metrics"
    echo "  im-gateway Metrics: http://localhost:9093/metrics"
    echo "  im-task Metrics:   http://localhost:9094/metrics"
}

# 主函数
main() {
    log_info "开始启动 GoChat 应用服务..."
    
    check_infrastructure
    create_directories
    generate_env_file
    
    # 检查镜像，如果不存在则跳过启动
    if ! build_images; then
        log_error "应用镜像不存在，请先构建镜像"
        exit 1
    fi
    
    start_applications
    wait_for_applications
    show_status
    
    log_success "GoChat 应用服务启动完成！"
}

# 脚本入口
if [[ "${BASH_SOURCE[0]}" == "${0}" ]]; then
    main "$@"
fi