# Cache 模块使用指南

## 快速开始

### 1. 基本使用

```go
package main

import (
    "context"
    "time"
    "github.com/ceyewan/gochat/im-infra/cache"
)

func main() {
    ctx := context.Background()
    
    // 字符串操作
    cache.Set(ctx, "user:123", "John Doe", time.Hour)
    user, _ := cache.Get(ctx, "user:123")
    
    // 哈希操作
    cache.HSet(ctx, "user:123:profile", "name", "John")
    cache.HSet(ctx, "user:123:profile", "email", "john@example.com")
    profile, _ := cache.HGetAll(ctx, "user:123:profile")
    
    // 集合操作
    cache.SAdd(ctx, "user:123:tags", "developer", "golang")
    tags, _ := cache.SMembers(ctx, "user:123:tags")
}
```

### 2. 自定义缓存实例

```go
// 创建自定义配置的缓存实例
userCfg := cache.NewConfigBuilder().KeyPrefix("user").Build()
userCache, _ := cache.New(userCfg)

sessionCfg := cache.NewConfigBuilder().KeyPrefix("session").Build()
sessionCache, _ := cache.New(sessionCfg)

// 使用自定义缓存实例
userCache.Set(ctx, "123", userData, time.Hour)
sessionCache.Set(ctx, "abc", sessionData, time.Minute*30)
```

### 3. 自定义配置

```go
// 使用配置构建器
cfg := cache.NewConfigBuilder().
    Addr("redis:6379").
    Password("secret").
    DB(0).
    PoolSize(20).
    KeyPrefix("myapp").
    EnableTracing().
    Build()

cacheInstance, _ := cache.New(cfg)
```

## 高级功能

### 分布式锁

```go
// 获取锁
lock, err := cache.AcquireLock(ctx, "resource:123", time.Minute*5)
if err != nil {
    // 处理锁获取失败
    return
}
defer lock.Unlock(ctx)

// 执行临界区代码
// ...

// 续期锁
lock.Refresh(ctx, time.Minute*10)
```

### 布隆过滤器

```go
// 初始化布隆过滤器
cache.BloomInit(ctx, "users", 1000000, 0.01)

// 添加元素
cache.BloomAdd(ctx, "users", "user123")

// 检查元素
exists, _ := cache.BloomExists(ctx, "users", "user123")
```

## 配置选项

### 预设配置

```go
// 开发环境
cfg := cache.DevelopmentConfig()

// 生产环境
cfg := cache.ProductionConfig()

// 测试环境
cfg := cache.TestConfig()

// 高性能场景
cfg := cache.HighPerformanceConfig()
```

### 锁配置

```go
// 快速锁（短过期时间）
lockCfg := cache.QuickLockConfig()

// 长时间锁
lockCfg := cache.LongLockConfig()
```

### 布隆过滤器配置

```go
// 小容量
bloomCfg := cache.SmallBloomConfig()

// 大容量
bloomCfg := cache.LargeBloomConfig()

// 高精度
bloomCfg := cache.HighPrecisionBloomConfig()
```

## 最佳实践

### 1. 缓存自定义缓存实例

```go
// ✅ 推荐：缓存自定义缓存实例
var userCache Cache

func init() {
    cfg := cache.NewConfigBuilder().KeyPrefix("user").Build()
    userCache, _ = cache.New(cfg)
}

func GetUser(id string) (*User, error) {
    return userCache.Get(ctx, id)
}

// ❌ 避免：重复创建缓存实例
func GetUser(id string) (*User, error) {
    cfg := cache.NewConfigBuilder().KeyPrefix("user").Build()
    userCache, _ := cache.New(cfg) // 有性能开销
    return userCache.Get(ctx, id)
}
```

### 2. 错误处理

```go
value, err := cache.Get(ctx, "key")
if err != nil {
    // 检查是否是键不存在错误
    if cache.IsKeyNotFoundError(err) {
        return defaultValue, nil
    }
    // 其他错误
    return "", fmt.Errorf("cache error: %w", err)
}
```

### 3. 上下文使用

```go
// ✅ 使用带超时的上下文
ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
defer cancel()

value, err := cache.Get(ctx, "key")
```

### 4. 批量操作

```go
// ✅ 使用批量操作
cache.Del(ctx, "key1", "key2", "key3")
count, _ := cache.Exists(ctx, "key1", "key2", "key3")

// ❌ 避免单个操作
cache.Del(ctx, "key1")
cache.Del(ctx, "key2")
cache.Del(ctx, "key3")
```

## 日志集成

cache 与 clog 深度集成，自动记录：

- 操作成功/失败日志
- 慢操作警告（>100ms）
- 连接状态变化
- 锁获取/释放事件
- 详细的性能指标

## 性能优化

1. **缓存模块缓存器**：避免重复调用 `Module()`
2. **使用批量操作**：`Del()`, `Exists()` 等支持多个键
3. **合理设置连接池**：根据并发量调整 `PoolSize`
4. **启用压缩**：对于大值启用压缩减少网络传输
5. **使用键前缀**：避免键名冲突，便于管理

## 监控和调试

启用指标收集：

```go
cfg := cache.NewConfigBuilder().
    EnableMetrics().
    EnableTracing().
    Build()
```

查看日志输出了解操作详情和性能指标。

## 故障排除

### 连接问题

1. 检查 Redis 服务器是否运行
2. 验证连接参数（地址、端口、密码）
3. 检查网络连通性
4. 查看日志中的错误信息

### 性能问题

1. 监控慢操作日志
2. 调整连接池大小
3. 检查网络延迟
4. 考虑启用压缩

### 锁问题

1. 检查锁的过期时间设置
2. 确保正确释放锁
3. 处理锁竞争情况
4. 考虑使用重试机制

## 示例代码

查看 `examples/` 目录获取更多示例：

- `examples/basic/main.go` - 基础功能演示
- `examples/advanced/main.go` - 高级功能演示

## 测试

运行测试：

```bash
cd im-infra/cache
go test -v .
```

注意：测试需要 Redis 服务器运行在 localhost:6379
