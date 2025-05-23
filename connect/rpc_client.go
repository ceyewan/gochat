package connect

// todo 修改 PRC 逻辑
import (
	"context"
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
			clog.Module("connect").Errorf("[RPC] Service discovery failed: %v", err)
			return
		}

		// 初始化客户端
		LogicClient = logicproto.NewChatLogicServiceClient(conn)
		LogicRPCObj = &LogicRPC{}
		clog.Module("connect").Infof("[RPC] Logic client initialized successfully")
	})
}

// createContext 创建带超时的上下文
func createContext() (context.Context, context.CancelFunc) {
	return context.WithTimeout(context.Background(), defaultTimeout)
}

// Connect 处理用户连接
func (l *LogicRPC) Connect(authToken, instanceID string, userID, roomID int) error {
	clog.Module("connect").Infof("[RPC] Connect, instanceID: %s", instanceID)
	ctx, cancel := createContext()
	defer cancel()

	req := &logicproto.ConnectRequest{
		UserId:     int32(userID),
		InstanceId: instanceID,
		RoomId:     int32(roomID),
		Token:      authToken,
	}

	reply, err := LogicClient.Connect(ctx, req)
	if err != nil {
		clog.Module("connect").Errorf("[RPC] Connect failed: %v", err)
		return err
	}

	clog.Module("connect").Infof("[RPC] Connect success, code: %d", reply.Code)
	return nil
}

// Disconnect 处理用户断开连接
func (l *LogicRPC) Disconnect(userID, roomID int) error {
	clog.Module("connect").Infof("[RPC] Disconnect, userID: %d", userID)
	ctx, cancel := createContext()
	defer cancel()

	req := &logicproto.DisConnectRequest{
		UserId: int32(userID),
		RoomId: int32(roomID),
	}

	reply, err := LogicClient.DisConnect(ctx, req)
	if err != nil {
		clog.Module("connect").Errorf("[RPC] Disconnect failed: %v", err)
		return err
	}

	clog.Module("connect").Infof("[RPC] Disconnect success, code: %d", reply.Code)
	return nil
}
