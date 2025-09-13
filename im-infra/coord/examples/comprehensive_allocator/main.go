package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/ceyewan/gochat/im-infra/clog"
	"github.com/ceyewan/gochat/im-infra/coord"
	"github.com/ceyewan/gochat/im-infra/coord/allocator"
)

// ServiceInstance 模拟一个服务实例
type ServiceInstance struct {
	ID        int
	Service   string
	StartTime time.Time
	logger    clog.Logger
	allocator allocator.AllocatedID
}

// NewServiceInstance 创建一个新的服务实例
func NewServiceInstance(ctx context.Context, provider coord.Provider, serviceName string) (*ServiceInstance, error) {
	logger := clog.Namespace("service-instance")

	// 获取实例 ID 分配器
	allocator, err := provider.InstanceIDAllocator(serviceName, 100)
	if err != nil {
		return nil, fmt.Errorf("创建分配器失败: %w", err)
	}

	// 分配实例 ID
	instanceID, err := allocator.AcquireID(ctx)
	if err != nil {
		return nil, fmt.Errorf("分配实例 ID 失败: %w", err)
	}

	instance := &ServiceInstance{
		ID:        instanceID.ID(),
		Service:   serviceName,
		StartTime: time.Now(),
		logger:    logger.With(clog.Int("instance_id", instanceID.ID())),
		allocator: instanceID,
	}

	instance.logger.Info("服务实例创建成功")
	return instance, nil
}

// Start 启动服务实例
func (si *ServiceInstance) Start(ctx context.Context) error {
	si.logger.Info("启动服务实例")

	// 模拟服务运行
	go func() {
		ticker := time.NewTicker(5 * time.Second)
		defer ticker.Stop()

		for {
			select {
			case <-ticker.C:
				si.logger.Info("服务实例正常运行", 
					clog.Duration("uptime", time.Since(si.StartTime)))
			case <-ctx.Done():
				si.logger.Info("接收到停止信号")
				return
			}
		}
	}()

	return nil
}

// Stop 停止服务实例
func (si *ServiceInstance) Stop(ctx context.Context) error {
	si.logger.Info("停止服务实例")

	// 释放实例 ID
	if si.allocator != nil {
		err := si.allocator.Close(ctx)
		if err != nil {
			si.logger.Error("释放实例 ID 失败", clog.Err(err))
			return err
		}
		si.logger.Info("实例 ID 已释放")
	}

	return nil
}

// ClusterManager 模拟集群管理器
type ClusterManager struct {
	provider     coord.Provider
	serviceName  string
	instances    map[int]*ServiceInstance
	instanceChan chan *ServiceInstance
	logger       clog.Logger
	ctx          context.Context
	cancel       context.CancelFunc
}

// NewClusterManager 创建集群管理器
func NewClusterManager(provider coord.Provider, serviceName string) *ClusterManager {
	ctx, cancel := context.WithCancel(context.Background())

	return &ClusterManager{
		provider:     provider,
		serviceName:  serviceName,
		instances:    make(map[int]*ServiceInstance),
		instanceChan: make(chan *ServiceInstance, 10),
		logger:       clog.Namespace("cluster-manager"),
		ctx:          ctx,
		cancel:       cancel,
	}
}

// Start 启动集群管理器
func (cm *ClusterManager) Start() error {
	cm.logger.Info("启动集群管理器")

	// 启动实例管理 goroutine
	go cm.manageInstances()

	// 启动健康检查
	go cm.healthCheck()

	return nil
}

// manageInstances 管理服务实例
func (cm *ClusterManager) manageInstances() {
	for {
		select {
		case instance := <-cm.instanceChan:
			cm.instances[instance.ID] = instance
			cm.logger.Info("添加服务实例到集群", 
				clog.Int("instance_id", instance.ID),
				clog.Int("total_instances", len(cm.instances)))
		case <-cm.ctx.Done():
			cm.logger.Info("停止管理实例")
			return
		}
	}
}

// healthCheck 健康检查
func (cm *ClusterManager) healthCheck() {
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			cm.logger.Info("执行健康检查", 
				clog.Int("active_instances", len(cm.instances)))
			
			// 模拟健康检查逻辑
			for id, instance := range cm.instances {
				cm.logger.Debug("实例健康状态", 
					clog.Int("instance_id", id),
					clog.Duration("uptime", time.Since(instance.StartTime)))
			}
		case <-cm.ctx.Done():
			return
		}
	}
}

