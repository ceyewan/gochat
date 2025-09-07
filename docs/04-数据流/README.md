# 服务数据流文档

## 概述

GoChat 系统的数据流设计基于微服务架构，通过消息队列实现服务间的异步通信和数据流转。本文档详细描述了系统中的各种数据流和业务流程。

## 数据流架构

### 整体架构图

```
┌─────────────┐    ┌─────────────┐    ┌─────────────┐    ┌─────────────┐
│   Client    │    │ im-gateway  │    │  im-logic   │    │  im-repo    │
│             │    │             │    │             │    │             │
│  ┌─────────┐│    │  ┌─────────┐│    │  ┌─────────┐│    │  ┌─────────┐│
│  │WebSocket││◀──▶│  │ HTTP    ││◀──▶│  │ gRPC    ││◀──▶│  │ MySQL   ││
│  │         ││    │  │         ││    │  │         ││    │  │         ││
│  └─────────┘│    │  └─────────┘│    │  └─────────┘│    │  └─────────┘│
│             │    │             │    │             │    │             │
│  ┌─────────┐│    │  ┌─────────┐│    │  ┌─────────┐│    │  ┌─────────┐│
│  │   API   ││    │  │ Kafka   ││    │  │ Kafka   ││    │  │ Redis   ││
│  │         ││    │  │Producer ││    │  │Consumer ││    │  │ Cache   ││
│  └─────────┘│    │  └─────────┘│    │  └─────────┘│    │  └─────────┘│
└─────────────┘    └─────────────┘    └─────────────┘    └─────────────┘
       │                   │                   │                   │
       │                   │                   │                   │
       └───────────────────┼───────────────────┼───────────────────┘
                           │                   │
                           │        ┌─────────────┐
                           │        │  im-task    │
                           │        │             │
                           │        │  ┌─────────┐│
                           │        │  │ Kafka   ││
                           │        │  │Consumer ││
                           │        │  └─────────┘│
                           │        │             │
                           │        │  ┌─────────┐│
                           │        │  │ gRPC    ││
                           │        │  │ Client  ││
                           │        │  └─────────┘│
                           │        └─────────────┘
                           │
                           └───────────────────┘
```

## 核心数据流

### 1. 用户认证数据流

#### 登录流程

```
Client → im-gateway → im-logic → im-repo → MySQL → Redis
```

#### 详细流程

1. **客户端请求**
   - 用户发送登录请求到 im-gateway
   - 包含用户名、密码、设备ID

2. **网关处理**
   - im-gateway 验证请求格式
   - 转发到 im-logic 服务

3. **逻辑处理**
   - im-logic 调用 im-repo 验证用户凭据
   - 生成 JWT Token
   - 记录登录日志

4. **数据存储**
   - 更新用户登录状态
   - 缓存用户信息到 Redis
   - 记录设备信息

#### 代码实现

```go
// 登录处理流程
func (s *LogicServer) handleLogin(ctx context.Context, req *AuthRequest) (*AuthResponse, error) {
    // 1. 验证用户凭据
    user, err := s.repoClient.GetUser(ctx, &pb.GetUserRequest{
        Username: req.Username,
    })
    if err != nil {
        return nil, err
    }
    
    // 2. 验证密码
    if !validatePassword(req.Password, user.Password) {
        return nil, errors.New("invalid password")
    }
    
    // 3. 生成 Token
    accessToken, err := generateAccessToken(user.UserId)
    if err != nil {
        return nil, err
    }
    
    refreshToken, err := generateRefreshToken(user.UserId)
    if err != nil {
        return nil, err
    }
    
    // 4. 更新登录状态
    s.updateUserLoginStatus(ctx, user.UserId, req.DeviceId)
    
    // 5. 记录登录日志
    s.recordLoginLog(ctx, user.UserId, req.DeviceId, "success")
    
    return &AuthResponse{
        Success:      true,
        AccessToken:  accessToken,
        RefreshToken: refreshToken,
        ExpiresIn:    7200,
        UserInfo:     user,
    }, nil
}
```

### 2. 消息发送数据流

#### 单聊消息流程

