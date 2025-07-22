package slogx

import (
	"fmt"
	"io"
	"os"

	"gopkg.in/natefinch/lumberjack.v2"
)

// FileRotationConfig configures log file rotation.
// Based on lumberjack.v2 for reliable file rotation.
type FileRotationConfig struct {
	// Filename is the file to write logs to.
	Filename string `json:"filename" yaml:"filename"`

	// MaxSize is the maximum size in megabytes of the log file before it gets rotated.
	// Default: 100 MB.
	MaxSize int `json:"maxSize" yaml:"maxSize"`

	// MaxAge is the maximum number of days to retain old log files.
	// Default: 30 days.
	MaxAge int `json:"maxAge" yaml:"maxAge"`

	// MaxBackups is the maximum number of old log files to retain.
	// Default: 10 files.
	MaxBackups int `json:"maxBackups" yaml:"maxBackups"`

	// LocalTime determines if the time used for formatting the timestamps
	// in backup files is the computer's local time. Default: false (UTC).
	LocalTime bool `json:"localTime" yaml:"localTime"`

	// Compress determines if the rotated log files should be compressed using gzip.
	// Default: false.
	Compress bool `json:"compress" yaml:"compress"`
}

// NewWriter creates an io.Writer based on the writer type and optional file rotation config.
// Supported writer types:
//   - "stdout": writes to os.Stdout
//   - "stderr": writes to os.Stderr
//   - "file": writes to a file with optional rotation via lumberjack
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

// newFileWriter creates a file writer with optional rotation.
// If fileRotation is nil, it will use default rotation settings.
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

// getMaxSize returns the max size with a reasonable default.
func getMaxSize(maxSize int) int {
	if maxSize <= 0 {
		return 100 // Default: 100 MB
	}
	return maxSize
}

// getMaxAge returns the max age with a reasonable default.
func getMaxAge(maxAge int) int {
	if maxAge <= 0 {
		return 30 // Default: 30 days
	}
	return maxAge
}

// getMaxBackups returns the max backups with a reasonable default.
func getMaxBackups(maxBackups int) int {
	if maxBackups <= 0 {
		return 10 // Default: 10 backup files
	}
	return maxBackups
}
