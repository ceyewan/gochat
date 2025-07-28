# clog API 文档

## 核心接口

### Logger 接口
```go
type Logger interface {
    Debug(msg string, fields ...Field)
    Info(msg string, fields ...Field)
    Warn(msg string, fields ...Field)
    Error(msg string, fields ...Field)
    
    With(fields ...Field) Logger
    WithOptions(opts ...zap.Option) Logger
    Module(name string) Logger
}
```

## 配置结构

### Config
```go
type Config struct {
    Level       string           // 日志级别
    Format      string           // 输出格式
    Output      string           // 输出目标
    AddSource   bool             // 是否显示调用位置
    EnableColor bool             // 是否启用颜色
    RootPath    string           // 项目根路径
    Rotation    *RotationConfig  // 轮转配置
}
```

### RotationConfig
```go
type RotationConfig struct {
    MaxSize    int  // 单个文件最大大小(MB)
    MaxBackups int  // 最多保留文件个数
    MaxAge     int  // 日志保留天数
    Compress   bool // 是否压缩轮转文件
}
```

## 全局函数

### 初始化函数

#### New
```go
func New(config ...Config) (Logger, error)
```
创建新的 Logger 实例。推荐在需要依赖注入时使用。

**参数**：
- `config`: 可选的配置参数，不提供则使用默认配置

**返回**：
- `Logger`: Logger 实例
- `error`: 错误信息

**示例**：
```go
// 使用默认配置
logger, err := clog.New()

// 使用自定义配置
config := clog.Config{Level: "debug", Format: "json"}
logger, err := clog.New(config)
```

#### Init
```go
func Init(config Config) error
```
初始化全局 Logger。适用于简单应用或需要全局访问的场景。

**参数**：
- `config`: 日志配置

**返回**：
- `error`: 错误信息

**示例**：
```go
err := clog.Init(clog.Config{
    Level:  "info",
    Format: "console",
    Output: "stdout",
})
```

#### DefaultConfig
```go
func DefaultConfig() Config
```
返回开发环境友好的默认配置。

**返回**：
```go
Config{
    Level:       "info",
    Format:      "console", 
    Output:      "stdout",
    AddSource:   true,
    EnableColor: true,
    RootPath:    "gochat",
}
```

### 日志记录函数

#### Debug/Info/Warn/Error
```go
func Debug(msg string, fields ...Field)
func Info(msg string, fields ...Field)
func Warn(msg string, fields ...Field)
func Error(msg string, fields ...Field)
```
全局日志记录函数。

**参数**：
- `msg`: 日志消息
- `fields`: 可选的结构化字段

**示例**：
```go
clog.Info("用户登录", clog.String("userID", "123"))
clog.Error("连接失败", clog.String("host", "localhost"), clog.Int("port", 3306))
```

### Context 函数

#### WithContext
```go
func WithContext(ctx context.Context) Logger
```
创建带 Context 的 Logger，自动提取 TraceID。

**参数**：
- `ctx`: 包含 TraceID 的 Context

**返回**：
- `Logger`: 带 TraceID 的 Logger

**示例**：
```go
ctx := context.WithValue(context.Background(), "traceID", "abc-123")
clog.WithContext(ctx).Info("处理请求")
```

#### C
```go
func C(ctx context.Context) Logger
```
`WithContext` 的简写形式，用于链式调用。

**示例**：
```go
clog.C(ctx).Module("user").Info("用户操作")
```

### 模块函数

#### Module
```go
func Module(name string) Logger
```
创建模块 Logger，自动添加 module 字段。

**参数**：
- `name`: 模块名称

**返回**：
- `Logger`: 模块 Logger

**示例**：
```go
userLogger := clog.Module("user")
userLogger.Info("用户创建")  // 输出包含 {"module": "user"}
```

### TraceID Hook

#### SetTraceIDHook
```go
func SetTraceIDHook(hook func(context.Context) (string, bool))
```
设置自定义 TraceID 提取函数。

**参数**：
- `hook`: TraceID 提取函数
  - 输入：`context.Context`
  - 输出：`(traceID string, found bool)`

**示例**：
```go
clog.SetTraceIDHook(func(ctx context.Context) (string, bool) {
    if val := ctx.Value("custom-trace-id"); val != nil {
        return val.(string), true
    }
    return "", false
})
```

## 字段类型

### 常用字段函数
```go
func String(key, val string) Field
func Int(key string, val int) Field
func Int64(key string, val int64) Field
func Float64(key string, val float64) Field
func Bool(key string, val bool) Field
func Time(key string, val time.Time) Field
func Duration(key string, val time.Duration) Field
func Err(err error) Field
func Any(key string, val interface{}) Field
```

**示例**：
```go
clog.Info("用户操作",
    clog.String("userID", "123"),
    clog.Int("age", 25),
    clog.Bool("active", true),
    clog.Time("loginTime", time.Now()),
)
```

## 使用模式

### 1. 全局使用模式
```go
// 初始化
clog.Init(config)

// 使用
clog.Info("消息")
clog.Module("user").Info("用户操作")
clog.C(ctx).Info("带 TraceID 的消息")
```

### 2. 依赖注入模式
```go
// 创建实例
logger, err := clog.New(config)

// 注入到服务
service := NewUserService(logger)

// 在服务中使用
func (s *UserService) CreateUser() {
    s.logger.Info("创建用户")
}
```

### 3. 链式调用模式
```go
// Context + Module
clog.C(ctx).Module("payment").Info("支付处理")

// Logger 实例 + Module
logger.Module("database").Error("连接失败")

// 添加字段
logger.With(clog.String("component", "api")).Info("请求处理")
```

## 错误处理

### 常见错误
- **配置错误**：无效的日志级别、格式等
- **文件权限错误**：无法创建或写入日志文件
- **轮转配置错误**：无效的轮转参数

### 错误处理示例
```go
logger, err := clog.New(config)
if err != nil {
    // 使用备用 logger 或退出程序
    log.Fatalf("初始化日志失败: %v", err)
}
```

## 性能考虑

### 最佳实践
1. **复用 Logger 实例**：避免频繁创建新的 Logger
2. **使用结构化字段**：比字符串拼接更高效
3. **合理设置日志级别**：生产环境避免 Debug 级别
4. **模块 Logger 缓存**：全局模块 Logger 会自动缓存

### 性能优化
```go
// ✅ 好的做法：复用模块 Logger
var userLogger = clog.Module("user")

func handleUser() {
    userLogger.Info("处理用户请求")
}

// ❌ 避免：每次都创建新的 Logger
func handleUser() {
    clog.Module("user").Info("处理用户请求")  // 每次都创建新实例
}
```
