package etcd

import (
	"context"
	"fmt"
	"log"
	"time"

	clientv3 "go.etcd.io/etcd/client/v3"
	"google.golang.org/grpc/resolver"
)

// EtcdResolverBuilder 实现resolver.Builder接口
type EtcdResolverBuilder struct {
	client      *clientv3.Client
	serviceName string
}

// Build 构建解析器
func (b *EtcdResolverBuilder) Build(target resolver.Target, cc resolver.ClientConn, opts resolver.BuildOptions) (resolver.Resolver, error) {
	r := &EtcdResolver{
		client:      b.client,
		serviceName: b.serviceName,
		cc:          cc,
		ctx:         context.Background(),
		cancel:      nil,
	}
	r.start()
	return r, nil
}

// Scheme 返回解析器方案
func (b *EtcdResolverBuilder) Scheme() string {
	return "etcd"
}

// EtcdResolver 实现resolver.Resolver接口
type EtcdResolver struct {
	client      *clientv3.Client
	serviceName string
	cc          resolver.ClientConn
	ctx         context.Context
	cancel      context.CancelFunc
}

// start 启动解析器
func (r *EtcdResolver) start() {
	r.ctx, r.cancel = context.WithCancel(context.Background())
	go r.watch()
}

// watch 监听服务变化
func (r *EtcdResolver) watch() {
	prefix := fmt.Sprintf("/services/%s/", r.serviceName)

	for {
		// 检查上下文是否取消
		select {
		case <-r.ctx.Done():
			return
		default:
		}

		// 获取当前服务列表
		resp, err := r.client.Get(context.Background(), prefix, clientv3.WithPrefix())
		if err != nil {
			log.Printf("Resolver failed to get services for %s: %v", r.serviceName, err)
			time.Sleep(1 * time.Second)
			continue
		}

		// 更新地址列表
		var addresses []resolver.Address
		for _, kv := range resp.Kvs {
			addresses = append(addresses, resolver.Address{Addr: string(kv.Value)})
		}

		err = r.cc.UpdateState(resolver.State{Addresses: addresses})
		if err != nil {
			log.Printf("Resolver failed to update state for %s: %v", r.serviceName, err)
			time.Sleep(1 * time.Second)
			continue
		}

		log.Printf("Resolver updated %d addresses for service %s", len(addresses), r.serviceName)

		// 监听变化
		watchCtx, cancel := context.WithCancel(r.ctx)
		watchChan := r.client.Watch(watchCtx, prefix, clientv3.WithPrefix())

		// 等待变化或上下文取消
		select {
		case <-watchChan:
			cancel()
			continue
		case <-r.ctx.Done():
			cancel()
			return
		}
	}
}

// ResolveNow 实现接口
func (r *EtcdResolver) ResolveNow(resolver.ResolveNowOptions) {}

// Close 关闭解析器
func (r *EtcdResolver) Close() {
	if r.cancel != nil {
		r.cancel()
	}
}
