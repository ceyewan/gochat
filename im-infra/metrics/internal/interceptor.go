package internal

import (
	"context"
	"time"

	"github.com/gin-gonic/gin"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	otelcodes "go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/metric"
	"go.opentelemetry.io/otel/propagation"
	semconv "go.opentelemetry.io/otel/semconv/v1.17.0"
	"go.opentelemetry.io/otel/trace"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
)

const (
	InstrumentationName = "gochat/im-infra/metrics"
)

var (
	tracer = otel.Tracer(InstrumentationName)
	meter  = otel.Meter(InstrumentationName)

	// gRPC server metrics
	grpcServerRequests metric.Int64Counter
	grpcServerDuration metric.Float64Histogram

	// gRPC client metrics
	grpcClientRequests metric.Int64Counter
	grpcClientDuration metric.Float64Histogram

	// HTTP server metrics
	httpServerRequests metric.Int64Counter
	httpServerDuration metric.Float64Histogram
)

func init() {
	var err error
	grpcServerRequests, err = meter.Int64Counter("rpc.server.requests.count", metric.WithDescription("Number of gRPC requests received."))
	handleErr(err)
	grpcServerDuration, err = meter.Float64Histogram("rpc.server.duration", metric.WithDescription("Duration of gRPC requests in seconds."), metric.WithUnit("s"))
	handleErr(err)
	grpcClientRequests, err = meter.Int64Counter("rpc.client.requests.count", metric.WithDescription("Number of gRPC requests sent."))
	handleErr(err)
	grpcClientDuration, err = meter.Float64Histogram("rpc.client.duration", metric.WithDescription("Duration of gRPC client requests in seconds."), metric.WithUnit("s"))
	handleErr(err)
	httpServerRequests, err = meter.Int64Counter("http.server.requests.count", metric.WithDescription("Number of HTTP requests received."))
	handleErr(err)
	httpServerDuration, err = meter.Float64Histogram("http.server.duration", metric.WithDescription("Duration of HTTP requests in seconds."), metric.WithUnit("s"))
	handleErr(err)
}

