package internal

import (
	"context"
	"time"

	"github.com/ceyewan/gochat/im-infra/clog"
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
	// InstrumentationName 定义了本库的 OpenTelemetry instrumentation 名称。
	// 用于标识来自本库的 traces 和 metrics。
	InstrumentationName = "gochat/im-infra/metrics"
)

var (
	// OpenTelemetry 核心组件
	tracer = otel.Tracer(InstrumentationName)
	meter  = otel.Meter(InstrumentationName)

	// 模块化日志器
	interceptorLogger = clog.Module("metrics.interceptor")
	grpcServerLogger  = clog.Module("metrics.grpc.server")
	grpcClientLogger  = clog.Module("metrics.grpc.client")
	httpServerLogger  = clog.Module("metrics.http.server")

	// gRPC 服务端指标
	grpcServerRequests metric.Int64Counter
	grpcServerDuration metric.Float64Histogram

	// gRPC 客户端指标
	grpcClientRequests metric.Int64Counter
	grpcClientDuration metric.Float64Histogram

	// HTTP 服务端指标
	httpServerRequests metric.Int64Counter
	httpServerDuration metric.Float64Histogram
)

// init 初始化所有的 metrics 仪表。
// 如果初始化失败，会记录详细的错误日志。
func init() {
	interceptorLogger.Debug("初始化 metrics 仪表")

	var err error

	// 初始化 gRPC 服务端指标
	grpcServerRequests, err = meter.Int64Counter(
		"rpc.server.requests.count",
		metric.WithDescription("Number of gRPC requests received."))
	if err != nil {
		interceptorLogger.Error("failed to create grpc server requests counter", clog.Err(err))
		return
	}

	grpcServerDuration, err = meter.Float64Histogram(
		"rpc.server.duration",
		metric.WithDescription("Duration of gRPC requests in seconds."),
		metric.WithUnit("s"))
	if err != nil {
		interceptorLogger.Error("failed to create grpc server duration histogram", clog.Err(err))
		return
	}

	// 初始化 gRPC 客户端指标
	grpcClientRequests, err = meter.Int64Counter(
		"rpc.client.requests.count",
		metric.WithDescription("Number of gRPC requests sent."))
	if err != nil {
		interceptorLogger.Error("failed to create grpc client requests counter", clog.Err(err))
		return
	}

	grpcClientDuration, err = meter.Float64Histogram(
		"rpc.client.duration",
		metric.WithDescription("Duration of gRPC client requests in seconds."),
		metric.WithUnit("s"))
	if err != nil {
		interceptorLogger.Error("failed to create grpc client duration histogram", clog.Err(err))
		return
	}

	// 初始化 HTTP 服务端指标
	httpServerRequests, err = meter.Int64Counter(
		"http.server.requests.count",
		metric.WithDescription("Number of HTTP requests received."))
	if err != nil {
		interceptorLogger.Error("failed to create http server requests counter", clog.Err(err))
		return
	}

	httpServerDuration, err = meter.Float64Histogram(
		"http.server.duration",
		metric.WithDescription("Duration of HTTP requests in seconds."),
		metric.WithUnit("s"))
	if err != nil {
		interceptorLogger.Error("failed to create http server duration histogram", clog.Err(err))
		return
	}

	interceptorLogger.Info("所有 metrics 仪表初始化完成")
}

