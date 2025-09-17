package main

import (
	"context"
	"fmt"
	"log"
	"os"

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

	// 3. 创建 Provider 和获取 Admin 接口
	logger := clog.Namespace("topic-admin")
	provider, err := kafka.NewProvider(ctx, config, kafka.WithLogger(logger))
	if err != nil {
		log.Fatal("创建 Provider 失败:", err)
	}
	defer provider.Close()

	admin := provider.Admin()

	// 4. 定义要创建的 Topics
	testTopics := map[string]struct {
		partitions        int32
		replicationFactor int16
		config            map[string]string
	}{
		"example.user.events": {
			partitions:        3,
			replicationFactor: 1,
			config: map[string]string{
				"retention.ms":     "86400000", // 24 小时
				"cleanup.policy":   "delete",
				"compression.type": "lz4",
			},
		},
		"example.test-topic": {
			partitions:        1,
			replicationFactor: 1,
			config: map[string]string{
				"retention.ms":      "3600000", // 1 小时
				"cleanup.policy":    "delete",
				"max.message.bytes": "1048576", // 1MB
			},
		},
		"example.performance": {
			partitions:        6,
			replicationFactor: 1,
			config: map[string]string{
				"retention.ms":     "1800000", // 30 分钟
				"cleanup.policy":   "delete",
				"compression.type": "zstd",
			},
		},
	}

	// 5. 批量创建 Topics
	fmt.Println("=== 批量创建 Topics ===")
	for topicName, topicConfig := range testTopics {
		err = admin.CreateTopic(ctx, topicName, topicConfig.partitions, topicConfig.replicationFactor, topicConfig.config)
		if err != nil {
			logger.Error("创建 Topic 失败",
				clog.String("topic", topicName),
				clog.Err(err),
			)
			fmt.Printf("创建 Topic '%s' 失败: %v\n", topicName, err)
		} else {
			fmt.Printf("✅ Topic '%s' 创建成功!\n", topicName)
		}
	}

	// 6. 列出所有 Topics
	fmt.Println("\n=== 列出 Topics ===")
	topics, err := admin.ListTopics(ctx)
	if err != nil {
		logger.Error("列出 Topics 失败", clog.Err(err))
	} else {
		fmt.Printf("📋 找到 %d 个 Topics:\n", len(topics))
		for topicName, detail := range topics {
			fmt.Printf("  - %s (分区数: %d, 副本数: %d)\n",
				topicName,
				detail.NumPartitions,
				detail.ReplicationFactor,
			)
		}
	}

	// 7. 检查特定 Topic 是否存在
	fmt.Println("\n=== 检查 Topic 存在性 ===")
	testTopicName := "example.user.events"
	if topics, err := admin.ListTopics(ctx); err == nil {
		exists := false
		if _, found := topics[testTopicName]; found {
			exists = true
		}
		fmt.Printf("🔍 Topic '%s' 存在: %t\n", testTopicName, exists)

		// 8. 获取 Topic 详细信息
		if exists {
			fmt.Println("\n=== Topic 详细信息 ===")
			metadata, err := admin.GetTopicMetadata(ctx, testTopicName)
			if err != nil {
				logger.Error("获取 Topic 详细信息失败", clog.String("topic", testTopicName), clog.Err(err))
			} else {
				fmt.Printf("📄 Topic '%s' 详细信息:\n", testTopicName)
				fmt.Printf("  - 分区数: %d\n", metadata.NumPartitions)
				fmt.Printf("  - 副本因子: %d\n", metadata.ReplicationFactor)
				fmt.Printf("  - 配置: %v\n", metadata.Config)
			}
		}
	} else {
		logger.Error("检查 Topic 存在性失败", clog.String("topic", testTopicName), clog.Err(err))
	}

	// 9. 清理测试 Topics (可选)
	fmt.Println("\n=== 清理测试 Topics ===")
	cleanup := os.Getenv("CLEANUP_TOPICS")
	if cleanup == "true" || cleanup == "1" {
		for topicName := range testTopics {
			err = admin.DeleteTopic(ctx, topicName)
			if err != nil {
				logger.Error("删除 Topic 失败",
					clog.String("topic", topicName),
					clog.Err(err),
				)
				fmt.Printf("删除 Topic '%s' 失败: %v\n", topicName, err)
			} else {
				fmt.Printf("🧹 Topic '%s' 删除成功!\n", topicName)
			}
		}
	} else {
		fmt.Println("💡 跳过清理 Topics (设置 CLEANUP_TOPICS=true 来启用清理)")
	}

	fmt.Println("\n🎉 Topic 管理示例完成!")
}
