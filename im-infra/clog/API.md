# clog API 文档

## 概述

`clog` 是一个基于 Go 标准库 `log/slog` 的高性能结构化日志库，提供了简洁易用的 API 和丰富的功能。

## 核心特性

- 🌟 **全局日志方法**：支持 `clog.Info()` 等全局方法，无需显式创建日志器
- 📦 **模块日志器**：`clog.Module("name")` 创建模块特定日志器，单例模式，配置继承
- 🚀 **基于 slog**：充分利用 Go 标准库 `log/slog`，性能与兼容性俱佳
- 📝 **双格式支持**：支持 JSON 和文本格式输出
- 🔄 **多目标输出**：可同时输出到多个目标（stdout、stderr、文件等）
- 📁 **文件滚动**：内置日志文件滚动与压缩
- 🏷️ **TraceID 集成**：自动从 context 注入 TraceID
- ⚡ **动态日志级别**：运行时可调整日志级别

## 全局日志方法

### 基础日志方法

```go
func Debug(msg string, args ...any)
func Info(msg string, args ...any)
func Warn(msg string, args ...any)
func Error(msg string, args ...any)
```

**使用示例：**
```go
clog.Debug("调试信息", "key", "value")
clog.Info("用户登录", "user_id", 12345, "username", "alice")
clog.Warn("警告信息", "component", "auth", "reason", "rate_limit")
clog.Error("错误信息", "error", err, "operation", "database_query")
```

### 带 Context 的日志方法

```go
func DebugContext(ctx context.Context, msg string, args ...any)
func InfoContext(ctx context.Context, msg string, args ...any)
func WarnContext(ctx context.Context, msg string, args ...any)
func ErrorContext(ctx context.Context, msg string, args ...any)
```

**使用示例：**
```go
ctx := context.WithValue(context.Background(), "trace_id", "req-123")
clog.InfoContext(ctx, "处理请求", "endpoint", "/api/users")
clog.ErrorContext(ctx, "请求失败", "error", err, "status_code", 500)
```

## 模块日志器

### Module 函数

```go
func Module(name string) Logger
```

创建或获取指定名称的模块日志器。对于相同的模块名，返回相同的日志器实例（单例模式）。

**特性：**
- 单例缓存：相同模块名返回相同实例
- 配置继承：继承默认日志器的所有配置
- 线程安全：支持并发访问
- 自动标识：自动添加 `module` 字段

**使用示例：**
```go
// 创建模块日志器
dbLogger := clog.Module("database")
apiLogger := clog.Module("api")
authLogger := clog.Module("auth")

// 使用模块日志器
dbLogger.Info("连接已建立", "host", "localhost", "port", 5432)
// 输出: time=... level=INFO msg="连接已建立" module=database host=localhost port=5432

apiLogger.Error("请求失败", "endpoint", "/users", "status", 500)
// 输出: time=... level=ERROR msg="请求失败" module=api endpoint=/users status=500
```

**性能建议：**
```go
// ✅ 推荐：缓存模块日志器
var logger = clog.Module("service")
logger.Info("message")

// ❌ 避免：重复调用 Module()
clog.Module("service").Info("message") // 有额外开销
```

## 传统 API（向后兼容）

### Default 函数

```go
func Default() Logger
```

返回默认日志器实例，与全局日志方法使用相同的日志器。

**使用示例：**
```go
logger := clog.Default()
logger.Info("Hello, World!")
logger.Warn("This is a warning", "component", "example")
```

### New 函数

```go
func New(cfg Config) (Logger, error)
```

使用自定义配置创建新的日志器实例。

**使用示例：**
```go
cfg := clog.Config{
    Level: "debug",
    Outputs: []clog.OutputConfig{
        {
            Format: "json",
            Writer: "stdout",
        },
    },
    EnableTraceID: true,
    TraceIDKey:    "trace_id",
    AddSource:     true,
}

logger, err := clog.New(cfg)
if err != nil {
    return err
}

logger.Debug("Debug message")
logger.Info("Application started", "version", "1.0.0")
```

## Logger 接口

```go
type Logger interface {
    // 基础日志方法
    Debug(msg string, args ...any)
    Info(msg string, args ...any)
    Warn(msg string, args ...any)
    Error(msg string, args ...any)
    
    // 带 Context 的日志方法
    DebugContext(ctx context.Context, msg string, args ...any)
    InfoContext(ctx context.Context, msg string, args ...any)
    WarnContext(ctx context.Context, msg string, args ...any)
    ErrorContext(ctx context.Context, msg string, args ...any)
    
    // 结构化日志
    With(args ...any) Logger
    WithGroup(name string) Logger
    
    // 动态配置
    SetLevel(level string) error
    
    // 启用/禁用功能
    Enabled(ctx context.Context, level slog.Level) bool
}
```

### 结构化日志

```go
// 添加固定属性
userLogger := logger.With("user_id", 12345, "session", "abc123")
userLogger.Info("用户操作", "action", "login")
// 输出: ... user_id=12345 session=abc123 action=login

// 创建分组（已弃用，推荐使用 Module）
dbLogger := logger.WithGroup("database")
dbLogger.Info("查询执行", "table", "users", "duration", "150ms")
// 输出: ... database.table=users database.duration=150ms
```

### 动态级别调整

```go
logger := clog.Default()

// 检查当前级别
if logger.Enabled(ctx, slog.LevelDebug) {
    logger.Debug("Debug message")
}

// 动态调整级别
err := logger.SetLevel("debug")
if err != nil {
    clog.Error("Failed to set level", "error", err)
}
```

## 配置

### Config 结构

