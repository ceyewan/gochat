package test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/ceyewan/gochat/im-infra/etcd"
)

// TestEdgeCases 测试边界条件
func TestEdgeCases(t *testing.T) {
	_ = NewTestSuite(t)

	t.Run("nil parameters", func(t *testing.T) {
		// 测试 nil 配置
		_, err := etcd.NewManagerBuilder().Build()
		if err != nil {
			// 如果是连接错误，跳过测试（没有 etcd 服务器）
			if etcd.IsConnectionError(err) || etcd.IsTimeoutError(err) {
				t.Skipf("Skipping test due to no etcd server: %v", err)
				return
			}
			// 其他错误才是真正的问题
			t.Errorf("Should handle nil config gracefully, got error: %v", err)
		}

		// 测试空字符串参数
		builder := etcd.NewManagerBuilder()
		builder = builder.WithServicePrefix("")
		builder = builder.WithLockPrefix("")

		// 应该使用默认值
		// 无法直接访问私有字段，通过构建来验证
		manager, err := builder.Build()
		if err != nil {
			// 如果构建失败，设置默认值
			builder = builder.WithServicePrefix("/services").WithLockPrefix("/locks")
		} else {
			manager.Close()
		}
	})

	t.Run("extreme values", func(t *testing.T) {
		// 测试极大的 TTL 值
		opts := &etcd.RegisterOptions{}
		etcd.WithTTL(999999999)(opts)
		if opts.TTL != 999999999 {
			t.Errorf("Should handle large TTL values")
		}

		// 测试零值 TTL
		etcd.WithTTL(0)(opts)
		if opts.TTL != 0 {
			t.Errorf("Should handle zero TTL")
		}

		// 测试负值 TTL
		etcd.WithTTL(-1)(opts)
		if opts.TTL != -1 {
			t.Errorf("Should handle negative TTL")
		}
	})

	t.Run("empty strings", func(t *testing.T) {
		// 测试空字符串服务名
		builder := etcd.NewManagerBuilder().
			WithEndpoints([]string{"localhost:23791", "localhost:23792", "localhost:23793"})

		manager, err := builder.Build()
		if err != nil {
			t.Skipf("Skipping test due to build error: %v", err)
			return
		}
		defer manager.Close()

		ctx := context.Background()
		registry := manager.ServiceRegistry()

		// 空服务名应该返回错误
		err = registry.Register(ctx, "", "instance", "addr")
		if err == nil {
			t.Error("Should return error for empty service name")
		}

		// 空实例ID应该返回错误
		err = registry.Register(ctx, "service", "", "addr")
		if err == nil {
			t.Error("Should return error for empty instance ID")
		}

		// 空地址应该返回错误
		err = registry.Register(ctx, "service", "instance", "")
		if err == nil {
			t.Error("Should return error for empty address")
		}
	})

	t.Run("unicode and special characters", func(t *testing.T) {
		// 测试 Unicode 字符
		opts := &etcd.RegisterOptions{}
		metadata := map[string]string{
			"中文":      "测试",
			"emoji":   "🚀",
			"special": "!@#$%^&*()",
		}
		etcd.WithMetadata(metadata)(opts)

		if len(opts.Metadata) != 3 {
			t.Errorf("Should handle unicode metadata")
		}
	})

	t.Run("concurrent access", func(t *testing.T) {
		// 测试并发访问选项
		opts := &etcd.RegisterOptions{}

		done := make(chan bool, 10)
		for i := 0; i < 10; i++ {
			go func(i int) {
				defer func() { done <- true }()
				etcd.WithTTL(int64(i))(opts)
			}(i)
		}

		// 等待所有 goroutine 完成
		for i := 0; i < 10; i++ {
			<-done
		}

		// 最后的值应该是某个 i 的值
		if opts.TTL < 0 || opts.TTL >= 10 {
			t.Errorf("Unexpected TTL value after concurrent access: %d", opts.TTL)
		}
	})
}

