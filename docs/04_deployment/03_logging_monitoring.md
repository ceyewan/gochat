# 日志记录和监控

本文档提供了 GoChat 系统的日志记录和监控基础设施指南。

## 1. 架构

监控和日志记录堆栈旨在提供对系统健康和性能的全面可见性。

-   **日志记录**: `应用程序 -> Vector -> Loki -> Grafana`
-   **指标**: `应用程序 -> Prometheus -> Grafana`
-   **追踪**: `应用程序 -> Jaeger`

## 2. 组件概述

-   **Vector**: 一个高性能数据管道，从所有应用程序容器收集日志并将其转发到 Loki。
-   **Loki**: 一个受 Prometheus 启发的水平可扩展、多租户日志聚合系统。它索引关于日志的元数据，而不是完整的日志内容，使其高效。
-   **Prometheus**: 一个时间序列数据库，抓取并存储来自所有服务的指标。
-   **Grafana**: 一个统一的仪表板，用于查询、可视化和警报来自 Loki 的日志和来自 Prometheus 的指标。
-   **Jaeger**: 一个分布式追踪系统，用于监控和排除复杂分布式系统中的事务故障。

## 3. 访问仪表板

| 服务   | 地址                 | 凭据               | 目的                     |
| --------- | ----------------------- | ------------------------- | --------------------------- |
| Grafana   | `http://localhost:3000` | `admin`/`gochat_grafana_2024` | 统一日志和指标      |
| Jaeger    | `http://localhost:16686`| -                         | 分布式追踪         |
| Prometheus| `http://localhost:9090` | -                         | 指标查询            |

## 4. 如何使用

### 查看日志

1.  导航到 Grafana：`http://localhost:3000`
2.  使用上面提供的凭据登录。
3.  转到"Explore"视图。
4.  选择"Loki"数据源。
5.  使用"Log browser"或编写 LogQL 查询来查找日志。

**示例 LogQL 查询：**

-   显示来自 `im-logic` 服务的所有日志：
    `{service="im-logic"}`
-   显示来自任何服务的所有错误级别日志：
    `{level="ERROR"}`
-   查找包含文本"failed to connect"的日志：
    `{} |= "failed to connect"`

### 查看指标

1.  导航到 Grafana：`http://localhost:3000`
2.  找到 GoChat 服务的预构建仪表板。
3.  或者，转到"Explore"视图并选择"Prometheus"数据源来构建自定义查询。

### 查看追踪

1.  导航到 Jaeger：`http://localhost:16686`
2.  从"Service"下拉列表中选择一个服务。
3.  单击"Find Traces"以查看最近的请求。
4.  单击追踪以查看请求在微服务中生命周期的详细分解。

## 5. 应用程序日志记录配置

为了正确收集和解析日志，应用程序必须：

1.  **记录到文件**: `clog` 库配置为记录到容器内的文件（例如，`/app/logs/app.log`）。
2.  **使用 JSON 格式**: 日志记录器必须配置为以 JSON 格式输出日志。
3.  **包含标准字段**: 所有日志应包括标准字段，如 `service`、`level` 和 `trace_id`，以便在 Grafana 中进行有效的过滤和关联。

有关日志记录最佳实践的更多详细信息，请参阅[代码风格和约定](./../03_development/02_style_guide.md)。