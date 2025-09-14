# Cache - 分布式缓存服务

`cache` 是一个基于 [go-redis](https://github.com/redis/go-redis) 的高级 Redis 操作包，为 GoChat 项目提供了统一、高性能且功能丰富的分布式缓存能力。它遵循 `im-infra` 的核心设计理念，提供了清晰的分层架构、类型安全的接口和灵活的配置选项。

## 核心特性

- 🏗️ **模块化架构**: 清晰的 `外部 API` -> `内部实现` 分层，职责分离。
- 🔌 **面向接口编程**: 所有功能均通过 `cache.Provider` 接口暴露，易于测试和模拟 (mock)。
- 🛡️ **类型安全**: 所有与时间相关的参数均使用 `time.Duration`，避免整数转换错误。
- 📝 **功能完备**: 提供字符串、哈希、集合、分布式锁、布隆过滤器和 Lua 脚本执行等丰富操作。
- ⚙️ **灵活配置**: 提供 `GetDefaultConfig()` 和 `Option` 函数（如 `WithLogger`），易于定制。
- 📦 **封装设计**: 内部实现对用户透明，通过键前缀（`KeyPrefix`）支持命名空间隔离。
- 📊 **日志集成**: 与 `im-infra/clog` 无缝集成，提供结构化的日志输出。
- 🚫 **错误处理**: 提供标准的 `ErrCacheMiss` 错误类型，便于缓存未命中处理。

## 快速开始

### 安装

```bash
go get github.com/ceyewan/gochat/im-infra/cache
```

### 基础用法

下面的示例展示了如何初始化 `cache` 客户端并执行基本的 `Set` 和 `Get` 操作。

```go
package main

import (
	"context"
	"log"
	"time"

	"github.com/ceyewan/gochat/im-infra/cache"
	"github.com/ceyewan/gochat/im-infra/clog"
)

func main() {
	logger := clog.Namespace("cache-example")
	ctx := context.Background()

	// 使用默认配置，并指定 Redis 地址
	cfg := cache.GetDefaultConfig("development")
	cfg.Addr = "localhost:6379"

	// 创建 Cache 实例
	cacheClient, err := cache.New(ctx, cfg, cache.WithLogger(logger))
	if err != nil {
		log.Fatalf("无法创建缓存客户端: %v", err)
	}
	defer cacheClient.Close()

	// 设置一个键值对，过期时间为 5 分钟
	err = cacheClient.String().Set(ctx, "mykey", "hello world", 5*time.Minute)
	if err != nil {
		log.Fatalf("设置值失败: %v", err)
	}

	// 获取刚刚设置的值
	value, err := cacheClient.String().Get(ctx, "mykey")
	if err != nil {
		if err == cache.ErrCacheMiss {
			log.Printf("键不存在: %v", err)
		} else {
			log.Fatalf("获取值失败: %v", err)
		}
	}

	log.Printf("成功获取值: %s", value)
}
```

## 架构设计

`cache` 包遵循 `im-infra` 中定义的 **客户端包装型 (Client Wrapper)** 原型。

- **公共 API 层 (`cache.go`, `interfaces.go`)**: 定义了所有用户可直接调用的公共接口和 `New` 工厂函数。
- **内部实现层 (`internal/`)**: 包含所有接口的具体实现，通过不同的 `*_ops.go` 文件将功能模块化。
- **依赖流向**: `cache.New()` -> `internal.NewCache()` -> 创建并组装所有操作模块（`stringOperations`, `lockOperations` 等）。

### 目录结构

```
cache/
├── cache.go              # 主入口，New 工厂函数
├── interfaces.go         # 所有公共接口定义 (Cache, Lock, etc.)
├── config.go             # 配置结构体 (Config)
├── options.go            # Option 函数 (WithLogger, etc.)
├── README.md             # 本文档
├── examples/             # 使用示例
│   ├── basic/main.go
│   └── advanced/main.go
└── internal/             # 内部实现
    ├── client.go         # 核心客户端实现
    ├── string_ops.go     # 字符串操作
    ├── hash_ops.go       # 哈希操作
    ├── set_ops.go        # 集合操作
    ├── lock_ops.go       # 分布式锁操作
    ├── bloom_ops.go      # 布隆过滤器操作
    └── scripting_ops.go  # Lua 脚本操作
```

## API 参考

### 主接口 (`cache.Provider`)

`Provider` 接口是所有操作的入口，它提供了访问各种子操作接口的方法。

```go
type Provider interface {
	String() StringOperations
	Hash() HashOperations
	Set() SetOperations
	Lock() LockOperations
	Bloom() BloomFilterOperations
	Script() ScriptingOperations
	Ping(ctx context.Context) error
	Close() error
}
```

### 配置选项 (`cache.Config`)

```go
type Config struct {
	Addr            string        `json:"addr"`
	Password        string        `json:"password"`
	DB              int           `json:"db"`
	PoolSize        int           `json:"poolSize"`
	DialTimeout     time.Duration `json:"dialTimeout"`
	ReadTimeout     time.Duration `json:"readTimeout"`
	WriteTimeout    time.Duration `json:"writeTimeout"`
	KeyPrefix       string        `json:"keyPrefix"`
	// ... 更多选项
}
```

### 操作接口

#### 字符串 (`StringOperations`)
- `Set(ctx, key, value, expiration)`: 设置键值对
- `Get(ctx, key)`: 获取值，不存在时返回 `ErrCacheMiss`
- `GetSet(ctx, key, value)`: 设置新值并返回旧值
- `Incr(ctx, key)` / `Decr(ctx, key)`: 递增/递减计数器
- `Del(ctx, keys...)`: 删除键
- `Exists(ctx, keys...)`: 检查键是否存在
- `SetNX(ctx, key, value, expiration)`: 键不存在时设置

#### 哈希 (`HashOperations`)
- `HSet(ctx, key, field, value)`: 设置哈希字段
- `HGet(ctx, key, field)`: 获取哈希字段值，不存在时返回 `ErrCacheMiss`
- `HGetAll(ctx, key)`: 获取所有哈希字段
- `HDel(ctx, key, fields...)`: 删除哈希字段
- `HExists(ctx, key, field)`: 检查字段是否存在
- `HLen(ctx, key)`: 获取哈希字段数量

#### 集合 (`SetOperations`)
- `SAdd(ctx, key, members...)`: 添加成员到集合
- `SIsMember(ctx, key, member)`: 检查成员是否在集合中

#### 分布式锁 (`LockOperations`)
- `Acquire(ctx, key, expiration)`: 获取一个锁实例
- `lock.Unlock(ctx)`: 释放锁
- `lock.Refresh(ctx, expiration)`: 为锁续期

#### 布隆过滤器 (`BloomFilterOperations`)
- `BFReserve(ctx, key, errorRate, capacity)`: 初始化过滤器
- `BFAdd(ctx, key, item)`: 添加元素
- `BFExists(ctx, key, item)`: 检查元素是否存在

#### Lua 脚本 (`ScriptingOperations`)
- `ScriptLoad(ctx, script)`: 加载 Lua 脚本并返回 SHA1
- `ScriptExists(ctx, sha1)`: 检查脚本是否存在
- `EvalSha(ctx, sha1, keys, args)`: 执行已加载的脚本

## 示例代码

- **基础用法**: [examples/basic/main.go](./examples/basic/main.go) - 字符串、哈希、集合操作
- **高级用法**: [examples/advanced/main.go](./examples/advanced/main.go) - 分布式锁、布隆过滤器
- **综合演示**: [examples/comprehensive/main.go](./examples/comprehensive/main.go) - 所有接口的完整演示

### 错误处理

缓存操作可能返回 `ErrCacheMiss` 错误，表示请求的键不存在：

```go
value, err := cacheClient.String().Get(ctx, "key")
if err != nil {
    if err == cache.ErrCacheMiss {
        // 键不存在，执行相应处理
        log.Printf("缓存未命中")
    } else {
        // 其他错误
        log.Printf("获取失败: %v", err)
    }
}
```

### 配置选项

#### 环境相关配置

`GetDefaultConfig()` 函数根据环境返回不同的默认配置：

```go
// 开发环境配置
devConfig := cache.GetDefaultConfig("development")
// devConfig.Addr = "localhost:6379"
// devConfig.PoolSize = 10

// 生产环境配置
prodConfig := cache.GetDefaultConfig("production")
// prodConfig.Addr = "redis:6379"
// prodConfig.PoolSize = 100
```

#### 选项模式

使用 `Option` 函数进行定制化配置：

```go
logger := clog.Namespace("my-app")
cacheClient, err := cache.New(ctx, cfg, cache.WithLogger(logger))
```

## 贡献

欢迎通过提交 Issue 和 Pull Request 来改进此包。
