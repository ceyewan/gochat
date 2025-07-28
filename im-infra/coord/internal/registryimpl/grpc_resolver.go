package registryimpl

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/ceyewan/gochat/im-infra/clog"
	"github.com/ceyewan/gochat/im-infra/coord/internal/client"
	"github.com/ceyewan/gochat/im-infra/coord/registry"
	clientv3 "go.etcd.io/etcd/client/v3"
	"google.golang.org/grpc/resolver"
)

const (
	// EtcdScheme 是 etcd resolver 的 scheme
	EtcdScheme = "etcd"
)

// EtcdResolverBuilder 实现 gRPC resolver.Builder 接口
type EtcdResolverBuilder struct {
	client *client.EtcdClient
	prefix string
	logger clog.Logger
}

// NewEtcdResolverBuilder 创建新的 etcd resolver builder
func NewEtcdResolverBuilder(client *client.EtcdClient, prefix string, logger clog.Logger) *EtcdResolverBuilder {
	if prefix == "" {
		prefix = "/services"
	}
	if logger == nil {
		logger = clog.Module("coordination.resolver")
	}
	return &EtcdResolverBuilder{
		client: client,
		prefix: prefix,
		logger: logger,
	}
}

// Build 创建并返回新的 resolver
func (b *EtcdResolverBuilder) Build(target resolver.Target, cc resolver.ClientConn, opts resolver.BuildOptions) (resolver.Resolver, error) {
	serviceName := target.Endpoint()
	if serviceName == "" {
		return nil, fmt.Errorf("service name cannot be empty")
	}

	r := &EtcdResolver{
		client:      b.client,
		prefix:      b.prefix,
		serviceName: serviceName,
		cc:          cc,
		logger:      b.logger,
		ctx:         context.Background(),
		cancel:      nil,
		closed:      make(chan struct{}),
	}

	r.ctx, r.cancel = context.WithCancel(r.ctx)

	// 启动 resolver
	go r.start()

	return r, nil
}

// Scheme 返回 resolver 的 scheme
func (b *EtcdResolverBuilder) Scheme() string {
	return EtcdScheme
}

// EtcdResolver 实现 gRPC resolver.Resolver 接口
type EtcdResolver struct {
	client      *client.EtcdClient
	prefix      string
	serviceName string
	cc          resolver.ClientConn
	logger      clog.Logger

	ctx    context.Context
	cancel context.CancelFunc
	closed chan struct{}

	mu        sync.RWMutex
	addresses []resolver.Address
}

// start 启动 resolver，开始监听服务变化
func (r *EtcdResolver) start() {
	defer close(r.closed)

	// 首次解析服务地址
	if err := r.resolveNow(); err != nil {
		r.logger.Error("Initial service resolution failed",
			clog.String("service", r.serviceName),
			clog.Err(err))
		r.cc.ReportError(err)
		return
	}

	// 开始监听服务变化
	r.watch()
}

// resolveNow 立即解析服务地址
func (r *EtcdResolver) resolveNow() error {
	prefix := r.buildServicePrefix(r.serviceName)
	resp, err := r.client.Get(r.ctx, prefix, clientv3.WithPrefix())
	if err != nil {
		return client.NewError(client.ErrCodeConnection, "failed to discover services", err)
	}

	var addresses []resolver.Address
	for _, kv := range resp.Kvs {
		var service registry.ServiceInfo
		if err := json.Unmarshal(kv.Value, &service); err != nil {
			r.logger.Warn("Failed to unmarshal service info",
				clog.String("key", string(kv.Key)),
				clog.Err(err))
			continue
		}

		addr := resolver.Address{
			Addr: fmt.Sprintf("%s:%d", service.Address, service.Port),
		}
		addresses = append(addresses, addr)
	}

	r.mu.Lock()
	r.addresses = addresses
	r.mu.Unlock()

	// 更新 gRPC 连接状态
	state := resolver.State{
		Addresses: addresses,
	}

	// 处理空地址列表的情况
	if len(addresses) == 0 {
		r.logger.Warn("No service instances available",
			clog.String("service", r.serviceName))
		// 仍然更新状态，让 gRPC 处理无可用后端的情况
		// 这会导致连接进入 TRANSIENT_FAILURE 状态，这是正确的行为
	}

	if err := r.cc.UpdateState(state); err != nil {
		// 如果更新状态失败，记录错误但不返回错误
		// 这样可以避免在服务全部下线时产生错误日志
		r.logger.Debug("Failed to update resolver state",
			clog.String("service", r.serviceName),
			clog.Int("address_count", len(addresses)),
			clog.Err(err))
		return nil // 不返回错误，避免影响 watch 循环
	}

	if len(addresses) > 0 {
		r.logger.Info("Service addresses updated",
			clog.String("service", r.serviceName),
			clog.Int("count", len(addresses)))
	} else {
		r.logger.Info("Service addresses cleared (no instances available)",
			clog.String("service", r.serviceName))
	}

	return nil
}

// watch 监听服务变化
func (r *EtcdResolver) watch() {
	prefix := r.buildServicePrefix(r.serviceName)

	for {
		select {
		case <-r.ctx.Done():
			return
		default:
		}

		watchCh := r.client.Watch(r.ctx, prefix, clientv3.WithPrefix())

		for resp := range watchCh {
			if err := resp.Err(); err != nil {
				r.logger.Error("Watch error occurred",
					clog.String("service", r.serviceName),
					clog.Err(err))
				r.cc.ReportError(err)

				// 等待一段时间后重试
				select {
				case <-r.ctx.Done():
					return
				case <-time.After(time.Second):
					break
				}
				continue
			}

			// 处理服务变化事件
			hasChanges := false
			for _, event := range resp.Events {
				switch event.Type {
				case clientv3.EventTypePut, clientv3.EventTypeDelete:
					hasChanges = true
				}
			}

			// 如果有变化，重新解析服务地址
			if hasChanges {
				if err := r.resolveNow(); err != nil {
					r.logger.Error("Failed to resolve services after watch event",
						clog.String("service", r.serviceName),
						clog.Err(err))
					r.cc.ReportError(err)
				}
			}
		}

		// watch 通道关闭，等待一段时间后重新建立 watch
		select {
		case <-r.ctx.Done():
			return
		case <-time.After(time.Second):
		}
	}
}

// ResolveNow 立即触发地址解析
func (r *EtcdResolver) ResolveNow(opts resolver.ResolveNowOptions) {
	go func() {
		if err := r.resolveNow(); err != nil {
			r.logger.Error("ResolveNow failed",
				clog.String("service", r.serviceName),
				clog.Err(err))
			r.cc.ReportError(err)
		}
	}()
}

// Close 关闭 resolver
func (r *EtcdResolver) Close() {
	if r.cancel != nil {
		r.cancel()
	}
	<-r.closed
}

// buildServicePrefix 构建服务前缀
func (r *EtcdResolver) buildServicePrefix(serviceName string) string {
	return fmt.Sprintf("%s/%s/", strings.TrimSuffix(r.prefix, "/"), serviceName)
}
