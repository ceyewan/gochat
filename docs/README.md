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
-   **[核心业务流程](./06_data_models/02_core_im_flows.md)**：详细阐述 IM 核心功能（如消息收发、在线状态）的实现流程。

## 7. 近期重构计划 (Roadmap)

- **[重构任务列表](./07_todo_task/README.md)**: 本目录包含了项目后续的核心重构和开发计划书，是了解项目演进方向的重要参考。

## 8. 基础设施组件 (`im-infra`)

`im-infra` 是 GoChat 项目的基石，提供了一系列高质量、生产级别的基础组件。

-   **[总览与设计规范](./08_infra/README.md)**: `im-infra` 库的整体设计原则、通用架构模式及组件列表。
-   **[快速上手指南 (Usage Guide)](./08_infra/usage_guide.md)**: **(首选阅读)** 提供了覆盖所有组件的、生产级别的统一初始化范例和核心用法。

以下是每个核心组件的官方“契约”文档 (按字母排序):

-   **[熔断器 (breaker)](./08_infra/breaker.md)**
-   **[缓存 (cache)](./08_infra/cache.md)**
-   **[日志 (clog)](./08_infra/clog.md)**
-   **[分布式协调 (coord)](./08_infra/coord.md)**
-   **[数据库 (db)](./08_infra/db.md)**
-   **[可观测性 (metrics)](./08_infra/metrics.md)**
-   **[消息队列 (mq)](./08_infra/mq.md)**
-   **[幂等操作 (once)](./08_infra/once.md)**
-   **[分布式限流 (ratelimit)](./08_infra/ratelimit.md)**
-   **[唯一ID (uid)](./08_infra/uid.md)**

本文档旨在成为 GoChat 项目的唯一真实来源。所有团队成员都应该阅读、理解并为其做出贡献。