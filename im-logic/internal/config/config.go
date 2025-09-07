package config

import (
	"fmt"
	"time"

	"github.com/spf13/viper"
)

// Config 配置结构体
type Config struct {
	Server     ServerConfig     `yaml:"server"`
	Logging    LoggingConfig    `yaml:"logging"`
	Database   DatabaseConfig   `yaml:"database"`
	Redis      RedisConfig      `yaml:"redis"`
	Kafka      KafkaConfig      `yaml:"kafka"`
	JWT        JWTConfig        `yaml:"jwt"`
	Repo       RepoConfig       `yaml:"repo"`
	Message    MessageConfig    `yaml:"message"`
	Group      GroupConfig      `yaml:"group"`
	Monitoring MonitoringConfig `yaml:"monitoring"`
	Discovery  DiscoveryConfig  `yaml:"discovery"`
}

// ServerConfig 服务器配置
type ServerConfig struct {
	GRPC GRPCServerConfig `yaml:"grpc"`
	HTTP HTTPServerConfig `yaml:"http"`
}

// GRPCServerConfig gRPC 服务器配置
type GRPCServerConfig struct {
	Host       string `yaml:"host"`
	Port       int    `yaml:"port"`
	MaxConn    int    `yaml:"max_conn"`
	MaxMsgSize int    `yaml:"max_msg_size"`
	Timeout    int    `yaml:"timeout"`
}

// HTTPServerConfig HTTP 服务器配置
type HTTPServerConfig struct {
	Host    string `yaml:"host"`
	Port    int    `yaml:"port"`
	Timeout int    `yaml:"timeout"`
}

// LoggingConfig 日志配置
type LoggingConfig struct {
	Level      string `yaml:"level"`
	Format     string `yaml:"format"`
	FilePath   string `yaml:"file_path"`
	MaxSize    int    `yaml:"max_size"`
	MaxAge     int    `yaml:"max_age"`
	MaxBackups int    `yaml:"max_backups"`
	Compress   bool   `yaml:"compress"`
}

// DatabaseConfig 数据库配置
type DatabaseConfig struct {
	Type            string `yaml:"type"`
	Host            string `yaml:"host"`
	Port            int    `yaml:"port"`
	Name            string `yaml:"name"`
	Username        string `yaml:"username"`
	Password        string `yaml:"password"`
	MaxConn         int    `yaml:"max_conn"`
	MaxIdleConn     int    `yaml:"max_idle_conn"`
	ConnMaxLifetime int    `yaml:"conn_max_lifetime"`
	ConnMaxIdleTime int    `yaml:"conn_max_idle_time"`
}

// RedisConfig Redis 配置
type RedisConfig struct {
	Addr         string `yaml:"addr"`
	Password     string `yaml:"password"`
	DB           int    `yaml:"db"`
	PoolSize     int    `yaml:"pool_size"`
	MinIdleConns int    `yaml:"min_idle_conns"`
	DialTimeout  int    `yaml:"dial_timeout"`
	ReadTimeout  int    `yaml:"read_timeout"`
	WriteTimeout int    `yaml:"write_timeout"`
}

// KafkaConfig Kafka 配置
type KafkaConfig struct {
	Brokers               []string            `yaml:"brokers"`
	UpstreamTopic         string              `yaml:"upstream_topic"`
	DownstreamTopicPrefix string              `yaml:"downstream_topic_prefix"`
	TaskTopic             string              `yaml:"task_topic"`
	ConsumerGroup         string              `yaml:"consumer_group"`
	BatchSize             int                 `yaml:"batch_size"`
	BatchTimeout          int                 `yaml:"batch_timeout"`
	SessionTimeout        int                 `yaml:"session_timeout"`
	HeartbeatInterval     int                 `yaml:"heartbeat_interval"`
	Retry                 KafkaRetryConfig    `yaml:"retry"`
	Producer              KafkaProducerConfig `yaml:"producer"`
}

// KafkaRetryConfig Kafka 重试配置
type KafkaRetryConfig struct {
	MaxRetries    int `yaml:"max_retries"`
	RetryInterval int `yaml:"retry_interval"`
	BackoffFactor int `yaml:"backoff_factor"`
}

