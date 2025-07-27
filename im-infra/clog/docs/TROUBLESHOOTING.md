# CallerSkip 故障排除指南

## 快速诊断

### 1. 识别问题

如果你的日志输出显示类似以下位置，说明存在 CallerSkip 问题：

```bash
# 错误的输出
2025-07-27 21:46:44.421 INFO    runtime/proc.go:283     消息内容
2025-07-27 21:46:44.422 INFO    runtime/asm_arm64.s:1223    消息内容

# 正确的输出应该是
2025-07-27 21:46:44.421 INFO    your_file.go:123    消息内容
```

### 2. 快速调试工具

创建以下调试函数来分析调用栈：

```go
package main

import (
    "runtime"
    "fmt"
)

func debugCallerSkip() {
    fmt.Println("=== 调用栈分析 ===")
    for i := 0; i < 8; i++ {
        pc, file, line, ok := runtime.Caller(i)
        if !ok {
            break
        }
        fn := runtime.FuncForPC(pc)
        fmt.Printf("Frame %d: %s:%d %s\n", i, file, line, fn.Name())
    }
    fmt.Println()
}

func main() {
    debugCallerSkip()
    
    // 在这里调用你的日志函数
    // clog.Info("test message")
}
```

### 3. 确定正确的 CallerSkip 值

运行调试工具，找到你想要显示的用户代码位置（通常是 `main.main` 或你的业务函数），然后计算需要跳过的层数。

## 常见问题及解决方案

### 问题 1：显示 runtime 位置

**现象**：
```
INFO    runtime/proc.go:283     消息内容
INFO    runtime/asm_arm64.s:1223    消息内容
```

**原因**：CallerSkip 值过大，跳过了用户代码

**解决方案**：
1. 减少 CallerSkip 值
2. 检查是否有多处设置 CallerSkip 导致累积

### 问题 2：显示日志库内部位置

**现象**：
```
INFO    clog/clog.go:139    消息内容
INFO    internal/logger.go:115    消息内容
```

**原因**：CallerSkip 值过小，没有跳过日志库内部调用

**解决方案**：
1. 增加 CallerSkip 值
2. 确保跳过所有日志库内部调用层

### 问题 3：不同调用方式显示不一致

**现象**：
```
INFO    main.go:10    直接调用正确
INFO    runtime/proc.go:283    模块调用错误
INFO    runtime/asm_arm64.s:1223    Context调用错误
```

**原因**：不同调用方式的调用栈深度不同，但使用了相同的 CallerSkip

**解决方案**：
为不同的调用方式设置不同的 CallerSkip 值

## 修复步骤

### 步骤 1：分析当前实现

1. 找到所有设置 CallerSkip 的地方
2. 分析每种调用方式的调用栈
3. 确定问题根源

### 步骤 2：制定修复方案

```go
// 1. 移除基础logger的固定CallerSkip
func NewLogger() *zap.Logger {
    // 错误：不要在基础logger设置固定CallerSkip
    // return logger.WithOptions(zap.AddCallerSkip(2))
    
    // 正确：只添加AddCaller
    return logger.WithOptions(zap.AddCaller())
}

// 2. 在具体调用处设置合适的CallerSkip
func Info(msg string, fields ...Field) {
    // 分析调用栈：main -> Info -> zap.Logger.Info
    // 需要跳过1层（Info函数）
    getDefaultLogger().WithOptions(zap.AddCallerSkip(1)).Info(msg, fields...)
}
```

### 步骤 3：逐个修复

1. **全局函数**：通常需要 CallerSkip(1)
2. **模块函数**：通常不需要额外 CallerSkip
3. **Context函数**：通常不需要额外 CallerSkip

### 步骤 4：验证修复

创建测试用例验证每种调用方式：

```go
func TestCallerSkip(t *testing.T) {
    // 测试全局函数
    clog.Info("全局函数测试")
    
    // 测试模块函数
    clog.Module("test").Info("模块函数测试")
    
    // 测试Context函数
    ctx := context.Background()
    clog.C(ctx).Info("Context函数测试")
    
    // 检查输出是否显示正确的文件位置
}
```

## 预防措施

### 1. 设计原则

- **单一职责**：CallerSkip 只在一个地方处理
- **避免累积**：不要在多个层次都设置 CallerSkip
- **明确文档**：记录每个 CallerSkip 值的设计原因

### 2. 代码审查检查点

- [ ] 是否有多处设置 CallerSkip？
- [ ] 基础 logger 是否设置了固定 CallerSkip？
- [ ] 不同调用方式是否考虑了调用栈差异？
- [ ] 是否有充分的测试覆盖？

### 3. 测试策略

```go
// 为每种调用方式创建专门的测试
func TestDirectCall(t *testing.T) {
    // 测试 clog.Info()
}

func TestModuleCall(t *testing.T) {
    // 测试 clog.Module().Info()
}

func TestContextCall(t *testing.T) {
    // 测试 clog.C(ctx).Info()
}

func TestChainCall(t *testing.T) {
    // 测试 clog.C(ctx).Module().Info()
}
```

## 工具和资源

### 调试工具

1. **runtime.Caller**：分析调用栈
2. **zap.AddStacktrace**：添加堆栈跟踪
3. **自定义调试函数**：打印详细调用信息

### 参考资源

- [Go runtime 包文档](https://pkg.go.dev/runtime)
- [zap 日志库文档](https://pkg.go.dev/go.uber.org/zap)
- [CallerSkip 最佳实践](./REFERENCE_IMPLEMENTATION.md)

## 总结

CallerSkip 问题虽然复杂，但通过系统性的分析和调试，可以有效解决。关键是理解调用栈机制，避免多层累积，并为每种使用场景设置正确的跳过层数。

记住：**简单的设计往往是最好的设计**。避免过度复杂的封装，保持 CallerSkip 逻辑的清晰和可维护性。
