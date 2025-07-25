package registryimpl

import (
	"context"
	"encoding/json"
	"github.com/ceyewan/gochat/im-infra/coord/registry"
	"path"
	"strings"
	"time"

	"github.com/ceyewan/gochat/im-infra/clog"
	"github.com/ceyewan/gochat/im-infra/coord/internal/client"
	clientv3 "go.etcd.io/etcd/client/v3"
)

// EtcdServiceRegistry 基于 etcd 的服务注册发现实现
type EtcdServiceRegistry struct {
	client *client.EtcdClient
	prefix string
	logger clog.Logger
}

// NewEtcdServiceRegistry 创建新的服务注册发现实例
func NewEtcdServiceRegistry(client *client.EtcdClient, prefix string) *EtcdServiceRegistry {
	if prefix == "" {
		prefix = "/services"
	}

	return &EtcdServiceRegistry{
		client: client,
		prefix: prefix,
		logger: clog.Module("coordination.registry"),
	}
}

// Register 注册服务
func (r *EtcdServiceRegistry) Register(ctx context.Context, service registry.ServiceInfo) error {
	// 注意：内部仍然可以使用一个更详细的结构体，但对外的接口必须是 registry.ServiceInfo
	internalService := toInternalServiceInfo(service)

	if err := r.validateServiceInfo(internalService); err != nil {
		return err
	}

	serviceKey := r.buildServiceKey(service.Name, service.ID)

	r.logger.Info("registering service",
		clog.String("service_name", service.Name),
		clog.String("service_id", service.ID),
		clog.String("address", service.Address),
		clog.Int("port", service.Port),
		clog.Duration("ttl", service.TTL))

	// 序列化服务信息
	serviceData, err := json.Marshal(internalService)
	if err != nil {
		r.logger.Error("failed to serialize service info",
			clog.String("service_name", service.Name),
			clog.String("service_id", service.ID),
			clog.Err(err))
		return client.NewCoordinationError(
			client.ErrCodeValidation,
			"failed to serialize service info",
			err,
		)
	}

	// 创建租约
	leaseResp, err := r.client.Grant(ctx, int64(service.TTL.Seconds()))
	if err != nil {
		r.logger.Error("failed to create lease for service",
			clog.String("service_name", service.Name),
			clog.String("service_id", service.ID),
			clog.Err(err))
		return err
	}

	// 注册服务（带租约）
	_, err = r.client.Put(ctx, serviceKey, string(serviceData), clientv3.WithLease(leaseResp.ID))
	if err != nil {
		// 如果注册失败，撤销租约
		r.client.Revoke(context.Background(), leaseResp.ID)
		r.logger.Error("failed to register service",
			clog.String("service_name", service.Name),
			clog.String("service_id", service.ID),
			clog.Err(err))
		return err
	}

	// 启动租约续期
	keepAliveCh, err := r.client.KeepAlive(ctx, leaseResp.ID)
	if err != nil {
		r.client.Revoke(context.Background(), leaseResp.ID)
		r.logger.Error("failed to start lease keep alive",
			clog.String("service_name", service.Name),
			clog.String("service_id", service.ID),
			clog.Err(err))
		return err
	}

	// 启动后台 goroutine 处理 keep alive 响应
	go r.handleKeepAlive(service.Name, service.ID, leaseResp.ID, keepAliveCh)

	r.logger.Info("service registered successfully",
		clog.String("service_name", service.Name),
		clog.String("service_id", service.ID),
		clog.Int64("lease_id", int64(leaseResp.ID)))

	return nil
}

