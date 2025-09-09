# 基础设施: UID 唯一ID生成

## 1. 设计理念

`uid` 是 `gochat` 项目中用于生成唯一标识符的统一组件。它提供了一个**统一的 Provider 接口**，封装了两种不同场景的 ID 生成方案：

- **Snowflake ID**: 提供**有状态的**、`int64` 类型的、趋势递增的分布式唯一 ID。适合用作数据库主键、消息 ID 等需要高性能和排序性的场景。
- **UUID v7**: 提供**无状态的**、`string` 类型的、符合 RFC 规范的时间有序的通用唯一标识符。适合用作对外暴露的资源 ID、请求 ID 等场景。

`uid` 组件遵循 `im-infra` 的核心规范，通过依赖注入获取 `coord.Provider` 来分配 Snowflake 所需的实例 ID，实现了完全的组件自治。

## 2. 核心 API 契约

### 2.1 构造函数

```go
// Config 是 uid 组件的配置结构体。
type Config struct {
	// ServiceName 服务名称，用于 Snowflake 实例ID分配的命名空间
	ServiceName string `json:"serviceName"`
	// MaxInstanceID Snowflake 支持的最大实例ID，默认为 1023 (2^10 - 1)
	MaxInstanceID int `json:"maxInstanceID"`
}

// GetDefaultConfig 返回默认的 uid 配置。
func GetDefaultConfig(env string) *Config

// Option 定义了用于定制 uid Provider 的函数。
type Option func(*options)

// WithLogger 将一个 clog.Logger 实例注入 uid，用于记录内部日志。
func WithLogger(logger clog.Logger) Option

// WithCoordProvider 注入 coord.Provider，用于分配 Snowflake 实例ID。
// 对于需要生成 Snowflake ID 的服务，此选项是必需的。
func WithCoordProvider(provider coord.Provider) Option

// New 创建一个新的 uid Provider 实例。
// 如果提供了 WithCoordProvider，此函数会在内部获取一个 Snowflake 实例ID并初始化生成器。
// 如果获取实例ID失败，此函数将返回错误。
func New(ctx context.Context, config *Config, opts ...Option) (Provider, error)
```

### 2.2 Provider 接口

```go
// Provider 定义了 uid 组件提供的所有能力。
type Provider interface {
	// GetUUIDV7 生成一个符合 RFC 规范的、时间有序的 UUID v7 字符串。
	GetUUIDV7() string

	// GenerateSnowflake 生成一个 int64 类型的、全局唯一的、趋势递增的雪花ID。
	// 可能会因为时钟回拨等问题返回错误。
	GenerateSnowflake() (int64, error)

	// IsValidUUID 检查一个字符串是否是合法的 UUID 格式。
	IsValidUUID(s string) bool

	// ParseSnowflake 解析一个雪花ID，返回其组成部分：时间戳、实例ID和序列号。
	ParseSnowflake(id int64) (timestamp, instanceID, sequence int64)

	// Close 关闭 Provider 并释放所有已分配的资源（如 Snowflake 实例ID）。
	Close() error
}
```

## 3. 标准用法

### 场景 1: 在服务启动时初始化 uid Provider

```go
// 在 main.go 中
func main() {
    // ... 首先初始化 clog 和 coord ...
    
    // 使用默认配置（推荐）
    config := uid.GetDefaultConfig("production") // 或 "development"
    
    // 根据实际服务覆盖配置
    config.ServiceName = "message-service"
    
    // 创建 uid Provider
    uidProvider, err := uid.New(context.Background(), config,
        uid.WithLogger(clog.Namespace("uid")),
        uid.WithCoordProvider(coordProvider), // 对于 Snowflake 是必需的依赖
    )
    if err != nil {
        clog.Fatal("初始化 uid 失败", clog.Err(err))
    }
    defer uidProvider.Close()
    
    clog.Info("uid Provider 初始化成功")
}
```

### 场景 2: 生成数据库主键 (使用 Snowflake)

```go
// 在服务的构造函数中注入 uidProvider
type MessageService struct {
    uid uid.Provider
    db  db.Provider
}

func NewMessageService(uidProvider uid.Provider, dbProvider db.Provider) *MessageService {
    return &MessageService{
        uid: uidProvider,
        db:  dbProvider,
    }
}

// 在业务逻辑中使用
func (s *MessageService) CreateMessage(ctx context.Context, content string) (*Message, error) {
    logger := clog.WithContext(ctx)
    
    // 直接调用方法生成唯一ID
    messageID, err := s.uid.GenerateSnowflake()
    if err != nil {
        return nil, fmt.Errorf("生成消息ID失败: %w", err)
    }
    
    msg := &Message{
        ID:      messageID,
        Content: content,
        Created: time.Now(),
    }
    
    // 保存到数据库
    if err := s.db.DB(ctx).Create(msg).Error; err != nil {
        return nil, fmt.Errorf("保存消息失败: %w", err)
    }
    
    logger.Info("消息创建成功", clog.Int64("message_id", messageID))
    return msg, nil
}
```

### 场景 3: 生成请求 ID (使用 UUID v7)

```go
// 在 Gin 中间件中
func RequestIDMiddleware(uidProvider uid.Provider) gin.HandlerFunc {
    return func(c *gin.Context) {
        // 检查请求头中是否已有 Request-ID
        requestID := c.GetHeader("X-Request-ID")
        
        if requestID == "" || !uidProvider.IsValidUUID(requestID) {
            // 生成新的请求ID
            requestID = uidProvider.GetUUIDV7()
        }
        
        // 设置到响应头和上下文中
        c.Header("X-Request-ID", requestID)
        
        // 注入到日志上下文（与 clog 集成）
        ctx := clog.WithTraceID(c.Request.Context(), requestID)
        c.Request = c.Request.WithContext(ctx)
        
        c.Next()
    }
}
```

## 4. 设计注记

### 4.1 GetDefaultConfig 默认值说明

`GetDefaultConfig` 对所有环境返回相同的默认配置：

```go
&Config{
    ServiceName:   "",     // 需要用户设置，通常为服务名
    MaxInstanceID: 1023,   // Snowflake 标准的最大实例ID (2^10 - 1)
}
```

用户必须设置 `ServiceName` 以确保不同服务之间的实例ID分配隔离。

### 4.2 实例ID分配机制

`uid` 组件通过 `coord.Provider` 的 `InstanceIDAllocator` 来管理 Snowflake 实例ID：

-   **分配时机**: 在服务启动调用 `uid.New` 时，组件会**一次性**获取一个可用的实例ID，并用它初始化内部的 Snowflake 生成器。
-   **释放时机**: 当服务关闭并调用 `Provider.Close()` 时，之前占用的实例ID会被自动释放回 `coord`。
-   **故障恢复**: 当服务异常崩溃时，之前占用的实例ID会通过 `coord` 的租约机制自动超时并回收。

这种设计将获取实例ID的重操作严格限制在服务初始化阶段，保证了运行时 `GenerateSnowflake` 的高性能。

### 4.3 线程安全性

-   **`GetUUIDV7`**: 完全无状态，天然线程安全。
-   **`GenerateSnowflake`**: 内部使用互斥锁保证对序列号的并发访问安全。
-   **`Provider` 所有方法**: 均为线程安全。