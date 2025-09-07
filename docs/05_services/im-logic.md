# im-logic 核心业务逻辑服务设计

`im-logic` 是 GoChat 系统的“中央司令部”，负责处理所有核心的、实时的业务逻辑。它不直接与客户端交互，而是通过 gRPC 和 Kafka 与其他内部服务协作，是整个系统业务规则的最终执行者。

## 1. 核心职责

1.  **业务逻辑编排器 (Orchestrator)**: 以 **gRPC 服务**的形式，对外提供所有核心业务能力。它负责编排对下游服务（`im-repo`, `im-ai`, `im-recommend`）的调用，聚合结果，并返回给上游（`im-gateway`）。
2.  **消息流转中心**: 作为 **Kafka 消费者**，处理所有从 `im-gateway` 发来的上行消息；同时作为 **Kafka 生产者**，将处理后的消息分发给下游（`im-gateway` 或 `im-task`），或将需要索引的消息发送到 `indexing-topic`。
3.  **数据协调者**: 它本身**不直接操作数据库或缓存**。所有的数据读写请求，都必须通过调用 `im-repo` 模块的 gRPC 接口来完成。对于搜索请求，则直接调用 `Elasticsearch`。
4.  **决策者**: 它是业务规则的执行者。例如，根据群规模决定采用实时扩散还是异步任务，或根据消息内容决定是否需要进行 AI 处理。

**设计目标**: 开发一个**无状态、高可靠、逻辑清晰**的业务核心服务。

## 2. 架构与技术栈

`im-logic` 是一个纯粹的后端服务，其技术栈围绕 gRPC 服务和 Kafka 消息处理构建。

| 用途 | 库/工具 | 说明 |
| :--- | :--- | :--- |
| **基础库** | `im-infra` | 提供统一的基础能力，如ID生成、配置、日志、RPC等。 |
| **gRPC 框架** | `google.golang.org/grpc` | 用于实现对 `im-gateway` 暴露的 gRPC 服务。 |
| **Kafka 客户端** | `im-infra/mq` | 封装了生产者和消费者组，用于处理消息流。 |
| **gRPC 客户端** | `im-infra/rpc` | 用于调用 `im-repo`, `im-ai`, `im-recommend` 等下游服务的 gRPC 接口。 |
| **服务发现** | `im-infra/coord` | 通过 etcd 发现所有下游服务，并向 etcd 注册自身。 |
| **Elasticsearch 客户端** | `github.com/olivere/elastic` | 用于直接查询 Elasticsearch 以支持快速的全文搜索。 |

## 3. 核心流程与设计

### 3.1 消息处理生命周期

这是 `im-logic` 最核心的流程，它展示了一条消息如何被处理和分发。

```mermaid
sequenceDiagram
    participant Kafka as Kafka
    participant Logic as im-logic
    participant Repo as im-repo
    participant Task as im-task

    Logic->>-Kafka: 1. 消费上行消息<br>(im-upstream-topic)
    Logic->>+Repo: 2. 检查幂等性 (client_msg_id)
    Repo-->>-Logic: 3. 确认消息唯一性
    Logic->>+Repo: 4. 生成 message_id 和 seq_id
    Repo-->>-Logic: 5. 返回新生成的ID
    Logic->>+Repo: 6. **持久化优先**: 调用 SaveMessage RPC
    Repo-->>-Logic: 7. 消息持久化成功
    Logic->>Logic: 8. **执行分发决策**
    alt 单聊或中小群
        Logic->>+Repo: 9a. 查询接收者/群成员的在线网关
        Repo-->>-Logic: 10a. 返回网关ID列表
        Logic->>+Kafka: 11a. 生产下行消息到对应的网关Topic
    else 超大群
        Logic->>+Kafka: 9b. **生产异步任务**到 `im-task-topic`
    end
```

### 3.2 消息分发决策 (核心设计)

为了在实时性和系统负载之间取得平衡，`im-logic` 采用了混合分发模型：

-   **单聊**: 直接查询接收用户的在线状态（通过 `im-repo` 从 Redis 获取），并将消息生产到该用户所在 `im-gateway` 对应的 Kafka Topic (`im-downstream-topic-{gateway_id}`)。
-   **群聊 (混合模型)**:
    -   **中小群 (<=500人)**: `im-logic` 会实时处理。它会获取群内所有成员，批量查询他们的在线状态，然后将消息生产到所有在线成员对应的 `im-gateway` Topic。这个过程是实时的，保证了小群的低延迟体验。
    -   **超大群 (>500人)**: 为了避免 `im-logic` 因处理大量成员而阻塞，它会将消息扩散任务外包出去。它会构造一个轻量级的异步任务（包含 `group_id` 和 `message_id`），并将其生产到 `im-task-topic`。后续的扩散工作由 `im-task` 服务异步完成。

### 3.3 gRPC 服务与业务编排

`im-logic` 通过 gRPC 提供所有非消息类的业务功能，并作为核心的**业务编排者**。

#### 示例 1: 获取会话列表
1.  `im-gateway` 接收到 HTTP 请求。
2.  `im-gateway` 调用 `im-logic` 的 `GetConversations` gRPC 接口。
3.  `im-logic` 在 `GetConversations` 方法中，会执行复杂的业务逻辑：
    -   调用 `im-repo` 获取用户的所有会话ID。
    -   使用 `goroutine` **并发地**为每个会话调用 `im-repo` 获取最后一条消息、未读数、对端（好友/群组）的详细信息。
    -   聚合所有数据，按最后消息时间排序后，返回给 `im-gateway`。

#### 示例 2: 全文搜索
1.  `im-gateway` 接收到 `GET /search?q=...` 请求。
2.  `im-gateway` 调用 `im-logic` 的 `Search` gRPC 接口。
3.  `im-logic` 直接使用 Elasticsearch 客户端查询索引，获取结果。
4.  (可选) `im-logic` 调用 `im-repo` 补充搜索结果的详细信息（如用户头像）。
5.  将结果返回给 `im-gateway`。

#### 示例 3: 与 AI 对话
1.  `im-gateway` 接收到发往 AI 的消息。
2.  `im-gateway` 调用 `im-logic` 的 `SendMessageToAI` gRPC 接口。
3.  `im-logic` 调用 `im-ai` 服务的 `Chat` gRPC 接口，并将 AI 的响应返回。

这种设计将复杂的业务编排逻辑集中在 `im-logic`，而将纯粹的数据读写、AI 计算、推荐计算等能力下沉到各自的下游服务中。
