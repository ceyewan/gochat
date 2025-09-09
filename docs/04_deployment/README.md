# GoChat 部署与配置指南

本文档旨在为 GoChat 系统的部署、配置和日常管理提供一个清晰、准确且现代化的操作指引。我们遵循“功能完备”与“管理简洁”相结合的原则，在保留多实例集群、监控和调试工具的同时，提供统一的操作入口。

## 部署架构

我们采用两层分离但统一管理的部署架构：

-   **`deployment/infrastructure`**: 包含三个组合式的 docker-compose 文件，按功能模块化管理基础设施服务：
    -   **`docker-compose.yml`**: 核心基础设施 - 3节点的 `etcd` 集群、3节点的 `Kafka` 集群、`MySQL` 和 `Redis`
    -   **`docker-compose.monitoring.yml`**: 监控与日志 - `Prometheus`、`Loki`、`Grafana`、`Promtail`
    -   **`docker-compose.admin.yml`**: 管理调试工具 - `Kafka-UI`、`RedisInsight`、`phpMyAdmin`、`etcd-workbench`

-   **`deployment/applications`**: 包含一个独立的 `docker-compose.yml`，用于管理所有 GoChat 应用微服务 (`im-repo`, `im-logic`, `im-gateway`, `im-task`)。

**核心理念**:
1.  **模块化部署**: 基础设施按功能模块分为核心服务、监控服务和管理工具三个组合式 compose 文件，支持按需启动
2.  **脚本化管理**: 通过 `deployment/scripts/` 下的脚本封装部署逻辑（如生成Kafka集群ID），提供可靠的、可重复的部署操作
3.  **统一入口**: 所有部署操作都通过项目根目录的 `Makefile` 提供，例如 `make infra-up`、`make monitoring-up`、`make admin-up`，支持灵活的组合启动
4.  **配置中心驱动**: 所有应用服务仅通过一个环境变量 (`COORD_ETCD_ENDPOINTS`) 连接到 `etcd`，并从中获取所有其他配置。这杜绝了在 `docker-compose.yml` 中硬编码业务配置的弊端
5.  **日志标准输出**: 所有应用服务将结构化日志输出到 `stdout`，由 `Promtail` 自动发现并收集，实现了日志的集中管理和高效查询

## 部署流程

