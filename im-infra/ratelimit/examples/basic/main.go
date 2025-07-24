package main

import (
	"context"
	"fmt"
	"time"

	"github.com/ceyewan/gochat/im-infra/ratelimit"
)

func main() {
	// 定义默认规则，以防无法从配置中心加载
	defaultRules := map[string]ratelimit.Rule{
		"api_requests": {Rate: 5, Capacity: 5},   // 每秒 5 个请求
		"ws_messages":  {Rate: 20, Capacity: 40}, // 每秒 20 条消息
	}

	// 初始化限流器
	// 在真实应用中，ctx 应该来自应用的生命周期管理
	limiter, err := ratelimit.New(
		context.Background(),
		"my_awesome_app", // 这将用于在 etcd 中查找 /config/{env}/my_awesome_app/ratelimit/*
		ratelimit.WithDefaultRules(defaultRules),
	)
	if err != nil {
		panic(err)
	}
	defer limiter.Close()

	// 模拟 API 请求限流
	fmt.Println("--- 模拟 API 请求 (5次/秒) ---")
	for i := 0; i < 7; i++ {
		allowed, err := limiter.Allow(context.Background(), "user_ip:192.168.1.100", "api_requests")
		if err != nil {
			fmt.Printf("请求 %d 出错: %v\n", i+1, err)
		}

		if allowed {
			fmt.Printf("请求 %d: ✅ 允许\n", i+1)
		} else {
			fmt.Printf("请求 %d: ❌ 拒绝\n", i+1)
		}
		time.Sleep(100 * time.Millisecond) // 每 0.1 秒发一个请求
	}

	fmt.Println("\n--- 等待 1 秒让令牌桶回满 ---")
	time.Sleep(1 * time.Second)

	// 模拟 WebSocket 消息限流
	fmt.Println("\n--- 模拟 WebSocket 消息 (20次/秒) ---")
	for i := 0; i < 25; i++ {
		userID := "user_abc"
		allowed, _ := limiter.Allow(context.Background(), userID, "ws_messages")
		if allowed {
			fmt.Printf("用户 %s 的消息 %d: ✅ 允许\n", userID, i+1)
		} else {
			fmt.Printf("用户 %s 的消息 %d: ❌ 拒绝\n", userID, i+1)
		}
		time.Sleep(40 * time.Millisecond) // 每 0.04 秒发一个请求
	}
}