// Unregister 注销服务
func (r *EtcdServiceRegistry) Unregister(ctx context.Context, serviceID string) error {
	if serviceID == "" {
		return client.NewCoordinationError(
			client.ErrCodeValidation,
			"service ID cannot be empty",
			nil,
		)
	}

	r.logger.Info("unregistering service",
		clog.String("service_id", serviceID))

	// 查找服务键
	serviceKey, err := r.findServiceKey(ctx, serviceID)
	if err != nil {
		return err
	}

	if serviceKey == "" {
		r.logger.Debug("service not found for unregistration",
			clog.String("service_id", serviceID))
		return client.NewCoordinationError(
			client.ErrCodeNotFound,
			"service not found",
			nil,
		)
	}

	// 删除服务
	resp, err := r.client.Delete(ctx, serviceKey)
	if err != nil {
		r.logger.Error("failed to unregister service",
			clog.String("service_id", serviceID),
			clog.String("service_key", serviceKey),
			clog.Err(err))
		return err
	}

	if resp.Deleted == 0 {
		r.logger.Debug("service key not found for deletion",
			clog.String("service_id", serviceID),
			clog.String("service_key", serviceKey))
		return client.NewCoordinationError(
			client.ErrCodeNotFound,
			"service not found",
			nil,
		)
	}

	r.logger.Info("service unregistered successfully",
		clog.String("service_id", serviceID))

	return nil
}

// Discover 发现服务
func (r *EtcdServiceRegistry) Discover(ctx context.Context, serviceName string) ([]registry.ServiceInfo, error) {
	if serviceName == "" {
		return nil, client.NewCoordinationError(
			client.ErrCodeValidation,
			"service name cannot be empty",
			nil,
		)
	}

	servicePrefix := r.buildServicePrefix(serviceName)

	r.logger.Info("discovering services",
		clog.String("service_name", serviceName),
		clog.String("prefix", servicePrefix))

	resp, err := r.client.Get(ctx, servicePrefix, clientv3.WithPrefix())
	if err != nil {
		r.logger.Error("failed to discover services",
			clog.String("service_name", serviceName),
			clog.Err(err))
		return nil, err
	}

	var services []registry.ServiceInfo
	for _, kv := range resp.Kvs {
		var service internalServiceInfo
		if err := json.Unmarshal(kv.Value, &service); err != nil {
			r.logger.Warn("failed to unmarshal service info, skipping",
				clog.String("key", string(kv.Key)),
				clog.Err(err))
			continue
		}
		services = append(services, toPublicServiceInfo(service))
	}

	r.logger.Info("services discovered successfully",
		clog.String("service_name", serviceName),
		clog.Int("count", len(services)))

	return services, nil
}

