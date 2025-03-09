package logic

import (
	"context"
	"encoding/json"
	"fmt"
	"gochat/clog"
	"gochat/config"
	pb "gochat/proto/logicproto"
	"gochat/tools"
	"gochat/tools/queue"
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
	lis, err := net.Listen("tcp", fmt.Sprintf(":%d", config.Conf.RPC.Port))
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
	}
	instanceID := fmt.Sprintf("logic-server-%d-%s", config.Conf.RPC.Port, ip)

	// 创建上下文，用于服务注册和取消注册
	ctx, cancel := context.WithCancel(context.Background())

	// 服务地址
	addr := fmt.Sprintf("%s:%d", ip, config.Conf.RPC.Port)

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

	clog.Info("logic RPC server starting on port %d", config.Conf.RPC.Port)

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
	// 如果用户已经登录，撤销旧 Token
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

func (s *server) Push(ctx context.Context, in *pb.Send) (*pb.SuccessReply, error) {
	bodyBytes, err := json.Marshal(in)
	if err != nil {
		return &pb.SuccessReply{Code: config.RPCCodeFailed}, nil
	}
	userSidKey := fmt.Sprintf("gochat_%d", in.ToUserId)
	serverIdStr := RedisClient.Get(ctx, userSidKey).Val()
	QueueMsg := queue.QueueMsg{
		Op:       config.OpSingleSend,
		ServerId: serverIdStr,
		Msg:      bodyBytes,
		UserId:   int(in.ToUserId),
	}
	err = queue.DefaultQueue.PublishMessage(&QueueMsg)
	if err != nil {
		return &pb.SuccessReply{Code: config.RPCCodeFailed}, nil
	}
	return &pb.SuccessReply{Code: config.RPCCodeSuccess}, nil
}

func (s *server) PushRoom(ctx context.Context, in *pb.Send) (*pb.SuccessReply, error) {
	bodyBytes, err := json.Marshal(in)
	if err != nil {
		return &pb.SuccessReply{Code: config.RPCCodeFailed}, nil
	}
	roomUserKey := fmt.Sprintf("gochat_room_%d", in.RoomId)
	roomUserInfo, err := RedisClient.HGetAll(ctx, roomUserKey).Result()
	if err != nil {
		return &pb.SuccessReply{Code: config.RPCCodeFailed}, nil
	}
	QueueMsg := queue.QueueMsg{
		Op:           config.OpRoomSend,
		RoomId:       int(in.RoomId),
		Count:        len(roomUserInfo),
		Msg:          bodyBytes,
		RoomUserInfo: roomUserInfo,
	}
	err = queue.DefaultQueue.PublishMessage(&QueueMsg)
	if err != nil {
		return &pb.SuccessReply{Code: config.RPCCodeFailed}, nil
	}
	return &pb.SuccessReply{Code: config.RPCCodeSuccess}, nil
}

func (s *server) Count(ctx context.Context, in *pb.Send) (*pb.SuccessReply, error) {
	count, err := RedisClient.Get(ctx, fmt.Sprintf("gochat_room_online_count_%d", in.RoomId)).Int()
	if err != nil {
		return &pb.SuccessReply{Code: config.RPCCodeFailed}, nil
	}
	QueueMsg := queue.QueueMsg{
		Op:     config.OpRoomCountSend,
		RoomId: int(in.RoomId),
		Count:  count,
	}
	err = queue.DefaultQueue.PublishMessage(&QueueMsg)
	if err != nil {
		return &pb.SuccessReply{Code: config.RPCCodeFailed}, nil
	}
	return &pb.SuccessReply{Code: config.RPCCodeSuccess}, nil
}

func (s *server) GetRoomInfo(ctx context.Context, in *pb.Send) (*pb.SuccessReply, error) {
	roomUserKey := fmt.Sprintf("gochat_room_%d", in.RoomId)
	roomUserInfo, err := RedisClient.HGetAll(ctx, roomUserKey).Result()
	if err != nil {
		return &pb.SuccessReply{Code: config.RPCCodeFailed}, nil
	}
	QueueMsg := queue.QueueMsg{
		Op:           config.OpRoomInfoSend,
		RoomId:       int(in.RoomId),
		Count:        len(roomUserInfo),
		RoomUserInfo: roomUserInfo,
	}
	err = queue.DefaultQueue.PublishMessage(&QueueMsg)
	if err != nil {
		return &pb.SuccessReply{Code: config.RPCCodeFailed}, nil
	}
	return &pb.SuccessReply{Code: config.RPCCodeSuccess}, nil
}