// GRPCServerInterceptor 返回一个新的 gRPC 服务端拦截器，用于链路追踪和指标收集。
//
// 该拦截器会自动：
//   - 提取来自客户端的 trace context
//   - 创建新的 span 记录请求处理过程
//   - 收集请求计数和延迟指标
//   - 记录请求状态和错误信息
//   - 在请求完成时记录详细的处理日志
func GRPCServerInterceptor() grpc.UnaryServerInterceptor {
	return func(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (interface{}, error) {
		// 提取客户端传递的 trace context
		md, ok := metadata.FromIncomingContext(ctx)
		if !ok {
			md = metadata.New(nil)
		}
		ctx = otel.GetTextMapPropagator().Extract(ctx, &grpcMetadataCarrier{MD: md})

		// 创建 span
		spanCtx, span := tracer.Start(ctx, info.FullMethod,
			trace.WithSpanKind(trace.SpanKindServer),
			trace.WithAttributes(
				semconv.RPCSystemGRPC,
				semconv.RPCServiceKey.String(info.FullMethod)))
		defer span.End()

		grpcServerLogger.Debug("开始处理 gRPC 请求",
			clog.String("method", info.FullMethod))

		// 记录请求开始时间
		startTime := time.Now()

		// 执行实际的处理逻辑
		resp, err := handler(spanCtx, req)

		// 计算处理耗时
		duration := time.Since(startTime)
		statusCode := status.Code(err)

		// 记录指标
		attrs := attribute.NewSet(
			semconv.RPCSystemGRPC,
			semconv.RPCServiceKey.String(info.FullMethod),
			semconv.RPCGRPCStatusCodeKey.Int(int(statusCode)),
		)
		grpcServerRequests.Add(spanCtx, 1, metric.WithAttributeSet(attrs))
		grpcServerDuration.Record(spanCtx, duration.Seconds(), metric.WithAttributeSet(attrs))

		// 设置 span 状态
		sCode, sMsg := statusCodeToSpanStatus(statusCode)
		span.SetStatus(sCode, sMsg)
		if err != nil {
			span.RecordError(err)
		}

		// 记录请求完成日志
		logFields := []clog.Field{
			clog.String("method", info.FullMethod),
			clog.Duration("duration", duration),
			clog.String("status", statusCode.String()),
			clog.Int("status_code", int(statusCode)),
		}

		if err != nil {
			grpcServerLogger.Warn("gRPC 请求处理完成（有错误）", append(logFields, clog.Err(err))...)
		} else {
			// 根据耗时决定日志级别
			if duration > 1*time.Second {
				grpcServerLogger.Warn("gRPC 请求处理完成（耗时较长）", logFields...)
			} else {
				grpcServerLogger.Debug("gRPC 请求处理完成", logFields...)
			}
		}

		return resp, err
	}
}

// GRPCClientInterceptor 返回一个新的 gRPC 客户端拦截器，用于链路追踪和指标收集。
//
// 该拦截器会自动：
//   - 向服务端传递当前的 trace context
//   - 创建新的 span 记录请求发送过程
//   - 收集请求计数和延迟指标
//   - 记录请求状态和错误信息
//   - 在请求完成时记录详细的调用日志
func GRPCClientInterceptor() grpc.UnaryClientInterceptor {
	return func(ctx context.Context, method string, req, reply interface{}, cc *grpc.ClientConn, invoker grpc.UnaryInvoker, opts ...grpc.CallOption) error {
		// 创建 span
		spanCtx, span := tracer.Start(ctx, method,
			trace.WithSpanKind(trace.SpanKindClient),
			trace.WithAttributes(
				semconv.RPCSystemGRPC,
				semconv.RPCServiceKey.String(method)))
		defer span.End()

		// 注入 trace context 到 metadata
		md, ok := metadata.FromOutgoingContext(spanCtx)
		if !ok {
			md = metadata.New(nil)
		} else {
			md = md.Copy()
		}
		otel.GetTextMapPropagator().Inject(spanCtx, &grpcMetadataCarrier{MD: md})
		spanCtx = metadata.NewOutgoingContext(spanCtx, md)

		grpcClientLogger.Debug("开始发送 gRPC 请求",
			clog.String("method", method),
			clog.String("target", cc.Target()))

		// 记录请求开始时间
		startTime := time.Now()

		// 执行实际的请求
		err := invoker(spanCtx, method, req, reply, cc, opts...)

		// 计算请求耗时
		duration := time.Since(startTime)
		statusCode := status.Code(err)

		// 记录指标
		attrs := attribute.NewSet(
			semconv.RPCSystemGRPC,
			semconv.RPCServiceKey.String(method),
			semconv.RPCGRPCStatusCodeKey.Int(int(statusCode)),
		)
		grpcClientRequests.Add(spanCtx, 1, metric.WithAttributeSet(attrs))
		grpcClientDuration.Record(spanCtx, duration.Seconds(), metric.WithAttributeSet(attrs))

		// 设置 span 状态
		sCode, sMsg := statusCodeToSpanStatus(statusCode)
		span.SetStatus(sCode, sMsg)
		if err != nil {
			span.RecordError(err)
		}

		// 记录请求完成日志
		logFields := []clog.Field{
			clog.String("method", method),
			clog.String("target", cc.Target()),
			clog.Duration("duration", duration),
			clog.String("status", statusCode.String()),
			clog.Int("status_code", int(statusCode)),
		}

		if err != nil {
			grpcClientLogger.Warn("gRPC 请求发送完成（有错误）", append(logFields, clog.Err(err))...)
		} else {
			// 根据耗时决定日志级别
			if duration > 1*time.Second {
				grpcClientLogger.Warn("gRPC 请求发送完成（耗时较长）", logFields...)
			} else {
				grpcClientLogger.Debug("gRPC 请求发送完成", logFields...)
			}
		}

		return err
	}
}

// HTTPMiddleware 返回一个新的 Gin 中间件，用于 HTTP 请求的链路追踪和指标收集。
//
// 该中间件会自动：
//   - 提取来自客户端的 trace context
//   - 创建新的 span 记录请求处理过程
//   - 收集请求计数和延迟指标
//   - 记录请求状态和错误信息
//   - 在请求完成时记录详细的处理日志
func HTTPMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		// 提取客户端传递的 trace context
		ctx := otel.GetTextMapPropagator().Extract(c.Request.Context(),
			propagation.HeaderCarrier(c.Request.Header))

		// 创建 span
		spanCtx, span := tracer.Start(ctx, c.FullPath(),
			trace.WithSpanKind(trace.SpanKindServer),
			trace.WithAttributes(
				semconv.HTTPMethodKey.String(c.Request.Method),
				semconv.HTTPURLKey.String(c.Request.URL.String()),
				semconv.NetHostNameKey.String(c.Request.Host),
			))
		defer span.End()

		// 将 span context 注入到请求中
		c.Request = c.Request.WithContext(spanCtx)

		httpServerLogger.Debug("开始处理 HTTP 请求",
			clog.String("method", c.Request.Method),
			clog.String("path", c.FullPath()),
			clog.String("remote_addr", c.ClientIP()),
			clog.String("user_agent", c.Request.UserAgent()))

		// 记录请求开始时间
		startTime := time.Now()

		// 执行实际的处理逻辑
		c.Next()

		// 计算处理耗时
		duration := time.Since(startTime)
		statusCode := c.Writer.Status()

		// 记录指标
		attrs := attribute.NewSet(
			semconv.HTTPMethodKey.String(c.Request.Method),
			semconv.HTTPRouteKey.String(c.FullPath()),
			semconv.HTTPStatusCodeKey.Int(statusCode),
		)
		httpServerRequests.Add(spanCtx, 1, metric.WithAttributeSet(attrs))
		httpServerDuration.Record(spanCtx, duration.Seconds(), metric.WithAttributeSet(attrs))

		// 设置 span 状态
		sCode, sMsg := httpStatusCodeToSpanStatus(statusCode)
		span.SetStatus(sCode, sMsg)
		if len(c.Errors) > 0 {
			span.RecordError(c.Errors.Last().Err)
		}

		// 记录请求完成日志
		logFields := []clog.Field{
			clog.String("method", c.Request.Method),
			clog.String("path", c.FullPath()),
			clog.String("remote_addr", c.ClientIP()),
			clog.Duration("duration", duration),
			clog.Int("status_code", statusCode),
			clog.Int("response_size", c.Writer.Size()),
		}

		if len(c.Errors) > 0 {
			httpServerLogger.Warn("HTTP 请求处理完成（有错误）",
				append(logFields, clog.Err(c.Errors.Last().Err))...)
		} else if statusCode >= 400 {
			httpServerLogger.Warn("HTTP 请求处理完成（客户端错误）", logFields...)
		} else if statusCode >= 500 {
			httpServerLogger.Error("HTTP 请求处理完成（服务端错误）", logFields...)
		} else {
			// 根据耗时决定日志级别
			if duration > 1*time.Second {
				httpServerLogger.Warn("HTTP 请求处理完成（耗时较长）", logFields...)
			} else {
				httpServerLogger.Debug("HTTP 请求处理完成", logFields...)
			}
		}
	}
}

