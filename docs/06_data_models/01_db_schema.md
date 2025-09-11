# GoChat 数据模型

本文档定义了 GoChat 系统的数据库架构。主数据库是 MySQL，采用统一的会话抽象设计。

## 1. 表：`users`

存储用户账户信息。

| 列         | 类型          | 约束      | 描述                               |
| -------------- | ------------- | ---------------- | ----------------------------------------- |
| `id`           | `BIGINT`      | `PRIMARY KEY`    | 用户的唯一标识符。           |
| `username`     | `VARCHAR(50)` | `UNIQUE, NOT NULL` | 用户的唯一登录名和显示名。 |
| `password_hash`| `VARCHAR(255)`| `NOT NULL`       | 用户的哈希密码。               |
| `nickname`     | `VARCHAR(100)`|                  | 用户昵称（显示名）。           |
| `avatar_url`   | `VARCHAR(255)`|                  | 用户头像图像的 URL。           |
| `is_guest`     | `BOOLEAN`     | `NOT NULL, DEFAULT false` | 如果用户是访客账户则为 `true`。 |
| `status`       | `INT`         | `NOT NULL, DEFAULT 0` | 用户状态 (0: 正常, 1: 禁用/待回收)。 |
| `created_at`   | `DATETIME`    | `NOT NULL`       | 创建用户的时间戳。   |
| `updated_at`   | `DATETIME`    | `NOT NULL`       | 最后更新的时间戳。             |

**索引**:
- `PRIMARY KEY (id)`
- `UNIQUE KEY uk_username (username)`
- `INDEX idx_status (status)`
- `INDEX idx_is_guest (is_guest)`

## 2. 表：`conversations`

存储所有类型会话的统一抽象，包括单聊、群聊和世界聊天室。

| 列          | 类型          | 约束   | 描述                               |
| --------------- | ------------- | ------------- | ----------------------------------------- |
| `id`            | `VARCHAR(64)` | `PRIMARY KEY` | 会话的唯一标识符。   |
| `type`          | `INT`         | `NOT NULL`    | 会话类型（1：单聊，2：群聊，3：世界聊天室）。 |
| `name`          | `VARCHAR(100)`|               | 会话名称（群名、频道名，单聊可为空）。 |
| `avatar_url`    | `VARCHAR(255)`|               | 会话头像的 URL。 |
| `description`   | `TEXT`        |               | 会话描述（群描述、频道介绍）。 |
| `owner_id`      | `BIGINT`      |               | 会话所有者的用户 ID（群主、频道主，单聊为空）。 |
| `member_count`  | `INT`         | `NOT NULL, DEFAULT 0` | 会话中的成员数量。 |
| `settings`      | `TEXT`        |               | 会话配置（JSON格式，存储类型特有配置）。 |
| `last_message_id`| `BIGINT`      |               | 会话中最后一条消息的 ID。 |
| `created_at`    | `DATETIME`    | `NOT NULL`    | 创建会话的时间戳。 |
| `updated_at`    | `DATETIME`    | `NOT NULL`    | 最后更新的时间戳。             |

**索引**:
- `PRIMARY KEY (id)`
- `INDEX idx_type (type)`
- `INDEX idx_owner (owner_id)`
- `INDEX idx_updated (updated_at)`
- `INDEX idx_type_updated (type, updated_at)`

**settings 字段示例**:
```json
// 群聊配置
{
  "group_settings": {
    "join_approval_required": true,
    "invite_enabled": true,
    "max_members": 500,
    "mute_all": false
  }
}

// 世界聊天室配置
{
  "world_settings": {
    "guest_allowed": true,
    "message_rate_limit": 10,
    "auto_join_guests": true
  }
}
```

## 3. 表：`conversation_members`

统一管理所有类型会话的成员关系，是实现高效会话列表拉取的核心表。

| 列 | 类型 | 约束 | 描述 |
| :--- | :--- | :--- | :--- |
| `id` | `BIGINT` | `PRIMARY KEY` | 记录的唯一标识符。 |
| `conversation_id` | `VARCHAR(64)` | `NOT NULL` | 会话的 ID。 |
| `user_id` | `BIGINT` | `NOT NULL` | 用户的 ID。 |
| `role` | `INT` | `NOT NULL, DEFAULT 1` | 用户在会话中的角色（1:成员, 2:管理员, 3:所有者）。 |
| `nickname` | `VARCHAR(100)` | | 用户在此会话中的昵称。 |
| `muted` | `BOOLEAN` | `NOT NULL, DEFAULT false` | 用户是否在此会话中被禁言。 |
| `joined_at` | `DATETIME` | `NOT NULL` | 用户加入会话的时间戳。 |
| `updated_at` | `DATETIME` | `NOT NULL` | 记录更新的时间戳。 |

