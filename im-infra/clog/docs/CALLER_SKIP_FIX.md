# CallerSkip 问题修复文档

## 问题描述

在使用 zap 封装的日志库时，遇到了严重的 CallerSkip 问题：日志输出显示的调用位置不正确，显示的是 Go runtime 内部位置而不是用户代码位置。

### 问题现象

```bash
$ go run im-infra/clog/examples/caller_test/main.go
2025-07-27 21:46:44.421 INFO    runtime/proc.go:283     直接调用全局方法
2025-07-27 21:46:44.422 INFO    runtime/asm_arm64.s:1223        模块日志        {"module": "user"}
2025-07-27 21:46:44.422 INFO    runtime/asm_arm64.s:1223        Context 日志    {"traceID": "test-123"}
2025-07-27 21:46:44.422 INFO    runtime/asm_arm64.s:1223        链式调用        {"traceID": "test-123", "module": "order"}
2025-07-27 21:46:44.422 INFO    runtime/proc.go:283     自定义 logger
```

**期望结果**：应该显示用户代码的实际位置，如 `caller_test/main.go:11`

## 问题分析

### 1. 调用栈理解

在 Go 中，`runtime.Caller(skip)` 用于获取调用栈信息：
- `skip=0`：当前函数
- `skip=1`：调用当前函数的函数
- `skip=2`：调用者的调用者

### 2. 根本原因

原始代码存在**多层累积的 CallerSkip 问题**：

1. **基础 logger 层**：`internal/logger.go` 设置了固定的 `zap.AddCallerSkip(2)`
2. **封装函数层**：`clog.go` 中又额外添加了 `zap.AddCallerSkip(3)`
3. **累积效果**：总共跳过了 5 层调用栈，远超实际需要

### 3. 调用栈分析

以 `clog.Info("msg")` 为例：

```
Frame 0: zap.Logger.Info (内部实现)
Frame 1: clog.Info (全局函数)
Frame 2: main.main (用户代码 - 我们想要的)
Frame 3: runtime.main
Frame 4: runtime.goexit
```

**正确的 CallerSkip**：应该跳过 1 层（clog.Info），显示 Frame 2（main.main）

## 解决方案

### 1. 修改基础 Logger

**文件**：`im-infra/clog/internal/logger.go`

**修改前**：
```go
if config.AddSource {
    buildOptions = append(buildOptions, zap.AddCaller(), zap.AddCallerSkip(2))
}
```

**修改后**：
```go
if config.AddSource {
    // 只添加 AddCaller，不设置固定的 CallerSkip
    buildOptions = append(buildOptions, zap.AddCaller())
}
```

### 2. 修改全局函数

**文件**：`im-infra/clog/clog.go`

**修改前**：
```go
func Info(msg string, fields ...Field) {
    getDefaultLogger().Info(msg, fields...)
}
```

**修改后**：
```go
func Info(msg string, fields ...Field) {
    getDefaultLogger().WithOptions(zap.AddCallerSkip(1)).Info(msg, fields...)
}
```

### 3. 修改 Module 函数

**修改前**：
```go
func Module(name string) Logger {
    // ...
    moduleLogger := getDefaultLogger().WithOptions(zap.AddCallerSkip(3)).With(String("module", name))
    // ...
}
```

**修改后**：
```go
func Module(name string) Logger {
    // ...
    // 不需要额外跳过层数，因为Module().Info()是直接调用
    moduleLogger := getDefaultLogger().With(String("module", name))
    // ...
}
```

### 4. 修改 WithContext 函数

**修改前**：
```go
func WithContext(ctx context.Context) Logger {
    logger := getDefaultLogger()
    logger = logger.WithOptions(zap.AddCallerSkip(3))
    // ...
}
```

**修改后**：
```go
func WithContext(ctx context.Context) Logger {
    logger := getDefaultLogger()
    // 不需要额外跳过层数，因为C(ctx).Info()是直接调用
    // logger = logger.WithOptions(zap.AddCallerSkip(1))
    // ...
}
```

## 调试技巧

### 1. 使用 runtime.Caller 调试

创建调试函数来理解调用栈：

```go
func printStack(skip int) {
    fmt.Printf("=== CallerSkip %d ===\n", skip)
    for i := 0; i < 10; i++ {
        pc, file, line, ok := runtime.Caller(i + skip)
        if !ok {
            break
        }
        fn := runtime.FuncForPC(pc)
        fmt.Printf("Frame %d: %s:%d %s\n", i, file, line, fn.Name())
    }
}
```

### 2. 逐步测试不同的 skip 值

```go
for skip := 0; skip <= 5; skip++ {
    printStack(skip)
}
```

## 修复结果

修复后的输出：

```bash
$ go run im-infra/clog/examples/caller_test/main.go
2025-07-27 22:46:29.010	INFO	caller_test/main.go:11	直接调用全局方法
2025-07-27 22:46:29.010	INFO	caller_test/main.go:15	模块日志	{"module": "user"}
2025-07-27 22:46:29.010	INFO	caller_test/main.go:19	Context 日志	{"traceID": "test-123"}
2025-07-27 22:46:29.011	INFO	caller_test/main.go:22	链式调用	{"traceID": "test-123", "module": "order"}
2025-07-27 22:46:29.011	INFO	caller_test/main.go:26	自定义 logger
```

