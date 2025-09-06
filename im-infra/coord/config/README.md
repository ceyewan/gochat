# 通用配置管理器

通用配置管理器是一个基于泛型的配置管理解决方案，为所有基础设施模块提供统一的配置获取、验证、更新和监听能力。

## 功能特性

- **🔧 类型安全**：使用 Go 泛型确保配置类型安全
- **🛡️ 降级策略**：配置中心不可用时自动使用默认配置
- **🔄 热更新**：支持配置热更新和实时监听
- **✅ 配置验证**：支持自定义配置验证器
- **🔄 更新回调**：支持配置更新时的自定义逻辑
- **📝 日志集成**：完整的日志记录和错误处理
- **🔌 无循环依赖**：通过接口抽象避免模块间循环依赖

## 核心接口

### Manager[T]

通用配置管理器，支持任意配置类型：

```go
type Manager[T any] struct {
    // 内部实现
}

// 创建配置管理器
func NewManager[T any](
    configCenter ConfigCenter,
    env, service, component string,
    defaultConfig T,
    opts ...ManagerOption[T],
) *Manager[T]

// 获取当前配置
func (m *Manager[T]) GetCurrentConfig() *T

// 启动配置管理器和监听器
func (m *Manager[T]) Start()

// 停止配置管理器和监听器
func (m *Manager[T]) Stop()

// 重新加载配置
func (m *Manager[T]) ReloadConfig()

// 关闭管理器（向后兼容，推荐使用 Stop）
func (m *Manager[T]) Close()
```

### 可选组件接口

```go
// 配置验证器
type Validator[T any] interface {
    Validate(config *T) error
}

// 配置更新器
type ConfigUpdater[T any] interface {
    OnConfigUpdate(oldConfig, newConfig *T) error
}

// 日志器 - 直接使用 clog.Logger
// import "github.com/ceyewan/gochat/im-infra/clog"
// logger := clog.Module("config")
```

## 生命周期管理

配置管理器支持明确的生命周期管理：

```go
// 创建配置管理器（不自动启动）
manager := config.NewManager(configCenter, "dev", "app", "component", defaultConfig)

// 显式启动（幂等操作，可安全多次调用）
manager.Start()

// 使用配置
currentConfig := manager.GetCurrentConfig()

// 停止管理器（幂等操作，可安全多次调用）
manager.Stop()

// 支持重新启动
manager.Start()
```

**注意**：
- `NewManager()` 创建的管理器需要手动调用 `Start()` 启动
- 便捷工厂函数（`SimpleManager`, `ValidatedManager`, `FullManager`）会自动启动
- `Start()` 和 `Stop()` 是幂等操作，支持重复调用和重新启动

## 使用方法

### 1. 简单配置管理

适用于不需要验证和更新回调的场景：

```go
type MyConfig struct {
    Name  string `json:"name"`
    Value int    `json:"value"`
}

defaultConfig := MyConfig{Name: "default", Value: 100}

manager := config.SimpleManager(
    configCenter,
    "dev", "myapp", "component",
    defaultConfig,
    logger,
)

currentConfig := manager.GetCurrentConfig()
```

### 2. 带验证的配置管理

适用于需要配置验证的场景：

```go
type validator struct{}

func (v *validator) Validate(cfg *MyConfig) error {
    if cfg.Name == "" {
        return fmt.Errorf("name cannot be empty")
    }
    return nil
}

manager := config.ValidatedManager(
    configCenter,
    "dev", "myapp", "component",
    defaultConfig,
    &validator{},
    logger,
)
```

### 3. 完整功能配置管理

适用于需要验证和更新回调的场景：

```go
type updater struct{}

func (u *updater) OnConfigUpdate(old, new *MyConfig) error {
    log.Printf("Config updated: %s -> %s", old.Name, new.Name)
    // 执行更新逻辑
    return nil
}

manager := config.FullManager(
    configCenter,
    "dev", "myapp", "component",
    defaultConfig,
    &validator{},
    &updater{},
    logger,
)
```

### 4. 自定义选项

使用选项模式进行更灵活的配置：

```go
manager := config.NewManager(
    configCenter,
    "dev", "myapp", "component",
    defaultConfig,
    config.WithValidator[MyConfig](&validator{}),
    config.WithUpdater[MyConfig](&updater{}),
    config.WithLogger[MyConfig](logger),
)
```

## 集成示例

### clog 集成

```go
// clog 已经集成了通用配置管理器
clog.SetupConfigCenterFromCoord(configCenter, "dev", "gochat", "clog")

// 使用 clog
logger := clog.Module("example")
logger.Info("Hello from config center!")
```

### db 集成

```go
// db 已经集成了通用配置管理器
db.SetupConfigCenterFromCoord(configCenter, "dev", "gochat", "db")

// 使用 db
database := db.GetDB()
```

### 自定义模块集成

```go
// 在你的模块中
type MyModuleConfig struct {
    // 配置字段
}

var globalConfigManager *config.Manager[MyModuleConfig]

func SetupConfigCenter(configCenter config.ConfigCenter, env, service, component string) {
    defaultConfig := MyModuleConfig{/* 默认值 */}
    globalConfigManager = config.SimpleManager(
        configCenter, env, service, component,
        defaultConfig, logger,
    )
}

func GetCurrentConfig() *MyModuleConfig {
    return globalConfigManager.GetCurrentConfig()
}
```

## 配置路径规则

配置在配置中心中的路径遵循以下规则：

```
/config/{env}/{service}/{component}
```

示例：
- clog 配置：`/config/dev/gochat/clog`
- db 配置：`/config/dev/gochat/db`
- 自定义模块：`/config/prod/myapp/mycomponent`

## 最佳实践

1. **使用默认配置兜底**：始终提供合理的默认配置
2. **实现配置验证**：对关键配置实现验证器
3. **谨慎使用更新器**：更新器中的错误会阻止配置更新
4. **合理的超时设置**：配置获取使用 5 秒超时，避免阻塞启动
5. **日志记录**：提供日志器以便调试配置问题

## 错误处理

配置管理器采用优雅降级策略：

- **配置中心不可用**：使用默认配置，记录警告日志
- **配置格式错误**：使用当前配置，记录错误日志
- **配置验证失败**：使用当前配置，记录警告日志
- **更新器失败**：不更新配置，记录错误日志

这确保了应用在任何情况下都能正常启动和运行。
