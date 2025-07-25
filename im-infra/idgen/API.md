# ID 生成器 API 使用文档

本文档提供 id-gen 库的完整 API 使用指南，包含所有接口的使用方法和示例代码。

## 快速开始

### 安装

```bash
go get github.com/ceyewan/gochat/im-infra/id-gen
```

### 基本使用

```go
package main

import (
    "context"
    "fmt"
    "github.com/ceyewan/gochat/im-infra/id-gen"
)

func main() {
    ctx := context.Background()
    
    // 生成 int64 类型的 ID（默认使用雪花算法）
    id, err := idgen.GenerateInt64(ctx)
    if err != nil {
        panic(err)
    }
    fmt.Printf("生成的 ID: %d\n", id)
    
    // 生成字符串类型的 ID
    idStr, err := idgen.GenerateString(ctx)
    if err != nil {
        panic(err)
    }
    fmt.Printf("生成的字符串 ID: %s\n", idStr)
}
```

## 全局方法 API

### GenerateInt64

生成 int64 类型的 ID。

```go
func GenerateInt64(ctx context.Context) (int64, error)
```

**参数：**
- `ctx`：上下文，用于超时控制和取消操作

**返回值：**
- `int64`：生成的 ID
- `error`：错误信息

**示例：**
```go
ctx := context.Background()
id, err := idgen.GenerateInt64(ctx)
if err != nil {
    log.Fatal(err)
}
fmt.Printf("ID: %d\n", id)
```

### GenerateString

生成字符串类型的 ID。

```go
func GenerateString(ctx context.Context) (string, error)
```

**参数：**
- `ctx`：上下文

**返回值：**
- `string`：生成的 ID 字符串
- `error`：错误信息

**示例：**
```go
ctx := context.Background()
id, err := idgen.GenerateString(ctx)
if err != nil {
    log.Fatal(err)
}
fmt.Printf("ID: %s\n", id)
```

### Type

获取默认生成器的类型。

```go
func Type() GeneratorType
```

**返回值：**
- `GeneratorType`：生成器类型（"snowflake"、"uuid"、"redis"）

**示例：**
```go
generatorType := idgen.Type()
fmt.Printf("当前生成器类型: %s\n", generatorType)
```

### GetSnowflakeID（向后兼容）

生成雪花算法 ID，保持向后兼容性。

```go
func GetSnowflakeID() int64
```

**返回值：**
- `int64`：雪花算法生成的 ID

**示例：**
```go
id := idgen.GetSnowflakeID()
fmt.Printf("雪花 ID: %d\n", id)
```

## 工厂方法 API

### New

根据配置创建 ID 生成器。

```go
func New(cfg Config) (IDGenerator, error)
```

**参数：**
- `cfg`：生成器配置

**返回值：**
- `IDGenerator`：生成器实例
- `error`：错误信息

**示例：**
```go
config := idgen.Config{
    Type: idgen.SnowflakeType,
    Snowflake: &idgen.SnowflakeConfig{
        NodeID:     1,
        AutoNodeID: false,
        Epoch:      1288834974657,
    },
}

generator, err := idgen.New(config)
if err != nil {
    log.Fatal(err)
}
defer generator.Close()

id, err := generator.GenerateInt64(ctx)
```

### NewSnowflakeGenerator

创建雪花算法生成器。

```go
func NewSnowflakeGenerator(config *SnowflakeConfig) (SnowflakeGenerator, error)
```

**参数：**
- `config`：雪花算法配置，传 nil 使用默认配置

**返回值：**
- `SnowflakeGenerator`：雪花算法生成器
- `error`：错误信息

**示例：**
```go
// 使用默认配置
generator, err := idgen.NewSnowflakeGenerator(nil)
if err != nil {
    log.Fatal(err)
}
defer generator.Close()

// 使用自定义配置
config := &idgen.SnowflakeConfig{
    NodeID:     123,
    AutoNodeID: false,
    Epoch:      1288834974657,
}
generator, err = idgen.NewSnowflakeGenerator(config)
```

