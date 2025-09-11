# GoChat 内部 gRPC 服务

本文档概述了用于微服务之间内部通信的 gRPC 服务。有关详细的消息和服务定义，请参考相应的 `.proto` 文件。

## 1. 概述

GoChat 的内部通信通过 gRPC 处理，它提供高性能、强类型和语言无关的 RPC 框架。这确保了微服务之间高效可靠的通信。

-   **`im-logic` 服务**: 暴露业务逻辑功能。
-   **`im-repo` 服务**: 暴露数据持久化功能。
-   **`im-gateway` 服务**: 暴露一个内部 gRPC 服务，用于接收来自后端的推送指令。

## 2. `im-gateway` 内部服务

### `PusherService`

-   **描述**: 提供一个内部 gRPC 入口，允许后端服务（如 `im-pusher`）向一个或多个在线用户推送实时信令或通知。
-   **主要 RPC**:
    -   `PushToUsers`: 将一条消息或信令推送给指定的一批用户。`im-gateway` 会查询这些用户当前连接在哪个 `gateway` 实例上，并将请求转发给对应的实例进行推送。

## 3. `im-logic` 服务

`im-logic` 微服务暴露了几个 gRPC 服务，它们封装了应用程序的核心业务逻辑。

-   **Proto 文件位置**: `api/proto/im_logic/v1/`

### `AuthService`

-   **描述**: 处理所有用户认证和令牌管理逻辑。
-   **Proto 文件**: [`auth.proto`](../../../api/proto/im_logic/v1/auth.proto)
-   **主要 RPC**:
    -   `Login`: 使用用户名和密码认证用户。
    -   `Register`: 创建新用户账户。
    -   `GuestLogin`: 创建临时访客账户。
    -   `ValidateToken`: 验证 JWT 访问令牌。

### `ConversationService`

-   **描述**: 管理会话相关逻辑，例如获取会话列表和消息。
-   **Proto 文件**: [`conversation.proto`](../../../api/proto/im_logic/v1/conversation.proto)
-   **主要 RPC**:
    -   `GetConversations`: 检索用户的会话列表（传统方式，可能存在N+1查询）。
    -   `GetConversationsOptimized`: **[推荐]** 优化的会话列表查询，一次性获取所有必要信息，避免N+1问题。
    -   `CreateConversation`: 创建新的私人会话。
    -   `GetMessages`: 获取会话的消息历史记录。
    -   `MarkAsRead`: 更新用户在会话中的已读指针。

### `MessageService`

-   **描述**: 处理发送消息的逻辑。
-   **Proto 文件**: [`message.proto`](../../../api/proto/im_logic/v1/message.proto)
-   **主要 RPC**:
    -   `SendMessage`: 处理外发消息，保存它并触发消息扩散。

## 3. `im-repo` 服务

`im-repo` 微服务暴露了用于数据访问的 gRPC 服务，从业务逻辑层抽象了数据库和缓存。

-   **Proto 文件位置**: `api/proto/im_repo/v1/`

### `UserService`

-   **描述**: 为用户数据提供 CRUD 操作。
-   **Proto 文件**: [`user.proto`](../../../api/proto/im_repo/v1/user.proto)
-   **主要 RPC**:
    -   `CreateUser`: 在数据库中插入新用户记录。
    -   `GetUser`: 通过 ID 检索用户。
    -   `GetUserByUsername`: 通过用户名检索用户。
    -   `VerifyPassword`: 验证用户的密码哈希。

### `ConversationService`

-   **描述**: 为会话及成员关系提供统一的数据访问操作。
-   **Proto 文件**: [`conversation.proto`](../../../api/proto/im_repo/v1/conversation.proto)
-   **主要 RPC**:
    -   `CreateConversation`: 创建新的会话记录。
    -   `GetUserConversations`: 检索用户所属的会话 ID。
    -   `GetUserConversationsWithDetails`: **[核心优化]** 一次性查询获取用户会话的完整信息，包括会话详情、最后消息、未读数等，彻底解决N+1查询问题。
    -   `UpdateReadPointer`: 在数据库中更新用户的已读进度。
    -   `BatchGetUnreadCounts`: 批量获取多个会话的未读消息数。
    -   `AddConversationMember`: 向会话中添加一个或多个成员。
    -   `RemoveConversationMember`: 从会话中移除成员。
    -   `GetConversationMembers`: 获取会话的成员列表。

### `MessageService`

-   **描述**: 为消息提供数据访问操作。
-   **Proto 文件**: [`message.proto`](../../../api/proto/im_repo/v1/message.proto)
-   **主要 RPC**:
    -   `SaveMessage`: 将消息保存到数据库。
    -   `GetConversationMessages`: 检索会话的消息列表。

### `OnlineStatusService`

-   **描述**: 管理用户在线状态，主要使用 Redis。
-   **Proto 文件**: [`online_status.proto`](../../../api/proto/im_repo/v1/online_status.proto)
-   **主要 RPC**:
    -   `SetUserOnline`, `SetUserOffline`: 更新用户的在线状态。
    -   `GetUserOnlineStatus`: 检索用户的在线状态。