**先决条件**:
-   [Docker](https://www.docker.com/get-started)
-   [Docker Compose V2](https://docs.docker.com/compose/install/) (确保您使用的是 `docker compose` 而非 `docker-compose`)

**重要提醒**: 
-   必须先启动基础设施服务，再启动应用服务
-   基础设施服务启动后需要等待 30-60 秒确保所有服务完全就绪
-   所有部署命令已集成到项目根目录的 `Makefile` 中，请使用 `make` 命令进行操作

**快速启动指南**:
```bash
# 1. 启动完整基础设施 (推荐开发环境)
make admin-up

# 2. 等待服务启动完成 (查看状态)
docker compose -f deployment/infrastructure/docker-compose.yml -f deployment/infrastructure/docker-compose.monitoring.yml -f deployment/infrastructure/docker-compose.admin.yml ps

# 3. 启动应用服务
make app-up

# 4. 检查应用状态
docker compose -f deployment/applications/docker-compose.yml ps
```

### 步骤 1: 启动基础设施

根据需要选择不同的启动方式：

```bash
# 选项 1: 仅启动核心基础设施 (etcd, kafka, mysql, redis)
make infra-up

# 选项 2: 启动核心 + 监控服务 (上述 + prometheus, loki, grafana, promtail)
make monitoring-up

# 选项 3: 启动核心 + 监控 + 管理工具 (上述 + kafka-ui, etcd-workbench, redis-insight, phpmyadmin)
make admin-up

# 选项 4: 启动所有基础设施服务 (等同于选项 3)
make infra-up-all
```

**推荐**: 开发环境使用 `make admin-up`，生产环境使用 `make monitoring-up`

### 步骤 2: 启动应用服务

在基础设施完全启动后，调用 `deployment/scripts/start-apps.sh` 脚本启动所有 GoChat 微服务。

```bash
# 在项目根目录运行
make app-up
```

### 步骤 3: 停止环境

我们提供了对应的停止命令，按正确的顺序停止服务：

```bash
# 停止核心基础设施
make infra-down

# 停止核心 + 监控服务
make monitoring-down

# 停止核心 + 监控 + 管理工具
make admin-down

# 停止所有基础设施服务 (等同于 admin-down)
make infra-down-all

# 仅停止应用服务
make app-down
```

*注意: down 命令会按正确的顺序停止对应的服务组合，确保依赖关系的正确处理*

## 配置管理

### 核心原则
- **KISS**: 配置文件 (`config/dev/`) 中只保留最核心、必须由环境决定的配置（如数据库DSN，服务地址等）。
- **默认值优先**: 所有非核心配置项（如连接池大小、超时时间等）均在 `im-infra` 组件代码中提供合理的默认值，无需在JSON文件中定义。
- **配置中心**: `etcd` 是所有应用服务的唯一可信配置来源。

### 操作流程

我们推荐通过项目根目录的 `Makefile` 来管理配置的同步，这提供了统一且简化的操作入口。

1.  **修改本地配置**: 在 `config/dev/` 目录下找到并修改你需要的 JSON 配置文件。

2.  **同步到 etcd**: 打开终端，在 **项目根目录** 运行以下命令，即可将 `dev` 环境的所有配置同步到 etcd。

    ```bash
    # 将 config/dev/ 目录下的所有配置同步到 etcd
    make config-sync-dev
    ```
    *这个命令会自动编译并运行 `config-cli` 工具，并强制执行同步，无需手动确认。*

3.  **预览同步 (可选)**: 如果你想在实际写入前查看哪些配置会被同步，可以进入 `config-cli` 目录手动执行命令并使用 `--dry-run` 标志。
    ```bash
    cd config/config-cli
    go run . sync dev --dry-run
    ```

4.  **自动加载**: 正在运行的服务会自动从 `etcd` 拉取最新配置并进行热加载，无需重启服务。

## 监控与日志

- **统一入口**: 所有监控、日志查询和告警都通过 **Grafana** 进行。
- **地址**: `http://localhost:3000`
- **凭据**: `admin` / `gochat_grafana_2024`

### 查看日志
1.  登录 Grafana。
2.  进入 "Explore" 页面，选择 "Loki" 数据源。
3.  使用 LogQL 查询，例如: `{container_name="gochat-im-logic"}`。

### 查看指标
1.  登录 Grafana。
2.  进入 "Explore" 页面，选择 "Prometheus" 数据源。
3.  使用 PromQL 查询，例如: `rate(grpc_server_handled_total[5m])`。

## 服务访问端点

### 核心基础设施端口

| 服务 | 节点 | 外部端口 | 内部端口 | 用途 |
|---|---|---|---|---|
| **etcd** | etcd1 | 2379, 2380 | 2379, 2380 | etcd 集群节点 1 |
| **etcd** | etcd2 | 12379, 12380 | 2379, 2380 | etcd 集群节点 2 |
| **etcd** | etcd3 | 22379, 22380 | 2379, 2380 | etcd 集群节点 3 |
| **Kafka** | kafka1 | 9092, 9093 | 9092, 9093 | Kafka 集群节点 1 |
| **Kafka** | kafka2 | 19092, 19093 | 9092, 9093 | Kafka 集群节点 2 |
| **Kafka** | kafka3 | 29092, 29093 | 9092, 9093 | Kafka 集群节点 3 |
| **MySQL** | mysql | 3306 | 3306 | 数据库服务 |
| **Redis** | redis | 6379 | 6379 | 缓存服务 |

### 监控与管理界面

| 服务 | 地址 | 凭据 | 用途 |
|---|---|---|---|
| **Grafana** | `http://localhost:3000` | `admin` / `gochat_grafana_2024` | **统一监控日志平台** |
| Prometheus | `http://localhost:9090` | - | 指标系统 |
| Loki | `http://localhost:3100` | - | 日志系统 |
| Kafka UI | `http://localhost:8088` | - | Kafka 管理界面 |
| etcd-workbench | `http://localhost:8002` | - | etcd 管理界面 |
| RedisInsight | `http://localhost:5540` | - | Redis 管理界面 |
| phpMyAdmin | `http://localhost:8083` | `root` / `gochat_root_2024` | MySQL 管理界面 |

### 应用服务端点

| 服务 | 地址 | 用途 |
|---|---|---|
| **GoChat Gateway** | `http://localhost:8081` | **应用主入口 (HTTP API)** |
| **GoChat Gateway** | `ws://localhost:8082`   | **应用主入口 (WebSocket)** |
| GoChat Repo | `http://localhost:8090` | 数据仓库服务 (HTTP API + 健康检查) |
| GoChat Logic | `grpc://localhost:9000` | 业务逻辑服务 (gRPC) |
| GoChat Task | `http://localhost:9094` | 任务处理服务 (指标 + 健康检查) |

### 监控指标端点

| 服务 | 指标端点 | 用途 |
|---|---|---|
| GoChat Repo | `http://localhost:9091/metrics` | Prometheus 指标 |
| GoChat Logic | `http://localhost:9092/metrics` | Prometheus 指标 |
| GoChat Gateway | `http://localhost:9093/metrics` | Prometheus 指标 |
| GoChat Task | `http://localhost:9094/metrics` | Prometheus 指标 |

## 故障排除

### 常见问题

1. **端口冲突**
   ```bash
   # 检查端口占用
   lsof -i :3306  # MySQL
   lsof -i :6379  # Redis
   lsof -i :9092  # Kafka
   ```

2. **服务无法启动**
   ```bash
   # 查看服务日志
   docker compose -f deployment/infrastructure/docker-compose.yml logs [service_name]
   
   # 查看所有服务状态
   make help  # 查看可用命令
   ```

3. **网络连接问题**
   ```bash
   # 检查网络
   docker network ls
   docker network inspect deployment_infrastructure_infra-net
   ```

4. **配置同步失败**
   ```bash
   # 手动检查 etcd 连接
   docker exec gochat-etcd1 etcdctl endpoint health
   
   # 重新同步配置
   make config-sync-dev
   ```

### 清理和重置

```bash
# 完全清理并重新开始
make infra-down-all
make app-down
docker system prune -f
docker volume prune -f

# 重新启动
make admin-up
# 等待服务就绪
make app-up
```