package kafka

import (
	"context"
	"testing"
	"time"

	"github.com/ceyewan/gochat/im-infra/clog"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewProvider(t *testing.T) {
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

	// 测试正常创建 Provider
	provider, err := NewProvider(context.Background(), config)
	require.NoError(t, err)
	require.NotNil(t, provider)
	defer provider.Close()

	// 测试 Provider 基本操作
	assert.NotNil(t, provider.Producer())
	assert.NotNil(t, provider.Consumer("test-group"))
	assert.NotNil(t, provider.Admin())

	// 测试 Ping
	ctx := context.Background()
	err = provider.Ping(ctx)
	assert.NoError(t, err)
}

func TestProviderErrorHandling(t *testing.T) {
	// 测试空配置
	_, err := NewProvider(context.Background(), nil)
	assert.Error(t, err)
	assert.True(t, IsConfigError(err))

	// 测试无效配置
	config := &Config{}
	_, err = NewProvider(context.Background(), config)
	assert.Error(t, err)
	assert.True(t, IsConfigError(err))
}

func TestProviderProducerOperations(t *testing.T) {
	if testing.Short() {
		t.Skip("跳过集成测试")
	}

	config := GetDefaultConfig("development")
	config.Brokers = []string{"localhost:9092"}

	provider, err := NewProvider(context.Background(), config)
	require.NoError(t, err)
	defer provider.Close()

	producer := provider.Producer()
	assert.NotNil(t, producer)

	// 测试生产者指标
	metrics := producer.GetMetrics()
	assert.NotNil(t, metrics)
	assert.Contains(t, metrics, "total_messages")
	assert.Contains(t, metrics, "total_bytes")
	assert.Contains(t, metrics, "success_messages")
	assert.Contains(t, metrics, "failed_messages")

	// 测试 Ping
	err = producer.Ping(context.Background())
	assert.NoError(t, err)
}

func TestProviderConsumerOperations(t *testing.T) {
	if testing.Short() {
		t.Skip("跳过集成测试")
	}

	config := GetDefaultConfig("development")
	config.Brokers = []string{"localhost:9092"}

	provider, err := NewProvider(context.Background(), config)
	require.NoError(t, err)
	defer provider.Close()

	// 测试获取消费者
	consumer := provider.Consumer("test-group")
	require.NotNil(t, consumer)

	// 测试重复获取同一个 groupID 的消费者返回同一个实例
	consumer2 := provider.Consumer("test-group")
	assert.Same(t, consumer, consumer2)

	// 测试不同 groupID 返回不同的消费者
	consumer3 := provider.Consumer("test-group-2")
	assert.NotSame(t, consumer, consumer3)

	// 测试空 groupID
	nilConsumer := provider.Consumer("")
	assert.Nil(t, nilConsumer)

	// 测试消费者指标
	metrics := consumer.GetMetrics()
	assert.NotNil(t, metrics)
	assert.Contains(t, metrics, "total_messages")
	assert.Contains(t, metrics, "processed_messages")
	assert.Contains(t, metrics, "failed_messages")

	// 测试 Ping
	err = consumer.Ping(context.Background())
	assert.NoError(t, err)
}

func TestProviderAdminOperations(t *testing.T) {
	if testing.Short() {
		t.Skip("跳过集成测试")
	}

	config := GetDefaultConfig("development")
	config.Brokers = []string{"localhost:9092"}

	provider, err := NewProvider(context.Background(), config)
	require.NoError(t, err)
	defer provider.Close()

	admin := provider.Admin()
	require.NotNil(t, admin)

	ctx := context.Background()

	// 测试创建主题
	err = admin.CreateTopic(ctx, "test-provider-topic", 3, 1, map[string]string{
		"cleanup.policy": "delete",
		"retention.ms":  "86400000",
	})
	assert.NoError(t, err)

	// 等待主题创建完成
	time.Sleep(1 * time.Second)

	// 测试获取主题元数据
	metadata, err := admin.GetTopicMetadata(ctx, "test-provider-topic")
	require.NoError(t, err)
	assert.NotNil(t, metadata)
	assert.Equal(t, int32(3), metadata.NumPartitions)
	assert.Equal(t, int16(1), metadata.ReplicationFactor)

	// 测试列出所有主题
	topics, err := admin.ListTopics(ctx)
	assert.NoError(t, err)
	assert.Contains(t, topics, "test-provider-topic")

	// 测试删除主题
	err = admin.DeleteTopic(ctx, "test-provider-topic")
	assert.NoError(t, err)

	// 等待主题删除完成
	time.Sleep(1 * time.Second)
}

func TestProviderWithOptions(t *testing.T) {
	if testing.Short() {
		t.Skip("跳过集成测试")
	}

	config := GetDefaultConfig("development")
	config.Brokers = []string{"localhost:9092"}

	// 测试使用自定义 logger
	logger := clog.Namespace("test-kafka")
	provider, err := NewProvider(context.Background(), config, WithLogger(logger))
	require.NoError(t, err)
	defer provider.Close()

	// 验证 Provider 正常工作
	assert.NotNil(t, provider.Producer())
	assert.NotNil(t, provider.Consumer("test-group"))
	assert.NotNil(t, provider.Admin())
}

func TestProviderBackwardCompatibility(t *testing.T) {
	if testing.Short() {
		t.Skip("跳过集成测试")
	}

	config := GetDefaultConfig("development")
	config.Brokers = []string{"localhost:9092"}

	// 测试 NewProducer 仍然可用
	producer, err := NewProducer(context.Background(), config)
	require.NoError(t, err)
	defer producer.Close()

	assert.NotNil(t, producer)

	// 测试 NewConsumer 仍然可用
	consumer, err := NewConsumer(context.Background(), config, "test-group")
	require.NoError(t, err)
	defer consumer.Close()

	assert.NotNil(t, consumer)
}

func TestKafkaErrorTypes(t *testing.T) {
	// 测试错误类型判断
	configErr := ErrInvalidConfig("test config error")
	assert.True(t, IsConfigError(configErr))
	assert.False(t, IsConnectionError(configErr))

	connErr := ErrConnection("connection failed", nil)
	assert.True(t, IsConnectionError(connErr))
	assert.False(t, IsConfigError(connErr))

	producerErr := ErrProducer("producer error", nil)
	assert.True(t, IsProducerError(producerErr))
	assert.False(t, IsConsumerError(producerErr))

	consumerErr := ErrConsumer("consumer error", nil)
	assert.True(t, IsConsumerError(consumerErr))
	assert.False(t, IsProducerError(consumerErr))

	adminErr := ErrAdmin("admin error", nil)
	assert.True(t, IsAdminError(adminErr))
	assert.False(t, IsConfigError(adminErr))

	timeoutErr := ErrTimeout("timeout error", nil)
	assert.True(t, IsTimeoutError(timeoutErr))
	assert.False(t, IsConnectionError(timeoutErr))

	argErr := ErrInvalidArg("invalid argument")
	assert.True(t, IsInvalidArgError(argErr))
	assert.False(t, IsConfigError(argErr))
}

func TestMessageOperations(t *testing.T) {
	// 测试消息结构
	msg := &Message{
		Topic: "test-topic",
		Key:   []byte("test-key"),
		Value: []byte("test-value"),
		Headers: map[string][]byte{
			"content-type": []byte("application/json"),
			"trace-id":     []byte("trace-123"),
		},
	}

	assert.Equal(t, "test-topic", msg.Topic)
	assert.Equal(t, []byte("test-key"), msg.Key)
	assert.Equal(t, []byte("test-value"), msg.Value)
	assert.Len(t, msg.Headers, 2)
	assert.Equal(t, []byte("application/json"), msg.Headers["content-type"])
	assert.Equal(t, []byte("trace-123"), msg.Headers["trace-id"])
}

func TestTraceIDKeyConstant(t *testing.T) {
	// 测试 TraceIDKey 常量
	assert.Equal(t, "trace-id", TraceIDKey)
}