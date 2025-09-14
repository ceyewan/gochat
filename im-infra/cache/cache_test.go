package cache_test

import (
	"context"
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/ceyewan/gochat/im-infra/cache"
	"github.com/ceyewan/gochat/im-infra/clog"
	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var (
	testClient cache.Provider
	ctx        = context.Background()
)

// TestMain 是所有测试的入口点，用于设置和清理测试环境。
func TestMain(m *testing.M) {
	// 从环境变量获取 Redis 地址，如果未设置则使用默认值
	redisAddr := os.Getenv("REDIS_ADDR")
	if redisAddr == "" {
		redisAddr = "localhost:6379"
	}

	cfg := cache.GetDefaultConfig("development")
	cfg.Addr = redisAddr
	cfg.KeyPrefix = "gochat_test" // 为测试使用独立的前缀

	// 创建一个不输出任何内容的 logger 用于测试
	logger := clog.Namespace("cache-test")

	var err error
	testClient, err = cache.New(ctx, cfg, cache.WithLogger(logger))
	if err != nil {
		fmt.Printf("无法连接到 Redis 进行测试: %v\n", err)
		os.Exit(1)
	}

	// 运行所有测试
	exitCode := m.Run()

	// 清理测试数据
	cleanup(redisAddr, cfg.KeyPrefix)

	os.Exit(exitCode)
}

// cleanup 清理所有测试创建的键
func cleanup(addr, prefix string) {
	// 创建一个临时的 Redis 客户端用于清理
	rdb := redis.NewClient(&redis.Options{Addr: addr})
	defer rdb.Close()

	keys, err := rdb.Keys(ctx, prefix+":*").Result()
	if err != nil {
		fmt.Printf("清理测试键失败: %v\n", err)
		return
	}
	if len(keys) > 0 {
		rdb.Del(ctx, keys...)
		fmt.Printf("成功清理 %d 个测试键\n", len(keys))
	}
}

func TestCacheIntegration(t *testing.T) {
	// --- 字符串操作 ---
	t.Run("StringOperations", func(t *testing.T) {
		key := "string:mykey"
		value := "hello world"
		err := testClient.String().Set(ctx, key, value, 1*time.Minute)
		require.NoError(t, err)

		retrieved, err := testClient.String().Get(ctx, key)
		require.NoError(t, err)
		assert.Equal(t, value, retrieved)

		// 测试 Incr
		incrKey := "string:counter"
		val, err := testClient.String().Incr(ctx, incrKey)
		require.NoError(t, err)
		assert.Equal(t, int64(1), val)
		val, err = testClient.String().Incr(ctx, incrKey)
		require.NoError(t, err)
		assert.Equal(t, int64(2), val)
	})

	// --- 哈希操作 ---
	t.Run("HashOperations", func(t *testing.T) {
		key := "hash:myhash"
		field := "myfield"
		value := "myvalue"
		err := testClient.Hash().HSet(ctx, key, field, value)
		require.NoError(t, err)

		retrieved, err := testClient.Hash().HGet(ctx, key, field)
		require.NoError(t, err)
		assert.Equal(t, value, retrieved)
	})

	// --- 集合操作 ---
	t.Run("SetOperations", func(t *testing.T) {
		key := "set:myset"
		member := "member1"
		err := testClient.Set().SAdd(ctx, key, member)
		require.NoError(t, err)

		isMember, err := testClient.Set().SIsMember(ctx, key, member)
		require.NoError(t, err)
		assert.True(t, isMember)
	})

	// --- 分布式锁操作 ---
	t.Run("LockOperations", func(t *testing.T) {
		key := "lock:mylock"
		lock, err := testClient.Lock().Acquire(ctx, key, 10*time.Second)
		require.NoError(t, err)
		require.NotNil(t, lock, "第一次应该能成功获取锁")

		// 尝试再次获取锁（应该失败）
		sameLock, err := testClient.Lock().Acquire(ctx, key, 10*time.Second)
		assert.Error(t, err, "第二次获取同一个锁应该返回错误")
		assert.Nil(t, sameLock, "第二次获取同一个锁应该返回 nil")

		// 释放锁
		err = lock.Unlock(ctx)
		require.NoError(t, err)

		// 再次获取锁（现在应该成功）
		newLock, err := testClient.Lock().Acquire(ctx, key, 10*time.Second)
		require.NoError(t, err)
		assert.NotNil(t, newLock, "释放后应该能再次获取锁")
		newLock.Unlock(ctx)
	})

	// --- 布隆过滤器操作 ---
	t.Run("BloomFilterOperations", func(t *testing.T) {
		key := "bf:myfilter"
		err := testClient.Bloom().BFReserve(ctx, key, 0.01, 1000)
		require.NoError(t, err)

		// 测试已存在的元素
		item1 := "hello"
		err = testClient.Bloom().BFAdd(ctx, key, item1)
		require.NoError(t, err)
		exists, err := testClient.Bloom().BFExists(ctx, key, item1)
		require.NoError(t, err)
		assert.True(t, exists, "已添加的元素应该存在")

		// 测试不存在的元素
		item2 := "world"
		exists, err = testClient.Bloom().BFExists(ctx, key, item2)
		require.NoError(t, err)
		assert.False(t, exists, "未添加的元素应该不存在")
	})

	// --- GetSet 方法测试 ---
	t.Run("GetSetMethod", func(t *testing.T) {
		key := "string:getset"
		initialValue := "initial"
		newValue := "updated"

		// 先设置一个值
		err := testClient.String().Set(ctx, key, initialValue, 1*time.Minute)
		require.NoError(t, err)

		// 使用 GetSet 获取旧值并设置新值
		oldValue, err := testClient.String().GetSet(ctx, key, newValue)
		require.NoError(t, err)
		assert.Equal(t, initialValue, oldValue)

		// 验证新值已设置
		retrievedValue, err := testClient.String().Get(ctx, key)
		require.NoError(t, err)
		assert.Equal(t, newValue, retrievedValue)
	})

	// --- ErrCacheMiss 处理测试 ---
	t.Run("ErrCacheMissHandling", func(t *testing.T) {
		key := "nonexistent:key"

		// 测试获取不存在的键
		_, err := testClient.String().Get(ctx, key)
		assert.ErrorIs(t, err, cache.ErrCacheMiss, "应该返回 ErrCacheMiss 错误")

		// 测试获取不存在的哈希字段
		_, err = testClient.Hash().HGet(ctx, "nonexistent:hash", "field")
		assert.ErrorIs(t, err, cache.ErrCacheMiss, "应该返回 ErrCacheMiss 错误")
	})

	// --- ZSET 操作测试 ---
	t.Run("ZSetOperations", func(t *testing.T) {
		key := "zset:session:messages"
		now := time.Now()

		// 测试 ZAdd
		messages := []*cache.ZMember{
			{Member: "msg1", Score: float64(now.Add(-10 * time.Minute).Unix())},
			{Member: "msg2", Score: float64(now.Add(-8 * time.Minute).Unix())},
			{Member: "msg3", Score: float64(now.Add(-6 * time.Minute).Unix())},
			{Member: "msg4", Score: float64(now.Add(-4 * time.Minute).Unix())},
			{Member: "msg5", Score: float64(now.Add(-2 * time.Minute).Unix())},
		}

		err := testClient.ZSet().ZAdd(ctx, key, messages...)
		require.NoError(t, err)

		// 测试 ZCard
		count, err := testClient.ZSet().ZCard(ctx, key)
		require.NoError(t, err)
		assert.Equal(t, int64(5), count)

		// 测试 ZRange (从早到晚)
		earlyMessages, err := testClient.ZSet().ZRange(ctx, key, 0, -1)
		require.NoError(t, err)
		assert.Len(t, earlyMessages, 5)
		assert.Equal(t, "msg1", earlyMessages[0].Member) // 最早的消息
		assert.Equal(t, "msg5", earlyMessages[4].Member) // 最晚的消息

		// 测试 ZRevRange (从晚到早)
		recentMessages, err := testClient.ZSet().ZRevRange(ctx, key, 0, 2) // 获取最新的3条
		require.NoError(t, err)
		assert.Len(t, recentMessages, 3)
		assert.Equal(t, "msg5", recentMessages[0].Member) // 最新的消息
		assert.Equal(t, "msg4", recentMessages[1].Member)
		assert.Equal(t, "msg3", recentMessages[2].Member)

		// 测试 ZRangeByScore
		sevenHoursAgo := float64(now.Add(-7 * time.Hour).Unix())
		fiveMinutesAgo := float64(now.Add(-5 * time.Minute).Unix())
		recentByScore, err := testClient.ZSet().ZRangeByScore(ctx, key, sevenHoursAgo, fiveMinutesAgo)
		require.NoError(t, err)
		// 应该包含 msg1, msg2, msg3 (在7小时前到5分钟前范围内)
		assert.True(t, len(recentByScore) >= 3)

		// 测试 ZCount
		countInRange, err := testClient.ZSet().ZCount(ctx, key, sevenHoursAgo, float64(now.Unix()))
		require.NoError(t, err)
		assert.Equal(t, int64(5), countInRange) // 所有消息都在这个范围内

		// 测试 ZScore
		score, err := testClient.ZSet().ZScore(ctx, key, "msg3")
		require.NoError(t, err)
		assert.Equal(t, messages[2].Score, score)

		// 测试不存在的消息
		_, err = testClient.ZSet().ZScore(ctx, key, "nonexistent")
		assert.ErrorIs(t, err, cache.ErrCacheMiss)

		// 测试 ZRem
		err = testClient.ZSet().ZRem(ctx, key, "msg3")
		require.NoError(t, err)

		// 验证消息已被移除
		_, err = testClient.ZSet().ZScore(ctx, key, "msg3")
		assert.ErrorIs(t, err, cache.ErrCacheMiss)

		// 验证数量减少
		newCount, err := testClient.ZSet().ZCard(ctx, key)
		require.NoError(t, err)
		assert.Equal(t, int64(4), newCount)

		// 测试 ZRemRangeByRank (移除排名最早的1条)
		err = testClient.ZSet().ZRemRangeByRank(ctx, key, 0, 0)
		require.NoError(t, err)

		// 验证最早的消息已被移除
		remainingMessages, err := testClient.ZSet().ZRange(ctx, key, 0, -1)
		require.NoError(t, err)
		assert.Len(t, remainingMessages, 3)
		assert.Equal(t, "msg2", remainingMessages[0].Member) // msg1 应该已被移除

		// 测试 ZSetExpire
		err = testClient.ZSet().ZSetExpire(ctx, key, 5*time.Minute)
		require.NoError(t, err)

		// 验证 ZSET 是否支持过期操作
		// 注意：在测试环境中我们无法直接验证TTL，但至少确认方法调用成功
	})

	// --- ZSET 会话消息管理场景测试 ---
	t.Run("ZSetSessionMessageManagement", func(t *testing.T) {
		sessionKey := "zset:session:chat123"
		now := time.Now()

		// 模拟添加60条消息，每条消息间隔1分钟
		var messages []*cache.ZMember
		for i := 1; i <= 60; i++ {
			// 计算时间戳：msg1是59分钟前，msg2是58分钟前，...，msg60是当前时间
			minutesAgo := 60 - i
			score := float64(now.Add(-time.Duration(minutesAgo) * time.Minute).Unix())
			messages = append(messages, &cache.ZMember{
				Member: fmt.Sprintf("msg%d", i),
				Score:  score,
			})
		}

		// 添加所有消息
		err := testClient.ZSet().ZAdd(ctx, sessionKey, messages...)
		require.NoError(t, err)

		// 验证总数
		count, err := testClient.ZSet().ZCard(ctx, sessionKey)
		require.NoError(t, err)
		assert.Equal(t, int64(60), count)

		// 获取最新的50条消息（通过分数范围）
		// 49分钟前到现在的消息应该是 msg12 到 msg60，共50条（包含边界值）
		fortyNineMinutesAgo := float64(now.Add(-49 * time.Minute).Unix())
		recentMessages, err := testClient.ZSet().ZRangeByScore(ctx, sessionKey, fortyNineMinutesAgo, float64(now.Unix()))
		require.NoError(t, err)
		assert.Len(t, recentMessages, 50)

		// 验证最新消息的顺序
		assert.Equal(t, "msg60", recentMessages[len(recentMessages)-1].Member) // 最新的消息
		assert.Equal(t, "msg11", recentMessages[0].Member)                    // 50条中的最早消息（msg11是49分钟前发送的）

		// 使用 ZRevRange 获取最新的5条消息
		latest5, err := testClient.ZSet().ZRevRange(ctx, sessionKey, 0, 4)
		require.NoError(t, err)
		assert.Len(t, latest5, 5)
		assert.Equal(t, "msg60", latest5[0].Member)
		assert.Equal(t, "msg59", latest5[1].Member)
		assert.Equal(t, "msg58", latest5[2].Member)
		assert.Equal(t, "msg57", latest5[3].Member)
		assert.Equal(t, "msg56", latest5[4].Member)

		// 设置过期时间
		err = testClient.ZSet().ZSetExpire(ctx, sessionKey, 2*time.Hour)
		require.NoError(t, err)
	})

	// --- GetDefaultConfig 函数测试 ---
	t.Run("GetDefaultConfig", func(t *testing.T) {
		// 测试开发环境配置
		devConfig := cache.GetDefaultConfig("development")
		assert.Equal(t, "localhost:6379", devConfig.Addr)
		assert.Equal(t, 10, devConfig.PoolSize)
		assert.Equal(t, "dev:", devConfig.KeyPrefix)

		// 测试生产环境配置
		prodConfig := cache.GetDefaultConfig("production")
		assert.Equal(t, "redis:6379", prodConfig.Addr)
		assert.Equal(t, 100, prodConfig.PoolSize)
		assert.Equal(t, "gochat:", prodConfig.KeyPrefix)

		// 测试未知环境（默认为开发环境）
		unknownConfig := cache.GetDefaultConfig("unknown")
		assert.Equal(t, devConfig.Addr, unknownConfig.Addr)
		assert.Equal(t, devConfig.PoolSize, unknownConfig.PoolSize)
	})
}
