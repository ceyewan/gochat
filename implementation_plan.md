# 实施计划

[概述]  
本文档概述了通过建立清晰的代码架构和创建全面、开发者友好的文档来重构GoChat IM系统的计划。主要目标是为现有前端创建健壮、可扩展和可维护的后端。架构将基于确认的微服务角色：`im-gateway` 用于客户端连接，`im-logic` 用于业务逻辑，`im-repo` 用于数据持久化，`im-task` 用于异步后台作业。文档将在 `docs/` 目录中以Markdown格式创建。

[类型]  
核心数据结构将在每个微服务的 `internal/model` 或 `internal/types` 包中定义为Go结构体，并作为Protobuf消息用于RPC通信。

-   **User**：`(id, username, password_hash, avatar, created_at, updated_at)`
-   **Message**：`(id, conversation_id, sender_id, content, type, seq, created_at)`
-   **Conversation**：`(id, type, last_message_id, created_at, updated_at)`
-   **ConversationMember**：`(conversation_id, user_id, unread_count, read_seq, role)`
-   **Group**：`(id, name, avatar, owner_id, member_count, created_at, updated_at)`（注意：世界聊天室将是群组的一个特殊实例）。

[文件]
新文档将在 `docs/` 目录中创建，现有的微服务代码将被重构以提高清晰度和一致性。

-   **要创建的新文件**：
    -   `docs/01_overview.md`：高层架构概述。
    -   `docs/02_http_api.md`：描述 `im-gateway` 暴露的RESTful和WebSocket API。
    -   `docs/03_rpc_services.md`：记录内部gRPC服务，并链接到 `.proto` 文件。
    -   `docs/04_mq_topics.md`：定义用于异步通信的Kafka主题和消息格式。
    -   `docs/05_data_models.md`：详细说明数据库模式和数据模型。
-   **要修改的现有文件**：
    -   `api/proto/im_logic/logic.proto`：定义认证、用户、对话、群组和消息逻辑的服务。
    -   `api/proto/im_repo/repo.proto`：定义数据访问操作的服务。
    -   `im-gateway/internal/server/http.go`：基于 `docs/02_http_api.md` 实现HTTP处理器。
    -   `im-gateway/internal/server/ws.go`：实现WebSocket连接处理。
    -   `im-logic/internal/service/*.go`：为每个gRPC服务实现业务逻辑。
    -   `im-repo/internal/repository/*.go`：使用GORM和Redis实现数据访问逻辑。
    -   `im-task/internal/processor/fanout.go`：基于Kafka消息实现消息扇出逻辑。
-   **要删除的文件**：
    -   `docs/`：整个旧文档目录将被 `docs/` 替换。

[函数]  
函数将在各自微服务中的服务和存储库中组织。

-   **新函数**：
    -   `im-logic/internal/service/auth_service.go`：`Register()`、`Login()`、`GenerateToken()`、`ParseToken()`。
    -   `im-logic/internal/service/message_service.go`：`SendMessage()`、`GetMessages()`。
    -   `im-repo/internal/repository/conversation_repo.go`：`CreateConversation()`、`GetConversations()`。
-   **修改函数**：
    -   `im-repo/internal/service/*.go` 中的所有现有函数将被重构以与 `repo.proto` 中定义的新gRPC接口对齐。
    -   `im-gateway/cmd/main.go`：将被更新以初始化和运行HTTP和WebSocket服务器。

[类]  
Go没有类，但结构体将用于定义服务和存储库。

-   **新结构体**：
    -   `im-logic/internal/service/AuthService`：实现认证gRPC服务。
    -   `im-logic/internal/service/MessageService`：实现消息gRPC服务。
    -   `im-repo/internal/repository/UserRepository`：实现用户数据访问逻辑。
-   **修改结构体**：
    -   现有的服务和存储库结构体将被更新以实现新的gRPC接口，并与重构的架构对齐。

[依赖项]  
主要依赖项（`gin`、`gorm`、`grpc`、`kafka-go`、`redis`）已在 `go.mod` 中存在。不需要主要的新依赖项。

-   我们将确保所有微服务使用共享依赖项（如 `gRPC` 和 `Protobuf`）的一致版本。

[测试]  
每个微服务将有自己的单元和集成测试套件。

-   **测试文件要求**：
    -   将为每个服务和存储库创建单元测试（例如，`im-logic/internal/service/auth_service_test.go`）。
    -   将添加集成测试以验证服务之间的交互（例如，`im-gateway` 调用 `im-logic`）。
-   **验证策略**：
    -   在单元测试中使用依赖项的模拟实现。
    -   使用 `docker-compose` 为集成测试启动依赖服务（DB、Kafka、Redis）。

[实施顺序]  
实施将按逻辑顺序进行，从数据层开始向上到API网关。

1.  **定义合约**：完成所有 `.proto` 文件以用于gRPC服务（`im_logic.proto`、`im_repo.proto`）。
2.  **生成代码**：从 `.proto` 文件生成gRPC服务器和客户端代码。
3.  **实施 `im-repo`**：在 `im-repo` 微服务中实施数据访问逻辑和gRPC服务。
4.  **实施 `im-logic`**：在 `im-logic` 微服务中实施核心业务逻辑和gRPC服务，该服务调用 `im-repo`。
5.  **实施 `im-task`**：在 `im-task` 中实施Kafka消费者以处理异步消息处理。
6.  **实施 `im-gateway`**：在 `im-gateway` 中实施HTTP/WebSocket处理器，该处理器调用 `im-logic`。
7.  **创建文档**：在 `docs/` 目录中编写Markdown文档。
8.  **更新部署**：更新 `docker-compose.yml` 文件以反映任何配置更改。
9.  **测试**：为所有组件编写和运行单元和集成测试。
