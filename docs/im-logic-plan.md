## **GoChat - `im-logic` 模块开发设计与计划**

**致实习生同学：**

你好！接下来你将负责开发 GoChat 系统最核心的模块——`im-logic`。可以把它想象成我们系统的“中央司令部”，所有的业务规则、决策和数据流转都在这里汇集处理。这个任务将极大地锻炼你的业务抽象能力、并发编程能力和对分布式系统的理解。

本指南将引导你完成从零到一的构建过程。请仔细遵循，大胆实践。

### **1. 模块职责与目标 (The "What")**

`im-logic` 不直接与客户端交互，它是一个纯粹的后端服务。它的核心职责是：

1.  **业务逻辑处理器**: 以 **gRPC 服务**的形式，对外提供所有核心业务能力。例如，处理用户注册、登录请求；处理获取会话列表、好友列表的请求等。
2.  **消息流转中心**: 作为 **Kafka 消费者**，处理所有从 `im-gateway` 发来的上行消息；同时作为 **Kafka 生产者**，将处理后的消息分发给下游（`im-gateway` 或 `im-task`）。
3.  **数据协调者**: 它本身**不直接操作数据库或缓存**。所有的数据读写请求，都必须通过调用 `im-repo` 模块的 gRPC 接口来完成。
4.  **决策者**: 它是业务规则的执行者。例如，判断一条群消息应该由自己实时扩散，还是应该派发给 `im-task` 异步处理。

**我们的目标**: 开发一个**无状态、高可靠、逻辑清晰**的业务核心服务。

### **2. 技术栈与依赖 (The "Tools")**

| 用途 | 库/工具 | 学习重点 |
| :--- | :--- | :--- |
| **gRPC 框架** | `google.golang.org/grpc` | 定义 `.proto` 服务、实现 gRPC Server、错误处理 |
| **Kafka 客户端** | `github.com/segmentio/kafka-go` | 生产者和消费者组（ReaderGroup）的配置与使用 |
| **gRPC 客户端** | `google.golang.org/grpc` | 调用 `im-repo` 服务的 gRPC 接口 |
| **分布式ID** | (来自 `im-infra`) | Snowflake 算法的使用 |
| **配置管理** | `github.com/spf13/viper` | 加载服务配置 |
| **日志** | `go.uber.org/zap` | 结构化日志 |
| **服务发现** | `go.etcd.io/etcd/client/v3` | 注册服务、发现 `im-repo` |

### **3. 项目结构 (The "Blueprint")**

`im-logic` 的结构应该围绕其核心职责——gRPC 服务和 Kafka 消息处理来组织。

```
im-logic/
├── cmd/
│   └── main.go              # 程序入口：初始化、启动 gRPC 和 Kafka 服务
├── internal/
│   ├── config/              # 配置定义与加载
│   ├── service/
│   │   ├── auth_service.go    # 认证相关的 gRPC 服务实现
│   │   ├── conv_service.go    # 会话相关的 gRPC 服务实现
│   │   └── ...                # 其他 gRPC 服务实现
│   ├── consumer/
│   │   └── message_consumer.go # Kafka 消息消费和处理的核心逻辑
│   └── rpc/
│       └── repo_client.go     # im-repo 的 gRPC 客户端封装
├── pkg/
│   └── proto/                 # 存放 im-logic 对外提供的 .proto 文件
├── go.mod
└── go.sum
```

### **4. 开发计划：分阶段进行 (The "Plan")**

我们将开发过程分为四个阶段。这种方式有助于控制复杂性，保证每一步都有坚实的基础。

---

#### **阶段一：搭建骨架与实现 gRPC 服务 (预计3天)**

**目标**: 让 `im-logic` 作为一个 gRPC 服务器跑起来，并能处理用户认证等基础业务。

