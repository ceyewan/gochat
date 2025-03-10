package connect

import (
	"encoding/json"
	"gochat/clog"
	"gochat/config"
	"net/http"
	"sync"

	"github.com/gorilla/websocket"
)

// ConnectMsg 定义连接相关的消息结构
type ConnectMsg struct {
	UserID  int    `json:"user_id"`
	RoomID  int    `json:"room_id"`
	Token   string `json:"token"`
	Message string `json:"message"`
}

type SendMsg struct {
	Count        int               `json:"count"`
	Msg          string            `json:"msg"`
	RoomUserInfo map[string]string `json:"room_user_info"`
}

// Channel 封装了一个WebSocket连接和相关的元数据
type Channel struct {
	conn   *websocket.Conn
	userID int
	roomID int
	send   chan []byte
}

// NewChannel 创建一个新的Channel实例
func NewChannel(bufferSize int) *Channel {
	return &Channel{
		send: make(chan []byte, bufferSize),
	}
}

// WSServer 定义WebSocket服务器配置和状态
type WSServer struct {
	Options struct {
		ReadBufferSize  int
		WriteBufferSize int
		BroadcastSize   int
	}
	InstanceID string
	connMgr    *ConnectionManager
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
	DefaultWSServer.connMgr = connectionManager

	http.HandleFunc("/ws", func(w http.ResponseWriter, r *http.Request) {
		serveWs(DefaultWSServer, w, r)
	})
	clog.Info("WebSocket server started at %s", config.Conf.Connect.Websocket.Bind)
	err := http.ListenAndServe(config.Conf.Connect.Websocket.Bind, nil)
	if err != nil {
		clog.Error("WebSocket server failed to start: %v", err)
	}
	return err
}

// serveWs 处理单个WebSocket连接请求
func serveWs(server *WSServer, w http.ResponseWriter, r *http.Request) {
	upGrader := websocket.Upgrader{
		ReadBufferSize:  server.Options.ReadBufferSize,
		WriteBufferSize: server.Options.WriteBufferSize,
		CheckOrigin:     func(r *http.Request) bool { return true },
	}

	conn, err := upGrader.Upgrade(w, r, nil)
	if err != nil {
		clog.Error("Failed to upgrade connection: %v", err)
		return
	}

	ch := NewChannel(server.Options.BroadcastSize)
	ch.conn = conn

	var wg sync.WaitGroup
	wg.Add(2)

	defer func() {
		wg.Wait()
		if ch.userID != 0 {
			server.connMgr.RemoveUser(ch.userID, ch.roomID)
			LogicRPCObj.Disconnect(ch.userID, ch.roomID)
		}
		conn.Close()
	}()

	go func() {
		defer wg.Done()
		readMessages(server, ch)
	}()

	go func() {
		defer wg.Done()
		writeMessages(ch)
	}()
}

// readMessages 处理从WebSocket接收的消息
func readMessages(server *WSServer, ch *Channel) {
	for {
		_, message, err := ch.conn.ReadMessage()
		if err != nil {
			clog.Error("Read message error: %v", err)
			break
		}

		var msg ConnectMsg
		if err := json.Unmarshal(message, &msg); err != nil {
			clog.Error("Unmarshal message error: %v", err)
			continue
		}

		if msg.Message == "connect" {
			userID, err := LogicRPCObj.Connect(msg.Token, server.InstanceID, msg.RoomID)
			if err != nil {
				clog.Error("Connect error: %v", err)
				break
			}
			ch.userID = userID
			ch.roomID = msg.RoomID
			server.connMgr.AddUser(userID, msg.RoomID, ch)
			clog.Info("User %d connected to room %d", userID, msg.RoomID)
		}
	}
}

// writeMessages 处理向WebSocket发送的消息
func writeMessages(ch *Channel) {
	for message := range ch.send {
		if err := ch.conn.WriteMessage(websocket.TextMessage, message); err != nil {
			clog.Error("Write message error: %v", err)
			return
		}
	}
	ch.conn.WriteMessage(websocket.CloseMessage, []byte{})
}
