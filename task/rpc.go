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
func (t *Task) pushSingleToConnect(instanceID string, userID int, msg []byte) {
	conn, err := tools.GetServiceInstanceConn("connect-service", instanceID)
	if err != nil {
		clog.Error("Failed to get connection, serverID: %s, err: %v", instanceID, err)
		return
	}

	client := connectproto.NewConnectServiceClient(conn)
	req := &connectproto.PushMsgRequest{
		UserId: int32(userID),
		Msg: &connectproto.Msg{
			Ver:       1,
			Operation: 2,
			SeqId:     tools.GetSnowflakeID(),
			Body:      msg,
		},
	}

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	reply, err := client.PushSingleMsg(ctx, req)
	if err != nil {
		clog.Error("Failed to push single message, serverID: %s, userID: %d, err: %v", instanceID, userID, err)
		return
	}
	clog.Info("Successfully pushed single message, serverID: %s, userID: %d, reply: %v", instanceID, userID, reply)
}

// broadcastRoomToConnect 向指定房间广播消息
func (t *Task) broadcastRoomToConnect(roomID int, msg []byte) {
	conns, err := tools.GetAllServiceInstanceConns("connect-service")
	if err != nil {
		clog.Error("Failed to get all connections: %v", err)
		return
	}

	if len(conns) == 0 {
		clog.Warning("No available connect-service instances")
		return
	}

	req := &connectproto.PushRoomMsgRequest{
		RoomId: int32(roomID),
		Msg: &connectproto.Msg{
			Ver:       1,
			Operation: config.OpRoomSend,
			SeqId:     tools.GetSnowflakeID(),
			Body:      msg,
		},
	}

	for serverID, conn := range conns {
		client := connectproto.NewConnectServiceClient(conn)
		ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
		reply, err := client.PushRoomMsg(ctx, req)
		cancel()

		if err != nil {
			clog.Error("Failed to broadcast room message, serverID: %s, roomID: %d, err: %v", serverID, roomID, err)
			continue
		}

		clog.Info("Successfully broadcasted room message, serverID: %s, roomID: %d, reply: %v", serverID, roomID, reply)
	}
}

// broadcastRoomCountToConnect 广播房间在线用户数量
func (t *Task) broadcastRoomCountToConnect(roomID int, count int) {
	conns, err := tools.GetAllServiceInstanceConns("connect-service")
	if err != nil {
		clog.Error("Failed to get all connections: %v", err)
		return
	}

	if len(conns) == 0 {
		clog.Warning("No available connect-service instances")
		return
	}

	roomCountMsg := &connectproto.RedisRoomCountMsg{
		Count: int32(count),
		Op:    int32(config.OpRoomCountSend),
	}

	roomCountBytes, err := json.Marshal(roomCountMsg)
	if err != nil {
		clog.Error("Failed to serialize room count message: %v", err)
		return
	}

	req := &connectproto.PushRoomMsgRequest{
		RoomId: int32(roomID),
		Msg: &connectproto.Msg{
			Ver:       1,
			Operation: int32(config.OpRoomCountSend),
			SeqId:     tools.GetSnowflakeID(),
			Body:      roomCountBytes,
		},
	}

	for serverID, conn := range conns {
		client := connectproto.NewConnectServiceClient(conn)
		ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
		reply, err := client.PushRoomCount(ctx, req)
		cancel()

		if err != nil {
			clog.Error("Failed to broadcast room count, serverID: %s, roomID: %d, err: %v", serverID, roomID, err)
			continue
		}

		clog.Info("Successfully broadcasted room count, serverID: %s, roomID: %d, count: %d, reply: %v", serverID, roomID, count, reply)
	}
}

// broadcastRoomInfoToConnect 广播房间信息
func (t *Task) broadcastRoomInfoToConnect(roomID int, roomUserInfo map[string]string) {
	conns, err := tools.GetAllServiceInstanceConns("connect-service")
	if err != nil {
		clog.Error("Failed to get all connections: %v", err)
		return
	}

	if len(conns) == 0 {
		clog.Warning("No available connect-service instances")
		return
	}

	roomInfoMsg := &connectproto.RedisRoomInfo{
		Count:        int32(len(roomUserInfo)),
		Op:           int32(config.OpRoomInfoSend),
		RoomId:       int32(roomID),
		RoomUserInfo: roomUserInfo,
	}

	roomInfoBytes, err := json.Marshal(roomInfoMsg)
	if err != nil {
		clog.Error("Failed to serialize room info message: %v", err)
		return
	}

	req := &connectproto.PushRoomMsgRequest{
		RoomId: int32(roomID),
		Msg: &connectproto.Msg{
			Ver:       1,
			Operation: int32(config.OpRoomInfoSend),
			SeqId:     tools.GetSnowflakeID(),
			Body:      roomInfoBytes,
		},
	}

	for serverID, conn := range conns {
		client := connectproto.NewConnectServiceClient(conn)
		ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
		reply, err := client.PushRoomInfo(ctx, req)
		cancel()

		if err != nil {
			clog.Error("Failed to broadcast room info, serverID: %s, roomID: %d, err: %v", serverID, roomID, err)
			continue
		}

		clog.Info("Successfully broadcasted room info, serverID: %s, roomID: %d, reply: %v", serverID, roomID, reply)
	}
}