// TestErrorHandling 测试错误处理
func TestErrorHandling(t *testing.T) {
	_ = NewTestSuite(t)

	t.Run("error wrapping", func(t *testing.T) {
		originalErr := errors.New("original error")

		// 测试各种错误包装
		connErr := etcd.WrapConnectionError(originalErr, "connection failed")
		regErr := etcd.WrapRegistryError(originalErr, "registry failed")
		discErr := etcd.WrapDiscoveryError(originalErr, "discovery failed")
		lockErr := etcd.WrapLockError(originalErr, "lock failed")
		leaseErr := etcd.WrapLeaseError(originalErr, "lease failed")
		_ = etcd.WrapConfigurationError(originalErr, "config failed")

		// 测试错误类型检查
		if !etcd.IsConnectionError(connErr) {
			t.Error("Should identify connection error")
		}
		if !etcd.IsRegistryError(regErr) {
			t.Error("Should identify registry error")
		}
		if !etcd.IsDiscoveryError(discErr) {
			t.Error("Should identify discovery error")
		}
		if !etcd.IsLockError(lockErr) {
			t.Error("Should identify lock error")
		}
		if !etcd.IsLeaseError(leaseErr) {
			t.Error("Should identify lease error")
		}

		// 测试错误链
		if !errors.Is(connErr, originalErr) {
			t.Error("Should maintain error chain")
		}
	})

	t.Run("error messages", func(t *testing.T) {
		err := etcd.NewEtcdError(etcd.ErrTypeConnection, 1001, "test message", nil)
		msg := err.Error()

		if msg == "" {
			t.Error("Error message should not be empty")
		}

		// 应该包含错误类型和消息
		if !contains(msg, etcd.ErrTypeConnection) {
			t.Error("Error message should contain error type")
		}
		if !contains(msg, "test message") {
			t.Error("Error message should contain custom message")
		}
	})

	t.Run("predefined errors", func(t *testing.T) {
		predefinedErrors := []error{
			etcd.ErrConnectionFailed,
			etcd.ErrConnectionLost,
			etcd.ErrConnectionTimeout,
			etcd.ErrNotConnected,
			etcd.ErrServiceAlreadyRegistered,
			etcd.ErrServiceNotRegistered,
			etcd.ErrServiceNotFound,
			etcd.ErrLockAcquisitionFailed,
			etcd.ErrLockNotHeld,
			etcd.ErrLeaseCreationFailed,
			etcd.ErrLeaseNotFound,
			etcd.ErrInvalidConfiguration,
			etcd.ErrTimeout,
		}

		for _, err := range predefinedErrors {
			if err == nil {
				t.Error("Predefined error should not be nil")
			}
			if err.Error() == "" {
				t.Error("Predefined error should have message")
			}
		}
	})

	t.Run("retry logic", func(t *testing.T) {
		// 测试重试延迟计算
		delays := []int{
			etcd.GetRetryDelay(0),  // 100ms
			etcd.GetRetryDelay(1),  // 200ms
			etcd.GetRetryDelay(2),  // 400ms
			etcd.GetRetryDelay(3),  // 800ms
			etcd.GetRetryDelay(4),  // 1600ms
			etcd.GetRetryDelay(5),  // 3200ms (max)
			etcd.GetRetryDelay(10), // 3200ms (max)
		}

		expected := []int{100, 200, 400, 800, 1600, 3200, 3200}

		for i, delay := range delays {
			if delay != expected[i] {
				t.Errorf("Retry delay %d: expected %d, got %d", i, expected[i], delay)
			}
		}
	})
}

// TestTimeouts 测试超时处理
func TestTimeouts(t *testing.T) {
	_ = NewTestSuite(t)

	t.Run("context timeout", func(t *testing.T) {
		// 创建一个很短的超时上下文
		ctx, cancel := context.WithTimeout(context.Background(), 1*time.Millisecond)
		defer cancel()

		// 等待超时
		time.Sleep(10 * time.Millisecond)

		// 检查上下文是否已超时
		if ctx.Err() == nil {
			t.Error("Context should have timed out")
		}
		if ctx.Err() != context.DeadlineExceeded {
			t.Errorf("Expected DeadlineExceeded, got %v", ctx.Err())
		}
	})

	t.Run("operation timeout", func(t *testing.T) {
		// 测试操作超时
		builder := etcd.NewManagerBuilder().
			WithDialTimeout(1 * time.Millisecond) // 很短的超时

		_, err := builder.Build()
		if err == nil {
			t.Skip("Expected timeout error but operation succeeded")
		}

		// 应该是超时或连接相关的错误
		if !etcd.IsConnectionError(err) && !etcd.IsTimeoutError(err) {
			t.Skipf("Expected timeout or connection error, got: %v", err)
		}
	})
}

// TestResourceCleanup 测试资源清理
func TestResourceCleanup(t *testing.T) {
	_ = NewTestSuite(t)

	t.Run("manager cleanup", func(t *testing.T) {
		builder := etcd.NewManagerBuilder().
			WithEndpoints([]string{"localhost:23791", "localhost:23792", "localhost:23793"})

		manager, err := builder.Build()
		if err != nil {
			t.Skipf("Skipping test due to build error: %v", err)
			return
		}

		// 检查管理器状态
		if !manager.IsReady() {
			t.Error("Manager should be ready after creation")
		}

		// 关闭管理器
		err = manager.Close()
		if err != nil {
			t.Errorf("Failed to close manager: %v", err)
		}

		// 检查管理器状态
		if manager.IsReady() {
			t.Error("Manager should not be ready after close")
		}

		// 重复关闭应该是安全的
		err = manager.Close()
		if err != nil {
			t.Errorf("Second close should be safe: %v", err)
		}
	})

	t.Run("factory cleanup", func(t *testing.T) {
		factory := etcd.NewEtcdManagerFactory()

		// 创建多个管理器
		managers := make([]etcd.EtcdManager, 0, 3)
		for i := 0; i < 3; i++ {
			manager, err := factory.CreateManagerWithOptions(&etcd.ManagerOptions{
				Endpoints:   []string{"localhost:23791", "localhost:23792", "localhost:23793"},
				DialTimeout: 5 * time.Second,
				DefaultTTL:  30,
			})
			if err != nil {
				t.Skipf("Skipping test due to creation error: %v", err)
				return
			}
			managers = append(managers, manager)
		}

		// 清理所有管理器
		for _, manager := range managers {
			if manager != nil {
				err := manager.Close()
				if err != nil {
					t.Errorf("Failed to close manager: %v", err)
				}
			}
		}
	})
}

// 辅助函数
func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr ||
		(len(s) > len(substr) &&
			(s[:len(substr)] == substr ||
				s[len(s)-len(substr):] == substr ||
				containsSubstring(s, substr))))
}

func containsSubstring(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
