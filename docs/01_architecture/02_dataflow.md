# 核心场景数据流

本文档使用 Mermaid 序列图详细描述了 GoChat 系统中几个核心业务场景的数据流转过程。

### 1. 用户注册与登录认证流程

此图描述了用户从客户端发起注册或登录，到最终完成认证并建立 WebSocket 连接的全过程。

```mermaid
sequenceDiagram
    participant Client as 客户端
    participant Gateway as im-gateway
    participant Logic as im-logic
    participant Repo as im-repo
    participant Redis as Redis
    participant MySQL as MySQL

    Client->>+Gateway: 1. 发起 HTTP 注册/登录请求
    Gateway->>+Logic: 2. 发起 gRPC 调用 (Register/Login)
    Logic->>+Repo: 3. 发起 gRPC 调用 (CreateUser/GetUser)
    Repo->>+MySQL: 4. 读/写 users 表
    MySQL-->>-Repo: 5. 返回结果
    Repo-->>-Logic: 6. 返回结果
    Logic-->>-Gateway: 7. 返回JWT Token
    Gateway-->>-Client: 8. 返回JWT Token

    Client->>+Gateway: 9. 使用Token发起 WebSocket 连接
    Gateway->>Gateway: 10. 验证JWT Token
    Gateway->>+Redis: 11. 写入在线状态 (user_session)
    Redis-->>-Gateway: 12. 确认写入
    Gateway-->>Client: 13. WebSocket 连接建立成功
```

### 2. 单聊消息发送与接收流程

此图描述了用户A发送一条单聊消息给用户B的完整生命周期。

```mermaid
sequenceDiagram
    participant ClientA as 客户端 A
    participant GatewayA as Gateway (A所在)
    participant Kafka as Kafka
    participant Logic as im-logic
    participant GatewayB as Gateway (B所在)
    participant ClientB as 客户端 B
    participant Task as im-task
    participant Repo as im-repo

    ClientA->>+GatewayA: 1. WebSocket 发送消息
    GatewayA->>+Kafka: 2. 生产上行消息 (gochat.messages.upstream)

    Kafka->>+Logic: 3. 消费上行消息
    Logic->>Logic: 4. 业务处理, 生成 message_id, 查询B所在网关
    Logic->>+Kafka: 5. 生产下行消息 (gochat.messages.downstream.{gatewayB_id})
    
    note right of Kafka: 并发消费阶段
    Kafka->>+GatewayB: 6. 消费下行消息
    GatewayB->>+ClientB: 7. WebSocket 推送新消息
    
    Kafka->>+Task: 8. 消费下行消息 (订阅 gochat.messages.downstream.*)
    Task->>+Repo: 9. gRPC 调用 (SaveMessage)
    Repo-->>-Task: 10. 持久化成功
```

### 3. 群聊消息发送流程 (中小群)

此图描述了在一个成员数小于阈值（如500人）的群聊中，消息被实时扩散的流程。

```mermaid
sequenceDiagram
    participant Client as 客户端
    participant Gateway as im-gateway
    participant Kafka as Kafka
    participant Logic as im-logic
    participant Repo as im-repo

    Client->>+Gateway: 1. WebSocket 发送群聊消息
    Gateway->>+Kafka: 2. 生产上行消息 (gochat.messages.upstream)

    Kafka->>+Logic: 3. 消费上行消息
    Logic->>+Repo: 4. 获取群成员列表及在线状态
    Repo-->>-Logic: 5. 返回在线成员与网关映射
    
    Logic->>+Kafka: 6. **循环/批量**生产下行消息到多个网关Topic (gochat.messages.downstream.{gateway_id})
    
    note right of Logic: Task服务会通过通配符订阅消费这些消息并持久化
```

### 4. 群聊消息发送流程 (超大群)

此图描述了在一个成员数超过阈值的群聊中，消息扩散任务被异步处理的流程。

```mermaid
sequenceDiagram
    participant Client as 客户端
    participant Gateway as im-gateway
    participant Kafka as Kafka
    participant Logic as im-logic
    participant Task as im-task
    participant Repo as im-repo

    Client->>+Gateway: 1. WebSocket 发送群聊消息
    Gateway->>+Kafka: 2. 生产上行消息 (gochat.messages.upstream)

    Kafka->>+Logic: 3. 消费上行消息
    Logic->>Logic: 4. 业务处理, 生成 message_id
    par 并行流程
        Logic->>+Kafka: 5a. 生产持久化消息 (gochat.messages.persist)
        Kafka->>+Task: 6a. 消费并调用 repo 持久化 (仅一次)

    and
        Logic->>+Kafka: 5b. 生产异步扇出任务 (gochat.tasks.fanout)
        Kafka->>+Task: 6b. 消费扇出任务
        Task->>+Repo: 7b. 分批获取群成员
        Task->>+Kafka: 8b. **循环/批量**生产下行消息到多个网关Topic
    end
```
