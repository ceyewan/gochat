# 基础设施: Metrics 可观测性

## 1. 概述

`metrics` 是 `gochat` 项目统一的可观测性基础设施库，基于业界标准 **OpenTelemetry** 构建。它为所有微服务提供了开箱即用的、自动化的**指标 (Metrics)** 和**链路追踪 (Tracing)** 能力。

该组件的核心设计理念是**自动化与零侵入**。开发者只需在服务初始化时注入相应的拦截器，即可自动获得所有请求的性能指标和分布式追踪数据，无需在业务代码中进行手动埋点。

## 2. 核心用法

### 2.1 初始化

所有服务（如 `im-gateway`, `im-logic`）都应在启动时初始化 `metrics` Provider。

```go
import "github.com/ceyewan/gochat/im-infra/metrics"

// 1. 获取默认配置，并设置服务名和 Prometheus 监听地址
cfg := metrics.DefaultConfig()
cfg.ServiceName = "im-logic"
cfg.PrometheusListenAddr = ":9091" // 暴露 /metrics 端点

// 2. 创建 Provider 实例
provider, err := metrics.New(cfg)
if err != nil {
    log.Fatalf("无法创建 metrics provider: %v", err)
}
// 确保在服务关闭时优雅地关闭 provider，以发送所有缓冲的数据
defer provider.Shutdown(context.Background())
```

### 2.2 gRPC 服务集成

对于 gRPC 服务，只需在创建 Server 时链入拦截器。

```go
import "google.golang.org/grpc"

// 获取 gRPC 服务端拦截器
grpcInterceptor := provider.GRPCServerInterceptor()

// 创建 gRPC 服务器
server := grpc.NewServer(
    grpc.ChainUnaryInterceptor(grpcInterceptor),
    // ... 其他拦截器
)
```

### 2.3 HTTP (Gin) 服务集成

对于使用 Gin 框架的 HTTP 服务，只需将中间件注册到引擎中。

```go
import "github.com/gin-gonic/gin"

// 获取 Gin 中间件
httpMiddleware := provider.HTTPMiddleware()

// 创建 Gin 引擎并使用中间件
engine := gin.New()
engine.Use(httpMiddleware)
```

### 2.4 自定义指标

除了自动收集的指标，您还可以定义自己的业务指标。

#### 计数器 (Counter)

用于记录事件发生的次数，例如用户注册数、消息发送数。

```go
// 在服务初始化时创建计数器
loginCounter, err := metrics.NewCounter(
    "user_logins_total",
    "用户登录总次数",
)
if err != nil {
    // ...
}

// 在业务逻辑中增加计数值
func handleUserLogin(c *gin.Context) {
    // ... 登录逻辑
    isSuccess := true // or false
    
    // 使用标签(label)来区分不同维度
    loginCounter.Inc(c.Request.Context(), 
        attribute.Bool("success", isSuccess),
        attribute.String("login_method", "password"),
    )
}
```

#### 直方图 (Histogram)

用于记录数值的分布情况，例如消息体的大小、处理任务的耗时。

```go
// 在服务初始化时创建直方图
messageSizeHist, err := metrics.NewHistogram(
    "message_size_bytes",
    "消息体大小分布",
    "bytes", // 单位
)
if err != nil {
    // ...
}

// 在业务逻辑中记录观测值
func handleSendMessage(c *gin.Context) {
    // ...
    messageSize := len(messageBody)
    messageSizeHist.Record(c.Request.Context(), float64(messageSize))
}
```

## 3. API 参考

```go
// Provider 是与本库交互的唯一入口。
type Provider interface {
	// GRPCServerInterceptor 返回 gRPC 服务端拦截器。
	GRPCServerInterceptor() grpc.UnaryServerInterceptor
	// GRPCClientInterceptor 返回 gRPC 客户端拦截器。
	GRPCClientInterceptor() grpc.UnaryClientInterceptor
	// HTTPMiddleware 返回 Gin HTTP 中间件。
	HTTPMiddleware() gin.HandlerFunc
	// Shutdown 优雅关闭所有 metrics 相关服务。
	Shutdown(ctx context.Context) error
}

// New 创建一个新的 metrics 和 tracing provider 实例。
func New(cfg *Config) (Provider, error)

// NewCounter 创建一个新的计数器指标。
// name: 指标名称 (e.g., "user_login_total")
// description: 指标描述
func NewCounter(name, description string) (*Counter, error)

// Counter 是一个只能递增的计数器指标。
type Counter interface {
    Inc(ctx context.Context, attrs ...attribute.KeyValue)
    Add(ctx context.Context, value int64, attrs ...attribute.KeyValue)
}

// NewHistogram 创建一个新的直方图指标。
// name: 指标名称 (e.g., "request_duration_seconds")
// description: 指标描述
// unit: 数据单位 (e.g., "ms", "bytes")
func NewHistogram(name, description, unit string) (*Histogram, error)

// Histogram 是一个直方图指标，用于记录数值分布情况。
type Histogram interface {
    Record(ctx context.Context, value float64, attrs ...attribute.KeyValue)
}
```

## 4. 配置

`metrics` 组件通过 `Config` 结构体进行配置，通常由 `coord` 配置中心管理。

```go
type Config struct {
	// ServiceName 服务的唯一标识名称 (必填)
	ServiceName string
	// ExporterType 指定 trace 数据的导出器类型: "jaeger", "zipkin", "stdout"
	ExporterType string
	// ExporterEndpoint 指定 trace exporter 的目标地址
	ExporterEndpoint string
	// PrometheusListenAddr 指定 Prometheus metrics 端点的监听地址 (e.g., ":9090")
	PrometheusListenAddr string
	// SamplerType 指定 trace 采样策略: "always_on", "always_off", "trace_id_ratio"
	SamplerType string
	// SamplerRatio 采样比例 (0.0 to 1.0)，仅当 SamplerType 为 "trace_id_ratio" 时有效
	SamplerRatio float64
}