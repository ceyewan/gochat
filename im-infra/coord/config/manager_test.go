package config

import (
	"context"
	"sync"
	"testing"
)

// mockConfigCenter 模拟配置中心
type mockConfigCenter struct {
	data    map[string]interface{}
	version map[string]int64
	mu      sync.RWMutex
}

func newMockConfigCenter() *mockConfigCenter {
	return &mockConfigCenter{
		data:    make(map[string]interface{}),
		version: make(map[string]int64),
	}
}

func (m *mockConfigCenter) Get(ctx context.Context, key string, v interface{}) error {
	m.mu.RLock()
	defer m.mu.RUnlock()

	if data, exists := m.data[key]; exists {
		// 简单的类型断言，实际应该使用 JSON 序列化
		if ptr, ok := v.(*map[string]interface{}); ok {
			if mapData, ok := data.(map[string]interface{}); ok {
				*ptr = mapData
				return nil
			}
		}
	}
	return NewError(ErrCodeNotFound, "key not found", nil)
}

func (m *mockConfigCenter) GetWithVersion(ctx context.Context, key string, v interface{}) (int64, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	if data, exists := m.data[key]; exists {
		if ptr, ok := v.(*map[string]interface{}); ok {
			if mapData, ok := data.(map[string]interface{}); ok {
				*ptr = mapData
				return m.version[key], nil
			}
		}
	}
	return 0, NewError(ErrCodeNotFound, "key not found", nil)
}

func (m *mockConfigCenter) Set(ctx context.Context, key string, value interface{}) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.data[key] = value
	m.version[key]++
	return nil
}

func (m *mockConfigCenter) CompareAndSet(ctx context.Context, key string, value interface{}, expectedVersion int64) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	currentVersion := m.version[key]
	if currentVersion != expectedVersion {
		return NewError(ErrCodeConflict, "version mismatch", nil)
	}

	m.data[key] = value
	m.version[key]++
	return nil
}

func (m *mockConfigCenter) Delete(ctx context.Context, key string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	delete(m.data, key)
	delete(m.version, key)
	return nil
}

func (m *mockConfigCenter) Watch(ctx context.Context, key string, v interface{}) (Watcher[any], error) {
	return nil, NewError(ErrCodeUnavailable, "watch not implemented in mock", nil)
}

func (m *mockConfigCenter) WatchPrefix(ctx context.Context, prefix string, v interface{}) (Watcher[any], error) {
	return nil, NewError(ErrCodeUnavailable, "watch prefix not implemented in mock", nil)
}

func (m *mockConfigCenter) List(ctx context.Context, prefix string) ([]string, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	var keys []string
	for key := range m.data {
		keys = append(keys, key)
	}
	return keys, nil
}

// 错误类型定义（简化版）
type ErrorCode string

const (
	ErrCodeNotFound    ErrorCode = "NOT_FOUND"
	ErrCodeConflict    ErrorCode = "CONFLICT"
	ErrCodeUnavailable ErrorCode = "UNAVAILABLE"
)

type Error struct {
	Code    ErrorCode
	Message string
	Cause   error
}

func (e *Error) Error() string {
	return string(e.Code) + ": " + e.Message
}

func NewError(code ErrorCode, message string, cause error) *Error {
	return &Error{Code: code, Message: message, Cause: cause}
}

// 测试配置类型
type TestConfig struct {
	Name    string `json:"name"`
	Value   int    `json:"value"`
	Enabled bool   `json:"enabled"`
}

// 测试验证器
type testValidator struct{}

func (v *testValidator) Validate(config *TestConfig) error {
	if config.Name == "" {
		return NewError(ErrCodeValidation, "name cannot be empty", nil)
	}
	return nil
}

// 测试更新器
type testUpdater struct {
	updateCount int
	mu          sync.Mutex
}

func (u *testUpdater) OnConfigUpdate(oldConfig, newConfig *TestConfig) error {
	u.mu.Lock()
	defer u.mu.Unlock()
	u.updateCount++
	return nil
}