func (s *server) Connect(ctx context.Context, in *pb.ConnectRequest) (*pb.ConnectReply, error) {
	userID, userName, _, err := tools.ValidateToken(in.AuthToken)
	if err != nil {
		return &pb.ConnectReply{UserId: int32(userID)}, nil
	}
	// 1. 建立 userID 到 serverID 的映射关系
	userSidKey := fmt.Sprintf("gochat_%d", userID)
	err = RedisClient.Set(ctx, userSidKey, in.ServerId, 0).Err()
	if err != nil {
		clog.Error("failed to set user-server mapping: %v", err)
		return &pb.ConnectReply{UserId: int32(userID)}, nil
	}
	// 2. 添加用户到房间成员信息中 (使用 userID 作为字段名，serverID 作为值)
	roomUserKey := fmt.Sprintf("gochat_room_%d", in.RoomId)
	err = RedisClient.HSet(ctx, roomUserKey, fmt.Sprintf("%d", userID), userName).Err()
	if err != nil {
		clog.Error("failed to add user to room: %v", err)
		return &pb.ConnectReply{UserId: int32(userID)}, nil
	}
	// 3. 在线用户数量加一
	_, err = RedisClient.Incr(ctx, fmt.Sprintf("gochat_room_online_count_%d", in.RoomId)).Result()
	if err != nil {
		clog.Error("failed to increment online count: %v", err)
		return &pb.ConnectReply{UserId: int32(userID)}, nil
	}
	// 4. 通过消息队列广播房间信息更新
	roomUserInfo, err := RedisClient.HGetAll(ctx, roomUserKey).Result()
	if err == nil {
		// 发送房间成员信息更新消息
		infoQueueMsg := queue.QueueMsg{
			Op:           config.OpRoomInfoSend,
			RoomId:       int(in.RoomId),
			Count:        len(roomUserInfo),
			RoomUserInfo: roomUserInfo,
		}
		queue.DefaultQueue.PublishMessage(&infoQueueMsg)

		// 发送房间人数更新消息
		countQueueMsg := queue.QueueMsg{
			Op:     config.OpRoomCountSend,
			RoomId: int(in.RoomId),
			Count:  len(roomUserInfo),
		}
		queue.DefaultQueue.PublishMessage(&countQueueMsg)
	}
	return &pb.ConnectReply{UserId: int32(userID)}, nil
}

func (s *server) Disconnect(ctx context.Context, in *pb.DisConnectRequest) (*pb.DisConnectReply, error) {
	// 1. 删除 userID 到 serverID 的映射关系
	userSidKey := fmt.Sprintf("gochat_%d", in.UserId)
	err := RedisClient.Del(ctx, userSidKey).Err()
	if err != nil {
		clog.Error("failed to delete user-server mapping: %v", err)
	}
	// 2. 从房间成员信息中删除用户
	roomUserKey := fmt.Sprintf("gochat_room_%d", in.RoomId)
	userIDStr := fmt.Sprintf("%d", in.UserId)
	_, err = RedisClient.HDel(ctx, roomUserKey, userIDStr).Result()
	if err != nil {
		clog.Error("failed to remove user from room: %v", err)
		return &pb.DisConnectReply{}, nil
	}
	// 3. 减少房间在线人数
	_, err = RedisClient.Decr(ctx, fmt.Sprintf("gochat_room_online_count_%d", in.RoomId)).Result()
	if err != nil {
		clog.Error("failed to decrement room count: %v", err)
		return &pb.DisConnectReply{}, nil
	}

	// 4. 通过消息队列广播房间信息更新
	roomUserInfo, err := RedisClient.HGetAll(ctx, roomUserKey).Result()
	if err == nil {
		// 发送房间成员信息更新消息
		infoQueueMsg := queue.QueueMsg{
			Op:           config.OpRoomInfoSend,
			RoomId:       int(in.RoomId),
			Count:        len(roomUserInfo),
			RoomUserInfo: roomUserInfo,
		}
		queue.DefaultQueue.PublishMessage(&infoQueueMsg)

		// 发送房间人数更新消息
		countQueueMsg := queue.QueueMsg{
			Op:     config.OpRoomCountSend,
			RoomId: int(in.RoomId),
			Count:  len(roomUserInfo),
		}
		queue.DefaultQueue.PublishMessage(&countQueueMsg)
	}
	return &pb.DisConnectReply{}, nil
}
