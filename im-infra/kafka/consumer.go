package kafka

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/ceyewan/gochat/im-infra/clog"
	"github.com/twmb/franz-go/pkg/kgo"
)

// consumer 实现 Consumer 接口
type consumer struct {
	client        *kgo.Client
	config        *Config
	groupID       string
	logger        clog.Logger
	metrics       consumerMetrics
	cancelContext context.CancelFunc
	wg            sync.WaitGroup
	ctx           context.Context
}

// consumerMetrics 消费者性能指标
type consumerMetrics struct {
	totalMessages    int64
	totalBytes       int64
	processedMessage int64
	failedMessages   int64
	mu               sync.RWMutex
}

// NewConsumer 创建一个新的消息消费者实例。
// groupID 是 Kafka 的消费者组ID，用于实现负载均衡和故障转移。
func NewConsumer(ctx context.Context, config *Config, groupID string, opts ...Option) (Consumer, error) {
	options := &options{
		logger: clog.Namespace("kafka-consumer"),
	}

	for _, opt := range opts {
		opt(options)
	}

	if config.ConsumerConfig == nil {
		return nil, fmt.Errorf("消费者配置不能为空")
	}

	if groupID == "" {
		return nil, fmt.Errorf("消费者组ID不能为空")
	}

	// 构建上下文
	consumerCtx, cancel := context.WithCancel(ctx)

	// 构建 franz-go 客户端配置
	kgoOpts := buildConsumerOpts(config.ConsumerConfig, groupID)

	// 设置 brokers
	kgoOpts = append(kgoOpts, kgo.SeedBrokers(config.Brokers...))

	// 设置安全协议（暂时只支持 PLAINTEXT）
	if config.SecurityProtocol != "PLAINTEXT" {
		// TODO: 后续可以扩展支持 SSL/SASL 配置
		// 当前只支持 PLAINTEXT 协议
	}

	client, err := kgo.NewClient(kgoOpts...)
	if err != nil {
		cancel()
		return nil, fmt.Errorf("创建 Kafka 客户端失败: %w", err)
	}

	consumer := &consumer{
		client:        client,
		config:        config,
		groupID:       groupID,
		logger:        options.logger,
		metrics:       consumerMetrics{},
		cancelContext: cancel,
		ctx:           consumerCtx,
	}

	consumer.logger.Info("Kafka 消费者初始化成功",
		clog.Strings("brokers", config.Brokers),
		clog.String("group_id", groupID),
		clog.String("auto_offset_reset", config.ConsumerConfig.AutoOffsetReset),
	)

	return consumer, nil
}

// buildConsumerOpts 构建消费者选项
func buildConsumerOpts(cfg *ConsumerConfig, groupID string) []kgo.Opt {
	opts := []kgo.Opt{
		kgo.ConsumerGroup(groupID),
		kgo.ConsumeTopics(), // 动态主题消费
		kgo.AutoCommitMarks(),
		kgo.AutoCommitInterval(time.Duration(cfg.AutoCommitIntervalMs) * time.Millisecond),
		kgo.SessionTimeout(time.Duration(cfg.SessionTimeoutMs) * time.Millisecond),
		kgo.HeartbeatInterval(time.Duration(cfg.HeartbeatIntervalMs) * time.Millisecond),
		kgo.FetchMaxBytes(int32(cfg.FetchMaxBytes)),
		kgo.FetchMinBytes(int32(cfg.FetchMinBytes)),
		kgo.FetchMaxWait(time.Duration(cfg.FetchMaxWaitMs) * time.Millisecond),
		kgo.FetchMaxPartitionBytes(int32(cfg.MaxPartitionFetchBytes)),
	}

	// 设置偏移量重置策略
	switch cfg.AutoOffsetReset {
	case "earliest":
		opts = append(opts, kgo.ConsumeResetOffset(kgo.NewOffset().AtStart()))
	case "latest":
		opts = append(opts, kgo.ConsumeResetOffset(kgo.NewOffset().AtEnd()))
	case "none":
		opts = append(opts, kgo.ConsumeResetOffset(kgo.NoResetOffset()))
	default:
		opts = append(opts, kgo.ConsumeResetOffset(kgo.NewOffset().AtStart()))
	}

	// 设置自动提交
	if !cfg.EnableAutoCommit {
		opts = append(opts, kgo.DisableAutoCommit())
	}

	// 设置 CRC 检查
	if !cfg.CheckCRCs {
		opts = append(opts, kgo.DisableFetchCRCValidation())
	}

	// 设置客户端 ID
	if cfg.ClientID != "" {
		opts = append(opts, kgo.ClientID(cfg.ClientID))
	}

	// 设置重平衡超时
	if cfg.RebalanceTimeoutMs > 0 {
		opts = append(opts, kgo.RebalanceTimeout(time.Duration(cfg.RebalanceTimeoutMs) * time.Millisecond))
	}

	return opts
}

