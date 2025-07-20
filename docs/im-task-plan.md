## **GoChat - `im-task` 模块开发设计与计划**

**致实习生同学：**

你好！你即将开始 `im-task` 模块的开发。如果说 `im-logic` 是我们系统的“大脑”，那么 `im-task` 就是系统的“强大的左膀右臂”。它负责处理所有耗时、繁重或需要与外部世界打交道的异步任务。

这个模块的开发将极大地锻炼你对**异步处理、消息队列、系统解耦和健壮性设计**的理解。它虽然不直接面向用户，但却是保障用户体验和系统稳定不可或缺的一环。

### **1. 模块职责与目标 (The "What")**

`im-task` 的核心职责非常明确：

1.  **异步任务消费者**: 它是 GoChat 系统中所有**非实时任务**的唯一消费者。它通过**消费 Kafka 中的任务消息**来驱动自己的工作。
2.  **重负载处理器**: 专门处理那些会严重消耗 CPU 或 I/O 资源的任务，从而避免拖垮核心的 `im-logic` 服务。
    *   **典型案例**: **超大群（如万人群）的消息扩散**。
3.  **外部系统集成器**: 负责与所有第三方系统进行交互。这些交互通常伴随着网络延迟和不确定性。
    *   **典型案例**: 调用苹果/谷歌的服务器进行**离线推送**；调用第三方服务进行**内容安全审核**。
4.  **后台作业执行者**: 执行所有预定的、周期性的后台任务。
    *   **典型案例**: **数据归档**、**生成统计报表**等。

**我们的目标**: 开发一个**可靠、可扩展、易于增加新任务类型**的异步任务处理服务。

### **2. 技术栈与依赖 (The "Tools")**

| 用途 | 库/工具 | 学习重点 |
| :--- | :--- | :--- |
| **Kafka 客户端** | `github.com/segmentio/kafka-go` | **消费者组 (ReaderGroup)** 的配置与使用，这是核心 |
| **gRPC 客户端** | `google.golang.org/grpc` | 调用 `im-repo` 服务获取任务所需的数据 |
| **配置管理** | `github.com/spf13/viper` | 加载服务配置 |
| **日志** | `go.uber.org/zap` | 结构化日志 |
| **服务发现** | `go.etcd.io/etcd/client/v3` | 发现 `im-repo` |
| **HTTP 客户端**| `net/http` | (未来) 用于调用第三方 RESTful API (如内容审核) |

### **3. 项目结构 (The "Blueprint")**

`im-task` 的设计核心是**任务处理的抽象**。我们应该能够非常方便地增加一种新的任务处理器，而不需要改动核心的消费逻辑。

```
im-task/
├── cmd/
│   └── main.go              # 程序入口：初始化、启动消费者
├── internal/
│   ├── config/              # 配置定义与加载
│   ├── consumer/
│   │   ├── consumer.go      # Kafka 消费者组的核心逻辑
│   │   └── dispatcher.go    # 任务分发器，根据任务类型调用不同的处理器
│   ├── processor/
│   │   ├── interface.go     # 定义所有任务处理器的通用接口
│   │   ├── fanout_processor.go # 大群扩散任务的处理器
│   │   └── push_processor.go   # (未来) 离线推送任务的处理器
│   └── rpc/
│       └── repo_client.go     # im-repo 的 gRPC 客户端封装
├── go.mod
└── go.sum
```
**设计亮点**: `processor` 目录的设计。我们定义一个通用 `interface`，所有具体的任务处理器都实现这个接口。`dispatcher` 则像一个工厂，根据消息内容决定使用哪个处理器。这种设计模式使得添加新任务类型变得即插即用。

### **4. 开发计划：分阶段进行 (The "Plan")**

---

#### **阶段一：搭建骨架与实现 Kafka 消费 (预计2天)**

**目标**: 让 `im-task` 能够作为一个 Kafka 消费者组跑起来，并能消费到来自 `im-logic` 的任务消息。

**任务分解**:
1.  **初始化项目**: `go mod init`，引入 `kafka-go`, `viper`, `zap`, `grpc-go` 等依赖。
2.  **配置管理 (`internal/config`)**:
    *   定义 `Config` 结构体，包含 Kafka `brokers`、任务 Topic (`im-task-topic`)、消费者组ID (`im-task-group`) 等。
    *   实现 `LoadConfig()` 函数。
3.  **`im-repo` 客户端 (`internal/rpc`)**:
    *   编写 `NewRepoClient()` 函数，用于连接 `im-repo` 服务（暂时可硬编码地址）。
4.  **程序入口 (`cmd/main.go`)**:
    *   加载配置，初始化日志，初始化 `repoClient`。
    *   初始化并启动 `consumer`。
5.  **核心消费逻辑 (`internal/consumer/consumer.go`)**:
    *   创建一个 `TaskConsumer` 结构体，持有 `repoClient` 和 `TaskDispatcher`。
    *   创建一个 `Start()` 方法。
    *   在 `Start()` 方法中，使用 `kafka.NewReader` 配置消费者组。
    *   在一个 `for` 循环中不断消费消息。
    *   **暂时**，只将消费到的消息（包括 `Header` 和 `Body`）打印到日志中。

