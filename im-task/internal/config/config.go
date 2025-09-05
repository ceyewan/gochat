package config

import (
	"time"

	"github.com/ceyewan/gochat/im-infra/coord"
	"github.com/ceyewan/gochat/im-infra/mq"
)

// Config 定义 im-task 服务的完整配置结构
type Config struct {
	// 服务器配置
	Server ServerConfig `yaml:"server" json:"server"`

	// gRPC 客户端配置
	GRPC GRPCConfig `yaml:"grpc" json:"grpc"`

	// Kafka 配置
	Kafka mq.Config `yaml:"kafka" json:"kafka"`

	// 服务发现配置
	Coordinator coord.CoordinatorConfig `yaml:"coordinator" json:"coordinator"`

	// 任务处理配置
	TaskProcessor TaskProcessorConfig `yaml:"task_processor" json:"task_processor"`

	// 外部服务配置
	ExternalServices ExternalServicesConfig `yaml:"external_services" json:"external_services"`

	// 日志配置
	Log LogConfig `yaml:"log" json:"log"`
}

// ServerConfig 服务器相关配置
type ServerConfig struct {
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

// TaskProcessorConfig 任务处理器配置
type TaskProcessorConfig struct {
	// 工作协程数量
	WorkerCount int `yaml:"worker_count" json:"worker_count"`

	// 任务队列缓冲区大小
	QueueBufferSize int `yaml:"queue_buffer_size" json:"queue_buffer_size"`

	// 任务超时时间
	TaskTimeout time.Duration `yaml:"task_timeout" json:"task_timeout"`

	// 最大重试次数
	MaxRetries int `yaml:"max_retries" json:"max_retries"`

	// 重试间隔
	RetryInterval time.Duration `yaml:"retry_interval" json:"retry_interval"`

	// 批处理配置
	Batch BatchConfig `yaml:"batch" json:"batch"`
}

// BatchConfig 批处理配置
type BatchConfig struct {
	// 批处理大小
	Size int `yaml:"size" json:"size"`

	// 批处理超时
	Timeout time.Duration `yaml:"timeout" json:"timeout"`

	// 并发批次数
	ConcurrentBatches int `yaml:"concurrent_batches" json:"concurrent_batches"`
}

// ExternalServicesConfig 外部服务配置
type ExternalServicesConfig struct {
	// 推送服务配置
	Push PushServiceConfig `yaml:"push" json:"push"`

	// 内容审核服务配置
	Audit AuditServiceConfig `yaml:"audit" json:"audit"`

	// Elasticsearch 配置
	Elasticsearch ElasticsearchConfig `yaml:"elasticsearch" json:"elasticsearch"`
}

// PushServiceConfig 推送服务配置
type PushServiceConfig struct {
	// APNs 配置
	APNs APNsConfig `yaml:"apns" json:"apns"`

	// FCM 配置
	FCM FCMConfig `yaml:"fcm" json:"fcm"`

	// 推送超时
	Timeout time.Duration `yaml:"timeout" json:"timeout"`

	// 最大重试次数
	MaxRetries int `yaml:"max_retries" json:"max_retries"`
}

// APNsConfig Apple Push Notification 配置
type APNsConfig struct {
	// 是否启用
	Enabled bool `yaml:"enabled" json:"enabled"`

	// 证书文件路径
	CertFile string `yaml:"cert_file" json:"cert_file"`

	// 私钥文件路径
	KeyFile string `yaml:"key_file" json:"key_file"`

	// Bundle ID
	BundleID string `yaml:"bundle_id" json:"bundle_id"`

	// 是否为生产环境
	Production bool `yaml:"production" json:"production"`
}

// FCMConfig Firebase Cloud Messaging 配置
type FCMConfig struct {
	// 是否启用
	Enabled bool `yaml:"enabled" json:"enabled"`

	// 服务账号密钥文件路径
	ServiceAccountFile string `yaml:"service_account_file" json:"service_account_file"`

	// 项目 ID
	ProjectID string `yaml:"project_id" json:"project_id"`
}

// AuditServiceConfig 内容审核服务配置
type AuditServiceConfig struct {
	// 是否启用
	Enabled bool `yaml:"enabled" json:"enabled"`

	// 服务 URL
	URL string `yaml:"url" json:"url"`

	// API 密钥
	APIKey string `yaml:"api_key" json:"api_key"`

	// 超时时间
	Timeout time.Duration `yaml:"timeout" json:"timeout"`
}

// ElasticsearchConfig Elasticsearch 配置
type ElasticsearchConfig struct {
	// 是否启用
	Enabled bool `yaml:"enabled" json:"enabled"`

	// 服务器地址列表
	Addresses []string `yaml:"addresses" json:"addresses"`

	// 用户名
	Username string `yaml:"username" json:"username"`

	// 密码
	Password string `yaml:"password" json:"password"`

	// 索引前缀
	IndexPrefix string `yaml:"index_prefix" json:"index_prefix"`

	// 批量索引大小
	BulkSize int `yaml:"bulk_size" json:"bulk_size"`

	// 批量索引超时
	BulkTimeout time.Duration `yaml:"bulk_timeout" json:"bulk_timeout"`
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
			ServiceName: "im-task",
			Version:     "1.0.0",
			HealthPort:  ":8003",
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
		TaskProcessor: TaskProcessorConfig{
			WorkerCount:     10,
			QueueBufferSize: 1000,
			TaskTimeout:     5 * time.Minute,
			MaxRetries:      3,
			RetryInterval:   30 * time.Second,
			Batch: BatchConfig{
				Size:              200,
				Timeout:           10 * time.Second,
				ConcurrentBatches: 5,
			},
		},
		ExternalServices: ExternalServicesConfig{
			Push: PushServiceConfig{
				APNs: APNsConfig{
					Enabled:    false,
					Production: false,
				},
				FCM: FCMConfig{
					Enabled: false,
				},
				Timeout:    30 * time.Second,
				MaxRetries: 3,
			},
			Audit: AuditServiceConfig{
				Enabled: false,
				Timeout: 10 * time.Second,
			},
			Elasticsearch: ElasticsearchConfig{
				Enabled:     false,
				Addresses:   []string{"http://localhost:9200"},
				IndexPrefix: "gochat",
				BulkSize:    100,
				BulkTimeout: 10 * time.Second,
			},
		},
		Log: LogConfig{
			Level:  "info",
			Format: "json",
			Output: "stdout",
		},
	}
}
