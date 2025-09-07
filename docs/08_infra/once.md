# 基础设施: once 幂等操作

## 1. 设计理念

`once` 组件是一个基于 Redis 的、高性能的分布式幂等库。它旨在为分布式系统中的各种操作提供“执行一次”的语义保证，有效防止因网络重试、消息重复等原因导致的操作重复执行。

遵循 **KISS (Keep It Simple, Stupid)** 原则，`once` 组件提供了一个极简但功能强大的 API，专注于解决幂等性这一个核心问题。

- **安全可靠的抽象**: 组件的核心是 `Do` 和 `Execute` 方法，它们封装了所有必要的 Redis 操作（如 `SETNX`, `GET`, `DEL`），并使用 Lua 脚本保证了复杂操作的原子性。这可以防止使用者写出有竞态条件的错误代码。
- **结果缓存**: 对于需要返回结果的操作，`Execute` 方法原子性地将“防重”和“结果缓存”结合起来，避免了业务代码需要进行“先检查、再获取”的额外数据库或服务调用，提升了性能和代码简洁性。
- **失败可重试**: 如果被包裹的业务逻辑执行失败，组件会自动清除幂等标记，允许后续的请求进行重试，保证了系统的健壮性。
- **易用性与灵活性兼顾**:
    - 提供全局的 `once.Do()` 和 `once.Execute()` 方法，内置一个默认的 Redis 客户端，使得在简单场景下的使用成本极低。
    - 同时提供 `once.New()` 构造函数，允许上层服务在需要时注入自定义的配置和依赖（如不同的 Redis 实例），与其他 `im-infra` 组件保持一致的设计模式。

## 2. 核心 API 契约

`once` 组件的公开 API 被精简为以下核心部分：

### 2.1 构造函数

```go
// Config 是 once 组件的配置结构体。
type Config struct {
    // CacheProvider 用于指定要连接的 Redis 实例。
    // 通常，这个 Provider 是从 cache.New() 创建的。
    CacheProvider cache.Provider
}

// New 创建一个新的、可定制的 Idempotent 客户端实例。
func New(ctx context.Context, config *Config, opts ...Option) (Idempotent, error)
```

### 2.2 Idempotent 接口

```go
// Idempotent 定义了幂等操作的核心接口。
type Idempotent interface {
    // Do 是核心的幂等操作方法。
    // 如果 key 对应的操作已经成功执行过，则直接返回 nil。
    // 否则，执行函数 f。如果 f 返回错误，Do 将返回该错误，并且幂等标记不会被持久化，允许重试。
    Do(ctx context.Context, key string, ttl time.Duration, f func() error) error

    // Execute 执行一个带返回值的幂等操作。
    // 如果操作已执行过，它会直接返回缓存的结果。
    // 否则，执行 callback，缓存其结果，并返回。
    // 注意：此方法仅适用于结果可以被安全地序列化和缓存的场景。
    Execute(ctx context.Context, key string, ttl time.Duration, callback func() (any, error)) (any, error)
}
```

### 2.3 全局方法 (推荐使用)

为了最大化易用性，`once` 包提供了直接可用的全局方法，它们在内部使用一个默认的 `cache.Provider`。

```go
// Do 使用全局默认客户端执行一个幂等操作。
// key: 全局唯一的幂等键，如 "payment:order-123"。
// ttl: 幂等键的有效期。
// f: 只有在首次执行时才会被调用的业务逻辑函数。
func Do(ctx context.Context, key string, ttl time.Duration, f func() error) error

// Execute 使用全局默认客户端执行一个带返回值的幂等操作。
// callback: 首次执行时调用的业务逻辑，其返回值会被缓存。
func Execute(ctx context.Context, key string, ttl time.Duration, callback func() (any, error)) (any, error)
```

## 3. 标准用法

### 场景 1: 保证消息队列消费者幂等性 (使用 `Do`)

```go
import "github.com/ceyewan/gochat/im-infra/once"

// Kafka 消费者逻辑
func (c *Consumer) HandlePaymentMessage(ctx context.Context, msg *mq.Message) error {
    // 使用消息的唯一ID（或业务ID）作为幂等键
    messageID := string(msg.Key)
    
    // 使用 once.Do 保证业务逻辑只执行一次
    err := once.Do(ctx, "payment:process:"+messageID, 24*time.Hour, func() error {
        // 核心业务逻辑：处理支付
        var paymentData Payment
        if err := json.Unmarshal(msg.Value, &paymentData); err != nil {
            return err
        }
        return c.paymentService.Process(ctx, paymentData)
    })

    if err != nil {
        // 记录错误，但不 ack 消息，以便 Kafka 重试
        clog.C(ctx).Error("处理支付消息失败", clog.Err(err), clog.String("messageID", messageID))
        return err
    }
    
    // 无论是否首次执行，都安全地 ack 消息
    return c.kafkaReader.CommitMessages(ctx, *msg)
}
```

### 场景 2: 防止 API 重复创建资源 (使用 `Execute`)

对于“创建并返回”类型的 API，`Execute` 可以原子性地完成防重和结果缓存。

```go
import "github.com/ceyewan/gochat/im-infra/once"

// 在 HTTP Handler 中
func (s *Server) CreateDocument(c *gin.Context) {
    idempotencyKey := c.GetHeader("X-Idempotency-Key")
    if idempotencyKey == "" {
        c.JSON(http.StatusBadRequest, gin.H{"error": "idempotency key required"})
        return
    }

    // 使用 once.Execute 来创建资源并缓存结果（新文档的ID）
    result, err := once.Execute(c.Request.Context(), "doc:create:"+idempotencyKey, 48*time.Hour, func() (any, error) {
        // 核心业务逻辑：创建文档并返回其ID
        return s.docService.Create(c.Request.Context(), c.Request.Body)
    })

    if err != nil {
        c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
        return
    }

    // 无论是否首次执行，都能拿到正确的文档ID
    docID := result.(string)
    c.JSON(http.StatusOK, gin.H{"document_id": docID})
}