package internal

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"
	"time"

	"github.com/ceyewan/gochat/im-infra/clog"
	clientv3 "go.etcd.io/etcd/client/v3"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/resolver"
)

// serviceRegistry 是 ServiceRegistry 接口的内部实现
type serviceRegistry struct {
	client    *clientv3.Client
	config    ServiceRegistryConfig
	logger    clog.Logger
	leases    map[string]clientv3.LeaseID // 服务实例的租约映射
	leaseMu   sync.RWMutex
	watchers  map[string]context.CancelFunc // 监听器映射
	watcherMu sync.RWMutex
	closed    bool
	closeMu   sync.RWMutex
}

// newServiceRegistry 创建新的服务注册实例
func newServiceRegistry(client *clientv3.Client, config ServiceRegistryConfig, logger clog.Logger) ServiceRegistry {
	return &serviceRegistry{
		client:   client,
		config:   config,
		logger:   logger,
		leases:   make(map[string]clientv3.LeaseID),
		watchers: make(map[string]context.CancelFunc),
	}
}

// Register 注册服务实例
func (sr *serviceRegistry) Register(ctx context.Context, service ServiceInfo) error {
	sr.closeMu.RLock()
	defer sr.closeMu.RUnlock()

	if sr.closed {
		return fmt.Errorf("service registry is closed")
	}

	// 验证服务信息
	if err := sr.validateServiceInfo(service); err != nil {
		return fmt.Errorf("invalid service info: %w", err)
	}

	// 设置默认值
	if service.Health == HealthUnknown {
		service.Health = HealthHealthy
	}
	service.RegisterTime = time.Now()
	service.LastHeartbeat = time.Now()

	// 创建租约
	lease, err := sr.client.Grant(ctx, int64(sr.config.TTL.Seconds()))
	if err != nil {
		sr.logger.Error("创建租约失败",
			clog.Err(err),
			clog.String("service", service.Name),
			clog.String("instance", service.InstanceID),
		)
		return fmt.Errorf("failed to create lease: %w", err)
	}

	// 序列化服务信息
	serviceData, err := json.Marshal(service)
	if err != nil {
		return fmt.Errorf("failed to marshal service info: %w", err)
	}

	// 构建键名
	key := sr.buildServiceKey(service.Name, service.InstanceID)

	// 注册服务
	_, err = sr.client.Put(ctx, key, string(serviceData), clientv3.WithLease(lease.ID))
	if err != nil {
		sr.logger.Error("注册服务失败",
			clog.Err(err),
			clog.String("service", service.Name),
			clog.String("instance", service.InstanceID),
			clog.String("key", key),
		)
		return fmt.Errorf("failed to register service: %w", err)
	}

	// 保存租约ID
	sr.leaseMu.Lock()
	sr.leases[key] = lease.ID
	sr.leaseMu.Unlock()

	// 启动租约续期
	go sr.keepAlive(ctx, lease.ID, service.Name, service.InstanceID)

	// 启动健康检查（如果启用）
	if sr.config.EnableHealthCheck {
		go sr.healthCheck(ctx, service)
	}

	sr.logger.Info("服务注册成功",
		clog.String("service", service.Name),
		clog.String("instance", service.InstanceID),
		clog.String("address", service.Address),
		clog.Duration("ttl", sr.config.TTL),
	)

	return nil
}

// Deregister 注销服务实例
func (sr *serviceRegistry) Deregister(ctx context.Context, serviceName, instanceID string) error {
	sr.closeMu.RLock()
	defer sr.closeMu.RUnlock()

	if sr.closed {
		return fmt.Errorf("service registry is closed")
	}

	key := sr.buildServiceKey(serviceName, instanceID)

	// 删除服务记录
	_, err := sr.client.Delete(ctx, key)
	if err != nil {
		sr.logger.Error("注销服务失败",
			clog.Err(err),
			clog.String("service", serviceName),
			clog.String("instance", instanceID),
		)
		return fmt.Errorf("failed to deregister service: %w", err)
	}

	// 清理租约
	sr.leaseMu.Lock()
	if leaseID, exists := sr.leases[key]; exists {
		sr.client.Revoke(context.Background(), leaseID)
		delete(sr.leases, key)
	}
	sr.leaseMu.Unlock()

	sr.logger.Info("服务注销成功",
		clog.String("service", serviceName),
		clog.String("instance", instanceID),
	)

	return nil
}

