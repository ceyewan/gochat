package tools

import (
	"errors"
	"net"
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