```
Client A → im-gateway → Kafka → im-logic → im-repo → Kafka → im-gateway → Client B
```

#### 详细流程

1. **消息发送**
   - Client A 发送消息到 im-gateway
   - im-gateway 验证用户权限
   - 发送到 Kafka 上游 Topic

2. **消息处理**
   - im-logic 消费消息
   - 验证消息内容
   - 调用 im-repo 持久化

3. **消息路由**
   - 查询目标用户网关
   - 发送到下游 Topic
   - im-gateway 推送给 Client B

#### 代码实现

```go
// 消息处理流程
func (s *LogicServer) processMessage(ctx context.Context, msg *KafkaMessage) error {
    // 1. 验证消息
    if err := s.validateMessage(msg); err != nil {
        return err
    }
    
    // 2. 持久化消息
    message, err := s.repoClient.CreateMessage(ctx, &pb.CreateMessageRequest{
        ConversationId: msg.ConversationID,
        FromUserId:     msg.FromUserID,
        ToUserId:       msg.ToUserID,
        Content:        msg.Content,
        MessageType:    msg.MessageType,
    })
    if err != nil {
        return err
    }
    
    // 3. 路由消息
    if err := s.routeMessage(ctx, message); err != nil {
        return err
    }
    
    // 4. 更新会话
    s.updateConversation(ctx, msg.ConversationID, message.MessageId)
    
    return nil
}
```

### 3. 群聊消息数据流

#### 小群消息流程

```
Client A → im-gateway → Kafka → im-logic → 多个下游 Topic → 多个 im-gateway → 多个 Client
```

#### 大群消息流程

```
Client A → im-gateway → Kafka → im-logic → Kafka → im-task → 批量下游 Topic → 多个 im-gateway → 多个 Client
```

#### 详细流程

1. **消息发送**
   - Client A 发送群聊消息
   - im-gateway 转发到上游 Topic

2. **群组判断**
   - im-logic 检查群组大小
   - 小群直接路由，大群异步处理

3. **消息扇出**
   - 获取群组成员列表
   - 批量发送到各成员网关

#### 代码实现

```go
// 群聊消息处理
func (s *LogicServer) handleGroupMessage(ctx context.Context, msg *KafkaMessage) error {
    // 1. 获取群组信息
    group, err := s.repoClient.GetGroup(ctx, &pb.GetGroupRequest{
        GroupId: msg.ConversationID,
    })
    if err != nil {
        return err
    }
    
    // 2. 根据群组大小选择处理方式
    if group.CurrentMembers < 100 {
        return s.handleSmallGroupMessage(ctx, msg, group)
    } else {
        return s.handleLargeGroupMessage(ctx, msg, group)
    }
}

// 大群消息处理
func (s *LogicServer) handleLargeGroupMessage(ctx context.Context, msg *KafkaMessage, group *pb.Group) error {
    // 1. 发送到 Task 服务
    taskMsg := &TaskMessage{
        TaskType: "group_fanout",
        Payload: map[string]interface{}{
            "group_id":        msg.ConversationID,
            "message_id":      msg.MessageID,
            "from_user_id":    msg.FromUserID,
            "content":         msg.Content,
            "message_type":    msg.MessageType,
            "exclude_user_id": msg.FromUserID,
        },
        Timestamp: time.Now().Unix(),
    }
    
    return s.kafkaProducer.SendMessage("im-task-topic", taskMsg)
}
```

### 4. 离线消息数据流

#### 离线消息流程

```
im-logic → Kafka → im-task → im-repo → 推送服务 → 客户端
```

#### 详细流程

1. **离线检测**
   - im-logic 检测用户离线
   - 发送离线处理任务

2. **消息持久化**
   - im-task 持久化消息
   - 更新消息状态

3. **推送通知**
   - 发送推送通知
   - 记录推送日志

#### 代码实现

