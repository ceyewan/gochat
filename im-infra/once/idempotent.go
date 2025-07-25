package idempotent

import (
	"context"
	"sync"
	"time"

	"github.com/ceyewan/gochat/im-infra/clog"
	"github.com/ceyewan/gochat/im-infra/once/internal"
)

// Idempotent 定义幂等操作的核心接口。
// 提供基于 Redis setnx 命令的幂等检查和设置功能。
type Idempotent = internal.Idempotent

// BatchIdempotent 定义批量幂等操作的接口
type BatchIdempotent = internal.BatchIdempotent

// ResultIdempotent 定义带结果存储的幂等操作接口
type ResultIdempotent = internal.ResultIdempotent

// AdvancedIdempotent 定义高级幂等操作的接口
type AdvancedIdempotent = internal.AdvancedIdempotent

// Config 是 idempotent 的主配置结构体。
// 用于声明式地定义幂等组件的行为和参数。
type Config = internal.Config

// RetryConfig 重试配置
type RetryConfig = internal.RetryConfig

// IdempotentResult 定义幂等操作的结果
type IdempotentResult = internal.IdempotentResult

// IdempotentStatus 定义幂等状态
type IdempotentStatus = internal.IdempotentStatus

var (
	// 全局默认幂等客户端实例
	defaultClient Idempotent
	// 确保默认客户端只初始化一次
	defaultClientOnce sync.Once
	// 模块日志器
	logger = clog.Module("idempotent")
)

// getDefaultClient 获取全局默认幂等客户端实例，使用懒加载和单例模式
func getDefaultClient() Idempotent {
	defaultClientOnce.Do(func() {
		cfg := DefaultConfig()
		var err error
		defaultClient, err = internal.NewIdempotentClient(cfg)
		if err != nil {
			logger.Error("创建默认幂等客户端失败", clog.Err(err))
			panic(err)
		}
		logger.Info("默认幂等客户端初始化成功")
	})
	return defaultClient
}

// New 根据提供的配置创建一个新的 Idempotent 实例。
// 用于自定义幂等客户端的主要入口。
//
// 示例：
//
//	cfg := idempotent.Config{
//	  KeyPrefix: "myapp",
//	  DefaultTTL: time.Hour,
//	  CacheConfig: cache.Config{
//	    Addr: "localhost:6379",
//	  },
//	}
//	client, err := idempotent.New(cfg)
//	if err != nil {
//	  log.Fatal(err)
//	}
//	success, _ := client.Set(ctx, "operation:123", time.Hour)
func New(cfg Config) (Idempotent, error) {
	return internal.NewIdempotentClient(cfg)
}

// DefaultConfig 返回一个带有合理默认值的 Config
// 默认配置适用于开发环境
func DefaultConfig() Config {
	return internal.DefaultConfig()
}

// DevelopmentConfig 返回适用于开发环境的配置
func DevelopmentConfig() Config {
	return internal.DevelopmentConfig()
}

// ProductionConfig 返回适用于生产环境的配置
func ProductionConfig() Config {
	return internal.ProductionConfig()
}

// TestConfig 返回适用于测试环境的配置
func TestConfig() Config {
	return internal.TestConfig()
}

// NewConfigBuilder 创建新的配置构建器
func NewConfigBuilder() *internal.ConfigBuilder {
	return internal.NewConfigBuilder()
}

// ===== 全局幂等方法 =====

// Check 使用全局默认客户端检查指定键是否已经存在（是否已执行过）
func Check(ctx context.Context, key string) (bool, error) {
	return getDefaultClient().Check(ctx, key)
}

// Set 使用全局默认客户端设置幂等标记，如果键已存在则返回 false
func Set(ctx context.Context, key string, ttl time.Duration) (bool, error) {
	return getDefaultClient().Set(ctx, key, ttl)
}

// CheckAndSet 使用全局默认客户端原子性地检查并设置幂等标记
func CheckAndSet(ctx context.Context, key string, ttl time.Duration) (bool, error) {
	return getDefaultClient().CheckAndSet(ctx, key, ttl)
}

