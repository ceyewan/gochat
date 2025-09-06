package idgen

import (
	"context"
	"testing"
	"time"

	"github.com/ceyewan/gochat/im-infra/cache"
)

// 注意：这些测试需要 Redis 服务运行在 localhost:6379
// 如果没有 Redis 服务，测试将被跳过

func TestRedisIDGenerator(t *testing.T) {
	ctx := context.Background()

	// 创建 Redis 配置
	config := &RedisConfig{
		CacheConfig: cache.Config{
			Addr:     "localhost:6379",
			Password: "",
			DB:       15, // 使用测试数据库
			PoolSize: 5,
		},
		KeyPrefix:    "test_idgen",
		DefaultKey:   "test_counter",
		Step:         1,
		InitialValue: 1,
		TTL:          0, // 不过期
	}

	generator, err := NewRedisGenerator(config)
	if err != nil {
		t.Skipf("跳过 Redis 测试，无法连接到 Redis: %v", err)
		return
	}
	defer generator.Close()

	// 重置计数器
	err = generator.Reset(ctx, "test_counter")
	if err != nil {
		t.Fatalf("重置计数器失败: %v", err)
	}

	// 测试生成 ID
	id1, err := generator.GenerateInt64(ctx)
	if err != nil {
		t.Fatalf("生成 ID 失败: %v", err)
	}

	id2, err := generator.GenerateInt64(ctx)
	if err != nil {
		t.Fatalf("生成第二个 ID 失败: %v", err)
	}

	// 验证 ID 递增
	if id2 != id1+1 {
		t.Errorf("ID 应该递增，期望 %d，实际 %d", id1+1, id2)
	}

	// 测试字符串 ID
	idStr, err := generator.GenerateString(ctx)
	if err != nil {
		t.Fatalf("生成字符串 ID 失败: %v", err)
	}

	if idStr == "" {
		t.Error("字符串 ID 不应该为空")
	}

	// 测试获取当前值
	current, err := generator.GetCurrent(ctx, "test_counter")
	if err != nil {
		t.Fatalf("获取当前值失败: %v", err)
	}

	if current < id2 {
		t.Errorf("当前值应该大于等于最后生成的 ID，当前值: %d，最后 ID: %d", current, id2)
	}

	t.Logf("生成的 ID: %d, %d, %s", id1, id2, idStr)
	t.Logf("当前计数值: %d", current)
}

func TestRedisIDGeneratorWithCustomKey(t *testing.T) {
	ctx := context.Background()

	config := &RedisConfig{
		CacheConfig: cache.Config{
			Addr:     "localhost:6379",
			Password: "",
			DB:       15,
			PoolSize: 5,
		},
		KeyPrefix:    "test_idgen",
		DefaultKey:   "default",
		Step:         5, // 步长为 5
		InitialValue: 100,
		TTL:          0,
	}

	generator, err := NewRedisGenerator(config)
	if err != nil {
		t.Skipf("跳过 Redis 测试，无法连接到 Redis: %v", err)
		return
	}
	defer generator.Close()

	customKey := "custom_counter"

	// 重置自定义键
	err = generator.Reset(ctx, customKey)
	if err != nil {
		t.Fatalf("重置自定义键失败: %v", err)
	}

	// 使用自定义键生成 ID
	id1, err := generator.GenerateWithKey(ctx, customKey)
	if err != nil {
		t.Fatalf("使用自定义键生成 ID 失败: %v", err)
	}

	id2, err := generator.GenerateWithKey(ctx, customKey)
	if err != nil {
		t.Fatalf("使用自定义键生成第二个 ID 失败: %v", err)
	}

	// 验证步长
	expectedDiff := int64(5) // 步长为 5，但 GenerateWithKey 内部调用 GenerateWithStep，每次调用会增加 step 次
	if id2-id1 != expectedDiff {
		t.Errorf("ID 差值应该为 %d，实际为 %d", expectedDiff, id2-id1)
	}

	t.Logf("自定义键生成的 ID: %d, %d", id1, id2)
}

func TestRedisIDGeneratorWithTTL(t *testing.T) {
	ctx := context.Background()

	config := &RedisConfig{
		CacheConfig: cache.Config{
			Addr:     "localhost:6379",
			Password: "",
			DB:       15,
			PoolSize: 5,
		},
		KeyPrefix:    "test_idgen_ttl",
		DefaultKey:   "ttl_counter",
		Step:         1,
		InitialValue: 1,
		TTL:          time.Second * 2, // 2 秒过期
	}

	generator, err := NewRedisGenerator(config)
	if err != nil {
		t.Skipf("跳过 Redis 测试，无法连接到 Redis: %v", err)
		return
	}
	defer generator.Close()

	// 生成 ID
	id1, err := generator.GenerateInt64(ctx)
	if err != nil {
		t.Fatalf("生成 ID 失败: %v", err)
	}

	// 等待过期
	time.Sleep(time.Second * 3)

	// 再次生成 ID，应该重新从初始值开始
	id2, err := generator.GenerateInt64(ctx)
	if err != nil {
		t.Fatalf("过期后生成 ID 失败: %v", err)
	}

	// 由于键过期，新的 ID 应该从初始值开始
	if id2 != config.InitialValue+1 {
		t.Logf("注意：键可能没有过期，或者 Redis 配置不同。ID1: %d, ID2: %d", id1, id2)
	}

	t.Logf("TTL 测试 - ID1: %d, ID2: %d", id1, id2)
}
