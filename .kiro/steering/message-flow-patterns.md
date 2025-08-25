---
inclusion: fileMatch
fileMatchPattern: "**/message*.go"
---

# GoChat 消息流模式

## 消息处理管道

### 消息生命周期概述
GoChat 中的每条消息都遵循这个标准化的生命周期：
1. **接收** (im-gateway): 通过 WebSocket 从客户端接收
2. **验证** (im-gateway): 验证格式和身份认证
3. **排队** (im-gateway): 发送到 Kafka 上游主题
4. **处理** (im-logic): 业务逻辑和持久化
5. **分发** (im-logic): 路由到适当的接收者
6. **投递** (im-gateway): 发送到客户端连接

### 消息结构标准
所有消息必须遵循这个标准化结构：

```go
// Core message structure
type Message struct {
    MessageID      int64     `json:"message_id"`
    ClientMsgID    string    `json:"client_msg_id"`    // Client-generated for idempotency
    ConversationID string    `json:"conversation_id"`
    SenderID       int64     `json:"sender_id"`
    Content        string    `json:"content"`
    MessageType    string    `json:"message_type"`     // "text", "image", "file", etc.
    SeqID          int64     `json:"seq_id"`          // Conversation sequence number
    CreatedAt      time.Time `json:"created_at"`
    TraceID        string    `json:"trace_id"`        // For distributed tracing
}

// Kafka message wrapper
type KafkaMessage struct {
    TraceID string            `json:"trace_id"`
    Header  map[string]string `json:"header"`
    Body    []byte            `json:"body"`    // Serialized message content
}

// WebSocket message envelope
type WSMessage struct {
    Type    string      `json:"type"`    // "send-message", "new-message", "message-ack"
    Data    interface{} `json:"data"`
    TraceID string      `json:"trace_id,omitempty"`
}
```

## 消息接收模式 (im-gateway)

### WebSocket 消息处理器
实现健壮的 WebSocket 消息处理：

```go
type WSConnection struct {
    conn     *websocket.Conn
    userID   int64
    send     chan []byte
    hub      *Hub
    logger   *zap.Logger
    producer *kafka.Producer
}

func (c *WSConnection) readPump() {
    defer func() {
        c.hub.unregister <- c
        c.conn.Close()
    }()
    
    c.conn.SetReadLimit(maxMessageSize)
    c.conn.SetReadDeadline(time.Now().Add(pongWait))
    c.conn.SetPongHandler(func(string) error {
        c.conn.SetReadDeadline(time.Now().Add(pongWait))
        return nil
    })
    
    for {
        _, messageBytes, err := c.conn.ReadMessage()
        if err != nil {
            if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
                c.logger.Error("websocket error", zap.Error(err))
            }
            break
        }
        
        if err := c.handleMessage(messageBytes); err != nil {
            c.logger.Error("failed to handle message", zap.Error(err))
            // Send error response to client
            c.sendError(err.Error())
        }
    }
}

func (c *WSConnection) handleMessage(data []byte) error {
    var wsMsg WSMessage
    if err := json.Unmarshal(data, &wsMsg); err != nil {
        return fmt.Errorf("invalid message format: %w", err)
    }
    
    // Generate trace ID if not present
    if wsMsg.TraceID == "" {
        wsMsg.TraceID = generateTraceID()
    }
    
    switch wsMsg.Type {
    case "send-message":
        return c.handleSendMessage(wsMsg)
    case "ping":
        return c.handlePing(wsMsg)
    default:
        return fmt.Errorf("unknown message type: %s", wsMsg.Type)
    }
}

func (c *WSConnection) handleSendMessage(wsMsg WSMessage) error {
    // Parse message data
    msgData, ok := wsMsg.Data.(map[string]interface{})
    if !ok {
        return errors.New("invalid message data format")
    }
    
    // Validate required fields
    if err := c.validateMessageData(msgData); err != nil {
        return fmt.Errorf("message validation failed: %w", err)
    }
    
    // Create message object
    msg := &Message{
        ClientMsgID:    msgData["tempMessageId"].(string),
        ConversationID: msgData["conversationId"].(string),
        SenderID:       c.userID,
        Content:        msgData["content"].(string),
        MessageType:    msgData["messageType"].(string),
        CreatedAt:      time.Now(),
        TraceID:        wsMsg.TraceID,
    }
    
    // Send to Kafka for processing
    if err := c.sendToKafka(msg); err != nil {
        return fmt.Errorf("failed to queue message: %w", err)
    }
    
    // Send immediate acknowledgment to client
    c.sendMessageAck(msg.ClientMsgID, wsMsg.TraceID)
    
    return nil
}
```

