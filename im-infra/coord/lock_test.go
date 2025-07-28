package coord

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestLockAcquireAndRelease 测试锁的获取和释放
func TestLockAcquireAndRelease(t *testing.T) {
	coord, err := New(DefaultConfig())
	require.NoError(t, err)
	defer coord.Close()

	lockService := coord.Lock()
	ctx := context.Background()
	lockKey := "test-lock-1"

	// 获取锁
	lock, err := lockService.Acquire(ctx, lockKey, 30*time.Second)
	require.NoError(t, err)
	assert.NotNil(t, lock)

	// 释放锁
	err = lock.Unlock(ctx)
	assert.NoError(t, err)
}

// TestLockTryAcquire 测试非阻塞锁获取
func TestLockTryAcquire(t *testing.T) {
	coord, err := New(DefaultConfig())
	require.NoError(t, err)
	defer coord.Close()

	lockService := coord.Lock()
	ctx := context.Background()
	lockKey := "test-lock-2"

	// 第一次获取应该成功
	lock1, err := lockService.TryAcquire(ctx, lockKey, 30*time.Second)
	require.NoError(t, err)
	assert.NotNil(t, lock1)

	// 第二次获取应该失败
	lock2, err := lockService.TryAcquire(ctx, lockKey, 30*time.Second)
	assert.Error(t, err)
	assert.Nil(t, lock2)

	// 释放第一个锁
	err = lock1.Unlock(ctx)
	assert.NoError(t, err)

	// 现在第三次获取应该成功
	lock3, err := lockService.TryAcquire(ctx, lockKey, 30*time.Second)
	require.NoError(t, err)
	assert.NotNil(t, lock3)

	err = lock3.Unlock(ctx)
	assert.NoError(t, err)
}

// TestConcurrentLocks 测试并发锁
func TestConcurrentLocks(t *testing.T) {
	coord, err := New(DefaultConfig())
	require.NoError(t, err)
	defer coord.Close()

	lockService := coord.Lock()
	ctx := context.Background()
	lockKey := "test-concurrent-lock"

	var wg sync.WaitGroup
	var successCount int32
	var mu sync.Mutex

	// 启动多个 goroutine 尝试获取同一个锁
	for i := 0; i < 5; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()

			lock, err := lockService.TryAcquire(ctx, lockKey, 10*time.Second)
			if err == nil && lock != nil {
				mu.Lock()
				successCount++
				mu.Unlock()

				// 持有锁一段时间
				time.Sleep(100 * time.Millisecond)
				lock.Unlock(ctx)
			}
		}(i)
	}

	wg.Wait()

	// 只有一个 goroutine 应该成功获取锁
	assert.Equal(t, int32(1), successCount)
}

// TestLockTimeout 测试锁超时
func TestLockTimeout(t *testing.T) {
	coord, err := New(DefaultConfig())
	require.NoError(t, err)
	defer coord.Close()

	lockService := coord.Lock()
	ctx := context.Background()
	lockKey := "test-timeout-lock"

	// 获取锁
	lock, err := lockService.Acquire(ctx, lockKey, 1*time.Second) // 很短的 TTL
	require.NoError(t, err)
	assert.NotNil(t, lock)

	// 释放锁
	err = lock.Unlock(ctx)
	assert.NoError(t, err)

	// 现在应该能够获取新的锁
	lock2, err := lockService.TryAcquire(ctx, lockKey, 30*time.Second)
	require.NoError(t, err)
	assert.NotNil(t, lock2)

	err = lock2.Unlock(ctx)
	assert.NoError(t, err)
}

// TestInvalidLockKey 测试无效的锁键
func TestInvalidLockKey(t *testing.T) {
	coord, err := New(DefaultConfig())
	require.NoError(t, err)
	defer coord.Close()

	lockService := coord.Lock()
	ctx := context.Background()

	// 空键应该失败
	lock, err := lockService.Acquire(ctx, "", 30*time.Second)
	assert.Error(t, err)
	assert.Nil(t, lock)

	// TryAcquire 也应该失败
	lock, err = lockService.TryAcquire(ctx, "", 30*time.Second)
	assert.Error(t, err)
	assert.Nil(t, lock)
}

// TestLockSequentialAccess 测试锁的顺序访问
func TestLockSequentialAccess(t *testing.T) {
	coord, err := New(DefaultConfig())
	require.NoError(t, err)
	defer coord.Close()

	lockService := coord.Lock()
	ctx := context.Background()
	lockKey := "test-sequential-lock"

	var accessOrder []int
	var mu sync.Mutex
	var wg sync.WaitGroup

	// 启动多个 goroutine 按顺序获取锁
	for i := 0; i < 3; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()

			lock, err := lockService.Acquire(ctx, lockKey, 30*time.Second)
			if err == nil && lock != nil {
				mu.Lock()
				accessOrder = append(accessOrder, id)
				mu.Unlock()

				// 持有锁一段时间
				time.Sleep(200 * time.Millisecond)
				lock.Unlock(ctx)
			}
		}(i)
	}

	wg.Wait()

	// 应该有3个访问记录
	assert.Len(t, accessOrder, 3)
	// 每个 goroutine 都应该能获取到锁
	assert.Contains(t, accessOrder, 0)
	assert.Contains(t, accessOrder, 1)
	assert.Contains(t, accessOrder, 2)
}

// TestLockWithContext 测试带上下文的锁操作
func TestLockWithContext(t *testing.T) {
	coord, err := New(DefaultConfig())
	require.NoError(t, err)
	defer coord.Close()

	lockService := coord.Lock()
	lockKey := "test-context-lock"

	// 测试上下文取消
	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()

	// 先获取一个锁
	lock1, err := lockService.Acquire(context.Background(), lockKey, 30*time.Second)
	require.NoError(t, err)
	defer lock1.Unlock(context.Background())

	// 尝试用短超时的上下文获取同一个锁，应该超时
	lock2, err := lockService.Acquire(ctx, lockKey, 30*time.Second)
	assert.Error(t, err)
	assert.Nil(t, lock2)
}

// BenchmarkLockAcquireRelease 基准测试锁的获取和释放
func BenchmarkLockAcquireRelease(b *testing.B) {
	coord, err := New(DefaultConfig())
	if err != nil {
		b.Fatal(err)
	}
	defer coord.Close()

	lockService := coord.Lock()
	ctx := context.Background()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		lockKey := "bench-lock"
		lock, err := lockService.Acquire(ctx, lockKey, 30*time.Second)
		if err != nil {
			b.Fatal(err)
		}
		lock.Unlock(ctx)
	}
}

// BenchmarkTryAcquire 基准测试非阻塞锁获取
func BenchmarkTryAcquire(b *testing.B) {
	coord, err := New(DefaultConfig())
	if err != nil {
		b.Fatal(err)
	}
	defer coord.Close()

	lockService := coord.Lock()
	ctx := context.Background()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		lockKey := "bench-try-lock"
		lock, err := lockService.TryAcquire(ctx, lockKey, 30*time.Second)
		if err == nil && lock != nil {
			lock.Unlock(ctx)
		}
	}
}

// BenchmarkConcurrentLockOperations 基准测试并发锁操作
func BenchmarkConcurrentLockOperations(b *testing.B) {
	coord, err := New(DefaultConfig())
	if err != nil {
		b.Fatal(err)
	}
	defer coord.Close()

	lockService := coord.Lock()
	ctx := context.Background()

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			lockKey := "bench-concurrent-lock"
			lock, err := lockService.TryAcquire(ctx, lockKey, 10*time.Second)
			if err == nil && lock != nil {
				lock.Unlock(ctx)
			}
		}
	})
}
