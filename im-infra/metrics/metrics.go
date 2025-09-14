// Package metrics 为 GoChat 分布式系统提供统一的可观测性基础设施。
//
// 本包基于 OpenTelemetry 构建，专为微服务架构设计，提供了：
//   - 自动化的链路追踪（Tracing）
//   - 指标收集（Metrics）
//   - 统一的拦截器和中间件
//   - 与监控系统的无缝集成
//
// # 核心设计理念
//
//   - 自动化优先：通过拦截器自动收集可观测性数据，无需手动埋点
//   - 零侵入集成：业务代码无需关心底层实现细节
//   - 生产就绪：内置最佳实践配置，开箱即用
//   - 标准兼容：基于 OpenTelemetry 标准，支持多种后端系统
//
// # 快速开始
//
// 创建一个 Provider 实例并集成到你的服务中：
//
//	cfg := metrics.DefaultConfig()
//	cfg.ServiceName = "im-logic"
//	cfg.PrometheusListenAddr = ":9091"
//
//	provider, err := metrics.New(cfg)
//	if err != nil {
//	    log.Fatal(err)
//	}
//	defer provider.Shutdown(context.Background())
//
//	// gRPC 服务集成
//	server := grpc.NewServer(
//	    grpc.ChainUnaryInterceptor(provider.GRPCServerInterceptor()),
//	)
//
//	// HTTP 服务集成
//	engine := gin.New()
//	engine.Use(provider.HTTPMiddleware())
//
// # 可观测性数据
//
// 本包会自动收集以下数据：
//   - 请求计数和延迟分布
//   - 错误率和状态码分布
//   - 分布式链路追踪信息
//   - 服务间调用关系图
//
// 详细的 API 文档请参考 API.md 文件。
package metrics

import (
	"context"

	"github.com/ceyewan/gochat/im-infra/clog"
	"github.com/ceyewan/gochat/im-infra/metrics/internal"
	"github.com/gin-gonic/gin"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/metric"
	"google.golang.org/grpc"
)

var (
	// 模块化日志器，用于记录 metrics 包的操作日志
	metricsLogger = clog.Namespace("metrics")
	helperLogger  = clog.Namespace("metrics.helper")
)

// Provider 定义了 metrics 和 tracing 系统的核心接口。
//
// 它是与本库交互的唯一入口，提供了：
//   - gRPC 服务端和客户端拦截器
//   - HTTP 中间件支持
//   - 优雅关闭机制
//
// 通过 Provider 接口，上层业务代码与底层 OpenTelemetry 实现完全解耦，
// 为未来的扩展和重构提供了最大的灵活性。
type Provider interface {
	// GRPCServerInterceptor 返回 gRPC 服务端拦截器。
	// 自动为所有 gRPC 请求添加 tracing 和 metrics 收集。
	GRPCServerInterceptor() grpc.UnaryServerInterceptor

	// GRPCClientInterceptor 返回 gRPC 客户端拦截器。
	// 自动为所有出站 gRPC 请求添加 tracing 和 metrics 收集。
	GRPCClientInterceptor() grpc.UnaryClientInterceptor

	// HTTPMiddleware 返回 Gin HTTP 中间件。
	// 自动为所有 HTTP 请求添加 tracing 和 metrics 收集。
	HTTPMiddleware() gin.HandlerFunc

	// Shutdown 优雅关闭所有 metrics 相关服务。
	// 应在应用程序退出时调用，确保所有数据都被正确导出。
	Shutdown(ctx context.Context) error
}

// provider 是 Provider 接口的内部实现。
// 它封装了 internal.Provider，作为内外部接口的桥梁。
type provider struct {
	internalProvider *internal.Provider
	serviceName      string // 缓存服务名称，用于日志记录
}

