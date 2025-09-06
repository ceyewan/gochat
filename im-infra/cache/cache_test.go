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
	testClient cache.Cache
	ctx        = context.Background()
)

// TestMain 是所有测试的入口点，用于设置和清理测试环境。
func TestMain(m *testing.M) {
	// 从环境变量获取 Redis 地址，如果未设置则使用默认值
	redisAddr := os.Getenv("REDIS_ADDR")
	if redisAddr == "" {
		redisAddr = "localhost:6379"
	}

	cfg := cache.DefaultConfig()
	cfg.Addr = redisAddr
	cfg.KeyPrefix = "gochat_test" // 为测试使用独立的前缀

	// 创建一个不输出任何内容的 logger 用于测试
	logCfg := clog.DefaultConfig()
	logger, _ := clog.New(logCfg)

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
		err := testClient.Set(ctx, key, value, 1*time.Minute)
		require.NoError(t, err)

		retrieved, err := testClient.Get(ctx, key)
		require.NoError(t, err)
		assert.Equal(t, value, retrieved)

		// 测试 Incr
		incrKey := "string:counter"
		val, err := testClient.Incr(ctx, incrKey)
		require.NoError(t, err)
		assert.Equal(t, int64(1), val)
		val, err = testClient.Incr(ctx, incrKey)
		require.NoError(t, err)
		assert.Equal(t, int64(2), val)
	})

	// --- 哈希操作 ---
	t.Run("HashOperations", func(t *testing.T) {
		key := "hash:myhash"
		field := "myfield"
		value := "myvalue"
		err := testClient.HSet(ctx, key, field, value)
		require.NoError(t, err)

		retrieved, err := testClient.HGet(ctx, key, field)
		require.NoError(t, err)
		assert.Equal(t, value, retrieved)
	})

	// --- 集合操作 ---
	t.Run("SetOperations", func(t *testing.T) {
		key := "set:myset"
		member := "member1"
		err := testClient.SAdd(ctx, key, member)
		require.NoError(t, err)

		isMember, err := testClient.SIsMember(ctx, key, member)
		require.NoError(t, err)
		assert.True(t, isMember)
	})

	// --- 分布式锁操作 ---
	t.Run("LockOperations", func(t *testing.T) {
		key := "lock:mylock"
		lock, err := testClient.Lock(ctx, key, 10*time.Second)
		require.NoError(t, err)
		require.NotNil(t, lock, "第一次应该能成功获取锁")

		// 尝试再次获取锁（应该失败）
		sameLock, err := testClient.Lock(ctx, key, 10*time.Second)
		require.NoError(t, err)
		assert.Nil(t, sameLock, "第二次获取同一个锁应该失败")

		// 释放锁
		err = lock.Unlock(ctx)
		require.NoError(t, err)

		// 再次获取锁（现在应该成功）
		newLock, err := testClient.Lock(ctx, key, 10*time.Second)
		require.NoError(t, err)
		assert.NotNil(t, newLock, "释放后应该能再次获取锁")
		newLock.Unlock(ctx)
	})

	// --- 布隆过滤器操作 ---
	t.Run("BloomFilterOperations", func(t *testing.T) {
		key := "bf:myfilter"
		err := testClient.BFInit(ctx, key, 0.01, 1000)
		require.NoError(t, err)

		// 测试已存在的元素
		item1 := "hello"
		err = testClient.BFAdd(ctx, key, item1)
		require.NoError(t, err)
		exists, err := testClient.BFExists(ctx, key, item1)
		require.NoError(t, err)
		assert.True(t, exists, "已添加的元素应该存在")

		// 测试不存在的元素
		item2 := "world"
		exists, err = testClient.BFExists(ctx, key, item2)
		require.NoError(t, err)
		assert.False(t, exists, "未添加的元素应该不存在")
	})
}
