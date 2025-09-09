# 基础设施: MQ 消息队列

## 1. 设计理念

`mq` 组件是 `gochat` 项目中用于与 `Kafka` 交互的唯一标准库。其设计核心是 **“极简与约定”**。

- **极简 (Simplicity)**: 提供一个极小化的 API 集合，只包含生产者和消费者的核心功能。目标是让业务开发者在不阅读 Kafka 文档的情况下，也能轻松、可靠地收发消息。
- **约定 (Convention)**: 组件将遵循 `im-infra` 的核心规范。
    - **配置驱动**: 所有 Kafka 的连接信息和行为配置均从 `coord` 获取，业务代码不直接接触配置。
    - **自动偏移量管理**: `Consumer` 自动处理 offset 提交，业务逻辑只需关注消息处理本身，极大降低了消费者的实现复杂性。
    - **内置可观测性**: 自动与 `clog` 集成，并实现了 `trace_id` 在消息 `Headers` 中的自动传递，保证了跨服务的调用链完整性。

## 2. 核心 API 契约

### 2.1 构造函数

```go
// Config 是 mq 组件的配置结构体。
type Config struct {
    // Brokers 是 Kafka 集群的地址列表
    Brokers []string `json:"brokers"`
    // SecurityProtocol 安全协议，如 "PLAINTEXT", "SASL_PLAINTEXT", "SASL_SSL"
    SecurityProtocol string `json:"securityProtocol"`
    // SASLMechanism SASL 认证机制，如 "PLAIN", "SCRAM-SHA-256", "SCRAM-SHA-512"
    SASLMechanism string `json:"saslMechanism,omitempty"`
    // SASLUsername SASL 用户名
    SASLUsername string `json:"saslUsername,omitempty"`
    // SASLPassword SASL 密码
    SASLPassword string `json:"saslPassword,omitempty"`
    // ProducerConfig 生产者专用配置
    ProducerConfig *ProducerConfig `json:"producerConfig,omitempty"`
    // ConsumerConfig 消费者专用配置
    ConsumerConfig *ConsumerConfig `json:"consumerConfig,omitempty"`
}

// ProducerConfig 定义生产者的专用配置
type ProducerConfig struct {
    // Acks 确认级别: 0, 1, -1(all)
    Acks int `json:"acks"`
    // RetryMax 最大重试次数
    RetryMax int `json:"retryMax"`
    // BatchSize 批处理大小
    BatchSize int `json:"batchSize"`
    // LingerMs 延迟发送时间(毫秒)
    LingerMs int `json:"lingerMs"`
}

// ConsumerConfig 定义消费者的专用配置
type ConsumerConfig struct {
    // AutoOffsetReset 偏移量重置策略: "earliest", "latest"
    AutoOffsetReset string `json:"autoOffsetReset"`
    // EnableAutoCommit 是否启用自动提交偏移量
    EnableAutoCommit bool `json:"enableAutoCommit"`
    // AutoCommitIntervalMs 自动提交间隔(毫秒)
    AutoCommitIntervalMs int `json:"autoCommitIntervalMs"`
    // SessionTimeoutMs 会话超时时间(毫秒)
    SessionTimeoutMs int `json:"sessionTimeoutMs"`
}

// GetDefaultConfig 返回默认的 mq 配置。
// 开发环境：较少的重试次数，较小的批处理大小
// 生产环境：较多的重试次数，较大的批处理大小，更强的持久性保证
func GetDefaultConfig(env string) *Config

// Option 定义了用于定制 mq Producer/Consumer 的函数。
type Option func(*options)

// WithLogger 将一个 clog.Logger 实例注入 mq，用于记录内部日志。
func WithLogger(logger clog.Logger) Option

// WithCoordProvider 注入 coord.Provider，用于从配置中心获取动态配置。
func WithCoordProvider(provider coord.Provider) Option

// NewProducer 创建一个新的消息生产者实例。
func NewProducer(ctx context.Context, config *Config, opts ...Option) (Producer, error)

// NewConsumer 创建一个新的消息消费者实例。
// groupID 是 Kafka 的消费者组ID，用于实现负载均衡和故障转移。
func NewConsumer(ctx context.Context, config *Config, groupID string, opts ...Option) (Consumer, error)
```

