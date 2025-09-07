# Implementation Plan

[Overview]
This document outlines the plan to refactor the GoChat IM system by establishing a clear code architecture and creating comprehensive, developer-friendly documentation to guide implementation. The primary goal is to create a robust, scalable, and maintainable backend for the existing frontend. The architecture will be based on the confirmed microservice roles: `im-gateway` for client-facing connections, `im-logic` for business logic, `im-repo` for data persistence, and `im-task` for asynchronous background jobs. Documentation will be created in Markdown format within the `new_docs/` directory.

[Types]
Core data structures will be defined as Go structs within each microservice's `internal/model` or `internal/types` package, and as Protobuf messages for RPC communication.

-   **User**: `(id, username, password_hash, avatar, created_at, updated_at)`
-   **Message**: `(id, conversation_id, sender_id, content, type, seq, created_at)`
-   **Conversation**: `(id, type, last_message_id, created_at, updated_at)`
-   **ConversationMember**: `(conversation_id, user_id, unread_count, read_seq, role)`
-   **Group**: `(id, name, avatar, owner_id, member_count, created_at, updated_at)` (Note: The World Chatroom will be a special instance of a group).

[Files]
New documentation will be created in the `new_docs/` directory, and existing microservice code will be refactored for clarity and consistency.

-   **New Files to be Created**:
    -   `new_docs/01_overview.md`: High-level architecture overview.
    -   `new_docs/02_http_api.md`: Describes the RESTful and WebSocket APIs exposed by `im-gateway`.
    -   `new_docs/03_rpc_services.md`: Documents the internal gRPC services, with links to `.proto` files.
    -   `new_docs/04_mq_topics.md`: Defines the Kafka topics and message formats used for asynchronous communication.
    -   `new_docs/05_data_models.md`: Details the database schema and data models.
-   **Existing Files to be Modified**:
    -   `api/proto/im_logic/logic.proto`: Define services for auth, user, conversation, group, and message logic.
    -   `api/proto/im_repo/repo.proto`: Define services for data access operations.
    -   `im-gateway/internal/server/http.go`: Implement HTTP handlers based on `new_docs/02_http_api.md`.
    -   `im-gateway/internal/server/ws.go`: Implement WebSocket connection handling.
    -   `im-logic/internal/service/*.go`: Implement business logic for each gRPC service.
    -   `im-repo/internal/repository/*.go`: Implement data access logic using GORM and Redis.
    -   `im-task/internal/processor/fanout.go`: Implement message fan-out logic based on Kafka messages.
-   **Files to be Deleted**:
    -   `docs/`: The entire old documentation directory will be replaced by `new_docs/`.

[Functions]
Functions will be organized into services and repositories within their respective microservices.

-   **New Functions**:
    -   `im-logic/internal/service/auth_service.go`: `Register()`, `Login()`, `GenerateToken()`, `ParseToken()`.
    -   `im-logic/internal/service/message_service.go`: `SendMessage()`, `GetMessages()`.
    -   `im-repo/internal/repository/conversation_repo.go`: `CreateConversation()`, `GetConversations()`.
-   **Modified Functions**:
    -   All existing functions in `im-repo/internal/service/*.go` will be refactored to align with the new gRPC interfaces defined in `repo.proto`.
    -   `im-gateway/cmd/main.go`: Will be updated to initialize and run both HTTP and WebSocket servers.

[Classes]
Go does not have classes, but structs will be used to define services and repositories.

-   **New Structs**:
    -   `im-logic/internal/service/AuthService`: Implements the authentication gRPC service.
    -   `im-logic/internal/service/MessageService`: Implements the messaging gRPC service.
    -   `im-repo/internal/repository/UserRepository`: Implements user data access logic.
-   **Modified Structs**:
    -   Existing service and repository structs will be updated to implement the new gRPC interfaces and align with the refactored architecture.

[Dependencies]
The primary dependencies (`gin`, `gorm`, `grpc`, `kafka-go`, `redis`) are already present in `go.mod`. No major new dependencies are required.

-   We will ensure all microservices use consistent versions of shared dependencies like `gRPC` and `Protobuf`.

[Testing]
Each microservice will have its own set of unit and integration tests.

-   **Test File Requirements**:
    -   Unit tests will be created for each service and repository (e.g., `im-logic/internal/service/auth_service_test.go`).
    -   Integration tests will be added to verify the interaction between services (e.g., `im-gateway` calling `im-logic`).
-   **Validation Strategies**:
    -   Use mock implementations for dependencies in unit tests.
    -   Use `docker-compose` to spin up dependent services (DB, Kafka, Redis) for integration tests.

[Implementation Order]
The implementation will proceed in a logical sequence, starting from the data layer and moving up to the API gateway.

1.  **Define Contracts**: Finalize all `.proto` files for gRPC services (`im_logic.proto`, `im_repo.proto`).
2.  **Generate Code**: Generate gRPC server and client code from the `.proto` files.
3.  **Implement `im-repo`**: Implement the data access logic and gRPC services in the `im-repo` microservice.
4.  **Implement `im-logic`**: Implement the core business logic and gRPC services in the `im-logic` microservice, which calls `im-repo`.
5.  **Implement `im-task`**: Implement the Kafka consumer in `im-task` to handle asynchronous message processing.
6.  **Implement `im-gateway`**: Implement the HTTP/WebSocket handlers in `im-gateway`, which call `im-logic`.
7.  **Create Documentation**: Write the Markdown documentation in the `new_docs/` directory.
8.  **Update Deployment**: Update `docker-compose.yml` files to reflect any configuration changes.
9.  **Testing**: Write and run unit and integration tests for all components.
