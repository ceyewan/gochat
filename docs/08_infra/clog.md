# 基础设施: clog 结构化日志

## 1. 设计哲学

`clog` 是 `gochat` 项目的官方结构化日志库，基于 `uber-go/zap` 构建。它旨在提供一个**简洁、高性能、易于使用**的日志解决方案，同时强制执行结构化日志记录的最佳实践。

- **简洁优先**: API 设计简单直观，学习成本低。
- **性能至上**: 基于 `zap` 的零内存分配日志记录。
- **上下文感知**: 自动从 `context.Context` 提取 `trace_id`，简化链路追踪。
- **模块化支持**: 轻松为不同服务或模块创建专用的、带 `module` 字段的日志器。

## 2. 核心用法

### 2.1 全局日志

最简单的使用方式，直接调用全局方法。

```go
import "github.com/ceyewan/gochat/im-infra/clog"

// 直接使用全局日志器
clog.Info("用户登录成功", clog.String("user_id", "12345"))
clog.Warn("连接超时", clog.Int("timeout_ms", 3000))

// 记录错误
if err != nil {
    clog.Error("数据库连接失败", clog.Err(err))
}
```

### 2.2 模块化日志

为不同的服务或业务模块创建专用的日志器，会自动添加 `module` 字段。

```go
// 在包级别或初始化时创建模块日志器
var userLogger = clog.Module("user-service")
var authLogger = clog.Module("auth-service")

func handleUserCreation() {
    userLogger.Info("开始创建用户", clog.String("username", "test"))
    // ...
}

func handleLogin() {
    authLogger.Warn("登录失败", clog.String("reason", "密码错误"))
    // ...
}
```

### 2.3 上下文日志 (Context Logger)

这是**强烈推荐**在处理请求的函数中使用的方式。它会自动从 `context` 中提取 `trace_id` 并添加到日志中。

```go
func handleRequest(ctx context.Context) {
    // ctx 中应包含 "trace_id"
    // clog.C(ctx) 会返回一个带有 "trace_id" 字段的日志器
    logger := clog.C(ctx)

    logger.Info("处理请求开始")
    // ... 业务逻辑 ...
    logger.Info("处理请求完成")
}
```

## 3. API 参考

### 3.1 日志级别

```go
clog.Debug(msg string, fields ...clog.Field)
clog.Info(msg string, fields ...clog.Field)
clog.Warn(msg string, fields ...clog.Field)
clog.Error(msg string, fields ...clog.Field)
clog.Fatal(msg string, fields ...clog.Field) // 会导致程序退出
```

### 3.2 常用字段类型

`clog` 直接暴露了 `zap` 的强类型字段构造函数，以实现零性能开销。

```go
clog.String(key string, value string)
clog.Int(key string, value int)
clog.Int64(key string, value int64)
clog.Bool(key string, value bool)
clog.Float64(key string, value float64)
clog.Duration(key string, value time.Duration)
clog.Time(key string, value time.Time)
clog.Err(err error) // 特别推荐，会正确处理 nil error
clog.Any(key string, value interface{}) // 性能较低，谨慎使用
```

### 3.3 初始化与配置

通常，服务在启动时会调用 `clog.Init()` 来应用从配置中心加载的配置。

```go
// Init 初始化全局日志器，通常在 main 函数中调用
func Init(cfg Config) error

// New 创建一个独立的日志器实例，而不是修改全局实例
func New(cfg Config) (Logger, error)
```

## 4. 配置

`clog` 的配置通过一个 `Config` 结构体定义，通常由 `coord` 配置中心管理。

### 4.1 配置结构

```go
type Config struct {
    // Level 日志级别: "debug", "info", "warn", "error", "fatal"
	Level string `json:"level" yaml:"level"`
    // Format 输出格式: "json" (生产环境推荐) 或 "console" (开发环境推荐)
	Format string `json:"format" yaml:"format"`
    // Output 输出目标: "stdout", "stderr", 或文件路径
	Output string `json:"output" yaml:"output"`
    // AddSource 是否在日志中包含源码文件名和行号
	AddSource bool `json:"addSource" yaml:"addSource"`
    // EnableColor 是否为 console 格式启用颜色
	EnableColor bool `json:"enableColor" yaml:"enableColor"`
    // Rotation 日志轮转配置 (可选)
	Rotation *RotationConfig `json:"rotation,omitempty" yaml:"rotation,omitempty"`
}

type RotationConfig struct {
    MaxSize    int  // 单个日志文件最大尺寸(MB)
	MaxBackups int  // 最多保留的旧文件个数
	MaxAge     int  // 日志保留天数
	Compress   bool // 是否压缩旧文件
}
```

### 4.2 生产环境推荐配置

```yaml
# 示例: clog.yaml
level: "info"
format: "json"
output: "/var/log/gochat/app.log" # 或 stdout/stderr
addSource: false
rotation:
  maxSize: 100    # 100 MB
  maxBackups: 5
  maxAge: 7       # 7 天
  compress: true
```

### 4.3 开发环境推荐配置

```yaml
# 示例: clog.yaml
level: "debug"
format: "console"
output: "stdout"
addSource: true
enableColor: true