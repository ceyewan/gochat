# CallerSkip 正确实现参考

## 简化版正确实现

这是一个基于用户提供的正确示例的简化实现，展示了 CallerSkip 的正确处理方式：

```go
package clog

import (
    "context"
    "go.uber.org/zap"
    "go.uber.org/zap/zapcore"
)

type Logger struct {
    base       *zap.Logger
    callerSkip int
    fields     []zap.Field
}

var defaultBase *zap.Logger

func init() {
    config := zap.NewProductionConfig()
    config.EncoderConfig.TimeKey = "timestamp"
    config.EncoderConfig.EncodeTime = zapcore.ISO8601TimeEncoder
    
    // 基础logger只添加AddCaller，不设置CallerSkip
    defaultBase, _ = config.Build(zap.AddCaller())
}

// 构建最终的 zap.Logger
func (l *Logger) build() *zap.Logger {
    logger := l.base
    if l.callerSkip > 0 {
        logger = logger.WithOptions(zap.AddCallerSkip(l.callerSkip))
    }
    if len(l.fields) > 0 {
        logger = logger.With(l.fields...)
    }
    return logger
}

// 日志方法
func (l *Logger) Info(msg string, fields ...zap.Field) {
    l.build().Info(msg, fields...)
}

func (l *Logger) Error(msg string, fields ...zap.Field) {
    l.build().Error(msg, fields...)
}

// 全局函数 - 需要跳过2层：Info -> logger.Info
func Info(msg string, fields ...zap.Field) {
    logger := &Logger{
        base:       defaultBase,
        callerSkip: 2, // 跳过 Info -> logger.Info
    }
    logger.Info(msg, fields...)
}

func Error(msg string, fields ...zap.Field) {
    logger := &Logger{
        base:       defaultBase,
        callerSkip: 2,
    }
    logger.Error(msg, fields...)
}

// Module 创建带模块的logger - 只跳过1层（Logger方法本身）
func Module(name string) *Logger {
    return &Logger{
        base:       defaultBase,
        callerSkip: 1,
        fields:     []zap.Field{zap.String("module", name)},
    }
}

// C 添加context
func (l *Logger) C(ctx context.Context) *Logger {
    fields := append(l.fields, extractFieldsFromContext(ctx)...)
    return &Logger{
        base:       l.base,
        callerSkip: l.callerSkip,
        fields:     fields,
    }
}

// C 返回一个带 context 的 logger
func C(ctx context.Context) *Logger {
    logger := &Logger{
        base:       defaultBase,
        callerSkip: 1,
    }
    return logger.C(ctx)
}

func extractFieldsFromContext(ctx context.Context) []zap.Field {
    var fields []zap.Field
    if traceID := ctx.Value("traceID"); traceID != nil {
        fields = append(fields, zap.String("traceID", traceID.(string)))
    }
    return fields
}
```

## 关键设计原则

### 1. 统一的 CallerSkip 管理

```go
type Logger struct {
    base       *zap.Logger  // 基础logger，不包含CallerSkip
    callerSkip int          // 统一管理CallerSkip
    fields     []zap.Field  // 预设字段
}

// 在build()方法中统一应用CallerSkip
func (l *Logger) build() *zap.Logger {
    logger := l.base
    if l.callerSkip > 0 {
        logger = logger.WithOptions(zap.AddCallerSkip(l.callerSkip))
    }
    return logger
}
```

### 2. 不同调用方式的 CallerSkip 设置

#### 全局函数调用
```go
// 调用栈：main -> clog.Info -> logger.Info -> zap.Logger.Info
// 需要跳过2层才能到达main
func Info(msg string, fields ...zap.Field) {
    logger := &Logger{
        base:       defaultBase,
        callerSkip: 2, // 跳过 clog.Info + logger.Info
    }
    logger.Info(msg, fields...)
}
```

#### 模块化调用
```go
// 调用栈：main -> Module().Info -> zap.Logger.Info
// 需要跳过1层才能到达main
func Module(name string) *Logger {
    return &Logger{
        base:       defaultBase,
        callerSkip: 1, // 跳过 logger.Info
        fields:     []zap.Field{zap.String("module", name)},
    }
}
```

#### Context 调用
```go
// 调用栈：main -> C().Info -> zap.Logger.Info
// 需要跳过1层才能到达main
func C(ctx context.Context) *Logger {
    return &Logger{
        base:       defaultBase,
        callerSkip: 1, // 跳过 logger.Info
    }
}
```

## 测试验证

```go
func main() {
    // 测试 1: 直接调用全局方法
    Info("直接调用全局方法") // 应该显示这一行的位置

    // 测试 2: 模块日志
    userModule := Module("user")
    userModule.Info("模块日志") // 应该显示这一行的位置

    // 测试 3: Context 日志
    ctx := context.WithValue(context.Background(), "traceID", "test-123")
    C(ctx).Info("Context 日志") // 应该显示这一行的位置

    // 测试 4: 链式调用
    C(ctx).C(ctx).Info("链式调用") // 应该显示这一行的位置
}
```

**期望输出**：
```
{"level":"info","timestamp":"2025-07-27T22:42:45.815+0800","caller":"test/main.go:90","msg":"直接调用全局方法"}
{"level":"info","timestamp":"2025-07-27T22:42:45.815+0800","caller":"test/main.go:94","msg":"模块日志","module":"user"}
{"level":"info","timestamp":"2025-07-27T22:42:45.815+0800","caller":"test/main.go:98","msg":"Context 日志","traceID":"test-123"}
{"level":"info","timestamp":"2025-07-27T22:42:45.815+0800","caller":"test/main.go:101","msg":"链式调用","traceID":"test-123"}
```

## 与原实现的对比

### 原实现问题
1. **多层累积**：在多个地方都添加了CallerSkip
2. **固定设置**：基础logger就设置了固定的CallerSkip
3. **复杂接口**：使用接口封装增加了调用层次

### 参考实现优势
1. **统一管理**：CallerSkip在一个地方统一处理
2. **清晰逻辑**：每种调用方式的CallerSkip逻辑清晰
3. **简单结构**：使用结构体而非接口，减少调用层次

## 迁移指南

如果要将现有实现改为参考实现：

1. **保持API兼容**：确保公开API不变
2. **逐步迁移**：先修复CallerSkip，再考虑结构优化
3. **充分测试**：确保所有使用场景都正确工作

这个参考实现展示了CallerSkip的正确处理方式，可以作为类似问题的解决方案参考。
