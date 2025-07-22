package internal

import (
	"context"
	"time"
)

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
	
	// Close 关闭幂等客户端，释放资源
	Close() error
}

// BatchIdempotent 定义批量幂等操作的接口
type BatchIdempotent interface {
	// BatchCheck 批量检查多个键是否存在
	BatchCheck(ctx context.Context, keys []string) (map[string]bool, error)
	
	// BatchSet 批量设置多个幂等标记
	BatchSet(ctx context.Context, keys []string, ttl time.Duration) (map[string]bool, error)
	
	// BatchDelete 批量删除多个幂等标记
	BatchDelete(ctx context.Context, keys []string) error
}

// ResultIdempotent 定义带结果存储的幂等操作接口
type ResultIdempotent interface {
	// SetWithTypedResult 设置幂等标记并存储指定类型的结果
	SetWithTypedResult(ctx context.Context, key string, result interface{}, ttl time.Duration) (bool, error)
	
	// GetTypedResult 获取指定类型的存储结果
	GetTypedResult(ctx context.Context, key string, result interface{}) error
	
	// HasResult 检查是否存储了结果
	HasResult(ctx context.Context, key string) (bool, error)
}

// AdvancedIdempotent 定义高级幂等操作的接口
type AdvancedIdempotent interface {
	Idempotent
	BatchIdempotent
	ResultIdempotent
	
	// SetWithCallback 设置幂等标记，如果是首次设置则执行回调函数
	SetWithCallback(ctx context.Context, key string, ttl time.Duration, callback func() (interface{}, error)) (interface{}, error)
	
	// GetOrSet 获取结果，如果不存在则执行回调并设置
	GetOrSet(ctx context.Context, key string, ttl time.Duration, callback func() (interface{}, error)) (interface{}, error)
	
	// CompareAndSet 比较并设置，只有当前值匹配时才设置新值
	CompareAndSet(ctx context.Context, key string, expectedValue, newValue interface{}, ttl time.Duration) (bool, error)
}

// KeyOperations 定义键操作的接口
type KeyOperations interface {
	// FormatKey 格式化键名（添加前缀等）
	FormatKey(key string) string
	
	// ValidateKey 验证键名是否有效
	ValidateKey(key string) error
	
	// GenerateKey 根据模板生成键名
	GenerateKey(template string, args ...interface{}) string
}

// ConnectionOperations 定义连接操作的接口
type ConnectionOperations interface {
	// Ping 检查 Redis 连接是否正常
	Ping(ctx context.Context) error
	
	// Stats 获取连接统计信息
	Stats() interface{}
	
	// Health 获取健康状态
	Health(ctx context.Context) error
}

// ManagementOperations 定义管理操作的接口
type ManagementOperations interface {
	KeyOperations
	ConnectionOperations
	
	// GetConfig 获取当前配置
	GetConfig() interface{}
	
	// SetConfig 更新配置
	SetConfig(config interface{}) error
}

// IdempotentResult 定义幂等操作的结果
type IdempotentResult struct {
	// Success 操作是否成功
	Success bool
	
	// FirstTime 是否为首次执行
	FirstTime bool
	
	// Value 存储的值
	Value interface{}
	
	// TTL 剩余过期时间
	TTL time.Duration
	
	// Error 错误信息
	Error error
}

// IdempotentStatus 定义幂等状态
type IdempotentStatus struct {
	// Exists 键是否存在
	Exists bool
	
	// HasResult 是否存储了结果
	HasResult bool
	
	// TTL 剩余过期时间
	TTL time.Duration
	
	// CreatedAt 创建时间
	CreatedAt time.Time
	
	// UpdatedAt 更新时间
	UpdatedAt time.Time
}

// CallbackFunc 定义回调函数类型
type CallbackFunc func() (interface{}, error)

// ValidatorFunc 定义验证函数类型
type ValidatorFunc func(key string) error

// SerializerFunc 定义序列化函数类型
type SerializerFunc func(value interface{}) ([]byte, error)

// DeserializerFunc 定义反序列化函数类型
type DeserializerFunc func(data []byte, value interface{}) error