// Subscribe 订阅消息并根据处理结果决定是否提交偏移量。
func (c *consumer) Subscribe(ctx context.Context, topics []string, callback ConsumeCallback) error {
	if len(topics) == 0 {
		return fmt.Errorf("订阅主题列表不能为空")
	}

	if callback == nil {
		return fmt.Errorf("回调函数不能为空")
	}

	// 添加主题到消费列表
	c.client.AddConsumeTopics(topics...)

	c.logger.Info("开始订阅主题",
		clog.Strings("topics", topics),
		clog.String("group_id", c.groupID),
	)

	// 启动消费 goroutine
	c.wg.Add(1)
	go func() {
		defer c.wg.Done()

		for {
			select {
			case <-ctx.Done():
				c.logger.Info("停止消费消息", clog.Err(ctx.Err()))
				return
			case <-c.ctx.Done():
				c.logger.Info("消费者被取消")
				return
			default:
				if err := c.consumeBatch(ctx, callback); err != nil {
					c.logger.Error("消费批次失败", clog.Err(err))
					// 短暂休眠后继续
					time.Sleep(1 * time.Second)
				}
			}
		}
	}()

	return nil
}

// consumeBatch 消费一批消息
func (c *consumer) consumeBatch(ctx context.Context, callback ConsumeCallback) error {
	// 拉取消息
	fetches := c.client.PollFetches(ctx)
	if fetches.IsClientClosed() {
		return fmt.Errorf("客户端已关闭")
	}

	if fetches.Err() != nil {
		c.logger.Error("拉取消息失败", clog.Err(fetches.Err()))
		return fetches.Err()
	}

	// 处理每条消息
	fetches.EachRecord(func(record *kgo.Record) {
		c.processRecord(ctx, record, callback)
	})

	return nil
}

// processRecord 处理单条消息
func (c *consumer) processRecord(ctx context.Context, record *kgo.Record, callback ConsumeCallback) {
	// 更新指标
	c.metrics.mu.Lock()
	c.metrics.totalMessages++
	c.metrics.totalBytes += int64(len(record.Value))
	c.metrics.mu.Unlock()

	// 转换为标准消息格式
	msg := &Message{
		Topic:   record.Topic,
		Key:     record.Key,
		Value:   record.Value,
		Headers: convertHeadersFromKgo(record.Headers),
	}

	// 从消息头中提取 trace_id 并注入到上下文中
	traceID := extractTraceIDFromHeaders(record.Headers)
	msgCtx := injectTraceID(ctx, traceID)

	// 处理消息
	err := callback(msgCtx, msg)
	if err != nil {
		c.metrics.mu.Lock()
		c.metrics.failedMessages++
		c.metrics.mu.Unlock()

		c.logger.Error("处理消息失败",
			clog.Err(err),
			clog.String("topic", record.Topic),
			clog.Int32("partition", record.Partition),
			clog.Int64("offset", record.Offset),
			clog.String("key", string(record.Key)),
			clog.Int("value_size", len(record.Value)),
		)
		return
	}

	c.metrics.mu.Lock()
	c.metrics.processedMessage++
	c.metrics.mu.Unlock()

	c.logger.Debug("成功处理消息",
		clog.String("topic", record.Topic),
		clog.Int32("partition", record.Partition),
		clog.Int64("offset", record.Offset),
		clog.String("key", string(record.Key)),
	)
}

// Close 优雅地关闭消费者。
func (c *consumer) Close() error {
	c.logger.Info("关闭 Kafka 消费者", clog.String("group_id", c.groupID))

	// 取消上下文
	c.cancelContext()

	// 等待所有 goroutine 完成
	c.wg.Wait()

	// 关闭客户端
	c.client.Close()

	return nil
}

// GetMetrics 获取消费者性能指标
func (c *consumer) GetMetrics() map[string]interface{} {
	c.metrics.mu.RLock()
	defer c.metrics.mu.RUnlock()

	successRate := float64(0)
	totalProcessed := c.metrics.processedMessage + c.metrics.failedMessages
	if totalProcessed > 0 {
		successRate = float64(c.metrics.processedMessage) / float64(totalProcessed) * 100
	}

	return map[string]interface{}{
		"total_messages":     c.metrics.totalMessages,
		"processed_messages": c.metrics.processedMessage,
		"failed_messages":    c.metrics.failedMessages,
		"total_bytes":        c.metrics.totalBytes,
		"success_rate":       successRate,
	}
}

// Ping 检查消费者健康状态
func (c *consumer) Ping(ctx context.Context) error {
	c.logger.Debug("检查消费者健康状态", clog.String("group_id", c.groupID))

	// 检查连接状态
	if len(c.client.SeedBrokers()) == 0 {
		return fmt.Errorf("没有可用的 seed broker 连接")
	}

	// 检查消费者组状态
	if c.groupID == "" {
		return fmt.Errorf("消费者组ID为空")
	}

	// 尝试 Ping 客户端
	if err := c.client.Ping(ctx); err != nil {
		return fmt.Errorf("客户端 Ping 失败: %w", err)
	}

	return nil
}

// convertHeadersFromKgo 转换 franz-go 消息头格式
func convertHeadersFromKgo(headers []kgo.RecordHeader) map[string][]byte {
	if len(headers) == 0 {
		return nil
	}

	result := make(map[string][]byte)
	for _, header := range headers {
		result[header.Key] = header.Value
	}
	return result
}

// extractTraceIDFromHeaders 从消息头中提取 trace_id
func extractTraceIDFromHeaders(headers []kgo.RecordHeader) string {
	for _, header := range headers {
		if header.Key == "X-Trace-ID" {
			return string(header.Value)
		}
	}
	return ""
}

// injectTraceID 将 trace_id 注入到上下文中
func injectTraceID(ctx context.Context, traceID string) context.Context {
	if traceID == "" {
		return ctx
	}
	return clog.WithTraceID(ctx, traceID)
}