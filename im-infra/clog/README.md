# clog - 简洁的结构化日志库

clog 是一个基于 zap 的简洁日志库，专为 GoChat 项目设计。

## 🚀 快速开始

### 基本使用

```go
import "github.com/ceyewan/gochat/im-infra/clog"

// 使用全局日志器
clog.Info("用户登录成功", clog.String("user_id", "12345"))
clog.Warn("连接超时", clog.Int("timeout", 30))
clog.Error("数据库连接失败", clog.Err(err))
clog.Fatal("致命错误，程序退出", clog.String("reason", "配置错误"))
```

### 模块化日志

```go
// 创建模块日志器
logger := clog.Module("user-service")
logger.Info("处理用户请求", clog.String("action", "create"))
logger.Error("用户创建失败", clog.Err(err))
```

### 带上下文的日志

```go
// 自动提取 TraceID，用于链路追踪
ctx := context.WithValue(context.Background(), "traceID", "abc123")
clog.C(ctx).Info("处理请求", clog.String("method", "POST"))
```

### 自定义配置

```go
// 使用自定义配置
config := clog.Config{
    Level:       "debug",
    Format:      "json",
    Output:      "/app/logs/app.log",
    AddSource:   true,
    EnableColor: false,
}

// 初始化全局日志器
err := clog.Init(config)
if err != nil {
    log.Fatal(err)
}

// 或创建独立的日志器
logger, err := clog.New(config)
if err != nil {
    log.Fatal(err)
}
```

## 📋 API 参考

### 全局日志方法

```go
clog.Debug(msg, fields...)   // 调试信息
clog.Info(msg, fields...)    // 一般信息  
clog.Warn(msg, fields...)    // 警告信息
clog.Error(msg, fields...)   // 错误信息
clog.Fatal(msg, fields...)   // 致命错误（会退出程序）
```

### 字段构造函数

```go
clog.String(key, value)      // 字符串字段
clog.Int(key, value)         // 整数字段
clog.Bool(key, value)        // 布尔字段
clog.Float64(key, value)     // 浮点数字段
clog.Duration(key, value)    // 时间间隔字段
clog.Time(key, value)        // 时间字段
clog.Err(err)                // 错误字段
clog.Any(key, value)         // 任意类型字段
```

### 实用方法

```go
clog.Module(name)            // 创建模块日志器
clog.C(ctx)                  // 创建带上下文的日志器
clog.Init(config)            // 初始化全局日志器
clog.New(config)             // 创建新的日志器实例
```

## ⚙️ 配置选项

```go
type Config struct {
    Level       string           // 日志级别: debug, info, warn, error
    Format      string           // 输出格式: json, console
    Output      string           // 输出目标: stdout, stderr, 或文件路径
    AddSource   bool             // 是否包含源码位置
    EnableColor bool             // 是否启用颜色（仅 console 格式）
    RootPath    string           // 项目根路径（用于简化文件路径显示）
    Rotation    *RotationConfig  // 日志轮转配置（可选）
}

type RotationConfig struct {
    MaxSize    int  // 单个文件最大尺寸(MB)
    MaxBackups int  // 最多保留文件个数
    MaxAge     int  // 日志保留天数
    Compress   bool // 是否压缩轮转文件
}
```

## 📝 使用示例

### 1. 基础日志记录

```go
// 简单消息
clog.Info("服务启动")

// 带字段的消息
clog.Info("用户操作", 
    clog.String("user_id", "12345"),
    clog.String("action", "login"),
    clog.Duration("duration", time.Since(start)))

// 错误日志
if err != nil {
    clog.Error("操作失败", clog.Err(err))
}
```

### 2. 模块化日志

```go
// 为不同模块创建专用日志器
userLogger := clog.Module("user-service")
authLogger := clog.Module("auth-service")

userLogger.Info("用户创建", clog.String("user_id", "123"))
authLogger.Warn("登录失败", clog.String("reason", "密码错误"))
```

### 3. 文件输出配置

```go
config := clog.Config{
    Level:    "info",
    Format:   "json",
    Output:   "/app/logs/app.log",
    Rotation: &clog.RotationConfig{
        MaxSize:    100,  // 100MB
        MaxBackups: 3,    // 保留3个备份
        MaxAge:     7,    // 保留7天
        Compress:   true, // 压缩旧文件
    },
}

clog.Init(config)
```

### 4. 开发环境配置

```go
// 开发环境：控制台输出，带颜色
devConfig := clog.Config{
    Level:       "debug",
    Format:      "console",
    Output:      "stdout",
    AddSource:   true,
    EnableColor: true,
    RootPath:    "gochat",
}

clog.Init(devConfig)
```

### 5. 生产环境配置

```go
// 生产环境：JSON 格式，文件输出
prodConfig := clog.Config{
    Level:    "info",
    Format:   "json",
    Output:   "/var/log/gochat/app.log",
    AddSource: false,
    Rotation: &clog.RotationConfig{
        MaxSize:    500,
        MaxBackups: 10,
        MaxAge:     30,
        Compress:   true,
    },
}

clog.Init(prodConfig)
```

## 🎯 设计理念

- **简洁优先**：API 简单直观，学习成本低
- **配置灵活**：支持用户传入配置，无配置时使用合理默认值
- **性能优化**：基于高性能的 zap 库
- **结构化日志**：强制使用结构化字段，便于日志分析
- **模块化支持**：支持为不同模块创建专用日志器
- **上下文感知**：自动提取 TraceID 等上下文信息

## 🔧 最佳实践

1. **使用结构化字段**
   ```go
   // ✅ 推荐
   clog.Info("用户登录", clog.String("user_id", userID))
   
   // ❌ 不推荐
   clog.Info(fmt.Sprintf("用户 %s 登录", userID))
   ```

2. **为不同模块创建专用日志器**
   ```go
   var logger = clog.Module("user-service")
   
   func CreateUser() {
       logger.Info("创建用户", clog.String("user_id", "123"))
   }
   ```

3. **在错误处理中使用 Err 字段**
   ```go
   if err != nil {
       clog.Error("操作失败", clog.Err(err))
       return err
   }
   ```

4. **使用上下文传递 TraceID**
   ```go
   func HandleRequest(ctx context.Context) {
       clog.C(ctx).Info("处理请求开始")
       // ... 处理逻辑
       clog.C(ctx).Info("处理请求完成")
   }
   ```

这个日志库专注于简洁和实用，避免了过度设计，完全满足 GoChat 项目的日志需求。