package test

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/ceyewan/gochat/im-infra/etcd"
)

func TestConfigurationIntegration(t *testing.T) {
	t.Run("完整配置优先级集成测试", func(t *testing.T) {
		// 清理现有配置文件
		os.Remove("etcd-config.json")
		os.Remove("etcd-config.yaml")

		// 场景1: 仅默认配置
		t.Log("测试场景1: 仅默认配置")
		manager1, err := etcd.QuickStart()
		if err != nil {
			// 这里可能因为连接到默认端点失败，但我们主要测试配置加载
			t.Logf("QuickStart with defaults failed (expected if no etcd): %v", err)
		} else {
			manager1.Close()
			t.Log("QuickStart with defaults succeeded")
		}

		// 场景2: 创建配置文件，测试文件配置优先级
		t.Log("测试场景2: 配置文件优先级")
		createTestConfigFile(t, "etcd-config.json", `{
			"endpoints": ["config-file:1001", "config-file:1002"],
			"dial_timeout": "8s",
			"log_level": "debug"
		}`)
		defer os.Remove("etcd-config.json")

		config2, err := etcd.LoadConfig(nil, "")
		if err != nil {
			t.Fatalf("LoadConfig failed: %v", err)
		}

		if config2.Source != etcd.SourceFile {
			t.Errorf("Expected file source, got %v", config2.Source)
		}

		if len(config2.Endpoints) != 2 || config2.Endpoints[0] != "config-file:1001" {
			t.Errorf("Config file not loaded correctly: %v", config2.Endpoints)
		}

		// 场景3: 用户输入覆盖配置文件
		t.Log("测试场景3: 用户输入覆盖配置文件")
		userEndpoints := []string{"user-input:2001", "user-input:2002"}
		config3, err := etcd.LoadConfig(userEndpoints, "")
		if err != nil {
			t.Fatalf("LoadConfig with user input failed: %v", err)
		}

		if config3.Source != etcd.SourceMixed {
			t.Errorf("Expected mixed source, got %v", config3.Source)
		}

		if len(config3.Endpoints) != 2 || config3.Endpoints[0] != "user-input:2001" {
			t.Errorf("User input not prioritized correctly: %v", config3.Endpoints)
		}

		// 其他配置应该来自文件
		if config3.DialTimeout != 8*time.Second {
			t.Errorf("Expected file timeout 8s, got %v", config3.DialTimeout)
		}

		// 场景4: 指定特定配置文件
		t.Log("测试场景4: 指定特定配置文件")
		createTestConfigFile(t, "custom-config.yaml", `
endpoints:
  - custom:3001
  - custom:3002
  - custom:3003
dial_timeout: 12s
log_level: warn
`)
		defer os.Remove("custom-config.yaml")

		config4, err := etcd.LoadConfig(nil, "custom-config.yaml")
		if err != nil {
			t.Fatalf("LoadConfig with custom file failed: %v", err)
		}

		if len(config4.Endpoints) != 3 || config4.Endpoints[0] != "custom:3001" {
			t.Errorf("Custom config file not loaded correctly: %v", config4.Endpoints)
		}

		if config4.DialTimeout != 12*time.Second {
			t.Errorf("Expected custom timeout 12s, got %v", config4.DialTimeout)
		}
	})
}

func TestQuickStartIntegration(t *testing.T) {
	t.Run("QuickStart不同配置组合", func(t *testing.T) {
		// 清理配置文件
		os.Remove("etcd-config.json")
		os.Remove("etcd-config.yaml")

		// 测试1: QuickStart() - 无参数，无配置文件
		t.Log("测试 QuickStart() - 无参数，无配置文件")
		config1, err := etcd.LoadConfig(nil, "")
		if err != nil {
			t.Fatalf("LoadConfig failed: %v", err)
		}

		expectedDefault := []string{"localhost:23791", "localhost:23792", "localhost:23793"}
		if !compareStringSlices(config1.Endpoints, expectedDefault) {
			t.Errorf("Expected default endpoints %v, got %v", expectedDefault, config1.Endpoints)
		}

		// 测试2: QuickStart("custom:4001") - 有参数
		t.Log("测试 QuickStart(\"custom:4001\") - 有参数")
		userEndpoints := []string{"custom:4001"}
		config2, err := etcd.LoadConfig(userEndpoints, "")
		if err != nil {
			t.Fatalf("LoadConfig with user endpoints failed: %v", err)
		}

		if len(config2.Endpoints) != 1 || config2.Endpoints[0] != "custom:4001" {
			t.Errorf("Expected user endpoint, got %v", config2.Endpoints)
		}

		// 测试3: 有配置文件时的 QuickStart()
		t.Log("测试有配置文件时的 QuickStart()")
		createTestConfigFile(t, "etcd-config.json", `{
			"endpoints": ["file:5001", "file:5002"],
			"dial_timeout": "10s"
		}`)
		defer os.Remove("etcd-config.json")

		config3, err := etcd.LoadConfig(nil, "")
		if err != nil {
			t.Fatalf("LoadConfig with config file failed: %v", err)
		}

		if len(config3.Endpoints) != 2 || config3.Endpoints[0] != "file:5001" {
			t.Errorf("Expected file endpoints, got %v", config3.Endpoints)
		}

		// 测试4: 有配置文件时的 QuickStart("override:6001")
		t.Log("测试有配置文件时的 QuickStart(\"override:6001\")")
		overrideEndpoints := []string{"override:6001"}
		config4, err := etcd.LoadConfig(overrideEndpoints, "")
		if err != nil {
			t.Fatalf("LoadConfig with override failed: %v", err)
		}

		if len(config4.Endpoints) != 1 || config4.Endpoints[0] != "override:6001" {
			t.Errorf("Expected override endpoint, got %v", config4.Endpoints)
		}

		// 但其他配置应该来自文件
		if config4.DialTimeout != 10*time.Second {
			t.Errorf("Expected file timeout, got %v", config4.DialTimeout)
		}

		if config4.Source != etcd.SourceMixed {
			t.Errorf("Expected mixed source, got %v", config4.Source)
		}
	})
}

