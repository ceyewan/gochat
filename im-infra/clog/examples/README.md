# clog 使用示例

本目录包含了 clog 包的各种使用示例，展示了如何按照 GoChat 项目规范使用结构化日志。

## 示例文件

### 1. comprehensive/main.go - 综合示例
**推荐阅读** - 这个示例完整演示了 clog 的所有核心功能和最佳实践。

**覆盖场景：**
- 服务启动时的全局 logger 初始化
- 层次化命名空间系统的使用
- 链式命名空间创建
- Gin HTTP 中间件集成
- 上下文感知的日志记录
- 完整的业务流程日志追踪

**运行方式：**
```bash
cd examples/comprehensive
go run main.go
```

### 2. simple/main.go - 基础示例
展示了 clog 的基本用法，包括：
- 基本日志记录
- 层次化命名空间
- 上下文集成
- 自定义配置
- 独立 logger 实例创建

### 3. basic/main.go - 基础功能
更基础的功能演示，适合初学者。

### 4. advanced/main.go - 高级功能
展示了高级配置和特性（需要更新以适配新 API）。

## 核心概念演示

### 1. 层次化命名空间系统

```go
// 创建不同层次的命名空间 logger
userLogger := clog.Namespace("user")
authLogger := userLogger.Namespace("auth")
dbLogger := userLogger.Namespace("database")

userLogger.Info("开始用户注册流程", clog.String("email", req.Email))
authLogger.Info("验证用户密码强度")
dbLogger.Info("检查用户是否已存在")
```

**输出示例：**
```json
{"namespace": "im-gateway.user", "trace_id": "abc123", "msg": "开始用户注册流程"}
{"namespace": "im-gateway.user.auth", "trace_id": "abc123", "msg": "验证用户密码强度"}
{"namespace": "im-gateway.user.database", "trace_id": "abc123", "msg": "检查用户是否已存在"}
```

### 2. 上下文与 TraceID 管理

```go
// 中间件：注入 traceID 到 context
func TraceMiddleware() gin.HandlerFunc {
    return func(c *gin.Context) {
        traceID := c.GetHeader("X-Trace-ID")
        if traceID == "" {
            traceID = uuid.NewString()
        }
        
        // 注入到 context
        ctx := clog.WithTraceID(c.Request.Context(), traceID)
        c.Request = c.Request.WithContext(ctx)
        
        c.Header("X-Trace-ID", traceID)
        c.Next()
    }
}

// 业务代码：从 context 获取 logger
func (s *UserService) CreateUser(ctx context.Context, req *RegisterRequest) error {
    logger := clog.WithContext(ctx) // 自动包含 trace_id
    
    logger.Info("开始处理创建用户请求",
        clog.String("sender_id", req.SenderID),
        clog.String("email", req.Email))
    
    // 业务逻辑...
    return nil
}
```

### 3. 服务初始化模式

```go
// 在 main.go 中初始化
func main() {
    // 使用环境相关的默认配置
    config := clog.GetDefaultConfig("production")
    
    // 初始化全局 logger，设置服务命名空间
    if err := clog.Init(context.Background(), config, clog.WithNamespace("im-gateway")); err != nil {
        log.Fatalf("初始化 clog 失败: %v", err)
    }

    clog.Info("服务启动成功")
}
```

### 4. 链式命名空间创建

```go
func (s *PaymentService) ProcessPayment(ctx context.Context, req *PaymentRequest) error {
    // 链式创建深层命名空间
    paymentLogger := clog.Namespace("payment").Namespace("processor").Namespace("stripe")
    
    paymentLogger.Info("开始处理支付请求", clog.String("order_id", req.OrderID))
    return nil
}
```

## 设计优势

### 层次化命名空间的核心价值

1. **消除功能重复**
   - 旧方案：`WithService("im-gateway")` + `Module("user")` → 两套相似API
   - 新方案：`WithNamespace("im-gateway")` + `Namespace("user")` → 统一的层次化API

2. **提供组合灵活性**
   ```go
   baseLogger := clog.Namespace("user")                    // "im-gateway.user"
   authLogger := baseLogger.Namespace("auth")              // "im-gateway.user.auth" 
   passwordLogger := authLogger.Namespace("password")      // "im-gateway.user.auth.password"
   ```

3. **概念清晰化**
   - 统一概念：所有标识都是"命名空间"
   - 自然层次：如文件系统路径般直观
   - 一致API：所有层次都使用相同的 `Namespace()` 方法

4. **可观测性增强**
   ```json
   {
     "namespace": "im-gateway.user.auth.password",
     "trace_id": "abc123-def456",
     "msg": "密码验证成功",
     "user_id": "12345"
   }
   ```

## 迁移指南

### 从旧 API 迁移到新 API

| 旧 API | 新 API | 说明 |
|--------|--------|------|
| `clog.Module("user")` | `clog.Namespace("user")` | 统一使用命名空间概念 |
| `clog.Init(config)` | `clog.Init(ctx, &config, opts...)` | 添加上下文和选项参数 |
| `clog.New(config)` | `clog.New(ctx, &config, opts...)` | 添加上下文和选项参数 |
| `ctx = context.WithValue(ctx, "traceID", id)` | `ctx = clog.WithTraceID(ctx, id)` | 使用类型安全的 traceID 注入 |
| `clog.SetTraceIDHook(hook)` | 移除 | 不再需要自定义 hook |
| `logger.Module("sub")` | `logger.Namespace("sub")` | 统一使用命名空间方法 |

## 最佳实践

1. **在服务启动时初始化**：在 `main.go` 中调用 `clog.Init()` 并设置服务命名空间
2. **使用层次化命名空间**：根据业务模块创建清晰的命名空间层次
3. **中间件注入 TraceID**：在请求入口处注入 traceID 并在整个调用链中传递
4. **上下文感知日志**：业务代码中始终使用 `clog.WithContext(ctx)` 获取 logger
5. **环境相关配置**：使用 `clog.GetDefaultConfig(env)` 获取环境优化的默认配置