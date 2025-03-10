package logic

import (
	"context"
	"fmt"
	"gochat/clog"
	"gochat/config"
	pb "gochat/proto/logicproto"
	"gochat/tools"
	"gochat/tools/queue"
	"net"
	"time"

	"google.golang.org/protobuf/proto"

	"google.golang.org/grpc"
)

// server 实现 ChatLogicServiceServer 接口
type server struct {
	pb.UnimplementedChatLogicServiceServer
}

// server 实现 ChatLogicServiceServer 接口
var _ pb.ChatLogicServiceServer = (*server)(nil)

// InitRPCServer 初始化RPC服务器并注册到服务发现
func InitRPCServer(ctx context.Context) (*grpc.Server, error) {
	// 创建监听器
	lis, err := net.Listen("tcp", fmt.Sprintf(":%d", config.Conf.RPC.Port))
	if err != nil {
		clog.Error("[RPC] Failed to listen: %v", err)
		return nil, err
	}

	// 创建gRPC服务器
	s := grpc.NewServer()
	pb.RegisterChatLogicServiceServer(s, &server{})

	// 获取本机IP用于服务注册
	ip, err := tools.GetLocalIP()
	if err != nil {
		clog.Error("[RPC] Failed to get local IP: %v", err)
		return nil, err
	}

	// 服务实例标识和地址
	instanceID := fmt.Sprintf("logic-server-%d-%s", config.Conf.RPC.Port, ip)
	addr := fmt.Sprintf("%s:%d", ip, config.Conf.RPC.Port)

	// 创建上下文，用于服务注册和取消注册
	ctx, cancel := context.WithCancel(context.Background())

	// 注册服务到etcd
	go func() {
		err := tools.ServiceRegistry(ctx, "logic-service", instanceID, addr)
		if err != nil {
			clog.Error("[RPC] Failed to register service: %v", err)
			cancel()
			return
		}
		clog.Info("[RPC] Service registered with etcd: logic-service/%s at %s", instanceID, addr)
	}()

	clog.Info("[RPC] Logic server starting on port %d", config.Conf.RPC.Port)

	// 启动gRPC服务器
	go func() {
		if err := s.Serve(lis); err != nil {
			clog.Error("[RPC] Failed to serve: %v", err)
		}
	}()

	return s, nil
}

// Login 处理用户登录
func (s *server) Login(ctx context.Context, in *pb.LoginRequest) (*pb.LoginReply, error) {
	// 验证用户名和密码
	user, err := CheckHaveUserName(in.Name)
	if err != nil {
		clog.Warning("[RPC] Login failed for user %s: user not found", in.Name)
		return &pb.LoginReply{Code: config.RPCCodeFailed}, err
	}

	if !checkPasswordHash(in.Password, user.Password) {
		clog.Warning("[RPC] Login failed for user %s: invalid password", in.Name)
		return &pb.LoginReply{Code: config.RPCCodeFailed}, fmt.Errorf("check password failed")
	}

	// 生成新Token
	sessionID := fmt.Sprintf("sess_map_%d", user.ID)
	newToken, err := tools.GenerateToken(int(user.ID), in.Name, in.Password, time.Hour)
	if err != nil {
		clog.Error("[RPC] Failed to generate token for user %s: %v", in.Name, err)
		return &pb.LoginReply{Code: config.RPCCodeFailed}, err
	}
	// 检查并处理旧Token
	oldToken, _ := RedisClient.Get(ctx, sessionID).Result()
	if oldToken != "" {
		tools.RevokeToken(oldToken)
		clog.Debug("[RPC] Replacing existing token for user %s", in.Name)
	}

	// 存储新的会话Token
	RedisClient.Set(ctx, sessionID, newToken, time.Hour)
	clog.Info("[RPC] User %s logged in successfully", in.Name)

	return &pb.LoginReply{
		Code:     config.RPCCodeSuccess,
		UserId:   int32(user.ID),
		UserName: in.Name,
		Token:    newToken,
	}, nil
}

// Register 处理用户注册
func (s *server) Register(ctx context.Context, in *pb.RegisterRequest) (*pb.CommonReply, error) {
	// 检查用户名是否已存在
	_, err := CheckHaveUserName(in.Name)
	if err == nil {
		clog.Warning("[RPC] Registration failed: username %s already exists", in.Name)
		return &pb.CommonReply{Code: config.RPCCodeFailed}, fmt.Errorf("username already exists")
	}

	// 创建新用户
	err = Add(in.Name, in.Password)
	if err != nil {
		clog.Error("[RPC] Failed to add new user %s: %v", in.Name, err)
		return &pb.CommonReply{Code: config.RPCCodeFailed}, err
	}

	clog.Info("[RPC] User %s registered successfully", in.Name)
	return &pb.CommonReply{Code: config.RPCCodeSuccess}, nil
}