func TestRealEtcdConnection(t *testing.T) {
	t.Run("实际etcd连接测试", func(t *testing.T) {
		// 这个测试只在有实际运行的 etcd 时才会成功
		// 使用 docker-compose 启动的 etcd 集群

		// 清理配置文件
		os.Remove("etcd-config.json")
		os.Remove("etcd-config.yaml")

		// 创建指向实际 etcd 的配置
		createTestConfigFile(t, "etcd-config.json", `{
			"endpoints": ["localhost:23791", "localhost:23792", "localhost:23793"],
			"dial_timeout": "5s",
			"log_level": "info"
		}`)
		defer os.Remove("etcd-config.json")

		// 尝试创建管理器
		manager, err := etcd.QuickStart()
		if err != nil {
			t.Logf("Real etcd connection failed (expected if etcd not running): %v", err)
			// 检查是否是连接错误
			if !etcd.IsConnectionError(err) && !etcd.IsConfigurationError(err) {
				t.Errorf("Expected connection error, got %v", err)
			}
			return
		}
		defer manager.Close()

		t.Log("Successfully connected to real etcd cluster")

		// 验证管理器状态
		if !manager.IsReady() {
			t.Error("Manager should be ready after successful connection")
		}

		// 执行健康检查
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		if err := manager.HealthCheck(ctx); err != nil {
			t.Errorf("Health check failed: %v", err)
		}

		t.Log("Health check passed")
	})
}

func TestConfigErrorHandling(t *testing.T) {
	t.Run("配置错误处理", func(t *testing.T) {
		// 测试1: 无效的JSON格式
		t.Log("测试无效JSON格式")
		createTestConfigFile(t, "invalid.json", `{
			"endpoints": ["test:1001"]
			"dial_timeout": "5s"  # 缺少逗号
		}`)
		defer os.Remove("invalid.json")

		_, err := etcd.LoadConfigFromFile("invalid.json")
		if err == nil {
			t.Error("Expected error for invalid JSON")
		}

		if !etcd.IsConfigurationError(err) {
			t.Errorf("Expected configuration error, got %v", err)
		}

		// 测试2: 无效的时间格式
		t.Log("测试无效时间格式")
		createTestConfigFile(t, "invalid-time.json", `{
			"endpoints": ["test:1002"],
			"dial_timeout": "invalid_duration"
		}`)
		defer os.Remove("invalid-time.json")

		_, err = etcd.LoadConfigFromFile("invalid-time.json")
		if err == nil {
			t.Error("Expected error for invalid time format")
		}

		if !etcd.IsConfigurationError(err) {
			t.Errorf("Expected configuration error, got %v", err)
		}

		// 测试3: 文件不存在（应该不返回错误，使用默认配置）
		t.Log("测试文件不存在")
		config, err := etcd.LoadConfig(nil, "nonexistent.json")
		if err != nil {
			t.Errorf("Expected no error for nonexistent file, got %v", err)
		}

		if config.Source != etcd.SourceDefault {
			t.Errorf("Expected default source, got %v", config.Source)
		}
	})
}

// 辅助函数

func createTestConfigFile(t *testing.T, filename, content string) {
	file, err := os.Create(filename)
	if err != nil {
		t.Fatalf("Failed to create test config file %s: %v", filename, err)
	}
	defer file.Close()

	if _, err := file.WriteString(content); err != nil {
		t.Fatalf("Failed to write test config file %s: %v", filename, err)
	}
}

func compareStringSlices(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}
