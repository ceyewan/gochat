package main

import (
	"context"
	"log"
	"time"

	"github.com/ceyewan/gochat/im-infra/cache"
	"github.com/ceyewan/gochat/im-infra/clog"
)

func main() {
	logger := clog.Namespace("cache-comprehensive-example")
	ctx := context.Background()

	// 获取配置并测试不同环境
	log.Println("=== 配置测试 ===")
	testConfigurations()

	// 创建缓存客户端
	cfg := cache.GetDefaultConfig("development")
	cfg.Addr = "localhost:6379"

	cacheClient, err := cache.New(ctx, cfg, cache.WithLogger(logger))
	if err != nil {
		log.Fatalf("无法创建缓存客户端: %v", err)
	}
	defer cacheClient.Close()

	log.Println("缓存客户端创建成功！")

	// --- String 接口演示 ---
	log.Println("\n=== String 接口演示 ===")
	demoStringOperations(ctx, cacheClient)

	// --- Hash 接口演示 ---
	log.Println("\n=== Hash 接口演示 ===")
	demoHashOperations(ctx, cacheClient)

	// --- Set 接口演示 ---
	log.Println("\n=== Set 接口演示 ===")
	demoSetOperations(ctx, cacheClient)

	// --- Lock 接口演示 ---
	log.Println("\n=== Lock 接口演示 ===")
	demoLockOperations(ctx, cacheClient)

	// --- Bloom 接口演示 ---
	log.Println("\n=== Bloom 接口演示 ===")
	demoBloomOperations(ctx, cacheClient)

	// --- Script 接口演示 ---
	log.Println("\n=== Script 接口演示 ===")
	demoScriptOperations(ctx, cacheClient)

	// --- ZSET 接口演示 ---
	log.Println("\n=== ZSET 接口演示 ===")
	demoZSetOperations(ctx, cacheClient)

	// --- ErrCacheMiss 处理演示 ---
	log.Println("\n=== ErrCacheMiss 处理演示 ===")
	demoErrorHandling(ctx, cacheClient)

	log.Println("\n所有接口演示完成！")
}

func testConfigurations() {
	// 测试不同环境的配置
	devConfig := cache.GetDefaultConfig("development")
	log.Printf("开发环境配置: 地址=%s, 连接池大小=%d, 键前缀=%s",
		devConfig.Addr, devConfig.PoolSize, devConfig.KeyPrefix)

	prodConfig := cache.GetDefaultConfig("production")
	log.Printf("生产环境配置: 地址=%s, 连接池大小=%d, 键前缀=%s",
		prodConfig.Addr, prodConfig.PoolSize, prodConfig.KeyPrefix)
}

func demoStringOperations(ctx context.Context, client cache.Provider) {
	stringOps := client.String()

	// 基本设置和获取
	key := "demo:string:basic"
	value := "Hello, World!"
	err := stringOps.Set(ctx, key, value, 5*time.Minute)
	if err != nil {
		log.Printf("设置失败: %v", err)
		return
	}

	retrieved, err := stringOps.Get(ctx, key)
	if err != nil {
		log.Printf("获取失败: %v", err)
		return
	}
	log.Printf("String - 基本操作: 设置='%s', 获取='%s'", value, retrieved)

	// GetSet 操作
	getSetKey := "demo:string:getset"
	err = stringOps.Set(ctx, getSetKey, "old_value", 5*time.Minute)
	if err != nil {
		log.Printf("GetSet 初始设置失败: %v", err)
		return
	}

	oldValue, err := stringOps.GetSet(ctx, getSetKey, "new_value")
	if err != nil {
		log.Printf("GetSet 失败: %v", err)
		return
	}

	newValue, err := stringOps.Get(ctx, getSetKey)
	if err != nil {
		log.Printf("获取新值失败: %v", err)
		return
	}
	log.Printf("String - GetSet: 旧值='%s', 新值='%s'", oldValue, newValue)

	// 计数器操作
	counterKey := "demo:string:counter"
	val, err := stringOps.Incr(ctx, counterKey)
	if err != nil {
		log.Printf("递增失败: %v", err)
		return
	}
	log.Printf("String - 计数器: 第一次递增 = %d", val)

	val, err = stringOps.Incr(ctx, counterKey)
	if err != nil {
		log.Printf("递增失败: %v", err)
		return
	}
	log.Printf("String - 计数器: 第二次递增 = %d", val)
}

