package cache

import (
	"context"
	"time"

	"github.com/ceyewan/gochat/im-infra/cache/internal"
	"github.com/ceyewan/gochat/im-infra/clog"
)

// providerWrapper 包装内部 client 实现 Provider 接口
type providerWrapper struct {
	client internal.Client
}

// Provider 接口实现
func (p *providerWrapper) String() StringOperations {
	return &stringOperationsWrapper{ops: p.client.String()}
}

func (p *providerWrapper) Hash() HashOperations {
	return &hashOperationsWrapper{ops: p.client.Hash()}
}

func (p *providerWrapper) Set() SetOperations {
	return p.client.Set()
}

func (p *providerWrapper) Lock() LockOperations {
	return &lockOperationsWrapper{ops: p.client.Lock()}
}

func (p *providerWrapper) Bloom() BloomFilterOperations {
	return &bloomOperationsWrapper{ops: p.client.Bloom()}
}

func (p *providerWrapper) Script() ScriptingOperations {
	return p.client.Script()
}

func (p *providerWrapper) Ping(ctx context.Context) error {
	return p.client.Ping(ctx)
}

func (p *providerWrapper) Close() error {
	return p.client.Close()
}

// stringOperationsWrapper 包装内部 StringOperations
type stringOperationsWrapper struct {
	ops internal.StringOperations
}

func (s *stringOperationsWrapper) Get(ctx context.Context, key string) (string, error) {
	return s.ops.Get(ctx, key)
}

func (s *stringOperationsWrapper) Set(ctx context.Context, key string, value interface{}, expiration time.Duration) error {
	return s.ops.Set(ctx, key, value, expiration)
}

func (s *stringOperationsWrapper) Del(ctx context.Context, keys ...string) error {
	return s.ops.Del(ctx, keys...)
}

func (s *stringOperationsWrapper) Incr(ctx context.Context, key string) (int64, error) {
	return s.ops.Incr(ctx, key)
}

func (s *stringOperationsWrapper) Decr(ctx context.Context, key string) (int64, error) {
	return s.ops.Decr(ctx, key)
}

func (s *stringOperationsWrapper) Exists(ctx context.Context, keys ...string) (int64, error) {
	return s.ops.Exists(ctx, keys...)
}

func (s *stringOperationsWrapper) SetNX(ctx context.Context, key string, value interface{}, expiration time.Duration) (bool, error) {
	return s.ops.SetNX(ctx, key, value, expiration)
}

func (s *stringOperationsWrapper) GetSet(ctx context.Context, key string, value interface{}) (string, error) {
	return s.ops.GetSet(ctx, key, value)
}

// hashOperationsWrapper 包装内部 HashOperations
type hashOperationsWrapper struct {
	ops internal.HashOperations
}

func (h *hashOperationsWrapper) HGet(ctx context.Context, key, field string) (string, error) {
	return h.ops.HGet(ctx, key, field)
}

func (h *hashOperationsWrapper) HSet(ctx context.Context, key, field string, value interface{}) error {
	return h.ops.HSet(ctx, key, field, value)
}

func (h *hashOperationsWrapper) HGetAll(ctx context.Context, key string) (map[string]string, error) {
	return h.ops.HGetAll(ctx, key)
}

func (h *hashOperationsWrapper) HDel(ctx context.Context, key string, fields ...string) error {
	return h.ops.HDel(ctx, key, fields...)
}

func (h *hashOperationsWrapper) HExists(ctx context.Context, key, field string) (bool, error) {
	return h.ops.HExists(ctx, key, field)
}

func (h *hashOperationsWrapper) HLen(ctx context.Context, key string) (int64, error) {
	return h.ops.HLen(ctx, key)
}

// lockOperationsWrapper 包装内部 LockOperations
type lockOperationsWrapper struct {
	ops internal.LockOperations
}

func (l *lockOperationsWrapper) Acquire(ctx context.Context, key string, expiration time.Duration) (Locker, error) {
	internalLocker, err := l.ops.Acquire(ctx, key, expiration)
	if err != nil {
		return nil, err
	}
	return &lockerWrapper{locker: internalLocker}, nil
}

// lockerWrapper 包装内部 Locker
type lockerWrapper struct {
	locker internal.Locker
}

func (l *lockerWrapper) Unlock(ctx context.Context) error {
	return l.locker.Unlock(ctx)
}

