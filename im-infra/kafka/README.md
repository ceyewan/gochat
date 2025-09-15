# Kafka 组件

极简的 Kafka 客户端封装，提供简洁的消息生产和消费接口。

## 特性

- 🚀 极简 API：只包含核心的生产者和消费者功能
- 🔄 自动追踪：内置 trace_id 自动传播机制
- 🛡️ 错误处理：消费者处理失败时会自动重试
- 📝 结构化日志：与 clog 组件深度集成
- 🔧 配置驱动：支持开发环境和生产环境的优化配置
- 📊 性能监控：内置指标收集和健康检查
- 🔄 优雅关闭：支持优雅关闭和上下文取消
- ⚡ 高性能：支持批量发送、压缩和连接池优化

## 快速开始

### 前置条件

在运行示例之前，请确保：

1. **Kafka 服务正在运行**
2. **安装了 Kafka 命令行工具** (`kafka-topics.sh` 等)

### 自动创建 Topics

示例程序会自动创建必要的 topics，如果失败的话，可以手动创建：

```bash
# 使用项目脚本创建测试 topics
cd /Users/harrick/CodeField/gochat/deployment/scripts
./init-kafka-example.sh

# 或者使用 admin 脚本创建
./kafka-admin.sh create example.user.events
./kafka-admin.sh create example.test-topic
./kafka-admin.sh create example.performance
./kafka-admin.sh create example.dead-letter
```

### 基本初始化

```go
package main

import (
    "context"
    "log"

    "github.com/ceyewan/gochat/im-infra/clog"
    "github.com/ceyewan/gochat/im-infra/kafka"
)

func main() {
    // 1. 初始化 clog
    clog.Init(context.Background(), clog.GetDefaultConfig("development"))

    // 2. 获取默认配置
    config := kafka.GetDefaultConfig("development")

    // 3. 覆盖必要的配置
    config.Brokers = []string{"localhost:9092"}

    // 4. 创建 Producer
    producer, err := kafka.NewProducer(
        context.Background(),
        config,
        kafka.WithNamespace("kafka-producer"),
    )
    if err != nil {
        log.Fatal(err)
    }
    defer producer.Close()

    // 5. 创建 Consumer
    consumer, err := kafka.NewConsumer(
        context.Background(),
        config,
        "my-service-group",
        kafka.WithNamespace("kafka-consumer"),
    )
    if err != nil {
        log.Fatal(err)
    }
    defer consumer.Close()
}
```

### 发送消息

```go
// 异步发送消息（推荐）
func sendMessage(producer kafka.Producer, userID string, userData []byte) {
    ctx := context.Background()

    msg := &kafka.Message{
        Topic: "user.events.registered",
        Key:   []byte(userID),
        Value: userData,
    }

    producer.Send(ctx, msg, func(err error) {
        if err != nil {
            clog.WithContext(ctx).Error("发送用户注册事件失败", clog.Err(err))
        } else {
            clog.WithContext(ctx).Info("用户注册事件发送成功", clog.String("user_id", userID))
        }
    })
}

// 同步发送消息（需要强一致性保证）
func sendOrderMessage(producer kafka.Producer, orderID string, orderData []byte) error {
    ctx := context.Background()

    msg := &kafka.Message{
        Topic: "order.events.created",
        Key:   []byte(orderID),
        Value: orderData,
    }

    return producer.SendSync(ctx, msg)
}
```

### 消费消息

```go
func startConsuming(consumer kafka.Consumer) {
    ctx := context.Background()

    handler := func(ctx context.Context, msg *kafka.Message) error {
        logger := clog.WithContext(ctx)
        logger.Info("收到新消息",
            clog.String("topic", msg.Topic),
            clog.String("key", string(msg.Key)),
        )

        // 处理业务逻辑
        if err := processMessage(msg); err != nil {
            logger.Error("处理消息失败", clog.Err(err))
            return err // 返回错误，消息会被重新消费
        }

        logger.Info("消息处理成功")
        return nil // 返回 nil，偏移量会被提交
    }

    topics := []string{"user.events", "order.events"}

    // 启动消费者（会阻塞）
    if err := consumer.Subscribe(ctx, topics, handler); err != nil {
        log.Fatal("消费者订阅失败", err)
    }
}
```

## 配置说明

### 开发环境配置

```go
config := kafka.GetDefaultConfig("development")
// 结果：
// - Brokers: ["localhost:9092"]
// - SecurityProtocol: "PLAINTEXT"
// - ProducerConfig: Acks=1, RetryMax=3, BatchSize=16384
// - ConsumerConfig: AutoOffsetReset="latest", EnableAutoCommit=true
```

### 生产环境配置

