package internal

import (
	"time"
)

// Config 是 mq 的主配置结构体。
// 用于声明式地定义Kafka连接和行为参数。
type Config struct {
	// Brokers Kafka broker地址列表
	// 格式：["host1:port1", "host2:port2"]
	// 默认：["localhost:9092"]
	Brokers []string `json:"brokers" yaml:"brokers"`

	// ClientID 客户端标识符
	// 默认："gochat-mq-client"
	ClientID string `json:"clientId" yaml:"clientId"`

	// SecurityProtocol 安全协议
	// 支持："PLAINTEXT", "SSL", "SASL_PLAINTEXT", "SASL_SSL"
	// 默认："PLAINTEXT"
	SecurityProtocol string `json:"securityProtocol" yaml:"securityProtocol"`

	// SASL配置
	SASL SASLConfig `json:"sasl" yaml:"sasl"`

	// SSL配置
	SSL SSLConfig `json:"ssl" yaml:"ssl"`

	// 连接配置
	Connection ConnectionConfig `json:"connection" yaml:"connection"`

	// 生产者配置
	ProducerConfig ProducerConfig `json:"producer" yaml:"producer"`

	// 消费者配置
	ConsumerConfig ConsumerConfig `json:"consumer" yaml:"consumer"`

	// 连接池配置
	PoolConfig PoolConfig `json:"pool" yaml:"pool"`

	// 性能配置
	Performance PerformanceConfig `json:"performance" yaml:"performance"`

	// 监控配置
	Monitoring MonitoringConfig `json:"monitoring" yaml:"monitoring"`
}

// SASLConfig SASL认证配置
type SASLConfig struct {
	// Mechanism SASL机制
	// 支持："PLAIN", "SCRAM-SHA-256", "SCRAM-SHA-512", "GSSAPI"
	// 默认：""（不使用SASL）
	Mechanism string `json:"mechanism" yaml:"mechanism"`

	// Username 用户名
	Username string `json:"username" yaml:"username"`

	// Password 密码
	Password string `json:"password" yaml:"password"`
}

// SSLConfig SSL/TLS配置
type SSLConfig struct {
	// Enabled 是否启用SSL
	// 默认：false
	Enabled bool `json:"enabled" yaml:"enabled"`

	// CertFile 客户端证书文件路径
	CertFile string `json:"certFile" yaml:"certFile"`

	// KeyFile 客户端私钥文件路径
	KeyFile string `json:"keyFile" yaml:"keyFile"`

	// CAFile CA证书文件路径
	CAFile string `json:"caFile" yaml:"caFile"`

	// InsecureSkipVerify 是否跳过证书验证
	// 默认：false
	InsecureSkipVerify bool `json:"insecureSkipVerify" yaml:"insecureSkipVerify"`
}

// ConnectionConfig 连接配置
type ConnectionConfig struct {
	// DialTimeout 连接超时时间
	// 默认：10秒
	DialTimeout time.Duration `json:"dialTimeout" yaml:"dialTimeout"`

	// ReadTimeout 读取超时时间
	// 默认：10秒
	ReadTimeout time.Duration `json:"readTimeout" yaml:"readTimeout"`

	// WriteTimeout 写入超时时间
	// 默认：10秒
	WriteTimeout time.Duration `json:"writeTimeout" yaml:"writeTimeout"`

	// KeepAlive TCP保活时间
	// 默认：7秒
	KeepAlive time.Duration `json:"keepAlive" yaml:"keepAlive"`

	// MaxRetries 最大重试次数
	// 默认：3
	MaxRetries int `json:"maxRetries" yaml:"maxRetries"`

	// RetryBackoff 重试间隔
	// 默认：100毫秒
	RetryBackoff time.Duration `json:"retryBackoff" yaml:"retryBackoff"`
}

