package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"

	"github.com/ceyewan/gochat/im-infra/mq"
)

// ChatMessage 聊天消息结构
type ChatMessage struct {
	MessageID   string    `json:"message_id"`
	FromUser    string    `json:"from_user"`
	ToUser      string    `json:"to_user"`
	Content     string    `json:"content"`
	Timestamp   time.Time `json:"timestamp"`
	MessageType string    `json:"message_type"`
}

func main() {
	// 创建自定义配置
	cfg := mq.Config{
		Brokers:  []string{"localhost:19092"},
		ClientID: "chat-example",
		ProducerConfig: mq.ProducerConfig{
			Compression:       "lz4",
			BatchSize:         16384,
			LingerMs:          5,
			EnableIdempotence: true,
		},
		ConsumerConfig: mq.ConsumerConfig{
			GroupID:         "chat-example-group",
			AutoOffsetReset: "latest",
		},
	}

	// 创建 MQ 实例
	mqInstance, err := mq.New(cfg)
	if err != nil {
		log.Fatalf("创建MQ实例失败: %v", err)
	}
	defer mqInstance.Close()

	// 启动消费者
	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()
		startConsumer(mqInstance)
	}()

	// 启动生产者
	wg.Add(1)
	go func() {
		defer wg.Done()
		startProducer(mqInstance)
	}()

	// 启动监控
	wg.Add(1)
	go func() {
		defer wg.Done()
		startMonitoring(mqInstance)
	}()

	// 等待中断信号
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
	<-sigChan

	log.Println("收到关闭信号，开始优雅关闭...")

	// 这里可以添加优雅关闭逻辑
	// 等待所有协程完成
	wg.Wait()

	log.Println("程序已退出")
}

// startConsumer 启动消费者
func startConsumer(mqInstance mq.MQ) {
	consumer := mqInstance.Consumer()

	callback := func(message *mq.Message, partition mq.TopicPartition, err error) bool {
		if err != nil {
			log.Printf("消费消息错误: %v", err)
			return true // 继续消费
		}

		// 反序列化消息
		var chatMsg ChatMessage
		if err := json.Unmarshal(message.Value, &chatMsg); err != nil {
			log.Printf("反序列化消息失败: %v", err)
			return true
		}

		// 处理聊天消息
		log.Printf("收到聊天消息 [%s -> %s]: %s (时间: %v)",
			chatMsg.FromUser, chatMsg.ToUser, chatMsg.Content, chatMsg.Timestamp)

		// 模拟消息处理时间
		time.Sleep(10 * time.Millisecond)

		return true // 继续消费
	}

	ctx := context.Background()
	err := consumer.Subscribe(ctx, []string{"chat-messages"}, callback)
	if err != nil {
		log.Fatalf("订阅主题失败: %v", err)
	}

	log.Println("消费者已启动，等待消息...")

	// 保持消费者运行
	select {}
}

// startProducer 启动生产者
func startProducer(mqInstance mq.MQ) {
	producer := mqInstance.Producer()

	// 模拟发送聊天消息
	users := []string{"alice", "bob", "charlie", "david", "eve"}
	messageCount := 0

	ticker := time.NewTicker(2 * time.Second)
	defer ticker.Stop()

	for range ticker.C {
		// 随机选择发送者和接收者
		fromUser := users[messageCount%len(users)]
		toUser := users[(messageCount+1)%len(users)]

		// 创建聊天消息
		msg := ChatMessage{
			MessageID:   fmt.Sprintf("msg_%d", messageCount),
			FromUser:    fromUser,
			ToUser:      toUser,
			Content:     fmt.Sprintf("这是第 %d 条消息", messageCount),
			Timestamp:   time.Now(),
			MessageType: "text",
		}

		// 序列化消息
		data, err := json.Marshal(msg)
		if err != nil {
			log.Printf("序列化消息失败: %v", err)
			continue
		}

		// 发送消息
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		err = producer.SendSyncWithKey(ctx, "chat-messages", []byte(msg.ToUser), data)
		cancel()

		if err != nil {
			log.Printf("发送消息失败: %v", err)
		} else {
			log.Printf("发送消息成功: %s -> %s", msg.FromUser, msg.ToUser)
		}

		messageCount++

		// 发送100条消息后停止
		if messageCount >= 100 {
			log.Println("已发送100条消息，生产者停止")
			return
		}
	}
}