**验收标准**:
*   启动 `im-task` 和 Kafka。
*   使用 Kafka 命令行工具向 `im-task-topic` 生产一条带 `Header` 的消息（例如 `task_type:large_group_fanout`）。
*   在 `im-task` 的日志中，应该能看到这条消息的完整内容被打印出来。

---

#### **阶段二：实现任务分发器与处理器抽象 (预计1天)**

**目标**: 建立一个可扩展的任务处理框架，为后续添加具体任务做准备。

**任务分解**:
1.  **定义处理器接口 (`internal/processor/interface.go`)**:
    ```go
    package processor

    import (
        "context"
        "github.com/segmentio/kafka-go"
    )

    // TaskProcessor 是所有任务处理器的通用接口
    type TaskProcessor interface {
        // Process 处理单个 Kafka 消息
        Process(ctx context.Context, msg kafka.Message) error
    }
    ```
2.  **实现任务分发器 (`internal/consumer/dispatcher.go`)**:
    *   创建一个 `TaskDispatcher` 结构体。它内部包含一个 `map[string]processor.TaskProcessor`，用于存储任务类型到具体处理器的映射。
    *   提供一个 `Register(taskType string, p processor.TaskProcessor)` 方法，用于注册处理器。
    *   提供一个 `Dispatch(ctx context.Context, msg kafka.Message)` 方法。其逻辑是：
        1.  从 `msg.Headers` 中解析出 `task_type`。
        2.  根据 `task_type` 从 map 中查找到对应的处理器。
        3.  如果找到，调用处理器的 `Process` 方法。
        4.  如果没找到，记录错误日志。
3.  **集成到消费者**:
    *   在 `cmd/main.go` 中，初始化 `TaskDispatcher`，并（暂时）不注册任何处理器。
    *   修改 `internal/consumer/consumer.go` 的消费循环，将打印日志的逻辑替换为调用 `dispatcher.Dispatch(msg)`。

**验收标准**:
*   代码能够编译通过。
*   向 Kafka 发送任务消息，`im-task` 日志应显示“未找到任务处理器”的错误，证明分发逻辑已生效。

---

#### **阶段三：实现第一个核心任务：大群消息扩散 (预计3天)**

**目标**: 完整实现超大群聊消息的异步扩散功能。

**任务分解**:
1.  **创建处理器 (`internal/processor/fanout_processor.go`)**:
    *   创建一个 `FanoutProcessor` 结构体，它需要持有 `repoClient` 和一个 Kafka `Producer`（用于生产下行消息）。
    *   让 `FanoutProcessor` 实现 `TaskProcessor` 接口，即实现 `Process` 方法。
2.  **实现 `Process` 方法**:
    *   **解析任务**: 从 `msg.Body` 中反序列化出任务内容，例如 `{"group_id": ..., "message_id": ...}`。
    *   **获取数据**:
        1.  调用 `repoClient` 的 `GetMessage(message_id)` 方法，获取完整的消息内容。
        2.  调用 `repoClient` 的 `GetGroupMembers(group_id)` 方法，获取群的所有成员 `user_id` 列表。
    *   **分批处理 (关键)**:
        *   不要一次性处理所有成员，这可能会消耗大量内存。将成员列表分批，例如每批 200 人。
        *   在一个循环中处理每一批成员：
            1.  调用 `repoClient` 的 `GetUsersSession(user_id_list)` 方法，**批量**查询这批成员的在线状态和 `gateway_id`。
            2.  遍历查询结果，对于每个在线的成员，构造下行消息体。
            3.  调用 Kafka `Producer` 将消息生产到对应的 `im-downstream-topic-{gateway_id}`。
    *   **错误处理**: 仔细处理 gRPC 调用失败和 Kafka 生产失败的情况。对于可重试的错误，可以考虑将任务重新投递到 Kafka（需要增加重试次数字段）。
3.  **注册处理器**:
    *   在 `cmd/main.go` 中，创建 `FanoutProcessor` 的实例，并将其注册到 `TaskDispatcher` 中：
        ```go
        fanoutProcessor := processor.NewFanoutProcessor(...)
        dispatcher.Register("large_group_fanout", fanoutProcessor)
        ```

**验收标准**:
*   启动全套服务 (`etcd`, `redis`, `kafka`, `im-repo`, `im-logic`, `im-task`)。
*   通过 `im-logic` 的逻辑（或手动模拟），向 `im-task-topic` 发送一个大群扩散任务。
*   在 `im-task` 的日志中，可以看到任务被正确处理，包括分批逻辑。
*   在 Kafka 中，可以观测到大量的下行消息被生产到了不同的 `im-downstream-topic`。

---

### **5. 总结与未来扩展**

完成以上三个阶段后，`im-task` 就拥有了一个健壮的核心框架和第一个关键功能。

**如何添加新任务 (如离线推送)?**
1.  在 `internal/processor/` 目录下创建一个新的 `push_processor.go` 文件。
2.  让 `PushProcessor` 实现 `TaskProcessor` 接口。
3.  在 `Process` 方法中编写调用 APNs/FCM 的逻辑。
4.  在 `cmd/main.go` 中注册新的处理器：`dispatcher.Register("offline_push", pushProcessor)`。
5.  完成！整个系统的其他部分无需任何改动。

`im-task` 的价值在于隔离复杂性。请务必保持其处理器之间的独立性，并编写健壮的错误处理与重试逻辑。祝你开发顺利！