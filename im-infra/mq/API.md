# MQ API 文档

## 概述

`mq` 是一个基于 franz-go 的高性能 Kafka 消息队列基础库，专为即时通讯场景优化。提供了生产者、消费者和连接池的统一管理，支持微秒级延迟和 100,000+ 消息/秒的高吞吐量。

## 核心特性

- 🚀 **高性能**：微秒级延迟，支持 100,000+ 消息/秒吞吐量
- 🔒 **幂等性保证**：内置幂等性支持，确保消息不重复
- 📦 **消息批处理**：智能批处理系统，优化小消息处理性能
- 🗜️ **压缩支持**：支持 LZ4、Snappy、Gzip、Zstd 压缩算法
- 🔄 **连接池管理**：高效的连接复用和健康检查
- 📊 **监控指标**：全面的性能指标收集和健康检查
- 🛡️ **错误处理**：完善的错误类型定义和优雅降级策略

## 快速开始

### 基本用法

#### 全局方法（推荐）

```go
package main

import (
    "context"
    "log"
    "github.com/ceyewan/gochat/im-infra/mq"
)

func main() {
    ctx := context.Background()
    
    // 发送消息
    err := mq.SendSync(ctx, "chat-messages", []byte("Hello, World!"))
    if err != nil {
        log.Fatal(err)
    }
    
    // 订阅消息
    callback := func(message *mq.Message, partition mq.TopicPartition, err error) bool {
        if err != nil {
            log.Printf("消费错误: %v", err)
            return false
        }
        
        log.Printf("收到消息: %s", string(message.Value))
        return true // 继续消费
    }
    
    err = mq.Subscribe(ctx, []string{"chat-messages"}, callback)
    if err != nil {
        log.Fatal(err)
    }
}
```

#### 自定义配置

```go
package main

import (
    "context"
    "log"
    "time"
    "github.com/ceyewan/gochat/im-infra/mq"
)

func main() {
    // 创建自定义配置
    cfg := mq.Config{
        Brokers:  []string{"localhost:9092", "localhost:9093"},
        ClientID: "my-chat-app",
        ProducerConfig: mq.ProducerConfig{
            Compression:         "lz4",
            BatchSize:           16384,
            LingerMs:            5,
            EnableIdempotence:   true,
            MaxInFlightRequests: 5,
        },
        ConsumerConfig: mq.ConsumerConfig{
            GroupID:            "chat-consumer-group",
            AutoOffsetReset:    "latest",
            EnableAutoCommit:   true,
            AutoCommitInterval: 5 * time.Second,
            MaxPollRecords:     500,
        },
    }
    
    // 创建 MQ 实例
    mqInstance, err := mq.New(cfg)
    if err != nil {
        log.Fatal(err)
    }
    defer mqInstance.Close()
    
    // 获取生产者
    producer := mqInstance.Producer()
    
    // 发送消息
    ctx := context.Background()
    err = producer.SendSync(ctx, "chat-messages", []byte("Hello from custom configimpl!"))
    if err != nil {
        log.Fatal(err)
    }
}
```

## API 参考

### 全局方法

#### 生产者方法

```go
// SendSync 同步发送消息
func SendSync(ctx context.Context, topic string, message []byte) error

// SendAsync 异步发送消息
func SendAsync(ctx context.Context, topic string, message []byte, callback func(error))

// SendBatchSync 同步发送消息批次
func SendBatchSync(ctx context.Context, batch MessageBatch) ([]ProduceResult, error)

// SendBatchAsync 异步发送消息批次
func SendBatchAsync(ctx context.Context, batch MessageBatch, callback func([]ProduceResult, error))
```

#### 消费者方法

```go
// Subscribe 订阅主题
func Subscribe(ctx context.Context, topics []string, callback ConsumeCallback) error

// Unsubscribe 取消订阅主题
func Unsubscribe(topics []string) error

// CommitOffset 提交偏移量
func CommitOffset(ctx context.Context, topic string, partition int32, offset int64) error
```

#### 连接池方法

```go
// GetConnection 获取连接
func GetConnection(ctx context.Context) (interface{}, error)

// ReleaseConnection 释放连接
func ReleaseConnection(conn interface{}) error

// Ping 检查连接健康状态
func Ping(ctx context.Context) error
```

### 核心接口

#### Producer 接口

```go
type Producer interface {
    // 同步发送方法
    SendSync(ctx context.Context, topic string, message []byte) error
    SendSyncWithKey(ctx context.Context, topic string, key []byte, message []byte) error
    SendSyncWithHeaders(ctx context.Context, topic string, key []byte, message []byte, headers map[string][]byte) error
    
    // 异步发送方法
    SendAsync(ctx context.Context, topic string, message []byte, callback func(error))
    SendAsyncWithKey(ctx context.Context, topic string, key []byte, message []byte, callback func(error))
    SendAsyncWithHeaders(ctx context.Context, topic string, key []byte, message []byte, headers map[string][]byte, callback func(error))
    
    // 批处理方法
    SendBatchSync(ctx context.Context, batch MessageBatch) ([]ProduceResult, error)
    SendBatchAsync(ctx context.Context, batch MessageBatch, callback func([]ProduceResult, error))
    
    // 管理方法
    Flush(ctx context.Context) error
    Close() error
    GetMetrics() ProducerMetrics
}
```

