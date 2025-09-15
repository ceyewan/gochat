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

// OrderEvent 订单事件
type OrderEvent struct {
	OrderID      string            `json:"order_id"`
	UserID       string            `json:"user_id"`
	Amount       float64           `json:"amount"`
	Currency     string            `json:"currency"`
	Status       string            `json:"status"`
	Timestamp    time.Time         `json:"timestamp"`
	Metadata     map[string]string `json:"metadata,omitempty"`
}

func main() {
	ctx := context.Background()

	// 1. 初始化 clog
	clog.Init(ctx, clog.GetDefaultConfig("development"))

	// 2. 获取 Kafka 配置
	config := kafka.GetDefaultConfig("development")
	config.Brokers = []string{"localhost:9092"}

	// 3. 创建 Provider (新的统一接口)
	provider, err := kafka.NewProvider(ctx, config, kafka.WithNamespace("provider-example"))
	if err != nil {
		log.Fatal("创建 Provider 失败:", err)
	}
	defer provider.Close()

	// 4. 获取生产者、消费者和 Admin 操作接口
	producer := provider.Producer()
	consumer := provider.Consumer("order-processor-group")
	admin := provider.Admin()

	// 5. 创建必要的 topics
	if err := setupTopics(ctx, admin); err != nil {
		log.Fatal("设置 topics 失败:", err)
	}

	// 6. 启动消费者
	go startOrderProcessor(ctx, consumer)

	// 7. 发送示例订单消息
	sendOrderEvents(ctx, producer)

	// 8. 展示 Admin 操作
	demoAdminOperations(ctx, admin)

	// 9. 等待退出信号
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
	<-sigChan

	fmt.Println("正在关闭程序...")
}

// setupTopics 设置必要的 topics
func setupTopics(ctx context.Context, admin kafka.AdminOperations) error {
	// 创建订单事件 topic
	if err := admin.CreateTopic(ctx, "order.events", 3, 1, map[string]string{
		"retention.ms": "604800000", // 7天保留期
		"cleanup.policy": "delete",
	}); err != nil {
		// Topic 可能已存在，这是可以接受的
		fmt.Printf("创建 order.events topic 失败（可能已存在）: %v\n", err)
	}

	// 创建订单处理结果 topic
	if err := admin.CreateTopic(ctx, "order.processed", 3, 1, map[string]string{
		"retention.ms": "2592000000", // 30天保留期
		"cleanup.policy": "delete",
	}); err != nil {
		fmt.Printf("创建 order.processed topic 失败（可能已存在）: %v\n", err)
	}

	return nil
}

// startOrderProcessor 启动订单处理器
func startOrderProcessor(ctx context.Context, consumer kafka.ConsumerOperations) {
	topics := []string{"order.events"}

	handler := func(ctx context.Context, msg *kafka.Message) error {
		// 从 context 中获取 trace ID
		if traceID, ok := ctx.Value(kafka.TraceIDKey).(string); ok {
			ctx = clog.WithTraceID(ctx, traceID)
		}

		logger := clog.WithContext(ctx)
		logger.Info("收到订单事件",
			clog.String("topic", msg.Topic),
			clog.String("key", string(msg.Key)),
			clog.Int("value_size", len(msg.Value)),
		)

		// 解析订单事件
		var event OrderEvent
		if err := json.Unmarshal(msg.Value, &event); err != nil {
			logger.Error("解析订单事件失败", clog.Err(err))
			return err
		}

		// 处理订单
		logger.Info("开始处理订单",
			clog.String("order_id", event.OrderID),
			clog.String("user_id", event.UserID),
			clog.Float64("amount", event.Amount),
			clog.String("status", event.Status),
		)

		// 模拟订单处理逻辑
		if err := processOrder(ctx, &event); err != nil {
			logger.Error("订单处理失败", clog.Err(err))
			return err
		}

		logger.Info("订单处理成功",
			clog.String("order_id", event.OrderID),
			clog.String("new_status", event.Status),
		)

		return nil
	}

	if err := consumer.Subscribe(ctx, topics, handler); err != nil {
		log.Fatal("消费者订阅失败:", err)
	}
}

// processOrder 模拟订单处理逻辑
func processOrder(ctx context.Context, event *OrderEvent) error {
	logger := clog.WithContext(ctx)

	// 模拟不同的处理逻辑
	switch event.Status {
	case "pending":
		event.Status = "processing"
		time.Sleep(100 * time.Millisecond) // 模拟处理时间

	case "processing":
		event.Status = "completed"
		time.Sleep(200 * time.Millisecond) // 模拟更长的处理时间

	case "completed":
		// 已经完成，不做处理
		return nil

	default:
		return fmt.Errorf("未知的订单状态: %s", event.Status)
	}

	// 模拟可能的失败
	if event.Amount > 10000 {
		return fmt.Errorf("订单金额过大，需要人工审核")
	}

	// 更新时间戳
	event.Timestamp = time.Now()

	logger.Info("订单状态更新",
		clog.String("order_id", event.OrderID),
		clog.String("old_status", event.Status),
		clog.String("new_status", event.Status),
	)

	return nil
}

