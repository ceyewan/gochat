# 任务书：实现 `im-infra/breaker` 组件

## 1. 背景与目标

**背景**: 为了防止微服务架构中的级联失败（雪崩效应），我们需要一个熔断器组件。它能在一个下游服务持续失败时，快速切断对其的调用，保护上游服务，并在下游服务恢复后自动恢复流量。

**目标**: 根据 **[breaker 组件开发者文档](../../08_infra/breaker.md)** 中定义的契约，实现一个全新的 `breaker` 组件。该组件内部基于成熟的 `sony/gobreaker` 库，但对外暴露的 API 必须严格遵循 `im-infra` 的核心规范。

## 2. 核心要求

1.  **API 对齐**: 实现必须严格匹配 `breaker.md` 中定义的 `Provider`, `Breaker`, `Policy`, `Config` 等所有公开的类型和函数。
2.  **封装实现**: `sony/gobreaker` 必须作为内部实现细节，不能泄露到任何公开的 API 或数据结构中。
3.  **Provider 模式**: 必须实现 `Provider` 来统一管理多个熔断器实例。`GetBreaker(name)` 方法必须是线程安全的，并能按需创建新的熔断器实例。
4.  **动态配置**: `Provider` 在初始化时，必须使用 `coord` 的 `Watch` 功能监听其在配置中心的策略路径 (`Config.PoliciesPath`)。当策略文件发生变更时，应能热更新内存中的策略映射。
5.  **错误处理**: `Breaker.Do` 方法在熔断器跳闸时，必须返回标准的 `breaker.ErrBreakerOpen` 错误。

## 3. 开发步骤

### 第一阶段：骨架与类型定义

1.  **创建目录和文件**: 创建 `im-infra/breaker/` 目录，并在其中创建 `breaker.go`, `options.go` 等文件。
2.  **定义公开 API (`breaker.go`)**:
    -   定义 `Breaker` 和 `Provider` 接口。
    -   定义 `Policy` 和 `Config` 结构体。
    -   定义 `ErrBreakerOpen` 公开错误。
    -   定义 `New` 函数签名，但暂时只返回 `nil, nil`。
3.  **引入依赖**: 在 `go.mod` 中添加 `github.com/sony/gobreaker`。

### 第二阶段：核心逻辑实现

1.  **实现 `internal/provider.go`**:
    -   定义 `provider` 结构体，其中包含 `sync.RWMutex`, `map[string]*gobreaker.CircuitBreaker` (用于存储熔断器实例), `map[string]Policy` (用于存储策略), 以及 `coord.Provider`, `clog.Logger` 等依赖。
    -   实现 `New` 函数的内部逻辑：
        a. 初始化 `provider` 结构体。
        b. 调用 `coord` 全量加载一次初始策略，存入 `policies` map。
        c. 启动一个后台 goroutine，使用 `coord.Watch` 监听策略路径，并在收到事件时，以加写锁的方式更新 `policies` map。
    -   实现 `GetBreaker(name string)` 方法：
        a. 加读锁检查 `breakers` map 中是否已存在该名称的实例。如果存在，直接返回。
        b. 如果不存在，加写锁再次检查（防止并发创建）。
        c. 如果仍不存在，从 `policies` map 中获取该名称对应的策略（若无则使用默认策略）。
        d. 根据策略创建 `gobreaker.Settings`，特别注意 `Name`, `ReadyToTrip`, `OnStateChange` 等字段的设置。
        e. 调用 `gobreaker.NewCircuitBreaker` 创建新实例，存入 `breakers` map，并返回一个包裹了它的 `breaker` 实例。
2.  **实现 `internal/breaker.go`**:
    -   定义 `breaker` 结构体，它只包含一个字段 `cb *gobreaker.CircuitBreaker`。
    -   实现 `Do` 方法，其内部直接调用 `b.cb.Execute()`。
    -   在 `Do` 方法中，对 `Execute` 返回的 `error` 进行处理，将 `gobreaker.ErrOpenState` 等错误统一转换为我们自己的 `breaker.ErrBreakerOpen`。

### 第三阶段：测试与文档

1.  **编写单元测试**:
    -   重点测试 `Provider.GetBreaker` 的并发安全性和按需创建逻辑。
    -   重点测试动态配置：模拟 `coord` 发出策略变更事件，验证下一次 `GetBreaker` 创建新实例时是否会使用新策略。
    -   测试 `Breaker.Do` 的核心逻辑：成功、失败、跳闸、半开后成功、半开后失败等场景。
2.  **更新 `README.md`**: 在 `im-infra/breaker/` 目录下创建一个 `README.md`，提供简洁的使用示例。
3.  **最终审查**: 确保所有公开的 API 都与 `docs/08_infra/breaker.md` 中的契约完全一致。

## 4. 验收标准

1.  `im-infra/breaker` 包已创建，并基于 `gobreaker` 实现。
2.  `Provider` 能够正确地管理多个熔断器实例，并能通过 `coord` 动态更新策略。
3.  `Breaker.Do` 的行为符合熔断器的标准状态机。
4.  所有代码都通过了单元测试。
5.  `docs/08_infra/breaker.md` 文档中的示例代码可以被直接编译和运行。