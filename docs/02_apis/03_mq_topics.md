# GoChat Message Queue (MQ) Topics

This document defines the Apache Kafka topics and message formats used for asynchronous communication between microservices in the GoChat system.

## 1. Overview

Kafka is used to decouple services and handle asynchronous tasks, such as fanning out messages to multiple recipients. This improves system resilience and scalability.

## 2. Topics

### `gochat.messages.dispatch`

-   **Description**: This is the primary topic for message fan-out. When a user sends a message, `im-logic` publishes it to this topic after it has been saved to the database. The `im-task` service consumes from this topic to handle the delivery to all recipients.
-   **Partitions**: It is recommended to have multiple partitions for this topic to allow for concurrent consumption by multiple `im-task` instances. Partitioning can be done by `conversation_id` to ensure messages within the same conversation are processed in order.

### `gochat.notifications.push`

-   **Description**: This topic is used for sending push notifications to users who are offline. When `im-task` determines that a recipient is offline, it can publish a message to this topic. A dedicated notification service (not yet implemented) would consume from this topic to send push notifications via services like Firebase Cloud Messaging (FCM) or Apple Push Notification Service (APNS).

## 3. Message Schemas

The definitive source of truth for all Kafka message structures is the Go code itself, located in the `api/kafka/message.go` file. This approach provides a strong, compile-time contract between the producer and consumer services.

-   **Primary Source of Truth**: [`/api/kafka/message.go`](../../../api/kafka/message.go)

Developers **must** import and use the structs defined in this file when producing or consuming messages to ensure consistency and type safety. The documentation below provides a high-level overview, but the Go file should always be considered the canonical definition.

### `UpstreamMessage`

-   **Topic**: `im-upstream-topic`
-   **Description**: A message sent from a client via `im-gateway` to be processed by `im-logic`.
-   **Key Fields**: `TraceID`, `UserID`, `ConversationID`, `Content`.

### `DownstreamMessage`

-   **Topic**: `im-downstream-topic-{gateway_id}`
-   **Description**: A processed message sent from `im-logic` or `im-task` to a specific `im-gateway` instance for delivery to a client.
-   **Key Fields**: `TargetUserID`, `MessageID`, `ConversationID`, `Content`.

### `TaskMessage`

-   **Topic**: `im-task-topic`
-   **Description**: An asynchronous task sent from `im-logic` to `im-task` for background processing.
-   **Key Fields**: `TaskType`, `TaskID`, `Data` (contains task-specific payload like `FanoutTaskData`).

## 4. Consumer Groups

-   **`im-task-dispatch-group`**: The consumer group for `im-task` instances consuming from the `gochat.messages.dispatch` topic.
-   **`notification-service-push-group`**: The consumer group for the future notification service consuming from the `gochat.notifications.push` topic.
