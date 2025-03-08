package queue

import (
	"context"
	"encoding/json"
	"gochat/clog"
	"gochat/tools"
	"sync"
	"time"

	"github.com/go-redis/redis/v8"
)

// 队列名称常量
const DefaultQueueName = "gochat:message:queue"

// 全局变量，用于确保初始化函数只执行一次
var (
	once    sync.Once
	initErr error
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

// InitDefaultQueue 初始化默认队列
func InitDefaultQueue() error {
	once.Do(func() {
		queue := NewRedisQueue(DefaultQueueName)
		initErr = queue.Initialize()
		if initErr == nil {
			DefaultQueue = queue
		} else {
			clog.Error("Redis队列初始化失败: %s", initErr.Error())
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
	// 使用全局Redis客户端
	client, err := tools.GetRedisClient()
	if err != nil {
		clog.Error("Redis队列初始化失败: %s", err.Error())
		return err
	}
	q.client = client

	// 测试连接
	pong, err := q.client.Ping(q.ctx).Result()
	if err != nil {
		clog.Error("Redis队列连接测试失败: %s", err.Error())
		return err
	}
	clog.Info("Redis队列连接成功: %s", pong)
	return nil
}

// Close 关闭Redis连接
func (q *RedisQueue) Close() error {
	// 由于使用全局Redis客户端，这里不再关闭连接
	// 连接的生命周期由tools.GetRedisClient()管理
	q.client = nil
	return nil
}

// PublishMessage 将消息发布到Redis队列
func (q *RedisQueue) PublishMessage(message *QueueMsg) error {
	// 确保client已初始化
	if q.client == nil {
		if err := q.Initialize(); err != nil {
			return err
		}
	}

	messageByte, err := json.Marshal(message)
	if err != nil {
		clog.Error("Redis队列序列化消息失败: %s", err.Error())
		return err
	}

	err = q.client.LPush(q.ctx, q.queueName, messageByte).Err()
	if err != nil {
		clog.Error("Redis队列发布消息失败: %s", err.Error())
		return err
	}

	return nil
}

// ConsumeMessages 从队列中消费消息，并通过回调函数处理
func (q *RedisQueue) ConsumeMessages(timeout time.Duration, callback func(*QueueMsg) error) error {
	// 确保client已初始化
	if q.client == nil {
		if err := q.Initialize(); err != nil {
			return err
		}
	}

	for {
		// 使用BRPOP阻塞获取消息，超时后返回
		result, err := q.client.BRPop(q.ctx, timeout, q.queueName).Result()
		if err != nil {
			if err == redis.Nil {
				// 超时，继续等待
				continue
			}
			clog.Error("Redis队列消费消息失败: %s", err.Error())
			return err
		}

		// result[0]是队列名，result[1]是消息内容
		if len(result) < 2 {
			clog.Warning("Redis队列收到异常结果: %v", result)
			continue
		}

		// 解析消息
		var message QueueMsg
		if err := json.Unmarshal([]byte(result[1]), &message); err != nil {
			clog.Error("Redis队列反序列化消息失败: %s", err.Error())
			continue
		}

		// 处理消息
		if err := callback(&message); err != nil {
			clog.Error("处理队列消息失败: %s", err.Error())
			// 根据需要可以将消息放回队列或发送到死信队列
		}
	}
}
