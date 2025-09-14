# clog - GoChat Structured Logging Library

clog is the official structured logging library for the GoChat project, built on uber-go/zap. It provides a **concise, high-performance, context-aware** logging solution that fully adheres to GoChat's development standards.

## 🚀 Quick Start

### Service Initialization

```go
import (
    "context"
    "github.com/ceyewan/gochat/im-infra/clog"
)

// Initialize with environment-specific default config
config := clog.GetDefaultConfig("production")
if err := clog.Init(context.Background(), config, clog.WithNamespace("im-gateway")); err != nil {
    log.Fatal(err)
}

clog.Info("Service started successfully")
// Output: {"namespace": "im-gateway", "msg": "Service started successfully"}
```

### Basic Usage

```go
// Global logger methods
clog.Info("User logged in", clog.String("user_id", "12345"))
clog.Warn("Connection timeout", clog.Int("timeout", 30))
clog.Error("Database connection failed", clog.Err(err))
clog.Fatal("Fatal error, exiting", clog.String("reason", "config error"))
```

### Hierarchical Namespaces

```go
// Chainable hierarchical namespaces
userLogger := clog.Namespace("user")
authLogger := userLogger.Namespace("auth")
dbLogger := userLogger.Namespace("database")

userLogger.Info("Starting user registration", clog.String("email", "user@example.com"))
// Output: {"namespace": "user", "msg": "Starting user registration", "email": "user@example.com"}

authLogger.Info("Validating password strength")
// Output: {"namespace": "user.auth", "msg": "Validating password strength"}

dbLogger.Info("Checking if user exists")
// Output: {"namespace": "user.database", "msg": "Checking if user exists"}
```

### Context-Aware Logging

```go
// Inject TraceID in middleware
ctx := clog.WithTraceID(context.Background(), "abc123-def456")

// Auto-retrieve logger with TraceID in business code
logger := clog.WithContext(ctx)
logger.Info("Processing request", clog.String("method", "POST"))
// Output: {"trace_id": "abc123-def456", "msg": "Processing request", "method": "POST"}

// Short alias
clog.C(ctx).Info("Request completed")
```

### Provider Mode for Independent Loggers

```go
// Create independent logger instance
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

logger.Info("Independent logger initialized")
```

## 📋 API Reference

### Provider Mode Interfaces

```go
// Standard Provider signature, following im-infra norms
func New(ctx context.Context, config *Config, opts ...Option) (Logger, error)
func Init(ctx context.Context, config *Config, opts ...Option) error
func GetDefaultConfig(env string) *Config  // "development" or "production"
```

### Global Logging Methods

```go
clog.Debug(msg string, fields ...Field)   // Debug info
clog.Info(msg string, fields ...Field)    // General info
clog.Warn(msg string, fields ...Field)    // Warnings
clog.Error(msg string, fields ...Field)   // Errors
clog.Fatal(msg string, fields ...Field)   // Fatal (exits program)
```

### Hierarchical Namespaces

```go
// Create namespaced logger, chainable
func Namespace(name string) Logger

// Example: Deep chaining
logger := clog.Namespace("payment").Namespace("processor").Namespace("stripe")
```

### Context-Aware Logging

```go
// Type-safe TraceID injection
func WithTraceID(ctx context.Context, traceID string) context.Context

// Retrieve logger from context (auto-adds trace_id if present)
func WithContext(ctx context.Context) Logger

// Short alias
func C(ctx context.Context) Logger  // Alias for WithContext
```

### Functional Options

```go
// Set root namespace
func WithNamespace(name string) Option
```

### Structured Field Constructors (zap.Field aliases)

```go
clog.String(key, value string) Field
clog.Int(key string, value int64) Field
clog.Bool(key, value string, bool) Field
clog.Float64(key string, value float64) Field
clog.Duration(key string, value time.Duration) Field
clog.Time(key string, value time.Time) Field
clog.Err(err error) Field
clog.Any(key string, value interface{}) Field
```

## ⚙️ Configuration