### NewUUIDGenerator

创建 UUID 生成器。

```go
func NewUUIDGenerator(config *UUIDConfig) (UUIDGenerator, error)
```

**参数：**
- `config`：UUID 配置，传 nil 使用默认配置

**返回值：**
- `UUIDGenerator`：UUID 生成器
- `error`：错误信息

**示例：**
```go
// 使用默认配置（UUID v4，标准格式，小写）
generator, err := idgen.NewUUIDGenerator(nil)
if err != nil {
    log.Fatal(err)
}
defer generator.Close()

// 使用自定义配置
config := &idgen.UUIDConfig{
    Version:   7,          // UUID v7（时间排序）
    Format:    "simple",   // 简单格式（无连字符）
    UpperCase: true,       // 大写
}
generator, err = idgen.NewUUIDGenerator(config)
```

### NewRedisGenerator

创建 Redis 自增 ID 生成器。

```go
func NewRedisGenerator(config *RedisConfig) (RedisIDGenerator, error)
```

**参数：**
- `config`：Redis 配置，不能为 nil

**返回值：**
- `RedisIDGenerator`：Redis 生成器
- `error`：错误信息

**示例：**
```go
config := &idgen.RedisConfig{
    CacheConfig: cache.Config{
        Addr:     "localhost:6379",
        Password: "",
        DB:       0,
        PoolSize: 10,
    },
    KeyPrefix:    "myapp",
    DefaultKey:   "counter",
    Step:         1,
    InitialValue: 1,
    TTL:          0, // 不过期
}

generator, err := idgen.NewRedisGenerator(config)
if err != nil {
    log.Fatal(err)
}
defer generator.Close()
```

## 接口方法 API

### IDGenerator 接口

所有生成器都实现的基础接口。

```go
type IDGenerator interface {
    GenerateString(ctx context.Context) (string, error)
    GenerateInt64(ctx context.Context) (int64, error)
    Type() GeneratorType
    Close() error
}
```

**方法说明：**
- `GenerateString`：生成字符串 ID
- `GenerateInt64`：生成 int64 ID
- `Type`：获取生成器类型
- `Close`：关闭生成器，释放资源

### SnowflakeGenerator 接口

雪花算法生成器的特化接口。

```go
type SnowflakeGenerator interface {
    IDGenerator
    GetNodeID() int64
    ParseID(id int64) (timestamp int64, nodeID int64, sequence int64)
}
```

**特有方法：**

#### GetNodeID

获取当前节点 ID。

```go
nodeID := generator.GetNodeID()
fmt.Printf("节点 ID: %d\n", nodeID)
```

#### ParseID

解析雪花 ID，提取时间戳、节点 ID 和序列号。

```go
id := int64(1234567890123456789)
timestamp, nodeID, sequence := generator.ParseID(id)
fmt.Printf("时间戳: %d, 节点ID: %d, 序列号: %d\n", timestamp, nodeID, sequence)
```

### UUIDGenerator 接口

UUID 生成器的特化接口。

```go
type UUIDGenerator interface {
    IDGenerator
    GenerateV4(ctx context.Context) (string, error)
    GenerateV7(ctx context.Context) (string, error)
    Validate(uuid string) bool
}
```

**特有方法：**

#### GenerateV4

生成 UUID v4（随机 UUID）。

```go
uuid, err := generator.GenerateV4(ctx)
if err != nil {
    log.Fatal(err)
}
fmt.Printf("UUID v4: %s\n", uuid)
```

#### GenerateV7

生成 UUID v7（时间排序 UUID）。

```go
uuid, err := generator.GenerateV7(ctx)
if err != nil {
    log.Fatal(err)
}
fmt.Printf("UUID v7: %s\n", uuid)
```

#### Validate

验证 UUID 格式是否正确。

```go
uuid := "550e8400-e29b-41d4-a716-446655440000"
isValid := generator.Validate(uuid)
fmt.Printf("UUID 有效性: %t\n", isValid)
```

### RedisIDGenerator 接口

