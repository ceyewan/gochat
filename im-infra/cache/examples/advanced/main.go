package main

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/ceyewan/gochat/im-infra/cache"
	"github.com/ceyewan/gochat/im-infra/clog"
)

func main() {
	// 初始化日志
	clog.Info("=== Cache 高级功能演示 ===")

	ctx := context.Background()

	// 1. 分布式锁演示
	demonstrateDistributedLock(ctx)

	// 2. 布隆过滤器演示
	demonstrateBloomFilter(ctx)

	// 3. 并发安全演示
	demonstrateConcurrency(ctx)

	// 4. 错误处理演示
	demonstrateErrorHandling(ctx)

	// 5. 性能优化演示
	demonstratePerformanceOptimization(ctx)

	clog.Info("=== 高级功能演示完成 ===")
}

func demonstrateDistributedLock(ctx context.Context) {
	clog.Info("=== 分布式锁演示 ===")

	// 基本锁操作
	lock, err := cache.AcquireLock(ctx, "resource:critical", time.Minute*5)
	if err != nil {
		clog.Error("获取锁失败", clog.Err(err))
		return
	}

	clog.Info("成功获取锁", clog.String("key", lock.Key()))

	// 模拟临界区操作
	time.Sleep(time.Second * 2)
	clog.Info("执行临界区代码...")

	// 检查锁状态
	isLocked, err := lock.IsLocked(ctx)
	if err != nil {
		clog.Error("检查锁状态失败", clog.Err(err))
	} else {
		clog.Info("锁状态检查", clog.Bool("isLocked", isLocked))
	}

	// 续期锁
	err = lock.Refresh(ctx, time.Minute*10)
	if err != nil {
		clog.Error("续期锁失败", clog.Err(err))
	} else {
		clog.Info("锁续期成功")
	}

	// 释放锁
	err = lock.Unlock(ctx)
	if err != nil {
		clog.Error("释放锁失败", clog.Err(err))
	} else {
		clog.Info("锁释放成功")
	}

	// 演示锁竞争
	demonstrateLockContention(ctx)

	clog.Info("分布式锁演示完成")
}

func demonstrateLockContention(ctx context.Context) {
	clog.Info("=== 锁竞争演示 ===")

	var wg sync.WaitGroup
	lockKey := "resource:contention"

	// 启动多个 goroutine 竞争同一个锁
	for i := 0; i < 3; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()

			clog.Info("尝试获取锁", clog.Int("goroutine", id))

			lock, err := cache.AcquireLock(ctx, lockKey, time.Second*10)
			if err != nil {
				clog.Error("获取锁失败", clog.Int("goroutine", id), clog.Err(err))
				return
			}

			clog.Info("成功获取锁", clog.Int("goroutine", id))

			// 模拟工作
			time.Sleep(time.Second * 2)

			err = lock.Unlock(ctx)
			if err != nil {
				clog.Error("释放锁失败", clog.Int("goroutine", id), clog.Err(err))
			} else {
				clog.Info("释放锁成功", clog.Int("goroutine", id))
			}
		}(i)
	}

	wg.Wait()
	clog.Info("锁竞争演示完成")
}

func demonstrateBloomFilter(ctx context.Context) {
	clog.Info("=== 布隆过滤器演示 ===")

	bloomKey := "users:bloom"

	// 初始化布隆过滤器
	err := cache.BloomInit(ctx, bloomKey, 100000, 0.01)
	if err != nil {
		clog.Error("初始化布隆过滤器失败", clog.Err(err))
		return
	}
	clog.Info("布隆过滤器初始化成功")

	// 添加用户到布隆过滤器
	users := []string{"user123", "user456", "user789", "admin", "guest"}
	for _, user := range users {
		err := cache.BloomAdd(ctx, bloomKey, user)
		if err != nil {
			clog.Error("添加用户到布隆过滤器失败", clog.String("user", user), clog.Err(err))
			continue
		}
		clog.Info("添加用户到布隆过滤器", clog.String("user", user))
	}

	// 检查用户是否存在
	testUsers := []string{"user123", "user999", "admin", "unknown", "guest"}
	for _, user := range testUsers {
		exists, err := cache.BloomExists(ctx, bloomKey, user)
		if err != nil {
			clog.Error("检查布隆过滤器失败", clog.String("user", user), clog.Err(err))
			continue
		}

		if exists {
			clog.Info("用户可能存在", clog.String("user", user))
		} else {
			clog.Info("用户肯定不存在", clog.String("user", user))
		}
	}

	clog.Info("布隆过滤器演示完成")
}

