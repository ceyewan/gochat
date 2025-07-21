package etcd

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"sync"
	"time"

	clientv3 "go.etcd.io/etcd/client/v3"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

// etcdServiceDiscovery 实现 ServiceDiscovery 接口
type etcdServiceDiscovery struct {
	connMgr ConnectionManager
	logger  Logger
	options *ManagerOptions
	mu      sync.RWMutex
}

// NewServiceDiscoveryWithManager 使用连接管理器创建服务发现组件
func NewServiceDiscoveryWithManager(connMgr ConnectionManager, logger Logger, options *ManagerOptions) (ServiceDiscovery, error) {
	if connMgr == nil {
		return nil, WrapConfigurationError(ErrInvalidConfiguration, "connection manager cannot be nil")
	}

	return &etcdServiceDiscovery{
		connMgr: connMgr,
		logger:  logger,
		options: options,
	}, nil
}

// GetConnection 获取服务的 gRPC 连接
func (sd *etcdServiceDiscovery) GetConnection(ctx context.Context, serviceName string, options ...DiscoveryOption) (*grpc.ClientConn, error) {
	opts := &DiscoveryOptions{
		LoadBalancer: "round_robin",
		Timeout:      5 * time.Second,
	}

	// 应用选项
	for _, opt := range options {
		opt(opts)
	}

	// 获取服务端点
	endpoints, err := sd.GetServiceEndpoints(ctx, serviceName)
	if err != nil {
		return nil, WrapDiscoveryError(err, "failed to get service endpoints")
	}

	if len(endpoints) == 0 {
		return nil, WrapDiscoveryError(ErrNoAvailableInstances, "no available instances for service: "+serviceName)
	}

	// 创建 gRPC 连接
	// 简单实现：使用第一个可用的端点
	target := endpoints[0]

	dialCtx, cancel := context.WithTimeout(ctx, opts.Timeout)
	defer cancel()

	conn, err := grpc.DialContext(dialCtx, target,
		grpc.WithTransportCredentials(insecure.NewCredentials()),
		grpc.WithBlock(),
	)
	if err != nil {
		// 如果第一个端点失败，尝试其他端点
		for i := 1; i < len(endpoints); i++ {
			dialCtx, cancel := context.WithTimeout(ctx, opts.Timeout)
			conn, err = grpc.DialContext(dialCtx, endpoints[i],
				grpc.WithTransportCredentials(insecure.NewCredentials()),
				grpc.WithBlock(),
			)
			cancel()
			if err == nil {
				break
			}
		}

		if err != nil {
			return nil, WrapDiscoveryError(err, "failed to connect to any service instance")
		}
	}

	return conn, nil
}

// GetServiceEndpoints 获取服务的所有端点
func (sd *etcdServiceDiscovery) GetServiceEndpoints(ctx context.Context, serviceName string) ([]string, error) {
	client := sd.connMgr.GetClient()
	if client == nil {
		return nil, ErrNotConnected
	}

	// 构建服务前缀
	prefix := fmt.Sprintf("%s/%s/", sd.options.ServicePrefix, serviceName)

	// 获取服务实例
	resp, err := client.Get(ctx, prefix, clientv3.WithPrefix())
	if err != nil {
		return nil, WrapDiscoveryError(err, "failed to get service instances from etcd")
	}

	var endpoints []string
	for _, kv := range resp.Kvs {
		// 解析服务信息
		var serviceInstance struct {
			Address  string            `json:"address"`
			Metadata map[string]string `json:"metadata,omitempty"`
		}

		if err := json.Unmarshal(kv.Value, &serviceInstance); err != nil {
			sd.logger.Warnf("Failed to parse service instance: %v", err)
			continue
		}

		endpoints = append(endpoints, serviceInstance.Address)
	}

	if len(endpoints) == 0 {
		return nil, WrapDiscoveryError(ErrServiceNotFound, "service not found: "+serviceName)
	}

	return endpoints, nil
}

