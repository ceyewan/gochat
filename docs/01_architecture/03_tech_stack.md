# GoChat Technology Stack

This document lists the key technologies, frameworks, and libraries used in the GoChat project.

## 1. Backend

-   **Programming Language**: [Go](https://golang.org/)
    -   Chosen for its performance, concurrency model, and strong standard library, making it well-suited for high-throughput network services.
-   **Web Framework**: [Gin](https://gin-gonic.com/)
    -   A high-performance, minimalist web framework used in `im-gateway` for handling HTTP requests.
-   **WebSocket Library**: [Gorilla WebSocket](https://github.com/gorilla/websocket)
    -   A widely-used WebSocket implementation for Go, used in `im-gateway` to manage real-time client connections.
-   **gRPC Framework**: [gRPC-Go](https://grpc.io/)
    -   Used for all internal RPC communication between microservices, providing high-performance, strongly-typed contracts with Protobuf.
-   **Database ORM**: [GORM](https://gorm.io/)
    -   The primary ORM used in `im-repo` for interacting with the MySQL database.

## 2. Data Stores & Messaging

-   **Database**: [MySQL](https://www.mysql.com/)
    -   The primary relational database for storing persistent data like users, messages, and groups.
-   **Cache**: [Redis](https://redis.io/)
    -   Used for caching frequently accessed data and managing transient state, such as user online status.
-   **Message Queue**: [Apache Kafka](https://kafka.apache.org/)
    -   The backbone of the asynchronous messaging system, used to decouple services and handle tasks like message fan-out.
-   **Configuration & Discovery**: [etcd](https://etcd.io/)
    -   A distributed key-value store used as the service discovery and configuration management backend.

## 3. Tooling & Deployment

-   **Containerization**: [Docker](https://www.docker.com/) & [Docker Compose](https://docs.docker.com/compose/)
    -   Used to containerize all microservices and infrastructure components for consistent development and deployment environments.
-   **API Definition**: [Protocol Buffers (Protobuf)](https://developers.google.com/protocol-buffers)
    -   Used to define the gRPC service contracts.
-   **Build Tooling**: [Buf](https://buf.build/)
    -   Used to build, lint, and generate code from the Protobuf definitions.
-   **Dependency Management**: Go Modules

## 4. Monitoring & Logging

-   **Logging**: [Loki](https://grafana.com/oss/loki/)
    -   The central log aggregation system.
-   **Log Shipper**: [Vector](https://vector.dev/)
    -   Collects logs from all services and forwards them to Loki.
-   **Metrics**: [Prometheus](https://prometheus.io/)
    -   The time-series database for storing application and system metrics.
-   **Visualization**: [Grafana](https://grafana.com/)
    -   The unified dashboard for visualizing logs from Loki and metrics from Prometheus.
-   **Distributed Tracing**: [Jaeger](https://www.jaegertracing.io/)
    -   Used to trace requests as they travel through the microservices, aiding in debugging and performance analysis.
