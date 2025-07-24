package main

import (
	"context"
	"fmt"
	"time"

	coordination "github.com/ceyewan/gochat/im-infra/coordination"
)

func main() {
	fmt.Println("=== 租约调试测试 ===")

	// 创建协调器
	opts := coordination.DefaultCoordinatorOptions()
	coord, err := coordination.NewCoordinator(opts)
	if err != nil {
		fmt.Printf("创建协调器失败: %v\n", err)
		return
	}
	defer coord.Close()

	ctx := context.Background()
	lockService := coord.Lock()

	// 获取锁
	fmt.Println("获取锁...")
	lock, err := lockService.Acquire(ctx, "debug-lock", 30*time.Second)
	if err != nil {
		fmt.Printf("获取锁失败: %v\n", err)
		return
	}
	fmt.Printf("锁获取成功: %s\n", lock.Key())

	// 立即查询 TTL
	fmt.Println("立即查询 TTL...")
	ttl, err := lock.TTL(ctx)
	if err != nil {
		fmt.Printf("TTL 查询失败: %v\n", err)
	} else {
		fmt.Printf("TTL: %v\n", ttl)
	}

	// 等待 1 秒后再查询
	fmt.Println("等待 1 秒后再查询...")
	time.Sleep(1 * time.Second)
	ttl, err = lock.TTL(ctx)
	if err != nil {
		fmt.Printf("TTL 查询失败: %v\n", err)
	} else {
		fmt.Printf("TTL: %v\n", ttl)
	}

	// 等待 5 秒后再查询
	fmt.Println("等待 5 秒后再查询...")
	time.Sleep(5 * time.Second)
	ttl, err = lock.TTL(ctx)
	if err != nil {
		fmt.Printf("TTL 查询失败: %v\n", err)
	} else {
		fmt.Printf("TTL: %v\n", ttl)
	}

	// 释放锁
	fmt.Println("释放锁...")
	err = lock.Unlock(ctx)
	if err != nil {
		fmt.Printf("释放锁失败: %v\n", err)
	} else {
		fmt.Println("锁释放成功")
	}

	fmt.Println("=== 测试完成 ===")
}
