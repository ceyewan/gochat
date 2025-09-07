package server

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"sync"
	"time"

	"github.com/ceyewan/gochat/api/kafka"
	"github.com/ceyewan/gochat/im-gateway/internal/config"
	"github.com/ceyewan/gochat/im-infra/clog"
	"github.com/gorilla/websocket"
)

// WebSocket 消息类型
type WSMessageType string

const (
	WSMsgTypeSendMessage  WSMessageType = "send-message"
	WSMsgTypeNewMessage   WSMessageType = "new-message"
	WSMsgTypeMessageAck   WSMessageType = "message-ack"
	WSMsgTypePing         WSMessageType = "ping"
	WSMsgTypePong         WSMessageType = "pong"
	WSMsgTypeOnlineStatus WSMessageType = "online-status"
)

// WebSocket 消息结构
type WSMessage struct {
	Type      WSMessageType   `json:"type"`
	MessageID string          `json:"message_id,omitempty"`
	Data      json.RawMessage `json:"data"`
	Timestamp int64           `json:"timestamp"`
}

// 发送消息数据结构
type SendMessageData struct {
	ConversationID string `json:"conversation_id"`
	MessageType    int    `json:"message_type"`
	Content        string `json:"content"`
	ClientMsgID    string `json:"client_msg_id"`
}

// 新消息数据结构
type NewMessageData struct {
	MessageID      string `json:"message_id"`
	ConversationID string `json:"conversation_id"`
	SenderID       string `json:"sender_id"`
	MessageType    int    `json:"message_type"`
	Content        string `json:"content"`
	SeqID          int64  `json:"seq_id"`
	Timestamp      int64  `json:"timestamp"`
}

// WebSocket 连接
type Connection struct {
	Conn      *websocket.Conn
	UserID    string
	GatewayID string
	Send      chan []byte
	Close     chan struct{}
}

// WebSocket 管理器
type WebSocketManager struct {
	config        *config.Config
	upgrader      websocket.Upgrader
	connections   map[string]*Connection // user_id -> connection
	mu            sync.RWMutex
	logger        clog.Logger
	messageChan   chan *WSMessage
	kafkaProducer kafka.Producer
	gatewayID     string
}

// NewWebSocketManager 创建 WebSocket 管理器
func NewWebSocketManager(cfg *config.Config, producer kafka.Producer) *WebSocketManager {
	return &WebSocketManager{
		config: cfg,
		upgrader: websocket.Upgrader{
			ReadBufferSize:  cfg.Server.WebSocket.ReadBufferSize,
			WriteBufferSize: cfg.Server.WebSocket.WriteBufferSize,
			CheckOrigin: func(r *http.Request) bool {
				return true // 允许所有来源
			},
		},
		connections:   make(map[string]*Connection),
		logger:        clog.Module("websocket-manager"),
		messageChan:   make(chan *WSMessage, 1000),
		kafkaProducer: producer,
		gatewayID:     generateGatewayID(),
	}
}

// HandleWebSocket 处理 WebSocket 连接
func (wm *WebSocketManager) HandleWebSocket(w http.ResponseWriter, r *http.Request) {
	// 从 URL 参数中获取 token
	token := r.URL.Query().Get("token")
	if token == "" {
		wm.logger.Error("WebSocket 连接缺少 token")
		http.Error(w, "Missing token", http.StatusUnauthorized)
		return
	}

	// TODO: 验证 token 并获取用户信息
	// 这里暂时使用 mock 数据
	userID := "user123" // 从 token 中解析

	// 升级 HTTP 连接为 WebSocket
	conn, err := wm.upgrader.Upgrade(w, r, nil)
	if err != nil {
		wm.logger.Error("WebSocket 升级失败", clog.Err(err))
		return
	}

	// 创建连接
	connection := &Connection{
		Conn:      conn,
		UserID:    userID,
		GatewayID: wm.gatewayID,
		Send:      make(chan []byte, 256),
		Close:     make(chan struct{}),
	}

	// 添加连接到管理器
	wm.addConnection(userID, connection)

	// TODO: 在 Redis 中注册用户在线状态
	// HSET user_session:{user_id} gateway_id {gateway_id}

	// 启动读写协程
	go wm.readPump(connection)
	go wm.writePump(connection)

	wm.logger.Info("WebSocket 连接建立",
		clog.String("user_id", userID),
		clog.String("gateway_id", wm.gatewayID))
}

