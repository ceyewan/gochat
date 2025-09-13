package main

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/ceyewan/gochat/im-infra/clog"
	"github.com/ceyewan/gochat/im-infra/coord"
	"github.com/ceyewan/gochat/im-infra/coord/allocator"
)

func main() {
	// 初始化日志
	config := clog.GetDefaultConfig("development")
	if err := clog.Init(context.Background(), config, clog.WithNamespace("allocator-example")); err != nil {
		log.Fatalf("初始化 clog 失败: %v", err)
	}

	logger := clog.Namespace("main")
	logger.Info("InstanceIDAllocator 示例程序启动")

	// 创建 coord provider
	coordConfig := coord.GetDefaultConfig("development")
	coordConfig.Endpoints = []string{"localhost:2379"} // 根据实际环境调整

	coordProvider, err := coord.New(context.Background(), coordConfig, coord.WithLogger(clog.Namespace("coord")))
	if err != nil {
		logger.Fatal("创建 coord provider 失败", clog.Err(err))
	}
	defer coordProvider.Close()

	// 示例 1: 基本使用
	logger.Info("=== 示例 1: 基本使用 ===")
	err = basicUsageExample(coordProvider)
	if err != nil {
		logger.Error("基本使用示例失败", clog.Err(err))
	}

	// 示例 2: 并发使用
	logger.Info("=== 示例 2: 并发使用 ===")
	err = concurrentUsageExample(coordProvider)
	if err != nil {
		logger.Error("并发使用示例失败", clog.Err(err))
	}

	// 示例 3: ID 重用测试
	logger.Info("=== 示例 3: ID 重用测试 ===")
	err = reuseExample(coordProvider)
	if err != nil {
		logger.Error("ID 重用示例失败", clog.Err(err))
	}

	// 示例 4: 错误处理
	logger.Info("=== 示例 4: 错误处理 ===")
	err = errorHandlingExample(coordProvider)
	if err != nil {
		logger.Error("错误处理示例失败", clog.Err(err))
	}

	logger.Info("所有示例执行完成")
}

// basicUsageExample 演示 InstanceIDAllocator 的基本使用
func basicUsageExample(provider coord.Provider) error {
	logger := clog.Namespace("basic-example")

	// 创建实例 ID 分配器
	// serviceName: "user-service", maxID: 10
	idAllocator, err := provider.InstanceIDAllocator("user-service", 10)
	if err != nil {
		return fmt.Errorf("创建分配器失败: %w", err)
	}

	// 获取一个实例 ID
	instanceID, err := idAllocator.AcquireID(context.Background())
	if err != nil {
		return fmt.Errorf("获取实例 ID 失败: %w", err)
	}

	logger.Info("成功获取实例 ID", clog.Int("id", instanceID.ID()))

	// 使用 ID 执行一些业务逻辑
	// 这里只是模拟，实际应用中可以使用这个 ID 来标识服务实例
	err = performBusinessLogic(instanceID.ID())
	if err != nil {
		logger.Error("业务逻辑执行失败", clog.Err(err))
		// 即使业务逻辑失败，也要释放 ID
		instanceID.Close(context.Background())
		return err
	}

	// 业务逻辑完成，释放 ID
	logger.Info("业务逻辑完成，释放实例 ID", clog.Int("id", instanceID.ID()))
	err = instanceID.Close(context.Background())
	if err != nil {
		return fmt.Errorf("释放实例 ID 失败: %w", err)
	}

	logger.Info("基本使用示例完成")
	return nil
}

// concurrentUsageExample 演示并发使用 InstanceIDAllocator
func concurrentUsageExample(provider coord.Provider) error {
	logger := clog.Namespace("concurrent-example")

	// 创建分配器
	idAllocator, err := provider.InstanceIDAllocator("concurrent-service", 20)
	if err != nil {
		return fmt.Errorf("创建分配器失败: %w", err)
	}

	const numWorkers = 5
	const numIDsPerWorker = 3

	done := make(chan bool, numWorkers)
	results := make(chan int, numWorkers*numIDsPerWorker)

	// 启动多个 worker 并发获取 ID
	for i := 0; i < numWorkers; i++ {
		go func(workerID int) {
			defer func() { done <- true }()

			for j := 0; j < numIDsPerWorker; j++ {
				instanceID, err := idAllocator.AcquireID(context.Background())
				if err != nil {
					logger.Error("Worker 获取 ID 失败", 
						clog.Int("worker_id", workerID),
						clog.Err(err))
					continue
				}

				logger.Info("Worker 获取 ID", 
					clog.Int("worker_id", workerID),
					clog.Int("instance_id", instanceID.ID()))

				// 模拟工作
				time.Sleep(100 * time.Millisecond)

				// 发送结果
				results <- instanceID.ID()

				// 释放 ID
				err = instanceID.Close(context.Background())
				if err != nil {
					logger.Error("Worker 释放 ID 失败", 
						clog.Int("worker_id", workerID),
						clog.Int("instance_id", instanceID.ID()),
						clog.Err(err))
				}
			}
		}(i)
	}

	// 等待所有 worker 完成
	for i := 0; i < numWorkers; i++ {
		<-done
	}
	close(results)

	// 收集结果
	var allocatedIDs []int
	for id := range results {
		allocatedIDs = append(allocatedIDs, id)
	}

	logger.Info("并发示例完成", 
		clog.Int("total_allocated", len(allocatedIDs)),
		clog.Ints("allocated_ids", allocatedIDs))

	return nil
}

