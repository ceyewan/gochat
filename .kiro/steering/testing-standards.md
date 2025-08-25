---
inclusion: fileMatch
fileMatchPattern: "**/*_test.go"
---

# GoChat 测试标准

## 测试理念

### 测试金字塔策略
- **单元测试 (70%)**: 针对单个函数和方法的快速、隔离测试
- **集成测试 (20%)**: 使用真实依赖测试服务交互
- **端到端测试 (10%)**: 包含所有组件的完整系统测试

### 测试原则
- 测试应该快速、可靠且独立
- 对多种场景使用表驱动测试
- 适当模拟外部依赖
- 测试正常路径和错误条件
- 维持高测试覆盖率（最低 80%）

## 单元测试模式

### 标准测试结构
遵循 Arrange-Act-Assert 模式：

```go
func TestMessageProcessor_ProcessMessage(t *testing.T) {
    tests := []struct {
        name           string
        input          *Message
        setupMocks     func(*MockRepository, *MockCache)
        expectedError  string
        expectedResult *Message
    }{
        {
            name: "successful message processing",
            input: &Message{
                ClientMsgID:    "client_123",
                ConversationID: "single_user1_user2",
                SenderID:       1,
                Content:        "Hello world",
                MessageType:    "text",
            },
            setupMocks: func(repo *MockRepository, cache *MockCache) {
                // Arrange: Setup mock expectations
                cache.EXPECT().SetNX(gomock.Any(), "msg_dedup:client_123", "1", time.Minute).Return(true, nil)
                cache.EXPECT().Incr(gomock.Any(), "conv_seq:single_user1_user2").Return(1, nil)
                repo.EXPECT().SaveMessage(gomock.Any(), gomock.Any()).Return(nil)
            },
            expectedError: "",
        },
        {
            name: "duplicate message handling",
            input: &Message{
                ClientMsgID: "client_123",
            },
            setupMocks: func(repo *MockRepository, cache *MockCache) {
                cache.EXPECT().SetNX(gomock.Any(), "msg_dedup:client_123", "1", time.Minute).Return(false, nil)
            },
            expectedError: "",
        },
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            // Arrange
            ctrl := gomock.NewController(t)
            defer ctrl.Finish()
            
            mockRepo := NewMockRepository(ctrl)
            mockCache := NewMockCache(ctrl)
            mockIDGen := NewMockIDGenerator(ctrl)
            
            if tt.setupMocks != nil {
                tt.setupMocks(mockRepo, mockCache)
            }
            
            processor := &MessageProcessor{
                repo:  mockRepo,
                cache: mockCache,
                idgen: mockIDGen,
                logger: zap.NewNop(),
            }
            
            // Act
            err := processor.ProcessMessage(context.Background(), tt.input)
            
            // Assert
            if tt.expectedError != "" {
                assert.Error(t, err)
                assert.Contains(t, err.Error(), tt.expectedError)
            } else {
                assert.NoError(t, err)
            }
        })
    }
}
```

### Mock 生成
使用 gomock 生成 mock：

```go
//go:generate mockgen -source=repository.go -destination=mocks/mock_repository.go

type Repository interface {
    SaveMessage(ctx context.Context, msg *Message) error
    GetMessage(ctx context.Context, messageID int64) (*Message, error)
    GetMessages(ctx context.Context, conversationID string, limit, offset int) ([]*Message, error)
}
```

### 测试辅助工具和固件
创建可重用的测试工具：

```go
// test_helpers.go
package testutil

import (
    "testing"
    "time"
    "github.com/stretchr/testify/require"
)

// CreateTestMessage creates a message for testing
func CreateTestMessage(t *testing.T, overrides ...func(*Message)) *Message {
    msg := &Message{
        MessageID:      12345,
        ClientMsgID:    "test_client_msg_123",
        ConversationID: "single_user1_user2",
        SenderID:       1,
        Content:        "Test message content",
        MessageType:    "text",
        SeqID:          1,
        CreatedAt:      time.Now(),
        TraceID:        "trace_123",
    }
    
    for _, override := range overrides {
        override(msg)
    }
    
    return msg
}

// CreateTestUser creates a user for testing
func CreateTestUser(t *testing.T, userID int64) *User {
    return &User{
        UserID:   userID,
        Username: fmt.Sprintf("user_%d", userID),
        Email:    fmt.Sprintf("user%d@example.com", userID),
        CreatedAt: time.Now(),
    }
}

// AssertMessageEqual compares two messages for equality
func AssertMessageEqual(t *testing.T, expected, actual *Message) {
    require.Equal(t, expected.MessageID, actual.MessageID)
    require.Equal(t, expected.Content, actual.Content)
    require.Equal(t, expected.SenderID, actual.SenderID)
    // Add other field comparisons as needed
}
```

