package ratelimit_test

import (
	"context"
	"testing"
	"time"

	"github.com/ceyewan/gochat/im-infra/cache"
	"github.com/ceyewan/gochat/im-infra/ratelimit"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRateLimiter_BasicFunctionality(t *testing.T) {
	// 使用测试配置初始化缓存
	cacheClient, err := cache.New(context.Background(), cache.DefaultConfig())
	require.NoError(t, err)
	defer cacheClient.Close()

	// 定义默认规则用于测试
	defaultRules := map[string]ratelimit.Rule{
		"test_rule": {
			Rate:     5, // 每秒 5 个令牌
			Capacity: 5, // 桶容量 5
		},
	}

	// 创建限流器实例
	limiter, err := ratelimit.New(
		context.Background(),
		"test_service",
		ratelimit.WithCacheClient(cacheClient),
		ratelimit.WithDefaultRules(defaultRules),
	)
	require.NoError(t, err)
	defer limiter.Close()

	ctx := context.Background()
	resource := "user:123"
	ruleName := "test_rule"

	// 连续请求 5 次，应该都成功
	for i := 0; i < 5; i++ {
		allowed, err := limiter.Allow(ctx, resource, ruleName)
		require.NoError(t, err)
		assert.True(t, allowed, "请求 %d 应该被允许", i+1)
	}

	// 第 6 次请求，令牌耗尽，应该失败
	allowed, err := limiter.Allow(ctx, resource, ruleName)
	require.NoError(t, err)
	assert.False(t, allowed, "第 6 次请求应该被拒绝")

	// 等待 1 秒，应该会补充 5 个令牌
	time.Sleep(1100 * time.Millisecond)

	// 再次请求 5 次，应该都成功
	for i := 0; i < 5; i++ {
		allowed, err := limiter.Allow(ctx, resource, ruleName)
		require.NoError(t, err)
		assert.True(t, allowed, "补充令牌后的请求 %d 应该被允许", i+1)
	}
}

func TestRateLimiter_AllowN(t *testing.T) {
	cacheClient, err := cache.New(context.Background(), cache.DefaultConfig())
	require.NoError(t, err)
	defer cacheClient.Close()

	defaultRules := map[string]ratelimit.Rule{
		"batch_rule": {
			Rate:     10, // 每秒 10 个令牌
			Capacity: 20, // 桶容量 20
		},
	}

	limiter, err := ratelimit.New(
		context.Background(),
		"test_service",
		ratelimit.WithCacheClient(cacheClient),
		ratelimit.WithDefaultRules(defaultRules),
	)
	require.NoError(t, err)
	defer limiter.Close()

	ctx := context.Background()
	resource := "batch:456"
	ruleName := "batch_rule"

	// 一次性请求 15 个令牌，应该成功（桶初始有 20 个）
	allowed, err := limiter.AllowN(ctx, resource, ruleName, 15)
	require.NoError(t, err)
	assert.True(t, allowed)

	// 再次请求 10 个令牌，应该失败（只剩 5 个）
	allowed, err = limiter.AllowN(ctx, resource, ruleName, 10)
	require.NoError(t, err)
	assert.False(t, allowed)

	// 请求 3 个令牌，应该成功
	allowed, err = limiter.AllowN(ctx, resource, ruleName, 3)
	require.NoError(t, err)
	assert.True(t, allowed)
}

func TestRateLimiter_BatchAllow(t *testing.T) {
	cacheClient, err := cache.New(context.Background(), cache.DefaultConfig())
	require.NoError(t, err)
	defer cacheClient.Close()

	defaultRules := map[string]ratelimit.Rule{
		"api_rule": {Rate: 5, Capacity: 10},
		"ws_rule":  {Rate: 20, Capacity: 40},
	}

	limiter, err := ratelimit.New(
		context.Background(),
		"test_service",
		ratelimit.WithCacheClient(cacheClient),
		ratelimit.WithDefaultRules(defaultRules),
	)
	require.NoError(t, err)
	defer limiter.Close()

	ctx := context.Background()

	// 批量限流请求
	requests := []ratelimit.RateLimitRequest{
		{Resource: "user:1", RuleName: "api_rule", Count: 1},
		{Resource: "user:2", RuleName: "api_rule", Count: 2},
		{Resource: "user:3", RuleName: "ws_rule", Count: 5},
		{Resource: "user:4", RuleName: "api_rule", Count: 3},
	}

	results, err := limiter.BatchAllow(ctx, requests)
	require.NoError(t, err)
	require.Len(t, results, 4)

	// 初始状态下，所有请求都应该被允许
	for i, allowed := range results {
		assert.True(t, allowed, "批量请求 %d 应该被允许", i)
	}
}

func TestRateLimiter_Statistics(t *testing.T) {
	cacheClient, err := cache.New(context.Background(), cache.DefaultConfig())
	require.NoError(t, err)
	defer cacheClient.Close()

	defaultRules := map[string]ratelimit.Rule{
		"stats_rule": {Rate: 1, Capacity: 1},
	}

	limiter, err := ratelimit.New(
		context.Background(),
		"test_service",
		ratelimit.WithCacheClient(cacheClient),
		ratelimit.WithDefaultRules(defaultRules),
	)
	require.NoError(t, err)
	defer limiter.Close()

	ctx := context.Background()
	resource := "stats:789"
	ruleName := "stats_rule"

	// 发送几个请求以生成统计数据
	for i := 0; i < 3; i++ {
		limiter.Allow(ctx, resource, ruleName)
		time.Sleep(50 * time.Millisecond)
	}

	// 获取统计信息
	stats, err := limiter.GetStatistics(ctx, resource, ruleName)
	require.NoError(t, err)
	require.NotNil(t, stats)

	assert.Equal(t, resource, stats.Resource)
	assert.Equal(t, ruleName, stats.RuleName)
	assert.Equal(t, int64(3), stats.TotalRequests)
	assert.GreaterOrEqual(t, stats.AllowedRequests, int64(1))
	assert.Equal(t, stats.TotalRequests-stats.AllowedRequests, stats.DeniedRequests)
}

func TestRateLimiter_UnknownRule(t *testing.T) {
	cacheClient, err := cache.New(context.Background(), cache.DefaultConfig())
	require.NoError(t, err)
	defer cacheClient.Close()

	limiter, err := ratelimit.New(
		context.Background(),
		"test_service",
		ratelimit.WithCacheClient(cacheClient),
		ratelimit.WithDefaultRules(map[string]ratelimit.Rule{}),
	)
	require.NoError(t, err)
	defer limiter.Close()

	ctx := context.Background()

	// 使用未知规则，应该默认允许
	allowed, err := limiter.Allow(ctx, "user:999", "unknown_rule")
	require.NoError(t, err)
	assert.True(t, allowed, "未知规则应该默认允许")
}

func TestRateLimiter_ErrorHandling(t *testing.T) {
	cacheClient, err := cache.New(context.Background(), cache.DefaultConfig())
	require.NoError(t, err)
	defer cacheClient.Close()

	limiter, err := ratelimit.New(
		context.Background(),
		"test_service",
		ratelimit.WithCacheClient(cacheClient),
		ratelimit.WithDefaultRules(map[string]ratelimit.Rule{
			"error_rule": {Rate: 1, Capacity: 1},
		}),
	)
	require.NoError(t, err)
	defer limiter.Close()

	ctx := context.Background()

	// 测试空资源
	allowed, err := limiter.Allow(ctx, "", "error_rule")
	require.NoError(t, err)
	assert.True(t, allowed, "空资源应该默认允许（错误处理）")

	// 测试空规则名
	allowed, err = limiter.Allow(ctx, "user:error", "")
	require.NoError(t, err)
	assert.True(t, allowed, "空规则名应该默认允许")
}

func TestRateLimiter_Concurrent(t *testing.T) {
	cacheClient, err := cache.New(context.Background(), cache.DefaultConfig())
	require.NoError(t, err)
	defer cacheClient.Close()

	defaultRules := map[string]ratelimit.Rule{
		"concurrent_rule": {Rate: 10, Capacity: 10},
	}

	limiter, err := ratelimit.New(
		context.Background(),
		"test_service",
		ratelimit.WithCacheClient(cacheClient),
		ratelimit.WithDefaultRules(defaultRules),
	)
	require.NoError(t, err)
	defer limiter.Close()

	ctx := context.Background()
	resource := "concurrent:test"
	ruleName := "concurrent_rule"

	// 并发测试
	const numGoroutines = 20
	results := make(chan bool, numGoroutines)

	for i := 0; i < numGoroutines; i++ {
		go func() {
			allowed, err := limiter.Allow(ctx, resource, ruleName)
			if err != nil {
				results <- false
				return
			}
			results <- allowed
		}()
	}

	// 收集结果
	allowedCount := 0
	for i := 0; i < numGoroutines; i++ {
		if <-results {
			allowedCount++
		}
	}

	// 由于桶容量为10，最多只能允许10个请求
	assert.LessOrEqual(t, allowedCount, 10, "并发请求中应该只允许不超过10个")
	assert.Greater(t, allowedCount, 0, "应该至少允许1个请求")
}

func TestSimpleRateLimiter(t *testing.T) {
	rules := map[string]ratelimit.Rule{
		"simple": {Rate: 2, Capacity: 5},
	}

	limiter, err := ratelimit.SimpleRateLimiter(
		context.Background(),
		"simple_service",
		rules,
	)
	require.NoError(t, err)
	defer limiter.Close()

	ctx := context.Background()

	// 测试简单限流器
	for i := 0; i < 5; i++ {
		allowed, err := limiter.Allow(ctx, "simple:test", "simple")
		require.NoError(t, err)
		assert.True(t, allowed, "简单限流器请求 %d 应该被允许", i+1)
	}

	// 第6次请求应该被拒绝
	allowed, err := limiter.Allow(ctx, "simple:test", "simple")
	require.NoError(t, err)
	assert.False(t, allowed, "简单限流器第6次请求应该被拒绝")
}

func TestValidateRule(t *testing.T) {
	// 测试有效规则
	validRule := ratelimit.Rule{Rate: 10, Capacity: 20}
	err := ratelimit.ValidateRule(validRule)
	assert.NoError(t, err)

	// 测试无效速率
	invalidRateRule := ratelimit.Rule{Rate: 0, Capacity: 20}
	err = ratelimit.ValidateRule(invalidRateRule)
	assert.Error(t, err)
	assert.Equal(t, ratelimit.ErrInvalidRate, err)

	// 测试无效容量
	invalidCapacityRule := ratelimit.Rule{Rate: 10, Capacity: 0}
	err = ratelimit.ValidateRule(invalidCapacityRule)
	assert.Error(t, err)
	assert.Equal(t, ratelimit.ErrInvalidCapacity, err)
}

func TestCreateDefaultRules(t *testing.T) {
	rules := ratelimit.CreateDefaultRules()
	assert.NotEmpty(t, rules)

	// 检查一些预期的规则
	assert.Contains(t, rules, "api_default")
	assert.Contains(t, rules, "user_action")
	assert.Contains(t, rules, "login")

	// 验证规则的有效性
	for name, rule := range rules {
		err := ratelimit.ValidateRule(rule)
		assert.NoError(t, err, "默认规则 %s 应该是有效的", name)
	}
}

func TestGetRuleByScenario(t *testing.T) {
	// 测试已知场景
	rule, exists := ratelimit.GetRuleByScenario("web_api_high")
	assert.True(t, exists)
	assert.Equal(t, float64(1000), rule.Rate)
	assert.Equal(t, int64(2000), rule.Capacity)

	// 测试未知场景
	_, exists = ratelimit.GetRuleByScenario("unknown_scenario")
	assert.False(t, exists)
}

func TestBuildResourceKeys(t *testing.T) {
	// 测试基本资源键构建
	key := ratelimit.BuildResourceKey("user", "123")
	assert.Equal(t, "user:123", key)

	// 测试用户资源键
	userKey := ratelimit.BuildUserResourceKey("456")
	assert.Equal(t, "user:456", userKey)

	// 测试IP资源键
	ipKey := ratelimit.BuildIPResourceKey("192.168.1.1")
	assert.Equal(t, "ip:192.168.1.1", ipKey)

	// 测试API资源键
	apiKey := ratelimit.BuildAPIResourceKey("/api/users")
	assert.Equal(t, "api:/api/users", apiKey)

	// 测试设备资源键
	deviceKey := ratelimit.BuildDeviceResourceKey("device123")
	assert.Equal(t, "device:device123", deviceKey)
}

func TestRateLimiter_Close(t *testing.T) {
	cacheClient, err := cache.New(context.Background(), cache.DefaultConfig())
	require.NoError(t, err)
	defer cacheClient.Close()

	limiter, err := ratelimit.New(
		context.Background(),
		"close_test_service",
		ratelimit.WithCacheClient(cacheClient),
	)
	require.NoError(t, err)

	// 测试关闭
	err = limiter.Close()
	assert.NoError(t, err)

	// 再次关闭应该也不会出错
	err = limiter.Close()
	assert.NoError(t, err)
}

func BenchmarkRateLimiter_Allow(b *testing.B) {
	cacheClient, err := cache.New(context.Background(), cache.DefaultConfig())
	require.NoError(b, err)
	defer cacheClient.Close()

	defaultRules := map[string]ratelimit.Rule{
		"benchmark_rule": {Rate: 1000, Capacity: 2000},
	}

	limiter, err := ratelimit.New(
		context.Background(),
		"benchmark_service",
		ratelimit.WithCacheClient(cacheClient),
		ratelimit.WithDefaultRules(defaultRules),
	)
	require.NoError(b, err)
	defer limiter.Close()

	ctx := context.Background()
	resource := "bench:test"
	ruleName := "benchmark_rule"

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			_, _ = limiter.Allow(ctx, resource, ruleName)
		}
	})
}

func BenchmarkRateLimiter_BatchAllow(b *testing.B) {
	cacheClient, err := cache.New(context.Background(), cache.DefaultConfig())
	require.NoError(b, err)
	defer cacheClient.Close()

	defaultRules := map[string]ratelimit.Rule{
		"batch_benchmark": {Rate: 1000, Capacity: 2000},
	}

	limiter, err := ratelimit.New(
		context.Background(),
		"batch_benchmark_service",
		ratelimit.WithCacheClient(cacheClient),
		ratelimit.WithDefaultRules(defaultRules),
	)
	require.NoError(b, err)
	defer limiter.Close()

	ctx := context.Background()
	requests := []ratelimit.RateLimitRequest{
		{Resource: "batch:1", RuleName: "batch_benchmark", Count: 1},
		{Resource: "batch:2", RuleName: "batch_benchmark", Count: 1},
		{Resource: "batch:3", RuleName: "batch_benchmark", Count: 1},
		{Resource: "batch:4", RuleName: "batch_benchmark", Count: 1},
		{Resource: "batch:5", RuleName: "batch_benchmark", Count: 1},
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = limiter.BatchAllow(ctx, requests)
	}
}
