# GoChat Architecture Overview

This document provides a high-level overview of the GoChat system architecture, its components, and the flow of data between them.

## 1. System Goal

The primary goal of GoChat is to provide a reliable, scalable, and low-latency instant messaging service. It supports core features like user registration, real-time one-on-one chat, group chat, message history, and notifications.

## 2. Microservice Architecture

The system is designed using a microservice architecture to ensure separation of concerns, independent scalability, and maintainability. The core services are:

-   **`im-gateway`**: The API Gateway is the single entry point for all clients (web, mobile). It handles client connections (HTTP/WebSocket), translates protocols, and routes requests to the appropriate backend services. It is responsible for managing persistent WebSocket connections for real-time communication.

-   **`im-logic`**: This service contains the core business logic of the application. Its responsibilities include user authentication (login, registration, guest access), session management, conversation and group logic, and composing responses by fetching data from other services. It is the central orchestrator for most user-facing operations.

-   **`im-repo`**: The Repository Service is responsible for all data persistence. It abstracts the database and cache layers, providing a clean gRPC API for other services to access and manipulate data. It directly interacts with MySQL for persistent storage and Redis for caching and managing transient data like online status.

-   **`im-task`**: The Task Service handles all asynchronous background processing. Its primary role is to consume messages from a message queue (Kafka) and perform tasks that do not need to be processed synchronously, such as message fan-out to multiple recipients in a group chat or pushing notifications. This decouples the core messaging flow and improves system resilience and performance.

## 3. Data Flow: Sending a Message

The following diagram illustrates the data flow when a user sends a message:

```
[Client] --(WebSocket)--> [im-gateway] --(gRPC)--> [im-logic]
    ^                                                   |
    |                                                   v
    |                                             [im-repo (DB/Cache)]
    |                                                   |
    |                                                   v
    +------------------ [im-task] <--(Kafka)-- [im-logic]
            (WebSocket)      ^
                             | (gRPC)
                             |
                       [im-repo (Cache)]
```

**Steps:**

1.  **Client to Gateway**: A user sends a message via a persistent WebSocket connection to `im-gateway`.
2.  **Gateway to Logic**: `im-gateway` receives the message and forwards it to `im-logic` via a gRPC call for processing.
3.  **Logic to Repo (Save)**: `im-logic` performs validation, requests a persistent sequence ID from `im-repo`, and then sends the message to `im-repo` to be saved in the MySQL database.
4.  **Logic to Message Queue**: After the message is saved, `im-logic` publishes a "message dispatch" event to a Kafka topic.
5.  **Task Consumes Event**: `im-task` is subscribed to the Kafka topic and consumes the dispatch event.
6.  **Task to Repo (Query)**: `im-task` determines all recipients of the message. It queries `im-repo` to get the online status of each recipient, which includes the specific `im-gateway` instance they are connected to.
7.  **Task to Gateway (Push)**: `im-task` pushes the message to the appropriate `im-gateway` instances for each online recipient.
8.  **Gateway to Client**: The `im-gateway` instances deliver the message to the online recipients over their respective WebSocket connections.
