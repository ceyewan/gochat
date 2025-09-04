# GoChat API 统一规范

**版本**: 2.0
**日期**: 2025-08-26

## 1. 概述

本文档是 GoChat 即时通讯系统的 API 统一规范，旨在为前后端开发、测试和集成提供清晰、一致的指导。它整合了系统所有对外和对内的接口协议，包括：

-   **外部接口**: 供客户端（Web/App）使用的 **RESTful API** 和 **WebSocket** 实时通信协议。
-   **内部接口**: 微服务之间通信使用的 **gRPC** 协议和 **Kafka** 消息格式。

## 2. 外部接口 (客户端 <-> im-gateway)

### 2.1 基础配置

-   **Base URL**: `http://localhost:8080/api` (由 `im-gateway` 服务提供)
-   **WebSocket URL**: `ws://localhost:8080/ws` (由 `im-gateway` 服务提供)
-   **Content-Type**: `application/json`
-   **认证方式**: Bearer Token (JWT)
-   **API版本**: v1
-   **字符编码**: UTF-8

### 2.2 通用响应格式

#### 成功响应
```json
{
  "success": true,
  "message": "操作成功",
  "data": {}
}
```

#### 错误响应
```json
{
  "success": false,
  "message": "错误描述",
  "code": "ERROR_CODE"
}
```

### 2.3 RESTful API

#### 认证接口 (`/auth`)

-   **`POST /auth/login`**: 用户登录
-   **`POST /auth/register`**: 用户注册
-   **`POST /auth/guest`**: 游客登录
-   **`POST /auth/logout`**: 用户登出

#### 用户管理接口 (`/users`)

-   **`GET /users/info`**: 获取当前用户信息
-   **`GET /users/{username}`**: 搜索用户

#### 会话管理接口 (`/conversations`)

-   **`GET /conversations`**: 获取会话列表
-   **`GET /conversations/{conversationId}/messages`**: 获取会话历史消息
-   **`PUT /conversations/{conversationId}/read`**: 标记会话已读
-   **`POST /conversations`**: 创建私聊会话

#### 好友管理接口 (`/friends`)

-   **`POST /friends`**: 添加好友
-   **`GET /friends`**: 获取好友列表

#### 群聊管理接口 (`/groups`)

-   **`POST /groups`**: 创建群聊
-   **`GET /groups/{groupId}`**: 获取群聊信息

#### 消息管理接口 (`/messages`)

-   **`POST /messages`**: 发送消息（HTTP方式，主要用于测试或特殊场景）

#### 文件上传接口 (`/uploads`)

-   **`POST /uploads`**: 请求上传凭证（获取预签名 URL）

#### 全局搜索接口 (`/search`)

-   **`GET /search`**: 执行全文搜索，参数 `q` 为查询关键字。

#### 智能服务接口 (`/ai`)

-   **`POST /ai/conversations`**: 创建一个新的 AI 对话会话。
-   **`POST /ai/conversations/{conversationId}/messages`**: 向指定的 AI 会话发送消息。

#### 内容推荐接口 (`/recommendations`)

-   **`GET /recommendations/users`**: 获取“可能认识的人”推荐列表。
-   **`GET /recommendations/groups`**: 获取“可能感兴趣的群”推荐列表。

### 2.4 WebSocket 实时通信协议

#### 连接建立
**URL**: `ws://localhost:8080/ws?token={jwt_token}`

#### 消息类型

-   **客户端发送**:
    -   `send-message`: 发送聊天消息
    -   `ping`: 心跳检测

-   **服务器推送**:
    -   `message-ack`: 确认收到客户端消息
    -   `new-message`: 推送新消息
    -   `friend-online`: 好友在线状态变更
    -   `pong`: 心跳响应

## 3. 内部接口 (服务间通信)

### 3.1 gRPC 接口

服务间的同步调用统一使用 gRPC。

#### 通用规范
- **命名**: 服务、方法、消息使用 `PascalCase`，字段使用 `snake_case`。
- **通用字段**: 所有请求和响应都应包含 `RequestHeader` 和 `ResponseHeader`，用于链路追踪和统一错误处理。

#### `im-logic` 对外接口
- **`AuthService`**: 处理用户认证。
- **`ConversationService`**: 处理会话管理。
- **`MessageService`**: 处理消息相关操作。
- **`UserService`**: 处理用户管理。
- **`GroupService`**: 处理群组管理。

#### `im-repo` 对外接口
- **`UserRepo`**: 封装用户数据的增删改查和缓存。
- **`MessageRepo`**: 封装消息数据的增删改查和缓存。
- **`GroupRepo`**: 封装群组数据的增删改查和缓存。

### 3.2 Kafka 消息格式

服务间的异步通信统一使用 Kafka。所有消息体都使用 Protobuf 进行序列化。

#### 通用消息格式
```protobuf
message KafkaMessage {
  string trace_id = 1;
  map<string, string> headers = 2; // 存放元数据，如 task_type
  google.protobuf.Any body = 3;    // 具体的业务消息体
}
```

#### 核心 Topic 与消息体

-   **`im-upstream-topic`**: 上行消息（客户端 -> 服务端）
    -   **Body**: `UpstreamMessage`
-   **`im-downstream-topic-{gateway_id}`**: 下行消息（服务端 -> 客户端）
    -   **Body**: `DownstreamMessage`
-   **`im-task-topic`**: 异步任务消息
    -   **Body**: `TaskMessage` (内含具体的任务 payload，如 `LargeGroupFanoutTask`)

## 4. 核心数据模型

以下是在各层之间流转的核心数据模型定义。

### User（用户）
```json
{
  "userId": "string",
  "username": "string",
  "avatar": "string",
  "isGuest": "boolean"
}
```

### Conversation（会话）
```json
{
  "conversationId": "string",
  "type": "single|group|world|ai",
  "target": { ... },
  "lastMessage": "string",
  "lastMessageTime": "string",
  "unreadCount": "number"
}
```

### Message（消息）
```json
{
  "messageId": "string",
  "conversationId": "string",
  "senderId": "string",
  "senderName": "string",
  "type": "text|image|file|system",
  "content": "string",
  "extra": {
    "url": "string (for image/file type)",
    "fileName": "string (for file type)",
    "fileSize": "number (for file type)"
  },
  "sendTime": "string",
  "seqId": "number"
}
