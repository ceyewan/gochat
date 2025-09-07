# GoChat 部署与配置指南

本文档旨在为 GoChat 系统的部署、配置和日常管理提供一个清晰、准确且现代化的操作指引。我们遵循“功能完备”与“管理简洁”相结合的原则，在保留多实例集群、监控和调试工具的同时，提供统一的操作入口。

## 部署架构

我们采用两层分离但统一管理的部署架构：

-   **`deployment/infrastructure`**: 包含一个统一的 `docker-compose.yml`，负责管理所有核心基础设施服务。这包括：
    -   **高可用核心服务**: 3节点的 `etcd` 集群和3节点的 `Kafka` 集群。
    -   **数据存储**: `MySQL` 和 `Redis`。
    -   **监控与日志**: `Prometheus`, `Loki`, `Grafana`, `Promtail`。
    -   **调试工具**: `Kafka-UI`, `RedisInsight`, `phpMyAdmin`, `etcd-workbench`。

-   **`deployment/applications`**: 包含一个独立的 `docker-compose.yml`，用于管理所有 GoChat 应用微服务 (`im-repo`, `im-logic`, `im-gateway`, `im-task`)。

**核心理念**:
1.  **脚本化管理**: 通过 `deployment/scripts/` 下的脚本封装部署逻辑（如生成Kafka集群ID），提供可靠的、可重复的部署操作。
2.  **统一入口**: 所有部署操作都通过项目根目录的 `Makefile` 提供，例如 `make infra-up`，简化了开发者的日常使用。
3.  **配置中心驱动**: 所有应用服务仅通过一个环境变量 (`COORD_ETCD_ENDPOINTS`) 连接到 `etcd`，并从中获取所有其他配置。这杜绝了在 `docker-compose.yml` 中硬编码业务配置的弊端。
4.  **日志标准输出**: 所有应用服务将结构化日志输出到 `stdout`，由 `Promtail` 自动发现并收集，实现了日志的集中管理和高效查询。

## 部署流程

**先决条件**:
-   [Docker](https://www.docker.com/get-started)
-   [Docker Compose V2](https://docs.docker.com/compose/install/) (确保您使用的是 `docker compose` 而非 `docker-compose`)

所有部署命令已集成到项目根目录的 `Makefile` 中，请使用 `make` 命令进行操作。

### 步骤 1: 启动基础设施

此命令将调用 `deployment/scripts/start-infra.sh` 脚本，启动一个功能完备的基础设施集群。

```bash
# 在项目根目录运行
make infra-up
```

### 步骤 2: 启动应用服务

在基础设施完全启动后，调用 `deployment/scripts/start-apps.sh` 脚本启动所有 GoChat 微服务。

```bash
# 在项目根目录运行
make app-up
```

### 步骤 3: 停止环境

我们提供了统一的清理脚本，它会按正确的顺序（先应用，后基础）停止所有服务。

```bash
# 在项目根目录运行
make infra-down
```
*注意: `make infra-down` 会调用 `cleanup.sh`，该脚本会同时停止应用和基础设施。如果只想单独停止应用，请运行 `make app-down`。*

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

| 服务 | 地址 | 用途 |
|---|---|---|
| Grafana | `http://localhost:3000` | 统一监控日志平台 |
| Prometheus | `http://localhost:9090` | 指标系统 |
| Kafka UI | `http://localhost:8088` | Kafka 管理界面 |
| etcd-workbench | `http://localhost:8002` | etcd 管理界面 |
| RedisInsight | `http://localhost:8001` | Redis 管理界面 |
| phpMyAdmin | `http://localhost:8083` | MySQL 管理界面 |
| **GoChat Gateway** | `http://localhost:8080` | **应用主入口 (HTTP)** |
| **GoChat Gateway** | `ws://localhost:8081`   | **应用主入口 (WebSocket)** |