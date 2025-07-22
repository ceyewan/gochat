package main

import (
	"context"
	"time"

	"github.com/ceyewan/gochat/im-infra/clog"
	"github.com/ceyewan/gochat/im-infra/etcd"
)

func main() {
	// 创建日志器
	logger := clog.Default().With("module", "etcd-example")
	logger.Info("开始 etcd 示例程序")

	// 示例1: 使用基本客户端
	basicClientExample()

	// 示例2: 使用管理器
	managerExample()

	// 示例3: 服务注册和发现
	serviceRegistryExample()

	logger.Info("etcd 示例程序结束")
}

// basicClientExample 基本客户端示例
func basicClientExample() {
	etcdLogger := clog.Default().With("module", "etcd", "example", "basic-client")
	etcdLogger.Info("=== 基本客户端示例 ===")

	// 创建配置
	config := &etcd.Config{
		Endpoints:   []string{"localhost:23791"},
		DialTimeout: 5 * time.Second,
	}

	// 创建客户端
	client, err := etcd.NewClient(config)
	if err != nil {
		etcdLogger.Error("创建客户端失败", "error", err)
		return
	}
	defer client.Close()

	etcdLogger.Info("基本客户端创建成功")
}

// managerExample 管理器示例
func managerExample() {
	etcdLogger := clog.Default().With("module", "etcd", "example", "manager")
	etcdLogger.Info("=== 管理器示例 ===")

	// 创建管理器选项
	options := &etcd.ManagerOptions{
		Endpoints:           []string{"localhost:23791"},
		DialTimeout:         5 * time.Second,
		Logger:              etcd.NewClogAdapter(etcdLogger),
		DefaultTTL:          30,
		ServicePrefix:       "/services",
		LockPrefix:          "/locks",
		HealthCheckInterval: 30 * time.Second,
		HealthCheckTimeout:  5 * time.Second,
	}

	// 创建管理器
	manager, err := etcd.NewEtcdManager(options)
	if err != nil {
		etcdLogger.Error("创建管理器失败", "error", err)
		return
	}
	defer manager.Close()

	etcdLogger.Info("管理器创建成功")

	// 检查健康状态
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := manager.HealthCheck(ctx); err != nil {
		etcdLogger.Error("健康检查失败", "error", err)
	} else {
		etcdLogger.Info("健康检查通过")
	}
}

// serviceRegistryExample 服务注册和发现示例
func serviceRegistryExample() {
	etcdLogger := clog.Default().With("module", "etcd", "example", "service-registry")
	etcdLogger.Info("=== 服务注册和发现示例 ===")

	// 创建管理器
	options := etcd.DefaultManagerOptions()
	options.Endpoints = []string{"localhost:2379"}

	manager, err := etcd.NewEtcdManager(options)
	if err != nil {
		etcdLogger.Error("创建管理器失败", "error", err)
		return
	}
	defer manager.Close()

	// 获取服务注册组件
	registry := manager.ServiceRegistry()
	discovery := manager.ServiceDiscovery()

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// 注册服务
	serviceName := "example-service"
	instanceID := "instance-1"
	address := "localhost:8080"

	etcdLogger.Info("注册服务",
		"service", serviceName,
		"instance", instanceID,
		"address", address)

	err = registry.Register(ctx, serviceName, instanceID, address,
		etcd.WithTTL(60),
		etcd.WithMetadata(map[string]string{
			"version": "1.0.0",
			"region":  "us-west-1",
		}))
	if err != nil {
		etcdLogger.Error("注册服务失败", "error", err)
		return
	}

	etcdLogger.Info("服务注册成功")

	// 发现服务
	etcdLogger.Info("发现服务", "service", serviceName)
	endpoints, err := discovery.GetServiceEndpoints(ctx, serviceName)
	if err != nil {
		etcdLogger.Error("发现服务失败", "error", err)
		return
	}

	etcdLogger.Info("服务发现成功",
		"service", serviceName,
		"endpoints", endpoints)

	// 获取服务实例详情
	instances, err := discovery.ResolveService(ctx, serviceName)
	if err != nil {
		etcdLogger.Error("解析服务失败", "error", err)
		return
	}

	for i, instance := range instances {
		etcdLogger.Info("服务实例",
			"index", i,
			"id", instance.ID,
			"address", instance.Address,
			"metadata", instance.Metadata)
	}

	// 监听服务变化
	etcdLogger.Info("开始监听服务变化")
	eventCh, err := discovery.WatchService(ctx, serviceName)
	if err != nil {
		etcdLogger.Error("监听服务失败", "error", err)
		return
	}

	// 启动一个协程来处理事件
	go func() {
		for event := range eventCh {
			etcdLogger.Info("收到服务事件",
				"type", int(event.Type),
				"service", event.Service,
				"instance_id", event.Instance.ID,
				"address", event.Instance.Address)
		}
	}()

	// 等待一段时间观察事件
	time.Sleep(2 * time.Second)

	// 注销服务
	etcdLogger.Info("注销服务", "service", serviceName, "instance", instanceID)
	err = registry.Deregister(ctx, serviceName, instanceID)
	if err != nil {
		etcdLogger.Error("注销服务失败", "error", err)
		return
	}

	etcdLogger.Info("服务注销成功")

	// 再等待一段时间观察注销事件
	time.Sleep(1 * time.Second)
}
