package main

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/ceyewan/gochat/im-infra/coordination"
)

func main() {
	// 创建协调器实例，使用用户的 etcd 集群
	opts := coordination.CoordinatorOptions{
		Endpoints: []string{"localhost:23791", "localhost:23792", "localhost:23793"},
		Timeout:   5 * time.Second,
	}

	coord, err := coordination.NewCoordinator(opts)
	if err != nil {
		fmt.Printf("创建协调器失败: %v\n", err)
		os.Exit(1)
	}
	defer coord.Close()

	ctx := context.Background()

	fmt.Println("=== Coordination 模块基础使用示例 ===")

	// 分布式锁示例
	fmt.Println("\n--- 分布式锁示例 ---")
	if err := lockExample(ctx, coord); err != nil {
		fmt.Printf("分布式锁示例失败: %v\n", err)
	}

	// 配置中心示例
	fmt.Println("\n--- 配置中心示例 ---")
	if err := configExample(ctx, coord); err != nil {
		fmt.Printf("配置中心示例失败: %v\n", err)
	}

	// 服务注册发现示例
	fmt.Println("\n--- 服务注册发现示例 ---")
	if err := registryExample(ctx, coord); err != nil {
		fmt.Printf("服务注册发现示例失败: %v\n", err)
	}

	// 全局方法示例
	fmt.Println("\n--- 全局方法示例 ---")
	if err := globalMethodsExample(ctx); err != nil {
		fmt.Printf("全局方法示例失败: %v\n", err)
	}

	fmt.Println("\n=== 示例执行完成 ===")
}

// lockExample 分布式锁示例
func lockExample(ctx context.Context, coord coordination.Coordinator) error {
	lockService := coord.Lock()

	// 获取锁
	fmt.Println("正在获取分布式锁...")
	lock, err := lockService.Acquire(ctx, "example-lock", 30*time.Second)
	if err != nil {
		return fmt.Errorf("获取锁失败: %w", err)
	}
	fmt.Printf("锁获取成功，键名: %s\n", lock.Key())

	// 稍微等待一下，确保锁的租约已经稳定
	time.Sleep(100 * time.Millisecond)

	// 检查锁的 TTL
	ttl, err := lock.TTL(ctx)
	if err != nil {
		fmt.Printf("获取锁 TTL 失败: %v\n", err)
	} else {
		fmt.Printf("锁剩余时间: %v\n", ttl)
	}

	// 模拟业务逻辑
	fmt.Println("执行受锁保护的业务逻辑...")
	time.Sleep(2 * time.Second)

	// 续期锁
	fmt.Println("续期锁...")
	if err := lock.Renew(ctx, 30*time.Second); err != nil {
		fmt.Printf("锁续期失败: %v\n", err)
	} else {
		fmt.Println("锁续期成功")
	}

	// 释放锁
	fmt.Println("释放锁...")
	if err := lock.Unlock(ctx); err != nil {
		return fmt.Errorf("释放锁失败: %w", err)
	}
	fmt.Println("锁释放成功")

	// 尝试非阻塞获取锁
	fmt.Println("\n尝试非阻塞获取锁...")
	tryLock, err := lockService.TryAcquire(ctx, "try-lock", 15*time.Second)
	if err != nil {
		fmt.Printf("尝试获取锁失败: %v\n", err)
	} else {
		fmt.Printf("非阻塞锁获取成功，键名: %s\n", tryLock.Key())
		tryLock.Unlock(ctx)
	}

	return nil
}

// configExample 配置中心示例
func configExample(ctx context.Context, coord coordination.Coordinator) error {
	configService := coord.Config()

	// 设置简单字符串配置
	fmt.Println("设置字符串配置...")
	if err := configService.Set(ctx, "app.name", "gochat"); err != nil {
		return fmt.Errorf("设置配置失败: %w", err)
	}
	fmt.Println("字符串配置设置成功")

	// 设置 JSON 对象配置
	fmt.Println("设置 JSON 对象配置...")
	dbConfig := map[string]interface{}{
		"host":     "localhost",
		"port":     3306,
		"database": "gochat",
		"username": "root",
		"password": "123456",
		"charset":  "utf8mb4",
		"timeout":  "30s",
	}
	if err := configService.Set(ctx, "database.mysql", dbConfig); err != nil {
		return fmt.Errorf("设置数据库配置失败: %w", err)
	}
	fmt.Println("JSON 对象配置设置成功")

	// 获取配置
	fmt.Println("获取配置...")
	appName, err := configService.Get(ctx, "app.name")
	if err != nil {
		return fmt.Errorf("获取应用名配置失败: %w", err)
	}
	fmt.Printf("应用名: %v\n", appName)

	dbConf, err := configService.Get(ctx, "database.mysql")
	if err != nil {
		return fmt.Errorf("获取数据库配置失败: %w", err)
	}
	fmt.Printf("数据库配置: %v\n", dbConf)

	// 列出配置键
	fmt.Println("列出所有配置键...")
	keys, err := configService.List(ctx, "")
	if err != nil {
		return fmt.Errorf("列出配置键失败: %w", err)
	}
	fmt.Printf("配置键列表: %v\n", keys)

	// 监听配置变化
	fmt.Println("开始监听配置变化...")
	watchCtx, cancel := context.WithTimeout(ctx, 3*time.Second)
	defer cancel()

	watchCh, err := configService.Watch(watchCtx, "app.name")
	if err != nil {
		return fmt.Errorf("监听配置变化失败: %w", err)
	}

	// 在另一个 goroutine 中监听事件
	go func() {
		for event := range watchCh {
			fmt.Printf("配置变化事件: 类型=%s, 键=%s, 值=%v, 时间=%s\n",
				event.Type, event.Key, event.Value, event.Timestamp.Format("15:04:05"))
		}
	}()

	// 更新配置触发监听
	time.Sleep(500 * time.Millisecond)
	configService.Set(ctx, "app.name", "gochat-v2")
	time.Sleep(500 * time.Millisecond)

	// 删除配置
	fmt.Println("删除配置...")
	if err := configService.Delete(ctx, "app.name"); err != nil {
		fmt.Printf("删除配置失败: %v\n", err)
	} else {
		fmt.Println("配置删除成功")
	}

	time.Sleep(500 * time.Millisecond)
	return nil
}