#### Consumer 接口

```go
type Consumer interface {
    // 订阅方法
    Subscribe(ctx context.Context, topics []string, callback ConsumeCallback) error
    SubscribePattern(ctx context.Context, pattern string, callback ConsumeCallback) error
    Unsubscribe(topics []string) error
    UnsubscribeAll() error
    
    // 暂停和恢复
    Pause(topicPartitions []TopicPartition) error
    Resume(topicPartitions []TopicPartition) error
    
    // 偏移量管理
    CommitOffset(ctx context.Context, topic string, partition int32, offset int64) error
    CommitOffsets(ctx context.Context, offsets map[TopicPartition]int64) error
    GetCommittedOffset(ctx context.Context, topic string, partition int32) (int64, error)
    GetCurrentOffset(topic string, partition int32) (int64, error)
    
    // 位置控制
    Seek(topic string, partition int32, offset int64) error
    SeekToBeginning(topicPartitions []TopicPartition) error
    SeekToEnd(topicPartitions []TopicPartition) error
    
    // 管理方法
    Close() error
    GetMetrics() ConsumerMetrics
}
```

### 配置结构

#### 主配置

```go
type Config struct {
    Brokers          []string          // Kafka broker地址列表
    ClientID         string            // 客户端标识符
    SecurityProtocol string            // 安全协议
    SASL             SASLConfig        // SASL配置
    SSL              SSLConfig         // SSL配置
    Connection       ConnectionConfig  // 连接配置
    ProducerConfig   ProducerConfig    // 生产者配置
    ConsumerConfig   ConsumerConfig    // 消费者配置
    PoolConfig       PoolConfig        // 连接池配置
    Performance      PerformanceConfig // 性能配置
    Monitoring       MonitoringConfig  // 监控配置
}
```

#### 生产者配置

```go
type ProducerConfig struct {
    Compression         string        // 压缩算法: "none", "gzip", "snappy", "lz4", "zstd"
    BatchSize           int           // 批次大小（字节）
    LingerMs            int           // 批次等待时间（毫秒）
    MaxMessageBytes     int           // 单条消息最大大小
    RequiredAcks        int           // 确认级别: 0, 1, -1
    RequestTimeout      time.Duration // 请求超时时间
    EnableIdempotence   bool          // 是否启用幂等性
    MaxInFlightRequests int           // 最大飞行中请求数
    RetryBackoff        time.Duration // 重试间隔
    MaxRetries          int           // 最大重试次数
}
```

#### 消费者配置

```go
type ConsumerConfig struct {
    GroupID            string        // 消费者组ID
    AutoOffsetReset    string        // 自动偏移量重置策略: "earliest", "latest", "none"
    EnableAutoCommit   bool          // 是否启用自动提交偏移量
    AutoCommitInterval time.Duration // 自动提交间隔
    SessionTimeout     time.Duration // 会话超时时间
    HeartbeatInterval  time.Duration // 心跳间隔
    MaxPollRecords     int           // 单次拉取最大记录数
    MaxPollInterval    time.Duration // 最大拉取间隔
    FetchMinBytes      int           // 拉取最小字节数
    FetchMaxBytes      int           // 拉取最大字节数
    FetchMaxWait       time.Duration // 拉取最大等待时间
    IsolationLevel     string        // 隔离级别: "read_uncommitted", "read_committed"
}
```

### 数据结构

#### 消息结构

```go
type Message struct {
    Topic     string            // 主题名称
    Partition int32             // 分区号
    Offset    int64             // 偏移量
    Key       []byte            // 消息键
    Value     []byte            // 消息值
    Headers   map[string][]byte // 消息头部
    Timestamp time.Time         // 消息时间戳
}
```

#### 消息批次

```go
type MessageBatch struct {
    Messages      []*Message // 消息列表
    MaxBatchSize  int        // 最大批次大小（字节）
    MaxBatchCount int        // 最大批次消息数量
    LingerMs      int        // 批次等待时间（毫秒）
}
```

#### 生产结果

```go
type ProduceResult struct {
    Topic     string        // 主题名称
    Partition int32         // 分区号
    Offset    int64         // 消息偏移量
    Error     error         // 发送错误
    Latency   time.Duration // 发送延迟
}
```

### 回调函数类型

```go
// ConsumeCallback 消费回调函数
// 返回值：true 继续消费，false 停止消费
type ConsumeCallback func(message *Message, partition TopicPartition, err error) bool

// ErrorHandler 错误处理函数
type ErrorHandler func(error)
```

## 默认配置

### 获取默认配置