// statusCodeToSpanStatus 将 gRPC 状态码转换为 OpenTelemetry span 状态。
func statusCodeToSpanStatus(code codes.Code) (otelcodes.Code, string) {
	if code >= codes.OK && code < codes.Canceled {
		return otelcodes.Ok, ""
	}
	return otelcodes.Error, code.String()
}

// httpStatusCodeToSpanStatus 将 HTTP 状态码转换为 OpenTelemetry span 状态。
func httpStatusCodeToSpanStatus(code int) (otelcodes.Code, string) {
	if code >= 200 && code < 400 {
		return otelcodes.Ok, ""
	}
	return otelcodes.Error, ""
}

// grpcMetadataCarrier 实现了 propagation.TextMapCarrier 接口，
// 用于在 gRPC metadata 中传播 trace context。
type grpcMetadataCarrier struct {
	MD metadata.MD
}

// Get 从 metadata 中获取指定 key 的值。
func (c *grpcMetadataCarrier) Get(key string) string {
	vals := c.MD.Get(key)
	if len(vals) > 0 {
		return vals[0]
	}
	return ""
}

// Set 向 metadata 中设置指定 key 的值。
func (c *grpcMetadataCarrier) Set(key, value string) {
	c.MD.Set(key, value)
}

// Keys 返回 metadata 中所有的 key。
func (c *grpcMetadataCarrier) Keys() []string {
	keys := make([]string, 0, len(c.MD))
	for k := range c.MD {
		keys = append(keys, k)
	}
	return keys
}
