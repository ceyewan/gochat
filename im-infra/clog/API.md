# `im-infra/clog` - API 参考文档

本文档详细描述了 `clog` 简化后的公共 API。`clog` 专为内部使用设计，提供简洁、高性能的结构化日志能力。

## 🎯 设计原则

- **极简接口**：只暴露 8 个核心函数，覆盖 99% 的使用场景
- **零配置**：内置生产环境最佳实践，开箱即用
- **高性能**：模块日志器缓存，基于高性能的 `slog` 实现
- **微服务友好**：原生支持模块化日志分类

## 📋 完整 API 列表

### 核心函数

```go
// 日志器创建
func New() Logger                                    // 创建独立日志器实例
func Module(name string) Logger                      // 创建模块日志器（带缓存）

// 全局日志方法
func Debug(msg string, fields ...Field)              // Debug 级别
func Info(msg string, fields ...Field)              // Info 级别
func Warn(msg string, fields ...Field)              // Warn 级别
func Error(msg string, fields ...Field)             // Error 级别

// 带 Context 的全局方法（自动注入 TraceID）
func DebugContext(ctx context.Context, msg string, fields ...Field)
func InfoContext(ctx context.Context, msg string, fields ...Field)
func WarnContext(ctx context.Context, msg string, fields ...Field)
func ErrorContext(ctx context.Context, msg string, fields ...Field)
```

### Logger 接口

```go
type Logger interface {
    // 基础日志方法
    Debug(msg string, fields ...Field)
    Info(msg string, fields ...Field)
    Warn(msg string, fields ...Field)
    Error(msg string, fields ...Field)

    // 带 Context 的方法（自动注入 TraceID）
    DebugContext(ctx context.Context, msg string, fields ...Field)
    InfoContext(ctx context.Context, msg string, fields ...Field)
    WarnContext(ctx context.Context, msg string, fields ...Field)
    ErrorContext(ctx context.Context, msg string, fields ...Field)

    // 扩展方法
    With(fields ...Field) Logger        // 添加通用字段
    Module(name string) Logger          // 创建子模块日志器
}
```

## 🚀 基础使用

### 1. 全局日志方法

最简单的使用方式，适合快速开发和通用日志记录：

```go
package main

import "github.com/ceyewan/gochat/im-infra/clog"

func main() {
    // 基础日志记录
    clog.Info("服务启动成功", clog.String("version", "1.0.0"))
    clog.Warn("配置文件缺失，使用默认配置")
    clog.Error("数据库连接失败", clog.Err(err), clog.Int("retry_count", 3))

    // 带 Context 的日志（自动注入 TraceID）
    ctx := context.WithValue(context.Background(), "trace_id", "req-123")
    clog.InfoContext(ctx, "处理用户请求", clog.String("user_id", "alice"))
}
```

**输出示例**：
```json
{
  "time": "2024-01-15T10:30:45.123Z",
  "level": "INFO",
  "source": {"function": "main.main", "file": "main.go", "line": 8},
  "msg": "服务启动成功",
  "version": "1.0.0"
}
```

### 2. 模块化日志

推荐用于微服务架构，实现日志的模块化分类：

```go
package main

import "github.com/ceyewan/gochat/im-infra/clog"

// 在包级别缓存模块日志器（最佳性能）
var (
    dbLogger   = clog.Module("database")
    apiLogger  = clog.Module("api")
    authLogger = clog.Module("auth")
)

func handleLogin() {
    // 每条日志自动带有 "module": "auth" 字段
    authLogger.Info("用户登录请求",
        clog.String("username", "alice"),
        clog.String("ip", "192.168.1.100"))

    // 数据库操作，自动带有 "module": "database" 字段
    dbLogger.Info("查询用户信息",
        clog.String("query", "SELECT * FROM users"),
        clog.Int("rows", 1))

    // API 响应，自动带有 "module": "api" 字段
    apiLogger.Info("登录成功", clog.String("user_id", "12345"))
}
```

**输出示例**：
```json
{
  "time": "2024-01-15T10:30:45.456Z",
  "level": "INFO",
  "source": {"function": "main.handleLogin", "file": "main.go", "line": 15},
  "msg": "用户登录请求",
  "module": "auth",
  "username": "alice",
  "ip": "192.168.1.100"
}
```

