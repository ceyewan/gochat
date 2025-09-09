# 基础设施: Once 分布式幂等操作

## 1. 设计理念

`once` 组件提供了一个**统一的分布式幂等接口**，封装了两种不同场景的幂等实现：

- **分布式幂等**: 基于 Redis 的分布式幂等，适用于多实例部署的微服务环境，确保集群级别的操作幂等性。
- **单机幂等**: 基于内存的单机幂等，适用于单实例部署或不需要跨实例协调的场景，性能更高。

`once` 组件遵循 `im-infra` 的核心规范：

- **统一接口**: 通过 Provider 模式提供一致的幂等体验，业务代码无需关心底层实现
- **结果缓存**: 对于需要返回结果的操作，提供原子性的"防重+结果缓存"功能
- **失败可重试**: 操作失败时自动清除幂等标记，允许后续重试
- **智能模式选择**: 根据配置自动选择最适合的幂等实现

## 2. 核心 API 契约

### 2.1 构造函数

```go
// Config 是 once 组件的配置结构体。
type Config struct {
	// Mode 幂等模式，支持 "distributed" 和 "local" 两种模式
	Mode string `json:"mode"`
	
	// ServiceName 用于日志记录和监控，以区分是哪个服务在使用幂等器
	ServiceName string `json:"serviceName"`
	
	// KeyPrefix 为所有幂等 key 自动添加前缀，用于命名空间隔离
	KeyPrefix string `json:"keyPrefix"`
	
}

// GetDefaultConfig 返回默认的 once 配置。
// 开发环境：使用单机模式，无 Redis 依赖。
// 生产环境：使用分布式模式，依赖 cache.Provider。
func GetDefaultConfig(env string) *Config

// Option 定义了用于定制 once Provider 的函数。
type Option func(*options)

// WithLogger 将一个 clog.Logger 实例注入 once，用于记录内部日志。
func WithLogger(logger clog.Logger) Option

// WithCacheProvider 注入 cache.Provider，用于分布式模式的 Redis 操作。
// 如果 config.Mode 为 "distributed"，此选项是必需的。
func WithCacheProvider(provider cache.Provider) Option

// New 创建一个新的 once Provider 实例。
// 这是与 once 组件交互的唯一入口。
func New(ctx context.Context, config *Config, opts ...Option) (Provider, error)
```

### 2.2 Provider 接口

```go
// Provider 定义了 once 组件提供的所有能力。
type Provider interface {
	// Do 执行一个幂等操作，无返回值。
	// 如果 key 对应的操作已经成功执行过，则直接返回 nil。
	// 否则，执行函数 f。如果 f 返回错误，幂等标记不会被持久化，允许重试。
	Do(ctx context.Context, key string, ttl time.Duration, f func() error) error

	// Execute 执行一个带返回值的幂等操作。
	// 如果操作已执行过，它会直接返回缓存的结果。
	// 否则，执行 callback，缓存其结果，并返回。
	Execute(ctx context.Context, key string, ttl time.Duration, callback func() (any, error)) (any, error)

	// Clear 主动清除指定 key 的幂等标记和缓存结果。
	// 适用于需要手动重置幂等状态的场景。
	Clear(ctx context.Context, key string) error

	// Close 关闭 Provider 并释放相关资源。
	Close() error
}
```

## 3. 标准用法

### 场景 1: 在服务启动时初始化 once Provider

```go
// 在 main.go 中
func main() {
    // ... 首先初始化 clog 和 cache ...
    
    // 1. 获取并覆盖配置
    config := once.GetDefaultConfig("production") // 或 "development"
    config.ServiceName = "message-service"
    config.KeyPrefix = "idempotent:"
    
    // 2. 准备 Options
    // New 函数内部会根据 config.Mode 决定是否使用 cacheProvider
    opts := []once.Option{
        once.WithLogger(clog.Namespace("once")),
        once.WithCacheProvider(cacheProvider), // 分布式模式依赖 cache 组件
    }

    // 3. 创建 Provider
    // 初始化逻辑被封装在 New 函数中，调用者无需关心具体模式
    onceProvider, err := once.New(context.Background(), config, opts...)
    if err != nil {
        clog.Fatal("初始化 once 失败", clog.Err(err))
    }
    defer onceProvider.Close()
    
    clog.Info("once Provider 初始化成功", clog.String("mode", config.Mode))
}
```

### 场景 2: 保证消息队列消费者幂等性 (使用 `Do`)

```go
import "github.com/ceyewan/gochat/im-infra/once"

// 在服务的构造函数中注入 onceProvider
type PaymentService struct {
    once once.Provider
    db   db.Provider
}

func NewPaymentService(onceProvider once.Provider, dbProvider db.Provider) *PaymentService {
    return &PaymentService{
        once: onceProvider,
        db:   dbProvider,
    }
}

// Kafka 消费者逻辑
func (s *PaymentService) HandlePaymentMessage(ctx context.Context, msg *mq.Message) error {
    logger := clog.WithContext(ctx)
    
    // 使用消息的唯一ID作为幂等键
    messageID := string(msg.Key)
    key := "payment:process:" + messageID
    
    // 使用 once.Do 保证业务逻辑只执行一次
    err := s.once.Do(ctx, key, 24*time.Hour, func() error {
        // 核心业务逻辑：处理支付
        var paymentData Payment
        if err := json.Unmarshal(msg.Value, &paymentData); err != nil {
            return fmt.Errorf("解析支付数据失败: %w", err)
        }
        
        logger.Info("开始处理支付", 
            clog.String("payment_id", paymentData.ID),
            clog.String("message_id", messageID))
        
        return s.processPayment(ctx, paymentData)
    })

    if err != nil {
        // 记录错误，但不 ack 消息，以便 Kafka 重试
        logger.Error("处理支付消息失败", 
            clog.Err(err), 
            clog.String("message_id", messageID))
        return err
    }
    
    // 无论是否首次执行，都安全地 ack 消息
    logger.Info("支付消息处理完成", clog.String("message_id", messageID))
    return nil
}

func (s *PaymentService) processPayment(ctx context.Context, payment Payment) error {
    // 具体的支付处理逻辑
    return s.db.DB(ctx).Create(&payment).Error
}
```

