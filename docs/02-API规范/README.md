# API 规范文档

## 概述

GoChat 系统提供多种 API 接口，包括 HTTP REST API、WebSocket API 和 gRPC API。本文档详细描述了各个 API 的规范。

## API 版本管理

### 版本策略

- **当前版本**: v1
- **版本格式**: `v{major}.{minor}.{patch}`
- **兼容性**: 主版本号不保证向后兼容，次版本号保证向后兼容

### 版本路径

```
HTTP API: /api/v1/
gRPC: 使用 proto 文件版本管理
```

## HTTP REST API

### 基础规范

#### 请求格式

- **Content-Type**: `application/json`
- **Accept**: `application/json`
- **字符编码**: `UTF-8`

#### 响应格式

```json
{
  "code": 0,
  "message": "success",
  "data": {},
  "timestamp": 1640995200000
}
```

#### 错误响应

```json
{
  "code": 1001,
  "message": "参数错误",
  "details": {
    "field": "username",
    "error": "required"
  },
  "timestamp": 1640995200000
}
```

### 错误码定义

| 错误码 | 错误信息 | 说明 |
|--------|----------|------|
| 0 | success | 成功 |
| 1001 | 参数错误 | 请求参数错误 |
| 1002 | 未授权 | 需要登录 |
| 1003 | 禁止访问 | 权限不足 |
| 1004 | 资源不存在 | 请求的资源不存在 |
| 1005 | 内部错误 | 服务器内部错误 |
| 1006 | 重复操作 | 重复请求 |
| 1007 | 频率限制 | 请求过于频繁 |

### 认证方式

#### JWT Token

```http
Authorization: Bearer <jwt_token>
```

#### Token 结构

```json
{
  "sub": "user_id",
  "iat": 1640995200,
  "exp": 1641081600,
  "type": "access"
}
```

## API 接口详情

### 1. 认证相关

#### 登录

```http
POST /api/v1/auth/login
```

**请求参数**:

```json
{
  "username": "string",
  "password": "string",
  "device_id": "string"
}
```

**响应示例**:

```json
{
  "code": 0,
  "message": "success",
  "data": {
    "access_token": "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9...",
    "refresh_token": "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9...",
    "expires_in": 7200,
    "user_info": {
      "user_id": "user_123",
      "username": "john_doe",
      "nickname": "John Doe",
      "avatar": "https://example.com/avatar.jpg"
    }
  }
}
```

#### 登出

```http
POST /api/v1/auth/logout
```

**请求参数**:

```json
{
  "device_id": "string"
}
```

#### 刷新 Token

```http
POST /api/v1/auth/refresh
```

**请求参数**:

```json
{
  "refresh_token": "string"
}
```

### 2. 用户管理

#### 注册用户

```http
POST /api/v1/users/register
```

**请求参数**:

```json
{
  "username": "string",
  "password": "string",
  "email": "string",
  "nickname": "string"
}
```

#### 获取用户信息

```http
GET /api/v1/users/profile
```

**响应示例**:

```json
{
  "code": 0,
  "message": "success",
  "data": {
    "user_id": "user_123",
    "username": "john_doe",
    "email": "john@example.com",
    "nickname": "John Doe",
    "avatar": "https://example.com/avatar.jpg",
    "status": 1,
    "created_at": "2024-01-01T00:00:00Z",
    "updated_at": "2024-01-01T00:00:00Z"
  }
}
```

#### 更新用户信息

```http
PUT /api/v1/users/profile
```

**请求参数**:

```json
{
  "nickname": "string",
  "avatar": "string",
  "status": 1
}
```

#### 搜索用户

```http
GET /api/v1/users/search
```

**查询参数**:

- `q`: 搜索关键词
- `page`: 页码 (默认: 1)
- `limit`: 每页数量 (默认: 20, 最大: 100)

**响应示例**:

```json
{
  "code": 0,
  "message": "success",
  "data": {
    "total": 50,
    "page": 1,
    "limit": 20,
    "users": [
      {
        "user_id": "user_123",
        "username": "john_doe",
        "nickname": "John Doe",
        "avatar": "https://example.com/avatar.jpg"
      }
    ]
  }
}
```

### 3. 消息管理

