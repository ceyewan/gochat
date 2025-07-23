package main

import (
	"context"
	"log"
	"time"

	"github.com/ceyewan/gochat/im-infra/mq"
)

func main() {
	log.Println("🔧 开始 Topic 管理示例...")

	// 创建管理客户端配置
	cfg := mq.Config{
		Brokers:  []string{"localhost:19092"},
		ClientID: "topic-admin",
	}

	// 创建管理客户端
	admin, err := mq.NewAdminClient(cfg)
	if err != nil {
		log.Fatalf("❌ 创建管理客户端失败: %v", err)
	}
	defer admin.Close()

	log.Println("✅ 管理客户端创建成功")

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// 1. 列出现有的 topics
	log.Println("\n📋 列出现有 topics...")
	topics, err := admin.ListTopics(ctx)
	if err != nil {
		log.Printf("❌ 获取 topic 列表失败: %v", err)
	} else {
		log.Printf("✅ 现有 topics (%d 个):", len(topics))
		for _, topic := range topics {
			log.Printf("  - %s", topic)
		}
	}

	// 2. 创建测试 topics
	testTopics := []mq.TopicConfig{
		{
			Name:              "test-connection",
			Partitions:        1,
			ReplicationFactor: 1,
		},
		{
			Name:              "chat-messages",
			Partitions:        3,
			ReplicationFactor: 1,
			Configs: map[string]string{
				"retention.ms": "604800000", // 7 天
			},
		},
		{
			Name:              "user-events",
			Partitions:        2,
			ReplicationFactor: 1,
		},
	}

	log.Println("\n🏗️ 创建测试 topics...")
	for _, topicConfig := range testTopics {
		log.Printf("创建 topic: %s (分区: %d, 副本: %d)", 
			topicConfig.Name, topicConfig.Partitions, topicConfig.ReplicationFactor)
		
		err := admin.CreateTopic(ctx, topicConfig)
		if err != nil {
			log.Printf("❌ 创建 topic %s 失败: %v", topicConfig.Name, err)
		} else {
			log.Printf("✅ topic %s 创建成功", topicConfig.Name)
		}
	}

	// 3. 再次列出 topics 确认创建成功
	log.Println("\n📋 创建后的 topic 列表...")
	topics, err = admin.ListTopics(ctx)
	if err != nil {
		log.Printf("❌ 获取 topic 列表失败: %v", err)
	} else {
		log.Printf("✅ 当前 topics (%d 个):", len(topics))
		for _, topic := range topics {
			log.Printf("  - %s", topic)
		}
	}

	// 4. 检查特定 topic 是否存在
	log.Println("\n🔍 检查 topic 是否存在...")
	checkTopics := []string{"chat-messages", "non-existent-topic"}
	for _, topicName := range checkTopics {
		exists, err := admin.TopicExists(ctx, topicName)
		if err != nil {
			log.Printf("❌ 检查 topic %s 失败: %v", topicName, err)
		} else if exists {
			log.Printf("✅ topic %s 存在", topicName)
		} else {
			log.Printf("❌ topic %s 不存在", topicName)
		}
	}

	log.Println("\n🎉 Topic 管理示例完成!")
}
