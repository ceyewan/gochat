# clog 日志库文档

## 概述

clog 是一个基于 zap 封装的高性能日志库，提供了简洁的 API 和强大的功能。

## 文档目录

### 📚 核心文档

- **[CallerSkip 问题修复文档](./CALLER_SKIP_FIX.md)** - 详细记录了 CallerSkip 问题的分析和解决过程
- **[参考实现](./REFERENCE_IMPLEMENTATION.md)** - 展示了 CallerSkip 的正确实现方式
- **[故障排除指南](./TROUBLESHOOTING.md)** - 快速诊断和解决 CallerSkip 相关问题

### 🚀 快速开始

#### 基本使用

```go
package main

import (
    "context"
    "github.com/ceyewan/gochat/im-infra/clog"
)

func main() {
    // 1. 全局日志方法
    clog.Info("服务启动", clog.String("version", "1.0.0"))
    clog.Error("连接失败", clog.String("host", "localhost"))

    // 2. 模块化日志
    userLogger := clog.Module("user")
    userLogger.Info("用户登录", clog.String("userID", "123"))

    // 3. Context 日志（自动注入 TraceID）
    ctx := context.WithValue(context.Background(), "traceID", "trace-001")
    clog.C(ctx).Info("处理请求", clog.String("action", "login"))

    // 4. 链式调用
    clog.C(ctx).Module("order").Info("创建订单", clog.String("orderID", "order-456"))
}
```

#### 输出示例

```bash
2025-07-27 22:46:29.010	INFO	main.go:11	服务启动	{"version": "1.0.0"}
2025-07-27 22:46:29.010	ERROR	main.go:12	连接失败	{"host": "localhost"}
2025-07-27 22:46:29.010	INFO	main.go:16	用户登录	{"module": "user", "userID": "123"}
2025-07-27 22:46:29.010	INFO	main.go:20	处理请求	{"traceID": "trace-001", "action": "login"}
2025-07-27 22:46:29.011	INFO	main.go:23	创建订单	{"traceID": "trace-001", "module": "order", "orderID": "order-456"}
```
### 🌟 最佳实践：依赖注入 (Best Practice: Dependency Injection)

虽然 `clog.Info` 这样的全局函数在 `main` 包或简单脚本中非常方便，但对于构建健壮、可测试和可维护的应用程序，我们**强烈推荐使用依赖注入（Dependency Injection）**的方式来传递 `Logger` 实例。

直接依赖全局日志记录器会使代码与全局状态紧密耦合，导致以下问题：
- **可测试性差**: 单元测试时难以模拟（mock）日志行为，无法断言日志是否被正确调用。
- **依赖不明确**: 函数或结构体对日志的依赖是隐式的，不够清晰。
- **灵活性低**: 无法为应用的不同部分轻松提供不同配置的日志实例。

#### Before: 依赖全局日志 (不推荐)
```go
package user

import "github.com/ceyewan/gochat/im-infra/clog"

type Service struct {
    // ... other dependencies
}

func (s *Service) CreateUser(name string) {
    // ... business logic
    clog.Module("user").Info("User created", clog.String("name", name))
}
```

#### After: 使用依赖注入 (推荐)
通过构造函数将 `clog.Logger` 注入到您的服务中。

```go
package user

import "github.com/ceyewan/gochat/im-infra/clog"

// Logger 定义了 Service 所需的日志接口，便于测试
type Logger interface {
    Info(msg string, fields ...clog.Field)
    Error(msg string, fields ...clog.Field)
}

type Service struct {
    logger Logger // 依赖接口，而非具体实现
    // ... other dependencies
}

// NewService 构造函数接收一个 Logger 实例
func NewService(logger clog.Logger) *Service {
    return &Service{
        // 为这个 service 的所有日志自动添加 "module" 字段
        logger: logger.Module("user-service"),
    }
}

func (s *Service) CreateUser(name string) {
    // ... business logic
    s.logger.Info("User created", clog.String("name", name))
}
```

在您的测试代码中，您可以轻松传入一个模拟的 logger：
```go
type mockLogger struct {
    // ...
}
func (m *mockLogger) Info(msg string, fields ...clog.Field) { /* ... */ }
func (m *mockLogger) Error(msg string, fields ...clog.Field) { /* ... */ }

func TestUserService(t *testing.T) {
    mock := &mockLogger{}
    service := NewService(mock)
    // ... run test
}
```

这种方法让您的代码更加模块化、清晰且易于测试。

### 🔧 配置

#### 默认配置

