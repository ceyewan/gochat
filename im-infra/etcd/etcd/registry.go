package etcd

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"strings"

	clientv3 "go.etcd.io/etcd/client/v3"
)

// EtcdRegistry 实现基于etcd的服务注册
type EtcdRegistry struct {
	client        *Client
	connMgr       ConnectionManager
	leaseMgr      LeaseManager
	logger        Logger
	options       *ManagerOptions
	servicePrefix string
}

// NewServiceRegistry 创建服务注册实例（向后兼容）
func NewServiceRegistry(client *Client) (*EtcdRegistry, error) {
	if client == nil {
		var err error
		client, err = GetDefaultClient()
		if err != nil {
			return nil, err
		}
	}

	return &EtcdRegistry{
		client:        client,
		servicePrefix: "/services",
		logger:        &DefaultLogger{Logger: log.Default()},
	}, nil
}

// NewServiceRegistryWithManager 使用管理器创建服务注册实例
func NewServiceRegistryWithManager(connMgr ConnectionManager, leaseMgr LeaseManager, logger Logger, options *ManagerOptions) (ServiceRegistry, error) {
	if connMgr == nil {
		return nil, WrapRegistryError(ErrInvalidConfiguration, "connection manager is required")
	}
	if leaseMgr == nil {
		return nil, WrapRegistryError(ErrInvalidConfiguration, "lease manager is required")
	}
	if logger == nil {
		logger = &DefaultLogger{Logger: log.Default()}
	}
	if options == nil {
		options = DefaultManagerOptions()
	}

	return &EtcdRegistry{
		connMgr:       connMgr,
		leaseMgr:      leaseMgr,
		logger:        logger,
		options:       options,
		servicePrefix: options.ServicePrefix,
	}, nil
}

// Register 注册服务（实现新接口）
func (r *EtcdRegistry) Register(ctx context.Context, serviceName, instanceID, addr string, options ...RegisterOption) error {
	return r.RegisterWithOptions(ctx, serviceName, instanceID, addr, options...)
}

// RegisterLegacy 注册服务（向后兼容旧接口）
func (r *EtcdRegistry) RegisterLegacy(ctx context.Context, serviceName, instanceID, addr string) error {
	return r.Register(ctx, serviceName, instanceID, addr)
}

// RegisterWithOptions 使用选项注册服务
func (r *EtcdRegistry) RegisterWithOptions(ctx context.Context, serviceName, instanceID, addr string, options ...RegisterOption) error {
	// 解析选项
	opts := &RegisterOptions{
		TTL: r.getDefaultTTL(),
	}
	for _, opt := range options {
		opt(opts)
	}

	var client *clientv3.Client
	var leaseID clientv3.LeaseID
	var err error

	// 使用新的管理器或旧的客户端
	if r.connMgr != nil && r.leaseMgr != nil {
		// 新的管理器方式
		if !r.connMgr.IsConnected() {
			return WrapRegistryError(ErrNotConnected, "not connected to etcd")
		}
		client = r.connMgr.GetClient()

		// 创建或使用现有租约
		if opts.LeaseID != 0 {
			leaseID = opts.LeaseID
		} else {
			leaseID, err = r.leaseMgr.CreateLease(ctx, opts.TTL)
			if err != nil {
				return WrapRegistryError(err, "failed to create lease")
			}

			// 启动租约保活
			_, err = r.leaseMgr.KeepAlive(ctx, leaseID)
			if err != nil {
				r.leaseMgr.RevokeLease(ctx, leaseID)
				return WrapRegistryError(err, "failed to start lease keepalive")
			}
		}
	} else {
		// 旧的客户端方式（向后兼容）
		client = r.client.client
		lease, err := client.Grant(ctx, opts.TTL)
		if err != nil {
			return fmt.Errorf("failed to create lease: %w", err)
		}
		leaseID = lease.ID

		// 设置保持活动
		keepAliveCh, err := client.KeepAlive(ctx, leaseID)
		if err != nil {
			return fmt.Errorf("failed to setup keepalive: %w", err)
		}

		// 处理keepalive响应
		go func() {
			for {
				select {
				case resp, ok := <-keepAliveCh:
					if !ok {
						log.Printf("Keepalive channel closed for service %s/%s", serviceName, instanceID)
						return
					}
					log.Printf("Lease renewed for service %s/%s, TTL: %d", serviceName, instanceID, resp.TTL)
				case <-ctx.Done():
					log.Printf("Service registry context canceled for %s/%s", serviceName, instanceID)
					return
				}
			}
		}()
	}

	// 构建服务键和值
	key := fmt.Sprintf("%s/%s/%s", r.servicePrefix, serviceName, instanceID)
	value := r.buildServiceValue(addr, opts.Metadata)

	// 写入服务信息
	_, err = client.Put(ctx, key, value, clientv3.WithLease(leaseID))
	if err != nil {
		if r.leaseMgr != nil {
			r.leaseMgr.RevokeLease(ctx, leaseID)
		}
		return WrapRegistryError(err, "failed to register service")
	}

	r.logger.Infof("Service registered successfully: %s/%s at %s", serviceName, instanceID, addr)
	return nil
}

