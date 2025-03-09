package queue

import (
	"context"
	"encoding/json"
	"sync"
	"time"

	"gochat/clog"
	"gochat/tools"

	"github.com/go-redis/redis/v8"
)

// 队列名称常量
const DefaultQueueName = "gochat:message:queue"

// 全局变量，用于确保初始化函数只执行一次
var (
	once         sync.Once
	initErr      error
)

// NewRedisQueue 创建Redis消息队列实例
func NewRedisQueue(queueName string) *RedisQueue {
	if queueName == "" {
		queueName = DefaultQueueName
	}

	return &RedisQueue{
		ctx:       context.Background(),
		queueName: queueName,
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

// RedisQueue 实现基于Redis的消息队列
type RedisQueue struct {
	client    *redis.Client
	ctx       context.Context
	queueName string
}

// Initialize 初始化Redis连接
func (q *RedisQueue) Initialize() error {
	client, err := tools.GetRedisClient()
	if err != nil {
		clog.Error("[RedisQueue] Initialization failed: %v", err)
		return err
	}
	q.client = client

	if err := q.testConnection(); err != nil {
		return err
	}

	clog.Info("[RedisQueue] Connection successful")
	return nil
}

// Close 关闭Redis连接
func (q *RedisQueue) Close() error {
	q.client = nil
	return nil
}

// PublishMessage 将消息发布到Redis队列
func (q *RedisQueue) PublishMessage(message *QueueMsg) error {
	if q.client == nil {
		if err := q.Initialize(); err != nil {
			return err
		}
	}

	messageByte, err := json.Marshal(message)
	if err != nil {
		clog.Error("[RedisQueue] Message serialization failed: %v", err)
		return err
	}

	if err := q.client.LPush(q.ctx, q.queueName, messageByte).Err(); err != nil {
		clog.Error("[RedisQueue] Message publishing failed: %v", err)
		return err
	}

	return nil
}

// ConsumeMessages 从队列中消费消息，并通过回调函数处理
func (q *RedisQueue) ConsumeMessages(timeout time.Duration, callback func(*QueueMsg) error) error {
	if q.client == nil {
		if err := q.Initialize(); err != nil {
			return err
		}
	}

	for {
		result, err := q.client.BRPop(q.ctx, timeout, q.queueName).Result()
		if err != nil {
			if err == redis.Nil {
				continue
			}
			clog.Error("[RedisQueue] Message consumption failed: %v", err)
			return err
		}

		if len(result) < 2 {
			clog.Warning("[RedisQueue] Received unexpected result: %v", result)
			continue
		}

		var message QueueMsg
		if err := json.Unmarshal([]byte(result[1]), &message); err != nil {
			clog.Error("[RedisQueue] Message deserialization failed: %v", err)
			continue
		}

		if err := callback(&message); err != nil {
			clog.Error("[RedisQueue] Message processing failed: %v", err)
		}
	}
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
