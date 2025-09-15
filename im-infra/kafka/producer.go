package kafka

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/ceyewan/gochat/im-infra/clog"
	"github.com/twmb/franz-go/pkg/kgo"
)

// producerImpl 实现 Producer 接口
type producerImpl struct {
	client *kgo.Client
	config *Config
	logger clog.Logger
	metrics producerMetrics
}

// producerMetrics 生产者性能指标
type producerMetrics struct {
	totalMessages   int64
	totalBytes      int64
	successMessages int64
	failedMessages  int64
	mu              sync.RWMutex
}

// newProducerImpl 创建一个新的消息生产者实例。
func newProducerImpl(ctx context.Context, config *Config, opts *options) (*producerImpl, error) {
	if opts == nil {
		opts = &options{
			logger: clog.Namespace("kafka-producer"),
		}
	}

	if config.ProducerConfig == nil {
		return nil, fmt.Errorf("生产者配置不能为空")
	}

	// 构建 franz-go 客户端配置
	kgoOpts := buildProducerOpts(config.ProducerConfig)

	// 设置 brokers
	kgoOpts = append(kgoOpts, kgo.SeedBrokers(config.Brokers...))

	// 设置安全协议（暂时只支持 PLAINTEXT）
	if config.SecurityProtocol != "PLAINTEXT" {
		// TODO: 后续可以扩展支持 SSL/SASL 配置
		// 当前只支持 PLAINTEXT 协议
	}

	client, err := kgo.NewClient(kgoOpts...)
	if err != nil {
		return nil, fmt.Errorf("创建 Kafka 客户端失败: %w", err)
	}

	producer := &producerImpl{
		client:  client,
		config:  config,
		logger:  opts.logger,
		metrics: producerMetrics{},
	}

	producer.logger.Info("Kafka 生产者初始化成功",
		clog.Strings("brokers", config.Brokers),
		clog.Int("batch_size", config.ProducerConfig.BatchSize),
		clog.Int("retry_max", config.ProducerConfig.RetryMax),
	)

	return producer, nil
}

// buildProducerOpts 构建生产者选项
func buildProducerOpts(cfg *ProducerConfig) []kgo.Opt {
	opts := []kgo.Opt{
		kgo.AllowAutoTopicCreation(),
		kgo.ProducerBatchMaxBytes(int32(cfg.BatchSize)),
		kgo.ProducerLinger(time.Duration(cfg.LingerMs) * time.Millisecond),
		kgo.RecordRetries(cfg.RetryMax),
		kgo.UnknownTopicRetries(cfg.UnknownTopicRetries),
		kgo.RequestRetries(cfg.RetryMax),
		kgo.RequestTimeoutOverhead(time.Duration(cfg.RequestTimeoutMs) * time.Millisecond),
		kgo.ProduceRequestTimeout(time.Duration(cfg.DeliveryTimeoutMs) * time.Millisecond),
		kgo.MaxProduceRequestsInflightPerBroker(cfg.MaxInFlightRequestsPerBroker),
	}

	// 设置确认级别
	var acks kgo.Acks
	switch cfg.Acks {
	case 0:
		acks = kgo.NoAck()
	case 1:
		acks = kgo.LeaderAck()
	case -1:
		acks = kgo.AllISRAcks()
	default:
		acks = kgo.LeaderAck()
	}
	opts = append(opts, kgo.RequiredAcks(acks))

	// 设置幂等性
	if cfg.EnableIdempotence {
		// 启用幂等性（默认启用）
	} else {
		opts = append(opts, kgo.DisableIdempotentWrite())
	}

	// 设置压缩
	switch cfg.Compression {
	case "gzip":
		opts = append(opts, kgo.ProducerBatchCompression(kgo.GzipCompression()))
	case "snappy":
		opts = append(opts, kgo.ProducerBatchCompression(kgo.SnappyCompression()))
	case "lz4":
		opts = append(opts, kgo.ProducerBatchCompression(kgo.Lz4Compression()))
	case "zstd":
		opts = append(opts, kgo.ProducerBatchCompression(kgo.ZstdCompression()))
	default:
		opts = append(opts, kgo.ProducerBatchCompression(kgo.NoCompression()))
	}

	// 设置缓冲区限制
	if cfg.MaxBufferedRecords > 0 {
		opts = append(opts, kgo.MaxBufferedRecords(cfg.MaxBufferedRecords))
	}
	if cfg.MaxBufferedBytes > 0 {
		opts = append(opts, kgo.MaxBufferedBytes(cfg.MaxBufferedBytes))
	}

	// 设置重试策略
	opts = append(opts, kgo.RetryBackoffFn(func(tries int) time.Duration {
		// 指数退避策略
		backoff := time.Duration(tries*tries) * 100 * time.Millisecond
		if backoff > 5*time.Second {
			backoff = 5 * time.Second
		}
		return backoff
	}))

	return opts
}

