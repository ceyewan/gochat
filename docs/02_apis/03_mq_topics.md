# GoChat 消息队列 (MQ) 主题

本文档定义了 GoChat 系统中用于微服务之间异步通信的 Apache Kafka 主题和消息格式。

## 1. 概述

Kafka 用于解耦服务并处理异步任务，例如将消息扩散到多个接收者。这提高了系统的弹性和可扩展性。

## 2. 主题

### `gochat.messages.dispatch`

-   **描述**: 这是消息扩散的主要主题。当用户发送消息时，`im-logic` 在将其保存到数据库后将其发布到此主题。`im-task` 服务从此主题消费以处理对所有接收者的传递。
-   **分区**: 建议此主题具有多个分区，以允许多个 `im-task` 实例并发消费。可以按 `conversation_id` 进行分区，以确保同一会话中的消息按顺序处理。

### `gochat.notifications.push`

-   **描述**: 此主题用于向离线用户发送推送通知。当 `im-task` 确定接收者离线时，它可以向此主题发布消息。专用的通知服务（尚未实现）将从此主题消费，通过 Firebase Cloud Messaging (FCM) 或 Apple Push Notification Service (APNS) 等服务发送推送通知。

## 3. 消息模式

所有 Kafka 消息结构的最终真实来源是 Go 代码本身，位于 `api/kafka/message.go` 文件中。这种方法在生产者和消费者服务之间提供了强类型的编译时契约。

-   **主要真实来源**: [`/api/kafka/message.go`](../../../api/kafka/message.go)

开发者在生产或消费消息时**必须**导入并使用此文件中定义的结构体，以确保一致性和类型安全。下面的文档提供了高级概述，但 Go 文件应始终被视为权威定义。

### `UpstreamMessage`

-   **主题**: `im-upstream-topic`
-   **描述**: 从客户端通过 `im-gateway` 发送的消息，由 `im-logic` 处理。
-   **关键字段**: `TraceID`, `UserID`, `ConversationID`, `Content`。

### `DownstreamMessage`

-   **主题**: `im-downstream-topic-{gateway_id}`
-   **描述**: 从 `im-logic` 或 `im-task` 发送到特定 `im-gateway` 实例的处理后消息，用于传递给客户端。
-   **关键字段**: `TargetUserID`, `MessageID`, `ConversationID`, `Content`。

### `TaskMessage`

-   **主题**: `im-task-topic`
-   **描述**: 从 `im-logic` 发送到 `im-task` 的异步任务，用于后台处理。
-   **关键字段**: `TaskType`, `TaskID`, `Data`（包含任务特定的有效载荷，如 `FanoutTaskData`）。

## 4. 消费者组

-   **`im-task-dispatch-group`**: 从 `gochat.messages.dispatch` 主题消费的 `im-task` 实例的消费者组。
-   **`notification-service-push-group`**: 未来通知服务从 `gochat.notifications.push` 主题消费的消费者组。