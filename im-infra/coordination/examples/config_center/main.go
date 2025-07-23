package main

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/ceyewan/gochat/im-infra/coordination"
)

func main() {
	fmt.Println("=== 配置中心示例 ===")

	// 创建协调器
	cfg := coordination.ExampleConfig()
	coordinator, err := coordination.New(cfg)
	if err != nil {
		log.Printf("创建协调器失败 (请确保 etcd 正在运行): %v", err)
		return
	}
	defer coordinator.Close()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// 测试连接
	if err := coordinator.Ping(ctx); err != nil {
		log.Printf("连接 etcd 失败: %v", err)
		return
	}
	fmt.Println("✓ 连接 etcd 成功")

	// 获取配置中心
	configCenter := coordinator.ConfigCenter()

	// 1. 基础配置操作
	fmt.Println("\n1. 基础配置操作:")
	basicConfigDemo(ctx, configCenter)

	// 2. 版本控制示例
	fmt.Println("\n2. 版本控制示例:")
	versionControlDemo(ctx, configCenter)

	// 3. 配置监听示例
	fmt.Println("\n3. 配置监听示例:")
	configWatchDemo(ctx, configCenter)

	// 4. 批量配置操作
	fmt.Println("\n4. 批量配置操作:")
	batchConfigDemo(ctx, configCenter)

	// 5. 模块配置示例
	fmt.Println("\n5. 模块配置示例:")
	moduleConfigDemo(ctx)

	// 6. 配置历史示例
	fmt.Println("\n6. 配置历史示例:")
	configHistoryDemo(ctx, configCenter)

	fmt.Println("\n=== 配置中心示例完成 ===")
}

func basicConfigDemo(ctx context.Context, configCenter coordination.ConfigCenter) {
	// 设置简单配置
	configs := map[string]interface{}{
		"app.name":      "my-application",
		"app.version":   "1.0.0",
		"app.debug":     true,
		"app.port":      8080,
		"database.host": "localhost",
		"database.port": 5432,
		"database.name": "myapp",
		"redis.host":    "localhost",
		"redis.port":    6379,
		"redis.timeout": "5s",
	}

	for key, value := range configs {
		if err := configCenter.Set(ctx, key, value, 0); err != nil {
			log.Printf("设置配置失败 %s: %v", key, err)
		} else {
			fmt.Printf("✓ 设置配置: %s = %v\n", key, value)
		}
	}

	// 获取配置
	fmt.Println("\n获取配置:")
	for key := range configs {
		config, err := configCenter.Get(ctx, key)
		if err != nil {
			log.Printf("获取配置失败 %s: %v", key, err)
			continue
		}
		fmt.Printf("  %s = %s (版本: %d)\n", config.Key, config.Value, config.Version)
	}
}

func versionControlDemo(ctx context.Context, configCenter coordination.ConfigCenter) {
	key := "app.feature.enabled"

	// 设置初始版本
	if err := configCenter.Set(ctx, key, false, 0); err != nil {
		log.Printf("设置初始配置失败: %v", err)
		return
	}
	fmt.Printf("✓ 设置初始配置: %s = false\n", key)

	// 获取当前版本
	version, err := configCenter.GetVersion(ctx, key)
	if err != nil {
		log.Printf("获取版本失败: %v", err)
		return
	}
	fmt.Printf("✓ 当前版本: %d\n", version)

	// 更新配置（使用正确的版本）
	if err := configCenter.Set(ctx, key, true, version+1); err != nil {
		log.Printf("更新配置失败: %v", err)
	} else {
		fmt.Printf("✓ 更新配置: %s = true (版本: %d)\n", key, version+1)
	}

	// 尝试使用过期版本更新（应该失败）
	if err := configCenter.Set(ctx, key, false, version); err != nil {
		fmt.Printf("✓ 预期的版本冲突错误: %v\n", err)
	} else {
		fmt.Println("意外成功：应该发生版本冲突")
	}

	// 获取最新配置
	config, err := configCenter.Get(ctx, key)
	if err != nil {
		log.Printf("获取最新配置失败: %v", err)
	} else {
		fmt.Printf("✓ 最新配置: %s = %s (版本: %d)\n", config.Key, config.Value, config.Version)
	}
}

