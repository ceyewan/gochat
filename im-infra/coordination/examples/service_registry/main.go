package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/ceyewan/gochat/im-infra/coordination"
)

func main() {
	fmt.Println("=== 服务注册与发现示例 ===")

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

	// 获取服务注册器
	registry := coordinator.ServiceRegistry()

	// 1. 注册多个服务实例
	fmt.Println("\n1. 注册服务实例:")
	services := []coordination.ServiceInfo{
		{
			Name:       "api-gateway",
			InstanceID: "gateway-1",
			Address:    "192.168.1.10:8080",
			Metadata: map[string]string{
				"version":     "1.0.0",
				"datacenter":  "dc1",
				"environment": "development",
			},
			Health: coordination.HealthHealthy,
		},
		{
			Name:       "api-gateway",
			InstanceID: "gateway-2",
			Address:    "192.168.1.11:8080",
			Metadata: map[string]string{
				"version":     "1.0.0",
				"datacenter":  "dc1",
				"environment": "development",
			},
			Health: coordination.HealthHealthy,
		},
		{
			Name:       "user-service",
			InstanceID: "user-1",
			Address:    "192.168.1.20:8081",
			Metadata: map[string]string{
				"version": "2.1.0",
				"team":    "backend",
			},
			Health: coordination.HealthHealthy,
		},
	}

	for _, service := range services {
		if err := registry.Register(ctx, service); err != nil {
			log.Printf("注册服务失败: %v", err)
			continue
		}
		fmt.Printf("✓ 注册服务: %s/%s @ %s\n", service.Name, service.InstanceID, service.Address)
	}

	// 2. 服务发现
	fmt.Println("\n2. 服务发现:")
	discoveredServices, err := registry.Discover(ctx, "api-gateway")
	if err != nil {
		log.Printf("发现服务失败: %v", err)
		return
	}

	fmt.Printf("发现 %d 个 api-gateway 实例:\n", len(discoveredServices))
	for _, svc := range discoveredServices {
		fmt.Printf("  - %s @ %s (健康状态: %s, 版本: %s)\n",
			svc.InstanceID, svc.Address, svc.Health.String(), svc.Metadata["version"])
	}

	// 3. 监听服务变化
	fmt.Println("\n3. 监听服务变化:")
	watchCh, err := registry.Watch(ctx, "api-gateway")
	if err != nil {
		log.Printf("监听服务变化失败: %v", err)
		return
	}

	// 启动监听 goroutine
	go func() {
		for {
			select {
			case services, ok := <-watchCh:
				if !ok {
					return
				}
				fmt.Printf("📡 服务变化通知: api-gateway 现有 %d 个实例\n", len(services))
				for _, svc := range services {
					fmt.Printf("   - %s @ %s (%s)\n", svc.InstanceID, svc.Address, svc.Health.String())
				}
			case <-ctx.Done():
				return
			}
		}
	}()

	// 4. 模拟健康状态变化
	fmt.Println("\n4. 模拟健康状态变化:")
	time.Sleep(2 * time.Second)

	// 将第一个实例标记为不健康
	err = registry.UpdateHealth(ctx, "api-gateway", "gateway-1", coordination.HealthUnhealthy)
	if err != nil {
		log.Printf("更新健康状态失败: %v", err)
	} else {
		fmt.Println("✓ 将 gateway-1 标记为不健康")
	}

	time.Sleep(2 * time.Second)

	// 恢复健康状态
	err = registry.UpdateHealth(ctx, "api-gateway", "gateway-1", coordination.HealthHealthy)
	if err != nil {
		log.Printf("更新健康状态失败: %v", err)
	} else {
		fmt.Println("✓ 将 gateway-1 恢复为健康")
	}

	// 5. 获取 gRPC 连接（演示负载均衡）
	fmt.Println("\n5. 负载均衡连接:")
	strategies := []coordination.LoadBalanceStrategy{
		coordination.LoadBalanceRoundRobin,
		coordination.LoadBalanceRandom,
	}

	for _, strategy := range strategies {
		conn, err := registry.GetConnection(ctx, "api-gateway", strategy)
		if err != nil {
			log.Printf("获取连接失败 (%s): %v", strategy.String(), err)
			continue
		}
		fmt.Printf("✓ 获取连接成功 (策略: %s)\n", strategy.String())
		conn.Close()
	}

	// 6. 使用模块协调器
	fmt.Println("\n6. 模块协调器示例:")
	monitorCoordinator := coordination.Module("monitor")
	monitorRegistry := monitorCoordinator.ServiceRegistry()

	monitorService := coordination.ServiceInfo{
		Name:       "monitor-service",
		InstanceID: "monitor-1",
		Address:    "192.168.1.30:8082",
		Metadata: map[string]string{
			"type":    "monitoring",
			"version": "1.5.0",
		},
		Health: coordination.HealthHealthy,
	}

	if err := monitorRegistry.Register(ctx, monitorService); err != nil {
		log.Printf("注册监控服务失败: %v", err)
	} else {
		fmt.Printf("✓ 注册监控服务: %s/%s\n", monitorService.Name, monitorService.InstanceID)
	}

	// 7. 等待中断信号
	fmt.Println("\n7. 服务运行中... (按 Ctrl+C 退出)")
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)

	select {
	case <-sigCh:
		fmt.Println("\n收到退出信号，开始清理...")
	case <-time.After(30 * time.Second):
		fmt.Println("\n演示时间结束，开始清理...")
	}

	// 8. 清理注册的服务
	fmt.Println("\n8. 清理服务注册:")
	for _, service := range services {
		if err := registry.Deregister(ctx, service.Name, service.InstanceID); err != nil {
			log.Printf("注销服务失败: %v", err)
		} else {
			fmt.Printf("✓ 注销服务: %s/%s\n", service.Name, service.InstanceID)
		}
	}

	if err := monitorRegistry.Deregister(ctx, monitorService.Name, monitorService.InstanceID); err != nil {
		log.Printf("注销监控服务失败: %v", err)
	} else {
		fmt.Printf("✓ 注销监控服务: %s/%s\n", monitorService.Name, monitorService.InstanceID)
	}

	fmt.Println("\n=== 服务注册与发现示例完成 ===")
}
