package main

import (
	"context"
	"fmt"
	"log"
	"sync"
	"sync/atomic"
	"time"

	"github.com/ceyewan/gochat/im-infra/coord"
	"github.com/ceyewan/gochat/im-infra/coord/config"
	"github.com/ceyewan/gochat/im-infra/coord/registry"
)

// 高级测试：错误处理、边界条件、性能测试

func main() {
	fmt.Println("=== Coord 模块高级测试示例 ===")
	fmt.Println("测试内容：错误处理、边界条件、性能验证、降级策略")

	// 创建协调器
	coordinator, err := coord.New(context.Background(), coord.DefaultConfig())
	if err != nil {
		log.Fatalf("创建协调器失败: %v", err)
	}
	defer coordinator.Close()

	ctx := context.Background()

	// 测试1: 错误处理和边界条件
	fmt.Println("\n--- 测试1: 错误处理和边界条件 ---")
	testErrorHandling(ctx, coordinator)

	// 测试2: 降级策略验证
	fmt.Println("\n--- 测试2: 降级策略验证 ---")
	testDegradationStrategy(ctx, coordinator)

	// 测试3: 性能基准测试
	fmt.Println("\n--- 测试3: 性能基准测试 ---")
	testPerformanceBenchmark(ctx, coordinator)

	// 测试4: 高并发压力测试
	fmt.Println("\n--- 测试4: 高并发压力测试 ---")
	testHighConcurrency(ctx, coordinator)

	// 测试5: 长时间运行稳定性
	fmt.Println("\n--- 测试5: 长时间运行稳定性 ---")
	testLongRunningStability(ctx, coordinator)

	fmt.Println("\n=== 高级测试完成 ===")
}

func testErrorHandling(ctx context.Context, coordinator coord.Provider) {
	fmt.Println("测试各种错误场景...")

	// 测试1: 空参数处理
	fmt.Println("1. 测试空参数处理...")
	_, err := coordinator.Lock().Acquire(ctx, "", 10*time.Second)
	if err != nil {
		fmt.Println("✓ 空锁键参数正确处理")
	} else {
		fmt.Println("✗ 空锁键参数未正确处理")
	}

	_, err = coordinator.Lock().TryAcquire(ctx, "valid-key", 0)
	if err != nil {
		fmt.Println("✓ 零TTL参数正确处理")
	} else {
		fmt.Println("✗ 零TTL参数未正确处理")
	}

	// 测试2: 无效服务信息
	fmt.Println("2. 测试无效服务信息...")
	invalidService := registry.ServiceInfo{
		ID:      "", // 空ID
		Name:    "test-service",
		Address: "127.0.0.1",
		Port:    8080,
	}
	err = coordinator.Registry().Register(ctx, invalidService, 30*time.Second)
	if err != nil {
		fmt.Println("✓ 空服务ID正确处理")
	} else {
		fmt.Println("✗ 空服务ID未正确处理")
	}

	// 测试3: 不存在的键操作
	fmt.Println("3. 测试不存在的键操作...")
	var result string
	err = coordinator.Config().Get(ctx, "non-existent-key-12345", &result)
	if err != nil {
		fmt.Println("✓ 不存在的配置键正确处理")
	} else {
		fmt.Println("✗ 不存在的配置键未正确处理")
	}

	// 测试4: 无效的CAS操作
	fmt.Println("4. 测试无效的CAS操作...")
	err = coordinator.Config().CompareAndSet(ctx, "test-cas-key", "new-value", 999999)
	if err != nil {
		fmt.Println("✓ 版本冲突的CAS操作正确处理")
	} else {
		fmt.Println("✗ 版本冲突的CAS操作未正确处理")
	}

	// 测试5: 上下文取消
	fmt.Println("5. 测试上下文取消...")
	cancelCtx, cancel := context.WithTimeout(ctx, 100*time.Millisecond)

	// 先获取锁，确保阻塞获取会被取消
	lock, err := coordinator.Lock().Acquire(ctx, "cancel-test-lock", 30*time.Second)
	if err == nil {
		defer lock.Unlock(ctx)

		// 尝试在取消的上下文中获取同一个锁
		start := time.Now()
		_, err = coordinator.Lock().Acquire(cancelCtx, "cancel-test-lock", 30*time.Second)
		duration := time.Since(start)

		if err != nil && duration < 200*time.Millisecond {
			fmt.Println("✓ 上下文取消正确处理")
		} else {
			fmt.Println("✗ 上下文取消未正确处理")
		}
	}
	cancel()
}

