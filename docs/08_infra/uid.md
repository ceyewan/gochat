# 基础设施: UID 唯一ID生成

## 1. 设计理念

`uid` 是 `gochat` 项目中用于生成唯一标识符的组件集合。它遵循**单一职责原则**，将两种完全不同的ID生成方案解耦到独立的子包中，以提供清晰、安全且易于使用的 API。

- **`uid/snowflake`**: 提供**有状态的**、`int64` 类型的、趋势递增的分布式唯一 ID。它必须依赖于 `coord` 组件分配的唯一实例 ID (`instanceID`)，非常适合用作数据库主键、消息 ID 等需要高性能和排序性的场景。
- **`uid/uuid`**: 提供**无状态的**、`string` 类型的、符合 RFC 规范的通用唯一标识符。它不依赖任何外部服务，非常适合用作对外暴露的资源 ID、请求 ID 等需要全局唯一且不希望暴露内部信息的场景。

这种设计避免了将有状态和无状态的逻辑混淆在一起，为开发者提供了明确的选择。

## 2. 核心 API 契约

### 2.1 `snowflake` 包

`snowflake` 包用于生成雪花 ID，其使用是**有状态的**。

#### 构造函数

```go
package snowflake

// New 创建一个新的雪花算法生成器。
// instanceID 必须是通过 coord.InstanceIDAllocator 获取的、在 [0, 1023] 范围内的唯一ID。
func New(instanceID int64) (Generator, error)
```

#### Generator 接口

```go
// Generator 定义了雪花算法生成器的接口。
type Generator interface {
	// Generate 生成一个 int64 类型的、全局唯一的雪花ID。
	Generate() int64
}
```

#### 工具函数

```go
// Parse 解析一个雪花ID，返回其组成部分：时间戳、实例ID和序列号。
func Parse(id int64) (timestamp, instanceID, sequence int64)
```

### 2.2 `uuid` 包

`uuid` 包用于生成 UUID，其使用是**无状态的**，因此只提供包级函数。

```go
package uuid

// NewV7 生成一个符合 RFC 规范的、时间有序的 UUID v7 字符串。
func NewV7() string

// IsValid 检查一个字符串是否是合法的 UUID 格式。
func IsValid(s string) bool
```

## 3. 标准用法

### 场景 1: 生成数据库主键 (使用 `snowflake`)

```go
import "github.com/ceyewan/gochat/im-infra/uid/snowflake"

// 在服务初始化时
func (s *MessageService) InitSnowflakeGenerator(coordProvider coord.Provider) error {
    // 1. 从 coord 获取 instanceID
    idAllocator, err := coordProvider.InstanceIDAllocator("message-service", 1023)
    if err != nil {
        return err
    }
    instanceID, err := idAllocator.AcquireID(context.Background())
    if err != nil {
        return err
    }

    // 2. 创建并保存雪花算法生成器
    snowGen, err := snowflake.New(int64(instanceID))
    if err != nil {
        return err
    }
    s.idGenerator = snowGen
    return nil
}

// 在业务逻辑中使用
func (s *MessageService) CreateMessage(ctx context.Context, content string) (*Message, error) {
    msg := &Message{
        ID:      s.idGenerator.Generate(), // 生成唯一ID
        Content: content,
    }
    // ... 保存到数据库 ...
    return msg, nil
}
```

### 场景 2: 生成对外暴露的请求 ID (使用 `uuid`)

```go
import "github.com/ceyewan/gochat/im-infra/uid/uuid"

// 在 Gin 中间件中
func RequestIDMiddleware() gin.HandlerFunc {
    return func(c *gin.Context) {
        // 为每个请求生成一个唯一的请求ID
        requestID := uuid.NewV7()
        // 设置到 header 和 context 中
        c.Header("X-Request-ID", requestID)
        c.Set("request_id", requestID)
        c.Next()
    }
}