// sendOrderEvents 发送示例订单事件
func sendOrderEvents(ctx context.Context, producer kafka.ProducerOperations) {
	orders := []OrderEvent{
		{
			OrderID:   "ORD-001",
			UserID:    "user123",
			Amount:    99.99,
			Currency:  "USD",
			Status:    "pending",
			Timestamp: time.Now(),
			Metadata: map[string]string{
				"source": "web",
				"region": "us-east-1",
			},
		},
		{
			OrderID:   "ORD-002",
			UserID:    "user456",
			Amount:    199.50,
			Currency:  "EUR",
			Status:    "pending",
			Timestamp: time.Now(),
			Metadata: map[string]string{
				"source": "mobile",
				"region": "eu-west-1",
			},
		},
		{
			OrderID:   "ORD-003",
			UserID:    "user789",
			Amount:    15000.00, // 大额订单，会触发处理失败
			Currency:  "USD",
			Status:    "pending",
			Timestamp: time.Now(),
			Metadata: map[string]string{
				"source": "api",
				"region": "us-west-1",
			},
		},
	}

	for i, order := range orders {
		data, err := json.Marshal(order)
		if err != nil {
			log.Printf("序列化订单失败: %v", err)
			continue
		}

		msg := &kafka.Message{
			Topic: "order.events",
			Key:   []byte(order.OrderID),
			Value: data,
			Headers: map[string][]byte{
				"event-type": []byte("order_created"),
				"version":    []byte("1.0"),
			},
		}

		// 添加 trace_id 到 context
		traceID := fmt.Sprintf("trace-order-%s-%d", order.OrderID, time.Now().UnixNano())
		traceCtx := context.WithValue(ctx, kafka.TraceIDKey, traceID)

		// 异步发送
		producer.Send(traceCtx, msg, func(err error) {
			if err != nil {
				clog.WithContext(traceCtx).Error("发送订单事件失败",
					clog.Err(err),
					clog.String("order_id", order.OrderID),
				)
			} else {
				clog.WithContext(traceCtx).Info("发送订单事件成功",
					clog.String("order_id", order.OrderID),
					clog.String("user_id", order.UserID),
					clog.Float64("amount", order.Amount),
				)
			}
		})

		// 间隔发送，便于观察
		time.Sleep(time.Second)
	}

	// 发送一个同步消息示例
	syncOrder := OrderEvent{
		OrderID:   "ORD-SYNC-001",
		UserID:    "sync-user",
		Amount:    299.99,
		Currency:  "USD",
		Status:    "pending",
		Timestamp: time.Now(),
	}

	syncData, _ := json.Marshal(syncOrder)
	syncMsg := &kafka.Message{
		Topic: "order.events",
		Key:   []byte(syncOrder.OrderID),
		Value: syncData,
	}

	if err := producer.SendSync(ctx, syncMsg); err != nil {
		clog.WithContext(ctx).Error("同步发送订单事件失败", clog.Err(err))
	} else {
		clog.WithContext(ctx).Info("同步发送订单事件成功",
			clog.String("order_id", syncOrder.OrderID),
		)
	}
}

// demoAdminOperations 展示 Admin 操作
func demoAdminOperations(ctx context.Context, admin kafka.AdminOperations) {
	// 等待一会儿让其他操作完成
	time.Sleep(3 * time.Second)

	fmt.Println("\n=== Admin 操作演示 ===")

	// 列出所有 topics
	topics, err := admin.ListTopics(ctx)
	if err != nil {
		fmt.Printf("列出 topics 失败: %v\n", err)
		return
	}

	fmt.Printf("集群中的 Topics (%d个):\n", len(topics))
	for name, detail := range topics {
		fmt.Printf("  - %s: %d 分区, %d 副本\n", name, detail.NumPartitions, detail.ReplicationFactor)
	}

	// 获取特定 topic 的元数据
	if metadata, err := admin.GetTopicMetadata(ctx, "order.events"); err == nil {
		fmt.Printf("\norder.events 元数据:\n")
		fmt.Printf("  分区数: %d\n", metadata.NumPartitions)
		fmt.Printf("  副本因子: %d\n", metadata.ReplicationFactor)
		fmt.Printf("  配置: %v\n", metadata.Config)
	}

	fmt.Println("=== Admin 操作演示完成 ===\n")
}