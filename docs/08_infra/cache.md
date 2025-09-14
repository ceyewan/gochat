# 基础设施: Cache 分布式缓存

## 1. 设计理念

`cache` 是 `gochat` 项目的统一分布式缓存组件，默认基于 `Redis` 实现。它旨在提供一个高性能、功能丰富且类型安全的缓存层。

`cache` 组件的设计哲学是 **"封装但不隐藏"**。它将 `go-redis` 的底层复杂性封装在一个简洁、统一的 `Provider` 接口之后，同时通过组合多个小接口（如 `StringOperations`, `HashOperations`）来清晰地组织其丰富的功能。此外，它还内置了分布式锁和布隆过滤器等高级功能，为上层业务提供了强大的支持。

## 2. 核心 API 契约

### 2.1 构造函数

```go
// Config 是 cache 组件的配置结构体。
type Config struct {
    // Addr 是 Redis 服务器地址，格式为 "host:port"
    Addr            string        `json:"addr"`
    // Password 是 Redis 认证密码，可选
    Password        string        `json:"password"`
    // DB 是 Redis 数据库编号，默认为 0
    DB              int           `json:"db"`
    // PoolSize 是连接池大小，0 表示使用默认值 (CPU核数 * 10)
    PoolSize        int           `json:"poolSize"`
    // DialTimeout 是连接超时时间
    DialTimeout     time.Duration `json:"dialTimeout"`
    // ReadTimeout 是读取超时时间
    ReadTimeout     time.Duration `json:"readTimeout"`
    // WriteTimeout 是写入超时时间
    WriteTimeout    time.Duration `json:"writeTimeout"`
    // KeyPrefix 为所有 key 自动添加前缀，用于命名空间隔离，强烈推荐设置
    KeyPrefix       string        `json:"keyPrefix"`
    // MinIdleConns 是连接池中的最小空闲连接数
    MinIdleConns    int           `json:"minIdleConns"`
    // MaxRetries 是命令执行的最大重试次数
    MaxRetries      int           `json:"maxRetries"`
}

// GetDefaultConfig 返回默认的 cache 配置。
// 开发环境：较少的连接数，较短的超时时间，无密码认证
// 生产环境：较多的连接数，较长的超时时间，启用重试机制
func GetDefaultConfig(env string) *Config

type Option func(*options)

// WithLogger 将一个 clog.Logger 实例注入 cache，用于记录内部日志。
func WithLogger(logger clog.Logger) Option

// New 创建一个新的 cache Provider 实例。
// 这是与 cache 组件交互的唯一入口。
func New(ctx context.Context, config *Config, opts ...Option) (Provider, error)
```

### 2.2 Provider 接口

`Provider` 接口是所有缓存操作的总入口，它通过方法将不同的 Redis 数据结构操作分离开。

