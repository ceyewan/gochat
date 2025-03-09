package config

import (
	"fmt"
	"os"

	"gochat/clog"

	"github.com/spf13/viper"
)

// Config 实例
var Conf Config

// 操作码定义
const (
	OpSingleSend    = iota // 发送消息给单个用户的操作码
	OpRoomSend             // 发送消息到聊天室的操作码
	OpRoomCountSend        // 获取在线用户数量的操作码
	OpRoomInfoSend         // 发送信息到聊天室的操作码
)

// RPC Code 定义
const (
	RPCCodeSuccess = iota
	RPCCodeFailed
	RPCUnknowError
	RPCSessionError
)

// 环境模式常量
const (
	ModeDev     = "dev"
	ModeTest    = "test"
	ModeRelease = "release"
)

// MySQL 定义 MySQL 数据库的配置结构体
type MySQL struct {
	Host     string `mapstructure:"host"`     // MySQL 服务器主机名
	Port     int    `mapstructure:"port"`     // MySQL 服务器端口
	Username string `mapstructure:"username"` // 用户名
	Password string `mapstructure:"password"` // 密码
	Charset  string `mapstructure:"charset"`  // 字符集
	DbName   string `mapstructure:"dbname"`   // 数据库名称
}

// Redis 定义 Redis 数据库的配置结构体
type Redis struct {
	Addr     string `mapstructure:"addr"`     // Redis 服务器主机名
	Password string `mapstructure:"password"` // 密码
	DB       int    `mapstructure:"db"`       // 数据库
}

// RPC 定义 RPC 服务的配置结构体
type RPC struct {
	Port int `mapstructure:"port"` // RPC 服务端口
}

// APIConfig 定义 API 服务的配置结构体
type APIConfig struct {
	Port int `mapstructure:"port"` // API 服务端口
}

// Etcd 定义 Etcd 服务的配置结构体
type Etcd struct {
	Addrs []string `mapstructure:"addrs"` // Etcd 服务器地址
}

// TaskConfig 定义 Task 服务的配置结构体
type TaskConfig struct {
	ChannelSize int `mapstructure:"channelsize"` // 通道大小
}

// ConnectConfig 定义 Connect 服务的配置结构体
type ConnectConfig struct {
	Websocket struct {
		Bind            string `mapstructure:"bind"`
		ReadBufferSize  int    `mapstructure:"read_buffer_size"`
		WriteBufferSize int    `mapstructure:"write_buffer_size"`
		BroadcastSize   int    `mapstructure:"broadcast_size"`
	} `mapstructure:"websocket"`
}

// EnvConfig 定义环境相关的配置结构体
type EnvConfig struct {
	Mode    string `mapstructure:"mode"`     // 运行模式: dev, test, release
	GinMode string `mapstructure:"gin_mode"` // Gin框架运行模式: debug, test, release
}

// JWTKey 定义 JWT 签名密钥配置结构体
type JWTKey struct {
	SignKey string `mapstructure:"signkey"` // 签名密钥
}

// Config 是应用程序的主配置结构体，包含了所有服务组件的配置
type Config struct {
	MySQL      MySQL         // MySQL数据库配置
	Redis      Redis         // Redis数据库配置
	RPC        RPC           // RPC服务配置
	APIConfig  APIConfig     // API服务配置
	Etcd       Etcd          // Etcd配置
	TaskConfig TaskConfig    // Task配置
	Connect    ConnectConfig // Connect服务配置
	Env        EnvConfig     // 环境相关配置
	JWTKey     JWTKey        // JWT签名密钥
}

func init() {
	clog.Debug("Initializing configuration module")
	err := LoadConfig()
	if err != nil {
		clog.Error("Failed to load configuration: %v", err)
	} else {
		clog.Info("Configuration loaded successfully")
	}
}

// LoadConfig 从配置文件加载配置
func LoadConfig() error {
	var configFile string

	// 从环境变量获取配置文件路径
	envConfigPath := os.Getenv("CONFIG_FILE")
	if envConfigPath != "" {
		configFile = envConfigPath
		clog.Debug("Using configuration file from environment: %s", configFile)
	} else {
		configFile = "config/config.yaml"
		clog.Debug("Using default configuration file: %s", configFile)
	}

	// 检查配置文件是否存在
	if _, err := os.Stat(configFile); os.IsNotExist(err) {
		clog.Warning("Configuration file %s does not exist", configFile)
		return fmt.Errorf("configuration file not found: %s", configFile)
	}

	viper.SetConfigFile(configFile)
	clog.Debug("Reading configuration from file: %s", configFile)

	if err := viper.ReadInConfig(); err != nil {
		clog.Error("Failed to read configuration file: %v", err)
		return fmt.Errorf("failed to read config file: %w", err)
	}

	if err := viper.Unmarshal(&Conf); err != nil {
		clog.Error("Failed to parse configuration: %v", err)
		return fmt.Errorf("failed to parse config file: %w", err)
	}

	// 补充配置，确保所有需要的值都有默认值
	setDefaultConfig()

	clog.Info("Configuration loaded successfully with mode: %s", GetMode())
	logConfigSummary()

	return nil
}

// setDefaultConfig 为缺失的配置设置默认值
func setDefaultConfig() {
	// 设置默认运行模式
	if Conf.Env.Mode == "" {
		Conf.Env.Mode = ModeDev
		clog.Debug("No environment mode specified, using default: %s", ModeDev)
	}

	// 设置默认Gin模式
	if Conf.Env.GinMode == "" {
		Conf.Env.GinMode = getGinModeFromEnvMode(Conf.Env.Mode)
		clog.Debug("No GinMode specified, derived from env mode: %s", Conf.Env.GinMode)
	}
}

// getGinModeFromEnvMode 根据环境模式获取对应的Gin模式
func getGinModeFromEnvMode(mode string) string {
	switch mode {
	case ModeDev, ModeTest:
		return "debug"
	case ModeRelease:
		return "release"
	default:
		clog.Warning("Unknown environment mode: %s, falling back to 'debug'", mode)
		return "debug"
	}
}

// logConfigSummary 输出配置摘要信息
func logConfigSummary() {
	clog.Debug("Configuration summary:")
	clog.Debug("- Environment: %s (Gin: %s)", Conf.Env.Mode, Conf.Env.GinMode)

	// 数据库配置摘要
	if Conf.MySQL.Host != "" {
		clog.Debug("- MySQL: %s:%d (user: %s)",
			Conf.MySQL.Host, Conf.MySQL.Port, Conf.MySQL.Username)
	}

	if Conf.Redis.Addr != "" {
		clog.Debug("- Redis: %s (DB: %d)", Conf.Redis.Addr, Conf.Redis.DB)
	}

	// 服务配置摘要
	if Conf.APIConfig.Port > 0 {
		clog.Debug("- API service: port %d", Conf.APIConfig.Port)
	}

	if Conf.RPC.Port > 0 {
		clog.Debug("- RPC service: port %d", Conf.RPC.Port)
	}

	if len(Conf.Etcd.Addrs) > 0 {
		clog.Debug("- Etcd: %v", Conf.Etcd.Addrs)
	}

	// 连接服务配置
	if Conf.Connect.Websocket.Bind != "" {
		clog.Debug("- WebSocket binding: %s", Conf.Connect.Websocket.Bind)
	}
}

// GetMode 获取当前运行模式
func GetMode() string {
	return Conf.Env.Mode
}

// GetGinRunMode 获取Gin框架运行模式
func GetGinRunMode() string {
	return Conf.Env.GinMode
}