Redis 自增 ID 生成器的特化接口。

```go
type RedisIDGenerator interface {
    IDGenerator
    GenerateWithKey(ctx context.Context, key string) (int64, error)
    GenerateWithStep(ctx context.Context, step int64) (int64, error)
    Reset(ctx context.Context, key string) error
    GetCurrent(ctx context.Context, key string) (int64, error)
}
```

**特有方法：**

#### GenerateWithKey

使用指定键生成 ID。

```go
id, err := generator.GenerateWithKey(ctx, "user_counter")
if err != nil {
    log.Fatal(err)
}
fmt.Printf("用户计数器 ID: %d\n", id)
```

#### GenerateWithStep

使用指定步长生成 ID。

```go
// 一次性获取 10 个 ID
id, err := generator.GenerateWithStep(ctx, 10)
if err != nil {
    log.Fatal(err)
}
fmt.Printf("批量获取的最后一个 ID: %d\n", id)
```

#### Reset

重置指定键的计数器。

```go
err := generator.Reset(ctx, "user_counter")
if err != nil {
    log.Fatal(err)
}
fmt.Println("计数器已重置")
```

#### GetCurrent

获取当前计数值。

```go
current, err := generator.GetCurrent(ctx, "user_counter")
if err != nil {
    log.Fatal(err)
}
fmt.Printf("当前计数值: %d\n", current)
```

## 配置 API

### 默认配置

获取预设的默认配置。

```go
// 默认配置（雪花算法）
defaultConfig := idgen.DefaultConfig()

// 雪花算法默认配置
snowflakeConfig := idgen.DefaultSnowflakeConfig()

// UUID 默认配置
uuidConfig := idgen.DefaultUUIDConfig()

// Redis 默认配置
redisConfig := idgen.DefaultRedisConfig()
```

### SnowflakeConfig

雪花算法配置结构体。

```go
type SnowflakeConfig struct {
    NodeID     int64 // 节点 ID（0-1023），为 0 时自动生成
    AutoNodeID bool  // 是否自动从 IP 地址生成节点 ID
    Epoch      int64 // 自定义起始时间戳（毫秒）
}
```

**使用示例：**
```go
config := &idgen.SnowflakeConfig{
    NodeID:     100,           // 手动指定节点 ID
    AutoNodeID: false,         // 不自动生成
    Epoch:      1640995200000, // 2022-01-01 00:00:00 UTC
}
```

### UUIDConfig

UUID 配置结构体。

```go
type UUIDConfig struct {
    Version   int    // UUID 版本（4 或 7）
    Format    string // 输出格式（"standard" 或 "simple"）
    UpperCase bool   // 是否使用大写字母
}
```

**使用示例：**
```go
config := &idgen.UUIDConfig{
    Version:   7,        // UUID v7（时间排序）
    Format:    "simple", // 简单格式（无连字符）
    UpperCase: true,     // 大写字母
}
// 输出示例：A1B2C3D4E5F6789012345678901234AB
```

### RedisConfig

Redis 自增 ID 配置结构体。

```go
type RedisConfig struct {
    CacheConfig  cache.Config  // Redis 连接配置
    KeyPrefix    string        // 键前缀
    DefaultKey   string        // 默认键名
    Step         int64         // 自增步长
    InitialValue int64         // 初始值
    TTL          time.Duration // 键的过期时间
}
```

**使用示例：**
```go
config := &idgen.RedisConfig{
    CacheConfig: cache.Config{
        Addr:     "localhost:6379",
        Password: "your-password",
        DB:       0,
        PoolSize: 20,
    },
    KeyPrefix:    "prod_app",    // 键前缀
    DefaultKey:   "global",      // 默认键名
    Step:         1,             // 每次递增 1
    InitialValue: 10000,         // 从 10000 开始
    TTL:          time.Hour * 24, // 24 小时过期
}
// Redis 键名：prod_app:global
```

## 错误处理

### 常见错误类型