## 集成测试模式

### 数据库集成测试
使用 testcontainers 进行数据库测试：

```go
func TestMessageRepository_Integration(t *testing.T) {
    if testing.Short() {
        t.Skip("skipping integration test")
    }
    
    // Setup test database container
    ctx := context.Background()
    mysqlContainer, err := mysql.RunContainer(ctx,
        testcontainers.WithImage("mysql:8.0"),
        mysql.WithDatabase("testdb"),
        mysql.WithUsername("testuser"),
        mysql.WithPassword("testpass"),
    )
    require.NoError(t, err)
    defer mysqlContainer.Terminate(ctx)
    
    // Get connection string
    connStr, err := mysqlContainer.ConnectionString(ctx)
    require.NoError(t, err)
    
    // Setup database
    db, err := gorm.Open(mysql.Open(connStr), &gorm.Config{})
    require.NoError(t, err)
    
    // Run migrations
    err = db.AutoMigrate(&Message{}, &User{}, &Group{})
    require.NoError(t, err)
    
    // Create repository
    repo := NewMessageRepository(db, zap.NewNop())
    
    t.Run("save and retrieve message", func(t *testing.T) {
        // Create test message
        msg := CreateTestMessage(t)
        
        // Save message
        err := repo.SaveMessage(ctx, msg)
        require.NoError(t, err)
        
        // Retrieve message
        retrieved, err := repo.GetMessage(ctx, msg.MessageID)
        require.NoError(t, err)
        
        // Assert equality
        AssertMessageEqual(t, msg, retrieved)
    })
}
```

### Redis 集成测试
使用真实 Redis 测试缓存行为：

```go
func TestCacheClient_Integration(t *testing.T) {
    if testing.Short() {
        t.Skip("skipping integration test")
    }
    
    // Setup Redis container
    ctx := context.Background()
    redisContainer, err := redis.RunContainer(ctx,
        testcontainers.WithImage("redis:7-alpine"),
    )
    require.NoError(t, err)
    defer redisContainer.Terminate(ctx)
    
    // Get connection string
    connStr, err := redisContainer.ConnectionString(ctx)
    require.NoError(t, err)
    
    // Create Redis client
    rdb := redis.NewClient(&redis.Options{
        Addr: connStr,
    })
    defer rdb.Close()
    
    cache := NewCacheClient(rdb, zap.NewNop())
    
    t.Run("set and get operations", func(t *testing.T) {
        key := "test_key"
        value := "test_value"
        
        // Set value
        err := cache.Set(ctx, key, value, time.Hour)
        require.NoError(t, err)
        
        // Get value
        retrieved, err := cache.Get(ctx, key)
        require.NoError(t, err)
        require.Equal(t, value, retrieved)
    })
    
    t.Run("cache miss handling", func(t *testing.T) {
        _, err := cache.Get(ctx, "nonexistent_key")
        require.Error(t, err)
        require.True(t, errors.Is(err, ErrCacheMiss))
    })
}
```

### Kafka 集成测试
测试消息队列操作：

```go
func TestKafkaProducer_Integration(t *testing.T) {
    if testing.Short() {
        t.Skip("skipping integration test")
    }
    
    // Setup Kafka container
    ctx := context.Background()
    kafkaContainer, err := kafka.RunContainer(ctx,
        kafka.WithClusterID("test-cluster"),
    )
    require.NoError(t, err)
    defer kafkaContainer.Terminate(ctx)
    
    // Get broker address
    brokers, err := kafkaContainer.Brokers(ctx)
    require.NoError(t, err)
    
    // Create producer
    producer := kafka.NewWriter(kafka.WriterConfig{
        Brokers: brokers,
        Topic:   "test-topic",
    })
    defer producer.Close()
    
    // Create consumer
    consumer := kafka.NewReader(kafka.ReaderConfig{
        Brokers: brokers,
        Topic:   "test-topic",
        GroupID: "test-group",
    })
    defer consumer.Close()
    
    t.Run("produce and consume message", func(t *testing.T) {
        // Produce message
        msg := kafka.Message{
            Key:   []byte("test-key"),
            Value: []byte("test-message"),
        }
        
        err := producer.WriteMessages(ctx, msg)
        require.NoError(t, err)
        
        // Consume message
        received, err := consumer.ReadMessage(ctx)
        require.NoError(t, err)
        
        require.Equal(t, msg.Key, received.Key)
        require.Equal(t, msg.Value, received.Value)
    })
}
```

