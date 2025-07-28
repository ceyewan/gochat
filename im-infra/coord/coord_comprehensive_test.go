package coord

import (
	"context"
	"net"
	"testing"
	"time"

	"github.com/ceyewan/gochat/im-infra/coord/registry"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"
	"google.golang.org/grpc/health"
	"google.golang.org/grpc/health/grpc_health_v1"
)

// TestCoordComprehensive 综合测试所有核心功能
func TestCoordComprehensive(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping comprehensive test in short mode")
	}

	// 创建协调器
	config := CoordinatorConfig{
		Endpoints: []string{"localhost:23791"},
		Timeout:   5 * time.Second,
	}
	coordinator, err := New(config)
	require.NoError(t, err)
	defer coordinator.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// 测试分布式锁
	t.Run("DistributedLock", func(t *testing.T) {
		lockKey := "test-lock"

		// 获取锁
		lock, err := coordinator.Lock().Acquire(ctx, lockKey, 10*time.Second)
		require.NoError(t, err)
		assert.Equal(t, lockKey, lock.Key())

		// 检查 TTL
		ttl, err := lock.TTL(ctx)
		require.NoError(t, err)
		assert.Greater(t, ttl, 5*time.Second)

		// 释放锁
		err = lock.Unlock(ctx)
		assert.NoError(t, err)
	})

	// 测试配置中心
	t.Run("ConfigCenter", func(t *testing.T) {
		configKey := "app/test-config"
		configValue := "test-value"

		// 设置配置
		err := coordinator.Config().Set(ctx, configKey, configValue)
		require.NoError(t, err)

		// 获取配置
		var result string
		err = coordinator.Config().Get(ctx, configKey, &result)
		require.NoError(t, err)
		assert.Equal(t, configValue, result)

		// 删除配置
		err = coordinator.Config().Delete(ctx, configKey)
		assert.NoError(t, err)
	})

	// 测试服务注册发现
	t.Run("ServiceRegistry", func(t *testing.T) {
		serviceName := "test-service"
		service := registry.ServiceInfo{
			ID:      "test-service-1",
			Name:    serviceName,
			Address: "127.0.0.1",
			Port:    8080,
		}

		// 注册服务
		err := coordinator.Registry().Register(ctx, service, 30*time.Second)
		require.NoError(t, err)
		defer coordinator.Registry().Unregister(ctx, service.ID)

		// 发现服务
		services, err := coordinator.Registry().Discover(ctx, serviceName)
		require.NoError(t, err)
		assert.Len(t, services, 1)
		assert.Equal(t, service.ID, services[0].ID)
	})

	// 测试 gRPC 动态服务发现
	t.Run("GRPCDynamicServiceDiscovery", func(t *testing.T) {
		// 创建测试 gRPC 服务器
		server, addr := createTestServer(t)
		defer server.Stop()

		serviceName := "grpc-test-service"
		service := registry.ServiceInfo{
			ID:      "grpc-test-1",
			Name:    serviceName,
			Address: addr.IP.String(),
			Port:    addr.Port,
		}

		// 注册服务
		err := coordinator.Registry().Register(ctx, service, 30*time.Second)
		require.NoError(t, err)
		defer coordinator.Registry().Unregister(ctx, service.ID)

		// 等待服务注册生效
		time.Sleep(200 * time.Millisecond)

		// 使用 gRPC resolver 创建连接
		conn, err := coordinator.Registry().GetConnection(ctx, serviceName)
		require.NoError(t, err)
		defer conn.Close()

		// 验证连接可用
		client := grpc_health_v1.NewHealthClient(conn)
		resp, err := client.Check(ctx, &grpc_health_v1.HealthCheckRequest{})
		require.NoError(t, err)
		assert.Equal(t, grpc_health_v1.HealthCheckResponse_SERVING, resp.Status)
	})
}

