# GoChat 部署指南

本目录包含 GoChat 项目的完整部署配置，采用基础设施与应用服务分离的架构设计。

## 目录结构

```
deployment/
├── infrastructure/           # 基础设施部署配置
│   ├── docker-compose.yml   # 主配置文件
│   ├── etcd.yml             # etcd 集群配置
│   ├── kafka.yml            # Kafka 集群配置
│   ├── storage.yml          # MySQL 和 Redis 配置
│   ├── monitoring.yml       # 监控服务配置
│   ├── config/              # 服务配置文件
│   └── init-scripts/        # 初始化脚本
├── applications/            # 应用服务部署配置
│   ├── docker-compose.yml   # 主配置文件
│   └── services/            # 各服务独立配置
├── scripts/                 # 部署和管理脚本
│   ├── start-infra.sh       # 启动基础设施
│   ├── start-apps.sh        # 启动应用服务
│   ├── health-check.sh      # 健康检查
│   └── cleanup.sh           # 环境清理
└── README.md               # 本文件
```

## 快速开始

### 1. 启动基础设施

```bash
# 启动所有基础设施组件
./scripts/start-infra.sh
```

基础设施包括：
- **etcd 集群**：3节点，用于配置中心和服务发现
- **Kafka 集群**：3节点，用于消息队列
- **MySQL**：数据库服务
- **Redis**：缓存服务
- **管理界面**：各服务的 Web 管理界面

### 2. 检查基础设施状态

```bash
# 检查所有基础设施服务
./scripts/health-check.sh --component infra
```

### 3. 启动应用服务

```bash
# 启动所有应用服务
./scripts/start-apps.sh
```

应用服务包括：
- **im-repo**：数据仓库服务
- **im-logic**：业务逻辑服务
- **im-gateway**：网关服务
- **im-task**：任务处理服务

### 4. 检查整体状态

```bash
# 检查所有服务
./scripts/health-check.sh

# 生成健康检查报告
./scripts/health-check.sh --report
```

## 服务访问地址

### 基础设施管理界面

| 服务 | 地址 | 用途 |
|------|------|------|
| Kafka UI | http://localhost:8080 | Kafka 集群管理 |
| etcd Manager | http://localhost:8081 | etcd 集群管理 |
| RedisInsight | http://localhost:8001 | Redis 可视化管理 |
| phpMyAdmin | http://localhost:8083 | MySQL 管理 |

### 监控和日志界面

| 服务 | 地址 | 用途 |
|------|------|------|
| Grafana | http://localhost:3000 | 统一可视化平台 (admin/gochat_grafana_2024) |
| Prometheus | http://localhost:9090 | 指标数据查询 |
| Loki | http://localhost:3100 | 日志数据 API |
| Vector API | http://localhost:8686 | 日志收集器状态 |
| Jaeger | http://localhost:16686 | 分布式链路追踪 |

### 应用服务接口

| 服务 | HTTP API | gRPC | Metrics | 用途 |
|------|----------|------|---------|------|
| im-repo | :8090 | - | :9091 | 数据仓库 |
| im-logic | - | :9000 | :9092 | 业务逻辑 |
| im-gateway | :8080 | - | :9093 | API 网关 |
| im-task | - | - | :9094 | 任务处理 |

### 基础设施端点

| 服务 | 端点 | 用途 |
|------|------|------|
| etcd | localhost:2379,2389,2399 | 配置中心 |
| Kafka | localhost:19092,29092,39092 | 消息队列 |
| MySQL | localhost:3306 | 数据库 |
| Redis | localhost:6379 | 缓存 |

## 高级操作

### 独立服务管理

可以独立启动和管理特定的服务组件：

```bash
# 只启动 etcd 集群
cd infrastructure && docker-compose -f etcd.yml up -d

# 只启动 Kafka 集群
cd infrastructure && docker-compose -f kafka.yml up -d

# 只启动存储服务
cd infrastructure && docker-compose -f storage.yml up -d

# 只启动监控服务
cd infrastructure && docker-compose -f monitoring.yml up -d
```

