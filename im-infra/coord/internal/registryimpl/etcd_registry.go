package registryimpl

import (
	"context"
	"encoding/json"
	"fmt"
	"path"
	"strings"
	"sync"
	"time"

	"github.com/ceyewan/gochat/im-infra/clog"
	"github.com/ceyewan/gochat/im-infra/coord/internal/client"
	"github.com/ceyewan/gochat/im-infra/coord/registry"
	clientv3 "go.etcd.io/etcd/client/v3"
	"go.etcd.io/etcd/client/v3/concurrency"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/resolver"
)

// EtcdServiceRegistry 使用 etcd 实现 registry.ServiceRegistry 接口
type EtcdServiceRegistry struct {
	client *client.EtcdClient // etcd 客户端
	prefix string             // 服务注册前缀
	logger clog.Logger        // 日志记录器

	// 跟踪当前实例注册的服务会话
	sessions   map[string]*concurrency.Session // 服务会话映射，便于注销
	sessionsMu sync.Mutex                      // 会话互斥锁

	// gRPC resolver builder（只注册一次）
	resolverBuilder *EtcdResolverBuilder // gRPC 解析器构建器
	resolverOnce    sync.Once            // 只注册一次
}

// NewEtcdServiceRegistry 创建一个基于 etcd 的服务注册表
func NewEtcdServiceRegistry(c *client.EtcdClient, prefix string, logger clog.Logger) *EtcdServiceRegistry {
	if prefix == "" {
		prefix = "/services"
	}
	if logger == nil {
		logger = clog.Module("coordination.registry")
	}

	registry := &EtcdServiceRegistry{
		client:   c,
		prefix:   prefix,
		logger:   logger,
		sessions: make(map[string]*concurrency.Session),
	}

	// 创建 resolver builder
	registry.resolverBuilder = NewEtcdResolverBuilder(c, prefix, logger)

	// 注册 gRPC resolver（只注册一次）
	registry.resolverOnce.Do(func() {
		resolver.Register(registry.resolverBuilder)
		logger.Info("gRPC etcd resolver registered", clog.String("scheme", EtcdScheme))
	})

	return registry
}

// Register 注册服务，ttl 是租约的有效期，服务会被持续保持直到 context 被取消或 Unregister 被调用
func (r *EtcdServiceRegistry) Register(ctx context.Context, service registry.ServiceInfo, ttl time.Duration) error {
	if err := validateServiceInfo(service); err != nil {
		return err
	}
	if ttl <= 0 {
		return client.NewError(client.ErrCodeValidation, "service TTL must be positive", nil)
	}

	// 使用会话管理租约并自动续约
	session, err := concurrency.NewSession(r.client.Client(), concurrency.WithTTL(int(ttl.Seconds())))
	if err != nil {
		return client.NewError(client.ErrCodeConnection, "failed to create etcd session", err)
	}

	serviceKey := r.buildServiceKey(service.Name, service.ID)
	serviceData, err := json.Marshal(service)
	if err != nil {
		_ = session.Close() // 尝试关闭会话，释放资源
		return client.NewError(client.ErrCodeValidation, "failed to serialize service info", err)
	}

	// 使用会话的租约注册服务
	_, err = r.client.Put(ctx, serviceKey, string(serviceData), clientv3.WithLease(session.Lease()))
	if err != nil {
		_ = session.Close() // 尝试关闭会话，释放资源
		return client.NewError(client.ErrCodeConnection, "failed to register service", err)
	}

	r.logger.Info("Service registered successfully",
		clog.String("service_name", service.Name),
		clog.String("service_id", service.ID),
		clog.Int64("lease_id", int64(session.Lease())))

	// 存储会话以便清理注销
	r.sessionsMu.Lock()
	r.sessions[service.ID] = session
	r.sessionsMu.Unlock()

	// 会话的 keep-alive 在后台运行，可通过 Done 通道监控会话过期
	go func() {
		<-session.Done()
		r.sessionsMu.Lock()
		delete(r.sessions, service.ID)
		r.sessionsMu.Unlock()
		r.logger.Warn("服务会话已过期或关闭",
			clog.String("service_name", service.Name),
			clog.String("service_id", service.ID))
	}()

	return nil
}

// Unregister 注销服务，优先关闭会话，找不到会话则直接删除 key
func (r *EtcdServiceRegistry) Unregister(ctx context.Context, serviceID string) error {
	if serviceID == "" {
		return client.NewError(client.ErrCodeValidation, "service ID cannot be empty", nil)
	}

	r.sessionsMu.Lock()
	session, ok := r.sessions[serviceID]
	r.sessionsMu.Unlock()

	// 如果本地有会话，关闭会话最干净
	if ok {
		r.logger.Info("通过关闭会话注销服务", clog.String("service_id", serviceID))
		delete(r.sessions, serviceID)
		if err := session.Close(); err != nil {
			return client.NewError(client.ErrCodeConnection, "注销服务时关闭会话失败", err)
		}
		return nil
	}

	// 如果服务由其他实例注册，则直接删除 key
	r.logger.Warn("本地未找到会话，通过删除 key 注销服务", clog.String("service_id", serviceID))
	key, err := r.findServiceKey(ctx, serviceID)
	if err != nil {
		return err
	}
	if key == "" {
		return client.NewError(client.ErrCodeNotFound, "service not found", nil)
	}

	_, err = r.client.Delete(ctx, key)
	if err != nil {
		return client.NewError(client.ErrCodeConnection, "failed to delete service key", err)
	}

	return nil
}

