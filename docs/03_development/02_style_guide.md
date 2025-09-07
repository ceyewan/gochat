# GoChat 代码风格和约定

本文档定义了 GoChat 项目的编码风格、格式化和注释标准。遵循这些约定对于维护代码质量、可读性和整个代码库的一致性至关重要。

## 1. Go 语言风格

-   **格式化**: 所有 Go 代码**必须**使用 `goimports` 进行格式化，它能自动格式化代码并优化 `import` 语句。我们推荐使用 `make fmt` 命令来完成此操作。
    ```bash
    make fmt
    ```
-   **代码检查**: 我们使用 `golangci-lint` 进行静态分析。所有代码在合并前必须通过代码检查。我们推荐使用 `make lint` 命令来运行检查。
    ```bash
    make lint
    ```
    - 完整的检查配置定义在项目根目录的 `.golangci.yml` 文件中。
-   **错误处理**:
    -   应该显式处理错误。不要使用空白标识符 (`_`) 忽略错误。
    -   错误消息应使用小写字母，不要大写。
    -   在添加上下文时，使用带有 `%w` 动词的 `fmt.Errorf` 来包装错误。
-   **变量命名**:
    -   局部变量和函数参数使用 `camelCase`。
    -   导出的标识符（函数、类型、变量）使用 `PascalCase`。
    -   保持变量名称简短但描述性强。避免使用单字母变量名，循环计数器（`i`, `j`）除外。
-   **包命名**:
    -   包名称应简短、简洁且全小写。
    -   避免在包名中使用下划线或 `camelCase`。

## 2. 代码组织

-   **包结构**: 每个微服务都遵循标准的包结构：
    -   `/cmd`: 主应用程序入口点。
    -   `/internal`: 私有应用程序和库代码。
        -   `/internal/service`: 业务逻辑层（gRPC 服务实现）。
        -   `/internal/repository`: 数据访问层。
        -   `/internal/model`: 数据库模型（结构体）。
        -   `/internal/config`: 配置加载。
    -   `/api`: Protobuf 定义和生成的代码。
-   **关注点分离**:
    -   业务逻辑应位于 `service` 层。
    -   数据库和缓存交互应位于 `repository` 层。
    -   HTTP/WebSocket 处理和 gRPC 服务器设置应位于 `server` 或 `cmd` 包中。

## 3. 注释

-   **公共 API**: 所有导出的函数、类型和变量**必须**有文档注释。
    -   注释应以它所描述的标识符名称开头。
    -   示例：`// UserService 是用于用户相关逻辑的服务。`
-   **复杂逻辑**: 添加注释来解释代码的复杂或不明显的部分。解释"为什么"，而不是"什么"。
-   **TODO 注释**: 使用 `// TODO:` 标记需要未来工作的区域。包括需要完成的简短描述。

## 4. 日志记录规范

为了实现高效的日志聚合、查询和基于日志的分布式追踪，所有服务都**必须**遵循以下日志记录规范。

-   **库**: 统一使用 `im-infra/clog` 库进行日志记录。
-   **结构化日志**: 所有日志**必须**是结构化的 JSON 格式。

### 强制性日志字段

每一条日志记录都**必须**包含以下两个字段，`clog` 库应提供机制来自动注入它们：

1.  **`service` (string)**:
    -   **值**: 当前服务的名称（例如, `"im-logic"`, `"im-gateway"`）。
    -   **注入**: 此字段应在服务启动时，通过 `clog` 的全局配置进行设置。

2.  **`trace_id` (string)**:
    -   **值**: 标志单个请求链路的唯一标识符。
    -   **生成**: 在请求入口处（如 `im-gateway` 的 HTTP 处理器）生成。如果上游请求已包含 `trace_id`，则直接使用。
    -   **传递**: **必须**通过 `context.Context` 在整个调用链中（包括 gRPC 调用和 Kafka 消息）进行传递。
    -   **注入**: `clog` 库应能自动从 `context.Context` 中提取 `trace_id` 并添加到日志中。

### 日志级别

-   `Debug`: 用于开发过程中的调试信息。
-   `Info`: 用于记录关键的业务流程节点（如“用户登录成功”）。
-   `Warn`: 用于可预期的、非关键的错误（如“缓存未命中”）。
-   `Error`: 用于需要关注的、影响功能的错误（如“数据库连接失败”）。

### 代码示例

```go
// main.go: 初始化 clog，设置全局 service 字段
logger := clog.New(clog.WithService("im-logic"))

// ---

// gateway: 生成 trace_id 并存入 context
ctx := context.Background()
traceID := newTraceID() // 生成唯一ID
ctx = clog.SetTraceID(ctx, traceID)

// ---

// logic (gRPC handler): 从 context 中获取 logger
// clog 库应提供一个从 context 获取 logger 的方法，该 logger 自动包含 trace_id
logger := clog.FromContext(ctx)

// 记录日志，trace_id 和 service 会被自动添加
logger.Info("开始处理业务", clog.String("param", "value"))
```

## 5. API 和接口定义

-   **Protobuf 风格**: 遵循 [Google Cloud API 设计指南](https://cloud.google.com/apis/design/style_guide) 处理 `.proto` 文件。
-   **清晰性和一致性**: 确保 API 请求和响应在 `.proto` 文件中清晰、一致且文档齐全。