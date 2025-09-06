# UID 组件设计文档

## 概述

UID 组件为 GoChat 系统提供分布式唯一 ID 生成功能。它实现了两种主要的 ID 生成算法：

1. **雪花算法 (Snowflake)**: Twitter 的分布式 ID 生成算法
2. **UUID v4/v7**: RFC 4122 标准的通用唯一标识符

## 架构设计

### 组件结构

```
im-infra/uid/
├── uid.go              # 公共 API 和配置
├── internal/
│   ├── client.go       # 核心实现
│   └── errors.go       # 错误定义
├── examples/
│   ├── basic/         # 基本使用示例
│   └── advanced/      # 高级使用模式
├── uid_test.go        # 测试套件
├── README.md          # 用户文档（中文）
└── DESIGN.md          # 设计文档（中文）
```

### 设计原则

遵循 GoChat 基础设施组件的设计模式：

- **接口驱动设计**: 公共 API 与实现分离
- **配置驱动**: 支持 YAML/JSON 配置并带验证
- **函数式选项**: 使用选项模式进行灵活初始化
- **结构化日志**: 集成 GoChat 日志基础设施
- **全面测试**: 单元测试、集成测试和基准测试

## 雪花算法实现

### 算法详情

雪花算法生成 64 位唯一 ID，位分配如下：

```
0                   1                   2                   3
0 1 2 3 4 5 6 7 8 9 0 1 2 3 4 5 6 7 8 9 0 1 2 3 4 5 6 7 8 9 0 1
+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+
|0|                    41-bit timestamp                          |
+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+
|                timestamp                |  10-bit datacenter   |
+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+
|  datacenter   |    10-bit worker ID    |   12-bit sequence    |
+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+
```

**位分布：**
- **符号位**: 1 位（始终为 0）
- **时间戳**: 41 位（相对于自定义起始时间的毫秒数）
- **数据中心 ID**: 5 位 (0-31)
- **Worker ID**: 5 位 (0-31)
- **序列号**: 12 位 (0-4095)

### 自定义起始时间

使用 Twitter 的自定义起始时间（2010年11月4日 01:42:54 UTC）以保证兼容性：

```go
const twepoch = int64(1288834974657) // Twitter epoch
```

### 时钟同步处理

实现对时钟漂移场景的处理：

```go
if timestamp < c.lastTimestamp {
    // 时钟向后移动
    c.logger.Error("时钟向后移动，等待直到",
        clog.Int64("lastTimestamp", c.lastTimestamp),
        clog.Int64("currentTimestamp", timestamp))
    time.Sleep(time.Duration(c.lastTimestamp-timestamp) * time.Millisecond)
    timestamp = c.currentTimestamp() - twepoch
}
```

### 序列号管理

序列号用于管理同一毫秒内的并发 ID 生成：

```go
if c.lastTimestamp == timestamp {
    c.sequence = (c.sequence + 1) & maxSequence
    if c.sequence == 0 {
        // 序列号溢出，等待下一毫秒
        timestamp = c.waitNextMillis(timestamp)
    }
} else {
    c.sequence = 0
}
```

## UUID 实现增强

### UUID v4 生成

使用标准 `github.com/google/uuid` 库生成符合 RFC 4122 的 UUID v4：

```go
func (c *Client) generateUUIDV4() string {
    return uuid.NewString()
}
```

### UUID v7 生成

支持 UUID v7（时间排序 UUID）：

```go
func (c *Client) generateUUIDV7() string {
    id := uuid.NewV7()
    return id.String()
}
```

### UUID 验证

提供 UUID 格式验证功能：

```go
func (c *Client) validateUUID(uuidStr string) bool {
    _, err := uuid.Parse(uuidStr)
    return err == nil
}
```

## 线程安全

### 同步策略

实现使用互斥锁确保线程安全的 ID 生成：

```go
type Client struct {
    mu sync.Mutex
    // ... 其他字段
}

func (c *Client) GenerateInt64() int64 {
    c.mu.Lock()
    defer c.mu.Unlock()
    // ... ID 生成逻辑
}
```

### 性能考虑

- **锁竞争**: 最小化 ID 生成的临界区
- **序列优化**: 同一毫秒内快速序列递增
- **内存分配**: 雪花算法整数零分配

## 配置设计

### 配置结构

```go
type Config struct {
    WorkerID     int64 `json:"workerID" yaml:"workerID"`
    DatacenterID int64 `json:"datacenterID" yaml:"datacenterID"`
    EnableUUID   bool  `json:"enableUUID" yaml:"enableUUID"`
}
```

