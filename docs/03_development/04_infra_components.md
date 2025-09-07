# 使用 `im-infra` 组件

`im-infra` 目录包含在所有微服务中使用的共享库。本指南解释如何使用关键组件。

## 1. `clog` - 结构化日志

`clog` 库提供标准化的结构化日志接口。

-   **初始化**: 日志记录器通常在 `main.go` 中初始化并传递给其他组件。
-   **用法**:
    ```go
    import "github.com/ceyewan/gochat/im-infra/clog"

    // 获取特定模块的日志记录器
    logger := clog.Module("user-service")

    // 使用结构化上下文记录消息
    logger.Info("用户已登录",
        clog.String("username", "test"),
        clog.Uint64("user_id", 123),
    )

    logger.Error("无法连接到数据库", clog.Err(err))
    ```
-   **配置**: 日志记录器通过 `clog.json` 文件配置，该文件由 `coord` 服务加载。有关详细信息，请参阅[配置管理](./../../04_deployment/02_configuration.md)指南。

## 2. `coord` - 分布式协调

`coord` 库为分布式协调服务提供接口，包括服务发现、配置管理和分布式锁，使用 `etcd` 作为后端。

-   **初始化**: `coord.Provider` 在 `main.go` 中创建，用于访问不同的协调服务。
    ```go
    import "github.com/ceyewan/gochat/im-infra/coord"

    // cfg 从配置文件加载
    coordinator, err := coord.New(context.Background(), cfg)
    if err != nil {
        // 处理错误
    }
    defer coordinator.Close()
    ```

### 服务发现和 gRPC

-   `coord` 库与 gRPC 集成以提供客户端负载平衡。
-   **获取 gRPC 客户端连接**:
    ```go
    // 获取到 "user-service" 的连接
    conn, err := coordinator.Registry().GetConnection(ctx, "user-service")
    if err != nil {
        // 处理错误
    }
    defer conn.Close()

    // 创建 gRPC 客户端
    userClient := userpb.NewUserServiceClient(conn)
    ```

### 配置管理

-   服务使用 `coord` 提供程序从 `etcd` 获取其配置。
-   **获取配置**:
    ```go
    var dbConfig myapp.DatabaseConfig
    err := coordinator.Config().Get(ctx, "/config/dev/im-repo/db", &dbConfig)
    if err != nil {
        // 处理错误
    }
    ```

### 分布式锁

-   `coord` 库提供了获取和释放分布式锁的简单接口。
-   **获取锁**:
    ```go
    // 获取具有 30 秒 TTL 的锁
    lock, err := coordinator.Lock().Acquire(ctx, "my-resource-key", 30*time.Second)
    if err != nil {
        // 处理错误（例如，锁已被持有）
    }
    defer lock.Unlock(ctx)

    // ... 执行关键部分工作 ...
    ```

有关 `coord` 模块的设计和功能的更多详细信息，请参阅其[设计文档](../../../im-infra/coord/DESIGN.md)和[README](../../../im-infra/coord/README.md)。