# clog - GoChat 结构化日志库

clog 是为 GoChat 项目设计的结构化日志库，基于 uber-go/zap 构建。提供**简洁、高性能、上下文感知**的日志解决方案，完全遵循 GoChat 项目的开发规范。

## 🚀 快速开始

### 服务初始化

```go
import (
    "context"
    "github.com/ceyewan/gochat/im-infra/clog"
)

// 使用环境相关的默认配置初始化
config := clog.GetDefaultConfig("production")
if err := clog.Init(context.Background(), config, clog.WithNamespace("im-gateway")); err != nil {
    log.Fatal(err)
}

clog.Info("服务启动成功")
// 输出: {"namespace": "im-gateway", "msg": "服务启动成功"}
```

### 基本使用

```go
// 使用全局日志器
clog.Info("用户登录成功", clog.String("user_id", "12345"))
clog.Warn("连接超时", clog.Int("timeout", 30))
clog.Error("数据库连接失败", clog.Err(err))
clog.Fatal("致命错误，程序退出", clog.String("reason", "配置错误"))
```

### 层次化命名空间

```go
// 支持链式调用的层次化命名空间
userLogger := clog.Namespace("user")
authLogger := userLogger.Namespace("auth")
dbLogger := userLogger.Namespace("database")

userLogger.Info("开始用户注册流程", clog.String("email", "user@example.com"))
// 输出: {"namespace": "user", "msg": "开始用户注册流程", "email": "user@example.com"}

authLogger.Info("验证用户密码强度")
// 输出: {"namespace": "user.auth", "msg": "验证用户密码强度"}

dbLogger.Info("检查用户是否已存在")
// 输出: {"namespace": "user.database", "msg": "检查用户是否已存在"}
```

### 上下文感知日志

```go
// 在中间件中注入 TraceID
ctx := clog.WithTraceID(context.Background(), "abc123-def456")

// 业务代码中自动获取带 TraceID 的 logger
logger := clog.WithContext(ctx)
logger.Info("处理请求", clog.String("method", "POST"))
// 输出: {"trace_id": "abc123-def456", "msg": "处理请求", "method": "POST"}

// 简短别名形式
clog.C(ctx).Info("处理请求完成")
```

### Provider 模式创建独立日志器

```go
// 使用 Provider 模式创建独立的日志器实例
config := &clog.Config{
    Level:       "debug",
    Format:      "json",
    Output:      "/app/logs/app.log",
    AddSource:   true,
    EnableColor: false,
}

logger, err := clog.New(context.Background(), config, clog.WithNamespace("payment-service"))
if err != nil {
    log.Fatal(err)
}

logger.Info("独立日志器初始化成功")
```

## 📋 API 参考

### Provider 模式接口

```go
// 标准 Provider 签名，完全遵循 im-infra 组件规范
func New(ctx context.Context, config *Config, opts ...Option) (Logger, error)
func Init(ctx context.Context, config *Config, opts ...Option) error
func GetDefaultConfig(env string) *Config
```

### 全局日志方法

```go
clog.Debug(msg, fields...)   // 调试信息
clog.Info(msg, fields...)    // 一般信息  
clog.Warn(msg, fields...)    // 警告信息
clog.Error(msg, fields...)   // 错误信息
clog.Fatal(msg, fields...)   // 致命错误（会退出程序）
```

### 层次化命名空间

```go
// 创建命名空间日志器，支持链式调用
clog.Namespace(name) Logger

// 示例：链式创建深层命名空间
logger := clog.Namespace("payment").Namespace("processor").Namespace("stripe")
```

### 上下文感知日志

```go
// 类型安全的 TraceID 注入
func WithTraceID(ctx context.Context, traceID string) context.Context

// 从 context 获取带 TraceID 的 logger
func WithContext(ctx context.Context) Logger

// 简短别名
func C(ctx context.Context) Logger
```

### 功能选项

