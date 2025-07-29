package main

import (
	"fmt"
	"time"

	"github.com/ceyewan/gochat/im-infra/clog"
	"github.com/ceyewan/gochat/im-infra/coord"
)

// 两阶段初始化演示
//
// 阶段一：降级启动 (Bootstrap)
// - clog 使用默认配置启动，确保基础日志功能可用
// - coord 启动并连接到 etcd
//
// 阶段二：功能完备 (Full-Power)
// - clog 从配置中心重新加载配置
// - 其他组件启动

func main() {
	fmt.Println("=== 两阶段初始化演示 ===")

	// ==================== 阶段一：降级启动 ====================
	fmt.Println("\n🚀 阶段一：降级启动 (Bootstrap)")

	// 1. clog 降级启动 - 使用默认配置
	fmt.Println("1. 初始化 clog（降级模式）...")
	err := clog.Init() // 使用默认配置
	if err != nil {
		panic(fmt.Sprintf("Failed to init clog in bootstrap mode: %v", err))
	}

	// 此时 clog 已经可以使用，但使用的是默认配置
	clog.Info("clog 降级启动成功", clog.String("mode", "bootstrap"))

	// 2. 启动 coordination 组件
	fmt.Println("2. 启动 coordination 组件...")
	coordinator, err := coord.New()
	if err != nil {
		clog.Error("Failed to create coordinator", clog.Err(err))
		panic(err)
	}
	defer coordinator.Close()

	clog.Info("coordination 组件启动成功")

	// 检查 etcd 连接
	fmt.Println("3. 检查 etcd 连接...")
	// 这里可以添加 etcd 健康检查逻辑
	time.Sleep(1 * time.Second) // 模拟连接检查

	clog.Info("etcd 连接检查完成")

	// ==================== 阶段二：功能完备 ====================
	fmt.Println("\n⚡ 阶段二：功能完备 (Full-Power)")

	// 4. 设置配置中心
	fmt.Println("4. 设置配置中心...")
	clog.SetupConfigCenterFromCoord(coordinator.Config(), "dev", "im-infra", "clog")
	clog.Info("配置中心设置完成")

	// 5. clog 配置重载
	fmt.Println("5. 重新加载 clog 配置...")
	clog.ReloadConfig() // 从配置中心重新加载配置

	// 重新初始化全局 logger
	err = clog.Init()
	if err != nil {
		clog.Error("Failed to reload clog config", clog.Err(err))
	} else {
		clog.Info("clog 配置重载成功", clog.String("mode", "full-power"))
	}

	// 6. 启动其他组件
	fmt.Println("6. 启动其他组件...")
	// 这里可以启动 metrics、其他基础库等
	time.Sleep(500 * time.Millisecond) // 模拟组件启动

	clog.Info("其他组件启动完成")

	// 7. 启动业务逻辑
	fmt.Println("7. 启动业务逻辑...")
	time.Sleep(500 * time.Millisecond) // 模拟业务逻辑启动

	clog.Info("业务逻辑启动完成")

	// ==================== 运行演示 ====================
	fmt.Println("\n✅ 系统启动完成，进入运行状态")

	// 演示配置动态更新
	fmt.Println("\n📊 演示配置动态更新...")
	for i := 0; i < 10; i++ {
		clog.Info("系统运行中",
			clog.Int("iteration", i+1),
			clog.String("status", "running"))

		clog.Debug("调试信息",
			clog.Int("iteration", i+1),
			clog.String("detail", "debug info"))

		time.Sleep(2 * time.Second)

		// 在第5次迭代时提示用户可以更新配置
		if i == 4 {
			fmt.Println("\n💡 提示：现在可以使用以下命令更新配置：")
			fmt.Println("   cd ../../config")
			fmt.Println("   go run update/update.go dev im-infra clog '{\"level\":\"debug\",\"format\":\"json\"}'")
			fmt.Println("   观察日志输出的变化...")
		}
	}

	fmt.Println("\n🎉 演示完成")
}
