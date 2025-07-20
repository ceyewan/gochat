package tools

import (
	"context"
	"fmt"
	"net"
	"testing"
	"time"

	"gochat/clog"
	"gochat/config"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/connectivity"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/emptypb"
)

// MockService 定义测试用的gRPC服务接口
type MockService interface {
	Ping(context.Context, *emptypb.Empty) (*PingResponse, error)
}

// PingResponse 是Ping方法的响应
type PingResponse struct {
	Message   string `json:"message"`
	Instance  string `json:"instance"`
	Timestamp int64  `json:"timestamp"`
}

// mockServiceServer 是MockService的实现
type mockServiceServer struct {
	instanceID                     string
	UnimplementedMockServiceServer // 添加未实现的方法存根
}

// 添加未实现的服务存根接口
type UnimplementedMockServiceServer struct{}

// Ping 实现MockService接口的Ping方法
func (s *mockServiceServer) Ping(ctx context.Context, _ *emptypb.Empty) (*PingResponse, error) {
	return &PingResponse{
		Message:   "pong",
		Instance:  s.instanceID,
		Timestamp: time.Now().Unix(),
	}, nil
}

// MockServiceClient 是MockService的客户端
type MockServiceClient struct {
	instanceID string
	conn       *grpc.ClientConn
}

// Ping 调用远程Ping方法
func (c *MockServiceClient) Ping(ctx context.Context, _ *emptypb.Empty) (*PingResponse, error) {
	// 这里简单模拟RPC调用
	if c.conn.GetState() != connectivity.Ready && c.conn.GetState() != connectivity.Idle {
		return nil, status.Error(codes.Unavailable, "connection not ready")
	}
	return &PingResponse{
		Message:   "pong",
		Instance:  c.instanceID,
		Timestamp: time.Now().Unix(),
	}, nil
}

// 测试常量
const (
	testServiceName = "test-service"
	etcdEndpoint    = "localhost:23791" // 请根据环境修改
	basePort        = 50000
)

// TestServiceRegistryAndDiscovery 测试服务注册和发现的完整流程
func TestServiceRegistryAndDiscovery(t *testing.T) {
	// 初始化配置
	if err := setupTestConfig(); err != nil {
		t.Fatalf("Failed to setup test config: %v", err)
	}

	// 初始化etcd客户端
	if err := InitEtcdClient(); err != nil {
		t.Skipf("Skipping test: Failed to initialize etcd client: %v", err)
		return
	}
	defer CloseEtcdClient()

	// 启动模拟服务器
	servers := startMockServers(t, 3)
	defer func() {
		for _, s := range servers {
			s.server.GracefulStop()
		}
	}()

	// 创建根上下文
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// 1. 为每个服务器注册服务
	for i, s := range servers {
		instanceID := fmt.Sprintf("instance-%d", i+1)
		err := ServiceRegistry(ctx, testServiceName, instanceID, s.addr)
		if err != nil {
			t.Fatalf("Failed to register service instance %d: %v", i+1, err)
		}
		t.Logf("Registered service instance %d: %s at %s", i+1, instanceID, s.addr)
	}

	// 等待服务注册生效
	time.Sleep(1 * time.Second)

	// 2. 使用服务发现获取负载均衡连接
	t.Log("Testing service discovery...")
	conn, err := ServiceDiscovery(ctx, testServiceName)
	if err != nil {
		t.Fatalf("Failed to discover service: %v", err)
	}
	defer conn.Close()

	// 3. 验证服务实例连接
	instances, err := GetAllServiceInstanceConns(testServiceName)
	if err != nil {
		t.Fatalf("Failed to get service instances: %v", err)
	}

	if len(instances) != len(servers) {
		t.Fatalf("Expected %d service instances, got %d", len(servers), len(instances))
	}
	t.Logf("Successfully discovered %d service instances", len(instances))

	// 4. 测试负载均衡(模拟多次调用)
	t.Log("Testing load balancing...")
	client := &MockServiceClient{conn: conn}

	// 多次调用服务，验证负载均衡
	for i := 0; i < 10; i++ {
		resp, err := client.Ping(ctx, &emptypb.Empty{})
		if err != nil {
			t.Errorf("Failed to call service: %v", err)
			continue
		}
		t.Logf("Call %d responded by instance: %s", i+1, resp.Instance)
	}

	// 5. 模拟一个服务实例下线
	t.Log("Testing service instance removal...")
	// 停止第一个服务器
	servers[0].server.GracefulStop()

	// 取消第一个实例的服务注册
	_, instanceCancel1 := context.WithCancel(context.Background())
	instanceCancel1()

	// 等待服务注册过期(租约TTL为5秒)
	t.Log("Waiting for service lease to expire...")
	time.Sleep(6 * time.Second)

	// 6. 验证服务发现更新
	instances, err = GetAllServiceInstanceConns(testServiceName)
	if err != nil {
		t.Fatalf("Failed to get service instances: %v", err)
	}

	// 应该剩下2个实例
	expectedCount := len(servers) - 1
	if len(instances) != expectedCount {
		t.Fatalf("Expected %d service instances after removal, got %d",
			expectedCount, len(instances))
	}
	t.Logf("Successfully detected service instance removal, %d instances remaining", len(instances))

	// 7. 测试动态添加服务实例
	t.Log("Testing dynamic service addition...")
	newServer := startMockServer(t, len(servers)+1)
	defer newServer.server.GracefulStop()

	newInstanceID := fmt.Sprintf("instance-%d", len(servers)+1)
	err = ServiceRegistry(ctx, testServiceName, newInstanceID, newServer.addr)
	if err != nil {
		t.Fatalf("Failed to register new service instance: %v", err)
	}
	t.Logf("Registered new service instance: %s at %s", newInstanceID, newServer.addr)

	// 等待服务发现更新
	time.Sleep(2 * time.Second)

	// 8. 验证新服务实例被发现
	instances, err = GetAllServiceInstanceConns(testServiceName)
	if err != nil {
		t.Fatalf("Failed to get service instances: %v", err)
	}

	// 应该有3个实例(2个原有+1个新增)
	if len(instances) != expectedCount+1 {
		t.Fatalf("Expected %d service instances after addition, got %d",
			expectedCount+1, len(instances))
	}
	t.Logf("Successfully discovered new service instance, total %d instances", len(instances))
}

