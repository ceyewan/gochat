# Cache - 分布式缓存服务

`cache` 是一个基于 [go-redis](https://github.com/redis/go-redis) 的高级 Redis 操作包，为 GoChat 项目提供了统一、高性能且功能丰富的分布式缓存能力。它遵循 `im-infra` 的核心设计理念，提供了清晰的分层架构、类型安全的接口和灵活的配置选项。

## 核心特性

- 🏗️ **模块化架构**: 清晰的 `外部 API` -> `内部实现` 分层，职责分离。
- 🔌 **面向接口编程**: 所有功能均通过 `cache.Cache` 接口暴露，易于测试和模拟 (mock)。
- 🛡️ **类型安全**: 所有与时间相关的参数均使用 `time.Duration`，避免整数转换错误。
- 📝 **功能完备**: 提供字符串、哈希、集合、分布式锁、布隆过滤器和 Lua 脚本执行等丰富操作。
- ⚙️ **灵活配置**: 提供 `DefaultConfig()` 和 `Option` 函数（如 `WithLogger`），易于定制。
- 📦 **封装设计**: 内部实现对用户透明，通过键前缀（`KeyPrefix`）支持命名空间隔离。
- 📊 **日志集成**: 与 `im-infra/clog` 无缝集成，提供结构化的日志输出。

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
	logger := clog.Module("cache-example")
	ctx := context.Background()

	// 使用默认配置，并指定 Redis 地址
	cfg := cache.DefaultConfig()
	cfg.Addr = "localhost:6379"

	// 创建 Cache 实例
	cacheClient, err := cache.New(ctx, cfg, cache.WithLogger(logger))
	if err != nil {
		log.Fatalf("无法创建缓存客户端: %v", err)
	}
	defer cacheClient.Close()

	// 设置一个键值对，过期时间为 5 分钟
	err = cacheClient.Set(ctx, "mykey", "hello world", 5*time.Minute)
	if err != nil {
		log.Fatalf("设置值失败: %v", err)
	}

	// 获取刚刚设置的值
	value, err := cacheClient.Get(ctx, "mykey")
	if err != nil {
		log.Fatalf("获取值失败: %v", err)
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

### 主接口 (`cache.Cache`)

`Cache` 接口是所有操作的入口，它组合了各种数据结构的操作接口。

```go
type Cache interface {
	StringOperations
	HashOperations
	SetOperations
	LockOperations
	BloomFilterOperations
	ScriptingOperations

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
- `Set(ctx, key, value, expiration)`
- `Get(ctx, key)`
- `Incr(ctx, key)` / `Decr(ctx, key)`
- `Del(ctx, keys...)`

#### 哈希 (`HashOperations`)
- `HSet(ctx, key, field, value)`
- `HGet(ctx, key, field)`
- `HGetAll(ctx, key)`

#### 集合 (`SetOperations`)
- `SAdd(ctx, key, members...)`
- `SIsMember(ctx, key, member)`
- `SMembers(ctx, key)`

#### 分布式锁 (`LockOperations`)
- `Lock(ctx, key, expiration)`: 获取一个锁实例。
- `lock.Unlock(ctx)`: 释放锁。
- `lock.Refresh(ctx, expiration)`: 为锁续期。

#### 布隆过滤器 (`BloomFilterOperations`)
- `BFInit(ctx, key, errorRate, capacity)`: 初始化过滤器。
- `BFAdd(ctx, key, item)`: 添加元素。
- `BFExists(ctx, key, item)`: 检查元素是否存在。

## 示例代码

- **基础用法**: [examples/basic/main.go](./examples/basic/main.go)
- **高级用法** (分布式锁, 布隆过滤器): [examples/advanced/main.go](./examples/advanced/main.go)

## 贡献

欢迎通过提交 Issue 和 Pull Request 来改进此包。