func demoHashOperations(ctx context.Context, client cache.Provider) {
	hashOps := client.Hash()

	key := "demo:hash:user"

	// 设置哈希字段
	fields := map[string]interface{}{
		"name":  "张三",
		"age":   "25",
		"email": "zhangsan@example.com",
	}

	for field, value := range fields {
		err := hashOps.HSet(ctx, key, field, value)
		if err != nil {
			log.Printf("设置哈希字段失败: %v", err)
			return
		}
	}

	// 获取单个字段
	name, err := hashOps.HGet(ctx, key, "name")
	if err != nil {
		log.Printf("获取哈希字段失败: %v", err)
		return
	}
	log.Printf("Hash - 获取单个字段: name = %s", name)

	// 获取所有字段
	allFields, err := hashOps.HGetAll(ctx, key)
	if err != nil {
		log.Printf("获取所有哈希字段失败: %v", err)
		return
	}
	log.Printf("Hash - 获取所有字段: %v", allFields)

	// 检查字段是否存在
	exists, err := hashOps.HExists(ctx, key, "email")
	if err != nil {
		log.Printf("检查哈希字段存在性失败: %v", err)
		return
	}
	log.Printf("Hash - 字段存在性检查: email 存在 = %v", exists)

	// 获取哈希长度
	length, err := hashOps.HLen(ctx, key)
	if err != nil {
		log.Printf("获取哈希长度失败: %v", err)
		return
	}
	log.Printf("Hash - 哈希长度: %d", length)
}

func demoSetOperations(ctx context.Context, client cache.Provider) {
	setOps := client.Set()

	key := "demo:set:users"

	// 添加成员
	members := []string{"user1", "user2", "user3", "user1"} // user1 重复

	for _, member := range members {
		err := setOps.SAdd(ctx, key, member)
		if err != nil {
			log.Printf("添加集合成员失败: %v", err)
			return
		}
	}

	// 检查成员
	for _, member := range []string{"user1", "user4"} {
		isMember, err := setOps.SIsMember(ctx, key, member)
		if err != nil {
			log.Printf("检查集合成员失败: %v", err)
			return
		}
		log.Printf("Set - 成员检查: %s 存在 = %v", member, isMember)
	}
}

func demoLockOperations(ctx context.Context, client cache.Provider) {
	lockOps := client.Lock()

	lockKey := "demo:lock:resource"

	// 获取锁
	lock, err := lockOps.Acquire(ctx, lockKey, 10*time.Second)
	if err != nil {
		log.Printf("获取锁失败: %v", err)
		return
	}

	if lock == nil {
		log.Printf("锁已被占用")
		return
	}

	log.Printf("Lock - 成功获取锁")

	// 模拟工作
	time.Sleep(1 * time.Second)

	// 释放锁
	err = lock.Unlock(ctx)
	if err != nil {
		log.Printf("释放锁失败: %v", err)
		return
	}

	log.Printf("Lock - 锁已释放")

	// 尝试再次获取锁
	lock, err = lockOps.Acquire(ctx, lockKey, 5*time.Second)
	if err != nil {
		log.Printf("再次获取锁失败: %v", err)
		return
	}

	if lock != nil {
		log.Printf("Lock - 成功重新获取锁")
		lock.Unlock(ctx)
	}
}

