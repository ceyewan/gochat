# etcd-grpc 服务注册与发现组件

etcd-grpc 是一个基于 etcd 构建的企业级服务注册与发现组件，专为 gRPC 微服务架构设计。它提供了完整的分布式服务管理功能，包括服务注册发现、分布式锁、租约管理等核心特性。

## 🚀 核心功能

- **服务注册与发现** - 自动服务注册、实时服务发现、负载均衡
- **分布式锁** - 基于 etcd 的可靠分布式锁实现
- **租约管理** - 统一的租约生命周期管理和自动续约
- **连接管理** - 连接池、健康检查、自动重连
- **配置管理** - 灵活的配置选项和建造者模式
- **错误处理** - 结构化错误处理和重试机制

## 🏗️ 架构概览

```
┌─────────────────┐    ┌─────────────────┐    ┌─────────────────┐
│   EtcdManager   │────│ ServiceRegistry │    │ ServiceDiscovery│
│   (统一管理)     │    │   (服务注册)     │    │   (服务发现)     │
└─────────────────┘    └─────────────────┘    └─────────────────┘
         │
         ├─────────────────┬─────────────────┬─────────────────┐
         │                 │                 │                 │
┌─────────────────┐ ┌─────────────────┐ ┌─────────────────┐ ┌─────────────────┐
│ DistributedLock │ │  LeaseManager   │ │ConnectionManager│ │   gRPC Resolver │
│   (分布式锁)     │ │   (租约管理)      │ │   (连接管理)     │ │   (解析器集成)    │
└─────────────────┘ └─────────────────┘ └─────────────────┘ └─────────────────┘
```

## � 快速开始

### 基本使用

```go
package main

import (
    "context"
    "log"
    "time"

    "github.com/your-org/etcd-grpc/etcd"
)

func main() {
    // 快速启动 - 自动使用配置文件或默认配置
    manager, err := etcd.QuickStart()
    if err != nil {
        log.Fatalf("Failed to start: %v", err)
    }
    defer manager.Close()

    ctx := context.Background()

    // 服务注册
    registry := manager.ServiceRegistry()
    err = registry.Register(ctx, "user-service", "instance-1", "localhost:50051")
    if err != nil {
        log.Fatalf("Failed to register: %v", err)
    }

    // 服务发现
    discovery := manager.ServiceDiscovery()
    conn, err := discovery.GetConnection(ctx, "user-service")
    if err != nil {
        log.Fatalf("Failed to discover: %v", err)
    }
    defer conn.Close()

    // 分布式锁
    lock := manager.DistributedLock()
    err = lock.Lock(ctx, "my-lock", 30*time.Second)
    if err != nil {
        log.Fatalf("Failed to acquire lock: %v", err)
    }
    defer lock.Unlock(ctx, "my-lock")
}
```

### 高级配置

```go
// 使用建造者模式创建自定义配置
manager, err := etcd.NewManagerBuilder().
    WithEndpoints([]string{"etcd1:2379", "etcd2:2379", "etcd3:2379"}).
    WithDialTimeout(10 * time.Second).
    WithDefaultTTL(60).
    WithServicePrefix("/prod/services").
    WithLockPrefix("/prod/locks").
    WithHealthCheck(30*time.Second, 5*time.Second).
    WithRetryConfig(&etcd.RetryConfig{
        MaxRetries:      5,
        InitialInterval: 200 * time.Millisecond,
        MaxInterval:     5 * time.Second,
        Multiplier:      2.0,
    }).
    Build()
```

## ⚙️ 配置管理

### 配置优先级

etcd-grpc 支持灵活的配置管理，按以下优先级顺序加载配置：

1. **用户输入参数** (最高优先级)
2. **配置文件**
3. **默认值** (最低优先级)

### 配置文件示例

#### JSON 格式 (etcd-config.json)

```json
{
  "endpoints": ["localhost:23791", "localhost:23792", "localhost:23793"],
  "dial_timeout": "5s",
  "log_level": "info"
}
```

#### YAML 格式 (etcd-config.yaml)

```yaml
endpoints:
  - localhost:23791
  - localhost:23792
  - localhost:23793
dial_timeout: 5s
log_level: info
```

### 配置文件位置

