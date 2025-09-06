package main

import (
	"context"
	"fmt"
	"os"
	"sync"
	"time"

	"github.com/ceyewan/gochat/im-infra/clog"
	"github.com/ceyewan/gochat/im-infra/coord"
	"github.com/ceyewan/gochat/im-infra/coord/registry"
	"github.com/google/uuid"
)

func main() {
	// clog 包是零配置的，不需要显式初始化
	cfg := coord.DefaultConfig()
	provider, err := coord.New(context.Background(), cfg)
	if err != nil {
		clog.Error("failed to create coordinator", clog.Err(err))
		os.Exit(1)
	}
	defer provider.Close()

	registryService := provider.Registry()
	const serviceName = "my-awesome-service"

	var wg sync.WaitGroup
	wg.Add(2)

	// --- 服务消费者 ---
	go func() {
		defer wg.Done()
		consumerCtx, cancel := context.WithCancel(context.Background())
		defer cancel()

		// 启动监听
		eventCh, err := registryService.Watch(consumerCtx, serviceName)
		if err != nil {
			clog.Error("[Consumer] Failed to start watching", clog.Err(err))
			return
		}
		clog.Info("[Consumer] Started watching for service changes.")

		// 监听 goroutine
		go func() {
			for event := range eventCh {
				clog.Info("[Consumer] Received service event",
					clog.String("type", string(event.Type)),
					clog.String("service_id", event.Service.ID),
					clog.String("service_name", event.Service.Name),
					clog.String("address", event.Service.Address),
				)
			}
			clog.Info("[Consumer] Watch channel closed.")
		}()

		// 1. 初始发现
		time.Sleep(2 * time.Second) // 等待服务注册
		discoverServices(registryService, serviceName)

		// 2. 等待服务下线
		clog.Info("[Consumer] Waiting for 6 seconds to see unregister event...")
		time.Sleep(6 * time.Second)

		// 3. 最终发现
		discoverServices(registryService, serviceName)

		// 停止监听
		cancel()
		time.Sleep(1 * time.Second) // 等待 watch goroutine 退出
	}()

	// --- 服务提供者 ---
	go func() {
		defer wg.Done()
		providerCtx, cancel := context.WithCancel(context.Background())
		defer cancel()

		// 注册两个服务实例
		service1 := registry.ServiceInfo{
			ID:      uuid.NewString(),
			Name:    serviceName,
			Address: "127.0.0.1",
			Port:    8080,
		}
		service2 := registry.ServiceInfo{
			ID:      uuid.NewString(),
			Name:    serviceName,
			Address: "127.0.0.1",
			Port:    8081,
		}

		if err := registryService.Register(providerCtx, service1, 10*time.Second); err != nil {
			clog.Error("[Provider] Failed to register service 1", clog.Err(err))
		} else {
			clog.Info("[Provider] Service 1 registered.", clog.String("id", service1.ID))
		}

		if err := registryService.Register(providerCtx, service2, 10*time.Second); err != nil {
			clog.Error("[Provider] Failed to register service 2", clog.Err(err))
		} else {
			clog.Info("[Provider] Service 2 registered.", clog.String("id", service2.ID))
		}

		// 让服务运行一段时间
		clog.Info("[Provider] Services are running for 5 seconds...")
		time.Sleep(5 * time.Second)

		// 注销其中一个服务
		clog.Info("[Provider] Unregistering service 1...")
		if err := registryService.Unregister(providerCtx, service1.ID); err != nil {
			clog.Error("[Provider] Failed to unregister service 1", clog.Err(err))
		} else {
			clog.Info("[Provider] Service 1 unregistered.")
		}

		// 等待示例结束
		time.Sleep(5 * time.Second)
	}()

	wg.Wait()
	fmt.Println("\nRegistry example finished.")
}

func discoverServices(r registry.ServiceRegistry, name string) {
	clog.Info("[Discovery] Discovering services...", clog.String("name", name))
	services, err := r.Discover(context.Background(), name)
	if err != nil {
		clog.Error("[Discovery] Failed to discover services", clog.Err(err))
		return
	}
	clog.Info("[Discovery] Found services", clog.Int("count", len(services)))
	for _, s := range services {
		clog.Info("[Discovery]  - Service",
			clog.String("id", s.ID),
			clog.String("address", s.Address),
			clog.Int("port", s.Port),
		)
	}
}