### 3. 创建独立实例

适合需要独立配置或隔离的场景：

```go
package main

import "github.com/ceyewan/gochat/im-infra/clog"

func main() {
    // 创建独立的日志器实例
    logger := clog.New()

    // 添加通用字段
    serviceLogger := logger.With(
        clog.String("service", "user-service"),
        clog.String("version", "2.1.0"),
        clog.String("environment", "production"))

    serviceLogger.Info("服务初始化完成")

    // 从实例创建模块日志器
    dbModule := serviceLogger.Module("database")
    dbModule.Info("连接池初始化", clog.Int("pool_size", 10))
}
```

## 📊 Field 构造函数

`clog` 提供了完整的类型化 Field 构造函数，确保类型安全和最佳性能。

### 基础类型

```go
clog.String(key, value string) Field                // 字符串
clog.Int(key string, value int) Field               // 整数
clog.Int32(key string, value int32) Field           // 32位整数
clog.Int64(key string, value int64) Field           // 64位整数
clog.Uint(key string, value uint) Field             // 无符号整数
clog.Bool(key string, value bool) Field             // 布尔值
clog.Float32(key string, value float32) Field       // 32位浮点数
clog.Float64(key string, value float64) Field       // 64位浮点数
```

### 时间相关

```go
clog.Time(key string, value time.Time) Field        // 时间
clog.Duration(key string, value time.Duration) Field // 时长
```

### 特殊类型

```go
clog.Err(err error) Field                           // 错误（推荐用法）
clog.Any(key string, value any) Field              // 任意类型
clog.Binary(key string, value []byte) Field        // 二进制数据
clog.Stringer(key string, value fmt.Stringer) Field // 实现String()接口的类型
```

### 数组类型

```go
clog.Strings(key string, values []string) Field    // 字符串数组
clog.Ints(key string, values []int) Field          // 整数数组
```

**使用示例**：

```go
clog.Info("用户操作",
    clog.String("action", "login"),
    clog.Int("user_id", 12345),
    clog.Bool("success", true),
    clog.Duration("response_time", 150*time.Millisecond),
    clog.Time("timestamp", time.Now()),
    clog.Strings("roles", []string{"user", "admin"}))
```

## 🔧 内置配置

`clog` 使用经过生产环境验证的默认配置：

| 配置项 | 值 | 说明 |
|-------|-----|------|
| **Level** | `"info"` | 平衡性能和信息量的最佳级别 |
| **Format** | `"json"` | 便于日志收集系统处理和分析 |
| **Writer** | `"stdout"` | 标准输出，配合容器日志收集 |
| **EnableTraceID** | `true` | 微服务追踪必备功能 |
| **TraceIDKey** | `"trace_id"` | 标准化的 TraceID 字段名 |
| **AddSource** | `true` | 包含源码信息，便于开发调试 |

### 日志输出格式

每条日志包含完整的结构化信息：

```json
{
  "time": "2024-01-15T10:30:45.123456789Z",     // 时间戳
  "level": "INFO",                               // 日志级别
  "source": {                                    // 源码信息
    "function": "main.handleRequest",
    "file": "handler.go",
    "line": 42
  },
  "msg": "处理用户请求",                          // 日志消息
  "module": "api",                               // 模块名（通过 Module() 添加）
  "user_id": "12345",                           // 业务字段
  "request_id": "req-789",
  "trace_id": "trace-abc-123"                    // 自动注入的 TraceID
}
```

## ⚡ 性能优化

### 模块日志器缓存

`clog.Module()` 内置缓存机制，相同模块名返回同一实例：

```go
// ✅ 推荐：包级别缓存，零开销
var dbLogger = clog.Module("database")

func queryUser() {
    dbLogger.Info("查询用户")  // 直接使用缓存实例
}

// ❌ 避免：每次调用都创建
func queryUser() {
    clog.Module("database").Info("查询用户")  // 每次都查缓存，有开销
}
```

### With 方法复用