// Discover 发现指定服务的所有健康实例
func (sr *serviceRegistry) Discover(ctx context.Context, serviceName string) ([]ServiceInfo, error) {
	sr.closeMu.RLock()
	defer sr.closeMu.RUnlock()

	if sr.closed {
		return nil, fmt.Errorf("service registry is closed")
	}

	prefix := sr.buildServicePrefix(serviceName)

	resp, err := sr.client.Get(ctx, prefix, clientv3.WithPrefix())
	if err != nil {
		sr.logger.Error("发现服务失败",
			clog.Err(err),
			clog.String("service", serviceName),
		)
		return nil, fmt.Errorf("failed to discover service: %w", err)
	}

	var services []ServiceInfo
	for _, kv := range resp.Kvs {
		var service ServiceInfo
		if err := json.Unmarshal(kv.Value, &service); err != nil {
			sr.logger.Warn("解析服务信息失败",
				clog.Err(err),
				clog.String("key", string(kv.Key)),
			)
			continue
		}

		// 只返回健康的服务实例
		if service.Health == HealthHealthy {
			services = append(services, service)
		}
	}

	sr.logger.Debug("发现服务实例",
		clog.String("service", serviceName),
		clog.Int("count", len(services)),
	)

	return services, nil
}

// Watch 监听指定服务的实例变化
func (sr *serviceRegistry) Watch(ctx context.Context, serviceName string) (<-chan []ServiceInfo, error) {
	sr.closeMu.RLock()
	defer sr.closeMu.RUnlock()

	if sr.closed {
		return nil, fmt.Errorf("service registry is closed")
	}

	prefix := sr.buildServicePrefix(serviceName)
	ch := make(chan []ServiceInfo, 10)

	// 创建监听上下文
	watchCtx, cancel := context.WithCancel(ctx)

	// 保存取消函数
	sr.watcherMu.Lock()
	sr.watchers[serviceName] = cancel
	sr.watcherMu.Unlock()

	go func() {
		defer close(ch)
		defer func() {
			sr.watcherMu.Lock()
			delete(sr.watchers, serviceName)
			sr.watcherMu.Unlock()
		}()

		// 首先发送当前状态
		if services, err := sr.Discover(watchCtx, serviceName); err == nil {
			select {
			case ch <- services:
			case <-watchCtx.Done():
				return
			}
		}

		// 监听变化
		watchCh := sr.client.Watch(watchCtx, prefix, clientv3.WithPrefix())
		for {
			select {
			case <-watchCtx.Done():
				return
			case watchResp, ok := <-watchCh:
				if !ok {
					return
				}

				if watchResp.Err() != nil {
					sr.logger.Error("监听服务变化失败",
						clog.Err(watchResp.Err()),
						clog.String("service", serviceName),
					)
					continue
				}

				// 获取最新的服务列表
				if services, err := sr.Discover(watchCtx, serviceName); err == nil {
					select {
					case ch <- services:
					case <-watchCtx.Done():
						return
					}
				}
			}
		}
	}()

	return ch, nil
}

// UpdateHealth 更新服务实例的健康状态
func (sr *serviceRegistry) UpdateHealth(ctx context.Context, serviceName, instanceID string, status HealthStatus) error {
	sr.closeMu.RLock()
	defer sr.closeMu.RUnlock()

	if sr.closed {
		return fmt.Errorf("service registry is closed")
	}

	key := sr.buildServiceKey(serviceName, instanceID)

	// 获取当前服务信息
	resp, err := sr.client.Get(ctx, key)
	if err != nil {
		return fmt.Errorf("failed to get service info: %w", err)
	}

	if len(resp.Kvs) == 0 {
		return fmt.Errorf("service instance not found")
	}

	var service ServiceInfo
	if err := json.Unmarshal(resp.Kvs[0].Value, &service); err != nil {
		return fmt.Errorf("failed to unmarshal service info: %w", err)
	}

	// 更新健康状态
	service.Health = status
	service.LastHeartbeat = time.Now()

	// 序列化并更新
	serviceData, err := json.Marshal(service)
	if err != nil {
		return fmt.Errorf("failed to marshal service info: %w", err)
	}

	_, err = sr.client.Put(ctx, key, string(serviceData))
	if err != nil {
		return fmt.Errorf("failed to update health status: %w", err)
	}

	sr.logger.Debug("更新服务健康状态",
		clog.String("service", serviceName),
		clog.String("instance", instanceID),
		clog.String("status", status.String()),
	)

	return nil
}

