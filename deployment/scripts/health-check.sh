#!/bin/bash

# GoChat 健康检查脚本
# 用于检查所有服务的健康状态

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

# 检查单个服务健康状态
check_service_health() {
    local service_name=$1
    local health_check_cmd=$2
    local description=$3
    
    if eval "$health_check_cmd" &> /dev/null; then
        log_success "$service_name: $description - 健康"
        return 0
    else
        log_error "$service_name: $description - 不健康"
        return 1
    fi
}

# 检查基础设施服务
check_infrastructure() {
    log_info "检查基础设施服务健康状态..."
    
    local failed_count=0
    
    # 检查 etcd 集群
    check_service_health "etcd1" "docker exec gochat-etcd1 etcdctl endpoint health --endpoints=http://localhost:2379" "etcd 节点 1" || ((failed_count++))
    check_service_health "etcd2" "docker exec gochat-etcd2 etcdctl endpoint health --endpoints=http://localhost:2379" "etcd 节点 2" || ((failed_count++))
    check_service_health "etcd3" "docker exec gochat-etcd3 etcdctl endpoint health --endpoints=http://localhost:2379" "etcd 节点 3" || ((failed_count++))
    
    # 检查 MySQL
    check_service_health "mysql" "docker exec gochat-mysql mysqladmin ping -h localhost -u root -pgochat_root_2024" "MySQL 数据库" || ((failed_count++))
    
    # 检查 Redis
    check_service_health "redis" "docker exec gochat-redis redis-cli ping | grep -q PONG" "Redis 缓存" || ((failed_count++))
    
    # 检查 Kafka 集群
    check_service_health "kafka1" "docker exec gochat-kafka1 kafka-broker-api-versions.sh --bootstrap-server localhost:9092" "Kafka 节点 1" || ((failed_count++))
    check_service_health "kafka2" "docker exec gochat-kafka2 kafka-broker-api-versions.sh --bootstrap-server localhost:9092" "Kafka 节点 2" || ((failed_count++))
    check_service_health "kafka3" "docker exec gochat-kafka3 kafka-broker-api-versions.sh --bootstrap-server localhost:9092" "Kafka 节点 3" || ((failed_count++))
    
    # 检查管理界面
    check_service_health "kafka-ui" "curl -f http://localhost:8080/actuator/health" "Kafka UI" || ((failed_count++))
    check_service_health "etcd-manager" "curl -f http://localhost:8081" "etcd 管理界面" || ((failed_count++))
    check_service_health "redis-insight" "curl -f http://localhost:8001" "RedisInsight" || ((failed_count++))
    check_service_health "phpmyadmin" "curl -f http://localhost:8083" "phpMyAdmin" || ((failed_count++))
    
    if [ $failed_count -eq 0 ]; then
        log_success "所有基础设施服务健康"
    else
        log_error "$failed_count 个基础设施服务不健康"
    fi
    
    return $failed_count
}

# 检查应用服务
check_applications() {
    log_info "检查应用服务健康状态..."
    
    local failed_count=0
    
    # 检查应用服务
    check_service_health "im-repo" "curl -f http://localhost:8090/health" "im-repo 服务" || ((failed_count++))
    check_service_health "im-logic" "docker exec gochat-im-logic grpc_health_probe -addr=localhost:9000" "im-logic 服务" || ((failed_count++))
    check_service_health "im-gateway" "curl -f http://localhost:8080/health" "im-gateway 服务" || ((failed_count++))
    check_service_health "im-task" "curl -f http://localhost:9094/health" "im-task 服务" || ((failed_count++))
    
    if [ $failed_count -eq 0 ]; then
        log_success "所有应用服务健康"
    else
        log_error "$failed_count 个应用服务不健康"
    fi
    
    return $failed_count
}

