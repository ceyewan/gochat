# 任务书：实现服务实例的唯一 ID (`instanceID`) 分配

## 1. 背景与目标

**背景**: 在我们的消息队列架构中，`im-gateway` 的每个实例都需要监听一个专属的 Kafka Topic (例如 `gochat.messages.downstream.{instanceID}`)。为此，我们需要一个可靠的机制，为每个启动的 `im-gateway` 实例动态分配一个在 `[1, 1023]` 范围内全局唯一的 `instanceID`。

**目标**: 利用 `im-infra/coord` 组件提供的分布式锁和租约能力，开发一个健壮、高可用的 `instanceID` 分配和回收模块。这个模块将作为 `im-gateway` 启动流程的一部分。

## 2. 设计方案

该方案利用 `coord` 的核心能力，无需引入新的外部依赖。

### 流程设计

1.  **ID 池定义**: 在 `etcd` 中创建一个路径，如 `/instance_ids/im-gateway/`，用于表示可用的 ID 池。
2.  **获取 ID**:
    a. `im-gateway` 实例启动时，它会尝试获取一个全局锁，例如 `/locks/assign_instance_id/im-gateway`。
    b. 获取锁成功后，它会从 `etcd` 中读取 `/instance_ids/im-gateway/` 下所有已分配的 ID。
    c. 它会从 `1` 到 `1023` 遍历，找到第一个未被分配的 ID。
    d. 找到可用 ID 后，它会在 `/instance_ids/im-gateway/{id}` 路径下创建一个带有 **租约（Lease）** 的键，值为自己的服务实例信息（如 Pod IP 或主机名）。
    e. 释放全局锁。
3.  **ID 续约与释放**:
    a. 这个与 ID 绑定的租约与该 `im-gateway` 实例的服务注册租约是 **同一个**。
    b. 只要服务实例存活并成功向 `coord` 续约，它的 `instanceID` 就会一直被保留。
    c. 如果服务实例崩溃或正常关闭，其租约会自动过期，`etcd` 会自动删除 `/instance_ids/im-gateway/{id}` 这个键。
    d. 这样，该 ID 就会被自动释放，可供下一个启动的实例使用。

### 优势

- **高可用**: 整个分配过程是分布式的，没有单点故障。
- **一致性**: 利用分布式锁确保了 ID 分配的原子性，不会出现 ID 冲突。
- **自动回收**: 利用租约机制实现了 ID 的自动回收，无需手动清理。

## 3. 开发步骤

1.  **创建 `InstanceIDAllocator` 接口**:
    在 `im-gateway` 的 `internal` 目录中，定义一个分配器接口。
    ```go
    package allocator

    type InstanceIDAllocator interface {
        // AcquireID 获取一个唯一的实例ID。此方法会阻塞直到成功获取或上下文超时。
        AcquireID(ctx context.Context) (int, error)
        // ReleaseID 释放当前实例持有的ID。通常在服务正常关闭时调用。
        ReleaseID(ctx context.Context) error
    }
    ```
2.  **实现 `etcdAllocator`**:
    -   创建一个 `etcdAllocator` 结构体，它接收一个 `coord.Provider` 作为依赖。
    -   实现 `AcquireID` 方法，严格按照上述“设计方案”中的流程执行。
    -   实现 `ReleaseID` 方法，主动删除在 `etcd` 中对应的 ID 键。
3.  **集成到 `im-gateway`**:
    -   在 `im-gateway` 的启动流程中，初始化 `etcdAllocator`。
    -   调用 `AcquireID` 方法获取 `instanceID`。
    -   将获取到的 `instanceID` 用于构建 Kafka consumer 要监听的 Topic 名称。
    -   使用 `defer` 或 Go 的 `signal` 处理，确保在服务关闭时调用 `ReleaseID` 方法。

## 4. 验收标准

1.  连续启动多个 `im-gateway` 实例，每个实例都能获取到 `[1, 1023]` 范围内的唯一 `instanceID`。
2.  `etcd` 中 `/instance_ids/im-gateway/` 路径下能看到正确创建的、带租约的键。
3.  正常关闭一个 `im-gateway` 实例后，其在 `etcd` 中对应的 ID 键被删除。
4.  强制 kill 一个 `im-gateway` 实例后，等待租约过期（例如 30 秒后），其在 `etcd` 中对应的 ID 键被自动删除。
5.  所有 `im-gateway` 实例都关闭后，再次启动新实例，能够复用之前被释放的 ID。