```go
type Config struct {
    Level         string         `json:"level"`          // 日志级别: debug, info, warn, error
    Outputs       []OutputConfig `json:"outputs"`        // 输出配置列表
    EnableTraceID bool           `json:"enable_trace_id"` // 是否启用 TraceID
    TraceIDKey    any            `json:"trace_id_key"`   // TraceID 在 context 中的键
    AddSource     bool           `json:"add_source"`     // 是否添加源码位置信息
}
```

### OutputConfig 结构

```go
type OutputConfig struct {
    Format       string               `json:"format"`        // 输出格式: text, json
    Writer       string               `json:"writer"`        // 输出目标: stdout, stderr, file
    FileRotation *FileRotationConfig  `json:"file_rotation"` // 文件滚动配置（仅当 writer=file 时）
}
```

### FileRotationConfig 结构

```go
type FileRotationConfig struct {
    Filename   string `json:"filename"`    // 日志文件路径
    MaxSize    int    `json:"max_size"`    // 最大文件大小（MB）
    MaxAge     int    `json:"max_age"`     // 最大保留天数
    MaxBackups int    `json:"max_backups"` // 最大备份文件数
    LocalTime  bool   `json:"local_time"`  // 是否使用本地时间
    Compress   bool   `json:"compress"`    // 是否压缩备份文件
}
```

## 字段辅助函数

```go
// 创建任意类型的字段
func Any(key string, value any) Field

// 创建字符串字段
func String(key, value string) Field

// 创建整数字段
func Int(key string, value int) Field
func Int64(key string, value int64) Field

// 创建浮点数字段
func Float64(key string, value float64) Field

// 创建布尔字段
func Bool(key string, value bool) Field

// 创建时间字段
func Time(key string, value time.Time) Field

// 创建持续时间字段
func Duration(key string, value time.Duration) Field

// 创建错误字段
func Err(err error) Field           // 使用 "error" 作为键名
func ErrorValue(err error) Field    // 创建 error 类型字段（重命名后的函数）
```

**使用示例：**
```go
clog.Info("操作完成",
    clog.String("operation", "user_create"),
    clog.Int("user_id", 12345),
    clog.Duration("elapsed", time.Since(start)),
    clog.Bool("success", true),
)

// 或者直接使用键值对
clog.Info("操作完成",
    "operation", "user_create",
    "user_id", 12345,
    "elapsed", time.Since(start),
    "success", true,
)
```

## 最佳实践

### 1. 选择合适的日志方法

```go
// ✅ 简单场景：使用全局方法
clog.Info("应用启动", "version", "1.0.0")

// ✅ 模块化场景：使用模块日志器
var dbLogger = clog.Module("database")
dbLogger.Info("连接建立", "host", "localhost")

// ✅ 复杂配置：使用自定义日志器
logger, _ := clog.New(customConfig)
logger.Info("自定义日志器")
```

### 2. 性能优化

```go
// ✅ 缓存模块日志器
var (
    dbLogger  = clog.Module("database")
    apiLogger = clog.Module("api")
)

func handleRequest() {
    dbLogger.Info("查询数据")  // 无额外开销
    apiLogger.Info("处理请求") // 无额外开销
}

// ❌ 避免重复调用
func handleRequest() {
    clog.Module("database").Info("查询数据")  // 有额外开销
    clog.Module("api").Info("处理请求")       // 有额外开销
}
```

### 3. 结构化日志

```go
// ✅ 使用结构化字段
clog.Info("用户登录",
    "user_id", 12345,
    "username", "alice",
    "ip", "192.168.1.100",
    "user_agent", "Mozilla/5.0...",
)

// ❌ 避免在消息中嵌入变量
clog.Info(fmt.Sprintf("用户 %s (ID: %d) 登录", username, userID))
```

### 4. 错误处理

```go
// ✅ 使用 Err 辅助函数
if err != nil {
    clog.Error("操作失败", clog.Err(err), "operation", "user_create")
}

// ✅ 或者直接使用键值对
if err != nil {
    clog.Error("操作失败", "error", err, "operation", "user_create")
}
```

### 5. Context 使用

```go
// ✅ 传递 context 以支持 TraceID
func handleRequest(ctx context.Context) {
    clog.InfoContext(ctx, "开始处理请求")
    
    // 业务逻辑...
    
    clog.InfoContext(ctx, "请求处理完成", "duration", time.Since(start))
}
```

## 迁移指南

### 从 WithGroup 迁移到 Module

```go
// 旧方式（已弃用）
dbLogger := logger.WithGroup("database")
dbLogger.Info("连接建立")

// 新方式（推荐）
dbLogger := clog.Module("database")
dbLogger.Info("连接建立")
```

### 从传统方式迁移到全局方法

```go
// 旧方式
logger := clog.Default()
logger.Info("消息")

// 新方式（推荐）
clog.Info("消息")
```

## 常见问题

### Q: 全局方法和模块日志器的区别？
A: 全局方法适用于简单场景，模块日志器适用于需要区分不同组件的场景。模块日志器会自动添加 `module` 字段。

### Q: 模块日志器是否线程安全？
A: 是的，模块日志器使用读写锁保护，完全线程安全。

### Q: 如何选择输出格式？
A: 开发环境推荐使用 `text` 格式（易读），生产环境推荐使用 `json` 格式（易于解析和分析）。

### Q: 如何处理敏感信息？
A: 避免在日志中记录密码、令牌等敏感信息。如需记录，请先进行脱敏处理。

### Q: 性能如何？
A: 基于 `log/slog`，性能优异。模块日志器查找开销约 6ns，建议缓存使用以获得最佳性能。