// Send 异步发送消息。
func (p *producerImpl) Send(ctx context.Context, msg *Message, callback func(error)) {
	// 参数校验
	if msg == nil {
		if callback != nil {
			callback(fmt.Errorf("消息不能为空"))
		}
		return
	}

	if msg.Topic == "" {
		if callback != nil {
			callback(fmt.Errorf("消息主题不能为空"))
		}
		return
	}

	// 更新指标
	p.metrics.mu.Lock()
	p.metrics.totalMessages++
	p.metrics.totalBytes += int64(len(msg.Value))
	p.metrics.mu.Unlock()

	// 自动注入 trace_id 到消息头
	if msg.Headers == nil {
		msg.Headers = make(map[string][]byte)
	}

	if traceID := extractTraceID(ctx); traceID != "" {
		msg.Headers["X-Trace-ID"] = []byte(traceID)
	}

	// 添加时间戳头
	msg.Headers["X-Timestamp"] = []byte(time.Now().Format(time.RFC3339))

	// 转换为 franz-go 消息格式
	record := &kgo.Record{
		Topic:   msg.Topic,
		Key:     msg.Key,
		Value:   msg.Value,
		Headers: convertHeaders(msg.Headers),
	}

	// 异步发送
	p.client.Produce(ctx, record, func(r *kgo.Record, err error) {
		if err != nil {
			p.metrics.mu.Lock()
			p.metrics.failedMessages++
			p.metrics.mu.Unlock()

			p.logger.Error("发送消息失败",
				clog.Err(err),
				clog.String("topic", msg.Topic),
				clog.String("key", string(msg.Key)),
				clog.Int("value_size", len(msg.Value)),
			)
		} else {
			p.metrics.mu.Lock()
			p.metrics.successMessages++
			p.metrics.mu.Unlock()

			p.logger.Debug("发送消息成功",
				clog.String("topic", msg.Topic),
				clog.String("key", string(msg.Key)),
				clog.Int64("offset", r.Offset),
				clog.Int32("partition", r.Partition),
			)
		}

		if callback != nil {
			callback(err)
		}
	})
}