func configWatchDemo(ctx context.Context, configCenter coordination.ConfigCenter) {
	key := "app.dynamic.config"

	// 启动配置监听
	watchCh, err := configCenter.Watch(ctx, key)
	if err != nil {
		log.Printf("启动配置监听失败: %v", err)
		return
	}

	// 启动监听 goroutine
	go func() {
		for {
			select {
			case change, ok := <-watchCh:
				if !ok {
					return
				}
				fmt.Printf("📡 配置变更通知: %s\n", change.Key)
				fmt.Printf("   类型: %s\n", change.Type.String())
				if change.OldValue != nil {
					fmt.Printf("   旧值: %s (版本: %d)\n", change.OldValue.Value, change.OldValue.Version)
				}
				if change.NewValue != nil {
					fmt.Printf("   新值: %s (版本: %d)\n", change.NewValue.Value, change.NewValue.Version)
				}
				fmt.Printf("   时间: %s\n", change.Timestamp.Format(time.RFC3339))
			case <-ctx.Done():
				return
			}
		}
	}()

	// 等待监听器启动
	time.Sleep(1 * time.Second)

	// 创建配置
	if err := configCenter.Set(ctx, key, "initial-value", 0); err != nil {
		log.Printf("创建配置失败: %v", err)
	} else {
		fmt.Printf("✓ 创建配置: %s = initial-value\n", key)
	}

	time.Sleep(1 * time.Second)

	// 更新配置
	if err := configCenter.Set(ctx, key, "updated-value", 0); err != nil {
		log.Printf("更新配置失败: %v", err)
	} else {
		fmt.Printf("✓ 更新配置: %s = updated-value\n", key)
	}

	time.Sleep(1 * time.Second)

	// 删除配置
	version, _ := configCenter.GetVersion(ctx, key)
	if err := configCenter.Delete(ctx, key, version); err != nil {
		log.Printf("删除配置失败: %v", err)
	} else {
		fmt.Printf("✓ 删除配置: %s\n", key)
	}

	time.Sleep(1 * time.Second)
}

func batchConfigDemo(ctx context.Context, configCenter coordination.ConfigCenter) {
	// 设置一组相关配置
	prefix := "microservice.user"
	configs := map[string]interface{}{
		prefix + ".database.host":    "user-db.example.com",
		prefix + ".database.port":    5432,
		prefix + ".database.name":    "users",
		prefix + ".cache.host":       "user-cache.example.com",
		prefix + ".cache.port":       6379,
		prefix + ".api.rate_limit":   1000,
		prefix + ".api.timeout":      "30s",
		prefix + ".feature.new_auth": true,
	}

	// 批量设置配置
	for key, value := range configs {
		if err := configCenter.Set(ctx, key, value, 0); err != nil {
			log.Printf("设置配置失败 %s: %v", key, err)
		}
	}
	fmt.Printf("✓ 批量设置 %d 个配置\n", len(configs))

	// 监听前缀变化
	watchCh, err := configCenter.WatchPrefix(ctx, prefix)
	if err != nil {
		log.Printf("监听前缀变化失败: %v", err)
		return
	}

	// 启动前缀监听 goroutine
	go func() {
		for {
			select {
			case change, ok := <-watchCh:
				if !ok {
					return
				}
				fmt.Printf("📡 前缀变更通知: %s (%s)\n", change.Key, change.Type.String())
			case <-ctx.Done():
				return
			}
		}
	}()

	time.Sleep(1 * time.Second)

	// 更新其中一个配置
	updateKey := prefix + ".api.rate_limit"
	if err := configCenter.Set(ctx, updateKey, 2000, 0); err != nil {
		log.Printf("更新配置失败: %v", err)
	} else {
		fmt.Printf("✓ 更新配置: %s = 2000\n", updateKey)
	}

	time.Sleep(1 * time.Second)

	// 清理配置
	for key := range configs {
		version, _ := configCenter.GetVersion(ctx, key)
		if err := configCenter.Delete(ctx, key, version); err != nil {
			log.Printf("删除配置失败 %s: %v", key, err)
		}
	}
	fmt.Printf("✓ 清理 %d 个配置\n", len(configs))

	time.Sleep(1 * time.Second)
}

