# API接口对接文档

## 接口概览

本文档描述了即时通讯系统的RESTful API接口和WebSocket实时通信协议，供后端开发者对接使用。

## 基础信息

- **Base URL**: `http://localhost:8080/api` (开发环境)
- **WebSocket URL**: `ws://localhost:8080/ws` (开发环境)
- **Content-Type**: `application/json`
- **认证方式**: Bearer Token

## 通用响应格式

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

### HTTP状态码
- `200` - 请求成功
- `201` - 创建成功
- `400` - 请求参数错误
- `401` - 未认证
- `403` - 无权限
- `404` - 资源不存在
- `409` - 资源冲突
- `500` - 服务器错误

## 认证接口

### 1. 用户登录
**接口**: `POST /auth/login`

**请求参数**:
```json
{
  "username": "string",    // 用户名，必填
  "password": "string"     // 密码，必填
}
```

**响应数据**:
```json
{
  "success": true,
  "message": "登录成功",
  "data": {
    "token": "string",     // JWT token
    "user": {
      "userId": "string",
      "username": "string",
      "avatar": "string",
      "email": "string"
    }
  }
}
```

**错误码**:
- `INVALID_CREDENTIALS` - 用户名或密码错误
- `USER_NOT_FOUND` - 用户不存在

### 2. 用户注册
**接口**: `POST /auth/register`

**请求参数**:
```json
{
  "username": "string",    // 用户名，3-20字符
  "password": "string",    // 密码，6-50字符
  "email": "string"        // 邮箱，可选
}
```

**响应数据**:
```json
{
  "success": true,
  "message": "注册成功",
  "data": {
    "user": {
      "userId": "string",
      "username": "string",
      "avatar": "string",
      "email": "string"
    }
  }
}
```

**错误码**:
- `USERNAME_EXISTS` - 用户名已存在
- `INVALID_USERNAME` - 用户名格式错误
- `INVALID_PASSWORD` - 密码格式错误

### 3. 用户登出
**接口**: `POST /auth/logout`

**请求头**:
```
Authorization: Bearer <token>
```

**响应数据**:
```json
{
  "success": true,
  "message": "登出成功"
}
```

## 用户管理接口

### 1. 获取当前用户信息
**接口**: `GET /users/info`

**请求头**:
```
Authorization: Bearer <token>
```

**响应数据**:
```json
{
  "success": true,
  "data": {
    "userId": "string",
    "username": "string",
    "avatar": "string",
    "email": "string",
    "createTime": "string"
  }
}
```

### 2. 搜索用户
**接口**: `GET /users/{username}`

**请求头**:
```
Authorization: Bearer <token>
```

**路径参数**:
- `username` - 要搜索的用户名

**响应数据**:
```json
{
  "success": true,
  "data": {
    "userId": "string",
    "username": "string",
    "avatar": "string",
    "email": "string"
  }
}
```

**错误码**:
- `USER_NOT_FOUND` - 用户不存在

### 3. 更新用户信息
**接口**: `PUT /users/profile`

**请求头**:
```
Authorization: Bearer <token>
```

**请求参数**:
```json
{
  "username": "string",    // 新用户名，可选
  "avatar": "string"       // 头像URL，可选
}
```

**响应数据**:
```json
{
  "success": true,
  "message": "更新成功",
  "data": {
    "userId": "string",
    "username": "string",
    "avatar": "string",
    "email": "string"
  }
}
```

## 会话管理接口

### 1. 获取会话列表
**接口**: `GET /conversations`

**请求头**:
```
Authorization: Bearer <token>
```

**响应数据**:
```json
{
  "success": true,
  "data": [
    {
      "conversationId": "string",
      "type": "single|group",    // 会话类型
      "target": {
        // 单聊时
        "userId": "string",
        "username": "string",
        "avatar": "string",
        // 群聊时
        "groupId": "string",
        "groupName": "string",
        "avatar": "string",
        "memberCount": "number"
      },
      "lastMessage": "string",
      "lastMessageTime": "string",
      "unreadCount": "number"
    }
  ]
}
```

