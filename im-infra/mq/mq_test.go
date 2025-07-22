package mq

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/ceyewan/gochat/im-infra/mq/internal"
)

// TestDefaultConfig 测试默认配置
func TestDefaultConfig(t *testing.T) {
	cfg := DefaultConfig()

	if len(cfg.Brokers) == 0 {
		t.Error("默认配置应该包含broker地址")
	}

	if cfg.ClientID == "" {
		t.Error("默认配置应该包含客户端ID")
	}

	if cfg.ProducerConfig.Compression == "" {
		t.Error("默认配置应该包含压缩设置")
	}

	if cfg.ConsumerConfig.GroupID != "" {
		t.Error("默认消费者配置的GroupID应该为空")
	}
}

// TestProducerConfig 测试生产者配置
func TestProducerConfig(t *testing.T) {
	cfg := DefaultProducerConfig()

	if cfg.Compression != "lz4" {
		t.Errorf("期望压缩算法为lz4，实际为%s", cfg.Compression)
	}

	if cfg.BatchSize <= 0 {
		t.Error("批次大小应该大于0")
	}

	if cfg.LingerMs <= 0 {
		t.Error("等待时间应该大于0")
	}

	if !cfg.EnableIdempotence {
		t.Error("应该启用幂等性")
	}
}

// TestConsumerConfig 测试消费者配置
func TestConsumerConfig(t *testing.T) {
	cfg := DefaultConsumerConfig()

	if cfg.AutoOffsetReset != "latest" {
		t.Errorf("期望自动偏移量重置为latest，实际为%s", cfg.AutoOffsetReset)
	}

	if !cfg.EnableAutoCommit {
		t.Error("应该启用自动提交")
	}

	if cfg.SessionTimeout <= 0 {
		t.Error("会话超时时间应该大于0")
	}

	if cfg.HeartbeatInterval <= 0 {
		t.Error("心跳间隔应该大于0")
	}
}

// TestErrorTypes 测试错误类型
func TestErrorTypes(t *testing.T) {
	// 测试MQError
	err := internal.NewMQError("TEST_CODE", "测试错误", nil)
	if err.Code != "TEST_CODE" {
		t.Errorf("期望错误代码为TEST_CODE，实际为%s", err.Code)
	}

	if err.Message != "测试错误" {
		t.Errorf("期望错误消息为'测试错误'，实际为%s", err.Message)
	}

	// 测试错误上下文
	err.WithContext("key", "value")
	if err.Context["key"] != "value" {
		t.Error("错误上下文设置失败")
	}

	// 测试可重试错误判断
	if !IsRetryableError(internal.ErrConnectionFailed) {
		t.Error("连接失败错误应该是可重试的")
	}

	if IsRetryableError(internal.ErrAuthenticationFailed) {
		t.Error("认证失败错误不应该是可重试的")
	}

	// 测试致命错误判断
	if !IsFatalError(internal.ErrAuthenticationFailed) {
		t.Error("认证失败错误应该是致命的")
	}

	if IsFatalError(internal.ErrConnectionFailed) {
		t.Error("连接失败错误不应该是致命的")
	}
}

// TestMessageSerialization 测试消息序列化
func TestMessageSerialization(t *testing.T) {
	utils := NewMessageUtils("json", "lz4")

	// 测试序列化和压缩
	testData := map[string]interface{}{
		"message":   "Hello, World!",
		"timestamp": time.Now().Unix(),
		"user_id":   12345,
	}

	compressed, err := utils.SerializeAndCompress(testData)
	if err != nil {
		t.Fatalf("序列化和压缩失败: %v", err)
	}

	// 测试解压和反序列化
	var result map[string]interface{}
	err = utils.DecompressAndDeserialize(compressed, &result)
	if err != nil {
		t.Fatalf("解压和反序列化失败: %v", err)
	}

	if result["message"] != testData["message"] {
		t.Error("反序列化后数据不匹配")
	}
}

