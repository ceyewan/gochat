# 基础设施: clog 结构化日志

## 1. 设计理念

`clog` 是 `gochat` 项目的官方结构化日志库，基于 `uber-go/zap` 构建。它旨在提供一个**简洁、高性能、上下文感知**的日志解决#### 设计注记：为何提供 `WithTraceID`?

一个合理的设计问题是：为何 `clog` 要提供 `WithTraceID` 来"写"上下文，而不是让调用者自己管理？

答案是：**为了实现完美的封装和提供类型安全的、统一的 API**。

- **封装**: `WithTraceID` 将用于存储 `trace_id` 的 `context.key` 作为包内私有变量，避免了将这个实现细节泄露给外部。这是 Go 社区处理上下文传递的**最佳实践**。
- **API 完整性**: 提供 `WithTraceID` 和 `WithContext` 形成了一套完整、对称的 API，所有与日志上下文相关的操作都内聚在 `clog` 包内，职责清晰。

#### 关键疑问解答：WithTraceID 和 WithContext 的关系

很多开发者会困惑：**为什么需要两个看起来无关的方法？**

**流程解释**：
1. `WithTraceID(ctx, traceID)` → **注入阶段**：在中间件/拦截器中将 traceID 存储到 context
2. `WithContext(ctx)` → **提取阶段**：在业务代码中从 context 提取 traceID 并创建带追踪的 logger

```go
// 中间件：注入 traceID 到 context
ctx := clog.WithTraceID(originalCtx, "abc123")

// 业务代码：从 context 提取 traceID 创建 logger  
logger := clog.WithContext(ctx)  // 内部会自动提取 "abc123" 并添加到日志字段
```

**并发安全性保证**：
- 每个 HTTP 请求都有**独立的 context.Context**
- `WithTraceID` 创建**新的 context**，不修改原有的
- 不同请求的 context 互相**完全隔离**

**方法命名的设计意图**：
- `WithTraceID`: 明确表示"创建包含追踪ID的上下文"
- `WithContext`: 明确表示"根据上下文创建日志器"
- 统一的 `With` 前缀，保持 API 风格一致性

这种设计使得 traceID 的管理**完全透明化**：业务代码只需要传递 context，无需关心 traceID 的具体实现。结构化日志记录的最佳实践。

- **简洁易用**: API 设计简单直观，提供了方便的全局方法，极大降低了使用门槛。
- **高性能**: 基于 `zap` 的零内存分配日志记录引擎，对业务性能影响降至最低。
- **上下文感知**: 能够自动从 `context.Context` 中提取 `trace_id`，将分散的日志条目串联成完整的请求链路，是实现微服务可观测性的关键。
- **模块化**: 支持为不同的服务或业务模块创建专用的、带 `module` 字段的日志器，便于日志的分类和筛选。

## 2. 核心 API 契约

`clog` 的 API 设计兼顾了易用性（全局方法）和灵活性（可实例化的 Logger）。

### 2.1 构造函数与初始化

`clog` 支持两种初始化方式：全局初始化和独立实例创建。

```go
// Config 是 clog 组件的配置结构体。
type Config struct {
	// Level 日志级别: "debug", "info", "warn", "error", "fatal"
	Level string `json:"level"`
	// Format 输出格式: "json" (生产环境推荐) 或 "console" (开发环境推荐)
	Format string `json:"format"`
	// Output 输出目标: "stdout", "stderr", 或文件路径
	Output string `json:"output"`
    // ... 其他配置如 AddSource, EnableColor, Rotation 等
}

// GetDefaultConfig 返回默认的日志配置。
// 开发环境：console 格式，debug 级别，带颜色
// 生产环境：json 格式，info 级别，无颜色
func GetDefaultConfig(env string) *Config

// Init 初始化全局默认的日志器。
// 这是最常用的方式，通常在服务的 main 函数中调用一次。
// ctx: 仅用于控制本次初始化过程的上下文。Logger 实例本身不会持有此上下文。
// opts: 一系列功能选项，如 WithNamespace()，用于定制 Logger 的行为。
func Init(ctx context.Context, config *Config, opts ...Option) error

// New 创建一个独立的、可自定义的 Logger 实例。
// 这在需要将日志输出到不同位置或使用不同格式的特殊场景下很有用。
// ctx: 仅用于控制本次初始化过程的上下文。Logger 实例本身不会持有此上下文。
// opts: 一系列功能选项，如 WithNamespace()，用于定制 Logger 的行为。
func New(ctx context.Context, config *Config, opts ...Option) (Logger, error)
```

### 2.2 功能选项 (Options)

`clog` 使用 `Option` 模式来提供灵活的功能定制。

```go
// Option 是一个用于配置 Logger 实例的功能选项。
type Option func(*options)

// WithNamespace 为 Logger 实例设置根命名空间（通常是服务名）。
// 这个命名空间会出现在该 Logger 实例产生的所有日志中，作为层次化标识的根节点。
//
// 返回值: 一个可用于 New() 或 Init() 的 Option。
func WithNamespace(name string) Option
```

### 2.3 Logger 接口

