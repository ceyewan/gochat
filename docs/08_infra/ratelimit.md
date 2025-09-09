# 基础设施: RateLimit 分布式限流

## 1. 设计理念

`ratelimit` 组件提供了一个**统一的限流接口**，封装了两种不同场景的限流实现：

- **分布式限流**: 基于 Redis 的令牌桶算法，适用于多实例部署的微服务环境，确保集群级别的流量控制。
- **单机限流**: 基于 Go 官方 `golang.org/x/time/rate` 库的内存限流，适用于单实例部署或不需要跨实例协调的场景，性能更高。

`ratelimit` 组件遵循 `im-infra` 的核心规范：

- **统一接口**: 通过 Provider 模式提供一致的限流体验，业务代码无需关心底层实现
- **声明式配置**: 限流规则通过配置中心进行管理，支持动态热更新
- **组件自治**: 内部自动监听配置变化，实现配置的热加载
- **多维度防护**: 支持基于用户ID、IP地址、API端点等多种维度的限流。

`ratelimit` 组件的设计遵循 **KISS (Keep It Simple, Stupid)** 和 **高内聚** 的原则，旨在提供一个极简、可靠且对开发者友好的分布式限流解决方案。

- **极简 API**: 组件只暴露一个核心方法 `Allow`，保证接口的纯粹性和易用性。
- **声明式配置**: 限流规则的管理完全通过配置中心（由 `coord` 提供支持）进行。运维人员通过修改配置文件来“声明”期望的状态，而不是通过 API “命令”组件修改规则。这更符合现代的 GitOps 理念。
- **组件自治的动态配置**: `ratelimit` 组件内部自己负责监听其在配置中心的规则变化。它直接使用 `coord` 提供的 `Watch` 功能，实现了“组件自治”的热更新，无需一个复杂的、全局的配置分发框架。这种模式保证了 `coord` 的简洁性和 `ratelimit` 的高内聚性。
- **平滑与突发兼顾**: 采用令牌桶算法，允许在平均速率限制之下，处理一定量的突发流量（由桶的容量决定），这比简单的计数器算法更能适应真实世界的流量模式。
- **多维度防护**: 组件的设计支持基于多种维度的限流，如用户ID、IP地址、API端点等，通过构建不同的 `resource` 键来实现。

## 2. 核心 API 契约

### 2.1 构造函数

```go
// Config 是 ratelimit 组件的配置结构体。
type Config struct {
	// Mode 限流模式，支持 "distributed" 和 "local" 两种模式
	Mode string `json:"mode"`
	
	// ServiceName 用于日志记录和监控，以区分是哪个服务在使用限流器
	ServiceName string `json:"serviceName"`
	
	// RulesPath 是在 coord 配置中心存储此服务限流规则的根路径
	// 约定：此路径必须以 "/" 结尾
	// 例如："/config/dev/im-gateway/ratelimit/"
	RulesPath string `json:"rulesPath"`
	
}

// GetDefaultConfig 返回默认的 ratelimit 配置。
// 开发环境：使用单机模式，无 Redis 依赖。
// 生产环境：使用分布式模式，依赖 cache.Provider。
func GetDefaultConfig(env string) *Config

// Option 定义了用于定制 ratelimit Provider 的函数。
type Option func(*options)

// WithLogger 将一个 clog.Logger 实例注入 ratelimit，用于记录内部日志。
func WithLogger(logger clog.Logger) Option

// WithCoordProvider 注入 coord.Provider，用于从配置中心获取动态配置。
// 此选项是必需的，如果不提供将返回错误。
func WithCoordProvider(provider coord.Provider) Option

// WithCacheProvider 注入 cache.Provider，用于分布式模式的 Redis 操作。
// 如果 config.Mode 为 "distributed"，此选项是必需的。
func WithCacheProvider(provider cache.Provider) Option

// New 创建一个新的 ratelimit Provider 实例。
// 这是与 ratelimit 组件交互的唯一入口。
func New(ctx context.Context, config *Config, opts ...Option) (Provider, error)
```

### 2.2 Provider 接口

```go
// Provider 定义了 ratelimit 组件提供的所有能力。
type Provider interface {
	// Allow 检查给定资源的单个请求是否被允许。
	// resource 是被限流的唯一标识，如 "user:123" 或 "ip:1.2.3.4"。
	// ruleName 是要应用的规则名，如 "api_default"。如果规则不存在，将采用失败策略（默认为拒绝）。
	Allow(ctx context.Context, resource, ruleName string) (bool, error)

	// Close 关闭限流器，释放后台协程和连接等资源。
	Close() error
}
```

## 3. 标准用法

### 场景 1: 在服务启动时初始化 ratelimit Provider

