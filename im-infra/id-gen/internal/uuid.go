package internal

import (
	"context"
	"crypto/rand"
	"fmt"
	"regexp"
	"strings"
	"time"

	"github.com/ceyewan/gochat/im-infra/clog"
)

// uuidGenerator UUID 生成器的实现
type uuidGenerator struct {
	config UUIDConfig
	logger clog.Logger
}

// NewUUIDGenerator 创建新的 UUID 生成器
func NewUUIDGenerator(config UUIDConfig) (UUIDGenerator, error) {
	if err := config.Validate(); err != nil {
		return nil, fmt.Errorf("invalid uuid config: %w", err)
	}

	logger := clog.Module("idgen")
	logger.Info("创建 UUID 生成器",
		clog.Int("version", config.Version),
		clog.String("format", config.Format),
		clog.Bool("upper_case", config.UpperCase),
	)

	return &uuidGenerator{
		config: config,
		logger: logger,
	}, nil
}

// GenerateString 生成字符串类型的 UUID
func (g *uuidGenerator) GenerateString(ctx context.Context) (string, error) {
	switch g.config.Version {
	case 4:
		return g.GenerateV4(ctx)
	case 7:
		return g.GenerateV7(ctx)
	default:
		return g.GenerateV4(ctx)
	}
}

// GenerateInt64 UUID 不支持 int64 类型
func (g *uuidGenerator) GenerateInt64(ctx context.Context) (int64, error) {
	return 0, fmt.Errorf("uuid generator does not support int64 generation")
}

// GenerateV4 生成 UUID v4
func (g *uuidGenerator) GenerateV4(ctx context.Context) (string, error) {
	start := time.Now()
	defer func() {
		duration := time.Since(start)
		g.logger.Debug("生成 UUID v4",
			clog.Duration("duration", duration),
		)
	}()

	// 生成 16 字节的随机数据
	bytes := make([]byte, 16)
	if _, err := rand.Read(bytes); err != nil {
		g.logger.Error("生成随机数据失败", clog.Err(err))
		return "", fmt.Errorf("failed to generate random bytes: %w", err)
	}

	// 设置版本号 (4) 和变体位
	bytes[6] = (bytes[6] & 0x0f) | 0x40 // 版本 4
	bytes[8] = (bytes[8] & 0x3f) | 0x80 // 变体位

	uuid := g.formatUUID(bytes)
	g.logger.Debug("生成 UUID v4 成功", clog.String("uuid", uuid))
	return uuid, nil
}

// GenerateV7 生成 UUID v7 (时间排序)
func (g *uuidGenerator) GenerateV7(ctx context.Context) (string, error) {
	start := time.Now()
	defer func() {
		duration := time.Since(start)
		g.logger.Debug("生成 UUID v7",
			clog.Duration("duration", duration),
		)
	}()

	// 获取当前时间戳（毫秒）
	timestamp := time.Now().UnixMilli()

	// 生成 16 字节数组
	bytes := make([]byte, 16)

	// 前 6 字节：48 位时间戳（毫秒）
	bytes[0] = byte(timestamp >> 40)
	bytes[1] = byte(timestamp >> 32)
	bytes[2] = byte(timestamp >> 24)
	bytes[3] = byte(timestamp >> 16)
	bytes[4] = byte(timestamp >> 8)
	bytes[5] = byte(timestamp)

	// 后 10 字节：随机数据
	if _, err := rand.Read(bytes[6:]); err != nil {
		g.logger.Error("生成随机数据失败", clog.Err(err))
		return "", fmt.Errorf("failed to generate random bytes: %w", err)
	}

	// 设置版本号 (7) 和变体位
	bytes[6] = (bytes[6] & 0x0f) | 0x70 // 版本 7
	bytes[8] = (bytes[8] & 0x3f) | 0x80 // 变体位

	uuid := g.formatUUID(bytes)
	g.logger.Debug("生成 UUID v7 成功", clog.String("uuid", uuid))
	return uuid, nil
}

// formatUUID 格式化 UUID 字节数组为字符串
func (g *uuidGenerator) formatUUID(bytes []byte) string {
	var uuid string
	
	if g.config.Format == "simple" {
		// 简单格式：不带连字符
		uuid = fmt.Sprintf("%x", bytes)
	} else {
		// 标准格式：带连字符
		uuid = fmt.Sprintf("%x-%x-%x-%x-%x",
			bytes[0:4], bytes[4:6], bytes[6:8], bytes[8:10], bytes[10:16])
	}

	if g.config.UpperCase {
		uuid = strings.ToUpper(uuid)
	}

	return uuid
}

// Validate 验证 UUID 格式是否正确
func (g *uuidGenerator) Validate(uuid string) bool {
	if uuid == "" {
		return false
	}

	// 标准格式：8-4-4-4-12
	standardPattern := `^[0-9a-fA-F]{8}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{12}$`
	// 简单格式：32 位十六进制字符
	simplePattern := `^[0-9a-fA-F]{32}$`

	standardRegex := regexp.MustCompile(standardPattern)
	simpleRegex := regexp.MustCompile(simplePattern)

	return standardRegex.MatchString(uuid) || simpleRegex.MatchString(uuid)
}

// Type 返回生成器类型
func (g *uuidGenerator) Type() GeneratorType {
	return UUIDType
}

// Close 关闭生成器
func (g *uuidGenerator) Close() error {
	g.logger.Info("关闭 UUID 生成器")
	return nil
}
