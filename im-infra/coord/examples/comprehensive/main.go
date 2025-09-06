package main

import (
	"context"
	"fmt"
	"log"
	"sync"
	"time"

	"github.com/ceyewan/gochat/im-infra/coord"
	"github.com/ceyewan/gochat/im-infra/coord/config"
	"github.com/ceyewan/gochat/im-infra/coord/registry"
)

// AppConfig 应用配置示例
type AppConfig struct {
	AppName     string `json:"app_name"`
	Version     string `json:"version"`
	MaxConns    int    `json:"max_conns"`
	EnableDebug bool   `json:"enable_debug"`
}

// appConfigValidator 配置验证器
type appConfigValidator struct{}

func (v *appConfigValidator) Validate(cfg *AppConfig) error {
	if cfg.AppName == "" {
		return fmt.Errorf("app_name cannot be empty")
	}
	if cfg.Version == "" {
		return fmt.Errorf("version cannot be empty")
	}
	if cfg.MaxConns <= 0 {
		return fmt.Errorf("max_conns must be positive")
	}
	return nil
}

// appConfigUpdater 配置更新器
type appConfigUpdater struct {
	updateCount int
	mu          sync.Mutex
}

func (u *appConfigUpdater) OnConfigUpdate(oldConfig, newConfig *AppConfig) error {
	u.mu.Lock()
	defer u.mu.Unlock()
	u.updateCount++
	log.Printf("配置更新 #%d: %s -> %s", u.updateCount, oldConfig.Version, newConfig.Version)
	return nil
}

func (u *appConfigUpdater) GetUpdateCount() int {
	u.mu.Lock()
	defer u.mu.Unlock()
	return u.updateCount
}

func main() {
	fmt.Println("=== Coord 模块综合测试示例 ===")

	// 创建协调器
	coordinator, err := coord.New(context.Background(), coord.DefaultConfig())
	if err != nil {
		log.Fatalf("创建协调器失败: %v", err)
	}
	defer coordinator.Close()

	ctx := context.Background()

	// 测试1: 分布式锁
	fmt.Println("\n--- 测试1: 分布式锁 ---")
	testDistributedLock(ctx, coordinator)

	// 测试2: 配置中心
	fmt.Println("\n--- 测试2: 配置中心 ---")
	testConfigCenter(ctx, coordinator)

	// 测试3: 服务注册发现
	fmt.Println("\n--- 测试3: 服务注册发现 ---")
	testServiceRegistry(ctx, coordinator)

	// 测试4: 通用配置管理器
	fmt.Println("\n--- 测试4: 通用配置管理器 ---")
	testConfigManager(ctx, coordinator)

	// 测试5: 并发测试
	fmt.Println("\n--- 测试5: 并发测试 ---")
	testConcurrency(ctx, coordinator)

	fmt.Println("\n=== 所有测试完成 ===")
}

func testDistributedLock(ctx context.Context, coordinator coord.Provider) {
	lockKey := "test-resource-lock"
	ttl := 15 * time.Second

	// 测试阻塞获取
	fmt.Println("获取分布式锁 (阻塞模式)...")
	lock1, err := coordinator.Lock().Acquire(ctx, lockKey, ttl)
	if err != nil {
		log.Printf("获取锁失败: %v", err)
		return
	}
	fmt.Printf("✓ 锁获取成功: %s\n", lock1.Key())

	// 测试非阻塞获取 (应该失败)
	fmt.Println("尝试非阻塞获取锁...")
	_, err = coordinator.Lock().TryAcquire(ctx, lockKey, ttl)
	if err != nil {
		fmt.Println("✓ 非阻塞获取失败 (符合预期)")
	} else {
		fmt.Println("✗ 非阻塞获取成功 (不符合预期)")
	}

	// 测试TTL
	ttlRemaining, err := lock1.TTL(ctx)
	if err != nil {
		log.Printf("获取TTL失败: %v", err)
	} else {
		fmt.Printf("✓ 锁剩余TTL: %v\n", ttlRemaining)
	}

	// 释放锁
	if err := lock1.Unlock(ctx); err != nil {
		log.Printf("释放锁失败: %v", err)
	} else {
		fmt.Println("✓ 锁释放成功")
	}

	// 验证锁已释放，可以重新获取
	fmt.Println("验证锁已释放...")
	lock2, err := coordinator.Lock().TryAcquire(ctx, lockKey, ttl)
	if err != nil {
		log.Printf("重新获取锁失败: %v", err)
	} else {
		fmt.Println("✓ 锁已释放，可以重新获取")
		lock2.Unlock(ctx)
	}
}