```go
// 获取默认主配置
cfg := mq.DefaultConfig()

// 获取默认生产者配置
producerCfg := mq.DefaultProducerConfig()

// 获取默认消费者配置
consumerCfg := mq.DefaultConsumerConfig()
```

### 默认值说明

- **压缩算法**: LZ4（低延迟优化）
- **批次大小**: 16KB
- **等待时间**: 5毫秒（微秒级延迟优化）
- **幂等性**: 启用
- **自动提交**: 启用，间隔5秒
- **最大拉取记录数**: 500（高吞吐量优化）

## 工厂方法

### 创建独立组件

```go
// 创建独立的生产者
producer, err := mq.NewProducer(producerConfig)

// 创建独立的消费者
consumer, err := mq.NewConsumer(consumerConfig)

// 创建独立的连接池
pool, err := mq.NewConnectionPool(config)
```

### 使用默认实例

```go
// 使用默认配置的 MQ 实例
mqInstance := mq.Default()

// 检查连接健康状态
err := mq.Ping(context.Background())
```

## 使用示例

### 即时通讯场景

#### 聊天消息发送

```go
package main

import (
    "context"
    "encoding/json"
    "log"
    "time"
    "github.com/ceyewan/gochat/im-infra/mq"
)

// ChatMessage 聊天消息结构
type ChatMessage struct {
    MessageID string    `json:"message_id"`
    FromUser  string    `json:"from_user"`
    ToUser    string    `json:"to_user"`
    Content   string    `json:"content"`
    Timestamp time.Time `json:"timestamp"`
    MessageType string  `json:"message_type"` // text, image, file, etc.
}

func sendChatMessage() {
    // 创建聊天消息
    msg := ChatMessage{
        MessageID:   "msg_123456",
        FromUser:    "user_001",
        ToUser:      "user_002",
        Content:     "你好，这是一条测试消息",
        Timestamp:   time.Now(),
        MessageType: "text",
    }

    // 序列化消息
    data, err := json.Marshal(msg)
    if err != nil {
        log.Fatal(err)
    }

    // 发送消息（使用用户ID作为分区键确保消息顺序）
    ctx := context.Background()
    err = mq.SendSync(ctx, "chat-messages", data)
    if err != nil {
        log.Printf("发送聊天消息失败: %v", err)
        return
    }

    log.Printf("聊天消息发送成功: %s -> %s", msg.FromUser, msg.ToUser)
}
```

#### 聊天消息消费

```go
func consumeChatMessages() {
    callback := func(message *mq.Message, partition mq.TopicPartition, err error) bool {
        if err != nil {
            log.Printf("消费消息错误: %v", err)
            return true // 继续消费其他消息
        }

        // 反序列化消息
        var chatMsg ChatMessage
        if err := json.Unmarshal(message.Value, &chatMsg); err != nil {
            log.Printf("反序列化消息失败: %v", err)
            return true
        }

        // 处理聊天消息
        log.Printf("收到聊天消息: %s -> %s: %s",
            chatMsg.FromUser, chatMsg.ToUser, chatMsg.Content)

        // 这里可以添加业务逻辑：
        // 1. 推送给在线用户
        // 2. 存储到数据库
        // 3. 更新未读消息计数

        return true // 继续消费
    }

    ctx := context.Background()
    err := mq.Subscribe(ctx, []string{"chat-messages"}, callback)
    if err != nil {
        log.Fatal(err)
    }
}
```

### 高性能批处理

#### 批量发送消息

```go
func sendMessageBatch() {
    // 创建消息批次
    batch := mq.MessageBatch{
        Messages:      make([]*mq.Message, 0, 100),
        MaxBatchSize:  16384, // 16KB
        MaxBatchCount: 100,
        LingerMs:      5,
    }

    // 添加多条消息到批次
    for i := 0; i < 100; i++ {
        msg := ChatMessage{
            MessageID:   fmt.Sprintf("batch_msg_%d", i),
            FromUser:    "system",
            ToUser:      fmt.Sprintf("user_%03d", i),
            Content:     fmt.Sprintf("批量消息 #%d", i),
            Timestamp:   time.Now(),
            MessageType: "text",
        }

        data, _ := json.Marshal(msg)

        message := &mq.Message{
            Topic: "chat-messages",
            Key:   []byte(msg.ToUser), // 使用接收用户作为分区键
            Value: data,
            Headers: map[string][]byte{
                "message_type": []byte(msg.MessageType),
                "from_user":    []byte(msg.FromUser),
            },
        }

        batch.Messages = append(batch.Messages, message)
    }

    // 同步发送批次
    ctx := context.Background()
    results, err := mq.SendBatchSync(ctx, batch)
    if err != nil {
        log.Printf("批量发送失败: %v", err)
        return
    }

    // 检查发送结果
    successCount := 0
    for _, result := range results {
        if result.Error == nil {
            successCount++
        } else {
            log.Printf("消息发送失败: %v", result.Error)
        }
    }

    log.Printf("批量发送完成: %d/%d 成功", successCount, len(results))
}
```

### 异步处理

#### 异步发送消息