#### 发送消息

```http
POST /api/v1/messages/send
```

**请求参数**:

```json
{
  "conversation_id": "conv_123",
  "to_user_id": "user_456",
  "content": "Hello World",
  "message_type": "text",
  "metadata": {}
}
```

**响应示例**:

```json
{
  "code": 0,
  "message": "success",
  "data": {
    "message_id": "msg_123",
    "conversation_id": "conv_123",
    "created_at": "2024-01-01T00:00:00Z"
  }
}
```

#### 获取消息历史

```http
GET /api/v1/messages/history
```

**查询参数**:

- `conversation_id`: 会话ID
- `before_message_id`: 分页锚点 (可选)
- `limit`: 每页数量 (默认: 50, 最大: 200)

**响应示例**:

```json
{
  "code": 0,
  "message": "success",
  "data": {
    "conversation_id": "conv_123",
    "messages": [
      {
        "message_id": "msg_123",
        "from_user_id": "user_123",
        "to_user_id": "user_456",
        "content": "Hello World",
        "message_type": "text",
        "status": 1,
        "created_at": "2024-01-01T00:00:00Z"
      }
    ],
    "has_more": false
  }
}
```

#### 获取未读消息

```http
GET /api/v1/messages/unread
```

**响应示例**:

```json
{
  "code": 0,
  "message": "success",
  "data": {
    "total_count": 5,
    "conversations": [
      {
        "conversation_id": "conv_123",
        "unread_count": 3,
        "last_message": {
          "message_id": "msg_123",
          "content": "Hello World",
          "created_at": "2024-01-01T00:00:00Z"
        }
      }
    ]
  }
}
```

#### 标记消息已读

```http
PUT /api/v1/messages/read
```

**请求参数**:

```json
{
  "conversation_id": "conv_123",
  "message_id": "msg_123"
}
```

### 4. 会话管理

#### 获取会话列表

```http
GET /api/v1/conversations
```

**查询参数**:

- `page`: 页码 (默认: 1)
- `limit`: 每页数量 (默认: 20, 最大: 100)

**响应示例**:

```json
{
  "code": 0,
  "message": "success",
  "data": {
    "total": 10,
    "page": 1,
    "limit": 20,
    "conversations": [
      {
        "conversation_id": "conv_123",
        "type": "single",
        "name": "John Doe",
        "avatar": "https://example.com/avatar.jpg",
        "last_message": {
          "message_id": "msg_123",
          "content": "Hello World",
          "created_at": "2024-01-01T00:00:00Z"
        },
        "unread_count": 3,
        "updated_at": "2024-01-01T00:00:00Z"
      }
    ]
  }
}
```

#### 创建会话

```http
POST /api/v1/conversations/create
```

**请求参数**:

```json
{
  "type": "single",
  "user_ids": ["user_456"]
}
```

#### 更新会话

```http
PUT /api/v1/conversations/{conversation_id}
```

**请求参数**:

```json
{
  "name": "string",
  "avatar": "string"
}
```

#### 删除会话

```http
DELETE /api/v1/conversations/{conversation_id}
```

### 5. 群组管理

#### 创建群组

```http
POST /api/v1/groups/create
```

**请求参数**:

```json
{
  "name": "技术交流群",
  "description": "技术讨论和交流",
  "avatar": "https://example.com/avatar.jpg",
  "max_members": 200
}
```

**响应示例**:

```json
{
  "code": 0,
  "message": "success",
  "data": {
    "group_id": "group_123",
    "name": "技术交流群",
    "description": "技术讨论和交流",
    "avatar": "https://example.com/avatar.jpg",
    "owner_id": "user_123",
    "max_members": 200,
    "current_members": 1,
    "created_at": "2024-01-01T00:00:00Z"
  }
}
```

#### 获取群组信息

```http
GET /api/v1/groups/{group_id}
```

**响应示例**:

```json
{
  "code": 0,
  "message": "success",
  "data": {
    "group_id": "group_123",
    "name": "技术交流群",
    "description": "技术讨论和交流",
    "avatar": "https://example.com/avatar.jpg",
    "owner_id": "user_123",
    "max_members": 200,
    "current_members": 15,
    "created_at": "2024-01-01T00:00:00Z",
    "updated_at": "2024-01-01T00:00:00Z"
  }
}
```

