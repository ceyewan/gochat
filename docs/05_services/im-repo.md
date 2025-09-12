# im-repo 数据仓储服务设计文档

`im-repo` 是 GoChat 系统的“数据管家”，是所有持久化数据（MySQL）和缓存数据（Redis）的唯一访问入口。它通过提供统一的 gRPC 接口，为上层服务（`im-logic`, `im-task`）屏蔽了所有底层数据存储的复杂性。

**设计目标**: 开发一个**高可靠、高性能、易于扩展和维护**的数据仓储服务，确保数据一致性，并为上层业务提供稳定、高效的数据支持。

## 1. 核心职责

1.  **数据访问的唯一入口**: 系统中任何其他微服务都**不允许**直接连接或操作 MySQL/Redis。所有数据操作都必须通过调用 `im-repo` 提供的 gRPC 接口来完成。
2.  **封装底层复杂性**: `im-repo` 对上层服务**完全透明**地封装了所有存储细节，包括但不限于：
    *   数据库表结构和索引设计。
    *   缓存的读写策略（Cache-Aside, Write-Through 等）。
    *   未来的数据库读写分离、分库分表（Sharding）。
    *   冷热数据分离存储。
3.  **提供原子化的业务操作**: 提供的 gRPC 接口是面向业务领域的、原子化的操作。例如，`CreateConversation` 接口会事务性地完成 `conversations` 表和 `conversation_members` 表的写入。
4.  **保证数据一致性**: 负责实现和维护缓存与数据库之间的数据一致性，上层服务无需关心此问题。

## 2. 数据模型与 gRPC 接口

`im-repo` 的设计严格遵循 `docs/06_data_models/01_db_schema.md` 中定义的数据模型。其暴露的 gRPC 接口旨在为上层提供高效、便捷的数据操作。

### 2.1 gRPC 服务定义 (`api/proto/im_repo/v1/`)

`im-repo` 对外暴露以下 gRPC 服务：

*   **`UserService`**: 管理用户数据的 CRUD。
*   **`ConversationService`**: 管理会话、成员关系和已读状态。
*   **`MessageService`**: 管理消息的持久化和查询。
*   **`FriendshipService`**: 管理好友关系和申请。
*   **`SequenceService`**: 提供原子序列号生成。
*   **`IdempotencyService`**: 提供幂等性检查。
*   **`OnlineStatusService`**: 提供对用户在线状态的只读访问。

### 2.2 核心 gRPC 接口示例

*   `user.proto`:
    *   `CreateUser(User)`
    *   `GetUser(user_id)`
    *   `BatchGetUsers(user_ids)`
    *   `GetUserByUsername(username)`
*   `conversation.proto`:
    *   `CreateConversation(Conversation, initial_member_ids)`
    *   `GetUserConversationsWithDetails(user_id, limit, offset)`: **核心优化接口**，一次性拉取会话列表所有信息。
    *   `AddMembers(conversation_id, user_ids)`
    *   `UpdateReadPointer(user_id, conversation_id, seq_id)`
*   `message.proto`:
    *   `SaveMessage(Message)`
    *   `GetMessages(conversation_id, cursor_seq_id, limit)`
*   `friendship.proto`:
    *   `CreateFriendshipRequest(requester_id, target_id, message)`
    *   `UpdateFriendshipStatus(request_id, status)`
    *   `GetFriendList(user_id)`
*   `sequence.proto`:
    *   `GetNextSeq(conversation_id)`

## 3. 核心数据操作流程

### 3.1 Cache-Aside (旁路缓存) 读取模式

这是 `im-repo` 读取数据（如用户信息、群成员）时遵循的核心策略。

```mermaid
sequenceDiagram
    participant Service as 上层服务
    participant Repo as im-repo
    participant Redis as Redis
    participant MySQL as MySQL

    Service->>+Repo: 1. gRPC 请求数据 (e.g., GetUser)
    Repo->>+Redis: 2. 查询缓存 (GET user:info:{id})
    alt 缓存命中
        Redis-->>-Repo: 3a. 返回缓存数据
    else 缓存未命中
        Redis-->>-Repo: 3b. 缓存未命中
        Repo->>+MySQL: 4b. 从数据库读取
        MySQL-->>-Repo: 5b. 返回数据
        Repo->>+Redis: 6b. 将数据写入缓存 (SETEX)
        Redis-->>-Repo: 7b. 确认写入
    end
    Repo-->>-Service: 4a/8b. 返回数据
```

### 3.2 “更新DB，删除Cache” 写入模式

当数据发生变更时（如用户修改昵称），`im-repo` 采用此策略保证数据一致性。

1.  **更新数据库**: 首先，将变更写入主存储 MySQL。
2.  **删除缓存**: 数据库操作成功后，**直接删除** Redis 中对应的缓存键（如 `DEL user:info:{user_id}`）。

这种策略可以有效地避免因并发写导致的缓存与数据库不一致问题。缓存的重建将交由下一次读请求（触发 Cache-Aside）来完成。

### 3.3 核心优化：会话列表拉取流程

此流程旨在彻底解决会话列表的 N+1 查询问题。

1.  **gRPC 请求**: `im-logic` 调用 `repo.GetUserConversationsWithDetails(user_id, ...)`。
2.  **查缓存**: `im-repo` 检查 Redis `GET user:conversations:full:{user_id}`。
3.  **缓存命中**: 直接返回缓存的完整列表数据。
4.  **缓存未命中**:
    a.  执行 `01_db_schema.md` 中定义的**核心优化SQL查询**。
    b.  在内存中将查询结果组装成 `ConversationOptimized` 结构体列表。
    c.  将整个列表序列化后写入 Redis `SET user:conversations:full:{user_id} "..." EX 3600`。
    d.  返回数据给 `im-logic`。

## 4. 数据扩展性设计

所有扩展性方案均在 `im-repo` 内部实现，对上层服务透明。

### 4.1 读写分离

*   **实现**: 通过 `im-infra/db` 模块配置一主多从。
*   **路由**: 写请求路由到主库，读请求路由到从库。
*   **一致性**: 提供在事务中或强制走主库的选项，解决“写后立即读”场景下的主从延迟问题。

### 4.2 垂直分库

*   **策略**: 按业务领域将表拆分到不同数据库实例（如用户库、会话库、消息库）。
*   **影响**: `im-repo` 内部需要处理跨库 `JOIN` 的问题，通过多次查询并在内存中组装数据来解决。

### 4.3 水平分片 (Sharding)

*   **分片键**: `messages` 表按 `conversation_id` 分片；`users` 表按 `user_id` 分片。
*   **实现**: `im-repo` 内部根据分片键计算目标分片，并将 SQL 路由到正确的分片执行。
*   **世界聊天室**: 对 `conversation_id` 固定的世界聊天室，采用按**时间**（如每月一张表）的二次分片策略，避免数据倾斜。

### 4.4 冷热数据分离

*   **策略**: `im-task` 定期将超过6个月的冷消息从 MySQL 迁移到成本更低的对象存储（如 S3/MinIO）或列式数据库（如 ClickHouse）。
*   **查询**: `im-repo` 的 `GetMessages` 接口会根据查询的时间范围，智能地决定是从 MySQL 查询热数据，还是从冷存储系统查询历史数据，或两者都查并合并结果。

通过以上设计，`im-repo` 构成了 GoChat 系统坚实的数据底座，为上层业务的快速迭代和未来海量数据的挑战做好了充分准备。
