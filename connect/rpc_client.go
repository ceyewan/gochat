package connect

import (
	"context"
	"gochat/clog"
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

func (l *LogicRPC) Connect(authToken, serverID string, roomID int) (int, error) {
	clog.Info("RPC Connect, serverID: %s", serverID)
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	req := &logicproto.ConnectRequest{
		ServerId:  serverID,
		RoomId:    int32(roomID),
		AuthToken: authToken,
	}
	reply, err := LogicClient.Connect(ctx, req)
	if err != nil {
		clog.Error("RPC Connect failed: %v", err)
		return 0, err
	}
	clog.Info("RPC Connect success, uid: %d", reply.UserId)
	return int(reply.UserId), nil
}

func (l *LogicRPC) Disconnect(userID, roomID int) error {
	clog.Info("RPC Disconnect, userID: %d", userID)
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	req := &logicproto.DisConnectRequest{
		UserId: int32(userID),
		RoomId: int32(roomID),
	}
	_, err := LogicClient.DisConnect(ctx, req)
	if err != nil {
		clog.Error("RPC Disconnect failed: %v", err)
		return err
	}
	clog.Info("RPC Disconnect success")
	return nil
}