```go
// 在 main.go 中
func main() {
    // ... 首先初始化 clog、coord 和 cache ...
    
    // 1. 获取并覆盖配置
    // GetDefaultConfig 会根据环境（"production" 或 "development"）返回推荐配置
    config := ratelimit.GetDefaultConfig("production")
    config.ServiceName = "message-service"
    config.RulesPath = "/config/prod/message-service/ratelimit/"
    
    // 2. 准备 Options
    // New 函数内部会根据 config.Mode 决定是否使用 cacheProvider
    opts := []ratelimit.Option{
        ratelimit.WithLogger(clog.Namespace("ratelimit")),
        ratelimit.WithCoordProvider(coordProvider),
        ratelimit.WithCacheProvider(cacheProvider), // 分布式模式依赖 cache 组件
    }

    // 3. 创建 Provider
    // 初始化逻辑被封装在 New 函数中，调用者无需关心具体模式
    rateLimitProvider, err := ratelimit.New(context.Background(), config, opts...)
    if err != nil {
        clog.Fatal("初始化 ratelimit 失败", clog.Err(err))
    }
    defer rateLimitProvider.Close()
    
    clog.Info("ratelimit Provider 初始化成功", clog.String("mode", config.Mode))
}
```

### 场景 2: 双重限流保护 (单机 + 分布式)

```go
import "github.com/ceyewan/gochat/im-infra/ratelimit"

// MessageService 使用同一个 Provider 进行双重限流
type MessageService struct {
    rateLimit ratelimit.Provider
}

func (s *MessageService) SendMessage(ctx context.Context, userID, content string) error {
    logger := clog.WithContext(ctx)
    
    // 1. 先检查单机限流 - 保护本机资源（QPS=500）
    localAllowed, err := s.rateLimit.Allow(ctx, "machine", "local_machine_qps")
    if err != nil {
        logger.Error("单机限流检查失败", clog.Err(err))
    }
    if !localAllowed {
        return fmt.Errorf("单机QPS限制：请求被拒绝")
    }
    
    // 2. 再检查分布式限流 - 控制用户行为（每秒30条消息）
    userAllowed, err := s.rateLimit.Allow(ctx, "user:"+userID, "user_send_message")
    if err != nil {
        logger.Error("用户限流检查失败", clog.Err(err))
        // 降级策略：分布式限流异常时仍然执行单机限流
    }
    if !userAllowed {
        return fmt.Errorf("用户发送频率限制：请求被拒绝")
    }
    
    // 3. 执行业务逻辑
    logger.Info("消息发送通过限流检查", 
        clog.String("user_id", userID),
        clog.String("content", content))
    
    return s.doSendMessage(ctx, userID, content)
}
```

### 场景 3: 在 Gin 中间件中保护 API

```go
import "github.com/ceyewan/gochat/im-infra/ratelimit"

// RateLimitMiddleware 创建一个 Gin 中间件用于 API 限流。
func RateLimitMiddleware(limiter ratelimit.Provider) gin.HandlerFunc {
    return func(c *gin.Context) {
        clientIP := c.ClientIP()
        path := c.Request.URL.Path

        var ruleName string
        var resource string

        // 针对特定接口应用不同的规则和维度
        switch path {
        case "/api/v1/auth/login":
            ruleName = "user_login_attempt"  // 使用分布式限流
            resource = "ip:" + clientIP      // 基于 IP 限流
        case "/api/v1/messages":
            userID, _ := getUserIDFromContext(c)
            ruleName = "user_send_message"   // 使用分布式限流
            resource = "user:" + userID      // 基于用户ID限流
        default:
            // 其他接口使用单机默认规则
            ruleName = "api_default"         // 使用单机限流
            resource = "ip:" + clientIP
        }

        allowed, err := limiter.Allow(c.Request.Context(), resource, ruleName)
        if err != nil {
            // 降级策略：如果限流器本身出错，记录日志并暂时放行
            clog.WithContext(c.Request.Context()).Error("限流器检查失败", clog.Err(err))
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

func getUserIDFromContext(c *gin.Context) (string, bool) {
    userID, exists := c.Get("user_id")
    if !exists {
        return "", false
    }
    return userID.(string), true
}

```

## 4. 配置管理

`ratelimit` 的所有规则都通过 `coord` 配置中心进行管理。运维人员只需修改对应路径下的 JSON 文件即可动态更新限流策略。

**规则路径**: 由 `Config.RulesPath` 决定，例如 `/config/prod/message-service/ratelimit/`。

**规则文件**: 在上述路径下，每个 `.json` 文件代表一条规则，**文件名即规则名**。

### 4.1 规则配置示例

**单一服务的完整限流配置** (`rules.json`):
```json
{
  "local_machine_qps": {
    "mode": "local",
    "rate": 500.0,
    "capacity": 600,
    "description": "单机QPS限制：每秒500请求，允许600突发"
  },
  "user_send_message": {
    "mode": "distributed",
    "rate": 30.0,
    "capacity": 60,
    "description": "用户发送消息：每秒30条，允许60条突发（集群级别）"
  },
  "user_login_attempt": {
    "mode": "distributed",
    "rate": 5.0,
    "capacity": 10,
    "description": "用户登录尝试：每秒5次，允许10次突发（集群级别）"
  },
  "api_default": {
    "mode": "local",
    "rate": 100.0,
    "capacity": 200,
    "description": "API默认限流：每秒100请求，允许200次突发（单机）"
  }
}
```