func testDegradationStrategy(ctx context.Context, coordinator coord.Provider) {
	fmt.Println("测试降级策略...")

	// 测试1: 配置管理器降级（无配置中心）
	fmt.Println("1. 测试配置管理器降级...")

	type TestConfig struct {
		Name  string `json:"name"`
		Value int    `json:"value"`
	}

	defaultConfig := TestConfig{Name: "default", Value: 100}

	// 创建没有配置中心的管理器
	manager := config.NewManager(
		nil, // 无配置中心
		"test", "degradation", "component",
		defaultConfig,
	)
	manager.Start()
	defer manager.Stop()

	// 应该使用默认配置
	currentConfig := manager.GetCurrentConfig()
	if currentConfig.Name == "default" && currentConfig.Value == 100 {
		fmt.Println("✓ 配置管理器降级成功")
	} else {
		fmt.Println("✗ 配置管理器降级失败")
	}

	// 测试2: 服务发现降级
	fmt.Println("2. 测试服务发现降级...")

	// 发现不存在的服务（应该返回空列表而不是错误）
	services, err := coordinator.Registry().Discover(ctx, "non-existent-service-xyz")
	if err == nil && len(services) == 0 {
		fmt.Println("✓ 服务发现降级成功")
	} else {
		fmt.Printf("✗ 服务发现降级失败: %v\n", err)
	}

	// 测试3: 重试机制
	fmt.Println("3. 测试重试机制...")

	// 使用无效的etcd地址测试重试
	invalidConfig := coord.CoordinatorConfig{
		Endpoints: []string{"localhost:9999"}, // 无效地址
		Timeout:   1 * time.Second,
		RetryConfig: &coord.RetryConfig{
			MaxAttempts:  2,
			InitialDelay: 100 * time.Millisecond,
			MaxDelay:     500 * time.Millisecond,
			Multiplier:   2.0,
		},
	}

	start := time.Now()
	_, err = coord.New(context.Background(), invalidConfig)
	duration := time.Since(start)

	if err != nil && duration >= 200*time.Millisecond {
		fmt.Println("✓ 重试机制工作正常")
	} else {
		fmt.Printf("✗ 重试机制异常: %v, duration: %v\n", err, duration)
	}
}

func testPerformanceBenchmark(ctx context.Context, coordinator coord.Provider) {
	fmt.Println("性能基准测试...")

	// 测试1: 锁操作性能
	fmt.Println("1. 测试锁操作性能...")

	const iterations = 100
	lockKey := "perf-test-lock"
	ttl := 10 * time.Second

	start := time.Now()
	successCount := 0

	for i := 0; i < iterations; i++ {
		lock, err := coordinator.Lock().TryAcquire(ctx, lockKey+fmt.Sprintf("-%d", i), ttl)
		if err == nil {
			lock.Unlock(ctx)
			successCount++
		}
	}

	duration := time.Since(start)
	avgDuration := duration / time.Duration(iterations)

	fmt.Printf("✓ 锁操作: %d 次, 总时间: %v, 平均: %v, 成功率: %.1f%%\n",
		iterations, duration, avgDuration, float64(successCount)/float64(iterations)*100)

	// 测试2: 配置操作性能
	fmt.Println("2. 测试配置操作性能...")

	configKey := "perf-test-config"
	testValue := map[string]interface{}{"data": "test-value", "timestamp": time.Now().Unix()}

	start = time.Now()
	for i := 0; i < iterations; i++ {
		key := configKey + fmt.Sprintf("-%d", i)
		coordinator.Config().Set(ctx, key, testValue)

		var result map[string]interface{}
		coordinator.Config().Get(ctx, key, &result)

		coordinator.Config().Delete(ctx, key)
	}

	duration = time.Since(start)
	avgDuration = duration / time.Duration(iterations)

	fmt.Printf("✓ 配置操作: %d 次, 总时间: %v, 平均: %v, TPS: %.0f\n",
		iterations, duration, avgDuration, float64(iterations)/duration.Seconds())

	// 测试3: 服务发现性能
	fmt.Println("3. 测试服务发现性能...")

	// 先注册一些服务
	for i := 0; i < 10; i++ {
		service := registry.ServiceInfo{
			ID:      fmt.Sprintf("perf-service-%d", i),
			Name:    "perf-test-service",
			Address: "127.0.0.1",
			Port:    9000 + i,
		}
		coordinator.Registry().Register(ctx, service, 60*time.Second)
		defer coordinator.Registry().Unregister(ctx, service.ID)
	}

	start = time.Now()
	for i := 0; i < iterations; i++ {
		services, _ := coordinator.Registry().Discover(ctx, "perf-test-service")
		_ = len(services) // 使用结果避免编译器优化
	}

	duration = time.Since(start)
	avgDuration = duration / time.Duration(iterations)

	fmt.Printf("✓ 服务发现: %d 次, 总时间: %v, 平均: %v\n",
		iterations, duration, avgDuration)
}