```go
config := kafka.GetDefaultConfig("production")
// 结果：
// - Brokers: ["kafka1:9092", "kafka2:9092", "kafka3:9092"]
// - SecurityProtocol: "SASL_SSL"
// - ProducerConfig: Acks=-1, RetryMax=10, BatchSize=65536
// - ConsumerConfig: AutoOffsetReset="earliest", EnableAutoCommit=true
```

## Trace ID 传播

组件自动处理 trace_id 在消息传递过程中的传播：

**发送端**：
- 自动从 context 中提取 trace_id
- 将 trace_id 作为消息头 `X-Trace-ID` 注入

**接收端**：
- 从消息头中提取 trace_id
- 将 trace_id 注入到处理函数的 context 中
- 使用 `clog.WithContext(ctx)` 自动记录 trace_id

```go
// 发送端
ctx := clog.WithTraceID(context.Background(), "abc123")
producer.Send(ctx, msg, callback) // trace_id 会自动注入

// 接收端
handler := func(ctx context.Context, msg *kafka.Message) error {
    logger := clog.WithContext(ctx) // 日志会自动包含 trace_id
    logger.Info("处理消息") // 输出: {"level":"info","msg":"处理消息","trace_id":"abc123"}
    return nil
}
```

## 死信队列设计

当前版本不实现死信队列，但预留了扩展能力：

```go
// 消费者处理失败时的建议模式
handler := func(ctx context.Context, msg *kafka.Message) error {
    // 1. 记录错误日志
    clog.WithContext(ctx).Error("处理消息失败", clog.Err(err))

    // 2. 业务层重试逻辑
    if shouldRetry(msg) {
        return err // 返回错误，Kafka 会重新投递
    }

    // 3. 发送到死信队列（业务代码实现）
    if err := sendToDeadLetterQueue(ctx, msg); err != nil {
        return err
    }

    // 4. 成功处理，提交偏移量
    return nil
}
```

## 管理 Topics

### 创建 Topics

```go
// 创建单个 topic
admin, err := kafka.NewAdminClient(ctx, config)
if err != nil {
    log.Fatal("创建 admin 客户端失败:", err)
}
defer admin.Close()

err = admin.CreateTopic(ctx, "my-topic", 3, 1)
if err != nil {
    log.Fatal("创建 topic 失败:", err)
}

// 批量创建 topics
topics := []kafka.TopicConfig{
    {
        Name:             "topic1",
        Partitions:       3,
        ReplicationFactor: 1,
    },
    {
        Name:             "topic2",
        Partitions:       6,
        ReplicationFactor: 1,
    },
}

err = admin.CreateTopics(ctx, topics)
if err != nil {
    log.Fatal("批量创建 topics 失败:", err)
}
```

### 管理 Topics

```go
// 检查 topic 是否存在
exists := admin.TopicExists(ctx, "my-topic")

// 列出所有 topics
topics, err := admin.ListTopics(ctx)
if err != nil {
    log.Fatal("获取 topic 列表失败:", err)
}

// 删除 topic
err = admin.DeleteTopic(ctx, "old-topic")
if err != nil {
    log.Fatal("删除 topic 失败:", err)
}
```

### 快速创建示例 Topics

```go
// 一键创建所有示例 topics
err := kafka.CreateExampleTopics(ctx, config)
if err != nil {
    log.Fatal("创建示例 topics 失败:", err)
}
```

## 监控和健康检查

### 生产者监控
```go
// 获取性能指标
metrics := producer.GetMetrics()
fmt.Printf("成功率: %.2f%%\n", metrics["success_rate"])
fmt.Printf("总消息数: %d\n", metrics["total_messages"])

// 健康检查
if err := producer.Ping(ctx); err != nil {
    log.Fatal("生产器不健康:", err)
}
```

### 消费者监控
```go
// 获取性能指标
metrics := consumer.GetMetrics()
fmt.Printf("处理成功率: %.2f%%\n", metrics["success_rate"])
fmt.Printf("处理失败数: %d\n", metrics["failed_messages"])

// 健康检查
if err := consumer.Ping(ctx); err != nil {
    log.Fatal("消费者不健康:", err)
}
```

### 最佳实践

### Topic 命名规范
- 使用 `{domain}.{entity}.{event}` 格式
- 例如：`user.events.registered`, `order.events.created`

### Consumer Group 命名
- 使用 `{service}.{purpose}.group` 格式
- 例如：`notification-service.user-events.group`

### 错误处理
- 生产者：使用回调函数处理异步发送错误
- 消费者：返回错误让消息重新消费，业务层实现重试逻辑

### 性能优化
- 生产环境使用批量发送和更大的批处理大小
- 合理设置 linger 时间平衡延迟和吞吐量
- 使用合适的分区数量实现并行消费
- 定期监控指标并调整配置参数