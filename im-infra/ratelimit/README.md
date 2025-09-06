# RateLimit - 分布式限流组件

`ratelimit` 是一个基于令牌桶算法的高性能分布式限流组件，专为 GoChat 系统设计。支持动态配置、多维度限流、批量操作和实时统计。

## 🚀 核心特性

- **高性能**: 基于 cache 抽象层和 Lua 脚本实现原子操作，支持高并发场景
- **分布式**: 天然支持分布式架构，适用于微服务集群  
- **动态配置**: 与 coord 组件集成，支持实时调整限流规则
- **多维度**: 支持基于用户、IP、API、设备等多维度的限流策略
- **易扩展**: 模块化设计，支持自定义限流算法和存储后端
- **可观测**: 内置统计信息和监控指标，便于运维管理

## 📦 快速开始

### 安装

```bash
go get github.com/ceyewan/gochat/im-infra/ratelimit
```

### 基本用法

```go
package main

import (
    "context"
    "fmt"
    "log"
    "time"

    "github.com/ceyewan/gochat/im-infra/ratelimit"
)

func main() {
    ctx := context.Background()
    
    // 定义默认规则（当配置中心不可用时使用）
    defaultRules := map[string]ratelimit.Rule{
        "api_requests": {Rate: 100, Capacity: 200},   // 每秒 100 个请求，突发 200
        "user_actions": {Rate: 10, Capacity: 20},     // 每秒 10 个操作，突发 20
        "login":        {Rate: 5, Capacity: 10},      // 每秒 5 次登录，突发 10
    }

    // 创建限流器
    limiter, err := ratelimit.New(
        ctx,
        "my-service", // 服务名称，用于配置中心路径
        ratelimit.WithDefaultRules(defaultRules),
    )
    if err != nil {
        log.Fatal(err)
    }
    defer limiter.Close()

    // 检查单个请求是否被允许
    userID := "user123"
    resource := ratelimit.BuildUserResourceKey(userID)
    
    allowed, err := limiter.Allow(ctx, resource, "api_requests")
    if err != nil {
        log.Printf("限流检查失败: %v", err)
        return
    }
    
    if allowed {
        fmt.Println("✅ 请求被允许")
        // 处理请求
    } else {
        fmt.Println("❌ 请求被限流")
        // 返回限流错误
    }
}
```

### 高级用法

#### 批量限流检查

```go
// 批量限流请求
requests := []ratelimit.RateLimitRequest{
    {Resource: "user:123", RuleName: "api_requests", Count: 1},
    {Resource: "user:456", RuleName: "user_actions", Count: 2},
    {Resource: "ip:192.168.1.1", RuleName: "login", Count: 1},
}

results, err := limiter.BatchAllow(ctx, requests)
if err != nil {
    log.Fatal(err)
}

for i, allowed := range results {
    fmt.Printf("请求 %d: %v\n", i+1, allowed)
}
```

#### 多令牌消费

```go
// 一次性消费多个令牌
allowed, err := limiter.AllowN(ctx, "user:789", "api_requests", 5)
if err != nil {
    log.Fatal(err)
}
```

#### 自定义配置

```go
import (
    "github.com/ceyewan/gochat/im-infra/cache"
    "github.com/ceyewan/gochat/im-infra/coord"
)

// 自定义缓存配置
cacheClient, err := cache.New(ctx, cache.Config{
    Addr: "redis://localhost:6379",
    DB:   1,
})
if err != nil {
    log.Fatal(err)
}

// 自定义协调客户端
coordClient, err := coord.New(ctx, coord.CoordinatorConfig{
    Endpoints: []string{"localhost:2379"},
})
if err != nil {
    log.Fatal(err)
}

// 使用自定义客户端创建限流器
limiter, err := ratelimit.New(
    ctx,
    "my-service",
    ratelimit.WithCacheClient(cacheClient),
    ratelimit.WithCoordinationClient(coordClient),
    ratelimit.WithDefaultRules(defaultRules),
    ratelimit.WithRuleRefreshInterval(30*time.Second),
    ratelimit.WithFailurePolicy(ratelimit.FailurePolicyAllow),
)
```

#### 管理功能

