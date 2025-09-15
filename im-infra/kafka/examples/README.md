# Kafka 组件使用示例

这个目录包含了 GoChat Kafka 组件的完整使用示例。

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

使用项目脚本创建测试 Topics：

```bash
cd /Users/harrick/CodeField/gochat/deployment/scripts
./init-kafka-example.sh
```

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

### 自定义 Topic 配置
```go
// 创建自定义 topic
admin, err := kafka.NewAdminClient(ctx, config)
if err != nil {
    log.Fatal(err)
}
defer admin.Close()

err = admin.CreateTopic(ctx, "custom-topic", 6, 2)
if err != nil {
    log.Fatal(err)
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
```