# GoChat 核心认证与同步流程

本文档详细描述了用户从注册、登录到获取初始会话数据的完整操作流程，并明确了各微服务之间的交互、数据格式和数据库操作。

---

## 1. 用户注册流程 (正式用户)

**目标**: 创建一个新的正式用户账户。

1.  **用户输入 (HTTP 请求)**
    *   **端点**: `POST /auth/register`
    *   **处理服务**: `im-gateway`
    *   **输入格式** (JSON Body):
        ```json
        {
          "username": "testuser",
          "password": "password123"
        }
        ```

2.  **处理者: `im-gateway` 服务**
    *   **接收请求**: `im-gateway` 接收到 HTTP POST 请求。
    *   **核心逻辑**: 将 HTTP 请求转换为 gRPC 请求，调用 `im-logic` 的 `AuthService.Register`。
    *   **调用格式** ([`RegisterRequest`](../../api/proto/im_logic/v1/auth.proto)):
        ```protobuf
        message RegisterRequest {
          string username = 1; // "testuser"
          string password = 2; // "password123"
        }
        ```

3.  **处理者: `im-logic` 服务**
    *   **接收请求**: `AuthService` 接收到来自 `im-gateway` 的 gRPC 请求。
    *   **核心逻辑**:
        1.  **参数校验**: 验证 `username` 和 `password` 的格式与长度。
        2.  **密码哈希**: 对明文密码进行哈希处理。
        3.  **调用下游服务**: 调用 `im-repo` 的 `UserService.CreateUser` RPC。
    *   **调用格式** ([`CreateUserRequest`](../../api/proto/im_repo/v1/user.proto)):
        ```protobuf
        message CreateUserRequest {
          string username = 1;      // "testuser"
          string password_hash = 2; // "hashed_password_string" (im-logic 计算后)
          bool is_guest = 3;        // false
        }
        ```

4.  **处理者: `im-repo` 服务**
    *   **接收请求**: `UserService` 接收到 `CreateUserRequest`。
    *   **查表 (数据库操作)**:
        1.  **检查用户名是否存在**:
            ```sql
            SELECT id FROM users WHERE username = 'testuser' LIMIT 1;
            ```
        2.  **插入新用户**: 如果用户名不存在，则执行插入。
            ```sql
            INSERT INTO users (id, username, password_hash, is_guest, status, created_at, updated_at)
            VALUES (1234567890, 'testuser', 'hashed_password_string', false, 0, NOW(), NOW());
            -- id 由雪花算法在 im-repo 服务中生成
            ```
    *   **处理之后**:
        *   向 `im-logic` 返回包含新用户信息的 [`CreateUserResponse`](../../api/proto/im_repo/v1/user.proto:52)。如果用户名已存在，则返回错误。

5.  **返回给客户端 (HTTP 响应)**
    *   `im-logic` 将 `im-repo` 的响应包装成 `RegisterResponse` gRPC 响应返回给 `im-gateway`。
    *   `im-gateway` 接收到 gRPC 成功响应后，向客户端返回 `200 OK` 的 HTTP 响应。
    *   **响应格式** (JSON Body):
        ```json
        {
          "success": true,
          "message": "注册成功",
          "data": {
            "id": "1234567890",
            "username": "testuser"
          }
        }
        ```

---

## 2. 用户登录与认证流程 (包含游客)

**目标**: 验证用户身份，返回用于后续操作的 `access_token`。

1.  **用户输入 (HTTP 请求)**
    *   **端点**:
        *   正式用户: `POST /auth/login`
        *   游客: `POST /auth/guest`
    *   **处理服务**: `im-gateway`
    *   **输入格式** (JSON Body):
        ```json
        // 正式用户
        {
          "username": "testuser",
          "password": "password123"
        }

        // 游客 (无请求体)
        {}
        ```

2.  **处理者: `im-gateway` 服务**
    *   **接收请求**: `im-gateway` 接收 HTTP 请求。
    *   **核心逻辑**: 将 HTTP 请求转换为对应的 gRPC 请求，调用 `im-logic` 的 `AuthService`。
    *   **调用格式**: `Login(LoginRequest)` 或 `GuestLogin(GuestLoginRequest)`。

3.  **处理者: `im-logic` 服务**
    *   **接收请求**: `AuthService` 接收 gRPC 请求。
    *   **核心逻辑 (正式用户)**:
        1.  调用 `im-repo` 的 `UserService.GetUserByUsername` 获取用户信息。
        2.  调用 `im-repo` 的 `UserService.VerifyPassword` 验证密码。
        3.  验证成功后，生成 JWT `access_token` 和 `refresh_token`。
    *   **核心逻辑 (游客)**:
        1.  调用 `im-repo` 的 `UserService.CreateUser` 创建一个 `is_guest=true` 的临时用户。`im-repo` 内部会为其生成一个唯一的 `username` (例如: `guest_xxxx`)。
        2.  生成 JWT `access_token` 和 `refresh_token`。
        3.  **自动加入世界聊天室**: 调用 `im-repo` 的 `ConversationService.AddConversationMember`，将游客加入到固定的世界聊天室会话中。
    *   **调用格式 (示例)**:
        ```protobuf
        // 验证密码
        message VerifyPasswordRequest {
          string user_id = 1;    // 从 GetUserByUsername 获得
          string password = 2; // "password123"
        }
        ```