// SetWithResult 使用全局默认客户端设置幂等标记并存储操作结果
func SetWithResult(ctx context.Context, key string, result interface{}, ttl time.Duration) (bool, error) {
	return getDefaultClient().SetWithResult(ctx, key, result, ttl)
}

// GetResult 使用全局默认客户端获取存储的操作结果
func GetResult(ctx context.Context, key string) (interface{}, error) {
	return getDefaultClient().GetResult(ctx, key)
}

// Delete 使用全局默认客户端删除幂等标记
func Delete(ctx context.Context, key string) error {
	return getDefaultClient().Delete(ctx, key)
}

// Exists 使用全局默认客户端检查键是否存在（别名方法，与 Check 功能相同）
func Exists(ctx context.Context, key string) (bool, error) {
	return getDefaultClient().Exists(ctx, key)
}

// TTL 使用全局默认客户端获取键的剩余过期时间
func TTL(ctx context.Context, key string) (time.Duration, error) {
	return getDefaultClient().TTL(ctx, key)
}

// Refresh 使用全局默认客户端刷新键的过期时间
func Refresh(ctx context.Context, key string, ttl time.Duration) error {
	return getDefaultClient().Refresh(ctx, key, ttl)
}

// ===== 便捷方法 =====

// Execute 执行幂等操作，如果是首次执行则调用回调函数
// 这是一个高级便捷方法，封装了常见的幂等使用模式
//
// 示例：
//
//	result, err := idempotent.Execute(ctx, "user:create:123", time.Hour, func() (interface{}, error) {
//	    // 执行实际的业务逻辑
//	    user := createUser(123)
//	    return user, nil
//	})
func Execute(ctx context.Context, key string, ttl time.Duration, callback func() (interface{}, error)) (interface{}, error) {
	// 先检查是否已经执行过
	exists, err := Check(ctx, key)
	if err != nil {
		return nil, err
	}

	if exists {
		// 已经执行过，获取结果
		result, err := GetResult(ctx, key)
		if err != nil {
			return nil, err
		}
		logger.Debug("幂等操作已执行，返回缓存结果",
			clog.String("key", key),
		)
		return result, nil
	}

	// 首次执行，调用回调函数
	result, err := callback()
	if err != nil {
		logger.Error("幂等操作回调执行失败",
			clog.String("key", key),
			clog.Err(err),
		)
		return nil, err
	}

	// 存储结果
	success, err := SetWithResult(ctx, key, result, ttl)
	if err != nil {
		return nil, err
	}

	if !success {
		// 并发情况下，其他协程已经设置了结果
		// 获取已设置的结果
		cachedResult, err := GetResult(ctx, key)
		if err != nil {
			return nil, err
		}
		logger.Debug("并发执行检测到，返回已缓存结果",
			clog.String("key", key),
		)
		return cachedResult, nil
	}

	logger.Debug("幂等操作首次执行完成",
		clog.String("key", key),
	)
	return result, nil
}

// ExecuteSimple 执行简单的幂等操作，只设置标记不存储结果
//
// 示例：
//
//	err := idempotent.ExecuteSimple(ctx, "notification:send:123", time.Hour, func() error {
//	    return sendNotification(123)
//	})
func ExecuteSimple(ctx context.Context, key string, ttl time.Duration, callback func() error) error {
	// 尝试设置幂等标记
	success, err := Set(ctx, key, ttl)
	if err != nil {
		return err
	}

	if !success {
		// 已经执行过
		logger.Debug("幂等操作已执行，跳过",
			clog.String("key", key),
		)
		return nil
	}

	// 首次执行，调用回调函数
	err = callback()
	if err != nil {
		// 执行失败，删除标记以允许重试
		deleteErr := Delete(ctx, key)
		if deleteErr != nil {
			logger.Error("删除失败的幂等标记时出错",
				clog.String("key", key),
				clog.Err(deleteErr),
			)
		}
		logger.Error("幂等操作回调执行失败",
			clog.String("key", key),
			clog.Err(err),
		)
		return err
	}

	logger.Debug("幂等操作首次执行完成",
		clog.String("key", key),
	)
	return nil
}

// Default 返回全局默认幂等客户端实例
func Default() Idempotent {
	return getDefaultClient()
}