// readPump 读取消息
func (wm *WebSocketManager) readPump(conn *Connection) {
	defer func() {
		wm.removeConnection(conn.UserID)
	}()

	// 设置读取超时
	conn.Conn.SetReadLimit(wm.config.Server.WebSocket.MaxMessageSize)

	// 设置心跳
	conn.Conn.SetReadDeadline(time.Now().Add(wm.config.Server.WebSocket.PongTimeout))
	conn.Conn.SetPongHandler(func(string) error {
		conn.Conn.SetReadDeadline(time.Now().Add(wm.config.Server.WebSocket.PongTimeout))
		return nil
	})

	for {
		_, message, err := conn.Conn.ReadMessage()
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				wm.logger.Error("WebSocket 读取错误", clog.Err(err))
			}
			break
		}

		// 解析消息
		var wsMsg WSMessage
		if err := json.Unmarshal(message, &wsMsg); err != nil {
			wm.logger.Error("解析 WebSocket 消息失败", clog.Err(err))
			continue
		}

		// 处理消息
		if err := wm.handleMessage(conn, &wsMsg); err != nil {
			wm.logger.Error("处理 WebSocket 消息失败", clog.Err(err))
		}
	}
}

// writePump 写入消息
func (wm *WebSocketManager) writePump(conn *Connection) {
	ticker := time.NewTicker(wm.config.Server.WebSocket.PingInterval)
	defer func() {
		ticker.Stop()
		wm.removeConnection(conn.UserID)
	}()

	for {
		select {
		case message, ok := <-conn.Send:
			conn.Conn.SetWriteDeadline(time.Now().Add(wm.config.Server.WebSocket.WriteTimeout))
			if !ok {
				// 连接关闭
				conn.Conn.WriteMessage(websocket.CloseMessage, []byte{})
				return
			}

			// 发送消息
			if err := conn.Conn.WriteMessage(websocket.TextMessage, message); err != nil {
				wm.logger.Error("WebSocket 写入消息失败", clog.Err(err))
				return
			}

		case <-ticker.C:
			// 发送心跳
			conn.Conn.SetWriteDeadline(time.Now().Add(wm.config.Server.WebSocket.WriteTimeout))
			if err := conn.Conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				wm.logger.Error("发送心跳失败", clog.Err(err))
				return
			}

		case <-conn.Close:
			// 关闭连接
			conn.Conn.WriteMessage(websocket.CloseMessage, []byte{})
			return
		}
	}
}

// handleMessage 处理 WebSocket 消息
func (wm *WebSocketManager) handleMessage(conn *Connection, msg *WSMessage) error {
	switch msg.Type {
	case WSMsgTypePing:
		// 响应心跳
		pongMsg := WSMessage{
			Type:      WSMsgTypePong,
			Timestamp: time.Now().Unix(),
		}
		return wm.sendMessage(conn, &pongMsg)

	case WSMsgTypeSendMessage:
		// 处理发送消息
		var data SendMessageData
		if err := json.Unmarshal(msg.Data, &data); err != nil {
			return err
		}

		// 构造上行消息
		upstreamMsg := &kafka.UpstreamMessage{
			TraceID:        generateTraceID(),
			UserID:         conn.UserID,
			GatewayID:      wm.gatewayID,
			ConversationID: data.ConversationID,
			MessageType:    data.MessageType,
			Content:        data.Content,
			ClientMsgID:    data.ClientMsgID,
			Timestamp:      time.Now().Unix(),
			Headers:        make(map[string]string),
		}

		// 发送到 Kafka
		return wm.sendToKafka(kafka.TopicUpstream, upstreamMsg)

	default:
		wm.logger.Warn("未知的 WebSocket 消息类型", clog.String("type", string(msg.Type)))
		return nil
	}
}