// New 创建一个新的 metrics 和 tracing provider 实例。
//
// 该函数会根据提供的配置初始化整个可观测性系统，包括：
//   - OpenTelemetry TracerProvider（链路追踪）
//   - OpenTelemetry MeterProvider（指标收集）
//   - 配置的 Exporter（数据导出）
//   - Prometheus HTTP 服务器（如果启用）
//
// 如果初始化过程中发生任何错误，会返回详细的错误信息。
// 成功创建的 Provider 实例应该在应用程序退出时调用 Shutdown 方法。
//
// 参数：
//   - cfg: 配置信息，包含服务名称、导出器类型等设置
//
// 返回：
//   - Provider: 可观测性系统的操作接口
//   - error: 初始化过程中的错误信息
func New(cfg *Config) (Provider, error) {
	metricsLogger.Info("开始创建 metrics provider",
		clog.String("service_name", cfg.ServiceName),
		clog.String("exporter_type", cfg.ExporterType))

	// 将公共配置转换为内部配置
	internalCfg := &internal.Config{
		ServiceName:          cfg.ServiceName,
		ExporterType:         cfg.ExporterType,
		ExporterEndpoint:     cfg.ExporterEndpoint,
		PrometheusListenAddr: cfg.PrometheusListenAddr,
		SamplerType:          cfg.SamplerType,
		SamplerRatio:         cfg.SamplerRatio,
		SlowRequestThreshold: cfg.SlowRequestThreshold,
	}

	// 创建内部 provider
	p, err := internal.NewProvider(internalCfg)
	if err != nil {
		metricsLogger.Error("failed to create internal provider",
			clog.String("service_name", cfg.ServiceName),
			clog.Err(err))
		return nil, err
	}

	metricsLogger.Info("metrics provider 创建成功",
		clog.String("service_name", cfg.ServiceName))

	return &provider{
		internalProvider: p,
		serviceName:      cfg.ServiceName,
	}, nil
}

// GRPCServerInterceptor 返回 gRPC 服务端拦截器。
func (p *provider) GRPCServerInterceptor() grpc.UnaryServerInterceptor {
	metricsLogger.Debug("获取 gRPC 服务端拦截器",
		clog.String("service_name", p.serviceName))
	return p.internalProvider.GRPCServerInterceptor()
}

// GRPCClientInterceptor 返回 gRPC 客户端拦截器。
func (p *provider) GRPCClientInterceptor() grpc.UnaryClientInterceptor {
	metricsLogger.Debug("获取 gRPC 客户端拦截器",
		clog.String("service_name", p.serviceName))
	return p.internalProvider.GRPCClientInterceptor()
}

// HTTPMiddleware 返回 Gin HTTP 中间件。
func (p *provider) HTTPMiddleware() gin.HandlerFunc {
	metricsLogger.Debug("获取 HTTP 中间件",
		clog.String("service_name", p.serviceName))
	return p.internalProvider.HTTPMiddleware()
}

// Shutdown 优雅关闭 metrics provider。
//
// 该方法会依次关闭：
//   - TracerProvider（等待所有 span 导出完成）
//   - MeterProvider（等待所有指标导出完成）
//   - Prometheus HTTP 服务器（如果启用）
//
// 建议在应用程序退出时调用此方法，确保所有可观测性数据都被正确处理。
func (p *provider) Shutdown(ctx context.Context) error {
	metricsLogger.Info("开始关闭 metrics provider",
		clog.String("service_name", p.serviceName))

	err := p.internalProvider.Shutdown(ctx)
	if err != nil {
		metricsLogger.Error("metrics provider 关闭时发生错误",
			clog.String("service_name", p.serviceName),
			clog.Err(err))
		return err
	}

	metricsLogger.Info("metrics provider 关闭完成",
		clog.String("service_name", p.serviceName))
	return nil
}

// --- 自定义指标辅助函数 ---

// Counter 是一个只能递增的计数器指标。
//
// 适用于记录请求次数、错误次数等累积型数据。
// Counter 是线程安全的，可以在并发环境中使用。
type Counter struct {
	counter metric.Int64Counter
	name    string // 指标名称，用于日志记录
}

// NewCounter 创建一个新的计数器指标。
//
// 计数器是最常用的指标类型，用于记录事件发生的次数。
// 创建后的计数器可以通过 Inc() 和 Add() 方法进行操作。
//
// 参数：
//   - name: 指标名称，应该具有描述性且符合命名规范
//   - description: 指标描述，说明该指标的用途和含义
//
// 返回：
//   - *Counter: 计数器实例
//   - error: 创建过程中的错误信息
//
// 示例：
//
//	loginCounter, err := metrics.NewCounter(
//	    "user_login_total",
//	    "Total number of user login attempts",
//	)
//	if err != nil {
//	    log.Fatal(err)
//	}
func NewCounter(name, description string) (*Counter, error) {
	helperLogger.Debug("创建新的计数器指标",
		clog.String("name", name),
		clog.String("description", description))

	counter, err := otel.Meter(internal.InstrumentationName).Int64Counter(
		name,
		metric.WithDescription(description))
	if err != nil {
		helperLogger.Error("failed to create counter",
			clog.String("name", name),
			clog.Err(err))
		return nil, err
	}

	helperLogger.Info("计数器指标创建成功",
		clog.String("name", name))

	return &Counter{
		counter: counter,
		name:    name,
	}, nil
}

