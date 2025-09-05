#!/bin/bash

# GoChat 环境清理脚本
# 用于清理所有容器、网络、数据卷等资源

set -e

# 脚本配置
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
DEPLOYMENT_DIR="$(dirname "$SCRIPT_DIR")"

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

# 确认操作
confirm_cleanup() {
    local cleanup_type="$1"
    
    echo ""
    log_warning "即将执行 $cleanup_type 清理操作"
    log_warning "这将删除所有相关的容器、网络和数据卷"
    echo ""
    
    read -p "确定要继续吗？(y/N): " -n 1 -r
    echo ""
    
    if [[ ! $REPLY =~ ^[Yy]$ ]]; then
        log_info "操作已取消"
        exit 0
    fi
}

# 停止并删除应用服务
cleanup_applications() {
    log_info "清理应用服务..."
    
    local apps_dir="$DEPLOYMENT_DIR/applications"
    
    if [ -d "$apps_dir" ] && [ -f "$apps_dir/docker-compose.yml" ]; then
        cd "$apps_dir"
        
        # 停止并删除容器
        log_info "停止应用服务容器..."
        docker-compose down --remove-orphans || true
        
        # 删除应用镜像（可选）
        if [ "$1" = "--remove-images" ]; then
            log_info "删除应用镜像..."
            docker-compose down --rmi all || true
        fi
        
        log_success "应用服务清理完成"
    else
        log_warning "应用服务配置不存在，跳过清理"
    fi
}

# 停止并删除基础设施服务
cleanup_infrastructure() {
    log_info "清理基础设施服务..."
    
    local infra_dir="$DEPLOYMENT_DIR/infrastructure"
    
    if [ -d "$infra_dir" ] && [ -f "$infra_dir/docker-compose.yml" ]; then
        cd "$infra_dir"
        
        # 停止并删除容器
        log_info "停止基础设施容器..."
        docker-compose down --remove-orphans || true
        
        # 删除数据卷（如果指定）
        if [ "$1" = "--remove-volumes" ]; then
            log_warning "删除数据卷（数据将永久丢失）..."
            docker-compose down --volumes || true
        fi
        
        log_success "基础设施服务清理完成"
    else
        log_warning "基础设施配置不存在，跳过清理"
    fi
}

# 清理 Docker 资源
cleanup_docker_resources() {
    log_info "清理 Docker 资源..."
    
    # 删除所有 gochat 相关容器
    log_info "删除 gochat 相关容器..."
    docker ps -a --format "{{.Names}}" | grep -E "gochat|etcd|kafka|mysql|redis" | xargs -r docker rm -f || true
    
    # 删除悬空镜像
    log_info "删除悬空镜像..."
    docker image prune -f || true
    
    # 删除未使用的网络
    log_info "删除未使用的网络..."
    docker network prune -f || true
    
    # 删除未使用的数据卷（如果指定）
    if [ "$1" = "--remove-volumes" ]; then
        log_warning "删除未使用的数据卷..."
        docker volume prune -f || true
    fi
    
    log_success "Docker 资源清理完成"
}

# 清理本地文件
cleanup_local_files() {
    log_info "清理本地文件..."
    
    # 清理日志文件
    if [ -d "$DEPLOYMENT_DIR/infrastructure/logs" ]; then
        log_info "清理基础设施日志文件..."
        rm -rf "$DEPLOYMENT_DIR/infrastructure/logs"/*
    fi
    
    if [ -d "$DEPLOYMENT_DIR/applications/logs" ]; then
        log_info "清理应用服务日志文件..."
        rm -rf "$DEPLOYMENT_DIR/applications/logs"/*
    fi
    
    # 清理临时文件
    if [ -d "$DEPLOYMENT_DIR/applications/tmp" ]; then
        log_info "清理临时文件..."
        rm -rf "$DEPLOYMENT_DIR/applications/tmp"/*
    fi
    
    # 清理环境变量文件
    rm -f "$DEPLOYMENT_DIR/infrastructure/.env"
    rm -f "$DEPLOYMENT_DIR/applications/.env"
    
    log_success "本地文件清理完成"
}

# 显示清理后状态
show_cleanup_status() {
    log_info "清理后状态："
    
    echo ""
    echo "剩余容器："
    docker ps -a --format "table {{.Names}}\t{{.Status}}" | grep -E "gochat|etcd|kafka|mysql|redis" || echo "无相关容器"
    
    echo ""
    echo "剩余网络："
    docker network ls | grep -E "infrastructure|gochat" || echo "无相关网络"
    
    echo ""
    echo "剩余数据卷："
    docker volume ls | grep -E "infrastructure|gochat" || echo "无相关数据卷"
}

# 显示使用帮助
show_help() {
    echo "GoChat 环境清理脚本"
    echo ""
    echo "用法: $0 [选项]"
    echo ""
    echo "选项:"
    echo "  --apps                只清理应用服务"
    echo "  --infra               只清理基础设施服务"
    echo "  --all                 清理所有服务（默认）"
    echo "  --remove-volumes      同时删除数据卷（数据将永久丢失）"
    echo "  --remove-images       同时删除应用镜像"
    echo "  --force               跳过确认提示"
    echo "  --help                显示此帮助信息"
    echo ""
    echo "示例:"
    echo "  $0                    清理所有服务（保留数据）"
    echo "  $0 --apps             只清理应用服务"
    echo "  $0 --all --remove-volumes  清理所有服务和数据"
    echo "  $0 --force --remove-volumes  强制清理所有数据"
}

# 主函数
main() {
    local cleanup_type="all"
    local remove_volumes=false
    local remove_images=false
    local force=false
    
    # 解析命令行参数
    while [[ $# -gt 0 ]]; do
        case $1 in
            --apps)
                cleanup_type="apps"
                shift
                ;;
            --infra)
                cleanup_type="infra"
                shift
                ;;
            --all)
                cleanup_type="all"
                shift
                ;;
            --remove-volumes)
                remove_volumes=true
                shift
                ;;
            --remove-images)
                remove_images=true
                shift
                ;;
            --force)
                force=true
                shift
                ;;
            --help)
                show_help
                exit 0
                ;;
            *)
                log_error "未知参数: $1"
                show_help
                exit 1
                ;;
        esac
    done
    
    # 确认操作（除非使用 --force）
    if [ "$force" != true ]; then
        confirm_cleanup "$cleanup_type"
    fi
    
    log_info "开始 GoChat 环境清理..."
    
    # 根据参数执行相应的清理操作
    case $cleanup_type in
        "apps")
            cleanup_applications $([ "$remove_images" = true ] && echo "--remove-images")
            ;;
        "infra")
            cleanup_infrastructure $([ "$remove_volumes" = true ] && echo "--remove-volumes")
            ;;
        "all")
            cleanup_applications $([ "$remove_images" = true ] && echo "--remove-images")
            cleanup_infrastructure $([ "$remove_volumes" = true ] && echo "--remove-volumes")
            cleanup_docker_resources $([ "$remove_volumes" = true ] && echo "--remove-volumes")
            cleanup_local_files
            ;;
        *)
            log_error "无效的清理类型: $cleanup_type"
            show_help
            exit 1
            ;;
    esac
    
    show_cleanup_status
    
    log_success "GoChat 环境清理完成！"
}

# 脚本入口
if [[ "${BASH_SOURCE[0]}" == "${0}" ]]; then
    main "$@"
fi