4.  **处理者: `im-repo` 服务**
    *   **查表 (正式用户登录)**:
        ```sql
        -- 1. 根据用户名查找用户
        SELECT id, password_hash FROM users WHERE username = 'testuser';
        -- 2. 在服务层代码中比对 hash(password) 与 password_hash
        ```
    *   **查表 (游客登录)**:
        ```sql
        -- 1. 创建游客用户 (username 由 im-repo 服务层生成)
        INSERT INTO users (id, username, password_hash, is_guest, status, ...)
        VALUES (..., 'guest_a1b2c3d4', 'N/A', true, 0, ...);
        
        -- 2. 将游客加入世界聊天室 (由 AddConversationMember RPC 封装)
        INSERT INTO conversation_members (conversation_id, user_id, role, ...)
        VALUES ('world_chat_room', [new_guest_user_id], 1, ...);
        ```

5.  **返回给客户端 (HTTP 响应)**
    *   `im-logic` 将包含 token 和用户信息的 `LoginResponse` gRPC 响应返回给 `im-gateway`。
    *   `im-gateway` 接收到 gRPC 成功响应后，向客户端返回 `200 OK` 的 HTTP 响应。
    *   **响应格式** (JSON Body):
        ```json
        {
          "success": true,
          "message": "登录成功",
          "data": {
            "accessToken": "jwt_access_token_string",
            "refreshToken": "jwt_refresh_token_string",
            "expiresIn": 1668888888,
            "user": {
              "id": "1234567890",
              "username": "testuser"
            }
          }
        }
        ```

---

## 3. 登录后拉取初始会话列表

**目标**: 用户登录成功后，客户端拉取其所有会话的列表及最新状态。

1.  **用户输入 (HTTP 请求)**
    *   **端点**: `GET /conversations`
    *   **处理服务**: `im-gateway`
    *   **认证**: `Authorization: Bearer <jwt_token>`
    *   **输入格式** (Query Params):
        ```
        ?page=1&size=20
        ```

2.  **处理者: `im-gateway` 服务**
    *   **接收请求**: `im-gateway` 接收 HTTP GET 请求。
    *   **核心逻辑**:
        1.  **认证**: 从 `Authorization` 头中解析 `access_token`，调用 `im-logic` 的 `AuthService.ValidateToken` 进行验证，并获取 `user_id`。
        2.  **调用下游服务**: 调用 `im-logic` 的 `ConversationService.GetConversations` gRPC 方法。
    *   **调用格式** ([`GetConversationsRequest`](../../api/proto/im_logic/v1/conversation.proto)):
        ```protobuf
        // message GetConversationsRequest in im_logic
        message GetConversationsRequest {
          // user_id 从 token 中解析得到，由 gateway 填充
          string user_id = 1; 
          int32 page = 2; // 1
          int32 size = 3; // 20
        }
        ```

3.  **处理者: `im-logic` 服务**
    *   **接收请求**: `ConversationService` 接收来自 `im-gateway` 的 gRPC 请求。
    *   **核心逻辑**: 调用 `im-repo` 的 `ConversationService.GetUserConversations` RPC，透传分页参数和 `user_id`。
    *   **调用格式** ([`GetUserConversationsRequest`](../../api/proto/im_repo/v1/conversation.proto)):
        ```protobuf
        message GetUserConversationsRequest {
          string user_id = 1; // 从 token 中解析出的用户 ID
          int32 limit = 20;
          int32 offset = 0;
        }
        ```

4.  **处理者: `im-repo` 服务**
    *   **接收请求**: `ConversationService` 接收请求。
    *   **查表 (数据库操作)**:
        1.  **获取会话 ID 列表**:
            ```sql
            -- 从新表 conversation_members 中高效查找
            SELECT conversation_id FROM conversation_members WHERE user_id = ? ORDER BY updated_at DESC LIMIT 20 OFFSET 0;
            ```
        2.  **查出来之后**: 得到 `conversation_id` 列表，例如 `['conv1', 'conv2', 'world_chat_room']`。
        3.  **批量获取会话详情**:
            *   **批量获取未读数**:
                ```sql
                -- 伪代码，实际通过循环或 JOIN 实现
                SELECT conversation_id, COUNT(*) 
                FROM messages 
                WHERE conversation_id IN ('conv1', 'conv2', ...) 
                  AND seq_id > (SELECT last_read_seq_id FROM user_read_pointers WHERE user_id = ? AND conversation_id = messages.conversation_id)
                GROUP BY conversation_id;
                ```
            *   **批量获取最后一条消息**:
                ```sql
                -- 使用窗口函数高效获取每组的最后一条消息
                SELECT * FROM (
                  SELECT *, ROW_NUMBER() OVER(PARTITION BY conversation_id ORDER BY seq_id DESC) as rn
                  FROM messages
                  WHERE conversation_id IN ('conv1', 'conv2', ...)
                ) tmp WHERE rn = 1;
                ```
            *   **批量获取会话元信息** (如群名、群头像):
                ```sql
                SELECT id, name, avatar_url FROM conversations WHERE id IN ('conv1', 'conv2', ...);
                -- 对于单聊，需要额外查询对方用户的信息
                ```
    *   **处理之后**:
        *   在 `im-repo` 服务内存中，将上述查询结果聚合成一个包含完整信息的会话对象列表。
        *   向 `im-logic` 返回 `GetUserConversationsResponse`。

5.  **返回给客户端 (HTTP 响应)**
    *   `im-logic` 将从 `im-repo` 获取到的会话列表 gRPC 响应返回给 `im-gateway`。
    *   `im-gateway` 接收到 gRPC 成功响应后，向客户端返回 `200 OK` 的 HTTP 响应。
    *   **响应格式** (JSON Body):
        ```json
        {
          "success": true,
          "message": "获取成功",
          "data": {
            "conversations": [
              {
                "id": "conv1",
                "type": 2,
                "name": "技术交流群",
                "avatarUrl": "http://...",
                "lastMessage": { "...": "..." },
                "unreadCount": 5,
                "updatedAt": 1668888999
              }
            ],
            "hasMore": true
          }
        }