package coord

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type TestConfig struct {
	Name    string `json:"name"`
	Version string `json:"version"`
	Port    int    `json:"port"`
	Enabled bool   `json:"enabled"`
}

// TestConfigSetAndGet 测试配置设置和获取
func TestConfigSetAndGet(t *testing.T) {
	coord, err := New(DefaultConfig())
	require.NoError(t, err)
	defer coord.Close()

	cc := coord.Config()
	ctx := context.Background()

	// 测试字符串类型
	key := "app/name"
	value := "gochat"

	err = cc.Set(ctx, key, value)
	require.NoError(t, err)

	var retrievedValue string
	err = cc.Get(ctx, key, &retrievedValue)
	require.NoError(t, err)
	assert.Equal(t, value, retrievedValue)

	// 测试整数类型
	intKey := "app/port"
	intValue := 8080

	err = cc.Set(ctx, intKey, intValue)
	require.NoError(t, err)

	var retrievedIntValue int
	err = cc.Get(ctx, intKey, &retrievedIntValue)
	require.NoError(t, err)
	assert.Equal(t, intValue, retrievedIntValue)

	// 测试布尔类型
	boolKey := "app/enabled"
	boolValue := true

	err = cc.Set(ctx, boolKey, boolValue)
	require.NoError(t, err)

	var retrievedBoolValue bool
	err = cc.Get(ctx, boolKey, &retrievedBoolValue)
	require.NoError(t, err)
	assert.Equal(t, boolValue, retrievedBoolValue)
}

// TestConfigStruct 测试结构体配置
func TestConfigStruct(t *testing.T) {
	coord, err := New(DefaultConfig())
	require.NoError(t, err)
	defer coord.Close()

	cc := coord.Config()
	ctx := context.Background()

	key := "app/config"
	config := TestConfig{
		Name:    "TestApp",
		Version: "1.0.0",
		Port:    9090,
		Enabled: true,
	}

	// 设置结构体配置
	err = cc.Set(ctx, key, &config)
	require.NoError(t, err)

	// 获取结构体配置
	var retrievedConfig TestConfig
	err = cc.Get(ctx, key, &retrievedConfig)
	require.NoError(t, err)
	assert.Equal(t, config, retrievedConfig)
}

// TestConfigDelete 测试配置删除
func TestConfigDelete(t *testing.T) {
	coord, err := New(DefaultConfig())
	require.NoError(t, err)
	defer coord.Close()

	cc := coord.Config()
	ctx := context.Background()

	key := "app/temp"
	value := "temporary"

	// 设置配置
	err = cc.Set(ctx, key, value)
	require.NoError(t, err)

	// 验证配置存在
	var retrievedValue string
	err = cc.Get(ctx, key, &retrievedValue)
	require.NoError(t, err)
	assert.Equal(t, value, retrievedValue)

	// 删除配置
	err = cc.Delete(ctx, key)
	require.NoError(t, err)

	// 验证配置不存在
	err = cc.Get(ctx, key, &retrievedValue)
	assert.Error(t, err)
}

// TestConfigWatch 测试配置监听
func TestConfigWatch(t *testing.T) {
	coord, err := New(DefaultConfig())
	require.NoError(t, err)
	defer coord.Close()

	cc := coord.Config()
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	key := "watch/test"

	// 开始监听
	var watchValue string
	watcher, err := cc.Watch(ctx, key, &watchValue)
	require.NoError(t, err)
	defer watcher.Close()

	// 在另一个 goroutine 中修改配置
	go func() {
		time.Sleep(100 * time.Millisecond)
		cc.Set(context.Background(), key, "value1")

		time.Sleep(100 * time.Millisecond)
		cc.Set(context.Background(), key, "value2")

		time.Sleep(100 * time.Millisecond)
		cc.Delete(context.Background(), key)
	}()

	// 接收事件
	eventCount := 0
	for event := range watcher.Chan() {
		eventCount++
		assert.NotEmpty(t, event.Type)
		assert.Equal(t, key, event.Key)

		if eventCount >= 3 { // SET, SET, DELETE
			break
		}
	}

	assert.GreaterOrEqual(t, eventCount, 1)
}

