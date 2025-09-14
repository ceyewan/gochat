package internal

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/ceyewan/gochat/im-infra/clog"
	"github.com/gin-gonic/gin"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/exporters/jaeger"
	"go.opentelemetry.io/otel/exporters/prometheus"
	"go.opentelemetry.io/otel/exporters/stdout/stdouttrace"
	"go.opentelemetry.io/otel/exporters/zipkin"
	"go.opentelemetry.io/otel/propagation"
	sdkmetric "go.opentelemetry.io/otel/sdk/metric"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.17.0"
	"google.golang.org/grpc"
)

var (
	// 模块化日志器，用于不同组件的日志记录
	providerLogger   = clog.Namespace("metrics.provider")
	exporterLogger   = clog.Namespace("metrics.exporter")
	shutdownLogger   = clog.Namespace("metrics.shutdown")
	prometheusLogger = clog.Namespace("metrics.prometheus")
)

// ShutdownFunc 是一个可以被调用来优雅关闭 metrics 和 tracing providers 的函数。
type ShutdownFunc func(ctx context.Context) error

// Provider 是 metrics 和 tracing provider 的内部实现。
// 它封装了 OpenTelemetry 的复杂性，为上层提供简洁的接口。
type Provider struct {
	shutdownFunc ShutdownFunc
}

// NewProvider 创建一个新的内部 provider 实例。
//
// 该函数会依次初始化：
//   - OpenTelemetry Resource（服务标识）
//   - TracerProvider（链路追踪）
//   - MeterProvider（指标收集）
//   - 全局 propagator（跨服务上下文传播）
//
// 如果初始化过程中发生错误，会返回详细的错误信息。
func NewProvider(cfg *Config) (*Provider, error) {
	providerLogger.Info("开始初始化 metrics provider",
		clog.String("service_name", cfg.ServiceName),
		clog.String("exporter_type", cfg.ExporterType),
		clog.String("exporter_endpoint", cfg.ExporterEndpoint),
		clog.String("prometheus_addr", cfg.PrometheusListenAddr),
		clog.String("sampler_type", cfg.SamplerType),
		clog.Float64("sampler_ratio", cfg.SamplerRatio),
		clog.Duration("slow_threshold", cfg.SlowRequestThreshold))

	if cfg.ServiceName == "" {
		providerLogger.Error("service name cannot be empty")
		return nil, fmt.Errorf("service name must be configured")
	}

	// 创建 OpenTelemetry Resource
	providerLogger.Debug("创建 OpenTelemetry resource")
	res, err := resource.New(
		context.Background(),
		resource.WithAttributes(
			semconv.ServiceName(cfg.ServiceName),
		),
	)
	if err != nil {
		providerLogger.Error("failed to create OpenTelemetry resource",
			clog.Err(err))
		return nil, fmt.Errorf("failed to create resource: %w", err)
	}
	providerLogger.Debug("OpenTelemetry resource created successfully")

	// 初始化 TracerProvider
	providerLogger.Debug("初始化 tracer provider")
	tp, err := newTracerProvider(cfg, res)
	if err != nil {
		providerLogger.Error("failed to create tracer provider",
			clog.Err(err))
		return nil, fmt.Errorf("failed to create tracer provider: %w", err)
	}
	providerLogger.Info("tracer provider initialized successfully",
		clog.String("exporter_type", cfg.ExporterType),
		clog.String("sampler_type", cfg.SamplerType))

	// 设置全局 TracerProvider 和 Propagator
	otel.SetTracerProvider(tp)
	otel.SetTextMapPropagator(propagation.NewCompositeTextMapPropagator(
		propagation.TraceContext{},
		propagation.Baggage{}))
	providerLogger.Debug("global tracer provider and propagator set")

	// 初始化 MeterProvider
	providerLogger.Debug("初始化 meter provider")
	mp, err := newMeterProvider(cfg, res)
	if err != nil {
		providerLogger.Error("failed to create meter provider",
			clog.Err(err))
		return nil, fmt.Errorf("failed to create meter provider: %w", err)
	}
	otel.SetMeterProvider(mp)
	providerLogger.Info("meter provider initialized successfully",
		clog.String("prometheus_addr", cfg.PrometheusListenAddr))

	// 创建优雅关闭函数
	shutdown := func(ctx context.Context) error {
		shutdownLogger.Info("开始关闭 metrics provider")

		var errs []error

		// 关闭 TracerProvider
		shutdownLogger.Debug("关闭 tracer provider")
		if err := tp.Shutdown(ctx); err != nil {
			shutdownLogger.Error("failed to shutdown tracer provider", clog.Err(err))
			errs = append(errs, fmt.Errorf("failed to shutdown tracer provider: %w", err))
		} else {
			shutdownLogger.Debug("tracer provider shutdown successfully")
		}

		// 关闭 MeterProvider
		shutdownLogger.Debug("关闭 meter provider")
		if err := mp.Shutdown(ctx); err != nil {
			shutdownLogger.Error("failed to shutdown meter provider", clog.Err(err))
			errs = append(errs, fmt.Errorf("failed to shutdown meter provider: %w", err))
		} else {
			shutdownLogger.Debug("meter provider shutdown successfully")
		}

		if len(errs) > 0 {
			shutdownLogger.Error("shutdown completed with errors",
				clog.Int("error_count", len(errs)))
			return errs[0]
		}

		shutdownLogger.Info("metrics provider shutdown completed successfully")
		return nil
	}

	providerLogger.Info("metrics provider 初始化完成")
	return &Provider{shutdownFunc: shutdown}, nil
}

// Shutdown 调用内部的关闭函数，优雅地停止所有 metrics 相关服务。
func (p *Provider) Shutdown(ctx context.Context) error {
	return p.shutdownFunc(ctx)
}

