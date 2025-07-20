## **GoChat 分布式即时通讯系统 - 系统详细设计文档 (V3.0)**

**版本**: 3.0  
**日期**: 2025-07-05  
**作者**: Gemini (AI System Architect)

### 1. 引言

#### 1.1 文档目的
本文档旨在为 GoChat 后端系统的开发、测试和运维提供全面、精确的技术指导。它详细描述了系统的模块划分、核心业务流程、接口协议、数据模型、关键算法及非功能性设计，是连接架构蓝图与代码实现的桥梁。

#### 1.2 范围
本文档覆盖 `im-gateway`, `im-logic`, `im-task`, `im-repo` 四大核心服务及 `im-infra` 基础库的详细设计。

#### 1.3 读者对象
本文档的主要读者为 GoChat 项目的后端开发工程师、测试工程师、运维工程师及项目经理。

### 2. 核心实体与ID设计

为保证系统的全局唯一性和可扩展性，ID设计是基础。

#### 2.1 ID生成策略
*   **分布式ID生成**: 采用Snowflake算法，集成在`im-infra`基础库中，各服务通过调用统一接口生成ID。
*   **ID组成**: 64位 = 1位符号位 + 41位时间戳 + 10位机器ID + 12位序列号。
*   **机器ID分配**: 通过etcd自动分配和管理，避免冲突。

#### 2.2 实体ID定义
*   **`user_id`**: `BIGINT UNSIGNED`。由 `im-infra.IDGen` 在用户注册时生成。
*   **`group_id`**: `BIGINT UNSIGNED`。由 `im-infra.IDGen` 在群组创建时生成。
*   **`message_id`**: `BIGINT UNSIGNED`。由 `im-infra.IDGen` 在 `im-logic` 消息处理时生成。
*   **`conversation_id`**: `VARCHAR(64)`。
    *   **单聊**: `"single_" + md5(min(user_id1, user_id2) + "_" + max(user_id1, user_id2))[0:16]`
    *   **群聊**: `"group_" + group_id`
    *   **世界聊天室**: `"world"`
*   **`seq_id`**: `BIGINT UNSIGNED`。会话内单调递增序列，由 `im-logic` 通过 Redis `INCR conv_seq:{conversation_id}` 命令原子生成。
*   **`client_msg_id`**: `VARCHAR(64)`。由客户端生成（`user_id + "_" + timestamp_ms + "_" + random_int`），用于实现**发送幂等性**。

### 3. 核心业务流程详解

#### 3.1 用户生命周期与数据同步

##### 3.1.1 注册/登录流程
1.  **Client -> Gateway (HTTP)**: `POST /api/auth/login` 或 `POST /api/auth/register`。
2.  **Gateway -> Logic (gRPC)**: 调用 `logic.AuthService` 的相应 RPC。
3.  **Logic -> Repo (gRPC)**:
    *   **注册**: 调用 `repo.UserRepo.CreateUser`。`im-repo` 需处理 `username` 的唯一性约束冲突。
    *   **登录**: 调用 `repo.UserRepo.GetUserByUsername`。
4.  **Logic (处理)**:
    *   密码使用 `bcrypt` 进行哈希存储和比较。
    *   登录成功后，生成 JWT，Payload 包含 `{"user_id": "...", "exp": ...}`。
5.  **Logic -> Gateway -> Client**: 返回 JWT 及用户信息。

##### 3.1.2 WebSocket 连接与会话建立
1.  **Client**: 使用获取的 JWT 发起 `ws://.../ws?token={jwt}` 连接。
2.  **Gateway**:
    *   拦截连接请求，解析并验证 JWT。失败则拒绝连接。
    *   成功后，向 Redis 写入在线状态: `HSET user_session:{user_id} gateway_id {current_gateway_id} login_at {timestamp}`。
    *   在本地内存中建立 `user_id -> websocket_connection` 的映射，用于快速消息推送。
    *   启动该连接的 `readPump` 和 `writePump` goroutine。

##### 3.1.3 首次登录/重连后的数据拉取
1.  **Client**: 在 WebSocket 连接成功后，立即发起 `GET /api/conversations` 请求。
2.  **Gateway -> Logic (gRPC)**: 调用 `logic.ConversationService.GetConversations`。
3.  **Logic (实现)**:
    *   调用 `repo.ConversationRepo.GetUserConversations(user_id)` 获取用户的所有 `conversation_id` 列表。
    *   **并行/批量处理**:
        *   批量从 Redis 获取所有会话的未读数: `MGET unread:conv_id1:user_id1 unread:conv_id2:user_id1 ...`。
        *   批量从 Redis 获取所有会话的最后一条消息: 使用 Redis Pipeline 执行多个 `ZREVRANGE hot_messages:conv_id 0 0`。
        *   批量从 `im-repo` 获取所有会话对端（好友/群组）的详细信息。
    *   聚合所有信息，按 `last_message_time` 排序后返回。

