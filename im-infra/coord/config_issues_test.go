package coord

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestConfigWatchTypeIssues 专门测试 watch 功能的类型问题
func TestConfigWatchTypeIssues(t *testing.T) {
	coord, err := New(DefaultConfig())
	require.NoError(t, err)
	defer coord.Close()

	cc := coord.Config()
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	t.Run("StringWatchReceivesIntValue", func(t *testing.T) {
		// 使用 string 类型监听，但设置 int 值
		var watchValue string
		watcher, err := cc.Watch(ctx, "type-issue/int-as-string", &watchValue)
		require.NoError(t, err)
		defer watcher.Close()

		// 设置一个整数值
		err = cc.Set(ctx, "type-issue/int-as-string", 42)
		require.NoError(t, err)

		// 尝试接收事件
		select {
		case event := <-watcher.Chan():
			t.Logf("Received event: Type=%s, Key=%s, Value=%v (%T)",
				event.Type, event.Key, event.Value, event.Value)
			// 这里应该会有类型转换问题
		case <-time.After(2 * time.Second):
			t.Log("No event received within timeout - this might indicate a problem")
		}
	})

	t.Run("InterfaceWatchReceivesDifferentTypes", func(t *testing.T) {
		// 使用 interface{} 监听不同类型
		var watchValue interface{}
		watcher, err := cc.WatchPrefix(ctx, "mixed-types", &watchValue)
		require.NoError(t, err)
		defer watcher.Close()

		// 设置不同类型的值
		testCases := []struct {
			key   string
			value interface{}
		}{
			{"mixed-types/string", "hello"},
			{"mixed-types/int", 123},
			{"mixed-types/bool", true},
			{"mixed-types/float", 3.14},
		}

		for _, tc := range testCases {
			err = cc.Set(ctx, tc.key, tc.value)
			require.NoError(t, err)
		}

		// 接收事件
		receivedEvents := 0
		timeout := time.After(3 * time.Second)

		for receivedEvents < len(testCases) {
			select {
			case event := <-watcher.Chan():
				receivedEvents++
				t.Logf("Event %d: Type=%s, Key=%s, Value=%v (%T)",
					receivedEvents, event.Type, event.Key, event.Value, event.Value)
			case <-timeout:
				t.Logf("Timeout after receiving %d/%d events", receivedEvents, len(testCases))
				break
			}
		}

		// 验证是否收到了所有事件
		if receivedEvents < len(testCases) {
			t.Errorf("Expected %d events, but only received %d", len(testCases), receivedEvents)
		}
	})
}

// TestConfigWatchTimeout 测试 watch 超时问题
func TestConfigWatchTimeout(t *testing.T) {
	coord, err := New(DefaultConfig())
	require.NoError(t, err)
	defer coord.Close()

	cc := coord.Config()
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	var watchValue string
	watcher, err := cc.Watch(ctx, "timeout-test", &watchValue)
	require.NoError(t, err)
	defer watcher.Close()

	// 不设置任何值，只是等待看是否会超时
	select {
	case event := <-watcher.Chan():
		t.Logf("Unexpected event received: %v", event)
	case <-time.After(1 * time.Second):
		t.Log("No event received as expected")
	}

	// 现在设置一个值
	err = cc.Set(ctx, "timeout-test", "test-value")
	require.NoError(t, err)

	// 应该能收到事件
	select {
	case event := <-watcher.Chan():
		t.Logf("Received expected event: %v", event)
		assert.Equal(t, "timeout-test", event.Key)
	case <-time.After(2 * time.Second):
		t.Error("Failed to receive event within timeout")
	}
}

// TestConfigWatchCleanup 测试 watch 清理
func TestConfigWatchCleanup(t *testing.T) {
	coord, err := New(DefaultConfig())
	require.NoError(t, err)
	defer coord.Close()

	cc := coord.Config()
	ctx := context.Background()

	// 创建多个 watcher 并立即关闭
	for i := 0; i < 5; i++ {
		var watchValue string
		watcher, err := cc.Watch(ctx, "cleanup-test", &watchValue)
		require.NoError(t, err)

		// 立即关闭
		watcher.Close()
	}

	// 验证没有资源泄漏（这个测试主要是确保不会 panic）
	t.Log("Cleanup test completed successfully")
}

// TestConfigWatchConcurrentAccess 测试并发访问
func TestConfigWatchConcurrentAccess(t *testing.T) {
	coord, err := New(DefaultConfig())
	require.NoError(t, err)
	defer coord.Close()

	cc := coord.Config()
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// 创建多个 watcher 监听同一个键
	numWatchers := 3
	watchers := make([]interface{ Close() }, numWatchers)
	eventCounts := make([]int, numWatchers)

	for i := 0; i < numWatchers; i++ {
		var watchValue string
		watcher, err := cc.Watch(ctx, "concurrent-test", &watchValue)
		require.NoError(t, err)
		watchers[i] = watcher

		// 启动 goroutine 监听事件
		go func(index int) {
			for event := range watcher.Chan() {
				eventCounts[index]++
				t.Logf("Watcher %d received event: %v", index, event)
			}
		}(i)
	}

	// 清理
	defer func() {
		for _, w := range watchers {
			if w != nil {
				w.Close()
			}
		}
	}()

	// 等待 watchers 启动
	time.Sleep(100 * time.Millisecond)

	// 设置值
	err = cc.Set(ctx, "concurrent-test", "test-value")
	require.NoError(t, err)

	// 等待事件传播
	time.Sleep(500 * time.Millisecond)

	// 验证所有 watcher 都收到了事件
	for i, count := range eventCounts {
		assert.GreaterOrEqual(t, count, 1, "Watcher %d should receive at least 1 event", i)
	}
}

// TestConfigWatchErrorHandling 测试错误处理
func TestConfigWatchErrorHandling(t *testing.T) {
	coord, err := New(DefaultConfig())
	require.NoError(t, err)
	defer coord.Close()

	cc := coord.Config()
	ctx := context.Background()

	// 测试无效参数
	var watchValue string

	// 空键
	watcher, err := cc.Watch(ctx, "", &watchValue)
	assert.Error(t, err)
	assert.Nil(t, watcher)

	// nil 值指针
	watcher, err = cc.Watch(ctx, "test-key", nil)
	assert.Error(t, err)
	assert.Nil(t, watcher)

	// 空前缀
	watcher, err = cc.WatchPrefix(ctx, "", &watchValue)
	assert.Error(t, err)
	assert.Nil(t, watcher)
}

// TestConfigWatchContextCancellation2 测试上下文取消（避免重复）
func TestConfigWatchContextCancellation2(t *testing.T) {
	coord, err := New(DefaultConfig())
	require.NoError(t, err)
	defer coord.Close()

	cc := coord.Config()
	ctx, cancel := context.WithCancel(context.Background())

	var watchValue string
	watcher, err := cc.Watch(ctx, "cancel-test-2", &watchValue)
	require.NoError(t, err)

	// 启动 goroutine 监听事件
	done := make(chan bool, 1)
	go func() {
		for range watcher.Chan() {
			// 接收事件
		}
		done <- true
	}()

	// 取消上下文
	cancel()

	// 验证 watcher 在合理时间内关闭
	select {
	case <-done:
		t.Log("Watcher closed successfully after context cancellation")
	case <-time.After(2 * time.Second):
		t.Error("Watcher did not close within timeout after context cancellation")
	}

	watcher.Close()
}