使用 `With()` 方法复用带通用字段的日志器：

```go
// ✅ 推荐：复用通用字段
serviceLogger := clog.New().With(
    clog.String("service", "payment"),
    clog.String("version", "1.0"))

serviceLogger.Info("开始处理")    // 自动包含 service 和 version
serviceLogger.Info("处理完成")    // 自动包含 service 和 version

// ❌ 避免：重复添加相同字段
clog.Info("开始处理", clog.String("service", "payment"), clog.String("version", "1.0"))
clog.Info("处理完成", clog.String("service", "payment"), clog.String("version", "1.0"))
```

## 🏆 最佳实践

### 1. 模块化组织

为每个业务模块创建专门的日志器：

```go
package database

import "github.com/ceyewan/gochat/im-infra/clog"

var logger = clog.Module("database")

func Connect() error {
    logger.Info("开始连接数据库", clog.String("host", "localhost"))
    // ...
}
```

### 2. 错误处理

统一使用 `clog.Err()` 记录错误：

```go
// ✅ 推荐：完整的错误信息
if err := db.Connect(); err != nil {
    clog.Error("数据库连接失败",
        clog.Err(err),                    // 完整错误信息
        clog.String("operation", "connect"),
        clog.Int("retry_count", 3))
}

// ❌ 不推荐：丢失错误堆栈
clog.Error("数据库连接失败", clog.String("error", err.Error()))
```

### 3. Context 传递

充分利用 Context 传递 TraceID：

```go
func handleRequest(ctx context.Context, req *Request) {
    // 自动注入 TraceID
    clog.InfoContext(ctx, "开始处理请求",
        clog.String("request_id", req.ID))

    // 传递 context 到下游服务
    if err := callDownstream(ctx, req); err != nil {
        clog.ErrorContext(ctx, "下游调用失败", clog.Err(err))
        return
    }

    clog.InfoContext(ctx, "请求处理完成")
}
```

### 4. 结构化字段

避免字符串拼接，使用结构化字段：

```go
// ✅ 推荐：结构化，便于查询和分析
clog.Info("用户登录",
    clog.String("user_id", userID),
    clog.String("username", username),
    clog.String("ip", clientIP),
    clog.Int("login_count", count))

// ❌ 不推荐：非结构化，难以查询
clog.Info(fmt.Sprintf("用户 %s (ID: %s) 从 %s 第 %d 次登录",
    username, userID, clientIP, count))
```

### 5. 日志级别选择

- **Info**：正常业务流程、重要状态变更
- **Warn**：异常情况但不影响正常流程
- **Error**：错误情况、需要关注的问题

```go
// Info: 正常业务流程
clog.Info("用户注册成功", clog.String("user_id", userID))

// Warn: 异常但可恢复
clog.Warn("缓存未命中，使用数据库查询", clog.String("key", cacheKey))

// Error: 错误需要关注
clog.Error("支付失败", clog.Err(err), clog.String("order_id", orderID))
```

## 🔄 迁移指南

### 从复杂配置迁移

```go
// 之前：复杂配置
cfg := clog.Config{
    Level: "info",
    Outputs: []clog.OutputConfig{
        {Format: "json", Writer: "stdout"},
    },
    EnableTraceID: true,
    TraceIDKey: "trace_id",
    AddSource: true,
}
logger, err := clog.New(cfg)

// 现在：零配置
logger := clog.New()  // 内置相同的最佳实践配置
```

### 从其他日志库迁移

```go
// 从标准库 log 迁移
log.Printf("User %s login from %s", userID, ip)
// 👇
clog.Info("用户登录", clog.String("user_id", userID), clog.String("ip", ip))

// 从 logrus 迁移
logrus.WithFields(logrus.Fields{"user_id": userID}).Info("User login")
// 👇
clog.Info("用户登录", clog.String("user_id", userID))
```

## 📊 总结

`clog` 通过简化设计实现了极致的易用性：

- **8 个函数**覆盖所有常用场景
- **零配置**开箱即用
- **高性能**缓存优化
- **完整输出**包含所有必要信息

从复杂到简单，`clog` 让开发者专注于业务逻辑而不是日志配置。