// 为了注册服务，我们需要定义gRPC服务描述符
type mockServiceDesc struct {
	srv interface{}
	ss  grpc.ServiceDesc
}

// RegisterMockService 注册MockService到gRPC服务器
func RegisterMockService(s *grpc.Server, srv *mockServiceServer) {
	desc := grpc.ServiceDesc{
		ServiceName: "MockService",
		HandlerType: (*MockService)(nil),
		Methods: []grpc.MethodDesc{
			{
				MethodName: "Ping",
				Handler: func(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
					in := new(emptypb.Empty)
					if err := dec(in); err != nil {
						return nil, err
					}
					if interceptor == nil {
						return srv.(MockService).Ping(ctx, in)
					}
					info := &grpc.UnaryServerInfo{
						Server:     srv,
						FullMethod: "/MockService/Ping",
					}
					handler := func(ctx context.Context, req interface{}) (interface{}, error) {
						return srv.(MockService).Ping(ctx, req.(*emptypb.Empty))
					}
					return interceptor(ctx, in, info, handler)
				},
			},
		},
		Streams: []grpc.StreamDesc{},
	}
	s.RegisterService(&desc, srv)
}

// mockServer 表示一个模拟的gRPC服务器
type mockServer struct {
	server   *grpc.Server
	listener net.Listener
	addr     string
	id       string
}

// startMockServer 启动单个模拟服务器
func startMockServer(t *testing.T, id int) *mockServer {
	// 创建实际的TCP监听器
	port := basePort + id
	addr := fmt.Sprintf("localhost:%d", port)

	lis, err := net.Listen("tcp", addr)
	if err != nil {
		t.Fatalf("Failed to listen on %s: %v", addr, err)
	}

	server := grpc.NewServer()
	instanceID := fmt.Sprintf("instance-%d", id)

	// 注册服务实现
	mockSvc := &mockServiceServer{instanceID: instanceID}
	// 修复：将服务实现注册到gRPC服务器
	RegisterMockService(server, mockSvc)

	go func() {
		if err := server.Serve(lis); err != nil {
			t.Logf("Server %s stopped: %v", instanceID, err)
		}
	}()

	return &mockServer{
		server:   server,
		listener: lis,
		addr:     addr,
		id:       instanceID,
	}
}

// startMockServers 启动多个模拟服务器
func startMockServers(t *testing.T, count int) []*mockServer {
	servers := make([]*mockServer, count)
	for i := 0; i < count; i++ {
		servers[i] = startMockServer(t, i+1)
	}
	return servers
}

// setupTestConfig 设置测试配置
func setupTestConfig() error {
	clog.Module("etcd_test").Debugf("Set log level to debug")

	// 设置etcd配置
	config.Conf.Etcd.Addrs = []string{etcdEndpoint}

	return nil
}