```go
// ErrCacheMiss 表示在缓存中未找到指定的 key。
// 所有 Get 操作在缓存未命中时，都应返回此错误。
var ErrCacheMiss = errors.New("cache: key not found")

// Provider 定义了 cache 组件提供的所有能力。
type Provider interface {
    String() StringOperations
    Hash() HashOperations
    Set() SetOperations
    ZSet() ZSetOperations
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
    // Get 获取一个 key。如果 key 不存在，将返回 cache.ErrCacheMiss 错误。
    Get(ctx context.Context, key string) (string, error)
    // Set 存入一个 key-value 对。
    // 注意：value (interface{}) 参数需要调用者自行序列化为字符串或字节数组。
    Set(ctx context.Context, key string, value interface{}, expiration time.Duration) error
    Del(ctx context.Context, keys ...string) error
    Incr(ctx context.Context, key string) (int64, error)
    Decr(ctx context.Context, key string) (int64, error)
    Exists(ctx context.Context, keys ...string) (int64, error)
    // SetNX (Set if Not Exists) 存入一个 key-value 对，仅当 key 不存在时。
    // 注意：value (interface{}) 参数需要调用者自行序列化。
    SetNX(ctx context.Context, key string, value interface{}, expiration time.Duration) (bool, error)
    // GetSet 设置新值并返回旧值。如果 key 不存在，返回 cache.ErrCacheMiss。
    // 注意：value (interface{}) 参数需要调用者自行序列化。
    GetSet(ctx context.Context, key string, value interface{}) (string, error)
}

// HashOperations 定义了所有与 Redis 哈希相关的操作。
type HashOperations interface {
    // HGet 获取哈希表 key 中一个 field 的值。如果 key 或 field 不存在，返回 cache.ErrCacheMiss。
    HGet(ctx context.Context, key, field string) (string, error)
    // HSet 设置哈希表 key 中一个 field 的值。
    // 注意：value (interface{}) 参数需要调用者自行序列化。
    HSet(ctx context.Context, key, field string, value interface{}) error
    HGetAll(ctx context.Context, key string) (map[string]string, error)
    HDel(ctx context.Context, key string, fields ...string) error
    HExists(ctx context.Context, key, field string) (bool, error)
    HLen(ctx context.Context, key string) (int64, error)
}

// SetOperations 定义了所有与 Redis 集合相关的操作。
type SetOperations interface {
    SAdd(ctx context.Context, key string, members ...interface{}) error
    SRem(ctx context.Context, key string, members ...interface{}) error
    SMembers(ctx context.Context, key string) ([]string, error)
    SIsMember(ctx context.Context, key string, member interface{}) (bool, error)
    SCard(ctx context.Context, key string) (int64, error)
}

// ZMember 表示有序集合中的成员
type ZMember struct {
    Member interface{} // 成员值
    Score  float64     // 分数
}

// ZSetOperations 定义了所有与 Redis 有序集合相关的操作。
type ZSetOperations interface {
    // ZAdd 添加一个或多个成员到有序集合
    ZAdd(ctx context.Context, key string, members ...*ZMember) error
    // ZRange 获取有序集合中指定范围内的成员，按分数从低到高排序
    ZRange(ctx context.Context, key string, start, stop int64) ([]*ZMember, error)
    // ZRevRange 获取有序集合中指定范围内的成员，按分数从高到低排序
    ZRevRange(ctx context.Context, key string, start, stop int64) ([]*ZMember, error)
    // ZRangeByScore 获取指定分数范围内的成员
    ZRangeByScore(ctx context.Context, key string, min, max float64) ([]*ZMember, error)
    // ZRem 从有序集合中移除一个或多个成员
    ZRem(ctx context.Context, key string, members ...interface{}) error
    // ZRemRangeByRank 移除有序集合中指定排名区间内的成员
    ZRemRangeByRank(ctx context.Context, key string, start, stop int64) error
    // ZCard 获取有序集合的成员数量
    ZCard(ctx context.Context, key string) (int64, error)
    // ZCount 获取指定分数范围内的成员数量
    ZCount(ctx context.Context, key string, min, max float64) (int64, error)
    // ZScore 获取成员的分数
    ZScore(ctx context.Context, key string, member string) (float64, error)
    // ZSetExpire 为有序集合设置过期时间
    ZSetExpire(ctx context.Context, key string, expiration time.Duration) error
}

// LockOperations 定义了分布式锁的操作。
type LockOperations interface {
    // Acquire 尝试获取一个锁。如果成功，返回一个 Locker 对象；否则返回错误。
    Acquire(ctx context.Context, key string, expiration time.Duration) (Locker, error)
}

// Locker 定义了锁对象的接口。
type Locker interface {
    // Unlock 释放锁
    Unlock(ctx context.Context) error
    // Refresh 刷新锁的过期时间
    Refresh(ctx context.Context, expiration time.Duration) error
}

// BloomFilterOperations 定义了布隆过滤器的操作 (需要 RedisBloom 模块)。
type BloomFilterOperations interface {
    BFAdd(ctx context.Context, key string, item string) error
    BFExists(ctx context.Context, key string, item string) (bool, error)
    BFReserve(ctx context.Context, key string, errorRate float64, capacity uint64) error
}

// ScriptingOperations 定义了与 Redis Lua 脚本相关的操作。
type ScriptingOperations interface {
    EvalSha(ctx context.Context, sha1 string, keys []string, args ...interface{}) (interface{}, error)
    ScriptLoad(ctx context.Context, script string) (string, error)
    ScriptExists(ctx context.Context, sha1 ...string) ([]bool, error)
}
```

