package clog

import (
	"fmt"
	"time"
)

// String 创建一个字符串类型的日志字段。
func String(key, value string) Field {
	return Field{Key: key, Value: value}
}

// Int 创建一个 int 类型的日志字段。
func Int(key string, value int) Field {
	return Field{Key: key, Value: value}
}

// Int32 创建一个 int32 类型的日志字段。
func Int32(key string, value int32) Field {
	return Field{Key: key, Value: value}
}

// Int64 创建一个 int64 类型的日志字段。
func Int64(key string, value int64) Field {
	return Field{Key: key, Value: value}
}

// Uint 创建一个 uint 类型的日志字段。
func Uint(key string, value uint) Field {
	return Field{Key: key, Value: value}
}

// Uint32 创建一个 uint32 类型的日志字段。
func Uint32(key string, value uint32) Field {
	return Field{Key: key, Value: value}
}

// Uint64 创建一个 uint64 类型的日志字段。
func Uint64(key string, value uint64) Field {
	return Field{Key: key, Value: value}
}

// Float32 创建一个 float32 类型的日志字段。
func Float32(key string, value float32) Field {
	return Field{Key: key, Value: value}
}

// Float64 创建一个 float64 类型的日志字段。
func Float64(key string, value float64) Field {
	return Field{Key: key, Value: value}
}

// Bool 创建一个 bool 类型的日志字段。
func Bool(key string, value bool) Field {
	return Field{Key: key, Value: value}
}

// Time 创建一个 time.Time 类型的日志字段。
func Time(key string, value time.Time) Field {
	return Field{Key: key, Value: value}
}

// Duration 创建一个 time.Duration 类型的日志字段。
func Duration(key string, value time.Duration) Field {
	return Field{Key: key, Value: value}
}

// Any 创建一个任意类型的日志字段。
// 注意：应优先使用类型明确的辅助函数以获得更好的性能和类型安全。
// Any 主要用于处理标准库中没有对应 slog.Value 类型的数据。
func Any(key string, value any) Field {
	return Field{Key: key, Value: value}
}

// Stringer 创建一个实现了 fmt.Stringer 接口的日志字段。
// value 会被转换为其字符串表示形式进行记录。
func Stringer(key string, value fmt.Stringer) Field {
	if value == nil {
		return Field{Key: key, Value: nil}
	}
	return Field{Key: key, Value: value.String()}
}

// Binary 创建一个二进制数据的日志字段。
// 底层 slog 通常会使用 base64 编码来展示。
func Binary(key string, value []byte) Field {
	return Field{Key: key, Value: value}
}

// Strings 创建一个字符串切片的日志字段。
func Strings(key string, values []string) Field {
	return Field{Key: key, Value: values}
}

// Ints 创建一个整数切片的日志字段。
func Ints(key string, values []int) Field {
	return Field{Key: key, Value: values}
}

// Err 是一个用于记录错误的便捷辅助函数。
// 它以 "error" 为标准键名，记录完整的 error 对象，这允许 slog 的 Handler
// 决定如何最高效地序列化错误（例如，可能只记录 err.Error() 字符串）。
// 这是记录错误信息的推荐方式。
func Err(err error) Field {
	return Field{Key: "error", Value: err}
}
