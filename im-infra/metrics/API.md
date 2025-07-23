# `im-infra/metrics` - API 文档

本库为 GoChat 微服务提供可观测性能力。本文档详细描述了如何使用本库的公共 API。

## 1. 核心组件

本库提供一个核心的 `Provider` 接口，它是与本库交互的唯一入口。

```go
// im-infra/metrics/metrics.go

type Provider interface {
    // 获取 gRPC 服务端拦截器
	GRPCServerInterceptor() grpc.UnaryServerInterceptor
    // 获取 gRPC 客户端拦截器
	GRPCClientInterceptor() grpc.UnaryClientInterceptor
    // 获取 Gin HTTP 中间件
	HTTPMiddleware() gin.HandlerFunc
    // 优雅关闭
	Shutdown(ctx context.Context) error
}
```

## 2. 快速开始

在你的微服务中集成可观测性能力，只需三步。

### 第 1 步：配置和初始化

在服务的 `main` 函数或初始化逻辑中，创建 `metrics` provider。

```go
// main.go

import (
    "log"
    "context"
    "github.com/ceyewan/gochat/im-infra/metrics"
)

func main() {
    // 使用默认配置，或从文件加载
    cfg := metrics.DefaultConfig()
    cfg.ServiceName = "im-logic"
    cfg.PrometheusListenAddr = ":9091" // 暴露 Prometheus 指标端点
    cfg.ExporterType = "jaeger"       // 将 Traces 发送到 Jaeger
    cfg.ExporterEndpoint = "http://jaeger-collector:14268/api/traces"
    
    // 创建 Provider 实例
    metricsProvider, err := metrics.New(cfg)
    if err != nil {
        log.Fatalf("failed to create metrics provider: %v", err)
    }

    // 确保在服务退出时优雅关闭
    defer metricsProvider.Shutdown(context.Background())

    // ... 你的服务启动逻辑 ...
}
```

### 第 2 步：集成拦截器/中间件

根据你的服务类型，集成对应的拦截器。

#### gRPC 服务 (`im-logic`, `im-repo`, etc.)

在创建 gRPC Server 时，将 `GRPCServerInterceptor` 添加到拦截器链中。

```go
// server.go

import "google.golang.org/grpc"

server := grpc.NewServer(
    grpc.ChainUnaryInterceptor(
        metricsProvider.GRPCServerInterceptor(),
        // ... 其他你的拦截器 ...
    ),
)
```

如果你需要调用其他 gRPC 服务，请在创建 Client Conn 时添加 `GRPCClientInterceptor`。

```go
// client.go

conn, err := grpc.Dial(
    target,
    grpc.WithUnaryInterceptor(
        metricsProvider.GRPCClientInterceptor(),
    ),
)
```

#### HTTP/WebSocket 网关 (`im-gateway`)

在创建 Gin Engine 时，将 `HTTPMiddleware` 添加为全局中间件。

```go
// gateway.go

import "github.com/gin-gonic/gin"

engine := gin.New()
engine.Use(metricsProvider.HTTPMiddleware())

// ... 你的路由设置 ...
```

### 第 3 步：(可选) 添加自定义业务指标

本库提供了便捷的辅助函数来创建和记录自定义指标。

#### 创建一个计数器 (Counter)

```go
// service.go

// 在服务初始化时创建
loginSuccessCounter, err := metrics.NewCounter(
    "login_success_total", 
    "Total number of successful logins",
)
if err != nil { /* handle error */ }

// 在业务逻辑中调用
func (s *UserService) Login(...) error {
    // ... 登录逻辑 ...
    
    // 记录登录成功
    loginSuccessCounter.Inc(context.Background(), attribute.String("login_method", "password"))
    
    return nil
}
```

#### 创建一个直方图 (Histogram)

```go
// service.go

// 在服务初始化时创建
messageSizeHistogram, err := metrics.NewHistogram(
    "message_size_bytes", 
    "Size of processed messages in bytes",
    "bytes", // 单位
)
if err != nil { /* handle error */ }

// 在业务逻辑中调用
func (s *MessageService) Process(...) {
    // ... 处理消息 ...
    
    // 记录消息大小
    messageSizeHistogram.Record(context.Background(), float64(len(message.Body)))
}
```

## 3. 配置项说明

通过 `metrics.Config` 结构体进行配置。

| 字段 | 类型 | 描述 | 默认值 |
| :--- | :--- | :--- | :--- |
| `ServiceName` | `string` | **(必需)** 服务名称，如 "im-logic"。 | `unknown-service` |
| `ExporterType` | `string` | Trace Exporter 类型。支持: `jaeger`, `zipkin`, `stdout`。 | `stdout` |
| `ExporterEndpoint`| `string` | Trace Exporter 的地址。 | `http://localhost:14268/api/traces` |
| `PrometheusListenAddr`| `string` | Prometheus 指标端点的监听地址。如果为空，则不暴露。| `""` (关闭) |
| `SamplerType` | `string` | 采样策略。支持: `always_on`, `always_off`, `trace_id_ratio`。| `always_on` |
| `SamplerRatio` | `float64` | 如果采样策略为 `trace_id_ratio`，此为采样率 (0.0 to 1.0)。| `1.0` |
| `SlowRequestThreshold`| `time.Duration`| 慢请求阈值，用于指标记录。| `500ms` |

---
**完。**