// ProducerConfig 生产者配置
type ProducerConfig struct {
	// Brokers Kafka broker地址列表（继承自主配置）
	Brokers []string `json:"brokers" yaml:"brokers"`

	// ClientID 客户端标识符（继承自主配置）
	ClientID string `json:"clientId" yaml:"clientId"`

	// Compression 压缩算法
	// 支持："none", "gzip", "snappy", "lz4", "zstd"
	// 默认："lz4"（低延迟优化）
	Compression string `json:"compression" yaml:"compression"`

	// BatchSize 批次大小（字节）
	// 默认：16384（16KB）
	BatchSize int `json:"batchSize" yaml:"batchSize"`

	// LingerMs 批次等待时间（毫秒）
	// 默认：5毫秒（微秒级延迟优化）
	LingerMs int `json:"lingerMs" yaml:"lingerMs"`

	// MaxMessageBytes 单条消息最大大小（字节）
	// 默认：1048576（1MB）
	MaxMessageBytes int `json:"maxMessageBytes" yaml:"maxMessageBytes"`

	// RequiredAcks 确认级别
	// 0: 不等待确认
	// 1: 等待leader确认
	// -1: 等待所有副本确认
	// 默认：1
	RequiredAcks int `json:"requiredAcks" yaml:"requiredAcks"`

	// RequestTimeout 请求超时时间
	// 默认：30秒
	RequestTimeout time.Duration `json:"requestTimeout" yaml:"requestTimeout"`

	// EnableIdempotence 是否启用幂等性
	// 默认：true
	EnableIdempotence bool `json:"enableIdempotence" yaml:"enableIdempotence"`

	// MaxInFlightRequests 最大飞行中请求数
	// 默认：5
	MaxInFlightRequests int `json:"maxInFlightRequests" yaml:"maxInFlightRequests"`

	// RetryBackoff 重试间隔
	// 默认：100毫秒
	RetryBackoff time.Duration `json:"retryBackoff" yaml:"retryBackoff"`

	// MaxRetries 最大重试次数
	// 默认：3
	MaxRetries int `json:"maxRetries" yaml:"maxRetries"`
}

// ConsumerConfig 消费者配置
type ConsumerConfig struct {
	// Brokers Kafka broker地址列表（继承自主配置）
	Brokers []string `json:"brokers" yaml:"brokers"`

	// ClientID 客户端标识符（继承自主配置）
	ClientID string `json:"clientId" yaml:"clientId"`

	// GroupID 消费者组ID
	// 默认：""（必须设置）
	GroupID string `json:"groupId" yaml:"groupId"`

	// AutoOffsetReset 自动偏移量重置策略
	// 支持："earliest", "latest", "none"
	// 默认："latest"
	AutoOffsetReset string `json:"autoOffsetReset" yaml:"autoOffsetReset"`

	// EnableAutoCommit 是否启用自动提交偏移量
	// 默认：true
	EnableAutoCommit bool `json:"enableAutoCommit" yaml:"enableAutoCommit"`

	// AutoCommitInterval 自动提交间隔
	// 默认：5秒
	AutoCommitInterval time.Duration `json:"autoCommitInterval" yaml:"autoCommitInterval"`

	// SessionTimeout 会话超时时间
	// 默认：10秒
	SessionTimeout time.Duration `json:"sessionTimeout" yaml:"sessionTimeout"`

	// HeartbeatInterval 心跳间隔
	// 默认：3秒
	HeartbeatInterval time.Duration `json:"heartbeatInterval" yaml:"heartbeatInterval"`

	// MaxPollRecords 单次拉取最大记录数
	// 默认：500
	MaxPollRecords int `json:"maxPollRecords" yaml:"maxPollRecords"`

	// MaxPollInterval 最大拉取间隔
	// 默认：5分钟
	MaxPollInterval time.Duration `json:"maxPollInterval" yaml:"maxPollInterval"`

	// FetchMinBytes 拉取最小字节数
	// 默认：1
	FetchMinBytes int `json:"fetchMinBytes" yaml:"fetchMinBytes"`

	// FetchMaxBytes 拉取最大字节数
	// 默认：52428800（50MB）
	FetchMaxBytes int `json:"fetchMaxBytes" yaml:"fetchMaxBytes"`

	// FetchMaxWait 拉取最大等待时间
	// 默认：500毫秒
	FetchMaxWait time.Duration `json:"fetchMaxWait" yaml:"fetchMaxWait"`

	// IsolationLevel 隔离级别
	// 支持："read_uncommitted", "read_committed"
	// 默认："read_uncommitted"
	IsolationLevel string `json:"isolationLevel" yaml:"isolationLevel"`
}

