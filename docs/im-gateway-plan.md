## **GoChat - `im-gateway` 模块开发设计与计划**

**致实习生同学：**

你好！欢迎加入 GoChat 项目。`im-gateway` 是我们整个系统的“大门”，所有用户流量都将从这里经过。它的稳定性和性能至关重要。这个任务非常有挑战性，也充满了学习机会。

本指南将带你一步一步地完成 `im-gateway` 的开发。请仔细阅读，不要害怕提问。让我们开始吧！

### **1. 模块职责与目标 (The "What")**

在动手编码前，我们必须再次明确 `im-gateway` 的核心职责：

1.  **协议转换器**: 它是连接“外部世界”（客户端）和“内部世界”（后端服务）的桥梁。
    *   **外部**: 对客户端暴露 **HTTP/RESTful API** 和 **WebSocket** 协议。
    *   **内部**: 与后端服务（如 `im-logic`）通过 **gRPC** 通信；与消息系统通过 **Kafka** 通信。
2.  **用户认证官**: 它是安全的第一道防线，负责验证所有传入请求的合法性（JWT Token）。
3.  **连接管理器**: 它需要稳定地维护成千上万个来自客户端的 WebSocket 长连接。
4.  **消息代理人**:
    *   **上行**: 将客户端发来的消息可靠地投递到 Kafka。
    *   **下行**: 从 Kafka 消费只属于本节点用户的消息，并精确地推送给他们。

**我们的目标**: 开发一个**无状态、高可用、可水平扩展**的网关服务。

### **2. 技术栈与依赖 (The "Tools")**

我们将使用以下 Go 语言社区中最成熟、最流行的库来构建 `im-gateway`：

| 用途 | 库/工具 | 学习重点 |
| :--- | :--- | :--- |
| **HTTP 框架** | `github.com/gin-gonic/gin` | 路由、中间件、参数绑定、JSON响应 |
| **WebSocket** | `github.com/gorilla/websocket` | 连接升级（Upgrade）、读/写消息、Ping/Pong |
| **gRPC 客户端** | `google.golang.org/grpc` | 创建客户端连接、调用远程方法、处理上下文(Context) |
| **Kafka 客户端** | `github.com/segmentio/kafka-go` | 生产者(Producer)和消费者(Reader)的配置与使用 |
| **配置管理** | `github.com/spf13/viper` | 从文件/环境变量读取配置 |
| **日志** | `go.uber.org/zap` | 结构化日志、日志分级 |
| **服务发现** | `go.etcd.io/etcd/client/v3` | 连接etcd、注册服务、发现其他服务 |

### **3. 项目结构 (The "Blueprint")**

一个清晰的项目结构是良好维护的开始。建议 `im-gateway` 的目录结构如下：

```
im-gateway/
├── cmd/
│   └── main.go              # 程序入口：初始化、启动服务
├── internal/
│   ├── config/              # 配置加载与定义
│   ├── handler/
│   │   ├── http_handler.go    # 所有 HTTP API 的处理器
│   │   └── ws_handler.go      # WebSocket 的核心处理器
│   ├── middleware/
│   │   └── auth.go            # JWT 认证中间件
│   ├── service/
│   │   ├── kafka_consumer.go  # Kafka 消费者逻辑
│   │   └── ws_conn.go         # WebSocket 连接的封装
│   ├── router/
│   │   └── router.go          # 注册所有路由
│   └── rpc/
│       └── logic_client.go    # im-logic 的 gRPC 客户端封装
├── go.mod
└── go.sum
```

### **4. 开发计划：分阶段进行 (The "Plan")**

我们将开发过程分为四个阶段。请按顺序完成，每完成一个阶段，你都会获得一个可运行、可验证的里程碑。

---

#### **阶段一：搭建骨架与实现 HTTP API (预计2天)**

**目标**: 让 `im-gateway` 作为一个独立的 HTTP 服务器跑起来，并能处理用户登录等基础 API。

**任务分解**:
1.  **初始化项目**:
    *   创建 `im-gateway` 目录，执行 `go mod init`。
    *   引入 `gin`, `viper`, `zap` 等基础依赖。
2.  **配置管理 (`internal/config`)**:
    *   定义一个 `Config` 结构体，包含服务端口、日志级别等。
    *   编写一个 `LoadConfig()` 函数，使用 `viper` 从本地 `config.yaml` 文件中加载配置。
