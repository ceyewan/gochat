package ratelimit_test

import (
	"context"
	"testing"
	"time"

	"github.com/ceyewan/gochat/im-infra/cache"
	"github.com/ceyewan/gochat/im-infra/ratelimit"
	"github.com/stretchr/testify/assert"
)

func TestRateLimiter_Allow(t *testing.T) {
	// 使用测试配置初始化缓存
	cacheClient, err := cache.New(cache.TestConfig())
	assert.NoError(t, err)

	// 定义默认规则用于测试
	defaultRules := map[string]ratelimit.Rule{
		"test_rule": {
			Rate:     10, // 10 tokens per second
			Capacity: 10, // bucket size of 10
		},
	}

	// 创建限流器实例
	limiter, err := ratelimit.New(
		context.Background(),
		"test_service",
		ratelimit.WithCacheClient(cacheClient),
		ratelimit.WithDefaultRules(defaultRules),
	)
	assert.NoError(t, err)
	defer limiter.Close()

	// 清理 Redis Key，确保测试环境干净
	resource := "user:123"
	ruleName := "test_rule"
	key := "ratelimit:test_service:test_rule:user:123"
	cacheClient.Del(context.Background(), key)

	// 连续请求 10 次，应该都成功
	for i := 0; i < 10; i++ {
		allowed, err := limiter.Allow(context.Background(), resource, ruleName)
		assert.NoError(t, err)
		assert.True(t, allowed, "请求 %d 应该被允许", i+1)
	}

	// 第 11 次请求，令牌耗尽，应该失败
	allowed, err := limiter.Allow(context.Background(), resource, ruleName)
	assert.NoError(t, err)
	assert.False(t, allowed, "第 11 次请求应该被拒绝")

	// 等待 0.5 秒，应该会补充 5 个令牌
	time.Sleep(500 * time.Millisecond)

	// 再次请求 5 次，应该都成功
	for i := 0; i < 5; i++ {
		allowed, err := limiter.Allow(context.Background(), resource, ruleName)
		assert.NoError(t, err)
		assert.True(t, allowed, "补充令牌后的请求 %d 应该被允许", i+1)
	}

	// 第 6 次请求，令牌再次耗尽，应该失败
	allowed, err = limiter.Allow(context.Background(), resource, ruleName)
	assert.NoError(t, err)
	assert.False(t, allowed, "补充令牌后的第 6 次请求应该被拒绝")
}