```go
// 使用管理器版本获得更多功能
manager, err := ratelimit.NewManager(
    ctx,
    "my-service",
    ratelimit.WithDefaultRules(defaultRules),
)
if err != nil {
    log.Fatal(err)
}
defer manager.Close()

// 动态设置规则
newRule := ratelimit.Rule{Rate: 50, Capacity: 100}
err = manager.SetRule(ctx, "new_rule", newRule)
if err != nil {
    log.Printf("设置规则失败: %v", err)
}

// 列出所有规则
rules := manager.ListRules()
for name, rule := range rules {
    fmt.Printf("规则 %s: 速率=%.2f, 容量=%d\n", name, rule.Rate, rule.Capacity)
}

// 导出规则到配置中心
err = manager.ExportRules(ctx)
if err != nil {
    log.Printf("导出规则失败: %v", err)
}
```

## ⚙️ 配置

### 限流规则格式

限流规则存储在配置中心，路径格式：
```
/config/{环境}/{服务}/ratelimit/{规则名}
```

规则 JSON 格式：
```json
{
  "rate": 10.0,        // 令牌产生速率 (tokens/second)
  "capacity": 20,      // 桶容量 (最大突发流量)
  "description": "API限流规则"
}
```

### 预定义规则场景

组件提供了多种预定义的限流场景：

```go
// 获取推荐规则
rule, exists := ratelimit.GetRuleByScenario("web_api_high")
if exists {
    fmt.Printf("高频 API 推荐规则: %+v\n", rule)
}

// 创建默认规则集合
defaultRules := ratelimit.CreateDefaultRules()
```

### 配置选项

```go
limiter, err := ratelimit.New(
    ctx,
    "my-service",
    // 基础配置
    ratelimit.WithCacheClient(cacheClient),           // 自定义缓存客户端
    ratelimit.WithCoordinationClient(coordClient),    // 自定义协调客户端
    ratelimit.WithDefaultRules(rules),                // 默认规则
    
    // 行为配置
    ratelimit.WithRuleRefreshInterval(30*time.Second), // 规则刷新间隔
    ratelimit.WithFailurePolicy(ratelimit.FailurePolicyAllow), // 失败策略
    ratelimit.WithBatchSize(100),                      // 批处理大小
    
    // 功能开关
    ratelimit.WithMetricsEnabled(true),               // 启用指标收集
    ratelimit.WithStatisticsEnabled(true),            // 启用统计功能
    ratelimit.WithScriptCacheEnabled(true),           // 启用脚本缓存
    
    // 高级配置
    ratelimit.WithKeyPrefix("custom_ratelimit"),      // 自定义键前缀
    ratelimit.WithDefaultTTL(24*time.Hour),          // 默认过期时间
    ratelimit.WithMaxRetries(3),                      // 最大重试次数
    ratelimit.WithRetryDelay(100*time.Millisecond),  // 重试延迟
)
```

## 📊 监控与统计

### 统计信息

```go
// 获取限流统计信息
stats, err := limiter.GetStatistics(ctx, "user:123", "api_requests")
if err != nil {
    log.Fatal(err)
}

fmt.Printf("统计信息:\n")
fmt.Printf("  总请求数: %d\n", stats.TotalRequests)
fmt.Printf("  允许请求数: %d\n", stats.AllowedRequests)
fmt.Printf("  拒绝请求数: %d\n", stats.DeniedRequests)
fmt.Printf("  当前令牌数: %d\n", stats.CurrentTokens)
fmt.Printf("  成功率: %.2f%%\n", stats.SuccessRate*100)
fmt.Printf("  最后更新: %s\n", stats.LastUpdated.Format("2006-01-02 15:04:05"))
```

### 错误处理

```go
allowed, err := limiter.Allow(ctx, resource, ruleName)
if err != nil {
    // 使用预定义的错误检查函数
    if ratelimit.IsRateLimited(err) {
        // 处理限流错误
        fmt.Println("请求被限流")
    } else if ratelimit.IsCacheError(err) {
        // 处理缓存错误
        fmt.Println("缓存服务异常")
    } else if ratelimit.IsConfigError(err) {
        // 处理配置错误
        fmt.Println("配置中心异常")
    } else {
        // 其他错误
        fmt.Printf("未知错误: %v\n", err)
    }
}
```

## 🔧 资源键构建

