package tools

import (
	"context"
	"fmt"
	"gochat/clog"
	"gochat/config"
	"sync"
	"time"

	"go.etcd.io/etcd/api/v3/mvccpb"
	clientv3 "go.etcd.io/etcd/client/v3"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/resolver"
)

var (
	etcdClient     *clientv3.Client
	etcdClientOnce sync.Once

	// 服务实例连接池管理器，key是serviceName
	serviceConnManagers      = make(map[string]*ServiceInstanceConnManager)
	serviceConnManagersMutex sync.RWMutex
)

// InitEtcdClient 初始化etcd客户端，确保只初始化一次
func InitEtcdClient() error {
	var err error
	clog.Debug("Initializing etcd client with endpoints: %v", config.Conf.Etcd.Addrs)
	etcdClientOnce.Do(func() {
		etcdClient, err = clientv3.New(clientv3.Config{
			Endpoints:   config.Conf.Etcd.Addrs,
			DialTimeout: 5 * time.Second,
		})
		if err != nil {
			clog.Error("Failed to initialize etcd client: %v", err)
		} else {
			clog.Info("Etcd client initialized successfully")
		}
	})
	return err
}

// ServiceRegistry 处理服务注册
func ServiceRegistry(ctx context.Context, serviceName, instanceID, addr string) error {
	if etcdClient == nil {
		clog.Error("Failed to register service: etcd client not initialized")
		return fmt.Errorf("etcd client not initialized, call InitEtcdClient first")
	}
	// 创建租约，TTL 设置为 5 秒
	lease, err := etcdClient.Grant(ctx, 5)
	if err != nil {
		clog.Error("Failed to create lease for service %s/%s: %v", serviceName, instanceID, err)
		return fmt.Errorf("failed to create lease: %w", err)
	}
	// 设置自动续租
	keepAliveCh, err := etcdClient.KeepAlive(ctx, lease.ID)
	if err != nil {
		clog.Error("Failed to setup lease keepalive for service %s/%s: %v", serviceName, instanceID, err)
		return fmt.Errorf("failed to setup keepalive: %w", err)
	}
	// 处理续租响应
	go func() {
		for {
			select {
			case resp, ok := <-keepAliveCh:
				if !ok {
					clog.Warning("Keepalive channel closed for service %s/%s", serviceName, instanceID)
					return
				}
				clog.Debug("Lease renewed for service %s/%s, TTL: %d", serviceName, instanceID, resp.TTL)
			case <-ctx.Done():
				clog.Info("Service registry context canceled for %s/%s", serviceName, instanceID)
				return
			}
		}
	}()
	// 注册服务实例，键为 /services/{serviceName}/{instanceID}，值为地址（如 127.0.0.1:8080）
	key := fmt.Sprintf("/services/%s/%s", serviceName, instanceID)
	_, err = etcdClient.Put(ctx, key, addr, clientv3.WithLease(lease.ID))
	if err != nil {
		clog.Error("Failed to register service %s/%s: %v", serviceName, instanceID, err)
		return fmt.Errorf("failed to register service: %w", err)
	}

	clog.Info("Service registered successfully: %s/%s at %s", serviceName, instanceID, addr)
	return nil
}

// ServiceDiscovery 使用etcd发现服务，返回支持负载均衡的gRPC连接
func ServiceDiscovery(ctx context.Context, serviceName string) (*grpc.ClientConn, error) {
	if etcdClient == nil {
		clog.Error("Failed to discover service %s: etcd client not initialized", serviceName)
		return nil, fmt.Errorf("etcd client not initialized, call InitEtcdClient first")
	}

	clog.Debug("Setting up service discovery for %s", serviceName)

	// 创建自定义resolver，关联etcd客户端和要查找的服务名
	builder := &etcdResolverBuilder{
		client:      etcdClient,
		serviceName: serviceName,
	}
	// 向gRPC注册这个自定义的解析器，使gRPC能够识别etcd://开头的URL
	resolver.Register(builder)
	// 创建gRPC连接
	conn, err := grpc.DialContext(
		ctx,
		fmt.Sprintf("etcd:///%s", serviceName),
		grpc.WithTransportCredentials(insecure.NewCredentials()),
		grpc.WithDefaultServiceConfig(`{"loadBalancingPolicy":"round_robin"}`),
	)
	if err != nil {
		clog.Error("Failed to create gRPC connection for service %s: %v", serviceName, err)
		return nil, fmt.Errorf("failed to create gRPC connection: %w", err)
	}

	clog.Info("Service discovery enabled for %s", serviceName)
	return conn, nil
}

// ServiceInstanceConnManager 管理服务实例连接
type ServiceInstanceConnManager struct {
	serviceName      string
	connMap          map[string]*grpc.ClientConn // 实例ID到连接的映射
	connMapMutex     sync.RWMutex
	ctx              context.Context
	cancel           context.CancelFunc
	watchInitialized sync.WaitGroup
}

