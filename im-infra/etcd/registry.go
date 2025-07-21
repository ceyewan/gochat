package etcd

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/ceyewan/gochat/im-infra/clog"
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
	etcdLogger := clog.Module("etcd")
	etcdLogger.Info("创建服务注册实例")

	if client == nil {
		etcdLogger.Info("客户端为空，使用默认客户端")
		var err error
		client, err = GetDefaultClient()
		if err != nil {
			etcdLogger.Error("获取默认客户端失败", clog.Err(err))
			return nil, err
		}
	}

	etcdLogger.Info("服务注册实例创建成功", clog.String("service_prefix", "/services"))
	return &EtcdRegistry{
		client:        client,
		servicePrefix: "/services",
		logger:        NewClogAdapter(etcdLogger),
	}, nil
}

// NewServiceRegistryWithManager 使用管理器创建服务注册实例
func NewServiceRegistryWithManager(connMgr ConnectionManager, leaseMgr LeaseManager, logger Logger, options *ManagerOptions) (ServiceRegistry, error) {
	etcdLogger := clog.Module("etcd")
	etcdLogger.Info("使用管理器创建服务注册实例")

	if connMgr == nil {
		etcdLogger.Error("连接管理器为空")
		return nil, WrapRegistryError(ErrInvalidConfiguration, "connection manager is required")
	}
	if leaseMgr == nil {
		etcdLogger.Error("租约管理器为空")
		return nil, WrapRegistryError(ErrInvalidConfiguration, "lease manager is required")
	}
	if logger == nil {
		etcdLogger.Info("日志器为空，使用默认日志器")
		logger = NewClogAdapter(etcdLogger)
	}
	if options == nil {
		etcdLogger.Info("选项为空，使用默认选项")
		options = DefaultManagerOptions()
	}

	etcdLogger.Info("服务注册实例创建成功",
		clog.String("service_prefix", options.ServicePrefix),
		clog.Int64("default_ttl", options.DefaultTTL))

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
	r.logger.Infof("开始注册服务: %s/%s at %s", serviceName, instanceID, addr)

	// 解析选项
	opts := &RegisterOptions{
		TTL: r.getDefaultTTL(),
	}
	for _, opt := range options {
		opt(opts)
	}

	r.logger.Infof("服务注册选项: TTL=%d, LeaseID=%d, Metadata=%v",
		opts.TTL, opts.LeaseID, opts.Metadata)

	var client *clientv3.Client
	var leaseID clientv3.LeaseID
	var err error

	// 使用新的管理器或旧的客户端
	if r.connMgr != nil && r.leaseMgr != nil {
		// 新的管理器方式
		r.logger.Debug("使用连接管理器方式注册服务")
		if !r.connMgr.IsConnected() {
			r.logger.Error("etcd 连接未建立")
			return WrapRegistryError(ErrNotConnected, "not connected to etcd")
		}
		client = r.connMgr.GetClient()
		r.logger.Debug("获取 etcd 客户端成功")

		// 创建或使用现有租约
		if opts.LeaseID != 0 {
			leaseID = opts.LeaseID
			r.logger.Infof("使用现有租约: %d", leaseID)
		} else {
			r.logger.Infof("创建新租约，TTL: %d 秒", opts.TTL)
			leaseID, err = r.leaseMgr.CreateLease(ctx, opts.TTL)
			if err != nil {
				r.logger.Errorf("创建租约失败: %v", err)
				return WrapRegistryError(err, "failed to create lease")
			}
			r.logger.Infof("租约创建成功: %d", leaseID)

			// 启动租约保活
			r.logger.Debug("启动租约保活")
			_, err = r.leaseMgr.KeepAlive(ctx, leaseID)
			if err != nil {
				r.logger.Errorf("启动租约保活失败: %v", err)
				r.leaseMgr.RevokeLease(ctx, leaseID)
				return WrapRegistryError(err, "failed to start lease keepalive")
			}
			r.logger.Debug("租约保活启动成功")
		}
	} else {
		// 旧的客户端方式（向后兼容）
		r.logger.Debug("使用旧客户端方式注册服务")
		client = r.client.client

		r.logger.Infof("创建租约，TTL: %d 秒", opts.TTL)
		lease, err := client.Grant(ctx, opts.TTL)
		if err != nil {
			r.logger.Errorf("创建租约失败: %v", err)
			return fmt.Errorf("failed to create lease: %w", err)
		}
		leaseID = lease.ID
		r.logger.Infof("租约创建成功: %d", leaseID)

		// 设置保持活动
		r.logger.Debug("设置租约保活")
		keepAliveCh, err := client.KeepAlive(ctx, leaseID)
		if err != nil {
			r.logger.Errorf("设置租约保活失败: %v", err)
			return fmt.Errorf("failed to setup keepalive: %w", err)
		}

		// 处理keepalive响应
		go func() {
			r.logger.Debug("启动租约保活协程")
			for {
				select {
				case resp, ok := <-keepAliveCh:
					if !ok {
						r.logger.Warnf("租约保活通道关闭，服务: %s/%s", serviceName, instanceID)
						return
					}
					r.logger.Debugf("租约续期成功，服务: %s/%s, TTL: %d", serviceName, instanceID, resp.TTL)
				case <-ctx.Done():
					r.logger.Infof("服务注册上下文取消: %s/%s", serviceName, instanceID)
					return
				}
			}
		}()
	}

	// 构建服务键和值
	key := fmt.Sprintf("%s/%s/%s", r.servicePrefix, serviceName, instanceID)
	value := r.buildServiceValue(addr, opts.Metadata)

	r.logger.Infof("准备写入服务信息到 etcd, key: %s, value: %s", key, value)

	// 写入服务信息
	_, err = client.Put(ctx, key, value, clientv3.WithLease(leaseID))
	if err != nil {
		r.logger.Errorf("写入服务信息失败: %v, key: %s", err, key)
		if r.leaseMgr != nil {
			r.logger.Debug("撤销租约")
			r.leaseMgr.RevokeLease(ctx, leaseID)
		}
		return WrapRegistryError(err, "failed to register service")
	}

	r.logger.Infof("服务注册成功: %s/%s at %s, lease: %d", serviceName, instanceID, addr, leaseID)
	return nil
}

// Deregister 注销服务
func (r *EtcdRegistry) Deregister(ctx context.Context, serviceName, instanceID string) error {
	r.logger.Infof("开始注销服务: %s/%s", serviceName, instanceID)

	key := fmt.Sprintf("%s/%s/%s", r.servicePrefix, serviceName, instanceID)
	r.logger.Debugf("服务注销 key: %s", key)

	var client *clientv3.Client
	if r.connMgr != nil {
		r.logger.Debug("使用连接管理器获取客户端")
		if !r.connMgr.IsConnected() {
			r.logger.Error("etcd 连接未建立")
			return WrapRegistryError(ErrNotConnected, "not connected to etcd")
		}
		client = r.connMgr.GetClient()
	} else {
		r.logger.Debug("使用旧客户端方式")
		client = r.client.client
	}

	r.logger.Debugf("准备删除服务键: %s", key)
	_, err := client.Delete(ctx, key)
	if err != nil {
		r.logger.Errorf("删除服务失败: %v, key: %s", err, key)
		return WrapRegistryError(err, "failed to deregister service")
	}

	r.logger.Infof("服务注销成功: %s/%s", serviceName, instanceID)
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
