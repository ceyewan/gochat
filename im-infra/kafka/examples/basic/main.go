package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/ceyewan/gochat/im-infra/clog"
	"github.com/ceyewan/gochat/im-infra/kafka"
)

// UserEvent 用户事件
type UserEvent struct {
	UserID    string    `json:"user_id"`
	EventType string    `json:"event_type"`
	Timestamp time.Time `json:"timestamp"`
	Data      any       `json:"data,omitempty"`
}

func main() {
	ctx := context.Background()

	// 1. 初始化 clog
	clog.Init(ctx, clog.GetDefaultConfig("development"))

	// 2. 获取 Kafka 配置
	config := kafka.GetDefaultConfig("development")
	config.Brokers = []string{"localhost:9092", "localhost:19092", "localhost:29092"}

	// 3. 创建生产者
	producer, err := kafka.NewProducer(ctx, config, kafka.WithNamespace("example-producer"))
	if err != nil {
		log.Fatal("创建生产者失败:", err)
	}
	defer producer.Close()

	// 4. 创建消费者
	consumer, err := kafka.NewConsumer(ctx, config, "example-consumer-group", kafka.WithNamespace("example-consumer"))
	if err != nil {
		log.Fatal("创建消费者失败:", err)
	}
	defer consumer.Close()

	// 5. 启动消费者
	go startConsumer(ctx, consumer)

	// 6. 发送一些示例消息
	sendExampleMessages(ctx, producer)

	// 7. 等待退出信号
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
	<-sigChan

	fmt.Println("正在关闭程序...")
}

// startConsumer 启动消费者
func startConsumer(ctx context.Context, consumer kafka.Consumer) {
	topics := []string{"example.user.events"}

	handler := func(ctx context.Context, msg *kafka.Message) error {
		// 从 context 中获取 trace ID
		traceID := ctx.Value(kafka.TraceIDKey).(string)
		logger := clog.WithTraceID(ctx, traceID)
		logger.Info("收到消息",
			clog.String("topic", msg.Topic),
			clog.String("key", string(msg.Key)),
			clog.Int("value_size", len(msg.Value)),
			clog.String("trace_id", traceID),
		)

		// 解析消息
		var event UserEvent
		if err := json.Unmarshal(msg.Value, &event); err != nil {
			logger.Error("解析消息失败", clog.Err(err))
			return err
		}

		// 处理消息
		logger.Info("处理用户事件",
			clog.String("user_id", event.UserID),
			clog.String("event_type", event.EventType),
		)

		// 模拟处理逻辑
		if event.EventType == "error" {
			logger.Error("模拟处理失败")
			return fmt.Errorf("模拟处理失败")
		}

		logger.Info("事件处理成功")
		return nil
	}

	if err := consumer.Subscribe(ctx, topics, handler); err != nil {
		log.Fatal("消费者订阅失败:", err)
	}
}

// sendExampleMessages 发送示例消息
func sendExampleMessages(ctx context.Context, producer kafka.Producer) {
	events := []UserEvent{
		{
			UserID:    "user001",
			EventType: "registered",
			Timestamp: time.Now(),
			Data:      map[string]string{"name": "Alice", "email": "alice@example.com"},
		},
		{
			UserID:    "user002",
			EventType: "registered",
			Timestamp: time.Now(),
			Data:      map[string]string{"name": "Bob", "email": "bob@example.com"},
		},
		{
			UserID:    "user001",
			EventType: "updated",
			Timestamp: time.Now(),
			Data:      map[string]string{"name": "Alice Smith"},
		},
		{
			UserID:    "user003",
			EventType: "error", // 这个消息会触发重试
			Timestamp: time.Now(),
		},
	}

	for _, event := range events {
		data, err := json.Marshal(event)
		if err != nil {
			log.Printf("序列化事件失败: %v", err)
			continue
		}

		msg := &kafka.Message{
			Topic: "example.user.events",
			Key:   []byte(event.UserID),
			Value: data,
		}

		// 添加 trace_id 到 context
		traceID := fmt.Sprintf("trace-%s-%d", event.UserID, time.Now().UnixNano())
		traceCtx := context.WithValue(ctx, kafka.TraceIDKey, traceID)

		// 异步发送
		producer.Send(traceCtx, msg, func(err error) {
			if err != nil {
				clog.WithContext(traceCtx).Error("发送消息失败",
					clog.Err(err),
					clog.String("user_id", event.UserID),
					clog.String("event_type", event.EventType),
				)
			} else {
				clog.WithContext(traceCtx).Info("发送消息成功",
					clog.String("user_id", event.UserID),
					clog.String("event_type", event.EventType),
				)
			}
		})

		// 间隔发送
		time.Sleep(500 * time.Millisecond)
	}

	// 发送一个同步消息示例
	syncEvent := UserEvent{
		UserID:    "sync-user",
		EventType: "sync-event",
		Timestamp: time.Now(),
		Data:      map[string]string{"note": "这是一个同步发送的消息"},
	}

	syncData, _ := json.Marshal(syncEvent)
	syncMsg := &kafka.Message{
		Topic: "example.user.events",
		Key:   []byte(syncEvent.UserID),
		Value: syncData,
	}

	if err := producer.SendSync(ctx, syncMsg); err != nil {
		clog.WithContext(ctx).Error("同步发送消息失败", clog.Err(err))
	} else {
		clog.WithContext(ctx).Info("同步发送消息成功")
	}
}
