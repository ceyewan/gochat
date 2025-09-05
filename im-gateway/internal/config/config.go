package config

import (
	"time"

	"github.com/ceyewan/gochat/im-infra/coord"
	"github.com/ceyewan/gochat/im-infra/mq"
)

// Config 定义 im-gateway 服务的完整配置结构
type Config struct {
	// 服务器配置
	Server ServerConfig `yaml:"server" json:"server"`

	// JWT 认证配置
	JWT JWTConfig `yaml:"jwt" json:"jwt"`

	// gRPC 客户端配置
	GRPC GRPCConfig `yaml:"grpc" json:"grpc"`

	// Kafka 配置
	Kafka mq.Config `yaml:"kafka" json:"kafka"`

	// 服务发现配置
	Coordinator coord.CoordinatorConfig `yaml:"coordinator" json:"coordinator"`

	// 日志配置
	Log LogConfig `yaml:"log" json:"log"`
}

// ServerConfig 服务器相关配置
type ServerConfig struct {
	// HTTP 服务器监听地址
	HTTPAddr string `yaml:"http_addr" json:"http_addr"`

	// WebSocket 路径
	WSPath string `yaml:"ws_path" json:"ws_path"`

	// 读取超时
	ReadTimeout time.Duration `yaml:"read_timeout" json:"read_timeout"`

	// 写入超时
	WriteTimeout time.Duration `yaml:"write_timeout" json:"write_timeout"`

	// 空闲超时
	IdleTimeout time.Duration `yaml:"idle_timeout" json:"idle_timeout"`

	// WebSocket 配置
	WebSocket WebSocketConfig `yaml:"websocket" json:"websocket"`
}

// WebSocketConfig WebSocket 相关配置
type WebSocketConfig struct {
	// 读取缓冲区大小
	ReadBufferSize int `yaml:"read_buffer_size" json:"read_buffer_size"`

	// 写入缓冲区大小
	WriteBufferSize int `yaml:"write_buffer_size" json:"write_buffer_size"`

	// 心跳间隔
	PingInterval time.Duration `yaml:"ping_interval" json:"ping_interval"`

	// 心跳超时
	PongTimeout time.Duration `yaml:"pong_timeout" json:"pong_timeout"`

	// 写入超时
	WriteTimeout time.Duration `yaml:"write_timeout" json:"write_timeout"`

	// 最大消息大小
	MaxMessageSize int64 `yaml:"max_message_size" json:"max_message_size"`
}

// JWTConfig JWT 认证配置
type JWTConfig struct {
	// JWT 签名密钥
	Secret string `yaml:"secret" json:"secret"`

	// 访问令牌过期时间
	AccessTokenExpire time.Duration `yaml:"access_token_expire" json:"access_token_expire"`

	// 刷新令牌过期时间
	RefreshTokenExpire time.Duration `yaml:"refresh_token_expire" json:"refresh_token_expire"`

	// 发行者
	Issuer string `yaml:"issuer" json:"issuer"`
}

// GRPCConfig gRPC 客户端配置
type GRPCConfig struct {
	// im-logic 服务配置
	Logic GRPCServiceConfig `yaml:"logic" json:"logic"`

	// 连接超时
	ConnTimeout time.Duration `yaml:"conn_timeout" json:"conn_timeout"`

	// 请求超时
	RequestTimeout time.Duration `yaml:"request_timeout" json:"request_timeout"`

	// 重试配置
	Retry RetryConfig `yaml:"retry" json:"retry"`
}

// GRPCServiceConfig 单个 gRPC 服务配置
type GRPCServiceConfig struct {
	// 服务名称（用于服务发现）
	ServiceName string `yaml:"service_name" json:"service_name"`

	// 直连地址（可选，用于开发环境）
	DirectAddr string `yaml:"direct_addr" json:"direct_addr"`
}

// RetryConfig 重试配置
type RetryConfig struct {
	// 最大重试次数
	MaxRetries int `yaml:"max_retries" json:"max_retries"`

	// 初始延迟
	InitialDelay time.Duration `yaml:"initial_delay" json:"initial_delay"`

	// 最大延迟
	MaxDelay time.Duration `yaml:"max_delay" json:"max_delay"`

	// 退避因子
	BackoffFactor float64 `yaml:"backoff_factor" json:"backoff_factor"`
}

// LogConfig 日志配置
type LogConfig struct {
	// 日志级别
	Level string `yaml:"level" json:"level"`

	// 输出格式 (json/text)
	Format string `yaml:"format" json:"format"`

	// 输出目标 (stdout/file)
	Output string `yaml:"output" json:"output"`

	// 日志文件路径（当 output 为 file 时）
	FilePath string `yaml:"file_path" json:"file_path"`
}

// Load 加载配置文件
// 支持从配置中心、环境变量和配置文件加载
func Load() (*Config, error) {
	// TODO: 实现配置加载逻辑
	// 1. 从 etcd 配置中心加载
	// 2. 从环境变量覆盖
	// 3. 从本地配置文件加载默认值

	// 暂时返回默认配置
	return DefaultConfig(), nil
}

// DefaultConfig 返回默认配置
func DefaultConfig() *Config {
	return &Config{
		Server: ServerConfig{
			HTTPAddr:     ":8080",
			WSPath:       "/ws",
			ReadTimeout:  30 * time.Second,
			WriteTimeout: 30 * time.Second,
			IdleTimeout:  60 * time.Second,
			WebSocket: WebSocketConfig{
				ReadBufferSize:  1024,
				WriteBufferSize: 1024,
				PingInterval:    30 * time.Second,
				PongTimeout:     10 * time.Second,
				WriteTimeout:    10 * time.Second,
				MaxMessageSize:  1024 * 1024, // 1MB
			},
		},
		JWT: JWTConfig{
			Secret:             "your-secret-key",
			AccessTokenExpire:  24 * time.Hour,
			RefreshTokenExpire: 7 * 24 * time.Hour,
			Issuer:             "gochat-gateway",
		},
		GRPC: GRPCConfig{
			Logic: GRPCServiceConfig{
				ServiceName: "im-logic",
				DirectAddr:  "localhost:9001",
			},
			ConnTimeout:    5 * time.Second,
			RequestTimeout: 10 * time.Second,
			Retry: RetryConfig{
				MaxRetries:    3,
				InitialDelay:  100 * time.Millisecond,
				MaxDelay:      5 * time.Second,
				BackoffFactor: 2.0,
			},
		},
		Kafka: mq.DefaultConfig(),
		Coordinator: coord.CoordinatorConfig{
			Endpoints: []string{"localhost:2379"},
		},
		Log: LogConfig{
			Level:  "info",
			Format: "json",
			Output: "stdout",
		},
	}
}