**索引**:
- `PRIMARY KEY (id)`
- `UNIQUE KEY uk_user_conv (user_id, conversation_id)`: 用于快速查找用户的所有会话
- `INDEX idx_conversation (conversation_id)`: 用于快速查找一个会话的所有成员
- `INDEX idx_user_updated (user_id, updated_at)`: 用于用户会话列表的分页查询
- `INDEX idx_role (conversation_id, role)`: 用于按角色查询会话成员

## 4. 表：`messages`

存储在会话中发送的所有消息。

| 列          | 类型          | 约束                  | 描述                               |
| --------------- | ------------- | ---------------------------- | ----------------------------------------- |
| `id`            | `BIGINT`      | `PRIMARY KEY`                | 消息的唯一标识符。        |
| `conversation_id`| `VARCHAR(64)` | `NOT NULL` | 消息所属的会话的 ID。 |
| `sender_id`     | `BIGINT`      | `NOT NULL`                   | 发送者的用户 ID。                |
| `message_type`  | `INT`         | `NOT NULL, DEFAULT 1`        | 消息类型（1=文本, 2=图片, 3=文件, 4=系统消息）。  |
| `content`       | `TEXT`        | `NOT NULL`                   | 消息的内容。               |
| `seq_id`        | `BIGINT`      | `NOT NULL` | 会话内的顺序 ID，单调递增。 |
| `client_msg_id` | `VARCHAR(64)` |                              | 客户端消息 ID，用于幂等性处理。 |
| `deleted`       | `BOOLEAN`     | `NOT NULL, DEFAULT false`    | 如果消息已被软删除则为 `true`。 |
| `extra`         | `TEXT`        |                              | 消息的额外元数据（JSON）。    |
| `created_at`    | `DATETIME`    | `NOT NULL`                   | 创建消息的时间戳。 |
| `updated_at`    | `DATETIME`    | `NOT NULL`                   | 最后更新的时间戳。             |

**索引**:
- `PRIMARY KEY (id)`
- `UNIQUE KEY uk_conv_seq (conversation_id, seq_id)`: 确保会话内序列号唯一
- `INDEX idx_sender (sender_id)`: 用于查询用户发送的消息
- `INDEX idx_conv_created (conversation_id, created_at)`: 用于按时间查询消息
- `INDEX idx_client_msg (client_msg_id)`: 用于幂等性检查
- `INDEX idx_deleted (deleted)`: 用于过滤已删除消息

## 5. 表：`user_read_pointers`

存储每个用户在每个会话中最后读取的消息序列。

| 列          | 类型          | 约束                  | 描述                               |
| --------------- | ------------- | ---------------------------- | ----------------------------------------- |
| `id`            | `BIGINT`      | `PRIMARY KEY`                | 记录的唯一标识符。         |
| `user_id`       | `BIGINT`      | `NOT NULL` | 用户的 ID。                       |
| `conversation_id`| `VARCHAR(64)` | `NOT NULL` | 会话的 ID。               |
| `last_read_seq_id`| `BIGINT`      | `NOT NULL, DEFAULT 0`                   | 最后读取消息的序列 ID。 |
| `updated_at`    | `DATETIME`    | `NOT NULL`                   | 最后更新的时间戳。             |

**索引**:
- `PRIMARY KEY (id)`
- `UNIQUE KEY uk_user_conv (user_id, conversation_id)`: 确保用户在每个会话中只有一条已读记录
- `INDEX idx_conversation (conversation_id)`: 用于查询会话的已读状态

## 6. 表：`friendship_requests`

存储好友申请和好友关系，统一管理好友相关的所有状态。

| 列 | 类型 | 约束 | 描述 |
|---|---|---|---|
| `id` | `BIGINT` | `PRIMARY KEY` | 记录的唯一标识符。 |
| `requester_id` | `BIGINT` | `NOT NULL` | 发起申请的用户 ID。 |
| `target_id` | `BIGINT` | `NOT NULL` | 目标用户 ID。 |
| `status` | `INT` | `NOT NULL, DEFAULT 0` | 关系状态（0: 待处理, 1: 已接受, 2: 已拒绝, 3: 已拉黑）。 |
| `requester_remarks` | `VARCHAR(100)` | | 申请者对目标用户的备注名。 |
| `target_remarks` | `VARCHAR(100)` | | 目标用户对申请者的备注名。 |
| `message` | `TEXT` | | 好友申请消息。 |
| `created_at` | `DATETIME` | `NOT NULL` | 申请创建时间戳。 |
| `updated_at` | `DATETIME` | `NOT NULL` | 关系状态最后更新的时间戳。 |