// KafkaProducerConfig Kafka 生产者配置
type KafkaProducerConfig struct {
	Acks         string `yaml:"acks"`
	Retries      int    `yaml:"retries"`
	RetryBackoff int    `yaml:"retry_backoff"`
	BatchSize    int    `yaml:"batch_size"`
	BatchTimeout int    `yaml:"batch_timeout"`
	Compression  string `yaml:"compression"`
}

// JWTConfig JWT 配置
type JWTConfig struct {
	Secret             string `yaml:"secret"`
	AccessTokenExpire  int    `yaml:"access_token_expire"`
	RefreshTokenExpire int    `yaml:"refresh_token_expire"`
	SigningMethod      string `yaml:"signing_method"`
}

// RepoConfig 数据仓储服务配置
type RepoConfig struct {
	GRPC RepoGRPCConfig `yaml:"grpc"`
}

// RepoGRPCConfig 数据仓储 gRPC 配置
type RepoGRPCConfig struct {
	Address       string `yaml:"address"`
	Timeout       int    `yaml:"timeout"`
	MaxRetries    int    `yaml:"max_retries"`
	RetryInterval int    `yaml:"retry_interval"`
	EnableTLS     bool   `yaml:"enable_tls"`
	CACert        string `yaml:"ca_cert"`
	ClientCert    string `yaml:"client_cert"`
	ClientKey     string `yaml:"client_key"`
}

// MessageConfig 消息处理配置
type MessageConfig struct {
	MaxLength      int                  `yaml:"max_length"`
	AllowedTypes   []int                `yaml:"allowed_types"`
	SensitiveWords SensitiveWordsConfig `yaml:"sensitive_words"`
	RateLimit      RateLimitConfig      `yaml:"rate_limit"`
}

// SensitiveWordsConfig 敏感词过滤配置
type SensitiveWordsConfig struct {
	Enabled     bool   `yaml:"enabled"`
	WordsFile   string `yaml:"words_file"`
	ReplaceChar string `yaml:"replace_char"`
}

// RateLimitConfig 限流配置
type RateLimitConfig struct {
	Enabled      bool `yaml:"enabled"`
	MaxPerSecond int  `yaml:"max_per_second"`
	MaxPerMinute int  `yaml:"max_per_minute"`
	MaxPerHour   int  `yaml:"max_per_hour"`
}

// GroupConfig 群组配置
type GroupConfig struct {
	MaxMembers       int    `yaml:"max_members"`
	DefaultName      string `yaml:"default_name"`
	CreatePermission bool   `yaml:"create_permission"`
	JoinApproval     bool   `yaml:"join_approval"`
}

// MonitoringConfig 监控配置
type MonitoringConfig struct {
	MetricsEnabled bool   `yaml:"metrics_enabled"`
	MetricsPort    int    `yaml:"metrics_port"`
	TracingEnabled bool   `yaml:"tracing_enabled"`
	TracingAddress string `yaml:"tracing_address"`
	ServiceName    string `yaml:"service_name"`
}

// DiscoveryConfig 服务发现配置
type DiscoveryConfig struct {
	Type              string   `yaml:"type"`
	Endpoints         []string `yaml:"endpoints"`
	ServiceName       string   `yaml:"service_name"`
	ServiceAddress    string   `yaml:"service_address"`
	RegisterInterval  int      `yaml:"register_interval"`
	HeartbeatInterval int      `yaml:"heartbeat_interval"`
	LeaseTTL          int      `yaml:"lease_ttl"`
}

// Load 加载配置
func Load() (*Config, error) {
	viper.SetConfigName("config")
	viper.SetConfigType("yaml")
	viper.AddConfigPath("configs")
	viper.AddConfigPath(".")

	// 设置默认值
	setDefaults()

	// 绑定环境变量
	bindEnvVars()

	// 读取配置文件
	if err := viper.ReadInConfig(); err != nil {
		return nil, fmt.Errorf("读取配置文件失败: %w", err)
	}

	var config Config
	if err := viper.Unmarshal(&config); err != nil {
		return nil, fmt.Errorf("解析配置失败: %w", err)
	}

	return &config, nil
}

