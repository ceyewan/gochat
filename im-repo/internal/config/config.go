package config

import (
	"time"

	"github.com/ceyewan/gochat/im-infra/cache"
	"github.com/ceyewan/gochat/im-infra/coord"
	"github.com/ceyewan/gochat/im-infra/db"
)

// Config 定义 im-repo 服务的完整配置结构
type Config struct {
	// 服务器配置
	Server ServerConfig `yaml:"server" json:"server"`

	// 数据库配置
	Database db.Config `yaml:"database" json:"database"`

	// 缓存配置
	Cache cache.Config `yaml:"cache" json:"cache"`

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

// BusinessConfig 业务相关配置
type BusinessConfig struct {
	// 缓存配置
	Cache CacheConfig `yaml:"cache" json:"cache"`

	// 数据库配置
	Database DatabaseConfig `yaml:"database" json:"database"`

	// ID 生成配置
	IDGenerator IDGeneratorConfig `yaml:"id_generator" json:"id_generator"`
}

// CacheConfig 缓存业务配置
type CacheConfig struct {
	// 用户信息缓存 TTL
	UserInfoTTL time.Duration `yaml:"user_info_ttl" json:"user_info_ttl"`

	// 群组成员缓存 TTL
	GroupMembersTTL time.Duration `yaml:"group_members_ttl" json:"group_members_ttl"`

	// 热点消息缓存数量
	HotMessagesCount int `yaml:"hot_messages_count" json:"hot_messages_count"`

	// 消息去重 TTL
	MessageDedupTTL time.Duration `yaml:"message_dedup_ttl" json:"message_dedup_ttl"`

	// 在线状态 TTL
	OnlineStatusTTL time.Duration `yaml:"online_status_ttl" json:"online_status_ttl"`
}

// DatabaseConfig 数据库业务配置
type DatabaseConfig struct {
	// 批量查询大小
	BatchSize int `yaml:"batch_size" json:"batch_size"`

	// 查询超时
	QueryTimeout time.Duration `yaml:"query_timeout" json:"query_timeout"`

	// 事务超时
	TransactionTimeout time.Duration `yaml:"transaction_timeout" json:"transaction_timeout"`

	// 是否启用软删除
	EnableSoftDelete bool `yaml:"enable_soft_delete" json:"enable_soft_delete"`
}

// IDGeneratorConfig ID 生成器配置
type IDGeneratorConfig struct {
	// 机器 ID（用于雪花算法）
	MachineID int64 `yaml:"machine_id" json:"machine_id"`

	// 数据中心 ID
	DatacenterID int64 `yaml:"datacenter_id" json:"datacenter_id"`
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
			GRPCAddr:    ":9002",
			ServiceName: "im-repo",
			Version:     "1.0.0",
			HealthPort:  ":8002",
		},
		Database: db.DefaultConfig(),
		Cache:    cache.DefaultConfig(),
		Coordinator: coord.CoordinatorConfig{
			Endpoints: []string{"localhost:2379"},
		},
		Business: BusinessConfig{
			Cache: CacheConfig{
				UserInfoTTL:      24 * time.Hour,
				GroupMembersTTL:  6 * time.Hour,
				HotMessagesCount: 300,
				MessageDedupTTL:  60 * time.Second,
				OnlineStatusTTL:  5 * time.Minute,
			},
			Database: DatabaseConfig{
				BatchSize:          200,
				QueryTimeout:       10 * time.Second,
				TransactionTimeout: 30 * time.Second,
				EnableSoftDelete:   true,
			},
			IDGenerator: IDGeneratorConfig{
				MachineID:    1,
				DatacenterID: 1,
			},
		},
		Log: LogConfig{
			Level:  "info",
			Format: "json",
			Output: "stdout",
		},
	}
}
