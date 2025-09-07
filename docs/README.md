# GoChat Documentation Hub

Welcome to the official documentation for the GoChat project. This documentation hub provides a comprehensive guide for developers, operators, and project managers.

## 1. Core Concepts

-   **[Architecture Overview](./01_architecture/01_overview.md)**: A high-level look at the system's microservice architecture and data flow.
-   **[Data Flow Diagrams](./01_architecture/02_dataflow.md)**: Detailed diagrams of core business processes.
-   **[Technology Stack](./01_architecture/03_tech_stack.md)**: A list of the technologies and frameworks used in the project.

## 2. API and Interface Definitions

-   **[HTTP & WebSocket API](./02_apis/01_http_ws_api.md)**: Defines the external API for client applications.
-   **[Internal gRPC Services](./02_apis/02_grpc_services.md)**: Documents the internal RPC communication between microservices.
-   **[Message Queue Topics](./02_apis/03_mq_topics.md)**: Defines the Kafka topics and message schemas.

## 3. Development Guide

-   **[Development Workflow](./03_development/01_workflow.md)**: Outlines the standards for coding, branching, and pull requests.
-   **[Code Style and Conventions](./03_development/02_style_guide.md)**: Defines the coding style, formatting, and commenting standards.
-   **[Microservice Development](./03_development/03_service_guide.md)**: A guide to developing and testing individual microservices.
-   **[Using `im-infra` Components](./03_development/04_infra_components.md)**: How to use the shared infrastructure components like `clog` and `coord`.

## 4. Deployment and Operations

-   **[Deployment Guide](./04_deployment/01_deployment.md)**: Step-by-step instructions for deploying the system using Docker Compose.
-   **[Configuration Management](./04_deployment/02_configuration.md)**: How to manage service configurations using `etcd` and the `config-cli` tool.
-   **[Logging and Monitoring](./04_deployment/03_logging_monitoring.md)**: A guide to using the logging (Loki) and monitoring (Prometheus, Grafana) stack.

## 5. Service-Specific Documentation

-   **[im-gateway](./05_services/im-gateway.md)**
-   **[im-logic](./05_services/im-logic.md)**
-   **[im-repo](./05_services/im-repo.md)**
-   **[im-task](./05_services/im-task.md)**
-   **[im-infra](./05_services/im-infra.md)**

## 6. Data Models

-   **[Database Schema](./06_data_models/01_db_schema.md)**: Detailed information about the MySQL database tables and relationships.

This documentation is intended to be the single source of truth for the GoChat project. All team members are expected to read, understand, and contribute to it.