// Logout 处理用户登出
func (s *server) Logout(ctx context.Context, in *pb.LogoutRequest) (*pb.CommonReply, error) {
	// 验证Token
	userID, _, _, err := tools.ValidateToken(in.Token)
	if err != nil {
		clog.Warning("[RPC] Logout failed: invalid token")
		return &pb.CommonReply{Code: config.RPCCodeFailed}, err
	}

	// 检查会话是否存在
	sessionID := fmt.Sprintf("sess_map_%d", userID)
	oldToken, _ := RedisClient.Get(ctx, sessionID).Result()
	if oldToken == "" {
		clog.Warning("[RPC] Logout failed: session not found for user ID %d", userID)
		return &pb.CommonReply{Code: config.RPCCodeFailed}, fmt.Errorf("session not found")
	}

	// 删除会话
	RedisClient.Del(ctx, sessionID)
	tools.RevokeToken(oldToken)
	clog.Info("[RPC] User %d logged out successfully", userID)

	return &pb.CommonReply{Code: config.RPCCodeSuccess}, nil
}

// CheckAuth 验证用户认证状态
func (s *server) CheckAuth(ctx context.Context, in *pb.CheckAuthRequest) (*pb.CommonReply, error) {
	// 验证Token
	userID, userName, _, err := tools.ValidateToken(in.Token)
	if err != nil {
		clog.Warning("[RPC] Authentication check failed: invalid token")
		return &pb.CommonReply{Code: config.RPCCodeFailed}, err
	}

	clog.Debug("[RPC] Authentication successful for user %s (ID: %d)", userName, userID)
	return &pb.CommonReply{Code: config.RPCCodeSuccess}, nil
}

// Push 发送单聊消息
func (s *server) Push(ctx context.Context, in *pb.PushRequest) (*pb.CommonReply, error) {
	// 构建消息内容，使用protobuf序列化，后续也要使用protobuf反序列化
	msgBytes, err := proto.Marshal(in)
	if err != nil {
		clog.Error("[RPC] Failed to marshal message: %v", err)
		return &pb.CommonReply{Code: config.RPCCodeFailed}, err
	}

	// 获取目标用户的服务器ID
	userSidKey := fmt.Sprintf("gochat_%d", in.ToUserId)
	instanceIdStr := RedisClient.Get(ctx, userSidKey).Val()

	// 发布消息到队列
	queueMsg := queue.QueueMsg{
		Op:         config.OpSingleSend,
		InstanceId: instanceIdStr,
		Msg:        msgBytes,
		UserId:     int(in.ToUserId),
	}

	if err = queue.DefaultQueue.PublishMessage(&queueMsg); err != nil {
		clog.Error("[RPC] Failed to publish message: %v", err)
		return &pb.CommonReply{Code: config.RPCCodeFailed}, err
	}

	clog.Info("[RPC] Message sent from %s to user %d", in.FromUserName, in.ToUserId)
	return &pb.CommonReply{Code: config.RPCCodeSuccess}, nil
}

// PushRoom 发送群聊消息
func (s *server) PushRoom(ctx context.Context, in *pb.PushRequest) (*pb.CommonReply, error) {
	// 构建消息内容
	msgBytes, err := proto.Marshal(in)
	if err != nil {
		clog.Error("[RPC] Failed to marshal room message: %v", err)
		return &pb.CommonReply{Code: config.RPCCodeFailed}, err
	}

	// 获取房间成员信息
	roomUserKey := fmt.Sprintf("gochat_room_%d", in.RoomId)
	roomUserInfo, err := RedisClient.HGetAll(ctx, roomUserKey).Result()
	if err != nil {
		clog.Error("[RPC] Failed to get room %d users: %v", in.RoomId, err)
		return &pb.CommonReply{Code: config.RPCCodeFailed}, nil
	}

	// 发布消息到队列
	queueMsg := queue.QueueMsg{
		Op:     config.OpRoomSend,
		RoomId: int(in.RoomId),
		Msg:    msgBytes,
	}

	if err = queue.DefaultQueue.PublishMessage(&queueMsg); err != nil {
		clog.Error("[RPC] Failed to publish room message: %v", err)
		return &pb.CommonReply{Code: config.RPCCodeFailed}, err
	}

	clog.Info("[RPC] Room message sent from %s to room %d (%d users)",
		in.FromUserName, in.RoomId, len(roomUserInfo))
	return &pb.CommonReply{Code: config.RPCCodeSuccess}, nil
}

