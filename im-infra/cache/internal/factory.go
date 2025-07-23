package internal

import (
	"fmt"

	"github.com/ceyewan/gochat/im-infra/clog"
	"github.com/redis/go-redis/v9"
)

var (
	// 模块日志器
	logger = clog.Module("cache")
)

// NewCache 根据提供的配置创建一个新的 Cache 实例。
// 这是核心工厂函数，按配置组装所有组件。
func NewCache(cfg Config) (Cache, error) {
	// 验证配置
	if err := cfg.Validate(); err != nil {
		return nil, fmt.Errorf("invalid config: %w", err)
	}

	logger.Info("创建缓存实例",
		clog.String("addr", cfg.Addr),
		clog.Int("db", cfg.DB),
		clog.Int("poolSize", cfg.PoolSize),
		clog.String("serializer", cfg.Serializer),
	)

	// 创建 Redis 客户端选项
	opts := &redis.Options{
		Addr:            cfg.Addr,
		Password:        cfg.Password,
		DB:              cfg.DB,
		PoolSize:        cfg.PoolSize,
		MinIdleConns:    cfg.MinIdleConns,
		MaxIdleConns:    cfg.MaxIdleConns,
		ConnMaxIdleTime: cfg.ConnMaxIdleTime,
		ConnMaxLifetime: cfg.ConnMaxLifetime,
		DialTimeout:     cfg.DialTimeout,
		ReadTimeout:     cfg.ReadTimeout,
		WriteTimeout:    cfg.WriteTimeout,
		PoolTimeout:     cfg.PoolTimeout,
		MaxRetries:      cfg.MaxRetries,
		MinRetryBackoff: cfg.MinRetryBackoff,
		MaxRetryBackoff: cfg.MaxRetryBackoff,
	}

	// 创建 Redis 客户端
	rdb := redis.NewClient(opts)

	// 创建缓存实例
	cache := &cache{
		client:      rdb,
		config:      cfg,
		lockConfig:  DefaultLockConfig(),
		bloomConfig: DefaultBloomConfig(),
		logger:      logger,
	}

	logger.Info("缓存实例创建成功")
	return cache, nil
}

// NewDefaultCache 创建一个默认配置的缓存实例。
// 用于公共 API 的 Default() 方法。
func NewDefaultCache() Cache {
	cfg := DefaultConfig()
	cache, err := NewCache(cfg)
	if err != nil {
		// 这不应该发生，但如果发生了，我们将记录错误并返回 nil
		logger.Error("创建默认缓存实例失败", clog.Err(err))
		return nil
	}
	return cache
}

// NewCacheWithOptions 使用自定义选项创建缓存实例
func NewCacheWithOptions(opts *redis.Options) Cache {
	if opts == nil {
		return NewDefaultCache()
	}

	logger.Info("使用自定义选项创建缓存实例",
		clog.String("addr", opts.Addr),
		clog.Int("db", opts.DB),
		clog.Int("poolSize", opts.PoolSize),
	)

	rdb := redis.NewClient(opts)

	cache := &cache{
		client:      rdb,
		config:      DefaultConfig(),
		lockConfig:  DefaultLockConfig(),
		bloomConfig: DefaultBloomConfig(),
		logger:      logger,
	}

	logger.Info("缓存实例创建成功")
	return cache
}

// NewCacheWithClient 使用现有的 Redis 客户端创建缓存实例
func NewCacheWithClient(client *redis.Client) Cache {
	if client == nil {
		logger.Error("Redis 客户端不能为空")
		return nil
	}

	logger.Info("使用现有客户端创建缓存实例")

	cache := &cache{
		client:      client,
		config:      DefaultConfig(),
		lockConfig:  DefaultLockConfig(),
		bloomConfig: DefaultBloomConfig(),
		logger:      logger,
	}

	logger.Info("缓存实例创建成功")
	return cache
}

// ValidateConfig 验证配置的完整性和合理性
func ValidateConfig(cfg Config) error {
	if cfg.Addr == "" {
		return fmt.Errorf("Redis 地址不能为空")
	}

	if cfg.PoolSize <= 0 {
		return fmt.Errorf("连接池大小必须大于 0")
	}

	if cfg.MinIdleConns < 0 {
		return fmt.Errorf("最小空闲连接数不能小于 0")
	}

	if cfg.MaxIdleConns < cfg.MinIdleConns {
		return fmt.Errorf("最大空闲连接数不能小于最小空闲连接数")
	}

	if cfg.DialTimeout <= 0 {
		return fmt.Errorf("连接超时时间必须大于 0")
	}

	if cfg.ReadTimeout <= 0 {
		return fmt.Errorf("读取超时时间必须大于 0")
	}

	if cfg.WriteTimeout <= 0 {
		return fmt.Errorf("写入超时时间必须大于 0")
	}

	if cfg.MaxRetries < 0 {
		return fmt.Errorf("最大重试次数不能小于 0")
	}

	validSerializers := map[string]bool{
		"json":    true,
		"msgpack": true,
		"gob":     true,
	}

	if !validSerializers[cfg.Serializer] {
		return fmt.Errorf("不支持的序列化器: %s", cfg.Serializer)
	}

	return nil
}

// CreateRedisOptions 从配置创建 Redis 选项
func CreateRedisOptions(cfg Config) *redis.Options {
	return &redis.Options{
		Addr:            cfg.Addr,
		Password:        cfg.Password,
		DB:              cfg.DB,
		PoolSize:        cfg.PoolSize,
		MinIdleConns:    cfg.MinIdleConns,
		MaxIdleConns:    cfg.MaxIdleConns,
		ConnMaxIdleTime: cfg.ConnMaxIdleTime,
		ConnMaxLifetime: cfg.ConnMaxLifetime,
		DialTimeout:     cfg.DialTimeout,
		ReadTimeout:     cfg.ReadTimeout,
		WriteTimeout:    cfg.WriteTimeout,
		PoolTimeout:     cfg.PoolTimeout,
		MaxRetries:      cfg.MaxRetries,
		MinRetryBackoff: cfg.MinRetryBackoff,
		MaxRetryBackoff: cfg.MaxRetryBackoff,
	}
}
