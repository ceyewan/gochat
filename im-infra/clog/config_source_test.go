package clog

import (
	"context"
	"testing"
)

// mockConfigSource 模拟配置源
type mockConfigSource struct {
	config *Config
	err    error
}

func (m *mockConfigSource) GetConfig(ctx context.Context, env, service, component string) (*Config, error) {
	if m.err != nil {
		return nil, m.err
	}
	return m.config, nil
}

func (m *mockConfigSource) Watch(ctx context.Context, env, service, component string) (ConfigWatcher, error) {
	return &mockConfigWatcher{
		ch: make(chan *Config, 1),
	}, nil
}

// mockConfigWatcher 模拟配置监听器
type mockConfigWatcher struct {
	ch chan *Config
}

func (m *mockConfigWatcher) Chan() <-chan *Config {
	if m.ch == nil {
		m.ch = make(chan *Config, 1)
	}
	return m.ch
}

func (m *mockConfigWatcher) Close() {
	if m.ch != nil {
		close(m.ch)
	}
}

func TestConfigManager(t *testing.T) {
	// 创建测试配置管理器
	cm := NewConfigManager("test", "im-infra", "clog")
	defer cm.Close()

	// 测试默认配置
	defaultConfig := cm.GetCurrentConfig()
	if defaultConfig == nil {
		t.Error("Default config should not be nil")
	}

	// 测试设置配置源
	mockSource := &mockConfigSource{
		config: &Config{
			Level:  "debug",
			Format: "json",
			Output: "stdout",
		},
	}

	cm.Setup(mockSource, "test", "im-infra", "clog")

	// 验证配置已更新
	config := cm.GetCurrentConfig()
	if config.Level != "debug" {
		t.Errorf("Expected level 'debug', got '%s'", config.Level)
	}

	if config.Format != "json" {
		t.Errorf("Expected format 'json', got '%s'", config.Format)
	}
}

func TestConfigManagerFallback(t *testing.T) {
	// 创建没有配置源的配置管理器
	cm := NewConfigManager("test", "im-infra", "clog")
	defer cm.Close()

	// 应该返回默认配置
	config := cm.GetCurrentConfig()
	if config == nil {
		t.Error("Expected default config when no source is set")
	}

	// 验证是默认配置
	defaultConfig := DefaultConfig()
	if config.Level != defaultConfig.Level {
		t.Errorf("Expected default level '%s', got '%s'", defaultConfig.Level, config.Level)
	}
}

func TestNewWithConfigManager(t *testing.T) {
	// 设置全局配置管理器
	mockSource := &mockConfigSource{
		config: &Config{
			Level:       "warn",
			Format:      "console",
			Output:      "stdout",
			AddSource:   false,
			EnableColor: false,
		},
	}

	// 设置配置源
	SetupConfigCenter(mockSource, "test", "im-infra", "clog")

	// 测试 New 函数使用配置管理器
	logger, err := New()
	if err != nil {
		t.Errorf("Failed to create logger: %v", err)
	}

	if logger == nil {
		t.Error("Logger should not be nil")
	}
}

func TestInitWithConfigManager(t *testing.T) {
	// 设置模拟配置源
	mockSource := &mockConfigSource{
		config: &Config{
			Level:       "error",
			Format:      "json",
			Output:      "stdout",
			AddSource:   true,
			EnableColor: false,
		},
	}

	// 设置配置源
	SetupConfigCenter(mockSource, "test", "im-infra", "clog")

	// 测试 Init 函数使用配置管理器
	err := Init()
	if err != nil {
		t.Errorf("Failed to init logger: %v", err)
	}

	// 验证全局 logger 已更新
	logger := getDefaultLogger()
	if logger == nil {
		t.Error("Default logger should not be nil after Init")
	}
}
