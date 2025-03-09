package connect

import (
	"context"
	"fmt"
	"gochat/clog"
	"gochat/config"
	pb "gochat/proto/connectproto"
	"gochat/tools"
	"net"
	"os"
	"os/signal"
	"syscall"

	"google.golang.org/grpc"
)

type server struct {
	pb.UnimplementedConnectServiceServer
}

// 匿名类型 server 实现了 ConnectServiceServer 接口
var _ pb.ConnectServiceServer = (*server)(nil)

// InitRPCServer 初始化RPC服务器并将服务注册到etcd
func InitRPCServer() {
	// 创建gRPC服务器
	lis, err := net.Listen("tcp", fmt.Sprintf(":%d", config.Conf.RPC.Port))
	if err != nil {
		clog.Error("failed to listen: %v", err)
		return
	}

	// 创建gRPC服务器实例
	s := grpc.NewServer()

	// 注册connect服务
	pb.RegisterConnectServiceServer(s, &server{})

	// 生成唯一实例ID
	ip, err := tools.GetLocalIP()
	if err != nil {
		clog.Error("failed to get local IP: %v", err)
	}
	instanceID := fmt.Sprintf("connect-server-%d-%s", config.Conf.RPC.Port, ip)

	// 创建上下文，用于服务注册和取消注册
	ctx, cancel := context.WithCancel(context.Background())

	// 服务地址
	addr := fmt.Sprintf("%s:%d", ip, config.Conf.RPC.Port)

	// 注册服务到etcd
	go func() {
		err := tools.ServiceRegistry(ctx, "connect-service", instanceID, addr)
		if err != nil {
			clog.Error("failed to register service: %v", err)
			cancel() // 注册失败，取消上下文
			return
		}
		clog.Info("service registered with etcd: connect-service/%s at %s", instanceID, addr)
	}()

	// 设置优雅关闭
	go func() {
		sigCh := make(chan os.Signal, 1)
		signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
		<-sigCh

		// 收到关闭信号
		clog.Info("shutting down gRPC server...")
		s.GracefulStop()
		cancel() // 取消服务注册
		clog.Info("service unregistered and server stopped")
	}()

	clog.Info("Connect RPC server starting on port %d", config.Conf.RPC.Port)

	// 启动gRPC服务器
	if err := s.Serve(lis); err != nil {
		clog.Error("failed to serve: %v", err)
	}
}

func (s *server) PushSingleMsg(ctx context.Context, in *pb.PushMsgRequest) (*pb.SuccessReply, error) {
	// todo: 实现推送单条消息的逻辑
	return &pb.SuccessReply{Code: config.RPCCodeSuccess, Msg: "push msg to user success"}, nil
}

func (s *server) PushRoomMsg(ctx context.Context, in *pb.PushRoomMsgRequest) (*pb.SuccessReply, error) {
	// todo: 实现推送房间消息的逻辑
	return &pb.SuccessReply{Code: config.RPCCodeSuccess, Msg: "push msg to room success"}, nil
}

func (s *server) PushRoomCount(ctx context.Context, in *pb.PushRoomMsgRequest) (*pb.SuccessReply, error) {
	// todo: 实现推送房间人数的逻辑
	return &pb.SuccessReply{Code: config.RPCCodeSuccess, Msg: "push room count success"}, nil
}

func (s *server) PushRoomInfo(ctx context.Context, in *pb.PushRoomMsgRequest) (*pb.SuccessReply, error) {
	// todo: 实现推送房间信息的逻辑
	return &pb.SuccessReply{Code: config.RPCCodeSuccess, Msg: "push room info success"}, nil
}
