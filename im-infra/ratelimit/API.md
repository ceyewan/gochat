# RateLimit API 文档

本文档详细介绍了 `ratelimit` 组件的公共 API。

## 概述

`ratelimit` 组件提供了一个 `RateLimiter` 接口，以及用于创建实例的工厂函数。

## 工场函数

### New

```go
func New(ctx context.Context, serviceName string, opts ...Option) (RateLimiter, error)
```

创建一个新的限流器实例。

- `ctx`: 上下文，用于控制限流器实例的生命周期。当此 `ctx` 被取消时，后台的规则刷新协程将停止。
- `serviceName`: 服务名称，用于构建在 etcd 中查找配置的路径。例如，如果 `serviceName` 为 "im-gateway"，则配置路径为 `/config/{env}/im-gateway/ratelimit/*`。
- `opts`: 一系列配置选项，用于自定义限流器的行为。

### Default

```go
func Default() RateLimiter
```

返回一个全局共享的单例 `RateLimiter` 实例。该实例使用默认配置，`serviceName` 为 "default"。适用于简单的场景或快速测试。

## 接口：RateLimiter

```go
type RateLimiter interface {
    Allow(ctx context.Context, resource string, ruleName string) (bool, error)
    Close() error
}
```

### Allow

```go
func (l *limiter) Allow(ctx context.Context, resource string, ruleName string) (bool, error)
```

检查对特定资源的请求是否应被允许。这是限流器的核心方法。

- `ctx`: 请求级别的上下文。
- `resource`: 受限资源的唯一标识符。这可以是一个用户ID、一个IP地址、一个设备ID等。例如: `"user:12345"`, `"ip:192.168.1.1"`.
- `ruleName`: 要应用的规则的名称。组件将使用此名称从配置中查找对应的令牌桶容量和速率。例如: `"user_message_frequency"`, `"api_registration_limit"`.
- **返回值**:
    - `bool`: `true` 表示请求被允许，`false` 表示被拒绝。
    - `error`: 如果在与 Redis 通信过程中发生错误，将返回一个错误。注意：在发生错误时，为了保证系统可用性，默认行为是允许请求通过。

### Close

```go
func (l *limiter) Close() error
```

释放限流器持有的资源，主要是停止后台的配置刷新协程。对于通过 `New` 创建的每个实例，都应该在使用完毕后调用 `Close` 以避免协程泄漏。

## 配置选项 (Options)

通过向 `New` 函数传递一个或多个 `Option` 来自定义限流器。

### Option

```go
type Option func(*Options)
```

`Option` 是一个函数类型，用于修改内部的 `Options` 结构。

### WithCacheClient

```go
func WithCacheClient(client cache.Cache) Option
```

传入一个自定义的 `cache.Cache` 实例。如果未提供，则默认使用 `cache.Default()`。

### WithCoordinationClient

```go
func WithCoordinationClient(client coordination.Coordinator) Option
```

传入一个自定义的 `coordination.Coordinator` 实例。如果未提供，则默认使用 `coordination.Default()`。

### WithDefaultRules

```go
func WithDefaultRules(rules map[string]Rule) Option
```

设置一套默认规则。当无法从 `coordination` 服务获取到配置时（例如，在服务启动阶段或 etcd 故障时），将使用这些规则。

### WithRuleRefreshInterval

```go
func WithRuleRefreshInterval(interval time.Duration) Option
```

设置从配置中心检查规则更新的频率。默认值为 1 分钟。

## 结构体

### Rule

```go
type Rule struct {
	Rate     float64 `json:"rate"`
	Capacity int64   `json:"capacity"`
}
```

定义了一个限流规则。

- `Rate`: 每秒生成的令牌数。可以为小数。
- `Capacity`: 令牌桶的最大容量，即允许的突发请求峰值。