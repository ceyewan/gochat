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
// 注意：此 Config 通常由 coord 自动获取和填充，业务代码很少直接创建。
type Config struct {
    Brokers []string `json:"brokers"`
    // ... 其他 Kafka 相关配置，如 SASL, TLS 等
}

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

// ConsumeCallback 是消费者处理消息的回调函数。
// 如果函数返回 error，将触发错误日志记录，但消费流程会继续。
type ConsumeCallback func(ctx context.Context, msg *Message) error

// Consumer 是一个消费者组的接口。
type Consumer interface {
	// Subscribe 订阅一个或多个 Topic，并使用提供的回调函数处理消息。
	// 此方法是阻塞的，它会启动一个循环来拉取和处理消息。
	// 偏移量由组件在后台自动提交。
	Subscribe(ctx context.Context, topics []string, callback ConsumeCallback) error

	// Close 优雅地关闭消费者，完成当前正在处理的消息并提交最后一次偏移量。
	Close() error
}
```

## 3. 标准用法

### 场景 1: 生产者发送消息

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

    eventData, _ := json.Marshal(user)
    msg := &mq.Message{
        Topic: "user.events.registered",
        Key:   []byte(user.ID),
        Value: eventData,
    }

    // 使用异步发送，并记录可能的错误
    // trace_id 会自动从 ctx 中提取并注入到消息头
    producer.Send(ctx, msg, func(err error) {
        if err != nil {
            clog.C(ctx).Error("发送用户注册事件失败", clog.Err(err))
        }
    })
    
    return nil
}
```

### 场景 2: 消费者处理消息

```go
// 1. 在服务启动时初始化 Consumer
var mqConfig mq.Config
// ... 从 coord 加载配置 ...
consumer, err := mq.NewConsumer(context.Background(), &mqConfig, "notification-service-group")
if err != nil {
    log.Fatal(err)
}
defer consumer.Close()

// 2. 定义消息处理逻辑
handler := func(ctx context.Context, msg *mq.Message) error {
    // trace_id 已被自动从消息头提取并注入到 ctx 中
    logger := clog.C(ctx)
    logger.Info("收到新用户注册事件", clog.String("key", string(msg.Key)))
    
    var user User
    if err := json.Unmarshal(msg.Value, &user); err != nil {
        logger.Error("反序列化用户事件失败", clog.Err(err))
        return err // 返回错误，但不会中断消费
    }

    // ... 发送欢迎邮件等业务逻辑 ...
    return nil
}

// 3. 启动订阅（此方法会阻塞）
go func() {
    topics := []string{"user.events.registered"}
    if err := consumer.Subscribe(context.Background(), topics, handler); err != nil {
        // Subscribe 只有在不可恢复的错误下才会返回 error
        clog.Fatal("消费者订阅失败", clog.Err(err))
    }
}()