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
    participant Repo as im-repo
    participant GatewayB as Gateway (B所在)
    participant ClientB as 客户端 B

    ClientA->>+GatewayA: 1. WebSocket 发送消息 (send-message)
    GatewayA->>+Kafka: 2. 生产上行消息 (im-upstream-topic)
    Kafka-->>-GatewayA: (Ack)
    GatewayA-->>-ClientA: (可选) 发送成功

    Kafka->>+Logic: 3. 消费上行消息
    Logic->>+Repo: 4. 检查幂等性, 生成 message_id, seq_id
    Repo-->>-Logic: 5. 返回ID
    Logic->>+Repo: 6. 持久化消息 (MySQL + Redis)
    Repo-->>-Logic: 7. 持久化成功
    Logic->>+Repo: 8. 查询B的在线网关 (GetUserSession)
    Repo-->>-Logic: 9. 返回 GatewayB 的 ID
    Logic->>+Kafka: 10. 生产下行消息 (im-downstream-topic-gatewayB)
    Kafka-->>-Logic: (Ack)
    Logic-->>-Kafka: (关闭)

    Kafka->>+GatewayB: 11. 消费下行消息
    GatewayB->>+ClientB: 12. WebSocket 推送新消息 (new-message)
    ClientB-->>-GatewayB: (Ack)
    GatewayB-->>-Kafka: (关闭)
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
    participant Redis as Redis

    Client->>+Gateway: 1. WebSocket 发送群聊消息
    Gateway->>+Kafka: 2. 生产上行消息 (im-upstream-topic)

    Kafka->>+Logic: 3. 消费上行消息
    Logic->>+Repo: 4. 持久化消息 (MySQL + Redis)
    Repo-->>-Logic: 5. 持久化成功

    Logic->>+Repo: 6. 获取群成员列表 (GetGroupMembers)
    Repo->>+Redis: 7. (缓存命中) 读 group_members:{group_id}
    Redis-->>-Repo: 8. 返回成员列表
    Repo-->>-Logic: 9. 返回成员列表

    Logic->>+Repo: 10. 批量查询在线成员的网关 (GetUsersSession)
    Repo-->>-Logic: 11. 返回在线成员与网关映射

    Logic->>+Kafka: 12. **循环/批量**生产下行消息到多个网关Topic
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
    Gateway->>+Kafka: 2. 生产上行消息 (im-upstream-topic)

    Kafka->>+Logic: 3. 消费上行消息
    Logic->>+Repo: 4. 持久化消息
    Repo-->>-Logic: 5. 持久化成功

    Logic->>Logic: 6. 判断为超大群
    Logic->>+Kafka: 7. 生产异步任务 (im-task-large-group-fanout-topic)

    Kafka->>+Task: 8. 消费异步任务
    Task->>+Repo: 9. 分批获取群成员列表
    Repo-->>-Task: 10. 返回一批成员
    Task->>+Repo: 11. 查询该批成员的在线网关
    Repo-->>-Task: 12. 返回在线信息
    Task->>+Kafka: 13. 循环/批量生产下行消息到多个网关Topic
    Note right of Task: 重复9-13步直到所有成员处理完毕
```

### 5. 异步消息索引流程 (搜索与AI)

此图描述了消息如何被异步地索引到 Elasticsearch 和向量数据库中。

```mermaid
sequenceDiagram
    participant Logic as im-logic
    participant Kafka as Kafka
    participant Task as im-task (Indexing Worker)
    participant Elasticsearch as Elasticsearch
    participant VectorDB as Vector DB

    Logic->>+Kafka: 1. 生产消息到<br>indexing-topic
    
    Kafka->>+Task: 2. 消费消息
    Task->>Task: 3. (可选) 缓冲与分块
    Task->>+Elasticsearch: 4. 写入文档
    Task->>+VectorDB: 5. 写入向量
    
    Elasticsearch-->>-Task: (Ack)
    VectorDB-->>-Task: (Ack)
    Task-->>-Kafka: (关闭)
```

### 6. AI 对话交互流程

此图描述了用户与 AI Agent 对话时的后端处理流程。

```mermaid
sequenceDiagram
    participant Client as 客户端
    participant Gateway as im-gateway
    participant Logic as im-logic
    participant AI as im-ai
    participant VectorDB as Vector DB
    participant LLM as LLM API

    Client->>+Gateway: 1. 发送 AI 消息
    Gateway->>+Logic: 2. gRPC 调用 (SendMessageToAI)
    
    Logic->>+AI: 3. gRPC 调用 (Chat)
    AI->>AI: 4. 查询向量化
    AI->>+VectorDB: 5. 检索相关上下文
    VectorDB-->>-AI: 6. 返回上下文 Chunks
    AI->>AI: 7. 构建 Prompt
    AI->>+LLM: 8. 调用大模型 API
    LLM-->>-AI: 9. 返回结果
    AI-->>-Logic: 10. 返回最终答复
    
    Logic->>+Gateway: 11. 推送消息到客户端
    Gateway->>+Client: 12. WebSocket 推送 (new-message)
```

### 7. 文件上传流程

此图描述了客户端上传图片或文件并发送消息的流程。

```mermaid
sequenceDiagram
    participant Client as 客户端
    participant Logic as im-logic
    participant MinIO as MinIO (对象存储)
    
    Client->>+Logic: 1. 请求上传凭证 (HTTP)
    Logic->>+MinIO: 2. 生成预签名 URL
    MinIO-->>-Logic: 3. 返回 URL
    Logic-->>-Client: 4. 返回预签名 URL
    
    Client->>+MinIO: 5. **直接上传文件** (HTTP PUT)
    MinIO-->>-Client: 6. 上传成功
    
    Client->>Client: 7. 构造包含文件 URL 的消息体
    Client->>Logic: 8. 发送消息 (WebSocket)
```
