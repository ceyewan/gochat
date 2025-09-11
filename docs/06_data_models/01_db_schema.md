# GoChat 数据模型

本文档定义了 GoChat 系统的数据库架构。主数据库是 MySQL。

## 1. 表：`users`

存储用户账户信息。

| 列         | 类型          | 约束      | 描述                               |
| -------------- | ------------- | ---------------- | ----------------------------------------- |
| `id`           | `BIGINT`      | `PRIMARY KEY`    | 用户的唯一标识符。           |
| `username`     | `VARCHAR(50)` | `UNIQUE, NOT NULL` | 用户的唯一登录名和显示名。 |
| `password_hash`| `VARCHAR(255)`| `NOT NULL`       | 用户的哈希密码。               |
| `avatar_url`   | `VARCHAR(255)`|                  | 用户头像图像的 URL。           |
| `is_guest`     | `BOOLEAN`     | `NOT NULL, DEFAULT false` | 如果用户是访客账户则为 `true`。 |
| `status`       | `INT`         | `NOT NULL, DEFAULT 0` | 用户状态 (0: 正常, 1: 禁用/待回收)。 |
| `created_at`   | `DATETIME`    | `NOT NULL`       | 创建用户的时间戳。   |
| `updated_at`   | `DATETIME`    | `NOT NULL`       | 最后更新的时间戳。             |

## 2. 表：`conversations`

存储有关每个会话的信息，可以是单个聊天或群组聊天。

| 列          | 类型          | 约束   | 描述                               |
| --------------- | ------------- | ------------- | ----------------------------------------- |
| `id`            | `VARCHAR(64)` | `PRIMARY KEY` | 会话的唯一标识符。   |
| `type`          | `INT`         | `NOT NULL`    | 会话的类型（1：单个，2：群组）。 |
| `last_message_id`| `BIGINT`      |               | 会话中最后一条消息的 ID。 |
| `created_at`    | `DATETIME`    | `NOT NULL`    | 创建会话的时间戳。 |
| `updated_at`    | `DATETIME`    | `NOT NULL`    | 最后更新的时间戳。             |

## 3. 表：`groups`

存储有关群组聊天的信息。

| 列        | 类型          | 约束      | 描述                               |
| ------------- | ------------- | ---------------- | ----------------------------------------- |
| `id`          | `BIGINT`      | `PRIMARY KEY`    | 群组的唯一标识符。          |
| `name`        | `VARCHAR(50)` | `NOT NULL`       | 群组的名称。                    |
| `owner_id`    | `BIGINT`      | `NOT NULL`       | 群组所有者的用户 ID。           |
| `member_count`| `INT`         | `NOT NULL, DEFAULT 0` | 群组中的成员数量。       |
| `avatar_url`  | `VARCHAR(255)`|                  | 群组头像图像的 URL。          |
| `description` | `TEXT`        |                  | 群组的描述。               |
| `created_at`  | `DATETIME`    | `NOT NULL`       | 创建群组的时间戳。  |
| `updated_at`  | `DATETIME`    | `NOT NULL`       | 最后更新的时间戳。             |

## 4. 表：`conversation_members` (新)

核心表，用于记录用户与所有类型会话（单聊、群聊）的成员关系。这张表是实现高效会话列表拉取的基石。原 `group_members` 表的功能已被此表完全取代。

| 列 | 类型 | 约束 | 描述 |
| :--- | :--- | :--- | :--- |
| `id` | `BIGINT` | `PRIMARY KEY` | 记录的唯一标识符。 |
| `conversation_id` | `VARCHAR(64)` | `UNIQUE(user_id, conversation_id)` | 会话的 ID。 |
| `user_id` | `BIGINT` | `UNIQUE(user_id, conversation_id)` | 用户的 ID。 |
| `role` | `INT` | `NOT NULL, DEFAULT 1` | 用户在会话中的角色（1:成员, 2:管理员, 3:所有者）。 |
| `created_at` | `DATETIME` | `NOT NULL` | 用户加入会话的时间戳。 |
| `updated_at` | `DATETIME` | `NOT NULL` | 记录更新的时间戳。 |