// registryExample 服务注册发现示例
func registryExample(ctx context.Context, coord coordination.Coordinator) error {
	registryService := coord.Registry()

	// 注册服务
	fmt.Println("注册服务...")
	service := coordination.ServiceInfo{
		ID:      "chat-service-001",
		Name:    "chat-service",
		Address: "127.0.0.1",
		Port:    8080,
		Metadata: map[string]string{
			"version": "1.0.0",
			"region":  "us-west",
		},
		TTL: 30 * time.Second,
	}

	if err := registryService.Register(ctx, service); err != nil {
		return fmt.Errorf("注册服务失败: %w", err)
	}
	fmt.Printf("服务注册成功: %s\n", service.ID)

	// 发现服务
	fmt.Println("发现服务...")
	services, err := registryService.Discover(ctx, "chat-service")
	if err != nil {
		return fmt.Errorf("发现服务失败: %w", err)
	}
	fmt.Printf("发现 %d 个服务实例:\n", len(services))
	for _, svc := range services {
		fmt.Printf("  服务: ID=%s, 地址=%s:%d, 元数据=%v\n",
			svc.ID, svc.Address, svc.Port, svc.Metadata)
	}

	// 监听服务变化
	fmt.Println("开始监听服务变化...")
	watchCtx, cancel := context.WithTimeout(ctx, 3*time.Second)
	defer cancel()

	watchCh, err := registryService.Watch(watchCtx, "chat-service")
	if err != nil {
		return fmt.Errorf("监听服务变化失败: %w", err)
	}

	// 在另一个 goroutine 中监听事件
	go func() {
		for event := range watchCh {
			fmt.Printf("服务变化事件: 类型=%s, 服务=%s, 时间=%s\n",
				event.Type, event.Service.ID, event.Timestamp.Format("15:04:05"))
		}
	}()

	// 注册另一个服务实例触发监听
	time.Sleep(500 * time.Millisecond)
	service2 := service
	service2.ID = "chat-service-002"
	service2.Port = 8081
	registryService.Register(ctx, service2)
	time.Sleep(500 * time.Millisecond)

	// 注销服务
	fmt.Println("注销服务...")
	if err := registryService.Unregister(ctx, service.ID); err != nil {
		fmt.Printf("注销服务失败: %v\n", err)
	} else {
		fmt.Printf("服务注销成功: %s\n", service.ID)
	}

	if err := registryService.Unregister(ctx, service2.ID); err != nil {
		fmt.Printf("注销服务失败: %v\n", err)
	} else {
		fmt.Printf("服务注销成功: %s\n", service2.ID)
	}

	time.Sleep(500 * time.Millisecond)
	return nil
}

// globalMethodsExample 全局方法示例
func globalMethodsExample(ctx context.Context) error {
	fmt.Println("使用全局方法...")

	// 全局锁方法
	fmt.Println("使用全局锁方法...")
	lock, err := coordination.AcquireLock(ctx, "global-lock", 15*time.Second)
	if err != nil {
		return fmt.Errorf("全局获取锁失败: %w", err)
	}
	fmt.Printf("全局锁获取成功: %s\n", lock.Key())
	lock.Unlock(ctx)

	// 全局配置方法
	fmt.Println("使用全局配置方法...")
	if err := coordination.SetConfig(ctx, "global.setting", "test-value"); err != nil {
		return fmt.Errorf("全局设置配置失败: %w", err)
	}

	value, err := coordination.GetConfig(ctx, "global.setting")
	if err != nil {
		return fmt.Errorf("全局获取配置失败: %w", err)
	}
	fmt.Printf("全局配置值: %v\n", value)

	// 全局服务注册方法
	fmt.Println("使用全局服务注册方法...")
	globalService := coordination.ServiceInfo{
		ID:      "global-service-001",
		Name:    "global-service",
		Address: "127.0.0.1",
		Port:    9090,
		TTL:     15 * time.Second,
	}

	if err := coordination.RegisterService(ctx, globalService); err != nil {
		return fmt.Errorf("全局注册服务失败: %w", err)
	}
	fmt.Printf("全局服务注册成功: %s\n", globalService.ID)

	services, err := coordination.DiscoverServices(ctx, "global-service")
	if err != nil {
		return fmt.Errorf("全局发现服务失败: %w", err)
	}
	fmt.Printf("全局发现 %d 个服务\n", len(services))

	coordination.UnregisterService(ctx, globalService.ID)
	fmt.Println("全局服务注销完成")

	return nil
}
