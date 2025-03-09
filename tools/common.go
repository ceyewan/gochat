package tools

import (
	"errors"
	"net"
	"sync"
	"time"

	"github.com/bwmarrin/snowflake"
)

// GetLocalIP 获取本机首个非环回IPv4地址
// 返回格式如："192.168.1.100"
func GetLocalIP() (string, error) {
	// 获取所有网络接口
	interfaces, err := net.Interfaces()
	if err != nil {
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
			return ip.String(), nil
		}
	}

	return "", errors.New("无法获取本机IP地址")
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
	// 获取本机IP地址
	ip, err := GetLocalIP()
	if err != nil {
		// 如果获取IP失败，使用默认节点ID 1
		snowflakeNode, nodeInitError = snowflake.NewNode(1)
		return
	}
	// 解析IP地址
	ipObj := net.ParseIP(ip)
	if ipObj == nil {
		// 如果解析IP失败，使用默认节点ID 1
		snowflakeNode, nodeInitError = snowflake.NewNode(1)
		return
	}
	// 使用IP地址的最后一个字节作为节点ID (范围 0-255)
	// snowflake库的Node ID范围是 0-1023，所以我们可以直接使用IP的最后一个字节
	nodeID := int64(ipObj.To4()[3])
	// 创建雪花算法节点
	snowflakeNode, nodeInitError = snowflake.NewNode(nodeID)
}

// GetSnowflakeID 生成一个全局唯一的64位整数ID
func GetSnowflakeID() int64 {
	// 确保节点只被初始化一次
	onceSnow.Do(initSnowflakeNode)

	// 如果初始化失败，返回时间戳作为备用方案
	if nodeInitError != nil || snowflakeNode == nil {
		return int64(time.Now().UnixNano())
	}

	// 生成雪花ID并返回
	return snowflakeNode.Generate().Int64()
}
