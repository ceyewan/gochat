# GoChat 部署指南

本文档提供了使用提供的 Docker Compose 设置部署 GoChat 系统的说明。

## 1. 架构

部署分为两个主要部分：

-   **基础设施**: 包含应用程序所需的所有后端服务，例如数据库、缓存和消息队列。
-   **应用程序**: 包含 GoChat 微服务本身。

这种分离允许基础设施启动一次并保持运行，而应用程序服务可以在开发期间独立重建和重新启动。

## 2. 先决条件

-   [Docker](https://www.docker.com/get-started)
-   [Docker Compose](https://docs.docker.com/compose/install/)

## 3. 快速开始

### 第 1 步：启动基础设施

导航到 `deployment` 目录并运行 `start-infra.sh` 脚本。这将启动所有必要的后端服务。

```bash
cd deployment
./scripts/start-infra.sh
```

基础设施堆栈包括：
-   etcd（用于配置和服务发现）
-   Kafka（用于消息传递）
-   MySQL（用于数据持久化）
-   Redis（用于缓存）
-   Loki、Prometheus、Grafana、Jaeger（用于监控和日志记录）

### 第 2 步：构建和启动应用程序服务

每个微服务都有自己的 `Dockerfile` 用于构建容器映像。`start-apps.sh` 脚本将构建并启动所有应用程序服务。

```bash
cd deployment
./scripts/start-apps.sh
```

这将启动以下服务：
-   `im-gateway`
-   `im-logic`
-   `im-repo`
-   `im-task`

### 第 3 步：验证部署

使用 `health-check.sh` 脚本验证所有服务是否正确运行。

```bash
cd deployment
./scripts/health-check.sh
```

## 4. 服务端点

所有服务运行后，您可以在以下地址访问它们：

### 管理 UI

| 服务        | 地址                   | 凭据               |
| ------------ | ------------------------- | ------------------------- |
| Grafana        | `http://localhost:3000`   | `admin`/`gochat_grafana_2024` |
| Kafka UI       | `http://localhost:8080`   | -                         |
| etcd Manager   | `http://localhost:8081`   | -                         |
| RedisInsight   | `http://localhost:8001`   | -                         |
| phpMyAdmin     | `http://localhost:8083`   | -                         |
| Jaeger         | `http://localhost:16686`  | -                         |

### 应用程序 API

| 服务      | 地址                 |
| ------------ | ----------------------- |
| **GoChat API** | `http://localhost:8080` |
| **WebSocket**  | `ws://localhost:8080/ws`  |

## 5. 停止环境

要停止服务，请使用 `cleanup.sh` 脚本。

```bash
cd deployment

# 仅停止应用程序服务
./scripts/cleanup.sh --apps

# 停止所有服务（基础设施和应用程序）
./scripts/cleanup.sh --all

# 停止所有服务并删除数据卷
./scripts/cleanup.sh --all --remove-volumes
```

## 6. 单个服务管理

您可以直接使用 `docker-compose` 命令管理单个服务。

```bash
# 查看特定服务的日志
docker-compose -f applications/docker-compose.yml logs -f im-logic

# 重新构建并重新启动单个服务
docker-compose -f applications/docker-compose.yml up -d --build im-gateway
```