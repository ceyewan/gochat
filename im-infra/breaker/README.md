# Breaker 熔断器组件

Breaker 是 GoChat 项目中用于实现服务保护、防止雪崩效应的核心组件。

## 特性

- **防止雪崩，快速失败**：当下游服务持续失败时，熔断器会"跳闸"，阻止对该服务的进一步调用
- **自动恢复探测**：熔断器具备自动恢复能力，在跳闸一段时间后会进入"半开"状态
- **独立实例管理**：每个需要保护的资源都有独立的熔断器实例
- **动态配置支持**：通过配置中心实现熔断策略的动态更新
- **标准接口设计**：遵循 im-infra 组件的标准契约

## 快速开始

### 基本使用

```go
package main

import (
    "context"
    "log"

    "github.com/ceyewan/gochat/im-infra/breaker"
    "github.com/ceyewan/gochat/im-infra/clog"
)

func main() {
    // 1. 创建配置
    config := breaker.GetDefaultConfig("my-service", "development")

    // 2. 创建 Provider
    provider, err := breaker.New(context.Background(), config,
        breaker.WithLogger(clog.Namespace("breaker")),
    )
    if err != nil {
        log.Fatal(err)
    }
    defer provider.Close()

    // 3. 获取熔断器
    serviceBreaker := provider.GetBreaker("external-api")

    // 4. 使用熔断器保护操作
    err = serviceBreaker.Do(context.Background(), func() error {
        // 这里调用可能失败的外部服务
        return callExternalService()
    })

    if err != nil {
        if err == breaker.ErrBreakerOpen {
            log.Println("服务暂时不可用，熔断器已打开")
        } else {
            log.Printf("调用失败: %v", err)
        }
    }
}
```

### gRPC 集成

```go
// 创建 gRPC 客户端拦截器
func BreakerClientInterceptor(provider breaker.Provider) grpc.UnaryClientInterceptor {
    return func(ctx context.Context, method string, req, reply interface{},
             cc *grpc.ClientConn, invoker grpc.UnaryInvoker, opts ...grpc.CallOption) error {

        b := provider.GetBreaker(method)

        return b.Do(ctx, func() error {
            return invoker(ctx, method, req, reply, cc, opts...)
        })
    }
}

// 使用拦截器
conn, err := grpc.Dial("target-service",
    grpc.WithUnaryInterceptor(BreakerClientInterceptor(breakerProvider)),
)
```

### 配置中心集成

```go
// 创建带有配置中心的 Provider
provider, err := breaker.New(context.Background(), config,
    breaker.WithLogger(logger),
    breaker.WithCoordProvider(coordProvider),
)
```

## 配置

### 策略配置

熔断器策略通过以下参数控制：

- `FailureThreshold`: 触发跳闸的连续失败次数阈值
- `SuccessThreshold`: 半开状态下需要连续成功的次数
- `OpenStateTimeout`: 熔断器打开状态的持续时间

### 配置中心结构

策略存储在配置中心的路径结构：

```
/config/{env}/{service}/breakers/
├── default.json              # 默认策略
├── grpc:user-service.json    # 用户服务策略
├── grpc:order-service.json   # 订单服务策略
└── http:payment-api.json     # 支付API策略
```

示例策略文件：

```json
{
  "failureThreshold": 5,
  "successThreshold": 2,
  "openStateTimeout": "1m"
}
```

## 监控和日志

熔断器会记录以下关键事件：

- 熔断器创建
- 状态变更（关闭 → 打开 → 半开 → 关闭）
- 策略更新
- 操作失败

日志示例：

```json
{
  "level": "info",
  "msg": "circuit breaker state changed",
  "name": "grpc:user-service",
  "from": "CLOSED",
  "to": "OPEN"
}
```

## 最佳实践

### 1. 熔断器命名

使用有意义的命名规则，便于监控和调试：

```go
// 好的命名
provider.GetBreaker("grpc:user-service")
provider.GetBreaker("http:payment-api")
provider.GetBreaker("db:main-cluster")

// 避免的命名
provider.GetBreaker("breaker1")
provider.GetBreaker("service")
```

### 2. 策略调优

根据服务特性调整熔断策略：

- **关键服务**：较低的 FailureThreshold，较短的 OpenStateTimeout
- **非关键服务**：较高的 FailureThreshold，较长的 OpenStateTimeout
- **高频调用**：较低的 SuccessThreshold，快速恢复
- **低频调用**：较高的 SuccessThreshold，确保稳定性

### 3. 错误处理

正确处理熔断器错误：

```go
err := breaker.Do(ctx, func() error {
    return callService()
})

switch {
case err == nil:
    // 调用成功
case errors.Is(err, breaker.ErrBreakerOpen):
    // 熔断器打开，执行降级逻辑
    return executeFallback()
case errors.Is(err, context.DeadlineExceeded):
    // 超时错误
    return handleTimeout()
default:
    // 其他错误
    return handleError(err)
}
```

## 示例项目

查看 `examples/` 目录中的完整示例：

- `examples/basic/`: 基本使用示例
- `examples/grpc/`: gRPC 集成示例
- `examples/advanced/`: 高级配置示例

## 测试

运行测试：

```bash
go test ./im-infra/breaker/... -v
```

## 许可证

遵循 GoChat 项目的许可证。