# 检查监控服务
check_monitoring() {
    log_info "检查监控服务健康状态..."
    
    local failed_count=0
    
    # 检查监控服务（如果启动了的话）
    if docker ps --format "table {{.Names}}" | grep -q "gochat-prometheus"; then
        check_service_health "prometheus" "curl -f http://localhost:9090/-/healthy" "Prometheus" || ((failed_count++))
    else
        log_warning "Prometheus 未启动"
    fi
    
    if docker ps --format "table {{.Names}}" | grep -q "gochat-loki"; then
        check_service_health "loki" "curl -f http://localhost:3100/ready" "Loki" || ((failed_count++))
    else
        log_warning "Loki 未启动"
    fi
    
    if docker ps --format "table {{.Names}}" | grep -q "gochat-vector"; then
        check_service_health "vector" "curl -f http://localhost:8686/health" "Vector" || ((failed_count++))
    else
        log_warning "Vector 未启动"
    fi
    
    if docker ps --format "table {{.Names}}" | grep -q "gochat-grafana"; then
        check_service_health "grafana" "curl -f http://localhost:3000/api/health" "Grafana" || ((failed_count++))
    else
        log_warning "Grafana 未启动"
    fi
    
    if docker ps --format "table {{.Names}}" | grep -q "gochat-jaeger"; then
        check_service_health "jaeger" "curl -f http://localhost:14269/" "Jaeger" || ((failed_count++))
    else
        log_warning "Jaeger 未启动"
    fi
    
    if [ $failed_count -eq 0 ]; then
        log_success "所有监控服务健康"
    else
        log_error "$failed_count 个监控服务不健康"
    fi
    
    return $failed_count
}

# 生成健康检查报告
generate_report() {
    local output_file="$1"
    
    log_info "生成健康检查报告..."
    
    {
        echo "# GoChat 系统健康检查报告"
        echo "生成时间: $(date)"
        echo ""
        
        echo "## 容器状态"
        docker ps --format "table {{.Names}}\t{{.Status}}\t{{.Ports}}" | grep gochat
        echo ""
        
        echo "## 网络状态"
        docker network ls | grep -E "(infrastructure|gochat)"
        echo ""
        
        echo "## 数据卷状态"
        docker volume ls | grep -E "(infrastructure|gochat)"
        echo ""
        
        echo "## 资源使用情况"
        docker stats --no-stream --format "table {{.Container}}\t{{.CPUPerc}}\t{{.MemUsage}}\t{{.NetIO}}\t{{.BlockIO}}" | grep gochat
        echo ""
        
    } > "$output_file"
    
    log_success "健康检查报告已生成: $output_file"
}

# 显示使用帮助
show_help() {
    echo "GoChat 健康检查脚本"
    echo ""
    echo "用法: $0 [选项]"
    echo ""
    echo "选项:"
    echo "  --component infra     只检查基础设施服务"
    echo "  --component apps      只检查应用服务"
    echo "  --component monitoring 只检查监控服务"
    echo "  --report [文件]       生成健康检查报告"
    echo "  --help               显示此帮助信息"
    echo ""
    echo "示例:"
    echo "  $0                    检查所有服务"
    echo "  $0 --component infra  只检查基础设施"
    echo "  $0 --report report.md 生成报告到文件"
}

# 主函数
main() {
    local component="all"
    local generate_report_file=""
    
    # 解析命令行参数
    while [[ $# -gt 0 ]]; do
        case $1 in
            --component)
                component="$2"
                shift 2
                ;;
            --report)
                generate_report_file="${2:-health-report-$(date +%Y%m%d-%H%M%S).md}"
                shift 2
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
    
    log_info "开始 GoChat 系统健康检查..."
    
    local total_failed=0
    
    # 根据参数检查相应组件
    case $component in
        "infra")
            check_infrastructure || total_failed=$?
            ;;
        "apps")
            check_applications || total_failed=$?
            ;;
        "monitoring")
            check_monitoring || total_failed=$?
            ;;
        "all")
            check_infrastructure || ((total_failed+=$?))
            check_applications || ((total_failed+=$?))
            check_monitoring || ((total_failed+=$?))
            ;;
        *)
            log_error "无效的组件类型: $component"
            show_help
            exit 1
            ;;
    esac
    
    # 生成报告（如果需要）
    if [ -n "$generate_report_file" ]; then
        generate_report "$generate_report_file"
    fi
    
    # 输出总结
    echo ""
    if [ $total_failed -eq 0 ]; then
        log_success "所有检查的服务都健康！"
        exit 0
    else
        log_error "发现 $total_failed 个服务不健康"
        exit 1
    fi
}

# 脚本入口
if [[ "${BASH_SOURCE[0]}" == "${0}" ]]; then
    main "$@"
fi