// Watch 监听服务变化
func (r *EtcdServiceRegistry) Watch(ctx context.Context, serviceName string) (<-chan registry.ServiceEvent, error) {
	if serviceName == "" {
		return nil, client.NewCoordinationError(
			client.ErrCodeValidation,
			"service name cannot be empty",
			nil,
		)
	}

	servicePrefix := r.buildServicePrefix(serviceName)

	r.logger.Info("starting to watch service changes",
		clog.String("service_name", serviceName),
		clog.String("prefix", servicePrefix))

	watchCh := r.client.Watch(ctx, servicePrefix, clientv3.WithPrefix())
	eventCh := make(chan registry.ServiceEvent, 10)

	go func() {
		defer close(eventCh)
		defer r.logger.Info("service watch stopped",
			clog.String("service_name", serviceName))

		for resp := range watchCh {
			if resp.Err() != nil {
				r.logger.Error("watch error occurred",
					clog.String("service_name", serviceName),
					clog.Err(resp.Err()))
				continue
			}

			for _, event := range resp.Events {
				serviceEvent := r.convertEvent(event)
				if serviceEvent != nil {
					r.logger.Info("service change detected",
						clog.String("service_name", serviceName),
						clog.String("service_id", serviceEvent.Service.ID),
						clog.String("type", string(serviceEvent.Type)))

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

// validateServiceInfo 验证服务信息
func (r *EtcdServiceRegistry) validateServiceInfo(service internalServiceInfo) error {
	if service.ID == "" {
		return client.NewCoordinationError(
			client.ErrCodeValidation,
			"service ID cannot be empty",
			nil,
		)
	}

	if service.Name == "" {
		return client.NewCoordinationError(
			client.ErrCodeValidation,
			"service name cannot be empty",
			nil,
		)
	}

	if service.Address == "" {
		return client.NewCoordinationError(
			client.ErrCodeValidation,
			"service address cannot be empty",
			nil,
		)
	}

	if service.Port <= 0 || service.Port > 65535 {
		return client.NewCoordinationError(
			client.ErrCodeValidation,
			"service port must be between 1 and 65535",
			nil,
		)
	}

	if service.TTL <= 0 {
		return client.NewCoordinationError(
			client.ErrCodeValidation,
			"service TTL must be positive",
			nil,
		)
	}

	return nil
}

// buildServiceKey 构建服务键
func (r *EtcdServiceRegistry) buildServiceKey(serviceName, serviceID string) string {
	return path.Join(r.prefix, serviceName, serviceID)
}

// buildServicePrefix 构建服务前缀
func (r *EtcdServiceRegistry) buildServicePrefix(serviceName string) string {
	return path.Join(r.prefix, serviceName) + "/"
}

// findServiceKey 查找服务键
func (r *EtcdServiceRegistry) findServiceKey(ctx context.Context, serviceID string) (string, error) {
	// 搜索所有服务
	resp, err := r.client.Get(ctx, r.prefix+"/", clientv3.WithPrefix())
	if err != nil {
		return "", err
	}

	for _, kv := range resp.Kvs {
		var service internalServiceInfo
		if err := json.Unmarshal(kv.Value, &service); err != nil {
			continue
		}

		if service.ID == serviceID {
			return string(kv.Key), nil
		}
	}

	return "", nil
}

// convertEvent 转换 etcd 事件为服务事件
func (r *EtcdServiceRegistry) convertEvent(event *clientv3.Event) *registry.ServiceEvent {
	key := string(event.Kv.Key)

	// 检查是否为服务键
	if !strings.HasPrefix(key, r.prefix+"/") {
		return nil
	}

	var eventType registry.EventType
	var service internalServiceInfo

	switch event.Type {
	case clientv3.EventTypePut:
		eventType = registry.EventTypePut
		if err := json.Unmarshal(event.Kv.Value, &service); err != nil {
			r.logger.Warn("failed to unmarshal service info in event",
				clog.String("key", key),
				clog.Err(err))
			return nil
		}
	case clientv3.EventTypeDelete:
		eventType = registry.EventTypeDelete
		// 对于删除事件，尝试从键中提取服务信息
		parts := strings.Split(strings.TrimPrefix(key, r.prefix+"/"), "/")
		if len(parts) >= 2 {
			service.Name = parts[0]
			service.ID = parts[1]
		}
	default:
		return nil
	}

	return &registry.ServiceEvent{
		Type:    eventType,
		Service: toPublicServiceInfo(service),
	}
}

// internalServiceInfo 包含内部实现细节（如TTL）的服务信息
type internalServiceInfo struct {
	ID       string            `json:"id"`
	Name     string            `json:"name"`
	Address  string            `json:"address"`
	Metadata map[string]string `json:"metadata,omitempty"`
	TTL      time.Duration     `json:"ttl"`
	Port     int               `json:"port"` // 假设 Port 也是内部细节
}

// toInternalServiceInfo 将公共API结构转换为内部结构
func toInternalServiceInfo(s registry.ServiceInfo) internalServiceInfo {
	// 默认 TTL
	ttl := 30 * time.Second
	return internalServiceInfo{
		ID:       s.ID,
		Name:     s.Name,
		Address:  s.Address,
		Metadata: s.Metadata,
		TTL:      ttl, // 在这里处理默认值或从元数据解析
	}
}

// toPublicServiceInfo 将内部结构转换为公共API结构
func toPublicServiceInfo(s internalServiceInfo) registry.ServiceInfo {
	return registry.ServiceInfo{
		ID:       s.ID,
		Name:     s.Name,
		Address:  s.Address,
		Metadata: s.Metadata,
	}
}

// handleKeepAlive 处理租约续期响应
func (r *EtcdServiceRegistry) handleKeepAlive(serviceName, serviceID string, leaseID clientv3.LeaseID, keepAliveCh <-chan *clientv3.LeaseKeepAliveResponse) {
	for resp := range keepAliveCh {
		if resp == nil {
			r.logger.Warn("service keep alive channel closed",
				clog.String("service_name", serviceName),
				clog.String("service_id", serviceID),
				clog.Int64("lease_id", int64(leaseID)))
			break
		}

		r.logger.Debug("service lease keep alive response received",
			clog.String("service_name", serviceName),
			clog.String("service_id", serviceID),
			clog.Int64("lease_id", int64(leaseID)),
			clog.Int64("ttl", resp.TTL))
	}
}