// startMonitoring 启动监控
func startMonitoring(mqInstance mq.MQ) {
	ticker := time.NewTicker(10 * time.Second)
	defer ticker.Stop()

	for range ticker.C {
		// 获取生产者指标
		producerMetrics := mqInstance.Producer().GetMetrics()
		log.Printf("生产者指标 - 总消息: %d, 成功: %d, 失败: %d, 平均延迟: %v",
			producerMetrics.TotalMessages,
			producerMetrics.SuccessMessages,
			producerMetrics.FailedMessages,
			producerMetrics.AverageLatency)

		// 获取消费者指标
		consumerMetrics := mqInstance.Consumer().GetMetrics()
		log.Printf("消费者指标 - 总消息: %d, 延迟: %d, 吞吐量: %.2f 消息/秒",
			consumerMetrics.TotalMessages,
			consumerMetrics.Lag,
			consumerMetrics.MessagesPerSecond)

		// 获取连接池统计
		poolStats := mqInstance.ConnectionPool().GetStats()
		log.Printf("连接池统计 - 总连接: %d, 活跃: %d, 空闲: %d",
			poolStats.TotalConnections,
			poolStats.ActiveConnections,
			poolStats.IdleConnections)

		// 健康检查
		ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
		err := mqInstance.Ping(ctx)
		cancel()

		if err != nil {
			log.Printf("健康检查失败: %v", err)
		} else {
			log.Println("健康检查通过")
		}
	}
}

// 批量发送示例
func batchSendExample(mqInstance mq.MQ) {
	producer := mqInstance.Producer()

	// 创建消息批次
	batch := mq.MessageBatch{
		Messages:      make([]*mq.Message, 0, 50),
		MaxBatchSize:  16384,
		MaxBatchCount: 50,
		LingerMs:      5,
	}

	// 添加消息到批次
	for i := 0; i < 50; i++ {
		msg := ChatMessage{
			MessageID:   fmt.Sprintf("batch_msg_%d", i),
			FromUser:    "system",
			ToUser:      fmt.Sprintf("user_%d", i),
			Content:     fmt.Sprintf("批量消息 #%d", i),
			Timestamp:   time.Now(),
			MessageType: "text",
		}

		data, _ := json.Marshal(msg)

		message := &mq.Message{
			Topic: "chat-messages",
			Key:   []byte(msg.ToUser),
			Value: data,
			Headers: map[string][]byte{
				"message_type": []byte(msg.MessageType),
				"from_user":    []byte(msg.FromUser),
			},
		}

		batch.Messages = append(batch.Messages, message)
	}

	// 发送批次
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	start := time.Now()
	results, err := producer.SendBatchSync(ctx, batch)
	elapsed := time.Since(start)

	if err != nil {
		log.Printf("批量发送失败: %v", err)
		return
	}

	// 统计结果
	successCount := 0
	for _, result := range results {
		if result.Error == nil {
			successCount++
		}
	}

	log.Printf("批量发送完成: %d/%d 成功, 耗时: %v, 平均延迟: %v",
		successCount, len(results), elapsed, elapsed/time.Duration(len(results)))
}

// 异步发送示例
func asyncSendExample(mqInstance mq.MQ) {
	producer := mqInstance.Producer()

	var wg sync.WaitGroup
	successCount := int64(0)
	errorCount := int64(0)

	// 异步发送100条消息
	for i := 0; i < 100; i++ {
		wg.Add(1)

		msg := ChatMessage{
			MessageID:   fmt.Sprintf("async_msg_%d", i),
			FromUser:    "async_sender",
			ToUser:      fmt.Sprintf("user_%d", i%10),
			Content:     fmt.Sprintf("异步消息 #%d", i),
			Timestamp:   time.Now(),
			MessageType: "text",
		}

		data, _ := json.Marshal(msg)

		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		producer.SendAsyncWithKey(ctx, "chat-messages", []byte(msg.ToUser), data, func(err error) {
			defer wg.Done()
			defer cancel()

			if err != nil {
				errorCount++
				log.Printf("异步发送失败: %v", err)
			} else {
				successCount++
				log.Printf("异步发送成功: %s", msg.MessageID)
			}
		})
	}

	// 等待所有异步发送完成
	wg.Wait()

	log.Printf("异步发送完成: 成功 %d, 失败 %d", successCount, errorCount)
}
