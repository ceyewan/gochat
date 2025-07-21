# clog 架构设计

本文档阐述了 `clog` 日志库的内部架构，主要面向需要进行二次开发或深度定制的开发者。

## 1. 设计哲学

- **接口驱动**: 核心组件均基于接口设计，实现高内聚、低耦合。
- **职责单一**: 每个组件只做一件事，如注册、构建、合并配置等。
- **默认实现**: 提供开箱即用的默认实现，同时允许完全替换。
- **依赖注入**: 支持通过依赖注入的方式构建服务，便于测试和扩展。

## 2. 核心组件与分层

`clog` 采用经典的分层架构，各层职责清晰：

```
+---------------------+
|      API Layer      |  (clog.go) - 全局函数，提供便捷访问
+---------------------+
|    Service Layer    |  (service.go) - 编排核心逻辑，如模块创建
+---------------------+
|  Repository Layer   |  (registry.go) - 管理日志器实例状态
+---------------------+
|    Factory Layer    |  (factory.go) - 负责 zap 实例的复杂创建过程
+---------------------+
|      Core Layer     |  (interfaces.go, config.go) - 定义接口和数据结构
+---------------------+
```

### 2.1. 核心接口 (`interfaces.go`)

- `Logger`: 日志记录器接口，定义了所有日志操作。
- `LoggerRegistry`: 日志器注册表接口，负责存储和检索 `Logger` 实例。
- `LoggerService`: 日志服务接口，封装了 `Init` 和 `Module` 等高级功能。

### 2.2. 全局服务 (`manager.go`)

`clog` 维护一个全局的 `LoggerService` 单例 (`globalLoggerService`)，所有全局函数（如 `clog.Info`）都通过此实例提供服务。

## 3. 工作流程示例：`clog.Module("db")`

1. **API Layer**: `clog.Module("db")` 调用 `globalLoggerService.GetOrCreateModule("db")`。
2. **Service Layer**:
   a. `LoggerService` 首先请求 `LoggerRegistry` 查找名为 "db" 的日志器。
   b. 如果找到，直接返回。
   c. 如果未找到，`LoggerService` 开始创建流程：
      i. 从 `LoggerRegistry` 获取默认日志器的配置。
      ii. 将默认配置与 `Module` 函数传入的 `Option` 合并。
      iii. 调用 `factory.NewLogger()` 创建新的 `Logger` 实例。
      iv. 将新创建的 `Logger` 实例注册到 `LoggerRegistry` 中。
      v. 返回新的 `Logger` 实例。
3. **Factory Layer**: `NewLogger()` 函数负责与 `zap` 库的底层交互，包括创建 `Encoder`、`Core` 和最终的 `zap.Logger`。

## 4. 如何扩展和定制

`clog` 的架构设计使其易于扩展。以下是一些常见的定制场景。

### 4.1. 替换日志器注册表

默认的 `MemoryLoggerRegistry` 将日志器实例存储在内存中。您可以实现自己的 `LoggerRegistry`，例如，将其存储在分布式配置中心（如 etcd）中，以实现集群范围内的日志配置同步。

**步骤**:
1. 实现 `LoggerRegistry` 接口。
2. 使用 `NewLoggerServiceWithDeps` 创建 `LoggerService` 实例。
3. 通过 `SetGlobalLoggerService` 将其设置为全局服务。

```go
type EtcdLoggerRegistry struct { /* ... */ }
// ... 实现 LoggerRegistry 接口 ...

func main() {
    etcdRegistry := NewEtcdLoggerRegistry()
    // ...
    customService := clog.NewLoggerServiceWithDeps(etcdRegistry, clog.NewDefaultConfigMerger())
    clog.SetGlobalLoggerService(customService)

    // 现在 clog 会使用你的 EtcdLoggerRegistry
    clog.Init(...)
}
```

### 4.2. 在单元测试中 Mock 日志器

由于 `clog.Logger` 是一个接口，您可以在测试中轻松地 Mock 它，而无需实际的日志输出。

```go
import "github.com/stretchr/testify/mock"

type MockLogger struct {
    mock.Mock
}

// 实现 Logger 接口的一个方法
func (m *MockLogger) Info(msg string, fields ...clog.Field) {
    m.Called(msg, fields)
}
// ... 实现其他需要的方法 ...

func TestMyService(t *testing.T) {
    mockLogger := new(MockLogger)
    // 注入 mockLogger 到你的服务中
    myService := NewMyService(mockLogger)

    // 设置期望
    mockLogger.On("Info", "操作成功", mock.Anything).Return()

    // 执行测试
    myService.DoSomething()

    // 断言
    mockLogger.AssertExpectations(t)
}
```

### 4.3. 添加自定义配置选项

您可以创建自己的 `Option` 函数来扩展配置能力。

**示例**: 添加一个 `WithHook(hook func())` 选项，在每次记录日志后执行一个钩子函数。

1. 在 `Config` 结构体中添加字段: `PostLogHook func() `
2. 创建 `Option` 函数:
   ```go
   func WithHook(hook func()) clog.Option {
       return func(c *clog.Config) {
           c.PostLogHook = hook
       }
   }
   ```
3. 修改 `factory.go` 或 `logger.go` 的实现，在记录日志后调用此钩子。

## 5. 总结

`clog` 的核心优势在于其**灵活性**和**可测试性**。通过理解其分层架构和接口设计，内部开发者可以：
- **快速上手**: 使用简洁的全局 API 满足 90% 的需求。
- **深度定制**: 替换核心组件以适应特殊的业务场景。
- **自信测试**: 轻松地在单元测试中隔离和 Mock 日志行为。