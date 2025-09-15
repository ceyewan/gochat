package kafka

import (
	"context"

	"github.com/ceyewan/gochat/im-infra/clog"
)

// NewProvider 创建一个新的 Kafka Provider 实例
func NewProvider(ctx context.Context, config *Config, opts ...Option) (Provider, error) {
	if err := validateConfig(config); err != nil {
		return nil, err
	}

	options := &options{
		logger: clog.Namespace("kafka"),
	}

	for _, opt := range opts {
		opt(options)
	}

	// 创建生产者和消费者
	producer, err := newProducerImpl(ctx, config, options)
	if err != nil {
		return nil, err
	}

	return &kafkaProvider{
		config:   config,
		options:  options,
		producer: producer,
		clients:  make(map[string]*consumerImpl),
		logger:   options.logger,
	}, nil
}

// NewProducer 创建一个新的消息生产者实例（向后兼容）
func NewProducer(ctx context.Context, config *Config, opts ...Option) (Producer, error) {
	options := &options{
		logger: clog.Namespace("kafka-producer"),
	}

	for _, opt := range opts {
		opt(options)
	}

	return newProducerImpl(ctx, config, options)
}

// NewConsumer 创建一个新的消息消费者实例（向后兼容）
func NewConsumer(ctx context.Context, config *Config, groupID string, opts ...Option) (Consumer, error) {
	options := &options{
		logger: clog.Namespace("kafka-consumer"),
	}

	for _, opt := range opts {
		opt(options)
	}

	return newConsumerImpl(ctx, config, groupID, options)
}

// kafkaProvider 实现 Provider 接口
type kafkaProvider struct {
	config   *Config
	options  *options
	producer *producerImpl
	clients  map[string]*consumerImpl
	logger   clog.Logger
}

func (p *kafkaProvider) Producer() ProducerOperations {
	return p.producer
}

func (p *kafkaProvider) Consumer(groupID string) ConsumerOperations {
	if groupID == "" {
		return nil
	}

	// 检查是否已存在该 groupID 的消费者
	if cons, exists := p.clients[groupID]; exists {
		return cons
	}

	// 创建新的消费者
	cons, err := newConsumerImpl(context.Background(), p.config, groupID, p.options)
	if err != nil {
		p.logger.Error("创建消费者失败", clog.Err(err), clog.String("group_id", groupID))
		return nil
	}

	p.clients[groupID] = cons
	return cons
}

func (p *kafkaProvider) Admin() AdminOperations {
	return newAdminImpl(p.config, p.logger)
}

func (p *kafkaProvider) Ping(ctx context.Context) error {
	// 使用生产者的 ping 方法
	return p.producer.Ping(ctx)
}

func (p *kafkaProvider) Close() error {
	p.logger.Info("正在关闭 Kafka Provider")

	// 关闭所有消费者
	for groupID, cons := range p.clients {
		p.logger.Info("关闭消费者", clog.String("group_id", groupID))
		cons.Close()
		delete(p.clients, groupID)
	}

	// 关闭生产者
	if p.producer != nil {
		p.producer.Close()
	}

	p.logger.Info("Kafka Provider 已关闭")
	return nil
}

// validateConfig 验证配置
func validateConfig(config *Config) error {
	if config == nil {
		return ErrInvalidConfig("配置不能为空")
	}

	if len(config.Brokers) == 0 {
		return ErrInvalidConfig("Broker 地址列表不能为空")
	}

	if config.ProducerConfig == nil {
		return ErrInvalidConfig("生产者配置不能为空")
	}

	if config.ConsumerConfig == nil {
		return ErrInvalidConfig("消费者配置不能为空")
	}

	// 验证生产者配置
	if config.ProducerConfig.Acks < 0 || config.ProducerConfig.Acks > 1 && config.ProducerConfig.Acks != -1 {
		return ErrInvalidConfig("无效的 Acks 值，必须是 0、1 或 -1")
	}

	if config.ProducerConfig.RetryMax < 0 {
		return ErrInvalidConfig("重试次数不能为负数")
	}

	if config.ProducerConfig.BatchSize <= 0 {
		return ErrInvalidConfig("批处理大小必须大于 0")
	}

	// 验证消费者配置
	validAutoOffsetReset := map[string]bool{
		"earliest": true,
		"latest":   true,
		"none":     true,
	}
	if !validAutoOffsetReset[config.ConsumerConfig.AutoOffsetReset] {
		return ErrInvalidConfig("无效的 AutoOffsetReset 值，必须是 earliest、latest 或 none")
	}

	if config.ConsumerConfig.AutoCommitIntervalMs <= 0 {
		return ErrInvalidConfig("自动提交间隔必须大于 0")
	}

	if config.ConsumerConfig.SessionTimeoutMs <= 0 {
		return ErrInvalidConfig("会话超时必须大于 0")
	}

	return nil
}