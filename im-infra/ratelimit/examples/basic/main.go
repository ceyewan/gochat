package main

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/ceyewan/gochat/im-infra/ratelimit"
)

func main() {
	fmt.Println("=== RateLimit 组件基本使用示例 ===")

	// 定义默认规则，以防无法从配置中心加载
	defaultRules := map[string]ratelimit.Rule{
		"api_requests": {Rate: 5, Capacity: 10},  // 每秒 5 个请求，突发 10 个
		"user_actions": {Rate: 2, Capacity: 5},   // 每秒 2 个操作，突发 5 个
		"login":        {Rate: 1, Capacity: 3},   // 每秒 1 次登录，突发 3 次
		"sms_send":     {Rate: 0.1, Capacity: 1}, // 每 10 秒 1 条短信，突发 1 条
	}

	// 创建限流器实例
	ctx := context.Background()
	limiter, err := ratelimit.New(
		ctx,
		"demo_service", // 服务名称，用于配置中心路径
		ratelimit.WithDefaultRules(defaultRules),
		ratelimit.WithRuleRefreshInterval(30*time.Second), // 30秒刷新一次配置
	)
	if err != nil {
		log.Fatalf("创建限流器失败: %v", err)
	}
	defer limiter.Close()

	// 示例1: API 请求限流
	fmt.Println("\n--- 示例1: API 请求限流 ---")
	testAPIRateLimit(ctx, limiter)

	// 示例2: 用户操作限流
	fmt.Println("\n--- 示例2: 用户操作限流 ---")
	testUserActionRateLimit(ctx, limiter)

	// 示例3: 登录限流
	fmt.Println("\n--- 示例3: 登录限流 ---")
	testLoginRateLimit(ctx, limiter)

	// 示例4: 短信发送限流
	fmt.Println("\n--- 示例4: 短信发送限流 ---")
	testSMSRateLimit(ctx, limiter)

	// 示例5: 批量请求限流
	fmt.Println("\n--- 示例5: 批量请求限流 ---")
	testBatchRateLimit(ctx, limiter)

	// 示例6: 获取限流统计信息
	fmt.Println("\n--- 示例6: 限流统计信息 ---")
	testStatistics(ctx, limiter)

	fmt.Println("\n=== 示例完成 ===")
}

// testAPIRateLimit 测试 API 请求限流
func testAPIRateLimit(ctx context.Context, limiter ratelimit.RateLimiter) {
	resource := ratelimit.BuildAPIResourceKey("/api/users")
	rule := "api_requests"

	fmt.Printf("测试 API 限流 (资源: %s, 规则: %s)\n", resource, rule)
	fmt.Println("规则: 每秒 5 个请求，突发 10 个")

	// 连续发送 12 个请求，前 10 个应该成功，后 2 个被限流
	for i := 1; i <= 12; i++ {
		allowed, err := limiter.Allow(ctx, resource, rule)
		if err != nil {
			fmt.Printf("请求 %d 出错: %v\n", i, err)
			continue
		}

		status := "✅ 允许"
		if !allowed {
			status = "❌ 限流"
		}
		fmt.Printf("  请求 %d: %s\n", i, status)

		time.Sleep(50 * time.Millisecond) // 模拟请求间隔
	}
}

// testUserActionRateLimit 测试用户操作限流
func testUserActionRateLimit(ctx context.Context, limiter ratelimit.RateLimiter) {
	userID := "user123"
	resource := ratelimit.BuildUserResourceKey(userID)
	rule := "user_actions"

	fmt.Printf("测试用户操作限流 (用户: %s, 规则: %s)\n", userID, rule)
	fmt.Println("规则: 每秒 2 个操作，突发 5 个")

	actions := []string{"发帖", "点赞", "评论", "分享", "收藏", "关注", "举报"}

	for i, action := range actions {
		allowed, err := limiter.Allow(ctx, resource, rule)
		if err != nil {
			fmt.Printf("用户操作 '%s' 出错: %v\n", action, err)
			continue
		}

		status := "✅ 允许"
		if !allowed {
			status = "❌ 限流"
		}
		fmt.Printf("  用户操作 '%s': %s\n", action, status)

		if i == 4 { // 在第5个操作后等待一段时间，让令牌桶补充
			fmt.Println("  --- 等待 1 秒，让令牌桶补充 ---")
			time.Sleep(1 * time.Second)
		}
	}
}