系统会自动查找以下配置文件（按优先级顺序）：

- `etcd-config.json`
- `etcd-config.yaml`
- `etcd-config.yml`
- `config/etcd.json`
- `config/etcd.yaml`
- `config/etcd.yml`

### 使用示例

```go
// 方式1: 自动配置 - 优先使用配置文件，回退到默认值
manager, err := etcd.QuickStart()

// 方式2: 指定端点 - 用户输入覆盖配置文件
manager, err := etcd.QuickStart("custom:2379")

// 方式3: 指定配置文件
config, err := etcd.LoadConfigFromFile("custom-config.json")
if err != nil {
    log.Fatal(err)
}
manager, err := etcd.NewManagerWithConfig(config)
```

## ⚙️ 配置选项

### 连接配置
- `WithEndpoints([]string)` - etcd 服务器地址列表
- `WithDialTimeout(time.Duration)` - 连接超时时间
- `WithAuth(username, password)` - 认证信息
- `WithTLS(*TLSConfig)` - TLS 安全配置

### 服务配置
- `WithDefaultTTL(int64)` - 默认服务 TTL（秒）
- `WithServicePrefix(string)` - 服务注册前缀
- `WithLockPrefix(string)` - 分布式锁前缀
- `WithDefaultMetadata(map[string]string)` - 默认元数据

### 可靠性配置
- `WithHealthCheck(interval, timeout)` - 健康检查配置
- `WithRetryConfig(*RetryConfig)` - 重试策略配置
- `WithConnectionPool(maxIdle, maxActive, maxLifetime)` - 连接池配置

### 预设环境配置

```go
// 开发环境
manager, err := etcd.NewDevelopmentManager()

// 生产环境
manager, err := etcd.NewProductionManager([]string{
    "etcd1.prod.com:2379",
    "etcd2.prod.com:2379",
    "etcd3.prod.com:2379",
})

// 测试环境
manager, err := etcd.NewTestManager()
```

## 🔧 功能特性

### 服务注册与发现
- 自动服务注册和注销
- 支持服务元数据
- 实时服务发现
- gRPC 负载均衡集成
- 服务变化监听

### 分布式锁
- 阻塞和非阻塞锁获取
- 自动锁续约
- 锁状态查询
- 超时处理

### 租约管理
- 租约创建和撤销
- 自动续约机制
- 租约状态监控
- 过期清理

### 连接管理
- 连接池管理
- 健康检查
- 自动重连
- TLS 支持

## 📋 最佳实践

### 配置管理
- **生产环境**：使用配置文件管理端点，避免硬编码
- **开发环境**：创建 `etcd-config.json` 指向本地 etcd 集群
- **测试环境**：使用独立的配置文件和服务前缀
- **配置验证**：在部署前验证配置文件格式和端点可达性

### 生产环境
- 使用 etcd 集群确保高可用性
- 配置适当的超时和重试策略
- 监控服务注册状态和连接健康
- 合理设置连接池大小和租约 TTL
- 使用配置文件而非硬编码端点

### 开发测试
- 使用不同的服务前缀区分环境
- 编写单元测试和集成测试
- 使用预设环境配置
- 创建环境特定的配置文件

### 错误处理
- 实现优雅降级机制
- 使用指数退避重试策略
- 记录详细的错误信息
- 检查配置错误和连接错误

### 故障排除

#### 常见问题

1. **连接失败**
   ```bash
   # 检查 etcd 服务状态
   docker ps | grep etcd
   
   # 检查端口是否可达
   telnet localhost 23791
   ```

2. **配置文件未生效**
   ```bash
   # 确认配置文件位置
   ls -la etcd-config.json
   
   # 验证配置文件格式
   cat etcd-config.json | jq .
   ```

3. **默认端点问题**
   ```go
   // 显式指定端点
   manager, err := etcd.QuickStart("localhost:2379")
   ```

## 📚 文档

- [API 文档](API.md) - 完整的 API 参考
- [示例代码](main.go) - 详细的使用示例
- [测试用例](test/) - 单元测试和集成测试

## 🤝 贡献

欢迎提交 Issue 和 Pull Request 来改进这个项目。

## 📄 许可证

本项目采用 MIT 许可证。详见 [LICENSE](LICENSE) 文件。