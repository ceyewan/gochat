package test

import (
	"context"
	"sync"
	"testing"
	"time"

	"myetcd/etcd"
)

// MockConnectionManager 模拟连接管理器
type MockConnectionManager struct {
	connected bool
	client    *MockEtcdClient
	mu        sync.RWMutex
}

func NewMockConnectionManager() *MockConnectionManager {
	return &MockConnectionManager{
		connected: false,
		client:    NewMockEtcdClient(),
	}
}

func (m *MockConnectionManager) Connect(ctx context.Context) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.connected = true
	return nil
}

func (m *MockConnectionManager) Disconnect() error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.connected = false
	return nil
}

func (m *MockConnectionManager) IsConnected() bool {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.connected
}

func (m *MockConnectionManager) HealthCheck(ctx context.Context) error {
	if !m.IsConnected() {
		return etcd.ErrNotConnected
	}
	return nil
}

func (m *MockConnectionManager) GetClient() interface{} {
	return m.client
}

func (m *MockConnectionManager) Reconnect(ctx context.Context) error {
	return m.Connect(ctx)
}

func (m *MockConnectionManager) GetConnectionStatus() etcd.ConnectionStatus {
	return etcd.ConnectionStatus{
		Connected: m.IsConnected(),
		Endpoint:  "mock:2379",
		LastPing:  time.Now(),
	}
}

func (m *MockConnectionManager) Close() error {
	return m.Disconnect()
}

// MockEtcdClient 模拟 etcd 客户端
type MockEtcdClient struct {
	data   map[string]string
	leases map[int64]*MockLease
	mu     sync.RWMutex
}

type MockLease struct {
	ID  int64
	TTL int64
}

func NewMockEtcdClient() *MockEtcdClient {
	return &MockEtcdClient{
		data:   make(map[string]string),
		leases: make(map[int64]*MockLease),
	}
}

func (m *MockEtcdClient) Put(key, value string, leaseID int64) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.data[key] = value
	return nil
}

func (m *MockEtcdClient) Get(key string) (string, bool) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	value, exists := m.data[key]
	return value, exists
}

func (m *MockEtcdClient) Delete(key string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	delete(m.data, key)
	return nil
}

func (m *MockEtcdClient) CreateLease(ttl int64) (int64, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	leaseID := time.Now().UnixNano()
	m.leases[leaseID] = &MockLease{ID: leaseID, TTL: ttl}
	return leaseID, nil
}

func (m *MockEtcdClient) RevokeLease(leaseID int64) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	delete(m.leases, leaseID)
	return nil
}

// MockLeaseManager 模拟租约管理器
type MockLeaseManager struct {
	leases map[int64]*MockLease
	mu     sync.RWMutex
}

func NewMockLeaseManager() *MockLeaseManager {
	return &MockLeaseManager{
		leases: make(map[int64]*MockLease),
	}
}

func (m *MockLeaseManager) CreateLease(ctx context.Context, ttl int64) (int64, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	leaseID := time.Now().UnixNano()
	m.leases[leaseID] = &MockLease{ID: leaseID, TTL: ttl}
	return leaseID, nil
}

func (m *MockLeaseManager) KeepAlive(ctx context.Context, leaseID int64) (<-chan interface{}, error) {
	ch := make(chan interface{}, 1)
	go func() {
		defer close(ch)
		ticker := time.NewTicker(time.Duration(m.leases[leaseID].TTL/3) * time.Second)
		defer ticker.Stop()

		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				ch <- &MockLease{ID: leaseID, TTL: m.leases[leaseID].TTL}
			}
		}
	}()
	return ch, nil
}

func (m *MockLeaseManager) RevokeLease(ctx context.Context, leaseID int64) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	delete(m.leases, leaseID)
	return nil
}

func (m *MockLeaseManager) GetLeaseInfo(ctx context.Context, leaseID int64) (interface{}, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	if lease, exists := m.leases[leaseID]; exists {
		return lease, nil
	}
	return nil, etcd.ErrLeaseNotFound
}

func (m *MockLeaseManager) ListLeases(ctx context.Context) ([]interface{}, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	var leases []interface{}
	for _, lease := range m.leases {
		leases = append(leases, lease)
	}
	return leases, nil
}

func (m *MockLeaseManager) RefreshLease(ctx context.Context, leaseID int64, ttl int64) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if lease, exists := m.leases[leaseID]; exists {
		lease.TTL = ttl
		return nil
	}
	return etcd.ErrLeaseNotFound
}

// TestServiceRegistryIntegration 测试服务注册集成
func TestServiceRegistryIntegration(t *testing.T) {
	_ = NewTestSuite(t)

	t.Run("service lifecycle", func(t *testing.T) {
		// 跳过需要实际 etcd 的测试
		t.Skip("Skipping integration test that requires etcd server")

		manager, err := etcd.NewTestManager()
		if err != nil {
			t.Fatalf("Failed to create test manager: %v", err)
		}
		defer manager.Close()

		ctx := context.Background()
		registry := manager.ServiceRegistry()

		// 注册服务
		err = registry.Register(ctx, "test-service", "instance-1", "localhost:50051",
			etcd.WithTTL(30),
			etcd.WithMetadata(map[string]string{"version": "1.0"}),
		)
		if err != nil {
			t.Fatalf("Failed to register service: %v", err)
		}

		// 列出服务
		services, err := registry.ListServices(ctx)
		if err != nil {
			t.Fatalf("Failed to list services: %v", err)
		}

		found := false
		for _, service := range services {
			if service.Name == "test-service" {
				found = true
				if len(service.Instances) != 1 {
					t.Errorf("Expected 1 instance, got %d", len(service.Instances))
				}
				if service.Instances[0].Address != "localhost:50051" {
					t.Errorf("Expected address localhost:50051, got %s", service.Instances[0].Address)
				}
			}
		}
		if !found {
			t.Error("Service not found in list")
		}

		// 注销服务
		err = registry.Deregister(ctx, "test-service", "instance-1")
		if err != nil {
			t.Fatalf("Failed to deregister service: %v", err)
		}
	})
}

