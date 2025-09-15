package kafka

import (
	"context"
	"fmt"
	"time"

	"github.com/ceyewan/gochat/im-infra/clog"
	"github.com/twmb/franz-go/pkg/kadm"
	"github.com/twmb/franz-go/pkg/kgo"
)

// StringPtr 返回字符串指针的辅助函数
func StringPtr(s string) *string {
	return &s
}

// TopicConfig Topic配置选项
type TopicConfig struct {
	// 分区数量，-1 表示使用 broker 默认值 (Kafka 2.4+)
	Partitions int32

	// 副本因子，-1 表示使用 broker 默认值 (Kafka 2.4+)
	ReplicationFactor int16

	// Topic 级别配置
	Configs map[string]*string

	// 创建超时时间
	Timeout time.Duration
}

// DefaultTopicConfig 返回默认的 Topic 配置
func DefaultTopicConfig() *TopicConfig {
	return &TopicConfig{
		Partitions:         3,                    // 默认 3 个分区
		ReplicationFactor: 1,                    // 默认 1 个副本
		Configs:           make(map[string]*string),
		Timeout:           30 * time.Second,     // 默认 30 秒超时
	}
}

// adminImpl 实现AdminOperations接口
type adminImpl struct {
	config *Config
	client *kgo.Client
	tm     *TopicManager
	logger clog.Logger
}

// TopicManager Topic管理器
type TopicManager struct {
	client     *kgo.Client
	kadmClient *kadm.Client
	logger     clog.Logger
}

// NewTopicManager 创建一个新的 Topic 管理器
func NewTopicManager(client *kgo.Client, logger clog.Logger) *TopicManager {
	kadmClient := kadm.NewClient(client)

	return &TopicManager{
		client:     client,
		kadmClient: kadmClient,
		logger:     logger,
	}
}

// CreateTopic 创建单个 Topic
func (tm *TopicManager) CreateTopic(ctx context.Context, topicName string, config *TopicConfig) error {
	if config == nil {
		config = DefaultTopicConfig()
	}

	// 设置超时上下文
	if config.Timeout > 0 {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, config.Timeout)
		defer cancel()
	}

	tm.logger.Info("开始创建 Topic",
		clog.String("topic", topicName),
		clog.Int32("partitions", config.Partitions),
		clog.Int16("replication_factor", config.ReplicationFactor),
		clog.Int("config_count", len(config.Configs)),
	)

	// 创建 Topic
	responses, err := tm.kadmClient.CreateTopics(
		ctx,
		config.Partitions,
		config.ReplicationFactor,
		config.Configs,
		topicName,
	)

	if err != nil {
		tm.logger.Error("创建 Topic 请求失败",
			clog.String("topic", topicName),
			clog.Err(err),
		)
		return fmt.Errorf("创建 Topic 请求失败: %w", err)
	}

	// 检查创建结果
	response, exists := responses[topicName]
	if !exists {
		tm.logger.Error("Topic 创建响应不存在", clog.String("topic", topicName))
		return fmt.Errorf("Topic 创建响应不存在")
	}

	if response.Err != nil {
		// 检查是否是 Topic 已存在的错误
		if response.Err.Error() == "TOPIC_ALREADY_EXISTS: Topic with this name already exists." {
			tm.logger.Info("Topic 已存在，跳过创建",
				clog.String("topic", topicName),
			)
			return nil
		}

		tm.logger.Error("Topic 创建失败",
			clog.String("topic", topicName),
			clog.Err(response.Err),
		)
		return fmt.Errorf("Topic 创建失败: %w", response.Err)
	}

	tm.logger.Info("Topic 创建成功",
		clog.String("topic", topicName),
		clog.String("topic_id", response.ID.String()),
	)

	return nil
}

