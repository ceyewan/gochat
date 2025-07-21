package test

import (
	"os"
	"testing"
	"time"

	"myetcd/etcd"
)

func TestConfigPriority(t *testing.T) {
	// 测试配置优先级：用户输入 > 配置文件 > 默认值

	t.Run("用户输入优先级最高", func(t *testing.T) {
		userEndpoints := []string{"user:1001", "user:1002"}
		config, err := etcd.LoadConfig(userEndpoints, "")
		if err != nil {
			t.Fatalf("LoadConfig failed: %v", err)
		}

		if len(config.Endpoints) != 2 {
			t.Errorf("Expected 2 endpoints, got %d", len(config.Endpoints))
		}

		if config.Endpoints[0] != "user:1001" || config.Endpoints[1] != "user:1002" {
			t.Errorf("Expected user endpoints, got %v", config.Endpoints)
		}

		if config.Source != etcd.SourceMixed && config.Source != etcd.SourceUserInput {
			t.Errorf("Expected user input source, got %v", config.Source)
		}
	})

	t.Run("配置文件优先级高于默认值", func(t *testing.T) {
		// 创建临时配置文件
		tmpfile, err := os.CreateTemp("", "test-config-*.json")
		if err != nil {
			t.Fatalf("Failed to create temp file: %v", err)
		}
		defer os.Remove(tmpfile.Name())

		configContent := `{
			"endpoints": ["file:2001", "file:2002", "file:2003"],
			"dial_timeout": "10s",
			"log_level": "debug"
		}`

		if _, err := tmpfile.Write([]byte(configContent)); err != nil {
			t.Fatalf("Failed to write temp file: %v", err)
		}
		tmpfile.Close()

		config, err := etcd.LoadConfig(nil, tmpfile.Name())
		if err != nil {
			t.Fatalf("LoadConfig failed: %v", err)
		}

		if len(config.Endpoints) != 3 {
			t.Errorf("Expected 3 endpoints, got %d", len(config.Endpoints))
		}

		if config.Endpoints[0] != "file:2001" {
			t.Errorf("Expected file endpoint, got %s", config.Endpoints[0])
		}

		if config.DialTimeout != 10*time.Second {
			t.Errorf("Expected 10s timeout, got %v", config.DialTimeout)
		}

		if config.Source != etcd.SourceFile {
			t.Errorf("Expected file source, got %v", config.Source)
		}
	})

	t.Run("用户输入覆盖配置文件", func(t *testing.T) {
		// 创建临时配置文件
		tmpfile, err := os.CreateTemp("", "test-config-*.json")
		if err != nil {
			t.Fatalf("Failed to create temp file: %v", err)
		}
		defer os.Remove(tmpfile.Name())

		configContent := `{
			"endpoints": ["file:3001", "file:3002"],
			"dial_timeout": "8s"
		}`

		if _, err := tmpfile.Write([]byte(configContent)); err != nil {
			t.Fatalf("Failed to write temp file: %v", err)
		}
		tmpfile.Close()

		userEndpoints := []string{"user:4001", "user:4002"}
		config, err := etcd.LoadConfig(userEndpoints, tmpfile.Name())
		if err != nil {
			t.Fatalf("LoadConfig failed: %v", err)
		}

		// 用户输入的端点应该覆盖配置文件
		if len(config.Endpoints) != 2 {
			t.Errorf("Expected 2 endpoints, got %d", len(config.Endpoints))
		}

		if config.Endpoints[0] != "user:4001" {
			t.Errorf("Expected user endpoint, got %s", config.Endpoints[0])
		}

		// 但其他配置应该来自文件
		if config.DialTimeout != 8*time.Second {
			t.Errorf("Expected 8s timeout from file, got %v", config.DialTimeout)
		}

		if config.Source != etcd.SourceMixed {
			t.Errorf("Expected mixed source, got %v", config.Source)
		}
	})

	t.Run("使用默认配置", func(t *testing.T) {
		config, err := etcd.LoadConfig(nil, "nonexistent-file.json")
		if err != nil {
			t.Fatalf("LoadConfig failed: %v", err)
		}

		// 应该使用默认配置
		if len(config.Endpoints) != 3 {
			t.Errorf("Expected 3 default endpoints, got %d", len(config.Endpoints))
		}

		expectedEndpoints := []string{"localhost:23791", "localhost:23792", "localhost:23793"}
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

func TestConfigFileFormats(t *testing.T) {
	t.Run("JSON格式配置文件", func(t *testing.T) {
		tmpfile, err := os.CreateTemp("", "test-config-*.json")
		if err != nil {
			t.Fatalf("Failed to create temp file: %v", err)
		}
		defer os.Remove(tmpfile.Name())

		configContent := `{
			"endpoints": ["json:5001", "json:5002"],
			"dial_timeout": "15s",
			"log_level": "error"
		}`

		if _, err := tmpfile.Write([]byte(configContent)); err != nil {
			t.Fatalf("Failed to write temp file: %v", err)
		}
		tmpfile.Close()

		config, err := etcd.LoadConfigFromFile(tmpfile.Name())
		if err != nil {
			t.Fatalf("LoadConfigFromFile failed: %v", err)
		}

		if len(config.Endpoints) != 2 {
			t.Errorf("Expected 2 endpoints, got %d", len(config.Endpoints))
		}

		if config.DialTimeout != 15*time.Second {
			t.Errorf("Expected 15s timeout, got %v", config.DialTimeout)
		}

		if config.LogLevel != "error" {
			t.Errorf("Expected error log level, got %s", config.LogLevel)
		}
	})

	t.Run("YAML格式配置文件", func(t *testing.T) {
		tmpfile, err := os.CreateTemp("", "test-config-*.yaml")
		if err != nil {
			t.Fatalf("Failed to create temp file: %v", err)
		}
		defer os.Remove(tmpfile.Name())

		configContent := `
endpoints:
  - yaml:6001
  - yaml:6002
  - yaml:6003
dial_timeout: 20s
log_level: warn
`

		if _, err := tmpfile.Write([]byte(configContent)); err != nil {
			t.Fatalf("Failed to write temp file: %v", err)
		}
		tmpfile.Close()

		config, err := etcd.LoadConfigFromFile(tmpfile.Name())
		if err != nil {
			t.Fatalf("LoadConfigFromFile failed: %v", err)
		}

		if len(config.Endpoints) != 3 {
			t.Errorf("Expected 3 endpoints, got %d", len(config.Endpoints))
		}

		if config.Endpoints[0] != "yaml:6001" {
			t.Errorf("Expected yaml endpoint, got %s", config.Endpoints[0])
		}

		if config.DialTimeout != 20*time.Second {
			t.Errorf("Expected 20s timeout, got %v", config.DialTimeout)
		}

		if config.LogLevel != "warn" {
			t.Errorf("Expected warn log level, got %s", config.LogLevel)
		}
	})
}

func TestConfigMerging(t *testing.T) {
	t.Run("合并配置测试", func(t *testing.T) {
		defaultConfig := etcd.DefaultConfig()

		fileConfig := &etcd.Config{
			Endpoints:   []string{"file:7001", "file:7002"},
			DialTimeout: 12 * time.Second,
			LogLevel:    "debug",
			Source:      etcd.SourceFile,
		}

		userConfig := &etcd.Config{
			Endpoints: []string{"user:8001"},
			Source:    etcd.SourceUserInput,
		}

		merged := etcd.MergeConfigs(userConfig, fileConfig, defaultConfig)

		// 端点应该来自用户输入
		if len(merged.Endpoints) != 1 || merged.Endpoints[0] != "user:8001" {
			t.Errorf("Expected user endpoint, got %v", merged.Endpoints)
		}

		// 超时应该来自文件配置
		if merged.DialTimeout != 12*time.Second {
			t.Errorf("Expected file timeout, got %v", merged.DialTimeout)
		}

		// 日志级别应该来自文件配置
		if merged.LogLevel != "debug" {
			t.Errorf("Expected file log level, got %s", merged.LogLevel)
		}

		// 源应该是混合
		if merged.Source != etcd.SourceMixed {
			t.Errorf("Expected mixed source, got %v", merged.Source)
		}
	})
}

func TestConfigToManagerOptions(t *testing.T) {
	t.Run("配置转换为管理器选项", func(t *testing.T) {
		config := &etcd.Config{
			Endpoints:   []string{"test:9001", "test:9002"},
			DialTimeout: 30 * time.Second,
			LogLevel:    "info",
			Source:      etcd.SourceFile,
		}

		options := config.ToManagerOptions()

		if len(options.Endpoints) != 2 {
			t.Errorf("Expected 2 endpoints, got %d", len(options.Endpoints))
		}

		if options.DialTimeout != 30*time.Second {
			t.Errorf("Expected 30s timeout, got %v", options.DialTimeout)
		}

		if options.DefaultTTL != 30 {
			t.Errorf("Expected default TTL 30, got %d", options.DefaultTTL)
		}

		if options.ServicePrefix != "/services" {
			t.Errorf("Expected /services prefix, got %s", options.ServicePrefix)
		}
	})
}
