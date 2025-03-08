package rpc

import (
	"context"
	"gochat/clog"
	"gochat/config"
	"gochat/proto/logicproto"
	"gochat/tools"
	"sync"
	"time"
)

// LogicClient 是与逻辑服务通信的全局 gRPC 客户端
var LogicClient logicproto.ChatLogicServiceClient

// once 用于确保 RPC 客户端只被初始化一次
var once sync.Once

// LogicRPC 结构体封装了所有与逻辑服务器交互的 RPC 方法
type LogicRPC struct{}

// LogicRPCObj 是 LogicRPC 的实例
var LogicRPCObj *LogicRPC

// InitLogicRPCClient 初始化逻辑服务的 RPC 客户端
func InitLogicRPCClient() {
	once.Do(func() {
		// 使用服务发现机制连接到逻辑服务
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		conn, err := tools.ServiceDiscovery(ctx, "logic-service") // 服务名应与逻辑服务注册时使用的名称一致
		if err != nil {
			clog.Error("服务发现连接失败: %v", err)
			return
		}
		// 初始化gRPC客户端
		LogicClient = logicproto.NewChatLogicServiceClient(conn)
		clog.Info("成功初始化Logic RPC客户端")
		// 初始化logicRPC实例
		LogicRPCObj = &LogicRPC{}
	})
}

// Login 处理用户登录请求
func (l *LogicRPC) Login(name, password string) (int, string) {
	clog.Info("RPC Login, name: %s, password: %s", name, password)
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	req := &logicproto.LoginRequest{
		Name:     name,
		Password: password,
	}
	reply, err := LogicClient.Login(ctx, req)
	if err != nil {
		clog.Error("RPC Login failed: %v", err)
		return config.RPCCodeFailed, ""
	}
	clog.Info("RPC Login success, code: %d, authToken: %s", reply.Code, reply.AuthToken)
	return int(reply.Code), reply.AuthToken
}

// Register 处理用户注册请求
func (l *LogicRPC) Register(name, password string) (int, string) {
	clog.Info("RPC Register, name: %s, password: %s", name, password)
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	req := &logicproto.RegisterRequest{
		Name:     name,
		Password: password,
	}
	reply, err := LogicClient.Register(ctx, req)
	if err != nil {
		clog.Error("RPC Register failed: %v", err)
		return config.RPCCodeFailed, ""
	}
	clog.Info("RPC Register success, code: %d, authToken: %s", reply.Code, reply.AuthToken)
	return int(reply.Code), reply.AuthToken
}

// Logout 处理用户登出请求
func (l *LogicRPC) Logout(authToken string) int {
	clog.Info("RPC Logout, authToken: %s", authToken)
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	req := &logicproto.LogoutRequest{
		AuthToken: authToken,
	}
	reply, err := LogicClient.Logout(ctx, req)
	if err != nil {
		clog.Error("RPC Logout failed: %v", err)
		return config.RPCCodeFailed
	}
	clog.Info("RPC Logout success, code: %d", reply.Code)
	return int(reply.Code)
}

// CheckAuth 检查用户身份验证令牌
func (l *LogicRPC) CheckAuth(authToken string) (int, int, string) {
	clog.Info("RPC CheckAuth, authToken: %s", authToken)
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	req := &logicproto.CheckAuthRequest{
		AuthToken: authToken,
	}
	reply, err := LogicClient.CheckAuth(ctx, req)
	if err != nil {
		clog.Error("RPC CheckAuth failed: %v", err)
		return config.RPCCodeFailed, 0, ""
	}
	clog.Info("RPC CheckAuth success, code: %d, userID: %d, userName: %s", reply.Code, reply.UserId, reply.UserName)
	return int(reply.Code), int(reply.UserId), reply.UserName
}

// GetUserNameByID 通过用户 ID 获取用户名
func (l *LogicRPC) GetUserNameByID(userID int) (int, string) {
	clog.Info("RPC GetUserNameByID, userID: %d", userID)
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	req := &logicproto.GetUserInfoRequest{
		UserId: int32(userID),
	}
	reply, err := LogicClient.GetUserInfoByUserId(ctx, req)
	if err != nil {
		clog.Error("RPC GetUserNameByID failed: %v", err)
		return config.RPCCodeFailed, ""
	}
	clog.Info("RPC GetUserNameByID success, code: %d, userName: %s", reply.Code, reply.UserName)
	return int(reply.Code), reply.UserName
}

// Push 发送单聊消息
func (l *LogicRPC) Push(fromUserID, toUserID, roomID, Op int, msg, fromUserName, toUserName, createTime string) int {
	clog.Info("RPC Push, fromUserID: %d, toUserID: %d, msg: %s", fromUserID, toUserID, msg)
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	req := &logicproto.Send{
		FromUserId:   int32(fromUserID),
		FromUserName: fromUserName,
		ToUserId:     int32(toUserID),
		ToUserName:   toUserName,
		RoomId:       int32(roomID),
		Op:           int32(Op),
		CreateTime:   createTime,
		Msg:          msg,
	}
	reply, err := LogicClient.Push(ctx, req)
	if err != nil {
		clog.Error("RPC Push failed: %v", err)
		return config.RPCCodeFailed
	}
	clog.Info("RPC Push success, code: %d", reply.Code)
	return int(reply.Code)
}

// PushRoom 发送群聊消息
func (l *LogicRPC) PushRoom(fromUserID, roomID, Op int, msg, fromUserName, createTime string) int {
	clog.Info("RPC PushRoom, fromUserID: %d, roomID: %d, msg: %s", fromUserID, roomID, msg)
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	req := &logicproto.Send{
		FromUserId:   int32(fromUserID),
		FromUserName: fromUserName,
		RoomId:       int32(roomID),
		Op:           int32(Op),
		CreateTime:   createTime,
		Msg:          msg,
	}
	reply, err := LogicClient.PushRoom(ctx, req)
	if err != nil {
		clog.Error("RPC PushRoom failed: %v", err)
		return config.RPCCodeFailed
	}
	clog.Info("RPC PushRoom success, code: %d", reply.Code)
	return int(reply.Code)
}

// Count 获取房间在线用户数量
func (l *LogicRPC) Count(roomID, Op int) int {
	clog.Info("RPC Count, roomID: %d", roomID)
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	req := &logicproto.Send{
		RoomId: int32(roomID),
		Op:     int32(Op),
	}
	reply, err := LogicClient.Count(ctx, req)
	if err != nil {
		clog.Error("RPC Count failed: %v", err)
		return config.RPCCodeFailed
	}
	clog.Info("RPC Count success, code: %d", reply.Code)
	return int(reply.Code)
}

// GetRoomInfo 获取房间信息
func (l *LogicRPC) GetRoomInfo(roomID, Op int) int {
	clog.Info("RPC GetRoomInfo, roomID: %d", roomID)
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	req := &logicproto.Send{
		RoomId: int32(roomID),
		Op:     int32(Op),
	}
	reply, err := LogicClient.GetRoomInfo(ctx, req)
	if err != nil {
		clog.Error("RPC GetRoomInfo failed: %v", err)
		return config.RPCCodeFailed
	}
	clog.Info("RPC GetRoomInfo success, code: %d", reply.Code)
	return int(reply.Code)
}
