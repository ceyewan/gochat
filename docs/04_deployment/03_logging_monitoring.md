# Logging and Monitoring

This document provides a guide to the logging and monitoring infrastructure of the GoChat system.

## 1. Architecture

The monitoring and logging stack is designed to provide comprehensive visibility into the system's health and performance.

-   **Logging**: `Application -> Vector -> Loki -> Grafana`
-   **Metrics**: `Application -> Prometheus -> Grafana`
-   **Tracing**: `Application -> Jaeger`

## 2. Component Overview

-   **Vector**: A high-performance data pipeline that collects logs from all application containers and forwards them to Loki.
-   **Loki**: A horizontally-scalable, multi-tenant log aggregation system inspired by Prometheus. It indexes metadata about the logs, not the full log content, making it efficient.
-   **Prometheus**: A time-series database that scrapes and stores metrics from all services.
-   **Grafana**: A unified dashboard for querying, visualizing, and alerting on logs from Loki and metrics from Prometheus.
-   **Jaeger**: A distributed tracing system used to monitor and troubleshoot transactions in complex distributed systems.

## 3. Accessing Dashboards

| Service   | Address                 | Credentials               | Purpose                     |
| --------- | ----------------------- | ------------------------- | --------------------------- |
| Grafana   | `http://localhost:3000` | `admin`/`gochat_grafana_2024` | Unified Logs & Metrics      |
| Jaeger    | `http://localhost:16686`| -                         | Distributed Tracing         |
| Prometheus| `http://localhost:9090` | -                         | Metrics Querying            |

## 4. How to Use

### Viewing Logs

1.  Navigate to Grafana: `http://localhost:3000`
2.  Log in with the credentials provided above.
3.  Go to the "Explore" view.
4.  Select the "Loki" data source.
5.  Use the "Log browser" or write a LogQL query to find logs.

**Example LogQL Queries:**

-   Show all logs from the `im-logic` service:
    `{service="im-logic"}`
-   Show all error-level logs from any service:
    `{level="ERROR"}`
-   Find logs containing the text "failed to connect":
    `{} |= "failed to connect"`

### Viewing Metrics

1.  Navigate to Grafana: `http://localhost:3000`
2.  Find the pre-built dashboards for GoChat services.
3.  Alternatively, go to the "Explore" view and select the "Prometheus" data source to build custom queries.

### Viewing Traces

1.  Navigate to Jaeger: `http://localhost:16686`
2.  Select a service from the "Service" dropdown.
3.  Click "Find Traces" to see recent requests.
4.  Click on a trace to see a detailed breakdown of the request's lifecycle across microservices.

## 5. Application Logging Configuration

For logs to be correctly collected and parsed, applications must:

1.  **Log to a file**: The `clog` library is configured to log to a file inside the container (e.g., `/app/logs/app.log`).
2.  **Use JSON format**: The logger must be configured to output logs in JSON format.
3.  **Include standard fields**: All logs should include standard fields like `service`, `level`, and `trace_id` for effective filtering and correlation in Grafana.

See the [Code Style and Conventions](./../03_development/02_style_guide.md) for more details on logging best practices.