✅ 所有调用位置都正确显示为用户代码的实际位置

## 关键原则

### 1. CallerSkip 设置原则

- **全局函数**：需要跳过 1 层（函数本身）
- **方法调用**：通常不需要额外跳过（如 `Module().Info()`）
- **避免累积**：不要在多个地方重复添加 CallerSkip

### 2. 调用栈分析方法

1. **识别调用路径**：从用户代码到最终的 zap.Logger.Info
2. **计算层数**：确定需要跳过多少层才能到达用户代码
3. **逐步验证**：使用调试工具验证每个 skip 值的效果

### 3. 测试验证

- 创建专门的测试用例验证不同调用方式
- 使用 runtime.Caller 进行调试
- 确保所有使用场景都正确显示位置

## 最佳实践

1. **简化设计**：避免过度复杂的封装层次
2. **统一处理**：在一个地方集中处理 CallerSkip 逻辑
3. **充分测试**：为每种调用方式创建测试用例
4. **文档记录**：记录 CallerSkip 的设计决策和原理

## 技术细节

### 1. zap.AddCallerSkip 工作原理

```go
// zap 内部实现（简化版）
func (log *Logger) Info(msg string, fields ...Field) {
    // skip 参数会传递给 runtime.Caller(skip)
    // 默认 skip=1，跳过 Info 方法本身
    if ce := log.check(InfoLevel, msg); ce != nil {
        ce.Write(fields...)
    }
}

// WithOptions 可以修改 skip 值
func (log *Logger) WithOptions(opts ...Option) *Logger {
    // AddCallerSkip(n) 会在原有 skip 基础上增加 n
    return log.clone(opts...)
}
```

### 2. 不同调用方式的 CallerSkip 计算

#### 直接调用全局函数
```go
// 用户代码
clog.Info("message")

// 调用栈
// Frame 0: zap.Logger.Info
// Frame 1: clog.Info          <- 需要跳过
// Frame 2: main.main          <- 目标位置
// CallerSkip = 1
```

#### 模块化调用
```go
// 用户代码
clog.Module("user").Info("message")

// 调用栈
// Frame 0: zap.Logger.Info
// Frame 1: (Module返回的logger).Info  <- 直接调用，无需跳过
// Frame 2: main.main                   <- 目标位置
// CallerSkip = 0 (基础logger已有默认skip)
```

#### Context 调用
```go
// 用户代码
clog.C(ctx).Info("message")

// 调用栈
// Frame 0: zap.Logger.Info
// Frame 1: (C返回的logger).Info      <- 直接调用，无需跳过
// Frame 2: main.main                  <- 目标位置
// CallerSkip = 0 (基础logger已有默认skip)
```

### 3. 常见错误模式

#### 错误：累积 CallerSkip
```go
// 错误示例
func Info(msg string, fields ...Field) {
    // 基础logger已有skip，这里又加了skip，导致累积
    getDefaultLogger().WithOptions(zap.AddCallerSkip(2)).Info(msg, fields...)
}
```

#### 错误：固定 CallerSkip
```go
// 错误示例
func NewLogger() *zap.Logger {
    // 在基础logger就设置固定skip，影响所有调用
    return logger.WithOptions(zap.AddCallerSkip(2))
}
```

## 验证测试

### 测试用例设计

```go
func TestCallerSkip(t *testing.T) {
    tests := []struct {
        name     string
        logFunc  func()
        expected string
    }{
        {
            name: "全局函数调用",
            logFunc: func() { clog.Info("test") },
            expected: "caller_test.go:XX",
        },
        {
            name: "模块调用",
            logFunc: func() { clog.Module("test").Info("test") },
            expected: "caller_test.go:XX",
        },
        {
            name: "Context调用",
            logFunc: func() { clog.C(ctx).Info("test") },
            expected: "caller_test.go:XX",
        },
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            // 捕获日志输出并验证caller信息
            tt.logFunc()
            // 验证输出包含期望的文件位置
        })
    }
}
```

## 总结

这个问题的核心在于理解 Go 调用栈机制和 zap 的 CallerSkip 工作原理。通过系统性的分析和逐步调试，我们成功解决了多层封装导致的调用位置显示错误问题，确保日志能够准确显示用户代码的位置，大大提升了调试体验。

### 关键收获

1. **理解调用栈**：深入理解 Go 的调用栈机制和 runtime.Caller 工作原理
2. **避免累积**：在多层封装中避免 CallerSkip 的累积效应
3. **分层设计**：合理设计日志库的层次结构，简化 CallerSkip 逻辑
4. **充分测试**：为每种使用场景创建测试用例，确保修复的完整性

这次修复不仅解决了当前问题，也为未来的日志库设计提供了宝贵的经验和最佳实践。