// TestCompressionRatio 测试压缩比
func TestCompressionRatio(t *testing.T) {
	utils := NewMessageUtils("json", "lz4")

	// 创建一个较大的测试数据
	testData := make([]byte, 1024)
	for i := range testData {
		testData[i] = byte(i % 256)
	}

	ratio, err := utils.GetCompressionRatio(testData)
	if err != nil {
		t.Fatalf("计算压缩比失败: %v", err)
	}

	if ratio <= 0 || ratio > 1 {
		t.Errorf("压缩比应该在0-1之间，实际为%f", ratio)
	}
}

// TestSmallMessageOptimization 测试小消息优化
func TestSmallMessageOptimization(t *testing.T) {
	utils := NewMessageUtils("json", "lz4")

	// 测试小消息（不压缩）
	smallData := []byte("small message")
	optimized, err := utils.OptimizeForSmallMessages(smallData, "text")
	if err != nil {
		t.Fatalf("小消息优化失败: %v", err)
	}

	// 小消息应该不被压缩
	if len(optimized) != len(smallData) {
		t.Error("小消息不应该被压缩")
	}

	// 测试是否有益压缩
	if utils.IsCompressionBeneficial(smallData, 100) {
		t.Error("小消息压缩应该被认为是无益的")
	}
}

// BenchmarkProducerLatency 基准测试生产者延迟
func BenchmarkProducerLatency(b *testing.B) {
	// 注意：这个测试需要实际的Kafka环境
	// 在CI/CD环境中可能需要跳过
	if testing.Short() {
		b.Skip("跳过需要Kafka环境的基准测试")
	}

	cfg := DefaultProducerConfig()
	cfg.Brokers = []string{"localhost:9092"}
	cfg.ClientID = "benchmark-producer"

	producer, err := NewProducer(cfg)
	if err != nil {
		b.Fatalf("创建生产者失败: %v", err)
	}
	defer producer.Close()

	ctx := context.Background()
	message := []byte("benchmark message")

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			start := time.Now()
			err := producer.SendSync(ctx, "benchmark-topic", message)
			latency := time.Since(start)

			if err != nil {
				b.Errorf("发送消息失败: %v", err)
			}

			// 验证微秒级延迟要求（1毫秒 = 1000微秒）
			if latency > time.Millisecond {
				b.Logf("延迟超过1毫秒: %v", latency)
			}
		}
	})
}

// BenchmarkConsumerThroughput 基准测试消费者吞吐量
func BenchmarkConsumerThroughput(b *testing.B) {
	if testing.Short() {
		b.Skip("跳过需要Kafka环境的基准测试")
	}

	cfg := DefaultConsumerConfig()
	cfg.Brokers = []string{"localhost:9092"}
	cfg.ClientID = "benchmark-consumer"
	cfg.GroupID = "benchmark-group"

	consumer, err := NewConsumer(cfg)
	if err != nil {
		b.Fatalf("创建消费者失败: %v", err)
	}
	defer consumer.Close()

	var messageCount int64
	var wg sync.WaitGroup

	callback := func(message *Message, partition TopicPartition, err error) bool {
		if err != nil {
			b.Errorf("消费消息失败: %v", err)
			return false
		}
		messageCount++
		wg.Done()
		return true
	}

	ctx := context.Background()
	err = consumer.Subscribe(ctx, []string{"benchmark-topic"}, callback)
	if err != nil {
		b.Fatalf("订阅主题失败: %v", err)
	}

	// 预期消费N条消息
	expectedMessages := int64(b.N)
	wg.Add(int(expectedMessages))

	b.ResetTimer()
	start := time.Now()

	// 等待消费完成或超时
	done := make(chan struct{})
	go func() {
		wg.Wait()
		close(done)
	}()

	select {
	case <-done:
		elapsed := time.Since(start)
		throughput := float64(messageCount) / elapsed.Seconds()

		// 验证吞吐量要求（100,000消息/秒）
		if throughput < 100000 {
			b.Logf("吞吐量低于要求: %.2f 消息/秒", throughput)
		}

		b.ReportMetric(throughput, "messages/sec")

	case <-time.After(30 * time.Second):
		b.Fatal("消费超时")
	}
}