func demonstrateConcurrency(ctx context.Context) {
	clog.Info("=== 并发安全演示 ===")

	var wg sync.WaitGroup
	// 使用默认缓存进行并发测试

	// 并发写入
	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()

			key := fmt.Sprintf("user:%d", id)
			value := fmt.Sprintf("User %d Data", id)

			err := cache.Set(ctx, key, value, time.Hour)
			if err != nil {
				clog.Error("并发写入失败", clog.Int("id", id), clog.Err(err))
			} else {
				clog.Debug("并发写入成功", clog.Int("id", id))
			}
		}(i)
	}

	// 并发读取
	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()

			key := fmt.Sprintf("user:%d", id)

			value, err := cache.Get(ctx, key)
			if err != nil {
				clog.Error("并发读取失败", clog.Int("id", id), clog.Err(err))
			} else {
				clog.Debug("并发读取成功", clog.Int("id", id), clog.String("value", value))
			}
		}(i)
	}

	wg.Wait()
	clog.Info("并发安全演示完成")
}

func demonstrateErrorHandling(ctx context.Context) {
	clog.Info("=== 错误处理演示 ===")

	// 1. 键不存在错误
	_, err := cache.Get(ctx, "nonexistent:key")
	if err != nil {
		clog.Info("处理键不存在错误", clog.Err(err))
	}

	// 2. 无效参数错误
	err = cache.Set(ctx, "", "value", time.Hour)
	if err != nil {
		clog.Info("处理无效参数错误", clog.Err(err))
	}

	// 3. 超时上下文
	timeoutCtx, cancel := context.WithTimeout(ctx, time.Nanosecond)
	defer cancel()

	time.Sleep(time.Millisecond) // 确保上下文超时

	_, err = cache.Get(timeoutCtx, "some:key")
	if err != nil {
		clog.Info("处理上下文超时错误", clog.Err(err))
	}

	// 4. 锁获取失败
	lock1, err := cache.AcquireLock(ctx, "error:demo:lockimpl", time.Minute)
	if err != nil {
		clog.Error("获取第一个锁失败", clog.Err(err))
		return
	}
	defer lock1.Unlock(ctx)

	// 尝试获取同一个锁（应该失败）
	_, err = cache.AcquireLock(ctx, "error:demo:lockimpl", time.Minute)
	if err != nil {
		clog.Info("处理锁竞争错误", clog.Err(err))
	}

	clog.Info("错误处理演示完成")
}

func demonstratePerformanceOptimization(ctx context.Context) {
	clog.Info("=== 性能优化演示 ===")

	// 1. 自定义缓存实例 vs 全局缓存
	clog.Info("演示自定义缓存实例与全局缓存的性能对比")

	// 使用全局缓存
	start := time.Now()
	for i := 0; i < 1000; i++ {
		cache.Set(ctx, fmt.Sprintf("global_key:%d", i), "value", time.Hour)
	}
	duration1 := time.Since(start)
	clog.Info("全局缓存耗时", clog.Duration("duration", duration1))

	// 使用自定义缓存实例
	customCache, _ := cache.New(cache.DefaultConfig())
	start = time.Now()
	for i := 0; i < 1000; i++ {
		customCache.Set(ctx, fmt.Sprintf("custom_key:%d", i), "value", time.Hour)
	}
	duration2 := time.Since(start)
	clog.Info("自定义缓存实例耗时", clog.Duration("duration", duration2))

	improvement := float64(duration1-duration2) / float64(duration1) * 100
	clog.Info("性能对比", clog.Float64("improvement_percent", improvement))

	// 2. 批量操作
	clog.Info("演示批量操作的优势")

	// 单个删除
	start = time.Now()
	for i := 0; i < 100; i++ {
		cache.Del(ctx, fmt.Sprintf("single:%d", i))
	}
	duration1 = time.Since(start)
	clog.Info("单个删除耗时", clog.Duration("duration", duration1))

	// 批量删除
	keys := make([]string, 100)
	for i := 0; i < 100; i++ {
		keys[i] = fmt.Sprintf("batch:%d", i)
		cache.Set(ctx, keys[i], "value", time.Hour) // 先设置
	}

	start = time.Now()
	cache.Del(ctx, keys...)
	duration2 = time.Since(start)
	clog.Info("批量删除耗时", clog.Duration("duration", duration2))

	improvement = float64(duration1-duration2) / float64(duration1) * 100
	clog.Info("批量操作性能提升", clog.Float64("improvement_percent", improvement))

	// 3. 连接测试
	start = time.Now()
	err := cache.Ping(ctx)
	duration := time.Since(start)
	if err != nil {
		clog.Error("连接测试失败", clog.Err(err))
	} else {
		clog.Info("连接测试成功", clog.Duration("ping_time", duration))
	}

	clog.Info("性能优化演示完成")
}
