package internal

import (
	"bytes"
	"compress/gzip"
	"encoding/json"
	"io"

	"github.com/golang/snappy"
	"github.com/pierrec/lz4/v4"
)

// jsonSerializer JSON序列化器实现
type jsonSerializer struct{}

// newJSONSerializer 创建JSON序列化器
func newJSONSerializer() MessageSerializer {
	return &jsonSerializer{}
}

// Serialize 序列化消息
func (s *jsonSerializer) Serialize(message interface{}) ([]byte, error) {
	data, err := json.Marshal(message)
	if err != nil {
		return nil, NewSerializationError("JSON序列化失败", err)
	}
	return data, nil
}

// Deserialize 反序列化消息
func (s *jsonSerializer) Deserialize(data []byte, target interface{}) error {
	if err := json.Unmarshal(data, target); err != nil {
		return NewDeserializationError("JSON反序列化失败", err)
	}
	return nil
}

// ContentType 返回内容类型
func (s *jsonSerializer) ContentType() string {
	return "application/json"
}

// gzipCodec Gzip压缩编解码器
type gzipCodec struct{}

// newGzipCodec 创建Gzip压缩编解码器
func newGzipCodec() CompressionCodec {
	return &gzipCodec{}
}

// Compress 压缩数据
func (c *gzipCodec) Compress(data []byte) ([]byte, error) {
	var buf bytes.Buffer
	writer := gzip.NewWriter(&buf)

	if _, err := writer.Write(data); err != nil {
		writer.Close()
		return nil, NewCompressionError("Gzip压缩失败", err)
	}

	if err := writer.Close(); err != nil {
		return nil, NewCompressionError("Gzip压缩关闭失败", err)
	}

	return buf.Bytes(), nil
}

// Decompress 解压数据
func (c *gzipCodec) Decompress(data []byte) ([]byte, error) {
	reader, err := gzip.NewReader(bytes.NewReader(data))
	if err != nil {
		return nil, NewDecompressionError("Gzip解压初始化失败", err)
	}
	defer reader.Close()

	result, err := io.ReadAll(reader)
	if err != nil {
		return nil, NewDecompressionError("Gzip解压失败", err)
	}

	return result, nil
}

// Type 返回压缩类型
func (c *gzipCodec) Type() string {
	return "gzip"
}

// snappyCodec Snappy压缩编解码器
type snappyCodec struct{}

// newSnappyCodec 创建Snappy压缩编解码器
func newSnappyCodec() CompressionCodec {
	return &snappyCodec{}
}

// Compress 压缩数据
func (c *snappyCodec) Compress(data []byte) ([]byte, error) {
	compressed := snappy.Encode(nil, data)
	return compressed, nil
}

// Decompress 解压数据
func (c *snappyCodec) Decompress(data []byte) ([]byte, error) {
	decompressed, err := snappy.Decode(nil, data)
	if err != nil {
		return nil, NewDecompressionError("Snappy解压失败", err)
	}
	return decompressed, nil
}

// Type 返回压缩类型
func (c *snappyCodec) Type() string {
	return "snappy"
}

// lz4Codec LZ4压缩编解码器
type lz4Codec struct{}

// newLZ4Codec 创建LZ4压缩编解码器
func newLZ4Codec() CompressionCodec {
	return &lz4Codec{}
}

// Compress 压缩数据
func (c *lz4Codec) Compress(data []byte) ([]byte, error) {
	var buf bytes.Buffer
	writer := lz4.NewWriter(&buf)

	if _, err := writer.Write(data); err != nil {
		writer.Close()
		return nil, NewCompressionError("LZ4压缩失败", err)
	}

	if err := writer.Close(); err != nil {
		return nil, NewCompressionError("LZ4压缩关闭失败", err)
	}

	return buf.Bytes(), nil
}

// Decompress 解压数据
func (c *lz4Codec) Decompress(data []byte) ([]byte, error) {
	reader := lz4.NewReader(bytes.NewReader(data))

	result, err := io.ReadAll(reader)
	if err != nil {
		return nil, NewDecompressionError("LZ4解压失败", err)
	}

	return result, nil
}

// Type 返回压缩类型
func (c *lz4Codec) Type() string {
	return "lz4"
}

// noCompressionCodec 无压缩编解码器
type noCompressionCodec struct{}

// newNoCompressionCodec 创建无压缩编解码器
func newNoCompressionCodec() CompressionCodec {
	return &noCompressionCodec{}
}

// Compress 不进行压缩，直接返回原数据
func (c *noCompressionCodec) Compress(data []byte) ([]byte, error) {
	return data, nil
}

// Decompress 不进行解压，直接返回原数据
func (c *noCompressionCodec) Decompress(data []byte) ([]byte, error) {
	return data, nil
}

