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

## 3. 领域事件主题

除了核心的消息收发链路，系统还定义了一组用于广播领域事件的 Topic。这些 Topic 用于服务间的解耦，允许非核心服务（如数据分析、风控、审计等）订阅系统内发生的重要事件。

### 3.1 用户事件

-   **主题名称**: `gochat.user-events`
-   **描述**: 用于广播与用户状态和行为相关的事件。
-   **典型事件**:
    -   用户上线/下线
    -   用户个人资料更新
-   **主要生产者**: `im-gateway`, `im-repo`
-   **主要消费者**: 数据分析服务、好友关系服务（用于更新在线状态）
-   **建议消息体**:
    ```json
    {
      "event_id": "uuid-v4",
      "event_type": "user.online", // or "user.offline", "user.profile.updated"
      "timestamp": 1678886400,
      "user_id": "user-123",
      "payload": {
        "last_seen": 1678886400,
        "gateway_id": "gateway-abc"
      }
    }
    ```

### 3.2 消息事件

-   **主题名称**: `gochat.message-events`
-   **描述**: 用于广播与消息生命周期相关的事件。
-   **典型事件**:
    -   消息已读
    -   消息被撤回
-   **主要生产者**: `im-logic`
-   **主要消费者**: 数据分析服务（统计已读率）、内容审核服务
-   **建议消息体**:
    ```json
    {
      "event_id": "uuid-v4",
      "event_type": "message.read", // or "message.recalled"
      "timestamp": 1678886400,
      "conversation_id": "conv-456",
      "operator_id": "user-123", // 执行操作的用户
      "payload": {
        "message_id": 789,
        "seq_id": 102
      }
    }
    ```

### 3.3 系统通知

-   **主题名称**: `gochat.notifications`
-   **描述**: 用于发送非聊天消息类的业务通知，这些通知通常需要被推送给用户。
-   **典型事件**:
    -   用户被拉入新群聊
    -   收到新的好友申请
    -   群公告更新
-   **主要生产者**: `im-logic`
-   **主要消费者**: 专用的通知服务 (`notification-service`)，负责将事件转换为 App 推送、短信或邮件。
-   **建议消息体**:
    ```json
    {
      "event_id": "uuid-v4",
      "event_type": "group.invited",
      "timestamp": 1678886400,
      "target_user_id": "user-123",
      "payload": {
        "group_id": "group-789",
        "inviter_id": "user-456",
        "group_name": "技术交流群"
      }
    }
    ```

## 4. 消费者组

-   **`logic.upstream.group`**: `im-logic` 服务实例组成的消费者组，用于消费 `gochat.messages.upstream`。
-   **`gateway.downstream.group.{instanceID}`**: 每个 `im-gateway` 实例的专属消费者组，用于消费自己的下行主题。
-   **`task.persist.group`**: `im-task` 服务实例组成的消费者组，用于消费 `gochat.messages.persist` 主题并进行持久化。
-   **`task.fanout.group`**: `im-task` 服务实例组成的消费者组，用于处理超大群的扇出任务。
-   **`analytics.events.group`**: (示例) 数据分析服务组成的消费者组，用于统一消费所有 `gochat.*-events` 主题。
