# 基础设施: breaker 熔断器

## 1. 设计理念

`breaker` 是 `gochat` 项目中用于实现服务保护、防止雪崩效应的核心组件。

其核心设计理念是：
- **防止雪崩，快速失败**: 当一个下游服务持续失败时，熔断器会“跳闸”（进入 Open 状态），在一段时间内阻止所有对该服务的进一步调用。这既保护了下游服务，也避免了上游服务因无谓的等待和重试而耗尽资源。
- **自动恢复探测**: 熔断器具备自动恢复能力。在跳闸一段时间后，它会进入“半开”状态，小心翼翼地放行少量请求去探测下游服务是否已恢复。如果探测成功，则完全关闭熔断器，恢复正常流量。
- **独立实例，统一管理**: 每个需要保护的“资源”（例如，对“用户服务gRPC”的调用）都应该有一个独立的熔断器实例，拥有自己独立的状态。我们通过 `Provider` 模式来统一管理这些实例及其配置。
- **封装与适配**: 组件内部基于成熟、轻量的 `sony/gobreaker` 库实现，我们将其所有细节封装起来，对外只暴露符合 `im-infra` 核心规范的、简洁的 API。

## 2. 核心 API 契约

`breaker` 组件是**有状态的**，因此需要通过构造函数创建 `Provider` 来管理。

### 2.1 构造函数与配置

```go
// Policy 定义了熔断器的行为策略。
type Policy struct {
    // FailureThreshold 是触发跳闸的连续失败次数阈值。
    FailureThreshold int `json:"failureThreshold"`
    // SuccessThreshold 是在【半开】状态下，需要多少次连续成功才能关闭熔断器。
    SuccessThreshold int `json:"successThreshold"`
    // OpenStateTimeout 是熔断器在【打开】状态下持续的时间。
    // 经过这段时间后，熔断器会转为【半开】状态。
    OpenStateTimeout time.Duration `json:"openStateTimeout"`
}

// Config 是 breaker 组件的配置结构体。
type Config struct {
    // ServiceName 用于日志记录和监控。
    ServiceName string `json:"serviceName"`
    // PoliciesPath 是在 coord 配置中心存储此服务熔断器策略的根路径。
    // 例如："/config/dev/im-gateway/breakers/"
    PoliciesPath string `json:"policiesPath"`
}

// Provider 是熔断器组件的提供者，负责创建和管理多个熔断器实例。
type Provider interface {
    // GetBreaker 获取或创建一个指定名称的熔断器实例。
    // name 是被保护资源的唯一标识，例如 "grpc:user-service" 或 "http:payment-api"。
    // 如果配置中心没有该名称的策略，会使用默认策略。
    GetBreaker(name string) Breaker
    
    // Close 关闭 Provider，停止所有后台任务。
    Close() error
}

// GetDefaultConfig 返回一个推荐的默认配置。
// serviceName 会被用作 Config.ServiceName，并用于构建默认的 PoliciesPath。
// 生产环境 ("production") 和开发环境 ("development") 会有不同的熔断策略。
func GetDefaultConfig(serviceName, env string) *Config

// Option 是用于配置 breaker Provider 的函数式选项。
type Option func(*providerOptions)

// WithLogger 为 breaker Provider 设置一个 clog.Logger 实例。
func WithLogger(logger clog.Logger) Option

// WithCoordProvider 为 breaker Provider 设置一个 coord.Provider 实例。
// 这是动态加载和更新熔断策略所必需的。
func WithCoordProvider(coordProvider coord.Provider) Option

// New 创建一个新的熔断器 Provider。
// 它会自动从 coord 加载所有策略，并监听后续变更。
func New(ctx context.Context, config *Config, opts ...Option) (Provider, error)
```

### 2.2 Breaker 接口

```go
// Breaker 是熔断器的主接口。
type Breaker interface {
    // Do 执行一个受熔断器保护的操作。
    // 如果熔断器是【打开】状态，此方法会立即返回 ErrBreakerOpen 错误，op 不会被执行。
    // 否则，执行 op。如果 op 失败，则增加失败计数并可能触发跳闸。
    Do(ctx context.Context, op func() error) error
}

// ErrBreakerOpen 是当熔断器处于打开状态时，Do 方法返回的标准错误。
var ErrBreakerOpen = errors.New("circuit breaker is open")
```

## 3. 标准用法

### 场景 1: 在 gRPC 客户端拦截器中集成

这是最理想的集成方式，可以对所有出站 gRPC 调用实现无侵入的保护。

```go
// 1. 在服务启动时初始化 Breaker Provider
// 推荐使用 GetDefaultConfig 获取标准配置，然后按需覆盖
config := breaker.GetDefaultConfig("im-gateway", "production")
// config.PoliciesPath = "/custom/path/if/needed" // 按需覆盖

// 创建 Provider 实例，并通过 With... Options 注入依赖
breakerProvider, err := breaker.New(context.Background(), config,
    breaker.WithLogger(clog.Namespace("breaker")),
    breaker.WithCoordProvider(coordProvider), // 依赖 coord 组件
)
if err != nil {
    log.Fatal(err)
}
defer breakerProvider.Close()

// 2. 创建一个 gRPC 客户端拦截器
func BreakerClientInterceptor(provider breaker.Provider) grpc.UnaryClientInterceptor {
    return func(ctx context.Context, method string, req, reply interface{}, cc *grpc.ClientConn, invoker grpc.UnaryInvoker, opts ...grpc.CallOption) error {
        // 使用 gRPC 的方法名作为熔断器的名称，为每个方法创建独立的熔断器
        b := provider.GetBreaker(method)
        
        // 将真正的 gRPC 调用包裹在熔断器的 Do 方法中
        err := b.Do(ctx, func() error {
            return invoker(ctx, method, req, reply, cc, opts...)
        })

        // 可选：将熔断器错误转换为标准的 gRPC 错误码
        if errors.Is(err, breaker.ErrBreakerOpen) {
            return status.Error(codes.Unavailable, err.Error())
        }
        return err
    }
}

// 3. 创建 gRPC 客户端连接时使用拦截器
conn, err := grpc.Dial(
    "target-service",
    grpc.WithUnaryInterceptor(BreakerClientInterceptor(breakerProvider)),
)
```


## 4. 配置管理

与 `ratelimit` 类似，`breaker` 的所有策略都通过 `coord` 配置中心进行管理。

**策略路径**: 由 `Config.PoliciesPath` 决定，例如 `/config/dev/im-gateway/breakers/`。

**策略文件**: 在上述路径下，每个 `.json` 文件代表一条策略，**文件名即熔断器名称**。

例如，要为名为 `grpc:user-service` 的熔断器定义策略，只需创建文件 `/config/dev/im-gateway/breakers/grpc:user-service.json`：

```json
{
  "failureThreshold": 5,
  "successThreshold": 2,
  "openStateTimeout": "1m"
}