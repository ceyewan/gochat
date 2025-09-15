# Kafka 组件使用示例

这个目录包含了 GoChat Kafka 组件的完整使用示例，包括传统 API 和新的 Provider 接口。

## 快速开始

### 1. 启动 Kafka 服务

确保 Kafka 服务正在运行：

```bash
# 如果使用 Docker
docker run -d --name kafka \
  -p 9092:9092 \
  -e KAFKA_BROKER_ID=1 \
  -e KAFKA_ZOOKEEPER_CONNECT=zookeeper:2181 \
  -e KAFKA_ADVERTISED_LISTENERS=PLAINTEXT://localhost:9092 \
  -e KAFKA_OFFSETS_TOPIC_REPLICATION_FACTOR=1 \
  confluentinc/cp-kafka:latest
```

### 2. 创建测试 Topics

**运行前必须手动创建测试 Topics**：

```bash
cd /Users/harrick/CodeField/gochat/deployment/scripts
./init-kafka-example.sh
```

如果脚本失败，可以使用 admin 脚本手动创建：

```bash
./kafka-admin.sh create example.user.events
./kafka-admin.sh create example.test-topic
./kafka-admin.sh create example.performance
./kafka-admin.sh create example.dead-letter
```

**注意**: 脚本默认使用 `localhost:9092,localhost:119092,localhost:29092` 作为 Kafka broker 地址。如果需要修改，请设置 `KAFKA_BROKER` 环境变量。

### 3. 运行示例程序

```bash
cd /Users/harrick/CodeField/gochat/im-infra/kafka/examples
go run main.go
```

## 示例程序功能

### 消息结构
```go
type UserEvent struct {
    UserID    string    `json:"user_id"`
    EventType string    `json:"event_type"`
    Timestamp time.Time `json:"timestamp"`
    Data      any       `json:"data,omitempty"`
}
```

### 发送的消息类型
1. **用户注册事件** (`registered`)
2. **用户更新事件** (`updated`)
3. **错误测试事件** (`error`) - 用于测试重试机制
4. **同步事件** (`sync-event`) - 演示同步发送

### 特性演示
- ✅ 异步消息发送
- ✅ 同步消息发送
- ✅ 自动 trace_id 注入
- ✅ 消息处理失败重试
- ✅ 优雅关闭
- ✅ 结构化日志
- ✅ 性能指标收集

## 预期输出

程序启动后，你会看到类似以下的日志：

```
[INFO] Kafka 生产者初始化成功
[INFO] Kafka 消费者初始化成功
[INFO] 开始订阅主题
[INFO] 收到消息 {topic: "example.user.events", key: "user001"}
[INFO] 处理用户事件 {user_id: "user001", event_type: "registered"}
[INFO] 事件处理成功
[INFO] 发送消息成功 {user_id: "user001", event_type: "registered"}
```

对于错误事件，你会看到重试日志：
```
[ERROR] 处理消息失败 {error: "模拟处理失败"}
[ERROR] 消费批次失败 {error: "模拟处理失败"}
```

## 故障排除

### Topic 不存在错误
如果看到 `UNKNOWN_TOPIC_OR_PARTITION` 错误：
1. 确保 Kafka 服务正在运行
2. 手动创建 Topics: `./init-kafka-example.sh`
3. 检查 Kafka 工具是否可用: `kafka-topics.sh --list`

### 连接失败
如果无法连接到 Kafka：
1. 检查 Kafka 服务是否在 `localhost:9092` 运行
2. 检查防火墙设置
3. 确认 Kafka 配置正确

### 命令行工具缺失
如果 `kafka-topics.sh` 命令不存在：
1. 安装 Kafka 或使用 Docker 容器
2. 或者手动创建 Topics（需要管理员权限）

## 高级用法

### 使用 kadm 库管理 Topics

如果需要在代码中管理 topics，推荐使用 franz-go 的 kadm 包：

