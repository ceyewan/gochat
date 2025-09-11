# GoChat 核心业务流程与数据流

本文档描述 GoChat 系统的核心业务流程，明确使用的数据模型和 RPC 接口。

## 1. 会话创建流程

### 1.1 好友申请 → 单聊会话

**业务流程：**
```
用户A发起好友申请 → im-logic处理申请 → im-repo存储申请记录 → 发布好友事件
用户B同意申请 → 更新好友关系 → 自动创建单聊会话 → 通知双方
```

**涉及的数据模型：**
- `friendship_requests` 表：存储好友申请记录
- `conversations` 表：创建单聊会话记录（type=1）
- `conversation_members` 表：添加会话成员关系

**使用的 RPC 接口：**
```protobuf
// im-logic 层
rpc CreateConversation(CreateConversationRequest) returns (CreateConversationResponse);

// im-repo 层  
rpc CreateUser(CreateUserRequest) returns (CreateUserResponse);
rpc CreateConversation(CreateConversationRequest) returns (CreateConversationResponse);
rpc AddConversationMember(AddConversationMemberRequest) returns (AddConversationMemberResponse);
```

**Kafka 消息：**
- Topic: `gochat.friend-events`
- Topic: `gochat.conversation-events`

### 1.2 群聊创建流程

**业务流程：**
```
创建者发起 → 验证权限（游客无法创建群聊）→ 创建会话记录 → 批量添加成员 → 发布会话创建事件
```

**涉及的数据模型：**
- `conversations` 表：type=2, owner_id=创建者
- `conversation_members` 表：批量插入成员记录，创建者角色为owner

**使用的 RPC 接口：**
```protobuf
// im-logic 层
rpc CreateConversation(CreateConversationRequest) returns (CreateConversationResponse);

// im-repo 层
rpc CreateConversation(CreateConversationRequest) returns (CreateConversationResponse);
rpc AddConversationMember(AddConversationMemberRequest) returns (AddConversationMemberResponse);
rpc GetConversationMembers(GetConversationMembersRequest) returns (GetConversationMembersResponse);
```

## 2. 消息收发流程

### 2.1 统一消息发送流程

**数据流向：**
```
Client WebSocket → Gateway → Kafka(upstream) → Logic → Repo(持久化) 
                                            ↓
                             Kafka(downstream) → Gateway → WebSocket → Client
                                            ↓
                             Kafka(task) → Task(异步处理)
```

**涉及的数据模型：**
- `messages` 表：存储消息记录，包含 seq_id（会话内单调递增）
- `conversations` 表：更新 last_message_id 字段
- `conversation_members` 表：验证发送者权限

**使用的 RPC 接口：**
```protobuf
// im-logic 层
rpc SendMessage(SendMessageRequest) returns (SendMessageResponse);

// im-repo 层
rpc SaveMessage(SaveMessageRequest) returns (SaveMessageResponse);
rpc GetConversationMembers(GetConversationMembersRequest) returns (GetConversationMembersResponse);
rpc GetUserOnlineStatus(GetUserOnlineStatusRequest) returns (GetUserOnlineStatusResponse);
```

**Kafka 消息结构：**
```go
// 上游消息
type UpstreamMessage struct {
    UserID         string `json:"user_id"`
    ConversationID string `json:"conversation_id"`
    MessageType    int    `json:"message_type"`
    Content        string `json:"content"`
    ClientMsgID    string `json:"client_msg_id"`
    Timestamp      int64  `json:"timestamp"`
}

// 下游消息  
type DownstreamMessage struct {
    TargetUserID   string    `json:"target_user_id"`
    MessageID      int64     `json:"message_id"`
    ConversationID string    `json:"conversation_id"`
    SenderInfo     *UserInfo `json:"sender_info"`
    Content        string    `json:"content"`
    SeqID          int64     `json:"seq_id"`
}
```

### 2.2 扇出策略选择

