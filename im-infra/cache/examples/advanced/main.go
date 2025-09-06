package main

import (
	"context"
	"log"
	"sync"
	"time"

	"github.com/ceyewan/gochat/im-infra/cache"
	"github.com/ceyewan/gochat/im-infra/clog"
)

func main() {
	logger := clog.Module("cache-advanced-example")
	ctx := context.Background()
	cfg := cache.DefaultConfig()
	cfg.Addr = "localhost:6379"

	cacheClient, err := cache.New(ctx, cfg, cache.WithLogger(logger))
	if err != nil {
		log.Fatalf("无法创建缓存客户端: %v", err)
	}
	defer cacheClient.Close()

	log.Println("缓存客户端创建成功！")

	// --- 分布式锁演示 ---
	log.Println("\n--- 分布式锁演示 ---")
	lockKey := "my-distributed-lock"
	var wg sync.WaitGroup
	wg.Add(2)

	// 协程 1: 获取锁并执行任务
	go func() {
		defer wg.Done()
		lock, err := cacheClient.Lock(ctx, lockKey, 10*time.Second)
		if err != nil {
			log.Printf("协程 1: 获取锁失败: %v", err)
			return
		}
		if lock == nil {
			log.Printf("协程 1: 锁已被占用")
			return
		}
		defer lock.Unlock(ctx)

		log.Println("协程 1: 成功获取锁，执行任务...")
		time.Sleep(2 * time.Second)
		log.Println("协程 1: 任务完成，释放锁")
	}()

	// 协程 2: 尝试获取同一个锁
	go func() {
		defer wg.Done()
		time.Sleep(500 * time.Millisecond) // 确保协程1先获取锁
		log.Println("协程 2: 尝试获取锁...")
		lock, err := cacheClient.Lock(ctx, lockKey, 10*time.Second)
		if err != nil {
			log.Printf("协程 2: 获取锁时发生错误: %v", err)
			return
		}
		if lock == nil {
			log.Println("协程 2: 获取锁失败，锁已被其他协程占用")
		} else {
			log.Println("协程 2: 意外地获取了锁！")
			lock.Unlock(ctx)
		}
	}()

	wg.Wait()

	// --- 布隆过滤器演示 ---
	log.Println("\n--- 布隆过滤器演示 ---")
	bfKey := "user-blacklist"
	// 初始化布隆过滤器，错误率 0.1%，容量 1000
	err = cacheClient.BFInit(ctx, bfKey, 0.001, 1000)
	if err != nil {
		log.Fatalf("初始化布隆过滤器失败: %v", err)
	}
	log.Printf("成功初始化布隆过滤器 '%s'", bfKey)

	// 添加用户到黑名单
	blacklistedUser := "bad-user-123"
	err = cacheClient.BFAdd(ctx, bfKey, blacklistedUser)
	if err != nil {
		log.Fatalf("添加用户到布隆过滤器失败: %v", err)
	}
	log.Printf("成功将用户 '%s' 添加到布隆过滤器", blacklistedUser)

	// 检查存在的用户
	exists, err := cacheClient.BFExists(ctx, bfKey, blacklistedUser)
	if err != nil {
		log.Fatalf("检查布隆过滤器失败: %v", err)
	}
	log.Printf("用户 '%s' 是否存在于布隆过滤器中: %v", blacklistedUser, exists)
	if !exists {
		log.Fatalf("布隆过滤器检查失败：应存在但未找到！")
	}

	// 检查不存在的用户
	goodUser := "good-user-456"
	exists, err = cacheClient.BFExists(ctx, bfKey, goodUser)
	if err != nil {
		log.Fatalf("检查布隆过滤器失败: %v", err)
	}
	log.Printf("用户 '%s' 是否存在于布隆过滤器中: %v", goodUser, exists)
	if exists {
		log.Printf("布隆过滤器发生误判（这是正常现象）")
	}

	log.Println("\n高级功能示例执行完毕！")
}