**任务分解**:
1.  **初始化项目**: 创建目录，`go mod init`，引入 `grpc-go`, `viper`, `zap` 等依赖。
2.  **定义 gRPC 接口 (`pkg/proto`)**:
    *   创建一个 `auth_logic.proto` 文件。
    *   在文件中定义 `AuthService`，包含 `Register` 和 `Login` 两个 RPC 方法，并定义好 `Request` 和 `Response` 消息体。
    *   使用 `protoc` 工具生成 Go 代码。
3.  **配置与入口 (`internal/config`, `cmd/main.go`)**:
    *   定义 `Config` 结构体，包含 gRPC 服务端口、`im-repo` 地址、日志级别等。
    *   在 `main.go` 中：加载配置、初始化日志、创建 TCP 监听 (`net.Listen`)、初始化 gRPC 服务器 (`grpc.NewServer()`)。
4.  **`im-repo` 客户端 (`internal/rpc`)**:
    *   编写 `NewRepoClient()` 函数，使用 `grpc.Dial` 连接到 `im-repo` 服务（暂时可硬编码地址）。
5.  **实现 gRPC 服务 (`internal/service/auth_service.go`)**:
    *   创建一个 `AuthServiceServer` 结构体，它需要持有 `repoClient`。
    *   为这个结构体实现 `Register` 和 `Login` 方法，方法签名必须与生成的 proto 代码一致。
    *   在方法内部：
        *   校验输入参数。
        *   调用 `repoClient` 的相应方法（如 `CreateUser`, `GetUserByUsername`）来与数据层交互。
        *   执行业务逻辑（如密码哈希比较、生成JWT）。
        *   返回响应或 gRPC 错误。
6.  **注册并启动服务**:
    *   在 `main.go` 中，创建 `AuthServiceServer` 的实例。
    *   将该实例注册到 gRPC 服务器：`pb.RegisterAuthServiceServer(grpcServer, authSvc)`。
    *   启动服务：`grpcServer.Serve(lis)`。

**验收标准**:
*   启动 `im-logic` 和 `im-repo` 服务。
*   使用一个 gRPC 客户端工具（如 `grpcurl`）或编写一个简单的 Go 客户端，可以成功调用 `im-logic` 的 `Login` RPC，并获得正确的 JWT 响应。

---

#### **阶段二：实现 Kafka 消息消费 (预计2天)**

**目标**: 让 `im-logic` 能够消费来自 `im-gateway` 的上行消息。

**任务分解**:
1.  **引入 Kafka 依赖**: `go get github.com/segmentio/kafka-go`。
2.  **配置 Kafka**: 在配置中增加 Kafka `brokers` 和上行 Topic (`im-upstream-topic`) 的配置。
3.  **消息消费逻辑 (`internal/consumer/message_consumer.go`)**:
    *   创建一个 `MessageConsumer` 结构体，它需要持有 `repoClient` 和 Kafka 生产者（用于后续分发）。
    *   创建一个 `Start()` 方法。
    *   在 `Start()` 方法中：
        *   使用 `kafka.NewReader(...)` 创建一个消费者（或使用 `ReaderGroup`）。
        *   在一个 `for` 循环中不断调用 `reader.FetchMessage()` 或 `reader.ReadMessage()` 来消费消息。
        *   **暂时**，只将消费到的消息内容打印到日志中。**关键**: 记得在处理完消息后调用 `reader.CommitMessages()` 来提交位移。
4.  **启动消费者**:
    *   在 `cmd/main.go` 中，创建 `MessageConsumer` 实例，并异步启动它：`go msgConsumer.Start()`。

**验收标准**:
*   启动 `im-logic` 和 Kafka。
*   使用 Kafka 命令行工具向 `im-upstream-topic` 生产一条消息。
*   在 `im-logic` 的日志中，应该能看到这条消息的内容被打印出来。

---

#### **阶段三：实现消息处理与分发核心逻辑 (预计3天)**

**目标**: 将阶段二消费到的消息，进行完整的业务处理，并推送到下行 Kafka Topic。