```go
import "github.com/twmb/franz-go/pkg/kadm"

// 创建 kadm 客户端
kadmClient := kadm.NewClient(kgoClient)

// 创建 topic
responses, err := kadmClient.CreateTopics(ctx,
    kadm.NewTopicCreate("custom-topic").NumPartitions(6),
)
if err != nil {
    log.Fatal("创建 topic 失败:", err)
}
```

### 监控指标
```go
// 获取生产者指标
producerMetrics := producer.GetMetrics()
fmt.Printf("成功率: %.2f%%\n", producerMetrics["success_rate"])

// 获取消费者指标
consumerMetrics := consumer.GetMetrics()
fmt.Printf("处理成功率: %.2f%%\n", consumerMetrics["success_rate"])
```

### 健康检查
```go
// 检查生产器健康状态
if err := producer.Ping(ctx); err != nil {
    log.Printf("生产器不健康: %v", err)
}

// 检查消费者健康状态
if err := consumer.Ping(ctx); err != nil {
    log.Printf("消费者不健康: %v", err)
}

## 新的 Provider 接口示例

### 1. 统一 Provider 接口示例 (`provider/`)

展示了新的统一 Provider 接口的使用方法，包括：
- 使用 `NewProvider()` 创建统一入口
- 通过 `Provider.Producer()` 获取生产者
- 通过 `Provider.Consumer()` 获取消费者
- 通过 `Provider.Admin()` 进行管理操作
- Topic 的创建和管理
- 订单处理业务场景

**运行方式：**
```bash
cd provider
go run main.go
```

### 2. 指标和监控示例 (`metrics/`)

展示了 Kafka 组件的监控和指标收集功能，包括：
- 多消费者组的负载均衡
- 实时指标收集和报告
- 组件级别的性能监控
- 集群状态查看

**运行方式：**
```bash
cd metrics
go run main.go
```

## 新的 Provider 接口使用指南

### 基本用法

```go
// 创建 Provider
provider, err := kafka.NewProvider(ctx, config, kafka.WithLogger(logger))
defer provider.Close()

// 获取各种操作接口
producer := provider.Producer()
consumer := provider.Consumer("my-group")
admin := provider.Admin()
```

### 主要优势

1. **统一入口**：一个 Provider 实例管理所有 Kafka 操作
2. **资源管理**：自动管理消费者实例的生命周期
3. **向后兼容**：仍然支持 `NewProducer()` 和 `NewConsumer()`
4. **完整功能**：包含生产、消费、管理等所有操作

### 错误处理增强

```go
// 类型化的错误检查
if kafka.IsConfigError(err) {
    // 处理配置错误
} else if kafka.IsConnectionError(err) {
    // 处理连接错误
} else if kafka.IsProducerError(err) {
    // 处理生产者错误
} else if kafka.IsConsumerError(err) {
    // 处理消费者错误
} else if kafka.IsAdminError(err) {
    // 处理管理错误
}
```

### Admin 操作

```go
// 创建 Topic
err := admin.CreateTopic(ctx, "my-topic", 3, 1, map[string]string{
    "retention.ms": "604800000",
})

// 列出 Topics
topics, err := admin.ListTopics(ctx)

// 获取 Topic 元数据
metadata, err := admin.GetTopicMetadata(ctx, "my-topic")

// 删除 Topic
err := admin.DeleteTopic(ctx, "my-topic")
```

## 迁移指南

### 从传统 API 迁移到 Provider 接口

**传统方式：**
```go
producer, err := kafka.NewProducer(ctx, config)
consumer, err := kafka.NewConsumer(ctx, config, "my-group")
```

**新的 Provider 方式：**
```go
provider, err := kafka.NewProvider(ctx, config)
producer := provider.Producer()
consumer := provider.Consumer("my-group")
defer provider.Close()  // 统一管理所有资源
```

### 主要改进

1. **资源管理**：Provider 统一管理所有资源，避免资源泄漏
2. **消费者缓存**：相同 groupID 的消费者实例会被复用
3. **Admin 功能**：内置完整的 Topic 管理功能
4. **错误处理**：提供更细粒度的错误类型判断
5. **健康检查**：统一的 Ping 接口检查组件状态