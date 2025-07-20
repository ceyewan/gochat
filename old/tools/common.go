package tools

import (
	"errors"
	"gochat/clog"
	"net"
	"sync"
	"time"

	"github.com/bwmarrin/snowflake"
)

// IP 相关常量
const (
	defaultNodeID = 1 // 默认雪花算法节点ID
)

// GetLocalIP 获取本机首个非环回IPv4地址
// 返回格式如："192.168.1.100"
func GetLocalIP() (string, error) {
	clog.Module("common").Debugf("Attempting to get local IP address")

	// 获取所有网络接口
	interfaces, err := net.Interfaces()
	if err != nil {
		clog.Module("common").Errorf("Failed to get network interfaces: %v", err)
		return "", err
	}

	// 遍历所有网络接口
	for _, iface := range interfaces {
		// 跳过禁用的接口
		if iface.Flags&net.FlagUp == 0 {
			continue
		}
		// 跳过回环接口
		if iface.Flags&net.FlagLoopback != 0 {
			continue
		}

		// 获取该接口的所有地址
		addrs, err := iface.Addrs()
		if err != nil {
			clog.Module("common").Debugf("Failed to get addresses for interface %s: %v", iface.Name, err)
			continue
		}

		// 遍历所有地址
		for _, addr := range addrs {
			// 检查是否为IP地址，而不是Unix域套接字地址
			ipNet, ok := addr.(*net.IPNet)
			if !ok {
				continue
			}

			// 检查是否为IPv4地址，且不是环回地址
			ip := ipNet.IP.To4()
			if ip == nil || ip.IsLoopback() {
				continue
			}

			// 返回找到的第一个有效IP地址
			ipAddress := ip.String()
			clog.Module("common").Debugf("Found valid local IP: %s", ipAddress)
			return ipAddress, nil
		}
	}

	clog.Module("common").Warnf("Unable to find any valid local IP address")
	return "", errors.New("no valid local IP address found")
}

// snowflakeNode 是雪花算法的节点实例
var (
	snowflakeNode *snowflake.Node
	onceSnow      sync.Once
	nodeInitError error
)

// initSnowflakeNode 初始化雪花算法节点
// 使用本机IP地址的最后一个字节作为节点ID
func initSnowflakeNode() {
	clog.Module("common").Debugf("Initializing snowflake node")

	// 获取本机IP地址
	ip, err := GetLocalIP()
	if err != nil {
		// 如果获取IP失败，使用默认节点ID
		clog.Module("common").Warnf("Failed to get local IP, using default node ID %d: %v", defaultNodeID, err)
		snowflakeNode, nodeInitError = snowflake.NewNode(defaultNodeID)
		return
	}

	// 解析IP地址
	ipObj := net.ParseIP(ip)
	if ipObj == nil {
		// 如果解析IP失败，使用默认节点ID
		clog.Module("common").Warnf("Failed to parse IP %s, using default node ID %d", ip, defaultNodeID)
		snowflakeNode, nodeInitError = snowflake.NewNode(defaultNodeID)
		return
	}

	// 使用IP地址的最后一个字节作为节点ID (范围 0-255)
	// snowflake库的Node ID范围是 0-1023，所以我们可以直接使用IP的最后一个字节
	nodeID := int64(ipObj.To4()[3])
	clog.Module("common").Debugf("Using node ID %d derived from IP %s", nodeID, ip)

	// 创建雪花算法节点
	snowflakeNode, nodeInitError = snowflake.NewNode(nodeID)

	if nodeInitError != nil {
		clog.Module("common").Errorf("Failed to create snowflake node: %v", nodeInitError)
	} else {
		clog.Module("common").Infof("Snowflake node initialized successfully with ID %d", nodeID)
	}
}

// GetSnowflakeID 生成一个全局唯一的64位整数ID
func GetSnowflakeID() int64 {
	// 确保节点只被初始化一次
	onceSnow.Do(initSnowflakeNode)

	// 如果初始化失败，返回时间戳作为备用方案
	if nodeInitError != nil || snowflakeNode == nil {
		timestamp := int64(time.Now().UnixNano())
		clog.Module("common").Warnf("Using fallback timestamp ID due to snowflake initialization failure")
		return timestamp
	}

	// 生成雪花ID并返回
	id := snowflakeNode.Generate().Int64()
	clog.Module("common").Debugf("Generated snowflake ID: %d", id)
	return id
}

// SendMsg 用于发送消息的结构体
type SendMsg struct {
	Count        int               `json:"count"`          // 计数
	Msg          string            `json:"msg"`            // 消息内容
	RoomUserInfo map[string]string `json:"room_user_info"` // 房间用户信息
}
