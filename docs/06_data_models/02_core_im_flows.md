# GoChat 核心业务流程与数据流

本文档详细阐述了基于统一会话抽象设计的 GoChat 系统中各项核心 IM 功能的实现流程。系统将单聊、群聊、世界聊天室统一为 conversation 概念，通过事件驱动架构实现业务解耦和高可扩展性。

## 核心理念：统一会话抽象 + 事件驱动

系统采用**统一会话抽象**设计，所有社交交互都基于 `conversation` 概念：
- **单聊**：两人之间的 conversation (type=1)
- **群聊**：多人参与的 conversation (type=2)  
- **世界聊天室**：特殊的群聊 conversation (type=3)

同时遵循**事件驱动**设计模式，核心业务服务完成本职工作后发布领域事件到 Kafka，`im-task` 服务作为异步任务处理中心订阅这些事件并处理衍生任务，实现业务解耦和系统弹性。

---

## 1. 统一会话创建流程

**目标**: 展示单聊、群聊创建的统一处理流程，体现"添加好友 = 创建单聊"的设计理念。

### 1.1 好友申请 = 单聊会话邀请

```mermaid
sequenceDiagram
    participant User A as 用户 A
    participant Gateway as im-gateway
    participant Logic as im-logic
    participant Repo as im-repo
    participant Kafka
    participant Task as im-task
    participant User B as 用户 B

    User A->>+Gateway: POST /friends/requests (添加好友申请)
    Gateway->>+Logic: RPC: SendFriendRequest(target_user_id)
    Logic->>Logic: 权限验证 (游客无法发送好友申请)
    
    Logic->>+Repo: RPC: CreateFriendRequest(requester_id, target_id)
    Repo->>Repo: 写入 friendship_requests 表 (status=0)
    Repo-->>-Logic: 返回申请记录

    Logic->>+Kafka: Produce 'gochat.friend-events' (type: friend.request.sent)
    Kafka-->>-Logic: acks
    Logic-->>-Gateway: 返回成功响应
    Gateway-->>-User A: HTTP 201 Created

    par 异步处理好友申请通知
        Kafka->>+Task: Consume 'gochat.friend-events'
        Task->>Task: 构建 SystemNotificationEvent
        Task->>+Kafka: Produce 'gochat.system-notifications' (type: friend.request.new)
        Task->>+Gateway: RPC: PushToUsers([B], notification)
        Gateway-->>-Task: acks
        Gateway->>User B: 推送好友申请通知
        Task-->>-Kafka: Commit offset
    end
```

### 1.2 好友申请同意 = 自动创建单聊会话

```mermaid
sequenceDiagram
    participant User B as 用户 B  
    participant Gateway as im-gateway
    participant Logic as im-logic
    participant Repo as im-repo
    participant Kafka
    participant Task as im-task
    participant User A as 用户 A

    User B->>+Gateway: PUT /friends/requests/{id} (action: accept)
    Gateway->>+Logic: RPC: HandleFriendRequest(accept)
    
    par 并行处理
        Logic->>+Repo: RPC: UpdateFriendRequest(status=1)
        Repo->>Repo: 更新 friendship_requests 状态
        Repo->>Repo: 创建双向好友关系记录
        Repo-->>-Logic: 更新成功

    and 自动创建单聊会话
        Logic->>+Repo: RPC: CreateConversation(type=1, members=[A,B])
        Repo->>Repo: 写入 conversations 表 (type=1)
        Repo->>Repo: 写入 conversation_members 表 (两条记录)
        Repo-->>-Logic: 返回会话信息
    end

    Logic->>+Kafka: Produce 'gochat.friend-events' (type: friend.request.accepted)
    Logic->>+Kafka: Produce 'gochat.conversation-events' (type: conversation.created)
    Kafka-->>-Logic: acks
    Logic-->>-Gateway: 返回成功响应
    Gateway-->>-User B: HTTP 200 OK

    par 异步通知处理
        Kafka->>+Task: Consume friend + conversation events
        Task->>Task: 构建双重通知 (好友添加成功 + 新会话)
        Task->>+Gateway: RPC: PushToUsers([A,B], signals)
        Gateway->>User A: 推送好友添加成功 + 新会话通知
        Gateway->>User B: 推送会话创建确认
        Task-->>-Kafka: Commit offset
    end
```

### 1.3 群聊创建流程

