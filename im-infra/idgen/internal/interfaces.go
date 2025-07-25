package internal

import (
	"context"
)

// IDGenerator 定义 ID 生成器的核心接口。
// 提供统一的 ID 生成抽象，支持不同类型的 ID 生成策略。
type IDGenerator interface {
	// GenerateString 生成字符串类型的 ID
	GenerateString(ctx context.Context) (string, error)
	
	// GenerateInt64 生成 int64 类型的 ID
	GenerateInt64(ctx context.Context) (int64, error)
	
	// Type 返回 ID 生成器的类型
	Type() GeneratorType
	
	// Close 关闭 ID 生成器，释放资源
	Close() error
}

// SnowflakeGenerator 定义雪花算法 ID 生成器的接口
type SnowflakeGenerator interface {
	IDGenerator
	
	// GetNodeID 获取当前节点 ID
	GetNodeID() int64
	
	// ParseID 解析雪花 ID，返回时间戳、节点 ID 和序列号
	ParseID(id int64) (timestamp int64, nodeID int64, sequence int64)
}

// UUIDGenerator 定义 UUID 生成器的接口
type UUIDGenerator interface {
	IDGenerator
	
	// GenerateV4 生成 UUID v4
	GenerateV4(ctx context.Context) (string, error)
	
	// GenerateV7 生成 UUID v7 (时间排序)
	GenerateV7(ctx context.Context) (string, error)
	
	// Validate 验证 UUID 格式是否正确
	Validate(uuid string) bool
}

// RedisIDGenerator 定义 Redis 自增 ID 生成器的接口
type RedisIDGenerator interface {
	IDGenerator
	
	// GenerateWithKey 使用指定键生成 ID
	GenerateWithKey(ctx context.Context, key string) (int64, error)
	
	// GenerateWithStep 使用指定步长生成 ID
	GenerateWithStep(ctx context.Context, step int64) (int64, error)
	
	// Reset 重置指定键的计数器
	Reset(ctx context.Context, key string) error
	
	// GetCurrent 获取当前计数值
	GetCurrent(ctx context.Context, key string) (int64, error)
}

// GeneratorType 定义 ID 生成器类型
type GeneratorType string

const (
	// SnowflakeType 雪花算法类型
	SnowflakeType GeneratorType = "snowflake"
	
	// UUIDType UUID 类型
	UUIDType GeneratorType = "uuid"
	
	// RedisType Redis 自增类型
	RedisType GeneratorType = "redis"
)

// String 返回生成器类型的字符串表示
func (t GeneratorType) String() string {
	return string(t)
}

// IsValid 检查生成器类型是否有效
func (t GeneratorType) IsValid() bool {
	switch t {
	case SnowflakeType, UUIDType, RedisType:
		return true
	default:
		return false
	}
}

// StringOperations 定义字符串 ID 操作的接口
type StringOperations interface {
	GenerateString(ctx context.Context) (string, error)
}

// Int64Operations 定义 int64 ID 操作的接口
type Int64Operations interface {
	GenerateInt64(ctx context.Context) (int64, error)
}

// ValidationOperations 定义 ID 验证操作的接口
type ValidationOperations interface {
	Validate(id string) bool
}

// ManagementOperations 定义 ID 生成器管理操作的接口
type ManagementOperations interface {
	Type() GeneratorType
	Close() error
}
