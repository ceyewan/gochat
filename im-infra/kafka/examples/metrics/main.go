package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"

	"github.com/ceyewan/gochat/im-infra/clog"
	"github.com/ceyewan/gochat/im-infra/kafka"
)

// MetricsEvent 指标事件
type MetricsEvent struct {
	EventType   string                 `json:"event_type"`
	Timestamp   time.Time              `json:"timestamp"`
	Value       float64                `json:"value"`
	Tags        map[string]string      `json:"tags,omitempty"`
	Dimensions  map[string]interface{} `json:"dimensions,omitempty"`
}

func main() {
	ctx := context.Background()

	// 初始化 clog
	clog.Init(ctx, clog.GetDefaultConfig("development"))

	// 获取 Kafka 配置
	config := kafka.GetDefaultConfig("development")
	config.Brokers = []string{"localhost:9092"}

	// 创建 Provider
	provider, err := kafka.NewProvider(ctx, config, kafka.WithNamespace("metrics-example"))
	if err != nil {
		log.Fatal("创建 Provider 失败:", err)
	}
	defer provider.Close()

	// 获取组件
	producer := provider.Producer()
	consumer1 := provider.Consumer("metrics-processor-1")
	consumer2 := provider.Consumer("metrics-processor-2")
	admin := provider.Admin()

	// 设置 topic
	setupMetricsTopic(ctx, admin)

	// 启动多个消费者组展示负载均衡
	var wg sync.WaitGroup
	wg.Add(2)

	go func() {
		defer wg.Done()
		startMetricsConsumer(ctx, consumer1, "Processor-1")
	}()

	go func() {
		defer wg.Done()
		startMetricsConsumer(ctx, consumer2, "Processor-2")
	}()

	// 启动指标收集器
	go startMetricsCollector(ctx, producer)
	go startMetricsReporter(ctx, provider)

	// 等待退出信号
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
	<-sigChan

	fmt.Println("\n正在关闭程序...")
	wg.Wait()
}

// setupMetricsTopic 设置指标 topic
func setupMetricsTopic(ctx context.Context, admin kafka.AdminOperations) error {
	// 创建高吞吐量的指标 topic
	if err := admin.CreateTopic(ctx, "metrics.stream", 6, 2, map[string]string{
		"retention.ms":      "86400000",  // 1天保留期
		"cleanup.policy":    "delete",
		"compression.type":  "lz4",        // 启用压缩
		"max.message.bytes": "1048576",    // 1MB 最大消息大小
	}); err != nil {
		fmt.Printf("创建 metrics.stream topic 失败（可能已存在）: %v\n", err)
	}
	return nil
}

// startMetricsConsumer 启动指标消费者
func startMetricsConsumer(ctx context.Context, consumer kafka.ConsumerOperations, processorID string) {
	topics := []string{"metrics.stream"}

	handler := func(ctx context.Context, msg *kafka.Message) error {
		// 设置 trace ID
		if traceID, ok := ctx.Value(kafka.TraceIDKey).(string); ok {
			ctx = clog.WithTraceID(ctx, traceID)
		}

		logger := clog.WithContext(ctx)
		startTime := time.Now()

		// 解析指标事件
		var event MetricsEvent
		if err := json.Unmarshal(msg.Value, &event); err != nil {
			logger.Error("解析指标事件失败",
				clog.Err(err),
				clog.String("processor", processorID),
			)
			return err
		}

		// 处理指标
		logger.Info("处理指标事件",
			clog.String("processor", processorID),
			clog.String("event_type", event.EventType),
			clog.Float64("value", event.Value),
			clog.String("topic", msg.Topic),
			clog.String("partition", fmt.Sprintf("%d", msg.Partition)),
		)

		// 模拟处理时间
		processingTime := time.Duration(10+time.Now().UnixNano()%30) * time.Millisecond
		time.Sleep(processingTime)

		// 记录处理成功
		logger.Info("指标事件处理完成",
			clog.String("processor", processorID),
			clog.String("event_type", event.EventType),
			clog.Duration("processing_time", processingTime),
		)

		return nil
	}

	logger := clog.WithContext(ctx)
	logger.Info("启动指标消费者", clog.String("processor", processorID))

	if err := consumer.Subscribe(ctx, topics, handler); err != nil {
		logger.Error("消费者订阅失败",
			clog.Err(err),
			clog.String("processor", processorID),
		)
	}
}