```mermaid
sequenceDiagram
    participant Creator as 创建者
    participant Gateway as im-gateway
    participant Logic as im-logic
    participant Repo as im-repo
    participant Kafka
    participant Task as im-task
    participant Members as 邀请成员

    Creator->>+Gateway: POST /conversations (type=2, 群聊创建)
    Gateway->>+Logic: RPC: CreateConversation(type=2)
    Logic->>Logic: 权限验证 (游客无法创建群聊)
    
    Logic->>+Repo: RPC: CreateConversation(type=2, name, members)
    Repo->>Repo: 写入 conversations 表 (type=2, owner_id=creator)
    Repo->>Repo: 批量写入 conversation_members 表 (创建者=owner)
    Repo-->>-Logic: 返回群聊会话信息

    Logic->>+Kafka: Produce 'gochat.conversation-events' (type: conversation.created)
    Kafka-->>-Logic: acks
    Logic-->>-Gateway: 返回群聊信息
    Gateway-->>-Creator: HTTP 201 Created

    par 异步邀请通知处理
        Kafka->>+Task: Consume conversation events
        Task->>+Repo: RPC: GetConversationMembers(conversation_id)
        Repo-->>-Task: 返回成员列表 (包括创建者)
        
        loop 为每个被邀请成员
            Task->>Task: 构建 SystemNotificationEvent (conversation.invited)
            Task->>+Gateway: RPC: PushToUsers([member], invitation)
            Gateway->>Members: 推送群聊邀请通知
        end
        
        Task-->>-Kafka: Commit offset
    end
```

---

## 2. 统一消息收发流程

**目标**: 展示单聊、群聊、世界聊天室消息的统一处理流程，体现架构的一致性。

### 2.1 统一消息发送流程

```mermaid
sequenceDiagram
    participant Sender as 发送者
    participant Gateway as im-gateway
    participant Logic as im-logic
    participant Kafka
    participant Task as im-task
    participant Repo as im-repo
    participant Recipients as 接收者

    Sender->>+Gateway: WebSocket: send-message
    Gateway->>Gateway: 提取 conversation_id, content, message_type
    Gateway->>+Kafka: Produce 'gochat.messages.upstream'
    Kafka-->>-Gateway: acks
    Gateway-->>-Sender: WebSocket: message-ack (临时确认)

    Kafka->>+Logic: Consume 'gochat.messages.upstream'
    Logic->>Logic: 业务逻辑处理
    
    par 权限验证与消息生成
        Logic->>+Repo: RPC: ValidateUserInConversation(user_id, conversation_id)
        Repo-->>-Logic: 验证通过 + 会话类型信息
        Logic->>+Repo: RPC: GenerateSeqID(conversation_id)
        Repo-->>-Logic: 返回 seq_id
        Logic->>Logic: 生成 message_id, 构建完整消息
    end

    par 并行处理: 持久化 + 推送
        Logic->>+Kafka: Produce 'gochat.messages.persist' (持久化优先)
        
    and 推送策略 (基于会话类型和规模)
        alt 单聊 (type=1)
            Logic->>+Repo: RPC: GetConversationMembers(conversation_id)
            Repo-->>-Logic: 返回对方用户 + 在线状态
            Logic->>+Kafka: Produce 'gochat.messages.downstream.{gateway_id}'
        else 中小群 (type=2, member_count <= 500)
            Logic->>+Repo: RPC: BatchGetOnlineStatus(member_ids)
            Repo-->>-Logic: 返回在线成员 + 网关分布
            Logic->>+Kafka: 批量 Produce 'gochat.messages.downstream.{gateway_id}'
        else 大群/世界聊天室 (member_count > 500 或 type=3)
            Logic->>+Kafka: Produce 'gochat.tasks.fanout' (异步扇出)
        end
    end

    par 消息持久化处理
        Kafka->>+Task: Consume 'gochat.messages.persist'
        Task->>+Repo: RPC: SaveMessage(message_data)
        Repo->>Repo: 写入 messages 表
        Repo->>Repo: 更新 conversations.last_message_id
        Repo-->>-Task: 持久化成功
        Task-->>-Kafka: Commit offset (持久化完成)
    end

    par 消息推送处理
        Kafka->>+Gateway: Consume 'gochat.messages.downstream.{instance_id}'
        Gateway->>Gateway: 查找目标用户连接
        Gateway->>Recipients: WebSocket: new-message
        Gateway-->>-Kafka: Commit offset
    end

    opt 大群异步扇出处理
        Kafka->>+Task: Consume 'gochat.tasks.fanout'
        Task->>+Repo: RPC: GetConversationMembers(conversation_id, batch_size=1000)
        
        loop 分批处理成员
            Task->>+Repo: RPC: BatchGetOnlineStatus(member_batch)
            Repo-->>-Task: 返回在线状态 + 网关分布
            Task->>+Kafka: 批量 Produce 'gochat.messages.downstream.{gateway_id}'
        end
        
        Task-->>-Kafka: Commit offset (扇出完成)
    end
```

### 2.2 已读状态更新流程