```go
func sendAsyncMessage() {
    msg := ChatMessage{
        MessageID:   "async_msg_001",
        FromUser:    "user_001",
        ToUser:      "user_002",
        Content:     "异步发送的消息",
        Timestamp:   time.Now(),
        MessageType: "text",
    }

    data, _ := json.Marshal(msg)

    // 异步发送消息
    ctx := context.Background()
    mq.SendAsync(ctx, "chat-messages", data, func(err error) {
        if err != nil {
            log.Printf("异步发送失败: %v", err)
            // 这里可以添加重试逻辑或错误处理
        } else {
            log.Printf("异步发送成功: %s", msg.MessageID)
            // 这里可以添加成功后的处理逻辑
        }
    })

    log.Printf("异步发送请求已提交: %s", msg.MessageID)
}
```

### 错误处理和重试

#### 带重试的消息发送

```go
func sendMessageWithRetry(message []byte, maxRetries int) error {
    ctx := context.Background()

    for attempt := 0; attempt <= maxRetries; attempt++ {
        err := mq.SendSync(ctx, "chat-messages", message)
        if err == nil {
            return nil // 发送成功
        }

        // 检查是否为可重试错误
        if !mq.IsRetryableError(err) {
            return fmt.Errorf("不可重试错误: %w", err)
        }

        if attempt < maxRetries {
            // 指数退避重试
            backoff := time.Duration(1<<attempt) * 100 * time.Millisecond
            log.Printf("发送失败，%v 后重试 (尝试 %d/%d): %v",
                backoff, attempt+1, maxRetries, err)
            time.Sleep(backoff)
        }
    }

    return fmt.Errorf("重试次数耗尽，最后错误: %w", err)
}
```

### 监控和指标

#### 获取性能指标

```go
func monitorPerformance() {
    // 使用自定义配置创建MQ实例以获取详细指标
    cfg := mq.DefaultConfig()
    mqInstance, err := mq.New(cfg)
    if err != nil {
        log.Fatal(err)
    }
    defer mqInstance.Close()

    // 定期收集指标
    ticker := time.NewTicker(10 * time.Second)
    defer ticker.Stop()

    for range ticker.C {
        // 获取生产者指标
        producerMetrics := mqInstance.Producer().GetMetrics()
        log.Printf("生产者指标:")
        log.Printf("  总消息数: %d", producerMetrics.TotalMessages)
        log.Printf("  成功消息数: %d", producerMetrics.SuccessMessages)
        log.Printf("  失败消息数: %d", producerMetrics.FailedMessages)
        log.Printf("  平均延迟: %v", producerMetrics.AverageLatency)
        log.Printf("  最大延迟: %v", producerMetrics.MaxLatency)
        log.Printf("  吞吐量: %.2f 消息/秒", producerMetrics.MessagesPerSecond)

        // 获取消费者指标
        consumerMetrics := mqInstance.Consumer().GetMetrics()
        log.Printf("消费者指标:")
        log.Printf("  总消息数: %d", consumerMetrics.TotalMessages)
        log.Printf("  消费延迟: %d", consumerMetrics.Lag)
        log.Printf("  吞吐量: %.2f 消息/秒", consumerMetrics.MessagesPerSecond)

        // 获取连接池统计
        poolStats := mqInstance.ConnectionPool().GetStats()
        log.Printf("连接池统计:")
        log.Printf("  总连接数: %d", poolStats.TotalConnections)
        log.Printf("  活跃连接数: %d", poolStats.ActiveConnections)
        log.Printf("  空闲连接数: %d", poolStats.IdleConnections)
    }
}
```

### 健康检查

#### 实现健康检查

```go
func healthCheck() {
    ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
    defer cancel()

    err := mq.Ping(ctx)
    if err != nil {
        log.Printf("健康检查失败: %v", err)
        // 这里可以添加告警逻辑
        return
    }

    log.Println("MQ 健康检查通过")
}

// 定期健康检查
func startHealthCheck() {
    ticker := time.NewTicker(30 * time.Second)
    defer ticker.Stop()

    for range ticker.C {
        healthCheck()
    }
}
```

## 最佳实践

### 性能优化

#### 1. 选择合适的压缩算法

```go
// 对于低延迟场景，推荐使用 LZ4
cfg.ProducerConfig.Compression = "lz4"

// 对于高压缩比场景，可以使用 Snappy
cfg.ProducerConfig.Compression = "snappy"

// 对于网络带宽受限场景，可以使用 Gzip
cfg.ProducerConfig.Compression = "gzip"
```

#### 2. 优化批处理设置

```go
// 针对即时通讯的小消息优化
cfg.ProducerConfig.BatchSize = 16384    // 16KB 批次大小
cfg.ProducerConfig.LingerMs = 5         // 5毫秒等待时间，平衡延迟和吞吐量

// 针对高吞吐量场景
cfg.ProducerConfig.BatchSize = 65536    // 64KB 批次大小
cfg.ProducerConfig.LingerMs = 10        // 10毫秒等待时间
```

