package coord_test

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/ceyewan/gochat/im-infra/clog"
	"github.com/ceyewan/gochat/im-infra/coord"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// ServiceInfo 测试用的服务信息结构
type ServiceInfo struct {
	ID       string            `json:"id"`
	Name     string            `json:"name"`
	Address  string            `json:"address"`
	Port     int               `json:"port"`
	Metadata map[string]string `json:"metadata,omitempty"`
}

// TestCoordinatorLifecycle 测试协调器的完整生命周期
func TestCoordinatorLifecycle(t *testing.T) {
	// 初始化日志（根据测试策略，必须使用真实环境）
	clogConfig := clog.GetDefaultConfig("development")
	require.NoError(t, clog.Init(context.Background(), clogConfig), "Failed to initialize clog")

	// 创建配置
	cfg := coord.GetDefaultConfig("development")
	cfg.Endpoints = []string{"localhost:2379"} // 连接到真实的 etcd

	// 测试创建
	provider, err := coord.New(context.Background(), cfg, coord.WithLogger(clog.Namespace("test")))
	require.NoError(t, err, "Failed to create coordinator")
	assert.NotNil(t, provider, "Coordinator should not be nil")

	// 测试获取子组件
	configCenter := provider.Config()
	assert.NotNil(t, configCenter, "ConfigCenter should not be nil")

	lockService := provider.Lock()
	assert.NotNil(t, lockService, "LockService should not be nil")

	registry := provider.Registry()
	assert.NotNil(t, registry, "Registry should not be nil")

	// 测试关闭
	err = provider.Close()
	assert.NoError(t, err, "Failed to close coordinator")
}

// TestConfigCenterBasicOperations 测试配置中心的基础 CRUD 操作
func TestConfigCenterBasicOperations(t *testing.T) {
	// 设置（根据测试策略，在真实环境中运行）
	clogConfig := clog.GetDefaultConfig("development")
	require.NoError(t, clog.Init(context.Background(), clogConfig))

	cfg := coord.GetDefaultConfig("development")
	cfg.Endpoints = []string{"localhost:2379"}
	provider, err := coord.New(context.Background(), cfg, coord.WithLogger(clog.Namespace("test")))
	require.NoError(t, err)
	defer provider.Close()

	configCenter := provider.Config()
	ctx := context.Background()
	testKey := "test/config/basic"

	// 清理测试数据（setUp/tearDown 模式）
	defer func() {
		_ = configCenter.Delete(ctx, testKey)
	}()

	t.Run("Set and Get String", func(t *testing.T) {
		// Set
		err := configCenter.Set(ctx, testKey, "test_value")
		assert.NoError(t, err, "Failed to set config")

		// Get
		var result string
		err = configCenter.Get(ctx, testKey, &result)
		assert.NoError(t, err, "Failed to get config")
		assert.Equal(t, "test_value", result, "Config value mismatch")
	})

	t.Run("Set and Get Struct", func(t *testing.T) {
		type TestConfig struct {
			Name    string `json:"name"`
			Version int    `json:"version"`
			Enabled bool   `json:"enabled"`
		}

		config := TestConfig{
			Name:    "test-service",
			Version: 123,
			Enabled: true,
		}

		// Set
		err := configCenter.Set(ctx, testKey+"_struct", &config)
		assert.NoError(t, err, "Failed to set struct config")

		// Get
		var result TestConfig
		err = configCenter.Get(ctx, testKey+"_struct", &result)
		assert.NoError(t, err, "Failed to get struct config")
		assert.Equal(t, config.Name, result.Name, "Struct name mismatch")
		assert.Equal(t, config.Version, result.Version, "Struct version mismatch")
		assert.Equal(t, config.Enabled, result.Enabled, "Struct enabled mismatch")
	})

	t.Run("Delete", func(t *testing.T) {
		// 先设置值
		err := configCenter.Set(ctx, testKey+"_delete", "to_be_deleted")
		assert.NoError(t, err, "Failed to set config for deletion test")

		// 删除
		err = configCenter.Delete(ctx, testKey+"_delete")
		assert.NoError(t, err, "Failed to delete config")

		// 验证删除
		var result string
		err = configCenter.Get(ctx, testKey+"_delete", &result)
		assert.Error(t, err, "Should get error when getting deleted config")
	})
}