// sendMessage 发送消息到客户端
func (wm *WebSocketManager) sendMessage(conn *Connection, msg *WSMessage) error {
	data, err := json.Marshal(msg)
	if err != nil {
		return err
	}

	select {
	case conn.Send <- data:
		return nil
	default:
		// 发送队列满，关闭连接
		wm.logger.Warn("WebSocket 发送队列满，关闭连接")
		close(conn.Close)
		return errors.New("send queue full")
	}
}

// sendToKafka 发送消息到 Kafka
func (wm *WebSocketManager) sendToKafka(topic string, message interface{}) error {
	data, err := json.Marshal(message)
	if err != nil {
		return err
	}

	// 使用同步发送确保消息不丢失
	return wm.kafkaProducer.Send(context.Background(), topic, data)
}

// addConnection 添加连接
func (wm *WebSocketManager) addConnection(userID string, conn *Connection) {
	wm.mu.Lock()
	defer wm.mu.Unlock()

	// 如果用户已有连接，先关闭旧连接
	if oldConn, exists := wm.connections[userID]; exists {
		close(oldConn.Close)
		oldConn.Conn.Close()
	}

	wm.connections[userID] = conn
	wm.logger.Info("用户连接建立", clog.String("user_id", userID))
}

// removeConnection 移除连接
func (wm *WebSocketManager) removeConnection(userID string) {
	wm.mu.Lock()
	defer wm.mu.Unlock()

	if conn, exists := wm.connections[userID]; exists {
		delete(wm.connections, userID)
		close(conn.Close)
		conn.Conn.Close()

		// TODO: 从 Redis 中移除用户在线状态
		// DEL user_session:{user_id}

		wm.logger.Info("用户连接断开", clog.String("user_id", userID))
	}
}

// GetConnection 获取用户连接
func (wm *WebSocketManager) GetConnection(userID string) (*Connection, bool) {
	wm.mu.RLock()
	defer wm.mu.RUnlock()

	conn, exists := wm.connections[userID]
	return conn, exists
}

// GetGatewayID 获取网关 ID
func (wm *WebSocketManager) GetGatewayID() string {
	return wm.gatewayID
}

// PushMessage 推送消息给指定用户
func (wm *WebSocketManager) PushMessage(userID string, msg *WSMessage) error {
	wm.mu.RLock()
	conn, exists := wm.connections[userID]
	wm.mu.RUnlock()

	if !exists {
		return errors.New("user not connected")
	}

	return wm.sendMessage(conn, msg)
}

// Start 启动 WebSocket 管理器
func (wm *WebSocketManager) Start() {
	wm.logger.Info("WebSocket 管理器启动", clog.String("gateway_id", wm.gatewayID))
}

// Stop 停止 WebSocket 管理器
func (wm *WebSocketManager) Stop() {
	wm.mu.Lock()
	defer wm.mu.Unlock()

	// 关闭所有连接
	for _, conn := range wm.connections {
		close(conn.Close)
		conn.Conn.Close()
	}

	wm.connections = make(map[string]*Connection)
	wm.logger.Info("WebSocket 管理器停止")
}

// 辅助函数

func generateGatewayID() string {
	return "gateway-" + time.Now().Format("20060102150405") + "-" + generateRandomString(8)
}

func generateTraceID() string {
	return "trace-" + time.Now().Format("20060102150405") + "-" + generateRandomString(12)
}

func generateRandomString(length int) string {
	const charset = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
	b := make([]byte, length)
	for i := range b {
		b[i] = charset[time.Now().UnixNano()%int64(len(charset))]
	}
	return string(b)
}
