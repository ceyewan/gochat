# GoChat 技术栈

本文档列出了 GoChat 项目中使用的关键技术、框架和库。

## 1. 后端

-   **编程语言**: [Go](https://golang.org/)
    -   选择它是因为其性能、并发模型和强大的标准库，使其非常适合高吞吐量的网络服务。
-   **Web 框架**: [Gin](https://gin-gonic.com/)
    -   一个高性能、极简的 Web 框架，在 `im-gateway` 中用于处理 HTTP 请求。
-   **WebSocket 库**: [Gorilla WebSocket](https://github.com/gorilla/websocket)
    -   一个广泛使用的 Go WebSocket 实现，在 `im-gateway` 中用于管理实时客户端连接。
-   **gRPC 框架**: [gRPC-Go](https://grpc.io/)
    -   用于微服务之间的所有内部 RPC 通信，提供高性能、强类型化的 Protobuf 契约。
-   **数据库 ORM**: [GORM](https://gorm.io/)
    -   在 `im-repo` 中用于与 MySQL 数据库交互的主要 ORM。

## 2. 数据存储和消息传递

-   **数据库**: [MySQL](https://www.mysql.com/)
    -   用于存储持久化数据（如用户、消息和群组）的主要关系型数据库。
-   **缓存**: [Redis](https://redis.io/)
    -   用于缓存频繁访问的数据和管理瞬时状态，如用户在线状态。
-   **消息队列**: [Apache Kafka](https://kafka.apache.org/)
    -   异步消息系统的骨干，用于解耦服务并处理消息扩散等任务。
-   **配置和发现**: [etcd](https://etcd.io/)
    -   用作服务发现和配置管理后端的分布式键值存储。

## 3. 工具和部署

-   **容器化**: [Docker](https://www.docker.com/) 和 [Docker Compose](https://docs.docker.com/compose/)
    -   用于容器化所有微服务和基础设施组件，以实现一致的开发和部署环境。
-   **API 定义**: [Protocol Buffers (Protobuf)](https://developers.google.com/protocol-buffers)
    -   用于定义 gRPC 服务契约。
-   **构建工具**: [Buf](https://buf.build/)
    -   用于从 Protobuf 定义构建、检查和生成代码。
-   **依赖管理**: Go Modules

## 4. 监控和日志

-   **日志聚合**: [Loki](https://grafana.com/oss/loki/)
    -   中央日志聚合系统，专为 Grafana 生态设计的高效日志存储。
-   **日志收集器**: [Promtail](https://grafana.com/docs/loki/latest/clients/promtail/)
    -   Grafana Loki 的官方日志收集代理，从所有容器收集日志并转发到 Loki。
-   **指标监控**: [Prometheus](https://prometheus.io/)
    -   用于收集和存储应用程序及系统指标的时间序列数据库。
-   **可视化与告警**: [Grafana](https://grafana.com/)
    -   用于可视化 Loki 日志和 Prometheus 指标的统一仪表板，支持告警和通知。
-   **分布式追踪**: [Jaeger](https://www.jaegertracing.io/) *(计划中)*
    -   用于追踪请求在微服务中的传播，有助于调试和性能分析。