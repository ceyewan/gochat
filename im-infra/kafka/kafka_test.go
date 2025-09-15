package kafka

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/ceyewan/gochat/im-infra/clog"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestKafkaProducerConsumer(t *testing.T) {
	if testing.Short() {
		t.Skip("跳过集成测试")
	}

	// 检查 Kafka 是否可用
	config := GetDefaultConfig("development")
	config.Brokers = []string{"localhost:9092"}

	// 尝试创建 Provider 来检查 Kafka 是否可用
	testProvider, err := NewProvider(context.Background(), config)
	if err != nil {
		t.Skipf("Kafka 不可用，跳过集成测试: %v", err)
	}
	testProvider.Close()

	// 初始化 clog
	clog.Init(context.Background(), clog.GetDefaultConfig("test"))

	// 创建配置 - 使用之前声明的 config
	config.ProducerConfig = GetDefaultConfig("development").ProducerConfig
	config.ConsumerConfig = GetDefaultConfig("development").ConsumerConfig

	// 创建 Provider
	provider, err := NewProvider(context.Background(), config)
	require.NoError(t, err)
	defer provider.Close()

	// 获取生产者
	producer := provider.Producer()
	defer producer.Close()

	// 获取消费者
	consumer := provider.Consumer("test-consumer-group")
	require.NotNil(t, consumer)
	defer consumer.Close()

	// 测试数据
	testMessage := map[string]string{
		"user_id": "test123",
		"action":  "test_event",
	}

	messageData, err := json.Marshal(testMessage)
	require.NoError(t, err)

	// 启动消费者
	messageChan := make(chan *Message, 1)
	errorChan := make(chan error, 1)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	go func() {
		handler := func(ctx context.Context, msg *Message) error {
			messageChan <- msg
			return nil
		}

		if err := consumer.Subscribe(ctx, []string{"example.test-topic"}, handler); err != nil {
			errorChan <- err
		}
	}()

	// 等待消费者准备就绪（增加等待时间确保消费者完全就绪）
	time.Sleep(3 * time.Second)

	// 使用 admin 接口创建 topic
	admin := provider.Admin()
	require.NotNil(t, admin)

	// 创建测试 topic
	err = admin.CreateTopic(ctx, "example.test-topic", 1, 1, nil)
	if err != nil {
		t.Logf("创建 topic 失败（可能已存在）: %v", err)
	}

	// 等待 topic 创建完成
	time.Sleep(1 * time.Second)

	// 发送消息
	msg := &Message{
		Topic: "example.test-topic",
		Key:   []byte("test-key"),
		Value: messageData,
	}

	// 测试同步发送
	err = producer.SendSync(context.Background(), msg)
	require.NoError(t, err)

	// 等待接收消息
	select {
	case receivedMsg := <-messageChan:
		assert.Equal(t, "example.test-topic", receivedMsg.Topic)
		assert.Equal(t, "test-key", string(receivedMsg.Key))

		var receivedData map[string]string
		err = json.Unmarshal(receivedMsg.Value, &receivedData)
		require.NoError(t, err)
		assert.Equal(t, testMessage, receivedData)

	case err := <-errorChan:
		t.Fatalf("消费者错误: %v", err)

	case <-time.After(10 * time.Second):
		t.Fatal("等待消息超时")
	}
}

func TestTraceIDPropagation(t *testing.T) {
	// 测试生产者端 trace ID 提取
	ctx := context.WithValue(context.Background(), TraceIDKey, "test-trace-123")

	// 测试 extractTraceID 函数
	traceID := extractTraceID(ctx)
	assert.Equal(t, "test-trace-123", traceID)

	// 测试空上下文
	traceID = extractTraceID(context.Background())
	assert.Empty(t, traceID)

	// 测试 nil 上下文
	traceID = extractTraceID(context.TODO())
	assert.Empty(t, traceID)

	// 测试错误类型的值
	ctxWrongType := context.WithValue(context.Background(), TraceIDKey, 123)
	traceID = extractTraceID(ctxWrongType)
	assert.Empty(t, traceID)
}

func TestTraceIDInjection(t *testing.T) {
	// 测试消费者端 trace ID 注入
	ctx := context.Background()

	// 测试正常 trace ID 注入
	newCtx := injectTraceID(ctx, "injected-trace-456")
	assert.Equal(t, "injected-trace-456", newCtx.Value(TraceIDKey))

	// 测试空 trace ID 注入
	sameCtx := injectTraceID(ctx, "")
	assert.Equal(t, ctx, sameCtx)

	// 测试覆盖已有 trace ID
	ctxWithTrace := context.WithValue(ctx, TraceIDKey, "old-trace")
	newCtx = injectTraceID(ctxWithTrace, "new-trace")
	assert.Equal(t, "new-trace", newCtx.Value(TraceIDKey))
}

func TestTraceIDEndToEnd(t *testing.T) {
	// 测试完整的 trace ID 传播流程
	originalTraceID := "end-to-end-trace-789"

	// 1. 业务代码设置 trace ID
	businessCtx := context.WithValue(context.Background(), TraceIDKey, originalTraceID)

	// 2. 生产者提取 trace ID（模拟）
	extractedTraceID := extractTraceID(businessCtx)
	assert.Equal(t, originalTraceID, extractedTraceID)

	// 3. 模拟消费者收到消息并提取 trace ID（从消息头）
	receivedTraceID := extractedTraceID // 实际从消息头 "X-Trace-ID" 获取

	// 4. 消费者注入 trace ID 到新 context
	consumerCtx := injectTraceID(context.Background(), receivedTraceID)

	// 5. 业务处理函数获取 trace ID
	finalTraceID := consumerCtx.Value(TraceIDKey).(string)
	assert.Equal(t, originalTraceID, finalTraceID)
}

func TestGetDefaultConfig(t *testing.T) {
	// 测试开发环境配置
	devConfig := GetDefaultConfig("development")
	assert.Equal(t, []string{"localhost:9092"}, devConfig.Brokers)
	assert.Equal(t, "PLAINTEXT", devConfig.SecurityProtocol)
	assert.Equal(t, 1, devConfig.ProducerConfig.Acks)
	assert.Equal(t, 3, devConfig.ProducerConfig.RetryMax)
	assert.Equal(t, "latest", devConfig.ConsumerConfig.AutoOffsetReset)

	// 测试生产环境配置
	prodConfig := GetDefaultConfig("production")
	assert.Equal(t, []string{"kafka1:9092", "kafka2:9092", "kafka3:9092"}, prodConfig.Brokers)
	assert.Equal(t, "SASL_SSL", prodConfig.SecurityProtocol)
	assert.Equal(t, -1, prodConfig.ProducerConfig.Acks)
	assert.Equal(t, 10, prodConfig.ProducerConfig.RetryMax)
	assert.Equal(t, "earliest", prodConfig.ConsumerConfig.AutoOffsetReset)
}

func TestMessageStructure(t *testing.T) {
	// 测试消息结构
	msg := &Message{
		Topic: "test-topic",
		Key:   []byte("test-key"),
		Value: []byte("test-value"),
		Headers: map[string][]byte{
			"X-Trace-ID": []byte("trace-123"),
			"Content-Type": []byte("application/json"),
		},
	}

	assert.Equal(t, "test-topic", msg.Topic)
	assert.Equal(t, "test-key", string(msg.Key))
	assert.Equal(t, "test-value", string(msg.Value))
	assert.Equal(t, "trace-123", string(msg.Headers["X-Trace-ID"]))
	assert.Equal(t, "application/json", string(msg.Headers["Content-Type"]))
}