## 服务测试模式

### gRPC 服务测试
使用适当设置测试 gRPC 服务：

```go
func TestMessageService_gRPC(t *testing.T) {
    // Setup in-memory gRPC server
    lis := bufconn.Listen(1024 * 1024)
    s := grpc.NewServer()
    
    // Create service with mocked dependencies
    ctrl := gomock.NewController(t)
    defer ctrl.Finish()
    
    mockRepo := NewMockRepository(ctrl)
    service := NewMessageService(mockRepo, zap.NewNop())
    
    pb.RegisterMessageServiceServer(s, service)
    
    go func() {
        if err := s.Serve(lis); err != nil {
            t.Logf("Server exited with error: %v", err)
        }
    }()
    defer s.Stop()
    
    // Create client connection
    conn, err := grpc.DialContext(context.Background(), "bufnet",
        grpc.WithContextDialer(func(context.Context, string) (net.Conn, error) {
            return lis.Dial()
        }),
        grpc.WithTransportCredentials(insecure.NewCredentials()),
    )
    require.NoError(t, err)
    defer conn.Close()
    
    client := pb.NewMessageServiceClient(conn)
    
    t.Run("get messages success", func(t *testing.T) {
        // Setup mock expectations
        expectedMessages := []*Message{
            CreateTestMessage(t),
        }
        mockRepo.EXPECT().GetMessages(gomock.Any(), "conv_123", 10, 0).Return(expectedMessages, nil)
        
        // Make gRPC call
        resp, err := client.GetMessages(context.Background(), &pb.GetMessagesRequest{
            ConversationId: "conv_123",
            Limit:         10,
            Offset:        0,
        })
        
        require.NoError(t, err)
        require.Len(t, resp.Messages, 1)
        require.Equal(t, expectedMessages[0].Content, resp.Messages[0].Content)
    })
}
```

### WebSocket 测试
测试 WebSocket 连接和消息处理：

```go
func TestWebSocketHandler(t *testing.T) {
    // Create test server
    hub := NewHub(zap.NewNop())
    go hub.Run()
    
    handler := &WSHandler{
        hub:    hub,
        logger: zap.NewNop(),
    }
    
    server := httptest.NewServer(http.HandlerFunc(handler.HandleWebSocket))
    defer server.Close()
    
    // Convert http://127.0.0.1 to ws://127.0.0.1
    wsURL := "ws" + strings.TrimPrefix(server.URL, "http") + "/ws?token=" + createTestJWT(t, 123)
    
    t.Run("successful connection and message", func(t *testing.T) {
        // Connect to WebSocket
        ws, _, err := websocket.DefaultDialer.Dial(wsURL, nil)
        require.NoError(t, err)
        defer ws.Close()
        
        // Send test message
        testMsg := WSMessage{
            Type: "send-message",
            Data: map[string]interface{}{
                "tempMessageId":  "temp_123",
                "conversationId": "single_user1_user2",
                "content":        "Hello test",
                "messageType":    "text",
            },
        }
        
        err = ws.WriteJSON(testMsg)
        require.NoError(t, err)
        
        // Read acknowledgment
        var ackMsg WSMessage
        err = ws.ReadJSON(&ackMsg)
        require.NoError(t, err)
        require.Equal(t, "message-ack", ackMsg.Type)
    })
}

func createTestJWT(t *testing.T, userID int64) string {
    token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
        "user_id": userID,
        "exp":     time.Now().Add(time.Hour).Unix(),
    })
    
    tokenString, err := token.SignedString([]byte("test-secret"))
    require.NoError(t, err)
    
    return tokenString
}
```

## 性能测试

### 基准测试
为关键路径编写基准测试：