// setDefaults 设置默认值
func setDefaults() {
	// 服务器配置
	viper.SetDefault("server.grpc.host", "0.0.0.0")
	viper.SetDefault("server.grpc.port", 9001)
	viper.SetDefault("server.grpc.max_conn", 1000)
	viper.SetDefault("server.grpc.max_msg_size", 10)
	viper.SetDefault("server.grpc.timeout", 30)
	viper.SetDefault("server.http.host", "0.0.0.0")
	viper.SetDefault("server.http.port", 9002)
	viper.SetDefault("server.http.timeout", 10)

	// 日志配置
	viper.SetDefault("logging.level", "info")
	viper.SetDefault("logging.format", "json")
	viper.SetDefault("logging.max_size", 100)
	viper.SetDefault("logging.max_age", 30)
	viper.SetDefault("logging.max_backups", 10)
	viper.SetDefault("logging.compress", true)

	// 数据库配置
	viper.SetDefault("database.type", "mysql")
	viper.SetDefault("database.host", "localhost")
	viper.SetDefault("database.port", 3306)
	viper.SetDefault("database.name", "gochat")
	viper.SetDefault("database.username", "root")
	viper.SetDefault("database.password", "password")
	viper.SetDefault("database.max_conn", 100)
	viper.SetDefault("database.max_idle_conn", 20)
	viper.SetDefault("database.conn_max_lifetime", 1)
	viper.SetDefault("database.conn_max_idle_time", 10)

	// Redis 配置
	viper.SetDefault("redis.addr", "localhost:6379")
	viper.SetDefault("redis.db", 0)
	viper.SetDefault("redis.pool_size", 20)
	viper.SetDefault("redis.min_idle_conns", 5)
	viper.SetDefault("redis.dial_timeout", 5)
	viper.SetDefault("redis.read_timeout", 3)
	viper.SetDefault("redis.write_timeout", 3)

	// Kafka 配置
	viper.SetDefault("kafka.brokers", []string{"localhost:9092"})
	viper.SetDefault("kafka.upstream_topic", "im-upstream-topic")
	viper.SetDefault("kafka.downstream_topic_prefix", "im-downstream-topic-")
	viper.SetDefault("kafka.task_topic", "im-task-topic")
	viper.SetDefault("kafka.consumer_group", "im-logic-group")
	viper.SetDefault("kafka.batch_size", 100)
	viper.SetDefault("kafka.batch_timeout", 100)
	viper.SetDefault("kafka.session_timeout", 30)
	viper.SetDefault("kafka.heartbeat_interval", 3)
	viper.SetDefault("kafka.retry.max_retries", 3)
	viper.SetDefault("kafka.retry.retry_interval", 5)
	viper.SetDefault("kafka.retry.backoff_factor", 2)
	viper.SetDefault("kafka.producer.acks", "all")
	viper.SetDefault("kafka.producer.retries", 3)
	viper.SetDefault("kafka.producer.retry_backoff", 100)
	viper.SetDefault("kafka.producer.batch_size", 1048576)
	viper.SetDefault("kafka.producer.batch_timeout", 10)
	viper.SetDefault("kafka.producer.compression", "gzip")

	// JWT 配置
	viper.SetDefault("jwt.secret", "your-secret-key-here")
	viper.SetDefault("jwt.access_token_expire", 24)
	viper.SetDefault("jwt.refresh_token_expire", 7)
	viper.SetDefault("jwt.signing_method", "HS256")

	// 数据仓储配置
	viper.SetDefault("repo.grpc.address", "localhost:9002")
	viper.SetDefault("repo.grpc.timeout", 10)
	viper.SetDefault("repo.grpc.max_retries", 3)
	viper.SetDefault("repo.grpc.retry_interval", 1)
	viper.SetDefault("repo.grpc.enable_tls", false)

	// 消息配置
	viper.SetDefault("message.max_length", 5000)
	viper.SetDefault("message.allowed_types", []int{1, 2, 3, 4, 5, 6})
	viper.SetDefault("message.sensitive_words.enabled", true)
	viper.SetDefault("message.sensitive_words.replace_char", "*")
	viper.SetDefault("message.rate_limit.enabled", true)
	viper.SetDefault("message.rate_limit.max_per_second", 10)
	viper.SetDefault("message.rate_limit.max_per_minute", 100)
	viper.SetDefault("message.rate_limit.max_per_hour", 1000)

	// 群组配置
	viper.SetDefault("group.max_members", 500)
	viper.SetDefault("group.default_name", "未命名群组")
	viper.SetDefault("group.create_permission", true)
	viper.SetDefault("group.join_approval", false)

	// 监控配置
	viper.SetDefault("monitoring.metrics_enabled", true)
	viper.SetDefault("monitoring.metrics_port", 9003)
	viper.SetDefault("monitoring.tracing_enabled", true)
	viper.SetDefault("monitoring.tracing_address", "localhost:14268")
	viper.SetDefault("monitoring.service_name", "im-logic")

	// 服务发现配置
	viper.SetDefault("discovery.type", "etcd")
	viper.SetDefault("discovery.endpoints", []string{"localhost:2379"})
	viper.SetDefault("discovery.service_name", "im-logic")
	viper.SetDefault("discovery.service_address", "localhost:9001")
	viper.SetDefault("discovery.register_interval", 10)
	viper.SetDefault("discovery.heartbeat_interval", 5)
	viper.SetDefault("discovery.lease_ttl", 30)
}