// testLoginRateLimit 测试登录限流
func testLoginRateLimit(ctx context.Context, limiter ratelimit.RateLimiter) {
	ip := "192.168.1.100"
	resource := ratelimit.BuildIPResourceKey(ip)
	rule := "login"

	fmt.Printf("测试登录限流 (IP: %s, 规则: %s)\n", ip, rule)
	fmt.Println("规则: 每秒 1 次登录，突发 3 次")

	for i := 1; i <= 5; i++ {
		allowed, err := limiter.Allow(ctx, resource, rule)
		if err != nil {
			fmt.Printf("登录尝试 %d 出错: %v\n", i, err)
			continue
		}

		status := "✅ 允许"
		if !allowed {
			status = "❌ 限流"
		}
		fmt.Printf("  登录尝试 %d: %s\n", i, status)

		time.Sleep(200 * time.Millisecond)
	}
}

// testSMSRateLimit 测试短信发送限流
func testSMSRateLimit(ctx context.Context, limiter ratelimit.RateLimiter) {
	phoneNumber := "13800138000"
	resource := ratelimit.BuildResourceKey("phone", phoneNumber)
	rule := "sms_send"

	fmt.Printf("测试短信发送限流 (手机: %s, 规则: %s)\n", phoneNumber, rule)
	fmt.Println("规则: 每 10 秒 1 条短信，突发 1 条")

	for i := 1; i <= 3; i++ {
		allowed, err := limiter.Allow(ctx, resource, rule)
		if err != nil {
			fmt.Printf("短信发送 %d 出错: %v\n", i, err)
			continue
		}

		status := "✅ 允许"
		if !allowed {
			status = "❌ 限流"
		}
		fmt.Printf("  短信发送 %d: %s\n", i, status)

		if i == 1 && allowed {
			fmt.Println("  --- 等待 2 秒后再次发送 ---")
			time.Sleep(2 * time.Second)
		}
	}
}

// testBatchRateLimit 测试批量请求限流
func testBatchRateLimit(ctx context.Context, limiter ratelimit.RateLimiter) {
	fmt.Println("测试批量请求限流")

	// 构造批量请求
	requests := []ratelimit.RateLimitRequest{
		{Resource: ratelimit.BuildUserResourceKey("user1"), RuleName: "api_requests", Count: 1},
		{Resource: ratelimit.BuildUserResourceKey("user2"), RuleName: "api_requests", Count: 2},
		{Resource: ratelimit.BuildIPResourceKey("10.0.0.1"), RuleName: "login", Count: 1},
		{Resource: ratelimit.BuildAPIResourceKey("/api/upload"), RuleName: "user_actions", Count: 1},
		{Resource: ratelimit.BuildResourceKey("device", "device123"), RuleName: "api_requests", Count: 1},
	}

	results, err := limiter.BatchAllow(ctx, requests)
	if err != nil {
		fmt.Printf("批量请求出错: %v\n", err)
		return
	}

	fmt.Println("批量请求结果:")
	for i, result := range results {
		req := requests[i]
		status := "✅ 允许"
		if !result {
			status = "❌ 限流"
		}
		fmt.Printf("  请求 %d (%s, %s): %s\n", i+1, req.Resource, req.RuleName, status)
	}
}

// testStatistics 测试获取统计信息
func testStatistics(ctx context.Context, limiter ratelimit.RateLimiter) {
	fmt.Println("测试获取限流统计信息")

	// 先发送一些请求生成统计数据
	resource := ratelimit.BuildUserResourceKey("stats_user")
	rule := "api_requests"

	fmt.Println("发送测试请求以生成统计数据...")
	for i := 0; i < 8; i++ {
		limiter.Allow(ctx, resource, rule)
		time.Sleep(100 * time.Millisecond)
	}

	// 获取统计信息
	stats, err := limiter.GetStatistics(ctx, resource, rule)
	if err != nil {
		fmt.Printf("获取统计信息出错: %v\n", err)
		return
	}

	// 显示统计信息
	fmt.Printf("统计信息 (资源: %s, 规则: %s):\n", resource, rule)
	fmt.Printf("  总请求数: %d\n", stats.TotalRequests)
	fmt.Printf("  允许请求数: %d\n", stats.AllowedRequests)
	fmt.Printf("  拒绝请求数: %d\n", stats.DeniedRequests)
	fmt.Printf("  当前令牌数: %d\n", stats.CurrentTokens)
	fmt.Printf("  成功率: %.2f%%\n", stats.SuccessRate*100)
	fmt.Printf("  最后更新时间: %s\n", stats.LastUpdated.Format("2006-01-02 15:04:05"))
}