### 2.2 核心接口与数据结构

```go
// Message 是跨服务的标准消息结构。
type Message struct {
	Topic   string
	Key     []byte
	Value   []byte
	Headers map[string][]byte
}

// Producer 是一个线程安全的消息生产者接口。
type Producer interface {
	// Send 异步发送消息。此方法立即返回，并通过回调函数处理发送结果。
	// 这是推荐的、性能最高的方式。
	Send(ctx context.Context, msg *Message, callback func(error))

	// SendSync 同步发送消息。此方法将阻塞直到消息发送成功或失败。
	// 适用于需要强一致性保证的场景。
	SendSync(ctx context.Context, msg *Message) error

	// Close 关闭生产者，并确保所有缓冲区的消息都已发送。
	Close() error
}

// ConsumeCallback 是标准的消息处理回调函数。
// 返回 nil: 消息处理成功，偏移量将被自动提交。
// 返回 error: 消息处理失败，偏移量不会被提交，消息将在后续被重新消费。
type ConsumeCallback func(ctx context.Context, msg *Message) error

// Consumer 是一个消费者组的接口。
type Consumer interface {
	// Subscribe 订阅消息并根据处理结果决定是否提交偏移量。
	// 只有当回调函数返回 nil (无错误) 时，偏移量才会被自动提交。
	// 如果返回 error，偏移量将不会被提交，消息会在下一次拉取时被重新消费。
	// 这是标准的、推荐的消费方式。
	Subscribe(ctx context.Context, topics []string, callback ConsumeCallback) error

	// Close 优雅地关闭消费者，完成当前正在处理的消息并提交最后一次偏移量。
	Close() error
}
```

## 3. 标准用法

### 场景 1: 基本初始化

```go
// 在服务的 main 函数中初始化 Producer 和 Consumer
func main() {
    // ... 首先初始化 clog 和 coord ...
    clog.Init(...)
    coordProvider, _ := coord.New(...)

    // 1. 使用默认配置（推荐），或从配置中心加载
    config := mq.GetDefaultConfig("development") // "development" or "production"
    
    // 2. 根据环境覆盖必要的配置
    config.Brokers = []string{"localhost:9092"} // 开发环境单节点
    // config.Brokers = []string{"kafka1:9092", "kafka2:9092", "kafka3:9092"} // 生产环境集群
    
    // 3. 创建 Producer 实例
    producer, err := mq.NewProducer(
        context.Background(),
        config,
        mq.WithLogger(clog.Module("mq-producer")),
        mq.WithCoordProvider(coordProvider),
    )
    if err != nil {
        log.Fatalf("初始化 mq producer 失败: %v", err)
    }
    defer producer.Close()
    
    // 4. 创建 Consumer 实例
    consumer, err := mq.NewConsumer(
        context.Background(),
        config,
        "notification-service-user-events-group", // 遵循命名规范的 GroupID
        mq.WithLogger(clog.Module("mq-consumer")),
        mq.WithCoordProvider(coordProvider),
    )
    if err != nil {
        log.Fatalf("初始化 mq consumer 失败: %v", err)
    }
    defer consumer.Close()
    
    // 后续可以将 producer 和 consumer 注入到业务服务中
    // ...
}
```

### 场景 2: 生产者发送消息

