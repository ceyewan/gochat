package test

import (
	"os"
	"testing"
	"time"

	"myetcd/etcd"
)

func TestEndpointPriority(t *testing.T) {
	t.Run("QuickStart无参数使用配置文件", func(t *testing.T) {
		// 创建临时配置文件
		tmpfile, err := os.CreateTemp("", "test-priority-*.json")
		if err != nil {
			t.Fatalf("Failed to create temp file: %v", err)
		}
		defer os.Remove(tmpfile.Name())

		// 写入配置文件
		configContent := `{
			"endpoints": ["config:1001", "config:1002", "config:1003"],
			"dial_timeout": "3s",
			"log_level": "debug"
		}`

		if _, err := tmpfile.Write([]byte(configContent)); err != nil {
			t.Fatalf("Failed to write temp file: %v", err)
		}
		tmpfile.Close()

		// 重命名为 etcd-config.json
		configPath := "etcd-config.json"
		if err := os.Rename(tmpfile.Name(), configPath); err != nil {
			t.Fatalf("Failed to rename config file: %v", err)
		}
		defer os.Remove(configPath)

		// 测试配置加载
		config, err := etcd.LoadConfig(nil, "")
		if err != nil {
			t.Fatalf("LoadConfig failed: %v", err)
		}

		if len(config.Endpoints) != 3 {
			t.Errorf("Expected 3 endpoints from config file, got %d", len(config.Endpoints))
		}

		if config.Endpoints[0] != "config:1001" {
			t.Errorf("Expected config endpoint, got %s", config.Endpoints[0])
		}

		if config.Source != etcd.SourceFile {
			t.Errorf("Expected file source, got %v", config.Source)
		}
	})

	t.Run("QuickStart有参数优先使用用户输入", func(t *testing.T) {
		// 创建临时配置文件
		tmpfile, err := os.CreateTemp("", "test-priority-*.json")
		if err != nil {
			t.Fatalf("Failed to create temp file: %v", err)
		}
		defer os.Remove(tmpfile.Name())

		// 写入配置文件
		configContent := `{
			"endpoints": ["config:2001", "config:2002"],
			"dial_timeout": "5s"
		}`

		if _, err := tmpfile.Write([]byte(configContent)); err != nil {
			t.Fatalf("Failed to write temp file: %v", err)
		}
		tmpfile.Close()

		// 重命名为 etcd-config.json
		configPath := "etcd-config.json"
		if err := os.Rename(tmpfile.Name(), configPath); err != nil {
			t.Fatalf("Failed to rename config file: %v", err)
		}
		defer os.Remove(configPath)

		// 测试用户输入优先级
		userEndpoints := []string{"user:3001", "user:3002"}
		config, err := etcd.LoadConfig(userEndpoints, "")
		if err != nil {
			t.Fatalf("LoadConfig failed: %v", err)
		}

		if len(config.Endpoints) != 2 {
			t.Errorf("Expected 2 user endpoints, got %d", len(config.Endpoints))
		}

		if config.Endpoints[0] != "user:3001" {
			t.Errorf("Expected user endpoint, got %s", config.Endpoints[0])
		}

		// 其他配置应该来自文件
		if config.DialTimeout != 5*time.Second {
			t.Errorf("Expected file timeout, got %v", config.DialTimeout)
		}

		if config.Source != etcd.SourceMixed {
			t.Errorf("Expected mixed source, got %v", config.Source)
		}
	})

	t.Run("没有配置文件使用默认值", func(t *testing.T) {
		// 确保没有配置文件
		os.Remove("etcd-config.json")
		os.Remove("etcd-config.yaml")

		config, err := etcd.LoadConfig(nil, "")
		if err != nil {
			t.Fatalf("LoadConfig failed: %v", err)
		}

		// 应该使用默认配置
		expectedEndpoints := []string{"localhost:23791", "localhost:23792", "localhost:23793"}
		if len(config.Endpoints) != len(expectedEndpoints) {
			t.Errorf("Expected %d default endpoints, got %d", len(expectedEndpoints), len(config.Endpoints))
		}

		for i, expected := range expectedEndpoints {
			if i >= len(config.Endpoints) || config.Endpoints[i] != expected {
				t.Errorf("Expected default endpoint %s, got %s", expected, config.Endpoints[i])
			}
		}

		if config.Source != etcd.SourceDefault {
			t.Errorf("Expected default source, got %v", config.Source)
		}
	})
}