### 消息验证
实现全面的消息验证：

```go
func (c *WSConnection) validateMessageData(data map[string]interface{}) error {
    // Check required fields
    requiredFields := []string{"tempMessageId", "conversationId", "content", "messageType"}
    for _, field := range requiredFields {
        if _, exists := data[field]; !exists {
            return fmt.Errorf("missing required field: %s", field)
        }
    }
    
    // Validate content length
    content, ok := data["content"].(string)
    if !ok {
        return errors.New("content must be a string")
    }
    if len(content) == 0 || len(content) > maxContentLength {
        return fmt.Errorf("content length must be between 1 and %d characters", maxContentLength)
    }
    
    // Validate message type
    msgType, ok := data["messageType"].(string)
    if !ok {
        return errors.New("messageType must be a string")
    }
    if !isValidMessageType(msgType) {
        return fmt.Errorf("invalid message type: %s", msgType)
    }
    
    // Validate conversation ID format
    conversationID, ok := data["conversationId"].(string)
    if !ok {
        return errors.New("conversationId must be a string")
    }
    if !isValidConversationID(conversationID) {
        return fmt.Errorf("invalid conversation ID format: %s", conversationID)
    }
    
    return nil
}
```

## 消息处理模式 (im-logic)

### 核心消息处理器
实现主要的消息处理逻辑：

```go
type MessageProcessor struct {
    repo     MessageRepository
    cache    CacheClient
    producer *kafka.Producer
    idgen    IDGenerator
    logger   *zap.Logger
}

func (mp *MessageProcessor) ProcessMessage(ctx context.Context, kafkaMsg *kafka.Message) error {
    // Extract trace ID and add to context
    traceID := extractTraceID(kafkaMsg.Headers)
    ctx = context.WithValue(ctx, "trace_id", traceID)
    logger := mp.logger.With(zap.String("trace_id", traceID))
    
    // Deserialize message
    var msg Message
    if err := json.Unmarshal(kafkaMsg.Value, &msg); err != nil {
        logger.Error("failed to deserialize message", zap.Error(err))
        return fmt.Errorf("deserialization failed: %w", err)
    }
    
    // Idempotency check
    if isDuplicate, err := mp.checkDuplicate(ctx, msg.ClientMsgID); err != nil {
        logger.Warn("failed to check duplicate", zap.Error(err))
        // Continue processing (fail-open)
    } else if isDuplicate {
        logger.Info("duplicate message detected", zap.String("client_msg_id", msg.ClientMsgID))
        return nil
    }
    
    // Generate IDs
    msg.MessageID = mp.idgen.NextID()
    seqID, err := mp.generateSeqID(ctx, msg.ConversationID)
    if err != nil {
        logger.Error("failed to generate sequence ID", zap.Error(err))
        return fmt.Errorf("sequence ID generation failed: %w", err)
    }
    msg.SeqID = seqID
    
    // Persist message (critical path)
    if err := mp.persistMessage(ctx, &msg); err != nil {
        logger.Error("failed to persist message", zap.Error(err))
        return fmt.Errorf("persistence failed: %w", err)
    }
    
    // Distribute message
    if err := mp.distributeMessage(ctx, &msg); err != nil {
        logger.Error("failed to distribute message", zap.Error(err))
        // Don't return error - message is already persisted
        // This will be retried by the retry mechanism
    }
    
    logger.Info("message processed successfully",
        zap.Int64("message_id", msg.MessageID),
        zap.String("conversation_id", msg.ConversationID))
    
    return nil
}
```

### 消息持久化策略
实现带缓存的原子消息持久化：