// GRPCServerInterceptor 返回一个新的 gRPC 服务端拦截器。
func (p *Provider) GRPCServerInterceptor() grpc.UnaryServerInterceptor {
	return GRPCServerInterceptor()
}

// GRPCClientInterceptor 返回一个新的 gRPC 客户端拦截器。
func (p *Provider) GRPCClientInterceptor() grpc.UnaryClientInterceptor {
	return GRPCClientInterceptor()
}

// HTTPMiddleware 返回一个新的 Gin 中间件。
func (p *Provider) HTTPMiddleware() gin.HandlerFunc {
	return HTTPMiddleware()
}

// newTracerProvider 创建并配置 TracerProvider。
//
// 根据配置的 exporter 类型，创建对应的 span exporter：
//   - jaeger: 将 traces 发送到 Jaeger 收集器
//   - zipkin: 将 traces 发送到 Zipkin 收集器
//   - stdout: 将 traces 输出到标准输出（调试用）
//
// 同时根据配置的采样策略设置采样器。
func newTracerProvider(cfg *Config, res *resource.Resource) (*sdktrace.TracerProvider, error) {
	exporterLogger.Debug("创建 trace exporter",
		clog.String("type", cfg.ExporterType),
		clog.String("endpoint", cfg.ExporterEndpoint))

	var exporter sdktrace.SpanExporter
	var err error

	switch cfg.ExporterType {
	case "jaeger":
		exporter, err = jaeger.New(jaeger.WithCollectorEndpoint(
			jaeger.WithEndpoint(cfg.ExporterEndpoint)))
		if err == nil {
			exporterLogger.Info("jaeger exporter created successfully",
				clog.String("endpoint", cfg.ExporterEndpoint))
		}
	case "zipkin":
		exporter, err = zipkin.New(cfg.ExporterEndpoint)
		if err == nil {
			exporterLogger.Info("zipkin exporter created successfully",
				clog.String("endpoint", cfg.ExporterEndpoint))
		}
	case "stdout":
		exporter, err = stdouttrace.New(stdouttrace.WithPrettyPrint())
		if err == nil {
			exporterLogger.Info("stdout exporter created successfully")
		}
	default:
		exporterLogger.Error("unsupported trace exporter type",
			clog.String("type", cfg.ExporterType))
		return nil, fmt.Errorf("unsupported tracer exporter type: %s", cfg.ExporterType)
	}

	if err != nil {
		exporterLogger.Error("failed to create trace exporter",
			clog.String("type", cfg.ExporterType),
			clog.Err(err))
		return nil, fmt.Errorf("failed to create %s exporter: %w", cfg.ExporterType, err)
	}

	// 配置采样器
	var sampler sdktrace.Sampler
	switch cfg.SamplerType {
	case "always_on":
		sampler = sdktrace.AlwaysSample()
		exporterLogger.Debug("使用 always_on 采样策略")
	case "always_off":
		sampler = sdktrace.NeverSample()
		exporterLogger.Debug("使用 always_off 采样策略")
	case "trace_id_ratio":
		sampler = sdktrace.TraceIDRatioBased(cfg.SamplerRatio)
		exporterLogger.Debug("使用 trace_id_ratio 采样策略",
			clog.Float64("ratio", cfg.SamplerRatio))
	default:
		sampler = sdktrace.AlwaysSample()
		exporterLogger.Warn("未知的采样策略，使用默认的 always_on",
			clog.String("sampler_type", cfg.SamplerType))
	}

	tp := sdktrace.NewTracerProvider(
		sdktrace.WithBatcher(exporter),
		sdktrace.WithResource(res),
		sdktrace.WithSampler(sampler),
	)

	exporterLogger.Info("tracer provider created successfully")
	return tp, nil
}

// newMeterProvider 创建并配置 MeterProvider。
//
// 如果配置了 Prometheus 监听地址，会创建 Prometheus exporter 并启动 HTTP 服务器。
// 否则创建一个基本的 MeterProvider。
func newMeterProvider(cfg *Config, res *resource.Resource) (*sdkmetric.MeterProvider, error) {
	if cfg.PrometheusListenAddr == "" {
		exporterLogger.Info("prometheus 未启用，创建基本的 meter provider")
		return sdkmetric.NewMeterProvider(sdkmetric.WithResource(res)), nil
	}

	exporterLogger.Debug("创建 prometheus exporter")
	promExporter, err := prometheus.New()
	if err != nil {
		exporterLogger.Error("failed to create prometheus exporter", clog.Err(err))
		return nil, fmt.Errorf("failed to create prometheus exporter: %w", err)
	}
	exporterLogger.Debug("prometheus exporter created successfully")

	mp := sdkmetric.NewMeterProvider(
		sdkmetric.WithReader(promExporter),
		sdkmetric.WithResource(res),
	)

	// 启动 Prometheus HTTP 服务器
	go func() {
		prometheusLogger.Info("启动 prometheus metrics 服务器",
			clog.String("address", cfg.PrometheusListenAddr))

		mux := http.NewServeMux()
		mux.Handle("/metrics", promhttp.Handler())
		server := &http.Server{
			Addr:              cfg.PrometheusListenAddr,
			Handler:           mux,
			ReadHeaderTimeout: 10 * time.Second,
		}

		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			prometheusLogger.Error("prometheus server failed", clog.Err(err))
			// 使用 otel.Handle 确保错误被正确处理，但不会导致程序崩溃
			otel.Handle(fmt.Errorf("prometheus server failed: %w", err))
		} else {
			prometheusLogger.Info("prometheus server stopped")
		}
	}()

	exporterLogger.Info("meter provider with prometheus exporter created successfully")
	return mp, nil
}