func testConfigCenter(ctx context.Context, coordinator coord.Provider) {
	configKey := "test/app/config"

	// 测试基本配置操作
	fmt.Println("测试配置基本操作...")

	// 设置配置
	configValue := map[string]interface{}{
		"version": "1.0.0",
		"debug":   true,
		"port":    8080,
	}

	if err := coordinator.Config().Set(ctx, configKey, configValue); err != nil {
		log.Printf("设置配置失败: %v", err)
		return
	}
	fmt.Println("✓ 配置设置成功")

	// 获取配置
	var retrievedConfig map[string]interface{}
	if err := coordinator.Config().Get(ctx, configKey, &retrievedConfig); err != nil {
		log.Printf("获取配置失败: %v", err)
		return
	}
	fmt.Printf("✓ 配置获取成功: %+v\n", retrievedConfig)

	// 测试CAS操作
	fmt.Println("测试CAS操作...")
	version, err := coordinator.Config().GetWithVersion(ctx, configKey, &retrievedConfig)
	if err != nil {
		log.Printf("获取配置版本失败: %v", err)
		return
	}

	newConfig := map[string]interface{}{
		"version": "1.1.0",
		"debug":   false,
		"port":    9090,
	}

	if err := coordinator.Config().CompareAndSet(ctx, configKey, newConfig, version); err != nil {
		log.Printf("CAS操作失败: %v", err)
	} else {
		fmt.Println("✓ CAS操作成功")
	}

	// 清理测试数据
	if err := coordinator.Config().Delete(ctx, configKey); err != nil {
		log.Printf("删除配置失败: %v", err)
	} else {
		fmt.Println("✓ 配置删除成功")
	}
}

func testServiceRegistry(ctx context.Context, coordinator coord.Provider) {
	serviceName := "test-service"

	// 注册多个服务实例
	fmt.Println("注册服务实例...")
	services := []registry.ServiceInfo{
		{
			ID:       "instance-1",
			Name:     serviceName,
			Address:  "127.0.0.1",
			Port:     8081,
			Metadata: map[string]string{"version": "1.0.0", "region": "us-east"},
		},
		{
			ID:       "instance-2",
			Name:     serviceName,
			Address:  "127.0.0.1",
			Port:     8082,
			Metadata: map[string]string{"version": "1.0.1", "region": "us-west"},
		},
	}

	for _, service := range services {
		if err := coordinator.Registry().Register(ctx, service, 30*time.Second); err != nil {
			log.Printf("注册服务 %s 失败: %v", service.ID, err)
			continue
		}
		fmt.Printf("✓ 服务 %s 注册成功\n", service.ID)
	}

	// 发现服务
	fmt.Println("发现服务实例...")
	discoveredServices, err := coordinator.Registry().Discover(ctx, serviceName)
	if err != nil {
		log.Printf("发现服务失败: %v", err)
		return
	}

	fmt.Printf("✓ 发现 %d 个服务实例:\n", len(discoveredServices))
	for _, svc := range discoveredServices {
		fmt.Printf("  - ID: %s, Address: %s:%d, Version: %s\n",
			svc.ID, svc.Address, svc.Port, svc.Metadata["version"])
	}

	// 测试服务监听
	fmt.Println("测试服务变更监听...")
	watchCtx, cancel := context.WithCancel(ctx)
	defer cancel() // 确保函数退出时取消监听

	eventCh, err := coordinator.Registry().Watch(watchCtx, serviceName)
	if err != nil {
		log.Printf("创建服务监听器失败: %v", err)
		return
	}

	var wgWatch sync.WaitGroup
	wgWatch.Add(1)
	// 启动监听 goroutine
	go func() {
		defer wgWatch.Done()
		for event := range eventCh {
			fmt.Printf("✓ 服务事件: Type=%s, Service=%s, ID=%s\n",
				event.Type, event.Service.Name, event.Service.ID)
		}
	}()

	// 注销一个服务实例来触发事件
	time.Sleep(100 * time.Millisecond) // 等待监听器启动
	fmt.Println("注销一个服务实例...")
	if err := coordinator.Registry().Unregister(ctx, "instance-1"); err != nil {
		log.Printf("注销服务失败: %v", err)
	} else {
		fmt.Println("✓ 服务 instance-1 注销成功")
	}

	time.Sleep(200 * time.Millisecond) // 等待事件处理

	// 停止监听并等待 goroutine 退出
	cancel()
	wgWatch.Wait()

	// 清理剩余服务
	for _, service := range services {
		if service.ID != "instance-1" { // 已经注销的跳过
			coordinator.Registry().Unregister(ctx, service.ID)
		}
	}
}