// Connect 处理用户连接
func (s *server) Connect(ctx context.Context, in *pb.ConnectRequest) (*pb.CommonReply, error) {
	// 验证Token
	userID, userName, _, err := tools.ValidateToken(in.Token)
	if err != nil {
		clog.Warning("[RPC] Connect failed: invalid token")
		return &pb.CommonReply{Code: config.RPCCodeFailed}, err
	}

	// 1. 建立用户到服务器的映射
	userSidKey := fmt.Sprintf("gochat_%d", userID)
	if err := RedisClient.Set(ctx, userSidKey, in.InstanceId, 0).Err(); err != nil {
		clog.Error("[RPC] Failed to set user-server mapping: %v", err)
		return &pb.CommonReply{Code: config.RPCCodeFailed}, err
	}

	// 2. 添加用户到房间
	roomUserKey := fmt.Sprintf("gochat_room_%d", in.RoomId)
	if err := RedisClient.HSet(ctx, roomUserKey, fmt.Sprintf("%d", userID), userName).Err(); err != nil {
		clog.Error("[RPC] Failed to add user to room: %v", err)
		return &pb.CommonReply{Code: config.RPCCodeFailed}, err
	}

	// 3. 增加房间在线人数
	_, err = RedisClient.Incr(ctx, fmt.Sprintf("gochat_room_online_count_%d", in.RoomId)).Result()
	if err != nil {
		clog.Error("[RPC] Failed to increment room count: %v", err)
		return &pb.CommonReply{Code: config.RPCCodeFailed}, err
	}

	// 4. 广播房间信息更新
	updateRoomInfo(int(in.RoomId))

	clog.Info("[RPC] User %s (ID: %d) connected to room %d on server %s",
		userName, userID, in.RoomId, in.InstanceId)
	return &pb.CommonReply{Code: config.RPCCodeSuccess}, nil
}

// Disconnect 处理用户断开连接
func (s *server) Disconnect(ctx context.Context, in *pb.DisConnectRequest) (*pb.CommonReply, error) {
	// 1. 删除用户到服务器的映射
	userSidKey := fmt.Sprintf("gochat_%d", in.UserId)
	if err := RedisClient.Del(ctx, userSidKey).Err(); err != nil {
		clog.Error("[RPC] Failed to delete user-server mapping: %v", err)
		return &pb.CommonReply{Code: config.RPCCodeFailed}, err
	}

	// 2. 从房间成员中移除用户
	roomUserKey := fmt.Sprintf("gochat_room_%d", in.RoomId)
	userIDStr := fmt.Sprintf("%d", in.UserId)
	if _, err := RedisClient.HDel(ctx, roomUserKey, userIDStr).Result(); err != nil {
		clog.Error("[RPC] Failed to remove user from room: %v", err)
		return &pb.CommonReply{Code: config.RPCCodeFailed}, err
	}

	// 3. 减少房间在线人数
	_, err := RedisClient.Decr(ctx, fmt.Sprintf("gochat_room_online_count_%d", in.RoomId)).Result()
	if err != nil {
		clog.Error("[RPC] Failed to decrement room count: %v", err)
		return &pb.CommonReply{Code: config.RPCCodeFailed}, err
	}

	// 4. 广播房间信息更新
	updateRoomInfo(int(in.RoomId))

	clog.Info("[RPC] User %d disconnected from room %d", in.UserId, in.RoomId)
	return &pb.CommonReply{Code: config.RPCCodeSuccess}, nil
}

// updateRoomInfo 更新并广播房间信息
func updateRoomInfo(roomID int) error {
	roomUserKey := fmt.Sprintf("gochat_room_%d", roomID)
	roomUserInfo, err := RedisClient.HGetAll(context.Background(), roomUserKey).Result()
	if err != nil {
		clog.Error("[RPC] Failed to get room users for update: %v", err)
		return err
	}

	// 发送房间成员信息更新
	infoQueueMsg := queue.QueueMsg{
		Op:           config.OpRoomInfoSend,
		RoomId:       roomID,
		Count:        len(roomUserInfo),
		RoomUserInfo: roomUserInfo,
	}
	if err := queue.DefaultQueue.PublishMessage(&infoQueueMsg); err != nil {
		clog.Error("[RPC] Failed to publish room info update: %v", err)
		return err
	}

	// 发送房间人数更新
	// countQueueMsg := queue.QueueMsg{
	// 	Op:     config.OpRoomCountSend,
	// 	RoomId: roomID,
	// 	Count:  len(roomUserInfo),
	// }
	// if err := queue.DefaultQueue.PublishMessage(&countQueueMsg); err != nil {
	// 	clog.Error("[RPC] Failed to publish room count update: %v", err)
	// 	return err
	// }

	clog.Debug("[RPC] Room %d info updated: %d users", roomID, len(roomUserInfo))
	return nil
}
