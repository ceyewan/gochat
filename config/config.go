package config

import (
	"fmt"
	"os"

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

// MySQL 定义 MySQL 数据库的配置结构体
type MySQL struct {
	Host     string `mapstructure:"host"`     // MySQL 服务器主机名
	Port     int    `mapstructure:"port"`     // MySQL 服务器端口
	Username string `mapstructure:"username"` // 用户名
	Password string `mapstructure:"password"` // 密码
	Charset  string `mapstructure:"charset"`  // 字符集
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

type TaskConfig struct {
	ChannelSize int `mapstructure:"channelsize"` // 通道大小
}

// Config 是应用程序的主配置结构体，包含了所有服务组件的配置
type Config struct {
	MySQL      MySQL      // MySQL数据库配置
	Redis      Redis      // Redis数据库配置
	LogicRPC   RPC        // RPC服务配置
	APIConfig  APIConfig  // API服务配置
	Etcd       Etcd       // Etcd配置
	TaskConfig TaskConfig // Task配置
}

func init() {
	LoadConfig()
}

// LoadConfig 从配置文件加载配置
func LoadConfig() error {
	mode := GetMode()
	configFile := fmt.Sprintf("config/config.%s.yaml", mode)

	// 检查配置文件是否存在，不存在则使用默认配置文件
	if _, err := os.Stat(configFile); os.IsNotExist(err) {
		configFile = "config/config.yaml"
	}

	viper.SetConfigFile(configFile)

	if err := viper.ReadInConfig(); err != nil {
		return fmt.Errorf("读取配置文件失败: %w", err)
	}

	if err := viper.Unmarshal(&Conf); err != nil {
		return fmt.Errorf("解析配置文件失败: %w", err)
	}

	return nil
}

// GetMode 获取当前运行模式，从环境变量RUN_MODE中读取，默认为"dev"
func GetMode() string {
	env := os.Getenv("RUN_MODE")
	if env == "" {
		env = "dev"
	}
	return env
}

// GetGinRunMode 根据当前运行模式返回Gin框架对应的运行模式
// dev和test环境返回debug模式，prod环境返回release模式
func GetGinRunMode() string {
	env := GetMode()
	//gin have debug,test,release mode
	if env == "dev" {
		return "debug"
	}
	if env == "test" {
		return "debug"
	}
	if env == "prod" {
		return "release"
	}
	return "release"
}