// reuseExample 演示 ID 的重用机制
func reuseExample(provider coord.Provider) error {
	logger := clog.Namespace("reuse-example")

	idAllocator, err := provider.InstanceIDAllocator("reuse-service", 5)
	if err != nil {
		return fmt.Errorf("创建分配器失败: %w", err)
	}

	// 连续获取并释放相同的 ID，验证重用机制
	for i := 0; i < 3; i++ {
		instanceID, err := idAllocator.AcquireID(context.Background())
		if err != nil {
			return fmt.Errorf("第 %d 次获取 ID 失败: %w", i+1, err)
		}

		id := instanceID.ID()
		logger.Info("第 %d 次获取 ID", clog.Int("attempt", i+1), clog.Int("id", id))

		// 短暂使用
		time.Sleep(50 * time.Millisecond)

		// 释放
		err = instanceID.Close(context.Background())
		if err != nil {
			return fmt.Errorf("第 %d 次释放 ID 失败: %w", i+1, err)
		}

		// 等待一段时间，让 ID 被回收
		time.Sleep(100 * time.Millisecond)
	}

	logger.Info("ID 重用示例完成")
	return nil
}

// errorHandlingExample 演示错误处理场景
func errorHandlingExample(provider coord.Provider) error {
	logger := clog.Namespace("error-example")

	// 创建一个小容量的分配器来演示 ID 耗尽
	idAllocator, err := provider.InstanceIDAllocator("error-service", 2)
	if err != nil {
		return fmt.Errorf("创建分配器失败: %w", err)
	}

	// 获取所有可用的 ID
	var allocatedIDs []allocator.AllocatedID
	for i := 0; i < 2; i++ {
		instanceID, err := idAllocator.AcquireID(context.Background())
		if err != nil {
			return fmt.Errorf("获取第 %d 个 ID 失败: %w", i+1, err)
		}
		allocatedIDs = append(allocatedIDs, instanceID)
		logger.Info("获取 ID", clog.Int("id", instanceID.ID()))
	}

	// 尝试获取第 3 个 ID，应该失败
	_, err = idAllocator.AcquireID(context.Background())
	if err != nil {
		logger.Info("预期中的错误：无法获取更多 ID", clog.Err(err))
	} else {
		logger.Error("意外：应该无法获取更多 ID")
	}

	// 释放一个 ID
	logger.Info("释放第一个 ID")
	err = allocatedIDs[0].Close(context.Background())
	if err != nil {
		return fmt.Errorf("释放 ID 失败: %w", err)
	}

	// 现在应该能够获取新的 ID
	time.Sleep(50 * time.Millisecond) // 给 etcd 一些时间处理
	newInstanceID, err := idAllocator.AcquireID(context.Background())
	if err != nil {
		return fmt.Errorf("释放后重新获取 ID 失败: %w", err)
	}

	logger.Info("成功重新获取 ID", clog.Int("id", newInstanceID.ID()))

	// 释放所有剩余的 ID
	err = allocatedIDs[1].Close(context.Background())
	if err != nil {
		return fmt.Errorf("释放 ID 失败: %w", err)
	}
	err = newInstanceID.Close(context.Background())
	if err != nil {
		return fmt.Errorf("释放 ID 失败: %w", err)
	}

	logger.Info("错误处理示例完成")
	return nil
}

// performBusinessLogic 模拟业务逻辑
func performBusinessLogic(instanceID int) error {
	logger := clog.Namespace("business-logic")
	
	logger.Info("开始执行业务逻辑", clog.Int("instance_id", instanceID))
	
	// 模拟一些业务操作
	time.Sleep(200 * time.Millisecond)
	
	// 模拟 90% 的成功率
	if time.Now().UnixNano()%10 == 0 {
		return fmt.Errorf("模拟业务逻辑失败")
	}
	
	logger.Info("业务逻辑执行成功", clog.Int("instance_id", instanceID))
	return nil
}