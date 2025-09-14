package breaker

import (
	"context"
	"errors"
	"fmt"
	"testing"
	"time"

	"github.com/ceyewan/gochat/im-infra/clog"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

// mockLogger 是一个用于测试的日志器实现
type mockLogger struct {
	debugMessages []string
	infoMessages  []string
	warnMessages  []string
	errorMessages []string
	fatalMessages []string
}

func (m *mockLogger) Debug(msg string, fields ...Field) {
	m.debugMessages = append(m.debugMessages, msg)
}

func (m *mockLogger) Info(msg string, fields ...Field) {
	m.infoMessages = append(m.infoMessages, msg)
}

func (m *mockLogger) Warn(msg string, fields ...Field) {
	m.warnMessages = append(m.warnMessages, msg)
}

func (m *mockLogger) Error(msg string, fields ...Field) {
	m.errorMessages = append(m.errorMessages, msg)
}

func (m *mockLogger) Fatal(msg string, fields ...Field) {
	m.fatalMessages = append(m.fatalMessages, msg)
}

func (m *mockLogger) With(fields ...Field) Logger {
	return m
}

func (m *mockLogger) WithOptions(opts ...zap.Option) clog.Logger {
	return m
}

func (m *mockLogger) Namespace(name string) clog.Logger {
	return m
}

// mockCoordProvider 是一个用于测试的配置中心实现
type mockCoordProvider struct {
	configs map[string][]byte
	watcher chan ConfigEvent[any]
}

func (m *mockCoordProvider) Get(ctx context.Context, key string, v interface{}) error {
	if _, exists := m.configs[key]; exists {
		// 简化处理，直接赋值
		if policy, ok := v.(*Policy); ok {
			*policy = Policy{
				FailureThreshold: 3,
				SuccessThreshold: 1,
				OpenStateTimeout: time.Second * 30,
			}
		}
		return nil
	}
	return errors.New("config not found")
}

func (m *mockCoordProvider) Set(ctx context.Context, key string, value interface{}) error {
	return nil
}

func (m *mockCoordProvider) Delete(ctx context.Context, key string) error {
	return nil
}

func (m *mockCoordProvider) WatchPrefix(ctx context.Context, prefix string, v interface{}) (Watcher[any], error) {
	return &mockWatcher{ch: m.watcher}, nil
}

func (m *mockCoordProvider) List(ctx context.Context, prefix string) ([]string, error) {
	var keys []string
	for key := range m.configs {
		if len(key) >= len(prefix) && key[:len(prefix)] == prefix {
			keys = append(keys, key)
		}
	}
	return keys, nil
}

type mockWatcher struct {
	ch chan ConfigEvent[any]
}

func (m *mockWatcher) Chan() <-chan ConfigEvent[any] {
	return m.ch
}

func (m *mockWatcher) Close() {
	close(m.ch)
}

func TestGetDefaultConfig(t *testing.T) {
	tests := []struct {
		name        string
		serviceName string
		env         string
		expected    string
	}{
		{
			name:        "development environment",
			serviceName: "test-service",
			env:         "development",
			expected:    "/config/dev/test-service/breakers/",
		},
		{
			name:        "production environment",
			serviceName: "test-service",
			env:         "production",
			expected:    "/config/prod/test-service/breakers/",
		},
		{
			name:        "custom environment",
			serviceName: "custom-service",
			env:         "staging",
			expected:    "/config/dev/custom-service/breakers/",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			config := GetDefaultConfig(tt.serviceName, tt.env)
			assert.Equal(t, tt.serviceName, config.ServiceName)
			assert.Equal(t, tt.expected, config.PoliciesPath)
		})
	}
}

func TestGetDefaultPolicy(t *testing.T) {
	policy := GetDefaultPolicy()
	assert.Equal(t, 5, policy.FailureThreshold)
	assert.Equal(t, 2, policy.SuccessThreshold)
	assert.Equal(t, time.Minute, policy.OpenStateTimeout)
}

func TestNewProvider(t *testing.T) {
	logger := &mockLogger{}
	config := GetDefaultConfig("test-service", "development")

	provider, err := New(context.Background(), config, WithLogger(logger))
	require.NoError(t, err)
	assert.NotNil(t, provider)

	defer provider.Close()
}