#### 加入群组

```http
POST /api/v1/groups/{group_id}/join
```

#### 离开群组

```http
POST /api/v1/groups/{group_id}/leave
```

#### 获取群组成员

```http
GET /api/v1/groups/{group_id}/members
```

**查询参数**:

- `page`: 页码 (默认: 1)
- `limit`: 每页数量 (默认: 50, 最大: 200)

#### 添加群组成员

```http
POST /api/v1/groups/{group_id}/members
```

**请求参数**:

```json
{
  "user_ids": ["user_456", "user_789"]
}
```

#### 移除群组成员

```http
DELETE /api/v1/groups/{group_id}/members/{user_id}
```

#### 更新群组信息

```http
PUT /api/v1/groups/{group_id}
```

**请求参数**:

```json
{
  "name": "string",
  "description": "string",
  "avatar": "string",
  "max_members": 200
}
```

## WebSocket API

### 连接规范

#### 连接 URL

```javascript
ws://localhost:8080/ws?token={jwt_token}
```

#### 心跳机制

```json
// 客户端发送
{
  "type": "ping",
  "timestamp": 1640995200000
}

// 服务器响应
{
  "type": "pong",
  "timestamp": 1640995200000
}
```

### 消息格式

#### 发送消息

```json
{
  "type": "message",
  "data": {
    "conversation_id": "conv_123",
    "content": "Hello World",
    "message_type": "text",
    "metadata": {}
  },
  "message_id": "msg_client_123"
}
```

#### 接收消息

```json
{
  "type": "message",
  "data": {
    "message_id": "msg_123",
    "conversation_id": "conv_123",
    "from_user_id": "user_456",
    "content": "Hello World",
    "message_type": "text",
    "metadata": {},
    "created_at": "2024-01-01T00:00:00Z"
  }
}
```

#### 消息状态更新

```json
{
  "type": "message_status",
  "data": {
    "message_id": "msg_123",
    "status": "read",
    "timestamp": 1640995200000
  }
}
```

#### 系统通知

```json
{
  "type": "notification",
  "data": {
    "notification_type": "group_invite",
    "title": "群组邀请",
    "content": "John Doe 邀请你加入技术交流群",
    "metadata": {
      "group_id": "group_123",
      "inviter_id": "user_123"
    },
    "created_at": "2024-01-01T00:00:00Z"
  }
}
```

### 消息类型

| 类型 | 说明 | 示例 |
|------|------|------|
| text | 文本消息 | "Hello World" |
| image | 图片消息 | {"url": "https://example.com/image.jpg", "width": 800, "height": 600} |
| file | 文件消息 | {"url": "https://example.com/file.pdf", "name": "document.pdf", "size": 1024000} |
| audio | 音频消息 | {"url": "https://example.com/audio.mp3", "duration": 30} |
| video | 视频消息 | {"url": "https://example.com/video.mp4", "duration": 60, "thumbnail": "https://example.com/thumb.jpg"} |
| location | 位置消息 | {"latitude": 39.9042, "longitude": 116.4074, "address": "北京市朝阳区"} |

## gRPC API

### 服务定义

#### LogicService

```protobuf
service LogicService {
    rpc Authenticate(AuthRequest) returns (AuthResponse);
    rpc ValidateToken(ValidateTokenRequest) returns (ValidateTokenResponse);
    rpc GetUserProfile(GetUserProfileRequest) returns (GetUserProfileResponse);
    rpc UpdateUserProfile(UpdateUserProfileRequest) returns (UpdateUserProfileResponse);
    rpc SendMessage(SendMessageRequest) returns (SendMessageResponse);
    rpc GetMessageHistory(GetMessageHistoryRequest) returns (GetMessageHistoryResponse);
    rpc MarkMessageRead(MarkMessageReadRequest) returns (MarkMessageReadResponse);
    rpc CreateConversation(CreateConversationRequest) returns (CreateConversationResponse);
    rpc GetConversations(GetConversationsRequest) returns (GetConversationsResponse);
    rpc UpdateConversation(UpdateConversationRequest) returns (UpdateConversationResponse);
    rpc CreateGroup(CreateGroupRequest) returns (CreateGroupResponse);
    rpc GetGroupInfo(GetGroupInfoRequest) returns (GetGroupInfoResponse);
    rpc JoinGroup(JoinGroupRequest) returns (JoinGroupResponse);
    rpc LeaveGroup(LeaveGroupRequest) returns (LeaveGroupResponse);
    rpc AddGroupMember(AddGroupMemberRequest) returns (AddGroupMemberResponse);
    rpc RemoveGroupMember(RemoveGroupMemberRequest) returns (RemoveGroupMemberResponse);
}
```

