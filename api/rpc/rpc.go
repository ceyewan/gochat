package rpc

import (
	"context"
	"gochat/api/dto"
	"sync"
	"time"

	"gochat/clog"
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
			clog.Module("rpc").Errorf("[RPC] Service discovery failed: %v", err)
			return
		}

		// 初始化客户端
		LogicClient = logicproto.NewChatLogicServiceClient(conn)
		LogicRPCObj = &LogicRPC{}
		clog.Module("rpc").Info("[RPC] Logic client initialized successfully")
	})
}

// createContext 创建带超时的上下文
func createContext() (context.Context, context.CancelFunc) {
	return context.WithTimeout(context.Background(), defaultTimeout)
}

// Login 处理用户登录
func (l *LogicRPC) Login(name, password string) (int, string, string, error) {
	ctx, cancel := createContext()
	defer cancel()

	req := &logicproto.LoginRequest{
		Name:     name,
		Password: password,
	}
	reply, err := LogicClient.Login(ctx, req)
	if err != nil {
		clog.Module("rpc").Errorf("[RPC] Login failed: %v", err)
		return 0, "", "", err
	}

	clog.Module("rpc").Infof("[RPC] Login success: name=%s, id=%d", name, reply.UserId)
	return int(reply.UserId), reply.UserName, reply.Token, nil
}

// Register 处理用户注册
func (l *LogicRPC) Register(name, password string) error {
	ctx, cancel := createContext()
	defer cancel()

	req := &logicproto.RegisterRequest{
		Name:     name,
		Password: password,
	}

	reply, err := LogicClient.Register(ctx, req)
	if err != nil {
		clog.Module("rpc").Errorf("[RPC] Register failed: %v", err)
		return err
	}

	clog.Module("rpc").Infof("[RPC] Register success: name=%s, code=%d", name, reply.Code)
	return nil
}

// Logout 处理用户登出
func (l *LogicRPC) Logout(Token string) error {
	ctx, cancel := createContext()
	defer cancel()

	req := &logicproto.LogoutRequest{
		Token: Token,
	}

	reply, err := LogicClient.Logout(ctx, req)
	if err != nil {
		clog.Module("rpc").Errorf("[RPC] Logout failed: %v", err)
		return err
	}

	clog.Module("rpc").Infof("[RPC] Logout success: code=%d", reply.Code)
	return nil
}

// CheckAuth 检查用户认证状态
func (l *LogicRPC) CheckAuth(Token string) error {
	ctx, cancel := createContext()
	defer cancel()

	req := &logicproto.CheckAuthRequest{
		Token: Token,
	}

	reply, err := LogicClient.CheckAuth(ctx, req)
	if err != nil {
		clog.Module("rpc").Errorf("[RPC] CheckAuth failed: %v", err)
		return err
	}

	clog.Module("rpc").Debugf("[RPC] CheckAuth token: %v success, code=%d", Token, reply.Code)
	return nil
}

// Push 发送单聊消息
func (l *LogicRPC) Push(args *dto.PushRequest) error {
	ctx, cancel := createContext()
	defer cancel()

	req := &logicproto.PushRequest{
		FromUserId:   int32(args.FromUserID),
		FromUserName: args.FromUserName,
		ToUserId:     int32(args.ToUserID),
		ToUserName:   args.ToUserName,
		RoomId:       int32(args.RoomID),
		CreateTime:   getCurrentTime(),
		Msg:          args.Msg,
	}

	reply, err := LogicClient.Push(ctx, req)
	if err != nil {
		clog.Module("rpc").Errorf("[RPC] Push failed: from=%d, to=%d, error=%v", args.FromUserID, args.ToUserID, err)
		return err
	}

	clog.Module("rpc").Infof("[RPC] Push success: from=%d, to=%d, code=%d", args.FromUserID, args.ToUserID, reply.Code)
	return nil
}

// PushRoom 发送群聊消息
func (l *LogicRPC) PushRoom(args *dto.PushRequest) error {
	ctx, cancel := createContext()
	defer cancel()

	req := &logicproto.PushRequest{
		FromUserId:   int32(args.FromUserID),
		FromUserName: args.FromUserName,
		ToUserId:     int32(args.ToUserID),
		ToUserName:   args.ToUserName,
		RoomId:       int32(args.RoomID),
		CreateTime:   getCurrentTime(),
		Msg:          args.Msg,
	}

	reply, err := LogicClient.PushRoom(ctx, req)
	if err != nil {
		clog.Module("rpc").Errorf("[RPC] PushRoom failed: from=%d, to=%d, error=%v", args.FromUserID, args.ToUserID, err)
		return err
	}

	clog.Module("rpc").Infof("[RPC] PushRoom success: from=%d, to=%d, code=%d", args.FromUserID, args.ToUserID, reply.Code)
	return nil
}

// 获取当前时间的格式化字符串
func getCurrentTime() string {
	return time.Now().Format("2006-01-02 15:04:05")
}