```go
func (mp *MessageProcessor) persistMessage(ctx context.Context, msg *Message) error {
    // Start transaction for atomic operations
    return mp.repo.WithTransaction(ctx, func(tx Repository) error {
        // 1. Save to MySQL (primary storage)
        if err := tx.SaveMessage(ctx, msg); err != nil {
            return fmt.Errorf("failed to save to database: %w", err)
        }
        
        // 2. Update hot message cache (Redis ZSET)
        msgJSON, _ := json.Marshal(msg)
        cacheKey := fmt.Sprintf("hot_messages:%s", msg.ConversationID)
        if err := mp.cache.ZAdd(ctx, cacheKey, msg.SeqID, string(msgJSON)); err != nil {
            mp.logger.Warn("failed to update hot cache", zap.Error(err))
            // Don't fail the transaction for cache errors
        }
        
        // 3. Trim hot cache to keep only recent messages
        if err := mp.cache.ZRemRangeByRank(ctx, cacheKey, 0, -301); err != nil {
            mp.logger.Warn("failed to trim hot cache", zap.Error(err))
        }
        
        // 4. Update unread counters for conversation participants
        if err := mp.updateUnreadCounters(ctx, msg); err != nil {
            mp.logger.Warn("failed to update unread counters", zap.Error(err))
        }
        
        return nil
    })
}

func (mp *MessageProcessor) updateUnreadCounters(ctx context.Context, msg *Message) error {
    participants, err := mp.getConversationParticipants(ctx, msg.ConversationID)
    if err != nil {
        return fmt.Errorf("failed to get participants: %w", err)
    }
    
    // Update unread count for all participants except sender
    for _, userID := range participants {
        if userID == msg.SenderID {
            continue // Don't increment unread for sender
        }
        
        unreadKey := fmt.Sprintf("unread:%s:%d", msg.ConversationID, userID)
        if err := mp.cache.Incr(ctx, unreadKey); err != nil {
            mp.logger.Warn("failed to increment unread counter",
                zap.Int64("user_id", userID),
                zap.Error(err))
        }
    }
    
    return nil
}
```

## 消息分发模式

### 智能分发策略
基于会话类型和大小实现智能消息分发：

```go
func (mp *MessageProcessor) distributeMessage(ctx context.Context, msg *Message) error {
    switch {
    case strings.HasPrefix(msg.ConversationID, "single_"):
        return mp.distributeSingleMessage(ctx, msg)
    case strings.HasPrefix(msg.ConversationID, "group_"):
        return mp.distributeGroupMessage(ctx, msg)
    case msg.ConversationID == "world":
        return mp.distributeWorldMessage(ctx, msg)
    default:
        return fmt.Errorf("unknown conversation type: %s", msg.ConversationID)
    }
}

func (mp *MessageProcessor) distributeSingleMessage(ctx context.Context, msg *Message) error {
    // Extract user IDs from conversation ID
    userIDs, err := extractSingleChatUsers(msg.ConversationID)
    if err != nil {
        return fmt.Errorf("failed to extract user IDs: %w", err)
    }
    
    // Find recipient (not sender)
    var recipientID int64
    for _, userID := range userIDs {
        if userID != msg.SenderID {
            recipientID = userID
            break
        }
    }
    
    // Get recipient's gateway
    gatewayID, err := mp.getUserGateway(ctx, recipientID)
    if err != nil {
        if errors.Is(err, ErrUserOffline) {
            mp.logger.Info("recipient is offline", zap.Int64("user_id", recipientID))
            return nil // Not an error - user is simply offline
        }
        return fmt.Errorf("failed to get user gateway: %w", err)
    }
    
    // Send to recipient's gateway
    return mp.sendToGateway(ctx, gatewayID, recipientID, msg)
}

func (mp *MessageProcessor) distributeGroupMessage(ctx context.Context, msg *Message) error {
    groupID, err := extractGroupID(msg.ConversationID)
    if err != nil {
        return fmt.Errorf("failed to extract group ID: %w", err)
    }
    
    // Get group info to determine distribution strategy
    groupInfo, err := mp.repo.GetGroupInfo(ctx, groupID)
    if err != nil {
        return fmt.Errorf("failed to get group info: %w", err)
    }
    
    if groupInfo.MemberCount <= smallGroupThreshold {
        // Small group: real-time fanout
        return mp.distributeSmallGroup(ctx, msg, groupInfo)
    } else {
        // Large group: async processing
        return mp.scheduleLargeGroupDistribution(ctx, msg, groupInfo)
    }
}

func (mp *MessageProcessor) distributeSmallGroup(ctx context.Context, msg *Message, groupInfo *GroupInfo) error {
    // Get online members
    onlineMembers, err := mp.getOnlineGroupMembers(ctx, groupInfo.GroupID)
    if err != nil {
        return fmt.Errorf("failed to get online members: %w", err)
    }
    
    // Send to all online members except sender
    var errors []error
    for _, member := range onlineMembers {
        if member.UserID == msg.SenderID {
            continue
        }
        
        if err := mp.sendToGateway(ctx, member.GatewayID, member.UserID, msg); err != nil {
            mp.logger.Warn("failed to send to member",
                zap.Int64("user_id", member.UserID),
                zap.String("gateway_id", member.GatewayID),
                zap.Error(err))
            errors = append(errors, err)
        }
    }
    
    if len(errors) > 0 {
        return fmt.Errorf("failed to send to %d members", len(errors))
    }
    
    return nil
}
```