```mermaid
sequenceDiagram
    participant Reader as 读者
    participant Gateway as im-gateway
    participant Logic as im-logic
    participant Repo as im-repo
    participant Kafka
    participant Task as im-task
    participant Sender as 原发送者

    Reader->>+Gateway: PUT /conversations/{id}/read (seqId=12345)
    Gateway->>+Logic: RPC: MarkAsRead(conversation_id, seq_id)
    
    Logic->>+Repo: RPC: UpdateReadPointer(user_id, conversation_id, seq_id)
    Repo->>Repo: 更新 user_read_pointers 表
    Repo->>Repo: 计算更新后的未读数
    Repo-->>-Logic: 返回新未读数

    Logic->>+Kafka: Produce 'gochat.message-events' (type: message.read)
    Kafka-->>-Logic: acks
    Logic-->>-Gateway: 返回未读数
    Gateway-->>-Reader: HTTP 200 OK (unreadCount: 0)

    par 异步已读回执处理
        Kafka->>+Task: Consume 'gochat.message-events'
        Task->>+Repo: RPC: GetMessagesInRange(conversation_id, seq_id范围)
        Repo-->>-Task: 返回受影响的消息 + 发送者信息
        
        Task->>Task: 构建 SystemSignalMessage (type: message.read)
        Task->>Task: 去重发送者列表 (避免重复通知)
        Task->>+Gateway: RPC: PushToUsers(senders, read_receipt)
        Gateway->>Sender: WebSocket: message-read-receipt
        Task-->>-Kafka: Commit offset
    end
```

---

## 3. 会话成员管理流程

**目标**: 展示群聊成员的添加、移除、角色变更等管理操作的统一处理。

### 3.1 添加会话成员流程

```mermaid
sequenceDiagram
    participant Admin as 管理员
    participant Gateway as im-gateway
    participant Logic as im-logic
    participant Repo as im-repo
    participant Kafka
    participant Task as im-task
    participant NewMembers as 新成员
    participant ExistingMembers as 现有成员

    Admin->>+Gateway: POST /conversations/{id}/members
    Gateway->>+Logic: RPC: AddMembers(conversation_id, user_ids)
    Logic->>Logic: 权限验证 (只有管理员/群主可以添加成员)
    
    par 批量成员验证与添加
        Logic->>+Repo: RPC: ValidateUsers(user_ids) 
        Repo-->>-Logic: 返回有效用户列表
        Logic->>+Repo: RPC: CheckConversationCapacity(conversation_id)
        Repo-->>-Logic: 检查群容量限制
        
        Logic->>+Repo: RPC: BatchAddConversationMembers(conversation_id, valid_user_ids)
        Repo->>Repo: 批量写入 conversation_members 表
        Repo->>Repo: 更新 conversations.member_count
        Repo-->>-Logic: 返回成功添加的用户列表 + 失败列表
    end

    Logic->>+Kafka: Produce 'gochat.conversation-events' (type: conversation.member.added)
    Kafka-->>-Logic: acks
    Logic-->>-Gateway: 返回操作结果 (成功/失败用户列表)
    Gateway-->>-Admin: HTTP 201 Created

    par 异步成员变更通知
        Kafka->>+Task: Consume 'gochat.conversation-events'
        Task->>+Repo: RPC: GetConversation(conversation_id) + GetMembers()
        Repo-->>-Task: 返回完整会话信息 + 所有成员列表

        par 向新成员推送
            Task->>Task: 构建 SystemNotificationEvent (conversation.invited)
            Task->>+Gateway: RPC: PushToUsers(new_members, invitation_with_conversation_info)
            Gateway->>NewMembers: WebSocket: conversation-joined (包含会话详情)
        
        and 向现有成员推送
            Task->>Task: 构建 SystemSignalMessage (conversation.member.added)
            Task->>+Gateway: RPC: PushToUsers(existing_members, member_change_signal)
            Gateway->>ExistingMembers: WebSocket: member-added (更新成员列表)
        end
        
        Task-->>-Kafka: Commit offset
    end
```

### 3.2 移除会话成员流程

