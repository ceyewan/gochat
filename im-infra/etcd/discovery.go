package etcd

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/ceyewan/gochat/im-infra/clog"
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
	etcdLogger := clog.Default().WithGroup("etcd")
	etcdLogger.Info("创建服务发现组件")

	if connMgr == nil {
		etcdLogger.Error("连接管理器为空")
		return nil, WrapConfigurationError(ErrInvalidConfiguration, "connection manager cannot be nil")
	}

	if logger == nil {
		etcdLogger.Info("日志器为空，使用 clog 适配器")
		logger = NewClogAdapter(etcdLogger)
	}

	etcdLogger.Info("服务发现组件创建成功")
	return &etcdServiceDiscovery{
		connMgr: connMgr,
		logger:  logger,
		options: options,
	}, nil
}

// GetConnection 获取服务的 gRPC 连接
func (sd *etcdServiceDiscovery) GetConnection(ctx context.Context, serviceName string, options ...DiscoveryOption) (*grpc.ClientConn, error) {
	sd.logger.Infof("开始获取服务连接: %s", serviceName)

	opts := &DiscoveryOptions{
		LoadBalancer: "round_robin",
		Timeout:      5 * time.Second,
	}

	// 应用选项
	for _, opt := range options {
		opt(opts)
	}

	sd.logger.Debugf("连接选项: LoadBalancer=%s, Timeout=%v", opts.LoadBalancer, opts.Timeout)

	// 获取服务端点
	sd.logger.Debugf("获取服务端点: %s", serviceName)
	endpoints, err := sd.GetServiceEndpoints(ctx, serviceName)
	if err != nil {
		sd.logger.Errorf("获取服务端点失败: %v, service: %s", err, serviceName)
		return nil, WrapDiscoveryError(err, "failed to get service endpoints")
	}

	if len(endpoints) == 0 {
		sd.logger.Warnf("服务无可用实例: %s", serviceName)
		return nil, WrapDiscoveryError(ErrNoAvailableInstances, "no available instances for service: "+serviceName)
	}

	sd.logger.Infof("发现 %d 个服务端点: %v", len(endpoints), endpoints)

	// 创建 gRPC 连接
	// 简单实现：使用第一个可用的端点
	target := endpoints[0]
	sd.logger.Infof("尝试连接到主端点: %s", target)

	dialCtx, cancel := context.WithTimeout(ctx, opts.Timeout)
	defer cancel()

	conn, err := grpc.DialContext(dialCtx, target,
		grpc.WithTransportCredentials(insecure.NewCredentials()),
		grpc.WithBlock(),
	)
	if err != nil {
		sd.logger.Warnf("连接主端点失败: %v, target: %s", err, target)
		// 如果第一个端点失败，尝试其他端点
		sd.logger.Infof("尝试连接其他端点，共 %d 个备选", len(endpoints)-1)
		for i := 1; i < len(endpoints); i++ {
			target = endpoints[i]
			sd.logger.Debugf("尝试连接备选端点 %d: %s", i, target)

			dialCtx, cancel := context.WithTimeout(ctx, opts.Timeout)
			conn, err = grpc.DialContext(dialCtx, target,
				grpc.WithTransportCredentials(insecure.NewCredentials()),
				grpc.WithBlock(),
			)
			cancel()
			if err == nil {
				sd.logger.Infof("成功连接到备选端点: %s", target)
				break
			}
			sd.logger.Warnf("连接备选端点失败: %v, target: %s", err, target)
		}

		if err != nil {
			sd.logger.Errorf("连接所有端点都失败: %v, service: %s", err, serviceName)
			return nil, WrapDiscoveryError(err, "failed to connect to any service instance")
		}
	} else {
		sd.logger.Infof("成功连接到主端点: %s", target)
	}

	sd.logger.Infof("gRPC 连接创建成功，服务: %s, 端点: %s", serviceName, target)
	return conn, nil
}

