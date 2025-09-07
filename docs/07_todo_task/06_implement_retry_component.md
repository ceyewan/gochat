# 任务书：实现 `im-infra/retry` 组件

## 1. 背景与目标

**背景**: 在微服务架构中，网络和服务间的瞬时故障是常态。为了提升系统的韧性 (Resilience)，我们需要一个统一的、标准化的机制来自动处理这些可恢复的错误，而不是在业务代码中到处编写分散的、不一致的重试逻辑。

**目标**: 根据 **[retry 组件开发者文档](../../08_infra/retry.md)** 中定义的契约，实现一个全新的 `retry` 组件。该组件必须提供一个极其简洁的 `retry.Do()` 接口，同时内部支持灵活的、策略驱动的重试行为。

## 2. 核心要求

1.  **API 对齐**: 实现必须严格匹配 `retry.md` 中定义的 `Do`, `Policy`, `Operation`, `BackoffStrategy` 等所有公开的函数和类型。
2.  **上下文感知**: `retry.Do` 的实现必须在每次循环开始时检查 `ctx.Done()`。一旦上下文被取消，必须立即停止所有重试并返回 `ctx.Err()`。
3.  **策略驱动**: 所有的重试逻辑（次数、间隔、抖动、错误判断）都必须由传入的 `Policy` 对象驱动。
4.  **智能错误分类**: `Policy.IsRetryable` 回调必须被正确调用。如果 `op` 返回的错误被 `IsRetryable` 判断为不可重试，或者重试次数已达上限，则必须立即停止并返回该错误。
5.  **无状态设计**: `retry` 包本身必须是无状态的。所有状态（如当前尝试次数）都应在 `retry.Do` 函数的单次调用栈中管理。

## 3. 开发步骤

### 第一阶段：定义类型与策略

1.  **创建目录和文件**: 创建 `im-infra/retry/` 目录，并在其中创建 `retry.go`, `backoff.go`, `jitter.go`, `errors.go` 等文件。
2.  **定义核心类型 (`retry.go`)**:
    -   定义 `Operation` 函数类型。
    -   定义 `BackoffStrategy` 和 `JitterStrategy` 接口。
    -   定义 `Policy` 结构体。
3.  **实现退避策略 (`backoff.go`)**:
    -   实现 `Fixed` 策略，它总是返回固定的 `delay`。
    -   实现 `Exponential` 策略，它根据 `initialDelay`, `factor` 和 `attempt` 计算下一次的延迟。
4.  **实现抖动策略 (`jitter.go`)**:
    -   实现 `Full` 抖动策略，返回 `[0, baseDelay]` 之间的随机值。
    -   实现 `Proportional` 抖动策略，返回 `baseDelay ± (baseDelay * factor)` 之间的随机值。
5.  **实现错误分类辅助函数 (`errors.go`)**:
    -   实现 `IsNetworkError`，检查 `errors.Is` 或 `errors.As` 是否为 `net.Error`。
    -   实现 `IsGRPCCodeRetryable`，使用 `status.FromError` 从 `error` 中提取 gRPC 状态码并进行比较。

### 第二阶段：实现核心 `Do` 函数

1.  **实现 `retry.Do` (`retry.go`)**:
    -   这是整个组件的核心。其内部逻辑应该是一个 `for` 循环，从 `attempt = 1` 到 `policy.MaxAttempts`。
    -   **循环开始前**: 检查 `ctx.Done()`。
    -   **执行操作**: 调用 `op(ctx)`。
    -   **处理结果**:
        -   如果 `err == nil`，操作成功，立即返回 `nil`。
        -   如果 `err != nil`：
            a.  检查是否是最后一次尝试 (`attempt == policy.MaxAttempts`)。如果是，直接返回 `err`。
            b.  调用 `policy.IsRetryable(err)`（如果它不为 `nil`）。如果返回 `false`，说明错误不可重试，立即返回 `err`。
            c.  如果可以重试，计算下一次的延迟：
                i.  `delay := policy.Backoff.NextDelay(attempt)`
                ii. 如果 `policy.Jitter != nil`，则 `delay = policy.Jitter.Apply(delay)`。
            d.  等待延迟：使用 `time.NewTimer` 和一个 `select` 块来等待。
                ```go
                select {
                case <-time.After(delay):
                    // 继续下一次循环
                case <-ctx.Done():
                    // 如果在等待期间上下文被取消，立即返回错误
                    return ctx.Err()
                }
                ```
    -   如果循环正常结束（意味着所有尝试都失败了），返回最后一次遇到的错误。

### 第三阶段：测试与文档

1.  **编写单元测试**:
    -   为 `Fixed` 和 `Exponential` 退避策略编写测试。
    -   为 `Do` 函数编写全面的测试用例：
        -   一次成功。
        -   重试一次后成功。
        -   达到最大次数后失败。
        -   因不可重试的错误而立即失败。
        -   因上下文取消而立即失败（包括在等待延迟期间取消）。
2.  **更新 `README.md`**: 在 `im-infra/retry/` 目录下创建一个 `README.md`，提供简洁的使用示例。
3.  **最终审查**: 确保所有公开的 API 都与 `docs/08_infra/retry.md` 中的契约完全一致。

## 4. 验收标准

1.  `im-infra/retry` 包已创建，并包含所有必要的实现。
2.  `retry.Do` 函数的行为完全符合策略定义（重试次数、延迟、错误分类、上下文取消）。
3.  所有代码都通过了单元测试。
4.  `docs/08_infra/retry.md` 文档中的示例代码可以被直接编译和运行。