## **GoChat - `im-repo` 模块开发设计与计划**

**致实习生同学：**

你好！你即将负责 `im-repo` 模块的开发。这个模块是 GoChat 系统的“数据管家”，是所有持久化数据和缓存数据的唯一看门人。其他所有服务都依赖 `im-repo` 来保证数据的准确和高效存取。

这个任务将深度锻炼你的数据库设计、缓存策略应用以及 gRPC 服务开发能力。它非常关键，也极具价值。让我们开始构建这座坚实的数据基石。

### **1. 模块职责与目标 (The "What")**

`im-repo` 的职责非常纯粹和聚焦：

1.  **数据访问的唯一入口**: 系统中**任何模块都不允许直接连接 MySQL 或 Redis**。所有数据操作都必须通过调用 `im-repo` 提供的 gRPC 接口来完成。
2.  **封装底层复杂性**: `im-repo` 对上层服务（`im-logic`, `im-task`）**屏蔽**了所有底层存储的细节。这包括：
    *   数据库表结构。
    *   缓存的读写策略（Cache-Aside）。
    *   未来可能引入的数据库读写分离、分库分表。
3.  **提供原子化的数据操作**: 提供的 gRPC 接口应该是业务层面的、原子化的操作。例如，`SaveMessage` 接口应该同时完成写入数据库和更新缓存两个动作。

**我们的目标**: 开发一个**高可靠、高性能、易维护**的数据仓储服务。

### **2. 核心设计：数据模型 (The "Blueprint")**

这是 `im-repo` 最核心的部分。我们将详细设计 MySQL 表结构和 Redis 缓存结构。

#### **2.1 数据库表设计 (MySQL)**

| 表名 | 字段名 | 类型 | 约束/索引 | 描述 |
| :--- | :--- | :--- | :--- | :--- |
| **users** | `id` | `BIGINT UNSIGNED` | **PK**, NOT NULL | 用户ID (分布式ID生成) |
| | `username` | `VARCHAR(50)` | **UNIQUE**, NOT NULL | 用户名 |
| | `password_hash` | `VARCHAR(255)` | NOT NULL | bcrypt 哈希后的密码 |
| | `nickname` | `VARCHAR(50)` | DEFAULT '' | 昵称 |
| | `avatar_url` | `VARCHAR(255)` | DEFAULT '' | 头像URL |
| | `created_at` | `TIMESTAMP` | NOT NULL | 创建时间 |
| **groups** | `id` | `BIGINT UNSIGNED` | **PK**, NOT NULL | 群组ID (分布式ID生成) |
| | `name` | `VARCHAR(50)` | NOT NULL | 群名称 |
| | `owner_id` | `BIGINT UNSIGNED` | **KEY**, NOT NULL | 群主用户ID |
| | `member_count`| `INT UNSIGNED` | NOT NULL, DEFAULT 0 | 成员数量 (冗余字段，用于快速查询) |
| | `created_at` | `TIMESTAMP` | NOT NULL | 创建时间 |
| **group_members** | `id` | `BIGINT UNSIGNED` | **PK**, NOT NULL | 记录ID (分布式ID生成) |
| | `group_id` | `BIGINT UNSIGNED` | **UNIQUE(group_id, user_id)** | 群组ID |
| | `user_id` | `BIGINT UNSIGNED` | **KEY(user_id)** | 用户ID |
| | `role` | `TINYINT` | NOT NULL, DEFAULT 1 | 角色 (1:成员, 2:管理员, 3:群主) |
| | `joined_at` | `TIMESTAMP` | NOT NULL | 加入时间 |
| **messages** | `id` | `BIGINT UNSIGNED` | **PK**, NOT NULL | 消息ID (分布式ID生成) |
| | `conversation_id` | `VARCHAR(64)` | **UNIQUE(conv_id, seq_id)**, **KEY(conv_id, created_at)** | 会话ID |
| | `sender_id` | `BIGINT UNSIGNED` | NOT NULL | 发送者ID |
| | `message_type`| `TINYINT` | NOT NULL, DEFAULT 1 | 消息类型 (1:文本, 2:图片...) |
| | `content` | `TEXT` | NOT NULL | 消息内容 (或资源URL) |
| | `seq_id` | `BIGINT UNSIGNED` | NOT NULL | 会话内单调递增序列号 |
| | `created_at` | `TIMESTAMP(3)`| NOT NULL | 创建时间 (精确到毫秒) |

#### **2.2 缓存设计 (Redis)**

缓存的设计目标是减少对 MySQL 的直接访问，提升性能。

