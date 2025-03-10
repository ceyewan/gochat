package connect

import (
	"gochat/tools"
)

// MessageType 消息类型常量
const (
	MessageTypeConnect  = "connect"  // 连接类型，客户端连接服务器
	MessageTypePushMsg  = "pushmsg"  // 消息推送类型，服务器推送消息给客户端
	MessageTypeRoomInfo = "roominfo" // 房间信息类型，服务器推送房间信息给客户端
	ConnectSuccess      = "success"  // 连接成功
	ConnectFail         = "fail"     // 连接失败
)

// WSRequest 客户端发送给服务器的请求消息
type WSRequest struct {
	UserID  int    `json:"user_id"`  // 用户ID
	RoomID  int    `json:"room_id"`  // 房间ID
	Token   string `json:"token"`    // 认证token
	MsgType string `json:"msg_type"` // 消息类型，一般为connect
}

// WSResponse 服务器推送给客户端的消息
type WSResponse struct {
	MsgType string      `json:"msg_type"` // 消息类型：pushmsg或roominfo
	Data    interface{} `json:"data"`     // 消息内容，根据MsgType不同而不同
}

// MessageData 消息推送的具体内容
type MessageData struct {
	FromUserID   int    `json:"from_user_id"`   // 发送者ID
	ToUserID     int    `json:"to_user_id"`     // 接收者ID
	FromUserName string `json:"from_user_name"` // 发送者用户名
	ToUserName   string `json:"to_user_name"`   // 接收者用户名
	RoomID       int    `json:"room_id"`        // 房间ID
	Message      string `json:"message"`        // 消息内容
	CreateTime   string `json:"create_time"`    // 创建时间
	MessageID    int    `json:"message_id"`     // 消息ID
}

// RoomInfoData 房间信息的具体内容
type RoomInfoData struct {
	RoomID   int               `json:"room_id"`   // 房间ID
	Count    int               `json:"count"`     // 房间内人数
	UserInfo map[string]string `json:"user_info"` // 用户信息映射，key为用户ID，value为用户名
}

// NewWSResponse 创建一个新的WebSocket响应
func NewWSResponse(msgType string, data interface{}) *WSResponse {
	return &WSResponse{
		MsgType: msgType,
		Data:    data,
	}
}

// NewMessageData 创建一个新的消息数据
func NewMessageData(chatMsg *tools.ChatMessage) *MessageData {
	return &MessageData{
		FromUserID:   chatMsg.FromUserID,
		ToUserID:     chatMsg.ToUserID,
		FromUserName: chatMsg.FromUserName,
		ToUserName:   chatMsg.ToUserName,
		RoomID:       chatMsg.RoomID,
		Message:      chatMsg.Message,
		CreateTime:   chatMsg.CreateTime,
		MessageID:    chatMsg.MessageID,
	}
}

// NewRoomInfoData 创建一个新的房间信息数据
func NewRoomInfoData(roomInfo *tools.RoomInfo) *RoomInfoData {
	return &RoomInfoData{
		RoomID:   roomInfo.RoomID,
		Count:    roomInfo.Count,
		UserInfo: roomInfo.UserInfo,
	}
}