func testHighConcurrency(ctx context.Context, coordinator coord.Provider) {
	fmt.Println("高并发压力测试...")

	const (
		numWorkers          = 50
		operationsPerWorker = 20
	)

	var (
		successCount int64
		errorCount   int64
		wg           sync.WaitGroup
		// operationCount int64
		// errorCount2    int64
	)

	// 启动多个worker进行并发操作
	fmt.Printf("启动 %d 个worker，每个执行 %d 次操作...\n", numWorkers, operationsPerWorker)

	start := time.Now()

	for i := 0; i < numWorkers; i++ {
		wg.Add(1)
		go func(workerID int) {
			defer wg.Done()

			for j := 0; j < operationsPerWorker; j++ {
				// 随机选择操作类型
				opType := j % 3

				switch opType {
				case 0: // 锁操作
					lockKey := fmt.Sprintf("concurrent-lock-%d-%d", workerID, j%5)
					lock, err := coordinator.Lock().TryAcquire(ctx, lockKey, 5*time.Second)
					if err == nil {
						atomic.AddInt64(&successCount, 1)
						time.Sleep(10 * time.Millisecond) // 模拟工作
						lock.Unlock(ctx)
					} else {
						atomic.AddInt64(&errorCount, 1)
					}

				case 1: // 配置操作
					configKey := fmt.Sprintf("concurrent-config-%d-%d", workerID, j%10)
					value := map[string]interface{}{"worker": workerID, "op": j}

					err := coordinator.Config().Set(ctx, configKey, value)
					if err == nil {
						atomic.AddInt64(&successCount, 1)

						var result map[string]interface{}
						coordinator.Config().Get(ctx, configKey, &result)
						coordinator.Config().Delete(ctx, configKey)
					} else {
						atomic.AddInt64(&errorCount, 1)
					}

				case 2: // 服务注册发现
					serviceName := fmt.Sprintf("concurrent-service-%d", j%5)
					service := registry.ServiceInfo{
						ID:      fmt.Sprintf("worker-%d-service-%d", workerID, j),
						Name:    serviceName,
						Address: "127.0.0.1",
						Port:    8000 + workerID,
					}

					err := coordinator.Registry().Register(ctx, service, 10*time.Second)
					if err == nil {
						atomic.AddInt64(&successCount, 1)

						services, _ := coordinator.Registry().Discover(ctx, serviceName)
						_ = len(services) // 使用结果

						coordinator.Registry().Unregister(ctx, service.ID)
					} else {
						atomic.AddInt64(&errorCount, 1)
					}
				}
			}
		}(i)
	}

	wg.Wait()
	duration := time.Since(start)
	totalOperations := int64(numWorkers * operationsPerWorker)

	fmt.Printf("✓ 并发测试完成\n")
	fmt.Printf("  总操作数: %d\n", totalOperations)
	fmt.Printf("  成功数: %d (%.1f%%)\n", successCount, float64(successCount)/float64(totalOperations)*100)
	fmt.Printf("  失败数: %d (%.1f%%)\n", errorCount, float64(errorCount)/float64(totalOperations)*100)
	fmt.Printf("  总时间: %v\n", duration)
	fmt.Printf("  平均延迟: %v\n", duration/time.Duration(totalOperations))
	fmt.Printf("  TPS: %.0f\n", float64(totalOperations)/duration.Seconds())
}