**根据会话类型和规模：**
```go
if conversationType == 1 { // 单聊
    // Logic 直接推送，查询对方在线状态
    RPC: GetUserOnlineStatus(target_user_id)
} else if memberCount <= 500 { // 中小群
    // Logic 批量查询在线成员并推送
    RPC: BatchGetOnlineStatus(member_ids)
} else { // 大群或世界聊天室  
    // 发送到异步扇出队列
    Kafka: gochat.tasks.fanout
}
```

## 3. 会话列表优化（解决N+1问题）

### 3.1 优化前的问题
```
1. 获取用户会话ID列表：GetUserConversations()
2. 循环查询每个会话详情：GetConversation() × N次  
3. 循环查询每个会话未读数：GetUnreadCount() × N次
总计：1 + N + N = 2N+1 次查询
```

### 3.2 优化后的解决方案

**使用的 RPC 接口：**
```protobuf
// im-repo 层新增接口
rpc GetUserConversationsWithDetails(GetUserConversationsWithDetailsRequest) 
    returns (GetUserConversationsWithDetailsResponse);

message GetUserConversationsWithDetailsRequest {
    string user_id = 1;
    int32 offset = 2;
    int32 limit = 3;
    bool include_last_message = 5;
    bool include_sender_info = 6;
}

message ConversationWithDetails {
    Conversation conversation = 1;              // 会话基本信息
    ConversationMember user_membership = 2;     // 用户在会话中的角色
    LastMessage last_message = 3;               // 最后一条消息
    int64 unread_count = 4;                     // 未读数量
}
```

**数据库查询优化：**
```sql
-- 一次复杂查询获取所有信息
SELECT 
    c.id, c.type, c.name, c.avatar_url, c.member_count,
    cm.role, cm.joined_at, cm.muted,
    lm.id as last_message_id, lm.content as last_content, lm.created_at as last_time,
    sender.username as sender_name, sender.avatar_url as sender_avatar,
    -- 计算未读数
    (SELECT COUNT(*) FROM messages m2 
     WHERE m2.conversation_id = c.id 
       AND m2.seq_id > COALESCE(urp.last_read_seq_id, 0)
       AND m2.deleted = false
    ) as unread_count
FROM conversation_members cm
JOIN conversations c ON cm.conversation_id = c.id
LEFT JOIN messages lm ON c.last_message_id = lm.id  
LEFT JOIN users sender ON lm.sender_id = sender.id
LEFT JOIN user_read_pointers urp ON (urp.user_id = cm.user_id AND urp.conversation_id = c.id)
WHERE cm.user_id = ?
ORDER BY c.updated_at DESC
LIMIT ? OFFSET ?;
```

### 3.3 缓存策略

**Redis 缓存层次：**
```yaml
user:conversations:full:{user_id}:     # 完整会话列表信息 (TTL: 1h)
conversation:basic:{conversation_id}:  # 会话基本信息 (TTL: 30min)  
user:unread:{user_id}:                 # 用户未读数映射 (TTL: 10min)
```

## 4. 用户状态管理

### 4.1 在线状态流程

**上线流程：**
```
WebSocket连接 → 验证JWT → 更新在线状态 → 发布用户上线事件 → 通知好友
```

**涉及的数据模型：**
- Redis: `user:online:{user_id}` 存储在线状态
- `friendship_requests` 表：查询已确认的好友关系

**使用的 RPC 接口：**
```protobuf
// im-repo 层
rpc SetUserOnline(SetUserOnlineRequest) returns (SetUserOnlineResponse);
rpc GetUserFriends(GetUserFriendsRequest) returns (GetUserFriendsResponse);
```

**下线流程：**
```
连接断开 → 更新离线状态 → 延迟30秒确认仍离线 → 发布下线事件 → 通知好友
```

### 4.2 已读状态更新

**业务流程：**
```
用户标记已读 → 更新read_pointer → 重新计算未读数 → 发布已读事件 → 通知消息发送方
```

**涉及的数据模型：**
- `user_read_pointers` 表：更新 last_read_seq_id
- `messages` 表：基于 seq_id 计算未读数量