#### 3.2 消息生命周期

##### 3.2.1 消息发送 (上行)
1.  **Client**: 构造消息体 JSON，包含 `conversationId`, `content`, `messageType`, `tempMessageId` (`client_msg_id`)，通过 WebSocket 发送。
2.  **Gateway (`readPump`)**:
    *   读取到消息，解析 JSON。
    *   将消息体转换为 `protobuf.SendMessageRequest`。
    *   构造 `protobuf.KafkaMessage`，生成 `trace_id`。
    *   调用 `infra.mq.Producer` 将消息生产到 Kafka Topic: `im-upstream-topic`。
    *   **立即**向客户端的 `writePump` channel 发送一个 `message-ack` 消息，包含 `tempMessageId` 和一个临时 `messageId` (可以是`tempMessageId`本身)，用于优化UI。

##### 3.2.2 消息处理 (核心逻辑)
1.  **Logic (消费)**: 作为消费者组消费 `im-upstream-topic`，支持并发消费和负载均衡。
2.  **Logic (幂等性检查)**:
    *   执行 `SET msg_dedup:{client_msg_id} 1 EX 60 NX`
    *   如果返回0（已存在），记录重复消息日志并直接 `ack`，流程终止
    *   如果Redis操作失败，记录错误但继续处理（容错设计）
3.  **Logic (ID与序列生成)**:
    *   调用 `infra.idgen.NextID()` 生成全局唯一的 `message_id`
    *   调用 Redis `INCR conv_seq:{conversation_id}` 原子获取 `seq_id`
    *   如果序列生成失败，进行重试（最多3次）
4.  **Logic (持久化 - 关键路径)**:
    *   构造 `repo.SaveMessageRequest`，包含所有消息字段和元数据
    *   调用 `repo.MessageRepo.SaveMessage` RPC，设置合理超时（如5秒）
    *   `im-repo` 内部实现事务性操作：
        *   **步骤1**: 将消息写入 MySQL `messages` 表（主存储）
        *   **步骤2**: 将消息JSON写入 Redis ZSET: `ZADD hot_messages:{conversation_id} {seq_id} {message_json}`
        *   **步骤3**: 修剪ZSET保留最近300条: `ZREMRANGEBYRANK hot_messages:{conversation_id} 0 -301`
        *   **步骤4**: 更新会话未读数: `INCR unread:{conversation_id}:{user_id}` (对所有相关用户)
    *   如果持久化失败，记录错误并进入重试队列
5.  **Logic (分发决策)**:
    *   **单聊**: 调用 `repo.UserRepo.GetUserSession(to_user_id)` 获取接收者的 `gateway_id`
    *   **群聊**:
        *   调用 `repo.GroupRepo.GetGroupInfo(group_id)` 获取群成员数
        *   **中小群(≤500人)**: 调用 `repo.GroupRepo.GetOnlineGroupMembers(group_id)` 批量获取在线成员的 `gateway_id`
        *   **超大群(>500人)**: 构造异步任务消息，生产到 `im-task-large-group-fanout-topic`
    *   **世界聊天室**: 获取所有在线用户的 `gateway_id` 列表
6.  **Logic (消息推送)**:
    *   构造下行消息体，包含完整消息内容、`seq_id`、发送者信息
    *   根据分发决策结果，批量生产消息到对应的下行Topic: `im-downstream-topic-{gateway_id}`
    *   使用Kafka事务保证消息推送的原子性
    *   记录推送统计信息用于监控

##### 3.2.3 消息接收 (下行)
1.  **Gateway (消费)**: 每个 `gateway` 实例消费自己的下行 Topic。
2.  **Gateway (`writePump`)**:
    *   从 Topic 收到消息，解析出 `user_id`。
    *   从内存 `user_id -> websocket_connection` 映射中找到对应的连接。
    *   将消息（格式化为 `new-message` JSON）写入该连接的 `writePump` channel。
    *   `writePump` goroutine 将消息通过 WebSocket 发送给客户端。
3.  **Client**: 接收到 `new-message`，根据 `seq_id` 渲染到正确位置。若发现 `seq_id` 不连续，可主动调用 API 拉取缺失消息。

### 4. 模块详细设计

#### 4.1 `im-infra` (基础库)
*   **定位**: 提供被所有服务依赖的、标准化的基础能力封装，确保技术栈统一和开发效率。

##### 4.1.1 核心模块设计
*   **`rpc-proto`**:
    *   定义所有服务间的 gRPC 接口 (`.proto` 文件)
    *   提供protobuf代码生成脚本和版本管理
    *   包含通用的错误码定义和响应结构