```go
// 1. 在服务启动时初始化 Producer
var mqConfig mq.Config
// ... 从 coord 加载配置 ...
producer, err := mq.NewProducer(context.Background(), &mqConfig)
if err != nil {
    log.Fatal(err)
}
defer producer.Close()

// 2. 在业务逻辑中发送消息
func (s *UserService) Register(ctx context.Context, user *User) error {
    // ... 创建用户的业务逻辑 ...

    eventData, err := json.Marshal(user)
    if err != nil {
        return fmt.Errorf("序列化用户事件失败: %w", err)
    }
    
    msg := &mq.Message{
        Topic: "user.events.registered",
        Key:   []byte(user.ID),
        Value: eventData,
    }

    // 使用异步发送（推荐），并记录可能的错误
    // trace_id 会自动从 ctx 中提取并注入到消息头
    s.producer.Send(ctx, msg, func(err error) {
        if err != nil {
            clog.WithContext(ctx).Error("发送用户注册事件失败", clog.Err(err))
        } else {
            clog.WithContext(ctx).Info("用户注册事件发送成功", clog.String("user_id", user.ID))
        }
    })
    
    return nil
}

// 对于需要强一致性的场景，使用同步发送
func (s *OrderService) CreateOrder(ctx context.Context, order *Order) error {
    // ... 创建订单的业务逻辑 ...

    eventData, err := json.Marshal(order)
    if err != nil {
        return fmt.Errorf("序列化订单事件失败: %w", err)
    }
    
    msg := &mq.Message{
        Topic: "order.events.created",
        Key:   []byte(order.ID),
        Value: eventData,
    }

    // 使用同步发送，确保消息发送成功后再返回
    if err := s.producer.SendSync(ctx, msg); err != nil {
        return fmt.Errorf("发送订单创建事件失败: %w", err)
    }
    
    clog.WithContext(ctx).Info("订单创建事件发送成功", clog.String("order_id", order.ID))
    return nil
}
```

### 场景 3: 消费者处理消息

```go
// 在服务启动时设置消费者
func (s *NotificationService) StartConsuming(ctx context.Context) error {
    // 定义消息处理逻辑
    handler := func(ctx context.Context, msg *mq.Message) error {
        // trace_id 已被自动从消息头提取并注入到 ctx 中
        logger := clog.WithContext(ctx)
        logger.Info("收到新用户注册事件", clog.String("key", string(msg.Key)))
        
        var user User
        if err := json.Unmarshal(msg.Value, &user); err != nil {
            logger.Error("反序列化用户事件失败", clog.Err(err))
            // 返回错误，偏移量不会被提交，消息后续会重试
            return err
        }

        // 发送欢迎邮件
        if err := s.sendWelcomeEmail(ctx, &user); err != nil {
            logger.Error("发送欢迎邮件失败", clog.Err(err), clog.String("user_id", user.ID))
            // 返回错误，偏移量不会被提交，消息后续会重试
            return err
        }
        
        logger.Info("成功处理用户注册事件", clog.String("user_id", user.ID))
        // 返回 nil，偏移量将被自动提交
        return nil
    }

    // 启动订阅（此方法会阻塞）
    topics := []string{"gochat.user-events", "gochat.message-events"}
    go func() {
        if err := s.consumer.Subscribe(ctx, topics, handler); err != nil {
            // Subscribe 只有在不可恢复的错误下才会返回 error
            clog.Fatal("消费者订阅失败", clog.Err(err))
        }
    }()
    
    return nil
}
```

## 4. 设计注记

### 4.1 GetDefaultConfig 默认值说明

`GetDefaultConfig` 根据环境返回优化的默认配置：

**开发环境 (development)**:
```go
&Config{
    Brokers:          []string{"localhost:9092"},
    SecurityProtocol: "PLAINTEXT",
    ProducerConfig: &ProducerConfig{
        Acks:      1,        // 适中的确认级别
        RetryMax:  3,        // 较少重试次数
        BatchSize: 16384,    // 较小批处理大小
        LingerMs:  5,        // 较短延迟时间
    },
    ConsumerConfig: &ConsumerConfig{
        AutoOffsetReset:      "latest",
        EnableAutoCommit:     true,
        AutoCommitIntervalMs: 1000,
        SessionTimeoutMs:     10000,
    },
}
```

