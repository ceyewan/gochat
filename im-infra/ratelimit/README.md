# RateLimit 分布式限流组件

`ratelimit` 是一个基于 Redis 的高性能分布式限流组件，专为微服务架构设计。

## 核心特性

- **🚀 高性能**: 基于 Redis 和 Lua 脚本实现，保证原子性操作和低延迟。
- **令牌桶算法**: 采用令牌桶算法，能够平滑处理突发流量，比传统计数器更灵活。
- **🌐 分布式**: 天然支持分布式环境，适用于微服务集群。
- **⚙️ 动态配置**: 与 `coordination` 组件集成，可通过 etcd 动态更新限流规则，无需重启服务。
- **🛡️ 高可用**: 在配置中心或 Redis 故障时，可配置默认规则或选择性放行，保证核心业务不受影响。
- **📝 结构化日志**: 与 `clog` 组件深度集成，提供详细的结构化日志，便于监控和排障。
- **🧩 模块化设计**: 遵循 `im-infra` 的设计规范，接口清晰，易于集成和扩展。

## 快速开始

### 1. 安装

```bash
go get github.com/ceyewan/gochat/im-infra/ratelimit
```

### 2. 基本用法

```go
package main

import (
	"context"
	"fmt"
	"time"

	"github.com/ceyewan/gochat/im-infra/ratelimit"
)

func main() {
	// 定义默认规则，以防无法从配置中心加载
	defaultRules := map[string]ratelimit.Rule{
		"api_requests": {Rate: 5, Capacity: 5}, // 每秒 5 个请求
	}

	// 初始化限流器
	limiter, err := ratelimit.New(
		context.Background(),
		"my-service", // 用于在 etcd 中查找 /configimpl/{env}/my-service/ratelimit/*
		ratelimit.WithDefaultRules(defaultRules),
	)
	if err != nil {
		panic(err)
	}
	defer limiter.Close()

	// 模拟限流检查
	for i := 0; i < 7; i++ {
		// 对 "ip:1.2.3.4" 这个资源应用 "api_requests" 规则
		allowed, _ := limiter.Allow(context.Background(), "ip:1.2.3.4", "api_requests")
		if allowed {
			fmt.Printf("请求 %d: ✅ 允许\n", i+1)
		} else {
			fmt.Printf("请求 %d: ❌ 拒绝\n", i+1)
		}
		time.Sleep(100 * time.Millisecond)
	}
}
```

## 配置

限流规则通过 `coordination` 组件从 etcd 中加载。路径遵循以下格式：

`/config/{环境}/{服务名}/ratelimit/{规则名}`

- `{环境}`: 由环境变量 `APP_ENV` 指定，默认为 `dev`。
- `{服务名}`: 在 `ratelimit.New()` 中传入的 `serviceName`。
- `{规则名}`: 在 `limiter.Allow()` 中传入的 `ruleName`。

**配置值 (JSON 格式):**

```json
{
  "rate": 10.0,
  "capacity": 20
}
```

- `rate`: 每秒生成的令牌数。
- `capacity`: 令牌桶的最大容量。

## 贡献

欢迎提交 issue 和 pull request。