**索引**:
- `UNIQUE(user_id, conversation_id)`: 用于快速查找用户的所有会话。
- `INDEX(conversation_id)`: 用于快速查找一个会话的所有成员。

## 5. 表：`messages`

存储在会话中发送的所有消息。

| 列          | 类型          | 约束                  | 描述                               |
| --------------- | ------------- | ---------------------------- | ----------------------------------------- |
| `id`            | `BIGINT`      | `PRIMARY KEY`                | 消息的唯一标识符。        |
| `conversation_id`| `VARCHAR(64)` | `UNIQUE(conversation_id, seq_id)` | 消息所属的会话的 ID。 |
| `sender_id`     | `BIGINT`      | `NOT NULL`                   | 发送者的用户 ID。                |
| `message_type`  | `INT`         | `NOT NULL, DEFAULT 1`        | 消息的类型（例如，文本、图像）。  |
| `content`       | `TEXT`        | `NOT NULL`                   | 消息的内容。               |
| `seq_id`        | `BIGINT`      | `UNIQUE(conversation_id, seq_id)` | 会话内的顺序 ID。 |
| `deleted`       | `BOOLEAN`     | `NOT NULL, DEFAULT false`    | 如果消息已被软删除则为 `true`。 |
| `extra`         | `TEXT`        |                              | 消息的额外元数据（JSON）。    |
| `created_at`    | `DATETIME`    | `NOT NULL`                   | 创建消息的时间戳。 |
| `updated_at`    | `DATETIME`    | `NOT NULL`                   | 最后更新的时间戳。             |

## 6. 表：`user_read_pointers`

存储每个用户在每个会话中最后读取的消息序列。

| 列          | 类型          | 约束                  | 描述                               |
| --------------- | ------------- | ---------------------------- | ----------------------------------------- |
| `id`            | `BIGINT`      | `PRIMARY KEY`                | 记录的唯一标识符。         |
| `user_id`       | `BIGINT`      | `UNIQUE(user_id, conversation_id)` | 用户的 ID。                       |
| `conversation_id`| `VARCHAR(64)` | `UNIQUE(user_id, conversation_id)` | 会话的 ID。               |
| `last_read_seq_id`| `BIGINT`      | `NOT NULL`                   | 最后读取消息的序列 ID。 |
| `updated_at`    | `DATETIME`    | `NOT NULL`                   | 最后更新的时间戳。             |

## 7. 表：`friends`

存储用户之间的好友关系。为了确保关系的唯一性和查询效率，每对好友关系会存储两条记录（A->B 和 B->A）。

| 列 | 类型 | 约束 | 描述 |
|---|---|---|---|
| `id` | `BIGINT` | `PRIMARY KEY` | 记录的唯一标识符。 |
| `user_id` | `BIGINT` | `UNIQUE(user_id, friend_id)` | 用户的 ID。 |
| `friend_id` | `BIGINT` | `UNIQUE(user_id, friend_id)` | 好友的 ID。 |
| `status` | `INT` | `NOT NULL, DEFAULT 0` | 关系状态（0: 待处理, 1: 已接受, 2: 已拒绝, 3: 已拉黑）。 |
| `remarks` | `VARCHAR(100)` | | 用户对好友的备注名。 |
| `created_at` | `DATETIME` | `NOT NULL` | 关系创建或发起的时间戳。 |
| `updated_at` | `DATETIME` | `NOT NULL` | 关系状态最后更新的时间戳。 |

## 8. 持久化与数据拉取策略

### 8.1 消息持久化流程

根据系统架构，消息的持久化遵循以下流程：

1.  **生产者**: `im-logic` 服务在完成业务逻辑后，将下行消息生产到 Kafka 的 `gochat.downstream.topic` 主题。
2.  **消费者**: `im-task` 服务作为持久化任务的唯一消费者，订阅 `gochat.downstream.topic`。
3.  **执行者**: `im-task` 收到消息后，调用 `im-repo` 服务的 `SaveMessage` gRPC 接口。
4.  **存储**: `im-repo` 服务负责将消息数据写入 `MySQL` 数据库，并可能会更新 `Redis` 中的相关缓存（如会话的最后一条消息）。