func testConfigManager(ctx context.Context, coordinator coord.Provider) {
	fmt.Println("创建通用配置管理器...")

	// 默认配置
	defaultConfig := AppConfig{
		AppName:     "test-app",
		Version:     "1.0.0",
		MaxConns:    100,
		EnableDebug: false,
	}

	// 创建验证器和更新器
	validator := &appConfigValidator{}
	updater := &appConfigUpdater{}

	// 添加超时控制，避免无限等待
	managerCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()

	// 创建配置管理器，但先不添加 watcher
	fmt.Println("创建配置管理器（无watcher）...")
	manager := config.NewManager(
		coordinator.Config(),
		"test", "comprehensive", "app",
		defaultConfig,
		config.WithValidator[AppConfig](validator),
		config.WithUpdater[AppConfig](updater),
	)
	fmt.Println("✓ 配置管理器创建成功")

	// 获取当前配置（在Start之前也可以获取）
	fmt.Println("获取默认配置...")
	currentConfig := manager.GetCurrentConfig()
	fmt.Printf("✓ 当前配置: %+v\n", *currentConfig)

	// 测试基本的配置中心操作
	fmt.Println("测试基本配置操作...")
	configKey := "test/comprehensive/app"

	testConfig := AppConfig{
		AppName:     "test-app",
		Version:     "2.0.0",
		MaxConns:    200,
		EnableDebug: true,
	}

	// 直接使用 configCenter 设置配置
	if err := coordinator.Config().Set(managerCtx, configKey, testConfig); err != nil {
		log.Printf("设置配置失败: %v", err)
		return
	}
	fmt.Println("✓ 配置设置成功")

	// 手动重载配置（这会从配置中心加载）
	fmt.Println("手动重载配置...")
	manager.ReloadConfig()

	reloadedConfig := manager.GetCurrentConfig()
	fmt.Printf("✓ 重载后配置: %+v\n", *reloadedConfig)

	// 验证配置是否正确加载
	if reloadedConfig.Version == "2.0.0" {
		fmt.Println("✓ 配置重载成功")
	} else {
		fmt.Printf("✗ 配置重载失败，期望版本 2.0.0，实际 %s\n", reloadedConfig.Version)
	}

	// 清理测试数据
	fmt.Println("清理测试数据...")
	coordinator.Config().Delete(managerCtx, configKey)
	fmt.Println("✓ 配置管理器测试完成（简化版）")
}

func testConcurrency(ctx context.Context, coordinator coord.Provider) {
	fmt.Println("并发测试 - 多个goroutine同时获取锁...")

	const (
		numWorkers = 5
		lockKey    = "concurrent-test-lock"
		ttl        = 10 * time.Second
	)

	var wg sync.WaitGroup
	results := make(chan string, numWorkers)

	// 启动多个worker
	for i := 0; i < numWorkers; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()

			workerName := fmt.Sprintf("worker-%d", id)
			startTime := time.Now()

			// 尝试获取锁
			lock, err := coordinator.Lock().TryAcquire(ctx, lockKey, ttl)
			if err != nil {
				results <- fmt.Sprintf("%s: 获取锁失败 (%.2fs)", workerName, time.Since(startTime).Seconds())
				return
			}

			// 成功获取锁
			acquireTime := time.Since(startTime)
			results <- fmt.Sprintf("%s: 获取锁成功 (%.2fs)", workerName, acquireTime.Seconds())

			// 模拟工作
			time.Sleep(500 * time.Millisecond)

			// 释放锁
			lock.Unlock(ctx)
		}(i)
	}

	// 收集结果
	go func() {
		wg.Wait()
		close(results)
	}()

	// 显示结果
	successCount := 0
	for result := range results {
		fmt.Printf("  %s\n", result)
		if contains(result, "成功") {
			successCount++
		}
	}

	fmt.Printf("✓ 并发测试结果: %d/%d 成功获取锁\n", successCount, numWorkers)

	// 验证只有一个成功
	if successCount == 1 {
		fmt.Println("✓ 并发安全性验证通过")
	} else {
		fmt.Printf("✗ 并发安全性验证失败，期望1个成功，实际%d个\n", successCount)
	}
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && s[len(s)-len(substr):] == substr ||
		len(s) >= len(substr) && s[:len(substr)] == substr ||
		len(s) > len(substr) && containsSubstring(s, substr)
}

func containsSubstring(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
