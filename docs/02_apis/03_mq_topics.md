# GoChat 消息队列 (MQ) 主题

本文档定义了 GoChat 系统中用于微服务之间异步通信的 Apache Kafka 主题和消息格式。

## 1. 概述

Kafka 用于解耦服务并处理异步任务，例如将消息扩散到多个接收者。这提高了系统的弹性和可扩展性。

## 2. 主题与消息流

### 2.1 上行消息

-   **主题名称**: `gochat.messages.upstream`
-   **生产者**: `im-gateway`
-   **消费者**: `im-logic`
-   **描述**: 所有客户端发送的原始消息都进入此主题。这是一个单一的、共享的主题。
-   **消息体**: `UpstreamMessage` (定义于 `api/kafka/message.go`)

### 2.2 下行消息 (单聊/中小群)

-   **主题名称**: `gochat.messages.downstream.{instanceID}`
-   **生产者**: `im-logic`
-   **消费者**:
    1.  `im-gateway` (特定实例)
    2.  `im-task` (通过通配符订阅 `gochat.messages.downstream.*`)
-   **描述**: `im-logic` 处理完单聊或中小群消息后，将其发送到目标用户所在的 `im-gateway` 实例的专属 Topic。`im-task` 服务也会消费这些消息以进行持久化。
-   **消息体**: `DownstreamMessage` (定义于 `api/kafka/message.go`)

### 2.3 异步任务 (超大群扇出)

-   **主题名称**: `gochat.tasks.fanout`
-   **生产者**: `im-logic`
-   **消费者**: `im-task`
-   **描述**: 当 `im-logic` 判断为超大群消息时，它会将一个轻量级的扇出任务发送到此主题。
-   **消息体**: `FanoutTask` (定义于 `api/kafka/message.go`)
-   **后续流程**: `im-task` 消费此任务后，会获取群成员列表，并为每个在线成员所在的 `im-gateway` 生产一条下行消息到对应的 `gochat.messages.downstream.{instanceID}` 主题。

## 3. 消费者组

-   **`logic.upstream.group`**: `im-logic` 服务实例组成的消费者组，用于消费 `gochat.messages.upstream`。
-   **`gateway.downstream.group.{instanceID}`**: 每个 `im-gateway` 实例的专属消费者组，用于消费自己的下行主题。
-   **`task.persist.group`**: `im-task` 服务实例组成的消费者组，用于消费 `gochat.messages.persist` 主题并进行持久化。
-   **`task.fanout.group`**: `im-task` 服务实例组成的消费者组，用于处理超大群的扇出任务。