// GetConnection 获取到指定服务的 gRPC 连接，支持负载均衡
func (sr *serviceRegistry) GetConnection(ctx context.Context, serviceName string, strategy LoadBalanceStrategy) (*grpc.ClientConn, error) {
	sr.closeMu.RLock()
	defer sr.closeMu.RUnlock()

	if sr.closed {
		return nil, fmt.Errorf("service registry is closed")
	}

	// 注册自定义解析器
	builder := &etcdResolverBuilder{
		client:      sr.client,
		serviceName: serviceName,
		registry:    sr,
	}
	resolver.Register(builder)

	// 创建连接
	target := fmt.Sprintf("etcd:///%s", serviceName)
	conn, err := grpc.NewClient(
		target,
		grpc.WithTransportCredentials(insecure.NewCredentials()),
		grpc.WithDefaultServiceConfig(fmt.Sprintf(`{"loadBalancingPolicy":"%s"}`, strategy.String())),
	)
	if err != nil {
		sr.logger.Error("创建 gRPC 连接失败",
			clog.Err(err),
			clog.String("service", serviceName),
			clog.String("strategy", strategy.String()),
		)
		return nil, fmt.Errorf("failed to create gRPC connection: %w", err)
	}

	sr.logger.Debug("创建 gRPC 连接成功",
		clog.String("service", serviceName),
		clog.String("strategy", strategy.String()),
	)

	return conn, nil
}

// 辅助方法

// validateServiceInfo 验证服务信息
func (sr *serviceRegistry) validateServiceInfo(service ServiceInfo) error {
	if service.Name == "" {
		return fmt.Errorf("service name cannot be empty")
	}
	if service.InstanceID == "" {
		return fmt.Errorf("instance ID cannot be empty")
	}
	if service.Address == "" {
		return fmt.Errorf("service address cannot be empty")
	}
	return nil
}

// buildServiceKey 构建服务键名
func (sr *serviceRegistry) buildServiceKey(serviceName, instanceID string) string {
	return fmt.Sprintf("%s/%s/%s", sr.config.KeyPrefix, serviceName, instanceID)
}

// buildServicePrefix 构建服务前缀
func (sr *serviceRegistry) buildServicePrefix(serviceName string) string {
	return fmt.Sprintf("%s/%s/", sr.config.KeyPrefix, serviceName)
}

// keepAlive 保持租约活跃
func (sr *serviceRegistry) keepAlive(ctx context.Context, leaseID clientv3.LeaseID, serviceName, instanceID string) {
	keepAliveCh, err := sr.client.KeepAlive(ctx, leaseID)
	if err != nil {
		sr.logger.Error("启动租约续期失败",
			clog.Err(err),
			clog.String("service", serviceName),
			clog.String("instance", instanceID),
		)
		return
	}

	for {
		select {
		case <-ctx.Done():
			sr.logger.Debug("租约续期上下文取消",
				clog.String("service", serviceName),
				clog.String("instance", instanceID),
			)
			return
		case resp, ok := <-keepAliveCh:
			if !ok {
				sr.logger.Warn("租约续期通道关闭",
					clog.String("service", serviceName),
					clog.String("instance", instanceID),
				)
				return
			}
			if resp != nil {
				sr.logger.Debug("租约续期成功",
					clog.String("service", serviceName),
					clog.String("instance", instanceID),
					clog.Int64("ttl", resp.TTL),
				)
			}
		}
	}
}