func (u *testUpdater) GetUpdateCount() int {
	u.mu.Lock()
	defer u.mu.Unlock()
	return u.updateCount
}

// 测试日志器
type testLogger struct {
	logs []string
	mu   sync.Mutex
}

func (l *testLogger) Debug(msg string, fields ...any) { l.log("DEBUG: " + msg) }
func (l *testLogger) Info(msg string, fields ...any)  { l.log("INFO: " + msg) }
func (l *testLogger) Warn(msg string, fields ...any)  { l.log("WARN: " + msg) }
func (l *testLogger) Error(msg string, fields ...any) { l.log("ERROR: " + msg) }

func (l *testLogger) log(msg string) {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.logs = append(l.logs, msg)
}

func (l *testLogger) GetLogs() []string {
	l.mu.Lock()
	defer l.mu.Unlock()
	return append([]string{}, l.logs...)
}

// 测试用例
func TestManagerLifecycle(t *testing.T) {
	configCenter := newMockConfigCenter()
	defaultConfig := TestConfig{Name: "test", Value: 42, Enabled: true}

	manager := NewManager(
		configCenter,
		"test", "service", "component",
		defaultConfig,
	)

	// 测试初始状态
	config := manager.GetCurrentConfig()
	if config.Name != "test" || config.Value != 42 {
		t.Errorf("Expected default config, got %+v", config)
	}

	// 测试启动
	manager.Start()

	// 测试停止
	manager.Stop()

	// 多次调用应该是安全的
	manager.Start()
	manager.Start()
	manager.Stop()
	manager.Stop()
}

func TestManagerWithValidator(t *testing.T) {
	configCenter := newMockConfigCenter()
	defaultConfig := TestConfig{Name: "test", Value: 42, Enabled: true}
	validator := &testValidator{}
	logger := &testLogger{}

	manager := NewManager(
		configCenter,
		"test", "service", "component",
		defaultConfig,
		WithValidator[TestConfig](validator),
		WithLogger[TestConfig](logger),
	)

	manager.Start()
	defer manager.Stop()

	// 测试有效配置更新
	validConfig := TestConfig{Name: "updated", Value: 100, Enabled: false}
	err := manager.safeUpdateAndApply(&validConfig)
	if err != nil {
		t.Errorf("Expected no error for valid config, got %v", err)
	}

	// 验证配置已更新
	currentConfig := manager.GetCurrentConfig()
	if currentConfig.Name != "updated" {
		t.Errorf("Expected updated config, got %+v", currentConfig)
	}
}

const ErrCodeValidation ErrorCode = "VALIDATION_ERROR"

func TestManagerWithUpdater(t *testing.T) {
	configCenter := newMockConfigCenter()
	defaultConfig := TestConfig{Name: "test", Value: 42, Enabled: true}
	updater := &testUpdater{}

	manager := NewManager(
		configCenter,
		"test", "service", "component",
		defaultConfig,
		WithUpdater[TestConfig](updater),
	)

	manager.Start()
	defer manager.Stop()

	// 测试配置更新触发更新器
	newConfig := TestConfig{Name: "updated", Value: 100, Enabled: false}
	err := manager.safeUpdateAndApply(&newConfig)
	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}

	// 验证更新器被调用
	if updater.GetUpdateCount() != 1 {
		t.Errorf("Expected updater to be called once, got %d", updater.GetUpdateCount())
	}
}

func TestConcurrentAccess(t *testing.T) {
	configCenter := newMockConfigCenter()
	defaultConfig := TestConfig{Name: "test", Value: 42, Enabled: true}

	manager := NewManager(
		configCenter,
		"test", "service", "component",
		defaultConfig,
	)

	manager.Start()
	defer manager.Stop()

	// 并发读取配置
	var wg sync.WaitGroup
	for i := 0; i < 100; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			config := manager.GetCurrentConfig()
			if config == nil {
				t.Errorf("Expected non-nil config")
			}
		}()
	}

	wg.Wait()
}