// getServiceInstanceConnManager 获取服务实例连接管理器
func getServiceInstanceConnManager(serviceName string) (*ServiceInstanceConnManager, error) {
	if etcdClient == nil {
		clog.Error("Failed to get service manager: etcd client not initialized")
		return nil, fmt.Errorf("etcd client not initialized, call InitEtcdClient first")
	}

	serviceConnManagersMutex.RLock()
	manager, exists := serviceConnManagers[serviceName]
	serviceConnManagersMutex.RUnlock()

	if exists {
		return manager, nil
	}

	// 创建新的管理器
	serviceConnManagersMutex.Lock()
	defer serviceConnManagersMutex.Unlock()

	// 双重检查，避免在获取锁的过程中已有其他goroutine创建了管理器
	manager, exists = serviceConnManagers[serviceName]
	if exists {
		return manager, nil
	}

	clog.Debug("Creating service instance connection manager for %s", serviceName)

	ctx, cancel := context.WithCancel(context.Background())
	manager = &ServiceInstanceConnManager{
		serviceName: serviceName,
		connMap:     make(map[string]*grpc.ClientConn),
		ctx:         ctx,
		cancel:      cancel,
	}

	// 设置初始化同步
	manager.watchInitialized.Add(1)

	// 启动监听
	go manager.watchServiceInstances()

	// 等待初始化完成
	manager.watchInitialized.Wait()

	serviceConnManagers[serviceName] = manager
	clog.Info("Service instance connection manager created for %s", serviceName)
	return manager, nil
}

// watchServiceInstances 监听服务实例变化并维护连接
func (m *ServiceInstanceConnManager) watchServiceInstances() {
	prefix := fmt.Sprintf("/services/%s/", m.serviceName)

	// 首次获取所有服务实例
	resp, err := etcdClient.Get(m.ctx, prefix, clientv3.WithPrefix())
	if err != nil {
		clog.Error("Failed to get initial service instances for %s: %v", m.serviceName, err)
		m.watchInitialized.Done()
		return
	}

	// 为每个实例创建连接
	m.updateInstances(resp.Kvs)
	clog.Info("Initial service instances loaded for %s: found %d instances", m.serviceName, len(resp.Kvs))

	// 标记初始化完成
	m.watchInitialized.Done()

	// 监听后续变化
	watchChan := etcdClient.Watch(m.ctx, prefix, clientv3.WithPrefix())
	clog.Debug("Watching for service instance changes: %s", m.serviceName)

	for watchResp := range watchChan {
		for _, event := range watchResp.Events {
			key := string(event.Kv.Key)
			instanceID := key[len(prefix):]

			m.connMapMutex.Lock()

			switch event.Type {
			case clientv3.EventTypePut:
				// 新增或更新实例
				addr := string(event.Kv.Value)
				// 如果连接已存在，先关闭
				if conn, exists := m.connMap[instanceID]; exists {
					conn.Close()
					clog.Debug("Closing existing connection for updated instance %s/%s", m.serviceName, instanceID)
				}

				// 创建新连接
				conn, err := grpc.DialContext(
					m.ctx,
					addr,
					grpc.WithTransportCredentials(insecure.NewCredentials()),
				)
				if err == nil {
					m.connMap[instanceID] = conn
					clog.Debug("Created connection for instance %s/%s at %s", m.serviceName, instanceID, addr)
				} else {
					clog.Error("Failed to create connection for instance %s/%s: %v", m.serviceName, instanceID, err)
				}

			case clientv3.EventTypeDelete:
				// 删除实例
				if conn, exists := m.connMap[instanceID]; exists {
					conn.Close()
					delete(m.connMap, instanceID)
					clog.Debug("Removed connection for deleted instance %s/%s", m.serviceName, instanceID)
				}
			}

			m.connMapMutex.Unlock()
		}
	}
}

// updateInstances 更新实例连接
func (m *ServiceInstanceConnManager) updateInstances(kvs []*mvccpb.KeyValue) {
	prefix := fmt.Sprintf("/services/%s/", m.serviceName)
	currentInstanceIDs := make(map[string]bool)

	m.connMapMutex.Lock()
	defer m.connMapMutex.Unlock()

	// 添加或更新现有实例
	for _, kv := range kvs {
		key := string(kv.Key)
		instanceID := key[len(prefix):]
		addr := string(kv.Value)

		currentInstanceIDs[instanceID] = true

		// 创建新连接
		conn, err := grpc.DialContext(
			m.ctx,
			addr,
			grpc.WithTransportCredentials(insecure.NewCredentials()),
		)
		if err == nil {
			m.connMap[instanceID] = conn
		}
	}

	// 关闭并移除不再存在的实例连接
	for instanceID, conn := range m.connMap {
		if _, exists := currentInstanceIDs[instanceID]; !exists {
			conn.Close()
			delete(m.connMap, instanceID)
		}
	}
}