**字段说明**:
- `mode`: 限流模式，`"local"`（单机）或 `"distributed"`（分布式）
- `rate`: 令牌生成速率（每秒允许的请求数），支持小数
- `capacity`: 令牌桶容量（允许的最大突发请求数）
- `description`: 规则描述，用于文档和监控

### 4.2 智能模式选择

**关键设计**：每个规则都可以独立指定其限流模式（`mode` 字段），这样在同一个 Provider 中可以同时使用单机和分布式限流：

- **`mode: "local"`**: 使用 Go 官方 `rate` 库，在当前实例内限流，保护本机资源
- **`mode: "distributed"`**: 使用 Redis 分布式算法，在整个集群内限流，控制业务流量

**使用示例**:
```go
// 先检查单机限流 - 保护本机资源
allowed, err := limiter.Allow(ctx, "machine", "local_machine_qps")

// 再检查分布式限流 - 控制用户行为
allowed, err := limiter.Allow(ctx, "user:123", "user_send_message")
```

### 4.3 配置文件的灵活组织

您也可以根据需要将规则分组到不同的文件中：

**按模式分组** - `local_rules.json` 和`distributed_rules.json`：
```json
// local_rules.json
{
  "machine_qps": {"mode": "local", "rate": 500.0, "capacity": 600},
  "api_default": {"mode": "local", "rate": 100.0, "capacity": 200}
}

// distributed_rules.json  
{
  "user_send_message": {"mode": "distributed", "rate": 30.0, "capacity": 60},
  "user_login": {"mode": "distributed", "rate": 5.0, "capacity": 10}
}
```

**按业务分组** - `user_rules.json` 和 `system_rules.json`：
```json
// user_rules.json
{
  "send_message": {"mode": "distributed", "rate": 30.0, "capacity": 60},
  "login_attempt": {"mode": "distributed", "rate": 5.0, "capacity": 10}
}

// system_rules.json
{
  "machine_qps": {"mode": "local", "rate": 500.0, "capacity": 600},
  "health_check": {"mode": "local", "rate": 1000.0, "capacity": 1000}
}
```

当这些文件被创建、更新或删除时，`ratelimit` 实例会自动监听到变化，并在几秒内应用新的规则，无需重启任何服务。

## 5. 设计注记

### 5.1 GetDefaultConfig 默认值说明

`GetDefaultConfig` 根据环境返回优化的默认配置：

**开发环境 (development)**:
```go
&Config{
    Mode:        "local",              // 使用单机模式，无 Redis 依赖
    ServiceName: "",                   // 需要用户设置
    RulesPath:   "/config/dev/ratelimit/",
}
```

**生产环境 (production)**:
```go
&Config{
    Mode:        "distributed",       // 使用分布式模式，依赖 cache.Provider
    ServiceName: "",                  // 需要用户设置
    RulesPath:   "/config/prod/ratelimit/",
}
```

用户仍需要根据实际部署环境覆盖 `ServiceName` 等关键配置。

### 5.2 智能模式路由

**核心创新**：`ratelimit` Provider 会根据每个规则的 `mode` 配置自动选择合适的底层实现：

- **当 `mode: "local"`** 时，使用 Go 官方 `golang.org/x/time/rate` 库进行单机限流
- **当 `mode: "distributed"`** 时，使用基于 Redis 的分布式令牌桶算法

**使用体验**：业务代码完全无需关心底层实现，通过同一个 `Allow` 方法就能享受到不同模式的限流保护：

```go
// 这个调用会使用单机限流（保护本机资源）
allowed, _ := provider.Allow(ctx, "machine", "local_machine_qps")

// 这个调用会使用分布式限流（控制集群业务流量）  
allowed, _ := provider.Allow(ctx, "user:123", "user_send_message")
```

### 5.3 令牌桶算法实现

**分布式模式**: 基于 Redis 的原子操作和 Lua 脚本实现，确保多实例间的状态一致性
**单机模式**: 基于 Go 官方 `golang.org/x/time/rate` 库，提供高性能的内存限流

两种模式都支持：
- **平滑限流**: 基于令牌生成速率的平均限制
- **突发处理**: 通过桶容量允许短时间的流量突发
- **动态配置**: 运行时热更新规则，无需重启服务

### 5.4 错误处理和降级策略

**限流器异常处理**:
- Redis 连接失败：记录错误，可配置为放行或拒绝
- 配置加载失败：使用默认规则或拒绝请求
- 规则不存在：默认拒绝策略，确保系统安全

**建议的降级策略**:
- 对于关键业务接口：限流器异常时可以临时放行
- 对于非关键接口：限流器异常时可以拒绝请求
- 实现熔断机制：持续异常时自动切换到单机模式