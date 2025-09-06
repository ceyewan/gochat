package main

import (
	"context"
	"log"
	"time"

	"github.com/ceyewan/gochat/im-infra/cache"
	"github.com/ceyewan/gochat/im-infra/clog"
)

func main() {
	// 使用自定义 Logger
	logger := clog.Module("cache-example")
	ctx := context.Background()

	// 使用默认配置
	cfg := cache.DefaultConfig()
	cfg.Addr = "localhost:6379" // 请确保 Redis 在此地址运行

	// 创建 Cache 实例
	cacheClient, err := cache.New(ctx, cfg, cache.WithLogger(logger))
	if err != nil {
		log.Fatalf("无法创建缓存客户端: %v", err)
	}
	defer cacheClient.Close()

	log.Println("缓存客户端创建成功！")

	// --- 字符串操作 ---
	log.Println("--- 字符串操作 ---")
	key := "mykey"
	value := "hello world"
	expiration := 5 * time.Minute

	err = cacheClient.Set(ctx, key, value, expiration)
	if err != nil {
		log.Fatalf("设置值失败: %v", err)
	}
	log.Printf("成功设置键 '%s'，值为 '%s'，过期时间为 %v", key, value, expiration)

	retrievedValue, err := cacheClient.Get(ctx, key)
	if err != nil {
		log.Fatalf("获取值失败: %v", err)
	}
	log.Printf("成功获取键 '%s'，值为 '%s'", key, retrievedValue)

	if retrievedValue != value {
		log.Fatalf("获取的值与设置的值不匹配！")
	}

	// --- 哈希操作 ---
	log.Println("\n--- 哈希操作 ---")
	hKey := "myhash"
	hField := "field1"
	hValue := "value1"

	err = cacheClient.HSet(ctx, hKey, hField, hValue)
	if err != nil {
		log.Fatalf("设置哈希字段失败: %v", err)
	}
	log.Printf("成功设置哈希键 '%s' 的字段 '%s' 为 '%s'", hKey, hField, hValue)

	retrievedHValue, err := cacheClient.HGet(ctx, hKey, hField)
	if err != nil {
		log.Fatalf("获取哈希字段失败: %v", err)
	}
	log.Printf("成功获取哈希键 '%s' 的字段 '%s'，值为 '%s'", hKey, hField, retrievedHValue)

	if retrievedHValue != hValue {
		log.Fatalf("获取的哈希值与设置的值不匹配！")
	}

	// --- 集合操作 ---
	log.Println("\n--- 集合操作 ---")
	sKey := "myset"
	sMember := "member1"

	err = cacheClient.SAdd(ctx, sKey, sMember)
	if err != nil {
		log.Fatalf("向集合添加成员失败: %v", err)
	}
	log.Printf("成功向集合 '%s' 添加成员 '%s'", sKey, sMember)

	isMember, err := cacheClient.SIsMember(ctx, sKey, sMember)
	if err != nil {
		log.Fatalf("检查集合成员失败: %v", err)
	}
	log.Printf("成员 '%s' 是否在集合 '%s' 中: %v", sMember, sKey, isMember)

	if !isMember {
		log.Fatalf("集合成员检查失败！")
	}

	log.Println("\n示例执行完毕！")
}
