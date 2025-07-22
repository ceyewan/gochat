package main

import (
	"context"
	"fmt"
	"time"

	"github.com/ceyewan/gochat/im-infra/cache"
	"github.com/ceyewan/gochat/im-infra/clog"
)

func main() {
	// 初始化日志
	clog.Info("=== Cache 基础功能演示 ===")

	ctx := context.Background()

	// 1. 全局缓存方法演示
	demonstrateGlobalMethods(ctx)

	// 2. 自定义缓存实例演示
	demonstrateCustomInstances(ctx)

	// 3. 字符串操作演示
	demonstrateStringOperations(ctx)

	// 4. 哈希操作演示
	demonstrateHashOperations(ctx)

	// 5. 集合操作演示
	demonstrateSetOperations(ctx)

	// 6. 自定义配置演示
	demonstrateCustomConfig(ctx)

	clog.Info("=== 演示完成 ===")
}

func demonstrateGlobalMethods(ctx context.Context) {
	clog.Info("=== 全局缓存方法演示 ===")

	// 基本的字符串操作
	err := cache.Set(ctx, "user:123", "John Doe", time.Hour)
	if err != nil {
		clog.Error("设置缓存失败", clog.ErrorValue(err))
		return
	}

	value, err := cache.Get(ctx, "user:123")
	if err != nil {
		clog.Error("获取缓存失败", clog.ErrorValue(err))
		return
	}
	clog.Info("获取用户信息", clog.String("user", value))

	// 数值操作
	err = cache.Set(ctx, "counter", 0, time.Hour)
	if err != nil {
		clog.Error("设置计数器失败", clog.ErrorValue(err))
		return
	}

	newValue, err := cache.Incr(ctx, "counter")
	if err != nil {
		clog.Error("递增计数器失败", clog.ErrorValue(err))
		return
	}
	clog.Info("计数器递增", clog.Int64("value", newValue))

	// 过期时间操作
	err = cache.Expire(ctx, "user:123", time.Minute*30)
	if err != nil {
		clog.Error("设置过期时间失败", clog.ErrorValue(err))
		return
	}

	ttl, err := cache.TTL(ctx, "user:123")
	if err != nil {
		clog.Error("获取TTL失败", clog.ErrorValue(err))
		return
	}
	clog.Info("获取TTL", clog.Duration("ttl", ttl))

	clog.Info("全局缓存方法演示完成")
}

func demonstrateCustomInstances(ctx context.Context) {
	clog.Info("=== 自定义缓存实例演示 ===")

	// 创建不同的缓存实例
	userCfg := cache.NewConfigBuilder().
		Addr("localhost:6379").
		DB(0).
		KeyPrefix("user").
		Build()
	userCache, err := cache.New(userCfg)
	if err != nil {
		clog.Error("创建用户缓存失败", clog.ErrorValue(err))
		return
	}

	sessionCfg := cache.NewConfigBuilder().
		Addr("localhost:6379").
		DB(0).
		KeyPrefix("session").
		Build()
	sessionCache, err := cache.New(sessionCfg)
	if err != nil {
		clog.Error("创建会话缓存失败", clog.ErrorValue(err))
		return
	}

	// 用户缓存操作
	userData := map[string]interface{}{
		"id":    123,
		"name":  "John Doe",
		"email": "john@example.com",
	}
	err = userCache.Set(ctx, "123", fmt.Sprintf("%+v", userData), time.Hour)
	if err != nil {
		clog.Error("设置用户缓存失败", clog.ErrorValue(err))
		return
	}

	// 会话缓存操作
	sessionData := "session_token_abc123"
	err = sessionCache.Set(ctx, "abc", sessionData, time.Minute*30)
	if err != nil {
		clog.Error("设置会话缓存失败", clog.ErrorValue(err))
		return
	}

	// 验证缓存实例独立性
	user, err := userCache.Get(ctx, "123")
	if err != nil {
		clog.Error("获取用户缓存失败", clog.ErrorValue(err))
		return
	}
	clog.Info("获取用户缓存数据", clog.String("user", user))

	session, err := sessionCache.Get(ctx, "abc")
	if err != nil {
		clog.Error("获取会话缓存失败", clog.ErrorValue(err))
		return
	}
	clog.Info("获取会话缓存数据", clog.String("session", session))

	clog.Info("自定义缓存实例演示完成")
}