// CreateTopics 批量创建 Topics
func (tm *TopicManager) CreateTopics(ctx context.Context, topicConfigs map[string]*TopicConfig) error {
	if len(topicConfigs) == 0 {
		return fmt.Errorf("topic 配置不能为空")
	}

	// 使用第一个配置的默认值
	var defaultConfig *TopicConfig
	for _, config := range topicConfigs {
		defaultConfig = config
		break
	}

	if defaultConfig == nil {
		defaultConfig = DefaultTopicConfig()
	}

	// 设置超时上下文
	if defaultConfig.Timeout > 0 {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, defaultConfig.Timeout)
		defer cancel()
	}

	topicNames := make([]string, 0, len(topicConfigs))
	for name := range topicConfigs {
		topicNames = append(topicNames, name)
	}

	tm.logger.Info("开始批量创建 Topics",
		clog.Strings("topics", topicNames),
		clog.Int("count", len(topicNames)),
	)

	// 使用相同的分区和副本数创建所有 topics
	// 注意：如果需要不同的配置，需要分别调用 CreateTopic
	responses, err := tm.kadmClient.CreateTopics(
		ctx,
		defaultConfig.Partitions,
		defaultConfig.ReplicationFactor,
		defaultConfig.Configs,
		topicNames...,
	)

	if err != nil {
		tm.logger.Error("批量创建 Topics 请求失败", clog.Err(err))
		return fmt.Errorf("批量创建 Topics 请求失败: %w", err)
	}

	// 检查每个 Topic 的创建结果
	var failedTopics []string
	var existingTopics []string
	for topicName, response := range responses {
		if response.Err != nil {
			// 检查是否是 Topic 已存在的错误
			if response.Err.Error() == "TOPIC_ALREADY_EXISTS: Topic with this name already exists." {
				tm.logger.Info("Topic 已存在，跳过创建",
					clog.String("topic", topicName),
				)
				existingTopics = append(existingTopics, topicName)
			} else {
				tm.logger.Error("Topic 创建失败",
					clog.String("topic", topicName),
					clog.Err(response.Err),
				)
				failedTopics = append(failedTopics, topicName)
			}
		} else {
			tm.logger.Info("Topic 创建成功",
				clog.String("topic", topicName),
				clog.String("topic_id", response.ID.String()),
			)
		}
	}

	// 输出总结信息
	if len(existingTopics) > 0 {
		tm.logger.Info("部分 Topics 已存在，已跳过创建",
			clog.Strings("existing_topics", existingTopics),
			clog.Int("count", len(existingTopics)),
		)
	}

	if len(failedTopics) > 0 {
		return fmt.Errorf("以下 Topics 创建失败: %v", failedTopics)
	}

	return nil
}

// DeleteTopic 删除 Topic
func (tm *TopicManager) DeleteTopic(ctx context.Context, topicName string) error {
	tm.logger.Info("开始删除 Topic", clog.String("topic", topicName))

	response, err := tm.kadmClient.DeleteTopic(ctx, topicName)
	if err != nil {
		tm.logger.Error("删除 Topic 请求失败",
			clog.String("topic", topicName),
			clog.Err(err),
		)
		return fmt.Errorf("删除 Topic 请求失败: %w", err)
	}

	if response.Err != nil {
		tm.logger.Error("Topic 删除失败",
			clog.String("topic", topicName),
			clog.Err(response.Err),
		)
		return fmt.Errorf("Topic 删除失败: %w", response.Err)
	}

	tm.logger.Info("Topic 删除成功", clog.String("topic", topicName))
	return nil
}

// DeleteTopics 批量删除 Topics
func (tm *TopicManager) DeleteTopics(ctx context.Context, topicNames ...string) error {
	if len(topicNames) == 0 {
		return fmt.Errorf("topic 名列表不能为空")
	}

	tm.logger.Info("开始批量删除 Topics", clog.Strings("topics", topicNames))

	responses, err := tm.kadmClient.DeleteTopics(ctx, topicNames...)
	if err != nil {
		tm.logger.Error("批量删除 Topics 请求失败", clog.Err(err))
		return fmt.Errorf("批量删除 Topics 请求失败: %w", err)
	}

	// 检查删除结果
	var failedTopics []string
	for topicName, response := range responses {
		if response.Err != nil {
			tm.logger.Error("Topic 删除失败",
				clog.String("topic", topicName),
				clog.Err(response.Err),
			)
			failedTopics = append(failedTopics, topicName)
		} else {
			tm.logger.Info("Topic 删除成功", clog.String("topic", topicName))
		}
	}

	if len(failedTopics) > 0 {
		return fmt.Errorf("以下 Topics 删除失败: %v", failedTopics)
	}

	return nil
}

// ListTopics 列出 Topics
func (tm *TopicManager) ListTopics(ctx context.Context, topics ...string) (map[string]kadm.TopicDetail, error) {
	tm.logger.Info("开始列出 Topics", clog.Strings("topics", topics))

	details, err := tm.kadmClient.ListTopics(ctx, topics...)
	if err != nil {
		tm.logger.Error("列出 Topics 失败", clog.Err(err))
		return nil, fmt.Errorf("列出 Topics 失败: %w", err)
	}

	tm.logger.Info("成功列出 Topics", clog.Int("count", len(details)))
	return details, nil
}

