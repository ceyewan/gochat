package coordination

import (
	"testing"
	"time"

	"github.com/ceyewan/gochat/im-infra/coordination/internal"
)

func TestDefaultConfig(t *testing.T) {
	cfg := DefaultConfig()

	// 验证默认配置
	if len(cfg.Endpoints) == 0 {
		t.Error("默认配置应该包含 etcd 端点")
	}

	if cfg.DialTimeout <= 0 {
		t.Error("默认配置应该有正的连接超时时间")
	}

	if cfg.ServiceRegistry.KeyPrefix == "" {
		t.Error("服务注册配置应该有键前缀")
	}

	if cfg.DistributedLock.KeyPrefix == "" {
		t.Error("分布式锁配置应该有键前缀")
	}

	if cfg.ConfigCenter.KeyPrefix == "" {
		t.Error("配置中心配置应该有键前缀")
	}
}

func TestDevelopmentConfig(t *testing.T) {
	cfg := DevelopmentConfig()

	if cfg.LogLevel != "debug" {
		t.Errorf("开发配置的日志级别应该是 debug，实际是 %s", cfg.LogLevel)
	}

	if cfg.ServiceRegistry.TTL != 15*time.Second {
		t.Errorf("开发配置的服务 TTL 应该是 15s，实际是 %v", cfg.ServiceRegistry.TTL)
	}
}

func TestProductionConfig(t *testing.T) {
	cfg := ProductionConfig()

	if cfg.LogLevel != "warn" {
		t.Errorf("生产配置的日志级别应该是 warn，实际是 %s", cfg.LogLevel)
	}

	if !cfg.EnableMetrics {
		t.Error("生产配置应该启用指标收集")
	}

	if !cfg.EnableTracing {
		t.Error("生产配置应该启用链路追踪")
	}
}

func TestTestConfig(t *testing.T) {
	cfg := TestConfig()

	if cfg.LogLevel != "debug" {
		t.Errorf("测试配置的日志级别应该是 debug，实际是 %s", cfg.LogLevel)
	}

	if cfg.ServiceRegistry.TTL != 5*time.Second {
		t.Errorf("测试配置的服务 TTL 应该是 5s，实际是 %v", cfg.ServiceRegistry.TTL)
	}

	if cfg.Retry.MaxRetries != 1 {
		t.Errorf("测试配置的最大重试次数应该是 1，实际是 %d", cfg.Retry.MaxRetries)
	}
}