func demonstrateStringOperations(ctx context.Context) {
	clog.Info("=== 字符串操作演示 ===")

	// 基本设置和获取
	cache.Set(ctx, "string:basic", "Hello, Redis!", time.Hour)
	value, _ := cache.Get(ctx, "string:basic")
	clog.Info("基本字符串操作", clog.String("value", value))

	// 数值递增递减
	cache.Set(ctx, "string:number", 10, time.Hour)
	newVal, _ := cache.Incr(ctx, "string:number")
	clog.Info("递增操作", clog.Int64("value", newVal))

	newVal, _ = cache.Decr(ctx, "string:number")
	clog.Info("递减操作", clog.Int64("value", newVal))

	// 批量删除和检查
	cache.Set(ctx, "string:temp1", "value1", time.Hour)
	cache.Set(ctx, "string:temp2", "value2", time.Hour)

	count, _ := cache.Exists(ctx, "string:temp1", "string:temp2", "string:nonexistent")
	clog.Info("检查键存在", clog.Int64("count", count))

	cache.Del(ctx, "string:temp1", "string:temp2")
	clog.Info("删除临时键完成")

	clog.Info("字符串操作演示完成")
}

func demonstrateHashOperations(ctx context.Context) {
	clog.Info("=== 哈希操作演示 ===")

	// 设置哈希字段
	cache.HSet(ctx, "user:456:profile", "name", "Jane Smith")
	cache.HSet(ctx, "user:456:profile", "email", "jane@example.com")
	cache.HSet(ctx, "user:456:profile", "age", 28)
	cache.HSet(ctx, "user:456:profile", "city", "New York")

	// 获取单个字段
	name, _ := cache.HGet(ctx, "user:456:profile", "name")
	clog.Info("获取哈希字段", clog.String("name", name))

	// 获取所有字段
	profile, _ := cache.HGetAll(ctx, "user:456:profile")
	clog.Info("获取所有哈希字段", clog.Any("profile", profile))

	// 检查字段存在
	exists, _ := cache.HExists(ctx, "user:456:profile", "email")
	clog.Info("检查哈希字段存在", clog.Bool("exists", exists))

	// 获取字段数量
	count, _ := cache.HLen(ctx, "user:456:profile")
	clog.Info("哈希字段数量", clog.Int64("count", count))

	// 删除字段
	cache.HDel(ctx, "user:456:profile", "age")
	clog.Info("删除哈希字段完成")

	clog.Info("哈希操作演示完成")
}

func demonstrateSetOperations(ctx context.Context) {
	clog.Info("=== 集合操作演示 ===")

	// 添加集合成员
	cache.SAdd(ctx, "user:789:tags", "developer", "golang", "redis", "backend")

	// 检查成员存在
	isMember, _ := cache.SIsMember(ctx, "user:789:tags", "golang")
	clog.Info("检查集合成员", clog.Bool("isMember", isMember))

	// 获取所有成员
	members, _ := cache.SMembers(ctx, "user:789:tags")
	clog.Info("获取集合成员", clog.Strings("members", members))

	// 获取成员数量
	count, _ := cache.SCard(ctx, "user:789:tags")
	clog.Info("集合成员数量", clog.Int64("count", count))

	// 移除成员
	cache.SRem(ctx, "user:789:tags", "backend")
	clog.Info("移除集合成员完成")

	// 再次获取成员验证
	members, _ = cache.SMembers(ctx, "user:789:tags")
	clog.Info("移除后的集合成员", clog.Strings("members", members))

	clog.Info("集合操作演示完成")
}

func demonstrateCustomConfig(ctx context.Context) {
	clog.Info("=== 自定义配置演示 ===")

	// 使用配置构建器
	cfg := cache.NewConfigBuilder().
		Addr("localhost:6379").
		DB(1). // 使用不同的数据库
		PoolSize(5).
		KeyPrefix("demo").
		Build()

	// 创建自定义缓存实例
	customCache, err := cache.New(cfg)
	if err != nil {
		clog.Error("创建自定义缓存失败", clog.ErrorValue(err))
		return
	}

	// 使用自定义缓存
	err = customCache.Set(ctx, "custom:key", "custom value", time.Hour)
	if err != nil {
		clog.Error("设置自定义缓存失败", clog.ErrorValue(err))
		return
	}

	value, err := customCache.Get(ctx, "custom:key")
	if err != nil {
		clog.Error("获取自定义缓存失败", clog.ErrorValue(err))
		return
	}
	clog.Info("自定义缓存操作", clog.String("value", value))

	// 测试连接
	err = customCache.Ping(ctx)
	if err != nil {
		clog.Error("自定义缓存连接测试失败", clog.ErrorValue(err))
		return
	}
	clog.Info("自定义缓存连接正常")

	clog.Info("自定义配置演示完成")
}
