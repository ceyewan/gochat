# Implementation Plan

[Overview]
本计划旨在对 `im-infra` 库进行全面重构，以统一组件初始化方式、引入标准化的依赖注入机制、为关键组件启用配置热更新，并改进命名约定，从而提升整个基础库的模块化、可观测性和可维护性。

当前的 `im-infra` 组件在初始化和依赖管理上缺乏一致性，这使得集成日志、指标和动态配置等跨领域关注点变得困难。本次重构将引入标准的 `Option` 函数模式、`HotReloadable` 接口、统一的 `componentName` 概念，并采纳更地道的 Go 命名风格，使所有组件符合现代化、健壮且可扩展的设计哲学。

[Types]
本次重构将引入几个新的标准类型和接口，供 `im-infra` 组件共享。

将在 `im-infra/` 目录下创建一个新文件 `options.go`，用于存放通用的 `Option` 模式类型和辅助函数。

```go
// file: im-infra/options.go

package infra

import (
    "github.com/ceyewan/gochat/im-infra/clog"
    "github.com/ceyewan/gochat/im-infra/metrics"
    "github.com/ceyewan/gochat/im-infra/coord"
)

// Options 结构体持有组件的可配置依赖项。
type Options struct {
    Logger          clog.Logger
    MetricsProvider metrics.Provider
    Coordinator     coord.Provider
    ComponentName   string
}

// Option 定义了一个用于配置 Options 结构体的函数类型。
type Option func(*Options)

// WithLogger 创建一个用于设置日志记录器的 Option。
func WithLogger(logger clog.Logger) Option {
    return func(o *Options) {
        o.Logger = logger
    }
}

// WithMetricsProvider 创建一个用于设置指标提供者的 Option。
func WithMetricsProvider(provider metrics.Provider) Option {
    return func(o *Options) {
        o.MetricsProvider = provider
    }
}

// WithCoordinator 创建一个用于设置分布式协调器的 Option。
func WithCoordinator(c coord.Provider) Option {
    return func(o *Options) {
        o.Coordinator = c
    }
}

// WithComponentName 创建一个用于设置组件名称的 Option，以增强可观测性。
func WithComponentName(name string) Option {
    return func(o *Options) {
        o.ComponentName = name
    }
}
```

将在 `im-infra/` 目录下创建一个新文件 `hotreload.go`，用于定义热更新接口。

```go
// file: im-infra/hotreload.go

package infra

import "context"

// HotReloadable 定义了支持配置热更新的组件必须实现的接口。
type HotReloadable interface {
    // HotReload 在检测到配置变更时被触发。
    // 它接收新的配置对象，并应以平滑、无中断的方式应用新配置。
    HotReload(ctx context.Context, newConfig any) error
}
```

[Files]
本次重构将修改 `im-infra` 中几乎所有组件的初始化逻辑和公共 API，并重命名部分包。

-   **New Files**:
    -   `im-infra/options.go`: 定义共享的 `Option` 类型和函数。
    -   `im-infra/hotreload.go`: 定义 `HotReloadable` 接口。
    -   `docs/services/im-infra.md`: 创建一份新的、更完善的中文版指南，以替代旧文件。

-   **Renamed Directories**:
    -   `im-infra/idgen` 将被重命名为 `im-infra/uid`。

-   **Modified Files**:
    -   `im-infra/clog/clog.go`: 更新 `New` 函数。在文档中将全局函数标记为不推荐使用。
    -   `im-infra/db/db.go`: `New` 函数签名变更为 `New(ctx context.Context, cfg Config, opts ...infra.Option)`。全局函数如 `GetDB` 将被标记为不推荐。
    -   `im-infra/cache/cache.go`: `New` 函数签名变更为 `New(ctx context.Context, cfg Config, opts ...infra.Option)`。
    -   `im-infra/coord/coord.go`: `New` 函数签名变更为 `New(ctx context.Context, cfg Config, opts ...infra.Option)`。
    -   `im-infra/ratelimit/ratelimit.go`: `New` 函数签名变更为 `New(ctx context.Context, cfg Config, opts ...infra.Option)`，并将实现 `HotReloadable` 接口。
    -   `im-infra/metrics/metrics.go`: `New` 函数签名变更为 `New(ctx context.Context, cfg Config, opts ...infra.Option)`，并将实现 `HotReloadable` 接口。
    -   `im-infra/mq/mq.go`: `New` 函数签名变更为 `New(ctx context.Context, cfg Config, opts ...infra.Option)`。
    -   `im-infra/uid/uid.go` (原 `idgen.go`): `New` 函数签名变更为 `New(ctx context.Context, cfg Config, opts ...infra.Option)`。
    -   `im-infra/once/idempotent.go`: `New` 函数签名变更为 `New(ctx context.Context, cfg Config, opts ...infra.Option)`。
    -   所有组件的内部实现文件（如 `internal/client.go`）将更新以存储和使用注入的依赖。
    -   所有组件目录下的 `README.md` 文件将更新以反映新的初始化 API。