```go
type Config struct {
    Level       string           `json:"level"`      // "debug", "info", "warn", "error", "fatal"
    Format      string           `json:"format"`     // "json" (prod) or "console" (dev)
    Output      string           `json:"output"`     // "stdout", "stderr", or file path
    AddSource   bool             `json:"add_source"` // Include source file/line
    EnableColor bool             `json:"enable_color"` // Colors for console
    RootPath    string           `json:"root_path"`  // Project root for path display
    Rotation    *RotationConfig  `json:"rotation"`   // File rotation (if Output is file)
}

type RotationConfig struct {
    MaxSize    int  `json:"max_size"`    // Max file size (MB)
    MaxBackups int  `json:"max_backups"` // Max backup files
    MaxAge     int  `json:"max_age"`     // Retention days
    Compress   bool `json:"compress"`    // Compress rotated files
}
```

### Environment-Aware Defaults

```go
// Development: console, debug, colored
devConfig := clog.GetDefaultConfig("development")

// Production: json, info, no color
prodConfig := clog.GetDefaultConfig("production")
```

## 📝 Usage Examples

### 1. Service Initialization (Recommended)

```go
func main() {
    config := clog.GetDefaultConfig("production")
    if err := clog.Init(context.Background(), config, clog.WithNamespace("im-gateway")); err != nil {
        log.Fatal(err)
    }
    clog.Info("Service started")
}
```

### 2. Gin Middleware Integration

```go
import (
    "github.com/gin-gonic/gin"
    "github.com/google/uuid"
    "github.com/ceyewan/gochat/im-infra/clog"
)

func TraceMiddleware() gin.HandlerFunc {
    return func(c *gin.Context) {
        traceID := c.GetHeader("X-Trace-ID")
        if traceID == "" {
            traceID = uuid.New().String()
        }
        ctx := clog.WithTraceID(c.Request.Context(), traceID)
        c.Request = c.Request.WithContext(ctx)
        c.Header("X-Trace-ID", traceID)
        c.Next()
    }
}

func handler(c *gin.Context) {
    logger := clog.WithContext(c.Request.Context())
    logger.Info("Handling request", clog.String("path", c.Request.URL.Path))
}
```

### 3. Hierarchical Namespaces in Business Logic

```go
func (s *PaymentService) ProcessPayment(ctx context.Context, req *PaymentRequest) error {
    logger := clog.WithContext(ctx)
    logger.Info("Starting payment", clog.String("order_id", req.OrderID))
    
    validationLogger := logger.Namespace("validation")
    validationLogger.Info("Validating payment data")
    
    processorLogger := logger.Namespace("processor").Namespace("stripe")
    processorLogger.Info("Calling Stripe API")
    
    return nil
}
```

### 4. File Output with Rotation

```go
config := &clog.Config{
    Level:    "info",
    Format:   "json",
    Output:   "/app/logs/app.log",
    Rotation: &clog.RotationConfig{
        MaxSize:    100,  // 100MB
        MaxBackups: 3,
        MaxAge:     7,
        Compress:   true,
    },
}

clog.Init(context.Background(), config)
```

### 5. Context Propagation Best Practice

```go
func processUserRequest(ctx context.Context, userID string) error {
    logger := clog.WithContext(ctx)
    logger.Info("Processing user request", clog.String("user_id", userID))
    
    if err := validateUser(ctx, userID); err != nil {
        logger.Error("Validation failed", clog.Err(err))
        return err
    }
    
    logger.Info("Request completed")
    return nil
}

func validateUser(ctx context.Context, userID string) error {
    logger := clog.WithContext(ctx).Namespace("validation")
    logger.Info("Validating user", clog.String("user_id", userID))
    // Validation logic...
    return nil
}
```

## 🎯 Key Features

- **Standards Compliant**: Follows im-infra Provider pattern.
- **Context Aware**: Auto-extracts trace_id for distributed tracing.
- **Hierarchical Namespaces**: Chainable for clear module boundaries.
- **Type Safe**: Encapsulated context keys, compile-time checks.
- **Environment Aware**: Optimized defaults for dev/prod.
- **High Performance**: Zero-allocation via zap.
- **Observable**: Full namespace paths for filtering/analysis.

For detailed architecture and design rationale, see [DESIGN.md](DESIGN.md).
