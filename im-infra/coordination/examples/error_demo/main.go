package main

import (
	"fmt"

	"github.com/ceyewan/gochat/im-infra/coordination"
)

func main() {
	fmt.Println("=== 错误处理演示 ===")

	// 使用不存在的 etcd 端点来演示错误处理
	cfg := coordination.ExampleConfig()
	cfg.Endpoints = []string{"localhost:9999"} // 不存在的端口

	fmt.Printf("尝试连接到不存在的 etcd 端点: %v\n", cfg.Endpoints)
	fmt.Printf("连接超时设置: %v\n", cfg.DialTimeout)

	coordinator, err := coordination.New(cfg)
	if err != nil {
		fmt.Printf("\n预期的连接错误:\n%v\n", err)
		return
	}
	defer coordinator.Close()

	fmt.Println("意外成功连接！")
}
