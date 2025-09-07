# 任务书：重构 `im-infra/ratelimit` 组件

## 1. 背景与目标

**背景**: 当前的 `im-infra/ratelimit` 组件功能过于复杂，API 冗余（`Manager`, `AllowN`, `BatchAllow`），且其动态配置的实现方式（“组件自治”）未被提升为标准架构模式，构造函数签名也与其他 `im-infra` 组件不一致。

**目标**: 根据最新的 **[ratelimit 组件开发者文档](../../08_infra/ratelimit.md)**，对 `ratelimit` 组件进行彻底重构。新的实现必须遵循最终确定的极简设计，使其 API 更专注、配置管理方式更标准、架构更清晰一致。

## 2. 核心要求

1.  **API 极简**:
    -   `RateLimiter` 接口必须且只能包含 `Allow` 和 `Close` 两个方法。
    -   必须彻底移除 `RateLimiterManager` 接口以及 `AllowN`, `BatchAllow` 等所有非核心方法。
2.  **统一构造函数**:
    -   必须提供一个新的、统一的构造函数 `New(ctx context.Context, config *Config, opts ...Option)`。
    -   `Config` 结构体必须包含 `ServiceName` 和 `RulesPath` 字段，将配置路径的构建逻辑从组件内部移到调用方。
3.  **标准化的动态配置**:
    -   必须采用“组件自治”的动态配置模式。在 `New` 函数中，组件需自行使用 `coord.Provider` 的 `WatchPrefix` 功能来监听 `Config.RulesPath` 的变化。
    -   当监听到配置变更时，组件必须以线程安全的方式（推荐使用 `sync.RWMutex`）热更新其内存中的规则集。
    -   必须废弃 `HotReloadable` 接口，不在此组件中实现它。
4.  **声明式配置**: 组件不再提供任何命令式的规则修改 API（如 `SetRule`, `DeleteRule`）。规则的增、删、改完全通过修改配置中心的文件来实现。
5.  **健壮性**: 组件在 `coord` 或 `cache` 等依赖项不可用时，应有明确的失败策略（如日志记录、默认放行/拒绝等）。

## 3. 开发步骤

### 第一阶段：清理与骨架搭建

1.  **清理旧代码**:
    -   打开 `im-infra/ratelimit/ratelimit.go`，删除旧的 `RateLimiter` 和 `RateLimiterManager` 接口定义。
    -   删除 `AllowN`, `BatchAllow` 等全局方法的实现。
    -   删除 `internal/config.go` 中与 `Manager` 相关的 `setRule`, `deleteRule`, `exportRules` 等函数。
2.  **定义新接口与配置**:
    -   在 `ratelimit.go` 中，写入新的、只包含 `Allow` 和 `Close` 的 `RateLimiter` 接口。
    -   定义新的 `Config` 结构体。
    -   定义新的 `New` 函数签名，但暂时只返回 `nil, nil`。

### 第二阶段：核心逻辑实现

1.  **实现 `New` 构造函数**:
    -   此函数是重构的核心。它需要：
        a.  接收 `Config` 对象和 `opts`。
        b.  初始化 `limiter` 结构体，包含 `logger`, `cache`, `coord` 等客户端。
        c.  调用 `coord.Provider` 的 `Config().List()` 方法，根据 `Config.RulesPath` 全量加载一次初始规则，并存入 `limiter` 的 `rules` map 中。
        d.  启动一个后台 goroutine。
2.  **实现后台监听 Goroutine**:
    -   这个 goroutine 的职责是监听配置变化。
    -   调用 `coord.Provider.Config().WatchPrefix(ctx, config.RulesPath, ...)`。
    -   在一个 `for-select` 循环中等待 `watcher.Chan()` 的事件。
    -   当收到事件时（无论增删改），**重新调用 `coord.List()` 全量加载** `RulesPath` 下的所有规则，并以**加写锁**的方式（`mu.Lock()`）完整替换内存中的 `rules` map。这种全量替换比增量更新更简单、更健壮。
    -   确保 goroutine 能在 `ctx.Done()` 时优雅退出。
3.  **实现 `Allow` 方法**:
    -   以**加读锁**的方式（`mu.RLock()`）从内存中的 `rules` map 获取规则。
    -   如果规则存在，则调用 `cache.Provider` 执行 Lua 脚本进行令牌桶计算。
    -   如果规则不存在，则根据预设的策略（如默认拒绝）进行处理。
4.  **实现 `Close` 方法**:
    -   此方法需要能取消 `New` 函数中启动的后台 goroutine 的 `context`，以确保其能正常退出。

### 第三阶段：测试与文档

1.  **编写/更新单元测试**:
    -   重点测试 `New` 函数能否正确加载初始规则。
    -   重点测试动态配置：模拟 `coord` 发出变更事件，验证 `ratelimit` 内存中的规则是否被正确、线程安全地更新。
    -   测试 `Allow` 方法在规则存在、不存在、Redis 故障等情况下的行为。
2.  **更新 `README.md`**: 确保 `README.md` 文件中的所有示例代码都使用新的 API，并移除对旧 API 的所有引用。
3.  **最终审查**: 确保所有代码都符合新的接口契约文档，特别是 `Manager` 等相关代码已被彻底删除。

## 4. 验收标准

1.  `im-infra/ratelimit` 的公开 API 与 `docs/08_infra/ratelimit.md` 中定义的完全一致。
2.  `RateLimiterManager` 接口和 `AllowN`, `BatchAllow` 方法已被彻底删除。
3.  `New` 函数签名符合 `im-infra` 的标准，接收 `Config` 对象。
4.  动态配置完全由组件内的 `WatchPrefix` 逻辑驱动，无需外部调用。
5.  单元测试覆盖了核心的动态配置和并发访问场景，并通过测试。