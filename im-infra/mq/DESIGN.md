# MQ 组件接口设计 (V2)

## 1. 设计哲学

当前版本的 `mq` 库暴露了过多的底层细节，增加了使用者的心智负担，违背了 `im-infra` 作为基础设施的初衷。V2 版本的设计核心是 **“极简与约定”**。

- **极简 (Simplicity)**: 提供一个极小化的 API 集合，只包含生产者和消费者的核心功能。目标是让业务开发者在不阅读 Kafka 文档的情况下，也能轻松收发消息。
- **约定 (Convention)**: 组件将遵循 `gochat` 项目的最佳实践。
    - **配置中心驱动**: 所有配置均从 `coord` 获取，业务代码不直接接触配置结构体。
    - **自动偏移量管理**: 消费者自动处理 offset 提交，业务逻辑只需关注消息处理本身。
    - **内置可观测性**: 自动与 `clog` 集成，并预留 `trace_id` 的传递机制。

旧的 `API.md` 文档将被废弃，本文档是未来重构的唯一真相。

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

## 3. 工厂函数与配置

组件的实例化将完全依赖 `im-infra` 的标准模式：通过 `Option` 注入依赖，通过 `coord` 获取配置。

```go
package mq

import (
	"context"
	"github.com/ceyewan/gochat/im-infra/clog"
)

// NewProducer 创建一个新的生产者实例。
// serviceName 用于从配置中心获取对应的 Kafka 配置。
func NewProducer(ctx context.Context, serviceName string, opts ...Option) (Producer, error) {
	// 1. 创建默认 options
	// 2. 应用 opts (如 WithLogger)
	// 3. 使用 serviceName 从 coord 获取 Kafka broker 地址等配置
	// 4. 初始化 franz-go 客户端
	// 5. 返回 Producer 实例
}

// NewConsumer 创建一个新的消费者实例。
// serviceName 和 groupID 用于从配置中心获取配置并标识消费者组。
func NewConsumer(ctx context.Context, serviceName, groupID string, opts ...Option) (Consumer, error) {
	// 1. 创建默认 options
	// 2. 应用 opts (如 WithLogger)
	// 3. 使用 serviceName 从 coord 获取 Kafka broker 地址
	// 4. 使用 groupID 初始化消费者组
	// 5. 初始化 franz-go 客户端
	// 6. 返回 Consumer 实例
}

// Option 是用于配置组件的可选参数。
type Option func(*options)

// WithLogger 注入一个自定义的 logger。
func WithLogger(logger clog.Logger) Option {
	// ...
}
```

## 4. 使用示例

### 生产者

```go
// 1. 初始化
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

### 消费者

```go
// 1. 初始化
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

## 5. Topic 管理

Topic 的创建和管理是运维操作，不应与业务逻辑耦合。我们提供一个独立的管理工具或方法来处理。

```go
package mq

// AdminClient 提供了 Topic 管理等运维功能。
type AdminClient interface {
	CreateTopic(ctx context.Context, topicName string, partitions int, replicationFactor int) error
	Close() error
}

// NewAdminClient 创建一个新的管理客户端。
func NewAdminClient(ctx context.Context, serviceName string, opts ...Option) (AdminClient, error) {
    // ...
}

## 6. 配置规范 (`kafka.yaml`)

`mq` 组件的所有配置都应存储在 `coord` 配置中心的一个名为 `kafka.yaml` 的文件中。组件在初始化时（`NewProducer`/`NewConsumer`）会根据传入的 `serviceName` 自动加载此配置。

以下是该配置文件的标准结构：

```yaml
# Kafka/MQ 组件的统一配置文件
#
# 本文件是 gochat 项目中所有 Kafka 相关配置的唯一真相来源。
# mq 组件将通过 coord 配置中心读取此文件。

# 通用设置，适用于所有客户端
common:
  # Kafka broker 地址列表
  brokers:
    - "kafka-1:9092"
    - "kafka-2:9092"
    - "kafka-3:9092"
  # 客户端ID前缀，最终的 ClientID 会是 <clientIDPrefix>-<serviceName>
  clientIDPrefix: "gochat"
  # 安全协议相关配置 (SASL, SSL)，留空表示不启用
  security:
    sasl:
      mechanism: "" # e.g., "PLAIN", "SCRAM-SHA-256"
      username: ""
      password: ""
    ssl:
      ca_cert_file: ""
      client_cert_file: ""
      client_key_file: ""
      insecure_skip_verify: false

# 生产者的默认配置
# 这些配置可以被特定服务的生产者配置覆盖
producer:
  # 确认级别: 0=不等待, 1=等待leader, -1=等待所有副本
  requiredAcks: -1
  # 是否启用幂等性，强烈建议保持 true
  enableIdempotence: true
  # 压缩算法: "none", "gzip", "snappy", "lz4", "zstd"
  compression: "lz4"
  # 批处理大小 (字节)
  batchSize: 16384 # 16KB
  # 批处理等待时间 (毫秒)，增加此值可提高吞吐量，但会增加延迟
  lingerMs: 5

# 消费者的默认配置
consumer:
  # 自动提交偏移量的间隔
  autoCommitIntervalMs: 5000 # 5 seconds
  # 当没有初始偏移量时，从何处开始消费: "earliest", "latest"
  autoOffsetReset: "latest"
  # 会话超时时间 (毫秒)
  sessionTimeoutMs: 30000
  # 心跳间隔 (毫秒)，应小于 sessionTimeoutMs 的 1/3
  heartbeatIntervalMs: 10000
```