```mermaid
sequenceDiagram
    participant Admin as 管理员
    participant Gateway as im-gateway
    participant Logic as im-logic
    participant Repo as im-repo
    participant Kafka
    participant Task as im-task
    participant RemovedUser as 被移除用户
    participant OtherMembers as 其他成员

    Admin->>+Gateway: DELETE /conversations/{id}/members/{user_id}
    Gateway->>+Logic: RPC: RemoveMembers(conversation_id, user_ids)
    Logic->>Logic: 权限验证 (管理员权限 + 不能移除群主)
    
    Logic->>+Repo: RPC: RemoveConversationMember(conversation_id, user_id)
    Repo->>Repo: 删除 conversation_members 记录
    Repo->>Repo: 更新 conversations.member_count
    Repo-->>-Logic: 移除成功

    Logic->>+Kafka: Produce 'gochat.conversation-events' (type: conversation.member.removed)
    Kafka-->>-Logic: acks
    Logic-->>-Gateway: 移除成功
    Gateway-->>-Admin: HTTP 200 OK

    par 异步成员移除通知
        Kafka->>+Task: Consume 'gochat.conversation-events'
        Task->>+Repo: RPC: GetUser(removed_user_id) + GetConversationMembers()
        Repo-->>-Task: 返回被移除用户信息 + 剩余成员列表

        par 通知被移除用户
            Task->>Task: 构建 SystemSignalMessage (conversation.removed)
            Task->>+Gateway: RPC: PushToUsers([removed_user], removal_notice)
            Gateway->>RemovedUser: WebSocket: conversation-removed (会话从列表中移除)
        
        and 通知其他成员
            Task->>Task: 构建 SystemSignalMessage (conversation.member.removed)
            Task->>+Gateway: RPC: PushToUsers(other_members, member_change_signal)
            Gateway->>OtherMembers: WebSocket: member-removed (更新成员列表)
        end
        
        Task-->>-Kafka: Commit offset
    end
```

### 3.3 成员角色变更流程

```mermaid
sequenceDiagram
    participant Owner as 群主
    participant Gateway as im-gateway
    participant Logic as im-logic
    participant Repo as im-repo
    participant Kafka
    participant Task as im-task
    participant TargetUser as 目标用户
    participant AllMembers as 所有成员

    Owner->>+Gateway: PUT /conversations/{id}/members/{user_id} (role=2, 设为管理员)
    Gateway->>+Logic: RPC: UpdateMemberRole(conversation_id, user_id, new_role)
    Logic->>Logic: 权限验证 (只有群主可以变更角色)
    Logic->>Logic: 角色变更规则验证 (不能降级自己等)
    
    Logic->>+Repo: RPC: UpdateConversationMemberRole(conversation_id, user_id, new_role)
    Repo->>Repo: 更新 conversation_members.role
    Repo-->>-Logic: 角色更新成功

    Logic->>+Kafka: Produce 'gochat.conversation-events' (type: conversation.member.role.updated)
    Kafka-->>-Logic: acks
    Logic-->>-Gateway: 角色更新成功
    Gateway-->>-Owner: HTTP 200 OK

    par 异步角色变更通知
        Kafka->>+Task: Consume 'gochat.conversation-events'
        Task->>+Repo: RPC: GetConversationMember(conversation_id, user_id)
        Repo-->>-Task: 返回更新后的成员信息

        par 通知目标用户
            Task->>Task: 构建 SystemSignalMessage (conversation.role.updated)
            Task->>+Gateway: RPC: PushToUsers([target_user], role_update_notice)
            Gateway->>TargetUser: WebSocket: role-updated (权限变更通知)
        
        and 通知所有成员 
            Task->>Task: 构建 SystemSignalMessage (conversation.member.role.changed)
            Task->>+Gateway: RPC: PushToUsers(all_members, member_role_change)
            Gateway->>AllMembers: WebSocket: member-role-changed (成员列表角色更新)
        end
        
        Task-->>-Kafka: Commit offset
    end
```

---

## 4. 用户在线状态与好友通知流程

**目标**: 展示基于好友关系的在线状态通知机制。

### 4.1 用户上线通知流程

```mermaid
sequenceDiagram
    participant User A as 用户 A
    participant Gateway as im-gateway
    participant Repo as im-repo
    participant Kafka
    participant Task as im-task
    participant Friends as A的好友们

    User A->>+Gateway: 建立 WebSocket 连接 (携带 JWT token)
    Gateway->>Gateway: 验证 JWT token, 提取 user_id
    Gateway->>+Repo: RPC: SetUserOnline(user_id, gateway_id, client_info)
    Repo->>Repo: 写入/更新 Redis 在线状态记录
    Repo-->>-Gateway: 在线状态设置成功
    
    Gateway->>+Kafka: Produce 'gochat.user-events' (type: user.online)
    Kafka-->>-Gateway: acks
    Gateway-->>-User A: WebSocket 连接建立成功

    par 异步好友上线通知
        Kafka->>+Task: Consume 'gochat.user-events'
        Task->>+Repo: RPC: GetUserFriends(user_id, status=1) // 只获取已确认的好友
        Repo-->>-Task: 返回好友用户列表
        
        Task->>Task: 构建 SystemSignalMessage (type: user.online)
        Task->>+Gateway: RPC: PushToUsers(friends_list, online_signal)
        
        loop 向每个在线好友推送
            Gateway->>Friends: WebSocket: friend-online (好友上线通知)
        end
        
        Task-->>-Kafka: Commit offset
    end

    Note over User A, Friends: 用户 A 上线后，所有好友都会收到实时的上线通知
```