func moduleConfigDemo(ctx context.Context) {
	// 使用模块特定的配置中心
	userServiceCoordinator := coordination.Module("user-service")
	orderServiceCoordinator := coordination.Module("order-service")

	userConfigCenter := userServiceCoordinator.ConfigCenter()
	orderConfigCenter := orderServiceCoordinator.ConfigCenter()

	// 不同模块可以使用相同的配置键名，但实际上是隔离的
	configKey := "database.host"

	// 用户服务配置
	if err := userConfigCenter.Set(ctx, configKey, "user-db.example.com", 0); err != nil {
		log.Printf("设置用户服务配置失败: %v", err)
	} else {
		fmt.Printf("✓ 用户服务配置: %s = user-db.example.com\n", configKey)
	}

	// 订单服务配置
	if err := orderConfigCenter.Set(ctx, configKey, "order-db.example.com", 0); err != nil {
		log.Printf("设置订单服务配置失败: %v", err)
	} else {
		fmt.Printf("✓ 订单服务配置: %s = order-db.example.com\n", configKey)
	}

	// 验证配置隔离
	userConfig, err := userConfigCenter.Get(ctx, configKey)
	if err != nil {
		log.Printf("获取用户服务配置失败: %v", err)
	} else {
		fmt.Printf("✓ 用户服务读取: %s = %s\n", configKey, userConfig.Value)
	}

	orderConfig, err := orderConfigCenter.Get(ctx, configKey)
	if err != nil {
		log.Printf("获取订单服务配置失败: %v", err)
	} else {
		fmt.Printf("✓ 订单服务读取: %s = %s\n", configKey, orderConfig.Value)
	}

	// 清理
	userVersion, _ := userConfigCenter.GetVersion(ctx, configKey)
	userConfigCenter.Delete(ctx, configKey, userVersion)

	orderVersion, _ := orderConfigCenter.GetVersion(ctx, configKey)
	orderConfigCenter.Delete(ctx, configKey, orderVersion)
}

func configHistoryDemo(ctx context.Context, configCenter coordination.ConfigCenter) {
	key := "app.version"

	// 先删除可能已存在的配置，避免版本冲突
	if currentVersion, err := configCenter.GetVersion(ctx, key); err == nil {
		configCenter.Delete(ctx, key, currentVersion)
		fmt.Printf("✓ 清理已存在的配置: %s\n", key)
	}

	// 创建多个版本的配置，使用版本号0让系统自动生成新版本号
	versions := []string{"1.0.0", "1.1.0", "1.2.0", "2.0.0", "2.1.0"}

	for _, version := range versions {
		// 使用版本号0，让系统自动生成新版本号
		if err := configCenter.Set(ctx, key, version, 0); err != nil {
			log.Printf("设置版本 %s 失败: %v", version, err)
		} else {
			fmt.Printf("✓ 设置版本: %s = %s\n", key, version)
		}
		time.Sleep(100 * time.Millisecond) // 确保时间戳不同
	}

	// 获取配置历史
	history, err := configCenter.GetHistory(ctx, key, 10)
	if err != nil {
		log.Printf("获取配置历史失败: %v", err)
		return
	}

	fmt.Printf("✓ 配置历史 (%d 个版本):\n", len(history))
	for _, h := range history {
		fmt.Printf("  版本 %d: %s (%s)\n", h.Version, h.Value, h.CreateTime.Format("15:04:05"))
	}

	// 清理
	currentVersion, _ := configCenter.GetVersion(ctx, key)
	configCenter.Delete(ctx, key, currentVersion)
}
