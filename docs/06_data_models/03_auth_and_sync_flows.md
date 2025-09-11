# GoChat 认证与数据同步流程

本文档描述用户认证、数据同步、好友关系管理的核心流程，明确使用的数据模型和 RPC 接口。

## 1. 用户认证流程

### 1.1 用户注册流程

**业务流程：**
```
POST /auth/register → 验证用户名唯一性 → 密码哈希处理 → 存储用户记录 → 返回用户信息
```

**涉及的数据模型：**
- `users` 表：插入新用户记录
  - `username`: 唯一用户名
  - `password_hash`: BCrypt哈希后的密码
  - `is_guest`: false（正式用户）
  - `status`: 0（正常状态）

**使用的 RPC 接口：**
```protobuf
// im-logic 层
rpc Register(RegisterRequest) returns (RegisterResponse);

message RegisterRequest {
  string username = 1;
  string password = 2;
  string avatar_url = 3;
}

// im-repo 层
rpc GetUserByUsername(GetUserByUsernameRequest) returns (GetUserByUsernameResponse);  // 检查用户名
rpc CreateUser(CreateUserRequest) returns (CreateUserResponse);                        // 创建用户

message CreateUserRequest {
  string username = 1;
  string password_hash = 2;
  string avatar_url = 3;
  bool is_guest = 4;
}
```

### 1.2 用户登录流程

**业务流程：**
```
POST /auth/login → 查询用户记录 → 验证密码 → 生成JWT token → 设置在线状态 → 返回认证信息
```

**涉及的数据模型：**
- `users` 表：验证用户存在和状态
- Redis: `user:online:{user_id}` 设置在线状态

**使用的 RPC 接口：**
```protobuf
// im-logic 层
rpc Login(LoginRequest) returns (LoginResponse);

message LoginRequest {
  string username = 1;
  string password = 2;
  string client_info = 3;
}

message LoginResponse {
  string access_token = 1;
  string refresh_token = 2;
  int64 expires_in = 3;
  User user = 4;
}

// im-repo 层
rpc GetUserByUsername(GetUserByUsernameRequest) returns (GetUserByUsernameResponse);
rpc VerifyPassword(VerifyPasswordRequest) returns (VerifyPasswordResponse);
rpc SetUserOnline(SetUserOnlineRequest) returns (SetUserOnlineResponse);
```

### 1.3 游客登录流程

**业务流程：**
```
POST /auth/guest → 生成临时用户名 → 创建游客账户 → 自动加入世界聊天室 → 生成临时token
```

**涉及的数据模型：**
- `users` 表：创建游客用户（is_guest=true, username='guest_xxxx'）
- `conversation_members` 表：自动加入世界聊天室

**使用的 RPC 接口：**
```protobuf
// im-logic 层
rpc GuestLogin(GuestLoginRequest) returns (GuestLoginResponse);

message GuestLoginResponse {
  string access_token = 1;
  string refresh_token = 2;
  int64 expires_in = 3;
  User user = 4;
}

// im-repo 层
rpc CreateUser(CreateUserRequest) returns (CreateUserResponse);
rpc AddConversationMember(AddConversationMemberRequest) returns (AddConversationMemberResponse);
```

**Kafka 消息：**
- Topic: `gochat.user-events` (type: user.guest.created)
- Topic: `gochat.conversation-events` (type: conversation.member.added)

### 1.4 Token 验证流程

**业务流程：**
```
每次请求 → 提取Authorization头 → 验证JWT签名 → 检查过期时间 → 提取用户信息 → 继续业务处理
```

**使用的 RPC 接口：**
```protobuf
// im-logic 层
rpc ValidateToken(ValidateTokenRequest) returns (ValidateTokenResponse);

message ValidateTokenRequest {
  string access_token = 1;
}

message ValidateTokenResponse {
  bool valid = 1;
  User user = 2;
  int64 expires_at = 3;
}
```

## 2. 数据同步策略

### 2.1 会话列表同步

**业务流程：**
```
客户端请求 → 获取用户最后同步时间 → 增量查询会话变更 → 返回变更数据 → 客户端合并本地数据
```

**涉及的数据模型：**
- `conversation_members` 表：基于 updated_at 增量查询
- `conversations` 表：获取会话详细信息
- `user_read_pointers` 表：获取未读数信息

**HTTP API：**
```
GET /conversations?last_sync={timestamp}&limit=50
```