**索引**:
- `PRIMARY KEY (id)`
- `UNIQUE KEY uk_requester_target (requester_id, target_id)`: 确保两用户间只有一条申请记录
- `INDEX idx_target_status (target_id, status)`: 用于查询用户收到的好友申请
- `INDEX idx_requester_status (requester_id, status)`: 用于查询用户发出的好友申请
- `INDEX idx_status_created (status, created_at)`: 用于按状态和时间查询

**注意**: 为了查询效率，好友关系确认后会在此表中创建双向记录（A->B 和 B->A），以便快速查询双方的好友列表。

## 7. 持久化与数据拉取策略

### 7.1 消息持久化流程

根据系统架构，消息的持久化遵循以下流程：

1.  **生产者**: `im-logic` 服务在完成业务逻辑后，将消息生产到 Kafka 的 `gochat.messages.persist` 主题。
2.  **消费者**: `im-task` 服务作为持久化任务的唯一消费者，订阅此主题。
3.  **执行者**: `im-task` 收到消息后，调用 `im-repo` 服务的 `SaveMessage` gRPC 接口。
4.  **存储**: `im-repo` 服务负责将消息数据写入 `MySQL` 数据库，并更新 `Redis` 中的相关缓存。

### 7.2 会话列表拉取

-   **触发**: 客户端在登录或重新连接后，调用 `GET /conversations` 接口。
-   **逻辑**:
    1.  `im-logic` 调用 `im-repo` 的 `GetUserConversations` 接口。
    2.  `im-repo` 首先尝试从 `Redis` 缓存中获取该用户的会话列表。
    3.  如果缓存未命中，从 `conversation_members` 表中查出用户所属的所有会话，再聚合其他表的信息。
    4.  查询结果被缓存到 `Redis` 中并返回。
-   **核心查询**:
    ```sql
    -- 高效的会话列表查询
    SELECT cm.conversation_id, c.type, c.name, c.avatar_url, c.updated_at,
           cm.role, cm.nickname, cm.joined_at
    FROM conversation_members cm
    JOIN conversations c ON cm.conversation_id = c.id
    WHERE cm.user_id = ? 
    ORDER BY c.updated_at DESC 
    LIMIT ? OFFSET ?;
    ```

### 7.3 历史消息拉取

-   **分页策略**: 采用基于游标（seq_id）的分页策略，以获得更好的性能和一致性。
-   **接口**: `GET /conversations/{conversationId}/messages?cursor={last_seq_id}&limit=50`
-   **核心查询**:
    ```sql
    SELECT id, sender_id, message_type, content, seq_id, created_at, extra
    FROM messages 
    WHERE conversation_id = ? AND seq_id < ? AND deleted = false
    ORDER BY seq_id DESC 
    LIMIT ?;
    ```

### 7.4 未读数计算

-   **实时计算**: 基于用户的已读指针和会话的最新消息序列号。
-   **核心逻辑**:
    ```sql
    -- 计算单个会话的未读数
    SELECT COUNT(*) as unread_count
    FROM messages m
    LEFT JOIN user_read_pointers urp ON (urp.user_id = ? AND urp.conversation_id = m.conversation_id)
    WHERE m.conversation_id = ? 
      AND m.seq_id > COALESCE(urp.last_read_seq_id, 0)
      AND m.deleted = false;
    ```

## 8. 数据库扩展性策略

### 8.1 `messages` 表水平分片

随着消息量的增长，`messages` 表需要进行水平分片以保证性能：

-   **分片键**: `conversation_id`
-   **分片策略**: 使用一致性哈希算法，确保同一会话的所有消息在同一分片
-   **优势**: 历史消息查询可以精确路由到单个分片，避免跨分片查询

### 8.2 缓存策略

#### Redis 缓存层次：
1. **用户会话列表**: `user:conversations:{user_id}` (TTL: 1小时)
2. **会话信息**: `conversation:{conversation_id}` (TTL: 30分钟)
3. **用户在线状态**: `user:online:{user_id}` (TTL: 5分钟)
4. **热点消息**: `conversation:messages:{conversation_id}:latest` (TTL: 10分钟)

### 8.3 索引优化

#### 复合索引设计原则：
- **最左前缀匹配**: 将最常用的查询条件放在最左侧
- **覆盖索引**: 尽可能让索引包含查询需要的所有字段
- **分区索引**: 对于大表考虑按时间或类型分区

## 9. 数据一致性保证

### 9.1 分布式事务处理
- **Saga 模式**: 用于跨服务的数据一致性保证
- **最终一致性**: 通过事件发布/订阅模式实现数据最终一致

### 9.2 并发控制
- **乐观锁**: 使用版本号控制并发更新
- **分布式锁**: 使用 Redis 实现关键操作的互斥

### 9.3 数据备份与恢复
- **主从复制**: MySQL 主从架构保证数据可用性
- **定期备份**: 每日全量备份，实时增量备份
- **跨地域容灾**: 考虑多地域部署提高可用性