**使用的 RPC 接口：**
```protobuf
// im-logic 层
rpc MarkAsRead(MarkAsReadRequest) returns (MarkAsReadResponse);

// im-repo 层
rpc UpdateReadPointer(UpdateReadPointerRequest) returns (UpdateReadPointerResponse);
rpc GetUnreadCount(GetUnreadCountRequest) returns (GetUnreadCountResponse);
```

## 5. 成员管理流程

### 5.1 添加会话成员

**业务流程：**
```
管理员发起 → 验证权限 → 批量验证用户 → 检查群容量 → 添加成员记录 → 发布成员变更事件
```

**涉及的数据模型：**
- `conversation_members` 表：批量插入新成员
- `conversations` 表：更新 member_count 字段
- `users` 表：验证用户存在性

**使用的 RPC 接口：**
```protobuf
// im-logic 层
rpc AddMembers(AddMembersRequest) returns (AddMembersResponse);

// im-repo 层
rpc GetUsers(GetUsersRequest) returns (GetUsersResponse);              // 批量验证用户
rpc AddConversationMember(AddConversationMemberRequest) returns (AddConversationMemberResponse);
rpc GetConversation(GetConversationRequest) returns (GetConversationResponse);  // 检查容量
```

### 5.2 移除会话成员

**涉及的数据模型：**
- `conversation_members` 表：删除成员记录
- `conversations` 表：更新 member_count

**使用的 RPC 接口：**
```protobuf
// im-logic 层
rpc RemoveMembers(RemoveMembersRequest) returns (RemoveMembersResponse);

// im-repo 层
rpc RemoveConversationMember(RemoveConversationMemberRequest) returns (RemoveConversationMemberResponse);
```

## 6. 错误处理与重试

### 6.1 消息发送失败处理

**失败场景及处理：**
```go
// 权限验证失败
ErrorCode: ERROR_CODE_NOT_CONVERSATION_MEMBER
Action: 返回错误给客户端，不重试

// 数据库写入失败  
ErrorCode: ERROR_CODE_INTERNAL_ERROR
Action: 客户端重试，使用相同client_msg_id保证幂等

// 扇出失败
Action: 记录失败日志，部分用户可能收不到消息
```

### 6.2 会话同步失败

**重试策略：**
```
网络超时：指数退避重试 (1s, 2s, 4s, 8s, 最大30s)
数据冲突：放弃本地修改，重新拉取服务器数据
服务不可用：使用本地缓存，显示离线状态
```

## 7. 监控指标

**关键业务指标：**
- `gochat_message_send_total`: 消息发送总数
- `gochat_message_send_failures_total`: 消息发送失败数
- `gochat_conversation_load_duration_seconds`: 会话列表加载耗时
- `gochat_fanout_latency_seconds`: 消息扇出延迟
- `gochat_cache_hit_ratio`: 缓存命中率

**告警规则：**
```yaml
- alert: MessageSendFailureHigh
  expr: rate(gochat_message_send_failures_total[5m]) / rate(gochat_message_send_total[5m]) > 0.01
  
- alert: ConversationLoadSlow  
  expr: histogram_quantile(0.95, rate(gochat_conversation_load_duration_seconds_bucket[5m])) > 0.5

- alert: CacheHitRateLow
  expr: gochat_cache_hit_ratio < 0.9
```

## 8. 好友管理流程

### 8.1 发送好友申请

**业务流程：**
```
用户A搜索用户B → 发起好友申请 → 验证权限 → 存储申请记录 → 发布好友事件 → 通知用户B
```

**涉及的数据模型：**
- `friendship_requests` 表：插入申请记录（status=0待处理）
- `users` 表：验证目标用户存在

**使用的 RPC 接口：**
```protobuf
// im-logic 层（需新增）
rpc SendFriendRequest(SendFriendRequestRequest) returns (SendFriendRequestResponse);

// im-repo 层（需新增）
rpc CreateFriendRequest(CreateFriendRequestRequest) returns (CreateFriendRequestResponse);
rpc GetUserByUsername(GetUserByUsernameRequest) returns (GetUserByUsernameResponse);
```

**HTTP API：**
```
POST /friends/requests
{
  "targetUserId": "user_123",
  "message": "Hello, let's be friends!"
}
```

