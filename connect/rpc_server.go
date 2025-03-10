package connect

import (
	"context"
	"encoding/json"
	"fmt"
	"gochat/clog"
	"gochat/config"
	pb "gochat/proto/connectproto"
	"gochat/tools"
	"net"
	"strings"

	"google.golang.org/grpc"
)

// 全局连接管理器
var connectionManager *ConnectionManager

// RPC服务实现
type server struct {
	pb.UnimplementedConnectServiceServer
	connMgr *ConnectionManager
}

// 确保server实现了ConnectServiceServer接口
var _ pb.ConnectServiceServer = (*server)(nil)

// InitRPCServer 初始化并启动RPC服务器
func InitRPCServer(ctx context.Context) (*grpc.Server, error) {
	// 创建连接管理器
	connectionManager = NewConnectionManager()

	// 监听指定端口
	lis, err := net.Listen("tcp", fmt.Sprintf(":%d", config.Conf.RPC.Port+1))
	if err != nil {
		clog.Error("Failed to listen on port %d: %v", config.Conf.RPC.Port+1, err)
		return nil, err
	}

	// 创建gRPC服务器实例
	s := grpc.NewServer()

	// 注册connect服务
	svr := &server{connMgr: connectionManager}
	pb.RegisterConnectServiceServer(s, svr)

	// 生成实例ID和本机IP地址
	instanceID := DefaultWSServer.InstanceID
	splitInstanceID := strings.Split(instanceID, "-")
	addr := fmt.Sprintf("%s:%d", splitInstanceID[len(splitInstanceID)-1], config.Conf.RPC.Port+1)

	// 服务注册上下文
	serviceCtx, cancel := context.WithCancel(context.Background())

	// 注册服务到etcd
	go func() {
		if err := tools.ServiceRegistry(serviceCtx, "connect-service", instanceID, addr); err != nil {
			clog.Error("Failed to register service: %v", err)
			cancel()
			return
		}
		clog.Info("Service registered with etcd: connect-service/%s at %s", instanceID, addr)
	}()

	// 启动gRPC服务器
	go func() {
		clog.Info("Connect RPC server starting on port %d", config.Conf.RPC.Port+1)
		if err := s.Serve(lis); err != nil {
			clog.Error("Failed to serve: %v", err)
		}
	}()

	return s, nil
}

// PushSingleMsg 向单个用户推送消息
func (s *server) PushSingleMsg(ctx context.Context, in *pb.PushSingleMsgRequest) (*pb.SuccessReply, error) {
	userID := int(in.UserId)
	ch, exists := s.connMgr.GetUser(userID)

	if !exists {
		clog.Debug("User %d not connected", userID)
		return &pb.SuccessReply{
			Code: config.RPCCodeFailed,
		}, fmt.Errorf("user %d not connected", userID)
	}
	// 将 in.Msg 使用 json.Unmarshal 解析为 ChatMessage 结构体
	var chatMsg tools.ChatMessage
	if err := json.Unmarshal(in.Msg, &chatMsg); err != nil {
		clog.Error("Failed to unmarshal chat message: %v", err)
		return &pb.SuccessReply{
			Code: config.RPCCodeFailed,
		}, err
	}
	msgDataBytes, _ := json.Marshal(NewMessageData(&chatMsg))
	ch.send <- msgDataBytes
	clog.Debug("Message pushed to user %d", userID)
	return &pb.SuccessReply{Code: config.RPCCodeSuccess}, nil
}

// PushRoomMsg 向房间内所有用户推送消息
func (s *server) PushRoomMsg(ctx context.Context, in *pb.PushRoomMsgRequest) (*pb.SuccessReply, error) {
	roomID := int(in.RoomId)
	// 将 in.Msg 使用 json.Unmarshal 解析为 ChatMessage 结构体
	var chatMsg tools.ChatMessage
	if err := json.Unmarshal(in.Msg, &chatMsg); err != nil {
		clog.Error("Failed to unmarshal chat message: %v", err)
		return &pb.SuccessReply{
			Code: config.RPCCodeFailed,
		}, err
	}
	msgDataBytes, _ := json.Marshal(NewMessageData(&chatMsg))
	if !s.connMgr.BroadcastToRoom(roomID, msgDataBytes) {
		clog.Warning("Room %d not found", roomID)
		return &pb.SuccessReply{
			Code: config.RPCCodeFailed,
		}, fmt.Errorf("room %d not found", roomID)
	}

	clog.Debug("Message pushed to room %d", roomID)
	return &pb.SuccessReply{Code: config.RPCCodeSuccess}, nil
}

// PushRoomInfo 向房间推送用户信息
func (s *server) PushRoomInfo(ctx context.Context, in *pb.PushRoomInfoRequest) (*pb.SuccessReply, error) {
	roomID := int(in.RoomId)

	// 将 in.Msg 使用 json.Unmarshal 解析为 RoomInfo 结构体
	var roomInfo tools.RoomInfo
	if err := json.Unmarshal(in.Info, &roomInfo); err != nil {
		clog.Error("Failed to unmarshal room info: %v", err)
		return &pb.SuccessReply{
			Code: config.RPCCodeFailed,
		}, err
	}
	roomInfobytes, _ := json.Marshal(NewRoomInfoData(&roomInfo))

	// 广播到房间
	if !s.connMgr.BroadcastToRoom(roomID, roomInfobytes) {
		clog.Warning("Room %d not found for info update", roomID)
		return &pb.SuccessReply{
			Code: config.RPCCodeFailed,
		}, fmt.Errorf("room %d not found for info update", roomID)
	}

	clog.Debug("Room info pushed to room %d", roomID)
	return &pb.SuccessReply{Code: config.RPCCodeSuccess}, nil
}