### 验证规则

```go
func (c *Config) Validate() error {
    if c.WorkerID < 0 || c.WorkerID > 31 {
        return fmt.Errorf("workerID 必须在 0 和 31 之间")
    }
    if c.DatacenterID < 0 || c.DatacenterID > 31 {
        return fmt.Errorf("datacenterID 必须在 0 和 31 之间")
    }
    return nil
}
```

### 默认配置

```go
func DefaultConfig() Config {
    return Config{
        WorkerID:     1,
        DatacenterID: 1,
        EnableUUID:   true,
    }
}
```

## 错误处理

### 错误类型

```go
var (
    ErrInvalidWorkerID     = errors.New("无效的 worker ID")
    ErrInvalidDatacenterID = errors.New("无效的 datacenter ID")
    ErrClockBackwards      = errors.New("时钟向后移动")
)
```

### 错误传播

错误包装上下文以便更好地调试：

```go
if err != nil {
    return nil, fmt.Errorf("创建 uid 客户端失败: %w", err)
}
```

## 日志集成

### 结构化日志

与 GoChat 的日志基础设施集成：

```go
logger.Info("uid 组件初始化成功",
    clog.Int64("workerID", cfg.WorkerID),
    clog.Int64("datacenterID", cfg.DatacenterID),
    clog.Bool("enableUUID", cfg.EnableUUID),
)
```

### 日志级别

- **Info**: 组件初始化、配置
- **Warning**: 时钟同步问题
- **Error**: 配置错误、运行时故障

## 性能特性

### 基准测试结果

```
BenchmarkGenerateInt64-8      30000000    45.2 ns/op    0 B/op    0 allocs/op
BenchmarkGenerateString-8     2000000    742 ns/op    48 B/op    1 allocs/op
```

### 可扩展性

- **吞吐量**: ~22M IDs/秒（雪花算法整数）
- **并发性**: 随 goroutine 线性扩展
- **内存**: 最小内存占用

### 性能对比

| 算法 | 延迟 | 吞吐量 | 分配 |
|------|------|--------|------|
| 雪花算法 | ~45ns | ~22M ops/s | 0 allocs |
| UUID v4 | ~742ns | ~1.3M ops/s | 1 alloc |

## 测试策略

### 单元测试

- 配置验证
- ID 唯一性验证
- 线程安全测试
- 错误条件处理

### 集成测试

- 并发生成压力测试
- 时钟漂移模拟
- 多 worker 配置测试

### 性能测试

- 吞吐量基准测试
- 延迟测量
- 可扩展性测试

## 部署考虑

### Worker ID 管理

确保所有实例使用唯一的 worker ID：

- **静态分配**: 预配置的 worker ID
- **动态发现**: 服务发现集成
- **协调服务**: 基于 etcd 的 worker ID 分配

### 时钟同步

保持所有实例的时钟同步：

- **NTP 配置**: 可靠的时间源
- **监控**: 时钟漂移检测
- **故障转移**: 优雅处理时间同步问题

### 监控

需要监控的关键指标：

- **ID 生成率**: 每秒请求数
- **延迟**: P99、P95、P50 响应时间
- **错误率**: 配置和运行时错误
- **时钟漂移**: 时间同步状态

## 安全考虑

### 信息泄露

雪花算法 ID 暴露时间信息：

- **时间戳暴露**: 可以推断生成时间
- **Worker ID 暴露**: 可能的实例识别
- **速率限制**: ID 生成模式可能暴露流量

### 缓解策略

- **敏感数据使用 UUID**: 面向用户的 ID 使用 UUID
- **内部 vs 外部**: 不同上下文使用不同的 ID 类型
- **速率限制**: 实现 ID 生成速率限制

## 未来增强

### 计划功能

1. **雪花算法变体**: 支持不同的位分配
2. **UUID 格式**: 额外的 UUID 版本（v1, v6, v7）
3. **批量生成**: 批量 ID 生成以提高效率
4. **持久化**: 可选的 ID 生成审计日志

### 性能优化

1. **无锁算法**: 研究无锁 ID 生成
2. **内存池**: 减少 UUID 的分配开销
3. **硬件加速**: CPU 特定优化

## 结论

UID 组件为 GoChat 系统提供了健壮、高性能的分布式唯一 ID 生成解决方案。其双算法方式为不同用例提供了灵活性，同时保持了基础设施组件期望的一致性和可靠性。

该设计遵循既定的 GoChat 模式，并无缝集成到现有生态系统中，为分布式系统识别需求提供了坚实的基础。