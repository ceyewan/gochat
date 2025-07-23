package coordination

import (
	"time"

	"github.com/ceyewan/gochat/im-infra/coordination/internal"
)

// Config 是 coordination 的主配置结构体。
// 用于声明式地定义分布式协调组件的行为和连接参数。
type Config = internal.Config

// ServiceRegistryConfig 定义服务注册与发现的配置。
type ServiceRegistryConfig = internal.ServiceRegistryConfig

// DistributedLockConfig 定义分布式锁的配置。
type DistributedLockConfig = internal.DistributedLockConfig

// ConfigCenterConfig 定义配置中心的配置。
type ConfigCenterConfig = internal.ConfigCenterConfig

// RetryConfig 定义重试策略配置。
type RetryConfig = internal.RetryConfig

// DefaultConfig 返回一个带有合理默认值的 Config。
// 默认配置适用于大多数开发和测试场景。
//
// 示例：
//
//	// 使用默认配置
//	cfg := coordination.DefaultConfig()
//
//	// 可以根据需要修改特定配置
//	cfg.Endpoints = []string{"localhost:2379", "localhost:2380"}
//	cfg.ServiceRegistry.TTL = 60 * time.Second
//
//	coordinator, err := coordination.New(cfg)
//	if err != nil {
//		log.Fatal(err)
//	}
func DefaultConfig() Config {
	return internal.DefaultConfig()
}

// DevelopmentConfig 返回适用于开发环境的配置。
// 开发配置使用较短的超时时间和更详细的日志。
//
// 示例：
//
//	cfg := coordination.DevelopmentConfig()
//	coordinator, err := coordination.New(cfg)
//	if err != nil {
//		log.Fatal(err)
//	}
func DevelopmentConfig() Config {
	return internal.DevelopmentConfig()
}

// ProductionConfig 返回适用于生产环境的配置。
// 生产配置使用较长的超时时间、重试机制和性能优化。
//
// 示例：
//
//	cfg := coordination.ProductionConfig()
//	// 根据实际环境调整 etcd 集群地址
//	cfg.Endpoints = []string{
//		"etcd-1.example.com:2379",
//		"etcd-2.example.com:2379",
//		"etcd-3.example.com:2379",
//	}
//	coordinator, err := coordination.New(cfg)
//	if err != nil {
//		log.Fatal(err)
//	}
func ProductionConfig() Config {
	return internal.ProductionConfig()
}

// TestConfig 返回适用于测试环境的配置。
// 测试配置使用内存存储和快速超时，适合单元测试和集成测试。
//
// 示例：
//
//	cfg := coordination.TestConfig()
//	coordinator, err := coordination.New(cfg)
//	if err != nil {
//		t.Fatal(err)
//	}
//	defer coordinator.Close()
func TestConfig() Config {
	return internal.TestConfig()
}

// ExampleConfig 返回适用于示例演示的配置。
// 提供快速失败和清晰的错误信息，适合演示和学习使用。
//
// 示例：
//
//	cfg := coordination.ExampleConfig()
//	coordinator, err := coordination.New(cfg)
//	if err != nil {
//		log.Printf("连接失败: %v", err)
//		return
//	}
//	defer coordinator.Close()
func ExampleConfig() Config {
	return internal.ExampleConfig()
}

// NewServiceRegistryConfig 创建服务注册配置。
//
// 示例：
//
//	registryConfig := coordination.NewServiceRegistryConfig(
//		"/services",           // keyPrefix
//		30*time.Second,        // ttl
//		5*time.Second,         // healthCheckInterval
//		true,                  // enableHealthCheck
//	)
func NewServiceRegistryConfig(keyPrefix string, ttl, healthCheckInterval time.Duration, enableHealthCheck bool) ServiceRegistryConfig {
	return internal.NewServiceRegistryConfig(keyPrefix, ttl, healthCheckInterval, enableHealthCheck)
}

// NewDistributedLockConfig 创建分布式锁配置。
//
// 示例：
//
//	lockConfig := coordination.NewDistributedLockConfig(
//		"/locks",              // keyPrefix
//		30*time.Second,        // defaultTTL
//		5*time.Second,         // renewInterval
//		true,                  // enableReentrant
//	)
func NewDistributedLockConfig(keyPrefix string, defaultTTL, renewInterval time.Duration, enableReentrant bool) DistributedLockConfig {
	return internal.NewDistributedLockConfig(keyPrefix, defaultTTL, renewInterval, enableReentrant)
}

// NewConfigCenterConfig 创建配置中心配置。
//
// 示例：
//
//	configConfig := coordination.NewConfigCenterConfig(
//		"/config",             // keyPrefix
//		true,                  // enableVersioning
//		100,                   // maxVersionHistory
//		true,                  // enableValidation
//	)
func NewConfigCenterConfig(keyPrefix string, enableVersioning bool, maxVersionHistory int, enableValidation bool) ConfigCenterConfig {
	return internal.NewConfigCenterConfig(keyPrefix, enableVersioning, maxVersionHistory, enableValidation)
}

// NewRetryConfig 创建重试配置。
//
// 示例：
//
//	retryConfig := coordination.NewRetryConfig(
//		3,                     // maxRetries
//		100*time.Millisecond,  // initialInterval
//		5*time.Second,         // maxInterval
//		2.0,                   // multiplier
//	)
func NewRetryConfig(maxRetries int, initialInterval, maxInterval time.Duration, multiplier float64) RetryConfig {
	return internal.NewRetryConfig(maxRetries, initialInterval, maxInterval, multiplier)
}
