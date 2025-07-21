package test

import (
	"fmt"
	"log"
	"sync"
	"testing"
	"time"

	"github.com/ceyewan/gochat/im-infra/etcd"
)

// MockLogger 模拟日志器
type MockLogger struct {
	logs []string
	mu   sync.Mutex
}

func (m *MockLogger) Debug(args ...interface{}) { m.log("DEBUG", args...) }
func (m *MockLogger) Info(args ...interface{})  { m.log("INFO", args...) }
func (m *MockLogger) Warn(args ...interface{})  { m.log("WARN", args...) }
func (m *MockLogger) Error(args ...interface{}) { m.log("ERROR", args...) }

func (m *MockLogger) Debugf(format string, args ...interface{}) { m.logf("DEBUG", format, args...) }
func (m *MockLogger) Infof(format string, args ...interface{})  { m.logf("INFO", format, args...) }
func (m *MockLogger) Warnf(format string, args ...interface{})  { m.logf("WARN", format, args...) }
func (m *MockLogger) Errorf(format string, args ...interface{}) { m.logf("ERROR", format, args...) }

func (m *MockLogger) log(level string, args ...interface{}) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.logs = append(m.logs, fmt.Sprintf("[%s] %v", level, args))
}

func (m *MockLogger) logf(level string, format string, args ...interface{}) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.logs = append(m.logs, fmt.Sprintf("[%s] %s", level, fmt.Sprintf(format, args...)))
}

func (m *MockLogger) GetLogs() []string {
	m.mu.Lock()
	defer m.mu.Unlock()
	return append([]string{}, m.logs...)
}

func (m *MockLogger) Clear() {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.logs = nil
}

// TestSuite 测试套件
type TestSuite struct {
	t      *testing.T
	logger *MockLogger
}

func NewTestSuite(t *testing.T) *TestSuite {
	return &TestSuite{
		t:      t,
		logger: &MockLogger{},
	}
}

// TestManagerBuilder 测试管理器建造者
func TestManagerBuilder(t *testing.T) {
	_ = NewTestSuite(t)

	tests := []struct {
		name    string
		builder func() *etcd.ManagerBuilder
		wantErr bool
	}{
		{
			name: "valid configuration",
			builder: func() *etcd.ManagerBuilder {
				return etcd.NewManagerBuilder().
					WithEndpoints([]string{"localhost:23791", "localhost:23792", "localhost:23793"}).
					WithDialTimeout(5 * time.Second).
					WithDefaultTTL(30)
			},
			wantErr: false,
		},
		{
			name: "empty endpoints",
			builder: func() *etcd.ManagerBuilder {
				return etcd.NewManagerBuilder().
					WithEndpoints([]string{}).
					WithDialTimeout(5 * time.Second)
			},
			wantErr: true,
		},
		{
			name: "invalid timeout",
			builder: func() *etcd.ManagerBuilder {
				return etcd.NewManagerBuilder().
					WithEndpoints([]string{"localhost:23791", "localhost:23792", "localhost:23793"}).
					WithDialTimeout(-1 * time.Second)
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			builder := tt.builder()
			_, err := builder.Build()

			if tt.wantErr && err == nil {
				t.Errorf("Expected error but got none")
			}
			if !tt.wantErr && err != nil {
				t.Errorf("Unexpected error: %v", err)
			}
		})
	}
}

// TestManagerFactory 测试管理器工厂
func TestManagerFactory(t *testing.T) {
	_ = NewTestSuite(t)

	t.Run("create factory", func(t *testing.T) {
		factory := etcd.NewEtcdManagerFactory()
		if factory == nil {
			t.Fatal("Factory should not be nil")
		}

		// 测试默认配置
		defaultOpts := factory.GetDefaultOptions()
		if defaultOpts == nil {
			t.Fatal("Default options should not be nil")
		}

		if len(defaultOpts.Endpoints) == 0 {
			t.Error("Default options should have endpoints")
		}
	})

	t.Run("set default options", func(t *testing.T) {
		factory := etcd.NewEtcdManagerFactory()

		customOpts := &etcd.ManagerOptions{
			Endpoints:   []string{"custom:2379"},
			DialTimeout: 10 * time.Second,
			DefaultTTL:  60,
		}

		err := factory.SetDefaultOptions(customOpts)
		if err != nil {
			t.Fatalf("Failed to set default options: %v", err)
		}

		retrievedOpts := factory.GetDefaultOptions()
		if len(retrievedOpts.Endpoints) != 1 || retrievedOpts.Endpoints[0] != "custom:2379" {
			t.Error("Custom endpoints not set correctly")
		}
	})

	t.Run("invalid default options", func(t *testing.T) {
		factory := etcd.NewEtcdManagerFactory()

		invalidOpts := &etcd.ManagerOptions{
			Endpoints:   []string{}, // 空端点
			DialTimeout: 5 * time.Second,
		}

		err := factory.SetDefaultOptions(invalidOpts)
		if err == nil {
			t.Error("Should return error for invalid options")
		}
	})
}

// TestServiceRegistryOptions 测试服务注册选项
func TestServiceRegistryOptions(t *testing.T) {
	_ = NewTestSuite(t)

	t.Run("register options", func(t *testing.T) {
		opts := &etcd.RegisterOptions{}

		// 测试 WithTTL
		etcd.WithTTL(60)(opts)
		if opts.TTL != 60 {
			t.Errorf("Expected TTL 60, got %d", opts.TTL)
		}

		// 测试 WithMetadata
		metadata := map[string]string{"version": "1.0", "env": "test"}
		etcd.WithMetadata(metadata)(opts)
		if len(opts.Metadata) != 2 {
			t.Errorf("Expected 2 metadata entries, got %d", len(opts.Metadata))
		}
		if opts.Metadata["version"] != "1.0" {
			t.Errorf("Expected version 1.0, got %s", opts.Metadata["version"])
		}
	})
}