```go
// 设置根命名空间
func WithNamespace(name string) Option
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

## ⚙️ 配置选项

```go
type Config struct {
    Level       string           // 日志级别: debug, info, warn, error, fatal
    Format      string           // 输出格式: json (生产环境推荐) 或 console (开发环境推荐)
    Output      string           // 输出目标: stdout, stderr, 或文件路径
    AddSource   bool             // 是否包含源码文件名和行号
    EnableColor bool             // 是否启用颜色（仅 console 格式）
    RootPath    string           // 项目根目录，用于控制文件路径显示
    Rotation    *RotationConfig  // 日志轮转配置（仅文件输出）
}

type RotationConfig struct {
    MaxSize    int  // 单个日志文件最大尺寸(MB)
    MaxBackups int  // 最多保留文件个数
    MaxAge     int  // 日志保留天数
    Compress   bool // 是否压缩轮转文件
}
```

### 环境感知默认配置

```go
// 开发环境：console 格式，debug 级别，带颜色
devConfig := clog.GetDefaultConfig("development")
// 返回: &Config{Level: "debug", Format: "console", EnableColor: true, ...}

// 生产环境：json 格式，info 级别，无颜色  
prodConfig := clog.GetDefaultConfig("production")
// 返回: &Config{Level: "info", Format: "json", EnableColor: false, ...}
```

## 📝 使用示例

### 1. 服务初始化（推荐方式）

```go
func main() {
    // 使用环境相关的默认配置
    config := clog.GetDefaultConfig("production")
    
    // 初始化全局 logger，设置服务命名空间
    if err := clog.Init(context.Background(), config, clog.WithNamespace("im-gateway")); err != nil {
        log.Fatal(err)
    }
    
    clog.Info("服务启动成功")
}
```

### 2. Gin 中间件集成

```go
func TraceMiddleware() gin.HandlerFunc {
    return func(c *gin.Context) {
        // 获取或生成 traceID
        traceID := c.GetHeader("X-Trace-ID")
        if traceID == "" {
            traceID = uuid.NewString()
        }
        
        // 注入 traceID 到 context
        ctx := clog.WithTraceID(c.Request.Context(), traceID)
        c.Request = c.Request.WithContext(ctx)
        
        c.Header("X-Trace-ID", traceID)
        c.Next()
    }
}

