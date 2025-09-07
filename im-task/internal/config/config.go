package config

import (
	"fmt"
	"time"

	"github.com/spf13/viper"
)

type Config struct {
	Server      ServerConfig      `mapstructure:"server"`
	Grpc        GrpcConfig        `mapstructure:"grpc"`
	Kafka       KafkaConfig       `mapstructure:"kafka"`
	Redis       RedisConfig       `mapstructure:"redis"`
	RepoService RepoServiceConfig `mapstructure:"repo_service"`
	Tracing     TracingConfig     `mapstructure:"tracing"`
	Log         LogConfig         `mapstructure:"log"`
	Metrics     MetricsConfig     `mapstructure:"metrics"`
	Health      HealthConfig      `mapstructure:"health"`
	Task        TaskConfig        `mapstructure:"task"`
}

type ServerConfig struct {
	Name    string `mapstructure:"name"`
	Version string `mapstructure:"version"`
	Host    string `mapstructure:"host"`
	Port    int    `mapstructure:"port"`
	Mode    string `mapstructure:"mode"`
}

type GrpcConfig struct {
	Server GrpcServerConfig `mapstructure:"server"`
	Client GrpcClientConfig `mapstructure:"client"`
}

type GrpcServerConfig struct {
	Port    int           `mapstructure:"port"`
	Timeout time.Duration `mapstructure:"timeout"`
}

type GrpcClientConfig struct {
	Timeout time.Duration `mapstructure:"timeout"`
	Retry   RetryConfig   `mapstructure:"retry"`
}

type RetryConfig struct {
	MaxAttempts int           `mapstructure:"max_attempts"`
	Backoff     time.Duration `mapstructure:"backoff"`
	MaxBackoff  time.Duration `mapstructure:"max_backoff"`
}

type KafkaConfig struct {
	Brokers           []string      `mapstructure:"brokers"`
	TaskTopic         string        `mapstructure:"task_topic"`
	PersistenceTopic  string        `mapstructure:"persistence_topic"`
	ConsumerGroup     string        `mapstructure:"consumer_group"`
	PersistenceGroup  string        `mapstructure:"persistence_group"`
	SessionTimeout    time.Duration `mapstructure:"session_timeout"`
	HeartbeatInterval time.Duration `mapstructure:"heartbeat_interval"`
	RebalanceTimeout  time.Duration `mapstructure:"rebalance_timeout"`
	BatchSize         int           `mapstructure:"batch_size"`
	BatchTimeout      time.Duration `mapstructure:"batch_timeout"`
	StartOffset       string        `mapstructure:"start_offset"`
}

type RedisConfig struct {
	Host         string        `mapstructure:"host"`
	Port         int           `mapstructure:"port"`
	Password     string        `mapstructure:"password"`
	DB           int           `mapstructure:"db"`
	PoolSize     int           `mapstructure:"pool_size"`
	MinIdleConns int           `mapstructure:"min_idle_conns"`
	DialTimeout  time.Duration `mapstructure:"dial_timeout"`
	ReadTimeout  time.Duration `mapstructure:"read_timeout"`
	WriteTimeout time.Duration `mapstructure:"write_timeout"`
}

type RepoServiceConfig struct {
	Name    string        `mapstructure:"name"`
	Host    string        `mapstructure:"host"`
	Port    int           `mapstructure:"port"`
	Timeout time.Duration `mapstructure:"timeout"`
}

type TracingConfig struct {
	Enabled bool         `mapstructure:"enabled"`
	Jaeger  JaegerConfig `mapstructure:"jaeger"`
}

type JaegerConfig struct {
	Endpoint     string `mapstructure:"endpoint"`
	ServiceName  string `mapstructure:"service_name"`
	SamplerType  string `mapstructure:"sampler_type"`
	SamplerParam int    `mapstructure:"sampler_param"`
}

type LogConfig struct {
	Level      string `mapstructure:"level"`
	Format     string `mapstructure:"format"`
	Output     string `mapstructure:"output"`
	Filename   string `mapstructure:"filename"`
	MaxSize    int    `mapstructure:"max_size"`
	MaxBackups int    `mapstructure:"max_backups"`
	MaxAge     int    `mapstructure:"max_age"`
	Compress   bool   `mapstructure:"compress"`
	CallerSkip int    `mapstructure:"caller_skip"`
}