### 4.2 用户下线通知流程

```mermaid
sequenceDiagram
    participant User A as 用户 A
    participant Gateway as im-gateway
    participant Repo as im-repo
    participant Kafka
    participant Task as im-task
    participant Friends as A的好友们

    User A->>+Gateway: 关闭 WebSocket 连接 (主动/被动断开)
    Gateway->>Gateway: 检测连接关闭, 记录下线原因
    Gateway->>+Repo: RPC: SetUserOffline(user_id, gateway_id, reason)
    Repo->>Repo: 更新 Redis 在线状态 (last_seen 时间戳)
    Repo-->>-Gateway: 下线状态更新成功
    
    Gateway->>+Kafka: Produce 'gochat.user-events' (type: user.offline)
    Kafka-->>-Gateway: acks

    par 异步好友下线通知 (延迟处理避免频繁切换)
        Kafka->>+Task: Consume 'gochat.user-events'
        Task->>Task: 等待 30 秒 (避免快速重连的误通知)
        Task->>+Repo: RPC: GetUserOnlineStatus(user_id)
        Repo-->>-Task: 确认用户仍处于离线状态
        
        Task->>+Repo: RPC: GetUserFriends(user_id, status=1)
        Repo-->>-Task: 返回好友用户列表
        
        Task->>Task: 构建 SystemSignalMessage (type: user.offline)
        Task->>+Gateway: RPC: PushToUsers(friends_list, offline_signal)
        
        loop 向每个在线好友推送
            Gateway->>Friends: WebSocket: friend-offline (好友离线通知)
        end
        
        Task-->>-Kafka: Commit offset
    end
```

---

## 5. 世界聊天室特殊处理流程

**目标**: 展示世界聊天室作为特殊群聊的自动加入和高并发消息处理机制。

### 5.1 游客自动加入世界聊天室

```mermaid
sequenceDiagram
    participant Guest as 游客
    participant Gateway as im-gateway
    participant Logic as im-logic
    participant Repo as im-repo
    participant Kafka
    participant Task as im-task

    Guest->>+Gateway: POST /auth/guest (游客登录)
    Gateway->>+Logic: RPC: GuestLogin()
    
    par 并行处理: 创建游客 + 自动加入世界聊天室
        Logic->>+Repo: RPC: CreateUser(is_guest=true, username=auto_generated)
        Repo->>Repo: 写入 users 表 (is_guest=true, username='guest_xxxx')
        Repo-->>-Logic: 返回游客用户信息
        
    and 自动加入世界聊天室
        Logic->>+Repo: RPC: AddConversationMember('world_chat_room', guest_user_id, role=1)
        Repo->>Repo: 写入 conversation_members 表
        Repo->>Repo: 更新世界聊天室成员数
        Repo-->>-Logic: 加入成功
    end

    Logic->>Logic: 生成 JWT tokens
    Logic->>+Kafka: Produce 'gochat.conversation-events' (type: conversation.member.added)
    Kafka-->>-Logic: acks
    Logic-->>-Gateway: 返回游客登录信息 + 世界聊天室会话
    Gateway-->>-Guest: HTTP 201 Created (包含世界聊天室信息)

    par 异步欢迎处理 (可选)
        Kafka->>+Task: Consume conversation events
        Task->>Task: 检查是否为世界聊天室新成员
        Task->>Task: 构建欢迎消息 (系统消息)
        Task->>+Kafka: Produce 'gochat.messages.downstream.world_chat_room'
        Task-->>-Kafka: Commit offset
        
        Gateway->>Guest: WebSocket: 推送欢迎消息到世界聊天室
    end
```

### 5.2 世界聊天室高并发消息处理

