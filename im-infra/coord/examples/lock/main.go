package main

import (
	"context"
	"fmt"
	"os"
	"sync"
	"time"

	"github.com/ceyewan/gochat/im-infra/clog"
	"github.com/ceyewan/gochat/im-infra/coord"
)

func main() {
	// clog 包是零配置的，不需要显式初始化

	// 创建协调器
	provider, err := coord.New(context.Background())
	if err != nil {
		clog.Error("failed to create coordinator", clog.Err(err))
		os.Exit(1)
	}
	defer provider.Close()

	lockService := provider.Lock()
	const lockKey = "my-distributed-lock"

	var wg sync.WaitGroup
	wg.Add(2)

	// --- 协程 1: 成功获取锁并持有 ---
	go func() {
		defer wg.Done()
		clog.Info("[Worker 1] Attempting to acquire lock (blocking)...")
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()

		// 阻塞式获取锁
		l, err := lockService.Acquire(ctx, lockKey, 15*time.Second)
		if err != nil {
			clog.Error("[Worker 1] Failed to acquire lock", clog.Err(err))
			return
		}
		clog.Info("[Worker 1] Lock acquired successfully!", clog.String("key", l.Key()))

		// 模拟持有锁工作 5 秒
		clog.Info("[Worker 1] Working with the lock for 5 seconds...")
		time.Sleep(5 * time.Second)

		// 释放锁
		if err := l.Unlock(context.Background()); err != nil {
			clog.Error("[Worker 1] Failed to unlock", clog.Err(err))
		} else {
			clog.Info("[Worker 1] Lock released successfully.")
		}
	}()

	// 等待协程 1 获取锁
	time.Sleep(1 * time.Second)

	// --- 协程 2: 尝试获取锁，会失败，然后等待获取 ---
	go func() {
		defer wg.Done()
		clog.Info("[Worker 2] Attempting to acquire lock (non-blocking)...")

		// 1. 尝试非阻塞获取，预期会失败
		tryCtx, tryCancel := context.WithTimeout(context.Background(), 2*time.Second)
		defer tryCancel()
		l, err := lockService.TryAcquire(tryCtx, lockKey, 10*time.Second)
		if err != nil {
			clog.Warn("[Worker 2] TryAcquire failed as expected", clog.Err(err))
		} else {
			// 这不应该发生
			clog.Error("[Worker 2] TryAcquire unexpectedly succeeded!")
			_ = l.Unlock(context.Background())
			return
		}

		// 2. 等待并以阻塞方式获取锁
		clog.Info("[Worker 2] Now attempting to acquire lock (blocking)...")
		acquireCtx, acquireCancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer acquireCancel()
		l, err = lockService.Acquire(acquireCtx, lockKey, 10*time.Second)
		if err != nil {
			clog.Error("[Worker 2] Failed to acquire lock after waiting", clog.Err(err))
			return
		}
		clog.Info("[Worker 2] Lock acquired after waiting!", clog.String("key", l.Key()))

		// 模拟工作
		clog.Info("[Worker 2] Working with the lock for 2 seconds...")
		time.Sleep(2 * time.Second)

		// 释放锁
		if err := l.Unlock(context.Background()); err != nil {
			clog.Error("[Worker 2] Failed to unlock", clog.Err(err))
		} else {
			clog.Info("[Worker 2] Lock released successfully.")
		}
	}()

	wg.Wait()
	fmt.Println("\nLock example finished.")
}
