# clog

一个现代化、高性能的 Go 结构化日志库，基于 Go 1.21+ 标准库 `log/slog` 构建。clog 提供简洁、可组合的接口，支持多目标输出、动态日志级别调整、TraceID 注入和文件滚动等高级特性。

## 功能特色

- 🚀 **基于 slog**：充分利用 Go 标准库 `log/slog`，性能与兼容性俱佳
- 🎯 **接口驱动**：抽象清晰，封装合理
- 📝 **双格式支持**：支持 JSON 和文本格式输出
- 🔄 **多目标输出**：可同时输出到多个目标（stdout、stderr、文件等）
- 📁 **文件滚动**：内置日志文件滚动与压缩（依赖 lumberjack）
- 🏷️ **TraceID 集成**：自动从 context 注入 TraceID
- ⚡ **动态日志级别**：运行时可调整日志级别
- 🎨 **结构化日志**：丰富的结构化分组数据支持
- 🔧 **零依赖**：仅依赖 Go 标准库和 lumberjack（用于文件滚动）

## 安装

```bash
go get github.com/ceyewan/gochat/im-infra/clog
```

## 快速开始

### 基本用法

```go
package main

import (
    "github.com/ceyewan/gochat/im-infra/clog"
)

func main() {
    // 使用默认日志器
    logger := clog.Default()
    
    logger.Info("你好，世界！")
    logger.Warn("这是一个警告", "component", "example")
    logger.Error("这是一个错误", "error_code", 500)
}
```

### 自定义配置

```go
package main

import (
    "context"
    "github.com/ceyewan/gochat/im-infra/clog"
)

func main() {
    cfg := clog.Config{
        Level: "debug",
        Outputs: []clog.OutputConfig{
            {
                Format: "json",
                Writer: "stdout",
            },
        },
        EnableTraceID: true,
        TraceIDKey:    "request_id",
        AddSource:     true,
    }

    logger, err := clog.New(cfg)
    if err != nil {
        panic(err)
    }

    // 带 TraceID 的上下文日志
    ctx := context.WithValue(context.Background(), "request_id", "req-123")
    logger.InfoContext(ctx, "处理请求", "endpoint", "/api/users")
}
```

## 配置说明

### 配置结构体

```go
type Config struct {
    Level         string         `json:"level"`         // "debug", "info", "warn", "error"
    Outputs       []OutputConfig `json:"outputs"`       // 多个输出目标
    EnableTraceID bool           `json:"enableTraceID"` // 自动从 context 注入 TraceID
    TraceIDKey    any            `json:"traceIDKey"`    // 从 context 提取 TraceID 的 key
    AddSource     bool           `json:"addSource"`     // 是否包含源码文件/行号
}

type OutputConfig struct {
    Format       string              `json:"format"`       // "json" 或 "text"
    Writer       string              `json:"writer"`       // "stdout"、"stderr" 或 "file"
    FileRotation *FileRotationConfig `json:"fileRotation"` // 文件滚动配置（仅 file 有效）
}

type FileRotationConfig struct {
    Filename   string `json:"filename"`   // 日志文件路径
    MaxSize    int    `json:"maxSize"`    // 单文件最大 MB
    MaxAge     int    `json:"maxAge"`     // 最大保存天数
    MaxBackups int    `json:"maxBackups"` // 最大备份文件数
    LocalTime  bool   `json:"localTime"`  // 备份时间是否用本地时间
    Compress   bool   `json:"compress"`   // 是否压缩备份文件
}
```

### 默认配置

```go
cfg := clog.DefaultConfig()
// 等价于:
// Config{
//     Level: "info",
//     Outputs: []OutputConfig{
//         {Format: "text", Writer: "stdout"},
//     },
//     EnableTraceID: false,
//     AddSource: false,
// }
```

## 高级用法

### 多目标日志输出

