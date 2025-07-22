package internal

// Config is the main configuration structure for xlog.
// It provides a declarative way to define logging behavior.
type Config struct {
	// Level controls the minimum log level for output.
	// Supported values: "debug", "info", "warn", "error". Default: "info".
	Level string `json:"level" yaml:"level"`

	// Outputs defines one or more log output destinations.
	// Each output can have different formats and writers.
	Outputs []OutputConfig `json:"outputs" yaml:"outputs"`

	// EnableTraceID controls whether to automatically inject TraceID from context.Context.
	// When enabled, the logger will look for TraceID in the context and add it to log records.
	EnableTraceID bool `json:"enableTraceID" yaml:"enableTraceID"`

	// TraceIDKey is the key used to extract TraceID from context.
	// Default: "traceID".
	TraceIDKey any `json:"traceIDKey" yaml:"traceIDKey"`

	// AddSource controls whether to include source file name and line number in logs.
	// When enabled, logs will include source information for debugging.
	AddSource bool `json:"addSource" yaml:"addSource"`
}

// OutputConfig defines the configuration for a single output destination.
type OutputConfig struct {
	// Format defines the log format.
	// Supported values: "json", "text".
	Format string `json:"format" yaml:"format"`

	// Writer defines where logs are written.
	// Supported values: "stdout", "stderr", "file".
	Writer string `json:"writer" yaml:"writer"`

	// FileRotation provides log rotation configuration for "file" writers.
	// This configuration is ignored when Writer is not "file".
	FileRotation *FileRotationConfig `json:"fileRotation,omitempty" yaml:"fileRotation,omitempty"`
}

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

// DefaultConfig returns a Config with reasonable defaults.
// Default configuration:
//   - Level: "info"
//   - Format: "text"
//   - Writer: "stdout"
//   - TraceID: disabled
//   - AddSource: false
func DefaultConfig() Config {
	return Config{
		Level: "info",
		Outputs: []OutputConfig{
			{
				Format: "text",
				Writer: "stdout",
			},
		},
		EnableTraceID: false,
		TraceIDKey:    "traceID",
		AddSource:     false,
	}
}
