# `im-infra/clog` - GoChat 简化高性能日志库

`clog` 是一个专为内部使用设计的现代化 Go 结构化日志库，基于 Go 1.21+ 标准库 `log/slog` 构建。它为 GoChat 微服务生态提供了简化、高性能且易于使用的日志记录解决方案。

## 🎯 设计理念

### 为什么选择简化设计？

在复杂的微服务架构中，日志库应该是"透明"的工具——强大但不复杂，高效但不繁琐。传统的日志库往往提供过多的配置选项和接口，增加了学习成本和使用复杂度。

**`clog` 采用"约定优于配置"的设计哲学**：

- 🎯 **接口极简**：只暴露 8 个核心函数，5 分钟上手
- ⚡ **性能优先**：内置最佳实践配置，模块日志器缓存优化
- 🔧 **开箱即用**：无需配置，直接使用生产环境优化的默认设置
- 🏗️ **微服务友好**：Module 功能原生支持微服务架构的日志分类

## ✨ 核心特性

- 🚀 **基于 `slog`**：享受 Go 官方标准库的高性能和零依赖优势
- 📊 **结构化日志**：完整的 Field 类型支持，便于日志分析和查询
- 🏷️ **智能模块化**：`Module()` 支持微服务日志分类，带缓存优化
- 🔄 **TraceID 注入**：自动从 `context.Context` 提取并注入 TraceID
- 📁 **完整信息**：timestamp、level、module、msg、fields、source 一应俱全
- ⚡ **高性能缓存**：模块日志器缓存机制，避免重复创建

## 🚀 快速上手

### 基础使用

```go
package main

import (
    "context"
    "github.com/ceyewan/gochat/im-infra/clog"
)

func main() {
    // 1. 直接使用全局方法（最简单）
    clog.Info("服务启动成功", clog.String("version", "1.0.0"))
    clog.Warn("配置文件不存在，使用默认配置", clog.String("file", "config.yaml"))
    clog.Error("数据库连接失败", clog.Err(err), clog.Int("retry", 3))

    // 2. 带 Context 的日志（自动注入 TraceID）
    ctx := context.WithValue(context.Background(), "trace_id", "req-123")
    clog.InfoContext(ctx, "用户请求处理完成", 
        clog.String("user_id", "12345"), 
        clog.Int("status_code", 200))
}
```

### 模块化日志（推荐）

```go
package main

import "github.com/ceyewan/gochat/im-infra/clog"

// 在包级别缓存模块日志器（最佳性能）
var (
    dbLogger   = clog.Module("database")
    apiLogger  = clog.Module("api")
    authLogger = clog.Module("auth")
)

func handleUserLogin() {
    // 每条日志自动带有 "module": "auth" 字段
    authLogger.Info("用户登录请求", 
        clog.String("username", "alice"),
        clog.String("ip", "192.168.1.100"))
    
    // 数据库操作日志自动带有 "module": "database" 字段
    dbLogger.Info("查询用户信息", 
        clog.String("sql", "SELECT * FROM users WHERE username = ?"),
        clog.Int("execution_time_ms", 45))
    
    // API 响应日志自动带有 "module": "api" 字段
    apiLogger.Info("登录成功", 
        clog.String("user_id", "12345"),
        clog.Int("response_time_ms", 123))
}
```

### 创建独立实例

```go
package main

import "github.com/ceyewan/gochat/im-infra/clog"

func main() {
    // 创建独立的日志器实例（使用相同的最佳实践配置）
    logger := clog.New()
    
    // 添加通用字段
    serviceLogger := logger.With(
        clog.String("service", "user-service"),
        clog.String("version", "2.1.0"))
    
    serviceLogger.Info("服务初始化完成")
    
    // 创建该服务的模块日志器
    dbModule := serviceLogger.Module("database")
    dbModule.Info("数据库连接池初始化", clog.Int("pool_size", 10))
}
```

## 📋 完整 API 参考

### 核心函数

```go
// 创建日志器
func New() Logger                                    // 创建独立的日志器实例
func Module(name string) Logger                      // 创建模块日志器（带缓存）

// 全局日志方法
func Info(msg string, fields ...Field)              // Info 级别日志
func Warn(msg string, fields ...Field)              // Warn 级别日志  
func Error(msg string, fields ...Field)             // Error 级别日志

// 带 Context 的全局方法（自动注入 TraceID）
func InfoContext(ctx context.Context, msg string, fields ...Field)
func WarnContext(ctx context.Context, msg string, fields ...Field)
func ErrorContext(ctx context.Context, msg string, fields ...Field)
```

### Logger 接口

```go
type Logger interface {
    Info(msg string, fields ...Field)
    Warn(msg string, fields ...Field)
    Error(msg string, fields ...Field)
    
    InfoContext(ctx context.Context, msg string, fields ...Field)
    WarnContext(ctx context.Context, msg string, fields ...Field)
    ErrorContext(ctx context.Context, msg string, fields ...Field)
    
    With(fields ...Field) Logger        // 添加通用字段
    Module(name string) Logger          // 创建子模块日志器
}
```