type MetricsConfig struct {
	Enabled bool   `mapstructure:"enabled"`
	Port    int    `mapstructure:"port"`
	Path    string `mapstructure:"path"`
}

type HealthConfig struct {
	Enabled bool   `mapstructure:"enabled"`
	Path    string `mapstructure:"path"`
}

type TaskConfig struct {
	LargeGroupThreshold     int           `mapstructure:"large_group_threshold"`
	FanoutBatchSize         int           `mapstructure:"fanout_batch_size"`
	PushNotificationEnabled bool          `mapstructure:"push_notification_enabled"`
	OfflinePushTimeout      time.Duration `mapstructure:"offline_push_timeout"`
	MessageRetryAttempts    int           `mapstructure:"message_retry_attempts"`
	MessageRetryBackoff     time.Duration `mapstructure:"message_retry_backoff"`
}

func Load(configPath string) (*Config, error) {
	viper.SetConfigFile(configPath)
	viper.AutomaticEnv()

	if err := viper.ReadInConfig(); err != nil {
		return nil, fmt.Errorf("failed to read config file: %w", err)
	}

	var config Config
	if err := viper.Unmarshal(&config); err != nil {
		return nil, fmt.Errorf("failed to unmarshal config: %w", err)
	}

	return &config, nil
}

func Default() *Config {
	return &Config{
		Server: ServerConfig{
			Name:    "im-task",
			Version: "1.0.0",
			Host:    "0.0.0.0",
			Port:    9003,
			Mode:    "dev",
		},
		Grpc: GrpcConfig{
			Server: GrpcServerConfig{
				Port:    9003,
				Timeout: 30 * time.Second,
			},
			Client: GrpcClientConfig{
				Timeout: 10 * time.Second,
				Retry: RetryConfig{
					MaxAttempts: 3,
					Backoff:     100 * time.Millisecond,
					MaxBackoff:  1 * time.Second,
				},
			},
		},
		Kafka: KafkaConfig{
			Brokers:           []string{"localhost:9092"},
			TaskTopic:         "im-task-topic",
			PersistenceTopic:  "im-persistence-topic",
			ConsumerGroup:     "im-task-group",
			PersistenceGroup:  "im-persistence-group",
			SessionTimeout:    30 * time.Second,
			HeartbeatInterval: 3 * time.Second,
			RebalanceTimeout:  60 * time.Second,
			BatchSize:         100,
			BatchTimeout:      1 * time.Second,
			StartOffset:       "newest",
		},
		Redis: RedisConfig{
			Host:         "localhost",
			Port:         6379,
			Password:     "",
			DB:           0,
			PoolSize:     10,
			MinIdleConns: 5,
			DialTimeout:  5 * time.Second,
			ReadTimeout:  3 * time.Second,
			WriteTimeout: 3 * time.Second,
		},
		RepoService: RepoServiceConfig{
			Name:    "im-repo",
			Host:    "localhost",
			Port:    9002,
			Timeout: 10 * time.Second,
		},
		Tracing: TracingConfig{
			Enabled: true,
			Jaeger: JaegerConfig{
				Endpoint:     "localhost:14268",
				ServiceName:  "im-task",
				SamplerType:  "const",
				SamplerParam: 1,
			},
		},
		Log: LogConfig{
			Level:      "info",
			Format:     "json",
			Output:     "stdout",
			Filename:   "",
			MaxSize:    100,
			MaxBackups: 10,
			MaxAge:     30,
			Compress:   true,
			CallerSkip: 2,
		},
		Metrics: MetricsConfig{
			Enabled: true,
			Port:    9093,
			Path:    "/metrics",
		},
		Health: HealthConfig{
			Enabled: true,
			Path:    "/health",
		},
		Task: TaskConfig{
			LargeGroupThreshold:     100,
			FanoutBatchSize:         50,
			PushNotificationEnabled: true,
			OfflinePushTimeout:      5 * time.Second,
			MessageRetryAttempts:    3,
			MessageRetryBackoff:     1 * time.Second,
		},
	}
}
