package config

import "os"

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

// API 端口配置
var APIConfig struct {
	Port int `json:"port"`
}
