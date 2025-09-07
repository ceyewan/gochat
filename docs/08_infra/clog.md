# 基础设施: clog 结构化日志

## 1. 设计理念

`clog` 是 `gochat` 项目的官方结构化日志库，基于 `uber-go/zap` 构建。它旨在提供一个**简洁、高性能、上下文感知**的日志解决方案，并强制执行结构化日志记录的最佳实践。

- **简洁易用**: API 设计简单直观，提供了方便的全局方法，极大降低了使用门槛。
- **高性能**: 基于 `zap` 的零内存分配日志记录引擎，对业务性能影响降至最低。
- **上下文感知**: 能够自动从 `context.Context` 中提取 `trace_id`，将分散的日志条目串联成完整的请求链路，是实现微服务可观测性的关键。
- **模块化**: 支持为不同的服务或业务模块创建专用的、带 `module` 字段的日志器，便于日志的分类和筛选。

## 2. 核心 API 契约

`clog` 的 API 设计兼顾了易用性（全局方法）和灵活性（可实例化的 Logger）。

### 2.1 构造函数与初始化

`clog` 支持两种初始化方式：全局初始化和独立实例创建。

```go
// Config 是 clog 组件的配置结构体。
type Config struct {
	// Level 日志级别: "debug", "info", "warn", "error", "fatal"
	Level string `json:"level"`
	// Format 输出格式: "json" (生产环境推荐) 或 "console" (开发环境推荐)
	Format string `json:"format"`
	// Output 输出目标: "stdout", "stderr", 或文件路径
	Output string `json:"output"`
    // ... 其他配置如 AddSource, EnableColor, Rotation 等
}

// Init 初始化全局默认的日志器。
// 这是最常用的方式，通常在服务的 main 函数中调用一次。
func Init(config *Config) error

// New 创建一个独立的、可自定义的 Logger 实例。
// 这在需要将日志输出到不同位置或使用不同格式的特殊场景下很有用。
func New(config *Config) (Logger, error)
```

### 2.2 Logger 接口

`Logger` 接口定义了日志记录的核心操作。全局方法和 `Module()`、`C()` 返回的实例都实现了此接口。

```go
// Logger 定义了日志记录器的核心接口。
type Logger interface {
	Debug(msg string, fields ...Field)
	Info(msg string, fields ...Field)
	Warn(msg string, fields ...Field)
	Error(msg string, fields ...Field)
	Fatal(msg string, fields ...Field) // 会导致程序退出
}

// Field 是一个强类型的键值对，用于结构化日志。
// clog 直接暴露了 zap.Field 的所有构造函数，如 clog.String, clog.Int, clog.Err 等。
type Field = zap.Field
```

### 2.3 上下文与模块化

```go
// C 从 context 中获取一个 Logger 实例。
// 如果 ctx 中包含 trace_id，返回的 Logger 会自动在每条日志中添加 "trace_id" 字段。
// 这是在处理请求的函数中进行日志记录的【首选方式】。
func C(ctx context.Context) Logger

// Module 创建一个带有 "module" 字段的 Logger 实例。
// 这对于区分不同业务模块或分层的日志非常有用。
func Module(name string) Logger
```

## 3. 标准用法

### 场景 1: 在服务启动时初始化全局 Logger

```go
// 在 main.go 中
func main() {
    // 假设 config 是从 coord 或本地文件加载的
    var clogConfig clog.Config
    // ... 加载配置 ...

    if err := clog.Init(&clogConfig); err != nil {
        log.Fatalf("初始化 clog 失败: %v", err)
    }

    clog.Info("服务启动成功", clog.String("service", "im-gateway"))
    // ...
}
```

### 场景 2: 在 gRPC 拦截器中处理 Trace ID

```go
// 在一个 gRPC 拦截器中
func TraceIDInterceptor(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (interface{}, error) {
    // 1. 从 gRPC metadata 中提取或生成一个新的 trace_id
    traceID := getTraceIDFromMetadata(ctx)
    if traceID == "" {
        traceID = uuid.NewV7()
    }

    // 2. 将 trace_id 注入到 context 中
    ctx = clog.SetTraceID(ctx, traceID)

    // 3. 调用下一个 handler
    return handler(ctx, req)
}
```

### 场景 3: 在业务逻辑中使用上下文日志

```go
// 在一个业务处理函数中
func (s *MessageService) SendMessage(ctx context.Context, req *pb.SendMessageRequest) (*pb.SendMessageResponse, error) {
    // 使用 clog.C(ctx) 获取带有 trace_id 的 logger
    logger := clog.C(ctx)

    logger.Info("开始处理发送消息请求",
        clog.String("sender_id", req.SenderID),
        clog.String("receiver_id", req.ReceiverID),
    )

    // ... 业务逻辑 ...
    if err != nil {
        logger.Error("发送消息失败", clog.Err(err))
        return nil, err
    }

    logger.Info("成功发送消息")
    return &pb.SendMessageResponse{}, nil
}