**使用的 RPC 接口：**
```protobuf
// im-logic 层
rpc GetConversationsOptimized(GetConversationsOptimizedRequest) returns (GetConversationsOptimizedResponse);

message GetConversationsOptimizedRequest {
  string user_id = 1;
  int64 last_sync_time = 2;  // Unix时间戳
  int32 limit = 3;
  bool include_last_message = 4;
}

// im-repo 层
rpc GetUserConversationsWithDetails(GetUserConversationsWithDetailsRequest) 
    returns (GetUserConversationsWithDetailsResponse);
```

### 2.2 消息增量同步

**业务流程：**
```
客户端指定会话和起始位置 → 基于seq_id增量拉取 → 返回新消息列表 → 客户端去重合并
```

**涉及的数据模型：**
- `messages` 表：基于 seq_id 范围查询
- `users` 表：获取消息发送者信息

**HTTP API：**
```
GET /conversations/{conversation_id}/messages?after_seq={seq_id}&limit=50
```

**使用的 RPC 接口：**
```protobuf
// im-logic 层
rpc GetMessages(GetMessagesRequest) returns (GetMessagesResponse);

message GetMessagesRequest {
  string user_id = 1;
  string conversation_id = 2;
  int64 start_seq_id = 3;        // 起始序列号
  int32 limit = 5;
  bool ascending = 6;            // 时间顺序
}

// im-repo 层  
rpc GetConversationMessages(GetConversationMessagesRequest) returns (GetConversationMessagesResponse);
```

**数据库查询：**
```sql
SELECT m.id, m.sender_id, m.content, m.seq_id, m.created_at,
       u.username, u.avatar_url
FROM messages m
JOIN users u ON m.sender_id = u.id
WHERE m.conversation_id = ? 
  AND m.seq_id > ?
  AND m.deleted = false
ORDER BY m.seq_id ASC
LIMIT ?;
```

### 2.3 未读数实时同步

**业务流程：**
```
登录时批量获取 → 新消息到达时增量更新 → 标记已读时重新计算 → 实时推送变更
```

**涉及的数据模型：**
- `user_read_pointers` 表：存储用户已读位置
- `messages` 表：基于 seq_id 计算未读数量
- Redis: `user:unread:{user_id}` 缓存未读数映射

**使用的 RPC 接口：**
```protobuf
// im-logic 层
rpc GetUnreadCount(GetUnreadCountRequest) returns (GetUnreadCountResponse);
rpc MarkAsRead(MarkAsReadRequest) returns (MarkAsReadResponse);

// im-repo 层
rpc BatchGetUnreadCounts(BatchGetUnreadCountsRequest) returns (BatchGetUnreadCountsResponse);
rpc UpdateReadPointer(UpdateReadPointerRequest) returns (UpdateReadPointerResponse);

message BatchGetUnreadCountsRequest {
  string user_id = 1;
  repeated string conversation_ids = 2;
}
```

## 3. 好友关系管理

### 3.1 发送好友申请

**业务流程：**
```
A发起申请 → 验证权限（游客不能发申请）→ 检查重复申请 → 存储申请记录 → 发布好友事件 → 通知B
```

**涉及的数据模型：**
- `friendship_requests` 表：插入申请记录
  - `requester_id`: 申请人ID
  - `target_id`: 目标用户ID  
  - `status`: 0（待处理）
  - `message`: 申请消息

**使用的 RPC 接口：**
```protobuf
// im-logic 层（需新增）
rpc SendFriendRequest(SendFriendRequestRequest) returns (SendFriendRequestResponse);

message SendFriendRequestRequest {
  string requester_id = 1;
  string target_id = 2;
  string message = 3;
}

// im-repo 层（需新增）
rpc CreateFriendRequest(CreateFriendRequestRequest) returns (CreateFriendRequestResponse);
rpc CheckExistingFriendRequest(CheckExistingFriendRequestRequest) returns (CheckExistingFriendRequestResponse);
```

**Kafka 消息：**
```go
// 好友申请事件
type FriendEvent struct {
    EventType  string `json:"event_type"`  // "friend.request.sent"
    UserID     string `json:"user_id"`     // 申请人ID
    FriendID   string `json:"friend_id"`   // 目标用户ID  
    RequestID  string `json:"request_id"`  // 申请记录ID
    Message    string `json:"message"`     // 申请消息
    Timestamp  int64  `json:"timestamp"`
}
```