#### RepoService

```protobuf
service RepoService {
    rpc CreateUser(CreateUserRequest) returns (CreateUserResponse);
    rpc GetUser(GetUserRequest) returns (GetUserResponse);
    rpc UpdateUser(UpdateUserRequest) returns (UpdateUserResponse);
    rpc CreateMessage(CreateMessageRequest) returns (CreateMessageResponse);
    rpc GetMessages(GetMessagesRequest) returns (GetMessagesResponse);
    rpc UpdateMessageStatus(UpdateMessageStatusRequest) returns (UpdateMessageStatusResponse);
    rpc CreateConversation(CreateConversationRequest) returns (CreateConversationResponse);
    rpc GetConversation(GetConversationRequest) returns (GetConversationResponse);
    rpc UpdateConversation(UpdateConversationRequest) returns (UpdateConversationResponse);
    rpc CreateGroup(CreateGroupRequest) returns (CreateGroupResponse);
    rpc GetGroup(GetGroupRequest) returns (GetGroupResponse);
    rpc UpdateGroup(UpdateGroupRequest) returns (UpdateGroupResponse);
    rpc AddGroupMember(AddGroupMemberRequest) returns (AddGroupMemberResponse);
    rpc RemoveGroupMember(RemoveGroupMemberRequest) returns (RemoveGroupMemberResponse);
}
```

### 错误处理

#### gRPC 状态码

| 状态码 | 说明 |
|--------|------|
| OK | 成功 |
| INVALID_ARGUMENT | 参数错误 |
| UNAUTHENTICATED | 未认证 |
| PERMISSION_DENIED | 权限不足 |
| NOT_FOUND | 资源不存在 |
| INTERNAL | 内部错误 |
| UNAVAILABLE | 服务不可用 |

## 文件上传

### 上传接口

```http
POST /api/v1/upload
Content-Type: multipart/form-data
```

**请求参数**:

- `file`: 文件内容
- `type`: 文件类型 (image, file, audio, video)

**响应示例**:

```json
{
  "code": 0,
  "message": "success",
  "data": {
    "file_id": "file_123",
    "url": "https://example.com/files/file_123.jpg",
    "name": "example.jpg",
    "size": 1024000,
    "content_type": "image/jpeg",
    "created_at": "2024-01-01T00:00:00Z"
  }
}
```

### 文件限制

- **图片**: 最大 10MB
- **文件**: 最大 100MB
- **音频**: 最大 50MB
- **视频**: 最大 500MB

## 限流策略

### API 限流

| 接口类型 | 限制 |
|----------|------|
| 认证接口 | 10次/分钟 |
| 消息发送 | 60次/分钟 |
| 文件上传 | 5次/分钟 |
| 其他接口 | 100次/分钟 |

### WebSocket 限流

- **消息发送**: 10条/秒
- **连接数**: 每个用户最多 5 个连接

## 安全规范

### 输入验证

- 所有输入参数必须进行验证
- 特殊字符必须进行转义
- 文件上传必须检查文件类型和大小

### 输出过滤

- 敏感信息必须脱敏
- 输出内容必须进行 XSS 过滤
- 错误信息不能包含敏感数据

### HTTPS 要求

- 所有 API 必须使用 HTTPS
- WebSocket 必须使用 wss
- 证书必须使用权威 CA 签发

## 监控和日志

### API 监控

- **响应时间**: P95 < 200ms
- **错误率**: < 1%
- **可用性**: > 99.9%

### 日志记录

- 所有 API 调用必须记录日志
- 错误信息必须记录详细信息
- 敏感信息不能记录到日志中