// TestServiceDiscoveryOptions 测试服务发现选项
func TestServiceDiscoveryOptions(t *testing.T) {
	_ = NewTestSuite(t)

	t.Run("discovery options", func(t *testing.T) {
		opts := &etcd.DiscoveryOptions{}

		// 测试 WithLoadBalancer
		etcd.WithLoadBalancer("random")(opts)
		if opts.LoadBalancer != "random" {
			t.Errorf("Expected load balancer 'random', got %s", opts.LoadBalancer)
		}

		// 测试 WithDiscoveryTimeout
		etcd.WithDiscoveryTimeout(10 * time.Second)(opts)
		if opts.Timeout != 10*time.Second {
			t.Errorf("Expected timeout 10s, got %v", opts.Timeout)
		}

		// 测试 WithDiscoveryMetadata
		metadata := map[string]string{"region": "us-west"}
		etcd.WithDiscoveryMetadata(metadata)(opts)
		if len(opts.Metadata) != 1 {
			t.Errorf("Expected 1 metadata entry, got %d", len(opts.Metadata))
		}
	})
}

// TestErrorTypes 测试错误类型
func TestErrorTypes(t *testing.T) {
	_ = NewTestSuite(t)

	t.Run("error creation", func(t *testing.T) {
		err := etcd.NewEtcdError(etcd.ErrTypeConnection, 1001, "test error", nil)
		if err.Type != etcd.ErrTypeConnection {
			t.Errorf("Expected error type %s, got %s", etcd.ErrTypeConnection, err.Type)
		}
		if err.Code != 1001 {
			t.Errorf("Expected error code 1001, got %d", err.Code)
		}
	})

	t.Run("error wrapping", func(t *testing.T) {
		originalErr := fmt.Errorf("original error")
		wrappedErr := etcd.WrapConnectionError(originalErr, "wrapped error")

		if !etcd.IsConnectionError(wrappedErr) {
			t.Error("Wrapped error should be connection error")
		}
	})

	t.Run("error checking", func(t *testing.T) {
		if !etcd.IsRetryableError(etcd.ErrConnectionTimeout) {
			t.Error("Connection timeout should be retryable")
		}

		if etcd.IsRetryableError(etcd.ErrInvalidConfiguration) {
			t.Error("Invalid configuration should not be retryable")
		}
	})

	t.Run("retry delay", func(t *testing.T) {
		delay1 := etcd.GetRetryDelay(0)
		delay2 := etcd.GetRetryDelay(1)
		delay3 := etcd.GetRetryDelay(10) // 应该被限制在最大值

		if delay1 != 100 {
			t.Errorf("Expected delay 100ms for attempt 0, got %d", delay1)
		}
		if delay2 != 200 {
			t.Errorf("Expected delay 200ms for attempt 1, got %d", delay2)
		}
		if delay3 != 3200 {
			t.Errorf("Expected delay 3200ms for attempt 10, got %d", delay3)
		}
	})
}

// TestDefaultLogger 测试默认日志器
func TestDefaultLogger(t *testing.T) {
	_ = NewTestSuite(t)

	t.Run("logger methods", func(t *testing.T) {
		logger := &etcd.DefaultLogger{Logger: log.Default()}

		// 测试日志方法不会panic
		logger.Debug("debug message")
		logger.Info("info message")
		logger.Warn("warn message")
		logger.Error("error message")

		logger.Debugf("debug %s", "formatted")
		logger.Infof("info %s", "formatted")
		logger.Warnf("warn %s", "formatted")
		logger.Errorf("error %s", "formatted")
	})
}

// TestConfigValidation 测试配置验证
func TestConfigValidation(t *testing.T) {
	_ = NewTestSuite(t)

	tests := []struct {
		name    string
		options *etcd.ManagerOptions
		wantErr bool
	}{
		{
			name: "valid config",
			options: &etcd.ManagerOptions{
				Endpoints:   []string{"localhost:23791", "localhost:23792", "localhost:23793"},
				DialTimeout: 5 * time.Second,
				DefaultTTL:  30,
			},
			wantErr: false,
		},
		{
			name: "empty endpoints",
			options: &etcd.ManagerOptions{
				Endpoints:   []string{},
				DialTimeout: 5 * time.Second,
			},
			wantErr: true,
		},
		{
			name: "negative timeout",
			options: &etcd.ManagerOptions{
				Endpoints:   []string{"localhost:23791", "localhost:23792", "localhost:23793"},
				DialTimeout: -1 * time.Second,
			},
			wantErr: true,
		},
		{
			name: "zero TTL gets default",
			options: &etcd.ManagerOptions{
				Endpoints:   []string{"localhost:23791", "localhost:23792", "localhost:23793"},
				DialTimeout: 5 * time.Second,
				DefaultTTL:  0,
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			builder := etcd.NewManagerBuilder().
				WithEndpoints(tt.options.Endpoints).
				WithDialTimeout(tt.options.DialTimeout).
				WithDefaultTTL(tt.options.DefaultTTL)
			_, err := builder.Build()

			if tt.wantErr && err == nil {
				t.Error("Expected error but got none")
			}
			if !tt.wantErr && err != nil {
				t.Errorf("Unexpected error: %v", err)
			}
		})
	}
}
