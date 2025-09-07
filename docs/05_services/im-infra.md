# 服务文档: `im-infra` 共享基础设施

## 1. 概述

`im-infra` 是 GoChat 项目的共享基础设施层，它以内部库的形式存在，为所有上层微服务（如 `im-gateway`, `im-logic` 等）提供了一系列标准、统一、生产就绪的基础组件。

其核心目标是：
- **封装复杂性**: 将与分布式系统相关的通用问题（如服务发现、配置管理、消息队列、分布式锁等）进行封装，让业务开发更简单。
- **统一技术栈**: 为所有服务提供一套标准的技术解决方案，避免重复造轮子，降低维护成本。
- **提升开发效率**: 提供简洁、稳定、文档齐全的 API，让业务开发者可以快速构建可靠的服务。
- **保证系统健壮性**: 内置日志、监控、错误处理、优雅关闭等生产级特性，从基础上保证整个系统的稳定性和可观测性。

## 2. 核心设计与开发规范

`im-infra` 的所有组件都遵循一套严格的设计和开发规范，这套规范是保证其质量和一致性的基石。

> **[info] 唯一真实来源 (Single Source of Truth)**
>
> 关于 `im-infra` 库的完整设计理念、统一接口规范、代码组织风格和所有组件的详细“契约”文档，请参阅：
>
> **[im-infra 开发核心规范](../../docs/08_infra/README.md)**

所有 `im-infra` 的使用者和贡献者都应首先阅读并理解上述核心规范文档。

## 3. 组件列表

`im-infra` 库由以下核心组件构成。每个组件的详细设计和用法，请点击链接查阅其官方契约文档。

- **[日志 (clog)](../../docs/08_infra/clog.md)**: 提供高性能的结构化日志记录能力。
- **[分布式协调 (coord)](../../docs/08_infra/coord.md)**: 基于 etcd 实现服务发现、分布式锁和动态配置管理。
- **[缓存 (cache)](../../docs/08_infra/cache.md)**: 提供统一的分布式缓存接口，默认基于 Redis 实现。
- **[数据库 (db)](../../docs/08_infra/db.md)**: 封装了 GORM，提供便捷的数据库操作和分片支持。
- **[消息队列 (mq)](../../docs/08_infra/mq.md)**: 提供了消息生产和消费的统一接口，支持 Kafka。
- **[唯一ID (uid)](../../docs/08_infra/uid.md)**: 提供分布式唯一ID生成方案，包括 Snowflake 和 UUID v7。
- **[可观测性 (metrics)](../../docs/08_infra/metrics.md)**: 基于 OpenTelemetry 实现 Metrics 和 Tracing 的零侵入收集。
- **[幂等操作 (once)](../../docs/08_infra/once.md)**: 基于 Redis 实现的分布式幂等操作保证。
- **[分布式限流 (ratelimit)](../../docs/08_infra/ratelimit.md)**: 基于令牌桶算法的分布式限流解决方案。
- **[优雅重试 (retry)](../../docs/08_infra/retry.md)**: 提供策略驱动的、统一的错误重试机制。
- **[熔断器 (breaker)](../../docs/08_infra/breaker.md)**: 提供服务保护，防止雪崩效应。

## 4. 使用方式

所有上层服务通过 Go Modules 直接依赖 `im-infra` 库。在服务启动时，根据需要初始化相应的组件，并通过依赖注入的方式将其传递给业务逻辑层。

```go
// 示例：一个服务的典型初始化流程
func main() {
    // 1. 初始化核心依赖
    clog.Init(...)
    coordProvider, _ := coord.New(...)
    cacheProvider, _ := cache.New(...)
    dbProvider, _ := db.New(...)
    
    // 2. 初始化业务需要的 infra 组件
    // 注意：组件的配置应从 coordProvider 获取
    mqProducer, _ := mq.NewProducer(...)
    
    // 3. 将 infra 组件注入到业务服务中
    userService := user.NewService(dbProvider, cacheProvider)
    authService := auth.NewService(userService, mqProducer)
    
    // 4. 启动服务
    // ...
}