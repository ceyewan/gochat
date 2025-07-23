package main

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/ceyewan/gochat/im-infra/coordination"
)

func main() {
	fmt.Println("=== Coordination Library Basic Example ===")

	// 1. 使用示例配置创建协调器
	fmt.Println("\n1. 创建协调器实例:")
	cfg := coordination.ExampleConfig()
	// 注意：这需要运行的 etcd 实例，在实际使用中请确保 etcd 可用
	fmt.Printf("配置端点: %v, 超时: %v\n", cfg.Endpoints, cfg.DialTimeout)

	coordinator, err := coordination.New(cfg)
	if err != nil {
		log.Printf("创建协调器失败 (这是正常的，如果没有运行 etcd): %v", err)
		fmt.Println("提示：要运行此示例，请先启动 etcd 服务")
		return
	}
	defer coordinator.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// 2. 测试连接
	fmt.Println("\n2. 测试 etcd 连接:")
	if err := coordinator.Ping(ctx); err != nil {
		log.Printf("连接测试失败: %v", err)
		return
	}
	fmt.Println("✓ etcd 连接正常")

	// 3. 服务注册示例
	fmt.Println("\n3. 服务注册示例:")
	registry := coordinator.ServiceRegistry()

	serviceInfo := coordination.ServiceInfo{
		Name:       "example-service",
		InstanceID: "instance-1",
		Address:    "localhost:8080",
		Metadata: map[string]string{
			"version": "1.0.0",
			"region":  "us-west-1",
		},
		Health: coordination.HealthHealthy,
	}

	if err := registry.Register(ctx, serviceInfo); err != nil {
		log.Printf("服务注册失败: %v", err)
		return
	}
	fmt.Printf("✓ 服务注册成功: %s/%s\n", serviceInfo.Name, serviceInfo.InstanceID)

	// 4. 服务发现示例
	fmt.Println("\n4. 服务发现示例:")
	services, err := registry.Discover(ctx, "example-service")
	if err != nil {
		log.Printf("服务发现失败: %v", err)
		return
	}
	fmt.Printf("✓ 发现 %d 个服务实例:\n", len(services))
	for _, svc := range services {
		fmt.Printf("  - %s/%s @ %s (健康状态: %s)\n",
			svc.Name, svc.InstanceID, svc.Address, svc.Health.String())
	}

	// 5. 分布式锁示例
	fmt.Println("\n5. 分布式锁示例:")
	lockManager := coordinator.Lock()

	lock, err := lockManager.Acquire(ctx, "example-lock", 30*time.Second)
	if err != nil {
		log.Printf("获取锁失败: %v", err)
		return
	}
	fmt.Printf("✓ 获取锁成功: %s\n", lock.Key())

	// 检查锁状态
	held, err := lock.IsHeld(ctx)
	if err != nil {
		log.Printf("检查锁状态失败: %v", err)
	} else {
		fmt.Printf("✓ 锁状态: %t\n", held)
	}

	// 释放锁
	if err := lock.Release(ctx); err != nil {
		log.Printf("释放锁失败: %v", err)
	} else {
		fmt.Println("✓ 锁释放成功")
	}

	// 6. 配置中心示例
	fmt.Println("\n6. 配置中心示例:")
	configCenter := coordinator.ConfigCenter()

	// 设置配置
	configKey := "app.database.url"
	configValue := "postgresql://localhost:5432/myapp"
	if err := configCenter.Set(ctx, configKey, configValue, 0); err != nil {
		log.Printf("设置配置失败: %v", err)
		return
	}
	fmt.Printf("✓ 设置配置成功: %s = %s\n", configKey, configValue)

	// 获取配置
	retrievedConfig, err := configCenter.Get(ctx, configKey)
	if err != nil {
		log.Printf("获取配置失败: %v", err)
		return
	}
	fmt.Printf("✓ 获取配置成功: %s = %s (版本: %d)\n",
		retrievedConfig.Key, retrievedConfig.Value, retrievedConfig.Version)

	// 7. 模块特定协调器示例
	fmt.Println("\n7. 模块特定协调器示例:")
	userServiceCoordinator := coordination.Module("user-service")
	userRegistry := userServiceCoordinator.ServiceRegistry()

	userServiceInfo := coordination.ServiceInfo{
		Name:       "user-service",
		InstanceID: "user-instance-1",
		Address:    "localhost:8081",
		Metadata: map[string]string{
			"version": "2.0.0",
			"team":    "backend",
		},
		Health: coordination.HealthHealthy,
	}

	if err := userRegistry.Register(ctx, userServiceInfo); err != nil {
		log.Printf("用户服务注册失败: %v", err)
		return
	}
	fmt.Printf("✓ 用户服务注册成功: %s/%s\n", userServiceInfo.Name, userServiceInfo.InstanceID)

	// 8. 清理
	fmt.Println("\n8. 清理资源:")

	// 注销服务
	if err := registry.Deregister(ctx, serviceInfo.Name, serviceInfo.InstanceID); err != nil {
		log.Printf("注销服务失败: %v", err)
	} else {
		fmt.Println("✓ 服务注销成功")
	}

	if err := userRegistry.Deregister(ctx, userServiceInfo.Name, userServiceInfo.InstanceID); err != nil {
		log.Printf("注销用户服务失败: %v", err)
	} else {
		fmt.Println("✓ 用户服务注销成功")
	}

	// 删除配置
	version, _ := configCenter.GetVersion(ctx, configKey)
	if err := configCenter.Delete(ctx, configKey, version); err != nil {
		log.Printf("删除配置失败: %v", err)
	} else {
		fmt.Println("✓ 配置删除成功")
	}

	fmt.Println("\n=== 示例完成 ===")
	fmt.Println("提示：这个示例展示了 coordination 库的基本功能")
	fmt.Println("在生产环境中，请根据实际需求配置 etcd 集群地址和其他参数")
}
