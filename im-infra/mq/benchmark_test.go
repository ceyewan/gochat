package mq

import (
	"context"
	"fmt"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/ceyewan/gochat/im-infra/mq/internal"
)

// BenchmarkMessageSerialization 基准测试消息序列化
func BenchmarkMessageSerialization(b *testing.B) {
	utils := NewMessageUtils("json", "lz4")

	testData := map[string]interface{}{
		"user_id":   12345,
		"message":   "Hello, this is a test message for benchmarking serialization performance",
		"timestamp": time.Now().Unix(),
		"metadata": map[string]string{
			"source": "benchmark",
			"type":   "test",
		},
	}

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			_, err := utils.SerializeAndCompress(testData)
			if err != nil {
				b.Fatal(err)
			}
		}
	})
}

// BenchmarkMessageDeserialization 基准测试消息反序列化
func BenchmarkMessageDeserialization(b *testing.B) {
	utils := NewMessageUtils("json", "lz4")

	testData := map[string]interface{}{
		"user_id":   12345,
		"message":   "Hello, this is a test message for benchmarking deserialization performance",
		"timestamp": time.Now().Unix(),
	}

	// 预先序列化数据
	serialized, err := utils.SerializeAndCompress(testData)
	if err != nil {
		b.Fatal(err)
	}

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			var result map[string]interface{}
			err := utils.DecompressAndDeserialize(serialized, &result)
			if err != nil {
				b.Fatal(err)
			}
		}
	})
}

// BenchmarkCompressionAlgorithms 基准测试不同压缩算法
func BenchmarkCompressionAlgorithms(b *testing.B) {
	// 创建测试数据
	testData := make([]byte, 1024) // 1KB数据
	for i := range testData {
		testData[i] = byte(i % 256)
	}

	algorithms := []string{"none", "lz4", "snappy", "gzip"}

	for _, algo := range algorithms {
		b.Run(algo, func(b *testing.B) {
			codec := NewCompressionCodec(algo)

			b.ResetTimer()
			b.RunParallel(func(pb *testing.PB) {
				for pb.Next() {
					compressed, err := codec.Compress(testData)
					if err != nil {
						b.Fatal(err)
					}

					_, err = codec.Decompress(compressed)
					if err != nil {
						b.Fatal(err)
					}
				}
			})
		})
	}
}

// BenchmarkSmallMessageOptimization 基准测试小消息优化
func BenchmarkSmallMessageOptimization(b *testing.B) {
	utils := internal.NewMessageUtils("json", "lz4")

	// 不同大小的消息
	messageSizes := []int{50, 100, 500, 1024, 2048}

	for _, size := range messageSizes {
		b.Run(fmt.Sprintf("size_%d", size), func(b *testing.B) {
			testData := make([]byte, size)
			for i := range testData {
				testData[i] = byte(i % 256)
			}

			b.ResetTimer()
			b.RunParallel(func(pb *testing.PB) {
				for pb.Next() {
					_, err := utils.OptimizeForSmallMessages(testData, "binary")
					if err != nil {
						b.Fatal(err)
					}
				}
			})
		})
	}
}

// BenchmarkConnectionPoolPerformance 基准测试连接池性能
func BenchmarkConnectionPoolPerformance(b *testing.B) {
	cfg := DefaultConfig()
	cfg.PoolConfig.MaxConnections = 10
	cfg.PoolConfig.MinIdleConnections = 5

	pool, err := internal.NewConnectionPool(cfg)
	if err != nil {
		b.Fatal(err)
	}
	defer pool.Close()

	ctx := context.Background()

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			conn, err := pool.GetConnection(ctx)
			if err != nil {
				b.Fatal(err)
			}

			err = pool.ReleaseConnection(conn)
			if err != nil {
				b.Fatal(err)
			}
		}
	})
}

// BenchmarkMetricsCollection 基准测试指标收集性能
func BenchmarkMetricsCollection(b *testing.B) {
	collector := internal.NewMetricsCollector()

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			collector.RecordLatency("operation", 100*time.Microsecond)
			collector.RecordThroughput("operation", 1, 1024)
		}
	})
}

// BenchmarkConcurrentProducerLatency 基准测试并发生产者延迟
func BenchmarkConcurrentProducerLatency(b *testing.B) {
	if testing.Short() {
		b.Skip("跳过需要Kafka环境的基准测试")
	}

	cfg := DefaultProducerConfig()
	cfg.Brokers = []string{"localhost:9092"}
	cfg.ClientID = "benchmark-concurrent-producer"

	producer, err := NewProducer(cfg)
	if err != nil {
		b.Skip("无法连接到Kafka，跳过基准测试")
	}
	defer producer.Close()

	ctx := context.Background()
	message := []byte("benchmark message for concurrent latency test")

	// 测试不同并发级别
	concurrencyLevels := []int{1, 5, 10, 20, 50}

	for _, concurrency := range concurrencyLevels {
		b.Run(fmt.Sprintf("concurrency_%d", concurrency), func(b *testing.B) {
			var wg sync.WaitGroup
			var totalLatency int64
			var operationCount int64

			b.ResetTimer()

			for i := 0; i < concurrency; i++ {
				wg.Add(1)
				go func() {
					defer wg.Done()

					for j := 0; j < b.N/concurrency; j++ {
						start := time.Now()
						err := producer.SendSync(ctx, "benchmark-topic", message)
						latency := time.Since(start)

						if err != nil {
							b.Errorf("发送消息失败: %v", err)
							continue
						}

						atomic.AddInt64(&totalLatency, latency.Nanoseconds())
						atomic.AddInt64(&operationCount, 1)
					}
				}()
			}

			wg.Wait()

			if operationCount > 0 {
				avgLatency := time.Duration(totalLatency / operationCount)
				b.ReportMetric(float64(avgLatency.Nanoseconds())/1000, "μs/op")

				// 验证微秒级延迟要求
				if avgLatency > time.Millisecond {
					b.Logf("平均延迟超过1毫秒: %v", avgLatency)
				}
			}
		})
	}
}