### Field 构造函数

```go
// 基础类型
func String(key, value string) Field
func Int(key string, value int) Field
func Bool(key string, value bool) Field
func Float64(key string, value float64) Field

// 时间相关
func Time(key string, value time.Time) Field
func Duration(key string, value time.Duration) Field

// 特殊类型
func Err(err error) Field                           // 错误处理（推荐）
func Any(key string, value any) Field              // 任意类型
func Strings(key string, values []string) Field     // 字符串数组

// 更多类型：Int32, Int64, Uint, Uint32, Uint64, Float32, Binary 等
```

## 🔧 内置配置说明

`clog` 使用生产环境优化的默认配置，无需用户配置：

```yaml
Level: "info"                    # 平衡性能和信息量
Format: "json"                   # 便于日志收集系统处理
Writer: "stdout"                 # 标准输出，配合容器日志收集
EnableTraceID: true              # 微服务追踪必备
TraceIDKey: "trace_id"           # 标准化的 TraceID 字段名
AddSource: true                  # 包含源码信息，便于调试
```

### 日志输出格式

每条日志包含完整的结构化信息：

```json
{
  "time": "2024-01-15T10:30:45.123456789Z",     // timestamp
  "level": "INFO",                               // level
  "source": {                                    // source
    "function": "main.handleRequest", 
    "file": "main.go", 
    "line": 42
  },
  "msg": "处理用户请求",                          // msg
  "module": "api",                               // module (通过 Module() 添加)
  "user_id": "12345",                           // fields
  "request_id": "req-789",
  "trace_id": "trace-abc-123"                    // 自动注入的 TraceID
}
```

## 📈 性能优化

### 模块日志器缓存

```go
// ✅ 推荐：包级别缓存，零开销复用
var dbLogger = clog.Module("database")

func queryUser() {
    dbLogger.Info("查询用户")  // 直接使用缓存的实例
}

// ❌ 避免：每次调用都创建，有性能开销
func queryUser() {
    clog.Module("database").Info("查询用户")  // 每次都查缓存
}
```

### 字段复用

```go
// ✅ 推荐：复用带通用字段的日志器
serviceLogger := clog.New().With(
    clog.String("service", "user-service"),
    clog.String("version", "1.0.0"))

serviceLogger.Info("操作A完成")  // 自动包含 service 和 version 字段
serviceLogger.Info("操作B完成")  // 自动包含 service 和 version 字段
```

## 🏆 最佳实践

1. **优先使用模块日志器**：为每个业务模块创建专门的日志器
   ```go
   var dbLogger = clog.Module("database")
   var redisLogger = clog.Module("redis")
   ```

2. **包级别缓存日志器**：避免在热路径上重复创建
   ```go
   // ✅ 好的做法
   var logger = clog.Module("payment")
   
   func processPayment() {
       logger.Info("开始处理支付")
   }
   ```

3. **统一使用 clog.Err()**：标准化错误日志记录
   ```go
   // ✅ 推荐
   clog.Error("操作失败", clog.Err(err), clog.String("operation", "payment"))
   
   // ❌ 不推荐
   clog.Error("操作失败", clog.String("error", err.Error()))
   ```

4. **充分利用结构化字段**：避免字符串拼接
   ```go
   // ✅ 推荐：结构化，便于查询分析
   clog.Info("用户登录", clog.String("user_id", userID), clog.String("ip", clientIP))
   
   // ❌ 不推荐：非结构化，难以查询
   clog.Info(fmt.Sprintf("用户 %s 从 %s 登录", userID, clientIP))
   ```

5. **善用 Context 传递 TraceID**：实现链路追踪
   ```go
   func handleRequest(ctx context.Context) {
       clog.InfoContext(ctx, "开始处理请求")  // 自动注入 TraceID
       
       // 传递 context 到下游
       result, err := callDownstream(ctx)
       if err != nil {
           clog.ErrorContext(ctx, "下游调用失败", clog.Err(err))
       }
   }
   ```

## 🔄 从复杂配置迁移

如果你之前使用复杂的配置，迁移非常简单：

```go
// 之前：复杂配置
cfg := clog.Config{
    Level: "info",
    Outputs: []clog.OutputConfig{...},
    EnableTraceID: true,
    // ... 更多配置
}
logger, err := clog.New(cfg)

// 现在：零配置
logger := clog.New()  // 内置最佳实践配置
```

## 🎯 总结

`clog` 通过简化设计实现了"少即是多"的哲学：

- **8 个核心函数**解决 99% 的日志需求
- **零配置**开箱即用，内置生产环境最佳实践
- **高性能**模块缓存和 slog 底层优化
- **微服务友好**的模块化设计

从复杂到简单，从配置到约定，`clog` 让你专注于业务逻辑，而不是日志配置。