// AddInstance 添加新的服务实例
func (cm *ClusterManager) AddInstance(ctx context.Context) (*ServiceInstance, error) {
	instance, err := NewServiceInstance(ctx, cm.provider, cm.serviceName)
	if err != nil {
		return nil, fmt.Errorf("创建服务实例失败: %w", err)
	}

	err = instance.Start(cm.ctx)
	if err != nil {
		instance.Stop(ctx)
		return nil, fmt.Errorf("启动服务实例失败: %w", err)
	}

	cm.instanceChan <- instance
	return instance, nil
}

// RemoveInstance 移除服务实例
func (cm *ClusterManager) RemoveInstance(ctx context.Context, instanceID int) error {
	instance, exists := cm.instances[instanceID]
	if !exists {
		return fmt.Errorf("实例 %d 不存在", instanceID)
	}

	err := instance.Stop(ctx)
	if err != nil {
		return fmt.Errorf("停止实例失败: %w", err)
	}

	delete(cm.instances, instanceID)
	cm.logger.Info("移除服务实例", 
		clog.Int("instance_id", instanceID),
		clog.Int("remaining_instances", len(cm.instances)))

	return nil
}

// Stop 停止集群管理器
func (cm *ClusterManager) Stop() error {
	cm.logger.Info("停止集群管理器")
	cm.cancel()

	// 停止所有实例
	for id, instance := range cm.instances {
		err := instance.Stop(context.Background())
		if err != nil {
			cm.logger.Error("停止实例失败", 
				clog.Int("instance_id", id),
				clog.Err(err))
		}
	}

	cm.logger.Info("集群管理器已停止")
	return nil
}

func main() {
	// 初始化日志
	config := clog.GetDefaultConfig("development")
	if err := clog.Init(context.Background(), config, clog.WithNamespace("comprehensive-allocator")); err != nil {
		log.Fatalf("初始化 clog 失败: %v", err)
	}

	logger := clog.Namespace("main")
	logger.Info("InstanceIDAllocator 综合示例启动")

	// 创建 coord provider
	coordConfig := coord.GetDefaultConfig("development")
	coordConfig.Endpoints = []string{"localhost:2379"}

	coordProvider, err := coord.New(context.Background(), coordConfig, coord.WithLogger(clog.Namespace("coord")))
	if err != nil {
		logger.Fatal("创建 coord provider 失败", clog.Err(err))
	}
	defer coordProvider.Close()

	// 创建集群管理器
	clusterManager := NewClusterManager(coordProvider, "comprehensive-service")
	err = clusterManager.Start()
	if err != nil {
		logger.Fatal("启动集群管理器失败", clog.Err(err))
	}
	defer clusterManager.Stop()

	// 模拟动态添加和移除实例
	logger.Info("开始模拟集群生命周期")

	// 初始启动几个实例
	for i := 0; i < 3; i++ {
		instance, err := clusterManager.AddInstance(context.Background())
		if err != nil {
			logger.Error("添加实例失败", clog.Err(err))
			continue
		}
		logger.Info("添加初始实例", clog.Int("instance_id", instance.ID))
		time.Sleep(1 * time.Second)
	}

	// 模拟运行一段时间后添加更多实例
	time.Sleep(10 * time.Second)

	for i := 0; i < 2; i++ {
		instance, err := clusterManager.AddInstance(context.Background())
		if err != nil {
			logger.Error("添加实例失败", clog.Err(err))
			continue
		}
		logger.Info("扩展集群，添加新实例", clog.Int("instance_id", instance.ID))
		time.Sleep(2 * time.Second)
	}

	// 模拟移除一些实例
	time.Sleep(15 * time.Second)

	if len(clusterManager.instances) > 0 {
		// 获取第一个实例的 ID
		var firstInstanceID int
		for id := range clusterManager.instances {
			firstInstanceID = id
			break
		}

		err = clusterManager.RemoveInstance(context.Background(), firstInstanceID)
		if err != nil {
			logger.Error("移除实例失败", clog.Err(err))
		} else {
			logger.Info("收缩集群，移除实例", clog.Int("instance_id", firstInstanceID))
		}
	}

	// 等待信号
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	logger.Info("服务运行中，按 Ctrl+C 退出...")
	<-sigChan

	logger.Info("接收到退出信号，开始清理...")
	
	// 清理会自动完成（通过 defer）

	logger.Info("InstanceIDAllocator 综合示例结束")
}