`Logger` 接口定义了日志记录的核心操作。全局方法和 `Namespace()`、`C()` 返回的实例都实现了此接口。

```go
// Logger 定义了日志记录器的核心接口。
type Logger interface {
	Debug(msg string, fields ...Field)
	Info(msg string, fields ...Field)
	Warn(msg string, fields ...Field)
	Error(msg string, fields ...Field)
	Fatal(msg string, fields ...Field) // 会导致程序退出
	
	// Namespace 创建一个子命名空间的 Logger 实例，支持链式调用。
	// 子命名空间会与父命名空间组合形成完整的层次化路径。
	Namespace(name string) Logger
}

// Field 是一个强类型的键值对，用于结构化日志。
// clog 直接暴露了 zap.Field 的所有构造函数，如 clog.String, clog.Int, clog.Err 等。
type Field = zap.Field
```

### 2.4 上下文与命名空间

```go
// WithContext 从 context 中获取一个 Logger 实例。
// 如果 ctx 中包含 trace_id，返回的 Logger 会自动在每条日志中添加 "trace_id" 字段。
// 这是在处理请求的函数中进行日志记录的【首选方式】。
func WithContext(ctx context.Context) Logger

// WithTraceID 将一个 trace_id 注入到 context 中，并返回一个新的 context。
// 这个函数通常在请求入口处（如 gRPC 拦截器或 HTTP 中间件）调用。
func WithTraceID(ctx context.Context, traceID string) context.Context

// Namespace 创建一个带有层次化命名空间的 Logger 实例。
// 支持链式调用来构建深层的命名空间路径，如 "service.module.component"。
// 这是区分不同业务模块或分层的推荐方式。
func Namespace(name string) Logger

// 为了保持向后兼容，提供简短别名
var C = WithContext  // C(ctx) 等价于 WithContext(ctx)
```

#### 设计理念：层次化命名空间系统

`clog` 采用**层次化命名空间**的设计理念，将日志标识统一为可组合的命名空间系统：

- **根命名空间**: 通过 `WithNamespace()` 在初始化时设置，通常是服务名
- **子命名空间**: 通过 `Namespace()` 方法创建，可以表示业务模块、功能组件等
- **层次组合**: 支持链式调用，形成如 `"im-gateway.user.auth"` 的完整路径

这种设计消除了功能重复，提供了清晰的概念模型和一致的 API 体验。

#### 设计注记：为何提供 `WithTraceID`?

一个合理的设计问题是：为何 `clog` 要提供 `WithTraceID` 来"写"上下文，而不是让调用者自己管理？

答案是：**为了实现完美的封装和提供类型安全的、统一的 API**。

- **封装**: `WithTraceID` 将用于存储 `trace_id` 的 `context.key` 作为包内私有变量，避免了将这个实现细节泄露给外部。这是 Go 社区处理上下文传递的**最佳实践**。
- **API 完整性**: 提供 `WithTraceID` 和 `C` (Get) 形成了一套完整、对称的 API，所有与日志上下文相关的操作都内聚在 `clog` 包内，职责清晰。

## 3. 标准用法

### 场景 1: 在服务启动时初始化全局 Logger

```go
// 在 main.go 中
func main() {
    // 使用默认配置（推荐）
    config := clog.GetDefaultConfig("production") // 或 "development"
    
    // 初始化全局 logger
    if err := clog.Init(context.Background(), config, clog.WithNamespace("im-gateway")); err != nil {
        log.Fatalf("初始化 clog 失败: %v", err)
    }

    clog.Info("服务启动成功") // 输出: {"namespace": "im-gateway", "msg": "服务启动成功"}
}
```

### 场景 2: 在 Gin 中间件中集成

```go
import (
    "github.com/gin-gonic/gin"
    "github.com/google/uuid"
    "github.com/ceyewan/gochat/im-infra/clog"
)

// TraceMiddleware 处理链路追踪 ID
func TraceMiddleware() gin.HandlerFunc {
    return func(c *gin.Context) {
        // 1. 获取或生成 traceID
        traceID := c.GetHeader("X-Trace-ID")
        if traceID == "" {
            traceID = uuid.NewString()
        }

        // 2. 注入到 context（clog 核心用法）
        ctx := clog.WithTraceID(c.Request.Context(), traceID)
        c.Request = c.Request.WithContext(ctx)
        
        c.Header("X-Trace-ID", traceID)
        c.Next()
    }
}

// 业务处理函数
func createUser(c *gin.Context) {
    // 3. 从 context 获取带 traceID 的 logger（clog 核心用法）
    logger := clog.WithContext(c.Request.Context())
    
    logger.Info("开始创建用户请求")
    
    // 调用服务层，传递 context
    user, err := userService.CreateUser(c.Request.Context(), req)
    if err != nil {
        logger.Error("创建用户失败", clog.Err(err))
        return
    }
    
    logger.Info("用户创建成功", clog.String("user_id", user.ID))
}
```

### 场景 3: 在 gRPC 拦截器中集成