| 用途 | Key 格式 | Value 类型 | 描述与作用 |
| :--- | :--- | :--- | :--- |
| **用户信息** | `user_info:{user_id}` | **HASH** | 缓存 `users` 表的行数据。字段如 `nickname`, `avatar_url`。 |
| **用户在线状态** | `user_session:{user_id}` | **HASH** | 由 `im-gateway` 写入，`im-logic` 读取。字段: `gateway_id`。 |
| **会话序列号** | `conv_seq:{conversation_id}` | **STRING** | 使用 `INCR` 命令为 `im-logic` 原子地生成 `seq_id`。 |
| **群组成员列表** | `group_members:{group_id}` | **SET** | 缓存一个群的所有 `user_id`。用于快速获取成员列表和判断成员关系。 |
| **消息去重** | `msg_dedup:{client_msg_id}`| **STRING** | 由 `im-logic` 写入，`SETEX` 60秒。用于实现消息发送的幂等性。 |
| **热点消息缓存** | `hot_messages:{conv_id}` | **ZSET** | 缓存每个会话最近的N条(如300条)消息。`Score` 是 `seq_id`，`Member` 是消息体的 JSON 字符串。用于快速加载聊天界面首页，避免直接查库。 |
| **会话未读数** | `unread:{conv_id}:{user_id}` | **STRING** | 使用 `INCR` 命令记录用户的未读消息数。 |

### **3. 技术栈与依赖 (The "Tools")**

| 用途 | 库/工具 | 学习重点 |
| :--- | :--- | :--- |
| **gRPC 框架** | `google.golang.org/grpc` | 实现 gRPC Server |
| **数据库 ORM** | `gorm.io/gorm` | 连接数据库、CRUD 操作、事务处理 |
| **Redis 客户端**| `github.com/go-redis/redis/v8` | 连接 Redis（包括集群）、常用命令操作 |
| **配置管理** | `github.com/spf13/viper` | 加载数据库和 Redis 的连接信息 |
| **日志** | `go.uber.org/zap` | 结构化日志 |
| **服务发现** | `go.etcd.io/etcd/client/v3` | 向 etcd 注册本服务 |

### **4. 项目结构 (The "Blueprint")**

```
im-repo/
├── cmd/
│   └── main.go              # 程序入口：初始化、启动 gRPC 服务
├── internal/
│   ├── config/              # 配置定义与加载
│   ├── data/
│   │   ├── mysql.go         # GORM 初始化与封装
│   │   └── redis.go         # Redis Client 初始化与封装
│   ├── biz/                 # 业务逻辑层 (Business Logic)
│   │   ├── user_repo.go     # User 相关的业务逻辑实现
│   │   └── message_repo.go  # Message 相关的业务逻辑实现
│   └── service/
│       └── grpc_service.go  # gRPC 服务的实现 (将 biz 层封装成 gRPC)
├── pkg/
│   └── proto/                 # 存放 im-repo 对外提供的 .proto 文件
└── go.mod
```**设计说明**: 我们引入了 `biz` (业务逻辑) 和 `data` (数据访问) 两层。`data` 层只负责与数据库和 Redis 交互，`biz` 层负责实现具体的业务逻辑（如 Cache-Aside），`service` 层则将 `biz` 层的能力暴露为 gRPC 接口。这种分层更清晰。

### **5. 开发计划：分阶段进行 (The "Plan")**

---

#### **阶段一：搭建骨架与数据库连接 (预计2天)**

**目标**: 让 `im-repo` 能够成功连接到 MySQL 和 Redis，并跑起一个空的 gRPC 服务。

**任务分解**:
1.  **初始化项目**: `go mod init`，引入 `grpc-go`, `gorm`, `go-redis`, `viper`, `zap`。
2.  **定义 `.proto` 接口 (`pkg/proto`)**:
    *   创建 `user_repo.proto`。定义 `UserRepo` 服务和 `GetUser` RPC。
    *   使用 `protoc` 生成 Go 代码。
3.  **配置管理 (`internal/config`)**:
    *   定义 `Config` 结构体，包含 MySQL 和 Redis 的 DSN(连接字符串)、gRPC 端口等。
    *   实现 `LoadConfig()` 函数。
4.  **数据访问层 (`internal/data`)**:
    *   `mysql.go`: 实现 `NewMySQLClient()` 函数，使用 `gorm.Open` 初始化数据库连接池。
    *   `redis.go`: 实现 `NewRedisClient()` 函数，使用 `redis.NewClient` (或 `NewClusterClient`) 初始化 Redis 连接。
5.  **程序入口 (`cmd/main.go`)**:
    *   加载配置，初始化日志。
    *   调用 `NewMySQLClient` 和 `NewRedisClient`，并处理连接错误。
    *   初始化并启动 gRPC 服务器（暂时不注册任何服务）。

**验收标准**:
*   启动 `im-repo` 服务。
*   日志中没有出现数据库或 Redis 的连接错误。
*   服务能够正常监听在指定的 gRPC 端口上。