### 8.2 会话列表拉取

-   **触发**: 客户端在登录或重新连接后，调用 `GET /conversations` 接口。
-   **逻辑**:
    1.  `im-logic` 调用 `im-repo` 的 `GetUserConversations` 接口。
    2.  `im-repo` 首先尝试从 `Redis` 缓存中获取该用户的会话列表。
    3.  如果缓存未命中，`im-repo` 从 `MySQL` 的 `conversation_members` 表中查出用户所属的所有 `conversation_id`，再聚合 `conversations` 和 `user_read_pointers` 表的信息，计算每个会话的未读数 (`unread_count`) 和最后一条消息 (`last_message`)。
    4.  查询结果被缓存到 `Redis` 中并返回给 `im-logic`。
-   **数据模型**: 返回的会话对象应包含 `id`, `type`, `name`, `avatar`, `last_message`, `unread_count`, `updated_at` 等关键信息。

### 8.3 历史消息拉取

-   **触发**: 用户在会话内向上滚动时，客户端调用 `GET /conversations/{conversationId}/messages` 接口。
-   **分页策略**: 采用基于游标（Cursor-based）的分页策略，以获得更好的性能和一致性。
    -   **接口**: `GET /conversations/{conversationId}/messages?cursor={last_message_seq_id}&limit=50`
    -   `cursor`: 上一页返回的最后一条消息的 `seq_id`。首次请求时为空。
    -   `limit`: 每次拉取的消息数量，默认为 50。
-   **逻辑**:
    1.  `im-logic` 调用 `im-repo` 的 `GetMessages` 接口，并传递 `conversation_id`, `cursor`, `limit`。
    2.  `im-repo` 执行类似 `SELECT * FROM messages WHERE conversation_id = ? AND seq_id < ? ORDER BY seq_id DESC LIMIT ?` 的查询。
    3.  返回消息列表和下一页的 `cursor`。

## 9. 数据库扩展性策略 (新增)

### 9.1 `messages` 表水平分片

随着消息量的增长，`messages` 表将成为系统最大的表，必须进行水平分片（Sharding）以保证性能和可扩展性。

-   **分片键 (Sharding Key)**: `conversation_id`。
-   **分片策略**:
    -   采用 **Range** 或 **Hash** 模式对 `conversation_id` 进行分片。
    -   **原理**: 确保同一个会话的所有消息都落在同一个物理分片（数据库实例或表）上。
    -   **优势**:
        1.  **查询高效**: 拉取历史消息等绝大多数查询都基于 `conversation_id`，可以被精确路由到单个分片，避免了耗费资源的跨分片查询。
        2.  **数据隔离**: 不同会话的数据天然隔离，便于管理和迁移。
-   **实现方案**: 建议引入成熟的数据库中间件（如 [ShardingSphere](https://shardingsphere.apache.org/), [Vitess](https://vitess.io/)）来对应用层屏蔽分片的复杂性。

### 9.2 全局搜索与数据异构

基于 `conversation_id` 的分片策略使得跨会话的全局搜索变得困难。

-   **问题**: 如果需要实现“搜索某个用户在所有会话中发送的消息”或“全局搜索包含某个关键词的消息”等功能，直接查询数据库将需要扫描所有分片，效率极低。
-   **解决方案**: **数据异构**，将消息数据同步到专门的搜索引擎。
    1.  **技术选型**: [Elasticsearch](https://www.elastic.co/elasticsearch/)。
    2.  **同步流程**: `im-task` 服务在消费到持久化消息的事件后，除了调用 `im-repo` 将消息写入 MySQL，还应**异步地**将消息数据写入 Elasticsearch。
    3.  **查询实现**: 全局搜索功能将直接查询 Elasticsearch，而不是关系型数据库。