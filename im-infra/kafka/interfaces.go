package kafka

import (
	"context"
)

// Message 是跨服务的标准消息结构。
type Message struct {
	Topic   string
	Key     []byte
	Value   []byte
	Headers map[string][]byte
}

// Producer 是一个线程安全的消息生产者接口。
type Producer interface {
	// Send 异步发送消息。此方法立即返回，并通过回调函数处理发送结果。
	// 这是推荐的、性能最高的方式。
	Send(ctx context.Context, msg *Message, callback func(error))

	// SendSync 同步发送消息。此方法将阻塞直到消息发送成功或失败。
	// 适用于需要强一致性保证的场景。
	SendSync(ctx context.Context, msg *Message) error

	// Close 关闭生产者，并确保所有缓冲区的消息都已发送。
	Close() error
}

// ConsumeCallback 是标准的消息处理回调函数。
// 返回 nil: 消息处理成功，偏移量将被自动提交。
// 返回 error: 消息处理失败，偏移量不会被提交，消息将在后续被重新消费。
type ConsumeCallback func(ctx context.Context, msg *Message) error

// Consumer 是一个消费者组的接口。
type Consumer interface {
	// Subscribe 订阅消息并根据处理结果决定是否提交偏移量。
	// 只有当回调函数返回 nil (无错误) 时，偏移量才会被自动提交。
	// 如果返回 error，偏移量将不会被提交，消息会在下一次拉取时被重新消费。
	// 这是标准的、推荐的消费方式。
	Subscribe(ctx context.Context, topics []string, callback ConsumeCallback) error

	// Close 优雅地关闭消费者，完成当前正在处理的消息并提交最后一次偏移量。
	Close() error
}