func handler(c *gin.Context) {
    // 自动获取带 traceID 的 logger
    logger := clog.WithContext(c.Request.Context())
    logger.Info("处理请求", clog.String("path", c.Request.URL.Path))
}
```

### 3. 层次化命名空间使用

```go
func (s *PaymentService) ProcessPayment(ctx context.Context, req *PaymentRequest) error {
    // 自动获取带 traceID 的 logger
    logger := clog.WithContext(ctx)
    
    // 使用层次化命名空间
    validationLogger := logger.Namespace("validation")
    processorLogger := logger.Namespace("processor").Namespace("stripe")
    
    logger.Info("开始处理支付", clog.String("order_id", req.OrderID))
    validationLogger.Info("验证支付数据")
    processorLogger.Info("调用 Stripe API")
    
    return nil
}
```

### 4. 文件输出与轮转

```go
config := &clog.Config{
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

if err := clog.Init(context.Background(), config); err != nil {
    log.Fatal(err)
}
```

### 5. 创建独立日志器

```go
// 为特定模块创建独立的日志器
paymentLogger, err := clog.New(context.Background(), &clog.Config{
    Level:  "debug",
    Format: "json",
    Output: "/app/logs/payment.log",
}, clog.WithNamespace("payment-service"))

if err != nil {
    log.Fatal(err)
}

paymentLogger.Info("支付服务日志器初始化成功")
```

### 6. 上下文传递的最佳实践

```go
func processUserRequest(ctx context.Context, userID string) error {
    // 始终从 context 获取 logger，自动包含 traceID
    logger := clog.WithContext(ctx)
    
    logger.Info("开始处理用户请求", clog.String("user_id", userID))
    
    // 在子函数中也传递 context
    if err := validateUser(ctx, userID); err != nil {
        logger.Error("用户验证失败", clog.Err(err))
        return err
    }
    
    logger.Info("用户请求处理完成")
    return nil
}

func validateUser(ctx context.Context, userID string) error {
    // 使用更具体的命名空间
    logger := clog.WithContext(ctx).Namespace("validation")
    logger.Info("验证用户信息", clog.String("user_id", userID))
    // ... 验证逻辑
    return nil
}
```

## 🎯 设计理念

- **规范优先**：严格遵循 im-infra 组件设计规范，使用标准的 Provider 模式
- **上下文感知**：自动从 context 中提取 trace_id，支持分布式追踪
- **层次化命名空间**：统一的命名空间系统，支持链式调用构建完整路径
- **类型安全**：封装 context 键，避免键名冲突，提供编译时类型检查
- **环境感知**：提供环境相关的默认配置，开发/生产环境优化
- **高性能**：基于 uber-go/zap，零分配日志记录
- **可观测性强**：完整的命名空间路径便于精确过滤和分析

## 🔧 最佳实践

### 1. 服务初始化模式
```go
// ✅ 推荐：使用环境相关的默认配置
config := clog.GetDefaultConfig("production")
if err := clog.Init(context.Background(), config, clog.WithNamespace("my-service")); err != nil {
    log.Fatal(err)
}
```

### 2. 层次化命名空间使用
```go
// ✅ 推荐：使用层次化命名空间，提供清晰的业务边界
func (s *UserService) CreateUser(ctx context.Context, req *CreateUserRequest) error {
    logger := clog.WithContext(ctx)
    
    logger.Info("开始创建用户", clog.String("email", req.Email))
    
    // 使用具体的子命名空间
    validationLogger := logger.Namespace("validation")
    validationLogger.Info("验证用户数据")
    
    return nil
}
```

### 3. 上下文传递 TraceID
```go
// ✅ 推荐：在中间件中注入 TraceID
func TraceMiddleware() gin.HandlerFunc {
    return func(c *gin.Context) {
        traceID := c.GetHeader("X-Trace-ID")
        if traceID == "" {
            traceID = uuid.NewString()
        }
        
        // 类型安全的 TraceID 注入
        ctx := clog.WithTraceID(c.Request.Context(), traceID)
        c.Request = c.Request.WithContext(ctx)
        
        c.Header("X-Trace-ID", traceID)
        c.Next()
    }
}
```

### 4. 业务代码中的日志记录
```go
// ✅ 推荐：始终从 context 获取 logger，自动包含 traceID
func HandleRequest(ctx context.Context) {
    logger := clog.WithContext(ctx)
    
    logger.Info("处理请求开始")
    
    if err := processBusiness(ctx); err != nil {
        logger.Error("业务处理失败", clog.Err(err))
        return
    }
    
    logger.Info("处理请求完成")
}
```

### 5. 结构化字段使用
```go
// ✅ 推荐：使用结构化字段，便于日志分析
clog.Info("用户登录", 
    clog.String("user_id", "12345"),
    clog.String("action", "login"),
    clog.Duration("duration", time.Since(start)),
    clog.String("client_ip", "192.168.1.100"))

// ❌ 不推荐：字符串拼接，难以查询和分析
clog.Info(fmt.Sprintf("用户 %s 登录，耗时 %v", userID, time.Since(start)))
```

### 6. 错误处理
```go
// ✅ 推荐：使用专门的 Err 字段处理错误
if err := database.SaveUser(user); err != nil {
    clog.Error("保存用户失败", 
        clog.String("user_id", user.ID),
        clog.Err(err))
    return err
}
```

## 🔄 迁移指南

### 从旧版本迁移

1. **模块化 → 命名空间**
   ```go
   // 旧代码
   logger := clog.Module("user")
   
   // 新代码  
   logger := clog.Namespace("user")
   ```

2. **初始化方式**
   ```go
   // 旧代码
   clog.Init(config)
   
   // 新代码
   clog.Init(context.Background(), &config, clog.WithNamespace("my-service"))
   ```

3. **TraceID 管理**
   ```go
   // 旧代码
   ctx := context.WithValue(ctx, "traceID", "abc123")
   
   // 新代码
   ctx := clog.WithTraceID(ctx, "abc123")
   ```

clog 专为 GoChat 项目设计，提供了完整的分布式日志解决方案，支持微服务架构下的链路追踪和可观测性需求。