package internal

import (
	"context"
	"os"
	"path/filepath"
	"reflect"
	"strings"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"gopkg.in/natefinch/lumberjack.v2"
)

// Hook 定义 context 钩子函数类型
type Hook func(context.Context) (string, bool)

// Logger 定义日志接口
type Logger interface {
	Debug(msg string, fields ...zap.Field)
	Info(msg string, fields ...zap.Field)
	Warn(msg string, fields ...zap.Field)
	Error(msg string, fields ...zap.Field)
	Fatal(msg string, fields ...zap.Field)

	With(fields ...zap.Field) Logger
	WithOptions(opts ...zap.Option) Logger
	Module(name string) Logger
}

// zapLogger 封装 zap.Logger
type zapLogger struct {
	*zap.Logger
	hook Hook
}

// Option 定义配置选项
type Option func(*options)

type options struct {
	hook Hook
}

// WithHook 设置 context 钩子
func WithHook(hook Hook) Option {
	return func(o *options) {
		o.hook = hook
	}
}

// rotationConfig 日志轮转配置
type rotationConfig struct {
	MaxSize    int
	MaxBackups int
	MaxAge     int
	Compress   bool
}

// config 内部配置结构，避免循环依赖
type config struct {
	Level       string
	Format      string
	Output      string
	AddSource   bool
	EnableColor bool
	RootPath    string
	Rotation    *rotationConfig
}

// NewLogger 创建新的 logger
func NewLogger(cfg interface{}, opts ...Option) (Logger, error) {
	// 解析选项
	var opt options
	for _, o := range opts {
		o(&opt)
	}

	// 类型断言获取配置
	config := parseConfig(cfg)

	// 创建 zap 配置
	zapConfig := zap.Config{
		Level:            zap.NewAtomicLevelAt(parseLevel(config.Level)),
		Encoding:         config.Format,
		OutputPaths:      []string{config.Output},
		ErrorOutputPaths: []string{"stderr"},
		EncoderConfig:    buildEncoderConfig(config.Format, config.EnableColor, config.RootPath, config.AddSource),
	}

	// 处理文件输出
	if config.Output != "stdout" && config.Output != "stderr" {
		if err := ensureDir(config.Output); err != nil {
			return nil, err
		}

		// 如果需要轮转，使用自定义的文件写入器
		if config.Rotation != nil {
			// 对于轮转文件，我们需要使用自定义的核心
			return buildLoggerWithRotation(config, opt.hook)
		}
	}

	// 构建 logger
	buildOptions := []zap.Option{
		zap.AddStacktrace(zapcore.ErrorLevel),
	}
	if config.AddSource {
		// 只添加 AddCaller，不设置固定的 CallerSkip
		buildOptions = append(buildOptions, zap.AddCaller())
	}

	baseLogger, err := zapConfig.Build(buildOptions...)
	if err != nil {
		return nil, err
	}

	return &zapLogger{
		Logger: baseLogger,
		hook:   opt.hook,
	}, nil
}

// NewFallbackLogger 创建备用 logger
func NewFallbackLogger() Logger {
	logger, _ := zap.NewProduction()
	return &zapLogger{Logger: logger}
}

// With 添加字段
func (l *zapLogger) With(fields ...zap.Field) Logger {
	return &zapLogger{
		Logger: l.Logger.With(fields...),
		hook:   l.hook,
	}
}

// WithOptions 添加选项
func (l *zapLogger) WithOptions(opts ...zap.Option) Logger {
	return &zapLogger{
		Logger: l.Logger.WithOptions(opts...),
		hook:   l.hook,
	}
}

// Fatal 记录 Fatal 级别的日志并退出程序
func (l *zapLogger) Fatal(msg string, fields ...zap.Field) {
	l.Logger.Fatal(msg, fields...)
	os.Exit(1)
}

// Module 创建模块日志器 - 只能基于默认 logger，不支持嵌套
func (l *zapLogger) Module(name string) Logger {
	return l.With(zap.String("module", name))
}