// TestConfigWatchPrefix 测试前缀监听
func TestConfigWatchPrefix(t *testing.T) {
	coord, err := New(DefaultConfig())
	require.NoError(t, err)
	defer coord.Close()

	cc := coord.Config()
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	prefix := "prefix/test"

	// 开始前缀监听
	var watchValue string
	watcher, err := cc.WatchPrefix(ctx, prefix, &watchValue)
	require.NoError(t, err)
	defer watcher.Close()

	// 在另一个 goroutine 中修改配置
	go func() {
		time.Sleep(100 * time.Millisecond)
		cc.Set(context.Background(), prefix+"/key1", "value1")

		time.Sleep(100 * time.Millisecond)
		cc.Set(context.Background(), prefix+"/key2", "value2")

		time.Sleep(100 * time.Millisecond)
		cc.Delete(context.Background(), prefix+"/key1")
	}()

	// 接收事件
	eventCount := 0
	for event := range watcher.Chan() {
		eventCount++
		assert.NotEmpty(t, event.Type)
		assert.Contains(t, event.Key, prefix)

		if eventCount >= 3 { // SET, SET, DELETE
			break
		}
	}

	assert.GreaterOrEqual(t, eventCount, 1)
}

// TestInvalidConfigOperations 测试无效的配置操作
func TestInvalidConfigOperations(t *testing.T) {
	coord, err := New(DefaultConfig())
	require.NoError(t, err)
	defer coord.Close()

	cc := coord.Config()
	ctx := context.Background()

	// 空键设置
	err = cc.Set(ctx, "", "value")
	assert.Error(t, err)

	// 空键获取
	var value string
	err = cc.Get(ctx, "", &value)
	assert.Error(t, err)

	// 空键删除
	err = cc.Delete(ctx, "")
	assert.Error(t, err)

	// 获取不存在的键
	err = cc.Get(ctx, "nonexistent/key", &value)
	assert.Error(t, err)
}

// TestConfigComplexTypes 测试复杂类型配置
func TestConfigComplexTypes(t *testing.T) {
	coord, err := New(DefaultConfig())
	require.NoError(t, err)
	defer coord.Close()

	cc := coord.Config()
	ctx := context.Background()

	// 测试数组
	arrayKey := "app/features"
	arrayValue := []string{"feature1", "feature2", "feature3"}

	err = cc.Set(ctx, arrayKey, arrayValue)
	require.NoError(t, err)

	var retrievedArray []string
	err = cc.Get(ctx, arrayKey, &retrievedArray)
	require.NoError(t, err)
	assert.Equal(t, arrayValue, retrievedArray)

	// 测试 map
	mapKey := "app/metadata"
	mapValue := map[string]string{
		"env":     "test",
		"version": "1.0.0",
		"team":    "backend",
	}

	err = cc.Set(ctx, mapKey, mapValue)
	require.NoError(t, err)

	var retrievedMap map[string]string
	err = cc.Get(ctx, mapKey, &retrievedMap)
	require.NoError(t, err)
	assert.Equal(t, mapValue, retrievedMap)
}

// TestConfigConcurrentOperations 测试并发配置操作
func TestConfigConcurrentOperations(t *testing.T) {
	coord, err := New(DefaultConfig())
	require.NoError(t, err)
	defer coord.Close()

	cc := coord.Config()
	ctx := context.Background()

	// 并发设置不同的键
	for i := 0; i < 10; i++ {
		go func(id int) {
			key := "concurrent/key" + string(rune(id+'0'))
			value := "value" + string(rune(id+'0'))

			err := cc.Set(ctx, key, value)
			assert.NoError(t, err)

			var retrievedValue string
			err = cc.Get(ctx, key, &retrievedValue)
			assert.NoError(t, err)
			assert.Equal(t, value, retrievedValue)
		}(i)
	}

	time.Sleep(1 * time.Second) // 等待所有 goroutine 完成
}

// BenchmarkConfigSet 基准测试配置设置
func BenchmarkConfigSet(b *testing.B) {
	coord, err := New(DefaultConfig())
	if err != nil {
		b.Fatal(err)
	}
	defer coord.Close()

	cc := coord.Config()
	ctx := context.Background()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		key := "bench/key"
		value := "bench-value"
		cc.Set(ctx, key, value)
	}
}

// BenchmarkConfigGet 基准测试配置获取
func BenchmarkConfigGet(b *testing.B) {
	coord, err := New(DefaultConfig())
	if err != nil {
		b.Fatal(err)
	}
	defer coord.Close()

	cc := coord.Config()
	ctx := context.Background()

	// 预先设置一个配置
	key := "bench/get/key"
	value := "bench-get-value"
	cc.Set(ctx, key, value)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		var retrievedValue string
		cc.Get(ctx, key, &retrievedValue)
	}
}
