package task

import (
	"context"
	"encoding/json"
	"gochat/clog"
	"gochat/config"
	"gochat/proto/connectproto"
	"gochat/tools"
	"time"
)

// pushSingleToConnect 向指定服务器上的指定用户发送消息
func (t *Task) pushSingleToConnect(serverID string, userID int, msg []byte) {
	// 获取指定服务器的gRPC连接
	conn, err := tools.GetServiceInstanceConn("connect-service", serverID)
	if err != nil {
		clog.Error("获取连接失败, serverID: %s, err: %v", serverID, err)
		return
	}

	// 创建gRPC客户端
	client := connectproto.NewConnectServiceClient(conn)

	// 创建请求
	req := &connectproto.PushMsgRequest{
		UserId: int32(userID),
		Msg: &connectproto.Msg{
			Ver:       1,
			Operation: 2,
			SeqId:     tools.GetSnowflakeID(),
			Body:      msg,
		},
	}
	// 设置超时上下文
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	// 调用RPC方法
	reply, err := client.PushSingleMsg(ctx, req)
	if err != nil {
		clog.Error("推送单聊消息失败, serverID: %s, userID: %d, err: %v", serverID, userID, err)
		return
	}
	clog.Info("推送单聊消息成功, serverID: %s, userID: %d, reply: %v", serverID, userID, reply)
}

// broadcastRoomToConnect 向指定房间广播消息
func (t *Task) broadcastRoomToConnect(roomID int, msg []byte) {
	// 获取所有connect-service服务实例的连接
	conns, err := tools.GetAllServiceInstanceConns("connect-service")
	if err != nil {
		clog.Error("获取所有连接失败: %v", err)
		return
	}

	if len(conns) == 0 {
		clog.Warning("没有可用的connect-service实例")
		return
	}

	// 创建请求
	req := &connectproto.PushRoomMsgRequest{
		RoomId: int32(roomID),
		Msg: &connectproto.Msg{
			Ver:       1,
			Operation: config.OpRoomSend,
			SeqId:     tools.GetSnowflakeID(),
			Body:      msg,
		},
	}

	// 向所有服务实例广播房间消息
	for serverID, conn := range conns {
		client := connectproto.NewConnectServiceClient(conn)

		// 设置超时上下文
		ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)

		// 调用RPC方法
		reply, err := client.PushRoomMsg(ctx, req)
		cancel() // 确保在每次循环中取消上下文

		if err != nil {
			clog.Error("向服务器广播房间消息失败, serverID: %s, roomID: %d, err: %v", serverID, roomID, err)
			continue
		}

		clog.Info("向服务器广播房间消息成功, serverID: %s, roomID: %d, reply: %v", serverID, roomID, reply)
	}
}

// broadcastRoomCountToConnect 广播房间在线用户数量
func (t *Task) broadcastRoomCountToConnect(roomID int, count int) {
	// 获取所有connect-service服务实例的连接
	conns, err := tools.GetAllServiceInstanceConns("connect-service")
	if err != nil {
		clog.Error("获取所有连接失败: %v", err)
		return
	}

	if len(conns) == 0 {
		clog.Warning("没有可用的connect-service实例")
		return
	}

	// 创建房间人数消息
	roomCountMsg := &connectproto.RedisRoomCountMsg{
		Count: int32(count),
		Op:    int32(config.OpRoomCountSend),
	}

	// 序列化消息
	roomCountBytes, err := json.Marshal(roomCountMsg)
	if err != nil {
		clog.Error("序列化房间人数消息失败: %v", err)
		return
	}

	// 创建请求
	req := &connectproto.PushRoomMsgRequest{
		RoomId: int32(roomID),
		Msg: &connectproto.Msg{
			Ver:       1,
			Operation: int32(config.OpRoomCountSend),
			SeqId:     tools.GetSnowflakeID(),
			Body:      roomCountBytes,
		},
	}

	// 向所有服务实例广播房间人数
	for serverID, conn := range conns {
		client := connectproto.NewConnectServiceClient(conn)

		// 设置超时上下文
		ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)

		// 调用RPC方法
		reply, err := client.PushRoomCount(ctx, req)
		cancel() // 确保在每次循环中取消上下文

		if err != nil {
			clog.Error("向服务器广播房间人数失败, serverID: %s, roomID: %d, err: %v", serverID, roomID, err)
			continue
		}

		clog.Info("向服务器广播房间人数成功, serverID: %s, roomID: %d, count: %d, reply: %v",
			serverID, roomID, count, reply)
	}
}

// broadcastRoomInfoToConnect 广播房间信息
func (t *Task) broadcastRoomInfoToConnect(roomID int, roomUserInfo map[string]string) {
	// 获取所有connect-service服务实例的连接
	conns, err := tools.GetAllServiceInstanceConns("connect-service")
	if err != nil {
		clog.Error("获取所有连接失败: %v", err)
		return
	}

	if len(conns) == 0 {
		clog.Warning("没有可用的connect-service实例")
		return
	}

	// 创建房间信息消息
	roomInfoMsg := &connectproto.RedisRoomInfo{
		Count:        int32(len(roomUserInfo)),
		Op:           int32(config.OpRoomInfoSend),
		RoomId:       int32(roomID),
		RoomUserInfo: roomUserInfo,
	}

	// 序列化消息
	roomInfoBytes, err := json.Marshal(roomInfoMsg)
	if err != nil {
		clog.Error("序列化房间信息消息失败: %v", err)
		return
	}

	// 创建请求
	req := &connectproto.PushRoomMsgRequest{
		RoomId: int32(roomID),
		Msg: &connectproto.Msg{
			Ver:       1,
			Operation: int32(config.OpRoomInfoSend),
			SeqId:     tools.GetSnowflakeID(),
			Body:      roomInfoBytes,
		},
	}

	// 向所有服务实例广播房间信息
	for serverID, conn := range conns {
		client := connectproto.NewConnectServiceClient(conn)

		// 设置超时上下文
		ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)

		// 调用RPC方法
		reply, err := client.PushRoomInfo(ctx, req)
		cancel() // 确保在每次循环中取消上下文

		if err != nil {
			clog.Error("向服务器广播房间信息失败, serverID: %s, roomID: %d, err: %v", serverID, roomID, err)
			continue
		}

		clog.Info("向服务器广播房间信息成功, serverID: %s, roomID: %d, reply: %v", serverID, roomID, reply)
	}
}