### 场景 3: 防止 API 重复创建资源 (使用 `Execute`)

```go
// 在 HTTP Handler 中
func (s *DocumentService) CreateDocument(c *gin.Context) {
    idempotencyKey := c.GetHeader("X-Idempotency-Key")
    if idempotencyKey == "" {
        c.JSON(http.StatusBadRequest, gin.H{"error": "idempotency key required"})
        return
    }

    logger := clog.WithContext(c.Request.Context())
    key := "doc:create:" + idempotencyKey

    // 使用 once.Execute 来创建资源并缓存结果
    result, err := s.once.Execute(c.Request.Context(), key, 48*time.Hour, func() (any, error) {
        logger.Info("开始创建文档", clog.String("idempotency_key", idempotencyKey))
        
        // 核心业务逻辑：创建文档并返回其完整信息
        var reqData CreateDocumentRequest
        if err := c.ShouldBindJSON(&reqData); err != nil {
            return nil, fmt.Errorf("请求参数错误: %w", err)
        }
        
        doc, err := s.createDocument(c.Request.Context(), reqData)
        if err != nil {
            return nil, fmt.Errorf("创建文档失败: %w", err)
        }
        
        logger.Info("文档创建成功", 
            clog.String("document_id", doc.ID),
            clog.String("idempotency_key", idempotencyKey))
        
        return doc, nil
    })

    if err != nil {
        logger.Error("文档创建失败", 
            clog.Err(err), 
            clog.String("idempotency_key", idempotencyKey))
        c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
        return
    }

    // 无论是否首次执行，都能拿到正确的文档信息
    doc := result.(*Document)
    logger.Info("返回文档信息", 
        clog.String("document_id", doc.ID),
        clog.Bool("from_cache", result != nil))
    
    c.JSON(http.StatusOK, gin.H{"document": doc})
}

func (s *DocumentService) createDocument(ctx context.Context, req CreateDocumentRequest) (*Document, error) {
    doc := &Document{
        ID:      s.uid.GetUUIDV7(), // 使用 UID 组件生成ID
        Title:   req.Title,
        Content: req.Content,
        Created: time.Now(),
    }
    
    err := s.db.DB(ctx).Create(doc).Error
    return doc, err
}
```

### 场景 4: 幂等状态管理

```go
// 管理接口：清除幂等状态（用于数据订正或手动重试）
func (s *AdminService) ClearIdempotentState(ctx context.Context, key string) error {
    logger := clog.WithContext(ctx)
    
    // 直接调用 Clear，无需预先检查
    // 如果 key 不存在，Clear 操作通常是无害的
    if err := s.once.Clear(ctx, key); err != nil {
        logger.Error("清除幂等状态失败", clog.Err(err), clog.String("key", key))
        return fmt.Errorf("清除幂等状态失败: %w", err)
    }
    
    logger.Info("幂等状态清除成功", clog.String("key", key))
    return nil
}
```

## 4. 设计注记

### 4.1 GetDefaultConfig 默认值说明

`GetDefaultConfig` 根据环境返回优化的默认配置：

**开发环境 (development)**:
```go
&Config{
    Mode:        "local",              // 使用单机模式，无 Redis 依赖
    ServiceName: "",                   // 需要用户设置
    KeyPrefix:   "idempotent:",        // 默认前缀
}
```

**生产环境 (production)**:
```go
&Config{
    Mode:        "distributed",       // 使用分布式模式，依赖 cache.Provider
    ServiceName: "",                  // 需要用户设置
    KeyPrefix:   "idempotent:",       // 默认前缀
}
```

用户仍需要根据实际部署环境覆盖 `ServiceName` 等关键配置。

### 4.2 分布式 vs 单机模式选择

**分布式模式 (distributed)**:
- **优势**: 集群级别的幂等保证，多实例间状态一致
- **适用场景**: 微服务集群、水平扩展的应用、消息队列消费者
- **依赖**: 需要 Redis 和 cache.Provider
- **性能**: 略低（网络 I/O 开销）

**单机模式 (local)**:
- **优势**: 极高性能，无网络依赖，简单可靠
- **适用场景**: 单实例部署、内存密集型操作、开发测试环境
- **依赖**: 无额外依赖
- **限制**: 无法跨实例协调

### 4.3 幂等键设计最佳实践

**命名规范**: 建议使用分层命名，如 `{业务域}:{操作类型}:{业务ID}`
- `payment:process:order-123`
- `user:register:email-abc@example.com`
- `doc:create:idempotency-xyz`

**TTL 设置**: 根据业务特点合理设置过期时间
- 短期操作（API 调用）: 1-24 小时
- 长期操作（批处理）: 7-30 天
- 关键操作（支付）: 永久保存或长期保存

**前缀隔离**: 通过 `KeyPrefix` 实现不同服务、环境的命名空间隔离

### 4.4 错误处理和重试机制

**业务逻辑失败**: 如果 `Do` 或 `Execute` 中的回调函数返回错误，幂等标记不会被设置，允许后续重试

**幂等器异常**: 如果幂等器本身异常（如 Redis 连接失败），建议的降级策略：
- 关键操作：抛出错误，阻止执行
- 非关键操作：记录日志，继续执行

**数据一致性**: 在分布式模式下，使用 Lua 脚本确保 Redis 操作的原子性