### 2. 获取会话详情
**接口**: `GET /conversations/{conversationId}`

**请求头**:
```
Authorization: Bearer <token>
```

**路径参数**:
- `conversationId` - 会话ID

**响应数据**:
```json
{
  "success": true,
  "data": {
    "conversationId": "string",
    "type": "single|group",
    "target": {},
    "lastMessage": "string",
    "lastMessageTime": "string",
    "unreadCount": "number"
  }
}
```

### 3. 获取会话消息
**接口**: `GET /conversations/{conversationId}/messages`

**请求头**:
```
Authorization: Bearer <token>
```

**查询参数**:
- `page` - 页码，默认1
- `size` - 每页数量，默认50

**响应数据**:
```json
{
  "success": true,
  "data": [
    {
      "messageId": "string",
      "conversationId": "string",
      "senderId": "string",
      "senderName": "string",
      "content": "string",
      "type": "text|image|file",
      "sendTime": "string",
      "status": "sent|delivered|read"
    }
  ],
  "pagination": {
    "page": "number",
    "size": "number",
    "total": "number",
    "hasMore": "boolean"
  }
}
```

### 4. 标记会话已读
**接口**: `PUT /conversations/{conversationId}/read`

**请求头**:
```
Authorization: Bearer <token>
```

**响应数据**:
```json
{
  "success": true,
  "message": "标记已读成功"
}
```

### 5. 创建会话
**接口**: `POST /conversations`

**请求头**:
```
Authorization: Bearer <token>
```

**请求参数**:
```json
{
  "targetUserId": "string"    // 目标用户ID（私聊）
}
```

**响应数据**:
```json
{
  "success": true,
  "message": "会话创建成功",
  "data": {
    "conversationId": "string",
    "type": "single",
    "target": {
      "userId": "string",
      "username": "string",
      "avatar": "string"
    }
  }
}
```

## 好友管理接口

### 1. 添加好友
**接口**: `POST /friends`

**请求头**:
```
Authorization: Bearer <token>
```

**请求参数**:
```json
{
  "friendId": "string"    // 好友用户ID
}
```

**响应数据**:
```json
{
  "success": true,
  "message": "好友添加成功",
  "data": {
    "friendship": {
      "userId": "string",
      "friendId": "string",
      "status": "accepted"
    },
    "conversation": {
      "conversationId": "string",
      "type": "single",
      "target": {}
    }
  }
}
```

### 2. 获取好友列表
**接口**: `GET /friends`

**请求头**:
```
Authorization: Bearer <token>
```

**响应数据**:
```json
{
  "success": true,
  "data": [
    {
      "userId": "string",
      "username": "string",
      "avatar": "string",
      "online": "boolean"
    }
  ]
}
```

### 3. 删除好友
**接口**: `DELETE /friends/{friendId}`

**请求头**:
```
Authorization: Bearer <token>
```

**路径参数**:
- `friendId` - 好友用户ID

**响应数据**:
```json
{
  "success": true,
  "message": "好友删除成功"
}
```

## 群聊管理接口

### 1. 创建群聊
**接口**: `POST /groups`

**请求头**:
```
Authorization: Bearer <token>
```

**请求参数**:
```json
{
  "groupName": "string",     // 群名称
  "members": ["string"]      // 成员用户ID数组
}
```

**响应数据**:
```json
{
  "success": true,
  "message": "群聊创建成功",
  "data": {
    "group": {
      "groupId": "string",
      "groupName": "string",
      "avatar": "string",
      "description": "string",
      "members": [
        {
          "userId": "string",
          "role": "admin|member",
          "joinTime": "string"
        }
      ],
      "createTime": "string"
    },
    "conversation": {
      "conversationId": "string",
      "type": "group"
    }
  }
}
```

### 2. 获取群聊信息
**接口**: `GET /groups/{groupId}`

**请求头**:
```
Authorization: Bearer <token>
```

