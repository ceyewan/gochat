package metrics

import (
	"context"

	"github.com/ceyewan/gochat/im-infra/metrics/internal"
	"github.com/gin-gonic/gin"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/metric"
	"google.golang.org/grpc"
)

// Provider is the interface for the metrics and tracing system.
type Provider interface {
	GRPCServerInterceptor() grpc.UnaryServerInterceptor
	GRPCClientInterceptor() grpc.UnaryClientInterceptor
	HTTPMiddleware() gin.HandlerFunc
	Shutdown(ctx context.Context) error
}

type provider struct {
	internalProvider *internal.Provider
}

// New creates a new metrics and tracing provider based on the given config.
func New(cfg *Config) (Provider, error) {
	internalCfg := &internal.Config{
		ServiceName:          cfg.ServiceName,
		ExporterType:         cfg.ExporterType,
		ExporterEndpoint:     cfg.ExporterEndpoint,
		PrometheusListenAddr: cfg.PrometheusListenAddr,
		SamplerType:          cfg.SamplerType,
		SamplerRatio:         cfg.SamplerRatio,
		SlowRequestThreshold: cfg.SlowRequestThreshold,
	}
	p, err := internal.NewProvider(internalCfg)
	if err != nil {
		return nil, err
	}
	return &provider{internalProvider: p}, nil
}

func (p *provider) GRPCServerInterceptor() grpc.UnaryServerInterceptor {
	return p.internalProvider.GRPCServerInterceptor()
}

func (p *provider) GRPCClientInterceptor() grpc.UnaryClientInterceptor {
	return p.internalProvider.GRPCClientInterceptor()
}

func (p *provider) HTTPMiddleware() gin.HandlerFunc {
	return p.internalProvider.HTTPMiddleware()
}

func (p *provider) Shutdown(ctx context.Context) error {
	return p.internalProvider.Shutdown(ctx)
}

// --- Helper functions for custom metrics ---

// Counter is a metric that accumulates values over time.
type Counter struct {
	counter metric.Int64Counter
}

// NewCounter creates a new counter with a given name and description.
func NewCounter(name, description string) (*Counter, error) {
	counter, err := otel.Meter(internal.InstrumentationName).Int64Counter(name, metric.WithDescription(description))
	if err != nil {
		return nil, err
	}
	return &Counter{counter: counter}, nil
}

// Inc increments the counter by 1.
func (c *Counter) Inc(ctx context.Context, attrs ...attribute.KeyValue) {
	c.counter.Add(ctx, 1, metric.WithAttributes(attrs...))
}

// Add adds a value to the counter.
func (c *Counter) Add(ctx context.Context, value int64, attrs ...attribute.KeyValue) {
	c.counter.Add(ctx, value, metric.WithAttributes(attrs...))
}

// Histogram is a metric that samples observations.
type Histogram struct {
	histogram metric.Float64Histogram
}

// NewHistogram creates a new histogram with a given name, description, and unit.
func NewHistogram(name, description, unit string) (*Histogram, error) {
	histogram, err := otel.Meter(internal.InstrumentationName).Float64Histogram(name,
		metric.WithDescription(description),
		metric.WithUnit(unit),
	)
	if err != nil {
		return nil, err
	}
	return &Histogram{histogram: histogram}, nil
}

// Record records a new value for the histogram.
func (h *Histogram) Record(ctx context.Context, value float64, attrs ...attribute.KeyValue) {
	h.histogram.Record(ctx, value, metric.WithAttributes(attrs...))
}