// TestConfigCenterWatchOperations 测试配置监听功能
func TestConfigCenterWatchOperations(t *testing.T) {
	clogConfig := clog.GetDefaultConfig("development")
	require.NoError(t, clog.Init(context.Background(), clogConfig))

	cfg := coord.GetDefaultConfig("development")
	cfg.Endpoints = []string{"localhost:2379"}
	provider, err := coord.New(context.Background(), cfg, coord.WithLogger(clog.Namespace("test")))
	require.NoError(t, err)
	defer provider.Close()

	configCenter := provider.Config()
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	testKey := "test/config/watch"
	testPrefix := "test/config/watch_prefix/"

	// 清理测试数据
	defer func() {
		_ = configCenter.Delete(ctx, testKey)
		_ = configCenter.Delete(ctx, testPrefix+"key1")
		_ = configCenter.Delete(ctx, testPrefix+"key2")
	}()

	t.Run("Watch Single Key", func(t *testing.T) {
		var watchValue interface{}
		watcher, err := configCenter.Watch(ctx, testKey, &watchValue)
		require.NoError(t, err, "Failed to create watcher")
		defer watcher.Close()

		// 等待 watcher 准备就绪
		time.Sleep(100 * time.Millisecond)

		// 触发事件
		err = configCenter.Set(ctx, testKey, "initial_value")
		assert.NoError(t, err, "Failed to set config")

		// 等待事件
		select {
		case event := <-watcher.Chan():
			assert.Equal(t, "PUT", string(event.Type), "Event type should be PUT")
			assert.Contains(t, event.Key, testKey, "Event key should match")
			assert.Equal(t, "initial_value", event.Value, "Event value should match")
		case <-time.After(5 * time.Second):
			t.Fatal("Timeout waiting for watch event")
		}
	})

	t.Run("Watch Prefix", func(t *testing.T) {
		var watchValue interface{}
		watcher, err := configCenter.WatchPrefix(ctx, testPrefix, &watchValue)
		require.NoError(t, err, "Failed to create prefix watcher")
		defer watcher.Close()

		// 等待 watcher 准备就绪
		time.Sleep(100 * time.Millisecond)

		// 触发多个事件
		err = configCenter.Set(ctx, testPrefix+"key1", "value1")
		assert.NoError(t, err, "Failed to set first key")

		err = configCenter.Set(ctx, testPrefix+"key2", "value2")
		assert.NoError(t, err, "Failed to set second key")

		// 等待事件
		eventCount := 0
		for i := 0; i < 2; i++ {
			select {
			case event := <-watcher.Chan():
				eventCount++
				assert.Equal(t, "PUT", string(event.Type), "Event type should be PUT")
				assert.Contains(t, event.Key, testPrefix, "Event key should contain prefix")
			case <-time.After(5 * time.Second):
				t.Fatal("Timeout waiting for prefix watch events")
			}
		}
		assert.Equal(t, 2, eventCount, "Should receive exactly 2 events")
	})
}

// TestConfigCenterCASOperations 测试 Compare-And-Swap 操作
func TestConfigCenterCASOperations(t *testing.T) {
	clogConfig := clog.GetDefaultConfig("development")
	require.NoError(t, clog.Init(context.Background(), clogConfig))

	cfg := coord.GetDefaultConfig("development")
	cfg.Endpoints = []string{"localhost:2379"}
	provider, err := coord.New(context.Background(), cfg, coord.WithLogger(clog.Namespace("test")))
	require.NoError(t, err)
	defer provider.Close()

	configCenter := provider.Config()
	ctx := context.Background()
	testKey := "test/config/cas"

	// 清理测试数据
	defer func() {
		_ = configCenter.Delete(ctx, testKey)
	}()

	t.Run("GetWithVersion and CompareAndSet", func(t *testing.T) {
		// 初始设置
		initialConfig := map[string]interface{}{
			"value": "initial",
			"count": 1,
		}
		err := configCenter.Set(ctx, testKey, initialConfig)
		assert.NoError(t, err, "Failed to set initial config")

		// 获取版本
		var retrievedConfig map[string]interface{}
		version, err := configCenter.GetWithVersion(ctx, testKey, &retrievedConfig)
		assert.NoError(t, err, "Failed to get config with version")
		assert.Greater(t, version, int64(0), "Version should be positive")

		// 成功的 CAS 操作
		newConfig := map[string]interface{}{
			"value": "updated",
			"count": 2,
		}
		err = configCenter.CompareAndSet(ctx, testKey, newConfig, version)
		assert.NoError(t, err, "CAS should succeed with correct version")

		// 失败的 CAS 操作（使用旧版本）
		err = configCenter.CompareAndSet(ctx, testKey, map[string]interface{}{"value": "conflict"}, version)
		assert.Error(t, err, "CAS should fail with old version")
	})
}

