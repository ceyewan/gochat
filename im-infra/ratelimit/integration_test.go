//go:build integration
// +build integration

package ratelimit

import (
	"context"
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/ceyewan/gochat/im-infra/cache"
	"github.com/ceyewan/gochat/im-infra/coord"
	"github.com/ceyewan/gochat/im-infra/ratelimit"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const (
	redisAddr = "localhost:6379"
	etcdAddr  = "localhost:2379"
)

func TestIntegration_BasicRateLimiting(t *testing.T) {
	ctx := context.Background()

	// 创建缓存客户端
	cacheClient, err := cache.New(ctx, cache.Config{
		Addr: fmt.Sprintf("redis://%s", redisAddr),
		DB:   1, // 使用测试数据库
	})
	require.NoError(t, err)
	defer cacheClient.Close()

	// 定义测试规则
	testRules := map[string]ratelimit.Rule{
		"integration_test": {Rate: 5, Capacity: 10},
		"slow_test":        {Rate: 1, Capacity: 2},
	}

	// 创建限流器
	limiter, err := ratelimit.New(
		ctx,
		"integration_test_service",
		ratelimit.WithCacheClient(cacheClient),
		ratelimit.WithDefaultRules(testRules),
	)
	require.NoError(t, err)
	defer limiter.Close()

	// 测试基本限流功能
	resource := "integration:test:user1"
	ruleName := "integration_test"

	// 前10个请求应该都成功（桶初始容量）
	for i := 0; i < 10; i++ {
		allowed, err := limiter.Allow(ctx, resource, ruleName)
		require.NoError(t, err)
		assert.True(t, allowed, "请求 %d 应该被允许", i+1)
	}

	// 第11个请求应该失败
	allowed, err := limiter.Allow(ctx, resource, ruleName)
	require.NoError(t, err)
	assert.False(t, allowed, "第11个请求应该被拒绝")

	t.Log("基本限流功能测试通过")
}

func TestIntegration_WithCoordination(t *testing.T) {
	ctx := context.Background()

	// 创建协调客户端
	coordClient, err := coord.New(ctx, coord.CoordinatorConfig{
		Endpoints: []string{etcdAddr},
		Timeout:   5 * time.Second,
	})
	require.NoError(t, err)
	defer coordClient.Close()

	// 创建缓存客户端
	cacheClient, err := cache.New(ctx, cache.Config{
		Addr: fmt.Sprintf("redis://%s", redisAddr),
		DB:   2, // 使用测试数据库
	})
	require.NoError(t, err)
	defer cacheClient.Close()

	// 设置配置中心中的规则
	configKey := "/config/test/integration_coord_service/ratelimit/coord_test_rule"
	ruleConfig := map[string]interface{}{
		"rate":        2.0,
		"capacity":    int64(5),
		"description": "集成测试规则",
	}

	err = coordClient.Config().Set(ctx, configKey, ruleConfig)
	require.NoError(t, err)

	// 创建限流器（使用配置中心）
	limiter, err := ratelimit.New(
		ctx,
		"integration_coord_service",
		ratelimit.WithCacheClient(cacheClient),
		ratelimit.WithCoordinationClient(coordClient),
		ratelimit.WithRuleRefreshInterval(1*time.Second),
	)
	require.NoError(t, err)
	defer limiter.Close()

	// 等待配置加载
	time.Sleep(2 * time.Second)

	// 测试使用配置中心的规则
	resource := "integration:coord:test"
	ruleName := "coord_test_rule"

	allowedCount := 0
	for i := 0; i < 8; i++ {
		allowed, err := limiter.Allow(ctx, resource, ruleName)
		require.NoError(t, err)
		if allowed {
			allowedCount++
		}
		time.Sleep(100 * time.Millisecond)
	}

	// 应该有一定数量的请求被允许（基于配置的容量）
	assert.GreaterOrEqual(t, allowedCount, 1)
	assert.LessOrEqual(t, allowedCount, 8)

	// 清理配置
	coordClient.Config().Delete(ctx, configKey)

	t.Logf("配置中心集成测试通过，允许请求数: %d", allowedCount)
}

func TestIntegration_HighConcurrency(t *testing.T) {
	ctx := context.Background()

	// 创建缓存客户端
	cacheClient, err := cache.New(ctx, cache.Config{
		Addr: fmt.Sprintf("redis://%s", redisAddr),
		DB:   3, // 使用测试数据库
	})
	require.NoError(t, err)
	defer cacheClient.Close()

	// 定义高并发测试规则
	testRules := map[string]ratelimit.Rule{
		"concurrency_test": {Rate: 50, Capacity: 100},
	}

	// 创建限流器
	limiter, err := ratelimit.New(
		ctx,
		"integration_concurrency_service",
		ratelimit.WithCacheClient(cacheClient),
		ratelimit.WithDefaultRules(testRules),
	)
	require.NoError(t, err)
	defer limiter.Close()

	// 高并发测试
	const numGoroutines = 100
	const requestsPerGoroutine = 10

	var (
		wg           sync.WaitGroup
		totalAllowed int64
		mu           sync.Mutex
	)

	wg.Add(numGoroutines)

	startTime := time.Now()

	for i := 0; i < numGoroutines; i++ {
		go func(workerID int) {
			defer wg.Done()

			localAllowed := 0
			resource := fmt.Sprintf("integration:concurrency:worker:%d", workerID)

			for j := 0; j < requestsPerGoroutine; j++ {
				allowed, err := limiter.Allow(ctx, resource, "concurrency_test")
				if err != nil {
					t.Errorf("Worker %d 请求 %d 失败: %v", workerID, j, err)
					continue
				}

				if allowed {
					localAllowed++
				}
			}

			mu.Lock()
			totalAllowed += int64(localAllowed)
			mu.Unlock()
		}(i)
	}

	wg.Wait()

	duration := time.Since(startTime)
	totalRequests := int64(numGoroutines * requestsPerGoroutine)

	t.Logf("高并发测试完成:")
	t.Logf("  总请求数: %d", totalRequests)
	t.Logf("  允许请求数: %d", totalAllowed)
	t.Logf("  成功率: %.2f%%", float64(totalAllowed)/float64(totalRequests)*100)
	t.Logf("  耗时: %v", duration)
	t.Logf("  TPS: %.0f", float64(totalRequests)/duration.Seconds())

	// 验证至少有一些请求被允许
	assert.Greater(t, totalAllowed, int64(0), "至少应该有一些请求被允许")

	// 验证不是所有请求都被允许（说明限流生效）
	assert.Less(t, totalAllowed, totalRequests, "不是所有请求都应该被允许")
}

func TestIntegration_Statistics(t *testing.T) {
	ctx := context.Background()

	// 创建缓存客户端
	cacheClient, err := cache.New(ctx, cache.Config{
		Addr: fmt.Sprintf("redis://%s", redisAddr),
		DB:   4, // 使用测试数据库
	})
	require.NoError(t, err)
	defer cacheClient.Close()

	// 定义统计测试规则
	testRules := map[string]ratelimit.Rule{
		"stats_test": {Rate: 3, Capacity: 5},
	}

	// 创建限流器
	limiter, err := ratelimit.New(
		ctx,
		"integration_stats_service",
		ratelimit.WithCacheClient(cacheClient),
		ratelimit.WithDefaultRules(testRules),
		ratelimit.WithStatisticsEnabled(true),
	)
	require.NoError(t, err)
	defer limiter.Close()

	resource := "integration:stats:test"
	ruleName := "stats_test"

	// 发送一系列请求
	expectedTotal := 10
	for i := 0; i < expectedTotal; i++ {
		limiter.Allow(ctx, resource, ruleName)
		time.Sleep(100 * time.Millisecond)
	}

	// 获取统计信息
	stats, err := limiter.GetStatistics(ctx, resource, ruleName)
	require.NoError(t, err)
	require.NotNil(t, stats)

	// 验证统计信息
	assert.Equal(t, resource, stats.Resource)
	assert.Equal(t, ruleName, stats.RuleName)
	assert.Equal(t, int64(expectedTotal), stats.TotalRequests)
	assert.Greater(t, stats.AllowedRequests, int64(0))
	assert.GreaterOrEqual(t, stats.DeniedRequests, int64(0))
	assert.Equal(t, stats.TotalRequests, stats.AllowedRequests+stats.DeniedRequests)

	if stats.TotalRequests > 0 {
		expectedSuccessRate := float64(stats.AllowedRequests) / float64(stats.TotalRequests)
		assert.InDelta(t, expectedSuccessRate, stats.SuccessRate, 0.01)
	}

	t.Logf("统计信息测试通过:")
	t.Logf("  总请求数: %d", stats.TotalRequests)
	t.Logf("  允许请求数: %d", stats.AllowedRequests)
	t.Logf("  拒绝请求数: %d", stats.DeniedRequests)
	t.Logf("  当前令牌数: %d", stats.CurrentTokens)
	t.Logf("  成功率: %.2f%%", stats.SuccessRate*100)
}

func TestIntegration_BatchOperations(t *testing.T) {
	ctx := context.Background()

	// 创建缓存客户端
	cacheClient, err := cache.New(ctx, cache.Config{
		Addr: fmt.Sprintf("redis://%s", redisAddr),
		DB:   5, // 使用测试数据库
	})
	require.NoError(t, err)
	defer cacheClient.Close()

	// 定义批处理测试规则
	testRules := map[string]ratelimit.Rule{
		"batch_test_high": {Rate: 10, Capacity: 20},
		"batch_test_low":  {Rate: 2, Capacity: 3},
	}

	// 创建限流器
	limiter, err := ratelimit.New(
		ctx,
		"integration_batch_service",
		ratelimit.WithCacheClient(cacheClient),
		ratelimit.WithDefaultRules(testRules),
		ratelimit.WithBatchSize(50),
	)
	require.NoError(t, err)
	defer limiter.Close()

	// 构建批量请求
	requests := []ratelimit.RateLimitRequest{
		{Resource: "batch:user1", RuleName: "batch_test_high", Count: 1},
		{Resource: "batch:user2", RuleName: "batch_test_high", Count: 2},
		{Resource: "batch:user3", RuleName: "batch_test_low", Count: 1},
		{Resource: "batch:user4", RuleName: "batch_test_low", Count: 1},
		{Resource: "batch:user5", RuleName: "batch_test_high", Count: 3},
	}

	// 执行批量请求
	results, err := limiter.BatchAllow(ctx, requests)
	require.NoError(t, err)
	require.Len(t, results, len(requests))

	// 验证结果
	allowedCount := 0
	for i, allowed := range results {
		if allowed {
			allowedCount++
		}
		t.Logf("批量请求 %d (%s, %s, %d): %v",
			i+1, requests[i].Resource, requests[i].RuleName, requests[i].Count, allowed)
	}

	// 应该至少有一些请求被允许
	assert.Greater(t, allowedCount, 0, "至少应该有一些批量请求被允许")

	t.Logf("批量操作测试通过，允许的请求数: %d/%d", allowedCount, len(requests))
}

func TestIntegration_RuleManagement(t *testing.T) {
	ctx := context.Background()

	// 创建协调客户端（用于规则管理）
	coordClient, err := coord.New(ctx, coord.CoordinatorConfig{
		Endpoints: []string{etcdAddr},
		Timeout:   5 * time.Second,
	})
	require.NoError(t, err)
	defer coordClient.Close()

	// 创建缓存客户端
	cacheClient, err := cache.New(ctx, cache.Config{
		Addr: fmt.Sprintf("redis://%s", redisAddr),
		DB:   6, // 使用测试数据库
	})
	require.NoError(t, err)
	defer cacheClient.Close()

	// 创建管理器
	manager, err := ratelimit.NewManager(
		ctx,
		"integration_management_service",
		ratelimit.WithCacheClient(cacheClient),
		ratelimit.WithCoordinationClient(coordClient),
	)
	require.NoError(t, err)
	defer manager.Close()

	// 测试动态设置规则
	testRule := ratelimit.Rule{Rate: 5, Capacity: 10}
	err = manager.SetRule(ctx, "dynamic_rule", testRule)
	require.NoError(t, err)

	// 测试列出规则
	rules := manager.ListRules()
	assert.Contains(t, rules, "dynamic_rule")
	assert.Equal(t, testRule, rules["dynamic_rule"])

	// 测试使用动态设置的规则
	allowed, err := manager.Allow(ctx, "management:test", "dynamic_rule")
	require.NoError(t, err)
	assert.True(t, allowed, "使用动态规则的第一个请求应该被允许")

	// 测试导出规则
	err = manager.ExportRules(ctx)
	require.NoError(t, err)

	// 测试重新加载规则
	err = manager.ReloadRules()
	require.NoError(t, err)

	// 测试删除规则
	err = manager.DeleteRule(ctx, "dynamic_rule")
	require.NoError(t, err)

	// 验证规则已删除
	updatedRules := manager.ListRules()
	assert.NotContains(t, updatedRules, "dynamic_rule")

	t.Log("规则管理功能测试通过")
}

func TestIntegration_ErrorResilience(t *testing.T) {
	ctx := context.Background()

	// 使用不存在的 Redis 地址测试错误恢复
	invalidCacheClient, err := cache.New(ctx, cache.Config{
		Addr: "redis://localhost:9999", // 不存在的地址
		DB:   0,
	})
	if err == nil {
		// 如果创建成功，说明缓存客户端有容错机制，这是正常的
		defer invalidCacheClient.Close()
	}

	// 测试使用默认规则的情况（没有配置中心）
	defaultRules := map[string]ratelimit.Rule{
		"resilience_test": {Rate: 5, Capacity: 10},
	}

	// 创建一个简单的限流器（不依赖外部服务）
	limiter, err := ratelimit.SimpleRateLimiter(
		ctx,
		"integration_resilience_service",
		defaultRules,
	)
	require.NoError(t, err)
	defer limiter.Close()

	// 测试基本功能仍然工作
	allowed, err := limiter.Allow(ctx, "resilience:test", "resilience_test")
	require.NoError(t, err)
	assert.True(t, allowed, "即使没有外部依赖，基本功能也应该工作")

	// 测试未知规则的容错处理
	allowed, err = limiter.Allow(ctx, "resilience:test", "unknown_rule")
	require.NoError(t, err)
	assert.True(t, allowed, "未知规则应该默认允许")

	t.Log("错误恢复能力测试通过")
}
