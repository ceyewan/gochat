# 基础设施: Metrics 可观测性

## 1. 设计理念

`metrics` 是 `gochat` 项目统一的可观测性基础设施库，基于业界标准 **OpenTelemetry** 构建。它为所有微服务提供了开箱即用的、自动化的**指标 (Metrics)** 和**链路追踪 (Tracing)** 能力。

该组件的核心设计理念是**自动化与零侵入**。开发者只需在服务初始化时注入相应的 gRPC 拦截器或 HTTP 中间件，即可自动获得所有请求的性能指标和分布式追踪数据，无需在业务代码中进行手动埋点，极大地降低了可观测性接入的成本。

## 2. 核心 API 契约

### 2.1 构造函数

```go
// Config 是 metrics 组件的配置结构体。
type Config struct {
	// ServiceName 服务的唯一标识名称 (必填)。
	ServiceName string `json:"serviceName"`
	// ExporterType 指定 trace 数据的导出器类型: "jaeger", "zipkin", "stdout"。
	ExporterType string `json:"exporterType"`
	// ExporterEndpoint 指定 trace exporter 的目标地址。
	ExporterEndpoint string `json:"exporterEndpoint"`
	// PrometheusListenAddr 指定 Prometheus metrics 端点的监听地址 (e.g., ":9090")。
	PrometheusListenAddr string `json:"prometheusListenAddr"`
	// SamplerType 指定 trace 采样策略: "always_on", "always_off", "trace_id_ratio"。
	SamplerType string `json:"samplerType"`
	// SamplerRatio 采样比例 (0.0 to 1.0)，仅当 SamplerType 为 "trace_id_ratio" 时有效。
	SamplerRatio float64 `json:"samplerRatio"`
}

// GetDefaultConfig 返回一个推荐的默认配置。
// serviceName 是必填项。env 可以是 "development" 或 "production"。
func GetDefaultConfig(serviceName, env string) *Config

// Option 是用于配置 metrics Provider 的函数式选项。
type Option func(*providerOptions)

// WithLogger 为 metrics Provider 设置一个 clog.Logger 实例。
func WithLogger(logger clog.Logger) Option

// New 创建一个新的 metrics Provider 实例。
// 这是与 metrics 组件交互的唯一入口。
func New(ctx context.Context, config *Config, opts ...Option) (Provider, error)
```

### 2.2 Provider 接口

`Provider` 接口是所有可观测性能力的总入口。

```go
// Provider 定义了 metrics 组件提供的所有能力。
type Provider interface {
	// GRPCServerInterceptor 返回 gRPC 服务端拦截器。
	GRPCServerInterceptor() grpc.UnaryServerInterceptor
	// GRPCClientInterceptor 返回 gRPC 客户端拦截器。
	GRPCClientInterceptor() grpc.UnaryClientInterceptor
	// HTTPMiddleware 返回 Gin HTTP 中间件。
	HTTPMiddleware() gin.HandlerFunc

	// Shutdown 优雅关闭所有 metrics 相关服务，确保所有缓冲的数据都被成功发送。
	Shutdown(ctx context.Context) error
}
```

### 2.3 自定义指标接口

除了自动收集的指标，`metrics` 组件也支持创建自定义的业务指标。

```go
// NewCounter 创建一个新的计数器指标。
// name: 指标名称 (e.g., "user_logins_total")
func NewCounter(name, description string) (Counter, error)

// Counter 是一个只能递增的计数器指标。
type Counter interface {
    Inc(ctx context.Context, attrs ...attribute.KeyValue)
    Add(ctx context.Context, value int64, attrs ...attribute.KeyValue)
}

// NewHistogram 创建一个新的直方图指标。
// name: 指标名称 (e.g., "request_duration_seconds")
// unit: 数据单位 (e.g., "ms", "bytes")
func NewHistogram(name, description, unit string) (Histogram, error)

// Histogram 是一个直方图指标，用于记录数值分布情况。
type Histogram interface {
    Record(ctx context.Context, value float64, attrs ...attribute.KeyValue)
}
```

## 3. 标准用法

### 场景 1: 在 gRPC 服务中自动集成

```go
// 1. 在服务启动时初始化 Provider
// 推荐使用 GetDefaultConfig 获取标准配置，然后按需覆盖
config := metrics.GetDefaultConfig("im-gateway", "production")
// config.ExporterEndpoint = "http://my-jaeger:14268/api/traces" // 按需覆盖

// 创建 Provider 实例，并通过 With... Options 注入依赖
metricsProvider, err := metrics.New(context.Background(), config,
    metrics.WithLogger(clog.Namespace("metrics")),
)
if err != nil {
    log.Fatalf("无法创建 metrics provider: %v", err)
}
defer metricsProvider.Shutdown(context.Background())

// 2. 在创建 gRPC Server 时链入拦截器
server := grpc.NewServer(
    grpc.ChainUnaryInterceptor(
        metricsProvider.GRPCServerInterceptor(),
        // ... 其他拦截器
    ),
)

// 完成！此后所有 gRPC 请求的指标和 trace 都会被自动记录。
```

### 场景 2: 在 Gin 服务中自动集成

```go
// 1. 初始化 Provider (同上)

// 2. 在创建 Gin 引擎时注册中间件
engine := gin.New()
engine.Use(metricsProvider.HTTPMiddleware())

// 完成！此后所有 HTTP 请求的指标和 trace 都会被自动记录。
```

### 场景 3: 上报自定义业务指标

```go
// 1. 在服务初始化时创建指标
var loginCounter metrics.Counter
func initMetrics() {
    var err error
    loginCounter, err = metrics.NewCounter("user_logins_total", "用户登录总次数")
    if err != nil {
        // ... handle error
    }
}

// 2. 在业务逻辑中记录数据
func (s *AuthService) Login(ctx context.Context, req *pb.LoginRequest) (*pb.LoginResponse, error) {
    // ... 登录逻辑 ...
    isSuccess := true // or false
    
    // 使用标签(label)来区分不同维度
    loginCounter.Inc(ctx, 
        attribute.Bool("success", isSuccess),
        attribute.String("login_method", "password"),
    )

    return &pb.LoginResponse{}, nil
}