// SendSync 同步发送消息。
func (p *producerImpl) SendSync(ctx context.Context, msg *Message) error {
	// 参数校验
	if msg == nil {
		return fmt.Errorf("消息不能为空")
	}

	if msg.Topic == "" {
		return fmt.Errorf("消息主题不能为空")
	}

	// 更新指标
	p.metrics.mu.Lock()
	p.metrics.totalMessages++
	p.metrics.totalBytes += int64(len(msg.Value))
	p.metrics.mu.Unlock()

	// 检查上下文是否已取消
	if ctx.Err() != nil {
		return ctx.Err()
	}

	// 自动注入 trace_id 到消息头
	if msg.Headers == nil {
		msg.Headers = make(map[string][]byte)
	}

	if traceID := extractTraceID(ctx); traceID != "" {
		msg.Headers["X-Trace-ID"] = []byte(traceID)
	}

	// 添加时间戳头
	msg.Headers["X-Timestamp"] = []byte(time.Now().Format(time.RFC3339))

	// 转换为 franz-go 消息格式
	record := &kgo.Record{
		Topic:   msg.Topic,
		Key:     msg.Key,
		Value:   msg.Value,
		Headers: convertHeaders(msg.Headers),
	}

	// 同步发送
	results := p.client.ProduceSync(ctx, record)
	if results.FirstErr() != nil {
		p.metrics.mu.Lock()
		p.metrics.failedMessages++
		p.metrics.mu.Unlock()

		p.logger.Error("同步发送消息失败",
			clog.Err(results.FirstErr()),
			clog.String("topic", msg.Topic),
			clog.String("key", string(msg.Key)),
			clog.Int("value_size", len(msg.Value)),
		)
		return results.FirstErr()
	}

	p.metrics.mu.Lock()
	p.metrics.successMessages++
	p.metrics.mu.Unlock()

	p.logger.Debug("同步发送消息成功",
		clog.String("topic", msg.Topic),
		clog.String("key", string(msg.Key)),
	)

	return nil
}

// Close 关闭生产者。
func (p *producerImpl) Close() error {
	p.logger.Info("关闭 Kafka 生产者",
		clog.Int64("total_messages", p.metrics.totalMessages),
		clog.Int64("success_messages", p.metrics.successMessages),
		clog.Int64("failed_messages", p.metrics.failedMessages),
		clog.Int64("total_bytes", p.metrics.totalBytes),
	)

	// 刷新所有待发送的消息
	p.client.Flush(context.Background())

	// 关闭客户端
	p.client.Close()

	return nil
}

// GetMetrics 获取生产者性能指标
func (p *producerImpl) GetMetrics() map[string]interface{} {
	p.metrics.mu.RLock()
	defer p.metrics.mu.RUnlock()

	successRate := float64(0)
	if p.metrics.totalMessages > 0 {
		successRate = float64(p.metrics.successMessages) / float64(p.metrics.totalMessages) * 100
	}

	return map[string]interface{}{
		"total_messages":   p.metrics.totalMessages,
		"success_messages": p.metrics.successMessages,
		"failed_messages":  p.metrics.failedMessages,
		"total_bytes":      p.metrics.totalBytes,
		"success_rate":     successRate,
	}
}

// Flush 刷新所有待发送的消息
func (p *producerImpl) Flush(ctx context.Context) error {
	p.logger.Debug("刷新生产者缓冲区")
	return p.client.Flush(ctx)
}

// Ping 检查生产者健康状态
func (p *producerImpl) Ping(ctx context.Context) error {
	p.logger.Debug("检查生产者健康状态")

	// 检查连接状态
	if len(p.client.SeedBrokers()) == 0 {
		return fmt.Errorf("没有可用的 seed broker 连接")
	}

	// 尝试 Ping 客户端
	if err := p.client.Ping(ctx); err != nil {
		return fmt.Errorf("客户端 Ping 失败: %w", err)
	}

	return nil
}

// convertHeaders 转换消息头格式
func convertHeaders(headers map[string][]byte) []kgo.RecordHeader {
	if len(headers) == 0 {
		return nil
	}

	kgoHeaders := make([]kgo.RecordHeader, 0, len(headers))
	for key, value := range headers {
		kgoHeaders = append(kgoHeaders, kgo.RecordHeader{
			Key:   key,
			Value: value,
		})
	}
	return kgoHeaders
}

// GetClient 获取底层的 kgo.Client，用于高级操作
func (p *producerImpl) GetClient() *kgo.Client {
	return p.client
}

// extractTraceID 从上下文中提取 trace_id
func extractTraceID(ctx context.Context) string {
	if ctx == nil {
		return ""
	}

	// 从 context.Value 中提取 trace ID
	if traceID, ok := ctx.Value(TraceIDKey).(string); ok {
		return traceID
	}

	return ""
}