// WatchService 监听服务变化
func (sd *etcdServiceDiscovery) WatchService(ctx context.Context, serviceName string) (<-chan ServiceEvent, error) {
	client := sd.connMgr.GetClient()
	if client == nil {
		return nil, ErrNotConnected
	}

	eventCh := make(chan ServiceEvent, 10)
	prefix := fmt.Sprintf("%s/%s/", sd.options.ServicePrefix, serviceName)

	go func() {
		defer close(eventCh)

		watchCh := client.Watch(ctx, prefix, clientv3.WithPrefix())
		for watchResp := range watchCh {
			if watchResp.Err() != nil {
				sd.logger.Errorf("Watch error: %v", watchResp.Err())
				return
			}

			for _, event := range watchResp.Events {
				serviceEvent := sd.parseWatchEvent(event, serviceName)
				if serviceEvent != nil {
					select {
					case eventCh <- *serviceEvent:
					case <-ctx.Done():
						return
					}
				}
			}
		}
	}()

	return eventCh, nil
}

// ResolveService 解析服务的所有实例
func (sd *etcdServiceDiscovery) ResolveService(ctx context.Context, serviceName string) ([]ServiceInstance, error) {
	client := sd.connMgr.GetClient()
	if client == nil {
		return nil, ErrNotConnected
	}

	prefix := fmt.Sprintf("%s/%s/", sd.options.ServicePrefix, serviceName)

	resp, err := client.Get(ctx, prefix, clientv3.WithPrefix())
	if err != nil {
		return nil, WrapDiscoveryError(err, "failed to resolve service instances")
	}

	var instances []ServiceInstance
	for _, kv := range resp.Kvs {
		// 解析实例ID
		key := string(kv.Key)
		parts := strings.Split(key, "/")
		if len(parts) < 3 {
			continue
		}
		instanceID := parts[len(parts)-1]

		// 解析服务信息
		var serviceData struct {
			Address  string            `json:"address"`
			Metadata map[string]string `json:"metadata,omitempty"`
			TTL      int64             `json:"ttl,omitempty"`
		}

		if err := json.Unmarshal(kv.Value, &serviceData); err != nil {
			sd.logger.Warnf("Failed to parse service instance %s: %v", instanceID, err)
			continue
		}

		instance := ServiceInstance{
			ID:       instanceID,
			Address:  serviceData.Address,
			Metadata: serviceData.Metadata,
			TTL:      serviceData.TTL,
		}

		instances = append(instances, instance)
	}

	if len(instances) == 0 {
		return nil, WrapDiscoveryError(ErrServiceNotFound, "no instances found for service: "+serviceName)
	}

	return instances, nil
}

// parseWatchEvent 解析监视事件
func (sd *etcdServiceDiscovery) parseWatchEvent(event *clientv3.Event, serviceName string) *ServiceEvent {
	key := string(event.Kv.Key)
	parts := strings.Split(key, "/")
	if len(parts) < 3 {
		return nil
	}

	instanceID := parts[len(parts)-1]

	var serviceEvent ServiceEvent
	serviceEvent.Service = serviceName

	switch event.Type {
	case clientv3.EventTypePut:
		// 可能是新增或更新
		var serviceData struct {
			Address  string            `json:"address"`
			Metadata map[string]string `json:"metadata,omitempty"`
			TTL      int64             `json:"ttl,omitempty"`
		}

		if err := json.Unmarshal(event.Kv.Value, &serviceData); err != nil {
			sd.logger.Warnf("Failed to parse service event: %v", err)
			return nil
		}

		serviceEvent.Instance = ServiceInstance{
			ID:       instanceID,
			Address:  serviceData.Address,
			Metadata: serviceData.Metadata,
			TTL:      serviceData.TTL,
		}

		// 判断是新增还是更新（简化实现，统一当作新增处理）
		serviceEvent.Type = ServiceEventAdd

	case clientv3.EventTypeDelete:
		serviceEvent.Type = ServiceEventDelete
		serviceEvent.Instance = ServiceInstance{
			ID: instanceID,
		}
	}

	return &serviceEvent
}
