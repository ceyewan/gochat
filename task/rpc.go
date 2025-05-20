package task

import (
	"context"
	"gochat/clog"
	"gochat/proto/connectproto"
	"gochat/tools"
	"time"
)

// pushSingleToConnect 向指定服务器上的指定用户发送消息
func (t *Task) pushSingleToConnect(instanceID string, userID int, msg []byte) {
	conn, err := tools.GetServiceInstanceConn("connect-service", instanceID)
	if err != nil {
		clog.Module("task").Errorf("Failed to get connection, instanceID: %s, err: %v", instanceID, err)
		return
	}

	client := connectproto.NewConnectServiceClient(conn)
	req := &connectproto.PushSingleMsgRequest{
		UserId: int32(userID),
		Msg:    msg,
	}
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	reply, err := client.PushSingleMsg(ctx, req)
	if err != nil {
		clog.Module("task").Errorf("Failed to push single message, instanceID: %s, userID: %d, err: %v", instanceID, userID, err)
		return
	}
	clog.Module("task").Infof("Successfully pushed single message, instanceID: %s, userID: %d, reply: %v", instanceID, userID, reply)
}

// broadcastRoomToConnect 向指定房间广播消息
func (t *Task) broadcastRoomToConnect(roomID int, msg []byte) {
	conns, err := tools.GetAllServiceInstanceConns("connect-service")
	if err != nil {
		clog.Module("task").Errorf("Failed to get all connections: %v", err)
		return
	}

	if len(conns) == 0 {
		clog.Module("task").Warnf("No available connect-service instances")
		return
	}

	req := &connectproto.PushRoomMsgRequest{
		RoomId: int32(roomID),
		Msg:    msg,
	}

	for instanceID, conn := range conns {
		client := connectproto.NewConnectServiceClient(conn)
		ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
		reply, err := client.PushRoomMsg(ctx, req)
		cancel()

		if err != nil {
			clog.Module("task").Errorf("Failed to broadcast room message, instanceID: %s, roomID: %d, err: %v", instanceID, roomID, err)
			continue
		}

		clog.Module("task").Infof("Successfully broadcasted room message, instanceID: %s, roomID: %d, reply: %v", instanceID, roomID, reply)
	}
}

// broadcastRoomInfoToConnect 广播房间信息
func (t *Task) broadcastRoomInfoToConnect(roomID int, roomUserInfo []byte) {
	conns, err := tools.GetAllServiceInstanceConns("connect-service")
	if err != nil {
		clog.Module("task").Errorf("Failed to get all connections: %v", err)
		return
	}

	if len(conns) == 0 {
		clog.Module("task").Warnf("No available connect-service instances")
		return
	}

	req := &connectproto.PushRoomInfoRequest{
		RoomId: int32(roomID),
		Info:   roomUserInfo,
	}

	for instanceID, conn := range conns {
		client := connectproto.NewConnectServiceClient(conn)
		ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
		reply, err := client.PushRoomInfo(ctx, req)
		cancel()

		if err != nil {
			clog.Module("task").Errorf("Failed to broadcast room info, instanceID: %s, roomID: %d, err: %v", instanceID, roomID, err)
			continue
		}

		clog.Module("task").Infof("Successfully broadcasted room info, instanceID: %s, roomID: %d, reply: %v", instanceID, roomID, reply)
	}
}