**响应数据**:
```json
{
  "success": true,
  "data": {
    "groupId": "string",
    "groupName": "string",
    "avatar": "string",
    "description": "string",
    "members": [
      {
        "userId": "string",
        "username": "string",
        "avatar": "string",
        "role": "admin|member",
        "joinTime": "string"
      }
    ],
    "createTime": "string"
  }
}
```

### 3. 添加群成员
**接口**: `POST /groups/{groupId}/members`

**请求头**:
```
Authorization: Bearer <token>
```

**请求参数**:
```json
{
  "memberIds": ["string"]    // 新成员用户ID数组
}
```

**响应数据**:
```json
{
  "success": true,
  "message": "成功添加 2 名成员",
  "data": [
    {
      "userId": "string",
      "role": "member",
      "joinTime": "string"
    }
  ]
}
```

### 4. 移除群成员
**接口**: `DELETE /groups/{groupId}/members/{userId}`

**请求头**:
```
Authorization: Bearer <token>
```

**路径参数**:
- `groupId` - 群聊ID
- `userId` - 要移除的用户ID

**响应数据**:
```json
{
  "success": true,
  "message": "移除成员成功"
}
```

## 消息管理接口

### 1. 发送消息（HTTP方式）
**接口**: `POST /messages`

**请求头**:
```
Authorization: Bearer <token>
```

**请求参数**:
```json
{
  "conversationId": "string",
  "content": "string",
  "type": "text|image|file"
}
```

**响应数据**:
```json
{
  "success": true,
  "message": "消息发送成功",
  "data": {
    "messageId": "string",
    "conversationId": "string",
    "senderId": "string",
    "senderName": "string",
    "content": "string",
    "type": "text",
    "sendTime": "string",
    "status": "sent"
  }
}
```

### 2. 撤回消息
**接口**: `DELETE /messages/{messageId}`

**请求头**:
```
Authorization: Bearer <token>
```

**路径参数**:
- `messageId` - 消息ID

**响应数据**:
```json
{
  "success": true,
  "message": "消息撤回成功",
  "data": {
    "messageId": "string",
    "content": "消息已撤回",
    "type": "recalled",
    "recallTime": "string"
  }
}
```

**错误码**:
- `MESSAGE_NOT_FOUND` - 消息不存在
- `NO_PERMISSION` - 无权限撤回
- `TIME_LIMIT_EXCEEDED` - 超过撤回时间限制

### 3. 搜索消息
**接口**: `GET /messages/search/{keyword}`

**请求头**:
```
Authorization: Bearer <token>
```

**查询参数**:
- `conversationId` - 限制搜索范围到特定会话（可选）

**响应数据**:
```json
{
  "success": true,
  "data": [
    {
      "messageId": "string",
      "conversationId": "string",
      "senderId": "string",
      "senderName": "string",
      "content": "string",
      "type": "text",
      "sendTime": "string"
    }
  ],
  "total": "number"
}
```

## WebSocket实时通信协议

### 连接建立
**URL**: `ws://localhost:8080/ws?token={jwt_token}`

**连接确认**:
```json
{
  "type": "connected",
  "data": {
    "message": "WebSocket连接成功"
  }
}
```

### 消息类型

#### 1. 发送消息
**客户端发送**:
```json
{
  "type": "send-message",
  "data": {
    "conversationId": "string",
    "content": "string",
    "messageType": "text|image|file",
    "tempMessageId": "string"    // 客户端临时ID
  }
}
```

**服务器确认**:
```json
{
  "type": "message-ack",
  "data": {
    "messageId": "string",      // 服务器生成的消息ID
    "tempMessageId": "string"   // 对应的临时ID
  }
}
```

#### 2. 接收新消息
**服务器推送**:
```json
{
  "type": "new-message",
  "data": {
    "conversationId": "string",
    "message": {
      "messageId": "string",
      "conversationId": "string",
      "senderId": "string",
      "senderName": "string",
      "content": "string",
      "type": "text",
      "sendTime": "string",
      "status": "sent"
    }
  }
}
```

#### 3. 在线状态更新
**服务器推送**:
```json
{
  "type": "friend-online",
  "data": {
    "userId": "string",
    "online": "boolean"
  }
}
```