// TestConcurrentProducer 测试并发生产者
func TestConcurrentProducer(t *testing.T) {
	if testing.Short() {
		t.Skip("跳过需要Kafka环境的并发测试")
	}

	cfg := DefaultProducerConfig()
	cfg.Brokers = []string{"localhost:9092"}
	cfg.ClientID = "concurrent-producer"

	producer, err := NewProducer(cfg)
	if err != nil {
		t.Fatalf("创建生产者失败: %v", err)
	}
	defer producer.Close()

	const numGoroutines = 10
	const messagesPerGoroutine = 100

	var wg sync.WaitGroup
	var successCount int64
	var errorCount int64

	ctx := context.Background()

	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func(goroutineID int) {
			defer wg.Done()

			for j := 0; j < messagesPerGoroutine; j++ {
				message := []byte("concurrent message")
				err := producer.SendSync(ctx, "concurrent-topic", message)
				if err != nil {
					t.Errorf("发送消息失败: %v", err)
					errorCount++
				} else {
					successCount++
				}
			}
		}(i)
	}

	wg.Wait()

	expectedMessages := int64(numGoroutines * messagesPerGoroutine)
	if successCount != expectedMessages {
		t.Errorf("期望成功发送%d条消息，实际成功%d条", expectedMessages, successCount)
	}

	if errorCount > 0 {
		t.Errorf("发生了%d个错误", errorCount)
	}
}

// TestConnectionPool 测试连接池
func TestConnectionPool(t *testing.T) {
	cfg := DefaultConfig()
	cfg.PoolConfig.MaxConnections = 5
	cfg.PoolConfig.MinIdleConnections = 2

	pool, err := internal.NewConnectionPool(cfg)
	if err != nil {
		t.Fatalf("创建连接池失败: %v", err)
	}
	defer pool.Close()

	ctx := context.Background()

	// 测试获取连接
	conn, err := pool.GetConnection(ctx)
	if err != nil {
		t.Fatalf("获取连接失败: %v", err)
	}

	// 测试释放连接
	err = pool.ReleaseConnection(conn)
	if err != nil {
		t.Fatalf("释放连接失败: %v", err)
	}

	// 测试连接池统计
	stats := pool.GetStats()
	if stats.MaxConnections != 5 {
		t.Errorf("期望最大连接数为5，实际为%d", stats.MaxConnections)
	}
}

// TestMetricsCollection 测试指标收集
func TestMetricsCollection(t *testing.T) {
	collector := internal.NewMetricsCollector()

	// 记录延迟
	collector.RecordLatency("send", 100*time.Microsecond)
	collector.RecordLatency("send", 200*time.Microsecond)

	// 记录吞吐量
	collector.RecordThroughput("produce", 100, 1024)

	// 记录错误
	collector.RecordError("send", "timeout", internal.ErrTimeout)

	// 获取指标
	metrics := collector.GetMetrics()

	if metrics["latency"] == nil {
		t.Error("应该包含延迟指标")
	}

	if metrics["throughput"] == nil {
		t.Error("应该包含吞吐量指标")
	}

	if metrics["errors"] == nil {
		t.Error("应该包含错误指标")
	}
}

// TestHealthCheck 测试健康检查
func TestHealthCheck(t *testing.T) {
	checker := internal.NewHealthChecker()

	// 注册健康检查
	checker.RegisterCheck("test", func(ctx context.Context) error {
		return nil // 健康
	})

	checker.RegisterCheck("failing", func(ctx context.Context) error {
		return internal.ErrConnectionFailed // 不健康
	})

	ctx := context.Background()
	status := checker.CheckHealth(ctx)

	if status.Overall {
		t.Error("整体状态应该是不健康的，因为有一个检查失败")
	}

	if len(status.Checks) != 2 {
		t.Errorf("期望2个检查结果，实际为%d", len(status.Checks))
	}

	if !status.Checks["test"].Healthy {
		t.Error("test检查应该是健康的")
	}

	if status.Checks["failing"].Healthy {
		t.Error("failing检查应该是不健康的")
	}
}