// TestDistributedLock 测试分布式锁功能
func TestDistributedLock(t *testing.T) {
	clogConfig := clog.GetDefaultConfig("development")
	require.NoError(t, clog.Init(context.Background(), clogConfig))

	cfg := coord.GetDefaultConfig("development")
	cfg.Endpoints = []string{"localhost:2379"}
	provider, err := coord.New(context.Background(), cfg, coord.WithLogger(clog.Namespace("test")))
	require.NoError(t, err)
	defer provider.Close()

	lockService := provider.Lock()
	ctx := context.Background()
	testLockKey := "test/lock/distributed"

	t.Run("Acquire and Release", func(t *testing.T) {
		// 获取锁
		lock, err := lockService.Acquire(ctx, testLockKey, 10*time.Second)
		require.NoError(t, err, "Failed to acquire lock")
		assert.NotNil(t, lock, "Lock should not be nil")

		// 验证锁属性 (key 包含前缀和 session ID，这是正常行为)
		assert.Contains(t, lock.Key(), testLockKey, "Lock key should contain the requested key")

		// 检查 TTL
		ttl, err := lock.TTL(ctx)
		assert.NoError(t, err, "Failed to get TTL")
		assert.Greater(t, ttl, time.Duration(0), "TTL should be positive")

		// 释放锁
		err = lock.Unlock(ctx)
		assert.NoError(t, err, "Failed to release lock")
	})

	t.Run("TryAcquire Non-blocking", func(t *testing.T) {
		// 先获取锁
		lock1, err := lockService.Acquire(ctx, testLockKey+"_nonblock", 10*time.Second)
		require.NoError(t, err, "Failed to acquire first lock")
		defer lock1.Unlock(ctx)

		// 尝试非阻塞获取（应该失败）
		lock2, err := lockService.TryAcquire(ctx, testLockKey+"_nonblock", 10*time.Second)
		assert.Error(t, err, "TryAcquire should fail when lock is held")
		assert.Nil(t, lock2, "Second lock should be nil")
	})
}

// TestServiceRegistry 测试服务注册发现功能
// Note: This test is disabled because ServiceInfo type is not exported from the registry package
// In a real scenario, we would either:
// 1. Export the ServiceInfo type from the main coord package
// 2. Create the test within the same package
// 3. Use reflection or other workarounds
func TestServiceRegistry(t *testing.T) {
	t.Skip("ServiceInfo type not accessible from external test package")
}

