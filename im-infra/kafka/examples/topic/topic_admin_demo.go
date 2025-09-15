package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/ceyewan/gochat/im-infra/clog"
	"github.com/ceyewan/gochat/im-infra/kafka"
)

func main() {
	ctx := context.Background()

	// 1. 初始化 clog
	clog.Init(ctx, clog.GetDefaultConfig("development"))

	// 2. 获取 Kafka 配置
	config := kafka.GetDefaultConfig("development")
	config.Brokers = []string{"localhost:9092", "localhost:19092", "localhost:29092"}

	// 3. 创建生产者 (用于获取 kgo.Client)
	producer, err := kafka.NewProducer(ctx, config, kafka.WithNamespace("topic-admin"))
	if err != nil {
		log.Fatal("创建生产者失败:", err)
	}
	defer producer.Close()

	// 4. 创建 Topic 管理器
	logger := clog.Namespace("topic-admin")
	topicManager := kafka.NewTopicManager(producer.GetClient(), logger)
	defer topicManager.Close()

	// 5. 定义要创建的 Topics
	testTopics := map[string]*kafka.TopicConfig{
		"example.user.events": {
			Partitions:        3,
			ReplicationFactor: 1,
			Configs: map[string]*string{
				"retention.ms":     kafka.StringPtr("86400000"), // 24 小时
				"cleanup.policy":   kafka.StringPtr("delete"),
				"compression.type": kafka.StringPtr("lz4"),
			},
			Timeout: 30 * time.Second,
		},
		"example.test-topic": {
			Partitions:        1,
			ReplicationFactor: 1,
			Configs: map[string]*string{
				"retention.ms":      kafka.StringPtr("3600000"), // 1 小时
				"cleanup.policy":    kafka.StringPtr("delete"),
				"max.message.bytes": kafka.StringPtr("1048576"), // 1MB
			},
			Timeout: 30 * time.Second,
		},
		"example.performance": {
			Partitions:        6,
			ReplicationFactor: 1,
			Configs: map[string]*string{
				"retention.ms":     kafka.StringPtr("1800000"), // 30 分钟
				"cleanup.policy":   kafka.StringPtr("delete"),
				"compression.type": kafka.StringPtr("zstd"),
			},
			Timeout: 30 * time.Second,
		},
	}

	// 6. 批量创建 Topics
	fmt.Println("=== 批量创建 Topics ===")
	err = topicManager.CreateTopics(ctx, testTopics)
	if err != nil {
		logger.Error("批量创建 Topics 失败", clog.Err(err))
		fmt.Printf("错误: %v\n", err)
	} else {
		fmt.Println("✅ 所有 Topics 创建成功!")
	}

	// 7. 列出所有 Topics
	fmt.Println("\n=== 列出 Topics ===")
	details, err := topicManager.ListTopics(ctx)
	if err != nil {
		logger.Error("列出 Topics 失败", clog.Err(err))
	} else {
		fmt.Printf("📋 找到 %d 个 Topics:\n", len(details))
		for topicName, detail := range details {
			numPartitions := len(detail.Partitions)
			replicationFactor := 1 // 默认值，如果无法从分区详情中获取
			if len(detail.Partitions) > 0 {
				for _, partitionDetail := range detail.Partitions {
					replicationFactor = len(partitionDetail.Replicas)
					break
				}
			}
			fmt.Printf("  - %s (分区数: %d, 副本数: %d)\n",
				topicName,
				numPartitions,
				replicationFactor,
			)
		}
	}

	// 8. 检查特定 Topic 是否存在
	fmt.Println("\n=== 检查 Topic 存在性 ===")
	testTopicName := "example.user.events"
	exists, err := topicManager.TopicExists(ctx, testTopicName)
	if err != nil {
		logger.Error("检查 Topic 存在性失败", clog.String("topic", testTopicName), clog.Err(err))
	} else {
		fmt.Printf("🔍 Topic '%s' 存在: %t\n", testTopicName, exists)
	}

	// 9. 获取 Topic 详细信息
	if exists {
		fmt.Println("\n=== Topic 详细信息 ===")
		detail, err := topicManager.GetTopicDetail(ctx, testTopicName)
		if err != nil {
			logger.Error("获取 Topic 详细信息失败", clog.String("topic", testTopicName), clog.Err(err))
		} else {
			fmt.Printf("📄 Topic '%s' 详细信息:\n", testTopicName)
			fmt.Printf("  - Topic ID: %s\n", detail.ID)
			fmt.Printf("  - 分区数: %d\n", len(detail.Partitions))
			replicationFactor := 1
			if len(detail.Partitions) > 0 {
				for _, partitionDetail := range detail.Partitions {
					replicationFactor = len(partitionDetail.Replicas)
					break
				}
			}
			fmt.Printf("  - 副本因子: %d\n", replicationFactor)
			fmt.Printf("  - IsInternal: %t\n", detail.IsInternal)
		}
	}

	// 10. 清理测试 Topics (可选)
	fmt.Println("\n=== 清理测试 Topics ===")
	cleanup := os.Getenv("CLEANUP_TOPICS")
	if cleanup == "true" || cleanup == "1" {
		var topicsToDelete []string
		for topicName := range testTopics {
			topicsToDelete = append(topicsToDelete, topicName)
		}

		err = topicManager.DeleteTopics(ctx, topicsToDelete...)
		if err != nil {
			logger.Error("删除测试 Topics 失败", clog.Err(err))
			fmt.Printf("删除失败: %v\n", err)
		} else {
			fmt.Println("🧹 所有测试 Topics 删除成功!")
		}
	} else {
		fmt.Println("💡 跳过清理 Topics (设置 CLEANUP_TOPICS=true 来启用清理)")
	}

	fmt.Println("\n🎉 Topic 管理示例完成!")
}