// TestCoordBasic 基本功能测试（快速测试）
func TestCoordBasic(t *testing.T) {
	config := CoordinatorConfig{
		Endpoints: []string{"localhost:23791"},
		Timeout:   5 * time.Second,
	}
	coordinator, err := New(config)
	require.NoError(t, err)
	defer coordinator.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// 基本锁测试
	lock, err := coordinator.Lock().TryAcquire(ctx, "basic-lock", 5*time.Second)
	if err == nil {
		assert.NotNil(t, lock)
		lock.Unlock(ctx)
	}

	// 基本配置测试
	err = coordinator.Config().Set(ctx, "basic/test", "value")
	if err == nil {
		var result string
		coordinator.Config().Get(ctx, "basic/test", &result)
		assert.Equal(t, "value", result)
	}
}

// TestCoordErrors 错误处理测试
func TestCoordErrors(t *testing.T) {
	config := CoordinatorConfig{
		Endpoints: []string{"localhost:23791"},
		Timeout:   5 * time.Second,
	}
	coordinator, err := New(config)
	require.NoError(t, err)
	defer coordinator.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// 测试空服务名
	_, err = coordinator.Registry().GetConnection(ctx, "")
	assert.Error(t, err)

	// 测试不存在的服务
	ctx2, cancel2 := context.WithTimeout(context.Background(), 1*time.Second)
	defer cancel2()
	_, err = coordinator.Registry().GetConnection(ctx2, "non-existent-service")
	assert.Error(t, err)
}

// createTestServer 创建测试用的 gRPC 服务器
func createTestServer(t *testing.T) (*grpc.Server, *net.TCPAddr) {
	lis, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err)

	server := grpc.NewServer()
	healthServer := health.NewServer()
	grpc_health_v1.RegisterHealthServer(server, healthServer)
	healthServer.SetServingStatus("", grpc_health_v1.HealthCheckResponse_SERVING)

	go func() {
		if err := server.Serve(lis); err != nil {
			t.Logf("Server exited: %v", err)
		}
	}()

	time.Sleep(100 * time.Millisecond)
	return server, lis.Addr().(*net.TCPAddr)
}

// TestCoordConcurrency 并发测试
func TestCoordConcurrency(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping concurrency test in short mode")
	}

	config := CoordinatorConfig{
		Endpoints: []string{"localhost:23791"},
		Timeout:   5 * time.Second,
	}
	coordinator, err := New(config)
	require.NoError(t, err)
	defer coordinator.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// 并发锁测试
	t.Run("ConcurrentLocks", func(t *testing.T) {
		const numGoroutines = 5
		results := make(chan bool, numGoroutines)

		for i := 0; i < numGoroutines; i++ {
			go func(id int) {
				lock, err := coordinator.Lock().TryAcquire(ctx, "concurrent-lock", 2*time.Second)
				if err == nil && lock != nil {
					time.Sleep(100 * time.Millisecond)
					lock.Unlock(ctx)
					results <- true
				} else {
					results <- false
				}
			}(i)
		}

		// 至少应该有一个成功获取锁
		successCount := 0
		for i := 0; i < numGoroutines; i++ {
			if <-results {
				successCount++
			}
		}
		assert.Greater(t, successCount, 0)
	})
}

// TestCoordPerformance 性能测试
func TestCoordPerformance(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping performance test in short mode")
	}

	config := CoordinatorConfig{
		Endpoints: []string{"localhost:23791"},
		Timeout:   5 * time.Second,
	}
	coordinator, err := New(config)
	require.NoError(t, err)
	defer coordinator.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// 配置操作性能测试
	t.Run("ConfigPerformance", func(t *testing.T) {
		start := time.Now()
		for i := 0; i < 50; i++ {
			key := "perf/test"
			coordinator.Config().Set(ctx, key, "value")
			var result string
			coordinator.Config().Get(ctx, key, &result)
		}
		duration := time.Since(start)
		t.Logf("50 config operations took: %v", duration)
		assert.Less(t, duration, 5*time.Second)
	})
}
