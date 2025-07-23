package main

import (
	"context"
	"log"
	"time"

	"github.com/ceyewan/gochat/im-infra/mq"
)

func main() {
	log.Println("🔍 开始 Kafka 连接测试...")

	// 使用最简单的配置
	cfg := mq.Config{
		Brokers:  []string{"localhost:19092"},
		ClientID: "connection-test",
		ConsumerConfig: mq.ConsumerConfig{
			GroupID: "connection-test-group",
		},
	}

	log.Printf("📡 尝试连接到 Kafka: %v", cfg.Brokers)

	// 创建 MQ 实例
	mqInstance, err := mq.New(cfg)
	if err != nil {
		log.Fatalf("❌ 创建MQ实例失败: %v", err)
	}
	defer mqInstance.Close()

	log.Println("✅ MQ 实例创建成功")

	// 测试连接
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	log.Println("🏥 执行健康检查...")
	err = mqInstance.Ping(ctx)
	if err != nil {
		log.Printf("❌ 健康检查失败: %v", err)
	} else {
		log.Println("✅ 健康检查通过")
	}

	// 尝试发送一条简单消息到一个简单的 topic
	producer := mqInstance.Producer()

	log.Println("📤 尝试发送测试消息...")

	// 使用一个简单的 topic 名称
	testTopic := "test-connection"
	testMessage := []byte("Hello Kafka!")

	sendCtx, sendCancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer sendCancel()

	err = producer.SendSync(sendCtx, testTopic, testMessage)
	if err != nil {
		log.Printf("❌ 发送消息失败: %v", err)

		// 分析错误类型
		if err.Error() == "UNKNOWN_TOPIC_OR_PARTITION: This server does not host this topic-partition." {
			log.Println("💡 这个错误表示 topic 不存在")
			log.Println("💡 可能的原因:")
			log.Println("   1. Kafka 配置了 auto.create.topics.enable=false")
			log.Println("   2. 需要手动创建 topic")
			log.Println("   3. 权限问题")
		}
	} else {
		log.Println("✅ 消息发送成功!")
	}

	// 获取生产者指标
	metrics := producer.GetMetrics()
	log.Printf("📊 生产者指标: 总消息=%d, 成功=%d, 失败=%d",
		metrics.TotalMessages, metrics.SuccessMessages, metrics.FailedMessages)

	log.Println("🏁 测试完成")
}