// GetServiceEndpoints 获取服务的所有端点
func (sd *etcdServiceDiscovery) GetServiceEndpoints(ctx context.Context, serviceName string) ([]string, error) {
	sd.logger.Debugf("获取服务端点: %s", serviceName)

	client := sd.connMgr.GetClient()
	if client == nil {
		sd.logger.Error("etcd 客户端为空")
		return nil, ErrNotConnected
	}

	// 构建服务前缀
	prefix := fmt.Sprintf("%s/%s/", sd.options.ServicePrefix, serviceName)
	sd.logger.Debugf("查询前缀: %s", prefix)

	// 获取服务实例
	resp, err := client.Get(ctx, prefix, clientv3.WithPrefix())
	if err != nil {
		sd.logger.Errorf("从 etcd 获取服务实例失败: %v, prefix: %s", err, prefix)
		return nil, WrapDiscoveryError(err, "failed to get service instances from etcd")
	}

	sd.logger.Debugf("从 etcd 获取到 %d 个键值对", len(resp.Kvs))

	var endpoints []string
	for _, kv := range resp.Kvs {
		sd.logger.Debugf("解析服务实例: key=%s, value=%s", string(kv.Key), string(kv.Value))

		// 解析服务信息
		var serviceInstance struct {
			Address  string            `json:"address"`
			Metadata map[string]string `json:"metadata,omitempty"`
		}

		if err := json.Unmarshal(kv.Value, &serviceInstance); err != nil {
			sd.logger.Warnf("解析服务实例失败: %v, key: %s, value: %s", err, string(kv.Key), string(kv.Value))
			continue
		}

		sd.logger.Debugf("解析到服务端点: %s", serviceInstance.Address)
		endpoints = append(endpoints, serviceInstance.Address)
	}

	if len(endpoints) == 0 {
		sd.logger.Warnf("未找到服务: %s", serviceName)
		return nil, WrapDiscoveryError(ErrServiceNotFound, "service not found: "+serviceName)
	}

	sd.logger.Infof("获取服务端点成功: %s, 端点数量: %d, 端点列表: %v", serviceName, len(endpoints), endpoints)
	return endpoints, nil
}

// WatchService 监听服务变化
func (sd *etcdServiceDiscovery) WatchService(ctx context.Context, serviceName string) (<-chan ServiceEvent, error) {
	sd.logger.Infof("开始监听服务变化: %s", serviceName)

	client := sd.connMgr.GetClient()
	if client == nil {
		sd.logger.Error("etcd 客户端为空")
		return nil, ErrNotConnected
	}

	eventCh := make(chan ServiceEvent, 10)
	prefix := fmt.Sprintf("%s/%s/", sd.options.ServicePrefix, serviceName)
	sd.logger.Debugf("监听前缀: %s", prefix)

	go func() {
		defer func() {
			close(eventCh)
			sd.logger.Infof("服务监听协程结束: %s", serviceName)
		}()

		sd.logger.Debugf("启动 etcd watch: %s", prefix)
		watchCh := client.Watch(ctx, prefix, clientv3.WithPrefix())
		for watchResp := range watchCh {
			if watchResp.Err() != nil {
				sd.logger.Errorf("监听错误: %v, service: %s", watchResp.Err(), serviceName)
				return
			}

			sd.logger.Debugf("收到 %d 个监听事件, service: %s", len(watchResp.Events), serviceName)
			for _, event := range watchResp.Events {
				serviceEvent := sd.parseWatchEvent(event, serviceName)
				if serviceEvent != nil {
					sd.logger.Infof("服务事件: type=%d, service=%s, instance=%s",
						serviceEvent.Type, serviceEvent.Service, serviceEvent.Instance.ID)
					select {
					case eventCh <- *serviceEvent:
					case <-ctx.Done():
						sd.logger.Debug("监听上下文取消")
						return
					}
				}
			}
		}
	}()

	sd.logger.Infof("服务监听启动成功: %s", serviceName)
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