### 8.2 处理好友申请

**业务流程：**
```
用户B查看申请列表 → 同意/拒绝申请 → 更新申请状态 → 创建好友关系 → 自动创建单聊会话
```

**涉及的数据模型：**
- `friendship_requests` 表：更新status（1=同意, 2=拒绝）
- `conversations` 表：自动创建单聊会话（type=1）
- `conversation_members` 表：添加双方为会话成员

**使用的 RPC 接口：**
```protobuf
// im-logic 层
rpc HandleFriendRequest(HandleFriendRequestRequest) returns (HandleFriendRequestResponse);
rpc CreateConversation(CreateConversationRequest) returns (CreateConversationResponse);

// im-repo 层
rpc UpdateFriendRequest(UpdateFriendRequestRequest) returns (UpdateFriendRequestResponse);
```

**HTTP API：**
```
PUT /friends/requests/{requestId}
{
  "action": 1  // 1=accept, 2=reject
}
```

## 9. 用户管理功能

### 9.1 用户搜索

**业务流程：**
```
输入搜索关键词 → 模糊匹配用户名 → 返回用户列表 → 支持发起好友申请
```

**涉及的数据模型：**
- `users` 表：基于username字段模糊查询

**使用的 RPC 接口：**
```protobuf
// im-logic 层（需新增）
rpc SearchUsers(SearchUsersRequest) returns (SearchUsersResponse);

// im-repo 层（需新增）
rpc SearchUsersByUsername(SearchUsersByUsernameRequest) returns (SearchUsersByUsernameResponse);
```

**HTTP API：**
```
GET /users/search?q=john&limit=10
```

### 9.2 用户资料管理

**业务流程：**
```
获取/更新用户资料 → 验证权限 → 更新数据库 → 清除缓存 → 发布用户更新事件
```

**涉及的数据模型：**
- `users` 表：更新avatar_url等字段

**使用的 RPC 接口：**
```protobuf
// im-repo 层
rpc UpdateUser(UpdateUserRequest) returns (UpdateUserResponse);
```

**HTTP API：**
```
GET /users/profile
PUT /users/profile
{
  "avatarUrl": "https://example.com/avatar.jpg"
}
```

## 10. 消息增强功能

### 10.1 消息删除/撤回

**业务流程：**
```
用户选择删除消息 → 验证权限和时间限制 → 软删除消息 → 发布消息撤回事件 → 通知会话成员
```

**涉及的数据模型：**
- `messages` 表：更新deleted字段为true

**使用的 RPC 接口：**
```protobuf
// im-logic 层（需新增）
rpc DeleteMessage(DeleteMessageRequest) returns (DeleteMessageResponse);

// im-repo 层
rpc DeleteMessage(DeleteMessageRequest) returns (DeleteMessageResponse);
```

### 10.2 消息幂等性检查

**业务流程：**
```
接收消息请求 → 检查client_msg_id → 如果已存在返回原消息 → 否则创建新消息
```

**使用的 RPC 接口：**
```protobuf
// im-repo 层
rpc CheckMessageIdempotency(CheckMessageIdempotencyRequest) returns (CheckMessageIdempotencyResponse);
```

## 11. 心跳与状态维护

### 11.1 WebSocket心跳机制

**业务流程：**
```
客户端每30秒发送ping → 服务器返回pong → 更新用户最后活跃时间 → 维护在线状态
```

**使用的 RPC 接口：**
```protobuf
// im-repo 层
rpc UpdateHeartbeat(UpdateHeartbeatRequest) returns (UpdateHeartbeatResponse);
```

### 11.2 离线状态清理

**业务流程：**
```
定时任务扫描 → 清理长时间未活跃用户 → 批量更新离线状态 → 通知相关好友
```

**使用的 RPC 接口：**
```protobuf
// im-repo 层
rpc CleanupExpiredStatus(CleanupExpiredStatusRequest) returns (CleanupExpiredStatusResponse);
```

通过明确的数据模型使用和 RPC 接口调用，确保系统各层职责清晰，数据流向明确。