// Type 返回压缩类型
func (c *noCompressionCodec) Type() string {
	return "none"
}

// NewCompressionCodec 根据类型创建压缩编解码器
func NewCompressionCodec(compressionType string) CompressionCodec {
	return newCompressionCodec(compressionType)
}

// newCompressionCodec 根据类型创建压缩编解码器
func newCompressionCodec(compressionType string) CompressionCodec {
	switch compressionType {
	case "gzip":
		return newGzipCodec()
	case "snappy":
		return newSnappyCodec()
	case "lz4":
		return newLZ4Codec()
	case "none", "":
		return newNoCompressionCodec()
	default:
		// 默认使用无压缩
		return newNoCompressionCodec()
	}
}

// MessageUtils 消息工具类
type MessageUtils struct {
	serializer MessageSerializer
	compressor CompressionCodec
}

// NewMessageUtils 创建消息工具类
func NewMessageUtils(serializerType, compressionType string) *MessageUtils {
	var serializer MessageSerializer
	switch serializerType {
	case "json":
		serializer = newJSONSerializer()
	default:
		serializer = newJSONSerializer()
	}

	compressor := newCompressionCodec(compressionType)

	return &MessageUtils{
		serializer: serializer,
		compressor: compressor,
	}
}

// SerializeAndCompress 序列化并压缩消息
func (mu *MessageUtils) SerializeAndCompress(message interface{}) ([]byte, error) {
	// 序列化
	serialized, err := mu.serializer.Serialize(message)
	if err != nil {
		return nil, err
	}

	// 压缩
	compressed, err := mu.compressor.Compress(serialized)
	if err != nil {
		return nil, err
	}

	return compressed, nil
}

// DecompressAndDeserialize 解压并反序列化消息
func (mu *MessageUtils) DecompressAndDeserialize(data []byte, target interface{}) error {
	// 解压
	decompressed, err := mu.compressor.Decompress(data)
	if err != nil {
		return err
	}

	// 反序列化
	if err := mu.serializer.Deserialize(decompressed, target); err != nil {
		return err
	}

	return nil
}

// GetCompressionRatio 计算压缩比
func (mu *MessageUtils) GetCompressionRatio(originalData []byte) (float64, error) {
	compressed, err := mu.compressor.Compress(originalData)
	if err != nil {
		return 0, err
	}

	if len(originalData) == 0 {
		return 0, nil
	}

	ratio := float64(len(compressed)) / float64(len(originalData))
	return ratio, nil
}

// IsCompressionBeneficial 判断压缩是否有益
// 对于小于阈值的消息，压缩可能不会带来显著收益，甚至可能增加开销
func (mu *MessageUtils) IsCompressionBeneficial(data []byte, threshold int) bool {
	if len(data) < threshold {
		return false
	}

	// 对于小消息，快速检查压缩效果
	if len(data) < 1024 { // 1KB以下
		compressed, err := mu.compressor.Compress(data)
		if err != nil {
			return false
		}

		// 如果压缩后大小减少不到10%，则认为压缩无益
		return float64(len(compressed))/float64(len(data)) < 0.9
	}

	// 对于较大消息，默认认为压缩有益
	return true
}

// OptimizeForSmallMessages 为小消息优化
// 根据消息大小和类型选择最佳的序列化和压缩策略
func (mu *MessageUtils) OptimizeForSmallMessages(data []byte, messageType string) ([]byte, error) {
	// 对于非常小的消息（<100字节），不进行压缩
	if len(data) < 100 {
		return data, nil
	}

	// 对于小消息（<1KB），使用快速压缩算法
	if len(data) < 1024 {
		if mu.compressor.Type() == "lz4" || mu.compressor.Type() == "snappy" {
			return mu.compressor.Compress(data)
		}
		// 如果当前不是快速压缩算法，则不压缩
		return data, nil
	}

	// 对于较大消息，使用配置的压缩算法
	return mu.compressor.Compress(data)
}

// CreateCompressionError 创建压缩错误
func CreateCompressionError(message string, cause error) *MQError {
	return NewMQError(ErrCodeCompressionFailed, message, cause)
}

// CreateDecompressionError 创建解压错误
func CreateDecompressionError(message string, cause error) *MQError {
	return NewMQError(ErrCodeDecompressionFailed, message, cause)
}

// NewCompressionError 创建压缩错误（向后兼容）
func NewCompressionError(message string, cause error) *MQError {
	return CreateCompressionError(message, cause)
}

// NewDecompressionError 创建解压错误（向后兼容）
func NewDecompressionError(message string, cause error) *MQError {
	return CreateDecompressionError(message, cause)
}