```go
// 离线消息处理
func (s *TaskServer) handleOfflineMessage(ctx context.Context, msg *KafkaMessage) error {
    // 1. 持久化消息
    _, err := s.repoClient.CreateMessage(ctx, &pb.CreateMessageRequest{
        ConversationId: msg.ConversationID,
        FromUserId:     msg.FromUserID,
        ToUserId:       msg.ToUserID,
        Content:        msg.Content,
        MessageType:    msg.MessageType,
    })
    if err != nil {
        return err
    }
    
    // 2. 发送推送通知
    pushMsg := &PushMessage{
        UserId:     msg.ToUserID,
        Title:      "新消息",
        Content:    fmt.Sprintf("%s: %s", msg.FromUserID, msg.Content),
        MessageType: msg.MessageType,
        Timestamp:  time.Now().Unix(),
    }
    
    return s.pushService.SendNotification(ctx, pushMsg)
}
```

### 5. 会话管理数据流

#### 创建会话流程

```
Client → im-gateway → im-logic → im-repo → MySQL → Redis
```

#### 详细流程

1. **会话创建**
   - 客户端请求创建会话
   - im-logic 验证权限
   - im-repo 创建会话记录

2. **成员管理**
   - 添加会话成员
   - 通知相关用户

3. **缓存更新**
   - 更新会话缓存
   - 更新用户会话列表

#### 代码实现

```go
// 创建会话
func (s *LogicServer) createConversation(ctx context.Context, req *CreateConversationRequest) (*CreateConversationResponse, error) {
    // 1. 验证用户权限
    for _, userID := range req.UserIds {
        if err := s.validateUserPermission(ctx, userID); err != nil {
            return nil, err
        }
    }
    
    // 2. 创建会话
    conversation, err := s.repoClient.CreateConversation(ctx, &pb.CreateConversationRequest{
        Type:           req.Type,
        Name:           req.Name,
        ParticipantIds: req.UserIds,
    })
    if err != nil {
        return nil, err
    }
    
    // 3. 通知成员
    for _, userID := range req.UserIds {
        s.notifyConversationCreated(ctx, userID, conversation)
    }
    
    // 4. 更新缓存
    s.updateConversationCache(ctx, conversation)
    
    return &CreateConversationResponse{
        ConversationId: conversation.ConversationId,
        Conversation:   conversation,
    }, nil
}
```

### 6. 群组管理数据流

#### 创建群组流程

```
Client → im-gateway → im-logic → im-repo → MySQL → Redis → 通知成员
```

#### 详细流程

1. **群组创建**
   - 客户端请求创建群组
   - im-logic 验证权限
   - im-repo 创建群组记录

2. **成员管理**
   - 添加群组成员
   - 设置群组权限

3. **通知发送**
   - 通知被邀请成员
   - 更新群组缓存

#### 代码实现

```go
// 创建群组
func (s *LogicServer) createGroup(ctx context.Context, req *CreateGroupRequest) (*CreateGroupResponse, error) {
    // 1. 验证创建者权限
    if err := s.validateUserPermission(ctx, req.OwnerId); err != nil {
        return nil, err
    }
    
    // 2. 创建群组
    group, err := s.repoClient.CreateGroup(ctx, &pb.CreateGroupRequest{
        Name:        req.Name,
        Description: req.Description,
        Avatar:      req.Avatar,
        OwnerId:     req.OwnerId,
        MaxMembers:  req.MaxMembers,
    })
    if err != nil {
        return nil, err
    }
    
    // 3. 添加成员
    for _, userID := range req.MemberIds {
        s.repoClient.AddGroupMember(ctx, &pb.AddGroupMemberRequest{
            GroupId: group.GroupId,
            UserId:  userID,
            Role:    "member",
        })
    }
    
    // 4. 通知成员
    for _, userID := range req.MemberIds {
        s.notifyGroupInvitation(ctx, userID, group)
    }
    
    // 5. 更新缓存
    s.updateGroupCache(ctx, group)
    
    return &CreateGroupResponse{
        GroupId: group.GroupId,
        Group:   group,
    }, nil
}
```

## 数据缓存策略

### 1. 多级缓存架构