// Deregister 注销服务
func (r *EtcdRegistry) Deregister(ctx context.Context, serviceName, instanceID string) error {
	key := fmt.Sprintf("%s/%s/%s", r.servicePrefix, serviceName, instanceID)

	var client *clientv3.Client
	if r.connMgr != nil {
		if !r.connMgr.IsConnected() {
			return WrapRegistryError(ErrNotConnected, "not connected to etcd")
		}
		client = r.connMgr.GetClient()
	} else {
		client = r.client.client
	}

	_, err := client.Delete(ctx, key)
	if err != nil {
		return WrapRegistryError(err, "failed to deregister service")
	}

	r.logger.Infof("Service deregistered: %s/%s", serviceName, instanceID)
	return nil
}

// UpdateService 更新服务信息
func (r *EtcdRegistry) UpdateService(ctx context.Context, serviceName, instanceID, addr string) error {
	// 先注销再注册
	if err := r.Deregister(ctx, serviceName, instanceID); err != nil {
		r.logger.Warnf("Failed to deregister service during update: %v", err)
	}

	return r.Register(ctx, serviceName, instanceID, addr)
}

// ListServices 列出所有已注册的服务
func (r *EtcdRegistry) ListServices(ctx context.Context) ([]ServiceInfo, error) {
	var client *clientv3.Client
	if r.connMgr != nil {
		if !r.connMgr.IsConnected() {
			return nil, WrapRegistryError(ErrNotConnected, "not connected to etcd")
		}
		client = r.connMgr.GetClient()
	} else {
		client = r.client.client
	}

	resp, err := client.Get(ctx, r.servicePrefix+"/", clientv3.WithPrefix())
	if err != nil {
		return nil, WrapRegistryError(err, "failed to list services")
	}

	services := make(map[string]*ServiceInfo)

	for _, kv := range resp.Kvs {
		serviceName, instance, err := r.parseServiceKey(string(kv.Key))
		if err != nil {
			r.logger.Warnf("Failed to parse service key %s: %v", string(kv.Key), err)
			continue
		}

		addr, metadata, err := r.parseServiceValue(string(kv.Value))
		if err != nil {
			r.logger.Warnf("Failed to parse service value %s: %v", string(kv.Value), err)
			continue
		}

		if services[serviceName] == nil {
			services[serviceName] = &ServiceInfo{
				Name:      serviceName,
				Instances: []ServiceInstance{},
			}
		}

		services[serviceName].Instances = append(services[serviceName].Instances, ServiceInstance{
			ID:       instance,
			Address:  addr,
			Metadata: metadata,
		})
	}

	var result []ServiceInfo
	for _, service := range services {
		result = append(result, *service)
	}

	return result, nil
}