#### 3. 调优消费者设置

```go
// 高吞吐量消费配置
cfg.ConsumerConfig.MaxPollRecords = 1000     // 增加单次拉取记录数
cfg.ConsumerConfig.FetchMaxBytes = 52428800  // 50MB 最大拉取大小
cfg.ConsumerConfig.FetchMaxWait = 500 * time.Millisecond

// 低延迟消费配置
cfg.ConsumerConfig.MaxPollRecords = 100      // 减少单次拉取记录数
cfg.ConsumerConfig.FetchMaxWait = 100 * time.Millisecond
```

#### 4. 连接池优化

```go
// 根据并发需求调整连接池大小
cfg.PoolConfig.MaxConnections = 20           // 最大连接数
cfg.PoolConfig.MinIdleConnections = 5        // 最小空闲连接数
cfg.PoolConfig.MaxIdleConnections = 10       // 最大空闲连接数
cfg.PoolConfig.ConnectionMaxLifetime = time.Hour
cfg.PoolConfig.HealthCheckInterval = 30 * time.Second
```

### 错误处理策略

#### 1. 分类错误处理

```go
func handleError(err error) {
    if mq.IsFatalError(err) {
        // 致命错误：停止服务，发送告警
        log.Fatalf("致命错误，服务停止: %v", err)
    } else if mq.IsRetryableError(err) {
        // 可重试错误：实施重试策略
        log.Printf("可重试错误: %v", err)
        // 实施指数退避重试
    } else {
        // 其他错误：记录日志，继续处理
        log.Printf("一般错误: %v", err)
    }
}
```

#### 2. 实现断路器模式

```go
type CircuitBreaker struct {
    failureCount    int
    failureThreshold int
    resetTimeout    time.Duration
    lastFailureTime time.Time
    state          string // "closed", "open", "half-open"
    mu             sync.Mutex
}

func (cb *CircuitBreaker) Call(fn func() error) error {
    cb.mu.Lock()
    defer cb.mu.Unlock()

    if cb.state == "open" {
        if time.Since(cb.lastFailureTime) > cb.resetTimeout {
            cb.state = "half-open"
        } else {
            return errors.New("断路器开启，拒绝请求")
        }
    }

    err := fn()
    if err != nil {
        cb.failureCount++
        cb.lastFailureTime = time.Now()

        if cb.failureCount >= cb.failureThreshold {
            cb.state = "open"
        }
        return err
    }

    // 成功时重置
    cb.failureCount = 0
    cb.state = "closed"
    return nil
}
```

### 消息设计原则

#### 1. 消息结构设计

```go
// 推荐的消息结构
type StandardMessage struct {
    // 元数据
    MessageID   string            `json:"message_id"`   // 全局唯一ID
    Timestamp   time.Time         `json:"timestamp"`    // 消息时间戳
    Version     string            `json:"version"`      // 消息版本
    Source      string            `json:"source"`       // 消息来源

    // 业务数据
    EventType   string            `json:"event_type"`   // 事件类型
    Payload     interface{}       `json:"payload"`      // 业务负载

    // 路由信息
    TargetUser  string            `json:"target_user"`  // 目标用户
    TargetGroup string            `json:"target_group"` // 目标群组

    // 处理选项
    Priority    int               `json:"priority"`     // 消息优先级
    TTL         time.Duration     `json:"ttl"`          // 消息生存时间
    Retry       bool              `json:"retry"`        // 是否允许重试
}
```

#### 2. 分区策略

```go
// 基于用户ID的分区策略（保证用户消息顺序）
func getUserPartitionKey(userID string) []byte {
    return []byte(userID)
}

// 基于会话ID的分区策略（保证会话消息顺序）
func getSessionPartitionKey(sessionID string) []byte {
    return []byte(sessionID)
}

// 发送消息时使用分区键
err := producer.SendSyncWithKey(ctx, "chat-messages",
    getUserPartitionKey(msg.FromUser), messageData)
```

### 监控和告警

#### 1. 关键指标监控

```go
// 定义监控指标
type MQMetrics struct {
    // 延迟指标
    ProduceLatencyP99 time.Duration
    ConsumeLatencyP99 time.Duration

    // 吞吐量指标
    ProduceRate       float64 // 消息/秒
    ConsumeRate       float64 // 消息/秒

    // 错误率指标
    ProduceErrorRate  float64 // 错误率 %
    ConsumeErrorRate  float64 // 错误率 %

    // 积压指标
    ConsumerLag       int64   // 消费延迟
}

// 监控阈值
const (
    MaxLatencyThreshold    = 10 * time.Millisecond  // 最大延迟阈值
    MinThroughputThreshold = 50000                   // 最小吞吐量阈值
    MaxErrorRateThreshold  = 0.01                    // 最大错误率阈值 1%
    MaxConsumerLagThreshold = 10000                  // 最大消费延迟阈值
)

func checkMetrics(metrics MQMetrics) {
    if metrics.ProduceLatencyP99 > MaxLatencyThreshold {
        sendAlert("生产者延迟过高", metrics.ProduceLatencyP99)
    }

    if metrics.ProduceRate < MinThroughputThreshold {
        sendAlert("生产者吞吐量过低", metrics.ProduceRate)
    }

    if metrics.ProduceErrorRate > MaxErrorRateThreshold {
        sendAlert("生产者错误率过高", metrics.ProduceErrorRate)
    }

    if metrics.ConsumerLag > MaxConsumerLagThreshold {
        sendAlert("消费者延迟过高", metrics.ConsumerLag)
    }
}
```