// PoolConfig 连接池配置
type PoolConfig struct {
	// MaxConnections 最大连接数
	// 默认：10
	MaxConnections int `json:"maxConnections" yaml:"maxConnections"`

	// MinIdleConnections 最小空闲连接数
	// 默认：2
	MinIdleConnections int `json:"minIdleConnections" yaml:"minIdleConnections"`

	// MaxIdleConnections 最大空闲连接数
	// 默认：5
	MaxIdleConnections int `json:"maxIdleConnections" yaml:"maxIdleConnections"`

	// ConnectionMaxLifetime 连接最大生存时间
	// 默认：1小时
	ConnectionMaxLifetime time.Duration `json:"connectionMaxLifetime" yaml:"connectionMaxLifetime"`

	// ConnectionMaxIdleTime 连接最大空闲时间
	// 默认：30分钟
	ConnectionMaxIdleTime time.Duration `json:"connectionMaxIdleTime" yaml:"connectionMaxIdleTime"`

	// HealthCheckInterval 健康检查间隔
	// 默认：30秒
	HealthCheckInterval time.Duration `json:"healthCheckInterval" yaml:"healthCheckInterval"`
}

// PerformanceConfig 性能配置
type PerformanceConfig struct {
	// TargetLatencyMicros 目标延迟（微秒）
	// 默认：1000（1毫秒）
	TargetLatencyMicros int `json:"targetLatencyMicros" yaml:"targetLatencyMicros"`

	// TargetThroughputPerSec 目标吞吐量（消息/秒）
	// 默认：100000
	TargetThroughputPerSec int `json:"targetThroughputPerSec" yaml:"targetThroughputPerSec"`

	// OptimizeForSmallMessages 是否优化小消息处理
	// 默认：true
	OptimizeForSmallMessages bool `json:"optimizeForSmallMessages" yaml:"optimizeForSmallMessages"`

	// SmallMessageThresholdBytes 小消息阈值（字节）
	// 默认：1024（1KB）
	SmallMessageThresholdBytes int `json:"smallMessageThresholdBytes" yaml:"smallMessageThresholdBytes"`
}

// MonitoringConfig 监控配置
type MonitoringConfig struct {
	// EnableMetrics 是否启用指标收集
	// 默认：true
	EnableMetrics bool `json:"enableMetrics" yaml:"enableMetrics"`

	// MetricsInterval 指标收集间隔
	// 默认：10秒
	MetricsInterval time.Duration `json:"metricsInterval" yaml:"metricsInterval"`

	// EnableTracing 是否启用链路追踪
	// 默认：false
	EnableTracing bool `json:"enableTracing" yaml:"enableTracing"`

	// LogLevel 日志级别
	// 支持："debug", "info", "warn", "error"
	// 默认："info"
	LogLevel string `json:"logLevel" yaml:"logLevel"`
}

// DefaultConfig 返回一个带有合理默认值的 Config。
// 默认配置适用于大多数开发和测试场景，针对即时通讯场景优化。
func DefaultConfig() Config {
	return Config{
		Brokers:          []string{"localhost:9092"},
		ClientID:         "gochat-mq-client",
		SecurityProtocol: "PLAINTEXT",
		SASL:             DefaultSASLConfig(),
		SSL:              DefaultSSLConfig(),
		Connection:       DefaultConnectionConfig(),
		ProducerConfig:   DefaultProducerConfig(),
		ConsumerConfig:   DefaultConsumerConfig(),
		PoolConfig:       DefaultPoolConfig(),
		Performance:      DefaultPerformanceConfig(),
		Monitoring:       DefaultMonitoringConfig(),
	}
}