// CloseServiceManager 关闭服务连接管理器并清理所有连接
func CloseServiceManager(serviceName string) {
	serviceConnManagersMutex.Lock()
	defer serviceConnManagersMutex.Unlock()

	if manager, exists := serviceConnManagers[serviceName]; exists {
		clog.Info("Closing service connection manager for %s", serviceName)
		manager.cancel() // 取消监听

		manager.connMapMutex.Lock()
		for instanceID, conn := range manager.connMap {
			conn.Close()
			clog.Debug("Closed connection for instance %s/%s", serviceName, instanceID)
		}
		manager.connMapMutex.Unlock()

		delete(serviceConnManagers, serviceName)
	}
}

// GetServiceInstanceConn 获取特定服务实例的gRPC连接
func GetServiceInstanceConn(serviceName, instanceID string) (*grpc.ClientConn, error) {
	manager, err := getServiceInstanceConnManager(serviceName)
	if err != nil {
		clog.Error("Failed to get connection for %s/%s: %v", serviceName, instanceID, err)
		return nil, err
	}

	manager.connMapMutex.RLock()
	defer manager.connMapMutex.RUnlock()

	conn, exists := manager.connMap[instanceID]
	if !exists {
		clog.Warning("Connection not found for instance %s/%s", serviceName, instanceID)
		return nil, fmt.Errorf("connection not found for service instance: %s/%s", serviceName, instanceID)
	}

	clog.Debug("Retrieved connection for instance %s/%s", serviceName, instanceID)
	return conn, nil
}

// GetAllServiceInstanceConns 获取某个服务下所有的gRPC连接
func GetAllServiceInstanceConns(serviceName string) (map[string]*grpc.ClientConn, error) {
	manager, err := getServiceInstanceConnManager(serviceName)
	if err != nil {
		clog.Error("Failed to get connections for service %s: %v", serviceName, err)
		return nil, err
	}

	manager.connMapMutex.RLock()
	defer manager.connMapMutex.RUnlock()

	// 创建结果的副本
	result := make(map[string]*grpc.ClientConn, len(manager.connMap))
	for id, conn := range manager.connMap {
		result[id] = conn
	}

	clog.Debug("Retrieved all connections for service %s: found %d", serviceName, len(result))
	return result, nil
}

// etcdResolverBuilder 实现resolver.Builder接口
type etcdResolverBuilder struct {
	client      *clientv3.Client
	serviceName string
}

// Build 接收目标服务信息和gRPC客户端连接，构建并返回一个新的解析器
func (b *etcdResolverBuilder) Build(target resolver.Target, cc resolver.ClientConn, opts resolver.BuildOptions) (resolver.Resolver, error) {
	r := &etcdResolver{
		client:      b.client,
		serviceName: b.serviceName,
		cc:          cc,
	}
	// 初始解析
	r.start()
	return r, nil
}

// Scheme 定义 URL 方案为 etcd，使 gRPC 能够识别 etcd:///{serviceName} 形式的地址
func (b *etcdResolverBuilder) Scheme() string {
	return "etcd"
}

// etcdResolver 实现resolver.Resolver接口
type etcdResolver struct {
	client      *clientv3.Client
	serviceName string
	cc          resolver.ClientConn
}

func (r *etcdResolver) start() {
	// 获取初始服务列表
	go r.watch()
}

// watch 监听服务变化
func (r *etcdResolver) watch() {
	prefix := fmt.Sprintf("/services/%s/", r.serviceName)

	for {
		// 获取当前所有服务实例
		resp, err := r.client.Get(context.Background(), prefix, clientv3.WithPrefix())
		if err != nil {
			clog.Error("Resolver failed to get services for %s: %v", r.serviceName, err)
			time.Sleep(1 * time.Second)
			continue
		}

		var addresses []resolver.Address
		for _, kv := range resp.Kvs {
			addresses = append(addresses, resolver.Address{Addr: string(kv.Value)})
		}

		// 更新gRPC客户端连接状态
		err = r.cc.UpdateState(resolver.State{Addresses: addresses})
		if err != nil {
			clog.Error("Resolver failed to update state for %s: %v", r.serviceName, err)
			return
		}

		clog.Debug("Resolver updated %d addresses for service %s", len(addresses), r.serviceName)

		// 监听服务变更
		watchChan := r.client.Watch(context.Background(), prefix, clientv3.WithPrefix())
		for range watchChan {
			// 服务列表发生变化，重新获取列表
			break
		}
	}
}

func (r *etcdResolver) ResolveNow(resolver.ResolveNowOptions) {}

func (r *etcdResolver) Close() {}

// CloseEtcdClient 关闭etcd客户端连接并释放资源
func CloseEtcdClient() error {
	clog.Debug("Attempting to close etcd client")

	if etcdClient == nil {
		clog.Debug("Etcd client already closed or not initialized")
		return nil
	}

	// 关闭etcd客户端
	if err := etcdClient.Close(); err != nil {
		clog.Error("Failed to close etcd client: %v", err)
		return err
	}

	etcdClient = nil
	clog.Info("Etcd client closed successfully")
	return nil
}
