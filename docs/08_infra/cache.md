# 基础设施: Cache 分布式缓存

## 1. 设计理念

`cache` 是 `gochat` 项目的统一分布式缓存组件，默认基于 `Redis` 实现。它旨在提供一个高性能、功能丰富且类型安全的缓存层。

`cache` 组件的设计哲学是 **“封装但不隐藏”**。它将 `go-redis` 的底层复杂性封装在一个简洁、统一的 `Provider` 接口之后，同时通过组合多个小接口（如 `StringOperations`, `HashOperations`）来清晰地组织其丰富的功能。此外，它还内置了分布式锁和布隆过滤器等高级功能，为上层业务提供了强大的支持。

## 2. 核心 API 契约

### 2.1 构造函数

```go
// Config 是 cache 组件的配置结构体。
type Config struct {
	Addr            string        `json:"addr"`
	Password        string        `json:"password"`
	DB              int           `json:"db"`
	PoolSize        int           `json:"poolSize"`
	DialTimeout     time.Duration `json:"dialTimeout"`
	ReadTimeout     time.Duration `json:"readTimeout"`
	WriteTimeout    time.Duration `json:"writeTimeout"`
	// KeyPrefix 为所有 key 自动添加前缀，用于命名空间隔离，强烈推荐设置。
	KeyPrefix       string        `json:"keyPrefix"`
}

// New 创建一个新的 cache Provider 实例。
// 这是与 cache 组件交互的唯一入口。
func New(ctx context.Context, config *Config, opts ...Option) (Provider, error)
```

### 2.2 Provider 接口

`Provider` 接口是所有缓存操作的总入口，它通过方法将不同的 Redis 数据结构操作分离开。

```go
// Provider 定义了 cache 组件提供的所有能力。
type Provider interface {
	String() StringOperations
	Hash() HashOperations
	Set() SetOperations
	Lock() LockOperations
	Bloom() BloomFilterOperations
	Script() ScriptingOperations

	// Ping 检查与 Redis 服务器的连接。
	Ping(ctx context.Context) error
	// Close 关闭所有与 Redis 的连接。
	Close() error
}
```

### 2.3 功能子接口

`Provider` 组合了多个功能单一的子接口，使得 API 非常清晰。

```go
// StringOperations 定义了所有与 Redis 字符串相关的操作。
type StringOperations interface {
	Get(ctx context.Context, key string) (string, error)
	Set(ctx context.Context, key string, value interface{}, expiration time.Duration) error
	Del(ctx context.Context, keys ...string) error
	Incr(ctx context.Context, key string) (int64, error)
	// ...
}

// HashOperations 定义了所有与 Redis 哈希相关的操作。
type HashOperations interface {
	HGet(ctx context.Context, key, field string) (string, error)
	HSet(ctx context.Context, key, field string, value interface{}) error
	HGetAll(ctx context.Context, key string) (map[string]string, error)
	// ...
}

// LockOperations 定义了分布式锁的操作。
type LockOperations interface {
	// Acquire 尝试获取一个锁。如果成功，返回一个 Locker 对象；否则返回错误。
	Acquire(ctx context.Context, key string, expiration time.Duration) (Locker, error)
}

// Locker 定义了锁对象的接口。
type Locker interface {
    Unlock(ctx context.Context) error
    Refresh(ctx context.Context, expiration time.Duration) error
}

// BloomFilterOperations 定义了布隆过滤器的操作 (需要 RedisBloom 模块)。
type BloomFilterOperations interface {
	BFAdd(ctx context.Context, key string, item string) error
	BFExists(ctx context.Context, key string, item string) (bool, error)
}

// ScriptingOperations 定义了与 Redis Lua 脚本相关的操作。
type ScriptingOperations interface {
	EvalSha(ctx context.Context, sha1 string, keys []string, args ...interface{}) (interface{}, error)
	ScriptLoad(ctx context.Context, script string) (string, error)
}
```

## 3. 标准用法

### 场景 1: 缓存用户信息

```go
// 在服务的构造函数中初始化 cacheProvider
func NewUserService(cacheProvider cache.Provider) *UserService {
    return &UserService{cache: cacheProvider}
}

// 在业务方法中使用
func (s *UserService) GetUserProfile(ctx context.Context, userID string) (*Profile, error) {
    key := "user:" + userID + ":profile"
    
    // 1. 尝试从缓存获取
    profileJSON, err := s.cache.String().Get(ctx, key)
    if err == nil {
        // 缓存命中
        var profile Profile
        if json.Unmarshal([]byte(profileJSON), &profile) == nil {
            return &profile, nil
        }
    }
    
    // 2. 缓存未命中，从数据库获取
    profile, err := s.db.GetUserProfile(ctx, userID)
    if err != nil {
        return nil, err
    }
    
    // 3. 将结果写入缓存，设置 1 小时过期
    profileJSON, _ = json.Marshal(profile)
    s.cache.String().Set(ctx, key, profileJSON, 1*time.Hour)
    
    return profile, nil
}
```

### 场景 2: 使用分布式锁执行定时任务

```go
func (s *ReportService) GenerateDailyReport(ctx context.Context) {
    // 尝试获取一个租期为 10 分钟的锁
    lock, err := s.cache.Lock().Acquire(ctx, "lock:daily_report_job", 10*time.Minute)
    if err != nil {
        // 获取锁失败，说明已有其他实例在执行，当前实例直接退出。
        s.logger.Info("获取报表生成锁失败，任务已由其他实例执行。")
        return
    }
    // 确保任务完成后释放锁
    defer lock.Unlock(ctx)

    s.logger.Info("成功获取锁，开始生成日报表...")
    // --- 执行报表生成逻辑 ---
}