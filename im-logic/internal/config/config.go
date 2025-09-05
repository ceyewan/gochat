package config

import (
	"time"

	"github.com/ceyewan/gochat/im-infra/coord"
	"github.com/ceyewan/gochat/im-infra/mq"
)

// Config 定义 im-logic 服务的完整配置结构
type Config struct {
	// 服务器配置
	Server ServerConfig `yaml:"server" json:"server"`

	// gRPC 客户端配置
	GRPC GRPCConfig `yaml:"grpc" json:"grpc"`

	// Kafka 配置
	Kafka mq.Config `yaml:"kafka" json:"kafka"`

	// 服务发现配置
	Coordinator coord.CoordinatorConfig `yaml:"coordinator" json:"coordinator"`

	// 业务配置
	Business BusinessConfig `yaml:"business" json:"business"`

	// 日志配置
	Log LogConfig `yaml:"log" json:"log"`
}

// ServerConfig 服务器相关配置
type ServerConfig struct {
	// gRPC 服务器监听地址
	GRPCAddr string `yaml:"grpc_addr" json:"grpc_addr"`

	// 服务名称（用于服务注册）
	ServiceName string `yaml:"service_name" json:"service_name"`

	// 服务版本
	Version string `yaml:"version" json:"version"`

	// 健康检查端口
	HealthPort string `yaml:"health_port" json:"health_port"`
}

// GRPCConfig gRPC 客户端配置
type GRPCConfig struct {
	// im-repo 服务配置
	Repo GRPCServiceConfig `yaml:"repo" json:"repo"`

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

// BusinessConfig 业务相关配置
type BusinessConfig struct {
	// 消息分发配置
	MessageDistribution MessageDistributionConfig `yaml:"message_distribution" json:"message_distribution"`

	// JWT 配置
	JWT JWTConfig `yaml:"jwt" json:"jwt"`
}

// MessageDistributionConfig 消息分发配置
type MessageDistributionConfig struct {
	// 大群阈值（超过此数量的群组使用异步分发）
	LargeGroupThreshold int `yaml:"large_group_threshold" json:"large_group_threshold"`

	// 批处理大小
	BatchSize int `yaml:"batch_size" json:"batch_size"`

	// 分发超时
	DistributionTimeout time.Duration `yaml:"distribution_timeout" json:"distribution_timeout"`
}

// JWTConfig JWT 配置
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
			GRPCAddr:    ":9001",
			ServiceName: "im-logic",
			Version:     "1.0.0",
			HealthPort:  ":8001",
		},
		GRPC: GRPCConfig{
			Repo: GRPCServiceConfig{
				ServiceName: "im-repo",
				DirectAddr:  "localhost:9002",
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
		Business: BusinessConfig{
			MessageDistribution: MessageDistributionConfig{
				LargeGroupThreshold: 500,
				BatchSize:           200,
				DistributionTimeout: 30 * time.Second,
			},
			JWT: JWTConfig{
				Secret:             "your-secret-key",
				AccessTokenExpire:  24 * time.Hour,
				RefreshTokenExpire: 7 * 24 * time.Hour,
				Issuer:             "gochat-logic",
			},
		},
		Log: LogConfig{
			Level:  "info",
			Format: "json",
			Output: "stdout",
		},
	}
}