### 服务扩缩容

```bash
# 扩展 Kafka 消费者
cd applications && docker-compose up -d --scale im-task=3

# 扩展 API 网关
cd applications && docker-compose up -d --scale im-gateway=2
```

### 日志查看

```bash
# 查看基础设施日志
cd infrastructure && docker-compose logs -f [service_name]

# 查看应用服务日志
cd applications && docker-compose logs -f [service_name]

# 查看特定服务日志
docker logs -f gochat-mysql
docker logs -f gochat-kafka1
```

### 数据备份

```bash
# 备份 MySQL 数据
docker exec gochat-mysql mysqldump -u root -pgochat_root_2024 --all-databases > backup.sql

# 备份 etcd 数据
docker exec gochat-etcd1 etcdctl snapshot save /tmp/etcd-backup.db
docker cp gochat-etcd1:/tmp/etcd-backup.db ./etcd-backup.db
```

## 环境清理

### 清理应用服务

```bash
# 只清理应用服务
./scripts/cleanup.sh --apps
```

### 清理基础设施

```bash
# 只清理基础设施（保留数据）
./scripts/cleanup.sh --infra

# 清理基础设施和数据
./scripts/cleanup.sh --infra --remove-volumes
```

### 完全清理

```bash
# 清理所有服务（保留数据）
./scripts/cleanup.sh --all

# 清理所有服务和数据
./scripts/cleanup.sh --all --remove-volumes

# 强制清理（跳过确认）
./scripts/cleanup.sh --all --remove-volumes --force
```

## 故障排除

### 常见问题

1. **端口冲突**
   - 检查端口是否被占用：`lsof -i :端口号`
   - 修改配置文件中的端口映射

2. **容器启动失败**
   - 查看容器日志：`docker logs 容器名`
   - 检查资源使用情况：`docker stats`

3. **网络连接问题**
   - 检查网络配置：`docker network ls`
   - 验证容器间连通性：`docker exec 容器名 ping 目标容器`

4. **数据持久化问题**
   - 检查数据卷：`docker volume ls`
   - 验证挂载点：`docker inspect 容器名`

### 健康检查

使用健康检查脚本诊断问题：

```bash
# 检查特定组件
./scripts/health-check.sh --component infra
./scripts/health-check.sh --component apps

# 生成详细报告
./scripts/health-check.sh --report health-report.md
```

### 性能监控

如果启用了监控服务：

- **Prometheus**: http://localhost:9090
- **Grafana**: http://localhost:3000 (admin/gochat_grafana_2024)
- **Jaeger**: http://localhost:16686

## 配置说明

### 环境变量

主要环境变量在各服务的 docker-compose 文件中定义：

- `APP_ENV`: 应用环境 (dev/test/prod)
- `DB_*`: 数据库连接配置
- `REDIS_*`: Redis 连接配置
- `ETCD_*`: etcd 连接配置
- `KAFKA_*`: Kafka 连接配置

### 配置文件

- 基础设施配置文件位于 `infrastructure/config/`
- 应用服务配置文件位于 `applications/config/`
- 初始化脚本位于 `infrastructure/init-scripts/`

### 数据持久化

所有重要数据都通过 Docker 数据卷持久化：

- etcd 数据：`etcd-data-{1,2,3}`
- Kafka 数据：`kafka-data-{1,2,3}`
- MySQL 数据：`mysql-data`
- Redis 数据：`redis-data`

## 安全注意事项

1. **默认密码**：生产环境请修改所有默认密码
2. **网络隔离**：生产环境建议使用更严格的网络隔离
3. **访问控制**：启用服务的认证和授权机制
4. **数据加密**：生产环境启用传输和存储加密

## 支持

如有问题，请查看：
1. 服务日志文件
2. 健康检查报告
3. Docker 容器状态
4. 网络和存储配置

更多详细信息请参考各服务的官方文档。