### 3.2 处理好友申请

**业务流程：**
```
B同意/拒绝申请 → 更新申请状态 → 创建双向好友关系（同意时）→ 自动创建单聊会话 → 通知双方
```

**涉及的数据模型：**
- `friendship_requests` 表：更新 status (1=同意, 2=拒绝)
- `friendship_requests` 表：创建双向好友记录（A→B 和 B→A）
- `conversations` 表：创建单聊会话（type=1）
- `conversation_members` 表：添加两个成员

**使用的 RPC 接口：**
```protobuf
// im-logic 层
rpc HandleFriendRequest(HandleFriendRequestRequest) returns (HandleFriendRequestResponse);

message HandleFriendRequestRequest {
  string request_id = 1;
  string handler_id = 2;      // 处理人ID  
  int32 action = 3;           // 1=同意, 2=拒绝
}

// im-repo 层
rpc UpdateFriendRequest(UpdateFriendRequestRequest) returns (UpdateFriendRequestResponse);
rpc CreateFriendshipRelation(CreateFriendshipRelationRequest) returns (CreateFriendshipRelationResponse);
rpc CreateConversation(CreateConversationRequest) returns (CreateConversationResponse);
```

### 3.3 好友列表同步

**业务流程：**
```
获取已确认好友列表 → 批量查询好友基本信息 → 批量查询在线状态 → 返回完整好友数据
```

**涉及的数据模型：**
- `friendship_requests` 表：查询 status=1 的好友关系
- `users` 表：获取好友基本信息
- Redis: `user:online:{user_id}` 获取在线状态

**使用的 RPC 接口：**
```protobuf
// im-logic 层
rpc GetFriends(GetFriendsRequest) returns (GetFriendsResponse);

// im-repo 层  
rpc GetUserFriends(GetUserFriendsRequest) returns (GetUserFriendsResponse);
rpc GetUsers(GetUsersRequest) returns (GetUsersResponse);                    // 批量获取用户信息
rpc BatchGetOnlineStatus(BatchGetOnlineStatusRequest) returns (BatchGetOnlineStatusResponse);

message GetUserFriendsRequest {
  string user_id = 1;
  int32 status = 2;           // 1=已确认的好友
}
```

## 4. 在线状态管理

### 4.1 状态更新流程

**上线流程：**
```
WebSocket连接建立 → 验证JWT token → 更新在线状态 → 发布上线事件 → 异步通知好友
```

**涉及的数据模型：**
- Redis: `user:online:{user_id}` 存储在线状态信息
- `friendship_requests` 表：查询好友列表用于通知

**使用的 RPC 接口：**
```protobuf
// im-repo 层
rpc SetUserOnline(SetUserOnlineRequest) returns (SetUserOnlineResponse);

message SetUserOnlineRequest {
  string user_id = 1;
  string gateway_id = 2;      // 所在网关实例ID
  string client_info = 3;     // 客户端信息
}

rpc GetUserFriends(GetUserFriendsRequest) returns (GetUserFriendsResponse);
```

**Kafka 消息：**
```go
// 用户在线事件
type UserEvent struct {
    EventType string `json:"event_type"`  // "user.online" 
    UserID    string `json:"user_id"`
    GatewayID string `json:"gateway_id"`
    Timestamp int64  `json:"timestamp"`
}
```

**下线流程：**
```
WebSocket断开 → 更新离线状态 → 延迟确认（避免频繁切换）→ 发布下线事件 → 通知好友
```

### 4.2 状态查询优化

**批量查询接口：**
```protobuf
// im-repo 层
rpc BatchGetOnlineStatus(BatchGetOnlineStatusRequest) returns (BatchGetOnlineStatusResponse);

message BatchGetOnlineStatusRequest {
  repeated string user_ids = 1;
}

message BatchGetOnlineStatusResponse {
  map<string, OnlineStatus> user_status = 1;
}

message OnlineStatus {
  string user_id = 1;
  bool is_online = 2;
  string gateway_id = 3;
  int64 last_seen = 4;
}
```

## 5. 数据一致性保证

### 5.1 消息幂等性处理

**机制：**
- 客户端生成唯一 `client_msg_id`
- 服务端检查重复，相同 `client_msg_id` 返回已存在消息

