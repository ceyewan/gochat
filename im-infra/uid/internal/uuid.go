package internal

import (
	"context"
	"fmt"
	"regexp"
	"strings"
	"time"

	"github.com/ceyewan/gochat/im-infra/clog"
	"github.com/google/uuid"
)

// uuidGenerator UUID 生成器的实现
type uuidGenerator struct {
	config UUIDConfig
	logger clog.Logger
}

// NewUUIDGenerator 创建新的 UUID 生成器
func NewUUIDGenerator(config UUIDConfig) (UUIDGenerator, error) {
	if err := config.Validate(); err != nil {
		return nil, fmt.Errorf("invalid uuid configimpl: %w", err)
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

	u, err := uuid.NewRandom()
	if err != nil {
		g.logger.Error("生成 UUID v4 失败", clog.Err(err))
		return "", fmt.Errorf("failed to generate uuid v4: %w", err)
	}

	uuidStr := u.String()
	if g.config.Format == "simple" {
		uuidStr = strings.ReplaceAll(uuidStr, "-", "")
	}
	if g.config.UpperCase {
		uuidStr = strings.ToUpper(uuidStr)
	}

	g.logger.Debug("生成 UUID v4 成功", clog.String("uuid", uuidStr))
	return uuidStr, nil
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

	u, err := uuid.NewUUID()
	if err != nil {
		g.logger.Error("生成 UUID v7 失败", clog.Err(err))
		return "", fmt.Errorf("failed to generate uuid v7: %w", err)
	}

	uuidStr := u.String()
	if g.config.Format == "simple" {
		uuidStr = strings.ReplaceAll(uuidStr, "-", "")
	}
	if g.config.UpperCase {
		uuidStr = strings.ToUpper(uuidStr)
	}

	g.logger.Debug("生成 UUID v7 成功", clog.String("uuid", uuidStr))
	return uuidStr, nil
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
