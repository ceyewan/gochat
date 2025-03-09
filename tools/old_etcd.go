package tools

// import (
// 	"context"
// 	"fmt"
// 	"sync"
// 	"time"

// 	clientv3 "go.etcd.io/etcd/client/v3"
// 	"google.golang.org/grpc"
// 	"google.golang.org/grpc/credentials/insecure"
// 	"google.golang.org/grpc/resolver"
// )

// var (
// 	etcdClient     *clientv3.Client
// 	etcdClientOnce sync.Once
// )

// // InitEtcdClient 初始化etcd客户端，确保只初始化一次
// func InitEtcdClient(endpoints []string, dialTimeout time.Duration) error {
// 	var err error
// 	etcdClientOnce.Do(func() {
// 		etcdClient, err = clientv3.New(clientv3.Config{
// 			Endpoints:   endpoints,
// 			DialTimeout: dialTimeout,
// 		})
// 	})
// 	return err
// }

// // ServiceRegistry 处理服务注册
// func ServiceRegistry(ctx context.Context, serviceName, instanceID, addr string) error {
// 	if etcdClient == nil {
// 		return fmt.Errorf("etcd client未初始化，请先调用InitEtcdClient")
// 	}
// 	// 创建租约，TTL 设置为 5 秒
// 	lease, err := etcdClient.Grant(ctx, 5)
// 	if err != nil {
// 		return fmt.Errorf("创建租约失败: %v", err)
// 	}
// 	// 设置自动续租
// 	keepAliveCh, err := etcdClient.KeepAlive(ctx, lease.ID)
// 	if err != nil {
// 		return fmt.Errorf("设置自动续租失败: %v", err)
// 	}
// 	// 后台处理续租响应
// 	go func() {
// 		for {
// 			select {
// 			case <-keepAliveCh:
// 				// 续租成功，忽略响应
// 			case <-ctx.Done():
// 				return
// 			}
// 		}
// 	}()
// 	// 注册服务实例，键为 /services/{serviceName}/{instanceID}，值为地址（如 127.0.0.1:8080）
// 	key := fmt.Sprintf("/services/%s/%s", serviceName, instanceID)
// 	_, err = etcdClient.Put(ctx, key, addr, clientv3.WithLease(lease.ID))
// 	if err != nil {
// 		return fmt.Errorf("注册服务失败: %v", err)
// 	}
// 	return nil
// }

// // ServiceDiscovery 处理服务发现，返回gRPC连接池，这个连接池会自动处理负载均衡
// // 连接池在某个服务实例不可用时会自动切换到其他可用实例，也具有自动重连功能
// func ServiceDiscovery(ctx context.Context, serviceName string) (*grpc.ClientConn, error) {
// 	// 创建自定义resolver，关联etcd客户端和要查找的服务名
// 	builder := &etcdResolverBuilder{
// 		client:      etcdClient,
// 		serviceName: serviceName,
// 	}
// 	// 向gRPC注册这个自定义的解析器，使gRPC能够识别etcd://开头的URL
// 	resolver.Register(builder)
// 	// 创建gRPC连接
// 	conn, err := grpc.DialContext(
// 		ctx,
// 		fmt.Sprintf("etcd:///%s", serviceName),
// 		grpc.WithTransportCredentials(insecure.NewCredentials()),
// 		grpc.WithDefaultServiceConfig(`{"loadBalancingPolicy":"round_robin"}`),
// 	)
// 	if err != nil {
// 		return nil, fmt.Errorf("创建gRPC连接失败: %v", err)
// 	}
// 	return conn, nil
// }

// // etcdResolverBuilder 实现resolver.Builder接口
// type etcdResolverBuilder struct {
// 	client      *clientv3.Client
// 	serviceName string
// }

// // Build 接收目标服务信息和gRPC客户端连接，构建并返回一个新的解析器
// func (b *etcdResolverBuilder) Build(target resolver.Target, cc resolver.ClientConn, opts resolver.BuildOptions) (resolver.Resolver, error) {
// 	r := &etcdResolver{
// 		client:      b.client,
// 		serviceName: b.serviceName,
// 		cc:          cc,
// 	}
// 	// 初始解析
// 	r.start()
// 	return r, nil
// }

// // Scheme 定义 URL 方案为 etcd，使 gRPC 能够识别 etcd:///{serviceName} 形式的地址
// func (b *etcdResolverBuilder) Scheme() string {
// 	return "etcd"
// }

// // etcdResolver 实现resolver.Resolver接口
// type etcdResolver struct {
// 	client      *clientv3.Client
// 	serviceName string
// 	cc          resolver.ClientConn
// }

// func (r *etcdResolver) start() {
// 	// 获取初始服务列表
// 	go r.watch()
// }

// // watch 监听服务变化
// func (r *etcdResolver) watch() {
// 	prefix := fmt.Sprintf("/services/%s/", r.serviceName)
// 	for {
// 		// 获取当前所有服务实例
// 		resp, err := r.client.Get(context.Background(), prefix, clientv3.WithPrefix())
// 		if err != nil {
// 			continue
// 		}
// 		var addresses []resolver.Address
// 		for _, kv := range resp.Kvs {
// 			addresses = append(addresses, resolver.Address{Addr: string(kv.Value)})
// 		}
// 		// 更新gRPC客户端连接状态
// 		err = r.cc.UpdateState(resolver.State{Addresses: addresses})
// 		if err != nil {
// 			return
// 		}
// 		// 监听服务变更
// 		watchChan := r.client.Watch(context.Background(), prefix, clientv3.WithPrefix())
// 		for range watchChan {
// 			// 服务列表发生变化，重新获取列表
// 			break
// 		}
// 	}
// }

// func (r *etcdResolver) ResolveNow(resolver.ResolveNowOptions) {}

// func (r *etcdResolver) Close() {}