```mermaid
sequenceDiagram
    participant User as 用户
    participant Gateway as im-gateway
    participant Logic as im-logic
    participant Kafka
    participant Task1 as im-task-1
    participant Task2 as im-task-2
    participant TaskN as im-task-N
    participant Repo as im-repo
    participant AllUsers as 所有在线用户

    User->>+Gateway: WebSocket: 发送世界聊天室消息
    Gateway->>+Kafka: Produce 'gochat.messages.upstream' (conversation_id='world_chat_room')
    
    Kafka->>+Logic: Consume upstream message
    Logic->>Logic: 识别为世界聊天室消息 (type=3)
    Logic->>Logic: 应用频率限制和内容过滤
    
    par 并行处理: 持久化 + 大规模扇出
        Logic->>+Kafka: Produce 'gochat.messages.persist'
        
    and 分片扇出任务
        Logic->>+Kafka: Produce 'gochat.tasks.fanout' (batch_size=5000, shard_count=10)
    end

    par 消息持久化
        Kafka->>+Task1: Consume persist message
        Task1->>+Repo: RPC: SaveMessage(world_chat_room_message)
        Repo-->>-Task1: 持久化成功
    end

    par 分片并行扇出 (提高吞吐量)
        Kafka->>+Task1: Consume fanout task (shard 1: users 1-5000)
        Task1->>+Repo: RPC: BatchGetOnlineStatus(user_batch_1)
        Repo-->>-Task1: 返回在线状态 + 网关分布
        Task1->>+Kafka: 批量 Produce downstream messages
        
        Kafka->>+Task2: Consume fanout task (shard 2: users 5001-10000) 
        Task2->>+Repo: RPC: BatchGetOnlineStatus(user_batch_2)
        Repo-->>-Task2: 返回在线状态 + 网关分布
        Task2->>+Kafka: 批量 Produce downstream messages
        
        Kafka->>+TaskN: Consume fanout task (shard N: users ...)
        TaskN->>+Repo: RPC: BatchGetOnlineStatus(user_batch_N)
        Repo-->>-TaskN: 返回在线状态 + 网关分布
        TaskN->>+Kafka: 批量 Produce downstream messages
    end

    par 消息推送到各网关
        loop 多个 Gateway 实例并行处理
            Kafka->>Gateway: Consume 'gochat.messages.downstream.{instance_id}'
            Gateway->>AllUsers: WebSocket: 批量推送世界聊天室消息
        end
    end

    Note over User, AllUsers: 通过分片并行处理，世界聊天室可以支持数万并发用户的实时消息收发
```

---

## 6. 错误处理与回退机制

### 6.1 消息发送失败回退

```mermaid
sequenceDiagram
    participant Client as 客户端
    participant Gateway as im-gateway
    participant Logic as im-logic
    participant Kafka
    participant Task as im-task
    participant Repo as im-repo

    Client->>+Gateway: WebSocket: 发送消息
    Gateway->>Gateway: 生成 client_msg_id (幂等性)
    Gateway->>+Kafka: Produce 'gochat.messages.upstream'
    Kafka-->>-Gateway: acks
    Gateway-->>-Client: 临时确认 (status: sending)

    Kafka->>+Logic: Consume upstream message
    Logic->>Logic: 业务逻辑验证
    
    alt 验证失败 (权限不足、内容违规等)
        Logic->>+Kafka: Produce 'gochat.message-events' (type: message.send.failed)
        Logic-->>-Gateway: 返回错误响应
        Gateway->>Client: WebSocket: message-error (发送失败通知)
        
    else 验证成功但持久化失败
        Logic->>+Kafka: Produce 'gochat.messages.persist'
        Kafka->>+Task: Consume persist message
        Task->>+Repo: RPC: SaveMessage()
        Repo-->>-Task: 返回数据库错误
        Task->>+Kafka: Produce 'gochat.message-events' (type: message.persist.failed)
        Task->>+Gateway: RPC: PushToUsers([sender], persist_failed_signal)
        Gateway->>Client: WebSocket: message-persist-failed (重试提示)
        
    else 验证成功且持久化成功
        Logic->>+Kafka: Produce 'gochat.messages.persist' + 'downstream'
        Task->>+Repo: RPC: SaveMessage()
        Repo-->>-Task: 持久化成功
        Task->>+Gateway: RPC: PushToUsers([sender], message_confirmed)
        Gateway->>Client: WebSocket: message-confirmed (发送成功确认)
    end
```

### 6.2 服务降级与熔断机制

```mermaid
sequenceDiagram
    participant Client as 客户端
    participant Gateway as im-gateway
    participant Logic as im-logic
    participant Repo as im-repo
    participant Circuit as 熔断器

    Client->>+Gateway: 高并发请求
    Gateway->>+Logic: RPC 调用
    Logic->>+Circuit: 检查服务状态
    
    alt Repo 服务正常
        Circuit-->>Logic: 状态: CLOSED
        Logic->>+Repo: RPC: 正常调用
        Repo-->>-Logic: 正常响应
        Circuit->>Circuit: 记录成功调用
        
    else Repo 服务异常 (连续失败超阈值)
        Circuit-->>Logic: 状态: OPEN (熔断开启)
        Logic->>Logic: 启用降级策略
        Logic->>Logic: 从本地缓存/默认值返回
        Logic-->>Gateway: 降级响应 (标记为 degraded)
        Gateway->>Client: 降级模式响应 + 提示
        
    else Repo 服务恢复中
        Circuit-->>Logic: 状态: HALF_OPEN
        Logic->>+Repo: RPC: 探测性调用
        alt 调用成功
            Repo-->>-Logic: 成功响应
            Circuit->>Circuit: 重置为 CLOSED
        else 调用仍失败
            Logic-->>Logic: 继续降级模式
            Circuit->>Circuit: 重新开启熔断
        end
    end
```

---

