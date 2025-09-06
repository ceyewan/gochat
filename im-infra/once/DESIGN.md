# idempotent 组件设计文档

## 概述

idempotent 是一个轻量级、高性能的分布式幂等组件，专为 GoChat 系统设计。基于 Redis 的 SETNX 命令实现，提供简洁易用的幂等操作能力。

## 设计目标

- **简洁性**：提供最小但完整的幂等操作接口
- **高性能**：基于 Redis 原子操作，最小化网络开销
- **易用性**：直观的 API 设计，支持全局方法和自定义客户端
- **可靠性**：完善的错误处理和并发控制
- **可扩展性**：模块化设计，易于扩展和维护

## 架构设计

### 组件结构

```
im-infra/once/
├── idempotent.go        # 公共 API 和全局方法
├── internal/
│   ├── interfaces.go    # 核心接口定义
│   ├── config.go       # 配置管理
│   └── client.go       # 核心实现
├── examples/           # 使用示例
├── idempotent_test.go  # 测试用例
├── README.md          # 用户文档（中文）
└── DESIGN.md          # 设计文档（中文）
```

### 核心接口

```go
type Idempotent interface {
    // Check 检查指定键是否已经存在
    Check(ctx context.Context, key string) (bool, error)
    
    // Set 设置幂等标记，如果键已存在则返回 false
    Set(ctx context.Context, key string, ttl time.Duration) (bool, error)
    
    // Do 执行幂等操作，如果已执行过则跳过，否则执行函数
    Do(ctx context.Context, key string, f func() error) error
    
    // SetWithResult 设置幂等标记并存储操作结果
    SetWithResult(ctx context.Context, key string, result interface{}, ttl time.Duration) (bool, error)
    
    // GetResult 获取存储的操作结果
    GetResult(ctx context.Context, key string) (interface{}, error)
    
    // Delete 删除幂等标记
    Delete(ctx context.Context, key string) error
    
    // 其他辅助方法...
}
```

### 设计原则

1. **接口驱动设计**：清晰的接口分离，便于测试和模拟
2. **配置驱动**：简洁的配置结构，支持环境特定配置
3. **函数式选项**：灵活的配置方式，支持依赖注入
4. **结构化日志**：与 GoChat 日志基础设施集成
5. **错误处理**：完善的错误传播和处理机制

## 核心实现

### 幂等性保证

基于 Redis 的 SETNX（Set if Not Exists）命令实现：

```go
// Set 操作的核心逻辑
func (c *client) Set(ctx context.Context, key string, ttl time.Duration) (bool, error) {
    formattedKey := c.formatKey(key)
    
    // 使用 SETNX 实现原子性操作
    success, err := c.cache.SetNX(ctx, formattedKey, "1", ttl)
    if err != nil {
        return false, fmt.Errorf("failed to set idempotent key: %w", err)
    }
    
    return success, nil
}
```

### Do 操作实现

核心的幂等操作，参考 sync.Once 的设计模式：

```go
func (c *client) Do(ctx context.Context, key string, f func() error) error {
    // 1. 检查是否已经执行过
    exists, err := c.Check(ctx, key)
    if err != nil {
        return fmt.Errorf("failed to check key existence: %w", err)
    }
    
    if exists {
        return nil // 已执行过，直接返回
    }
    
    // 2. 尝试设置幂等标记
    success, err := c.Set(ctx, key, c.config.DefaultTTL)
    if err != nil {
        return fmt.Errorf("failed to set idempotent key: %w", err)
    }
    
    if !success {
        return nil // 并发情况下其他协程已设置
    }
    
    // 3. 首次执行，调用函数
    if err := f(); err != nil {
        // 执行失败，删除标记允许重试
        c.Delete(ctx, key)
        return fmt.Errorf("function execution failed: %w", err)
    }
    
    return nil
}
```

### 结果存储

支持存储操作结果，避免重复计算：

```go
func (c *client) SetWithResult(ctx context.Context, key string, result interface{}, ttl time.Duration) (bool, error) {
    // 序列化结果
    serializedResult, err := json.Marshal(result)
    if err != nil {
        return false, fmt.Errorf("failed to serialize result: %w", err)
    }
    
    // 使用 SETNX 存储序列化后的结果
    return c.cache.SetNX(ctx, formattedKey, string(serializedResult), ttl)
}
```