func demoBloomOperations(ctx context.Context, client cache.Provider) {
	bloomOps := client.Bloom()

	key := "demo:bloom:emails"

	// 初始化布隆过滤器
	err := bloomOps.BFReserve(ctx, key, 0.01, 1000)
	if err != nil {
		if err.Error() == "redis server does not support bloom filter commands (RedisBloom module may not be installed)" {
			log.Printf("Bloom - 跳过布隆过滤器演示: %v", err)
			return
		}
		log.Printf("初始化布隆过滤器失败: %v", err)
		return
	}

	log.Printf("Bloom - 布隆过滤器初始化成功")

	// 添加元素
	emails := []string{"user1@example.com", "user2@example.com"}
	for _, email := range emails {
		err := bloomOps.BFAdd(ctx, key, email)
		if err != nil {
			log.Printf("添加到布隆过滤器失败: %v", err)
			return
		}
		log.Printf("Bloom - 添加邮箱: %s", email)
	}

	// 检查元素
	testEmails := []string{
		"user1@example.com", // 存在
		"user3@example.com", // 不存在
	}

	for _, email := range testEmails {
		exists, err := bloomOps.BFExists(ctx, key, email)
		if err != nil {
			log.Printf("检查布隆过滤器失败: %v", err)
			return
		}
		log.Printf("Bloom - 邮箱 '%s' 存在: %v", email, exists)
	}
}

func demoScriptOperations(ctx context.Context, client cache.Provider) {
	scriptOps := client.Script()

	// 简单的 Lua 脚本：返回 "HELLO " + ARGV[1]
	simpleScript := `
		return "HELLO " .. ARGV[1]
	`

	// 加载脚本
	sha1, err := scriptOps.ScriptLoad(ctx, simpleScript)
	if err != nil {
		log.Printf("加载脚本失败: %v", err)
		return
	}
	log.Printf("Script - 脚本加载成功, SHA1: %s", sha1)

	// 检查脚本是否存在
	exists, err := scriptOps.ScriptExists(ctx, sha1)
	if err != nil {
		log.Printf("检查脚本存在性失败: %v", err)
		return
	}
	log.Printf("Script - 脚本存在: %v", exists)

	// 执行脚本
	result, err := scriptOps.EvalSha(ctx, sha1, []string{}, "World")
	if err != nil {
		log.Printf("执行脚本失败: %v", err)
		return
	}
	log.Printf("Script - 脚本执行结果: %s", result)
}

func demoErrorHandling(ctx context.Context, client cache.Provider) {
	stringOps := client.String()
	hashOps := client.Hash()

	// 测试不存在的键
	nonexistentKey := "demo:nonexistent:key"
	_, err := stringOps.Get(ctx, nonexistentKey)
	if err == cache.ErrCacheMiss {
		log.Printf("Error - 正确处理了不存在的键: %v", err)
	} else if err != nil {
		log.Printf("Error - 意外的错误: %v", err)
	}

	// 测试不存在的哈希字段
	_, err = hashOps.HGet(ctx, "demo:nonexistent:hash", "field")
	if err == cache.ErrCacheMiss {
		log.Printf("Error - 正确处理了不存在的哈希字段: %v", err)
	} else if err != nil {
		log.Printf("Error - 意外的错误: %v", err)
	}

	log.Printf("Error - 错误处理演示完成")
}

