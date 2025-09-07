# `im-infra`: 共享基础设施组件库

`im-infra` 是 GoChat 项目的基石，它为所有微服务提供了一套统一、标准化的基础设施组件。其核心目标是封装复杂性、统一技术栈、提高开发效率和系统的可维护性。

## 1. 核心设计原则

-   **高内聚，低耦合**: 每个组件专注于一个特定问题，对外提供简洁、稳定的接口。
-   **面向接口编程**: 业务代码应依赖于 `im-infra` 定义的抽象接口，而非具体实现。
-   **统一配置**: 所有组件都应能通过 `coord` 服务从配置中心加载配置，并支持优雅降级。
-   **生产就绪**: 所有组件都应内置日志、指标、错误处理和优雅关闭等生产级功能。

---

## 2. 核心组件 API 设计

本节详细定义了 `im-infra` 中各个核心组件的职责和关键接口。

### 2.1 `clog` - 上下文日志服务

-   **职责**: 提供全系统统一的、上下文感知的结构化日志记录方案。
-   **核心功能**:
    -   自动注入 `service` 名称和从 `context.Context` 中获取的 `trace_id`。
    -   所有日志以 JSON 格式输出到标准输出。
-   **关键接口**:
    ```go
    // Logger 定义了日志记录器的接口
    type Logger interface {
        Info(msg string, fields ...Field)
        Error(msg string, fields ...Field)
        // ... 其他级别
    }

    // 从上下文中获取一个预置了 trace_id 的 Logger
    func FromContext(ctx context.Context) Logger

    // 将 trace_id 注入到上下文中
    func SetTraceID(ctx context.Context, traceID string) context.Context
    ```

### 2.2 `coord` - 分布式协调服务

-   **职责**: 基于 etcd 提供服务发现、配置管理、分布式锁和实例 ID 分配。
-   **关键接口**:
    ```go
    // Provider 是 coord 服务的总入口
    type Provider interface {
        Registry() ServiceRegistry // 服务注册与发现
        Config() ConfigManager   // 配置管理
        Lock() Locker            // 分布式锁
        Instance() Instancer     // 实例ID管理器
        Close() error
    }

    // Instancer 定义了实例ID分配器的接口
    type Instancer interface {
        // 获取当前服务实例的唯一ID，该ID在服务重启后会改变
        GetInstanceID() (int64, error)
    }
    ```

### 2.3 `db` - 数据库访问层

-   **职责**: 封装 GORM，提供统一的数据库连接和事务管理。
-   **关键接口**:
    ```go
    // Provider 提供了访问数据库的能力
    type Provider interface {
        // 获取一个 gorm.DB 实例用于执行查询
        DB() *gorm.DB
        // 开始一个事务
        BeginTx(ctx context.Context) (Transaction, error)
    }

    // Transaction 定义了事务的接口
    type Transaction interface {
        DB() *gorm.DB // 在事务中执行操作
        Commit() error
        Rollback() error
    }
    ```

### 2.4 `mq` - 消息队列服务

-   **职责**: 封装 Kafka 生产者和消费者的通用逻辑，简化消息收发。
-   **关键接口**:
    ```go
    // Producer 定义了消息生产者的接口
    type Producer interface {
        // Publish 方法将 trace_id 从 context 中提取并注入到消息头
        Publish(ctx context.Context, topic string, message Message) error
    }

    // Consumer 定义了消息消费者的接口
    type Consumer interface {
        // Subscribe 的 handler 回调函数收到的 context 中已包含 trace_id
        Subscribe(ctx context.Context, topic string, handler func(ctx context.Context, msg Message) error)
    }
    ```

### 2.5 `uid` - 分布式唯一 ID 生成器

-   **职责**: 基于雪花算法（Snowflake）生成全局唯一的64位 ID。
-   **核心功能**: 在初始化时必须接收由 `coord.Instance()` 分配的 `instanceID` 作为 worker ID。
-   **关键接口**:
    ```go
    // Generator 定义了唯一ID生成器的接口
    type Generator interface {
        NextID() int64
    }

    // NewGenerator 创建一个新的ID生成器
    func NewGenerator(instanceID int64) (Generator, error)
    ```

### 2.6 `cache` - 分布式缓存服务

-   **职责**: 提供对 Redis 的标准化、统一的访问接口。
-   **关键接口**:
    ```go
    // Cache 定义了缓存操作的接口
    type Cache interface {
        Get(ctx context.Context, key string) (string, error)
        Set(ctx context.Context, key string, value interface{}, expiration time.Duration) error
        Del(ctx context.Context, keys ...string) error
    }