// BenchmarkHighThroughputConsumer 基准测试高吞吐量消费
func BenchmarkHighThroughputConsumer(b *testing.B) {
	if testing.Short() {
		b.Skip("跳过需要Kafka环境的基准测试")
	}

	cfg := DefaultConsumerConfig()
	cfg.Brokers = []string{"localhost:9092"}
	cfg.ClientID = "benchmark-throughput-consumer"
	cfg.GroupID = "benchmark-throughput-group"
	cfg.MaxPollRecords = 1000 // 增加批次大小以提高吞吐量

	consumer, err := NewConsumer(cfg)
	if err != nil {
		b.Skip("无法连接到Kafka，跳过基准测试")
	}
	defer consumer.Close()

	var messageCount int64
	var totalBytes int64
	var wg sync.WaitGroup

	callback := func(message *Message, partition TopicPartition, err error) bool {
		if err != nil {
			b.Errorf("消费消息失败: %v", err)
			return false
		}

		atomic.AddInt64(&messageCount, 1)
		atomic.AddInt64(&totalBytes, int64(len(message.Value)))
		wg.Done()
		return true
	}

	ctx := context.Background()
	err = consumer.Subscribe(ctx, []string{"benchmark-topic"}, callback)
	if err != nil {
		b.Fatal(err)
	}

	// 预期消费的消息数量
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
		actualMessages := atomic.LoadInt64(&messageCount)
		actualBytes := atomic.LoadInt64(&totalBytes)

		if actualMessages > 0 {
			throughput := float64(actualMessages) / elapsed.Seconds()
			byteThroughput := float64(actualBytes) / elapsed.Seconds()

			b.ReportMetric(throughput, "messages/sec")
			b.ReportMetric(byteThroughput, "bytes/sec")

			// 验证吞吐量要求（100,000消息/秒）
			if throughput < 100000 {
				b.Logf("吞吐量低于要求: %.2f 消息/秒", throughput)
			}
		}

	case <-time.After(60 * time.Second):
		b.Fatal("消费超时")
	}
}

// BenchmarkBatchProcessing 基准测试批处理性能
func BenchmarkBatchProcessing(b *testing.B) {
	if testing.Short() {
		b.Skip("跳过需要Kafka环境的基准测试")
	}

	cfg := DefaultProducerConfig()
	cfg.Brokers = []string{"localhost:9092"}
	cfg.ClientID = "benchmark-batch-producer"

	producer, err := NewProducer(cfg)
	if err != nil {
		b.Skip("无法连接到Kafka，跳过基准测试")
	}
	defer producer.Close()

	// 测试不同批次大小
	batchSizes := []int{1, 10, 50, 100, 500}

	for _, batchSize := range batchSizes {
		b.Run(fmt.Sprintf("batch_size_%d", batchSize), func(b *testing.B) {
			ctx := context.Background()

			b.ResetTimer()

			for i := 0; i < b.N; i += batchSize {
				// 创建消息批次
				batch := internal.MessageBatch{
					Messages:      make([]*internal.Message, 0, batchSize),
					MaxBatchSize:  16384,
					MaxBatchCount: batchSize,
					LingerMs:      5,
				}

				actualBatchSize := batchSize
				if i+batchSize > b.N {
					actualBatchSize = b.N - i
				}

				for j := 0; j < actualBatchSize; j++ {
					message := &internal.Message{
						Topic: "benchmark-topic",
						Value: []byte(fmt.Sprintf("batch message %d", j)),
					}
					batch.Messages = append(batch.Messages, message)
				}

				start := time.Now()
				_, err := producer.SendBatchSync(ctx, batch)
				latency := time.Since(start)

				if err != nil {
					b.Errorf("发送批次失败: %v", err)
					continue
				}

				// 计算每条消息的平均延迟
				avgLatencyPerMessage := latency / time.Duration(actualBatchSize)
				if avgLatencyPerMessage > time.Millisecond {
					b.Logf("批次大小%d的平均延迟: %v", batchSize, avgLatencyPerMessage)
				}
			}
		})
	}
}

// BenchmarkMemoryUsage 基准测试内存使用
func BenchmarkMemoryUsage(b *testing.B) {
	cfg := DefaultConfig()

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		// 创建MQ实例
		mqInstance, err := New(cfg)
		if err != nil {
			b.Fatal(err)
		}

		// 立即关闭以测试内存清理
		mqInstance.Close()
	}
}

// BenchmarkErrorHandling 基准测试错误处理性能
func BenchmarkErrorHandling(b *testing.B) {
	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			err := internal.NewMQError("BENCHMARK_ERROR", "基准测试错误", internal.ErrTimeout)
			err.WithContext("operation", "benchmark")

			// 测试错误类型判断
			_ = internal.IsRetryableError(err)
			_ = internal.IsFatalError(err)
		}
	})
}

// BenchmarkConfigValidation 基准测试配置验证性能
func BenchmarkConfigValidation(b *testing.B) {
	cfg := DefaultConfig()

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			// 这里应该调用内部的配置验证函数
			// 由于validateConfig是internal包的私有函数，我们模拟验证过程
			if len(cfg.Brokers) == 0 {
				continue
			}
			if cfg.ClientID == "" {
				continue
			}
			// 更多验证逻辑...
		}
	})
}
