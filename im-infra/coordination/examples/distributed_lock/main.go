package main

import (
	"context"
	"fmt"
	"log"
	"sync"
	"time"

	"github.com/ceyewan/gochat/im-infra/coordination"
)

func main() {
	fmt.Println("=== 分布式锁示例 ===")

	// 创建协调器
	cfg := coordination.ExampleConfig()
	coordinator, err := coordination.New(cfg)
	if err != nil {
		log.Printf("创建协调器失败 (请确保 etcd 正在运行): %v", err)
		return
	}
	defer coordinator.Close()

	ctx := context.Background()

	// 测试连接
	if err := coordinator.Ping(ctx); err != nil {
		log.Printf("连接 etcd 失败: %v", err)
		return
	}
	fmt.Println("✓ 连接 etcd 成功")

	// 获取分布式锁管理器
	lockManager := coordinator.Lock()

	// 1. 基础锁示例
	fmt.Println("\n1. 基础分布式锁示例:")
	basicLockDemo(ctx, lockManager)

	// 2. 可重入锁示例
	fmt.Println("\n2. 可重入锁示例:")
	reentrantLockDemo(ctx, lockManager)

	// 3. 读写锁示例
	fmt.Println("\n3. 读写锁示例:")
	readWriteLockDemo(ctx, lockManager)

	// 4. 并发锁竞争示例
	fmt.Println("\n4. 并发锁竞争示例:")
	concurrentLockDemo(ctx, lockManager)

	// 5. 锁超时示例
	fmt.Println("\n5. 锁超时示例:")
	lockTimeoutDemo(ctx, lockManager)

	// 6. 模块锁示例
	fmt.Println("\n6. 模块锁示例:")
	moduleLockDemo(ctx)

	fmt.Println("\n=== 分布式锁示例完成 ===")
}

func basicLockDemo(ctx context.Context, lockManager coordination.DistributedLock) {
	lockKey := "basic-lock-demo"

	// 获取锁
	lock, err := lockManager.Acquire(ctx, lockKey, 30*time.Second)
	if err != nil {
		log.Printf("获取锁失败: %v", err)
		return
	}
	fmt.Printf("✓ 获取锁成功: %s\n", lock.Key())

	// 检查锁状态
	held, err := lock.IsHeld(ctx)
	if err != nil {
		log.Printf("检查锁状态失败: %v", err)
	} else {
		fmt.Printf("✓ 锁状态: %t\n", held)
	}

	// 获取锁的 TTL
	ttl, err := lock.TTL(ctx)
	if err != nil {
		log.Printf("获取锁 TTL 失败: %v", err)
	} else {
		fmt.Printf("✓ 锁 TTL: %v\n", ttl)
	}

	// 模拟临界区操作
	fmt.Println("  执行临界区操作...")
	time.Sleep(2 * time.Second)

	// 续期锁
	if err := lock.Renew(ctx, 60*time.Second); err != nil {
		log.Printf("续期锁失败: %v", err)
	} else {
		fmt.Println("✓ 锁续期成功")
	}

	// 释放锁
	if err := lock.Release(ctx); err != nil {
		log.Printf("释放锁失败: %v", err)
	} else {
		fmt.Println("✓ 锁释放成功")
	}
}

func reentrantLockDemo(ctx context.Context, lockManager coordination.DistributedLock) {
	lockKey := "reentrant-lock-demo"

	// 获取可重入锁
	lock, err := lockManager.AcquireReentrant(ctx, lockKey, 30*time.Second)
	if err != nil {
		log.Printf("获取可重入锁失败: %v", err)
		return
	}
	fmt.Printf("✓ 获取可重入锁成功: %s (获取次数: %d)\n", lock.Key(), lock.AcquireCount())

	// 再次获取锁（可重入）
	if err := lock.Acquire(ctx); err != nil {
		log.Printf("再次获取锁失败: %v", err)
	} else {
		fmt.Printf("✓ 再次获取锁成功 (获取次数: %d)\n", lock.AcquireCount())
	}

	// 第三次获取锁
	if err := lock.Acquire(ctx); err != nil {
		log.Printf("第三次获取锁失败: %v", err)
	} else {
		fmt.Printf("✓ 第三次获取锁成功 (获取次数: %d)\n", lock.AcquireCount())
	}

	// 逐步释放锁
	for lock.AcquireCount() > 0 {
		count := lock.AcquireCount()
		if err := lock.Release(ctx); err != nil {
			log.Printf("释放锁失败: %v", err)
			break
		}
		if lock.AcquireCount() == 0 {
			fmt.Printf("✓ 锁完全释放 (从获取次数 %d 到 0)\n", count)
		} else {
			fmt.Printf("✓ 部分释放锁 (获取次数: %d -> %d)\n", count, lock.AcquireCount())
		}
	}
}