```go
cfg := clog.Config{
    Level: "info",
    Outputs: []clog.OutputConfig{
        // 控制台文本输出
        {
            Format: "text",
            Writer: "stdout",
        },
        // 文件 JSON 输出并滚动
        {
            Format: "json",
            Writer: "file",
            FileRotation: &clog.FileRotationConfig{
                Filename:   "logs/app.log",
                MaxSize:    100, // 100MB
                MaxAge:     30,  // 30天
                MaxBackups: 10,  // 10个备份
                LocalTime:  true,
                Compress:   true,
            },
        },
    },
    EnableTraceID: true,
    TraceIDKey:    "trace_id",
    AddSource:     true,
}

logger, _ := clog.New(cfg)
logger.Info("这条消息会同时输出到控制台和文件")
```

### 结构化属性日志

```go
// 创建带持久属性的子日志器
serviceLogger := logger.With("service", "user-service", "version", "1.2.3")
serviceLogger.Info("服务启动", "port", 8080)

// 链式添加属性
userLogger := serviceLogger.With("user_id", 12345)
userLogger.Info("用户认证成功", "username", "alice")
```

### 分组日志

```go
// 创建分组日志器
dbLogger := logger.WithGroup("database")
dbLogger.Info("连接已建立", "host", "localhost", "port", 5432)
// 输出: {"database": {"host": "localhost", "port": 5432}, "msg": "连接已建立"}

// 分组与属性链式组合
apiLogger := logger.WithGroup("api").With("version", "v1")
apiLogger.Info("请求已处理", "endpoint", "/users")
// 输出: {"api": {"version": "v1", "endpoint": "/users"}, "msg": "请求已处理"}
```

### 动态日志级别控制

```go
logger := clog.Default()

logger.Info("这条会显示")
logger.Debug("这条不会显示（默认 info 级别）")

// 运行时调整级别
logger.SetLevel("debug")
logger.Debug("现在这条会显示！")

// 切换到 error 级别
logger.SetLevel("error")
logger.Info("这条不会再显示")
logger.Error("但错误日志仍会显示")
```

### TraceID 集成

```go
cfg := clog.Config{
    Level: "info",
    Outputs: []clog.OutputConfig{{Format: "json", Writer: "stdout"}},
    EnableTraceID: true,
    TraceIDKey:    "request_id", // context 中查找的 key
}

logger, _ := clog.New(cfg)

// 带 TraceID 的 context
ctx := context.WithValue(context.Background(), "request_id", "req-abc-123")
logger.InfoContext(ctx, "处理请求")
// 输出: {"request_id": "req-abc-123", "msg": "处理请求"}

// 无 TraceID 的 context
ctx2 := context.Background()
logger.InfoContext(ctx2, "另一个请求")
// 输出: {"msg": "另一个请求"}
```

## API 参考

### Logger 接口

```go
type Logger interface {
    // 基础日志方法
    Debug(msg string, args ...any)
    Info(msg string, args ...any)
    Warn(msg string, args ...any)
    Error(msg string, args ...any)
    
    // 带 context 的日志方法
    DebugContext(ctx context.Context, msg string, args ...any)
    InfoContext(ctx context.Context, msg string, args ...any)
    WarnContext(ctx context.Context, msg string, args ...any)
    ErrorContext(ctx context.Context, msg string, args ...any)
    
    // 创建子日志器
    With(args ...any) Logger
    WithGroup(name string) Logger
    
    // 动态日志级别
    SetLevel(level string) error
}
```

### 工厂方法

```go
// 使用自定义配置创建日志器
func New(cfg Config) (Logger, error)

// 创建默认日志器
func Default() Logger

// 获取默认配置
func DefaultConfig() Config
```

## 示例

详见 [examples](examples/) 目录，包含完整示例：

- [基础示例](examples/basic/main.go) - 简单用法
- [高级示例](examples/advanced/main.go) - 多输出、文件滚动、TraceID

运行示例：

```bash
go run ./examples/basic/main.go
go run ./examples/advanced/main.go
```

## 性能

clog 基于 Go 标准库 `log/slog`，具备优秀性能：

- 禁用级别时零分配
- 高效结构化数据处理
- context 操作开销极低
- JSON 与文本格式优化