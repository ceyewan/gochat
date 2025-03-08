package logic

import (
	"context"
	"fmt"
	"gochat/clog"
	"gochat/config"
	pb "gochat/proto/logicproto"
	"gochat/tools"
	"net"
	"os"
	"os/signal"
	"syscall"
	"time"

	"google.golang.org/grpc"
)

type server struct {
	pb.UnimplementedChatLogicServiceServer
}

// 匿名类型 server 实现 ChatLogicServiceServer 接口
var _ pb.ChatLogicServiceServer = (*server)(nil)

// InitRPCServer 初始化RPC服务器并将服务注册到etcd
func InitRPCServer() {
	// 创建gRPC服务器
	lis, err := net.Listen("tcp", fmt.Sprintf(":%d", config.Conf.LogicRPC.Port))
	if err != nil {
		clog.Error("failed to listen: %v", err)
		return
	}

	// 创建gRPC服务器实例
	s := grpc.NewServer()

	// 注册Logic服务
	pb.RegisterChatLogicServiceServer(s, &server{})

	// 生成唯一实例ID
	ip, err := tools.GetLocalIP()
	if err != nil {
		clog.Error("failed to get local IP: %v", err)
		ip = "127.0.0.1" // fallback to localhost
	}
	instanceID := fmt.Sprintf("logic-server-%d-%s", config.Conf.LogicRPC.Port, ip)

	// 创建上下文，用于服务注册和取消注册
	ctx, cancel := context.WithCancel(context.Background())

	// 服务地址
	addr := fmt.Sprintf("%s:%d", ip, config.Conf.LogicRPC.Port)

	// 注册服务到etcd
	go func() {
		err := tools.ServiceRegistry(ctx, "logic-service", instanceID, addr)
		if err != nil {
			clog.Error("failed to register service: %v", err)
			cancel() // 注册失败，取消上下文
			return
		}
		clog.Info("service registered with etcd: logic-service/%s at %s", instanceID, addr)
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

	clog.Info("logic RPC server starting on port %d", config.Conf.LogicRPC.Port)

	// 启动gRPC服务器
	if err := s.Serve(lis); err != nil {
		clog.Error("failed to serve: %v", err)
	}
}

func (s *server) Login(ctx context.Context, in *pb.LoginRequest) (*pb.LoginResponse, error) {
	user, err := CheckHaveUserName(in.Name)
	if err != nil {
		return &pb.LoginResponse{Code: config.RPCCodeFailed, AuthToken: ""}, nil
	}
	if !checkPasswordHash(in.Password, user.Password) {
		return &pb.LoginResponse{Code: config.RPCCodeFailed, AuthToken: ""}, nil
	}
	sessionID := fmt.Sprintf("sess_map_%d", user.ID)
	newToken, err := tools.GenerateToken(int(user.ID), in.Name, in.Password, time.Hour)
	if err != nil {
		return &pb.LoginResponse{Code: config.RPCCodeFailed, AuthToken: ""}, nil
	}
	oldToken, _ := RedisClient.Get(ctx, sessionID).Result()
	if oldToken != "" {
		// Todo JWT 撤销旧Token
	}
	RedisClient.Set(ctx, sessionID, newToken, time.Hour)
	return &pb.LoginResponse{Code: config.RPCCodeSuccess, AuthToken: newToken}, nil
}

func (s *server) Register(ctx context.Context, in *pb.RegisterRequest) (*pb.RegisterReply, error) {
	_, err := CheckHaveUserName(in.Name)
	if err == nil {
		return &pb.RegisterReply{Code: config.RPCCodeFailed}, nil
	}
	err = Add(in.Name, in.Password)
	if err != nil {
		return &pb.RegisterReply{Code: config.RPCCodeFailed}, nil
	}
	return &pb.RegisterReply{Code: config.RPCCodeSuccess}, nil
}

func (s *server) Logout(ctx context.Context, in *pb.LogoutRequest) (*pb.LogoutResponse, error) {
	userID, _, _, err := tools.ValidateToken(in.AuthToken)
	if err != nil {
		return &pb.LogoutResponse{Code: config.RPCCodeFailed}, nil
	}
	sessionID := fmt.Sprintf("sess_map_%d", userID)
	oldToken, _ := RedisClient.Get(ctx, sessionID).Result()
	if oldToken == "" {
		return &pb.LogoutResponse{Code: config.RPCCodeFailed}, nil
	}
	// Todo JWT 撤销Token
	RedisClient.Del(ctx, sessionID)
	return &pb.LogoutResponse{Code: config.RPCCodeSuccess}, nil
}

func (s *server) CheckAuth(ctx context.Context, in *pb.CheckAuthRequest) (*pb.CheckAuthResponse, error) {
	userID, userName, _, err := tools.ValidateToken(in.AuthToken)
	if err != nil {
		return &pb.CheckAuthResponse{Code: config.RPCCodeFailed}, nil
	}
	return &pb.CheckAuthResponse{Code: config.RPCCodeSuccess, UserId: int32(userID), UserName: userName}, nil
}

func (s *server) GetUserInfoByUserId(ctx context.Context, in *pb.GetUserInfoRequest) (*pb.GetUserInfoResponse, error) {
	userName, err := GetUserNameByUserID(uint(in.UserId))
	if err != nil {
		return &pb.GetUserInfoResponse{Code: config.RPCCodeFailed}, nil
	}
	return &pb.GetUserInfoResponse{Code: config.RPCCodeSuccess, UserName: userName}, nil
}