*   **`config`**:
    *   `viper` 封装，支持多种配置源（文件、环境变量、etcd）
    *   提供配置热更新机制和配置验证
    *   支持不同环境的配置隔离
*   **`logger`**:
    *   `zap` 封装，提供结构化、带 `TraceID` 的日志
    *   支持日志级别动态调整和日志轮转
    *   集成链路追踪信息自动注入
*   **`mq`**:
    *   `segmentio/kafka-go` 封装，提供简洁的 `Producer` 和 `ConsumerGroup` 接口
    *   支持消息序列化/反序列化、重试机制、死信队列
    *   提供消息发送和消费的监控指标
*   **`redis`**:
    *   `go-redis/v8` 封装，支持单机、哨兵、集群模式
    *   提供常用命令的封装和连接池管理
    *   支持分布式锁、限流器等高级功能
*   **`mysql`**:
    *   `gorm` 封装，管理主从连接池和读写分离
    *   提供事务管理、慢查询监控、连接池监控
    *   支持数据库迁移和健康检查
*   **`etcd`**:
    *   `etcd/client/v3` 封装，提供服务注册、发现和分布式锁
    *   支持服务健康检查、自动续约、故障转移
    *   提供分布式配置管理和选主功能
*   **`idgen`**:
    *   Snowflake 算法实现，支持机器ID自动分配
    *   提供ID生成性能监控和时钟回拨检测
    *   支持自定义时间戳精度和序列号位数
*   **`tracing`**:
    *   OpenTelemetry SDK 封装，提供全链路追踪
    *   自动为 gRPC 和 Kafka 注入拦截器/中间件
    *   支持采样策略配置和性能优化

#### 4.2 `im-repo` (仓储层)
*   **gRPC 接口**:
    ```protobuf
    // message_repo.proto
    service MessageRepo {
      // 保存消息到MySQL并更新缓存
      rpc SaveMessage(SaveMessageRequest) returns (google.protobuf.Empty);
      // 从MySQL分页获取历史消息
      rpc GetMessages(GetMessagesRequest) returns (GetMessagesResponse);
    }
    ```
*   **实现细节**:
    *   所有 gRPC 方法需处理数据库错误和缓存操作错误。
    *   `gorm` 配置需开启日志，用于调试。
    *   **读写分离**: 在 `im-repo` 内部维护两个 `gorm.DB` 实例（主库和从库），所有写操作走主库，读操作走从库。

#### 4.3 `im-logic` (逻辑层)
*   **gRPC 接口**:
    ```protobuf
    // conversation_logic.proto
    service ConversationLogic {
      // 获取会话列表
      rpc GetConversations(GetConversationsRequest) returns (GetConversationsResponse);
      // 标记会话已读
      rpc MarkConversationAsRead(MarkAsReadRequest) returns (google.protobuf.Empty);
    }
    ```
*   **实现细节**:
    *   服务启动时，初始化 Kafka 消费者组，开始消费上行消息。
    *   所有业务逻辑严格遵守：**输入校验 -> 调用 `im-repo` -> 业务决策 -> 调用 `im-repo`/`mq`** 的模式。
    *   **并发控制**: 对于需要访问外部资源的操作（如调用多个 `im-repo` RPC），应使用 `goroutine` 和 `sync.WaitGroup` 或 `errgroup` 进行并发处理，以降低延迟。

#### 4.4 `im-gateway` (网关层)
*   **实现细节**:
    *   **`main.go`**: 初始化配置、日志、etcd、gRPC客户端、Kafka生产者等。启动 HTTP 和 WebSocket 服务。
    *   **`http_handler.go`**: `gin` 路由和处理器，负责将 HTTP 请求转换为 gRPC 调用。
    *   **`ws_handler.go`**: WebSocket `Upgrade` 逻辑和连接管理。
    *   **`ws_conn.go`**: 定义 `Connection` 结构体，封装 `*websocket.Conn`，包含 `readPump`, `writePump` 方法和 `send chan []byte`。
    *   **`kafka_consumer.go`**: `gateway` 的 Kafka 消费者实现，收到消息后分发到对应的 `Connection.send` channel。

#### 4.5 `im-task` (任务层)
*   **实现细节**:
    *   **`main.go`**: 初始化配置、日志、gRPC客户端、Kafka生产者/消费者。
    *   **`consumer.go`**: 启动消费者组，循环消费 `im-task-topic`。
    *   **`processor.go`**: 定义任务分发器和各个任务的处理器。
        ```go
        // 任务处理器接口
        type TaskProcessor interface {
            Process(ctx context.Context, msg *sarama.ConsumerMessage) error
        }
        ```
    *   **`large_group_fanout.go`**: `TaskProcessor` 的具体实现，处理大群扩散逻辑。内部可能包含分片逻辑，将一个大任务拆分成多个小任务再次投递到 Kafka。

