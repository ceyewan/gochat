package rpc

import (
	"context"
	"sync"
	"time"

	"gochat/clog"
	"gochat/config"
	"gochat/proto/logicproto"
	"gochat/tools"
)

// LogicRPC 封装与逻辑服务交互的方法
type LogicRPC struct{}

var (
	// LogicClient 与逻辑服务通信的gRPC客户端
	LogicClient logicproto.ChatLogicServiceClient

	// LogicRPCObj 是LogicRPC的全局实例
	LogicRPCObj *LogicRPC

	// once 确保RPC客户端只初始化一次
	once sync.Once

	// 默认RPC超时时间
	defaultTimeout = 5 * time.Second
)

// InitLogicRPCClient 初始化逻辑服务RPC客户端
func InitLogicRPCClient() {
	once.Do(func() {
		// 使用服务发现连接逻辑服务
		ctx, cancel := context.WithTimeout(context.Background(), defaultTimeout)
		defer cancel()

		conn, err := tools.ServiceDiscovery(ctx, "logic-service")
		if err != nil {
			clog.Error("[RPC] Service discovery failed: %v", err)
			return
		}

		// 初始化客户端
		LogicClient = logicproto.NewChatLogicServiceClient(conn)
		LogicRPCObj = &LogicRPC{}
		clog.Info("[RPC] Logic client initialized successfully")
	})
}

// createContext 创建带超时的上下文
func createContext() (context.Context, context.CancelFunc) {
	return context.WithTimeout(context.Background(), defaultTimeout)
}

// Login 处理用户登录
func (l *LogicRPC) Login(name, password string) (int, string) {
	ctx, cancel := createContext()
	defer cancel()

	req := &logicproto.LoginRequest{
		Name:     name,
		Password: password,
	}

	reply, err := LogicClient.Login(ctx, req)
	if err != nil {
		clog.Error("[RPC] Login failed: %v", err)
		return config.RPCCodeFailed, ""
	}

	clog.Info("[RPC] Login success: name=%s, code=%d", name, reply.Code)
	return int(reply.Code), reply.AuthToken
}

// Register 处理用户注册
func (l *LogicRPC) Register(name, password string) int {
	ctx, cancel := createContext()
	defer cancel()

	req := &logicproto.RegisterRequest{
		Name:     name,
		Password: password,
	}

	reply, err := LogicClient.Register(ctx, req)
	if err != nil {
		clog.Error("[RPC] Register failed: %v", err)
		return config.RPCCodeFailed
	}

	clog.Info("[RPC] Register success: name=%s, code=%d", name, reply.Code)
	return int(reply.Code)
}

// Logout 处理用户登出
func (l *LogicRPC) Logout(authToken string) int {
	ctx, cancel := createContext()
	defer cancel()

	req := &logicproto.LogoutRequest{
		AuthToken: authToken,
	}

	reply, err := LogicClient.Logout(ctx, req)
	if err != nil {
		clog.Error("[RPC] Logout failed: %v", err)
		return config.RPCCodeFailed
	}

	clog.Info("[RPC] Logout success: code=%d", reply.Code)
	return int(reply.Code)
}

// CheckAuth 检查用户认证状态
func (l *LogicRPC) CheckAuth(authToken string) (int, int, string) {
	ctx, cancel := createContext()
	defer cancel()

	req := &logicproto.CheckAuthRequest{
		AuthToken: authToken,
	}

	reply, err := LogicClient.CheckAuth(ctx, req)
	if err != nil {
		clog.Error("[RPC] CheckAuth failed: %v", err)
		return config.RPCCodeFailed, 0, ""
	}

	clog.Debug("[RPC] CheckAuth success: userID=%d, userName=%s", reply.UserId, reply.UserName)
	return int(reply.Code), int(reply.UserId), reply.UserName
}

// GetUserNameByID 通过用户ID获取用户名
func (l *LogicRPC) GetUserNameByID(userID int) (int, string) {
	ctx, cancel := createContext()
	defer cancel()

	req := &logicproto.GetUserInfoRequest{
		UserId: int32(userID),
	}

	reply, err := LogicClient.GetUserInfoByUserId(ctx, req)
	if err != nil {
		clog.Error("[RPC] GetUserNameByID failed: userID=%d, error=%v", userID, err)
		return config.RPCCodeFailed, ""
	}

	clog.Debug("[RPC] GetUserNameByID success: userID=%d, userName=%s", userID, reply.UserName)
	return int(reply.Code), reply.UserName
}

// Push 发送单聊消息
func (l *LogicRPC) Push(fromUserID, toUserID, roomID, Op int, msg, fromUserName, toUserName, createTime string) int {
	ctx, cancel := createContext()
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
		clog.Error("[RPC] Push failed: from=%d, to=%d, error=%v", fromUserID, toUserID, err)
		return config.RPCCodeFailed
	}

	clog.Info("[RPC] Push success: from=%d, to=%d", fromUserID, toUserID)
	return int(reply.Code)
}

// PushRoom 发送群聊消息
func (l *LogicRPC) PushRoom(fromUserID, roomID, Op int, msg, fromUserName, createTime string) int {
	ctx, cancel := createContext()
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
		clog.Error("[RPC] PushRoom failed: from=%d, room=%d, error=%v", fromUserID, roomID, err)
		return config.RPCCodeFailed
	}

	clog.Info("[RPC] PushRoom success: from=%d, room=%d", fromUserID, roomID)
	return int(reply.Code)
}