```
┌─────────────┐    ┌─────────────┐    ┌─────────────┐
│   Client    │    │ im-gateway  │    │  im-logic   │
│             │    │             │    │             │
│  ┌─────────┐│    │  ┌─────────┐│    │  ┌─────────┐│
│  │ Local   ││    │  │ Local   ││    │  │ Local   ││
│  │ Cache   ││    │  │ Cache   ││    │  │ Cache   ││
│  └─────────┘│    │  └─────────┘│    │  └─────────┘│
│             │    │             │    │             │
│  ┌─────────┐│    │  ┌─────────┐│    │  ┌─────────┐│
│  │ Redis   ││    │  │ Redis   ││    │  │ Redis   ││
│  │ Cache   ││    │  │ Cache   ││    │  │ Cache   ││
│  └─────────┘│    │  └─────────┘│    │  └─────────┘│
└─────────────┘    └─────────────┘    └─────────────┘
```

### 2. 缓存键设计

```go
// 用户信息缓存
const UserCacheKey = "user:%s"

// 会话信息缓存
const ConversationCacheKey = "conversation:%s"

// 群组信息缓存
const GroupCacheKey = "group:%s"

// 用户网关映射缓存
const UserGatewayCacheKey = "user_gateway:%s"

// 消息缓存
const MessageCacheKey = "message:%s"

// 用户会话列表缓存
const UserConversationsCacheKey = "user_conversations:%s"
```

### 3. 缓存更新策略

```go
// 缓存更新接口
type CacheUpdater interface {
    UpdateUserCache(ctx context.Context, user *pb.User) error
    UpdateConversationCache(ctx context.Context, conversation *pb.Conversation) error
    UpdateGroupCache(ctx context.Context, group *pb.Group) error
    UpdateMessageCache(ctx context.Context, message *pb.Message) error
    InvalidateUserCache(ctx context.Context, userID string) error
    InvalidateConversationCache(ctx context.Context, conversationID string) error
    InvalidateGroupCache(ctx context.Context, groupID string) error
}

// 缓存更新实现
func (s *CacheService) UpdateUserCache(ctx context.Context, user *pb.User) error {
    // 1. 序列化用户数据
    userData, err := json.Marshal(user)
    if err != nil {
        return err
    }
    
    // 2. 更新缓存
    key := fmt.Sprintf(UserCacheKey, user.UserId)
    return s.redisClient.Set(ctx, key, userData, time.Hour*24).Err()
}
```

## 数据一致性保证

### 1. 事务管理

```go
// 事务管理接口
type TransactionManager interface {
    WithTransaction(ctx context.Context, fn func(ctx context.Context) error) error
}

// 事务管理实现
func (s *TransactionManager) WithTransaction(ctx context.Context, fn func(ctx context.Context) error) error {
    // 1. 开始事务
    tx := s.db.Begin()
    defer func() {
        if r := recover(); r != nil {
            tx.Rollback()
        }
    }()
    
    // 2. 创建事务上下文
    txCtx := context.WithValue(ctx, "transaction", tx)
    
    // 3. 执行事务函数
    if err := fn(txCtx); err != nil {
        tx.Rollback()
        return err
    }
    
    // 4. 提交事务
    return tx.Commit().Error
}
```

### 2. 数据同步

```go
// 数据同步接口
type DataSync interface {
    SyncUserToCache(ctx context.Context, userID string) error
    SyncConversationToCache(ctx context.Context, conversationID string) error
    SyncGroupToCache(ctx context.Context, groupID string) error
}

// 数据同步实现
func (s *DataSyncService) SyncUserToCache(ctx context.Context, userID string) error {
    // 1. 从数据库获取用户信息
    user, err := s.repoClient.GetUser(ctx, &pb.GetUserRequest{
        UserId: userID,
    })
    if err != nil {
        return err
    }
    
    // 2. 更新缓存
    return s.cacheUpdater.UpdateUserCache(ctx, user)
}
```

### 3. 最终一致性

```go
// 最终一致性检查
func (s *ConsistencyChecker) CheckDataConsistency(ctx context.Context) error {
    // 1. 检查用户数据一致性
    if err := s.checkUserDataConsistency(ctx); err != nil {
        return err
    }
    
    // 2. 检查会话数据一致性
    if err := s.checkConversationDataConsistency(ctx); err != nil {
        return err
    }
    
    // 3. 检查群组数据一致性
    if err := s.checkGroupDataConsistency(ctx); err != nil {
        return err
    }
    
    return nil
}
```

