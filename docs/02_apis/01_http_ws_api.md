# GoChat HTTP 和 WebSocket API

本文档定义了 GoChat 系统的 RESTful API 和 WebSocket 协议。所有客户端应用程序（Web、移动端）都应遵循这些规范。

## 1. 通用信息

-   **基础 URL**: `/api`
-   **WebSocket URL**: `/ws`
-   **认证**: 所有受保护的端点都需要在 `Authorization` 头中提供 `Bearer Token`。
    -   `Authorization: Bearer <jwt_token>`
-   **内容类型**: `application/json`

## 2. 通用响应格式

### 成功响应

```json
{
  "success": true,
  "message": "操作成功",
  "data": {}
}
```

### 错误响应

```json
{
  "success": false,
  "message": "错误描述",
  "code": "ERROR_CODE"
}
```

## 3. 认证 API (`/auth`)

### `POST /auth/register`

-   **描述**: 注册新用户。
-   **请求体**:
    ```json
    {
      "username": "string",
      "password": "string"
    }
    ```
-   **响应**: 返回新用户的信息。

### `POST /auth/login`

-   **描述**: 登录已注册用户。
-   **请求体**:
    ```json
    {
      "username": "string",
      "password": "string"
    }
    ```
-   **响应**: 返回 JWT 令牌和用户信息。

### `POST /auth/guest`

-   **描述**: 登录访客用户。
-   **请求体**:
    ```json
    {
      "guestName": "string" // 可选，如果未提供则生成随机名称
    }
    ```
-   **响应**: 返回 JWT 令牌和访客用户信息。

### `POST /auth/logout`

-   **描述**: 登出当前用户。
-   **认证**: 必需。
-   **响应**: 确认成功登出。

## 4. 用户 API (`/users`)

### `GET /users/info`

-   **描述**: 获取当前已认证用户的个人资料。
-   **认证**: 必需。
-   **响应**: 返回用户的个人资料信息。

## 5. 会话 API (`/conversations`)

### `GET /conversations`

-   **描述**: 获取当前用户的会话列表。
-   **认证**: 必需。
-   **响应**: 会话对象列表。

### `POST /conversations`

-   **描述**: 创建新的私人（一对一）会话。
-   **认证**: 必需。
-   **请求体**:
    ```json
    {
      "targetUserId": "string"
    }
    ```
-   **响应**: 新创建的会话对象。

### `GET /conversations/{conversationId}/messages`

-   **描述**: 获取会话的消息历史记录。
-   **认证**: 必需。
-   **查询参数**: `page`, `size`。
-   **响应**: 分页的消息列表。

### `PUT /conversations/{conversationId}/read`

-   **描述**: 将会话中的所有消息标记为已读。
-   **认证**: 必需。
-   **响应**: 成功确认。

## 6. 群组 API (`/groups`)

### `POST /groups`

-   **描述**: 创建新的群组聊天。
-   **认证**: 必需。
-   **请求体**:
    ```json
    {
      "groupName": "string",
      "members": ["userId1", "userId2"]
    }
    ```
-   **响应**: 新创建的群组和会话对象。

### `GET /groups/{groupId}`

-   **描述**: 获取群组的详细信息。
-   **认证**: 必需。
-   **响应**: 群组信息和成员列表。

## 7. WebSocket 协议

### 连接

-   **URL**: `ws://<host>/ws?token={jwt_token}`
-   JWT 令牌作为查询参数传递以进行认证。

### 消息类型（客户端 -> 服务器）

客户端通过 WebSocket 发送的事件都应有明确的 Go struct 定义。

-   **发送消息 (`send-message`)**:
    ```go
    // 示例: im-gateway/internal/ws/models.go
    type SendMessagePayload struct {
        ConversationID string `json:"conversationId"`
        Content        string `json:"content"`
        MessageType    string `json:"messageType"`
        TempMessageID  string `json:"tempMessageId"` // 客户端生成的临时ID
    }
    ```
    *当 `im-gateway` 收到此事件后，它会调用 `im-logic` 的 `SendMessage` gRPC 方法，而**不是**直接生产到 Kafka。*

-   **心跳 (`ping`)**:
    ```json
    { "type": "ping" }
    ```

### 消息类型（服务器 -> 客户端）

-   **新消息**:
    ```json
    {
      "type": "new-message",
      "data": { /* 消息对象 */ }
    }
    ```
-   **消息确认**:
    ```json
    {
      "type": "message-ack",
      "data": {
        "tempMessageId": "string", // 客户端的临时 ID
        "messageId": "string"      // 服务器生成的最终 ID
      }
    }
    ```
-   **心跳响应**:
    ```json
    { "type": "pong" }
    ```
-   **错误通知**:
    ```json
    {
      "type": "error",
      "data": {
        "message": "错误描述",
        "code": "ERROR_CODE"
      }
    }