#### 2. 健康检查实现

```go
func comprehensiveHealthCheck() error {
    ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
    defer cancel()

    // 1. 基本连接检查
    if err := mq.Ping(ctx); err != nil {
        return fmt.Errorf("连接检查失败: %w", err)
    }

    // 2. 生产者健康检查
    testMessage := []byte("health-check-" + time.Now().Format(time.RFC3339))
    if err := mq.SendSync(ctx, "health-check", testMessage); err != nil {
        return fmt.Errorf("生产者检查失败: %w", err)
    }

    // 3. 性能指标检查
    cfg := mq.DefaultConfig()
    mqInstance, err := mq.New(cfg)
    if err != nil {
        return fmt.Errorf("创建MQ实例失败: %w", err)
    }
    defer mqInstance.Close()

    metrics := mqInstance.Producer().GetMetrics()
    if metrics.AverageLatency > 10*time.Millisecond {
        return fmt.Errorf("平均延迟过高: %v", metrics.AverageLatency)
    }

    return nil
}
```

### 部署和运维

#### 1. 环境配置

```go
// 开发环境配置
func getDevelopmentConfig() mq.Config {
    cfg := mq.DefaultConfig()
    cfg.Brokers = []string{"localhost:9092"}
    cfg.ProducerConfig.LingerMs = 0  // 开发环境不等待
    cfg.ConsumerConfig.AutoOffsetReset = "earliest"
    return cfg
}

// 生产环境配置
func getProductionConfig() mq.Config {
    cfg := mq.DefaultConfig()
    cfg.Brokers = []string{
        "kafka-1.prod.com:9092",
        "kafka-2.prod.com:9092",
        "kafka-3.prod.com:9092",
    }
    cfg.ProducerConfig.RequiredAcks = -1  // 等待所有副本确认
    cfg.ProducerConfig.EnableIdempotence = true
    cfg.ConsumerConfig.IsolationLevel = "read_committed"
    return cfg
}
```

#### 2. 优雅关闭

```go
func gracefulShutdown(mqInstance mq.MQ) {
    // 创建关闭信号通道
    sigChan := make(chan os.Signal, 1)
    signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

    // 等待关闭信号
    <-sigChan
    log.Println("收到关闭信号，开始优雅关闭...")

    // 设置关闭超时
    ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
    defer cancel()

    // 停止接收新消息
    log.Println("停止消费新消息...")

    // 刷新待发送的消息
    log.Println("刷新待发送消息...")
    if err := mqInstance.Producer().Flush(ctx); err != nil {
        log.Printf("刷新消息失败: %v", err)
    }

    // 关闭MQ实例
    log.Println("关闭MQ连接...")
    if err := mqInstance.Close(); err != nil {
        log.Printf("关闭MQ失败: %v", err)
    }

    log.Println("优雅关闭完成")
}
```

## 故障排除指南

### 常见问题

#### 1. 连接问题

**问题**: 无法连接到 Kafka broker
```
错误: CONNECTION_FAILED: 连接失败: dial tcp 127.0.0.1:9092: connect: connection refused
```

**解决方案**:
```go
// 检查 broker 地址配置
cfg := mq.DefaultConfig()
cfg.Brokers = []string{
    "kafka-1:9092",
    "kafka-2:9092",
    "kafka-3:9092", // 配置多个 broker 提高可用性
}

// 增加连接超时时间
cfg.Connection.DialTimeout = 30 * time.Second
cfg.Connection.MaxRetries = 5
cfg.Connection.RetryBackoff = 2 * time.Second
```

#### 2. 延迟过高问题

**问题**: 消息发送延迟超过预期
```
平均延迟: 50ms (期望: <1ms)
```

**解决方案**:
```go
// 优化生产者配置
cfg.ProducerConfig.LingerMs = 0           // 立即发送，不等待批次
cfg.ProducerConfig.BatchSize = 1024       // 减小批次大小
cfg.ProducerConfig.Compression = "lz4"    // 使用快速压缩算法

// 优化消费者配置
cfg.ConsumerConfig.FetchMaxWait = 1 * time.Millisecond
cfg.ConsumerConfig.MaxPollRecords = 1     // 减少单次拉取数量
```

#### 3. 吞吐量不足问题

**问题**: 消息吞吐量低于预期
```
当前吞吐量: 10,000 消息/秒 (期望: 100,000+ 消息/秒)
```