[Functions]
核心变更是将所有组件的构造函数标准化。

-   **New Functions**:
    -   `infra.WithLogger(clog.Logger) Option` (位于 `im-infra/options.go`)
    -   `infra.WithMetricsProvider(metrics.Provider) Option` (位于 `im-infra/options.go`)
    -   `infra.WithCoordinator(coord.Provider) Option` (位于 `im-infra/options.go`)
    -   `infra.WithComponentName(string) Option` (位于 `im-infra/options.go`)

-   **Modified Functions**:
    -   `clog.New(config ...Config)` -> `clog.New(cfg Config, opts ...infra.Option)`
    -   `db.New(cfg Config)` -> `db.New(ctx context.Context, cfg Config, opts ...infra.Option) (DB, error)`
    -   `cache.New(cfg Config)` -> `cache.New(ctx context.Context, cfg Config, opts ...infra.Option) (Cache, error)`
    -   `coord.New(config ...CoordinatorConfig)` -> `coord.New(ctx context.Context, cfg Config, opts ...infra.Option) (Provider, error)`
    -   `ratelimit.New(...)` -> `ratelimit.New(ctx context.Context, cfg Config, opts ...infra.Option) (Limiter, error)`
    -   `metrics.New(cfg Config)` -> `metrics.New(ctx context.Context, cfg Config, opts ...infra.Option) (Provider, error)`
    -   `mq.New(cfg Config)` -> `mq.New(ctx context.Context, cfg Config, opts ...infra.Option) (MQ, error)`
    -   `uid.New(cfg Config)` (原 `idgen.New`) -> `uid.New(ctx context.Context, cfg Config, opts ...infra.Option) (Generator, error)`
    -   `once.New(cfg Config)` -> `once.New(ctx context.Context, cfg Config, opts ...infra.Option) (Idempotent, error)`

-   **Deprecated Functions**:
    -   所有全局访问函数（如 `db.GetDB()`, `cache.Set()`, `ratelimit.Default()`）将在文档中被标记为**不推荐使用**。目标是推动从依赖全局单例转向显式的依赖注入。

[Classes]
本次重构不引入新的类。

-   **Modified Classes/Structs**:
    -   每个组件的内部实现结构体（如 `internal.db`, `internal.cacheClient`）将增加 `infra.Options` 字段或独立的依赖字段（如 `logger clog.Logger`）。
    -   `ratelimit` 和 `metrics` 的内部实现将实现 `infra.HotReloadable` 接口。

[Dependencies]
本次重构主要关注 `im-infra` 内部的依赖关系。

-   不会添加新的第三方依赖包。
-   依赖流程将变得明确：使用 `im-infra` 的服务将首先初始化 `clog`, `coord`, `metrics`，然后通过新的 `Option` 函数将这些实例注入到其他组中。

[Testing]
测试策略需要更新以适应新的初始化模式。

-   所有组件的单元测试（`*_test.go`）将被重构，以使用新的 `New(ctx, cfg, opts...)` 构造函数。
-   测试需要创建模拟的 `logger` 和其他依赖，以隔离被测组件。
-   将为 `ratelimit` 和 `metrics` 组件添加新的测试，以验证热更新功能。

[Implementation Order]
为有效管理依赖，实施将按以下顺序进行：

1.  **创建核心类型**: 实现 `im-infra/options.go` 和 `im-infra/hotreload.go`。
2.  **重命名 `idgen`**: 将 `im-infra/idgen` 目录重命名为 `im-infra/uid`。
3.  **重构 `clog`**: 更新 `clog.New()`。
4.  **重构 `coord`**: 更新 `coord.New()` 以接受 `logger`。
5.  **重构 `metrics`**: 更新 `metrics.New()` 以接受依赖并实现 `HotReloadable`。
6.  **重构 `ratelimit`**: 更新 `ratelimit.New()` 以接受依赖并实现 `HotReloadable`。
7.  **重构剩余组件**: 依次更新 `db`, `cache`, `mq`, `uid`, `once`。
8.  **更新单元测试**: 在重构每个组件的同时，更新其对应的单元测试。
9.  **更新文档**: 更新所有组件的 `README.md`，并创建新的中文版 `docs/services/im-infra.md`。
10. **标记弃用**: 为所有全局函数添加 `// Deprecated: ...` 注释。
