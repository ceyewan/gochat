# GoChat 文档中心

欢迎使用 GoChat 项目的官方文档。本文档中心为开发人员、运维人员和项目经理提供了全面的指南。

## 1. 核心概念

-   **[架构概览](./01_architecture/01_overview.md)**：系统微服务架构和数据流的高级视图。
-   **[数据流图](./01_architecture/02_dataflow.md)**：核心业务流程的详细图表。
-   **[技术栈](./01_architecture/03_tech_stack.md)**：项目中使用的技术和框架列表。

## 2. API 和接口定义

-   **[HTTP 和 WebSocket API](./02_apis/01_http_ws_api.md)**：定义客户端应用程序的外部 API。
-   **[内部 gRPC 服务](./02_apis/02_grpc_services.md)**：记录微服务之间的内部 RPC 通信。
-   **[消息队列主题](./02_apis/03_mq_topics.md)**：定义 Kafka 主题和消息模式。

## 3. 开发指南

-   **[开发工作流](./03_development/01_workflow.md)**：概述编码、分支和拉取请求的标准。
-   **[代码风格和约定](./03_development/02_style_guide.md)**：定义编码风格、格式化和注释标准。
-   **[微服务开发](./03_development/03_service_guide.md)**：开发和测试单个微服务的指南。
-   **[使用 `im-infra` 组件](./03_development/04_infra_components.md)**：如何使用共享的基础设施组件，如 `clog` 和 `coord`。

## 4. 部署和运维

-   **[部署指南](./04_deployment/01_deployment.md)**：使用 Docker Compose 部署系统的分步说明。
-   **[配置管理](./04_deployment/02_configuration.md)**：如何使用 `etcd` 和 `config-cli` 工具管理服务配置。
-   **[日志记录和监控](./04_deployment/03_logging_monitoring.md)**：使用日志记录（Loki）和监控（Prometheus、Grafana）堆栈的指南。

## 5. 特定服务文档

-   **[im-gateway](./05_services/im-gateway.md)**
-   **[im-logic](./05_services/im-logic.md)**
-   **[im-repo](./05_services/im-repo.md)**
-   **[im-task](./05_services/im-task.md)**
-   **[im-infra](./05_services/im-infra.md)**

## 6. 数据模型

-   **[数据库架构](./06_data_models/01_db_schema.md)**：关于 MySQL 数据库表和关系的详细信息。

## 7. 基础设施组件

-   **[基础设施层 (im-infra) 总览](./08_infra/README.md)**: `im-infra` 库的整体设计原则、通用架构模式及组件列表。
-   **[MQ 消息队列](./08_infra/mq.md)**: `mq` 组件的接口设计与使用指南。
-   **[Clog 结构化日志](./08_infra/clog.md)**: `clog` 组件的设计理念与使用方法。
-   **[Coord 分布式协调](./08_infra/coord.md)**: `coord` 组件的使用与 `instanceID` 设计方案。
-   **[Cache 分布式缓存](./08_infra/cache.md)**: `cache` 组件的设计理念与使用指南。
-   **[UID 唯一ID生成](./08_infra/uid.md)**: `uid` 组件的设计理念与使用指南。
-   **[DB 数据库操作](./08_infra/db.md)**: `db` 组件的设计理念与使用指南。
-   **[Metrics 可观测性](./08_infra/metrics.md)**: `metrics` 组件的设计理念与使用指南。
-   **[Once 幂等操作](./08_infra/once.md)**: `once` 组件的设计理念与使用指南。
-   **[RateLimit 分布式限流](./08_infra/ratelimit.md)**: `ratelimit` 组件的设计理念与使用指南。

本文档旨在成为 GoChat 项目的唯一真实来源。所有团队成员都应该阅读、理解并为其做出贡献。