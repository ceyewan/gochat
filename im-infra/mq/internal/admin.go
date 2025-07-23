package internal

import (
	"context"
	"fmt"

	"github.com/ceyewan/gochat/im-infra/clog"
	"github.com/twmb/franz-go/pkg/kadm"
	"github.com/twmb/franz-go/pkg/kgo"
)

// TopicConfig topic 配置
type TopicConfig struct {
	Name              string            // topic 名称
	Partitions        int32             // 分区数，默认 1
	ReplicationFactor int16             // 副本因子，默认 1
	Configs           map[string]string // topic 配置
}

// AdminClient 管理客户端接口
type AdminClient interface {
	// CreateTopic 创建 topic
	CreateTopic(ctx context.Context, config TopicConfig) error

	// DeleteTopic 删除 topic
	DeleteTopic(ctx context.Context, topicName string) error

	// ListTopics 列出所有 topic
	ListTopics(ctx context.Context) ([]string, error)

	// TopicExists 检查 topic 是否存在
	TopicExists(ctx context.Context, topicName string) (bool, error)

	// Close 关闭管理客户端
	Close() error
}

// adminClient 管理客户端实现
type adminClient struct {
	client *kgo.Client
	admin  *kadm.Client
	logger clog.Logger
}

// NewAdminClient 创建管理客户端
func NewAdminClient(cfg Config) (AdminClient, error) {
	logger := clog.Module("mq.admin")

	// 创建 Kafka 客户端
	opts := []kgo.Opt{
		kgo.SeedBrokers(cfg.Brokers...),
		kgo.ClientID(cfg.ClientID + "-admin"),
	}

	client, err := kgo.NewClient(opts...)
	if err != nil {
		return nil, NewConfigError("创建管理客户端失败", err)
	}

	// 创建管理客户端
	admin := kadm.NewClient(client)

	ac := &adminClient{
		client: client,
		admin:  admin,
		logger: logger,
	}

	logger.Info("管理客户端创建成功")
	return ac, nil
}

// CreateTopic 创建 topic
func (ac *adminClient) CreateTopic(ctx context.Context, config TopicConfig) error {
	// 设置默认值
	if config.Partitions <= 0 {
		config.Partitions = 1
	}
	if config.ReplicationFactor <= 0 {
		config.ReplicationFactor = 1
	}

	// 检查 topic 是否已存在
	exists, err := ac.TopicExists(ctx, config.Name)
	if err != nil {
		return fmt.Errorf("检查topic是否存在失败: %w", err)
	}
	if exists {
		ac.logger.Info("topic已存在，跳过创建", clog.String("topic", config.Name))
		return nil
	}

	ac.logger.Info("开始创建topic",
		clog.String("topic", config.Name),
		clog.Int("partitions", int(config.Partitions)),
		clog.Int("replication_factor", int(config.ReplicationFactor)))

	// 准备配置
	var configs map[string]*string
	if config.Configs != nil {
		configs = make(map[string]*string)
		for k, v := range config.Configs {
			val := v
			configs[k] = &val
		}
	}

	// 执行创建
	results, err := ac.admin.CreateTopics(ctx, config.Partitions, config.ReplicationFactor, configs, config.Name)
	if err != nil {
		ac.logger.Error("创建topic失败",
			clog.String("topic", config.Name),
			clog.Err(err))
		return fmt.Errorf("创建topic失败: %w", err)
	}

	// 检查结果
	for _, result := range results {
		if result.Err != nil {
			ac.logger.Error("创建topic失败",
				clog.String("topic", result.Topic),
				clog.Err(result.Err))
			return fmt.Errorf("创建topic %s 失败: %w", result.Topic, result.Err)
		}
	}

	ac.logger.Info("topic创建成功", clog.String("topic", config.Name))
	return nil
}

// DeleteTopic 删除 topic
func (ac *adminClient) DeleteTopic(ctx context.Context, topicName string) error {
	ac.logger.Info("开始删除topic", clog.String("topic", topicName))

	results, err := ac.admin.DeleteTopics(ctx, topicName)
	if err != nil {
		ac.logger.Error("删除topic失败",
			clog.String("topic", topicName),
			clog.Err(err))
		return fmt.Errorf("删除topic失败: %w", err)
	}

	// 检查结果
	for _, result := range results {
		if result.Err != nil {
			ac.logger.Error("删除topic失败",
				clog.String("topic", result.Topic),
				clog.Err(result.Err))
			return fmt.Errorf("删除topic %s 失败: %w", result.Topic, result.Err)
		}
	}

	ac.logger.Info("topic删除成功", clog.String("topic", topicName))
	return nil
}

// ListTopics 列出所有 topic
func (ac *adminClient) ListTopics(ctx context.Context) ([]string, error) {
	metadata, err := ac.admin.Metadata(ctx)
	if err != nil {
		ac.logger.Error("获取topic列表失败", clog.Err(err))
		return nil, fmt.Errorf("获取topic列表失败: %w", err)
	}

	var topics []string
	for topic := range metadata.Topics {
		topics = append(topics, topic)
	}

	ac.logger.Debug("获取topic列表成功", clog.Int("count", len(topics)))
	return topics, nil
}

// TopicExists 检查 topic 是否存在
func (ac *adminClient) TopicExists(ctx context.Context, topicName string) (bool, error) {
	topics, err := ac.ListTopics(ctx)
	if err != nil {
		return false, err
	}

	for _, topic := range topics {
		if topic == topicName {
			return true, nil
		}
	}

	return false, nil
}

// Close 关闭管理客户端
func (ac *adminClient) Close() error {
	ac.logger.Info("开始关闭管理客户端")

	if ac.admin != nil {
		ac.admin.Close()
	}

	if ac.client != nil {
		ac.client.Close()
	}

	ac.logger.Info("管理客户端已关闭")
	return nil
}