**涉及的数据模型：**
- `messages` 表：`client_msg_id` 字段建立唯一索引

**数据库查询：**
```sql
-- 检查消息是否已存在
SELECT id, seq_id FROM messages 
WHERE client_msg_id = ? AND sender_id = ?;

-- 如果不存在，插入新消息
INSERT INTO messages (conversation_id, sender_id, content, client_msg_id, ...) 
VALUES (?, ?, ?, ?, ...);
```

### 5.2 会话状态同步

**冲突检测：**
- 基于 `updated_at` 时间戳进行版本控制
- 客户端提交修改时携带上次获取的版本

**处理策略：**
```go
if clientVersion < serverVersion {
    return ConflictError{
        Code: "VERSION_CONFLICT",
        Message: "数据已被其他客户端修改，请重新获取"
    }
}
```

### 5.3 缓存一致性保证

**更新策略：**
```
数据库写入成功 → 发布缓存失效事件 → 删除相关缓存 → 下次访问时重新加载
```

**Kafka 消息：**
```go
type CacheInvalidationEvent struct {
    CacheType string   `json:"cache_type"`  // "user_conversations", "conversation_info"
    Keys      []string `json:"keys"`        // 需要失效的缓存键列表
}
```

## 6. 错误处理与重试机制

### 6.1 认证失败处理

**错误码定义：**
```protobuf
enum ErrorCode {
  ERROR_CODE_INVALID_TOKEN = 2003;        // Token无效
  ERROR_CODE_TOKEN_EXPIRED = 2004;        // Token过期
  ERROR_CODE_USER_DISABLED = 2005;        // 用户被禁用
  ERROR_CODE_INVALID_PASSWORD = 2002;     // 密码错误
}
```

**处理策略：**
- Token过期：返回401，客户端自动刷新token
- Token无效：返回403，客户端重新登录
- 用户禁用：返回423，提示账户被锁定

### 6.2 同步失败重试

**重试策略：**
```go
type RetryConfig struct {
    MaxRetries    int           `json:"max_retries"`     // 最大重试次数
    BaseDelay     time.Duration `json:"base_delay"`      // 基础延迟
    MaxDelay      time.Duration `json:"max_delay"`       // 最大延迟
    BackoffFactor float64       `json:"backoff_factor"`  // 退避因子
}

// 指数退避：1s, 2s, 4s, 8s, 16s, 30s(cap)
```

### 6.3 数据冲突解决

**解决策略：**
1. **服务器优先**：放弃客户端修改，使用服务器数据
2. **时间戳对比**：保留最新修改，丢弃较旧修改  
3. **用户选择**：提示用户手动解决冲突

## 7. 性能优化与监控

### 7.1 认证性能优化

**JWT缓存：**
```yaml
# Redis缓存已验证的token
user:token:verified:{token_hash}: {user_info}  # TTL: token剩余有效期
```

**批量操作优化：**
```sql
-- 批量查询好友信息
SELECT * FROM users WHERE id IN (?, ?, ?, ...);

-- 批量查询在线状态  
MGET user:online:123 user:online:456 user:online:789
```

### 7.2 监控指标

**认证相关：**
```yaml
gochat_auth_requests_total{type="login|register|guest"}:     # 认证请求总数
gochat_auth_failures_total{reason="invalid_password|token_expired"}:  # 认证失败数
gochat_token_validation_duration_seconds:                   # Token验证耗时
```

**同步相关：**
```yaml
gochat_sync_requests_total{type="conversations|messages|friends"}:  # 同步请求总数
gochat_sync_duration_seconds{type="conversations|messages"}:        # 同步耗时
gochat_cache_operations_total{operation="hit|miss|evict"}:           # 缓存操作数
```

**告警规则：**
```yaml
- alert: AuthFailureRateHigh
  expr: rate(gochat_auth_failures_total[5m]) / rate(gochat_auth_requests_total[5m]) > 0.05
  
- alert: SyncLatencyHigh  
  expr: histogram_quantile(0.95, rate(gochat_sync_duration_seconds_bucket[5m])) > 1.0

- alert: CacheHitRateLow
  expr: rate(gochat_cache_operations_total{operation="hit"}[5m]) / rate(gochat_cache_operations_total[5m]) < 0.85
```

## 8. 好友管理完整流程

### 8.1 好友申请列表查询

