package internal

import (
	"context"
	"fmt"
	"net/http"
	"time"

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

// ShutdownFunc is a function that can be called to gracefully shutdown the metrics and tracing providers.
type ShutdownFunc func(ctx context.Context) error

// Provider is the internal implementation of the metrics and tracing provider.
type Provider struct {
	shutdownFunc ShutdownFunc
}

// NewProvider creates a new internal provider.
func NewProvider(cfg *Config) (*Provider, error) {
	if cfg.ServiceName == "" {
		return nil, fmt.Errorf("service name must be configured")
	}

	res, err := resource.New(
		context.Background(),
		resource.WithAttributes(
			semconv.ServiceName(cfg.ServiceName),
		),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create resource: %w", err)
	}

	tp, err := newTracerProvider(cfg, res)
	if err != nil {
		return nil, fmt.Errorf("failed to create tracer provider: %w", err)
	}

	otel.SetTracerProvider(tp)
	otel.SetTextMapPropagator(propagation.NewCompositeTextMapPropagator(propagation.TraceContext{}, propagation.Baggage{}))

	mp, err := newMeterProvider(cfg, res)
	if err != nil {
		return nil, fmt.Errorf("failed to create meter provider: %w", err)
	}
	otel.SetMeterProvider(mp)

	shutdown := func(ctx context.Context) error {
		var errs []error
		if err := tp.Shutdown(ctx); err != nil {
			errs = append(errs, fmt.Errorf("failed to shutdown tracer provider: %w", err))
		}
		if err := mp.Shutdown(ctx); err != nil {
			errs = append(errs, fmt.Errorf("failed to shutdown meter provider: %w", err))
		}
		if len(errs) > 0 {
			return errs[0]
		}
		return nil
	}

	return &Provider{shutdownFunc: shutdown}, nil
}

// Shutdown calls the internal shutdown function.
func (p *Provider) Shutdown(ctx context.Context) error {
	return p.shutdownFunc(ctx)
}

// GRPCServerInterceptor returns a new gRPC server interceptor.
func (p *Provider) GRPCServerInterceptor() grpc.UnaryServerInterceptor {
	return GRPCServerInterceptor()
}

// GRPCClientInterceptor returns a new gRPC client interceptor.
func (p *Provider) GRPCClientInterceptor() grpc.UnaryClientInterceptor {
	return GRPCClientInterceptor()
}

// HTTPMiddleware returns a new Gin middleware.
func (p *Provider) HTTPMiddleware() gin.HandlerFunc {
	return HTTPMiddleware()
}

func newTracerProvider(cfg *Config, res *resource.Resource) (*sdktrace.TracerProvider, error) {
	var exporter sdktrace.SpanExporter
	var err error

	switch cfg.ExporterType {
	case "jaeger":
		exporter, err = jaeger.New(jaeger.WithCollectorEndpoint(jaeger.WithEndpoint(cfg.ExporterEndpoint)))
	case "zipkin":
		exporter, err = zipkin.New(cfg.ExporterEndpoint)
	case "stdout":
		exporter, err = stdouttrace.New(stdouttrace.WithPrettyPrint())
	default:
		return nil, fmt.Errorf("unsupported tracer exporter type: %s", cfg.ExporterType)
	}
	if err != nil {
		return nil, fmt.Errorf("failed to create %s exporter: %w", cfg.ExporterType, err)
	}

	var sampler sdktrace.Sampler
	switch cfg.SamplerType {
	case "always_on":
		sampler = sdktrace.AlwaysSample()
	case "always_off":
		sampler = sdktrace.NeverSample()
	case "trace_id_ratio":
		sampler = sdktrace.TraceIDRatioBased(cfg.SamplerRatio)
	default:
		sampler = sdktrace.AlwaysSample()
	}

	tp := sdktrace.NewTracerProvider(
		sdktrace.WithBatcher(exporter),
		sdktrace.WithResource(res),
		sdktrace.WithSampler(sampler),
	)
	return tp, nil
}

func newMeterProvider(cfg *Config, res *resource.Resource) (*sdkmetric.MeterProvider, error) {
	if cfg.PrometheusListenAddr == "" {
		return sdkmetric.NewMeterProvider(sdkmetric.WithResource(res)), nil
	}

	promExporter, err := prometheus.New()
	if err != nil {
		return nil, fmt.Errorf("failed to create prometheus exporter: %w", err)
	}

	mp := sdkmetric.NewMeterProvider(
		sdkmetric.WithReader(promExporter),
		sdkmetric.WithResource(res),
	)

	go func() {
		mux := http.NewServeMux()
		mux.Handle("/metrics", promhttp.Handler())
		server := &http.Server{
			Addr:              cfg.PrometheusListenAddr,
			Handler:           mux,
			ReadHeaderTimeout: 10 * time.Second,
		}
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			otel.Handle(fmt.Errorf("prometheus server failed: %w", err))
		}
	}()

	return mp, nil
}
