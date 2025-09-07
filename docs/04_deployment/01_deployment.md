# GoChat 部署指南

本文档提供了使用 Docker Compose 部署 GoChat 系统的说明，该部署分为**基础设施**和**应用服务**两个层次。

## 1. 部署架构

-   **`deployment/infrastructure`**: 此目录包含一个统一的 `docker-compose.yml`，用于管理所有核心基础设施服务，包括：
    -   **数据存储**: `MySQL`, `Redis`
    -   **消息与协调**: `Kafka`, `etcd`
    -   **监控与日志**: `Prometheus`, `Loki`, `Grafana`, `Promtail`
    
-   **`deployment/applications`**: 此目录包含一个独立的 `docker-compose.yml`，用于管理所有 GoChat 应用微服务：`im-gateway`, `im-logic`, `im-repo`, `im-task`。

这种分层结构确保了基础服务和应用服务的解耦，便于独立管理和维护。

## 2. 先决条件

-   [Docker](https://www.docker.com/get-started)
-   [Docker Compose](https://docs.docker.com/compose/install/)

## 3. 部署流程

**核心原则**: 必须先成功启动 `infrastructure` 层，然后再启动 `applications` 层。

### 步骤 1: 启动基础设施

```bash
# 切换到 infrastructure 目录
cd deployment/infrastructure

# 以后台模式启动所有基础设施服务
docker-compose up -d

# 检查所有容器是否正常运行
docker-compose ps
```
*请等待片刻，确保数据库等服务完成初始化。*

### 步骤 2: 启动应用服务

```bash
# 切换到 applications 目录
cd deployment/applications

# 以后台模式启动所有应用服务
docker-compose up -d

# 检查所有应用服务是否正常运行
docker-compose ps
```

## 4. 停止环境

停止环境时，建议遵循与启动相反的顺序。

```bash
# 停止并移除应用服务容器
cd deployment/applications
docker-compose down

# 停止并移除基础设施容器
cd deployment/infrastructure
docker-compose down --volumes # 可选，--volumes 会删除数据卷
```

## 5. 访问端点

部署成功后，可以通过以下地址访问相关的管理界面：

| 服务        | 地址                   | 凭据                          |
|-------------|---------------------------|-------------------------------|
| Grafana     | `http://localhost:3000`   | `admin` / `gochat_grafana_2024` |
| Prometheus  | `http://localhost:9090`   | -                             |
| Kafka UI    | `http://localhost:8080`   | -                             |
| ...         | ...                       | ...                           |

*注意: 根据新的监控架构，Jaeger 已被移除。*