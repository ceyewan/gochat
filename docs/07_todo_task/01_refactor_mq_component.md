# 任务书：重构 `im-infra/mq` 组件

## 1. 背景与目标

**背景**: 当前的 `im-infra/mq` 组件 API 过于复杂，暴露了大量底层 Kafka 的配置细节，增加了业务开发者的使用难度和心智负担。这违背了 `im-infra` 库“简化基础设施”的核心原则。

**目标**: 根据新制定的 **[im-infra/mq/DESIGN.md](../../im-infra/mq/DESIGN.md)** 文档，对 `mq` 组件进行彻底重构。新的实现必须严格遵循 V2 设计，提供一个极简、配置驱动、高度自动化的消息队列接口。

## 2. 核心要求

1.  **接口对齐**: 新的实现必须 100% 匹配 `DESIGN.md` 中定义的 `Producer` 和 `Consumer` 接口。任何多余的公开方法都应被移除。
2.  **配置中心驱动**: 移除所有手动的配置结构体。组件初始化时，必须通过传入的 `serviceName` 从 `coord` 配置中心获取 Kafka brokers 等连接信息。
3.  **依赖注入**: 必须支持通过 `Option` 模式注入 `clog.Logger` 等依赖。
4.  **自动偏移量管理**: `Consumer` 必须在后台自动、定期地提交偏移量。不允许暴露手动的 `Commit` 方法。
5.  **优雅关闭**: `Producer` 和 `Consumer` 的 `Close()` 方法必须实现优雅关闭逻辑。`Producer` 要确保所有缓冲区的消息都被发出；`Consumer` 要确保当前正在处理的消息完成并提交最后一次偏移量。
6.  **上下文与追踪**: 所有接口方法都应接受 `context.Context` 作为第一个参数，并正确处理 `trace_id` 在消息 `Headers` 中的传递。

## 3. 开发步骤

### 第一阶段：骨架与生产者实现

1.  **清理旧代码**: 删除 `im-infra/mq` 目录下旧的实现文件。
2.  **定义核心接口**: 创建 `im-infra/mq/mq.go` 文件，将 `DESIGN.md` 中定义的 `Message`, `Producer`, `Consumer`, `ConsumeCallback` 等接口和类型粘贴进去。
3.  **实现 `Option` 模式**: 创建 `im-infra/mq/options.go` 文件，实现 `Option` 函数及 `WithLogger` 等具体选项。
4.  **实现 `Producer`**:
    -   创建 `im-infra/mq/producer.go` 文件。
    -   实现 `NewProducer` 工厂函数。该函数内部需要：
        -   从 `coord` 获取配置。
        -   使用 `franz-go` 初始化一个 `kgo.Client`。
    -   实现 `Send` (异步) 和 `SendSync` (同步) 方法。注意 `kgo.Client` 的 `Produce` 和 `ProduceSync` 方法的正确使用。
    -   实现 `Close` 方法，调用 `kgo.Client.Close()`。

### 第二阶段：消费者实现

1.  **实现 `Consumer`**:
    -   创建 `im-infra/mq/consumer.go` 文件。
    -   实现 `NewConsumer` 工厂函数。该函数内部需要：
        -   从 `coord` 获取配置。
        -   设置 `kgo.Opt`，特别是 `kgo.ConsumerGroup()`, `kgo.ConsumeTopics()`。
        -   确保 `kgo.DisableAutoCommit()` 设置为 `false`，以启用 `franz-go` 的自动提交功能。
    -   实现 `Subscribe` 方法。这是一个阻塞方法，其内部应该是一个 `for` 循环，持续调用 `kgo.Client.PollFetches()` 来获取消息。
    -   在循环中，对获取的每条消息，调用用户传入的 `ConsumeCallback`。
    -   实现 `Close` 方法。

### 第三阶段：管理工具与文档

1.  **实现 `AdminClient` (可选)**:
    -   创建 `im-infra/mq/admin.go` 文件。
    -   实现 `NewAdminClient` 和 `CreateTopic` 方法。
2.  **编写单元测试**: 为 `Producer` 和 `Consumer` 的核心功能编写单元测试。需要使用 `gomock` 或 `testify/mock` 来模拟依赖。
3.  **更新示例代码**: 如果项目中有示例代码，需要更新以匹配新的 API。
4.  **最终审查**: 确保所有代码都符合 `DESIGN.md` 的规范，并且旧的 `API.md` 已被删除。

## 4. 验收标准

1.  `im-infra/mq` 目录下的所有 `.go` 代码都已更新为 V2 实现。
2.  旧的 `API.md` 文件已删除，`README.md` 已更新。
3.  新的实现能够通过所有单元测试。
4.  在 `im-gateway`, `im-logic`, `im-task` 等服务中，可以用一行代码（`mq.NewProducer(...)` 或 `mq.NewConsumer(...)`）成功初始化 `mq` 组件，无需任何手动配置。
5.  消息收发功能在整个系统中运行正常，且日志中能看到正确的 `trace_id`。