// TopicExists 检查 Topic 是否存在
func (tm *TopicManager) TopicExists(ctx context.Context, topicName string) (bool, error) {
	details, err := tm.ListTopics(ctx, topicName)
	if err != nil {
		return false, err
	}

	_, exists := details[topicName]
	return exists, nil
}

// GetTopicDetail 获取 Topic 详细信息
func (tm *TopicManager) GetTopicDetail(ctx context.Context, topicName string) (*kadm.TopicDetail, error) {
	details, err := tm.ListTopics(ctx, topicName)
	if err != nil {
		return nil, err
	}

	detail, exists := details[topicName]
	if !exists {
		return nil, fmt.Errorf("Topic 不存在: %s", topicName)
	}

	return &detail, nil
}

// Close 关闭 Topic 管理器
func (tm *TopicManager) Close() {
	// kadmClient 不需要单独关闭，它使用的是 kgo.Client
	tm.logger.Info("Topic 管理器已关闭")
}

// newAdminImpl 创建一个新的admin实例
func newAdminImpl(config *Config, logger clog.Logger) *adminImpl {
	// 创建一个临时的kafka客户端用于admin操作
	client, err := kgo.NewClient(
		kgo.SeedBrokers(config.Brokers...),
		kgo.DefaultProduceTopic("admin-temp"),
	)
	if err != nil {
		return nil
	}

	return &adminImpl{
		config: config,
		client: client,
		tm:     NewTopicManager(client, logger),
		logger: logger,
	}
}

// CreateTopic 创建主题
func (a *adminImpl) CreateTopic(ctx context.Context, topic string, partitions int32, replicationFactor int16, config map[string]string) error {
	topicConfig := &TopicConfig{
		Partitions:        partitions,
		ReplicationFactor: replicationFactor,
		Configs:           make(map[string]*string),
	}

	// 转换config格式
	for k, v := range config {
		topicConfig.Configs[k] = StringPtr(v)
	}

	return a.tm.CreateTopic(ctx, topic, topicConfig)
}

// DeleteTopic 删除主题
func (a *adminImpl) DeleteTopic(ctx context.Context, topic string) error {
	return a.tm.DeleteTopic(ctx, topic)
}

// ListTopics 列出所有主题
func (a *adminImpl) ListTopics(ctx context.Context) (map[string]TopicDetail, error) {
	topics, err := a.tm.ListTopics(ctx)
	if err != nil {
		return nil, err
	}

	result := make(map[string]TopicDetail)
	for name, detail := range topics {
		numPartitions := int32(len(detail.Partitions))
		replicationFactor := int16(1) // default value

		// 获取副本因子
		if len(detail.Partitions) > 0 {
			for _, partitionDetail := range detail.Partitions {
				replicationFactor = int16(len(partitionDetail.Replicas))
				break
			}
		}

		result[name] = TopicDetail{
			NumPartitions:     numPartitions,
			ReplicationFactor: replicationFactor,
			Config:            make(map[string]string), // TODO: 从detail中提取配置
		}
	}

	return result, nil
}

// GetTopicMetadata 获取主题元数据
func (a *adminImpl) GetTopicMetadata(ctx context.Context, topic string) (*TopicDetail, error) {
	detail, err := a.tm.GetTopicDetail(ctx, topic)
	if err != nil {
		return nil, err
	}

	numPartitions := int32(len(detail.Partitions))
	replicationFactor := int16(1) // default value

	// 获取副本因子
	if len(detail.Partitions) > 0 {
		for _, partitionDetail := range detail.Partitions {
			replicationFactor = int16(len(partitionDetail.Replicas))
			break
		}
	}

	return &TopicDetail{
		NumPartitions:     numPartitions,
		ReplicationFactor: replicationFactor,
		Config:            make(map[string]string), // TODO: 从detail中提取配置
	}, nil
}

// CreatePartitions 增加主题分区数
func (a *adminImpl) CreatePartitions(ctx context.Context, topic string, newPartitionCount int32) error {
	// TODO: 实现CreatePartitions功能
	// 这需要使用kadm.CreatePartitions或其他相关API
	return ErrAdmin("CreatePartitions not implemented", nil)
}