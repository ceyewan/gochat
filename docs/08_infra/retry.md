# 基础设施: retry 优雅重试

## 1. 设计理念

`retry` 是一个用于处理临时性、可恢复错误的统一重试组件。在分布式系统中，网络抖动、服务临时过载等瞬时故障是常态而非例外。`retry` 组件旨在通过一个优雅、统一的机制，自动处理这些故障，从而极大提升系统的韧性 (Resilience) 和健壮性。

其核心设计理念是：
- **非侵入式与简洁性**: 开发者只需将可能失败的操作封装在一个函数中，并调用统一的 `retry.Do()` 方法即可。所有复杂的重试逻辑（如退避、抖动、错误判断）都由组件在内部处理，保持业务代码的干净和聚焦。
- **策略驱动**: 重试行为完全由一个可配置的 `Policy` 对象定义，而不是硬编码在代码中。这使得重试逻辑可以被轻松地调整、复用和测试。
- **智能错误处理**: 组件能够区分“可重试的错误”（如网络超时）和“不可重试的错误”（如参数错误），避免对确定性失败进行无效的重试。
- **上下文感知**: 整个重试过程严格尊重 `context.Context` 的超时或取消信号，一旦上游请求被取消，所有重试将立即停止，避免资源浪费。

## 2. 核心 API 契约

`retry` 组件被设计为一个无状态的工具包，不需实例化，直接通过包级函数调用。

### 2.1 核心入口

```go
package retry

// Do 执行一个操作，并根据提供的策略进行重试。
// 这是与 retry 组件交互的唯一入口。
//
// ctx: 控制整个重试周期的上下文。一旦 ctx.Done()，所有重试将立即停止。
// policy: 定义重试行为的策略对象。
// op: 需要被执行和可能需要重试的操作。
func Do(ctx context.Context, policy Policy, op Operation) error
```

### 2.2 核心数据结构

```go
// Operation 是需要被执行和可能需要重试的操作。
// 它接收一个上下文，该上下文可能带有单次尝试的超时。
type Operation func(ctx context.Context) error

// Policy 定义了完整的重试策略。
type Policy struct {
    // MaxAttempts 是最大尝试次数（包括首次尝试）。
    // 例如，设置为 3 表示最多执行 1 次初始尝试 + 2 次重试。
    MaxAttempts int
    
    // Backoff 是两次重试之间的退避策略，决定了等待多长时间再进行下一次尝试。
    Backoff BackoffStrategy
    
    // Jitter 为退避时间增加随机性，以防止“惊群效应”。此为可选配置。
    Jitter JitterStrategy
    
    // IsRetryable 是一个回调函数，用于判断一个错误是否是可重试的。
    // 如果为 nil，则默认所有错误都可重试。这是进行智能错误分类的关键。
    IsRetryable func(err error) bool
}

// BackoffStrategy 定义了两次重试之间的等待时延策略。
type BackoffStrategy interface {
    NextDelay(attempt int) time.Duration
}
```

### 2.3 内置策略

`retry` 组件提供了一系列开箱即用的策略，以简化 `Policy` 的创建。

```go
// --- Backoff Strategies ---

// Fixed 返回一个固定时延的退避策略。
func Fixed(delay time.Duration) BackoffStrategy

// Exponential 返回一个指数退避策略。
// 每次重试的延迟时间是前一次的 factor 倍。
func Exponential(initialDelay time.Duration, factor float64) BackoffStrategy

// --- Jitter Strategies ---

// Full 返回一个全抖动策略 (在 [0, baseDelay] 之间取随机值)。
func Full() JitterStrategy

// Proportional 返回一个按比例抖动的策略 (在 baseDelay ± factor 之间取随机值)。
func Proportional(factor float64) JitterStrategy

// --- Error Classification ---

// IsNetworkError 判断一个错误是否是典型的网络错误 (如 net.Error, io.EOF)。
func IsNetworkError(err error) bool

// IsGRPCCodeRetryable 判断一个 gRPC 错误码是否是可重试的 (如 codes.Unavailable)。
func IsGRPCCodeRetryable(err error, codes ...codes.Code) bool
```

## 3. 标准用法

### 场景：调用一个可能超时的 gRPC 服务

```go
import (
    "context"
    "time"
    "google.golang.org/grpc/codes"
    "github.com/ceyewan/gochat/im-infra/retry"
)

func (c *UserInfoClient) GetProfile(ctx context.Context, userID string) (*Profile, error) {
    var profile *Profile
    
    // 1. 定义一个清晰、可复用的重试策略
    grpcRetryPolicy := retry.Policy{
        MaxAttempts: 3,
        // 初始延迟 100ms，每次延迟翻倍
        Backoff: retry.Exponential(100*time.Millisecond, 2.0),
        // 增加 ±30% 的随机抖动
        Jitter: retry.Proportional(0.3),
        // 定义只有 gRPC 的 Unavailable 和 DeadlineExceeded 错误才进行重试
        IsRetryable: func(err error) bool {
            return retry.IsGRPCCodeRetryable(err, codes.Unavailable, codes.DeadlineExceeded)
        },
    }
    
    // 2. 定义要执行的操作
    getUserInfoOp := func(opCtx context.Context) error {
        // opCtx 是由 retry.Do 传入的、可能带有单次超时控制的上下文
        resp, err := c.grpcClient.GetProfile(opCtx, &pb.GetProfileRequest{UserId: userID})
        if err != nil {
            return err // 将错误返回给 retry.Do，由 IsRetryable 判断是否重试
        }
        profile = convert(resp) // 如果成功，将结果赋给闭包外的变量
        return nil
    }
    
    // 3. 用一行代码执行带重试逻辑的操作
    if err := retry.Do(ctx, grpcRetryPolicy, getUserInfoOp); err != nil {
        // 如果所有重试都失败了，这里会返回最后一次的错误
        clog.C(ctx).Error("获取用户信息失败，已重试3次", clog.Err(err))
        return nil, err
    }
    
    return profile, nil
}