## 3. 标准用法

### 场景 1: 在服务启动时初始化 cache Provider

```go
// 在 main.go 中
func main() {
    // ... 首先初始化 clog 和 coord ...
    
    // 使用默认配置（推荐）
    config := cache.GetDefaultConfig("production") // 或 "development"
    
    // 根据实际部署环境覆盖特定配置
    config.Addr = "redis-cluster:6379"
    config.Password = "your-redis-password"
    config.KeyPrefix = "gochat:"
    
    // 创建 cache Provider
    cacheProvider, err := cache.New(context.Background(), config, 
        cache.WithLogger(clog.Namespace("cache")),
    )
    if err != nil {
        clog.Fatal("初始化 cache 失败", clog.Err(err))
    }
    defer cacheProvider.Close()
    
    // 验证连接
    if err := cacheProvider.Ping(context.Background()); err != nil {
        clog.Fatal("Redis 连接失败", clog.Err(err))
    }
    
    clog.Info("cache Provider 初始化成功")
}
```

### 场景 2: 缓存用户信息

```go
// 在服务的构造函数中注入 cacheProvider
type UserService struct {
    cache cache.Provider
    db    db.Provider
}

func NewUserService(cacheProvider cache.Provider, dbProvider db.Provider) *UserService {
    return &UserService{
        cache: cacheProvider,
        db:    dbProvider,
    }
}

// 在业务方法中使用
func (s *UserService) GetUserProfile(ctx context.Context, userID string) (*Profile, error) {
    logger := clog.WithContext(ctx)
    key := "user:" + userID + ":profile"
    
    // 1. 尝试从缓存获取
    profileJSON, err := s.cache.String().Get(ctx, key)
    if err == nil {
        // 缓存命中
        var profile Profile
        if json.Unmarshal([]byte(profileJSON), &profile) == nil {
            logger.Info("用户资料缓存命中", clog.String("user_id", userID))
            return &profile, nil
        }
    }
    
    logger.Info("用户资料缓存未命中，从数据库查询", clog.String("user_id", userID))
    
    // 2. 缓存未命中，从数据库获取
    profile, err := s.getUserFromDB(ctx, userID)
    if err != nil {
        return nil, fmt.Errorf("从数据库获取用户资料失败: %w", err)
    }
    
    // 3. 将结果写入缓存，设置 1 小时过期
    profileJSON, _ := json.Marshal(profile)
    if err := s.cache.String().Set(ctx, key, profileJSON, 1*time.Hour); err != nil {
        logger.Error("写入用户资料缓存失败", clog.Err(err))
        // 不影响业务流程，继续返回结果
    }
    
    return profile, nil
}

func (s *UserService) getUserFromDB(ctx context.Context, userID string) (*Profile, error) {
    var profile Profile
    err := s.db.DB(ctx).Where("id = ?", userID).First(&profile).Error
    if err != nil {
        return nil, err
    }
    return &profile, nil
}
```

### 场景 3: 使用分布式锁执行定时任务

```go
func (s *ReportService) GenerateDailyReport(ctx context.Context) error {
    logger := clog.WithContext(ctx)
    
    // 尝试获取一个租期为 10 分钟的锁
    lock, err := s.cache.Lock().Acquire(ctx, "lock:daily_report_job", 10*time.Minute)
    if err != nil {
        // 获取锁失败，说明已有其他实例在执行，当前实例直接退出
        logger.Info("获取报表生成锁失败，任务已由其他实例执行")
        return nil
    }
    
    // 确保任务完成后释放锁
    defer func() {
        if err := lock.Unlock(context.Background()); err != nil {
            logger.Error("释放报表生成锁失败", clog.Err(err))
        }
    }()

    logger.Info("成功获取锁，开始生成日报表")
    
    // 执行报表生成逻辑
    if err := s.generateReport(ctx); err != nil {
        return fmt.Errorf("生成报表失败: %w", err)
    }
    
    logger.Info("日报表生成完成")
    return nil
}
```

### 场景 4: 使用 ZSET 管理会话消息记录

