# GoChat HTTP 和 WebSocket API

本文档提供了 GoChat 系统的 HTTP API 和 WebSocket 协议的概述。详细的 HTTP API 规范请参考 `00_openapi.yaml` 文件。

## 1. 通用信息

-   **基础 URL**: `/api`
-   **WebSocket URL**: `/ws`
-   **认证**: 所有受保护的端点都需要在 `Authorization` 头中提供 `Bearer Token`
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

## 3. HTTP API 端点概览

### 认证 API (`/auth`)
- `POST /auth/register` - 注册新用户
- `POST /auth/login` - 用户登录
- `POST /auth/guest` - 游客登录
- `POST /auth/logout` - 用户登出

### 用户 API (`/users`)
- `GET /users/profile` - 获取当前用户信息
- `GET /users/search` - 搜索用户

### 会话 API (`/conversations`)
- `GET /conversations` - 获取会话列表
- `POST /conversations` - 创建会话（单聊或群聊）
- `GET /conversations/{conversationId}/messages` - 获取消息历史
- `POST /conversations/{conversationId}/messages` - 发送消息
- `PUT /conversations/{conversationId}/read` - 标记已读

> **注意**: 详细的请求/响应格式、数据模型和错误处理信息请参考 `00_openapi.yaml` 文件。

## 4. WebSocket 实时通信协议

### 4.1 连接建立

-   **连接 URL**: `ws://<host>/ws?token={jwt_token}`
-   **认证方式**: JWT 令牌作为查询参数传递
-   **连接限制**: 每个用户只能同时保持一个 WebSocket 连接

**连接示例**:
```javascript
// 前端连接示例
const ws = new WebSocket(`ws://localhost:8080/ws?token=${jwtToken}`);
```

### 4.2 消息格式

所有 WebSocket 消息都使用 JSON 格式，遵循统一的结构：

```json
{
  "type": "message_type",
  "data": { /* 具体数据 */ },
  "timestamp": 1640995200
}
```

### 4.3 客户端 → 服务器 消息类型

#### 4.3.1 发送消息 (`send-message`)

**用途**: 向指定会话发送新消息
**数据结构**:
```json
{
  "type": "send-message",
  "data": {
    "conversationId": "string",
    "content": "string",
    "messageType": 1,
    "client_msg_id": "string"
  }
}
```

**字段说明**:
- `conversationId`: 目标会话ID
- `content`: 消息内容
- `messageType`: 消息类型（1=文本，2=图片，3=文件等）
- `client_msg_id`: 客户端生成的临时消息ID，用于后续确认

**处理流程**:
1. `im-gateway` 接收消息
2. 调用 `im-logic` 的 `SendMessage` gRPC 方法
3. `im-logic` 处理业务逻辑并持久化到 `im-repo`
4. 通过 Kafka 下游队列推送给目标用户

#### 4.3.2 心跳保活 (`ping`)

**用途**: 保持连接活跃，检测连接状态
**数据结构**:
```json
{
  "type": "ping"
}
```

**响应**: 服务器返回 `pong` 消息
**间隔**: 建议每 30 秒发送一次

#### 4.3.3 标记已读 (`mark-read`)

**用途**: 将会话中的消息标记为已读
**数据结构**:
```json
{
  "type": "mark-read",
  "data": {
    "conversationId": "string",
    "seqId": 12345
  }
}
```

### 4.4 服务器 → 客户端 消息类型

#### 4.4.1 新消息通知 (`new-message`)

**用途**: 推送新消息到客户端
**数据结构**:
```json
{
  "type": "new-message",
  "data": {
    "id": "string",
    "conversationId": "string",
    "sender": {
      "id": "string",
      "username": "string",
      "avatarUrl": "string"
    },
    "content": "string",
    "messageType": 1,
    "seqId": 123,
    "createdAt": 1640995200
  }
}
```

#### 4.4.2 消息确认 (`message-ack`)

**用途**: 确认消息已成功处理
**数据结构**:
```json
{
  "type": "message-ack",
  "data": {
    "client_msg_id": "string",
    "messageId": "string",
    "status": "success"
  }
}
```

#### 4.4.3 在线状态更新 (`online-status`)

**用途**: 通知用户好友在线状态变化
**数据结构**:
```json
{
  "type": "online-status",
  "data": {
    "userId": "string",
    "status": "online|offline"
  }
}
```

#### 4.4.4 心跳响应 (`pong`)

**用途**: 响应客户端心跳
**数据结构**:
```json
{
  "type": "pong",
  "timestamp": 1640995200
}
```

#### 4.4.5 错误通知 (`error`)

**用途**: 通知客户端发生的错误
**数据结构**:
```json
{
  "type": "error",
  "data": {
    "code": "ERROR_CODE",
    "message": "错误描述",
    "details": {}
  }
}
```

### 4.5 错误处理

#### 常见错误代码
- `AUTH_FAILED`: 认证失败
- `INVALID_MESSAGE`: 消息格式错误
- `CONVERSATION_NOT_FOUND`: 会话不存在
- `MESSAGE_TOO_LARGE`: 消息过大
- `RATE_LIMITED`: 发送频率超限

#### 重连策略
1. 连接断开时，客户端应自动重连
2. 重连间隔采用指数退避算法（1s, 2s, 4s, 8s, 15s, 30s）
3. 最大重连间隔为 30 秒
4. 重连成功后，客户端应重新获取未读消息

### 4.6 安全注意事项

- 所有 WebSocket 连接必须通过 HTTPS/WSS 加密
- JWT 令牌有过期时间，过期后需要重新获取
- 服务器会对消息频率进行限制，防止恶意攻击
- 敏感信息不应在消息内容中传输