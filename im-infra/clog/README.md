# clog - 高性能结构化日志库

clog 是一个基于 zap 的高性能 Go 日志库，专为生产环境设计。它解决了常见日志库的痛点，提供准确的调用位置显示、自动 TraceID 注入和灵活的路径控制。

## ✨ 核心特性

### 🎯 精确的 CallerSkip 管理
解决了日志库中调用位置显示错误的问题，确保每种调用方式都显示正确的源码位置。

```bash
# ❌ 其他库常见问题
INFO    runtime/proc.go:283     消息内容
INFO    internal/logger.go:115  消息内容

# ✅ clog 正确显示
INFO    main.go:11              消息内容
INFO    user_service.go:45      消息内容
```

### 🔗 自动 TraceID 注入
从 `context.Context` 中自动提取 TraceID，支持多种常用格式：
- `traceID` (推荐) • `trace_id` • `TraceID` • `X-Trace-ID` • `request-id`

### 📁 智能路径控制 (RootPath)
通过 `RootPath` 配置控制文件路径显示：

```bash
# 默认显示（最后两层）
INFO    examples/main.go:10     消息

# 设置 RootPath="gochat" 后
INFO    im-infra/clog/examples/main.go:10    消息

# RootPath 不匹配时显示绝对路径
INFO    /full/path/to/file.go:10    消息
```

### 🎨 双格式支持
- **Console 格式**：开发环境友好，支持彩色输出
- **JSON 格式**：生产环境首选，便于日志收集和分析

### 📦 模块化日志
内置模块支持，自动添加模块标识，便于日志分类和过滤。

### ⚙️ 配置中心集成
- **通用配置管理器**：基于 coord 的通用配置管理器，类型安全且功能完整
- **降级策略**：配置中心不可用时自动使用默认配置
- **热更新**：支持配置热更新和实时监听
- **安全更新**：内置配置验证和回滚机制

## 🚀 快速开始

### 安装
```bash
go get github.com/ceyewan/gochat/im-infra/clog
```

### 基础使用
```go
package main

import (
    "context"
    "github.com/ceyewan/gochat/im-infra/clog"
)

func main() {
    // 1. 基础日志
    clog.Info("服务启动", clog.String("version", "1.0.0"))
    
    // 2. 模块日志
    userModule := clog.Module("user")
    userModule.Info("用户登录", clog.String("userID", "123"))
    
    // 3. Context 日志（自动 TraceID）
    ctx := context.WithValue(context.Background(), "traceID", "abc-123")
    clog.C(ctx).Info("处理请求", clog.String("action", "login"))
    
    // 4. 链式调用
    clog.C(ctx).Module("order").Info("创建订单", clog.String("orderID", "456"))
}
```

### 输出示例

**Console 格式**：
```bash
2025-07-28 21:19:07.597	INFO	main.go:11	服务启动	{"version": "1.0.0"}
2025-07-28 21:19:07.598	INFO	main.go:15	用户登录	{"module": "user", "userID": "123"}
2025-07-28 21:19:07.598	INFO	main.go:19	处理请求	{"traceID": "abc-123", "action": "login"}
2025-07-28 21:19:07.598	INFO	main.go:22	创建订单	{"traceID": "abc-123", "module": "order", "orderID": "456"}
```

**JSON 格式**：
```json
{"level":"info","time":"2025-07-28 21:19:07.597","caller":"main.go:11","msg":"服务启动","version":"1.0.0"}
{"level":"info","time":"2025-07-28 21:19:07.598","caller":"main.go:15","msg":"用户登录","module":"user","userID":"123"}
{"level":"info","time":"2025-07-28 21:19:07.598","caller":"main.go:19","msg":"处理请求","traceID":"abc-123","action":"login"}
{"level":"info","time":"2025-07-28 21:19:07.598","caller":"main.go:22","msg":"创建订单","traceID":"abc-123","module":"order","orderID":"456"}
```

