package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"

	"github.com/ceyewan/gochat/im-infra/mq"
)

func main() {
	log.Println("启动优雅退出示例...")

	// 使用部分配置，其他字段将使用默认值
	cfg := mq.Config{
		Brokers:  []string{"localhost:19092"},
		ClientID: "graceful-shutdown-example",
		ConsumerConfig: mq.ConsumerConfig{
			GroupID: "graceful-shutdown-group",
		},
	}

	// 创建 MQ 实例
	mqInstance, err := mq.New(cfg)
	if err != nil {
		log.Fatalf("创建MQ实例失败: %v", err)
	}
	defer mqInstance.Close()

	// 创建用于优雅关闭的 context
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	var wg sync.WaitGroup

	// 启动一个简单的生产者
	wg.Add(1)
	go func() {
		defer wg.Done()
		simpleProducer(ctx, mqInstance)
	}()

	// 启动一个简单的监控
	wg.Add(1)
	go func() {
		defer wg.Done()
		simpleMonitor(ctx, mqInstance)
	}()

	// 等待中断信号
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	log.Println("程序已启动，按 Ctrl+C 测试优雅退出...")
	<-sigChan

	log.Println("🛑 收到关闭信号，开始优雅关闭...")

	// 取消 context，通知所有协程退出
	cancel()

	// 等待所有协程完成
	log.Println("⏳ 等待所有协程完成...")
	wg.Wait()

	log.Println("✅ 程序已优雅退出")
}

// simpleProducer 简单的生产者，每5秒发送一条消息
func simpleProducer(ctx context.Context, mqInstance mq.MQ) {
	producer := mqInstance.Producer()
	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()

	messageCount := 0

	for {
		select {
		case <-ctx.Done():
			log.Println("📤 生产者收到退出信号，正在关闭...")
			return
		case <-ticker.C:
			messageCount++
			message := []byte("Hello from graceful shutdown example #" + string(rune(messageCount+'0')))

			sendCtx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
			err := producer.SendSync(sendCtx, "test-topic", message)
			cancel()

			if err != nil {
				log.Printf("📤 发送消息失败: %v", err)
			} else {
				log.Printf("📤 发送消息成功: %s", string(message))
			}
		}
	}
}

// simpleMonitor 简单的监控，每10秒输出一次状态
func simpleMonitor(ctx context.Context, mqInstance mq.MQ) {
	ticker := time.NewTicker(10 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			log.Println("📊 监控服务收到退出信号，正在关闭...")
			return
		case <-ticker.C:
			// 获取生产者指标
			metrics := mqInstance.Producer().GetMetrics()
			log.Printf("📊 生产者状态 - 总消息: %d, 成功: %d, 失败: %d",
				metrics.TotalMessages,
				metrics.SuccessMessages,
				metrics.FailedMessages)

			// 健康检查
			pingCtx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
			err := mqInstance.Ping(pingCtx)
			cancel()

			if err != nil {
				log.Printf("❌ 健康检查失败: %v", err)
			} else {
				log.Println("✅ 健康检查通过")
			}
		}
	}
}