```go
// 配置错误
err := config.Validate()
if err != nil {
    // 处理配置验证错误
}

// 连接错误（Redis）
generator, err := idgen.NewRedisGenerator(config)
if err != nil {
    // 处理 Redis 连接错误
}

// 生成错误
id, err := generator.GenerateInt64(ctx)
if err != nil {
    // 处理 ID 生成错误
}
```

### 错误处理最佳实践

```go
func generateIDWithFallback(ctx context.Context) int64 {
    id, err := idgen.GenerateInt64(ctx)
    if err != nil {
        // 记录错误
        log.Error("ID 生成失败", "error", err)

        // 使用备用方案
        return time.Now().UnixNano()
    }
    return id
}
```

## 完整示例

### 多生成器使用示例

```go
package main

import (
    "context"
    "fmt"
    "log"
    "time"

    "github.com/ceyewan/gochat/im-infra/id-gen"
    "github.com/ceyewan/gochat/im-infra/cache"
)

func main() {
    ctx := context.Background()

    // 1. 使用全局方法（默认雪花算法）
    fmt.Println("=== 全局方法 ===")
    globalID, err := idgen.GenerateInt64(ctx)
    if err != nil {
        log.Fatal(err)
    }
    fmt.Printf("全局 ID: %d\n", globalID)

    // 2. 雪花算法生成器
    fmt.Println("\n=== 雪花算法 ===")
    snowflakeGen, err := idgen.NewSnowflakeGenerator(&idgen.SnowflakeConfig{
        NodeID:     1,
        AutoNodeID: false,
        Epoch:      1288834974657,
    })
    if err != nil {
        log.Fatal(err)
    }
    defer snowflakeGen.Close()

    snowflakeID, _ := snowflakeGen.GenerateInt64(ctx)
    nodeID := snowflakeGen.GetNodeID()
    timestamp, parsedNodeID, sequence := snowflakeGen.ParseID(snowflakeID)

    fmt.Printf("雪花 ID: %d\n", snowflakeID)
    fmt.Printf("节点 ID: %d\n", nodeID)
    fmt.Printf("解析结果 - 时间戳: %d, 节点: %d, 序列: %d\n",
        timestamp, parsedNodeID, sequence)

    // 3. UUID 生成器
    fmt.Println("\n=== UUID ===")
    uuidGen, err := idgen.NewUUIDGenerator(&idgen.UUIDConfig{
        Version:   4,
        Format:    "standard",
        UpperCase: false,
    })
    if err != nil {
        log.Fatal(err)
    }
    defer uuidGen.Close()

    uuid4, _ := uuidGen.GenerateV4(ctx)
    uuid7, _ := uuidGen.GenerateV7(ctx)
    isValid := uuidGen.Validate(uuid4)

    fmt.Printf("UUID v4: %s\n", uuid4)
    fmt.Printf("UUID v7: %s\n", uuid7)
    fmt.Printf("UUID 验证: %t\n", isValid)

    // 4. Redis 生成器（需要 Redis 服务）
    fmt.Println("\n=== Redis 自增 ===")
    redisGen, err := idgen.NewRedisGenerator(&idgen.RedisConfig{
        CacheConfig: cache.Config{
            Addr:     "localhost:6379",
            Password: "",
            DB:       0,
            PoolSize: 5,
        },
        KeyPrefix:    "example",
        DefaultKey:   "counter",
        Step:         1,
        InitialValue: 1,
        TTL:          0,
    })
    if err != nil {
        log.Printf("Redis 连接失败，跳过 Redis 测试: %v", err)
    } else {
        defer redisGen.Close()

        redisID, _ := redisGen.GenerateInt64(ctx)
        customID, _ := redisGen.GenerateWithKey(ctx, "custom")
        current, _ := redisGen.GetCurrent(ctx, "counter")

        fmt.Printf("Redis ID: %d\n", redisID)
        fmt.Printf("自定义键 ID: %d\n", customID)
        fmt.Printf("当前计数: %d\n", current)
    }
}
```

这个 API 文档提供了完整的使用指南，您可以根据具体需求选择合适的 API 和配置。