// Discover 查询指定服务的所有实例
func (r *EtcdServiceRegistry) Discover(ctx context.Context, serviceName string) ([]registry.ServiceInfo, error) {
	if serviceName == "" {
		return nil, client.NewError(client.ErrCodeValidation, "服务名不能为空", nil)
	}

	prefix := r.buildServicePrefix(serviceName)
	resp, err := r.client.Get(ctx, prefix, clientv3.WithPrefix())
	if err != nil {
		return nil, client.NewError(client.ErrCodeConnection, "failed to discover services", err)
	}

	services := make([]registry.ServiceInfo, 0, len(resp.Kvs))
	for _, kv := range resp.Kvs {
		var service registry.ServiceInfo
		if err := json.Unmarshal(kv.Value, &service); err != nil {
			r.logger.Warn("Failed to unmarshal service info, skipping",
				clog.String("key", string(kv.Key)),
				clog.Err(err))
			continue
		}
		services = append(services, service)
	}

	return services, nil
}

// Watch 监听服务变更事件
func (r *EtcdServiceRegistry) Watch(ctx context.Context, serviceName string) (<-chan registry.ServiceEvent, error) {
	if serviceName == "" {
		return nil, client.NewError(client.ErrCodeValidation, "service name cannot be empty", nil)
	}

	prefix := r.buildServicePrefix(serviceName)
	etcdWatchCh := r.client.Watch(ctx, prefix, clientv3.WithPrefix())
	eventCh := make(chan registry.ServiceEvent, 10)

	go func() {
		defer close(eventCh)
		for resp := range etcdWatchCh {
			if err := resp.Err(); err != nil {
				r.logger.Error("监听服务发生错误", clog.String("service_name", serviceName), clog.Err(err))
				// 可选：向通道发送错误事件
				return
			}
			for _, event := range resp.Events {
				serviceEvent := r.convertEvent(event)
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

// buildServiceKey 构建服务实例的 etcd key
func (r *EtcdServiceRegistry) buildServiceKey(serviceName, serviceID string) string {
	return path.Join(r.prefix, serviceName, serviceID)
}

// buildServicePrefix 构建服务前缀
func (r *EtcdServiceRegistry) buildServicePrefix(serviceName string) string {
	return path.Join(r.prefix, serviceName) + "/"
}

// findServiceKey 查找指定 serviceID 的 etcd key
func (r *EtcdServiceRegistry) findServiceKey(ctx context.Context, serviceID string) (string, error) {
	resp, err := r.client.Get(ctx, r.prefix+"/", clientv3.WithPrefix())
	if err != nil {
		return "", client.NewError(client.ErrCodeConnection, "failed to search for service key", err)
	}
	for _, kv := range resp.Kvs {
		if strings.HasSuffix(string(kv.Key), "/"+serviceID) {
			return string(kv.Key), nil
		}
	}
	return "", nil
}

// convertEvent 将 etcd 事件转换为服务事件
func (r *EtcdServiceRegistry) convertEvent(event *clientv3.Event) *registry.ServiceEvent {
	var service registry.ServiceInfo
	var eventType registry.EventType

	switch event.Type {
	case clientv3.EventTypePut:
		eventType = registry.EventTypePut
		if err := json.Unmarshal(event.Kv.Value, &service); err != nil {
			r.logger.Warn("事件中服务信息解析失败", clog.String("key", string(event.Kv.Key)), clog.Err(err))
			return nil
		}
	case clientv3.EventTypeDelete:
		eventType = registry.EventTypeDelete
		// 删除事件无法获取完整服务信息，仅能从 key 解析 Name 和 ID
		parts := strings.Split(strings.TrimPrefix(string(event.Kv.Key), r.prefix+"/"), "/")
		if len(parts) >= 2 {
			service.Name = parts[0]
			service.ID = parts[1]
		}
	default:
		return nil
	}

	return &registry.ServiceEvent{
		Type:    eventType,
		Service: service,
	}
}

// validateServiceInfo 校验服务信息合法性
func validateServiceInfo(service registry.ServiceInfo) error {
	if service.ID == "" {
		return client.NewError(client.ErrCodeValidation, "服务 ID 不能为空", nil)
	}
	if service.Name == "" {
		return client.NewError(client.ErrCodeValidation, "服务名不能为空", nil)
	}
	if service.Address == "" {
		return client.NewError(client.ErrCodeValidation, "服务地址不能为空", nil)
	}
	if service.Port <= 0 || service.Port > 65535 {
		return client.NewError(client.ErrCodeValidation, "服务端口必须在 1~65535 之间", nil)
	}
	return nil
}

// GetConnection 获取到指定服务的 gRPC 连接，支持动态服务发现和负载均衡
func (r *EtcdServiceRegistry) GetConnection(ctx context.Context, serviceName string) (*grpc.ClientConn, error) {
	if serviceName == "" {
		return nil, client.NewError(client.ErrCodeValidation, "服务名不能为空", nil)
	}

	// 使用 etcd resolver 创建连接
	// target 格式: etcd:///<service-name>
	target := fmt.Sprintf("%s:///%s", EtcdScheme, serviceName)

	// 创建 gRPC 连接，使用 etcd resolver 进行动态服务发现
	conn, err := grpc.NewClient(target,
		grpc.WithTransportCredentials(insecure.NewCredentials()),
		grpc.WithDefaultServiceConfig(`{"loadBalancingPolicy":"round_robin"}`), // 使用轮询负载均衡
	)
	if err != nil {
		return nil, client.NewError(client.ErrCodeConnection, "连接服务失败", err)
	}

	r.logger.Info("已建立 gRPC 动态服务发现连接",
		clog.String("service_name", serviceName),
		clog.String("target", target))

	return conn, nil
}
