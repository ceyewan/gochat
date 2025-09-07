# GoChat Deployment Guide

This document provides instructions for deploying the GoChat system using the provided Docker Compose setup.

## 1. Architecture

The deployment is split into two main parts:

-   **Infrastructure**: Contains all the backend services required by the application, such as databases, caches, and message queues.
-   **Applications**: Contains the GoChat microservices themselves.

This separation allows the infrastructure to be started once and remain running while the application services can be rebuilt and restarted independently during development.

## 2. Prerequisites

-   [Docker](https://www.docker.com/get-started)
-   [Docker Compose](https://docs.docker.com/compose/install/)

## 3. Quick Start

### Step 1: Start the Infrastructure

Navigate to the `deployment` directory and run the `start-infra.sh` script. This will start all the necessary backend services.

```bash
cd deployment
./scripts/start-infra.sh
```

The infrastructure stack includes:
-   etcd (for configuration and service discovery)
-   Kafka (for messaging)
-   MySQL (for data persistence)
-   Redis (for caching)
-   Loki, Prometheus, Grafana, Jaeger (for monitoring and logging)

### Step 2: Build and Start the Application Services

Each microservice has its own `Dockerfile` for building a container image. The `start-apps.sh` script will build and start all the application services.

```bash
cd deployment
./scripts/start-apps.sh
```

This will start the following services:
-   `im-gateway`
-   `im-logic`
-   `im-repo`
-   `im-task`

### Step 3: Verify the Deployment

Use the `health-check.sh` script to verify that all services are running correctly.

```bash
cd deployment
./scripts/health-check.sh
```

## 4. Service Endpoints

Once all services are running, you can access them at the following addresses:

### Management UIs

| Service        | Address                   | Credentials               |
| -------------- | ------------------------- | ------------------------- |
| Grafana        | `http://localhost:3000`   | `admin`/`gochat_grafana_2024` |
| Kafka UI       | `http://localhost:8080`   | -                         |
| etcd Manager   | `http://localhost:8081`   | -                         |
| RedisInsight   | `http://localhost:8001`   | -                         |
| phpMyAdmin     | `http://localhost:8083`   | -                         |
| Jaeger         | `http://localhost:16686`  | -                         |

### Application API

| Service      | Address                 |
| ------------ | ----------------------- |
| **GoChat API** | `http://localhost:8080` |
| **WebSocket**  | `ws://localhost:8080/ws`  |

## 5. Stopping the Environment

To stop the services, use the `cleanup.sh` script.

```bash
cd deployment

# Stop only the application services
./scripts/cleanup.sh --apps

# Stop all services (infrastructure and applications)
./scripts/cleanup.sh --all

# Stop all services and remove data volumes
./scripts/cleanup.sh --all --remove-volumes
```

## 6. Individual Service Management

You can manage individual services using `docker-compose` commands directly.

```bash
# View logs for a specific service
docker-compose -f applications/docker-compose.yml logs -f im-logic

# Rebuild and restart a single service
docker-compose -f applications/docker-compose.yml up -d --build im-gateway
