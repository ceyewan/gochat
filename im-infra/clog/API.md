# `im-infra/clog` - API 参考文档

本库为 GoChat 微服务生态提供统一的结构化日志能力。本文档详细描述了如何使用本库的公共 API。

## 1. 核心 API 概览

`clog` 提供了两种主要的交互方式：**全局日志方法**和**日志器实例**。

### 全局日志方法

为了方便快速开发，可以直接使用包级别的全局函数。

```go
// 直接记录不同级别的日志
clog.Info(msg string, fields ...clog.Field)
clog.Warn(msg string, fields ...clog.Field)
clog.Error(msg string, fields ...clog.Field)

// 记录带 context 的日志，用于自动注入 TraceID
clog.InfoContext(ctx context.Context, msg string, fields ...clog.Field)
```

### Logger 接口

所有日志器实例都实现了 `Logger` 接口，这是与日志器交互的核心。

```go
type Logger interface {
    // 基础日志方法
	Debug(msg string, fields ...Field)
	Info(msg string, fields ...Field)
	Warn(msg string, fields ...Field)
	Error(msg string, fields ...Field)

    // 带 Context 的日志方法
	DebugContext(ctx context.Context, msg string, fields ...Field)
	InfoContext(ctx context.Context, msg string, fields ...Field)
    // ...

    // 创建子日志器
	With(fields ...Field) Logger
	Module(name string) Logger

    // 动态配置
	SetLevel(level string) error
}
```

## 2. 快速上手

在你的服务中集成 `clog` 非常简单。

```go
package main

import (
	"errors"
	"github.com/ceyewan/gochat/im-infra/clog"
)

// 推荐：在包级别获取模块日志器
var (
	dbLogger  = clog.Module("database")
	apiLogger = clog.Module("api")
)

func main() {
	// 使用全局方法记录一般信息
	clog.Info("服务启动", clog.String("version", "1.0.0"))

	// 使用模块日志器记录特定领域的日志
	dbLogger.Info("连接数据库成功", clog.String("host", "localhost"))
	apiLogger.Debug("收到一个外部请求", clog.String("path", "/v1/users"))

	// 记录一个错误
	err := errors.New("something went wrong")
	apiLogger.Error("请求处理失败", clog.Err(err), clog.Int("status_code", 500))
}
```

## 3. 核心功能详解

### 自定义日志器

通过 `clog.New()` 和 `clog.Config`，你可以创建一个完全独立的日志器实例。

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
    // handle error
}

logger.Info("这是一条来自自定义日志器的消息")
```

### 模块化日志 (`Module`)

`Module(name)` 是 `With(clog.String("module", name))` 的一层封装，是组织大型项目日志的最佳实践。它可以从任何 `Logger` 实例调用。

```go
// 1. 基于全局日志器创建模块
var serviceALogger = clog.Module("service-a")

// 2. 基于自定义日志器创建模块
customLogger, _ := clog.New(cfg)
var serviceBLogger = customLogger.Module("service-b")

serviceALogger.Info("来自服务A的日志")
serviceBLogger.Info("来自服务B的日志，它将遵循 customLogger 的配置")
```

### 多目标输出

在 `Config.Outputs` 数组中定义多个输出目标。

```go
cfg.Outputs = []clog.OutputConfig{
    // 目标1: 向控制台输出 Text 格式
    {Format: "text", Writer: "stdout"},
    // 目标2: 向文件输出 JSON 格式
    {Format: "json", Writer: "file", FileRotation: &clog.FileRotationConfig{...}},
}
```

### 动态级别调整

在服务运行时，可以随时更改日志级别。

```go
logger := clog.Default()
logger.Debug("这条不会显示") // 默认是 Info 级别

err := logger.SetLevel("debug")
if err != nil {
    // handle error
}

logger.Debug("这条现在会显示了！")
```

## 4. 配置项说明 (`Config`)

| 字段 | 类型 | 描述 | 默认值 |
| :--- | :--- | :--- | :--- |
| `Level` | `string` | **(必需)** 日志级别，支持: `debug`, `info`, `warn`, `error`。 | `info` |
| `Outputs` | `[]OutputConfig` | **(必需)** 定义一个或多个输出目标。 | `stdout` + `text` |
| `EnableTraceID` | `bool` | 是否从 `context.Context` 自动注入 TraceID。 | `false` |
| `TraceIDKey` | `any` | 从 context 中提取 TraceID 的键。 | `traceID` |
| `AddSource`| `bool` | 是否在日志中包含源码文件和行号信息。 | `false` |

### `OutputConfig`

| 字段 | 类型 | 描述 |
| :--- | :--- | :--- |
| `Format` | `string` | **(必需)** 输出格式，支持: `json`, `text`。 |
| `Writer` | `string` | **(必需)** 输出目标，支持: `stdout`, `stderr`, `file`。 |
| `FileRotation` | `*FileRotationConfig`| 当 `Writer` 为 `file` 时的滚动配置。 |

### `FileRotationConfig`

| 字段 | 类型 | 描述 | 默认值 |
| :--- | :--- | :--- | :--- |
| `Filename`| `string` | **(必需)** 日志文件路径。 | - |
| `MaxSize` | `int` | 单个文件最大大小 (MB)。 | `100` |
| `MaxAge` | `int` | 日志文件最大保留天数。 | `30` |
| `MaxBackups`| `int` | 最大保留的旧日志文件数。| `10` |
| `LocalTime`| `bool` | 备份文件时间戳是否用本地时间。| `false` (UTC) |
| `Compress`| `bool` | 是否对滚动的日志文件进行 gzip 压缩。| `false` |


## 5. 最佳实践

1.  **优先使用模块日志器**：对于任何有明确归属的业务逻辑，都应使用 `clog.Module()` 创建独立的日志器进行记录，而不是直接使用全局方法。

2.  **缓存日志器实例**：无论是通过 `clog.New()` 还是 `clog.Module()` 获取的日志器，都应在包级别变量中缓存起来，避免在函数调用或循环中重复创建。

3.  **使用 `clog.Err()` 处理错误**：统一使用 `clog.Err(err)` 来记录错误信息，而不是 `clog.String("error", err.Error())`，前者能保留更完整的错误信息。

4.  **利用结构化优势**：始终使用 `clog.String()`, `clog.Int()` 等类型化辅助函数，避免使用 `fmt.Sprintf` 将变量拼接到消息字符串中。
    ```go
    // ✅ 推荐
    clog.Info("用户登录成功", clog.Int("user_id", 123), clog.String("ip", "1.2.3.4"))

    // ❌ 不推荐
    clog.Info(fmt.Sprintf("用户 %d 从 IP %s 登录成功", 123, "1.2.3.4"))