**解决方案**:
```go
// 优化批处理设置
cfg.ProducerConfig.BatchSize = 65536      // 增大批次大小
cfg.ProducerConfig.LingerMs = 10          // 适当增加等待时间
cfg.ProducerConfig.MaxInFlightRequests = 10

// 优化消费者设置
cfg.ConsumerConfig.MaxPollRecords = 1000
cfg.ConsumerConfig.FetchMaxBytes = 52428800 // 50MB

// 增加并发处理
func processMessagesParallel() {
    const numWorkers = 10
    messageChan := make(chan *mq.Message, 1000)

    // 启动工作协程
    for i := 0; i < numWorkers; i++ {
        go func() {
            for msg := range messageChan {
                processMessage(msg)
            }
        }()
    }

    // 消费消息并分发给工作协程
    callback := func(message *mq.Message, partition mq.TopicPartition, err error) bool {
        if err != nil {
            return true
        }

        select {
        case messageChan <- message:
        default:
            log.Println("消息通道已满，丢弃消息")
        }
        return true
    }

    mq.Subscribe(context.Background(), []string{"high-throughput-topic"}, callback)
}
```

#### 4. 消费延迟问题

**问题**: 消费者延迟过高
```
消费延迟: 100,000 条消息
```

**解决方案**:
```go
// 增加消费者实例
func scaleConsumers() {
    const numConsumers = 5

    for i := 0; i < numConsumers; i++ {
        go func(consumerID int) {
            cfg := mq.DefaultConsumerConfig()
            cfg.GroupID = "chat-consumer-group"
            cfg.ClientID = fmt.Sprintf("consumer-%d", consumerID)

            consumer, err := mq.NewConsumer(cfg)
            if err != nil {
                log.Fatal(err)
            }
            defer consumer.Close()

            // 消费逻辑...
        }(i)
    }
}

// 优化消费处理
func optimizeConsumerProcessing() {
    cfg := mq.DefaultConsumerConfig()
    cfg.MaxPollRecords = 1000
    cfg.FetchMaxWait = 100 * time.Millisecond

    // 使用批量处理
    var messageBatch []*mq.Message
    const batchSize = 100

    callback := func(message *mq.Message, partition mq.TopicPartition, err error) bool {
        if err != nil {
            return true
        }

        messageBatch = append(messageBatch, message)

        if len(messageBatch) >= batchSize {
            processBatch(messageBatch)
            messageBatch = messageBatch[:0] // 重置批次
        }

        return true
    }

    mq.Subscribe(context.Background(), []string{"chat-messages"}, callback)
}
```

#### 5. 内存使用过高问题

**问题**: 内存使用持续增长
```
内存使用: 2GB+ (期望: <500MB)
```

**解决方案**:
```go
// 限制批次大小和缓冲区
cfg.ProducerConfig.BatchSize = 16384      // 限制批次大小
cfg.ConsumerConfig.MaxPollRecords = 500   // 限制单次拉取数量

// 优化连接池配置
cfg.PoolConfig.MaxConnections = 10
cfg.PoolConfig.ConnectionMaxLifetime = 30 * time.Minute
cfg.PoolConfig.ConnectionMaxIdleTime = 10 * time.Minute

// 定期清理资源
func periodicCleanup() {
    ticker := time.NewTicker(5 * time.Minute)
    defer ticker.Stop()

    for range ticker.C {
        runtime.GC() // 强制垃圾回收

        // 检查内存使用
        var m runtime.MemStats
        runtime.ReadMemStats(&m)

        if m.Alloc > 500*1024*1024 { // 500MB
            log.Printf("内存使用过高: %d MB", m.Alloc/1024/1024)
            // 可以考虑重启或减少负载
        }
    }
}
```

### 性能调优指南

#### 1. 延迟优化

```go
// 超低延迟配置（微秒级）
func getUltraLowLatencyConfig() mq.Config {
    cfg := mq.DefaultConfig()

    // 生产者优化
    cfg.ProducerConfig.LingerMs = 0           // 不等待
    cfg.ProducerConfig.BatchSize = 1024       // 小批次
    cfg.ProducerConfig.Compression = "none"   // 不压缩
    cfg.ProducerConfig.RequiredAcks = 1       // 只等待 leader 确认

    // 消费者优化
    cfg.ConsumerConfig.FetchMaxWait = 1 * time.Millisecond
    cfg.ConsumerConfig.MaxPollRecords = 1
    cfg.ConsumerConfig.FetchMinBytes = 1

    // 连接优化
    cfg.Connection.DialTimeout = 1 * time.Second
    cfg.Connection.ReadTimeout = 1 * time.Second
    cfg.Connection.WriteTimeout = 1 * time.Second

    return cfg
}
```

#### 2. 吞吐量优化