## 7. 性能优化策略

### 7.1 缓存策略优化

#### Redis 缓存层次设计
```markdown
1. **L1 缓存 (热点数据, TTL=5min)**:
   - `user:online:{user_id}` - 用户在线状态
   - `conversation:members:{conversation_id}` - 会话成员列表
   - `message:latest:{conversation_id}` - 会话最新消息

2. **L2 缓存 (温数据, TTL=30min)**:
   - `user:profile:{user_id}` - 用户基本信息
   - `conversation:info:{conversation_id}` - 会话元信息
   - `user:conversations:{user_id}` - 用户会话列表

3. **L3 缓存 (冷数据, TTL=2hour)**:
   - `conversation:messages:{conversation_id}:latest:50` - 会话最新50条消息
   - `user:friends:{user_id}` - 用户好友列表
```

#### 缓存更新策略
```mermaid
sequenceDiagram
    participant App as 应用服务
    participant Redis as Redis缓存
    participant MySQL as MySQL数据库
    participant Kafka

    App->>+Redis: 尝试获取缓存数据
    alt 缓存命中
        Redis-->>-App: 返回缓存数据
    else 缓存未命中
        Redis-->>App: 缓存 MISS
        App->>+MySQL: 查询数据库
        MySQL-->>-App: 返回数据
        App->>+Redis: 写入缓存 (设置TTL)
        Redis-->>-App: 缓存写入成功
        App-->>App: 返回数据
    end

    Note over App, Kafka: 数据更新时的缓存失效策略
    App->>MySQL: 更新数据
    App->>+Kafka: 发布缓存失效事件
    Kafka->>Redis: 删除相关缓存键
    App->>+Redis: 预热新数据 (可选)
```

### 7.2 数据库查询优化

#### 会话列表查询优化
```sql
-- 优化前: 多次 JOIN 查询
SELECT c.*, m.content as last_content, 
       COUNT(msg.id) as unread_count
FROM conversation_members cm
JOIN conversations c ON cm.conversation_id = c.id
LEFT JOIN messages m ON c.last_message_id = m.id
LEFT JOIN messages msg ON (msg.conversation_id = c.id 
    AND msg.seq_id > COALESCE(urp.last_read_seq_id, 0))
LEFT JOIN user_read_pointers urp ON (urp.user_id = ? AND urp.conversation_id = c.id)
WHERE cm.user_id = ?
GROUP BY c.id
ORDER BY c.updated_at DESC;

-- 优化后: 分层查询 + 批量获取
-- Step 1: 快速获取会话ID列表 (使用覆盖索引)
SELECT conversation_id, updated_at 
FROM conversation_members 
WHERE user_id = ? 
ORDER BY updated_at DESC 
LIMIT 20;

-- Step 2: 批量获取会话基本信息
SELECT * FROM conversations WHERE id IN (?, ?, ...);

-- Step 3: 批量获取未读数 (通过 Redis 计数器)
MGET unread_count:user_123:conv_456 unread_count:user_123:conv_789 ...

-- Step 4: 批量获取最新消息 (通过 Redis 缓存)
MGET latest_message:conv_456 latest_message:conv_789 ...
```

#### 消息查询分片策略
```sql
-- 分片路由逻辑 (在应用层实现)
function getShardForConversation(conversationId) {
    return crc32(conversationId) % SHARD_COUNT;
}

-- 各分片的查询 (相同结构)
SELECT id, sender_id, content, seq_id, created_at
FROM messages_shard_${shardId}
WHERE conversation_id = ? 
  AND seq_id < ? 
  AND deleted = false
ORDER BY seq_id DESC 
LIMIT ?;
```

### 7.3 Kafka 消息处理优化

#### 批量处理优化
```go
// im-task 服务中的批量消息处理
type MessageBatch struct {
    ConversationID string
    Messages       []DownstreamMessage
    BatchSize      int
    ProcessedAt    time.Time
}

// 批量持久化消息
func (s *TaskService) processPersistenceBatch(batch []PersistenceMessage) error {
    // 按会话ID分组批量处理
    groupedByConversation := groupMessagesByConversation(batch)
    
    for conversationID, messages := range groupedByConversation {
        // 批量生成 seq_id (减少数据库调用)
        seqIDs := s.generateSequentialIDs(conversationID, len(messages))
        
        // 批量插入消息
        err := s.repo.BatchSaveMessages(conversationID, messages, seqIDs)
        if err != nil {
            return fmt.Errorf("batch save failed: %w", err)
        }
        
        // 异步更新缓存
        go s.updateConversationCache(conversationID, messages[len(messages)-1])
    }
    
    return nil
}
```