#### 4. 群成员在线状态
**服务器推送**:
```json
{
  "type": "group-member-online",
  "data": {
    "groupId": "string",
    "userId": "string",
    "online": "boolean"
  }
}
```

#### 5. 会话更新
**服务器推送**:
```json
{
  "type": "conversation-update",
  "data": {
    "conversationId": "string",
    "updates": {
      "lastMessage": "string",
      "lastMessageTime": "string",
      "unreadCount": "number"
    }
  }
}
```

#### 6. 心跳检测
**客户端发送**:
```json
{
  "type": "ping"
}
```

**服务器响应**:
```json
{
  "type": "pong"
}
```

## 数据模型

### User（用户）
```json
{
  "userId": "string",      // 用户ID
  "username": "string",    // 用户名
  "avatar": "string",      // 头像URL
  "email": "string",       // 邮箱
  "createTime": "string",  // 创建时间
  "online": "boolean"      // 在线状态
}
```

### Conversation（会话）
```json
{
  "conversationId": "string",    // 会话ID
  "type": "single|group",        // 会话类型
  "target": {                    // 目标对象
    "userId": "string",          // 单聊：对方用户ID
    "username": "string",        // 单聊：对方用户名
    "groupId": "string",         // 群聊：群ID
    "groupName": "string",       // 群聊：群名称
    "avatar": "string",          // 头像
    "memberCount": "number"      // 群聊：成员数量
  },
  "lastMessage": "string",       // 最后消息
  "lastMessageTime": "string",   // 最后消息时间
  "unreadCount": "number"        // 未读消息数
}
```

### Message（消息）
```json
{
  "messageId": "string",         // 消息ID
  "conversationId": "string",    // 会话ID
  "senderId": "string",          // 发送者ID
  "senderName": "string",        // 发送者姓名
  "content": "string",           // 消息内容
  "type": "text|image|file|recalled",  // 消息类型
  "sendTime": "string",          // 发送时间
  "status": "sending|sent|delivered|read|failed"  // 消息状态
}
```

### Group（群聊）
```json
{
  "groupId": "string",           // 群ID
  "groupName": "string",         // 群名称
  "avatar": "string",            // 群头像
  "description": "string",       // 群描述
  "members": [                   // 群成员
    {
      "userId": "string",
      "role": "admin|member",
      "joinTime": "string"
    }
  ],
  "createTime": "string"         // 创建时间
}
```

## 错误处理

### 常见错误码
- `INVALID_TOKEN` - 无效的认证令牌
- `TOKEN_EXPIRED` - 令牌已过期
- `PERMISSION_DENIED` - 权限不足
- `RESOURCE_NOT_FOUND` - 资源不存在
- `DUPLICATE_RESOURCE` - 资源重复
- `VALIDATION_ERROR` - 参数验证失败
- `RATE_LIMIT_EXCEEDED` - 请求频率超限
- `SERVER_ERROR` - 服务器内部错误

### 错误响应示例
```json
{
  "success": false,
  "message": "用户名已存在",
  "code": "USERNAME_EXISTS",
  "timestamp": "2025-01-13T15:30:00.000Z"
}
```

## 开发注意事项

### 1. 认证和安全
- 所有需要认证的接口必须在请求头中包含有效的JWT token
- Token应设置合理的过期时间，建议7天
- 敏感操作应进行二次验证

### 2. 分页处理
- 消息历史、好友列表等大数据量接口需要支持分页
- 建议每页数量限制在50以内
- 提供hasMore字段指示是否有更多数据

### 3. 实时性要求
- 消息发送/接收应优先使用WebSocket
- HTTP接口作为WebSocket的备用方案
- 在线状态更新应实时推送

### 4. 性能优化
- 支持消息内容压缩
- 实现适当的缓存策略
- 提供CDN支持用于文件上传

### 5. 数据一致性
- 确保会话列表与消息列表的一致性
- 处理并发操作的冲突
- 实现适当的数据备份机制

---
更新时间：2025-01-13 23:59