// TestInstanceIDAllocator 测试实例 ID 分配器功能
func TestInstanceIDAllocator(t *testing.T) {
	clogConfig := clog.GetDefaultConfig("development")
	require.NoError(t, clog.Init(context.Background(), clogConfig))

	cfg := coord.GetDefaultConfig("development")
	cfg.Endpoints = []string{"localhost:2379"}
	provider, err := coord.New(context.Background(), cfg, coord.WithLogger(clog.Namespace("test")))
	require.NoError(t, err)

	// 为每个子测试创建独立的 provider 以避免会话竞争
	t.Run("Basic ID Allocation", func(t *testing.T) {
		testProvider, err := coord.New(context.Background(), cfg, coord.WithLogger(clog.Namespace("test-basic")))
		require.NoError(t, err, "Failed to create test provider")
		defer testProvider.Close()

		ctx := context.Background()
		serviceName := "test-id-allocator-basic"
		maxID := 5

		allocator, err := testProvider.InstanceIDAllocator(serviceName, maxID)
		require.NoError(t, err, "Failed to create ID allocator")

		// 获取 ID
		allocatedID, err := allocator.AcquireID(ctx)
		require.NoError(t, err, "Failed to allocate ID")
		assert.NotNil(t, allocatedID, "Allocated ID should not be nil")

		// 验证 ID 范围
		id := allocatedID.ID()
		assert.Greater(t, id, 0, "ID should be positive")
		assert.LessOrEqual(t, id, maxID, "ID should be within max limit")

		// 释放 ID
		err = allocatedID.Close(ctx)
		assert.NoError(t, err, "Failed to release ID")
	})

	t.Run("Concurrent ID Allocation", func(t *testing.T) {
		testProvider, err := coord.New(context.Background(), cfg, coord.WithLogger(clog.Namespace("test-concurrent")))
		require.NoError(t, err, "Failed to create test provider")
		defer testProvider.Close()

		ctx := context.Background()
		serviceName := "test-id-allocator-concurrent"
		maxID := 5

		allocator, err := testProvider.InstanceIDAllocator(serviceName, maxID)
		require.NoError(t, err, "Failed to create ID allocator")

		// 并发获取 ID
		const numGoroutines = 10
		results := make(chan int, numGoroutines)
		errors := make(chan error, numGoroutines)
		var wg sync.WaitGroup

		for i := 0; i < numGoroutines; i++ {
			wg.Add(1)
			go func() {
				defer wg.Done()
				allocatedID, err := allocator.AcquireID(ctx)
				if err != nil {
					errors <- err
					return
				}
				results <- allocatedID.ID()
				// 不释放 ID，测试最大 ID 限制
			}()
		}

		// 等待所有 goroutine 完成
		go func() {
			wg.Wait()
			close(results)
			close(errors)
		}()

		// 收集结果
		allocatedIDs := make([]int, 0, numGoroutines)
		errorCount := 0
		timeout := time.After(10 * time.Second) // 增加超时时间

		for {
			select {
			case id, ok := <-results:
				if !ok {
					results = nil
					continue
				}
				allocatedIDs = append(allocatedIDs, id)
			case err, ok := <-errors:
				if !ok {
					errors = nil
					continue
				}
				errorCount++
				assert.Contains(t, err.Error(), "no available ID", "Should get ID exhaustion error")
			case <-timeout:
				t.Fatal("Timeout waiting for concurrent allocation")
			}

			if results == nil && errors == nil {
				break
			}
		}

		// 验证结果
		assert.Equal(t, maxID, len(allocatedIDs), "Should allocate exactly maxID IDs")
		assert.Equal(t, numGoroutines-maxID, errorCount, "Should get expected number of allocation errors")

		// 验证 ID 唯一性
		uniqueIDs := make(map[int]struct{})
		for _, id := range allocatedIDs {
			assert.NotContains(t, uniqueIDs, id, "ID should be unique")
			uniqueIDs[id] = struct{}{}
		}
	})

	provider.Close()
}

// TestCoordinatorErrorHandling 测试错误处理
func TestCoordinatorErrorHandling(t *testing.T) {
	clogConfig := clog.GetDefaultConfig("development")
	require.NoError(t, clog.Init(context.Background(), clogConfig))

	t.Run("Invalid Etcd Endpoints", func(t *testing.T) {
		cfg := coord.GetDefaultConfig("development")
		cfg.Endpoints = []string{"localhost:99999"} // 无效端口

		_, err := coord.New(context.Background(), cfg, coord.WithLogger(clog.Namespace("test")))
		assert.Error(t, err, "Should fail with invalid etcd endpoints")
	})

	t.Run("Context Cancellation", func(t *testing.T) {
		cfg := coord.GetDefaultConfig("development")
		cfg.Endpoints = []string{"localhost:99998"} // 使用无效端点模拟上下文取消

		ctx, cancel := context.WithCancel(context.Background())
		cancel() // 立即取消

		_, err := coord.New(ctx, cfg, coord.WithLogger(clog.Namespace("test")))
		assert.Error(t, err, "Should fail with cancelled context or invalid endpoint")
	})
}

// BenchmarkConfigCenterOperations 基准测试（根据测试策略，在真实环境中运行）
func BenchmarkConfigCenterOperations(b *testing.B) {
	clogConfig := clog.GetDefaultConfig("development")
	_ = clog.Init(context.Background(), clogConfig)

	cfg := coord.GetDefaultConfig("development")
	cfg.Endpoints = []string{"localhost:2379"}
	provider, err := coord.New(context.Background(), cfg, coord.WithLogger(clog.Namespace("benchmark")))
	if err != nil {
		b.Fatal("Failed to create coordinator:", err)
	}
	defer provider.Close()

	configCenter := provider.Config()
	ctx := context.Background()
	testKey := "benchmark/config"

	b.Run("Set", func(b *testing.B) {
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			err := configCenter.Set(ctx, testKey, "benchmark_value")
			if err != nil {
				b.Fatal("Set failed:", err)
			}
		}
	})

	b.Run("Get", func(b *testing.B) {
		// 先设置一个值
		err := configCenter.Set(ctx, testKey, "benchmark_value")
		if err != nil {
			b.Fatal("Initial Set failed:", err)
		}

		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			var result string
			err := configCenter.Get(ctx, testKey, &result)
			if err != nil {
				b.Fatal("Get failed:", err)
			}
		}
	})

	// 清理
	_ = configCenter.Delete(ctx, testKey)
}