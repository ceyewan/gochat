package coord

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/ceyewan/gochat/im-infra/coord/registry"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestIntegrationFullWorkflow 测试完整的工作流程
func TestIntegrationFullWorkflow(t *testing.T) {
	coord, err := New(DefaultConfig())
	require.NoError(t, err)
	defer coord.Close()

	ctx := context.Background()

	// 1. 测试分布式锁
	t.Run("DistributedLock", func(t *testing.T) {
		lockService := coord.Lock()
		lockKey := "integration-lock"

		lock, err := lockService.Acquire(ctx, lockKey, 30*time.Second)
		require.NoError(t, err)
		assert.NotNil(t, lock)

		err = lock.Unlock(ctx)
		assert.NoError(t, err)
	})

	// 2. 测试配置中心
	t.Run("ConfigCenter", func(t *testing.T) {
		configService := coord.Config()

		// 设置配置
		err := configService.Set(ctx, "integration/app/name", "gochat")
		require.NoError(t, err)

		// 获取配置
		var appName string
		err = configService.Get(ctx, "integration/app/name", &appName)
		require.NoError(t, err)
		assert.Equal(t, "gochat", appName)

		// 删除配置
		err = configService.Delete(ctx, "integration/app/name")
		assert.NoError(t, err)
	})

	// 3. 测试服务注册发现
	t.Run("ServiceRegistry", func(t *testing.T) {
		registryService := coord.Registry()

		service := registry.ServiceInfo{
			ID:      "integration-service-1",
			Name:    "integration-service",
			Address: "127.0.0.1",
			Port:    8080,
			Metadata: map[string]string{
				"version": "1.0.0",
			},
		}

		// 注册服务
		err := registryService.Register(ctx, service, 30*time.Second)
		require.NoError(t, err)

		// 发现服务
		services, err := registryService.Discover(ctx, "integration-service")
		require.NoError(t, err)
		assert.Len(t, services, 1)
		assert.Equal(t, service.ID, services[0].ID)

		// 注销服务
		err = registryService.Unregister(ctx, "integration-service-1")
		assert.NoError(t, err)
	})
}

// TestIntegrationConcurrentOperations 测试并发操作
func TestIntegrationConcurrentOperations(t *testing.T) {
	coord, err := New(DefaultConfig())
	require.NoError(t, err)
	defer coord.Close()

	ctx := context.Background()
	var wg sync.WaitGroup

	// 并发锁操作
	t.Run("ConcurrentLocks", func(t *testing.T) {
		lockService := coord.Lock()
		lockKey := "concurrent-lock"
		successCount := int32(0)
		var mu sync.Mutex

		for i := 0; i < 10; i++ {
			wg.Add(1)
			go func(id int) {
				defer wg.Done()

				lock, err := lockService.TryAcquire(ctx, lockKey, 10*time.Second)
				if err == nil && lock != nil {
					mu.Lock()
					successCount++
					mu.Unlock()

					time.Sleep(50 * time.Millisecond)
					lock.Unlock(ctx)
				}
			}(i)
		}

		wg.Wait()
		assert.Equal(t, int32(1), successCount)
	})

	// 并发配置操作
	t.Run("ConcurrentConfig", func(t *testing.T) {
		configService := coord.Config()

		for i := 0; i < 5; i++ {
			wg.Add(1)
			go func(id int) {
				defer wg.Done()

				key := "concurrent/config"
				value := "value"

				err := configService.Set(ctx, key, value)
				assert.NoError(t, err)

				var retrievedValue string
				err = configService.Get(ctx, key, &retrievedValue)
				assert.NoError(t, err)
			}(i)
		}

		wg.Wait()
	})

	// 并发服务注册
	t.Run("ConcurrentRegistry", func(t *testing.T) {
		registryService := coord.Registry()
		serviceName := "concurrent-service"

		for i := 0; i < 5; i++ {
			wg.Add(1)
			go func(id int) {
				defer wg.Done()

				service := registry.ServiceInfo{
					ID:      "concurrent-service-" + string(rune(id+'0')),
					Name:    serviceName,
					Address: "127.0.0.1",
					Port:    8080 + id,
				}

				err := registryService.Register(ctx, service, 30*time.Second)
				assert.NoError(t, err)

				services, err := registryService.Discover(ctx, serviceName)
				assert.NoError(t, err)
				assert.NotEmpty(t, services)

				err = registryService.Unregister(ctx, service.ID)
				assert.NoError(t, err)
			}(i)
		}

		wg.Wait()
	})
}

