package coord

import (
	"context"
	"testing"
	"time"

	"github.com/ceyewan/gochat/im-infra/coord/registry"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestServiceRegisterAndDiscover 测试服务注册和发现
func TestServiceRegisterAndDiscover(t *testing.T) {
	coord, err := New(DefaultConfig())
	require.NoError(t, err)
	defer coord.Close()

	reg := coord.Registry()
	ctx := context.Background()

	service := registry.ServiceInfo{
		ID:      "test-service-1",
		Name:    "test-service",
		Address: "127.0.0.1",
		Port:    8080,
		Metadata: map[string]string{
			"version": "1.0.0",
			"env":     "test",
		},
	}

	// 注册服务
	err = reg.Register(ctx, service, 30*time.Second)
	require.NoError(t, err)

	// 发现服务
	services, err := reg.Discover(ctx, "test-service")
	require.NoError(t, err)
	assert.Len(t, services, 1)
	assert.Equal(t, service.ID, services[0].ID)
	assert.Equal(t, service.Name, services[0].Name)
	assert.Equal(t, service.Address, services[0].Address)
	assert.Equal(t, service.Port, services[0].Port)

	// 注销服务
	err = reg.Unregister(ctx, "test-service-1")
	require.NoError(t, err)

	// 再次发现应该为空
	services, err = reg.Discover(ctx, "test-service")
	require.NoError(t, err)
	assert.Len(t, services, 0)
}

// TestMultipleServiceInstances 测试多个服务实例
func TestMultipleServiceInstances(t *testing.T) {
	coord, err := New(DefaultConfig())
	require.NoError(t, err)
	defer coord.Close()

	reg := coord.Registry()
	ctx := context.Background()

	serviceName := "multi-service"
	services := []registry.ServiceInfo{
		{
			ID:      "instance-1",
			Name:    serviceName,
			Address: "127.0.0.1",
			Port:    8080,
		},
		{
			ID:      "instance-2",
			Name:    serviceName,
			Address: "127.0.0.1",
			Port:    8081,
		},
		{
			ID:      "instance-3",
			Name:    serviceName,
			Address: "127.0.0.1",
			Port:    8082,
		},
	}

	// 注册所有服务实例
	for _, service := range services {
		err := reg.Register(ctx, service, 30*time.Second)
		require.NoError(t, err)
	}

	// 发现所有实例
	discoveredServices, err := reg.Discover(ctx, serviceName)
	require.NoError(t, err)
	assert.Len(t, discoveredServices, 3)

	// 注销一个实例
	err = reg.Unregister(ctx, "instance-2")
	require.NoError(t, err)

	// 应该只剩下两个实例
	discoveredServices, err = reg.Discover(ctx, serviceName)
	require.NoError(t, err)
	assert.Len(t, discoveredServices, 2)

	// 清理剩余实例
	for _, service := range services {
		if service.ID != "instance-2" {
			reg.Unregister(ctx, service.ID)
		}
	}
}

// TestServiceWatch 测试服务监听
func TestServiceWatch(t *testing.T) {
	coord, err := New(DefaultConfig())
	require.NoError(t, err)
	defer coord.Close()

	reg := coord.Registry()
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	serviceName := "watch-service"

	// 开始监听
	watchCh, err := reg.Watch(ctx, serviceName)
	require.NoError(t, err)

	// 在另一个 goroutine 中注册服务
	go func() {
		time.Sleep(100 * time.Millisecond)
		service := registry.ServiceInfo{
			ID:      "watch-service-1",
			Name:    serviceName,
			Address: "127.0.0.1",
			Port:    9090,
		}
		reg.Register(context.Background(), service, 30*time.Second)

		time.Sleep(100 * time.Millisecond)
		reg.Unregister(context.Background(), "watch-service-1")
	}()

	// 接收事件
	eventCount := 0
	for event := range watchCh {
		eventCount++
		assert.NotEmpty(t, event.Type)
		assert.NotNil(t, event.Service)

		if eventCount >= 2 { // 注册和注销事件
			break
		}
	}

	assert.GreaterOrEqual(t, eventCount, 1)
}

// TestInvalidServiceOperations 测试无效的服务操作
func TestInvalidServiceOperations(t *testing.T) {
	coord, err := New(DefaultConfig())
	require.NoError(t, err)
	defer coord.Close()

	reg := coord.Registry()
	ctx := context.Background()

	// 空服务名注册
	err = reg.Register(ctx, registry.ServiceInfo{
		ID:      "test-id",
		Name:    "",
		Address: "127.0.0.1",
		Port:    8080,
	}, 30*time.Second)
	assert.Error(t, err)

	// 空服务 ID 注册
	err = reg.Register(ctx, registry.ServiceInfo{
		ID:      "",
		Name:    "test-service",
		Address: "127.0.0.1",
		Port:    8080,
	}, 30*time.Second)
	assert.Error(t, err)

	// 空服务名发现
	services, err := reg.Discover(ctx, "")
	assert.Error(t, err)
	assert.Nil(t, services)

	// 空服务 ID 注销
	err = reg.Unregister(ctx, "")
	assert.Error(t, err)
}

// TestServiceTTL 测试服务 TTL
func TestServiceTTL(t *testing.T) {
	coord, err := New(DefaultConfig())
	require.NoError(t, err)
	defer coord.Close()

	reg := coord.Registry()
	ctx := context.Background()

	service := registry.ServiceInfo{
		ID:      "ttl-service-1",
		Name:    "ttl-service",
		Address: "127.0.0.1",
		Port:    8080,
	}

	// 注册服务
	err = reg.Register(ctx, service, 30*time.Second)
	require.NoError(t, err)

	// 立即发现应该能找到
	services, err := reg.Discover(ctx, "ttl-service")
	require.NoError(t, err)
	assert.Len(t, services, 1)

	// 注销服务
	err = reg.Unregister(ctx, "ttl-service-1")
	require.NoError(t, err)

	// 现在应该找不到服务了
	services, err = reg.Discover(ctx, "ttl-service")
	require.NoError(t, err)
	assert.Len(t, services, 0)
}

// TestServiceMetadata 测试服务元数据
func TestServiceMetadata(t *testing.T) {
	coord, err := New(DefaultConfig())
	require.NoError(t, err)
	defer coord.Close()

	reg := coord.Registry()
	ctx := context.Background()

	service := registry.ServiceInfo{
		ID:      "metadata-service-1",
		Name:    "metadata-service",
		Address: "127.0.0.1",
		Port:    8080,
		Metadata: map[string]string{
			"version":     "2.1.0",
			"environment": "production",
			"region":      "us-west-2",
			"team":        "backend",
		},
	}

	// 注册服务
	err = reg.Register(ctx, service, 30*time.Second)
	require.NoError(t, err)

	// 发现服务并验证元数据
	services, err := reg.Discover(ctx, "metadata-service")
	require.NoError(t, err)
	assert.Len(t, services, 1)

	discoveredService := services[0]
	assert.Equal(t, service.Metadata, discoveredService.Metadata)
	assert.Equal(t, "2.1.0", discoveredService.Metadata["version"])
	assert.Equal(t, "production", discoveredService.Metadata["environment"])

	// 清理
	reg.Unregister(ctx, "metadata-service-1")
}

// TestServiceDiscoverNonExistent 测试发现不存在的服务
func TestServiceDiscoverNonExistent(t *testing.T) {
	coord, err := New(DefaultConfig())
	require.NoError(t, err)
	defer coord.Close()

	reg := coord.Registry()
	ctx := context.Background()

	// 发现不存在的服务
	services, err := reg.Discover(ctx, "non-existent-service")
	require.NoError(t, err)
	assert.Len(t, services, 0)
}

// BenchmarkServiceRegister 基准测试服务注册
func BenchmarkServiceRegister(b *testing.B) {
	coord, err := New(DefaultConfig())
	if err != nil {
		b.Fatal(err)
	}
	defer coord.Close()

	reg := coord.Registry()
	ctx := context.Background()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		service := registry.ServiceInfo{
			ID:      "bench-service",
			Name:    "bench-service",
			Address: "127.0.0.1",
			Port:    8080,
		}
		reg.Register(ctx, service, 30*time.Second)
		reg.Unregister(ctx, "bench-service")
	}
}

// BenchmarkServiceDiscover 基准测试服务发现
func BenchmarkServiceDiscover(b *testing.B) {
	coord, err := New(DefaultConfig())
	if err != nil {
		b.Fatal(err)
	}
	defer coord.Close()

	reg := coord.Registry()
	ctx := context.Background()

	// 预先注册一个服务
	service := registry.ServiceInfo{
		ID:      "bench-discover-service",
		Name:    "bench-discover",
		Address: "127.0.0.1",
		Port:    8080,
	}
	reg.Register(ctx, service, 300*time.Second)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		reg.Discover(ctx, "bench-discover")
	}
}