// bindEnvVars 绑定环境变量
func bindEnvVars() {
	// 服务器配置
	viper.BindEnv("server.grpc.host", "SERVER_GRPC_HOST")
	viper.BindEnv("server.grpc.port", "SERVER_GRPC_PORT")
	viper.BindEnv("server.http.port", "SERVER_HTTP_PORT")

	// 日志配置
	viper.BindEnv("logging.level", "LOG_LEVEL")
	viper.BindEnv("logging.format", "LOG_FORMAT")

	// 数据库配置
	viper.BindEnv("database.host", "DB_HOST")
	viper.BindEnv("database.port", "DB_PORT")
	viper.BindEnv("database.name", "DB_NAME")
	viper.BindEnv("database.username", "DB_USERNAME")
	viper.BindEnv("database.password", "DB_PASSWORD")

	// Redis 配置
	viper.BindEnv("redis.addr", "REDIS_ADDR")
	viper.BindEnv("redis.password", "REDIS_PASSWORD")
	viper.BindEnv("redis.db", "REDIS_DB")

	// Kafka 配置
	viper.BindEnv("kafka.brokers", "KAFKA_BROKERS")
	viper.BindEnv("kafka.upstream_topic", "KAFKA_UPSTREAM_TOPIC")
	viper.BindEnv("kafka.downstream_topic_prefix", "KAFKA_DOWNSTREAM_TOPIC_PREFIX")
	viper.BindEnv("kafka.task_topic", "KAFKA_TASK_TOPIC")

	// JWT 配置
	viper.BindEnv("jwt.secret", "JWT_SECRET")

	// 数据仓储配置
	viper.BindEnv("repo.grpc.address", "REPO_GRPC_ADDRESS")
}

// GetGRPCAddr 获取 gRPC 地址
func (c *Config) GetGRPCAddr() string {
	return fmt.Sprintf("%s:%d", c.Server.GRPC.Host, c.Server.GRPC.Port)
}

// GetHTTPAddr 获取 HTTP 地址
func (c *Config) GetHTTPAddr() string {
	return fmt.Sprintf("%s:%d", c.Server.HTTP.Host, c.Server.HTTP.Port)
}

// GetDatabaseDSN 获取数据库连接字符串
func (c *Config) GetDatabaseDSN() string {
	switch c.Database.Type {
	case "mysql":
		return fmt.Sprintf("%s:%s@tcp(%s:%d)/%s?charset=utf8mb4&parseTime=True&loc=Local",
			c.Database.Username, c.Database.Password, c.Database.Host, c.Database.Port, c.Database.Name)
	case "postgresql":
		return fmt.Sprintf("host=%s port=%d user=%s password=%s dbname=%s sslmode=disable",
			c.Database.Host, c.Database.Port, c.Database.Username, c.Database.Password, c.Database.Name)
	case "sqlite":
		return c.Database.Name
	default:
		return ""
	}
}

// GetDownstreamTopic 获取下行消息 Topic
func (c *Config) GetDownstreamTopic(gatewayID string) string {
	return c.Kafka.DownstreamTopicPrefix + gatewayID
}

// GetAccessTokenExpireDuration 获取访问令牌过期时间
func (c *Config) GetAccessTokenExpireDuration() time.Duration {
	return time.Duration(c.JWT.AccessTokenExpire) * time.Hour
}

// GetRefreshTokenExpireDuration 获取刷新令牌过期时间
func (c *Config) GetRefreshTokenExpireDuration() time.Duration {
	return time.Duration(c.JWT.RefreshTokenExpire) * 24 * time.Hour
}