// TestDistributedLockIntegration 测试分布式锁集成
func TestDistributedLockIntegration(t *testing.T) {
	_ = NewTestSuite(t)

	t.Run("lock lifecycle", func(t *testing.T) {
		// 跳过需要实际 etcd 的测试
		t.Skip("Skipping integration test that requires etcd server")

		manager, err := etcd.NewTestManager()
		if err != nil {
			t.Fatalf("Failed to create test manager: %v", err)
		}
		defer manager.Close()

		ctx := context.Background()
		lock := manager.DistributedLock()

		// 尝试获取锁
		acquired, err := lock.TryLock(ctx, "test-lock", 30*time.Second)
		if err != nil {
			t.Fatalf("Failed to try lock: %v", err)
		}
		if !acquired {
			t.Fatal("Should have acquired lock")
		}

		// 检查锁状态
		locked, err := lock.IsLocked(ctx, "test-lock")
		if err != nil {
			t.Fatalf("Failed to check lock status: %v", err)
		}
		if !locked {
			t.Error("Lock should be held")
		}

		// 获取锁信息
		lockInfo, err := lock.GetLockInfo(ctx, "test-lock")
		if err != nil {
			t.Fatalf("Failed to get lock info: %v", err)
		}
		if lockInfo.Key != "test-lock" {
			t.Errorf("Expected lock key 'test-lock', got %s", lockInfo.Key)
		}

		// 释放锁
		err = lock.Unlock(ctx, "test-lock")
		if err != nil {
			t.Fatalf("Failed to unlock: %v", err)
		}

		// 检查锁已释放
		locked, err = lock.IsLocked(ctx, "test-lock")
		if err != nil {
			t.Fatalf("Failed to check lock status after unlock: %v", err)
		}
		if locked {
			t.Error("Lock should be released")
		}
	})

	t.Run("concurrent lock access", func(t *testing.T) {
		// 跳过需要实际 etcd 的测试
		t.Skip("Skipping integration test that requires etcd server")

		manager, err := etcd.NewTestManager()
		if err != nil {
			t.Fatalf("Failed to create test manager: %v", err)
		}
		defer manager.Close()

		ctx := context.Background()
		lock1 := manager.DistributedLock()
		lock2 := manager.DistributedLock()

		// 第一个锁获取成功
		acquired1, err := lock1.TryLock(ctx, "concurrent-lock", 30*time.Second)
		if err != nil {
			t.Fatalf("Failed to try lock1: %v", err)
		}
		if !acquired1 {
			t.Fatal("Lock1 should have acquired lock")
		}

		// 第二个锁获取失败
		acquired2, err := lock2.TryLock(ctx, "concurrent-lock", 30*time.Second)
		if err != nil {
			t.Fatalf("Failed to try lock2: %v", err)
		}
		if acquired2 {
			t.Error("Lock2 should not have acquired lock")
		}

		// 释放第一个锁
		err = lock1.Unlock(ctx, "concurrent-lock")
		if err != nil {
			t.Fatalf("Failed to unlock lock1: %v", err)
		}

		// 现在第二个锁可以获取
		acquired2, err = lock2.TryLock(ctx, "concurrent-lock", 30*time.Second)
		if err != nil {
			t.Fatalf("Failed to try lock2 again: %v", err)
		}
		if !acquired2 {
			t.Error("Lock2 should have acquired lock after lock1 released")
		}

		// 清理
		err = lock2.Unlock(ctx, "concurrent-lock")
		if err != nil {
			t.Fatalf("Failed to unlock lock2: %v", err)
		}
	})
}

// BenchmarkManagerCreation 基准测试管理器创建
func BenchmarkManagerCreation(b *testing.B) {
	for i := 0; i < b.N; i++ {
		builder := etcd.NewManagerBuilder().
			WithEndpoints([]string{"localhost:2379"}).
			WithDialTimeout(5 * time.Second)

		// 只测试建造者，不实际创建连接
		_ = builder
	}
}

// BenchmarkErrorCreation 基准测试错误创建
func BenchmarkErrorCreation(b *testing.B) {
	for i := 0; i < b.N; i++ {
		err := etcd.NewEtcdError(etcd.ErrTypeConnection, 1001, "test error", nil)
		_ = err.Error()
	}
}

// BenchmarkOptionApplication 基准测试选项应用
func BenchmarkOptionApplication(b *testing.B) {
	for i := 0; i < b.N; i++ {
		opts := &etcd.RegisterOptions{}
		etcd.WithTTL(60)(opts)
		etcd.WithMetadata(map[string]string{"version": "1.0"})(opts)
	}
}