func demoZSetOperations(ctx context.Context, client cache.Provider) {
	zsetOps := client.ZSet()

	// 模拟会话消息管理：维护最近50条消息记录
	sessionKey := "demo:zset:session:12345"

	// 添加消息到会话（使用时间戳作为分数）
	now := time.Now()
	messages := []struct {
		id      string
		content string
		score   float64
	}{
		{"msg1", "你好", float64(now.Add(-10 * time.Minute).Unix())},
		{"msg2", "最近怎么样？", float64(now.Add(-8 * time.Minute).Unix())},
		{"msg3", "我很好，谢谢！", float64(now.Add(-6 * time.Minute).Unix())},
		{"msg4", "今天天气不错", float64(now.Add(-4 * time.Minute).Unix())},
		{"msg5", "是啊，很适合出去走走", float64(now.Add(-2 * time.Minute).Unix())},
	}

	// 添加消息到ZSET
	for _, msg := range messages {
		member := &cache.ZMember{
			Member: msg.id,
			Score:  msg.score,
		}
		err := zsetOps.ZAdd(ctx, sessionKey, member)
		if err != nil {
			log.Printf("添加消息失败: %v", err)
			return
		}
		log.Printf("ZSet - 添加消息: ID=%s, 时间戳=%f", msg.id, msg.score)
	}

	// 设置过期时间（1小时）
	err := zsetOps.ZSetExpire(ctx, sessionKey, time.Hour)
	if err != nil {
		log.Printf("设置ZSET过期时间失败: %v", err)
		return
	}
	log.Printf("ZSet - 设置过期时间: 1小时")

	// 获取消息总数
	count, err := zsetOps.ZCard(ctx, sessionKey)
	if err != nil {
		log.Printf("获取消息总数失败: %v", err)
		return
	}
	log.Printf("ZSet - 会话消息总数: %d", count)

	// 按时间从早到晚获取消息（ZRange）
	earlyMessages, err := zsetOps.ZRange(ctx, sessionKey, 0, -1)
	if err != nil {
		log.Printf("获取早到晚消息失败: %v", err)
		return
	}
	log.Printf("ZSet - 按时间从早到晚的消息（%d条）:", len(earlyMessages))
	for i, msg := range earlyMessages {
		log.Printf("  [%d] ID=%s, 时间戳=%f", i+1, msg.Member, msg.Score)
	}

	// 按时间从晚到早获取消息（ZRevRange）- 最新的在前
	recentMessages, err := zsetOps.ZRevRange(ctx, sessionKey, 0, 4) // 获取最新的5条
	if err != nil {
		log.Printf("获取最新消息失败: %v", err)
		return
	}
	log.Printf("ZSet - 最新的5条消息:")
	for i, msg := range recentMessages {
		log.Printf("  [%d] ID=%s, 时间戳=%f", i+1, msg.Member, msg.Score)
	}

	// 获取特定分数范围内的消息（最近1小时内的）
	oneHourAgo := float64(now.Add(-time.Hour).Unix())
	recentMessagesByScore, err := zsetOps.ZRangeByScore(ctx, sessionKey, oneHourAgo, float64(now.Unix()))
	if err != nil {
		log.Printf("按分数范围获取消息失败: %v", err)
		return
	}
	log.Printf("ZSet - 最近1小时内的消息（%d条）:", len(recentMessagesByScore))
	for i, msg := range recentMessagesByScore {
		log.Printf("  [%d] ID=%s, 时间戳=%f", i+1, msg.Member, msg.Score)
	}

	// 获取特定消息的分数（时间戳）
	msgID := "msg3"
	score, err := zsetOps.ZScore(ctx, sessionKey, msgID)
	if err != nil {
		log.Printf("获取消息分数失败: %v", err)
		return
	}
	log.Printf("ZSet - 消息 '%s' 的时间戳: %f", msgID, score)

	// 移除旧消息（模拟维护最近50条的限制）
	// 这里演示移除最早的消息
	if count > 3 {
		err := zsetOps.ZRemRangeByRank(ctx, sessionKey, 0, 0) // 移除排名最早的一条
		if err != nil {
			log.Printf("移除旧消息失败: %v", err)
			return
		}
		log.Printf("ZSet - 移除了 1 条旧消息")
	}

	// 验证移除后的消息数量
	newCount, err := zsetOps.ZCard(ctx, sessionKey)
	if err != nil {
		log.Printf("获取新的消息总数失败: %v", err)
		return
	}
	log.Printf("ZSet - 移除后剩余消息数: %d", newCount)
}