**业务流程：**
```
用户请求好友申请列表 → 查询收到的申请 → 返回申请详情和发送者信息
```

**涉及的数据模型：**
- `friendship_requests` 表：查询 target_id=当前用户 的申请
- `users` 表：获取申请发送者信息

**使用的 RPC 接口：**
```protobuf
// im-logic 层（需新增）
rpc GetFriendRequests(GetFriendRequestsRequest) returns (GetFriendRequestsResponse);

message GetFriendRequestsRequest {
  string user_id = 1;
  int32 status = 2;           // 0=待处理, 1=已同意, 2=已拒绝
  int32 type = 3;             // 1=收到的申请, 2=发出的申请
}

// im-repo 层（需新增）
rpc GetUserFriendRequests(GetUserFriendRequestsRequest) returns (GetUserFriendRequestsResponse);
```

**HTTP API：**
```
GET /friends/requests?type=received&status=0
```

### 8.2 好友关系数据模型优化

**friendship_requests 表的完整设计：**
```sql
CREATE TABLE friendship_requests (
    id BIGINT PRIMARY KEY,
    requester_id BIGINT NOT NULL,           -- 申请发送者
    target_id BIGINT NOT NULL,              -- 申请接收者  
    status INT NOT NULL DEFAULT 0,          -- 0:待处理, 1:已同意, 2:已拒绝
    requester_remarks VARCHAR(100),         -- 申请者备注名
    target_remarks VARCHAR(100),            -- 接收者备注名
    message TEXT,                           -- 申请消息
    created_at DATETIME NOT NULL,
    updated_at DATETIME NOT NULL,
    
    -- 索引优化
    UNIQUE KEY uk_requester_target (requester_id, target_id),
    INDEX idx_target_status (target_id, status),
    INDEX idx_requester_status (requester_id, status)
);
```

**好友关系的双向记录策略：**
```sql
-- 同意好友申请时，创建双向记录
INSERT INTO friendship_requests (requester_id, target_id, status, created_at) VALUES
(user_a, user_b, 1, NOW()),
(user_b, user_a, 1, NOW());
```

### 8.3 好友列表增量同步

**业务流程：**
```
客户端请求好友列表 → 基于last_sync_time增量查询 → 返回新增/更新/删除的好友
```

**使用的 RPC 接口：**
```protobuf
// im-logic 层
rpc GetFriendsWithSync(GetFriendsWithSyncRequest) returns (GetFriendsWithSyncResponse);

message GetFriendsWithSyncRequest {
  string user_id = 1;
  int64 last_sync_time = 2;        // 上次同步时间戳
  bool include_online_status = 3;   // 是否包含在线状态
}

// im-repo 层
rpc GetUserFriendsWithSync(GetUserFriendsWithSyncRequest) returns (GetUserFriendsWithSyncResponse);
```

**数据库增量查询：**
```sql
SELECT fr.*, u.username, u.avatar_url, u.is_guest
FROM friendship_requests fr
JOIN users u ON fr.target_id = u.id  
WHERE fr.requester_id = ? 
  AND fr.status = 1 
  AND fr.updated_at > ?
ORDER BY fr.updated_at DESC;
```

## 9. Token 刷新机制

### 9.1 Token 刷新流程

**业务流程：**
```
Access Token 即将过期 → 使用 Refresh Token 请求新令牌 → 验证 Refresh Token → 生成新的 Token 对
```

**使用的 RPC 接口：**
```protobuf
// im-logic 层
rpc RefreshToken(RefreshTokenRequest) returns (RefreshTokenResponse);

message RefreshTokenRequest {
  string refresh_token = 1;
}

message RefreshTokenResponse {
  string access_token = 1;
  string refresh_token = 2;      // 新的刷新令牌
  int64 expires_in = 3;
}
```

**Token 存储策略：**
- Redis: `refresh_token:{token_hash}` 存储令牌信息（TTL: 30天）
- 旧的 Refresh Token 在使用后立即失效

### 9.2 Token 黑名单机制

**业务流程：**
```
用户登出/强制下线 → 将 Token 加入黑名单 → 后续请求检查黑名单 → 拒绝已失效 Token
```

**Redis 黑名单存储：**
```yaml
token:blacklist:{token_hash}: 1    # TTL: token剩余有效期
```

通过明确的数据模型、RPC接口和错误处理机制，确保认证和同步流程的可靠性和性能。