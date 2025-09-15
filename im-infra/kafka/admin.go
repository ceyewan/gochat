package kafka

import (
	"context"
	"fmt"
	"os/exec"
	"strings"
	"time"

	"github.com/ceyewan/gochat/im-infra/clog"
)

// AdminClient Kafka 管理客户端，用于创建和管理 topics
type AdminClient struct {
	brokers []string
	logger  clog.Logger
}

// NewAdminClient 创建一个新的 Kafka 管理客户端
func NewAdminClient(ctx context.Context, config *Config, opts ...Option) (*AdminClient, error) {
	options := &options{
		logger: clog.Namespace("kafka-admin"),
	}

	for _, opt := range opts {
		opt(options)
	}

	admin := &AdminClient{
		brokers: config.Brokers,
		logger:  options.logger,
	}

	admin.logger.Info("Kafka Admin 客户端初始化成功",
		clog.Strings("brokers", config.Brokers),
	)

	return admin, nil
}

// CreateTopic 创建单个 topic
func (a *AdminClient) CreateTopic(ctx context.Context, topic string, partitions int32, replicationFactor int16) error {
	a.logger.Info("创建 Topic",
		clog.String("topic", topic),
		clog.Int32("partitions", partitions),
		clog.Int16("replication_factor", replicationFactor),
	)

	// 检查 topic 是否已存在
	if a.TopicExists(ctx, topic) {
		a.logger.Info("Topic 已存在，跳过创建", clog.String("topic", topic))
		return nil
	}

	// 执行 kafka-topics.sh 命令
	cmd := exec.CommandContext(ctx, "kafka-topics.sh",
		"--bootstrap-server", strings.Join(a.brokers, ","),
		"--create",
		"--topic", topic,
		"--partitions", fmt.Sprintf("%d", partitions),
		"--replication-factor", fmt.Sprintf("%d", replicationFactor),
		"--config", "retention.ms=86400000",
		"--config", "segment.ms=3600000",
	)

	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("创建 topic 失败: %w, 输出: %s", err, string(output))
	}

	a.logger.Info("Topic 创建成功", clog.String("topic", topic))
	return nil
}

// CreateTopics 批量创建 topics
func (a *AdminClient) CreateTopics(ctx context.Context, topics []TopicConfig) error {
	a.logger.Info("批量创建 Topics", clog.Int("count", len(topics)))

	var errors []string
	successCount := 0

	for _, topic := range topics {
		if err := a.CreateTopic(ctx, topic.Name, topic.Partitions, topic.ReplicationFactor); err != nil {
			a.logger.Error("Topic 创建失败",
				clog.String("topic", topic.Name),
				clog.Err(err),
			)
			errors = append(errors, fmt.Sprintf("%s: %v", topic.Name, err))
		} else {
			successCount++
		}
	}

	a.logger.Info("Topics 创建完成",
		clog.Int("success_count", successCount),
		clog.Int("failure_count", len(errors)),
	)

	if len(errors) > 0 {
		return fmt.Errorf("部分 topics 创建失败: %s", strings.Join(errors, "; "))
	}

	return nil
}

// DeleteTopic 删除 topic
func (a *AdminClient) DeleteTopic(ctx context.Context, topic string) error {
	a.logger.Info("删除 Topic", clog.String("topic", topic))

	cmd := exec.CommandContext(ctx, "kafka-topics.sh",
		"--bootstrap-server", strings.Join(a.brokers, ","),
		"--delete",
		"--topic", topic,
	)

	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("删除 topic 失败: %w, 输出: %s", err, string(output))
	}

	a.logger.Info("Topic 删除成功", clog.String("topic", topic))
	return nil
}

// ListTopics 列出所有 topics
func (a *AdminClient) ListTopics(ctx context.Context) ([]string, error) {
	a.logger.Debug("获取 Topic 列表")

	cmd := exec.CommandContext(ctx, "kafka-topics.sh",
		"--bootstrap-server", strings.Join(a.brokers, ","),
		"--list",
	)

	output, err := cmd.CombinedOutput()
	if err != nil {
		return nil, fmt.Errorf("获取 topic 列表失败: %w, 输出: %s", err, string(output))
	}

	topics := strings.Split(strings.TrimSpace(string(output)), "\n")

	// 过滤掉空行
	var result []string
	for _, topic := range topics {
		if topic != "" {
			result = append(result, topic)
		}
	}

	a.logger.Debug("获取到 Topics", clog.Int("count", len(result)))

	return result, nil
}

// TopicExists 检查 topic 是否存在
func (a *AdminClient) TopicExists(ctx context.Context, topic string) bool {
	topics, err := a.ListTopics(ctx)
	if err != nil {
		a.logger.Error("检查 topic 存在性失败", clog.String("topic", topic), clog.Err(err))
		return false
	}

	for _, t := range topics {
		if t == topic {
			return true
		}
	}

	return false
}

// Close 关闭管理客户端
func (a *AdminClient) Close() error {
	a.logger.Info("关闭 Kafka Admin 客户端")
	return nil
}

// TopicConfig Topic 配置
type TopicConfig struct {
	Name             string
	Partitions       int32
	ReplicationFactor int16
}

// CreateExampleTopics 创建示例 topics
func CreateExampleTopics(ctx context.Context, config *Config) error {
	admin, err := NewAdminClient(ctx, config)
	if err != nil {
		return fmt.Errorf("创建 admin 客户端失败: %w", err)
	}
	defer admin.Close()

	// 定义示例 topics
	topics := []TopicConfig{
		{
			Name:             "example.user.events",
			Partitions:       3,
			ReplicationFactor: 1,
		},
		{
			Name:             "example.test-topic",
			Partitions:       1,
			ReplicationFactor: 1,
		},
		{
			Name:             "example.performance",
			Partitions:       6,
			ReplicationFactor: 1,
		},
		{
			Name:             "example.dead-letter",
			Partitions:       1,
			ReplicationFactor: 1,
		},
	}

	// 创建 topics
	if err := admin.CreateTopics(ctx, topics); err != nil {
		return fmt.Errorf("创建示例 topics 失败: %w", err)
	}

	// 等待 topics 创建完成
	time.Sleep(2 * time.Second)

	// 验证创建结果
	for _, topic := range topics {
		if !admin.TopicExists(ctx, topic.Name) {
			return fmt.Errorf("topic '%s' 创建失败", topic.Name)
		}
	}

	return nil
}