```go
func UnaryTraceServerInterceptor() grpc.UnaryServerInterceptor {
    return func(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (interface{}, error) {
        // 1. 从 metadata 提取 traceID
        var traceID string
        if md, ok := metadata.FromIncomingContext(ctx); ok {
            if vals := md.Get("x-trace-id"); len(vals) > 0 {
                traceID = vals[0]
            }
        }
        if traceID == "" {
            traceID = uuid.NewString()
        }

        // 2. 注入到 context（clog 核心用法）
        ctx = clog.WithTraceID(ctx, traceID)

        return handler(ctx, req)
    }
}
```

### 场景 4: 在业务逻辑中使用上下文日志

```go
func (s *MessageService) SendMessage(ctx context.Context, req *pb.SendMessageRequest) (*pb.SendMessageResponse, error) {
    // 从 context 获取带 traceID 的 logger（clog 核心用法）
    logger := clog.WithContext(ctx)

    logger.Info("开始处理发送消息请求",
        clog.String("sender_id", req.SenderID),
        clog.String("receiver_id", req.ReceiverID))

    // 业务逻辑...
    if err != nil {
        logger.Error("发送消息失败", clog.Err(err))
        return nil, err
    }

    logger.Info("成功发送消息")
    return &pb.SendMessageResponse{}, nil
}
```

### 场景 5: 层次化命名空间的使用

```go
func (s *UserService) HandleUserRegistration(ctx context.Context, req *RegisterRequest) error {
    // 创建不同层次的命名空间 logger（clog 核心用法）
    userLogger := clog.Namespace("user")
    authLogger := userLogger.Namespace("auth")
    dbLogger := userLogger.Namespace("database")
    
    userLogger.Info("开始用户注册流程", clog.String("email", req.Email))
    
    authLogger.Info("验证用户密码强度")
    if !s.validatePassword(req.Password) {
        authLogger.Warn("密码强度不足")
        return errors.New("密码强度不足")
    }
    
    dbLogger.Info("检查用户是否已存在")
    exists, err := s.userRepo.UserExists(ctx, req.Email)
    if err != nil {
        dbLogger.Error("查询用户失败", clog.Err(err))
        return err
    }
    
    userLogger.Info("用户注册成功")
    return nil
}
```

**日志输出示例**:
```json
{"namespace": "im-gateway.user", "trace_id": "abc123", "msg": "开始用户注册流程"}
{"namespace": "im-gateway.user.auth", "trace_id": "abc123", "msg": "验证用户密码强度"}
{"namespace": "im-gateway.user.database", "trace_id": "abc123", "msg": "检查用户是否已存在"}
```

### 场景 6: 链式命名空间创建

```go
func (s *PaymentService) ProcessPayment(ctx context.Context, req *PaymentRequest) error {
    // 链式创建深层命名空间（clog 核心用法）
    paymentLogger := clog.Namespace("payment").Namespace("processor").Namespace("stripe")
    
    paymentLogger.Info("开始处理支付请求", clog.String("order_id", req.OrderID))
    return nil
}
```

**输出结果**: `"namespace": "im-gateway.payment.processor.stripe"`

## 4. 设计优势总结

### 4.1 层次化命名空间的核心价值

通过引入层次化命名空间系统，`clog` 实现了以下设计目标：

#### **消除功能重复**
- **Before**: `WithService("im-gateway")` + `Module("user")` → 两套相似API
- **After**: `WithNamespace("im-gateway")` + `Namespace("user")` → 统一的层次化API

#### **提供组合灵活性**
```go
// 可根据需要创建任意层次的标识
baseLogger := clog.Namespace("user")                    // "im-gateway.user"
authLogger := baseLogger.Namespace("auth")              // "im-gateway.user.auth" 
passwordLogger := authLogger.Namespace("password")      // "im-gateway.user.auth.password"
```

#### **概念清晰化**
- **统一概念**: 所有标识都是"命名空间"，避免 service vs module 的概念混淆
- **自然层次**: 如文件系统路径般直观的层次关系
- **一致API**: 所有层次都使用相同的 `Namespace()` 方法

#### **可观测性增强**
```json
{
  "namespace": "im-gateway.user.auth.password",
  "trace_id": "abc123-def456",
  "msg": "密码验证成功",
  "user_id": "12345"
}
```

通过完整的命名空间路径，运维人员可以：
- **快速定位**: 立即知道日志来自哪个服务的哪个模块的哪个组件
- **精确过滤**: 按任意层次进行日志查询和分析
- **链路追踪**: 结合 trace_id 实现完整的请求链路可视化

### 4.2 与传统方案的对比

| 维度 | 传统方案 | 层次化命名空间 |
|------|----------|---------------|
| **API数量** | 2套API (WithService + Module) | 1套API (WithNamespace + Namespace) |
| **概念复杂度** | 高 (service vs module 边界模糊) | 低 (统一的命名空间概念) |
| **扩展性** | 差 (固定的两层结构) | 强 (任意层次的嵌套) |
| **一致性** | 差 (Option vs 直接方法) | 高 (一致的调用方式) |
| **可读性** | 中 (需要理解两套概念) | 高 (自然的层次结构) |

这种设计体现了 **"简单的概念 + 组合的力量 = 强大的表达能力"** 的设计哲学。
}
```