**任务分解**:
1.  **初始化 Kafka 生产者**: 在 `main.go` 中初始化一个全局的 Kafka `Producer`，用于向所有下行和任务 Topic 发送消息。
2.  **增强 `MessageConsumer`**:
    *   在 `MessageConsumer` 的处理循环中，替换掉之前的日志打印逻辑。
    *   **反序列化**: 将 Kafka 消息的 `body`（`[]byte`）反序列化为 `protobuf.SendMessageRequest`。
    *   **幂等性检查**: 调用 Redis（通过 `im-repo` 的新接口或直接在 `im-logic` 中引入 redis client）检查 `client_msg_id` 是否重复。
    *   **ID与序列生成**:
        *   调用 `infra.idgen` 生成 `message_id`。
        *   调用 `im-repo` 的接口（需要新增）从 Redis `INCR` 获取 `seq_id`。
    *   **持久化**: 调用 `im-repo` 的 `SaveMessage` RPC。
    *   **分发决策 (核心)**:
        *   判断是单聊还是群聊。
        *   **单聊**: 调用 `im-repo` 获取接收者的 `gateway_id`。
        *   **群聊**: 调用 `im-repo` 获取群成员数，如果小于阈值（如500），则再调用 `im-repo` 获取所有在线成员的 `gateway_id` 列表。
    *   **消息生产**:
        *   根据分发决策的结果，构造完整的下行消息体（`protobuf`）。
        *   调用 Kafka `Producer`，将消息生产到对应的下行 Topic（`im-downstream-topic-{gateway_id}`）。对于群聊，这可能是一个循环操作。

**验收标准**:
*   启动 `im-logic`, `im-repo`, `redis`, `kafka`。
*   向 `im-upstream-topic` 发送一条单聊消息。
*   能在 Kafka 中观测到一条消息被生产到了正确的下行 Topic（如 `im-downstream-topic-gateway123`）。
*   向 `im-upstream-topic` 发送一条小群聊消息。
*   能在 Kafka 中观测到多条消息被生产到了多个不同的下行 Topic。

---

#### **阶段四：实现高级业务与服务发现 (预计2天)**

**目标**: 实现会话列表等复杂查询，并集成服务发现。

**任务分解**:
1.  **集成服务发现**:
    *   修改 `internal/rpc/repo_client.go`，不再硬编码 `im-repo` 地址。而是通过 etcd 发现 `im-repo` 的服务地址。
    *   在 `cmd/main.go` 中，`im-logic` 启动时，需要将自己的 gRPC 服务注册到 etcd。
2.  **实现会话列表服务 (`internal/service/conv_service.go`)**:
    *   定义 `ConversationService` 的 gRPC 接口 (`.proto`) 和服务实现。
    *   实现 `GetConversations` 方法：
        *   调用 `im-repo` 获取用户的会话ID列表。
        *   **并发/批量调用**: 使用 `goroutine` 并发地为每个会话调用 `im-repo` 获取最后一条消息、未读数、对端用户信息。
        *   聚合所有数据，排序后返回。
3.  **实现大群聊任务派发**:
    *   在 `MessageConsumer` 的分发决策逻辑中，补充大群聊的分支。
    *   当判断为大群时，构造一个轻量级的任务消息（如 `{"group_id": ..., "message_id": ...}`），并生产到 `im-task-topic`。

**验收标准**:
*   启动包括 `etcd` 在内的全套服务。服务间能够自动发现并通信。
*   通过 gRPC 客户端调用 `GetConversations` 接口，能够返回结构正确、数据完整的会话列表。
*   向 `im-upstream-topic` 发送一条大群聊消息，能在 `im-task-topic` 中消费到对应的任务消息。

---

### **5. 总结与后续**

完成以上四个阶段，`im-logic` 模块的核心功能便已成型。后续的工作将是根据需要，不断增加新的 gRPC 业务接口和完善错误处理、监控等细节。

记住，`im-logic` 的代码质量直接决定了我们业务的上限。请务必编写清晰、可测试的代码，并为关键逻辑添加详尽的注释。祝你编码愉快！