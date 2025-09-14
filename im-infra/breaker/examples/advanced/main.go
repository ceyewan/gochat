package main

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/ceyewan/gochat/im-infra/breaker"
	"github.com/ceyewan/gochat/im-infra/clog"
)

// mockCoordProvider 模拟配置中心，用于演示动态配置更新
type mockCoordProvider struct {
	breakerConfigs map[string]*breaker.Policy
	watchers       map[string][]chan breaker.ConfigEvent[any]
}

func NewMockCoordProvider() *mockCoordProvider {
	return &mockCoordProvider{
		breakerConfigs: make(map[string]*breaker.Policy),
		watchers:       make(map[string][]chan breaker.ConfigEvent[any]),
	}
}

func (m *mockCoordProvider) Get(ctx context.Context, key string, v interface{}) error {
	if policy, exists := m.breakerConfigs[key]; exists {
		if p, ok := v.(*breaker.Policy); ok {
			*p = *policy
			return nil
		}
	}
	return fmt.Errorf("config not found: %s", key)
}

func (m *mockCoordProvider) Set(ctx context.Context, key string, value interface{}) error {
	if policy, ok := value.(*breaker.Policy); ok {
		m.breakerConfigs[key] = policy
		m.notifyWatchers(key, policy, breaker.EventTypePut)
		return nil
	}
	return fmt.Errorf("invalid policy type")
}

func (m *mockCoordProvider) Delete(ctx context.Context, key string) error {
	delete(m.breakerConfigs, key)
	m.notifyWatchers(key, nil, breaker.EventTypeDelete)
	return nil
}

func (m *mockCoordProvider) WatchPrefix(ctx context.Context, prefix string, v interface{}) (breaker.Watcher[any], error) {
	ch := make(chan breaker.ConfigEvent[any], 10)

	// 简化处理：将 watcher 添加到所有匹配前缀的键中
	for key := range m.breakerConfigs {
		if len(key) >= len(prefix) && key[:len(prefix)] == prefix {
			m.watchers[key] = append(m.watchers[key], ch)
		}
	}

	return &mockWatcher{ch: ch}, nil
}

func (m *mockCoordProvider) List(ctx context.Context, prefix string) ([]string, error) {
	var keys []string
	for key := range m.breakerConfigs {
		if len(key) >= len(prefix) && key[:len(prefix)] == prefix {
			keys = append(keys, key)
		}
	}
	return keys, nil
}

func (m *mockCoordProvider) notifyWatchers(key string, value interface{}, eventType breaker.EventType) {
	event := breaker.ConfigEvent[any]{
		Type:  eventType,
		Key:   key,
		Value: value,
	}

	if watchers, exists := m.watchers[key]; exists {
		for _, ch := range watchers {
			select {
			case ch <- event:
			default:
				// 防止阻塞
			}
		}
	}
}

type mockWatcher struct {
	ch chan breaker.ConfigEvent[any]
}

func (m *mockWatcher) Chan() <-chan breaker.ConfigEvent[any] {
	return m.ch
}

func (m *mockWatcher) Close() {
	close(m.ch)
}

// failingService 模拟一个会失败的服务
type failingService struct {
	shouldFail bool
}

func (s *failingService) call() error {
	if s.shouldFail {
		return fmt.Errorf("service failure")
	}
	return nil
}

func main() {
	// 初始化日志器
	logger := clog.Namespace("advanced-breaker-example")

	// 创建模拟配置中心
	coordProvider := NewMockCoordProvider()

	// 设置默认策略
	defaultPolicy := &breaker.Policy{
		FailureThreshold: 3,
		SuccessThreshold: 2,
		OpenStateTimeout: 5 * time.Second,
	}
	coordProvider.Set(context.Background(), "/config/dev/advanced-service/breakers/default.json", defaultPolicy)

	// 创建特定服务的策略
	userServicePolicy := &breaker.Policy{
		FailureThreshold: 2,
		SuccessThreshold: 1,
		OpenStateTimeout: 3 * time.Second,
	}
	coordProvider.Set(context.Background(), "/config/dev/advanced-service/breakers/grpc:user-service.json", userServicePolicy)

	// 创建 breaker 配置
	config := breaker.GetDefaultConfig("advanced-service", "development")

	// 创建 breaker Provider
	breakerProvider, err := breaker.New(context.Background(), config,
		breaker.WithLogger(logger),
		breaker.WithCoordProvider(coordProvider),
	)
	if err != nil {
		log.Fatal(err)
	}
	defer breakerProvider.Close()

	// 获取熔断器
	userServiceBreaker := breakerProvider.GetBreaker("grpc:user-service")
	orderServiceBreaker := breakerProvider.GetBreaker("grpc:order-service")

	service := &failingService{}

	fmt.Println("=== Initial Testing ===")
	testBreaker("User Service", userServiceBreaker, service, 5)
	testBreaker("Order Service", orderServiceBreaker, service, 5)

	fmt.Println("\n=== Dynamic Configuration Update ===")
	// 动态更新用户服务的策略
	newPolicy := &breaker.Policy{
		FailureThreshold: 1, // 更敏感的熔断器
		SuccessThreshold: 1,
		OpenStateTimeout: 2 * time.Second,
	}

	fmt.Println("Updating user service policy to be more sensitive...")
	coordProvider.Set(context.Background(), "/config/dev/advanced-service/breakers/grpc:user-service.json", newPolicy)

	// 等待配置更新
	time.Sleep(100 * time.Millisecond)

	fmt.Println("Testing with updated policy...")
	testBreaker("User Service (Updated)", userServiceBreaker, service, 3)

	fmt.Println("\nAdvanced example completed. Observe how circuit breakers can be dynamically configured.")
}

func testBreaker(name string, br breaker.Breaker, service *failingService, attempts int) {
	fmt.Printf("Testing %s (%d attempts):\n", name, attempts)

	for i := 1; i <= attempts; i++ {
		service.shouldFail = i%2 == 0 // 交替成功和失败

		err := br.Do(context.Background(), func() error {
			return service.call()
		})

		if err != nil {
			if err == breaker.ErrBreakerOpen {
				fmt.Printf("  Attempt %d: ❌ Circuit breaker OPEN\n", i)
			} else {
				fmt.Printf("  Attempt %d: ❌ Call failed: %v\n", i, err)
			}
		} else {
			fmt.Printf("  Attempt %d: ✅ Call succeeded\n", i)
		}

		time.Sleep(200 * time.Millisecond)
	}
}