// startMetricsCollector 启动指标收集器
func startMetricsCollector(ctx context.Context, producer kafka.ProducerOperations) {
	logger := clog.WithContext(ctx)
	logger.Info("启动指标收集器")

	eventTypes := []string{"cpu_usage", "memory_usage", "request_count", "error_count", "response_time"}

	ticker := time.NewTicker(500 * time.Millisecond)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			// 生成随机指标事件
			eventType := eventTypes[time.Now().UnixNano()%int64(len(eventTypes))]

			event := MetricsEvent{
				EventType: eventType,
				Timestamp: time.Now(),
				Value:     generateRandomValue(eventType),
				Tags: map[string]string{
					"host":     fmt.Sprintf("host-%d", time.Now().UnixNano()%5+1),
					"service":  fmt.Sprintf("service-%d", time.Now().UnixNano()%3+1),
					"env":      "production",
				},
				Dimensions: map[string]interface{}{
					"version": "1.0.0",
					"region":  []string{"us-east-1", "eu-west-1"}[time.Now().UnixNano()%2],
				},
			}

			data, err := json.Marshal(event)
			if err != nil {
				logger.Error("序列化指标事件失败", clog.Err(err))
				continue
			}

			msg := &kafka.Message{
				Topic: "metrics.stream",
				Key:   []byte(event.EventType),
				Value: data,
				Headers: map[string][]byte{
					"content-type": []byte("application/json"),
					"metric-type":  []byte(eventType),
				},
			}

			// 添加 trace ID
			traceID := fmt.Sprintf("metrics-%d", time.Now().UnixNano())
			traceCtx := context.WithValue(ctx, kafka.TraceIDKey, traceID)

			// 异步发送
			producer.Send(traceCtx, msg, func(err error) {
				if err != nil {
					logger.Error("发送指标事件失败",
						clog.Err(err),
						clog.String("event_type", eventType),
					)
				}
			})
		}
	}
}

// startMetricsReporter 启动指标报告器
func startMetricsReporter(ctx context.Context, provider kafka.Provider) {
	logger := clog.WithContext(ctx)
	logger.Info("启动指标报告器")

	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			// 收集并报告所有组件的指标
			reportMetrics(ctx, provider)
		}
	}
}

// reportMetrics 报告所有组件的指标
func reportMetrics(ctx context.Context, provider kafka.Provider) {
	logger := clog.WithContext(ctx)

	// 获取生产者指标
	producerMetrics := provider.Producer().GetMetrics()
	logger.Info("=== 生产者指标 ===")
	for key, value := range producerMetrics {
		logger.Info("生产器指标",
			clog.String("metric", key),
			clog.Any("value", value),
		)
	}

	// 获取消费者指标
	consumer1Metrics := provider.Consumer("metrics-processor-1").GetMetrics()
	consumer2Metrics := provider.Consumer("metrics-processor-2").GetMetrics()

	logger.Info("=== 消费者-1 指标 ===")
	for key, value := range consumer1Metrics {
		logger.Info("消费者-1 指标",
			clog.String("metric", key),
			clog.Any("value", value),
		)
	}

	logger.Info("=== 消费者-2 指标 ===")
	for key, value := range consumer2Metrics {
		logger.Info("消费者-2 指标",
			clog.String("metric", key),
			clog.Any("value", value),
		)
	}

	// 获取 admin 操作指标
	admin := provider.Admin()
	if topics, err := admin.ListTopics(ctx); err == nil {
		logger.Info("=== 集群信息 ===")
		logger.Info("Topic 数量", clog.Int("count", len(topics)))
		for name, detail := range topics {
			logger.Info("Topic 详情",
				clog.String("name", name),
				clog.Int32("partitions", detail.NumPartitions),
				clog.Int16("replication_factor", detail.ReplicationFactor),
			)
		}
	}

	logger.Info("=== 指标报告完成 ===\n")
}

// generateRandomValue 根据事件类型生成随机值
func generateRandomValue(eventType string) float64 {
	switch eventType {
	case "cpu_usage":
		return float64(20+time.Now().UnixNano()%60) + float64(time.Now().UnixNano()%100)/100.0
	case "memory_usage":
		return float64(1024+time.Now().UnixNano()%4096) + float64(time.Now().UnixNano()%100)/100.0
	case "request_count":
		return float64(time.Now().UnixNano() % 1000)
	case "error_count":
		return float64(time.Now().UnixNano() % 10)
	case "response_time":
		return float64(50+time.Now().UnixNano()%200) + float64(time.Now().UnixNano()%1000)/1000.0
	default:
		return float64(time.Now().UnixNano() % 100)
	}
}