// TestIntegrationServiceDiscoveryWithWatch 测试服务发现与监听的集成
func TestIntegrationServiceDiscoveryWithWatch(t *testing.T) {
	coord, err := New(DefaultConfig())
	require.NoError(t, err)
	defer coord.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	registryService := coord.Registry()
	serviceName := "watch-integration-service"

	// 开始监听
	watchCh, err := registryService.Watch(ctx, serviceName)
	require.NoError(t, err)

	var wg sync.WaitGroup
	eventCount := 0

	// 监听事件
	wg.Add(1)
	go func() {
		defer wg.Done()
		for event := range watchCh {
			eventCount++
			assert.NotEmpty(t, event.Type)
			assert.NotNil(t, event.Service)

			if eventCount >= 4 { // 注册2个，注销2个
				return
			}
		}
	}()

	// 注册和注销服务
	wg.Add(1)
	go func() {
		defer wg.Done()

		time.Sleep(200 * time.Millisecond)

		// 注册第一个服务
		service1 := registry.ServiceInfo{
			ID:      "watch-service-1",
			Name:    serviceName,
			Address: "127.0.0.1",
			Port:    8080,
		}
		registryService.Register(context.Background(), service1, 30*time.Second)

		time.Sleep(200 * time.Millisecond)

		// 注册第二个服务
		service2 := registry.ServiceInfo{
			ID:      "watch-service-2",
			Name:    serviceName,
			Address: "127.0.0.1",
			Port:    8081,
		}
		registryService.Register(context.Background(), service2, 30*time.Second)

		time.Sleep(200 * time.Millisecond)

		// 注销服务
		registryService.Unregister(context.Background(), "watch-service-1")

		time.Sleep(200 * time.Millisecond)

		registryService.Unregister(context.Background(), "watch-service-2")
	}()

	wg.Wait()
	assert.GreaterOrEqual(t, eventCount, 2)
}

// TestIntegrationConfigWatchWithLock 测试配置监听与锁的集成
func TestIntegrationConfigWatchWithLock(t *testing.T) {
	coord, err := New(DefaultConfig())
	require.NoError(t, err)
	defer coord.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	configService := coord.Config()
	lockService := coord.Lock()

	configKey := "integration/watch/config"
	lockKey := "integration-config-lock"

	// 获取锁
	lock, err := lockService.Acquire(ctx, lockKey, 30*time.Second)
	require.NoError(t, err)
	defer lock.Unlock(ctx)

	// 开始监听配置
	var watchValue string
	watcher, err := configService.Watch(ctx, configKey, &watchValue)
	require.NoError(t, err)
	defer watcher.Close()

	var wg sync.WaitGroup
	eventReceived := false

	// 监听配置变化
	wg.Add(1)
	go func() {
		defer wg.Done()
		for event := range watcher.Chan() {
			if event.Key == configKey {
				eventReceived = true
				return
			}
		}
	}()

	// 在持有锁的情况下修改配置
	time.Sleep(100 * time.Millisecond)
	err = configService.Set(ctx, configKey, "locked-value")
	require.NoError(t, err)

	wg.Wait()
	assert.True(t, eventReceived)

	// 清理
	configService.Delete(ctx, configKey)
}

// BenchmarkIntegrationFullWorkflow 基准测试完整工作流程
func BenchmarkIntegrationFullWorkflow(b *testing.B) {
	coord, err := New(DefaultConfig())
	if err != nil {
		b.Fatal(err)
	}
	defer coord.Close()

	ctx := context.Background()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		// 锁操作
		lock, err := coord.Lock().TryAcquire(ctx, "bench-lock", 10*time.Second)
		if err == nil && lock != nil {
			lock.Unlock(ctx)
		}

		// 配置操作
		coord.Config().Set(ctx, "bench/config", "value")
		var value string
		coord.Config().Get(ctx, "bench/config", &value)

		// 服务注册操作
		service := registry.ServiceInfo{
			ID:      "bench-service",
			Name:    "bench-service",
			Address: "127.0.0.1",
			Port:    8080,
		}
		coord.Registry().Register(ctx, service, 30*time.Second)
		coord.Registry().Discover(ctx, "bench-service")
		coord.Registry().Unregister(ctx, "bench-service")
	}
}
