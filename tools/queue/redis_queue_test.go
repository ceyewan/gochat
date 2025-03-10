package queue

// 使用 go test -v gochat/tools/queue 运行测试

import (
	"fmt"
	"gochat/config"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

// TestIntegrationRedisQueue 是一个集成测试，需要实际的Redis服务
func TestIntegrationRedisQueue(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	// 为测试手动设置Redis配置，避免依赖配置文件
	config.Conf.Redis.Addr = "localhost:6379"
	config.Conf.Redis.Password = ""
	config.Conf.Redis.DB = 0

	// 使用唯一的流名称，避免测试干扰
	streamName := fmt.Sprintf("test:integration:stream:%d", time.Now().UnixNano())
	fmt.Printf("Using test stream: %s\n", streamName)

	// 初始化队列
	q := NewRedisQueue(streamName)
	err := q.Initialize()
	if err != nil {
		t.Fatalf("Failed to initialize queue: %v", err)
	}
	defer q.Close()

	// 测试发布多条消息并消费
	var wg sync.WaitGroup
	messageCount := 5
	receivedMessages := make(map[int]bool)
	var mu sync.Mutex

	// 先发布消息
	fmt.Println("Publishing messages...")
	for i := 1; i <= messageCount; i++ {
		msg := &QueueMsg{
			Op:         i,
			InstanceId: fmt.Sprintf("test-server-%d", i),
			Msg:        []byte(fmt.Sprintf("test message %d", i)),
			UserId:     i * 100,
			RoomId:     i * 1000,
		}

		err := q.PublishMessage(msg)
		assert.NoError(t, err)
		fmt.Printf("Published message %d\n", i)
	}

	// 启动消费者
	fmt.Println("Starting consumer...")
	wg.Add(1)
	go func() {
		defer wg.Done()

		consumed := 0

		// 消费消息处理函数
		err := q.ConsumeMessages(1*time.Second, func(msg *QueueMsg) error {
			mu.Lock()
			receivedMessages[msg.UserId] = true
			count := len(receivedMessages)
			mu.Unlock()

			consumed++
			fmt.Printf("Consumed message %d, UserID: %d, Op: %d\n", consumed, msg.UserId, msg.Op)

			// 如果收到了所有消息，可以提前结束
			if count >= messageCount {
				fmt.Println("All messages received, stopping consumer")
				return ErrStopConsumer // 使用特殊错误来终止消费循环
			}

			return nil
		})

		if err != nil {
			t.Errorf("ConsumeMessages returned error: %v", err)
		}
	}()

	// 等待消费完成或超时
	fmt.Println("Waiting for messages to be consumed...")
	done := make(chan struct{})
	go func() {
		wg.Wait()
		close(done)
	}()

	select {
	case <-done:
		// 验证是否收到了所有消息
		mu.Lock()
		receivedCount := len(receivedMessages)
		mu.Unlock()

		fmt.Printf("Test completed: received %d/%d messages\n", receivedCount, messageCount)
		assert.Equal(t, messageCount, receivedCount)
	case <-time.After(10 * time.Second):
		// 在超时前查看已收到的消息数
		mu.Lock()
		receivedCount := len(receivedMessages)
		mu.Unlock()
		t.Fatalf("Test timed out: only received %d/%d messages", receivedCount, messageCount)
	}
}