**生产环境 (production)**:
```go
&Config{
    Brokers:          []string{"kafka1:9092", "kafka2:9092", "kafka3:9092"},
    SecurityProtocol: "SASL_SSL",
    ProducerConfig: &ProducerConfig{
        Acks:      -1,       // 最强确认级别(all)
        RetryMax:  10,       // 更多重试次数
        BatchSize: 65536,    // 较大批处理大小
        LingerMs:  10,       // 适中延迟时间
    },
    ConsumerConfig: &ConsumerConfig{
        AutoOffsetReset:      "earliest",
        EnableAutoCommit:     true,
        AutoCommitIntervalMs: 5000,
        SessionTimeoutMs:     30000,
    },
}
```

用户仍需要根据实际部署环境覆盖 `Brokers` 配置，并根据安全需求设置认证信息。

### 4.2 Trace ID 自动传播机制

`mq` 组件实现了分布式追踪的无缝集成：

**发送端**: 
- `Send/SendSync` 方法自动从 `context.Context` 中提取 `trace_id`
- 将 `trace_id` 作为消息头(`X-Trace-ID`)自动添加到 Kafka 消息中

**接收端**:
- `Subscribe` 回调函数接收的 `context.Context` 已自动包含从消息头提取的 `trace_id`
- 业务代码使用 `clog.WithContext(ctx)` 即可获得带追踪的日志器

这种设计确保了跨服务的调用链完整性，无需业务代码手动处理。

### 4.3 错误处理和重试策略

**消费者错误处理**:
- 回调函数返回 `error` 不会中断消费流程
- 偏移量仍会被自动提交，避免重复消费同一消息
- 建议在业务层面实现重试逻辑和死信队列

**生产者错误处理**:
- 异步发送通过回调函数处理错误
- 同步发送直接返回错误
- 内置重试机制由 `ProducerConfig.RetryMax` 控制

### 4.4 Topic 和 GroupID 的设计原则

**Topic 命名规范**:
- GoChat 项目采用 `gochat.{module}.{type}` 格式
- **核心消息流**: 
  - `gochat.messages.upstream` (客户端 → im-logic)
  - `gochat.messages.downstream.{instanceID}` (im-logic → im-gateway)
  - `gochat.tasks.fanout` (超大群扇出任务)
- **领域事件**:
  - `gochat.user-events` (用户上线、下线、资料更新)
  - `gochat.message-events` (消息已读、撤回)
  - `gochat.notifications` (系统通知)
- **业务代码直接使用预定义常量**，参考 `api/kafka/message.go`

**GroupID 命名规范**:
- 采用 `{service}.{purpose}.group` 格式
- **核心服务组**:
  - `logic.upstream.group` (im-logic 消费上行消息)
  - `gateway.downstream.group.{instanceID}` (im-gateway 消费下行消息)
  - `task.fanout.group` (im-task 处理扇出任务)
- **领域事件消费组**:
  - `analytics.events.group` (数据分析服务)
  - `notification.events.group` (通知服务)

**管理操作边界**:
- `mq` 组件**不提供** Topic 创建、删除等管理功能
- 提供**独立的管理脚本**处理基础设施初始化
- Topic 管理由运维脚本统一处理，确保环境一致性

### 4.5 消息路由机制详解

**Topic 路由**:
```go
msg := &mq.Message{
    Topic: "user.events.registered", // 直接指定目标 Topic
    Key:   []byte(user.ID),          // 用于分区路由，相同 key 到同一分区
    Value: eventData,                // 消息内容
    Headers: map[string][]byte{      // 元数据，不影响路由
        "X-Trace-ID": []byte(traceID),
        "X-Source":   []byte("user-service"),
    },
}
```