func readWriteLockDemo(ctx context.Context, lockManager coordination.DistributedLock) {
	lockKey := "rw-lock-demo"

	// 获取读锁
	readLock1, err := lockManager.AcquireReadLock(ctx, lockKey, 30*time.Second)
	if err != nil {
		log.Printf("获取读锁1失败: %v", err)
		return
	}
	fmt.Printf("✓ 获取读锁1成功: %s\n", readLock1.Key())

	// 获取另一个读锁（应该成功，因为读锁可以并发）
	readLock2, err := lockManager.AcquireReadLock(ctx, lockKey, 30*time.Second)
	if err != nil {
		log.Printf("获取读锁2失败: %v", err)
	} else {
		fmt.Printf("✓ 获取读锁2成功: %s\n", readLock2.Key())
	}

	// 模拟读操作
	fmt.Println("  执行并发读操作...")
	time.Sleep(2 * time.Second)

	// 释放读锁
	if readLock2 != nil {
		if err := readLock2.Release(ctx); err != nil {
			log.Printf("释放读锁2失败: %v", err)
		} else {
			fmt.Println("✓ 读锁2释放成功")
		}
	}

	if err := readLock1.Release(ctx); err != nil {
		log.Printf("释放读锁1失败: %v", err)
	} else {
		fmt.Println("✓ 读锁1释放成功")
	}

	// 获取写锁
	writeLock, err := lockManager.AcquireWriteLock(ctx, lockKey, 30*time.Second)
	if err != nil {
		log.Printf("获取写锁失败: %v", err)
		return
	}
	fmt.Printf("✓ 获取写锁成功: %s\n", writeLock.Key())

	// 模拟写操作
	fmt.Println("  执行独占写操作...")
	time.Sleep(2 * time.Second)

	// 释放写锁
	if err := writeLock.Release(ctx); err != nil {
		log.Printf("释放写锁失败: %v", err)
	} else {
		fmt.Println("✓ 写锁释放成功")
	}
}

func concurrentLockDemo(ctx context.Context, lockManager coordination.DistributedLock) {
	lockKey := "concurrent-lock-demo"
	var wg sync.WaitGroup

	// 启动多个 goroutine 竞争同一个锁
	for i := 0; i < 3; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()

			fmt.Printf("  Goroutine %d: 尝试获取锁...\n", id)
			lock, err := lockManager.Acquire(ctx, lockKey, 10*time.Second)
			if err != nil {
				fmt.Printf("  Goroutine %d: 获取锁失败: %v\n", id, err)
				return
			}

			fmt.Printf("  Goroutine %d: ✓ 获取锁成功\n", id)

			// 模拟工作
			time.Sleep(2 * time.Second)
			fmt.Printf("  Goroutine %d: 完成工作\n", id)

			// 释放锁
			if err := lock.Release(ctx); err != nil {
				fmt.Printf("  Goroutine %d: 释放锁失败: %v\n", id, err)
			} else {
				fmt.Printf("  Goroutine %d: ✓ 释放锁成功\n", id)
			}
		}(i)
	}

	wg.Wait()
	fmt.Println("✓ 所有 goroutine 完成")
}

func lockTimeoutDemo(ctx context.Context, lockManager coordination.DistributedLock) {
	lockKey := "timeout-lock-demo"

	// 获取一个锁
	lock1, err := lockManager.Acquire(ctx, lockKey, 5*time.Second)
	if err != nil {
		log.Printf("获取锁1失败: %v", err)
		return
	}
	fmt.Printf("✓ 获取锁1成功: %s\n", lock1.Key())

	// 尝试获取同一个锁（应该超时）
	timeoutCtx, cancel := context.WithTimeout(ctx, 2*time.Second)
	defer cancel()

	fmt.Println("  尝试获取已被占用的锁（2秒超时）...")
	start := time.Now()
	lock2, err := lockManager.Acquire(timeoutCtx, lockKey, 5*time.Second)
	duration := time.Since(start)

	if err != nil {
		fmt.Printf("✓ 预期的超时错误: %v (耗时: %v)\n", err, duration)
	} else {
		fmt.Printf("意外获取到锁2: %s\n", lock2.Key())
		lock2.Release(ctx)
	}

	// 等待锁1过期
	fmt.Println("  等待锁1自动过期...")
	time.Sleep(6 * time.Second)

	// 现在应该能够获取锁了
	lock3, err := lockManager.Acquire(ctx, lockKey, 5*time.Second)
	if err != nil {
		log.Printf("获取锁3失败: %v", err)
	} else {
		fmt.Printf("✓ 锁过期后获取新锁成功: %s\n", lock3.Key())
		lock3.Release(ctx)
	}

	// 清理
	if lock1 != nil {
		lock1.Release(ctx)
	}
}

func moduleLockDemo(ctx context.Context) {
	// 使用模块特定的协调器
	userServiceCoordinator := coordination.Module("user-service")
	orderServiceCoordinator := coordination.Module("order-service")

	userLockManager := userServiceCoordinator.Lock()
	orderLockManager := orderServiceCoordinator.Lock()

	// 不同模块可以使用相同的锁键名，但实际上是隔离的
	lockKey := "resource-lock"

	userLock, err := userLockManager.Acquire(ctx, lockKey, 10*time.Second)
	if err != nil {
		log.Printf("用户服务获取锁失败: %v", err)
		return
	}
	fmt.Printf("✓ 用户服务获取锁成功: %s\n", userLock.Key())

	orderLock, err := orderLockManager.Acquire(ctx, lockKey, 10*time.Second)
	if err != nil {
		log.Printf("订单服务获取锁失败: %v", err)
	} else {
		fmt.Printf("✓ 订单服务获取锁成功: %s\n", orderLock.Key())
		orderLock.Release(ctx)
	}

	userLock.Release(ctx)
}
