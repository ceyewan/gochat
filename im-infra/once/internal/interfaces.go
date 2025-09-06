package internal

import (
	"context"
	"time"
)

// IdempotentStatus 定义幂等状态
type IdempotentStatus int

const (
	// StatusNotExecuted 未执行状态
	StatusNotExecuted IdempotentStatus = iota
	// StatusExecuting 执行中状态
	StatusExecuting
	// StatusExecuted 已执行状态
	StatusExecuted
	// StatusFailed 执行失败状态
	StatusFailed
)

// IdempotentResult 定义幂等操作的结果
type IdempotentResult struct {
	// Status 幂等状态
	Status IdempotentStatus
	// Result 操作结果
	Result interface{}
	// Error 错误信息
	Error error
	// CreatedAt 创建时间
	CreatedAt time.Time
	// UpdatedAt 更新时间
	UpdatedAt time.Time
}

// BatchIdempotent 定义批量幂等操作的接口
type BatchIdempotent interface {
	// BatchCheck 批量检查键是否存在
	BatchCheck(ctx context.Context, keys []string) (map[string]bool, error)
	// BatchSet 批量设置幂等标记
	BatchSet(ctx context.Context, keys []string, ttl time.Duration) (map[string]bool, error)
	// BatchDelete 批量删除幂等标记
	BatchDelete(ctx context.Context, keys []string) error
}

// ResultIdempotent 定义带结果存储的幂等操作接口
type ResultIdempotent interface {
	// SetWithResult 设置幂等标记并存储操作结果
	SetWithResult(ctx context.Context, key string, result interface{}, ttl time.Duration) (bool, error)
	// GetResult 获取存储的操作结果
	GetResult(ctx context.Context, key string) (interface{}, error)
	// GetResultWithStatus 获取结果和状态
	GetResultWithStatus(ctx context.Context, key string) (*IdempotentResult, error)
}

// AdvancedIdempotent 定义高级幂等操作的接口
type AdvancedIdempotent interface {
	// Do 执行幂等操作，如果已执行过则跳过，否则执行函数
	Do(ctx context.Context, key string, f func() error) error
	// Execute 执行幂等操作并返回结果
	Execute(ctx context.Context, key string, ttl time.Duration, callback func() (interface{}, error)) (interface{}, error)
	// ExecuteSimple 执行简单幂等操作
	ExecuteSimple(ctx context.Context, key string, ttl time.Duration, callback func() error) error
}

// Idempotent 定义幂等操作的核心接口。
// 提供基于 Redis setnx 命令的幂等检查和设置功能。
type Idempotent interface {
	// Check 检查指定键是否已经存在（是否已执行过）
	Check(ctx context.Context, key string) (bool, error)

	// Set 设置幂等标记，如果键已存在则返回 false
	Set(ctx context.Context, key string, ttl time.Duration) (bool, error)

	// CheckAndSet 原子性地检查并设置幂等标记
	// 返回值：(是否首次设置, 错误)
	CheckAndSet(ctx context.Context, key string, ttl time.Duration) (bool, error)

	// SetWithResult 设置幂等标记并存储操作结果
	SetWithResult(ctx context.Context, key string, result interface{}, ttl time.Duration) (bool, error)

	// GetResult 获取存储的操作结果
	GetResult(ctx context.Context, key string) (interface{}, error)

	// Delete 删除幂等标记
	Delete(ctx context.Context, key string) error

	// Exists 检查键是否存在（别名方法，与 Check 功能相同）
	Exists(ctx context.Context, key string) (bool, error)

	// TTL 获取键的剩余过期时间
	TTL(ctx context.Context, key string) (time.Duration, error)

	// Refresh 刷新键的过期时间
	Refresh(ctx context.Context, key string, ttl time.Duration) error

	// Do 执行幂等操作，如果已执行过则跳过，否则执行函数
	Do(ctx context.Context, key string, f func() error) error

	// Close 关闭幂等客户端，释放资源
	Close() error
}