func TestProviderGetBreaker(t *testing.T) {
	// 初始化 clog 避免默认 logger 问题
	clog.Init(context.Background(), &clog.Config{
		Level:       "debug",
		Format:      "console",
		Output:      "stdout",
		EnableColor: false,
	})

	logger := &mockLogger{}
	config := GetDefaultConfig("test-service", "development")

	provider, err := New(context.Background(), config, WithLogger(logger))
	require.NoError(t, err)
	defer provider.Close()

	// 获取熔断器
	fmt.Println("About to call GetBreaker...")
	breaker1 := provider.GetBreaker("test-breaker")
	fmt.Println("GetBreaker returned")
	assert.NotNil(t, breaker1)

	// 再次获取相同的熔断器，应该返回同一个实例
	breaker2 := provider.GetBreaker("test-breaker")
	assert.Same(t, breaker1, breaker2)

	// 获取不同的熔断器
	breaker3 := provider.GetBreaker("another-breaker")
	assert.NotNil(t, breaker3)
	assert.NotSame(t, breaker1, breaker3)
}

func TestBreakerDo(t *testing.T) {
	logger := &mockLogger{}
	config := GetDefaultConfig("test-service", "development")

	provider, err := New(context.Background(), config, WithLogger(logger))
	require.NoError(t, err)
	defer provider.Close()

	breaker := provider.GetBreaker("test-breaker")

	// 测试成功操作
	err = breaker.Do(context.Background(), func() error {
		return nil
	})
	assert.NoError(t, err)

	// 测试失败操作
	err = breaker.Do(context.Background(), func() error {
		return errors.New("operation failed")
	})
	assert.Error(t, err)
	assert.Equal(t, "operation failed", err.Error())
}

func TestBreakerWithCoordProvider(t *testing.T) {
	logger := &mockLogger{}
	config := GetDefaultConfig("test-service", "development")

	// 创建模拟的配置中心
	mockCoord := &mockCoordProvider{
		configs: make(map[string][]byte),
		watcher: make(chan ConfigEvent[any], 1),
	}

	provider, err := New(context.Background(), config,
		WithLogger(logger),
		WithCoordProvider(mockCoord))
	require.NoError(t, err)
	defer provider.Close()

	// 测试在没有特定策略时使用默认策略
	breaker := provider.GetBreaker("test-breaker")
	assert.NotNil(t, breaker)
}

func TestProviderClose(t *testing.T) {
	logger := &mockLogger{}
	config := GetDefaultConfig("test-service", "development")

	provider, err := New(context.Background(), config, WithLogger(logger))
	require.NoError(t, err)

	// 获取一些熔断器
	provider.GetBreaker("breaker1")
	provider.GetBreaker("breaker2")

	// 关闭 provider
	err = provider.Close()
	assert.NoError(t, err)

	// 关闭后再获取熔断器应该返回 noop breaker
	breaker := provider.GetBreaker("breaker3")
	assert.NotNil(t, breaker)

	// noop breaker 应该直接执行操作而不进行熔断
	err = breaker.Do(context.Background(), func() error {
		return nil
	})
	assert.NoError(t, err)
}

func TestProviderWithoutCoord(t *testing.T) {
	logger := &mockLogger{}
	config := GetDefaultConfig("test-service", "development")

	// 不提供配置中心
	provider, err := New(context.Background(), config, WithLogger(logger))
	require.NoError(t, err)
	defer provider.Close()

	breaker := provider.GetBreaker("test-breaker")
	assert.NotNil(t, breaker)

	// 应该能正常工作，使用默认策略
	err = breaker.Do(context.Background(), func() error {
		return nil
	})
	assert.NoError(t, err)
}

func TestBreakerStateChanges(t *testing.T) {
	logger := &mockLogger{}

	// 创建一个自定义配置，使用更短的 timeout 和更低的阈值
	config := &Config{
		ServiceName:  "test-service",
		PoliciesPath: "/config/dev/test-service/breakers/",
	}

	provider, err := New(context.Background(), config, WithLogger(logger))
	require.NoError(t, err)
	defer provider.Close()

	breaker := provider.GetBreaker("state-test-breaker")

	// 连续失败多次以触发熔断器打开
	for i := 0; i < 10; i++ {
		err := breaker.Do(context.Background(), func() error {
			return errors.New("continuous failure")
		})
		t.Logf("Attempt %d: err = %v", i, err)

		if i < 5 { // 前5次应该通过（FailureThreshold=5）
			assert.Error(t, err)
			assert.Equal(t, "continuous failure", err.Error())
		} else { // 第6次及以后应该返回熔断器打开错误
			assert.Error(t, err)
			assert.Contains(t, err.Error(), "circuit breaker is open")
		}

		// 添加小延迟，确保熔断器有时间处理状态变化
		time.Sleep(10 * time.Millisecond)
	}
}
