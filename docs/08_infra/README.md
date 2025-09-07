# 基础设施层 (im-infra)

`im-infra` 是 GoChat 项目的基石，它提供了一系列高质量、生产级别的基础组件。这些组件旨在解决分布式系统中的通用问题，如日志、配置、服务发现、数据库访问、消息队列、缓存、度量、唯一ID生成、幂等性和限流等。

通过将这些通用能力下沉到 `im-infra` 层，业务服务（如 `im-logic`, `im-gateway`）可以更专注于实现核心业务逻辑，从而提高开发效率、代码质量和系统的整体稳定性。

## 1. 核心设计原则

`im-infra` 库中的所有组件都遵循一套统一的设计原则，以确保它们之间的高度一致性和组合性。

### 依赖注入 (Dependency Injection)

所有组件都通过构造函数或选项模式来接收其依赖项（如 `clog.Logger`, `cache.Provider`），而不是在内部创建。这使得上层服务可以完全控制组件的依赖关系，极大地增强了代码的可测试性和灵活性。

### 选项模式 (Options Pattern)

对于拥有多个配置项的组件，我们广泛采用选项模式。通过 `WithXXX` 形式的函数，可以链式地、清晰地构建组件的配置，同时保持构造函数的简洁。

```go
// 示例：创建一个复杂的 ratelimit 组件
limiter, err := ratelimit.New(
    ctx,
    "my-service",
    ratelimit.WithCacheClient(cacheClient),
    ratelimit.WithCoordinationClient(coordClient),
    ratelimit.WithDefaultRules(rules),
    ratelimit.WithFailurePolicy(ratelimit.FailurePolicyAllow),
)
```

### 接口抽象与内部实现分离

每个组件都通过 Go 的接口（Interface）来暴露其公共 API。所有具体的实现细节都被封装在 `internal` 包中。这种做法为使用者提供了清晰、稳定的“契约”，同时保留了内部实现未来迭代和优化的自由。

## 2. 架构模式: 配置热重载

**动态配置**是现代分布式系统的核心要求之一。`im-infra` 通过一个统一的架构模式——`HotReloadable` 接口，将动态配置能力融入到了需要它的组件中。

```go
// im-infra/base/hotreload.go

// HotReloadable 定义了支持配置热更新的组件必须实现的接口。
type HotReloadable interface {
	// HotReload 在检测到配置变更时被触发。
	// 它接收新的配置对象，并应以平滑、无中断的方式应用新配置。
	HotReload(ctx context.Context, newConfig any) error
}
```

这个简单的接口是一个强大的约定。它由 `coord` 组件中的配置管理器（ConfigManager）驱动。当 `coord` 监听到配置中心（如 etcd）的变更时，它会检查对应的组件是否实现了 `HotReloadable` 接口。如果实现了，`coord` 就会调用其 `HotReload` 方法，将新的配置注入进去。

这种模式的优势在于：
- **统一性**: 所有支持动态配置的组件都遵循相同的机制。
- **解耦**: `coord` 作为配置中心适配器，不关心组件如何处理配置；组件则不关心配置来自何处。
- **高可用**: 它促使组件开发者必须思考如何“平滑地”应用新配置，例如，`ratelimit` 组件在更新规则时不会丢失现有的令牌桶状态。

## 3. 核心组件列表

以下是 `im-infra` 库提供的核心组件及其文档链接。

- **[日志 (clog)](./clog.md)**: 提供高性能的结构化日志记录能力。
- **[分布式协调 (coord)](./coord.md)**: 基于 etcd 实现服务发现、分布式锁和动态配置管理。
- **[缓存 (cache)](./cache.md)**: 提供统一的分布式缓存接口，默认基于 Redis 实现。
- **[数据库 (db)](./db.md)**: 封装了 GORM，提供便捷的数据库操作和分片支持。
- **[消息队列 (mq)](./mq.md)**: 提供了消息生产和消费的统一接口，支持 Kafka。
- **[唯一ID (uid)](./uid.md)**: 提供分布式唯一ID生成方案，包括 Snowflake 和 UUID v7。
- **[可观测性 (metrics)](./metrics.md)**: 基于 OpenTelemetry 实现 Metrics 和 Tracing 的零侵入收集。
- **[幂等操作 (once)](./once.md)**: 基于 Redis 实现的分布式幂等操作保证。
- **[分布式限流 (ratelimit)](./ratelimit.md)**: 基于令牌桶算法的分布式限流解决方案。