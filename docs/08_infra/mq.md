# 基础设施: MQ 消息队列

## 1. 设计哲学

`mq` 组件是 `gochat` 项目中用于与 Kafka 交互的唯一标准库。其设计核心是 **“极简与约定”**。

- **极简 (Simplicity)**: 提供一个极小化的 API 集合，只包含生产者和消费者的核心功能。目标是让业务开发者在不阅读 Kafka 文档的情况下，也能轻松收发消息。
- **约定 (Convention)**: 组件将遵循 `gochat` 项目的最佳实践。
    - **配置中心驱动**: 所有配置均从 `coord` 获取，业务代码不直接接触配置结构体。
    - **自动偏移量管理**: 消费者自动处理 offset 提交，业务逻辑只需关注消息处理本身。
    - **内置可观测性**: 自动与 `clog` 集成，并预留 `trace_id` 的传递机制。

本文档是开发者使用 `mq` 组件的“契约”和唯一参考。

## 2. 核心接口

```go
package mq

import (
	"context"
)

// Message 是跨服务的标准消息结构。
// Key 用于分区，确保同一 Key 的消息进入同一分区。
// Headers 用于传递元数据，如 trace_id。
type Message struct {
	Topic   string
	Key     []byte
	Value   []byte
	Headers map[string][]byte
}

// Producer 是一个线程安全的消息生产者接口。
type Producer interface {
	// Send 异步发送消息。
	// 此方法立即返回，并通过回调函数处理发送结果。
	// 这是推荐的、性能最高的方式。
	Send(ctx context.Context, msg *Message, callback func(error))

	// SendSync 同步发送消息。
	// 此方法将阻塞直到消息发送成功或失败。
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

## 3. 使用指南

### 3.1 初始化

组件的实例化将完全依赖 `im-infra` 的标准模式：通过 `Option` 注入依赖，通过 `coord` 获取配置。

```go
package mq

import (
	"context"
	"github.com/ceyewan/gochat/im-infra/clog"
)

// NewProducer 创建一个新的生产者实例。
// serviceName 用于从配置中心获取对应的 Kafka 配置。
func NewProducer(ctx context.Context, serviceName string, opts ...Option) (Producer, error)

// NewConsumer 创建一个新的消费者实例。
// serviceName 和 groupID 用于从配置中心获取配置并标识消费者组。
func NewConsumer(ctx context.Context, serviceName, groupID string, opts ...Option) (Consumer, error)

// Option 是用于配置组件的可选参数。
type Option func(*options)

// WithLogger 注入一个自定义的 logger。
func WithLogger(logger clog.Logger) Option
```

### 3.2 生产者示例

```go
// 1. 初始化
// serviceName "im-gateway" 将用于从 coord 加载配置
producer, err := mq.NewProducer(context.Background(), "im-gateway", mq.WithLogger(logger))
if err != nil {
    log.Fatal(err)
}
defer producer.Close()

// 2. 准备消息
msg := &mq.Message{
    Topic:   "gochat.messages.upstream",
    Key:     []byte(userID),
    Value:   data,
    Headers: map[string][]byte{"trace_id": []byte(traceID)},
}

// 3. 异步发送 (推荐)
producer.Send(context.Background(), msg, func(err error) {
    if err != nil {
        logger.Error("failed to send message", "error", err)
    }
})

// 4. 或同步发送
if err := producer.SendSync(context.Background(), msg); err != nil {
    logger.Error("failed to send message sync", "error", err)
}
```

### 3.3 消费者示例

```go
// 1. 初始化
// serviceName "im-logic" 和 groupID "im-logic-group" 将用于从 coord 加载配置和标识消费者组
consumer, err := mq.NewConsumer(context.Background(), "im-logic", "im-logic-group", mq.WithLogger(logger))
if err != nil {
    log.Fatal(err)
}
defer consumer.Close()

// 2. 定义处理逻辑
handler := func(ctx context.Context, msg *mq.Message) error {
    traceID := string(msg.Headers["trace_id"])
    logger.Info("message received", "topic", msg.Topic, "key", string(msg.Key), "trace_id", traceID)
    // ... 业务处理 ...
    return nil
}

// 3. 开始订阅 (阻塞)
topics := []string{"gochat.messages.upstream"}
if err := consumer.Subscribe(context.Background(), topics, handler); err != nil {
    // Subscribe 只有在不可恢复的错误下才会返回 error
    logger.Fatal("consumer subscribe failed", "error", err)
}
```

## 4. 配置

`mq` 组件的所有配置均由 `coord` 配置中心管理。标准的配置文件路径为 `kafka.yaml`。详细的配置结构和字段说明，请参考 **[配置契约](../../config/dev/kafka.md)**。