3.  **程序入口 (`cmd/main.go`)**:
    *   调用 `LoadConfig()` 加载配置。
    *   初始化 `zap` 日志记录器。
    *   初始化 `gin` 引擎。
    *   调用路由注册函数。
    *   启动 HTTP 服务器 `r.Run(":8080")`。
4.  **路由 (`internal/router`)**:
    *   创建一个 `RegisterRoutes(r *gin.Engine)` 函数。
    *   定义 `/api/auth/login` 和 `/api/auth/register` 路由，并暂时绑定到“桩函数”(stub function)。
5.  **gRPC 客户端 (`internal/rpc`)**:
    *   假设 `im-logic` 的 gRPC 服务已经可以运行（或者你有一个 mock server）。
    *   编写 `NewLogicClient()` 函数，使用 `grpc.Dial` 连接到 `im-logic` 服务。你需要处理服务发现的逻辑（暂时可以硬编码地址，后续再集成 etcd）。
6.  **HTTP 处理器 (`internal/handler/http_handler.go`)**:
    *   实现 `LoginHandler(c *gin.Context)` 和 `RegisterHandler(c *gin.Context)`。
    *   在 Handler 中：
        *   使用 `c.ShouldBindJSON()` 解析请求体。
        *   调用第5步中创建的 `logicClient`，向 `im-logic` 发起 gRPC 请求。
        *   根据 gRPC 的返回结果，使用 `c.JSON()` 向客户端返回成功或失败的响应。

**验收标准**:
*   启动 `im-gateway` 服务。
*   使用 `curl` 或 Postman 工具，可以成功调用 `/api/auth/login` 接口，并能看到来自 `im-logic` 的（模拟）响应。

---

#### **阶段二：实现 WebSocket 连接管理 (预计3天)**

**目标**: 让客户端可以成功与 `im-gateway` 建立 WebSocket 连接，并实现心跳维持。

**任务分解**:
1.  **JWT 认证中间件 (`internal/middleware/auth.go`)**:
    *   编写一个 `JWTMiddleware()` 函数，它是一个 `gin.HandlerFunc`。
    *   在这个中间件中：
        *   从请求头 `Authorization: Bearer <token>` 或查询参数 `?token=<token>` 中提取 JWT。
        *   使用 `jwt-go` 库验证 Token 的签名和有效期。
        *   验证成功后，将解析出的 `user_id` 等信息存入 `gin.Context` 中，供后续的 Handler 使用 (`c.Set("userId", userId)`)。
        *   验证失败则返回 `401 Unauthorized` 错误。
2.  **WebSocket 连接封装 (`internal/service/ws_conn.go`)**:
    *   定义一个 `Connection` 结构体，封装 `*websocket.Conn`、一个 `send chan []byte`（用于向客户端发送消息的缓冲 channel）以及 `user_id` 等信息。
    *   为 `Connection` 实现两个核心方法：
        *   `readPump()`: 在一个 `for` 循环中不断调用 `conn.ReadMessage()` 读取客户端消息。读取到的消息暂时可以只打印日志。**关键**: 此方法需要处理心跳（`Pong` 消息）和连接关闭错误。
        *   `writePump()`: 在一个 `for` 循环中不断从 `send` channel 中读取消息，并调用 `conn.WriteMessage()` 发送给客户端。**关键**: 此方法需要处理 `Ping` 消息的发送。
3.  **连接管理器**:
    *   在 `ws_handler.go` 中定义一个全局的、线程安全的 `Manager` 结构体。
    *   `Manager` 内部包含一个 `map[string]*Connection`，用于存储 `user_id` 到 `Connection` 的映射。**别忘了使用 `sync.RWMutex` 保护并发读写！**
    *   `Manager` 需要有 `Register`, `Unregister`, `GetUserConn` 等方法。
4.  **WebSocket 处理器 (`internal/handler/ws_handler.go`)**:
    *   创建一个 `WSHandler(c *gin.Context)`。
    *   在 `router` 中为 `/ws` 路径注册此 Handler，并应用第1步的 `JWTMiddleware`。
    *   在 Handler 中：
        *   使用 `gorilla/websocket` 的 `upgrader.Upgrade()` 方法将 HTTP 连接升级为 WebSocket 连接。
        *   从 `gin.Context` 中获取 `user_id`。
        *   创建一个新的 `Connection` 实例。
        *   调用 `Manager.Register()` 将新连接注册进去。
        *   **异步启动**: `go conn.writePump()` 和 `go conn.readPump()`。

