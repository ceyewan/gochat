# GoChat 即时通讯系统 API 文档

## 概述

本文档定义了 GoChat 即时通讯系统的 RESTful API 接口和 WebSocket 实时通信协议，供后端开发者实现使用。

## 基础配置

- **Base URL**: `http://localhost:8080/api` (im-gateway服务)
- **WebSocket URL**: `ws://localhost:8080/ws` (im-gateway服务)
- **Content-Type**: `application/json`
- **认证方式**: Bearer Token (JWT)
- **API版本**: v1
- **字符编码**: UTF-8

## 架构说明

本API由GoChat微服务架构中的`im-gateway`服务对外提供，内部通过gRPC调用`im-logic`服务处理业务逻辑，通过Kafka进行异步消息处理。所有数据访问通过`im-repo`服务统一管理。

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
- `200` - 成功
- `201` - 创建成功
- `400` - 参数错误
- `401` - 未认证
- `403` - 无权限
- `404` - 资源不存在
- `500` - 服务器错误

## 1. 认证接口

### 用户登录
`POST /auth/login`

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
  "data": {
    "token": "string",     // JWT token
    "user": {
      "userId": "string",
      "username": "string",
      "avatar": "string"
    }
  }
}
```

### 用户注册
`POST /auth/register`

**请求参数**:
```json
{
  "username": "string",    // 用户名，3-20字符
  "password": "string"     // 密码，6-50字符
}
```

**响应数据**:
```json
{
  "success": true,
  "data": {
    "user": {
      "userId": "string",
      "username": "string",
      "avatar": "string"
    }
  }
}
```

### 游客登录
`POST /auth/guest`

**请求参数**:
```json
{
  "guestName": "string"    // 游客昵称，可选，默认生成随机昵称
}
```

**响应数据**:
```json
{
  "success": true,
  "data": {
    "token": "string",     // JWT token
    "user": {
      "userId": "string",
      "username": "string",
      "avatar": "string",
      "isGuest": true
    }
  }
}
```

### 用户登出
`POST /auth/logout`

**请求头**: `Authorization: Bearer <token>`

**响应数据**:
```json
{
  "success": true,
  "message": "登出成功"
}
```

## 2. 用户管理接口

### 获取当前用户信息
`GET /users/info`

**请求头**: `Authorization: Bearer <token>`

**响应数据**:
```json
{
  "success": true,
  "data": {
    "userId": "string",
    "username": "string",
    "avatar": "string",
    "isGuest": "boolean"
  }
}
```

### 搜索用户
`GET /users/{username}`

**请求头**: `Authorization: Bearer <token>`

**响应数据**:
```json
{
  "success": true,
  "data": {
    "userId": "string",
    "username": "string",
    "avatar": "string"
  }
}
```

## 3. 会话管理接口

### 获取会话列表
`GET /conversations`

**请求头**: `Authorization: Bearer <token>`

**响应数据**:
```json
{
  "success": true,
  "data": [
    {
      "conversationId": "world",
      "type": "world",
      "target": {
        "groupId": "world",
        "groupName": "世界聊天室",
        "avatar": "string",
        "description": "所有用户都可以参与的公共聊天室"
      },
      "lastMessage": "string",
      "lastMessageTime": "string",
      "unreadCount": "number"
    },
    {
      "conversationId": "string",
      "type": "single",
      "target": {
        "userId": "string",
        "username": "string",
        "avatar": "string"
      },
      "lastMessage": "string",
      "lastMessageTime": "string",
      "unreadCount": "number"
    },
    {
      "conversationId": "string",
      "type": "group",
      "target": {
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

### 获取会话消息
`GET /conversations/{conversationId}/messages`

**请求头**: `Authorization: Bearer <token>`

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
      "type": "text",
      "sendTime": "string"
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

### 标记会话已读
`PUT /conversations/{conversationId}/read`

**请求头**: `Authorization: Bearer <token>`

### 创建私聊会话
`POST /conversations`

**请求头**: `Authorization: Bearer <token>`

**请求参数**:
```json
{
  "targetUserId": "string"    // 目标用户ID
}
```

**响应数据**:
```json
{
  "success": true,
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



## 4. 好友管理接口

### 添加好友
`POST /friends`

**请求头**: `Authorization: Bearer <token>`

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
  "data": {
    "conversation": {
      "conversationId": "string",
      "type": "single",
      "target": {
        "userId": "string",
        "username": "string",
        "avatar": "string"
      }
    }
  }
}
```

### 获取好友列表
`GET /friends`

**请求头**: `Authorization: Bearer <token>`

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

## 5. 群聊管理接口

### 创建群聊
`POST /groups`

**请求头**: `Authorization: Bearer <token>`

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
  "data": {
    "group": {
      "groupId": "string",
      "groupName": "string",
      "avatar": "string",
      "members": [
        {
          "userId": "string",
          "role": "admin|member"
        }
      ]
    },
    "conversation": {
      "conversationId": "string",
      "type": "group"
    }
  }
}
```

### 获取群聊信息
`GET /groups/{groupId}`

**请求头**: `Authorization: Bearer <token>`

**响应数据**:
```json
{
  "success": true,
  "data": {
    "groupId": "string",
    "groupName": "string",
    "avatar": "string",
    "members": [
      {
        "userId": "string",
        "username": "string",
        "avatar": "string",
        "role": "admin|member"
      }
    ]
  }
}
```

## 6. 消息管理接口

### 发送消息（HTTP方式）
`POST /messages`

**请求头**: `Authorization: Bearer <token>`

**请求参数**:
```json
{
  "conversationId": "string",
  "content": "string",
  "type": "text"
}
```

**响应数据**:
```json
{
  "success": true,
  "data": {
    "messageId": "string",
    "conversationId": "string",
    "senderId": "string",
    "senderName": "string",
    "content": "string",
    "type": "text",
    "sendTime": "string"
  }
}
```

## 7. WebSocket 实时通信协议

### 连接建立
**URL**: `ws://localhost:8080/ws?token={jwt_token}`

### 消息类型

#### 发送消息
**客户端发送**:
```json
{
  "type": "send-message",
  "data": {
    "conversationId": "string",     // 会话ID
    "content": "string",            // 消息内容
    "messageType": "text",          // 消息类型 (text/image/file)
    "clientMsgId": "string"         // 客户端消息ID (用于幂等性)
  },
  "traceId": "string"               // 可选：链路追踪ID
}
```

**服务器确认** (立即返回，优化UI体验):
```json
{
  "type": "message-ack",
  "data": {
    "messageId": "string",          // 服务器生成的消息ID
    "clientMsgId": "string",        // 对应的客户端消息ID
    "seqId": "number",              // 会话内序列号
    "sendTime": "string"            // 服务器处理时间
  },
  "traceId": "string"
}
```

#### 接收新消息
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
      "sendTime": "string"
    }
  }
}
```

#### 在线状态更新
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

#### 心跳检测
**客户端发送**: `{"type": "ping"}`
**服务器响应**: `{"type": "pong"}`

## 8. 数据模型

### User（用户）
```json
{
  "userId": "string",      // 用户ID
  "username": "string",    // 用户名
  "avatar": "string",      // 头像URL
  "isGuest": "boolean"     // 是否为游客
}
```

### Conversation（会话）
```json
{
  "conversationId": "string",    // 会话ID，格式：single_xxx/group_xxx/world
  "type": "single|group|world",  // 会话类型
  "target": {                    // 目标对象
    "userId": "string",          // 单聊：对方用户ID (BIGINT UNSIGNED)
    "username": "string",        // 单聊：对方用户名
    "groupId": "string",         // 群聊/世界聊天室：群ID (BIGINT UNSIGNED)
    "groupName": "string",       // 群聊/世界聊天室：群名称
    "avatar": "string",          // 头像URL
    "memberCount": "number",     // 群聊：成员数量（世界聊天室无此字段）
    "description": "string"      // 世界聊天室：描述信息
  },
  "lastMessage": "string",       // 最后消息内容
  "lastMessageTime": "string",   // 最后消息时间 (ISO 8601格式)
  "lastMessageSeq": "number",    // 最后消息序列号
  "unreadCount": "number"        // 未读消息数
}
```

### Message（消息）
```json
{
  "messageId": "string",         // 消息ID (BIGINT UNSIGNED，Snowflake生成)
  "conversationId": "string",    // 会话ID
  "senderId": "string",          // 发送者ID (BIGINT UNSIGNED)
  "senderName": "string",        // 发送者用户名
  "content": "string",           // 消息内容
  "type": "text|image|file",     // 消息类型 (1:文本, 2:图片, 3:文件)
  "seqId": "number",             // 会话内序列号
  "sendTime": "string",          // 发送时间 (ISO 8601格式)
  "clientMsgId": "string"        // 客户端消息ID (用于去重)
}
```

## 9. 特殊说明

### 世界聊天室
- 世界聊天室是一个特殊的群聊，`conversationId` 固定为 `"world"`
- 所有用户（包括游客）在获取会话列表时都会包含世界聊天室
- 世界聊天室不需要维护成员关系，所有用户都能发送和接收消息
- 消息推送时，需要向所有在线用户推送世界聊天室的消息

### 游客限制
- 游客用户 `isGuest` 字段为 `true`
- 游客不能添加好友（前端隐藏添加好友按钮）
- 游客不能创建群聊（前端隐藏创建群聊按钮）
- 游客可以参与世界聊天室和接受其他用户发起的私聊

### 实现要点
- 后端在返回会话列表时，始终包含世界聊天室作为第一个会话
- 游客用户的会话列表中，只显示世界聊天室和被动创建的私聊会话
- 世界聊天室消息使用与群聊相同的推送机制，但推送给所有在线用户
- 前端根据用户的 `isGuest` 字段控制功能按钮的显示

---
更新时间：2025-01-19