组件提供了便利的资源键构建函数：

```go
// 基本资源键
key := ratelimit.BuildResourceKey("order", "12345")         // "order:12345"

// 用户相关
userKey := ratelimit.BuildUserResourceKey("user123")       // "user:user123"

// IP 相关  
ipKey := ratelimit.BuildIPResourceKey("192.168.1.1")       // "ip:192.168.1.1"

// API 相关
apiKey := ratelimit.BuildAPIResourceKey("/api/users")      // "api:/api/users"

// 设备相关
deviceKey := ratelimit.BuildDeviceResourceKey("mobile123") // "device:mobile123"
```

## 🎯 使用场景

### Web API 限流

```go
// 不同级别的 API 限流
rules := map[string]ratelimit.Rule{
    "api_public":   {Rate: 1000, Capacity: 2000}, // 公开 API
    "api_private":  {Rate: 100, Capacity: 200},   // 私有 API
    "api_admin":    {Rate: 50, Capacity: 100},    // 管理 API
}

// 按 API 端点限流
resource := ratelimit.BuildAPIResourceKey(r.URL.Path)
allowed, _ := limiter.Allow(ctx, resource, "api_public")
```

### 用户行为限流

```go
// 不同用户操作的限流
rules := map[string]ratelimit.Rule{
    "user_read":      {Rate: 100, Capacity: 200}, // 读操作
    "user_write":     {Rate: 10, Capacity: 20},   // 写操作
    "user_sensitive": {Rate: 1, Capacity: 3},     // 敏感操作
}

// 按用户ID限流
resource := ratelimit.BuildUserResourceKey(userID)
allowed, _ := limiter.Allow(ctx, resource, "user_write")
```

### 安全防护

```go
// 安全相关限流
rules := map[string]ratelimit.Rule{
    "login_attempt":  {Rate: 5, Capacity: 10},    // 登录尝试
    "password_reset": {Rate: 0.1, Capacity: 1},   // 密码重置
    "captcha_verify": {Rate: 10, Capacity: 20},   // 验证码验证
}

// 按IP限流
resource := ratelimit.BuildIPResourceKey(clientIP)
allowed, _ := limiter.Allow(ctx, resource, "login_attempt")
```

### 资源保护

```go
// 资源密集型操作限流
rules := map[string]ratelimit.Rule{
    "file_upload":    {Rate: 2, Capacity: 5},     // 文件上传
    "report_generate": {Rate: 0.5, Capacity: 2},  // 报表生成
    "export_data":    {Rate: 0.1, Capacity: 1},   // 数据导出
}
```

## 🧪 测试

### 运行测试

```bash
# 运行所有测试
go test ./...

# 运行带覆盖率的测试
go test -cover ./...

# 运行性能测试
go test -bench=. -benchmem ./...
```

### 集成测试

组件需要 Redis 来运行集成测试：

```bash
# 启动 Redis（使用 Docker）
docker run -d -p 6379:6379 redis:alpine

# 运行集成测试
go test -tags=integration ./...
```

### 示例程序

```bash
# 运行基本示例
cd examples/basic
go run main.go

# 运行高级示例
cd examples/advanced  
go run main.go
```

## 📚 架构设计

### 核心组件

- **RateLimiter**: 限流器主接口，提供基本限流功能
- **TokenBucket**: 令牌桶算法实现，基于 Lua 脚本
- **ConfigManager**: 配置管理器，支持动态配置更新
- **Statistics**: 统计信息收集器
- **Cache Layer**: 缓存抽象层，支持不同的缓存后端

### 设计原则

1. **接口优先**: 通过接口抽象核心功能，便于测试和扩展
2. **配置分离**: 支持多种配置来源，降低耦合度
3. **优雅降级**: 组件异常时采用安全的默认策略
4. **可观测性**: 内置统计和监控功能
5. **高性能**: 使用原子操作和批量处理优化性能

## 🤝 贡献指南

欢迎提交 Issue 和 Pull Request 来改进此组件。

### 开发环境

```bash
# 启动依赖服务
make dev

# 安装开发工具
make install-tools

# 运行测试
make test

# 代码格式化
make fmt

# 代码检查
make lint
```

## 📄 许可证

MIT License - 详见项目根目录的 LICENSE 文件