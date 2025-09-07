# 任务书：重构 `im-infra/uid` 组件 (V2)

## 1. 背景与目标

**背景**: 当前的 `im-infra/uid` 组件将雪花算法和多种 UUID 生成方式耦合在一起，接口定义不清，配置复杂。

**目标**: 对 `uid` 组件进行彻底重构，将其拆分为两个独立的、职责清晰的包：`snowflake` 和 `uuid`。新的实现必须简洁、高效，并遵循我们最新的架构决策。

## 2. 核心要求

### 2.1 `snowflake` 包

1.  **状态依赖**: 雪花算法的实现是**有状态的**。它必须依赖一个在 `[0, 1023]` 范围内的、全局唯一的 `instanceID`。
2.  **ID 获取**: 这个 `instanceID` **严禁**通过配置文件或手动方式传入。它必须通过 `im-infra/coord` 组件的 `InstanceIDAllocator` 服务动态获取。
3.  **接口定义**:
    *   提供一个工厂函数 `New(instanceID int64) (Generator, error)`。
    *   提供一个接口 `Generator`，其中只包含一个方法 `Generate() int64`。
4.  **位分配**: 新的雪花算法实现应使用 41 位时间戳、**10 位 `instanceID`** 和 12 位序列号。原有的 `datacenterID` 和 `workerID` 概念被废弃。
5.  **工具函数**: 提供一个独立的包级别函数 `Parse(id int64) (timestamp, instanceID, sequence int64)` 用于解析 ID。

### 2.2 `uuid` 包

1.  **无状态**: UUID 的生成是**无状态的**。
2.  **接口定义**:
    *   **不应**提供任何工厂函数 (`New`) 或接口 (`Generator`)。
    *   直接提供一个包级别的函数 `NewV7() string`，用于生成符合 RFC 规范的、时间有序的 UUID v7。
    *   （可选）提供 `IsValid(s string) bool` 用于验证。

## 3. 开发步骤

1.  **创建新目录**: 在 `im-infra/uid/` 下创建 `snowflake/` 和 `uuid/` 两个新目录。
2.  **实现 `uuid` 包**:
    *   在 `im-infra/uid/uuid/uuid.go` 中，基于 `github.com/google/uuid` 实现 `NewV7()` 函数。
    *   添加单元测试。
3.  **实现 `snowflake` 包**:
    *   在 `im-infra/uid/snowflake/snowflake.go` 中定义 `Generator` 接口和 `New(instanceID int64)` 工厂函数。
    *   实现雪花算法逻辑，确保使用 10 位的 `instanceID`。
    *   实现 `Parse` 工具函数。
    *   编写单元测试，重点测试 ID 唯一性、并发安全性和 `instanceID` 的正确嵌入。
4.  **清理旧代码**:
    *   删除 `im-infra/uid/` 目录下所有旧的 `.go` 和 `_test.go` 文件。
    *   更新 `im-infra/uid/README.md` 以反映新的、解耦后的设计。
5.  **更新调用方 (示例)**:
    *   **获取雪花 ID**:
        ```go
        // 1. 从 coord 获取 instanceID
        idAllocator, _ := coordinator.InstanceIDAllocator("my-service", 1023)
        instanceID, _ := idAllocator.AcquireID(ctx)
        // 2. 创建生成器
        snowGen, _ := snowflake.New(int64(instanceID))
        // 3. 生成 ID
        messageID := snowGen.Generate()
        ```
    *   **获取 UUID**:
        ```go
        requestID := uuid.NewV7()
        ```

## 4. 验收标准

1.  旧的 `uid` 实现已被完全移除。
2.  新的 `snowflake` 和 `uuid` 包已按要求实现。
3.  `snowflake.New` 严格依赖 `instanceID` 参数。
4.  `uuid` 包是无状态的，只提供包级别函数。
5.  所有代码都通过了单元测试。