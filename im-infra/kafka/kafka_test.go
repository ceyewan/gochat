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

	// 尝试创建生产者来检查 Kafka 是否可用
	testProducer, err := NewProducer(context.Background(), config)
	if err != nil {
		t.Skipf("Kafka 不可用，跳过集成测试: %v", err)
	}
	testProducer.Close()

	// 初始化 clog
	clog.Init(context.Background(), clog.GetDefaultConfig("test"))

	// 创建配置 - 使用之前声明的 config
	config.ProducerConfig = GetDefaultConfig("development").ProducerConfig
	config.ConsumerConfig = GetDefaultConfig("development").ConsumerConfig

	// 创建生产者
	producer, err := NewProducer(context.Background(), config)
	require.NoError(t, err)
	defer producer.Close()

	// 创建消费者
	consumer, err := NewConsumer(context.Background(), config, "test-consumer-group")
	require.NoError(t, err)
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

	// 等待消费者准备就绪
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
		assert.Equal(t, "test-topic", receivedMsg.Topic)
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
	// 测试 trace_id 传播逻辑
	ctx := clog.WithTraceID(context.Background(), "test-trace-123")

	// 由于 traceIDKey 是私有的，我们只能测试 WithTraceID 函数本身
	// 实际的 trace_id 传播功能需要在业务代码中手动处理
	// 这里我们只测试 clog 的功能是否正常
	logger := clog.WithContext(ctx)
	assert.NotNil(t, logger)

	// 测试空上下文
	logger = clog.WithContext(context.Background())
	assert.NotNil(t, logger)
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