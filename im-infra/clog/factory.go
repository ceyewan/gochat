package clog

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"gopkg.in/natefinch/lumberjack.v2"
)

// NewLogger 创建一个新的日志器实例。
// 它接受一系列的 Option 函数来定制日志器的行为。
func NewLogger(opts ...Option) (Logger, error) {
	// 1. 初始化默认配置
	cfg := DefaultConfig()

	// 2. 应用所有用户提供的选项
	for _, opt := range opts {
		opt(cfg)
	}

	// 3. 填充未明确设置的默认值
	cfg.fillDefaultConfig()

	// 验证配置
	if err := cfg.Validate(); err != nil {
		return nil, err
	}

	// 4. 创建原子级别控制器
	atomicLevel := zap.NewAtomicLevel()
	if err := atomicLevel.UnmarshalText([]byte(cfg.Level)); err != nil {
		atomicLevel.SetLevel(zapcore.InfoLevel)
	}

	// 5. 创建编码器配置
	encoderConfig := createEncoderConfig(cfg)

	// 6. 创建核心 (Core)
	var cores []zapcore.Core

	// 文件核心
	if cfg.Filename != "" {
		fileWriter, _, err := createLogWriter(cfg)
		if err != nil {
			return nil, fmt.Errorf("无法创建日志文件写入器: %w", err)
		}
		// 根据配置创建文件编码器 (文件输出不带颜色)
		fileEncoder := createEncoder(cfg, encoderConfig, true, false)
		cores = append(cores, zapcore.NewCore(fileEncoder, fileWriter, atomicLevel))
	}

	// 控制台核心
	if cfg.ConsoleOutput {
		consoleWriter := zapcore.AddSync(os.Stdout)
		consoleEncoder := createEncoder(cfg, encoderConfig, false, cfg.EnableColor) // 控制台可根据配置带颜色
		cores = append(cores, zapcore.NewCore(consoleEncoder, consoleWriter, atomicLevel))
	}

	// 如果没有配置任何输出，则默认输出到控制台
	if len(cores) == 0 {
		consoleWriter := zapcore.AddSync(os.Stdout)
		consoleEncoder := createEncoder(cfg, encoderConfig, false, true)
		cores = append(cores, zapcore.NewCore(consoleEncoder, consoleWriter, atomicLevel))
	}

	core := zapcore.NewTee(cores...)

	// 7. 创建 Zap 日志器
	zapLogger := createZapLogger(core, cfg)

	// 8. 创建我们的 ZapLogger 封装
	logger := &ZapLogger{
		zap:         zapLogger,
		sugar:       zapLogger.Sugar(),
		config:      cfg,
		atomicLevel: atomicLevel,
	}

	return logger, nil
}

// createEncoderConfig 创建编码器配置
func createEncoderConfig(config *Config) zapcore.EncoderConfig {
	encoderConfig := zapcore.EncoderConfig{
		TimeKey:        "timestamp",
		LevelKey:       "level",
		NameKey:        "module",
		CallerKey:      "caller",
		MessageKey:     "message",
		StacktraceKey:  "stacktrace",
		LineEnding:     zapcore.DefaultLineEnding,
		EncodeLevel:    zapcore.LowercaseLevelEncoder,
		EncodeTime:     zapcore.ISO8601TimeEncoder,
		EncodeDuration: zapcore.SecondsDurationEncoder,
		EncodeCaller:   zapcore.ShortCallerEncoder,
		EncodeName: func(name string, enc zapcore.PrimitiveArrayEncoder) {
			if name != "" {
				enc.AppendString(name)
			}
		},
	}

	// 使用 ISO 8601 时间格式
	encoderConfig.EncodeTime = zapcore.ISO8601TimeEncoder

	return encoderConfig
}

// createLogWriter 创建日志文件写入器
func createLogWriter(config *Config) (zapcore.WriteSyncer, *lumberjack.Logger, error) {
	dir := filepath.Dir(config.Filename)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return nil, nil, fmt.Errorf("创建日志目录失败: %w", err)
	}

	rotator := &lumberjack.Logger{
		Filename:   config.Filename,
		MaxSize:    config.FileRotation.MaxSize,
		MaxBackups: config.FileRotation.MaxBackups,
		MaxAge:     config.FileRotation.MaxAge,
		Compress:   config.FileRotation.Compress,
	}
	writer := zapcore.AddSync(rotator)

	return writer, rotator, nil
}

// createEncoder 根据配置创建编码器
func createEncoder(config *Config, encoderConfig zapcore.EncoderConfig, isFile, withColor bool) zapcore.Encoder {
	// 确定要使用的格式
	format := config.Format
	if isFile && config.FileFormat != "" {
		format = config.FileFormat
	}

	// 对于 JSON 格式，颜色选项无效
	if format == FormatJSON {
		return zapcore.NewJSONEncoder(encoderConfig)
	}

	// 对于 Console 格式，根据 withColor 参数决定是否使用颜色
	if withColor {
		encoderConfig.EncodeLevel = zapcore.CapitalColorLevelEncoder
	} else {
		encoderConfig.EncodeLevel = zapcore.CapitalLevelEncoder
	}
	return zapcore.NewConsoleEncoder(encoderConfig)
}

// createZapLogger 创建并配置zap日志器
func createZapLogger(core zapcore.Core, config *Config) *zap.Logger {
	var zapOpts []zap.Option

	if config.EnableCaller {
		// 定义常量避免魔法数字
		const DefaultCallerSkip = 1 // clog.Info -> logger.Info -> zap.Info
		zapOpts = append(zapOpts, zap.AddCaller(), zap.AddCallerSkip(DefaultCallerSkip))
	}

	// 添加堆栈跟踪
	zapOpts = append(zapOpts, zap.AddStacktrace(parseLevel(ErrorLevel)))

	// 添加初始化字段
	if len(config.InitialFields) > 0 {
		zapOpts = append(zapOpts, zap.Fields(config.InitialFields...))
	}

	zapLogger := zap.New(core, zapOpts...)

	// 只要设置了名称，就将其添加到日志记录器中
	if config.Name != "" {
		zapLogger = zapLogger.Named(config.Name)
	}

	return zapLogger
}

// parseLevel 将字符串级别转换为 zapcore.Level
func parseLevel(level string) zapcore.Level {
	switch strings.ToLower(level) {
	case DebugLevel:
		return zapcore.DebugLevel
	case InfoLevel:
		return zapcore.InfoLevel
	case WarnLevel:
		return zapcore.WarnLevel
	case ErrorLevel:
		return zapcore.ErrorLevel
	case FatalLevel:
		return zapcore.FatalLevel
	default:
		return zapcore.InfoLevel
	}
}