**验收标准**:
*   使用一个简单的 WebSocket 客户端（如 `wscat` 或浏览器控制台），携带有效的 JWT Token，可以成功连接到 `/ws`。
*   在 `im-gateway` 的日志中，可以看到连接建立、心跳消息以及连接断开的日志。
*   可以同时连接多个客户端，`Manager` 中能正确维护所有连接。

---

#### **阶段三：实现消息上行 (预计1天)**

**目标**: 将从客户端收到的消息，可靠地生产到 Kafka。

**任务分解**:
1.  **引入 Kafka 依赖**: `go get github.com/segmentio/kafka-go`。
2.  **配置 Kafka**: 在 `config.yaml` 和 `Config` 结构体中增加 Kafka 的 `brokers` 和 `topic`（上行 Topic）配置。
3.  **封装 Kafka 生产者**:
    *   在 `cmd/main.go` 中初始化一个全局的 Kafka `Producer`。
    *   `kafka.NewWriter(...)`
4.  **修改 `readPump`**:
    *   在 `internal/service/ws_conn.go` 的 `readPump` 方法中，当读取到普通消息（非控制消息）时：
        *   （暂时）假设消息是 JSON 格式，解析它。
        *   构造一个 `kafka-go.Message`。
        *   调用全局的 `Producer.WriteMessages()` 将消息发送到 Kafka。
        *   处理发送失败的情况（如记录错误日志，后续可以增加重试逻辑）。

**验收标准**:
*   启动 `im-gateway` 和一个 Kafka 服务（可以使用 Docker）。
*   客户端通过 WebSocket 发送一条消息。
*   使用 Kafka 的命令行工具（`kafka-console-consumer.sh`）可以消费到这条消息。

---

#### **阶段四：实现消息下行与服务发现 (预计2天)**

**目标**: 消费 Kafka 中的下行消息，并精确地推送到目标用户。

**任务分解**:
1.  **服务注册与发现**:
    *   引入 `etcd` 客户端依赖。
    *   在 `cmd/main.go` 中，每个 `im-gateway` 实例启动时，必须生成一个唯一的 `gateway_id`（例如 `hostname + "-" + random_string`）。
    *   **服务注册**: 将自己的服务信息（如 gRPC 地址）注册到 etcd 的一个特定路径下。
    *   **在线状态注册**: 在 `ws_handler.go` 中，当 WebSocket 连接建立后，需要向 Redis 写入 `HSET user_session:{user_id} gateway_id {current_gateway_id}`。
2.  **Kafka 消费者 (`internal/service/kafka_consumer.go`)**:
    *   编写一个 `StartConsumer()` 函数。
    *   在函数内部，根据当前 `gateway_id` 确定要订阅的下行 Topic 名称（`im-downstream-topic-{gateway_id}`）。
    *   使用 `kafka.NewReader(...)` 创建一个消费者。
    *   在一个 `for` 循环中不断调用 `reader.FetchMessage()`。
    *   当消费到消息后：
        *   解析消息，获取到目标 `user_id` 和消息内容。
        *   调用 `Manager.GetUserConn(userId)` 找到对应的 `Connection`。
        *   如果找到连接，将消息内容写入 `conn.send` channel。
3.  **启动消费者**:
    *   在 `cmd/main.go` 中，异步启动消费者：`go kafka_consumer.StartConsumer()`。

**验收标准**:
*   启动 `im-gateway`, `etcd`, `redis`, `kafka`。
*   一个客户端 A 连接到 `im-gateway`。
*   手动使用 Kafka 命令行工具向 `im-downstream-topic-{gateway_id_of_A}` Topic 生产一条消息。
*   客户端 A 应该能收到这条消息。

---

### **5. 最终集成与测试**

完成以上四个阶段后，`im-gateway` 的核心功能就全部完成了。接下来需要与 `im-logic` 等其他服务联调，确保整个消息链路畅通无阻。

祝你开发顺利！记住，每次只专注于一小部分，积小胜为大胜。随时准备好向你的导师或同事寻求帮助。