```go
// 使用 clog.New() 创建一个独立的 logger 实例。
// 这是推荐的方式，特别是在需要将 logger 作为依赖注入时。
logger, err := clog.New()
if err != nil {
    // 处理错误
}
// 使用 logger ...
```

#### 自定义配置

```go
// 通过传递配置给 clog.New() 来创建 logger
config := clog.Config{
    Level:       "info",
    Format:      "json",        // 或 "console"
    Output:      "stdout",      // 或文件路径
    AddSource:   true,          // 显示调用位置
    EnableColor: false,         // 控制台彩色输出
    Rotation: &clog.RotationConfig{
        MaxSize:    100,        // MB
        MaxBackups: 3,
        MaxAge:     7,          // 天
        Compress:   true,
    },
}

logger, err := clog.New(config)
if err != nil {
    // 处理错误
}
```

#### 初始化全局 Logger (可选)

```go
// 对于简单的应用或为了兼容旧代码，可以初始化全局 logger
// Init 内部会调用 New()
err := clog.Init(config)
if err != nil {
    // 处理错误
}

// 现在可以全局调用
clog.Info("全局 logger 初始化完成")
```

### 🎯 核心特性

#### 1. 准确的调用位置显示

✅ **修复前的问题**：
```bash
INFO    runtime/proc.go:283     消息内容
INFO    runtime/asm_arm64.s:1223    消息内容
```

✅ **修复后的效果**：
```bash
INFO    main.go:11    消息内容
INFO    user_service.go:45    消息内容
```

#### 2. 自动 TraceID 注入

clog 会自动从 `context.Context` 中查找并注入 TraceID。它会按顺序查找以下常用的 key：
- `traceID` (最常用)
- `trace_id`
- `TraceID`
- `X-Trace-ID`
- `trace-id`
- `TRACE_ID`

#### 3. 模块化日志

```go
// 创建模块日志器
userModule := clog.Module("user")
orderModule := clog.Module("order")

// 使用模块日志器
userModule.Info("用户操作")    // 自动添加 {"module": "user"}
orderModule.Error("订单错误")  // 自动添加 {"module": "order"}
```

#### 4. 高性能设计

- 基于 zap 的零分配日志
- 支持字段缓存和复用
- 模块日志器缓存机制

### 🐛 常见问题

#### Q: 日志显示 runtime 位置而不是我的代码位置？

A: 这是 CallerSkip 设置问题，请参考 [故障排除指南](./TROUBLESHOOTING.md)

#### Q: 不同调用方式显示的位置不一致？

A: 不同调用方式的调用栈深度不同，需要设置不同的 CallerSkip 值。详见 [CallerSkip 修复文档](./CALLER_SKIP_FIX.md)

#### Q: 如何自定义 TraceID 提取逻辑？

A: 使用 `clog.SetTraceIDHook()` 设置自定义提取函数：

```go
clog.SetTraceIDHook(func(ctx context.Context) (string, bool) {
    // 自定义提取逻辑
    if val := ctx.Value("custom-trace-id"); val != nil {
        return val.(string), true
    }
    return "", false
})
```

### 📊 性能建议

#### 1. 使用模块日志器缓存

```go
// 好的做法：复用模块日志器
var userLogger = clog.Module("user")

func handleUser() {
    userLogger.Info("处理用户请求")
}
```

#### 2. 使用 With 方法预设字段

```go
// 好的做法：预设常用字段
serviceLogger := clog.Module("user-service").With(
    clog.String("version", "2.1.0"),
    clog.String("instance", "srv-001"),
)

serviceLogger.Info("服务启动")
serviceLogger.Error("服务错误")
```

#### 3. 避免频繁创建 Context 日志器

```go
// 好的做法：在请求开始时创建，然后传递
func handleRequest(ctx context.Context) {
    logger := clog.C(ctx).Module("api")
    
    logger.Info("请求开始")
    // ... 处理逻辑
    logger.Info("请求完成")
}
```

### 🧪 测试

运行测试用例：

```bash
# 基本功能测试
go run im-infra/clog/examples/basic/main.go

# CallerSkip 测试
go run im-infra/clog/examples/caller_test/main.go

# TraceID 测试
go run im-infra/clog/examples/trace_test/main.go
```

### 📈 版本历史

- **v1.1.0** - 修复 CallerSkip 问题，所有调用位置现在都能正确显示
- **v1.0.0** - 初始版本，基础功能实现

### 🤝 贡献

如果你发现问题或有改进建议，请：

1. 查阅相关文档
2. 创建 Issue 描述问题
3. 提交 Pull Request

### 📄 许可证

本项目采用 MIT 许可证。