#### 消息扇出优化
```go
// 智能扇出策略
func (s *TaskService) processFanoutTask(task FanoutTask) error {
    conversation, err := s.repo.GetConversation(task.ConversationID)
    if err != nil {
        return err
    }
    
    switch {
    case conversation.MemberCount <= 100:
        // 小群: 直接获取所有成员并推送
        return s.processSmallGroupFanout(task)
        
    case conversation.MemberCount <= 1000:
        // 中群: 分批处理，每批200人
        return s.processMediumGroupFanout(task, 200)
        
    case conversation.Type == ConversationTypeWorld:
        // 世界聊天室: 特殊优化，基于地理位置/在线时长分片
        return s.processWorldChatFanout(task)
        
    default:
        // 大群: 最大分片并行处理
        return s.processLargeGroupFanout(task, 500)
    }
}
```

### 7.4 WebSocket 连接优化

#### 连接池管理
```go
// Gateway 中的连接管理优化
type ConnectionManager struct {
    // 按用户ID索引的连接
    userConnections map[string]*Connection
    // 按会话ID索引的连接 (用于群推送)
    conversationConnections map[string]map[string]*Connection
    // 连接统计信息
    stats ConnectionStats
    // 读写锁保护
    mu sync.RWMutex
}

// 智能推送优化
func (cm *ConnectionManager) PushToConversation(conversationID string, message []byte) {
    cm.mu.RLock()
    connections := cm.conversationConnections[conversationID]
    cm.mu.RUnlock()
    
    if len(connections) == 0 {
        return
    }
    
    // 并发推送优化 (批量+协程池)
    semaphore := make(chan struct{}, 100) // 限制并发数
    var wg sync.WaitGroup
    
    for userID, conn := range connections {
        wg.Add(1)
        semaphore <- struct{}{} // 获取信号量
        
        go func(uid string, c *Connection) {
            defer func() {
                <-semaphore // 释放信号量
                wg.Done()
            }()
            
            if err := c.WriteMessage(message); err != nil {
                // 连接失败时自动清理
                cm.removeConnection(uid)
                log.Warn("Failed to push message to user", "user_id", uid, "error", err)
            }
        }(userID, conn)
    }
    
    wg.Wait()
}
```

---

## 8. 监控与可观测性

### 8.1 关键指标监控

#### 业务指标
```markdown
1. **消息处理指标**:
   - 消息发送成功率 (>99.9%)
   - 消息端到端延迟 (P95 < 100ms)
   - 消息持久化延迟 (P95 < 50ms)

2. **会话管理指标**:
   - 会话创建成功率 (>99.5%)
   - 会话列表加载时间 (P95 < 200ms)
   - 成员变更操作延迟 (P95 < 500ms)

3. **用户体验指标**:
   - 用户上线通知延迟 (P95 < 2s)
   - 好友申请处理时间 (P95 < 1s)
   - WebSocket 连接成功率 (>99.8%)
```

#### 系统指标
```markdown
1. **服务健康度**:
   - 各微服务可用性 (>99.9%)
   - gRPC 调用成功率 (>99.5%)
   - Kafka 消费延迟 (<1s)

2. **资源使用**:
   - 内存使用率 (<80%)
   - CPU 使用率 (<70%)
   - 数据库连接池使用率 (<85%)

3. **缓存效率**:
   - Redis 缓存命中率 (>95%)
   - 缓存更新延迟 (<10ms)
   - 缓存内存使用率 (<90%)
```

### 8.2 告警策略

```yaml
# Prometheus 告警规则示例
groups:
  - name: gochat.rules
    rules:
      - alert: MessageSendFailureHigh
        expr: rate(gochat_message_send_failures_total[5m]) > 0.01
        for: 2m
        labels:
          severity: warning
        annotations:
          summary: "消息发送失败率过高"
          
      - alert: ConversationLoadSlow
        expr: histogram_quantile(0.95, rate(gochat_conversation_load_duration_seconds_bucket[5m])) > 0.5
        for: 1m
        labels:
          severity: critical
        annotations:
          summary: "会话加载延迟过高"
          
      - alert: KafkaConsumerLag
        expr: kafka_consumer_lag_sum > 1000
        for: 30s
        labels:
          severity: critical
        annotations:
          summary: "Kafka 消费积压严重"
```

通过这套完整的业务流程设计，GoChat 系统实现了：

1. **统一的会话抽象**: 单聊、群聊、世界聊天室使用相同的处理逻辑
2. **事件驱动架构**: 通过 Kafka 实现服务解耦和异步处理
3. **高性能优化**: 多层缓存、数据库分片、批量处理等
4. **容错机制**: 熔断、降级、重试等保证系统稳定性
5. **可观测性**: 全面的监控和告警体系

整个系统设计体现了"简单而不简陋"的设计哲学，通过统一抽象降低复杂度，通过事件驱动提升扩展性，通过性能优化保证用户体验。