func (l *lockerWrapper) Refresh(ctx context.Context, expiration time.Duration) error {
	return l.locker.Refresh(ctx, expiration)
}

// bloomOperationsWrapper 包装内部 BloomFilterOperations
type bloomOperationsWrapper struct {
	ops internal.BloomFilterOperations
}

func (b *bloomOperationsWrapper) BFAdd(ctx context.Context, key string, item string) error {
	return b.ops.BFAdd(ctx, key, item)
}

func (b *bloomOperationsWrapper) BFExists(ctx context.Context, key string, item string) (bool, error) {
	return b.ops.BFExists(ctx, key, item)
}

func (b *bloomOperationsWrapper) BFReserve(ctx context.Context, key string, errorRate float64, capacity uint64) error {
	return b.ops.BFReserve(ctx, key, errorRate, capacity)
}

// New 创建一个新的 cache Provider 实例。
// 这是与 cache 组件交互的唯一入口。
func New(ctx context.Context, config *Config, opts ...Option) (Provider, error) {
	// 应用选项
	options := &options{}
	for _, opt := range opts {
		opt(options)
	}

	// 设置默认 Logger
	var componentLogger clog.Logger
	if options.logger != nil {
		componentLogger = options.logger
	} else {
		componentLogger = clog.Namespace("cache")
	}

	componentLogger.Info("创建 cache 实例",
		clog.String("addr", config.Addr),
		clog.Int("db", config.DB),
		clog.Int("poolSize", config.PoolSize))

	// 将顶层 Config 转换为内部使用的 internal.Config
	internalCfg := internal.Config{
		Addr:            config.Addr,
		Password:        config.Password,
		DB:              config.DB,
		PoolSize:        config.PoolSize,
		MinIdleConns:    config.MinIdleConns,
		MaxIdleConns:    config.MaxIdleConns,
		ConnMaxIdleTime: config.ConnMaxIdleTime,
		ConnMaxLifetime: config.ConnMaxLifetime,
		DialTimeout:     config.DialTimeout,
		ReadTimeout:     config.ReadTimeout,
		WriteTimeout:    config.WriteTimeout,
		PoolTimeout:     config.PoolTimeout,
		MaxRetries:      config.MaxRetries,
		MinRetryBackoff: config.MinRetryBackoff,
		MaxRetryBackoff: config.MaxRetryBackoff,
		KeyPrefix:       config.KeyPrefix,
	}

	// 创建 cache 实例
	internalClient, err := internal.NewCache(ctx, internalCfg, componentLogger)
	if err != nil {
		return nil, err
	}

	return &providerWrapper{
		client: internalClient,
	}, nil
}

// GetDefaultConfig 返回默认的 cache 配置。
// 开发环境：较少的连接数，较短的超时时间，无密码认证
// 生产环境：较多的连接数，较长的超时时间，启用重试机制
func GetDefaultConfig(env string) *Config {
	if env == "production" {
		return &Config{
			Addr:            "redis:6379",
			Password:        "",
			DB:              0,
			PoolSize:        100,
			MinIdleConns:    10,
			MaxIdleConns:    20,
			ConnMaxIdleTime: 30 * time.Minute,
			ConnMaxLifetime: 1 * time.Hour,
			DialTimeout:     10 * time.Second,
			ReadTimeout:     5 * time.Second,
			WriteTimeout:    5 * time.Second,
			PoolTimeout:     5 * time.Second,
			MaxRetries:      3,
			MinRetryBackoff: 8 * time.Millisecond,
			MaxRetryBackoff: 512 * time.Millisecond,
			KeyPrefix:       "gochat:",
		}
	}

	// development environment
	return &Config{
		Addr:            "localhost:6379",
		Password:        "",
		DB:              0,
		PoolSize:        10,
		MinIdleConns:    2,
		MaxIdleConns:    5,
		ConnMaxIdleTime: 30 * time.Minute,
		ConnMaxLifetime: 1 * time.Hour,
		DialTimeout:     5 * time.Second,
		ReadTimeout:     3 * time.Second,
		WriteTimeout:    3 * time.Second,
		PoolTimeout:     4 * time.Second,
		MaxRetries:      1,
		MinRetryBackoff: 8 * time.Millisecond,
		MaxRetryBackoff: 512 * time.Millisecond,
		KeyPrefix:       "dev:",
	}
}
