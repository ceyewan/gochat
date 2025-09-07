# 任务书：重构 `im-infra/once` 组件

## 1. 背景与目标

**背景**: 当前的 `im-infra/once` 组件 API 设计虽然功能全面，但违背了 KISS (Keep It Simple, Stupid) 原则。它暴露了过多的底层操作（`Check`, `Set`, `Delete`），可能诱导开发者写出有竞态条件的非原子性代码。此外，其构造函数签名也与其他 `im-infra` 组件不一致。

**目标**: 根据最新的 **[once 组件开发者文档](../../08_infra/once.md)**，对 `once` 组件进行彻底重构。新的实现必须严格遵循精简后的 API 契约，提供一个更安全、更易用、更聚焦的幂等性解决方案。

## 2. 核心要求

1.  **API 精简**: 新的 `Idempotent` 接口必须且只能包含 `Do` 和 `Execute` 两个方法。所有其他方法（如 `Check`, `Set`, `SetWithResult`, `GetResult`, `Delete` 等）必须被移除。
2.  **签名统一**:
    -   `Do` 和 `Execute` 方法的签名中必须包含 `ttl time.Duration` 参数，让调用者显式地控制幂等键的有效期。
    -   提供一个新的构造函数 `New(ctx context.Context, serviceName string, config *Config, opts ...Option)`，使其签名风格与其他 `im-infra` 组件保持一致。
3.  **原子性保证**: `Execute` 方法的实现必须使用 Lua 脚本来保证“检查、执行、缓存”整个过程的原子性，防止并发场景下的数据不一致。`Do` 方法的实现也应保证其操作的原子性。
4.  **失败可重试**: 在 `Do` 或 `Execute` 中，如果用户传入的业务逻辑函数 `f` 或 `callback` 返回错误，组件必须自动删除已设置的幂等键（如果存在），以允许后续的请求可以重试。
5.  **保留便利性**: 继续提供包级别的全局方法 `once.Do()` 和 `once.Execute()`，它们在内部使用一个默认的、延迟初始化的客户端，以方便在简单场景下快速使用。

## 3. 开发步骤

### 第一阶段：清理与骨架搭建

1.  **清理旧接口**: 打开 `im-infra/once/idempotent.go` (或类似文件)，删除旧的 `Idempotent` 接口定义和所有不再需要的全局方法 (`Check`, `Set`, `Delete` 等)。
2.  **定义新接口**: 在 `idempotent.go` 中，写入新的 `Idempotent` 接口定义，只包含 `Do` 和 `Execute` 方法。
3.  **定义新配置与构造函数**:
    -   定义新的 `Config` 结构体，其中应包含 `cache.Config`。
    -   定义新的 `New` 函数签名，但暂时只返回 `nil, nil`。
4.  **调整 `Option` 模式**: 如有必要，创建或修改 `options.go` 文件，以支持新的 `New` 函数。

### 第二阶段：核心逻辑实现

1.  **实现 `Do` 方法**:
    -   其核心逻辑是 `SET key value NX EX ttl`。
    -   如果设置成功（返回 1），则执行业务函数 `f()`。
    -   如果 `f()` 返回错误，必须执行 `DEL key` 来清除标记，然后返回业务错误。
    -   如果设置失败（返回 0），则说明操作已执行，直接返回 `nil`。
2.  **实现 `Execute` 方法 (Lua 脚本)**:
    -   编写一个 Lua 脚本，该脚本原子性地执行以下逻辑：
        1.  尝试 `GET` 幂等键对应的结果。如果存在，直接返回结果。
        2.  如果结果不存在，尝试 `SET key placeholder NX EX ttl` 来获取锁。
        3.  如果获取锁失败，说明有并发请求正在处理，脚本可以短暂休眠后重试 `GET` 结果（或直接返回特定状态码由客户端处理）。
        4.  如果获取锁成功，脚本返回一个“可以执行”的信号。
    -   在 Go 代码中：
        -   调用 Lua 脚本。
        -   如果脚本返回“可以执行”，则调用业务回调 `callback()`。
        -   如果 `callback()` 成功，将其结果序列化，并调用 `SET key result EX ttl` 来更新占位符。
        -   如果 `callback()` 失败，调用 `DEL key` 清除占位符。
3.  **实现 `New` 构造函数**:
    -   根据传入的 `Config` 初始化一个新的 `cache.Provider`。
    -   创建一个实现了 `Idempotent` 接口的 `client` 结构体，并将 `cache.Provider` 存入。
    -   返回这个新的 `client`。
4.  **实现全局方法**:
    -   实现 `once.Do` 和 `once.Execute`，它们内部通过 `sync.Once` 来初始化一个全局默认的 `Idempotent` 客户端，然后调用该客户端对应的方法。

### 第三阶段：测试与文档

1.  **编写/更新单元测试**:
    -   重点测试 `Do` 和 `Execute` 的各种情况：首次执行、重复执行、并发执行、业务逻辑成功/失败。
    -   为 `Execute` 的 Lua 脚本编写专门的测试。
2.  **更新 `README.md`**: 确保 `README.md` 文件中的所有示例代码都使用新的 API，并移除对旧 API 的所有引用。
3.  **最终审查**: 确保所有代码都符合新的接口契约文档。

## 4. 验收标准

1.  `im-infra/once` 的公开 API 与 `docs/08_infra/once.md` 中定义的完全一致。
2.  所有旧的、不安全的 API 都已被移除。
3.  `New` 函数签名符合 `im-infra` 的标准。
4.  单元测试覆盖了所有核心场景，并通过测试。
5.  使用 `once` 组件的业务代码（如果存在）已更新为使用新的 API。