### 异步大群处理
异步处理大群消息分发：

```go
func (mp *MessageProcessor) scheduleLargeGroupDistribution(ctx context.Context, msg *Message, groupInfo *GroupInfo) error {
    // Create async task for large group fanout
    task := &LargeGroupFanoutTask{
        MessageID:      msg.MessageID,
        ConversationID: msg.ConversationID,
        GroupID:        groupInfo.GroupID,
        SenderID:       msg.SenderID,
        TraceID:        msg.TraceID,
        CreatedAt:      time.Now(),
    }
    
    taskData, err := json.Marshal(task)
    if err != nil {
        return fmt.Errorf("failed to serialize task: %w", err)
    }
    
    kafkaMsg := &kafka.Message{
        Topic: "im-task-large-group-fanout",
        Key:   []byte(fmt.Sprintf("group_%d", groupInfo.GroupID)),
        Value: taskData,
        Headers: []kafka.Header{
            {Key: "trace_id", Value: []byte(msg.TraceID)},
            {Key: "task_type", Value: []byte("large_group_fanout")},
        },
    }
    
    return mp.producer.WriteMessages(ctx, kafkaMsg)
}
```

## 消息投递模式 (im-gateway)

### 网关消息消费者
实现向 WebSocket 连接的消息投递：

```go
type GatewayConsumer struct {
    reader *kafka.Reader
    hub    *Hub
    logger *zap.Logger
}

func (gc *GatewayConsumer) Start(ctx context.Context) error {
    for {
        select {
        case <-ctx.Done():
            return ctx.Err()
        default:
            msg, err := gc.reader.ReadMessage(ctx)
            if err != nil {
                gc.logger.Error("failed to read message", zap.Error(err))
                continue
            }
            
            if err := gc.deliverMessage(ctx, msg); err != nil {
                gc.logger.Error("failed to deliver message", zap.Error(err))
            }
        }
    }
}

func (gc *GatewayConsumer) deliverMessage(ctx context.Context, kafkaMsg kafka.Message) error {
    // Extract user ID from message key or headers
    userID, err := extractUserID(kafkaMsg)
    if err != nil {
        return fmt.Errorf("failed to extract user ID: %w", err)
    }
    
    // Find user's WebSocket connection
    conn := gc.hub.GetConnection(userID)
    if conn == nil {
        gc.logger.Debug("user not connected to this gateway", zap.Int64("user_id", userID))
        return nil // User not connected to this gateway instance
    }
    
    // Deserialize message
    var msg Message
    if err := json.Unmarshal(kafkaMsg.Value, &msg); err != nil {
        return fmt.Errorf("failed to deserialize message: %w", err)
    }
    
    // Create WebSocket message
    wsMsg := WSMessage{
        Type: "new-message",
        Data: msg,
        TraceID: msg.TraceID,
    }
    
    // Send to connection
    return conn.SendMessage(wsMsg)
}
```

### 连接管理
实现健壮的 WebSocket 连接管理：

```go
type Hub struct {
    connections map[int64]*WSConnection
    register    chan *WSConnection
    unregister  chan *WSConnection
    broadcast   chan []byte
    mutex       sync.RWMutex
    logger      *zap.Logger
}

func (h *Hub) Run() {
    for {
        select {
        case conn := <-h.register:
            h.mutex.Lock()
            h.connections[conn.userID] = conn
            h.mutex.Unlock()
            h.logger.Info("user connected", zap.Int64("user_id", conn.userID))
            
        case conn := <-h.unregister:
            h.mutex.Lock()
            if _, ok := h.connections[conn.userID]; ok {
                delete(h.connections, conn.userID)
                close(conn.send)
            }
            h.mutex.Unlock()
            h.logger.Info("user disconnected", zap.Int64("user_id", conn.userID))
        }
    }
}

func (h *Hub) GetConnection(userID int64) *WSConnection {
    h.mutex.RLock()
    defer h.mutex.RUnlock()
    return h.connections[userID]
}
```