### 5. 数据模型详细设计

#### 5.1 数据库表 (MySQL)
(同上一份文档中的优化版，包含 `users`, `groups`, `group_members`, `messages` 表，并强调了主键和索引设计)

#### 5.2 缓存设计 (Redis)

##### 5.2.1 缓存策略概述
*   **缓存模式**: 主要采用Cache-Aside模式，部分场景使用Write-Through
*   **一致性保证**: 通过缓存失效而非更新来保证最终一致性
*   **容错设计**: 缓存故障不影响核心功能，降级到直接访问数据库

##### 5.2.2 缓存结构设计

| 用途 | Key 格式 | Value 类型 | TTL | 描述与策略 |
| :--- | :--- | :--- | :--- | :--- |
| **用户信息** | `user_info:{user_id}` | **HASH** | 24小时 | 缓存用户基本信息。读时加载，写时失效。 |
| **用户在线状态** | `user_session:{user_id}` | **HASH** | 30分钟 | 存储用户在线状态和gateway_id。支持自动过期清理。 |
| **会话序列号** | `conv_seq:{conversation_id}` | **STRING** | 永久 | 使用INCR原子生成seq_id。定期备份到MySQL。 |
| **群组成员列表** | `group_members:{group_id}` | **SET** | 1小时 | 缓存群成员user_id列表。成员变更时失效。 |
| **消息去重** | `msg_dedup:{client_msg_id}` | **STRING** | 60秒 | 防止消息重复处理。使用SETEX原子操作。 |
| **热点消息缓存** | `hot_messages:{conv_id}` | **ZSET** | 7天 | 缓存最近300条消息。Score为seq_id，Member为消息JSON。 |
| **会话未读数** | `unread:{conv_id}:{user_id}` | **STRING** | 30天 | 使用INCR/DECR原子操作。支持批量重置。 |
| **用户会话列表** | `user_conversations:{user_id}` | **ZSET** | 1小时 | 缓存用户参与的会话列表。Score为最后消息时间。 |
| **在线用户计数** | `online_users_count` | **STRING** | 实时 | 全局在线用户数统计。用于监控和限流。 |

##### 5.2.3 缓存更新策略
*   **用户信息**: 读时加载，用户信息变更时删除缓存
*   **群组信息**: 读时加载，成员变更时删除相关缓存
*   **消息缓存**: 写时更新，定期清理过期数据
*   **计数器**: 实时更新，定期同步到数据库

##### 5.2.4 缓存监控与告警
*   **命中率监控**: 各类缓存的命中率统计，目标>90%
*   **性能监控**: 缓存操作延迟、QPS统计
*   **容量监控**: Redis内存使用率、键数量统计
*   **异常告警**: 缓存操作失败率、连接异常告警

### 6. 非功能性设计

#### 6.1 性能与可扩展性
*   **全链路压测**: 使用 `JMeter` 或 `k6` 等工具，模拟大量用户连接和消息收发，识别系统瓶颈。
*   **数据库优化**: 定期分析慢查询日志，优化 SQL 和索引。
*   **伸缩策略**: 为 K8s `Deployment` 配置 `HorizontalPodAutoscaler (HPA)`，基于 CPU 和内存使用率自动伸缩服务实例。

#### 6.2 高可用性
*   **心跳机制**: WebSocket 连接采用双向心跳。客户端定时 `ping`，服务端若在 `3 * ping_interval` 内未收到则认为断线。服务端也定时 `ping`，可穿透某些网络设备的NAT超时。
*   **健康检查**: 所有服务需实现 gRPC 健康检查协议，并暴露 HTTP 健康检查端点 `/healthz`，供 K8s 进行存活探针和就绪探针检测。

#### 6.3 安全性
*   **输入验证**: 所有来自客户端的输入（API参数、消息内容）都必须在 `im-gateway` 或 `im-logic` 层进行严格验证（长度、格式、类型）。
*   **权限控制**: `im-logic` 在处理敏感操作（如解散群聊、踢人）时，必须校验操作者是否具有相应权限（如群主或管理员）。

### 7. 部署与运维

*   **容器化**: 所有服务均打包为轻量级的 Docker 镜像（基于 `alpine`）。
*   **CI/CD**: 搭建自动化持续集成/持续部署流水线（如 Jenkins, GitLab CI），代码提交后自动运行测试、构建镜像并部署到测试环境。
*   **配置管理**: 使用 K8s `ConfigMap` 和 `Secret` 管理配置文件和敏感信息，并通过 etcd 实现动态配置更新。
*   **可观测性**: 部署 Prometheus + Grafana + Alertmanager 进行指标监控和告警，部署 Jaeger 进行链路追踪，部署 ELK/Loki 栈进行日志聚合与查询。