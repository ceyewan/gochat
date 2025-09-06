package cache

import (
	"context"
	"time"

	"github.com/ceyewan/gochat/im-infra/cache/internal"
	"github.com/ceyewan/gochat/im-infra/clog"
)

// New 根据提供的配置创建一个新的 Cache 实例。
// 这是核心工厂函数，按配置组装所有组件。
//
// 参数：
//   - ctx: 上下文，用于超时控制和取消操作
//   - cfg: Redis 配置
//   - opts: 可选配置项，支持注入 Logger 等依赖
//
// 示例：
//
//	cfg := cache.DefaultConfig()
//	logger := clog.Module("my-cache")
//	cache, err := cache.New(ctx, cfg, cache.WithLogger(logger))
//	if err != nil {
//		log.Fatal(err)
//	}
func New(ctx context.Context, cfg Config, opts ...Option) (Cache, error) {
	// 应用选项
	options := &Options{}
	for _, opt := range opts {
		opt(options)
	}

	// 设置默认 Logger
	var componentLogger clog.Logger
	if options.Logger != nil {
		if options.ComponentName != "" {
			componentLogger = options.Logger.With(clog.String("component", options.ComponentName))
		} else {
			componentLogger = options.Logger.With(clog.String("component", "cache"))
		}
	} else {
		if options.ComponentName != "" {
			componentLogger = clog.Module("cache").With(clog.String("name", options.ComponentName))
		} else {
			componentLogger = clog.Module("cache")
		}
	}

	componentLogger.Info("创建 cache 实例",
		clog.String("addr", cfg.Addr),
		clog.Int("db", cfg.DB),
		clog.Int("poolSize", cfg.PoolSize))

	// 将顶层 Config 转换为内部使用的 internal.Config
	internalCfg := internal.Config{
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
		KeyPrefix:       cfg.KeyPrefix,
	}

	// 创建 cache 实例
	return internal.NewCache(ctx, internalCfg, componentLogger)
}

// DefaultConfig 返回适用于开发和生产环境的默认配置
func DefaultConfig() Config {
	return Config{
		Addr:            "localhost:6379",
		Password:        "",
		DB:              0,
		PoolSize:        10,
		MinIdleConns:    5,
		MaxIdleConns:    10,
		ConnMaxIdleTime: 30 * time.Minute,
		ConnMaxLifetime: 1 * time.Hour,
		DialTimeout:     5 * time.Second,
		ReadTimeout:     3 * time.Second,
		WriteTimeout:    3 * time.Second,
		PoolTimeout:     4 * time.Second,
		MaxRetries:      3,
		MinRetryBackoff: 8 * time.Millisecond,
		MaxRetryBackoff: 512 * time.Millisecond,
		KeyPrefix:       "",
	}
}