```go
// SessionMessageService 管理会话的最近消息记录
type SessionMessageService struct {
    cache cache.Provider
}

func NewSessionMessageService(cacheProvider cache.Provider) *SessionMessageService {
    return &SessionMessageService{
        cache: cacheProvider,
    }
}

// AddMessage 添加消息到会话，使用时间戳排序
func (s *SessionMessageService) AddMessage(ctx context.Context, sessionID, messageID string) error {
    logger := clog.WithContext(ctx)
    key := "session:" + sessionID + ":messages"

    // 使用时间戳作为分数
    message := &cache.ZMember{
        Member: messageID,
        Score:  float64(time.Now().Unix()),
    }

    // 添加消息到 ZSET
    if err := s.cache.ZSet().ZAdd(ctx, key, message); err != nil {
        return fmt.Errorf("添加消息失败: %w", err)
    }

    // 维护最近50条消息限制
    count, err := s.cache.ZSet().ZCard(ctx, key)
    if err != nil {
        return fmt.Errorf("获取消息数量失败: %w", err)
    }

    // 如果超过50条，移除多余的旧消息
    if count > 50 {
        err := s.cache.ZSet().ZRemRangeByRank(ctx, key, 0, count-51)
        if err != nil {
            logger.Error("移除旧消息失败", clog.Err(err))
            // 不影响主流程
        }
    }

    // 设置过期时间，防止活跃会话内存占用过大
    err = s.cache.ZSet().ZSetExpire(ctx, key, 2*time.Hour)
    if err != nil {
        logger.Error("设置会话过期时间失败", clog.Err(err))
    }

    return nil
}

// GetRecentMessages 获取会话最近的消息记录
func (s *SessionMessageService) GetRecentMessages(ctx context.Context, sessionID string, limit int) ([]string, error) {
    key := "session:" + sessionID + ":messages"

    // 获取最新的N条消息
    members, err := s.cache.ZSet().ZRevRange(ctx, key, 0, int64(limit-1))
    if err != nil {
        return nil, fmt.Errorf("获取最近消息失败: %w", err)
    }

    // 提取消息ID
    var messageIDs []string
    for _, member := range members {
        if id, ok := member.Member.(string); ok {
            messageIDs = append(messageIDs, id)
        }
    }

    return messageIDs, nil
}

// GetMessagesByTimeRange 获取指定时间范围内的消息
func (s *SessionMessageService) GetMessagesByTimeRange(ctx context.Context, sessionID string, startTime, endTime time.Time) ([]string, error) {
    key := "session:" + sessionID + ":messages"

    // 使用时间戳范围查询
    minScore := float64(startTime.Unix())
    maxScore := float64(endTime.Unix())

    members, err := s.cache.ZSet().ZRangeByScore(ctx, key, minScore, maxScore)
    if err != nil {
        return nil, fmt.Errorf("按时间范围获取消息失败: %w", err)
    }

    // 提取消息ID并按时间排序
    var messageIDs []string
    for _, member := range members {
        if id, ok := member.Member.(string); ok {
            messageIDs = append(messageIDs, id)
        }
    }

    return messageIDs, nil
}
```

### 场景 5: 使用布隆过滤器进行重复检测

```go
func (s *MessageService) CheckDuplicateMessage(ctx context.Context, messageID string) (bool, error) {
    logger := clog.WithContext(ctx)
    key := "bloom:duplicate_messages"

    // 检查消息是否已存在
    exists, err := s.cache.Bloom().BFExists(ctx, key, messageID)
    if err != nil {
        return false, fmt.Errorf("检查布隆过滤器失败: %w", err)
    }

    if exists {
        logger.Warn("检测到重复消息", clog.String("message_id", messageID))
        return true, nil
    }

    // 添加到布隆过滤器
    if err := s.cache.Bloom().BFAdd(ctx, key, messageID); err != nil {
        logger.Error("添加到布隆过滤器失败", clog.Err(err))
        // 不影响业务流程
    }

    return false, nil
}
```

## 4. 设计注记

### 4.1 GetDefaultConfig 默认值说明

`GetDefaultConfig` 根据环境返回优化的默认配置：

