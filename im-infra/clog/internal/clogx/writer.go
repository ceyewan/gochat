package slogx

import (
	"fmt"
	"io"
	"os"

	"gopkg.in/natefinch/lumberjack.v2"
)

// FileRotationConfig 配置日志文件滚动。
// 基于 lumberjack.v2，可靠的文件滚动方案。
type FileRotationConfig struct {
	// Filename 日志写入的文件路径。
	Filename string `json:"filename" yaml:"filename"`

	// MaxSize 单个日志文件最大 MB，超过则滚动。
	// 默认：100 MB。
	MaxSize int `json:"maxSize" yaml:"maxSize"`

	// MaxAge 日志文件最大保存天数。
	// 默认：30 天。
	MaxAge int `json:"maxAge" yaml:"maxAge"`

	// MaxBackups 最大保留的旧日志文件数。
	// 默认：10 个文件。
	MaxBackups int `json:"maxBackups" yaml:"maxBackups"`

	// LocalTime 备份文件时间戳是否使用本地时间。默认：false（UTC）。
	LocalTime bool `json:"localTime" yaml:"localTime"`

	// Compress 滚动后的日志文件是否使用 gzip 压缩。默认：false。
	Compress bool `json:"compress" yaml:"compress"`
}

// NewWriter 根据 writer 类型和可选文件滚动配置创建 io.Writer。
// 支持的 writer 类型：
//   - "stdout"：写入 os.Stdout
//   - "stderr"：写入 os.Stderr
//   - "file"：写入文件并可选滚动（lumberjack 实现）
func NewWriter(writerType string, fileRotation *FileRotationConfig) (io.Writer, error) {
	switch writerType {
	case "stdout":
		return os.Stdout, nil
	case "stderr":
		return os.Stderr, nil
	case "file":
		return newFileWriter(fileRotation)
	default:
		return nil, fmt.Errorf("unsupported writer type: %s", writerType)
	}
}

// newFileWriter 创建带滚动功能的文件 writer。
// fileRotation 为 nil 时，使用默认滚动设置。
func newFileWriter(fileRotation *FileRotationConfig) (io.Writer, error) {
	if fileRotation == nil {
		return nil, fmt.Errorf("file rotation config is required for file writer")
	}

	if fileRotation.Filename == "" {
		return nil, fmt.Errorf("filename is required for file writer")
	}

	// Create lumberjack logger with rotation settings
	lumber := &lumberjack.Logger{
		Filename:   fileRotation.Filename,
		MaxSize:    getMaxSize(fileRotation.MaxSize),
		MaxAge:     getMaxAge(fileRotation.MaxAge),
		MaxBackups: getMaxBackups(fileRotation.MaxBackups),
		LocalTime:  fileRotation.LocalTime,
		Compress:   fileRotation.Compress,
	}

	return lumber, nil
}

// getMaxSize 返回合理默认值的最大文件大小。
func getMaxSize(maxSize int) int {
	if maxSize <= 0 {
		return 100 // Default: 100 MB
	}
	return maxSize
}

// getMaxAge 返回合理默认值的最大保存天数。
func getMaxAge(maxAge int) int {
	if maxAge <= 0 {
		return 30 // Default: 30 days
	}
	return maxAge
}

// getMaxBackups 返回合理默认值的最大备份文件数。
func getMaxBackups(maxBackups int) int {
	if maxBackups <= 0 {
		return 10 // Default: 10 backup files
	}
	return maxBackups
}