// DefaultSASLConfig 返回默认的SASL配置
func DefaultSASLConfig() SASLConfig {
	return SASLConfig{
		Mechanism: "",
		Username:  "",
		Password:  "",
	}
}

// DefaultSSLConfig 返回默认的SSL配置
func DefaultSSLConfig() SSLConfig {
	return SSLConfig{
		Enabled:            false,
		CertFile:           "",
		KeyFile:            "",
		CAFile:             "",
		InsecureSkipVerify: false,
	}
}

// DefaultConnectionConfig 返回默认的连接配置
func DefaultConnectionConfig() ConnectionConfig {
	return ConnectionConfig{
		DialTimeout:  10 * time.Second,
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 10 * time.Second,
		KeepAlive:    7 * time.Second,
		MaxRetries:   3,
		RetryBackoff: 100 * time.Millisecond,
	}
}

// DefaultProducerConfig 返回默认的生产者配置
// 针对即时通讯场景优化：低延迟、高吞吐量、小消息优化
func DefaultProducerConfig() ProducerConfig {
	return ProducerConfig{
		Brokers:             []string{"localhost:9092"},
		ClientID:            "gochat-producer",
		Compression:         "lz4",   // LZ4压缩，低延迟
		BatchSize:           16384,   // 16KB批次大小
		LingerMs:            5,       // 5毫秒等待时间，微秒级延迟优化
		MaxMessageBytes:     1048576, // 1MB最大消息大小
		RequiredAcks:        1,       // 等待leader确认
		RequestTimeout:      30 * time.Second,
		EnableIdempotence:   true, // 启用幂等性保证
		MaxInFlightRequests: 5,
		RetryBackoff:        100 * time.Millisecond,
		MaxRetries:          3,
	}
}

// DefaultConsumerConfig 返回默认的消费者配置
// 针对即时通讯场景优化：高吞吐量、低延迟
func DefaultConsumerConfig() ConsumerConfig {
	return ConsumerConfig{
		Brokers:            []string{"localhost:9092"},
		ClientID:           "gochat-consumer",
		GroupID:            "", // 必须由用户设置
		AutoOffsetReset:    "latest",
		EnableAutoCommit:   true,
		AutoCommitInterval: 5 * time.Second,
		SessionTimeout:     10 * time.Second,
		HeartbeatInterval:  3 * time.Second,
		MaxPollRecords:     500, // 支持高吞吐量
		MaxPollInterval:    5 * time.Minute,
		FetchMinBytes:      1,
		FetchMaxBytes:      52428800, // 50MB
		FetchMaxWait:       500 * time.Millisecond,
		IsolationLevel:     "read_uncommitted",
	}
}

// DefaultPoolConfig 返回默认的连接池配置
func DefaultPoolConfig() PoolConfig {
	return PoolConfig{
		MaxConnections:        10,
		MinIdleConnections:    2,
		MaxIdleConnections:    5,
		ConnectionMaxLifetime: time.Hour,
		ConnectionMaxIdleTime: 30 * time.Minute,
		HealthCheckInterval:   30 * time.Second,
	}
}

// DefaultPerformanceConfig 返回默认的性能配置
// 针对即时通讯场景优化
func DefaultPerformanceConfig() PerformanceConfig {
	return PerformanceConfig{
		TargetLatencyMicros:        1000,   // 1毫秒目标延迟
		TargetThroughputPerSec:     100000, // 10万消息/秒目标吞吐量
		OptimizeForSmallMessages:   true,
		SmallMessageThresholdBytes: 1024, // 1KB小消息阈值
	}
}

// DefaultMonitoringConfig 返回默认的监控配置
func DefaultMonitoringConfig() MonitoringConfig {
	return MonitoringConfig{
		EnableMetrics:   true,
		MetricsInterval: 10 * time.Second,
		EnableTracing:   false,
		LogLevel:        "info",
	}
}