```go
func BenchmarkMessageProcessor_ProcessMessage(b *testing.B) {
    // Setup
    processor := setupTestProcessor(b)
    msg := CreateTestMessage(b.(*testing.T))
    
    b.ResetTimer()
    b.RunParallel(func(pb *testing.PB) {
        for pb.Next() {
            err := processor.ProcessMessage(context.Background(), msg)
            if err != nil {
                b.Fatal(err)
            }
        }
    })
}

func BenchmarkCacheOperations(b *testing.B) {
    cache := setupTestCache(b)
    
    b.Run("Set", func(b *testing.B) {
        b.RunParallel(func(pb *testing.PB) {
            i := 0
            for pb.Next() {
                key := fmt.Sprintf("key_%d", i)
                err := cache.Set(context.Background(), key, "value", time.Hour)
                if err != nil {
                    b.Fatal(err)
                }
                i++
            }
        })
    })
    
    b.Run("Get", func(b *testing.B) {
        // Pre-populate cache
        for i := 0; i < 1000; i++ {
            cache.Set(context.Background(), fmt.Sprintf("key_%d", i), "value", time.Hour)
        }
        
        b.ResetTimer()
        b.RunParallel(func(pb *testing.PB) {
            i := 0
            for pb.Next() {
                key := fmt.Sprintf("key_%d", i%1000)
                _, err := cache.Get(context.Background(), key)
                if err != nil && !errors.Is(err, ErrCacheMiss) {
                    b.Fatal(err)
                }
                i++
            }
        })
    })
}
```

### 负载测试
为系统组件创建负载测试：

```go
func TestMessageProcessor_LoadTest(t *testing.T) {
    if testing.Short() {
        t.Skip("skipping load test")
    }
    
    processor := setupRealProcessor(t)
    
    // Test parameters
    numGoroutines := 100
    messagesPerGoroutine := 1000
    
    var wg sync.WaitGroup
    errors := make(chan error, numGoroutines*messagesPerGoroutine)
    
    start := time.Now()
    
    for i := 0; i < numGoroutines; i++ {
        wg.Add(1)
        go func(workerID int) {
            defer wg.Done()
            
            for j := 0; j < messagesPerGoroutine; j++ {
                msg := CreateTestMessage(t, func(m *Message) {
                    m.ClientMsgID = fmt.Sprintf("worker_%d_msg_%d", workerID, j)
                    m.Content = fmt.Sprintf("Load test message %d from worker %d", j, workerID)
                })
                
                if err := processor.ProcessMessage(context.Background(), msg); err != nil {
                    errors <- err
                    return
                }
            }
        }(i)
    }
    
    wg.Wait()
    close(errors)
    
    duration := time.Since(start)
    totalMessages := numGoroutines * messagesPerGoroutine
    
    // Check for errors
    var errorCount int
    for err := range errors {
        t.Logf("Error: %v", err)
        errorCount++
    }
    
    // Report results
    t.Logf("Processed %d messages in %v", totalMessages, duration)
    t.Logf("Throughput: %.2f messages/second", float64(totalMessages)/duration.Seconds())
    t.Logf("Error rate: %.2f%%", float64(errorCount)/float64(totalMessages)*100)
    
    require.Equal(t, 0, errorCount, "No errors should occur during load test")
}
```

## 测试组织

### 测试文件结构
将测试与源代码一起组织：

```
service/
├── message_processor.go
├── message_processor_test.go
├── integration/
│   ├── message_flow_test.go
│   └── database_test.go
├── testutil/
│   ├── helpers.go
│   ├── fixtures.go
│   └── containers.go
└── mocks/
    ├── mock_repository.go
    └── mock_cache.go
```

### 测试配置
为不同测试类型使用构建标签：

```go
//go:build integration
// +build integration

package integration

// Integration tests that require external dependencies
```

### Makefile 目标
提供便捷的测试命令：

```makefile
.PHONY: test test-unit test-integration test-load test-coverage

test: test-unit test-integration

test-unit:
	go test -short -race ./...

test-integration:
	go test -tags=integration ./integration/...

test-load:
	go test -tags=load -timeout=30m ./load/...

test-coverage:
	go test -short -race -coverprofile=coverage.out ./...
	go tool cover -html=coverage.out -o coverage.html

test-bench:
	go test -bench=. -benchmem ./...
```