func testLongRunningStability(ctx context.Context, coordinator coord.Provider) {
	fmt.Println("长时间运行稳定性测试...")

	const testDuration = 5 * time.Second

	fmt.Printf("运行 %v 稳定性测试...\n", testDuration)

	start := time.Now()
	operationCount := int64(0)
	errorCount := int64(0)

	// 创建多个goroutine进行不同类型的操作
	ctx, cancel := context.WithTimeout(ctx, testDuration)
	defer cancel()

	var wg sync.WaitGroup

	// 锁操作goroutineßß
	wg.Add(1)
	go func() {
		defer wg.Done()
		for {
			select {
			case <-ctx.Done():
				return
			default:
				lockKey := fmt.Sprintf("stability-lock-%d", time.Now().UnixNano()%10)
				lock, err := coordinator.Lock().TryAcquire(ctx, lockKey, 5*time.Second)
				if err == nil {
					atomic.AddInt64(&operationCount, 1)
					time.Sleep(50 * time.Millisecond)
					lock.Unlock(ctx)
				} else {
					atomic.AddInt64(&errorCount, 1)
				}
			}
		}
	}()

	// 配置操作goroutine
	wg.Add(1)
	go func() {
		defer wg.Done()
		for {
			select {
			case <-ctx.Done():
				return
			default:
				configKey := fmt.Sprintf("stability-config-%d", time.Now().UnixNano()%20)
				value := map[string]interface{}{"timestamp": time.Now().Unix()}

				err := coordinator.Config().Set(ctx, configKey, value)
				if err == nil {
					atomic.AddInt64(&operationCount, 1)

					var result map[string]interface{}
					coordinator.Config().Get(ctx, configKey, &result)
					coordinator.Config().Delete(ctx, configKey)
				} else {
					atomic.AddInt64(&errorCount, 1)
				}
			}
		}
	}()

	// 服务注册发现goroutine
	wg.Add(1)
	go func() {
		defer wg.Done()
		for {
			select {
			case <-ctx.Done():
				return
			default:
				serviceName := fmt.Sprintf("stability-service-%d", time.Now().UnixNano()%5)
				service := registry.ServiceInfo{
					ID:      fmt.Sprintf("stability-%d", time.Now().UnixNano()),
					Name:    serviceName,
					Address: "127.0.0.1",
					Port:    int(time.Now().UnixNano()%1000 + 9000),
				}

				err := coordinator.Registry().Register(ctx, service, 10*time.Second)
				if err == nil {
					atomic.AddInt64(&operationCount, 1)

					services, _ := coordinator.Registry().Discover(ctx, serviceName)
					_ = len(services)

					coordinator.Registry().Unregister(ctx, service.ID)
				} else {
					atomic.AddInt64(&errorCount, 1)
				}
			}
		}
	}()

	// 等待测试完成
	wg.Wait()
	duration := time.Since(start)
	finalOpCount := atomic.LoadInt64(&operationCount)
	finalErrCount := atomic.LoadInt64(&errorCount)

	successRate := float64(finalOpCount) / float64(finalOpCount+finalErrCount) * 100

	fmt.Printf("✓ 稳定性测试完成\n")
	fmt.Printf("  运行时间: %v\n", duration)
	fmt.Printf("  总操作数: %d\n", finalOpCount+finalErrCount)
	fmt.Printf("  成功数: %d\n", finalOpCount)
	fmt.Printf("  失败数: %d\n", finalErrCount)
	fmt.Printf("  成功率: %.1f%%\n", successRate)
	fmt.Printf("  平均TPS: %.0f\n", float64(finalOpCount+finalErrCount)/duration.Seconds())

	if successRate > 95 {
		fmt.Println("✓ 稳定性测试通过")
	} else {
		fmt.Printf("✗ 稳定性测试失败，成功率 %.1f%% 低于95%%\n", successRate)
	}
}
