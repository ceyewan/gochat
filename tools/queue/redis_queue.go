package queue

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"
	"time"

	"gochat/clog"
	"gochat/tools"

	"github.com/redis/go-redis/v9"
)

// 队列相关常量
const (
	DefaultQueueName     = "gochat:message:stream"
	DefaultConsumerGroup = "gochat:consumer:group"
	DefaultConsumerName  = "consumer"
	DefaultMaxLen        = 1000             // 流的最大长度
	DefaultClaimMinIdle  = 30 * time.Second // 最小闲置时间，用于XCLAIM
	DefaultClaimCount    = 10               // 每次尝试认领的消息数量
)

// 全局变量，用于确保初始化函数只执行一次
var (
	once    sync.Once
	initErr error
	// 用于发送停止消费信号
	ErrStopConsumer = fmt.Errorf("stop consumer requested")
)

// NewRedisQueue 创建Redis Stream消息队列实例
func NewRedisQueue(queueName string) *RedisQueue {
	if queueName == "" {
		queueName = DefaultQueueName
	}

	return &RedisQueue{
		ctx:           context.Background(),
		streamName:    queueName,
		consumerGroup: DefaultConsumerGroup,
		consumerName:  DefaultConsumerName,
	}
}

// InitRedisQueue 初始化默认队列
func InitRedisQueue() error {
	once.Do(func() {
		queue := NewRedisQueue(DefaultQueueName)
		initErr = queue.Initialize()
		if initErr == nil {
			DefaultQueue = queue
		} else {
			clog.Error("[RedisQueue] Initialization failed: %v", initErr)
		}
	})
	return initErr
}

// RedisQueue 实现基于Redis Stream的消息队列
type RedisQueue struct {
	client        *redis.Client
	ctx           context.Context
	streamName    string
	consumerGroup string
	consumerName  string
}

// Initialize 初始化Redis连接和Stream
func (q *RedisQueue) Initialize() error {
	// 获取Redis客户端
	client, err := tools.GetRedisClient()
	if err != nil {
		clog.Error("[RedisQueue] Initialization failed: %v", err)
		return err
	}
	q.client = client

	// 测试连接
	if err := q.testConnection(); err != nil {
		return err
	}

	// 创建消费者组(如果不存在)
	err = q.createConsumerGroupIfNotExist()
	if err != nil {
		clog.Error("[RedisQueue] Failed to create consumer group: %v", err)
		return err
	}

	clog.Info("[RedisQueue] Connection and stream setup successful")
	return nil
}

// Close 关闭Redis连接
func (q *RedisQueue) Close() error {
	q.client = nil
	return nil
}

// PublishMessage 将消息发布到Redis Stream
func (q *RedisQueue) PublishMessage(message *QueueMsg) error {
	if q.client == nil {
		if err := q.Initialize(); err != nil {
			return err
		}
	}

	// 将消息使用 proto 序列化为字节
	messageByte, err := json.Marshal(message)
	if err != nil {
		clog.Error("[RedisQueue] Message serialization failed: %v", err)
		return err
	}

	// 向Stream添加消息，使用单一字段来存储整个消息
	values := map[string]interface{}{
		"message": string(messageByte),
	}

	// 添加消息并限制流长度
	if err := q.client.XAdd(q.ctx, &redis.XAddArgs{
		Stream: q.streamName,
		MaxLen: DefaultMaxLen,
		Values: values,
	}).Err(); err != nil {
		clog.Error("[RedisQueue] Message publishing failed: %v", err)
		return err
	}

	return nil
}

// ConsumeMessages 从Stream中消费消息
func (q *RedisQueue) ConsumeMessages(timeout time.Duration, callback func(*QueueMsg) error) error {
	if q.client == nil {
		if err := q.Initialize(); err != nil {
			return err
		}
	}

	// 从未处理的消息开始读取
	lastID := "0-0" // 第一次从头开始

	for {
		// 读取新消息
		entries, err := q.client.XReadGroup(q.ctx, &redis.XReadGroupArgs{
			Group:    q.consumerGroup,
			Consumer: q.consumerName,
			Streams:  []string{q.streamName, lastID},
			Count:    10, // 每次获取10条消息
			Block:    timeout,
		}).Result()

		if err != nil {
			if err == redis.Nil { // 超时，没有新消息
				// 尝试处理pending消息（之前已读取但未确认的消息）
				if err := q.processPendingMessages(callback); err != nil {
					clog.Error("[RedisQueue] Failed to process pending messages: %v", err)
				}
				// 继续使用相同的lastID
				continue
			}
			clog.Error("[RedisQueue] Failed to read messages from stream: %v", err)
			time.Sleep(1 * time.Second) // 出错后暂停一下再重试
			continue
		}

		// 没有返回entries或者没有消息，尝试处理pending消息
		if len(entries) == 0 || len(entries[0].Messages) == 0 {
			if err := q.processPendingMessages(callback); err != nil {
				clog.Error("[RedisQueue] Failed to process pending messages: %v", err)
			}
			// 更新为特殊标识符">"，表示只获取新消息
			lastID = ">"
			continue
		}

		// 处理获取到的消息
		for _, message := range entries[0].Messages {
			// 提取消息内容
			messageStr, ok := message.Values["message"].(string)
			if !ok {
				clog.Warning("[RedisQueue] Received message with invalid format")
				q.ackMessage(message.ID) // 确认并丢弃格式错误的消息
				continue
			}

			// 解析消息
			var queueMsg QueueMsg
			if err := json.Unmarshal([]byte(messageStr), &queueMsg); err != nil {
				clog.Error("[RedisQueue] Message deserialization failed: %v", err)
				q.ackMessage(message.ID) // 确认并丢弃解析失败的消息
				continue
			}

			// 处理消息
			if err := callback(&queueMsg); err != nil {
				clog.Error("[RedisQueue] Message processing failed: %v", err)
				// 这里不确认消息，让它留在pending列表中，稍后重试

				// 检查回调返回的错误是否是一个特殊标记，表示要停止消费
				if err == ErrStopConsumer {
					return nil
				}
			} else {
				// 成功处理后确认消息
				q.ackMessage(message.ID)
			}
		}

		// 更新为特殊标识符">"，表示只获取新消息
		lastID = ">"
	}
}