---

#### **阶段二：实现用户相关的 Repo (预计2天)**

**目标**: 完整实现 `UserRepo` 服务，包括数据库和缓存的交互。

**任务分解**:
1.  **数据模型 (`internal/data`)**:
    *   定义与 `users` 表对应的 GORM 模型 `User` 结构体。
2.  **业务逻辑 (`internal/biz/user_repo.go`)**:
    *   创建一个 `UserRepo` 结构体，持有 `*gorm.DB` 和 `*redis.Client`。
    *   实现 `GetUserByID(id int64)` 方法，并**严格遵循 Cache-Aside 模式**：
        1.  先从 Redis (`user_info:{id}`) 查询用户信息。
        2.  **缓存命中**: 解析 HASH 数据并返回。
        3.  **缓存未命中**: 从 MySQL 查询 `users` 表。
        4.  如果数据库中查到了数据，将其写入 Redis 缓存 (`HSET user_info:{id} ...`)，并设置一个合理的过期时间（如24小时）。
        5.  返回从数据库中查到的数据。
    *   实现 `CreateUser(...)` 方法：
        1.  将用户信息写入 MySQL `users` 表。
        2.  **注意**: 这里不需要写缓存，让缓存自然失效或等待下次查询时填充（Lazy Loading）。
3.  **gRPC 服务层 (`internal/service/grpc_service.go`)**:
    *   创建一个 `RepoService` 结构体，持有 `biz.UserRepo`。
    *   为 `RepoService` 实现 `GetUser` gRPC 方法。
    *   在方法内部，调用 `biz.UserRepo.GetUserByID` 并处理返回结果。
4.  **注册服务**: 在 `main.go` 中，将 `RepoService` 实例注册到 gRPC 服务器。

**验收标准**:
*   使用 `grpcurl` 或其他 gRPC 客户端。
*   第一次调用 `GetUser`，日志显示从 MySQL 查询。
*   第二次调用 `GetUser`，日志显示从 Redis 查询（或没有数据库查询日志）。
*   调用 `CreateUser` 后，能在数据库中看到新用户。

---

#### **阶段三：实现消息与群组相关的 Repo (预计3天)**

**目标**: 完整实现 `MessageRepo` 和 `GroupRepo` 服务。

**任务分解**:
1.  **定义 `.proto` 接口**: 为 `MessageRepo` 和 `GroupRepo` 定义 gRPC 接口，如 `SaveMessage`, `GetMessages`, `GetGroupMembers` 等。
2.  **实现 `SaveMessage` (`internal/biz/message_repo.go`)**:
    *   **核心逻辑**:
        1.  将消息写入 MySQL `messages` 表。
        2.  将消息序列化为 JSON 字符串。
        3.  将其写入 Redis 的 ZSET: `ZADD hot_messages:{conv_id} {seq_id} {message_json}`。
        4.  对 ZSET 进行修剪: `ZREMRANGEBYRANK hot_messages:{conv_id} 0 -301` (保留最新的300条)。
    *   **事务性**: 写入 MySQL 和 Redis 的操作，虽然不是严格的分布式事务，但应保证顺序。通常先写主存储 MySQL。
3.  **实现 `GetMessages` (`internal/biz/message_repo.go`)**:
    *   这是一个纯数据库操作，根据分页参数从 `messages` 表倒序查询。
4.  **实现 `GetGroupMembers` (`internal/biz/group_repo.go`)**:
    *   同样遵循 Cache-Aside 模式。
    *   先从 Redis `SMEMBERS group_members:{group_id}` 获取。
    *   未命中则从 MySQL `group_members` 表查询，并写回 Redis (`SADD group_members:{group_id} ...`)。
5.  **实现 `service` 层**: 将所有 `biz` 层的方法封装成 gRPC 接口。

**验收标准**:
*   所有 gRPC 接口都能通过 `grpcurl` 成功调用。
*   调用 `SaveMessage` 后，能在 MySQL 和 Redis ZSET 中都看到相应数据。
*   多次调用 `GetGroupMembers`，能验证缓存逻辑是否生效。

---

### **6. 总结与最佳实践**

*   **错误处理**: `im-repo` 的每个 gRPC 方法都必须仔细处理数据库和 Redis 可能返回的错误，并将其转换为标准的 gRPC 状态码（如 `codes.NotFound`, `codes.Internal`）。
*   **日志**: 在关键的数据操作前后打印详细日志，包括输入的参数和操作结果，便于问题排查。
*   **测试**: 编写单元测试，特别是对 `biz` 层的缓存逻辑进行测试，是保证质量的最好方法。

`im-repo` 是一个没有复杂业务逻辑，但对稳定性和严谨性要求极高的模块。请务必细心、耐心，严格按照设计完成开发。祝你成功！