// Inc 将计数器的值增加 1。
//
// 这是最常用的计数器操作，适用于记录事件发生次数。
// 可以通过 attrs 参数添加标签来区分不同的计数维度。
//
// 参数：
//   - ctx: 上下文，用于传递 trace 信息
//   - attrs: 可选的属性标签，用于数据分组和过滤
//
// 示例：
//
//	loginCounter.Inc(ctx,
//	    attribute.String("method", "password"),
//	    attribute.String("result", "success"))
func (c *Counter) Inc(ctx context.Context, attrs ...attribute.KeyValue) {
	c.counter.Add(ctx, 1, metric.WithAttributes(attrs...))
}

// Add 将计数器的值增加指定的数量。
//
// 用于批量计数或者根据实际业务逻辑增加不同的数值。
// value 必须为非负数，负数会被忽略。
//
// 参数：
//   - ctx: 上下文，用于传递 trace 信息
//   - value: 要增加的数值，必须为非负数
//   - attrs: 可选的属性标签，用于数据分组和过滤
func (c *Counter) Add(ctx context.Context, value int64, attrs ...attribute.KeyValue) {
	if value < 0 {
		helperLogger.Warn("忽略负数计数器增量",
			clog.String("counter", c.name),
			clog.Int64("value", value))
		return
	}
	c.counter.Add(ctx, value, metric.WithAttributes(attrs...))
}

// Histogram 是一个直方图指标，用于记录数值分布情况。
//
// 适用于记录请求延迟、文件大小、队列长度等连续型数据。
// Histogram 会自动计算数据的分位数、平均值等统计信息。
type Histogram struct {
	histogram metric.Float64Histogram
	name      string // 指标名称，用于日志记录
}

// NewHistogram 创建一个新的直方图指标。
//
// 直方图用于观察数值的分布情况，特别适合记录延迟、大小等指标。
// OpenTelemetry 会自动为直方图生成多个时间序列，包括计数、总和和分桶信息。
//
// 参数：
//   - name: 指标名称，应该具有描述性且符合命名规范
//   - description: 指标描述，说明该指标的用途和含义
//   - unit: 数据单位，如 "ms"、"bytes"、"requests" 等
//
// 返回：
//   - *Histogram: 直方图实例
//   - error: 创建过程中的错误信息
//
// 示例：
//
//	responseTimeHist, err := metrics.NewHistogram(
//	    "http_request_duration_seconds",
//	    "HTTP request duration in seconds",
//	    "s",
//	)
//	if err != nil {
//	    log.Fatal(err)
//	}
func NewHistogram(name, description, unit string) (*Histogram, error) {
	helperLogger.Debug("创建新的直方图指标",
		clog.String("name", name),
		clog.String("description", description),
		clog.String("unit", unit))

	histogram, err := otel.Meter(internal.InstrumentationName).Float64Histogram(
		name,
		metric.WithDescription(description),
		metric.WithUnit(unit),
	)
	if err != nil {
		helperLogger.Error("failed to create histogram",
			clog.String("name", name),
			clog.Err(err))
		return nil, err
	}

	helperLogger.Info("直方图指标创建成功",
		clog.String("name", name))

	return &Histogram{
		histogram: histogram,
		name:      name,
	}, nil
}

// Record 记录一个新的观测值到直方图中。
//
// 每次调用都会将数值添加到直方图的相应分桶中，
// 监控系统会基于这些数据计算百分位数、平均值等统计信息。
//
// 参数：
//   - ctx: 上下文，用于传递 trace 信息
//   - value: 要记录的数值
//   - attrs: 可选的属性标签，用于数据分组和过滤
//
// 示例：
//
//	// 记录 HTTP 请求处理时间
//	start := time.Now()
//	// ... 处理请求 ...
//	duration := time.Since(start)
//	responseTimeHist.Record(ctx, duration.Seconds(),
//	    attribute.String("method", "GET"),
//	    attribute.String("status", "200"))
func (h *Histogram) Record(ctx context.Context, value float64, attrs ...attribute.KeyValue) {
	h.histogram.Record(ctx, value, metric.WithAttributes(attrs...))
}