## 配置设计

### 配置结构

```go
type Config struct {
    CacheConfig cache.Config  // Redis 连接配置
    KeyPrefix   string        // 键前缀，用于业务隔离
    DefaultTTL  time.Duration // 默认过期时间
}
```

### 预设配置

- **DefaultConfig()**: 开发环境默认配置
- **DevelopmentConfig()**: 开发环境专用配置
- **ProductionConfig()**: 生产环境专用配置
- **TestConfig()**: 测试环境专用配置

### 配置验证

```go
func (c *Config) Validate() error {
    if c.KeyPrefix == "" {
        return fmt.Errorf("key_prefix cannot be empty")
    }
    
    if c.DefaultTTL < 0 {
        return fmt.Errorf("default_ttl cannot be negative")
    }
    
    return nil
}
```

## 并发控制

### 竞态条件处理

在高并发场景下，多个协程可能同时尝试执行同一幂等操作：

1. **检查阶段**：多个协程同时检查发现操作未执行
2. **设置阶段**：通过 SETNX 的原子性，只有一个协程能成功设置标记
3. **执行阶段**：成功的协程执行操作，其他协程跳过

### 错误恢复

操作执行失败时的处理策略：

1. **自动清理**：执行失败时自动删除幂等标记
2. **允许重试**：清理后允许后续请求重试操作
3. **错误传播**：原始错误被包装并返回给调用者

## 性能优化

### Redis 操作优化

1. **最小化命令**：每个操作使用最少数量的 Redis 命令
2. **连接复用**：复用 cache 组件的连接池
3. **管道优化**：批量操作使用 Redis 管道

### 内存使用优化

1. **零分配路径**：常用操作路径避免内存分配
2. **对象池化**：考虑对临时对象使用对象池
3. **GC 友好**：减少短生命周期对象的创建

## 错误处理

### 错误类型

```go
var (
    ErrInvalidKey      = errors.New("invalid key format")
    ErrCacheOperation  = errors.New("cache operation failed")
    ErrSerialization   = errors.New("serialization failed")
)
```

### 错误处理策略

1. **包装错误**：所有底层错误都被包装，提供上下文信息
2. **日志记录**：关键错误被记录到日志系统
3. **错误分类**：区分可重试错误和永久错误

## 测试策略

### 单元测试

- 配置验证测试
- 幂等性验证测试
- 并发安全测试
- 错误处理测试

### 集成测试

- Redis 连接测试
- 并发执行测试
- TTL 功能测试
- 结果存储测试

## 部署考虑

### Redis 配置

1. **高可用**：生产环境使用 Redis 集群或哨兵模式
2. **持久化**：根据业务需求配置合适的持久化策略
3. **内存管理**：设置合理的内存限制和淘汰策略

### 监控指标

1. **操作成功率**：幂等操作的成功/失败比率
2. **Redis 延迟**：命令执行延迟统计
3. **并发冲突**：并发设置冲突的频率
4. **内存使用**：存储数据的空间占用

## 安全考虑

### 键名安全

1. **注入防护**：验证键名格式，防止注入攻击
2. **长度限制**：限制键名最大长度
3. **字符过滤**：过滤危险字符

### 数据安全

1. **序列化安全**：使用安全的序列化格式
2. **访问控制**：确保 Redis 访问权限控制
3. **数据加密**：敏感数据考虑加密存储

## 扩展方向

### 未来增强

1. **批量操作**：支持批量幂等操作
2. **分布式锁**：集成更复杂的分布式锁机制
3. **事件驱动**：支持幂等操作的事件通知
4. **指标监控**：集成指标收集和监控

### 性能优化

1. **本地缓存**：考虑本地缓存热点数据
2. **异步处理**：支持异步幂等操作
3. **分片存储**：大数据量的分片存储策略

## 总结

idempotent 组件提供了一个轻量级、高性能的分布式幂等解决方案。其简洁的设计使得集成和使用变得非常容易，同时保证了在分布式环境下的可靠性和性能。通过合理的架构设计和错误处理机制，能够满足 GoChat 系统对幂等操作的各种需求。