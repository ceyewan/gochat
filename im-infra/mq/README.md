# MQ - 高性能 Kafka 消息队列库

[![Go Version](https://img.shields.io/badge/Go-1.19+-00ADD8?style=flat&logo=go)](https://golang.org/)
[![Kafka Version](https://img.shields.io/badge/Kafka-2.8+-231F20?style=flat&logo=apache-kafka)](https://kafka.apache.org/)
[![License](https://img.shields.io/badge/License-MIT-blue.svg)](LICENSE)

一个基于 [franz-go](https://github.com/twmb/franz-go) 的高性能 Kafka 消息队列基础库，专为即时通讯场景优化。

## ✨ 核心特性

- 🚀 **超高性能**: 微秒级延迟，支持 100,000+ 消息/秒吞吐量
- 🔒 **幂等性保证**: 内置幂等性支持，确保消息不重复
- 📦 **智能批处理**: 自适应批处理系统，优化小消息处理性能
- 🗜️ **多种压缩**: 支持 LZ4、Snappy、Gzip、Zstd 压缩算法
- 🔄 **连接池管理**: 高效的连接复用和健康检查机制
- 📊 **全面监控**: 内置性能指标收集和健康检查
- 🛡️ **错误处理**: 完善的错误分类和优雅降级策略
- 🌐 **易于使用**: 简洁的 API 设计，支持全局方法和实例方法

## 🚀 快速开始

### 安装

```bash
go get github.com/ceyewan/gochat/im-infra/mq
```

### 基本用法

#### 发送消息

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
    
    log.Println("消息发送成功!")
}
```

#### 接收消息

```go
package main

import (
    "context"
    "log"
    "github.com/ceyewan/gochat/im-infra/mq"
)

func main() {
    ctx := context.Background()
    
    // 订阅消息
    callback := func(message *mq.Message, partition mq.TopicPartition, err error) bool {
        if err != nil {
            log.Printf("消费错误: %v", err)
            return false
        }
        
        log.Printf("收到消息: %s", string(message.Value))
        return true // 继续消费
    }
    
    err := mq.Subscribe(ctx, []string{"chat-messages"}, callback)
    if err != nil {
        log.Fatal(err)
    }
}
```

### 自定义配置

```go
package main

import (
    "time"
    "github.com/ceyewan/gochat/im-infra/mq"
)

func main() {
    // 创建自定义配置
    cfg := mq.Config{
        Brokers:  []string{"localhost:9092"},
        ClientID: "my-chat-app",
        ProducerConfig: mq.ProducerConfig{
            Compression:       "lz4",   // 低延迟压缩
            BatchSize:         16384,   // 16KB 批次
            LingerMs:          5,       // 5毫秒等待
            EnableIdempotence: true,    // 启用幂等性
        },
        ConsumerConfig: mq.ConsumerConfig{
            GroupID:            "my-consumer-group",
            AutoOffsetReset:    "latest",
            EnableAutoCommit:   true,
            AutoCommitInterval: 5 * time.Second,
        },
    }
    
    // 创建 MQ 实例
    mqInstance, err := mq.New(cfg)
    if err != nil {
        log.Fatal(err)
    }
    defer mqInstance.Close()
    
    // 使用实例...
}
```

## 📖 完整示例

查看 [examples/chat_example.go](examples/chat_example.go) 了解完整的聊天应用示例，包括：

- 生产者和消费者的完整实现
- 批量消息处理
- 异步消息发送
- 性能监控
- 优雅关闭

运行示例：

```bash
cd examples
go run chat_example.go
```

## 🔧 配置选项

### 生产者配置

| 参数 | 默认值 | 说明 |
|------|--------|------|
| `Compression` | `"lz4"` | 压缩算法：none, gzip, snappy, lz4, zstd |
| `BatchSize` | `16384` | 批次大小（字节） |
| `LingerMs` | `5` | 批次等待时间（毫秒） |
| `EnableIdempotence` | `true` | 是否启用幂等性 |
| `RequiredAcks` | `1` | 确认级别：0, 1, -1 |

### 消费者配置

| 参数 | 默认值 | 说明 |
|------|--------|------|
| `GroupID` | `""` | 消费者组ID（必须设置） |
| `AutoOffsetReset` | `"latest"` | 偏移量重置策略：earliest, latest, none |
| `EnableAutoCommit` | `true` | 是否启用自动提交 |
| `MaxPollRecords` | `500` | 单次拉取最大记录数 |
| `SessionTimeout` | `10s` | 会话超时时间 |

### 性能配置

| 参数 | 默认值 | 说明 |
|------|--------|------|
| `TargetLatencyMicros` | `1000` | 目标延迟（微秒） |
| `TargetThroughputPerSec` | `100000` | 目标吞吐量（消息/秒） |
| `OptimizeForSmallMessages` | `true` | 是否优化小消息处理 |

## 📊 性能基准

在标准测试环境下的性能表现：

| 指标 | 性能 |
|------|------|
| **延迟** | < 1ms (P99) |
| **吞吐量** | 100,000+ 消息/秒 |
| **小消息优化** | < 1KB 消息优化处理 |
| **压缩比** | LZ4: ~60%, Snappy: ~65% |
| **内存使用** | < 500MB (正常负载) |

## 🛠️ API 文档

详细的 API 文档请参考 [API.md](API.md)，包括：

- 完整的 API 参考
- 使用示例
- 最佳实践指南
- 性能调优建议
- 故障排除指南

## 🧪 测试

运行单元测试：

```bash
go test ./...
```

运行基准测试：

```bash
go test -bench=. -benchmem
```

运行性能测试（需要 Kafka 环境）：

```bash
go test -v -run=TestConcurrent
go test -v -run=BenchmarkProducerLatency
```

## 📈 监控

### 获取性能指标

```go
// 获取生产者指标
metrics := mqInstance.Producer().GetMetrics()
fmt.Printf("延迟: %v, 吞吐量: %.2f 消息/秒\n", 
    metrics.AverageLatency, metrics.MessagesPerSecond)

// 获取消费者指标
consumerMetrics := mqInstance.Consumer().GetMetrics()
fmt.Printf("消费延迟: %d 条消息\n", consumerMetrics.Lag)

// 获取连接池统计
poolStats := mqInstance.ConnectionPool().GetStats()
fmt.Printf("连接池: %d/%d 活跃连接\n", 
    poolStats.ActiveConnections, poolStats.MaxConnections)
```

### 健康检查

```go
ctx := context.Background()
err := mq.Ping(ctx)
if err != nil {
    log.Printf("健康检查失败: %v", err)
} else {
    log.Println("系统健康")
}
```

## 🔍 故障排除

### 常见问题

1. **连接失败**
   ```
   错误: CONNECTION_FAILED: dial tcp: connection refused
   ```
   - 检查 Kafka broker 地址和端口
   - 确认 Kafka 服务正在运行
   - 检查网络连接和防火墙设置

2. **延迟过高**
   ```
   平均延迟: 50ms (期望: <1ms)
   ```
   - 设置 `LingerMs = 0` 立即发送
   - 减小 `BatchSize`
   - 使用 `Compression = "none"`

3. **吞吐量不足**
   ```
   当前: 10,000 消息/秒 (期望: 100,000+)
   ```
   - 增大 `BatchSize` 和 `MaxPollRecords`
   - 增加消费者实例数量
   - 优化消息处理逻辑

更多故障排除信息请参考 [API.md](API.md#故障排除指南)。

## 🤝 贡献

欢迎提交 Issue 和 Pull Request！

在提交代码前，请确保：

1. 代码通过所有测试
2. 添加了适当的单元测试
3. 更新了相关文档
4. 遵循项目的代码风格

## 📄 许可证

本项目采用 [MIT 许可证](LICENSE)。

## 🙏 致谢

- [franz-go](https://github.com/twmb/franz-go) - 优秀的 Kafka Go 客户端
- [Apache Kafka](https://kafka.apache.org/) - 强大的分布式流处理平台

## 📞 支持

如有问题或建议，请：

- 提交 [GitHub Issue](https://github.com/ceyewan/gochat/issues)
- 查看 [API 文档](API.md)
- 参考 [使用示例](examples/)

---

**注意**: 本库专为即时通讯场景优化，在其他场景使用前请评估性能特性是否符合需求。
