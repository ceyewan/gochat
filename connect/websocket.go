package connect

import (
	"encoding/json"
	"gochat/clog"
	"gochat/config"
	"net/http"

	"github.com/gorilla/websocket"
)

// ConnectMsg 定义连接相关的消息结构
// 用于客户端给服务器发送消息
type ConnectMsg struct {
	UserID  int    `json:"user_id"` // 用户ID
	RoomID  int    `json:"room_id"` // 房间ID
	Token   string `json:"token"`   // 认证令牌
	Message string `json:"message"` // 消息内容/指令
}

type SendMsg struct {
	Count        int               `json:"count"`
	Msg          string            `json:"msg"`
	RoomUserInfo map[string]string `json:"room_user_info"`
}

// Channel 封装了一个WebSocket连接和相关的元数据
// 负责单个客户端连接的数据收发
type Channel struct {
	conn   *websocket.Conn // WebSocket连接对象
	userID int             // 与该连接关联的用户ID
	roomID int             // 该连接所在的房间ID
	send   chan []byte     // 发送消息的通道
}

// NewChannel 创建一个新的Channel实例
// bufferSize: 发送缓冲区大小
func NewChannel(bufferSize int) *Channel {
	return &Channel{
		send: make(chan []byte, bufferSize),
	}
}

// WSServer 定义WebSocket服务器配置和状态
type WSServer struct {
	Options struct {
		ReadBufferSize  int // WebSocket读缓冲区大小
		WriteBufferSize int // WebSocket写缓冲区大小
		BroadcastSize   int // 广播缓冲区大小
	}
	ServerID string             // 服务器唯一标识
	connMgr  *ConnectionManager // 连接管理器引用
}

// DefaultWSServer 是默认的WebSocket服务器实例
var DefaultWSServer = &WSServer{
	Options: struct {
		ReadBufferSize  int
		WriteBufferSize int
		BroadcastSize   int
	}{
		ReadBufferSize:  config.Conf.Connect.Websocket.ReadBufferSize,
		WriteBufferSize: config.Conf.Connect.Websocket.WriteBufferSize,
		BroadcastSize:   config.Conf.Connect.Websocket.BroadcastSize,
	},
}

// InitWebSocket 初始化WebSocket服务并启动监听
func InitWebSocket() error {
	// 使用全局连接管理器
	DefaultWSServer.connMgr = connectionManager

	http.HandleFunc("/ws", func(w http.ResponseWriter, r *http.Request) {
		serveWs(DefaultWSServer, w, r)
	})
	err := http.ListenAndServe(config.Conf.Connect.Websocket.Bind, nil)
	return err
}

// serveWs 处理单个WebSocket连接请求
func serveWs(server *WSServer, w http.ResponseWriter, r *http.Request) {
	var upGrader = websocket.Upgrader{
		ReadBufferSize:  server.Options.ReadBufferSize,
		WriteBufferSize: server.Options.WriteBufferSize,
	}
	// 允许跨域请求
	upGrader.CheckOrigin = func(r *http.Request) bool { return true }

	conn, err := upGrader.Upgrade(w, r, nil)
	if err != nil {
		clog.Error("serverWs err:%s", err.Error())
		return
	}

	// 创建channel并关联连接
	ch := NewChannel(server.Options.BroadcastSize)
	ch.conn = conn

	// 处理连接断开
	defer func() {
		if ch.userID != 0 {
			// 清理连接管理器中的用户记录
			server.connMgr.RemoveUser(ch.userID, ch.roomID)
			// 通知逻辑服务用户断开连接
			LogicRPCObj.Disconnect(ch.userID, ch.roomID)
		}
		conn.Close()
	}()

	// 启动消息读取协程
	go handleMessages(server, ch, conn)

	// 启动消息发送循环
	server.writePump(ch)
}

// handleMessages 处理从WebSocket接收的消息
func handleMessages(server *WSServer, ch *Channel, conn *websocket.Conn) {
	for {
		_, message, err := conn.ReadMessage()
		if err != nil {
			clog.Error("read message error: %v", err)
			break
		}

		var msg ConnectMsg
		if err := json.Unmarshal(message, &msg); err != nil {
			clog.Error("unmarshal message error: %v", err)
			continue
		}

		// 处理connect消息
		if msg.Message == "connect" {
			userID, err := LogicRPCObj.Connect(msg.Token, server.ServerID, msg.RoomID)
			if err != nil {
				clog.Error("connect error: %v", err)
				break
			}
			ch.userID = userID
			ch.roomID = msg.RoomID

			// 添加到连接管理器中
			server.connMgr.AddUser(userID, msg.RoomID, ch)
		}
	}
}

// writePump 负责将消息写入WebSocket连接
func (s *WSServer) writePump(ch *Channel) {
	defer func() {
		ch.conn.Close()
	}()

	for message := range ch.send {
		if err := ch.conn.WriteMessage(websocket.TextMessage, message); err != nil {
			return
		}
	}
	ch.conn.WriteMessage(websocket.CloseMessage, []byte{})
}