// healthCheck 执行健康检查
func (sr *serviceRegistry) healthCheck(ctx context.Context, service ServiceInfo) {
	ticker := time.NewTicker(sr.config.HealthCheckInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			// 这里可以实现具体的健康检查逻辑
			// 目前只是更新心跳时间
			if err := sr.UpdateHealth(ctx, service.Name, service.InstanceID, HealthHealthy); err != nil {
				sr.logger.Warn("健康检查更新失败",
					clog.Err(err),
					clog.String("service", service.Name),
					clog.String("instance", service.InstanceID),
				)
			}
		}
	}
}

// Close 关闭服务注册器
func (sr *serviceRegistry) Close() error {
	sr.closeMu.Lock()
	defer sr.closeMu.Unlock()

	if sr.closed {
		return nil
	}

	sr.closed = true

	// 取消所有监听器
	sr.watcherMu.Lock()
	for _, cancel := range sr.watchers {
		cancel()
	}
	sr.watchers = make(map[string]context.CancelFunc)
	sr.watcherMu.Unlock()

	// 撤销所有租约
	sr.leaseMu.Lock()
	for _, leaseID := range sr.leases {
		sr.client.Revoke(context.Background(), leaseID)
	}
	sr.leases = make(map[string]clientv3.LeaseID)
	sr.leaseMu.Unlock()

	sr.logger.Info("服务注册器已关闭")
	return nil
}

// etcdResolverBuilder 实现 gRPC resolver.Builder 接口
type etcdResolverBuilder struct {
	client      *clientv3.Client
	serviceName string
	registry    *serviceRegistry
}

// Build 构建解析器
func (erb *etcdResolverBuilder) Build(target resolver.Target, cc resolver.ClientConn, opts resolver.BuildOptions) (resolver.Resolver, error) {
	r := &etcdResolver{
		client:      erb.client,
		serviceName: erb.serviceName,
		registry:    erb.registry,
		cc:          cc,
	}
	r.start()
	return r, nil
}

// Scheme 返回解析器方案
func (erb *etcdResolverBuilder) Scheme() string {
	return "etcd"
}

// etcdResolver 实现 gRPC resolver.Resolver 接口
type etcdResolver struct {
	client      *clientv3.Client
	serviceName string
	registry    *serviceRegistry
	cc          resolver.ClientConn
	ctx         context.Context
	cancel      context.CancelFunc
}

// start 启动解析器
func (er *etcdResolver) start() {
	er.ctx, er.cancel = context.WithCancel(context.Background())
	go er.watch()
}

// watch 监听服务变化
func (er *etcdResolver) watch() {
	// 获取初始服务列表
	er.updateAddresses()

	// 监听服务变化
	ch, err := er.registry.Watch(er.ctx, er.serviceName)
	if err != nil {
		er.registry.logger.Error("监听服务变化失败",
			clog.Err(err),
			clog.String("service", er.serviceName),
		)
		return
	}

	for {
		select {
		case <-er.ctx.Done():
			return
		case services, ok := <-ch:
			if !ok {
				return
			}
			er.updateAddressesFromServices(services)
		}
	}
}

// updateAddresses 更新地址列表
func (er *etcdResolver) updateAddresses() {
	services, err := er.registry.Discover(er.ctx, er.serviceName)
	if err != nil {
		er.registry.logger.Error("获取服务列表失败",
			clog.Err(err),
			clog.String("service", er.serviceName),
		)
		return
	}

	er.updateAddressesFromServices(services)
}

// updateAddressesFromServices 从服务列表更新地址
func (er *etcdResolver) updateAddressesFromServices(services []ServiceInfo) {
	var addresses []resolver.Address
	for _, service := range services {
		addresses = append(addresses, resolver.Address{
			Addr: service.Address,
		})
	}

	err := er.cc.UpdateState(resolver.State{Addresses: addresses})
	if err != nil {
		er.registry.logger.Error("更新解析器状态失败",
			clog.Err(err),
			clog.String("service", er.serviceName),
		)
	} else {
		er.registry.logger.Debug("更新解析器地址",
			clog.String("service", er.serviceName),
			clog.Int("count", len(addresses)),
		)
	}
}

// ResolveNow 立即解析
func (er *etcdResolver) ResolveNow(resolver.ResolveNowOptions) {
	er.updateAddresses()
}

// Close 关闭解析器
func (er *etcdResolver) Close() {
	if er.cancel != nil {
		er.cancel()
	}
}