**分区路由逻辑**:
- 有 Key：`hash(Key) % partition_count` 
- 无 Key：轮询或随机分配到各分区
- **相同 Key 保证顺序性**：同一用户的事件按顺序处理

**消费者组负载均衡**:
```go
// 同一组的多个实例，分摊处理 Topic 的各个分区
consumer1, _ := mq.NewConsumer(ctx, config, "notification-service-group") // 实例1
consumer2, _ := mq.NewConsumer(ctx, config, "notification-service-group") // 实例2

// 不同组，都会收到所有消息（广播效果）
notificationConsumer, _ := mq.NewConsumer(ctx, config, "notification-service-group")
analyticsConsumer, _   := mq.NewConsumer(ctx, config, "analytics-service-group")
```

### 4.6 与其他组件的集成

**与 clog 集成**:
- 通过 `WithLogger` 选项注入日志器
- 自动记录连接、发送、消费等关键事件
- 支持 trace_id 在消息传递过程中的自动传播

**与 coord 集成**:
- 通过 `WithCoordProvider` 选项支持动态配置
- 可从配置中心获取 Kafka 集群信息和认证配置
- 支持运行时配置热更新（如调整生产者批处理大小等）

**与 retry 集成**:
- 在消费者回调中使用 retry 组件处理临时故障
- 避免因网络抖动等临时问题导致的消息处理失败

## 5. 常见问题 (FAQ)

### Q1: Topic 需要预先创建吗？
**A**: 
- **开发环境**: 可配置 Kafka 自动创建 Topic，方便开发测试
- **生产环境**: 强烈建议预先创建 Topic，并设置合适的分区数和副本数
- **GoChat 管理**: 使用 `deployment/scripts/kafka-admin.sh create-all` 创建所有必要的 Topics
- **mq 组件职责**: 只负责消息的发送和接收，不提供 Topic 管理功能

### Q2: GroupID 可以随意命名吗？
**A**: 
- **不建议随意命名**，应遵循 `{service}.{purpose}.group` 规范
- **GoChat 规范**: 
  - `logic.upstream.group` (im-logic 消费上行消息)
  - `gateway.downstream.group.{instanceID}` (im-gateway 消费下行消息)
  - `task.fanout.group` (im-task 处理扇出任务)
- **相同 GroupID**: 实现负载均衡，多个实例分摊处理消息
- **不同 GroupID**: 实现广播消费，每个组都收到所有消息

### Q3: 如何管理 Kafka Topics 和 Consumer Groups？
**A**:
- **管理脚本**: 使用 `deployment/scripts/kafka-admin.sh`
- **常用命令**:
  ```bash
  # 创建所有 Topics
  ./kafka-admin.sh create-all
  
  # 列出所有 Topics
  ./kafka-admin.sh list
  
  # 查看 Consumer Groups
  ./kafka-admin.sh list-groups
  
  # 监控 Topics 状态
  ./kafka-admin.sh monitor
  ```
- **初始化**: 使用 `deployment/scripts/init-kafka.sh` 一键初始化

### Q3: 如何保证消息的顺序性？
**A**:
- **设置 Message.Key**: 相同 Key 的消息会路由到同一分区
- **单分区消费**: 同一分区内的消息保证 FIFO 顺序
- **示例**: 用户相关事件使用 `user.ID` 作为 Key

### Q4: 消息发送失败怎么办？
**A**:
- **异步发送**: 通过回调函数处理错误，记录日志或发送到死信队列
- **同步发送**: 直接返回错误，业务代码决定重试策略
- **内置重试**: 由 `ProducerConfig.RetryMax` 控制自动重试

### Q5: 消费者处理失败的消息会重复消费吗？
**A**:
- **不会重复消费**: 偏移量会自动提交，即使处理失败
- **错误处理**: 建议在业务层实现重试逻辑
- **死信队列**: 对于永久失败的消息，发送到专门的错误处理 Topic