// processPendingMessages 处理pending中的消息（已经读取但未确认的消息）
func (q *RedisQueue) processPendingMessages(callback func(*QueueMsg) error) error {
	// 获取pending列表中的消息
	pendingMessages, err := q.client.XPendingExt(q.ctx, &redis.XPendingExtArgs{
		Stream:   q.streamName,
		Group:    q.consumerGroup,
		Start:    "-",
		End:      "+",
		Count:    DefaultClaimCount,
		Consumer: q.consumerName,
		Idle:     DefaultClaimMinIdle, // 只处理至少闲置30秒的消息
	}).Result()

	if err != nil && err != redis.Nil {
		return err
	}

	if len(pendingMessages) == 0 {
		return nil
	}

	// 处理pending消息
	for _, pendingMsg := range pendingMessages {
		// 认领消息
		claimedMessages, err := q.client.XClaim(q.ctx, &redis.XClaimArgs{
			Stream:   q.streamName,
			Group:    q.consumerGroup,
			Consumer: q.consumerName,
			MinIdle:  DefaultClaimMinIdle,
			Messages: []string{pendingMsg.ID},
		}).Result()

		if err != nil {
			clog.Error("[RedisQueue] Failed to claim message %s: %v", pendingMsg.ID, err)
			continue
		}

		for _, message := range claimedMessages {
			// 提取消息内容
			messageStr, ok := message.Values["message"].(string)
			if !ok {
				q.ackMessage(message.ID) // 确认并丢弃格式错误的消息
				continue
			}

			// 解析消息
			var queueMsg QueueMsg
			if err := json.Unmarshal([]byte(messageStr), &queueMsg); err != nil {
				clog.Error("[RedisQueue] Pending message deserialization failed: %v", err)
				q.ackMessage(message.ID) // 确认并丢弃解析失败的消息
				continue
			}

			// 重新处理消息
			if err := callback(&queueMsg); err != nil {
				clog.Error("[RedisQueue] Pending message processing failed: %v", err)
				// 不确认，让消息留在pending列表中
			} else {
				// 成功处理后确认消息
				q.ackMessage(message.ID)
			}
		}
	}

	return nil
}

// ackMessage 确认消息已处理
func (q *RedisQueue) ackMessage(messageID string) {
	err := q.client.XAck(q.ctx, q.streamName, q.consumerGroup, messageID).Err()
	if err != nil {
		clog.Error("[RedisQueue] Failed to acknowledge message %s: %v", messageID, err)
	}
}

// createConsumerGroupIfNotExist 如果不存在则创建消费者组
func (q *RedisQueue) createConsumerGroupIfNotExist() error {
	// 检查Stream是否存在
	exists, err := q.client.Exists(q.ctx, q.streamName).Result()
	if err != nil {
		return err
	}

	// 如果Stream不存在，则创建一个空Stream
	if exists == 0 {
		// 创建一个带有空消息的Stream，稍后会被自动删除
		_, err = q.client.XGroupCreateMkStream(q.ctx, q.streamName, q.consumerGroup, "$").Result()
		if err != nil && err.Error() != "BUSYGROUP Consumer Group name already exists" {
			return err
		}
		return nil
	}

	// 如果Stream存在，检查消费者组是否已创建
	groups, err := q.client.XInfoGroups(q.ctx, q.streamName).Result()
	if err != nil {
		return err
	}

	// 检查是否有我们需要的消费者组
	for _, group := range groups {
		if group.Name == q.consumerGroup {
			return nil // 消费者组已存在
		}
	}

	// 创建消费者组，从Stream的末尾开始消费
	_, err = q.client.XGroupCreate(q.ctx, q.streamName, q.consumerGroup, "$").Result()
	if err != nil && err.Error() != "BUSYGROUP Consumer Group name already exists" {
		return err
	}

	return nil
}

// testConnection 测试Redis连接
func (q *RedisQueue) testConnection() error {
	pong, err := q.client.Ping(q.ctx).Result()
	if err != nil {
		clog.Error("[RedisQueue] Connection test failed: %v", err)
		return err
	}
	clog.Debug("[RedisQueue] Ping response: %s", pong)
	return nil
}
