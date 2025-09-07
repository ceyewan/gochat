# `im-logic`: 核心业务逻辑服务

`im-logic` 是 GoChat 系统的“中央司令部”，负责处理所有核心的、实时的业务逻辑。它消费上行消息，并根据业务规则决定如何处理和分发。

## 1. 核心职责

-   **消息消费与处理**: 作为 `gochat.messages.upstream` 主题的消费者，处理所有上行业务逻辑。
-   **消息生产与分发**:
    -   对于单聊和中小群，向 `gochat.messages.downstream.{instanceID}` 主题生产下行消息。
    -   对于超大群，向 `gochat.tasks.fanout` 主题生产异步任务。
-   **业务决策**: 它是业务规则的执行者。例如，根据群规模决定采用实时扩散还是异步任务。
-   **数据读取编排**: 编排对 `im-repo` 的**只读**调用，以获取业务决策所需的数据（如群成员、用户在线状态等）。
-   **无持久化职责**: `im-logic` **绝不**直接或间接地执行任何数据写入或修改操作。所有持久化任务都由 `im-task` 服务负责。

## 2. 核心工作流程

### 2.1 中小群消息处理

```mermaid
sequenceDiagram
    participant Kafka as Kafka
    participant Logic as im-logic
    participant Repo as im-repo

    Logic->>-Kafka: 1. 消费上行消息 (gochat.messages.upstream)
    Logic->>+Repo: 2. gRPC 调用: GetGroupMembersAndOnlineStatus(ctx, req)
    Repo-->>-Logic: 3. 返回在线成员与网关映射
    Logic->>Logic: 4. 业务处理 (生成 message_id 等)
    Logic->>+Kafka: 5. **循环/批量**生产下行消息到多个网关Topic (gochat.messages.downstream.{gateway_id})
```

### 2.2 超大群消息处理

```mermaid
sequenceDiagram
    participant Kafka as Kafka
    participant Logic as im-logic

    Logic->>-Kafka: 1. 消费上行消息 (gochat.messages.upstream)
    Logic->>Logic: 2. 判断为超大群
    Logic->>+Kafka: 3. 生产异步任务到 `gochat.tasks.fanout`
```

## 3. 依赖关系

-   **`im-infra/coord`**: 用于服务发现，连接到 `im-repo`。
-   **`im-infra/clog`**: 用于结构化日志记录。
-   **`im-infra/mq`**: 同时作为消费者和生产者。
-   **`im-repo` 服务**: **只读**依赖，用于获取决策所需的数据。

## 4. 配置项

-   **`repo_service_name`**: `im-repo` 服务的名称。
-   **`kafka.brokers`**: Kafka broker 的地址列表。
-   **`kafka.topics.upstream`**: 上行消息主题名称。
-   **`kafka.topics.downstream_prefix`**: 下行消息主题的前缀 (`gochat.messages.downstream`)。
-   **`kafka.topics.fanout_task`**: 扇出任务主题的名称。
-   **`log.level`**: 日志记录级别。
