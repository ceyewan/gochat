# UID - 唯一ID生成器

UID 是 GoChat 的高性能唯一ID生成服务，提供雪花算法和 UUID 生成功能。

## 功能特性

- **雪花算法**: 分布式、时间有序的 64 位整数 ID 
- **UUID 生成**: 符合 RFC 4122 标准的 UUID v4 和 v7
- **线程安全**: 并发 ID 生成，无重复
- **高性能**: 针对高吞吐量场景优化
- **可配置**: 支持 Worker ID 和数据中心 ID 配置
- **零依赖**: 最小化外部依赖

## 快速开始

### 安装

```bash
go get github.com/ceyewan/gochat/im-infra/uid
```

### 基本使用

```go
package main

import (
    "context"
    "fmt"
    "log"

    "github.com/ceyewan/gochat/im-infra/uid"
)

func main() {
    // 使用默认配置创建生成器
    cfg := uid.DefaultConfig()
    generator, err := uid.New(context.Background(), cfg)
    if err != nil {
        log.Fatal(err)
    }
    defer generator.Close()

    // 生成雪花算法 ID
    snowflakeID := generator.GenerateInt64()
    fmt.Printf("雪花算法 ID: %d\n", snowflakeID)

    // 生成 UUID 字符串
    uuid := generator.GenerateString()
    fmt.Printf("UUID: %s\n", uuid)
}
```

## 配置选项

```go
type Config struct {
    WorkerID     int64 `json:"workerID" yaml:"workerID"`        // 0-31
    DatacenterID int64 `json:"datacenterID" yaml:"datacenterID"` // 0-31  
    EnableUUID   bool  `json:"enableUUID" yaml:"enableUUID"`   // true 使用 UUID，false 使用雪花字符串
}
```

### 默认配置

```go
cfg := uid.DefaultConfig()
// WorkerID: 1
// DatacenterID: 1
// EnableUUID: true
```

### 自定义配置

```go
cfg := uid.Config{
    WorkerID:     5,
    DatacenterID: 3,
    EnableUUID:   false,
}

generator, err := uid.New(context.Background(), cfg)
```

## 高级功能

### UUID v4 和 v7 生成

```go
// 生成 UUID v4（随机 UUID）
uuidV4 := generator.GenerateUUIDV4()
fmt.Printf("UUID v4: %s\n", uuidV4)

// 生成 UUID v7（时间排序 UUID）
uuidV7 := generator.GenerateUUIDV7()
fmt.Printf("UUID v7: %s\n", uuidV7)

// 验证 UUID 格式
isValid := generator.ValidateUUID("550e8400-e29b-41d4-a716-446655440000")
fmt.Printf("UUID 有效性: %t\n", isValid)
```

### 雪花算法 ID 解析

```go
// 生成雪花算法 ID
id := generator.GenerateInt64()

// 解析 ID 获取详细信息
timestamp, workerID, datacenterID, sequence := generator.ParseSnowflake(id)
fmt.Printf("时间戳: %d, WorkerID: %d, 数据中心ID: %d, 序列号: %d\n", 
    timestamp, workerID, datacenterID, sequence)
```

## 选项配置

### WithLogger

```go
import "github.com/ceyewan/gochat/im-infra/clog"

logger := clog.Module("my-app")
generator, err := uid.New(ctx, cfg, uid.WithLogger(logger))
```

### WithComponentName

```go
generator, err := uid.New(ctx, cfg, uid.WithComponentName("user-service"))
```

## 使用示例

### 基本用法

```go
// 生成唯一 ID
for i := 0; i < 10; i++ {
    id := generator.GenerateInt64()
    fmt.Printf("ID %d: %d\n", i, id)
}
```

### 并发生成

```go
var wg sync.WaitGroup
idMap := sync.Map{}

for i := 0; i < 100; i++ {
    wg.Add(1)
    go func() {
        defer wg.Done()
        id := generator.GenerateInt64()
        if _, loaded := idMap.LoadOrStore(id, true); loaded {
            fmt.Println("检测到重复 ID!")
        }
    }()
}

wg.Wait()
```

### 混合模式使用

```go
// UUID 模式（默认）
uuidGen, _ := uid.New(ctx, uid.DefaultConfig())
uuid := uuidGen.GenerateString() // 返回 UUID 格式

// 雪花字符串模式
cfg := uid.DefaultConfig()
cfg.EnableUUID = false
snowflakeGen, _ := uid.New(ctx, cfg)
id := snowflakeGen.GenerateString() // 返回雪花算法字符串
```

## 雪花算法 ID 结构

雪花算法 ID 是 64 位整数，结构如下：

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

- **41 位**: 时间戳（毫秒，相对于起始时间）
- **10 位**: 数据中心 ID (0-31)
- **10 位**: Worker ID (0-31)
- **12 位**: 序列号 (0-4095)

## 性能表现

在典型硬件上的基准测试结果：

```
BenchmarkGenerateInt64-8      30000000    45.2 ns/op    0 B/op    0 allocs/op
BenchmarkGenerateString-8     2000000    742 ns/op    48 B/op    1 allocs/op
```

## 错误处理

组件在启动时验证配置：

```go
cfg := uid.Config{
    WorkerID: 32, // 无效：必须是 0-31
}

generator, err := uid.New(ctx, cfg)
if err != nil {
    // 处理配置错误
    log.Printf("配置无效: %v", err)
}
```

## 测试

运行测试套件：

```bash
go test ./im-infra/uid/...
```

运行基准测试：

```bash
go test -bench=. ./im-infra/uid/...
```

## 最佳实践

1. **Worker ID 管理**: 确保所有实例使用唯一的 Worker ID
2. **数据中心 ID**: 用于多数据中心部署
3. **时钟同步**: 保持系统时钟同步以确保雪花算法正确性
4. **资源清理**: 使用完毕后始终调用 `Close()`
5. **配置验证**: 部署前验证配置

## 示例代码

查看 [examples](examples/) 目录获取更多详细用法：

- [基本用法](examples/basic/main.go)
- [高级特性](examples/advanced/main.go)

## 许可证

此组件是 GoChat 项目的一部分，遵循相同的许可条款。