```go
// 高吞吐量配置
func getHighThroughputConfig() mq.Config {
    cfg := mq.DefaultConfig()

    // 生产者优化
    cfg.ProducerConfig.BatchSize = 65536      // 64KB 批次
    cfg.ProducerConfig.LingerMs = 10          // 等待更多消息
    cfg.ProducerConfig.Compression = "lz4"    // 快速压缩
    cfg.ProducerConfig.MaxInFlightRequests = 10

    // 消费者优化
    cfg.ConsumerConfig.MaxPollRecords = 1000
    cfg.ConsumerConfig.FetchMaxBytes = 52428800 // 50MB
    cfg.ConsumerConfig.FetchMaxWait = 500 * time.Millisecond

    // 连接池优化
    cfg.PoolConfig.MaxConnections = 20
    cfg.PoolConfig.MinIdleConnections = 10

    return cfg
}
```

#### 3. 资源使用优化

```go
// 资源节约配置
func getResourceEfficientConfig() mq.Config {
    cfg := mq.DefaultConfig()

    // 减少内存使用
    cfg.ProducerConfig.BatchSize = 8192       // 8KB 批次
    cfg.ConsumerConfig.MaxPollRecords = 100
    cfg.ConsumerConfig.FetchMaxBytes = 1048576 // 1MB

    // 减少连接数
    cfg.PoolConfig.MaxConnections = 5
    cfg.PoolConfig.MaxIdleConnections = 2

    // 启用压缩节省网络带宽
    cfg.ProducerConfig.Compression = "gzip"

    return cfg
}
```

### 监控和告警设置

#### 1. 关键指标监控

```go
// 监控配置
type MonitoringConfig struct {
    MetricsInterval     time.Duration
    AlertThresholds     AlertThresholds
    NotificationChannel string
}

type AlertThresholds struct {
    MaxLatency          time.Duration
    MinThroughput       float64
    MaxErrorRate        float64
    MaxConsumerLag      int64
    MaxMemoryUsage      int64
}

func setupMonitoring() {
    config := MonitoringConfig{
        MetricsInterval: 10 * time.Second,
        AlertThresholds: AlertThresholds{
            MaxLatency:     10 * time.Millisecond,
            MinThroughput:  50000, // 50k 消息/秒
            MaxErrorRate:   0.01,  // 1%
            MaxConsumerLag: 10000,
            MaxMemoryUsage: 500 * 1024 * 1024, // 500MB
        },
        NotificationChannel: "slack://alerts",
    }

    startMonitoring(config)
}
```

#### 2. 自动恢复机制

```go
// 自动恢复配置
type AutoRecoveryConfig struct {
    EnableAutoRestart   bool
    MaxRestartAttempts  int
    RestartBackoff      time.Duration
    HealthCheckInterval time.Duration
}

func setupAutoRecovery(mqInstance mq.MQ) {
    config := AutoRecoveryConfig{
        EnableAutoRestart:   true,
        MaxRestartAttempts:  3,
        RestartBackoff:      30 * time.Second,
        HealthCheckInterval: 10 * time.Second,
    }

    go func() {
        ticker := time.NewTicker(config.HealthCheckInterval)
        defer ticker.Stop()

        restartAttempts := 0

        for range ticker.C {
            ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
            err := mqInstance.Ping(ctx)
            cancel()

            if err != nil {
                log.Printf("健康检查失败: %v", err)

                if config.EnableAutoRestart && restartAttempts < config.MaxRestartAttempts {
                    log.Printf("尝试自动恢复 (第 %d 次)", restartAttempts+1)

                    // 重启逻辑
                    if err := restartMQ(mqInstance); err != nil {
                        log.Printf("自动恢复失败: %v", err)
                        restartAttempts++
                        time.Sleep(config.RestartBackoff)
                    } else {
                        log.Println("自动恢复成功")
                        restartAttempts = 0
                    }
                }
            } else {
                restartAttempts = 0 // 重置重启计数
            }
        }
    }()
}

func restartMQ(mqInstance mq.MQ) error {
    // 关闭现有实例
    if err := mqInstance.Close(); err != nil {
        log.Printf("关闭MQ实例失败: %v", err)
    }

    // 等待一段时间
    time.Sleep(5 * time.Second)

    // 重新创建实例
    cfg := mq.DefaultConfig()
    newInstance, err := mq.New(cfg)
    if err != nil {
        return fmt.Errorf("重新创建MQ实例失败: %w", err)
    }

    // 替换实例（这里需要根据实际架构调整）
    mqInstance = newInstance

    return nil
}
```

## 版本兼容性

### 支持的 Kafka 版本

- Kafka 2.8+
- Kafka 3.0+
- Kafka 3.1+
- Kafka 3.2+

### Go 版本要求

- Go 1.19+
- Go 1.20+
- Go 1.21+

## 许可证

本项目采用 MIT 许可证。详见 LICENSE 文件。

## 贡献指南

欢迎提交 Issue 和 Pull Request。在提交代码前，请确保：

1. 代码通过所有测试
2. 添加了适当的单元测试
3. 更新了相关文档
4. 遵循项目的代码风格

## 支持

如有问题或建议，请通过以下方式联系：

- 提交 GitHub Issue
- 发送邮件至项目维护者
- 加入项目讨论群
```
```