func TestConfigValidation(t *testing.T) {
	tests := []struct {
		name    string
		config  Config
		wantErr bool
	}{
		{
			name:    "有效的默认配置",
			config:  DefaultConfig(),
			wantErr: false,
		},
		{
			name: "空的端点列表",
			config: Config{
				Endpoints:   []string{},
				DialTimeout: 5 * time.Second,
			},
			wantErr: true,
		},
		{
			name: "负的连接超时",
			config: Config{
				Endpoints:   []string{"localhost:2379"},
				DialTimeout: -1 * time.Second,
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.config.Validate()
			if (err != nil) != tt.wantErr {
				t.Errorf("Config.Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestServiceInfo(t *testing.T) {
	service := ServiceInfo{
		Name:       "test-service",
		InstanceID: "test-instance",
		Address:    "localhost:8080",
		Metadata: map[string]string{
			"version": "1.0.0",
		},
		Health: internal.HealthHealthy,
	}

	if service.Name != "test-service" {
		t.Errorf("服务名称不匹配，期望 test-service，实际 %s", service.Name)
	}

	if service.Health.String() != "healthy" {
		t.Errorf("健康状态字符串不匹配，期望 healthy，实际 %s", service.Health.String())
	}
}

func TestHealthStatus(t *testing.T) {
	tests := []struct {
		status   HealthStatus
		expected string
	}{
		{internal.HealthHealthy, "healthy"},
		{internal.HealthUnhealthy, "unhealthy"},
		{internal.HealthMaintenance, "maintenance"},
		{internal.HealthUnknown, "unknown"},
	}

	for _, tt := range tests {
		t.Run(tt.expected, func(t *testing.T) {
			if got := tt.status.String(); got != tt.expected {
				t.Errorf("HealthStatus.String() = %v, want %v", got, tt.expected)
			}
		})
	}
}

func TestLoadBalanceStrategy(t *testing.T) {
	tests := []struct {
		strategy LoadBalanceStrategy
		expected string
	}{
		{internal.LoadBalanceRoundRobin, "round_robin"},
		{internal.LoadBalanceRandom, "random"},
		{internal.LoadBalanceWeighted, "weighted"},
		{internal.LoadBalanceLeastConn, "least_conn"},
	}

	for _, tt := range tests {
		t.Run(tt.expected, func(t *testing.T) {
			if got := tt.strategy.String(); got != tt.expected {
				t.Errorf("LoadBalanceStrategy.String() = %v, want %v", got, tt.expected)
			}
		})
	}
}

func TestConfigChangeType(t *testing.T) {
	tests := []struct {
		changeType internal.ConfigChangeType
		expected   string
	}{
		{internal.ConfigChangeCreate, "create"},
		{internal.ConfigChangeUpdate, "update"},
		{internal.ConfigChangeDelete, "delete"},
	}

	for _, tt := range tests {
		t.Run(tt.expected, func(t *testing.T) {
			if got := tt.changeType.String(); got != tt.expected {
				t.Errorf("ConfigChangeType.String() = %v, want %v", got, tt.expected)
			}
		})
	}
}

func TestNewServiceRegistryConfig(t *testing.T) {
	config := NewServiceRegistryConfig(
		"/test-services",
		60*time.Second,
		20*time.Second,
		true,
	)

	if config.KeyPrefix != "/test-services" {
		t.Errorf("键前缀不匹配，期望 /test-services，实际 %s", config.KeyPrefix)
	}

	if config.TTL != 60*time.Second {
		t.Errorf("TTL 不匹配，期望 60s，实际 %v", config.TTL)
	}

	if !config.EnableHealthCheck {
		t.Error("应该启用健康检查")
	}
}

func TestNewDistributedLockConfig(t *testing.T) {
	config := NewDistributedLockConfig(
		"/test-locks",
		45*time.Second,
		15*time.Second,
		true,
	)

	if config.KeyPrefix != "/test-locks" {
		t.Errorf("键前缀不匹配，期望 /test-locks，实际 %s", config.KeyPrefix)
	}

	if config.DefaultTTL != 45*time.Second {
		t.Errorf("默认 TTL 不匹配，期望 45s，实际 %v", config.DefaultTTL)
	}

	if !config.EnableReentrant {
		t.Error("应该启用可重入锁")
	}
}

func TestNewConfigCenterConfig(t *testing.T) {
	config := NewConfigCenterConfig(
		"/test-config",
		true,
		50,
		true,
	)

	if config.KeyPrefix != "/test-config" {
		t.Errorf("键前缀不匹配，期望 /test-config，实际 %s", config.KeyPrefix)
	}

	if !config.EnableVersioning {
		t.Error("应该启用版本控制")
	}

	if config.MaxVersionHistory != 50 {
		t.Errorf("最大版本历史不匹配，期望 50，实际 %d", config.MaxVersionHistory)
	}
}

func TestNewRetryConfig(t *testing.T) {
	config := NewRetryConfig(
		5,
		200*time.Millisecond,
		10*time.Second,
		1.5,
	)

	if config.MaxRetries != 5 {
		t.Errorf("最大重试次数不匹配，期望 5，实际 %d", config.MaxRetries)
	}

	if config.Multiplier != 1.5 {
		t.Errorf("倍数不匹配，期望 1.5，实际 %f", config.Multiplier)
	}
}

// 基准测试
func BenchmarkDefaultConfig(b *testing.B) {
	for i := 0; i < b.N; i++ {
		_ = DefaultConfig()
	}
}

func BenchmarkConfigValidation(b *testing.B) {
	cfg := DefaultConfig()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		_ = cfg.Validate()
	}
}