func TestConnectionRetry(t *testing.T) {
	t.Run("连接失败重试机制", func(t *testing.T) {
		// 创建一个无效的端点配置
		invalidEndpoints := []string{"invalid:9999"}
		config, err := etcd.LoadConfig(invalidEndpoints, "")
		if err != nil {
			t.Fatalf("LoadConfig failed: %v", err)
		}

		options := config.ToManagerOptions()
		options.DialTimeout = 1 * time.Second // 短超时以快速失败

		// 尝试创建管理器，应该失败
		manager, err := etcd.NewEtcdManager(options)
		if err == nil {
			manager.Close()
			t.Error("Expected connection to fail with invalid endpoint")
		}

		// 检查错误类型（可能被包装在配置错误中）
		if !etcd.IsConnectionError(err) && !etcd.IsConfigurationError(err) {
			t.Errorf("Expected connection or configuration error, got %v", err)
		}
	})

	t.Run("错误重试判断", func(t *testing.T) {
		testCases := []struct {
			name      string
			err       error
			retryable bool
		}{
			{
				name:      "连接错误可重试",
				err:       etcd.ErrConnectionFailed,
				retryable: true,
			},
			{
				name:      "超时错误可重试",
				err:       etcd.ErrTimeout,
				retryable: true,
			},
			{
				name:      "配置错误不可重试",
				err:       etcd.ErrInvalidConfiguration,
				retryable: false,
			},
			{
				name:      "服务未找到不可重试",
				err:       etcd.ErrServiceNotFound,
				retryable: false,
			},
		}

		for _, tc := range testCases {
			t.Run(tc.name, func(t *testing.T) {
				isRetryable := etcd.IsRetryableError(tc.err)
				if isRetryable != tc.retryable {
					t.Errorf("Expected retryable=%v for %s, got %v", tc.retryable, tc.name, isRetryable)
				}
			})
		}
	})

	t.Run("重试延迟计算", func(t *testing.T) {
		testCases := []struct {
			attempt     int
			expectedMin int
			expectedMax int
		}{
			{0, 100, 100},
			{1, 200, 200},
			{2, 400, 400},
			{3, 800, 800},
			{4, 1600, 1600},
			{5, 3200, 3200},  // 应该限制在最大值
			{10, 3200, 3200}, // 应该限制在最大值
		}

		for _, tc := range testCases {
			delay := etcd.GetRetryDelay(tc.attempt)
			if delay < tc.expectedMin || delay > tc.expectedMax {
				t.Errorf("Attempt %d: expected delay between %d-%d ms, got %d ms",
					tc.attempt, tc.expectedMin, tc.expectedMax, delay)
			}
		}
	})
}

func TestQuickStartBehavior(t *testing.T) {
	t.Run("QuickStart配置优先级验证", func(t *testing.T) {
		// 确保清理现有配置文件
		os.Remove("etcd-config.json")
		os.Remove("etcd-config.yaml")

		// 测试1: 无参数，无配置文件，使用默认配置
		// 注意：这里我们不实际创建连接，只验证配置加载
		config, err := etcd.LoadConfig(nil, "")
		if err != nil {
			t.Fatalf("LoadConfig failed: %v", err)
		}

		if config.Source != etcd.SourceDefault {
			t.Errorf("Expected default source, got %v", config.Source)
		}

		// 测试2: 有参数，覆盖默认配置
		userEndpoints := []string{"user:4001"}
		config, err = etcd.LoadConfig(userEndpoints, "")
		if err != nil {
			t.Fatalf("LoadConfig failed: %v", err)
		}

		if config.Source != etcd.SourceUserInput {
			t.Errorf("Expected user input source, got %v", config.Source)
		}

		if config.Endpoints[0] != "user:4001" {
			t.Errorf("Expected user endpoint, got %s", config.Endpoints[0])
		}
	})
}

func TestConfigurationValidation(t *testing.T) {
	t.Run("无效配置验证", func(t *testing.T) {
		// 创建无效配置的临时文件
		tmpfile, err := os.CreateTemp("", "invalid-config-*.json")
		if err != nil {
			t.Fatalf("Failed to create temp file: %v", err)
		}
		defer os.Remove(tmpfile.Name())

		// 写入无效的JSON
		invalidJSON := `{
			"endpoints": ["test:5001"]
			"dial_timeout": "invalid_duration"
		}`

		if _, err := tmpfile.Write([]byte(invalidJSON)); err != nil {
			t.Fatalf("Failed to write temp file: %v", err)
		}
		tmpfile.Close()

		// 尝试加载配置，应该失败
		_, err = etcd.LoadConfigFromFile(tmpfile.Name())
		if err == nil {
			t.Error("Expected error when loading invalid JSON config")
		}

		if !etcd.IsConfigurationError(err) {
			t.Errorf("Expected configuration error, got %v", err)
		}
	})

	t.Run("空端点列表验证", func(t *testing.T) {
		// 创建空端点配置
		tmpfile, err := os.CreateTemp("", "empty-endpoints-*.json")
		if err != nil {
			t.Fatalf("Failed to create temp file: %v", err)
		}
		defer os.Remove(tmpfile.Name())

		// 写入空端点配置
		emptyEndpoints := `{
			"endpoints": [],
			"dial_timeout": "5s"
		}`

		if _, err := tmpfile.Write([]byte(emptyEndpoints)); err != nil {
			t.Fatalf("Failed to write temp file: %v", err)
		}
		tmpfile.Close()

		// 加载配置
		config, err := etcd.LoadConfigFromFile(tmpfile.Name())
		if err != nil {
			t.Fatalf("LoadConfigFromFile failed: %v", err)
		}

		// 将配置转换为ManagerOptions并验证
		options := config.ToManagerOptions()

		// 创建建造者并验证
		builder := etcd.NewManagerBuilder()
		builder = builder.WithEndpoints(options.Endpoints).
			WithDialTimeout(options.DialTimeout)

		// 尝试构建，应该失败因为空端点
		_, err = builder.Build()
		if err == nil {
			t.Error("Expected error when building with empty endpoints")
		}

		if !etcd.IsConfigurationError(err) {
			t.Errorf("Expected configuration error, got %v", err)
		}
	})
}