// GetServiceInstances 获取指定服务的所有实例
func (r *EtcdRegistry) GetServiceInstances(ctx context.Context, serviceName string) ([]ServiceInstance, error) {
	var client *clientv3.Client
	if r.connMgr != nil {
		if !r.connMgr.IsConnected() {
			return nil, WrapRegistryError(ErrNotConnected, "not connected to etcd")
		}
		client = r.connMgr.GetClient()
	} else {
		client = r.client.client
	}

	prefix := fmt.Sprintf("%s/%s/", r.servicePrefix, serviceName)
	resp, err := client.Get(ctx, prefix, clientv3.WithPrefix())
	if err != nil {
		return nil, WrapRegistryError(err, "failed to get service instances")
	}

	var instances []ServiceInstance
	for _, kv := range resp.Kvs {
		_, instanceID, err := r.parseServiceKey(string(kv.Key))
		if err != nil {
			r.logger.Warnf("Failed to parse service key %s: %v", string(kv.Key), err)
			continue
		}

		addr, metadata, err := r.parseServiceValue(string(kv.Value))
		if err != nil {
			r.logger.Warnf("Failed to parse service value %s: %v", string(kv.Value), err)
			continue
		}

		instances = append(instances, ServiceInstance{
			ID:       instanceID,
			Address:  addr,
			Metadata: metadata,
		})
	}

	return instances, nil
}

// 辅助方法

// getDefaultTTL 获取默认TTL
func (r *EtcdRegistry) getDefaultTTL() int64 {
	if r.options != nil && r.options.DefaultTTL > 0 {
		return r.options.DefaultTTL
	}
	return 30 // 默认30秒
}

// buildServiceValue 构建服务值
func (r *EtcdRegistry) buildServiceValue(addr string, metadata map[string]string) string {
	if len(metadata) == 0 {
		return addr
	}

	// 构建包含元数据的JSON值
	value := map[string]interface{}{
		"address":  addr,
		"metadata": metadata,
	}

	data, err := json.Marshal(value)
	if err != nil {
		r.logger.Warnf("Failed to marshal service value, using address only: %v", err)
		return addr
	}

	return string(data)
}

// parseServiceKey 解析服务键
func (r *EtcdRegistry) parseServiceKey(key string) (serviceName, instanceID string, err error) {
	// 移除前缀
	if !strings.HasPrefix(key, r.servicePrefix+"/") {
		return "", "", fmt.Errorf("invalid service key format: %s", key)
	}

	path := key[len(r.servicePrefix)+1:]
	parts := strings.Split(path, "/")

	if len(parts) != 2 {
		return "", "", fmt.Errorf("invalid service key format: %s", key)
	}

	return parts[0], parts[1], nil
}

// parseServiceValue 解析服务值
func (r *EtcdRegistry) parseServiceValue(value string) (addr string, metadata map[string]string, err error) {
	// 尝试解析为JSON
	var data map[string]interface{}
	if err := json.Unmarshal([]byte(value), &data); err != nil {
		// 如果不是JSON，则认为是纯地址
		return value, nil, nil
	}

	// 提取地址
	if addrVal, ok := data["address"]; ok {
		if addrStr, ok := addrVal.(string); ok {
			addr = addrStr
		} else {
			return "", nil, fmt.Errorf("invalid address format in service value")
		}
	} else {
		return "", nil, fmt.Errorf("missing address in service value")
	}

	// 提取元数据
	if metaVal, ok := data["metadata"]; ok {
		if metaMap, ok := metaVal.(map[string]interface{}); ok {
			metadata = make(map[string]string)
			for k, v := range metaMap {
				if vStr, ok := v.(string); ok {
					metadata[k] = vStr
				}
			}
		}
	}

	return addr, metadata, nil
}