// GRPCServerInterceptor returns a new gRPC server interceptor for tracing and metrics.
func GRPCServerInterceptor() grpc.UnaryServerInterceptor {
	return func(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (interface{}, error) {
		md, ok := metadata.FromIncomingContext(ctx)
		if !ok {
			md = metadata.New(nil)
		}
		ctx = otel.GetTextMapPropagator().Extract(ctx, &grpcMetadataCarrier{MD: md})

		spanCtx, span := tracer.Start(ctx, info.FullMethod, trace.WithSpanKind(trace.SpanKindServer),
			trace.WithAttributes(semconv.RPCSystemGRPC, semconv.RPCServiceKey.String(info.FullMethod)))
		defer span.End()

		startTime := time.Now()
		resp, err := handler(spanCtx, req)
		duration := time.Since(startTime)
		statusCode := status.Code(err)

		attrs := attribute.NewSet(
			semconv.RPCSystemGRPC,
			semconv.RPCServiceKey.String(info.FullMethod),
			semconv.RPCGRPCStatusCodeKey.Int(int(statusCode)),
		)
		grpcServerRequests.Add(spanCtx, 1, metric.WithAttributeSet(attrs))
		grpcServerDuration.Record(spanCtx, duration.Seconds(), metric.WithAttributeSet(attrs))

		sCode, sMsg := statusCodeToSpanStatus(statusCode)
		span.SetStatus(sCode, sMsg)
		if err != nil {
			span.RecordError(err)
		}
		return resp, err
	}
}

// GRPCClientInterceptor returns a new gRPC client interceptor for tracing and metrics.
func GRPCClientInterceptor() grpc.UnaryClientInterceptor {
	return func(ctx context.Context, method string, req, reply interface{}, cc *grpc.ClientConn, invoker grpc.UnaryInvoker, opts ...grpc.CallOption) error {
		spanCtx, span := tracer.Start(ctx, method, trace.WithSpanKind(trace.SpanKindClient),
			trace.WithAttributes(semconv.RPCSystemGRPC, semconv.RPCServiceKey.String(method)))
		defer span.End()

		md, ok := metadata.FromOutgoingContext(spanCtx)
		if !ok {
			md = metadata.New(nil)
		} else {
			md = md.Copy()
		}
		otel.GetTextMapPropagator().Inject(spanCtx, &grpcMetadataCarrier{MD: md})
		spanCtx = metadata.NewOutgoingContext(spanCtx, md)

		startTime := time.Now()
		err := invoker(spanCtx, method, req, reply, cc, opts...)
		duration := time.Since(startTime)
		statusCode := status.Code(err)

		attrs := attribute.NewSet(
			semconv.RPCSystemGRPC,
			semconv.RPCServiceKey.String(method),
			semconv.RPCGRPCStatusCodeKey.Int(int(statusCode)),
		)
		grpcClientRequests.Add(spanCtx, 1, metric.WithAttributeSet(attrs))
		grpcClientDuration.Record(spanCtx, duration.Seconds(), metric.WithAttributeSet(attrs))

		sCode, sMsg := statusCodeToSpanStatus(statusCode)
		span.SetStatus(sCode, sMsg)
		if err != nil {
			span.RecordError(err)
		}
		return err
	}
}

// HTTPMiddleware returns a new Gin middleware for tracing and metrics.
func HTTPMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		ctx := otel.GetTextMapPropagator().Extract(c.Request.Context(), propagation.HeaderCarrier(c.Request.Header))
		spanCtx, span := tracer.Start(ctx, c.FullPath(), trace.WithSpanKind(trace.SpanKindServer),
			trace.WithAttributes(
				semconv.HTTPMethodKey.String(c.Request.Method),
				semconv.HTTPURLKey.String(c.Request.URL.String()),
				semconv.NetHostNameKey.String(c.Request.Host),
			))
		defer span.End()

		c.Request = c.Request.WithContext(spanCtx)
		startTime := time.Now()
		c.Next()
		duration := time.Since(startTime)
		statusCode := c.Writer.Status()

		attrs := attribute.NewSet(
			semconv.HTTPMethodKey.String(c.Request.Method),
			semconv.HTTPRouteKey.String(c.FullPath()),
			semconv.HTTPStatusCodeKey.Int(statusCode),
		)
		httpServerRequests.Add(spanCtx, 1, metric.WithAttributeSet(attrs))
		httpServerDuration.Record(spanCtx, duration.Seconds(), metric.WithAttributeSet(attrs))

		sCode, sMsg := httpStatusCodeToSpanStatus(statusCode)
		span.SetStatus(sCode, sMsg)
		if len(c.Errors) > 0 {
			span.RecordError(c.Errors.Last().Err)
		}
	}
}

func handleErr(err error) {
	if err != nil {
		otel.Handle(err)
	}
}

func statusCodeToSpanStatus(code codes.Code) (otelcodes.Code, string) {
	if code >= codes.OK && code < codes.Canceled {
		return otelcodes.Ok, ""
	}
	return otelcodes.Error, code.String()
}

func httpStatusCodeToSpanStatus(code int) (otelcodes.Code, string) {
	if code >= 200 && code < 400 {
		return otelcodes.Ok, ""
	}
	return otelcodes.Error, ""
}

type grpcMetadataCarrier struct {
	MD metadata.MD
}

func (c *grpcMetadataCarrier) Get(key string) string {
	vals := c.MD.Get(key)
	if len(vals) > 0 {
		return vals[0]
	}
	return ""
}

func (c *grpcMetadataCarrier) Set(key, value string) {
	c.MD.Set(key, value)
}

func (c *grpcMetadataCarrier) Keys() []string {
	keys := make([]string, 0, len(c.MD))
	for k := range c.MD {
		keys = append(keys, k)
	}
	return keys
}