// parseConfig 解析配置
func parseConfig(cfg interface{}) *config {
	// 使用反射来解析配置，避免循环依赖
	if cfg == nil {
		return getDefaultConfig()
	}

	// 尝试使用反射获取字段值
	config := &config{
		Level:       getStringField(cfg, "Level", "info"),
		Format:      getStringField(cfg, "Format", "json"),
		Output:      getStringField(cfg, "Output", "stdout"),
		AddSource:   getBoolField(cfg, "AddSource", true),
		EnableColor: getBoolField(cfg, "EnableColor", false),
		RootPath:    getStringField(cfg, "RootPath", ""),
	}

	// 处理轮转配置
	if rotationField := getField(cfg, "Rotation"); rotationField != nil {
		config.Rotation = &rotationConfig{
			MaxSize:    getIntField(rotationField, "MaxSize", 100),
			MaxBackups: getIntField(rotationField, "MaxBackups", 3),
			MaxAge:     getIntField(rotationField, "MaxAge", 7),
			Compress:   getBoolField(rotationField, "Compress", false),
		}
	}

	return config
}

// getDefaultConfig 返回默认配置
func getDefaultConfig() *config {
	return &config{
		Level:       "info",
		Format:      "json",
		Output:      "stdout",
		AddSource:   true,
		EnableColor: false,
		RootPath:    "",
	}
}

// parseLevel 解析日志级别
func parseLevel(level string) zapcore.Level {
	switch strings.ToLower(level) {
	case "debug":
		return zapcore.DebugLevel
	case "info":
		return zapcore.InfoLevel
	case "warn":
		return zapcore.WarnLevel
	case "error":
		return zapcore.ErrorLevel
	default:
		return zapcore.InfoLevel
	}
}

// buildLoggerWithRotation 构建带轮转的日志器
func buildLoggerWithRotation(config *config, hook Hook) (Logger, error) {
	// 创建编码器
	encoderConfig := buildEncoderConfig(config.Format, config.EnableColor, config.RootPath, config.AddSource)
	encoder := createEncoder(config.Format, encoderConfig)

	// 创建轮转写入器
	rotatingWriter := &lumberjack.Logger{
		Filename:   config.Output,
		MaxSize:    config.Rotation.MaxSize,
		MaxBackups: config.Rotation.MaxBackups,
		MaxAge:     config.Rotation.MaxAge,
		Compress:   config.Rotation.Compress,
		LocalTime:  true,
	}

	// 创建核心
	core := zapcore.NewCore(
		encoder,
		zapcore.AddSync(rotatingWriter),
		parseLevel(config.Level),
	)

	// 构建选项
	opts := []zap.Option{
		zap.AddStacktrace(zapcore.ErrorLevel),
	}

	if config.AddSource {
		// 只添加 AddCaller，不设置固定的 CallerSkip
		opts = append(opts, zap.AddCaller())
	}

	// 创建 logger
	logger := zap.New(core, opts...)

	return &zapLogger{
		Logger: logger,
		hook:   hook,
	}, nil
}

func ensureDir(filename string) error {
	dir := filepath.Dir(filename)
	return os.MkdirAll(dir, 0755)
}

// 反射辅助函数
func getField(obj interface{}, fieldName string) interface{} {
	if obj == nil {
		return nil
	}

	v := reflect.ValueOf(obj)
	if v.Kind() == reflect.Ptr {
		v = v.Elem()
	}

	if v.Kind() != reflect.Struct {
		return nil
	}

	field := v.FieldByName(fieldName)
	if !field.IsValid() {
		return nil
	}

	return field.Interface()
}

func getStringField(obj interface{}, fieldName, defaultValue string) string {
	field := getField(obj, fieldName)
	if field == nil {
		return defaultValue
	}

	if str, ok := field.(string); ok {
		return str
	}

	return defaultValue
}

func getBoolField(obj interface{}, fieldName string, defaultValue bool) bool {
	field := getField(obj, fieldName)
	if field == nil {
		return defaultValue
	}

	if b, ok := field.(bool); ok {
		return b
	}

	return defaultValue
}

func getIntField(obj interface{}, fieldName string, defaultValue int) int {
	field := getField(obj, fieldName)
	if field == nil {
		return defaultValue
	}

	if i, ok := field.(int); ok {
		return i
	}

	return defaultValue
}
