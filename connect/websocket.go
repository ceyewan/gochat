package connect

import (
	"encoding/json"
	"gochat/clog"
	"gochat/config"
	"net/http"
	"sync"
	"time"

	"github.com/gorilla/websocket"
)

// 添加心跳相关常量
const (
	// 心跳间隔时间
	heartbeatInterval = 30 * time.Second

	// 心跳超时时间
	heartbeatTimeout = 60 * time.Second

	// 消息类型常量
	messageTypePing = "ping"
	messageTypePong = "pong"
)

// Channel 封装了一个WebSocket连接和相关的元数据
type Channel struct {
	conn         *websocket.Conn
	userID       int
	roomID       int
	send         chan []byte
	lastActivity time.Time  // 添加最后活动时间
	closed       bool       // 标记通道是否已关闭
	closeMutex   sync.Mutex // 保护closed字段
}

// NewChannel 创建一个新的Channel实例
func NewChannel(bufferSize int) *Channel {
	return &Channel{
		send:         make(chan []byte, bufferSize),
		lastActivity: time.Now(),
		closed:       false,
	}
}

// 更新心跳时间
func (ch *Channel) updateActivity() {
	ch.lastActivity = time.Now()
}

// 检查通道是否应该关闭
func (ch *Channel) shouldClose() bool {
	return time.Since(ch.lastActivity) > heartbeatTimeout
}

// 安全关闭通道
func (ch *Channel) safeClose() bool {
	ch.closeMutex.Lock()
	defer ch.closeMutex.Unlock()

	if ch.closed {
		return false
	}

	ch.closed = true
	close(ch.send)
	return true
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
	wg.Add(3)

	defer func() {
		ch.safeClose()
		wg.Wait()
		server.connMgr.RemoveUser(ch.userID, ch.roomID)
		LogicRPCObj.Disconnect(ch.userID, ch.roomID)
		clog.Info("User %d disconnected from room %d", ch.userID, ch.roomID)
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

	go func() {
		defer wg.Done()
		heartbeat(ch)
	}()
}

// 心跳检测
func heartbeat(ch *Channel) {
	ticker := time.NewTicker(heartbeatInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			if ch.shouldClose() {
				clog.Warning("Heartbeat timeout for user %d, closing connection", ch.userID)
				ch.conn.Close()
				return
			}

			// 发送ping消息
			pingMessage, _ := json.Marshal(map[string]string{"type": messageTypePing})
			err := ch.conn.WriteMessage(websocket.PingMessage, pingMessage)
			if err != nil {
				clog.Error("Failed to send ping: %v", err)
				return
			}
		}

		if ch.closed {
			return
		}
	}
}

// readMessages 处理从WebSocket接收的消息
func readMessages(server *WSServer, ch *Channel) {
	for {
		_, message, err := ch.conn.ReadMessage()
		if err != nil {
			clog.Error("Read message error: %v", err)
			break
		}

		var msg WSRequest
		if err := json.Unmarshal(message, &msg); err != nil {
			// 检查是否是ping/pong消息
			var pingMsg map[string]string
			if json.Unmarshal(message, &pingMsg) == nil {
				if pingMsg["type"] == messageTypePing {
					// 回复pong
					pongMsg, _ := json.Marshal(map[string]string{"type": messageTypePong})
					ch.conn.WriteMessage(websocket.PongMessage, pongMsg)
					continue
				}
				if pingMsg["type"] == messageTypePong {
					continue // 忽略客户端的pong响应
				}
			}
			clog.Error("Unmarshal message error: %v", err)
			continue
		}

		if msg.MsgType == MessageTypeConnect {
			err := LogicRPCObj.Connect(msg.Token, server.InstanceID, msg.UserID, msg.RoomID)
			if err != nil {
				// 发送连接失败响应
				failResp := NewWSResponse(ConnectFail, nil)
				resp, _ := json.Marshal(failResp)
				ch.send <- resp
				clog.Error("Connect error: %v", err)
				break
			}
			ch.userID = msg.UserID
			ch.roomID = msg.RoomID
			server.connMgr.AddUser(msg.UserID, msg.RoomID, ch)
			// 发送连接成功响应
			successResp := NewWSResponse(ConnectSuccess, nil)
			resp, _ := json.Marshal(successResp)
			ch.send <- resp
			clog.Info("User %d connected to room %d", msg.UserID, msg.RoomID)
		}
	}
}

// writeMessages 处理向WebSocket发送的消息
func writeMessages(ch *Channel) {
	for message := range ch.send {
		ch.updateActivity()
		if err := ch.conn.WriteMessage(websocket.TextMessage, message); err != nil {
			clog.Error("Write message error: %v", err)
			break
		}
	}
	ch.conn.WriteMessage(websocket.CloseMessage, []byte{})
}
