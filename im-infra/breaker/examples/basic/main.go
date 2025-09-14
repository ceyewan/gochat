package main

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/ceyewan/gochat/im-infra/breaker"
	"github.com/ceyewan/gochat/im-infra/clog"
)

// mockService 模拟一个可能失败的外部服务
type mockService struct {
	failCount int
}

func (s *mockService) call() error {
	s.failCount++
	if s.failCount%3 == 0 { // 每3次调用失败1次
		return fmt.Errorf("service temporarily unavailable")
	}
	return nil
}

func main() {
	// 初始化日志器
	logger := clog.Namespace("breaker-example")

	// 创建 breaker 配置
	config := breaker.GetDefaultConfig("example-service", "development")

	// 创建 breaker Provider
	breakerProvider, err := breaker.New(context.Background(), config,
		breaker.WithLogger(logger),
		// breaker.WithCoordProvider(coordProvider) // 在实际环境中传入配置中心
	)
	if err != nil {
		log.Fatal(err)
	}
	defer breakerProvider.Close()

	// 获取熔断器
	serviceBreaker := breakerProvider.GetBreaker("mock-service")

	// 模拟服务
	service := &mockService{}

	// 执行多次调用，观察熔断器行为
	for i := 1; i <= 10; i++ {
		fmt.Printf("Attempt %d: ", i)

		err := serviceBreaker.Do(context.Background(), func() error {
			return service.call()
		})

		if err != nil {
			if err == breaker.ErrBreakerOpen {
				fmt.Println("❌ Circuit breaker OPEN - call blocked")
			} else {
				fmt.Printf("❌ Call failed: %v\n", err)
			}
		} else {
			fmt.Println("✅ Call succeeded")
		}

		time.Sleep(500 * time.Millisecond)
	}

	fmt.Println("\nExample completed. Observe how the circuit breaker protects against repeated failures.")
}