## 📖 配置详解

### 生产环境配置
```go
config := clog.Config{
    Level:    "info",
    Format:   "json",
    Output:   "/var/log/app.log",
    RootPath: "myproject",  // 路径控制
    Rotation: &clog.RotationConfig{
        MaxSize:    100,  // 100MB
        MaxBackups: 10,   // 保留10个文件  
        MaxAge:     30,   // 保留30天
        Compress:   true,
    },
}

clog.Init(config)
```

### 开发环境配置
```go
config := clog.Config{
    Level:       "debug",
    Format:      "console",
    Output:      "stdout",
    AddSource:   true,
    EnableColor: true,
    RootPath:    "gochat",
}

logger, err := clog.New(config)
```

### 自定义 TraceID Hook
```go
clog.SetTraceIDHook(func(ctx context.Context) (string, bool) {
    // 自定义 TraceID 提取逻辑
    if val := ctx.Value("custom-trace-id"); val != nil {
        return val.(string), true
    }
    return "", false
})
```

### 配置中心集成（两阶段启动）
```go
package main

import (
    "github.com/ceyewan/gochat/im-infra/clog"
    "github.com/ceyewan/gochat/im-infra/coord"
)

func main() {
    // 阶段一：降级启动 - 使用默认配置确保基础日志功能可用
    clog.Info("应用启动", clog.String("stage", "fallback"))

    // 创建协调器
    coordinator, err := coord.New()
    if err != nil {
        panic(err)
    }
    defer coordinator.Close()

    // 阶段二：配置中心集成 - 从配置中心获取配置并支持热更新
    clog.SetupConfigCenterFromCoord(coordinator.Config(), "prod", "im-infra", "clog")

    // 重新初始化，使用配置中心的配置
    err = clog.Init()
    if err != nil {
        // 如果配置中心不可用，会继续使用当前配置，不会中断服务
        clog.Warn("配置中心不可用，继续使用当前配置", clog.Err(err))
    }

    clog.Info("配置中心集成完成", clog.String("stage", "config-center"))
}
```

## 🏗️ 最佳实践

### 依赖注入模式（推荐）
```go
type UserService struct {
    logger clog.Logger
}

func NewUserService(logger clog.Logger) *UserService {
    return &UserService{
        logger: logger.Module("user"),
    }
}

func (s *UserService) CreateUser(name string) {
    s.logger.Info("创建用户", clog.String("name", name))
}
```

### 全局使用模式（简单场景）
```go
func main() {
    clog.Init(clog.Config{
        Level:  "info",
        Format: "console",
        Output: "stdout",
    })
    
    clog.Info("应用启动")
}
```

## 📚 文档

- **[API 文档](docs/API.md)** - 完整的 API 参考
- **[示例代码](examples/)** - 基础和高级使用示例

## 🔧 配置参数

| 参数 | 类型 | 默认值 | 说明 |
|------|------|--------|------|
| `Level` | string | "info" | 日志级别：debug, info, warn, error |
| `Format` | string | "console" | 输出格式：console, json |
| `Output` | string | "stdout" | 输出目标：stdout, stderr 或文件路径 |
| `AddSource` | bool | true | 是否显示调用位置 |
| `EnableColor` | bool | true | 控制台是否启用颜色 |
| `RootPath` | string | "" | 项目根路径，用于路径截取 |
| `Rotation` | *RotationConfig | nil | 日志轮转配置 |

## 🚀 性能特性

- **零分配日志**：基于 zap 的高性能设计
- **模块缓存**：自动缓存模块 Logger，避免重复创建
- **智能 CallerSkip**：精确的调用栈管理，无性能损失
- **结构化字段**：高效的字段序列化

## 📄 许可证

MIT License - 详见 [LICENSE](LICENSE) 文件

## 🤝 贡献

欢迎提交 Issue 和 Pull Request！

---

**clog** - 让日志记录更简单、更准确、更高效 🚀
