package coordination

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCoordinatorOptions_Validate(t *testing.T) {
	tests := []struct {
		name    string
		opts    CoordinatorOptions
		wantErr bool
	}{
		{
			name: "valid options",
			opts: CoordinatorOptions{
				Endpoints: []string{"localhost:2379"},
				Timeout:   5 * time.Second,
			},
			wantErr: false,
		},
		{
			name: "empty endpoints",
			opts: CoordinatorOptions{
				Endpoints: []string{},
				Timeout:   5 * time.Second,
			},
			wantErr: true,
		},
		{
			name: "zero timeout",
			opts: CoordinatorOptions{
				Endpoints: []string{"localhost:2379"},
				Timeout:   0,
			},
			wantErr: true,
		},
		{
			name: "invalid retry config",
			opts: CoordinatorOptions{
				Endpoints: []string{"localhost:2379"},
				Timeout:   5 * time.Second,
				RetryConfig: &RetryConfig{
					MaxAttempts:  -1,
					InitialDelay: time.Second,
					MaxDelay:     5 * time.Second,
					Multiplier:   2.0,
				},
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.opts.Validate()
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestDefaultCoordinatorOptions(t *testing.T) {
	opts := DefaultCoordinatorOptions()

	assert.NotEmpty(t, opts.Endpoints)
	assert.Greater(t, opts.Timeout, time.Duration(0))
	assert.NotNil(t, opts.RetryConfig)
	assert.Greater(t, opts.RetryConfig.MaxAttempts, 0)
	assert.Greater(t, opts.RetryConfig.InitialDelay, time.Duration(0))
	assert.Greater(t, opts.RetryConfig.MaxDelay, time.Duration(0))
	assert.Greater(t, opts.RetryConfig.Multiplier, 1.0)

	// 验证默认选项是有效的
	assert.NoError(t, opts.Validate())
}

func TestCoordinationError(t *testing.T) {
	// 测试错误创建
	err := NewCoordinationError(ErrCodeValidation, "test error", nil)
	assert.Equal(t, ErrCodeValidation, err.Code)
	assert.Equal(t, "test error", err.Message)
	assert.Nil(t, err.Cause)
	assert.Equal(t, "[VALIDATION_ERROR] test error", err.Error())

	// 测试带原因的错误
	cause := assert.AnError
	err = NewCoordinationError(ErrCodeConnection, "connection failed", cause)
	assert.Equal(t, ErrCodeConnection, err.Code)
	assert.Equal(t, "connection failed", err.Message)
	assert.Equal(t, cause, err.Cause)
	assert.Contains(t, err.Error(), "connection failed")
	assert.Contains(t, err.Error(), cause.Error())

	// 测试错误检查函数
	assert.True(t, IsCoordinationError(err))
	assert.False(t, IsCoordinationError(assert.AnError))

	// 测试错误码获取
	assert.Equal(t, ErrCodeConnection, GetErrorCode(err))
	assert.Equal(t, ErrorCode(""), GetErrorCode(assert.AnError))
}

func TestNewCoordinator(t *testing.T) {
	// 测试无效配置
	opts := CoordinatorOptions{
		Endpoints: []string{}, // 空端点应该失败
		Timeout:   5 * time.Second,
	}

	_, err := NewCoordinator(opts)
	assert.Error(t, err)
	assert.True(t, IsCoordinationError(err))
	assert.Equal(t, ErrCodeValidation, GetErrorCode(err))
}

// 集成测试 - 需要 etcd 运行
func TestCoordinatorIntegration(t *testing.T) {
	if testing.Short() {
		t.Skip("跳过集成测试")
	}

	opts := CoordinatorOptions{
		Endpoints: []string{"localhost:2379"},
		Timeout:   3 * time.Second,
		RetryConfig: &RetryConfig{
			MaxAttempts:  2,
			InitialDelay: 100 * time.Millisecond,
			MaxDelay:     1 * time.Second,
			Multiplier:   2.0,
		},
	}

	coord, err := NewCoordinator(opts)
	if err != nil {
		t.Skipf("无法连接到 etcd，跳过集成测试: %v", err)
		return
	}
	defer coord.Close()

	ctx := context.Background()

	// 测试分布式锁
	t.Run("DistributedLock", func(t *testing.T) {
		lockService := coord.Lock()
		require.NotNil(t, lockService)

		// 获取锁
		lock, err := lockService.Acquire(ctx, "test-lock", 10*time.Second)
		require.NoError(t, err)
		require.NotNil(t, lock)
		assert.Equal(t, "/locks/test-lock", lock.Key())

		// 检查 TTL
		ttl, err := lock.TTL(ctx)
		require.NoError(t, err)
		assert.Greater(t, ttl, 5*time.Second)

		// 释放锁
		err = lock.Unlock(ctx)
		require.NoError(t, err)

		// 尝试非阻塞获取锁
		tryLock, err := lockService.TryAcquire(ctx, "test-try-lock", 5*time.Second)
		require.NoError(t, err)
		require.NotNil(t, tryLock)
		tryLock.Unlock(ctx)
	})

	// 测试配置中心
	t.Run("ConfigCenter", func(t *testing.T) {
		configService := coord.Config()
		require.NotNil(t, configService)

		testKey := "test-config"
		testValue := "test-value"

		// 设置配置
		err := configService.Set(ctx, testKey, testValue)
		require.NoError(t, err)

		// 获取配置
		value, err := configService.Get(ctx, testKey)
		require.NoError(t, err)
		assert.Equal(t, testValue, value)

		// 列出配置
		keys, err := configService.List(ctx, "")
		require.NoError(t, err)
		assert.Contains(t, keys, testKey)

		// 删除配置
		err = configService.Delete(ctx, testKey)
		require.NoError(t, err)

		// 验证删除
		_, err = configService.Get(ctx, testKey)
		assert.Error(t, err)
		assert.True(t, IsCoordinationError(err))
		assert.Equal(t, ErrCodeNotFound, GetErrorCode(err))
	})

	// 测试服务注册发现
	t.Run("ServiceRegistry", func(t *testing.T) {
		registryService := coord.Registry()
		require.NotNil(t, registryService)

		service := ServiceInfo{
			ID:      "test-service-001",
			Name:    "test-service",
			Address: "127.0.0.1",
			Port:    8080,
			Metadata: map[string]string{
				"version": "1.0.0",
			},
			TTL: 30 * time.Second,
		}

		// 注册服务
		err := registryService.Register(ctx, service)
		require.NoError(t, err)

		// 发现服务
		services, err := registryService.Discover(ctx, "test-service")
		require.NoError(t, err)
		require.Len(t, services, 1)
		assert.Equal(t, service.ID, services[0].ID)
		assert.Equal(t, service.Name, services[0].Name)
		assert.Equal(t, service.Address, services[0].Address)
		assert.Equal(t, service.Port, services[0].Port)

		// 注销服务
		err = registryService.Unregister(ctx, service.ID)
		require.NoError(t, err)

		// 验证注销
		services, err = registryService.Discover(ctx, "test-service")
		require.NoError(t, err)
		assert.Empty(t, services)
	})
}

// 全局方法测试
func TestGlobalMethods(t *testing.T) {
	if testing.Short() {
		t.Skip("跳过集成测试")
	}

	ctx := context.Background()

	// 测试全局锁方法
	t.Run("GlobalLockMethods", func(t *testing.T) {
		lock, err := AcquireLock(ctx, "global-test-lock", 10*time.Second)
		if err != nil {
			t.Skipf("无法连接到 etcd，跳过全局方法测试: %v", err)
			return
		}
		require.NotNil(t, lock)

		err = lock.Unlock(ctx)
		require.NoError(t, err)

		// 测试尝试获取锁
		tryLock, err := TryAcquireLock(ctx, "global-try-lock", 5*time.Second)
		require.NoError(t, err)
		require.NotNil(t, tryLock)
		tryLock.Unlock(ctx)
	})

	// 测试全局配置方法
	t.Run("GlobalConfigMethods", func(t *testing.T) {
		testKey := "global-test-config"
		testValue := "global-test-value"

		err := SetConfig(ctx, testKey, testValue)
		if err != nil {
			t.Skipf("无法连接到 etcd，跳过全局配置测试: %v", err)
			return
		}

		value, err := GetConfig(ctx, testKey)
		require.NoError(t, err)
		assert.Equal(t, testValue, value)

		keys, err := ListConfigs(ctx, "")
		require.NoError(t, err)
		assert.Contains(t, keys, testKey)

		err = DeleteConfig(ctx, testKey)
		require.NoError(t, err)
	})

	// 测试全局服务注册方法
	t.Run("GlobalRegistryMethods", func(t *testing.T) {
		service := ServiceInfo{
			ID:      "global-test-service-001",
			Name:    "global-test-service",
			Address: "127.0.0.1",
			Port:    9090,
			TTL:     30 * time.Second,
		}

		err := RegisterService(ctx, service)
		if err != nil {
			t.Skipf("无法连接到 etcd，跳过全局服务注册测试: %v", err)
			return
		}

		services, err := DiscoverServices(ctx, "global-test-service")
		require.NoError(t, err)
		require.Len(t, services, 1)
		assert.Equal(t, service.ID, services[0].ID)

		err = UnregisterService(ctx, service.ID)
		require.NoError(t, err)
	})
}

// 基准测试
func BenchmarkLockAcquireRelease(b *testing.B) {
	if testing.Short() {
		b.Skip("跳过基准测试")
	}

	opts := DefaultCoordinatorOptions()
	coord, err := NewCoordinator(opts)
	if err != nil {
		b.Skipf("无法连接到 etcd，跳过基准测试: %v", err)
		return
	}
	defer coord.Close()

	ctx := context.Background()
	lockService := coord.Lock()

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		i := 0
		for pb.Next() {
			lockKey := fmt.Sprintf("bench-lock-%d", i)
			lock, err := lockService.Acquire(ctx, lockKey, 5*time.Second)
			if err != nil {
				b.Errorf("获取锁失败: %v", err)
				continue
			}
			lock.Unlock(ctx)
			i++
		}
	})
}

func BenchmarkConfigSetGet(b *testing.B) {
	if testing.Short() {
		b.Skip("跳过基准测试")
	}

	opts := DefaultCoordinatorOptions()
	coord, err := NewCoordinator(opts)
	if err != nil {
		b.Skipf("无法连接到 etcd，跳过基准测试: %v", err)
		return
	}
	defer coord.Close()

	ctx := context.Background()
	configService := coord.Config()

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		i := 0
		for pb.Next() {
			key := fmt.Sprintf("bench-config-%d", i)
			value := fmt.Sprintf("bench-value-%d", i)

			err := configService.Set(ctx, key, value)
			if err != nil {
				b.Errorf("设置配置失败: %v", err)
				continue
			}

			_, err = configService.Get(ctx, key)
			if err != nil {
				b.Errorf("获取配置失败: %v", err)
				continue
			}

			configService.Delete(ctx, key)
			i++
		}
	})
}