## 数据监控和追踪

### 1. 数据流监控

```go
// 数据流监控指标
type DataFlowMetrics struct {
    // 消息处理指标
    MessagesProcessed     prometheus.Counter
    MessagesFailed        prometheus.Counter
    MessageLatency        prometheus.Histogram
    
    // 数据库操作指标
    DBQueries             prometheus.Counter
    DBQueryLatency        prometheus.Histogram
    
    // 缓存操作指标
    CacheHits            prometheus.Counter
    CacheMisses          prometheus.Counter
    CacheLatency         prometheus.Histogram
}

// 监控消息处理
func (m *DataFlowMetrics) ObserveMessageProcessing(duration time.Duration, success bool) {
    m.MessageLatency.Observe(duration.Seconds())
    if success {
        m.MessagesProcessed.Inc()
    } else {
        m.MessagesFailed.Inc()
    }
}
```

### 2. 链路追踪

```go
// 链路追踪中间件
func (s *Server) TracingMiddleware(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (interface{}, error) {
    // 1. 开始追踪
    span := trace.StartSpan(ctx, info.FullMethod)
    defer span.End()
    
    // 2. 添加追踪信息
    ctx = trace.NewContext(ctx, span)
    
    // 3. 处理请求
    resp, err := handler(ctx, req)
    
    // 4. 记录结果
    if err != nil {
        span.SetStatus(trace.Status{
            Code:    trace.StatusCodeInternal,
            Message: err.Error(),
        })
    }
    
    return resp, err
}
```

## 数据安全

### 1. 数据加密

```go
// 数据加密服务
type EncryptionService struct {
    key []byte
}

// 加密敏感数据
func (s *EncryptionService) Encrypt(data string) (string, error) {
    block, err := aes.NewCipher(s.key)
    if err != nil {
        return "", err
    }
    
    gcm, err := cipher.NewGCM(block)
    if err != nil {
        return "", err
    }
    
    nonce := make([]byte, gcm.NonceSize())
    if _, err = io.ReadFull(rand.Reader, nonce); err != nil {
        return "", err
    }
    
    encrypted := gcm.Seal(nonce, nonce, []byte(data), nil)
    return base64.StdEncoding.EncodeToString(encrypted), nil
}

// 解密数据
func (s *EncryptionService) Decrypt(encrypted string) (string, error) {
    data, err := base64.StdEncoding.DecodeString(encrypted)
    if err != nil {
        return "", err
    }
    
    block, err := aes.NewCipher(s.key)
    if err != nil {
        return "", err
    }
    
    gcm, err := cipher.NewGCM(block)
    if err != nil {
        return "", err
    }
    
    nonceSize := gcm.NonceSize()
    if len(data) < nonceSize {
        return "", errors.New("encrypted data too short")
    }
    
    nonce, ciphertext := data[:nonceSize], data[nonceSize:]
    decrypted, err := gcm.Open(nil, nonce, ciphertext, nil)
    if err != nil {
        return "", err
    }
    
    return string(decrypted), nil
}
```

### 2. 数据脱敏

```go
// 数据脱敏服务
type MaskingService struct{}

// 脱敏手机号
func (s *MaskingService) MaskPhone(phone string) string {
    if len(phone) != 11 {
        return phone
    }
    return phone[:3] + "****" + phone[7:]
}

// 脱敏邮箱
func (s *MaskingService) MaskEmail(email string) string {
    parts := strings.Split(email, "@")
    if len(parts) != 2 {
        return email
    }
    
    username := parts[0]
    domain := parts[1]
    
    if len(username) <= 2 {
        return username + "@" + domain
    }
    
    return username[:2] + "***@" + domain
}
```

## 总结

GoChat 系统的数据流设计采用了微服务架构，通过消息队列实现服务间的异步通信。系统设计了完善的数据流处理机制，包括用户认证、消息发送、群聊处理、离线消息等核心流程。同时，系统还提供了数据缓存、一致性保证、监控追踪和安全防护等机制，确保数据流的可靠性和安全性。