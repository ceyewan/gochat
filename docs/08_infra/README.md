# 规范: `im-infra` 基础设施层核心设计

## 1. 哲学

`im-infra` 是 GoChat 项目的基石。本规范是构建其所有组件的“宪法”，旨在保证整个基础设施层在架构上的一致性、可维护性和健壮性。

> **寻找用法？** 请直接阅读 -> **[快速上手指南 (Usage Guide)](./usage_guide.md)**

---

## 2. 接口契约 (The Interface Contract)

所有组件都必须遵循统一的接口设计模式，这是它们之间以及与业务层之间交互的“契约法”。

### 2.1 Provider 模式与构造函数

所有有状态、有配置或有外部依赖的组件，都必须通过 `Provider` 模式暴露其能力，并提供一个遵循标准签名的 `New` 构造函数。

- **标准签名**:
  ```go
  New(ctx context.Context, config *Config, opts ...Option) (Provider, error)
  ```
- **`config` 职责**: `config` 结构体负责传递组件自身的核心、静态配置（例如 `ratelimit` 的规则路径 `RulesPath`）。
- **`opts` 职责**: `opts ...Option` 负责以函数式选项模式传递所有**外部依赖**（例如 `clog.Logger`, `coord.Provider`）。
- **例外情况**: 对于完全无状态、无依赖的纯工具包（例如，一个纯粹的数学计算或字符串处理库），可以直接提供包级函数，无需 `Provider` 模式。

### 2.2 默认配置 (`GetDefaultConfig`)

为了降低组件的接入成本，所有需要配置的组件**必须**提供一个 `GetDefaultConfig` 函数。

- **标准签名**:
  ```go
  GetDefaultConfig(env string) *Config
  ```
- **环境参数**: `env` 参数必须支持 `"development"` 和 `"production"` 两种值，并返回针对不同环境优化的合理默认配置。
- **覆盖机制**: 调用方在获取默认配置后，仍可根据实际部署环境覆盖特定的配置项（如连接地址、认证信息等）。

### 2.3 接口方法

- **上下文感知 (Context-Aware)**: 所有可能发生 I/O、阻塞或需要传递追踪信息的接口方法，**必须**接受 `context.Context` 作为其第一个参数。
- **最小化原则 (Keep It Simple)**: 接口应保持最小化，只暴露真正需要被外部调用的核心功能。避免提供仅为方便内部实现的公共方法。

---

## 3. 配置契约 (The Configuration Contract)

所有组件都必须遵循统一的配置管理模式，这是保证系统动态性和可运维性的“行政法”。

### 3.1 声明式配置

- 所有可变的配置（如日志级别、限流规则）都应通过 `coord` 在配置中心进行**声明式**管理。
- 组件**不应**提供命令式的配置修改 API（如 `SetLogLevel`, `AddRule`）。系统的状态应由配置中心这个“唯一真实来源”驱动。

### 3.2 组件自治的动态配置

- **标准模式**: 需要支持配置热更新的组件，必须采用“组件自治”模式。
- **实现方式**: 在组件的 `New` 函数中，自行使用注入的 `coord.Provider` 的 `Watch` 功能来监听自身的配置变更，并以线程安全的方式更新内部状态。
- **`coord.Watch` 接口约定**: `coord` 组件的 `ConfigCenter.Watch` 接口必须支持前缀监听。当 `Watch` 方法接收的 `key` 参数以 `/` 结尾时，它应自动启用对该前缀下所有键的监听。

---

## 4. 文档契约 (The Documentation Contract)

### 4.1 核心组件与文档

对于希望在服务中使用这些组件的开发者，请从这里开始：

- **[快速上手指南 (Usage Guide)](./usage_guide.md)**: **(首选阅读)** 提供了覆盖所有组件的、生产级别的统一初始化范例和核心用法。

当您需要深入了解某个特定组件的设计理念和完整 API 时，请参考以下官方“契约”文档 (按字母排序):

- **[熔断器 (breaker)](./breaker.md)**
- **[缓存 (cache)](./cache.md)**
- **[日志 (clog)](./clog.md)**
- **[分布式协调 (coord)](./coord.md)**
- **[数据库 (db)](./db.md)**
- **[可观测性 (metrics)](./metrics.md)**
- **[消息队列 (mq)](./mq.md)**
- **[幂等操作 (once)](./once.md)**
- **[分布式限流 (ratelimit)](./ratelimit.md)**
- **[唯一ID (uid)](./uid.md)**