**开发环境 (development)**:
```go
&Config{
    Addr:            "localhost:6379",
    Password:        "",                    // 无密码
    DB:              0,                     // 默认数据库
    PoolSize:        10,                    // 较少连接数
    DialTimeout:     5 * time.Second,      // 较短连接超时
    ReadTimeout:     3 * time.Second,      // 较短读取超时
    WriteTimeout:    3 * time.Second,      // 较短写入超时
    KeyPrefix:       "dev:",               // 开发环境前缀
    MinIdleConns:    2,                    // 较少空闲连接
    MaxRetries:      1,                    // 较少重试次数
}
```

**生产环境 (production)**:
```go
&Config{
    Addr:            "redis:6379",
    Password:        "",                    // 需要用户覆盖
    DB:              0,
    PoolSize:        100,                   // 较多连接数
    DialTimeout:     10 * time.Second,     // 较长连接超时
    ReadTimeout:     5 * time.Second,      // 较长读取超时
    WriteTimeout:    5 * time.Second,      // 较长写入超时
    KeyPrefix:       "gochat:",            // 生产环境前缀
    MinIdleConns:    10,                   // 较多空闲连接
    MaxRetries:      3,                    // 较多重试次数
}
```

用户仍需要根据实际部署环境覆盖 `Addr`、`Password` 等关键配置。

### 4.2 KeyPrefix 命名空间隔离

`KeyPrefix` 是一个重要的配置项，用于实现多环境、多服务的 key 命名空间隔离：

- **多环境隔离**: `dev:`、`test:`、`prod:` 等前缀区分不同环境
- **多服务隔离**: `gochat:user:`、`gochat:message:` 等前缀区分不同微服务
- **自动拼接**: 组件会自动将 `KeyPrefix` 添加到所有 key 的前面

### 4.3 分布式锁的实现机制

`LockOperations` 基于 Redis 的 `SET key value EX seconds NX` 命令实现：

- **原子性保证**: 使用 Redis 的原子命令确保只有一个客户端能获取锁
- **自动过期**: 通过 `expiration` 参数防止死锁
- **唯一标识**: 每个锁都有唯一的 value 值，确保只有持锁者能释放
- **安全释放**: 使用 Lua 脚本确保释放锁的原子性

### 4.4 布隆过滤器支持

`BloomFilterOperations` 需要 Redis 服务器安装 `RedisBloom` 模块：

- **内存高效**: 适用于大规模数据的去重检测
- **误判特性**: 可能有假阳性，但没有假阴性
- **预配置**: 通过 `BFReserve` 预先配置误判率和容量
- **持久化**: 数据持久化到 Redis，支持集群部署

### 4.6 ZSET 有序集合的设计与使用

**时间戳排序策略**: ZSET 操作专门为会话消息管理优化，使用 Unix 时间戳作为分数，实现消息的时间排序。

**内存管理机制**: 通过 `ZRemRangeByRank` 和 `ZSetExpire` 实现消息数量控制和内存占用管理，防止活跃会话无限增长。

**会话生命周期**: 每个会话使用独立的 ZSET key，格式为 `session:{sessionID}:messages`，便于管理和清理。

**性能优化**:
- 使用 `ZRevRange` 获取最新消息，避免全量数据传输
- 通过 `ZRangeByScore` 实现时间范围查询，支持历史消息回溯
- 定期清理过期会话，避免内存碎片

**典型应用场景**:
1. **会话消息管理**: 维护每个会话最近50条消息记录
2. **排行榜系统**: 实现实时分数排序和排名查询
3. **时间序列数据**: 按时间顺序存储和查询事件记录
4. **延迟队列**: 使用时间戳实现定时任务调度

### 4.7 错误处理最佳实践

**缓存穿透保护**: 对于不存在的数据，也应设置短期缓存（如空字符串或特定的 "null" 值），以防止恶意攻击。
**缓存击穿保护**: 在高并发场景下，如果一个热点 key 失效，可能导致大量请求同时访问数据库。建议的解决方案是在缓存回源逻辑中，使用 `cache.Lock().Acquire()` 获取分布式锁，确保只有一个请求去加载数据库，其他请求等待结果。
**降级机制**: 当 Redis 不可用时，应优雅降级到直接访问数据库，并记录错误。
**日志记录**: 缓存操作失败应记录日志，但不影响主业务流程。
**超时控制**: 通过 context 控制操作超时，避免长时间阻塞。
