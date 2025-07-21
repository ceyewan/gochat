# clog API 参考

本文档提供了 `clog` 日志库的核心 API 参考，旨在帮助开发者快速查找和使用。

## 1. 全局函数

这些函数通过全局单例提供服务，是项目中最常用的日志记录方式。

### 1.1. 初始化

| 函数签名 | 描述 |
| :--- | :--- |
| `Init(opts ...Option) error` | 初始化全局日志器。必须在应用启动时调用一次。 |

**示例:**
```go
err := clog.Init(clog.WithLevel("debug"))
if err != nil {
    panic(err)
}
```

### 1.2. 日志记录

| 函数签名 | 描述 |
| :--- | :--- |
| `Debug(msg string, fields ...Field)` | 记录 Debug 级别日志。 |
| `Info(msg string, fields ...Field)` | 记录 Info 级别日志。 |
| `Warn(msg string, fields ...Field)` | 记录 Warn 级别日志。 |
| `Error(msg string, fields ...Field)` | 记录 Error 级别日志。 |
| `Fatal(msg string, fields ...Field)` | 记录 Fatal 级别日志并退出应用。 |
| `Debugf(format string, args ...any)` | 记录格式化的 Debug 日志。 |
| `Infof(format string, args ...any)` | 记录格式化的 Info 日志。 |
| `Warnf(format string, args ...any)` | 记录格式化的 Warn 日志。 |
| `Errorf(format string, args ...any)` | 记录格式化的 Error 日志。 |
| `Fatalf(format string, args ...any)` | 记录格式化的 Fatal 日志并退出。 |

### 1.3. 模块化

| 函数签名 | 描述 |
| :--- | :--- |
| `Module(name string, opts ...Option) Logger` | 获取或创建一个模块专用的日志器。 |

**示例:**
```go
dbLogger := clog.Module("database")
dbLogger.Info("数据库连接成功")
```

### 1.4. 管理

| 函数签名 | 描述 |
| :--- | :--- |
| `Sync() error` | 刷新默认日志器的缓冲区。建议在 `main` 函数退出前调用。 |
| `SyncAll() error` | 刷新所有已注册日志器的缓冲区。 |
| `SetDefaultLevel(level string) error` | 动态调整默认日志器的级别。 |
| `GetLogger(name string) Logger` | 按名称获取已注册的日志器。 |

## 2. 配置选项 (Option)

通过 `Init` 或 `Module` 函数的 `opts` 参数传入。

| 选项函数 | 描述 |
| :--- | :--- |
| `WithLevel(string)` | 设置日志级别 (`debug`, `info`, `warn`, `error`, `fatal`)。 |
| `WithFormat(string)` | 设置控制台输出格式 (`console`, `json`)。 |
| `WithFilename(string)` | 设置日志输出文件名，启用文件日志。 |
| `WithFileFormat(string)` | 单独设置文件日志的格式 (`console`, `json`)。 |
| `WithConsoleOutput(bool)` | 是否输出到控制台 (默认 `true`)。 |
| `WithEnableCaller(bool)` | 是否记录调用者信息 (默认 `true`)。 |
| `WithEnableColor(bool)` | 是否为控制台输出启用颜色 (默认 `true`)。 |
| `WithTraceID(string)` | 添加一个跟踪 ID 到日志中。 |
| `WithInitialFields(...Field)` | 添加全局固定字段。 |
| `WithFileRotation(*FileRotationConfig)` | 设置文件轮转策略。 |

## 3. 结构化字段 (Field)

推荐使用结构化字段代替字符串拼接。

| 字段函数 | 描述 |
| :--- | :--- |
| `String(key, val string)` | 字符串字段。 |
| `Int(key, val int)` | 整数字段。 |
| `Int64(key, val int64)` | 64位整数字段。 |
| `Uint64(key, val uint64)` | 64位无符号整数字段。 |
| `Float64(key, val float64)` | 浮点数字段。 |
| `Bool(key, val bool)` | 布尔字段。 |
| `Err(err error)` | 错误字段，键为 `error`。 |
| `Time(key, val time.Time)` | 时间字段。 |
| `Duration(key, val time.Duration)` | 时长字段。 |
| `Any(key, val any)` | 任意类型字段，可能涉及反射。 |

## 4. 核心类型

### 4.1. `Logger` (接口)

`clog.Logger` 是一个接口，提供了与全局函数对应的实例方法。`Module()` 和 `GetLogger()` 返回此接口类型。

```go
type Logger interface {
    Debug(msg string, fields ...Field)
    Info(msg string, fields ...Field)
    // ... 其他日志方法
    SetLevel(level string) error
    Sync() error
    Close() error
    GetConfig() *Config
}
```

### 4.2. `Config`

日志配置结构体，通常通过 `Option` 函数进行配置。

```go
type Config struct {
    Level         string
    Format        string
    Filename      string
    // ... 其他字段
}
```

### 4.3. `FileRotationConfig`

文件轮转配置。

```go
type FileRotationConfig struct {
    MaxSize    int  // 单个日志文件最大尺寸(MB)
    MaxBackups int  // 最多保留文件个数
    MaxAge     int  // 日志保留天数
    Compress   bool // 是否压缩轮转文件
}