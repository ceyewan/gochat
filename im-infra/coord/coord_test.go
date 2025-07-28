package coord

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestCoordinatorCreation 测试协调器创建
func TestCoordinatorCreation(t *testing.T) {
	tests := []struct {
		name    string
		config  CoordinatorConfig
		wantErr bool
	}{
		{
			name:    "default config",
			config:  DefaultConfig(),
			wantErr: false,
		},
		{
			name: "custom config",
			config: CoordinatorConfig{
				Endpoints: []string{"localhost:2379"},
				Timeout:   10 * time.Second,
			},
			wantErr: false,
		},
		{
			name: "invalid endpoints",
			config: CoordinatorConfig{
				Endpoints: []string{"invalid:99999"},
				Timeout:   1 * time.Second,
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			coord, err := New(tt.config)
			if tt.wantErr {
				assert.Error(t, err)
				assert.Nil(t, coord)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, coord)
				if coord != nil {
					defer coord.Close()

					// 验证各个服务都可以获取
					assert.NotNil(t, coord.Lock())
					assert.NotNil(t, coord.Registry())
					assert.NotNil(t, coord.Config())
				}
			}
		})
	}
}

// TestCoordinatorServices 测试协调器服务
func TestCoordinatorServices(t *testing.T) {
	coord, err := New(DefaultConfig())
	require.NoError(t, err)
	defer coord.Close()

	// 测试获取各个服务
	lockService := coord.Lock()
	assert.NotNil(t, lockService)

	registryService := coord.Registry()
	assert.NotNil(t, registryService)

	configService := coord.Config()
	assert.NotNil(t, configService)
}

// TestCoordinatorClose 测试协调器关闭
func TestCoordinatorClose(t *testing.T) {
	coord, err := New(DefaultConfig())
	require.NoError(t, err)

	// 第一次关闭应该成功
	err = coord.Close()
	assert.NoError(t, err)

	// 第二次关闭应该也成功（幂等操作）
	err = coord.Close()
	assert.NoError(t, err)
}

// TestDefaultConfig 测试默认配置
func TestDefaultConfig(t *testing.T) {
	config := DefaultConfig()

	assert.NotEmpty(t, config.Endpoints)
	assert.Equal(t, []string{"localhost:2379"}, config.Endpoints)
	assert.Equal(t, 5*time.Second, config.Timeout)
	assert.NotNil(t, config.RetryConfig)
	assert.Equal(t, 3, config.RetryConfig.MaxAttempts)
	assert.Equal(t, 100*time.Millisecond, config.RetryConfig.InitialDelay)
	assert.Equal(t, 2*time.Second, config.RetryConfig.MaxDelay)
	assert.Equal(t, 2.0, config.RetryConfig.Multiplier)
}

// TestRetryConfig 测试重试配置
func TestRetryConfig(t *testing.T) {
	retryConfig := &RetryConfig{
		MaxAttempts:  5,
		InitialDelay: 200 * time.Millisecond,
		MaxDelay:     10 * time.Second,
		Multiplier:   1.5,
	}

	config := CoordinatorConfig{
		Endpoints:   []string{"localhost:2379"},
		Timeout:     5 * time.Second,
		RetryConfig: retryConfig,
	}

	coord, err := New(config)
	require.NoError(t, err)
	defer coord.Close()

	assert.NotNil(t, coord)
}

// BenchmarkCoordinatorCreation 基准测试协调器创建
func BenchmarkCoordinatorCreation(b *testing.B) {
	config := DefaultConfig()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		coord, err := New(config)
		if err != nil {
			b.Fatal(err)
		}
		coord.Close()
	}
}

// BenchmarkCoordinatorServices 基准测试服务获取
func BenchmarkCoordinatorServices(b *testing.B) {
	coord, err := New(DefaultConfig())
	if err != nil {
		b.Fatal(err)
	}
	defer coord.Close()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = coord.Lock()
		_ = coord.Registry()
		_ = coord.Config()
	}
}
