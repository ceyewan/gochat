# 基础设施: ratelimit 分布式限流

## 1. 设计理念

`ratelimit` 组件的设计遵循 **KISS (Keep It Simple, Stupid)** 和 **高内聚** 的原则，旨在提供一个极简、可靠且对开发者友好的分布式限流解决方案。

- **极简 API**: 组件只暴露一个核心方法 `Allow`。所有复杂的功能如批量操作、多令牌消耗都被移除，以保证接口的纯粹性和易用性。
- **声明式配置**: 限流规则的管理完全通过配置中心（由 `coord` 提供支持）进行。运维人员通过修改配置文件来“声明”期望的状态，而不是通过 API “命令”组件修改规则。这更符合现代的 GitOps 理念。
- **组件自治的动态配置**: `ratelimit` 组件内部自己负责监听其在配置中心的规则变化。它直接使用 `coord` 提供的 `Watch` 功能，实现了“组件自治”的热更新，无需一个复杂的、全局的配置分发框架。这种模式保证了 `coord` 的简洁性和 `ratelimit` 的高内聚性。
- **平滑与突发兼顾**: 采用令牌桶算法，允许在平均速率限制之下，处理一定量的突发流量（由桶的容量决定），这比简单的计数器算法更能适应真实世界的流量模式。
- **多维度防护**: 组件的设计支持基于多种维度的限流，如用户ID、IP地址、API端点等，通过构建不同的 `resource` 键来实现。

## 2. 核心 API 契约

`ratelimit` 的公开 API 被精简到极致，只包含一个核心功能接口和一个标准化的构造函数。

### 2.1 构造函数

```go
// Config 是 ratelimit 组件的配置结构体。
type Config struct {
    // ServiceName 用于日志记录和监控，以区分是哪个服务在使用限流器。
    ServiceName string `json:"serviceName"`

    // RulesPath 是在 coord 配置中心存储此服务限流规则的根路径。
    // 约定：此路径必须以 "/" 结尾。
    // 例如："/config/dev/im-gateway/ratelimit/"
    RulesPath string `json:"rulesPath"`
}

// New 创建一个新的限流器实例。
// 它会自动从 coord 配置中心加载初始规则，并启动一个后台协程监听后续的规则变更。
func New(ctx context.Context, config *Config, opts ...Option) (RateLimiter, error)
```
*注意：`Option` 可用于注入自定义的 `clog.Logger` 或 `coord.Provider` 等依赖。*

### 2.2 RateLimiter 接口

```go
// RateLimiter 是限流器的主接口。
type RateLimiter interface {
    // Allow 检查给定资源的单个请求是否被允许。
    // resource 是被限流的唯一标识，如 "user:123" 或 "ip:1.2.3.4"。
    // ruleName 是要应用的规则名，如 "api_default"。如果规则不存在，将采用失败策略（默认为拒绝）。
    Allow(ctx context.Context, resource, ruleName string) (bool, error)

    // Close 关闭限流器，释放后台协程和连接等资源。
    Close() error
}
```

## 3. 标准用法

### 场景：在 Gin 中间件中保护 API

```go
import "github.com/ceyewan/gochat/im-infra/ratelimit"

// RateLimitMiddleware 创建一个 Gin 中间件用于 API 限流。
func RateLimitMiddleware(limiter ratelimit.RateLimiter) gin.HandlerFunc {
    return func(c *gin.Context) {
        clientIP := c.ClientIP()
        path := c.Request.URL.Path

        var ruleName string
        var resource string

        // 针对特定接口应用不同的规则和维度
        switch path {
        case "/api/v1/auth/login":
            ruleName = "login_attempt"
            resource = "ip:" + clientIP // 基于 IP 限流
        case "/api/v1/messages":
            userID, _ := getUserIDFromContext(c)
            ruleName = "send_message"
            resource = "user:" + userID // 基于用户ID限流
        default:
            // 其他接口使用默认规则，并基于 IP 限流
            ruleName = "api_default"
            resource = "ip:" + clientIP
        }

        allowed, err := limiter.Allow(c.Request.Context(), resource, ruleName)
        if err != nil {
            // 降级策略：如果限流器本身出错（如 Redis 连接失败），记录日志并暂时放行。
            clog.C(c.Request.Context()).Error("限流器检查失败", clog.Err(err))
            c.Next()
            return
        }

        if !allowed {
            c.AbortWithStatusJSON(http.StatusTooManyRequests, gin.H{"error": "too many requests"})
            return
        }

        c.Next()
    }
}
```

## 4. 配置管理

`ratelimit` 的所有规则都通过 `coord` 配置中心进行管理。运维人员只需修改对应路径下的 JSON 文件即可动态更新限流策略。

**规则路径**: 由 `Config.RulesPath` 决定，例如 `/config/dev/im-gateway/ratelimit/`。

**规则文件**: 在上述路径下，每个 `.json` 文件代表一条规则，**文件名即规则名**。

例如，要定义一条名为 `login_attempt` 的规则，只需在配置中心创建或修改文件 `/config/dev/im-gateway/ratelimit/login_attempt.json`：

```json
{
  "rate": 5,
  "capacity": 10,
  "description": "限制每个IP每秒最多5次登录尝试，允许10次突发。"
}
```

当此文件被创建、更新或删除时，`ratelimit` 实例会自动监听到变化，并在几秒内应用新的规则，无需重启任何服务。